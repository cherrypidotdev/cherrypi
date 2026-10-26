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

	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

func (s *SqliteStore) GetActorByID(ctx context.Context, id ids.ActorID) (Actor, error) {
	row, err := s.q.GetActorByID(ctx, int64(id))
	if err != nil {
		return Actor{}, err
	}
	return fromSqliteActor(row), nil
}

func (s *SqliteStore) GetActorByUsername(
	ctx context.Context,
	username string,
) (Actor, error) {
	row, err := s.q.GetActorByUsername(ctx, username)
	if err != nil {
		return Actor{}, err
	}
	return fromSqliteActor(row), nil
}

func (s *SqliteStore) GetActorByEmail(ctx context.Context, email string) (Actor, error) {
	row, err := s.q.GetActorByEmail(ctx, email)
	if err != nil {
		return Actor{}, err
	}
	return fromSqliteActor(row), nil
}

func (s *SqliteStore) UpdateActorPassword(
	ctx context.Context,
	p UpdateActorPasswordParams,
) error {
	return s.q.UpdateActorPassword(ctx, sqlitedb.UpdateActorPasswordParams(p))
}

func (s *SqliteStore) UpdateActorTOTP(ctx context.Context, p UpdateActorTOTPParams) error {
	return s.q.UpdateActorTOTP(ctx, sqlitedb.UpdateActorTOTPParams(p))
}

func (s *SqliteStore) ClearActorTOTP(ctx context.Context, p ClearActorTOTPParams) error {
	return s.q.ClearActorTOTP(ctx, sqlitedb.ClearActorTOTPParams(p))
}

func (s *SqliteStore) UpdateActorProfile(
	ctx context.Context,
	p UpdateActorProfileParams,
) error {
	return s.q.UpdateActorProfile(ctx, sqlitedb.UpdateActorProfileParams(p))
}

func (s *SqliteStore) CountUsers(ctx context.Context) (int64, error) {
	return s.q.CountUsers(ctx)
}

func (s *SqliteStore) CountAdmins(ctx context.Context) (int64, error) {
	return s.q.CountAdmins(ctx)
}

func (s *SqliteStore) UpdateUserRole(ctx context.Context, p UpdateUserRoleParams) error {
	return s.q.UpdateUserRole(ctx, sqlitedb.UpdateUserRoleParams(p))
}

func (s *SqliteStore) UpdateUserStatus(ctx context.Context, p UpdateUserStatusParams) error {
	return s.q.UpdateUserStatus(ctx, sqlitedb.UpdateUserStatusParams(p))
}

func (s *SqliteStore) TouchUserLogin(ctx context.Context, p TouchUserLoginParams) error {
	return s.q.TouchUserLogin(ctx, sqlitedb.TouchUserLoginParams(p))
}

func (s *SqliteStore) SetUserMustChangePassword(
	ctx context.Context,
	p SetUserMustChangePasswordParams,
) error {
	return s.q.SetUserMustChangePassword(ctx, sqlitedb.SetUserMustChangePasswordParams(p))
}

func (s *SqliteStore) DeleteUserByActor(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteUserByActor(ctx, int64(actorID))
}

func (s *SqliteStore) CreateEmailCredential(
	ctx context.Context,
	p CreateEmailCredentialParams,
) (EmailCredential, error) {
	row, err := s.q.CreateEmailCredential(ctx, sqlitedb.CreateEmailCredentialParams(p))
	if err != nil {
		return EmailCredential{}, err
	}
	return fromSqliteEmailCredential(row), nil
}

func (s *SqliteStore) ListEmailCredentials(
	ctx context.Context,
	actorID ids.ActorID,
) ([]EmailCredential, error) {
	rows, err := s.q.ListEmailCredentials(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteEmailCredentialSlice(rows), nil
}

func (s *SqliteStore) GetEmailCredentialByEmail(
	ctx context.Context,
	email string,
) (EmailCredential, error) {
	row, err := s.q.GetEmailCredentialByEmail(ctx, email)
	if err != nil {
		return EmailCredential{}, err
	}
	return fromSqliteEmailCredential(row), nil
}

func (s *SqliteStore) GetEmailCredentialByID(
	ctx context.Context,
	id int64,
) (EmailCredential, error) {
	row, err := s.q.GetEmailCredentialByID(ctx, id)
	if err != nil {
		return EmailCredential{}, err
	}
	return fromSqliteEmailCredential(row), nil
}

func (s *SqliteStore) ConfirmEmailCredential(
	ctx context.Context,
	p ConfirmEmailCredentialParams,
) error {
	return s.q.ConfirmEmailCredential(ctx, sqlitedb.ConfirmEmailCredentialParams(p))
}

func (s *SqliteStore) DeleteEmailCredential(
	ctx context.Context,
	p DeleteEmailCredentialParams,
) error {
	return s.q.DeleteEmailCredential(ctx, sqlitedb.DeleteEmailCredentialParams(p))
}

func (s *SqliteStore) CountConfirmedEmails(
	ctx context.Context,
	actorID ids.ActorID,
) (int64, error) {
	return s.q.CountConfirmedEmails(ctx, int64(actorID))
}

func (s *SqliteStore) CreateSession(
	ctx context.Context,
	p CreateSessionParams,
) (Session, error) {
	row, err := s.q.CreateSession(ctx, sqlitedb.CreateSessionParams(p))
	if err != nil {
		return Session{}, err
	}
	return fromSqliteSession(row), nil
}

func (s *SqliteStore) TouchSession(ctx context.Context, p TouchSessionParams) error {
	return s.q.TouchSession(ctx, sqlitedb.TouchSessionParams(p))
}

func (s *SqliteStore) DeleteSessionByHash(ctx context.Context, tokenHash []byte) error {
	return s.q.DeleteSessionByHash(ctx, tokenHash)
}

func (s *SqliteStore) DeleteSessionByID(
	ctx context.Context,
	p DeleteSessionByIDParams,
) error {
	return s.q.DeleteSessionByID(ctx, sqlitedb.DeleteSessionByIDParams(p))
}

func (s *SqliteStore) DeleteSessionsByActor(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteSessionsByActor(ctx, int64(actorID))
}

func (s *SqliteStore) DeleteOtherSessions(
	ctx context.Context,
	p DeleteOtherSessionsParams,
) error {
	return s.q.DeleteOtherSessions(ctx, sqlitedb.DeleteOtherSessionsParams(p))
}

func (s *SqliteStore) ListSessionsByActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Session, error) {
	rows, err := s.q.ListSessionsByActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteSessionSlice(rows), nil
}

func (s *SqliteStore) DeleteExpiredSessions(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredSessions(ctx, now)
}

func (s *SqliteStore) CreateToken(
	ctx context.Context,
	p CreateTokenParams,
) (Token, error) {
	row, err := s.q.CreateToken(ctx, sqlitedb.CreateTokenParams(p))
	if err != nil {
		return Token{}, err
	}
	return fromSqliteToken(row), nil
}

func (s *SqliteStore) GetTokenByHash(ctx context.Context, tokenHash []byte) (Token, error) {
	row, err := s.q.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		return Token{}, err
	}
	return fromSqliteToken(row), nil
}

func (s *SqliteStore) ConsumeToken(ctx context.Context, p ConsumeTokenParams) (int64, error) {
	return s.q.ConsumeToken(ctx, sqlitedb.ConsumeTokenParams(p))
}

func (s *SqliteStore) DeleteTokensByActorKind(
	ctx context.Context,
	p DeleteTokensByActorKindParams,
) error {
	return s.q.DeleteTokensByActorKind(ctx, sqlitedb.DeleteTokensByActorKindParams(p))
}

func (s *SqliteStore) DeleteExpiredTokens(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredTokens(ctx, now)
}

func (s *SqliteStore) CreatePersonalAccessToken(
	ctx context.Context,
	p CreatePersonalAccessTokenParams,
) (PersonalAccessToken, error) {
	row, err := s.q.CreatePersonalAccessToken(ctx, sqlitedb.CreatePersonalAccessTokenParams(p))
	if err != nil {
		return PersonalAccessToken{}, err
	}
	return fromSqlitePersonalAccessToken(row), nil
}

