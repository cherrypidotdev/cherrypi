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

package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	storepkg "cherrypi.vladfaust.com/agency/internal/store"
)

var (
	ErrHierarchyCycle   = errors.New("project hierarchy would create a cycle")
	ErrAlreadyHasParent = errors.New(
		"project already has a parent",
	)
	ErrNoParent                 = errors.New("project has no parent")
	ErrParentInvitationNotFound = errors.New(
		"parent invitation not found",
	)
	ErrHierarchyHasReferences = errors.New(
		"project hierarchy change would invalidate existing task references",
	)
)

type SharedOwnerTeam struct {
	ID   int64
	Name string
}

// child of it. CanInvite means the caller owns the parent side and may create

type ParentOption struct {
	Project          Project
	CanInvite        bool
	CanLinkDirectly  bool
	SharedOwnerTeams []SharedOwnerTeam
}

type ParentInvitation struct {
	ID                  int64
	ParentProjectID     string
	ParentName          string
	ParentSlug          string
	ChildProjectID      string
	ChildName           string
	ChildSlug           string
	Status              string
	Note                string
	InvitedByUsername   string
	RespondedByUsername string
	CreatedAt           time.Time
	ExpiresAt           time.Time
	RespondedAt         *time.Time
}

type Hierarchy struct {
	Project        Project
	Parent         *Project
	Children       []Project
	Incoming       *ParentInvitation
	Outgoing       []ParentInvitation
	ParentOptions  []ParentOption
	ChildOptions   []ParentOption
	CanDetach      bool
	CanRespond     bool
	CanInviteChild bool
}

type ParentLinkResult struct {
	Project    Project
	Invitation *ParentInvitation
	Direct     bool
}

func (s *Service) Parent(ctx context.Context, projectID string) (Project, bool, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return Project{}, false, err
	}
	if p.ParentID == "" {
		return Project{}, false, nil
	}
	parent, err := s.Get(ctx, p.ParentID)
	if err != nil {
		return Project{}, false, err
	}
	return parent, true, nil
}

func (s *Service) Children(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) ([]Project, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if _, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer); err != nil {
		return nil, err
	}
	return s.children(ctx, p.ID)
}

func (s *Service) children(ctx context.Context, projectID string) ([]Project, error) {
	rows, err := s.store.ListProjectChildren(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = toProject(row)
	}
	return out, nil
}

func (s *Service) Ancestors(ctx context.Context, projectID string) ([]Project, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ids, err := s.store.ListProjectAncestorIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]Project, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		ancestor, err := s.Get(ctx, ids[i])
		if err != nil {
			return nil, err
		}
		out = append(out, ancestor)
	}
	return out, nil
}

func (s *Service) ReferenceAllowed(
	ctx context.Context,
	fromProjectID, toProjectID string,
) (bool, error) {
	if fromProjectID == "" || toProjectID == "" || fromProjectID == toProjectID {
		return true, nil
	}
	descendant, err := s.store.IsProjectDescendant(ctx, toProjectID, fromProjectID)
	if err != nil {
		return false, err
	}
	if descendant {
		return true, nil
	}
	from, err := s.Get(ctx, fromProjectID)
	if err != nil {
		return false, err
	}
	to, err := s.Get(ctx, toProjectID)
	if err != nil {
		return false, err
	}
	return from.ParentID == to.ParentID, nil
}

func (s *Service) ReferenceProjectIDs(
	ctx context.Context,
	projectID string,
) ([]string, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	descendants, err := s.store.ListProjectDescendantIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	all, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	out := []string{p.ID}
	out = append(out, descendants...)
	siblings := make([]string, 0, len(all))
	for _, row := range all {
		if row.ID == p.ID || row.ParentProjectID.String != p.ParentID {
			continue
		}
		siblings = append(siblings, row.ID)
	}
	sort.Strings(siblings)
	return append(out, siblings...), nil
}

