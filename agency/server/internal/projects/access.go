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
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// InviteTTL is the locked project-team invitation lifetime.
const InviteTTL = 7 * 24 * time.Hour

const (
	DefaultLogLimit = 100
	MaxLogLimit     = 500
)

var (
	ErrForbidden          = errors.New("project owner or administrator required")
	ErrNotMember          = errors.New("not a member of the project")
	ErrTeamNotFound       = errors.New("team not found")
	ErrTeamAlreadyLinked  = errors.New("team is already attached to the project")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExists   = errors.New("a pending invitation already exists")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrNotTeamOwner       = errors.New("only an owner of the team may respond")
	ErrActorNotFound      = errors.New("actor not found")
	ErrMemberNotFound     = errors.New("project member not found")
	ErrInvalidRole        = errors.New("project role must be owner, member or viewer")
	ErrTeamHasNoAccess    = errors.New("team is not attached to the project")
	ErrRoleBelowInherited = errors.New(
		"team already has a stronger inherited role on the project",
	)
	ErrLastOwner = errors.New("a project must keep at least one owner")
)

type TeamOwnerChecker interface {
	IsOwner(ctx context.Context, teamID int64, actorID ids.ActorID) (bool, error)
}

type Member struct {
	ActorID          ids.ActorID
	Kind             string
	Username         string
	Name             string
	Description      string
	Role             string
	GrantedByActorID ids.ActorID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TeamLink struct {
	TeamID                   int64
	TeamName                 string
	TeamDescription          string
	Role                     string
	MemberCount              int64
	CreatedAt                time.Time
	UpdatedAt                time.Time
	InheritedRole            string
	InheritedFromProjectID   string
	InheritedFromProjectName string
	InheritedFromProjectSlug string
}

type TeamProject struct {
	ProjectID                string
	ProjectName              string
	ProjectSlug              string
	Role                     string
	Inherited                bool
	InheritedFromProjectID   string
	InheritedFromProjectName string
	InheritedFromProjectSlug string
}

type TeamInvitation struct {
	ID                  int64
	ProjectID           string
	ProjectName         string
	ProjectSlug         string
	TeamID              int64
	TeamName            string
	TeamDescription     string
	Role                string
	Status              string
	Note                string
	InvitedByUsername   string
	RespondedByUsername string
	CreatedAt           time.Time
	ExpiresAt           time.Time
	RespondedAt         *time.Time
}

// InviteTeamResult is the outcome of requesting team access: a pending

type InviteTeamResult struct {
	Invitation *TeamInvitation
	Team       *TeamLink
}

type LogEntry struct {
	ID            int64
	ProjectID     string
	At            time.Time
	ActorID       ids.ActorID
	ActorKind     string
	ActorName     string
	ActorUsername string
	Kind          string
	Field         string
	OldValue      string
	NewValue      string
	Message       string
	DataJSON      string
}

func (s *Service) EffectiveRole(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) (string, bool, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return "", false, err
	}
	if isAdmin {
		return RoleOwner, true, nil
	}
	if actorID == 0 {
		return "", false, nil
	}
	weight, err := s.store.GetEffectiveProjectRole(ctx, p.ID, actorID)
	if err != nil {
		return "", false, err
	}
	if weight <= 0 {
		return "", false, nil
	}
	return roleFromWeight(weight), true, nil
}

func (s *Service) RequireRole(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
	min string,
) (string, error) {
	role, ok, err := s.EffectiveRole(ctx, projectID, actorID, isAdmin)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotMember
	}
	if isAdmin || roleWeight(role) >= roleWeight(min) {
		return role, nil
	}
	return role, ErrForbidden
}

func PermissionsForRole(role string) Permissions {
	switch role {
	case RoleOwner, RoleMember:
		return Permissions{CanListTasks: true, CanCreateTasks: true, CanUpdateTasks: true}
	case RoleViewer:
		return Permissions{CanListTasks: true}
	default:
		return Permissions{}
	}
}

