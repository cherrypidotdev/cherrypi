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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/notifications"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

const RequestTTL = 7 * 24 * time.Hour

const (
	DefaultLogLimit = 50
	MaxLogLimit     = 200
)

const (
	KindMemberRequested   = "agent_group_member_requested"
	KindMemberApproved    = "agent_group_member_approved"
	KindMemberDeclined    = "agent_group_member_declined"
	KindMemberRemoved     = "agent_group_member_removed"
	KindMemberLeft        = "agent_group_member_left"
	KindAccessInvited     = "agent_group_access_invited"
	KindAccessRemoved     = "agent_group_access_removed"
	KindAccessMemberAdded = "agent_group_access_member_added"
)

var (
	ErrNotFound    = errors.New("agent group not found")
	ErrForbidden   = errors.New("agent group owner or administrator required")
	ErrInvalidName = errors.New("group name must be 1-100 characters")
	ErrInvalidSlug = errors.New(
		"group slug must be 1-100 lowercase letters, digits or dashes",
	)
	ErrSlugInUse         = errors.New("group slug is already taken")
	ErrArchived          = errors.New("agent group is archived")
	ErrMemberExists      = errors.New("agent is already a group member")
	ErrMemberNotFound    = errors.New("agent is not a group member")
	ErrAgentNotFound     = errors.New("agent not found")
	ErrLastOwner         = errors.New("a group must keep at least one owner")
	ErrOwnerNotFound     = errors.New("owner not found")
	ErrOwnerIsNotUser    = errors.New("only users may own a group")
	ErrRequestExists     = errors.New("a pending membership request already exists")
	ErrRequestNotFound   = errors.New("membership request not found")
	ErrRequestNotPending = errors.New("membership request is not pending")
	ErrRequestExpired    = errors.New("membership request has expired")
	ErrRequestNotDecider = errors.New("an owner of the agent or an administrator is required")
)

type AgentService interface {
	Get(ctx context.Context, id ids.AgentID) (agents.Agent, error)
	GetByActor(ctx context.Context, actorID ids.ActorID) (agents.Agent, error)
	IsOwner(ctx context.Context, agentID ids.AgentID, actorID ids.ActorID) (bool, error)
	Owners(ctx context.Context, agentID ids.AgentID) ([]agents.Owner, error)
}

type ProjectService interface {
	Get(ctx context.Context, idOrSlug string) (projects.Project, error)
	RequireRole(
		ctx context.Context,
		projectID string,
		actorID ids.ActorID,
		isAdmin bool,
		min string,
	) (string, error)
}

type TeamService interface {
	Name(ctx context.Context, teamID int64) (string, error)
	Get(ctx context.Context, teamID int64, actorID ids.ActorID, isAdmin bool) (teams.Team, error)
	IsOwner(ctx context.Context, teamID int64, actorID ids.ActorID) (bool, error)
}

type Notifier interface {
	Create(ctx context.Context, notification notifications.Notification) error
}

type Service struct {
	store         store.Store
	Agents        AgentService
	Projects      ProjectService
	Teams         TeamService
	Notifications Notifier
}

func New(st store.Store) *Service { return &Service{store: st} }

type Group struct {
	ID                  int64
	Name                string
	Slug                string
	Description         string
	PictureAttachmentID int64
	ArchivedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	MemberCount         int64
}

type Member struct {
	GroupID             int64
	AgentID             ids.AgentID
	ActorID             ids.ActorID
	Username            string
	Name                string
	Description         string
	PictureAttachmentID int64
	LastSeenAt          *time.Time
	AddedByActorID      ids.ActorID
	AddedByUsername     string
	CreatedAt           time.Time
}

type Owner struct {
	ActorID             ids.ActorID
	Kind                string
	Username            string
	Name                string
	Description         string
	PictureAttachmentID int64
	CreatedAt           time.Time
}

type MemberRequest struct {
	ID                  int64
	GroupID             int64
	GroupName           string
	GroupSlug           string
	AgentID             ids.AgentID
	AgentUsername       string
	AgentName           string
	RequestedByActorID  ids.ActorID
	RequestedByUsername string
	RequestedByName     string
	Status              string
	Note                string
	CreatedAt           time.Time
	ExpiresAt           time.Time
	RespondedAt         *time.Time
	RespondedByActorID  ids.ActorID
	RespondedByUsername string
	RespondedByName     string

	CanRespond bool
}

type MembershipResult struct {
	Added   bool
	Request MemberRequest
}

type GroupProject struct {
	Project        projects.Project
	MemberCount    int64
	Prompt         prompts.Prompt
	Disabled       bool
	DisabledReason string
}

type GroupTeam struct {
	TeamID     int64
	Name       string
	ProjectIDs []string
	Prompt     prompts.Prompt
}

