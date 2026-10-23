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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// TOTP parameters (RFC 6238): SHA-1, six digits, 30-second steps.
const (
	totpPeriod = 30
	totpDigits = 6
	totpSkew   = 1
	// recoveryCodeCount is how many one-time codes an enrollment mints.
	recoveryCodeCount = 10
	// totpIssuer is the otpauth issuer label.
	totpIssuer = "CherryPi"
)

// ErrTOTPEnabled is returned when enrolling while TOTP is already active.
var ErrTOTPEnabled = errors.New("two-factor authentication is already enabled")

var base32NoPad = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateTOTPSecret returns a new base32 secret (20 random bytes).
func GenerateTOTPSecret() string {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		panic("accounts: crypto/rand failed: " + err.Error())
	}
	return base32NoPad.EncodeToString(buf)
}

// TOTPCode computes the six-digit code for a secret and time.
func TOTPCode(secret string, at time.Time) (string, error) {
	key, err := base32NoPad.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	counter := uint64(at.Unix() / totpPeriod)

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	return fmt.Sprintf("%0*d", totpDigits, value%1_000_000), nil
}

// VerifyTOTP checks code against secret, allowing one step of clock skew.
func VerifyTOTP(secret []byte, code string, at time.Time) bool {
	code = strings.TrimSpace(code)
	if len(secret) == 0 || code == "" {
		return false
	}
	str := string(secret)
	for offset := -totpSkew; offset <= totpSkew; offset++ {
		want, err := TOTPCode(str, at.Add(time.Duration(offset)*totpPeriod*time.Second))
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// TOTPURI renders the otpauth:// enrollment URI.
func TOTPURI(username, secret string) string {
	label := totpIssuer + ":" + username
	return fmt.Sprintf(
		"otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		label, secret, totpIssuer, totpDigits, totpPeriod,
	)
}

// BeginTOTPEnrollment stores a pending secret and returns it with the otpauth
// URI. The secret is not active until ActivateTOTP.
func (s *Service) BeginTOTPEnrollment(
	ctx context.Context,
	actorID ids.ActorID,
) (string, string, error) {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return "", "", err
	}
	if actor.TotpEnabledAt.Valid {
		return "", "", ErrTOTPEnabled
	}

	secret := GenerateTOTPSecret()
	now := s.now()
	if err := s.store.UpdateActorTOTP(ctx, store.UpdateActorTOTPParams{
		TotpSecret:    []byte(secret),
		TotpEnabledAt: sql.NullInt64{},
		UpdatedAt:     now.UnixMilli(),
		ID:            int64(actorID),
	}); err != nil {
		return "", "", err
	}
	return secret, TOTPURI(actor.Username, secret), nil
}

// ActivateTOTP verifies the first code, enables TOTP and returns ten fresh
// recovery codes.
func (s *Service) ActivateTOTP(
	ctx context.Context,
	actorID ids.ActorID,
	code string,
) ([]string, error) {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if len(actor.TotpSecret) == 0 {
		return nil, ErrInvalidTOTP
	}
	if actor.TotpEnabledAt.Valid {
		return nil, ErrTOTPEnabled
	}
	if !VerifyTOTP(actor.TotpSecret, code, s.now()) {
		return nil, ErrInvalidTOTP
	}

	now := s.now()
	if err := s.store.UpdateActorTOTP(ctx, store.UpdateActorTOTPParams{
		TotpSecret:    actor.TotpSecret,
		TotpEnabledAt: nullInt64(now.UnixMilli(), true),
		UpdatedAt:     now.UnixMilli(),
		ID:            int64(actorID),
	}); err != nil {
		return nil, err
	}
	codes, err := s.rotateRecoveryCodes(ctx, actorID)
	if err != nil {
		return nil, err
	}
	s.auditUser(ctx, actorID, audit.KindTOTPEnable, "", "", "",
		"Enabled two-factor authentication", nil)
	return codes, nil
}

// DisableTOTP turns TOTP off after verifying the password. It requires a
// password to be set.
func (s *Service) DisableTOTP(ctx context.Context, actorID ids.ActorID, password string) error {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return err
	}
	if len(actor.PasswordHash) > 0 && !VerifyPassword(actor.PasswordHash, password) {
		return ErrInvalidCredentials
	}
	now := s.now()
	if err := s.store.ClearActorTOTP(ctx, store.ClearActorTOTPParams{
		UpdatedAt: now.UnixMilli(),
		ID:        int64(actorID),
	}); err != nil {
		return err
	}
	if err := s.store.DeleteRecoveryCodes(ctx, actorID); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindTOTPDisable, "", "", "",
		"Disabled two-factor authentication", nil)
	return nil
}

// RotateRecoveryCodes verifies the password and mints a fresh batch.
func (s *Service) RotateRecoveryCodes(
	ctx context.Context,
	actorID ids.ActorID,
	password string,
) ([]string, error) {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !actor.TotpEnabledAt.Valid {
		return nil, ErrInvalidTOTP
	}
	if len(actor.PasswordHash) > 0 && !VerifyPassword(actor.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	codes, err := s.rotateRecoveryCodes(ctx, actorID)
	if err != nil {
		return nil, err
	}
	s.auditUser(ctx, actorID, audit.KindRecoveryCodes, "", "", "",
		"Regenerated recovery codes", nil)
	return codes, nil
}

func (s *Service) rotateRecoveryCodes(ctx context.Context, actorID ids.ActorID) ([]string, error) {
	if err := s.store.DeleteRecoveryCodes(ctx, actorID); err != nil {
		return nil, err
	}

	now := s.now()
	codes := make([]string, 0, recoveryCodeCount)
	for i := 0; i < recoveryCodeCount; i++ {
		code := newRecoveryCode()
		if _, err := s.store.CreateRecoveryCode(ctx, store.CreateRecoveryCodeParams{
			ActorID:   int64(actorID),
			CodeHash:  HashToken(normalizeRecoveryCode(code)),
			CreatedAt: now.UnixMilli(),
		}); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// UseRecoveryCode consumes one unused recovery code, returning whether it
// matched.
func (s *Service) UseRecoveryCode(ctx context.Context, actorID ids.ActorID, code string) bool {
	normalized := normalizeRecoveryCode(code)
	if normalized == "" {
		return false
	}
	rows, err := s.store.ListUnusedRecoveryCodes(ctx, actorID)
	if err != nil {
		return false
	}
	for _, row := range rows {
		if !EqualHash(row.CodeHash, normalized) {
			continue
		}
		n, err := s.store.ConsumeRecoveryCode(ctx, store.ConsumeRecoveryCodeParams{
			UsedAt: nullInt64(s.now().UnixMilli(), true),
			ID:     row.ID,
		})
		return err == nil && n == 1
	}
	return false
}

// CountRecoveryCodes reports how many unused recovery codes remain.
func (s *Service) CountRecoveryCodes(ctx context.Context, actorID ids.ActorID) (int, error) {
	count, err := s.store.CountUnusedRecoveryCodes(ctx, actorID)
	return int(count), err
}

// newRecoveryCode returns a 20-hex-character code grouped in fours.
func newRecoveryCode() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		panic("accounts: crypto/rand failed: " + err.Error())
	}
	raw := hex.EncodeToString(buf)
	return raw[0:5] + "-" + raw[5:10] + "-" + raw[10:15] + "-" + raw[15:20]
}

func normalizeRecoveryCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}
