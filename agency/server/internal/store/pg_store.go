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
	"fmt"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

type PgStore struct {
	db *sql.DB
	q  *pgdb.Queries
}

func (s *PgStore) Close() error { return s.db.Close() }

func (s *PgStore) tx(ctx context.Context, fn func(*pgdb.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PgStore) deleteActorChildren(ctx context.Context, q *pgdb.Queries, actorID int64) error {
	if err := q.DeleteInvitationsByActor(ctx, actorID); err != nil {
		return err
	}
	if err := q.DeleteTeamMembershipsByActor(ctx, actorID); err != nil {
		return err
	}
	if err := q.DeleteTeamInvitationsByActor(ctx, actorID); err != nil {
		return err
	}
	if err := q.DeleteProjectActorGrantsByActor(ctx, actorID); err != nil {
		return err
	}
	if err := q.DeleteTaskLeadsByActor(ctx, actorID); err != nil {
		return err
	}
	if err := q.DeleteTaskSubscriptionsByActor(ctx, actorID); err != nil {
		return err
	}
	return q.DeleteAgentOwnerEdgesByActor(ctx, actorID)
}

func (s *PgStore) deleteProjectChildren(
	ctx context.Context,
	q *pgdb.Queries,
	projectID string,
) error {
	if err := q.DeleteProjectTeamInvitationsByProject(ctx, projectID); err != nil {
		return err
	}
	if err := q.DeleteProjectTeamMembershipsByProject(ctx, projectID); err != nil {
		return err
	}
	if err := q.DeleteProjectActorGrantsByProject(ctx, projectID); err != nil {
		return err
	}
	if err := q.DeleteProjectParentInvitationsByProject(ctx, projectID); err != nil {
		return err
	}
	if err := q.DeleteProjectAncestorRowsByProject(ctx, projectID); err != nil {
		return err
	}
	return q.DeleteProjectLogs(ctx, projectID)
}

func (s *PgStore) deleteTeamChildren(ctx context.Context, q *pgdb.Queries, teamID int64) error {
	if err := q.DeleteProjectTeamInvitationsByTeam(ctx, teamID); err != nil {
		return err
	}
	if err := q.DeleteTaskTeamsByTeam(ctx, teamID); err != nil {
		return err
	}
	return q.DeleteProjectTeamMembershipsByTeam(ctx, teamID)
}

func (s *PgStore) renameProjectTaskIDs(
	ctx context.Context,
	q *pgdb.Queries,
	projectID, prefix string,
) error {
	return q.RenameProjectTasks(ctx, pgdb.RenameProjectTasksParams(RenameProjectTasksParams{
		Prefix:    sql.NullString{String: prefix, Valid: true},
		ProjectID: sql.NullString{String: projectID, Valid: true},
	}))
}

func (s *PgStore) CreateUserActor(
	ctx context.Context,
	p CreateUserActorParams,
) (ids.ActorID, error) {
	id, err := s.q.CreateUserActor(ctx, pgdb.CreateUserActorParams(p))
	return ids.ActorID(id), err
}

func (s *PgStore) CreateUserAccount(
	ctx context.Context,
	actor CreateUserActorParams,
	user CreateUserRowParams,
) (UserRow, error) {
	var out UserRow
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		actorID, err := q.CreateUserActor(ctx, pgdb.CreateUserActorParams(actor))
		if err != nil {
			return err
		}
		user.ActorID = actorID
		row, err := q.CreateUserRow(ctx, pgdb.CreateUserRowParams(user))
		if err != nil {
			return err
		}
		out = UserRow{
			ID:           ids.UserID(row.ID),
			ActorID:      ids.ActorID(row.ActorID),
			Username:     actor.Username,
			Name:         actor.Name,
			Description:  actor.Description,
			Role:         row.Role,
			Status:       row.Status,
			PasswordHash: actor.PasswordHash,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
		return nil
	})
	return out, err
}

func (s *PgStore) DeleteAnyActor(ctx context.Context, id ids.ActorID) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := s.deleteActorChildren(ctx, q, int64(id)); err != nil {
			return err
		}
		return q.DeleteAnyActor(ctx, int64(id))
	})
}

