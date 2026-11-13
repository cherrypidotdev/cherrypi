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

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
)

func (s *SqliteStore) UpsertProjectGroupMembership(
	ctx context.Context,
	p UpsertProjectGroupMembershipParams,
) error {
	return s.q.UpsertProjectGroupMembership(ctx, sqlitedb.UpsertProjectGroupMembershipParams(p))
}

func (s *SqliteStore) GetProjectGroupMembership(
	ctx context.Context,
	p GetProjectGroupMembershipParams,
) (ProjectGroupMembership, error) {
	row, err := s.q.GetProjectGroupMembership(ctx, sqlitedb.GetProjectGroupMembershipParams(p))
	if err != nil {
		return ProjectGroupMembership{}, err
	}
	return fromSqliteProjectGroupMembership(row), nil
}

func (s *SqliteStore) DeleteProjectGroupMembership(
	ctx context.Context,
	p DeleteProjectGroupMembershipParams,
) (int64, error) {
	return s.q.DeleteProjectGroupMembership(ctx, sqlitedb.DeleteProjectGroupMembershipParams(p))
}

func (s *SqliteStore) DeleteProjectGroupMembershipsByGroup(
	ctx context.Context,
	groupID int64,
) error {
	return s.q.DeleteProjectGroupMembershipsByGroup(ctx, groupID)
}

func (s *SqliteStore) DeleteProjectGroupMembershipsByProject(
	ctx context.Context,
	projectID string,
) error {
	return s.q.DeleteProjectGroupMembershipsByProject(ctx, projectID)
}

func (s *SqliteStore) GetEffectiveProjectGroupRole(
	ctx context.Context,
	p GetEffectiveProjectGroupRoleParams,
) (int64, error) {
	return s.q.GetEffectiveProjectGroupRole(ctx, sqlitedb.GetEffectiveProjectGroupRoleParams(p))
}

func (s *SqliteStore) ListProjectGroupLinks(
	ctx context.Context,
	projectID string,
) ([]ListProjectGroupLinksRow, error) {
	rows, err := s.q.ListProjectGroupLinks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectGroupLinksRowSlice(rows), nil
}

func (s *SqliteStore) ListInheritedProjectGroupLinks(
	ctx context.Context,
	projectID string,
) ([]ListInheritedProjectGroupLinksRow, error) {
	rows, err := s.q.ListInheritedProjectGroupLinks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListInheritedProjectGroupLinksRowSlice(rows), nil
}

func (s *SqliteStore) ListProjectGroupInvitations(
	ctx context.Context,
	projectID string,
) ([]ListProjectGroupInvitationsRow, error) {
	rows, err := s.q.ListProjectGroupInvitations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectGroupInvitationsRowSlice(rows), nil
}

func (s *SqliteStore) GetProjectGroupInvitation(
	ctx context.Context,
	id int64,
) (ProjectGroupInvitation, error) {
	row, err := s.q.GetProjectGroupInvitation(ctx, id)
	if err != nil {
		return ProjectGroupInvitation{}, err
	}
	return fromSqliteProjectGroupInvitation(row), nil
}

func (s *SqliteStore) GetPendingProjectGroupInvitation(
	ctx context.Context,
	p GetPendingProjectGroupInvitationParams,
) (ProjectGroupInvitation, error) {
	row, err := s.q.GetPendingProjectGroupInvitation(
		ctx,
		sqlitedb.GetPendingProjectGroupInvitationParams(p),
	)
	if err != nil {
		return ProjectGroupInvitation{}, err
	}
	return fromSqliteProjectGroupInvitation(row), nil
}

func (s *SqliteStore) CreateProjectGroupInvitation(
	ctx context.Context,
	p CreateProjectGroupInvitationParams,
) (ProjectGroupInvitation, error) {
	row, err := s.q.CreateProjectGroupInvitation(
		ctx,
		sqlitedb.CreateProjectGroupInvitationParams(p),
	)
	if err != nil {
		return ProjectGroupInvitation{}, err
	}
	return fromSqliteProjectGroupInvitation(row), nil
}

