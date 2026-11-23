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

type KnowledgeBitRow struct {
	ID                string
	ProjectID         sql.NullString
	Title             string
	Content           string
	Status            string
	OriginTaskID      string
	OriginTaskRowID   sql.NullInt64
	Revision          int64
	SupersededByID    sql.NullString
	CreatedByActorID  sql.NullInt64
	CreatedBy         string
	CreatedAt         int64
	UpdatedAt         int64
	ReviewedByActorID sql.NullInt64
	ReviewedBy        string
	ReviewedAt        sql.NullInt64
	ReviewNote        string
	DeletedAt         sql.NullInt64
}

type KnowledgeBitSourceRow struct {
	ID                int64
	BitID             string
	Kind              string
	TaskID            string
	TaskRowID         sql.NullInt64
	MessageID         string
	RequestID         string
	QuestionID        string
	AnsweredByActorID sql.NullInt64
	AnsweredBy        string
	PromptID          sql.NullInt64
	PromptScope       string
	PromptTargetID    string
	Label             string
	At                sql.NullInt64
	Href              string
	Position          int64
}

type KnowledgeBitLinkRow struct {
	LinkedID  string
	Kind      string
	Title     string
	ProjectID sql.NullString
	Status    string
}

type KnowledgeBitLinkWrite struct {
	BitID    string
	TargetID string
	Kind     string
	Position int64
}

type KnowledgeBitRevisionRow struct {
	ID        int64
	BitID     string
	Revision  int64
	Title     string
	Content   string
	TagsJSON  string
	Status    string
	ActorID   sql.NullInt64
	ActorRef  string
	Summary   string
	CreatedAt int64
}

type KnowledgeBitFilter struct {
	ProjectID  *string
	OrgOnly    bool
	ProjectIDs []string
	Status     string
	Tag        string
	Query      string
	TaskID     string
	AuthorID   sql.NullInt64
	Limit      int
	Offset     int
}

type KnowledgeTagCount struct {
	Tag   string
	Count int64
}

const knowledgeBitColumns = `id, project_id, title, content, status,
	origin_task_id, origin_task_rowid, revision, superseded_by_id,
	created_by_actor_id, created_by, created_at, updated_at,
	reviewed_by_actor_id, reviewed_by, reviewed_at, review_note, deleted_at`

type knowledgeBitScanner interface {
	Scan(dest ...any) error
}

func scanKnowledgeBit(row knowledgeBitScanner) (KnowledgeBitRow, error) {
	var b KnowledgeBitRow
	err := row.Scan(
		&b.ID, &b.ProjectID, &b.Title, &b.Content, &b.Status,
		&b.OriginTaskID, &b.OriginTaskRowID, &b.Revision, &b.SupersededByID,
		&b.CreatedByActorID, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
		&b.ReviewedByActorID, &b.ReviewedBy, &b.ReviewedAt, &b.ReviewNote,
		&b.DeletedAt,
	)
	return b, err
}

func (s *SqliteStore) CreateKnowledgeBit(
	ctx context.Context,
	bit KnowledgeBitRow,
	tags []string,
	sources []KnowledgeBitSourceRow,
	links []KnowledgeBitLinkWrite,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRow, error) {
	return createKnowledgeBit(ctx, s.knowledge(), bit, tags, sources, links, rev)
}

func (s *PgStore) CreateKnowledgeBit(
	ctx context.Context,
	bit KnowledgeBitRow,
	tags []string,
	sources []KnowledgeBitSourceRow,
	links []KnowledgeBitLinkWrite,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRow, error) {
	return createKnowledgeBit(ctx, s.knowledge(), bit, tags, sources, links, rev)
}

