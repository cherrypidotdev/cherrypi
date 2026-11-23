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

package knowledge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/realtime"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

const (
	ScopeOrg = "org"

	ScopeProject = "project"

	ScopeAgent = "agent"

	StatusDraft      = "draft"
	StatusStable     = "stable"
	StatusDeprecated = "deprecated"

	StatusPending    = "pending"
	StatusApproved   = "approved"
	StatusRejected   = "rejected"
	StatusSuperseded = "superseded"

	TrustUnverified = "unverified"

	TrustMachineConfirmed = "machine-confirmed"

	TrustHumanReviewed = "human-reviewed"

	MemoryRoot = "memory/"
)

var (
	ErrNotFound = errors.New("knowledge not found")

	ErrForbidden = errors.New("not allowed to access this knowledge scope")

	ErrConflict = errors.New("knowledge changed; reload and retry")

	ErrInvalidPath = errors.New("invalid knowledge path")

	ErrInvalidInput = errors.New("invalid knowledge input")

	ErrProtected = errors.New("knowledge file is protected")

	ErrAgentMemoryOnly = errors.New("agent memory is writable only by its agent")

	ErrMemoryRecoveryNotFound = errors.New("memory recovery record not found")

	ErrReviewNotAllowed = errors.New("only a human may review a knowledge bit")

	ErrSourceInvalid = errors.New("knowledge source does not match a real input")

	ErrOriginTaskRequired = errors.New("an agent must record knowledge in a task")
)

var memoryPathPattern = regexp.MustCompile(`^memory(/[A-Za-z0-9._-]+)+$`)

var tagPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Service struct {
	store    store.Store
	projects *projects.Service
	agents   *agents.Registry
	accounts *accounts.Service
	tasks    *tasks.Service
	realtime *realtime.Bus
	now      func() time.Time
}

func New(
	st store.Store,
	projectsSvc *projects.Service,
	agentsReg *agents.Registry,
	accountsSvc *accounts.Service,
	tasksSvc *tasks.Service,
	bus *realtime.Bus,
) *Service {
	return &Service{
		store:    st,
		projects: projectsSvc,
		agents:   agentsReg,
		accounts: accountsSvc,
		tasks:    tasksSvc,
		realtime: bus,
		now:      time.Now,
	}
}

type Actor struct {
	ID       ids.ActorID
	Kind     string
	Username string
	Name     string

	Admin bool
}

func ActorFromIdentity(identity auth.Identity) Actor {
	return Actor{
		ID:       identity.ActorID,
		Kind:     identity.Kind,
		Username: identity.Username,
		Name:     identity.Name,
		Admin:    identity.IsAdmin(),
	}
}

func (a Actor) IsHuman() bool { return a.Kind == accounts.KindUser }

func (a Actor) Ref() string {
	switch a.Kind {
	case accounts.KindUser:
		if a.Username != "" {
			return "human:" + a.Username
		}
		return "human:unknown"
	case accounts.KindAgent:
		if a.Username != "" {
			return "cherrypi-agent/" + a.Username
		}
		return "cherrypi-agent/unknown"
	default:
		if a.Username != "" {
			return "process:" + a.Username
		}
		return "process:cherrypi"
	}
}

type ScopeRef struct {
	Scope   string
	ScopeID string
}

func (s ScopeRef) IsMemory() bool { return s.Scope == ScopeAgent }

type MemoryWriteInput struct {
	ProjectID string

	Target string

	Path string

	Content string

	Replace bool
}

type MemoryFile struct {
	Path      string
	Body      string
	Title     string
	UpdatedAt time.Time
	Revision  int64
}

type MemoryLayer struct {
	Name      string
	Path      string
	Chars     int
	Truncated bool
}

type MemoryContext struct {
	Text       string
	SnapshotAt time.Time
	Hash       string
	Layers     []MemoryLayer
	ProjectID  string
}

type MemoryStatus struct {
	ProjectID  string
	Files      []MemoryFile
	TotalChars int
	TodayPath  string
	Context    MemoryContext
}

func validScope(scope string) bool {
	return scope == ScopeOrg || scope == ScopeProject || scope == ScopeAgent
}

var ErrInvalidScope = errors.New("invalid knowledge scope")

func ValidatePath(path string) error {
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return ErrInvalidPath
	}
	if strings.Contains(path, "..") {
		return ErrInvalidPath
	}
	if !memoryPathPattern.MatchString(path) {
		return ErrInvalidPath
	}
	return nil
}

