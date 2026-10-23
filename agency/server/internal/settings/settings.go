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

// Package settings stores agency-wide configuration (name, primary color) in
// the key-value settings table.
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
	keyAgencyName   = "agency_name"
	keyPrimaryColor = "primary_color"
	// The agency logo/favicon live as settings keys holding an attachment id.
	// They are exported so the attachments service can link a blob to the
	// agency without importing the settings service.
	KeyAgencyLogoAttachmentID    = "agency_logo_attachment_id"
	KeyAgencyFaviconAttachmentID = "agency_favicon_attachment_id"
)

// ErrInvalidColor is returned for a non-`#rrggbb` primary color.
var ErrInvalidColor = errors.New("primary color must be a lowercase #rrggbb")

var colorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// Settings is the agency's configuration.
type Settings struct {
	AgencyName   string
	PrimaryColor string
	// LogoAttachmentID and FaviconAttachmentID reference the `attachments`
	// blobs backing the agency logo and favicon (0 = unset).
	LogoAttachmentID    int64
	FaviconAttachmentID int64
}

// Service reads and writes the settings table.
type Service struct {
	store store.Store
}

// New builds the service.
func New(st store.Store) *Service { return &Service{store: st} }

// Get returns the current settings (zero values when unset).
func (s *Service) Get(ctx context.Context) (Settings, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return Settings{}, err
	}

	var out Settings
	for _, row := range rows {
		switch row.Key {
		case keyAgencyName:
			out.AgencyName = row.Value
		case keyPrimaryColor:
			out.PrimaryColor = row.Value
		case KeyAgencyLogoAttachmentID:
			out.LogoAttachmentID = parseAttachmentID(row.Value)
		case KeyAgencyFaviconAttachmentID:
			out.FaviconAttachmentID = parseAttachmentID(row.Value)
		}
	}
	return out, nil
}

// parseAttachmentID reads a stored attachment id; anything unparsable (or an
// empty value) means unset.
func parseAttachmentID(value string) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

// Update validates and persists the name and primary color, returning what was
// stored. It leaves the logo/favicon references untouched and includes them in
// the result.
func (s *Service) Update(ctx context.Context, in Settings) (Settings, error) {
	name := strings.TrimSpace(in.AgencyName)
	color := strings.ToLower(strings.TrimSpace(in.PrimaryColor))

	if color != "" && !colorPattern.MatchString(color) {
		return Settings{}, ErrInvalidColor
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

	return Settings{
		AgencyName:          name,
		PrimaryColor:        color,
		LogoAttachmentID:    current.LogoAttachmentID,
		FaviconAttachmentID: current.FaviconAttachmentID,
	}, nil
}
