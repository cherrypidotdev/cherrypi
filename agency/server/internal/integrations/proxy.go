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

package integrations

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const proxyTimeout = 30 * time.Minute

const mcpTaskHeader = "X-CherryPi-Task"

var mcpForwardedHeaders = []string{
	"Content-Type",
	"Accept",
	"Mcp-Session-Id",
	"Mcp-Protocol-Version",
	"Last-Event-ID",
}

type Proxy struct {
	Service *Service

	AgentActorID func(context.Context) string

	TaskScope TaskScope

	Groups func(ctx context.Context, agentActorID string) ([]string, error)

	Client *http.Client
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, rest := splitProxyPath(r.URL.Path)
	if name == "" {
		writeError(w, http.StatusNotFound, "an MCP server name is required")
		return
	}
	if rest != "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	agentActorID := ""
	if p.AgentActorID != nil {
		agentActorID = p.AgentActorID(r.Context())
	}
	if agentActorID == "" {
		writeError(w, http.StatusUnauthorized, "agent identity is required")
		return
	}
	taskID := strings.TrimSpace(r.Header.Get(mcpTaskHeader))
	if taskID == "" {

		writeError(w, http.StatusBadRequest, "an MCP call must name its task")
		return
	}
	if p.TaskScope == nil {
		writeError(w, http.StatusForbidden, "task-scoped MCP calls are not available")
		return
	}
	projectID, teamIDs, ok := p.TaskScope(r.Context(), agentActorID, taskID)
	if !ok {
		writeError(w, http.StatusForbidden, "this task does not belong to the calling agent")
		return
	}

	caller, err := p.Service.CallerFor(
		r.Context(), agentActorID, projectID, teamIDs, p.Groups,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	row, applies, err := p.Service.ApplicableServer(r.Context(), name, caller)
	if err != nil {
		if errors.Is(err, ErrNotFound) {

			writeError(w, http.StatusNotFound, "unknown MCP server")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !applies {
		writeError(w, http.StatusForbidden, "this task does not receive that MCP server")
		return
	}

	switch r.Method {
	case http.MethodPost, http.MethodGet, http.MethodDelete:
	default:
		w.Header().Set("Allow", "POST, GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	headers, err := p.Service.openValues(row.HeadersJson)
	if err != nil {

		var missing MissingEnvError
		if errors.As(err, &missing) {
			writeError(w, http.StatusBadGateway, missing.Error())
			return
		}
		slog.Error("mcp proxy could not open the entry's headers",
			"server", row.Name, "error", err)
		writeError(w, http.StatusBadGateway, "the entry's stored headers could not be read")
		return
	}

	timeout := time.Duration(row.RequestTimeoutMs) * time.Millisecond
	if timeout <= 0 || r.Method == http.MethodGet {

		timeout = proxyTimeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	upstream, err := url.Parse(row.Url)
	if err != nil {
		writeError(w, http.StatusBadGateway, "the entry's URL is invalid")
		return
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, upstream.String(), r.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if r.ContentLength > 0 {

		req.ContentLength = r.ContentLength
	}
	for _, key := range mcpForwardedHeaders {
		if values, ok := r.Header[key]; ok {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	for key, value := range headers {

		req.Header.Set(key, value)
	}

	client := p.Client
	if client == nil {
		client = &http.Client{}
	}
	res, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			writeError(w, http.StatusGatewayTimeout, "the MCP server did not answer in time")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer func() { _ = res.Body.Close() }()

	for key, values := range res.Header {
		if isProxyHopHeader(key) {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	if err := flushCopy(w, res.Body); err != nil && !errors.Is(err, context.Canceled) {
		slog.Debug("mcp proxy stream ended", "server", row.Name, "error", err)
	}
}

func splitProxyPath(path string) (string, string) {
	trimmed := strings.TrimPrefix(path, ProxyPrefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "", ""
	}
	name, rest, _ := strings.Cut(trimmed, "/")
	return name, rest
}

func isProxyHopHeader(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
		"Te", "Trailer", "Transfer-Encoding", "Upgrade", "Host", "Content-Length",

		"Authorization", "Cookie", "X-Cherrypi-Task", "X-Cherrypi-Model":
		return true
	default:
		return false
	}
}

func flushCopy(w http.ResponseWriter, src io.Reader) error {
	controller := http.NewResponseController(w)
	buffer := make([]byte, 32<<10)
	for {
		n, err := src.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			_ = controller.Flush()
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
