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
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	"cherrypi.vladfaust.com/agent/internal/logging"
	"cherrypi.vladfaust.com/agent/internal/store"
)

// logRingMax bounds the in-memory stderr ring per session.
const logRingMax = 200

// ErrNoPendingDialog is returned when answering a dialog that isn't pending.
var ErrNoPendingDialog = errors.New("no pending extension dialog with that request id")

// Config controls how the manager spawns and drives Pi.
type Config struct {
	// PiBin is the Pi executable. Default: "pi".
	PiBin string
	// PiSessionDir is an optional `--session-dir` for Pi's session storage.
	PiSessionDir string
	// AgentDir is Pi's agent directory, used to locate the default session
	// store when PiSessionDir is unset. Default: ~/.pi/agent.
	AgentDir string
	// WorkspaceRoot is the parent of per-session workspaces.
	WorkspaceRoot string
	// EndGrace is how long a stopped Pi process gets to exit before SIGKILL.
	EndGrace time.Duration
	// PiEnv are extra environment variables for the Pi child process (each
	// "K=V"). Default: nil.
	PiEnv []string
	// PiArgs are extra CLI args appended after `--mode rpc --session-id …`.
	// Used in tests to pin `--model`, `--offline`, etc. Default: nil.
	PiArgs []string
	// FetchAttachment opens an agency attachment by its relative URL,
	// authenticated with the tunnel credential. nil disables attachment
	// materialization (tests that do not exercise it).
	FetchAttachment func(ctx context.Context, url string) (io.ReadCloser, error)
	// AgencyURL is the agency base URL, handed to Pi's bundled tools.
	AgencyURL string
	// AgentTokenFile is the path to the agent's personal access token. It is
	// passed to Pi as CHERRYPI_AGENT_TOKEN_FILE; the bundled tools read it on
	// every call, so a rotated token needs no Pi respawn. Empty leaves the
	// tools to fall back to the CHERRYPI_AGENT_TOKEN they inherit.
	AgentTokenFile string
	// ResolveTaskID resolves a task's current display id from its stable
	// (project, seq) reference, refreshing an id a prefix rename may have
	// changed. nil leaves the persisted display id in place.
	ResolveTaskID func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
	) (taskID string, err error)
	// FetchTaskHistory returns the agency's canonical Pi session entries for a
	// task, resolved by its stable (project, seq) reference. nil disables
	// history sync (tests that do not exercise it).
	FetchTaskHistory func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
	) ([]SessionEntry, error)
	// PushTaskHistory appends entries the agency may be missing for a task.
	PushTaskHistory func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
		entries []SessionEntry,
	) error
	// HistorySyncInterval is how often a running session flushes newly written
	// Pi session entries to the agency. Default: 1s.
	HistorySyncInterval time.Duration
	// IdleTTL is how long a settled session keeps its Pi process warm before
	// it is reaped; the next prompt resumes it. Default: 15m.
	IdleTTL time.Duration
	// Notify, if set, receives outbound webhook events ("user_input_requested",
	// "agent_state_changed") with a payload that includes the session id.
	Notify func(event string, payload map[string]any)
	// TitleGenerator derives a session's display name from its initial prompt.
	// When nil, the manager runs a one-shot `pi --print` child; tests override
	// it to keep the title path deterministic.
	TitleGenerator TitleGenerator
	// TitleTimeout bounds a best-effort title generation so a stuck child never
	// leaks a goroutine. Default: 60s.
	TitleTimeout time.Duration
}

// TitleGenerator derives a short session title from the initial prompt. The
// provider/model/level name the model to use, matching the session's
// selection.
type TitleGenerator func(ctx context.Context, prompt, provider, modelID, level string) (string, error)

// Session is the live runtime for one agent session. It stays in the manager
// even while its Pi process is reaped, so in-memory state (usage totals,
// subscribers, stderr ring) survives between runs.
type Session struct {
	cfg   Config
	store store.Store

	ID          string
	Workspace   string
	PiSessionID string

	// agencyTaskID is the agency-side task this Pi session serves, used to
	// scope artifact uploads. Empty when the caller did not supply one. It is
	// a display id and may go stale after a project prefix rename; the stable
	// reference below is authoritative.
	agencyTaskID string
	// agencyProjectID and agencySeq are the task's stable agency reference.
	// They are persisted on the agent's task row, so a respawn after an agent
	// restart can re-mint the task's scoped tokens even when agencyTaskID was
	// never restored. Empty/zero for a legacy session.
	agencyProjectID string
	agencySeq       int64

	// opMu serializes process-level operations (prompt/settle/end) so a run
	// isn't killed or resumed from under an in-flight command.
	opMu sync.Mutex

	mu       sync.Mutex
	started  bool
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	exited   chan struct{}
	stopping bool
	running  bool
	// runStartedAt is when the current in-flight run began; zero when idle. It
	// anchors the live token/cost rate reported by liveUsage.
	runStartedAt time.Time
	// runWaitAccum is how long the current run has spent waiting on user input
	// (extension dialogs); waitStartedAt marks an in-flight wait. Both reset on
	// the false->true run edge and feed the active-time accounting.
	runWaitAccum  time.Duration
	waitStartedAt time.Time
	pending       map[string]chan response
	reqID         uint64
	model         string
	lastUse       sqlitedb.SumUsageBySessionRow
	subs          map[chan Event]struct{}
	logRing       []string
	writeMu       sync.Mutex

	// interactive sessions surface extension dialogs to the user;
	// non-interactive ones auto-decline them.
	interactive bool
	// appendSystemPrompt is appended to Pi's system prompt on every spawn. It
	// is persisted with the session so reaping and resuming are lossless.
	appendSystemPrompt string
	// probe marks a throwaway session that talks to a `pi --no-session` child
	// to read global model options; it never touches the store or session
	// storage.
	probe bool
	// waiting is true while a dialog method is pending an answer.
	waiting bool
	// queue is Pi's current pending-message queue (steering + follow-up), kept
	// so a late watcher can be told what is waiting.
	queue QueueEvent
	// pendingExt holds un-answered dialog requests, keyed by Pi's request id.
	pendingExt map[string]extensionUIRequest
	// lastDialogJSON is the raw JSON of the most recent pending dialog.
	lastDialogJSON string
	// savedDialog is the persisted pending dialog of a session that was
	// waiting when the server restarted (the live process is gone).
	savedDialog string
	// savedAnswer answers the next re-emitted dialog after a restart.
	savedAnswer *ExtensionResponse

	// idleTimer reaps the Pi process after IdleTTL once a run settles;
	// idleGen invalidates a timer that a new prompt superseded.
	idleTimer *time.Timer
	idleGen   uint64

	// runHook is called when a run starts/finishes, so the manager can track
	// in-flight runs during graceful shutdown.
	runHook func(started bool)

	// histMu serializes history sync and guards the sync state below. A new
	// Pi process resets historyReconciled so the first flush of a run compares
	// the agency's stored tip with the local session file. historyDisabled is
	// set when the agency reports it does not know the task.
	histMu            sync.Mutex
	syncedTip         string
	historyReconciled bool
	historyDisabled   bool
	// historyCancel stops the history watcher; guarded by mu.
	historyCancel context.CancelFunc
}

type response struct {
	Success bool
	Error   string
	Data    json.RawMessage
}

