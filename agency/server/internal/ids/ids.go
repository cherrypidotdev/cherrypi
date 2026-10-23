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

// Package ids defines the brand types that keep CherryPi's two identity
// concepts apart: an ActorID identifies any actor (user or agent), an AgentID
// identifies a row in the agent-specific `agents` table. The compiler rejects
// mixing them.
package ids

// ActorID is the serial identity of any actor (user or agent).
type ActorID int64

// AgentID is the serial id of an agent row (`agents.id`). Agent-specific
// tables reference it.
type AgentID int64

// UserID is the serial id of a user row (`users.id`). It arrives with the
// accounts phase.
type UserID int64
