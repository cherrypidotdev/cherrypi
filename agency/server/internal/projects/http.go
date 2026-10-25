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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
)

const AccessPrefix = "/api/rest/projects"

type Governance struct {
	ProjectDeletionAdminOnly bool
}

type GovernanceProvider func(ctx context.Context) (Governance, error)

type Handler struct {
	Service    *Service
	Governance GovernanceProvider
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, AccessPrefix), "/")
	switch {
	case path == "invitations/pending":
		h.pendingInvitations(w, r, identity)
	case strings.HasPrefix(path, "invitations/"):
		h.respond(w, r, identity, strings.TrimPrefix(path, "invitations/"))
	case strings.HasPrefix(path, "parent-invitations/"):
		h.parentInvitation(w, r, identity, strings.TrimPrefix(path, "parent-invitations/"))
	default:
		h.project(w, r, identity, path)
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

func (h *Handler) project(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	path string,
) {
	parts := strings.Split(path, "/")
	projectID := parts[0]
	if projectID == "" {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	switch {
	case len(parts) == 2 && parts[1] == "access":
		h.access(w, r, identity, projectID)
	case len(parts) == 2 && parts[1] == "members":
		h.members(w, r, identity, projectID)
	case len(parts) == 3 && parts[1] == "members":
		h.member(w, r, identity, projectID, parts[2])
	case len(parts) == 2 && parts[1] == "teams":
		h.teams(w, r, identity, projectID)
	case len(parts) == 3 && parts[1] == "teams":
		h.team(w, r, identity, projectID, parts[2])
	case len(parts) == 2 && parts[1] == "invitations":
		h.invitations(w, r, identity, projectID)
	case len(parts) == 3 && parts[1] == "invitations":
		h.invitation(w, r, identity, projectID, parts[2])
	case len(parts) == 2 && parts[1] == "logs":
		h.logs(w, r, identity, projectID)
	case len(parts) == 2 && parts[1] == "lead-candidates":
		h.leadCandidates(w, r, identity, projectID)
	case len(parts) == 2 && parts[1] == "hierarchy":
		h.hierarchy(w, r, identity, projectID)
	case len(parts) == 2 && parts[1] == "parent":
		h.parent(w, r, identity, projectID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) leadCandidates(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID string,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	list, err := h.Service.LeadCandidates(
		r.Context(),
		projectID,
		identity.ActorID,
		identity.IsAdmin(),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]map[string]any, len(list))
	for i, c := range list {
		entry := map[string]any{
			"actor_id": actorid.Format(c.ActorID),
			"kind":     c.Kind,
			"username": c.Username,
			"name":     c.Name,
		}
		if c.PictureAttachmentID > 0 {
			entry["picture_url"] = attachments.URLFor(c.PictureAttachmentID)
		}
		out[i] = entry
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": out})
}

func (h *Handler) hierarchy(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID string,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	tree, err := h.Service.Hierarchy(
		r.Context(),
		projectID,
		identity.ActorID,
		identity.IsAdmin(),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	referenceIDs, err := h.Service.ReferenceProjectIDs(r.Context(), tree.Project.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := hierarchyJSON(tree)
	out["reference_project_ids"] = referenceIDs
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) parent(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	childID string,
) {
	switch r.Method {
	case http.MethodPost:
		var in struct {
			ParentID string `json:"parent_id"`
			Note     string `json:"note"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if strings.TrimSpace(in.ParentID) == "" {
			writeError(w, http.StatusBadRequest, "parent_id is required")
			return
		}
		result, err := h.Service.InviteParent(
			r.Context(),
			childID,
			in.ParentID,
			in.Note,
			identity.ActorID,
			identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		payload := map[string]any{
			"project": projectRefJSON(result.Project),
			"direct":  result.Direct,
		}
		if result.Invitation != nil {
			payload["invitation"] = parentInvitationJSON(*result.Invitation)
		}
		writeJSON(w, http.StatusOK, payload)
	case http.MethodDelete:
		project, err := h.Service.DetachParent(
			r.Context(),
			childID,
			identity.ActorID,
			identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"project": projectRefJSON(project),
		})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) parentInvitation(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "respond" && r.Method == http.MethodPost:
		var in struct {
			Approve bool `json:"approve"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		invitation, err := h.Service.RespondParentInvitation(
			r.Context(),
			id,
			identity.ActorID,
			identity.IsAdmin(),
			in.Approve,
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"invitation": parentInvitationJSON(invitation),
		})
	case len(parts) == 1 && r.Method == http.MethodDelete:
		if err := h.Service.RevokeParentInvitation(
			r.Context(),
			id,
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

func (h *Handler) access(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID string,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx := r.Context()
	role, ok, err := h.Service.EffectiveRole(ctx, projectID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusForbidden, ErrNotMember.Error())
		return
	}
	members, err := h.Service.Members(ctx, projectID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	teams, err := h.Service.Teams(ctx, projectID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	invitations, err := h.Service.Invitations(ctx, projectID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	project, err := h.Service.Get(ctx, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	gov, _ := h.governance(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id":        project.ID,
		"role":              role,
		"members":           membersJSON(members),
		"teams":             teamsJSON(teams),
		"invitations":       invitationsJSON(invitations),
		"can_delete":        identity.IsAdmin() || !gov.ProjectDeletionAdminOnly,
		"can_manage_access": role == RoleOwner,
	})
}

func (h *Handler) members(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID string,
) {
	switch r.Method {
	case http.MethodGet:
		members, err := h.Service.Members(
			r.Context(),
			projectID,
			identity.ActorID,
			identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"members": membersJSON(members)})
	case http.MethodPost:
		var in struct {
			ActorID  string `json:"actor_id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		target := actorid.ParseActor(in.ActorID)
		if target == 0 && strings.TrimSpace(in.Username) != "" {
			resolved, err := h.Service.LookupActor(r.Context(), in.Username)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			target = resolved
		}
		member, err := h.Service.AddMember(
			r.Context(), projectID, target, in.Role,
			identity.ActorID, identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"member": memberJSON(member)})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) member(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID, actorPart string,
) {
	target := actorid.ParseActor(actorPart)
	if target == 0 {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var in struct {
			Role string `json:"role"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		member, err := h.Service.SetMemberRole(
			r.Context(), projectID, target, in.Role, identity.ActorID, identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"member": memberJSON(member)})
	case http.MethodDelete:
		if err := h.Service.RemoveMember(
			r.Context(), projectID, target, identity.ActorID, identity.IsAdmin(),
		); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) teams(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID string,
) {
	switch r.Method {
	case http.MethodGet:
		teams, err := h.Service.Teams(r.Context(), projectID, identity.ActorID, identity.IsAdmin())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"teams": teamsJSON(teams)})
	case http.MethodPost:
		var in struct {
			TeamID int64  `json:"team_id"`
			Role   string `json:"role"`
			Note   string `json:"note"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		invitation, err := h.Service.InviteTeam(
			r.Context(), projectID, in.TeamID, in.Role, in.Note,
			identity.ActorID, identity.IsAdmin(),
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

func (h *Handler) team(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID, teamPart string,
) {
	teamID, err := strconv.ParseInt(teamPart, 10, 64)
	if err != nil || teamID == 0 {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var in struct {
			Role string `json:"role"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		link, err := h.Service.SetTeamRole(
			r.Context(), projectID, teamID, in.Role, identity.ActorID, identity.IsAdmin(),
		)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"team": teamJSON(link)})
	case http.MethodDelete:
		if err := h.Service.RemoveTeam(
			r.Context(), projectID, teamID, identity.ActorID, identity.IsAdmin(),
		); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) invitations(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID string,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	list, err := h.Service.Invitations(r.Context(), projectID, identity.ActorID, identity.IsAdmin())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": invitationsJSON(list)})
}

func (h *Handler) invitation(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	projectID, idPart string,
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
		r.Context(), projectID, id, identity.ActorID, identity.IsAdmin(),
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
	projectID string,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := h.Service.Logs(
		r.Context(), projectID, identity.ActorID, identity.IsAdmin(), limit, before,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logsJSON(entries)})
}

func (h *Handler) governance(ctx context.Context) (Governance, error) {
	if h.Governance == nil {
		return Governance{}, nil
	}
	return h.Governance(ctx)
}

func projectRefJSON(p Project) map[string]any {
	out := map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"slug":        p.Slug,
		"prefix":      p.Prefix,
		"description": p.Description,
		"parent_id":   p.ParentID,
		"archived":    !p.ArchivedAt.IsZero(),
	}
	return out
}

func parentInvitationJSON(invitation ParentInvitation) map[string]any {
	out := map[string]any{
		"id":                    invitation.ID,
		"parent_project_id":     invitation.ParentProjectID,
		"parent_project_name":   invitation.ParentName,
		"parent_project_slug":   invitation.ParentSlug,
		"child_project_id":      invitation.ChildProjectID,
		"child_project_name":    invitation.ChildName,
		"child_project_slug":    invitation.ChildSlug,
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

func parentOptionJSON(option ParentOption) map[string]any {
	shared := make([]map[string]any, len(option.SharedOwnerTeams))
	for i, team := range option.SharedOwnerTeams {
		shared[i] = map[string]any{"team_id": team.ID, "team_name": team.Name}
	}
	return map[string]any{
		"project":            projectRefJSON(option.Project),
		"can_invite":         option.CanInvite,
		"can_link_directly":  option.CanLinkDirectly,
		"shared_owner_teams": shared,
	}
}

func hierarchyJSON(tree Hierarchy) map[string]any {
	out := map[string]any{
		"project":          projectRefJSON(tree.Project),
		"children":         projectRefsJSON(tree.Children),
		"can_detach":       tree.CanDetach,
		"can_respond":      tree.CanRespond,
		"can_invite_child": tree.CanInviteChild,
	}
	if tree.Parent != nil {
		out["parent"] = projectRefJSON(*tree.Parent)
	}
	if tree.Incoming != nil {
		out["incoming_invitation"] = parentInvitationJSON(*tree.Incoming)
	}
	out["outgoing_invitations"] = parentInvitationsJSON(tree.Outgoing)
	options := make([]map[string]any, len(tree.ParentOptions))
	for i, option := range tree.ParentOptions {
		options[i] = parentOptionJSON(option)
	}
	out["parent_options"] = options
	children := make([]map[string]any, len(tree.ChildOptions))
	for i, option := range tree.ChildOptions {
		children[i] = parentOptionJSON(option)
	}
	out["child_options"] = children
	return out
}

func projectRefsJSON(projects []Project) []map[string]any {
	out := make([]map[string]any, len(projects))
	for i, project := range projects {
		out[i] = projectRefJSON(project)
	}
	return out
}

func parentInvitationsJSON(invitations []ParentInvitation) []map[string]any {
	out := make([]map[string]any, len(invitations))
	for i, invitation := range invitations {
		out[i] = parentInvitationJSON(invitation)
	}
	return out
}

func membersJSON(members []Member) []map[string]any {
	out := make([]map[string]any, len(members))
	for i, m := range members {
		out[i] = memberJSON(m)
	}
	return out
}

func memberJSON(m Member) map[string]any {
	return map[string]any{
		"actor_id":            actorid.Format(m.ActorID),
		"kind":                m.Kind,
		"username":            m.Username,
		"name":                m.Name,
		"description":         m.Description,
		"role":                m.Role,
		"granted_by_actor_id": actorid.Format(m.GrantedByActorID),
		"created_at":          formatTime(m.CreatedAt),
		"updated_at":          formatTime(m.UpdatedAt),
	}
}

func teamsJSON(teams []TeamLink) []map[string]any {
	out := make([]map[string]any, len(teams))
	for i, t := range teams {
		out[i] = teamJSON(t)
	}
	return out
}

func teamJSON(t TeamLink) map[string]any {
	return map[string]any{
		"team_id":          t.TeamID,
		"team_name":        t.TeamName,
		"team_description": t.TeamDescription,
		"role":             t.Role,
		"member_count":     t.MemberCount,
		"created_at":       formatTime(t.CreatedAt),
		"updated_at":       formatTime(t.UpdatedAt),
	}
}

func invitationsJSON(invitations []TeamInvitation) []map[string]any {
	out := make([]map[string]any, len(invitations))
	for i, invitation := range invitations {
		out[i] = invitationJSON(invitation)
	}
	return out
}

func invitationJSON(invitation TeamInvitation) map[string]any {
	out := map[string]any{
		"id":                    invitation.ID,
		"project_id":            invitation.ProjectID,
		"project_name":          invitation.ProjectName,
		"project_slug":          invitation.ProjectSlug,
		"team_id":               invitation.TeamID,
		"team_name":             invitation.TeamName,
		"team_description":      invitation.TeamDescription,
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
			"project_id":     entry.ProjectID,
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
	case errors.Is(err, ErrProjectMissing),
		errors.Is(err, ErrTeamNotFound),
		errors.Is(err, ErrInvitationNotFound),
		errors.Is(err, ErrParentInvitationNotFound),
		errors.Is(err, ErrActorNotFound),
		errors.Is(err, ErrMemberNotFound),
		errors.Is(err, ErrNoParent):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrNotMember),
		errors.Is(err, ErrNotTeamOwner):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidPrefix):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrTeamAlreadyLinked),
		errors.Is(err, ErrInvitationExists),
		errors.Is(err, ErrInvitationExpired),
		errors.Is(err, ErrTeamHasNoAccess),
		errors.Is(err, ErrLastOwner),
		errors.Is(err, ErrProjectHasTasks),
		errors.Is(err, ErrProjectHasChildren),
		errors.Is(err, ErrAlreadyHasParent),
		errors.Is(err, ErrHierarchyCycle),
		errors.Is(err, ErrHierarchyHasReferences),
		errors.Is(err, ErrAgentHasTasks):
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
