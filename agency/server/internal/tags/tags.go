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

// Package tags stores the per-project task labels. A tag belongs to exactly
// one project and its name is unique within that project; it is soft-deleted
// (archived) so completed tasks keep rendering it. The package owns the
// database shape only.
package tags

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

var (
	// ErrTagMissing is returned for an unknown tag id.
	ErrTagMissing = errors.New("tag not found")
	// ErrInvalidName is returned for an empty tag name.
	ErrInvalidName = errors.New("tag name must be non-empty")
	// ErrTagExists is returned when the name is already taken in the project
	// (case-insensitive).
	ErrTagExists = errors.New("tag name already exists in this project")
	// ErrProjectRequired is returned when a tag is created without a project.
	ErrProjectRequired = errors.New("tag project must be non-empty")
)

// Tag is the public view of a task label.
type Tag struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Color       string
	Archived    bool
	IsDefault   bool
	TaskCount   int
}

// Service reads and writes tags.
type Service struct {
	store store.Store
}

// New builds the service.
func New(st store.Store) *Service { return &Service{store: st} }

// List returns tags with their task counts, sorted by name. When projectID is
// set only that project's tags are returned; otherwise every project's tags
// are. Archived tags are included only when includeArchived is set.
func (s *Service) List(
	ctx context.Context,
	projectID string,
	includeArchived bool,
) ([]Tag, error) {
	rows, err := s.store.ListTags(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, row := range rows {
		out[i] = Tag{
			ID:          row.ID,
			ProjectID:   row.ProjectID.String,
			Name:        row.Name,
			Description: row.Description,
			Color:       row.Color,
			Archived:    row.ArchivedAt.Valid,
			IsDefault:   row.IsDefault != 0,
			TaskCount:   int(row.TaskCount),
		}
	}
	return out, nil
}

// Get resolves a tag by id, with its task count.
func (s *Service) Get(ctx context.Context, id string) (Tag, error) {
	row, err := s.store.GetTagWithCount(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Tag{}, ErrTagMissing
	}
	if err != nil {
		return Tag{}, err
	}
	return Tag{
		ID:          row.ID,
		ProjectID:   row.ProjectID.String,
		Name:        row.Name,
		Description: row.Description,
		Color:       row.Color,
		Archived:    row.ArchivedAt.Valid,
		IsDefault:   row.IsDefault != 0,
		TaskCount:   int(row.TaskCount),
	}, nil
}

// Create registers a tag in a project. The name must be unused in that project.
func (s *Service) Create(
	ctx context.Context,
	projectID, name, description, color string,
	isDefault bool,
) (Tag, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return Tag{}, ErrProjectRequired
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Tag{}, ErrInvalidName
	}
	if err := s.ensureNameFree(ctx, projectID, name, ""); err != nil {
		return Tag{}, err
	}

	now := time.Now().UnixMilli()
	row, err := s.store.CreateTag(ctx, store.CreateTagParams{
		ID:          uuid.NewString(),
		ProjectID:   sql.NullString{String: projectID, Valid: true},
		Name:        name,
		Description: strings.TrimSpace(description),
		Color:       strings.TrimSpace(color),
		IsDefault:   boolInt(isDefault),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return Tag{}, err
	}
	return s.Get(ctx, row.ID)
}

// Update changes a tag's name, description and color. A tag never moves
// between projects.
func (s *Service) Update(
	ctx context.Context,
	id, name, description, color string,
	isDefault bool,
) (Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Tag{}, ErrInvalidName
	}
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Tag{}, err
	}
	if err := s.ensureNameFree(ctx, cur.ProjectID, name, id); err != nil {
		return Tag{}, err
	}
	if _, err := s.store.UpdateTag(ctx, store.UpdateTagParams{
		Name:        name,
		Description: strings.TrimSpace(description),
		Color:       strings.TrimSpace(color),
		IsDefault:   boolInt(isDefault),
		UpdatedAt:   time.Now().UnixMilli(),
		ID:          id,
	}); err != nil {
		return Tag{}, err
	}
	return s.Get(ctx, id)
}

// SetArchived soft-deletes or restores a tag.
func (s *Service) SetArchived(ctx context.Context, id string, archived bool) (Tag, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return Tag{}, err
	}
	at := sql.NullInt64{}
	if archived {
		at = sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}
	}
	if _, err := s.store.SetTagArchived(ctx, store.SetTagArchivedParams{
		ArchivedAt: at,
		UpdatedAt:  time.Now().UnixMilli(),
		ID:         id,
	}); err != nil {
		return Tag{}, err
	}
	return s.Get(ctx, id)
}

// Exists reports whether a tag id resolves.
func (s *Service) Exists(ctx context.Context, id string) (bool, error) {
	_, err := s.store.GetTag(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ensureNameFree rejects a name already taken by another tag in the project.
func (s *Service) ensureNameFree(
	ctx context.Context,
	projectID, name, selfID string,
) error {
	row, err := s.store.GetTagByName(ctx, projectID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.ID != selfID {
		return ErrTagExists
	}
	return nil
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