// extensionUIRequest is one Pi `extension_ui_request`. Only the fields we
// forward are listed; RawJSON keeps the whole record for persistence and
// method-specific fields.
type extensionUIRequest struct {
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	Method      string   `json:"method"`
	Title       string   `json:"title"`
	Message     string   `json:"message"`
	Placeholder string   `json:"placeholder"`
	Prefill     string   `json:"prefill"`
	Options     []string `json:"options"`
	Timeout     uint32   `json:"timeout"`

	RawJSON string `json:"-"`
}

// isDialogMethod reports whether a method blocks the run awaiting an answer.
func isDialogMethod(method string) bool {
	switch method {
	case "select", "confirm", "input", "editor":
		return true
	default:
		return false
	}
}

// ExtensionResponseKind discriminates a RespondToExtensionUi answer.
type ExtensionResponseKind int

const (
	ExtensionResponseCancelled ExtensionResponseKind = iota
	ExtensionResponseValue
	ExtensionResponseConfirmed
)

// ExtensionResponse is a client's answer to a pending dialog.
type ExtensionResponse struct {
	Kind      ExtensionResponseKind
	Value     string
	Confirmed bool
	// Questionnaire carries every question/answer pair of a submitted
	// ask_user_question wizard. Set on the first answer only; used to enrich
	// the resume prompt when the dialog survived a restart.
	Questionnaire []QuestionnaireAnswer
}

// QuestionnaireAnswer is one question/answer pair of a submitted
// ask_user_question wizard.
type QuestionnaireAnswer struct {
	Question string
	Answer   string
}

// record is one Pi RPC record; only the fields we consume are listed.
type record struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Success *bool           `json:"success"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`

	AssistantMessageEvent json.RawMessage `json:"assistantMessageEvent"`
	Message               json.RawMessage `json:"message"`

	// queue_update carries Pi's pending-message queue.
	Steering []string `json:"steering"`
	FollowUp []string `json:"followUp"`

	ToolCallID    string          `json:"toolCallId"`
	ToolName      string          `json:"toolName"`
	Args          json.RawMessage `json:"args"`
	PartialResult json.RawMessage `json:"partialResult"`
	Result        json.RawMessage `json:"result"`
	IsError       bool            `json:"isError"`
}

func newSession(
	cfg Config,
	st store.Store,
	id, workspace, piSessionID string,
	interactive bool,
) *Session {
	return &Session{
		cfg:         cfg,
		store:       st,
		ID:          id,
		Workspace:   workspace,
		PiSessionID: piSessionID,
		interactive: interactive,
		pendingExt:  map[string]extensionUIRequest{},
		subs:        map[chan Event]struct{}{},
	}
}

// newID returns a random 32-hex-char session id. Safe for Pi's `--session-id`
// (alphanumeric, starts and ends with a letter or digit).
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// prompt ensures the Pi process is running, records the prompt as pending, and
// sends it. A prompt sent while a run is active is queued as a steering message
// and delivered between the run's tool calls.
func (s *Session) prompt(ctx context.Context, text string) error {
	return s.promptImages(ctx, text, nil)
}

// promptImages is prompt with image content for Pi. Images are sent as Pi's
// `images` parts; the text carries the on-disk paths for other files.
func (s *Session) promptImages(ctx context.Context, text string, images []imageAttachment) error {
	return s.promptConfigured(ctx, text, "", "", "", images)
}

// promptConfigured is prompt with an optional model/thinking selection applied
// to the freshly-started Pi process before the prompt is sent. A new session
// has no persisted selection, so this sets it for the first run. Empty values
// keep Pi's configured defaults.
func (s *Session) promptConfigured(
	ctx context.Context,
	text, provider, modelID, level string,
	images []imageAttachment,
) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	// A new prompt keeps the process: cancel any pending idle reap.
	s.cancelIdle()

	if err := s.ensureRunningLocked(ctx); err != nil {
		return err
	}

	if provider != "" || modelID != "" {
		if _, err := s.send(ctx, "set_model", map[string]any{
			"provider": provider,
			"modelId":  modelID,
		}); err != nil {
			return err
		}
		// Attribute the run's usage to the model we just selected rather than
		// the default `get_state` reported at startup.
		s.mu.Lock()
		s.model = modelID
		s.mu.Unlock()
	}

	if level != "" {
		if _, err := s.send(ctx, "set_thinking_level", map[string]any{
			"level": level,
		}); err != nil {
			return err
		}
	}

	// Set pending before writing to Pi: if we crash between the two, recovery
	// re-sends rather than silently drops the prompt.
	now := time.Now().UnixMilli()
	if err := s.store.SetSessionPendingPrompt(ctx, sqlitedb.SetSessionPendingPromptParams{
		PendingPrompt: sql.NullString{String: text, Valid: true},
		UpdatedAt:     now,
		ID:            s.ID,
	}); err != nil {
		return err
	}

	fields := map[string]any{"message": text}
	if len(images) > 0 {
		parts := make([]map[string]any, 0, len(images))
		for _, image := range images {
			parts = append(parts, map[string]any{
				"type":     "image",
				"data":     base64.StdEncoding.EncodeToString(image.data),
				"mimeType": image.mimeType,
			})
		}
		fields["images"] = parts
	}

	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		// Steer: Pi delivers the message between the in-flight run's tool calls
		// instead of only once the whole turn settles. Interrupting the run is a
		// separate action (Stop).
		fields["streamingBehavior"] = "steer"
	}

	if _, err := s.send(ctx, "prompt", fields); err != nil {
		return err
	}

	slog.Info("prompt sent",
		"session_id", s.ID,
		"task_id", s.agencyTaskID,
		"steer", running,
		"images", len(images),
	)

	s.setRunning(true)
	return nil
}

// childEnv builds the Pi process environment: the ambient one plus the static
// PiEnv, plus the per-task variables the bundled tools read (the agency URL,
// the agent token file, and the task's current display id).
//
// The stable (project, seq) reference is authoritative: when it is known, the
// agent resolves it with the agency first to refresh a display id a prefix
// rename may have changed. A legacy session with no reference falls back to the
// persisted display id. The token itself is never put in the environment; the
// tools read the token file so a rotation applies to a running Pi.
func (s *Session) childEnv(ctx context.Context) []string {
	env := append(os.Environ(), s.cfg.PiEnv...)

	if s.cfg.AgencyURL != "" {
		env = append(env, "CHERRYPI_AGENCY_URL="+s.cfg.AgencyURL)
	}
	if s.cfg.AgentTokenFile != "" {
		env = append(env, "CHERRYPI_AGENT_TOKEN_FILE="+s.cfg.AgentTokenFile)
	}

	taskID := s.agencyTaskID
	if s.agencyProjectID != "" && s.agencySeq > 0 && s.cfg.ResolveTaskID != nil {
		if fresh, err := s.cfg.ResolveTaskID(
			ctx,
			s.agencyProjectID,
			s.agencySeq,
		); err == nil && fresh != "" {
			taskID = fresh
		}
	}
	if taskID == "" {
		return env
	}

	return append(env, "CHERRYPI_TASK_ID="+taskID)
}

