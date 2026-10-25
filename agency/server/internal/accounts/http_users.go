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

package accounts

import (
	"net/http"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *Handler) users(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		users, err := h.Service.ListUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load users")
			return
		}
		invitations, _ := h.Service.ListInvitations(r.Context())
		byActor := make(map[ids.ActorID]Invitation, len(invitations))
		for _, invitation := range invitations {
			if invitation.Status == InvitePending {
				byActor[invitation.ActorID] = invitation
			}
		}
		payload := make([]map[string]any, len(users))
		for i, user := range users {
			payload[i] = h.actorJSON(r.Context(), user)
			if invitation, ok := byActor[user.ID]; ok {
				payload[i]["invitation"] = invitationJSON(invitation)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": payload})
	case http.MethodPost:
		var in createUserRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		if in.Role == "" {
			in.Role = RoleMember
		}
		actor, err := h.Service.CreateUser(
			r.Context(), in.Username, in.Password, in.Role, StatusActive,
		)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		h.Service.auditUserAs(r.Context(), actor.ID, identity.ActorID,
			audit.KindUserCreate, "role", "", in.Role,
			"Created user "+actor.Username, nil)
		writeJSON(w, http.StatusCreated, map[string]any{"user": h.actorJSON(r.Context(), actor)})
	default:
		methodNotAllowed(w)
	}
}

type updateUserRequest struct {
	Role        *string `json:"role"`
	Status      *string `json:"status"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *Handler) userAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	idPart, action, _ := strings.Cut(rest, "/")
	actorID, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	target := ids.ActorID(actorID)

	switch {
	case action == "" && r.Method == http.MethodGet:
		actor, err := h.Service.GetActor(r.Context(), target)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"user": h.actorJSON(r.Context(), actor)})
	case action == "" && (r.Method == http.MethodPatch || r.Method == http.MethodPut):
		h.updateUser(w, r, target)
	case action == "" && r.Method == http.MethodDelete:
		if err := h.Service.DeleteUser(r.Context(), target); err != nil {
			writeAccountError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case action == "password" && r.Method == http.MethodPost:
		var in struct {
			Password string `json:"password"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if err := h.Service.SetPassword(r.Context(), target, in.Password); err != nil {
			writeAccountError(w, err)
			return
		}
		h.Service.auditUserAs(r.Context(), target, identity.ActorID,
			audit.KindPasswordChange, "", "", "",
			"An administrator reset the password", nil)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, target ids.ActorID) {
	var in updateUserRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	actor, err := h.Service.GetActor(r.Context(), target)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	if in.Role != nil && *in.Role != actor.Role {
		if err := h.Service.SetRole(r.Context(), target, *in.Role); err != nil {
			writeAccountError(w, err)
			return
		}
	}
	if in.Status != nil && *in.Status != actor.Status {
		if err := h.Service.SetStatus(r.Context(), target, *in.Status); err != nil {
			writeAccountError(w, err)
			return
		}
	}
	if in.Name != nil || in.Description != nil {
		name := actor.Name
		description := actor.Description
		if in.Name != nil {
			name = *in.Name
		}
		if in.Description != nil {
			description = *in.Description
		}
		if err := h.Service.UpdateProfile(r.Context(), target, name, description); err != nil {
			writeAccountError(w, err)
			return
		}
	}
	updated, err := h.Service.GetActor(r.Context(), target)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": h.actorJSON(r.Context(), updated)})
}

type createInvitationRequest struct {
	Kind             string `json:"kind"`
	Username         string `json:"username"`
	Email            string `json:"email"`
	ProviderID       int64  `json:"provider_id"`
	ExternalUsername string `json:"external_username"`
	Role             string `json:"role"`
}

func (h *Handler) invitations(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	switch r.Method {
	case http.MethodGet:
		if !identity.IsAdmin() {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		invitations, err := h.Service.ListInvitations(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load invitations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"invitations": invitationJSONList(invitations),
		})
	case http.MethodPost:
		var in createInvitationRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		result, err := h.Service.CreateInvitation(
			r.Context(), identity, InviteInput(in),
		)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		resp := map[string]any{"invitation": invitationJSON(result.Invitation)}
		if result.Password != "" {
			resp["password"] = result.Password
		}
		if result.Link != "" {
			resp["link"] = result.Link
			resp["delivered"] = result.Delivered
			if result.DeliveryError != "" {
				resp["delivery_error"] = result.DeliveryError
			}
		}
		writeJSON(w, http.StatusCreated, resp)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) invitationAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	idPart, action, _ := strings.Cut(rest, "/")
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invitation id")
		return
	}
	switch {
	case action == "" && r.Method == http.MethodDelete:
		if err := h.Service.RevokeInvitation(r.Context(), id); err != nil {
			writeAccountError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case action == "resend" && r.Method == http.MethodPost:
		result, err := h.Service.ResendInvitation(r.Context(), id)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		resp := map[string]any{"invitation": invitationJSON(result.Invitation)}
		if result.Link != "" {
			resp["link"] = result.Link
			resp["delivered"] = result.Delivered
			if result.DeliveryError != "" {
				resp["delivery_error"] = result.DeliveryError
			}
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) invitationPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	preview, err := h.Service.PreviewInvitation(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"username":   preview.Username,
		"email":      preview.Email,
		"kind":       preview.Kind,
		"role":       preview.Role,
		"expires_at": preview.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

type acceptInvitationRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in acceptInvitationRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	actor, err := h.Service.AcceptInvitation(r.Context(), in.Token, in.Password)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	h.establishSession(w, r, actor, "invitation", false, false)
}

type smtpSettingsRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	From       string `json:"from"`
	Encryption string `json:"encryption"`
}

func (h *Handler) smtpSettings(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := h.Service.GetSMTPSettings(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load settings")
			return
		}
		writeJSON(w, http.StatusOK, smtpSettingsJSON(settings))
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		var in smtpSettingsRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		if err := h.Service.UpdateSMTPSettings(r.Context(), SMTPSettings(in)); err != nil {
			writeAccountError(w, err)
			return
		}
		settings, err := h.Service.GetSMTPSettings(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load settings")
			return
		}
		writeJSON(w, http.StatusOK, smtpSettingsJSON(settings))
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) testSMTP(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		To string `json:"to"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.sendMail(
		r.Context(),
		strings.TrimSpace(in.To),
		"CherryPi SMTP test",
		"Your CherryPi SMTP settings are working.",
	); err != nil {
		writeError(w, http.StatusBadGateway, "could not send: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func invitationJSON(invitation Invitation) map[string]any {
	out := map[string]any{
		"id":                  invitation.ID,
		"kind":                invitation.Kind,
		"username":            invitation.Username,
		"actor_id":            invitation.ActorID,
		"email":               invitation.Email,
		"provider_id":         invitation.ProviderID,
		"external_username":   invitation.ExternalUsername,
		"role":                invitation.Role,
		"status":              invitation.Status,
		"invited_by_username": invitation.InvitedByUsername,
		"created_at":          invitation.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"expires_at":          invitation.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if invitation.AcceptedAt != nil {
		out["accepted_at"] = invitation.AcceptedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if invitation.RevokedAt != nil {
		out["revoked_at"] = invitation.RevokedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}

func invitationJSONList(list []Invitation) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, invitation := range list {
		out[i] = invitationJSON(invitation)
	}
	return out
}

func smtpSettingsJSON(settings SMTPSettings) map[string]any {
	return map[string]any{
		"host":         settings.Host,
		"port":         settings.Port,
		"username":     settings.Username,
		"from":         settings.From,
		"encryption":   settings.Encryption,
		"has_password": settings.Password != "",
	}
}
