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

type AgentConfigStore interface {
	ListModelProviders(ctx context.Context) ([]ModelProvider, error)
	GetModelProvider(ctx context.Context, slug string) (ModelProvider, error)
	UpsertModelProvider(ctx context.Context, p UpsertModelProviderParams) error

	SaveModelProvider(ctx context.Context, p SaveModelProviderParams) error

	DeleteModelProvider(ctx context.Context, slug, revision string) (bool, error)

	ListModels(ctx context.Context) ([]Model, error)
	UpsertModel(ctx context.Context, p UpsertModelParams) error
	DeleteModelsForProvider(ctx context.Context, providerID int64) error

	ListModelGrants(ctx context.Context) ([]ModelGrant, error)
	UpsertModelGrant(ctx context.Context, p UpsertModelGrantParams) error
	DeleteModelGrant(ctx context.Context, p DeleteModelGrantParams) error
	DeleteModelGrantsForProvider(ctx context.Context, providerID int64) error

	AgentConfigRevision(ctx context.Context) (string, error)
	BumpAgentConfigRevision(ctx context.Context, revision string) error

	ListAgentGroupIDsForAgentActor(ctx context.Context, actorID int64) ([]int64, error)

	ListTeamIDsForAgentActor(ctx context.Context, actorID int64) ([]int64, error)
}

type SaveModelProviderParams struct {
	Provider UpsertModelProviderParams
	Models   []UpsertModelParams
	Grants   []UpsertModelGrantParams
	Revision string
}

func (s *SqliteStore) ListModelProviders(ctx context.Context) ([]ModelProvider, error) {
	rows, err := s.q.ListModelProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteModelProviderSlice(rows), nil
}

func (s *SqliteStore) GetModelProvider(ctx context.Context, slug string) (ModelProvider, error) {
	row, err := s.q.GetModelProvider(ctx, slug)
	if err != nil {
		return ModelProvider{}, err
	}
	return fromSqliteModelProvider(row), nil
}

func (s *SqliteStore) UpsertModelProvider(ctx context.Context, p UpsertModelProviderParams) error {
	return s.q.UpsertModelProvider(ctx, sqlitedb.UpsertModelProviderParams(p))
}