func (s *SqliteStore) ListPersonalAccessTokens(
	ctx context.Context,
	actorID ids.ActorID,
) ([]PersonalAccessToken, error) {
	rows, err := s.q.ListPersonalAccessTokens(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqlitePersonalAccessTokenSlice(rows), nil
}

func (s *SqliteStore) TouchPersonalAccessToken(
	ctx context.Context,
	p TouchPersonalAccessTokenParams,
) error {
	return s.q.TouchPersonalAccessToken(ctx, sqlitedb.TouchPersonalAccessTokenParams(p))
}

func (s *SqliteStore) DeletePersonalAccessToken(
	ctx context.Context,
	p DeletePersonalAccessTokenParams,
) error {
	return s.q.DeletePersonalAccessToken(ctx, sqlitedb.DeletePersonalAccessTokenParams(p))
}

func (s *SqliteStore) DeleteExpiredPersonalAccessTokens(
	ctx context.Context,
	now sql.NullInt64,
) (int64, error) {
	return s.q.DeleteExpiredPersonalAccessTokens(ctx, now)
}

func (s *SqliteStore) CreateRecoveryCode(
	ctx context.Context,
	p CreateRecoveryCodeParams,
) (TotpRecoveryCode, error) {
	row, err := s.q.CreateRecoveryCode(ctx, sqlitedb.CreateRecoveryCodeParams(p))
	if err != nil {
		return TotpRecoveryCode{}, err
	}
	return fromSqliteTotpRecoveryCode(row), nil
}

func (s *SqliteStore) ListUnusedRecoveryCodes(
	ctx context.Context,
	actorID ids.ActorID,
) ([]TotpRecoveryCode, error) {
	rows, err := s.q.ListUnusedRecoveryCodes(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteTotpRecoveryCodeSlice(rows), nil
}

func (s *SqliteStore) ConsumeRecoveryCode(
	ctx context.Context,
	p ConsumeRecoveryCodeParams,
) (int64, error) {
	return s.q.ConsumeRecoveryCode(ctx, sqlitedb.ConsumeRecoveryCodeParams(p))
}

func (s *SqliteStore) DeleteRecoveryCodes(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteRecoveryCodes(ctx, int64(actorID))
}

func (s *SqliteStore) CountUnusedRecoveryCodes(
	ctx context.Context,
	actorID ids.ActorID,
) (int64, error) {
	return s.q.CountUnusedRecoveryCodes(ctx, int64(actorID))
}

func (s *SqliteStore) GetActorSetting(
	ctx context.Context,
	actorID ids.ActorID,
	key string,
) (string, error) {
	return s.q.GetActorSetting(ctx, sqlitedb.GetActorSettingParams(GetActorSettingParams{
		ActorID: int64(actorID),
		Key:     key,
	}))
}

func (s *SqliteStore) ListActorSettings(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListActorSettingsRow, error) {
	rows, err := s.q.ListActorSettings(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListActorSettingsRowSlice(rows), nil
}

func (s *SqliteStore) UpsertActorSetting(
	ctx context.Context,
	p UpsertActorSettingParams,
) error {
	return s.q.UpsertActorSetting(ctx, sqlitedb.UpsertActorSettingParams(p))
}

func (s *SqliteStore) DeleteActorSetting(
	ctx context.Context,
	actorID ids.ActorID,
	key string,
) error {
	return s.q.DeleteActorSetting(ctx, sqlitedb.DeleteActorSettingParams(DeleteActorSettingParams{
		ActorID: int64(actorID),
		Key:     key,
	}))
}

func (s *SqliteStore) InsertUserLog(ctx context.Context, p InsertUserLogParams) error {
	return s.q.InsertUserLog(ctx, sqlitedb.InsertUserLogParams(p))
}

func (s *SqliteStore) ListUserLogs(
	ctx context.Context,
	p ListUserLogsParams,
) ([]ListUserLogsRow, error) {
	rows, err := s.q.ListUserLogs(ctx, sqlitedb.ListUserLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteListUserLogsRowSlice(rows), nil
}

func (s *SqliteStore) DeleteUserLogs(ctx context.Context, actorID int64) error {
	return s.q.DeleteUserLogs(ctx, actorID)
}

func (s *SqliteStore) PruneUserLogs(ctx context.Context, beforeAt int64) error {
	return s.q.PruneUserLogs(ctx, beforeAt)
}

func (s *SqliteStore) InsertAgentDomainLog(
	ctx context.Context,
	p InsertAgentDomainLogParams,
) error {
	return s.q.InsertAgentDomainLog(ctx, sqlitedb.InsertAgentDomainLogParams(p))
}

func (s *SqliteStore) ListAgentDomainLogs(
	ctx context.Context,
	p ListAgentDomainLogsParams,
) ([]ListAgentDomainLogsRow, error) {
	rows, err := s.q.ListAgentDomainLogs(ctx, sqlitedb.ListAgentDomainLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentDomainLogsRowSlice(rows), nil
}

func (s *SqliteStore) DeleteAgentDomainLogs(ctx context.Context, actorID int64) error {
	return s.q.DeleteAgentDomainLogs(ctx, actorID)
}

func (s *SqliteStore) PruneAgentDomainLogs(ctx context.Context, beforeAt int64) error {
	return s.q.PruneAgentDomainLogs(ctx, beforeAt)
}

func (s *SqliteStore) ListDomainLogs(
	ctx context.Context,
	p ListDomainLogsParams,
) ([]ListDomainLogsRow, error) {
	rows, err := s.q.ListDomainLogs(ctx, sqlitedb.ListDomainLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteListDomainLogsRowSlice(rows), nil
}

func (s *SqliteStore) CreateIdentityProvider(
	ctx context.Context,
	p CreateIdentityProviderParams,
) (IdentityProvider, error) {
	row, err := s.q.CreateIdentityProvider(ctx, sqlitedb.CreateIdentityProviderParams(p))
	if err != nil {
		return IdentityProvider{}, err
	}
	return fromSqliteIdentityProvider(row), nil
}

func (s *SqliteStore) GetIdentityProviderByID(
	ctx context.Context,
	id int64,
) (IdentityProvider, error) {
	row, err := s.q.GetIdentityProviderByID(ctx, id)
	if err != nil {
		return IdentityProvider{}, err
	}
	return fromSqliteIdentityProvider(row), nil
}

func (s *SqliteStore) GetIdentityProviderBySlug(
	ctx context.Context,
	slug string,
) (IdentityProvider, error) {
	row, err := s.q.GetIdentityProviderBySlug(ctx, slug)
	if err != nil {
		return IdentityProvider{}, err
	}
	return fromSqliteIdentityProvider(row), nil
}

func (s *SqliteStore) ListIdentityProviders(ctx context.Context) ([]IdentityProvider, error) {
	rows, err := s.q.ListIdentityProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteIdentityProviderSlice(rows), nil
}

func (s *SqliteStore) ListEnabledIdentityProviders(
	ctx context.Context,
) ([]IdentityProvider, error) {
	rows, err := s.q.ListEnabledIdentityProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteIdentityProviderSlice(rows), nil
}

func (s *SqliteStore) UpdateIdentityProvider(
	ctx context.Context,
	p UpdateIdentityProviderParams,
) error {
	return s.q.UpdateIdentityProvider(ctx, sqlitedb.UpdateIdentityProviderParams(p))
}

func (s *SqliteStore) CreateExternalCredential(
	ctx context.Context,
	p CreateExternalCredentialParams,
) (ExternalCredential, error) {
	row, err := s.q.CreateExternalCredential(ctx, sqlitedb.CreateExternalCredentialParams(p))
	if err != nil {
		return ExternalCredential{}, err
	}
	return fromSqliteExternalCredential(row), nil
}

func (s *SqliteStore) GetExternalCredential(
	ctx context.Context,
	providerID int64,
	subject string,
) (ExternalCredential, error) {
	row, err := s.q.GetExternalCredential(
		ctx,
		sqlitedb.GetExternalCredentialParams(GetExternalCredentialParams{
			ProviderID: providerID,
			Subject:    subject,
		}),
	)
	if err != nil {
		return ExternalCredential{}, err
	}
	return fromSqliteExternalCredential(row), nil
}

func (s *SqliteStore) ListExternalCredentialsByActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ExternalCredential, error) {
	rows, err := s.q.ListExternalCredentialsByActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteExternalCredentialSlice(rows), nil
}

func (s *SqliteStore) DeleteExternalCredential(
	ctx context.Context,
	id int64,
	actorID ids.ActorID,
) error {
	return s.q.DeleteExternalCredential(
		ctx,
		sqlitedb.DeleteExternalCredentialParams(DeleteExternalCredentialParams{
			ID:      id,
			ActorID: int64(actorID),
		}),
	)
}

func (s *SqliteStore) CreateLDAPProvider(
	ctx context.Context,
	p CreateLDAPProviderParams,
) (LdapProvider, error) {
	row, err := s.q.CreateLDAPProvider(ctx, sqlitedb.CreateLDAPProviderParams(p))
	if err != nil {
		return LdapProvider{}, err
	}
	return fromSqliteLdapProvider(row), nil
}

func (s *SqliteStore) GetLDAPProviderByID(
	ctx context.Context,
	id int64,
) (LdapProvider, error) {
	row, err := s.q.GetLDAPProviderByID(ctx, id)
	if err != nil {
		return LdapProvider{}, err
	}
	return fromSqliteLdapProvider(row), nil
}

func (s *SqliteStore) GetLDAPProviderBySlug(
	ctx context.Context,
	slug string,
) (LdapProvider, error) {
	row, err := s.q.GetLDAPProviderBySlug(ctx, slug)
	if err != nil {
		return LdapProvider{}, err
	}
	return fromSqliteLdapProvider(row), nil
}

func (s *SqliteStore) ListLDAPProviders(ctx context.Context) ([]LdapProvider, error) {
	rows, err := s.q.ListLDAPProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteLdapProviderSlice(rows), nil
}

func (s *SqliteStore) ListEnabledLDAPProviders(ctx context.Context) ([]LdapProvider, error) {
	rows, err := s.q.ListEnabledLDAPProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteLdapProviderSlice(rows), nil
}

func (s *SqliteStore) UpdateLDAPProvider(
	ctx context.Context,
	p UpdateLDAPProviderParams,
) error {
	return s.q.UpdateLDAPProvider(ctx, sqlitedb.UpdateLDAPProviderParams(p))
}

func (s *SqliteStore) CreateLDAPCredential(
	ctx context.Context,
	p CreateLDAPCredentialParams,
) (LdapCredential, error) {
	row, err := s.q.CreateLDAPCredential(ctx, sqlitedb.CreateLDAPCredentialParams(p))
	if err != nil {
		return LdapCredential{}, err
	}
	return fromSqliteLdapCredential(row), nil
}

func (s *SqliteStore) GetLDAPCredential(
	ctx context.Context,
	providerID int64,
	username string,
) (LdapCredential, error) {
	row, err := s.q.GetLDAPCredential(ctx, sqlitedb.GetLDAPCredentialParams(GetLDAPCredentialParams{
		ProviderID: providerID,
		Username:   username,
	}))
	if err != nil {
		return LdapCredential{}, err
	}
	return fromSqliteLdapCredential(row), nil
}

func (s *SqliteStore) ListLDAPCredentialsByActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]LdapCredential, error) {
	rows, err := s.q.ListLDAPCredentialsByActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteLdapCredentialSlice(rows), nil
}

func (s *SqliteStore) DeleteLDAPCredential(
	ctx context.Context,
	id int64,
	actorID ids.ActorID,
) error {
	return s.q.DeleteLDAPCredential(
		ctx,
		sqlitedb.DeleteLDAPCredentialParams(DeleteLDAPCredentialParams{
			ID:      id,
			ActorID: int64(actorID),
		}),
	)
}

func (s *SqliteStore) AcceptInvitation(
	ctx context.Context,
	p AcceptInvitationParams,
) (int64, error) {
	return s.q.AcceptInvitation(ctx, sqlitedb.AcceptInvitationParams(p))
}

func (s *SqliteStore) UpdateInvitationTokenHash(
	ctx context.Context,
	p UpdateInvitationTokenHashParams,
) error {
	return s.q.UpdateInvitationTokenHash(ctx, sqlitedb.UpdateInvitationTokenHashParams(p))
}

func (s *SqliteStore) RevokeInvitation(
	ctx context.Context,
	p RevokeInvitationParams,
) (int64, error) {
	return s.q.RevokeInvitation(ctx, sqlitedb.RevokeInvitationParams(p))
}

func (s *SqliteStore) DeleteInvitationsByActor(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteInvitationsByActor(ctx, int64(actorID))
}

func (s *SqliteStore) DeleteExpiredInvitedActors(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredInvitedActors(ctx, now)
}

func (s *SqliteStore) DeleteExpiredInvitations(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredInvitations(ctx, now)
}

func (s *SqliteStore) UpsertProjectActorGrant(
	ctx context.Context,
	p UpsertProjectActorGrantParams,
) error {
	return s.q.UpsertProjectActorGrant(ctx, sqlitedb.UpsertProjectActorGrantParams(p))
}

func (s *SqliteStore) UpdateProjectActorGrantRole(
	ctx context.Context,
	p UpdateProjectActorGrantRoleParams,
) error {
	return s.q.UpdateProjectActorGrantRole(ctx, sqlitedb.UpdateProjectActorGrantRoleParams(p))
}

func (s *SqliteStore) RemoveProjectActorGrant(
	ctx context.Context,
	p RemoveProjectActorGrantParams,
) error {
	return s.q.RemoveProjectActorGrant(ctx, sqlitedb.RemoveProjectActorGrantParams(p))
}

func (s *SqliteStore) GetProjectActorGrant(
	ctx context.Context,
	p GetProjectActorGrantParams,
) (ProjectActorMembership, error) {
	row, err := s.q.GetProjectActorGrant(ctx, sqlitedb.GetProjectActorGrantParams(p))
	if err != nil {
		return ProjectActorMembership{}, err
	}
	return fromSqliteProjectActorMembership(row), nil
}

func (s *SqliteStore) ListProjectActorGrants(
	ctx context.Context,
	projectID string,
) ([]ListProjectActorGrantsRow, error) {
	rows, err := s.q.ListProjectActorGrants(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectActorGrantsRowSlice(rows), nil
}

func (s *SqliteStore) UpsertProjectTeamMembership(
	ctx context.Context,
	p UpsertProjectTeamMembershipParams,
) error {
	return s.q.UpsertProjectTeamMembership(ctx, sqlitedb.UpsertProjectTeamMembershipParams(p))
}

func (s *SqliteStore) UpdateProjectTeamMembershipRole(
	ctx context.Context,
	p UpdateProjectTeamMembershipRoleParams,
) error {
	return s.q.UpdateProjectTeamMembershipRole(
		ctx,
		sqlitedb.UpdateProjectTeamMembershipRoleParams(p),
	)
}

func (s *SqliteStore) RemoveProjectTeamMembership(
	ctx context.Context,
	p RemoveProjectTeamMembershipParams,
) error {
	return s.q.RemoveProjectTeamMembership(ctx, sqlitedb.RemoveProjectTeamMembershipParams(p))
}

func (s *SqliteStore) GetProjectTeamMembership(
	ctx context.Context,
	p GetProjectTeamMembershipParams,
) (ProjectTeamMembership, error) {
	row, err := s.q.GetProjectTeamMembership(ctx, sqlitedb.GetProjectTeamMembershipParams(p))
	if err != nil {
		return ProjectTeamMembership{}, err
	}
	return fromSqliteProjectTeamMembership(row), nil
}

func (s *SqliteStore) ListProjectTeamMemberships(
	ctx context.Context,
	projectID string,
) ([]ListProjectTeamMembershipsRow, error) {
	rows, err := s.q.ListProjectTeamMemberships(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectTeamMembershipsRowSlice(rows), nil
}

func (s *SqliteStore) ListProjectTeamMembershipsForTeam(
	ctx context.Context,
	teamID int64,
) ([]ListProjectTeamMembershipsForTeamRow, error) {
	rows, err := s.q.ListProjectTeamMembershipsForTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectTeamMembershipsForTeamRowSlice(rows), nil
}

func (s *SqliteStore) CountProjectsOwnedByTeam(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountProjectsOwnedByTeam(ctx, teamID)
}

func (s *SqliteStore) CreateProjectTeamInvitation(
	ctx context.Context,
	p CreateProjectTeamInvitationParams,
) (ProjectTeamInvitation, error) {
	row, err := s.q.CreateProjectTeamInvitation(ctx, sqlitedb.CreateProjectTeamInvitationParams(p))
	if err != nil {
		return ProjectTeamInvitation{}, err
	}
	return fromSqliteProjectTeamInvitation(row), nil
}

func (s *SqliteStore) GetProjectTeamInvitation(
	ctx context.Context,
	id int64,
) (ProjectTeamInvitation, error) {
	row, err := s.q.GetProjectTeamInvitation(ctx, id)
	if err != nil {
		return ProjectTeamInvitation{}, err
	}
	return fromSqliteProjectTeamInvitation(row), nil
}

func (s *SqliteStore) GetPendingProjectTeamInvitation(
	ctx context.Context,
	p GetPendingProjectTeamInvitationParams,
) (ProjectTeamInvitation, error) {
	row, err := s.q.GetPendingProjectTeamInvitation(
		ctx,
		sqlitedb.GetPendingProjectTeamInvitationParams(p),
	)
	if err != nil {
		return ProjectTeamInvitation{}, err
	}
	return fromSqliteProjectTeamInvitation(row), nil
}

func (s *SqliteStore) ListProjectTeamInvitations(
	ctx context.Context,
	projectID string,
) ([]ListProjectTeamInvitationsRow, error) {
	rows, err := s.q.ListProjectTeamInvitations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectTeamInvitationsRowSlice(rows), nil
}

func (s *SqliteStore) ListPendingProjectInvitationsForTeamOwner(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListPendingProjectInvitationsForTeamOwnerRow, error) {
	rows, err := s.q.ListPendingProjectInvitationsForTeamOwner(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListPendingProjectInvitationsForTeamOwnerRowSlice(rows), nil
}

func (s *SqliteStore) RespondProjectTeamInvitation(
	ctx context.Context,
	p RespondProjectTeamInvitationParams,
) (int64, error) {
	return s.q.RespondProjectTeamInvitation(ctx, sqlitedb.RespondProjectTeamInvitationParams(p))
}

func (s *SqliteStore) RevokeProjectTeamInvitation(
	ctx context.Context,
	p RevokeProjectTeamInvitationParams,
) (int64, error) {
	return s.q.RevokeProjectTeamInvitation(ctx, sqlitedb.RevokeProjectTeamInvitationParams(p))
}

func (s *SqliteStore) ExpireProjectTeamInvitations(ctx context.Context, now int64) (int64, error) {
	return s.q.ExpireProjectTeamInvitations(
		ctx,
		sqlitedb.ExpireProjectTeamInvitationsParams(ExpireProjectTeamInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now, Valid: true},
			Now:         now,
		}),
	)
}

func (s *SqliteStore) ListProjectsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Project, error) {
	rows, err := s.q.ListProjectsForActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteProjectSlice(rows), nil
}

func (s *SqliteStore) ListAgentProjectMemberships(
	ctx context.Context,
	agentID ids.AgentID,
) ([]ListAgentProjectMembershipsRow, error) {
	rows, err := s.q.ListAgentProjectMemberships(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentProjectMembershipsRowSlice(rows), nil
}

func (s *SqliteStore) ListProjectMemberActorIDs(
	ctx context.Context,
	projectID string,
) ([]int64, error) {
	return s.q.ListProjectMemberActorIDs(ctx, projectID)
}

func (s *SqliteStore) InsertProjectLog(ctx context.Context, p InsertProjectLogParams) error {
	return s.q.InsertProjectLog(ctx, sqlitedb.InsertProjectLogParams(p))
}

func (s *SqliteStore) ListProjectLogs(
	ctx context.Context,
	p ListProjectLogsParams,
) ([]ListProjectLogsRow, error) {
	rows, err := s.q.ListProjectLogs(ctx, sqlitedb.ListProjectLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectLogsRowSlice(rows), nil
}

func (s *SqliteStore) TouchAgentSeen(ctx context.Context, id ids.AgentID, at int64) error {
	return s.q.TouchAgentSeen(ctx, sqlitedb.TouchAgentSeenParams(TouchAgentSeenParams{
		LastSeenAt: sql.NullInt64{Int64: at, Valid: true},
		ID:         int64(id),
	}))
}

func (s *SqliteStore) CreateAgentCredential(
	ctx context.Context,
	p CreateAgentCredentialParams,
) (AgentCredential, error) {
	row, err := s.q.CreateAgentCredential(ctx, sqlitedb.CreateAgentCredentialParams(p))
	if err != nil {
		return AgentCredential{}, err
	}
	return fromSqliteAgentCredential(row), nil
}

func (s *SqliteStore) ListAgentCredentials(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentCredential, error) {
	rows, err := s.q.ListAgentCredentials(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteAgentCredentialSlice(rows), nil
}

func (s *SqliteStore) GetAgentCredential(
	ctx context.Context,
	username string,
) (AgentCredential, error) {
	row, err := s.q.GetAgentCredential(ctx, username)
	if err != nil {
		return AgentCredential{}, err
	}
	return fromSqliteAgentCredential(row), nil
}

func (s *SqliteStore) RevokeAgentCredential(
	ctx context.Context,
	p RevokeAgentCredentialParams,
) (int64, error) {
	return s.q.RevokeAgentCredential(ctx, sqlitedb.RevokeAgentCredentialParams(p))
}

func (s *SqliteStore) TouchAgentCredentialUsed(
	ctx context.Context,
	p TouchAgentCredentialUsedParams,
) error {
	return s.q.TouchAgentCredentialUsed(ctx, sqlitedb.TouchAgentCredentialUsedParams(p))
}

func (s *SqliteStore) ListSettings(ctx context.Context) ([]Setting, error) {
	rows, err := s.q.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteSettingSlice(rows), nil
}

func (s *SqliteStore) GetSetting(ctx context.Context, key string) (string, error) {
	return s.q.GetSetting(ctx, key)
}

func (s *SqliteStore) UpsertSetting(ctx context.Context, p UpsertSettingParams) error {
	return s.q.UpsertSetting(ctx, sqlitedb.UpsertSettingParams(p))
}

func (s *SqliteStore) ClearSettingValue(ctx context.Context, key, value string) error {
	return s.q.ClearSettingValue(ctx, sqlitedb.ClearSettingValueParams(ClearSettingValueParams{
		Key:   key,
		Value: value,
	}))
}

func (s *SqliteStore) CreateProject(
	ctx context.Context,
	p CreateProjectParams,
) (Project, error) {
	row, err := s.q.CreateProject(ctx, sqlitedb.CreateProjectParams(p))
	if err != nil {
		return Project{}, err
	}
	return fromSqliteProject(row), nil
}

func (s *SqliteStore) GetProject(ctx context.Context, id string) (Project, error) {
	row, err := s.q.GetProject(ctx, id)
	if err != nil {
		return Project{}, err
	}
	return fromSqliteProject(row), nil
}

func (s *SqliteStore) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	row, err := s.q.GetProjectBySlug(ctx, slug)
	if err != nil {
		return Project{}, err
	}
	return fromSqliteProject(row), nil
}

func (s *SqliteStore) GetProjectByPrefix(
	ctx context.Context,
	prefix string,
) (Project, error) {
	row, err := s.q.GetProjectByPrefix(ctx, prefix)
	if err != nil {
		return Project{}, err
	}
	return fromSqliteProject(row), nil
}

func (s *SqliteStore) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.q.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteProjectSlice(rows), nil
}

func (s *SqliteStore) SetProjectArchived(
	ctx context.Context,
	p SetProjectArchivedParams,
) (Project, error) {
	row, err := s.q.SetProjectArchived(ctx, sqlitedb.SetProjectArchivedParams(p))
	if err != nil {
		return Project{}, err
	}
	return fromSqliteProject(row), nil
}

func (s *SqliteStore) AllocateTaskNumber(ctx context.Context, projectID string) (int64, error) {
	return s.q.AllocateTaskNumber(ctx, projectID)
}

func (s *SqliteStore) ListAgentProjects(
	ctx context.Context,
	agentID ids.AgentID,
) ([]Project, error) {
	rows, err := s.q.ListAgentProjects(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteProjectSlice(rows), nil
}

func (s *SqliteStore) InsertPrompt(
	ctx context.Context,
	p InsertPromptParams,
) (Prompt, error) {
	row, err := s.q.InsertPrompt(ctx, sqlitedb.InsertPromptParams(p))
	if err != nil {
		return Prompt{}, err
	}
	return fromSqlitePrompt(row), nil
}

func (s *SqliteStore) GetPrompt(ctx context.Context, id int64) (Prompt, error) {
	row, err := s.q.GetPrompt(ctx, id)
	if err != nil {
		return Prompt{}, err
	}
	return fromSqlitePrompt(row), nil
}

func (s *SqliteStore) LatestPrompt(
	ctx context.Context,
	p LatestPromptParams,
) (Prompt, error) {
	row, err := s.q.LatestPrompt(ctx, sqlitedb.LatestPromptParams(p))
	if err != nil {
		return Prompt{}, err
	}
	return fromSqlitePrompt(row), nil
}

func (s *SqliteStore) LatestPromptVersion(
	ctx context.Context,
	p LatestPromptVersionParams,
) (int64, error) {
	return s.q.LatestPromptVersion(ctx, sqlitedb.LatestPromptVersionParams(p))
}

func (s *SqliteStore) CreateTask(
	ctx context.Context,
	p CreateTaskParams,
) (Task, error) {
	row, err := s.q.CreateTask(ctx, sqlitedb.CreateTaskParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) GetTask(ctx context.Context, id string) (Task, error) {
	row, err := s.q.GetTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) GetTaskByProjectSeq(
	ctx context.Context,
	projectID string,
	seq int64,
) (Task, error) {
	row, err := s.q.GetTaskByProjectSeq(
		ctx,
		sqlitedb.GetTaskByProjectSeqParams(GetTaskByProjectSeqParams{
			ProjectID: sql.NullString{String: projectID, Valid: true},
			Seq:       seq,
		}),
	)
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) ListTasks(
	ctx context.Context,
	p ListTasksParams,
) ([]Task, error) {
	rows, err := s.q.ListTasks(ctx, sqlitedb.ListTasksParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) ListQueuedTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.q.ListQueuedTasks(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) ListRunningTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.q.ListRunningTasks(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) UpdateDraftTask(
	ctx context.Context,
	p UpdateDraftTaskParams,
) (Task, error) {
	row, err := s.q.UpdateDraftTask(ctx, sqlitedb.UpdateDraftTaskParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) ClearTaskPendingPrompt(
	ctx context.Context,
	p ClearTaskPendingPromptParams,
) (int64, error) {
	return s.q.ClearTaskPendingPrompt(ctx, sqlitedb.ClearTaskPendingPromptParams(p))
}

func (s *SqliteStore) SetTaskPrompt(
	ctx context.Context,
	p SetTaskPromptParams,
) (int64, error) {
	return s.q.SetTaskPrompt(ctx, sqlitedb.SetTaskPromptParams(p))
}

func (s *SqliteStore) SetTaskPendingPrompt(
	ctx context.Context,
	p SetTaskPendingPromptParams,
) (int64, error) {
	return s.q.SetTaskPendingPrompt(ctx, sqlitedb.SetTaskPendingPromptParams(p))
}

func (s *SqliteStore) TaskIDBySession(
	ctx context.Context,
	p TaskIDBySessionParams,
) (string, error) {
	return s.q.TaskIDBySession(ctx, sqlitedb.TaskIDBySessionParams(p))
}

func (s *SqliteStore) SessionAgentID(
	ctx context.Context,
	sessionID string,
) (ids.AgentID, error) {
	id, err := s.q.SessionAgentID(ctx, sessionID)
	return ids.AgentID(id), err
}

func (s *SqliteStore) SetTaskError(ctx context.Context, p SetTaskErrorParams) (int64, error) {
	return s.q.SetTaskError(ctx, sqlitedb.SetTaskErrorParams(p))
}

func (s *SqliteStore) SetTaskSession(
	ctx context.Context,
	p SetTaskSessionParams,
) (int64, error) {
	return s.q.SetTaskSession(ctx, sqlitedb.SetTaskSessionParams(p))
}

func (s *SqliteStore) AppendTaskSessionEntry(
	ctx context.Context,
	p AppendTaskSessionEntryParams,
) (int64, error) {
	return s.q.AppendTaskSessionEntry(ctx, sqlitedb.AppendTaskSessionEntryParams(p))
}

func (s *SqliteStore) ListTaskSessionEntries(
	ctx context.Context,
	taskID string,
) ([]TaskSessionEntry, error) {
	rows, err := s.q.ListTaskSessionEntries(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSessionEntrySlice(rows), nil
}

func (s *SqliteStore) CountTaskSessionEntries(ctx context.Context, taskID string) (int64, error) {
	return s.q.CountTaskSessionEntries(ctx, taskID)
}

func (s *SqliteStore) ListTaskHistoryTaskIDs(ctx context.Context) ([]string, error) {
	return s.q.ListTaskHistoryTaskIDs(ctx)
}

func (s *SqliteStore) TaskSessionEntryTip(
	ctx context.Context,
	taskID string,
) (TaskSessionEntryTipRow, error) {
	row, err := s.q.TaskSessionEntryTip(ctx, taskID)
	if err != nil {
		return TaskSessionEntryTipRow{}, err
	}
	return fromSqliteTaskSessionEntryTipRow(row), nil
}

func (s *SqliteStore) RenameTask(
	ctx context.Context,
	p RenameTaskParams,
) (Task, error) {
	row, err := s.q.RenameTask(ctx, sqlitedb.RenameTaskParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) SetTaskPriority(
	ctx context.Context,
	p SetTaskPriorityParams,
) (Task, error) {
	row, err := s.q.SetTaskPriority(ctx, sqlitedb.SetTaskPriorityParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) SetTaskSize(
	ctx context.Context,
	p SetTaskSizeParams,
) (Task, error) {
	row, err := s.q.SetTaskSize(ctx, sqlitedb.SetTaskSizeParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) SetTaskAssignee(
	ctx context.Context,
	p SetTaskAssigneeParams,
) (Task, error) {
	row, err := s.q.SetTaskAssignee(ctx, sqlitedb.SetTaskAssigneeParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) MarkTaskSessionDeleted(
	ctx context.Context,
	p MarkTaskSessionDeletedParams,
) (Task, error) {
	row, err := s.q.MarkTaskSessionDeleted(ctx, sqlitedb.MarkTaskSessionDeletedParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromSqliteTask(row), nil
}

func (s *SqliteStore) SetTaskNamePrompt(
	ctx context.Context,
	p SetTaskNamePromptParams,
) error {
	return s.q.SetTaskNamePrompt(ctx, sqlitedb.SetTaskNamePromptParams(p))
}

func (s *SqliteStore) ListTasksByProject(
	ctx context.Context,
	projectID sql.NullString,
) ([]Task, error) {
	rows, err := s.q.ListTasksByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) ListTasksByProjectAgent(
	ctx context.Context,
	p ListTasksByProjectAgentParams,
) ([]Task, error) {
	rows, err := s.q.ListTasksByProjectAgent(ctx, sqlitedb.ListTasksByProjectAgentParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) CountTasksByProject(
	ctx context.Context,
	projectID sql.NullString,
) (int64, error) {
	return s.q.CountTasksByProject(ctx, projectID)
}

func (s *SqliteStore) ListRunningTasksByAgent(
	ctx context.Context,
	agentID ids.AgentID,
) ([]Task, error) {
	rows, err := s.q.ListRunningTasksByAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) ListRunningTaskTagsByAgent(
	ctx context.Context,
	agentID ids.AgentID,
) ([]ListRunningTaskTagsByAgentRow, error) {
	rows, err := s.q.ListRunningTaskTagsByAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListRunningTaskTagsByAgentRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskLinks(ctx context.Context) ([]ListTaskLinksRow, error) {
	rows, err := s.q.ListTaskLinks(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskLinksRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskLinksByTask(
	ctx context.Context,
	taskID string,
) ([]ListTaskLinksByTaskRow, error) {
	rows, err := s.q.ListTaskLinksByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskLinksByTaskRowSlice(rows), nil
}

func (s *SqliteStore) DeleteTaskLinks(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskLinks(ctx, taskID)
}

func (s *SqliteStore) InsertTaskLink(ctx context.Context, p InsertTaskLinkParams) error {
	return s.q.InsertTaskLink(ctx, sqlitedb.InsertTaskLinkParams(p))
}

func (s *SqliteStore) InsertTaskLog(ctx context.Context, p InsertTaskLogParams) error {
	return s.q.InsertTaskLog(ctx, sqlitedb.InsertTaskLogParams(p))
}

func (s *SqliteStore) ListTaskLogsByTask(
	ctx context.Context,
	taskID string,
) ([]ListTaskLogsByTaskRow, error) {
	rows, err := s.q.ListTaskLogsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskLogsByTaskRowSlice(rows), nil
}

func (s *SqliteStore) AddTaskReference(
	ctx context.Context,
	p UpsertTaskReferenceParams,
) error {
	return s.q.UpsertTaskReference(ctx, sqlitedb.UpsertTaskReferenceParams(p))
}

func (s *SqliteStore) RemoveTaskReference(
	ctx context.Context,
	p RemoveTaskReferenceParams,
) (int64, error) {
	return s.q.RemoveTaskReference(ctx, sqlitedb.RemoveTaskReferenceParams(p))
}

func (s *SqliteStore) ListTaskReferences(ctx context.Context, taskID string) ([]Task, error) {
	rows, err := s.q.ListTaskReferences(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) ListTaskReferencedBy(
	ctx context.Context,
	taskID string,
) ([]Task, error) {
	rows, err := s.q.ListTaskReferencedBy(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskSlice(rows), nil
}

func (s *SqliteStore) ListTaskReferenceIDs(ctx context.Context, taskID string) ([]string, error) {
	return s.q.ListTaskReferenceIDs(ctx, taskID)
}

func (s *SqliteStore) CountIncompleteBlockingReferences(
	ctx context.Context,
	taskID string,
) (int64, error) {
	return s.q.CountIncompleteBlockingReferences(ctx, taskID)
}

func (s *SqliteStore) ListAllBlockingTaskReferences(
	ctx context.Context,
) ([]ListAllBlockingTaskReferencesRow, error) {
	rows, err := s.q.ListAllBlockingTaskReferences(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAllBlockingTaskReferencesRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskReferenceRefs(
	ctx context.Context,
) ([]ListTaskReferenceRefsRow, error) {
	rows, err := s.q.ListTaskReferenceRefs(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskReferenceRefsRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskReferencedByRefs(
	ctx context.Context,
) ([]ListTaskReferencedByRefsRow, error) {
	rows, err := s.q.ListTaskReferencedByRefs(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskReferencedByRefsRowSlice(rows), nil
}

func (s *SqliteStore) CreateTag(
	ctx context.Context,
	p CreateTagParams,
) (Tag, error) {
	row, err := s.q.CreateTag(ctx, sqlitedb.CreateTagParams(p))
	if err != nil {
		return Tag{}, err
	}
	return fromSqliteTag(row), nil
}

func (s *SqliteStore) GetTag(ctx context.Context, id string) (Tag, error) {
	row, err := s.q.GetTag(ctx, id)
	if err != nil {
		return Tag{}, err
	}
	return fromSqliteTag(row), nil
}

func (s *SqliteStore) GetTagByName(
	ctx context.Context,
	projectID, name string,
) (Tag, error) {
	row, err := s.q.GetTagByName(ctx, sqlitedb.GetTagByNameParams(GetTagByNameParams{
		ProjectID: sql.NullString{String: projectID, Valid: projectID != ""},
		Name:      name,
	}))
	if err != nil {
		return Tag{}, err
	}
	return fromSqliteTag(row), nil
}

func (s *SqliteStore) GetTagWithCount(
	ctx context.Context,
	id string,
) (GetTagWithCountRow, error) {
	row, err := s.q.GetTagWithCount(ctx, id)
	if err != nil {
		return GetTagWithCountRow{}, err
	}
	return fromSqliteGetTagWithCountRow(row), nil
}

func (s *SqliteStore) UpdateTag(
	ctx context.Context,
	p UpdateTagParams,
) (Tag, error) {
	row, err := s.q.UpdateTag(ctx, sqlitedb.UpdateTagParams(p))
	if err != nil {
		return Tag{}, err
	}
	return fromSqliteTag(row), nil
}

func (s *SqliteStore) SetTagArchived(
	ctx context.Context,
	p SetTagArchivedParams,
) (Tag, error) {
	row, err := s.q.SetTagArchived(ctx, sqlitedb.SetTagArchivedParams(p))
	if err != nil {
		return Tag{}, err
	}
	return fromSqliteTag(row), nil
}

func (s *SqliteStore) DeleteTagsByProject(ctx context.Context, projectID string) error {
	return s.q.DeleteTagsByProject(ctx, sql.NullString{
		String: projectID,
		Valid:  projectID != "",
	})
}

func (s *SqliteStore) ListTaskTags(ctx context.Context) ([]ListTaskTagsRow, error) {
	rows, err := s.q.ListTaskTags(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskTagsRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskTagsByTask(
	ctx context.Context,
	taskID string,
) ([]ListTaskTagsByTaskRow, error) {
	rows, err := s.q.ListTaskTagsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskTagsByTaskRowSlice(rows), nil
}

func (s *SqliteStore) DeleteTaskTags(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskTags(ctx, taskID)
}

func (s *SqliteStore) AddTaskTag(ctx context.Context, p AddTaskTagParams) error {
	return s.q.AddTaskTag(ctx, sqlitedb.AddTaskTagParams(p))
}

func (s *SqliteStore) DeleteTaskTag(ctx context.Context, p DeleteTaskTagParams) error {
	return s.q.DeleteTaskTag(ctx, sqlitedb.DeleteTaskTagParams(p))
}

func (s *SqliteStore) GetAgentTaskLimits(
	ctx context.Context,
	agentID ids.AgentID,
) (AgentTaskLimit, error) {
	row, err := s.q.GetAgentTaskLimits(ctx, int64(agentID))
	if err != nil {
		return AgentTaskLimit{}, err
	}
	return fromSqliteAgentTaskLimit(row), nil
}

func (s *SqliteStore) UpsertAgentTaskLimits(
	ctx context.Context,
	p UpsertAgentTaskLimitsParams,
) error {
	return s.q.UpsertAgentTaskLimits(ctx, sqlitedb.UpsertAgentTaskLimitsParams(p))
}

func (s *SqliteStore) ListAgentTagLimits(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentTagLimit, error) {
	rows, err := s.q.ListAgentTagLimits(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteAgentTagLimitSlice(rows), nil
}

func (s *SqliteStore) DeleteAgentTagLimits(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentTagLimits(ctx, int64(agentID))
}

func (s *SqliteStore) InsertAgentTagLimit(
	ctx context.Context,
	p InsertAgentTagLimitParams,
) error {
	return s.q.InsertAgentTagLimit(ctx, sqlitedb.InsertAgentTagLimitParams(p))
}

func (s *SqliteStore) ListAgentProjectLimits(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentProjectLimit, error) {
	rows, err := s.q.ListAgentProjectLimits(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteAgentProjectLimitSlice(rows), nil
}

func (s *SqliteStore) DeleteAgentProjectLimits(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentProjectLimits(ctx, int64(agentID))
}

func (s *SqliteStore) InsertAgentProjectLimit(
	ctx context.Context,
	p InsertAgentProjectLimitParams,
) error {
	return s.q.InsertAgentProjectLimit(ctx, sqlitedb.InsertAgentProjectLimitParams(p))
}

func (s *SqliteStore) ListAgentProjectTagLimits(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentProjectTagLimit, error) {
	rows, err := s.q.ListAgentProjectTagLimits(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromSqliteAgentProjectTagLimitSlice(rows), nil
}

func (s *SqliteStore) DeleteAgentProjectTagLimits(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentProjectTagLimits(ctx, int64(agentID))
}

func (s *SqliteStore) InsertAgentProjectTagLimit(
	ctx context.Context,
	p InsertAgentProjectTagLimitParams,
) error {
	return s.q.InsertAgentProjectTagLimit(ctx, sqlitedb.InsertAgentProjectTagLimitParams(p))
}

func (s *SqliteStore) CreateWebhook(
	ctx context.Context,
	p CreateWebhookParams,
) (Webhook, error) {
	row, err := s.q.CreateWebhook(ctx, sqlitedb.CreateWebhookParams(p))
	if err != nil {
		return Webhook{}, err
	}
	return fromSqliteWebhook(row), nil
}

func (s *SqliteStore) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.q.ListWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteWebhookSlice(rows), nil
}

func (s *SqliteStore) GetWebhook(ctx context.Context, id int64) (Webhook, error) {
	row, err := s.q.GetWebhook(ctx, id)
	if err != nil {
		return Webhook{}, err
	}
	return fromSqliteWebhook(row), nil
}

func (s *SqliteStore) UpdateWebhook(
	ctx context.Context,
	p UpdateWebhookParams,
) (Webhook, error) {
	row, err := s.q.UpdateWebhook(ctx, sqlitedb.UpdateWebhookParams(p))
	if err != nil {
		return Webhook{}, err
	}
	return fromSqliteWebhook(row), nil
}

func (s *SqliteStore) DeleteWebhook(ctx context.Context, id int64) error {
	return s.q.DeleteWebhook(ctx, id)
}

func (s *SqliteStore) CreateNotification(
	ctx context.Context,
	p CreateNotificationParams,
) (Notification, error) {
	row, err := s.q.CreateNotification(ctx, sqlitedb.CreateNotificationParams(p))
	if err != nil {
		return Notification{}, err
	}
	return fromSqliteNotification(row), nil
}

func (s *SqliteStore) RefreshUnreadNotification(
	ctx context.Context,
	p RefreshUnreadNotificationParams,
) (Notification, error) {
	row, err := s.q.RefreshUnreadNotification(ctx, sqlitedb.RefreshUnreadNotificationParams(p))
	if err != nil {
		return Notification{}, err
	}
	return fromSqliteNotification(row), nil
}

func (s *SqliteStore) ListNotifications(
	ctx context.Context,
	recipient, limit int64,
) ([]ListNotificationsRow, error) {
	rows, err := s.q.ListNotifications(
		ctx,
		sqlitedb.ListNotificationsParams(ListNotificationsParams{
			RecipientActorID: nullActorID(recipient),
			MaxCount:         limit,
		}),
	)
	if err != nil {
		return nil, err
	}
	return fromSqliteListNotificationsRowSlice(rows), nil
}

func (s *SqliteStore) CountUnreadNotifications(
	ctx context.Context,
	recipient int64,
) (int64, error) {
	return s.q.CountUnreadNotifications(ctx, nullActorID(recipient))
}

func (s *SqliteStore) MarkNotificationRead(
	ctx context.Context,
	p MarkNotificationReadParams,
) error {
	return s.q.MarkNotificationRead(ctx, sqlitedb.MarkNotificationReadParams(p))
}

func (s *SqliteStore) MarkTaskNotificationsRead(
	ctx context.Context,
	p MarkTaskNotificationsReadParams,
) error {
	return s.q.MarkTaskNotificationsRead(ctx, sqlitedb.MarkTaskNotificationsReadParams(p))
}

func (s *SqliteStore) MarkAllNotificationsRead(
	ctx context.Context,
	readAt sql.NullInt64,
	recipient int64,
) error {
	return s.q.MarkAllNotificationsRead(
		ctx,
		sqlitedb.MarkAllNotificationsReadParams(MarkAllNotificationsReadParams{
			ReadAt:           readAt,
			RecipientActorID: nullActorID(recipient),
		}),
	)
}

func (s *SqliteStore) DeleteNotification(ctx context.Context, id, recipient int64) error {
	return s.q.DeleteNotification(ctx, sqlitedb.DeleteNotificationParams(DeleteNotificationParams{
		ID:               id,
		RecipientActorID: nullActorID(recipient),
	}))
}

func (s *SqliteStore) DeleteAllNotifications(ctx context.Context, recipient int64) error {
	return s.q.DeleteAllNotifications(ctx, nullActorID(recipient))
}

func (s *SqliteStore) InsertAgentUsage(ctx context.Context, p InsertAgentUsageParams) error {
	return s.q.InsertAgentUsage(ctx, sqlitedb.InsertAgentUsageParams(p))
}

func (s *SqliteStore) ListAgentUsage(
	ctx context.Context,
	agentID sql.NullInt64,
	projectID sql.NullString,
) ([]ListAgentUsageRow, error) {
	rows, err := s.q.ListAgentUsage(ctx, sqlitedb.ListAgentUsageParams(ListAgentUsageParams{
		AgentID:   agentID,
		ProjectID: projectID,
	}))
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentUsageRowSlice(rows), nil
}

func (s *SqliteStore) SumAgentUsage(ctx context.Context) (SumAgentUsageRow, error) {
	row, err := s.q.SumAgentUsage(ctx)
	if err != nil {
		return SumAgentUsageRow{}, err
	}
	return fromSqliteSumAgentUsageRow(row), nil
}

func (s *SqliteStore) ListAgentUsageByDay(
	ctx context.Context,
	agentID sql.NullInt64,
	projectID sql.NullString,
) ([]ListAgentUsageByDayRow, error) {
	rows, err := s.q.ListAgentUsageByDay(
		ctx,
		sqlitedb.ListAgentUsageByDayParams(ListAgentUsageByDayParams{
			AgentID:   agentID,
			ProjectID: projectID,
		}),
	)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentUsageByDayRowSlice(rows), nil
}

func (s *SqliteStore) UpdateAgentCredentialPassword(
	ctx context.Context,
	p UpdateAgentCredentialPasswordParams,
) error {
	return s.q.UpdateAgentCredentialPassword(ctx, sqlitedb.UpdateAgentCredentialPasswordParams(p))
}

func (s *SqliteStore) CreateAttachment(
	ctx context.Context,
	p CreateAttachmentParams,
) (Attachment, error) {
	row, err := s.q.CreateAttachment(ctx, sqlitedb.CreateAttachmentParams(p))
	if err != nil {
		return Attachment{}, err
	}
	return fromSqliteAttachment(row), nil
}

func (s *SqliteStore) GetAttachment(ctx context.Context, id int64) (Attachment, error) {
	row, err := s.q.GetAttachment(ctx, id)
	if err != nil {
		return Attachment{}, err
	}
	return fromSqliteAttachment(row), nil
}

func (s *SqliteStore) SetAttachmentSize(ctx context.Context, id, size int64) error {
	return s.q.SetAttachmentSize(ctx, sqlitedb.SetAttachmentSizeParams(SetAttachmentSizeParams{
		Size: size,
		ID:   id,
	}))
}

func (s *SqliteStore) ListOrphanAttachments(ctx context.Context) ([]Attachment, error) {
	rows, err := s.q.ListOrphanAttachments(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteAttachmentSlice(rows), nil
}

func (s *SqliteStore) LinkTaskAttachment(
	ctx context.Context,
	p LinkTaskAttachmentParams,
) error {
	return s.q.LinkTaskAttachment(ctx, sqlitedb.LinkTaskAttachmentParams(p))
}

func (s *SqliteStore) UnlinkTaskAttachment(
	ctx context.Context,
	taskID string,
	attachmentID int64,
) error {
	return s.q.UnlinkTaskAttachment(
		ctx,
		sqlitedb.UnlinkTaskAttachmentParams(UnlinkTaskAttachmentParams{
			TaskID:       taskID,
			AttachmentID: attachmentID,
		}),
	)
}

func (s *SqliteStore) ListTaskAttachments(
	ctx context.Context,
	taskID string,
) ([]ListTaskAttachmentsRow, error) {
	rows, err := s.q.ListTaskAttachments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskAttachmentsRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskAttachmentIDs(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskAttachmentIDs(ctx, taskID)
}

func (s *SqliteStore) CreateTaskMessage(
	ctx context.Context,
	p CreateTaskMessageParams,
) error {
	return s.q.CreateTaskMessage(ctx, sqlitedb.CreateTaskMessageParams(p))
}

func (s *SqliteStore) ListTaskMessages(
	ctx context.Context,
	taskID string,
) ([]ListTaskMessagesRow, error) {
	rows, err := s.q.ListTaskMessages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskMessagesRowSlice(rows), nil
}

func (s *SqliteStore) LinkMessageAttachment(
	ctx context.Context,
	p LinkMessageAttachmentParams,
) error {
	return s.q.LinkMessageAttachment(ctx, sqlitedb.LinkMessageAttachmentParams(p))
}

func (s *SqliteStore) SetProjectPicture(
	ctx context.Context,
	p SetProjectPictureParams,
) error {
	return s.q.SetProjectPicture(ctx, sqlitedb.SetProjectPictureParams(p))
}

func (s *SqliteStore) ClearProjectPicture(
	ctx context.Context,
	p ClearProjectPictureParams,
) error {
	return s.q.ClearProjectPicture(ctx, sqlitedb.ClearProjectPictureParams(p))
}

func (s *SqliteStore) SetAgentPicture(ctx context.Context, p SetAgentPictureParams) error {
	return s.q.SetAgentPicture(ctx, sqlitedb.SetAgentPictureParams(p))
}

func (s *SqliteStore) ClearAgentPicture(
	ctx context.Context,
	p ClearAgentPictureParams,
) error {
	return s.q.ClearAgentPicture(ctx, sqlitedb.ClearAgentPictureParams(p))
}

func (s *SqliteStore) SetUserPicture(ctx context.Context, p SetUserPictureParams) error {
	return s.q.SetUserPicture(ctx, sqlitedb.SetUserPictureParams(p))
}

func (s *SqliteStore) ClearUserPicture(ctx context.Context, p ClearUserPictureParams) error {
	return s.q.ClearUserPicture(ctx, sqlitedb.ClearUserPictureParams(p))
}

func (s *SqliteStore) GetUserPicture(
	ctx context.Context,
	actorID ids.ActorID,
) (Attachment, error) {
	row, err := s.q.GetUserPicture(ctx, int64(actorID))
	if err != nil {
		return Attachment{}, err
	}
	return fromSqliteAttachment(row), nil
}

func (s *SqliteStore) ListUserPictures(ctx context.Context) ([]ListUserPicturesRow, error) {
	rows, err := s.q.ListUserPictures(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListUserPicturesRowSlice(rows), nil
}

func (s *SqliteStore) GetProjectPicture(
	ctx context.Context,
	projectID string,
) (Attachment, error) {
	row, err := s.q.GetProjectPicture(ctx, projectID)
	if err != nil {
		return Attachment{}, err
	}
	return fromSqliteAttachment(row), nil
}

func (s *SqliteStore) GetAgentPicture(
	ctx context.Context,
	agentID ids.AgentID,
) (Attachment, error) {
	row, err := s.q.GetAgentPicture(ctx, int64(agentID))
	if err != nil {
		return Attachment{}, err
	}
	return fromSqliteAttachment(row), nil
}

func (s *SqliteStore) ListProjectPictures(
	ctx context.Context,
) ([]ListProjectPicturesRow, error) {
	rows, err := s.q.ListProjectPictures(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListProjectPicturesRowSlice(rows), nil
}

func (s *SqliteStore) ListAgentPictures(ctx context.Context) ([]ListAgentPicturesRow, error) {
	rows, err := s.q.ListAgentPictures(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentPicturesRowSlice(rows), nil
}

func (s *SqliteStore) SetAttachmentStorageKey(
	ctx context.Context,
	id int64,
	storageKey string,
) error {
	return s.q.SetAttachmentStorageKey(
		ctx,
		sqlitedb.SetAttachmentStorageKeyParams(SetAttachmentStorageKeyParams{
			StorageKey: storageKey,
			ID:         id,
		}),
	)
}

func (s *SqliteStore) ListTaskArtifacts(
	ctx context.Context,
	taskID string,
) ([]TaskArtifact, error) {
	rows, err := s.q.ListTaskArtifacts(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskArtifactSlice(rows), nil
}

func (s *SqliteStore) ListArtifactParts(
	ctx context.Context,
	taskID string,
) ([]ArtifactPart, error) {
	rows, err := s.q.ListArtifactParts(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteArtifactPartSlice(rows), nil
}

func (s *SqliteStore) GetTaskArtifact(
	ctx context.Context,
	taskID, artifactID string,
) (TaskArtifact, error) {
	row, err := s.q.GetTaskArtifact(ctx, sqlitedb.GetTaskArtifactParams(GetTaskArtifactParams{
		TaskID:     taskID,
		ArtifactID: artifactID,
	}))
	if err != nil {
		return TaskArtifact{}, err
	}
	return fromSqliteTaskArtifact(row), nil
}

func (s *SqliteStore) MaxTaskArtifactPosition(ctx context.Context, taskID string) (int64, error) {
	return s.q.MaxTaskArtifactPosition(ctx, taskID)
}

func (s *SqliteStore) MaxArtifactPartPosition(
	ctx context.Context,
	taskID, artifactID string,
) (int64, error) {
	return s.q.MaxArtifactPartPosition(
		ctx,
		sqlitedb.MaxArtifactPartPositionParams(MaxArtifactPartPositionParams{
			TaskID:     taskID,
			ArtifactID: artifactID,
		}),
	)
}

func (s *SqliteStore) UpsertTaskArtifact(
	ctx context.Context,
	p UpsertTaskArtifactParams,
) error {
	return s.q.UpsertTaskArtifact(ctx, sqlitedb.UpsertTaskArtifactParams(p))
}

func (s *SqliteStore) SetTaskArtifactStreaming(
	ctx context.Context,
	p SetTaskArtifactStreamingParams,
) error {
	return s.q.SetTaskArtifactStreaming(ctx, sqlitedb.SetTaskArtifactStreamingParams(p))
}

func (s *SqliteStore) InsertArtifactPart(
	ctx context.Context,
	p InsertArtifactPartParams,
) error {
	return s.q.InsertArtifactPart(ctx, sqlitedb.InsertArtifactPartParams(p))
}

func (s *SqliteStore) DeleteArtifactParts(ctx context.Context, taskID, artifactID string) error {
	return s.q.DeleteArtifactParts(
		ctx,
		sqlitedb.DeleteArtifactPartsParams(DeleteArtifactPartsParams{
			TaskID:     taskID,
			ArtifactID: artifactID,
		}),
	)
}

func (s *SqliteStore) DeleteTaskArtifact(ctx context.Context, taskID, artifactID string) error {
	return s.q.DeleteTaskArtifact(ctx, sqlitedb.DeleteTaskArtifactParams(DeleteTaskArtifactParams{
		TaskID:     taskID,
		ArtifactID: artifactID,
	}))
}

func (s *SqliteStore) DeleteTaskArtifacts(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskArtifacts(ctx, taskID)
}

func (s *SqliteStore) DeleteArtifactPartsByTask(ctx context.Context, taskID string) error {
	return s.q.DeleteArtifactPartsByTask(ctx, taskID)
}

func (s *SqliteStore) DeleteArtifactPartsByAttachment(
	ctx context.Context,
	attachmentID int64,
) error {
	return s.q.DeleteArtifactPartsByAttachment(ctx, sql.NullInt64{
		Int64: attachmentID,
		Valid: true,
	})
}

func (s *SqliteStore) DeleteEmptyTaskArtifacts(ctx context.Context) error {
	return s.q.DeleteEmptyTaskArtifacts(ctx)
}

func (s *SqliteStore) ListAgentLogs(
	ctx context.Context,
	p ListAgentLogsParams,
) ([]AgentLog, error) {
	rows, err := s.q.ListAgentLogs(ctx, sqlitedb.ListAgentLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteAgentLogSlice(rows), nil
}

func (s *SqliteStore) DeleteAgentLogs(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentLogs(ctx, sql.NullInt64{Int64: int64(agentID), Valid: true})
}

func (s *SqliteStore) ListTaskTeams(
	ctx context.Context,
) ([]ListTaskTeamsRow, error) {
	rows, err := s.q.ListTaskTeams(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskTeamsRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskTeamsByTask(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskTeamsByTask(ctx, taskID)
}

func (s *SqliteStore) DeleteTaskTeams(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskTeams(ctx, taskID)
}

func (s *SqliteStore) AddTaskTeam(ctx context.Context, p AddTaskTeamParams) error {
	return s.q.AddTaskTeam(ctx, sqlitedb.AddTaskTeamParams(p))
}

func (s *SqliteStore) AddTaskTeamPrompt(
	ctx context.Context,
	p AddTaskTeamPromptParams,
) error {
	return s.q.AddTaskTeamPrompt(ctx, sqlitedb.AddTaskTeamPromptParams(p))
}

func (s *SqliteStore) DeleteTaskTeamPrompts(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskTeamPrompts(ctx, taskID)
}

func (s *SqliteStore) ListTaskTeamPromptsByTask(
	ctx context.Context,
	taskID string,
) ([]TaskTeamPrompt, error) {
	rows, err := s.q.ListTaskTeamPromptsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]TaskTeamPrompt, len(rows))
	for i, row := range rows {
		out[i] = TaskTeamPrompt{
			TeamID:          row.TeamID,
			PromptID:        row.PromptID,
			ProjectPromptID: row.ProjectPromptID,
		}
	}
	return out, nil
}

func (s *SqliteStore) DeleteTaskTeamsByProjectTeam(
	ctx context.Context,
	p DeleteTaskTeamsByProjectTeamParams,
) error {
	return s.q.DeleteTaskTeamsByProjectTeam(ctx, sqlitedb.DeleteTaskTeamsByProjectTeamParams(p))
}

func (s *SqliteStore) IsActorInTaskTeam(
	ctx context.Context,
	p IsActorInTaskTeamParams,
) (bool, error) {
	return s.q.IsActorInTaskTeam(ctx, sqlitedb.IsActorInTaskTeamParams(p))
}

func (s *SqliteStore) ListTaskTeamMemberActorIDs(
	ctx context.Context,
	taskID string,
) ([]int64, error) {
	return s.q.ListTaskTeamMemberActorIDs(ctx, taskID)
}

func (s *SqliteStore) CountTasksOwnedByTeam(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountTasksOwnedByTeam(ctx, teamID)
}

func (s *SqliteStore) ListActorProjectTeams(
	ctx context.Context,
	p ListActorProjectTeamsParams,
) ([]int64, error) {
	return s.q.ListActorProjectTeams(ctx, sqlitedb.ListActorProjectTeamsParams(p))
}

func (s *SqliteStore) ListTaskLeads(
	ctx context.Context,
) ([]ListTaskLeadsRow, error) {
	rows, err := s.q.ListTaskLeads(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskLeadsRowSlice(rows), nil
}

func (s *SqliteStore) ListTaskLeadsByTask(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskLeadsByTask(ctx, taskID)
}

func (s *SqliteStore) DeleteTaskLeads(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskLeads(ctx, taskID)
}

func (s *SqliteStore) AddTaskLead(ctx context.Context, p AddTaskLeadParams) error {
	return s.q.AddTaskLead(ctx, sqlitedb.AddTaskLeadParams(p))
}

func (s *SqliteStore) IsActorTaskLead(
	ctx context.Context,
	p IsActorTaskLeadParams,
) (bool, error) {
	return s.q.IsActorTaskLead(ctx, sqlitedb.IsActorTaskLeadParams(p))
}

func (s *SqliteStore) AddTaskSubscription(
	ctx context.Context,
	p AddTaskSubscriptionParams,
) error {
	return s.q.AddTaskSubscription(ctx, sqlitedb.AddTaskSubscriptionParams(p))
}

func (s *SqliteStore) DeleteTaskSubscription(
	ctx context.Context,
	p DeleteTaskSubscriptionParams,
) error {
	return s.q.DeleteTaskSubscription(ctx, sqlitedb.DeleteTaskSubscriptionParams(p))
}

func (s *SqliteStore) IsActorSubscribedToTask(
	ctx context.Context,
	p IsActorSubscribedToTaskParams,
) (bool, error) {
	return s.q.IsActorSubscribedToTask(ctx, sqlitedb.IsActorSubscribedToTaskParams(p))
}

func (s *SqliteStore) ListTaskSubscriptionsByTask(
	ctx context.Context,
	taskID string,
) ([]int64, error) {
	return s.q.ListTaskSubscriptionsByTask(ctx, taskID)
}

func (s *SqliteStore) ListTaskNotificationRecipients(
	ctx context.Context,
	taskID string,
) ([]int64, error) {
	return s.q.ListTaskNotificationRecipients(ctx, taskID)
}

func (s *SqliteStore) GetTeam(ctx context.Context, id int64) (Team, error) {
	row, err := s.q.GetTeam(ctx, id)
	if err != nil {
		return Team{}, err
	}
	return fromSqliteTeam(row), nil
}

func (s *SqliteStore) GetTeamByName(ctx context.Context, name string) (Team, error) {
	row, err := s.q.GetTeamByName(ctx, name)
	if err != nil {
		return Team{}, err
	}
	return fromSqliteTeam(row), nil
}

func (s *SqliteStore) ListTeams(ctx context.Context) ([]ListTeamsRow, error) {
	rows, err := s.q.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamsRowSlice(rows), nil
}

func (s *SqliteStore) ListTeamsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListTeamsForActorRow, error) {
	rows, err := s.q.ListTeamsForActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamsForActorRowSlice(rows), nil
}

func (s *SqliteStore) UpdateTeam(
	ctx context.Context,
	p UpdateTeamParams,
) (Team, error) {
	row, err := s.q.UpdateTeam(ctx, sqlitedb.UpdateTeamParams(p))
	if err != nil {
		return Team{}, err
	}
	return fromSqliteTeam(row), nil
}

func (s *SqliteStore) SetTeamPicture(
	ctx context.Context,
	p SetTeamPictureParams,
) error {
	return s.q.SetTeamPicture(ctx, sqlitedb.SetTeamPictureParams(p))
}

func (s *SqliteStore) ClearTeamPicture(
	ctx context.Context,
	p ClearTeamPictureParams,
) error {
	return s.q.ClearTeamPicture(ctx, sqlitedb.ClearTeamPictureParams(p))
}

func (s *SqliteStore) CountTeams(ctx context.Context) (int64, error) {
	return s.q.CountTeams(ctx)
}

func (s *SqliteStore) AddTeamMember(
	ctx context.Context,
	p AddTeamMemberParams,
) error {
	return s.q.AddTeamMember(ctx, sqlitedb.AddTeamMemberParams(p))
}

func (s *SqliteStore) RemoveTeamMember(
	ctx context.Context,
	p RemoveTeamMemberParams,
) (int64, error) {
	return s.q.RemoveTeamMember(ctx, sqlitedb.RemoveTeamMemberParams(p))
}

func (s *SqliteStore) GetTeamMember(
	ctx context.Context,
	p GetTeamMemberParams,
) (TeamMember, error) {
	row, err := s.q.GetTeamMember(ctx, sqlitedb.GetTeamMemberParams(p))
	if err != nil {
		return TeamMember{}, err
	}
	return fromSqliteTeamMember(row), nil
}

func (s *SqliteStore) ListTeamMembers(
	ctx context.Context,
	teamID int64,
) ([]ListTeamMembersRow, error) {
	rows, err := s.q.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamMembersRowSlice(rows), nil
}

func (s *SqliteStore) CountTeamMembers(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountTeamMembers(ctx, teamID)
}

func (s *SqliteStore) CountTeamOwners(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountTeamOwners(ctx, teamID)
}

func (s *SqliteStore) CreateTeamInvitation(
	ctx context.Context,
	p CreateTeamInvitationParams,
) (TeamInvitation, error) {
	row, err := s.q.CreateTeamInvitation(ctx, sqlitedb.CreateTeamInvitationParams(p))
	if err != nil {
		return TeamInvitation{}, err
	}
	return fromSqliteTeamInvitation(row), nil
}

func (s *SqliteStore) GetTeamInvitation(
	ctx context.Context,
	id int64,
) (TeamInvitation, error) {
	row, err := s.q.GetTeamInvitation(ctx, id)
	if err != nil {
		return TeamInvitation{}, err
	}
	return fromSqliteTeamInvitation(row), nil
}

func (s *SqliteStore) GetPendingTeamInvitation(
	ctx context.Context,
	p GetPendingTeamInvitationParams,
) (TeamInvitation, error) {
	row, err := s.q.GetPendingTeamInvitation(ctx, sqlitedb.GetPendingTeamInvitationParams(p))
	if err != nil {
		return TeamInvitation{}, err
	}
	return fromSqliteTeamInvitation(row), nil
}

func (s *SqliteStore) ListTeamInvitations(
	ctx context.Context,
	teamID int64,
) ([]ListTeamInvitationsRow, error) {
	rows, err := s.q.ListTeamInvitations(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamInvitationsRowSlice(rows), nil
}

func (s *SqliteStore) ListPendingTeamInvitationsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListPendingTeamInvitationsForActorRow, error) {
	rows, err := s.q.ListPendingTeamInvitationsForActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListPendingTeamInvitationsForActorRowSlice(rows), nil
}

func (s *SqliteStore) RespondTeamInvitation(
	ctx context.Context,
	p RespondTeamInvitationParams,
) (int64, error) {
	return s.q.RespondTeamInvitation(ctx, sqlitedb.RespondTeamInvitationParams(p))
}

func (s *SqliteStore) RevokeTeamInvitation(
	ctx context.Context,
	p RevokeTeamInvitationParams,
) (int64, error) {
	return s.q.RevokeTeamInvitation(ctx, sqlitedb.RevokeTeamInvitationParams(p))
}

func (s *SqliteStore) ExpireTeamInvitations(ctx context.Context, now int64) (int64, error) {
	return s.q.ExpireTeamInvitations(
		ctx,
		sqlitedb.ExpireTeamInvitationsParams(ExpireTeamInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now, Valid: true},
			Now:         now,
		}),
	)
}

func (s *SqliteStore) AddAgentOwner(
	ctx context.Context,
	p AddAgentOwnerParams,
) error {
	return s.q.AddAgentOwner(ctx, sqlitedb.AddAgentOwnerParams(p))
}

func (s *SqliteStore) RemoveAgentOwner(
	ctx context.Context,
	p RemoveAgentOwnerParams,
) (int64, error) {
	return s.q.RemoveAgentOwner(ctx, sqlitedb.RemoveAgentOwnerParams(p))
}

func (s *SqliteStore) CountAgentOwners(
	ctx context.Context,
	agentActorID ids.ActorID,
) (int64, error) {
	return s.q.CountAgentOwners(ctx, int64(agentActorID))
}

func (s *SqliteStore) ListAgentOwners(
	ctx context.Context,
	agentActorID ids.ActorID,
) ([]ListAgentOwnersRow, error) {
	rows, err := s.q.ListAgentOwners(ctx, int64(agentActorID))
	if err != nil {
		return nil, err
	}
	return fromSqliteListAgentOwnersRowSlice(rows), nil
}

func (s *SqliteStore) ListAgentsOwnedBy(
	ctx context.Context,
	ownerActorID ids.ActorID,
) ([]int64, error) {
	return s.q.ListAgentsOwnedBy(ctx, int64(ownerActorID))
}

func (s *SqliteStore) DeleteAgentOwners(ctx context.Context, agentActorID ids.ActorID) error {
	return s.q.DeleteAgentOwners(ctx, int64(agentActorID))
}

func (s *SqliteStore) InsertTeamLog(ctx context.Context, p InsertTeamLogParams) error {
	return s.q.InsertTeamLog(ctx, sqlitedb.InsertTeamLogParams(p))
}

func (s *SqliteStore) ListTeamLogs(
	ctx context.Context,
	p ListTeamLogsParams,
) ([]ListTeamLogsRow, error) {
	rows, err := s.q.ListTeamLogs(ctx, sqlitedb.ListTeamLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromSqliteListTeamLogsRowSlice(rows), nil
}
