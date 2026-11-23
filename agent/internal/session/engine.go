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
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	_ "github.com/sky-valley/pi/ai/providers"
	"github.com/sky-valley/pi/coding"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
)

type engine struct {
	sess *coding.Session

	compaction *coding.CompactionSettings

	manualCompaction atomic.Bool

	queuedCompaction atomic.Bool

	tokensMu      sync.Mutex
	tokens        int
	tokensAtRaw   int
	tokensAtStamp uint64

	projectionMu    sync.Mutex
	projected       []agent.AgentMessage
	projectedRawLen int
	projectionStamp uint64

	projectedTrusted int
}

type modelCatalog struct {
	models ai.MutableModels

	donors map[string]*ai.Model

	declared map[string]bool

	forcePlan map[string]bool

	compaction map[string]modelCompaction

	variables []AgencyVariable
}

type modelCompaction struct {
	reserveTokens    int
	keepRecentTokens int
}

func (c *modelCatalog) declareModel(
	provider, model string,
	forcePlan bool,
	reserveTokens, keepRecentTokens int,
) {
	key := provider + "/" + model
	if c.declared == nil {
		c.declared = map[string]bool{}
	}
	c.declared[key] = true
	if forcePlan {
		if c.forcePlan == nil {
			c.forcePlan = map[string]bool{}
		}
		c.forcePlan[key] = true
	}
	if reserveTokens > 0 || keepRecentTokens > 0 {
		if c.compaction == nil {
			c.compaction = map[string]modelCompaction{}
		}
		c.compaction[key] = modelCompaction{
			reserveTokens:    reserveTokens,
			keepRecentTokens: keepRecentTokens,
		}
	}
}

func (c *modelCatalog) compactionFor(model *ai.Model) (reserveTokens, keepRecentTokens int) {
	if c == nil || model == nil || c.compaction == nil {
		return 0, 0
	}
	settings := c.compaction[string(model.Provider)+"/"+model.ID]
	return settings.reserveTokens, settings.keepRecentTokens
}

func (c *modelCatalog) forcesPlan(model *ai.Model) bool {
	if c == nil || model == nil {
		return false
	}
	return c.forcePlan[string(model.Provider)+"/"+model.ID]
}

func (c *modelCatalog) offered(model *ai.Model) bool {
	if c == nil {
		return false
	}
	return c.declared[string(model.Provider)+"/"+model.ID]
}

func (c *modelCatalog) offeredModels() []*ai.Model {
	if c == nil {
		return nil
	}
	available := c.models.GetModels("")
	out := make([]*ai.Model, 0, len(available))
	for _, model := range available {
		if model.ID == "" || !c.offered(model) {
			continue
		}
		out = append(out, model)
	}
	return out
}

func (m *Manager) modelsRegistry(ctx context.Context, taskID string) *modelCatalog {
	now := time.Now()

	m.catalogMu.Lock()
	if m.catalogs == nil {
		m.catalogs = map[string]*catalogEntry{}
	}
	m.pruneCatalogsLocked(now)
	entry := m.catalogs[taskID]
	if entry != nil && !entry.needsRefresh(now) {
		entry.lastUsed = now
		built := entry.built
		m.catalogMu.Unlock()
		return built
	}
	if entry == nil {
		entry = &catalogEntry{}
		m.catalogs[taskID] = entry
	}
	revision := entry.revision
	fetch := m.cfg.AgencyCatalog != nil && !now.Before(entry.retryAt)
	m.catalogMu.Unlock()

	if fetch {
		fetchCtx, cancel := context.WithTimeout(ctx, catalogFetchTimeout)
		declared, changed, err := m.cfg.AgencyCatalog.Fetch(fetchCtx, taskID, revision)
		cancel()
		m.catalogMu.Lock()
		switch {
		case err != nil:

			slog.Warn("agency model catalog", "task_id", taskID, "error", err)
			entry.retryAt = time.Now().Add(catalogRetryInterval)
		case changed:
			entry.declared = declared
			entry.revision = declared.Revision
		}
		entry.fetchedAt = time.Now()
		m.catalogMu.Unlock()
	}

	m.catalogMu.Lock()
	declared := entry.declared
	m.catalogMu.Unlock()

	built := buildCatalog(m.cfg, declared, taskID)

	m.catalogMu.Lock()
	entry.built = built
	entry.lastUsed = time.Now()
	m.catalogMu.Unlock()
	return built
}

