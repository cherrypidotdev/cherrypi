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

// Team REST surface. It is mounted behind the shared auth middleware, so the
// request already carries an authenticated identity; the team policy (member,
// owner, admin) lives in the service. The two org-wide governance switches are
// supplied by the caller so this package stays free of account dependencies.
package teams

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Prefix is the REST base for teams.
const Prefix = "/api/rest/teams"

// Governance is the org-wide policy the team surface needs.
type Governance struct {
	// MembersCanCreateTeams lets a plain member create a team.
	MembersCanCreateTeams bool
	// TeamDeletionAdminOnly restricts deletion to administrators.
	TeamDeletionAdminOnly bool
}

// GovernanceProvider reads the current governance settings.
type GovernanceProvider func(ctx context.Context) (Governance, error)

// Handler serves teams.
type Handler struct {
	Service    *Service
	Governance GovernanceProvider
	// Projects lists the projects a team is attached to. Optional; when nil the
	// team-projects route returns 503.
	Projects ProjectTeamService
}

// TeamProjectLink is one project a team has been granted access to. It stays a
// local type so this package does not import internal/projects (the project
// tests import teams).
type TeamProjectLink struct {
	ProjectID   string
	ProjectName string
	ProjectSlug string
	Role        string
}

// ProjectTeamService is the slice of internal/projects the team page needs: the
// projects a team has been granted access to.
type ProjectTeamService interface {
	TeamProjects(ctx context.Context, teamID int64) ([]TeamProjectLink, error)
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix), "/")
	switch {
	case path == "":
		h.teams(w, r, identity)
	case path == "invitations/pending":
		h.pendingInvitations(w, r, identity)
	case strings.HasPrefix(path, "invitations/"):
		h.respond(w, r, identity, strings.TrimPrefix(path, "invitations/"))
	default:
		h.team(w, r, identity, path)
	}
}

// --- Collections -----------------------------------------------------------

func (h *Handler) teams(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Service.List(r.Context(), identity.ActorID, identity.IsAdmin())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"teams": teamsJSON(list)})
	case http.MethodPost:
		gov, err := h.governance(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !identity.IsAdmin() && !gov.MembersCanCreateTeams {
			writeError(w, http.StatusForbidden, "creating teams is disabled for members")
			return
		}
		var in struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		team, err := h.Service.Create(r.Context(), identity.ActorID, in.Name, in.Description)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"team": teamJSON(team)})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) pendingInvitations(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	list, err := h.Service.PendingInvitations(r.Context(), identity.ActorID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": invitationsJSON(list)})
}

func (h *Handler) respond(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[1] != "respond" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	var in struct {
		Approve bool `json:"approve"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	invitation, err := h.Service.Respond(
		r.Context(), id, identity.ActorID, identity.IsAdmin(), in.Approve,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitation": invitationJSON(invitation)})
}

// --- One team --------------------------------------------------------------

func (h *Handler) team(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	path string,
) {
	parts := strings.Split(path, "/")
	teamID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || teamID == 0 {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}

	switch {
	case len(parts) == 1:
		h.oneTeam(w, r, identity, teamID)
	case len(parts) == 2 && parts[1] == "members":
		h.members(w, r, identity, teamID)
	case len(parts) == 3 && parts[1] == "members":
		h.member(w, r, identity, teamID, actorid.ParseActor(parts[2]))
	case len(parts) == 2 && parts[1] == "invitations":
		h.invitations(w, r, identity, teamID)
	case len(parts) == 3 && parts[1] == "invitations":
		h.invitation(w, r, identity, teamID, parts[2])
	case len(parts) == 2 && parts[1] == "logs":
		h.logs(w, r, identity, teamID)
	case len(parts) == 2 && parts[1] == "projects":
		h.projects(w, r, identity, teamID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) oneTeam(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
) {
	switch r.Method {
	case http.MethodGet:
		team, err := h.Service.Get(r.Context(), teamID, identity.ActorID, identity.IsAdmin())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"team": teamJSON(team)})
	case http.MethodPut:
		var in struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		team, err := h.Service.Update(
			r.Context(), teamID, identity.ActorID, identity.IsAdmin(),
			in.Name, in.Description,
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"team": teamJSON(team)})
	case http.MethodDelete:
		gov, err := h.governance(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !identity.IsAdmin() && gov.TeamDeletionAdminOnly {
			writeError(w, http.StatusForbidden, "deleting teams is admin-only")
			return
		}
		if err := h.Service.Delete(
			r.Context(),
			teamID,
			identity.ActorID,
			identity.IsAdmin(),
		); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) members(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	members, err := h.Service.Members(r.Context(), teamID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": membersJSON(members)})
}

func (h *Handler) member(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
	targetActorID ids.ActorID,
) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	if targetActorID == 0 {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	err := h.Service.RemoveMember(
		r.Context(), teamID, identity.ActorID, identity.IsAdmin(), targetActorID,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) invitations(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Service.Invitations(
			r.Context(),
			teamID,
			identity.ActorID,
			identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invitations": invitationsJSON(list)})
	case http.MethodPost:
		var in struct {
			Username string `json:"username"`
			Role     string `json:"role"`
			Note     string `json:"note"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		invitation, err := h.Service.Invite(
			r.Context(), teamID, identity.ActorID, identity.IsAdmin(),
			in.Username, in.Role, in.Note,
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"invitation": invitationJSON(invitation)})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) invitation(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
	idPart string,
) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	if err := h.Service.RevokeInvitation(
		r.Context(), teamID, id, identity.ActorID, identity.IsAdmin(),
	); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) logs(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := h.Service.Logs(
		r.Context(), teamID, identity.ActorID, identity.IsAdmin(), limit, before,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logsJSON(entries)})
}

