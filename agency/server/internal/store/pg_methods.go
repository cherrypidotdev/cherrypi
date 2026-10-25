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

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

func (s *PgStore) GetActorByID(ctx context.Context, id ids.ActorID) (Actor, error) {
	row, err := s.q.GetActorByID(ctx, int64(id))
	if err != nil {
		return Actor{}, err
	}
	return fromPgActor(row), nil
}

func (s *PgStore) GetActorByUsername(
	ctx context.Context,
	username string,
) (Actor, error) {
	row, err := s.q.GetActorByUsername(ctx, username)
	if err != nil {
		return Actor{}, err
	}
	return fromPgActor(row), nil
}

func (s *PgStore) GetActorByEmail(ctx context.Context, email string) (Actor, error) {
	row, err := s.q.GetActorByEmail(ctx, email)
	if err != nil {
		return Actor{}, err
	}
	return fromPgActor(row), nil
}

func (s *PgStore) UpdateActorPassword(
	ctx context.Context,
	p UpdateActorPasswordParams,
) error {
	return s.q.UpdateActorPassword(ctx, pgdb.UpdateActorPasswordParams(p))
}

func (s *PgStore) UpdateActorTOTP(ctx context.Context, p UpdateActorTOTPParams) error {
	return s.q.UpdateActorTOTP(ctx, pgdb.UpdateActorTOTPParams(p))
}

func (s *PgStore) ClearActorTOTP(ctx context.Context, p ClearActorTOTPParams) error {
	return s.q.ClearActorTOTP(ctx, pgdb.ClearActorTOTPParams(p))
}

func (s *PgStore) UpdateActorProfile(
	ctx context.Context,
	p UpdateActorProfileParams,
) error {
	return s.q.UpdateActorProfile(ctx, pgdb.UpdateActorProfileParams(p))
}

func (s *PgStore) CountUsers(ctx context.Context) (int64, error) {
	return s.q.CountUsers(ctx)
}

func (s *PgStore) CountAdmins(ctx context.Context) (int64, error) {
	return s.q.CountAdmins(ctx)
}

func (s *PgStore) UpdateUserRole(ctx context.Context, p UpdateUserRoleParams) error {
	return s.q.UpdateUserRole(ctx, pgdb.UpdateUserRoleParams(p))
}

func (s *PgStore) UpdateUserStatus(ctx context.Context, p UpdateUserStatusParams) error {
	return s.q.UpdateUserStatus(ctx, pgdb.UpdateUserStatusParams(p))
}

func (s *PgStore) TouchUserLogin(ctx context.Context, p TouchUserLoginParams) error {
	return s.q.TouchUserLogin(ctx, pgdb.TouchUserLoginParams(p))
}

func (s *PgStore) SetUserMustChangePassword(
	ctx context.Context,
	p SetUserMustChangePasswordParams,
) error {
	return s.q.SetUserMustChangePassword(ctx, pgdb.SetUserMustChangePasswordParams(p))
}

func (s *PgStore) DeleteUserByActor(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteUserByActor(ctx, int64(actorID))
}

func (s *PgStore) CreateEmailCredential(
	ctx context.Context,
	p CreateEmailCredentialParams,
) (EmailCredential, error) {
	row, err := s.q.CreateEmailCredential(ctx, pgdb.CreateEmailCredentialParams(p))
	if err != nil {
		return EmailCredential{}, err
	}
	return fromPgEmailCredential(row), nil
}

func (s *PgStore) ListEmailCredentials(
	ctx context.Context,
	actorID ids.ActorID,
) ([]EmailCredential, error) {
	rows, err := s.q.ListEmailCredentials(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgEmailCredentialSlice(rows), nil
}

func (s *PgStore) GetEmailCredentialByEmail(
	ctx context.Context,
	email string,
) (EmailCredential, error) {
	row, err := s.q.GetEmailCredentialByEmail(ctx, email)
	if err != nil {
		return EmailCredential{}, err
	}
	return fromPgEmailCredential(row), nil
}

func (s *PgStore) GetEmailCredentialByID(
	ctx context.Context,
	id int64,
) (EmailCredential, error) {
	row, err := s.q.GetEmailCredentialByID(ctx, id)
	if err != nil {
		return EmailCredential{}, err
	}
	return fromPgEmailCredential(row), nil
}

func (s *PgStore) ConfirmEmailCredential(
	ctx context.Context,
	p ConfirmEmailCredentialParams,
) error {
	return s.q.ConfirmEmailCredential(ctx, pgdb.ConfirmEmailCredentialParams(p))
}

func (s *PgStore) DeleteEmailCredential(
	ctx context.Context,
	p DeleteEmailCredentialParams,
) error {
	return s.q.DeleteEmailCredential(ctx, pgdb.DeleteEmailCredentialParams(p))
}

func (s *PgStore) CountConfirmedEmails(ctx context.Context, actorID ids.ActorID) (int64, error) {
	return s.q.CountConfirmedEmails(ctx, int64(actorID))
}

func (s *PgStore) CreateSession(
	ctx context.Context,
	p CreateSessionParams,
) (Session, error) {
	row, err := s.q.CreateSession(ctx, pgdb.CreateSessionParams(p))
	if err != nil {
		return Session{}, err
	}
	return fromPgSession(row), nil
}

func (s *PgStore) TouchSession(ctx context.Context, p TouchSessionParams) error {
	return s.q.TouchSession(ctx, pgdb.TouchSessionParams(p))
}

func (s *PgStore) DeleteSessionByHash(ctx context.Context, tokenHash []byte) error {
	return s.q.DeleteSessionByHash(ctx, tokenHash)
}

func (s *PgStore) DeleteSessionByID(
	ctx context.Context,
	p DeleteSessionByIDParams,
) error {
	return s.q.DeleteSessionByID(ctx, pgdb.DeleteSessionByIDParams(p))
}

func (s *PgStore) DeleteSessionsByActor(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteSessionsByActor(ctx, int64(actorID))
}

func (s *PgStore) DeleteOtherSessions(
	ctx context.Context,
	p DeleteOtherSessionsParams,
) error {
	return s.q.DeleteOtherSessions(ctx, pgdb.DeleteOtherSessionsParams(p))
}

func (s *PgStore) ListSessionsByActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Session, error) {
	rows, err := s.q.ListSessionsByActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgSessionSlice(rows), nil
}

func (s *PgStore) DeleteExpiredSessions(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredSessions(ctx, now)
}

func (s *PgStore) CreateToken(
	ctx context.Context,
	p CreateTokenParams,
) (Token, error) {
	row, err := s.q.CreateToken(ctx, pgdb.CreateTokenParams(p))
	if err != nil {
		return Token{}, err
	}
	return fromPgToken(row), nil
}

func (s *PgStore) GetTokenByHash(ctx context.Context, tokenHash []byte) (Token, error) {
	row, err := s.q.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		return Token{}, err
	}
	return fromPgToken(row), nil
}

func (s *PgStore) ConsumeToken(ctx context.Context, p ConsumeTokenParams) (int64, error) {
	return s.q.ConsumeToken(ctx, pgdb.ConsumeTokenParams(p))
}

func (s *PgStore) DeleteTokensByActorKind(
	ctx context.Context,
	p DeleteTokensByActorKindParams,
) error {
	return s.q.DeleteTokensByActorKind(ctx, pgdb.DeleteTokensByActorKindParams(p))
}

func (s *PgStore) DeleteExpiredTokens(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredTokens(ctx, now)
}

func (s *PgStore) CreatePersonalAccessToken(
	ctx context.Context,
	p CreatePersonalAccessTokenParams,
) (PersonalAccessToken, error) {
	row, err := s.q.CreatePersonalAccessToken(ctx, pgdb.CreatePersonalAccessTokenParams(p))
	if err != nil {
		return PersonalAccessToken{}, err
	}
	return fromPgPersonalAccessToken(row), nil
}

