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

package audit

const (
	KindLogin             = "login"
	KindLoginFailed       = "login_failed"
	KindLogout            = "logout"
	KindRegistered        = "registered"
	KindPasswordChange    = "password_change"
	KindPasswordReset     = "password_reset"
	KindPasswordRequested = "password_reset_requested"
	KindMagicLink         = "magic_link"
	KindEmailAdd          = "email_add"
	KindEmailRemove       = "email_remove"
	KindEmailConfirm      = "email_confirm"
	KindTOTPEnable        = "totp_enable"
	KindTOTPDisable       = "totp_disable"
	KindRecoveryCodes     = "recovery_codes"
	KindSessionRevoke     = "session_revoke"
	KindSessionsRevoke    = "sessions_revoke"
	KindPATCreate         = "pat_create"
	KindPATRevoke         = "pat_revoke"

	KindUserCreate     = "user_create"
	KindUserUpdate     = "user_update"
	KindUserRole       = "user_role"
	KindUserStatus     = "user_status"
	KindUserDelete     = "user_delete"
	KindInviteCreate   = "invitation_create"
	KindInviteRevoke   = "invitation_revoke"
	KindInviteResend   = "invitation_resend"
	KindInviteAccept   = "invitation_accept"
	KindIdentityLink   = "identity_link"
	KindIdentityUnlink = "identity_unlink"
	KindProviderCreate = "provider_create"
	KindProviderUpdate = "provider_update"
	KindProviderDelete = "provider_delete"
	KindAuthSettings   = "auth_settings"
	KindSMTPSettings   = "smtp_settings"
	KindPreferences    = "preferences"

	KindCreated          = "created"
	KindUpdated          = "updated"
	KindDeleted          = "deleted"
	KindCredentialCreate = "credential_create"
	KindCredentialRevoke = "credential_revoke"
	KindTokenCreate      = "token_create"
	KindTokenRevoke      = "token_revoke"
	KindOwnerAdd         = "owner_add"
	KindOwnerRemove      = "owner_remove"
	KindDefaults         = "defaults"
	KindLimits           = "limits"
	KindHierarchy        = "hierarchy"
	KindPrompt           = "prompt"
	KindCleanup          = "cleanup"
	KindSessionDelete    = "session_delete"
)
