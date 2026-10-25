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
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	KindUser  = "user"
	KindAgent = "agent"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"

	StatusInvited  = "invited"
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

const (
	SessionTTL       = 7 * 24 * time.Hour
	RememberTTL      = 30 * 24 * time.Hour
	MagicLinkTTL     = 15 * time.Minute
	EmailConfirmTTL  = 24 * time.Hour
	PasswordResetTTL = time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrTOTPRequired       = errors.New("two-factor code required")
	ErrInvalidTOTP        = errors.New("invalid two-factor code")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUsernameInUse      = errors.New("username is already taken")
	ErrEmailInUse         = errors.New("e-mail is already registered")
	ErrEmailNotConfirmed  = errors.New("e-mail is not confirmed")
	ErrPasswordNotSet     = errors.New("account has no password")
	ErrInvalidUsername    = errors.New(
		"username must be non-empty and contain no whitespace, ':' or ','",
	)
	ErrInvalidPassword = errors.New("password must be 1-72 bytes")
	ErrInvalidEmail    = errors.New("invalid e-mail address")
	ErrNotFound        = errors.New("not found")
	ErrLastAdmin       = errors.New("cannot remove the last administrator")

	// Phase 3: invites, user lifecycle and org toggles.
	ErrInvitationNotFound        = errors.New("invitation not found")
	ErrInvitationExpired         = errors.New("invitation has expired")
	ErrPasswordLoginDisabled     = errors.New("password sign-in is disabled")
	ErrMagicLinkLoginDisabled    = errors.New("magic-link sign-in is disabled")
	ErrEmailConfirmationRequired = errors.New("e-mail confirmation is required")
	ErrInvitationsDisabled       = errors.New("inviting users is disabled")
	ErrSMTPNotConfigured         = errors.New("SMTP is not configured")
	ErrMembersCannotInvite       = errors.New("members may not invite users")
	ErrProviderRequired          = errors.New("provider is required")
	ErrProviderKind              = errors.New("provider does not match the invitation kind")
	ErrInvitationEmailRequired   = errors.New("e-mail invites require an address")
)

var hashCost = bcrypt.DefaultCost

type Actor struct {
	ID                  ids.ActorID
	Kind                string
	Username            string
	Name                string
	Description         string
	PictureAttachmentID int64

	Role        string
	Status      string
	LastLoginAt *time.Time

	// (username invites).
	MustChangePassword bool
	PasswordSet        bool
	TOTPEnabled        bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (a Actor) IsAdmin() bool { return a.Kind == KindUser && a.Role == RoleAdmin }

type Session struct {
	ID           int64
	ActorID      ids.ActorID
	Method       string
	UserAgent    string
	IP           string
	Remember     bool
	TOTPVerified bool
	CreatedAt    time.Time
	LastSeenAt   time.Time
	ExpiresAt    time.Time
}

type EmailCredential struct {
	ID        int64
	Email     string
	Confirmed bool
	CreatedAt time.Time
}

type PAT struct {
	ID         int64
	Name       string
	Prefix     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
}

type Service struct {
	store   store.Store
	now     func() time.Time
	mailer  Mailer
	baseURL string

	Audit *audit.Service
}

func New(st store.Store) *Service {
	return &Service{store: st, now: time.Now, mailer: SMTPSender{}}
}

func ValidateUsername(username string) error {
	if username == "" || strings.ContainsAny(username, ":, \t\r\n") {
		return ErrInvalidUsername
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) == 0 || len(password) > 72 {
		return ErrInvalidPassword
	}
	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return ErrInvalidEmail
	}
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return ErrInvalidEmail
	}
	domain := email[at+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") ||
		strings.HasSuffix(domain, ".") || strings.ContainsAny(email, " \t\r\n") {
		return ErrInvalidEmail
	}
	return nil
}

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), hashCost)
}

