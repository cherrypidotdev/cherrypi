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

package appcontext

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agentgroups"
	"cherrypi.vladfaust.com/agency/internal/agentlogs"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/archiver"
	"cherrypi.vladfaust.com/agency/internal/artifacts"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	agencyevents "cherrypi.vladfaust.com/agency/internal/events"
	"cherrypi.vladfaust.com/agency/internal/health"
	"cherrypi.vladfaust.com/agency/internal/kpilayout"
	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	pluginhost "cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/processlog"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/promptaccess"
	"cherrypi.vladfaust.com/agency/internal/promptflow"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/realtime"
	"cherrypi.vladfaust.com/agency/internal/scheduler"
	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/sites"
	"cherrypi.vladfaust.com/agency/internal/speech"
	"cherrypi.vladfaust.com/agency/internal/storage"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/taskinput"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/taskwake"
	"cherrypi.vladfaust.com/agency/internal/teams"
	"cherrypi.vladfaust.com/agency/internal/templates"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	"cherrypi.vladfaust.com/agency/internal/webhook"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
)

type AppContext struct {
	Config        Config
	Store         store.Store
	Storage       storage.Store
	Accounts      *accounts.Service
	Agents        *agents.Registry
	Settings      *settings.Service
	Projects      *projects.Service
	Prompts       *prompts.Service
	PromptFlow    *promptflow.Service
	Tasks         *tasks.Service
	Attachments   *attachments.Service
	Artifacts     *artifacts.Service
	Sites         *sites.Service
	Tags          *tags.Service
	Limits        *limits.Service
	Scheduler     *scheduler.Scheduler
	Hub           *tunnel.Hub
	Notifications *notifications.Service
	Webhooks      *webhook.Notifier
	AgentLogs     *agentlogs.Service
	Health        *health.Service
	Teams         *teams.Service
	Groups        *agentgroups.Service
	Templates     *templates.Service
	Audit         *audit.Service

	Realtime *realtime.Bus

	ProcessLogs *processlog.Log

	Speech *speech.Service

	Plugins *pluginhost.Service

	Input *taskinput.Service

	KpiLayouts *kpilayout.Service
}

func taskTransitionNotifier(
	notificationsSvc *notifications.Service,
	tasksSvc *tasks.Service,
	agentsReg *agents.Registry,
) func(context.Context, tasks.Transition) {
	catalog := map[tasks.State]struct {
		kind     string
		title    string
		severity string
	}{
		tasks.StateRunning:   {kind: "task_started", title: "Task started", severity: "info"},
		tasks.StateSettled:   {kind: "task_finished", title: "Task settled", severity: "info"},
		tasks.StateCompleted: {kind: "task_completed", title: "Task completed", severity: "info"},
		tasks.StateCancelled: {kind: "task_cancelled", title: "Task cancelled", severity: "info"},
	}

	return func(ctx context.Context, tr tasks.Transition) {
		entry, ok := catalog[tr.State]
		if !ok {
			return
		}

		if (tr.State == tasks.StateCompleted || tr.State == tasks.StateCancelled) &&
			tasks.ActorFrom(ctx) == tasks.ActorUser {
			return
		}
		task, err := tasksSvc.Get(ctx, tr.TaskID)
		if err != nil {
			return
		}
		agent, err := agentsReg.GetByActor(ctx, task.AssigneeActorID)
		if err != nil {
			return
		}

		recipients, _ := tasksSvc.Recipients(ctx, task.ID)
		_ = notificationsSvc.CreateForTask(ctx, task.ID, recipients, notifications.Notification{
			AgentID:  agent.ID,
			Kind:     entry.kind,
			Severity: entry.severity,
			Title:    entry.title,
			Body:     taskTransitionBody(task),
		})
	}
}