func (s *Service) AddAgent(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (ProjectAgent, error) {
	return s.upsertAgentGrant(ctx, projectID, agentID, RoleViewer, true)
}

func (s *Service) UpdateAgentPermissions(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
	perms Permissions,
) (ProjectAgent, error) {
	role := RoleViewer
	if perms.CanCreateTasks || perms.CanUpdateTasks {
		role = RoleMember
	}
	return s.upsertAgentGrant(ctx, projectID, agentID, role, false)
}

func (s *Service) upsertAgentGrant(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
	role string,
	keepExisting bool,
) (ProjectAgent, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return ProjectAgent{}, err
	}
	agent, err := s.store.GetAgent(ctx, agentID)
	if err != nil {
		return ProjectAgent{}, err
	}
	if keepExisting {
		if _, err := s.store.GetProjectActorGrant(ctx, store.GetProjectActorGrantParams{
			ProjectID: p.ID,
			ActorID:   int64(agent.ActorID),
		}); err == nil {
			return s.agentMember(ctx, p.ID, agentID)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return ProjectAgent{}, err
		}
	}
	now := time.Now().UnixMilli()
	if err := s.store.UpsertProjectActorGrant(ctx, store.UpsertProjectActorGrantParams{
		ProjectID:        p.ID,
		ActorID:          int64(agent.ActorID),
		Role:             role,
		GrantedByActorID: nullActor(actingActor(ctx)),
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		return ProjectAgent{}, err
	}
	s.writeLog(ctx, p.ID, "member", "role", "", role,
		"Set "+agent.Username+"'s project role to "+role,
		map[string]any{"actor_id": agent.ActorID, "role": role})
	return s.agentMember(ctx, p.ID, agentID)
}

func (s *Service) Permissions(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (Permissions, bool, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return Permissions{}, false, err
	}
	agent, err := s.store.GetAgent(ctx, agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return Permissions{}, false, nil
	}
	if err != nil {
		return Permissions{}, false, err
	}
	weight, err := s.store.GetEffectiveProjectRole(ctx, p.ID, agent.ActorID)
	if err != nil {
		return Permissions{}, false, err
	}
	if weight <= 0 {
		return Permissions{}, false, nil
	}
	return PermissionsForRole(roleFromWeight(weight)), true, nil
}

func (s *Service) RemoveAgent(ctx context.Context, projectID string, agentID ids.AgentID) error {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return err
	}
	agent, err := s.store.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}
	count, err := s.store.ListTasksByProjectAgent(ctx, store.ListTasksByProjectAgentParams{
		ProjectID: sql.NullString{String: p.ID, Valid: true},
		AgentID:   int64(agentID),
	})
	if err != nil {
		return err
	}
	if len(count) > 0 {
		return ErrAgentHasTasks
	}
	if err := s.store.RemoveProjectActorGrant(ctx, store.RemoveProjectActorGrantParams{
		ProjectID: p.ID,
		ActorID:   int64(agent.ActorID),
	}); err != nil {
		return err
	}
	s.writeLog(ctx, p.ID, "member", "removed", agent.Username, "",
		"Removed "+agent.Username+" from the project", map[string]any{"actor_id": agent.ActorID})
	return nil
}

func (s *Service) IsMember(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (bool, error) {
	_, ok, err := s.Permissions(ctx, projectID, agentID)
	return ok, err
}

func (s *Service) ActorCanPrompt(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
) (bool, error) {
	role, ok, err := s.EffectiveRole(ctx, projectID, actorID, false)
	if err != nil {
		return false, err
	}
	return ok && roleWeight(role) >= roleWeight(RoleMember), nil
}

func (s *Service) ListAgents(ctx context.Context, projectID string) ([]ProjectAgent, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorIDs, err := s.store.ListProjectMemberActorIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	taskRows, err := s.store.ListTasksByProject(ctx, sql.NullString{String: p.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	counts := map[int64]int{}
	for _, row := range taskRows {
		if row.AssigneeActorID.Valid {
			counts[row.AssigneeActorID.Int64]++
		}
	}
	out := []ProjectAgent{}
	for _, actorID := range actorIDs {
		actor, err := s.store.GetActorByID(ctx, ids.ActorID(actorID))
		if err != nil || actor.Kind != "agent" {
			continue
		}
		agent, err := s.store.GetAgentByActor(ctx, ids.ActorID(actorID))
		if err != nil {
			continue
		}
		weight, err := s.store.GetEffectiveProjectRole(ctx, p.ID, ids.ActorID(actorID))
		if err != nil {
			return nil, err
		}
		out = append(out, ProjectAgent{
			AgentID:     ids.AgentID(agent.ID),
			TaskCount:   counts[actorID],
			Permissions: PermissionsForRole(roleFromWeight(weight)),
		})
	}
	return out, nil
}

func (s *Service) ListAgentMemberships(
	ctx context.Context,
	agentID ids.AgentID,
) ([]AgentProject, error) {
	rows, err := s.store.ListAgentProjectMemberships(ctx, agentID)
	if err != nil {
		return nil, err
	}
	out := make([]AgentProject, len(rows))
	for i, row := range rows {
		out[i] = AgentProject{
			Project: Project{
				ID:          row.ID,
				Name:        row.Name,
				Slug:        row.Slug,
				Prefix:      row.Prefix,
				Description: row.Description,
				ParentID:    row.ParentProjectID.String,
				CreatedAt:   time.UnixMilli(row.CreatedAt),
				UpdatedAt:   time.UnixMilli(row.UpdatedAt),
			},
			Permissions: PermissionsForRole(row.Role),
		}
		if row.ArchivedAt.Valid {
			out[i].Project.ArchivedAt = time.UnixMilli(row.ArchivedAt.Int64)
		}
	}
	return out, nil
}

func (s *Service) ListAgentProjects(ctx context.Context, agentID ids.AgentID) ([]Project, error) {
	rows, err := s.store.ListAgentProjects(ctx, agentID)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = toProject(row)
	}
	return out, nil
}

func (s *Service) agentMember(
	ctx context.Context,
	projectID string,
	agentID ids.AgentID,
) (ProjectAgent, error) {
	rows, err := s.store.ListTasksByProjectAgent(
		ctx,
		store.ListTasksByProjectAgentParams{
			ProjectID: sql.NullString{String: projectID, Valid: true},
			AgentID:   int64(agentID),
		},
	)
	if err != nil {
		return ProjectAgent{}, err
	}
	perms, _, err := s.Permissions(ctx, projectID, agentID)
	if err != nil {
		return ProjectAgent{}, err
	}
	return ProjectAgent{AgentID: agentID, TaskCount: len(rows), Permissions: perms}, nil
}

func (s *Service) LookupActor(ctx context.Context, username string) (ids.ActorID, error) {
	actor, err := s.store.GetActorByUsername(ctx, strings.TrimSpace(username))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrActorNotFound
	}
	if err != nil {
		return 0, err
	}
	return ids.ActorID(actor.ID), nil
}

func (s *Service) Members(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]Member, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer); err != nil {
		return nil, err
	}
	rows, err := s.store.ListProjectActorGrants(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(rows))
	for i, row := range rows {
		out[i] = Member{
			ActorID:          ids.ActorID(row.ActorID),
			Kind:             row.ActorKind,
			Username:         row.ActorUsername,
			Name:             row.ActorName,
			Description:      row.ActorDescription,
			Role:             row.Role,
			GrantedByActorID: ids.ActorID(row.GrantedByActorID.Int64),
			CreatedAt:        time.UnixMilli(row.CreatedAt),
			UpdatedAt:        time.UnixMilli(row.UpdatedAt),
		}
	}
	return out, nil
}

