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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agent/internal/auth"
	"cherrypi.vladfaust.com/agent/internal/desktop"
	"cherrypi.vladfaust.com/agent/internal/events"
	"cherrypi.vladfaust.com/agent/internal/session"
	"cherrypi.vladfaust.com/agent/internal/store"
	"cherrypi.vladfaust.com/agent/internal/terminal"
)

type AppContext struct {
	Config    Config
	Store     store.Store
	Sessions  *session.Manager
	Terminals *terminal.Manager
	Desktops  *desktop.Manager
	Events    *events.Hub
}

func newAttachmentFetcher(config Config) func(context.Context, string) (io.ReadCloser, error) {
	if config.AgencyURL == "" || (config.AgentToken == "" && config.AgentUsername == "") {
		return nil
	}

	base := strings.TrimRight(config.AgencyURL, "/")
	client := &http.Client{}

	return func(ctx context.Context, url string) (io.ReadCloser, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+url, nil)
		if err != nil {
			return nil, err
		}
		setAuth(req, config)

		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			_ = res.Body.Close()
			return nil, fmt.Errorf("attachment fetch: %s", res.Status)
		}
		return res.Body, nil
	}
}

func setAuth(req *http.Request, config Config) {
	token := config.AgentToken
	if !config.AgentTokenFixed {
		if rotated := auth.LoadToken("", config.AgentTokenFile); rotated != "" {
			token = rotated
		}
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return
	}
	req.SetBasicAuth(config.AgentUsername, config.AgentPassword)
}

func ResolveAgentToken(ctx context.Context, config *Config) error {
	if token := auth.LoadToken(config.AgentToken, config.AgentTokenFile); token != "" {
		config.AgentToken = token
		if err := auth.SaveToken(config.AgentTokenFile, token); err != nil {
			log.Printf("Cache agent token: %v", err)
		}
		return nil
	}

	if config.AgencyURL == "" || config.AgentUsername == "" {
		return nil
	}

	token, err := bootstrapAgentToken(ctx, *config)
	if err != nil {
		return err
	}
	config.AgentToken = token
	if err := auth.SaveToken(config.AgentTokenFile, token); err != nil {
		log.Printf("Cache agent token: %v", err)
	}
	return nil
}

func bootstrapAgentToken(ctx context.Context, config Config) (string, error) {
	endpoint := strings.TrimRight(config.AgencyURL, "/") + "/api/rest/agent-tools/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(config.AgentUsername, config.AgentPassword)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent token bootstrap: %s", res.Status)
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Token == "" {
		return "", fmt.Errorf("agent token bootstrap: empty token")
	}
	return payload.Token, nil
}

func newTaskIDResolver(
	config Config,
) func(context.Context, string, int64) (string, error) {
	if config.AgencyURL == "" || (config.AgentToken == "" && config.AgentUsername == "") {
		return nil
	}

	base := strings.TrimRight(config.AgencyURL, "/")
	client := &http.Client{}

	return func(ctx context.Context, projectID string, seq int64) (string, error) {
		endpoint := base + "/api/rest/agent-tools/tasks/resolve?project_id=" +
			url.QueryEscape(projectID) + "&seq=" + strconv.FormatInt(seq, 10)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return "", err
		}
		setAuth(req, config)

		res, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("resolve task: %s", res.Status)
		}

		var payload struct {
			TaskID string `json:"task_id"`
		}
		if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
			return "", err
		}
		return payload.TaskID, nil
	}
}

func newTaskHistoryFetcher(
	config Config,
) func(context.Context, string, int64) ([]session.SessionEntry, error) {
	if config.AgencyURL == "" || (config.AgentToken == "" && config.AgentUsername == "") {
		return nil
	}

	base := strings.TrimRight(config.AgencyURL, "/")
	client := &http.Client{}

	return func(
		ctx context.Context,
		projectID string,
		seq int64,
	) ([]session.SessionEntry, error) {
		endpoint := base + "/api/rest/task-history?project_id=" +
			url.QueryEscape(projectID) + "&seq=" + strconv.FormatInt(seq, 10)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		setAuth(req, config)

		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode == http.StatusNotFound {
			return nil, session.ErrTaskHistoryNotFound
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("task history fetch: %s", res.Status)
		}

		var payload struct {
			Entries []session.SessionEntry `json:"entries"`
		}
		if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
			return nil, err
		}
		return payload.Entries, nil
	}
}

func newTaskHistoryPusher(
	config Config,
) func(context.Context, string, int64, []session.SessionEntry) error {
	if config.AgencyURL == "" || (config.AgentToken == "" && config.AgentUsername == "") {
		return nil
	}

	base := strings.TrimRight(config.AgencyURL, "/")
	client := &http.Client{}

	return func(
		ctx context.Context,
		projectID string,
		seq int64,
		entries []session.SessionEntry,
	) error {
		body, err := json.Marshal(map[string]any{
			"project_id": projectID,
			"seq":        seq,
			"entries":    entries,
		})
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			base+"/api/rest/task-history",
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		setAuth(req, config)

		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode == http.StatusNotFound {
			return session.ErrTaskHistoryNotFound
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("task history push: %s", res.Status)
		}
		return nil
	}
}

func NewAppContext(config Config, eventHub *events.Hub) *AppContext {

	if err := os.MkdirAll(config.Dir, 0o755); err != nil {
		log.Fatalf("Failed to create CHERRYPI_DIR %s: %v", config.Dir, err)
	}

	st := store.NewStore(config.DatabaseDSN, nil)

	extensionPaths, err := session.InstallExtensions(filepath.Join(config.Dir, "extensions"))
	if err != nil {
		log.Fatalf("Failed to install Pi extensions: %v", err)
	}
	piArgs := config.PiArgs
	for _, path := range extensionPaths {
		piArgs = append(piArgs, "--extension", path)
	}

	sessions := session.NewManager(session.Config{
		PiBin:            config.PiBin,
		PiSessionDir:     config.PiSessionDir,
		AgentDir:         config.PiAgentDir,
		PiArgs:           piArgs,
		WorkspaceRoot:    config.WorkspaceRoot,
		Notify:           eventHub.Publish,
		FetchAttachment:  newAttachmentFetcher(config),
		AgencyURL:        config.AgencyURL,
		AgentTokenFile:   config.AgentTokenFile,
		ResolveTaskID:    newTaskIDResolver(config),
		FetchTaskHistory: newTaskHistoryFetcher(config),
		PushTaskHistory:  newTaskHistoryPusher(config),
	}, st)

	terminals := terminal.NewManager(terminal.Config{
		Shell: config.Shell,
	}, st)

	desktops := desktop.NewManager(desktop.Config{
		Display:    config.Display,
		XAUTHORITY: config.XAuthority,
	})

	return &AppContext{
		Config:    config,
		Store:     st,
		Sessions:  sessions,
		Terminals: terminals,
		Desktops:  desktops,
		Events:    eventHub,
	}
}
