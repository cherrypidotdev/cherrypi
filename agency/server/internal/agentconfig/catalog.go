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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/sealed"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/variables"
)

var ErrInvalidConfig = errors.New("invalid model configuration")

var ErrNoSecretKey = sealed.ErrNoKey

var supportedAPIs = map[string]bool{
	"anthropic-messages":   true,
	"openai-completions":   true,
	"openai-responses":     true,
	"google-generative-ai": true,
	"pi-messages":          true,
}

const (
	ScopeOrg     = "org"
	ScopeAgent   = "agent"
	ScopeTeam    = "team"
	ScopeProject = "project"
	ScopeGroup   = "group"
)

type Service struct {
	store  store.AgentConfigStore
	sealer *sealed.Sealer
	now    func() time.Time

	variables VariableResolver
}

type VariableResolver interface {
	Resolve(ctx context.Context, caller variables.Caller) ([]variables.Resolved, error)
}

func (s *Service) SetVariables(resolver VariableResolver) { s.variables = resolver }

func New(st store.AgentConfigStore, secretKey string) (*Service, error) {
	seal, err := sealed.New(secretKey)
	if err != nil {
		return nil, err
	}
	return &Service{store: st, sealer: seal, now: time.Now}, nil
}

func (s *Service) Enabled() bool { return s.sealer.Enabled() }

type Model struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Reasoning bool     `json:"reasoning"`
	Input     []string `json:"input"`

	ThinkingLevels []string `json:"thinkingLevels,omitempty"`

	ForcePlan     bool      `json:"forcePlan,omitempty"`
	ContextWindow int64     `json:"contextWindow"`
	MaxTokens     int64     `json:"maxTokens"`
	Cost          ModelCost `json:"cost"`
}

var ThinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

func NormalizeThinkingLevels(levels []string) []string {
	if len(levels) == 0 {
		return []string{}
	}
	declared := make(map[string]bool, len(levels))
	for _, level := range levels {
		declared[level] = true
	}
	out := make([]string, 0, len(declared))
	for _, level := range ThinkingLevels {
		if declared[level] {
			out = append(out, level)
		}
	}
	return out
}

func PickerThinkingLevels(reasoning bool, levels []string) []string {
	if !reasoning {
		return []string{"off"}
	}
	if len(levels) == 0 {
		return append([]string(nil), ThinkingLevels[:5]...)
	}
	return NormalizeThinkingLevels(levels)
}

type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type Provider struct {
	Slug      string            `json:"slug"`
	Name      string            `json:"name"`
	API       string            `json:"api"`
	ProxyPath string            `json:"proxyPath"`
	Headers   map[string]string `json:"headers,omitempty"`
	Models    []Model           `json:"models"`
}

type Bundle struct {
	Revision  string     `json:"revision"`
	Providers []Provider `json:"providers"`

	Variables []BundleVariable `json:"variables,omitempty"`
}

type BundleVariable struct {
	Name   string `json:"name"`
	Note   string `json:"note,omitempty"`
	Secret bool   `json:"secret"`
	Value  string `json:"value"`
}

type Scope struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	ModelID string `json:"modelId,omitempty"`
}

type Caller struct {
	TaskScoped bool

	AgentActorID string

	ProjectID string
	TeamIDs   []string

	GroupIDs []string
}

func scopesOf(caller Caller) []Scope {
	scopes := []Scope{{Kind: ScopeOrg}}
	if caller.AgentActorID != "" {
		scopes = append(scopes, Scope{Kind: ScopeAgent, ID: caller.AgentActorID})
	}
	if caller.ProjectID != "" {
		scopes = append(scopes, Scope{Kind: ScopeProject, ID: caller.ProjectID})
	}
	for _, team := range caller.TeamIDs {
		if team != "" {
			scopes = append(scopes, Scope{Kind: ScopeTeam, ID: team})
		}
	}
	for _, group := range caller.GroupIDs {
		if group != "" {
			scopes = append(scopes, Scope{Kind: ScopeGroup, ID: group})
		}
	}
	return scopes
}

func (s *Service) withGroups(ctx context.Context, caller Caller) (Caller, error) {
	if caller.AgentActorID == "" {
		return caller, nil
	}
	actorID, err := strconv.ParseInt(caller.AgentActorID, 10, 64)
	if err != nil {
		return caller, nil
	}
	groups, err := s.store.ListAgentGroupIDsForAgentActor(ctx, actorID)
	if err != nil {
		return caller, err
	}
	for _, group := range groups {
		caller.GroupIDs = append(caller.GroupIDs, strconv.FormatInt(group, 10))
	}
	return caller, nil
}

func scopedRevision(revision string, scopes []Scope, extra []string) string {
	keys := make([]string, 0, len(scopes)+len(extra))
	for _, scope := range scopes {
		keys = append(keys, scope.Kind+":"+scope.ID)
	}
	keys = append(keys, extra...)
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\x00")))
	return revision + ":" + fmt.Sprintf("%x", sum[:8])
}

func (s *Service) variableScopes(ctx context.Context, caller Caller) ([]string, error) {
	if caller.AgentActorID == "" {
		return nil, nil
	}
	actorID, err := strconv.ParseInt(caller.AgentActorID, 10, 64)
	if err != nil {
		return nil, nil
	}
	teams, err := s.store.ListTeamIDsForAgentActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(teams))
	for _, team := range teams {
		keys = append(keys, "variable-team:"+strconv.FormatInt(team, 10))
	}
	return keys, nil
}