func (s *Service) Hierarchy(
	ctx context.Context,
	projectID string,
	actorID ids.ActorID,
	isAdmin bool,
) (Hierarchy, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return Hierarchy{}, err
	}
	role, err := s.RequireRole(ctx, p.ID, actorID, isAdmin, RoleViewer)
	if err != nil {
		return Hierarchy{}, err
	}
	children, err := s.children(ctx, p.ID)
	if err != nil {
		return Hierarchy{}, err
	}
	out := Hierarchy{Project: p, Children: children}
	if parent, ok, err := s.Parent(ctx, p.ID); err != nil {
		return Hierarchy{}, err
	} else if ok {
		out.Parent = &parent
	}
	if row, err := s.store.GetPendingProjectParentInvitationForChild(
		ctx,
		p.ID,
	); err == nil {
		parentName, parentSlug := "", ""
		if parent, err := s.Get(ctx, row.ParentProjectID); err == nil {
			parentName, parentSlug = parent.Name, parent.Slug
		}
		invitation := toParentInvitationRow(
			row,
			parentName,
			parentSlug,
			p.Name,
			p.Slug,
		)
		invitation.InvitedByUsername = actorLabel(
			ctx, s, ids.ActorID(row.InvitedByActorID.Int64),
		)
		invitation.RespondedByUsername = actorLabel(
			ctx, s, ids.ActorID(row.RespondedByActorID.Int64),
		)
		out.Incoming = &invitation
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Hierarchy{}, err
	}
	invitations, err := s.store.ListProjectParentInvitationsByParent(ctx, p.ID)
	if err != nil {
		return Hierarchy{}, err
	}
	out.Outgoing = make([]ParentInvitation, len(invitations))
	for i, row := range invitations {
		out.Outgoing[i] = toParentInvitationByParentRow(row)
	}

	owner := isAdmin || role == RoleOwner
	out.CanDetach = out.Parent != nil && owner
	out.CanRespond = out.Incoming != nil && owner
	out.CanInviteChild = owner

	parentOptions, err := s.parentOptions(ctx, p, actorID, isAdmin)
	if err != nil {
		return Hierarchy{}, err
	}
	out.ParentOptions = parentOptions
	childOptions, err := s.childOptions(ctx, p, actorID, isAdmin, owner)
	if err != nil {
		return Hierarchy{}, err
	}
	out.ChildOptions = childOptions
	return out, nil
}

func (s *Service) parentOptions(
	ctx context.Context,
	p Project,
	actorID ids.ActorID,
	isAdmin bool,
) ([]ParentOption, error) {
	if p.ParentID != "" {
		return nil, nil
	}
	descendants, err := s.store.ListProjectDescendantIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	blocked := make(map[string]bool, len(descendants)+1)
	blocked[p.ID] = true
	for _, id := range descendants {
		blocked[id] = true
	}
	all, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	out := []ParentOption{}
	for _, row := range all {
		candidate := toProject(row)
		if blocked[candidate.ID] {
			continue
		}
		option, ok, err := s.parentOption(ctx, candidate, p, actorID, isAdmin, true)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, option)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Project.Name) <
			strings.ToLower(out[j].Project.Name)
	})
	return out, nil
}

func (s *Service) childOptions(
	ctx context.Context,
	p Project,
	actorID ids.ActorID,
	isAdmin bool,
	owner bool,
) ([]ParentOption, error) {
	ancestors, err := s.store.ListProjectAncestorIDs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	blocked := make(map[string]bool, len(ancestors)+1)
	blocked[p.ID] = true
	for _, id := range ancestors {
		blocked[id] = true
	}
	all, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	out := []ParentOption{}
	for _, row := range all {
		candidate := toProject(row)
		if blocked[candidate.ID] || candidate.ParentID != "" {
			continue
		}
		option, ok, err := s.parentOption(ctx, p, candidate, actorID, isAdmin, owner)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, option)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Project.Name) <
			strings.ToLower(out[j].Project.Name)
	})
	return out, nil
}

