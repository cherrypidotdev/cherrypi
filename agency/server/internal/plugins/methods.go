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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/tasks"

	lua "github.com/yuin/gopher-lua"
)

func (h *Host) attachAgency(L *lua.LState, root *lua.LTable) {
	root.RawSetString("get_org", L.NewFunction(func(L *lua.LState) int {
		out, err := h.Bridge.Org(h.Ctx)
		if err != nil {
			L.RaiseError("get_org: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_project", L.NewFunction(func(L *lua.LState) int {
		out, err := h.getProject(L.CheckString(1))
		if err != nil {
			L.RaiseError("get_project: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_team", L.NewFunction(func(L *lua.LState) int {
		out, err := h.Bridge.Team(h.Ctx, L.CheckInt64(1))
		if err != nil {
			L.RaiseError("get_team: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_agent", L.NewFunction(func(L *lua.LState) int {
		out, err := h.getAgent(L.CheckString(1))
		if err != nil {
			L.RaiseError("get_agent: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_user", L.NewFunction(func(L *lua.LState) int {
		out, err := h.getUser(L.CheckString(1))
		if err != nil {
			L.RaiseError("get_user: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_task", L.NewFunction(func(L *lua.LState) int {
		out, err := h.getTask(L.CheckString(1))
		if err != nil {
			L.RaiseError("get_task: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_project_tasks", L.NewFunction(func(L *lua.LState) int {
		out, err := h.getProjectTasks(L.CheckString(1), L.OptTable(2, L.NewTable()))
		if err != nil {
			L.RaiseError("get_project_tasks: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("get_task_history", L.NewFunction(func(L *lua.LState) int {
		out, err := h.getTaskHistory(L.CheckString(1), L.OptTable(2, L.NewTable()))
		if err != nil {
			L.RaiseError("get_task_history: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("create_task", L.NewFunction(func(L *lua.LState) int {
		id, err := h.createTask(L.CheckTable(1))
		if err != nil {
			L.RaiseError("create_task: %v", err)
			return 0
		}
		L.Push(lua.LString(id))
		return 1
	}))
	root.RawSetString("list_agents", L.NewFunction(func(L *lua.LState) int {
		out, err := h.listAgents()
		if err != nil {
			L.RaiseError("list_agents: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("list_project_tags", L.NewFunction(func(L *lua.LState) int {
		out, err := h.listProjectTags(L.CheckString(1))
		if err != nil {
			L.RaiseError("list_project_tags: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("list_settings", L.NewFunction(func(L *lua.LState) int {
		out, err := h.listSettings(L.CheckString(1))
		if err != nil {
			L.RaiseError("list_settings: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
	root.RawSetString("add_task_message", L.NewFunction(func(L *lua.LState) int {
		if err := h.addTaskMessage(L.CheckString(1), L.CheckString(2)); err != nil {
			L.RaiseError("add_task_message: %v", err)
			return 0
		}
		return 0
	}))
	root.RawSetString("close_task", L.NewFunction(func(L *lua.LState) int {
		if err := h.closeTask(L.CheckString(1), L.OptString(2, "completed")); err != nil {
			L.RaiseError("close_task: %v", err)
			return 0
		}
		return 0
	}))
	root.RawSetString("set_task_badges", L.NewFunction(func(L *lua.LState) int {
		if err := h.setTaskBadges(L.CheckString(1), L.Get(2)); err != nil {
			L.RaiseError("set_task_badges: %v", err)
			return 0
		}
		return 0
	}))
	root.RawSetString("hold_task", L.NewFunction(func(L *lua.LState) int {
		hold := L.OptTable(2, L.NewTable())
		if err := h.holdTask(L.CheckString(1), hold); err != nil {
			L.RaiseError("hold_task: %v", err)
			return 0
		}
		return 0
	}))
	root.RawSetString("release_task", L.NewFunction(func(L *lua.LState) int {
		opts := L.OptTable(3, L.NewTable())
		if err := h.releaseTask(L.CheckString(1), L.CheckString(2), opts); err != nil {
			L.RaiseError("release_task: %v", err)
			return 0
		}
		return 0
	}))
	root.RawSetString("list_task_holds", L.NewFunction(func(L *lua.LState) int {
		out, err := h.listTaskHolds(L.CheckString(1))
		if err != nil {
			L.RaiseError("list_task_holds: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}))
}

func (h *Host) getProject(ref string) (map[string]any, error) {
	p, err := h.Bridge.Project(h.Ctx, ref)
	if err != nil {
		return nil, err
	}
	member, link, err := h.projectAccess(p.ID)
	if err != nil {
		return nil, err
	}
	if !member && !link.Enabled {
		return nil, errors.New("the plugin is not installed in this project")
	}
	return projectMap(p), nil
}

func (h *Host) listProjectTags(projectID string) ([]map[string]any, error) {
	link, err := h.Bridge.ProjectSettings(h.Ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !link.Enabled {
		return nil, errors.New("the plugin is not enabled in this project")
	}
	list, err := h.Bridge.ProjectTags(h.Ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(list))
	for i, tag := range list {
		out[i] = map[string]any{"id": tag.ID, "name": tag.Name}
	}
	return out, nil
}

func (h *Host) getAgent(ref string) (map[string]any, error) {
	agent, err := h.Bridge.Agent(h.Ctx, ref)
	if err != nil {
		return nil, err
	}
	allowed, err := h.agentAllowed(agent.ActorID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("the plugin is not enabled for this agent")
	}
	return agentMap(agent), nil
}

func (h *Host) getUser(ref string) (map[string]any, error) {
	actor, err := h.Bridge.User(h.Ctx, ref)
	if err != nil {
		return nil, err
	}
	if actor.Kind != accounts.KindUser {
		return nil, errors.New("not a user")
	}
	allowed, err := h.userAllowed(actor.ID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("the plugin is not enabled for this user")
	}
	return actorMap(actor), nil
}

func (h *Host) getTask(id string) (map[string]any, error) {
	t, err := h.Bridge.Task(h.Ctx, id)
	if err != nil {
		return nil, err
	}
	allowed, err := h.taskAllowed(t)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("the plugin may not read this task")
	}
	return taskMap(t), nil
}

func (h *Host) getProjectTasks(projectID string, opts *lua.LTable) ([]map[string]any, error) {
	link, err := h.Bridge.ProjectSettings(h.Ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !link.Enabled {
		return nil, errors.New("the plugin is not enabled in this project")
	}
	if _, ok, err := h.Bridge.ProjectRole(h.Ctx, projectID, h.pluginActorID()); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("the plugin actor is not a member of the project")
	}
	filter := tasks.Filter{ProjectID: projectID}
	filter.Limit = int64Opt(opts, "limit", 100)
	if state := stringOpt(opts, "state", ""); state != "" {
		filter.State = tasks.State(state)
	}
	if search := stringOpt(opts, "search", ""); search != "" {
		filter.Search = search
	}
	filter.Archived = boolOpt(opts, "archived", false)
	list, err := h.Bridge.Tasks(h.Ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(list))
	for i, t := range list {
		out[i] = taskMap(t)
	}
	return out, nil
}

func (h *Host) getTaskHistory(taskID string, opts *lua.LTable) ([]map[string]any, error) {
	t, err := h.Bridge.Task(h.Ctx, taskID)
	if err != nil {
		return nil, err
	}
	allowed, err := h.taskAllowed(t)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("the plugin may not read this task")
	}
	entries, err := h.Bridge.TaskHistory(h.Ctx, taskID)
	if err != nil {
		return nil, err
	}
	limit := int(int64Opt(opts, "limit", 100))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	out := make([]map[string]any, len(entries))
	for i, entry := range entries {
		out[i] = map[string]any{
			"seq":       entry.Seq,
			"entry_id":  entry.EntryID,
			"parent_id": entry.ParentID,
			"type":      entry.Type,
			"raw":       entry.Raw,
		}
	}
	return out, nil
}

func (h *Host) listAgents() ([]map[string]any, error) {
	rows, err := h.Bridge.ListSettings(h.Ctx, ScopeAgent)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		agent, err := h.Bridge.Agent(h.Ctx, row.TargetID)
		if err != nil {
			continue
		}
		m := agentMap(agent)
		m["settings"] = row.Values
		out = append(out, m)
	}
	return out, nil
}

func (h *Host) listSettings(scope string) ([]map[string]any, error) {
	if scope == ScopeProject {
		list, err := h.Bridge.ActorProjects(h.Ctx, h.pluginActorID())
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(list))
		for _, p := range list {
			row, err := h.Bridge.ProjectSettings(h.Ctx, p.ID)
			if err != nil {
				return nil, err
			}
			out = append(out, map[string]any{
				"target_id":           p.ID,
				"enabled":             row.Enabled,
				"allow_task_creation": row.AllowTaskCreation,
				"max_priority":        row.MaxPriority,
				"settings":            row.Values,
			})
		}
		return out, nil
	}
	rows, err := h.Bridge.ListSettings(h.Ctx, scope)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"target_id": row.TargetID,
			"enabled":   row.Enabled,
			"settings":  row.Values,
		})
	}
	return out, nil
}

func (h *Host) createTask(input *lua.LTable) (string, error) {
	projectRef := stringField(input, "project")
	if projectRef == "" {
		return "", errors.New("project is required")
	}
	project, err := h.Bridge.Project(h.Ctx, projectRef)
	if err != nil {
		return "", err
	}
	link, err := h.Bridge.ProjectSettings(h.Ctx, project.ID)
	if err != nil {
		return "", err
	}
	if !link.Enabled {
		return "", errors.New("the plugin is not enabled in this project")
	}
	if !link.AllowTaskCreation {
		return "", errors.New("the project does not allow plugin-created tasks")
	}
	if _, ok, err := h.Bridge.ProjectRole(
		h.Ctx, project.ID, h.pluginActorID(),
	); err != nil {
		return "", err
	} else if !ok {
		return "", errors.New("the plugin actor is not a member of the project")
	}

	priority, err := parsePriorityInput(input.RawGetString("priority"))
	if err != nil {
		return "", err
	}
	if priority > link.MaxPriority {
		return "", fmt.Errorf(
			"priority %s exceeds the project's plugin cap",
			priorityName(priority),
		)
	}

	tagIDs, err := h.resolveTags(project.ID, input.RawGetString("tags"))
	if err != nil {
		return "", err
	}
	assignee, err := h.resolveAssignee(project.ID, input.RawGetString("assignee"))
	if err != nil {
		return "", err
	}
	refs, err := h.referenceInputs(input.RawGetString("task_references"))
	if err != nil {
		return "", err
	}
	tools, err := h.taskTools(input.RawGetString("tools"))
	if err != nil {
		return "", err
	}
	size := strings.ToLower(strings.TrimSpace(
		luaValueString(input.RawGetString("size")),
	))
	switch size {
	case "":
		size = tasks.SizeMedium
	case tasks.SizeSmall, tasks.SizeMedium, tasks.SizeBig, tasks.SizeEpic:
	default:
		return "", fmt.Errorf("unknown size %q", size)
	}

	title := stringField(input, "title")
	t, err := h.Bridge.CreateTask(h.Ctx, tasks.CreateParams{
		Prompt:          stringField(input, "prompt"),
		Name:            title,
		ProjectID:       project.ID,
		AssigneeActorID: assignee,
		TagIDs:          tagIDs,
		References:      refs,
		Priority:        priority,
		Size:            size,
		State:           tasks.StateBacklog,
		AutoTitle:       title == "",

		LeadActorIDs: []ids.ActorID{h.pluginActorID()},
	})
	if err != nil {
		return "", err
	}
	if err := h.Bridge.AttachTools(h.Ctx, t.ID, tools); err != nil {
		return "", err
	}
	if _, err := h.Bridge.QueueTask(h.Ctx, t.ID); err != nil {
		return "", err
	}
	h.log(
		LevelInfo,
		"created task "+t.ID,
		map[string]any{"project": project.ID},
		[]LogSurface{{Scope: ScopeProject, TargetID: project.ID}},
	)
	return t.ID, nil
}

func (h *Host) resolveTags(projectID string, value lua.LValue) ([]string, error) {
	if value == nil || value == lua.LNil {
		return nil, nil
	}
	if _, ok := value.(*lua.LTable); !ok {
		return nil, errors.New("tags must be a list")
	}
	list, err := h.Bridge.ProjectTags(h.Ctx, projectID)
	if err != nil {
		return nil, err
	}
	byID := map[string]bool{}
	byName := map[string]string{}
	for _, tag := range list {
		byID[tag.ID] = true
		byName[strings.ToLower(tag.Name)] = tag.ID
	}
	var out []string
	for _, item := range luaList(value) {
		text := luaValueString(item)
		if text == "" {
			continue
		}
		if byID[text] {
			out = append(out, text)
			continue
		}
		if id, ok := byName[strings.ToLower(text)]; ok {
			out = append(out, id)
			continue
		}
		return nil, fmt.Errorf("tag %q is not defined in the project", text)
	}
	return out, nil
}

func (h *Host) resolveAssignee(projectID string, value lua.LValue) (ids.ActorID, error) {
	text := strings.TrimSpace(luaValueString(value))
	if text == "" {
		return 0, nil
	}
	agent, err := h.Bridge.Agent(h.Ctx, text)
	if err != nil {
		return 0, errors.New("assignee is not an agent")
	}
	role, ok, err := h.Bridge.ProjectRole(h.Ctx, projectID, agent.ActorID)
	if err != nil {
		return 0, err
	}
	if !ok || role == "" {
		return 0, errors.New("assignee is not a member of the project")
	}
	return agent.ActorID, nil
}

func (h *Host) referenceInputs(value lua.LValue) ([]tasks.ReferenceInput, error) {
	if value == nil || value == lua.LNil {
		return nil, nil
	}
	if _, ok := value.(*lua.LTable); !ok {
		return nil, errors.New("task_references must be a list")
	}
	out := make([]tasks.ReferenceInput, 0)
	for _, item := range luaList(value) {
		if table, ok := item.(*lua.LTable); ok {
			id := stringField(table, "task_id")
			if id == "" {
				id = stringField(table, "id")
			}
			if id == "" {
				return nil, errors.New("a task reference needs task_id")
			}
			kind := tasks.ReferenceKind(strings.TrimSpace(
				luaValueString(table.RawGetString("kind")),
			))
			if kind == "" {
				kind = tasks.ReferenceInformational
			}
			out = append(out, tasks.ReferenceInput{TaskID: id, Kind: kind})
			continue
		}
		id := strings.TrimSpace(luaValueString(item))
		if id == "" {
			continue
		}
		out = append(out, tasks.ReferenceInput{
			TaskID: id,
			Kind:   tasks.ReferenceInformational,
		})
	}
	return out, nil
}

func (h *Host) taskTools(value lua.LValue) ([]Tool, error) {
	if value == nil || value == lua.LNil {
		return nil, nil
	}
	if _, ok := value.(*lua.LTable); !ok {
		return nil, errors.New("tools must be a list")
	}
	var out []Tool
	for _, item := range luaList(value) {
		table, ok := item.(*lua.LTable)
		if !ok {
			return nil, errors.New("each tool must be a table")
		}
		name := stringField(table, "name")
		declared, ok := h.Plugin.Manifest.Tool(name)
		if !ok {
			return nil, fmt.Errorf("tool %q is not declared in the manifest", name)
		}
		tool := Tool{
			Name:        declared.Name,
			Description: stringField(table, "description"),
			InputSchema: declared.InputSchema,
		}
		if tool.Description == "" {
			tool.Description = declared.Description
		}
		if raw := table.RawGetString("arg_schema"); raw != lua.LNil {
			tool.InputSchema = jsonBytes(raw)
		}
		if raw := table.RawGetString("input_schema"); raw != lua.LNil {
			tool.InputSchema = jsonBytes(raw)
		}
		if len(tool.InputSchema) == 0 {
			tool.InputSchema = declared.InputSchema
		}
		out = append(out, tool)
	}
	return out, nil
}

func (h *Host) projectAccess(projectID string) (bool, ProjectSettings, error) {
	link, err := h.Bridge.ProjectSettings(h.Ctx, projectID)
	if err != nil {
		return false, ProjectSettings{}, err
	}
	role, ok, err := h.Bridge.ProjectRole(h.Ctx, projectID, h.pluginActorID())
	if err != nil {
		return false, ProjectSettings{}, err
	}
	return ok && role != "", link, nil
}

func (h *Host) agentAllowed(actorID ids.ActorID) (bool, error) {
	row, err := h.Bridge.Settings(
		h.Ctx, ScopeAgent, strconv.FormatInt(int64(actorID), 10),
	)
	if err != nil {
		return false, err
	}
	if row.Enabled {
		return true, nil
	}
	projects, err := h.Bridge.ActorProjects(h.Ctx, actorID)
	if err != nil {
		return false, err
	}
	for _, project := range projects {
		link, err := h.Bridge.ProjectSettings(h.Ctx, project.ID)
		if err != nil {
			return false, err
		}
		if link.Enabled {
			return true, nil
		}
	}
	return false, nil
}

func (h *Host) userAllowed(actorID ids.ActorID) (bool, error) {
	row, err := h.Bridge.Settings(
		h.Ctx, ScopeUser, strconv.FormatInt(int64(actorID), 10),
	)
	if err != nil {
		return false, err
	}
	return row.Enabled, nil
}

func (h *Host) taskAllowed(t tasks.Task) (bool, error) {
	attached, ok, err := h.Bridge.TaskPlugin(h.Ctx, t.ID)
	if err != nil {
		return false, err
	}
	if ok && attached.PluginID == h.Plugin.ID {
		return true, nil
	}
	link, err := h.Bridge.ProjectSettings(h.Ctx, t.ProjectID)
	if err != nil {
		return false, err
	}
	if !link.Enabled {
		return false, nil
	}
	role, ok, err := h.Bridge.ProjectRole(h.Ctx, t.ProjectID, h.pluginActorID())
	if err != nil {
		return false, err
	}
	return ok && role != "", nil
}

func (h *Host) pluginActorID() ids.ActorID {
	return ids.ActorID(h.Plugin.ActorID)
}

func projectMap(p projects.Project) map[string]any {
	return map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"slug":        p.Slug,
		"description": p.Description,
		"parent_id":   p.ParentID,
	}
}

func agentMap(a AgentInfo) map[string]any {
	return map[string]any{
		"id":          int64(a.ID),
		"actor_id":    int64(a.ActorID),
		"username":    a.Username,
		"name":        a.Name,
		"description": a.Description,
		"role":        a.Role,
		"status":      a.Status,
		"version":     a.Version,
	}
}

func actorMap(a accounts.Actor) map[string]any {
	return map[string]any{
		"id":          int64(a.ID),
		"actor_id":    int64(a.ID),
		"kind":        a.Kind,
		"username":    a.Username,
		"name":        a.Name,
		"description": a.Description,
	}
}

func taskMap(t tasks.Task) map[string]any {
	tagsOut := make([]map[string]any, len(t.Tags))
	for i, tag := range t.Tags {
		tagsOut[i] = map[string]any{
			"id":    tag.ID,
			"name":  tag.Name,
			"color": tag.Color,
		}
	}
	return map[string]any{
		"id":            t.ID,
		"name":          t.Name,
		"prompt":        t.Prompt,
		"project_id":    t.ProjectID,
		"assignee":      int64(t.AssigneeActorID),
		"creator":       int64(t.CreatorActorID),
		"priority":      t.Priority,
		"size":          t.Size,
		"state":         string(t.State),
		"tags":          tagsOut,
		"references":    t.References,
		"session_id":    t.SessionID,
		"error":         t.Error,
		"created_at":    t.CreatedAt.UnixMilli(),
		"updated_at":    t.UpdatedAt.UnixMilli(),
		"state_updated": t.StateUpdatedAt.UnixMilli(),
	}
}

func parsePriorityInput(value lua.LValue) (int64, error) {
	text := strings.ToLower(strings.TrimSpace(luaValueString(value)))
	switch text {
	case "", "normal", "medium", "0":
		return tasks.PriorityNormal, nil
	case "low", "-1":
		return tasks.PriorityLow, nil
	case "high", "1":
		return tasks.PriorityHigh, nil
	case "critical", "2":
		return tasks.PriorityCritical, nil
	default:
		return 0, fmt.Errorf("unknown priority %q", text)
	}
}

func priorityName(priority int64) string {
	switch priority {
	case tasks.PriorityLow:
		return "low"
	case tasks.PriorityHigh:
		return "high"
	case tasks.PriorityCritical:
		return "critical"
	default:
		return "normal"
	}
}

func stringField(table *lua.LTable, key string) string {
	return strings.TrimSpace(luaValueString(table.RawGetString(key)))
}

func luaValueString(value lua.LValue) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case lua.LString:
		return string(v)
	case lua.LNumber:
		if float64(int64(v)) == float64(v) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case lua.LBool:
		if bool(v) {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func luaList(value lua.LValue) []lua.LValue {
	table, ok := value.(*lua.LTable)
	if !ok {
		return nil
	}
	out := make([]lua.LValue, 0, table.Len())
	for i := 1; i <= table.Len(); i++ {
		out = append(out, table.RawGetInt(i))
	}
	return out
}

func int64Opt(table *lua.LTable, key string, fallback int64) int64 {
	if table == nil {
		return fallback
	}
	value := table.RawGetString(key)
	if value == lua.LNil {
		return fallback
	}
	if n, ok := value.(lua.LNumber); ok {
		return int64(n)
	}
	if n, err := strconv.ParseInt(luaValueString(value), 10, 64); err == nil {
		return n
	}
	return fallback
}

func stringOpt(table *lua.LTable, key, fallback string) string {
	if table == nil {
		return fallback
	}
	value := table.RawGetString(key)
	if value == lua.LNil {
		return fallback
	}
	out := luaValueString(value)
	if out == "" {
		return fallback
	}
	return out
}

func boolOpt(table *lua.LTable, key string, fallback bool) bool {
	if table == nil {
		return fallback
	}
	value := table.RawGetString(key)
	if value == lua.LNil {
		return fallback
	}
	if b, ok := value.(lua.LBool); ok {
		return bool(b)
	}
	return fallback
}

func jsonBytes(value lua.LValue) []byte {
	raw := fromLua(value)
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	return encoded
}
