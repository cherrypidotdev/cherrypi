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

package agentgroups

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

var (
	ErrAccessNotFound         = errors.New("group access invitation not found")
	ErrAccessNotPending       = errors.New("group access invitation is not pending")
	ErrAccessExpired          = errors.New("group access invitation has expired")
	ErrAccessAlreadyLinked    = errors.New("group already has access")
	ErrAccessInvitationExists = errors.New("a pending group access invitation already exists")
	ErrAccessNotLinked        = errors.New("group does not have access")
	ErrAccessInvalidRole      = errors.New("group project role must be viewer or member")
	ErrAccessInvalidTarget    = errors.New("access target must be project or team")
	ErrAccessForbidden        = errors.New("group owner, project owner or team owner required")
)

const (
	AccessTargetProject = "project"
	AccessTargetTeam    = "team"
)

type ProjectAccess struct {
	ProjectID         string
	ProjectName       string
	ProjectSlug       string
	Role              string
	GrantedByActorID  ids.ActorID
	GrantedByUsername string
	GrantedByName     string
	CreatedAt         time.Time
}

type TeamAccess struct {
	TeamID            int64
	TeamName          string
	TeamDescription   string
	MemberCount       int64
	GrantedByActorID  ids.ActorID
	GrantedByUsername string
	GrantedByName     string
	CreatedAt         time.Time
}

type AccessInvitation struct {
	ID                  int64
	TargetKind          string
	ProjectID           string
	ProjectName         string
	ProjectSlug         string
	TeamID              int64
	TeamName            string
	GroupID             int64
	GroupName           string
	GroupSlug           string
	Role                string
	Status              string
	Note                string
	InvitedByActorID    ids.ActorID
	InvitedByUsername   string
	InvitedByName       string
	CreatedAt           time.Time
	ExpiresAt           time.Time
	RespondedAt         *time.Time
	RespondedByActorID  ids.ActorID
	RespondedByUsername string
	RespondedByName     string
	CanRespond          bool
}

type Access struct {
	Projects    []ProjectAccess
	Teams       []TeamAccess
	Invitations []AccessInvitation
}

type ProjectGroupLink struct {
	GroupID                  int64
	GroupName                string
	GroupSlug                string
	GroupDescription         string
	Role                     string
	MemberCount              int64
	Inherited                bool
	InheritedFromProjectID   string
	InheritedFromProjectName string
	InheritedFromProjectSlug string
	GrantedByActorID         ids.ActorID
	GrantedByUsername        string
	GrantedByName            string
	CreatedAt                time.Time
}

type ProjectGroupAccess struct {
	Links       []ProjectGroupLink
	Inherited   []ProjectGroupLink
	Invitations []AccessInvitation
}

type TeamGroupLink struct {
	GroupID           int64
	GroupName         string
	GroupSlug         string
	GroupDescription  string
	MemberCount       int64
	GrantedByActorID  ids.ActorID
	GrantedByUsername string
	GrantedByName     string
	CreatedAt         time.Time
}

type TeamGroupAccess struct {
	Links       []TeamGroupLink
	Invitations []AccessInvitation
}

// InviteResult reports whether an invitation was created or the group was

type InviteResult struct {
	Invitation AccessInvitation
	Attached   bool
}

func (s *Service) Access(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
) (Access, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return Access{}, err
	}
	projectLinks, err := s.projectAccess(ctx, groupID)
	if err != nil {
		return Access{}, err
	}
	teamLinks, err := s.teamAccess(ctx, groupID)
	if err != nil {
		return Access{}, err
	}
	invitations, err := s.groupInvitations(ctx, groupID)
	if err != nil {
		return Access{}, err
	}
	canRespond := identity.IsAdmin()
	if !canRespond {
		canRespond, err = s.isOwner(ctx, groupID, identity.ActorID)
		if err != nil {
			return Access{}, err
		}
	}
	for i := range invitations {
		invitations[i].CanRespond = canRespond && invitations[i].Status == "pending"
	}
	return Access{Projects: projectLinks, Teams: teamLinks, Invitations: invitations}, nil
}

