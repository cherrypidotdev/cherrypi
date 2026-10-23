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
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// StartupCredential is a CHERRYPI_AGENCY_CREDENTIALS pair. It seeds a user
// account on boot for SDK/CLI and demo access.
type StartupCredential struct {
	Username string
	Password string
}

// EnsureFirstAdmin creates the very first administrator on an empty
// installation. When password is empty a random one is generated and returned
// so the caller can print it once. It is a no-op when any user already exists.
func (s *Service) EnsureFirstAdmin(
	ctx context.Context,
	username, password string,
) (Actor, string, bool, error) {
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return Actor{}, "", false, err
	}
	if count > 0 {
		return Actor{}, "", false, nil
	}

	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	generated := ""
	if password == "" {
		password = randomPassword()
		generated = password
	}

	actor, err := s.CreateUser(ctx, username, password, RoleAdmin, StatusActive)
	if err != nil {
		return Actor{}, "", false, err
	}
	return actor, generated, true, nil
}

// SetRole changes a user's role, refusing to demote the last administrator.
func (s *Service) SetRole(ctx context.Context, actorID ids.ActorID, role string) error {
	if role != RoleAdmin && role != RoleMember {
		return ErrInvalidUsername
	}
	actor, err := s.GetActor(ctx, actorID)
	if err != nil {
		return err
	}
	if actor.Role == RoleAdmin && role != RoleAdmin {
		admins, err := s.store.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}
	if err := s.store.UpdateUserRole(ctx, store.UpdateUserRoleParams{
		Role:      role,
		UpdatedAt: s.now().UnixMilli(),
		ActorID:   int64(actorID),
	}); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindUserRole, "role", actor.Role, role,
		"Changed role of "+actor.Username, nil)
	return nil
}

// SetStatus changes a user's account status.
func (s *Service) SetStatus(ctx context.Context, actorID ids.ActorID, status string) error {
	switch status {
	case StatusInvited, StatusActive, StatusDisabled:
	default:
		return ErrInvalidUsername
	}
	if err := s.store.UpdateUserStatus(ctx, store.UpdateUserStatusParams{
		Status:    status,
		UpdatedAt: s.now().UnixMilli(),
		ActorID:   int64(actorID),
	}); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindUserStatus, "status", "", status,
		"Changed account status", nil)
	return nil
}

// DeleteUser removes a user account and its actor, refusing to delete the last
// administrator. Pending invitations are removed first, since the store does
// not enforce foreign keys.
func (s *Service) DeleteUser(ctx context.Context, actorID ids.ActorID) error {
	actor, err := s.GetActor(ctx, actorID)
	if err != nil {
		return err
	}
	if actor.Role == RoleAdmin {
		admins, err := s.store.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}
	if err := s.store.DeleteInvitationsByActor(ctx, actorID); err != nil {
		return err
	}
	// The user's domain log is scoped to the actor; foreign keys are off at
	// runtime, so clean it up explicitly. Entries the user made on other scopes
	// survive, with a NULL actor (see migration 043).
	if err := s.store.DeleteUserLogs(ctx, int64(actorID)); err != nil {
		return err
	}
	if err := s.store.DeleteAnyActor(ctx, actorID); err != nil {
		return err
	}
	return nil
}

// UpdateProfile changes the actor's display name and description.
func (s *Service) UpdateProfile(
	ctx context.Context,
	actorID ids.ActorID,
	name, description string,
) error {
	if err := s.store.UpdateActorProfile(ctx, store.UpdateActorProfileParams{
		Name:        strings.TrimSpace(name),
		Description: description,
		UpdatedAt:   s.now().UnixMilli(),
		ID:          int64(actorID),
	}); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindUserUpdate, "profile", "", "",
		"Updated profile", nil)
	return nil
}

// randomPassword returns a 32-hex-character password.
func randomPassword() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic("accounts: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

// CountUsers reports how many user accounts exist. It gates first boot.
func (s *Service) CountUsers(ctx context.Context) (int64, error) {
	return s.store.CountUsers(ctx)
}

// EnsureStartupUser seeds a `CHERRYPI_AGENCY_CREDENTIALS` pair: it creates the
// user when missing and otherwise resets its password to the configured value,
// so a compose/demo environment stays reproducible. It never demotes an
// existing admin; passing RoleAdmin promotes an existing member.
func (s *Service) EnsureStartupUser(
	ctx context.Context,
	username, password, role string,
) (Actor, error) {
	actor, created, err := s.EnsureUser(ctx, username, password, role)
	if err != nil {
		return Actor{}, err
	}
	if created {
		return actor, nil
	}
	if err := s.SetPassword(ctx, actor.ID, password); err != nil {
		return Actor{}, err
	}
	if role == RoleAdmin && actor.Role != RoleAdmin {
		if err := s.SetRole(ctx, actor.ID, RoleAdmin); err != nil {
			return Actor{}, err
		}
	}
	return s.GetActor(ctx, actor.ID)
}

// AuthenticateBasic verifies a username/password pair for the legacy basic-auth
// path (SDK/CLI compatibility). TOTP is not required on this non-interactive
// path.
func (s *Service) AuthenticateBasic(
	ctx context.Context,
	username, password string,
) (Identity, bool) {
	user, err := s.GetUserByUsername(ctx, username)
	if err != nil || user.Status == StatusDisabled {
		return Identity{}, false
	}
	row, err := s.store.GetActorByID(ctx, user.ID)
	if err != nil || !VerifyPassword(row.PasswordHash, password) {
		return Identity{}, false
	}
	return Identity{
		ActorID:  user.ID,
		Kind:     user.Kind,
		Username: user.Username,
		Name:     user.Name,
		Role:     user.Role,
		Status:   user.Status,
	}, true
}

// TouchLastLogin is exposed for transports that authenticate outside Login.
func (s *Service) TouchLastLogin(ctx context.Context, actorID ids.ActorID) {
	_ = s.store.TouchUserLogin(ctx, store.TouchUserLoginParams{
		LastLoginAt: nullInt64(time.Now().UnixMilli(), true),
		ActorID:     int64(actorID),
	})
}
