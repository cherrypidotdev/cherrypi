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
	"os"
	"path/filepath"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/tags"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/teams"
)

var ErrNotFound = errors.New("plugin not found")

type Config struct {
	DSN               string
	DataDir           string
	RequirePGDatabase bool

	AllowPrivateNetwork bool

	DisableTicks bool
}

type Service struct {
	Config  Config
	Store   *Store
	Runtime *Runtime

	Projects *projects.Service
	Agents   *agents.Registry
	Accounts *accounts.Service
	Tasks    *tasks.Service
	Settings *settings.Service
	Teams    *teams.Service
	Tags     *tags.Service

	mu    sync.Mutex
	dbs   map[string]*PluginDB
	slots map[string]chan struct{}

	taskToolWarned map[string]string

	tickers        map[string]*tickLoop
	tickBusy       map[string]bool
	tickSkipLogged map[string]bool
	rootCtx        context.Context
	rootCancel     context.CancelFunc
	Clock          TickClock
	Jitter         func(time.Duration) time.Duration

	FollowUp func(
		ctx context.Context,
		t tasks.Task,
		prompt string,
		actorID ids.ActorID,
	) error

	RecordMessage func(
		ctx context.Context,
		taskID, messageID, text string,
		actorID ids.ActorID,
		at int64,
	)

	eventsOnce sync.Once
	events     chan eventJob
}

func NewService(cfg Config, st *Store) *Service {
	return &Service{
		Config:         cfg,
		Store:          st,
		Runtime:        NewRuntime(),
		dbs:            map[string]*PluginDB{},
		slots:          map[string]chan struct{}{},
		tickers:        map[string]*tickLoop{},
		tickBusy:       map[string]bool{},
		tickSkipLogged: map[string]bool{},
		events:         make(chan eventJob, 256),
		taskToolWarned: map[string]string{},
	}
}

func (s *Service) Start(ctx context.Context) {
	s.eventsOnce.Do(func() {

		_, cancel := s.root()
		context.AfterFunc(ctx, cancel)
		for i := 0; i < 2; i++ {
			go s.eventWorker(ctx)
		}
	})
}

func (s *Service) Bootstrap(ctx context.Context) error {
	s.Start(ctx)
	list, err := s.Store.List(ctx)
	if err != nil {
		return err
	}
	for _, p := range list {
		if !p.Enabled {
			continue
		}
		if err := s.ensureRuntime(ctx, p); err != nil {
			s.Store.Log(
				ctx,
				p.ID,
				"error",
				"lifecycle",
				"",
				"failed to load plugin",
				map[string]any{"error": err.Error()},
			)
			continue
		}
		s.startTicker(p)
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]Plugin, error) { return s.Store.List(ctx) }

func (s *Service) Get(ctx context.Context, id string) (Plugin, error) {
	p, err := s.Store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Plugin{}, ErrNotFound
	}
	return p, err
}

func (s *Service) Install(
	ctx context.Context,
	installerID int64,
	data []byte,
) (Plugin, error) {
	bundle, err := ReadBundle(data)
	if err != nil {
		return Plugin{}, err
	}
	manifest := bundle.Manifest
	if ok, err := manifest.RequiresAPIVersion(APIVersion); err != nil {
		return Plugin{}, err
	} else if !ok {
		return Plugin{}, fmt.Errorf(
			"bundle requires agency %s, this agency implements %s",
			manifest.RequiresAgency,
			APIVersion,
		)
	}
	versionDir := filepath.Join(
		s.Config.DataDir,
		"plugins",
		manifest.ID,
		"versions",
		manifest.Version,
	)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return Plugin{}, err
	}
	normalized, err := jsonMarshalIndent(manifest)
	if err != nil {
		return Plugin{}, err
	}
	if err := os.WriteFile(
		filepath.Join(versionDir, "plugin.json"),
		normalized,
		0o644,
	); err != nil {
		return Plugin{}, err
	}
	for name, body := range bundle.Files {
		path := filepath.Join(versionDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Plugin{}, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return Plugin{}, err
		}
	}
	now := time.Now()
	p := Plugin{
		ID:          manifest.ID,
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: manifest.Description,
		Manifest:    manifest,
		ManifestRaw: normalized,
		Runtime:     manifest.Runtime,
		Entry:       manifest.Entry,
		Checksum:    bundle.Checksum,
		Enabled:     true,
		InstalledBy: installerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	existing, err := s.Store.Get(ctx, manifest.ID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := s.Store.Insert(ctx, p); err != nil {
			return Plugin{}, err
		}
	} else if err != nil {
		return Plugin{}, err
	} else {
		p.CreatedAt = existing.CreatedAt
		p.Enabled = existing.Enabled
		if err := s.Store.Update(ctx, p); err != nil {
			return Plugin{}, err
		}
		if existing.Version != p.Version {
			s.Runtime.Unload(p.ID)
		}
	}
	if err := s.ensureRuntime(ctx, p); err != nil {
		return Plugin{}, err
	}
	s.startTicker(p)
	s.Store.Log(
		ctx,
		p.ID,
		"info",
		"lifecycle",
		"",
		"plugin installed",
		map[string]any{"version": p.Version, "checksum": bundle.Checksum},
	)
	return s.Get(ctx, p.ID)
}

