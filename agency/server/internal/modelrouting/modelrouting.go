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

package modelrouting

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	orgKeyPrefix     = "model_routing"
	projectKeyPrefix = "model_routing:project:"
)

var thinkingLevels = map[string]bool{
	"off": true, "minimal": true, "low": true,
	"medium": true, "high": true, "xhigh": true, "max": true,
}

var sizes = map[string]bool{
	"small": true, "medium": true, "big": true, "epic": true,
}

var (
	ErrInvalidSelection = errors.New(
		"model provider and id must be set together and the thinking level must be known",
	)

	ErrInvalidRoute = errors.New("a model route needs at least one condition and valid values")

	ErrInvalidTagPin = errors.New("a tag pin needs a tag")
)

type Selection struct {
	Provider      string `json:"provider,omitempty"`
	ModelID       string `json:"model_id,omitempty"`
	ModelName     string `json:"model_name,omitempty"`
	ThinkingLevel string `json:"thinking_level,omitempty"`
}

func (s Selection) IsZero() bool {
	return s.Provider == "" && s.ModelID == ""
}

type Route struct {
	Priority int64 `json:"priority,omitempty"`

	Size string `json:"size,omitempty"`

	Tag string `json:"tag,omitempty"`

	Model Selection `json:"model"`
}

type OrganizationModelRouting struct {
	Default Selection `json:"default,omitempty"`
	Routes  []Route   `json:"routes,omitempty"`
}

type ProjectModelRouting struct {
	Default *Selection `json:"default,omitempty"`
	TagPins []Route    `json:"tag_pins,omitempty"`
	Routes  []Route    `json:"routes,omitempty"`
}

type ResolveInput struct {
	ProjectID string
	Priority  int64
	Size      string
	Tags      []string
}

type Match struct {
	Selection Selection
	Scope     string
	Index     int
}

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

func orgKey() string { return orgKeyPrefix }

func projectKey(projectID string) string {
	return projectKeyPrefix + projectID
}

func (s *Service) GetOrganization(ctx context.Context) (OrganizationModelRouting, error) {
	var out OrganizationModelRouting
	err := s.read(ctx, orgKey(), &out)
	if err != nil {
		return OrganizationModelRouting{}, err
	}
	return NormalizeOrganization(out), nil
}

func (s *Service) UpdateOrganization(
	ctx context.Context,
	in OrganizationModelRouting,
) (OrganizationModelRouting, error) {
	next, err := ValidateOrganization(in)
	if err != nil {
		return OrganizationModelRouting{}, err
	}
	if err := s.write(ctx, orgKey(), next); err != nil {
		return OrganizationModelRouting{}, err
	}
	return next, nil
}

func (s *Service) GetProject(ctx context.Context, projectID string) (ProjectModelRouting, error) {
	var out ProjectModelRouting
	if strings.TrimSpace(projectID) == "" {
		return ProjectModelRouting{}, nil
	}
	err := s.read(ctx, projectKey(projectID), &out)
	if err != nil {
		return ProjectModelRouting{}, err
	}
	return NormalizeProject(out), nil
}

func (s *Service) UpdateProject(
	ctx context.Context,
	projectID string,
	in ProjectModelRouting,
) (ProjectModelRouting, error) {
	if strings.TrimSpace(projectID) == "" {
		return ProjectModelRouting{}, errors.New("project id is required")
	}
	next, err := ValidateProject(in)
	if err != nil {
		return ProjectModelRouting{}, err
	}
	if err := s.write(ctx, projectKey(projectID), next); err != nil {
		return ProjectModelRouting{}, err
	}
	return next, nil
}