func (m *Manager) modelsRegistryForced(ctx context.Context, taskID string) *modelCatalog {
	m.catalogMu.Lock()
	if entry := m.catalogs[taskID]; entry != nil {
		entry.fetchedAt = time.Time{}
	}
	m.catalogMu.Unlock()
	return m.modelsRegistry(ctx, taskID)
}

func (m *Manager) pruneCatalogsLocked(now time.Time) {
	for taskID, entry := range m.catalogs {
		if entry.lastUsed.IsZero() {
			continue
		}
		if now.Sub(entry.lastUsed) > catalogIdleTTL {
			delete(m.catalogs, taskID)
		}
	}
}

func (e *catalogEntry) needsRefresh(now time.Time) bool {
	if e.built == nil {
		return true
	}
	if now.Sub(e.fetchedAt) >= catalogRefreshInterval {
		return true
	}
	return !e.retryAt.IsZero() && !now.Before(e.retryAt)
}

func buildCatalog(
	cfg Config,
	declared AgencyModels,
	taskID string,
) *modelCatalog {
	catalog := &modelCatalog{
		models:    ai.BuiltinModels(nil),
		variables: declared.Variables,
	}

	catalog.donors = builtinDonors(catalog.models)
	applyAgencyModels(catalog, declared, cfg.AgencyURL, cfg.AgentToken, taskID)
	return catalog
}

func resolveModel(
	catalog *modelCatalog,
	provider, modelID string,
) (*ai.Model, error) {
	if catalog == nil {
		return nil, fmt.Errorf("no models are available: declare one in the agency")
	}
	offered := catalog.offeredModels()
	if provider != "" || modelID != "" {
		for _, model := range offered {
			if string(model.Provider) == provider && model.ID == modelID {
				return model, nil
			}
		}
		if modelID == "" {
			return nil, fmt.Errorf(
				"model %s has no id: give it one in the agency's model catalog",
				provider,
			)
		}
		return nil, fmt.Errorf(
			"model %s/%s is not available on this agent: enable it and grant it to the task in the agency",
			provider,
			modelID,
		)
	}
	for _, model := range offered {
		if string(model.Provider) == "anthropic" {
			return model, nil
		}
	}
	if len(offered) == 0 {
		return nil, fmt.Errorf("no models are available: declare one in the agency")
	}
	return offered[0], nil
}

