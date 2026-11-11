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
	"fmt"
	"log"
	"net/http"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	agentsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/agents"
	dashboardgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/dashboard"
	desktopgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/desktop"
	kpigrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/kpi"
	notificationsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/notifications"
	pluginsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/plugins"
	projectsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/projects"
	promptsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/prompts"
	realtimegrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/realtime"
	settingsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/settings"
	sitesgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/sites"
	tagsgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/tags"
	taskgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/task"
	tasktemplatesgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/tasktemplates"
	terminalgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/terminal"
	webhookgrpc "cherrypi.vladfaust.com/agency/cmd/agency/grpc/webhook"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	pluginhost "cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/telemetry"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agencyv1connect "cherrypi.vladfaust.com/shared/gen/buf/agency/v1/agencyv1connect"
	tunnelv1connect "cherrypi.vladfaust.com/shared/gen/buf/tunnel/v1/tunnelv1connect"
	sharedtunnel "cherrypi.vladfaust.com/shared/tunnel"
	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
)

func pluginInputRequest(
	taskID, requestID string,
	answer pluginhost.TaskInputAnswer,
) (*agencyv1.RespondToExtensionUiRequest, error) {
	req := &agencyv1.RespondToExtensionUiRequest{
		Id:        taskID,
		RequestId: requestID,
	}
	switch answer.Kind {
	case pluginhost.InputAnswerValue:
		req.Response = &agencyv1.RespondToExtensionUiRequest_Value{
			Value: answer.Value,
		}
	case pluginhost.InputAnswerConfirmed:
		req.Response = &agencyv1.RespondToExtensionUiRequest_Confirmed{
			Confirmed: answer.Confirmed,
		}
	case pluginhost.InputAnswerCancelled:
		req.Response = &agencyv1.RespondToExtensionUiRequest_Cancelled{
			Cancelled: true,
		}
	default:
		return nil, fmt.Errorf("unknown input answer kind %q", answer.Kind)
	}
	for _, qa := range answer.Questionnaire {
		req.Questionnaire = append(req.Questionnaire,
			&agencyv1.QuestionnaireAnswer{
				Question: qa.Question,
				Answer:   qa.Answer,
			})
	}
	return req, nil
}

