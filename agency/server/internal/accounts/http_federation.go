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
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

const oauthStateCookieName = "cherrypi_oauth"

const oauthStateCookieMaxAge = 600

type oauthState struct {
	State    string `json:"s"`
	Nonce    string `json:"n,omitempty"`
	Verifier string `json:"v,omitempty"`
	Provider string `json:"p"`
	Mode     string `json:"m"`
	Redirect string `json:"r,omitempty"`
}

func (h *Handler) loginProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	providers, err := h.Service.ListEnabledIdentityProviders(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load providers")
		return
	}
	ldapProviders, err := h.Service.ListEnabledLDAPProviders(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load providers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"providers":      identityProviderJSONList(providers),
		"ldap_providers": ldapProviderJSONList(ldapProviders),
	})
}

func (h *Handler) providerFlow(w http.ResponseWriter, r *http.Request, rest string) {
	slug, action, _ := strings.Cut(rest, "/")
	switch action {
	case "start":
		h.providerStart(w, r, slug)
	case "callback":
		h.providerCallback(w, r, slug)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) providerStart(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	provider, err := h.Service.GetIdentityProviderBySlug(r.Context(), slug)
	if err != nil || !provider.Enabled {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "login"
	}
	if mode == "link" {

		if _, ok := auth.IdentityFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
	}

	redirectURI := h.providerRedirectURI(slug)
	var request AuthRequest
	switch provider.Kind {
	case ProviderOIDC:
		request, err = h.Service.BeginOIDC(r.Context(), provider, redirectURI)
	default:
		request, err = h.Service.BeginOAuth2(provider, redirectURI)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "identity provider is unavailable")
		return
	}

	setOAuthStateCookie(w, r, oauthState{
		State:    request.State,
		Nonce:    request.Nonce,
		Verifier: request.Verifier,
		Provider: slug,
		Mode:     mode,
		Redirect: cleanRedirect(r.URL.Query().Get("redirect")),
	})
	http.Redirect(w, r, request.URL, http.StatusFound)
}

