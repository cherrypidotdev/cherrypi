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
	"errors"
	"fmt"
	"log"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

type AgentInfo struct {
	ID          ids.AgentID
	ActorID     ids.ActorID
	Username    string
	Name        string
	Description string
	Role        string
	Status      string
	Version     string
}

type Bridge interface {
	Org(ctx context.Context) (map[string]any, error)
	Project(ctx context.Context, ref string) (projects.Project, error)
	ProjectRole(ctx context.Context, projectID string, actorID ids.ActorID) (string, bool, error)
	ProjectSettings(ctx context.Context, projectID string) (ProjectSettings, error)
	ProjectTags(ctx context.Context, projectID string) ([]tags.Tag, error)
	Agent(ctx context.Context, ref string) (AgentInfo, error)
	Agents(ctx context.Context) ([]AgentInfo, error)
	User(ctx context.Context, ref string) (accounts.Actor, error)
	ActorProjects(ctx context.Context, actorID ids.ActorID) ([]projects.Project, error)
	Task(ctx context.Context, id string) (tasks.Task, error)
	Tasks(ctx context.Context, filter tasks.Filter) ([]tasks.Task, error)
	TaskHistory(ctx context.Context, id string) ([]tasks.SessionHistoryEntry, error)
	CreateTask(ctx context.Context, p tasks.CreateParams) (tasks.Task, error)
	QueueTask(ctx context.Context, id string) (tasks.Task, error)
	AttachTools(ctx context.Context, taskID string, tools []Tool) error
	TaskPlugin(ctx context.Context, taskID string) (TaskPlugin, bool, error)

	AddTaskMessage(ctx context.Context, taskID, text string) error

	RespondTaskInput(ctx context.Context, taskID, requestID string, answer TaskInputAnswer) error

	CloseTask(ctx context.Context, taskID string, outcome tasks.Resolution) error

	SetTaskBadges(ctx context.Context, taskID string, badges []TaskBadge) error

	HoldTask(ctx context.Context, taskID string, hold HoldInput) error

	ReleaseTask(ctx context.Context, taskID, key string, opts ReleaseOptions) error

	TaskHolds(ctx context.Context, taskID string) ([]tasks.TaskHold, error)
	Settings(ctx context.Context, scope, target string) (Settings, error)
	ListSettings(ctx context.Context, scope string) ([]Settings, error)
	Team(ctx context.Context, id int64) (map[string]any, error)
	Log(ctx context.Context, invocation, level, message string, data any, surfaces []LogSurface)
}

type serviceBridge struct {
	svc      *Service
	pluginID string
}

func (b serviceBridge) Org(ctx context.Context) (map[string]any, error) {

	publicURL := b.svc.PublicURL()
	if b.svc.Settings == nil {
		return map[string]any{"id": "org", "name": "", "url": publicURL}, nil
	}
	cfg, err := b.svc.Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": "org", "name": cfg.AgencyName, "url": publicURL}, nil
}

func (b serviceBridge) Project(ctx context.Context, ref string) (projects.Project, error) {
	if b.svc.Projects == nil {
		return projects.Project{}, errors.New("projects are unavailable")
	}
	return b.svc.Projects.Get(ctx, ref)
}

func (b serviceBridge) ProjectRole(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
) (string, bool, error) {
	if b.svc.Projects == nil {
		return "", false, errors.New("projects are unavailable")
	}
	return b.svc.Projects.EffectiveRole(ctx, projectID, actorID, false)
}

func (b serviceBridge) ProjectSettings(
	ctx context.Context,
	projectID string,
) (ProjectSettings, error) {
	row, err := b.svc.Store.ProjectSettings(ctx, b.pluginID, projectID)
	if err != nil {
		return row, err
	}
	if p, err := b.svc.Get(ctx, b.pluginID); err == nil {
		row.Values = p.Manifest.ApplySettingsDefaults(ScopeProject, row.Values)
	}
	return row, nil
}

