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

package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

var (
	ErrTemplateMissing = errors.New("task template not found")
	ErrVersionMissing  = errors.New("task template version not found")
	ErrNameRequired    = errors.New("template name must be non-empty")
	ErrNameTaken       = errors.New("a template with that name already exists in the project")
	ErrBodyRequired    = errors.New("template body must be non-empty")
	ErrArchived        = errors.New("task template is archived")
	ErrProjectMismatch = errors.New("task template belongs to another project")
	ErrInvalidContent  = errors.New("invalid task template")
	ErrVersionConflict = errors.New("task template version conflict; retry")
)

type ArgType string

const (
	ArgText      ArgType = "text"
	ArgMultiline ArgType = "multiline"
	ArgNumber    ArgType = "number"
	ArgBoolean   ArgType = "boolean"
	ArgSelect    ArgType = "select"
)

type ArgOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Arg struct {
	Key          string      `json:"key"`
	Label        string      `json:"label"`
	Description  string      `json:"description"`
	Type         ArgType     `json:"type"`
	Required     bool        `json:"required"`
	DefaultValue string      `json:"default_value"`
	Options      []ArgOption `json:"options,omitempty"`
}

type Content struct {
	Body                string
	TitleTemplate       string
	Args                []Arg
	DefaultPriority     int64
	DefaultSize         string
	DefaultTagIDs       []string
	DefaultTeamIDs      []int64
	DefaultLeadActorIDs []ids.ActorID
}

type Version struct {
	TemplateID          string
	Version             int64
	Body                string
	TitleTemplate       string
	Args                []Arg
	DefaultPriority     int64
	DefaultSize         string
	DefaultTagIDs       []string
	DefaultTeamIDs      []int64
	DefaultLeadActorIDs []ids.ActorID
	CreatedByActorID    ids.ActorID
	CreatedAt           time.Time
}

type Template struct {
	ID               string
	ProjectID        string
	Name             string
	Description      string
	CreatedByActorID ids.ActorID
	ArchivedAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LatestVersion    int64
	Latest           *Version
}

type Validation struct {
	Errors              []string
	Warnings            []string
	UndeclaredVariables []string
	UnusedArguments     []string
}

func (v Validation) Valid() bool { return len(v.Errors) == 0 }

type Rendered struct {
	TemplateID   string
	TemplateName string
	Version      int64
	Prompt       string
	Title        string
	Priority     int64
	Size         string
	TagIDs       []string
	TeamIDs      []int64
	LeadActorIDs []ids.ActorID
	ArgsJSON     string
}

type Service struct {
	store    store.Store
	projects *projects.Service
}

func New(st store.Store, projectService *projects.Service) *Service {
	return &Service{store: st, projects: projectService}
}

func (s *Service) requireRole(ctx context.Context, projectID, min string) error {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil
	}
	_, err := s.projects.RequireRole(
		ctx, projectID, identity.ActorID, identity.IsAdmin(), min,
	)
	return err
}

func actingActor(ctx context.Context) ids.ActorID {
	if id, ok := auth.ActorID(ctx); ok {
		return id
	}
	return 0
}

