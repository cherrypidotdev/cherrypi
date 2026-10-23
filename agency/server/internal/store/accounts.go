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

package store

import (
	"database/sql"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Accounts store surface: users, e-mail credentials, sessions, single-use
// tokens, personal access tokens and TOTP recovery codes. The accounts service
// owns the policy; this file is a thin translation layer over sqlc.

// UserRow is the joined actor + user view.
type UserRow struct {
	ID                  ids.UserID
	ActorID             ids.ActorID
	Username            string
	Name                string
	Description         string
	PictureAttachmentID sql.NullInt64
	Role                string
	Status              string
	LastLoginAt         sql.NullInt64
	MustChangePassword  bool
	PasswordHash        []byte
	TOTPEnabledAt       sql.NullInt64
	CreatedAt           int64
	UpdatedAt           int64
}

// SessionRow is the joined session + actor + user view.
type SessionRow struct {
	ID                  int64
	ActorID             ids.ActorID
	TokenHash           []byte
	Method              string
	UserAgent           string
	IP                  string
	Remember            bool
	TOTPVerified        bool
	CreatedAt           int64
	LastSeenAt          int64
	ExpiresAt           int64
	Kind                string
	Username            string
	Name                string
	Description         string
	PictureAttachmentID sql.NullInt64
	PasswordHash        []byte
	TOTPSecret          []byte
	TOTPEnabledAt       sql.NullInt64
	Role                sql.NullString
	Status              sql.NullString
	MustChangePassword  sql.NullInt64
}

// PATRow is the joined personal-access-token + actor + user view.
type PATRow struct {
	ID            int64
	ActorID       ids.ActorID
	Name          string
	TokenHash     []byte
	Prefix        string
	CreatedAt     int64
	LastUsedAt    sql.NullInt64
	ExpiresAt     sql.NullInt64
	Kind          string
	Username      string
	ActorName     string
	TOTPEnabledAt sql.NullInt64
	Role          sql.NullString
	Status        sql.NullString
}

// CreateUser creates the actor and its user row atomically.

func userRowFrom(
	id, actorID int64,
	username, name, description string,
	picture sql.NullInt64,
	role, status string,
	lastLogin sql.NullInt64,
	mustChangePassword bool,
	passwordHash []byte,
	totpEnabledAt sql.NullInt64,
	createdAt, updatedAt int64,
) UserRow {
	return UserRow{
		ID:                  ids.UserID(id),
		ActorID:             ids.ActorID(actorID),
		Username:            username,
		Name:                name,
		Description:         description,
		PictureAttachmentID: picture,
		Role:                role,
		Status:              status,
		LastLoginAt:         lastLogin,
		MustChangePassword:  mustChangePassword,
		PasswordHash:        passwordHash,
		TOTPEnabledAt:       totpEnabledAt,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}
}

// E-mail credentials --------------------------------------------------------

// Sessions ------------------------------------------------------------------

// Tokens --------------------------------------------------------------------

// Personal access tokens ----------------------------------------------------

// TOTP recovery codes -------------------------------------------------------