func (s *Service) ProjectAccess(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) (ProjectGroupAccess, error) {
	project, err := s.Projects.Get(ctx, projectID)
	if err != nil {
		return ProjectGroupAccess{}, err
	}
	if _, err := s.Projects.RequireRole(
		ctx, project.ID, actorID, isAdmin, projects.RoleViewer,
	); err != nil {
		return ProjectGroupAccess{}, err
	}
	direct, err := s.store.ListProjectGroupLinks(ctx, project.ID)
	if err != nil {
		return ProjectGroupAccess{}, err
	}
	inherited, err := s.store.ListInheritedProjectGroupLinks(ctx, project.ID)
	if err != nil {
		return ProjectGroupAccess{}, err
	}
	rows, err := s.store.ListProjectGroupInvitations(ctx, project.ID)
	if err != nil {
		return ProjectGroupAccess{}, err
	}
	out := ProjectGroupAccess{
		Links:     make([]ProjectGroupLink, len(direct)),
		Inherited: make([]ProjectGroupLink, len(inherited)),
	}
	for i, row := range direct {
		out.Links[i] = ProjectGroupLink{
			GroupID:           row.GroupID,
			GroupName:         row.GroupName,
			GroupSlug:         row.GroupSlug,
			GroupDescription:  row.GroupDescription,
			Role:              row.Role,
			MemberCount:       row.MemberCount,
			GrantedByActorID:  ids.ActorID(row.GrantedByActorID.Int64),
			GrantedByUsername: row.GrantedByUsername.String,
			GrantedByName:     row.GrantedByName.String,
			CreatedAt:         time.UnixMilli(row.CreatedAt),
		}
	}
	for i, row := range inherited {
		out.Inherited[i] = ProjectGroupLink{
			GroupID:                  row.GroupID,
			GroupName:                row.GroupName,
			GroupSlug:                row.GroupSlug,
			GroupDescription:         row.GroupDescription,
			Role:                     roleFromWeight(row.RoleWeight),
			MemberCount:              row.MemberCount,
			Inherited:                true,
			InheritedFromProjectID:   row.SourceProjectID,
			InheritedFromProjectName: row.SourceProjectName,
			InheritedFromProjectSlug: row.SourceProjectSlug,
		}
	}
	for _, row := range rows {
		out.Invitations = append(
			out.Invitations,
			projectInvitationFromHostList(row, project.Name, project.Slug),
		)
	}
	return out, nil
}

func (s *Service) TeamAccess(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) (TeamGroupAccess, error) {
	team, err := s.Teams.Get(ctx, teamID, actorID, isAdmin)
	if err != nil {
		return TeamGroupAccess{}, err
	}
	links, err := s.store.ListTeamGroupLinks(ctx, teamID)
	if err != nil {
		return TeamGroupAccess{}, err
	}
	rows, err := s.store.ListTeamGroupInvitations(ctx, teamID)
	if err != nil {
		return TeamGroupAccess{}, err
	}
	out := TeamGroupAccess{Links: make([]TeamGroupLink, len(links))}
	for i, row := range links {
		out.Links[i] = TeamGroupLink{
			GroupID:           row.GroupID,
			GroupName:         row.GroupName,
			GroupSlug:         row.GroupSlug,
			GroupDescription:  row.GroupDescription,
			MemberCount:       row.MemberCount,
			GrantedByActorID:  ids.ActorID(row.GrantedByActorID.Int64),
			GrantedByUsername: row.GrantedByUsername.String,
			GrantedByName:     row.GrantedByName.String,
			CreatedAt:         time.UnixMilli(row.CreatedAt),
		}
	}
	for _, row := range rows {
		out.Invitations = append(out.Invitations, teamInvitationFromHostList(row, team.Name))
	}
	return out, nil
}

// InviteProject asks a group to join a project. A project owner starts the

func (s *Service) InviteProject(
	ctx context.Context,
	identity auth.Identity,
	projectID string,
	groupID int64,
	role, note string,
) (InviteResult, error) {
	project, err := s.Projects.Get(ctx, projectID)
	if err != nil {
		return InviteResult{}, err
	}
	if _, err := s.Projects.RequireRole(
		ctx, project.ID, identity.ActorID, identity.IsAdmin(), projects.RoleOwner,
	); err != nil {
		return InviteResult{}, err
	}
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return InviteResult{}, err
	}
	if group.ArchivedAt != nil {
		return InviteResult{}, ErrArchived
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = projects.RoleMember
	}
	if role != projects.RoleViewer && role != projects.RoleMember {
		return InviteResult{}, ErrAccessInvalidRole
	}
	if _, err := s.store.GetProjectGroupMembership(
		ctx,
		store.GetProjectGroupMembershipParams{ProjectID: project.ID, GroupID: groupID},
	); err == nil {
		return InviteResult{}, ErrAccessAlreadyLinked
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}
	if _, err := s.store.GetPendingProjectGroupInvitation(
		ctx,
		store.GetPendingProjectGroupInvitationParams{
			ProjectID: project.ID,
			GroupID:   groupID,
		},
	); err == nil {
		return InviteResult{}, ErrAccessInvitationExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}

	inherited, err := s.store.GetEffectiveProjectGroupRole(
		ctx,
		store.GetEffectiveProjectGroupRoleParams{
			ProjectID: project.ID,
			GroupID:   groupID,
		},
	)
	if err != nil {
		return InviteResult{}, err
	}
	groupOwner, err := s.isOwner(ctx, groupID, identity.ActorID)
	if err != nil {
		return InviteResult{}, err
	}
	if identity.IsAdmin() || groupOwner || inherited >= roleWeight(role) {
		if err := s.attachProject(ctx, project.ID, groupID, role, identity.ActorID); err != nil {
			return InviteResult{}, err
		}
		s.writeProjectLog(ctx, project.ID, identity.ActorID, "group", "attached", "",
			group.Name, "Attached group "+group.Name+" as "+role,
			map[string]any{"group_id": groupID, "role": role})
		s.writeLog(ctx, groupID, identity.ActorID, "access", "attached", "", role,
			"Attached to project "+project.Name+" as "+role,
			map[string]any{"project_id": project.ID, "role": role})
		return InviteResult{Attached: true, Invitation: AccessInvitation{
			TargetKind:  AccessTargetProject,
			ProjectID:   project.ID,
			ProjectName: project.Name,
			ProjectSlug: project.Slug,
			GroupID:     groupID,
			GroupName:   group.Name,
			GroupSlug:   group.Slug,
			Role:        role,
			Status:      "accepted",
		}}, nil
	}

	now := time.Now()
	row, err := s.store.CreateProjectGroupInvitation(
		ctx,
		store.CreateProjectGroupInvitationParams{
			ProjectID: project.ID,
			GroupID:   groupID,
			InvitedByActorID: sql.NullInt64{
				Int64: int64(identity.ActorID),
				Valid: identity.ActorID != 0,
			},
			Role:      role,
			Note:      strings.TrimSpace(note),
			CreatedAt: now.UnixMilli(),
			ExpiresAt: now.Add(RequestTTL).UnixMilli(),
		},
	)
	if err != nil {
		return InviteResult{}, err
	}
	s.writeProjectLog(ctx, project.ID, identity.ActorID, "invitation", "invited", "",
		group.Name, "Invited group "+group.Name+" as "+role,
		map[string]any{"group_id": groupID, "role": role})
	s.writeLog(ctx, groupID, identity.ActorID, "access", "invited", "", role,
		"Invited to project "+project.Name+" as "+role,
		map[string]any{"project_id": project.ID, "role": role})
	s.notifyGroupOwners(ctx, groupID, notifications.Notification{
		Kind:  KindAccessInvited,
		Title: "Invitation: " + group.Name + " → " + project.Name,
		Body: "A project owner invited the group to " + project.Name +
			" as " + role + ". A group owner must accept it.",
	})
	return InviteResult{Invitation: projectInvitationFromRecord(
		row, project.Name, project.Slug, group.Name, group.Slug,
	)}, nil
}

