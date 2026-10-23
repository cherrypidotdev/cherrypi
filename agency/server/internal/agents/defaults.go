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

package agents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

// ErrInvalidDefaults is returned when a model selection sets only half of the
// provider/id pair.
var ErrInvalidDefaults = errors.New("model provider and id must be set together")

// Defaults is the model and reasoning level a new task starts with when the
// task itself does not pin one. Empty fields fall back to Pi's configured
// defaults.
type Defaults struct {
	ModelProvider string
	ModelID       string
	ModelName     string
	ThinkingLevel string
}

// Keys in the agency's key-value settings table. The agent id scopes each key,
// so the table stays a single flat namespace.
const defaultsKeyPrefix = "agent_defaults:"

func defaultsKey(agentID ids.AgentID, field string) string {
	return defaultsKeyPrefix + strconv.FormatInt(int64(agentID), 10) + ":" + field
}

// Defaults returns the agent's configured default model/thinking pair. An
// unset value is empty, not an error.
func (r *Registry) Defaults(ctx context.Context, agentID ids.AgentID) (Defaults, error) {
	if _, err := r.store.GetAgent(ctx, agentID); errors.Is(err, sql.ErrNoRows) {
		return Defaults{}, ErrAgentMissing
	} else if err != nil {
		return Defaults{}, err
	}

	out := Defaults{}
	for field, target := range map[string]*string{
		"model_provider": &out.ModelProvider,
		"model_id":       &out.ModelID,
		"model_name":     &out.ModelName,
		"thinking_level": &out.ThinkingLevel,
	} {
		value, err := r.store.GetSetting(ctx, defaultsKey(agentID, field))
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return Defaults{}, err
		}
		*target = value
	}
	return out, nil
}

// SetDefaults replaces an agent's default model/thinking pair. A model must be
// selected as a provider/id pair; clearing both falls back to Pi's default.
func (r *Registry) SetDefaults(
	ctx context.Context,
	agentID ids.AgentID,
	defaults Defaults,
) (Defaults, error) {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return Defaults{}, ErrAgentMissing
	}
	if err != nil {
		return Defaults{}, err
	}

	next := Defaults{
		ModelProvider: strings.TrimSpace(defaults.ModelProvider),
		ModelID:       strings.TrimSpace(defaults.ModelID),
		ModelName:     strings.TrimSpace(defaults.ModelName),
		ThinkingLevel: strings.TrimSpace(defaults.ThinkingLevel),
	}
	if (next.ModelProvider == "") != (next.ModelID == "") {
		return Defaults{}, ErrInvalidDefaults
	}
	if next.ModelProvider == "" {
		next.ModelName = ""
	}

	before, err := r.Defaults(ctx, agentID)
	if err != nil {
		return Defaults{}, err
	}
	for field, value := range map[string]string{
		"model_provider": next.ModelProvider,
		"model_id":       next.ModelID,
		"model_name":     next.ModelName,
		"thinking_level": next.ThinkingLevel,
	} {
		if err := r.store.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   defaultsKey(agentID, field),
			Value: value,
		}); err != nil {
			return Defaults{}, err
		}
	}
	r.logAgent(ctx, agent.ActorID, audit.KindDefaults, "defaults",
		defaultsLabel(before), defaultsLabel(next),
		"Updated the default model and thinking level", nil)
	return next, nil
}

func defaultsLabel(defaults Defaults) string {
	model := "Pi default"
	if defaults.ModelProvider != "" && defaults.ModelID != "" {
		model = fmt.Sprintf("%s/%s", defaults.ModelProvider, defaults.ModelID)
	}
	level := defaults.ThinkingLevel
	if level == "" {
		level = "model default"
	}
	return model + " · " + level
}
