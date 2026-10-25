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

package agents

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

const Prefix = "/api/rest/agents"

type TeamService interface {
	MembershipsForActor(ctx context.Context, actorID ids.ActorID) ([]teams.Team, error)
	PendingInvitations(ctx context.Context, actorID ids.ActorID) ([]teams.Invitation, error)
	Respond(
		ctx context.Context,
		invitationID int64,
		actorID ids.ActorID,
		isAdmin, approve bool,
	) (teams.Invitation, error)
	RemoveMember(
		ctx context.Context,
		teamID int64,
		actorID ids.ActorID,
		isAdmin bool,
		targetActorID ids.ActorID,
	) error
}

type Handler struct {
	Registry *Registry

	Teams TeamService

	Audit *audit.Service
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, Prefix)
	path = strings.Trim(path, "/")

	if path == "owned" {
		h.ownedAgents(w, r)
		return
	}
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	agentID := actorid.ParseAgent(parts[0])
	if agentID == 0 {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	switch parts[1] {
	case "owners":
		switch {
		case len(parts) == 2:
			h.owners(w, r, identity, agentID)
		case len(parts) == 3:
			h.owner(w, r, identity, agentID, actorid.ParseActor(parts[2]))
		default:
			writeError(w, http.StatusNotFound, "not found")
		}
	case "teams":
		h.agentTeams(w, r, identity, agentID, parts)
	case "invitations":
		h.agentInvitations(w, r, identity, agentID, parts)
	case "logs":
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.logs(w, r, identity, agentID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) ownedAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	owned, err := h.Registry.OwnerActorIDs(r.Context(), identity.ActorID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	out := make([]string, len(owned))
	for i, actorID := range owned {
		out[i] = actorid.Format(actorID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"actor_ids": out})
}

func (h *Handler) owners(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	agentID ids.AgentID,
) {
	switch r.Method {
	case http.MethodGet:
		owners, err := h.Registry.Owners(r.Context(), agentID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"owners": ownersJSON(owners)})
	case http.MethodPost:
		if !h.canManage(w, r, identity, agentID) {
			return
		}
		var in struct {
			Username string `json:"username"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		owner, err := h.Registry.AddOwnerByUsername(r.Context(), agentID, in.Username)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"owner": ownerJSON(owner)})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) owner(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	agentID ids.AgentID,
	ownerActorID ids.ActorID,
) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	if !h.canManage(w, r, identity, agentID) {
		return
	}
	if ownerActorID == 0 {
		writeError(w, http.StatusNotFound, "owner not found")
		return
	}
	if err := h.Registry.RemoveOwner(r.Context(), agentID, ownerActorID); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) canManage(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	agentID ids.AgentID,
) bool {
	if identity.IsAdmin() {
		return true
	}
	ok, err := h.Registry.IsOwner(r.Context(), agentID, identity.ActorID)
	if err != nil {
		writeStoreError(w, err)
		return false
	}
	if !ok {
		writeError(w, http.StatusForbidden, "agent owner or administrator required")
		return false
	}
	return true
}

func (h *Handler) logs(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	agentID ids.AgentID,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.Audit == nil {
		writeError(w, http.StatusServiceUnavailable, "logs are unavailable")
		return
	}
	agent, err := h.Registry.Get(r.Context(), agentID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !h.canManage(w, r, identity, agentID) {
		return
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := h.Audit.ListAgent(r.Context(), agent.ActorID, before, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"logs": audit.EntriesJSON(entries),
	})
}

func ownersJSON(owners []Owner) []map[string]any {
	out := make([]map[string]any, len(owners))
	for i, owner := range owners {
		out[i] = ownerJSON(owner)
	}
	return out
}

func ownerJSON(owner Owner) map[string]any {
	return map[string]any{
		"actor_id": actorid.Format(owner.ActorID),
		"kind":     owner.Kind,
		"username": owner.Username,
		"name":     owner.Name,
		"added_at": owner.AddedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAgentMissing):
		writeError(w, http.StatusNotFound, "agent not found")
	case errors.Is(err, ErrOwnerMissing):
		writeError(w, http.StatusNotFound, "owner not found")
	case errors.Is(err, ErrOwnerUnknownActor):
		writeError(w, http.StatusNotFound, "actor not found")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (h *Handler) agentTeams(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	agentID ids.AgentID,
	parts []string,
) {
	if h.Teams == nil {
		writeError(w, http.StatusServiceUnavailable, "teams unavailable")
		return
	}
	if !h.canManage(w, r, identity, agentID) {
		return
	}
	agent, err := h.Registry.Get(r.Context(), agentID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	switch {
	case len(parts) == 2:
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		list, err := h.Teams.MembershipsForActor(r.Context(), agent.ActorID)
		if err != nil {
			writeTeamError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"teams": agentTeamsJSON(list)})
	case len(parts) == 3:
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		teamID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || teamID == 0 {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		if err := h.Teams.RemoveMember(
			r.Context(), teamID, identity.ActorID, identity.IsAdmin(), agent.ActorID,
		); err != nil {
			writeTeamError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) agentInvitations(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	agentID ids.AgentID,
	parts []string,
) {
	if h.Teams == nil {
		writeError(w, http.StatusServiceUnavailable, "teams unavailable")
		return
	}
	if !h.canManage(w, r, identity, agentID) {
		return
	}
	agent, err := h.Registry.Get(r.Context(), agentID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	switch {
	case len(parts) == 2:
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		list, err := h.Teams.PendingInvitations(r.Context(), agent.ActorID)
		if err != nil {
			writeTeamError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invitations": agentInvitationsJSON(list)})
	case len(parts) == 4 && parts[3] == "respond":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		invitationID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || invitationID == 0 {
			writeError(w, http.StatusNotFound, "invitation not found")
			return
		}
		var in struct {
			Approve bool `json:"approve"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		invitation, err := h.Teams.Respond(
			r.Context(), invitationID, identity.ActorID, identity.IsAdmin(), in.Approve,
		)
		if err != nil {
			writeTeamError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invitation": agentInvitationJSON(invitation)})
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func agentTeamsJSON(list []teams.Team) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, team := range list {
		out[i] = map[string]any{
			"id":           team.ID,
			"name":         team.Name,
			"description":  team.Description,
			"member_count": team.MemberCount,
			"role":         team.Role,
			"created_at":   team.CreatedAt.UTC().Format(time.RFC3339),
			"updated_at":   team.UpdatedAt.UTC().Format(time.RFC3339),
		}
	}
	return out
}

func agentInvitationsJSON(list []teams.Invitation) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, invitation := range list {
		out[i] = agentInvitationJSON(invitation)
	}
	return out
}

func agentInvitationJSON(invitation teams.Invitation) map[string]any {
	out := map[string]any{
		"id":                  invitation.ID,
		"team_id":             invitation.TeamID,
		"team_name":           invitation.TeamName,
		"team_description":    invitation.TeamDescription,
		"actor_id":            actorid.Format(invitation.ActorID),
		"kind":                invitation.Kind,
		"username":            invitation.Username,
		"name":                invitation.Name,
		"role":                invitation.Role,
		"status":              invitation.Status,
		"note":                invitation.Note,
		"invited_by_username": invitation.InvitedByUsername,
		"created_at":          invitation.CreatedAt.UTC().Format(time.RFC3339),
		"expires_at":          invitation.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if invitation.RespondedAt != nil {
		out["responded_at"] = invitation.RespondedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func writeTeamError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, teams.ErrTeamNotFound),
		errors.Is(err, teams.ErrMemberNotFound),
		errors.Is(err, teams.ErrInvitationNotFound),
		errors.Is(err, teams.ErrInviteeNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, teams.ErrForbidden),
		errors.Is(err, teams.ErrNotMember),
		errors.Is(err, teams.ErrAgentNeedsOwner):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, teams.ErrLastOwner),
		errors.Is(err, teams.ErrTeamNotEmpty):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