// InviteTeam asks a group to join a team. Any team member starts the invitation;

func (s *Service) InviteTeam(
	ctx context.Context,
	identity auth.Identity,
	teamID, groupID int64,
	note string,
) (InviteResult, error) {
	team, err := s.Teams.Get(ctx, teamID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		return InviteResult{}, err
	}
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return InviteResult{}, err
	}
	if group.ArchivedAt != nil {
		return InviteResult{}, ErrArchived
	}
	if _, err := s.store.GetTeamGroupMembership(
		ctx,
		store.GetTeamGroupMembershipParams{TeamID: teamID, GroupID: groupID},
	); err == nil {
		return InviteResult{}, ErrAccessAlreadyLinked
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}
	if _, err := s.store.GetPendingTeamGroupInvitation(
		ctx,
		store.GetPendingTeamGroupInvitationParams{TeamID: teamID, GroupID: groupID},
	); err == nil {
		return InviteResult{}, ErrAccessInvitationExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}
	groupOwner, err := s.isOwner(ctx, groupID, identity.ActorID)
	if err != nil {
		return InviteResult{}, err
	}
	if identity.IsAdmin() || groupOwner {
		if err := s.attachTeam(ctx, teamID, groupID, identity.ActorID); err != nil {
			return InviteResult{}, err
		}
		s.writeTeamLog(ctx, teamID, identity.ActorID, "group", "attached", "",
			group.Name, "Attached group "+group.Name,
			map[string]any{"group_id": groupID})
		s.writeLog(ctx, groupID, identity.ActorID, "access", "attached", "", "member",
			"Attached to team "+team.Name, map[string]any{"team_id": teamID})
		return InviteResult{Attached: true, Invitation: AccessInvitation{
			TargetKind: AccessTargetTeam,
			TeamID:     teamID,
			TeamName:   team.Name,
			GroupID:    groupID,
			GroupName:  group.Name,
			GroupSlug:  group.Slug,
			Role:       teams.RoleMember,
			Status:     "accepted",
		}}, nil
	}

	now := time.Now()
	row, err := s.store.CreateTeamGroupInvitation(
		ctx,
		store.CreateTeamGroupInvitationParams{
			TeamID:  teamID,
			GroupID: groupID,
			InvitedByActorID: sql.NullInt64{
				Int64: int64(identity.ActorID),
				Valid: identity.ActorID != 0,
			},
			Note:      strings.TrimSpace(note),
			CreatedAt: now.UnixMilli(),
			ExpiresAt: now.Add(RequestTTL).UnixMilli(),
		},
	)
	if err != nil {
		return InviteResult{}, err
	}
	s.writeTeamLog(ctx, teamID, identity.ActorID, "invitation", "invited", "",
		group.Name, "Invited group "+group.Name, map[string]any{"group_id": groupID})
	s.writeLog(ctx, groupID, identity.ActorID, "access", "invited", "", "member",
		"Invited to team "+team.Name, map[string]any{"team_id": teamID})
	s.notifyGroupOwners(ctx, groupID, notifications.Notification{
		Kind:  KindAccessInvited,
		Title: "Invitation: " + group.Name + " → " + team.Name,
		Body: "A team member invited the group to " + team.Name +
			". A group owner must accept it.",
	})
	return InviteResult{Invitation: teamInvitationFromRecord(
		row, team.Name, team.Description, group.Name, group.Slug,
	)}, nil
}

