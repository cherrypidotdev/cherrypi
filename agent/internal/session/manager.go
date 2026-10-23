// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package session

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	"cherrypi.vladfaust.com/agent/internal/store"
)

// resumeNudge is sent on recovery when Pi already persisted the interrupted
// prompt, so re-sending it would duplicate the user turn.
const resumeNudge = "Your previous run was interrupted by a server restart. Continue from where you left off and finish the task."

var (
	// ErrInvalidPrompt is returned when a prompt is empty.
	ErrInvalidPrompt = errors.New("prompt is required")
	// ErrInvalidName is returned when a session name is empty.
	ErrInvalidName = errors.New("session name is required")
	// ErrInvalidCursor is returned when a history cursor is not a valid value.
	ErrInvalidCursor = errors.New("invalid history cursor")
	// ErrInvalidModel is returned when a model selection is incomplete.
	ErrInvalidModel = errors.New("provider and model id are required")
	// ErrInvalidThinkingLevel is returned when a thinking level is empty.
	ErrInvalidThinkingLevel = errors.New("thinking level is required")
)

// History paging bounds: a console loads the newest events first, then walks
// backwards on demand.
const (
	defaultHistoryPageSize = 200
	maxHistoryPageSize     = 1000
)

// maxNameLen caps a session's display name; longer input is truncated.
const maxNameLen = 200

// defaultName derives a session's initial name from its prompt: the first
// line, trimmed and truncated.
func defaultName(prompt string) string {
	name := strings.TrimSpace(prompt)
	if i := strings.IndexAny(name, "\r\n"); i >= 0 {
		name = strings.TrimSpace(name[:i])
	}
	if r := []rune(name); len(r) > 80 {
		name = string(r[:80])
	}
	return name
}

// Manager owns every session's Pi process and their lifecycle.
type Manager struct {
	cfg   Config
	store store.Store

	mu   sync.Mutex
	byID map[string]*Session

	// runs counts in-flight agent runs; graceful shutdown waits for it to
	// drain.
	runs atomic.Int64
}

func NewManager(cfg Config, st store.Store) *Manager {
	if cfg.PiBin == "" {
		cfg.PiBin = "pi"
	}
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = "./sessions"
	}
	if cfg.EndGrace <= 0 {
		cfg.EndGrace = 5 * time.Second
	}
	if cfg.TitleTimeout <= 0 {
		cfg.TitleTimeout = titleTimeout
	}
	return &Manager{cfg: cfg, store: st, byID: map[string]*Session{}}
}

// CreateOptions configures a new session. The zero value matches Pi's
// configured defaults and a server-assigned workspace.
type CreateOptions struct {
	// Workspace overrides the server-assigned `$CHERRYPI_DIR/sessions/<id>`.
	Workspace string
	// Interactive surfaces extension dialogs instead of auto-declining them.
	Interactive bool
	// ModelProvider and ModelID select the session's model. Both must be set
	// together; empty keeps Pi's configured model.
	ModelProvider string
	ModelID       string
	// ThinkingLevel selects the reasoning level. Empty keeps Pi's configured
	// default for the selected model.
	ThinkingLevel string
	// Title is an explicit session display name. When set it wins over
	// AutoTitle and is stored verbatim (trimmed and capped).
	Title string
	// AutoTitle generates the display name from the initial prompt on a
	// best-effort basis after Create returns. It is ignored when Title is set.
	AutoTitle bool
	// AppendSystemPrompt is appended to Pi's system prompt for every run. The
	// agency composes it from the organization, agent, project, and
	// agent-project prompts.
	AppendSystemPrompt string
	// Attachments are files the agency attached to the initial prompt. The
	// manager downloads them into the workspace before prompting.
	Attachments []Attachment
	// AgencyTaskID is the agency-side task id, used to scope artifact uploads.
	AgencyTaskID string
	// AgencyProjectID and AgencySeq are the task's stable reference. The
	// manager persists them so a resumed session can still mint its scoped
	// tokens after an agent restart or a project prefix rename; the rendered
	// AgencyTaskID may go stale.
	AgencyProjectID string
	AgencySeq       int64
	// Sender is the actor the initial prompt is attributed to. It is rendered
	// into the envelope handed to Pi; nil becomes a system placeholder.
	Sender *PromptSender
}

