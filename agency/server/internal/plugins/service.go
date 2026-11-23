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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	PublicURL string

	DisableTicks bool
	// LogRetention is the per-plugin log line cap the maintenance pass applies

	LogRetention int
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

	LogRetention int

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
		force bool,
	) error

	RespondInput func(
		ctx context.Context,
		plugin Plugin,
		task tasks.Task,
		requestID string,
		answer TaskInputAnswer,
	) error

	RecordMessage func(
		ctx context.Context,
		taskID, messageID, text string,
		actorID ids.ActorID,
		at int64,
	)

	bundlesInDB bool

	eventsOnce sync.Once
	events     chan eventJob
}

func NewService(cfg Config, st *Store) *Service {
	return &Service{
		Config:         cfg,
		Store:          st,
		LogRetention:   cfg.LogRetention,
		Runtime:        NewRuntime(),
		bundlesInDB:    isPostgres(cfg.DSN),
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
		if s.bundlesInRegistry() {
			s.backfillBundle(ctx, p)
		}
		if !p.Enabled && p.LoadError == "" {

			continue
		}
		if !p.Enabled && !s.bundleAvailable(ctx, p) {

			continue
		}
		if err := s.ensureRuntime(ctx, p); err != nil {
			s.disableAfterLoadFailure(ctx, p, err)
			continue
		}
		if !p.Enabled {

			if err := s.Store.SetEnabled(ctx, p.ID, true); err != nil {
				log.Printf("[plugins] re-enable %s after a successful load: %v", p.ID, err)
				continue
			}
			p.Enabled = true
			s.Store.Log(
				ctx,
				p.ID,
				"info",
				"lifecycle",
				"",
				"plugin re-enabled after a load failure",
				map[string]any{"version": p.Version},
			)
		}
		s.clearLoadError(ctx, p.ID)
		s.startTicker(p)
	}
	return nil
}

func (s *Service) disableAfterLoadFailure(ctx context.Context, p Plugin, cause error) {
	s.stopTicker(p.ID)
	if err := s.Store.SetLoadFailure(ctx, p.ID, s.publicLoadError(cause)); err != nil {
		log.Printf("[plugins] record load failure %s: %v", p.ID, err)
	}
	if err := s.releasePluginHolds(ctx, p.ID); err != nil {
		log.Printf("[plugins] release holds of unloadable %s: %v", p.ID, err)
	}
	s.Runtime.Unload(p.ID)
	s.closeDB(p.ID)
	s.Store.Log(
		ctx,
		p.ID,
		"error",
		"lifecycle",
		"",
		"failed to load plugin",
		map[string]any{"error": cause.Error(), "disabled": true},
	)
}

func (s *Service) publicLoadError(err error) string {
	msg := err.Error()
	for _, secret := range []string{
		strings.TrimRight(s.Config.DataDir, string(os.PathSeparator)),
		strings.TrimSpace(s.Config.DSN),
	} {

		if len(secret) < 2 {
			continue
		}
		msg = strings.ReplaceAll(msg, secret+string(os.PathSeparator), "")
		msg = strings.ReplaceAll(msg, secret, "")
	}
	return msg
}

func (s *Service) clearLoadError(ctx context.Context, id string) {
	if err := s.Store.ClearLoadError(ctx, id); err != nil {
		log.Printf("[plugins] clear load failure %s: %v", id, err)
	}
}

func (s *Service) List(ctx context.Context) ([]Plugin, error) { return s.Store.List(ctx) }

const DefaultLogRetention = 20000

func (s *Service) PruneLogs(ctx context.Context) error {
	if s == nil || s.Store == nil {
		return nil
	}
	list, err := s.Store.List(ctx)
	if err != nil {
		return err
	}
	keep := s.LogRetention
	if keep <= 0 {
		keep = DefaultLogRetention
	}
	var errs []error
	for _, p := range list {
		if err := s.Store.PruneLogs(ctx, p.ID, int64(keep)); err != nil {
			errs = append(errs, fmt.Errorf("prune %s logs: %w", p.ID, err))
		}
	}
	return errors.Join(errs...)
}

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
	versionDir := s.versionDir(manifest.ID, manifest.Version)
	if err := writeVersionDir(versionDir, bundle); err != nil {
		return Plugin{}, err
	}
	normalized, err := jsonMarshalIndent(manifest)
	if err != nil {
		return Plugin{}, err
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
	recovered := false
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

		recovered = !existing.Enabled && existing.LoadError != ""
		if err := s.Store.Update(ctx, p); err != nil {
			return Plugin{}, err
		}
		if existing.Version != p.Version {
			s.Runtime.Unload(p.ID)
		}
	}

	if s.bundlesInRegistry() {
		if err := s.Store.PutBundle(ctx, p.ID, p.Version, bundle.Checksum, data); err != nil {
			log.Printf("[plugins] keep the stored bundle %s %s: %v", p.ID, p.Version, err)
		}
	}
	if err := s.ensureRuntime(ctx, p); err != nil {

		s.disableAfterLoadFailure(ctx, p, err)
		return Plugin{}, err
	}
	if recovered {
		if err := s.Store.SetEnabled(ctx, p.ID, true); err != nil {
			return Plugin{}, err
		}
		p.Enabled = true
		s.Store.Log(
			ctx,
			p.ID,
			"info",
			"lifecycle",
			"",
			"plugin re-enabled after a reinstall",
			map[string]any{"version": p.Version},
		)
	}
	s.clearLoadError(ctx, p.ID)
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

			s.disableAfterLoadFailure(ctx, p, err)
			return err
		}
		s.clearLoadError(ctx, id)
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