type LeadCandidate struct {
	ActorID  ids.ActorID
	Kind     string
	Username string
	Name     string

	PictureAttachmentID int64
}

func (s *Service) LeadCandidates(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]LeadCandidate, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer); err != nil {
		return nil, err
	}
	actorIDs, err := s.store.ListProjectMemberActorIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]LeadCandidate, 0, len(actorIDs))
	for _, raw := range actorIDs {
		candidateID := ids.ActorID(raw)
		weight, err := s.store.GetEffectiveProjectRole(ctx, p.ID, candidateID)
		if err != nil {
			return nil, err
		}

		if weight < int64(roleWeight(RoleMember)) {
			continue
		}
		actor, err := s.store.GetActorByID(ctx, candidateID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, LeadCandidate{
			ActorID:             ids.ActorID(actor.ID),
			Kind:                actor.Kind,
			Username:            actor.Username,
			Name:                actor.Name,
			PictureAttachmentID: actor.PictureAttachmentID.Int64,
		})
	}
	return out, nil
}

func (s *Service) AddMember(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	role string,
	caller ids.ActorID,
	isAdmin bool,
) (Member, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return Member{}, err
	}
	if role == "" {
		role = RoleMember
	}
	if !validRole(role) {
		return Member{}, ErrInvalidRole
	}
	if _, err := s.RequireRole(ctx, p.ID, caller, isAdmin, RoleOwner); err != nil {
		return Member{}, err
	}
	target, err := s.store.GetActorByID(ctx, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrActorNotFound
	}
	if err != nil {
		return Member{}, err
	}
	now := time.Now().UnixMilli()
	if err := s.store.UpsertProjectActorGrant(ctx, store.UpsertProjectActorGrantParams{
		ProjectID:        p.ID,
		ActorID:          int64(actorID),
		Role:             role,
		GrantedByActorID: nullActor(caller),
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		return Member{}, err
	}
	s.writeLog(ctx, p.ID, "member", "granted", "", target.Username,
		"Granted "+target.Username+" the "+role+" role",
		map[string]any{"actor_id": actorID, "role": role})
	return s.member(ctx, p.ID, actorID)
}

func (s *Service) SetMemberRole(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	role string,
	caller ids.ActorID,
	isAdmin bool,
) (Member, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return Member{}, err
	}
	if !validRole(role) {
		return Member{}, ErrInvalidRole
	}
	if _, err := s.RequireRole(ctx, p.ID, caller, isAdmin, RoleOwner); err != nil {
		return Member{}, err
	}
	existing, err := s.store.GetProjectActorGrant(ctx, store.GetProjectActorGrantParams{
		ProjectID: p.ID,
		ActorID:   int64(actorID),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrMemberNotFound
	}
	if err != nil {
		return Member{}, err
	}
	if existing.Role == RoleOwner && role != RoleOwner {
		if err := s.requireAnotherOwner(ctx, p.ID); err != nil {
			return Member{}, err
		}
	}
	if err := s.store.UpdateProjectActorGrantRole(
		ctx,
		store.UpdateProjectActorGrantRoleParams{
			Role:      role,
			UpdatedAt: time.Now().UnixMilli(),
			ProjectID: p.ID,
			ActorID:   int64(actorID),
		},
	); err != nil {
		return Member{}, err
	}
	target, _ := s.store.GetActorByID(ctx, actorID)
	s.writeLog(ctx, p.ID, "member", "role", existing.Role, role,
		"Set "+target.Username+"'s project role to "+role,
		map[string]any{"actor_id": actorID, "role": role})
	return s.member(ctx, p.ID, actorID)
}

