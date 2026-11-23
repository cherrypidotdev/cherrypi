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

// name: the agent host's `~/.agents/mcp-settings.json`, then the agency's

// the agency denies). A server that fails to connect is a log line and a report

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type Value = string

type ToolFilter struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

const (
	TransportStdio = "stdio"

	TransportHTTP = "http"

	TransportAgency = "agency"
)

type Server struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Transport string `json:"transport"`

	Command          string            `json:"command"`
	Args             []string          `json:"args"`
	Cwd              string            `json:"cwd"`
	Env              map[string]string `json:"env"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers"`
	ConnectTimeoutMs int64             `json:"connectTimeoutMs"`
	RequestTimeoutMs int64             `json:"requestTimeoutMs"`
	Tools            ToolFilter        `json:"tools"`
	DirectTools      bool              `json:"directTools"`

	Origin string `json:"-"`

	Owner string `json:"owner"`

	ProxyPath string `json:"proxyPath"`

	AgencyURL   string `json:"-"`
	AgencyToken string `json:"-"`
	TaskID      string `json:"-"`
}

type Tool struct {
	Server      string
	Name        string
	Description string
	Schema      json.RawMessage
}

type Conn struct {
	server  Server
	client  *client.Client
	tools   []Tool
	closeFn func()
}

func (c *Conn) Tools() []Tool { return c.tools }

func (c *Conn) Server() Server { return c.server }

func (c *Conn) Call(
	ctx context.Context,
	name string,
	args map[string]any,
) (string, bool, error) {
	timeout := time.Duration(c.server.RequestTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := c.client.CallTool(callCtx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		return "", true, err
	}
	return renderResult(result), result.IsError, nil
}

func (c *Conn) Close() {
	if c.closeFn != nil {
		c.closeFn()
	}
}

func Connect(ctx context.Context, server Server) (*Conn, error) {
	timeout := time.Duration(server.ConnectTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		cli *client.Client
		err error
	)
	switch server.Transport {
	case TransportAgency, TransportHTTP:
		target := server.URL
		headers := map[string]string{}
		for key, value := range server.Headers {
			headers[key] = value
		}
		if server.Transport == TransportAgency {
			if server.AgencyURL == "" || server.AgencyToken == "" {
				return nil, fmt.Errorf("agency MCP server %q has no agency endpoint", server.Name)
			}
			target = strings.TrimRight(server.AgencyURL, "/") + server.ProxyPath

			headers = map[string]string{
				"Authorization":   "Bearer " + server.AgencyToken,
				"X-CherryPi-Task": server.TaskID,
			}
		}
		options := []transport.StreamableHTTPCOption{}
		if len(headers) > 0 {
			options = append(options, transport.WithHTTPHeaders(headers))
		}
		cli, err = client.NewStreamableHttpClient(target, options...)
		if err != nil {
			return nil, err
		}
		if err := cli.Start(connectCtx); err != nil {
			return nil, err
		}
	default:
		env := os.Environ()
		for key, value := range server.Env {
			env = append(env, key+"="+value)
		}
		cwd := server.Cwd
		cli, err = client.NewStdioMCPClientWithOptions(
			server.Command,
			env,
			server.Args,
			transport.WithCommandFunc(func(
				ctx context.Context,
				command string,
				env []string,
				args []string,
			) (*exec.Cmd, error) {
				cmd := exec.CommandContext(ctx, command, args...)
				cmd.Env = env
				if cwd != "" {
					cmd.Dir = cwd
				}
				return cmd, nil
			}),
		)
		if err != nil {
			return nil, err
		}
	}
	if _, err := cli.Initialize(connectCtx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "cherrypi-agent",
				Version: "1",
			},
		},
	}); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}
	listCtx, cancelList := context.WithTimeout(ctx, timeout)
	defer cancelList()
	result, err := cli.ListTools(listCtx, mcp.ListToolsRequest{})
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("list tools: %w", err)
	}
	conn := &Conn{
		server:  server,
		client:  cli,
		closeFn: func() { _ = cli.Close() },
	}
	for _, tool := range result.Tools {
		if !server.DirectTools || !allowedTool(tool.Name, server.Tools) {
			continue
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			continue
		}
		conn.tools = append(conn.tools, Tool{
			Server:      server.Name,
			Name:        tool.Name,
			Description: tool.Description,
			Schema:      schema,
		})
	}
	return conn, nil
}

func allowedTool(name string, filter ToolFilter) bool {
	for _, pattern := range filter.Exclude {
		if globMatch(pattern, name) {
			return false
		}
	}
	if len(filter.Include) == 0 {
		return true
	}
	for _, pattern := range filter.Include {
		if globMatch(pattern, name) {
			return true
		}
	}
	return false
}

func globMatch(pattern, name string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == name
	}
	var expr strings.Builder
	expr.WriteString("^")
	for i, part := range parts {
		if i > 0 {
			expr.WriteString(".*")
		}
		expr.WriteString(regexp.QuoteMeta(part))
	}
	expr.WriteString("$")
	matched, err := regexp.MatchString(expr.String(), name)
	return err == nil && matched
}

func renderResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		switch value := content.(type) {
		case mcp.TextContent:
			parts = append(parts, value.Text)
		case *mcp.TextContent:
			parts = append(parts, value.Text)
		default:
			raw, err := json.Marshal(content)
			if err != nil {
				continue
			}
			parts = append(parts, string(raw))
		}
	}
	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if text == "" && result.StructuredContent != nil {
		if raw, err := json.Marshal(result.StructuredContent); err == nil {
			text = string(raw)
		}
	}
	return text
}

// file is not an error; a malformed one is one log line and no servers.
func HostServers() []Server {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".agents", "mcp-settings.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("host mcp settings", "path", path, "error", err)
		}
		return nil
	}
	servers, err := parseSettings(raw)
	if err != nil {
		slog.Warn("host mcp settings", "path", path, "error", err)
		return nil
	}
	for i := range servers {
		servers[i].Origin = "host"
	}
	return servers
}

type hostSettings struct {
	Servers map[string]json.RawMessage `json:"mcpServers"`
}

type hostServer struct {
	Command          string            `json:"command"`
	Args             []string          `json:"args"`
	Cwd              string            `json:"cwd"`
	Env              map[string]string `json:"env"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers"`
	Disabled         bool              `json:"disabled"`
	Description      string            `json:"description"`
	ConnectTimeoutMs int64             `json:"connectTimeoutMs"`
	RequestTimeoutMs int64             `json:"requestTimeoutMs"`
	IncludeTools     []string          `json:"includeTools"`
	ExcludeTools     []string          `json:"excludeTools"`

	DirectTools json.RawMessage `json:"directTools"`
}

func directToolNames(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil
	}
	return names
}

func directTools(name string, raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var flag bool
	if err := json.Unmarshal(raw, &flag); err == nil {
		return flag
	}
	if names := directToolNames(raw); names != nil {
		return len(names) > 0
	}
	var mode string
	if err := json.Unmarshal(raw, &mode); err == nil && mode == "search" {
		slog.Warn("host mcp directTools", "server", name, "mode", mode,
			"detail", "codemode is not implemented; every tool is declared directly")
		return true
	}
	slog.Warn("host mcp directTools", "server", name,
		"detail", "unrecognised shape; every tool is declared directly")
	return true
}

func parseSettings(raw []byte) ([]Server, error) {
	var parsed hostSettings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(parsed.Servers))
	for name := range parsed.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Server, 0, len(names))
	for _, name := range names {
		var entry hostServer
		if err := json.Unmarshal(parsed.Servers[name], &entry); err != nil {
			slog.Warn("host mcp server", "server", name, "error", err)
			continue
		}
		if entry.Disabled {
			continue
		}
		transport := "stdio"
		if strings.TrimSpace(entry.URL) != "" {
			transport = "http"
		}
		direct := directTools(name, entry.DirectTools)
		include := entry.IncludeTools

		if names := directToolNames(entry.DirectTools); len(names) > 0 && len(include) == 0 {
			include = names
		}
		out = append(out, Server{
			Name:             name,
			Description:      entry.Description,
			Transport:        transport,
			Command:          entry.Command,
			Args:             entry.Args,
			Cwd:              entry.Cwd,
			Env:              entry.Env,
			URL:              entry.URL,
			Headers:          entry.Headers,
			ConnectTimeoutMs: entry.ConnectTimeoutMs,
			RequestTimeoutMs: entry.RequestTimeoutMs,
			Tools: ToolFilter{
				Include: include,
				Exclude: entry.ExcludeTools,
			},
			DirectTools: direct,
		})
	}
	return out, nil
}

func expandValue(value string) (string, bool) {
	if !strings.Contains(value, "${") {
		return value, true
	}
	var out strings.Builder
	rest := value
	for {
		start := strings.Index(rest, "${")
		if start < 0 {
			out.WriteString(rest)
			return out.String(), true
		}
		out.WriteString(rest[:start])
		end := strings.Index(rest[start:], "}")
		if end < 0 {
			out.WriteString(rest[start:])
			return out.String(), true
		}
		reference := rest[start+2 : start+end]
		name, fallback, hasFallback := strings.Cut(reference, ":-")
		if resolved, ok := os.LookupEnv(name); ok && resolved != "" {
			out.WriteString(resolved)
		} else if hasFallback {
			out.WriteString(fallback)
		} else {
			return "", false
		}
		rest = rest[start+end+1:]
	}
}

func (s Server) Resolve() (Server, bool) {
	env, ok := resolveValues(s.Env)
	if !ok {
		return Server{}, false
	}
	headers, ok := resolveValues(s.Headers)
	if !ok {
		return Server{}, false
	}
	s.Env = env
	s.Headers = headers
	return s, true
}

func resolveValues(values map[string]string) (map[string]string, bool) {
	out := make(map[string]string, len(values))
	for key, value := range values {
		expanded, ok := expandValue(value)
		if !ok {
			slog.Warn("mcp value unresolved", "key", key)
			return nil, false
		}
		out[key] = expanded
	}
	return out, true
}

func Merge(host, agency []Server, deny []string) []Server {
	denied := make(map[string]bool, len(deny))
	for _, name := range deny {
		denied[name] = true
	}
	out := make([]Server, 0, len(host)+len(agency))
	seen := map[string]bool{}
	for _, server := range agency {
		if denied[server.Name] {
			continue
		}
		seen[server.Name] = true
		out = append(out, server)
	}
	for _, server := range host {
		if denied[server.Name] || seen[server.Name] {
			continue
		}
		out = append(out, server)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