func (s *PgStore) GetUserByActor(ctx context.Context, actorID ids.ActorID) (UserRow, error) {
	row, err := s.q.GetUserByActor(ctx, int64(actorID))
	if err != nil {
		return UserRow{}, err
	}
	return userRowFrom(
		row.ID, row.ActorID, row.Username, row.Name, row.Description,
		row.PictureAttachmentID, row.Role, row.Status, row.LastLoginAt,
		row.MustChangePassword != 0, row.PasswordHash, row.TotpEnabledAt,
		row.CreatedAt, row.UpdatedAt,
	), nil
}

func (s *PgStore) GetUserByUsername(ctx context.Context, username string) (UserRow, error) {
	row, err := s.q.GetUserByUsername(ctx, username)
	if err != nil {
		return UserRow{}, err
	}
	return userRowFrom(
		row.ID, row.ActorID, row.Username, row.Name, row.Description,
		row.PictureAttachmentID, row.Role, row.Status, row.LastLoginAt,
		row.MustChangePassword != 0, row.PasswordHash, row.TotpEnabledAt,
		row.CreatedAt, row.UpdatedAt,
	), nil
}

func (s *PgStore) ListUsers(ctx context.Context) ([]UserRow, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserRow, len(rows))
	for i, row := range rows {
		out[i] = userRowFrom(
			row.ID, row.ActorID, row.Username, row.Name, row.Description,
			row.PictureAttachmentID, row.Role, row.Status, row.LastLoginAt,
			row.MustChangePassword != 0, row.PasswordHash, row.TotpEnabledAt,
			row.CreatedAt, row.UpdatedAt,
		)
	}
	return out, nil
}

func (s *PgStore) GetSessionByHash(ctx context.Context, tokenHash []byte) (SessionRow, error) {
	row, err := s.q.GetSessionByHash(ctx, tokenHash)
	if err != nil {
		return SessionRow{}, err
	}
	return SessionRow{
		ID:                  row.ID,
		ActorID:             ids.ActorID(row.ActorID),
		TokenHash:           row.TokenHash,
		Method:              row.Method,
		UserAgent:           row.UserAgent,
		IP:                  row.Ip,
		Remember:            row.Remember != 0,
		TOTPVerified:        row.TotpVerified != 0,
		CreatedAt:           row.CreatedAt,
		LastSeenAt:          row.LastSeenAt,
		ExpiresAt:           row.ExpiresAt,
		Kind:                row.Kind,
		Username:            row.Username,
		Name:                row.Name,
		Description:         row.Description,
		PictureAttachmentID: row.PictureAttachmentID,
		PasswordHash:        row.PasswordHash,
		TOTPSecret:          row.TotpSecret,
		TOTPEnabledAt:       row.TotpEnabledAt,
		Role:                row.Role,
		Status:              row.Status,
		MustChangePassword:  row.MustChangePassword,
	}, nil
}

func (s *PgStore) GetPersonalAccessTokenByHash(
	ctx context.Context,
	tokenHash []byte,
) (PATRow, error) {
	row, err := s.q.GetPersonalAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		return PATRow{}, err
	}
	return PATRow{
		ID:            row.ID,
		ActorID:       ids.ActorID(row.ActorID),
		Name:          row.Name,
		TokenHash:     row.TokenHash,
		Prefix:        row.Prefix,
		CreatedAt:     row.CreatedAt,
		LastUsedAt:    row.LastUsedAt,
		ExpiresAt:     row.ExpiresAt,
		Kind:          row.Kind,
		Username:      row.Username,
		ActorName:     row.ActorName,
		TOTPEnabledAt: row.TotpEnabledAt,
		Role:          row.Role,
		Status:        row.Status,
	}, nil
}

func (s *PgStore) PruneDomainLogs(ctx context.Context, beforeAt int64) error {
	if err := s.q.PruneTaskLogs(ctx, beforeAt); err != nil {
		return err
	}
	if err := s.q.PruneProjectLogs(ctx, beforeAt); err != nil {
		return err
	}
	if err := s.q.PruneTeamLogs(ctx, beforeAt); err != nil {
		return err
	}
	if err := s.q.PruneAgentDomainLogs(ctx, beforeAt); err != nil {
		return err
	}
	return s.q.PruneUserLogs(ctx, beforeAt)
}

func (s *PgStore) DeleteIdentityProvider(ctx context.Context, id int64) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.DeleteExternalCredentialsByProvider(ctx, id); err != nil {
			return err
		}
		return q.DeleteIdentityProvider(ctx, id)
	})
}

