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
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	pluginhost "cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/projects"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	Plugins  *pluginhost.Service
	Projects *projects.Service
	Agents   *agents.Registry
	Accounts *accounts.Service
}

func (s *Server) ListPlugins(
	ctx context.Context,
	_ *agencyv1.ListPluginsRequest,
) (*agencyv1.ListPluginsResponse, error) {
	if _, err := s.identity(ctx); err != nil {
		return nil, err
	}
	list, err := s.Plugins.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.Plugin, len(list))
	for i, p := range list {
		out[i] = toProtoPlugin(p)
	}
	return &agencyv1.ListPluginsResponse{Plugins: out}, nil
}

func (s *Server) GetPlugin(
	ctx context.Context,
	req *agencyv1.GetPluginRequest,
) (*agencyv1.GetPluginResponse, error) {
	if _, err := s.identity(ctx); err != nil {
		return nil, err
	}
	p, err := s.Plugins.Get(ctx, req.Id)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &agencyv1.GetPluginResponse{Plugin: toProtoPlugin(p)}, nil
}

func (s *Server) InspectPlugin(
	ctx context.Context,
	req *agencyv1.InspectPluginRequest,
) (*agencyv1.InspectPluginResponse, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if len(req.Bundle) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bundle is required"))
	}
	preview, err := s.Plugins.Inspect(ctx, req.Bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &agencyv1.InspectPluginResponse{Preview: toProtoPreview(preview)}, nil
}

func (s *Server) InstallPlugin(
	ctx context.Context,
	req *agencyv1.InstallPluginRequest,
) (*agencyv1.InstallPluginResponse, error) {
	identity, err := s.requireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Bundle) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bundle is required"))
	}
	p, err := s.Plugins.Install(ctx, int64(identity.ActorID), req.Bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &agencyv1.InstallPluginResponse{Plugin: toProtoPlugin(p)}, nil
}

func (s *Server) SetPluginEnabled(
	ctx context.Context,
	req *agencyv1.SetPluginEnabledRequest,
) (*agencyv1.SetPluginEnabledResponse, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.Plugins.SetEnabled(ctx, req.Id, req.Enabled); err != nil {
		return nil, mapPluginError(err)
	}
	p, err := s.Plugins.Get(ctx, req.Id)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &agencyv1.SetPluginEnabledResponse{Plugin: toProtoPlugin(p)}, nil
}

func (s *Server) UninstallPlugin(
	ctx context.Context,
	req *agencyv1.UninstallPluginRequest,
) (*agencyv1.UninstallPluginResponse, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.Plugins.Uninstall(ctx, req.Id, req.Purge); err != nil {
		return nil, mapPluginError(err)
	}
	return &agencyv1.UninstallPluginResponse{}, nil
}

