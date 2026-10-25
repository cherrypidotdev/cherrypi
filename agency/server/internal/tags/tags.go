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
	ErrTagMissing = errors.New("tag not found")

	ErrInvalidName = errors.New("tag name must be non-empty")

	ErrTagExists = errors.New("tag name already exists in this project")

	ErrProjectRequired = errors.New("tag project must be non-empty")
)

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

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

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
