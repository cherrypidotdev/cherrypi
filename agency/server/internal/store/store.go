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

// AgentRow is the store's agent view: the agent surrogate id, the shared actor
// id, and the identity carried by `actors`.
type AgentRow struct {
	ID                  ids.AgentID
	ActorID             ids.ActorID
	Username            string
	Name                string
	Description         string
	PictureAttachmentID sql.NullInt64
	CreatedAt           int64
	UpdatedAt           int64
	LastSeenAt          sql.NullInt64
}

func toAgentRow(
	id, actorID int64,
	username, name, description string,
	picture sql.NullInt64,
	createdAt, updatedAt int64,
	lastSeen sql.NullInt64,
) AgentRow {
	return AgentRow{
		ID:                  ids.AgentID(id),
		ActorID:             ids.ActorID(actorID),
		Username:            username,
		Name:                name,
		Description:         description,
		PictureAttachmentID: picture,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
		LastSeenAt:          lastSeen,
	}
}

// nullActorID wraps an actor id for a recipient column; 0 stays NULL, which
// broadcasts the row (or matches nothing on a write).
func nullActorID(actorID int64) sql.NullInt64 {
	return sql.NullInt64{Int64: actorID, Valid: actorID != 0}
}