// setAgencyRef records the task's stable agency reference in memory and
// persists it, so a later respawn can restore it. It is a no-op for an empty
// reference or one that matches what is already stored.
func (s *Session) setAgencyRef(ctx context.Context, projectID string, seq int64) {
	s.mu.Lock()
	current := s.agencyProjectID == projectID && s.agencySeq == seq
	if projectID == "" || seq <= 0 || current {
		s.mu.Unlock()
		return
	}
	s.agencyProjectID = projectID
	s.agencySeq = seq
	s.mu.Unlock()
	if err := s.store.SetSessionAgencyRef(ctx, sqlitedb.SetSessionAgencyRefParams{
		AgencyProjectID: projectID,
		AgencyTaskSeq:   seq,
		UpdatedAt:       time.Now().UnixMilli(),
		ID:              s.ID,
	}); err != nil {
		log.Printf("[session %s] persist agency ref: %v", s.ID, err)
	}
}

// agencyRef reads the task's stable agency reference under the lock.
func (s *Session) agencyRef() (string, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agencyProjectID, s.agencySeq
}

// setRunning updates the run flag and notifies the manager on every edge. A
// false->true edge stamps the run's start time for live rate reporting.
func (s *Session) setRunning(v bool) {
	s.mu.Lock()
	changed := s.running != v
	s.running = v
	if changed {
		if v {
			s.runStartedAt = time.Now()
			s.runWaitAccum = 0
			s.waitStartedAt = time.Time{}
		} else {
			s.runStartedAt = time.Time{}
		}
	}
	s.mu.Unlock()

	if changed && s.runHook != nil {
		s.runHook(v)
	}
}

// ensureRunning spawns Pi if it isn't already running.
func (s *Session) ensureRunning(ctx context.Context) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	return s.ensureRunningLocked(ctx)
}

func (s *Session) ensureRunningLocked(ctx context.Context) error {
	s.mu.Lock()
	if s.cmd != nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// Before Pi starts, reconcile the agency's canonical history with the local
	// session file, so a fresh agent (or a lost workspace) continues the task
	// from the stored conversation rather than an empty session.
	if !s.probe && s.historyEnabled() {
		s.resetHistorySync()
		s.syncHistory(ctx)
	}

	args := []string{"--mode", "rpc"}
	if s.probe {
		args = append(args, "--no-session")
	} else {
		args = append(args, "--session-id", s.PiSessionID)
		if s.cfg.PiSessionDir != "" {
			args = append(args, "--session-dir", s.cfg.PiSessionDir)
		}
	}
	args = append(args, s.cfg.PiArgs...)
	// The session's system prompt is appended last so test-pinned flags cannot
	// suppress it. Pi joins repeated flags in order, double-newline separated.
	if s.appendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", s.appendSystemPrompt)
	}

	cmd := exec.Command(s.cfg.PiBin, args...)
	cmd.Dir = s.Workspace
	cmd.Env = s.childEnv(ctx)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", s.cfg.PiBin, err)
	}

	slog.Info("pi started",
		"session_id", s.ID,
		"task_id", s.agencyTaskID,
		"pid", pidOf(cmd),
		"bin", s.cfg.PiBin,
	)

	s.mu.Lock()
	s.cmd = cmd
	s.stdin = stdin
	s.exited = make(chan struct{})
	s.stopping = false
	s.pending = map[string]chan response{}
	first := !s.started
	s.started = true
	s.mu.Unlock()

	go s.readStdout(stdout)
	go s.readStderr(stderr)
	go s.reap(cmd)

	if !s.probe {
		s.startHistoryWatch()
	}

	if !s.probe {
		now := time.Now().UnixMilli()
		if err := s.store.SetSessionPid(ctx, sqlitedb.SetSessionPidParams{
			Pid:       pidOf(cmd),
			UpdatedAt: now,
			ID:        s.ID,
		}); err != nil {
			log.Printf("[session %s] record pid: %v", s.ID, err)
		}

		if first {
			s.loadInitialState(ctx)
		}
	}

	return nil
}

func pidOf(cmd *exec.Cmd) sql.NullInt64 {
	if cmd.Process == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(cmd.Process.Pid), Valid: true}
}