func (s *Server) ListPluginLogs(
	ctx context.Context,
	req *agencyv1.ListPluginLogsRequest,
) (*agencyv1.ListPluginLogsResponse, error) {
	if err := s.requirePluginLogReader(ctx); err != nil {
		return nil, err
	}
	rows, err := s.Plugins.Logs(ctx, req.Id, int(req.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.PluginLog, len(rows))
	for i, row := range rows {
		out[i] = toProtoLog(row)
	}
	return &agencyv1.ListPluginLogsResponse{Logs: out}, nil
}

func (s *Server) ListPluginRuntimeLogs(
	ctx context.Context,
	req *agencyv1.ListPluginRuntimeLogsRequest,
) (*agencyv1.ListPluginRuntimeLogsResponse, error) {
	if _, _, err := s.authorizeLogScope(ctx, req.Scope, req.TargetId); err != nil {
		return nil, err
	}
	minLevel := ""
	if req.MinLevel != "" {
		normalized, ok := pluginhost.NormalizeLogLevel(req.MinLevel)
		if !ok {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("min_level must be trace, debug, info, warn, or error"),
			)
		}
		minLevel = normalized
	}
	rows, err := s.Plugins.RuntimeLogs(ctx, req.PluginId, pluginhost.RuntimeLogFilter{
		Scope:    req.Scope,
		TargetID: req.TargetId,
		MinLevel: minLevel,
		Search:   req.Search,
		Limit:    int(req.Limit),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.PluginLog, len(rows))
	for i, row := range rows {
		out[i] = toProtoLog(row)
	}
	return &agencyv1.ListPluginRuntimeLogsResponse{Logs: out}, nil
}

func (s *Server) GetPluginSettings(
	ctx context.Context,
	req *agencyv1.GetPluginSettingsRequest,
) (*agencyv1.GetPluginSettingsResponse, error) {
	scope, target, err := s.authorizeSurfaces(ctx, req.Scope, req.TargetId)
	if err != nil {
		return nil, err
	}
	p, err := s.Plugins.Get(ctx, req.PluginId)
	if err != nil {
		return nil, mapPluginError(err)
	}
	out, err := s.pluginSettings(ctx, p, scope, target)
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetPluginSettingsResponse{Settings: out}, nil
}

func (s *Server) UpdatePluginSettings(
	ctx context.Context,
	req *agencyv1.UpdatePluginSettingsRequest,
) (*agencyv1.UpdatePluginSettingsResponse, error) {
	scope, target, err := s.authorizeSurfaces(ctx, req.Scope, req.TargetId)
	if err != nil {
		return nil, err
	}
	identity, _ := s.identity(ctx)
	values := map[string]any{}
	if req.SettingsJson != "" {
		if err := json.Unmarshal([]byte(req.SettingsJson), &values); err != nil {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("settings_json must be a JSON object"),
			)
		}
	}
	err = s.Plugins.UpdateSettings(
		ctx, req.PluginId, scope, target, values, int64(identity.ActorID),
	)
	if err != nil {
		return nil, mapPluginError(err)
	}
	p, err := s.Plugins.Get(ctx, req.PluginId)
	if err != nil {
		return nil, mapPluginError(err)
	}
	out, err := s.pluginSettings(ctx, p, scope, target)
	if err != nil {
		return nil, err
	}
	return &agencyv1.UpdatePluginSettingsResponse{Settings: out}, nil
}

func (s *Server) SetPluginSettingsEnabled(
	ctx context.Context,
	req *agencyv1.SetPluginSettingsEnabledRequest,
) (*agencyv1.SetPluginSettingsEnabledResponse, error) {
	scope, target, err := s.authorizeSurfaces(ctx, req.Scope, req.TargetId)
	if err != nil {
		return nil, err
	}
	if scope == pluginhost.ScopeOrg || scope == pluginhost.ScopeProject {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("org is enabled by installation and projects use UpdateProjectPlugin"),
		)
	}
	identity, _ := s.identity(ctx)
	if err := s.Plugins.SetSettingsEnabled(
		ctx, req.PluginId, scope, target, req.Enabled, int64(identity.ActorID),
	); err != nil {
		return nil, mapPluginError(err)
	}
	p, err := s.Plugins.Get(ctx, req.PluginId)
	if err != nil {
		return nil, mapPluginError(err)
	}
	out, err := s.pluginSettings(ctx, p, scope, target)
	if err != nil {
		return nil, err
	}
	return &agencyv1.SetPluginSettingsEnabledResponse{Settings: out}, nil
}

func (s *Server) ListSurfacePlugins(
	ctx context.Context,
	req *agencyv1.ListSurfacePluginsRequest,
) (*agencyv1.ListSurfacePluginsResponse, error) {
	scope, target, err := s.authorizeSurfaces(ctx, req.Scope, req.TargetId)
	if err != nil {
		return nil, err
	}
	list, err := s.Plugins.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &agencyv1.ListSurfacePluginsResponse{}
	admin := s.isAdmin(ctx)
	for _, p := range list {
		if !p.Manifest.HasSurface(scope) {
			continue
		}
		if scope == pluginhost.ScopeProject {
			row, err := s.projectPlugin(ctx, p, target, admin)
			if err != nil {
				return nil, err
			}
			out.ProjectPlugins = append(out.ProjectPlugins, row)
			continue
		}
		settings, err := s.pluginSettings(ctx, p, scope, target)
		if err != nil {
			return nil, err
		}
		out.Settings = append(out.Settings, settings)
	}
	return out, nil
}

func (s *Server) GetProjectPlugin(
	ctx context.Context,
	req *agencyv1.GetProjectPluginRequest,
) (*agencyv1.GetProjectPluginResponse, error) {
	if _, err := s.requireProjectOwner(ctx, req.ProjectId); err != nil {
		return nil, err
	}
	p, err := s.Plugins.Get(ctx, req.PluginId)
	if err != nil {
		return nil, mapPluginError(err)
	}
	out, err := s.projectPlugin(ctx, p, req.ProjectId, s.isAdmin(ctx))
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetProjectPluginResponse{ProjectPlugin: out}, nil
}