func (s *SqliteStore) RespondProjectGroupInvitation(
	ctx context.Context,
	p RespondProjectGroupInvitationParams,
) (int64, error) {
	return s.q.RespondProjectGroupInvitation(ctx, sqlitedb.RespondProjectGroupInvitationParams(p))
}

func (s *SqliteStore) RevokeProjectGroupInvitation(
	ctx context.Context,
	p RevokeProjectGroupInvitationParams,
) (int64, error) {
	return s.q.RevokeProjectGroupInvitation(ctx, sqlitedb.RevokeProjectGroupInvitationParams(p))
}

func (s *SqliteStore) ExpireProjectGroupInvitations(
	ctx context.Context,
	p ExpireProjectGroupInvitationsParams,
) (int64, error) {
	return s.q.ExpireProjectGroupInvitations(ctx, sqlitedb.ExpireProjectGroupInvitationsParams(p))
}

func (s *SqliteStore) DeleteProjectGroupInvitationsByGroup(
	ctx context.Context,
	groupID int64,
) error {
	return s.q.DeleteProjectGroupInvitationsByGroup(ctx, groupID)
}

func (s *SqliteStore) DeleteProjectGroupInvitationsByProject(
	ctx context.Context,
	projectID string,
) error {
	return s.q.DeleteProjectGroupInvitationsByProject(ctx, projectID)
}