// projects lists the projects a team has been granted access to. Any team
// member (or an admin) may read it; the team page uses it for its Projects
// tab.
func (h *Handler) projects(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	teamID int64,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.Projects == nil {
		writeError(w, http.StatusServiceUnavailable, "project listing unavailable")
		return
	}
	if _, err := h.Service.Get(
		r.Context(), teamID, identity.ActorID, identity.IsAdmin(),
	); err != nil {
		writeServiceError(w, err)
		return
	}
	list, err := h.Projects.TeamProjects(r.Context(), teamID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]map[string]any, len(list))
	for i, project := range list {
		out[i] = map[string]any{
			"id":   project.ProjectID,
			"name": project.ProjectName,
			"slug": project.ProjectSlug,
			"role": project.Role,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

// --- Helpers ---------------------------------------------------------------

func (h *Handler) governance(ctx context.Context) (Governance, error) {
	if h.Governance == nil {
		return Governance{}, nil
	}
	return h.Governance(ctx)
}

func teamsJSON(teams []Team) []map[string]any {
	out := make([]map[string]any, len(teams))
	for i, team := range teams {
		out[i] = teamJSON(team)
	}
	return out
}

func teamJSON(team Team) map[string]any {
	return map[string]any{
		"id":                  team.ID,
		"name":                team.Name,
		"description":         team.Description,
		"member_count":        team.MemberCount,
		"role":                team.Role,
		"created_by_actor_id": actorid.Format(team.CreatedByActorID),
		"created_at":          formatTime(team.CreatedAt),
		"updated_at":          formatTime(team.UpdatedAt),
	}
}

func membersJSON(members []Member) []map[string]any {
	out := make([]map[string]any, len(members))
	for i, member := range members {
		out[i] = map[string]any{
			"actor_id":    actorid.Format(member.ActorID),
			"kind":        member.Kind,
			"username":    member.Username,
			"name":        member.Name,
			"description": member.Description,
			"role":        member.Role,
			"joined_at":   formatTime(member.JoinedAt),
		}
	}
	return out
}

func invitationsJSON(invitations []Invitation) []map[string]any {
	out := make([]map[string]any, len(invitations))
	for i, invitation := range invitations {
		out[i] = invitationJSON(invitation)
	}
	return out
}

func invitationJSON(invitation Invitation) map[string]any {
	out := map[string]any{
		"id":                    invitation.ID,
		"team_id":               invitation.TeamID,
		"team_name":             invitation.TeamName,
		"team_description":      invitation.TeamDescription,
		"actor_id":              actorid.Format(invitation.ActorID),
		"kind":                  invitation.Kind,
		"username":              invitation.Username,
		"name":                  invitation.Name,
		"role":                  invitation.Role,
		"status":                invitation.Status,
		"note":                  invitation.Note,
		"invited_by_username":   invitation.InvitedByUsername,
		"responded_by_username": invitation.RespondedByUsername,
		"created_at":            formatTime(invitation.CreatedAt),
		"expires_at":            formatTime(invitation.ExpiresAt),
	}
	if invitation.RespondedAt != nil {
		out["responded_at"] = formatTime(*invitation.RespondedAt)
	}
	return out
}

func logsJSON(entries []LogEntry) []map[string]any {
	out := make([]map[string]any, len(entries))
	for i, entry := range entries {
		out[i] = map[string]any{
			"id":             entry.ID,
			"team_id":        entry.TeamID,
			"at":             formatTime(entry.At),
			"actor_id":       actorid.Format(entry.ActorID),
			"actor_kind":     entry.ActorKind,
			"actor_name":     entry.ActorName,
			"actor_username": entry.ActorUsername,
			"kind":           entry.Kind,
			"field":          entry.Field,
			"old_value":      entry.OldValue,
			"new_value":      entry.NewValue,
			"message":        entry.Message,
			"data_json":      entry.DataJSON,
		}
	}
	return out
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTeamNotFound),
		errors.Is(err, ErrMemberNotFound),
		errors.Is(err, ErrInvitationNotFound),
		errors.Is(err, ErrInviteeNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrNotMember),
		errors.Is(err, ErrAgentNeedsOwner):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrTeamNameInUse),
		errors.Is(err, ErrMemberExists),
		errors.Is(err, ErrInvitationExists),
		errors.Is(err, ErrInvalidTeamName),
		errors.Is(err, ErrInviteeSelf),
		errors.Is(err, ErrLastOwner),
		errors.Is(err, ErrTeamNotEmpty),
		errors.Is(err, ErrTeamOwnsProjects),
		errors.Is(err, ErrTeamOwnsTasks),
		errors.Is(err, ErrInvitationExpired):
		writeError(w, http.StatusConflict, err.Error())
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