func (s *Server) UpdateProjectPlugin(
	ctx context.Context,
	req *agencyv1.UpdateProjectPluginRequest,
) (*agencyv1.UpdateProjectPluginResponse, error) {
	identity, err := s.requireProjectOwner(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}
	p, err := s.Plugins.Get(ctx, req.PluginId)
	if err != nil {
		return nil, mapPluginError(err)
	}
	if req.MaxPriority < -1 || req.MaxPriority > 2 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("max_priority must be between -1 and 2"),
		)
	}
	if req.Enabled {
		if err := s.requireMembership(ctx, req.ProjectId, p); err != nil {
			return nil, err
		}
	}
	var values map[string]any
	if req.SettingsJson != nil {
		values = map[string]any{}
		if err := json.Unmarshal([]byte(*req.SettingsJson), &values); err != nil {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("settings_json must be a JSON object"),
			)
		}
	}
	err = s.Plugins.UpdateProjectSettings(
		ctx,
		req.PluginId,
		req.ProjectId,
		req.Enabled,
		req.AllowTaskCreation,
		int64(req.MaxPriority),
		values,
		int64(identity.ActorID),
	)
	if err != nil {
		return nil, mapPluginError(err)
	}
	out, err := s.projectPlugin(ctx, p, req.ProjectId, s.isAdmin(ctx))
	if err != nil {
		return nil, err
	}
	return &agencyv1.UpdateProjectPluginResponse{ProjectPlugin: out}, nil
}

func (s *Server) InvitePluginToProject(
	ctx context.Context,
	req *agencyv1.InvitePluginToProjectRequest,
) (*agencyv1.InvitePluginToProjectResponse, error) {
	identity, err := s.requireProjectOwner(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}
	p, err := s.Plugins.Get(ctx, req.PluginId)
	if err != nil {
		return nil, mapPluginError(err)
	}
	role := req.Role
	if role == "" {
		role = projects.RoleMember
	}
	if _, err := s.Projects.AddMember(
		ctx,
		req.ProjectId,
		ids.ActorID(p.ActorID),
		role,
		identity.ActorID,
		identity.IsAdmin(),
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out, err := s.projectPlugin(ctx, p, req.ProjectId, s.isAdmin(ctx))
	if err != nil {
		return nil, err
	}
	return &agencyv1.InvitePluginToProjectResponse{ProjectPlugin: out}, nil
}

func (s *Server) identity(ctx context.Context) (auth.Identity, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return auth.Identity{}, connect.NewError(
			connect.CodeUnauthenticated,
			errors.New("unauthorized"),
		)
	}
	return identity, nil
}

func (s *Server) isAdmin(ctx context.Context) bool {
	identity, ok := auth.IdentityFrom(ctx)
	return ok && identity.IsAdmin()
}

func (s *Server) requireAdmin(ctx context.Context) (auth.Identity, error) {
	identity, err := s.identity(ctx)
	if err != nil {
		return identity, err
	}
	if !identity.IsAdmin() {
		return identity, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("administrator required"),
		)
	}
	return identity, nil
}

func (s *Server) requireProjectOwner(ctx context.Context, projectID string) (auth.Identity, error) {
	identity, err := s.identity(ctx)
	if err != nil {
		return identity, err
	}
	project, err := s.Projects.Get(ctx, projectID)
	if err != nil {
		return identity, connect.NewError(connect.CodeNotFound, errors.New("project not found"))
	}
	if _, err := s.Projects.RequireRole(
		ctx, project.ID, identity.ActorID, identity.IsAdmin(), projects.RoleOwner,
	); err != nil {
		return identity, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("project owner required"),
		)
	}
	return identity, nil
}