func (s *Service) SetEnabled(ctx context.Context, id string, enabled bool) error {
	p, err := s.Store.Get(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if enabled {
		p.Enabled = true
		if err := s.ensureRuntime(ctx, p); err != nil {
			return err
		}
	} else {

		if err := s.Store.SetEnabled(ctx, id, false); err != nil {
			return err
		}

		s.stopTicker(id)
		if err := s.releasePluginHolds(ctx, id); err != nil {
			s.Store.Log(ctx, id, "error", "lifecycle", "", "failed to release task holds",
				map[string]any{"error": err.Error()})
		}
		s.Runtime.Unload(id)
	}
	if enabled {
		if err := s.Store.SetEnabled(ctx, id, true); err != nil {
			return err
		}
		s.startTicker(p)
		return nil
	}
	s.closeDB(id)
	s.Store.Log(ctx, id, "info", "lifecycle", "", "plugin disabled", nil)
	return nil
}

func (s *Service) Uninstall(ctx context.Context, id string, purge bool) error {

	if err := s.Store.SetEnabled(ctx, id, false); err != nil {
		return err
	}
	s.stopTicker(id)

	if err := s.releasePluginHolds(ctx, id); err != nil {
		s.Store.Log(ctx, id, "error", "lifecycle", "", "failed to release task holds",
			map[string]any{"error": err.Error()})
	}
	s.Runtime.Unload(id)
	s.closeDB(id)
	if purge {
		if err := os.RemoveAll(filepath.Join(s.Config.DataDir, "plugins", id)); err != nil {
			return err
		}
	}
	return s.Store.Delete(ctx, id)
}

func (s *Service) ensureRuntime(ctx context.Context, p Plugin) error {
	versionDir := filepath.Join(
		s.Config.DataDir,
		"plugins",
		p.ID,
		"versions",
		p.Version,
	)
	if _, err := os.Stat(filepath.Join(versionDir, p.Manifest.Entry)); err != nil {
		return fmt.Errorf("plugin %s has no %s in %s", p.ID, p.Manifest.Entry, versionDir)
	}
	if err := s.Runtime.Load(p, versionDir); err != nil {
		return err
	}
	db, err := s.pluginDB(ctx, p)
	if err != nil {
		return err
	}
	return db.Migrate(ctx, migrationsDir(versionDir, isPostgres(s.Config.DSN)))
}

func (s *Service) pluginDB(ctx context.Context, p Plugin) (*PluginDB, error) {
	s.mu.Lock()
	if db := s.dbs[p.ID]; db != nil {
		s.mu.Unlock()
		return db, nil
	}
	s.mu.Unlock()
	db, err := OpenPluginDB(ctx, s.Config.DSN, p.ID, s.Config.DataDir, s.Config.RequirePGDatabase)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.dbs[p.ID] = db
	s.mu.Unlock()
	return db, nil
}

func (s *Service) closeDB(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if db := s.dbs[id]; db != nil {
		_ = db.Close()
		delete(s.dbs, id)
	}
}

func (s *Service) PluginIDByActor(ctx context.Context, actorID int64) (string, bool) {
	if s == nil || s.Store == nil {
		return "", false
	}
	return s.Store.PluginIDByActor(ctx, actorID)
}

func (s *Service) TaskPlugin(ctx context.Context, taskID string) (TaskPlugin, bool) {
	if s == nil || s.Store == nil {
		return TaskPlugin{}, false
	}
	row, ok, err := s.Store.TaskPlugin(ctx, taskID)
	if err != nil {
		return TaskPlugin{}, false
	}
	return row, ok
}

func (s *Service) SettingsFor(
	ctx context.Context,
	id, scope, target string,
) (map[string]any, map[string]bool, error) {
	p, err := s.Store.Get(ctx, id)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	if !p.Manifest.HasSurface(scope) {
		return nil, nil, fmt.Errorf("plugin %s has no %s settings", id, scope)
	}
	values, err := s.settingsForPlugin(ctx, p, scope, target)
	if err != nil {
		return nil, nil, err
	}
	return values, p.Manifest.SecretKeys(scope), nil
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	id, scope, target string,
	values map[string]any,
	updatedBy int64,
) error {
	p, err := s.Store.Get(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if !p.Manifest.HasSurface(scope) {
		return fmt.Errorf("plugin %s has no %s settings", id, scope)
	}
	secrets := p.Manifest.SecretKeys(scope)
	previous, err := s.settingsForPlugin(ctx, p, scope, target)
	if err != nil {
		return err
	}
	for key := range secrets {
		if value, ok := values[key].(string); !ok || value == "" {
			if old, ok := previous[key]; ok {
				values[key] = old
			}
		}
	}
	if err := s.validateSettings(ctx, p, scope, target, values, previous); err != nil {
		return err
	}
	if scope == ScopeProject {
		row, err := s.Store.ProjectSettings(ctx, id, target)
		if err != nil {
			return err
		}
		row.Values = values
		return s.Store.SetProjectSettings(ctx, row, updatedBy)
	}
	row, err := s.Store.Settings(ctx, id, scope, target)
	if err != nil {
		return err
	}
	row.Values = values

	row.Enabled = row.Enabled || scope == ScopeOrg
	return s.Store.SetSettings(ctx, row, updatedBy)
}

func (s *Service) validateSettings(
	ctx context.Context,
	p Plugin,
	scope, target string,
	values, previous map[string]any,
) error {
	if !p.Enabled {
		return nil
	}
	orgSettings, err := s.settingsForPlugin(ctx, p, ScopeOrg, "")
	if err != nil {
		return err
	}
	db, err := s.pluginDB(ctx, p)
	if err != nil {
		return err
	}
	release, err := s.acquire(ctx, p)
	if err != nil {
		return err
	}
	defer release()
	host := NewHost(
		ctx,
		p,
		Caller{Kind: "agency"},
		newInvocationID(),
		orgSettings,
		db,
		serviceBridge{svc: s, pluginID: p.ID},
		s.Config.AllowPrivateNetwork,
	)
	resp, err := s.Runtime.Call(ctx, p, host, Invocation{
		ID:       host.Invocation,
		Kind:     HandlerSettings,
		Caller:   Caller{Kind: "agency"},
		Scope:    scope,
		TargetID: target,
		Values:   values,
		Previous: previous,
	})
	if err != nil {
		return fmt.Errorf("update_settings: %w", err)
	}
	return parseSettingErrors(resp.Body)
}

func settingsRejectedOnly(err error) bool {
	if err == nil {
		return true
	}
	var rejected *SettingsRejected
	return errors.As(err, &rejected)
}

func parseSettingErrors(body []byte) error {
	if len(body) == 0 {
		return nil
	}

	var shape any
	if err := json.Unmarshal(body, &shape); err != nil {
		return fmt.Errorf("update_settings must return a list of {path, errorText}: %w", err)
	}
	switch value := shape.(type) {
	case nil:
		return nil
	case []any:
		if len(value) == 0 {
			return nil
		}
	case map[string]any:
		if len(value) == 0 {
			return nil
		}
	}
	var raw []struct {
		Path      string `json:"path"`
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("update_settings must return a list of {path, errorText}: %w", err)
	}
	out := make([]SettingError, 0, len(raw))
	for _, item := range raw {
		if item.Path == "" {
			return errors.New("update_settings returned an entry without a path")
		}
		if item.ErrorText == "" {
			return fmt.Errorf("update_settings returned an empty error for %q", item.Path)
		}
		out = append(out, SettingError{Path: item.Path, ErrorText: item.ErrorText})
	}
	return &SettingsRejected{Errors: out}
}

func (s *Service) SetSettingsEnabled(
	ctx context.Context,
	id, scope, target string,
	enabled bool,
	updatedBy int64,
) error {
	p, err := s.Store.Get(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if !p.Manifest.HasSurface(scope) {
		return fmt.Errorf("plugin %s has no %s settings", id, scope)
	}
	return s.Store.SetEnabledForTarget(ctx, id, scope, target, enabled, updatedBy)
}

func (s *Service) SettingsRow(
	ctx context.Context,
	id, scope, target string,
) (Settings, error) {
	return s.Store.Settings(ctx, id, scope, target)
}

func (s *Service) ProjectSettings(
	ctx context.Context,
	id, projectID string,
) (ProjectSettings, error) {
	return s.Store.ProjectSettings(ctx, id, projectID)
}

func (s *Service) UpdateProjectSettings(
	ctx context.Context,
	id, projectID string,
	enabled, allowTaskCreation bool,
	maxPriority int64,
	values map[string]any,
	updatedBy int64,
) error {
	p, err := s.Store.Get(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	row, err := s.Store.ProjectSettings(ctx, id, projectID)
	if err != nil {
		return err
	}
	if values == nil {
		values = row.Values
	} else {
		secrets := p.Manifest.SecretKeys(ScopeProject)
		previous := row.Values
		for key := range secrets {
			if value, ok := values[key].(string); !ok || value == "" {
				if old, ok := previous[key]; ok {
					values[key] = old
				}
			}
		}
		if err := s.validateSettings(
			ctx, p, ScopeProject, projectID, values, previous,
		); err != nil {
			return err
		}
	}
	row.Enabled = enabled
	row.AllowTaskCreation = allowTaskCreation
	row.MaxPriority = maxPriority
	row.Values = values
	return s.Store.SetProjectSettings(ctx, row, updatedBy)
}

func (s *Service) settingsForPlugin(
	ctx context.Context,
	p Plugin,
	scope, target string,
) (map[string]any, error) {
	if scope == ScopeProject {
		row, err := s.Store.ProjectSettings(ctx, p.ID, target)
		if err != nil {
			return nil, err
		}
		return row.Values, nil
	}
	row, err := s.Store.Settings(ctx, p.ID, scope, target)
	if err != nil {
		return nil, err
	}
	return row.Values, nil
}

func (s *Service) AgentSettings(ctx context.Context, id string) ([]Settings, error) {
	return s.Store.ListSettings(ctx, id, ScopeAgent)
}

func (s *Service) Logs(
	ctx context.Context,
	id string,
	limit int,
	beforeID int64,
) ([]LogLine, error) {
	return s.Store.Logs(ctx, id, limit, beforeID)
}

func (s *Service) RuntimeLogs(
	ctx context.Context,
	id string,
	filter RuntimeLogFilter,
) ([]LogLine, error) {
	return s.Store.RuntimeLogs(ctx, id, filter)
}

func (s *Service) Invoke(ctx context.Context, p Plugin, inv Invocation) (Response, error) {
	if !p.Enabled && !inv.allowDisabled {
		return Response{}, errors.New("plugin is disabled")
	}
	settings, err := s.settingsForPlugin(ctx, p, ScopeOrg, "")
	if err != nil {
		return Response{}, err
	}
	db, err := s.pluginDB(ctx, p)
	if err != nil {
		return Response{}, err
	}
	release, err := s.acquire(ctx, p)
	if err != nil {
		return Response{}, err
	}
	defer release()

	host := NewHost(
		ctx,
		p,
		inv.Caller,
		inv.ID,
		settings,
		db,
		serviceBridge{svc: s, pluginID: p.ID},
		s.Config.AllowPrivateNetwork,
	)
	host.TaskID = inv.TaskID
	return s.Runtime.Call(ctx, p, host, inv)
}

func (s *Service) InvokeTool(
	ctx context.Context,
	pluginID, taskID, tool string,
	args map[string]any,
) (string, error) {
	p, err := s.Get(ctx, pluginID)
	if err != nil {
		return "", err
	}
	owner, err := s.ResolveTool(ctx, taskID, tool)
	if err != nil {
		return "", err
	}
	if owner != p.ID {
		return "", errors.New("the tool is not attached to this task")
	}
	resp, err := s.Invoke(ctx, p, Invocation{
		ID:     newInvocationID(),
		Kind:   HandlerTool,
		Caller: Caller{Kind: "agent"},
		TaskID: taskID,
		Tool:   tool,
		Args:   args,
	})
	if err != nil {
		return "", err
	}
	return string(resp.Body), nil
}

func (s *Service) acquire(ctx context.Context, p Plugin) (func(), error) {
	limit := defaultConcurrent
	if p.Manifest.Limits.MaxConcurrent > 0 {
		limit = p.Manifest.Limits.MaxConcurrent
	}
	s.mu.Lock()
	slot := s.slots[p.ID]
	if slot == nil || cap(slot) != limit {
		slot = make(chan struct{}, limit)
		s.slots[p.ID] = slot
	}
	s.mu.Unlock()
	select {
	case slot <- struct{}{}:
		return func() { <-slot }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func jsonMarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