func (s *PgStore) DeleteLDAPProvider(ctx context.Context, id int64) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.DeleteLDAPCredentialsByProvider(ctx, id); err != nil {
			return err
		}
		return q.DeleteLDAPProvider(ctx, id)
	})
}

func (s *PgStore) CreateInvitation(
	ctx context.Context,
	p CreateInvitationParams,
) (InvitationRow, error) {
	row, err := s.q.CreateInvitation(ctx, pgdb.CreateInvitationParams(p))
	if err != nil {
		return InvitationRow{}, err
	}
	return invitationRowFrom(
		row.ID, row.Kind, row.Username, row.ActorID, row.Email, row.ProviderID,
		row.ExternalUsername, row.Role, row.InvitedByActorID, row.CreatedAt,
		row.ExpiresAt, row.AcceptedAt, row.RevokedAt, "", sql.NullString{},
		sql.NullString{},
	), nil
}

func (s *PgStore) GetInvitationByID(ctx context.Context, id int64) (InvitationRow, error) {
	row, err := s.q.GetInvitationByID(ctx, id)
	if err != nil {
		return InvitationRow{}, err
	}
	return invitationRowFrom(
		row.ID, row.Kind, row.Username, row.ActorID, row.Email, row.ProviderID,
		row.ExternalUsername, row.Role, row.InvitedByActorID, row.CreatedAt,
		row.ExpiresAt, row.AcceptedAt, row.RevokedAt, "", sql.NullString{},
		sql.NullString{},
	), nil
}

func (s *PgStore) GetInvitationByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (InvitationRow, error) {
	row, err := s.q.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return InvitationRow{}, err
	}
	return invitationRowFrom(
		row.ID, row.Kind, row.Username, row.ActorID, row.Email, row.ProviderID,
		row.ExternalUsername, row.Role, row.InvitedByActorID, row.CreatedAt,
		row.ExpiresAt, row.AcceptedAt, row.RevokedAt, "", sql.NullString{},
		sql.NullString{},
	), nil
}

func (s *PgStore) ListInvitations(ctx context.Context) ([]InvitationRow, error) {
	rows, err := s.q.ListInvitations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InvitationRow, len(rows))
	for i, row := range rows {
		out[i] = invitationRowFrom(
			row.ID, row.Kind, row.Username, row.ActorID, row.Email,
			row.ProviderID, row.ExternalUsername, row.Role,
			row.InvitedByActorID, row.CreatedAt, row.ExpiresAt, row.AcceptedAt,
			row.RevokedAt, row.ActorName, row.ActorStatus,
			row.InvitedByUsername,
		)
	}
	return out, nil
}

func (s *PgStore) GetPendingInvitationByUsername(
	ctx context.Context,
	username string,
) (InvitationRow, error) {
	row, err := s.q.GetPendingInvitationByUsername(ctx, username)
	if err != nil {
		return InvitationRow{}, err
	}
	return invitationRowFrom(
		row.ID, row.Kind, row.Username, row.ActorID, row.Email, row.ProviderID,
		row.ExternalUsername, row.Role, row.InvitedByActorID, row.CreatedAt,
		row.ExpiresAt, row.AcceptedAt, row.RevokedAt, "", sql.NullString{},
		sql.NullString{},
	), nil
}

func (s *PgStore) GetPendingInvitationByActor(
	ctx context.Context,
	actorID ids.ActorID,
) (InvitationRow, error) {
	row, err := s.q.GetPendingInvitationByActor(ctx, int64(actorID))
	if err != nil {
		return InvitationRow{}, err
	}
	return invitationRowFrom(
		row.ID, row.Kind, row.Username, row.ActorID, row.Email, row.ProviderID,
		row.ExternalUsername, row.Role, row.InvitedByActorID, row.CreatedAt,
		row.ExpiresAt, row.AcceptedAt, row.RevokedAt, "", sql.NullString{},
		sql.NullString{},
	), nil
}

