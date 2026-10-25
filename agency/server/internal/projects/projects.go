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

package projects

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

var (
	ErrProjectMissing = errors.New("project not found")

	ErrInvalidName = errors.New("project name must be non-empty")

	ErrInvalidPrefix = errors.New("task prefix must contain letters or digits")

	ErrProjectHasTasks = errors.New("project still has tasks; delete them first")

	ErrAgentHasTasks = errors.New("agent still has tasks in the project; delete them first")

	ErrProjectHasChildren = errors.New(
		"project still has child projects; move or delete them first",
	)
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

var prefixRe = regexp.MustCompile(`[^A-Za-z0-9]+`)

const OrphanPrefix = "TASK"

type Project struct {
	ID          string
	Name        string
	Slug        string
	Prefix      string
	Description string

	ParentID   string
	ArchivedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TaskRef struct {
	ActorID   ids.ActorID
	SessionID string
}

type AgentProject struct {
	Project Project
	Permissions
}

type Permissions struct {
	CanListTasks   bool
	CanCreateTasks bool
	CanUpdateTasks bool
}

type ProjectAgent struct {
	AgentID   ids.AgentID
	TaskCount int
	Permissions
}

type Service struct {
	store store.Store

	TeamOwners TeamOwnerChecker
}

func New(st store.Store) *Service { return &Service{store: st} }

func (s *Service) List(ctx context.Context) ([]Project, error) {
	rows, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = toProject(row)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, idOrSlug string) (Project, error) {
	row, err := s.store.GetProject(ctx, idOrSlug)
	if errors.Is(err, sql.ErrNoRows) {
		row, err = s.store.GetProjectBySlug(ctx, idOrSlug)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrProjectMissing
	}
	if err != nil {
		return Project{}, err
	}
	return toProject(row), nil
}

func (s *Service) Create(
	ctx context.Context,
	name, slug, prefix, description string,
) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, ErrInvalidName
	}
	slug, err := s.uniqueSlug(ctx, slugOrName(slug, name))
	if err != nil {
		return Project{}, err
	}
	prefix, err = s.uniquePrefix(ctx, prefixOrSlug(prefix, slug))
	if err != nil {
		return Project{}, err
	}

	now := time.Now().UnixMilli()
	row, err := s.store.CreateProject(ctx, store.CreateProjectParams{
		ID:          uuid.NewString(),
		Name:        name,
		Slug:        slug,
		Prefix:      prefix,
		Description: strings.TrimSpace(description),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return Project{}, err
	}
	s.writeLog(ctx, row.ID, "created", "", "", "", "Created project "+row.Name, nil)

	if actorID := actingActor(ctx); actorID != 0 {
		_ = s.store.UpsertProjectActorGrant(ctx, store.UpsertProjectActorGrantParams{
			ProjectID:        row.ID,
			ActorID:          int64(actorID),
			Role:             RoleOwner,
			GrantedByActorID: nullActor(actorID),
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}
	return toProject(row), nil
}

func (s *Service) Update(
	ctx context.Context,
	id, name, description, prefix string,
) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, ErrInvalidName
	}
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Project{}, err
	}

	newPrefix := cur.Prefix
	rename := false
	if strings.TrimSpace(prefix) != "" {
		newPrefix = NormalizePrefix(prefix)
		if newPrefix == "" {
			return Project{}, ErrInvalidPrefix
		}
		if newPrefix != cur.Prefix {
			newPrefix, err = s.uniquePrefix(ctx, newPrefix)
			if err != nil {
				return Project{}, err
			}
			rename = true
		}
	}

	row, err := s.store.UpdateProject(ctx, store.UpdateProjectParams{
		Name:        name,
		Description: strings.TrimSpace(description),
		Prefix:      newPrefix,
		UpdatedAt:   time.Now().UnixMilli(),
		ID:          cur.ID,
	}, rename)
	if err != nil {
		return Project{}, err
	}
	if cur.Name != row.Name {
		s.writeLog(ctx, cur.ID, "settings", "name", cur.Name, row.Name,
			"Renamed project to "+row.Name, map[string]string{"old": cur.Name, "new": row.Name})
	}
	if cur.Description != row.Description {
		s.writeLog(ctx, cur.ID, "settings", "description", cur.Description, row.Description,
			"Updated the project description", nil)
	}
	if cur.Prefix != row.Prefix {
		s.writeLog(ctx, cur.ID, "settings", "prefix", cur.Prefix, row.Prefix,
			"Changed the task prefix to "+row.Prefix, nil)
	}
	return toProject(row), nil
}

