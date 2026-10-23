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

package store

import (
	"context"
	"database/sql"
	"time"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Project hierarchy store surface. The closure table `project_ancestors` is
// maintained by the service whenever a parent link changes; these wrappers
// expose the non-recursive reads and the transactional rebuild.

// SQLite --------------------------------------------------------------------

func (s *SqliteStore) GetProjectParent(
	ctx context.Context,
	childID string,
) (Project, error) {
	row, err := s.q.GetProjectParent(ctx, childID)
	return fromSqliteProject(row), err
}

func (s *SqliteStore) ListProjectChildren(
	ctx context.Context,
	parentID string,
) ([]Project, error) {
	rows, err := s.q.ListProjectChildren(ctx, sql.NullString{String: parentID, Valid: true})
	if err != nil {
		return nil, err
	}
	return fromSqliteProjectSlice(rows), nil
}

func (s *SqliteStore) CountProjectChildren(
	ctx context.Context,
	parentID string,
) (int64, error) {
	return s.q.CountProjectChildren(ctx, sql.NullString{String: parentID, Valid: true})
}

func (s *SqliteStore) ListProjectAncestorIDs(
	ctx context.Context,
	projectID string,
) ([]string, error) {
	return s.q.ListProjectAncestorIDs(ctx, projectID)
}

func (s *SqliteStore) ListProjectDescendantIDs(
	ctx context.Context,
	projectID string,
) ([]string, error) {
	return s.q.ListProjectDescendantIDs(ctx, projectID)
}

func (s *SqliteStore) IsProjectDescendant(
	ctx context.Context,
	projectID, ancestorID string,
) (bool, error) {
	return s.q.IsProjectDescendant(ctx, sqlitedb.IsProjectDescendantParams{
		ProjectID:  projectID,
		AncestorID: ancestorID,
	})
}

func (s *SqliteStore) ListProjectSharedOwnerTeams(
	ctx context.Context,
	parentID, childID string,
	actorID ids.ActorID,
) ([]ListProjectSharedOwnerTeamsRow, error) {
	rows, err := s.q.ListProjectSharedOwnerTeams(
		ctx,
		sqlitedb.ListProjectSharedOwnerTeamsParams{
			ParentProjectID: parentID,
			ChildProjectID:  childID,
			ActorID:         int64(actorID),
		},
	)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectSharedOwnerTeamsRowSlice(rows), nil
}

// SetProjectParentTree updates the child's parent and rebuilds the whole
// ancestor closure inside one transaction, so access resolution never observes
// a half-moved tree.
func (s *SqliteStore) SetProjectParentTree(
	ctx context.Context,
	childID string,
	parentID string,
	ancestors []InsertProjectAncestorParams,
) (Project, error) {
	var out Project
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		params := SetProjectParentParams{
			ParentProjectID: sql.NullString{String: parentID, Valid: parentID != ""},
			UpdatedAt:       time.Now().UnixMilli(),
			ID:              childID,
		}
		row, err := q.SetProjectParent(ctx, sqlitedb.SetProjectParentParams(params))
		if err != nil {
			return err
		}
		out = fromSqliteProject(row)
		if err := q.DeleteAllProjectAncestors(ctx); err != nil {
			return err
		}
		for _, ancestor := range ancestors {
			if err := q.InsertProjectAncestor(
				ctx,
				sqlitedb.InsertProjectAncestorParams(ancestor),
			); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func (s *SqliteStore) ListCrossProjectTaskReferences(
	ctx context.Context,
) ([]ListCrossProjectTaskReferencesRow, error) {
	rows, err := s.q.ListCrossProjectTaskReferences(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListCrossProjectTaskReferencesRowSlice(rows), nil
}

func (s *SqliteStore) CreateProjectParentInvitation(
	ctx context.Context,
	p CreateProjectParentInvitationParams,
) (ProjectParentInvitation, error) {
	row, err := s.q.CreateProjectParentInvitation(
		ctx,
		sqlitedb.CreateProjectParentInvitationParams(p),
	)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromSqliteProjectParentInvitation(row), nil
}

func (s *SqliteStore) GetProjectParentInvitation(
	ctx context.Context,
	id int64,
) (ProjectParentInvitation, error) {
	row, err := s.q.GetProjectParentInvitation(ctx, id)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromSqliteProjectParentInvitation(row), nil
}

func (s *SqliteStore) GetPendingProjectParentInvitationForChild(
	ctx context.Context,
	childID string,
) (ProjectParentInvitation, error) {
	row, err := s.q.GetPendingProjectParentInvitationForChild(ctx, childID)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromSqliteProjectParentInvitation(row), nil
}

func (s *SqliteStore) GetPendingProjectParentInvitation(
	ctx context.Context,
	p GetPendingProjectParentInvitationParams,
) (ProjectParentInvitation, error) {
	row, err := s.q.GetPendingProjectParentInvitation(
		ctx,
		sqlitedb.GetPendingProjectParentInvitationParams(p),
	)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromSqliteProjectParentInvitation(row), nil
}

func (s *SqliteStore) ListProjectParentInvitationsByParent(
	ctx context.Context,
	parentID string,
) ([]ListProjectParentInvitationsByParentRow, error) {
	rows, err := s.q.ListProjectParentInvitationsByParent(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectParentInvitationsByParentRowSlice(rows), nil
}

func (s *SqliteStore) ListProjectParentInvitationsByChild(
	ctx context.Context,
	childID string,
) ([]ListProjectParentInvitationsByChildRow, error) {
	rows, err := s.q.ListProjectParentInvitationsByChild(ctx, childID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectParentInvitationsByChildRowSlice(rows), nil
}

func (s *SqliteStore) RespondProjectParentInvitation(
	ctx context.Context,
	p RespondProjectParentInvitationParams,
) (int64, error) {
	return s.q.RespondProjectParentInvitation(
		ctx,
		sqlitedb.RespondProjectParentInvitationParams(p),
	)
}

func (s *SqliteStore) RevokeProjectParentInvitation(
	ctx context.Context,
	p RevokeProjectParentInvitationParams,
) (int64, error) {
	return s.q.RevokeProjectParentInvitation(
		ctx,
		sqlitedb.RevokeProjectParentInvitationParams(p),
	)
}

func (s *SqliteStore) RevokePendingProjectParentInvitationsForChild(
	ctx context.Context,
	p RevokePendingProjectParentInvitationsForChildParams,
) (int64, error) {
	return s.q.RevokePendingProjectParentInvitationsForChild(
		ctx,
		sqlitedb.RevokePendingProjectParentInvitationsForChildParams(p),
	)
}

func (s *SqliteStore) ExpireProjectParentInvitations(
	ctx context.Context,
	now int64,
) (int64, error) {
	return s.q.ExpireProjectParentInvitations(
		ctx,
		sqlitedb.ExpireProjectParentInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now, Valid: true},
			Now:         now,
		},
	)
}

// PostgreSQL ----------------------------------------------------------------

func (s *PgStore) GetProjectParent(
	ctx context.Context,
	childID string,
) (Project, error) {
	row, err := s.q.GetProjectParent(ctx, childID)
	return fromPgProject(row), err
}

func (s *PgStore) ListProjectChildren(
	ctx context.Context,
	parentID string,
) ([]Project, error) {
	rows, err := s.q.ListProjectChildren(ctx, sql.NullString{String: parentID, Valid: true})
	if err != nil {
		return nil, err
	}
	return fromPgProjectSlice(rows), nil
}

func (s *PgStore) CountProjectChildren(
	ctx context.Context,
	parentID string,
) (int64, error) {
	return s.q.CountProjectChildren(ctx, sql.NullString{String: parentID, Valid: true})
}

func (s *PgStore) ListProjectAncestorIDs(
	ctx context.Context,
	projectID string,
) ([]string, error) {
	return s.q.ListProjectAncestorIDs(ctx, projectID)
}

func (s *PgStore) ListProjectDescendantIDs(
	ctx context.Context,
	projectID string,
) ([]string, error) {
	return s.q.ListProjectDescendantIDs(ctx, projectID)
}

func (s *PgStore) IsProjectDescendant(
	ctx context.Context,
	projectID, ancestorID string,
) (bool, error) {
	return s.q.IsProjectDescendant(ctx, pgdb.IsProjectDescendantParams{
		ProjectID:  projectID,
		AncestorID: ancestorID,
	})
}

func (s *PgStore) ListProjectSharedOwnerTeams(
	ctx context.Context,
	parentID, childID string,
	actorID ids.ActorID,
) ([]ListProjectSharedOwnerTeamsRow, error) {
	rows, err := s.q.ListProjectSharedOwnerTeams(
		ctx,
		pgdb.ListProjectSharedOwnerTeamsParams{
			ParentProjectID: parentID,
			ChildProjectID:  childID,
			ActorID:         int64(actorID),
		},
	)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectSharedOwnerTeamsRowSlice(rows), nil
}

func (s *PgStore) SetProjectParentTree(
	ctx context.Context,
	childID string,
	parentID string,
	ancestors []InsertProjectAncestorParams,
) (Project, error) {
	var out Project
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		params := SetProjectParentParams{
			ParentProjectID: sql.NullString{String: parentID, Valid: parentID != ""},
			UpdatedAt:       time.Now().UnixMilli(),
			ID:              childID,
		}
		row, err := q.SetProjectParent(ctx, pgdb.SetProjectParentParams(params))
		if err != nil {
			return err
		}
		out = fromPgProject(row)
		if err := q.DeleteAllProjectAncestors(ctx); err != nil {
			return err
		}
		for _, ancestor := range ancestors {
			if err := q.InsertProjectAncestor(
				ctx,
				pgdb.InsertProjectAncestorParams(ancestor),
			); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func (s *PgStore) ListCrossProjectTaskReferences(
	ctx context.Context,
) ([]ListCrossProjectTaskReferencesRow, error) {
	rows, err := s.q.ListCrossProjectTaskReferences(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListCrossProjectTaskReferencesRowSlice(rows), nil
}

func (s *PgStore) CreateProjectParentInvitation(
	ctx context.Context,
	p CreateProjectParentInvitationParams,
) (ProjectParentInvitation, error) {
	row, err := s.q.CreateProjectParentInvitation(
		ctx,
		pgdb.CreateProjectParentInvitationParams(p),
	)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromPgProjectParentInvitation(row), nil
}

func (s *PgStore) GetProjectParentInvitation(
	ctx context.Context,
	id int64,
) (ProjectParentInvitation, error) {
	row, err := s.q.GetProjectParentInvitation(ctx, id)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromPgProjectParentInvitation(row), nil
}

func (s *PgStore) GetPendingProjectParentInvitationForChild(
	ctx context.Context,
	childID string,
) (ProjectParentInvitation, error) {
	row, err := s.q.GetPendingProjectParentInvitationForChild(ctx, childID)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromPgProjectParentInvitation(row), nil
}

func (s *PgStore) GetPendingProjectParentInvitation(
	ctx context.Context,
	p GetPendingProjectParentInvitationParams,
) (ProjectParentInvitation, error) {
	row, err := s.q.GetPendingProjectParentInvitation(
		ctx,
		pgdb.GetPendingProjectParentInvitationParams(p),
	)
	if err != nil {
		return ProjectParentInvitation{}, err
	}
	return fromPgProjectParentInvitation(row), nil
}

func (s *PgStore) ListProjectParentInvitationsByParent(
	ctx context.Context,
	parentID string,
) ([]ListProjectParentInvitationsByParentRow, error) {
	rows, err := s.q.ListProjectParentInvitationsByParent(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectParentInvitationsByParentRowSlice(rows), nil
}

func (s *PgStore) ListProjectParentInvitationsByChild(
	ctx context.Context,
	childID string,
) ([]ListProjectParentInvitationsByChildRow, error) {
	rows, err := s.q.ListProjectParentInvitationsByChild(ctx, childID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectParentInvitationsByChildRowSlice(rows), nil
}

func (s *PgStore) RespondProjectParentInvitation(
	ctx context.Context,
	p RespondProjectParentInvitationParams,
) (int64, error) {
	return s.q.RespondProjectParentInvitation(
		ctx,
		pgdb.RespondProjectParentInvitationParams(p),
	)
}

func (s *PgStore) RevokeProjectParentInvitation(
	ctx context.Context,
	p RevokeProjectParentInvitationParams,
) (int64, error) {
	return s.q.RevokeProjectParentInvitation(
		ctx,
		pgdb.RevokeProjectParentInvitationParams(p),
	)
}

func (s *PgStore) RevokePendingProjectParentInvitationsForChild(
	ctx context.Context,
	p RevokePendingProjectParentInvitationsForChildParams,
) (int64, error) {
	return s.q.RevokePendingProjectParentInvitationsForChild(
		ctx,
		pgdb.RevokePendingProjectParentInvitationsForChildParams(p),
	)
}

func (s *PgStore) ExpireProjectParentInvitations(
	ctx context.Context,
	now int64,
) (int64, error) {
	return s.q.ExpireProjectParentInvitations(
		ctx,
		pgdb.ExpireProjectParentInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now, Valid: true},
			Now:         now,
		},
	)
}
