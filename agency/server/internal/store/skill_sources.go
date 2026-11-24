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
	"database/sql"
	"errors"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
)

type GitCredentialParams struct {
	ID               int64
	Name             string
	Kind             string
	Host             string
	Port             int64
	Username         string
	SecretSealed     []byte
	KnownHosts       string
	PublicKey        string
	Fingerprint      string
	CreatedByActorID int64
	CreatedAt        int64
	UpdatedAt        int64
}

type SkillSourceParams struct {
	ID               int64
	Slug             string
	Name             string
	Description      string
	OwnerKind        string
	OwnerID          string
	RepoURL          string
	Ref              string
	PathGlob         string
	CredentialID     int64
	UpdateMode       string
	PollIntervalS    int64
	Paused           int64
	CreatedByActorID int64
	CreatedAt        int64
	UpdatedAt        int64
}

type SaveSkillSourceParams struct {
	Entry        SkillSourceParams
	Installs     []InstallParams
	KeepInstalls bool
	Revision     string
}

type SkillSourceCheckParams struct {
	SourceID      int64
	CheckedAt     int64
	SyncState     string
	SyncError     string
	PendingCommit string
	PendingJSON   string

	ProblemsJSON string
}

type SkillSourceAppliedParams struct {
	SourceID  int64
	Commit    string
	AppliedAt int64
	CheckedAt int64

	SyncError string

	ProblemsJSON string
}

type SkillSourceSkillParams struct {
	Path                   string
	Name                   string
	Description            string
	Body                   string
	FilesJSON              string
	DisableModelInvocation int64
}

type ApplySkillSourceParams struct {
	SourceID         int64
	Commit           string
	OwnerKind        string
	OwnerID          string
	Skills           []SkillSourceSkillParams
	Installs         []InstallParams
	CreatedByActorID int64
	Now              int64
	Revision         string
}

type SkillSourceConflict struct {
	Path   string
	Name   string
	Reason string
}

type ApplySkillSourceResult struct {
	Created   int
	Updated   int
	Missing   int
	Conflicts []SkillSourceConflict
}

