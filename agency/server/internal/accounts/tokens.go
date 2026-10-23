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
	"database/sql"
	"errors"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Single-use token kinds.
const (
	kindMagicLink     = "magic_link"
	kindEmailConfirm  = "email_confirm"
	kindPasswordReset = "password_reset"
)

// issueToken invalidates the actor's outstanding tokens of the same kind and
// mints a fresh one.
func (s *Service) issueToken(
	ctx context.Context,
	kind string,
	actorID ids.ActorID,
	emailID sql.NullInt64,
	ttl time.Duration,
) (string, error) {
	if err := s.store.DeleteTokensByActorKind(ctx, store.DeleteTokensByActorKindParams{
		ActorID: sql.NullInt64{Int64: int64(actorID), Valid: true},
		Kind:    kind,
	}); err != nil {
		return "", err
	}

	token := randomToken(32)
	now := s.now()
	if _, err := s.store.CreateToken(ctx, store.CreateTokenParams{
		TokenHash:         HashToken(token),
		Kind:              kind,
		ActorID:           sql.NullInt64{Int64: int64(actorID), Valid: true},
		EmailCredentialID: emailID,
		CreatedAt:         now.UnixMilli(),
		ExpiresAt:         now.Add(ttl).UnixMilli(),
	}); err != nil {
		return "", err
	}
	return token, nil
}

// consumeToken validates and consumes a single-use token of the given kind.
func (s *Service) consumeToken(
	ctx context.Context,
	token, kind string,
) (store.Token, error) {
	return s.consumeTokenMatching(ctx, token, func(k string) bool { return k == kind })
}

// consumeTokenMatching validates and consumes a single-use token whose kind
// the predicate accepts.
func (s *Service) consumeTokenMatching(
	ctx context.Context,
	token string,
	match func(kind string) bool,
) (store.Token, error) {
	if token == "" {
		return store.Token{}, ErrInvalidToken
	}
	row, err := s.store.GetTokenByHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Token{}, ErrInvalidToken
		}
		return store.Token{}, err
	}
	if !match(row.Kind) || row.ConsumedAt.Valid || row.ExpiresAt <= s.now().UnixMilli() {
		return store.Token{}, ErrInvalidToken
	}

	n, err := s.store.ConsumeToken(ctx, store.ConsumeTokenParams{
		ConsumedAt: sql.NullInt64{Int64: s.now().UnixMilli(), Valid: true},
		ID:         row.ID,
	})
	if err != nil {
		return store.Token{}, err
	}
	if n == 0 {
		return store.Token{}, ErrInvalidToken
	}
	return row, nil
}

// RequestMagicLink mints a login link for the identifier (username or
// confirmed e-mail). Callers must not reveal whether it matched.
func (s *Service) RequestMagicLink(
	ctx context.Context,
	identifier string,
) (string, Actor, error) {
	actor, err := s.resolveLoginActor(ctx, identifier)
	if err != nil {
		return "", Actor{}, err
	}
	if !s.HasConfirmedEmail(ctx, actor.ID) {
		return "", Actor{}, ErrEmailNotConfirmed
	}
	token, err := s.issueToken(ctx, kindMagicLink, actor.ID, sql.NullInt64{}, MagicLinkTTL)
	if err != nil {
		return "", Actor{}, err
	}
	s.auditUserAs(ctx, actor.ID, actor.ID, audit.KindMagicLink, "", "", "",
		"Requested a magic sign-in link", nil)
	return token, actor, nil
}

// ConsumeMagicLink exchanges a magic-link token for the actor it belongs to.
func (s *Service) ConsumeMagicLink(ctx context.Context, token string) (Actor, error) {
	row, err := s.consumeToken(ctx, token, kindMagicLink)
	if err != nil {
		return Actor{}, err
	}
	if !row.ActorID.Valid {
		return Actor{}, ErrInvalidToken
	}
	actor, err := s.GetActor(ctx, ids.ActorID(row.ActorID.Int64))
	if err != nil || actor.Kind != KindUser {
		return Actor{}, ErrInvalidToken
	}
	if actor.Status == StatusDisabled {
		return Actor{}, ErrAccountDisabled
	}
	return actor, nil
}

