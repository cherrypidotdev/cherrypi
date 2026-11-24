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

type IntegrationsStore interface {
	ListMcpServers(ctx context.Context) ([]McpServer, error)
	GetMcpServer(ctx context.Context, id int64) (McpServer, error)

	GetMcpServerByName(ctx context.Context, name string) (McpServer, error)

	SaveMcpServer(ctx context.Context, p SaveMcpServerParams) (McpServer, error)
	DeleteMcpServer(ctx context.Context, id int64, revision string) (bool, error)

	ListSkillEntries(ctx context.Context) ([]Skill, error)
	GetSkillEntry(ctx context.Context, id int64) (Skill, error)
	SaveSkillEntry(ctx context.Context, p SaveSkillEntryParams) (Skill, error)
	DeleteSkillEntry(ctx context.Context, id int64, revision string) (bool, error)

	ListMcpInstalls(ctx context.Context) ([]McpInstall, error)
	ListSkillInstalls(ctx context.Context) ([]SkillInstall, error)

	SetMcpInstall(ctx context.Context, p SetInstallParams) error
	RemoveMcpInstall(ctx context.Context, p RemoveInstallParams) error
	SetSkillInstall(ctx context.Context, p SetInstallParams) error
	RemoveSkillInstall(ctx context.Context, p RemoveInstallParams) error

	ListGitCredentials(ctx context.Context) ([]GitCredential, error)
	GetGitCredential(ctx context.Context, id int64) (GitCredential, error)
	CountGitCredentialSources(ctx context.Context, id int64) (int64, error)
	CreateGitCredential(ctx context.Context, p GitCredentialParams) (GitCredential, error)
	UpdateGitCredential(ctx context.Context, p GitCredentialParams) (GitCredential, error)
	DeleteGitCredential(ctx context.Context, id int64) (bool, error)

	ListSkillSources(ctx context.Context) ([]SkillSource, error)
	GetSkillSource(ctx context.Context, id int64) (SkillSource, error)

	ListSkillsBySource(ctx context.Context, sourceID int64) ([]Skill, error)
	SaveSkillSource(ctx context.Context, p SaveSkillSourceParams) (SkillSource, error)
	DeleteSkillSource(ctx context.Context, id int64, revision string) (bool, error)

	SetSkillSourceCheck(ctx context.Context, p SkillSourceCheckParams) error
	SetSkillSourceApplied(ctx context.Context, p SkillSourceAppliedParams) error

	ListSkillSourceInstalls(ctx context.Context) ([]SkillSourceInstall, error)
	SetSkillSourceInstall(ctx context.Context, p SetInstallParams) error
	RemoveSkillSourceInstall(ctx context.Context, p RemoveInstallParams) error

	ApplySkillSource(ctx context.Context, p ApplySkillSourceParams) (ApplySkillSourceResult, error)

	DetachSkill(ctx context.Context, id int64, updatedAt int64, revision string) error

	GetTaskIntegrationReport(ctx context.Context, taskID string) (TaskIntegrationReport, error)
	PutTaskIntegrationReport(ctx context.Context, p PutTaskIntegrationReportParams) error
}

type SaveMcpServerParams struct {
	Entry        McpServerParams
	Installs     []InstallParams
	KeepInstalls bool
	Revision     string
}

type SetInstallParams struct {
	EntryID          int64
	ScopeKind        string
	ScopeID          string
	Enabled          int64
	CreatedByActorID int64
	CreatedAt        int64
	Revision         string
}

type RemoveInstallParams struct {
	EntryID   int64
	ScopeKind string
	ScopeID   string
	Revision  string
}

type SaveSkillEntryParams struct {
	Entry        SkillParams
	Installs     []InstallParams
	KeepInstalls bool
	Revision     string
}

type McpServerParams struct {
	ID               int64
	Name             string
	Description      string
	OwnerKind        string
	OwnerID          string
	Url              string
	HeadersJson      string
	ConnectTimeoutMs int64
	RequestTimeoutMs int64
	ToolsJson        string
	DirectTools      int64
	Enabled          int64
	CreatedByActorID int64
	CreatedAt        int64
	UpdatedAt        int64
}

type SkillParams struct {
	ID                     int64
	Name                   string
	Description            string
	Body                   string
	FilesJson              string
	DisableModelInvocation int64
	OwnerKind              string
	OwnerID                string
	Enabled                int64
	CreatedByActorID       int64
	CreatedAt              int64
	UpdatedAt              int64
}