func (s *Service) DecideAccess(
	ctx context.Context,
	identity auth.Identity,
	target string,
	invitationID int64,
	approve bool,
	note string,
) (AccessInvitation, error) {
	switch target {
	case AccessTargetProject:
		return s.decideProjectInvitation(ctx, identity, invitationID, approve, note)
	case AccessTargetTeam:
		return s.decideTeamInvitation(ctx, identity, invitationID, approve, note)
	default:
		return AccessInvitation{}, ErrAccessInvalidTarget
	}
}

func (s *Service) decideProjectInvitation(
	ctx context.Context,
	identity auth.Identity,
	invitationID int64,
	approve bool,
	note string,
) (AccessInvitation, error) {
	row, err := s.store.GetProjectGroupInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return AccessInvitation{}, ErrAccessNotFound
	}
	if err != nil {
		return AccessInvitation{}, err
	}
	if row.Status != "pending" {
		return AccessInvitation{}, ErrAccessNotPending
	}
	if err := s.requireOwner(ctx, row.GroupID, identity); err != nil {
		return AccessInvitation{}, err
	}
	now := time.Now()
	if row.ExpiresAt < now.UnixMilli() {
		_, _ = s.store.RespondProjectGroupInvitation(ctx,
			store.RespondProjectGroupInvitationParams{
				Status:      "revoked",
				RespondedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
				ID:          invitationID,
			})
		return AccessInvitation{}, ErrAccessExpired
	}
	status := "declined"
	if approve {
		status = "accepted"
	}
	n, err := s.store.RespondProjectGroupInvitation(ctx,
		store.RespondProjectGroupInvitationParams{
			Status:             status,
			RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			RespondedByActorID: sql.NullInt64{Int64: int64(identity.ActorID), Valid: true},
			ID:                 invitationID,
		})
	if err != nil {
		return AccessInvitation{}, err
	}
	if n == 0 {
		return AccessInvitation{}, ErrAccessNotPending
	}
	project, err := s.Projects.Get(ctx, row.ProjectID)
	if err != nil {
		return AccessInvitation{}, err
	}
	group, err := s.requireGroup(ctx, row.GroupID)
	if err != nil {
		return AccessInvitation{}, err
	}
	if approve {
		if err := s.attachProject(
			ctx,
			row.ProjectID,
			row.GroupID,
			row.Role,
			identity.ActorID,
		); err != nil {
			return AccessInvitation{}, err
		}
	}
	s.writeProjectLog(ctx, row.ProjectID, identity.ActorID, "invitation", status,
		group.Name, status, "Group invitation "+status,
		map[string]any{"group_id": row.GroupID, "role": row.Role, "note": strings.TrimSpace(note)})
	s.writeLog(ctx, row.GroupID, identity.ActorID, "access", status, "pending", status,
		"Project invitation "+status,
		map[string]any{"project_id": row.ProjectID, "role": row.Role})
	out := projectInvitationFromRecord(row, project.Name, project.Slug, group.Name, group.Slug)
	out.Status = status
	out.RespondedAt = &now
	out.RespondedByActorID = identity.ActorID
	return out, nil
}

func (s *Service) decideTeamInvitation(
	ctx context.Context,
	identity auth.Identity,
	invitationID int64,
	approve bool,
	note string,
) (AccessInvitation, error) {
	row, err := s.store.GetTeamGroupInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return AccessInvitation{}, ErrAccessNotFound
	}
	if err != nil {
		return AccessInvitation{}, err
	}
	if row.Status != "pending" {
		return AccessInvitation{}, ErrAccessNotPending
	}
	if err := s.requireOwner(ctx, row.GroupID, identity); err != nil {
		return AccessInvitation{}, err
	}
	now := time.Now()
	if row.ExpiresAt < now.UnixMilli() {
		_, _ = s.store.RespondTeamGroupInvitation(ctx,
			store.RespondTeamGroupInvitationParams{
				Status:      "revoked",
				RespondedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
				ID:          invitationID,
			})
		return AccessInvitation{}, ErrAccessExpired
	}
	status := "declined"
	if approve {
		status = "accepted"
	}
	n, err := s.store.RespondTeamGroupInvitation(ctx,
		store.RespondTeamGroupInvitationParams{
			Status:             status,
			RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			RespondedByActorID: sql.NullInt64{Int64: int64(identity.ActorID), Valid: true},
			ID:                 invitationID,
		})
	if err != nil {
		return AccessInvitation{}, err
	}
	if n == 0 {
		return AccessInvitation{}, ErrAccessNotPending
	}
	teamName, nameErr := s.Teams.Name(ctx, row.TeamID)
	if nameErr != nil {
		return AccessInvitation{}, nameErr
	}
	group, err := s.requireGroup(ctx, row.GroupID)
	if err != nil {
		return AccessInvitation{}, err
	}
	if approve {
		if err := s.attachTeam(ctx, row.TeamID, row.GroupID, identity.ActorID); err != nil {
			return AccessInvitation{}, err
		}
	}
	s.writeTeamLog(ctx, row.TeamID, identity.ActorID, "invitation", status,
		group.Name, status, "Group invitation "+status,
		map[string]any{"group_id": row.GroupID, "note": strings.TrimSpace(note)})
	s.writeLog(ctx, row.GroupID, identity.ActorID, "access", status, "pending", status,
		"Team invitation "+status, map[string]any{"team_id": row.TeamID})
	out := teamInvitationFromRecord(row, teamName, "", group.Name, group.Slug)
	out.Status = status
	out.RespondedAt = &now
	out.RespondedByActorID = identity.ActorID
	return out, nil
}

