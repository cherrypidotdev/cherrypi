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

func (s *SqliteStore) UpsertTaskHold(ctx context.Context, p UpsertTaskHoldParams) error {
	return s.q.UpsertTaskHold(ctx, sqlitedb.UpsertTaskHoldParams(p))
}

func (s *SqliteStore) DeleteTaskHold(ctx context.Context, p DeleteTaskHoldParams) (int64, error) {
	return s.q.DeleteTaskHold(ctx, sqlitedb.DeleteTaskHoldParams(p))
}

func (s *SqliteStore) DeleteTaskHolds(ctx context.Context, taskID string) (int64, error) {
	return s.q.DeleteTaskHolds(ctx, taskID)
}

func (s *SqliteStore) ListTaskHolds(
	ctx context.Context,
	taskID string,
) ([]ListTaskHoldsRow, error) {
	rows, err := s.q.ListTaskHolds(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromSqliteListTaskHoldsRowSlice(rows), nil
}

func (s *SqliteStore) ListAllTaskHolds(ctx context.Context) ([]ListAllTaskHoldsRow, error) {
	rows, err := s.q.ListAllTaskHolds(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteListAllTaskHoldsRowSlice(rows), nil
}

func (s *SqliteStore) CountTaskHolds(ctx context.Context, taskID string) (int64, error) {
	return s.q.CountTaskHolds(ctx, taskID)
}

func (s *SqliteStore) ListTaskHoldTaskIDsByPlugin(
	ctx context.Context,
	pluginID string,
) ([]string, error) {
	return s.q.ListTaskHoldTaskIDsByPlugin(ctx, pluginID)
}

func (s *PgStore) UpsertTaskHold(ctx context.Context, p UpsertTaskHoldParams) error {
	return s.q.UpsertTaskHold(ctx, pgdb.UpsertTaskHoldParams(p))
}

func (s *PgStore) DeleteTaskHold(ctx context.Context, p DeleteTaskHoldParams) (int64, error) {
	return s.q.DeleteTaskHold(ctx, pgdb.DeleteTaskHoldParams(p))
}

func (s *PgStore) DeleteTaskHolds(ctx context.Context, taskID string) (int64, error) {
	return s.q.DeleteTaskHolds(ctx, taskID)
}

func (s *PgStore) ListTaskHolds(ctx context.Context, taskID string) ([]ListTaskHoldsRow, error) {
	rows, err := s.q.ListTaskHolds(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return fromPgListTaskHoldsRowSlice(rows), nil
}

func (s *PgStore) ListAllTaskHolds(ctx context.Context) ([]ListAllTaskHoldsRow, error) {
	rows, err := s.q.ListAllTaskHolds(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgListAllTaskHoldsRowSlice(rows), nil
}

func (s *PgStore) CountTaskHolds(ctx context.Context, taskID string) (int64, error) {
	return s.q.CountTaskHolds(ctx, taskID)
}

func (s *PgStore) ListTaskHoldTaskIDsByPlugin(
	ctx context.Context,
	pluginID string,
) ([]string, error) {
	return s.q.ListTaskHoldTaskIDsByPlugin(ctx, pluginID)
}
