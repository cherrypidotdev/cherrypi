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
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

type Identity = auth.Identity

func (s *Service) CreateSession(
	ctx context.Context,
	actorID ids.ActorID,
	method string,
	remember, totpVerified bool,
	userAgent, ip string,
) (string, Session, error) {
	token := randomToken(32)
	now := s.now()
	ttl := s.sessionTTL(ctx, remember)
	rememberFlag := int64(0)
	if remember {
		rememberFlag = 1
	}
	totpFlag := int64(0)
	if totpVerified {
		totpFlag = 1
	}

	row, err := s.store.CreateSession(ctx, store.CreateSessionParams{
		ActorID:      int64(actorID),
		TokenHash:    HashToken(token),
		Method:       method,
		UserAgent:    userAgent,
		Ip:           ip,
		Remember:     rememberFlag,
		TotpVerified: totpFlag,
		CreatedAt:    now.UnixMilli(),
		LastSeenAt:   now.UnixMilli(),
		ExpiresAt:    now.Add(ttl).UnixMilli(),
	})
	if err != nil {
		return "", Session{}, err
	}
	s.auditUserAs(ctx, actorID, actorID, audit.KindLogin, "method", "", method,
		"Signed in with "+method, nil)
	return token, sessionFromRow(row), nil
}

func (s *Service) ResolveSession(
	ctx context.Context,
	token string,
) (Identity, Session, error) {
	if token == "" {
		return Identity{}, Session{}, ErrInvalidToken
	}
	row, err := s.store.GetSessionByHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Identity{}, Session{}, ErrInvalidToken
		}
		return Identity{}, Session{}, err
	}

	now := s.now()
	if row.ExpiresAt <= now.UnixMilli() {
		_ = s.store.DeleteSessionByHash(ctx, row.TokenHash)
		return Identity{}, Session{}, ErrInvalidToken
	}
	if row.Status.Valid && row.Status.String == StatusDisabled {
		return Identity{}, Session{}, ErrAccountDisabled
	}

	session := Session{
		ID:           row.ID,
		ActorID:      row.ActorID,
		Method:       row.Method,
		UserAgent:    row.UserAgent,
		IP:           row.IP,
		Remember:     row.Remember,
		TOTPVerified: row.TOTPVerified,
		CreatedAt:    time.UnixMilli(row.CreatedAt),
		LastSeenAt:   time.UnixMilli(row.LastSeenAt),
		ExpiresAt:    time.UnixMilli(row.ExpiresAt),
	}
	ttl := s.sessionTTL(ctx, row.Remember)
	_ = s.store.TouchSession(ctx, store.TouchSessionParams{
		LastSeenAt: now.UnixMilli(),
		ExpiresAt:  now.Add(ttl).UnixMilli(),
		ID:         row.ID,
	})

	return Identity{
		ActorID:            row.ActorID,
		Kind:               row.Kind,
		Username:           row.Username,
		Name:               row.Name,
		Role:               row.Role.String,
		Status:             row.Status.String,
		MustChangePassword: row.MustChangePassword.Valid && row.MustChangePassword.Int64 != 0,
	}, session, nil
}

func (s *Service) RevokeSession(ctx context.Context, actorID ids.ActorID, sessionID int64) error {
	if err := s.store.DeleteSessionByID(ctx, store.DeleteSessionByIDParams{
		ID:      sessionID,
		ActorID: int64(actorID),
	}); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindSessionRevoke, "", "", "",
		"Revoked a session", nil)
	return nil
}

func (s *Service) RevokeSessionByToken(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	hash := HashToken(token)
	actorID := ids.ActorID(0)
	if row, err := s.store.GetSessionByHash(ctx, hash); err == nil {
		actorID = row.ActorID
	}
	if err := s.store.DeleteSessionByHash(ctx, hash); err != nil {
		return err
	}
	if actorID != 0 {
		s.auditUserAs(ctx, actorID, actorID, audit.KindLogout, "", "", "",
			"Signed out", nil)
	}
	return nil
}

func (s *Service) RevokeOtherSessions(
	ctx context.Context,
	actorID ids.ActorID,
	keepID int64,
) error {
	if err := s.store.DeleteOtherSessions(ctx, store.DeleteOtherSessionsParams{
		ActorID: int64(actorID),
		ID:      keepID,
	}); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindSessionsRevoke, "", "", "",
		"Signed out other sessions", nil)
	return nil
}

func (s *Service) ListSessions(ctx context.Context, actorID ids.ActorID) ([]Session, error) {
	rows, err := s.store.ListSessionsByActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]Session, len(rows))
	for i, row := range rows {
		out[i] = sessionFromRow(row)
	}
	return out, nil
}

func (s *Service) sessionTTL(ctx context.Context, remember bool) time.Duration {
	settings, err := s.GetAuthSettings(ctx)
	if err != nil {
		if remember {
			return RememberTTL
		}
		return SessionTTL
	}
	if remember {
		return time.Duration(settings.RememberTTLHours) * time.Hour
	}
	return time.Duration(settings.SessionTTLHours) * time.Hour
}

func (s *Service) Cleanup(ctx context.Context) error {
	now := s.now().UnixMilli()
	if _, err := s.store.DeleteExpiredSessions(ctx, now); err != nil {
		return err
	}
	if _, err := s.store.DeleteExpiredTokens(ctx, now); err != nil {
		return err
	}
	if _, err := s.store.DeleteExpiredPersonalAccessTokens(
		ctx,
		sql.NullInt64{Int64: now, Valid: true},
	); err != nil {
		return err
	}

	// still-invited actor, then the expired rows themselves (this store runs

	if _, err := s.store.DeleteExpiredInvitedActors(ctx, now); err != nil {
		return err
	}
	_, err := s.store.DeleteExpiredInvitations(ctx, now)
	return err
}

func sessionFromRow(row store.Session) Session {
	return Session{
		ID:           row.ID,
		ActorID:      ids.ActorID(row.ActorID),
		Method:       row.Method,
		UserAgent:    row.UserAgent,
		IP:           row.Ip,
		Remember:     row.Remember != 0,
		TOTPVerified: row.TotpVerified != 0,
		CreatedAt:    time.UnixMilli(row.CreatedAt),
		LastSeenAt:   time.UnixMilli(row.LastSeenAt),
		ExpiresAt:    time.UnixMilli(row.ExpiresAt),
	}
}
