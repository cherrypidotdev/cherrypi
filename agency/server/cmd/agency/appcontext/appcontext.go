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
	"log"
	"os"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agentlogs"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/artifacts"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	agencyevents "cherrypi.vladfaust.com/agency/internal/events"
	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/scheduler"
	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/storage"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/teams"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	"cherrypi.vladfaust.com/agency/internal/webhook"
)

// AppContext wires the agency's singletons.
type AppContext struct {
	Config        Config
	Store         store.Store
	Storage       storage.Store
	Accounts      *accounts.Service
	Agents        *agents.Registry
	Settings      *settings.Service
	Projects      *projects.Service
	Prompts       *prompts.Service
	Tasks         *tasks.Service
	Attachments   *attachments.Service
	Artifacts     *artifacts.Service
	Tags          *tags.Service
	Limits        *limits.Service
	Scheduler     *scheduler.Scheduler
	Hub           *tunnel.Hub
	Notifications *notifications.Service
	Webhooks      *webhook.Notifier
	AgentLogs     *agentlogs.Service
	Teams         *teams.Service
	Audit         *audit.Service
}

// taskTransitionNotifier turns a task lifecycle transition into an inbox
// notification. The input and failed signals are excluded: the events
// consumer already records those with the agent's dialog title and error text.
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
		// Completing or cancelling a task is something the operator has just
		// done themselves, so it must not settle into their own inbox as news
		// (or chime on their own action). The hook runs on the request context,
		// so the coarse actor tells a manual close-out from a system/agent one.
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
		// A task notification reaches its leads and subscribers only; a legacy
		// task with neither falls back to a broadcast entry.
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

// taskTransitionBody renders a task's name, falling back to the first line of
// its prompt, so the inbox row says which task moved.
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

// NewAppContext parses the config and opens the database.
func NewAppContext() *AppContext {
	config := ParseConfig()

	// The default directory may not exist yet; SQLite won't create it.
	// PostgreSQL does not use it, so a read-only data directory is fine there.
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
		log.Printf("Object storage enabled (bucket %s)", config.S3.Bucket)
	} else {
		log.Print("Object storage disabled: CHERRYPI_S3_ENDPOINT/BUCKET unset")
	}

	startup := make([]accounts.StartupCredential, len(config.Credentials))
	for i, c := range config.Credentials {
		startup[i] = accounts.StartupCredential{Username: c.Username, Password: c.Password}
	}

	accountsSvc := accounts.New(st)
	if err := bootstrapAccounts(context.Background(), accountsSvc, config, startup); err != nil {
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

	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			_ = accountsSvc.Cleanup(ctx)
			_, _ = teamsSvc.PruneInvitations(ctx, time.Now())
			_, _ = projectsSvc.PruneInvitations(ctx, time.Now())
			_, _ = auditSvc.Prune(ctx, time.Now())
		}
	}()

	if config.SeedAgentUsername != "" {
		if err := agentRegistry.EnsureCredential(
			context.Background(),
			config.SeedAgentUsername,
			config.SeedAgentPassword,
		); err != nil {
			log.Fatalf("Failed to seed agent %q: %v", config.SeedAgentUsername, err)
		}
		log.Printf("Seeded agent %s", config.SeedAgentUsername)
	}

	attachmentsSvc := attachments.New(st, objectStore, attachments.Config{
		KeyPrefix:     config.S3.KeyPrefix,
		MaxBytes:      config.AttachmentMaxBytes,
		MaxPerMessage: config.AttachmentMaxPerMessage,
	})
	go attachmentsSvc.RunGC(context.Background(), 30*time.Minute)

	notificationsSvc := notifications.New(st)
	teamsSvc.Notifier = notificationsSvc
	notifier := webhook.NewNotifier(st)
	agentLogsSvc := agentlogs.New(st)
	hub := tunnel.NewHub(agentRegistry)

	promptsSvc := prompts.New(st)
	tagsSvc := tags.New(st)
	tasksSvc := tasks.New(st, projectsSvc)
	tasksSvc.OnTransition = taskTransitionNotifier(notificationsSvc, tasksSvc, agentRegistry)
	limitsSvc := limits.New(st)
	sched := scheduler.New(
		tasksSvc,
		projectsSvc,
		limitsSvc,
		promptsSvc,
		agentRegistry,
		hub,
	)
	sched.Attachments = attachmentsSvc

	artifactsSvc := artifacts.New(st)

	consumer := &agencyevents.Consumer{
		Store:         st,
		Notifier:      notifier,
		Notifications: notificationsSvc,
		Tasks:         tasksSvc,
		Artifacts:     artifactsSvc,
		Logs:          agentLogsSvc,
		OnTaskFailed:  sched.Wake,
	}
	hub.OnConnect = func(ctx context.Context, conn *tunnel.Connection) {
		go consumer.Run(ctx, conn)
		// A fresh agent may have a free slot; drain the queue onto it.
		sched.Wake()
		// The agent's own connect line races the tunnel subscribe, so the
		// agency records the authoritative connectivity event.
		logAgentEvent(agentLogsSvc, conn.ID, agentlogs.LevelInfo, "agent connected")
	}
	hub.OnDisconnect = func(agentID ids.AgentID) {
		logAgentEvent(agentLogsSvc, agentID, agentlogs.LevelWarn, "agent disconnected")
	}

	go sched.Run(context.Background())

	return &AppContext{
		Config:        config,
		Store:         st,
		Storage:       objectStore,
		Accounts:      accountsSvc,
		Agents:        agentRegistry,
		Settings:      settings.New(st),
		Projects:      projectsSvc,
		Prompts:       promptsSvc,
		Tasks:         tasksSvc,
		Attachments:   attachmentsSvc,
		Artifacts:     artifactsSvc,
		Tags:          tagsSvc,
		Limits:        limitsSvc,
		Scheduler:     sched,
		Hub:           hub,
		Notifications: notificationsSvc,
		Webhooks:      notifier,
		AgentLogs:     agentLogsSvc,
		Teams:         teamsSvc,
		Audit:         auditSvc,
	}
}

// logAgentEvent records an agency-observed connectivity event in the same log
// stream as the agent's own lines.
func logAgentEvent(svc *agentlogs.Service, agentID ids.AgentID, level int, message string) {
	_ = svc.Insert(context.Background(), agentlogs.Log{
		AgentID: agentID,
		At:      time.Now().UnixMilli(),
		Level:   level,
		Message: message,
	})
}
