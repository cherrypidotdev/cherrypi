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

package session

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sky-valley/pi/ai"
)

type AgencyModels struct {
	Revision  string           `json:"revision"`
	Providers []AgencyProvider `json:"providers"`

	Variables []AgencyVariable `json:"variables"`

	MCP    AgencyMCP    `json:"mcp"`
	Skills AgencySkills `json:"skills"`
}

type AgencyVariable struct {
	Name   string `json:"name"`
	Note   string `json:"note"`
	Secret bool   `json:"secret"`
	Value  string `json:"value"`
}

type AgencyCatalog interface {
	Fetch(ctx context.Context, taskID, knownRevision string) (AgencyModels, bool, error)
}

const taskHeader = "X-CherryPi-Task"

const modelHeader = "X-CherryPi-Model"

type AgencyProvider struct {
	Slug      string            `json:"slug"`
	Name      string            `json:"name"`
	API       string            `json:"api"`
	ProxyPath string            `json:"proxyPath"`
	Headers   map[string]string `json:"headers"`
	Models    []AgencyModel     `json:"models"`
}

type AgencyModel struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Reasoning     bool     `json:"reasoning"`
	Input         []string `json:"input"`
	ContextWindow int      `json:"contextWindow"`
	MaxTokens     int      `json:"maxTokens"`

	CompactionReserveTokens    int             `json:"compactionReserveTokens"`
	CompactionKeepRecentTokens int             `json:"compactionKeepRecentTokens"`
	Cost                       AgencyModelCost `json:"cost"`

	ThinkingLevels []string `json:"thinkingLevels"`

	ForcePlan bool `json:"forcePlan"`

	Overrides json.RawMessage `json:"overrides"`
}

type AgencyModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

var supportedAPIs = map[ai.Api]bool{
	ai.APIAnthropicMessages:  true,
	ai.APIOpenAICompletions:  true,
	ai.APIOpenAIResponses:    true,
	ai.APIGoogleGenerativeAI: true,
	ai.APIPiMessages:         true,
}

func applyAgencyModels(
	catalog *modelCatalog,
	declared AgencyModels,
	agencyURL, token, taskID string,
) {
	for _, provider := range declared.Providers {
		apiID := ai.Api(provider.API)
		if !supportedAPIs[apiID] {
			continue
		}
		streams, ok := ai.GetApiProvider(apiID)
		if !ok {
			continue
		}
		baseURL := strings.TrimRight(agencyURL, "/") + provider.ProxyPath
		if baseURL == "" || provider.ProxyPath == "" {
			continue
		}

		for _, model := range provider.Models {
			if strings.TrimSpace(model.ID) == "" {
				continue
			}
			catalog.declareModel(
				provider.Slug,
				model.ID,
				model.ForcePlan,
				model.CompactionReserveTokens,
				model.CompactionKeepRecentTokens,
			)
		}

		headers := provider.Headers
		if taskID != "" {

			merged := make(map[string]string, len(headers)+1)
			for key, value := range headers {
				merged[key] = value
			}
			merged[taskHeader] = taskID
			headers = merged
		}
		built := buildModels(
			provider.Slug,
			apiID,
			provider.Models,
			baseURL,
			headers,
			taskID != "",
			catalog.donors,
		)
		if len(built) == 0 {
			continue
		}

		for _, model := range built {
			model.Headers = mergeHeaderValues(model.Headers, proxyAuthHeaders(token))
		}
		mergeAgencyProvider(catalog, provider, built, apiID, streams, token)
	}
}

func mergeAgencyProvider(
	catalog *modelCatalog,
	provider AgencyProvider,
	built []*ai.Model,
	apiID ai.Api,
	streams ai.ApiProvider,
	token string,
) {
	existing := catalog.models.GetProvider(provider.Slug)
	if existing == nil {
		catalog.models.SetProvider(ai.CreateProvider(ai.CreateProviderOptions{
			ID:       provider.Slug,
			Name:     provider.Name,
			BaseURL:  firstBaseURL(built),
			Auth:     bearerAuth(provider.Name, token, "agent access token"),
			Models:   built,
			APIByApi: apiStreams(apiID, streams),
		}))
		return
	}

	name := provider.Name
	if name == "" {
		name = existing.Name()
	}
	catalog.models.SetProvider(ai.CreateProvider(ai.CreateProviderOptions{
		ID:       provider.Slug,
		Name:     name,
		BaseURL:  firstBaseURL(append(existing.GetModels(), built...)),
		Auth:     bearerAuth(name, token, "agent access token"),
		Models:   mergeModelsByID(existing.GetModels(), built),
		APIByApi: apiStreamsForModels(append(existing.GetModels(), built...)),
	}))
}

