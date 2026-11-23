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
	"io/fs"
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
	"time"

	"cherrypi.vladfaust.com/agency/cmd/agency/appcontext"
	"cherrypi.vladfaust.com/agency/cmd/agency/buildinfo"
	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agentconfig"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/integrations"
	pluginhost "cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/processlog"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/sites"
	"cherrypi.vladfaust.com/agency/internal/speech"
	"cherrypi.vladfaust.com/agency/internal/taskhistory"
	"cherrypi.vladfaust.com/agency/internal/teams"
	"cherrypi.vladfaust.com/agency/internal/telemetry"
	"cherrypi.vladfaust.com/agency/internal/variables"
	connectcors "connectrpc.com/cors"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	var otelProvider *telemetry.Provider
	if config.OTelEnabled {
		provider, err := telemetry.Setup(context.Background(), telemetry.Config{
			ServiceName:    "cherrypi-agency",
			ServiceVersion: buildinfo.Version,
		})
		if err != nil {
			log.Printf("OpenTelemetry disabled: %v", err)
		} else {
			otelProvider = provider
			defer flushTelemetry(otelProvider)
			provider.LogStartup()
		}
	}

	appCtx := appcontext.NewAppContext(config, processLog)
	catalog, err := agentconfig.New(appCtx.Store, os.Getenv("CHERRYPI_SECRET_KEY"))
	if err != nil {
		log.Fatalf("Model catalog: %v", err)
	}
	appCtx.AgentConfig = catalog
	variablesService, err := variables.New(
		appCtx.Store, appCtx.Store, appCtx.Store, os.Getenv("CHERRYPI_SECRET_KEY"),
	)
	if err != nil {
		log.Fatalf("Variables: %v", err)
	}
	appCtx.Variables = variablesService
	catalog.SetVariables(variablesService)

	library, err := integrations.New(
		appCtx.Store,
		os.Getenv("CHERRYPI_SECRET_KEY"),
		integrationsAuthorizer(appCtx),
	)
	if err != nil {
		log.Fatalf("MCP & Skills Library: %v", err)
	}
	appCtx.Integrations = library

	catalog.Integrations = library

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
	serveSpeech(mux, appCtx)
	serveSites(mux, appCtx)
	serveAgentConfig(mux, appCtx)
	serveIntegrations(mux, appCtx)
	serveTaskHistory(mux, appCtx)
	serveTerminals(mux, appCtx)
	serveRealtime(mux, appCtx)
	serveProcessLogs(mux, appCtx)
	servePlugins(mux, appCtx)
	serveDev(mux, appCtx)

	if devBuild && appCtx.Config.FrontendProxyUrl != nil {
		serveFrontendProxy(mux, appCtx.Config.FrontendProxyUrl)
	} else {
		serveEmbeddedFrontend(mux)
	}

	if devBuild && appCtx.Config.DocsProxyUrl != nil {
		serveDocsProxy(mux, appCtx.Config.DocsProxyUrl)
	} else {
		serveEmbeddedDocs(mux)
	}

	apiHandler := withAuth(appCtx.Accounts, appCtx.Agents, mux)
	if telemetry.Enabled() {
		apiHandler = instrumentHTTP(mux, apiHandler)
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),

		AllowedHeaders: append(connectcors.AllowedHeaders(), "Authorization", "X-CSRF-Token"),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         7200,
	}).Handler(apiHandler)

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

			flushTelemetry(otelProvider)
			log.Fatal(err)
		}
		return
	}

	log.Println("Shutting down")
	_ = server.Shutdown(context.Background())
	_ = server.Close()
	drainWebhooks(appCtx)
}

const webhookDrainTimeout = 5 * time.Second

func drainWebhooks(appCtx *appcontext.AppContext) {
	if appCtx == nil || appCtx.Webhooks == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), webhookDrainTimeout)
	defer cancel()
	if err := appCtx.Webhooks.Wait(ctx); err != nil {
		log.Printf("Webhook deliveries still in flight at shutdown: %v", err)
	}
}

func flushTelemetry(provider *telemetry.Provider) {
	if provider == nil {
		return
	}

	ctx, cancel := telemetry.ShutdownContext()
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		log.Printf("OpenTelemetry shutdown: %v", err)
	}
}

func instrumentHTTP(mux *http.ServeMux, next http.Handler) http.Handler {
	instrumented := otelhttp.NewHandler(
		next,
		"http.server",
		otelhttp.WithFilter(tracedHTTPRequest),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return httpSpanName(r)
		}),
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		r.Pattern = httpRoutePattern(mux, r)
		instrumented.ServeHTTP(w, r)
	})
}

func tracedHTTPRequest(r *http.Request) bool {
	path := r.URL.Path

	if strings.HasPrefix(path, "/api/grpc/") || path == "/api/grpc" {
		return false
	}

	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		path == "/api/tunnel/ws" {
		return false
	}

	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/plugins/")
}