// invite; a shared owner team still allows a direct link without that.
func (s *Service) parentOption(
	ctx context.Context,
	parent Project,
	child Project,
	actorID ids.ActorID,
	isAdmin bool,
	requireOwner bool,
) (ParentOption, bool, error) {
	shared, err := s.sharedOwnerTeams(ctx, parent.ID, child.ID, actorID)
	if err != nil {
		return ParentOption{}, false, err
	}
	canInvite := isAdmin
	if requireOwner && !canInvite {
		role, ok, err := s.EffectiveRole(ctx, parent.ID, actorID, false)
		if err != nil {
			return ParentOption{}, false, err
		}
		canInvite = ok && role == RoleOwner
	}
	direct := isAdmin || len(shared) > 0
	if !canInvite && !direct {
		return ParentOption{}, false, nil
	}
	return ParentOption{
		Project:          child,
		CanInvite:        canInvite,
		CanLinkDirectly:  direct,
		SharedOwnerTeams: shared,
	}, true, nil
}

func (s *Service) sharedOwnerTeams(
	ctx context.Context,
	parentID, childID string,
	actorID ids.ActorID,
) ([]SharedOwnerTeam, error) {
	if actorID == 0 {
		return nil, nil
	}
	rows, err := s.store.ListProjectSharedOwnerTeams(ctx, parentID, childID, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]SharedOwnerTeam, len(rows))
	for i, row := range rows {
		out[i] = SharedOwnerTeam{ID: row.TeamID, Name: row.TeamName}
	}
	return out, nil
}

// InviteParent links a child to a parent when the caller may shortcut the

func (s *Service) InviteParent(
	ctx context.Context,
	childID, parentID, note string,
	actorID ids.ActorID,
	isAdmin bool,
) (ParentLinkResult, error) {
	parent, err := s.Get(ctx, parentID)
	if err != nil {
		return ParentLinkResult{}, err
	}
	child, err := s.Get(ctx, childID)
	if err != nil {
		return ParentLinkResult{}, err
	}
	if parent.ID == child.ID {
		return ParentLinkResult{}, ErrHierarchyCycle
	}
	if child.ParentID != "" {
		return ParentLinkResult{}, ErrAlreadyHasParent
	}
	if _, err := s.store.GetPendingProjectParentInvitationForChild(
		ctx,
		child.ID,
	); err == nil {
		return ParentLinkResult{}, ErrInvitationExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ParentLinkResult{}, err
	}
	cycle, err := s.store.IsProjectDescendant(ctx, parent.ID, child.ID)
	if err != nil {
		return ParentLinkResult{}, err
	}
	if cycle {
		return ParentLinkResult{}, ErrHierarchyCycle
	}

	shared, err := s.sharedOwnerTeams(ctx, parent.ID, child.ID, actorID)
	if err != nil {
		return ParentLinkResult{}, err
	}
	direct := isAdmin || len(shared) > 0
	if !direct {
		if _, err := s.RequireRole(ctx, parent.ID, actorID, isAdmin, RoleOwner); err != nil {
			return ParentLinkResult{}, err
		}
	}
	if direct {
		updated, err := s.linkParent(ctx, child, parent, actorID)
		if err != nil {
			return ParentLinkResult{}, err
		}
		return ParentLinkResult{Project: updated, Direct: true}, nil
	}

	if err := s.validateHierarchyChange(ctx, child.ID, parent.ID); err != nil {
		return ParentLinkResult{}, err
	}
	now := time.Now()
	row, err := s.store.CreateProjectParentInvitation(
		ctx,
		storepkg.CreateProjectParentInvitationParams{
			ParentProjectID:  parent.ID,
			ChildProjectID:   child.ID,
			InvitedByActorID: nullActor(actorID),
			Note:             strings.TrimSpace(note),
			CreatedAt:        now.UnixMilli(),
			ExpiresAt:        now.Add(InviteTTL).UnixMilli(),
		},
	)
	if err != nil {
		return ParentLinkResult{}, err
	}
	s.writeLog(ctx, parent.ID, audit.KindHierarchy, "invitation", "", child.Name,
		"Invited "+child.Name+" to become a subproject",
		map[string]any{"child_project_id": child.ID})
	s.writeLog(ctx, child.ID, audit.KindHierarchy, "invitation", "", parent.Name,
		"Invited to become a subproject of "+parent.Name,
		map[string]any{"parent_project_id": parent.ID})
	invitation := toParentInvitationRow(
		row,
		parent.Name,
		parent.Slug,
		child.Name,
		child.Slug,
	)
	return ParentLinkResult{Project: child, Invitation: &invitation}, nil
}