func (h *Handler) providerCallback(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	state, ok := readOAuthState(w, r)
	if !ok || state.Provider != slug || state.State != r.URL.Query().Get("state") {
		h.redirectAuthError(w, r, "state")
		return
	}
	provider, err := h.Service.GetIdentityProviderBySlug(r.Context(), slug)
	if err != nil || !provider.Enabled {
		h.redirectAuthError(w, r, "provider")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectAuthError(w, r, "denied")
		return
	}

	redirectURI := h.providerRedirectURI(slug)
	method := MethodOIDC
	var user ExternalUser
	if provider.Kind == ProviderOIDC {
		user, err = h.Service.FinishOIDC(
			r.Context(), provider, code, state.Nonce, state.Verifier, redirectURI,
		)
	} else {
		method = MethodOAuth2
		user, err = h.Service.FinishOAuth2(
			r.Context(), provider, code, state.Verifier, redirectURI,
		)
	}
	if err != nil {
		h.redirectAuthError(w, r, "provider")
		return
	}

	if state.Mode == "link" {

		identity, ok := auth.IdentityFrom(r.Context())
		if !ok {
			h.redirectAuthError(w, r, "auth")
			return
		}
		if _, err := h.Service.LinkIdentity(
			r.Context(), identity.ActorID, provider.ID, user.Subject, user.Email,
		); err != nil {
			http.Redirect(w, r, "/account?error=identity", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/account?linked="+provider.Slug, http.StatusFound)
		return
	}

	actor, err := h.Service.ResolveExternalActor(r.Context(), provider.ID, user.Subject)
	if err != nil {
		// A provider-scoped invite may pre-create the link (phase 3): match

		actor, err = h.Service.AcceptExternalInvitationAny(
			r.Context(), InviteOIDC, provider.ID, []string{user.Username, user.Name},
			func(actorID ids.ActorID) error {
				_, linkErr := h.Service.LinkIdentity(
					r.Context(), actorID, provider.ID, user.Subject, user.Email,
				)
				return linkErr
			},
		)
		if err != nil {
			h.redirectAuthError(w, r, "unlinked")
			return
		}
	}
	if actor.Status == StatusDisabled {
		h.redirectAuthError(w, r, "disabled")
		return
	}
	if h.Service.NeedsSecondFactor(r.Context(), actor, method) {
		token, err := h.Service.BeginPendingLogin(r.Context(), actor.ID, method)
		if err != nil {
			h.redirectAuthError(w, r, "server")
			return
		}
		http.Redirect(
			w, r,
			"/login?pending="+token+"&method="+method,
			http.StatusFound,
		)
		return
	}
	if _, err := h.setSession(w, r, actor, method, true, false); err != nil {
		h.redirectAuthError(w, r, "server")
		return
	}
	http.Redirect(w, r, state.Redirect, http.StatusFound)
}

func (h *Handler) redirectAuthError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?error="+code, http.StatusFound)
}

func (h *Handler) providerRedirectURI(slug string) string {
	return strings.TrimRight(h.Options.BaseURL, "/") +
		"/api/rest/auth/providers/" + slug + "/callback"
}

type ldapLoginRequest struct {
	Provider string `json:"provider"`
	Username string `json:"username"`
	Password string `json:"password"`
	TOTP     string `json:"totp"`
	Remember bool   `json:"remember"`
}

func (h *Handler) ldapLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in ldapLoginRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	provider, err := h.Service.GetLDAPProviderBySlug(r.Context(), in.Provider)
	if err != nil || !provider.Enabled {
		writeError(w, http.StatusUnauthorized, "LDAP provider not found")
		return
	}
	ldapUser, err := h.Service.BindLDAP(
		r.Context(), provider, in.Username, in.Password,
	)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	actor, err := h.Service.ResolveLDAPActor(r.Context(), provider.ID, in.Username)
	if err != nil {
		// A provider-scoped LDAP invite may pre-create the link (phase 3).
		actor, err = h.Service.AcceptExternalInvitationAny(
			r.Context(), InviteLDAP, provider.ID,
			[]string{ldapUser.Username, in.Username},
			func(actorID ids.ActorID) error {
				_, linkErr := h.Service.LinkLDAPIdentity(
					r.Context(), actorID, provider.ID, ldapUser.Username,
				)
				return linkErr
			},
		)
		if err != nil {
			writeAccountError(w, err)
			return
		}
	}
	if actor.Status == StatusDisabled {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}
	if h.Service.NeedsSecondFactor(r.Context(), actor, MethodLDAP) {
		if in.TOTP == "" {
			writeJSON(w, http.StatusOK, map[string]any{"totpRequired": true})
			return
		}
		if !h.Service.verifySecondFactor(r.Context(), actor, in.TOTP) {
			writeError(w, http.StatusUnauthorized, "invalid two-factor code")
			return
		}
	}
	h.Service.TouchLastLogin(r.Context(), actor.ID)
	h.establishSession(w, r, actor, MethodLDAP, in.Remember, actor.TOTPEnabled)
}

type completeSecondFactorRequest struct {
	PendingToken string `json:"pending_token"`
	TOTP         string `json:"totp"`
	Remember     bool   `json:"remember"`
}

func (h *Handler) completeSecondFactor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in completeSecondFactorRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	actor, method, err := h.Service.CompletePendingLogin(
		r.Context(), in.PendingToken, in.TOTP,
	)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	h.establishSession(w, r, actor, method, in.Remember, true)
}

func (h *Handler) identities(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	externals, err := h.Service.ListIdentities(r.Context(), identity.ActorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load identities")
		return
	}
	ldapIdentities, err := h.Service.ListLDAPIdentities(r.Context(), identity.ActorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load identities")
		return
	}
	providers, _ := h.Service.ListEnabledIdentityProviders(r.Context())
	ldapProviders, _ := h.Service.ListEnabledLDAPProviders(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"identities":      externalIdentityJSONList(externals),
		"ldap_identities": ldapIdentityJSONList(ldapIdentities),
		"providers":       identityProviderJSONList(providers),
		"ldap_providers":  ldapProviderJSONList(ldapProviders),
	})
}