func (s *Session) ensureEngineLocked(ctx context.Context) error {
	s.mu.Lock()
	if s.eng != nil {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if !s.probe {
		s.restoreState(ctx)
	}

	catalog := s.catalog(ctx)
	var model *ai.Model
	if s.cfg.Model != nil {

		model = s.cfg.Model
	} else {
		var err error
		model, err = resolveModel(catalog, s.modelProvider, s.model)
		if err != nil {
			return err
		}
	}

	forcePlan := catalog.forcesPlan(model)
	if s.cfg.Model != nil {
		forcePlan = s.cfg.ForcePlan
	}
	s.mu.Lock()
	s.modelProvider = string(model.Provider)
	s.model = model.ID
	s.forcePlanAllowed = forcePlan
	s.mu.Unlock()
	level := agent.ThinkingLevel(s.thinkingLevel)
	if level == "" {
		level = coding.DefaultThinkingLevel
	}

	skills, skillDiags := hostSkills(s.Workspace)
	for _, diag := range skillDiags {
		slog.Warn("host skill",
			"session_id", s.ID,
			"type", diag.Type,
			"path", diag.Path,
			"message", diag.Message,
		)
	}
	if manual := manualSkillNames(skills); len(manual) > 0 {
		slog.Debug("host skills withheld from the model",
			"session_id", s.ID,
			"skills", manual,
		)
	}

	integrations := s.integrations(ctx)
	mcpTools, skillPrompt, integrationReport := s.startIntegrations(ctx, integrations)
	if len(mcpTools) > 0 {
		slog.Info("mcp servers connected",
			"session_id", s.ID,
			"tools", len(mcpTools),
		)
	}

	tools := append(s.tools(), mcpTools...)

	systemPrompt := appendPromptBlock(s.appendSystemPrompt, skillPrompt)
	memoryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	if snapshot, err := s.memoryContext(memoryCtx); err == nil && snapshot != "" {
		systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + snapshot)
	}
	cancel()

	if listing := s.variablesPrompt(); listing != "" {
		systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + listing)
	}
	opts := coding.SessionOptions{
		Model:         model,
		Cwd:           s.Workspace,
		ThinkingLevel: level,
		SessionID:     s.ID,
		SystemPrompt:  systemPrompt,
		CustomTools:   tools,

		MaxRetries:     3,
		BeforeToolCall: s.beforeToolCall,
		OnPayload:      s.forcePlanToolChoice,
	}
	if s.cfg.Model == nil {

		opts.Models = catalog.models
	}
	compaction := s.compactionSettings(catalog, model)
	if compaction != nil {
		opts.Compaction = compaction
	}

	sess, err := coding.NewSession(opts)
	if err != nil {
		return fmt.Errorf("create pi session: %w", err)
	}

	sess.Agent.SetTools(strictTools(s.wrapTools(s.withVariableShell(sess.Agent.State().Tools))))

	eng := &engine{sess: sess, compaction: compaction}
	enableAllSteering(sess.Agent)

	if s.hasRestoredProjection {
		sess.LoadBranch(s.restoredProjection)
		if hasCompaction(s.restoredProjection) {

			eng.projected = append([]agent.AgentMessage(nil), s.restoredProjection.Messages...)
			eng.projectedRawLen = len(s.restoredProjection.Messages)
			eng.projectedTrusted = projectionTrustedFrom(s.restoredProjection)
		}
	}

	previousTransform := sess.Agent.TransformContext
	sess.Agent.TransformContext = func(
		ctx context.Context,
		messages []agent.AgentMessage,
	) []agent.AgentMessage {
		view, _, trustedFrom := eng.modelView()
		manual := eng.manualCompaction.Load()
		queued := eng.queuedCompaction.Swap(false) && eng.compaction != nil
		reason := "threshold"
		if manual || queued {
			reason = "manual"
		}
		if queued {

			restoreProbe := eng.probeSummaries(&compactionProbe{})
			defer restoreProbe()
			forced := *eng.compaction
			forced.ReserveTokens = max(forced.ReserveTokens, eng.sess.Model.ContextWindow)
			eng.sess.EnableCompaction(forced)
			defer eng.sess.EnableCompaction(*eng.compaction)
		}
		compacting := manual || queued || eng.needsCompaction(view, trustedFrom)
		beforeTokens := estimateViewTokens(view, trustedFrom)
		if compacting {
			s.beginCompaction(time.Now(), reason, beforeTokens)
		}
		transformed := messages
		if previousTransform != nil {
			transformed = previousTransform(ctx, messages)
		}
		eng.observeProjection(messages, transformed)
		if compacting {

			s.finishCompaction(time.Now(), reason, !manual, estimateContextTokens(eng))
		}
		return transformed
	}

	sess.Agent.AfterToolCall = s.limitToolImages

	unsubscribe := sess.Subscribe(s.handleAgentEvent)
	syncCtx, stopSync := context.WithCancel(context.Background())

	s.mu.Lock()
	s.eng = eng
	s.historyCancel = stopSync
	s.unsubscribe = unsubscribe
	s.started = true
	s.stopping = false
	s.mu.Unlock()

	go s.syncLoop(syncCtx)
	s.reportIntegrations(ctx, integrationReport, integrations.Revision)

	slog.Info("pi session started",
		"session_id", s.ID,
		"task_id", s.agencyTaskID,
		"provider", model.Provider,
		"model", model.ID,
		"skills", promptSkillNames(skills),
	)

	if !s.probe {
		now := time.Now().UnixMilli()
		if err := s.store.SetSessionPid(ctx, sqlitedb.SetSessionPidParams{
			Pid:       sql.NullInt64{},
			UpdatedAt: now,
			ID:        s.ID,
		}); err != nil {
			slog.Debug("record pid", "session_id", s.ID, "error", err)
		}
	}
	return nil
}

func (s *Session) stopEngine() {

	s.closeThinking("")

	s.closeIntegrations()

	if s.stateEnabled() || s.eventsEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		s.queueTranscriptEvents()
		if err := s.flushEvents(ctx); err == nil {
			s.checkpointState(ctx)
		}
		cancel()
	}

	s.mu.Lock()
	eng := s.eng
	unsub := s.unsubscribe
	stopHistory := s.historyCancel
	s.eng = nil
	s.unsubscribe = nil
	s.historyCancel = nil
	s.mu.Unlock()
	if stopHistory != nil {
		stopHistory()
	}
	if unsub != nil {
		unsub()
	}
	if eng != nil {
		eng.sess.Agent.ClearAllQueues()
		eng.sess.Abort()
	}

}

