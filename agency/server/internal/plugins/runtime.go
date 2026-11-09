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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultRespLimit  = 5 << 20
	defaultConcurrent = 2
)

type Invocation struct {
	ID     string         `json:"invocation_id"`
	Kind   string         `json:"kind"`
	Caller Caller         `json:"caller"`
	Route  *RouteRequest  `json:"route,omitempty"`
	TaskID string         `json:"task_id,omitempty"`
	Tool   string         `json:"tool,omitempty"`
	Args   map[string]any `json:"args,omitempty"`
	Event  string         `json:"event,omitempty"`
	Body   map[string]any `json:"body,omitempty"`

	Scope    string         `json:"scope,omitempty"`
	TargetID string         `json:"target_id,omitempty"`
	Values   map[string]any `json:"values,omitempty"`
	Previous map[string]any `json:"previous,omitempty"`

	allowDisabled bool
}

type Caller struct {
	Kind     string `json:"kind"`
	ActorID  int64  `json:"actor_id,omitempty"`
	Username string `json:"username,omitempty"`
	Admin    bool   `json:"admin,omitempty"`
}

type RouteRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Query   map[string][]string `json:"query,omitempty"`
	Headers map[string]string   `json:"headers,omitempty"`
	Body    []byte              `json:"body,omitempty"`
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

type Runtime struct {
	mu      sync.Mutex
	plugins map[string]*compiled
}

type compiled struct {
	manifest Manifest
	protos   map[string]*lua.FunctionProto
}

func NewRuntime() *Runtime {
	return &Runtime{plugins: map[string]*compiled{}}
}

func (r *Runtime) Load(p Plugin, dir string) error {
	protos := map[string]*lua.FunctionProto{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		proto, err := compileLua(body, rel)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		protos[rel] = proto
		return nil
	})
	if err != nil {
		return err
	}
	entry := p.Manifest.Entry
	if entry == "" {
		entry = "main.lua"
	}
	if protos[entry] == nil {
		return fmt.Errorf("bundle needs %s", entry)
	}
	r.mu.Lock()
	r.plugins[p.ID] = &compiled{manifest: p.Manifest, protos: protos}
	r.mu.Unlock()
	return nil
}

func (r *Runtime) Unload(id string) {
	r.mu.Lock()
	delete(r.plugins, id)
	r.mu.Unlock()
}

func (r *Runtime) compiled(p Plugin) (*compiled, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.plugins[p.ID]
	if c == nil {
		return nil, errors.New("plugin is not loaded")
	}
	if c.manifest.Version != p.Version {
		return nil, errors.New("plugin version changed; reload it")
	}
	return c, nil
}

func compileLua(body []byte, name string) (*lua.FunctionProto, error) {
	chunk, err := parse.Parse(strings.NewReader(string(body)), name)
	if err != nil {
		return nil, err
	}
	return lua.Compile(chunk, name)
}