func (h *Handler) identityAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if rest == "ldap" {
		h.linkLDAPIdentity(w, r, identity)
		return
	}
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}
	if err := h.Service.UnlinkIdentity(r.Context(), identity.ActorID, id); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ldapIdentityAction(
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
		writeError(w, http.StatusBadRequest, "invalid identity id")
		return
	}
	if err := h.Service.UnlinkLDAPIdentity(r.Context(), identity.ActorID, id); err != nil {
		writeAccountError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type linkLDAPRequest struct {
	Provider string `json:"provider"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) linkLDAPIdentity(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in linkLDAPRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	provider, err := h.Service.GetLDAPProviderBySlug(r.Context(), in.Provider)
	if err != nil || !provider.Enabled {
		writeError(w, http.StatusNotFound, "LDAP provider not found")
		return
	}
	user, err := h.Service.BindLDAP(r.Context(), provider, in.Username, in.Password)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	linked, err := h.Service.LinkLDAPIdentity(
		r.Context(), identity.ActorID, provider.ID, user.Username,
	)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"identity": ldapIdentityJSON(linked)})
}

func (h *Handler) adminProviders(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		providers, err := h.Service.ListIdentityProviders(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load providers")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"providers": identityProviderJSONList(providers),
		})
	case http.MethodPost:
		var in providerRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		provider, err := h.Service.CreateIdentityProvider(
			r.Context(), in.toInput(true),
		)
		if err != nil {
			writeAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"provider": identityProviderJSON(provider)})
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) adminProviderAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		var in providerRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		provider, err := h.Service.UpdateIdentityProvider(r.Context(), id, in.toInput(false))
		if err != nil {
			writeAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"provider": identityProviderJSON(provider)})
	case http.MethodDelete:
		if err := h.Service.DeleteIdentityProvider(r.Context(), id); err != nil {
			writeAccountError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) adminLDAPProviders(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		providers, err := h.Service.ListLDAPProviders(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load providers")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ldap_providers": ldapProviderJSONList(providers),
		})
	case http.MethodPost:
		var in ldapProviderRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		provider, err := h.Service.CreateLDAPProvider(r.Context(), in.toInput(true))
		if err != nil {
			writeAccountError(w, err)
			return
		}
		writeJSON(
			w, http.StatusCreated,
			map[string]any{"ldap_provider": ldapProviderJSON(provider)},
		)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) adminLDAPProviderAction(
	w http.ResponseWriter,
	r *http.Request,
	identity auth.Identity,
	rest string,
) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		var in ldapProviderRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		provider, err := h.Service.UpdateLDAPProvider(r.Context(), id, in.toInput(false))
		if err != nil {
			writeAccountError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ldap_provider": ldapProviderJSON(provider)})
	case http.MethodDelete:
		if err := h.Service.DeleteLDAPProvider(r.Context(), id); err != nil {
			writeAccountError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) authSettings(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	if !identity.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := h.Service.GetAuthSettings(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load settings")
			return
		}
		writeJSON(w, http.StatusOK, authSettingsJSON(settings))
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		var in authSettingsRequest
		if !decodeJSON(w, r, &in) {
			return
		}
		if err := h.Service.UpdateAuthSettings(r.Context(), in.toSettings()); err != nil {
			writeAccountError(w, err)
			return
		}
		settings, err := h.Service.GetAuthSettings(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load settings")
			return
		}
		writeJSON(w, http.StatusOK, authSettingsJSON(settings))
	default:
		methodNotAllowed(w)
	}
}

func setOAuthStateCookie(w http.ResponseWriter, r *http.Request, state oauthState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(data),
		Path:     AuthPrefix + "/providers",
		MaxAge:   oauthStateCookieMaxAge,
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthState(w http.ResponseWriter, r *http.Request) (oauthState, bool) {
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || cookie.Value == "" {
		return oauthState{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return oauthState{}, false
	}
	var state oauthState
	if err := json.Unmarshal(raw, &state); err != nil {
		return oauthState{}, false
	}
	clearCookie(w, r, oauthStateCookieName)
	return state, true
}

func cleanRedirect(target string) string {
	if target == "" || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return "/tasks"
	}
	return target
}

type providerRequest struct {
	Kind                  string `json:"kind"`
	Slug                  string `json:"slug"`
	Name                  string `json:"name"`
	Enabled               *bool  `json:"enabled"`
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	ClientID              string `json:"client_id"`
	ClientSecret          string `json:"client_secret"`
	Scopes                string `json:"scopes"`
	SubjectPath           string `json:"subject_path"`
	EmailPath             string `json:"email_path"`
	NamePath              string `json:"name_path"`
}

func (in providerRequest) toInput(creating bool) IdentityProviderInput {
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	} else if !creating {
		enabled = false
	}
	return IdentityProviderInput{
		Kind:                  in.Kind,
		Slug:                  in.Slug,
		Name:                  in.Name,
		Enabled:               enabled,
		Issuer:                in.Issuer,
		AuthorizationEndpoint: in.AuthorizationEndpoint,
		TokenEndpoint:         in.TokenEndpoint,
		UserinfoEndpoint:      in.UserinfoEndpoint,
		JWKSURI:               in.JWKSURI,
		ClientID:              in.ClientID,
		ClientSecret:          in.ClientSecret,
		Scopes:                in.Scopes,
		SubjectPath:           in.SubjectPath,
		EmailPath:             in.EmailPath,
		NamePath:              in.NamePath,
	}
}

type ldapProviderRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Enabled      *bool  `json:"enabled"`
	Host         string `json:"host"`
	Port         int64  `json:"port"`
	Encryption   string `json:"encryption"`
	Insecure     bool   `json:"insecure"`
	BaseDN       string `json:"base_dn"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
	UserFilter   string `json:"user_filter"`
	UsernameAttr string `json:"username_attr"`
	EmailAttr    string `json:"email_attr"`
	NameAttr     string `json:"name_attr"`
}

func (in ldapProviderRequest) toInput(creating bool) LDAPProviderInput {
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	} else if !creating {
		enabled = false
	}
	return LDAPProviderInput{
		Slug:         in.Slug,
		Name:         in.Name,
		Enabled:      enabled,
		Host:         in.Host,
		Port:         in.Port,
		Encryption:   in.Encryption,
		Insecure:     in.Insecure,
		BaseDN:       in.BaseDN,
		BindDN:       in.BindDN,
		BindPassword: in.BindPassword,
		UserFilter:   in.UserFilter,
		UsernameAttr: in.UsernameAttr,
		EmailAttr:    in.EmailAttr,
		NameAttr:     in.NameAttr,
	}
}