func (s *Service) RespondParentInvitation(
	ctx context.Context,
	invitationID int64,
	actorID ids.ActorID,
	isAdmin bool,
	approve bool,
) (ParentInvitation, error) {
	row, err := s.store.GetProjectParentInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return ParentInvitation{}, ErrParentInvitationNotFound
	}
	if err != nil {
		return ParentInvitation{}, err
	}
	if row.Status != "pending" {
		return ParentInvitation{}, ErrParentInvitationNotFound
	}
	now := time.Now()
	if row.ExpiresAt < now.UnixMilli() {
		_, _ = s.store.RespondProjectParentInvitation(
			ctx,
			storepkg.RespondProjectParentInvitationParams{
				Status:             "revoked",
				RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
				RespondedByActorID: sql.NullInt64{},
				ID:                 invitationID,
			},
		)
		return ParentInvitation{}, ErrInvitationExpired
	}
	if _, err := s.RequireRole(
		ctx,
		row.ChildProjectID,
		actorID,
		isAdmin,
		RoleOwner,
	); err != nil {
		return ParentInvitation{}, err
	}

	parent, _ := s.Get(ctx, row.ParentProjectID)
	child, _ := s.Get(ctx, row.ChildProjectID)
	if approve {
		if child.ParentID != "" {
			return ParentInvitation{}, ErrAlreadyHasParent
		}
		if err := s.validateHierarchyChange(ctx, child.ID, parent.ID); err != nil {
			return ParentInvitation{}, err
		}
	}

	status := "declined"
	if approve {
		status = "accepted"
	}
	n, err := s.store.RespondProjectParentInvitation(
		ctx,
		storepkg.RespondProjectParentInvitationParams{
			Status:             status,
			RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			RespondedByActorID: nullActor(actorID),
			ID:                 invitationID,
		},
	)
	if err != nil {
		return ParentInvitation{}, err
	}
	if n == 0 {
		return ParentInvitation{}, ErrParentInvitationNotFound
	}

	if approve {
		if _, err := s.linkParent(ctx, child, parent, actorID); err != nil {
			return ParentInvitation{}, err
		}

		_, _ = s.store.RevokePendingProjectParentInvitationsForChild(
			ctx,
			storepkg.RevokePendingProjectParentInvitationsForChildParams{
				RespondedAt:        sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
				RespondedByActorID: nullActor(actorID),
				ChildProjectID:     child.ID,
			},
		)
	} else {
		s.writeLog(ctx, parent.ID, audit.KindHierarchy, "invitation", child.Name, "declined",
			child.Name+" declined the parent invitation",
			map[string]any{"child_project_id": child.ID})
		s.writeLog(ctx, child.ID, audit.KindHierarchy, "invitation", parent.Name, "declined",
			"Declined becoming a subproject of "+parent.Name,
			map[string]any{"parent_project_id": parent.ID})
	}
	return toParentInvitationRow(row, parent.Name, parent.Slug, child.Name, child.Slug),
		nil
}

