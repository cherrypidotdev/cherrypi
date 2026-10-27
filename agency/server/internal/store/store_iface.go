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

	"cherrypi.vladfaust.com/agency/internal/ids"
)

type Store interface {
	AcceptInvitation(ctx context.Context, p AcceptInvitationParams) (int64, error)
	RegisterAgent(ctx context.Context, p RegisterAgentParams) (AgentRow, error)
	TransitionTask(ctx context.Context, p TransitionTaskParams) (int64, error)
	AddAgentOwner(ctx context.Context, p AddAgentOwnerParams) error
	AddTaskLead(ctx context.Context, p AddTaskLeadParams) error
	AddTaskReference(ctx context.Context, p UpsertTaskReferenceParams) error
	AddTaskSubscription(ctx context.Context, p AddTaskSubscriptionParams) error
	AddTaskTag(ctx context.Context, p AddTaskTagParams) error
	AddTaskTeam(ctx context.Context, p AddTaskTeamParams) error
	AddTaskTeamPrompt(ctx context.Context, p AddTaskTeamPromptParams) error
	AddTeamMember(ctx context.Context, p AddTeamMemberParams) error
	AllocateTaskNumber(ctx context.Context, projectID string) (int64, error)
	AppendTaskSessionEntries(
		ctx context.Context,
		entries []AppendTaskSessionEntryParams,
	) (int64, error)
	AppendTaskSessionEntry(
		ctx context.Context,
		p AppendTaskSessionEntryParams,
	) (int64, error)
	ClearActorTOTP(ctx context.Context, p ClearActorTOTPParams) error
	ClearAgentPicture(ctx context.Context, p ClearAgentPictureParams) error
	ClearProjectPicture(ctx context.Context, p ClearProjectPictureParams) error
	ClearSettingValue(ctx context.Context, key string, value string) error
	ClearTaskPendingPrompt(
		ctx context.Context,
		p ClearTaskPendingPromptParams,
	) (int64, error)
	ClearTeamPicture(ctx context.Context, p ClearTeamPictureParams) error
	ClearUserPicture(ctx context.Context, p ClearUserPictureParams) error
	Close() error
	ConfirmEmailCredential(ctx context.Context, p ConfirmEmailCredentialParams) error
	ConsumeRecoveryCode(ctx context.Context, p ConsumeRecoveryCodeParams) (int64, error)
	ConsumeToken(ctx context.Context, p ConsumeTokenParams) (int64, error)
	CountAdmins(ctx context.Context) (int64, error)
	CountAgentOwners(ctx context.Context, agentActorID ids.ActorID) (int64, error)
	CountConfirmedEmails(ctx context.Context, actorID ids.ActorID) (int64, error)
	CountIncompleteBlockingReferences(ctx context.Context, taskID string) (int64, error)
	CountProjectChildren(ctx context.Context, parentID string) (int64, error)
	CountProjectsOwnedByTeam(ctx context.Context, teamID int64) (int64, error)
	CountTaskSessionEntries(ctx context.Context, taskID string) (int64, error)
	CountTasksByProject(ctx context.Context, projectID sql.NullString) (int64, error)
	CountTasksOwnedByTeam(ctx context.Context, teamID int64) (int64, error)
	CountTeamMembers(ctx context.Context, teamID int64) (int64, error)
	CountTeamOwners(ctx context.Context, teamID int64) (int64, error)
	CountTeams(ctx context.Context) (int64, error)
	CountUnreadNotifications(ctx context.Context, recipient int64) (int64, error)
	CountUnusedRecoveryCodes(ctx context.Context, actorID ids.ActorID) (int64, error)
	CountUsers(ctx context.Context) (int64, error)
	CreateAttachment(
		ctx context.Context,
		p CreateAttachmentParams,
	) (Attachment, error)
	CreateEmailCredential(
		ctx context.Context,
		p CreateEmailCredentialParams,
	) (EmailCredential, error)
	CreateExternalCredential(
		ctx context.Context,
		p CreateExternalCredentialParams,
	) (ExternalCredential, error)
	CreateIdentityProvider(
		ctx context.Context,
		p CreateIdentityProviderParams,
	) (IdentityProvider, error)
	CreateInvitation(ctx context.Context, p CreateInvitationParams) (InvitationRow, error)
	CreateLDAPCredential(
		ctx context.Context,
		p CreateLDAPCredentialParams,
	) (LdapCredential, error)
	CreateLDAPProvider(
		ctx context.Context,
		p CreateLDAPProviderParams,
	) (LdapProvider, error)
	CreateNotification(
		ctx context.Context,
		p CreateNotificationParams,
	) (Notification, error)
	CreatePersonalAccessToken(
		ctx context.Context,
		p CreatePersonalAccessTokenParams,
	) (PersonalAccessToken, error)
	CreateProject(ctx context.Context, p CreateProjectParams) (Project, error)
	CreateProjectParentInvitation(
		ctx context.Context,
		p CreateProjectParentInvitationParams,
	) (ProjectParentInvitation, error)
	CreateProjectTeamInvitation(
		ctx context.Context,
		p CreateProjectTeamInvitationParams,
	) (ProjectTeamInvitation, error)
	CreateRecoveryCode(
		ctx context.Context,
		p CreateRecoveryCodeParams,
	) (TotpRecoveryCode, error)
	CreateSession(ctx context.Context, p CreateSessionParams) (Session, error)
	CreateTag(ctx context.Context, p CreateTagParams) (Tag, error)
	CreateTask(ctx context.Context, p CreateTaskParams) (Task, error)
	CreateTaskMessage(ctx context.Context, p CreateTaskMessageParams) error
	CreateTeamInvitation(
		ctx context.Context,
		p CreateTeamInvitationParams,
	) (TeamInvitation, error)
	CreateTeamWithOwner(
		ctx context.Context,
		p CreateTeamParams,
		ownerActorID ids.ActorID,
	) (Team, error)
	CreateToken(ctx context.Context, p CreateTokenParams) (Token, error)
	CreateUserAccount(
		ctx context.Context,
		actor CreateUserActorParams,
		user CreateUserRowParams,
	) (UserRow, error)
	CreateUserActor(ctx context.Context, p CreateUserActorParams) (ids.ActorID, error)
	CreateWebhook(ctx context.Context, p CreateWebhookParams) (Webhook, error)
	DeleteActorSetting(ctx context.Context, actorID ids.ActorID, key string) error
	DeleteAgent(ctx context.Context, id ids.AgentID) error
	DeleteAgentDomainLogs(ctx context.Context, actorID int64) error
	DeleteAgentLogs(ctx context.Context, agentID ids.AgentID) error
	DeleteAgentOwners(ctx context.Context, agentActorID ids.ActorID) error
	DeleteAgentProjectLimits(ctx context.Context, agentID ids.AgentID) error
	DeleteAgentProjectTagLimits(ctx context.Context, agentID ids.AgentID) error
	DeleteAgentTagLimits(ctx context.Context, agentID ids.AgentID) error
	DeleteAllNotifications(ctx context.Context, recipient int64) error
	DeleteAnyActor(ctx context.Context, id ids.ActorID) error
	DeleteArtifactParts(ctx context.Context, taskID string, artifactID string) error
	DeleteArtifactPartsByAttachment(ctx context.Context, attachmentID int64) error
	DeleteArtifactPartsByTask(ctx context.Context, taskID string) error
	DeleteAttachment(ctx context.Context, id int64) error
	DeleteEmailCredential(ctx context.Context, p DeleteEmailCredentialParams) error
	DeleteEmptyTaskArtifacts(ctx context.Context) error
	DeleteExpiredInvitations(ctx context.Context, now int64) (int64, error)
	DeleteExpiredInvitedActors(ctx context.Context, now int64) (int64, error)
	DeleteExpiredPersonalAccessTokens(ctx context.Context, now sql.NullInt64) (int64, error)
	DeleteExpiredSessions(ctx context.Context, now int64) (int64, error)
	DeleteExpiredTokens(ctx context.Context, now int64) (int64, error)
	DeleteExternalCredential(ctx context.Context, id int64, actorID ids.ActorID) error
	DeleteIdentityProvider(ctx context.Context, id int64) error
	DeleteInvitationsByActor(ctx context.Context, actorID ids.ActorID) error
	DeleteLDAPCredential(ctx context.Context, id int64, actorID ids.ActorID) error
	DeleteLDAPProvider(ctx context.Context, id int64) error
	DeleteNotification(ctx context.Context, id int64, recipient int64) error
	DeleteOtherSessions(ctx context.Context, p DeleteOtherSessionsParams) error
	DeletePersonalAccessToken(ctx context.Context, p DeletePersonalAccessTokenParams) error
	DeleteProject(ctx context.Context, id string) error
	DeleteRecoveryCodes(ctx context.Context, actorID ids.ActorID) error
	DeleteSessionByHash(ctx context.Context, tokenHash []byte) error
	DeleteSessionByID(ctx context.Context, p DeleteSessionByIDParams) error
	DeleteSessionsByActor(ctx context.Context, actorID ids.ActorID) error
	DeleteTagsByProject(ctx context.Context, projectID string) error
	DeleteTask(ctx context.Context, id string) error
	DeleteTaskArtifact(ctx context.Context, taskID string, artifactID string) error
	DeleteTaskArtifacts(ctx context.Context, taskID string) error
	DeleteTaskLeads(ctx context.Context, taskID string) error
	DeleteTaskLinks(ctx context.Context, taskID string) error
	DeleteTaskSubscription(ctx context.Context, p DeleteTaskSubscriptionParams) error
	DeleteTaskTag(ctx context.Context, p DeleteTaskTagParams) error
	DeleteTaskTags(ctx context.Context, taskID string) error
	DeleteTaskTeamPrompts(ctx context.Context, taskID string) error
	DeleteTaskTeams(ctx context.Context, taskID string) error
	DeleteTaskTeamsByProjectTeam(
		ctx context.Context,
		p DeleteTaskTeamsByProjectTeamParams,
	) error
	DeleteTasksByProjectAgent(ctx context.Context, p DeleteTasksByProjectAgentParams) error
	DeleteTeam(ctx context.Context, id int64) error
	DeleteTokensByActorKind(ctx context.Context, p DeleteTokensByActorKindParams) error
	DeleteUserByActor(ctx context.Context, actorID ids.ActorID) error
	DeleteUserLogs(ctx context.Context, actorID int64) error
	DeleteWebhook(ctx context.Context, id int64) error
	ExpireProjectParentInvitations(ctx context.Context, now int64) (int64, error)
	ExpireProjectTeamInvitations(ctx context.Context, now int64) (int64, error)
	ExpireTeamInvitations(ctx context.Context, now int64) (int64, error)
	GetActorByEmail(ctx context.Context, email string) (Actor, error)
	GetActorByID(ctx context.Context, id ids.ActorID) (Actor, error)
	GetActorByUsername(ctx context.Context, username string) (Actor, error)
	GetActorSetting(ctx context.Context, actorID ids.ActorID, key string) (string, error)
	GetAgent(ctx context.Context, id ids.AgentID) (AgentRow, error)
	GetAgentByActor(ctx context.Context, actorID ids.ActorID) (AgentRow, error)
	GetAgentByUsername(ctx context.Context, username string) (AgentRow, error)
	GetAgentPicture(ctx context.Context, agentID ids.AgentID) (Attachment, error)
	GetAgentTaskLimits(ctx context.Context, agentID ids.AgentID) (AgentTaskLimit, error)
	GetAttachment(ctx context.Context, id int64) (Attachment, error)
	GetEffectiveProjectRole(
		ctx context.Context,
		projectID string,
		actorID ids.ActorID,
	) (int64, error)
	GetEmailCredentialByEmail(ctx context.Context, email string) (EmailCredential, error)
	GetEmailCredentialByID(ctx context.Context, id int64) (EmailCredential, error)
	GetExternalCredential(
		ctx context.Context,
		providerID int64,
		subject string,
	) (ExternalCredential, error)
	GetIdentityProviderByID(ctx context.Context, id int64) (IdentityProvider, error)
	GetIdentityProviderBySlug(ctx context.Context, slug string) (IdentityProvider, error)
	GetInvitationByID(ctx context.Context, id int64) (InvitationRow, error)
	GetInvitationByTokenHash(ctx context.Context, tokenHash []byte) (InvitationRow, error)
	GetLDAPCredential(
		ctx context.Context,
		providerID int64,
		username string,
	) (LdapCredential, error)
	GetLDAPProviderByID(ctx context.Context, id int64) (LdapProvider, error)
	GetLDAPProviderBySlug(ctx context.Context, slug string) (LdapProvider, error)
	GetPendingExternalInvitation(
		ctx context.Context,
		p GetPendingExternalInvitationParams,
	) (InvitationRow, error)
	GetPendingInvitationByActor(ctx context.Context, actorID ids.ActorID) (InvitationRow, error)
	GetPendingInvitationByUsername(ctx context.Context, username string) (InvitationRow, error)
	GetPendingProjectTeamInvitation(
		ctx context.Context,
		p GetPendingProjectTeamInvitationParams,
	) (ProjectTeamInvitation, error)
	GetPendingTeamInvitation(
		ctx context.Context,
		p GetPendingTeamInvitationParams,
	) (TeamInvitation, error)
	GetPersonalAccessTokenByHash(ctx context.Context, tokenHash []byte) (PATRow, error)
	GetProject(ctx context.Context, id string) (Project, error)
	GetProjectActorGrant(
		ctx context.Context,
		p GetProjectActorGrantParams,
	) (ProjectActorMembership, error)
	GetProjectByPrefix(ctx context.Context, prefix string) (Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (Project, error)
	GetProjectParent(ctx context.Context, childID string) (Project, error)
	GetProjectParentInvitation(ctx context.Context, id int64) (ProjectParentInvitation, error)
	GetPendingProjectParentInvitation(
		ctx context.Context,
		p GetPendingProjectParentInvitationParams,
	) (ProjectParentInvitation, error)
	GetPendingProjectParentInvitationForChild(
		ctx context.Context,
		childID string,
	) (ProjectParentInvitation, error)
	GetProjectPicture(ctx context.Context, projectID string) (Attachment, error)
	GetProjectTeamInvitation(ctx context.Context, id int64) (ProjectTeamInvitation, error)
	GetProjectTeamMembership(
		ctx context.Context,
		p GetProjectTeamMembershipParams,
	) (ProjectTeamMembership, error)
	GetPrompt(ctx context.Context, id int64) (Prompt, error)
	GetSessionByHash(ctx context.Context, tokenHash []byte) (SessionRow, error)
	GetSetting(ctx context.Context, key string) (string, error)
	GetTag(ctx context.Context, id string) (Tag, error)
	GetTagByName(ctx context.Context, projectID string, name string) (Tag, error)
	GetTagWithCount(ctx context.Context, id string) (GetTagWithCountRow, error)
	GetTask(ctx context.Context, id string) (Task, error)
	GetTaskArtifact(
		ctx context.Context,
		taskID string,
		artifactID string,
	) (TaskArtifact, error)
	GetTaskByProjectSeq(ctx context.Context, projectID string, seq int64) (Task, error)
	GetTeam(ctx context.Context, id int64) (Team, error)
	GetTeamByName(ctx context.Context, name string) (Team, error)
	GetTeamInvitation(ctx context.Context, id int64) (TeamInvitation, error)
	GetTeamMember(ctx context.Context, p GetTeamMemberParams) (TeamMember, error)
	GetTokenByHash(ctx context.Context, tokenHash []byte) (Token, error)
	GetUserByActor(ctx context.Context, actorID ids.ActorID) (UserRow, error)
	GetUserByUsername(ctx context.Context, username string) (UserRow, error)
	GetUserPicture(ctx context.Context, actorID ids.ActorID) (Attachment, error)
	GetWebhook(ctx context.Context, id int64) (Webhook, error)
	InsertAgentDomainLog(ctx context.Context, p InsertAgentDomainLogParams) error
	InsertAgentLog(ctx context.Context, p InsertAgentLogParams, keep int64) error
	InsertAgentProjectLimit(ctx context.Context, p InsertAgentProjectLimitParams) error
	InsertAgentProjectTagLimit(
		ctx context.Context,
		p InsertAgentProjectTagLimitParams,
	) error
	InsertAgentTagLimit(ctx context.Context, p InsertAgentTagLimitParams) error
	InsertAgentUsage(ctx context.Context, p InsertAgentUsageParams) error
	InsertArtifactPart(ctx context.Context, p InsertArtifactPartParams) error
	InsertProjectLog(ctx context.Context, p InsertProjectLogParams) error
	InsertPrompt(ctx context.Context, p InsertPromptParams) (Prompt, error)
	InsertTaskLink(ctx context.Context, p InsertTaskLinkParams) error
	InsertTaskLog(ctx context.Context, p InsertTaskLogParams) error
	InsertTeamLog(ctx context.Context, p InsertTeamLogParams) error
	InsertUserLog(ctx context.Context, p InsertUserLogParams) error
	IsActorInTaskTeam(ctx context.Context, p IsActorInTaskTeamParams) (bool, error)
	IsProjectDescendant(ctx context.Context, projectID, ancestorID string) (bool, error)
	IsActorSubscribedToTask(
		ctx context.Context,
		p IsActorSubscribedToTaskParams,
	) (bool, error)
	IsActorTaskLead(ctx context.Context, p IsActorTaskLeadParams) (bool, error)
	IsAgentOwner(
		ctx context.Context,
		agentActorID ids.ActorID,
		ownerActorID ids.ActorID,
	) (bool, error)
	LatestPrompt(ctx context.Context, p LatestPromptParams) (Prompt, error)
	LatestPromptVersion(ctx context.Context, p LatestPromptVersionParams) (int64, error)
	LinkMessageAttachment(ctx context.Context, p LinkMessageAttachmentParams) error
	LinkTaskAttachment(ctx context.Context, p LinkTaskAttachmentParams) error
	ListActorProjectTeams(
		ctx context.Context,
		p ListActorProjectTeamsParams,
	) ([]int64, error)
	ListActorSettings(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]ListActorSettingsRow, error)
	ListAgentDomainLogs(
		ctx context.Context,
		p ListAgentDomainLogsParams,
	) ([]ListAgentDomainLogsRow, error)
	ListAgentLogs(ctx context.Context, p ListAgentLogsParams) ([]AgentLog, error)
	ListAgentOwners(
		ctx context.Context,
		agentActorID ids.ActorID,
	) ([]ListAgentOwnersRow, error)
	ListAgentPictures(ctx context.Context) ([]ListAgentPicturesRow, error)
	ListAgentProjectLimits(
		ctx context.Context,
		agentID ids.AgentID,
	) ([]AgentProjectLimit, error)
	ListAgentProjectMemberships(
		ctx context.Context,
		agentID ids.AgentID,
	) ([]ListAgentProjectMembershipsRow, error)
	ListAgentProjectTagLimits(
		ctx context.Context,
		agentID ids.AgentID,
	) ([]AgentProjectTagLimit, error)
	ListAgentProjects(ctx context.Context, agentID ids.AgentID) ([]Project, error)
	ListAgentTagLimits(ctx context.Context, agentID ids.AgentID) ([]AgentTagLimit, error)
	ListAgentUsage(
		ctx context.Context,
		agentID sql.NullInt64,
		projectID sql.NullString,
		sinceMs sql.NullInt64,
		untilMs sql.NullInt64,
	) ([]ListAgentUsageRow, error)
	ListAgentUsageByDay(
		ctx context.Context,
		agentID sql.NullInt64,
		projectID sql.NullString,
		sinceMs sql.NullInt64,
		untilMs sql.NullInt64,
	) ([]ListAgentUsageByDayRow, error)
	ListAgents(ctx context.Context) ([]AgentRow, error)
	ListAgentsOwnedBy(ctx context.Context, ownerActorID ids.ActorID) ([]int64, error)
	ListAllBlockingTaskReferences(
		ctx context.Context,
	) ([]ListAllBlockingTaskReferencesRow, error)
	ListArtifactParts(ctx context.Context, taskID string) ([]ArtifactPart, error)
	ListDomainLogs(
		ctx context.Context,
		p ListDomainLogsParams,
	) ([]ListDomainLogsRow, error)
	ListEmailCredentials(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]EmailCredential, error)
	ListEnabledIdentityProviders(ctx context.Context) ([]IdentityProvider, error)
	ListEnabledLDAPProviders(ctx context.Context) ([]LdapProvider, error)
	ListExternalCredentialsByActor(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]ExternalCredential, error)
	ListIdentityProviders(ctx context.Context) ([]IdentityProvider, error)
	ListInvitations(ctx context.Context) ([]InvitationRow, error)
	ListLDAPCredentialsByActor(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]LdapCredential, error)
	ListLDAPProviders(ctx context.Context) ([]LdapProvider, error)
	ListNotifications(
		ctx context.Context,
		recipient int64,
		limit int64,
	) ([]ListNotificationsRow, error)
	ListOrphanAttachments(ctx context.Context) ([]Attachment, error)
	ListPendingProjectInvitationsForTeamOwner(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]ListPendingProjectInvitationsForTeamOwnerRow, error)
	ListPendingTeamInvitationsForActor(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]ListPendingTeamInvitationsForActorRow, error)
	ListPersonalAccessTokens(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]PersonalAccessToken, error)
	ListCrossProjectTaskReferences(
		ctx context.Context,
	) ([]ListCrossProjectTaskReferencesRow, error)
	ListProjectActorGrants(
		ctx context.Context,
		projectID string,
	) ([]ListProjectActorGrantsRow, error)
	ListProjectAncestorIDs(ctx context.Context, projectID string) ([]string, error)
	ListProjectChildren(ctx context.Context, parentID string) ([]Project, error)
	ListProjectDescendantIDs(ctx context.Context, projectID string) ([]string, error)
	ListProjectLogs(
		ctx context.Context,
		p ListProjectLogsParams,
	) ([]ListProjectLogsRow, error)
	ListProjectMemberActorIDs(ctx context.Context, projectID string) ([]int64, error)
	ListProjectParentInvitationsByChild(
		ctx context.Context,
		childID string,
	) ([]ListProjectParentInvitationsByChildRow, error)
	ListProjectParentInvitationsByParent(
		ctx context.Context,
		parentID string,
	) ([]ListProjectParentInvitationsByParentRow, error)
	ListProjectPictures(ctx context.Context) ([]ListProjectPicturesRow, error)
	ListProjectSharedOwnerTeams(
		ctx context.Context,
		parentID, childID string,
		actorID ids.ActorID,
	) ([]ListProjectSharedOwnerTeamsRow, error)
	ListProjectTeamInvitations(
		ctx context.Context,
		projectID string,
	) ([]ListProjectTeamInvitationsRow, error)
	ListProjectTeamMemberships(
		ctx context.Context,
		projectID string,
	) ([]ListProjectTeamMembershipsRow, error)
	ListProjectTeamMembershipsForTeam(
		ctx context.Context,
		teamID int64,
	) ([]ListProjectTeamMembershipsForTeamRow, error)
	ListProjects(ctx context.Context) ([]Project, error)
	ListProjectsForActor(ctx context.Context, actorID ids.ActorID) ([]Project, error)
	ListQueuedTasks(ctx context.Context) ([]Task, error)
	ListRunningTaskTagsByAgent(
		ctx context.Context,
		agentID ids.AgentID,
	) ([]ListRunningTaskTagsByAgentRow, error)
	ListRunningTasks(ctx context.Context) ([]Task, error)
	ListRunningTasksByAgent(ctx context.Context, agentID ids.AgentID) ([]Task, error)
	ListSessionsByActor(ctx context.Context, actorID ids.ActorID) ([]Session, error)
	ListSettings(ctx context.Context) ([]Setting, error)
	ListTags(
		ctx context.Context,
		projectID string,
		includeArchived bool,
	) ([]ListTagsRow, error)
	ListTaskArtifacts(ctx context.Context, taskID string) ([]TaskArtifact, error)
	ListTaskAttachmentIDs(ctx context.Context, taskID string) ([]int64, error)
	ListTaskAttachments(
		ctx context.Context,
		taskID string,
	) ([]ListTaskAttachmentsRow, error)
	ListTaskLeads(ctx context.Context) ([]ListTaskLeadsRow, error)
	ListTaskLeadsByTask(ctx context.Context, taskID string) ([]int64, error)
	ListTaskLinks(ctx context.Context) ([]ListTaskLinksRow, error)
	ListTaskLinksByTask(
		ctx context.Context,
		taskID string,
	) ([]ListTaskLinksByTaskRow, error)
	ListTaskLogsByTask(ctx context.Context, taskID string) ([]ListTaskLogsByTaskRow, error)
	ListTaskMessages(ctx context.Context, taskID string) ([]ListTaskMessagesRow, error)
	ListTaskNotificationRecipients(ctx context.Context, taskID string) ([]int64, error)
	ListTaskReferenceIDs(ctx context.Context, taskID string) ([]string, error)
	ListTaskReferenceRefs(ctx context.Context) ([]ListTaskReferenceRefsRow, error)
	ListTaskReferencedBy(ctx context.Context, taskID string) ([]Task, error)
	ListTaskReferencedByRefs(ctx context.Context) ([]ListTaskReferencedByRefsRow, error)
	ListTaskReferences(ctx context.Context, taskID string) ([]Task, error)
	ListTaskSessionEntries(ctx context.Context, taskID string) ([]TaskSessionEntry, error)
	ListTaskSubscriptionsByTask(ctx context.Context, taskID string) ([]int64, error)
	ListTaskTags(ctx context.Context) ([]ListTaskTagsRow, error)
	ListTaskTagsByTask(ctx context.Context, taskID string) ([]ListTaskTagsByTaskRow, error)
	ListTaskTeamMemberActorIDs(ctx context.Context, taskID string) ([]int64, error)
	ListTaskTeamPromptsByTask(ctx context.Context, taskID string) ([]TaskTeamPrompt, error)
	ListTaskTeams(ctx context.Context) ([]ListTaskTeamsRow, error)
	ListTaskTeamsByTask(ctx context.Context, taskID string) ([]int64, error)
	ListTasks(ctx context.Context, p ListTasksParams) ([]Task, error)
	ListTasksByProject(ctx context.Context, projectID sql.NullString) ([]Task, error)
	ListTasksByProjectAgent(
		ctx context.Context,
		p ListTasksByProjectAgentParams,
	) ([]Task, error)
	ListTeamInvitations(
		ctx context.Context,
		teamID int64,
	) ([]ListTeamInvitationsRow, error)
	ListTeamLogs(
		ctx context.Context,
		p ListTeamLogsParams,
	) ([]ListTeamLogsRow, error)
	ListTeamMembers(ctx context.Context, teamID int64) ([]ListTeamMembersRow, error)
	ListTeams(ctx context.Context) ([]ListTeamsRow, error)
	ListTeamsForActor(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]ListTeamsForActorRow, error)
	ListUnusedRecoveryCodes(
		ctx context.Context,
		actorID ids.ActorID,
	) ([]TotpRecoveryCode, error)
	ListUserLogs(
		ctx context.Context,
		p ListUserLogsParams,
	) ([]ListUserLogsRow, error)
	ListUserPictures(ctx context.Context) ([]ListUserPicturesRow, error)
	ListUsers(ctx context.Context) ([]UserRow, error)
	ListWebhooks(ctx context.Context) ([]Webhook, error)
	MarkAllNotificationsRead(ctx context.Context, readAt sql.NullInt64, recipient int64) error
	MarkNotificationRead(ctx context.Context, p MarkNotificationReadParams) error
	MarkTaskNotificationsRead(ctx context.Context, p MarkTaskNotificationsReadParams) error
	MarkTaskSessionDeleted(
		ctx context.Context,
		p MarkTaskSessionDeletedParams,
	) (Task, error)
	MaxArtifactPartPosition(ctx context.Context, taskID string, artifactID string) (int64, error)
	MaxTaskArtifactPosition(ctx context.Context, taskID string) (int64, error)
	PruneAgentDomainLogs(ctx context.Context, beforeAt int64) error
	PruneDomainLogs(ctx context.Context, beforeAt int64) error
	PruneUserLogs(ctx context.Context, beforeAt int64) error
	RefreshUnreadNotification(
		ctx context.Context,
		p RefreshUnreadNotificationParams,
	) (Notification, error)
	RemoveAgentOwner(ctx context.Context, p RemoveAgentOwnerParams) (int64, error)
	RemoveProjectActorGrant(ctx context.Context, p RemoveProjectActorGrantParams) error
	RemoveProjectTeamMembership(
		ctx context.Context,
		p RemoveProjectTeamMembershipParams,
	) error
	RemoveTaskReference(ctx context.Context, p RemoveTaskReferenceParams) (int64, error)
	RespondProjectParentInvitation(
		ctx context.Context,
		p RespondProjectParentInvitationParams,
	) (int64, error)
	RemoveTeamMember(ctx context.Context, p RemoveTeamMemberParams) (int64, error)
	RenameTask(ctx context.Context, p RenameTaskParams) (Task, error)
	RespondProjectTeamInvitation(
		ctx context.Context,
		p RespondProjectTeamInvitationParams,
	) (int64, error)
	RespondTeamInvitation(
		ctx context.Context,
		p RespondTeamInvitationParams,
	) (int64, error)
	RevokeInvitation(ctx context.Context, p RevokeInvitationParams) (int64, error)
	RevokePendingProjectParentInvitationsForChild(
		ctx context.Context,
		p RevokePendingProjectParentInvitationsForChildParams,
	) (int64, error)
	RevokeProjectParentInvitation(
		ctx context.Context,
		p RevokeProjectParentInvitationParams,
	) (int64, error)
	RevokeProjectTeamInvitation(
		ctx context.Context,
		p RevokeProjectTeamInvitationParams,
	) (int64, error)
	RevokeTeamInvitation(ctx context.Context, p RevokeTeamInvitationParams) (int64, error)
	SessionAgentID(ctx context.Context, sessionID string) (ids.AgentID, error)
	SetAgentPicture(ctx context.Context, p SetAgentPictureParams) error
	SetAttachmentSize(ctx context.Context, id int64, size int64) error
	SetAttachmentStorageKey(ctx context.Context, id int64, storageKey string) error
	SetProjectArchived(
		ctx context.Context,
		p SetProjectArchivedParams,
	) (Project, error)
	SetProjectParentTree(
		ctx context.Context,
		childID string,
		parentID string,
		ancestors []InsertProjectAncestorParams,
	) (Project, error)
	SetProjectPicture(ctx context.Context, p SetProjectPictureParams) error
	SetTagArchived(ctx context.Context, p SetTagArchivedParams) (Tag, error)
	SetTaskArtifactStreaming(ctx context.Context, p SetTaskArtifactStreamingParams) error
	SetTaskAssignee(ctx context.Context, p SetTaskAssigneeParams) (Task, error)
	SetTaskError(ctx context.Context, p SetTaskErrorParams) (int64, error)
	SetTaskNamePrompt(ctx context.Context, p SetTaskNamePromptParams) error
	SetTaskPendingPrompt(ctx context.Context, p SetTaskPendingPromptParams) (int64, error)
	SetTaskPriority(ctx context.Context, p SetTaskPriorityParams) (Task, error)
	SetTaskPrompt(ctx context.Context, p SetTaskPromptParams) (int64, error)
	SetTaskSession(ctx context.Context, p SetTaskSessionParams) (int64, error)
	SetTaskSize(ctx context.Context, p SetTaskSizeParams) (Task, error)
	SetTeamPicture(ctx context.Context, p SetTeamPictureParams) error
	SetUserMustChangePassword(ctx context.Context, p SetUserMustChangePasswordParams) error
	SetUserPicture(ctx context.Context, p SetUserPictureParams) error
	SumAgentUsage(ctx context.Context) (SumAgentUsageRow, error)
	TaskIDBySession(ctx context.Context, p TaskIDBySessionParams) (string, error)
	TaskSessionEntryTip(ctx context.Context, taskID string) (TaskSessionEntryTipRow, error)
	TouchAgentSeen(ctx context.Context, id ids.AgentID, at int64) error
	TouchPersonalAccessToken(ctx context.Context, p TouchPersonalAccessTokenParams) error
	TouchSession(ctx context.Context, p TouchSessionParams) error
	TouchUserLogin(ctx context.Context, p TouchUserLoginParams) error
	UnlinkTaskAttachment(ctx context.Context, taskID string, attachmentID int64) error
	UpdateActorPassword(ctx context.Context, p UpdateActorPasswordParams) error
	UpdateActorProfile(ctx context.Context, p UpdateActorProfileParams) error
	UpdateActorTOTP(ctx context.Context, p UpdateActorTOTPParams) error
	UpdateAgent(
		ctx context.Context,
		id ids.AgentID,
		name string,
		description string,
		updatedAt int64,
	) (AgentRow, error)
	SetAgentRole(ctx context.Context, id ids.AgentID, role string) error
	UpdateDraftTask(ctx context.Context, p UpdateDraftTaskParams) (Task, error)
	UpdateIdentityProvider(ctx context.Context, p UpdateIdentityProviderParams) error
	UpdateInvitationTokenHash(ctx context.Context, p UpdateInvitationTokenHashParams) error
	UpdateLDAPProvider(ctx context.Context, p UpdateLDAPProviderParams) error
	UpdateProject(
		ctx context.Context,
		p UpdateProjectParams,
		renameTasks bool,
	) (Project, error)
	UpdateProjectActorGrantRole(
		ctx context.Context,
		p UpdateProjectActorGrantRoleParams,
	) error
	UpdateProjectTeamMembershipRole(
		ctx context.Context,
		p UpdateProjectTeamMembershipRoleParams,
	) error
	UpdateTag(ctx context.Context, p UpdateTagParams) (Tag, error)
	UpdateTeam(ctx context.Context, p UpdateTeamParams) (Team, error)
	UpdateUserRole(ctx context.Context, p UpdateUserRoleParams) error
	UpdateUserStatus(ctx context.Context, p UpdateUserStatusParams) error
	UpdateWebhook(ctx context.Context, p UpdateWebhookParams) (Webhook, error)
	UpsertActorSetting(ctx context.Context, p UpsertActorSettingParams) error
	UpsertAgentTaskLimits(ctx context.Context, p UpsertAgentTaskLimitsParams) error
	UpsertProjectActorGrant(ctx context.Context, p UpsertProjectActorGrantParams) error
	UpsertProjectTeamMembership(
		ctx context.Context,
		p UpsertProjectTeamMembershipParams,
	) error
	UpsertSetting(ctx context.Context, p UpsertSettingParams) error
	UpsertTaskArtifact(ctx context.Context, p UpsertTaskArtifactParams) error
}
