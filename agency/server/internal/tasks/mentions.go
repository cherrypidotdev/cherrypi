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

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

var mentionReference = regexp.MustCompile(
	`\[[^\]\n]*\]\(\s*<?ref:(actor|team)/([^)>\s]+)`,
)

type mentionReferenceParts struct {
	kind string
	key  string
}

func parseMentionReferences(body string) []mentionReferenceParts {
	matches := mentionReference.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]mentionReferenceParts, 0, len(matches))
	for _, match := range matches {
		key := match[2]
		if decoded, err := url.PathUnescape(key); err == nil {
			key = decoded
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out = append(out, mentionReferenceParts{kind: match[1], key: key})
	}
	return out
}

func (s *Service) MentionedActors(
	ctx context.Context,
	t Task,
	body string,
	actorID ids.ActorID,
) []ids.ActorID {
	references := parseMentionReferences(body)
	if len(references) == 0 {
		return nil
	}

	seen := make(map[ids.ActorID]bool, len(references))
	out := make([]ids.ActorID, 0, len(references))
	add := func(id ids.ActorID) {
		if id == 0 || id == actorID || seen[id] {
			return
		}
		if !s.canBeMentioned(ctx, t, id) {
			return
		}
		seen[id] = true
		out = append(out, id)
	}

	for _, reference := range references {
		switch reference.kind {
		case "actor":
			actor, err := s.resolveMentionActor(ctx, reference.key)
			if err != nil {
				continue
			}
			add(ids.ActorID(actor.ID))
		case "team":
			teamID, err := strconv.ParseInt(reference.key, 10, 64)
			if err != nil {
				continue
			}
			members, err := s.store.ListEffectiveTeamMemberActorIDs(ctx, teamID)
			if err != nil {
				continue
			}
			for _, member := range members {
				add(ids.ActorID(member))
			}
		}
	}
	return out
}

func (s *Service) resolveMentionActor(
	ctx context.Context,
	key string,
) (store.Actor, error) {
	actor, err := s.store.GetActorByUsername(ctx, key)
	if err == nil {
		return actor, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.Actor{}, err
	}
	id, parseErr := strconv.ParseInt(key, 10, 64)
	if parseErr != nil {
		return store.Actor{}, sql.ErrNoRows
	}
	return s.store.GetActorByID(ctx, ids.ActorID(id))
}

func (s *Service) canBeMentioned(ctx context.Context, t Task, id ids.ActorID) bool {
	actor, err := s.store.GetActorByID(ctx, id)
	if err != nil || actor.Kind != "user" {
		return false
	}
	user, err := s.store.GetUserByActor(ctx, id)
	if err != nil || user.Status != "active" {
		return false
	}
	allowed, err := s.CanView(ctx, t.ProjectID, id, user.Role == "admin")
	return err == nil && allowed
}