func (s *PgStore) GetPendingExternalInvitation(
	ctx context.Context,
	p GetPendingExternalInvitationParams,
) (InvitationRow, error) {
	row, err := s.q.GetPendingExternalInvitation(ctx, pgdb.GetPendingExternalInvitationParams(p))
	if err != nil {
		return InvitationRow{}, err
	}
	return invitationRowFrom(
		row.ID, row.Kind, row.Username, row.ActorID, row.Email, row.ProviderID,
		row.ExternalUsername, row.Role, row.InvitedByActorID, row.CreatedAt,
		row.ExpiresAt, row.AcceptedAt, row.RevokedAt, "", sql.NullString{},
		sql.NullString{},
	), nil
}

func (s *PgStore) GetEffectiveProjectRole(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
) (int64, error) {
	value, err := s.q.GetEffectiveProjectRole(
		ctx,
		pgdb.GetEffectiveProjectRoleParams(GetEffectiveProjectRoleParams{
			ProjectID: projectID,
			ActorID:   int64(actorID),
		}),
	)
	if err != nil {
		return 0, err
	}
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case []byte:
		var out int64
		for _, b := range v {
			if b < '0' || b > '9' {
				continue
			}
			out = out*10 + int64(b-'0')
		}
		return out, nil
	default:
		return 0, nil
	}
}

func (s *PgStore) GetAgent(ctx context.Context, id ids.AgentID) (AgentRow, error) {
	row, err := s.q.GetAgent(ctx, int64(id))
	if err != nil {
		return AgentRow{}, err
	}
	return toAgentRow(row.ID, row.ActorID, row.Username, row.Name, row.Description, row.Role,
		row.PictureAttachmentID, row.CreatedAt, row.UpdatedAt, row.LastSeenAt), nil
}

func (s *PgStore) GetAgentByActor(ctx context.Context, actorID ids.ActorID) (AgentRow, error) {
	row, err := s.q.GetAgentByActor(ctx, int64(actorID))
	if err != nil {
		return AgentRow{}, err
	}
	return toAgentRow(row.ID, row.ActorID, row.Username, row.Name, row.Description, row.Role,
		row.PictureAttachmentID, row.CreatedAt, row.UpdatedAt, row.LastSeenAt), nil
}

func (s *PgStore) GetAgentByUsername(ctx context.Context, username string) (AgentRow, error) {
	row, err := s.q.GetAgentByUsername(ctx, username)
	if err != nil {
		return AgentRow{}, err
	}
	return toAgentRow(row.ID, row.ActorID, row.Username, row.Name, row.Description, row.Role,
		row.PictureAttachmentID, row.CreatedAt, row.UpdatedAt, row.LastSeenAt), nil
}

func (s *PgStore) ListAgents(ctx context.Context) ([]AgentRow, error) {
	rows, err := s.q.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgentRow, len(rows))
	for i, row := range rows {
		out[i] = toAgentRow(row.ID, row.ActorID, row.Username, row.Name, row.Description, row.Role,
			row.PictureAttachmentID, row.CreatedAt, row.UpdatedAt, row.LastSeenAt)
	}
	return out, nil
}

func (s *PgStore) UpdateAgent(
	ctx context.Context,
	id ids.AgentID,
	name, description string,
	updatedAt int64,
) (AgentRow, error) {
	if err := s.q.UpdateAgent(ctx, pgdb.UpdateAgentParams(UpdateAgentParams{
		Name:        name,
		Description: description,
		UpdatedAt:   updatedAt,
		ID:          int64(id),
	})); err != nil {
		return AgentRow{}, err
	}
	return s.GetAgent(ctx, id)
}

func (s *PgStore) SetAgentRole(ctx context.Context, id ids.AgentID, role string) error {
	return s.q.UpdateAgentRole(ctx, pgdb.UpdateAgentRoleParams{
		Role: role,
		ID:   int64(id),
	})
}

func (s *PgStore) DeleteAgent(ctx context.Context, id ids.AgentID) error {
	row, err := s.q.GetAgent(ctx, int64(id))
	if err != nil {
		return err
	}
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.DeleteAgent(ctx, int64(id)); err != nil {
			return err
		}
		if err := s.deleteActorChildren(ctx, q, row.ActorID); err != nil {
			return err
		}
		return q.DeleteActor(ctx, row.ActorID)
	})
}

func (s *PgStore) UpdateProject(
	ctx context.Context,
	p UpdateProjectParams,
	renameTasks bool,
) (Project, error) {
	var row Project
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		updated, err := q.UpdateProject(ctx, pgdb.UpdateProjectParams(p))
		if err != nil {
			return err
		}
		row = fromPgProject(updated)
		if !renameTasks {
			return nil
		}
		return s.renameProjectTaskIDs(ctx, q, p.ID, p.Prefix)
	})
	return row, err
}