func (s *Session) handleAgentEvent(_ context.Context, event agent.AgentEvent) error {
	switch event.Type {
	case agent.EvAgentStart:
		s.setRunning(true)
		s.beginPlanActivity()
		s.emit(Event{Kind: EventRunStarted, At: time.Now()})
		s.notifyAgentState("run_started", StatusInProgress)

	case agent.EvTurnStart:

		now := time.Now()
		s.startRequest(now)
		s.emitPending(&PendingActivityEvent{Kind: PendingModelRequest})

	case agent.EvMessageUpdate:
		if event.AssistantMessageEvent == nil {
			return nil
		}

		if wait := s.firstToken(time.Now()); wait.DurationMs > 0 {
			s.emit(Event{Kind: EventModelWait, At: time.Now(), ModelWait: &wait})
		}
		s.emitPending(nil)
		switch event.AssistantMessageEvent.Type {
		case ai.EventTextDelta:
			s.emit(Event{Kind: EventText, At: time.Now(), Delta: event.AssistantMessageEvent.Delta})
		case ai.EventThinkingStart:
			s.startThinking(time.Now())
		case ai.EventThinkingDelta:
			s.countThinking(event.AssistantMessageEvent.Delta)
			s.emit(
				Event{
					Kind:  EventThinking,
					At:    time.Now(),
					Delta: event.AssistantMessageEvent.Delta,
				},
			)
		case ai.EventThinkingEnd:

			s.closeThinking(event.AssistantMessageEvent.Content)
		}

	case agent.EvMessageStart:
		if text := userMessageText(event.Message); text != "" {

			s.consumeQueued(text)
			s.noteUserOutput()
			s.emit(Event{Kind: EventUser, At: time.Now(), Text: text})
		}

	case agent.EvMessageEnd:
		if assistant, ok := event.Message.(*ai.AssistantMessage); ok {
			s.recordRunUsage(assistant.Usage)
			s.noteAssistantOutput(assistant)
		}

	case agent.EvToolExecutionStart:
		s.emitPending(nil)
		s.startTool(event.ToolCallID, time.Now())
		s.emit(toolEvent(event, ToolStarted))
	case agent.EvToolExecutionUpdate:
		s.emit(toolEvent(event, ToolUpdated))
	case agent.EvToolExecutionEnd:
		s.finishTool(event.ToolCallID, time.Now())
		s.emit(toolEvent(event, ToolFinished))

	case agent.EvAgentEnd:
		s.emitPending(nil)

		s.closeThinking("")
		go s.finishRun(event)
	}
	return nil
}

func (s *Session) startThinking(now time.Time) {
	s.mu.Lock()
	if s.thinkingStartedAt.IsZero() {
		s.thinkingStartedAt = now
		s.thinkingChars = 0
	}
	s.mu.Unlock()
}

func (s *Session) countThinking(delta string) {
	s.mu.Lock()
	if s.thinkingStartedAt.IsZero() {
		s.thinkingStartedAt = time.Now()
	}
	s.thinkingChars += utf8.RuneCountInString(delta)
	s.mu.Unlock()
}

func (s *Session) closeThinking(content string) {
	now := time.Now()
	s.mu.Lock()
	started, chars := s.thinkingStartedAt, s.thinkingChars
	s.thinkingStartedAt = time.Time{}
	s.thinkingChars = 0
	s.mu.Unlock()
	if started.IsZero() {
		return
	}
	if content != "" {
		chars = utf8.RuneCountInString(content)
	}
	durationMs := now.Sub(started).Milliseconds()
	if chars <= 0 || durationMs <= 0 {
		return
	}
	s.emit(Event{Kind: EventThinkingSpan, At: now, ThinkingSpan: &ThinkingSpanEvent{
		DurationMs: durationMs,
		Chars:      int32(chars),
	}})
}

func toolEvent(event agent.AgentEvent, phase ToolPhase) Event {

	args := ""
	if phase != ToolFinished && event.Args != nil {
		if raw, err := json.Marshal(event.Args); err == nil {
			args = string(raw)
		}
	}
	result := ""
	details := ""
	if phase == ToolFinished {
		result, details = toolResultJSON(event.Result)
	} else if phase == ToolUpdated && event.PartialResult != nil {
		result, _ = toolResultJSON(event.PartialResult)
	}
	return Event{
		Kind: EventTool,
		At:   time.Now(),
		Tool: &ToolEvent{
			CallID:      event.ToolCallID,
			Name:        event.ToolName,
			Phase:       phase,
			ArgsJSON:    args,
			ResultJSON:  result,
			DetailsJSON: details,
			IsError:     event.IsError,
		},
	}
}

