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
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/tasks"

	lua "github.com/yuin/gopher-lua"
)

func (h *Host) attachTest(L *lua.LState) {
	fake := h.Fake
	if fake == nil {
		return
	}
	root := L.GetGlobal("cherrypi").(*lua.LTable)
	table := L.NewTable()
	root.RawSetString("test", table)

	if pkg, ok := L.GetGlobal("package").(*lua.LTable); ok {
		if preload, ok := pkg.RawGetString("preload").(*lua.LTable); ok {
			preload.RawSetString("cherrypi.test", L.NewFunction(func(L *lua.LState) int {
				L.Push(table)
				return 1
			}))
		}
	}

	table.RawSetString("settings", L.NewFunction(func(L *lua.LState) int {
		scope := L.CheckString(1)
		target := ""
		values := L.Get(2)
		if L.GetTop() >= 3 {
			target = L.OptString(2, "")
			values = L.Get(3)
		}
		m := tableMap(values)
		if scope == ScopeOrg && target == "" {
			h.Settings = m
		}
		fake.SetSettings(Settings{
			PluginID: h.Plugin.ID,
			Scope:    scope,
			TargetID: target,
			Values:   m,
			Enabled:  true,
		})
		return 0
	}))
	table.RawSetString("enable", L.NewFunction(func(L *lua.LState) int {
		scope := L.CheckString(1)
		target := L.OptString(2, "")
		row, err := fake.Settings(h.Ctx, scope, target)
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		row.Enabled = true
		fake.SetSettings(row)
		return 0
	}))
	table.RawSetString("project", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		id := stringField(tbl, "id")
		if id == "" {
			id = stringField(tbl, "name")
		}
		if id == "" {
			L.RaiseError("project needs an id or name")
			return 0
		}
		p := projects.Project{
			ID:          id,
			Name:        stringField(tbl, "name"),
			Slug:        stringField(tbl, "slug"),
			Description: stringField(tbl, "description"),
		}
		if p.Name == "" {
			p.Name = id
		}
		fake.SetProject(p, stringList(tbl.RawGetString("tags"))...)
		if role := stringOpt(tbl, "plugin_role", "member"); role != "" {
			fake.SetProjectRole(id, ids.ActorID(h.Plugin.ActorID), role)
		}
		row := ProjectSettings{
			PluginID:          h.Plugin.ID,
			ProjectID:         id,
			Enabled:           boolOpt(tbl, "enabled", true),
			AllowTaskCreation: boolOpt(tbl, "allow_task_creation", true),
			MaxPriority:       int64Opt(tbl, "max_priority", 0),
			Values:            tableMap(tbl.RawGetString("settings")),
		}
		fake.SetProjectSettings(row)
		return 0
	}))
	table.RawSetString("project_role", L.NewFunction(func(L *lua.LState) int {
		projectID := L.CheckString(1)
		actorID := ids.ActorID(L.CheckInt64(2))
		role := L.CheckString(3)
		fake.SetProjectRole(projectID, actorID, role)
		return 0
	}))
	table.RawSetString("agent", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		agent := AgentInfo{
			ID:          ids.AgentID(int64Opt(tbl, "id", 1)),
			ActorID:     ids.ActorID(int64Opt(tbl, "actor_id", 1)),
			Username:    stringField(tbl, "username"),
			Name:        stringField(tbl, "name"),
			Description: stringField(tbl, "description"),
			Role:        stringOpt(tbl, "role", "member"),
			Status:      stringOpt(tbl, "status", "online"),
			Version:     stringOpt(tbl, "version", "test"),
		}
		if agent.Username == "" {
			agent.Username = fmt.Sprintf("agent-%d", agent.ID)
		}
		fake.SetAgent(agent)
		if projectID := stringField(tbl, "project"); projectID != "" {
			fake.SetProjectRole(
				projectID,
				agent.ActorID,
				stringOpt(tbl, "project_role", "member"),
			)
		}
		return 0
	}))
	table.RawSetString("user", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		actor := accounts.Actor{
			ID:          ids.ActorID(int64Opt(tbl, "id", 1)),
			Kind:        stringOpt(tbl, "kind", accounts.KindUser),
			Username:    stringField(tbl, "username"),
			Name:        stringField(tbl, "name"),
			Description: stringField(tbl, "description"),
		}
		if actor.Username == "" {
			actor.Username = fmt.Sprintf("user-%d", actor.ID)
		}
		fake.SetUser(actor)
		return 0
	}))
	table.RawSetString("task", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		id := stringField(tbl, "id")
		if id == "" {
			L.RaiseError("task needs an id")
			return 0
		}
		t := tasks.Task{
			ID:              id,
			Name:            stringField(tbl, "name"),
			Prompt:          stringField(tbl, "prompt"),
			ProjectID:       stringField(tbl, "project_id"),
			AssigneeActorID: ids.ActorID(int64Opt(tbl, "assignee", 0)),
			State:           tasks.State(stringOpt(tbl, "state", string(tasks.StateQueued))),
			Size:            stringOpt(tbl, "size", tasks.SizeMedium),
		}
		fake.SetTask(t)
		return 0
	}))
	table.RawSetString("history", L.NewFunction(func(L *lua.LState) int {
		taskID := L.CheckString(1)
		entries := L.CheckTable(2)
		var out []tasks.SessionHistoryEntry
		entries.ForEach(func(_, value lua.LValue) {
			tbl, ok := value.(*lua.LTable)
			if !ok {
				return
			}
			out = append(out, tasks.SessionHistoryEntry{
				Seq:      int64Opt(tbl, "seq", int64(len(out)+1)),
				EntryID:  stringField(tbl, "entry_id"),
				ParentID: stringField(tbl, "parent_id"),
				Type:     stringOpt(tbl, "type", "message"),
				Raw:      stringField(tbl, "raw"),
			})
		})
		fake.SetTaskHistory(taskID, out)
		return 0
	}))
	table.RawSetString("team", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		fake.SetTeam(int64Opt(tbl, "id", 1), map[string]any{
			"id":          int64Opt(tbl, "id", 1),
			"name":        stringField(tbl, "name"),
			"description": stringField(tbl, "description"),
		})
		return 0
	}))
	table.RawSetString("expect_http", L.NewFunction(func(L *lua.LState) int {
		method := L.CheckString(1)
		path := L.CheckString(2)
		response := L.OptTable(3, L.NewTable())
		status := int(int64Opt(response, "status", 200))
		body := stringOpt(response, "body", "")
		if raw := response.RawGetString("json"); raw != lua.LNil {
			encoded, err := json.Marshal(fromLua(raw))
			if err != nil {
				L.RaiseError("%v", err)
				return 0
			}
			body = string(encoded)
		}
		headers := map[string]string{}
		if tbl, ok := response.RawGetString("headers").(*lua.LTable); ok {
			tbl.ForEach(func(key, value lua.LValue) {
				headers[key.String()] = value.String()
			})
		}
		fake.ExpectHTTP(method, path, status, body, headers)
		return 0
	}))
	table.RawSetString("http_request", L.NewFunction(func(L *lua.LState) int {
		fn, err := handler(L, "handle_http_request")
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		method := L.OptString(1, "GET")
		target := L.OptString(2, "/")
		body := L.OptString(3, "")
		routePath, query := splitRouteTarget(target)
		route := L.NewTable()
		route.RawSetString("method", lua.LString(method))
		route.RawSetString("path", lua.LString(routePath))
		route.RawSetString("query", queryTable(L, query))
		headers := L.NewTable()
		if tbl, ok := L.Get(4).(*lua.LTable); ok {
			tbl.ForEach(func(key, value lua.LValue) {
				headers.RawSetString(key.String(), value)
			})
		}
		if err := L.CallByParam(
			lua.P{Fn: fn, NRet: 3, Protect: true},
			route,
			headers,
			lua.LString(body),
		); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		return 3
	}))
	table.RawSetString("tool_call", L.NewFunction(func(L *lua.LState) int {
		fn, err := handler(L, "handle_tool_call")
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		taskID := L.CheckString(1)
		name := L.CheckString(2)
		args := L.OptTable(3, L.NewTable())
		if err := L.CallByParam(
			lua.P{Fn: fn, NRet: 1, Protect: true},
			lua.LString(taskID),
			lua.LString(name),
			args,
		); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		return 1
	}))

	table.RawSetString("task_tools", L.NewFunction(func(L *lua.LState) int {
		taskID := L.CheckString(1)
		declared := h.Plugin.Manifest.ProjectTools
		fn, ok := L.GetGlobal("handle_task_tools").(*lua.LFunction)
		if !ok || !h.Plugin.Manifest.HasHandler(HandlerTaskTools) {
			L.Push(toLua(L, declared))
			return 1
		}
		if err := L.CallByParam(
			lua.P{Fn: fn, NRet: 1, Protect: true},
			lua.LString(taskID),
		); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		ret := L.Get(-1)
		L.Pop(1)
		if ret == lua.LNil {
			L.Push(toLua(L, declared))
			return 1
		}
		body, err := json.Marshal(fromLua(ret))
		if err != nil {
			L.RaiseError("handle_task_tools must return a list of tool names: %v", err)
			return 0
		}
		names, err := parseTaskToolNames(body, declared)
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		L.Push(toLua(L, names))
		return 1
	}))
	table.RawSetString("event", L.NewFunction(func(L *lua.LState) int {
		fn, err := handler(L, "handle_event")
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		name := L.CheckString(1)
		payload := L.OptTable(2, L.NewTable())
		payload.RawSetString("event", lua.LString(name))
		if err := L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, payload); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		return 0
	}))

	table.RawSetString("tick", L.NewFunction(func(L *lua.LState) int {
		fn, err := handler(L, "handle_tick")
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		nowMs := L.OptInt64(1, time.Now().UnixMilli())
		if err := L.CallByParam(
			lua.P{Fn: fn, NRet: 0, Protect: true},
			lua.LNumber(nowMs),
		); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		return 0
	}))

	table.RawSetString("ticks", L.NewFunction(func(L *lua.LState) int {
		out := L.NewTable()
		if schedule := h.Plugin.Manifest.Schedule; schedule != nil {
			out.RawSetString("every_seconds", lua.LNumber(schedule.EverySeconds))
			out.RawSetString("interval_ms", lua.LNumber(schedule.EverySeconds*1000))
		}
		L.Push(out)
		return 1
	}))

	table.RawSetString("update_settings", L.NewFunction(func(L *lua.LState) int {
		scope := L.OptString(1, ScopeOrg)
		target := ""
		values := L.Get(2)
		if L.GetTop() >= 3 {
			target = L.OptString(2, "")
			values = L.Get(3)
		}
		fn, ok := L.GetGlobal("update_settings").(*lua.LFunction)
		if !ok {
			L.Push(L.NewTable())
			return 1
		}
		previous, err := fake.settingsForScope(h.Ctx, scope, target)
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		submitted, ok := values.(*lua.LTable)
		if !ok {
			submitted = L.NewTable()
		}

		merged := L.NewTable()
		submitted.ForEach(func(key, value lua.LValue) {
			merged.RawSet(key, value)
		})
		for key := range h.Plugin.Manifest.SecretKeys(scope) {
			current := submitted.RawGetString(key)
			blank := current == lua.LNil ||
				(current.Type() == lua.LTString && current.String() == "")
			if !blank {
				continue
			}
			if stored, ok := previous[key].(string); ok && stored != "" {
				merged.RawSetString(key, lua.LString(stored))
			}
		}
		if err := L.CallByParam(
			lua.P{Fn: fn, NRet: 1, Protect: true},
			lua.LString(scope),
			lua.LString(target),
			merged,
			toLua(L, previous),
		); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		ret := L.Get(-1)
		if ret == lua.LNil {
			L.Pop(1)
			L.Push(L.NewTable())
			return 1
		}
		body, err := json.Marshal(fromLua(ret))
		if err != nil {
			L.RaiseError("update_settings must return a list of {path, errorText}: %v", err)
			return 0
		}

		if err := parseSettingErrors(body); !settingsRejectedOnly(err) {
			L.RaiseError("%v", err)
			return 0
		}
		return 1
	}))
	table.RawSetString("created_tasks", L.NewFunction(func(L *lua.LState) int {
		rows := fake.CreatedTaskRows()
		out := L.NewTable()
		for _, row := range rows {
			out.Append(toLua(L, map[string]any{
				"id":          row.ID,
				"project":     row.ProjectID,
				"name":        row.Name,
				"prompt":      row.Prompt,
				"assignee":    int64(row.AssigneeActorID),
				"priority":    row.Priority,
				"size":        row.Size,
				"interactive": row.Interactive,
				"tags":        tagNames(row.Tags),
				"tools":       toolNames(fake.Tools(row.ID)),
				"state":       string(row.State),
			}))
		}
		L.Push(out)
		return 1
	}))
	table.RawSetString("http_calls", L.NewFunction(func(L *lua.LState) int {
		L.Push(toLua(L, fake.HTTPCalls()))
		return 1
	}))
	table.RawSetString("logs", L.NewFunction(func(L *lua.LState) int {
		L.Push(toLua(L, fake.Logs()))
		return 1
	}))
	table.RawSetString("task_messages", L.NewFunction(func(L *lua.LState) int {
		out := L.NewTable()
		for _, row := range fake.TaskMessages() {
			out.Append(toLua(L, map[string]any{
				"task_id": row.TaskID,
				"text":    row.Text,
			}))
		}
		L.Push(out)
		return 1
	}))
	table.RawSetString("closed_tasks", L.NewFunction(func(L *lua.LState) int {
		out := L.NewTable()
		for _, row := range fake.ClosedTasks() {
			out.Append(toLua(L, map[string]any{
				"task_id": row.TaskID,
				"outcome": string(row.Outcome),
			}))
		}
		L.Push(out)
		return 1
	}))

	table.RawSetString("holds", L.NewFunction(func(L *lua.LState) int {
		taskID := L.CheckString(1)
		out := L.NewTable()
		for _, hold := range fake.Holds(taskID) {
			out.Append(toLua(L, map[string]any{
				"plugin":      hold.PluginID,
				"plugin_name": hold.PluginName,
				"key":         hold.Key,
				"reason":      hold.Reason,
				"created_at":  hold.CreatedAt.UnixMilli(),
			}))
		}
		L.Push(out)
		return 1
	}))

	table.RawSetString("releases", L.NewFunction(func(L *lua.LState) int {
		taskID := L.OptString(1, "")
		out := L.NewTable()
		for _, release := range fake.Releases(taskID) {
			out.Append(toLua(L, map[string]any{
				"task_id": release.TaskID,
				"key":     release.Key,
				"mode":    release.Mode,
				"prompt":  release.Prompt,
			}))
		}
		L.Push(out)
		return 1
	}))
	table.RawSetString("task_badges", L.NewFunction(func(L *lua.LState) int {
		taskID := L.CheckString(1)
		out := L.NewTable()
		for _, badge := range fake.TaskBadges(taskID) {
			out.Append(toLua(L, map[string]any{
				"key":   badge.Key,
				"label": badge.Label,
				"value": badge.Value,
				"tone":  badge.Tone,
				"url":   badge.URL,
			}))
		}
		L.Push(out)
		return 1
	}))
	table.RawSetString("json", L.NewFunction(func(L *lua.LState) int {
		encoded, err := json.Marshal(fromLua(L.Get(1)))
		if err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		L.Push(lua.LString(encoded))
		return 1
	}))
	table.RawSetString("decode", L.NewFunction(func(L *lua.LState) int {
		var value any
		if err := json.Unmarshal([]byte(L.CheckString(1)), &value); err != nil {
			L.RaiseError("%v", err)
			return 0
		}
		L.Push(toLua(L, value))
		return 1
	}))
	table.RawSetString("equal", L.NewFunction(func(L *lua.LState) int {
		actual, expected := fromLua(L.Get(1)), fromLua(L.Get(2))
		if !reflect.DeepEqual(normalize(actual), normalize(expected)) {
			L.RaiseError(
				"%s: want %s, got %s",
				optionalMessage(L),
				describe(expected),
				describe(actual),
			)
		}
		return 0
	}))
	table.RawSetString("not_equal", L.NewFunction(func(L *lua.LState) int {
		actual, expected := fromLua(L.Get(1)), fromLua(L.Get(2))
		if reflect.DeepEqual(normalize(actual), normalize(expected)) {
			L.RaiseError("%s: values are equal (%s)", optionalMessage(L), describe(actual))
		}
		return 0
	}))
	table.RawSetString("contains", L.NewFunction(func(L *lua.LState) int {
		haystack := L.CheckString(1)
		needle := L.CheckString(2)
		if !strings.Contains(haystack, needle) {
			L.RaiseError(
				"%s: %q does not contain %q",
				optionalMessage(L),
				haystack,
				needle,
			)
		}
		return 0
	}))
	table.RawSetString("truthy", L.NewFunction(func(L *lua.LState) int {
		value := L.Get(1)
		if value == lua.LNil || value == lua.LFalse {
			L.RaiseError("%s: value is falsy", optionalMessage(L))
		}
		return 0
	}))
	table.RawSetString("falsy", L.NewFunction(func(L *lua.LState) int {
		value := L.Get(1)
		if value != lua.LNil && value != lua.LFalse {
			L.RaiseError("%s: value is truthy (%s)", optionalMessage(L), value.String())
		}
		return 0
	}))
	table.RawSetString("fail", L.NewFunction(func(L *lua.LState) int {
		L.RaiseError("%s", optionalMessage(L))
		return 0
	}))
}