func (b serviceBridge) ProjectTags(
	ctx context.Context,
	projectID string,
) ([]tags.Tag, error) {
	if b.svc.Tags == nil {
		return nil, errors.New("tags are unavailable")
	}
	return b.svc.Tags.List(ctx, projectID, false)
}

func (b serviceBridge) Agent(ctx context.Context, ref string) (AgentInfo, error) {
	return resolveAgent(ctx, b.svc.Agents, ref)
}

func (b serviceBridge) Agents(ctx context.Context) ([]AgentInfo, error) {
	if b.svc.Agents == nil {
		return nil, errors.New("agents are unavailable")
	}
	list, err := b.svc.Agents.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgentInfo, len(list))
	for i, agent := range list {
		out[i] = toAgentInfo(agent)
	}
	return out, nil
}

func (b serviceBridge) User(ctx context.Context, ref string) (accounts.Actor, error) {
	if b.svc.Accounts == nil {
		return accounts.Actor{}, errors.New("users are unavailable")
	}
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return b.svc.Accounts.GetActor(ctx, ids.ActorID(id))
	}
	return b.svc.Accounts.GetUserByUsername(ctx, ref)
}

func (b serviceBridge) ActorProjects(
	ctx context.Context,
	actorID ids.ActorID,
) ([]projects.Project, error) {
	if b.svc.Projects == nil {
		return nil, errors.New("projects are unavailable")
	}
	return b.svc.Projects.ProjectsForActor(ctx, actorID)
}

func (b serviceBridge) Task(ctx context.Context, id string) (tasks.Task, error) {
	if b.svc.Tasks == nil {
		return tasks.Task{}, errors.New("tasks are unavailable")
	}
	return b.svc.Tasks.Get(ctx, id)
}

func (b serviceBridge) Tasks(
	ctx context.Context,
	filter tasks.Filter,
) ([]tasks.Task, error) {
	if b.svc.Tasks == nil {
		return nil, errors.New("tasks are unavailable")
	}
	return b.svc.Tasks.List(ctx, filter)
}

func (b serviceBridge) TaskHistory(
	ctx context.Context,
	id string,
) ([]tasks.SessionHistoryEntry, error) {
	if b.svc.Tasks == nil {
		return nil, errors.New("tasks are unavailable")
	}
	return b.svc.Tasks.SessionHistory(ctx, id)
}

func (b serviceBridge) CreateTask(
	ctx context.Context,
	p tasks.CreateParams,
) (tasks.Task, error) {
	if b.svc.Tasks == nil {
		return tasks.Task{}, errors.New("tasks are unavailable")
	}
	return b.svc.Tasks.Create(ctx, p)
}

func (b serviceBridge) QueueTask(ctx context.Context, id string) (tasks.Task, error) {
	if b.svc.Tasks == nil {
		return tasks.Task{}, errors.New("tasks are unavailable")
	}
	return b.svc.Tasks.Queue(ctx, id)
}

func (b serviceBridge) AttachTools(ctx context.Context, taskID string, tools []Tool) error {
	return b.svc.Store.SetTaskPlugin(ctx, taskID, b.pluginID, tools)
}

func (b serviceBridge) TaskPlugin(
	ctx context.Context,
	taskID string,
) (TaskPlugin, bool, error) {
	row, ok, err := b.svc.Store.TaskPlugin(ctx, taskID)
	return row, ok, err
}

func (b serviceBridge) AddTaskMessage(ctx context.Context, taskID, text string) error {
	return b.svc.AddTaskMessage(ctx, b.pluginID, taskID, text)
}

func (b serviceBridge) RespondTaskInput(
	ctx context.Context,
	taskID, requestID string,
	answer TaskInputAnswer,
) error {
	return b.svc.RespondTaskInput(ctx, b.pluginID, taskID, requestID, answer)
}

