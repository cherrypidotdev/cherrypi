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
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

// Invitation kinds. The username is always reserved by the inviter.
const (
	InviteUsername = "username"
	InviteEmail    = "email"
	InviteOIDC     = "oidc"
	InviteLDAP     = "ldap"
)

// Invitation lifecycle labels, derived from the timestamp columns.
const (
	InvitePending  = "pending"
	InviteAccepted = "accepted"
	InviteRevoked  = "revoked"
	InviteExpired  = "expired"
)

// Invitation is the public view of an invite.
type Invitation struct {
	ID                int64
	Kind              string
	Username          string
	ActorID           ids.ActorID
	Email             string
	ProviderID        int64
	ExternalUsername  string
	Role              string
	InvitedByActorID  ids.ActorID
	InvitedByUsername string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	AcceptedAt        *time.Time
	RevokedAt         *time.Time
	Status            string
}

// InviteInput is a request to create one of the four invitation kinds.
type InviteInput struct {
	Kind             string
	Username         string
	Email            string
	ProviderID       int64
	ExternalUsername string
	Role             string
}

// InviteResult is returned once, at creation: a username invite carries the
// generated password; an e-mail invite carries the copyable acceptance link.
type InviteResult struct {
	Invitation Invitation
	Password   string
	Link       string
	// Delivered is true when an e-mail invite was handed to SMTP. A delivery
	// failure is non-fatal: the link stays copyable and DeliveryError explains
	// why the mail did not go out.
	Delivered     bool
	DeliveryError string
}

// ListInvitations returns every invitation, newest first (admin-only).
func (s *Service) ListInvitations(ctx context.Context) ([]Invitation, error) {
	rows, err := s.store.ListInvitations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, len(rows))
	for i, row := range rows {
		out[i] = invitationFromRow(row, s.now())
	}
	return out, nil
}

// GetInvitation returns one invitation by id.
func (s *Service) GetInvitation(ctx context.Context, id int64) (Invitation, error) {
	row, err := s.store.GetInvitationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Invitation{}, ErrInvitationNotFound
		}
		return Invitation{}, err
	}
	return invitationFromRow(row, s.now()), nil
}

// CreateInvitation provisions an invited user actor and records the invite.
// The inviter must be an admin unless members may invite.
func (s *Service) CreateInvitation(
	ctx context.Context,
	inviter auth.Identity,
	in InviteInput,
) (InviteResult, error) {
	if !inviter.IsAdmin() {
		settings, err := s.GetAuthSettings(ctx)
		if err != nil {
			return InviteResult{}, err
		}
		if !settings.MembersCanInviteUsers {
			return InviteResult{}, ErrMembersCannotInvite
		}
	}
	in.Kind = strings.TrimSpace(in.Kind)
	in.Username = strings.TrimSpace(in.Username)
	if in.Role == "" {
		in.Role = RoleMember
	}
	if in.Role != RoleAdmin && in.Role != RoleMember {
		return InviteResult{}, ErrInvalidUsername
	}
	if in.Role == RoleAdmin && !inviter.IsAdmin() {
		return InviteResult{}, ErrMembersCannotInvite
	}
	if err := ValidateUsername(in.Username); err != nil {
		return InviteResult{}, err
	}
	if _, err := s.store.GetActorByUsername(ctx, in.Username); err == nil {
		return InviteResult{}, ErrUsernameInUse
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}
	if pending, err := s.store.GetPendingInvitationByUsername(
		ctx,
		in.Username,
	); err == nil && pending.ExpiresAt >= s.now().UnixMilli() {
		return InviteResult{}, ErrUsernameInUse
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}

	var (
		result InviteResult
		err    error
	)
	switch in.Kind {
	case InviteUsername:
		result, err = s.inviteByUsername(ctx, inviter, in)
	case InviteEmail:
		result, err = s.inviteByEmail(ctx, inviter, in)
	case InviteOIDC, InviteLDAP:
		result, err = s.inviteByProvider(ctx, inviter, in)
	default:
		return InviteResult{}, fmt.Errorf("%w: unknown kind %q", ErrInvitationNotFound, in.Kind)
	}
	if err != nil {
		return InviteResult{}, err
	}
	s.auditUserAs(ctx, result.Invitation.ActorID, inviter.ActorID,
		audit.KindInviteCreate, "kind", "", in.Kind,
		"Invited "+in.Username+" ("+in.Kind+")", nil)
	return result, nil
}

