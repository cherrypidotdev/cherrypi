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
	"context"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	PreferenceNotifyFirstArchive = "notify_first_archive"

	PreferenceLocale = "locale"
)

var SupportedLocales = []string{"en", "ru"}

var ErrInvalidLocale = errors.New("unsupported locale")

type Preferences struct {
	NotifyFirstArchive bool

	Locale string
}

func DefaultPreferences() Preferences {
	return Preferences{NotifyFirstArchive: true}
}

func ValidateLocale(locale string) error {
	if locale == "" {
		return nil
	}
	for _, supported := range SupportedLocales {
		if locale == supported {
			return nil
		}
	}
	return ErrInvalidLocale
}

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
		case PreferenceLocale:
			if ValidateLocale(row.Value) == nil {
				prefs.Locale = row.Value
			}
		}
	}
	return prefs, nil
}

func (s *Service) UpdatePreferences(
	ctx context.Context,
	actorID ids.ActorID,
	prefs Preferences,
) error {
	if err := ValidateLocale(prefs.Locale); err != nil {
		return err
	}
	for _, setting := range []store.UpsertActorSettingParams{
		{
			ActorID: int64(actorID),
			Key:     PreferenceNotifyFirstArchive,
			Value:   boolString(prefs.NotifyFirstArchive),
		},
		{
			ActorID: int64(actorID),
			Key:     PreferenceLocale,
			Value:   prefs.Locale,
		},
	} {
		if err := s.store.UpsertActorSetting(ctx, setting); err != nil {
			return err
		}
	}
	return nil
}