type authSettingsRequest struct {
	TOTPPassword  *bool `json:"totp_password"`
	TOTPMagicLink *bool `json:"totp_magic_link"`
	TOTPOIDC      *bool `json:"totp_oidc"`
	TOTPOAuth2    *bool `json:"totp_oauth2"`
	TOTPLDAP      *bool `json:"totp_ldap"`
}

func (in authSettingsRequest) toSettings() AuthSettings {
	boolOr := func(value *bool, fallback bool) bool {
		if value == nil {
			return fallback
		}
		return *value
	}
	return AuthSettings{
		TOTPPassword:  boolOr(in.TOTPPassword, true),
		TOTPMagicLink: boolOr(in.TOTPMagicLink, true),
		TOTPOIDC:      boolOr(in.TOTPOIDC, true),
		TOTPOAuth2:    boolOr(in.TOTPOAuth2, true),
		TOTPLDAP:      boolOr(in.TOTPLDAP, true),
	}
}

func identityProviderJSON(p IdentityProvider) map[string]any {
	out := map[string]any{
		"id":                     p.ID,
		"kind":                   p.Kind,
		"slug":                   p.Slug,
		"name":                   p.Name,
		"enabled":                p.Enabled,
		"issuer":                 p.Issuer,
		"authorization_endpoint": p.AuthorizationEndpoint,
		"token_endpoint":         p.TokenEndpoint,
		"userinfo_endpoint":      p.UserinfoEndpoint,
		"jwks_uri":               p.JWKSURI,
		"client_id":              p.ClientID,
		"has_client_secret":      p.HasClientSecret,
		"scopes":                 p.Scopes,
		"subject_path":           p.SubjectPath,
		"email_path":             p.EmailPath,
		"name_path":              p.NamePath,
		"created_at":             p.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":             p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	return out
}

func identityProviderJSONList(list []IdentityProvider) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, p := range list {
		out[i] = identityProviderJSON(p)
	}
	return out
}

