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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

const (
	AuthPrefix    = "/api/rest/auth"
	AccountPrefix = "/api/rest/account"
)

type HandlerOptions struct {
	BaseURL string

	ExposeMagicLinks bool

	InsecureFirstRegisteredUserBecomeAdmin bool
}

type Handler struct {
	Service *Service
	Options HandlerOptions

	Attachments *attachments.Service

	Teams TeamService

	Projects ProjectService

	Agents AgentOwnership
}

type TeamService interface {
	MembershipsForActor(ctx context.Context, actorID ids.ActorID) ([]teams.Team, error)
}

type ProjectService interface {
	ProjectsForActor(ctx context.Context, actorID ids.ActorID) ([]projects.Project, error)
}

type AgentOwnership interface {
	OwnerActorIDs(ctx context.Context, ownerActorID ids.ActorID) ([]ids.ActorID, error)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, AuthPrefix):
		h.serveAuth(w, r)
	case strings.HasPrefix(r.URL.Path, AccountPrefix):
		h.serveAccount(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) serveAuth(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, AuthPrefix)
	switch path {
	case "/login":
		h.login(w, r)
	case "/logout":
		h.logout(w, r)
	case "/session":
		h.session(w, r)
	case "/magic-link":
		h.requestMagicLink(w, r)
	case "/magic-link/consume":
		h.consumeMagicLink(w, r)
	case "/password-reset":
		h.requestPasswordReset(w, r)
	case "/password-reset/consume":
		h.consumePasswordReset(w, r)
	case "/confirm-email":
		h.confirmEmail(w, r)
	case "/register":
		h.register(w, r)
	case "/config":
		h.config(w, r)
	case "/providers":
		h.loginProviders(w, r)
	case "/ldap/login":
		h.ldapLogin(w, r)
	case "/invitations/preview":
		h.invitationPreview(w, r)
	case "/invitations/accept":
		h.acceptInvitation(w, r)
	case "/2fa/complete":
		h.completeSecondFactor(w, r)
	default:
		if strings.HasPrefix(path, "/providers/") {
			h.providerFlow(w, r, strings.TrimPrefix(path, "/providers/"))
			return
		}
		writeError(w, http.StatusNotFound, "not found")
	}
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	TOTP       string `json:"totp"`
	Remember   bool   `json:"remember"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in loginRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Identifier == "" {
		in.Identifier = in.Username
	}

	actor, err := h.Service.Login(r.Context(), in.Identifier, in.Password, in.TOTP)
	switch {
	case errors.Is(err, ErrTOTPRequired):
		writeJSON(w, http.StatusOK, map[string]any{"totpRequired": true})
		return
	case errors.Is(err, ErrInvalidTOTP):
		writeError(w, http.StatusUnauthorized, "invalid two-factor code")
		return
	case errors.Is(err, ErrAccountDisabled):
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	case err != nil:
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	h.establishSession(w, r, actor, "password", in.Remember, actor.TOTPEnabled)
}

func (h *Handler) establishSession(
	w http.ResponseWriter,
	r *http.Request,
	actor Actor,
	method string,
	remember, totpVerified bool,
) {
	csrf, err := h.setSession(w, r, actor, method, remember, totpVerified)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"actor":      h.actorJSON(r.Context(), actor),
		"csrf_token": csrf,
	})
}

func (h *Handler) setSession(
	w http.ResponseWriter,
	r *http.Request,
	actor Actor,
	method string,
	remember, totpVerified bool,
) (string, error) {
	token, _, err := h.Service.CreateSession(
		r.Context(), actor.ID, method, remember, totpVerified,
		r.UserAgent(), clientIP(r),
	)
	if err != nil {
		return "", err
	}
	csrf := randomToken(24)
	setSessionCookie(w, r, token, remember)
	setCSRFCookie(w, r, csrf)
	return csrf, nil
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		_ = h.Service.RevokeSessionByToken(r.Context(), cookie.Value)
	}
	clearCookie(w, r, SessionCookieName)
	clearCookie(w, r, CSRFCookieName)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	actor, err := h.Service.GetActor(r.Context(), identity.ActorID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"actor": h.actorJSON(r.Context(), actor)})
}

type magicLinkRequest struct {
	Identifier string `json:"identifier"`
	Username   string `json:"username"`
}

func (h *Handler) requestMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in magicLinkRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Identifier == "" {
		in.Identifier = in.Username
	}

	if settings, err := h.Service.GetAuthSettings(r.Context()); err == nil &&
		!settings.MagicLinkLogin {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	token, actor, err := h.Service.RequestMagicLink(r.Context(), in.Identifier)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	link := strings.TrimRight(h.Options.BaseURL, "/") +
		"/magic-link?token=" + url.QueryEscape(token)
	if actor.ID != 0 {
		logMagicLink(h.Options, actor.Username, link)
	}
	if h.Options.ExposeMagicLinks {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "magic_link": link})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type tokenRequest struct {
	Token    string `json:"token"`
	TOTP     string `json:"totp"`
	Remember bool   `json:"remember"`
}

func (h *Handler) consumeMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in tokenRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	actor, err := h.Service.ConsumeMagicLink(r.Context(), in.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired link")
		return
	}

	if h.Service.NeedsSecondFactor(r.Context(), actor, MethodMagicLink) {
		token, err := h.Service.BeginPendingLogin(r.Context(), actor.ID, MethodMagicLink)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not start verification")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"totpRequired":  true,
			"pending_token": token,
		})
		return
	}
	h.establishSession(w, r, actor, MethodMagicLink, in.Remember, false)
}

func (h *Handler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in magicLinkRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Identifier == "" {
		in.Identifier = in.Username
	}
	token, actor, err := h.Service.RequestPasswordReset(r.Context(), in.Identifier)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	link := strings.TrimRight(
		h.Options.BaseURL,
		"/",
	) + "/reset-password?token=" + url.QueryEscape(
		token,
	)
	if actor.ID != 0 {
		logMagicLink(h.Options, actor.Username, link)
	}
	if h.Options.ExposeMagicLinks {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "reset_link": link})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) consumePasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in resetPasswordRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.ResetPassword(r.Context(), in.Token, in.Password); err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) confirmEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in tokenRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	cred, err := h.Service.ConfirmEmail(r.Context(), in.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired link")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"email": emailJSON(cred)})
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !h.Options.InsecureFirstRegisteredUserBecomeAdmin {
		writeError(w, http.StatusForbidden, "registration is disabled")
		return
	}
	var in registerRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	count, err := h.Service.CountUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not register")
		return
	}
	if count > 0 {
		writeError(w, http.StatusForbidden, "registration is closed")
		return
	}
	actor, err := h.Service.CreateUser(
		r.Context(), in.Username, in.Password, RoleAdmin, StatusActive,
	)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	h.establishSession(w, r, actor, "password", false, false)
}

func (h *Handler) config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	settings, err := h.Service.GetAuthSettings(r.Context())
	if err != nil {
		settings = AuthSettings{PasswordLogin: true, MagicLinkLogin: true}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"registration_open":  h.Options.InsecureFirstRegisteredUserBecomeAdmin,
		"magic_link":         true,
		"password_login":     settings.PasswordLogin,
		"magic_link_login":   settings.MagicLinkLogin,
		"email_confirmation": settings.EmailConfirmationRequired,
	})
}

func (h *Handler) serveAccount(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, AccountPrefix)
	switch {
	case path == "" || path == "/":
		h.getAccount(w, r, identity)
	case path == "/password":
		h.changePassword(w, r, identity)
	case path == "/logs":
		h.accountLogs(w, r, identity)
	case path == "/profile":
		h.updateProfile(w, r, identity)
	case path == "/preferences":
		h.preferences(w, r, identity)
	case path == "/emails":
		h.emails(w, r, identity)
	case strings.HasPrefix(path, "/emails/"):
		h.emailAction(w, r, identity, strings.TrimPrefix(path, "/emails/"))
	case path == "/totp/enroll":
		h.totpEnroll(w, r, identity)
	case path == "/totp/activate":
		h.totpActivate(w, r, identity)
	case path == "/totp/disable":
		h.totpDisable(w, r, identity)
	case path == "/totp/recovery-codes":
		h.recoveryCodes(w, r, identity)
	case path == "/tokens":
		h.tokens(w, r, identity)
	case strings.HasPrefix(path, "/tokens/"):
		h.tokenAction(w, r, identity, strings.TrimPrefix(path, "/tokens/"))
	case path == "/sessions":
		h.sessions(w, r, identity)
	case path == "/sessions/revoke-others":
		h.revokeOtherSessions(w, r, identity)
	case strings.HasPrefix(path, "/sessions/"):
		h.sessionAction(w, r, identity, strings.TrimPrefix(path, "/sessions/"))
	case path == "/identities":
		h.identities(w, r, identity)
	case strings.HasPrefix(path, "/identities/"):
		h.identityAction(w, r, identity, strings.TrimPrefix(path, "/identities/"))
	case strings.HasPrefix(path, "/ldap-identities/"):
		h.ldapIdentityAction(w, r, identity, strings.TrimPrefix(path, "/ldap-identities/"))
	case path == "/providers":
		h.adminProviders(w, r, identity)
	case strings.HasPrefix(path, "/providers/"):
		h.adminProviderAction(w, r, identity, strings.TrimPrefix(path, "/providers/"))
	case path == "/ldap-providers":
		h.adminLDAPProviders(w, r, identity)
	case strings.HasPrefix(path, "/ldap-providers/"):
		h.adminLDAPProviderAction(
			w, r, identity, strings.TrimPrefix(path, "/ldap-providers/"),
		)
	case path == "/auth-settings":
		h.authSettings(w, r, identity)
	case path == "/smtp-settings":
		h.smtpSettings(w, r, identity)
	case path == "/smtp-settings/test":
		h.testSMTP(w, r, identity)
	case path == "/users":
		h.users(w, r, identity)
	case strings.HasPrefix(path, "/users/"):
		h.userAction(w, r, identity, strings.TrimPrefix(path, "/users/"))
	case path == "/directory/users":
		h.directoryUsers(w, r)
	case strings.HasPrefix(path, "/directory/users/"):
		h.directoryUser(w, r, strings.TrimPrefix(path, "/directory/users/"))
	case path == "/invitations":
		h.invitations(w, r, identity)
	case strings.HasPrefix(path, "/invitations/"):
		h.invitationAction(w, r, identity, strings.TrimPrefix(path, "/invitations/"))
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) accountLogs(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.Service.Audit == nil {
		writeJSON(w, http.StatusOK, map[string]any{"logs": []any{}})
		return
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := h.Service.Audit.ListUser(r.Context(), identity.ActorID, before, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"logs": audit.EntriesJSON(entries),
	})
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	actor, err := h.Service.GetActor(r.Context(), identity.ActorID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	emails, _ := h.Service.ListEmails(r.Context(), identity.ActorID)
	pats, _ := h.Service.ListPATs(r.Context(), identity.ActorID)
	sessions, _ := h.Service.ListSessions(r.Context(), identity.ActorID)
	recovery, _ := h.Service.CountRecoveryCodes(r.Context(), identity.ActorID)

	writeJSON(w, http.StatusOK, map[string]any{
		"actor":               h.actorJSON(r.Context(), actor),
		"emails":              emailJSONList(emails),
		"tokens":              patJSONList(pats),
		"sessions":            sessionJSONList(sessions),
		"recovery_codes_left": recovery,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type profileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) preferences(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	switch r.Method {
	case http.MethodGet:
		prefs, err := h.Service.GetPreferences(r.Context(), identity.ActorID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, preferencesJSON(prefs))
	case http.MethodPost, http.MethodPut:
		var in preferencesRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		prefs, err := h.Service.GetPreferences(r.Context(), identity.ActorID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if in.NotifyFirstArchive != nil {
			prefs.NotifyFirstArchive = *in.NotifyFirstArchive
		}
		if err := h.Service.UpdatePreferences(r.Context(), identity.ActorID, prefs); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, preferencesJSON(prefs))
	default:
		methodNotAllowed(w)
	}
}

type preferencesRequest struct {
	NotifyFirstArchive *bool `json:"notify_first_archive"`
}

func preferencesJSON(prefs Preferences) map[string]any {
	return map[string]any{
		"notify_first_archive": prefs.NotifyFirstArchive,
	}
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in profileRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.UpdateProfile(
		r.Context(),
		identity.ActorID,
		in.Name,
		in.Description,
	); err != nil {
		writeAccountError(w, err)
		return
	}
	actor, err := h.Service.GetActor(r.Context(), identity.ActorID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"actor": h.actorJSON(r.Context(), actor)})
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in changePasswordRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.ChangePassword(
		r.Context(), identity.ActorID, in.CurrentPassword, in.NewPassword,
	); err != nil {
		writeAccountError(w, err)
		return
	}
	clearCookie(w, r, SessionCookieName)
	clearCookie(w, r, CSRFCookieName)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type addEmailRequest struct {
	Email string `json:"email"`
}

func (h *Handler) emails(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	switch r.Method {
	case http.MethodGet:
		emails, err := h.Service.ListEmails(r.Context(), identity.ActorID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load e-mails")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"emails": emailJSONList(emails)})
	case http.MethodPost:
		var in addEmailRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		cred, err := h.Service.AddEmail(r.Context(), identity.ActorID, in.Email)
		if err != nil {
			writeAccountError(w, err)
			return
		}

		link := ""
		if token, err := h.Service.IssueEmailConfirmation(
			r.Context(),
			identity.ActorID,
			cred.ID,
		); err == nil {
			link = strings.TrimRight(h.Options.BaseURL, "/") +
				"/confirm-email?token=" + url.QueryEscape(token)
			logMagicLink(h.Options, identity.Username, link)
		}
		resp := map[string]any{"email": emailJSON(cred)}
		if h.Options.ExposeMagicLinks && link != "" {
			resp["confirm_link"] = link
		}
		writeJSON(w, http.StatusCreated, resp)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) emailAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	idPart, action, _ := strings.Cut(rest, "/")
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid e-mail id")
		return
	}
	switch {
	case action == "" && r.Method == http.MethodDelete:
		if err := h.Service.DeleteEmail(r.Context(), identity.ActorID, id); err != nil {
			writeAccountError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case action == "confirm" && r.Method == http.MethodPost:
		token, err := h.Service.IssueEmailConfirmation(r.Context(), identity.ActorID, id)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		link := strings.TrimRight(h.Options.BaseURL, "/") +
			"/confirm-email?token=" + url.QueryEscape(token)
		logMagicLink(h.Options, identity.Username, link)
		resp := map[string]any{"ok": true}
		if h.Options.ExposeMagicLinks {
			resp["confirm_link"] = link
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) totpEnroll(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	secret, uri, err := h.Service.BeginTOTPEnrollment(r.Context(), identity.ActorID)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"secret": secret, "otpauth_url": uri})
}

type totpCodeRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (h *Handler) totpActivate(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in totpCodeRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	codes, err := h.Service.ActivateTOTP(r.Context(), identity.ActorID, in.Code)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (h *Handler) totpDisable(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in totpCodeRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.Service.DisableTOTP(r.Context(), identity.ActorID, in.Password); err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) recoveryCodes(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in totpCodeRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	codes, err := h.Service.RotateRecoveryCodes(r.Context(), identity.ActorID, in.Password)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

type createTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn int64  `json:"expires_in_days"`
}

func (h *Handler) tokens(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	switch r.Method {
	case http.MethodGet:
		pats, err := h.Service.ListPATs(r.Context(), identity.ActorID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load tokens")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tokens": patJSONList(pats)})
	case http.MethodPost:
		var in createTokenRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		var expiresAt *time.Time
		if in.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(in.ExpiresIn) * 24 * time.Hour)
			expiresAt = &t
		}
		token, pat, err := h.Service.CreatePAT(r.Context(), identity.ActorID, in.Name, expiresAt)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"token": token,
			"pat":   patJSON(pat),
		})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) tokenAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	if err := h.Service.RevokePAT(r.Context(), identity.ActorID, id); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sessions(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	sessions, err := h.Service.ListSessions(r.Context(), identity.ActorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load sessions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessionJSONList(sessions)})
}

func (h *Handler) revokeOtherSessions(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if _, session, err := h.Service.ResolveSession(r.Context(), cookie.Value); err == nil {
			if err := h.Service.RevokeOtherSessions(
				r.Context(),
				identity.ActorID,
				session.ID,
			); err != nil {
				writeError(w, http.StatusInternalServerError, "could not revoke sessions")
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) sessionAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	if err := h.Service.RevokeSession(r.Context(), identity.ActorID, id); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func actorBaseJSON(a Actor) map[string]any {
	out := map[string]any{
		"id":                   a.ID,
		"kind":                 a.Kind,
		"username":             a.Username,
		"name":                 a.Name,
		"description":          a.Description,
		"password_set":         a.PasswordSet,
		"totp_enabled":         a.TOTPEnabled,
		"must_change_password": a.MustChangePassword,
		"created_at":           a.CreatedAt.UTC().Format(time.RFC3339),
		"is_admin":             a.IsAdmin(),
	}
	if a.Kind == KindUser {
		out["role"] = a.Role
		out["status"] = a.Status
	}
	if a.LastLoginAt != nil {
		out["last_login_at"] = a.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (h *Handler) actorJSON(ctx context.Context, a Actor) map[string]any {
	out := actorBaseJSON(a)
	if h.Attachments != nil && a.Kind == KindUser && a.PictureAttachmentID > 0 {
		if picture, ok := h.Attachments.UserPicture(ctx, a.ID); ok {
			out["picture"] = pictureJSON(picture)
		}
	}
	return out
}

func pictureJSON(a attachments.Attachment) map[string]any {
	return map[string]any{
		"id":          a.ID,
		"filename":    a.Filename,
		"contentType": a.ContentType,
		"size":        a.Size,
		"url":         attachments.URLFor(a.ID),
		"createdAt":   a.CreatedAt.UTC().Format(time.RFC3339),
		"createdBy":   a.CreatedBy,
	}
}

func emailJSON(e EmailCredential) map[string]any {
	return map[string]any{
		"id":         e.ID,
		"email":      e.Email,
		"confirmed":  e.Confirmed,
		"created_at": e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func emailJSONList(list []EmailCredential) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, e := range list {
		out[i] = emailJSON(e)
	}
	return out
}

func patJSON(p PAT) map[string]any {
	out := map[string]any{
		"id":           p.ID,
		"name":         p.Name,
		"prefix":       p.Prefix,
		"created_at":   p.CreatedAt.UTC().Format(time.RFC3339),
		"last_used_at": nil,
		"expires_at":   nil,
	}
	if p.LastUsedAt != nil {
		out["last_used_at"] = p.LastUsedAt.UTC().Format(time.RFC3339)
	}
	if p.ExpiresAt != nil {
		out["expires_at"] = p.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return out
}

func patJSONList(list []PAT) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, p := range list {
		out[i] = patJSON(p)
	}
	return out
}

func sessionJSON(s Session) map[string]any {
	return map[string]any{
		"id":           s.ID,
		"method":       s.Method,
		"user_agent":   s.UserAgent,
		"ip":           s.IP,
		"remember":     s.Remember,
		"created_at":   s.CreatedAt.UTC().Format(time.RFC3339),
		"last_seen_at": s.LastSeenAt.UTC().Format(time.RFC3339),
		"expires_at":   s.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

func sessionJSONList(list []Session) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, s := range list {
		out[i] = sessionJSON(s)
	}
	return out
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeAccountError(w http.ResponseWriter, err error) {
	if writeFederationError(w, err) {
		return
	}
	switch {
	case errors.Is(err, ErrUsernameInUse):
		writeError(w, http.StatusConflict, "username is already taken")
	case errors.Is(err, ErrEmailInUse):
		writeError(w, http.StatusConflict, "e-mail is already registered")
	case errors.Is(err, ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid e-mail address")
	case errors.Is(err, ErrInvalidUsername):
		writeError(w, http.StatusBadRequest, "invalid username")
	case errors.Is(err, ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, "password must be 1-72 characters")
	case errors.Is(err, ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, ErrInvalidTOTP):
		writeError(w, http.StatusBadRequest, "invalid two-factor code")
	case errors.Is(err, ErrTOTPRequired):
		writeJSON(w, http.StatusOK, map[string]any{"totpRequired": true})
	case errors.Is(err, ErrTOTPEnabled):
		writeError(w, http.StatusConflict, "two-factor authentication is already enabled")
	case errors.Is(err, ErrLastAdmin):
		writeError(w, http.StatusConflict, "cannot remove the last administrator")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, ErrEmailNotConfirmed):
		writeError(w, http.StatusBadRequest, "no confirmed e-mail")
	case errors.Is(err, ErrInvitationNotFound), errors.Is(err, ErrInvitationExpired):
		writeError(w, http.StatusNotFound, "invitation is no longer valid")
	case errors.Is(err, ErrPasswordLoginDisabled):
		writeError(w, http.StatusForbidden, "password sign-in is disabled")
	case errors.Is(err, ErrMagicLinkLoginDisabled):
		writeError(w, http.StatusForbidden, "magic-link sign-in is disabled")
	case errors.Is(err, ErrEmailConfirmationRequired):
		writeError(w, http.StatusForbidden, "e-mail confirmation is required")
	case errors.Is(err, ErrMembersCannotInvite), errors.Is(err, ErrInvitationsDisabled):
		writeError(w, http.StatusForbidden, "you may not invite users")
	case errors.Is(err, ErrSMTPNotConfigured):
		writeError(w, http.StatusBadRequest, "SMTP is not configured")
	case errors.Is(err, ErrProviderRequired), errors.Is(err, ErrProviderKind):
		writeError(w, http.StatusBadRequest, "invalid invitation provider")
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, remember bool) {
	maxAge := int(SessionTTL / time.Second)
	if remember {
		maxAge = int(RememberTTL / time.Second)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func setCSRFCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(RememberTTL / time.Second),
		HttpOnly: false,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func clearCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: name == SessionCookieName,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first, _, ok := strings.Cut(forwarded, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	return host
}

func logMagicLink(opts HandlerOptions, username, link string) {
	if !opts.ExposeMagicLinks {

		println("accounts: link for", username, ":", link)
	}
}

func AccountFromContext(ctx context.Context) (auth.Identity, bool) {
	return auth.IdentityFrom(ctx)
}