// inviteByUsername generates a password, shows it once and forces a reset on
// the first sign-in.
func (s *Service) inviteByUsername(
	ctx context.Context,
	inviter auth.Identity,
	in InviteInput,
) (InviteResult, error) {
	settings, err := s.GetAuthSettings(ctx)
	if err != nil {
		return InviteResult{}, err
	}
	if !settings.PasswordLogin {
		return InviteResult{}, ErrPasswordLoginDisabled
	}
	password := randomPassword()
	actor, err := s.CreateUser(ctx, in.Username, password, in.Role, StatusInvited)
	if err != nil {
		return InviteResult{}, err
	}
	if err := s.store.SetUserMustChangePassword(
		ctx,
		store.SetUserMustChangePasswordParams{
			MustChangePassword: 1,
			UpdatedAt:          s.now().UnixMilli(),
			ActorID:            int64(actor.ID),
		},
	); err != nil {
		return InviteResult{}, err
	}
	invitation, err := s.recordInvitation(ctx, inviter, in, actor.ID, nil)
	if err != nil {
		return InviteResult{}, err
	}
	return InviteResult{Invitation: invitation, Password: password}, nil
}

// inviteByEmail mails a single-use acceptance token; no password is set.
func (s *Service) inviteByEmail(
	ctx context.Context,
	inviter auth.Identity,
	in InviteInput,
) (InviteResult, error) {
	if !s.SMTPConfigured(ctx) {
		return InviteResult{}, ErrSMTPNotConfigured
	}
	in.Email = NormalizeEmail(in.Email)
	if err := ValidateEmail(in.Email); err != nil {
		return InviteResult{}, err
	}
	if _, err := s.store.GetEmailCredentialByEmail(ctx, in.Email); err == nil {
		return InviteResult{}, ErrEmailInUse
	} else if !errors.Is(err, sql.ErrNoRows) {
		return InviteResult{}, err
	}

	actor, err := s.CreateUser(ctx, in.Username, "", in.Role, StatusInvited)
	if err != nil {
		return InviteResult{}, err
	}
	now := s.now()
	if _, err := s.store.CreateEmailCredential(
		ctx,
		store.CreateEmailCredentialParams{
			ActorID:     int64(actor.ID),
			Email:       in.Email,
			ConfirmedAt: sql.NullInt64{},
			CreatedAt:   now.UnixMilli(),
			UpdatedAt:   now.UnixMilli(),
		},
	); err != nil {
		return InviteResult{}, err
	}

	token := randomToken(32)
	invitation, err := s.recordInvitation(ctx, inviter, in, actor.ID, HashToken(token))
	if err != nil {
		return InviteResult{}, err
	}
	link := s.inviteLink(token)
	err = s.sendMail(
		ctx,
		in.Email,
		"Your CherryPi invitation",
		fmt.Sprintf(
			"You have been invited to CherryPi as %s.\n\n"+
				"Accept the invitation and choose a password:\n%s\n\n"+
				"The link expires in 7 days.",
			actor.Username, link,
		),
	)
	// The invitation is already stored; a delivery failure still returns the
	// copyable link.
	result := InviteResult{Invitation: invitation, Link: link, Delivered: err == nil}
	if err != nil {
		result.DeliveryError = err.Error()
	}
	return result, nil
}

