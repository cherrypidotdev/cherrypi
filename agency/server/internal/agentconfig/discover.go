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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sky-valley/pi/ai"
)

const DiscoverPath = AdminPath + "/discover"

const discoverTimeout = 20 * time.Second

const discoverBodyLimit = 4 << 20

type DiscoverInput struct {
	Slug    string            `json:"slug"`
	API     string            `json:"api"`
	BaseURL string            `json:"baseUrl"`
	APIKey  string            `json:"apiKey"`
	Headers map[string]string `json:"headers"`
}

type DiscoverResult struct {
	Models []ModelInput `json:"models"`
}

type upstreamModel struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	DisplayName         string         `json:"display_name"`
	DisplayNameCamel    string         `json:"displayName"`
	ContextLength       *int64         `json:"context_length"`
	MaxModelLen         *int64         `json:"max_model_len"`
	MaxCompletionTokens *int64         `json:"max_completion_tokens"`
	InputTokenLimit     *int64         `json:"inputTokenLimit"`
	OutputTokenLimit    *int64         `json:"outputTokenLimit"`
	TopProvider         *upstreamTop   `json:"top_provider"`
	Architecture        *upstreamArch  `json:"architecture"`
	Pricing             map[string]any `json:"pricing"`
	Vision              bool           `json:"vision"`
	ReasoningEfforts    []string       `json:"reasoning_efforts"`
	SupportedMethods    []string       `json:"supportedGenerationMethods"`
}

type upstreamTop struct {
	ContextLength       *int64 `json:"context_length"`
	MaxCompletionTokens *int64 `json:"max_completion_tokens"`
}

type upstreamArch struct {
	InputModalities []string `json:"input_modalities"`
}

type upstreamList struct {
	Data   []upstreamModel `json:"data"`
	Models []upstreamModel `json:"models"`
}

func (s *Service) Discover(ctx context.Context, input DiscoverInput) (DiscoverResult, error) {
	input.Slug = strings.TrimSpace(input.Slug)
	input.API = strings.TrimSpace(input.API)
	input.BaseURL = strings.TrimSpace(input.BaseURL)
	if !supportedAPIs[input.API] {
		return DiscoverResult{}, fmt.Errorf(
			"%w: unsupported wire API %q",
			ErrInvalidConfig,
			input.API,
		)
	}
	if input.BaseURL == "" {
		return DiscoverResult{}, fmt.Errorf(
			"%w: a provider needs a base URL",
			ErrInvalidConfig,
		)
	}

	key := strings.TrimSpace(input.APIKey)
	headers := input.Headers
	if input.Slug != "" {

		if stored, err := s.store.GetModelProvider(ctx, input.Slug); err == nil {
			if key == "" {
				key, err = s.sealer.Open(stored.ApiKeySealed)
				if err != nil {
					return DiscoverResult{}, err
				}
			}
			if len(headers) == 0 {
				headers = decodeMap(stored.HeadersJson)
			}
		}
	}
	if key == "" {
		return DiscoverResult{}, fmt.Errorf(
			"%w: an API key is required to fetch models",
			ErrInvalidConfig,
		)
	}

	upstream, err := joinUpstream(input.BaseURL, "/models", "")
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, upstream, nil)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	applyHeaders(req.Header, headers)
	authorizeProbe(req, input.API, key)
	req.Header.Set("Accept", "application/json")

	res, err := (&http.Client{Timeout: discoverTimeout}).Do(req)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf(
			"%w: could not reach the provider",
			ErrInvalidConfig,
		)
	}
	defer func() { _ = res.Body.Close() }()

	// status line can reach the API response, never the body.
	raw, readErr := io.ReadAll(io.LimitReader(res.Body, discoverBodyLimit))
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return DiscoverResult{}, fmt.Errorf(
			"%w: provider answered %s",
			ErrInvalidConfig,
			strings.TrimSpace(res.Status),
		)
	}
	if readErr != nil {
		return DiscoverResult{}, fmt.Errorf(
			"%w: could not read the provider's model list",
			ErrInvalidConfig,
		)
	}

	var list upstreamList
	if err := json.Unmarshal(raw, &list); err != nil {

		return DiscoverResult{}, fmt.Errorf(
			"%w: the provider did not return a JSON model list: %v",
			ErrInvalidConfig,
			err,
		)
	}
	entries := list.Data
	if len(entries) == 0 {
		entries = list.Models
	}

	result := DiscoverResult{Models: []ModelInput{}}
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			id = strings.TrimPrefix(strings.TrimSpace(entry.Name), "models/")
		}
		if id == "" || seen[id] {
			continue
		}
		if !generatesContent(entry.SupportedMethods) {
			continue
		}
		seen[id] = true
		known := builtinModel(input.API, id)

		if known != nil && ai.GetModelType(known) != ai.ModelTypeChat {
			continue
		}
		result.Models = append(result.Models, discoveredModel(id, entry, known))
	}
	return result, nil
}