func (s *Server) authorizeSurfaces(
	ctx context.Context,
	scope, target string,
) (string, string, error) {
	identity, err := s.identity(ctx)
	if err != nil {
		return "", "", err
	}
	switch scope {
	case pluginhost.ScopeOrg:
		if !identity.IsAdmin() {
			return "", "", connect.NewError(
				connect.CodePermissionDenied,
				errors.New("administrator required"),
			)
		}
		return scope, "", nil
	case pluginhost.ScopeProject:
		if target == "" {
			return "", "", connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("target_id is the project id"),
			)
		}
		if _, err := s.requireProjectOwner(ctx, target); err != nil {
			return "", "", err
		}
		return scope, target, nil
	case pluginhost.ScopeAgent:
		if target == "" {
			return "", "", connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("target_id is the agent actor id"),
			)
		}
		actorID, err := strconv.ParseInt(target, 10, 64)
		if err != nil {
			return "", "", connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("invalid agent id"),
			)
		}
		agent, err := s.Agents.GetByActor(ctx, ids.ActorID(actorID))
		if err != nil {
			return "", "", connect.NewError(connect.CodeNotFound, errors.New("agent not found"))
		}
		if !identity.IsAdmin() {
			owner, err := s.Agents.IsActorOwner(ctx, agent.ActorID, identity.ActorID)
			if err != nil || !owner {
				return "", "", connect.NewError(
					connect.CodePermissionDenied,
					errors.New("agent owner or administrator required"),
				)
			}
		}
		return scope, target, nil
	case pluginhost.ScopeUser:
		if target == "" {
			return "", "", connect.NewError(
				connect.CodeInvalidArgument,
				errors.New("target_id is the user actor id"),
			)
		}
		if target != strconv.FormatInt(int64(identity.ActorID), 10) {
			return "", "", connect.NewError(
				connect.CodePermissionDenied,
				errors.New("a user may only change their own plugin settings"),
			)
		}
		return scope, target, nil
	default:
		return "", "", connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("unknown settings scope"),
		)
	}
}

func (s *Server) authorizeLogScope(
	ctx context.Context,
	scope, target string,
) (string, string, error) {
	if scope == "" {
		if err := s.requirePluginLogReader(ctx); err != nil {
			return "", "", err
		}
		return "", "", nil
	}
	return s.authorizeSurfaces(ctx, scope, target)
}

func (s *Server) requirePluginLogReader(ctx context.Context) error {
	if s.canReadAllPluginLogs(ctx) {
		return nil
	}
	return connect.NewError(
		connect.CodePermissionDenied,
		errors.New("administrator or admin agent required"),
	)
}

func (s *Server) canReadAllPluginLogs(ctx context.Context) bool {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return false
	}
	if identity.IsAdmin() {
		return true
	}
	if identity.Kind != accounts.KindAgent || s.Agents == nil {
		return false
	}
	agent, err := s.Agents.GetByActor(ctx, identity.ActorID)
	return err == nil && agent.Role == agents.RoleAdmin
}

func (s *Server) requireMembership(
	ctx context.Context,
	projectID string,
	p pluginhost.Plugin,
) error {
	role, ok, err := s.Projects.EffectiveRole(
		ctx, projectID, ids.ActorID(p.ActorID), false,
	)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if !ok || role == "" {
		return connect.NewError(
			connect.CodeFailedPrecondition,
			errors.New("invite the plugin actor to the project first"),
		)
	}
	return nil
}

func (s *Server) pluginSettings(
	ctx context.Context,
	p pluginhost.Plugin,
	scope, target string,
) (*agencyv1.PluginSettings, error) {
	values, secrets, err := s.Plugins.SettingsFor(ctx, p.ID, scope, target)
	if err != nil {
		return nil, mapPluginError(err)
	}
	enabled := scope == pluginhost.ScopeOrg
	updated := p.UpdatedAt
	if scope == pluginhost.ScopeProject {
		row, err := s.Plugins.ProjectSettings(ctx, p.ID, target)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		enabled = row.Enabled
		updated = row.Updated
	} else if scope != pluginhost.ScopeOrg {
		row, err := s.Plugins.SettingsRow(ctx, p.ID, scope, target)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		enabled = row.Enabled
		if !row.Updated.IsZero() {
			updated = row.Updated
		}
	}

	raw, err := json.Marshal(values)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.PluginSettings{
		PluginId:     p.ID,
		Scope:        scope,
		TargetId:     target,
		SettingsJson: string(raw),
		SecretKeys:   pluginhost.SecretKeyList(secrets),
		Enabled:      enabled,
		SchemaJson:   string(p.Manifest.SettingsSchema(scope)),
		UpdatedAt:    timestamppb.New(updated),
	}, nil
}