// inviteByProvider reserves the username and links the identity on the first
// federated login.
func (s *Service) inviteByProvider(
	ctx context.Context,
	inviter auth.Identity,
	in InviteInput,
) (InviteResult, error) {
	in.ExternalUsername = strings.TrimSpace(in.ExternalUsername)
	if in.ProviderID == 0 || in.ExternalUsername == "" {
		return InviteResult{}, ErrProviderRequired
	}
	switch in.Kind {
	case InviteOIDC:
		provider, err := s.GetIdentityProvider(ctx, in.ProviderID)
		if err != nil {
			return InviteResult{}, err
		}
		if provider.Kind != ProviderOIDC && provider.Kind != ProviderOAuth2 {
			return InviteResult{}, ErrProviderKind
		}
	case InviteLDAP:
		if _, err := s.GetLDAPProvider(ctx, in.ProviderID); err != nil {
			return InviteResult{}, err
		}
	}
	actor, err := s.CreateUser(ctx, in.Username, "", in.Role, StatusInvited)
	if err != nil {
		return InviteResult{}, err
	}
	invitation, err := s.recordInvitation(ctx, inviter, in, actor.ID, nil)
	if err != nil {
		return InviteResult{}, err
	}
	return InviteResult{Invitation: invitation}, nil
}

// recordInvitation writes the invite row after the actor already exists.
func (s *Service) recordInvitation(
	ctx context.Context,
	inviter auth.Identity,
	in InviteInput,
	actorID ids.ActorID,
	tokenHash []byte,
) (Invitation, error) {
	now := s.now()
	row, err := s.store.CreateInvitation(ctx, store.CreateInvitationParams{
		Kind:             in.Kind,
		Username:         in.Username,
		ActorID:          int64(actorID),
		Email:            nullString(in.Email),
		ProviderID:       nullInt64(in.ProviderID, in.ProviderID != 0),
		ExternalUsername: nullString(in.ExternalUsername),
		Role:             in.Role,
		TokenHash:        tokenHash,
		InvitedByActorID: nullInt64(int64(inviter.ActorID), inviter.ActorID != 0),
		CreatedAt:        now.UnixMilli(),
		ExpiresAt:        now.Add(InviteTTL).UnixMilli(),
	})
	if err != nil {
		return Invitation{}, err
	}
	return invitationFromRow(row, now), nil
}

// RevokeInvitation cancels a pending invite and removes the reserved actor, so
// the username is free again.
func (s *Service) RevokeInvitation(ctx context.Context, id int64) error {
	invitation, err := s.store.GetInvitationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvitationNotFound
		}
		return err
	}
	if invitation.AcceptedAt.Valid {
		return ErrInvitationNotFound
	}
	now := s.now().UnixMilli()
	n, err := s.store.RevokeInvitation(ctx, store.RevokeInvitationParams{
		RevokedAt: sql.NullInt64{Int64: now, Valid: true},
		ID:        id,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvitationNotFound
	}
	// Delete the still-invited actor; the invitation and its (empty) domain log
	// cascade with it. The revocation is recorded on the acting actor instead,
	// since the invitee no longer exists.
	if err := s.store.DeleteUserLogs(ctx, int64(invitation.ActorID)); err != nil {
		return err
	}
	if err := s.store.DeleteAnyActor(ctx, invitation.ActorID); err != nil {
		return err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindInviteRevoke, "", "", "",
		"Revoked the invitation for "+invitation.Username, nil)
	return nil
}

// ResendInvitation re-issues the acceptance token for an e-mail invite and
// re-sends the mail. Other kinds are a no-op refresh.
func (s *Service) ResendInvitation(ctx context.Context, id int64) (InviteResult, error) {
	invitation, err := s.store.GetInvitationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return InviteResult{}, ErrInvitationNotFound
		}
		return InviteResult{}, err
	}
	if invitation.AcceptedAt.Valid || invitation.RevokedAt.Valid {
		return InviteResult{}, ErrInvitationNotFound
	}
	if invitation.ExpiresAt < s.now().UnixMilli() {
		return InviteResult{}, ErrInvitationExpired
	}
	if invitation.Kind != InviteEmail {
		return InviteResult{Invitation: invitationFromRow(invitation, s.now())}, nil
	}
	if !s.SMTPConfigured(ctx) {
		return InviteResult{}, ErrSMTPNotConfigured
	}
	token := randomToken(32)
	if err := s.store.UpdateInvitationTokenHash(
		ctx,
		store.UpdateInvitationTokenHashParams{
			TokenHash: HashToken(token),
			ID:        id,
		},
	); err != nil {
		return InviteResult{}, err
	}
	link := s.inviteLink(token)
	result := InviteResult{
		Invitation: invitationFromRow(invitation, s.now()),
		Link:       link,
	}
	if err := s.sendMail(
		ctx,
		invitation.Email,
		"Your CherryPi invitation",
		fmt.Sprintf(
			"You have been invited to CherryPi as %s.\n\n"+
				"Accept the invitation and choose a password:\n%s\n\n"+
				"The link expires in 7 days.",
			invitation.Username, link,
		),
	); err != nil {
		result.DeliveryError = err.Error()
	} else {
		result.Delivered = true
	}
	s.auditUser(ctx, invitation.ActorID, audit.KindInviteResend, "", "", "",
		"Resent the invitation for "+invitation.Username, nil)
	return result, nil
}