func (s *Service) RemoveMember(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	caller ids.ActorID,
	isAdmin bool,
) error {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if _, err := s.RequireRole(ctx, p.ID, caller, isAdmin, RoleOwner); err != nil {
		return err
	}
	existing, err := s.store.GetProjectActorGrant(ctx, store.GetProjectActorGrantParams{
		ProjectID: p.ID,
		ActorID:   int64(actorID),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMemberNotFound
	}
	if err != nil {
		return err
	}
	if existing.Role == RoleOwner {
		if err := s.requireAnotherOwner(ctx, p.ID); err != nil {
			return err
		}
	}
	if err := s.store.RemoveProjectActorGrant(ctx, store.RemoveProjectActorGrantParams{
		ProjectID: p.ID,
		ActorID:   int64(actorID),
	}); err != nil {
		return err
	}
	target, _ := s.store.GetActorByID(ctx, actorID)
	s.writeLog(ctx, p.ID, "member", "removed", target.Username, "",
		"Removed "+target.Username+" from the project",
		map[string]any{"actor_id": actorID})
	return nil
}

func (s *Service) member(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
) (Member, error) {
	rows, err := s.store.ListProjectActorGrants(ctx, projectID)
	if err != nil {
		return Member{}, err
	}
	for _, row := range rows {
		if ids.ActorID(row.ActorID) == actorID {
			return Member{
				ActorID:          ids.ActorID(row.ActorID),
				Kind:             row.ActorKind,
				Username:         row.ActorUsername,
				Name:             row.ActorName,
				Description:      row.ActorDescription,
				Role:             row.Role,
				GrantedByActorID: ids.ActorID(row.GrantedByActorID.Int64),
				CreatedAt:        time.UnixMilli(row.CreatedAt),
				UpdatedAt:        time.UnixMilli(row.UpdatedAt),
			}, nil
		}
	}
	return Member{}, ErrMemberNotFound
}

func (s *Service) Teams(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]TeamLink, error) {
	links, _, err := s.teamLinks(ctx, projectID, actorID, isAdmin)
	return links, err
}

func (s *Service) InheritedTeams(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]TeamLink, error) {
	_, inherited, err := s.teamLinks(ctx, projectID, actorID, isAdmin)
	return inherited, err
}

