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
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	"cherrypi.vladfaust.com/agency/cmd/agency/buildinfo"
	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/agenttools"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/processlog"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	"cherrypi.vladfaust.com/agency/internal/teams"
	connectcors "connectrpc.com/cors"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	_ = godotenv.Load()

	config := appcontext.ParseConfig()

	processLog, err := processlog.Setup(processlog.Config{
		Dir:     filepath.Join(config.Dir, "logs"),
		Keep:    config.LogFiles,
		Version: buildinfo.Version,
	})
	if err != nil {
		log.Printf("Process log unavailable: %v", err)
	}
	if processLog != nil {
		defer func() { _ = processLog.Close() }()
	}

	log.Printf("CherryPi Agency %s starting", buildinfo.Version)

	appCtx := appcontext.NewAppContext(config, processLog)

	if count, err := appCtx.Accounts.CountUsers(context.Background()); err == nil && count == 0 {
		log.Print("No user accounts configured: the agency is un-authenticate-able")
	}

	mux := http.NewServeMux()

	serveAPI(mux)
	serveGrpc(mux, appCtx)
	serveAccounts(mux, appCtx)
	serveTeams(mux, appCtx)
	serveProjectAccess(mux, appCtx)
	serveAgentOwners(mux, appCtx)
	serveEvents(mux, appCtx)
	serveAttachments(mux, appCtx)
	serveAgentTools(mux, appCtx)
	serveTaskHistory(mux, appCtx)
	serveTerminals(mux, appCtx)
	serveRealtime(mux, appCtx)
	serveProcessLogs(mux, appCtx)

	if devBuild && appCtx.Config.FrontendProxyUrl != nil {
		serveFrontendProxy(mux, appCtx.Config.FrontendProxyUrl)
	} else {
		serveEmbeddedFrontend(mux)
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         7200,
	}).Handler(withAuth(appCtx.Accounts, mux))

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	if appCtx.Config.H2C {
		protocols.SetUnencryptedHTTP2(true)
	}

	server := &http.Server{
		Addr:      ":" + strconv.Itoa(int(appCtx.Config.Port)),
		Handler:   corsHandler,
		Protocols: protocols,
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()

	log.Printf("CherryPi Agency listening on http://localhost:%d", appCtx.Config.Port)

	select {
	case <-sig:
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
		return
	}

	log.Println("Shutting down")
	_ = server.Shutdown(context.Background())
	_ = server.Close()
}

func serveAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/rest/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `OK`)
	})
}

func serveAccounts(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	appCtx.Accounts.SetBaseURL(appCtx.Config.PublicURL)
	handler := &accounts.Handler{
		Service:     appCtx.Accounts,
		Attachments: appCtx.Attachments,
		Teams:       appCtx.Teams,
		Projects:    appCtx.Projects,
		Agents:      appCtx.Agents,
		Options: accounts.HandlerOptions{
			BaseURL:                                appCtx.Config.PublicURL,
			ExposeMagicLinks:                       appCtx.Config.ExposeMagicLinks,
			InsecureFirstRegisteredUserBecomeAdmin: appCtx.Config.InsecureFirstRegisteredUserBecomeAdmin,
		},
	}
	mux.Handle(accounts.AuthPrefix, handler)
	mux.Handle(accounts.AuthPrefix+"/", handler)
	mux.Handle(accounts.AccountPrefix, handler)
	mux.Handle(accounts.AccountPrefix+"/", handler)
}

func serveTeams(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &teams.Handler{
		Service:  appCtx.Teams,
		Projects: teamProjects{service: appCtx.Projects},
		Governance: func(ctx context.Context) (teams.Governance, error) {
			settings, err := appCtx.Accounts.GetAuthSettings(ctx)
			if err != nil {
				return teams.Governance{}, err
			}
			return teams.Governance{
				MembersCanCreateTeams: settings.MembersCanCreateTeams,
				TeamDeletionAdminOnly: settings.TeamDeletionAdminOnly,
			}, nil
		},
	}
	mux.Handle(teams.Prefix, handler)
	mux.Handle(teams.Prefix+"/", handler)
}

