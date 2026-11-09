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

package plugins

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

type TaskBadge struct {
	Key string `json:"key"`

	Label string `json:"label"`

	Value string `json:"value,omitempty"`

	Tone string `json:"tone,omitempty"`

	URL string `json:"url,omitempty"`

	PluginID   string `json:"-"`
	PluginName string `json:"-"`
}

const (
	ToneNeutral = "neutral"
	ToneInfo    = "info"
	ToneSuccess = "success"
	ToneWarning = "warning"
	ToneDanger  = "danger"
)

const (
	maxTaskBadges   = 8
	maxBadgeLabel   = 48
	maxBadgeValue   = 96
	maxBadgeURL     = 2048
	maxBadgeMessage = 512
)

type StoredBadges struct {
	TaskID   string
	PluginID string
	Badges   []TaskBadge
}

func (s *Service) SetTaskBadges(
	ctx context.Context,
	pluginID, taskID string,
	badges []TaskBadge,
) error {
	p, err := s.Get(ctx, pluginID)
	if err != nil {
		return err
	}
	if !p.Enabled {
		return errors.New("plugin is disabled")
	}
	normalized, err := normalizeBadges(badges)
	if err != nil {
		return err
	}

	var before []TaskBadge
	for _, badge := range s.TaskBadges(ctx, taskID) {
		if badge.PluginID == pluginID {
			before = append(before, badge)
		}
	}
	if err := s.Store.SetTaskBadges(ctx, taskID, pluginID, normalized); err != nil {
		return err
	}
	if sameBadges(before, normalized) {
		return nil
	}
	if s.Tasks == nil {
		return nil
	}
	previous := badgeSummary(before)
	current := badgeSummary(normalized)
	logCtx := tasks.WithActor(ctx, tasks.ActorPlugin)
	logCtx = tasks.WithActorID(logCtx, ids.ActorID(p.ActorID))
	s.Tasks.AppendLog(
		logCtx, taskID, tasks.LogPlugin, "badges", previous, current,
		badgeMessage(p.Name, current),
		map[string]any{
			"plugin_id": p.ID,
			"plugin":    p.Name,
			"badges":    normalized,
		},
	)
	return nil
}

func (s *Service) TaskBadges(ctx context.Context, taskID string) []TaskBadge {
	return s.TaskBadgesFor(ctx, []string{taskID})[taskID]
}

func (s *Service) TaskBadgesFor(
	ctx context.Context,
	taskIDs []string,
) map[string][]TaskBadge {
	if s == nil || s.Store == nil || !s.Store.ready() || len(taskIDs) == 0 {
		return nil
	}
	rows, err := s.Store.TaskBadges(ctx, taskIDs)
	if err != nil {
		log.Printf("[plugins] task badges: %v", err)
		return nil
	}
	names := map[string]string{}
	out := map[string][]TaskBadge{}
	for _, row := range rows {
		name, ok := names[row.PluginID]
		if !ok {
			if p, err := s.Get(ctx, row.PluginID); err == nil {
				name = p.Name
			}
			names[row.PluginID] = name
		}
		for _, badge := range row.Badges {
			badge.PluginID = row.PluginID
			badge.PluginName = name
			out[row.TaskID] = append(out[row.TaskID], badge)
		}
	}
	return out
}

func normalizeBadges(badges []TaskBadge) ([]TaskBadge, error) {
	if len(badges) > maxTaskBadges {
		return nil, fmt.Errorf("at most %d badges are allowed", maxTaskBadges)
	}
	seen := map[string]bool{}
	out := make([]TaskBadge, 0, len(badges))
	for _, badge := range badges {
		badge.Key = strings.TrimSpace(badge.Key)
		badge.Label = strings.TrimSpace(badge.Label)
		badge.Value = strings.TrimSpace(badge.Value)
		badge.Tone = strings.ToLower(strings.TrimSpace(badge.Tone))
		badge.URL = strings.TrimSpace(badge.URL)
		if badge.Key == "" {
			return nil, errors.New("a badge needs a key")
		}
		if seen[badge.Key] {
			return nil, fmt.Errorf("badge key %q is duplicated", badge.Key)
		}
		seen[badge.Key] = true
		if badge.Label == "" {
			return nil, fmt.Errorf("badge %q needs a label", badge.Key)
		}
		badge.Label = truncate(badge.Label, maxBadgeLabel)
		badge.Value = truncate(badge.Value, maxBadgeValue)
		switch badge.Tone {
		case "", ToneNeutral:
			badge.Tone = ToneNeutral
		case ToneInfo, ToneSuccess, ToneWarning, ToneDanger:
		default:
			return nil, fmt.Errorf("unknown badge tone %q", badge.Tone)
		}

		if badge.Value == "" {
			badge.Tone = ToneNeutral
		}
		if badge.URL != "" {
			if len(badge.URL) > maxBadgeURL {
				return nil, fmt.Errorf("badge %q url is too long", badge.Key)
			}
			if !strings.HasPrefix(badge.URL, "https://") &&
				!strings.HasPrefix(badge.URL, "http://") {
				return nil, fmt.Errorf("badge %q url must be http(s)", badge.Key)
			}
		}
		out = append(out, badge)
	}
	return out, nil
}

func sameBadges(a, b []TaskBadge) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Label != b[i].Label ||
			a[i].Value != b[i].Value || a[i].Tone != b[i].Tone ||
			a[i].URL != b[i].URL {
			return false
		}
	}
	return true
}

func badgeSummary(badges []TaskBadge) string {
	parts := make([]string, 0, len(badges))
	for _, badge := range badges {
		part := badge.Label
		if badge.Value != "" {
			part += ": " + badge.Value
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " · ")
}

// badgeMessage is the change-log line a badge update records.
func badgeMessage(plugin, summary string) string {
	message := "Updated " + plugin + " badges"
	if summary != "" {
		message += ": " + summary
	}
	return truncate(message, maxBadgeMessage)
}

func truncate(value string, n int) string {
	if len(value) <= n {
		return value
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut]
}