func (s *Service) versionDir(id, version string) string {
	return filepath.Join(s.Config.DataDir, "plugins", id, "versions", version)
}

var errNoStoredBundle = errors.New("the registry keeps no archive of this version")

func (s *Service) bundlesInRegistry() bool { return s.bundlesInDB }

func writeVersionDir(dir string, bundle Bundle) error {
	normalized, err := jsonMarshalIndent(bundle.Manifest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), normalized, 0o644); err != nil {
		return err
	}
	for name, body := range bundle.Files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) restoreVersion(ctx context.Context, p Plugin, versionDir string) error {
	stored, ok, err := s.Store.Bundle(ctx, p.ID, p.Version)
	if err != nil {
		return err
	}
	if !ok {
		return errNoStoredBundle
	}
	sum := sha256.Sum256(stored.Data)
	if got := hex.EncodeToString(sum[:]); got != stored.Checksum {
		return fmt.Errorf(
			"the stored archive checksum is %s, the registry says %s", got, stored.Checksum,
		)
	}
	bundle, err := ReadBundle(stored.Data)
	if err != nil {
		return fmt.Errorf("the stored archive is not a bundle: %w", err)
	}
	if bundle.Manifest.ID != p.ID || bundle.Manifest.Version != p.Version {
		return fmt.Errorf(
			"the stored archive is %s %s", bundle.Manifest.ID, bundle.Manifest.Version,
		)
	}
	staging := versionDir + ".restore"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := writeVersionDir(staging, bundle); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.RemoveAll(versionDir); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(staging, versionDir); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	s.Store.Log(
		ctx,
		p.ID,
		"info",
		"lifecycle",
		"",
		"restored the bundle from the registry",
		map[string]any{"version": p.Version},
	)
	return nil
}

func (s *Service) backfillBundle(ctx context.Context, p Plugin) {
	versionDir := s.versionDir(p.ID, p.Version)
	if _, err := os.Stat(filepath.Join(versionDir, p.Manifest.Entry)); err != nil {
		return
	}
	ok, err := s.Store.HasBundle(ctx, p.ID, p.Version)
	if err != nil {
		log.Printf("[plugins] look up the stored bundle %s %s: %v", p.ID, p.Version, err)
		return
	}
	if ok {
		return
	}
	data, err := PackInstalledVersion(versionDir)
	if err != nil {
		log.Printf("[plugins] backfill the stored bundle %s %s: %v", p.ID, p.Version, err)
		return
	}
	bundle, err := ReadBundle(data)
	if err != nil {
		log.Printf("[plugins] backfill the stored bundle %s %s: %v", p.ID, p.Version, err)
		return
	}
	if bundle.Manifest.ID != p.ID || bundle.Manifest.Version != p.Version {
		log.Printf(
			"[plugins] skip backfilling %s %s: the directory holds %s %s",
			p.ID, p.Version, bundle.Manifest.ID, bundle.Manifest.Version,
		)
		return
	}
	if err := s.Store.PutBundle(ctx, p.ID, p.Version, bundle.Checksum, data); err != nil {
		log.Printf("[plugins] backfill the stored bundle %s %s: %v", p.ID, p.Version, err)
		return
	}
	s.Store.Log(
		ctx,
		p.ID,
		"info",
		"lifecycle",
		"",
		"kept the installed bundle in the registry",
		map[string]any{"version": p.Version},
	)
}

func (s *Service) bundleAvailable(ctx context.Context, p Plugin) bool {
	versionDir := s.versionDir(p.ID, p.Version)
	if _, err := os.Stat(filepath.Join(versionDir, p.Manifest.Entry)); err == nil {
		return true
	}
	if !s.bundlesInRegistry() {
		return false
	}
	ok, err := s.Store.HasBundle(ctx, p.ID, p.Version)
	return err == nil && ok
}