func mergeModelsByID(known, agency []*ai.Model) []*ai.Model {
	agencyByID := make(map[string]*ai.Model, len(agency))
	for _, model := range agency {
		agencyByID[model.ID] = model
	}
	merged := make([]*ai.Model, 0, len(known)+len(agency))
	for _, model := range known {
		if replacement, ok := agencyByID[model.ID]; ok {
			merged = append(merged, mergeModelCapabilities(model, replacement))
			delete(agencyByID, model.ID)
			continue
		}
		merged = append(merged, model)
	}
	for _, model := range agency {
		if _, ok := agencyByID[model.ID]; !ok {
			continue
		}
		merged = append(merged, model)
		delete(agencyByID, model.ID)
	}
	return merged
}

func mergeModelCapabilities(builtin, agency *ai.Model) *ai.Model {
	if agency == nil {
		return builtin
	}
	out := *agency
	if out.Type == "" {
		out.Type = builtin.Type
	}
	out.ThinkingLevelMap = mergeThinkingLevelMap(
		builtin.ThinkingLevelMap, out.ThinkingLevelMap,
	)
	if out.InputLimits == nil {
		out.InputLimits = builtin.InputLimits
	}
	if out.PromptCache == nil {
		out.PromptCache = builtin.PromptCache
	}
	if out.SamplingParams == nil {
		out.SamplingParams = builtin.SamplingParams
	}
	if out.SamplingParamsByThinkingLevel == nil {
		out.SamplingParamsByThinkingLevel = builtin.SamplingParamsByThinkingLevel
	}
	if len(out.Compat) == 0 {
		out.Compat = builtin.Compat
		if out.Provider != builtin.Provider {
			out.Compat = withoutInheritedEndpointCompat(out.Compat)
		}
	}
	return &out
}

var inheritedEndpointCompat = []string{
	"thinkingFormat",
	"supportsMidConvoToolAdditions",
	"supportsMidConvoSystemMessages",
}