// AddOptions configures a follow-up prompt.
type AddOptions struct {
	// Attachments are files sent with this prompt; the manager pulls them into
	// the workspace and hands images to Pi.
	Attachments []Attachment
	// AgencyTaskID refreshes the task's artifact-upload scope when set.
	AgencyTaskID string
	// AgencyProjectID and AgencySeq refresh the task's stable reference when
	// set, and are persisted for later respawns.
	AgencyProjectID string
	AgencySeq       int64
	// Sender attributes the prompt in the envelope handed to Pi; nil becomes
	// a system placeholder.
	Sender *PromptSender
}

// Create starts a new session and sends the first prompt. The workspace
// defaults to its own $CHERRYPI_DIR/sessions/<id>; a non-empty workspace
// overrides it (created if missing).
func (m *Manager) Create(
	ctx context.Context,
	prompt string,
	opts CreateOptions,
) (sqlitedb.Task, error) {
	if strings.TrimSpace(prompt) == "" {
		return sqlitedb.Task{}, ErrInvalidPrompt
	}
	if (opts.ModelProvider == "") != (opts.ModelID == "") {
		return sqlitedb.Task{}, ErrInvalidModel
	}

	id := newID()
	ws := opts.Workspace
	if ws == "" {
		ws = filepath.Join(m.cfg.WorkspaceRoot, id)
	}
	if abs, err := filepath.Abs(ws); err == nil {
		ws = abs
	}
	if err := os.MkdirAll(ws, 0o755); err != nil {
		return sqlitedb.Task{}, fmt.Errorf("create workspace: %w", err)
	}

	title := strings.TrimSpace(opts.Title)
	autoTitle := opts.AutoTitle && title == ""
	name := defaultName(prompt)
	switch {
	case title != "":
		name = title
	case autoTitle:
		// Generated asynchronously; an empty name is the pending state
		// clients render as a skeleton until the title arrives.
		name = ""
	}
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}

	now := time.Now().UnixMilli()
	row, err := m.store.CreateSession(ctx, sqlitedb.CreateSessionParams{
		ID:                 id,
		Status:             int64(StatusActive),
		Prompt:             prompt,
		Name:               name,
		Workspace:          ws,
		PiSessionID:        id,
		Interactive:        boolToInt(opts.Interactive),
		AppendSystemPrompt: opts.AppendSystemPrompt,
		AgencyProjectID:    opts.AgencyProjectID,
		AgencyTaskSeq:      opts.AgencySeq,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return sqlitedb.Task{}, err
	}
	slog.Info("task created",
		"session_id", id,
		"task_id", opts.AgencyTaskID,
		"workspace", ws,
		"interactive", opts.Interactive,
		"provider", opts.ModelProvider,
		"model", opts.ModelID,
	)

	// Pull the initial prompt's files into the workspace before Pi starts, so
	// the envelope can name their local paths.
	attachments, images, err := m.materializeAttachments(ctx, ws, opts.Attachments)
	if err != nil {
		return sqlitedb.Task{}, err
	}
	envelope := BuildPromptEnvelope(prompt, attachments, opts.Sender)

	s := m.sessionFor(row)
	s.agencyTaskID = opts.AgencyTaskID
	if err := s.promptConfigured(
		ctx,
		envelope,
		opts.ModelProvider,
		opts.ModelID,
		opts.ThinkingLevel,
		images,
	); err != nil {
		s.fail(err.Error())
		return row, err
	}

	s.notifyAgentState("created", StatusInProgress)

	if autoTitle {
		go m.autoTitle(id, prompt, opts.ModelProvider, opts.ModelID, opts.ThinkingLevel)
	}

	return m.store.GetSession(ctx, id)
}

// titleTimeout bounds a best-effort title generation so a stuck Pi child never
// leaks a goroutine forever. It is the default for Config.TitleTimeout.
const titleTimeout = 60 * time.Second

// titleStoreTimeout bounds the tiny name write that follows generation, which
// runs on its own context (see autoTitle).
const titleStoreTimeout = 5 * time.Second

// maxTitlePromptLen caps the prompt handed to the title model so a huge first
// message cannot blow up the one-shot request.
const maxTitlePromptLen = 4000

