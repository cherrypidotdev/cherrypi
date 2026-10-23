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

// Package projects stores agency-wide session groupings. A project has an
// immutable slug (its route handle), a name, a description, and an archived
// flag. The package owns the database shape only; fanning an archive out to
// the workers lives in the gRPC layer.
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
	// ErrProjectMissing is returned for an unknown project id or slug.
	ErrProjectMissing = errors.New("project not found")
	// ErrInvalidName is returned for an empty project name.
	ErrInvalidName = errors.New("project name must be non-empty")
	// ErrInvalidPrefix is returned for a task prefix with no usable characters.
	ErrInvalidPrefix = errors.New("task prefix must contain letters or digits")
	// ErrProjectHasTasks is returned when deleting a project that still has
	// tasks; every task must be deleted first.
	ErrProjectHasTasks = errors.New("project still has tasks; delete them first")
	// ErrAgentHasTasks is returned when removing an agent from a project while
	// it still has tasks there; they must be deleted first.
	ErrAgentHasTasks = errors.New("agent still has tasks in the project; delete them first")
	// ErrProjectHasChildren is returned when deleting a project that still has
	// child projects; they must be moved or deleted first.
	ErrProjectHasChildren = errors.New(
		"project still has child projects; move or delete them first",
	)
)

// slugSeparators matches any run of characters that are not lowercase letters
// or digits, used to derive a slug.
var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// prefixSeparators matches any run of characters that are not letters or
// digits, used to normalize a task prefix.
var prefixRe = regexp.MustCompile(`[^A-Za-z0-9]+`)

// OrphanPrefix is the task prefix reserved for tasks with no project.
const OrphanPrefix = "TASK"

// Project is the public view of a project.
type Project struct {
	ID          string
	Name        string
	Slug        string
	Prefix      string
	Description string
	// ParentID is the parent project's id, empty for a root project.
	ParentID   string
	ArchivedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TaskRef identifies one task on one agent.
type TaskRef struct {
	ActorID   ids.ActorID
	SessionID string
}

// AgentProject is one membership in an agent's project list: the project and
// the permissions the agent holds in it.
type AgentProject struct {
	Project Project
	Permissions
}

// Permissions are the task permissions an agent holds inside one project.
// They gate the agent's Pi tools; an agent may always list and update tasks
// assigned to itself, regardless of these flags.
type Permissions struct {
	CanListTasks   bool
	CanCreateTasks bool
	CanUpdateTasks bool
}

// ProjectAgent is one member of a project: the agent, how many tasks it
// currently has in the project, and its task permissions.
type ProjectAgent struct {
	AgentID   ids.AgentID
	TaskCount int
	Permissions
}

// Service reads and writes projects.
type Service struct {
	store store.Store
	// TeamOwners resolves a team's owners. Optional; without it no project-team
	// invitation can be approved (admins may still override).
	TeamOwners TeamOwnerChecker
}

// New builds the service.
func New(st store.Store) *Service { return &Service{store: st} }

// List returns every project, oldest first by creation date.
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

// Get resolves a project by id, falling back to slug lookup.
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

// Create registers a project. The slug is derived from `slug` (or the name
// when empty) and de-duplicated; it never changes afterwards. The task prefix
// is derived from `prefix` (or the slug when empty), normalized to uppercase,
// and de-duplicated too.
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
	// The creator becomes the project's first owner, so a new project is
	// manageable without an administrator. Unauthenticated callers (a bootstrap
	// or a test) leave it ownerless.
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

// Update changes a project's name, description and task prefix. The slug is
// immutable. Changing the prefix re-renders the project's task ids so they
// keep the `<prefix>-<seq>` shape; the tasks themselves and every child
// reference stay put.
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

// SetArchived sets or clears a project's archived flag. It does not touch the
// project's sessions; the caller archives them.
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

// Delete removes a project. It returns ErrProjectHasSessions while any session
// (archived or not) still belongs to it.
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
	// Tags belong to the project; they have no meaning once it is gone.
	if err := s.store.DeleteTagsByProject(ctx, p.ID); err != nil {
		return err
	}
	return s.store.DeleteProject(ctx, p.ID)
}

// TaskRefs returns the tasks currently associated with a project.
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

// DeleteAgentTasks drops an agent's task rows in a project, after the tasks
// themselves have been deleted.
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

// uniqueSlug turns a candidate into a unique lower-kebab slug.
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

// uniquePrefix turns a candidate into a unique uppercase task prefix, skipping
// the reserved orphan prefix.
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

// Slugify lowercases and kebab-cases a candidate slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// NormalizePrefix uppercases a task prefix and folds runs of disallowed
// characters into a single dash.
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
