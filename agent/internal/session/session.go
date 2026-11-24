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
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"github.com/sky-valley/pi/coding"

	"cherrypi.vladfaust.com/agent/internal/agency"
	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	"cherrypi.vladfaust.com/agent/internal/mcp"
	"cherrypi.vladfaust.com/agent/internal/store"
)

var ErrNoPendingDialog = errors.New("no pending extension dialog with that request id")

var ErrToolCallNotRunning = errors.New("tool call is not running")

type Config struct {
	WorkspaceRoot string

	FetchAttachment func(ctx context.Context, url string) (io.ReadCloser, error)

	AgencyURL string

	AgentToken string

	Agency *agency.Client

	AgencyCatalog AgencyCatalog

	Model *ai.Model

	ForcePlan bool

	Compaction *bool

	CompactionSettings *coding.CompactionSettings

	ResolveTaskID func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
	) (taskID string, err error)

	GetTaskSessionState func(
		ctx context.Context,
		taskID string,
	) (state TaskSessionState, found bool, err error)

	PutTaskSessionState func(
		ctx context.Context,
		taskID string,
		state TaskSessionState,
		expectedRevision int64,
	) (revision int64, err error)

	AppendTaskEvents func(
		ctx context.Context,
		taskID string,
		events []TaskEventRecord,
	) error

	FetchTaskHistory func(
		ctx context.Context,
		agencyProjectID string,
		agencySeq int64,
	) ([]SessionEntry, error)

	IdleTTL time.Duration

	Notify func(event string, payload map[string]any)

	TitleGenerator TitleGenerator

	TitleTimeout time.Duration

	RecoverRetryWindow time.Duration

	RecoverRetryDelay time.Duration

	ReportIntegrations func(
		ctx context.Context,
		taskID, revision string,
		report IntegrationReport,
	) error
}

type TitleGenerator func(ctx context.Context, prompt, provider, modelID, level string) (string, error)