func (s *Service) teamLinks(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]TeamLink, []TeamLink, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer); err != nil {
		return nil, nil, err
	}
	inherited, err := s.inheritedTeamLinks(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	inheritedByTeam := make(map[int64]TeamLink, len(inherited))
	for _, link := range inherited {
		inheritedByTeam[link.TeamID] = link
	}
	rows, err := s.store.ListProjectTeamMemberships(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	direct := make([]TeamLink, len(rows))
	for i, row := range rows {
		direct[i] = TeamLink{
			TeamID:          row.TeamID,
			TeamName:        row.TeamName,
			TeamDescription: row.TeamDescription,
			Role:            row.Role,
			MemberCount:     row.MemberCount,
			CreatedAt:       time.UnixMilli(row.CreatedAt),
			UpdatedAt:       time.UnixMilli(row.UpdatedAt),
		}
		if hint, ok := inheritedByTeam[row.TeamID]; ok {
			direct[i].InheritedRole = hint.Role
			direct[i].InheritedFromProjectID = hint.InheritedFromProjectID
			direct[i].InheritedFromProjectName = hint.InheritedFromProjectName
			direct[i].InheritedFromProjectSlug = hint.InheritedFromProjectSlug
		}
	}
	rest := make([]TeamLink, 0, len(inherited))
	for _, link := range inherited {
		if _, ok := findTeamLink(direct, link.TeamID); ok {
			continue
		}
		rest = append(rest, link)
	}
	return direct, rest, nil
}

func findTeamLink(links []TeamLink, teamID int64) (TeamLink, bool) {
	for _, link := range links {
		if link.TeamID == teamID {
			return link, true
		}
	}
	return TeamLink{}, false
}

func (s *Service) inheritedTeamLinks(
	ctx context.Context,
	projectID string,
) ([]TeamLink, error) {
	rows, err := s.store.ListInheritedProjectTeams(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamLink, len(rows))
	for i, row := range rows {
		out[i] = TeamLink{
			TeamID:                   row.TeamID,
			TeamName:                 row.TeamName,
			TeamDescription:          row.TeamDescription,
			Role:                     roleFromWeight(row.RoleWeight),
			MemberCount:              row.MemberCount,
			InheritedRole:            roleFromWeight(row.RoleWeight),
			InheritedFromProjectID:   row.SourceProjectID,
			InheritedFromProjectName: row.SourceProjectName,
			InheritedFromProjectSlug: row.SourceProjectSlug,
		}
	}
	return out, nil
}

func (s *Service) TeamRole(
	ctx context.Context,
	projectID string,
	teamID int64,
) (string, bool, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return "", false, err
	}
	weight := int64(0)
	if row, err := s.store.GetProjectTeamMembership(
		ctx,
		store.GetProjectTeamMembershipParams{ProjectID: p.ID, TeamID: teamID},
	); err == nil {
		weight = int64(roleWeight(row.Role))
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	inherited, err := s.inheritedTeamLinks(ctx, p.ID)
	if err != nil {
		return "", false, err
	}
	for _, link := range inherited {
		if link.TeamID == teamID && int64(roleWeight(link.Role)) > weight {
			weight = int64(roleWeight(link.Role))
		}
	}
	if weight <= 0 {
		return "", false, nil
	}
	return roleFromWeight(weight), true, nil
}

func (s *Service) TeamIDs(ctx context.Context, projectID string) ([]int64, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.store.ListProjectTeamIDsWithAccess(ctx, p.ID)
}

func (s *Service) SetTeamRole(
	ctx context.Context,
	projectID string,
	teamID int64,
	role string,
	caller ids.ActorID,
	isAdmin bool,
) (TeamLink, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return TeamLink{}, err
	}
	if !validRole(role) {
		return TeamLink{}, ErrInvalidRole
	}
	if _, err := s.RequireRole(ctx, p.ID, caller, isAdmin, RoleOwner); err != nil {
		return TeamLink{}, err
	}
	if inherited, ok, err := s.teamInheritedRole(ctx, p.ID, teamID); err != nil {
		return TeamLink{}, err
	} else if ok && roleWeight(role) < roleWeight(inherited) {
		return TeamLink{}, ErrRoleBelowInherited
	}
	if existing, err := s.store.GetProjectTeamMembership(
		ctx,
		store.GetProjectTeamMembershipParams{ProjectID: p.ID, TeamID: teamID},
	); err == nil && existing.Role == RoleOwner && role != RoleOwner {
		if err := s.requireAnotherOwner(ctx, p.ID); err != nil {
			return TeamLink{}, err
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return TeamLink{}, err
	}
	now := time.Now().UnixMilli()
	if err := s.store.UpsertProjectTeamMembership(
		ctx,
		store.UpsertProjectTeamMembershipParams{
			ProjectID:        p.ID,
			TeamID:           teamID,
			Role:             role,
			GrantedByActorID: nullActor(caller),
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	); err != nil {
		return TeamLink{}, err
	}
	team, _ := s.store.GetTeam(ctx, teamID)
	s.writeLog(ctx, p.ID, "team", "role", "", role,
		"Set "+team.Name+"'s project role to "+role,
		map[string]any{"team_id": teamID, "role": role})
	return s.teamLink(ctx, p.ID, teamID)
}

func (s *Service) RemoveTeam(
	ctx context.Context,
	projectID string,
	teamID int64,
	caller ids.ActorID,
	isAdmin bool,
) error {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if _, err := s.RequireRole(ctx, p.ID, caller, isAdmin, RoleOwner); err != nil {
		return err
	}
	existing, err := s.store.GetProjectTeamMembership(
		ctx,
		store.GetProjectTeamMembershipParams{ProjectID: p.ID, TeamID: teamID},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTeamHasNoAccess
	}
	if err != nil {
		return err
	}
	if existing.Role == RoleOwner {
		if err := s.requireAnotherOwner(ctx, p.ID); err != nil {
			return err
		}
	}
	if err := s.store.RemoveProjectTeamMembership(
		ctx,
		store.RemoveProjectTeamMembershipParams{ProjectID: p.ID, TeamID: teamID},
	); err != nil {
		return err
	}

	if err := s.store.DeleteTaskTeamsByProjectTeam(ctx, store.DeleteTaskTeamsByProjectTeamParams{
		ProjectID: sql.NullString{String: p.ID, Valid: true},
		TeamID:    teamID,
	}); err != nil {
		return err
	}
	team, _ := s.store.GetTeam(ctx, teamID)
	s.writeLog(ctx, p.ID, "team", "removed", team.Name, "",
		"Removed "+team.Name+" from the project", map[string]any{"team_id": teamID})
	return nil
}

func (s *Service) ProjectsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Project, error) {
	rows, err := s.store.ListProjectsForActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = toProject(row)
	}
	return out, nil
}

func (s *Service) TeamProjects(ctx context.Context, teamID int64) ([]TeamProject, error) {
	rows, err := s.store.ListProjectTeamMembershipsForTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamProject, len(rows))
	for i, row := range rows {
		out[i] = TeamProject{
			ProjectID:                row.ProjectID,
			ProjectName:              row.ProjectName,
			ProjectSlug:              row.ProjectSlug,
			Role:                     roleFromWeight(row.RoleWeight),
			Inherited:                row.SourceDepth > 0,
			InheritedFromProjectID:   row.SourceProjectID,
			InheritedFromProjectName: row.SourceProjectName,
			InheritedFromProjectSlug: row.SourceProjectSlug,
		}
	}
	return out, nil
}

func (s *Service) teamInheritedRole(
	ctx context.Context,
	projectID string,
	teamID int64,
) (string, bool, error) {
	links, err := s.inheritedTeamLinks(ctx, projectID)
	if err != nil {
		return "", false, err
	}
	for _, link := range links {
		if link.TeamID == teamID {
			return link.Role, true, nil
		}
	}
	return "", false, nil
}

func (s *Service) teamLink(
	ctx context.Context,
	projectID string,
	teamID int64,
) (TeamLink, error) {
	rows, err := s.store.ListProjectTeamMemberships(ctx, projectID)
	if err != nil {
		return TeamLink{}, err
	}
	for _, row := range rows {
		if row.TeamID == teamID {
			return TeamLink{
				TeamID:          row.TeamID,
				TeamName:        row.TeamName,
				TeamDescription: row.TeamDescription,
				Role:            row.Role,
				MemberCount:     row.MemberCount,
				CreatedAt:       time.UnixMilli(row.CreatedAt),
				UpdatedAt:       time.UnixMilli(row.UpdatedAt),
			}, nil
		}
	}
	return TeamLink{}, ErrTeamHasNoAccess
}

