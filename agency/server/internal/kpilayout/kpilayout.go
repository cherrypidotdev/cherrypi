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

package kpilayout

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	KindAgency  = "agency"
	KindProject = "project"
	KindAgent   = "agent"
)

const (
	GridColumns = 4
	GridRows    = 2

	MaxWidgets = GridColumns * GridRows
)

var usageRanges = map[string]bool{"": true, "7": true, "30": true, "90": true, "all": true}

var (
	ErrInvalidPage = errors.New("unknown KPI page")

	ErrInvalidLayout = errors.New("invalid KPI layout")
)

type Page struct {
	Kind string
	ID   string
}

type Widget struct {
	ID string `json:"id"`

	Kind string `json:"kind"`

	Width  int `json:"width"`
	Height int `json:"height"`

	Range string `json:"range,omitempty"`

	Scope string `json:"scope,omitempty"`
}

type Placement struct {
	X int
	Y int
}

type Layout struct {
	Widgets []Widget `json:"widgets"`
}

func (l Layout) Empty() bool { return len(l.Widgets) == 0 }

type Service struct {
	store store.Store
}

func New(st store.Store) *Service { return &Service{store: st} }

func Normalize(in Layout) (Layout, error) {
	if len(in.Widgets) > MaxWidgets {
		return Layout{}, fmt.Errorf("%w: at most %d widgets", ErrInvalidLayout, MaxWidgets)
	}
	out := Layout{Widgets: make([]Widget, len(in.Widgets))}
	seen := make(map[string]bool, len(in.Widgets))
	for i, widget := range in.Widgets {
		widget.ID = strings.TrimSpace(widget.ID)
		widget.Kind = strings.TrimSpace(widget.Kind)
		widget.Range = strings.TrimSpace(widget.Range)
		widget.Scope = strings.TrimSpace(widget.Scope)
		switch {
		case widget.ID == "":
			return Layout{}, fmt.Errorf("%w: widget %d has no id", ErrInvalidLayout, i)
		case len(widget.ID) > 64:
			return Layout{}, fmt.Errorf("%w: widget id is too long", ErrInvalidLayout)
		case seen[widget.ID]:
			return Layout{}, fmt.Errorf("%w: duplicate widget id %q", ErrInvalidLayout, widget.ID)
		case !validKind(widget.Kind):
			return Layout{}, fmt.Errorf("%w: invalid widget kind %q", ErrInvalidLayout, widget.Kind)
		case widget.Width < 1 || widget.Width > GridColumns:
			return Layout{}, fmt.Errorf(
				"%w: widget %q width must be 1-%d",
				ErrInvalidLayout,
				widget.ID,
				GridColumns,
			)
		case widget.Height < 1 || widget.Height > GridRows:
			return Layout{}, fmt.Errorf(
				"%w: widget %q height must be 1-%d",
				ErrInvalidLayout,
				widget.ID,
				GridRows,
			)
		case !usageRanges[widget.Range]:
			return Layout{}, fmt.Errorf(
				"%w: widget %q has an unknown time frame %q",
				ErrInvalidLayout,
				widget.ID,
				widget.Range,
			)
		case !validScope(widget.Scope):
			return Layout{}, fmt.Errorf(
				"%w: widget %q has an invalid scope %q",
				ErrInvalidLayout,
				widget.ID,
				widget.Scope,
			)
		}
		seen[widget.ID] = true
		out.Widgets[i] = widget
	}
	if _, err := Place(out.Widgets); err != nil {
		return Layout{}, err
	}
	return out, nil
}

func validKind(kind string) bool {
	if kind == "" || len(kind) > 100 {
		return false
	}
	for _, r := range kind {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func validScope(scope string) bool {
	if scope == "" {
		return true
	}
	kind, id, ok := strings.Cut(scope, ":")
	if !ok || (kind != KindAgent && kind != KindProject) {
		return false
	}
	return id != "" && len(scope) <= 100
}

func Place(widgets []Widget) ([]Placement, error) {
	var taken [GridRows][GridColumns]bool
	out := make([]Placement, len(widgets))
	for i, widget := range widgets {
		placed := false
		for y := 0; y+widget.Height <= GridRows && !placed; y++ {
			for x := 0; x+widget.Width <= GridColumns && !placed; x++ {
				if !fits(&taken, x, y, widget.Width, widget.Height) {
					continue
				}
				for dy := 0; dy < widget.Height; dy++ {
					for dx := 0; dx < widget.Width; dx++ {
						taken[y+dy][x+dx] = true
					}
				}
				out[i] = Placement{X: x, Y: y}
				placed = true
			}
		}
		if !placed {
			return nil, fmt.Errorf(
				"%w: widget %q does not fit the %dx%d grid",
				ErrInvalidLayout, widget.ID, GridColumns, GridRows,
			)
		}
	}
	return out, nil
}

func fits(taken *[GridRows][GridColumns]bool, x, y, width, height int) bool {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			if taken[y+dy][x+dx] {
				return false
			}
		}
	}
	return true
}

