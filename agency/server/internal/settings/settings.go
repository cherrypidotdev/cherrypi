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
	"errors"
	"regexp"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	keyAgencyName           = "agency_name"
	keyPrimaryColor         = "primary_color"
	keyShowUsageKPI         = "show_usage_kpi"
	keyShowConcurrencyKPI   = "show_concurrency_kpi"
	keyNotificationSoundSet = "notification_sound_set"

	KeyAgencyLogoAttachmentID    = "agency_logo_attachment_id"
	KeyAgencyFaviconAttachmentID = "agency_favicon_attachment_id"
)

var ErrInvalidColor = errors.New("primary color must be a lowercase #rrggbb")

var ErrInvalidSoundSet = errors.New("unknown notification sound set")

const DefaultNotificationSoundSet = "cherry"

var notificationSoundSets = map[string]bool{
	"classic": true,
	"cherry":  true,
	"wood":    true,
	"glass":   true,
	"arcade":  true,
}

var colorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

type Settings struct {
	AgencyName   string
	PrimaryColor string

	ShowUsageKPI       bool
	ShowConcurrencyKPI bool

	NotificationSoundSet string

	LogoAttachmentID    int64
	FaviconAttachmentID int64
}

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

func (s *Service) Get(ctx context.Context) (Settings, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return Settings{}, err
	}

	out := Settings{
		ShowUsageKPI:         true,
		ShowConcurrencyKPI:   true,
		NotificationSoundSet: DefaultNotificationSoundSet,
	}
	for _, row := range rows {
		switch row.Key {
		case keyAgencyName:
			out.AgencyName = row.Value
		case keyPrimaryColor:
			out.PrimaryColor = row.Value
		case keyShowUsageKPI:
			out.ShowUsageKPI = parseBoolDefault(row.Value)
		case keyShowConcurrencyKPI:
			out.ShowConcurrencyKPI = parseBoolDefault(row.Value)
		case keyNotificationSoundSet:
			if notificationSoundSets[row.Value] {
				out.NotificationSoundSet = row.Value
			}
		case KeyAgencyLogoAttachmentID:
			out.LogoAttachmentID = parseAttachmentID(row.Value)
		case KeyAgencyFaviconAttachmentID:
			out.FaviconAttachmentID = parseAttachmentID(row.Value)
		}
	}
	return out, nil
}

func parseBoolDefault(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return true
	}
	return parsed
}

func parseAttachmentID(value string) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

func (s *Service) Update(ctx context.Context, in Settings) (Settings, error) {
	name := strings.TrimSpace(in.AgencyName)
	color := strings.ToLower(strings.TrimSpace(in.PrimaryColor))
	soundSet := strings.TrimSpace(in.NotificationSoundSet)

	if color != "" && !colorPattern.MatchString(color) {
		return Settings{}, ErrInvalidColor
	}
	if soundSet == "" {
		soundSet = DefaultNotificationSoundSet
	}
	if !notificationSoundSets[soundSet] {
		return Settings{}, ErrInvalidSoundSet
	}

	current, err := s.Get(ctx)
	if err != nil {
		return Settings{}, err
	}

	if err := s.store.UpsertSetting(
		ctx,
		store.UpsertSettingParams{Key: keyAgencyName, Value: name},
	); err != nil {
		return Settings{}, err
	}
	if err := s.store.UpsertSetting(
		ctx,
		store.UpsertSettingParams{Key: keyPrimaryColor, Value: color},
	); err != nil {
		return Settings{}, err
	}
	if err := s.store.UpsertSetting(
		ctx,
		store.UpsertSettingParams{
			Key:   keyShowUsageKPI,
			Value: strconv.FormatBool(in.ShowUsageKPI),
		},
	); err != nil {
		return Settings{}, err
	}
	if err := s.store.UpsertSetting(
		ctx,
		store.UpsertSettingParams{
			Key:   keyShowConcurrencyKPI,
			Value: strconv.FormatBool(in.ShowConcurrencyKPI),
		},
	); err != nil {
		return Settings{}, err
	}
	if err := s.store.UpsertSetting(
		ctx,
		store.UpsertSettingParams{
			Key:   keyNotificationSoundSet,
			Value: soundSet,
		},
	); err != nil {
		return Settings{}, err
	}

	return Settings{
		AgencyName:           name,
		PrimaryColor:         color,
		ShowUsageKPI:         in.ShowUsageKPI,
		ShowConcurrencyKPI:   in.ShowConcurrencyKPI,
		NotificationSoundSet: soundSet,
		LogoAttachmentID:     current.LogoAttachmentID,
		FaviconAttachmentID:  current.FaviconAttachmentID,
	}, nil
}