// InviteTeam grants a team access to a project. Owner only. A team that

// invitee team's owners approves (admins may override).
func (s *Service) InviteTeam(
	ctx context.Context,
	projectID string,
	teamID int64,
	role, note string,
	caller ids.ActorID,
	isAdmin bool,
) (InviteTeamResult, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return InviteTeamResult{}, err
	}
	if role == "" {
		role = RoleMember
	}
	if !validRole(role) {
		return InviteTeamResult{}, ErrInvalidRole
	}
	if _, err := s.RequireRole(ctx, p.ID, caller, isAdmin, RoleOwner); err != nil {
		return InviteTeamResult{}, err
	}
	team, err := s.store.GetTeam(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		return InviteTeamResult{}, ErrTeamNotFound
	}
	if err != nil {
		return InviteTeamResult{}, err
	}
	if _, err := s.store.GetProjectTeamMembership(
		ctx,
		store.GetProjectTeamMembershipParams{ProjectID: p.ID, TeamID: teamID},
	); err == nil {
		return InviteTeamResult{}, ErrTeamAlreadyLinked
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteTeamResult{}, err
	}
	inherited, inherits, err := s.teamInheritedRole(ctx, p.ID, teamID)
	if err != nil {
		return InviteTeamResult{}, err
	}
	if inherits && roleWeight(role) < roleWeight(inherited) {
		return InviteTeamResult{}, ErrRoleBelowInherited
	}
	if inherits && roleWeight(role) <= roleWeight(inherited) {
		now := time.Now().UnixMilli()
		if err := s.store.UpsertProjectTeamMembership(
			ctx,
			store.UpsertProjectTeamMembershipParams{
				ProjectID:        p.ID,
				TeamID:           teamID,
				Role:             role,
				GrantedByActorID: nullActor(caller),
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		); err != nil {
			return InviteTeamResult{}, err
		}
		s.writeLog(ctx, p.ID, "team", "attached", "", role,
			"Attached "+team.Name+" as "+role+" (already inherited "+inherited+")",
			map[string]any{"team_id": teamID, "role": role, "inherited_role": inherited})
		link, err := s.teamLink(ctx, p.ID, teamID)
		if err != nil {
			return InviteTeamResult{}, err
		}
		return InviteTeamResult{Team: &link}, nil
	}
	if _, err := s.store.GetPendingProjectTeamInvitation(
		ctx,
		store.GetPendingProjectTeamInvitationParams{ProjectID: p.ID, TeamID: teamID},
	); err == nil {
		return InviteTeamResult{}, ErrInvitationExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteTeamResult{}, err
	}

	now := time.Now()
	row, err := s.store.CreateProjectTeamInvitation(
		ctx,
		store.CreateProjectTeamInvitationParams{
			ProjectID:        p.ID,
			TeamID:           teamID,
			InvitedByActorID: nullActor(caller),
			Role:             role,
			Note:             strings.TrimSpace(note),
			CreatedAt:        now.UnixMilli(),
			ExpiresAt:        now.Add(InviteTTL).UnixMilli(),
		},
	)
	if err != nil {
		return InviteTeamResult{}, err
	}
	s.writeLog(ctx, p.ID, "invitation", "invited", "", team.Name,
		"Invited team "+team.Name+" with the "+role+" role",
		map[string]any{"team_id": teamID, "role": role})
	invitation := toTeamInvitation(row, p.Name, p.Slug, team.Name, team.Description, "", "")
	return InviteTeamResult{Invitation: &invitation}, nil
}

func (s *Service) Invitations(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]TeamInvitation, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer); err != nil {
		return nil, err
	}
	rows, err := s.store.ListProjectTeamInvitations(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamInvitation, len(rows))
	for i, row := range rows {
		out[i] = TeamInvitation{
			ID:                  row.ID,
			ProjectID:           row.ProjectID,
			TeamID:              row.TeamID,
			TeamName:            row.TeamName,
			TeamDescription:     row.TeamDescription,
			Role:                row.Role,
			Status:              row.Status,
			Note:                row.Note,
			InvitedByUsername:   row.InvitedByUsername.String,
			RespondedByUsername: row.RespondedByUsername.String,
			CreatedAt:           time.UnixMilli(row.CreatedAt),
			ExpiresAt:           time.UnixMilli(row.ExpiresAt),
			RespondedAt:         nullTime(row.RespondedAt),
		}
	}
	return out, nil
}