func (r *Runtime) Call(
	ctx context.Context,
	p Plugin,
	host *Host,
	inv Invocation,
) (Response, error) {
	c, err := r.compiled(p)
	if err != nil {
		return Response{}, err
	}
	timeout := defaultTimeout
	if p.Manifest.Limits.TimeoutSeconds > 0 {
		timeout = time.Duration(p.Manifest.Limits.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	L := lua.NewState(lua.Options{
		SkipOpenLibs:        true,
		RegistrySize:        1024 * 8,
		CallStackSize:       256,
		MinimizeStackMemory: true,
	})
	defer L.Close()
	L.SetContext(ctx)
	defer L.RemoveContext()

	if err := openSandbox(L, host); err != nil {
		return Response{}, err
	}
	installModules(L, c.protos)
	host.attach(L)

	entry := c.manifest.Entry
	if entry == "" {
		entry = "main.lua"
	}
	if err := callProto(L, c.protos[entry], 0); err != nil {
		return Response{}, fmt.Errorf("%s: %w", entry, err)
	}

	switch inv.Kind {
	case HandlerHTTP:
		return callHTTPHandler(L, inv)
	case HandlerTool:
		return callToolHandler(L, inv)
	case HandlerEvent:
		return callEventHandler(L, inv)
	case HandlerSettings:
		return callSettingsHandler(L, inv)
	default:
		return Response{}, fmt.Errorf("unknown invocation kind %q", inv.Kind)
	}
}

func (r *Runtime) RunTestFile(
	ctx context.Context,
	p Plugin,
	host *Host,
	file string,
) error {
	c, err := r.compiled(p)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	L := lua.NewState(lua.Options{
		SkipOpenLibs:        true,
		RegistrySize:        1024 * 8,
		CallStackSize:       256,
		MinimizeStackMemory: true,
	})
	defer L.Close()
	L.SetContext(ctx)
	defer L.RemoveContext()

	if err := openSandbox(L, host); err != nil {
		return err
	}
	installModules(L, c.protos)
	host.attach(L)
	host.attachTest(L)

	entry := c.manifest.Entry
	if entry == "" {
		entry = "main.lua"
	}
	if err := callProto(L, c.protos[entry], 0); err != nil {
		return fmt.Errorf("%s: %w", entry, err)
	}
	proto := c.protos[file]
	if proto == nil {
		return fmt.Errorf("test file %s is not in the bundle", file)
	}
	return callProto(L, proto, 0)
}

func handler(L *lua.LState, name string) (*lua.LFunction, error) {
	fn, ok := L.GetGlobal(name).(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("plugin does not define %s", name)
	}
	return fn, nil
}

func callHTTPHandler(L *lua.LState, inv Invocation) (Response, error) {
	fn, err := handler(L, "handle_http_request")
	if err != nil {
		return Response{}, err
	}
	route := L.NewTable()
	route.RawSetString("method", lua.LString(inv.Route.Method))
	route.RawSetString("path", lua.LString(inv.Route.Path))
	query := L.NewTable()
	for key, values := range inv.Route.Query {
		list := L.NewTable()
		for _, value := range values {
			list.Append(lua.LString(value))
		}
		query.RawSetString(key, list)
	}
	route.RawSetString("query", query)
	headers := L.NewTable()
	for key, value := range inv.Route.Headers {
		headers.RawSetString(key, lua.LString(value))
	}
	if err := L.CallByParam(
		lua.P{Fn: fn, NRet: 3, Protect: true},
		route,
		headers,
		lua.LString(inv.Route.Body),
	); err != nil {
		return Response{}, err
	}

	if first := L.Get(-3); first.Type() == lua.LTTable {
		ret := L.Get(-1)
		L.Pop(3)
		return responseFromTable(ret)
	}
	status := 200
	headersOut := map[string]string{}
	bodyOut := []byte{}
	if n, ok := L.Get(-3).(lua.LNumber); ok && n > 0 {
		status = int(n)
	}
	if table, ok := L.Get(-2).(*lua.LTable); ok {
		table.ForEach(func(key, value lua.LValue) {
			headersOut[key.String()] = value.String()
		})
	}
	switch value := L.Get(-1).(type) {
	case lua.LString:
		bodyOut = []byte(value)
	case *lua.LTable:
		raw, err := json.Marshal(fromLua(value))
		if err != nil {
			return Response{}, err
		}
		bodyOut = raw
		if _, ok := headersOut["content-type"]; !ok {
			headersOut["content-type"] = "application/json"
		}
	case *lua.LNilType:
	default:
		bodyOut = []byte(value.String())
	}
	L.Pop(3)
	return Response{Status: status, Headers: headersOut, Body: bodyOut}, nil
}

func callToolHandler(L *lua.LState, inv Invocation) (Response, error) {
	fn, err := handler(L, "handle_tool_call")
	if err != nil {
		return Response{}, err
	}
	args := toLua(L, inv.Args)
	if args == lua.LNil {
		args = L.NewTable()
	}
	if err := L.CallByParam(
		lua.P{Fn: fn, NRet: 1, Protect: true},
		lua.LString(inv.TaskID),
		lua.LString(inv.Tool),
		args,
	); err != nil {
		return Response{}, err
	}
	ret := L.Get(-1)
	L.Pop(1)
	switch value := ret.(type) {
	case *lua.LNilType:
		return Response{Body: nil}, nil
	case lua.LString:
		return Response{Body: []byte(value)}, nil
	default:
		raw, err := json.Marshal(fromLua(value))
		if err != nil {
			return Response{}, err
		}
		return Response{Body: raw}, nil
	}
}

func callSettingsHandler(L *lua.LState, inv Invocation) (Response, error) {
	fn, ok := L.GetGlobal("update_settings").(*lua.LFunction)
	if !ok {
		return Response{}, nil
	}
	if err := L.CallByParam(
		lua.P{Fn: fn, NRet: 1, Protect: true},
		lua.LString(inv.Scope),
		lua.LString(inv.TargetID),
		toLua(L, inv.Values),
		toLua(L, inv.Previous),
	); err != nil {
		return Response{}, err
	}
	ret := L.Get(-1)
	L.Pop(1)
	if ret == lua.LNil {
		return Response{}, nil
	}
	raw, err := json.Marshal(fromLua(ret))
	if err != nil {
		return Response{}, err
	}
	return Response{Body: raw}, nil
}

func callEventHandler(L *lua.LState, inv Invocation) (Response, error) {
	fn, err := handler(L, "handle_event")
	if err != nil {
		return Response{}, err
	}
	payload := toLua(L, inv.Body)
	if payload == lua.LNil {
		payload = L.NewTable()
	}
	payload.(*lua.LTable).RawSetString("event", lua.LString(inv.Event))
	payload.(*lua.LTable).RawSetString("task_id", lua.LString(inv.TaskID))
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, payload); err != nil {
		return Response{}, err
	}
	return Response{}, nil
}

func callProto(L *lua.LState, proto *lua.FunctionProto, nret int) error {
	fn := L.NewFunctionFromProto(proto)
	return L.CallByParam(lua.P{Fn: fn, NRet: nret, Protect: true})
}

func openSandbox(L *lua.LState, host *Host) error {
	lua.OpenBase(L)
	lua.OpenString(L)
	lua.OpenTable(L)
	lua.OpenMath(L)
	lua.OpenCoroutine(L)
	for _, name := range []string{
		"dofile", "loadfile", "load", "loadstring", "collectgarbage",
	} {
		L.SetGlobal(name, lua.LNil)
	}

	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, 0, L.GetTop())
		for i := 1; i <= L.GetTop(); i++ {
			parts = append(parts, L.Get(i).String())
		}
		host.log(LevelInfo, strings.Join(parts, " "), nil, nil)
		return 0
	}))

	L.SetGlobal("package", L.NewTable())
	pkg := L.GetGlobal("package").(*lua.LTable)
	pkg.RawSetString("loaded", L.NewTable())
	pkg.RawSetString("preload", L.NewTable())
	if err := L.DoString(requireSource); err != nil {
		return fmt.Errorf("install require: %w", err)
	}
	return nil
}