func (s *SqliteStore) ListAgentGroupProjectAccess(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupProjectAccessRow, error) {
	rows, err := s.q.ListAgentGroupProjectAccess(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentGroupProjectAccessRowSlice(rows), nil
}

func (s *SqliteStore) ListAgentGroupProjectInvitations(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupProjectInvitationsRow, error) {
	rows, err := s.q.ListAgentGroupProjectInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentGroupProjectInvitationsRowSlice(rows), nil
}

func (s *SqliteStore) UpsertTeamGroupMembership(
	ctx context.Context,
	p UpsertTeamGroupMembershipParams,
) error {
	return s.q.UpsertTeamGroupMembership(ctx, sqlitedb.UpsertTeamGroupMembershipParams(p))
}

func (s *SqliteStore) GetTeamGroupMembership(
	ctx context.Context,
	p GetTeamGroupMembershipParams,
) (TeamGroupMembership, error) {
	row, err := s.q.GetTeamGroupMembership(ctx, sqlitedb.GetTeamGroupMembershipParams(p))
	if err != nil {
		return TeamGroupMembership{}, err
	}
	return fromSqliteTeamGroupMembership(row), nil
}

func (s *SqliteStore) DeleteTeamGroupMembership(
	ctx context.Context,
	p DeleteTeamGroupMembershipParams,
) (int64, error) {
	return s.q.DeleteTeamGroupMembership(ctx, sqlitedb.DeleteTeamGroupMembershipParams(p))
}

func (s *SqliteStore) DeleteTeamGroupMembershipsByGroup(ctx context.Context, groupID int64) error {
	return s.q.DeleteTeamGroupMembershipsByGroup(ctx, groupID)
}

func (s *SqliteStore) DeleteTeamGroupMembershipsByTeam(ctx context.Context, teamID int64) error {
	return s.q.DeleteTeamGroupMembershipsByTeam(ctx, teamID)
}

func (s *SqliteStore) ListTeamGroupLinks(
	ctx context.Context,
	teamID int64,
) ([]ListTeamGroupLinksRow, error) {
	rows, err := s.q.ListTeamGroupLinks(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamGroupLinksRowSlice(rows), nil
}

func (s *SqliteStore) ListTeamGroupInvitations(
	ctx context.Context,
	teamID int64,
) ([]ListTeamGroupInvitationsRow, error) {
	rows, err := s.q.ListTeamGroupInvitations(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamGroupInvitationsRowSlice(rows), nil
}

func (s *SqliteStore) GetTeamGroupInvitation(
	ctx context.Context,
	id int64,
) (TeamGroupInvitation, error) {
	row, err := s.q.GetTeamGroupInvitation(ctx, id)
	if err != nil {
		return TeamGroupInvitation{}, err
	}
	return fromSqliteTeamGroupInvitation(row), nil
}

func (s *SqliteStore) GetPendingTeamGroupInvitation(
	ctx context.Context,
	p GetPendingTeamGroupInvitationParams,
) (TeamGroupInvitation, error) {
	row, err := s.q.GetPendingTeamGroupInvitation(
		ctx,
		sqlitedb.GetPendingTeamGroupInvitationParams(p),
	)
	if err != nil {
		return TeamGroupInvitation{}, err
	}
	return fromSqliteTeamGroupInvitation(row), nil
}

func (s *SqliteStore) CreateTeamGroupInvitation(
	ctx context.Context,
	p CreateTeamGroupInvitationParams,
) (TeamGroupInvitation, error) {
	row, err := s.q.CreateTeamGroupInvitation(ctx, sqlitedb.CreateTeamGroupInvitationParams(p))
	if err != nil {
		return TeamGroupInvitation{}, err
	}
	return fromSqliteTeamGroupInvitation(row), nil
}

func (s *SqliteStore) RespondTeamGroupInvitation(
	ctx context.Context,
	p RespondTeamGroupInvitationParams,
) (int64, error) {
	return s.q.RespondTeamGroupInvitation(ctx, sqlitedb.RespondTeamGroupInvitationParams(p))
}

func (s *SqliteStore) RevokeTeamGroupInvitation(
	ctx context.Context,
	p RevokeTeamGroupInvitationParams,
) (int64, error) {
	return s.q.RevokeTeamGroupInvitation(ctx, sqlitedb.RevokeTeamGroupInvitationParams(p))
}

func (s *SqliteStore) ExpireTeamGroupInvitations(
	ctx context.Context,
	p ExpireTeamGroupInvitationsParams,
) (int64, error) {
	return s.q.ExpireTeamGroupInvitations(ctx, sqlitedb.ExpireTeamGroupInvitationsParams(p))
}

func (s *SqliteStore) DeleteTeamGroupInvitationsByGroup(ctx context.Context, groupID int64) error {
	return s.q.DeleteTeamGroupInvitationsByGroup(ctx, groupID)
}

func (s *SqliteStore) DeleteTeamGroupInvitationsByTeam(ctx context.Context, teamID int64) error {
	return s.q.DeleteTeamGroupInvitationsByTeam(ctx, teamID)
}

func (s *SqliteStore) ListAgentGroupTeamAccess(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupTeamAccessRow, error) {
	rows, err := s.q.ListAgentGroupTeamAccess(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentGroupTeamAccessRowSlice(rows), nil
}

func (s *SqliteStore) ListAgentGroupTeamInvitations(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupTeamInvitationsRow, error) {
	rows, err := s.q.ListAgentGroupTeamInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentGroupTeamInvitationsRowSlice(rows), nil
}

func (s *SqliteStore) ListProjectOwnerActorIDs(
	ctx context.Context,
	projectID string,
) ([]int64, error) {
	return s.q.ListProjectOwnerActorIDs(ctx, projectID)
}

func (s *SqliteStore) ListTeamOwnerActorIDs(ctx context.Context, teamID int64) ([]int64, error) {
	return s.q.ListTeamOwnerActorIDs(ctx, teamID)
}

func (s *PgStore) UpsertProjectGroupMembership(
	ctx context.Context,
	p UpsertProjectGroupMembershipParams,
) error {
	return s.q.UpsertProjectGroupMembership(ctx, pgdb.UpsertProjectGroupMembershipParams(p))
}

func (s *PgStore) GetProjectGroupMembership(
	ctx context.Context,
	p GetProjectGroupMembershipParams,
) (ProjectGroupMembership, error) {
	row, err := s.q.GetProjectGroupMembership(ctx, pgdb.GetProjectGroupMembershipParams(p))
	if err != nil {
		return ProjectGroupMembership{}, err
	}
	return fromPgProjectGroupMembership(row), nil
}

func (s *PgStore) DeleteProjectGroupMembership(
	ctx context.Context,
	p DeleteProjectGroupMembershipParams,
) (int64, error) {
	return s.q.DeleteProjectGroupMembership(ctx, pgdb.DeleteProjectGroupMembershipParams(p))
}

func (s *PgStore) DeleteProjectGroupMembershipsByGroup(ctx context.Context, groupID int64) error {
	return s.q.DeleteProjectGroupMembershipsByGroup(ctx, groupID)
}

func (s *PgStore) DeleteProjectGroupMembershipsByProject(
	ctx context.Context,
	projectID string,
) error {
	return s.q.DeleteProjectGroupMembershipsByProject(ctx, projectID)
}

func (s *PgStore) GetEffectiveProjectGroupRole(
	ctx context.Context,
	p GetEffectiveProjectGroupRoleParams,
) (int64, error) {
	return s.q.GetEffectiveProjectGroupRole(ctx, pgdb.GetEffectiveProjectGroupRoleParams(p))
}

func (s *PgStore) ListProjectGroupLinks(
	ctx context.Context,
	projectID string,
) ([]ListProjectGroupLinksRow, error) {
	rows, err := s.q.ListProjectGroupLinks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectGroupLinksRowSlice(rows), nil
}

func (s *PgStore) ListInheritedProjectGroupLinks(
	ctx context.Context,
	projectID string,
) ([]ListInheritedProjectGroupLinksRow, error) {
	rows, err := s.q.ListInheritedProjectGroupLinks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgListInheritedProjectGroupLinksRowSlice(rows), nil
}

func (s *PgStore) ListProjectGroupInvitations(
	ctx context.Context,
	projectID string,
) ([]ListProjectGroupInvitationsRow, error) {
	rows, err := s.q.ListProjectGroupInvitations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectGroupInvitationsRowSlice(rows), nil
}

func (s *PgStore) GetProjectGroupInvitation(
	ctx context.Context,
	id int64,
) (ProjectGroupInvitation, error) {
	row, err := s.q.GetProjectGroupInvitation(ctx, id)
	if err != nil {
		return ProjectGroupInvitation{}, err
	}
	return fromPgProjectGroupInvitation(row), nil
}

func (s *PgStore) GetPendingProjectGroupInvitation(
	ctx context.Context,
	p GetPendingProjectGroupInvitationParams,
) (ProjectGroupInvitation, error) {
	row, err := s.q.GetPendingProjectGroupInvitation(
		ctx,
		pgdb.GetPendingProjectGroupInvitationParams(p),
	)
	if err != nil {
		return ProjectGroupInvitation{}, err
	}
	return fromPgProjectGroupInvitation(row), nil
}

func (s *PgStore) CreateProjectGroupInvitation(
	ctx context.Context,
	p CreateProjectGroupInvitationParams,
) (ProjectGroupInvitation, error) {
	row, err := s.q.CreateProjectGroupInvitation(ctx, pgdb.CreateProjectGroupInvitationParams(p))
	if err != nil {
		return ProjectGroupInvitation{}, err
	}
	return fromPgProjectGroupInvitation(row), nil
}

func (s *PgStore) RespondProjectGroupInvitation(
	ctx context.Context,
	p RespondProjectGroupInvitationParams,
) (int64, error) {
	return s.q.RespondProjectGroupInvitation(ctx, pgdb.RespondProjectGroupInvitationParams(p))
}

func (s *PgStore) RevokeProjectGroupInvitation(
	ctx context.Context,
	p RevokeProjectGroupInvitationParams,
) (int64, error) {
	return s.q.RevokeProjectGroupInvitation(ctx, pgdb.RevokeProjectGroupInvitationParams(p))
}

func (s *PgStore) ExpireProjectGroupInvitations(
	ctx context.Context,
	p ExpireProjectGroupInvitationsParams,
) (int64, error) {
	return s.q.ExpireProjectGroupInvitations(ctx, pgdb.ExpireProjectGroupInvitationsParams(p))
}

func (s *PgStore) DeleteProjectGroupInvitationsByGroup(ctx context.Context, groupID int64) error {
	return s.q.DeleteProjectGroupInvitationsByGroup(ctx, groupID)
}

func (s *PgStore) DeleteProjectGroupInvitationsByProject(
	ctx context.Context,
	projectID string,
) error {
	return s.q.DeleteProjectGroupInvitationsByProject(ctx, projectID)
}

func (s *PgStore) ListAgentGroupProjectAccess(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupProjectAccessRow, error) {
	rows, err := s.q.ListAgentGroupProjectAccess(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromPgListAgentGroupProjectAccessRowSlice(rows), nil
}

func (s *PgStore) ListAgentGroupProjectInvitations(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupProjectInvitationsRow, error) {
	rows, err := s.q.ListAgentGroupProjectInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromPgListAgentGroupProjectInvitationsRowSlice(rows), nil
}

func (s *PgStore) UpsertTeamGroupMembership(
	ctx context.Context,
	p UpsertTeamGroupMembershipParams,
) error {
	return s.q.UpsertTeamGroupMembership(ctx, pgdb.UpsertTeamGroupMembershipParams(p))
}

func (s *PgStore) GetTeamGroupMembership(
	ctx context.Context,
	p GetTeamGroupMembershipParams,
) (TeamGroupMembership, error) {
	row, err := s.q.GetTeamGroupMembership(ctx, pgdb.GetTeamGroupMembershipParams(p))
	if err != nil {
		return TeamGroupMembership{}, err
	}
	return fromPgTeamGroupMembership(row), nil
}

func (s *PgStore) DeleteTeamGroupMembership(
	ctx context.Context,
	p DeleteTeamGroupMembershipParams,
) (int64, error) {
	return s.q.DeleteTeamGroupMembership(ctx, pgdb.DeleteTeamGroupMembershipParams(p))
}

func (s *PgStore) DeleteTeamGroupMembershipsByGroup(ctx context.Context, groupID int64) error {
	return s.q.DeleteTeamGroupMembershipsByGroup(ctx, groupID)
}

func (s *PgStore) DeleteTeamGroupMembershipsByTeam(ctx context.Context, teamID int64) error {
	return s.q.DeleteTeamGroupMembershipsByTeam(ctx, teamID)
}

func (s *PgStore) ListTeamGroupLinks(
	ctx context.Context,
	teamID int64,
) ([]ListTeamGroupLinksRow, error) {
	rows, err := s.q.ListTeamGroupLinks(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromPgListTeamGroupLinksRowSlice(rows), nil
}

func (s *PgStore) ListTeamGroupInvitations(
	ctx context.Context,
	teamID int64,
) ([]ListTeamGroupInvitationsRow, error) {
	rows, err := s.q.ListTeamGroupInvitations(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromPgListTeamGroupInvitationsRowSlice(rows), nil
}

func (s *PgStore) GetTeamGroupInvitation(
	ctx context.Context,
	id int64,
) (TeamGroupInvitation, error) {
	row, err := s.q.GetTeamGroupInvitation(ctx, id)
	if err != nil {
		return TeamGroupInvitation{}, err
	}
	return fromPgTeamGroupInvitation(row), nil
}

func (s *PgStore) GetPendingTeamGroupInvitation(
	ctx context.Context,
	p GetPendingTeamGroupInvitationParams,
) (TeamGroupInvitation, error) {
	row, err := s.q.GetPendingTeamGroupInvitation(ctx, pgdb.GetPendingTeamGroupInvitationParams(p))
	if err != nil {
		return TeamGroupInvitation{}, err
	}
	return fromPgTeamGroupInvitation(row), nil
}

func (s *PgStore) CreateTeamGroupInvitation(
	ctx context.Context,
	p CreateTeamGroupInvitationParams,
) (TeamGroupInvitation, error) {
	row, err := s.q.CreateTeamGroupInvitation(ctx, pgdb.CreateTeamGroupInvitationParams(p))
	if err != nil {
		return TeamGroupInvitation{}, err
	}
	return fromPgTeamGroupInvitation(row), nil
}

func (s *PgStore) RespondTeamGroupInvitation(
	ctx context.Context,
	p RespondTeamGroupInvitationParams,
) (int64, error) {
	return s.q.RespondTeamGroupInvitation(ctx, pgdb.RespondTeamGroupInvitationParams(p))
}

func (s *PgStore) RevokeTeamGroupInvitation(
	ctx context.Context,
	p RevokeTeamGroupInvitationParams,
) (int64, error) {
	return s.q.RevokeTeamGroupInvitation(ctx, pgdb.RevokeTeamGroupInvitationParams(p))
}

func (s *PgStore) ExpireTeamGroupInvitations(
	ctx context.Context,
	p ExpireTeamGroupInvitationsParams,
) (int64, error) {
	return s.q.ExpireTeamGroupInvitations(ctx, pgdb.ExpireTeamGroupInvitationsParams(p))
}

func (s *PgStore) DeleteTeamGroupInvitationsByGroup(ctx context.Context, groupID int64) error {
	return s.q.DeleteTeamGroupInvitationsByGroup(ctx, groupID)
}

func (s *PgStore) DeleteTeamGroupInvitationsByTeam(ctx context.Context, teamID int64) error {
	return s.q.DeleteTeamGroupInvitationsByTeam(ctx, teamID)
}

func (s *PgStore) ListAgentGroupTeamAccess(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupTeamAccessRow, error) {
	rows, err := s.q.ListAgentGroupTeamAccess(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromPgListAgentGroupTeamAccessRowSlice(rows), nil
}

func (s *PgStore) ListAgentGroupTeamInvitations(
	ctx context.Context,
	groupID int64,
) ([]ListAgentGroupTeamInvitationsRow, error) {
	rows, err := s.q.ListAgentGroupTeamInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return fromPgListAgentGroupTeamInvitationsRowSlice(rows), nil
}

func (s *PgStore) ListProjectOwnerActorIDs(ctx context.Context, projectID string) ([]int64, error) {
	return s.q.ListProjectOwnerActorIDs(ctx, projectID)
}

func (s *PgStore) ListTeamOwnerActorIDs(ctx context.Context, teamID int64) ([]int64, error) {
	return s.q.ListTeamOwnerActorIDs(ctx, teamID)
}

func (s *SqliteStore) GetEffectiveTeamMember(
	ctx context.Context,
	p GetTeamMemberParams,
) (EffectiveTeamMember, error) {
	row, err := s.q.GetEffectiveTeamMember(ctx, sqlitedb.GetEffectiveTeamMemberParams(p))
	if err != nil {
		return EffectiveTeamMember{}, err
	}
	return fromSqliteEffectiveTeamMember(row), nil
}

func (s *PgStore) GetEffectiveTeamMember(
	ctx context.Context,
	p GetTeamMemberParams,
) (EffectiveTeamMember, error) {
	row, err := s.q.GetEffectiveTeamMember(ctx, pgdb.GetEffectiveTeamMemberParams(p))
	if err != nil {
		return EffectiveTeamMember{}, err
	}
	return fromPgEffectiveTeamMember(row), nil
}

func (s *SqliteStore) ListEffectiveTeamMemberActorIDs(
	ctx context.Context,
	teamID int64,
) ([]int64, error) {
	return s.q.ListEffectiveTeamMemberActorIDs(ctx, teamID)
}

func (s *PgStore) ListEffectiveTeamMemberActorIDs(
	ctx context.Context,
	teamID int64,
) ([]int64, error) {
	return s.q.ListEffectiveTeamMemberActorIDs(ctx, teamID)
}
