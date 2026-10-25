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
	"net/http"

	"cherrypi.vladfaust.com/agent/cmd/agent/appcontext"
	dashboardgrpc "cherrypi.vladfaust.com/agent/cmd/agent/grpc/dashboard"
	desktopgrpc "cherrypi.vladfaust.com/agent/cmd/agent/grpc/desktop"
	settingsgrpc "cherrypi.vladfaust.com/agent/cmd/agent/grpc/settings"
	taskgrpc "cherrypi.vladfaust.com/agent/cmd/agent/grpc/task"
	terminalgrpc "cherrypi.vladfaust.com/agent/cmd/agent/grpc/terminal"
	"cherrypi.vladfaust.com/shared/gen/buf/agent/v1/agentv1connect"
)

func serveGrpc(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	taskService := &taskgrpc.TaskServer{
		Tasks: appCtx.Sessions,
	}

	path, handler := agentv1connect.NewTaskServiceHandler(taskService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	terminalService := &terminalgrpc.TerminalServer{
		Terminals: appCtx.Terminals,
	}

	path, handler = agentv1connect.NewTerminalServiceHandler(terminalService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	desktopService := &desktopgrpc.DesktopServer{
		Desktops: appCtx.Desktops,
	}

	path, handler = agentv1connect.NewDesktopServiceHandler(desktopService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	dashboardService := &dashboardgrpc.DashboardServer{
		Store: appCtx.Store,
		Tasks: appCtx.Sessions,
	}

	path, handler = agentv1connect.NewDashboardServiceHandler(dashboardService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))

	settingsService := &settingsgrpc.SettingsServer{Store: appCtx.Store}

	path, handler = agentv1connect.NewSettingsServiceHandler(settingsService)
	mux.Handle("/api/grpc"+path, http.StripPrefix("/api/grpc", handler))
}
