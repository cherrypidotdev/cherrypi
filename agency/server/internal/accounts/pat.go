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

// CreatePAT mints a personal access token for the actor and returns the
// plaintext exactly once. A nil expiry means the token does not expire.
func (s *Service) CreatePAT(
	ctx context.Context,
	actorID ids.ActorID,
	name string,
	expiresAt *time.Time,
) (string, PAT, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "token"
	}

	token := PATPrefix + randomToken(32)
	now := s.now()
	var expiry sql.NullInt64
	if expiresAt != nil {
		expiry = sql.NullInt64{Int64: expiresAt.UnixMilli(), Valid: true}
	}

	row, err := s.store.CreatePersonalAccessToken(ctx, store.CreatePersonalAccessTokenParams{
		ActorID:   int64(actorID),
		Name:      name,
		TokenHash: HashToken(token),
		Prefix:    displayPrefix(token),
		CreatedAt: now.UnixMilli(),
		ExpiresAt: expiry,
	})
	if err != nil {
		return "", PAT{}, err
	}
	pat := patFromRow(row)
	s.auditActor(ctx, actorID, audit.KindPATCreate, "name", "", name,
		"Created personal access token "+name,
		map[string]any{"prefix": pat.Prefix})
	return token, pat, nil
}

// ResolvePAT validates a personal access token and returns the actor it
// belongs to. It rejects expired tokens and disabled accounts.
func (s *Service) ResolvePAT(ctx context.Context, token string) (Identity, PAT, error) {
	if !strings.HasPrefix(token, PATPrefix) {
		return Identity{}, PAT{}, ErrInvalidToken
	}
	row, err := s.store.GetPersonalAccessTokenByHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Identity{}, PAT{}, ErrInvalidToken
		}
		return Identity{}, PAT{}, err
	}
	if row.ExpiresAt.Valid && row.ExpiresAt.Int64 <= s.now().UnixMilli() {
		return Identity{}, PAT{}, ErrInvalidToken
	}
	if row.Status.Valid && row.Status.String == StatusDisabled {
		return Identity{}, PAT{}, ErrAccountDisabled
	}

	_ = s.store.TouchPersonalAccessToken(ctx, store.TouchPersonalAccessTokenParams{
		LastUsedAt: sql.NullInt64{Int64: s.now().UnixMilli(), Valid: true},
		ID:         row.ID,
	})

	identity := Identity{
		ActorID:  row.ActorID,
		Kind:     row.Kind,
		Username: row.Username,
		Name:     row.ActorName,
		Role:     row.Role.String,
		Status:   row.Status.String,
	}
	pat := patFromRow(store.PersonalAccessToken{
		ID:         row.ID,
		ActorID:    int64(row.ActorID),
		Name:       row.Name,
		TokenHash:  row.TokenHash,
		Prefix:     row.Prefix,
		CreatedAt:  row.CreatedAt,
		LastUsedAt: row.LastUsedAt,
		ExpiresAt:  row.ExpiresAt,
	})
	return identity, pat, nil
}

// ListPATs returns the actor's tokens, newest first.
func (s *Service) ListPATs(ctx context.Context, actorID ids.ActorID) ([]PAT, error) {
	rows, err := s.store.ListPersonalAccessTokens(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]PAT, len(rows))
	for i, row := range rows {
		out[i] = patFromRow(row)
	}
	return out, nil
}

// RevokePAT deletes one of the actor's tokens.
func (s *Service) RevokePAT(ctx context.Context, actorID ids.ActorID, id int64) error {
	if err := s.store.DeletePersonalAccessToken(ctx,
		store.DeletePersonalAccessTokenParams{
			ID:      id,
			ActorID: int64(actorID),
		}); err != nil {
		return err
	}
	s.auditActor(ctx, actorID, audit.KindPATRevoke, "", "", "",
		"Revoked a personal access token", nil)
	return nil
}

func patFromRow(row store.PersonalAccessToken) PAT {
	pat := PAT{
		ID:        row.ID,
		Name:      row.Name,
		Prefix:    row.Prefix,
		CreatedAt: time.UnixMilli(row.CreatedAt),
	}
	if row.LastUsedAt.Valid {
		t := time.UnixMilli(row.LastUsedAt.Int64)
		pat.LastUsedAt = &t
	}
	if row.ExpiresAt.Valid {
		t := time.UnixMilli(row.ExpiresAt.Int64)
		pat.ExpiresAt = &t
	}
	return pat
}

// displayPrefix keeps the scheme plus the first eight random characters.
func displayPrefix(token string) string {
	body := strings.TrimPrefix(token, PATPrefix)
	if len(body) > 8 {
		body = body[:8]
	}
	return PATPrefix + body
}