func (s *Service) PendingInvitations(
	ctx context.Context,
	actorID ids.ActorID,
) ([]TeamInvitation, error) {
	if actorID == 0 {
		return nil, nil
	}
	rows, err := s.store.ListPendingProjectInvitationsForTeamOwner(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamInvitation, len(rows))
	for i, row := range rows {
		out[i] = TeamInvitation{
			ID:                row.ID,
			ProjectID:         row.ProjectID,
			ProjectName:       row.ProjectName,
			ProjectSlug:       row.ProjectSlug,
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			Role:              row.Role,
			Status:            row.Status,
			Note:              row.Note,
			InvitedByUsername: row.InvitedByUsername.String,
			CreatedAt:         time.UnixMilli(row.CreatedAt),
			ExpiresAt:         time.UnixMilli(row.ExpiresAt),
		}
	}
	return out, nil
}

// invitee team (or an administrator) may answer it.
func (s *Service) Respond(
	ctx context.Context,
	invitationID int64,
	actorID ids.ActorID,
	isAdmin bool,
	approve bool,
) (TeamInvitation, error) {
	row, err := s.store.GetProjectTeamInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return TeamInvitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return TeamInvitation{}, err
	}
	if row.Status != "pending" {
		return TeamInvitation{}, ErrInvitationNotFound
	}
	now := time.Now()
	if row.ExpiresAt < now.UnixMilli() {
		_, _ = s.store.RespondProjectTeamInvitation(
			ctx,
			store.RespondProjectTeamInvitationParams{
				Status:             "revoked",
				RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
				RespondedByActorID: sql.NullInt64{},
				ID:                 invitationID,
			},
		)
		return TeamInvitation{}, ErrInvitationExpired
	}

	if !isAdmin {
		owner, err := s.teamOwner(ctx, row.TeamID, actorID)
		if err != nil {
			return TeamInvitation{}, err
		}
		if !owner {
			return TeamInvitation{}, ErrNotTeamOwner
		}
	}

	status := "declined"
	if approve {
		status = "accepted"
	}
	n, err := s.store.RespondProjectTeamInvitation(
		ctx,
		store.RespondProjectTeamInvitationParams{
			Status:             status,
			RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			RespondedByActorID: nullActor(actorID),
			ID:                 invitationID,
		},
	)
	if err != nil {
		return TeamInvitation{}, err
	}
	if n == 0 {
		return TeamInvitation{}, ErrInvitationNotFound
	}

	if approve {
		if err := s.store.UpsertProjectTeamMembership(
			ctx,
			store.UpsertProjectTeamMembershipParams{
				ProjectID:        row.ProjectID,
				TeamID:           row.TeamID,
				Role:             row.Role,
				GrantedByActorID: nullActor(actorID),
				CreatedAt:        now.UnixMilli(),
				UpdatedAt:        now.UnixMilli(),
			},
		); err != nil {
			return TeamInvitation{}, err
		}
	}

	project, _ := s.Get(ctx, row.ProjectID)
	team, _ := s.store.GetTeam(ctx, row.TeamID)
	action := "declined"
	message := team.Name + " declined the project invitation"
	if approve {
		action = "joined"
		message = team.Name + " joined the project as " + row.Role
	}
	s.writeLog(ctx, row.ProjectID, "invitation", action, team.Name, status, message,
		map[string]any{"team_id": row.TeamID, "role": row.Role})
	return toTeamInvitation(row, project.Name, project.Slug, team.Name, team.Description,
		"", actorLabel(ctx, s, actorID)), nil
}

// inviter or an admin may revoke it.
func (s *Service) RevokeInvitation(
	ctx context.Context,
	projectID string,
	invitationID int64,
	actorID ids.ActorID,
	isAdmin bool,
) error {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if _, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer); err != nil {
		return err
	}
	row, err := s.store.GetProjectTeamInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) || row.ProjectID != p.ID {
		return ErrInvitationNotFound
	}
	if err != nil {
		return err
	}
	if !isAdmin && row.InvitedByActorID.Int64 != int64(actorID) {
		if _, err := s.RequireRole(ctx, p.ID, actorID, false, RoleOwner); err != nil {
			return err
		}
	}
	n, err := s.store.RevokeProjectTeamInvitation(
		ctx,
		store.RevokeProjectTeamInvitationParams{
			RespondedAt:        sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			RespondedByActorID: nullActor(actorID),
			ID:                 invitationID,
		},
	)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvitationNotFound
	}
	s.writeLog(ctx, p.ID, "invitation", "revoked", "", "",
		"Revoked a project invitation", nil)
	return nil
}

func (s *Service) PruneInvitations(ctx context.Context, now time.Time) (int64, error) {
	teams, err := s.store.ExpireProjectTeamInvitations(ctx, now.UnixMilli())
	if err != nil {
		return teams, err
	}
	parents, err := s.store.ExpireProjectParentInvitations(ctx, now.UnixMilli())
	return teams + parents, err
}