func splitRouteTarget(target string) (string, url.Values) {
	routePath, rawQuery, _ := strings.Cut(target, "?")
	query, _ := url.ParseQuery(rawQuery)
	if routePath == "" {
		routePath = "/"
	}
	return routePath, query
}

func queryTable(L *lua.LState, query url.Values) *lua.LTable {
	table := L.NewTable()
	for key, values := range query {
		list := L.NewTable()
		for _, value := range values {
			list.Append(lua.LString(value))
		}
		table.RawSetString(key, list)
	}
	return table
}

func optionalMessage(L *lua.LState) string {
	if msg := L.OptString(3, ""); msg != "" {
		return msg
	}
	return "assertion failed"
}

func describe(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(raw)
}

func normalize(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out any
	if json.Unmarshal(raw, &out) != nil {
		return value
	}
	return out
}

func tableMap(value lua.LValue) map[string]any {
	tbl, ok := value.(*lua.LTable)
	if !ok {
		return map[string]any{}
	}
	out, _ := fromLua(tbl).(map[string]any)
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func stringList(value lua.LValue) []string {
	tbl, ok := value.(*lua.LTable)
	if !ok {
		return nil
	}
	out := make([]string, 0, tbl.Len())
	for i := 1; i <= tbl.Len(); i++ {
		out = append(out, luaValueString(tbl.RawGetInt(i)))
	}
	return out
}

func tagNames(list []tasks.Tag) []string {
	out := make([]string, len(list))
	for i, tag := range list {
		out[i] = tag.Name
	}
	return out
}

func toolNames(list []Tool) []string {
	out := make([]string, len(list))
	for i, tool := range list {
		out[i] = tool.Name
	}
	return out
}
