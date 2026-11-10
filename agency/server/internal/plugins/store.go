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

package plugins

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/telemetry"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Plugin struct {
	ID            string
	Name          string
	Version       string
	Description   string
	Manifest      Manifest
	ManifestRaw   []byte
	Runtime       string
	Entry         string
	Checksum      string
	Enabled       bool
	InstalledBy   int64
	ActorID       int64
	ActorUsername string
	ActorName     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Settings struct {
	PluginID string
	Scope    string
	TargetID string
	Values   map[string]any
	Enabled  bool
	Updated  time.Time
}

type ProjectSettings struct {
	PluginID          string
	ProjectID         string
	Enabled           bool
	AllowTaskCreation bool
	MaxPriority       int64
	Values            map[string]any
	Updated           time.Time
}

// LogSurface is one settings surface a runtime log line addresses. TargetID is

type LogSurface struct {
	Scope    string `json:"scope"`
	TargetID string `json:"target_id"`
}

type SettingError struct {
	Path      string `json:"path"`
	ErrorText string `json:"error_text"`
}

type SettingsRejected struct {
	Errors []SettingError
}

// Error summarises the rejection for a log line or an API message.
func (e *SettingsRejected) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].ErrorText
	}
	return fmt.Sprintf("%d settings were rejected", len(e.Errors))
}

type LogLine struct {
	ID           int64        `json:"id"`
	At           time.Time    `json:"at"`
	Level        string       `json:"level"`
	Kind         string       `json:"kind"`
	InvocationID string       `json:"invocation_id"`
	Message      string       `json:"message"`
	Data         string       `json:"data"`
	Surfaces     []LogSurface `json:"surfaces,omitempty"`
}

// runtime line for the plugin. A BeforeID returns lines older than that id,

type RuntimeLogFilter struct {
	Scope    string
	TargetID string
	MinLevel string
	Search   string
	BeforeID int64
	Limit    int
}

type TaskPlugin struct {
	TaskID   string
	PluginID string
	Tools    []Tool
}

type Store struct {
	db       *sql.DB
	postgres bool
}

