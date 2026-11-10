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
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	lua "github.com/yuin/gopher-lua"
)

type Host struct {
	Ctx        context.Context
	Plugin     Plugin
	Caller     Caller
	Invocation string

	TaskID   string
	Settings map[string]any
	DB       *PluginDB

	Bridge Bridge

	Fake *TestBridge

	AllowPrivateNetwork bool
	MaxResp             int64
	Client              *http.Client
}

func NewHost(
	ctx context.Context,
	p Plugin,
	caller Caller,
	invocation string,
	settings map[string]any,
	db *PluginDB,
	bridge Bridge,
	allowPrivateNetwork bool,
) *Host {
	var maxResp int64 = defaultRespLimit
	if p.Manifest.Limits.MaxResponse > 0 {
		maxResp = p.Manifest.Limits.MaxResponse
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &Host{
		Ctx:                 ctx,
		Plugin:              p,
		Caller:              caller,
		Invocation:          invocation,
		Settings:            settings,
		DB:                  db,
		Bridge:              bridge,
		AllowPrivateNetwork: allowPrivateNetwork,
		MaxResp:             maxResp,
		Client:              newHTTPClient(allowPrivateNetwork),
	}
}

func (h *Host) log(level, message string, data any, surfaces []LogSurface) {
	if h.Bridge == nil {
		return
	}
	h.Bridge.Log(
		context.Background(),
		h.Invocation,
		level,
		message,
		data,
		surfaces,
	)
}

func logSurfaces(L *lua.LState, value lua.LValue) ([]LogSurface, error) {
	if value == nil || value == lua.LNil {
		return nil, nil
	}
	table, ok := value.(*lua.LTable)
	if !ok {
		return nil, errors.New("log surfaces must be a table")
	}
	out := make([]LogSurface, 0, 4)
	table.ForEach(func(key, item lua.LValue) {
		if out == nil {
			return
		}
		scope := strings.ToLower(strings.TrimSpace(key.String()))
		switch scope {
		case ScopeOrg, ScopeProject, ScopeAgent, ScopeUser:
		default:
			out = nil
			return
		}
		target := strings.TrimSpace(item.String())
		if scope == ScopeOrg {
			target = ""
		}
		if target == "" && scope != ScopeOrg {
			out = nil
			return
		}
		out = append(out, LogSurface{Scope: scope, TargetID: target})
	})
	if out == nil {
		return nil, fmt.Errorf("log surfaces must use org, project, agent, or user with an id")
	}
	return out, nil
}

func (h *Host) attach(L *lua.LState) {
	root := L.NewTable()
	L.SetGlobal("cherrypi", root)

	logTable := L.NewTable()
	logTable.RawSetString("trace", h.logFunc(L, LevelTrace))
	logTable.RawSetString("debug", h.logFunc(L, LevelDebug))
	logTable.RawSetString("info", h.logFunc(L, LevelInfo))
	logTable.RawSetString("warn", h.logFunc(L, LevelWarn))
	logTable.RawSetString("error", h.logFunc(L, LevelError))
	root.RawSetString("log", logTable)

	jsonTable := L.NewTable()
	jsonTable.RawSetString("encode", L.NewFunction(func(L *lua.LState) int {
		value := fromLua(L.Get(1))
		raw, err := json.Marshal(value)
		if err != nil {
			L.RaiseError("json.encode: %v", err)
			return 0
		}
		L.Push(lua.LString(raw))
		return 1
	}))
	jsonTable.RawSetString("decode", L.NewFunction(func(L *lua.LState) int {
		var value any
		if err := json.Unmarshal([]byte(L.CheckString(1)), &value); err != nil {
			L.RaiseError("json.decode: %v", err)
			return 0
		}
		L.Push(toLua(L, value))
		return 1
	}))
	root.RawSetString("json", jsonTable)

	httpTable := L.NewTable()
	httpTable.RawSetString("request", L.NewFunction(h.httpRequest))
	root.RawSetString("http", httpTable)

	dbTable := L.NewTable()
	dbTable.RawSetString("query", L.NewFunction(h.dbQuery))
	dbTable.RawSetString("exec", L.NewFunction(h.dbExec))
	root.RawSetString("db", dbTable)

	cryptoTable := L.NewTable()
	cryptoTable.RawSetString("hmac_sha256", L.NewFunction(func(L *lua.LState) int {
		secret := L.CheckString(1)
		message := L.CheckString(2)
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(message))
		L.Push(lua.LString(hex.EncodeToString(mac.Sum(nil))))
		return 1
	}))
	cryptoTable.RawSetString("secure_equals", L.NewFunction(func(L *lua.LState) int {
		a := L.CheckString(1)
		b := L.CheckString(2)
		L.Push(lua.LBool(subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1))
		return 1
	}))
	root.RawSetString("crypto", cryptoTable)

	timeTable := L.NewTable()
	timeTable.RawSetString("now_ms", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(time.Now().UnixMilli()))
		return 1
	}))
	root.RawSetString("time", timeTable)

	actor := L.NewTable()
	actor.RawSetString("kind", lua.LString(h.Caller.Kind))
	actor.RawSetString("actor_id", lua.LNumber(h.Caller.ActorID))
	actor.RawSetString("username", lua.LString(h.Caller.Username))
	actor.RawSetString("admin", lua.LBool(h.Caller.Admin))
	root.RawSetString("actor", actor)

	settingsTable := h.settingsTable(L)
	root.RawSetString("settings", settingsTable)

	h.attachAgency(L, root)
}

