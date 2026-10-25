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

package teams

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	"cherrypi.vladfaust.com/agency/internal/store"
)

// Membership roles. A team's creator is its first owner; any member may invite,

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

// InviteTTL is the locked invitation lifetime.
const InviteTTL = 7 * 24 * time.Hour

const (
	DefaultLogLimit = 100
	MaxLogLimit     = 500
)

var (
	ErrTeamNotFound       = errors.New("team not found")
	ErrTeamNameInUse      = errors.New("team name is already taken")
	ErrInvalidTeamName    = errors.New("team name must be 1-100 characters")
	ErrForbidden          = errors.New("team owner or administrator required")
	ErrNotMember          = errors.New("not a member of the team")
	ErrMemberExists       = errors.New("actor is already a team member")
	ErrMemberNotFound     = errors.New("team member not found")
	ErrLastOwner          = errors.New("a team must keep at least one owner")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExists   = errors.New("a pending invitation already exists")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInviteeNotFound    = errors.New("actor not found")
	ErrInviteeSelf        = errors.New("cannot invite yourself")
	ErrAgentNeedsOwner    = errors.New("only an owner of the agent may respond")
	ErrTeamNotEmpty       = errors.New("a team with members cannot be deleted")
	ErrTeamOwnsProjects   = errors.New("a team that owns projects cannot be deleted")
	ErrTeamOwnsTasks      = errors.New("a team that owns tasks cannot be deleted")
)

type AgentOwnerChecker interface {
	IsActorOwner(ctx context.Context, agentActorID, actorID ids.ActorID) (bool, error)
}

// invitations still work, they just do not reach the invitee's inbox.
type Notifier interface {
	Create(ctx context.Context, notification notifications.Notification) error
}

type Service struct {
	store store.Store
	// AgentOwners resolves an agent-invitee's owners. Optional; without it no

	AgentOwners AgentOwnerChecker
	// Notifier records user invitations in the invitee's inbox. Optional.
	Notifier Notifier
}

func New(st store.Store) *Service {
	return &Service{store: st}
}

type Team struct {
	ID               int64
	Name             string
	Description      string
	MemberCount      int64
	Role             string
	CreatedByActorID ids.ActorID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Member struct {
	ActorID     ids.ActorID
	Kind        string
	Username    string
	Name        string
	Description string
	Role        string
	JoinedAt    time.Time
}

type Invitation struct {
	ID                  int64
	TeamID              int64
	TeamName            string
	TeamDescription     string
	ActorID             ids.ActorID
	Kind                string
	Username            string
	Name                string
	Role                string
	Status              string
	Note                string
	InvitedByUsername   string
	RespondedByUsername string
	CreatedAt           time.Time
	ExpiresAt           time.Time
	RespondedAt         *time.Time
}

type LogEntry struct {
	ID            int64
	TeamID        int64
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

func (s *Service) Create(
	ctx context.Context,
	actorID ids.ActorID,
	name, description string,
) (Team, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return Team{}, ErrInvalidTeamName
	}
	if err := s.ensureNameFree(ctx, name, 0); err != nil {
		return Team{}, err
	}

	now := time.Now()
	row, err := s.store.CreateTeamWithOwner(ctx, store.CreateTeamParams{
		Name:             name,
		Description:      strings.TrimSpace(description),
		CreatedByActorID: int64(actorID),
		CreatedAt:        now.UnixMilli(),
		UpdatedAt:        now.UnixMilli(),
	}, actorID)
	if err != nil {
		return Team{}, err
	}

	s.writeLog(ctx, row.ID, actorID, "created", "", "", "", "Created team "+row.Name, nil)
	return toTeam(row, 1, RoleOwner), nil
}

func (s *Service) List(ctx context.Context, actorID ids.ActorID, isAdmin bool) ([]Team, error) {
	if isAdmin {
		rows, err := s.store.ListTeams(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]Team, len(rows))
		for i, row := range rows {
			out[i] = Team{
				ID:               row.ID,
				Name:             row.Name,
				Description:      row.Description,
				MemberCount:      row.MemberCount,
				CreatedByActorID: ids.ActorID(row.CreatedByActorID),
				CreatedAt:        time.UnixMilli(row.CreatedAt),
				UpdatedAt:        time.UnixMilli(row.UpdatedAt),
			}

			member, err := s.store.GetTeamMember(ctx, store.GetTeamMemberParams{
				TeamID:  row.ID,
				ActorID: int64(actorID),
			})
			if err == nil {
				out[i].Role = member.Role
			}
		}
		return out, nil
	}

	rows, err := s.store.ListTeamsForActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]Team, len(rows))
	for i, row := range rows {
		out[i] = Team{
			ID:               row.ID,
			Name:             row.Name,
			Description:      row.Description,
			MemberCount:      row.MemberCount,
			Role:             row.MemberRole,
			CreatedByActorID: ids.ActorID(row.CreatedByActorID),
			CreatedAt:        time.UnixMilli(row.CreatedAt),
			UpdatedAt:        time.UnixMilli(row.UpdatedAt),
		}
	}
	return out, nil
}