func (s *Service) validateScope(ref ScopeRef) error {
	if !validScope(ref.Scope) {
		return ErrInvalidScope
	}
	if ref.Scope == ScopeOrg {
		if ref.ScopeID != "" {
			return ErrInvalidInput
		}
		return nil
	}
	if ref.Scope == ScopeAgent {

		return nil
	}
	if ref.ScopeID == "" {
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) CanRead(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
) (bool, error) {
	if err := s.validateScope(ref); err != nil {
		return false, err
	}
	switch ref.Scope {
	case ScopeOrg:
		return actor.ID != 0, nil
	case ScopeProject:
		if s.projects == nil {
			return false, nil
		}
		_, err := s.projects.RequireRole(
			ctx, ref.ScopeID, actor.ID, actor.IsHuman() && actor.Admin,
			projects.RoleViewer,
		)
		if errors.Is(err, projects.ErrNotMember) || errors.Is(err, projects.ErrForbidden) {
			return false, nil
		}
		return err == nil, err
	case ScopeAgent:
		if ref.ScopeID == "" {
			return actor.Kind == accounts.KindAgent, nil
		}
		agentActorID := ids.ActorID(parseActorID(ref.ScopeID))
		if actor.ID == agentActorID {
			return true, nil
		}
		return s.isAgentOwnerOrAdmin(ctx, actor, agentActorID)
	default:
		return false, ErrInvalidScope
	}
}

func (s *Service) CanWrite(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
) (bool, error) {
	if err := s.validateScope(ref); err != nil {
		return false, err
	}
	switch ref.Scope {
	case ScopeOrg:
		return actor.ID != 0, nil
	case ScopeProject:
		if s.projects == nil {
			return false, nil
		}
		_, err := s.projects.RequireRole(
			ctx, ref.ScopeID, actor.ID, actor.IsHuman() && actor.Admin,
			projects.RoleMember,
		)
		if errors.Is(err, projects.ErrNotMember) || errors.Is(err, projects.ErrForbidden) {
			return false, nil
		}
		return err == nil, err
	case ScopeAgent:
		if ref.ScopeID == "" {
			return actor.Kind == accounts.KindAgent, nil
		}
		return actor.Kind == accounts.KindAgent &&
			actor.ID == ids.ActorID(parseActorID(ref.ScopeID)), nil
	default:
		return false, ErrInvalidScope
	}
}

func (s *Service) isAgentOwnerOrAdmin(
	ctx context.Context,
	actor Actor,
	agentActorID ids.ActorID,
) (bool, error) {
	if actor.IsHuman() && actor.Admin {
		return true, nil
	}
	if s.agents == nil || s.accounts == nil || agentActorID == 0 {
		return false, nil
	}
	account, err := s.accounts.GetActor(ctx, agentActorID)
	if err != nil || account.Kind != accounts.KindAgent {
		return false, nil
	}
	agent, err := s.agents.GetByActor(ctx, agentActorID)
	if err != nil {
		return false, nil
	}
	ok, err := s.agents.IsOwner(ctx, agent.ID, actor.ID)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func parseActorID(raw string) int64 {
	var id int64
	_, _ = fmt.Sscan(raw, &id)
	return id
}

func (s *Service) requireRead(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
) error {
	ok, err := s.CanRead(ctx, actor, ref)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireWrite(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
) error {
	ok, err := s.CanWrite(ctx, actor, ref)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func normalizeTags(tags []string) (string, []string, error) {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if !tagPattern.MatchString(tag) {
			return "", nil, fmt.Errorf(
				"%w: tag %q may only contain letters, digits, underscore and dash",
				ErrInvalidInput, tag,
			)
		}
		if len(tag) > 64 {
			return "", nil, fmt.Errorf("%w: tag %q is too long", ErrInvalidInput, tag)
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	raw, err := json.Marshal(out)
	return string(raw), out, err
}

func tagsJSONFor(tags []string) (string, error) {
	raw, _, err := normalizeTags(tags)
	return raw, err
}

func tagsFromJSON(raw string) []string {
	var tags []string
	if json.Unmarshal([]byte(raw), &tags) != nil {
		return nil
	}
	return tags
}

func nullActor(id ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0}
}

func conceptWriteFrom(c store.KnowledgeConcept) store.KnowledgeConceptWrite {
	return store.KnowledgeConceptWrite{
		Type:               c.Type,
		Title:              c.Title,
		Description:        c.Description,
		Resource:           c.Resource,
		TagsJson:           c.TagsJson,
		Body:               c.Body,
		FrontmatterJson:    c.FrontmatterJson,
		Status:             c.Status,
		StaleAfter:         c.StaleAfter,
		GeneratedByActorID: nullActor(ids.ActorID(c.GeneratedByActorID.Int64)),
		GeneratedBy:        c.GeneratedBy,
		GeneratedAt:        c.GeneratedAt,
		TrustTier:          c.TrustTier,
		Protected:          c.Protected,
	}
}

func (s *Service) signal() {
	if s.realtime != nil {
		s.realtime.Publish(agencyv1.RealtimeKind_REALTIME_KIND_KNOWLEDGE)
	}
}

func (s *Service) ProjectSlug(ctx context.Context, projectID string) (string, error) {
	if s.projects == nil || projectID == "" {
		return "", nil
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return project.Slug, nil
}

func renderFrontmatter(
	write store.KnowledgeConceptWrite,
	ref ScopeRef,
	path, extraJSON string,
) string {
	doc := map[string]any{}
	if extraJSON != "" {
		_ = json.Unmarshal([]byte(extraJSON), &doc)
	}
	doc["type"] = write.Type
	doc["title"] = write.Title
	doc["description"] = write.Description
	doc["resource"] = write.Resource
	doc["tags"] = tagsFromJSON(write.TagsJson)
	doc["status"] = write.Status
	doc["generated"] = map[string]any{
		"by": write.GeneratedBy,
		"at": time.UnixMilli(write.GeneratedAt).UTC().Format(time.RFC3339),
	}
	if write.StaleAfter.Valid {
		doc["stale_after"] = time.UnixMilli(write.StaleAfter.Int64).UTC().Format("2006-01-02")
	}
	doc["verified"] = []map[string]any{}
	doc["sources"] = []map[string]any{}
	doc["x-cherrypi"] = map[string]any{
		"scope":     ref.Scope,
		"scope_id":  ref.ScopeID,
		"path":      path,
		"protected": write.Protected != 0,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}
