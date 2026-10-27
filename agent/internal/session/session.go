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
	"bytes"
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

const logRingMax = 200

var ErrNoPendingDialog = errors.New("no pending extension dialog with that request id")

const (
	questionnaireUnavailableKind = "cherrypi.questionnaire.unavailable"

	questionnaireUnavailableMessage = "This task is non-interactive, so the questionnaire was not shown. Ask the user in plain text instead."
)

type Config struct {
	PiBin string

	PiSessionDir string

	AgentDir string

	WorkspaceRoot string

	EndGrace time.Duration

	PiEnv []string

	PiArgs []string

	FetchAttachment func(ctx context.Context, url string) (io.ReadCloser, error)

	AgencyURL string

	AgentTokenFile string

	ResolveTaskID func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
	) (taskID string, err error)

	FetchTaskHistory func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
	) ([]SessionEntry, error)

	PushTaskHistory func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
		entries []SessionEntry,
	) error

	RefreshTaskHistoryAuth func(ctx context.Context) error

	HistorySyncInterval time.Duration

	IdleTTL time.Duration

	Notify func(event string, payload map[string]any)

	TitleGenerator TitleGenerator

	TitleTimeout time.Duration
}

type TitleGenerator func(ctx context.Context, prompt, provider, modelID, level string) (string, error)

type Session struct {
	cfg   Config
	store store.Store

	ID          string
	Workspace   string
	PiSessionID string

	agencyTaskID string

	agencyProjectID string
	agencySeq       int64

	opMu sync.Mutex

	mu       sync.Mutex
	started  bool
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	exited   chan struct{}
	stopping bool
	running  bool

	runStartedAt time.Time

	runWaitAccum  time.Duration
	waitStartedAt time.Time
	pending       map[string]chan response
	reqID         uint64
	model         string
	lastUse       sqlitedb.SumUsageBySessionRow
	subs          map[chan Event]struct{}
	logRing       []string
	writeMu       sync.Mutex

	interactive bool

	appendSystemPrompt string

	probe bool

	waiting bool

	queue QueueEvent

	pendingExt map[string]extensionUIRequest

	lastDialogJSON string

	savedDialog string

	savedAnswer *ExtensionResponse

	idleTimer *time.Timer
	idleGen   uint64

	runHook func(started bool)

	histMu            sync.Mutex
	syncedTip         string
	historyReconciled bool
	historyDisabled   bool

	historyFailures     int
	historyRetryAt      time.Time
	historyAuthAttempts int

	historyCancel context.CancelFunc
}

type response struct {
	Success bool
	Error   string
	Data    json.RawMessage
}

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

func isDialogMethod(method string) bool {
	switch method {
	case "select", "confirm", "input", "editor":
		return true
	default:
		return false
	}
}

type ExtensionResponseKind int

const (
	ExtensionResponseCancelled ExtensionResponseKind = iota
	ExtensionResponseValue
	ExtensionResponseConfirmed
)

type ExtensionResponse struct {
	Kind      ExtensionResponseKind
	Value     string
	Confirmed bool

	Questionnaire []QuestionnaireAnswer
}

type QuestionnaireAnswer struct {
	Question string
	Answer   string
}

type record struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Success *bool           `json:"success"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`

	AssistantMessageEvent json.RawMessage `json:"assistantMessageEvent"`
	Message               json.RawMessage `json:"message"`

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

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func (s *Session) prompt(ctx context.Context, text string) error {
	return s.promptImages(ctx, text, nil)
}

func (s *Session) promptImages(ctx context.Context, text string, images []imageAttachment) error {
	return s.promptConfigured(ctx, text, "", "", "", images)
}

func (s *Session) promptConfigured(
	ctx context.Context,
	text, provider, modelID, level string,
	images []imageAttachment,
) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

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