func commentNotifier(
	notificationsSvc *notifications.Service,
	tasksSvc *tasks.Service,
	agentsReg *agents.Registry,
) func(context.Context, string, tasks.Comment) {
	return func(ctx context.Context, taskID string, comment tasks.Comment) {
		task, err := tasksSvc.Get(ctx, taskID)
		if err != nil {
			return
		}
		recipients, err := tasksSvc.Recipients(ctx, taskID)
		if err != nil {
			return
		}
		mentioned := make(map[ids.ActorID]bool, len(comment.MentionedActorIDs))
		for _, actorID := range comment.MentionedActorIDs {
			mentioned[actorID] = true
		}
		filtered := make([]ids.ActorID, 0, len(recipients))
		for _, recipient := range recipients {

			if recipient != comment.AuthorActorID && !mentioned[recipient] {
				filtered = append(filtered, recipient)
			}
		}
		if len(filtered) == 0 {
			return
		}

		agent, _ := agentsReg.GetByActor(ctx, task.AssigneeActorID)
		sender, _ := tasksSvc.Sender(ctx, comment.AuthorActorID)
		author := sender.Name
		if author == "" {
			author = sender.Username
		}
		body := commentExcerpt(comment.Body)
		if author != "" {
			body = author + ": " + body
		}
		_ = notificationsSvc.CreateForTask(ctx, taskID, filtered, notifications.Notification{
			AgentID:  agent.ID,
			Kind:     "task_comment",
			Severity: "info",
			Title:    "New comment on " + taskTransitionBody(task),
			Body:     body,
		})
	}
}

func mentionNotifier(
	notificationsSvc *notifications.Service,
	tasksSvc *tasks.Service,
	agentsReg *agents.Registry,
) func(context.Context, string, tasks.Comment) {
	return func(ctx context.Context, taskID string, comment tasks.Comment) {
		if len(comment.MentionedActorIDs) == 0 {
			return
		}
		task, err := tasksSvc.Get(ctx, taskID)
		if err != nil {
			return
		}
		agent, _ := agentsReg.GetByActor(ctx, task.AssigneeActorID)
		sender, _ := tasksSvc.Sender(ctx, comment.AuthorActorID)
		author := sender.Name
		if author == "" {
			author = sender.Username
		}
		body := commentExcerpt(comment.Body)
		if author != "" {
			body = author + ": " + body
		}
		_ = notificationsSvc.CreateForTask(
			ctx,
			taskID,
			comment.MentionedActorIDs,
			notifications.Notification{
				AgentID:  agent.ID,
				Kind:     "task_mention",
				Severity: "info",
				Title:    "You were mentioned on " + taskTransitionBody(task),
				Body:     body,
			},
		)
	}
}

// commentImage matches Markdown image syntax. An inbox line is a text preview,

var commentImage = regexp.MustCompile(`!\[([^\]\n]*)\]\([^)]+\)`)

func commentExcerpt(body string) string {
	const limit = 140
	body = commentImage.ReplaceAllString(body, "$1")
	text := strings.Join(strings.Fields(body), " ")
	if len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "…"
}

// taskTransitionBody renders a task's name, falling back to the first line of

const objectStorageCheckTimeout = 10 * time.Second

func checkObjectStorage(cfg storage.Config, objects storage.Store) error {
	ctx, cancel := context.WithTimeout(context.Background(), objectStorageCheckTimeout)
	defer cancel()

	result, err := objects.Check(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", describeObjectStorage(cfg), err)
	}
	if result.Region == "" {
		log.Printf(
			"Object storage bucket region unknown (%v); keeping CHERRYPI_S3_REGION=%s",
			result.RegionWarning,
			cfg.Region,
		)
		return nil
	}
	if !strings.EqualFold(result.Region, cfg.Region) {
		return fmt.Errorf(
			"%s, but the bucket is in region %q (from %s); set CHERRYPI_S3_REGION=%s",
			describeObjectStorage(cfg),
			result.Region,
			result.RegionSource,
			result.Region,
		)
	}
	return nil
}

func describeObjectStorage(cfg storage.Config) string {
	style := "virtual-host"
	if cfg.ForcePathStyle {
		style = "path-style"
	}
	description := fmt.Sprintf(
		"bucket %q at %s using %s addressing",
		cfg.Bucket,
		cfg.Endpoint,
		style,
	)
	if !cfg.ForcePathStyle && !isAWSEndpoint(cfg.Endpoint) {
		description += "; if the endpoint does not serve <bucket>.<host>, set CHERRYPI_S3_FORCE_PATH_STYLE=true"
	}
	return description
}

func isAWSEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "amazonaws.com" || strings.HasSuffix(host, ".amazonaws.com")
}

func taskTransitionBody(task tasks.Task) string {
	if name := strings.TrimSpace(task.Name); name != "" {
		return name
	}
	prompt := strings.TrimSpace(task.Prompt)
	if line, _, _ := strings.Cut(prompt, "\n"); line != "" {
		return line
	}
	return task.ID
}

func NewAppContext(config Config, processLog *processlog.Log) *AppContext {

	if !store.IsPostgresDSN(config.DatabaseDSN) {
		if err := os.MkdirAll(config.Dir, 0o755); err != nil {
			log.Fatalf("Failed to create CHERRYPI_AGENCY_DIR %s: %v", config.Dir, err)
		}
	}

	st := store.NewStore(config.DatabaseDSN)

	objectStore, err := storage.New(context.Background(), config.S3)
	if err != nil {
		log.Fatalf("Failed to configure object storage: %v", err)
	}
	if objectStore.Enabled() {
		if err := checkObjectStorage(config.S3, objectStore); err != nil {
			log.Fatalf("Object storage check failed: %v", err)
		}
		log.Printf(
			"Object storage enabled (bucket %s, endpoint %s, path-style=%t)",
			config.S3.Bucket,
			config.S3.Endpoint,
			config.S3.ForcePathStyle,
		)
	} else {
		log.Print("Object storage disabled: CHERRYPI_S3_ENDPOINT/BUCKET unset")
	}

	accountsSvc := accounts.New(st)
	if err := bootstrapAccounts(context.Background(), accountsSvc, config); err != nil {
		log.Fatalf("Failed to bootstrap accounts: %v", err)
	}

	agentRegistry := agents.NewRegistry(st)
	auditSvc := audit.New(st)
	accountsSvc.Audit = auditSvc
	agentRegistry.Audit = auditSvc

	teamsSvc := teams.New(st)
	teamsSvc.AgentOwners = agentRegistry

	projectsSvc := projects.New(st)
	projectsSvc.TeamOwners = teamsSvc

	groupsSvc := agentgroups.New(st)
	groupsSvc.Agents = agentRegistry
	groupsSvc.Projects = projectsSvc
	groupsSvc.Teams = teamsSvc
	templatesSvc := templates.New(st, projectsSvc)

	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			_ = accountsSvc.Cleanup(ctx)
			_, _ = teamsSvc.PruneInvitations(ctx, time.Now())
			_, _ = groupsSvc.PruneRequests(ctx, time.Now())
			_, _ = groupsSvc.PruneAccessInvitations(ctx, time.Now())
			_, _ = projectsSvc.PruneInvitations(ctx, time.Now())
			_, _ = auditSvc.Prune(ctx, time.Now())
		}
	}()

	attachmentsSvc := attachments.New(st, objectStore, attachments.Config{
		KeyPrefix:     config.S3.KeyPrefix,
		MaxBytes:      config.AttachmentMaxBytes,
		MaxPerMessage: config.AttachmentMaxPerMessage,
	})
	go attachmentsSvc.RunGC(context.Background(), 30*time.Minute)

	notificationsSvc := notifications.New(st)
	teamsSvc.Notifier = notificationsSvc
	groupsSvc.Notifications = notificationsSvc
	notifier := webhook.NewNotifier(st)
	agentLogsSvc := agentlogs.New(st)
	healthSvc := health.New(st)
	hub := tunnel.NewHub(agentRegistry)
	realtimeBus := realtime.New()

	promptsSvc := prompts.New(st)
	tagsSvc := tags.New(st)
	settingsSvc := settings.New(st)
	tasksSvc := tasks.New(st, projectsSvc)

	pluginStore, err := pluginhost.OpenStore(config.DatabaseDSN)
	if err != nil {
		log.Fatalf("Failed to open the plugin store: %v", err)
	}
	pluginsSvc := pluginhost.NewService(pluginhost.Config{
		DSN:                 config.DatabaseDSN,
		DataDir:             config.PluginDataDir,
		RequirePGDatabase:   config.PluginPGRequireDatabase,
		AllowPrivateNetwork: config.PluginAllowPrivateNetwork,
		PublicURL:           config.PublicURL,
		DisableTicks:        config.PluginDisableTicks,
	}, pluginStore)
	pluginsSvc.Projects = projectsSvc
	pluginsSvc.Agents = agentRegistry
	pluginsSvc.Accounts = accountsSvc
	pluginsSvc.Tasks = tasksSvc
	pluginsSvc.Settings = settingsSvc
	pluginsSvc.Teams = teamsSvc
	pluginsSvc.Tags = tagsSvc
	if err := pluginsSvc.Bootstrap(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap plugins: %v", err)
	}

	onTransition := taskTransitionNotifier(notificationsSvc, tasksSvc, agentRegistry)
	tasksSvc.OnTransition = func(ctx context.Context, tr tasks.Transition) {
		onTransition(ctx, tr)
		realtimeBus.PublishTask(tr.TaskID)

		if tr.Held {
			pluginsSvc.DispatchEvent(ctx, pluginhost.Event{
				Name:   "task_held",
				TaskID: tr.TaskID,
				State:  string(tr.State),
				Extra:  map[string]any{"holds": pluginHoldsPayload(tr.Holds)},
			})
			return
		}
		if event := pluginEventForState(tr.State); event != "" {
			pluginsSvc.DispatchEvent(ctx, pluginhost.Event{
				Name:   event,
				TaskID: tr.TaskID,
				State:  string(tr.State),
			})
		}
	}
	tasksSvc.OnCreated = func(ctx context.Context, taskID string) {
		pluginsSvc.DispatchEvent(ctx, pluginhost.Event{Name: "task_created", TaskID: taskID})
	}

	tasksSvc.OnHoldReleased = func(ctx context.Context, release tasks.HoldRelease) {
		pluginsSvc.DispatchEvent(ctx, pluginhost.HoldReleaseEvent(release))
	}
	tasksSvc.OnChange = func(_ context.Context, taskID string) {
		realtimeBus.PublishTask(taskID)
	}
	tasksSvc.OnComment = commentNotifier(notificationsSvc, tasksSvc, agentRegistry)
	tasksSvc.OnMention = mentionNotifier(notificationsSvc, tasksSvc, agentRegistry)
	limitsSvc := limits.New(st)
	tasksSvc.Limits = limitsSvc
	sched := scheduler.New(
		tasksSvc,
		projectsSvc,
		limitsSvc,
		promptsSvc,
		agentRegistry,
		hub,
	)
	sched.Attachments = attachmentsSvc
	sched.Plugins = pluginsSvc

	pluginsSvc.FollowUp = sched.FollowUp
	pluginsSvc.RecordMessage = func(
		ctx context.Context,
		taskID, messageID, text string,
		actorID ids.ActorID,
		at int64,
	) {
		if err := attachmentsSvc.RecordMessage(
			ctx, taskID, messageID, text, actorID, nil, at,
		); err != nil {
			log.Printf("[plugins] record message for %s: %v", taskID, err)
		}
	}

	waker := &taskwake.Waker{
		Tasks: tasksSvc,
		Wake:  sched.Wake,
		Steer: func(ctx context.Context, t tasks.Task, prompt string) error {
			agent, err := agentRegistry.GetByActor(ctx, t.AssigneeActorID)
			if err != nil {
				return err
			}
			conn, err := hub.Connection(agent.ID)
			if err != nil {
				return err
			}
			sender, err := tasksSvc.Sender(ctx, 0)
			if err != nil {
				return err
			}
			_, err = conn.Tasks.AddToTask(ctx, &agentv1.AddToTaskRequest{
				Id:     t.SessionID,
				Prompt: prompt,
				TaskId: t.ID,
				Ref: &agentv1.TaskRef{
					ProjectId: t.ProjectID,
					Seq:       t.Seq,
				},
				Sender: &agentv1.PromptSender{
					ActorId:  int64(sender.ActorID),
					Username: sender.Username,
					Name:     sender.Name,
					Kind:     sender.Kind,
				},
			})
			return err
		},
	}
	waker.LeadOf = tasksSvc.IsLead
	waker.AgentProtocol = func(ctx context.Context, actorID ids.ActorID) (string, error) {
		agent, err := agentRegistry.GetByActor(ctx, actorID)
		if err != nil {
			return "", err
		}
		return agent.ProtocolVersion, nil
	}
	tasksSvc.OnChildFinished = waker.OnChildFinished

	tasksSvc.OnInputCleared = func(ctx context.Context, t tasks.Task) {
		sched.DeliverPendingPrompt(ctx, t)
		waker.OnInputCleared(ctx, t)
	}

	tasksSvc.OnPendingPromptQueued = func(context.Context, string) {
		sched.Wake()
	}

	artifactsSvc := artifacts.New(st)
	sitesSvc := sites.New(
		st,
		attachmentsSvc,
		accountsSvc,
		agentRegistry,
		tasksSvc,
		realtimeBus,
		sites.Config{
			CacheDir: filepath.Join(config.Dir, "sites"),
			MaxBytes: sites.DefaultMaxBytes,
		},
	)
	if config.SitesInsecureHTTP {
		log.Print(
			"WARNING: CHERRYPI_SITES_INSECURE_HTTP is enabled: raw site content " +
				"will be served without authentication over plain HTTP. " +
				"Use HTTPS for any non-local deployment.",
		)
	}

	consumer := &agencyevents.Consumer{
		Store:         st,
		Notifier:      notifier,
		Notifications: notificationsSvc,
		Tasks:         tasksSvc,
		Artifacts:     artifactsSvc,
		Logs:          agentLogsSvc,
		Health:        healthSvc,
		Realtime:      realtimeBus,
		OnTaskFailed:  sched.Wake,
		OnTaskEvent: func(ctx context.Context, event, taskID string, extra map[string]any) {
			pluginsSvc.DispatchEvent(ctx, pluginhost.Event{
				Name:   event,
				TaskID: taskID,
				Extra:  extra,
			})
		},

		OnInputRequested: waker.OnInputRequested,
	}
	hub.OnConnect = func(ctx context.Context, conn *tunnel.Connection) {
		go consumer.Run(ctx, conn)

		go pushHealthInterval(ctx, agentRegistry, conn)

		sched.Wake()
		// The agent's own connect line races the tunnel subscribe, so the

		logAgentEvent(agentLogsSvc, conn.ID, agentlogs.LevelInfo, "agent connected")
		realtimeBus.Publish(agencyv1.RealtimeKind_REALTIME_KIND_AGENTS)
		realtimeBus.Publish(agencyv1.RealtimeKind_REALTIME_KIND_AGENT_LOGS)
	}
	hub.OnDisconnect = func(agentID ids.AgentID) {
		logAgentEvent(agentLogsSvc, agentID, agentlogs.LevelWarn, "agent disconnected")
		realtimeBus.Publish(agencyv1.RealtimeKind_REALTIME_KIND_AGENTS)
		realtimeBus.Publish(agencyv1.RealtimeKind_REALTIME_KIND_AGENT_LOGS)
	}
	hub.OnRelayDone = func(ids.AgentID) {
		realtimeBus.Publish(agencyv1.RealtimeKind_REALTIME_KIND_TERMINALS)
	}

	tasksSvc.DeleteAgentSession = deleteAgentSession(hub)

	go sched.Run(context.Background())
	go archiver.New(tasksSvc, settingsSvc).Run(context.Background())

	inputSvc := &taskinput.Service{
		Tasks:  tasksSvc,
		Agents: agentRegistry,
		Hub:    hub,
		Actors: func(ctx context.Context, id ids.ActorID) (taskinput.ActorInfo, error) {
			actor, err := accountsSvc.GetActor(ctx, id)
			if err != nil {
				return taskinput.ActorInfo{}, err
			}
			return taskinput.ActorInfo{
				Username: actor.Username,
				Name:     actor.Name,
				Kind:     actor.Kind,
			}, nil
		},
	}
	inputSvc.OnAnswered = func(ctx context.Context, answered taskinput.Answered) {
		pluginsSvc.DispatchEvent(ctx, pluginhost.TaskInputAnsweredEvent(
			answered.Task,
			answered.Dialog.RequestID,
			answered.Dialog.Method,
			answered.Dialog.Title,
			answered.Dialog.Message,
			answered.Summary(),
			answered.AnsweredBy(),
		))
	}

	return &AppContext{
		Config:   config,
		Store:    st,
		Storage:  objectStore,
		Accounts: accountsSvc,
		Agents:   agentRegistry,
		Settings: settingsSvc,
		Speech:   speech.New(settingsSvc),
		Projects: projectsSvc,
		Prompts:  promptsSvc,
		PromptFlow: &promptflow.Service{
			Prompts: promptsSvc,
			Access: &promptaccess.Authorizer{
				Agents:   agentRegistry,
				Projects: projectsSvc,
				Teams:    teamsSvc,
				Groups:   groupsSvc,
			},
			Notifications: notificationsSvc,
			Audit:         auditSvc,
			Projects:      projectsSvc,
			Teams:         teamsSvc,
			Groups:        groupsSvc,
			Agents:        agentRegistry,
			Accounts:      accountsSvc,
		},
		Tasks:         tasksSvc,
		Attachments:   attachmentsSvc,
		Artifacts:     artifactsSvc,
		Sites:         sitesSvc,
		Tags:          tagsSvc,
		Limits:        limitsSvc,
		Scheduler:     sched,
		Hub:           hub,
		Notifications: notificationsSvc,
		Webhooks:      notifier,
		AgentLogs:     agentLogsSvc,
		Health:        healthSvc,
		Teams:         teamsSvc,
		Groups:        groupsSvc,
		Templates:     templatesSvc,
		Audit:         auditSvc,
		Realtime:      realtimeBus,
		ProcessLogs:   processLog,
		Plugins:       pluginsSvc,
		KpiLayouts:    kpilayout.New(st),
		Input:         inputSvc,
	}
}