func httpRoutePattern(mux *http.ServeMux, r *http.Request) string {
	_, pattern := mux.Handler(r)
	if pattern == "/" {
		return ""
	}

	return pattern
}

func httpSpanName(r *http.Request) string {

	if r.Pattern == "" || r.Pattern == "/" {
		return r.Method + " <unmatched>"
	}

	if index := strings.IndexRune(r.Pattern, '/'); index > 0 &&
		strings.ContainsRune(r.Pattern[:index], ' ') {
		return r.Pattern
	}

	return r.Method + " " + r.Pattern
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
			ProjectID:                project.ProjectID,
			ProjectName:              project.ProjectName,
			ProjectSlug:              project.ProjectSlug,
			Role:                     project.Role,
			Inherited:                project.Inherited,
			InheritedFromProjectID:   project.InheritedFromProjectID,
			InheritedFromProjectName: project.InheritedFromProjectName,
			InheritedFromProjectSlug: project.InheritedFromProjectSlug,
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

func serveSites(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &sites.Handler{
		Service:           appCtx.Sites,
		Accounts:          appCtx.Accounts,
		AllowInsecureHTTP: appCtx.Config.SitesInsecureHTTP,
	}
	mux.Handle(sites.Prefix, handler)
	mux.Handle(sites.Prefix+"/", handler)
	content := http.HandlerFunc(handler.ServeContent)
	mux.Handle("GET /sites/{siteID}/content/", content)
	mux.Handle("GET /sites/{siteID}/content", content)
	mux.Handle("GET /sites/{siteID}/bridge.js", content)
}

func serveSpeech(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &speech.Handler{Service: appCtx.Speech}
	mux.Handle(speech.Prefix, handler)
	mux.Handle(speech.Prefix+"/", handler)
}

func serveTerminals(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.Handle("/api/agents/", appCtx.Hub.WSHandler())
}

func serveAgentConfig(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	service := appCtx.AgentConfig

	agent := func(ctx context.Context) string {
		id, ok := auth.IdentityFrom(ctx)
		if !ok || id.Kind != "agent" {
			return ""
		}
		return strconv.FormatInt(int64(id.ActorID), 10)
	}
	handler := &agentconfig.Handler{
		Service:      service,
		AgentActorID: agent,
		IsAdmin: func(r *http.Request) bool {
			id, _, ok := resolveIdentity(appCtx.Accounts, nil, r)
			return ok && id.IsAdmin()
		},
		TaskScope: taskScope(appCtx),
	}
	mux.Handle(agentconfig.BundlePath, withAgentAuth(appCtx.Accounts, appCtx.Agents, handler))
	mux.Handle(agentconfig.AdminPath, handler)
	mux.Handle(agentconfig.AdminPath+"/", handler)
	mux.Handle(
		agentconfig.ProxyPrefix+"/",
		withAgentAuth(
			appCtx.Accounts,
			appCtx.Agents,
			&agentconfig.Proxy{Service: service, TaskScope: taskScope(appCtx)},
		),
	)
}

func taskScope(appCtx *appcontext.AppContext) agentconfig.TaskScope {
	return func(
		ctx context.Context,
		agentActorID string,
		taskID string,
	) (string, []string, bool) {
		if taskID == "" || agentActorID == "" {
			return "", nil, false
		}
		task, err := appCtx.Tasks.Get(ctx, taskID)
		if err != nil {
			return "", nil, false
		}
		if strconv.FormatInt(int64(task.AssigneeActorID), 10) != agentActorID {
			return "", nil, false
		}
		teams := make([]string, 0, len(task.TeamIDs))
		for _, team := range task.TeamIDs {
			teams = append(teams, strconv.FormatInt(int64(team), 10))
		}
		return task.ProjectID, teams, true
	}
}

func serveIntegrations(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &integrations.Handler{
		Service: appCtx.Integrations,
		Identity: func(r *http.Request) (integrations.Identity, bool) {

			id, ok := auth.IdentityFrom(r.Context())
			if !ok {
				return integrations.Identity{}, false
			}
			return integrations.Identity{
				ActorID: int64(id.ActorID),
				Admin:   id.IsAdmin(),
			}, true
		},
		AgentActorID: func(ctx context.Context) string {
			id, ok := auth.IdentityFrom(ctx)
			if !ok || id.Kind != "agent" {
				return ""
			}
			return strconv.FormatInt(int64(id.ActorID), 10)
		},
		TaskForAgent: integrations.TaskScope(taskScope(appCtx)),
		TaskVisible: func(
			ctx context.Context,
			identity integrations.Identity,
			taskID string,
		) (bool, error) {
			task, err := appCtx.Tasks.Get(ctx, taskID)
			if err != nil {
				return false, nil
			}
			if identity.Admin || task.ProjectID == "" {
				return true, nil
			}
			role, ok, err := appCtx.Projects.EffectiveRole(
				ctx, task.ProjectID, ids.ActorID(identity.ActorID), false,
			)
			if err != nil {
				return false, err
			}
			return ok && role != "", nil
		},
		Groups: func(ctx context.Context, agentActorID string) ([]string, error) {
			actorID, err := strconv.ParseInt(agentActorID, 10, 64)
			if err != nil {
				return nil, nil
			}
			groups, err := appCtx.Store.ListAgentGroupIDsForAgentActor(ctx, actorID)
			if err != nil {
				return nil, err
			}
			out := make([]string, 0, len(groups))
			for _, group := range groups {
				out = append(out, strconv.FormatInt(group, 10))
			}
			return out, nil
		},
	}
	report := &integrations.Handler{Service: appCtx.Integrations}
	report.AgentActorID = handler.AgentActorID
	report.TaskForAgent = handler.TaskForAgent
	mux.Handle(integrations.AdminPath, handler)
	mux.Handle(integrations.AdminPath+"/", handler)
	mux.Handle(
		integrations.ReportPath,
		withAgentAuth(appCtx.Accounts, appCtx.Agents, report),
	)

	proxy := &integrations.Proxy{
		Service:      appCtx.Integrations,
		AgentActorID: handler.AgentActorID,
		TaskScope:    handler.TaskForAgent,
		Groups:       handler.Groups,
	}
	mux.Handle(
		integrations.ProxyPrefix+"/",
		withAgentAuth(appCtx.Accounts, appCtx.Agents, proxy),
	)
}

func integrationsAuthorizer(appCtx *appcontext.AppContext) *integrations.Authorizer {
	return &integrations.Authorizer{
		IsAdmin: func(ctx context.Context, actorID int64) bool {
			actor, err := appCtx.Accounts.GetActor(ctx, ids.ActorID(actorID))
			if err != nil {
				return false
			}
			return actor.IsAdmin()
		},
		ProjectRole: func(
			ctx context.Context,
			projectID string,
			actorID ids.ActorID,
			isAdmin bool,
		) (string, bool, error) {
			return appCtx.Projects.EffectiveRole(ctx, projectID, actorID, isAdmin)
		},
		TeamRole: func(
			ctx context.Context,
			teamID int64,
			actorID ids.ActorID,
		) (string, bool, error) {
			memberships, err := appCtx.Teams.MembershipsForActor(ctx, actorID)
			if err != nil {
				return "", false, err
			}
			for _, team := range memberships {
				if team.ID == teamID {
					return team.Role, true, nil
				}
			}
			return "", false, nil
		},
		GroupOwner: func(
			ctx context.Context,
			groupID int64,
			actorID ids.ActorID,
		) (bool, error) {
			return appCtx.Groups.IsOwner(ctx, groupID, actorID)
		},
		AgentOwner: func(
			ctx context.Context,
			agentActorID int64,
			actorID ids.ActorID,
		) (bool, error) {
			return appCtx.Agents.IsActorOwner(ctx, ids.ActorID(agentActorID), actorID)
		},
	}
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

func servePlugins(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	handler := &pluginhost.Handler{
		Service: appCtx.Plugins,
		Auth: func(r *http.Request) (pluginhost.Caller, bool) {
			identity, _, ok := resolveIdentity(
				appCtx.Accounts,
				resolvePAT(r.Context(), appCtx.Accounts, r),
				r,
			)
			if !ok {
				return pluginhost.Caller{}, false
			}
			return pluginhost.Caller{
				Kind:     identity.Kind,
				ActorID:  int64(identity.ActorID),
				Username: identity.Username,
				Admin:    identity.IsAdmin(),
			}, true
		},
	}
	mux.Handle("/plugins/", handler)

	icons := &pluginhost.IconHandler{Service: appCtx.Plugins}
	mux.Handle(pluginhost.IconRoutePrefix, icons)
	mux.Handle(pluginhost.IconRoutePrefix+"/", icons)
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

	mux.Handle("/", newFrontendHandler(frontendFs))
}

func newFrontendHandler(frontendFs fs.FS) http.Handler {
	etags := newFileETags()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		isImmutable := staticTree(name, "assets") || staticTree(name, "fonts")

		info, err := fs.Stat(frontendFs, name)
		if err != nil || info.IsDir() {
			if isImmutable {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
		}

		contentType := ""
		if path.Ext(name) == ".map" {
			contentType = "application/json; charset=utf-8"
		}

		cacheControl, immutable := staticCachePolicy(name)
		var etagStore *fileETags
		if !immutable {
			etagStore = etags
		}

		if !serveEmbeddedFile(w, r, frontendFs, name, contentType, cacheControl, etagStore) {
			http.NotFound(w, r)
		}
	})
}