func (s *Service) RevokeAccess(
	ctx context.Context,
	identity auth.Identity,
	target string,
	invitationID int64,
) error {
	now := time.Now().UnixMilli()
	switch target {
	case AccessTargetProject:
		row, err := s.store.GetProjectGroupInvitation(ctx, invitationID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccessNotFound
		}
		if err != nil {
			return err
		}
		if err := s.authorizeHost(ctx, identity, row.GroupID, func() error {
			_, err := s.Projects.RequireRole(
				ctx, row.ProjectID, identity.ActorID, identity.IsAdmin(), projects.RoleOwner,
			)
			return err
		}); err != nil {
			return err
		}
		if row.Status != "pending" {
			return ErrAccessNotPending
		}
		_, err = s.store.RespondProjectGroupInvitation(ctx,
			store.RespondProjectGroupInvitationParams{
				Status:             "revoked",
				RespondedAt:        sql.NullInt64{Int64: now, Valid: true},
				RespondedByActorID: sql.NullInt64{Int64: int64(identity.ActorID), Valid: true},
				ID:                 invitationID,
			})
		return err
	case AccessTargetTeam:
		row, err := s.store.GetTeamGroupInvitation(ctx, invitationID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccessNotFound
		}
		if err != nil {
			return err
		}
		if err := s.authorizeHost(ctx, identity, row.GroupID, func() error {
			owner, err := s.Teams.IsOwner(ctx, row.TeamID, identity.ActorID)
			if err != nil {
				return err
			}
			if !owner && !identity.IsAdmin() {
				return ErrAccessForbidden
			}
			return nil
		}); err != nil {
			return err
		}
		if row.Status != "pending" {
			return ErrAccessNotPending
		}
		_, err = s.store.RespondTeamGroupInvitation(ctx,
			store.RespondTeamGroupInvitationParams{
				Status:             "revoked",
				RespondedAt:        sql.NullInt64{Int64: now, Valid: true},
				RespondedByActorID: sql.NullInt64{Int64: int64(identity.ActorID), Valid: true},
				ID:                 invitationID,
			})
		return err
	default:
		return ErrAccessInvalidTarget
	}
}

func (s *Service) LeaveProject(
	ctx context.Context,
	identity auth.Identity,
	projectID string,
	groupID int64,
) error {
	project, err := s.Projects.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if err := s.authorizeHost(ctx, identity, groupID, func() error {
		_, err := s.Projects.RequireRole(
			ctx, project.ID, identity.ActorID, identity.IsAdmin(), projects.RoleOwner,
		)
		return err
	}); err != nil {
		return err
	}
	membership, err := s.store.GetProjectGroupMembership(
		ctx,
		store.GetProjectGroupMembershipParams{ProjectID: project.ID, GroupID: groupID},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAccessNotLinked
	}
	if err != nil {
		return err
	}
	n, err := s.store.DeleteProjectGroupMembership(
		ctx,
		store.DeleteProjectGroupMembershipParams{ProjectID: project.ID, GroupID: groupID},
	)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAccessNotLinked
	}
	group, _ := s.requireGroup(ctx, groupID)
	s.writeProjectLog(ctx, project.ID, identity.ActorID, "group", "removed", group.Name,
		"", "Removed group "+group.Name+" from the project",
		map[string]any{"group_id": groupID, "role": membership.Role})
	s.writeLog(ctx, groupID, identity.ActorID, "access", "removed", membership.Role,
		"", "Removed from project "+project.Name,
		map[string]any{"project_id": project.ID, "role": membership.Role})
	s.notifyAccessRemoved(ctx, identity, groupID, "project", project.ID, project.Name)
	return nil
}

