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

package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const ProxyPrefix = "/api/rest/agent-llm"

const proxyTimeout = 30 * time.Minute

const maxProxyBody = 64 << 20

var (
	errProxyBodyTooLarge = errors.New("request body is too large")
	errModelMismatch     = errors.New("model header does not match the request")
	errModelRequired     = errors.New("a model request must name its model")
)

var hopHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade", "Host", "Content-Length",
	"Authorization", "X-Api-Key", "X-Goog-Api-Key",

	"X-CherryPi-Task", "X-CherryPi-Model",
}

type proxy struct {
	service *Service

	agentActorID func(context.Context) string

	grants func(ctx context.Context, caller Caller, slug, modelID string) (bool, error)

	taskScope TaskScope
	client    *http.Client
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug, rest := splitProxyPath(r.URL.Path)
	if slug == "" {
		writeError(w, http.StatusNotFound, "provider is required")
		return
	}
	agentActorID := ""
	if p.agentActorID != nil {
		agentActorID = p.agentActorID(r.Context())
	}
	if agentActorID == "" {
		writeError(w, http.StatusUnauthorized, "agent identity is required")
		return
	}
	provider, err := p.service.store.GetModelProvider(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "unknown provider")
		return
	}
	if provider.Enabled == 0 || !supportedAPIs[provider.Api] {
		writeError(w, http.StatusForbidden, "provider is disabled")
		return
	}

	caller := Caller{AgentActorID: agentActorID}
	if taskID := r.Header.Get(TaskHeader); taskID != "" {
		if p.taskScope == nil {
			writeError(w, http.StatusForbidden, "task-scoped grants are not available")
			return
		}
		projectID, teamIDs, ok := p.taskScope(r.Context(), agentActorID, taskID)
		if !ok {
			writeError(w, http.StatusForbidden, "this task does not belong to the calling agent")
			return
		}
		caller.ProjectID, caller.TeamIDs = projectID, teamIDs
	}

	body, err := readRequestBody(r)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	}
	modelID, err := requestModel(r, provider.Api, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if modelID == "" && r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, errModelRequired.Error())
		return
	}

	allowed, err := p.grants(r.Context(), caller, slug, modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "this agent may not use that provider")
		return
	}

	upstream, err := joinUpstream(provider.BaseUrl, rest, r.URL.RawQuery)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), proxyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, r.Method, upstream, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	copyHeaders(req.Header, r.Header)
	applyHeaders(req.Header, decodeMap(provider.HeadersJson))
	if rewritesContentBlocks(provider.Api) {
		// The rewrite reads the stream line by line, so it needs the bytes the

		req.Header.Del("Accept-Encoding")
	}
	if err := p.authorize(r.Context(), req, provider); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	client := p.client
	if client == nil {
		client = &http.Client{Timeout: proxyTimeout}
	}
	res, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer func() { _ = res.Body.Close() }()

	for key, values := range res.Header {
		if isHopHeader(key) {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	if rewritesContentBlocks(provider.Api) && sseContentType(res.Header) {
		if encoding := contentEncoding(res.Header); encoding != "" {
			// The stream is still encoded, so the line-by-line rewrite cannot

			log.Printf(
				"[agent-llm] %s answered a %s-encoded stream; the content-block rewrite did not run",
				slug,
				encoding,
			)
			_ = flushCopy(w, res.Body)
			return
		}
		_ = flushNormalized(w, res.Body)
		return
	}
	_ = flushCopy(w, res.Body)
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

func requestModel(r *http.Request, api string, body []byte) (string, error) {
	headerModel := strings.TrimSpace(r.Header.Get(ModelHeader))
	bodyModel, bodyHasModel := modelFromBody(body)
	pathModel := modelFromPath(r.URL.Path)
	if headerModel != "" && bodyHasModel && headerModel != bodyModel {
		return "", errModelMismatch
	}
	if api == "google-generative-ai" && pathModel != "" {
		if headerModel != "" && headerModel != pathModel {
			return "", errModelMismatch
		}
		if bodyHasModel && bodyModel != pathModel {
			return "", errModelMismatch
		}
		return pathModel, nil
	}
	if bodyHasModel {
		if bodyModel == "" {
			return "", errModelRequired
		}
		return bodyModel, nil
	}
	if pathModel != "" {
		if headerModel != "" && headerModel != pathModel {
			return "", errModelMismatch
		}
		return pathModel, nil
	}
	if len(body) > 0 && r.Method != http.MethodGet {

		return "", errModelRequired
	}
	return headerModel, nil
}

func modelFromBody(body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	raw, ok := payload["model"]
	if !ok {
		return "", false
	}
	var model string
	if err := json.Unmarshal(raw, &model); err != nil {
		return "", true
	}
	return strings.TrimSpace(model), true
}

func modelFromPath(path string) string {
	rest := strings.TrimPrefix(path, ProxyPrefix)
	if rest == path {
		return ""
	}
	index := strings.Index(rest, "/models/")
	if index < 0 {
		return ""
	}
	model := rest[index+len("/models/"):]
	if colon := strings.IndexByte(model, ':'); colon >= 0 {
		model = model[:colon]
	}
	return strings.TrimSpace(model)
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxProxyBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxProxyBody {
		return nil, errProxyBodyTooLarge
	}
	return body, nil
}

func (p *proxy) authorize(
	ctx context.Context,
	req *http.Request,
	provider store.ModelProvider,
) error {
	key, err := p.service.sealer.Open(provider.ApiKeySealed)
	if err != nil {
		return err
	}
	if key == "" {
		return errors.New("provider has no key stored in the agency")
	}
	switch provider.Api {
	case "anthropic-messages":
		req.Header.Set("X-Api-Key", key)
	case "google-generative-ai":
		req.Header.Set("X-Goog-Api-Key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return nil
}

func splitProxyPath(path string) (slug, rest string) {
	trimmed := strings.TrimPrefix(path, ProxyPrefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "", ""
	}
	if index := strings.IndexByte(trimmed, '/'); index >= 0 {
		return trimmed[:index], trimmed[index:]
	}
	return trimmed, ""
}

func joinUpstream(baseURL, rest, query string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", err
	}
	parsed.Path = joinUpstreamPath(
		strings.TrimRight(parsed.Path, "/"),
		strings.TrimPrefix(rest, "/"),
	)
	if query != "" {
		parsed.RawQuery = query
	}
	return parsed.String(), nil
}

func joinUpstreamPath(base, rest string) string {
	if base == "" {
		return "/" + rest
	}
	trimmedBase := strings.TrimSuffix(base, "/")
	if rest == "" {
		return trimmedBase + "/"
	}

	if slash := strings.LastIndex(trimmedBase, "/"); slash >= 0 {
		last := trimmedBase[slash+1:]
		first, remainder, found := strings.Cut(rest, "/")
		if last != "" && last == first {
			if !found || remainder == "" {
				return trimmedBase
			}
			return trimmedBase + "/" + remainder
		}
	}
	return trimmedBase + "/" + rest
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func applyHeaders(dst http.Header, headers map[string]string) {
	for key, value := range headers {
		dst.Set(key, value)
	}
}

func contentEncoding(header http.Header) string {
	encoding := strings.TrimSpace(header.Get("Content-Encoding"))
	if strings.EqualFold(encoding, "identity") {
		return ""
	}
	return encoding
}

func isHopHeader(key string) bool {
	for _, hop := range hopHeaders {
		if strings.EqualFold(hop, key) {
			return true
		}
	}
	return false
}

type Proxy struct {
	Service *Service

	TaskScope TaskScope
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	inner := &proxy{
		service:      p.Service,
		agentActorID: agentActorIDFromRequest,
		grants:       p.Service.Grants,
		taskScope:    p.TaskScope,
	}
	inner.ServeHTTP(w, r)
}

func agentActorIDFromRequest(ctx context.Context) string {
	id, ok := auth.IdentityFrom(ctx)
	if !ok || id.Kind != "agent" {
		return ""
	}
	return strconv.FormatInt(int64(id.ActorID), 10)
}
