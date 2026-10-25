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
	"cherrypi.vladfaust.com/agency/internal/ids"
)

func (s *Service) AddEmail(
	ctx context.Context,
	actorID ids.ActorID,
	email string,
) (EmailCredential, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return EmailCredential{}, err
	}
	if _, err := s.store.GetEmailCredentialByEmail(ctx, email); err == nil {
		return EmailCredential{}, ErrEmailInUse
	} else if !errors.Is(err, sql.ErrNoRows) {
		return EmailCredential{}, err
	}

	now := s.now()
	row, err := s.store.CreateEmailCredential(ctx, store.CreateEmailCredentialParams{
		ActorID:     int64(actorID),
		Email:       email,
		ConfirmedAt: sql.NullInt64{},
		CreatedAt:   now.UnixMilli(),
		UpdatedAt:   now.UnixMilli(),
	})
	if err != nil {
		if isUniqueError(err) {
			return EmailCredential{}, ErrEmailInUse
		}
		return EmailCredential{}, err
	}
	s.auditUser(ctx, actorID, audit.KindEmailAdd, "email", "", email,
		"Added e-mail "+email, nil)
	return emailFromRow(row), nil
}

func (s *Service) ListEmails(ctx context.Context, actorID ids.ActorID) ([]EmailCredential, error) {
	rows, err := s.store.ListEmailCredentials(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]EmailCredential, len(rows))
	for i, row := range rows {
		out[i] = emailFromRow(row)
	}
	return out, nil
}

func (s *Service) DeleteEmail(ctx context.Context, actorID ids.ActorID, emailID int64) error {
	row, err := s.store.GetEmailCredentialByID(ctx, emailID)
	if err != nil {
		return ErrNotFound
	}
	if ids.ActorID(row.ActorID) != actorID {
		return ErrNotFound
	}
	if err := s.store.DeleteEmailCredential(ctx, store.DeleteEmailCredentialParams{
		ID:      emailID,
		ActorID: int64(actorID),
	}); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindEmailRemove, "email", row.Email, "",
		"Removed e-mail "+row.Email, nil)
	return nil
}

func (s *Service) IssueEmailConfirmation(
	ctx context.Context,
	actorID ids.ActorID,
	emailID int64,
) (string, error) {
	row, err := s.store.GetEmailCredentialByID(ctx, emailID)
	if err != nil {
		return "", ErrNotFound
	}
	if ids.ActorID(row.ActorID) != actorID {
		return "", ErrNotFound
	}
	if row.ConfirmedAt.Valid {
		return "", ErrEmailInUse
	}
	return s.issueToken(ctx, kindEmailConfirm, actorID, sql.NullInt64{
		Int64: emailID,
		Valid: true,
	}, EmailConfirmTTL)
}

func (s *Service) ConfirmEmail(ctx context.Context, token string) (EmailCredential, error) {
	row, err := s.consumeToken(ctx, token, kindEmailConfirm)
	if err != nil {
		return EmailCredential{}, err
	}
	if !row.EmailCredentialID.Valid {
		return EmailCredential{}, ErrInvalidToken
	}
	now := s.now()
	_ = s.store.ConfirmEmailCredential(ctx, store.ConfirmEmailCredentialParams{
		ConfirmedAt: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		UpdatedAt:   now.UnixMilli(),
		ID:          row.EmailCredentialID.Int64,
	})
	cred, err := s.store.GetEmailCredentialByID(ctx, row.EmailCredentialID.Int64)
	if err != nil {
		return EmailCredential{}, ErrInvalidToken
	}
	s.auditUser(ctx, ids.ActorID(cred.ActorID), audit.KindEmailConfirm, "email",
		"", cred.Email, "Confirmed e-mail "+cred.Email, nil)
	return emailFromRow(cred), nil
}

func (s *Service) HasConfirmedEmail(ctx context.Context, actorID ids.ActorID) bool {
	count, err := s.store.CountConfirmedEmails(ctx, actorID)
	return err == nil && count > 0
}

func emailFromRow(row store.EmailCredential) EmailCredential {
	return EmailCredential{
		ID:        row.ID,
		Email:     row.Email,
		Confirmed: row.ConfirmedAt.Valid,
		CreatedAt: time.UnixMilli(row.CreatedAt),
	}
}