// titleInstruction is the one-shot prompt used by the default title generator.
const titleInstruction = "Generate a short, descriptive title (at most 6 words) for a coding session that starts with the user prompt below. Reply with only the title, no quotes, labels, or trailing punctuation.\n\nUser prompt:\n\n%s"

// autoTitle generates a session name from its initial prompt, falling back to
// the truncated prompt when generation fails so the UI never stays pending.
func (m *Manager) autoTitle(sessionID, prompt, provider, modelID, level string) {
	genCtx, genCancel := context.WithTimeout(
		context.Background(),
		m.cfg.TitleTimeout,
	)
	title, err := m.titleGenerator(genCtx, prompt, provider, modelID, level)
	genCancel()
	if err != nil {
		log.Printf("[session %s] auto-title: %v", sessionID, err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = defaultName(prompt)
	}
	if r := []rune(title); len(r) > maxNameLen {
		title = string(r[:maxNameLen])
	}

	// The generation context is cancelled once it returns (a timeout also kills
	// the child), so the fallback must be persisted on a fresh context.
	// Otherwise a failed or timed-out title is never written and clients show
	// the pending skeleton forever.
	ctx, cancel := context.WithTimeout(context.Background(), titleStoreTimeout)
	defer cancel()

	// Only fill a still-empty name: a rename that raced the generator wins.
	row, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		log.Printf("[session %s] auto-title lookup: %v", sessionID, err)
		return
	}
	if strings.TrimSpace(row.Name) != "" {
		return
	}

	if err := m.store.SetSessionName(ctx, sqlitedb.SetSessionNameParams{
		Name:      title,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        sessionID,
	}); err != nil {
		log.Printf("[session %s] auto-title save: %v", sessionID, err)
	}
}

// titleGenerator runs the configured generator, defaulting to a one-shot Pi
// child so the title respects the session's selected model/level.
func (m *Manager) titleGenerator(
	ctx context.Context,
	prompt, provider, modelID, level string,
) (string, error) {
	if m.cfg.TitleGenerator != nil {
		return m.cfg.TitleGenerator(ctx, prompt, provider, modelID, level)
	}
	return m.runPiTitle(ctx, prompt, provider, modelID, level)
}