// InvitationPreview is the public, non-secret view of an invite used by the
// acceptance page before the invitee sets a password.
type InvitationPreview struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Kind      string    `json:"kind"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PreviewInvitation validates an acceptance token without consuming it.
func (s *Service) PreviewInvitation(
	ctx context.Context,
	token string,
) (InvitationPreview, error) {
	row, err := s.store.GetInvitationByTokenHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return InvitationPreview{}, ErrInvitationNotFound
		}
		return InvitationPreview{}, err
	}
	if row.AcceptedAt.Valid || row.RevokedAt.Valid {
		return InvitationPreview{}, ErrInvitationNotFound
	}
	if row.ExpiresAt < s.now().UnixMilli() {
		return InvitationPreview{}, ErrInvitationExpired
	}
	return InvitationPreview{
		Username:  row.Username,
		Email:     row.Email,
		Kind:      row.Kind,
		Role:      row.Role,
		ExpiresAt: time.UnixMilli(row.ExpiresAt),
	}, nil
}

// AcceptInvitation consumes an e-mail invite token, sets the password, confirms
// the address and activates the account.
func (s *Service) AcceptInvitation(
	ctx context.Context,
	token, password string,
) (Actor, error) {
	row, err := s.store.GetInvitationByTokenHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Actor{}, ErrInvitationNotFound
		}
		return Actor{}, err
	}
	if row.AcceptedAt.Valid || row.RevokedAt.Valid {
		return Actor{}, ErrInvitationNotFound
	}
	if row.ExpiresAt < s.now().UnixMilli() {
		return Actor{}, ErrInvitationExpired
	}
	if err := ValidatePassword(password); err != nil {
		return Actor{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return Actor{}, err
	}
	now := s.now()
	if err := s.store.UpdateActorPassword(ctx, store.UpdateActorPasswordParams{
		PasswordHash:      hash,
		PasswordUpdatedAt: nullInt64(now.UnixMilli(), true),
		UpdatedAt:         now.UnixMilli(),
		ID:                int64(row.ActorID),
	}); err != nil {
		return Actor{}, err
	}
	if email, err := s.store.ListEmailCredentials(ctx, row.ActorID); err == nil {
		for _, cred := range email {
			if !cred.ConfirmedAt.Valid {
				_ = s.store.ConfirmEmailCredential(
					ctx,
					store.ConfirmEmailCredentialParams{
						ConfirmedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
						UpdatedAt:   now.UnixMilli(),
						ID:          cred.ID,
					},
				)
			}
		}
	}
	if _, err := s.store.AcceptInvitation(ctx, store.AcceptInvitationParams{
		AcceptedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ID:         row.ID,
	}); err != nil {
		return Actor{}, err
	}
	if err := s.store.UpdateUserStatus(ctx, store.UpdateUserStatusParams{
		Status:    StatusActive,
		UpdatedAt: now.UnixMilli(),
		ActorID:   int64(row.ActorID),
	}); err != nil {
		return Actor{}, err
	}
	s.auditUserAs(ctx, row.ActorID, row.ActorID, audit.KindInviteAccept, "",
		"", "", "Accepted the invitation", nil)
	return s.GetActor(ctx, row.ActorID)
}

// AcceptExternalInvitation links a provider identity to the actor reserved by
// a pending OIDC/LDAP invite, then activates the account. It returns
// ErrExternalNotAllowed when no invite matches, preserving invite-only.
func (s *Service) AcceptExternalInvitation(
	ctx context.Context,
	kind string,
	providerID int64,
	externalUsername string,
	link func(actorID ids.ActorID) error,
) (Actor, error) {
	externalUsername = strings.TrimSpace(externalUsername)
	if externalUsername == "" {
		return Actor{}, ErrExternalNotAllowed
	}
	row, err := s.store.GetPendingExternalInvitation(
		ctx,
		store.GetPendingExternalInvitationParams{
			Kind:             kind,
			ProviderID:       nullInt64(providerID, providerID != 0),
			ExternalUsername: nullString(externalUsername),
			Now:              s.now().UnixMilli(),
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Actor{}, ErrExternalNotAllowed
		}
		return Actor{}, err
	}
	if err := link(row.ActorID); err != nil {
		return Actor{}, err
	}
	now := s.now()
	if _, err := s.store.AcceptInvitation(ctx, store.AcceptInvitationParams{
		AcceptedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ID:         row.ID,
	}); err != nil {
		return Actor{}, err
	}
	if err := s.store.UpdateUserStatus(ctx, store.UpdateUserStatusParams{
		Status:    StatusActive,
		UpdatedAt: now.UnixMilli(),
		ActorID:   int64(row.ActorID),
	}); err != nil {
		return Actor{}, err
	}
	s.auditUserAs(ctx, row.ActorID, row.ActorID, audit.KindInviteAccept, "",
		"", "", "Accepted the invitation", nil)
	return s.GetActor(ctx, row.ActorID)
}

// AcceptExternalInvitationAny tries each provider-side username candidate
// against a pending invite. It returns the linked actor or
// ErrExternalNotAllowed when none match.
func (s *Service) AcceptExternalInvitationAny(
	ctx context.Context,
	kind string,
	providerID int64,
	candidates []string,
	link func(actorID ids.ActorID) error,
) (Actor, error) {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		actor, err := s.AcceptExternalInvitation(ctx, kind, providerID, candidate, link)
		if err == nil {
			return actor, nil
		}
		if !errors.Is(err, ErrExternalNotAllowed) {
			return Actor{}, err
		}
	}
	return Actor{}, ErrExternalNotAllowed
}

// acceptPendingInvitation marks a username invite accepted when its actor first
// signs in. Failure is non-fatal: the login still succeeds.
func (s *Service) acceptPendingInvitation(ctx context.Context, actorID ids.ActorID) {
	row, err := s.store.GetPendingInvitationByActor(ctx, actorID)
	if err != nil {
		return
	}
	_, _ = s.store.AcceptInvitation(ctx, store.AcceptInvitationParams{
		AcceptedAt: sql.NullInt64{Int64: s.now().UnixMilli(), Valid: true},
		ID:         row.ID,
	})
}

// inviteLink builds the public acceptance URL.
func (s *Service) inviteLink(token string) string {
	return strings.TrimRight(s.baseURL, "/") +
		"/invite?token=" + token
}

// baseURL is set by the transport from HandlerOptions.BaseURL.
func (s *Service) SetBaseURL(baseURL string) { s.baseURL = baseURL }

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func invitationFromRow(row store.InvitationRow, now time.Time) Invitation {
	expires := time.UnixMilli(row.ExpiresAt)
	out := Invitation{
		ID:                row.ID,
		Kind:              row.Kind,
		Username:          row.Username,
		ActorID:           row.ActorID,
		Email:             row.Email,
		ProviderID:        row.ProviderID,
		ExternalUsername:  row.ExternalUsername,
		Role:              row.Role,
		InvitedByActorID:  row.InvitedByActorID,
		InvitedByUsername: row.InvitedByUsername,
		CreatedAt:         time.UnixMilli(row.CreatedAt),
		ExpiresAt:         expires,
		Status:            InvitePending,
	}
	if row.AcceptedAt.Valid {
		t := time.UnixMilli(row.AcceptedAt.Int64)
		out.AcceptedAt = &t
		out.Status = InviteAccepted
	} else if row.RevokedAt.Valid {
		t := time.UnixMilli(row.RevokedAt.Int64)
		out.RevokedAt = &t
		out.Status = InviteRevoked
	} else if expires.Before(now) {
		out.Status = InviteExpired
	}
	return out
}
