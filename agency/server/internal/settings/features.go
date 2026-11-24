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

package settings

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const keyGitSourcesEnabled = "git_sources_enabled"

var ErrSettingNotFound = errors.New("setting not found")

func (s *Service) GitSourcesEnabled(ctx context.Context) (bool, error) {
	value, err := s.store.GetSetting(ctx, keyGitSourcesEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, nil
	}
	return enabled, nil
}

func (s *Service) SetGitSourcesEnabled(ctx context.Context, enabled bool) error {
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   keyGitSourcesEnabled,
		Value: strconv.FormatBool(enabled),
	})
}