type stateData struct {
	Model struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"model"`
}

// loadInitialState records the model and seeds the usage baseline from the
// database, so per-run usage deltas stay correct across restarts.
func (s *Session) loadInitialState(ctx context.Context) {
	if data, err := s.send(ctx, "get_state", nil); err == nil {
		var st stateData
		if json.Unmarshal(data, &st) == nil {
			model := st.Model.ID
			if model == "" {
				model = st.Model.Name
			}
			s.mu.Lock()
			s.model = model
			s.mu.Unlock()
		}
	}

	if row, err := s.store.SumUsageBySession(ctx, s.ID); err == nil {
		s.mu.Lock()
		s.lastUse = row
		s.mu.Unlock()
	}
}

// ModelInfo is one selectable Pi model.
type ModelInfo struct {
	Provider string
	ID       string
	Name     string
	// Reasoning is whether the model supports thinking at all.
	Reasoning bool
	// ThinkingLevels are the levels the model supports, in canonical order,
	// always including "off".
	ThinkingLevels []string
}

// SessionConfig is a session's live model and reasoning configuration.
type SessionConfig struct {
	ModelProvider  string
	ModelID        string
	ModelName      string
	Models         []ModelInfo
	ThinkingLevel  string
	ThinkingLevels []string
}

// availableModelsData is Pi's `get_available_models` payload.
type availableModelsData struct {
	Models []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
		// Reasoning is whether the model supports thinking at all.
		Reasoning bool `json:"reasoning"`
		// ThinkingLevelMap maps a canonical level to the provider-specific value
		// (nil when the model does not support it).
		ThinkingLevelMap map[string]*string `json:"thinkingLevelMap"`
	} `json:"models"`
}

// baseThinkingLevels are the levels every reasoning model supports unless its
// thinkingLevelMap explicitly maps one to null (unsupported).
var baseThinkingLevels = []string{"minimal", "low", "medium", "high"}

// extendedThinkingLevels are opt-in and included only when a model maps them.
var extendedThinkingLevels = []string{"xhigh", "max"}

// modelThinkingLevels derives the levels a model supports from Pi's
// thinkingLevelMap, in Pi's canonical order. "off" is always selectable; a
// model without reasoning support reports only ["off"].
func modelThinkingLevels(reasoning bool, levelMap map[string]*string) []string {
	levels := []string{"off"}
	if !reasoning {
		return levels
	}
	for _, level := range baseThinkingLevels {
		if value, ok := levelMap[level]; ok && value == nil {
			continue
		}
		levels = append(levels, level)
	}
	for _, level := range extendedThinkingLevels {
		if value, ok := levelMap[level]; ok && value != nil {
			levels = append(levels, level)
		}
	}
	return levels
}

// thinkingLevelsData is Pi's `get_available_thinking_levels` payload.
type thinkingLevelsData struct {
	Levels []string `json:"levels"`
}

// configState is the model/thinking slice of Pi's `get_state` response.
type configState struct {
	Model struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
	} `json:"model"`
	ThinkingLevel string `json:"thinkingLevel"`
}

// Config reports the session's current model and reasoning level plus the
// models and levels it can switch to. A reaped session is started so it can
// still be inspected and configured.
func (s *Session) Config(ctx context.Context) (SessionConfig, error) {
	if err := s.ensureRunningForConfig(ctx); err != nil {
		return SessionConfig{}, err
	}

	// Schedule the idle reap after the reads, not before: a short IdleTTL must
	// not reap the process mid-call. Probing is stopped by its caller instead.
	if !s.probe {
		defer s.scheduleIdleReap()
	}

	var cfg SessionConfig
	if data, err := s.send(ctx, "get_state", nil); err == nil {
		var st configState
		if json.Unmarshal(data, &st) == nil {
			cfg.ModelProvider = st.Model.Provider
			cfg.ModelID = st.Model.ID
			cfg.ModelName = st.Model.Name
			cfg.ThinkingLevel = st.ThinkingLevel
		}
	}

	if data, err := s.send(ctx, "get_available_models", nil); err == nil {
		var models availableModelsData
		if json.Unmarshal(data, &models) == nil {
			for _, m := range models.Models {
				cfg.Models = append(cfg.Models, ModelInfo{
					Provider:       m.Provider,
					ID:             m.ID,
					Name:           m.Name,
					Reasoning:      m.Reasoning,
					ThinkingLevels: modelThinkingLevels(m.Reasoning, m.ThinkingLevelMap),
				})
			}
		}
	}

	if data, err := s.send(ctx, "get_available_thinking_levels", nil); err == nil {
		var levels thinkingLevelsData
		if json.Unmarshal(data, &levels) == nil {
			cfg.ThinkingLevels = levels.Levels
		}
	}

	s.setModel(cfg)
	return cfg, nil
}

// SetModel switches the session's model and returns the refreshed config.
func (s *Session) SetModel(ctx context.Context, provider, modelID string) (SessionConfig, error) {
	if provider == "" || modelID == "" {
		return SessionConfig{}, ErrInvalidModel
	}
	if err := s.ensureRunningForConfig(ctx); err != nil {
		return SessionConfig{}, err
	}
	if _, err := s.send(ctx, "set_model", map[string]any{
		"provider": provider,
		"modelId":  modelID,
	}); err != nil {
		return SessionConfig{}, err
	}
	return s.Config(ctx)
}

// SetThinkingLevel sets the session's reasoning level and returns the
// refreshed config.
func (s *Session) SetThinkingLevel(ctx context.Context, level string) (SessionConfig, error) {
	if level == "" {
		return SessionConfig{}, ErrInvalidThinkingLevel
	}
	if err := s.ensureRunningForConfig(ctx); err != nil {
		return SessionConfig{}, err
	}
	if _, err := s.send(ctx, "set_thinking_level", map[string]any{"level": level}); err != nil {
		return SessionConfig{}, err
	}
	return s.Config(ctx)
}

// ensureRunningForConfig starts Pi if needed for a config-only call. The caller
// reschedules the idle reap once its reads finish, so inspecting a reaped
// session does not leave a process warm forever.
func (s *Session) ensureRunningForConfig(ctx context.Context) error {
	return s.ensureRunning(ctx)
}

// setModel records the selected model for usage rows.
func (s *Session) setModel(cfg SessionConfig) {
	model := cfg.ModelID
	if model == "" {
		model = cfg.ModelName
	}
	if model == "" {
		return
	}
	s.mu.Lock()
	s.model = model
	s.mu.Unlock()
}

type sessionStats struct {
	Tokens struct {
		Input  int64 `json:"input"`
		Output int64 `json:"output"`
	} `json:"tokens"`
	Cost float64 `json:"cost"`
	// ContextUsage is Pi's estimate of the live context window; absent when the
	// model has no context window. `tokens` is null after a compaction until an
	// assistant reports usage again.
	ContextUsage *struct {
		Tokens        *int64 `json:"tokens"`
		ContextWindow int64  `json:"contextWindow"`
	} `json:"contextUsage"`
}

// settle runs after `agent_settled`: it records the run's usage delta, clears
// the pending prompt, and reaps the Pi process (per-run lifetime).
func (s *Session) settle() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	running := s.cmd != nil
	s.mu.Unlock()
	if !running {
		return
	}

	if data, err := s.send(ctx, "get_session_stats", nil); err == nil {
		var st sessionStats
		if json.Unmarshal(data, &st) == nil {
			s.recordUsage(ctx, st)
		}
	} else {
		log.Printf("[session %s] get_session_stats: %v", s.ID, err)
	}

	if err := s.store.ClearSessionPendingPrompt(ctx, sqlitedb.ClearSessionPendingPromptParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        s.ID,
	}); err != nil {
		log.Printf("[session %s] clear pending prompt: %v", s.ID, err)
	}

	s.setRunning(false)
	slog.Info("run settled", "session_id", s.ID, "task_id", s.agencyTaskID)

	// A settled session keeps its Pi process warm for IdleTTL, then reaps it;
	// the next prompt resumes Pi with the same session id.
	s.clearWaitingStore(ctx)
	s.notifyAgentState("run_settled", StatusDone)
	s.scheduleIdleReap()
}

// Stop aborts a session's in-flight run without ending or archiving it. Pi's
// abort waits until it emits `agent_settled`, so the normal path defers to
// Pi's settle (usage, pending prompt, status); only a failed abort is cleared
// locally so the console does not stay stuck. The process stays warm for the
// next prompt.
func (s *Session) Stop(ctx context.Context) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if !running {
		return nil
	}

	abortCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := s.send(abortCtx, "abort", nil); err != nil {
		// Pi could not take the abort (its process is gone): fold the
		// interruption into DONE locally so the console does not stay stuck.
		log.Printf("[session %s] abort: %v", s.ID, err)
		if err := s.store.ClearSessionPendingPrompt(ctx, sqlitedb.ClearSessionPendingPromptParams{
			UpdatedAt: time.Now().UnixMilli(),
			ID:        s.ID,
		}); err != nil {
			log.Printf("[session %s] clear pending prompt: %v", s.ID, err)
		}
		s.setRunning(false)
		s.clearWaitingStore(ctx)
		s.emit(statusEvent(StatusDone, ""))
		s.notifyAgentState("run_stopped", StatusDone)
	}
	return nil
}

// ClearQueue drops every message Pi has queued but not injected yet. Pi emits
// a fresh queue_update, so watchers see the queue empty. It is a no-op when the
// Pi process is not running.
func (s *Session) ClearQueue(ctx context.Context) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	hasProcess := s.stdin != nil
	s.mu.Unlock()
	if !hasProcess {
		return nil
	}

	_, err := s.send(ctx, "clear_queue", nil)
	return err
}

// recordUsage inserts a `usage` row for the delta since the last settled run,
// including the run's active work time (idle gaps and user-input waits are
// excluded).
func (s *Session) recordUsage(ctx context.Context, st sessionStats) {
	s.mu.Lock()
	model := s.model
	prev := s.lastUse
	activeMs := s.runActiveDurationLocked(time.Now()).Milliseconds()
	s.mu.Unlock()

	deltaIn := max(st.Tokens.Input-prev.InputTokens, 0)
	deltaOut := max(st.Tokens.Output-prev.OutputTokens, 0)
	deltaCost := max(st.Cost-prev.Cost, 0)

	if deltaIn == 0 && deltaOut == 0 && deltaCost == 0 && activeMs <= 0 {
		return
	}

	err := s.store.InsertUsage(ctx, sqlitedb.InsertUsageParams{
		SessionID:    s.ID,
		Model:        model,
		InputTokens:  deltaIn,
		OutputTokens: deltaOut,
		Cost:         deltaCost,
		ActiveMs:     activeMs,
		At:           time.Now().UnixMilli(),
	})
	if err != nil {
		log.Printf("[session %s] insert usage: %v", s.ID, err)
		return
	}

	// Publish the delta so the agency can aggregate usage across agents.
	s.notify("usage", map[string]any{
		"model":         model,
		"input_tokens":  deltaIn,
		"output_tokens": deltaOut,
		"cost":          deltaCost,
		"active_ms":     activeMs,
		"at":            time.Now().UnixMilli(),
	})

	s.mu.Lock()
	s.lastUse = sqlitedb.SumUsageBySessionRow{
		InputTokens:  prev.InputTokens + deltaIn,
		OutputTokens: prev.OutputTokens + deltaOut,
		Cost:         prev.Cost + deltaCost,
		ActiveMs:     prev.ActiveMs + activeMs,
	}
	s.mu.Unlock()
}

// send writes one RPC command and waits for its matching response.
func (s *Session) send(
	ctx context.Context,
	cmdType string,
	fields map[string]any,
) (json.RawMessage, error) {
	s.mu.Lock()
	if s.stdin == nil {
		s.mu.Unlock()
		return nil, errors.New("pi process is not running")
	}
	s.reqID++
	id := strconv.FormatUint(s.reqID, 10)
	ch := make(chan response, 1)
	s.pending[id] = ch
	stdin := s.stdin
	s.mu.Unlock()

	cmd := map[string]any{"id": id, "type": cmdType}
	for k, v := range fields {
		cmd[k] = v
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		s.dropPending(id)
		return nil, err
	}
	payload = append(payload, '\n')

	s.writeMu.Lock()
	_, err = stdin.Write(payload)
	s.writeMu.Unlock()
	if err != nil {
		s.dropPending(id)
		return nil, fmt.Errorf("write %s: %w", cmdType, err)
	}

	select {
	case resp := <-ch:
		if !resp.Success {
			return nil, fmt.Errorf("%s: %s", cmdType, resp.Error)
		}
		return resp.Data, nil
	case <-ctx.Done():
		s.dropPending(id)
		return nil, ctx.Err()
	}
}

func (s *Session) dropPending(id string) {
	s.mu.Lock()
	delete(s.pending, id)
	s.mu.Unlock()
}

func (s *Session) readStdout(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		s.handleRecord(scanner.Bytes())
	}
}

func (s *Session) handleRecord(b []byte) {
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil {
		return
	}

	switch rec.Type {
	case "response":
		s.mu.Lock()
		ch := s.pending[rec.ID]
		delete(s.pending, rec.ID)
		s.mu.Unlock()
		if ch != nil {
			ch <- response{
				Success: rec.Success == nil || *rec.Success,
				Error:   rec.Error,
				Data:    rec.Data,
			}
		}

	case "agent_start":
		s.setRunning(true)
		slog.Info("run started", "session_id", s.ID, "task_id", s.agencyTaskID)
		s.emit(Event{Kind: EventRunStarted})
		s.notifyAgentState("run_started", StatusInProgress)

	case "agent_settled":
		s.emit(Event{Kind: EventRunSettled})
		go s.settle()

	case "message_update":
		var ev struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		if json.Unmarshal(rec.AssistantMessageEvent, &ev) != nil {
			return
		}
		switch ev.Type {
		case "text_delta":
			s.emit(Event{Kind: EventText, Delta: ev.Delta})
		case "thinking_delta":
			s.emit(Event{Kind: EventThinking, Delta: ev.Delta})
		}

	case "message_start":
		// The user's turn, whether it is the initial prompt or a steering
		// message Pi just injected between blocks.
		if text := userMessageText(rec.Message); text != "" {
			s.emit(Event{Kind: EventUser, Text: text})
		}

	case "queue_update":
		s.setQueue(rec.Steering, rec.FollowUp)

	case "tool_execution_start":
		s.emit(toolEvent(rec, ToolStarted))
	case "tool_execution_update":
		s.emit(toolEvent(rec, ToolUpdated))
	case "tool_execution_end":
		s.emit(toolEvent(rec, ToolFinished))

	case "entry_appended":
		s.handleEntryAppended(b)

	case "extension_ui_request":
		s.handleExtensionUI(b)
	}
}

func toolEvent(rec record, phase ToolPhase) Event {
	result := rec.Result
	if phase == ToolUpdated {
		result = rec.PartialResult
	}
	details := ""
	if phase == ToolFinished {
		details = resultDetails(rec.Result)
	}
	return Event{
		Kind: EventTool,
		Tool: &ToolEvent{
			CallID:      rec.ToolCallID,
			Name:        rec.ToolName,
			Phase:       phase,
			ArgsJSON:    rawString(rec.Args),
			ResultJSON:  rawString(result),
			DetailsJSON: details,
			IsError:     rec.IsError,
		},
	}
}

// resultDetails extracts the Pi tool result's structured `details` as raw JSON.
// A result without details (or a non-object result) yields an empty string.
func resultDetails(result json.RawMessage) string {
	if len(result) == 0 {
		return ""
	}
	var envelope struct {
		Details json.RawMessage `json:"details"`
	}
	if json.Unmarshal(result, &envelope) != nil || len(envelope.Details) == 0 {
		return ""
	}
	return string(envelope.Details)
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return string(raw)
}

// Custom Pi entries the bundled extensions append, and the agent events they
// become.
const (
	taskURLsCustomType      = "cherrypi_task_urls"
	taskArtifactsCustomType = "cherrypi_task_artifacts"
)

// handleEntryAppended forwards the custom entries the bundled setTaskUrls and
// setTaskArtifacts tools record to the agency through the event hub. Other
// custom entries are ignored.
func (s *Session) handleEntryAppended(b []byte) {
	var rec struct {
		Entry struct {
			Type       string          `json:"type"`
			CustomType string          `json:"customType"`
			Data       json.RawMessage `json:"data"`
		} `json:"entry"`
	}
	if json.Unmarshal(b, &rec) != nil || rec.Entry.Type != "custom" {
		return
	}

	switch rec.Entry.CustomType {
	case taskURLsCustomType:
		s.handleTaskURLsEntry(rec.Entry.Data)
	case taskArtifactsCustomType:
		s.handleTaskArtifactsEntry(rec.Entry.Data)
	}
}

// handleTaskURLsEntry forwards a task's complete labeled-URL list.
func (s *Session) handleTaskURLsEntry(raw json.RawMessage) {
	var data struct {
		URLs []struct {
			URL   string `json:"url"`
			Label string `json:"label"`
		} `json:"urls"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return
	}

	links := make([]map[string]any, 0, len(data.URLs))
	for _, u := range data.URLs {
		if u.URL == "" {
			continue
		}
		links = append(links, map[string]any{"url": u.URL, "label": u.Label})
	}
	s.notify("task_link", map[string]any{"links": links})
}

