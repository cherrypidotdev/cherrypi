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

package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/sealed"
	"cherrypi.vladfaust.com/agency/internal/store"
)

var ErrInvalid = errors.New("invalid library entry")

var ErrNotFound = errors.New("library entry not found")

var ErrForbidden = errors.New("you may not change this scope")

const (
	KindOrg     = "org"
	KindProject = "project"
	KindTeam    = "team"
	KindAgent   = "agent"
	KindGroup   = "group"
)

var Kinds = []string{KindOrg, KindProject, KindTeam, KindGroup, KindAgent}

const (
	KindMCP   = "mcp"
	KindSkill = "skill"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type Scope struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (s Scope) String() string {
	if s.ID == "" {
		return s.Kind
	}
	return s.Kind + ":" + s.ID
}

type Value struct {
	Value  string `json:"value,omitempty"`
	Env    string `json:"env,omitempty"`
	Secret string `json:"secret,omitempty"`

	HasSecret bool `json:"hasSecret,omitempty"`

	KeepSecret bool `json:"keepSecret,omitempty"`
}

type ToolFilter struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type Server struct {
	ID               int64            `json:"id"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Owner            Scope            `json:"owner"`
	URL              string           `json:"url"`
	Headers          map[string]Value `json:"headers"`
	ConnectTimeoutMs int64            `json:"connectTimeoutMs"`
	RequestTimeoutMs int64            `json:"requestTimeoutMs"`
	Tools            ToolFilter       `json:"tools"`
	DirectTools      bool             `json:"directTools"`
	Enabled          bool             `json:"enabled"`
	Installs         []Install        `json:"installs"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type SkillFile struct {
	Path string `json:"path"`
	Text string `json:"text,omitempty"`
}

type Skill struct {
	ID                     int64       `json:"id"`
	Name                   string      `json:"name"`
	Description            string      `json:"description"`
	Owner                  Scope       `json:"owner"`
	Body                   string      `json:"body"`
	Files                  []SkillFile `json:"files"`
	DisableModelInvocation bool        `json:"disableModelInvocation"`
	Enabled                bool        `json:"enabled"`
	Installs               []Install   `json:"installs"`
	UpdatedAt              time.Time   `json:"updatedAt"`
}

type Install struct {
	Scope   Scope `json:"scope"`
	Enabled bool  `json:"enabled"`
}

type Library struct {
	Servers []Server `json:"servers"`
	Skills  []Skill  `json:"skills"`

	Seal bool `json:"seal"`
}

type Input struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Owner       Scope     `json:"owner"`
	Enabled     *bool     `json:"enabled"`
	Installs    []Install `json:"installs"`

	URL              string           `json:"url"`
	Headers          map[string]Value `json:"headers"`
	ConnectTimeoutMs int64            `json:"connectTimeoutMs"`
	RequestTimeoutMs int64            `json:"requestTimeoutMs"`
	Tools            ToolFilter       `json:"tools"`
	DirectTools      *bool            `json:"directTools"`

	Body                   string      `json:"body"`
	Files                  []SkillFile `json:"files"`
	DisableModelInvocation bool        `json:"disableModelInvocation"`
}

type Caller struct {
	AgentActorID string
	ProjectID    string
	TeamIDs      []string
	GroupIDs     []string
}

type Service struct {
	store store.IntegrationsStore
	seal  *sealed.Sealer
	auth  *Authorizer
	now   func() time.Time
}

func New(
	st store.IntegrationsStore,
	secretKey string,
	auth *Authorizer,
) (*Service, error) {
	seal, err := sealed.New(secretKey)
	if err != nil {
		return nil, err
	}
	return &Service{store: st, seal: seal, auth: auth, now: time.Now}, nil
}

func (s *Service) Sealing() bool { return s.seal.Enabled() }

func (s *Service) Library(ctx context.Context, actorID int64) (Library, error) {
	servers, err := s.store.ListMcpServers(ctx)
	if err != nil {
		return Library{}, err
	}
	skills, err := s.store.ListSkillEntries(ctx)
	if err != nil {
		return Library{}, err
	}
	mcpInstalls, err := s.store.ListMcpInstalls(ctx)
	if err != nil {
		return Library{}, err
	}
	skillInstalls, err := s.store.ListSkillInstalls(ctx)
	if err != nil {
		return Library{}, err
	}

	out := Library{
		Servers: []Server{},
		Skills:  []Skill{},
		Seal:    s.Sealing(),
	}
	for _, row := range servers {
		visible, err := s.canSee(ctx, actorID, Scope{Kind: row.OwnerKind, ID: row.OwnerID})
		if err != nil {
			return Library{}, err
		}
		if !visible {
			continue
		}
		entry, err := s.serverOf(row, mcpInstalls)
		if err != nil {
			return Library{}, err
		}
		out.Servers = append(out.Servers, entry)
	}
	for _, row := range skills {
		visible, err := s.canSee(ctx, actorID, Scope{Kind: row.OwnerKind, ID: row.OwnerID})
		if err != nil {
			return Library{}, err
		}
		if !visible {
			continue
		}
		out.Skills = append(out.Skills, skillOf(row, skillInstalls))
	}
	return out, nil
}

func (s *Service) canSee(ctx context.Context, actorID int64, scope Scope) (bool, error) {
	if s.auth == nil {
		return false, nil
	}
	return s.auth.CanSee(ctx, actorID, scope)
}

func (s *Service) GetServer(ctx context.Context, actorID, id int64) (Server, error) {
	row, err := s.store.GetMcpServer(ctx, id)
	if err != nil {
		return Server{}, mapStoreError(err)
	}
	visible, err := s.canSee(ctx, actorID, Scope{Kind: row.OwnerKind, ID: row.OwnerID})
	if err != nil {
		return Server{}, err
	}
	if !visible {
		return Server{}, ErrNotFound
	}
	installs, err := s.store.ListMcpInstalls(ctx)
	if err != nil {
		return Server{}, err
	}
	return s.serverOf(row, installs)
}

func (s *Service) GetSkill(ctx context.Context, actorID, id int64) (Skill, error) {
	row, err := s.store.GetSkillEntry(ctx, id)
	if err != nil {
		return Skill{}, mapStoreError(err)
	}
	visible, err := s.canSee(ctx, actorID, Scope{Kind: row.OwnerKind, ID: row.OwnerID})
	if err != nil {
		return Skill{}, err
	}
	if !visible {
		return Skill{}, ErrNotFound
	}
	installs, err := s.store.ListSkillInstalls(ctx)
	if err != nil {
		return Skill{}, err
	}
	return skillOf(row, installs), nil
}

func (s *Service) SaveServer(
	ctx context.Context,
	actorID int64,
	input Input,
) (Server, error) {
	previous, err := s.storedServer(ctx, input.ID)
	if err != nil {
		return Server{}, err
	}
	if err := s.authorize(ctx, actorID, input, storedOwner(previous)); err != nil {
		return Server{}, err
	}
	params, err := s.serverParams(ctx, input, actorID, previous)
	if err != nil {
		return Server{}, err
	}
	keepInstalls := input.ID != 0
	save := store.SaveMcpServerParams{
		Entry: params,

		KeepInstalls: keepInstalls,
		Revision:     s.nextRevision(),
	}
	if !keepInstalls {
		save.Installs = installParams(input.Installs, actorID, s.now().UnixMilli())
	}
	saved, err := s.store.SaveMcpServer(ctx, save)
	if err != nil {
		return Server{}, mapUniqueName(err)
	}
	all, err := s.store.ListMcpInstalls(ctx)
	if err != nil {
		return Server{}, err
	}
	return s.serverOf(saved, all)
}

func (s *Service) SaveSkill(
	ctx context.Context,
	actorID int64,
	input Input,
) (Skill, error) {
	previous, err := s.storedSkill(ctx, input.ID)
	if err != nil {
		return Skill{}, err
	}
	if err := s.authorize(ctx, actorID, input, storedOwnerOfSkill(previous)); err != nil {
		return Skill{}, err
	}
	params, err := skillParams(input, actorID, s.now().UnixMilli())
	if err != nil {
		return Skill{}, err
	}
	save := store.SaveSkillEntryParams{
		Entry:        params,
		KeepInstalls: input.ID != 0,
		Revision:     s.nextRevision(),
	}
	if input.ID == 0 {
		save.Installs = installParams(input.Installs, actorID, s.now().UnixMilli())
	}
	saved, err := s.store.SaveSkillEntry(ctx, save)
	if err != nil {
		return Skill{}, mapUniqueName(err)
	}
	all, err := s.store.ListSkillInstalls(ctx)
	if err != nil {
		return Skill{}, err
	}
	return skillOf(saved, all), nil
}

func (s *Service) DeleteServer(ctx context.Context, actorID, id int64) error {
	entry, err := s.GetServer(ctx, actorID, id)
	if err != nil {
		return err
	}
	if err := s.auth.CanAuthor(ctx, actorID, entry.Owner); err != nil {
		return err
	}
	deleted, err := s.store.DeleteMcpServer(ctx, id, s.nextRevision())
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeleteSkill(ctx context.Context, actorID, id int64) error {
	entry, err := s.GetSkill(ctx, actorID, id)
	if err != nil {
		return err
	}
	if err := s.auth.CanAuthor(ctx, actorID, entry.Owner); err != nil {
		return err
	}
	deleted, err := s.store.DeleteSkillEntry(ctx, id, s.nextRevision())
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) SetInstall(
	ctx context.Context,
	actorID int64,
	kind string,
	id int64,
	scope Scope,
	enabled bool,
	remove bool,
) error {
	owner, err := s.entryOwner(ctx, kind, id)
	if err != nil {
		return err
	}
	visible, err := s.canSee(ctx, actorID, owner)
	if err != nil {
		return err
	}
	if !visible {
		return ErrNotFound
	}
	scope = normalizeScope(scope)

	if err := validScope(scope, true); err != nil {
		return err
	}
	if err := s.auth.CanAdminister(ctx, actorID, scope); err != nil {
		return err
	}
	if remove {
		return s.removeInstall(ctx, kind, id, scope)
	}
	return s.writeInstall(ctx, kind, id, scope, enabled, actorID)
}

func (s *Service) entryOwner(ctx context.Context, kind string, id int64) (Scope, error) {
	if kind == KindSkill {
		row, err := s.store.GetSkillEntry(ctx, id)
		if err != nil {
			return Scope{}, mapStoreError(err)
		}
		return Scope{Kind: row.OwnerKind, ID: row.OwnerID}, nil
	}
	row, err := s.store.GetMcpServer(ctx, id)
	if err != nil {
		return Scope{}, mapStoreError(err)
	}
	return Scope{Kind: row.OwnerKind, ID: row.OwnerID}, nil
}

func (s *Service) writeInstall(
	ctx context.Context,
	kind string,
	id int64,
	scope Scope,
	enabled bool,
	actorID int64,
) error {
	params := store.SetInstallParams{
		EntryID:          id,
		ScopeKind:        scope.Kind,
		ScopeID:          scope.ID,
		Enabled:          boolInt(enabled),
		CreatedByActorID: actorID,
		CreatedAt:        s.now().UnixMilli(),
		Revision:         s.nextRevision(),
	}
	if kind == KindSkill {
		return s.store.SetSkillInstall(ctx, params)
	}
	return s.store.SetMcpInstall(ctx, params)
}

func (s *Service) removeInstall(ctx context.Context, kind string, id int64, scope Scope) error {
	params := store.RemoveInstallParams{
		EntryID:   id,
		ScopeKind: scope.Kind,
		ScopeID:   scope.ID,
		Revision:  s.nextRevision(),
	}
	if kind == KindSkill {
		return s.store.RemoveSkillInstall(ctx, params)
	}
	return s.store.RemoveMcpInstall(ctx, params)
}

func (s *Service) storedSkill(
	ctx context.Context,
	id int64,
) (store.Skill, error) {
	if id == 0 {
		return store.Skill{}, nil
	}
	row, err := s.store.GetSkillEntry(ctx, id)
	if err != nil {
		return store.Skill{}, mapStoreError(err)
	}
	return row, nil
}

func (s *Service) storedServer(
	ctx context.Context,
	id int64,
) (store.McpServer, error) {
	if id == 0 {
		return store.McpServer{}, nil
	}
	row, err := s.store.GetMcpServer(ctx, id)
	if err != nil {
		return store.McpServer{}, mapStoreError(err)
	}
	return row, nil
}

func storedOwner(row store.McpServer) Scope {
	if row.ID == 0 {
		return Scope{}
	}
	return Scope{Kind: row.OwnerKind, ID: row.OwnerID}
}

func storedOwnerOfSkill(row store.Skill) Scope {
	if row.ID == 0 {
		return Scope{}
	}
	return Scope{Kind: row.OwnerKind, ID: row.OwnerID}
}

func (s *Service) authorize(
	ctx context.Context,
	actorID int64,
	input Input,
	stored Scope,
) error {
	if input.ID != 0 && len(input.Installs) > 0 {
		return fmt.Errorf(
			"%w: an edit does not change installs; use PUT/DELETE /api/rest/integrations/{servers,skills}/<id>/install",
			ErrInvalid,
		)
	}
	if stored.Kind != "" && stored != normalizeScope(input.Owner) {
		if err := s.auth.CanAuthor(ctx, actorID, stored); err != nil {
			return fmt.Errorf("%w: this entry is owned by %s", ErrForbidden, stored)
		}
	}
	if err := s.auth.CanAuthor(ctx, actorID, input.Owner); err != nil {
		return err
	}

	visible, err := s.canSee(ctx, actorID, normalizeScope(input.Owner))
	if err != nil {
		return err
	}
	if !visible {
		return fmt.Errorf("%w: you cannot read this entry", ErrForbidden)
	}
	for _, install := range input.Installs {
		if err := s.auth.CanAdminister(ctx, actorID, install.Scope); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) serverParams(
	ctx context.Context,
	input Input,
	actorID int64,
	previous store.McpServer,
) (store.McpServerParams, error) {
	name := strings.TrimSpace(input.Name)
	if !namePattern.MatchString(name) {
		return store.McpServerParams{}, fmt.Errorf(
			"%w: the name must be 1–64 letters, digits, dots, dashes or underscores",
			ErrInvalid,
		)
	}
	if err := validScope(input.Owner, true); err != nil {
		return store.McpServerParams{}, err
	}

	target, err := validEndpoint(input.URL)
	if err != nil {
		return store.McpServerParams{}, err
	}
	headers, err := s.encodeValues(input.Headers, previous.HeadersJson)
	if err != nil {
		return store.McpServerParams{}, err
	}
	now := s.now().UnixMilli()
	params := store.McpServerParams{
		ID:               input.ID,
		Name:             name,
		Description:      strings.TrimSpace(input.Description),
		OwnerKind:        input.Owner.Kind,
		OwnerID:          input.Owner.ID,
		Url:              target,
		HeadersJson:      headers,
		ConnectTimeoutMs: clampTimeout(input.ConnectTimeoutMs, defaultConnectMs),
		RequestTimeoutMs: clampTimeout(input.RequestTimeoutMs, defaultRequestMs),
		ToolsJson:        encodeJSON(input.Tools, "{}"),
		DirectTools:      boolInt(boolValue(input.DirectTools, true)),
		Enabled:          boolInt(boolValue(input.Enabled, true)),
		CreatedByActorID: actorID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return params, nil
}

func validEndpoint(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: an MCP entry needs the server's HTTP URL", ErrInvalid)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("%w: %q is not a URL", ErrInvalid, trimmed)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf(
			"%w: an MCP entry must be an http(s) URL", ErrInvalid,
		)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func skillParams(
	input Input,
	actorID int64,
	now int64,
) (store.SkillParams, error) {
	name := strings.TrimSpace(input.Name)
	if !namePattern.MatchString(name) {
		return store.SkillParams{}, fmt.Errorf(
			"%w: the name must be 1–64 letters, digits, dots, dashes or underscores",
			ErrInvalid,
		)
	}
	if strings.TrimSpace(input.Description) == "" {
		return store.SkillParams{}, fmt.Errorf(
			"%w: a skill needs a description; it is what the model sees first",
			ErrInvalid,
		)
	}
	if strings.TrimSpace(input.Body) == "" {
		return store.SkillParams{}, fmt.Errorf(
			"%w: a skill needs a body", ErrInvalid,
		)
	}
	if err := validScope(input.Owner, true); err != nil {
		return store.SkillParams{}, err
	}
	for _, file := range input.Files {
		path := strings.TrimSpace(file.Path)
		if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
			return store.SkillParams{}, fmt.Errorf(
				"%w: a supporting file needs a relative path", ErrInvalid,
			)
		}
	}
	return store.SkillParams{
		ID:                     input.ID,
		Name:                   name,
		Description:            strings.TrimSpace(input.Description),
		Body:                   input.Body,
		FilesJson:              encodeJSON(input.Files, "[]"),
		DisableModelInvocation: boolInt(input.DisableModelInvocation),
		OwnerKind:              input.Owner.Kind,
		OwnerID:                input.Owner.ID,
		Enabled:                boolInt(boolValue(input.Enabled, true)),
		CreatedByActorID:       actorID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}, nil
}

func (s *Service) encodeValues(
	values map[string]Value,
	previousJSON string,
) (string, error) {
	previous := map[string]Value{}
	if strings.TrimSpace(previousJSON) != "" {
		_ = json.Unmarshal([]byte(previousJSON), &previous)
	}
	out := map[string]Value{}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch {
		case strings.TrimSpace(value.Secret) != "":
			sealed, err := s.seal.Seal(value.Secret)
			if err != nil {
				return "", err
			}
			out[key] = Value{Secret: base64Std(sealed)}
		case value.HasSecret || (value.KeepSecret && previous[key].Secret != ""):
			kept, ok := previous[key]
			if !ok || kept.Secret == "" {
				return "", fmt.Errorf(
					"%w: %s has no stored secret to keep", ErrInvalid, key,
				)
			}
			out[key] = Value{Secret: kept.Secret}
		case strings.TrimSpace(value.Env) != "":
			out[key] = Value{Env: strings.TrimSpace(value.Env)}
		default:
			out[key] = Value{Value: value.Value}
		}
	}
	return encodeJSON(out, "{}"), nil
}

func installParams(installs []Install, actorID, now int64) []store.InstallParams {
	out := make([]store.InstallParams, 0, len(installs))
	seen := map[string]bool{}
	for _, install := range installs {
		scope := normalizeScope(install.Scope)
		if validScope(scope, true) != nil {
			continue
		}
		key := scope.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, store.InstallParams{
			ScopeKind:        scope.Kind,
			ScopeID:          scope.ID,
			Enabled:          boolInt(install.Enabled),
			CreatedByActorID: actorID,
			CreatedAt:        now,
		})
	}
	return out
}

func (s *Service) serverOf(
	row store.McpServer,
	installs []store.McpInstall,
) (Server, error) {
	out := Server{
		ID:               row.ID,
		Name:             row.Name,
		Description:      row.Description,
		Owner:            Scope{Kind: row.OwnerKind, ID: row.OwnerID},
		URL:              row.Url,
		ConnectTimeoutMs: row.ConnectTimeoutMs,
		RequestTimeoutMs: row.RequestTimeoutMs,
		DirectTools:      row.DirectTools != 0,
		Enabled:          row.Enabled != 0,
		UpdatedAt:        time.UnixMilli(row.UpdatedAt).UTC(),
	}
	out.Headers = decodeValues(row.HeadersJson, true)
	if err := json.Unmarshal([]byte(row.ToolsJson), &out.Tools); err != nil {
		out.Tools = ToolFilter{}
	}

	if out.Tools.Include == nil {
		out.Tools.Include = []string{}
	}
	if out.Tools.Exclude == nil {
		out.Tools.Exclude = []string{}
	}
	out.Installs = installsOf(row.ID, installs)
	return out, nil
}

func skillOf(row store.Skill, installs []store.SkillInstall) Skill {

	out := Skill{
		ID:                     row.ID,
		Name:                   row.Name,
		Description:            row.Description,
		Owner:                  Scope{Kind: row.OwnerKind, ID: row.OwnerID},
		Body:                   row.Body,
		Files:                  []SkillFile{},
		Installs:               []Install{},
		DisableModelInvocation: row.DisableModelInvocation != 0,
		Enabled:                row.Enabled != 0,
		UpdatedAt:              time.UnixMilli(row.UpdatedAt).UTC(),
	}
	if err := json.Unmarshal([]byte(row.FilesJson), &out.Files); err != nil {
		out.Files = []SkillFile{}
	}
	if out.Files == nil {
		out.Files = []SkillFile{}
	}
	for _, install := range installs {
		if install.EntryID != row.ID {
			continue
		}
		out.Installs = append(out.Installs, Install{
			Scope:   Scope{Kind: install.ScopeKind, ID: install.ScopeID},
			Enabled: install.Enabled != 0,
		})
	}
	sortInstalls(out.Installs)
	return out
}

func installsOf(entryID int64, rows []store.McpInstall) []Install {
	out := make([]Install, 0, len(rows))
	for _, row := range rows {
		if row.EntryID != entryID {
			continue
		}
		out = append(out, Install{
			Scope:   Scope{Kind: row.ScopeKind, ID: row.ScopeID},
			Enabled: row.Enabled != 0,
		})
	}
	sortInstalls(out)
	return out
}

func sortInstalls(installs []Install) {
	rank := map[string]int{}
	for i, kind := range Kinds {
		rank[kind] = i
	}
	sort.SliceStable(installs, func(i, j int) bool {
		if rank[installs[i].Scope.Kind] != rank[installs[j].Scope.Kind] {
			return rank[installs[i].Scope.Kind] < rank[installs[j].Scope.Kind]
		}
		return installs[i].Scope.ID < installs[j].Scope.ID
	})
}

func validScope(scope Scope, allowOrg bool) error {
	switch scope.Kind {
	case KindOrg:
		if !allowOrg {
			return fmt.Errorf("%w: the organization is not a target here", ErrInvalid)
		}
		return nil
	case KindProject:
		if strings.TrimSpace(scope.ID) == "" {
			return fmt.Errorf("%w: a project scope needs an id", ErrInvalid)
		}
		return nil
	case KindTeam, KindAgent, KindGroup:
		if _, err := strconv.ParseInt(scope.ID, 10, 64); err != nil {
			return fmt.Errorf(
				"%w: a %s scope needs a numeric id", ErrInvalid, scope.Kind,
			)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown scope %q", ErrInvalid, scope.Kind)
	}
}

func normalizeScope(scope Scope) Scope {
	scope.Kind = strings.TrimSpace(scope.Kind)
	scope.ID = strings.TrimSpace(scope.ID)
	if scope.Kind == KindOrg {
		scope.ID = ""
	}
	return scope
}

func (s *Service) nextRevision() string {
	return strconv.FormatInt(s.now().UnixNano(), 10)
}

func mapStoreError(err error) error {
	if errors.Is(err, errNoRows()) {
		return ErrNotFound
	}
	return err
}

func mapUniqueName(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf(
			"%w: another entry already uses this name; names are unique across the agency",
			ErrInvalid,
		)
	}
	return err
}

func decodeValues(raw string, masked bool) map[string]Value {
	out := map[string]Value{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]Value{}
	}
	for key, value := range out {
		if value.Secret == "" {
			continue
		}
		if masked {
			out[key] = Value{HasSecret: true}
		}
	}
	return out
}

func encodeJSON(value any, fallback string) string {
	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return fallback
	}
	return string(raw)
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func clampTimeout(value, fallback int64) int64 {
	if value <= 0 {
		return fallback
	}
	if value < minTimeoutMs {
		return minTimeoutMs
	}
	if value > maxTimeoutMs {
		return maxTimeoutMs
	}
	return value
}

const (
	defaultConnectMs int64 = 15_000
	defaultRequestMs int64 = 900_000
	minTimeoutMs     int64 = 1_000
	maxTimeoutMs     int64 = 3_600_000
)