func deleteAgentSession(
	hub *tunnel.Hub,
) func(ctx context.Context, agentID ids.AgentID, sessionID string) error {
	return func(
		ctx context.Context,
		agentID ids.AgentID,
		sessionID string,
	) error {
		conn, err := hub.Connection(agentID)
		if err != nil {
			return err
		}
		_, err = conn.Tasks.DeleteTask(
			ctx,
			&agentv1.DeleteTaskRequest{Id: sessionID},
		)
		if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
			return err
		}
		return nil
	}
}

func pluginHoldsPayload(holds []tasks.TaskHold) []map[string]any {
	out := make([]map[string]any, len(holds))
	for i, hold := range holds {
		out[i] = map[string]any{
			"plugin":      hold.PluginID,
			"plugin_name": hold.PluginName,
			"key":         hold.Key,
			"reason":      hold.Reason,
			"created_at":  hold.CreatedAt.UnixMilli(),
		}
	}
	return out
}

func pluginEventForState(state tasks.State) string {
	switch state {
	case tasks.StateRunning:
		return "task_started"
	case tasks.StateSettled:
		return "task_settled"
	case tasks.StateCompleted:
		return "task_completed"
	case tasks.StateCancelled:
		return "task_cancelled"
	default:
		return ""
	}
}

func logAgentEvent(svc *agentlogs.Service, agentID ids.AgentID, level int, message string) {
	_ = svc.Insert(context.Background(), agentlogs.Log{
		AgentID: agentID,
		At:      time.Now().UnixMilli(),
		Level:   level,
		Message: message,
	})
}

func pushHealthInterval(
	ctx context.Context,
	reg *agents.Registry,
	conn *tunnel.Connection,
) {
	settings, err := reg.HealthSettings(ctx, conn.ID)
	if err != nil {
		log.Printf("Read health interval for agent %d: %v", conn.ID, err)
		return
	}
	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := conn.SetHealthInterval(pushCtx, settings.Seconds()); err != nil {
		if connect.CodeOf(err) == connect.CodeUnimplemented {
			return
		}
		log.Printf("Push health interval to agent %d: %v", conn.ID, err)
	}
}
