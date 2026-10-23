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

package accounts

import (
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Preference keys stored per actor.
const (
	// PreferenceNotifyFirstArchive controls the first-time archive notice: a
	// modal warning that archiving removes the task's Pi session and workspace.
	// It defaults to true; the modal's "do not show again" clears it.
	PreferenceNotifyFirstArchive = "notify_first_archive"
)

// Preferences is the per-user settings view. Known fields are booleans with
// documented defaults; an unset key yields the default.
type Preferences struct {
	NotifyFirstArchive bool
}

// DefaultPreferences is the preference set a user has before any write.
func DefaultPreferences() Preferences {
	return Preferences{NotifyFirstArchive: true}
}

// GetPreferences reads a user's settings, applying defaults for unset keys.
func (s *Service) GetPreferences(ctx context.Context, actorID ids.ActorID) (Preferences, error) {
	rows, err := s.store.ListActorSettings(ctx, actorID)
	if err != nil {
		return Preferences{}, err
	}
	prefs := DefaultPreferences()
	for _, row := range rows {
		switch row.Key {
		case PreferenceNotifyFirstArchive:
			prefs.NotifyFirstArchive = row.Value != "0" && row.Value != "false"
		}
	}
	return prefs, nil
}

// UpdatePreferences writes a user's settings.
func (s *Service) UpdatePreferences(
	ctx context.Context,
	actorID ids.ActorID,
	prefs Preferences,
) error {
	return s.store.UpsertActorSetting(ctx, store.UpsertActorSettingParams{
		ActorID: int64(actorID),
		Key:     PreferenceNotifyFirstArchive,
		Value:   boolString(prefs.NotifyFirstArchive),
	})
}