func (s *Service) Get(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) (Team, error) {
	team, err := s.requireTeam(ctx, teamID)
	if err != nil {
		return Team{}, err
	}
	member, err := s.requireMember(ctx, teamID, actorID, isAdmin)
	if err != nil {
		return Team{}, err
	}
	count, err := s.store.CountTeamMembers(ctx, teamID)
	if err != nil {
		return Team{}, err
	}
	return toTeam(team, count, member.Role), nil
}

func (s *Service) Name(ctx context.Context, teamID int64) (string, error) {
	team, err := s.requireTeam(ctx, teamID)
	if err != nil {
		return "", err
	}
	return team.Name, nil
}

func (s *Service) Update(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
	name, description string,
) (Team, error) {
	team, err := s.requireTeam(ctx, teamID)
	if err != nil {
		return Team{}, err
	}
	if _, err := s.requireOwner(ctx, teamID, actorID, isAdmin); err != nil {
		return Team{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return Team{}, ErrInvalidTeamName
	}
	if err := s.ensureNameFree(ctx, name, teamID); err != nil {
		return Team{}, err
	}
	description = strings.TrimSpace(description)

	now := time.Now()
	updated, err := s.store.UpdateTeam(ctx, store.UpdateTeamParams{
		Name:        name,
		Description: description,
		UpdatedAt:   now.UnixMilli(),
		ID:          teamID,
	})
	if err != nil {
		return Team{}, err
	}
	if team.Name != updated.Name {
		s.writeLog(
			ctx,
			teamID,
			actorID,
			"settings",
			"name",
			team.Name,
			updated.Name,
			"Renamed team to "+updated.Name,
			map[string]string{"old": team.Name, "new": updated.Name},
		)
	}
	if team.Description != updated.Description {
		s.writeLog(ctx, teamID, actorID, "settings", "description", team.Description,
			updated.Description, "Updated the team description", nil)
	}

	count, err := s.store.CountTeamMembers(ctx, teamID)
	if err != nil {
		return Team{}, err
	}
	member, _ := s.member(ctx, teamID, actorID)
	return toTeam(updated, count, member.Role), nil
}

func (s *Service) Delete(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) error {
	if _, err := s.requireTeam(ctx, teamID); err != nil {
		return err
	}
	if _, err := s.requireOwner(ctx, teamID, actorID, isAdmin); err != nil {
		return err
	}
	count, err := s.store.CountTeamMembers(ctx, teamID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrTeamNotEmpty
	}
	owned, err := s.store.CountProjectsOwnedByTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if owned > 0 {
		return ErrTeamOwnsProjects
	}
	tasks, err := s.store.CountTasksOwnedByTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if tasks > 0 {
		return ErrTeamOwnsTasks
	}
	return s.store.DeleteTeam(ctx, teamID)
}

func (s *Service) IsOwner(ctx context.Context, teamID int64, actorID ids.ActorID) (bool, error) {
	if actorID == 0 {
		return false, nil
	}
	row, err := s.store.GetTeamMember(ctx, store.GetTeamMemberParams{
		TeamID:  teamID,
		ActorID: int64(actorID),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.Role == RoleOwner, nil
}

func (s *Service) Members(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) ([]Member, error) {
	if _, err := s.requireMember(ctx, teamID, actorID, isAdmin); err != nil {
		return nil, err
	}
	rows, err := s.store.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(rows))
	for i, row := range rows {
		out[i] = Member{
			ActorID:     ids.ActorID(row.ActorID),
			Kind:        row.ActorKind,
			Username:    row.ActorUsername,
			Name:        row.ActorName,
			Description: row.ActorDescription,
			Role:        row.Role,
			JoinedAt:    time.UnixMilli(row.JoinedAt),
		}
	}
	return out, nil
}

func (s *Service) RemoveMember(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
	targetActorID ids.ActorID,
) error {
	if _, err := s.requireTeam(ctx, teamID); err != nil {
		return err
	}
	if targetActorID != actorID {
		if _, err := s.requireOwner(ctx, teamID, actorID, isAdmin); err != nil {

			if !s.ownsAgent(ctx, targetActorID, actorID) {
				return err
			}
		}
	} else if !isAdmin {
		if _, err := s.requireMember(ctx, teamID, actorID, false); err != nil {
			return err
		}
	}

	target, err := s.store.GetTeamMember(ctx, store.GetTeamMemberParams{
		TeamID:  teamID,
		ActorID: int64(targetActorID),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMemberNotFound
	}
	if err != nil {
		return err
	}
	if target.Role == RoleOwner && !isAdmin {
		owners, err := s.store.CountTeamOwners(ctx, teamID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return ErrLastOwner
		}
	}

	n, err := s.store.RemoveTeamMember(ctx, store.RemoveTeamMemberParams{
		TeamID:  teamID,
		ActorID: int64(targetActorID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMemberNotFound
	}
	s.writeLog(ctx, teamID, actorID, "member", "removed", "",
		actorLabel(ctx, s, targetActorID), "Removed a team member",
		map[string]any{"actor_id": targetActorID})
	return nil
}

// Invite creates a pending invitation for an existing actor. Any member (or
// admin) may invite; only an owner or admin may invite as an owner.
func (s *Service) Invite(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
	username, role, note string,
) (Invitation, error) {
	team, err := s.requireTeam(ctx, teamID)
	if err != nil {
		return Invitation{}, err
	}
	caller, err := s.requireMember(ctx, teamID, actorID, isAdmin)
	if err != nil {
		return Invitation{}, err
	}
	if role != RoleOwner {
		role = RoleMember
	}
	if role == RoleOwner && caller.Role != RoleOwner && !isAdmin {
		return Invitation{}, ErrForbidden
	}

	invitee, err := s.store.GetActorByUsername(ctx, strings.TrimSpace(username))
	if errors.Is(err, sql.ErrNoRows) {
		return Invitation{}, ErrInviteeNotFound
	}
	if err != nil {
		return Invitation{}, err
	}
	inviteeID := ids.ActorID(invitee.ID)
	if inviteeID == actorID {
		return Invitation{}, ErrInviteeSelf
	}
	if _, err := s.store.GetTeamMember(ctx, store.GetTeamMemberParams{
		TeamID:  teamID,
		ActorID: int64(inviteeID),
	}); err == nil {
		return Invitation{}, ErrMemberExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Invitation{}, err
	}
	if _, err := s.store.GetPendingTeamInvitation(ctx, store.GetPendingTeamInvitationParams{
		TeamID:  teamID,
		ActorID: int64(inviteeID),
	}); err == nil {
		return Invitation{}, ErrInvitationExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Invitation{}, err
	}

	now := time.Now()
	row, err := s.store.CreateTeamInvitation(ctx, store.CreateTeamInvitationParams{
		TeamID:           teamID,
		ActorID:          int64(inviteeID),
		InvitedByActorID: sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		Role:             role,
		Note:             strings.TrimSpace(note),
		CreatedAt:        now.UnixMilli(),
		ExpiresAt:        now.Add(InviteTTL).UnixMilli(),
	})
	if err != nil {
		return Invitation{}, err
	}
	s.writeLog(ctx, teamID, actorID, "invitation", "invited", "", invitee.Username,
		"Invited "+invitee.Username+" to the team",
		map[string]any{"actor_id": inviteeID, "role": role})
	s.notifyInvitee(ctx, team, inviteeID, invitee.Kind, actorID, role, note)
	return toInvitation(row, team, invitee.Kind, invitee.Username, invitee.Name, "", ""), nil
}

// notifyInvitee puts a user's team invitation in their inbox, so someone who

func (s *Service) notifyInvitee(
	ctx context.Context,
	team store.Team,
	inviteeID ids.ActorID,
	inviteeKind string,
	inviterID ids.ActorID,
	role, note string,
) {
	if s.Notifier == nil || inviteeKind != "user" {
		return
	}
	body := "You have been invited to join " + team.Name + " as " + role + "."
	if inviter := actorLabel(ctx, s, inviterID); inviter != "" {
		body = inviter + " invited you to join " + team.Name + " as " + role + "."
	}
	if note = strings.TrimSpace(note); note != "" {
		body += " " + note
	}
	if err := s.Notifier.Create(ctx, notifications.Notification{
		RecipientActorID: inviteeID,
		Kind:             "team_invitation",
		Severity:         "info",
		Title:            "Team invitation: " + team.Name,
		Body:             body,
	}); err != nil {
		log.Printf("notify team invitation %d: %v", team.ID, err)
	}
}

func (s *Service) Invitations(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) ([]Invitation, error) {
	if _, err := s.requireMember(ctx, teamID, actorID, isAdmin); err != nil {
		return nil, err
	}
	rows, err := s.store.ListTeamInvitations(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, 0, len(rows))
	for _, row := range rows {
		if row.ActorKind == "agent" {
			continue
		}
		out = append(out, Invitation{
			ID:                  row.ID,
			TeamID:              row.TeamID,
			ActorID:             ids.ActorID(row.ActorID),
			Kind:                row.ActorKind,
			Username:            row.ActorUsername,
			Name:                row.ActorName,
			Role:                row.Role,
			Status:              row.Status,
			Note:                row.Note,
			InvitedByUsername:   row.InvitedByUsername.String,
			RespondedByUsername: row.RespondedByUsername.String,
			CreatedAt:           time.UnixMilli(row.CreatedAt),
			ExpiresAt:           time.UnixMilli(row.ExpiresAt),
			RespondedAt:         nullTime(row.RespondedAt),
		})
	}
	return out, nil
}

func (s *Service) PendingInvitations(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Invitation, error) {
	rows, err := s.store.ListPendingTeamInvitationsForActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, len(rows))
	for i, row := range rows {
		out[i] = Invitation{
			ID:                row.ID,
			TeamID:            row.TeamID,
			TeamName:          row.TeamName,
			TeamDescription:   row.TeamDescription,
			ActorID:           ids.ActorID(row.ActorID),
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

func (s *Service) MembershipsForActor(
	ctx context.Context,
	actorID ids.ActorID,
) ([]Team, error) {
	rows, err := s.store.ListTeamsForActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]Team, len(rows))
	for i, row := range rows {
		out[i] = Team{
			ID:               row.ID,
			Name:             row.Name,
			Description:      row.Description,
			MemberCount:      row.MemberCount,
			Role:             row.MemberRole,
			CreatedByActorID: ids.ActorID(row.CreatedByActorID),
			CreatedAt:        time.UnixMilli(row.CreatedAt),
			UpdatedAt:        time.UnixMilli(row.UpdatedAt),
		}
	}
	return out, nil
}

// override either. Approving adds the member with the invited role.
func (s *Service) Respond(
	ctx context.Context,
	invitationID int64,
	actorID ids.ActorID,
	isAdmin bool,
	approve bool,
) (Invitation, error) {
	row, err := s.store.GetTeamInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return Invitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return Invitation{}, err
	}
	if row.Status != "pending" {
		return Invitation{}, ErrInvitationNotFound
	}
	now := time.Now()
	if row.ExpiresAt < now.UnixMilli() {
		_, _ = s.store.RespondTeamInvitation(ctx, store.RespondTeamInvitationParams{
			Status:             "revoked",
			RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			RespondedByActorID: sql.NullInt64{},
			ID:                 invitationID,
		})
		return Invitation{}, ErrInvitationExpired
	}

	inviteeID := ids.ActorID(row.ActorID)
	invitee, err := s.store.GetActorByID(ctx, inviteeID)
	if err != nil {
		return Invitation{}, err
	}

	if !isAdmin {
		switch invitee.Kind {
		case "agent":
			if s.AgentOwners == nil {
				return Invitation{}, ErrAgentNeedsOwner
			}
			owner, err := s.AgentOwners.IsActorOwner(ctx, inviteeID, actorID)
			if err != nil {
				return Invitation{}, err
			}
			if !owner {
				return Invitation{}, ErrAgentNeedsOwner
			}
		default:
			if actorID != inviteeID {
				return Invitation{}, ErrForbidden
			}
		}
	}

	status := "declined"
	if approve {
		status = "accepted"
	}
	n, err := s.store.RespondTeamInvitation(ctx, store.RespondTeamInvitationParams{
		Status:             status,
		RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		RespondedByActorID: sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		ID:                 invitationID,
	})
	if err != nil {
		return Invitation{}, err
	}
	if n == 0 {
		return Invitation{}, ErrInvitationNotFound
	}

	if approve {
		if err := s.store.AddTeamMember(ctx, store.AddTeamMemberParams{
			TeamID:   row.TeamID,
			ActorID:  int64(inviteeID),
			Role:     row.Role,
			JoinedAt: now.UnixMilli(),
		}); err != nil {
			return Invitation{}, err
		}
	}

	team, _ := s.store.GetTeam(ctx, row.TeamID)
	action := "declined"
	message := invitee.Username + " declined the team invitation"
	if approve {
		action = "joined"
		message = invitee.Username + " joined the team"
	}
	s.writeLog(ctx, row.TeamID, actorID, "invitation", action,
		invitee.Username, status, message,
		map[string]any{"actor_id": inviteeID, "role": row.Role})
	return toInvitation(row, team, invitee.Kind, invitee.Username, invitee.Name,
		"", actorLabel(ctx, s, actorID)), nil
}

// RevokeInvitation withdraws a pending invitation. The inviter, an owner or an

func (s *Service) RevokeInvitation(
	ctx context.Context,
	teamID, invitationID int64,
	actorID ids.ActorID,
	isAdmin bool,
) error {
	if _, err := s.requireMember(ctx, teamID, actorID, isAdmin); err != nil {
		return err
	}
	row, err := s.store.GetTeamInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) || row.TeamID != teamID {
		return ErrInvitationNotFound
	}
	if err != nil {
		return err
	}
	if !isAdmin && row.InvitedByActorID.Int64 != int64(actorID) {
		if _, err := s.requireOwner(ctx, teamID, actorID, false); err != nil {
			return err
		}
	}
	n, err := s.store.RevokeTeamInvitation(ctx, store.RevokeTeamInvitationParams{
		RespondedAt:        sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		RespondedByActorID: sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		ID:                 invitationID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvitationNotFound
	}
	s.writeLog(
		ctx,
		teamID,
		actorID,
		"invitation",
		"revoked",
		"",
		"",
		"Revoked a team invitation",
		nil,
	)
	return nil
}

func (s *Service) PruneInvitations(ctx context.Context, now time.Time) (int64, error) {
	return s.store.ExpireTeamInvitations(ctx, now.UnixMilli())
}

func (s *Service) Logs(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
	limit int64,
	beforeID int64,
) ([]LogEntry, error) {
	membership, err := s.requireMember(ctx, teamID, actorID, isAdmin)
	if err != nil {
		return nil, err
	}
	full := isAdmin || membership.Role == RoleOwner
	if limit <= 0 {
		limit = DefaultLogLimit
	}
	if limit > MaxLogLimit {
		limit = MaxLogLimit
	}
	if beforeID <= 0 {
		beforeID = math.MaxInt64
	}
	rows, err := s.store.ListTeamLogs(ctx, store.ListTeamLogsParams{
		TeamID:   teamID,
		BeforeID: beforeID,
		MaxCount: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]LogEntry, 0, len(rows))
	for _, row := range rows {
		entry := LogEntry{
			ID:            row.ID,
			TeamID:        row.TeamID,
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

func (s *Service) requireTeam(ctx context.Context, teamID int64) (store.Team, error) {
	team, err := s.store.GetTeam(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Team{}, ErrTeamNotFound
	}
	return team, err
}

func (s *Service) member(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
) (store.TeamMember, bool) {
	if actorID == 0 {
		return store.TeamMember{}, false
	}
	row, err := s.store.GetTeamMember(ctx, store.GetTeamMemberParams{
		TeamID:  teamID,
		ActorID: int64(actorID),
	})
	if err != nil {
		return store.TeamMember{}, false
	}
	return row, true
}

func (s *Service) requireMember(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) (store.TeamMember, error) {
	if row, ok := s.member(ctx, teamID, actorID); ok {
		return row, nil
	}
	if isAdmin {
		return store.TeamMember{}, nil
	}
	return store.TeamMember{}, ErrNotMember
}

func (s *Service) requireOwner(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	isAdmin bool,
) (store.TeamMember, error) {
	if row, ok := s.member(ctx, teamID, actorID); ok && row.Role == RoleOwner {
		return row, nil
	}
	if isAdmin {
		return store.TeamMember{}, nil
	}
	return store.TeamMember{}, ErrForbidden
}

func (s *Service) ensureNameFree(ctx context.Context, name string, selfID int64) error {
	existing, err := s.store.GetTeamByName(ctx, name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return err
	case existing.ID == selfID:
		return nil
	default:
		return ErrTeamNameInUse
	}
}

func (s *Service) writeLog(
	ctx context.Context,
	teamID int64,
	actorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	params := store.InsertTeamLogParams{
		TeamID:   teamID,
		At:       time.Now().UnixMilli(),
		ActorID:  sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		Kind:     kind,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
		DataJson: "",
	}
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			params.DataJson = string(raw)
		}
	}
	if err := s.store.InsertTeamLog(ctx, params); err != nil {
		log.Printf("failed to write team log for team %d: %v", teamID, err)
	}
}

func (s *Service) ownsAgent(
	ctx context.Context,
	targetActorID, actorID ids.ActorID,
) bool {
	if s.AgentOwners == nil {
		return false
	}
	owner, err := s.AgentOwners.IsActorOwner(ctx, targetActorID, actorID)
	return err == nil && owner
}

func actorLabel(ctx context.Context, s *Service, actorID ids.ActorID) string {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return ""
	}
	return actor.Username
}

func toTeam(row store.Team, memberCount int64, role string) Team {
	return Team{
		ID:               row.ID,
		Name:             row.Name,
		Description:      row.Description,
		MemberCount:      memberCount,
		Role:             role,
		CreatedByActorID: ids.ActorID(row.CreatedByActorID),
		CreatedAt:        time.UnixMilli(row.CreatedAt),
		UpdatedAt:        time.UnixMilli(row.UpdatedAt),
	}
}

func toInvitation(
	row store.TeamInvitation,
	team store.Team,
	kind, username, name, invitedBy, respondedBy string,
) Invitation {
	return Invitation{
		ID:                  row.ID,
		TeamID:              row.TeamID,
		TeamName:            team.Name,
		TeamDescription:     team.Description,
		ActorID:             ids.ActorID(row.ActorID),
		Kind:                kind,
		Username:            username,
		Name:                name,
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
