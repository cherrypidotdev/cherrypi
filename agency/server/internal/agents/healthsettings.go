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
	"strconv"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	MinHealthInterval = 10 * time.Second
	MaxHealthInterval = 24 * time.Hour
)

var ErrInvalidHealthInterval = errors.New(
	"health interval must be 0 (disabled) or between 10s and 24h",
)

type HealthSettings struct {
	Configured bool
	Interval   time.Duration
}

func (s HealthSettings) Seconds() *int64 {
	if !s.Configured {
		return nil
	}
	seconds := int64(s.Interval / time.Second)
	return &seconds
}

func (s HealthSettings) Label() string {
	if !s.Configured {
		return "agent default"
	}
	if s.Interval <= 0 {
		return "disabled"
	}
	return s.Interval.String()
}

const healthIntervalKeyPrefix = "agent_health_interval:"

func healthIntervalKey(agentID ids.AgentID) string {
	return healthIntervalKeyPrefix + strconv.FormatInt(int64(agentID), 10)
}

func (r *Registry) HealthSettings(
	ctx context.Context,
	agentID ids.AgentID,
) (HealthSettings, error) {
	if _, err := r.store.GetAgent(ctx, agentID); errors.Is(err, sql.ErrNoRows) {
		return HealthSettings{}, ErrAgentMissing
	} else if err != nil {
		return HealthSettings{}, err
	}

	raw, err := r.store.GetSetting(ctx, healthIntervalKey(agentID))
	if errors.Is(err, sql.ErrNoRows) {
		return HealthSettings{}, nil
	}
	if err != nil {
		return HealthSettings{}, err
	}
	if raw == "" {
		return HealthSettings{}, nil
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds < 0 {

		return HealthSettings{}, nil
	}
	return HealthSettings{
		Configured: true,
		Interval:   time.Duration(seconds) * time.Second,
	}, nil
}

func (r *Registry) SetHealthSettings(
	ctx context.Context,
	agentID ids.AgentID,
	settings HealthSettings,
) (HealthSettings, error) {
	agent, err := r.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return HealthSettings{}, ErrAgentMissing
	}
	if err != nil {
		return HealthSettings{}, err
	}

	next := HealthSettings{Configured: settings.Configured}
	if next.Configured {
		next.Interval = settings.Interval.Truncate(time.Second)
		if next.Interval != 0 &&
			(next.Interval < MinHealthInterval || next.Interval > MaxHealthInterval) {
			return HealthSettings{}, ErrInvalidHealthInterval
		}
	}

	before, err := r.HealthSettings(ctx, agentID)
	if err != nil {
		return HealthSettings{}, err
	}

	value := ""
	if next.Configured {
		value = strconv.FormatInt(int64(next.Interval/time.Second), 10)
	}
	if err := r.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   healthIntervalKey(agentID),
		Value: value,
	}); err != nil {
		return HealthSettings{}, err
	}
	if before != next {
		r.logAgent(ctx, agent.ActorID, audit.KindHealthInterval, "interval",
			before.Label(), next.Label(),
			"Set the health sampling interval to "+next.Label(), nil)
	}
	return next, nil
}