func (s *Service) RevokeParentInvitation(
	ctx context.Context,
	invitationID int64,
	actorID ids.ActorID,
	isAdmin bool,
) error {
	row, err := s.store.GetProjectParentInvitation(ctx, invitationID)
	if errors.Is(err, sql.ErrNoRows) || row.Status != "pending" {
		return ErrParentInvitationNotFound
	}
	if err != nil {
		return err
	}
	if !isAdmin && row.InvitedByActorID.Int64 != int64(actorID) {
		if _, err := s.RequireRole(
			ctx,
			row.ParentProjectID,
			actorID,
			false,
			RoleOwner,
		); err != nil {
			return err
		}
	}
	n, err := s.store.RevokeProjectParentInvitation(
		ctx,
		storepkg.RevokeProjectParentInvitationParams{
			RespondedAt:        sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			RespondedByActorID: nullActor(actorID),
			ID:                 invitationID,
		},
	)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrParentInvitationNotFound
	}
	parent, _ := s.Get(ctx, row.ParentProjectID)
	child, _ := s.Get(ctx, row.ChildProjectID)
	s.writeLog(ctx, parent.ID, audit.KindHierarchy, "invitation", child.Name, "",
		"Revoked the parent invitation for "+child.Name,
		map[string]any{"child_project_id": child.ID})
	s.writeLog(ctx, child.ID, audit.KindHierarchy, "invitation", parent.Name, "",
		"Revoked the parent invitation from "+parent.Name,
		map[string]any{"parent_project_id": parent.ID})
	return nil
}

func (s *Service) DetachParent(
	ctx context.Context,
	childID string,
	actorID ids.ActorID,
	isAdmin bool,
) (Project, error) {
	child, err := s.Get(ctx, childID)
	if err != nil {
		return Project{}, err
	}
	if child.ParentID == "" {
		return Project{}, ErrNoParent
	}
	parent, err := s.Get(ctx, child.ParentID)
	if err != nil {
		return Project{}, err
	}
	if _, err := s.RequireRole(ctx, parent.ID, actorID, isAdmin, RoleOwner); err != nil {
		return Project{}, err
	}
	if err := s.validateHierarchyChange(ctx, child.ID, ""); err != nil {
		return Project{}, err
	}
	updated, err := s.setParent(ctx, child, "")
	if err != nil {
		return Project{}, err
	}
	s.writeLog(ctx, parent.ID, audit.KindHierarchy, "parent", child.Name, "",
		"Removed "+child.Name+" as a subproject",
		map[string]any{"child_project_id": child.ID})
	s.writeLog(ctx, child.ID, audit.KindHierarchy, "parent", parent.Name, "",
		"Detached from parent project "+parent.Name,
		map[string]any{"parent_project_id": parent.ID})
	return updated, nil
}

func (s *Service) linkParent(
	ctx context.Context,
	child, parent Project,
	actorID ids.ActorID,
) (Project, error) {
	if err := s.validateHierarchyChange(ctx, child.ID, parent.ID); err != nil {
		return Project{}, err
	}
	updated, err := s.setParent(ctx, child, parent.ID)
	if err != nil {
		return Project{}, err
	}
	s.writeLog(ctx, parent.ID, audit.KindHierarchy, "parent", "", child.Name,
		"Added "+child.Name+" as a subproject",
		map[string]any{"child_project_id": child.ID})
	s.writeLog(ctx, child.ID, audit.KindHierarchy, "parent", "", parent.Name,
		"Attached to parent project "+parent.Name,
		map[string]any{"parent_project_id": parent.ID})
	return updated, nil
}

func (s *Service) setParent(
	ctx context.Context,
	child Project,
	parentID string,
) (Project, error) {
	rows, err := s.ancestorRows(ctx, child.ID, parentID)
	if err != nil {
		return Project{}, err
	}
	row, err := s.store.SetProjectParentTree(ctx, child.ID, parentID, rows)
	if err != nil {
		return Project{}, err
	}
	return toProject(row), nil
}