func discoveredModel(id string, entry upstreamModel, known *ai.Model) ModelInput {
	model := ModelInput{
		ID:            id,
		Name:          id,
		Input:         []string{"text"},
		ContextWindow: 0,
		MaxTokens:     0,

		Enabled: false,
	}

	name := firstNonEmpty(entry.DisplayName, entry.DisplayNameCamel)
	if name == "" && !strings.HasPrefix(entry.Name, "models/") {
		name = strings.TrimSpace(entry.Name)
	}
	if name != "" {
		model.Name = name
	}
	if upper := entry.InputTokenLimit; upper != nil && *upper > 0 {
		model.ContextWindow = *upper
	}
	if output := entry.OutputTokenLimit; output != nil && *output > 0 {
		model.MaxTokens = *output
	}
	if entry.ContextLength != nil && *entry.ContextLength > 0 {
		model.ContextWindow = *entry.ContextLength
	}
	if entry.MaxModelLen != nil && *entry.MaxModelLen > 0 {
		model.ContextWindow = *entry.MaxModelLen
	}
	if entry.MaxCompletionTokens != nil && *entry.MaxCompletionTokens > 0 {
		model.MaxTokens = *entry.MaxCompletionTokens
	}
	if top := entry.TopProvider; top != nil {
		if top.ContextLength != nil && *top.ContextLength > 0 {
			model.ContextWindow = *top.ContextLength
		}
		if top.MaxCompletionTokens != nil && *top.MaxCompletionTokens > 0 {
			model.MaxTokens = *top.MaxCompletionTokens
		}
	}
	upstreamInput := false
	if arch := entry.Architecture; arch != nil && len(arch.InputModalities) > 0 {
		if normalized := normalizeInput(arch.InputModalities); len(normalized) > 0 {
			model.Input = normalized
			upstreamInput = true
		}
	}

	if !upstreamInput && entry.Vision {
		model.Input = []string{"text", "image"}
		upstreamInput = true
	}

	model.Reasoning = len(entry.ReasoningEfforts) > 0
	model.ThinkingLevels = thinkingLevelsFor(entry.ReasoningEfforts, known)
	model.Cost = costFromUpstream(entry.Pricing)

	if known != nil {
		if model.Name == id && known.Name != "" {
			model.Name = known.Name
		}
		model.Reasoning = model.Reasoning || known.Reasoning
		if model.ContextWindow == 0 {
			model.ContextWindow = int64(known.ContextWindow)
		}
		if model.MaxTokens == 0 {
			model.MaxTokens = int64(known.MaxTokens)
		}
		if !upstreamInput && len(known.Input) > 0 {
			model.Input = append([]string(nil), known.Input...)
		}
		if len(model.ThinkingLevels) == 0 && known.Reasoning {
			model.ThinkingLevels = builtinThinkingLevels(known)
		}
		if model.Cost == (ModelCost{}) {
			model.Cost = ModelCost{
				Input:      known.Cost.Input,
				Output:     known.Cost.Output,
				CacheRead:  known.Cost.CacheRead,
				CacheWrite: known.Cost.CacheWrite,
			}
		}
	}

	if model.ContextWindow <= 0 {
		model.ContextWindow = 128000
	}
	if model.MaxTokens <= 0 {
		model.MaxTokens = 8192
	}
	if len(model.Input) == 0 {
		model.Input = []string{"text"}
	}
	return model
}

