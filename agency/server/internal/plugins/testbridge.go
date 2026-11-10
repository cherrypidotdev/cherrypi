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

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/tasks"

	lua "github.com/yuin/gopher-lua"
)

type TestBridge struct {
	mu sync.Mutex

	pluginID string
	org      map[string]any

	projects        map[string]projects.Project
	projectRoles    map[string]map[ids.ActorID]string
	projectSettings map[string]ProjectSettings
	projectTags     map[string][]tags.Tag
	agents          map[ids.ActorID]AgentInfo
	users           map[ids.ActorID]accounts.Actor
	tasks           map[string]tasks.Task
	history         map[string][]tasks.SessionHistoryEntry
	settings        map[string]map[string]Settings
	taskPlugins     map[string]TaskPlugin
	teams           map[int64]map[string]any
	logs            []string
	created         []tasks.CreateParams
	createdIDs      []string
	httpCalls       []map[string]any
	httpFixtures    []httpFixture
	messages        []testMessage
	closed          []testClose
	badges          map[string][]TaskBadge
	holds           map[string]map[string]tasks.TaskHold
	releases        []testRelease
	nextTask        int
}

type testMessage struct {
	TaskID string
	Text   string
}

type testClose struct {
	TaskID  string
	Outcome tasks.Resolution
}

type testRelease struct {
	TaskID string
	Key    string
	Mode   string
	Prompt string
}

type httpFixture struct {
	method string
	path   string
	status int
	body   string
	header map[string]string
}

func NewTestBridge(pluginID string) *TestBridge {
	return &TestBridge{
		pluginID:        pluginID,
		org:             map[string]any{"id": "org", "name": "Test Agency"},
		projects:        map[string]projects.Project{},
		projectRoles:    map[string]map[ids.ActorID]string{},
		projectSettings: map[string]ProjectSettings{},
		projectTags:     map[string][]tags.Tag{},
		agents:          map[ids.ActorID]AgentInfo{},
		users:           map[ids.ActorID]accounts.Actor{},
		tasks:           map[string]tasks.Task{},
		history:         map[string][]tasks.SessionHistoryEntry{},
		settings:        map[string]map[string]Settings{},
		taskPlugins:     map[string]TaskPlugin{},
		teams:           map[int64]map[string]any{},
		badges:          map[string][]TaskBadge{},
		holds:           map[string]map[string]tasks.TaskHold{},
	}
}

func (b *TestBridge) SetOrg(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.org = map[string]any{"id": "org", "name": name}
}

func (b *TestBridge) SetProject(p projects.Project, tagNames ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.projects[p.ID] = p
	if b.projectRoles[p.ID] == nil {
		b.projectRoles[p.ID] = map[ids.ActorID]string{}
	}
	tagsOut := make([]tags.Tag, len(tagNames))
	for i, name := range tagNames {
		tagsOut[i] = tags.Tag{ID: name, Name: name, ProjectID: p.ID}
	}
	b.projectTags[p.ID] = tagsOut
	if _, ok := b.projectSettings[p.ID]; !ok {
		b.projectSettings[p.ID] = ProjectSettings{
			PluginID:          b.pluginID,
			ProjectID:         p.ID,
			AllowTaskCreation: true,
			Values:            map[string]any{},
		}
	}
}

func (b *TestBridge) SetProjectRole(projectID string, actorID ids.ActorID, role string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.projectRoles[projectID] == nil {
		b.projectRoles[projectID] = map[ids.ActorID]string{}
	}
	b.projectRoles[projectID][actorID] = role
}

func (b *TestBridge) SetProjectSettings(row ProjectSettings) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if row.Values == nil {
		row.Values = map[string]any{}
	}
	b.projectSettings[row.ProjectID] = row
}

func (b *TestBridge) SetAgent(agent AgentInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.agents[agent.ActorID] = agent
}

func (b *TestBridge) SetUser(actor accounts.Actor) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.users[actor.ID] = actor
}

func (b *TestBridge) settingsForScope(
	ctx context.Context,
	scope, target string,
) (map[string]any, error) {
	if scope == ScopeProject {
		row, err := b.ProjectSettings(ctx, target)
		if err != nil {
			return nil, err
		}
		return row.Values, nil
	}
	row, err := b.Settings(ctx, scope, target)
	if err != nil {
		return nil, err
	}
	return row.Values, nil
}

func (b *TestBridge) SetTask(t tasks.Task) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tasks[t.ID] = t
}

func (b *TestBridge) SetTaskHistory(taskID string, entries []tasks.SessionHistoryEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.history[taskID] = entries
}