func createKnowledgeBit(
	ctx context.Context,
	q knowledgeQueries,
	bit KnowledgeBitRow,
	tags []string,
	sources []KnowledgeBitSourceRow,
	links []KnowledgeBitLinkWrite,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRow, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return KnowledgeBitRow{}, err
	}
	defer func() { _ = tx.Rollback() }()

	saved, err := scanKnowledgeBit(tx.QueryRowContext(ctx, q.sql(
		`INSERT INTO knowledge_bits (`+knowledgeBitColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING `+knowledgeBitColumns),
		bit.ID, bit.ProjectID, bit.Title, bit.Content, bit.Status,
		bit.OriginTaskID, bit.OriginTaskRowID, bit.Revision,
		bit.SupersededByID, bit.CreatedByActorID, bit.CreatedBy,
		bit.CreatedAt, bit.UpdatedAt, bit.ReviewedByActorID, bit.ReviewedBy,
		bit.ReviewedAt, bit.ReviewNote, bit.DeletedAt,
	))
	if err != nil {
		return KnowledgeBitRow{}, err
	}
	if err := insertKnowledgeBitTags(ctx, tx, q, bit.ID, tags); err != nil {
		return KnowledgeBitRow{}, err
	}
	for _, source := range sources {
		source.BitID = bit.ID
		if err := insertKnowledgeBitSource(ctx, tx, q, source); err != nil {
			return KnowledgeBitRow{}, err
		}
	}
	for _, link := range links {
		link.BitID = bit.ID
		if err := insertKnowledgeBitLink(ctx, tx, q, link); err != nil {
			return KnowledgeBitRow{}, err
		}
	}
	rev.BitID = bit.ID
	rev.Revision = bit.Revision
	if _, err := insertKnowledgeBitRevision(ctx, tx, q, rev); err != nil {
		return KnowledgeBitRow{}, err
	}
	if err := tx.Commit(); err != nil {
		return KnowledgeBitRow{}, err
	}
	return saved, nil
}

func (s *SqliteStore) GetKnowledgeBit(
	ctx context.Context,
	id string,
) (KnowledgeBitRow, error) {
	return getKnowledgeBit(ctx, s.knowledge(), id)
}

func (s *PgStore) GetKnowledgeBit(
	ctx context.Context,
	id string,
) (KnowledgeBitRow, error) {
	return getKnowledgeBit(ctx, s.knowledge(), id)
}

func getKnowledgeBit(
	ctx context.Context,
	q knowledgeQueries,
	id string,
) (KnowledgeBitRow, error) {
	return scanKnowledgeBit(q.db.QueryRowContext(ctx, q.sql(
		`SELECT `+knowledgeBitColumns+` FROM knowledge_bits
		WHERE id = ? AND deleted_at IS NULL`), id))
}

func (s *SqliteStore) ListKnowledgeBits(
	ctx context.Context,
	filter KnowledgeBitFilter,
) ([]KnowledgeBitRow, error) {
	return listKnowledgeBits(ctx, s.knowledge(), filter)
}

func (s *PgStore) ListKnowledgeBits(
	ctx context.Context,
	filter KnowledgeBitFilter,
) ([]KnowledgeBitRow, error) {
	return listKnowledgeBits(ctx, s.knowledge(), filter)
}

func listKnowledgeBits(
	ctx context.Context,
	q knowledgeQueries,
	filter KnowledgeBitFilter,
) ([]KnowledgeBitRow, error) {
	where := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 8)
	switch {
	case filter.OrgOnly:
		where = append(where, "project_id IS NULL")
	case filter.ProjectID != nil:
		where = append(where, "project_id = ?")
		args = append(args, *filter.ProjectID)
	default:
		visible := make([]string, 0, len(filter.ProjectIDs))
		for _, id := range filter.ProjectIDs {
			if id != "" {
				visible = append(visible, "?")
				args = append(args, id)
			}
		}
		if len(visible) == 0 {
			where = append(where, "project_id IS NULL")
		} else {
			where = append(where, "(project_id IS NULL OR project_id IN ("+
				strings.Join(visible, ", ")+"))")
		}
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	} else {
		where = append(where, "status = 'approved'")
	}
	if filter.Tag != "" {
		where = append(where, `EXISTS (
			SELECT 1 FROM knowledge_bit_tags t
			WHERE t.bit_id = knowledge_bits.id AND t.tag = ?
		)`)
		args = append(args, filter.Tag)
	}
	if filter.Query != "" {
		where = append(where, "(LOWER(title) LIKE ? OR LOWER(content) LIKE ?)")
		like := "%" + strings.ToLower(filter.Query) + "%"
		args = append(args, like, like)
	}
	if filter.TaskID != "" {
		where = append(where, `(origin_task_id = ? OR EXISTS (
			SELECT 1 FROM knowledge_bit_sources s
			WHERE s.bit_id = knowledge_bits.id AND s.task_id = ?
		))`)
		args = append(args, filter.TaskID, filter.TaskID)
	}
	if filter.AuthorID.Valid {
		where = append(where, "created_by_actor_id = ?")
		args = append(args, filter.AuthorID.Int64)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, max(filter.Offset, 0))
	rows, err := q.db.QueryContext(ctx, q.sql(
		`SELECT `+knowledgeBitColumns+` FROM knowledge_bits WHERE `+
			strings.Join(where, " AND ")+
			` ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?`), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]KnowledgeBitRow, 0, limit)
	for rows.Next() {
		b, err := scanKnowledgeBit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *SqliteStore) UpdateKnowledgeBit(
	ctx context.Context,
	id string,
	baseRevision int64,
	bit KnowledgeBitRow,
	tags []string,
	sources []KnowledgeBitSourceRow,
	links []KnowledgeBitLinkWrite,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRow, error) {
	return updateKnowledgeBit(
		ctx, s.knowledge(), id, baseRevision, bit, tags, sources, links, rev,
	)
}

func (s *PgStore) UpdateKnowledgeBit(
	ctx context.Context,
	id string,
	baseRevision int64,
	bit KnowledgeBitRow,
	tags []string,
	sources []KnowledgeBitSourceRow,
	links []KnowledgeBitLinkWrite,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRow, error) {
	return updateKnowledgeBit(
		ctx, s.knowledge(), id, baseRevision, bit, tags, sources, links, rev,
	)
}

func updateKnowledgeBit(
	ctx context.Context,
	q knowledgeQueries,
	id string,
	baseRevision int64,
	bit KnowledgeBitRow,
	tags []string,
	sources []KnowledgeBitSourceRow,
	links []KnowledgeBitLinkWrite,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRow, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return KnowledgeBitRow{}, err
	}
	defer func() { _ = tx.Rollback() }()

	saved, err := scanKnowledgeBit(tx.QueryRowContext(ctx, q.sql(
		`UPDATE knowledge_bits SET
		title = ?, content = ?, status = ?, reviewed_by_actor_id = ?,
		reviewed_by = ?, reviewed_at = ?, review_note = ?,
		revision = revision + 1, updated_at = ?
		WHERE id = ? AND revision = ? AND deleted_at IS NULL
		RETURNING `+knowledgeBitColumns),
		bit.Title, bit.Content, bit.Status, bit.ReviewedByActorID,
		bit.ReviewedBy, bit.ReviewedAt, bit.ReviewNote, bit.UpdatedAt,
		id, baseRevision,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return KnowledgeBitRow{}, ErrKnowledgeConflict
	}
	if err != nil {
		return KnowledgeBitRow{}, err
	}
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_bit_tags WHERE bit_id = ?`), id); err != nil {
		return KnowledgeBitRow{}, err
	}
	if err := insertKnowledgeBitTags(ctx, tx, q, id, tags); err != nil {
		return KnowledgeBitRow{}, err
	}
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_bit_sources WHERE bit_id = ?`), id); err != nil {
		return KnowledgeBitRow{}, err
	}
	for _, source := range sources {
		source.BitID = id
		if err := insertKnowledgeBitSource(ctx, tx, q, source); err != nil {
			return KnowledgeBitRow{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, q.sql(
		`DELETE FROM knowledge_bit_links WHERE bit_id = ?`), id); err != nil {
		return KnowledgeBitRow{}, err
	}
	for _, link := range links {
		link.BitID = id
		if err := insertKnowledgeBitLink(ctx, tx, q, link); err != nil {
			return KnowledgeBitRow{}, err
		}
	}
	rev.BitID = id
	rev.Revision = saved.Revision
	if _, err := insertKnowledgeBitRevision(ctx, tx, q, rev); err != nil {
		return KnowledgeBitRow{}, err
	}
	if err := tx.Commit(); err != nil {
		return KnowledgeBitRow{}, err
	}
	return saved, nil
}

func (s *SqliteStore) ReviewKnowledgeBit(
	ctx context.Context,
	id, status string,
	reviewerID sql.NullInt64,
	reviewer, note string,
	now int64,
	supersedeIDs []string,
) (KnowledgeBitRow, error) {
	return reviewKnowledgeBit(
		ctx, s.knowledge(), id, status, reviewerID, reviewer, note, now,
		supersedeIDs,
	)
}

func (s *PgStore) ReviewKnowledgeBit(
	ctx context.Context,
	id, status string,
	reviewerID sql.NullInt64,
	reviewer, note string,
	now int64,
	supersedeIDs []string,
) (KnowledgeBitRow, error) {
	return reviewKnowledgeBit(
		ctx, s.knowledge(), id, status, reviewerID, reviewer, note, now,
		supersedeIDs,
	)
}

func reviewKnowledgeBit(
	ctx context.Context,
	q knowledgeQueries,
	id, status string,
	reviewerID sql.NullInt64,
	reviewer, note string,
	now int64,
	supersedeIDs []string,
) (KnowledgeBitRow, error) {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return KnowledgeBitRow{}, err
	}
	defer func() { _ = tx.Rollback() }()

	saved, err := scanKnowledgeBit(tx.QueryRowContext(ctx, q.sql(
		`UPDATE knowledge_bits SET
		status = ?, reviewed_by_actor_id = ?, reviewed_by = ?,
		reviewed_at = ?, review_note = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
		RETURNING `+knowledgeBitColumns),
		status, reviewerID, reviewer, now, note, now, id,
	))
	if err != nil {
		return KnowledgeBitRow{}, err
	}
	if status == "approved" {
		for _, target := range supersedeIDs {
			if _, err := tx.ExecContext(ctx, q.sql(
				`UPDATE knowledge_bits SET status = 'superseded',
				superseded_by_id = ?, updated_at = ?
				WHERE id = ? AND deleted_at IS NULL AND id <> ?`),
				id, now, target, id); err != nil {
				return KnowledgeBitRow{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return KnowledgeBitRow{}, err
	}
	return saved, nil
}

func (s *SqliteStore) RetireKnowledgeBits(
	ctx context.Context,
	byID string,
	ids []string,
	now int64,
) error {
	return retireKnowledgeBits(ctx, s.knowledge(), byID, ids, now)
}

func (s *PgStore) RetireKnowledgeBits(
	ctx context.Context,
	byID string,
	ids []string,
	now int64,
) error {
	return retireKnowledgeBits(ctx, s.knowledge(), byID, ids, now)
}

func retireKnowledgeBits(
	ctx context.Context,
	q knowledgeQueries,
	byID string,
	ids []string,
	now int64,
) error {
	for _, id := range ids {
		if _, err := q.db.ExecContext(ctx, q.sql(
			`UPDATE knowledge_bits SET status = 'superseded',
			superseded_by_id = ?, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL`), byID, now, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) SoftDeleteKnowledgeBit(
	ctx context.Context,
	id string,
	now int64,
) error {
	return softDeleteKnowledgeBit(ctx, s.knowledge(), id, now)
}

func (s *PgStore) SoftDeleteKnowledgeBit(
	ctx context.Context,
	id string,
	now int64,
) error {
	return softDeleteKnowledgeBit(ctx, s.knowledge(), id, now)
}

func softDeleteKnowledgeBit(
	ctx context.Context,
	q knowledgeQueries,
	id string,
	now int64,
) error {
	_, err := q.db.ExecContext(ctx, q.sql(
		`UPDATE knowledge_bits SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`),
		now, id)
	return err
}

func (s *SqliteStore) KnowledgeBitTags(
	ctx context.Context,
	bitIDs []string,
) (map[string][]string, error) {
	return knowledgeBitTags(ctx, s.knowledge(), bitIDs)
}

func (s *PgStore) KnowledgeBitTags(
	ctx context.Context,
	bitIDs []string,
) (map[string][]string, error) {
	return knowledgeBitTags(ctx, s.knowledge(), bitIDs)
}

func knowledgeBitTags(
	ctx context.Context,
	q knowledgeQueries,
	bitIDs []string,
) (map[string][]string, error) {
	out := make(map[string][]string, len(bitIDs))
	if len(bitIDs) == 0 {
		return out, nil
	}
	marks := make([]string, len(bitIDs))
	args := make([]any, len(bitIDs))
	for i, id := range bitIDs {
		marks[i] = "?"
		args[i] = id
	}
	rows, err := q.db.QueryContext(ctx, q.sql(
		`SELECT bit_id, tag FROM knowledge_bit_tags WHERE bit_id IN (`+
			strings.Join(marks, ", ")+`) ORDER BY bit_id, position`), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var bitID, tag string
		if err := rows.Scan(&bitID, &tag); err != nil {
			return nil, err
		}
		out[bitID] = append(out[bitID], tag)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeTagCounts(
	ctx context.Context,
	filter KnowledgeBitFilter,
) ([]KnowledgeTagCount, error) {
	return listKnowledgeTagCounts(ctx, s.knowledge(), filter)
}

func (s *PgStore) ListKnowledgeTagCounts(
	ctx context.Context,
	filter KnowledgeBitFilter,
) ([]KnowledgeTagCount, error) {
	return listKnowledgeTagCounts(ctx, s.knowledge(), filter)
}

func listKnowledgeTagCounts(
	ctx context.Context,
	q knowledgeQueries,
	filter KnowledgeBitFilter,
) ([]KnowledgeTagCount, error) {
	where := []string{"b.deleted_at IS NULL"}
	args := make([]any, 0, 4)
	switch {
	case filter.OrgOnly:
		where = append(where, "b.project_id IS NULL")
	case filter.ProjectID != nil:
		where = append(where, "b.project_id = ?")
		args = append(args, *filter.ProjectID)
	default:
		visible := make([]string, 0, len(filter.ProjectIDs))
		for _, id := range filter.ProjectIDs {
			if id != "" {
				visible = append(visible, "?")
				args = append(args, id)
			}
		}
		if len(visible) == 0 {
			where = append(where, "b.project_id IS NULL")
		} else {
			where = append(where, "(b.project_id IS NULL OR b.project_id IN ("+
				strings.Join(visible, ", ")+"))")
		}
	}
	if filter.Status != "" {
		where = append(where, "b.status = ?")
		args = append(args, filter.Status)
	} else {
		where = append(where, "b.status = 'approved'")
	}
	rows, err := q.db.QueryContext(ctx, q.sql(
		`SELECT t.tag, COUNT(*) FROM knowledge_bit_tags t
		JOIN knowledge_bits b ON b.id = t.bit_id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY t.tag ORDER BY COUNT(*) DESC, t.tag LIMIT 200`), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeTagCount, 0, 16)
	for rows.Next() {
		var item KnowledgeTagCount
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeBitSources(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitSourceRow, error) {
	return listKnowledgeBitSources(ctx, s.knowledge(), bitID)
}

func (s *PgStore) ListKnowledgeBitSources(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitSourceRow, error) {
	return listKnowledgeBitSources(ctx, s.knowledge(), bitID)
}

func listKnowledgeBitSources(
	ctx context.Context,
	q knowledgeQueries,
	bitID string,
) ([]KnowledgeBitSourceRow, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(
		`SELECT id, bit_id, kind, task_id, task_rowid, message_id, request_id,
		question_id, answered_by_actor_id, answered_by, prompt_id,
		prompt_scope, prompt_target_id, label, at, href, position
		FROM knowledge_bit_sources WHERE bit_id = ? ORDER BY position, id`), bitID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeBitSourceRow, 0, 4)
	for rows.Next() {
		var s KnowledgeBitSourceRow
		if err := rows.Scan(
			&s.ID, &s.BitID, &s.Kind, &s.TaskID, &s.TaskRowID, &s.MessageID,
			&s.RequestID, &s.QuestionID, &s.AnsweredByActorID, &s.AnsweredBy,
			&s.PromptID, &s.PromptScope, &s.PromptTargetID, &s.Label, &s.At,
			&s.Href, &s.Position,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeBitLinks(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitLinkRow, error) {
	return listKnowledgeBitLinks(ctx, s.knowledge(), bitID, false)
}

func (s *PgStore) ListKnowledgeBitLinks(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitLinkRow, error) {
	return listKnowledgeBitLinks(ctx, s.knowledge(), bitID, false)
}

func (s *SqliteStore) ListKnowledgeBitBacklinks(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitLinkRow, error) {
	return listKnowledgeBitLinks(ctx, s.knowledge(), bitID, true)
}

func (s *PgStore) ListKnowledgeBitBacklinks(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitLinkRow, error) {
	return listKnowledgeBitLinks(ctx, s.knowledge(), bitID, true)
}

func listKnowledgeBitLinks(
	ctx context.Context,
	q knowledgeQueries,
	bitID string,
	backlinks bool,
) ([]KnowledgeBitLinkRow, error) {
	owner, join := "l.bit_id", "b.id = l.target_id"
	if backlinks {
		owner, join = "l.target_id", "b.id = l.bit_id"
	}
	rows, err := q.db.QueryContext(ctx, q.sql(
		`SELECT b.id, l.kind, b.title, b.project_id, b.status
		FROM knowledge_bit_links l
		JOIN knowledge_bits b ON `+join+`
		WHERE `+owner+` = ? AND b.deleted_at IS NULL
		ORDER BY l.position, b.id`), bitID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeBitLinkRow, 0, 4)
	for rows.Next() {
		var link KnowledgeBitLinkRow
		if err := rows.Scan(
			&link.LinkedID, &link.Kind, &link.Title, &link.ProjectID,
			&link.Status,
		); err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

func (s *SqliteStore) ListKnowledgeBitRevisions(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitRevisionRow, error) {
	return listKnowledgeBitRevisions(ctx, s.knowledge(), bitID)
}

func (s *PgStore) ListKnowledgeBitRevisions(
	ctx context.Context,
	bitID string,
) ([]KnowledgeBitRevisionRow, error) {
	return listKnowledgeBitRevisions(ctx, s.knowledge(), bitID)
}

func listKnowledgeBitRevisions(
	ctx context.Context,
	q knowledgeQueries,
	bitID string,
) ([]KnowledgeBitRevisionRow, error) {
	rows, err := q.db.QueryContext(ctx, q.sql(
		`SELECT id, bit_id, revision, title, content, tags_json, status,
		actor_id, actor_ref, summary, created_at
		FROM knowledge_bit_revisions WHERE bit_id = ?
		ORDER BY revision DESC`), bitID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]KnowledgeBitRevisionRow, 0, 8)
	for rows.Next() {
		var rev KnowledgeBitRevisionRow
		if err := rows.Scan(
			&rev.ID, &rev.BitID, &rev.Revision, &rev.Title, &rev.Content,
			&rev.TagsJSON, &rev.Status, &rev.ActorID, &rev.ActorRef,
			&rev.Summary, &rev.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}

func insertKnowledgeBitTags(
	ctx context.Context,
	tx *sql.Tx,
	q knowledgeQueries,
	bitID string,
	tags []string,
) error {
	for i, tag := range tags {
		if _, err := tx.ExecContext(ctx, q.sql(
			`INSERT INTO knowledge_bit_tags (bit_id, tag, position)
			VALUES (?, ?, ?)`), bitID, tag, i); err != nil {
			return err
		}
	}
	return nil
}

func insertKnowledgeBitSource(
	ctx context.Context,
	tx *sql.Tx,
	q knowledgeQueries,
	source KnowledgeBitSourceRow,
) error {
	return tx.QueryRowContext(ctx, q.sql(
		`INSERT INTO knowledge_bit_sources (
		bit_id, kind, task_id, task_rowid, message_id, request_id, question_id,
		answered_by_actor_id, answered_by, prompt_id, prompt_scope,
		prompt_target_id, label, at, href, position
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		source.BitID, source.Kind, source.TaskID, source.TaskRowID,
		source.MessageID, source.RequestID, source.QuestionID,
		source.AnsweredByActorID, source.AnsweredBy, source.PromptID,
		source.PromptScope, source.PromptTargetID, source.Label, source.At,
		source.Href, source.Position,
	).Scan(&source.ID)
}

func insertKnowledgeBitLink(
	ctx context.Context,
	tx *sql.Tx,
	q knowledgeQueries,
	link KnowledgeBitLinkWrite,
) error {
	_, err := tx.ExecContext(ctx, q.sql(
		`INSERT INTO knowledge_bit_links (bit_id, target_id, kind, position)
		VALUES (?, ?, ?, ?)`),
		link.BitID, link.TargetID, link.Kind, link.Position)
	return err
}

func insertKnowledgeBitRevision(
	ctx context.Context,
	tx *sql.Tx,
	q knowledgeQueries,
	rev KnowledgeBitRevisionRow,
) (KnowledgeBitRevisionRow, error) {
	err := tx.QueryRowContext(ctx, q.sql(
		`INSERT INTO knowledge_bit_revisions (
		bit_id, revision, title, content, tags_json, status, actor_id,
		actor_ref, summary, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		rev.BitID, rev.Revision, rev.Title, rev.Content, rev.TagsJSON,
		rev.Status, rev.ActorID, rev.ActorRef, rev.Summary, rev.CreatedAt,
	).Scan(&rev.ID)
	return rev, err
}