func (s *Service) SetArchived(ctx context.Context, id string, archived bool) (Project, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return Project{}, err
	}
	at := sql.NullInt64{}
	if archived {
		at = sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}
	}
	row, err := s.store.SetProjectArchived(ctx, store.SetProjectArchivedParams{
		ArchivedAt: at,
		UpdatedAt:  time.Now().UnixMilli(),
		ID:         p.ID,
	})
	if err != nil {
		return Project{}, err
	}
	if archived {
		s.writeLog(ctx, p.ID, "archived", "", "", "", "Archived the project", nil)
	} else {
		s.writeLog(ctx, p.ID, "unarchived", "", "", "", "Unarchived the project", nil)
	}
	return toProject(row), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	p, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	count, err := s.store.CountTasksByProject(
		ctx,
		sql.NullString{String: p.ID, Valid: true},
	)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrProjectHasTasks
	}
	children, err := s.store.CountProjectChildren(ctx, p.ID)
	if err != nil {
		return err
	}
	if children > 0 {
		return ErrProjectHasChildren
	}

	if err := s.store.DeleteTagsByProject(ctx, p.ID); err != nil {
		return err
	}
	return s.store.DeleteProject(ctx, p.ID)
}

func (s *Service) TaskRefs(ctx context.Context, id string) ([]TaskRef, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListTasksByProject(
		ctx,
		sql.NullString{String: p.ID, Valid: true},
	)
	if err != nil {
		return nil, err
	}
	out := make([]TaskRef, 0, len(rows))
	for _, row := range rows {
		if !row.AssigneeActorID.Valid || row.SessionID == "" {
			continue
		}
		out = append(
			out,
			TaskRef{ActorID: ids.ActorID(row.AssigneeActorID.Int64), SessionID: row.SessionID},
		)
	}
	return out, nil
}

func (s *Service) AgentTaskIDs(
	ctx context.Context,
	projectID string, agentID ids.AgentID,
) ([]string, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListTasksByProjectAgent(
		ctx,
		store.ListTasksByProjectAgentParams{
			ProjectID: sql.NullString{String: p.ID, Valid: true},
			AgentID:   int64(agentID),
		},
	)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.SessionID
	}
	return ids, nil
}

func (s *Service) DeleteAgentTasks(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) error {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return err
	}
	return s.store.DeleteTasksByProjectAgent(
		ctx,
		store.DeleteTasksByProjectAgentParams{
			ProjectID: sql.NullString{String: p.ID, Valid: true},
			AgentID:   int64(agentID),
		},
	)
}

func (s *Service) uniqueSlug(ctx context.Context, candidate string) (string, error) {
	base := Slugify(candidate)
	if base == "" {
		base = "project"
	}
	slug := base
	for i := 2; ; i++ {
		_, err := s.store.GetProjectBySlug(ctx, slug)
		if errors.Is(err, sql.ErrNoRows) {
			return slug, nil
		}
		if err != nil {
			return "", err
		}
		slug = base + "-" + strconv.Itoa(i)
	}
}

func (s *Service) uniquePrefix(ctx context.Context, candidate string) (string, error) {
	base := NormalizePrefix(candidate)
	if base == "" {
		return "", ErrInvalidPrefix
	}
	for i := 1; ; i++ {
		prefix := base
		if i > 1 {
			prefix = base + "-" + strconv.Itoa(i)
		}
		if prefix == OrphanPrefix {
			continue
		}
		_, err := s.store.GetProjectByPrefix(ctx, prefix)
		if errors.Is(err, sql.ErrNoRows) {
			return prefix, nil
		}
		if err != nil {
			return "", err
		}
	}
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func NormalizePrefix(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = prefixRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func slugOrName(slug, name string) string {
	if strings.TrimSpace(slug) != "" {
		return slug
	}
	return name
}

func prefixOrSlug(prefix, slug string) string {
	if strings.TrimSpace(prefix) != "" {
		return prefix
	}
	return slug
}

func toProject(row store.Project) Project {
	p := Project{
		ID:          row.ID,
		Name:        row.Name,
		Slug:        row.Slug,
		Prefix:      row.Prefix,
		Description: row.Description,
		CreatedAt:   time.UnixMilli(row.CreatedAt),
		UpdatedAt:   time.UnixMilli(row.UpdatedAt),
	}
	if row.ParentProjectID.Valid {
		p.ParentID = row.ParentProjectID.String
	}
	if row.ArchivedAt.Valid {
		p.ArchivedAt = time.UnixMilli(row.ArchivedAt.Int64)
	}
	return p
}