// runPiTitle shells out to a throwaway `pi --print` process with tools and
// context disabled, and parses the first line of its output as the title.
func (m *Manager) runPiTitle(
	ctx context.Context,
	prompt, provider, modelID, level string,
) (string, error) {
	instruction := fmt.Sprintf(
		titleInstruction,
		truncateRunes(strings.TrimSpace(prompt), maxTitlePromptLen),
	)

	args := []string{
		"--print",
		"--no-session",
		"--no-tools",
		"--no-context-files",
		"--no-skills",
		"--no-prompt-templates",
		"--no-extensions",
	}
	// Keep test-pinned flags (e.g. --offline), then let an explicit selection
	// win over any --model they carry.
	args = append(args, m.cfg.PiArgs...)
	if provider != "" && modelID != "" {
		args = append(args, "--model", provider+"/"+modelID)
	}
	if level != "" {
		args = append(args, "--thinking", level)
	}
	args = append(args, instruction)

	cmd := exec.CommandContext(ctx, m.cfg.PiBin, args...)
	cmd.Dir = os.TempDir()
	if len(m.cfg.PiEnv) > 0 {
		cmd.Env = append(os.Environ(), m.cfg.PiEnv...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("title pi: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return parseTitle(string(out)), nil
}

// parseTitle extracts the first non-empty line of a model's reply and strips
// the wrappers models habitually add (quotes, backticks, markdown bullets).
func parseTitle(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.Trim(line, " \t\r\n\"'`*#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return truncateRunes(line, maxNameLen)
	}
	return ""
}

// truncateRunes caps s at n runes without splitting a multi-byte character.
func truncateRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func boolToInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

// Add sends a follow-up prompt to a session, resuming Pi if its process was
// reaped after the previous run. An ended session (done/cancelled/failed) is
// reactivated first, so a new message brings it back to active. Attachments
// are pulled into the workspace and handed to Pi with the prompt.
func (m *Manager) Add(ctx context.Context, id, prompt string, opts AddOptions) error {
	if strings.TrimSpace(prompt) == "" {
		return ErrInvalidPrompt
	}

	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	s := m.sessionFor(row)
	if opts.AgencyTaskID != "" {
		s.agencyTaskID = opts.AgencyTaskID
	}
	s.setAgencyRef(ctx, opts.AgencyProjectID, opts.AgencySeq)
	if Terminal(Status(row.Status)) {
		if err := m.reactivate(ctx, s); err != nil {
			return err
		}
	}

	attachments, images, err := m.materializeAttachments(ctx, row.Workspace, opts.Attachments)
	if err != nil {
		return err
	}
	envelope := BuildPromptEnvelope(prompt, attachments, opts.Sender)

	slog.Info("task prompt added",
		"session_id", id,
		"task_id", s.agencyTaskID,
		"images", len(images),
	)
	return s.promptImages(ctx, envelope, images)
}

// reactivate moves an ended session back to active so a new prompt can be sent;
// it clears the end metadata and reopens the event stream to watchers.
func (m *Manager) reactivate(ctx context.Context, s *Session) error {
	if err := m.store.ReactivateSession(ctx, sqlitedb.ReactivateSessionParams{
		Status:    int64(StatusActive),
		UpdatedAt: time.Now().UnixMilli(),
		ID:        s.ID,
	}); err != nil {
		return err
	}

	s.emit(statusEvent(StatusInProgress, ""))
	s.notifyAgentState("reactivated", StatusInProgress)
	return nil
}

// End stops a session: a session with a run in flight is cancelled, an idle
// one is done. It leaves the session visible.
func (m *Manager) End(ctx context.Context, id string) (sqlitedb.Task, error) {
	return m.end(ctx, id)
}

// Stop aborts a session's in-flight run without ending it. The
// session stays active and a follow-up prompt can resume it immediately.
func (m *Manager) Stop(ctx context.Context, id string) error {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	if Terminal(Status(row.Status)) {
		return nil
	}
	s := m.get(id)
	if s == nil {
		return nil
	}
	if err := s.Stop(ctx); err != nil {
		slog.Error("task stop failed", "session_id", id, "error", err)
		return err
	}
	slog.Info("task stopped", "session_id", id)
	return nil
}

// ClearQueue drops a session's queued (not yet injected) prompts. A terminal
// or unwarmed session has nothing queued.
func (m *Manager) ClearQueue(ctx context.Context, id string) error {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	if Terminal(Status(row.Status)) {
		return nil
	}
	s := m.get(id)
	if s == nil {
		return nil
	}
	return s.ClearQueue(ctx)
}

// end stops a session.
func (m *Manager) end(ctx context.Context, id string) (sqlitedb.Task, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return row, err
	}
	if Terminal(Status(row.Status)) {
		return row, nil
	}

	status := StatusDone
	if s := m.get(id); s != nil {
		s.opMu.Lock()
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()
		if running {
			// Best-effort abort so an in-flight turn stops promptly. The
			// interruption is folded into DONE rather than a separate status.
			abortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, _ = s.send(abortCtx, "abort", nil)
			cancel()
		}
		s.stopProcess()
		s.setRunning(false)
		s.clearWaiting()
		s.opMu.Unlock()

		s.emit(statusEvent(status, ""))
		s.notifyAgentState("ended", status)
	}

	now := time.Now().UnixMilli()
	err = m.store.EndSession(ctx, sqlitedb.EndSessionParams{
		Status:    int64(status),
		EndedAt:   sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return row, err
	}

	return m.store.GetSession(ctx, id)
}

// Delete removes a session permanently: it stops the session's Pi process,
// deletes its server-assigned workspace directory, and removes its record.
// Token/cost usage history is kept. A workspace explicitly chosen by a client
// (outside WorkspaceRoot) is left on disk so an unrelated directory is never
// wiped.
func (m *Manager) Delete(ctx context.Context, id string) error {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	if s := m.get(id); s != nil {
		s.opMu.Lock()
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()
		if running {
			abortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, _ = s.send(abortCtx, "abort", nil)
			cancel()
		}
		s.stopProcess()
		s.setRunning(false)
		s.opMu.Unlock()
		m.forget(id)
	}

	if m.cfg.WorkspaceRoot != "" && withinDir(m.cfg.WorkspaceRoot, row.Workspace) {
		if err := os.RemoveAll(row.Workspace); err != nil {
			return fmt.Errorf("remove workspace: %w", err)
		}
	}

	// Best-effort: drop Pi's own session JSONL too. A throwaway Session is
	// enough to reuse its path resolution.
	if row.PiSessionID != "" {
		probe := &Session{cfg: m.cfg, PiSessionID: row.PiSessionID, Workspace: row.Workspace}
		if path := probe.sessionFilePath(); path != "" {
			_ = os.Remove(path)
		}
	}

	if err := m.store.DeleteSession(ctx, id); err != nil {
		return err
	}
	slog.Info("task deleted", "session_id", id, "workspace", row.Workspace)
	return nil
}

// forget drops a session from the in-memory map.
func (m *Manager) forget(id string) {
	m.mu.Lock()
	delete(m.byID, id)
	m.mu.Unlock()
}

// withinDir reports whether path is dir or nested under it.
func withinDir(dir, path string) bool {
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dirAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// Retry re-runs a failed session's last prompt. It prefers the persisted
// pending prompt (the one that failed) and falls back to the original prompt.
// The session is reactivated, and the same de-duplication as recovery is used:
// if Pi already persisted the prompt, it is nudged to continue instead of
// duplicating the user turn.
func (m *Manager) Retry(ctx context.Context, id, projectID string, seq int64) error {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return err
	}

	prompt := row.Prompt
	if row.PendingPrompt.Valid && strings.TrimSpace(row.PendingPrompt.String) != "" {
		prompt = row.PendingPrompt.String
	}
	if strings.TrimSpace(prompt) == "" {
		return ErrInvalidPrompt
	}

	s := m.sessionFor(row)
	s.setAgencyRef(ctx, projectID, seq)
	if Terminal(Status(row.Status)) {
		if err := m.reactivate(ctx, s); err != nil {
			return err
		}
	}

	if err := s.ensureRunning(ctx); err != nil {
		slog.Error("task resume failed", "session_id", id, "task_id", s.agencyTaskID, "error", err)
		return err
	}

	text := prompt
	if s.historyHasPrompt(ctx, prompt) {
		text = resumeNudge
	}
	slog.Info("task resumed", "session_id", id, "task_id", s.agencyTaskID)
	return s.prompt(ctx, text)
}

// Rename sets a session's display name. An empty name is rejected.
func (m *Manager) Rename(ctx context.Context, id, name string) (sqlitedb.Task, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return sqlitedb.Task{}, ErrInvalidName
	}
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}

	if err := m.store.SetSessionName(ctx, sqlitedb.SetSessionNameParams{
		Name:      name,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        id,
	}); err != nil {
		return sqlitedb.Task{}, err
	}

	return m.store.GetSession(ctx, id)
}

func (m *Manager) Get(ctx context.Context, id string) (sqlitedb.Task, error) {
	return m.store.GetSession(ctx, id)
}

// WaitingForInput reports whether a session is blocked on an extension dialog.
func (m *Manager) WaitingForInput(id string) (bool, string) {
	if s := m.get(id); s != nil {
		return s.waitingForInput()
	}
	return false, ""
}

// ProcessRunning reports whether a session's Pi process is currently alive.
func (m *Manager) ProcessRunning(id string) bool {
	if s := m.get(id); s != nil {
		return s.processRunning()
	}
	return false
}

// RespondExtensionUI answers a pending dialog on a session.
func (m *Manager) RespondExtensionUI(
	ctx context.Context,
	id, requestID string,
	resp ExtensionResponse,
) error {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	return m.sessionFor(row).respondExtensionUI(ctx, requestID, resp)
}

// LiveUsageFor reads Pi's live cumulative stats for one session, if its
// process is running. It is the session-scoped counterpart of LiveUsage, so a
// single-session view does not query every other running process.
func (m *Manager) LiveUsageFor(ctx context.Context, id string) (Usage, bool) {
	s := m.get(id)
	if s == nil || !s.processRunning() {
		return Usage{}, false
	}
	return s.liveUsage(ctx)
}

// LiveUsage reads Pi's live cumulative stats for every session whose process
// is running (its run has not settled, so no usage row exists yet).
func (m *Manager) LiveUsage(ctx context.Context) map[string]Usage {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.byID))
	for _, s := range m.byID {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	out := map[string]Usage{}
	for _, s := range sessions {
		if !s.processRunning() {
			continue
		}
		if u, ok := s.liveUsage(ctx); ok {
			out[s.ID] = u
		}
	}
	return out
}

