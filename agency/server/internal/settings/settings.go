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
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	keyAgencyName           = "agency_name"
	keyPrimaryColor         = "primary_color"
	keyNotificationSoundSet = "notification_sound_set"
	keyTaskArchiveEnabled   = "task_auto_archive_enabled"
	keyTaskArchiveDays      = "task_auto_archive_days"
	keyVoiceInputEnabled    = "voice_input_enabled"
	keySTTBaseURL           = "stt_base_url"
	keySTTModel             = "stt_model"
	keySTTAPIKey            = "stt_api_key"

	KeyAgencyLogoAttachmentID    = "agency_logo_attachment_id"
	KeyAgencyFaviconAttachmentID = "agency_favicon_attachment_id"
)

var ErrInvalidColor = errors.New("primary color must be a lowercase #rrggbb")

var ErrInvalidSoundSet = errors.New("unknown notification sound set")

var ErrInvalidSTTBaseURL = errors.New("speech-to-text base URL must be an absolute http(s) URL")

const DefaultSTTModel = "whisper-1"

const sttTranscriptionsPath = "/audio/transcriptions"

const DefaultNotificationSoundSet = "cherry"

const (
	DefaultTaskArchiveAfterDays = 30

	MaxTaskArchiveAfterDays = 3650
)

var ErrInvalidTaskArchiveDays = errors.New(
	"automatic archive delay must be between 1 and 3650 days",
)

var notificationSoundSets = map[string]bool{
	"classic": true,
	"cherry":  true,
	"wood":    true,
	"glass":   true,
	"arcade":  true,
}

var colorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

type TaskArchivePolicy struct {
	Enabled bool

	Days int
}

type Settings struct {
	AgencyName   string
	PrimaryColor string

	NotificationSoundSet string

	TaskArchive *TaskArchivePolicy

	VoiceInputEnabled bool

	STTBaseURL string

	STTModel string

	STTAPIKey string

	ClearSTTAPIKey bool

	LogoAttachmentID    int64
	FaviconAttachmentID int64
}

func (s Settings) ServerTranscription() bool {
	return s.VoiceInputEnabled && s.STTBaseURL != "" && s.STTModel != ""
}

func (s Settings) HasSTTAPIKey() bool { return s.STTAPIKey != "" }

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
		NotificationSoundSet: DefaultNotificationSoundSet,
		TaskArchive: &TaskArchivePolicy{
			Enabled: true,
			Days:    DefaultTaskArchiveAfterDays,
		},
		STTModel: DefaultSTTModel,
	}
	for _, row := range rows {
		switch row.Key {
		case keyAgencyName:
			out.AgencyName = row.Value
		case keyPrimaryColor:
			out.PrimaryColor = row.Value
		case keyNotificationSoundSet:
			if notificationSoundSets[row.Value] {
				out.NotificationSoundSet = row.Value
			}
		case keyTaskArchiveEnabled:
			out.TaskArchive.Enabled = parseBool(row.Value, true)
		case keyTaskArchiveDays:
			if days, err := strconv.Atoi(strings.TrimSpace(row.Value)); err == nil &&
				days >= 1 && days <= MaxTaskArchiveAfterDays {
				out.TaskArchive.Days = days
			}
		case keyVoiceInputEnabled:
			out.VoiceInputEnabled = parseBool(row.Value, false)
		case keySTTBaseURL:
			out.STTBaseURL = strings.TrimSpace(row.Value)
		case keySTTModel:
			if model := strings.TrimSpace(row.Value); model != "" {
				out.STTModel = model
			}
		case keySTTAPIKey:
			out.STTAPIKey = row.Value
		case KeyAgencyLogoAttachmentID:
			out.LogoAttachmentID = parseAttachmentID(row.Value)
		case KeyAgencyFaviconAttachmentID:
			out.FaviconAttachmentID = parseAttachmentID(row.Value)
		}
	}
	return out, nil
}

func parseBool(value string, def bool) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return def
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

func normalizeSTTBaseURL(raw string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return "", nil
	}
	trimmed = strings.TrimSuffix(trimmed, sttTranscriptionsPath)
	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", ErrInvalidSTTBaseURL
	}

	if parsed.User != nil {
		return "", ErrInvalidSTTBaseURL
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func (s *Service) Update(ctx context.Context, in Settings) (Settings, error) {
	name := strings.TrimSpace(in.AgencyName)
	color := strings.ToLower(strings.TrimSpace(in.PrimaryColor))
	soundSet := strings.TrimSpace(in.NotificationSoundSet)
	baseURL, err := normalizeSTTBaseURL(in.STTBaseURL)
	if err != nil {
		return Settings{}, err
	}
	model := strings.TrimSpace(in.STTModel)
	if model == "" {
		model = DefaultSTTModel
	}

	if color != "" && !colorPattern.MatchString(color) {
		return Settings{}, ErrInvalidColor
	}
	if soundSet == "" {
		soundSet = DefaultNotificationSoundSet
	}
	if !notificationSoundSets[soundSet] {
		return Settings{}, ErrInvalidSoundSet
	}
	if in.TaskArchive != nil {
		if in.TaskArchive.Days < 1 || in.TaskArchive.Days > MaxTaskArchiveAfterDays {
			return Settings{}, ErrInvalidTaskArchiveDays
		}
	}

	current, err := s.Get(ctx)
	if err != nil {
		return Settings{}, err
	}

	archive := current.TaskArchive
	if in.TaskArchive != nil {
		archive = &TaskArchivePolicy{
			Enabled: in.TaskArchive.Enabled,
			Days:    in.TaskArchive.Days,
		}
	}

	apiKey := strings.TrimSpace(in.STTAPIKey)
	switch {
	case in.ClearSTTAPIKey:
		apiKey = ""
	case apiKey == "":
		apiKey = current.STTAPIKey
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
			Key:   keyNotificationSoundSet,
			Value: soundSet,
		},
	); err != nil {
		return Settings{}, err
	}
	for key, value := range map[string]string{
		keyVoiceInputEnabled:  strconv.FormatBool(in.VoiceInputEnabled),
		keySTTBaseURL:         baseURL,
		keySTTModel:           model,
		keySTTAPIKey:          apiKey,
		keyTaskArchiveEnabled: strconv.FormatBool(archive.Enabled),
		keyTaskArchiveDays:    strconv.Itoa(archive.Days),
	} {
		if err := s.store.UpsertSetting(
			ctx,
			store.UpsertSettingParams{Key: key, Value: value},
		); err != nil {
			return Settings{}, err
		}
	}

	return Settings{
		AgencyName:           name,
		PrimaryColor:         color,
		NotificationSoundSet: soundSet,
		TaskArchive:          archive,
		VoiceInputEnabled:    in.VoiceInputEnabled,
		STTBaseURL:           baseURL,
		STTModel:             model,
		STTAPIKey:            apiKey,
		LogoAttachmentID:     current.LogoAttachmentID,
		FaviconAttachmentID:  current.FaviconAttachmentID,
	}, nil
}