func (s *Service) Bundle(
	ctx context.Context,
	caller Caller,
	knownRevision string,
) (*Bundle, bool, error) {
	caller, err := s.withGroups(ctx, caller)
	if err != nil {
		return nil, false, err
	}
	catalogRevision, err := s.store.AgentConfigRevision(ctx)
	if err != nil {
		return nil, false, err
	}
	scopes := scopesOf(caller)
	variableScopes, err := s.variableScopes(ctx, caller)
	if err != nil {
		return nil, false, err
	}
	revision := scopedRevision(catalogRevision, scopes, variableScopes)
	if knownRevision != "" && knownRevision == revision {
		return nil, false, nil
	}

	providers, err := s.store.ListModelProviders(ctx)
	if err != nil {
		return nil, false, err
	}
	models, err := s.store.ListModels(ctx)
	if err != nil {
		return nil, false, err
	}
	grants, err := s.store.ListModelGrants(ctx)
	if err != nil {
		return nil, false, err
	}

	granted := grantedKeys(grants, scopes)
	byProvider := map[int64][]store.Model{}
	for _, model := range models {
		if model.Enabled == 0 {
			continue
		}
		byProvider[model.ProviderID] = append(byProvider[model.ProviderID], model)
	}

	bundle := &Bundle{Revision: revision}
	for _, row := range providers {
		if row.Enabled == 0 || !supportedAPIs[row.Api] {
			continue
		}
		provider := Provider{
			Slug:      row.Slug,
			Name:      row.Name,
			API:       row.Api,
			ProxyPath: ProxyPrefix + "/" + row.Slug,
			Headers:   decodeMap(row.HeadersJson),
		}
		for _, model := range byProvider[row.ID] {

			if strings.TrimSpace(model.ModelID) == "" {
				continue
			}
			if !granted.allows(row.ID, model.ModelID) {
				continue
			}
			provider.Models = append(provider.Models, modelOf(model))
		}
		if len(provider.Models) > 0 {
			bundle.Providers = append(bundle.Providers, provider)
		}
	}
	if resolved, err := s.resolveVariables(ctx, caller); err != nil {
		return nil, false, err
	} else {
		bundle.Variables = resolved
	}
	return bundle, true, nil
}

func (s *Service) resolveVariables(
	ctx context.Context,
	caller Caller,
) ([]BundleVariable, error) {
	if s.variables == nil || !caller.TaskScoped || caller.AgentActorID == "" {
		return nil, nil
	}
	actorID, err := strconv.ParseInt(caller.AgentActorID, 10, 64)
	if err != nil {
		return nil, nil
	}
	resolved, err := s.variables.Resolve(ctx, variables.Caller{
		AgentActorID: actorID,
		ProjectID:    caller.ProjectID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]BundleVariable, 0, len(resolved))
	for _, variable := range resolved {
		out = append(out, BundleVariable{
			Name:   variable.Name,
			Note:   variable.Note,
			Secret: variable.Secret,
			Value:  variable.Value,
		})
	}
	return out, nil
}

type grantSet struct {
	providers map[int64]bool

	models map[int64]map[string]bool
}

func (g grantSet) allows(providerID int64, modelID string) bool {
	if g.providers[providerID] {
		return true
	}
	return g.models[providerID][modelID]
}

func grantedKeys(grants []store.ModelGrant, scopes []Scope) grantSet {
	set := grantSet{providers: map[int64]bool{}, models: map[int64]map[string]bool{}}
	covers := func(grant store.ModelGrant) bool {
		for _, scope := range scopes {
			if grant.ScopeKind == scope.Kind && grant.ScopeID == scope.ID {
				return true
			}
		}
		return false
	}
	for _, grant := range grants {
		if !covers(grant) {
			continue
		}
		if grant.ModelID == "" {
			set.providers[grant.ProviderID] = true
			continue
		}
		if set.models[grant.ProviderID] == nil {
			set.models[grant.ProviderID] = map[string]bool{}
		}
		set.models[grant.ProviderID][grant.ModelID] = true
	}
	return set
}

func modelOf(row store.Model) Model {
	input := decodeStrings(row.InputJson)
	if len(input) == 0 {
		input = []string{"text"}
	}
	return Model{
		ID:             row.ModelID,
		Name:           row.Name,
		Reasoning:      row.Reasoning != 0,
		Input:          input,
		ThinkingLevels: decodeStrings(row.ThinkingLevelsJson),
		ForcePlan:      row.ForcePlan != 0,
		ContextWindow:  row.ContextWindow,
		MaxTokens:      row.MaxTokens,
		Cost:           decodeCost(row.CostJson),
	}
}

func decodeMap(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := map[string]string{}
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return out
}

func decodeStrings(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return out
}

func decodeCost(raw string) ModelCost {
	var cost ModelCost
	if strings.TrimSpace(raw) == "" {
		return cost
	}
	_ = json.Unmarshal([]byte(raw), &cost)
	return cost
}

func encodeJSON(value any, fallback string) string {
	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return fallback
	}
	return string(raw)
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func (s *Service) Grants(
	ctx context.Context,
	caller Caller,
	slug, modelID string,
) (bool, error) {
	provider, err := s.store.GetModelProvider(ctx, slug)
	if err != nil {
		return false, nil
	}
	caller, err = s.withGroups(ctx, caller)
	if err != nil {
		return false, err
	}
	models, err := s.store.ListModels(ctx)
	if err != nil {
		return false, err
	}
	enabled := map[string]bool{}
	for _, model := range models {
		if model.ProviderID == provider.ID && model.Enabled != 0 {
			enabled[model.ModelID] = true
		}
	}
	grants, err := s.store.ListModelGrants(ctx)
	if err != nil {
		return false, err
	}
	set := grantedKeys(grants, scopesOf(caller))
	if modelID == "" {

		return len(enabled) > 0 &&
			(set.providers[provider.ID] || len(set.models[provider.ID]) > 0), nil
	}
	if !enabled[modelID] {
		return false, nil
	}
	return set.allows(provider.ID, modelID), nil
}