const requireSource = `
function require(name)
  local loaded = package.loaded[name]
  if loaded ~= nil then return loaded end
  local loader = package.preload[name]
  if loader == nil then
    error("module '" .. tostring(name) .. "' is not in the bundle", 2)
  end
  local result = loader(name)
  if result == nil then result = true end
  package.loaded[name] = result
  return result
end
`

func installModules(L *lua.LState, protos map[string]*lua.FunctionProto) {
	pkg := L.GetGlobal("package").(*lua.LTable)
	preload := pkg.RawGetString("preload").(*lua.LTable)
	names := make([]string, 0, len(protos))
	for name := range protos {
		if name != "main.lua" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		proto := protos[name]
		loader := L.NewFunction(func(L *lua.LState) int {
			fn := L.NewFunctionFromProto(proto)
			L.Push(fn)
			L.Push(L.Get(1))
			L.Call(1, 1)
			return 1
		})
		module := strings.TrimSuffix(name, ".lua")
		preload.RawSetString(strings.ReplaceAll(module, "/", "."), loader)
		preload.RawSetString(filepath.Base(module), loader)
	}
}

func toLua(L *lua.LState, value any) lua.LValue {
	switch v := value.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(v)
	case string:
		return lua.LString(v)
	case float64:
		return lua.LNumber(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case json.Number:
		if n, err := v.Float64(); err == nil {
			return lua.LNumber(n)
		}
		return lua.LString(v.String())
	case []byte:
		return lua.LString(string(v))
	case []string:
		t := L.NewTable()
		for _, item := range v {
			t.Append(lua.LString(item))
		}
		return t
	case []any:
		t := L.NewTable()
		for _, item := range v {
			t.Append(toLua(L, item))
		}
		return t
	case []map[string]any:

		t := L.NewTable()
		for _, item := range v {
			t.Append(toLua(L, item))
		}
		return t
	case map[string]any:
		t := L.NewTable()
		for key, item := range v {
			t.RawSetString(key, toLua(L, item))
		}
		return t
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return lua.LNil
		}
		var generic any
		if json.Unmarshal(raw, &generic) != nil {
			return lua.LNil
		}
		return toLua(L, generic)
	}
}

func fromLua(value lua.LValue) any {
	switch v := value.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		if isArray(v) {
			out := make([]any, 0, v.Len())
			for i := 1; i <= v.Len(); i++ {
				out = append(out, fromLua(v.RawGetInt(i)))
			}
			return out
		}
		out := map[string]any{}
		v.ForEach(func(key, item lua.LValue) {
			out[key.String()] = fromLua(item)
		})
		return out
	default:
		return v.String()
	}
}

func isArray(t *lua.LTable) bool {
	n := 0
	t.ForEach(func(_, _ lua.LValue) { n++ })
	if n == 0 {
		return false
	}
	for i := 1; i <= n; i++ {
		if t.RawGetInt(i) == lua.LNil {
			return false
		}
	}
	return true
}

func responseFromTable(value lua.LValue) (Response, error) {
	table, ok := value.(*lua.LTable)
	if !ok {
		return Response{Status: 200, Body: []byte(value.String())}, nil
	}
	raw := fromLua(table)
	fields, _ := raw.(map[string]any)
	out := Response{Status: 200}
	if v, ok := fields["status"].(float64); ok && v > 0 {
		out.Status = int(v)
	}
	if headers, ok := fields["headers"].(map[string]any); ok {
		out.Headers = map[string]string{}
		for key, value := range headers {
			out.Headers[key] = fmt.Sprint(value)
		}
	}
	switch body := fields["body"].(type) {
	case string:
		out.Body = []byte(body)
	case nil:
	default:
		encoded, err := json.Marshal(body)
		if err != nil {
			return Response{}, err
		}
		out.Body = encoded
		if out.Headers == nil {
			out.Headers = map[string]string{}
		}
		if _, ok := out.Headers["content-type"]; !ok {
			out.Headers["content-type"] = "application/json"
		}
	}
	return out, nil
}
