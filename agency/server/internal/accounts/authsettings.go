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
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
)

// Login methods, used as the per-method TOTP toggle keys.
const (
	MethodPassword  = "password"
	MethodMagicLink = "magic_link"
	MethodOIDC      = "oidc"
	MethodOAuth2    = "oauth2"
	MethodLDAP      = "ldap"
)

// Auth settings keys (the `settings` key/value table).
const (
	settingTOTPPassword  = "auth.totp.password"
	settingTOTPMagicLink = "auth.totp.magic_link"
	settingTOTPOIDC      = "auth.totp.oidc"
	settingTOTPOAuth2    = "auth.totp.oauth2"
	settingTOTPLDAP      = "auth.totp.ldap"

	settingPasswordLogin       = "auth.password.enabled"
	settingMagicLinkLogin      = "auth.magic_link.enabled"
	settingEmailConfirmation   = "auth.email_confirmation.required"
	settingSessionTTLHours     = "auth.session.ttl_hours"
	settingRememberTTLHours    = "auth.session.remember_ttl_hours"
	settingMembersInviteUsers  = "members.can_invite_users"
	settingMembersCreateAgents = "members.can_create_agents"
	settingMembersCreateTeams  = "members.can_create_teams"
	settingProjectDeletion     = "admin_only.project_deletion"
	settingTeamDeletion        = "admin_only.team_deletion"
)

// Default TTLs and the locked invite lifetime.
const (
	DefaultSessionTTLHours  = 168 // 7 days
	DefaultRememberTTLHours = 720 // 30 days
	InviteTTL               = 7 * 24 * time.Hour
)

// AuthSettings holds the org-wide login toggles. Each TOTP flag says whether a
// login through that method must present a second factor when the account has
// TOTP enabled. All TOTP flags default to true; password and magic-link
// sign-in default on; e-mail confirmation defaults off.
type AuthSettings struct {
	TOTPPassword  bool
	TOTPMagicLink bool
	TOTPOIDC      bool
	TOTPOAuth2    bool
	TOTPLDAP      bool

	PasswordLogin             bool
	MagicLinkLogin            bool
	EmailConfirmationRequired bool
	SessionTTLHours           int
	RememberTTLHours          int

	// Governance: whether plain members may do these things (admins always
	// may), and whether project/team deletion is admin-only.
	MembersCanInviteUsers    bool
	MembersCanCreateAgents   bool
	MembersCanCreateTeams    bool
	ProjectDeletionAdminOnly bool
	TeamDeletionAdminOnly    bool
}

// defaultAuthSettings is the baseline a missing key falls back to.
func defaultAuthSettings() AuthSettings {
	return AuthSettings{
		TOTPPassword:              true,
		TOTPMagicLink:             true,
		TOTPOIDC:                  true,
		TOTPOAuth2:                true,
		TOTPLDAP:                  true,
		PasswordLogin:             true,
		MagicLinkLogin:            true,
		EmailConfirmationRequired: false,
		SessionTTLHours:           DefaultSessionTTLHours,
		RememberTTLHours:          DefaultRememberTTLHours,
		MembersCanInviteUsers:     false,
		MembersCanCreateAgents:    false,
		MembersCanCreateTeams:     false,
		ProjectDeletionAdminOnly:  true,
		TeamDeletionAdminOnly:     true,
	}
}