func (s *Service) ancestorRows(
	ctx context.Context,
	childID, parentID string,
) ([]storepkg.InsertProjectAncestorParams, error) {
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	parents := make(map[string]string, len(projects))
	for _, row := range projects {
		parents[row.ID] = row.ParentProjectID.String
	}
	parents[childID] = parentID
	rows := []storepkg.InsertProjectAncestorParams{}
	for _, row := range projects {
		seen := map[string]bool{row.ID: true}
		current := parents[row.ID]
		depth := int64(1)
		for current != "" {
			if seen[current] {
				return nil, ErrHierarchyCycle
			}
			seen[current] = true
			rows = append(rows, storepkg.InsertProjectAncestorParams{
				ProjectID:  row.ID,
				AncestorID: current,
				Depth:      depth,
			})
			current = parents[current]
			depth++
		}
	}
	return rows, nil
}

func (s *Service) validateHierarchyChange(
	ctx context.Context,
	childID, parentID string,
) error {
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		return err
	}
	parents := make(map[string]string, len(projects))
	for _, row := range projects {
		parents[row.ID] = row.ParentProjectID.String
	}
	parents[childID] = parentID

	for id := range parents {
		seen := map[string]bool{id: true}
		for current := parents[id]; current != ""; current = parents[current] {
			if seen[current] {
				return ErrHierarchyCycle
			}
			seen[current] = true
		}
	}
	references, err := s.store.ListCrossProjectTaskReferences(ctx)
	if err != nil {
		return err
	}
	for _, edge := range references {
		if !referenceAllowedInTree(
			parents,
			edge.TaskProjectID.String,
			edge.ReferencedProjectID.String,
		) {
			return fmt.Errorf(
				"%w: %s in %s may not reference %s in %s",
				ErrHierarchyHasReferences,
				edge.TaskID,
				edge.TaskProjectID.String,
				edge.ReferencedTaskID,
				edge.ReferencedProjectID.String,
			)
		}
	}
	return nil
}

func referenceAllowedInTree(parents map[string]string, from, to string) bool {
	if from == "" || to == "" || from == to {
		return true
	}
	if descendantInTree(parents, to, from) {
		return true
	}
	return parents[from] == parents[to]
}

func descendantInTree(parents map[string]string, node, ancestor string) bool {
	seen := map[string]bool{node: true}
	for current := parents[node]; current != ""; current = parents[current] {
		if current == ancestor {
			return true
		}
		if seen[current] {
			return false
		}
		seen[current] = true
	}
	return false
}

func toParentInvitationRow(
	row storepkg.ProjectParentInvitation,
	parentName, parentSlug, childName, childSlug string,
) ParentInvitation {
	return ParentInvitation{
		ID:              row.ID,
		ParentProjectID: row.ParentProjectID,
		ParentName:      parentName,
		ParentSlug:      parentSlug,
		ChildProjectID:  row.ChildProjectID,
		ChildName:       childName,
		ChildSlug:       childSlug,
		Status:          row.Status,
		Note:            row.Note,
		CreatedAt:       time.UnixMilli(row.CreatedAt),
		ExpiresAt:       time.UnixMilli(row.ExpiresAt),
		RespondedAt:     nullTime(row.RespondedAt),
	}
}

func toParentInvitationByParentRow(
	row storepkg.ListProjectParentInvitationsByParentRow,
) ParentInvitation {
	return ParentInvitation{
		ID:                  row.ID,
		ParentProjectID:     row.ParentProjectID,
		ParentName:          row.ParentName,
		ParentSlug:          row.ParentSlug,
		ChildProjectID:      row.ChildProjectID,
		ChildName:           row.ChildName,
		ChildSlug:           row.ChildSlug,
		Status:              row.Status,
		Note:                row.Note,
		InvitedByUsername:   row.InvitedByUsername.String,
		RespondedByUsername: row.RespondedByUsername.String,
		CreatedAt:           time.UnixMilli(row.CreatedAt),
		ExpiresAt:           time.UnixMilli(row.ExpiresAt),
		RespondedAt:         nullTime(row.RespondedAt),
	}
}