func serveGrpc(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	var telemetryInterceptor *otelconnect.Interceptor
	if telemetry.Enabled() {
		var err error

		telemetryInterceptor, err = otelconnect.NewInterceptor(
			otelconnect.WithFilter(func(_ context.Context, spec connect.Spec) bool {
				return spec.StreamType == connect.StreamTypeUnary
			}),
		)
		if err != nil {
			log.Printf("OpenTelemetry RPC interceptor disabled: %v", err)
		}
	}

	handlerOptions := func(extra ...connect.HandlerOption) []connect.HandlerOption {
		opts := make([]connect.HandlerOption, 0, len(extra)+1)
		if telemetryInterceptor != nil {
			opts = append(opts, connect.WithInterceptors(telemetryInterceptor))
		}
		return append(opts, extra...)
	}

	agentService := &agentsgrpc.AgentServer{
		Agents:      appCtx.Agents,
		KpiLayouts:  appCtx.KpiLayouts,
		Prompts:     appCtx.Prompts,
		Limits:      appCtx.Limits,
		Scheduler:   appCtx.Scheduler,
		Attachments: appCtx.Attachments,
		Logs:        appCtx.AgentLogs,
		Tasks:       appCtx.Tasks,
		Hub:         appCtx.Hub,
		Accounts:    appCtx.Accounts,
	}
	path, handler := agencyv1connect.NewAgentServiceHandler(agentService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	settingsService := &settingsgrpc.SettingsServer{
		Settings:    appCtx.Settings,
		Attachments: appCtx.Attachments,
	}
	path, handler = agencyv1connect.NewSettingsServiceHandler(settingsService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	kpiService := &kpigrpc.Server{
		Layouts:  appCtx.KpiLayouts,
		Projects: appCtx.Projects,
		Agents:   appCtx.Agents,
	}
	path, handler = agencyv1connect.NewKpiLayoutServiceHandler(kpiService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	siteService := &sitesgrpc.SiteServer{
		Sites:       appCtx.Sites,
		Accounts:    appCtx.Accounts,
		Attachments: appCtx.Attachments,
	}
	path, handler = agencyv1connect.NewSiteServiceHandler(siteService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	projectService := &projectsgrpc.ProjectServer{
		Projects:    appCtx.Projects,
		KpiLayouts:  appCtx.KpiLayouts,
		Agents:      appCtx.Agents,
		Hub:         appCtx.Hub,
		Attachments: appCtx.Attachments,
		Limits:      appCtx.Limits,
		Tags:        appCtx.Tags,
		Scheduler:   appCtx.Scheduler,
	}
	path, handler = agencyv1connect.NewProjectServiceHandler(projectService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	promptService := &promptsgrpc.PromptServer{
		Flow:     appCtx.PromptFlow,
		Prompts:  appCtx.Prompts,
		Projects: appCtx.Projects,
		Agents:   appCtx.Agents,
		Tasks:    appCtx.Tasks,
		Settings: appCtx.Settings,
		Teams:    appCtx.Teams,
		Accounts: appCtx.Accounts,
	}
	path, handler = agencyv1connect.NewPromptServiceHandler(promptService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	templateService := &tasktemplatesgrpc.Server{Templates: appCtx.Templates}
	path, handler = agencyv1connect.NewTaskTemplateServiceHandler(
		templateService,
		handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	taskService := &taskgrpc.TaskServer{
		Tasks:       appCtx.Tasks,
		Tags:        appCtx.Tags,
		Agents:      appCtx.Agents,
		Hub:         appCtx.Hub,
		Scheduler:   appCtx.Scheduler,
		Attachments: appCtx.Attachments,
		Artifacts:   appCtx.Artifacts,
		Templates:   appCtx.Templates,
		Plugins:     appCtx.Plugins,
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
		Input:    appCtx.Input,
	}

	appCtx.Plugins.RespondInput = func(
		ctx context.Context,
		plugin pluginhost.Plugin,
		task tasks.Task,
		requestID string,
		answer pluginhost.TaskInputAnswer,
	) error {
		inner, err := pluginInputRequest(task.ID, requestID, answer)
		if err != nil {
			return err
		}
		actorID := ids.ActorID(plugin.ActorID)
		ctx = auth.WithIdentity(ctx, auth.Identity{
			ActorID:  actorID,
			Kind:     "plugin",
			Username: plugin.ActorUsername,
			Name:     plugin.ActorName,
		})
		ctx = tasks.WithActor(ctx, tasks.ActorPlugin)
		ctx = tasks.WithActorID(ctx, actorID)
		_, err = taskService.RespondToExtensionUi(ctx, inner)
		return err
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
		handlerOptions(connect.WithInterceptors(actorInterceptor))...,
	)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	tagService := &tagsgrpc.TagServer{
		Tags:      appCtx.Tags,
		Scheduler: appCtx.Scheduler,
	}
	path, handler = agencyv1connect.NewTagServiceHandler(tagService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	dashboardService := &dashboardgrpc.DashboardServer{
		Hub:    appCtx.Hub,
		Store:  appCtx.Store,
		Health: appCtx.Health,
	}
	path, handler = agencyv1connect.NewDashboardServiceHandler(
		dashboardService,
		handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	terminalService := &terminalgrpc.TerminalServer{
		Hub:      appCtx.Hub,
		Agents:   appCtx.Agents,
		Realtime: appCtx.Realtime,
	}
	path, handler = agencyv1connect.NewTerminalServiceHandler(terminalService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	realtimeService := &realtimegrpc.Server{
		Bus:  appCtx.Realtime,
		Task: taskService.ProtoTask,
	}
	path, handler = agencyv1connect.NewRealtimeServiceHandler(realtimeService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	desktopService := &desktopgrpc.DesktopServer{Hub: appCtx.Hub}
	path, handler = agencyv1connect.NewDesktopServiceHandler(desktopService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	notificationService := &notificationsgrpc.NotificationServer{
		Notifications: appCtx.Notifications,
	}
	path, handler = agencyv1connect.NewNotificationServiceHandler(
		notificationService,
		handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	webhookService := &webhookgrpc.WebhookServer{Store: appCtx.Store, Notifier: appCtx.Webhooks}
	path, handler = agencyv1connect.NewWebhookServiceHandler(webhookService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	pluginService := &pluginsgrpc.Server{
		Plugins:  appCtx.Plugins,
		Projects: appCtx.Projects,
		Agents:   appCtx.Agents,
		Accounts: appCtx.Accounts,
	}
	path, handler = agencyv1connect.NewPluginServiceHandler(pluginService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	tunnelService := &tunnelServer{appCtx: appCtx}
	path, handler = tunnelv1connect.NewTunnelServiceHandler(tunnelService, handlerOptions()...)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))
	mux.Handle(sharedtunnel.WSPath, http.HandlerFunc(tunnelService.WS))
}