func (s *Service) Page(ctx context.Context, page Page) (Layout, bool, error) {
	if err := page.Validate(); err != nil {
		return Layout{}, false, err
	}
	layout, found, err := s.read(ctx, pageKey(page))
	if err != nil || found {
		return layout, false, err
	}
	if page.Kind == KindAgency {
		return Layout{}, false, nil
	}
	fallback, _, err := s.read(ctx, defaultsKey(page.Kind))
	if err != nil {
		return Layout{}, false, err
	}
	return fallback, true, nil
}

func (s *Service) Fallback(ctx context.Context, kind string) (Layout, error) {
	if kind != KindProject && kind != KindAgent {
		return Layout{}, ErrInvalidPage
	}
	layout, _, err := s.read(ctx, defaultsKey(kind))
	return layout, err
}

func (s *Service) SetPage(ctx context.Context, page Page, layout Layout) (Layout, error) {
	if err := page.Validate(); err != nil {
		return Layout{}, err
	}
	normalized, err := Normalize(layout)
	if err != nil {
		return Layout{}, err
	}
	if err := s.write(ctx, pageKey(page), normalized); err != nil {
		return Layout{}, err
	}
	return normalized, nil
}

func (s *Service) ResetPage(ctx context.Context, page Page) error {
	if err := page.Validate(); err != nil {
		return err
	}
	return s.store.DeleteSetting(ctx, pageKey(page))
}

func (s *Service) DeletePage(ctx context.Context, page Page) error {
	if err := page.Validate(); err != nil || page.Kind == KindAgency {
		return err
	}
	return s.store.DeleteSetting(ctx, pageKey(page))
}

func (s *Service) Defaults(ctx context.Context) (Layout, Layout, error) {
	project, _, err := s.read(ctx, defaultsKey(KindProject))
	if err != nil {
		return Layout{}, Layout{}, err
	}
	agent, _, err := s.read(ctx, defaultsKey(KindAgent))
	if err != nil {
		return Layout{}, Layout{}, err
	}
	return project, agent, nil
}

func (s *Service) SetDefaults(ctx context.Context, project, agent Layout) (Layout, Layout, error) {
	normalizedProject, err := Normalize(project)
	if err != nil {
		return Layout{}, Layout{}, err
	}
	normalizedAgent, err := Normalize(agent)
	if err != nil {
		return Layout{}, Layout{}, err
	}
	if err := s.write(ctx, defaultsKey(KindProject), normalizedProject); err != nil {
		return Layout{}, Layout{}, err
	}
	if err := s.write(ctx, defaultsKey(KindAgent), normalizedAgent); err != nil {
		return Layout{}, Layout{}, err
	}
	return normalizedProject, normalizedAgent, nil
}

func (p Page) Validate() error {
	switch p.Kind {
	case KindAgency:
		if p.ID != "" {
			return fmt.Errorf("%w: the agency page takes no id", ErrInvalidPage)
		}
	case KindProject, KindAgent:
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("%w: %s page needs an id", ErrInvalidPage, p.Kind)
		}
	default:
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidPage, p.Kind)
	}
	return nil
}

func (s *Service) read(ctx context.Context, key string) (Layout, bool, error) {
	raw, err := s.store.GetSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return Layout{}, false, nil
	}
	if err != nil {
		return Layout{}, false, err
	}
	if strings.TrimSpace(raw) == "" {
		return Layout{}, false, nil
	}
	var layout Layout
	if err := json.Unmarshal([]byte(raw), &layout); err != nil {

		return Layout{}, false, nil
	}
	return layout, true, nil
}

func (s *Service) write(ctx context.Context, key string, layout Layout) error {
	encoded, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	return s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   key,
		Value: string(encoded),
	})
}

func pageKey(page Page) string {
	if page.Kind == KindAgency {
		return "kpi_layout:agency"
	}
	return "kpi_layout:" + page.Kind + ":" + page.ID
}

func defaultsKey(kind string) string {
	return "kpi_layout_default:" + kind
}