func toolResultJSON(result any) (text, details string) {
	switch value := result.(type) {
	case agent.AgentToolResult:
		return contentListText(value.Content), detailsJSON(value.Details)
	case *agent.AgentToolResult:
		if value != nil {
			return contentListText(value.Content), detailsJSON(value.Details)
		}
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return "", ""
	}
	return string(raw), resultDetails(raw)
}

func detailsJSON(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return ""
	}
	return string(raw)
}

func (s *Session) finishRun(event agent.AgentEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.mu.Lock()
	eng := s.eng
	stopped := s.stopRequested
	abandoned := s.shutdownAbandoned
	s.mu.Unlock()

	errMsg := ""
	if eng != nil && !stopped && !abandoned {
		if state := eng.sess.Agent.State(); state.ErrorMessage != "" {
			errMsg = state.ErrorMessage
		}
	}

	s.queueTranscriptEvents()
	if err := s.flushEvents(ctx); err == nil {
		s.checkpointState(ctx)
	}

	s.closeRunTiming()
	s.recordUsage(ctx)

	if errMsg == "" && !stopped && !abandoned && eng != nil {
		reminders := make([]agent.AgentMessage, 0, 2)
		if reminder, cont := s.shouldContinueWithoutRequiredTool(); cont {
			reminders = append(reminders, reminder)
		}
		if reminder, cont := s.shouldContinueBeforeSettle(); cont {
			reminders = append(reminders, reminder)
		}
		if len(reminders) > 0 {

			eng.sess.Agent.WaitForIdle()
			s.runStartMu.Lock()
			s.setRunning(false)
			s.setRunning(true)
			err := s.startRun(ctx, eng, reminders)
			s.runStartMu.Unlock()
			if err == nil {

				return
			}

			slog.Warn("settle continuation failed", "session_id", s.ID, "error", err)
			s.setRunning(false)
		}
	}

	if errMsg == "" && !abandoned {
		s.emit(Event{Kind: EventRunSettled, At: time.Now()})
	}
	s.settle(errMsg)
}

func (s *Session) runModel(
	ctx context.Context,
	provider, modelID string,
) (*ai.Model, error) {
	if s.cfg.Model != nil {
		return s.cfg.Model, nil
	}
	return resolveModel(s.catalog(ctx), provider, modelID)
}

// line the numbers behind a refusal are not in any log.
func (s *Session) logRunLimits(ctx context.Context) {
	s.mu.Lock()
	provider, modelID, level := s.modelProvider, s.model, s.thinkingLevel
	s.mu.Unlock()
	model, err := s.runModel(ctx, provider, modelID)
	if err != nil || model == nil {
		return
	}
	if level == "" {
		level = string(coding.DefaultThinkingLevel)
	}
	slog.Info("run model limits",
		"session_id", s.ID,
		"task_id", s.agencyTaskID,
		"provider", string(model.Provider),
		"model", model.ID,
		"thinking_level",
		string(ai.ClampThinkingLevel(model, ai.ModelThinkingLevel(level))),
		"context_window", model.ContextWindow,
		"max_tokens", model.MaxTokens,
	)
}

func (s *Session) recordRunUsage(usage ai.Usage) {
	s.mu.Lock()
	s.liveInput += int64(usage.Input)
	s.liveOutput += int64(usage.Output)
	s.liveCacheRead += int64(usage.CacheRead)
	s.liveCacheWrite += int64(usage.CacheWrite)
	s.liveCost += usage.Cost.Total
	s.mu.Unlock()
}