func (s *Session) agencyRef() (string, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agencyProjectID, s.agencySeq
}

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

	if !s.probe && s.historyEnabled() {
		s.resetHistorySync()
		s.syncHistory(ctx, true)
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

type ModelInfo struct {
	Provider string
	ID       string
	Name     string

	Reasoning bool

	ThinkingLevels []string
}

type SessionConfig struct {
	ModelProvider  string
	ModelID        string
	ModelName      string
	Models         []ModelInfo
	ThinkingLevel  string
	ThinkingLevels []string
}

type availableModelsData struct {
	Models []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`

		Reasoning bool `json:"reasoning"`

		ThinkingLevelMap map[string]*string `json:"thinkingLevelMap"`
	} `json:"models"`
}

var baseThinkingLevels = []string{"off", "minimal", "low", "medium", "high"}

var extendedThinkingLevels = []string{"xhigh", "max"}

func modelThinkingLevels(reasoning bool, levelMap map[string]*string) []string {
	if !reasoning {
		return []string{"off"}
	}
	levels := make([]string, 0, len(baseThinkingLevels)+len(extendedThinkingLevels))
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

type thinkingLevelsData struct {
	Levels []string `json:"levels"`
}

type configState struct {
	Model struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
	} `json:"model"`
	ThinkingLevel string `json:"thinkingLevel"`
}

func (s *Session) Config(ctx context.Context) (SessionConfig, error) {
	if err := s.ensureRunningForConfig(ctx); err != nil {
		return SessionConfig{}, err
	}

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

func (s *Session) ensureRunningForConfig(ctx context.Context) error {
	return s.ensureRunning(ctx)
}

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

	ContextUsage *struct {
		Tokens        *int64 `json:"tokens"`
		ContextWindow int64  `json:"contextWindow"`
	} `json:"contextUsage"`
}

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

	s.clearWaitingStore(ctx)
	s.notifyAgentState("run_settled", StatusDone)
	s.scheduleIdleReap()
}

const stopSettleTimeout = 10 * time.Second

func (s *Session) Stop(ctx context.Context) error {
	s.opMu.Lock()

	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if !running {

		err := s.foldInterruptedRun(ctx)
		s.opMu.Unlock()
		return err
	}

	abortCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	_, err := s.send(abortCtx, "abort", nil)
	cancel()
	if err != nil {
		log.Printf("[session %s] abort: %v", s.ID, err)
		s.finishStoppedRun(ctx)
		s.opMu.Unlock()
		return nil
	}
	s.opMu.Unlock()

	if s.waitSettled(ctx, stopSettleTimeout) {
		return nil
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	running = s.running
	s.mu.Unlock()
	if running {
		log.Printf(
			"[session %s] abort did not settle within %s; folding into done",
			s.ID,
			stopSettleTimeout,
		)
		s.finishStoppedRun(ctx)
	}
	return nil
}

func (s *Session) waitSettled(ctx context.Context, timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()
		if !running {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		case <-ticker.C:
		}
	}
}

func (s *Session) foldInterruptedRun(ctx context.Context) error {
	abortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	_, _ = s.send(abortCtx, "abort", nil)
	cancel()

	row, err := s.store.GetSession(ctx, s.ID)
	if err != nil {
		return err
	}
	if !row.PendingPrompt.Valid || row.PendingPrompt.String == "" {
		return nil
	}
	log.Printf("[session %s] folding interrupted run into done", s.ID)
	s.finishStoppedRun(ctx)
	return nil
}

func (s *Session) finishStoppedRun(ctx context.Context) {
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

	s.scheduleIdleReap()
}

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

	reader := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			s.handleRecord(bytes.TrimSuffix(line, []byte{'\n'}))
		}
		if err != nil {
			return
		}
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

const (
	taskURLsCustomType      = "cherrypi_task_urls"
	taskArtifactsCustomType = "cherrypi_task_artifacts"
)

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

		logging.Local().Info(line, "session_id", s.ID)
		s.emit(Event{Kind: EventLog, Line: line})
	}
}

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

func (s *Session) historyHasPrompt(prompt string) bool {
	all := s.readHistory()
	for i := len(all) - 1; i >= 0; i-- {
		ev := all[i].event
		if ev.Kind != EventUser {
			continue
		}
		return ev.Text == prompt
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

		s.writeExtensionResponse(req.ID, *answer)
		s.clearWaitingStore(context.Background())
		s.notifyAgentState("waiting_resolved", StatusInProgress)
		return
	}

	if !interactive {

		s.writeExtensionResponse(req.ID, nonInteractiveDialogResponse(req))
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

	if questions := questionnaireQuestions(req); questions != nil {
		payload["questions"] = questions
	}
	s.notify("user_input_requested", payload)
	s.notifyAgentState("waiting_for_input", StatusWaitingInput)
	s.emit(Event{Kind: EventExtensionUI, Extension: extensionEvent(req)})
}

func nonInteractiveDialogResponse(req extensionUIRequest) ExtensionResponse {
	if questionnaireQuestions(req) == nil {
		return ExtensionResponse{Kind: ExtensionResponseCancelled}
	}

	payload, err := json.Marshal(map[string]string{
		"kind":    questionnaireUnavailableKind,
		"message": questionnaireUnavailableMessage,
	})
	if err != nil {
		return ExtensionResponse{Kind: ExtensionResponseCancelled}
	}

	return ExtensionResponse{Kind: ExtensionResponseValue, Value: string(payload)}
}

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

func (s *Session) resumeAfterRestart(question string, resp ExtensionResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := s.prompt(ctx, resumeDialogNudge(question, resp)); err != nil {
		log.Printf("[session %s] resume for dialog: %v", s.ID, err)
	}
}

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

func formatQuestionnaire(answers []QuestionnaireAnswer) string {
	segments := make([]string, 0, len(answers))
	for _, qa := range answers {
		segments = append(segments, fmt.Sprintf("%q=%q", qa.Question, qa.Answer))
	}
	return strings.Join(segments, "; ")
}

func savedDialogTitle(raw string) string {
	var req extensionUIRequest
	if json.Unmarshal([]byte(raw), &req) != nil {
		return ""
	}
	return req.Title
}

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

func (s *Session) cancelIdle() {
	s.mu.Lock()
	s.idleGen++
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	s.mu.Unlock()
}

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

func (s *Session) accumulateWaitLocked(now time.Time) {
	if s.waitStartedAt.IsZero() {
		return
	}
	s.runWaitAccum += now.Sub(s.waitStartedAt)
	s.waitStartedAt = time.Time{}
}

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

type Usage struct {
	InputTokens  int64
	OutputTokens int64
	Cost         float64

	ActiveMs int64

	TokensPerSecond float64
	CostPerHour     float64

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

		}
	}
}

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