// GetAuthSettings reads the toggles; a missing key means the default.
func (s *Service) GetAuthSettings(ctx context.Context) (AuthSettings, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return AuthSettings{}, err
	}
	out := defaultAuthSettings()
	for _, row := range rows {
		switch row.Key {
		case settingTOTPPassword:
			out.TOTPPassword = parseSettingBool(row.Value)
		case settingTOTPMagicLink:
			out.TOTPMagicLink = parseSettingBool(row.Value)
		case settingTOTPOIDC:
			out.TOTPOIDC = parseSettingBool(row.Value)
		case settingTOTPOAuth2:
			out.TOTPOAuth2 = parseSettingBool(row.Value)
		case settingTOTPLDAP:
			out.TOTPLDAP = parseSettingBool(row.Value)
		case settingPasswordLogin:
			out.PasswordLogin = parseSettingBool(row.Value)
		case settingMagicLinkLogin:
			out.MagicLinkLogin = parseSettingBool(row.Value)
		case settingEmailConfirmation:
			out.EmailConfirmationRequired = parseSettingBool(row.Value)
		case settingSessionTTLHours:
			if value, err := strconv.Atoi(strings.TrimSpace(row.Value)); err == nil && value > 0 {
				out.SessionTTLHours = value
			}
		case settingRememberTTLHours:
			if value, err := strconv.Atoi(strings.TrimSpace(row.Value)); err == nil && value > 0 {
				out.RememberTTLHours = value
			}
		case settingMembersInviteUsers:
			out.MembersCanInviteUsers = parseSettingBool(row.Value)
		case settingMembersCreateAgents:
			out.MembersCanCreateAgents = parseSettingBool(row.Value)
		case settingMembersCreateTeams:
			out.MembersCanCreateTeams = parseSettingBool(row.Value)
		case settingProjectDeletion:
			out.ProjectDeletionAdminOnly = parseSettingBool(row.Value)
		case settingTeamDeletion:
			out.TeamDeletionAdminOnly = parseSettingBool(row.Value)
		}
	}
	return out, nil
}

// UpdateAuthSettings persists every toggle in one transaction.
func (s *Service) UpdateAuthSettings(ctx context.Context, in AuthSettings) error {
	if in.SessionTTLHours <= 0 {
		in.SessionTTLHours = DefaultSessionTTLHours
	}
	if in.RememberTTLHours <= 0 {
		in.RememberTTLHours = DefaultRememberTTLHours
	}
	values := []struct {
		key   string
		value string
	}{
		{settingTOTPPassword, boolString(in.TOTPPassword)},
		{settingTOTPMagicLink, boolString(in.TOTPMagicLink)},
		{settingTOTPOIDC, boolString(in.TOTPOIDC)},
		{settingTOTPOAuth2, boolString(in.TOTPOAuth2)},
		{settingTOTPLDAP, boolString(in.TOTPLDAP)},
		{settingPasswordLogin, boolString(in.PasswordLogin)},
		{settingMagicLinkLogin, boolString(in.MagicLinkLogin)},
		{settingEmailConfirmation, boolString(in.EmailConfirmationRequired)},
		{settingSessionTTLHours, strconv.Itoa(in.SessionTTLHours)},
		{settingRememberTTLHours, strconv.Itoa(in.RememberTTLHours)},
		{settingMembersInviteUsers, boolString(in.MembersCanInviteUsers)},
		{settingMembersCreateAgents, boolString(in.MembersCanCreateAgents)},
		{settingMembersCreateTeams, boolString(in.MembersCanCreateTeams)},
		{settingProjectDeletion, boolString(in.ProjectDeletionAdminOnly)},
		{settingTeamDeletion, boolString(in.TeamDeletionAdminOnly)},
	}
	for _, item := range values {
		if err := s.store.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   item.key,
			Value: item.value,
		}); err != nil {
			return err
		}
	}
	if actor := audit.ActorID(ctx); actor != 0 {
		s.auditUser(ctx, actor, audit.KindAuthSettings, "", "", "",
			"Updated authentication settings", nil)
	}
	return nil
}

// secondFactorRequired reports whether a login method enforces the account's
// TOTP. A read failure fails open to the default (required) only when the
// account has TOTP; callers gate on the account flag separately.
func (s *Service) secondFactorRequired(ctx context.Context, method string) bool {
	settings, err := s.GetAuthSettings(ctx)
	if err != nil {
		return true
	}
	switch method {
	case MethodPassword:
		return settings.TOTPPassword
	case MethodMagicLink:
		return settings.TOTPMagicLink
	case MethodOIDC:
		return settings.TOTPOIDC
	case MethodOAuth2:
		return settings.TOTPOAuth2
	case MethodLDAP:
		return settings.TOTPLDAP
	default:
		return true
	}
}

func parseSettingBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "off", "no", "disabled":
		return false
	default:
		return true
	}
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// ErrSecondFactorRequired is returned by external logins when the account has
// TOTP and the method enforces it; the caller starts the step-up flow.
var ErrSecondFactorRequired = errors.New("second factor required")
