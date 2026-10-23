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

// Package artifacts stores a task's outputs in the A2A (Agent2Agent) shape: an
// artifact has a stable id unique within its task, a name/description, and one
// or more parts (text, structured data, or a file). File parts are backed by
// the attachments blob registry. Agents publish them by emitting a
// `task_artifact` event; the agency upserts or appends parts.
package artifacts

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

// Part kinds, mirroring A2A's text/raw/url/data oneof.
const (
	KindText = "text"
	KindFile = "file"
	KindData = "data"
)

// Part is one artifact part.
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

// Artifact is a task output.
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

// Update is one incoming artifact from the agent.
type Update struct {
	// ID is the artifact id; empty ids are generated.
	ID           string
	Name         string
	Description  string
	MetadataJSON string
	// Append adds the parts to the existing artifact instead of replacing them.
	Append bool
	// LastChunk marks a streamed artifact complete (A2A's last_chunk).
	LastChunk bool
	Parts     []Part
}

// Service stores task artifacts. It is safe for concurrent use.
type Service struct {
	store store.Store
}

// New builds the service.
func New(st store.Store) *Service {
	return &Service{store: st}
}

// Apply upserts the artifacts an agent published. When replace is true the
// update names the task's complete artifact set and unlisted artifacts are
// removed; otherwise listed artifacts are upserted/appended and others kept.
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
			// Keep the existing name/description unless the update sets them.
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

		// Create or replace: drop the old parts and write the new set.
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

// appendParts writes parts after the artifact's current last part.
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

// List returns a task's artifacts, oldest first, each with its parts.
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