type LogEntry struct {
	ID            int64
	GroupID       int64
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
	name, slug, description string,
) (Group, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return Group{}, ErrInvalidName
	}
	slug, err := s.uniqueSlug(ctx, slugify(slugOrName(slug, name)), 0)
	if err != nil {
		return Group{}, err
	}
	now := time.Now()
	row, err := s.store.CreateAgentGroupWithOwner(ctx, store.CreateAgentGroupParams{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(description),
		CreatedAt:   now.UnixMilli(),
		UpdatedAt:   now.UnixMilli(),
	}, actorID)
	if err != nil {
		return Group{}, err
	}
	s.writeLog(ctx, row.ID, actorID, "created", "", "", "",
		"Created group "+row.Name, map[string]any{"slug": row.Slug})
	return toGroup(row, 1), nil
}

func (s *Service) List(ctx context.Context) ([]Group, error) {
	rows, err := s.store.ListAgentGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Group, len(rows))
	for i, row := range rows {
		out[i] = toGroupRow(row)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Group, error) {
	row, err := s.store.GetAgentGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	count, err := s.store.CountAgentGroupMembers(ctx, id)
	if err != nil {
		return Group{}, err
	}
	return toGroup(row, count), nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (Group, error) {
	row, err := s.store.GetAgentGroupBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	count, err := s.store.CountAgentGroupMembers(ctx, row.ID)
	if err != nil {
		return Group{}, err
	}
	return toGroup(row, count), nil
}

func (s *Service) ForAgent(ctx context.Context, agentID ids.AgentID) ([]Group, error) {
	rows, err := s.store.ListAgentGroupsForAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	out := make([]Group, len(rows))
	for i, row := range rows {
		out[i] = groupFrom(
			row.ID, row.Name, row.Slug, row.Description,
			row.PictureAttachmentID, row.ArchivedAt,
			row.CreatedAt, row.UpdatedAt, row.MemberCount,
		)
	}
	return out, nil
}

func (s *Service) Update(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	name, slug, description string,
) (Group, error) {
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	if err := s.requireOwner(ctx, groupID, identity); err != nil {
		return Group{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return Group{}, ErrInvalidName
	}
	slug, err = s.uniqueSlug(ctx, slugify(slugOrName(slug, name)), groupID)
	if err != nil {
		return Group{}, err
	}
	description = strings.TrimSpace(description)
	row, err := s.store.UpdateAgentGroup(ctx, store.UpdateAgentGroupParams{
		Name:        name,
		Slug:        slug,
		Description: description,
		UpdatedAt:   time.Now().UnixMilli(),
		ID:          groupID,
	})
	if err != nil {
		return Group{}, err
	}
	if group.Name != row.Name {
		s.writeLog(ctx, groupID, identity.ActorID, "updated", "name",
			group.Name, row.Name, "Renamed group to "+row.Name,
			map[string]string{"old": group.Name, "new": row.Name})
	}
	if group.Slug != row.Slug {
		s.writeLog(ctx, groupID, identity.ActorID, "updated", "slug",
			group.Slug, row.Slug, "Changed the group slug",
			map[string]string{"old": group.Slug, "new": row.Slug})
	}
	if group.Description != row.Description {
		s.writeLog(ctx, groupID, identity.ActorID, "updated", "description",
			group.Description, row.Description, "Updated the group description", nil)
	}
	count, err := s.store.CountAgentGroupMembers(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	return toGroup(row, count), nil
}

func (s *Service) SetArchived(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	archived bool,
) (Group, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return Group{}, err
	}
	if err := s.requireOwner(ctx, groupID, identity); err != nil {
		return Group{}, err
	}
	now := time.Now().UnixMilli()
	if archived {
		row, err := s.store.ArchiveAgentGroup(ctx, groupID, now, now)
		if err != nil {
			return Group{}, err
		}
		s.writeLog(ctx, groupID, identity.ActorID, "archived", "", "", "",
			"Archived the group and removed every membership", nil)
		return toGroup(row, 0), nil
	}
	row, err := s.store.SetAgentGroupArchived(ctx, store.SetAgentGroupArchivedParams{
		ArchivedAt: sql.NullInt64{},
		UpdatedAt:  now,
		ID:         groupID,
	})
	if err != nil {
		return Group{}, err
	}
	s.writeLog(ctx, groupID, identity.ActorID, "unarchived", "", "", "",
		"Un-archived the group (memberships are not restored)", nil)
	return toGroup(row, 0), nil
}

func (s *Service) Members(ctx context.Context, groupID int64) ([]Member, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListAgentGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(rows))
	for i, row := range rows {
		out[i] = Member{
			GroupID:             row.GroupID,
			AgentID:             ids.AgentID(row.AgentID),
			ActorID:             ids.ActorID(row.ActorID),
			Username:            row.AgentUsername,
			Name:                row.AgentName,
			Description:         row.AgentDescription,
			PictureAttachmentID: row.AgentPictureAttachmentID.Int64,
			LastSeenAt:          nullTime(row.LastSeenAt),
			AddedByActorID:      ids.ActorID(row.AddedByActorID.Int64),
			AddedByUsername:     row.AddedByUsername.String,
			CreatedAt:           time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

func (s *Service) AddMember(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	agentID ids.AgentID,
	note string,
) (MembershipResult, error) {
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return MembershipResult{}, err
	}
	if group.ArchivedAt != nil {
		return MembershipResult{}, ErrArchived
	}
	if err := s.requireOwner(ctx, groupID, identity); err != nil {
		return MembershipResult{}, err
	}
	if s.Agents == nil {
		return MembershipResult{}, ErrAgentNotFound
	}
	agent, err := s.Agents.Get(ctx, agentID)
	if errors.Is(err, agents.ErrAgentMissing) {
		return MembershipResult{}, ErrAgentNotFound
	}
	if err != nil {
		return MembershipResult{}, err
	}
	member, err := s.store.IsAgentGroupMember(ctx, store.IsAgentGroupMemberParams{
		GroupID: groupID,
		AgentID: int64(agentID),
	})
	if err != nil {
		return MembershipResult{}, err
	}
	if member {
		return MembershipResult{}, ErrMemberExists
	}

	now := time.Now()
	if identity.IsAdmin() {
		if err := s.addMemberRow(ctx, groupID, agentID, identity.ActorID, now); err != nil {
			return MembershipResult{}, err
		}
		return MembershipResult{Added: true}, nil
	}
	agentOwner, err := s.Agents.IsOwner(ctx, agentID, identity.ActorID)
	if err != nil {
		return MembershipResult{}, err
	}
	if agentOwner {
		if err := s.addMemberRow(ctx, groupID, agentID, identity.ActorID, now); err != nil {
			return MembershipResult{}, err
		}
		return MembershipResult{Added: true}, nil
	}

	if _, err := s.store.GetPendingAgentGroupMemberRequest(
		ctx,
		store.GetPendingAgentGroupMemberRequestParams{
			GroupID: groupID,
			AgentID: int64(agentID),
		},
	); err == nil {
		return MembershipResult{}, ErrRequestExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return MembershipResult{}, err
	}

	row, err := s.store.CreateAgentGroupMemberRequest(
		ctx,
		store.CreateAgentGroupMemberRequestParams{
			GroupID:            groupID,
			AgentID:            int64(agentID),
			RequestedByActorID: int64(identity.ActorID),
			Note:               strings.TrimSpace(note),
			CreatedAt:          now.UnixMilli(),
			ExpiresAt:          now.Add(RequestTTL).UnixMilli(),
		},
	)
	if err != nil {
		return MembershipResult{}, err
	}
	s.writeLog(ctx, groupID, identity.ActorID, "member_request", "requested",
		"", agent.Username, "Requested to add "+agent.Username,
		map[string]any{"agent_id": agentID, "request_id": row.ID})
	s.notifyAgentOwners(ctx, agentID, group, notifications.Notification{
		Kind:  KindMemberRequested,
		Title: "Membership request for " + agent.Username,
		Body: fmt.Sprintf(
			"%s wants to add %s to the %s group. An owner of the agent must approve it.",
			actorUsername(ctx, s.store, identity.ActorID),
			agent.Username,
			group.Name,
		),
	})
	request, err := s.request(ctx, row.ID, identity)
	if err != nil {
		return MembershipResult{}, err
	}
	return MembershipResult{Added: false, Request: request}, nil
}

func (s *Service) RemoveMember(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	agentID ids.AgentID,
) error {
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if s.Agents == nil {
		return ErrAgentNotFound
	}
	agent, err := s.Agents.Get(ctx, agentID)
	if errors.Is(err, agents.ErrAgentMissing) {
		return ErrAgentNotFound
	}
	if err != nil {
		return err
	}
	groupOwner, err := s.isOwner(ctx, groupID, identity.ActorID)
	if err != nil {
		return err
	}
	agentOwner := false
	if !identity.IsAdmin() && !groupOwner {
		agentOwner, err = s.Agents.IsOwner(ctx, agentID, identity.ActorID)
		if err != nil {
			return err
		}
		if !agentOwner {
			return ErrForbidden
		}
	}
	n, err := s.store.RemoveAgentGroupMember(ctx, store.RemoveAgentGroupMemberParams{
		GroupID: groupID,
		AgentID: int64(agentID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMemberNotFound
	}
	s.writeLog(ctx, groupID, identity.ActorID, "member", "removed",
		agent.Username, "", "Removed "+agent.Username+" from the group",
		map[string]any{"agent_id": agentID, "by_agent_owner": agentOwner && !groupOwner})
	if groupOwner || identity.IsAdmin() {

		s.notifyAgentOwners(ctx, agentID, group, notifications.Notification{
			Kind:  KindMemberRemoved,
			Title: agent.Username + " left the " + group.Name + " group",
			Body: fmt.Sprintf(
				"%s removed %s from the %s group. Its tasks stop receiving the group's prompts.",
				actorUsername(ctx, s.store, identity.ActorID),
				agent.Username,
				group.Name,
			),
		})
		return nil
	}

	s.notifyGroupOwners(ctx, groupID, notifications.Notification{
		Kind:  KindMemberLeft,
		Title: agent.Username + " left the " + group.Name + " group",
		Body: fmt.Sprintf("%s removed its agent %s from the %s group.",
			actorUsername(ctx, s.store, identity.ActorID), agent.Username, group.Name),
	})
	return nil
}

func (s *Service) MemberRequests(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	status string,
) ([]MemberRequest, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return nil, err
	}
	if status == "" {
		status = "pending"
	}
	rows, err := s.store.ListAgentGroupMemberRequests(
		ctx,
		store.ListAgentGroupMemberRequestsParams{GroupID: groupID, Status: status},
	)
	if err != nil {
		return nil, err
	}
	out := make([]MemberRequest, len(rows))
	for i, row := range rows {
		out[i] = s.toRequest(ctx, identity, requestRow{
			ID:                  row.ID,
			GroupID:             row.GroupID,
			GroupName:           row.GroupName,
			GroupSlug:           row.GroupSlug,
			AgentID:             row.AgentID,
			AgentUsername:       row.AgentUsername,
			AgentName:           row.AgentName,
			RequestedByActorID:  row.RequestedByActorID,
			RequestedByUsername: row.RequestedByUsername.String,
			RequestedByName:     row.RequestedByName.String,
			Status:              row.Status,
			Note:                row.Note,
			CreatedAt:           row.CreatedAt,
			ExpiresAt:           row.ExpiresAt,
			RespondedAt:         row.RespondedAt,
			RespondedByActorID:  row.RespondedByActorID.Int64,
			RespondedByUsername: row.RespondedByUsername.String,
			RespondedByName:     row.RespondedByName.String,
		})
	}
	return out, nil
}

func (s *Service) PendingRequestsForAgent(
	ctx context.Context,
	identity auth.Identity,
	agentID ids.AgentID,
) ([]MemberRequest, error) {
	rows, err := s.store.ListPendingAgentGroupMemberRequestsForAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	out := make([]MemberRequest, len(rows))
	for i, row := range rows {
		out[i] = s.toRequest(ctx, identity, requestRow{
			ID:                  row.ID,
			GroupID:             row.GroupID,
			GroupName:           row.GroupName,
			GroupSlug:           row.GroupSlug,
			AgentID:             row.AgentID,
			AgentUsername:       row.AgentUsername,
			AgentName:           row.AgentName,
			RequestedByActorID:  row.RequestedByActorID,
			RequestedByUsername: row.RequestedByUsername.String,
			RequestedByName:     row.RequestedByName.String,
			Status:              row.Status,
			Note:                row.Note,
			CreatedAt:           row.CreatedAt,
			ExpiresAt:           row.ExpiresAt,
			RespondedAt:         row.RespondedAt,
			RespondedByActorID:  row.RespondedByActorID.Int64,
			RespondedByUsername: row.RespondedByUsername.String,
			RespondedByName:     row.RespondedByName.String,
		})
	}
	return out, nil
}

func (s *Service) RespondRequest(
	ctx context.Context,
	identity auth.Identity,
	requestID int64,
	approve bool,
	note string,
) (MemberRequest, error) {
	row, err := s.store.GetAgentGroupMemberRequest(ctx, requestID)
	if errors.Is(err, sql.ErrNoRows) {
		return MemberRequest{}, ErrRequestNotFound
	}
	if err != nil {
		return MemberRequest{}, err
	}
	if row.Status != "pending" {
		return MemberRequest{}, ErrRequestNotPending
	}
	group, err := s.requireGroup(ctx, row.GroupID)
	if err != nil {
		return MemberRequest{}, err
	}
	if !identity.IsAdmin() {
		if s.Agents == nil {
			return MemberRequest{}, ErrRequestNotDecider
		}
		owner, err := s.Agents.IsOwner(ctx, ids.AgentID(row.AgentID), identity.ActorID)
		if err != nil {
			return MemberRequest{}, err
		}
		if !owner {
			return MemberRequest{}, ErrRequestNotDecider
		}
	}
	if time.Now().UnixMilli() > row.ExpiresAt {
		if _, err := s.store.ExpireAgentGroupMemberRequests(
			ctx,
			store.ExpireAgentGroupMemberRequestsParams{
				RespondedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
				Now:         time.Now().UnixMilli(),
			},
		); err != nil {
			return MemberRequest{}, err
		}
		return MemberRequest{}, ErrRequestExpired
	}

	status := "declined"
	if approve {
		status = "accepted"
	}
	n, err := s.store.RespondAgentGroupMemberRequest(
		ctx,
		store.RespondAgentGroupMemberRequestParams{
			Status:             status,
			RespondedAt:        sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			RespondedByActorID: sql.NullInt64{Int64: int64(identity.ActorID), Valid: true},
			ID:                 requestID,
		},
	)
	if err != nil {
		return MemberRequest{}, err
	}
	if n == 0 {
		return MemberRequest{}, ErrRequestNotPending
	}
	if approve {
		if err := s.addMemberRow(
			ctx,
			group.ID,
			ids.AgentID(row.AgentID),
			identity.ActorID,
			time.Now(),
		); err != nil {
			return MemberRequest{}, err
		}
	}
	s.writeLog(ctx, group.ID, identity.ActorID, "member_request", status,
		"pending", status, "Membership request "+status,
		map[string]any{"agent_id": row.AgentID, "request_id": requestID})
	s.notifyGroupOwners(ctx, group.ID, notifications.Notification{
		Kind:  memberDecisionKind(approve),
		Title: group.Name + " membership request " + status,
		Body: fmt.Sprintf("%s %s the request to add agent %d to %s.",
			actorUsername(ctx, s.store, identity.ActorID), status, row.AgentID, group.Name),
	})
	return s.request(ctx, requestID, identity)
}

func (s *Service) Owners(ctx context.Context, groupID int64) ([]Owner, error) {
	rows, err := s.store.ListAgentGroupOwners(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]Owner, len(rows))
	for i, row := range rows {
		out[i] = Owner{
			ActorID:             ids.ActorID(row.ActorID),
			Kind:                row.ActorKind,
			Username:            row.ActorUsername,
			Name:                row.ActorName,
			Description:         row.ActorDescription,
			PictureAttachmentID: row.ActorPictureAttachmentID.Int64,
			CreatedAt:           time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

func (s *Service) AddOwner(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	actorID ids.ActorID,
) (Owner, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return Owner{}, err
	}
	if err := s.requireOwner(ctx, groupID, identity); err != nil {
		return Owner{}, err
	}
	actor, err := s.store.GetActorByID(ctx, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return Owner{}, ErrOwnerNotFound
	}
	if err != nil {
		return Owner{}, err
	}
	if actor.Kind != "user" {
		return Owner{}, ErrOwnerIsNotUser
	}
	now := time.Now()
	if err := s.store.AddAgentGroupOwner(ctx, store.AddAgentGroupOwnerParams{
		GroupID:   groupID,
		ActorID:   int64(actorID),
		CreatedAt: now.UnixMilli(),
	}); err != nil {
		return Owner{}, err
	}
	s.writeLog(ctx, groupID, identity.ActorID, "owner", "added", "",
		actor.Username, "Added "+actor.Username+" as an owner", nil)
	return Owner{
		ActorID:             actorID,
		Kind:                actor.Kind,
		Username:            actor.Username,
		Name:                actor.Name,
		Description:         actor.Description,
		PictureAttachmentID: actor.PictureAttachmentID.Int64,
		CreatedAt:           now,
	}, nil
}

func (s *Service) RemoveOwner(
	ctx context.Context,
	identity auth.Identity,
	groupID int64,
	actorID ids.ActorID,
) error {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return err
	}
	if err := s.requireOwner(ctx, groupID, identity); err != nil {
		return err
	}
	count, err := s.store.CountAgentGroupOwners(ctx, groupID)
	if err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastOwner
	}
	n, err := s.store.RemoveAgentGroupOwner(ctx, store.RemoveAgentGroupOwnerParams{
		GroupID: groupID,
		ActorID: int64(actorID),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrOwnerNotFound
	}
	s.writeLog(ctx, groupID, identity.ActorID, "owner", "removed",
		actorUsername(ctx, s.store, actorID), "",
		"Removed an owner", map[string]any{"actor_id": actorID})
	return nil
}

func (s *Service) GroupProjects(ctx context.Context, groupID int64) ([]GroupProject, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListAgentGroupMemberProjects(ctx, groupID)
	if err != nil {
		return nil, err
	}
	stored, err := s.storedPrompts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]GroupProject, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		seen[row.ID] = true
		project := projects.Project{
			ID:          row.ID,
			Name:        row.Name,
			Slug:        row.Slug,
			Prefix:      row.Prefix,
			Description: row.Description,
		}
		if row.ArchivedAt.Valid {
			project.ArchivedAt = time.UnixMilli(row.ArchivedAt.Int64)
		}
		item := GroupProject{
			Project:     project,
			MemberCount: row.MemberCount,
			Prompt:      stored[promptKey(prompts.ScopeGroupProject, "", row.ID)],
			Disabled:    row.MemberCount == 0 || row.ArchivedAt.Valid,
		}
		if item.Disabled {
			item.DisabledReason = "no group member has access right now"
		}
		if row.ArchivedAt.Valid {
			item.DisabledReason = "the project is archived"
		}
		out = append(out, item)
	}

	for key, p := range stored {
		scope, teamID, projectID := splitPromptKey(key)
		if scope != prompts.ScopeGroupProject || seen[projectID] {
			continue
		}
		_ = teamID
		project, err := s.project(ctx, projectID)
		if err != nil {
			continue
		}
		out = append(out, GroupProject{
			Project:        project,
			Prompt:         p,
			Disabled:       true,
			DisabledReason: "no group member has access right now",
		})
	}
	sortProjects(out)
	return out, nil
}

func (s *Service) GroupTeams(ctx context.Context, groupID int64) ([]GroupTeam, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListAgentGroupMemberTeams(ctx, groupID)
	if err != nil {
		return nil, err
	}
	stored, err := s.storedPrompts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	byTeam := make(map[int64]*GroupTeam, len(rows))
	order := make([]int64, 0, len(rows))
	for _, row := range rows {
		item, ok := byTeam[row.TeamID]
		if !ok {
			item = &GroupTeam{TeamID: row.TeamID, Name: row.TeamName}
			byTeam[row.TeamID] = item
			order = append(order, row.TeamID)
		}
		item.ProjectIDs = append(item.ProjectIDs, row.ProjectID)
	}
	for key, p := range stored {
		scope, teamID, _ := splitPromptKey(key)
		if scope != prompts.ScopeGroupTeam || teamID == 0 {
			continue
		}
		item, ok := byTeam[teamID]
		if !ok {
			name := ""
			if s.Teams != nil {
				name, _ = s.Teams.Name(ctx, teamID)
			}
			item = &GroupTeam{TeamID: teamID, Name: name}
			byTeam[teamID] = item
			order = append(order, teamID)
		}
		item.Prompt = p
	}
	out := make([]GroupTeam, 0, len(order))
	for _, id := range order {
		out = append(out, *byTeam[id])
	}
	return out, nil
}

func (s *Service) Logs(
	ctx context.Context,
	groupID int64,
	limit int,
	beforeID int64,
) ([]LogEntry, error) {
	if _, err := s.requireGroup(ctx, groupID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = DefaultLogLimit
	}
	if limit > MaxLogLimit {
		limit = MaxLogLimit
	}
	if beforeID == 0 {
		beforeID = 1<<62 - 1
	}
	rows, err := s.store.ListAgentGroupLogs(ctx, store.ListAgentGroupLogsParams{
		GroupID:  groupID,
		BeforeID: beforeID,
		MaxCount: int64(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LogEntry, len(rows))
	for i, row := range rows {
		out[i] = LogEntry{
			ID:            row.ID,
			GroupID:       row.GroupID,
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
	}
	return out, nil
}

func (s *Service) IsOwner(ctx context.Context, groupID int64, actorID ids.ActorID) (bool, error) {
	return s.isOwner(ctx, groupID, actorID)
}

func (s *Service) IsMemberAgent(
	ctx context.Context,
	groupID int64,
	agentID ids.AgentID,
) (bool, error) {
	return s.store.IsAgentGroupMember(ctx, store.IsAgentGroupMemberParams{
		GroupID: groupID,
		AgentID: int64(agentID),
	})
}

func (s *Service) OwnerActorIDs(ctx context.Context, groupID int64) ([]ids.ActorID, error) {
	rows, err := s.store.ListAgentGroupOwners(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]ids.ActorID, len(rows))
	for i, row := range rows {
		out[i] = ids.ActorID(row.ActorID)
	}
	return out, nil
}

func (s *Service) Name(ctx context.Context, groupID int64) (string, error) {
	group, err := s.requireGroup(ctx, groupID)
	if err != nil {
		return "", err
	}
	return group.Name, nil
}

func (s *Service) LogPrompt(
	ctx context.Context,
	groupID int64,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	s.writeLog(ctx, groupID, audit.ActorID(ctx), kind, field,
		oldValue, newValue, message, data)
}

func (s *Service) GroupIDsForAgent(ctx context.Context, agentID ids.AgentID) ([]int64, error) {
	rows, err := s.store.ListAgentGroupsForAgent(ctx, int64(agentID))
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(rows))
	for i, row := range rows {
		out[i] = row.ID
	}
	return out, nil
}

func (s *Service) PruneRequests(ctx context.Context, now time.Time) (int64, error) {
	return s.store.ExpireAgentGroupMemberRequests(
		ctx,
		store.ExpireAgentGroupMemberRequestsParams{
			RespondedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			Now:         now.UnixMilli(),
		},
	)
}

func (s *Service) requireGroup(ctx context.Context, groupID int64) (Group, error) {
	row, err := s.store.GetAgentGroup(ctx, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	count, err := s.store.CountAgentGroupMembers(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	return toGroup(row, count), nil
}

func (s *Service) isOwner(ctx context.Context, groupID int64, actorID ids.ActorID) (bool, error) {
	if actorID == 0 {
		return false, nil
	}
	return s.store.IsAgentGroupOwner(ctx, store.IsAgentGroupOwnerParams{
		GroupID: groupID,
		ActorID: int64(actorID),
	})
}

func (s *Service) requireOwner(
	ctx context.Context,
	groupID int64,
	identity auth.Identity,
) error {
	if identity.IsAdmin() {
		return nil
	}
	owner, err := s.isOwner(ctx, groupID, identity.ActorID)
	if err != nil {
		return err
	}
	if !owner {
		return ErrForbidden
	}
	return nil
}

func (s *Service) addMemberRow(
	ctx context.Context,
	groupID int64,
	agentID ids.AgentID,
	actorID ids.ActorID,
	now time.Time,
) error {
	if err := s.store.AddAgentGroupMember(ctx, store.AddAgentGroupMemberParams{
		GroupID:        groupID,
		AgentID:        int64(agentID),
		AddedByActorID: sql.NullInt64{Int64: int64(actorID), Valid: actorID != 0},
		CreatedAt:      now.UnixMilli(),
	}); err != nil {
		return err
	}
	label := agentUsername(ctx, s.Agents, agentID)
	s.writeLog(ctx, groupID, actorID, "member", "added", "", label,
		"Added "+label+" to the group", map[string]any{"agent_id": agentID})
	s.notifyAccessExpansion(ctx, groupID, agentID, actorID)
	return nil
}

func agentUsername(ctx context.Context, agents AgentService, agentID ids.AgentID) string {
	if agents == nil {
		return "an agent"
	}
	agent, err := agents.Get(ctx, agentID)
	if err != nil || agent.Username == "" {
		return "an agent"
	}
	return agent.Username
}

type requestRow struct {
	ID                  int64
	GroupID             int64
	GroupName           string
	GroupSlug           string
	AgentID             int64
	AgentUsername       string
	AgentName           string
	RequestedByActorID  int64
	RequestedByUsername string
	RequestedByName     string
	Status              string
	Note                string
	CreatedAt           int64
	ExpiresAt           int64
	RespondedAt         sql.NullInt64
	RespondedByActorID  int64
	RespondedByUsername string
	RespondedByName     string
}

func (s *Service) request(
	ctx context.Context,
	requestID int64,
	identity auth.Identity,
) (MemberRequest, error) {
	row, err := s.store.GetAgentGroupMemberRequest(ctx, requestID)
	if errors.Is(err, sql.ErrNoRows) {
		return MemberRequest{}, ErrRequestNotFound
	}
	if err != nil {
		return MemberRequest{}, err
	}
	group, err := s.requireGroup(ctx, row.GroupID)
	if err != nil {
		return MemberRequest{}, err
	}
	agentName := ""
	agentUsername := ""
	if s.Agents != nil {
		if agent, err := s.Agents.Get(ctx, ids.AgentID(row.AgentID)); err == nil {
			agentUsername = agent.Username
			agentName = agent.DisplayName
		}
	}
	return s.toRequest(ctx, identity, requestRow{
		ID:                  row.ID,
		GroupID:             row.GroupID,
		GroupName:           group.Name,
		GroupSlug:           group.Slug,
		AgentID:             row.AgentID,
		AgentUsername:       agentUsername,
		AgentName:           agentName,
		RequestedByActorID:  row.RequestedByActorID,
		RequestedByUsername: actorUsername(ctx, s.store, ids.ActorID(row.RequestedByActorID)),
		Status:              row.Status,
		Note:                row.Note,
		CreatedAt:           row.CreatedAt,
		ExpiresAt:           row.ExpiresAt,
		RespondedAt:         row.RespondedAt,
		RespondedByActorID:  row.RespondedByActorID.Int64,
		RespondedByUsername: actorUsername(ctx, s.store, ids.ActorID(row.RespondedByActorID.Int64)),
	}), nil
}

func (s *Service) toRequest(
	ctx context.Context,
	identity auth.Identity,
	row requestRow,
) MemberRequest {
	out := MemberRequest{
		ID:                  row.ID,
		GroupID:             row.GroupID,
		GroupName:           row.GroupName,
		GroupSlug:           row.GroupSlug,
		AgentID:             ids.AgentID(row.AgentID),
		AgentUsername:       row.AgentUsername,
		AgentName:           row.AgentName,
		RequestedByActorID:  ids.ActorID(row.RequestedByActorID),
		RequestedByUsername: row.RequestedByUsername,
		RequestedByName:     row.RequestedByName,
		Status:              row.Status,
		Note:                row.Note,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
		RespondedByActorID:  ids.ActorID(row.RespondedByActorID),
		RespondedByUsername: row.RespondedByUsername,
		RespondedByName:     row.RespondedByName,
	}
	if row.Status == "pending" {
		if identity.IsAdmin() {
			out.CanRespond = true
		} else if s.Agents != nil {
			owner, err := s.Agents.IsOwner(ctx, ids.AgentID(row.AgentID), identity.ActorID)
			out.CanRespond = err == nil && owner
		}
	}
	return out
}

func (s *Service) notifyAgentOwners(
	ctx context.Context,
	agentID ids.AgentID,
	group Group,
	n notifications.Notification,
) {
	if s.Notifications == nil || s.Agents == nil {
		return
	}
	owners, err := s.Agents.Owners(ctx, agentID)
	if err != nil {
		return
	}
	for _, owner := range owners {
		n.RecipientActorID = owner.ActorID
		n.PromptScope = "group"
		n.PromptGroupID = group.ID
		if err := s.Notifications.Create(ctx, n); err != nil {
			continue
		}
	}
}

func (s *Service) notifyGroupOwners(
	ctx context.Context,
	groupID int64,
	n notifications.Notification,
) {
	if s.Notifications == nil {
		return
	}
	owners, err := s.OwnerActorIDs(ctx, groupID)
	if err != nil {
		return
	}
	for _, owner := range owners {
		n.RecipientActorID = owner
		n.PromptScope = "group"
		n.PromptGroupID = groupID
		if err := s.Notifications.Create(ctx, n); err != nil {
			continue
		}
	}
}

func (s *Service) writeLog(
	ctx context.Context,
	groupID int64,
	actorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if groupID == 0 {
		return
	}
	dataJSON := ""
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			dataJSON = string(raw)
		}
	}
	params := store.InsertAgentGroupLogParams{
		GroupID:  groupID,
		At:       time.Now().UnixMilli(),
		Kind:     kind,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
		DataJson: dataJSON,
	}
	if actorID != 0 {
		params.ActorID = sql.NullInt64{Int64: int64(actorID), Valid: true}
	}
	_ = s.store.InsertAgentGroupLog(ctx, params)
}

func (s *Service) storedPrompts(
	ctx context.Context,
	groupID int64,
) (map[string]prompts.Prompt, error) {
	rows, err := s.store.ListLatestGroupPrompts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]prompts.Prompt, len(rows))
	for _, row := range rows {
		scope := prompts.Scope(row.Scope)
		projectID := row.ProjectID.String
		teamID := row.TeamID.Int64
		out[promptKey(scope, teamKey(teamID), projectID)] = prompts.Prompt{
			ID:        row.ID,
			Content:   row.Content,
			Version:   row.Version,
			GroupID:   row.GroupID.Int64,
			UpdatedAt: time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

func (s *Service) project(ctx context.Context, projectID string) (projects.Project, error) {
	if s.Projects == nil || projectID == "" {
		return projects.Project{}, ErrNotFound
	}
	return s.Projects.Get(ctx, projectID)
}

func promptKey(scope prompts.Scope, teamID string, projectID string) string {
	return string(scope) + "|" + teamID + "|" + projectID
}

func splitPromptKey(key string) (prompts.Scope, int64, string) {
	parts := strings.SplitN(key, "|", 3)
	if len(parts) != 3 {
		return "", 0, ""
	}
	teamID, _ := strconv.ParseInt(parts[1], 10, 64)
	return prompts.Scope(parts[0]), teamID, parts[2]
}

func teamKey(teamID int64) string {
	if teamID == 0 {
		return ""
	}
	return strconv.FormatInt(teamID, 10)
}

func sortProjects(items []GroupProject) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && strings.ToLower(items[j].Project.Name) < strings.ToLower(items[j-1].Project.Name); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func toGroup(row store.AgentGroup, memberCount int64) Group {
	return Group{
		ID:                  row.ID,
		Name:                row.Name,
		Slug:                row.Slug,
		Description:         row.Description,
		PictureAttachmentID: row.PictureAttachmentID.Int64,
		ArchivedAt:          nullTime(row.ArchivedAt),
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		UpdatedAt:           time.UnixMilli(row.UpdatedAt),
		MemberCount:         memberCount,
	}
}

func toGroupRow(row store.ListAgentGroupsRow) Group {
	return groupFrom(
		row.ID, row.Name, row.Slug, row.Description,
		row.PictureAttachmentID, row.ArchivedAt,
		row.CreatedAt, row.UpdatedAt, row.MemberCount,
	)
}

func groupFrom(
	id int64, name, slug, description string,
	picture, archived sql.NullInt64,
	createdAt, updatedAt, memberCount int64,
) Group {
	return Group{
		ID:                  id,
		Name:                name,
		Slug:                slug,
		Description:         description,
		PictureAttachmentID: picture.Int64,
		ArchivedAt:          nullTime(archived),
		CreatedAt:           time.UnixMilli(createdAt),
		UpdatedAt:           time.UnixMilli(updatedAt),
		MemberCount:         memberCount,
	}
}

func nullTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := time.UnixMilli(value.Int64)
	return &t
}

func actorUsername(ctx context.Context, st store.Store, actorID ids.ActorID) string {
	if actorID == 0 {
		return "someone"
	}
	actor, err := st.GetActorByID(ctx, actorID)
	if err != nil {
		return "someone"
	}
	return actor.Username
}

func memberDecisionKind(approved bool) string {
	if approved {
		return KindMemberApproved
	}
	return KindMemberDeclined
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	slug := slugPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = strings.Trim(slug[:100], "-")
	}
	if slug == "" {
		slug = "group"
	}
	return slug
}

func slugOrName(slug, name string) string {
	if strings.TrimSpace(slug) != "" {
		return slug
	}
	return name
}

func (s *Service) uniqueSlug(ctx context.Context, candidate string, selfID int64) (string, error) {
	if candidate == "" || len(candidate) > 100 {
		return "", ErrInvalidSlug
	}
	base := candidate
	for n := 2; n < 1000; n++ {
		existing, err := s.store.GetAgentGroupBySlug(ctx, candidate)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		if existing.ID == selfID {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
	return "", ErrSlugInUse
}