func VerifyPassword(hash []byte, password string) bool {
	if len(hash) == 0 {
		return false
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

func nullInt64(v int64, ok bool) sql.NullInt64 {
	if !ok {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

func isUniqueError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
}

func actorFromRows(row store.Actor, user *store.UserRow) Actor {
	return Actor{
		ID:                  ids.ActorID(row.ID),
		Kind:                row.Kind,
		Username:            row.Username,
		Name:                row.Name,
		Description:         row.Description,
		PictureAttachmentID: row.PictureAttachmentID.Int64,
		Role:                userRole(user),
		Status:              userStatus(user),
		LastLoginAt:         lastLogin(user),
		MustChangePassword:  userMustChangePassword(user),
		PasswordSet:         len(row.PasswordHash) > 0,
		TOTPEnabled:         row.TotpEnabledAt.Valid,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		UpdatedAt:           time.UnixMilli(row.UpdatedAt),
	}
}

func actorFromUserRow(row store.UserRow) Actor {
	return Actor{
		ID:                  row.ActorID,
		Kind:                KindUser,
		Username:            row.Username,
		Name:                row.Name,
		Description:         row.Description,
		PictureAttachmentID: row.PictureAttachmentID.Int64,
		Role:                row.Role,
		Status:              row.Status,
		LastLoginAt:         lastLoginUser(row),
		MustChangePassword:  row.MustChangePassword,
		PasswordSet:         len(row.PasswordHash) > 0,
		TOTPEnabled:         row.TOTPEnabledAt.Valid,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		UpdatedAt:           time.UnixMilli(row.UpdatedAt),
	}
}

func userRole(u *store.UserRow) string {
	if u == nil {
		return ""
	}
	return u.Role
}

func userStatus(u *store.UserRow) string {
	if u == nil {
		return ""
	}
	return u.Status
}

func userMustChangePassword(u *store.UserRow) bool {
	return u != nil && u.MustChangePassword
}

func lastLogin(u *store.UserRow) *time.Time {
	if u == nil || !u.LastLoginAt.Valid {
		return nil
	}
	t := time.UnixMilli(u.LastLoginAt.Int64)
	return &t
}

func lastLoginUser(u store.UserRow) *time.Time {
	if !u.LastLoginAt.Valid {
		return nil
	}
	t := time.UnixMilli(u.LastLoginAt.Int64)
	return &t
}

func (s *Service) GetActor(ctx context.Context, id ids.ActorID) (Actor, error) {
	row, err := s.store.GetActorByID(ctx, id)
	if err != nil {
		return Actor{}, err
	}
	var user *store.UserRow
	if row.Kind == KindUser {
		if u, err := s.store.GetUserByActor(ctx, id); err == nil {
			user = &u
		}
	}
	return actorFromRows(row, user), nil
}

func (s *Service) GetUserByUsername(ctx context.Context, username string) (Actor, error) {
	row, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return Actor{}, err
	}
	return actorFromUserRow(row), nil
}

func (s *Service) ListUsers(ctx context.Context) ([]Actor, error) {
	rows, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Actor, len(rows))
	for i, row := range rows {
		out[i] = actorFromUserRow(row)
	}
	return out, nil
}

func (s *Service) CreateUser(
	ctx context.Context,
	username, password, role, status string,
) (Actor, error) {
	if err := ValidateUsername(username); err != nil {
		return Actor{}, err
	}
	if role == "" {
		role = RoleMember
	}
	if status == "" {
		status = StatusActive
	}

	now := s.now()
	var passwordHash []byte
	var passwordUpdatedAt int64
	if password != "" {
		if err := ValidatePassword(password); err != nil {
			return Actor{}, err
		}
		hash, err := HashPassword(password)
		if err != nil {
			return Actor{}, err
		}
		passwordHash = hash
		passwordUpdatedAt = now.UnixMilli()
	}

	if _, err := s.store.GetActorByUsername(ctx, username); err == nil {
		return Actor{}, ErrUsernameInUse
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Actor{}, err
	}

	row, err := s.store.CreateUserAccount(ctx, store.CreateUserActorParams{
		Username:          username,
		Name:              username,
		Description:       "",
		PasswordHash:      passwordHash,
		PasswordUpdatedAt: nullInt64(passwordUpdatedAt, passwordUpdatedAt != 0),
		CreatedAt:         now.UnixMilli(),
		UpdatedAt:         now.UnixMilli(),
	}, store.CreateUserRowParams{
		Role:      role,
		Status:    status,
		CreatedAt: now.UnixMilli(),
		UpdatedAt: now.UnixMilli(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Actor{}, ErrUsernameInUse
		}
		return Actor{}, err
	}
	return actorFromUserRow(row), nil
}

func (s *Service) EnsureUser(
	ctx context.Context,
	username, password, role string,
) (Actor, bool, error) {
	user, err := s.GetUserByUsername(ctx, username)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Actor{}, false, err
	}

	created, err := s.CreateUser(ctx, username, password, role, StatusActive)
	if err != nil {
		return Actor{}, false, err
	}
	return created, true, nil
}

func (s *Service) SetPassword(ctx context.Context, actorID ids.ActorID, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	now := s.now()
	if err := s.store.UpdateActorPassword(ctx, store.UpdateActorPasswordParams{
		PasswordHash:      hash,
		PasswordUpdatedAt: nullInt64(now.UnixMilli(), true),
		UpdatedAt:         now.UnixMilli(),
		ID:                int64(actorID),
	}); err != nil {
		return err
	}
	_ = s.store.SetUserMustChangePassword(ctx, store.SetUserMustChangePasswordParams{
		MustChangePassword: 0,
		UpdatedAt:          now.UnixMilli(),
		ActorID:            int64(actorID),
	})
	return s.store.DeleteSessionsByActor(ctx, actorID)
}

func (s *Service) ChangePassword(
	ctx context.Context,
	actorID ids.ActorID,
	current, next string,
) error {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return err
	}
	if len(actor.PasswordHash) > 0 && !VerifyPassword(actor.PasswordHash, current) {
		return ErrInvalidCredentials
	}
	if err := s.SetPassword(ctx, actorID, next); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindPasswordChange, "", "", "",
		"Changed password", nil)
	return nil
}

