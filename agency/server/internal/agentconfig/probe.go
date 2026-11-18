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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrUnknownProvider = errors.New("unknown provider")

const probeTimeout = 15 * time.Second

type ProbeResult struct {
	OK        bool   `json:"ok"`
	Status    int    `json:"status,omitempty"`
	LatencyMS int64  `json:"latencyMs"`
	Message   string `json:"message"`
}

func (s *Service) Probe(ctx context.Context, slug string) (ProbeResult, error) {
	provider, err := s.store.GetModelProvider(ctx, slug)
	if err != nil {
		return ProbeResult{}, ErrUnknownProvider
	}
	if !supportedAPIs[provider.Api] {
		return ProbeResult{}, fmt.Errorf(
			"%w: unsupported wire API %q",
			ErrInvalidConfig,
			provider.Api,
		)
	}
	key, err := s.sealer.Open(provider.ApiKeySealed)
	if err != nil {
		return ProbeResult{}, err
	}
	if key == "" {
		return ProbeResult{}, fmt.Errorf(
			"%w: a provider needs an API key the agency holds",
			ErrInvalidConfig,
		)
	}
	upstream, err := joinUpstream(provider.BaseUrl, "/models", "")
	if err != nil {
		return ProbeResult{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, upstream, nil)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	applyHeaders(req.Header, decodeMap(provider.HeadersJson))
	authorizeProbe(req, provider.Api, key)

	start := time.Now()
	res, err := (&http.Client{Timeout: probeTimeout}).Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{
			LatencyMS: latency,
			Message:   "could not reach the provider",
		}, nil
	}
	defer func() { _ = res.Body.Close() }()

	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))

	ok := res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices
	if !ok {
		return ProbeResult{
			Status:    res.StatusCode,
			LatencyMS: latency,
			Message:   fmt.Sprintf("provider answered %s", strings.TrimSpace(res.Status)),
		}, nil
	}
	return ProbeResult{
		OK:        true,
		Status:    res.StatusCode,
		LatencyMS: latency,
		Message:   "provider answered",
	}, nil
}

func authorizeProbe(req *http.Request, api, key string) {
	switch api {
	case "anthropic-messages":
		req.Header.Set("X-Api-Key", key)
		if req.Header.Get("Anthropic-Version") == "" {
			req.Header.Set("Anthropic-Version", "2023-06-01")
		}
	case "google-generative-ai":
		req.Header.Set("X-Goog-Api-Key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
	}
}