func (s *Service) LeaveTeam(
	ctx context.Context,
	identity auth.Identity,
	teamID, groupID int64,
) error {
	if err := s.authorizeHost(ctx, identity, groupID, func() error {
		owner, err := s.Teams.IsOwner(ctx, teamID, identity.ActorID)
		if err != nil {
			return err
		}
		if !owner && !identity.IsAdmin() {
			return ErrAccessForbidden
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.store.GetTeamGroupMembership(
		ctx,
		store.GetTeamGroupMembershipParams{TeamID: teamID, GroupID: groupID},
	); errors.Is(err, sql.ErrNoRows) {
		return ErrAccessNotLinked
	} else if err != nil {
		return err
	}
	n, err := s.store.DeleteTeamGroupMembership(
		ctx,
		store.DeleteTeamGroupMembershipParams{TeamID: teamID, GroupID: groupID},
	)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAccessNotLinked
	}
	teamName, _ := s.Teams.Name(ctx, teamID)
	group, _ := s.requireGroup(ctx, groupID)
	s.writeTeamLog(ctx, teamID, identity.ActorID, "group", "removed", group.Name,
		"", "Removed group "+group.Name+" from the team",
		map[string]any{"group_id": groupID})
	s.writeLog(ctx, groupID, identity.ActorID, "access", "removed", "member", "",
		"Removed from team "+teamName, map[string]any{"team_id": teamID})
	s.notifyAccessRemoved(ctx, identity, groupID, "team", teamID, teamName)
	return nil
}

func (s *Service) PruneAccessInvitations(ctx context.Context, now time.Time) (int64, error) {
	projectN, err := s.store.ExpireProjectGroupInvitations(
		ctx,
		store.ExpireProjectGroupInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			Now:         now.UnixMilli(),
		},
	)
	if err != nil {
		return 0, err
	}
	teamN, err := s.store.ExpireTeamGroupInvitations(
		ctx,
		store.ExpireTeamGroupInvitationsParams{
			RespondedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			Now:         now.UnixMilli(),
		},
	)
	if err != nil {
		return 0, err
	}
	return projectN + teamN, nil
}

func (s *Service) projectAccess(ctx context.Context, groupID int64) ([]ProjectAccess, error) {
	rows, err := s.store.ListAgentGroupProjectAccess(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]ProjectAccess, len(rows))
	for i, row := range rows {
		out[i] = ProjectAccess{
			ProjectID:         row.ProjectID,
			ProjectName:       row.ProjectName,
			ProjectSlug:       row.ProjectSlug,
			Role:              row.Role,
			GrantedByActorID:  ids.ActorID(row.GrantedByActorID.Int64),
			GrantedByUsername: row.GrantedByUsername.String,
			GrantedByName:     row.GrantedByName.String,
			CreatedAt:         time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

func (s *Service) teamAccess(ctx context.Context, groupID int64) ([]TeamAccess, error) {
	rows, err := s.store.ListAgentGroupTeamAccess(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamAccess, len(rows))
	for i, row := range rows {
		out[i] = TeamAccess{
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			TeamDescription:   row.TeamDescription,
			MemberCount:       row.MemberCount,
			GrantedByActorID:  ids.ActorID(row.GrantedByActorID.Int64),
			GrantedByUsername: row.GrantedByUsername.String,
			GrantedByName:     row.GrantedByName.String,
			CreatedAt:         time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

func (s *Service) groupInvitations(ctx context.Context, groupID int64) ([]AccessInvitation, error) {
	projectRows, err := s.store.ListAgentGroupProjectInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	teamRows, err := s.store.ListAgentGroupTeamInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]AccessInvitation, 0, len(projectRows)+len(teamRows))
	for _, row := range projectRows {
		out = append(out, projectInvitationFromList(row))
	}
	for _, row := range teamRows {
		out = append(out, teamInvitationFromList(row))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func projectInvitationFromHostList(
	row store.ListProjectGroupInvitationsRow,
	projectName, projectSlug string,
) AccessInvitation {
	return AccessInvitation{
		ID:                  row.ID,
		TargetKind:          AccessTargetProject,
		ProjectID:           row.ProjectID,
		ProjectName:         projectName,
		ProjectSlug:         projectSlug,
		GroupID:             row.GroupID,
		GroupName:           row.GroupName,
		GroupSlug:           row.GroupSlug,
		Role:                row.Role,
		Status:              row.Status,
		Note:                row.Note,
		InvitedByActorID:    ids.ActorID(row.InvitedByActorID.Int64),
		InvitedByUsername:   row.InvitedByUsername.String,
		InvitedByName:       row.InvitedByName.String,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
		RespondedByActorID:  ids.ActorID(row.RespondedByActorID.Int64),
		RespondedByUsername: row.RespondedByUsername.String,
		RespondedByName:     row.RespondedByName.String,
	}
}

func teamInvitationFromHostList(
	row store.ListTeamGroupInvitationsRow,
	teamName string,
) AccessInvitation {
	return AccessInvitation{
		ID:                  row.ID,
		TargetKind:          AccessTargetTeam,
		TeamID:              row.TeamID,
		TeamName:            teamName,
		GroupID:             row.GroupID,
		GroupName:           row.GroupName,
		GroupSlug:           row.GroupSlug,
		Role:                teams.RoleMember,
		Status:              row.Status,
		Note:                row.Note,
		InvitedByActorID:    ids.ActorID(row.InvitedByActorID.Int64),
		InvitedByUsername:   row.InvitedByUsername.String,
		InvitedByName:       row.InvitedByName.String,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
		RespondedByActorID:  ids.ActorID(row.RespondedByActorID.Int64),
		RespondedByUsername: row.RespondedByUsername.String,
		RespondedByName:     row.RespondedByName.String,
	}
}

func projectInvitationFromList(row store.ListAgentGroupProjectInvitationsRow) AccessInvitation {
	return AccessInvitation{
		ID:                  row.ID,
		TargetKind:          AccessTargetProject,
		ProjectID:           row.ProjectID,
		ProjectName:         row.ProjectName,
		ProjectSlug:         row.ProjectSlug,
		GroupID:             row.GroupID,
		Role:                row.Role,
		Status:              row.Status,
		Note:                row.Note,
		InvitedByActorID:    ids.ActorID(row.InvitedByActorID.Int64),
		InvitedByUsername:   row.InvitedByUsername.String,
		InvitedByName:       row.InvitedByName.String,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
		RespondedByActorID:  ids.ActorID(row.RespondedByActorID.Int64),
		RespondedByUsername: row.RespondedByUsername.String,
		RespondedByName:     row.RespondedByName.String,
	}
}

func teamInvitationFromList(row store.ListAgentGroupTeamInvitationsRow) AccessInvitation {
	return AccessInvitation{
		ID:                  row.ID,
		TargetKind:          AccessTargetTeam,
		TeamID:              row.TeamID,
		TeamName:            row.TeamName,
		GroupID:             row.GroupID,
		Role:                teams.RoleMember,
		Status:              row.Status,
		Note:                row.Note,
		InvitedByActorID:    ids.ActorID(row.InvitedByActorID.Int64),
		InvitedByUsername:   row.InvitedByUsername.String,
		InvitedByName:       row.InvitedByName.String,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
		RespondedByActorID:  ids.ActorID(row.RespondedByActorID.Int64),
		RespondedByUsername: row.RespondedByUsername.String,
		RespondedByName:     row.RespondedByName.String,
	}
}

func projectInvitationFromRecord(
	row store.ProjectGroupInvitation,
	projectName, projectSlug, groupName, groupSlug string,
) AccessInvitation {
	return AccessInvitation{
		ID:                 row.ID,
		TargetKind:         AccessTargetProject,
		ProjectID:          row.ProjectID,
		ProjectName:        projectName,
		ProjectSlug:        projectSlug,
		GroupID:            row.GroupID,
		GroupName:          groupName,
		GroupSlug:          groupSlug,
		Role:               row.Role,
		Status:             row.Status,
		Note:               row.Note,
		InvitedByActorID:   ids.ActorID(row.InvitedByActorID.Int64),
		CreatedAt:          time.UnixMilli(row.CreatedAt),
		ExpiresAt:          time.UnixMilli(row.ExpiresAt),
		RespondedAt:        nullTime(row.RespondedAt),
		RespondedByActorID: ids.ActorID(row.RespondedByActorID.Int64),
	}
}

func teamInvitationFromRecord(
	row store.TeamGroupInvitation,
	teamName, teamDescription, groupName, groupSlug string,
) AccessInvitation {
	return AccessInvitation{
		ID:                 row.ID,
		TargetKind:         AccessTargetTeam,
		TeamID:             row.TeamID,
		TeamName:           teamName,
		GroupID:            row.GroupID,
		GroupName:          groupName,
		GroupSlug:          groupSlug,
		Role:               teams.RoleMember,
		Status:             row.Status,
		Note:               row.Note,
		InvitedByActorID:   ids.ActorID(row.InvitedByActorID.Int64),
		CreatedAt:          time.UnixMilli(row.CreatedAt),
		ExpiresAt:          time.UnixMilli(row.ExpiresAt),
		RespondedAt:        nullTime(row.RespondedAt),
		RespondedByActorID: ids.ActorID(row.RespondedByActorID.Int64),
	}
}

func (s *Service) attachProject(
	ctx context.Context,
	projectID string,
	groupID int64,
	role string,
	actorID ids.ActorID,
) error {
	now := time.Now().UnixMilli()
	return s.store.UpsertProjectGroupMembership(
		ctx,
		store.UpsertProjectGroupMembershipParams{
			ProjectID:        projectID,
			GroupID:          groupID,
			Role:             role,
			GrantedByActorID: sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	)
}

func (s *Service) attachTeam(
	ctx context.Context,
	teamID, groupID int64,
	actorID ids.ActorID,
) error {
	return s.store.UpsertTeamGroupMembership(
		ctx,
		store.UpsertTeamGroupMembershipParams{
			TeamID:           teamID,
			GroupID:          groupID,
			GrantedByActorID: sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
			CreatedAt:        time.Now().UnixMilli(),
		},
	)
}

func (s *Service) authorizeHost(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	hostCheck func() error,
) error {
	if identity.IsAdmin() {
		return nil
	}
	owner, err := s.isOwner(ctx, groupID, identity.ActorID)
	if err != nil {
		return err
	}
	if owner {
		return nil
	}
	return hostCheck()
}

func (s *Service) writeProjectLog(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	dataJSON := ""
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			dataJSON = string(raw)
		}
	}
	_ = s.store.InsertProjectLog(ctx, store.InsertProjectLogParams{
		ProjectID: projectID,
		At:        time.Now().UnixMilli(),
		ActorID:   sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		Kind:      kind,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		Message:   message,
		DataJson:  dataJSON,
	})
}

func (s *Service) writeTeamLog(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	dataJSON := ""
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			dataJSON = string(raw)
		}
	}
	_ = s.store.InsertTeamLog(ctx, store.InsertTeamLogParams{
		TeamID:   teamID,
		At:       time.Now().UnixMilli(),
		ActorID:  sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		Kind:     kind,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
		DataJson: dataJSON,
	})
}

func (s *Service) notifyAccessRemoved(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	targetKind string,
	targetID any,
	targetName string,
) {
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return
	}
	groupOwner, err := s.isOwner(ctx, groupID, identity.ActorID)
	if err != nil {
		return
	}
	if !groupOwner {

		s.notifyGroupOwners(ctx, groupID, notifications.Notification{
			Kind:  KindAccessRemoved,
			Title: group.Name + " lost access to " + targetName,
			Body: "A " + targetKind + " owner removed " + group.Name +
				" from the " + targetKind + " " + targetName + ".",
		})
		return
	}
	var owners []int64
	switch targetKind {
	case AccessTargetProject:
		owners, _ = s.store.ListProjectOwnerActorIDs(ctx, targetID.(string))
	case AccessTargetTeam:
		owners, _ = s.store.ListTeamOwnerActorIDs(ctx, targetID.(int64))
	}
	for _, owner := range owners {
		if ids.ActorID(owner) == identity.ActorID {
			continue
		}
		s.notifyActor(ctx, groupID, ids.ActorID(owner), notifications.Notification{
			Kind:  KindAccessRemoved,
			Title: group.Name + " left " + targetName,
			Body: "A group owner removed " + group.Name +
				" from the " + targetKind + " " + targetName + ".",
		})
	}
}

func (s *Service) notifyAccessExpansion(
	ctx context.Context,
	groupID int64,
	agentID ids.AgentID,
	actorID ids.ActorID,
) {
	group, err := s.requireGroup(ctx, groupID)
	if err != nil || s.Notifications == nil || s.Agents == nil {
		return
	}
	agent, err := s.Agents.Get(ctx, agentID)
	if err != nil {
		return
	}
	projectLinks, err := s.store.ListAgentGroupProjectAccess(ctx, groupID)
	if err != nil {
		return
	}
	for _, link := range projectLinks {
		owners, err := s.store.ListProjectOwnerActorIDs(ctx, link.ProjectID)
		if err != nil {
			continue
		}
		for _, owner := range owners {
			if ids.ActorID(owner) == actorID {
				continue
			}
			s.notifyActor(ctx, groupID, ids.ActorID(owner), notifications.Notification{
				Kind:  KindAccessMemberAdded,
				Title: agent.Username + " joined " + group.Name,
				Body: "The agent now reaches project " + link.ProjectName +
					" through the group. Review the group's access if that is unexpected.",
			})
		}
	}
	teamLinks, err := s.store.ListAgentGroupTeamAccess(ctx, groupID)
	if err != nil {
		return
	}
	for _, link := range teamLinks {
		owners, err := s.store.ListTeamOwnerActorIDs(ctx, link.TeamID)
		if err != nil {
			continue
		}
		for _, owner := range owners {
			if ids.ActorID(owner) == actorID {
				continue
			}
			s.notifyActor(ctx, groupID, ids.ActorID(owner), notifications.Notification{
				Kind:  KindAccessMemberAdded,
				Title: agent.Username + " joined " + group.Name,
				Body: "The agent is now a member of team " + link.TeamName +
					" through the group.",
			})
		}
	}
}

func (s *Service) notifyActor(
	ctx context.Context,
	groupID int64,
	actorID ids.ActorID,
	n notifications.Notification,
) {
	if actorID == 0 || s.Notifications == nil {
		return
	}
	n.RecipientActorID = actorID
	n.PromptScope = "group"
	n.PromptGroupID = groupID
	_ = s.Notifications.Create(ctx, n)
}

func roleWeight(role string) int64 {
	switch role {
	case projects.RoleOwner:
		return 3
	case projects.RoleMember:
		return 2
	case projects.RoleViewer:
		return 1
	default:
		return 0
	}
}

func roleFromWeight(weight int64) string {
	switch {
	case weight >= 3:
		return projects.RoleOwner
	case weight == 2:
		return projects.RoleMember
	default:
		return projects.RoleViewer
	}
}