type InstallParams struct {
	EntryID          int64
	ScopeKind        string
	ScopeID          string
	Enabled          int64
	CreatedByActorID int64
	CreatedAt        int64
}

type PutTaskIntegrationReportParams struct {
	TaskID         string
	PayloadJson    string
	ConfigRevision string
	ReportedAt     int64
}

func (s *SqliteStore) ListMcpServers(ctx context.Context) ([]McpServer, error) {
	rows, err := s.q.ListMcpServers(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteMcpServerSlice(rows), nil
}

func (s *SqliteStore) GetMcpServer(ctx context.Context, id int64) (McpServer, error) {
	row, err := s.q.GetMcpServer(ctx, id)
	if err != nil {
		return McpServer{}, err
	}
	return fromSqliteMcpServer(row), nil
}

func (s *SqliteStore) GetMcpServerByName(
	ctx context.Context,
	name string,
) (McpServer, error) {
	row, err := s.q.GetMcpServerByName(ctx, name)
	if err != nil {
		return McpServer{}, err
	}
	return fromSqliteMcpServer(row), nil
}

func (s *SqliteStore) SaveMcpServer(
	ctx context.Context,
	p SaveMcpServerParams,
) (McpServer, error) {
	var saved McpServer
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		row, err := saveMcpServerSQLite(ctx, q, p.Entry)
		if err != nil {
			return err
		}
		saved = fromSqliteMcpServer(row)
		if !p.KeepInstalls {
			if err := replaceMcpInstallsSQLite(ctx, q, saved.ID, p.Installs); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return saved, err
}

func saveMcpServerSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	p McpServerParams,
) (sqlitedb.McpServer, error) {
	if p.ID != 0 {
		row, err := q.UpdateMcpServer(ctx, sqlitedb.UpdateMcpServerParams{
			Name:             p.Name,
			Description:      p.Description,
			OwnerKind:        p.OwnerKind,
			OwnerID:          p.OwnerID,
			Url:              p.Url,
			HeadersJson:      p.HeadersJson,
			ConnectTimeoutMs: p.ConnectTimeoutMs,
			RequestTimeoutMs: p.RequestTimeoutMs,
			ToolsJson:        p.ToolsJson,
			DirectTools:      p.DirectTools,
			Enabled:          p.Enabled,
			UpdatedAt:        p.UpdatedAt,
			ID:               p.ID,
		})
		return row, err
	}
	return q.CreateMcpServer(ctx, sqlitedb.CreateMcpServerParams{
		Name:             p.Name,
		Description:      p.Description,
		OwnerKind:        p.OwnerKind,
		OwnerID:          p.OwnerID,
		Url:              p.Url,
		HeadersJson:      p.HeadersJson,
		ConnectTimeoutMs: p.ConnectTimeoutMs,
		RequestTimeoutMs: p.RequestTimeoutMs,
		ToolsJson:        p.ToolsJson,
		DirectTools:      p.DirectTools,
		Enabled:          p.Enabled,
		CreatedByActorID: p.CreatedByActorID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
}

func replaceMcpInstallsSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	entryID int64,
	installs []InstallParams,
) error {
	existing, err := q.ListMcpInstalls(ctx)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.EntryID != entryID {
			continue
		}
		if _, err := q.DeleteMcpInstall(ctx, sqlitedb.DeleteMcpInstallParams{
			EntryID:   entryID,
			ScopeKind: row.ScopeKind,
			ScopeID:   row.ScopeID,
		}); err != nil {
			return err
		}
	}
	for _, install := range installs {
		install.EntryID = entryID
		if err := q.UpsertMcpInstall(ctx, sqlitedb.UpsertMcpInstallParams{
			EntryID:          entryID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: install.CreatedByActorID,
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) DeleteMcpServer(
	ctx context.Context,
	id int64,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		rows, err := q.DeleteMcpServer(ctx, id)
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

func (s *SqliteStore) ListSkillEntries(ctx context.Context) ([]Skill, error) {
	rows, err := s.q.ListSkillEntries(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteSkillSlice(rows), nil
}

func (s *SqliteStore) GetSkillEntry(ctx context.Context, id int64) (Skill, error) {
	row, err := s.q.GetSkillEntry(ctx, id)
	if err != nil {
		return Skill{}, err
	}
	return fromSqliteSkill(row), nil
}

func (s *SqliteStore) SaveSkillEntry(
	ctx context.Context,
	p SaveSkillEntryParams,
) (Skill, error) {
	var saved Skill
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		row, err := saveSkillSQLite(ctx, q, p.Entry)
		if err != nil {
			return err
		}
		saved = fromSqliteSkill(row)
		if !p.KeepInstalls {
			if err := replaceSkillInstallsSQLite(ctx, q, saved.ID, p.Installs); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return saved, err
}

func saveSkillSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	p SkillParams,
) (sqlitedb.Skill, error) {
	if p.ID != 0 {
		return q.UpdateSkillEntry(ctx, sqlitedb.UpdateSkillEntryParams{
			Name:                   p.Name,
			Description:            p.Description,
			Body:                   p.Body,
			FilesJson:              p.FilesJson,
			DisableModelInvocation: p.DisableModelInvocation,
			OwnerKind:              p.OwnerKind,
			OwnerID:                p.OwnerID,
			Enabled:                p.Enabled,
			UpdatedAt:              p.UpdatedAt,
			ID:                     p.ID,
		})
	}
	return q.CreateSkillEntry(ctx, sqlitedb.CreateSkillEntryParams{
		Name:                   p.Name,
		Description:            p.Description,
		Body:                   p.Body,
		FilesJson:              p.FilesJson,
		DisableModelInvocation: p.DisableModelInvocation,
		OwnerKind:              p.OwnerKind,
		OwnerID:                p.OwnerID,
		Enabled:                p.Enabled,
		CreatedByActorID:       p.CreatedByActorID,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
	})
}

func replaceSkillInstallsSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	entryID int64,
	installs []InstallParams,
) error {
	existing, err := q.ListSkillInstalls(ctx)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.EntryID != entryID {
			continue
		}
		if _, err := q.DeleteSkillInstall(ctx, sqlitedb.DeleteSkillInstallParams{
			EntryID:   entryID,
			ScopeKind: row.ScopeKind,
			ScopeID:   row.ScopeID,
		}); err != nil {
			return err
		}
	}
	for _, install := range installs {
		if err := q.UpsertSkillInstall(ctx, sqlitedb.UpsertSkillInstallParams{
			EntryID:          entryID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: install.CreatedByActorID,
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) DeleteSkillEntry(
	ctx context.Context,
	id int64,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		rows, err := q.DeleteSkillEntry(ctx, id)
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

func (s *SqliteStore) ListMcpInstalls(ctx context.Context) ([]McpInstall, error) {
	rows, err := s.q.ListMcpInstalls(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteMcpInstallSlice(rows), nil
}

func (s *SqliteStore) ListSkillInstalls(ctx context.Context) ([]SkillInstall, error) {
	rows, err := s.q.ListSkillInstalls(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteSkillInstallSlice(rows), nil
}

func (s *SqliteStore) GetTaskIntegrationReport(
	ctx context.Context,
	taskID string,
) (TaskIntegrationReport, error) {
	row, err := s.q.GetTaskIntegrationReport(ctx, taskID)
	if err != nil {
		return TaskIntegrationReport{}, err
	}
	return fromSqliteTaskIntegrationReport(row), nil
}

func (s *SqliteStore) PutTaskIntegrationReport(
	ctx context.Context,
	p PutTaskIntegrationReportParams,
) error {
	return s.q.UpsertTaskIntegrationReport(
		ctx,
		sqlitedb.UpsertTaskIntegrationReportParams(p),
	)
}

func (s *SqliteStore) SetMcpInstall(ctx context.Context, p SetInstallParams) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if err := q.UpsertMcpInstall(ctx, sqlitedb.UpsertMcpInstallParams{
			EntryID:          p.EntryID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *SqliteStore) RemoveMcpInstall(ctx context.Context, p RemoveInstallParams) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if _, err := q.DeleteMcpInstall(ctx, sqlitedb.DeleteMcpInstallParams{
			EntryID:   p.EntryID,
			ScopeKind: p.ScopeKind,
			ScopeID:   p.ScopeID,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *SqliteStore) SetSkillInstall(ctx context.Context, p SetInstallParams) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if err := q.UpsertSkillInstall(ctx, sqlitedb.UpsertSkillInstallParams{
			EntryID:          p.EntryID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *SqliteStore) RemoveSkillInstall(ctx context.Context, p RemoveInstallParams) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if _, err := q.DeleteSkillInstall(ctx, sqlitedb.DeleteSkillInstallParams{
			EntryID:   p.EntryID,
			ScopeKind: p.ScopeKind,
			ScopeID:   p.ScopeID,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) ListMcpServers(ctx context.Context) ([]McpServer, error) {
	rows, err := s.q.ListMcpServers(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgMcpServerSlice(rows), nil
}

func (s *PgStore) GetMcpServer(ctx context.Context, id int64) (McpServer, error) {
	row, err := s.q.GetMcpServer(ctx, id)
	if err != nil {
		return McpServer{}, err
	}
	return fromPgMcpServer(row), nil
}

func (s *PgStore) GetMcpServerByName(
	ctx context.Context,
	name string,
) (McpServer, error) {
	row, err := s.q.GetMcpServerByName(ctx, name)
	if err != nil {
		return McpServer{}, err
	}
	return fromPgMcpServer(row), nil
}

func (s *PgStore) SaveMcpServer(
	ctx context.Context,
	p SaveMcpServerParams,
) (McpServer, error) {
	var saved McpServer
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		row, err := saveMcpServerPg(ctx, q, p.Entry)
		if err != nil {
			return err
		}
		saved = fromPgMcpServer(row)
		if !p.KeepInstalls {
			if err := replaceMcpInstallsPg(ctx, q, saved.ID, p.Installs); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return saved, err
}

func saveMcpServerPg(
	ctx context.Context,
	q *pgdb.Queries,
	p McpServerParams,
) (pgdb.McpServer, error) {
	if p.ID != 0 {
		return q.UpdateMcpServer(ctx, pgdb.UpdateMcpServerParams{
			Name:             p.Name,
			Description:      p.Description,
			OwnerKind:        p.OwnerKind,
			OwnerID:          p.OwnerID,
			Url:              p.Url,
			HeadersJson:      p.HeadersJson,
			ConnectTimeoutMs: p.ConnectTimeoutMs,
			RequestTimeoutMs: p.RequestTimeoutMs,
			ToolsJson:        p.ToolsJson,
			DirectTools:      p.DirectTools,
			Enabled:          p.Enabled,
			UpdatedAt:        p.UpdatedAt,
			ID:               p.ID,
		})
	}
	return q.CreateMcpServer(ctx, pgdb.CreateMcpServerParams{
		Name:             p.Name,
		Description:      p.Description,
		OwnerKind:        p.OwnerKind,
		OwnerID:          p.OwnerID,
		Url:              p.Url,
		HeadersJson:      p.HeadersJson,
		ConnectTimeoutMs: p.ConnectTimeoutMs,
		RequestTimeoutMs: p.RequestTimeoutMs,
		ToolsJson:        p.ToolsJson,
		DirectTools:      p.DirectTools,
		Enabled:          p.Enabled,
		CreatedByActorID: p.CreatedByActorID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
}

func replaceMcpInstallsPg(
	ctx context.Context,
	q *pgdb.Queries,
	entryID int64,
	installs []InstallParams,
) error {
	existing, err := q.ListMcpInstalls(ctx)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.EntryID != entryID {
			continue
		}
		if _, err := q.DeleteMcpInstall(ctx, pgdb.DeleteMcpInstallParams{
			EntryID:   entryID,
			ScopeKind: row.ScopeKind,
			ScopeID:   row.ScopeID,
		}); err != nil {
			return err
		}
	}
	for _, install := range installs {
		if err := q.UpsertMcpInstall(ctx, pgdb.UpsertMcpInstallParams{
			EntryID:          entryID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: install.CreatedByActorID,
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *PgStore) DeleteMcpServer(
	ctx context.Context,
	id int64,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		rows, err := q.DeleteMcpServer(ctx, id)
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

func (s *PgStore) ListSkillEntries(ctx context.Context) ([]Skill, error) {
	rows, err := s.q.ListSkillEntries(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgSkillSlice(rows), nil
}

func (s *PgStore) GetSkillEntry(ctx context.Context, id int64) (Skill, error) {
	row, err := s.q.GetSkillEntry(ctx, id)
	if err != nil {
		return Skill{}, err
	}
	return fromPgSkill(row), nil
}

func (s *PgStore) SaveSkillEntry(
	ctx context.Context,
	p SaveSkillEntryParams,
) (Skill, error) {
	var saved Skill
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		row, err := saveSkillPg(ctx, q, p.Entry)
		if err != nil {
			return err
		}
		saved = fromPgSkill(row)
		if !p.KeepInstalls {
			if err := replaceSkillInstallsPg(ctx, q, saved.ID, p.Installs); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return saved, err
}

func saveSkillPg(
	ctx context.Context,
	q *pgdb.Queries,
	p SkillParams,
) (pgdb.Skill, error) {
	if p.ID != 0 {
		return q.UpdateSkillEntry(ctx, pgdb.UpdateSkillEntryParams{
			Name:                   p.Name,
			Description:            p.Description,
			Body:                   p.Body,
			FilesJson:              p.FilesJson,
			DisableModelInvocation: p.DisableModelInvocation,
			OwnerKind:              p.OwnerKind,
			OwnerID:                p.OwnerID,
			Enabled:                p.Enabled,
			UpdatedAt:              p.UpdatedAt,
			ID:                     p.ID,
		})
	}
	return q.CreateSkillEntry(ctx, pgdb.CreateSkillEntryParams{
		Name:                   p.Name,
		Description:            p.Description,
		Body:                   p.Body,
		FilesJson:              p.FilesJson,
		DisableModelInvocation: p.DisableModelInvocation,
		OwnerKind:              p.OwnerKind,
		OwnerID:                p.OwnerID,
		Enabled:                p.Enabled,
		CreatedByActorID:       p.CreatedByActorID,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
	})
}

func replaceSkillInstallsPg(
	ctx context.Context,
	q *pgdb.Queries,
	entryID int64,
	installs []InstallParams,
) error {
	existing, err := q.ListSkillInstalls(ctx)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.EntryID != entryID {
			continue
		}
		if _, err := q.DeleteSkillInstall(ctx, pgdb.DeleteSkillInstallParams{
			EntryID:   entryID,
			ScopeKind: row.ScopeKind,
			ScopeID:   row.ScopeID,
		}); err != nil {
			return err
		}
	}
	for _, install := range installs {
		if err := q.UpsertSkillInstall(ctx, pgdb.UpsertSkillInstallParams{
			EntryID:          entryID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: install.CreatedByActorID,
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *PgStore) DeleteSkillEntry(
	ctx context.Context,
	id int64,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		rows, err := q.DeleteSkillEntry(ctx, id)
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

func (s *PgStore) ListMcpInstalls(ctx context.Context) ([]McpInstall, error) {
	rows, err := s.q.ListMcpInstalls(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgMcpInstallSlice(rows), nil
}

func (s *PgStore) ListSkillInstalls(ctx context.Context) ([]SkillInstall, error) {
	rows, err := s.q.ListSkillInstalls(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgSkillInstallSlice(rows), nil
}

func (s *PgStore) GetTaskIntegrationReport(
	ctx context.Context,
	taskID string,
) (TaskIntegrationReport, error) {
	row, err := s.q.GetTaskIntegrationReport(ctx, taskID)
	if err != nil {
		return TaskIntegrationReport{}, err
	}
	return fromPgTaskIntegrationReport(row), nil
}

func (s *PgStore) SetMcpInstall(ctx context.Context, p SetInstallParams) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.UpsertMcpInstall(ctx, pgdb.UpsertMcpInstallParams{
			EntryID:          p.EntryID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) RemoveMcpInstall(ctx context.Context, p RemoveInstallParams) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if _, err := q.DeleteMcpInstall(ctx, pgdb.DeleteMcpInstallParams{
			EntryID:   p.EntryID,
			ScopeKind: p.ScopeKind,
			ScopeID:   p.ScopeID,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) SetSkillInstall(ctx context.Context, p SetInstallParams) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.UpsertSkillInstall(ctx, pgdb.UpsertSkillInstallParams{
			EntryID:          p.EntryID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) RemoveSkillInstall(ctx context.Context, p RemoveInstallParams) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if _, err := q.DeleteSkillInstall(ctx, pgdb.DeleteSkillInstallParams{
			EntryID:   p.EntryID,
			ScopeKind: p.ScopeKind,
			ScopeID:   p.ScopeID,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) PutTaskIntegrationReport(
	ctx context.Context,
	p PutTaskIntegrationReportParams,
) error {
	return s.q.UpsertTaskIntegrationReport(
		ctx,
		pgdb.UpsertTaskIntegrationReportParams(p),
	)
}
