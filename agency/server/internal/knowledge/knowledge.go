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
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/realtime"
	"cherrypi.vladfaust.com/agency/internal/store"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"github.com/google/uuid"
)

const (
	ScopeOrg = "org"

	ScopeProject = "project"

	ScopeAgent = "agent"

	StatusDraft      = "draft"
	StatusStable     = "stable"
	StatusDeprecated = "deprecated"

	TrustUnverified = "unverified"

	TrustMachineConfirmed = "machine-confirmed"

	TrustHumanReviewed = "human-reviewed"

	MemoryRoot = "memory/"
)

var (
	ErrNotFound = errors.New("knowledge concept not found")

	ErrForbidden = errors.New("not allowed to access this knowledge scope")

	ErrConflict = errors.New("knowledge concept changed; reload and retry")

	ErrInvalidPath = errors.New("invalid knowledge concept path")

	ErrInvalidInput = errors.New("invalid knowledge input")

	ErrProtected = errors.New("knowledge page is protected")

	ErrAgentMemoryOnly = errors.New("agent memory is writable only by its agent")

	ErrVerifyNotAllowed = errors.New("only a human may verify a knowledge page")

	ErrMemoryRecoveryNotFound = errors.New("memory recovery record not found")
)

var pathPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)

var memoryPathPattern = regexp.MustCompile(`^memory(/[A-Za-z0-9._-]+)+$`)

var wikiLinkPattern = regexp.MustCompile(`\[\[([a-z0-9][a-z0-9._/-]*)\]\]`)

type Service struct {
	store    store.Store
	projects *projects.Service
	agents   *agents.Registry
	accounts *accounts.Service
	realtime *realtime.Bus
	now      func() time.Time
}

func New(
	st store.Store,
	projectsSvc *projects.Service,
	agentsReg *agents.Registry,
	accountsSvc *accounts.Service,
	bus *realtime.Bus,
) *Service {
	return &Service{
		store:    st,
		projects: projectsSvc,
		agents:   agentsReg,
		accounts: accountsSvc,
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

type ConceptView struct {
	Concept       store.KnowledgeConcept
	Tags          []string
	Sources       []store.KnowledgeSource
	Links         []store.KnowledgeLink
	Backlinks     []store.KnowledgeConcept
	Verifications []store.KnowledgeVerification

	Decisions []store.KnowledgeDecision
}

type SourceInput struct {
	ID           string
	Resource     string
	Title        string
	AuthorRef    string
	UsageCount   *int64
	LastModified *int64
}

type LinkInput struct {
	TargetPath string
	Kind       string
}

type CreateInput struct {
	Scope       ScopeRef
	Path        string
	Type        string
	Title       string
	Description string
	Resource    string
	Body        string
	Status      string
	Tags        []string
	StaleAfter  *time.Time

	ExtraFrontmatterJSON string
	Sources              []SourceInput
	Links                []LinkInput

	DecisionIDs []int64
	Summary     string
}

type Patch struct {
	Path        *string
	Type        *string
	Title       *string
	Description *string
	Resource    *string
	Body        *string
	Status      *string
	Tags        *[]string
	StaleAfter  *time.Time
	ClearStale  bool

	ExtraFrontmatterJSON *string
	Sources              *[]SourceInput
	Links                *[]LinkInput

	DecisionIDs  *[]int64
	BaseRevision int64
	Summary      string
}

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

func ValidatePath(path string) error {
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return ErrInvalidPath
	}
	if strings.Contains(path, "..") {
		return ErrInvalidPath
	}
	if strings.HasPrefix(path, MemoryRoot) {
		if !memoryPathPattern.MatchString(path) {
			return ErrInvalidPath
		}
		return nil
	}
	if !pathPattern.MatchString(path) {
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

var ErrInvalidScope = errors.New("invalid knowledge scope")

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

func normalizeRef(actor Actor, ref ScopeRef) ScopeRef {
	if ref.Scope == ScopeAgent && ref.ScopeID == "" && actor.Kind == accounts.KindAgent {
		ref.ScopeID = fmt.Sprint(actor.ID)
	}
	return ref
}

func normalizeTags(tags []string) (string, []string, error) {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tag, "#")))
		if tag == "" {
			continue
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
	_ = json.Unmarshal([]byte(raw), &tags)
	if tags == nil {
		tags = []string{}
	}
	return tags
}

func normalizeStatus(status string) (string, error) {
	switch status {
	case "":
		return StatusStable, nil
	case StatusDraft, StatusStable, StatusDeprecated:
		return status, nil
	default:
		return "", fmt.Errorf("%w: unknown status %q", ErrInvalidInput, status)
	}
}

func validateSources(sources []SourceInput) error {
	for _, source := range sources {
		resource := strings.TrimSpace(source.Resource)
		if resource == "" {
			continue
		}
		parsed, err := url.Parse(resource)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("%w: a source resource must be an http(s) URL", ErrInvalidInput)
		}
	}
	return nil
}