func OpenStore(dsn string) (*Store, error) {
	postgres := isPostgres(dsn)
	driver := "sqlite"
	if postgres {
		driver = "pgx"
	} else {
		sep := "?"
		if strings.ContainsRune(dsn, '?') {
			sep = "&"
		}
		dsn += sep + "_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}
	system := semconv.DBSystemNameSQLite
	if postgres {
		system = semconv.DBSystemNamePostgreSQL
	}

	pool := telemetry.SQLPoolAttribute("plugin-registry")
	db, err := telemetry.SQLOpen(driver, dsn, system, pool)
	if err != nil {
		return nil, err
	}
	if !postgres {
		db.SetMaxOpenConns(1)
	}
	telemetry.RegisterDBStats(db, dsn, system, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, postgres: postgres}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ready() bool { return s != nil && s.db != nil }

func (s *Store) q(query string) string {
	if !s.postgres {
		return query
	}
	var b strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

const pluginSelect = `SELECT p.id,p.name,p.version,p.description,p.manifest_json,p.runtime,p.entry,p.checksum,p.enabled,
	COALESCE(p.installed_by_actor_id,0),COALESCE(pa.actor_id,0),COALESCE(a.username,''),COALESCE(a.name,''),p.created_at,p.updated_at
	FROM plugins p
	LEFT JOIN plugin_actors pa ON pa.plugin_id=p.id
	LEFT JOIN actors a ON a.id=pa.actor_id`

func (s *Store) List(ctx context.Context) ([]Plugin, error) {
	rows, err := s.db.QueryContext(ctx, pluginSelect+` ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Plugin
	for rows.Next() {
		p, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (Plugin, error) {
	row := s.db.QueryRowContext(ctx, s.q(pluginSelect+` WHERE p.id=?`), id)
	return scanPlugin(row)
}

type scanner interface{ Scan(...any) error }

func scanPlugin(row scanner) (Plugin, error) {
	var p Plugin
	var manifest string
	var enabled int
	var created, updated int64
	if err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Version,
		&p.Description,
		&manifest,
		&p.Runtime,
		&p.Entry,
		&p.Checksum,
		&enabled,
		&p.InstalledBy,
		&p.ActorID,
		&p.ActorUsername,
		&p.ActorName,
		&created,
		&updated,
	); err != nil {
		return Plugin{}, err
	}
	p.Enabled = enabled != 0
	p.ManifestRaw = []byte(manifest)
	if err := json.Unmarshal(p.ManifestRaw, &p.Manifest); err != nil {
		return Plugin{}, fmt.Errorf("decode stored manifest %s: %w", p.ID, err)
	}
	p.CreatedAt = time.UnixMilli(created)
	p.UpdatedAt = time.UnixMilli(updated)
	return p, nil
}

func (s *Store) Insert(ctx context.Context, p Plugin) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := p.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}
	manifest, err := json.Marshal(p.Manifest)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		s.q(
			`INSERT INTO plugins (id,name,version,description,manifest_json,runtime,entry,checksum,enabled,installed_by_actor_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		),
		p.ID,
		p.Name,
		p.Version,
		p.Description,
		string(manifest),
		p.Runtime,
		p.Entry,
		p.Checksum,
		boolInt(p.Enabled),
		nullInt(p.InstalledBy),
		now.UnixMilli(),
		now.UnixMilli(),
	); err != nil {
		return err
	}
	actorID, err := ensurePluginActor(ctx, tx, s.postgres, p.ID, p.Name, now)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		s.q(`INSERT INTO plugin_actors (plugin_id,actor_id,created_at) VALUES (?,?,?)`),
		p.ID,
		actorID,
		now.UnixMilli(),
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		s.q(
			`INSERT INTO plugin_settings (plugin_id,scope,target_id,values_json,enabled,updated_at) VALUES (?,'org','','{}',1,?)`,
		),
		p.ID,
		now.UnixMilli(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func ensurePluginActor(
	ctx context.Context,
	tx *sql.Tx,
	postgres bool,
	pluginID, name string,
	now time.Time,
) (int64, error) {
	username := "plugin-" + pluginID
	lookup := `SELECT id,kind FROM actors WHERE username=?`
	if postgres {
		lookup = `SELECT id,kind FROM actors WHERE username=$1`
	}
	var id int64
	var kind string
	err := tx.QueryRowContext(ctx, lookup, username).Scan(&id, &kind)
	if err == nil {
		if kind != "plugin" {
			return 0, fmt.Errorf("actor %q already exists as %s", username, kind)
		}
		if postgres {
			_, err = tx.ExecContext(
				ctx,
				`UPDATE actors SET name=$2,updated_at=$3 WHERE id=$1`,
				id,
				name,
				now.UnixMilli(),
			)
			return id, err
		}
		_, err = tx.ExecContext(
			ctx,
			`UPDATE actors SET name=?,updated_at=? WHERE id=?`,
			name,
			now.UnixMilli(),
			id,
		)
		return id, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if postgres {
		err = tx.QueryRowContext(
			ctx,
			`INSERT INTO actors (kind,username,name,created_at,updated_at) VALUES ('plugin',$1,$2,$3,$4) RETURNING id`,
			username,
			name,
			now.UnixMilli(),
			now.UnixMilli(),
		).Scan(&id)
		return id, err
	}
	res, err := tx.ExecContext(
		ctx,
		`INSERT INTO actors (kind,username,name,created_at,updated_at) VALUES ('plugin',?,?,?,?)`,
		username,
		name,
		now.UnixMilli(),
		now.UnixMilli(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) Update(ctx context.Context, p Plugin) error {
	manifest, err := json.Marshal(p.Manifest)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		s.q(
			`UPDATE plugins SET name=?,version=?,description=?,manifest_json=?,runtime=?,entry=?,checksum=?,updated_at=? WHERE id=?`,
		),
		p.Name,
		p.Version,
		p.Description,
		string(manifest),
		p.Runtime,
		p.Entry,
		p.Checksum,
		time.Now().UnixMilli(),
		p.ID,
	)
	return err
}

func (s *Store) SetEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := s.db.ExecContext(
		ctx,
		s.q(`UPDATE plugins SET enabled=?,updated_at=? WHERE id=?`),
		boolInt(enabled),
		time.Now().UnixMilli(),
		id,
	)
	return err
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM plugins WHERE id=?`), id)
	return err
}

func (s *Store) ActorID(ctx context.Context, id string) (int64, error) {
	var actorID int64
	err := s.db.QueryRowContext(
		ctx,
		s.q(`SELECT actor_id FROM plugin_actors WHERE plugin_id=?`),
		id,
	).Scan(&actorID)
	return actorID, err
}

func (s *Store) PluginIDByActor(ctx context.Context, actorID int64) (string, bool) {
	var id string
	err := s.db.QueryRowContext(
		ctx,
		s.q(`SELECT plugin_id FROM plugin_actors WHERE actor_id=?`),
		actorID,
	).Scan(&id)
	if err != nil {
		return "", false
	}
	return id, true
}

func (s *Store) Settings(
	ctx context.Context,
	id, scope, target string,
) (Settings, error) {
	var row Settings
	var raw string
	var enabled int
	var updated int64
	err := s.db.QueryRowContext(
		ctx,
		s.q(
			`SELECT plugin_id,scope,target_id,values_json,enabled,updated_at FROM plugin_settings WHERE plugin_id=? AND scope=? AND target_id=?`,
		),
		id,
		scope,
		target,
	).Scan(&row.PluginID, &row.Scope, &row.TargetID, &raw, &enabled, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{PluginID: id, Scope: scope, TargetID: target, Values: map[string]any{}}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	row.Enabled = enabled != 0
	row.Updated = time.UnixMilli(updated)
	if err := json.Unmarshal([]byte(raw), &row.Values); err != nil {
		return Settings{}, fmt.Errorf("decode plugin settings %s: %w", id, err)
	}
	return row, nil
}

func (s *Store) ListSettings(ctx context.Context, id, scope string) ([]Settings, error) {
	rows, err := s.db.QueryContext(
		ctx,
		s.q(
			`SELECT plugin_id,scope,target_id,values_json,enabled,updated_at FROM plugin_settings WHERE plugin_id=? AND scope=? ORDER BY target_id`,
		),
		id,
		scope,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Settings
	for rows.Next() {
		var row Settings
		var raw string
		var enabled int
		var updated int64
		if err := rows.Scan(
			&row.PluginID,
			&row.Scope,
			&row.TargetID,
			&raw,
			&enabled,
			&updated,
		); err != nil {
			return nil, err
		}
		row.Enabled = enabled != 0
		row.Updated = time.UnixMilli(updated)
		if err := json.Unmarshal([]byte(raw), &row.Values); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) SetSettings(
	ctx context.Context,
	row Settings,
	updatedBy int64,
) error {
	raw, err := json.Marshal(row.Values)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(
		ctx,
		s.q(
			`INSERT INTO plugin_settings (plugin_id,scope,target_id,values_json,enabled,updated_by_actor_id,updated_at) VALUES (?,?,?,?,?,?,?) ON CONFLICT(plugin_id,scope,target_id) DO UPDATE SET values_json=excluded.values_json,enabled=excluded.enabled,updated_by_actor_id=excluded.updated_by_actor_id,updated_at=excluded.updated_at`,
		),
		row.PluginID,
		row.Scope,
		row.TargetID,
		string(raw),
		boolInt(row.Enabled),
		nullInt(updatedBy),
		now,
	)
	return err
}

func (s *Store) SetEnabledForTarget(
	ctx context.Context,
	id, scope, target string,
	enabled bool,
	updatedBy int64,
) error {
	row, err := s.Settings(ctx, id, scope, target)
	if err != nil {
		return err
	}
	row.Enabled = enabled
	return s.SetSettings(ctx, row, updatedBy)
}

func (s *Store) ProjectSettings(
	ctx context.Context,
	id, projectID string,
) (ProjectSettings, error) {
	var row ProjectSettings
	var raw string
	var enabled, allow int
	var updated int64
	err := s.db.QueryRowContext(
		ctx,
		s.q(
			`SELECT plugin_id,project_id,enabled,allow_task_creation,max_priority,values_json,updated_at FROM plugin_project_settings WHERE plugin_id=? AND project_id=?`,
		),
		id,
		projectID,
	).Scan(&row.PluginID, &row.ProjectID, &enabled, &allow, &row.MaxPriority, &raw, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectSettings{
			PluginID:          id,
			ProjectID:         projectID,
			AllowTaskCreation: true,
			Values:            map[string]any{},
		}, nil
	}
	if err != nil {
		return ProjectSettings{}, err
	}
	row.Enabled = enabled != 0
	row.AllowTaskCreation = allow != 0
	row.Updated = time.UnixMilli(updated)
	if err := json.Unmarshal([]byte(raw), &row.Values); err != nil {
		return ProjectSettings{}, fmt.Errorf("decode plugin project settings %s: %w", id, err)
	}
	return row, nil
}

func (s *Store) ListProjectSettings(
	ctx context.Context,
	projectID string,
) ([]ProjectSettings, error) {
	rows, err := s.db.QueryContext(
		ctx,
		s.q(
			`SELECT plugin_id,project_id,enabled,allow_task_creation,max_priority,values_json,updated_at FROM plugin_project_settings WHERE project_id=? ORDER BY plugin_id`,
		),
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []ProjectSettings
	for rows.Next() {
		var row ProjectSettings
		var raw string
		var enabled, allow int
		var updated int64
		if err := rows.Scan(
			&row.PluginID,
			&row.ProjectID,
			&enabled,
			&allow,
			&row.MaxPriority,
			&raw,
			&updated,
		); err != nil {
			return nil, err
		}
		row.Enabled = enabled != 0
		row.AllowTaskCreation = allow != 0
		row.Updated = time.UnixMilli(updated)
		if err := json.Unmarshal([]byte(raw), &row.Values); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) SetProjectSettings(
	ctx context.Context,
	row ProjectSettings,
	updatedBy int64,
) error {
	raw, err := json.Marshal(row.Values)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(
		ctx,
		s.q(
			`INSERT INTO plugin_project_settings (plugin_id,project_id,enabled,allow_task_creation,max_priority,values_json,updated_by_actor_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT(plugin_id,project_id) DO UPDATE SET enabled=excluded.enabled,allow_task_creation=excluded.allow_task_creation,max_priority=excluded.max_priority,values_json=excluded.values_json,updated_by_actor_id=excluded.updated_by_actor_id,updated_at=excluded.updated_at`,
		),
		row.PluginID,
		row.ProjectID,
		boolInt(row.Enabled),
		boolInt(row.AllowTaskCreation),
		row.MaxPriority,
		string(raw),
		nullInt(updatedBy),
		now,
		now,
	)
	return err
}

func (s *Store) TaskPlugin(ctx context.Context, taskID string) (TaskPlugin, bool, error) {
	var row TaskPlugin
	var raw string
	err := s.db.QueryRowContext(
		ctx,
		s.q(`SELECT task_id,plugin_id,tools_json FROM task_plugins WHERE task_id=?`),
		taskID,
	).Scan(&row.TaskID, &row.PluginID, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskPlugin{}, false, nil
	}
	if err != nil {
		return TaskPlugin{}, false, err
	}
	if err := json.Unmarshal([]byte(raw), &row.Tools); err != nil {
		return TaskPlugin{}, false, err
	}
	return row, true, nil
}

func (s *Store) SetTaskPlugin(
	ctx context.Context,
	taskID, pluginID string,
	tools []Tool,
) error {
	raw, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		s.q(
			`INSERT INTO task_plugins (task_id,plugin_id,tools_json,created_at) VALUES (?,?,?,?) ON CONFLICT(task_id) DO UPDATE SET plugin_id=excluded.plugin_id,tools_json=excluded.tools_json`,
		),
		taskID,
		pluginID,
		string(raw),
		time.Now().UnixMilli(),
	)
	return err
}

func (s *Store) SetTaskBadges(
	ctx context.Context,
	taskID, pluginID string,
	badges []TaskBadge,
) error {
	if len(badges) == 0 {
		_, err := s.db.ExecContext(
			ctx,
			s.q(`DELETE FROM plugin_task_badges WHERE task_id=? AND plugin_id=?`),
			taskID,
			pluginID,
		)
		return err
	}
	raw, err := json.Marshal(badges)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		s.q(
			`INSERT INTO plugin_task_badges (task_id,plugin_id,badges_json,updated_at) VALUES (?,?,?,?) ON CONFLICT(task_id,plugin_id) DO UPDATE SET badges_json=excluded.badges_json,updated_at=excluded.updated_at`,
		),
		taskID,
		pluginID,
		string(raw),
		time.Now().UnixMilli(),
	)
	return err
}

func (s *Store) TaskBadges(
	ctx context.Context,
	taskIDs []string,
) ([]StoredBadges, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(taskIDs))
	args := make([]any, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := s.db.QueryContext(
		ctx,
		s.q(
			`SELECT task_id,plugin_id,badges_json FROM plugin_task_badges WHERE task_id IN (`+
				strings.Join(placeholders, ",")+`) ORDER BY task_id,plugin_id`,
		),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []StoredBadges
	for rows.Next() {
		var row StoredBadges
		var raw string
		if err := rows.Scan(&row.TaskID, &row.PluginID, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &row.Badges); err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) Log(
	ctx context.Context,
	id, level, kind, invocation, message string,
	data any,
) {
	raw, _ := json.Marshal(data)
	_, _ = s.db.ExecContext(
		ctx,
		s.q(
			`INSERT INTO plugin_logs (plugin_id,at,level,kind,invocation_id,message,data_json) VALUES (?,?,?,?,?,?,?)`,
		),
		id,
		time.Now().UnixMilli(),
		level,
		kind,
		invocation,
		message,
		string(raw),
	)
}

func (s *Store) Logs(
	ctx context.Context,
	id string,
	limit int,
	beforeID int64,
) ([]LogLine, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id,at,level,kind,invocation_id,message,data_json FROM plugin_logs WHERE plugin_id=? AND kind<>'runtime'`
	args := []any{id}
	if beforeID > 0 {
		query += ` AND id < ?`
		args = append(args, beforeID)
	}
	query += ` ORDER BY at DESC,id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, s.q(query), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []LogLine
	for rows.Next() {
		var l LogLine
		var at int64
		if err := rows.Scan(
			&l.ID,
			&at,
			&l.Level,
			&l.Kind,
			&l.InvocationID,
			&l.Message,
			&l.Data,
		); err != nil {
			return nil, err
		}
		l.At = time.UnixMilli(at)
		out = append(out, l)
	}
	return out, rows.Err()
}

// LogRuntime appends one runtime line and its surfaces in a transaction.
func (s *Store) LogRuntime(
	ctx context.Context,
	id, invocation, level, message string,
	data any,
	surfaces []LogSurface,
) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UnixMilli()
	var logID int64
	if s.postgres {
		err = tx.QueryRowContext(
			ctx,
			`INSERT INTO plugin_logs (plugin_id,at,level,kind,invocation_id,message,data_json) VALUES ($1,$2,$3,'runtime',$4,$5,$6) RETURNING id`,
			id,
			now,
			level,
			invocation,
			message,
			string(raw),
		).Scan(&logID)
	} else {
		var res sql.Result
		res, err = tx.ExecContext(
			ctx,
			`INSERT INTO plugin_logs (plugin_id,at,level,kind,invocation_id,message,data_json) VALUES (?,?,?,'runtime',?,?,?)`,
			id,
			now,
			level,
			invocation,
			message,
			string(raw),
		)
		if err == nil {
			logID, err = res.LastInsertId()
		}
	}
	if err != nil {
		return err
	}
	for _, surface := range surfaces {
		if _, err := tx.ExecContext(
			ctx,
			s.q(
				`INSERT INTO plugin_log_surfaces (log_id,plugin_id,scope,target_id) VALUES (?,?,?,?)`,
			),
			logID,
			id,
			surface.Scope,
			surface.TargetID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RuntimeLogs(
	ctx context.Context,
	id string,
	filter RuntimeLogFilter,
) ([]LogLine, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT l.id,l.at,l.level,l.kind,l.invocation_id,l.message,l.data_json FROM plugin_logs l`
	args := []any{}
	if filter.Scope != "" {
		query += ` JOIN plugin_log_surfaces s ON s.log_id=l.id AND s.scope=? AND s.target_id=?`
		args = append(args, filter.Scope, filter.TargetID)
	}
	query += ` WHERE l.plugin_id=? AND l.kind='runtime'`
	args = append(args, id)
	if levels := levelsAtLeast(filter.MinLevel); len(levels) > 0 {
		query += ` AND l.level IN (` + placeholders(len(levels)) + `)`
		for _, level := range levels {
			args = append(args, level)
		}
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		query += ` AND LOWER(l.message) LIKE ?`
		args = append(args, "%"+strings.ToLower(search)+"%")
	}
	if filter.BeforeID > 0 {
		query += ` AND l.id < ?`
		args = append(args, filter.BeforeID)
	}
	query += ` ORDER BY l.at DESC,l.id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, s.q(query), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []LogLine{}
	for rows.Next() {
		var l LogLine
		var at int64
		if err := rows.Scan(
			&l.ID,
			&at,
			&l.Level,
			&l.Kind,
			&l.InvocationID,
			&l.Message,
			&l.Data,
		); err != nil {
			return nil, err
		}
		l.At = time.UnixMilli(at)
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	surfaces, err := s.logSurfaces(ctx, out)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Surfaces = surfaces[out[i].ID]
	}
	return out, nil
}

func (s *Store) logSurfaces(
	ctx context.Context,
	rows []LogLine,
) (map[int64][]LogSurface, error) {
	out := map[int64][]LogSurface{}
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]any, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	query := `SELECT log_id,scope,target_id FROM plugin_log_surfaces WHERE log_id IN (` +
		placeholders(len(ids)) + `) ORDER BY scope,target_id`
	res, err := s.db.QueryContext(ctx, s.q(query), ids...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Close() }()
	for res.Next() {
		var logID int64
		var surface LogSurface
		if err := res.Scan(&logID, &surface.Scope, &surface.TargetID); err != nil {
			return nil, err
		}
		out[logID] = append(out[logID], surface)
	}
	return out, res.Err()
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