func (b *TestBridge) SetSettings(row Settings) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if row.Values == nil {
		row.Values = map[string]any{}
	}
	if b.settings[row.Scope] == nil {
		b.settings[row.Scope] = map[string]Settings{}
	}
	b.settings[row.Scope][row.TargetID] = row
}

func (b *TestBridge) SetTeam(id int64, value map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.teams[id] = value
}

func (b *TestBridge) ExpectHTTP(
	method, path string,
	status int,
	body string,
	header map[string]string,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.httpFixtures = append(b.httpFixtures, httpFixture{
		method: strings.ToUpper(method),
		path:   path,
		status: status,
		body:   body,
		header: header,
	})
}

func (b *TestBridge) CreatedTasks() []tasks.CreateParams {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]tasks.CreateParams, len(b.created))
	copy(out, b.created)
	return out
}

func (b *TestBridge) CreatedTaskRows() []tasks.Task {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]tasks.Task, 0, len(b.createdIDs))
	for _, id := range b.createdIDs {
		if t, ok := b.tasks[id]; ok {
			out = append(out, t)
		}
	}
	return out
}

func (b *TestBridge) HTTPCalls() []map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]map[string]any, len(b.httpCalls))
	copy(out, b.httpCalls)
	return out
}

func (b *TestBridge) Logs() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.logs))
	copy(out, b.logs)
	return out
}

func (b *TestBridge) Tools(taskID string) []Tool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if row, ok := b.taskPlugins[taskID]; ok {
		return row.Tools
	}
	return nil
}

func (b *TestBridge) TaskMessages() []testMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]testMessage, len(b.messages))
	copy(out, b.messages)
	return out
}

func (b *TestBridge) ClosedTasks() []testClose {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]testClose, len(b.closed))
	copy(out, b.closed)
	return out
}

func (b *TestBridge) TaskBadges(taskID string) []TaskBadge {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]TaskBadge(nil), b.badges[taskID]...)
}

func (b *TestBridge) Holds(taskID string) []tasks.TaskHold {
	b.mu.Lock()
	defer b.mu.Unlock()
	return sortedHolds(b.holds[taskID])
}

func (b *TestBridge) Releases(taskID string) []testRelease {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]testRelease, 0, len(b.releases))
	for _, release := range b.releases {
		if taskID == "" || release.TaskID == taskID {
			out = append(out, release)
		}
	}
	return out
}

func sortedHolds(byKey map[string]tasks.TaskHold) []tasks.TaskHold {
	out := make([]tasks.TaskHold, 0, len(byKey))
	for _, hold := range byKey {
		out = append(out, hold)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func (b *TestBridge) Org(context.Context) (map[string]any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.org, nil
}

func (b *TestBridge) Project(_ context.Context, ref string) (projects.Project, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if p, ok := b.projects[ref]; ok {
		return p, nil
	}
	for _, p := range b.projects {
		if p.Slug == ref || p.Name == ref {
			return p, nil
		}
	}
	return projects.Project{}, fmt.Errorf("project %q not found", ref)
}

func (b *TestBridge) ProjectRole(
	_ context.Context,
	projectID string,
	actorID ids.ActorID,
) (string, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	role, ok := b.projectRoles[projectID][actorID]
	return role, ok, nil
}

func (b *TestBridge) ProjectSettings(
	_ context.Context,
	projectID string,
) (ProjectSettings, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if row, ok := b.projectSettings[projectID]; ok {
		return row, nil
	}
	return ProjectSettings{
		PluginID:          b.pluginID,
		ProjectID:         projectID,
		AllowTaskCreation: true,
		Values:            map[string]any{},
	}, nil
}

func (b *TestBridge) ProjectTags(_ context.Context, projectID string) ([]tags.Tag, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]tags.Tag(nil), b.projectTags[projectID]...), nil
}

func (b *TestBridge) Agent(_ context.Context, ref string) (AgentInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		if agent, ok := b.agents[ids.ActorID(id)]; ok {
			return agent, nil
		}
	}
	for _, agent := range b.agents {
		if agent.Username == ref || strconv.FormatInt(int64(agent.ID), 10) == ref {
			return agent, nil
		}
	}
	return AgentInfo{}, fmt.Errorf("agent %q not found", ref)
}

func (b *TestBridge) Agents(context.Context) ([]AgentInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]AgentInfo, 0, len(b.agents))
	for _, agent := range b.agents {
		out = append(out, agent)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Username < out[j].Username })
	return out, nil
}