func (s *Service) validateDecisionIDs(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	ids []int64,
) ([]int64, error) {
	out := make([]int64, 0, len(ids))
	seen := map[int64]struct{}{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		row, err := s.store.GetKnowledgeDecision(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: decision %d", ErrDecisionNotFound, id)
		}
		if err != nil {
			return nil, err
		}
		decisionProject := ""
		if row.ProjectID.Valid {
			decisionProject = row.ProjectID.String
		}
		pageProject := ""
		if ref.Scope == ScopeProject {
			pageProject = ref.ScopeID
		}
		if decisionProject != pageProject {
			return nil, fmt.Errorf(
				"%w: decision %d belongs to another scope", ErrInvalidInput, id,
			)
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) Create(
	ctx context.Context,
	actor Actor,
	in CreateInput,
) (ConceptView, error) {
	if err := s.validateScope(in.Scope); err != nil {
		return ConceptView{}, err
	}
	if in.Scope.IsMemory() {
		return ConceptView{}, ErrAgentMemoryOnly
	}
	if err := s.requireWrite(ctx, actor, in.Scope); err != nil {
		return ConceptView{}, err
	}
	if err := ValidatePath(in.Path); err != nil {
		return ConceptView{}, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return ConceptView{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return ConceptView{}, err
	}
	tagsJSON, err := tagsJSONFor(in.Tags)
	if err != nil {
		return ConceptView{}, err
	}
	if err := validateSources(in.Sources); err != nil {
		return ConceptView{}, err
	}
	decisionIDs, err := s.validateDecisionIDs(ctx, actor, in.Scope, in.DecisionIDs)
	if err != nil {
		return ConceptView{}, err
	}

	_, err = s.store.GetKnowledgeConcept(ctx, in.Scope.Scope, in.Scope.ScopeID, in.Path)
	if err == nil {
		return ConceptView{}, ErrConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ConceptView{}, err
	}
	byPath, err := s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
		Scopes:         []string{in.Scope.Scope},
		ScopeID:        in.Scope.ScopeID,
		PathPrefix:     in.Path,
		IncludeDeleted: true,
		Limit:          5,
	})
	if err == nil {
		for _, old := range byPath {
			if old.Path == in.Path && old.DeletedAt.Valid {
				return s.revive(ctx, actor, old, in, status, tagsJSON)
			}
		}
	}

	now := s.now().UTC()
	id := uuid.NewString()
	write := store.KnowledgeConceptWrite{
		Type:               defaultType(in.Type),
		Title:              strings.TrimSpace(in.Title),
		Description:        strings.TrimSpace(in.Description),
		Resource:           in.Resource,
		TagsJson:           tagsJSON,
		Body:               in.Body,
		Status:             status,
		StaleAfter:         nullTime(in.StaleAfter),
		GeneratedByActorID: nullActor(actor.ID),
		GeneratedBy:        actor.Ref(),
		GeneratedAt:        now.UnixMilli(),
		TrustTier:          TrustUnverified,
		Protected:          0,
	}
	write.FrontmatterJson = renderFrontmatter(
		write,
		in.Scope,
		in.Path,
		in.ExtraFrontmatterJSON,
		nil,
		in.Sources,
		nil,
	)
	concept := store.KnowledgeConcept{
		ID:               id,
		Scope:            in.Scope.Scope,
		ScopeID:          in.Scope.ScopeID,
		Path:             in.Path,
		Revision:         1,
		CreatedByActorID: nullActor(actor.ID),
		CreatedAt:        now.UnixMilli(),
		UpdatedAt:        now.UnixMilli(),
	}
	created, err := s.store.CreateKnowledgeConcept(ctx, concept, write)
	if err != nil {
		return ConceptView{}, err
	}
	if _, err := s.store.CreateKnowledgeRevision(ctx, store.KnowledgeRevision{
		ConceptID:       created.ID,
		Revision:        created.Revision,
		Title:           created.Title,
		Body:            created.Body,
		FrontmatterJson: created.FrontmatterJson,
		Status:          created.Status,
		ActorID:         nullActor(actor.ID),
		ActorRef:        actor.Ref(),
		Summary:         in.Summary,
		CreatedAt:       created.CreatedAt,
	}); err != nil {
		return ConceptView{}, err
	}
	if err := s.store.ReplaceKnowledgeSources(
		ctx,
		created.ID,
		sourceRows(created.ID, in.Sources),
	); err != nil {
		return ConceptView{}, err
	}
	links := linkRows(created.ID, deriveLinks(created.Body, in.Links))
	if err := s.store.ReplaceKnowledgeLinks(ctx, created.ID, links); err != nil {
		return ConceptView{}, err
	}
	if len(decisionIDs) > 0 {
		if err := s.store.ReplaceKnowledgeConceptDecisions(
			ctx, created.ID, decisionIDs,
		); err != nil {
			return ConceptView{}, err
		}
	}
	s.signal()
	return s.view(ctx, created, actor)
}

func (s *Service) revive(
	ctx context.Context,
	actor Actor,
	old store.KnowledgeConcept,
	in CreateInput,
	status, tagsJSON string,
) (ConceptView, error) {
	now := s.now().UTC()
	write := store.KnowledgeConceptWrite{
		Type:               defaultType(in.Type),
		Title:              strings.TrimSpace(in.Title),
		Description:        strings.TrimSpace(in.Description),
		Resource:           in.Resource,
		TagsJson:           tagsJSON,
		Body:               in.Body,
		Status:             status,
		StaleAfter:         nullTime(in.StaleAfter),
		GeneratedByActorID: nullActor(actor.ID),
		GeneratedBy:        actor.Ref(),
		GeneratedAt:        now.UnixMilli(),
		TrustTier:          TrustUnverified,
		DeletedAt:          sql.NullInt64{},
	}
	write.FrontmatterJson = renderFrontmatter(
		write,
		in.Scope,
		in.Path,
		in.ExtraFrontmatterJSON,
		nil,
		in.Sources,
		nil,
	)
	rev := store.KnowledgeRevision{
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Summary:   in.Summary,
		CreatedAt: now.UnixMilli(),
	}
	rev.Title, rev.Body, rev.FrontmatterJson, rev.Status = write.Title, write.Body, write.FrontmatterJson, write.Status
	updated, err := s.store.SaveKnowledgeConcept(ctx, old.ID, old.Revision, write, rev)
	if err != nil {
		return ConceptView{}, err
	}
	if err := s.store.ReplaceKnowledgeSources(
		ctx,
		updated.ID,
		sourceRows(updated.ID, in.Sources),
	); err != nil {
		return ConceptView{}, err
	}
	if err := s.store.ReplaceKnowledgeLinks(
		ctx,
		updated.ID,
		linkRows(updated.ID, deriveLinks(updated.Body, in.Links)),
	); err != nil {
		return ConceptView{}, err
	}
	decisionIDs, err := s.validateDecisionIDs(ctx, actor, in.Scope, in.DecisionIDs)
	if err != nil {
		return ConceptView{}, err
	}
	if len(decisionIDs) > 0 {
		if err := s.store.ReplaceKnowledgeConceptDecisions(
			ctx, updated.ID, decisionIDs,
		); err != nil {
			return ConceptView{}, err
		}
	}
	s.signal()
	return s.view(ctx, updated, actor)
}

func (s *Service) Get(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path string,
) (ConceptView, error) {
	ref = normalizeRef(actor, ref)
	if err := s.requireRead(ctx, actor, ref); err != nil {
		return ConceptView{}, err
	}
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	if errors.Is(err, sql.ErrNoRows) {
		return ConceptView{}, ErrNotFound
	}
	if err != nil {
		return ConceptView{}, err
	}
	return s.view(ctx, concept, actor)
}

func (s *Service) view(
	ctx context.Context,
	concept store.KnowledgeConcept,
	actor Actor,
) (ConceptView, error) {
	sources, err := s.store.ListKnowledgeSources(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	links, err := s.store.ListKnowledgeLinks(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	backlinks, err := s.store.ListKnowledgeBacklinks(ctx, concept.Path)
	if err != nil {
		return ConceptView{}, err
	}
	verifications, err := s.store.ListKnowledgeVerifications(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	decisions, err := s.store.ListKnowledgeConceptDecisions(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	return ConceptView{
		Concept:       concept,
		Tags:          tagsFromJSON(concept.TagsJson),
		Sources:       sources,
		Links:         links,
		Backlinks:     backlinks,
		Verifications: verifications,
		Decisions:     decisions,
	}, nil
}

func (s *Service) List(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	query, pathPrefix, status, tag string,
	limit int,
) ([]store.KnowledgeConcept, error) {
	ref = normalizeRef(actor, ref)
	if err := s.requireRead(ctx, actor, ref); err != nil {
		return nil, err
	}
	return s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
		Scopes:     []string{ref.Scope},
		ScopeID:    ref.ScopeID,
		PathPrefix: pathPrefix,
		Query:      query,
		Status:     status,
		Tag:        tag,
		Limit:      limit,
	})
}

func (s *Service) Update(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path string,
	patch Patch,
) (ConceptView, error) {
	ref = normalizeRef(actor, ref)
	if err := s.requireWrite(ctx, actor, ref); err != nil {
		return ConceptView{}, err
	}
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	if errors.Is(err, sql.ErrNoRows) {
		return ConceptView{}, ErrNotFound
	}
	if err != nil {
		return ConceptView{}, err
	}
	if concept.Protected != 0 && !actor.IsHuman() {
		return ConceptView{}, ErrProtected
	}
	if patch.BaseRevision > 0 && patch.BaseRevision != concept.Revision {
		return ConceptView{}, ErrConflict
	}

	nextPath := concept.Path
	if patch.Path != nil && *patch.Path != concept.Path {
		if err := s.validateRename(ctx, actor, ref, concept, *patch.Path); err != nil {
			return ConceptView{}, err
		}
		nextPath = *patch.Path
	}
	write := store.KnowledgeConceptWrite{
		Type:               concept.Type,
		Title:              concept.Title,
		Description:        concept.Description,
		Resource:           concept.Resource,
		TagsJson:           concept.TagsJson,
		Body:               concept.Body,
		Status:             concept.Status,
		StaleAfter:         concept.StaleAfter,
		GeneratedByActorID: nullActor(actor.ID),
		GeneratedBy:        actor.Ref(),
		GeneratedAt:        s.now().UTC().UnixMilli(),
		TrustTier:          concept.TrustTier,
		Protected:          concept.Protected,
	}
	if patch.Type != nil {
		write.Type = defaultType(*patch.Type)
	}
	if patch.Title != nil {
		if strings.TrimSpace(*patch.Title) == "" {
			return ConceptView{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
		}
		write.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		write.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Resource != nil {
		write.Resource = *patch.Resource
	}
	if patch.Body != nil {
		write.Body = *patch.Body
	}
	if patch.Status != nil {
		status, err := normalizeStatus(*patch.Status)
		if err != nil {
			return ConceptView{}, err
		}
		write.Status = status
	}
	if patch.Tags != nil {
		tagsJSON, err := tagsJSONFor(*patch.Tags)
		if err != nil {
			return ConceptView{}, err
		}
		write.TagsJson = tagsJSON
	}
	if patch.ClearStale {
		write.StaleAfter = sql.NullInt64{}
	} else if patch.StaleAfter != nil {
		write.StaleAfter = nullTime(patch.StaleAfter)
	}

	verifications, err := s.store.ListKnowledgeVerifications(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	sources := sourcesToInput(concept.ID, mustSources(ctx, s.store, concept.ID))
	if patch.Sources != nil {
		if err := validateSources(*patch.Sources); err != nil {
			return ConceptView{}, err
		}
		sources = *patch.Sources
	}
	extra := extraFromFrontmatter(concept.FrontmatterJson)
	if patch.ExtraFrontmatterJSON != nil {
		extra = *patch.ExtraFrontmatterJSON
	}
	write.FrontmatterJson = renderFrontmatter(
		write,
		ref,
		nextPath,
		extra,
		verifications,
		sources,
		nil,
	)

	rev := store.KnowledgeRevision{
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Summary:   patch.Summary,
		CreatedAt: write.GeneratedAt,
	}
	rev.Title, rev.Body, rev.FrontmatterJson, rev.Status = write.Title, write.Body, write.FrontmatterJson, write.Status

	if nextPath != concept.Path {
		return s.rename(ctx, actor, ref, concept, nextPath, write, sources, patch)
	}

	updated, err := s.store.SaveKnowledgeConcept(ctx, concept.ID, concept.Revision, write, rev)
	if err != nil {
		if errors.Is(err, store.ErrKnowledgeConflict) {
			return ConceptView{}, ErrConflict
		}
		return ConceptView{}, err
	}
	if err := s.store.ReplaceKnowledgeSources(
		ctx,
		updated.ID,
		sourceRows(updated.ID, sources),
	); err != nil {
		return ConceptView{}, err
	}
	links := deriveLinks(updated.Body, nil)
	if patch.Links != nil {
		links = *patch.Links
	}
	if err := s.store.ReplaceKnowledgeLinks(
		ctx,
		updated.ID,
		linkRows(updated.ID, links),
	); err != nil {
		return ConceptView{}, err
	}
	if patch.DecisionIDs != nil {
		decisionIDs, err := s.validateDecisionIDs(ctx, actor, ref, *patch.DecisionIDs)
		if err != nil {
			return ConceptView{}, err
		}
		if err := s.store.ReplaceKnowledgeConceptDecisions(
			ctx, updated.ID, decisionIDs,
		); err != nil {
			return ConceptView{}, err
		}
	}
	s.signal()
	return s.view(ctx, updated, actor)
}

func (s *Service) rename(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	concept store.KnowledgeConcept,
	nextPath string,
	write store.KnowledgeConceptWrite,
	sources []SourceInput,
	patch Patch,
) (ConceptView, error) {
	links := deriveLinks(write.Body, nil)
	if patch.Links != nil {
		links = *patch.Links
	}

	decisionIDs := []int64(nil)
	if patch.DecisionIDs != nil {
		validated, err := s.validateDecisionIDs(ctx, actor, ref, *patch.DecisionIDs)
		if err != nil {
			return ConceptView{}, err
		}
		decisionIDs = validated
	} else {
		existing, err := s.store.GetKnowledgeDecisionIDs(ctx, concept.ID)
		if err != nil {
			return ConceptView{}, err
		}
		decisionIDs = existing
	}
	created, err := s.Create(ctx, actor, CreateInput{
		Scope:                ref,
		Path:                 nextPath,
		Type:                 write.Type,
		Title:                write.Title,
		Description:          write.Description,
		Resource:             write.Resource,
		Body:                 write.Body,
		Status:               write.Status,
		Tags:                 tagsFromJSON(write.TagsJson),
		StaleAfter:           timeFromNull(write.StaleAfter),
		ExtraFrontmatterJSON: extraFromFrontmatter(write.FrontmatterJson),
		Sources:              sources,
		Links:                links,
		DecisionIDs:          decisionIDs,
		Summary:              patch.Summary,
	})
	if err != nil {
		return ConceptView{}, err
	}
	if err := s.store.SoftDeleteKnowledgeConcept(
		ctx,
		concept.ID,
		write.GeneratedAt,
	); err != nil {

		_ = s.store.SoftDeleteKnowledgeConcept(ctx, created.Concept.ID, write.GeneratedAt)
		return ConceptView{}, err
	}
	s.signal()
	return created, nil
}

func (s *Service) validateRename(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	concept store.KnowledgeConcept,
	nextPath string,
) error {
	if err := ValidatePath(nextPath); err != nil {
		return err
	}

	rows, err := s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
		Scopes:         []string{ref.Scope},
		ScopeID:        ref.ScopeID,
		PathPrefix:     nextPath,
		IncludeDeleted: true,
		Limit:          5,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Path == nextPath && row.ID != concept.ID {
			return ErrConflict
		}
	}
	return nil
}

func (s *Service) Delete(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path string,
	baseRevision int64,
) error {
	ref = normalizeRef(actor, ref)
	if err := s.requireWrite(ctx, actor, ref); err != nil {
		return err
	}
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if concept.Protected != 0 && !actor.IsHuman() {
		return ErrProtected
	}
	if baseRevision > 0 && baseRevision != concept.Revision {
		return ErrConflict
	}
	if err := s.store.SoftDeleteKnowledgeConcept(
		ctx,
		concept.ID,
		s.now().UTC().UnixMilli(),
	); err != nil {
		return err
	}
	s.signal()
	return nil
}

func (s *Service) Revisions(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path string,
) ([]store.KnowledgeRevision, error) {
	ref = normalizeRef(actor, ref)
	concept, err := s.Get(ctx, actor, ref, path)
	if err != nil {
		return nil, err
	}
	return s.store.ListKnowledgeRevisions(ctx, concept.Concept.ID)
}

func (s *Service) Verify(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path, note string,
) (ConceptView, error) {
	ref = normalizeRef(actor, ref)
	if !actor.IsHuman() {
		return ConceptView{}, ErrVerifyNotAllowed
	}

	if err := s.requireWrite(ctx, actor, ref); err != nil {
		return ConceptView{}, err
	}
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	if errors.Is(err, sql.ErrNoRows) {
		return ConceptView{}, ErrNotFound
	}
	if err != nil {
		return ConceptView{}, err
	}
	now := s.now().UTC().UnixMilli()
	if _, err := s.store.CreateKnowledgeVerification(ctx, store.KnowledgeVerification{
		ConceptID: concept.ID,
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Note:      strings.TrimSpace(note),
		CreatedAt: now,
	}); err != nil {
		return ConceptView{}, err
	}
	return s.refreshTrust(ctx, actor, ref, concept, now)
}

func (s *Service) refreshTrust(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	concept store.KnowledgeConcept,
	now int64,
) (ConceptView, error) {
	verifications, err := s.store.ListKnowledgeVerifications(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	tier := deriveTrust(verifications)
	if err := s.store.SetKnowledgeConceptTrust(ctx, concept.ID, tier, now); err != nil {
		return ConceptView{}, err
	}
	concept.TrustTier = tier
	concept.UpdatedAt = now
	sources, _ := s.store.ListKnowledgeSources(ctx, concept.ID)
	write := conceptWriteFrom(concept)
	write.FrontmatterJson = renderFrontmatter(
		write,
		ref,
		concept.Path,
		extraFromFrontmatter(concept.FrontmatterJson),
		verifications,
		sourcesToInput(concept.ID, sources),
		nil,
	)
	if err := s.store.SetKnowledgeConceptFrontmatter(
		ctx,
		concept.ID,
		write.FrontmatterJson,
		now,
	); err != nil {
		return ConceptView{}, err
	}
	concept.FrontmatterJson = write.FrontmatterJson
	s.signal()
	return s.view(ctx, concept, actor)
}

func (s *Service) RevokeVerification(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path string,
	verificationID int64,
) (ConceptView, error) {
	ref = normalizeRef(actor, ref)
	if !actor.IsHuman() {
		return ConceptView{}, ErrVerifyNotAllowed
	}
	if err := s.requireRead(ctx, actor, ref); err != nil {
		return ConceptView{}, err
	}
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	if errors.Is(err, sql.ErrNoRows) {
		return ConceptView{}, ErrNotFound
	}
	if err != nil {
		return ConceptView{}, err
	}
	verifications, err := s.store.ListKnowledgeVerifications(ctx, concept.ID)
	if err != nil {
		return ConceptView{}, err
	}
	var target *store.KnowledgeVerification
	for i := range verifications {
		if verifications[i].ID == verificationID {
			target = &verifications[i]
			break
		}
	}
	if target == nil {
		return ConceptView{}, ErrNotFound
	}

	if !actor.Admin && target.ActorID.Int64 != int64(actor.ID) {
		return ConceptView{}, ErrForbidden
	}
	now := s.now().UTC().UnixMilli()
	if err := s.store.RevokeKnowledgeVerification(
		ctx,
		concept.ID,
		verificationID,
		actor.ID,
		now,
	); err != nil {
		return ConceptView{}, err
	}
	return s.refreshTrust(ctx, actor, ref, concept, now)
}

func (s *Service) Search(
	ctx context.Context,
	actor Actor,
	query string,
	scope *ScopeRef,
	limit int,
) ([]store.KnowledgeConcept, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []store.KnowledgeConcept{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	refs := []ScopeRef{}
	if scope != nil {
		refs = append(refs, normalizeRef(actor, *scope))
	} else {
		refs = append(refs, ScopeRef{Scope: ScopeOrg})
		if actor.ID != 0 && s.projects != nil {
			var list []projects.Project
			var err error
			if actor.IsHuman() && actor.Admin {
				list, err = s.projects.List(ctx)
			} else {
				list, err = s.projects.ProjectsForActor(ctx, actor.ID)
			}
			if err != nil {
				return nil, err
			}
			for _, p := range list {
				refs = append(refs, ScopeRef{Scope: ScopeProject, ScopeID: p.ID})
			}
		}
	}
	out := make([]store.KnowledgeConcept, 0, limit)
	seen := map[string]struct{}{}
	for _, ref := range refs {
		ok, err := s.CanRead(ctx, actor, ref)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		rows, err := s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
			Scopes:  []string{ref.Scope},
			ScopeID: ref.ScopeID,
			Query:   query,
			Limit:   limit,
		})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := seen[row.ID]; ok {
				continue
			}
			seen[row.ID] = struct{}{}
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := scopeRank(out[i].Scope), scopeRank(out[j].Scope)
		if a != b {
			return a < b
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func scopeRank(scope string) int {
	switch scope {
	case ScopeOrg:
		return 0
	case ScopeProject:
		return 1
	default:
		return 2
	}
}

func defaultType(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "Note"
	}
	return kind
}

func nullTime(t *time.Time) sql.NullInt64 {
	if t == nil || t.IsZero() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.UTC().UnixMilli(), Valid: true}
}

func timeFromNull(n sql.NullInt64) *time.Time {
	if !n.Valid {
		return nil
	}
	t := time.UnixMilli(n.Int64).UTC()
	return &t
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
	s.realtime.Publish(agencyv1.RealtimeKind_REALTIME_KIND_KNOWLEDGE)
}

func sourceRows(conceptID string, in []SourceInput) []store.KnowledgeSource {
	out := make([]store.KnowledgeSource, 0, len(in))
	for _, source := range in {
		var usage, modified sql.NullInt64
		if source.UsageCount != nil {
			usage = sql.NullInt64{Int64: *source.UsageCount, Valid: true}
		}
		if source.LastModified != nil {
			modified = sql.NullInt64{Int64: *source.LastModified, Valid: true}
		}
		out = append(out, store.KnowledgeSource{
			ConceptID:    conceptID,
			SourceID:     source.ID,
			Resource:     source.Resource,
			Title:        source.Title,
			AuthorRef:    source.AuthorRef,
			UsageCount:   usage,
			LastModified: modified,
		})
	}
	return out
}

func sourcesToInput(conceptID string, sources []store.KnowledgeSource) []SourceInput {
	out := make([]SourceInput, 0, len(sources))
	for _, source := range sources {
		in := SourceInput{
			ID:        source.SourceID,
			Resource:  source.Resource,
			Title:     source.Title,
			AuthorRef: source.AuthorRef,
		}
		if source.UsageCount.Valid {
			v := source.UsageCount.Int64
			in.UsageCount = &v
		}
		if source.LastModified.Valid {
			v := source.LastModified.Int64
			in.LastModified = &v
		}
		out = append(out, in)
	}
	return out
}

func mustSources(ctx context.Context, st store.Store, conceptID string) []store.KnowledgeSource {
	sources, err := st.ListKnowledgeSources(ctx, conceptID)
	if err != nil {
		return nil
	}
	return sources
}

func linkRows(conceptID string, in []LinkInput) []store.KnowledgeLink {
	out := make([]store.KnowledgeLink, 0, len(in))
	for _, link := range in {
		kind := link.Kind
		if kind == "" {
			kind = "link"
		}
		out = append(out, store.KnowledgeLink{
			ConceptID:  conceptID,
			TargetPath: link.TargetPath,
			Kind:       kind,
		})
	}
	return out
}

func deriveLinks(body string, explicit []LinkInput) []LinkInput {
	out := make([]LinkInput, 0, len(explicit))
	seen := map[string]struct{}{}
	for _, link := range explicit {
		target := strings.TrimSpace(strings.TrimSuffix(link.TargetPath, ".md"))
		if target == "" {
			continue
		}
		kind := link.Kind
		if kind == "" {
			kind = "link"
		}

		key := target + "|" + kind
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		link.Kind = kind
		out = append(out, link)
	}
	for _, match := range wikiLinkPattern.FindAllStringSubmatch(body, -1) {
		target := strings.TrimSpace(strings.TrimSuffix(match[1], ".md"))
		if target == "" {
			continue
		}
		key := target + "|link"
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, LinkInput{TargetPath: target, Kind: "link"})
	}
	return out
}

func deriveTrust(verifications []store.KnowledgeVerification) string {
	human := false
	machine := false
	for _, v := range verifications {
		if v.RevokedAt.Valid {
			continue
		}
		if strings.HasPrefix(v.ActorRef, "human:") {
			human = true
		} else {
			machine = true
		}
	}
	switch {
	case human:
		return TrustHumanReviewed
	case machine:
		return TrustMachineConfirmed
	default:
		return TrustUnverified
	}
}

func extraFromFrontmatter(raw string) string {
	var doc map[string]any
	if json.Unmarshal([]byte(raw), &doc) != nil {
		return ""
	}
	for _, key := range []string{"type", "title", "description", "tags", "generated", "verified", "status", "stale_after", "sources", "resource", "x-cherrypi"} {
		delete(doc, key)
	}
	if len(doc) == 0 {
		return ""
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return ""
	}
	return string(out)
}

func renderFrontmatter(
	write store.KnowledgeConceptWrite,
	ref ScopeRef,
	path, extraJSON string,
	verifications []store.KnowledgeVerification,
	sources []SourceInput,
	_ []LinkInput,
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
	verified := make([]map[string]any, 0, len(verifications))
	for _, v := range verifications {
		if v.RevokedAt.Valid {
			continue
		}
		entry := map[string]any{
			"by": v.ActorRef,
			"at": time.UnixMilli(v.CreatedAt).UTC().Format(time.RFC3339),
		}
		if v.Note != "" {
			entry["note"] = v.Note
		}
		verified = append(verified, entry)
	}
	doc["verified"] = verified
	outSources := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		entry := map[string]any{
			"id":       source.ID,
			"resource": source.Resource,
		}
		if source.Title != "" {
			entry["title"] = source.Title
		}
		if source.AuthorRef != "" {
			entry["author"] = source.AuthorRef
		}
		outSources = append(outSources, entry)
	}
	doc["sources"] = outSources
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
