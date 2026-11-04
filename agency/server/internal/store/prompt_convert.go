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
	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
)

func fromSqlitePrompt(v sqlitedb.Prompt) Prompt {
	return Prompt{
		ID:                v.ID,
		Scope:             v.Scope,
		AgentID:           v.AgentID,
		TeamID:            v.TeamID,
		ProjectID:         v.ProjectID,
		Content:           v.Content,
		Version:           v.Version,
		CreatedAt:         v.CreatedAt,
		AuthorActorID:     v.AuthorActorID,
		ApprovedByActorID: v.ApprovedByActorID,
		Note:              v.Note,
		ProposalID:        v.ProposalID,
		AutoApproved:      v.AutoApproved,
	}
}

func fromSqlitePromptSlice(vs []sqlitedb.Prompt) []Prompt {
	out := make([]Prompt, 0, len(vs))
	for _, v := range vs {
		out = append(out, fromSqlitePrompt(v))
	}
	return out
}

func fromPgPrompt(v pgdb.Prompt) Prompt {
	return Prompt{
		ID:                v.ID,
		Scope:             v.Scope,
		AgentID:           v.AgentID,
		TeamID:            v.TeamID,
		ProjectID:         v.ProjectID,
		Content:           v.Content,
		Version:           v.Version,
		CreatedAt:         v.CreatedAt,
		AuthorActorID:     v.AuthorActorID,
		ApprovedByActorID: v.ApprovedByActorID,
		Note:              v.Note,
		ProposalID:        v.ProposalID,
		AutoApproved:      v.AutoApproved,
	}
}

func fromPgPromptSlice(vs []pgdb.Prompt) []Prompt {
	out := make([]Prompt, 0, len(vs))
	for _, v := range vs {
		out = append(out, fromPgPrompt(v))
	}
	return out
}
