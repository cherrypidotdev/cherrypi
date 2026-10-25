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

// InvitationRow is the joined invitation + invited actor view.
type InvitationRow struct {
	ID                int64
	Kind              string
	Username          string
	ActorID           ids.ActorID
	Email             string
	ProviderID        int64
	ExternalUsername  string
	Role              string
	InvitedByActorID  ids.ActorID
	CreatedAt         int64
	ExpiresAt         int64
	AcceptedAt        sql.NullInt64
	RevokedAt         sql.NullInt64
	ActorName         string
	ActorStatus       string
	InvitedByUsername string
}

func invitationRowFrom(
	id int64,
	kind, username string,
	actorID int64,
	email sql.NullString,
	providerID sql.NullInt64,
	externalUsername sql.NullString,
	role string,
	invitedByActorID sql.NullInt64,
	createdAt, expiresAt int64,
	acceptedAt, revokedAt sql.NullInt64,
	actorName string,
	actorStatus sql.NullString,
	invitedByUsername sql.NullString,
) InvitationRow {
	return InvitationRow{
		ID:                id,
		Kind:              kind,
		Username:          username,
		ActorID:           ids.ActorID(actorID),
		Email:             email.String,
		ProviderID:        providerID.Int64,
		ExternalUsername:  externalUsername.String,
		Role:              role,
		InvitedByActorID:  ids.ActorID(invitedByActorID.Int64),
		CreatedAt:         createdAt,
		ExpiresAt:         expiresAt,
		AcceptedAt:        acceptedAt,
		RevokedAt:         revokedAt,
		ActorName:         actorName,
		ActorStatus:       actorStatus.String,
		InvitedByUsername: invitedByUsername.String,
	}
}