func (b *TestBridge) User(_ context.Context, ref string) (accounts.Actor, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		if actor, ok := b.users[ids.ActorID(id)]; ok {
			return actor, nil
		}
	}
	for _, actor := range b.users {
		if actor.Username == ref {
			return actor, nil
		}
	}
	return accounts.Actor{}, fmt.Errorf("user %q not found", ref)
}

func (b *TestBridge) ActorProjects(
	_ context.Context,
	actorID ids.ActorID,
) ([]projects.Project, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []projects.Project
	for id, roles := range b.projectRoles {
		if _, ok := roles[actorID]; ok {
			if p, ok := b.projects[id]; ok {
				out = append(out, p)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (b *TestBridge) Task(_ context.Context, id string) (tasks.Task, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if t, ok := b.tasks[id]; ok {
		return t, nil
	}
	return tasks.Task{}, fmt.Errorf("task %q not found", id)
}

func (b *TestBridge) Tasks(_ context.Context, filter tasks.Filter) ([]tasks.Task, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []tasks.Task
	for _, t := range b.tasks {
		if filter.ProjectID != "" && t.ProjectID != filter.ProjectID {
			continue
		}
		if filter.State != "" && t.State != filter.State {
			continue
		}
		if filter.Search != "" &&
			!strings.Contains(strings.ToLower(t.Prompt), strings.ToLower(filter.Search)) {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if filter.Limit > 0 && int64(len(out)) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (b *TestBridge) TaskHistory(
	_ context.Context,
	id string,
) ([]tasks.SessionHistoryEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]tasks.SessionHistoryEntry(nil), b.history[id]...), nil
}

func (b *TestBridge) CreateTask(_ context.Context, p tasks.CreateParams) (tasks.Task, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.projects[p.ProjectID]; !ok {
		return tasks.Task{}, fmt.Errorf("project %q not found", p.ProjectID)
	}
	b.created = append(b.created, p)
	b.nextTask++
	id := fmt.Sprintf("TEST-%d", b.nextTask)
	b.createdIDs = append(b.createdIDs, id)
	t := tasks.Task{
		ID:              id,
		Seq:             int64(b.nextTask),
		Prompt:          p.Prompt,
		Name:            p.Name,
		ProjectID:       p.ProjectID,
		AssigneeActorID: p.AssigneeActorID,
		Priority:        p.Priority,
		Size:            p.Size,
		Interactive:     p.Interactive == nil || *p.Interactive,
		State:           tasks.StateBacklog,
	}
	for _, tagID := range p.TagIDs {
		for _, tag := range b.projectTags[p.ProjectID] {
			if tag.ID == tagID {
				t.Tags = append(t.Tags, tasks.Tag{
					ID: tag.ID, ProjectID: tag.ProjectID, Name: tag.Name, Color: tag.Color,
				})
			}
		}
	}
	b.tasks[id] = t
	return t, nil
}

func (b *TestBridge) QueueTask(_ context.Context, id string) (tasks.Task, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tasks[id]
	if !ok {
		return tasks.Task{}, fmt.Errorf("task %q not found", id)
	}
	t.State = tasks.StateQueued
	b.tasks[id] = t
	return t, nil
}

func (b *TestBridge) AttachTools(_ context.Context, taskID string, tools []Tool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.taskPlugins[taskID] = TaskPlugin{
		TaskID:   taskID,
		PluginID: b.pluginID,
		Tools:    append([]Tool(nil), tools...),
	}
	return nil
}

func (b *TestBridge) TaskPlugin(
	_ context.Context,
	taskID string,
) (TaskPlugin, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	row, ok := b.taskPlugins[taskID]
	return row, ok, nil
}

func (b *TestBridge) AddTaskMessage(_ context.Context, taskID, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, testMessage{TaskID: taskID, Text: text})
	if t, ok := b.tasks[taskID]; ok {
		switch t.State {
		case tasks.StateSettled, tasks.StateFailed,
			tasks.StateCompleted, tasks.StateCancelled:
			t.State = tasks.StateQueued
			b.tasks[taskID] = t
		}
	}
	return nil
}

func (b *TestBridge) CloseTask(
	_ context.Context,
	taskID string,
	outcome tasks.Resolution,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	if t.State == tasks.StateRunning || t.State == tasks.StateInput {
		return fmt.Errorf("the task is running; close it out once it settles")
	}
	b.closed = append(b.closed, testClose{TaskID: taskID, Outcome: outcome})
	if outcome == tasks.ResolutionCancelled {
		t.State = tasks.StateCancelled
	} else {
		t.State = tasks.StateCompleted
	}
	b.tasks[taskID] = t
	return nil
}

func (b *TestBridge) SetTaskBadges(
	_ context.Context,
	taskID string,
	badges []TaskBadge,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.badges[taskID] = append([]TaskBadge(nil), badges...)
	return nil
}

func (b *TestBridge) HoldTask(
	_ context.Context,
	taskID string,
	in HoldInput,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	t, ok := b.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	switch {
	case t.Archived:
		return fmt.Errorf("task %q is archived", taskID)
	case t.State == tasks.StateCompleted, t.State == tasks.StateCancelled:
		return fmt.Errorf("task %q is %s", taskID, t.State)
	}
	if b.holds[taskID] == nil {
		b.holds[taskID] = map[string]tasks.TaskHold{}
	}
	hold, exists := b.holds[taskID][in.Key]
	if !exists && len(b.holds[taskID]) >= tasks.MaxHoldsPerPlugin {
		return tasks.ErrHoldLimit
	}
	hold.TaskID = taskID
	hold.PluginID = b.pluginID
	hold.PluginName = b.pluginID
	hold.Key = in.Key
	hold.Reason = strings.TrimSpace(in.Reason)
	if !exists {
		hold.CreatedAt = time.Now()
	}
	b.holds[taskID][in.Key] = hold
	if t.State == tasks.StateSettled || t.State == tasks.StateFailed {
		t.State = tasks.StateWaiting
		b.tasks[taskID] = t
	}
	return nil
}

func (b *TestBridge) ReleaseTask(
	_ context.Context,
	taskID, key string,
	opts ReleaseOptions,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	mode := strings.ToLower(strings.TrimSpace(opts.Mode))
	if mode == "" {
		mode = string(tasks.ReleaseSettle)
	}
	b.releases = append(b.releases, testRelease{
		TaskID: taskID,
		Key:    key,
		Mode:   mode,
		Prompt: opts.Prompt,
	})
	if _, ok := b.holds[taskID][key]; !ok {
		return nil
	}
	delete(b.holds[taskID], key)
	if len(b.holds[taskID]) > 0 {
		return nil
	}
	delete(b.holds, taskID)
	t, ok := b.tasks[taskID]
	if !ok || t.State != tasks.StateWaiting {
		return nil
	}
	switch tasks.ReleaseMode(mode) {
	case tasks.ReleaseQueue:
		t.State = tasks.StateQueued
		t.PendingPrompt = strings.TrimSpace(opts.Prompt)
	case tasks.ReleaseComplete:
		t.State = tasks.StateCompleted
	default:
		t.State = tasks.StateSettled
	}
	b.tasks[taskID] = t
	return nil
}

func (b *TestBridge) TaskHolds(
	_ context.Context,
	taskID string,
) ([]tasks.TaskHold, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return sortedHolds(b.holds[taskID]), nil
}

func (b *TestBridge) Settings(
	_ context.Context,
	scope, target string,
) (Settings, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if row, ok := b.settings[scope][target]; ok {
		return row, nil
	}
	return Settings{
		PluginID: b.pluginID,
		Scope:    scope,
		TargetID: target,
		Values:   map[string]any{},
	}, nil
}

func (b *TestBridge) ListSettings(_ context.Context, scope string) ([]Settings, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Settings, 0, len(b.settings[scope]))
	for _, row := range b.settings[scope] {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TargetID < out[j].TargetID })
	return out, nil
}

func (b *TestBridge) Team(_ context.Context, id int64) (map[string]any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if team, ok := b.teams[id]; ok {
		return team, nil
	}
	return nil, fmt.Errorf("team %d not found", id)
}

func (b *TestBridge) Log(
	_ context.Context,
	_, level, message string,
	_ any,
	_ []LogSurface,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.logs = append(b.logs, level+": "+message)
}

func (b *TestBridge) HTTP(method, rawURL string, opts *lua.LTable) (map[string]any, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	body := ""
	if opts != nil {
		if raw := opts.RawGetString("body"); raw != lua.LNil {
			body = raw.String()
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.httpCalls = append(b.httpCalls, map[string]any{
		"method": method,
		"url":    rawURL,
		"path":   parsed.Path,
		"body":   body,
	})
	for i, fixture := range b.httpFixtures {
		if fixture.method != method || fixture.path != parsed.Path {
			continue
		}

		b.httpFixtures = append(b.httpFixtures[:i], b.httpFixtures[i+1:]...)
		headers := map[string]any{}
		for key, value := range fixture.header {
			headers[key] = value
		}
		return map[string]any{
			"status":  fixture.status,
			"headers": headers,
			"body":    fixture.body,
		}, nil
	}
	return nil, fmt.Errorf("no mock for %s %s", method, parsed.Path)
}

func JSONBody(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
