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

package agents

import (
	"context"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Record appends one entry to an agent's domain log for a mutation that lives
// outside the registry (limits, prompt, session cleanup). It is exported for
// the gRPC transport.
func (r *Registry) Record(
	ctx context.Context,
	agentActorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	r.logAgent(ctx, agentActorID, kind, field, oldValue, newValue, message, data)
}

// logAgent records one entry in an agent's domain log. It is best-effort: a
// missing audit service (tests) or a write failure never fails the mutation.
func (r *Registry) logAgent(
	ctx context.Context,
	agentActorID ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if r.Audit == nil || agentActorID == 0 {
		return
	}
	r.Audit.LogAgent(ctx, agentActorID, audit.ActorID(ctx), kind, field,
		oldValue, newValue, message, data)
}