func (s *Service) VerifyPassword(ctx context.Context, actorID ids.ActorID, password string) bool {
	actor, err := s.store.GetActorByID(ctx, actorID)
	if err != nil {
		return false
	}
	return VerifyPassword(actor.PasswordHash, password)
}

func (s *Service) Login(
	ctx context.Context,
	identifier, password, totpCode string,
) (Actor, error) {
	actor, err := s.resolveLoginActor(ctx, identifier)
	if err != nil {
		return Actor{}, ErrInvalidCredentials
	}
	if actor.Status == StatusDisabled {
		s.auditUserAs(ctx, actor.ID, actor.ID, audit.KindLoginFailed, "status",
			"", actor.Status, "Sign-in rejected: account disabled", nil)
		return Actor{}, ErrAccountDisabled
	}
	row, err := s.store.GetActorByID(ctx, actor.ID)
	if err != nil || !VerifyPassword(row.PasswordHash, password) {
		s.auditUserAs(ctx, actor.ID, actor.ID, audit.KindLoginFailed, "method",
			"", MethodPassword, "Sign-in rejected: invalid credentials", nil)
		return Actor{}, ErrInvalidCredentials
	}

	settings, err := s.GetAuthSettings(ctx)
	if err == nil && !actor.IsAdmin() {
		if !settings.PasswordLogin {
			return Actor{}, ErrPasswordLoginDisabled
		}
		if settings.EmailConfirmationRequired && !s.HasConfirmedEmail(ctx, actor.ID) {
			return Actor{}, ErrEmailConfirmationRequired
		}
	}

	if actor.TOTPEnabled && s.secondFactorRequired(ctx, MethodPassword) {
		if totpCode == "" {
			return Actor{}, ErrTOTPRequired
		}
		if !s.verifySecondFactor(ctx, actor, totpCode) {
			s.auditUserAs(ctx, actor.ID, actor.ID, audit.KindLoginFailed, "method",
				"", MethodPassword, "Sign-in rejected: invalid second factor", nil)
			return Actor{}, ErrInvalidTOTP
		}
	}

	now := s.now()
	_ = s.store.TouchUserLogin(ctx, store.TouchUserLoginParams{
		LastLoginAt: nullInt64(now.UnixMilli(), true),
		ActorID:     int64(actor.ID),
	})
	if actor.Status == StatusInvited {
		_ = s.store.UpdateUserStatus(ctx, store.UpdateUserStatusParams{
			Status:    StatusActive,
			UpdatedAt: now.UnixMilli(),
			ActorID:   int64(actor.ID),
		})
		actor.Status = StatusActive
		s.acceptPendingInvitation(ctx, actor.ID)
	}
	return actor, nil
}

func (s *Service) resolveLoginActor(ctx context.Context, identifier string) (Actor, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return Actor{}, ErrInvalidCredentials
	}
	row, err := s.store.GetActorByUsername(ctx, identifier)
	if err == nil {
		if row.Kind != KindUser {
			return Actor{}, ErrInvalidCredentials
		}
		user, err := s.store.GetUserByActor(ctx, ids.ActorID(row.ID))
		if err != nil {
			return Actor{}, err
		}
		return actorFromUserRow(user), nil
	}
	if strings.Contains(identifier, "@") {
		actorRow, err := s.store.GetActorByEmail(ctx, NormalizeEmail(identifier))
		if err == nil && actorRow.Kind == KindUser {
			user, err := s.store.GetUserByActor(ctx, ids.ActorID(actorRow.ID))
			if err == nil {
				return actorFromUserRow(user), nil
			}
		}
	}
	return Actor{}, ErrInvalidCredentials
}

func (s *Service) verifySecondFactor(ctx context.Context, actor Actor, code string) bool {
	row, err := s.store.GetActorByID(ctx, actor.ID)
	if err != nil {
		return false
	}
	if VerifyTOTP(row.TotpSecret, code, s.now()) {
		return true
	}
	return s.UseRecoveryCode(ctx, actor.ID, code)
}