func (b serviceBridge) CloseTask(
	ctx context.Context,
	taskID string,
	outcome tasks.Resolution,
) error {
	return b.svc.CloseTask(ctx, b.pluginID, taskID, outcome)
}

func (b serviceBridge) SetTaskBadges(
	ctx context.Context,
	taskID string,
	badges []TaskBadge,
) error {
	return b.svc.SetTaskBadges(ctx, b.pluginID, taskID, badges)
}

func (b serviceBridge) HoldTask(
	ctx context.Context,
	taskID string,
	hold HoldInput,
) error {
	return b.svc.HoldTask(ctx, b.pluginID, taskID, hold)
}

func (b serviceBridge) ReleaseTask(
	ctx context.Context,
	taskID, key string,
	opts ReleaseOptions,
) error {
	return b.svc.ReleaseTask(ctx, b.pluginID, taskID, key, opts)
}

func (b serviceBridge) TaskHolds(
	ctx context.Context,
	taskID string,
) ([]tasks.TaskHold, error) {
	return b.svc.TaskHolds(ctx, taskID)
}

func (b serviceBridge) Settings(
	ctx context.Context,
	scope, target string,
) (Settings, error) {
	row, err := b.svc.Store.Settings(ctx, b.pluginID, scope, target)
	if err != nil {
		return row, err
	}
	if p, err := b.svc.Get(ctx, b.pluginID); err == nil {
		row.Values = p.Manifest.ApplySettingsDefaults(scope, row.Values)
	}
	return row, nil
}

func (b serviceBridge) ListSettings(
	ctx context.Context,
	scope string,
) ([]Settings, error) {
	rows, err := b.svc.Store.ListSettings(ctx, b.pluginID, scope)
	if err != nil {
		return nil, err
	}
	p, err := b.svc.Get(ctx, b.pluginID)
	if err != nil {
		return rows, nil
	}
	for i := range rows {
		rows[i].Values = p.Manifest.ApplySettingsDefaults(scope, rows[i].Values)
	}
	return rows, nil
}

func (b serviceBridge) Team(ctx context.Context, id int64) (map[string]any, error) {
	if b.svc.Teams == nil {
		return nil, errors.New("teams are unavailable")
	}
	team, err := b.svc.Teams.Get(ctx, id, 0, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          team.ID,
		"name":        team.Name,
		"description": team.Description,
	}, nil
}

func (b serviceBridge) Log(
	ctx context.Context,
	invocation, level, message string,
	data any,
	surfaces []LogSurface,
) {
	if err := b.svc.Store.LogRuntime(
		ctx,
		b.pluginID,
		invocation,
		level,
		message,
		data,
		surfaces,
	); err != nil {
		log.Printf("[plugins] runtime log %s: %v", b.pluginID, err)
	}
}

func toAgentInfo(agent agents.Agent) AgentInfo {
	status := "offline"
	if agent.Status == agents.StatusOnline {
		status = "online"
	}
	return AgentInfo{
		ID:          agent.ID,
		ActorID:     agent.ActorID,
		Username:    agent.Username,
		Name:        agent.DisplayName,
		Description: agent.Description,
		Role:        agent.Role,
		Status:      status,
		Version:     agent.Version,
	}
}

func resolveAgent(
	ctx context.Context,
	registry *agents.Registry,
	ref string,
) (AgentInfo, error) {
	if registry == nil {
		return AgentInfo{}, errors.New("agents are unavailable")
	}
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		if agent, err := registry.GetByActor(ctx, ids.ActorID(id)); err == nil {
			return toAgentInfo(agent), nil
		}
	}
	list, err := registry.List(ctx)
	if err != nil {
		return AgentInfo{}, err
	}
	for _, agent := range list {
		if agent.Username == ref || strconv.FormatInt(int64(agent.ID), 10) == ref {
			return toAgentInfo(agent), nil
		}
	}
	return AgentInfo{}, fmt.Errorf("agent %q not found", ref)
}