func (s *SqliteStore) ListGitCredentials(
	ctx context.Context,
) ([]GitCredential, error) {
	rows, err := s.q.ListGitCredentials(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteGitCredentialSlice(rows), nil
}

func (s *SqliteStore) GetGitCredential(
	ctx context.Context,
	id int64,
) (GitCredential, error) {
	row, err := s.q.GetGitCredential(ctx, id)
	if err != nil {
		return GitCredential{}, err
	}
	return fromSqliteGitCredential(row), nil
}

func (s *SqliteStore) CountGitCredentialSources(
	ctx context.Context,
	id int64,
) (int64, error) {
	return s.q.CountGitCredentialSources(ctx, sql.NullInt64{Int64: id, Valid: true})
}

func (s *SqliteStore) CreateGitCredential(
	ctx context.Context,
	p GitCredentialParams,
) (GitCredential, error) {
	row, err := s.q.CreateGitCredential(ctx, sqlitedb.CreateGitCredentialParams{
		Name:             p.Name,
		Kind:             p.Kind,
		Host:             p.Host,
		Port:             p.Port,
		Username:         p.Username,
		SecretSealed:     p.SecretSealed,
		KnownHosts:       p.KnownHosts,
		PublicKey:        p.PublicKey,
		Fingerprint:      p.Fingerprint,
		CreatedByActorID: p.CreatedByActorID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
	if err != nil {
		return GitCredential{}, err
	}
	return fromSqliteGitCredential(row), nil
}

func (s *SqliteStore) UpdateGitCredential(
	ctx context.Context,
	p GitCredentialParams,
) (GitCredential, error) {
	row, err := s.q.UpdateGitCredential(ctx, sqlitedb.UpdateGitCredentialParams{
		Name:         p.Name,
		Kind:         p.Kind,
		Host:         p.Host,
		Port:         p.Port,
		Username:     p.Username,
		SecretSealed: p.SecretSealed,
		KnownHosts:   p.KnownHosts,
		PublicKey:    p.PublicKey,
		Fingerprint:  p.Fingerprint,
		UpdatedAt:    p.UpdatedAt,
		ID:           p.ID,
	})
	if err != nil {
		return GitCredential{}, err
	}
	return fromSqliteGitCredential(row), nil
}

func (s *SqliteStore) DeleteGitCredential(ctx context.Context, id int64) (bool, error) {
	rows, err := s.q.DeleteGitCredential(ctx, id)
	return rows > 0, err
}

func (s *SqliteStore) ListSkillSources(ctx context.Context) ([]SkillSource, error) {
	rows, err := s.q.ListSkillSources(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteSkillSourceSlice(rows), nil
}

func (s *SqliteStore) GetSkillSource(
	ctx context.Context,
	id int64,
) (SkillSource, error) {
	row, err := s.q.GetSkillSource(ctx, id)
	if err != nil {
		return SkillSource{}, err
	}
	return fromSqliteSkillSource(row), nil
}

func (s *SqliteStore) ListSkillsBySource(
	ctx context.Context,
	sourceID int64,
) ([]Skill, error) {
	rows, err := s.q.ListSkillsBySource(ctx, sql.NullInt64{Int64: sourceID, Valid: true})
	if err != nil {
		return nil, err
	}
	return fromSqliteSkillSlice(rows), nil
}

func (s *SqliteStore) SaveSkillSource(
	ctx context.Context,
	p SaveSkillSourceParams,
) (SkillSource, error) {
	var saved SkillSource
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		row, err := saveSkillSourceSQLite(ctx, q, p.Entry)
		if err != nil {
			return err
		}
		saved = fromSqliteSkillSource(row)
		if !p.KeepInstalls {
			if err := replaceSourceInstallsSQLite(ctx, q, saved.ID, p.Installs); err != nil {
				return err
			}
		}

		if err := q.ReownSourcedSkills(ctx, sqlitedb.ReownSourcedSkillsParams{
			OwnerKind: saved.OwnerKind,
			OwnerID:   saved.OwnerID,
			SourceID:  sql.NullInt64{Int64: saved.ID, Valid: true},
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return saved, err
}

func saveSkillSourceSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	p SkillSourceParams,
) (sqlitedb.SkillSource, error) {
	credential := sql.NullInt64{}
	if p.CredentialID != 0 {
		credential = sql.NullInt64{Int64: p.CredentialID, Valid: true}
	}
	if p.ID != 0 {
		return q.UpdateSkillSource(ctx, sqlitedb.UpdateSkillSourceParams{
			Slug:          p.Slug,
			Name:          p.Name,
			Description:   p.Description,
			OwnerKind:     p.OwnerKind,
			OwnerID:       p.OwnerID,
			RepoUrl:       p.RepoURL,
			Ref:           p.Ref,
			PathGlob:      p.PathGlob,
			CredentialID:  credential,
			UpdateMode:    p.UpdateMode,
			PollIntervalS: p.PollIntervalS,
			Paused:        p.Paused,
			UpdatedAt:     p.UpdatedAt,
			ID:            p.ID,
		})
	}
	return q.CreateSkillSource(ctx, sqlitedb.CreateSkillSourceParams{
		Slug:             p.Slug,
		Name:             p.Name,
		Description:      p.Description,
		OwnerKind:        p.OwnerKind,
		OwnerID:          p.OwnerID,
		RepoUrl:          p.RepoURL,
		Ref:              p.Ref,
		PathGlob:         p.PathGlob,
		CredentialID:     credential,
		UpdateMode:       p.UpdateMode,
		PollIntervalS:    p.PollIntervalS,
		Paused:           p.Paused,
		CreatedByActorID: p.CreatedByActorID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
}

func replaceSourceInstallsSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	sourceID int64,
	installs []InstallParams,
) error {
	existing, err := q.ListSkillSourceInstalls(ctx)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.SourceID != sourceID {
			continue
		}
		if _, err := q.DeleteSkillSourceInstall(ctx, sqlitedb.DeleteSkillSourceInstallParams{
			SourceID:  sourceID,
			ScopeKind: row.ScopeKind,
			ScopeID:   row.ScopeID,
		}); err != nil {
			return err
		}
	}
	for _, install := range installs {
		if err := q.UpsertSkillSourceInstall(ctx, sqlitedb.UpsertSkillSourceInstallParams{
			SourceID:         sourceID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: int64(install.CreatedByActorID),
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) DeleteSkillSource(
	ctx context.Context,
	id int64,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		rows, err := q.DeleteSkillSource(ctx, id)
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

func (s *SqliteStore) SetSkillSourceCheck(
	ctx context.Context,
	p SkillSourceCheckParams,
) error {
	return s.q.SetSkillSourceCheck(ctx, sqlitedb.SetSkillSourceCheckParams{
		CheckedAt:     p.CheckedAt,
		SyncState:     p.SyncState,
		SyncError:     p.SyncError,
		PendingCommit: p.PendingCommit,
		PendingJson:   p.PendingJSON,
		ProblemsJson:  p.ProblemsJSON,
		ID:            p.SourceID,
	})
}

func (s *SqliteStore) SetSkillSourceApplied(
	ctx context.Context,
	p SkillSourceAppliedParams,
) error {
	return s.q.SetSkillSourceApplied(ctx, sqlitedb.SetSkillSourceAppliedParams{
		SyncedCommit: p.Commit,
		SyncedAt:     p.AppliedAt,
		CheckedAt:    p.CheckedAt,
		SyncError:    p.SyncError,
		ProblemsJson: p.ProblemsJSON,
		ID:           p.SourceID,
	})
}

func (s *SqliteStore) ListSkillSourceInstalls(
	ctx context.Context,
) ([]SkillSourceInstall, error) {
	rows, err := s.q.ListSkillSourceInstalls(ctx)
	if err != nil {
		return nil, err
	}
	return fromSqliteSkillSourceInstallSlice(rows), nil
}

func (s *SqliteStore) SetSkillSourceInstall(
	ctx context.Context,
	p SetInstallParams,
) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if err := q.UpsertSkillSourceInstall(ctx, sqlitedb.UpsertSkillSourceInstallParams{
			SourceID:         p.EntryID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
		if err := stampSourceInstallSQLite(ctx, q, p); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func stampSourceInstallSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	p SetInstallParams,
) error {
	rows, err := q.ListSkillsBySource(ctx, sql.NullInt64{Int64: p.EntryID, Valid: true})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := q.UpsertSkillInstall(ctx, sqlitedb.UpsertSkillInstallParams{
			EntryID:          row.ID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) RemoveSkillSourceInstall(
	ctx context.Context,
	p RemoveInstallParams,
) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		rows, err := q.DeleteSkillSourceInstall(ctx, sqlitedb.DeleteSkillSourceInstallParams{
			SourceID:  p.EntryID,
			ScopeKind: p.ScopeKind,
			ScopeID:   p.ScopeID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return nil
		}
		skills, err := q.ListSkillsBySource(ctx, sql.NullInt64{Int64: p.EntryID, Valid: true})
		if err != nil {
			return err
		}
		for _, row := range skills {
			if _, err := q.DeleteSkillInstall(ctx, sqlitedb.DeleteSkillInstallParams{
				EntryID:   row.ID,
				ScopeKind: p.ScopeKind,
				ScopeID:   p.ScopeID,
			}); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *SqliteStore) ApplySkillSource(
	ctx context.Context,
	p ApplySkillSourceParams,
) (ApplySkillSourceResult, error) {
	out := ApplySkillSourceResult{}
	err := s.tx(ctx, func(q *sqlitedb.Queries) error {
		existing, err := q.ListSkillsBySource(
			ctx,
			sql.NullInt64{Int64: p.SourceID, Valid: true},
		)
		if err != nil {
			return err
		}
		byPath := make(map[string]sqlitedb.Skill, len(existing))
		for _, row := range existing {
			byPath[row.SourcePath] = row
		}
		wanted := make(map[string]bool, len(p.Skills))
		for _, skill := range p.Skills {
			wanted[skill.Path] = true
			row, found := byPath[skill.Path]
			if err := applySourceSkillSQLite(ctx, q, p, skill, row, found, &out); err != nil {
				return err
			}
		}
		for path, row := range byPath {
			if wanted[path] || row.SourceMissing == 1 {
				continue
			}
			if err := q.MarkSourcedSkillMissing(ctx, sqlitedb.MarkSourcedSkillMissingParams{
				SourceCommit: p.Commit,
				UpdatedAt:    p.Now,
				ID:           row.ID,
			}); err != nil {
				return err
			}
			out.Missing++
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return out, err
}

func applySourceSkillSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	p ApplySkillSourceParams,
	skill SkillSourceSkillParams,
	row sqlitedb.Skill,
	found bool,
	out *ApplySkillSourceResult,
) error {
	if found && row.Name != skill.Name {
		holder, err := q.GetSkillEntryByName(ctx, skill.Name)
		switch {
		case err == nil && holder.ID != row.ID:
			out.Conflicts = append(out.Conflicts, SkillSourceConflict{
				Path:   skill.Path,
				Name:   skill.Name,
				Reason: "another entry already has this name",
			})
			return nil
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return err
		}
	}
	if !found {
		_, err := q.GetSkillEntryByName(ctx, skill.Name)
		switch {
		case err == nil:
			out.Conflicts = append(out.Conflicts, SkillSourceConflict{
				Path:   skill.Path,
				Name:   skill.Name,
				Reason: "another entry already has this name",
			})
			return nil
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		created, err := q.CreateSourcedSkill(ctx, sqlitedb.CreateSourcedSkillParams{
			Name:                   skill.Name,
			Description:            skill.Description,
			Body:                   skill.Body,
			FilesJson:              skill.FilesJSON,
			DisableModelInvocation: skill.DisableModelInvocation,
			OwnerKind:              p.OwnerKind,
			OwnerID:                p.OwnerID,
			Enabled:                1,
			SourceID:               sql.NullInt64{Int64: p.SourceID, Valid: true},
			SourcePath:             skill.Path,
			SourceCommit:           p.Commit,
			CreatedByActorID:       p.CreatedByActorID,
			CreatedAt:              p.Now,
			UpdatedAt:              p.Now,
		})
		if err != nil {
			return err
		}
		out.Created++
		return stampSkillInstallsSQLite(ctx, q, created.ID, p.Installs)
	}
	updated, err := q.UpdateSourcedSkill(ctx, sqlitedb.UpdateSourcedSkillParams{
		Name:                   skill.Name,
		Description:            skill.Description,
		Body:                   skill.Body,
		FilesJson:              skill.FilesJSON,
		DisableModelInvocation: skill.DisableModelInvocation,
		SourcePath:             skill.Path,
		SourceCommit:           p.Commit,
		UpdatedAt:              p.Now,
		ID:                     row.ID,
	})
	if err != nil {
		return err
	}
	out.Updated++
	return stampSkillInstallsSQLite(ctx, q, updated.ID, p.Installs)
}

func stampSkillInstallsSQLite(
	ctx context.Context,
	q *sqlitedb.Queries,
	skillID int64,
	installs []InstallParams,
) error {
	for _, install := range installs {
		if err := q.UpsertSkillInstall(ctx, sqlitedb.UpsertSkillInstallParams{
			EntryID:          skillID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: int64(install.CreatedByActorID),
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) DetachSkill(
	ctx context.Context,
	id int64,
	updatedAt int64,
	revision string,
) error {
	return s.tx(ctx, func(q *sqlitedb.Queries) error {
		if err := q.DetachSourcedSkill(ctx, sqlitedb.DetachSourcedSkillParams{
			UpdatedAt: updatedAt,
			ID:        id,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, revision)
	})
}

func (s *PgStore) ListGitCredentials(
	ctx context.Context,
) ([]GitCredential, error) {
	rows, err := s.q.ListGitCredentials(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgGitCredentialSlice(rows), nil
}

func (s *PgStore) GetGitCredential(
	ctx context.Context,
	id int64,
) (GitCredential, error) {
	row, err := s.q.GetGitCredential(ctx, id)
	if err != nil {
		return GitCredential{}, err
	}
	return fromPgGitCredential(row), nil
}

func (s *PgStore) CountGitCredentialSources(
	ctx context.Context,
	id int64,
) (int64, error) {
	return s.q.CountGitCredentialSources(ctx, sql.NullInt64{Int64: id, Valid: true})
}

func (s *PgStore) CreateGitCredential(
	ctx context.Context,
	p GitCredentialParams,
) (GitCredential, error) {
	row, err := s.q.CreateGitCredential(ctx, pgdb.CreateGitCredentialParams{
		Name:             p.Name,
		Kind:             p.Kind,
		Host:             p.Host,
		Port:             p.Port,
		Username:         p.Username,
		SecretSealed:     p.SecretSealed,
		KnownHosts:       p.KnownHosts,
		PublicKey:        p.PublicKey,
		Fingerprint:      p.Fingerprint,
		CreatedByActorID: p.CreatedByActorID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
	if err != nil {
		return GitCredential{}, err
	}
	return fromPgGitCredential(row), nil
}

func (s *PgStore) UpdateGitCredential(
	ctx context.Context,
	p GitCredentialParams,
) (GitCredential, error) {
	row, err := s.q.UpdateGitCredential(ctx, pgdb.UpdateGitCredentialParams{
		Name:         p.Name,
		Kind:         p.Kind,
		Host:         p.Host,
		Port:         p.Port,
		Username:     p.Username,
		SecretSealed: p.SecretSealed,
		KnownHosts:   p.KnownHosts,
		PublicKey:    p.PublicKey,
		Fingerprint:  p.Fingerprint,
		UpdatedAt:    p.UpdatedAt,
		ID:           p.ID,
	})
	if err != nil {
		return GitCredential{}, err
	}
	return fromPgGitCredential(row), nil
}

func (s *PgStore) DeleteGitCredential(ctx context.Context, id int64) (bool, error) {
	rows, err := s.q.DeleteGitCredential(ctx, id)
	return rows > 0, err
}

func (s *PgStore) ListSkillSources(ctx context.Context) ([]SkillSource, error) {
	rows, err := s.q.ListSkillSources(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgSkillSourceSlice(rows), nil
}

func (s *PgStore) GetSkillSource(
	ctx context.Context,
	id int64,
) (SkillSource, error) {
	row, err := s.q.GetSkillSource(ctx, id)
	if err != nil {
		return SkillSource{}, err
	}
	return fromPgSkillSource(row), nil
}

func (s *PgStore) ListSkillsBySource(
	ctx context.Context,
	sourceID int64,
) ([]Skill, error) {
	rows, err := s.q.ListSkillsBySource(ctx, sql.NullInt64{Int64: sourceID, Valid: true})
	if err != nil {
		return nil, err
	}
	return fromPgSkillSlice(rows), nil
}

func (s *PgStore) SaveSkillSource(
	ctx context.Context,
	p SaveSkillSourceParams,
) (SkillSource, error) {
	var saved SkillSource
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		row, err := saveSkillSourcePg(ctx, q, p.Entry)
		if err != nil {
			return err
		}
		saved = fromPgSkillSource(row)
		if !p.KeepInstalls {
			if err := replaceSourceInstallsPg(ctx, q, saved.ID, p.Installs); err != nil {
				return err
			}
		}
		if err := q.ReownSourcedSkills(ctx, pgdb.ReownSourcedSkillsParams{
			OwnerKind: saved.OwnerKind,
			OwnerID:   saved.OwnerID,
			SourceID:  sql.NullInt64{Int64: saved.ID, Valid: true},
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return saved, err
}

func saveSkillSourcePg(
	ctx context.Context,
	q *pgdb.Queries,
	p SkillSourceParams,
) (pgdb.SkillSource, error) {
	credential := sql.NullInt64{}
	if p.CredentialID != 0 {
		credential = sql.NullInt64{Int64: p.CredentialID, Valid: true}
	}
	if p.ID != 0 {
		return q.UpdateSkillSource(ctx, pgdb.UpdateSkillSourceParams{
			Slug:          p.Slug,
			Name:          p.Name,
			Description:   p.Description,
			OwnerKind:     p.OwnerKind,
			OwnerID:       p.OwnerID,
			RepoUrl:       p.RepoURL,
			Ref:           p.Ref,
			PathGlob:      p.PathGlob,
			CredentialID:  credential,
			UpdateMode:    p.UpdateMode,
			PollIntervalS: p.PollIntervalS,
			Paused:        p.Paused,
			UpdatedAt:     p.UpdatedAt,
			ID:            p.ID,
		})
	}
	return q.CreateSkillSource(ctx, pgdb.CreateSkillSourceParams{
		Slug:             p.Slug,
		Name:             p.Name,
		Description:      p.Description,
		OwnerKind:        p.OwnerKind,
		OwnerID:          p.OwnerID,
		RepoUrl:          p.RepoURL,
		Ref:              p.Ref,
		PathGlob:         p.PathGlob,
		CredentialID:     credential,
		UpdateMode:       p.UpdateMode,
		PollIntervalS:    p.PollIntervalS,
		Paused:           p.Paused,
		CreatedByActorID: p.CreatedByActorID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	})
}

func replaceSourceInstallsPg(
	ctx context.Context,
	q *pgdb.Queries,
	sourceID int64,
	installs []InstallParams,
) error {
	existing, err := q.ListSkillSourceInstalls(ctx)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.SourceID != sourceID {
			continue
		}
		if _, err := q.DeleteSkillSourceInstall(ctx, pgdb.DeleteSkillSourceInstallParams{
			SourceID:  sourceID,
			ScopeKind: row.ScopeKind,
			ScopeID:   row.ScopeID,
		}); err != nil {
			return err
		}
	}
	for _, install := range installs {
		if err := q.UpsertSkillSourceInstall(ctx, pgdb.UpsertSkillSourceInstallParams{
			SourceID:         sourceID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: int64(install.CreatedByActorID),
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *PgStore) DeleteSkillSource(
	ctx context.Context,
	id int64,
	revision string,
) (bool, error) {
	deleted := false
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		rows, err := q.DeleteSkillSource(ctx, id)
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

func (s *PgStore) SetSkillSourceCheck(
	ctx context.Context,
	p SkillSourceCheckParams,
) error {
	return s.q.SetSkillSourceCheck(ctx, pgdb.SetSkillSourceCheckParams{
		CheckedAt:     p.CheckedAt,
		SyncState:     p.SyncState,
		SyncError:     p.SyncError,
		PendingCommit: p.PendingCommit,
		PendingJson:   p.PendingJSON,
		ProblemsJson:  p.ProblemsJSON,
		ID:            p.SourceID,
	})
}

func (s *PgStore) SetSkillSourceApplied(
	ctx context.Context,
	p SkillSourceAppliedParams,
) error {
	return s.q.SetSkillSourceApplied(ctx, pgdb.SetSkillSourceAppliedParams{
		SyncedCommit: p.Commit,
		SyncedAt:     p.AppliedAt,
		CheckedAt:    p.CheckedAt,
		SyncError:    p.SyncError,
		ProblemsJson: p.ProblemsJSON,
		ID:           p.SourceID,
	})
}

func (s *PgStore) ListSkillSourceInstalls(
	ctx context.Context,
) ([]SkillSourceInstall, error) {
	rows, err := s.q.ListSkillSourceInstalls(ctx)
	if err != nil {
		return nil, err
	}
	return fromPgSkillSourceInstallSlice(rows), nil
}

func (s *PgStore) SetSkillSourceInstall(
	ctx context.Context,
	p SetInstallParams,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.UpsertSkillSourceInstall(ctx, pgdb.UpsertSkillSourceInstallParams{
			SourceID:         p.EntryID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
		if err := stampSourceInstallPg(ctx, q, p); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func stampSourceInstallPg(
	ctx context.Context,
	q *pgdb.Queries,
	p SetInstallParams,
) error {
	rows, err := q.ListSkillsBySource(ctx, sql.NullInt64{Int64: p.EntryID, Valid: true})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := q.UpsertSkillInstall(ctx, pgdb.UpsertSkillInstallParams{
			EntryID:          row.ID,
			ScopeKind:        p.ScopeKind,
			ScopeID:          p.ScopeID,
			Enabled:          p.Enabled,
			CreatedByActorID: p.CreatedByActorID,
			CreatedAt:        p.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *PgStore) RemoveSkillSourceInstall(
	ctx context.Context,
	p RemoveInstallParams,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		rows, err := q.DeleteSkillSourceInstall(ctx, pgdb.DeleteSkillSourceInstallParams{
			SourceID:  p.EntryID,
			ScopeKind: p.ScopeKind,
			ScopeID:   p.ScopeID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return nil
		}
		skills, err := q.ListSkillsBySource(ctx, sql.NullInt64{Int64: p.EntryID, Valid: true})
		if err != nil {
			return err
		}
		for _, row := range skills {
			if _, err := q.DeleteSkillInstall(ctx, pgdb.DeleteSkillInstallParams{
				EntryID:   row.ID,
				ScopeKind: p.ScopeKind,
				ScopeID:   p.ScopeID,
			}); err != nil {
				return err
			}
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
}

func (s *PgStore) ApplySkillSource(
	ctx context.Context,
	p ApplySkillSourceParams,
) (ApplySkillSourceResult, error) {
	out := ApplySkillSourceResult{}
	err := s.tx(ctx, func(q *pgdb.Queries) error {
		existing, err := q.ListSkillsBySource(
			ctx,
			sql.NullInt64{Int64: p.SourceID, Valid: true},
		)
		if err != nil {
			return err
		}
		byPath := make(map[string]pgdb.Skill, len(existing))
		for _, row := range existing {
			byPath[row.SourcePath] = row
		}
		wanted := make(map[string]bool, len(p.Skills))
		for _, skill := range p.Skills {
			wanted[skill.Path] = true
			row, found := byPath[skill.Path]
			if err := applySourceSkillPg(ctx, q, p, skill, row, found, &out); err != nil {
				return err
			}
		}
		for path, row := range byPath {
			if wanted[path] || row.SourceMissing == 1 {
				continue
			}
			if err := q.MarkSourcedSkillMissing(ctx, pgdb.MarkSourcedSkillMissingParams{
				SourceCommit: p.Commit,
				UpdatedAt:    p.Now,
				ID:           row.ID,
			}); err != nil {
				return err
			}
			out.Missing++
		}
		return q.BumpAgentConfigRevision(ctx, p.Revision)
	})
	return out, err
}

func applySourceSkillPg(
	ctx context.Context,
	q *pgdb.Queries,
	p ApplySkillSourceParams,
	skill SkillSourceSkillParams,
	row pgdb.Skill,
	found bool,
	out *ApplySkillSourceResult,
) error {
	if found && row.Name != skill.Name {
		holder, err := q.GetSkillEntryByName(ctx, skill.Name)
		switch {
		case err == nil && holder.ID != row.ID:
			out.Conflicts = append(out.Conflicts, SkillSourceConflict{
				Path:   skill.Path,
				Name:   skill.Name,
				Reason: "another entry already has this name",
			})
			return nil
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return err
		}
	}
	if !found {
		_, err := q.GetSkillEntryByName(ctx, skill.Name)
		switch {
		case err == nil:
			out.Conflicts = append(out.Conflicts, SkillSourceConflict{
				Path:   skill.Path,
				Name:   skill.Name,
				Reason: "another entry already has this name",
			})
			return nil
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		created, err := q.CreateSourcedSkill(ctx, pgdb.CreateSourcedSkillParams{
			Name:                   skill.Name,
			Description:            skill.Description,
			Body:                   skill.Body,
			FilesJson:              skill.FilesJSON,
			DisableModelInvocation: skill.DisableModelInvocation,
			OwnerKind:              p.OwnerKind,
			OwnerID:                p.OwnerID,
			Enabled:                1,
			SourceID:               sql.NullInt64{Int64: p.SourceID, Valid: true},
			SourcePath:             skill.Path,
			SourceCommit:           p.Commit,
			CreatedByActorID:       p.CreatedByActorID,
			CreatedAt:              p.Now,
			UpdatedAt:              p.Now,
		})
		if err != nil {
			return err
		}
		out.Created++
		return stampSkillInstallsPg(ctx, q, created.ID, p.Installs)
	}
	updated, err := q.UpdateSourcedSkill(ctx, pgdb.UpdateSourcedSkillParams{
		Name:                   skill.Name,
		Description:            skill.Description,
		Body:                   skill.Body,
		FilesJson:              skill.FilesJSON,
		DisableModelInvocation: skill.DisableModelInvocation,
		SourcePath:             skill.Path,
		SourceCommit:           p.Commit,
		UpdatedAt:              p.Now,
		ID:                     row.ID,
	})
	if err != nil {
		return err
	}
	out.Updated++
	return stampSkillInstallsPg(ctx, q, updated.ID, p.Installs)
}

func stampSkillInstallsPg(
	ctx context.Context,
	q *pgdb.Queries,
	skillID int64,
	installs []InstallParams,
) error {
	for _, install := range installs {
		if err := q.UpsertSkillInstall(ctx, pgdb.UpsertSkillInstallParams{
			EntryID:          skillID,
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: int64(install.CreatedByActorID),
			CreatedAt:        install.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *PgStore) DetachSkill(
	ctx context.Context,
	id int64,
	updatedAt int64,
	revision string,
) error {
	return s.tx(ctx, func(q *pgdb.Queries) error {
		if err := q.DetachSourcedSkill(ctx, pgdb.DetachSourcedSkillParams{
			UpdatedAt: updatedAt,
			ID:        id,
		}); err != nil {
			return err
		}
		return q.BumpAgentConfigRevision(ctx, revision)
	})
}