type Session struct {
	cfg   Config
	store store.Store

	ID        string
	Workspace string

	SessionID string

	agencyTaskID string

	agencyProjectID string
	agencySeq       int64

	runStartMu sync.Mutex

	stateStoodDown bool

	checkpointMu sync.Mutex

	opMu sync.Mutex

	mu       sync.Mutex
	started  bool
	stopping bool
	running  bool

	stopRequested bool

	runStartedAt time.Time

	runWaitAccum  time.Duration
	waitStartedAt time.Time

	runToolDuration      time.Duration
	runModelWaitDuration time.Duration
	runTTFTDuration      time.Duration
	runFirstToken        bool

	requestStartedAt time.Time

	compactionStartedAt    time.Time
	compactionTokensBefore int64

	openTools map[string]time.Time

	toolMu         sync.Mutex
	toolCancels    map[string]context.CancelFunc
	toolInterrupts map[string]string

	stopActor string

	thinkingStartedAt time.Time
	thinkingChars     int
	model             string
	subs              map[chan Event]struct{}

	interactive bool

	appendSystemPrompt string

	integrationsFn func(ctx context.Context) AgencyIntegrations

	mcpConns []*mcp.Conn

	probe bool

	fixedTools []agent.AgentTool

	waiting bool

	queue []queueEntry

	pendingActivity *Event

	pendingExt map[string]extensionUIRequest

	lastDialogJSON string

	savedDialog string

	shutdownAbandoned bool

	idleTimer *time.Timer
	idleGen   uint64

	runHook func(started bool)

	eng           *engine
	unsubscribe   func()
	historyCancel context.CancelFunc

	catalogFn func(ctx context.Context) *modelCatalog

	refreshFn func(ctx context.Context) *modelCatalog

	variables []AgencyVariable

	restoredProjection    coding.BranchProjection
	hasRestoredProjection bool
	historyRaw            []string
	pushed                int

	stateRevision int64
	eventSeq      int64

	transcriptMu sync.Mutex

	flushMu sync.Mutex

	eventsMu      sync.Mutex
	pendingEvents []TaskEventRecord

	eventRetryAt time.Time
	eventBackoff time.Duration

	modelProvider string
	thinkingLevel string

	progress             Progress
	progressTouchedRun   bool
	progressSettleNudges int

	progressBlockID         int
	progressBlockCovered    bool
	progressBlockMessage    *ai.AssistantMessage
	progressBlockBreak      bool
	progressForceToolChoice bool

	forcePlanAllowed bool

	requiredTools        []string
	requiredSatisfied    bool
	requiredGateBlocks   int
	requiredSettleNudges int
	requiredBatch        *ai.AssistantMessage
	requiredBatchBlocked bool

	dialogs map[string]chan ExtensionResponse

	dialogTimeout time.Duration

	toolTimeout time.Duration

	liveInput      int64
	liveOutput     int64
	liveCacheRead  int64
	liveCacheWrite int64
	liveCost       float64

	recordedInput      int64
	recordedOutput     int64
	recordedCacheRead  int64
	recordedCacheWrite int64
	recordedCost       float64

	pluginToolDefs []PluginTool
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

func newSession(
	cfg Config,
	st store.Store,
	id, workspace, sessionID string,
	interactive bool,
) *Session {
	return &Session{
		cfg:            cfg,
		store:          st,
		ID:             id,
		Workspace:      workspace,
		SessionID:      sessionID,
		interactive:    interactive,
		pendingExt:     map[string]extensionUIRequest{},
		openTools:      map[string]time.Time{},
		toolCancels:    map[string]context.CancelFunc{},
		toolInterrupts: map[string]string{},
		subs:           map[chan Event]struct{}{},
		toolTimeout:    maxToolCallDuration,
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
	if s.abandonedForShutdown() {
		return ErrShuttingDown
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.cancelIdle()

	s.discardWaitingForNewRun(ctx)

	s.refreshVariables(ctx)

	if err := s.selectModel(ctx, provider, modelID, level); err != nil {
		return err
	}

	if err := s.ensureEngineLocked(ctx); err != nil {
		return err
	}

	return s.promptEngine(ctx, text, images)
}

func (s *Session) selectModel(
	ctx context.Context,
	provider, modelID, level string,
) error {
	if provider != "" && modelID == "" {
		return fmt.Errorf(
			"model %s has no id: give it one in the agency's model catalog",
			provider,
		)
	}
	if provider == "" && modelID != "" {
		return ErrInvalidModel
	}
	var model *ai.Model
	var catalog *modelCatalog
	if provider != "" {
		catalog = s.catalog(ctx)
		resolved, err := resolveModel(catalog, provider, modelID)
		if err != nil {
			return err
		}
		model = resolved
	}

	s.mu.Lock()
	if model != nil {
		s.modelProvider = provider
		s.model = modelID
		s.forcePlanAllowed = catalog.forcesPlan(model)
	}
	if level != "" {
		s.thinkingLevel = level
	}
	eng := s.eng
	s.mu.Unlock()

	if eng != nil {
		if model != nil {
			eng.sess.SetModel(model)
			if settings := s.compactionSettings(catalog, model); settings != nil {
				eng.compaction = settings
				eng.sess.EnableCompaction(*settings)
			}
		}
		if level != "" {
			eng.sess.SetThinkingLevel(agent.ThinkingLevel(level))
		}
	}
	return nil
}

func (s *Session) resumeSelection(ctx context.Context, model ModelSelection) error {
	s.mu.Lock()
	selected := s.modelProvider != "" || s.model != ""
	s.mu.Unlock()
	if selected {
		return nil
	}
	return s.selectModel(ctx, model.Provider, model.ModelID, model.ThinkingLevel)
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

func (s *Session) ensureAgencyTaskID(ctx context.Context) string {
	s.mu.Lock()
	id := s.agencyTaskID
	projectID, seq := s.agencyProjectID, s.agencySeq
	resolve := s.cfg.ResolveTaskID
	s.mu.Unlock()
	if id != "" || resolve == nil || projectID == "" || seq <= 0 {
		return id
	}
	resolved, err := resolve(ctx, projectID, seq)
	if err != nil || resolved == "" {
		if err != nil {
			slog.Debug("resolve task id", "session_id", s.ID, "error", err)
		}
		return id
	}
	s.mu.Lock()
	s.agencyTaskID = resolved
	s.mu.Unlock()
	return resolved
}

func (s *Session) catalog(ctx context.Context) *modelCatalog {
	if s.catalogFn == nil {
		return buildCatalog(s.cfg, AgencyModels{}, s.agencyTaskID)
	}
	return s.catalogFn(ctx)
}

func (s *Session) setRunning(v bool) {
	s.mu.Lock()
	changed := s.running != v
	s.running = v
	if changed {
		if v {
			s.stopRequested = false
			s.runStartedAt = time.Now()
			s.runWaitAccum = 0
			s.waitStartedAt = time.Time{}
			s.runToolDuration = 0
			s.runModelWaitDuration = 0
			s.runTTFTDuration = 0
			s.runFirstToken = false
			s.requestStartedAt = time.Time{}
			s.compactionStartedAt = time.Time{}
			s.openTools = map[string]time.Time{}
			s.toolMu.Lock()
			s.stopActor = ""
			s.toolCancels = map[string]context.CancelFunc{}
			s.toolInterrupts = map[string]string{}
			s.toolMu.Unlock()
			s.thinkingStartedAt = time.Time{}
			s.thinkingChars = 0
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
	return s.ensureEngineLocked(ctx)
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

func (s *Session) Config(ctx context.Context) (SessionConfig, error) {
	s.adoptStoredSelection(ctx)
	s.mu.Lock()
	provider, modelID, level := s.modelProvider, s.model, s.thinkingLevel
	s.mu.Unlock()
	return s.configFromRegistry(ctx, provider, modelID, level), nil
}

func (s *Session) SetModel(ctx context.Context, provider, modelID string) (SessionConfig, error) {
	if provider == "" || modelID == "" {
		return SessionConfig{}, ErrInvalidModel
	}
	catalog := s.catalog(ctx)
	model, err := resolveModel(catalog, provider, modelID)
	if err != nil {
		return SessionConfig{}, err
	}
	s.mu.Lock()
	s.modelProvider = provider
	s.model = modelID
	s.forcePlanAllowed = catalog.forcesPlan(model)
	eng := s.eng
	level := s.thinkingLevel
	s.mu.Unlock()
	if eng != nil {
		eng.sess.SetModel(model)
		if settings := s.compactionSettings(catalog, model); settings != nil {
			eng.compaction = settings
			eng.sess.EnableCompaction(*settings)
		}
	}
	return s.configFromRegistry(ctx, provider, modelID, level), nil
}

func (s *Session) SetThinkingLevel(ctx context.Context, level string) (SessionConfig, error) {
	if level == "" {
		return SessionConfig{}, ErrInvalidThinkingLevel
	}
	s.mu.Lock()
	s.thinkingLevel = level
	provider, modelID := s.modelProvider, s.model
	eng := s.eng
	s.mu.Unlock()
	if eng != nil {
		eng.sess.SetThinkingLevel(agent.ThinkingLevel(level))
	}
	return s.configFromRegistry(ctx, provider, modelID, level), nil
}

func (s *Session) settle(errMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s.opMu.Lock()
	defer s.opMu.Unlock()

	if errMsg == "" {
		if err := s.store.ClearSessionPendingPrompt(ctx, sqlitedb.ClearSessionPendingPromptParams{
			UpdatedAt: time.Now().UnixMilli(),
			ID:        s.ID,
		}); err != nil {
			log.Printf("[session %s] clear pending prompt: %v", s.ID, err)
		}
	}

	s.setRunning(false)

	if s.abandonedForShutdown() {
		return
	}

	s.clearWaitingStore(ctx)
	if errMsg != "" {
		slog.Error("run failed", "session_id", s.ID, "task_id", s.agencyTaskID, "error", errMsg)
		s.markFailed(ctx, errMsg)
		s.scheduleIdleReap()
		return
	}

	slog.Info("run settled", "session_id", s.ID, "task_id", s.agencyTaskID)
	s.notifyAgentState("run_settled", StatusDone)
	s.scheduleIdleReap()
}

const stopSettleTimeout = 10 * time.Second

func (s *Session) Stop(ctx context.Context, actor string) error {
	s.mu.Lock()
	eng := s.eng
	running := s.running
	s.stopRequested = true
	s.mu.Unlock()
	if actor = strings.TrimSpace(actor); actor == "" {
		actor = "an operator"
	}
	s.toolMu.Lock()
	s.stopActor = actor
	s.toolMu.Unlock()
	if !running || eng == nil {
		_, err := s.foldInterruptedRun(ctx)
		return err
	}

	s.holdQueue()

	eng.sess.Abort()
	if s.waitSettled(ctx, stopSettleTimeout) {
		return nil
	}
	s.finishStoppedRun(ctx)
	return nil
}

func (s *Session) InterruptToolCall(ctx context.Context, callID, actor string) error {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ErrToolCallNotRunning
	}
	if actor = strings.TrimSpace(actor); actor == "" {
		actor = "an operator"
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		s.toolMu.Lock()
		cancel, ok := s.toolCancels[callID]
		if ok {
			s.toolInterrupts[callID] = actor
		}
		s.toolMu.Unlock()
		if ok {
			cancel()
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return ErrToolCallNotRunning
		}
		select {
		case <-ctx.Done():
			return ErrToolCallNotRunning
		case <-time.After(10 * time.Millisecond):
		}
	}
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

func (s *Session) foldInterruptedRun(ctx context.Context) (bool, error) {
	s.mu.Lock()
	eng := s.eng
	s.mu.Unlock()
	if eng != nil {
		eng.sess.Agent.ClearAllQueues()
	}

	row, err := s.store.GetSession(ctx, s.ID)
	if err != nil {
		return false, err
	}
	if !row.PendingPrompt.Valid || row.PendingPrompt.String == "" {
		return false, nil
	}
	slog.Info("folding interrupted run into done", "session_id", s.ID)
	s.finishStoppedRun(ctx)
	return true, nil
}

func (s *Session) finishStoppedRun(ctx context.Context) {
	if err := s.store.ClearSessionPendingPrompt(ctx, sqlitedb.ClearSessionPendingPromptParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        s.ID,
	}); err != nil {
		log.Printf("[session %s] clear pending prompt: %v", s.ID, err)
	}

	s.recordUsage(ctx)
	s.setRunning(false)
	s.clearWaiting()
	s.clearWaitingStore(ctx)
	s.emit(statusEvent(StatusDone, ""))
	s.notifyAgentState("run_stopped", StatusDone)

	s.scheduleIdleReap()
}

func (s *Session) RemoveQueued(ctx context.Context, id string) (bool, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	return s.dropQueued(id), nil
}

func (s *Session) ClearQueue(ctx context.Context) error {
	s.clearQueued()
	return nil
}

func (s *Session) Restart() error {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		return ErrSessionBusy
	}
	s.stopEngine()
	return nil
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

func (s *Session) markFailed(ctx context.Context, msg string) {
	if s.probe {
		return
	}

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
	s.notifyAgentFailure(msg)
}

func (s *Session) fail(msg string) {
	if s.probe {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.setRunning(false)
	s.clearWaiting()
	slog.Error("run failed", "session_id", s.ID, "task_id", s.agencyTaskID, "error", msg)
	s.markFailed(ctx, msg)
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

func (s *Session) historyHasUserMessage() bool {
	for _, h := range s.readHistory() {
		if h.event.Kind == EventUser {
			return true
		}
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

func (s *Session) respondExtensionUI(
	ctx context.Context,
	requestID string,
	resp ExtensionResponse,
) error {

	if s.abandonedForShutdown() {
		return ErrShuttingDown
	}
	if s.answerLocalDialog(requestID, resp) {
		s.clearWaitingStore(ctx)
		s.notifyAgentState("waiting_resolved", StatusInProgress)
		return nil
	}

	s.mu.Lock()
	saved := s.savedDialog
	question := savedDialogTitle(saved)
	if saved != "" {
		s.savedDialog = ""
		s.waiting = false
		s.accumulateWaitLocked(time.Now())
	}
	s.mu.Unlock()
	if saved == "" {
		return ErrNoPendingDialog
	}
	s.clearWaitingStore(ctx)
	answer := resp.Value
	if resp.Kind == ExtensionResponseCancelled {
		answer = "The user cancelled the questionnaire."
	}
	s.resumeWithDialogAnswer(question, answer)
	return nil
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
	if payload == nil {
		payload = map[string]any{}
	}
	payload["session_id"] = s.ID

	switch event {
	case "task_progress":
		s.enqueueEvent(DomainEventProto("plan", payload), "plan")
	case "task_artifact":
		s.enqueueEvent(DomainEventProto("artifact", payload), "artifact")
	case "task_link":
		s.enqueueEvent(DomainEventProto("link", payload), "link")
	}
	if s.cfg.Notify == nil {
		return
	}
	s.cfg.Notify(event, payload)
}

func (s *Session) notifyAgentState(state string, status Status) {
	s.notify("agent_state_changed", map[string]any{"state": state, "status": int64(status)})
}

func (s *Session) notifyAgentFailure(errMsg string) {
	s.notify("agent_state_changed", map[string]any{
		"state":  "failed",
		"status": int64(StatusFailed),
		"error":  errMsg,
	})
}

func (s *Session) busy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running || s.waiting || len(s.dialogs) > 0
}

func (s *Session) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
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
	stale := gen != s.idleGen || s.running || s.eng == nil
	s.mu.Unlock()
	if stale {
		return
	}

	s.stopEngine()
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
	s.savedDialog = ""
	s.accumulateWaitLocked(time.Now())
	s.mu.Unlock()
}

func (s *Session) abandonForShutdown() {
	s.mu.Lock()
	s.shutdownAbandoned = true
	s.mu.Unlock()
	s.setRunning(false)
}

func (s *Session) abandonedForShutdown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdownAbandoned
}

func (s *Session) discardWaitingForNewRun(ctx context.Context) {
	s.mu.Lock()
	discarded := !s.running &&
		(s.waiting || s.lastDialogJSON != "" || s.savedDialog != "")
	if !s.running {
		s.waiting = false
		s.pendingExt = map[string]extensionUIRequest{}
		s.lastDialogJSON = ""
		s.savedDialog = ""
		s.accumulateWaitLocked(time.Now())
	}
	s.mu.Unlock()

	if discarded {
		s.clearWaitingStore(ctx)
	}
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

func (s *Session) startRequest(now time.Time) {
	s.mu.Lock()
	if s.requestStartedAt.IsZero() {
		s.requestStartedAt = now
	}
	s.mu.Unlock()
}

func (s *Session) firstToken(now time.Time) ModelWaitEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.requestStartedAt.IsZero() {
		return ModelWaitEvent{}
	}
	waited := now.Sub(s.requestStartedAt)
	if waited > 0 {
		s.runModelWaitDuration += waited
	}
	s.requestStartedAt = time.Time{}
	event := ModelWaitEvent{
		Kind:       PendingModelRequest,
		Provider:   s.modelProvider,
		Model:      s.model,
		DurationMs: waited.Milliseconds(),
	}
	if s.runFirstToken {

		return event
	}
	s.runFirstToken = true
	if !s.runStartedAt.IsZero() {
		if ttft := now.Sub(s.runStartedAt); ttft > 0 {
			s.runTTFTDuration = ttft
		}
	}
	return event
}

func (s *Session) beginCompaction(now time.Time, reason string, beforeTokens int) {
	s.mu.Lock()
	s.requestStartedAt = time.Time{}
	if s.compactionStartedAt.IsZero() {
		s.compactionStartedAt = now
		s.compactionTokensBefore = int64(beforeTokens)
	}
	s.mu.Unlock()
	s.emitPending(&PendingActivityEvent{Kind: PendingCompaction, Reason: reason})
}

func (s *Session) finishCompaction(
	now time.Time,
	reason string,
	resumeTurn bool,
	afterTokens int,
) {
	s.mu.Lock()
	started := s.compactionStartedAt
	beforeTokens := s.compactionTokensBefore
	s.compactionStartedAt = time.Time{}
	s.compactionTokensBefore = 0
	if resumeTurn {
		s.requestStartedAt = now
	}
	waited := time.Duration(0)
	if !started.IsZero() {
		waited = now.Sub(started)
		if waited > 0 {
			s.runModelWaitDuration += waited
		}
	}
	provider, model := s.modelProvider, s.model
	s.mu.Unlock()
	if waited > 0 {
		s.emit(Event{Kind: EventModelWait, At: now, ModelWait: &ModelWaitEvent{
			Kind:                PendingCompaction,
			Provider:            provider,
			Model:               model,
			DurationMs:          waited.Milliseconds(),
			Reason:              reason,
			ContextTokensBefore: beforeTokens,
			ContextTokensAfter:  int64(afterTokens),
		}})
	}
	if resumeTurn {
		s.emitPending(&PendingActivityEvent{Kind: PendingModelRequest})
		return
	}
	s.emitPending(nil)
}

func (s *Session) startTool(callID string, now time.Time) {
	s.mu.Lock()
	if s.openTools == nil {
		s.openTools = map[string]time.Time{}
	}
	if _, ok := s.openTools[callID]; !ok {
		s.openTools[callID] = now
	}
	s.mu.Unlock()
}

func (s *Session) finishTool(callID string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	started, ok := s.openTools[callID]
	if !ok {
		return
	}
	delete(s.openTools, callID)
	if d := now.Sub(started); d > 0 {
		s.runToolDuration += d
	}
}

func (s *Session) closeRunTiming() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeRunTimingLocked(time.Now())
}

func (s *Session) closeRunTimingLocked(now time.Time) {
	if !s.compactionStartedAt.IsZero() {
		if wait := now.Sub(s.compactionStartedAt); wait > 0 {
			s.runModelWaitDuration += wait
		}
		s.compactionStartedAt = time.Time{}
	}
	if !s.requestStartedAt.IsZero() {
		if wait := now.Sub(s.requestStartedAt); wait > 0 {
			s.runModelWaitDuration += wait
		}
		s.requestStartedAt = time.Time{}
	}
	for callID, started := range s.openTools {
		if d := now.Sub(started); d > 0 {
			s.runToolDuration += d
		}
		delete(s.openTools, callID)
	}
}

func (s *Session) runThinkingLocked(now time.Time) time.Duration {
	tool := s.runToolDuration
	for _, started := range s.openTools {
		if d := now.Sub(started); d > 0 {
			tool += d
		}
	}
	wait := s.runModelWaitDuration
	for _, started := range [...]time.Time{s.requestStartedAt, s.compactionStartedAt} {
		if started.IsZero() {
			continue
		}
		if d := now.Sub(started); d > 0 {
			wait += d
		}
	}
	thinking := s.runActiveDurationLocked(now) - tool - wait
	if thinking > 0 {
		return thinking
	}
	return 0
}

func (s *Session) processRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eng != nil
}

func (s *Session) liveUsage(ctx context.Context) (Usage, bool) {
	s.mu.Lock()
	eng := s.eng

	input := s.liveInput - s.recordedInput
	output := s.liveOutput - s.recordedOutput
	cacheRead := s.liveCacheRead - s.recordedCacheRead
	cacheWrite := s.liveCacheWrite - s.recordedCacheWrite
	cost := s.liveCost - s.recordedCost
	running := s.running
	started := s.runStartedAt
	now := time.Now()

	var active, thinking, tool, modelWait, ttft time.Duration
	firstToken := false
	if running {
		active = s.runActiveDurationLocked(now)
		thinking = s.runThinkingLocked(now)
		tool = s.runToolDuration
		modelWait = s.runModelWaitDuration
		ttft = s.runTTFTDuration
		firstToken = s.runFirstToken
	}
	window := 0
	if eng != nil && eng.sess.Model != nil {
		window = eng.sess.Model.ContextWindow
	}
	s.mu.Unlock()
	if eng == nil {
		return Usage{}, false
	}

	out := Usage{
		InputTokens:      input,
		OutputTokens:     output,
		CacheReadTokens:  cacheRead,
		CacheWriteTokens: cacheWrite,
		Cost:             cost,
		ActiveMs:         active.Milliseconds(),
		ThinkingMs:       thinking.Milliseconds(),
		ToolMs:           tool.Milliseconds(),
		ModelWaitMs:      modelWait.Milliseconds(),
		TTFTMs:           ttft.Milliseconds(),
	}
	if firstToken {
		out.TTFTRuns = 1
	}
	if window > 0 {
		out.ContextWindow = int64(window)
		out.ContextTokens = int64(estimateContextTokens(eng))
	}
	if running && !started.IsZero() {
		if seconds := time.Since(started).Seconds(); seconds > 0 {
			out.TokensPerSecond = float64(input+output) / seconds
			out.CostPerHour = cost / seconds * 3600
		}
	}
	return out, true
}

type Usage struct {
	InputTokens  int64
	OutputTokens int64

	CacheReadTokens  int64
	CacheWriteTokens int64
	Cost             float64

	ActiveMs int64

	ThinkingMs  int64
	ModelWaitMs int64
	ToolMs      int64
	TTFTMs      int64
	TTFTRuns    int64

	TokensPerSecond float64
	CostPerHour     float64

	ContextTokens int64
	ContextWindow int64
}

func (s *Session) emit(ev Event) {
	if ev.At.IsZero() {
		ev.At = time.Now()
	}

	switch ev.Kind {
	case EventModelWait:
		s.queueTranscriptEvents()
		s.enqueueEvent(TaskEventProto(s.ID, ev), eventKind(ev))
	case EventStatus, EventRunStarted, EventRunSettled, EventThinkingSpan:
		s.enqueueEvent(TaskEventProto(s.ID, ev), eventKind(ev))
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

func (s *Session) emitPending(pending *PendingActivityEvent) {
	if pending != nil {
		s.mu.Lock()
		if pending.Model == "" {
			pending.Model = s.model
		}
		if pending.Provider == "" {
			pending.Provider = s.modelProvider
		}
		s.mu.Unlock()
	}

	ev := Event{Kind: EventPending, At: time.Now(), Pending: pending}

	s.mu.Lock()
	if pending == nil {
		s.pendingActivity = nil
	} else {
		stored := ev
		copy := *pending
		stored.Pending = &copy
		s.pendingActivity = &stored
	}
	s.mu.Unlock()

	s.emit(ev)
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

	queue := s.queueEventLocked()
	ch <- Event{Kind: EventQueue, At: time.Now(), Queue: &queue}

	if s.pendingActivity != nil {
		ev := *s.pendingActivity
		pending := *ev.Pending
		ev.Pending = &pending
		ch <- ev
	}
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

func (s *Session) waitIdle(ctx context.Context, timeout time.Duration) {
	s.mu.Lock()
	eng := s.eng
	s.mu.Unlock()
	if eng == nil {
		return
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if !eng.sess.Agent.State().IsStreaming {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
		}
	}
}