func withoutInheritedEndpointCompat(compat json.RawMessage) json.RawMessage {
	if len(compat) == 0 {
		return compat
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(compat, &fields); err != nil {
		return compat
	}
	changed := false
	for _, key := range inheritedEndpointCompat {
		if _, ok := fields[key]; !ok {
			continue
		}
		delete(fields, key)
		changed = true
	}
	if !changed {
		return compat
	}
	stripped, err := json.Marshal(fields)
	if err != nil {
		return compat
	}
	return stripped
}

func mergeThinkingLevelMap(
	builtin, agency ai.ThinkingLevelMap,
) ai.ThinkingLevelMap {
	if agency == nil {
		return builtin
	}
	if builtin == nil {
		return agency
	}
	out := make(ai.ThinkingLevelMap, len(agency))
	for level, value := range agency {
		if value == nil {
			out[level] = nil
			continue
		}
		if mapped := builtin[level]; mapped != nil {
			out[level] = mapped
			continue
		}
		out[level] = value
	}
	return out
}

func firstBaseURL(models []*ai.Model) string {
	for _, model := range models {
		if model.BaseURL != "" {
			return model.BaseURL
		}
	}
	return ""
}

func apiStreamsForModels(models []*ai.Model) map[ai.Api]ai.ProviderStreams {
	out := map[ai.Api]ai.ProviderStreams{}
	for _, model := range models {
		if !supportedAPIs[model.Api] {
			continue
		}
		if _, ok := out[model.Api]; ok {
			continue
		}
		if streams, ok := ai.GetApiProvider(model.Api); ok {
			out[model.Api] = ai.ProviderStreams{
				Stream:         streams.Stream,
				StreamSimple:   streams.StreamSimple,
				FetchDeferred:  streams.FetchDeferred,
				CancelDeferred: streams.CancelDeferred,
			}
		}
	}
	return out
}

func proxyAuthHeaders(token string) ai.ProviderHeaders {
	if token == "" {
		return nil
	}
	bearer := "Bearer " + token
	return ai.ProviderHeaders{
		"Authorization":  &bearer,
		"X-Api-Key":      &token,
		"X-Goog-Api-Key": &token,
	}
}

func mergeHeaderValues(base, override ai.ProviderHeaders) ai.ProviderHeaders {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(ai.ProviderHeaders, len(base)+len(override))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range override {
		out[key] = value
	}
	return out
}

func buildModels(
	slug string,
	apiID ai.Api,
	models []AgencyModel,
	baseURL string,
	headers map[string]string,
	withModelHeader bool,
	donors map[string]*ai.Model,
) []*ai.Model {
	built := make([]*ai.Model, 0, len(models))
	for _, model := range models {
		if strings.TrimSpace(model.ID) == "" {
			continue
		}
		input := model.Input
		if len(input) == 0 {
			input = []string{"text"}
		}
		modelHeaders := headers
		if withModelHeader {

			merged := make(map[string]string, len(headers)+1)
			for key, value := range headers {
				merged[key] = value
			}
			merged[modelHeader] = model.ID
			modelHeaders = merged
		}
		built = append(built, withBuiltinCapabilities(&ai.Model{
			ID:               model.ID,
			Name:             model.Name,
			Api:              apiID,
			Provider:         ai.ProviderId(slug),
			BaseURL:          baseURL,
			Reasoning:        model.Reasoning,
			ThinkingLevelMap: modelThinkingLevelMap(model),
			Input:            input,
			ContextWindow:    model.ContextWindow,
			MaxTokens:        model.MaxTokens,
			Cost: ai.ModelCost{
				Input:      model.Cost.Input,
				Output:     model.Cost.Output,
				CacheRead:  model.Cost.CacheRead,
				CacheWrite: model.Cost.CacheWrite,
			},
			Headers: providerHeaders(modelHeaders),
		}, donors[donorKey(apiID, model.ID)]))
	}
	return built
}

func withBuiltinCapabilities(model, donor *ai.Model) *ai.Model {
	if donor == nil || donor.Api != model.Api {
		return model
	}
	return mergeModelCapabilities(donor, model)
}

func builtinDonors(models ai.Models) map[string]*ai.Model {
	donors := map[string]*ai.Model{}
	for _, model := range models.GetModels("") {
		key := donorKey(model.Api, model.ID)
		if donors[key] == nil {
			donors[key] = model
		}
	}
	return donors
}

func donorKey(apiID ai.Api, modelID string) string {
	return string(apiID) + "\x00" + modelID
}

func apiStreams(apiID ai.Api, streams ai.ApiProvider) map[ai.Api]ai.ProviderStreams {
	return map[ai.Api]ai.ProviderStreams{apiID: {
		Stream:         streams.Stream,
		StreamSimple:   streams.StreamSimple,
		FetchDeferred:  streams.FetchDeferred,
		CancelDeferred: streams.CancelDeferred,
	}}
}

func bearerAuth(name, token, source string) ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.ApiKeyAuth{
		Name: name,
		Resolve: func(_ context.Context, _ ai.AuthContext, _ *ai.Credential) (*ai.AuthResult, error) {
			if token == "" {
				return nil, nil
			}
			return &ai.AuthResult{Auth: ai.ModelAuth{APIKey: token}, Source: source}, nil
		},
	}}
}

func modelThinkingLevelMap(model AgencyModel) ai.ThinkingLevelMap {
	return thinkingLevelMapFor(model.ThinkingLevels)
}

var canonicalThinkingLevels = []ai.ModelThinkingLevel{
	"off", "minimal", "low", "medium", "high", "xhigh", "max",
}

func thinkingLevelMapFor(levels []string) ai.ThinkingLevelMap {
	if len(levels) == 0 {
		return nil
	}
	accepted := make(map[ai.ModelThinkingLevel]bool, len(levels))
	for _, level := range levels {
		accepted[ai.ModelThinkingLevel(level)] = true
	}
	out := make(ai.ThinkingLevelMap, len(canonicalThinkingLevels))
	for _, level := range canonicalThinkingLevels {
		if !accepted[level] {
			out[level] = nil
			continue
		}
		value := string(level)
		out[level] = &value
	}
	return out
}

func providerHeaders(in map[string]string) ai.ProviderHeaders {
	if len(in) == 0 {
		return nil
	}
	out := make(ai.ProviderHeaders, len(in))
	for key, value := range in {
		out[key] = ai.HeaderValue(value)
	}
	return out
}