func (h *Host) logFunc(L *lua.LState, level string) *lua.LFunction {
	return L.NewFunction(func(L *lua.LState) int {
		message := L.OptString(1, "")
		var data any
		if L.GetTop() >= 2 {
			data = fromLua(L.Get(2))
		}
		surfaces, err := logSurfaces(L, L.Get(3))
		if err != nil {
			L.RaiseError("log: %v", err)
			return 0
		}
		h.log(level, message, data, surfaces)
		return 0
	})
}

func (h *Host) settingsTable(L *lua.LState) *lua.LTable {
	table := L.NewTable()

	meta := L.NewTable()
	meta.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		if value, ok := h.Settings[key]; ok {
			L.Push(toLua(L, value))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))
	L.SetMetatable(table, meta)
	get := L.NewFunction(func(L *lua.LState) int {
		scope := lua.LString(ScopeOrg).String()
		if L.GetTop() >= 1 {
			scope = L.CheckString(1)
		}
		target := ""
		if L.GetTop() >= 2 {
			target = L.OptString(2, "")
		}
		row, err := h.Bridge.Settings(h.Ctx, scope, target)
		if err != nil {
			L.RaiseError("settings.get: %v", err)
			return 0
		}
		values := row.Values
		L.Push(toLua(L, values))
		return 1
	})
	table.RawSetString("get", get)
	return table
}

func (h *Host) httpRequest(L *lua.LState) int {
	method := strings.ToUpper(L.CheckString(1))
	rawURL := L.CheckString(2)
	opts := L.OptTable(3, L.NewTable())

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		L.RaiseError("http.request: invalid url")
		return 0
	}
	if parsed.Scheme != "https" {
		if !h.AllowPrivateNetwork || parsed.Scheme != "http" {
			L.RaiseError("http.request: only https is allowed")
			return 0
		}
	}
	if !h.Plugin.Manifest.NetworkAllowed(parsed.Hostname()) {
		L.RaiseError("http.request: host %q is not in the manifest allowlist", parsed.Hostname())
		return 0
	}
	if h.Fake != nil {
		out, err := h.Fake.HTTP(method, rawURL, opts)
		if err != nil {
			L.RaiseError("http.request: %v", err)
			return 0
		}
		L.Push(toLua(L, out))
		return 1
	}

	var body io.Reader
	if raw := opts.RawGetString("body"); raw != lua.LNil {
		body = strings.NewReader(raw.String())
	}
	timeout := 20 * time.Second
	if ms := opts.RawGetString("timeout_ms"); ms != lua.LNil {
		if n, ok := ms.(lua.LNumber); ok && n > 0 {
			timeout = time.Duration(n) * time.Millisecond
		}
	}
	ctx, cancel := context.WithTimeout(h.Ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		L.RaiseError("http.request: %v", err)
		return 0
	}
	if headers, ok := opts.RawGetString("headers").(*lua.LTable); ok {
		headers.ForEach(func(key, value lua.LValue) {
			req.Header.Set(key.String(), value.String())
		})
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		L.RaiseError("http.request: %v", err)
		return 0
	}
	defer func() { _ = resp.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, h.MaxResp+1))
	if err != nil {
		L.RaiseError("http.request: %v", err)
		return 0
	}
	if int64(len(payload)) > h.MaxResp {
		L.RaiseError("http.request: response exceeds %d bytes", h.MaxResp)
		return 0
	}
	out := L.NewTable()
	out.RawSetString("status", lua.LNumber(resp.StatusCode))
	headers := L.NewTable()
	for key, values := range resp.Header {
		headers.RawSetString(strings.ToLower(key), lua.LString(strings.Join(values, ", ")))
	}
	out.RawSetString("headers", headers)
	out.RawSetString("body", lua.LString(string(payload)))
	h.log(
		LevelInfo,
		fmt.Sprintf("http %s %s -> %d", method, parsed.Host, resp.StatusCode),
		nil,
		nil,
	)
	L.Push(out)
	return 1
}

func (h *Host) dbQuery(L *lua.LState) int {
	if h.DB == nil {
		L.RaiseError("the plugin database is unavailable")
		return 0
	}
	sql := L.CheckString(1)
	args := luaArgs(L.OptTable(2, L.NewTable()))
	rows, err := h.DB.Query(h.Ctx, sql, args...)
	if err != nil {
		L.RaiseError("db.query: %v", err)
		return 0
	}
	L.Push(toLua(L, rows))
	return 1
}

func (h *Host) dbExec(L *lua.LState) int {
	if h.DB == nil {
		L.RaiseError("the plugin database is unavailable")
		return 0
	}
	sql := L.CheckString(1)
	args := luaArgs(L.OptTable(2, L.NewTable()))
	n, err := h.DB.Exec(h.Ctx, sql, args...)
	if err != nil {
		L.RaiseError("db.exec: %v", err)
		return 0
	}
	out := L.NewTable()
	out.RawSetString("rows_affected", lua.LNumber(n))
	L.Push(out)
	return 1
}

func luaArgs(table *lua.LTable) []any {
	out := make([]any, 0, table.Len())
	for i := 1; i <= table.Len(); i++ {
		value := fromLua(table.RawGetInt(i))
		if number, ok := value.(float64); ok && number == float64(int64(number)) {
			out = append(out, int64(number))
			continue
		}
		out = append(out, value)
	}
	return out
}

func newHTTPClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			if allowPrivate {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return errors.New("could not resolve the address")
			}
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
				ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
				return errors.New("private addresses are not allowed")
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			ResponseHeaderTimeout: 15 * time.Second,
			MaxIdleConns:          4,
		},
	}
}