func (s *SqliteStore) SaveModelProvider(ctx context.Context, p SaveModelProviderParams) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if err := q.UpsertModelProvider(
			ctx,
			sqlitedb.UpsertModelProviderParams(p.Provider),
		); err != nil {
			return err
		}
		provider, err := q.GetModelProvider(ctx, p.Provider.Slug)
		if err != nil {
			return err
		}
		if err := q.DeleteModelsForProvider(ctx, provider.ID); err != nil {
			return err
		}
		for _, model := range p.Models {
			model.ProviderID = provider.ID
			if err := q.UpsertModel(ctx, sqlitedb.UpsertModelParams(model)); err != nil {
				return err
			}
		}
		if err := q.DeleteModelGrantsForProvider(ctx, provider.ID); err != nil {
			return err
		}
		for _, grant := range p.Grants {
			grant.ProviderID = provider.ID
			if err := q.UpsertModelGrant(ctx, sqlitedb.UpsertModelGrantParams(grant)); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *SqliteStore) DeleteModelProvider(
	ctx context.Context,
	slug, revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		rows, err := q.DeleteModelProvider(ctx, slug)
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

func (s *SqliteStore) ListModels(ctx context.Context) ([]Model, error) {
	rows, err := s.q.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteModelSlice(rows), nil
}

func (s *SqliteStore) UpsertModel(ctx context.Context, p UpsertModelParams) error {
	return s.q.UpsertModel(ctx, sqlitedb.UpsertModelParams(p))
}

func (s *SqliteStore) DeleteModelsForProvider(ctx context.Context, providerID int64) error {
	return s.q.DeleteModelsForProvider(ctx, providerID)
}

func (s *SqliteStore) ListModelGrants(ctx context.Context) ([]ModelGrant, error) {
	rows, err := s.q.ListModelGrants(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteModelGrantSlice(rows), nil
}

func (s *SqliteStore) UpsertModelGrant(ctx context.Context, p UpsertModelGrantParams) error {
	return s.q.UpsertModelGrant(ctx, sqlitedb.UpsertModelGrantParams(p))
}

func (s *SqliteStore) DeleteModelGrant(ctx context.Context, p DeleteModelGrantParams) error {
	return s.q.DeleteModelGrant(ctx, sqlitedb.DeleteModelGrantParams(p))
}

func (s *SqliteStore) DeleteModelGrantsForProvider(ctx context.Context, providerID int64) error {
	return s.q.DeleteModelGrantsForProvider(ctx, providerID)
}

func (s *SqliteStore) ListAgentGroupIDsForAgentActor(
	ctx context.Context,
	actorID int64,
) ([]int64, error) {
	return s.q.ListAgentGroupIDsForAgentActor(ctx, actorID)
}

func (s *SqliteStore) AgentConfigRevision(ctx context.Context) (string, error) {
	return s.q.GetAgentConfigRevision(ctx)
}

func (s *SqliteStore) BumpAgentConfigRevision(ctx context.Context, revision string) error {
	return s.q.BumpAgentConfigRevision(ctx, revision)
}

func (s *PgStore) ListModelProviders(ctx context.Context) ([]ModelProvider, error) {
	rows, err := s.q.ListModelProviders(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgModelProviderSlice(rows), nil
}

func (s *PgStore) GetModelProvider(ctx context.Context, slug string) (ModelProvider, error) {
	row, err := s.q.GetModelProvider(ctx, slug)
	if err != nil {
		return ModelProvider{}, err
	}
	return fromPgModelProvider(row), nil
}

func (s *PgStore) UpsertModelProvider(ctx context.Context, p UpsertModelProviderParams) error {
	return s.q.UpsertModelProvider(ctx, pgdb.UpsertModelProviderParams(p))
}

func (s *PgStore) SaveModelProvider(ctx context.Context, p SaveModelProviderParams) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.UpsertModelProvider(
			ctx,
			pgdb.UpsertModelProviderParams(p.Provider),
		); err != nil {
			return err
		}
		provider, err := q.GetModelProvider(ctx, p.Provider.Slug)
		if err != nil {
			return err
		}
		if err := q.DeleteModelsForProvider(ctx, provider.ID); err != nil {
			return err
		}
		for _, model := range p.Models {
			model.ProviderID = provider.ID
			if err := q.UpsertModel(ctx, pgdb.UpsertModelParams(model)); err != nil {
				return err
			}
		}
		if err := q.DeleteModelGrantsForProvider(ctx, provider.ID); err != nil {
			return err
		}
		for _, grant := range p.Grants {
			grant.ProviderID = provider.ID
			if err := q.UpsertModelGrant(ctx, pgdb.UpsertModelGrantParams(grant)); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) DeleteModelProvider(
	ctx context.Context,
	slug, revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		rows, err := q.DeleteModelProvider(ctx, slug)
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

func (s *PgStore) ListModels(ctx context.Context) ([]Model, error) {
	rows, err := s.q.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgModelSlice(rows), nil
}

func (s *PgStore) UpsertModel(ctx context.Context, p UpsertModelParams) error {
	return s.q.UpsertModel(ctx, pgdb.UpsertModelParams(p))
}

func (s *PgStore) DeleteModelsForProvider(ctx context.Context, providerID int64) error {
	return s.q.DeleteModelsForProvider(ctx, providerID)
}

func (s *PgStore) ListModelGrants(ctx context.Context) ([]ModelGrant, error) {
	rows, err := s.q.ListModelGrants(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgModelGrantSlice(rows), nil
}

func (s *PgStore) UpsertModelGrant(ctx context.Context, p UpsertModelGrantParams) error {
	return s.q.UpsertModelGrant(ctx, pgdb.UpsertModelGrantParams(p))
}

func (s *PgStore) DeleteModelGrant(ctx context.Context, p DeleteModelGrantParams) error {
	return s.q.DeleteModelGrant(ctx, pgdb.DeleteModelGrantParams(p))
}

func (s *PgStore) DeleteModelGrantsForProvider(ctx context.Context, providerID int64) error {
	return s.q.DeleteModelGrantsForProvider(ctx, providerID)
}

func (s *PgStore) ListAgentGroupIDsForAgentActor(
	ctx context.Context,
	actorID int64,
) ([]int64, error) {
	return s.q.ListAgentGroupIDsForAgentActor(ctx, actorID)
}

func (s *PgStore) AgentConfigRevision(ctx context.Context) (string, error) {
	return s.q.GetAgentConfigRevision(ctx)
}

func (s *PgStore) BumpAgentConfigRevision(ctx context.Context, revision string) error {
	return s.q.BumpAgentConfigRevision(ctx, revision)
}