func ldapProviderJSON(p LDAPProvider) map[string]any {
	return map[string]any{
		"id":                p.ID,
		"slug":              p.Slug,
		"name":              p.Name,
		"enabled":           p.Enabled,
		"host":              p.Host,
		"port":              p.Port,
		"encryption":        p.Encryption,
		"insecure":          p.Insecure,
		"base_dn":           p.BaseDN,
		"bind_dn":           p.BindDN,
		"has_bind_password": p.HasBindPassword,
		"user_filter":       p.UserFilter,
		"username_attr":     p.UsernameAttr,
		"email_attr":        p.EmailAttr,
		"name_attr":         p.NameAttr,
		"created_at":        p.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":        p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func ldapProviderJSONList(list []LDAPProvider) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, p := range list {
		out[i] = ldapProviderJSON(p)
	}
	return out
}

func externalIdentityJSON(i ExternalIdentity) map[string]any {
	return map[string]any{
		"id":            i.ID,
		"provider_id":   i.ProviderID,
		"provider_slug": i.ProviderSlug,
		"provider_name": i.ProviderName,
		"kind":          i.Kind,
		"subject":       i.Subject,
		"email":         i.Email,
		"created_at":    i.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func externalIdentityJSONList(list []ExternalIdentity) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, identity := range list {
		out[i] = externalIdentityJSON(identity)
	}
	return out
}

func ldapIdentityJSON(i LDAPIdentity) map[string]any {
	return map[string]any{
		"id":            i.ID,
		"provider_id":   i.ProviderID,
		"provider_slug": i.ProviderSlug,
		"provider_name": i.ProviderName,
		"username":      i.Username,
		"created_at":    i.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ldapIdentityJSONList(list []LDAPIdentity) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, identity := range list {
		out[i] = ldapIdentityJSON(identity)
	}
	return out
}

func authSettingsJSON(settings AuthSettings) map[string]any {
	return map[string]any{
		"totp_password":   settings.TOTPPassword,
		"totp_magic_link": settings.TOTPMagicLink,
		"totp_oidc":       settings.TOTPOIDC,
		"totp_oauth2":     settings.TOTPOAuth2,
		"totp_ldap":       settings.TOTPLDAP,
	}
}

func writeFederationError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, ErrInvalidProvider):
		writeError(w, http.StatusBadRequest, "invalid provider configuration")
	case errors.Is(err, ErrProviderSlugInUse):
		writeError(w, http.StatusConflict, "provider slug is already taken")
	case errors.Is(err, ErrIdentityLinked):
		writeError(w, http.StatusConflict, "identity is already linked")
	case errors.Is(err, ErrIdentityNotFound):
		writeError(w, http.StatusNotFound, "identity not found")
	case errors.Is(err, ErrExternalNotAllowed):
		writeError(w, http.StatusForbidden, "no account is linked to this identity")
	case errors.Is(err, ErrProviderUnavailable):
		writeError(w, http.StatusBadGateway, "identity provider is unavailable")
	case errors.Is(err, ErrLDAPUnavailable):
		writeError(w, http.StatusBadGateway, "LDAP server is unavailable")
	case errors.Is(err, ErrLDAPUserNotFound):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	default:
		return false
	}
	return true
}
