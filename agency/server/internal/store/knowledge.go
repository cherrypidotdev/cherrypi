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
	"fmt"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

var ErrKnowledgeConflict = errors.New("knowledge concept revision conflict")

type KnowledgeConceptWrite struct {
	Type               string
	Title              string
	Description        string
	Resource           string
	TagsJson           string
	Body               string
	FrontmatterJson    string
	Status             string
	StaleAfter         sql.NullInt64
	GeneratedByActorID sql.NullInt64
	GeneratedBy        string
	GeneratedAt        int64
	TrustTier          string
	Protected          int64
	DeletedAt          sql.NullInt64
}

type KnowledgeConceptFilter struct {
	Scopes         []string
	ScopeID        string
	PathPrefix     string
	Query          string
	Status         string
	TrustTier      string
	Tag            string
	IncludeDeleted bool
	Limit          int
	Offset         int
}

type knowledgeQueries struct {
	db       *sql.DB
	postgres bool
}

func (q knowledgeQueries) sql(query string) string {
	if !q.postgres {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 1
	for _, r := range query {
		if r == '?' {
			fmt.Fprintf(&b, "$%d", n)
			n++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

const knowledgeConceptColumns = `id, scope, scope_id, path, type, title, description,
	resource, tags_json, body, frontmatter_json, status, stale_after,
	generated_by_actor_id, generated_by, generated_at, trust_tier, revision,
	protected, deleted_at, created_by_actor_id, created_at, updated_at`

type knowledgeScanner interface {
	Scan(dest ...any) error
}

func scanKnowledgeConcept(row knowledgeScanner) (KnowledgeConcept, error) {
	var c KnowledgeConcept
	err := row.Scan(
		&c.ID, &c.Scope, &c.ScopeID, &c.Path, &c.Type, &c.Title, &c.Description,
		&c.Resource, &c.TagsJson, &c.Body, &c.FrontmatterJson, &c.Status,
		&c.StaleAfter, &c.GeneratedByActorID, &c.GeneratedBy, &c.GeneratedAt,
		&c.TrustTier, &c.Revision, &c.Protected, &c.DeletedAt,
		&c.CreatedByActorID, &c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

func (s *SqliteStore) CreateKnowledgeConcept(
	ctx context.Context,
	c KnowledgeConcept,
	w KnowledgeConceptWrite,
) (KnowledgeConcept, error) {
	return createKnowledgeConcept(ctx, s.knowledge(), c, w)
}

func (s *PgStore) CreateKnowledgeConcept(
	ctx context.Context,
	c KnowledgeConcept,
	w KnowledgeConceptWrite,
) (KnowledgeConcept, error) {
	return createKnowledgeConcept(ctx, s.knowledge(), c, w)
}

func createKnowledgeConcept(
	ctx context.Context,
	q knowledgeQueries,
	c KnowledgeConcept,
	w KnowledgeConceptWrite,
) (KnowledgeConcept, error) {
	row := q.db.QueryRowContext(ctx, q.sql(`INSERT INTO knowledge_concepts (
		id, scope, scope_id, path, type, title, description, resource, tags_json,
		body, frontmatter_json, status, stale_after, generated_by_actor_id,
		generated_by, generated_at, trust_tier, revision, protected, deleted_at,
		created_by_actor_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING `+knowledgeConceptColumns),
		c.ID, c.Scope, c.ScopeID, c.Path, w.Type, w.Title, w.Description,
		w.Resource, w.TagsJson, w.Body, w.FrontmatterJson, w.Status, w.StaleAfter,
		w.GeneratedByActorID, w.GeneratedBy, w.GeneratedAt, w.TrustTier,
		c.Revision, w.Protected, w.DeletedAt, c.CreatedByActorID, c.CreatedAt,
		c.UpdatedAt,
	)
	return scanKnowledgeConcept(row)
}

func (s *SqliteStore) GetKnowledgeConcept(
	ctx context.Context,
	scope, scopeID, path string,
) (KnowledgeConcept, error) {
	return getKnowledgeConcept(ctx, s.knowledge(), scope, scopeID, path)
}

func (s *PgStore) GetKnowledgeConcept(
	ctx context.Context,
	scope, scopeID, path string,
) (KnowledgeConcept, error) {
	return getKnowledgeConcept(ctx, s.knowledge(), scope, scopeID, path)
}

func getKnowledgeConcept(
	ctx context.Context,
	q knowledgeQueries,
	scope, scopeID, path string,
) (KnowledgeConcept, error) {
	row := q.db.QueryRowContext(ctx, q.sql(`SELECT `+knowledgeConceptColumns+`
		FROM knowledge_concepts
		WHERE scope = ? AND scope_id = ? AND path = ? AND deleted_at IS NULL`),
		scope, scopeID, path)
	return scanKnowledgeConcept(row)
}

func (s *SqliteStore) GetKnowledgeConceptByID(
	ctx context.Context,
	id string,
) (KnowledgeConcept, error) {
	return getKnowledgeConceptByID(ctx, s.knowledge(), id)
}

func (s *PgStore) GetKnowledgeConceptByID(
	ctx context.Context,
	id string,
) (KnowledgeConcept, error) {
	return getKnowledgeConceptByID(ctx, s.knowledge(), id)
}

func getKnowledgeConceptByID(
	ctx context.Context,
	q knowledgeQueries,
	id string,
) (KnowledgeConcept, error) {
	row := q.db.QueryRowContext(ctx, q.sql(`SELECT `+knowledgeConceptColumns+`
		FROM knowledge_concepts WHERE id = ?`), id)
	return scanKnowledgeConcept(row)
}

func (s *SqliteStore) ListKnowledgeConcepts(
	ctx context.Context,
	f KnowledgeConceptFilter,
) ([]KnowledgeConcept, error) {
	return listKnowledgeConcepts(ctx, s.knowledge(), f)
}

func (s *PgStore) ListKnowledgeConcepts(
	ctx context.Context,
	f KnowledgeConceptFilter,
) ([]KnowledgeConcept, error) {
	return listKnowledgeConcepts(ctx, s.knowledge(), f)
}

func listKnowledgeConcepts(
	ctx context.Context,
	q knowledgeQueries,
	f KnowledgeConceptFilter,
) ([]KnowledgeConcept, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 12)
	if !f.IncludeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if len(f.Scopes) > 0 {
		placeholders := make([]string, len(f.Scopes))
		for i, scope := range f.Scopes {
			placeholders[i] = "?"
			args = append(args, scope)
		}
		where = append(where, "scope IN ("+strings.Join(placeholders, ", ")+")")
	}
	if f.ScopeID != "" {
		where = append(where, "scope_id = ?")
		args = append(args, f.ScopeID)
	}
	if f.PathPrefix != "" {
		where = append(where, "path LIKE ?")
		args = append(args, f.PathPrefix+"%")
	}
	if f.Query != "" {
		where = append(where,
			"lower(title || ' ' || description || ' ' || tags_json || ' ' || body) LIKE lower(?)")
		args = append(args, "%"+f.Query+"%")
	}
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	if f.TrustTier != "" {
		where = append(where, "trust_tier = ?")
		args = append(args, f.TrustTier)
	}
	if f.Tag != "" {
		where = append(where, "tags_json LIKE ?")
		args = append(args, `%`+f.Tag+`%`)
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT `+knowledgeConceptColumns+`
		FROM knowledge_concepts
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY CASE scope WHEN 'org' THEN 0 WHEN 'project' THEN 1 ELSE 2 END,
		         updated_at DESC, id
		LIMIT ? OFFSET ?`), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]KnowledgeConcept, 0, limit)
	for rows.Next() {
		c, err := scanKnowledgeConcept(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SqliteStore) SaveKnowledgeConcept(
	ctx context.Context,
	id string,
	baseRevision int64,
	w KnowledgeConceptWrite,
	rev KnowledgeRevision,
) (KnowledgeConcept, error) {
	return saveKnowledgeConcept(ctx, s.knowledge(), id, baseRevision, w, rev)
}

func (s *PgStore) SaveKnowledgeConcept(
	ctx context.Context,
	id string,
	baseRevision int64,
	w KnowledgeConceptWrite,
	rev KnowledgeRevision,
) (KnowledgeConcept, error) {
	return saveKnowledgeConcept(ctx, s.knowledge(), id, baseRevision, w, rev)
}

func saveKnowledgeConcept(
	ctx context.Context,
	q knowledgeQueries,
	id string,
	baseRevision int64,
	w KnowledgeConceptWrite,
	rev KnowledgeRevision,
) (KnowledgeConcept, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return KnowledgeConcept{}, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, q.sql(`UPDATE knowledge_concepts SET
		type = ?, title = ?, description = ?, resource = ?, tags_json = ?,
		body = ?, frontmatter_json = ?, status = ?, stale_after = ?,
		generated_by_actor_id = ?, generated_by = ?, generated_at = ?,
		trust_tier = ?, protected = ?, deleted_at = ?,
		revision = revision + 1, updated_at = ?
		WHERE id = ? AND revision = ?
		RETURNING `+knowledgeConceptColumns),
		w.Type, w.Title, w.Description, w.Resource, w.TagsJson, w.Body,
		w.FrontmatterJson, w.Status, w.StaleAfter, w.GeneratedByActorID,
		w.GeneratedBy, w.GeneratedAt, w.TrustTier, w.Protected, w.DeletedAt,
		rev.CreatedAt, id, baseRevision,
	)
	concept, err := scanKnowledgeConcept(row)
	if errors.Is(err, sql.ErrNoRows) {
		return KnowledgeConcept{}, ErrKnowledgeConflict
	}
	if err != nil {
		return KnowledgeConcept{}, err
	}

	rev.ConceptID = concept.ID
	rev.Revision = concept.Revision
	if _, err := tx.ExecContext(ctx, q.sql(`INSERT INTO knowledge_revisions (
		concept_id, revision, title, body, frontmatter_json, status, actor_id,
		actor_ref, summary, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		rev.ConceptID, rev.Revision, rev.Title, rev.Body, rev.FrontmatterJson,
		rev.Status, rev.ActorID, rev.ActorRef, rev.Summary, rev.CreatedAt,
	); err != nil {
		return KnowledgeConcept{}, err
	}

	if err := tx.Commit(); err != nil {
		return KnowledgeConcept{}, err
	}
	return concept, nil
}

func (s *SqliteStore) SoftDeleteKnowledgeConcept(
	ctx context.Context,
	id string,
	now int64,
) error {
	return softDeleteKnowledgeConcept(ctx, s.knowledge(), id, now)
}

func (s *PgStore) SoftDeleteKnowledgeConcept(
	ctx context.Context,
	id string,
	now int64,
) error {
	return softDeleteKnowledgeConcept(ctx, s.knowledge(), id, now)
}

func softDeleteKnowledgeConcept(
	ctx context.Context,
	q knowledgeQueries,
	id string,
	now int64,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(`UPDATE knowledge_concepts
		SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`),
		now, now, id)
	return err
}

func (s *SqliteStore) SetKnowledgeConceptTrust(
	ctx context.Context,
	id, tier string,
	now int64,
) error {
	return setKnowledgeConceptTrust(ctx, s.knowledge(), id, tier, now)
}

func (s *PgStore) SetKnowledgeConceptTrust(
	ctx context.Context,
	id, tier string,
	now int64,
) error {
	return setKnowledgeConceptTrust(ctx, s.knowledge(), id, tier, now)
}

func setKnowledgeConceptTrust(
	ctx context.Context,
	q knowledgeQueries,
	id, tier string,
	now int64,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(`UPDATE knowledge_concepts
		SET trust_tier = ?, updated_at = ? WHERE id = ?`), tier, now, id)
	return err
}

func (s *SqliteStore) CreateKnowledgeRevision(
	ctx context.Context,
	rev KnowledgeRevision,
) (KnowledgeRevision, error) {
	return createKnowledgeRevision(ctx, s.knowledge(), rev)
}

func (s *PgStore) CreateKnowledgeRevision(
	ctx context.Context,
	rev KnowledgeRevision,
) (KnowledgeRevision, error) {
	return createKnowledgeRevision(ctx, s.knowledge(), rev)
}

func createKnowledgeRevision(
	ctx context.Context,
	q knowledgeQueries,
	rev KnowledgeRevision,
) (KnowledgeRevision, error) {
	err := q.db.QueryRowContext(ctx, q.sql(`INSERT INTO knowledge_revisions (
		concept_id, revision, title, body, frontmatter_json, status, actor_id,
		actor_ref, summary, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		rev.ConceptID, rev.Revision, rev.Title, rev.Body, rev.FrontmatterJson,
		rev.Status, rev.ActorID, rev.ActorRef, rev.Summary, rev.CreatedAt,
	).Scan(&rev.ID)
	return rev, err
}

func (s *SqliteStore) SetKnowledgeConceptFrontmatter(
	ctx context.Context,
	id, frontmatter string,
	now int64,
) error {
	return setKnowledgeConceptFrontmatter(ctx, s.knowledge(), id, frontmatter, now)
}

func (s *PgStore) SetKnowledgeConceptFrontmatter(
	ctx context.Context,
	id, frontmatter string,
	now int64,
) error {
	return setKnowledgeConceptFrontmatter(ctx, s.knowledge(), id, frontmatter, now)
}

func setKnowledgeConceptFrontmatter(
	ctx context.Context,
	q knowledgeQueries,
	id, frontmatter string,
	now int64,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(`UPDATE knowledge_concepts
		SET frontmatter_json = ?, updated_at = ? WHERE id = ?`),
		frontmatter, now, id)
	return err
}

func (s *SqliteStore) ListKnowledgeRevisions(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeRevision, error) {
	return listKnowledgeRevisions(ctx, s.knowledge(), conceptID)
}

func (s *PgStore) ListKnowledgeRevisions(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeRevision, error) {
	return listKnowledgeRevisions(ctx, s.knowledge(), conceptID)
}

func listKnowledgeRevisions(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
) ([]KnowledgeRevision, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT
		id, concept_id, revision, title, body, frontmatter_json, status,
		actor_id, actor_ref, summary, created_at
		FROM knowledge_revisions WHERE concept_id = ?
		ORDER BY revision DESC`), conceptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeRevision, 0, 8)
	for rows.Next() {
		var r KnowledgeRevision
		if err := rows.Scan(&r.ID, &r.ConceptID, &r.Revision, &r.Title, &r.Body,
			&r.FrontmatterJson, &r.Status, &r.ActorID, &r.ActorRef, &r.Summary,
			&r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SqliteStore) CreateKnowledgeVerification(
	ctx context.Context,
	v KnowledgeVerification,
) (KnowledgeVerification, error) {
	return createKnowledgeVerification(ctx, s.knowledge(), v)
}

func (s *PgStore) CreateKnowledgeVerification(
	ctx context.Context,
	v KnowledgeVerification,
) (KnowledgeVerification, error) {
	return createKnowledgeVerification(ctx, s.knowledge(), v)
}

func createKnowledgeVerification(
	ctx context.Context,
	q knowledgeQueries,
	v KnowledgeVerification,
) (KnowledgeVerification, error) {
	err := q.db.QueryRowContext(ctx, q.sql(`INSERT INTO knowledge_verifications (
		concept_id, actor_id, actor_ref, note, created_at
	) VALUES (?, ?, ?, ?, ?) RETURNING id`),
		v.ConceptID, v.ActorID, v.ActorRef, v.Note, v.CreatedAt,
	).Scan(&v.ID)
	return v, err
}

func (s *SqliteStore) RevokeKnowledgeVerification(
	ctx context.Context,
	conceptID string,
	verificationID int64,
	actorID ids.ActorID,
	now int64,
) error {
	return revokeKnowledgeVerification(ctx, s.knowledge(), conceptID, verificationID, actorID, now)
}

func (s *PgStore) RevokeKnowledgeVerification(
	ctx context.Context,
	conceptID string,
	verificationID int64,
	actorID ids.ActorID,
	now int64,
) error {
	return revokeKnowledgeVerification(ctx, s.knowledge(), conceptID, verificationID, actorID, now)
}

func revokeKnowledgeVerification(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
	verificationID int64,
	actorID ids.ActorID,
	now int64,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(`UPDATE knowledge_verifications
		SET revoked_at = ?, revoked_by_actor_id = ?
		WHERE id = ? AND concept_id = ? AND revoked_at IS NULL`),
		now, actorID, verificationID, conceptID)
	return err
}

func (s *SqliteStore) ListKnowledgeVerifications(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeVerification, error) {
	return listKnowledgeVerifications(ctx, s.knowledge(), conceptID)
}

func (s *PgStore) ListKnowledgeVerifications(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeVerification, error) {
	return listKnowledgeVerifications(ctx, s.knowledge(), conceptID)
}

func listKnowledgeVerifications(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
) ([]KnowledgeVerification, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT
		id, concept_id, actor_id, actor_ref, note, created_at, revoked_at,
		revoked_by_actor_id
		FROM knowledge_verifications WHERE concept_id = ?
		ORDER BY created_at DESC, id DESC`), conceptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeVerification, 0, 4)
	for rows.Next() {
		var v KnowledgeVerification
		if err := rows.Scan(&v.ID, &v.ConceptID, &v.ActorID, &v.ActorRef, &v.Note,
			&v.CreatedAt, &v.RevokedAt, &v.RevokedByActorID); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ReplaceKnowledgeSources(
	ctx context.Context,
	conceptID string,
	sources []KnowledgeSource,
) error {
	return replaceKnowledgeSources(ctx, s.knowledge(), conceptID, sources)
}

func (s *PgStore) ReplaceKnowledgeSources(
	ctx context.Context,
	conceptID string,
	sources []KnowledgeSource,
) error {
	return replaceKnowledgeSources(ctx, s.knowledge(), conceptID, sources)
}

func replaceKnowledgeSources(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
	sources []KnowledgeSource,
) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_sources WHERE concept_id = ?`), conceptID); err != nil {
		return err
	}
	for i, source := range sources {
		if _, err := tx.ExecContext(ctx, q.sql(`INSERT INTO knowledge_sources (
			concept_id, source_id, resource, title, author_ref, usage_count,
			last_modified, position
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
			conceptID, source.SourceID, source.Resource, source.Title,
			source.AuthorRef, source.UsageCount, source.LastModified, i,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SqliteStore) ListKnowledgeSources(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeSource, error) {
	return listKnowledgeSources(ctx, s.knowledge(), conceptID)
}

func (s *PgStore) ListKnowledgeSources(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeSource, error) {
	return listKnowledgeSources(ctx, s.knowledge(), conceptID)
}

func listKnowledgeSources(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
) ([]KnowledgeSource, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT
		id, concept_id, source_id, resource, title, author_ref, usage_count,
		last_modified, position
		FROM knowledge_sources WHERE concept_id = ? ORDER BY position, id`), conceptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeSource, 0, 4)
	for rows.Next() {
		var s KnowledgeSource
		if err := rows.Scan(&s.ID, &s.ConceptID, &s.SourceID, &s.Resource, &s.Title,
			&s.AuthorRef, &s.UsageCount, &s.LastModified, &s.Position); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ReplaceKnowledgeLinks(
	ctx context.Context,
	conceptID string,
	links []KnowledgeLink,
) error {
	return replaceKnowledgeLinks(ctx, s.knowledge(), conceptID, links)
}

func (s *PgStore) ReplaceKnowledgeLinks(
	ctx context.Context,
	conceptID string,
	links []KnowledgeLink,
) error {
	return replaceKnowledgeLinks(ctx, s.knowledge(), conceptID, links)
}

func replaceKnowledgeLinks(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
	links []KnowledgeLink,
) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_links WHERE concept_id = ?`), conceptID); err != nil {
		return err
	}
	for i, link := range links {
		if _, err := tx.ExecContext(ctx, q.sql(`INSERT INTO knowledge_links (
			concept_id, target_path, kind, position
		) VALUES (?, ?, ?, ?)`),
			conceptID, link.TargetPath, link.Kind, i,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SqliteStore) ListKnowledgeLinks(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeLink, error) {
	return listKnowledgeLinks(ctx, s.knowledge(), conceptID)
}

func (s *PgStore) ListKnowledgeLinks(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeLink, error) {
	return listKnowledgeLinks(ctx, s.knowledge(), conceptID)
}

func listKnowledgeLinks(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
) ([]KnowledgeLink, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT
		concept_id, target_path, kind, position
		FROM knowledge_links WHERE concept_id = ? ORDER BY position, target_path`), conceptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeLink, 0, 8)
	for rows.Next() {
		var l KnowledgeLink
		if err := rows.Scan(&l.ConceptID, &l.TargetPath, &l.Kind, &l.Position); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeBacklinks(
	ctx context.Context,
	targetPath string,
) ([]KnowledgeConcept, error) {
	return listKnowledgeBacklinks(ctx, s.knowledge(), targetPath)
}

func (s *PgStore) ListKnowledgeBacklinks(
	ctx context.Context,
	targetPath string,
) ([]KnowledgeConcept, error) {
	return listKnowledgeBacklinks(ctx, s.knowledge(), targetPath)
}

func listKnowledgeBacklinks(
	ctx context.Context,
	q knowledgeQueries,
	targetPath string,
) ([]KnowledgeConcept, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT `+knowledgeConceptColumns+`
		FROM knowledge_concepts c
		WHERE c.deleted_at IS NULL AND EXISTS (
			SELECT 1 FROM knowledge_links l
			WHERE l.concept_id = c.id AND l.target_path = ?
		)
		ORDER BY c.updated_at DESC, c.id LIMIT 100`), targetPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeConcept, 0, 8)
	for rows.Next() {
		c, err := scanKnowledgeConcept(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SqliteStore) CreateKnowledgeMemoryRecovery(
	ctx context.Context,
	r KnowledgeMemoryRecovery,
) error {
	return createKnowledgeMemoryRecovery(ctx, s.knowledge(), r)
}

func (s *PgStore) CreateKnowledgeMemoryRecovery(
	ctx context.Context,
	r KnowledgeMemoryRecovery,
) error {
	return createKnowledgeMemoryRecovery(ctx, s.knowledge(), r)
}

func createKnowledgeMemoryRecovery(
	ctx context.Context,
	q knowledgeQueries,
	r KnowledgeMemoryRecovery,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(`INSERT INTO knowledge_memory_recovery (
		id, agent_actor_id, payload_json, created_at, restored_at
	) VALUES (?, ?, ?, ?, ?)`),
		r.ID, r.AgentActorID, r.PayloadJson, r.CreatedAt, r.RestoredAt)
	return err
}

func (s *SqliteStore) GetKnowledgeMemoryRecovery(
	ctx context.Context,
	id string,
) (KnowledgeMemoryRecovery, error) {
	return getKnowledgeMemoryRecovery(ctx, s.knowledge(), id)
}

func (s *PgStore) GetKnowledgeMemoryRecovery(
	ctx context.Context,
	id string,
) (KnowledgeMemoryRecovery, error) {
	return getKnowledgeMemoryRecovery(ctx, s.knowledge(), id)
}

func getKnowledgeMemoryRecovery(
	ctx context.Context,
	q knowledgeQueries,
	id string,
) (KnowledgeMemoryRecovery, error) {
	var r KnowledgeMemoryRecovery
	err := q.db.QueryRowContext(ctx, q.sql(`SELECT
		id, agent_actor_id, payload_json, created_at, restored_at
		FROM knowledge_memory_recovery WHERE id = ?`), id,
	).Scan(&r.ID, &r.AgentActorID, &r.PayloadJson, &r.CreatedAt, &r.RestoredAt)
	return r, err
}

func (s *SqliteStore) MarkKnowledgeMemoryRecoveryRestored(
	ctx context.Context,
	id string,
	now int64,
) error {
	return markKnowledgeMemoryRecoveryRestored(ctx, s.knowledge(), id, now)
}

func (s *PgStore) MarkKnowledgeMemoryRecoveryRestored(
	ctx context.Context,
	id string,
	now int64,
) error {
	return markKnowledgeMemoryRecoveryRestored(ctx, s.knowledge(), id, now)
}

func markKnowledgeMemoryRecoveryRestored(
	ctx context.Context,
	q knowledgeQueries,
	id string,
	now int64,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(`UPDATE knowledge_memory_recovery
		SET restored_at = COALESCE(restored_at, ?) WHERE id = ?`), now, id)
	return err
}

func (s *SqliteStore) knowledge() knowledgeQueries {
	return knowledgeQueries{db: s.db, postgres: false}
}

func (s *PgStore) knowledge() knowledgeQueries {
	return knowledgeQueries{db: s.db, postgres: true}
}

type DecisionFilter struct {
	ProjectID *string
	OrgWide   bool

	ReadableProjectIDs []string
	TaskID             string
	Kind               string

	BeforeID int64
	Limit    int
}

func (s *SqliteStore) CreateKnowledgeDecision(
	ctx context.Context,
	d KnowledgeDecision,
) (KnowledgeDecision, bool, error) {
	return createKnowledgeDecision(ctx, s.knowledge(), d)
}

func (s *PgStore) CreateKnowledgeDecision(
	ctx context.Context,
	d KnowledgeDecision,
) (KnowledgeDecision, bool, error) {
	return createKnowledgeDecision(ctx, s.knowledge(), d)
}

func createKnowledgeDecision(
	ctx context.Context,
	q knowledgeQueries,
	d KnowledgeDecision,
) (KnowledgeDecision, bool, error) {

	row := q.db.QueryRowContext(ctx, q.sql(`INSERT INTO knowledge_decisions (
		project_id, task_rowid, task_id, kind, actor_id, actor_ref, summary,
		payload_json, event_id, created_at
	) VALUES (?, (SELECT t.rowid FROM tasks t WHERE t.id = ?), ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT (kind, task_id, event_id) DO NOTHING
	RETURNING id`),
		d.ProjectID, d.TaskID, d.TaskID, d.Kind, d.ActorID, d.ActorRef,
		d.Summary, d.PayloadJson, d.EventID, d.CreatedAt,
	)
	err := row.Scan(&d.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return KnowledgeDecision{}, false, nil
	}
	if err != nil {
		return KnowledgeDecision{}, false, err
	}
	return d, true, nil
}

func (s *SqliteStore) ListKnowledgeDecisions(
	ctx context.Context,
	f DecisionFilter,
) ([]KnowledgeDecision, error) {
	return listKnowledgeDecisions(ctx, s.knowledge(), f)
}

func (s *PgStore) ListKnowledgeDecisions(
	ctx context.Context,
	f DecisionFilter,
) ([]KnowledgeDecision, error) {
	return listKnowledgeDecisions(ctx, s.knowledge(), f)
}

func listKnowledgeDecisions(
	ctx context.Context,
	q knowledgeQueries,
	f DecisionFilter,
) ([]KnowledgeDecision, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 6)
	if f.TaskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, f.TaskID)
	}
	if f.ProjectID != nil {
		where = append(where, "project_id = ?")
		args = append(args, *f.ProjectID)
	}
	if f.OrgWide {

		readable := make([]string, len(f.ReadableProjectIDs))
		for i, id := range f.ReadableProjectIDs {
			readable[i] = "?"
			args = append(args, id)
		}
		if len(readable) == 0 {
			where = append(where, "project_id IS NULL")
		} else {
			where = append(where, "(project_id IS NULL OR project_id IN ("+
				strings.Join(readable, ", ")+"))")
		}
	} else if f.ProjectID == nil && f.TaskID == "" {
		where = append(where, "project_id IS NULL")
	}
	if f.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, f.Kind)
	}
	if f.BeforeID > 0 {
		where = append(where, "id < ?")
		args = append(args, f.BeforeID)
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	args = append(args, limit)

	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT
		id, project_id, task_rowid, task_id, kind, actor_id, actor_ref, summary,
		payload_json, event_id, created_at
		FROM knowledge_decisions
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY created_at DESC, id DESC
		LIMIT ?`), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]KnowledgeDecision, 0, limit)
	for rows.Next() {
		var d KnowledgeDecision
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.TaskRowid, &d.TaskID,
			&d.Kind, &d.ActorID, &d.ActorRef, &d.Summary, &d.PayloadJson,
			&d.EventID, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *SqliteStore) GetKnowledgeDecision(
	ctx context.Context,
	id int64,
) (KnowledgeDecision, error) {
	return getKnowledgeDecision(ctx, s.knowledge(), id)
}

func (s *PgStore) GetKnowledgeDecision(
	ctx context.Context,
	id int64,
) (KnowledgeDecision, error) {
	return getKnowledgeDecision(ctx, s.knowledge(), id)
}

func getKnowledgeDecision(
	ctx context.Context,
	q knowledgeQueries,
	id int64,
) (KnowledgeDecision, error) {
	var d KnowledgeDecision
	err := q.db.QueryRowContext(ctx, q.sql(`SELECT
		id, project_id, task_rowid, task_id, kind, actor_id, actor_ref, summary,
		payload_json, event_id, created_at
		FROM knowledge_decisions WHERE id = ?`), id,
	).Scan(&d.ID, &d.ProjectID, &d.TaskRowid, &d.TaskID, &d.Kind, &d.ActorID,
		&d.ActorRef, &d.Summary, &d.PayloadJson, &d.EventID, &d.CreatedAt)
	return d, err
}

func (s *SqliteStore) LinkKnowledgeDecisionConcepts(
	ctx context.Context,
	decisionID int64,
	conceptIDs []string,
) error {
	return linkKnowledgeDecisionConcepts(ctx, s.knowledge(), decisionID, conceptIDs)
}

func (s *PgStore) LinkKnowledgeDecisionConcepts(
	ctx context.Context,
	decisionID int64,
	conceptIDs []string,
) error {
	return linkKnowledgeDecisionConcepts(ctx, s.knowledge(), decisionID, conceptIDs)
}

func linkKnowledgeDecisionConcepts(
	ctx context.Context,
	q knowledgeQueries,
	decisionID int64,
	conceptIDs []string,
) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_decision_concepts WHERE decision_id = ?`), decisionID); err != nil {
		return err
	}
	for i, conceptID := range conceptIDs {
		if _, err := tx.ExecContext(ctx, q.sql(`INSERT INTO knowledge_decision_concepts (
			decision_id, concept_id, position
		) VALUES (?, ?, ?)`), decisionID, conceptID, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SqliteStore) ReplaceKnowledgeConceptDecisions(
	ctx context.Context,
	conceptID string,
	decisionIDs []int64,
) error {
	return replaceKnowledgeConceptDecisions(ctx, s.knowledge(), conceptID, decisionIDs)
}

func (s *PgStore) ReplaceKnowledgeConceptDecisions(
	ctx context.Context,
	conceptID string,
	decisionIDs []int64,
) error {
	return replaceKnowledgeConceptDecisions(ctx, s.knowledge(), conceptID, decisionIDs)
}

func replaceKnowledgeConceptDecisions(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
	decisionIDs []int64,
) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_decision_concepts WHERE concept_id = ?`), conceptID); err != nil {
		return err
	}
	for i, decisionID := range decisionIDs {
		if _, err := tx.ExecContext(ctx, q.sql(`INSERT INTO knowledge_decision_concepts (
			decision_id, concept_id, position
		) VALUES (?, ?, ?)`), decisionID, conceptID, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SqliteStore) ListKnowledgeConceptDecisions(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeDecision, error) {
	return listKnowledgeConceptDecisions(ctx, s.knowledge(), conceptID)
}

func (s *PgStore) ListKnowledgeConceptDecisions(
	ctx context.Context,
	conceptID string,
) ([]KnowledgeDecision, error) {
	return listKnowledgeConceptDecisions(ctx, s.knowledge(), conceptID)
}

func listKnowledgeConceptDecisions(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
) ([]KnowledgeDecision, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT
		d.id, d.project_id, d.task_rowid, d.task_id, d.kind, d.actor_id,
		d.actor_ref, d.summary, d.payload_json, d.event_id, d.created_at
		FROM knowledge_decision_concepts l
		JOIN knowledge_decisions d ON d.id = l.decision_id
		WHERE l.concept_id = ?
		ORDER BY d.created_at DESC, d.id DESC LIMIT 100`), conceptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeDecision, 0, 4)
	for rows.Next() {
		var d KnowledgeDecision
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.TaskRowid, &d.TaskID,
			&d.Kind, &d.ActorID, &d.ActorRef, &d.Summary, &d.PayloadJson,
			&d.EventID, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *SqliteStore) GetKnowledgeDecisionIDs(
	ctx context.Context,
	conceptID string,
) ([]int64, error) {
	return getKnowledgeDecisionIDs(ctx, s.knowledge(), conceptID)
}

func (s *PgStore) GetKnowledgeDecisionIDs(
	ctx context.Context,
	conceptID string,
) ([]int64, error) {
	return getKnowledgeDecisionIDs(ctx, s.knowledge(), conceptID)
}

func getKnowledgeDecisionIDs(
	ctx context.Context,
	q knowledgeQueries,
	conceptID string,
) ([]int64, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT decision_id
		FROM knowledge_decision_concepts WHERE concept_id = ?
		ORDER BY position, decision_id`), conceptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]int64, 0, 4)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeDecisionConceptPaths(
	ctx context.Context,
	decisionID int64,
) ([]string, error) {
	return listKnowledgeDecisionConceptPaths(ctx, s.knowledge(), decisionID)
}

func (s *PgStore) ListKnowledgeDecisionConceptPaths(
	ctx context.Context,
	decisionID int64,
) ([]string, error) {
	return listKnowledgeDecisionConceptPaths(ctx, s.knowledge(), decisionID)
}

func listKnowledgeDecisionConceptPaths(
	ctx context.Context,
	q knowledgeQueries,
	decisionID int64,
) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT c.path
		FROM knowledge_decision_concepts l
		JOIN knowledge_concepts c ON c.id = l.concept_id
		WHERE l.decision_id = ?
		ORDER BY l.position, c.path`), decisionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]string, 0, 4)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeDecisionConcepts(
	ctx context.Context,
	decisionID int64,
) ([]string, error) {
	return listKnowledgeDecisionConcepts(ctx, s.knowledge(), decisionID)
}

func (s *PgStore) ListKnowledgeDecisionConcepts(
	ctx context.Context,
	decisionID int64,
) ([]string, error) {
	return listKnowledgeDecisionConcepts(ctx, s.knowledge(), decisionID)
}

func listKnowledgeDecisionConcepts(
	ctx context.Context,
	q knowledgeQueries,
	decisionID int64,
) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(`SELECT concept_id
		FROM knowledge_decision_concepts WHERE decision_id = ?
		ORDER BY position, concept_id`), decisionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]string, 0, 4)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