func (s *Service) ensureRuntime(ctx context.Context, p Plugin) error {
	versionDir := s.versionDir(p.ID, p.Version)
	if _, err := os.Stat(filepath.Join(versionDir, p.Manifest.Entry)); err != nil {
		if !s.bundlesInRegistry() {
			return fmt.Errorf("plugin %s has no %s in %s", p.ID, p.Manifest.Entry, versionDir)
		}
		if rerr := s.restoreVersion(ctx, p, versionDir); rerr != nil {
			if !errors.Is(rerr, errNoStoredBundle) {
				return fmt.Errorf(
					"plugin %s has no %s in %s, and the registry could not restore it: %w",
					p.ID, p.Manifest.Entry, versionDir, rerr,
				)
			}
			return fmt.Errorf("plugin %s has no %s in %s", p.ID, p.Manifest.Entry, versionDir)
		}
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

func (s *Service) PublicURL() string {
	return strings.TrimRight(s.Config.PublicURL, "/")
}

func (s *Service) PreviewTemplate(
	_ context.Context,
	p Plugin,
	scope, key, source string,
	samples map[string]any,
) (TemplatePreview, error) {
	if !p.Manifest.HasSurface(scope) {
		return TemplatePreview{}, fmt.Errorf(
			"plugin %s has no %s settings", p.ID, scope,
		)
	}
	field, ok := p.Manifest.TemplateField(scope, key)
	if !ok {
		return TemplatePreview{}, fmt.Errorf(
			"plugin %s has no %s template %q", p.ID, scope, key,
		)
	}

	return field.Preview(source, samples), nil
}

func (s *Service) SettingsSchema(
	p Plugin, scope, target string,
) json.RawMessage {
	return p.Manifest.RenderSettingsSchema(scope, target, s.PublicURL())
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
	return s.updateSettings(ctx, id, scope, target, values, updatedBy, false)
}

func (s *Service) updateSettings(
	ctx context.Context,
	id, scope, target string,
	values map[string]any,
	updatedBy int64,
	inFlight bool,
) error {
	if values == nil {
		return errors.New("values must be an object")
	}
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
	if err := s.validateSettings(
		ctx, p, scope, target, values, previous, inFlight,
	); err != nil {
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
	inFlight bool,
) error {

	if errs := p.Manifest.ValidateSettingsTemplates(scope, values); len(errs) > 0 {
		return &SettingsRejected{Errors: errs}
	}
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
	if !inFlight {
		release, err := s.acquire(ctx, p)
		if err != nil {
			return err
		}
		defer release()
	}
	host := NewHost(
		ctx,
		p,
		Caller{Kind: "agency"},
		newInvocationID(),
		orgSettings,
		db,
		serviceBridge{svc: s, pluginID: p.ID, plugin: p},
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

func (s *Service) UpdateSettingsAsPlugin(
	ctx context.Context,
	pluginID, scope, target string,
	values map[string]any,
) error {
	if values == nil {
		return errors.New("values must be an object")
	}
	p, err := s.Get(ctx, pluginID)
	if err != nil {
		return ErrNotFound
	}
	scope = strings.TrimSpace(scope)
	target = strings.TrimSpace(target)
	if !p.Manifest.HasSurface(scope) {
		return fmt.Errorf("plugin %s has no %s settings", pluginID, scope)
	}
	switch scope {
	case ScopeOrg:

		target = ""
	case ScopeProject:
		if target == "" {
			return errors.New("the project target is required")
		}
		row, err := s.Store.ProjectSettings(ctx, pluginID, target)
		if err != nil {
			return err
		}
		if !row.Enabled {
			return errors.New("the plugin is not enabled in that project")
		}
		if s.Projects != nil {
			role, ok, err := s.Projects.EffectiveRole(
				ctx, target, ids.ActorID(p.ActorID), false,
			)
			if err != nil {
				return err
			}
			if !ok || role == "" {
				return errors.New("the plugin is not a member of that project")
			}
		}
	case ScopeAgent, ScopeUser:
		if target == "" {
			return fmt.Errorf("the %s target is required", scope)
		}
		row, err := s.Store.Settings(ctx, pluginID, scope, target)
		if err != nil {
			return err
		}
		if !row.Enabled {
			return fmt.Errorf("the plugin is not enabled for %s %s", scope, target)
		}
	default:
		return fmt.Errorf("unknown settings scope %q", scope)
	}
	return s.updateSettings(ctx, pluginID, scope, target, values, p.ActorID, true)
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
		previous := p.Manifest.ApplySettingsDefaults(ScopeProject, row.Values)
		for key := range secrets {
			if value, ok := values[key].(string); !ok || value == "" {
				if old, ok := previous[key]; ok {
					values[key] = old
				}
			}
		}
		if err := s.validateSettings(
			ctx, p, ScopeProject, projectID, values, previous, false,
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
		return p.Manifest.ApplySettingsDefaults(scope, row.Values), nil
	}
	row, err := s.Store.Settings(ctx, p.ID, scope, target)
	if err != nil {
		return nil, err
	}
	return p.Manifest.ApplySettingsDefaults(scope, row.Values), nil
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
		serviceBridge{svc: s, pluginID: p.ID, plugin: p},
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