func (s *Service) Logs(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
	limit int64,
	beforeID int64,
) ([]LogEntry, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	role, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer)
	if err != nil {
		return nil, err
	}
	full := isAdmin || role == RoleOwner
	if limit <= 0 {
		limit = DefaultLogLimit
	}
	if limit > MaxLogLimit {
		limit = MaxLogLimit
	}
	if beforeID <= 0 {
		beforeID = math.MaxInt64
	}
	rows, err := s.store.ListProjectLogs(ctx, store.ListProjectLogsParams{
		ProjectID: p.ID,
		BeforeID:  beforeID,
		MaxCount:  limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]LogEntry, 0, len(rows))
	for _, row := range rows {
		entry := LogEntry{
			ID:            row.ID,
			ProjectID:     row.ProjectID,
			At:            time.UnixMilli(row.At),
			ActorID:       ids.ActorID(row.ActorID.Int64),
			ActorKind:     row.ActorKind.String,
			ActorName:     row.ActorName.String,
			ActorUsername: row.ActorUsername.String,
			Kind:          row.Kind,
			Field:         row.Field,
			OldValue:      row.OldValue,
			NewValue:      row.NewValue,
			Message:       row.Message,
			DataJSON:      row.DataJson,
		}
		if !audit.ScopedVisible(entry.ActorID, actorID, full, entry.Kind) {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Service) teamOwner(ctx context.Context, teamID int64, actorID ids.ActorID) (bool, error) {
	if s.TeamOwners == nil {
		return false, nil
	}
	return s.TeamOwners.IsOwner(ctx, teamID, actorID)
}

func (s *Service) requireAnotherOwner(ctx context.Context, projectID string) error {
	total, err := s.ownerCount(ctx, projectID)
	if err != nil {
		return err
	}
	if total <= 1 {
		return ErrLastOwner
	}
	return nil
}

func (s *Service) ownerCount(ctx context.Context, projectID string) (int, error) {

	projectIDs := []string{projectID}
	ancestors, err := s.store.ListProjectAncestorIDs(ctx, projectID)
	if err != nil {
		return 0, err
	}
	projectIDs = append(projectIDs, ancestors...)
	actorOwners := make(map[int64]bool)
	teamOwners := make(map[int64]bool)
	for _, id := range projectIDs {
		grants, err := s.store.ListProjectActorGrants(ctx, id)
		if err != nil {
			return 0, err
		}
		for _, grant := range grants {
			if grant.Role == RoleOwner {
				actorOwners[grant.ActorID] = true
			}
		}
		teams, err := s.store.ListProjectTeamMemberships(ctx, id)
		if err != nil {
			return 0, err
		}
		for _, team := range teams {
			if team.Role == RoleOwner {
				teamOwners[team.TeamID] = true
			}
		}
	}
	return len(actorOwners) + len(teamOwners), nil
}

func (s *Service) WriteLog(
	ctx context.Context,
	projectID string,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	s.writeLog(ctx, projectID, kind, field, oldValue, newValue, message, data)
}

func (s *Service) writeLog(
	ctx context.Context,
	projectID string,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	params := store.InsertProjectLogParams{
		ProjectID: projectID,
		At:        time.Now().UnixMilli(),
		ActorID:   nullActor(actingActor(ctx)),
		Kind:      kind,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		Message:   message,
		DataJson:  "",
	}
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			params.DataJson = string(raw)
		}
	}
	if err := s.store.InsertProjectLog(ctx, params); err != nil {
		log.Printf("failed to write project log for %s: %v", projectID, err)
	}
}

func actingActor(ctx context.Context) ids.ActorID {
	if actorID, ok := auth.ActorID(ctx); ok {
		return actorID
	}
	return 0
}

func actorLabel(ctx context.Context, s *Service, actorID ids.ActorID) string {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return ""
	}
	return actor.Username
}

func roleWeight(role string) int {
	switch role {
	case RoleOwner:
		return 3
	case RoleMember:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

func roleFromWeight(weight int64) string {
	switch {
	case weight >= 3:
		return RoleOwner
	case weight == 2:
		return RoleMember
	default:
		return RoleViewer
	}
}

func validRole(role string) bool {
	return role == RoleOwner || role == RoleMember || role == RoleViewer
}

func nullActor(actorID ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0}
}

func toTeamInvitation(
	row store.ProjectTeamInvitation,
	projectName, projectSlug, teamName, teamDescription, invitedBy, respondedBy string,
) TeamInvitation {
	return TeamInvitation{
		ID:                  row.ID,
		ProjectID:           row.ProjectID,
		ProjectName:         projectName,
		ProjectSlug:         projectSlug,
		TeamID:              row.TeamID,
		TeamName:            teamName,
		TeamDescription:     teamDescription,
		Role:                row.Role,
		Status:              row.Status,
		Note:                row.Note,
		InvitedByUsername:   invitedBy,
		RespondedByUsername: respondedBy,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
	}
}

func nullTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := time.UnixMilli(value.Int64)
	return &t
}

func (s *Service) OwnerActorIDs(ctx context.Context, projectID string) ([]ids.ActorID, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorIDs, err := s.store.ListProjectMemberActorIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]ids.ActorID, 0, len(actorIDs))
	for _, raw := range actorIDs {
		actorID := ids.ActorID(raw)
		weight, err := s.store.GetEffectiveProjectRole(ctx, p.ID, actorID)
		if err != nil {
			return nil, err
		}
		if weight >= int64(roleWeight(RoleOwner)) {
			out = append(out, actorID)
		}
	}
	return out, nil
}

func (s *Service) LogPrompt(
	ctx context.Context,
	projectID, kind, field, oldValue, newValue, message string,
	data any,
) {
	s.writeLog(ctx, projectID, kind, field, oldValue, newValue, message, data)
}