func (s *Session) recordUsage(ctx context.Context) {
	s.mu.Lock()
	provider := s.modelProvider
	model := s.model
	input, output, cost := s.liveInput, s.liveOutput, s.liveCost
	cacheRead, cacheWrite := s.liveCacheRead, s.liveCacheWrite
	recordedIn, recordedOut, recordedCost := s.recordedInput, s.recordedOutput, s.recordedCost
	recordedCacheRead, recordedCacheWrite := s.recordedCacheRead, s.recordedCacheWrite
	activeMs := s.runActiveDurationLocked(time.Now()).Milliseconds()
	toolMs := s.runToolDuration.Milliseconds()
	modelWaitMs := s.runModelWaitDuration.Milliseconds()
	ttftMs := s.runTTFTDuration.Milliseconds()
	s.mu.Unlock()

	deltaIn := max(input-recordedIn, 0)
	deltaOut := max(output-recordedOut, 0)
	deltaCacheRead := max(cacheRead-recordedCacheRead, 0)
	deltaCacheWrite := max(cacheWrite-recordedCacheWrite, 0)
	deltaCost := max(cost-recordedCost, 0)
	if deltaIn == 0 && deltaOut == 0 && deltaCost == 0 && activeMs <= 0 {
		return
	}

	at := time.Now().UnixMilli()
	if err := s.store.InsertUsage(ctx, sqlitedb.InsertUsageParams{
		SessionID:        s.ID,
		Provider:         provider,
		Model:            model,
		InputTokens:      deltaIn,
		OutputTokens:     deltaOut,
		CacheReadTokens:  deltaCacheRead,
		CacheWriteTokens: deltaCacheWrite,
		Cost:             deltaCost,
		ActiveMs:         activeMs,
		ToolMs:           toolMs,
		ModelWaitMs:      modelWaitMs,
		TtftMs:           ttftMs,
		At:               at,
	}); err != nil {
		slog.Warn("insert usage", "session_id", s.ID, "error", err)
		return
	}

	s.notify("usage", map[string]any{
		"provider":           provider,
		"model":              model,
		"input_tokens":       deltaIn,
		"output_tokens":      deltaOut,
		"cache_read_tokens":  deltaCacheRead,
		"cache_write_tokens": deltaCacheWrite,
		"cost":               deltaCost,
		"active_ms":          activeMs,
		"tool_ms":            toolMs,
		"model_wait_ms":      modelWaitMs,
		"ttft_ms":            ttftMs,
		"at":                 at,
	})

	s.mu.Lock()
	s.recordedInput, s.recordedOutput, s.recordedCost = input, output, cost
	s.recordedCacheRead, s.recordedCacheWrite = cacheRead, cacheWrite
	s.mu.Unlock()
}

func (s *Session) promptEngine(ctx context.Context, text string, images []imageAttachment) error {
	s.mu.Lock()
	eng := s.eng
	s.mu.Unlock()
	if eng == nil {
		return fmt.Errorf("session is not running")
	}

	s.runStartMu.Lock()
	defer s.runStartMu.Unlock()

	if eng.sess.Agent.State().IsStreaming {

		return s.steerQueued(ctx, QueuedPrompt{Text: text}, images)
	}

	held := s.takeQueue()
	opening := QueuedPrompt{Text: text}
	openingImages := images
	var following []queuedDelivery
	if len(held) > 0 {

		opening = held[0].entry
		openingImages = reloadImages(held[0].images)
		following = append(following, held[1:]...)
		following = append(following, queuedDelivery{
			entry:  QueuedPrompt{Text: text},
			images: images,
		})
	}

	now := time.Now().UnixMilli()
	if err := s.store.SetSessionPendingPrompt(ctx, sqlitedb.SetSessionPendingPromptParams{
		PendingPrompt: sql.NullString{String: opening.Text, Valid: true},
		UpdatedAt:     now,
		ID:            s.ID,
	}); err != nil {

		s.restoreHeld(held)
		return err
	}

	message := ai.UserMessage{
		Content:   userContent(opening.Text, openingImages),
		Timestamp: now,
	}
	s.setRunning(true)
	s.logRunLimits(ctx)

	prompts := append(s.beginPlanRun(), message)
	if err := s.startRun(ctx, eng, prompts); err != nil {
		s.restoreHeld(held)

		s.abandonRun(ctx, err)
		return err
	}

	slog.Info("prompt sent",
		"session_id", s.ID,
		"task_id", s.agencyTaskID,
		"images", len(openingImages),
		"held", len(following),
	)

	for i, delivery := range following {
		if err := s.steerQueued(ctx, delivery.entry, delivery.images); err != nil {
			s.restoreHeld(following[i:])
			return err
		}
	}
	return nil
}

func (s *Session) abandonRun(ctx context.Context, cause error) {
	slog.Warn("run start failed", "session_id", s.ID, "error", cause)
	if err := s.store.ClearSessionPendingPrompt(
		ctx,
		sqlitedb.ClearSessionPendingPromptParams{
			UpdatedAt: time.Now().UnixMilli(),
			ID:        s.ID,
		},
	); err != nil {
		slog.Warn("clear pending prompt", "session_id", s.ID, "error", err)
	}
	s.setRunning(false)
	s.emit(Event{Kind: EventRunSettled, At: time.Now()})
	s.notifyAgentState("run_settled", StatusDone)
}

const runStartTimeout = 10 * time.Second