// handleTaskArtifactsEntry forwards the artifacts the setTaskArtifacts tool
// published (A2A task outputs). The parts pass through as already-shaped JSON.
func (s *Session) handleTaskArtifactsEntry(raw json.RawMessage) {
	var data struct {
		Replace   bool             `json:"replace"`
		Artifacts []map[string]any `json:"artifacts"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return
	}
	s.notify("task_artifact", map[string]any{
		"replace":   data.Replace,
		"artifacts": data.Artifacts,
	})
}

func (s *Session) readStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		s.mu.Lock()
		s.logRing = append(s.logRing, line)
		if len(s.logRing) > logRingMax {
			s.logRing = s.logRing[len(s.logRing)-logRingMax:]
		}
		s.mu.Unlock()

		// Pi's stderr is high-volume and already streamed per task; keep it on
		// stdout only, out of the agency log mirror.
		logging.Local().Info(line, "session_id", s.ID)
		s.emit(Event{Kind: EventLog, Line: line})
	}
}

// reap waits for the process and marks the session failed on an unexpected
// exit (a process stopped by settle/end has `stopping` set).
func (s *Session) reap(cmd *exec.Cmd) {
	err := cmd.Wait()

	s.mu.Lock()
	expected := s.stopping || s.cmd != cmd
	if s.cmd == cmd {
		s.cmd = nil
		s.stdin = nil
		s.pending = nil
	}
	exited := s.exited
	s.mu.Unlock()

	select {
	case <-exited:
	default:
		close(exited)
	}

	// Pi has stopped appending; persist whatever is left before the file can
	// be replaced or the workspace cleaned up.
	s.stopHistoryWatch()

	if expected {
		slog.Info("pi exited", "session_id", s.ID)
		return
	}

	msg := "pi process exited unexpectedly"
	if err != nil {
		msg = err.Error()
	}
	slog.Warn("pi exited unexpectedly",
		"session_id", s.ID,
		"task_id", s.agencyTaskID,
		"reason", msg,
	)
	go s.fail(msg)
}

// stopProcess closes stdin (orderly shutdown), then SIGKILLs after the grace
// period, and clears the recorded pid. It is idempotent.
func (s *Session) stopProcess() {
	s.cancelIdle()

	s.mu.Lock()
	cmd := s.cmd
	stdin := s.stdin
	exited := s.exited
	s.cmd = nil
	s.stdin = nil
	s.pending = nil
	s.stopping = true
	s.mu.Unlock()

	defer s.clearPid()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd == nil || cmd.Process == nil || exited == nil {
		return
	}

	select {
	case <-exited:
	case <-time.After(s.cfg.EndGrace):
		_ = cmd.Process.Kill()
		<-exited
	}
}

func (s *Session) clearPid() {
	if s.probe {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.store.SetSessionPid(ctx, sqlitedb.SetSessionPidParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        s.ID,
	}); err != nil {
		log.Printf("[session %s] clear pid: %v", s.ID, err)
	}
}

// forceStop SIGKILLs the process without waiting on opMu, for forced shutdown.
func (s *Session) forceStop() {
	s.cancelIdle()

	s.mu.Lock()
	cmd := s.cmd
	s.cmd = nil
	s.stdin = nil
	s.pending = nil
	s.stopping = true
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// fail records a terminal failure.
func (s *Session) fail(msg string) {
	if s.probe {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.setRunning(false)
	slog.Error("run failed", "session_id", s.ID, "task_id", s.agencyTaskID, "error", msg)

	now := time.Now().UnixMilli()
	if err := s.store.EndSession(ctx, sqlitedb.EndSessionParams{
		Status:    int64(StatusFailed),
		Error:     sql.NullString{String: msg, Valid: true},
		EndedAt:   sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt: now,
		ID:        s.ID,
	}); err != nil {
		log.Printf("[session %s] mark failed: %v", s.ID, err)
	}

	s.emit(statusEvent(StatusFailed, msg))
	s.notifyAgentState("failed", StatusFailed)
}

// historyHasPrompt reports whether the last user message in Pi's history is the
// given prompt — used to decide between re-sending and nudging on recovery.
func (s *Session) historyHasPrompt(ctx context.Context, prompt string) bool {
	data, err := s.send(ctx, "get_messages", nil)
	if err != nil {
		return false
	}

	var msgs struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(data, &msgs) != nil {
		return false
	}

	for i := len(msgs.Messages) - 1; i >= 0; i-- {
		if msgs.Messages[i].Role != "user" {
			continue
		}
		return contentText(msgs.Messages[i].Content) == prompt
	}
	return false
}

func contentText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}

	out := ""
	for _, b := range blocks {
		out += b.Text
	}
	return out
}

// writeRaw writes one JSONL record to Pi without waiting for a response
// (extension_ui_response, which Pi never acknowledges).
func (s *Session) writeRaw(v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	s.mu.Lock()
	stdin := s.stdin
	s.mu.Unlock()
	if stdin == nil {
		return errors.New("pi process is not running")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err = stdin.Write(payload)
	return err
}

// handleExtensionUI forwards an extension-UI request. Dialog methods block the
// run: interactive sessions surface them (persist waiting + go waiting);
// non-interactive sessions auto-decline with `cancelled: true`. A dialog
// re-emitted after a restart is answered from `savedAnswer` without surfacing.
// Fire-and-forget methods are forwarded without expecting a response.
func (s *Session) handleExtensionUI(raw []byte) {
	var req extensionUIRequest
	if json.Unmarshal(raw, &req) != nil || req.ID == "" {
		return
	}
	req.RawJSON = string(raw)

	if !isDialogMethod(req.Method) {
		s.emit(Event{Kind: EventExtensionUI, Extension: extensionEvent(req)})
		return
	}

	s.mu.Lock()
	answer := s.savedAnswer
	interactive := s.interactive
	if answer != nil {
		s.savedAnswer = nil
		// The persisted dialog is resolved; stop reporting the session as
		// waiting so the console hides the wizard and does not replay it.
		s.savedDialog = ""
	} else if interactive {
		s.pendingExt[req.ID] = req
		if !s.waiting {
			s.waitStartedAt = time.Now()
		}
		s.waiting = true
		s.lastDialogJSON = req.RawJSON
	}
	s.mu.Unlock()

	if answer != nil {
		// Replayed dialog after a restart: answer it without surfacing.
		s.writeExtensionResponse(req.ID, *answer)
		s.clearWaitingStore(context.Background())
		s.notifyAgentState("waiting_resolved", StatusInProgress)
		return
	}

	if !interactive {
		// Nothing will answer this dialog: decline it immediately.
		s.writeExtensionResponse(req.ID, ExtensionResponse{Kind: ExtensionResponseCancelled})
		return
	}

	s.persistWaiting(req.RawJSON)
	payload := map[string]any{
		"request_id": req.ID,
		"method":     req.Method,
		"title":      req.Title,
		"message":    req.Message,
		"options":    req.Options,
	}
	// CherryPi's batch questionnaire carries every question in the dialog, so
	// the webhook delivers the whole set at once instead of one per question.
	if questions := questionnaireQuestions(req); questions != nil {
		payload["questions"] = questions
	}
	s.notify("user_input_requested", payload)
	s.notifyAgentState("waiting_for_input", StatusWaitingInput)
	s.emit(Event{Kind: EventExtensionUI, Extension: extensionEvent(req)})
}

// questionnaireQuestions extracts the question set from CherryPi's batch
// questionnaire dialog. The extension sends one `editor` request whose prefill
// is a JSON object marked `kind: "cherrypi.questionnaire"`; other editor
// dialogs (and any prefill that is not that shape) return nil.
func questionnaireQuestions(req extensionUIRequest) []map[string]any {
	if req.Method != "editor" || req.Prefill == "" {
		return nil
	}
	var payload struct {
		Kind      string           `json:"kind"`
		Questions []map[string]any `json:"questions"`
	}
	if json.Unmarshal([]byte(req.Prefill), &payload) != nil ||
		payload.Kind != "cherrypi.questionnaire" {
		return nil
	}
	return payload.Questions
}

func extensionEvent(req extensionUIRequest) *ExtensionUIEvent {
	return &ExtensionUIEvent{
		RequestID:   req.ID,
		Method:      req.Method,
		Title:       req.Title,
		Message:     req.Message,
		Placeholder: req.Placeholder,
		Prefill:     req.Prefill,
		Options:     req.Options,
		Timeout:     req.Timeout,
		RawJSON:     req.RawJSON,
	}
}

// writeExtensionResponse writes one `extension_ui_response` record to Pi.
func (s *Session) writeExtensionResponse(requestID string, resp ExtensionResponse) {
	fields := map[string]any{"type": "extension_ui_response", "id": requestID}
	switch resp.Kind {
	case ExtensionResponseValue:
		fields["value"] = resp.Value
	case ExtensionResponseConfirmed:
		fields["confirmed"] = resp.Confirmed
	default:
		fields["cancelled"] = true
	}
	if err := s.writeRaw(fields); err != nil {
		log.Printf("[session %s] extension response: %v", s.ID, err)
	}
}

// respondExtensionUI answers a pending dialog and writes the response to Pi. If
// the dialog was persisted (the server restarted while waiting), it resumes Pi
// with the answer in the prompt and keeps the answer for the re-emitted dialog
// in case the model re-invokes the tool.
func (s *Session) respondExtensionUI(
	ctx context.Context,
	requestID string,
	resp ExtensionResponse,
) error {
	s.mu.Lock()
	_, live := s.pendingExt[requestID]
	if live {
		delete(s.pendingExt, requestID)
		if len(s.pendingExt) == 0 {
			s.waiting = false
			s.lastDialogJSON = ""
			s.accumulateWaitLocked(time.Now())
		}
	}
	saved := s.savedDialog != ""
	question := savedDialogTitle(s.savedDialog)
	if !live && saved {
		s.savedDialog = ""
		answer := resp
		s.savedAnswer = &answer
	}
	s.mu.Unlock()

	if live {
		s.writeExtensionResponse(requestID, resp)
		s.clearWaitingStore(ctx)
		s.notifyAgentState("waiting_resolved", StatusInProgress)
		return nil
	}

	if saved {
		s.clearWaitingStore(ctx)
		go s.resumeAfterRestart(question, resp)
		return nil
	}

	return ErrNoPendingDialog
}

// resumeAfterRestart resumes a run blocked on a dialog the server lost to a
// restart. Pi does not re-execute the dangling tool call on its own and a model
// told only that the run was interrupted often continues without the answer, so
// the resume prompt carries the user's answer: the model proceeds with it even
// when it never re-invokes the tool. If it does re-invoke, handleExtensionUI
// consumes the saved answer as before.
func (s *Session) resumeAfterRestart(question string, resp ExtensionResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := s.prompt(ctx, resumeDialogNudge(question, resp)); err != nil {
		log.Printf("[session %s] resume for dialog: %v", s.ID, err)
	}
}

// resumeDialogNudge is the resume prompt for an answered persisted dialog. It
// sends the answer instead of a bare interruption notice, so a model that does
// not re-issue the tool call still has what it asked for. When the console
// submitted a whole ask_user_question wizard, every answer is listed even
// though only one dialog was persisted.
func resumeDialogNudge(question string, resp ExtensionResponse) string {
	if len(resp.Questionnaire) > 0 {
		return fmt.Sprintf(
			"The user answered your questions before the restart: %s. Continue from where you left off and finish the task.",
			formatQuestionnaire(resp.Questionnaire),
		)
	}

	answer := ""
	switch resp.Kind {
	case ExtensionResponseValue:
		answer = fmt.Sprintf("%q", resp.Value)
	case ExtensionResponseConfirmed:
		if resp.Confirmed {
			answer = "yes"
		} else {
			answer = "no"
		}
	default:
		answer = "no answer (declined)"
	}

	if question == "" {
		return fmt.Sprintf(
			"The user answered your pending question: %s. Continue from where you left off and finish the task.",
			answer,
		)
	}
	return fmt.Sprintf(
		"The user answered your pending question %q: %s. Continue from where you left off and finish the task.",
		question,
		answer,
	)
}

// formatQuestionnaire renders question/answer pairs as `"Q"="A"` segments.
func formatQuestionnaire(answers []QuestionnaireAnswer) string {
	segments := make([]string, 0, len(answers))
	for _, qa := range answers {
		segments = append(segments, fmt.Sprintf("%q=%q", qa.Question, qa.Answer))
	}
	return strings.Join(segments, "; ")
}

// savedDialogTitle extracts the question text from a persisted dialog, so the
// resume prompt can refer to what was asked.
func savedDialogTitle(raw string) string {
	var req extensionUIRequest
	if json.Unmarshal([]byte(raw), &req) != nil {
		return ""
	}
	return req.Title
}

// persistWaiting records the pending dialog so it survives a restart.
func (s *Session) persistWaiting(dialogJSON string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.store.SetSessionWaiting(ctx, sqlitedb.SetSessionWaitingParams{
		PendingDialog: sql.NullString{String: dialogJSON, Valid: true},
		UpdatedAt:     time.Now().UnixMilli(),
		ID:            s.ID,
	}); err != nil {
		log.Printf("[session %s] persist waiting: %v", s.ID, err)
	}
}

func (s *Session) clearWaitingStore(ctx context.Context) {
	if err := s.store.ClearSessionWaiting(ctx, sqlitedb.ClearSessionWaitingParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        s.ID,
	}); err != nil {
		log.Printf("[session %s] clear waiting: %v", s.ID, err)
	}
}

// notify publishes an event with the session id attached.
func (s *Session) notify(event string, payload map[string]any) {
	if s.cfg.Notify == nil {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["session_id"] = s.ID
	s.cfg.Notify(event, payload)
}

func (s *Session) notifyAgentState(state string, status Status) {
	s.notify("agent_state_changed", map[string]any{"state": state, "status": int64(status)})
}

// scheduleIdleReap reaps the Pi process after IdleTTL unless a new prompt
// supersedes the timer.
func (s *Session) scheduleIdleReap() {
	ttl := s.cfg.IdleTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	s.mu.Lock()
	s.idleGen++
	gen := s.idleGen
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	s.idleTimer = time.AfterFunc(ttl, func() { s.reapIdle(gen) })
	s.mu.Unlock()
}

func (s *Session) reapIdle(gen uint64) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	stale := gen != s.idleGen || s.running || s.cmd == nil
	s.mu.Unlock()
	if stale {
		return
	}

	s.stopProcess()
}

// cancelIdle invalidates any pending idle reap.
func (s *Session) cancelIdle() {
	s.mu.Lock()
	s.idleGen++
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	s.mu.Unlock()
}

// waitingForInput reports the waiting state and the pending dialog JSON. A
// session that was waiting when the server restarted reports its persisted
// dialog too.
func (s *Session) waitingForInput() (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.waiting {
		return true, s.lastDialogJSON
	}
	if s.savedDialog != "" {
		return true, s.savedDialog
	}
	return false, ""
}

// pendingDialogEvent reconstructs the extension-UI event for a dialog the user
// has not answered yet. Pi persists no extension-UI records, so a watcher that
// attaches after the dialog was first emitted (or after a server restart) would
// otherwise never learn the session is blocked on a question. Returns false when
// nothing is pending.
func (s *Session) pendingDialogEvent() (Event, bool) {
	s.mu.Lock()
	raw := s.lastDialogJSON
	if !s.waiting {
		raw = s.savedDialog
	}
	s.mu.Unlock()

	if raw == "" {
		return Event{}, false
	}
	var req extensionUIRequest
	if json.Unmarshal([]byte(raw), &req) != nil || req.ID == "" || !isDialogMethod(req.Method) {
		return Event{}, false
	}
	req.RawJSON = raw
	return Event{Kind: EventExtensionUI, Extension: extensionEvent(req)}, true
}

func (s *Session) clearWaiting() {
	s.mu.Lock()
	s.waiting = false
	s.pendingExt = map[string]extensionUIRequest{}
	s.lastDialogJSON = ""
	s.accumulateWaitLocked(time.Now())
	s.mu.Unlock()
}

// accumulateWaitLocked folds an in-flight user-input wait into the current
// run's accumulated wait time. Callers must hold s.mu.
func (s *Session) accumulateWaitLocked(now time.Time) {
	if s.waitStartedAt.IsZero() {
		return
	}
	s.runWaitAccum += now.Sub(s.waitStartedAt)
	s.waitStartedAt = time.Time{}
}

// runActiveDurationLocked returns the current run's active work time, excluding
// time spent waiting on user input. Callers must hold s.mu.
func (s *Session) runActiveDurationLocked(now time.Time) time.Duration {
	if s.runStartedAt.IsZero() {
		return 0
	}
	d := now.Sub(s.runStartedAt) - s.runWaitAccum
	if !s.waitStartedAt.IsZero() {
		d -= now.Sub(s.waitStartedAt)
	}
	if d < 0 {
		return 0
	}
	return d
}

func (s *Session) processRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cmd != nil
}

// liveUsage reads Pi's cumulative session stats for a running process, plus
// the in-flight run's token/cost rates when a run is active.
func (s *Session) liveUsage(ctx context.Context) (Usage, bool) {
	data, err := s.send(ctx, "get_session_stats", nil)
	if err != nil {
		return Usage{}, false
	}
	var st sessionStats
	if json.Unmarshal(data, &st) != nil {
		return Usage{}, false
	}

	s.mu.Lock()
	running := s.running
	started := s.runStartedAt
	base := s.lastUse
	active := s.runActiveDurationLocked(time.Now())
	s.mu.Unlock()

	out := Usage{
		InputTokens:  st.Tokens.Input,
		OutputTokens: st.Tokens.Output,
		Cost:         st.Cost,
		ActiveMs:     base.ActiveMs,
	}
	// Context usage is unknown after a compaction until an assistant reports
	// usage again; surface the meter only when both values are known.
	if st.ContextUsage != nil &&
		st.ContextUsage.Tokens != nil &&
		st.ContextUsage.ContextWindow > 0 {
		out.ContextTokens = *st.ContextUsage.Tokens
		out.ContextWindow = st.ContextUsage.ContextWindow
	}
	if running && !started.IsZero() {
		out.TokensPerSecond, out.CostPerHour = runRates(st, base, time.Since(started))
		out.ActiveMs += active.Milliseconds()
	}
	return out, true
}

// runRates derives a run's token rate and extrapolated hourly cost from Pi's
// cumulative stats, the settled baseline, and the run's elapsed time.
func runRates(
	st sessionStats,
	base sqlitedb.SumUsageBySessionRow,
	elapsed time.Duration,
) (tokensPerSecond, costPerHour float64) {
	seconds := elapsed.Seconds()
	if seconds <= 0 {
		return 0, 0
	}
	tokens := float64(
		st.Tokens.Input-base.InputTokens,
	) + float64(
		st.Tokens.Output-base.OutputTokens,
	)
	cost := st.Cost - base.Cost
	return math.Max(tokens, 0) / seconds, math.Max(cost, 0) / seconds * 3600
}

// Usage is a session's token/cost totals plus live run rates.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	Cost         float64
	// ActiveMs is the accumulated active work time in milliseconds (idle gaps
	// and user-input waits excluded), including any in-flight run.
	ActiveMs int64
	// TokensPerSecond and CostPerHour are the in-flight run's rates; zero when
	// no run is active.
	TokensPerSecond float64
	CostPerHour     float64
	// ContextTokens/ContextWindow are the live Pi session's context-window
	// usage; zero when unknown (see liveUsage).
	ContextTokens int64
	ContextWindow int64
}

func (s *Session) emit(ev Event) {
	if ev.At.IsZero() {
		ev.At = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- ev:
		default:
			// ponytail: slow watcher drops events rather than stalling the run.
		}
	}
}

// subscribe returns a channel of events plus an unsubscribe func. `history`
// (replayed conversation) and `initial` (current status) are delivered before
// any live event, so a watcher always sees the transcript then the state.
func (s *Session) subscribe(initial Event, history ...Event) (<-chan Event, func()) {
	if initial.At.IsZero() {
		initial.At = time.Now()
	}
	for i := range history {
		if history[i].At.IsZero() {
			history[i].At = time.Now()
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan Event, 256+len(history))

	for _, ev := range history {
		ch <- ev
	}
	ch <- initial
	// Tell a late watcher Pi's current queue, so a reload shows what is still
	// waiting to be injected.
	queue := QueueEvent{
		Steering: append([]string(nil), s.queue.Steering...),
		FollowUp: append([]string(nil), s.queue.FollowUp...),
	}
	ch <- Event{Kind: EventQueue, At: time.Now(), Queue: &queue}
	s.subs[ch] = struct{}{}

	return ch, func() {
		s.mu.Lock()
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
		s.mu.Unlock()
	}
}

// setQueue records Pi's pending-message queue and tells watchers, so a console
// can show what is waiting and shrink it as Pi injects each message.
func (s *Session) setQueue(steering, followUp []string) {
	queue := QueueEvent{
		Steering: append([]string(nil), steering...),
		FollowUp: append([]string(nil), followUp...),
	}

	s.mu.Lock()
	s.queue = queue
	s.mu.Unlock()

	s.emit(Event{Kind: EventQueue, Queue: &queue})
}

// userMessageText extracts the text of a `message_start` record when its
// message is the user's turn; it returns "" for any other role or shape.
func userMessageText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var msg struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &msg) != nil || msg.Role != "user" {
		return ""
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(msg.Content, &parts) != nil {
		return ""
	}

	var b strings.Builder
	for _, part := range parts {
		if part.Type == "text" {
			b.WriteString(part.Text)
		}
	}
	return b.String()
}