func serveProjectAccess(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &projects.Handler{
		Service: appCtx.Projects,
		Governance: func(ctx context.Context) (projects.Governance, error) {
			settings, err := appCtx.Accounts.GetAuthSettings(ctx)
			if err != nil {
				return projects.Governance{}, err
			}
			return projects.Governance{
				ProjectDeletionAdminOnly: settings.ProjectDeletionAdminOnly,
			}, nil
		},
	}
	mux.Handle(projects.AccessPrefix, handler)
	mux.Handle(projects.AccessPrefix+"/", handler)
}

func serveAgentOwners(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &agents.Handler{
		Registry: appCtx.Agents,
		Teams:    appCtx.Teams,
		Audit:    appCtx.Audit,
	}
	mux.Handle(agents.Prefix, handler)
	mux.Handle(agents.Prefix+"/", handler)
}

type teamProjects struct{ service *projects.Service }

func (a teamProjects) TeamProjects(
	ctx context.Context,
	teamID int64,
) ([]teams.TeamProjectLink, error) {
	list, err := a.service.TeamProjects(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]teams.TeamProjectLink, len(list))
	for i, project := range list {
		out[i] = teams.TeamProjectLink{
			ProjectID:   project.ProjectID,
			ProjectName: project.ProjectName,
			ProjectSlug: project.ProjectSlug,
			Role:        project.Role,
		}
	}
	return out, nil
}

func serveEvents(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &audit.Handler{Service: appCtx.Audit}
	mux.Handle(audit.Prefix, handler)
	mux.Handle(audit.Prefix+"/", handler)
}

func serveAttachments(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := withAttachmentAuth(
		appCtx.Accounts, appCtx.Agents,
		&attachments.Handler{Service: appCtx.Attachments},
	)
	mux.Handle(attachments.UploadPath, handler)
	mux.Handle(attachments.UploadPath+"/", handler)
}

func serveTerminals(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.Handle("/api/agents/", appCtx.Hub.WSHandler())
}

func serveAgentTools(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := withAgentToolsAuth(
		appCtx.Accounts,
		appCtx.Agents,
		&agenttools.Handler{
			Tasks:       appCtx.Tasks,
			Projects:    appCtx.Projects,
			Agents:      appCtx.Agents,
			Accounts:    appCtx.Accounts,
			Logs:        appCtx.AgentLogs,
			ProcessLogs: appCtx.ProcessLogs,
		},
	)
	mux.Handle(agenttools.Prefix, handler)
	mux.Handle(agenttools.Prefix+"/", handler)
}

func serveTaskHistory(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := withAgentAuth(
		appCtx.Accounts,
		appCtx.Agents,
		&taskhistory.Handler{Tasks: appCtx.Tasks, Agents: appCtx.Agents},
	)
	mux.Handle(taskhistory.Prefix, handler)
	mux.Handle(taskhistory.Prefix+"/", handler)
}

func serveRealtime(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.Handle("/api/rest/notifications/ws", appCtx.Notifications.WSHandler())
}

func serveProcessLogs(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &processlog.Handler{Log: appCtx.ProcessLogs}
	mux.Handle(processlog.Prefix, handler)
	mux.Handle(processlog.Prefix+"/", handler)
}

func serveFrontendProxy(mux *http.ServeMux, target *url.URL) {
	proxy := httputil.NewSingleHostReverseProxy(target)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	log.Printf("Proxying / requests to %s", target)
}

func serveEmbeddedFrontend(mux *http.ServeMux) {
	frontendFs, err := frontendFS()
	if err != nil {
		if !devBuild {
			log.Fatalf("Embedded frontend unavailable: %v", err)
		}
		mux.HandleFunc("/", http.NotFound)
		return
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := frontendFs.Open(strings.TrimPrefix(path.Clean(r.URL.Path), "/"))
		if err == nil {
			defer func() { _ = file.Close() }()
		}

		if os.IsNotExist(err) {
			r.URL.Path = "/"
		}

		http.FileServer(http.FS(frontendFs)).ServeHTTP(w, r)
	})
}
