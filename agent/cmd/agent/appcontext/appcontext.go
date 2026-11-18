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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agent/internal/agency"
	"cherrypi.vladfaust.com/agent/internal/auth"
	"cherrypi.vladfaust.com/agent/internal/desktop"
	"cherrypi.vladfaust.com/agent/internal/events"
	"cherrypi.vladfaust.com/agent/internal/session"
	"cherrypi.vladfaust.com/agent/internal/store"
	"cherrypi.vladfaust.com/agent/internal/terminal"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
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
	if config.AgencyURL == "" || config.AgentToken == "" {
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
	if config.AgentToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.AgentToken)
	}
}

func ResolveAgentToken(config *Config) error {
	if token := auth.LoadToken(config.AgentToken); token != "" {
		config.AgentToken = token
		return nil
	}
	return errors.New(
		"no agent token: pass --token or set CHERRYPI_AGENT_TOKEN to a personal access token",
	)
}

func newTaskIDResolver(
	client *agency.Client,
) func(context.Context, string, int64) (string, error) {
	if client == nil || client.BaseURL == "" {
		return nil
	}
	return func(ctx context.Context, projectID string, seq int64) (string, error) {
		res, err := client.Tasks.ResolveTaskRef(
			ctx,
			&agencyv1.ResolveTaskRefRequest{ProjectId: projectID, Seq: seq},
		)
		if err != nil {
			return "", err
		}
		return res.GetTaskId(), nil
	}
}

func newTaskHistoryFetcher(
	config Config,
) func(context.Context, string, int64) ([]session.SessionEntry, error) {
	if config.AgencyURL == "" || config.AgentToken == "" {
		return nil
	}

	base := strings.TrimRight(config.AgencyURL, "/")

	client := &http.Client{Timeout: 60 * time.Second}

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
		if res.StatusCode != http.StatusOK {
			return nil, taskHistoryStatusError(res)
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

func taskHistoryStatusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = res.Status
	}
	return &session.TaskHistoryStatusError{
		Status:  res.StatusCode,
		Message: message,
	}
}

func newTaskSessionStateGetter(
	client *agency.Client,
) func(context.Context, string) (session.TaskSessionState, bool, error) {
	if client == nil || client.BaseURL == "" {
		return nil
	}
	return func(
		ctx context.Context,
		taskID string,
	) (session.TaskSessionState, bool, error) {
		res, err := client.Tasks.GetTaskSessionState(
			ctx,
			&agencyv1.GetTaskSessionStateRequest{Id: taskID},
		)
		if err != nil {
			return session.TaskSessionState{}, false, err
		}
		state := res.GetState()
		if state == nil {
			return session.TaskSessionState{}, false, nil
		}
		return session.TaskSessionState{
			Revision:      state.GetRevision(),
			ModelProvider: state.GetModelProvider(),
			ModelID:       state.GetModelId(),
			ThinkingLevel: state.GetThinkingLevel(),
			EntriesJSON:   state.GetEntriesJson(),
			LastEventSeq:  state.GetLastEventSeq(),
		}, res.GetFound(), nil
	}
}

func newTaskSessionStatePutter(
	client *agency.Client,
) func(context.Context, string, session.TaskSessionState, int64) (int64, error) {
	if client == nil || client.BaseURL == "" {
		return nil
	}
	return func(
		ctx context.Context,
		taskID string,
		state session.TaskSessionState,
		expectedRevision int64,
	) (int64, error) {
		res, err := client.Tasks.PutTaskSessionState(
			ctx,
			&agencyv1.PutTaskSessionStateRequest{
				Id:               taskID,
				ExpectedRevision: expectedRevision,
				ModelProvider:    state.ModelProvider,
				ModelId:          state.ModelID,
				ThinkingLevel:    state.ThinkingLevel,
				EntriesJson:      state.EntriesJSON,
			},
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeAborted {
				return 0, session.ErrTaskSessionStateConflict
			}
			return 0, err
		}
		return res.GetRevision(), nil
	}
}

func newTaskEventAppender(
	client *agency.Client,
) func(context.Context, string, []session.TaskEventRecord) error {
	if client == nil || client.BaseURL == "" {
		return nil
	}
	return func(
		ctx context.Context,
		taskID string,
		events []session.TaskEventRecord,
	) error {
		records := make([]*agencyv1.TaskEventRecord, len(events))
		for i, event := range events {
			records[i] = &agencyv1.TaskEventRecord{
				Seq:         event.Seq,
				At:          event.At,
				Kind:        event.Kind,
				PayloadJson: event.PayloadJSON,
			}
		}
		_, err := client.Tasks.AppendTaskEvents(
			ctx,
			&agencyv1.AppendTaskEventsRequest{Id: taskID, Events: records},
		)
		return err
	}
}

func NewAppContext(config Config, eventHub *events.Hub) *AppContext {

	if err := os.MkdirAll(config.Dir, 0o755); err != nil {
		log.Fatalf("Failed to create CHERRYPI_DIR %s: %v", config.Dir, err)
	}

	st := store.NewStore(config.DatabaseDSN, nil)

	agencyClient := agency.New(config.AgencyURL, func() string {
		return config.AgentToken
	}, nil)

	sessions := session.NewManager(session.Config{
		WorkspaceRoot:       config.WorkspaceRoot,
		Notify:              eventHub.Publish,
		FetchAttachment:     newAttachmentFetcher(config),
		AgencyURL:           config.AgencyURL,
		AgentToken:          config.AgentToken,
		Agency:              agencyClient,
		AgencyCatalog:       &agentCatalog{client: agencyClient},
		ResolveTaskID:       newTaskIDResolver(agencyClient),
		FetchTaskHistory:    newTaskHistoryFetcher(config),
		GetTaskSessionState: newTaskSessionStateGetter(agencyClient),
		PutTaskSessionState: newTaskSessionStatePutter(agencyClient),
		AppendTaskEvents:    newTaskEventAppender(agencyClient),
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

type agentCatalog struct {
	client *agency.Client
}

func (c *agentCatalog) Fetch(
	ctx context.Context,
	taskID, knownRevision string,
) (session.AgencyModels, bool, error) {
	if c.client == nil || c.client.BaseURL == "" {
		return session.AgencyModels{}, false, nil
	}
	path := "/api/rest/agent-config"
	query := url.Values{}
	if taskID != "" {
		query.Set("task", taskID)
	}
	if knownRevision != "" {
		query.Set("revision", knownRevision)
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	raw, notModified, err := c.client.RESTConditional(ctx, path)
	if err != nil {
		return session.AgencyModels{}, false, err
	}
	if notModified {
		return session.AgencyModels{}, false, nil
	}
	var models session.AgencyModels
	if err := json.Unmarshal(raw, &models); err != nil {
		return session.AgencyModels{}, false, err
	}
	return models, true, nil
}