func (s *Server) projectPlugin(
	ctx context.Context,
	p pluginhost.Plugin,
	projectID string,
	admin bool,
) (*agencyv1.ProjectPlugin, error) {
	row, err := s.Plugins.ProjectSettings(ctx, p.ID, projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	role, member, err := s.Projects.EffectiveRole(
		ctx, projectID, ids.ActorID(p.ActorID), false,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	secrets := p.Manifest.SecretKeys(pluginhost.ScopeProject)

	raw, err := json.Marshal(row.Values)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.ProjectPlugin{
		Plugin:            toProtoPlugin(p),
		Member:            member && role != "",
		Enabled:           row.Enabled,
		AllowTaskCreation: row.AllowTaskCreation,
		MaxPriority:       int32(row.MaxPriority),
		SettingsJson:      string(raw),
		SecretKeys:        pluginhost.SecretKeyList(secrets),
		SchemaJson:        string(p.Manifest.SettingsSchema(pluginhost.ScopeProject)),
		CanManage:         true,
		UpdatedAt:         timestamppb.New(row.Updated),
	}, nil
}

func toProtoPlugin(p pluginhost.Plugin) *agencyv1.Plugin {
	return &agencyv1.Plugin{
		Id:               p.ID,
		Name:             p.Name,
		Version:          p.Version,
		Description:      p.Description,
		Enabled:          p.Enabled,
		Runtime:          p.Runtime,
		Entry:            p.Entry,
		Checksum:         p.Checksum,
		ActorId:          strconv.FormatInt(p.ActorID, 10),
		ActorUsername:    p.ActorUsername,
		ActorName:        p.ActorName,
		ManifestJson:     string(p.ManifestRaw),
		SettingsSurfaces: sortedSurfaces(p),
		CreatedAt:        timestamppb.New(p.CreatedAt),
		UpdatedAt:        timestamppb.New(p.UpdatedAt),
	}
}

func sortedSurfaces(p pluginhost.Plugin) []string {
	out := p.Manifest.Surfaces()
	sort.Strings(out)
	return out
}

func toProtoLog(row pluginhost.LogLine) *agencyv1.PluginLog {
	out := &agencyv1.PluginLog{
		Id:           row.ID,
		At:           timestamppb.New(row.At),
		Level:        row.Level,
		Kind:         row.Kind,
		InvocationId: row.InvocationID,
		Message:      row.Message,
		DataJson:     row.Data,
	}
	for _, surface := range row.Surfaces {
		out.Surfaces = append(out.Surfaces, &agencyv1.PluginLogSurface{
			Scope:    surface.Scope,
			TargetId: surface.TargetID,
		})
	}
	return out
}

func toProtoSettingErrors(errs []pluginhost.SettingError) []*agencyv1.PluginSettingError {
	out := make([]*agencyv1.PluginSettingError, len(errs))
	for i, item := range errs {
		out[i] = &agencyv1.PluginSettingError{Path: item.Path, ErrorText: item.ErrorText}
	}
	return out
}

func toProtoPreview(preview pluginhost.Preview) *agencyv1.PluginPreview {
	p := preview.Plugin
	manifest := p.Manifest
	return &agencyv1.PluginPreview{
		Id:               p.ID,
		Name:             p.Name,
		Version:          p.Version,
		Description:      p.Description,
		Runtime:          p.Runtime,
		Entry:            p.Entry,
		Checksum:         p.Checksum,
		ManifestJson:     string(p.ManifestRaw),
		Handlers:         manifest.Handlers,
		SettingsSurfaces: sortedSurfaces(p),
		Installed:        preview.Installed,
		InstalledVersion: preview.InstalledVersion,
		Warnings:         preview.Warnings,
		Compatible:       preview.Compatible,
	}
}

func mapPluginError(err error) error {
	if errors.Is(err, pluginhost.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, errors.New("plugin not found"))
	}

	var rejected *pluginhost.SettingsRejected
	if errors.As(err, &rejected) {
		connectErr := connect.NewError(connect.CodeInvalidArgument, rejected)
		detail, detailErr := connect.NewErrorDetail(
			&agencyv1.PluginSettingErrors{Errors: toProtoSettingErrors(rejected.Errors)},
		)
		if detailErr == nil {
			connectErr.AddDetail(detail)
		}
		return connectErr
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}
	return connect.NewError(connect.CodeInternal, err)
}
