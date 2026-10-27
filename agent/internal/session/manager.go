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

const resumeNudge = "Your previous run was interrupted by a server restart. Continue from where you left off and finish the task."

var (
	ErrInvalidPrompt = errors.New("prompt is required")

	ErrInvalidName = errors.New("session name is required")

	ErrInvalidCursor = errors.New("invalid history cursor")

	ErrInvalidModel = errors.New("provider and model id are required")

	ErrInvalidThinkingLevel = errors.New("thinking level is required")
)

const (
	defaultHistoryPageSize = 200
	maxHistoryPageSize     = 1000
)

const maxNameLen = 200

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

type Manager struct {
	cfg   Config
	store store.Store

	mu   sync.Mutex
	byID map[string]*Session

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

type CreateOptions struct {
	Workspace string

	ModelProvider string
	ModelID       string

	ThinkingLevel string

	Title string

	AutoTitle bool

	AppendSystemPrompt string

	Attachments []Attachment

	AgencyTaskID string

	AgencyProjectID string
	AgencySeq       int64

	Sender *PromptSender
}

type AddOptions struct {
	Attachments []Attachment

	AgencyTaskID string

	AgencyProjectID string
	AgencySeq       int64

	Sender *PromptSender
}

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
		Interactive:        1,
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
		"interactive", true,
		"provider", opts.ModelProvider,
		"model", opts.ModelID,
	)

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

const titleTimeout = 60 * time.Second

const titleStoreTimeout = 5 * time.Second

const maxTitlePromptLen = 4000

const titleInstruction = "Generate a short, descriptive title (at most 6 words) for a coding session that starts with the user prompt below. Reply with only the title, no quotes, labels, or trailing punctuation.\n\nUser prompt:\n\n%s"

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

func (m *Manager) titleGenerator(
	ctx context.Context,
	prompt, provider, modelID, level string,
) (string, error) {
	if m.cfg.TitleGenerator != nil {
		return m.cfg.TitleGenerator(ctx, prompt, provider, modelID, level)
	}
	return m.runPiTitle(ctx, prompt, provider, modelID, level)
}

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

func truncateRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

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

func (m *Manager) End(ctx context.Context, id string) (sqlitedb.Task, error) {
	return m.end(ctx, id)
}

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

		if !row.PendingPrompt.Valid || row.PendingPrompt.String == "" {
			return nil
		}
		s = m.sessionFor(row)
	}
	if err := s.Stop(ctx); err != nil {
		slog.Error("task stop failed", "session_id", id, "error", err)
		return err
	}
	slog.Info("task stopped", "session_id", id)
	return nil
}

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

func (m *Manager) forget(id string) {
	m.mu.Lock()
	delete(m.byID, id)
	m.mu.Unlock()
}

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
	if s.historyHasPrompt(prompt) {
		text = resumeNudge
	}
	slog.Info("task resumed", "session_id", id, "task_id", s.agencyTaskID)
	return s.prompt(ctx, text)
}

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

func (m *Manager) WaitingForInput(id string) (bool, string) {
	if s := m.get(id); s != nil {
		return s.waitingForInput()
	}
	return false, ""
}

func (m *Manager) ProcessRunning(id string) bool {
	if s := m.get(id); s != nil {
		return s.processRunning()
	}
	return false
}

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

func (m *Manager) LiveUsageFor(ctx context.Context, id string) (Usage, bool) {
	s := m.get(id)
	if s == nil || !s.processRunning() {
		return Usage{}, false
	}
	return s.liveUsage(ctx)
}

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

	if dialog, ok := s.pendingDialogEvent(); ok {
		history = append(history, dialog)
	}
	ch, unsub := s.subscribe(statusEvent(status, row.Error.String), history...)
	return ch, unsub, nil
}

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

func (m *Manager) Defaults(ctx context.Context) (SessionConfig, error) {
	s := newSession(m.cfg, m.store, "", os.TempDir(), "", false)
	s.probe = true
	defer s.stopProcess()

	return s.Config(ctx)
}

func (m *Manager) Config(ctx context.Context, id string) (SessionConfig, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return SessionConfig{}, err
	}
	return m.sessionFor(row).Config(ctx)
}

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

func (m *Manager) SetThinkingLevel(ctx context.Context, id, level string) (SessionConfig, error) {
	row, err := m.store.GetSession(ctx, id)
	if err != nil {
		return SessionConfig{}, err
	}
	return m.sessionFor(row).SetThinkingLevel(ctx, level)
}

func (m *Manager) Recover(ctx context.Context) error {
	rows, err := m.store.ListActiveSessions(ctx, int64(StatusActive))
	if err != nil {
		return err
	}

	for _, row := range rows {

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
	if s.historyHasPrompt(pending) {
		text = resumeNudge
	}

	if err := s.prompt(ctx, text); err != nil {
		log.Printf("[session %s] recover prompt: %v", row.ID, err)
	}
}

func (m *Manager) Shutdown(ctx context.Context) {

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
	s := newSession(m.cfg, m.store, row.ID, row.Workspace, row.PiSessionID, true)
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
