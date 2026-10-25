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
	"net/http"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

// invitations. Disabled and still-invited accounts are omitted: the directory

func (h *Handler) directoryUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	users, err := h.Service.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load users")
		return
	}
	payload := make([]map[string]any, 0, len(users))
	for _, user := range users {
		if user.Status == StatusDisabled || user.Status == StatusInvited {
			continue
		}
		payload = append(payload, h.actorJSON(r.Context(), user))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": payload})
}

func (h *Handler) directoryUser(
	w http.ResponseWriter,
	r *http.Request,
	username string,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	username = strings.TrimSpace(username)
	if username == "" {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	user, err := h.Service.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	payload := map[string]any{"user": h.actorJSON(r.Context(), user)}
	if h.Teams != nil {
		if memberships, err := h.Teams.MembershipsForActor(r.Context(), user.ID); err == nil {
			payload["teams"] = directoryTeamsJSON(memberships)
		}
	}
	if h.Projects != nil {
		if readable, err := h.Projects.ProjectsForActor(r.Context(), user.ID); err == nil {
			payload["projects"] = directoryProjectsJSON(readable)
		}
	}
	if h.Agents != nil {
		if owned, err := h.Agents.OwnerActorIDs(r.Context(), user.ID); err == nil {
			ownedIDs := make([]string, len(owned))
			for i, actorID := range owned {
				ownedIDs[i] = strconv.FormatInt(int64(actorID), 10)
			}
			payload["owned_agent_actor_ids"] = ownedIDs
		}
	}
	writeJSON(w, http.StatusOK, payload)
}

func directoryTeamsJSON(list []teams.Team) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, team := range list {
		out[i] = map[string]any{
			"id":           team.ID,
			"name":         team.Name,
			"description":  team.Description,
			"member_count": team.MemberCount,
			"role":         team.Role,
			"created_at":   team.CreatedAt.UTC().Format(time.RFC3339),
			"updated_at":   team.UpdatedAt.UTC().Format(time.RFC3339),
		}
	}
	return out
}

func directoryProjectsJSON(list []projects.Project) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, project := range list {
		out[i] = map[string]any{
			"id":          project.ID,
			"name":        project.Name,
			"slug":        project.Slug,
			"description": project.Description,
			"created_at":  project.CreatedAt.UTC().Format(time.RFC3339),
		}
		if !project.ArchivedAt.IsZero() {
			out[i]["archived_at"] = project.ArchivedAt.UTC().Format(time.RFC3339)
		}
	}
	return out
}