func (m *Manager) List(ctx context.Context) ([]sqlitedb.Task, error) {
	return m.store.ListSessions(ctx)
}

// Watch streams normalized events for a session. When replay is true, the
// session's persisted conversation history is delivered first, then the
// current status, then live events. The stream stays open for ended sessions
// too, so a reactivation (a new message) is delivered to existing watchers.
func (m *Manager) Watch(ctx context.Context, id string, replay bool) (<-chan Event, func(), error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	s := m.sessionFor(row)
	var history []Event
	if replay {
		history = s.loadHistory()
	}
	waiting, _ := s.waitingForInput()
	status := ClientStatus(Status(row.Status), row.PendingPrompt.Valid, waiting)
	// Re-surface a pending dialog: Pi keeps no extension-UI records in its
	// session file, so a console that connects after the question was asked
	// (or after a restart) only learns about it from this replay.
	if dialog, ok := s.pendingDialogEvent(); ok {
		history = append(history, dialog)
	}
	ch, unsub := s.subscribe(statusEvent(status, row.Error.String), history...)
	return ch, unsub, nil
}

// SyncHistory forces a session's local Pi history into the agency now. It is
// the on-demand counterpart of the running watcher: a session whose Pi process
// is gone (or last ran before automatic sync existed) still migrates its
// stored conversation file. projectID and seq are the task's stable agency
// reference; when non-empty they are recorded before the sync, so a legacy
// session that predates the persisted reference (or was created before the
// agent tracked it) can still migrate. It returns the number of entries in the
// local file after the sync, 0 when there is nothing to migrate.
func (m *Manager) SyncHistory(
	ctx context.Context,
	id, projectID string,
	seq int64,
) (int, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return 0, err
	}
	s := m.sessionFor(row)
	s.setAgencyRef(ctx, projectID, seq)
	s.resetHistorySync()
	s.syncHistory(ctx)
	return len(s.readSessionEntries()), nil
}