var effortAliases = map[string]string{
	"none":    "off",
	"off":     "off",
	"disable": "off",
	"minimal": "minimal",
	"min":     "minimal",
	"low":     "low",
	"medium":  "medium",
	"med":     "medium",
	"high":    "high",
	"xhigh":   "xhigh",
	"max":     "max",
}

func thinkingLevelsFor(efforts []string, known *ai.Model) []string {
	if len(efforts) == 0 {
		if known == nil || !known.Reasoning {
			return nil
		}
		return builtinThinkingLevels(known)
	}
	levels := make([]string, 0, len(efforts))
	for _, raw := range efforts {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}

		level, ok := effortAliases[token]
		if !ok {
			continue
		}
		levels = append(levels, level)
	}
	if len(levels) == 0 {

		return nil
	}
	return NormalizeThinkingLevels(levels)
}

func builtinThinkingLevels(known *ai.Model) []string {
	supported := ai.GetSupportedThinkingLevels(known)
	out := make([]string, 0, len(supported))
	for _, level := range supported {
		out = append(out, string(level))
	}
	return out
}

func builtinModel(api, id string) *ai.Model {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	for _, provider := range catalogProviders(api) {
		if known := ai.GetModel(provider, id); known != nil {
			return known
		}
	}
	for _, provider := range ai.GetProviders() {
		if known := ai.GetModel(provider, id); known != nil {
			return known
		}
	}
	return nil
}

func catalogProviders(api string) []string {
	switch api {
	case "anthropic-messages":
		return []string{"anthropic", "openrouter"}
	case "openai-completions", "openai-responses":
		return []string{"openai", "openrouter", "azure", "github-copilot"}
	case "google-generative-ai":
		return []string{"google", "google-vertex", "openrouter"}
	default:
		return []string{"openrouter"}
	}
}

func costFromUpstream(pricing map[string]any) ModelCost {

	currency := strings.ToUpper(strings.TrimSpace(priceText(pricing["currency"])))
	if currency != "" && currency != "USD" {
		return ModelCost{}
	}
	perToken := func(keys ...string) float64 {
		for _, key := range keys {
			if value, ok := priceValue(pricing[key]); ok && value >= 0 {
				return value * 1_000_000
			}
		}
		return 0
	}
	cost := ModelCost{
		Input:      perToken("prompt", "input"),
		Output:     perToken("completion", "output"),
		CacheRead:  perToken("input_cache_read", "cache_read"),
		CacheWrite: perToken("input_cache_write", "cache_write"),
	}

	perMillion := func(keys ...string) (float64, bool) {
		for _, key := range keys {
			if value, ok := priceValue(pricing[key]); ok && value >= 0 {
				return value, true
			}
		}
		return 0, false
	}
	if value, ok := perMillion("input_per_m", "input_per_million"); ok {
		cost.Input = value
	}
	if value, ok := perMillion("output_per_m", "output_per_million"); ok {
		cost.Output = value
	}
	if value, ok := perMillion("cached_input_per_m", "cache_read_per_m"); ok {
		cost.CacheRead = value
	}
	if value, ok := perMillion("cache_write_per_m"); ok {
		cost.CacheWrite = value
	}
	return cost
}

func priceValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func priceText(value any) string {
	text, _ := value.(string)
	return text
}

func normalizeInput(modalities []string) []string {
	out := make([]string, 0, len(modalities))
	for _, modality := range modalities {
		switch strings.ToLower(strings.TrimSpace(modality)) {
		case "text", "image":
			out = append(out, strings.ToLower(strings.TrimSpace(modality)))
		}
	}
	return out
}

func generatesContent(methods []string) bool {
	if len(methods) == 0 {
		return true
	}
	for _, method := range methods {
		if strings.EqualFold(strings.TrimSpace(method), "generateContent") {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