func (s *Service) ForgetProject(ctx context.Context, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return nil
	}
	err := s.store.DeleteSetting(ctx, projectKey(projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (s *Service) Resolve(
	ctx context.Context,
	in ResolveInput,
) (Selection, bool, error) {
	match, ok, err := s.ResolveMatch(ctx, in)
	return match.Selection, ok, err
}

func (s *Service) ResolveMatch(
	ctx context.Context,
	in ResolveInput,
) (Match, bool, error) {
	org, err := s.GetOrganization(ctx)
	if err != nil {
		return Match{}, false, err
	}
	var project ProjectModelRouting
	if in.ProjectID != "" {
		project, err = s.GetProject(ctx, in.ProjectID)
		if err != nil {
			return Match{}, false, err
		}
	}

	for i := range project.TagPins {
		if matches(project.TagPins[i], in) {
			return Match{
				Selection: project.TagPins[i].Model,
				Scope:     "project_tag_pin",
				Index:     i + 1,
			}, true, nil
		}
	}
	for i := range project.Routes {
		if matches(project.Routes[i], in) {
			return Match{
				Selection: project.Routes[i].Model,
				Scope:     "project_rule",
				Index:     i + 1,
			}, true, nil
		}
	}
	if project.Default != nil && !project.Default.IsZero() {
		return Match{Selection: *project.Default, Scope: "project_default"}, true, nil
	}
	for i := range org.Routes {
		if matches(org.Routes[i], in) {
			return Match{
				Selection: org.Routes[i].Model,
				Scope:     "organization_rule",
				Index:     i + 1,
			}, true, nil
		}
	}
	if !org.Default.IsZero() {
		return Match{Selection: org.Default, Scope: "organization_default"}, true, nil
	}
	return Match{}, false, nil
}

func matches(route Route, in ResolveInput) bool {
	if route.Priority != 0 && route.Priority != in.Priority {
		return false
	}
	if route.Size != "" && route.Size != in.Size {
		return false
	}
	if route.Tag != "" {
		matched := false
		for _, tag := range in.Tags {
			if strings.EqualFold(strings.TrimSpace(tag), route.Tag) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return route.Priority != 0 || route.Size != "" || route.Tag != ""
}

func (s *Service) read(ctx context.Context, key string, out any) error {
	raw, err := s.store.GetSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {

		log.Printf("[modelrouting] ignoring invalid %s: %v", key, err)
		return nil
	}
	return nil
}

func (s *Service) write(ctx context.Context, key string, in any) error {
	value, err := marshal(in)
	if err != nil {
		return err
	}
	if value == "" {
		err := s.store.DeleteSetting(ctx, key)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   key,
		Value: value,
	})
}

func marshal(in any) (string, error) {
	switch value := in.(type) {
	case OrganizationModelRouting:
		if value.Default.IsZero() && len(value.Routes) == 0 {
			return "", nil
		}
	case ProjectModelRouting:
		if value.Default == nil && len(value.TagPins) == 0 && len(value.Routes) == 0 {
			return "", nil
		}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func NormalizeOrganization(in OrganizationModelRouting) OrganizationModelRouting {
	out := OrganizationModelRouting{
		Default: normalizeSelection(in.Default),
		Routes:  normalizeRoutes(in.Routes),
	}
	return out
}

func NormalizeProject(in ProjectModelRouting) ProjectModelRouting {
	out := ProjectModelRouting{
		TagPins: normalizeRoutes(in.TagPins),
		Routes:  normalizeRoutes(in.Routes),
	}
	if in.Default != nil {
		selection := normalizeSelection(*in.Default)
		if !selection.IsZero() {
			out.Default = &selection
		}
	}
	return out
}

func ValidateOrganization(in OrganizationModelRouting) (OrganizationModelRouting, error) {
	out := OrganizationModelRouting{
		Default: normalizeSelection(in.Default),
	}
	if !out.Default.IsZero() {
		if err := validateSelection(out.Default); err != nil {
			return OrganizationModelRouting{}, err
		}
	}
	routes, err := validateRoutes(in.Routes, false)
	if err != nil {
		return OrganizationModelRouting{}, err
	}
	out.Routes = routes
	return out, nil
}

func ValidateProject(in ProjectModelRouting) (ProjectModelRouting, error) {
	out := ProjectModelRouting{}
	if in.Default != nil {
		selection := normalizeSelection(*in.Default)
		if !selection.IsZero() {
			if err := validateSelection(selection); err != nil {
				return ProjectModelRouting{}, err
			}
			out.Default = &selection
		}
	}
	pins, err := validateRoutes(in.TagPins, true)
	if err != nil {
		return ProjectModelRouting{}, err
	}
	out.TagPins = pins
	routes, err := validateRoutes(in.Routes, false)
	if err != nil {
		return ProjectModelRouting{}, err
	}
	out.Routes = routes
	return out, nil
}

func validateRoutes(in []Route, tagPin bool) ([]Route, error) {
	out := make([]Route, 0, len(in))
	for _, route := range in {
		next := Route{
			Priority: route.Priority,
			Size:     strings.ToLower(strings.TrimSpace(route.Size)),
			Tag:      strings.ToLower(strings.TrimSpace(route.Tag)),
			Model:    normalizeSelection(route.Model),
		}
		if next.Priority < 0 || next.Priority > 4 {
			return nil, fmt.Errorf(
				"%w: priority %d is out of range",
				ErrInvalidRoute,
				next.Priority,
			)
		}
		if next.Size != "" && !sizes[next.Size] {
			return nil, fmt.Errorf("%w: unknown size %q", ErrInvalidRoute, next.Size)
		}
		if tagPin {
			if next.Tag == "" {
				return nil, ErrInvalidTagPin
			}
			if next.Priority != 0 || next.Size != "" {
				return nil, fmt.Errorf("%w: a tag pin matches a tag only", ErrInvalidRoute)
			}
		}
		if next.Priority == 0 && next.Size == "" && next.Tag == "" {
			return nil, ErrInvalidRoute
		}
		if err := validateSelection(next.Model); err != nil {
			return nil, err
		}
		out = append(out, next)
	}
	return out, nil
}

func normalizeSelection(in Selection) Selection {
	return Selection{
		Provider:      strings.TrimSpace(in.Provider),
		ModelID:       strings.TrimSpace(in.ModelID),
		ModelName:     strings.TrimSpace(in.ModelName),
		ThinkingLevel: strings.ToLower(strings.TrimSpace(in.ThinkingLevel)),
	}
}

func validateSelection(in Selection) error {
	if (in.Provider == "") != (in.ModelID == "") {
		return ErrInvalidSelection
	}
	if in.Provider == "" {
		return fmt.Errorf("%w: a route needs a model", ErrInvalidSelection)
	}
	if in.ThinkingLevel != "" && !thinkingLevels[in.ThinkingLevel] {
		return fmt.Errorf("%w: unknown thinking level %q", ErrInvalidSelection, in.ThinkingLevel)
	}
	return nil
}

func normalizeRoutes(in []Route) []Route {
	out := make([]Route, 0, len(in))
	for _, route := range in {
		next := Route{
			Priority: route.Priority,
			Size:     strings.ToLower(strings.TrimSpace(route.Size)),
			Tag:      strings.ToLower(strings.TrimSpace(route.Tag)),
			Model:    normalizeSelection(route.Model),
		}
		if next.Model.IsZero() {
			continue
		}
		out = append(out, next)
	}
	return out
}

func PriorityName(priority int64) string {
	switch priority {
	case 1:
		return "low"
	case 2:
		return "normal"
	case 3:
		return "high"
	case 4:
		return "critical"
	default:
		return strconv.FormatInt(priority, 10)
	}
}