// RequestPasswordReset mints a reset token for the identifier.
func (s *Service) RequestPasswordReset(
	ctx context.Context,
	identifier string,
) (string, Actor, error) {
	actor, err := s.resolveLoginActor(ctx, identifier)
	if err != nil {
		return "", Actor{}, err
	}
	token, err := s.issueToken(ctx, kindPasswordReset, actor.ID, sql.NullInt64{}, PasswordResetTTL)
	if err != nil {
		return "", Actor{}, err
	}
	s.auditUserAs(ctx, actor.ID, actor.ID, audit.KindPasswordRequested, "", "", "",
		"Requested a password reset", nil)
	return token, actor, nil
}

// ResetPassword consumes a reset token and sets the new password.
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	row, err := s.consumeToken(ctx, token, kindPasswordReset)
	if err != nil {
		return err
	}
	if !row.ActorID.Valid {
		return ErrInvalidToken
	}
	actorID := ids.ActorID(row.ActorID.Int64)
	if err := s.SetPassword(ctx, actorID, password); err != nil {
		return err
	}
	s.auditUserAs(ctx, actorID, actorID, audit.KindPasswordReset, "", "", "",
		"Reset password", nil)
	return nil
}

// pendingLoginTTL is how long an external login may wait for its second
// factor before it must be restarted.
const pendingLoginTTL = 10 * time.Minute

// pendingKindPrefix marks a token that carries an authenticated external login
// until its second factor is verified. The suffix is the login method.
const pendingKindPrefix = "pending_"

// NeedsSecondFactor reports whether the actor must present a TOTP for a login
// through the method (the account has TOTP and the method enforces it).
func (s *Service) NeedsSecondFactor(ctx context.Context, actor Actor, method string) bool {
	return actor.TOTPEnabled && s.secondFactorRequired(ctx, method)
}

// BeginPendingLogin mints a short-lived token carrying an authenticated
// external login until its second factor is verified.
func (s *Service) BeginPendingLogin(
	ctx context.Context,
	actorID ids.ActorID,
	method string,
) (string, error) {
	return s.issueToken(
		ctx,
		pendingKindPrefix+method,
		actorID,
		sql.NullInt64{},
		pendingLoginTTL,
	)
}

// CompletePendingLogin consumes a pending token and verifies the second factor,
// returning the actor and the method that began the login.
func (s *Service) CompletePendingLogin(
	ctx context.Context,
	token, code string,
) (Actor, string, error) {
	row, err := s.consumeTokenMatching(ctx, token, func(kind string) bool {
		return strings.HasPrefix(kind, pendingKindPrefix)
	})
	if err != nil {
		return Actor{}, "", err
	}
	if !row.ActorID.Valid {
		return Actor{}, "", ErrInvalidToken
	}
	method := strings.TrimPrefix(row.Kind, pendingKindPrefix)
	actor, err := s.GetActor(ctx, ids.ActorID(row.ActorID.Int64))
	if err != nil {
		return Actor{}, "", ErrInvalidToken
	}
	if actor.Status == StatusDisabled {
		return Actor{}, "", ErrAccountDisabled
	}
	if !s.verifySecondFactor(ctx, actor, code) {
		return Actor{}, "", ErrInvalidTOTP
	}
	return actor, method, nil
}

// EmailForLogin returns the best confirmed address to deliver to: the
// identifier itself when it is a confirmed address, else the actor's first
// confirmed one. It returns "" when there is none.
func (s *Service) EmailForLogin(ctx context.Context, actor Actor, identifier string) string {
	normalized := NormalizeEmail(identifier)
	if strings.Contains(normalized, "@") {
		if cred, err := s.store.GetEmailCredentialByEmail(ctx, normalized); err == nil &&
			cred.ConfirmedAt.Valid {
			return cred.Email
		}
	}
	emails, err := s.ListEmails(ctx, actor.ID)
	if err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Confirmed {
			return e.Email
		}
	}
	return ""
}
