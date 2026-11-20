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
	"fmt"
	"regexp"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,47}$`)

type ProviderInput struct {
	Slug     string            `json:"slug"`
	Name     string            `json:"name"`
	API      string            `json:"api"`
	BaseURL  string            `json:"baseUrl"`
	APIKey   string            `json:"apiKey"`
	ClearKey bool              `json:"clearKey"`
	Headers  map[string]string `json:"headers"`
	Enabled  bool              `json:"enabled"`
	Models   []ModelInput      `json:"models"`

	Grantees []Scope `json:"grantees"`
}

type ModelInput struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Reasoning bool     `json:"reasoning"`
	Input     []string `json:"input"`

	ThinkingLevels []string  `json:"thinkingLevels"`
	ContextWindow  int64     `json:"contextWindow"`
	MaxTokens      int64     `json:"maxTokens"`
	Cost           ModelCost `json:"cost"`
	Enabled        bool      `json:"enabled"`
}

func (s *Service) SaveProvider(ctx context.Context, input ProviderInput) error {
	input.Slug = strings.TrimSpace(input.Slug)
	if !slugPattern.MatchString(input.Slug) {
		return fmt.Errorf("%w: provider slug must be a lowercase slug", ErrInvalidConfig)
	}
	if !supportedAPIs[input.API] {
		return fmt.Errorf("%w: unsupported wire API %q", ErrInvalidConfig, input.API)
	}
	if strings.TrimSpace(input.BaseURL) == "" {
		return fmt.Errorf("%w: a provider needs a base URL", ErrInvalidConfig)
	}

	for _, model := range input.Models {
		if strings.TrimSpace(model.ID) == "" {
			return fmt.Errorf("%w: every model row needs an id", ErrInvalidConfig)
		}
		if err := validateThinkingLevels(model.ThinkingLevels); err != nil {
			return err
		}
	}
	for _, scope := range input.Grantees {
		if !validScope(scope) {
			return fmt.Errorf("%w: unknown grant scope %q", ErrInvalidConfig, scope.Kind)
		}
	}

	var sealed []byte
	switch {
	case input.ClearKey:
		sealed = nil
	case strings.TrimSpace(input.APIKey) != "":
		key, err := s.sealer.Seal(input.APIKey)
		if err != nil {
			return err
		}
		sealed = key
	default:

		if existing, err := s.store.GetModelProvider(ctx, input.Slug); err == nil {
			sealed = existing.ApiKeySealed
		}
	}
	if len(sealed) == 0 {
		return fmt.Errorf("%w: a provider needs an API key the agency holds", ErrInvalidConfig)
	}

	now := s.now().UnixMilli()
	save := store.SaveModelProviderParams{
		Provider: store.UpsertModelProviderParams{
			Slug:         input.Slug,
			Name:         strings.TrimSpace(input.Name),
			Api:          input.API,
			BaseUrl:      strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
			ApiKeySealed: sealed,
			HeadersJson:  encodeJSON(input.Headers, "{}"),
			Enabled:      boolInt(input.Enabled),
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		Revision: s.nextRevision(),
	}
	for _, model := range input.Models {
		save.Models = append(save.Models, store.UpsertModelParams{
			ModelID:            strings.TrimSpace(model.ID),
			Name:               strings.TrimSpace(model.Name),
			Reasoning:          boolInt(model.Reasoning),
			InputJson:          encodeJSON(model.Input, `["text"]`),
			ContextWindow:      model.ContextWindow,
			MaxTokens:          model.MaxTokens,
			CostJson:           encodeJSON(model.Cost, "{}"),
			ThinkingLevelsJson: encodeJSON(NormalizeThinkingLevels(model.ThinkingLevels), "[]"),
			Enabled:            boolInt(model.Enabled),
		})
	}

	grantees := input.Grantees
	if len(grantees) == 0 {
		grantees = []Scope{{Kind: ScopeOrg}}
	}
	for _, scope := range grantees {
		save.Grants = append(save.Grants, store.UpsertModelGrantParams{
			ModelID:   strings.TrimSpace(scope.ModelID),
			ScopeKind: scope.Kind,
			ScopeID:   scope.ID,
			CreatedAt: now,
		})
	}

	return s.store.SaveModelProvider(ctx, save)
}

func (s *Service) DeleteProvider(ctx context.Context, slug string) error {
	deleted, err := s.store.DeleteModelProvider(ctx, slug, s.nextRevision())
	if err != nil {
		return err
	}
	if !deleted {
		return ErrUnknownProvider
	}
	return nil
}

func validateThinkingLevels(levels []string) error {
	for _, level := range levels {
		if canonicalThinkingLevel[level] {
			continue
		}
		return fmt.Errorf(
			"%w: unknown thinking level %q (want one of %s)",
			ErrInvalidConfig,
			level,
			strings.Join(ThinkingLevels, ", "),
		)
	}
	return nil
}

var canonicalThinkingLevel = func() map[string]bool {
	out := make(map[string]bool, len(ThinkingLevels))
	for _, level := range ThinkingLevels {
		out[level] = true
	}
	return out
}()

func validScope(scope Scope) bool {
	switch scope.Kind {
	case ScopeOrg:
		return true
	case ScopeAgent, ScopeTeam, ScopeProject, ScopeGroup:
		return strings.TrimSpace(scope.ID) != ""
	default:
		return false
	}
}

func (s *Service) nextRevision() string {
	return store.NewRevision()
}