func (s *Service) List(
	ctx context.Context,
	projectID string,
	includeArchived bool,
) ([]Template, error) {
	if err := s.requireRole(ctx, projectID, projects.RoleViewer); err != nil {
		return nil, err
	}
	rows, err := s.store.ListTaskTemplates(ctx, projectID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]Template, 0, len(rows))
	for _, row := range rows {
		t := templateFromSummary(row)
		if err := s.attachLatest(ctx, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id string) (Template, error) {
	row, err := s.store.GetTaskTemplate(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrTemplateMissing
	}
	if err != nil {
		return Template{}, err
	}
	t := templateFromSummary(row)
	if err := s.requireRole(ctx, t.ProjectID, projects.RoleViewer); err != nil {
		return Template{}, err
	}
	if err := s.attachLatest(ctx, &t); err != nil {
		return Template{}, err
	}
	return t, nil
}

func (s *Service) Create(
	ctx context.Context,
	projectID, name, description string,
	content Content,
) (Template, Validation, error) {
	if err := s.requireRole(ctx, projectID, projects.RoleMember); err != nil {
		return Template{}, Validation{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Template{}, Validation{}, ErrNameRequired
	}
	if err := s.ensureNameFree(ctx, projectID, name, ""); err != nil {
		return Template{}, Validation{}, err
	}
	normalized, validation, err := s.prepareContent(ctx, projectID, content)
	if err != nil {
		return Template{}, validation, err
	}
	if !validation.Valid() {
		return Template{}, validation, fmt.Errorf(
			"%w: %s",
			ErrInvalidContent,
			strings.Join(validation.Errors, "; "),
		)
	}

	now := time.Now().UnixMilli()
	row, err := s.store.CreateTaskTemplate(ctx, store.CreateTaskTemplateParams{
		ID:               uuid.NewString(),
		ProjectID:        projectID,
		Name:             name,
		Description:      strings.TrimSpace(description),
		CreatedByActorID: nullActor(actingActor(ctx)),
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return Template{}, validation, err
	}
	version, err := s.appendVersion(ctx, row.ID, normalized, now)
	if err != nil {
		return Template{}, validation, err
	}
	s.log(ctx, projectID, "created", "", "", name,
		"Created task template \""+name+"\"", map[string]any{
			"template_id": row.ID,
			"name":        name,
			"version":     version.Version,
		})
	t, err := s.Get(ctx, row.ID)
	if err != nil {
		return Template{}, validation, err
	}
	return t, validation, nil
}

func (s *Service) Update(
	ctx context.Context,
	id, name, description string,
	content *Content,
) (Template, bool, Validation, error) {
	existing, err := s.getRow(ctx, id)
	if err != nil {
		return Template{}, false, Validation{}, err
	}
	if err := s.requireRole(ctx, existing.ProjectID, projects.RoleMember); err != nil {
		return Template{}, false, Validation{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Template{}, false, Validation{}, ErrNameRequired
	}
	if err := s.ensureNameFree(ctx, existing.ProjectID, name, existing.ID); err != nil {
		return Template{}, false, Validation{}, err
	}
	var (
		normalized Content
		validation Validation
	)
	if content != nil {
		normalized, validation, err = s.prepareContent(ctx, existing.ProjectID, *content)
		if err != nil {
			return Template{}, false, validation, err
		}
		if !validation.Valid() {
			return Template{}, false, validation, fmt.Errorf(
				"%w: %s",
				ErrInvalidContent,
				strings.Join(validation.Errors, "; "),
			)
		}
	}
	now := time.Now().UnixMilli()
	if _, err := s.store.UpdateTaskTemplate(ctx, store.UpdateTaskTemplateParams{
		Name:        name,
		Description: strings.TrimSpace(description),
		UpdatedAt:   now,
		ID:          existing.ID,
	}); err != nil {
		return Template{}, false, validation, err
	}
	versionCreated := false
	if content != nil {
		latest, latestErr := s.latestVersion(ctx, existing.ID)
		if latestErr != nil && !errors.Is(latestErr, ErrVersionMissing) {
			return Template{}, false, validation, latestErr
		}
		if latestErr != nil || !sameContent(contentFromVersion(*latest), normalized) {
			if _, err := s.appendVersion(ctx, existing.ID, normalized, now); err != nil {
				return Template{}, false, validation, err
			}
			versionCreated = true
		}
	}
	if versionCreated {
		s.log(ctx, existing.ProjectID, "template", "updated", existing.Name, name,
			fmt.Sprintf("Updated task template \"%s\"", name), map[string]any{
				"template_id": existing.ID,
				"name":        name,
			})
	} else if existing.Name != name || existing.Description != strings.TrimSpace(description) {
		s.log(ctx, existing.ProjectID, "template", "updated", existing.Name, name,
			fmt.Sprintf("Updated task template \"%s\"", name), map[string]any{
				"template_id": existing.ID,
				"name":        name,
			})
	}
	t, err := s.Get(ctx, existing.ID)
	if err != nil {
		return Template{}, versionCreated, validation, err
	}
	return t, versionCreated, validation, nil
}

func (s *Service) SetArchived(ctx context.Context, id string, archived bool) (Template, error) {
	existing, err := s.getRow(ctx, id)
	if err != nil {
		return Template{}, err
	}
	if err := s.requireRole(ctx, existing.ProjectID, projects.RoleMember); err != nil {
		return Template{}, err
	}
	now := time.Now().UnixMilli()
	archivedAt := sql.NullInt64{}
	if archived {
		archivedAt = sql.NullInt64{Int64: now, Valid: true}
	}
	if _, err := s.store.SetTaskTemplateArchived(ctx, store.SetTaskTemplateArchivedParams{
		ArchivedAt: archivedAt,
		UpdatedAt:  now,
		ID:         existing.ID,
	}); err != nil {
		return Template{}, err
	}
	kind, field, message := "template", "archived", "Archived task template \""+existing.Name+"\""
	if !archived {
		field, message = "restored", "Restored task template \""+existing.Name+"\""
	}
	s.log(ctx, existing.ProjectID, kind, field, "", existing.Name, message, map[string]any{
		"template_id": existing.ID,
		"name":        existing.Name,
	})
	return s.Get(ctx, existing.ID)
}

func (s *Service) Versions(ctx context.Context, templateID string) ([]Version, error) {
	t, err := s.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListTaskTemplateVersions(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	out := make([]Version, len(rows))
	for i, row := range rows {
		out[i] = versionFromRow(row)
	}
	return out, nil
}

func (s *Service) RestoreVersion(
	ctx context.Context,
	templateID string,
	version int64,
) (Template, int64, error) {
	t, err := s.Get(ctx, templateID)
	if err != nil {
		return Template{}, 0, err
	}
	if err := s.requireRole(ctx, t.ProjectID, projects.RoleMember); err != nil {
		return Template{}, 0, err
	}
	old, err := s.version(ctx, t.ID, version)
	if err != nil {
		return Template{}, 0, err
	}
	now := time.Now().UnixMilli()
	content := contentFromVersion(old)
	created, err := s.appendVersion(ctx, t.ID, content, now)
	if err != nil {
		return Template{}, 0, err
	}
	if _, err := s.store.UpdateTaskTemplate(ctx, store.UpdateTaskTemplateParams{
		Name:        t.Name,
		Description: t.Description,
		UpdatedAt:   now,
		ID:          t.ID,
	}); err != nil {
		return Template{}, 0, err
	}
	s.log(
		ctx,
		t.ProjectID,
		"template",
		"version",
		fmt.Sprint(version),
		fmt.Sprint(created.Version),
		fmt.Sprintf(
			"Restored version %d of task template \"%s\" as version %d",
			version,
			t.Name,
			created.Version,
		),
		map[string]any{
			"template_id": t.ID,
			"name":        t.Name,
			"from":        version,
			"version":     created.Version,
		},
	)
	t, err = s.Get(ctx, t.ID)
	return t, created.Version, err
}

func (s *Service) Render(
	ctx context.Context,
	templateID string,
	version int64,
	argsJSON string,
) (Rendered, error) {
	t, err := s.Get(ctx, templateID)
	if err != nil {
		return Rendered{}, err
	}
	if !t.ArchivedAt.IsZero() {
		return Rendered{}, ErrArchived
	}
	v := t.Latest
	if version != 0 {
		selected, versionErr := s.version(ctx, t.ID, version)
		if versionErr != nil {
			return Rendered{}, versionErr
		}
		v = &selected
	}
	if v == nil {
		return Rendered{}, ErrVersionMissing
	}
	return renderVersion(t, *v, argsJSON)
}

func (s *Service) RenderForProject(
	ctx context.Context,
	projectID, templateID string,
	version int64,
	argsJSON string,
) (Rendered, error) {
	t, err := s.Get(ctx, templateID)
	if err != nil {
		return Rendered{}, err
	}
	if t.ProjectID != projectID {
		return Rendered{}, ErrProjectMismatch
	}
	if !t.ArchivedAt.IsZero() {
		return Rendered{}, ErrArchived
	}
	v := t.Latest
	if version != 0 {
		selected, versionErr := s.version(ctx, t.ID, version)
		if versionErr != nil {
			return Rendered{}, versionErr
		}
		v = &selected
	}
	if v == nil {
		return Rendered{}, ErrVersionMissing
	}
	return renderVersion(t, *v, argsJSON)
}

func (s *Service) RenderContent(
	ctx context.Context,
	projectID string,
	content Content,
	argsJSON string,
) (Rendered, error) {
	normalized, validation, err := s.prepareContent(ctx, projectID, content)
	if err != nil {
		return Rendered{}, err
	}
	if !validation.Valid() {
		return Rendered{}, fmt.Errorf(
			"%w: %s",
			ErrInvalidContent,
			strings.Join(validation.Errors, "; "),
		)
	}
	return renderVersion(Template{}, Version{
		Body:                normalized.Body,
		TitleTemplate:       normalized.TitleTemplate,
		Args:                normalized.Args,
		DefaultPriority:     normalized.DefaultPriority,
		DefaultSize:         normalized.DefaultSize,
		DefaultTagIDs:       normalized.DefaultTagIDs,
		DefaultTeamIDs:      normalized.DefaultTeamIDs,
		DefaultLeadActorIDs: normalized.DefaultLeadActorIDs,
	}, argsJSON)
}

func (s *Service) Validate(
	ctx context.Context,
	projectID string,
	content Content,
) (Validation, error) {
	if err := s.requireRole(ctx, projectID, projects.RoleViewer); err != nil {
		return Validation{}, err
	}
	_, validation, err := s.prepareContent(ctx, projectID, content)
	return validation, err
}

func (s *Service) prepareContent(
	ctx context.Context,
	projectID string,
	content Content,
) (Content, Validation, error) {
	normalized, validation := normalizeContent(content)
	if err := s.validateDefaults(ctx, projectID, &normalized, &validation); err != nil {
		return normalized, validation, err
	}
	renderErrors, warnings := renderSmokeCheck(normalized)
	validation.Errors = append(validation.Errors, renderErrors...)
	validation.Warnings = append(validation.Warnings, warnings...)
	return normalized, validation, nil
}

func (s *Service) ensureNameFree(
	ctx context.Context,
	projectID, name, exceptID string,
) error {
	row, err := s.store.GetTaskTemplateByProjectName(ctx, projectID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.ID != exceptID {
		return ErrNameTaken
	}
	return nil
}

func (s *Service) getRow(ctx context.Context, id string) (store.TaskTemplateSummary, error) {
	row, err := s.store.GetTaskTemplate(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TaskTemplateSummary{}, ErrTemplateMissing
	}
	return row, err
}

func (s *Service) attachLatest(ctx context.Context, t *Template) error {
	if t.LatestVersion == 0 {
		return nil
	}
	v, err := s.version(ctx, t.ID, t.LatestVersion)
	if err != nil {
		return err
	}
	t.Latest = &v
	return nil
}

func (s *Service) latestVersion(ctx context.Context, templateID string) (*Version, error) {
	row, err := s.store.GetTaskTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if row.LatestVersion == 0 {
		return nil, ErrVersionMissing
	}
	v, err := s.version(ctx, templateID, row.LatestVersion)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) version(
	ctx context.Context,
	templateID string,
	version int64,
) (Version, error) {
	row, err := s.store.GetTaskTemplateVersion(ctx, templateID, version)
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, ErrVersionMissing
	}
	if err != nil {
		return Version{}, err
	}
	return versionFromRow(row), nil
}

func (s *Service) appendVersion(
	ctx context.Context,
	templateID string,
	content Content,
	now int64,
) (Version, error) {
	argsJSON, err := json.Marshal(content.Args)
	if err != nil {
		return Version{}, err
	}
	tagsJSON, err := json.Marshal(content.DefaultTagIDs)
	if err != nil {
		return Version{}, err
	}
	teamsJSON, err := json.Marshal(content.DefaultTeamIDs)
	if err != nil {
		return Version{}, err
	}
	leadsJSON, err := json.Marshal(actorIDStrings(content.DefaultLeadActorIDs))
	if err != nil {
		return Version{}, err
	}

	row, err := s.store.CreateTaskTemplateVersion(ctx, store.CreateTaskTemplateVersionParams{
		TemplateID:              templateID,
		Body:                    content.Body,
		TitleTemplate:           content.TitleTemplate,
		ArgsJson:                string(argsJSON),
		DefaultPriority:         content.DefaultPriority,
		DefaultSize:             content.DefaultSize,
		DefaultTagIdsJson:       string(tagsJSON),
		DefaultTeamIdsJson:      string(teamsJSON),
		DefaultLeadActorIdsJson: string(leadsJSON),
		CreatedByActorID:        nullActor(actingActor(ctx)),
		CreatedAt:               now,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Version{}, ErrVersionConflict
		}
		return Version{}, err
	}
	return versionFromRow(row), nil
}

func (s *Service) validateDefaults(
	ctx context.Context,
	projectID string,
	content *Content,
	validation *Validation,
) error {

	if len(content.DefaultTagIDs) > 0 {
		rows, err := s.store.ListTags(ctx, projectID, true)
		if err != nil {
			return err
		}
		valid := make(map[string]bool, len(rows))
		for _, row := range rows {
			valid[row.ID] = true
		}
		for _, id := range content.DefaultTagIDs {
			if !valid[id] {
				validation.Errors = append(validation.Errors,
					fmt.Sprintf("default tag %q does not belong to this project", id))
			}
		}
	}
	if len(content.DefaultTeamIDs) > 0 {
		idsWithAccess, err := s.store.ListProjectTeamIDsWithAccess(ctx, projectID)
		if err != nil {
			return err
		}
		valid := make(map[int64]bool, len(idsWithAccess))
		for _, id := range idsWithAccess {
			valid[id] = true
		}
		for _, id := range content.DefaultTeamIDs {
			if !valid[id] {
				validation.Errors = append(validation.Errors,
					fmt.Sprintf("default team %d is not attached to this project", id))
			}
		}
	}
	if len(content.DefaultLeadActorIDs) > 0 {
		members, err := s.store.ListProjectMemberActorIDs(ctx, projectID)
		if err != nil {
			return err
		}
		valid := make(map[ids.ActorID]bool, len(members))
		for _, id := range members {
			valid[ids.ActorID(id)] = true
		}
		for _, id := range content.DefaultLeadActorIDs {
			if !valid[id] {
				validation.Errors = append(validation.Errors,
					fmt.Sprintf("default lead %d is not a project member", id))
			}
		}
	}
	return nil
}

func (s *Service) log(
	ctx context.Context,
	projectID, kind, field, oldValue, newValue, message string,
	data any,
) {
	if s.projects != nil {
		s.projects.WriteLog(ctx, projectID, kind, field, oldValue, newValue, message, data)
	}
}

func templateFromSummary(row store.TaskTemplateSummary) Template {
	return Template{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		Name:             row.Name,
		Description:      row.Description,
		CreatedByActorID: ids.ActorID(row.CreatedByActorID.Int64),
		ArchivedAt:       timeOrZero(row.ArchivedAt),
		CreatedAt:        time.UnixMilli(row.CreatedAt),
		UpdatedAt:        time.UnixMilli(row.UpdatedAt),
		LatestVersion:    row.LatestVersion,
	}
}

func versionFromRow(row store.TaskTemplateVersion) Version {
	return Version{
		TemplateID:          row.TemplateID,
		Version:             row.Version,
		Body:                row.Body,
		TitleTemplate:       row.TitleTemplate,
		Args:                decodeArgs(row.ArgsJson),
		DefaultPriority:     row.DefaultPriority,
		DefaultSize:         row.DefaultSize,
		DefaultTagIDs:       decodeStrings(row.DefaultTagIdsJson),
		DefaultTeamIDs:      decodeInt64s(row.DefaultTeamIdsJson),
		DefaultLeadActorIDs: decodeActorIDs(row.DefaultLeadActorIdsJson),
		CreatedByActorID:    ids.ActorID(row.CreatedByActorID.Int64),
		CreatedAt:           time.UnixMilli(row.CreatedAt),
	}
}

func contentFromVersion(v Version) Content {
	return Content{
		Body:                v.Body,
		TitleTemplate:       v.TitleTemplate,
		Args:                v.Args,
		DefaultPriority:     v.DefaultPriority,
		DefaultSize:         v.DefaultSize,
		DefaultTagIDs:       v.DefaultTagIDs,
		DefaultTeamIDs:      v.DefaultTeamIDs,
		DefaultLeadActorIDs: v.DefaultLeadActorIDs,
	}
}

func sameContent(a, b Content) bool {
	left, err := json.Marshal(normalizeComparable(a))
	if err != nil {
		return false
	}
	right, err := json.Marshal(normalizeComparable(b))
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

func normalizeComparable(c Content) Content {
	c.DefaultTagIDs = append([]string(nil), c.DefaultTagIDs...)
	sort.Strings(c.DefaultTagIDs)
	c.DefaultTeamIDs = append([]int64(nil), c.DefaultTeamIDs...)
	sort.Slice(
		c.DefaultTeamIDs,
		func(i, j int) bool { return c.DefaultTeamIDs[i] < c.DefaultTeamIDs[j] },
	)
	c.DefaultLeadActorIDs = append([]ids.ActorID(nil), c.DefaultLeadActorIDs...)
	sort.Slice(
		c.DefaultLeadActorIDs,
		func(i, j int) bool { return c.DefaultLeadActorIDs[i] < c.DefaultLeadActorIDs[j] },
	)
	return c
}

func decodeArgs(raw string) []Arg {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var args []Arg
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil
	}
	return args
}

func decodeStrings(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func decodeInt64s(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []int64
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func decodeActorIDs(raw string) []ids.ActorID {
	var out []ids.ActorID
	for _, value := range decodeStrings(raw) {
		var id int64
		if _, err := fmt.Sscan(value, &id); err == nil {
			out = append(out, ids.ActorID(id))
		}
	}
	return out
}

func actorIDStrings(in []ids.ActorID) []string {
	out := make([]string, len(in))
	for i, id := range in {
		out[i] = fmt.Sprint(id)
	}
	return out
}

func nullActor(id ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0}
}

func timeOrZero(value sql.NullInt64) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return time.UnixMilli(value.Int64)
}