func (s *PgStore) ListPersonalAccessTokens(
	ctx context.Context,
	actorID ids.ActorID,
) ([]PersonalAccessToken, error) {
	rows, err := s.q.ListPersonalAccessTokens(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgPersonalAccessTokenSlice(rows), nil
}

func (s *PgStore) TouchPersonalAccessToken(
	ctx context.Context,
	p TouchPersonalAccessTokenParams,
) error {
	return s.q.TouchPersonalAccessToken(ctx, pgdb.TouchPersonalAccessTokenParams(p))
}

func (s *PgStore) DeletePersonalAccessToken(
	ctx context.Context,
	p DeletePersonalAccessTokenParams,
) error {
	return s.q.DeletePersonalAccessToken(ctx, pgdb.DeletePersonalAccessTokenParams(p))
}

func (s *PgStore) DeleteExpiredPersonalAccessTokens(
	ctx context.Context,
	now sql.NullInt64,
) (int64, error) {
	return s.q.DeleteExpiredPersonalAccessTokens(ctx, now)
}

func (s *PgStore) CreateRecoveryCode(
	ctx context.Context,
	p CreateRecoveryCodeParams,
) (TotpRecoveryCode, error) {
	row, err := s.q.CreateRecoveryCode(ctx, pgdb.CreateRecoveryCodeParams(p))
	if err != nil {
		return TotpRecoveryCode{}, err
	}
	return fromPgTotpRecoveryCode(row), nil
}

func (s *PgStore) ListUnusedRecoveryCodes(
	ctx context.Context,
	actorID ids.ActorID,
) ([]TotpRecoveryCode, error) {
	rows, err := s.q.ListUnusedRecoveryCodes(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgTotpRecoveryCodeSlice(rows), nil
}

func (s *PgStore) ConsumeRecoveryCode(
	ctx context.Context,
	p ConsumeRecoveryCodeParams,
) (int64, error) {
	return s.q.ConsumeRecoveryCode(ctx, pgdb.ConsumeRecoveryCodeParams(p))
}

func (s *PgStore) DeleteRecoveryCodes(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteRecoveryCodes(ctx, int64(actorID))
}

func (s *PgStore) CountUnusedRecoveryCodes(
	ctx context.Context,
	actorID ids.ActorID,
) (int64, error) {
	return s.q.CountUnusedRecoveryCodes(ctx, int64(actorID))
}

func (s *PgStore) GetActorSetting(
	ctx context.Context,
	actorID ids.ActorID,
	key string,
) (string, error) {
	return s.q.GetActorSetting(ctx, pgdb.GetActorSettingParams(GetActorSettingParams{
		ActorID: int64(actorID),
		Key:     key,
	}))
}

func (s *PgStore) ListActorSettings(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListActorSettingsRow, error) {
	rows, err := s.q.ListActorSettings(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgListActorSettingsRowSlice(rows), nil
}

func (s *PgStore) UpsertActorSetting(
	ctx context.Context,
	p UpsertActorSettingParams,
) error {
	return s.q.UpsertActorSetting(ctx, pgdb.UpsertActorSettingParams(p))
}

func (s *PgStore) DeleteActorSetting(ctx context.Context, actorID ids.ActorID, key string) error {
	return s.q.DeleteActorSetting(ctx, pgdb.DeleteActorSettingParams(DeleteActorSettingParams{
		ActorID: int64(actorID),
		Key:     key,
	}))
}

func (s *PgStore) InsertUserLog(ctx context.Context, p InsertUserLogParams) error {
	return s.q.InsertUserLog(ctx, pgdb.InsertUserLogParams(p))
}

func (s *PgStore) ListUserLogs(
	ctx context.Context,
	p ListUserLogsParams,
) ([]ListUserLogsRow, error) {
	rows, err := s.q.ListUserLogs(ctx, pgdb.ListUserLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgListUserLogsRowSlice(rows), nil
}

func (s *PgStore) DeleteUserLogs(ctx context.Context, actorID int64) error {
	return s.q.DeleteUserLogs(ctx, actorID)
}

func (s *PgStore) PruneUserLogs(ctx context.Context, beforeAt int64) error {
	return s.q.PruneUserLogs(ctx, beforeAt)
}

func (s *PgStore) InsertAgentDomainLog(
	ctx context.Context,
	p InsertAgentDomainLogParams,
) error {
	return s.q.InsertAgentDomainLog(ctx, pgdb.InsertAgentDomainLogParams(p))
}

func (s *PgStore) ListAgentDomainLogs(
	ctx context.Context,
	p ListAgentDomainLogsParams,
) ([]ListAgentDomainLogsRow, error) {
	rows, err := s.q.ListAgentDomainLogs(ctx, pgdb.ListAgentDomainLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgListAgentDomainLogsRowSlice(rows), nil
}

func (s *PgStore) DeleteAgentDomainLogs(ctx context.Context, actorID int64) error {
	return s.q.DeleteAgentDomainLogs(ctx, actorID)
}

func (s *PgStore) PruneAgentDomainLogs(ctx context.Context, beforeAt int64) error {
	return s.q.PruneAgentDomainLogs(ctx, beforeAt)
}

func (s *PgStore) ListDomainLogs(
	ctx context.Context,
	p ListDomainLogsParams,
) ([]ListDomainLogsRow, error) {
	rows, err := s.q.ListDomainLogs(ctx, pgdb.ListDomainLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgListDomainLogsRowSlice(rows), nil
}

func (s *PgStore) CreateIdentityProvider(
	ctx context.Context,
	p CreateIdentityProviderParams,
) (IdentityProvider, error) {
	row, err := s.q.CreateIdentityProvider(ctx, pgdb.CreateIdentityProviderParams(p))
	if err != nil {
		return IdentityProvider{}, err
	}
	return fromPgIdentityProvider(row), nil
}

func (s *PgStore) GetIdentityProviderByID(
	ctx context.Context,
	id int64,
) (IdentityProvider, error) {
	row, err := s.q.GetIdentityProviderByID(ctx, id)
	if err != nil {
		return IdentityProvider{}, err
	}
	return fromPgIdentityProvider(row), nil
}

func (s *PgStore) GetIdentityProviderBySlug(
	ctx context.Context,
	slug string,
) (IdentityProvider, error) {
	row, err := s.q.GetIdentityProviderBySlug(ctx, slug)
	if err != nil {
		return IdentityProvider{}, err
	}
	return fromPgIdentityProvider(row), nil
}

func (s *PgStore) ListIdentityProviders(ctx context.Context) ([]IdentityProvider, error) {
	rows, err := s.q.ListIdentityProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgIdentityProviderSlice(rows), nil
}

func (s *PgStore) ListEnabledIdentityProviders(
	ctx context.Context,
) ([]IdentityProvider, error) {
	rows, err := s.q.ListEnabledIdentityProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgIdentityProviderSlice(rows), nil
}

func (s *PgStore) UpdateIdentityProvider(
	ctx context.Context,
	p UpdateIdentityProviderParams,
) error {
	return s.q.UpdateIdentityProvider(ctx, pgdb.UpdateIdentityProviderParams(p))
}

func (s *PgStore) CreateExternalCredential(
	ctx context.Context,
	p CreateExternalCredentialParams,
) (ExternalCredential, error) {
	row, err := s.q.CreateExternalCredential(ctx, pgdb.CreateExternalCredentialParams(p))
	if err != nil {
		return ExternalCredential{}, err
	}
	return fromPgExternalCredential(row), nil
}

func (s *PgStore) GetExternalCredential(
	ctx context.Context,
	providerID int64,
	subject string,
) (ExternalCredential, error) {
	row, err := s.q.GetExternalCredential(
		ctx,
		pgdb.GetExternalCredentialParams(GetExternalCredentialParams{
			ProviderID: providerID,
			Subject:    subject,
		}),
	)
	if err != nil {
		return ExternalCredential{}, err
	}
	return fromPgExternalCredential(row), nil
}

func (s *PgStore) ListExternalCredentialsByActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ExternalCredential, error) {
	rows, err := s.q.ListExternalCredentialsByActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgExternalCredentialSlice(rows), nil
}

func (s *PgStore) DeleteExternalCredential(
	ctx context.Context,
	id int64,
	actorID ids.ActorID,
) error {
	return s.q.DeleteExternalCredential(
		ctx,
		pgdb.DeleteExternalCredentialParams(DeleteExternalCredentialParams{
			ID:      id,
			ActorID: int64(actorID),
		}),
	)
}

func (s *PgStore) CreateLDAPProvider(
	ctx context.Context,
	p CreateLDAPProviderParams,
) (LdapProvider, error) {
	row, err := s.q.CreateLDAPProvider(ctx, pgdb.CreateLDAPProviderParams(p))
	if err != nil {
		return LdapProvider{}, err
	}
	return fromPgLdapProvider(row), nil
}

func (s *PgStore) GetLDAPProviderByID(
	ctx context.Context,
	id int64,
) (LdapProvider, error) {
	row, err := s.q.GetLDAPProviderByID(ctx, id)
	if err != nil {
		return LdapProvider{}, err
	}
	return fromPgLdapProvider(row), nil
}

func (s *PgStore) GetLDAPProviderBySlug(
	ctx context.Context,
	slug string,
) (LdapProvider, error) {
	row, err := s.q.GetLDAPProviderBySlug(ctx, slug)
	if err != nil {
		return LdapProvider{}, err
	}
	return fromPgLdapProvider(row), nil
}

func (s *PgStore) ListLDAPProviders(ctx context.Context) ([]LdapProvider, error) {
	rows, err := s.q.ListLDAPProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgLdapProviderSlice(rows), nil
}

func (s *PgStore) ListEnabledLDAPProviders(ctx context.Context) ([]LdapProvider, error) {
	rows, err := s.q.ListEnabledLDAPProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgLdapProviderSlice(rows), nil
}

func (s *PgStore) UpdateLDAPProvider(
	ctx context.Context,
	p UpdateLDAPProviderParams,
) error {
	return s.q.UpdateLDAPProvider(ctx, pgdb.UpdateLDAPProviderParams(p))
}

func (s *PgStore) CreateLDAPCredential(
	ctx context.Context,
	p CreateLDAPCredentialParams,
) (LdapCredential, error) {
	row, err := s.q.CreateLDAPCredential(ctx, pgdb.CreateLDAPCredentialParams(p))
	if err != nil {
		return LdapCredential{}, err
	}
	return fromPgLdapCredential(row), nil
}

func (s *PgStore) GetLDAPCredential(
	ctx context.Context,
	providerID int64,
	username string,
) (LdapCredential, error) {
	row, err := s.q.GetLDAPCredential(ctx, pgdb.GetLDAPCredentialParams(GetLDAPCredentialParams{
		ProviderID: providerID,
		Username:   username,
	}))
	if err != nil {
		return LdapCredential{}, err
	}
	return fromPgLdapCredential(row), nil
}

func (s *PgStore) ListLDAPCredentialsByActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]LdapCredential, error) {
	rows, err := s.q.ListLDAPCredentialsByActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgLdapCredentialSlice(rows), nil
}

func (s *PgStore) DeleteLDAPCredential(
	ctx context.Context,
	id int64,
	actorID ids.ActorID,
) error {
	return s.q.DeleteLDAPCredential(ctx, pgdb.DeleteLDAPCredentialParams(DeleteLDAPCredentialParams{
		ID:      id,
		ActorID: int64(actorID),
	}))
}

func (s *PgStore) AcceptInvitation(
	ctx context.Context,
	p AcceptInvitationParams,
) (int64, error) {
	return s.q.AcceptInvitation(ctx, pgdb.AcceptInvitationParams(p))
}

func (s *PgStore) UpdateInvitationTokenHash(
	ctx context.Context,
	p UpdateInvitationTokenHashParams,
) error {
	return s.q.UpdateInvitationTokenHash(ctx, pgdb.UpdateInvitationTokenHashParams(p))
}

func (s *PgStore) RevokeInvitation(
	ctx context.Context,
	p RevokeInvitationParams,
) (int64, error) {
	return s.q.RevokeInvitation(ctx, pgdb.RevokeInvitationParams(p))
}

func (s *PgStore) DeleteInvitationsByActor(ctx context.Context, actorID ids.ActorID) error {
	return s.q.DeleteInvitationsByActor(ctx, int64(actorID))
}

func (s *PgStore) DeleteExpiredInvitedActors(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredInvitedActors(ctx, now)
}

func (s *PgStore) DeleteExpiredInvitations(ctx context.Context, now int64) (int64, error) {
	return s.q.DeleteExpiredInvitations(ctx, now)
}

func (s *PgStore) UpsertProjectActorGrant(
	ctx context.Context,
	p UpsertProjectActorGrantParams,
) error {
	return s.q.UpsertProjectActorGrant(ctx, pgdb.UpsertProjectActorGrantParams(p))
}

func (s *PgStore) UpdateProjectActorGrantRole(
	ctx context.Context,
	p UpdateProjectActorGrantRoleParams,
) error {
	return s.q.UpdateProjectActorGrantRole(ctx, pgdb.UpdateProjectActorGrantRoleParams(p))
}

func (s *PgStore) RemoveProjectActorGrant(
	ctx context.Context,
	p RemoveProjectActorGrantParams,
) error {
	return s.q.RemoveProjectActorGrant(ctx, pgdb.RemoveProjectActorGrantParams(p))
}

func (s *PgStore) GetProjectActorGrant(
	ctx context.Context,
	p GetProjectActorGrantParams,
) (ProjectActorMembership, error) {
	row, err := s.q.GetProjectActorGrant(ctx, pgdb.GetProjectActorGrantParams(p))
	if err != nil {
		return ProjectActorMembership{}, err
	}
	return fromPgProjectActorMembership(row), nil
}

func (s *PgStore) ListProjectActorGrants(
	ctx context.Context,
	projectID string,
) ([]ListProjectActorGrantsRow, error) {
	rows, err := s.q.ListProjectActorGrants(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectActorGrantsRowSlice(rows), nil
}

func (s *PgStore) UpsertProjectTeamMembership(
	ctx context.Context,
	p UpsertProjectTeamMembershipParams,
) error {
	return s.q.UpsertProjectTeamMembership(ctx, pgdb.UpsertProjectTeamMembershipParams(p))
}

func (s *PgStore) UpdateProjectTeamMembershipRole(
	ctx context.Context,
	p UpdateProjectTeamMembershipRoleParams,
) error {
	return s.q.UpdateProjectTeamMembershipRole(ctx, pgdb.UpdateProjectTeamMembershipRoleParams(p))
}

func (s *PgStore) RemoveProjectTeamMembership(
	ctx context.Context,
	p RemoveProjectTeamMembershipParams,
) error {
	return s.q.RemoveProjectTeamMembership(ctx, pgdb.RemoveProjectTeamMembershipParams(p))
}

func (s *PgStore) GetProjectTeamMembership(
	ctx context.Context,
	p GetProjectTeamMembershipParams,
) (ProjectTeamMembership, error) {
	row, err := s.q.GetProjectTeamMembership(ctx, pgdb.GetProjectTeamMembershipParams(p))
	if err != nil {
		return ProjectTeamMembership{}, err
	}
	return fromPgProjectTeamMembership(row), nil
}

func (s *PgStore) ListProjectTeamMemberships(
	ctx context.Context,
	projectID string,
) ([]ListProjectTeamMembershipsRow, error) {
	rows, err := s.q.ListProjectTeamMemberships(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectTeamMembershipsRowSlice(rows), nil
}

func (s *PgStore) ListProjectTeamMembershipsForTeam(
	ctx context.Context,
	teamID int64,
) ([]ListProjectTeamMembershipsForTeamRow, error) {
	rows, err := s.q.ListProjectTeamMembershipsForTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectTeamMembershipsForTeamRowSlice(rows), nil
}

func (s *PgStore) CountProjectsOwnedByTeam(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountProjectsOwnedByTeam(ctx, teamID)
}

func (s *PgStore) CreateProjectTeamInvitation(
	ctx context.Context,
	p CreateProjectTeamInvitationParams,
) (ProjectTeamInvitation, error) {
	row, err := s.q.CreateProjectTeamInvitation(ctx, pgdb.CreateProjectTeamInvitationParams(p))
	if err != nil {
		return ProjectTeamInvitation{}, err
	}
	return fromPgProjectTeamInvitation(row), nil
}

func (s *PgStore) GetProjectTeamInvitation(
	ctx context.Context,
	id int64,
) (ProjectTeamInvitation, error) {
	row, err := s.q.GetProjectTeamInvitation(ctx, id)
	if err != nil {
		return ProjectTeamInvitation{}, err
	}
	return fromPgProjectTeamInvitation(row), nil
}

func (s *PgStore) GetPendingProjectTeamInvitation(
	ctx context.Context,
	p GetPendingProjectTeamInvitationParams,
) (ProjectTeamInvitation, error) {
	row, err := s.q.GetPendingProjectTeamInvitation(
		ctx,
		pgdb.GetPendingProjectTeamInvitationParams(p),
	)
	if err != nil {
		return ProjectTeamInvitation{}, err
	}
	return fromPgProjectTeamInvitation(row), nil
}

func (s *PgStore) ListProjectTeamInvitations(
	ctx context.Context,
	projectID string,
) ([]ListProjectTeamInvitationsRow, error) {
	rows, err := s.q.ListProjectTeamInvitations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectTeamInvitationsRowSlice(rows), nil
}

func (s *PgStore) ListPendingProjectInvitationsForTeamOwner(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListPendingProjectInvitationsForTeamOwnerRow, error) {
	rows, err := s.q.ListPendingProjectInvitationsForTeamOwner(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgListPendingProjectInvitationsForTeamOwnerRowSlice(rows), nil
}

func (s *PgStore) RespondProjectTeamInvitation(
	ctx context.Context,
	p RespondProjectTeamInvitationParams,
) (int64, error) {
	return s.q.RespondProjectTeamInvitation(ctx, pgdb.RespondProjectTeamInvitationParams(p))
}

func (s *PgStore) RevokeProjectTeamInvitation(
	ctx context.Context,
	p RevokeProjectTeamInvitationParams,
) (int64, error) {
	return s.q.RevokeProjectTeamInvitation(ctx, pgdb.RevokeProjectTeamInvitationParams(p))
}

func (s *PgStore) ExpireProjectTeamInvitations(ctx context.Context, now int64) (int64, error) {
	return s.q.ExpireProjectTeamInvitations(
		ctx,
		pgdb.ExpireProjectTeamInvitationsParams(ExpireProjectTeamInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now, Valid: true},
			Now:         now,
		}),
	)
}

func (s *PgStore) ListProjectsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Project, error) {
	rows, err := s.q.ListProjectsForActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgProjectSlice(rows), nil
}

func (s *PgStore) ListAgentProjectMemberships(
	ctx context.Context,
	agentID ids.AgentID,
) ([]ListAgentProjectMembershipsRow, error) {
	rows, err := s.q.ListAgentProjectMemberships(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgListAgentProjectMembershipsRowSlice(rows), nil
}

func (s *PgStore) ListProjectMemberActorIDs(
	ctx context.Context,
	projectID string,
) ([]int64, error) {
	return s.q.ListProjectMemberActorIDs(ctx, projectID)
}

func (s *PgStore) InsertProjectLog(ctx context.Context, p InsertProjectLogParams) error {
	return s.q.InsertProjectLog(ctx, pgdb.InsertProjectLogParams(p))
}

func (s *PgStore) ListProjectLogs(
	ctx context.Context,
	p ListProjectLogsParams,
) ([]ListProjectLogsRow, error) {
	rows, err := s.q.ListProjectLogs(ctx, pgdb.ListProjectLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgListProjectLogsRowSlice(rows), nil
}

func (s *PgStore) TouchAgentSeen(ctx context.Context, id ids.AgentID, at int64) error {
	return s.q.TouchAgentSeen(ctx, pgdb.TouchAgentSeenParams(TouchAgentSeenParams{
		LastSeenAt: sql.NullInt64{Int64: at, Valid: true},
		ID:         int64(id),
	}))
}

func (s *PgStore) CreateAgentCredential(
	ctx context.Context,
	p CreateAgentCredentialParams,
) (AgentCredential, error) {
	row, err := s.q.CreateAgentCredential(ctx, pgdb.CreateAgentCredentialParams(p))
	if err != nil {
		return AgentCredential{}, err
	}
	return fromPgAgentCredential(row), nil
}

func (s *PgStore) ListAgentCredentials(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentCredential, error) {
	rows, err := s.q.ListAgentCredentials(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgAgentCredentialSlice(rows), nil
}

func (s *PgStore) GetAgentCredential(
	ctx context.Context,
	username string,
) (AgentCredential, error) {
	row, err := s.q.GetAgentCredential(ctx, username)
	if err != nil {
		return AgentCredential{}, err
	}
	return fromPgAgentCredential(row), nil
}

func (s *PgStore) RevokeAgentCredential(
	ctx context.Context,
	p RevokeAgentCredentialParams,
) (int64, error) {
	return s.q.RevokeAgentCredential(ctx, pgdb.RevokeAgentCredentialParams(p))
}

func (s *PgStore) TouchAgentCredentialUsed(
	ctx context.Context,
	p TouchAgentCredentialUsedParams,
) error {
	return s.q.TouchAgentCredentialUsed(ctx, pgdb.TouchAgentCredentialUsedParams(p))
}

func (s *PgStore) ListSettings(ctx context.Context) ([]Setting, error) {
	rows, err := s.q.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgSettingSlice(rows), nil
}

func (s *PgStore) GetSetting(ctx context.Context, key string) (string, error) {
	return s.q.GetSetting(ctx, key)
}

func (s *PgStore) UpsertSetting(ctx context.Context, p UpsertSettingParams) error {
	return s.q.UpsertSetting(ctx, pgdb.UpsertSettingParams(p))
}

func (s *PgStore) ClearSettingValue(ctx context.Context, key, value string) error {
	return s.q.ClearSettingValue(ctx, pgdb.ClearSettingValueParams(ClearSettingValueParams{
		Key:   key,
		Value: value,
	}))
}

func (s *PgStore) CreateProject(
	ctx context.Context,
	p CreateProjectParams,
) (Project, error) {
	row, err := s.q.CreateProject(ctx, pgdb.CreateProjectParams(p))
	if err != nil {
		return Project{}, err
	}
	return fromPgProject(row), nil
}

func (s *PgStore) GetProject(ctx context.Context, id string) (Project, error) {
	row, err := s.q.GetProject(ctx, id)
	if err != nil {
		return Project{}, err
	}
	return fromPgProject(row), nil
}

func (s *PgStore) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	row, err := s.q.GetProjectBySlug(ctx, slug)
	if err != nil {
		return Project{}, err
	}
	return fromPgProject(row), nil
}

func (s *PgStore) GetProjectByPrefix(
	ctx context.Context,
	prefix string,
) (Project, error) {
	row, err := s.q.GetProjectByPrefix(ctx, prefix)
	if err != nil {
		return Project{}, err
	}
	return fromPgProject(row), nil
}

func (s *PgStore) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.q.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgProjectSlice(rows), nil
}

func (s *PgStore) SetProjectArchived(
	ctx context.Context,
	p SetProjectArchivedParams,
) (Project, error) {
	row, err := s.q.SetProjectArchived(ctx, pgdb.SetProjectArchivedParams(p))
	if err != nil {
		return Project{}, err
	}
	return fromPgProject(row), nil
}

func (s *PgStore) AllocateTaskNumber(ctx context.Context, projectID string) (int64, error) {
	return s.q.AllocateTaskNumber(ctx, projectID)
}

func (s *PgStore) ListAgentProjects(
	ctx context.Context,
	agentID ids.AgentID,
) ([]Project, error) {
	rows, err := s.q.ListAgentProjects(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgProjectSlice(rows), nil
}

func (s *PgStore) InsertPrompt(
	ctx context.Context,
	p InsertPromptParams,
) (Prompt, error) {
	row, err := s.q.InsertPrompt(ctx, pgdb.InsertPromptParams(p))
	if err != nil {
		return Prompt{}, err
	}
	return fromPgPrompt(row), nil
}

func (s *PgStore) GetPrompt(ctx context.Context, id int64) (Prompt, error) {
	row, err := s.q.GetPrompt(ctx, id)
	if err != nil {
		return Prompt{}, err
	}
	return fromPgPrompt(row), nil
}

func (s *PgStore) LatestPrompt(
	ctx context.Context,
	p LatestPromptParams,
) (Prompt, error) {
	row, err := s.q.LatestPrompt(ctx, pgdb.LatestPromptParams(p))
	if err != nil {
		return Prompt{}, err
	}
	return fromPgPrompt(row), nil
}

func (s *PgStore) LatestPromptVersion(
	ctx context.Context,
	p LatestPromptVersionParams,
) (int64, error) {
	return s.q.LatestPromptVersion(ctx, pgdb.LatestPromptVersionParams(p))
}

func (s *PgStore) CreateTask(
	ctx context.Context,
	p CreateTaskParams,
) (Task, error) {
	row, err := s.q.CreateTask(ctx, pgdb.CreateTaskParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) GetTask(ctx context.Context, id string) (Task, error) {
	row, err := s.q.GetTask(ctx, id)
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) GetTaskByProjectSeq(
	ctx context.Context,
	projectID string,
	seq int64,
) (Task, error) {
	row, err := s.q.GetTaskByProjectSeq(
		ctx,
		pgdb.GetTaskByProjectSeqParams(GetTaskByProjectSeqParams{
			ProjectID: sql.NullString{String: projectID, Valid: true},
			Seq:       seq,
		}),
	)
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) ListTasks(
	ctx context.Context,
	p ListTasksParams,
) ([]Task, error) {
	rows, err := s.q.ListTasks(ctx, pgdb.ListTasksParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) ListQueuedTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.q.ListQueuedTasks(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) ListRunningTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.q.ListRunningTasks(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) UpdateDraftTask(
	ctx context.Context,
	p UpdateDraftTaskParams,
) (Task, error) {
	row, err := s.q.UpdateDraftTask(ctx, pgdb.UpdateDraftTaskParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) ClearTaskPendingPrompt(
	ctx context.Context,
	p ClearTaskPendingPromptParams,
) (int64, error) {
	return s.q.ClearTaskPendingPrompt(ctx, pgdb.ClearTaskPendingPromptParams(p))
}

func (s *PgStore) SetTaskPrompt(
	ctx context.Context,
	p SetTaskPromptParams,
) (int64, error) {
	return s.q.SetTaskPrompt(ctx, pgdb.SetTaskPromptParams(p))
}

func (s *PgStore) SetTaskPendingPrompt(
	ctx context.Context,
	p SetTaskPendingPromptParams,
) (int64, error) {
	return s.q.SetTaskPendingPrompt(ctx, pgdb.SetTaskPendingPromptParams(p))
}

func (s *PgStore) TaskIDBySession(
	ctx context.Context,
	p TaskIDBySessionParams,
) (string, error) {
	return s.q.TaskIDBySession(ctx, pgdb.TaskIDBySessionParams(p))
}

func (s *PgStore) SetTaskError(ctx context.Context, p SetTaskErrorParams) (int64, error) {
	return s.q.SetTaskError(ctx, pgdb.SetTaskErrorParams(p))
}

func (s *PgStore) SetTaskSession(
	ctx context.Context,
	p SetTaskSessionParams,
) (int64, error) {
	return s.q.SetTaskSession(ctx, pgdb.SetTaskSessionParams(p))
}

func (s *PgStore) AppendTaskSessionEntry(
	ctx context.Context,
	p AppendTaskSessionEntryParams,
) (int64, error) {
	return s.q.AppendTaskSessionEntry(ctx, pgdb.AppendTaskSessionEntryParams(p))
}

func (s *PgStore) ListTaskSessionEntries(
	ctx context.Context,
	taskID string,
) ([]TaskSessionEntry, error) {
	rows, err := s.q.ListTaskSessionEntries(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgTaskSessionEntrySlice(rows), nil
}

func (s *PgStore) CountTaskSessionEntries(ctx context.Context, taskID string) (int64, error) {
	return s.q.CountTaskSessionEntries(ctx, taskID)
}

func (s *PgStore) ListTaskHistoryTaskIDs(ctx context.Context) ([]string, error) {
	return s.q.ListTaskHistoryTaskIDs(ctx)
}

func (s *PgStore) TaskSessionEntryTip(
	ctx context.Context,
	taskID string,
) (TaskSessionEntryTipRow, error) {
	row, err := s.q.TaskSessionEntryTip(ctx, taskID)
	if err != nil {
		return TaskSessionEntryTipRow{}, err
	}
	return fromPgTaskSessionEntryTipRow(row), nil
}

func (s *PgStore) RenameTask(
	ctx context.Context,
	p RenameTaskParams,
) (Task, error) {
	row, err := s.q.RenameTask(ctx, pgdb.RenameTaskParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) SetTaskPriority(
	ctx context.Context,
	p SetTaskPriorityParams,
) (Task, error) {
	row, err := s.q.SetTaskPriority(ctx, pgdb.SetTaskPriorityParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) SetTaskSize(
	ctx context.Context,
	p SetTaskSizeParams,
) (Task, error) {
	row, err := s.q.SetTaskSize(ctx, pgdb.SetTaskSizeParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) SetTaskAssignee(
	ctx context.Context,
	p SetTaskAssigneeParams,
) (Task, error) {
	row, err := s.q.SetTaskAssignee(ctx, pgdb.SetTaskAssigneeParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) MarkTaskSessionDeleted(
	ctx context.Context,
	p MarkTaskSessionDeletedParams,
) (Task, error) {
	row, err := s.q.MarkTaskSessionDeleted(ctx, pgdb.MarkTaskSessionDeletedParams(p))
	if err != nil {
		return Task{}, err
	}
	return fromPgTask(row), nil
}

func (s *PgStore) SetTaskNamePrompt(
	ctx context.Context,
	p SetTaskNamePromptParams,
) error {
	return s.q.SetTaskNamePrompt(ctx, pgdb.SetTaskNamePromptParams(p))
}

func (s *PgStore) ListTasksByProject(
	ctx context.Context,
	projectID sql.NullString,
) ([]Task, error) {
	rows, err := s.q.ListTasksByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) ListTasksByProjectAgent(
	ctx context.Context,
	p ListTasksByProjectAgentParams,
) ([]Task, error) {
	rows, err := s.q.ListTasksByProjectAgent(ctx, pgdb.ListTasksByProjectAgentParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) CountTasksByProject(
	ctx context.Context,
	projectID sql.NullString,
) (int64, error) {
	return s.q.CountTasksByProject(ctx, projectID)
}

func (s *PgStore) ListRunningTasksByAgent(
	ctx context.Context,
	agentID ids.AgentID,
) ([]Task, error) {
	rows, err := s.q.ListRunningTasksByAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) ListRunningTaskTagsByAgent(
	ctx context.Context,
	agentID ids.AgentID,
) ([]ListRunningTaskTagsByAgentRow, error) {
	rows, err := s.q.ListRunningTaskTagsByAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgListRunningTaskTagsByAgentRowSlice(rows), nil
}

func (s *PgStore) ListTaskLinks(ctx context.Context) ([]ListTaskLinksRow, error) {
	rows, err := s.q.ListTaskLinks(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskLinksRowSlice(rows), nil
}

func (s *PgStore) ListTaskLinksByTask(
	ctx context.Context,
	taskID string,
) ([]ListTaskLinksByTaskRow, error) {
	rows, err := s.q.ListTaskLinksByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskLinksByTaskRowSlice(rows), nil
}

func (s *PgStore) DeleteTaskLinks(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskLinks(ctx, taskID)
}

func (s *PgStore) InsertTaskLink(ctx context.Context, p InsertTaskLinkParams) error {
	return s.q.InsertTaskLink(ctx, pgdb.InsertTaskLinkParams(p))
}

func (s *PgStore) InsertTaskLog(ctx context.Context, p InsertTaskLogParams) error {
	return s.q.InsertTaskLog(ctx, pgdb.InsertTaskLogParams(p))
}

func (s *PgStore) ListTaskLogsByTask(
	ctx context.Context,
	taskID string,
) ([]ListTaskLogsByTaskRow, error) {
	rows, err := s.q.ListTaskLogsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskLogsByTaskRowSlice(rows), nil
}

func (s *PgStore) AddTaskReference(
	ctx context.Context,
	p UpsertTaskReferenceParams,
) error {
	return s.q.UpsertTaskReference(ctx, pgdb.UpsertTaskReferenceParams(p))
}

func (s *PgStore) RemoveTaskReference(
	ctx context.Context,
	p RemoveTaskReferenceParams,
) (int64, error) {
	return s.q.RemoveTaskReference(ctx, pgdb.RemoveTaskReferenceParams(p))
}

func (s *PgStore) ListTaskReferences(ctx context.Context, taskID string) ([]Task, error) {
	rows, err := s.q.ListTaskReferences(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) ListTaskReferencedBy(
	ctx context.Context,
	taskID string,
) ([]Task, error) {
	rows, err := s.q.ListTaskReferencedBy(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgTaskSlice(rows), nil
}

func (s *PgStore) ListTaskReferenceIDs(ctx context.Context, taskID string) ([]string, error) {
	return s.q.ListTaskReferenceIDs(ctx, taskID)
}

func (s *PgStore) CountIncompleteBlockingReferences(
	ctx context.Context,
	taskID string,
) (int64, error) {
	return s.q.CountIncompleteBlockingReferences(ctx, taskID)
}

func (s *PgStore) ListAllBlockingTaskReferences(
	ctx context.Context,
) ([]ListAllBlockingTaskReferencesRow, error) {
	rows, err := s.q.ListAllBlockingTaskReferences(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListAllBlockingTaskReferencesRowSlice(rows), nil
}

func (s *PgStore) ListTaskReferenceRefs(
	ctx context.Context,
) ([]ListTaskReferenceRefsRow, error) {
	rows, err := s.q.ListTaskReferenceRefs(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskReferenceRefsRowSlice(rows), nil
}

func (s *PgStore) ListTaskReferencedByRefs(
	ctx context.Context,
) ([]ListTaskReferencedByRefsRow, error) {
	rows, err := s.q.ListTaskReferencedByRefs(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskReferencedByRefsRowSlice(rows), nil
}

func (s *PgStore) CreateTag(
	ctx context.Context,
	p CreateTagParams,
) (Tag, error) {
	row, err := s.q.CreateTag(ctx, pgdb.CreateTagParams(p))
	if err != nil {
		return Tag{}, err
	}
	return fromPgTag(row), nil
}

func (s *PgStore) GetTag(ctx context.Context, id string) (Tag, error) {
	row, err := s.q.GetTag(ctx, id)
	if err != nil {
		return Tag{}, err
	}
	return fromPgTag(row), nil
}

func (s *PgStore) GetTagByName(
	ctx context.Context,
	projectID, name string,
) (Tag, error) {
	row, err := s.q.GetTagByName(ctx, pgdb.GetTagByNameParams(GetTagByNameParams{
		ProjectID: sql.NullString{String: projectID, Valid: projectID != ""},
		Name:      name,
	}))
	if err != nil {
		return Tag{}, err
	}
	return fromPgTag(row), nil
}

func (s *PgStore) GetTagWithCount(
	ctx context.Context,
	id string,
) (GetTagWithCountRow, error) {
	row, err := s.q.GetTagWithCount(ctx, id)
	if err != nil {
		return GetTagWithCountRow{}, err
	}
	return fromPgGetTagWithCountRow(row), nil
}

func (s *PgStore) UpdateTag(
	ctx context.Context,
	p UpdateTagParams,
) (Tag, error) {
	row, err := s.q.UpdateTag(ctx, pgdb.UpdateTagParams(p))
	if err != nil {
		return Tag{}, err
	}
	return fromPgTag(row), nil
}

func (s *PgStore) SetTagArchived(
	ctx context.Context,
	p SetTagArchivedParams,
) (Tag, error) {
	row, err := s.q.SetTagArchived(ctx, pgdb.SetTagArchivedParams(p))
	if err != nil {
		return Tag{}, err
	}
	return fromPgTag(row), nil
}

func (s *PgStore) DeleteTagsByProject(ctx context.Context, projectID string) error {
	return s.q.DeleteTagsByProject(ctx, sql.NullString{
		String: projectID,
		Valid:  projectID != "",
	})
}

func (s *PgStore) ListTaskTags(ctx context.Context) ([]ListTaskTagsRow, error) {
	rows, err := s.q.ListTaskTags(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskTagsRowSlice(rows), nil
}

func (s *PgStore) ListTaskTagsByTask(
	ctx context.Context,
	taskID string,
) ([]ListTaskTagsByTaskRow, error) {
	rows, err := s.q.ListTaskTagsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskTagsByTaskRowSlice(rows), nil
}

func (s *PgStore) DeleteTaskTags(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskTags(ctx, taskID)
}

func (s *PgStore) AddTaskTag(ctx context.Context, p AddTaskTagParams) error {
	return s.q.AddTaskTag(ctx, pgdb.AddTaskTagParams(p))
}

func (s *PgStore) DeleteTaskTag(ctx context.Context, p DeleteTaskTagParams) error {
	return s.q.DeleteTaskTag(ctx, pgdb.DeleteTaskTagParams(p))
}

func (s *PgStore) GetAgentTaskLimits(
	ctx context.Context,
	agentID ids.AgentID,
) (AgentTaskLimit, error) {
	row, err := s.q.GetAgentTaskLimits(ctx, int64(agentID))
	if err != nil {
		return AgentTaskLimit{}, err
	}
	return fromPgAgentTaskLimit(row), nil
}

func (s *PgStore) UpsertAgentTaskLimits(
	ctx context.Context,
	p UpsertAgentTaskLimitsParams,
) error {
	return s.q.UpsertAgentTaskLimits(ctx, pgdb.UpsertAgentTaskLimitsParams(p))
}

func (s *PgStore) ListAgentTagLimits(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentTagLimit, error) {
	rows, err := s.q.ListAgentTagLimits(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgAgentTagLimitSlice(rows), nil
}

func (s *PgStore) DeleteAgentTagLimits(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentTagLimits(ctx, int64(agentID))
}

func (s *PgStore) InsertAgentTagLimit(
	ctx context.Context,
	p InsertAgentTagLimitParams,
) error {
	return s.q.InsertAgentTagLimit(ctx, pgdb.InsertAgentTagLimitParams(p))
}

func (s *PgStore) ListAgentProjectLimits(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentProjectLimit, error) {
	rows, err := s.q.ListAgentProjectLimits(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgAgentProjectLimitSlice(rows), nil
}

func (s *PgStore) DeleteAgentProjectLimits(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentProjectLimits(ctx, int64(agentID))
}

func (s *PgStore) InsertAgentProjectLimit(
	ctx context.Context,
	p InsertAgentProjectLimitParams,
) error {
	return s.q.InsertAgentProjectLimit(ctx, pgdb.InsertAgentProjectLimitParams(p))
}

func (s *PgStore) ListAgentProjectTagLimits(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentProjectTagLimit, error) {
	rows, err := s.q.ListAgentProjectTagLimits(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	return fromPgAgentProjectTagLimitSlice(rows), nil
}

func (s *PgStore) DeleteAgentProjectTagLimits(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentProjectTagLimits(ctx, int64(agentID))
}

func (s *PgStore) InsertAgentProjectTagLimit(
	ctx context.Context,
	p InsertAgentProjectTagLimitParams,
) error {
	return s.q.InsertAgentProjectTagLimit(ctx, pgdb.InsertAgentProjectTagLimitParams(p))
}

func (s *PgStore) CreateWebhook(
	ctx context.Context,
	p CreateWebhookParams,
) (Webhook, error) {
	row, err := s.q.CreateWebhook(ctx, pgdb.CreateWebhookParams(p))
	if err != nil {
		return Webhook{}, err
	}
	return fromPgWebhook(row), nil
}

func (s *PgStore) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.q.ListWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgWebhookSlice(rows), nil
}

func (s *PgStore) GetWebhook(ctx context.Context, id int64) (Webhook, error) {
	row, err := s.q.GetWebhook(ctx, id)
	if err != nil {
		return Webhook{}, err
	}
	return fromPgWebhook(row), nil
}

func (s *PgStore) UpdateWebhook(
	ctx context.Context,
	p UpdateWebhookParams,
) (Webhook, error) {
	row, err := s.q.UpdateWebhook(ctx, pgdb.UpdateWebhookParams(p))
	if err != nil {
		return Webhook{}, err
	}
	return fromPgWebhook(row), nil
}

func (s *PgStore) DeleteWebhook(ctx context.Context, id int64) error {
	return s.q.DeleteWebhook(ctx, id)
}

func (s *PgStore) CreateNotification(
	ctx context.Context,
	p CreateNotificationParams,
) (Notification, error) {
	row, err := s.q.CreateNotification(ctx, pgdb.CreateNotificationParams(p))
	if err != nil {
		return Notification{}, err
	}
	return fromPgNotification(row), nil
}

func (s *PgStore) RefreshUnreadNotification(
	ctx context.Context,
	p RefreshUnreadNotificationParams,
) (Notification, error) {
	row, err := s.q.RefreshUnreadNotification(ctx, pgdb.RefreshUnreadNotificationParams(p))
	if err != nil {
		return Notification{}, err
	}
	return fromPgNotification(row), nil
}

func (s *PgStore) ListNotifications(
	ctx context.Context,
	recipient, limit int64,
) ([]ListNotificationsRow, error) {
	rows, err := s.q.ListNotifications(ctx, pgdb.ListNotificationsParams(ListNotificationsParams{
		RecipientActorID: nullActorID(recipient),
		MaxCount:         limit,
	}))
	if err != nil {
		return nil, err
	}
	return fromPgListNotificationsRowSlice(rows), nil
}

func (s *PgStore) CountUnreadNotifications(ctx context.Context, recipient int64) (int64, error) {
	return s.q.CountUnreadNotifications(ctx, nullActorID(recipient))
}

func (s *PgStore) MarkNotificationRead(
	ctx context.Context,
	p MarkNotificationReadParams,
) error {
	return s.q.MarkNotificationRead(ctx, pgdb.MarkNotificationReadParams(p))
}

func (s *PgStore) MarkTaskNotificationsRead(
	ctx context.Context,
	p MarkTaskNotificationsReadParams,
) error {
	return s.q.MarkTaskNotificationsRead(ctx, pgdb.MarkTaskNotificationsReadParams(p))
}

func (s *PgStore) MarkAllNotificationsRead(
	ctx context.Context,
	readAt sql.NullInt64,
	recipient int64,
) error {
	return s.q.MarkAllNotificationsRead(
		ctx,
		pgdb.MarkAllNotificationsReadParams(MarkAllNotificationsReadParams{
			ReadAt:           readAt,
			RecipientActorID: nullActorID(recipient),
		}),
	)
}

func (s *PgStore) DeleteNotification(ctx context.Context, id, recipient int64) error {
	return s.q.DeleteNotification(ctx, pgdb.DeleteNotificationParams(DeleteNotificationParams{
		ID:               id,
		RecipientActorID: nullActorID(recipient),
	}))
}

func (s *PgStore) DeleteAllNotifications(ctx context.Context, recipient int64) error {
	return s.q.DeleteAllNotifications(ctx, nullActorID(recipient))
}

func (s *PgStore) InsertAgentUsage(ctx context.Context, p InsertAgentUsageParams) error {
	return s.q.InsertAgentUsage(ctx, pgdb.InsertAgentUsageParams(p))
}

func (s *PgStore) ListAgentUsage(
	ctx context.Context,
	agentID sql.NullInt64,
	projectID sql.NullString,
) ([]ListAgentUsageRow, error) {
	rows, err := s.q.ListAgentUsage(ctx, pgdb.ListAgentUsageParams(ListAgentUsageParams{
		AgentID:   agentID,
		ProjectID: projectID,
	}))
	if err != nil {
		return nil, err
	}
	return fromPgListAgentUsageRowSlice(rows), nil
}

func (s *PgStore) SumAgentUsage(ctx context.Context) (SumAgentUsageRow, error) {
	row, err := s.q.SumAgentUsage(ctx)
	if err != nil {
		return SumAgentUsageRow{}, err
	}
	return fromPgSumAgentUsageRow(row), nil
}

func (s *PgStore) ListAgentUsageByDay(
	ctx context.Context,
	agentID sql.NullInt64,
	projectID sql.NullString,
) ([]ListAgentUsageByDayRow, error) {
	rows, err := s.q.ListAgentUsageByDay(
		ctx,
		pgdb.ListAgentUsageByDayParams(ListAgentUsageByDayParams{
			AgentID:   agentID,
			ProjectID: projectID,
		}),
	)
	if err != nil {
		return nil, err
	}
	return fromPgListAgentUsageByDayRowSlice(rows), nil
}

func (s *PgStore) UpdateAgentCredentialPassword(
	ctx context.Context,
	p UpdateAgentCredentialPasswordParams,
) error {
	return s.q.UpdateAgentCredentialPassword(ctx, pgdb.UpdateAgentCredentialPasswordParams(p))
}

func (s *PgStore) CreateAttachment(
	ctx context.Context,
	p CreateAttachmentParams,
) (Attachment, error) {
	row, err := s.q.CreateAttachment(ctx, pgdb.CreateAttachmentParams(p))
	if err != nil {
		return Attachment{}, err
	}
	return fromPgAttachment(row), nil
}

func (s *PgStore) GetAttachment(ctx context.Context, id int64) (Attachment, error) {
	row, err := s.q.GetAttachment(ctx, id)
	if err != nil {
		return Attachment{}, err
	}
	return fromPgAttachment(row), nil
}

func (s *PgStore) SetAttachmentSize(ctx context.Context, id, size int64) error {
	return s.q.SetAttachmentSize(ctx, pgdb.SetAttachmentSizeParams(SetAttachmentSizeParams{
		Size: size,
		ID:   id,
	}))
}

func (s *PgStore) ListOrphanAttachments(ctx context.Context) ([]Attachment, error) {
	rows, err := s.q.ListOrphanAttachments(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgAttachmentSlice(rows), nil
}

func (s *PgStore) LinkTaskAttachment(
	ctx context.Context,
	p LinkTaskAttachmentParams,
) error {
	return s.q.LinkTaskAttachment(ctx, pgdb.LinkTaskAttachmentParams(p))
}

func (s *PgStore) UnlinkTaskAttachment(
	ctx context.Context,
	taskID string,
	attachmentID int64,
) error {
	return s.q.UnlinkTaskAttachment(ctx, pgdb.UnlinkTaskAttachmentParams(UnlinkTaskAttachmentParams{
		TaskID:       taskID,
		AttachmentID: attachmentID,
	}))
}

func (s *PgStore) ListTaskAttachments(
	ctx context.Context,
	taskID string,
) ([]ListTaskAttachmentsRow, error) {
	rows, err := s.q.ListTaskAttachments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskAttachmentsRowSlice(rows), nil
}

func (s *PgStore) ListTaskAttachmentIDs(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskAttachmentIDs(ctx, taskID)
}

func (s *PgStore) CreateTaskMessage(
	ctx context.Context,
	p CreateTaskMessageParams,
) error {
	return s.q.CreateTaskMessage(ctx, pgdb.CreateTaskMessageParams(p))
}

func (s *PgStore) ListTaskMessages(
	ctx context.Context,
	taskID string,
) ([]ListTaskMessagesRow, error) {
	rows, err := s.q.ListTaskMessages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskMessagesRowSlice(rows), nil
}

func (s *PgStore) LinkMessageAttachment(
	ctx context.Context,
	p LinkMessageAttachmentParams,
) error {
	return s.q.LinkMessageAttachment(ctx, pgdb.LinkMessageAttachmentParams(p))
}

func (s *PgStore) SetProjectPicture(
	ctx context.Context,
	p SetProjectPictureParams,
) error {
	return s.q.SetProjectPicture(ctx, pgdb.SetProjectPictureParams(p))
}

func (s *PgStore) ClearProjectPicture(
	ctx context.Context,
	p ClearProjectPictureParams,
) error {
	return s.q.ClearProjectPicture(ctx, pgdb.ClearProjectPictureParams(p))
}

func (s *PgStore) SetAgentPicture(ctx context.Context, p SetAgentPictureParams) error {
	return s.q.SetAgentPicture(ctx, pgdb.SetAgentPictureParams(p))
}

func (s *PgStore) ClearAgentPicture(
	ctx context.Context,
	p ClearAgentPictureParams,
) error {
	return s.q.ClearAgentPicture(ctx, pgdb.ClearAgentPictureParams(p))
}

func (s *PgStore) SetUserPicture(ctx context.Context, p SetUserPictureParams) error {
	return s.q.SetUserPicture(ctx, pgdb.SetUserPictureParams(p))
}

func (s *PgStore) ClearUserPicture(ctx context.Context, p ClearUserPictureParams) error {
	return s.q.ClearUserPicture(ctx, pgdb.ClearUserPictureParams(p))
}

func (s *PgStore) GetUserPicture(
	ctx context.Context,
	actorID ids.ActorID,
) (Attachment, error) {
	row, err := s.q.GetUserPicture(ctx, int64(actorID))
	if err != nil {
		return Attachment{}, err
	}
	return fromPgAttachment(row), nil
}

func (s *PgStore) ListUserPictures(ctx context.Context) ([]ListUserPicturesRow, error) {
	rows, err := s.q.ListUserPictures(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListUserPicturesRowSlice(rows), nil
}

func (s *PgStore) GetProjectPicture(
	ctx context.Context,
	projectID string,
) (Attachment, error) {
	row, err := s.q.GetProjectPicture(ctx, projectID)
	if err != nil {
		return Attachment{}, err
	}
	return fromPgAttachment(row), nil
}

func (s *PgStore) GetAgentPicture(
	ctx context.Context,
	agentID ids.AgentID,
) (Attachment, error) {
	row, err := s.q.GetAgentPicture(ctx, int64(agentID))
	if err != nil {
		return Attachment{}, err
	}
	return fromPgAttachment(row), nil
}

func (s *PgStore) ListProjectPictures(
	ctx context.Context,
) ([]ListProjectPicturesRow, error) {
	rows, err := s.q.ListProjectPictures(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListProjectPicturesRowSlice(rows), nil
}

func (s *PgStore) ListAgentPictures(ctx context.Context) ([]ListAgentPicturesRow, error) {
	rows, err := s.q.ListAgentPictures(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListAgentPicturesRowSlice(rows), nil
}

func (s *PgStore) SetAttachmentStorageKey(ctx context.Context, id int64, storageKey string) error {
	return s.q.SetAttachmentStorageKey(
		ctx,
		pgdb.SetAttachmentStorageKeyParams(SetAttachmentStorageKeyParams{
			StorageKey: storageKey,
			ID:         id,
		}),
	)
}

func (s *PgStore) ListTaskArtifacts(
	ctx context.Context,
	taskID string,
) ([]TaskArtifact, error) {
	rows, err := s.q.ListTaskArtifacts(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgTaskArtifactSlice(rows), nil
}

func (s *PgStore) ListArtifactParts(
	ctx context.Context,
	taskID string,
) ([]ArtifactPart, error) {
	rows, err := s.q.ListArtifactParts(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgArtifactPartSlice(rows), nil
}

func (s *PgStore) GetTaskArtifact(
	ctx context.Context,
	taskID, artifactID string,
) (TaskArtifact, error) {
	row, err := s.q.GetTaskArtifact(ctx, pgdb.GetTaskArtifactParams(GetTaskArtifactParams{
		TaskID:     taskID,
		ArtifactID: artifactID,
	}))
	if err != nil {
		return TaskArtifact{}, err
	}
	return fromPgTaskArtifact(row), nil
}

func (s *PgStore) MaxTaskArtifactPosition(ctx context.Context, taskID string) (int64, error) {
	return s.q.MaxTaskArtifactPosition(ctx, taskID)
}

func (s *PgStore) MaxArtifactPartPosition(
	ctx context.Context,
	taskID, artifactID string,
) (int64, error) {
	return s.q.MaxArtifactPartPosition(
		ctx,
		pgdb.MaxArtifactPartPositionParams(MaxArtifactPartPositionParams{
			TaskID:     taskID,
			ArtifactID: artifactID,
		}),
	)
}

func (s *PgStore) UpsertTaskArtifact(
	ctx context.Context,
	p UpsertTaskArtifactParams,
) error {
	return s.q.UpsertTaskArtifact(ctx, pgdb.UpsertTaskArtifactParams(p))
}

func (s *PgStore) SetTaskArtifactStreaming(
	ctx context.Context,
	p SetTaskArtifactStreamingParams,
) error {
	return s.q.SetTaskArtifactStreaming(ctx, pgdb.SetTaskArtifactStreamingParams(p))
}

func (s *PgStore) InsertArtifactPart(
	ctx context.Context,
	p InsertArtifactPartParams,
) error {
	return s.q.InsertArtifactPart(ctx, pgdb.InsertArtifactPartParams(p))
}

func (s *PgStore) DeleteArtifactParts(ctx context.Context, taskID, artifactID string) error {
	return s.q.DeleteArtifactParts(ctx, pgdb.DeleteArtifactPartsParams(DeleteArtifactPartsParams{
		TaskID:     taskID,
		ArtifactID: artifactID,
	}))
}

func (s *PgStore) DeleteTaskArtifact(ctx context.Context, taskID, artifactID string) error {
	return s.q.DeleteTaskArtifact(ctx, pgdb.DeleteTaskArtifactParams(DeleteTaskArtifactParams{
		TaskID:     taskID,
		ArtifactID: artifactID,
	}))
}

func (s *PgStore) DeleteTaskArtifacts(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskArtifacts(ctx, taskID)
}

func (s *PgStore) DeleteArtifactPartsByTask(ctx context.Context, taskID string) error {
	return s.q.DeleteArtifactPartsByTask(ctx, taskID)
}

func (s *PgStore) DeleteArtifactPartsByAttachment(ctx context.Context, attachmentID int64) error {
	return s.q.DeleteArtifactPartsByAttachment(ctx, sql.NullInt64{
		Int64: attachmentID,
		Valid: true,
	})
}

func (s *PgStore) DeleteEmptyTaskArtifacts(ctx context.Context) error {
	return s.q.DeleteEmptyTaskArtifacts(ctx)
}

func (s *PgStore) ListAgentLogs(
	ctx context.Context,
	p ListAgentLogsParams,
) ([]AgentLog, error) {
	rows, err := s.q.ListAgentLogs(ctx, pgdb.ListAgentLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgAgentLogSlice(rows), nil
}

func (s *PgStore) DeleteAgentLogs(ctx context.Context, agentID ids.AgentID) error {
	return s.q.DeleteAgentLogs(ctx, sql.NullInt64{Int64: int64(agentID), Valid: true})
}

func (s *PgStore) ListTaskTeams(
	ctx context.Context,
) ([]ListTaskTeamsRow, error) {
	rows, err := s.q.ListTaskTeams(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskTeamsRowSlice(rows), nil
}

func (s *PgStore) ListTaskTeamsByTask(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskTeamsByTask(ctx, taskID)
}

func (s *PgStore) DeleteTaskTeams(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskTeams(ctx, taskID)
}

func (s *PgStore) AddTaskTeam(ctx context.Context, p AddTaskTeamParams) error {
	return s.q.AddTaskTeam(ctx, pgdb.AddTaskTeamParams(p))
}

func (s *PgStore) AddTaskTeamPrompt(
	ctx context.Context,
	p AddTaskTeamPromptParams,
) error {
	return s.q.AddTaskTeamPrompt(ctx, pgdb.AddTaskTeamPromptParams(p))
}

func (s *PgStore) DeleteTaskTeamPrompts(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskTeamPrompts(ctx, taskID)
}

func (s *PgStore) ListTaskTeamPromptsByTask(
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

func (s *PgStore) DeleteTaskTeamsByProjectTeam(
	ctx context.Context,
	p DeleteTaskTeamsByProjectTeamParams,
) error {
	return s.q.DeleteTaskTeamsByProjectTeam(ctx, pgdb.DeleteTaskTeamsByProjectTeamParams(p))
}

func (s *PgStore) IsActorInTaskTeam(
	ctx context.Context,
	p IsActorInTaskTeamParams,
) (bool, error) {
	return s.q.IsActorInTaskTeam(ctx, pgdb.IsActorInTaskTeamParams(p))
}

func (s *PgStore) ListTaskTeamMemberActorIDs(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskTeamMemberActorIDs(ctx, taskID)
}

func (s *PgStore) CountTasksOwnedByTeam(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountTasksOwnedByTeam(ctx, teamID)
}

func (s *PgStore) ListActorProjectTeams(
	ctx context.Context,
	p ListActorProjectTeamsParams,
) ([]int64, error) {
	return s.q.ListActorProjectTeams(ctx, pgdb.ListActorProjectTeamsParams(p))
}

func (s *PgStore) ListTaskLeads(
	ctx context.Context,
) ([]ListTaskLeadsRow, error) {
	rows, err := s.q.ListTaskLeads(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskLeadsRowSlice(rows), nil
}

func (s *PgStore) ListTaskLeadsByTask(ctx context.Context, taskID string) ([]int64, error) {
	return s.q.ListTaskLeadsByTask(ctx, taskID)
}

func (s *PgStore) DeleteTaskLeads(ctx context.Context, taskID string) error {
	return s.q.DeleteTaskLeads(ctx, taskID)
}

func (s *PgStore) AddTaskLead(ctx context.Context, p AddTaskLeadParams) error {
	return s.q.AddTaskLead(ctx, pgdb.AddTaskLeadParams(p))
}

func (s *PgStore) IsActorTaskLead(
	ctx context.Context,
	p IsActorTaskLeadParams,
) (bool, error) {
	return s.q.IsActorTaskLead(ctx, pgdb.IsActorTaskLeadParams(p))
}

func (s *PgStore) AddTaskSubscription(
	ctx context.Context,
	p AddTaskSubscriptionParams,
) error {
	return s.q.AddTaskSubscription(ctx, pgdb.AddTaskSubscriptionParams(p))
}

func (s *PgStore) DeleteTaskSubscription(
	ctx context.Context,
	p DeleteTaskSubscriptionParams,
) error {
	return s.q.DeleteTaskSubscription(ctx, pgdb.DeleteTaskSubscriptionParams(p))
}

func (s *PgStore) IsActorSubscribedToTask(
	ctx context.Context,
	p IsActorSubscribedToTaskParams,
) (bool, error) {
	return s.q.IsActorSubscribedToTask(ctx, pgdb.IsActorSubscribedToTaskParams(p))
}

func (s *PgStore) ListTaskSubscriptionsByTask(
	ctx context.Context,
	taskID string,
) ([]int64, error) {
	return s.q.ListTaskSubscriptionsByTask(ctx, taskID)
}

func (s *PgStore) ListTaskNotificationRecipients(
	ctx context.Context,
	taskID string,
) ([]int64, error) {
	return s.q.ListTaskNotificationRecipients(ctx, taskID)
}

func (s *PgStore) GetTeam(ctx context.Context, id int64) (Team, error) {
	row, err := s.q.GetTeam(ctx, id)
	if err != nil {
		return Team{}, err
	}
	return fromPgTeam(row), nil
}

func (s *PgStore) GetTeamByName(ctx context.Context, name string) (Team, error) {
	row, err := s.q.GetTeamByName(ctx, name)
	if err != nil {
		return Team{}, err
	}
	return fromPgTeam(row), nil
}

func (s *PgStore) ListTeams(ctx context.Context) ([]ListTeamsRow, error) {
	rows, err := s.q.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListTeamsRowSlice(rows), nil
}

func (s *PgStore) ListTeamsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListTeamsForActorRow, error) {
	rows, err := s.q.ListTeamsForActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgListTeamsForActorRowSlice(rows), nil
}

func (s *PgStore) UpdateTeam(
	ctx context.Context,
	p UpdateTeamParams,
) (Team, error) {
	row, err := s.q.UpdateTeam(ctx, pgdb.UpdateTeamParams(p))
	if err != nil {
		return Team{}, err
	}
	return fromPgTeam(row), nil
}

func (s *PgStore) SetTeamPicture(
	ctx context.Context,
	p SetTeamPictureParams,
) error {
	return s.q.SetTeamPicture(ctx, pgdb.SetTeamPictureParams(p))
}

func (s *PgStore) ClearTeamPicture(
	ctx context.Context,
	p ClearTeamPictureParams,
) error {
	return s.q.ClearTeamPicture(ctx, pgdb.ClearTeamPictureParams(p))
}

func (s *PgStore) CountTeams(ctx context.Context) (int64, error) {
	return s.q.CountTeams(ctx)
}

func (s *PgStore) AddTeamMember(
	ctx context.Context,
	p AddTeamMemberParams,
) error {
	return s.q.AddTeamMember(ctx, pgdb.AddTeamMemberParams(p))
}

func (s *PgStore) RemoveTeamMember(
	ctx context.Context,
	p RemoveTeamMemberParams,
) (int64, error) {
	return s.q.RemoveTeamMember(ctx, pgdb.RemoveTeamMemberParams(p))
}

func (s *PgStore) GetTeamMember(
	ctx context.Context,
	p GetTeamMemberParams,
) (TeamMember, error) {
	row, err := s.q.GetTeamMember(ctx, pgdb.GetTeamMemberParams(p))
	if err != nil {
		return TeamMember{}, err
	}
	return fromPgTeamMember(row), nil
}

func (s *PgStore) ListTeamMembers(
	ctx context.Context,
	teamID int64,
) ([]ListTeamMembersRow, error) {
	rows, err := s.q.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromPgListTeamMembersRowSlice(rows), nil
}

func (s *PgStore) CountTeamMembers(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountTeamMembers(ctx, teamID)
}

func (s *PgStore) CountTeamOwners(ctx context.Context, teamID int64) (int64, error) {
	return s.q.CountTeamOwners(ctx, teamID)
}

func (s *PgStore) CreateTeamInvitation(
	ctx context.Context,
	p CreateTeamInvitationParams,
) (TeamInvitation, error) {
	row, err := s.q.CreateTeamInvitation(ctx, pgdb.CreateTeamInvitationParams(p))
	if err != nil {
		return TeamInvitation{}, err
	}
	return fromPgTeamInvitation(row), nil
}

func (s *PgStore) GetTeamInvitation(
	ctx context.Context,
	id int64,
) (TeamInvitation, error) {
	row, err := s.q.GetTeamInvitation(ctx, id)
	if err != nil {
		return TeamInvitation{}, err
	}
	return fromPgTeamInvitation(row), nil
}

func (s *PgStore) GetPendingTeamInvitation(
	ctx context.Context,
	p GetPendingTeamInvitationParams,
) (TeamInvitation, error) {
	row, err := s.q.GetPendingTeamInvitation(ctx, pgdb.GetPendingTeamInvitationParams(p))
	if err != nil {
		return TeamInvitation{}, err
	}
	return fromPgTeamInvitation(row), nil
}

func (s *PgStore) ListTeamInvitations(
	ctx context.Context,
	teamID int64,
) ([]ListTeamInvitationsRow, error) {
	rows, err := s.q.ListTeamInvitations(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromPgListTeamInvitationsRowSlice(rows), nil
}

func (s *PgStore) ListPendingTeamInvitationsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ListPendingTeamInvitationsForActorRow, error) {
	rows, err := s.q.ListPendingTeamInvitationsForActor(ctx, int64(actorID))
	if err != nil {
		return nil, err
	}
	return fromPgListPendingTeamInvitationsForActorRowSlice(rows), nil
}

func (s *PgStore) RespondTeamInvitation(
	ctx context.Context,
	p RespondTeamInvitationParams,
) (int64, error) {
	return s.q.RespondTeamInvitation(ctx, pgdb.RespondTeamInvitationParams(p))
}

func (s *PgStore) RevokeTeamInvitation(
	ctx context.Context,
	p RevokeTeamInvitationParams,
) (int64, error) {
	return s.q.RevokeTeamInvitation(ctx, pgdb.RevokeTeamInvitationParams(p))
}

func (s *PgStore) ExpireTeamInvitations(ctx context.Context, now int64) (int64, error) {
	return s.q.ExpireTeamInvitations(
		ctx,
		pgdb.ExpireTeamInvitationsParams(ExpireTeamInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now, Valid: true},
			Now:         now,
		}),
	)
}

func (s *PgStore) AddAgentOwner(
	ctx context.Context,
	p AddAgentOwnerParams,
) error {
	return s.q.AddAgentOwner(ctx, pgdb.AddAgentOwnerParams(p))
}

func (s *PgStore) RemoveAgentOwner(
	ctx context.Context,
	p RemoveAgentOwnerParams,
) (int64, error) {
	return s.q.RemoveAgentOwner(ctx, pgdb.RemoveAgentOwnerParams(p))
}

func (s *PgStore) CountAgentOwners(ctx context.Context, agentActorID ids.ActorID) (int64, error) {
	return s.q.CountAgentOwners(ctx, int64(agentActorID))
}

func (s *PgStore) ListAgentOwners(
	ctx context.Context,
	agentActorID ids.ActorID,
) ([]ListAgentOwnersRow, error) {
	rows, err := s.q.ListAgentOwners(ctx, int64(agentActorID))
	if err != nil {
		return nil, err
	}
	return fromPgListAgentOwnersRowSlice(rows), nil
}

func (s *PgStore) ListAgentsOwnedBy(
	ctx context.Context,
	ownerActorID ids.ActorID,
) ([]int64, error) {
	return s.q.ListAgentsOwnedBy(ctx, int64(ownerActorID))
}

func (s *PgStore) DeleteAgentOwners(ctx context.Context, agentActorID ids.ActorID) error {
	return s.q.DeleteAgentOwners(ctx, int64(agentActorID))
}

func (s *PgStore) InsertTeamLog(ctx context.Context, p InsertTeamLogParams) error {
	return s.q.InsertTeamLog(ctx, pgdb.InsertTeamLogParams(p))
}

func (s *PgStore) ListTeamLogs(
	ctx context.Context,
	p ListTeamLogsParams,
) ([]ListTeamLogsRow, error) {
	rows, err := s.q.ListTeamLogs(ctx, pgdb.ListTeamLogsParams(p))
	if err != nil {
		return nil, err
	}
	return fromPgListTeamLogsRowSlice(rows), nil
}
