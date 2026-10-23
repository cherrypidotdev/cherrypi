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

package settingsgrpc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
	"cherrypi.vladfaust.com/agent/internal/store"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
)

// Keys in the `settings` table backing Settings' typed fields.
const (
	agentNameKey    = "agent_name"
	primaryColorKey = "primary_color"
)

// maxAgentNameLen caps the sidebar's agent name.
const maxAgentNameLen = 100

// hexColorRe matches a `#rrggbb` primary color.
var hexColorRe = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// SettingsServer serves the dashboard's agent configuration.
type SettingsServer struct {
	Store store.Store
}

func (s *SettingsServer) GetSettings(
	ctx context.Context,
	_ *agentv1.GetSettingsRequest,
) (*agentv1.GetSettingsResponse, error) {
	settings, err := s.read(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &agentv1.GetSettingsResponse{Settings: settings}, nil
}

func (s *SettingsServer) UpdateSettings(
	ctx context.Context,
	req *agentv1.UpdateSettingsRequest,
) (*agentv1.UpdateSettingsResponse, error) {
	if req.Settings == nil {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("settings are required"),
		)
	}

	name := strings.TrimSpace(req.Settings.AgentName)
	if r := []rune(name); len(r) > maxAgentNameLen {
		name = string(r[:maxAgentNameLen])
	}

	color := strings.ToLower(strings.TrimSpace(req.Settings.PrimaryColor))
	if color != "" && !hexColorRe.MatchString(color) {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("primary_color must be a #rrggbb hex color"),
		)
	}

	for key, value := range map[string]string{
		agentNameKey:    name,
		primaryColorKey: color,
	} {
		if err := s.Store.UpsertSetting(ctx, sqlitedb.UpsertSettingParams{
			Key:   key,
			Value: value,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	settings, err := s.read(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &agentv1.UpdateSettingsResponse{Settings: settings}, nil
}

// read loads the key-value settings and maps them onto the typed message.
func (s *SettingsServer) read(ctx context.Context) (*agentv1.Settings, error) {
	rows, err := s.Store.ListSettings(ctx)
	if err != nil {
		return nil, err
	}

	settings := &agentv1.Settings{}
	for _, row := range rows {
		switch row.Key {
		case agentNameKey:
			settings.AgentName = row.Value
		case primaryColorKey:
			settings.PrimaryColor = row.Value
		}
	}
	return settings, nil
}
