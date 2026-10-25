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

package artifacts

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

const (
	KindText = "text"
	KindFile = "file"
	KindData = "data"
)

type Part struct {
	Kind         string
	Text         string
	DataJSON     string
	URL          string
	AttachmentID int64
	Filename     string
	MediaType    string
	MetadataJSON string
}

type Artifact struct {
	ID           string
	Name         string
	Description  string
	MetadataJSON string
	Streaming    bool
	Parts        []Part
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Update struct {
	ID           string
	Name         string
	Description  string
	MetadataJSON string

	Append bool

	LastChunk bool
	Parts     []Part
}

type Service struct {
	store store.Store
}

func New(st store.Store) *Service {
	return &Service{store: st}
}

func (s *Service) Apply(
	ctx context.Context,
	taskID string,
	replace bool,
	updates []Update,
) error {
	existing, err := s.store.ListTaskArtifacts(ctx, taskID)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(existing))
	for _, row := range existing {
		known[row.ArtifactID] = true
	}

	if replace {
		keep := make(map[string]bool, len(updates))
		for _, update := range updates {
			if update.ID != "" {
				keep[update.ID] = true
			}
		}
		for _, row := range existing {
			if !keep[row.ArtifactID] {
				if err := s.delete(ctx, taskID, row.ArtifactID); err != nil {
					return err
				}
			}
		}
	}

	position, err := s.store.MaxTaskArtifactPosition(ctx, taskID)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	for _, update := range updates {
		id := update.ID
		if id == "" {
			id = uuid.NewString()
		}

		streaming := int64(0)
		if !update.LastChunk {
			streaming = 1
		}

		if update.Append && known[id] {

			current, err := s.store.GetTaskArtifact(ctx, taskID, id)
			if err != nil {
				return err
			}
			name := current.Name
			if update.Name != "" {
				name = update.Name
			}
			description := current.Description
			if update.Description != "" {
				description = update.Description
			}
			metadata := current.MetadataJson
			if update.MetadataJSON != "" {
				metadata = update.MetadataJSON
			}
			if err := s.store.UpsertTaskArtifact(ctx, store.UpsertTaskArtifactParams{
				TaskID:       taskID,
				ArtifactID:   id,
				Name:         name,
				Description:  description,
				MetadataJson: metadata,
				Streaming:    streaming,
				Position:     current.Position,
				CreatedAt:    current.CreatedAt,
				UpdatedAt:    now,
			}); err != nil {
				return err
			}
			if err := s.appendParts(ctx, taskID, id, update.Parts); err != nil {
				return err
			}
			continue
		}

		position++
		if err := s.store.UpsertTaskArtifact(ctx, store.UpsertTaskArtifactParams{
			TaskID:       taskID,
			ArtifactID:   id,
			Name:         update.Name,
			Description:  update.Description,
			MetadataJson: update.MetadataJSON,
			Streaming:    streaming,
			Position:     position,
			CreatedAt:    now,
			UpdatedAt:    now,
		}); err != nil {
			return err
		}
		if err := s.store.DeleteArtifactParts(ctx, taskID, id); err != nil {
			return err
		}
		if err := s.appendParts(ctx, taskID, id, update.Parts); err != nil {
			return err
		}
		known[id] = true
	}

	return nil
}

func (s *Service) appendParts(ctx context.Context, taskID, artifactID string, parts []Part) error {
	if len(parts) == 0 {
		return nil
	}
	position, err := s.store.MaxArtifactPartPosition(ctx, taskID, artifactID)
	if err != nil {
		return err
	}

	for _, part := range parts {
		position++
		if err := s.store.InsertArtifactPart(ctx, store.InsertArtifactPartParams{
			TaskID:       taskID,
			ArtifactID:   artifactID,
			Position:     position,
			Kind:         normalizeKind(part.Kind),
			Text:         part.Text,
			DataJson:     part.DataJSON,
			Url:          part.URL,
			AttachmentID: nullInt64(part.AttachmentID),
			Filename:     part.Filename,
			MediaType:    part.MediaType,
			MetadataJson: part.MetadataJSON,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) delete(ctx context.Context, taskID, artifactID string) error {
	if err := s.store.DeleteArtifactParts(ctx, taskID, artifactID); err != nil {
		return err
	}
	return s.store.DeleteTaskArtifact(ctx, taskID, artifactID)
}

func (s *Service) List(ctx context.Context, taskID string) ([]Artifact, error) {
	rows, err := s.store.ListTaskArtifacts(ctx, taskID)
	if err != nil {
		return nil, err
	}
	parts, err := s.store.ListArtifactParts(ctx, taskID)
	if err != nil {
		return nil, err
	}

	byArtifact := map[string][]Part{}
	for _, row := range parts {
		byArtifact[row.ArtifactID] = append(byArtifact[row.ArtifactID], Part{
			Kind:         row.Kind,
			Text:         row.Text,
			DataJSON:     row.DataJson,
			URL:          row.Url,
			AttachmentID: row.AttachmentID.Int64,
			Filename:     row.Filename,
			MediaType:    row.MediaType,
			MetadataJSON: row.MetadataJson,
		})
	}

	out := make([]Artifact, 0, len(rows))
	for _, row := range rows {
		out = append(out, Artifact{
			ID:           row.ArtifactID,
			Name:         row.Name,
			Description:  row.Description,
			MetadataJSON: row.MetadataJson,
			Streaming:    row.Streaming != 0,
			Parts:        byArtifact[row.ArtifactID],
			CreatedAt:    time.UnixMilli(row.CreatedAt),
			UpdatedAt:    time.UnixMilli(row.UpdatedAt),
		})
	}
	return out, nil
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case KindFile:
		return KindFile
	case KindData:
		return KindData
	default:
		return KindText
	}
}

func nullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v > 0}
}
