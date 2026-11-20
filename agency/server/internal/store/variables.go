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
	"context"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
)

type VariableStore interface {
	ListVariables(ctx context.Context) ([]Variable, error)
	GetVariable(ctx context.Context, p GetVariableParams) (Variable, error)

	SaveVariable(ctx context.Context, p UpsertVariableParams, revision string) error

	DeleteVariable(ctx context.Context, p DeleteVariableParams, revision string) (bool, error)

	DeleteVariablesForScope(ctx context.Context, p DeleteVariablesForScopeParams) error

	ListTeamIDsForAgentActor(ctx context.Context, actorID int64) ([]int64, error)
}

func (s *SqliteStore) ListVariables(ctx context.Context) ([]Variable, error) {
	rows, err := s.q.ListVariables(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteVariableSlice(rows), nil
}

func (s *SqliteStore) GetVariable(ctx context.Context, p GetVariableParams) (Variable, error) {
	row, err := s.q.GetVariable(ctx, sqlitedb.GetVariableParams(p))
	if err != nil {
		return Variable{}, err
	}
	return fromSqliteVariable(row), nil
}

func (s *SqliteStore) SaveVariable(
	ctx context.Context,
	p UpsertVariableParams,
	revision string,
) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if err := q.UpsertVariable(ctx, sqlitedb.UpsertVariableParams(p)); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, revision)
	})
}

func (s *SqliteStore) DeleteVariable(
	ctx context.Context,
	p DeleteVariableParams,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		rows, err := q.DeleteVariable(ctx, sqlitedb.DeleteVariableParams(p))
		if err != nil {
			return err
		}
		if rows == 0 {
			return nil
		}
		deleted = true
		return q.BumpAgentConfigRevision(ctx, revision)
	})
	return deleted, err
}

func (s *SqliteStore) DeleteVariablesForScope(
	ctx context.Context,
	p DeleteVariablesForScopeParams,
) error {
	return s.q.DeleteVariablesForScope(ctx, sqlitedb.DeleteVariablesForScopeParams(p))
}

func (s *SqliteStore) ListTeamIDsForAgentActor(
	ctx context.Context,
	actorID int64,
) ([]int64, error) {
	return s.q.ListTeamIDsForAgentActor(ctx, actorID)
}

func (s *PgStore) ListVariables(ctx context.Context) ([]Variable, error) {
	rows, err := s.q.ListVariables(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgVariableSlice(rows), nil
}

func (s *PgStore) GetVariable(ctx context.Context, p GetVariableParams) (Variable, error) {
	row, err := s.q.GetVariable(ctx, pgdb.GetVariableParams(p))
	if err != nil {
		return Variable{}, err
	}
	return fromPgVariable(row), nil
}

func (s *PgStore) SaveVariable(
	ctx context.Context,
	p UpsertVariableParams,
	revision string,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.UpsertVariable(ctx, pgdb.UpsertVariableParams(p)); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, revision)
	})
}

func (s *PgStore) DeleteVariable(
	ctx context.Context,
	p DeleteVariableParams,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		rows, err := q.DeleteVariable(ctx, pgdb.DeleteVariableParams(p))
		if err != nil {
			return err
		}
		if rows == 0 {
			return nil
		}
		deleted = true
		return q.BumpAgentConfigRevision(ctx, revision)
	})
	return deleted, err
}

func (s *PgStore) DeleteVariablesForScope(
	ctx context.Context,
	p DeleteVariablesForScopeParams,
) error {
	return s.q.DeleteVariablesForScope(ctx, pgdb.DeleteVariablesForScopeParams(p))
}

func (s *PgStore) ListTeamIDsForAgentActor(
	ctx context.Context,
	actorID int64,
) ([]int64, error) {
	return s.q.ListTeamIDsForAgentActor(ctx, actorID)
}