// History returns a page of a session's persisted conversation, oldest-first,
// plus the cursor for the next older page (empty when exhausted). `before` is
// an opaque cursor from a previous call; empty starts from the newest events.
func (m *Manager) History(
	ctx context.Context,
	id, before string,
	limit int,
) ([]Event, string, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return nil, "", err
	}

	var cursor int64
	if before != "" {
		value, err := strconv.ParseInt(before, 10, 64)
		if err != nil || value < 0 {
			return nil, "", ErrInvalidCursor
		}
		cursor = value
	}

	switch {
	case limit <= 0:
		limit = defaultHistoryPageSize
	case limit > maxHistoryPageSize:
		limit = maxHistoryPageSize
	}

	events, next := m.sessionFor(row).loadHistoryPage(cursor, limit)
	nextBefore := ""
	if next > 0 {
		nextBefore = strconv.FormatInt(next, 10)
	}
	return events, nextBefore, nil
}

// Defaults reports the model and thinking options a new session can start
// with: Pi's current model/reasoning level plus every selectable model and the
// levels it supports. Pi's model registry is process-global, so this probes a
// short-lived `pi --mode rpc --no-session` child rather than a persisted
// session.
//
// ponytail: a fresh probe per call; cache on the manager with a short TTL if
// the New Session form ever becomes hot.
func (m *Manager) Defaults(ctx context.Context) (SessionConfig, error) {
	s := newSession(m.cfg, m.store, "", os.TempDir(), "", false)
	s.probe = true
	defer s.stopProcess()

	return s.Config(ctx)
}

// Config reports a session's live model and reasoning configuration.
func (m *Manager) Config(ctx context.Context, id string) (SessionConfig, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return SessionConfig{}, err
	}
	return m.sessionFor(row).Config(ctx)
}

// SetModel switches a session's model and returns the refreshed config.
func (m *Manager) SetModel(
	ctx context.Context,
	id, provider, modelID string,
) (SessionConfig, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return SessionConfig{}, err
	}
	return m.sessionFor(row).SetModel(ctx, provider, modelID)
}