func (s *PgStore) DeleteProject(ctx context.Context, id string) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := s.deleteProjectChildren(ctx, q, id); err != nil {
			return err
		}
		return q.DeleteProject(ctx, id)
	})
}

func (s *PgStore) DeleteTasksByProjectAgent(
	ctx context.Context,
	p DeleteTasksByProjectAgentParams,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {

		if err := q.DeleteProjectAgentReferenceEdges(
			ctx,
			pgdb.DeleteProjectAgentReferenceEdgesParams(DeleteProjectAgentReferenceEdgesParams(p)),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskTagsByProjectAgent(
			ctx,
			pgdb.DeleteTaskTagsByProjectAgentParams(DeleteTaskTagsByProjectAgentParams(p)),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskLinksByProjectAgent(
			ctx,
			pgdb.DeleteTaskLinksByProjectAgentParams(DeleteTaskLinksByProjectAgentParams(p)),
		); err != nil {
			return err
		}
		if err := q.ClearTaskNotificationsByProjectAgent(
			ctx,
			pgdb.ClearTaskNotificationsByProjectAgentParams(
				ClearTaskNotificationsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteMessageAttachmentsByProjectAgent(
			ctx,
			pgdb.DeleteMessageAttachmentsByProjectAgentParams(
				DeleteMessageAttachmentsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskTeamsByProjectAgent(
			ctx,
			pgdb.DeleteTaskTeamsByProjectAgentParams(DeleteTaskTeamsByProjectAgentParams(p)),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskTeamPromptsByProjectAgent(
			ctx,
			pgdb.DeleteTaskTeamPromptsByProjectAgentParams(
				DeleteTaskTeamPromptsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskLeadsByProjectAgent(
			ctx,
			pgdb.DeleteTaskLeadsByProjectAgentParams(DeleteTaskLeadsByProjectAgentParams(p)),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskSubscriptionsByProjectAgent(
			ctx,
			pgdb.DeleteTaskSubscriptionsByProjectAgentParams(
				DeleteTaskSubscriptionsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskMessagesByProjectAgent(
			ctx,
			pgdb.DeleteTaskMessagesByProjectAgentParams(DeleteTaskMessagesByProjectAgentParams(p)),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskAttachmentsByProjectAgent(
			ctx,
			pgdb.DeleteTaskAttachmentsByProjectAgentParams(
				DeleteTaskAttachmentsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteArtifactPartsByProjectAgent(
			ctx,
			pgdb.DeleteArtifactPartsByProjectAgentParams(
				DeleteArtifactPartsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskArtifactsByProjectAgent(
			ctx,
			pgdb.DeleteTaskArtifactsByProjectAgentParams(
				DeleteTaskArtifactsByProjectAgentParams(p),
			),
		); err != nil {
			return err
		}
		if err := q.DeleteTaskLogsByProjectAgent(
			ctx,
			pgdb.DeleteTaskLogsByProjectAgentParams(DeleteTaskLogsByProjectAgentParams(p)),
		); err != nil {
			return err
		}
		if err := q.DeleteTasksByProjectAgent(
			ctx,
			pgdb.DeleteTasksByProjectAgentParams(p),
		); err != nil {
			return err
		}
		return nil
	})
}

func (s *PgStore) AppendTaskSessionEntries(
	ctx context.Context,
	entries []AppendTaskSessionEntryParams,
) (int64, error) {
	var appended int64
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		for _, p := range entries {
			n, err := q.AppendTaskSessionEntry(ctx, pgdb.AppendTaskSessionEntryParams(p))
			if err != nil {
				return err
			}
			appended += n
		}
		return nil
	})
	return appended, err
}

func (s *PgStore) DeleteTask(ctx context.Context, id string) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {

		if err := q.DeleteTaskReferenceEdges(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskTags(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskLinks(ctx, id); err != nil {
			return err
		}
		if err := q.ClearTaskNotifications(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteMessageAttachmentsByTask(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskTeams(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskTeamPrompts(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskLeads(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskSubscriptions(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskMessages(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskAttachments(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteArtifactPartsByTask(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskArtifacts(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTaskLogs(ctx, id); err != nil {
			return err
		}
		return q.DeleteTask(ctx, id)
	})
}

func (s *PgStore) ListTags(
	ctx context.Context,
	projectID string,
	includeArchived bool,
) ([]ListTagsRow, error) {
	var flag int64
	if includeArchived {
		flag = 1
	}
	rows, err := s.q.ListTags(ctx, pgdb.ListTagsParams(ListTagsParams{
		ProjectID:       sql.NullString{String: projectID, Valid: projectID != ""},
		IncludeArchived: flag,
	}))
	if err != nil {
		return nil, err
	}
	return fromPgListTagsRowSlice(rows), nil
}

func (s *PgStore) DeleteAttachment(ctx context.Context, id int64) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		attachmentID := sql.NullInt64{Int64: id, Valid: true}
		if err := q.ClearProjectPictureByAttachment(ctx, attachmentID); err != nil {
			return err
		}
		if err := q.ClearAgentPictureByAttachment(ctx, attachmentID); err != nil {
			return err
		}
		if err := q.ClearUserPictureByAttachment(ctx, attachmentID); err != nil {
			return err
		}
		if err := q.DeleteAttachmentTaskLinks(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteAttachmentMessageLinks(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteArtifactPartsByAttachment(ctx, attachmentID); err != nil {
			return err
		}
		if err := q.DeleteEmptyTaskArtifacts(ctx); err != nil {
			return err
		}
		return q.DeleteAttachment(ctx, id)
	})
}

func (s *PgStore) InsertAgentLog(
	ctx context.Context,
	p InsertAgentLogParams,
	keep int64,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.InsertAgentLog(ctx, pgdb.InsertAgentLogParams(p)); err != nil {
			return err
		}
		return q.PruneAgentLogs(ctx, pgdb.PruneAgentLogsParams(PruneAgentLogsParams{
			AgentID: p.AgentID,
			Keep:    keep,
		}))
	})
}

func (s *PgStore) CreateTeamWithOwner(
	ctx context.Context,
	p CreateTeamParams,
	ownerActorID ids.ActorID,
) (Team, error) {
	var out Team
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		team, err := q.CreateTeam(ctx, pgdb.CreateTeamParams(p))
		if err != nil {
			return err
		}
		if err := q.AddTeamMember(ctx, pgdb.AddTeamMemberParams(AddTeamMemberParams{
			TeamID:   team.ID,
			ActorID:  int64(ownerActorID),
			Role:     "owner",
			JoinedAt: p.CreatedAt,
		})); err != nil {
			return err
		}
		out = fromPgTeam(team)
		return nil
	})
	return out, err
}

func (s *PgStore) DeleteTeam(ctx context.Context, id int64) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := s.deleteTeamChildren(ctx, q, id); err != nil {
			return err
		}
		if err := q.DeleteTeamInvitationsByTeam(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTeamMembers(ctx, id); err != nil {
			return err
		}
		if err := q.DeleteTeamLogs(ctx, id); err != nil {
			return err
		}
		return q.DeleteTeam(ctx, id)
	})
}

func (s *PgStore) IsAgentOwner(
	ctx context.Context,
	agentActorID, ownerActorID ids.ActorID,
) (bool, error) {
	count, err := s.q.IsAgentOwner(ctx, pgdb.IsAgentOwnerParams(IsAgentOwnerParams{
		AgentActorID: int64(agentActorID),
		OwnerActorID: int64(ownerActorID),
	}))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PgStore) RegisterAgent(
	ctx context.Context,
	p RegisterAgentParams,
) (AgentRow, error) {
	var out AgentRow
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		actorID, err := q.CreateActor(ctx, pgdb.CreateActorParams(CreateActorParams{
			Username:    p.Username,
			Name:        p.DisplayName,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}))
		if err != nil {
			return err
		}
		row, err := q.CreateAgentRow(ctx, actorID)
		if err != nil {
			return err
		}
		if p.OwnerActorID != 0 {
			if err := q.AddAgentOwner(ctx, pgdb.AddAgentOwnerParams(AddAgentOwnerParams{
				AgentActorID: actorID,
				OwnerActorID: int64(p.OwnerActorID),
				CreatedAt:    p.CreatedAt,
			})); err != nil {
				return err
			}
		}
		out = toAgentRow(row.ID, row.ActorID, p.Username, p.DisplayName, p.Description, row.Role,
			sql.NullInt64{}, p.CreatedAt, p.UpdatedAt, sql.NullInt64{})
		return nil
	})
	return out, err
}

func (s *PgStore) TransitionTask(
	ctx context.Context,
	p TransitionTaskParams,
) (int64, error) {
	switch p.Event {
	case TaskPromote:
		return s.q.PromoteTask(ctx, pgdb.PromoteTaskParams(PromoteTaskParams{
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskDemote:
		return s.q.DemoteTask(ctx, pgdb.DemoteTaskParams(DemoteTaskParams{
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskQueue:
		return s.q.QueueTask(ctx, pgdb.QueueTaskParams(QueueTaskParams{
			QueuedAt:  p.QueuedAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskDequeue:
		return s.q.DequeueTask(ctx, pgdb.DequeueTaskParams(DequeueTaskParams{
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskArchive:
		return s.q.ArchiveTask(ctx, pgdb.ArchiveTaskParams(ArchiveTaskParams{
			ArchivedAt: p.ArchivedAt,
			UpdatedAt:  p.UpdatedAt,
			ID:         p.ID,
		}))
	case TaskUnarchive:
		return s.q.UnarchiveTask(ctx, pgdb.UnarchiveTaskParams(UnarchiveTaskParams{
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskSettle:
		return s.q.SettleTask(ctx, pgdb.SettleTaskParams(SettleTaskParams{
			SettledAt: p.SettledAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskInput:
		return s.q.MarkTaskInput(ctx, pgdb.MarkTaskInputParams(MarkTaskInputParams{
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskClearInput:
		return s.q.ClearTaskInput(ctx, pgdb.ClearTaskInputParams(ClearTaskInputParams{
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskResume:
		return s.q.ResumeTask(ctx, pgdb.ResumeTaskParams(ResumeTaskParams{
			SettledAt: p.SettledAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskComplete:
		return s.q.CompleteTask(ctx, pgdb.CompleteTaskParams(CompleteTaskParams{
			SettledAt: p.SettledAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskCancel:
		return s.q.CancelTask(ctx, pgdb.CancelTaskParams(CancelTaskParams{
			SettledAt: p.SettledAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskReopen:
		return s.q.ReopenTask(ctx, pgdb.ReopenTaskParams(ReopenTaskParams{
			State:     p.State,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskClaim:
		return s.q.ClaimTask(ctx, pgdb.ClaimTaskParams(ClaimTaskParams{
			AssigneeActorID:      p.AssigneeActorID,
			AssigneePinned:       p.AssigneePinned,
			StartedAt:            p.StartedAt,
			UpdatedAt:            p.UpdatedAt,
			OrgPromptID:          p.OrgPromptID,
			AgentPromptID:        p.AgentPromptID,
			ProjectPromptID:      p.ProjectPromptID,
			AgentProjectPromptID: p.AgentProjectPromptID,
			ID:                   p.ID,
		}))
	case TaskFail:
		return s.q.FailTask(ctx, pgdb.FailTaskParams(FailTaskParams{
			Error:     p.Error,
			SettledAt: p.SettledAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskFailBySession:
		return s.q.FailTaskBySession(ctx, pgdb.FailTaskBySessionParams(FailTaskBySessionParams{
			Error:     p.Error,
			SettledAt: p.SettledAt,
			UpdatedAt: p.UpdatedAt,
			AgentID:   int64(p.AgentID),
			SessionID: p.SessionID,
		}))
	case TaskRequeue:
		return s.q.RequeueTask(ctx, pgdb.RequeueTaskParams(RequeueTaskParams{
			Error:     p.Error,
			QueuedAt:  p.QueuedAt,
			UpdatedAt: p.UpdatedAt,
			ID:        p.ID,
		}))
	case TaskRequeueWithPrompt:
		return s.q.RequeueTaskWithPrompt(
			ctx,
			pgdb.RequeueTaskWithPromptParams(RequeueTaskWithPromptParams{
				PendingPrompt:        p.PendingPrompt,
				PendingPromptActorID: p.PendingPromptActorID,
				QueuedAt:             p.QueuedAt,
				UpdatedAt:            p.UpdatedAt,
				ID:                   p.ID,
			}),
		)
	default:
		return 0, fmt.Errorf("unknown task event %q", p.Event)
	}
}