func (s *Session) startRun(
	ctx context.Context,
	eng *engine,
	prompts []agent.AgentMessage,
) error {
	started := make(chan struct{}, 1)
	unsub := eng.sess.Agent.Subscribe(func(_ context.Context, event agent.AgentEvent) error {
		if event.Type == agent.EvAgentStart {
			select {
			case started <- struct{}{}:
			default:
			}
		}
		return nil
	})
	done := make(chan error, 1)
	go func() {
		defer unsub()
		_, err := eng.sess.RunMessages(context.Background(), prompts)
		done <- err
	}()

	timer := time.NewTimer(runStartTimeout)
	defer timer.Stop()
	select {
	case <-started:
		return nil
	case err := <-done:

		if err == nil {
			return nil
		}
		return err
	case <-ctx.Done():

		select {
		case <-started:
			return nil
		default:
		}
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("the runtime did not start the run")
	}
}

func userContent(text string, images []imageAttachment) ai.ContentList {
	content := ai.ContentList{ai.TextContent{Text: text}}
	for _, image := range images {
		content = append(content, ai.ImageContent{
			Data:     encodeBase64(image.data),
			MimeType: image.mimeType,
		})
	}
	return content
}

func userMessageText(message ai.Message) string {
	var content ai.ContentList
	switch user := message.(type) {
	case *ai.UserMessage:
		content = user.Content
	case ai.UserMessage:
		content = user.Content
	default:
		return ""
	}
	var parts []string
	for _, block := range content {
		if text, ok := block.(ai.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func (s *Session) compactionEnabled() bool {
	if s.cfg.Compaction == nil {
		return true
	}
	return *s.cfg.Compaction
}

func (s *Session) compactionSettings(
	catalog *modelCatalog,
	model *ai.Model,
) *coding.CompactionSettings {
	if !s.compactionEnabled() {
		return nil
	}
	if s.cfg.CompactionSettings != nil {
		settings := *s.cfg.CompactionSettings
		return &settings
	}
	settings := coding.DefaultCompactionSettings()
	reserve, keepRecent := catalog.compactionFor(model)
	if reserve > 0 {
		settings.ReserveTokens = reserve
	}
	if keepRecent > 0 {
		settings.KeepRecentTokens = keepRecent
	}
	return settings
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func (s *Session) configFromRegistry(
	ctx context.Context,
	provider, modelID, level string,
) SessionConfig {
	if provider == "" && modelID == "" {
		if fallback, err := resolveModel(s.catalog(ctx), "", ""); err == nil {
			provider, modelID = string(fallback.Provider), fallback.ID
		}
	}

	if level == "" {
		level = string(coding.DefaultThinkingLevel)
	}
	out := SessionConfig{
		ModelProvider: provider,
		ModelID:       modelID,
		ThinkingLevel: level,
	}
	catalog := s.catalog(ctx)
	for _, model := range catalog.offeredModels() {
		info := ModelInfo{
			Provider:       string(model.Provider),
			ID:             model.ID,
			Name:           model.Name,
			Reasoning:      model.Reasoning,
			ThinkingLevels: thinkingLevelsFor(model),
		}
		out.Models = append(out.Models, info)
		if string(model.Provider) == provider && model.ID == modelID {
			out.ModelName = model.Name
			out.ThinkingLevels = info.ThinkingLevels
			out.ThinkingLevel = string(ai.ClampThinkingLevel(model, ai.ModelThinkingLevel(level)))
		}
	}

	if out.ModelName == "" && modelID != "" {
		if model := catalog.models.GetModel(provider, modelID); model != nil {
			out.ModelName = model.Name
			out.ThinkingLevels = thinkingLevelsFor(model)
			out.ThinkingLevel = string(ai.ClampThinkingLevel(model, ai.ModelThinkingLevel(level)))
		}
	}
	if len(out.ThinkingLevels) == 0 {
		out.ThinkingLevels = []string{"off"}
	}
	return out
}

func thinkingLevelsFor(model *ai.Model) []string {
	if !model.Reasoning {
		return []string{"off"}
	}
	levels := make([]string, 0, 7)
	for _, level := range []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"} {
		value, mapped := model.ThinkingLevelMap[ai.ModelThinkingLevel(level)]
		switch {
		case !mapped && (level == "xhigh" || level == "max"):
			continue
		case mapped && value == nil:
			continue
		default:
			levels = append(levels, level)
		}
	}
	return levels
}

func (s *Session) forceStopEngine() {
	if s.abandonedForShutdown() {
		return
	}
	s.mu.Lock()
	eng := s.eng
	s.mu.Unlock()
	if eng != nil {
		eng.sess.Agent.ClearAllQueues()
		eng.sess.Abort()
	}

	s.stopEngine()
}

func estimateContextTokens(eng *engine) int {
	view, stamp, trustedFrom := eng.modelView()
	eng.tokensMu.Lock()
	defer eng.tokensMu.Unlock()
	if eng.tokensAtRaw == len(view) && eng.tokensAtStamp == stamp {
		return eng.tokens
	}
	eng.tokens = estimateViewTokens(view, trustedFrom)
	eng.tokensAtRaw = len(view)
	eng.tokensAtStamp = stamp
	return eng.tokens
}

func estimateViewTokens(messages []agent.AgentMessage, trustedFrom int) int {
	index, usage, ok := lastAssistantUsage(messages, trustedFrom)
	if ok {
		tokens := usage.TotalTokens
		if tokens <= 0 {
			tokens = usage.Input + usage.Output + usage.CacheRead + usage.CacheWrite
		}
		for _, message := range messages[index+1:] {
			tokens += coding.EstimateMessageTokens(message)
		}
		return tokens
	}
	if trustedFrom <= 0 {

		return coding.EstimateContextTokens(messages)
	}
	return heuristicViewTokens(messages)
}

func heuristicViewTokens(messages []agent.AgentMessage) int {
	tokens := 0
	if system, ok := ai.GetCurrentSystemMessage(messages); ok {
		tokens = coding.EstimateMessageTokens(system)
	}
	for _, message := range messages {
		if message.MessageRole() != ai.RoleSystem {
			tokens += coding.EstimateMessageTokens(message)
		}
	}
	return tokens
}

func lastAssistantUsage(
	messages []agent.AgentMessage,
	trustedFrom int,
) (int, ai.Usage, bool) {
	if trustedFrom < 0 {
		trustedFrom = 0
	}
	for index := len(messages) - 1; index >= trustedFrom; index-- {
		assistant, ok := assistantMessage(messages[index])
		if !ok || assistant.StopReason == ai.StopAborted ||
			assistant.StopReason == ai.StopError {
			continue
		}
		usage := assistant.Usage
		if usage.TotalTokens > 0 || usage.Input+usage.Output+usage.CacheRead+usage.CacheWrite > 0 {
			return index, usage, true
		}
	}
	return 0, ai.Usage{}, false
}

func assistantMessage(message agent.AgentMessage) (*ai.AssistantMessage, bool) {
	switch msg := message.(type) {
	case *ai.AssistantMessage:
		return msg, msg != nil
	case ai.AssistantMessage:
		return &msg, true
	}
	return nil, false
}

const (
	maxToolResultImageBytes = 4 * 1024 * 1024
	maxToolResultImageCount = 8
)

func (s *Session) limitToolImages(
	_ context.Context,
	call agent.AfterToolCallContext,
) *agent.AfterToolCallResult {
	images := 0
	var total int
	for _, block := range call.Result.Content {
		if _, ok := block.(ai.ImageContent); ok {
			images++
		}
	}
	if images == 0 {
		return nil
	}

	kept := make(ai.ContentList, 0, len(call.Result.Content))
	changed := false
	for _, block := range call.Result.Content {
		image, ok := block.(ai.ImageContent)
		if !ok {
			kept = append(kept, block)
			continue
		}
		size := len(image.Data) * 3 / 4
		if images > maxToolResultImageCount || total+size > maxToolResultImageBytes {
			changed = true
			kept = append(kept, ai.TextContent{
				Text: "[image omitted: this result exceeds the request image budget]",
			})
			continue
		}
		total += size
		kept = append(kept, image)
	}
	if !changed {
		return nil
	}
	return &agent.AfterToolCallResult{Content: kept, HasContent: true}
}

func (m *Manager) integrationsFor(
	ctx context.Context,
	taskID string,
) AgencyIntegrations {
	m.modelsRegistry(ctx, taskID)
	m.catalogMu.Lock()
	defer m.catalogMu.Unlock()
	entry := m.catalogs[taskID]
	if entry == nil {
		return AgencyIntegrations{}
	}
	return AgencyIntegrations{
		Revision: entry.declared.Revision,
		MCP:      entry.declared.MCP,
		Skills:   entry.declared.Skills,
	}
}

func appendPromptBlock(prompt, block string) string {
	if strings.TrimSpace(block) == "" {
		return prompt
	}
	if strings.TrimSpace(prompt) == "" {
		return block
	}
	return prompt + "\n\n" + block
}