// SetThinkingLevel sets a session's reasoning level and returns the refreshed
// config.
func (m *Manager) SetThinkingLevel(ctx context.Context, id, level string) (SessionConfig, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return SessionConfig{}, err
	}
	return m.sessionFor(row).SetThinkingLevel(ctx, level)
}

// Recover auto-continues sessions whose run was interrupted by a restart:
// re-send the prompt if Pi never persisted it, otherwise nudge Pi to continue.
func (m *Manager) Recover(ctx context.Context) error {
	rows, err := m.store.ListActiveSessions(ctx, int64(StatusActive))
	if err != nil {
		return err
	}

	for _, row := range rows {
		// An empty name means auto-titling was interrupted by the restart; the
		// selected model is not persisted, so fall back to the derived name
		// rather than leaving the UI pending forever.
		if strings.TrimSpace(row.Name) == "" {
			name := defaultName(row.Prompt)
			if err := m.store.SetSessionName(ctx, sqlitedb.SetSessionNameParams{
				Name:      name,
				UpdatedAt: time.Now().UnixMilli(),
				ID:        row.ID,
			}); err != nil {
				slog.Warn("recover name failed", "session_id", row.ID, "error", err)
			}
		}

		if row.WaitingForInput != 0 || row.PendingDialog.Valid {
			// A waiting session has no in-flight run to continue; it stays put
			// until the user answers the persisted dialog.
			slog.Info("waiting for input; skipping recovery", "session_id", row.ID)
			continue
		}
		if !row.PendingPrompt.Valid || row.PendingPrompt.String == "" {
			continue
		}
		slog.Info("recovering interrupted run", "session_id", row.ID)
		go m.autoContinue(row, m.sessionFor(row))
	}

	return nil
}

func (m *Manager) autoContinue(row sqlitedb.Task, s *Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pending := row.PendingPrompt.String

	if err := s.ensureRunning(ctx); err != nil {
		log.Printf("[session %s] recover: %v", row.ID, err)
		return
	}

	text := pending
	if s.historyHasPrompt(ctx, pending) {
		text = resumeNudge
	}

	if err := s.prompt(ctx, text); err != nil {
		log.Printf("[session %s] recover prompt: %v", row.ID, err)
	}
}

// Shutdown waits for all in-flight runs to finish, then stops any lingering
// Pi processes. It returns early if ctx is done (a forced shutdown).
func (m *Manager) Shutdown(ctx context.Context) {
	// A waiting session blocks its run indefinitely; on shutdown there is no
	// client to answer it, so stop those processes rather than waiting forever.
	m.stopWaiting()

	for m.runs.Load() > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}

	m.stopAll()
}

// stopWaiting force-stops every session blocked on an extension dialog.
func (m *Manager) stopWaiting() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.byID))
	for _, s := range m.byID {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	for _, s := range sessions {
		if waiting, _ := s.waitingForInput(); waiting {
			s.forceStop()
			s.setRunning(false)
			s.clearWaiting()
		}
	}
}

// Kill stops every Pi process immediately, without waiting for runs or locks.
func (m *Manager) Kill() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.byID))
	for _, s := range m.byID {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	for _, s := range sessions {
		s.forceStop()
		s.setRunning(false)
	}
}

func (m *Manager) stopAll() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.byID))
	for _, s := range m.byID {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, s := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			s.opMu.Lock()
			defer s.opMu.Unlock()
			s.stopProcess()
			s.setRunning(false)
		}(s)
	}
	wg.Wait()
}

// runHook tracks run edges for graceful shutdown.
func (m *Manager) runHook(started bool) {
	if started {
		m.runs.Add(1)
		return
	}
	m.runs.Add(-1)
}

func (m *Manager) get(id string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byID[id]
}

func (m *Manager) sessionFor(row sqlitedb.Task) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s := m.byID[row.ID]; s != nil {
		return s
	}
	s := newSession(m.cfg, m.store, row.ID, row.Workspace, row.PiSessionID, row.Interactive != 0)
	s.appendSystemPrompt = row.AppendSystemPrompt
	s.agencyProjectID = row.AgencyProjectID
	s.agencySeq = row.AgencyTaskSeq
	if row.WaitingForInput != 0 && row.PendingDialog.Valid {
		s.savedDialog = row.PendingDialog.String
	}
	s.runHook = m.runHook
	m.byID[row.ID] = s
	return s
}
