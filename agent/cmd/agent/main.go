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
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"cherrypi.vladfaust.com/agent/cmd/agent/appcontext"
	agentauth "cherrypi.vladfaust.com/agent/internal/auth"
	"cherrypi.vladfaust.com/agent/internal/buildinfo"
	"cherrypi.vladfaust.com/agent/internal/events"
	"cherrypi.vladfaust.com/agent/internal/health"
	"cherrypi.vladfaust.com/agent/internal/logging"
	"cherrypi.vladfaust.com/agent/internal/tunnel"
	sharedprotocol "cherrypi.vladfaust.com/shared/protocol"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	agencyURL := flag.String(
		"agency-url",
		"",
		"CherryPi Agency URL to tunnel to (overrides CHERRYPI_AGENCY_URL)",
	)
	agentToken := flag.String(
		"token",
		"",
		"Agent personal access token, pat_… (overrides CHERRYPI_AGENT_TOKEN)",
	)
	display := flag.String(
		"display",
		"",
		"X display the desktop bridge attaches to (overrides DISPLAY)",
	)
	xauthority := flag.String(
		"xauthority",
		"",
		"X authority file for --display (overrides XAUTHORITY)",
	)
	flag.Parse()

	config := appcontext.ParseConfig()
	if *agencyURL != "" {
		config.AgencyURL = *agencyURL
	}
	if *agentToken != "" {
		config.AgentToken = strings.TrimSpace(*agentToken)
	}
	if *display != "" {
		config.Display = *display
	}
	if *xauthority != "" {
		config.XAuthority = *xauthority
	}

	if config.AgencyURL == "" {
		log.Fatal("No agency configured: pass --agency-url (or set CHERRYPI_AGENCY_URL)")
	}

	eventHub := events.New()
	logging.Setup(logging.Config{
		Format:       config.LogFormat,
		Level:        config.LogLevel,
		Sink:         eventHub.Publish,
		File:         config.LogFile,
		FileMaxBytes: config.LogFileMaxBytes,
		FileKeep:     config.LogFiles,
	})

	log.Printf("CherryPi agent %s starting", buildinfo.Version)

	envToken := config.AgentToken
	if err := appcontext.ResolveAgentToken(&config); err != nil {
		log.Fatalf("No agent token: %v", err)
	}

	appCtx := appcontext.NewAppContext(config, eventHub)

	healthCtx, healthCancel := context.WithCancel(context.Background())
	defer healthCancel()
	collector := health.New(health.Options{
		HostProc: config.HealthHostProc,
		HostSys:  config.HealthHostSys,
		HostRoot: config.HealthHostRoot,
		Disks:    config.HealthDisks,
	})
	sampler := health.NewSampler(collector, config.HealthInterval, eventHub.Publish)
	go sampler.Run(healthCtx)

	mux := http.NewServeMux()

	serveApi(mux)
	serveEvents(mux, appCtx)
	serveGrpc(mux, appCtx, sampler, config.HealthInterval)
	serveTerminals(mux, appCtx)
	serveDesktop(mux, appCtx)

	tunnelHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := r.Header.Get("X-CherryPi-User"); user != "" {
			r = r.WithContext(agentauth.WithUsername(r.Context(), user))
		}
		mux.ServeHTTP(w, r)
	})

	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
	defer tunnelCancel()
	client := tunnel.New(tunnel.Config{
		AgencyURL: appCtx.Config.AgencyURL,
		Transport: appCtx.Config.TunnelTransport,
		Token: func() string {

			return agentauth.LoadToken(envToken)
		},
		Handler: tunnelHandler,

		BuildVersion:    buildinfo.Version,
		ProtocolVersion: sharedprotocol.Version,
	})
	go client.Run(tunnelCtx)
	log.Printf("Tunneling to agency %s", appCtx.Config.AgencyURL)

	if err := appCtx.Sessions.Recover(context.Background()); err != nil {
		log.Printf("Session recovery failed: %v", err)
	}

	if err := appCtx.Terminals.Recover(context.Background()); err != nil {
		log.Printf("Terminal recovery failed: %v", err)
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down: waiting for runs to finish (Ctrl+C again to force)")

	drained := make(chan struct{})
	go func() {
		appCtx.Sessions.Shutdown(context.Background())
		close(drained)
	}()

	forced := false
	select {
	case <-drained:
	case <-sig:
		forced = true
	}

	// The guard is armed before the first line that reports the drain and

	stopGuard := guardShutdown(
		sig,
		shutdownTeardownTimeout,
		shutdownNote,
		dumpGoroutines,
		os.Exit,
	)

	if forced {
		log.Println("Forced shutdown")
		appCtx.Sessions.Kill()
	} else {
		log.Println("All runs finished")
	}

	log.Println("Closing terminals")
	appCtx.Terminals.CloseAll()
	log.Println("Terminals closed")
	appCtx.Desktops.Stop()
	log.Println("Desktop bridges stopped")

	// same sinks that can block, so the final line must stay inside the

	finishShutdown(stopGuard, func(message string) { log.Println(message) })
}

func serveTerminals(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.Handle("/api/terminal/", appCtx.Terminals.WSHandler())
}

func serveDesktop(mux *http.ServeMux, appCtx *appcontext.AppContext) {
	mux.Handle("/api/desktop/", appCtx.Desktops.WSHandler())
}

func serveApi(mux *http.ServeMux) {
	mux.HandleFunc("/api/rest/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `OK`)
	})
}
