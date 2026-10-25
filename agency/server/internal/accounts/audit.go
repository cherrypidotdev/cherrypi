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

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

func (s *Service) auditUser(
	ctx context.Context,
	subject ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if s.Audit == nil || subject == 0 {
		return
	}
	actor := audit.ActorID(ctx)
	if actor == 0 {
		actor = subject
	}
	s.Audit.LogUser(ctx, subject, actor, kind, field, oldValue, newValue, message, data)
}

func (s *Service) auditUserAs(
	ctx context.Context,
	subject, actor ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if s.Audit == nil || subject == 0 {
		return
	}
	s.Audit.LogUser(ctx, subject, actor, kind, field, oldValue, newValue, message, data)
}

func (s *Service) auditActor(
	ctx context.Context,
	subject ids.ActorID,
	kind, field, oldValue, newValue, message string,
	data any,
) {
	if s.Audit == nil || subject == 0 {
		return
	}
	if actor, err := s.store.GetActorByID(ctx, subject); err == nil && actor.Kind == KindAgent {
		s.Audit.LogAgent(ctx, subject, audit.ActorID(ctx), kind, field, oldValue,
			newValue, message, data)
		return
	}
	s.auditUser(ctx, subject, kind, field, oldValue, newValue, message, data)
}
