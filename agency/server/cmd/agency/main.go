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
	"strconv"
	"strings"
	"syscall"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/agenttools"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	"cherrypi.vladfaust.com/agency/internal/teams"
	connectcors "connectrpc.com/cors"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

// The agency is the central web app: one SPA, one API, and (from Phase C) the
// tunnel hub that routes to connected agents.
func main() {
	_ = godotenv.Load()

	appCtx := appcontext.NewAppContext()

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
	protocols.SetUnencryptedHTTP2(true)

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

// serveAccounts mounts the account/auth REST surface. The public login,
// magic-link and registration routes live under accounts.AuthPrefix and are
// reachable unauthenticated; the account routes are protected by withAuth.
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

// serveTeams mounts the team REST surface. Its routes require an
// authenticated identity (withAuth), and the two governance toggles come from
// the account settings.
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

// serveProjectAccess mounts the project access REST surface (phase 5):
// members, team attachments and invitations, and the project log. Its routes
// require an authenticated identity (withAuth); the owner/member/viewer policy
// lives in the projects service, and the project-deletion toggle comes from
// the account settings.
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

// serveAgentOwners mounts the agent-ownership REST surface: the dashboard's
// owner management for an agent. It is authenticated by withAuth and gated
// per-agent (owner or admin) inside the handler.
func serveAgentOwners(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &agents.Handler{
		Registry: appCtx.Agents,
		Teams:    appCtx.Teams,
		Audit:    appCtx.Audit,
	}
	mux.Handle(agents.Prefix, handler)
	mux.Handle(agents.Prefix+"/", handler)
}

// serveEvents mounts the admin Events explorer (phase 7): a filtered UNION of
// the domain logs plus CSV/JSON export and the retention setting. It is
// authenticated by withAuth and administrator-only inside the handler.
// teamProjects adapts the projects service to the teams package's local
// TeamProjectLink type, keeping internal/projects out of internal/teams.
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

// serveAttachments mounts the attachment REST surface behind its own
// authentication: a user session/PAT/Basic or an agent personal access token
// (or legacy tunnel credential). withAuth skips these prefixes so the agent and
// the bundled upload tools can reach them.
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

// serveAgentTools mounts the agent-tool REST surface behind its own auth: a
// running Pi task's bundled extension calls it with the agent's personal
// access token (or the agent's legacy tunnel credential); the same surface
// carries the agent's PAT bootstrap/rotation route.
func serveAgentTools(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := withAgentToolsAuth(
		appCtx.Accounts,
		appCtx.Agents,
		&agenttools.Handler{
			Tasks:    appCtx.Tasks,
			Projects: appCtx.Projects,
			Agents:   appCtx.Agents,
			Accounts: appCtx.Accounts,
		},
	)
	mux.Handle(agenttools.Prefix, handler)
	mux.Handle(agenttools.Prefix+"/", handler)
}

// serveTaskHistory mounts the agent-authenticated session-history sync: an
// agent pushes newly written Pi session entries and pulls them back to
// reconstruct a resumable session file after being replaced.
func serveTaskHistory(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := withAgentAuth(
		appCtx.Accounts,
		appCtx.Agents,
		&taskhistory.Handler{Tasks: appCtx.Tasks, Agents: appCtx.Agents},
	)
	mux.Handle(taskhistory.Prefix, handler)
	mux.Handle(taskhistory.Prefix+"/", handler)
}

// serveRealtime mounts the browser-facing attention stream: new inbox entries
// are pushed to authenticated clients over a WebSocket instead of being polled.
func serveRealtime(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.Handle("/api/rest/notifications/ws", appCtx.Notifications.WSHandler())
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
