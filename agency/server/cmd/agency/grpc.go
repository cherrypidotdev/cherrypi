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

package main

import (
	"context"
	"net/http"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	agentsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/agents"
	dashboardgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/dashboard"
	desktopgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/desktop"
	notificationsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/notifications"
	projectsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/projects"
	promptsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/prompts"
	realtimegrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/realtime"
	settingsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/settings"
	tagsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/tags"
	taskgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/task"
	terminalgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/terminal"
	webhookgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/webhook"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	agencyv1connect "cherrypi.vladfaust.com/shared/gen/buf/agency/v1/agencyv1connect"
	tunnelv1connect "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1/tunnelv1connect"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
	"connectrpc.com/connect"
)

func serveGrpc(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	agentService := &agentsgrpc.AgentServer{
		Agents:      appCtx.Agents,
		Prompts:     appCtx.Prompts,
		Limits:      appCtx.Limits,
		Scheduler:   appCtx.Scheduler,
		Attachments: appCtx.Attachments,
		Logs:        appCtx.AgentLogs,
		Tasks:       appCtx.Tasks,
		Hub:         appCtx.Hub,
		Accounts:    appCtx.Accounts,
	}
	path, handler := agencyv1connect.NewAgentServiceHandler(agentService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	settingsService := &settingsgrpc.SettingsServer{
		Settings:    appCtx.Settings,
		Attachments: appCtx.Attachments,
	}
	path, handler = agencyv1connect.NewSettingsServiceHandler(settingsService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	projectService := &projectsgrpc.ProjectServer{
		Projects:    appCtx.Projects,
		Agents:      appCtx.Agents,
		Hub:         appCtx.Hub,
		Attachments: appCtx.Attachments,
	}
	path, handler = agencyv1connect.NewProjectServiceHandler(projectService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	promptService := &promptsgrpc.PromptServer{
		Prompts:  appCtx.Prompts,
		Projects: appCtx.Projects,
		Agents:   appCtx.Agents,
		Tasks:    appCtx.Tasks,
		Settings: appCtx.Settings,
		Teams:    appCtx.Teams,
	}
	path, handler = agencyv1connect.NewPromptServiceHandler(promptService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	taskService := &taskgrpc.TaskServer{
		Tasks:       appCtx.Tasks,
		Tags:        appCtx.Tags,
		Agents:      appCtx.Agents,
		Hub:         appCtx.Hub,
		Scheduler:   appCtx.Scheduler,
		Attachments: appCtx.Attachments,
		Artifacts:   appCtx.Artifacts,
		Actors: func(ctx context.Context, id ids.ActorID) (taskgrpc.ActorInfo, error) {
			actor, err := appCtx.Accounts.GetActor(ctx, id)
			if err != nil {
				return taskgrpc.ActorInfo{}, err
			}
			info := taskgrpc.ActorInfo{
				Username: actor.Username,
				Name:     actor.Name,
				Kind:     actor.Kind,
			}
			if actor.PictureAttachmentID > 0 {
				info.PictureURL = attachments.URLFor(actor.PictureAttachmentID)
			}
			return info, nil
		},
		TeamName: appCtx.Teams.Name,
	}

	actorInterceptor := connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				ctx = tasks.WithActor(ctx, tasks.ActorUser)
				if actorID, ok := auth.ActorID(ctx); ok {
					ctx = tasks.WithActorID(ctx, actorID)
				}
				return next(ctx, req)
			}
		},
	)
	path, handler = agencyv1connect.NewTaskServiceHandler(
		taskService,
		connect.WithInterceptors(actorInterceptor),
	)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	tagService := &tagsgrpc.TagServer{Tags: appCtx.Tags}
	path, handler = agencyv1connect.NewTagServiceHandler(tagService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	dashboardService := &dashboardgrpc.DashboardServer{Hub: appCtx.Hub, Store: appCtx.Store}
	path, handler = agencyv1connect.NewDashboardServiceHandler(dashboardService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	terminalService := &terminalgrpc.TerminalServer{
		Hub:      appCtx.Hub,
		Agents:   appCtx.Agents,
		Realtime: appCtx.Realtime,
	}
	path, handler = agencyv1connect.NewTerminalServiceHandler(terminalService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	realtimeService := &realtimegrpc.Server{Bus: appCtx.Realtime}
	path, handler = agencyv1connect.NewRealtimeServiceHandler(realtimeService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	desktopService := &desktopgrpc.DesktopServer{Hub: appCtx.Hub}
	path, handler = agencyv1connect.NewDesktopServiceHandler(desktopService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	notificationService := &notificationsgrpc.NotificationServer{
		Notifications: appCtx.Notifications,
	}
	path, handler = agencyv1connect.NewNotificationServiceHandler(notificationService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	webhookService := &webhookgrpc.WebhookServer{Store: appCtx.Store, Notifier: appCtx.Webhooks}
	path, handler = agencyv1connect.NewWebhookServiceHandler(webhookService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	tunnelService := &tunnelServer{appCtx: appCtx}
	path, handler = tunnelv1connect.NewTunnelServiceHandler(tunnelService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))
	mux.Handle(sharedtunnel.WSPath, http.HandlerFunc(tunnelService.WS))
}
