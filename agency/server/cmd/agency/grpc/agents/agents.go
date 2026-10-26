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

package agentsgrpc

import (
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"errors"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/agentlogs"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/limits"
	"cherrypi.vladfaust.com/agency/internal/prompts"
	"cherrypi.vladfaust.com/agency/internal/scheduler"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AgentConnector interface {
	Connection(agentID ids.AgentID) (*tunnel.Connection, error)
}

type AgentServer struct {
	Agents *agents.Registry

	Prompts *prompts.Service

	Limits *limits.Service

	Scheduler *scheduler.Scheduler

	Attachments *attachments.Service

	Logs *agentlogs.Service

	Tasks *tasks.Service

	Hub AgentConnector

	Accounts *accounts.Service
}

const (
	defaultAgentLogLimit = 200
	maxAgentLogLimit     = 1000
)

func (s *AgentServer) ListAgentLogs(
	ctx context.Context,
	req *agencyv1.ListAgentLogsRequest,
) (*agencyv1.ListAgentLogsResponse, error) {
	if s.Logs == nil {
		return &agencyv1.ListAgentLogsResponse{}, nil
	}
	if _, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		return nil, mapError(err)
	}

	limit := req.Limit
	switch {
	case limit <= 0:
		limit = defaultAgentLogLimit
	case limit > maxAgentLogLimit:
		limit = maxAgentLogLimit
	}

	rows, err := s.Logs.List(
		ctx,
		actorid.ParseAgent(req.AgentId),
		agentLogLevel(req.MinLevel),
		req.BeforeId,
		int64(limit),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := make([]*agencyv1.AgentLog, len(rows))
	for i, row := range rows {
		out[i] = toProtoAgentLog(row)
	}
	return &agencyv1.ListAgentLogsResponse{Logs: out}, nil
}

func agentLogLevel(level agencyv1.AgentLogLevel) int {
	if level == agencyv1.AgentLogLevel_AGENT_LOG_LEVEL_UNSPECIFIED {
		return agentlogs.LevelDebug
	}
	return int(level) - 1
}

func toProtoAgentLog(row agentlogs.Row) *agencyv1.AgentLog {
	return &agencyv1.AgentLog{
		Id:        row.ID,
		AgentId:   actorid.Format(row.AgentID),
		At:        timestamppb.New(time.UnixMilli(row.At)),
		Level:     agencyv1.AgentLogLevel(row.Level + 1),
		Message:   row.Message,
		TaskId:    row.TaskID,
		SessionId: row.SessionID,
		AttrsJson: row.AttrsJSON,
	}
}

func (s *AgentServer) ListAgents(
	ctx context.Context,
	_ *agencyv1.ListAgentsRequest,
) (*agencyv1.ListAgentsResponse, error) {
	list, err := s.Agents.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pictures map[ids.AgentID]attachments.Attachment
	if s.Attachments != nil {
		pictures, _ = s.Attachments.AgentPictures(ctx)
	}
	out := make([]*agencyv1.Agent, len(list))
	for i, a := range list {
		out[i] = toProto(a)
		if picture, ok := pictures[a.ID]; ok {
			out[i].Picture = attachments.ToProto(picture)
		}
	}
	return &agencyv1.ListAgentsResponse{Agents: out}, nil
}

func (s *AgentServer) GetAgent(
	ctx context.Context,
	req *agencyv1.GetAgentRequest,
) (*agencyv1.GetAgentResponse, error) {
	a, err := s.Agents.Get(ctx, actorid.ParseAgent(req.Id))
	if err != nil {
		return nil, mapError(err)
	}
	out := toProto(a)
	s.enrichPicture(ctx, a.ID, out)
	return &agencyv1.GetAgentResponse{Agent: out}, nil
}

func (s *AgentServer) enrichPicture(ctx context.Context, agentID ids.AgentID, out *agencyv1.Agent) {
	if s.Attachments == nil {
		return
	}
	if picture, ok := s.Attachments.AgentPicture(ctx, agentID); ok {
		out.Picture = attachments.ToProto(picture)
	}
}

func (s *AgentServer) requireAgentAccess(ctx context.Context, agentID ids.AgentID) error {
	if auth.IsAdmin(ctx) {
		return nil
	}
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("not authenticated"))
	}
	owner, err := s.Agents.IsOwner(ctx, agentID, identity.ActorID)
	if err != nil {
		return mapError(err)
	}
	if !owner {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("agent owner or administrator required"),
		)
	}
	return nil
}

func (s *AgentServer) requireCreateAgent(ctx context.Context) (ids.ActorID, error) {
	identity, ok := auth.IdentityFrom(ctx)
	if !ok {
		return 0, connect.NewError(connect.CodeUnauthenticated, errors.New("not authenticated"))
	}
	if identity.IsAdmin() {
		return identity.ActorID, nil
	}
	if s.Accounts == nil {
		return 0, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("creating agents is admin-only"),
		)
	}
	settings, err := s.Accounts.GetAuthSettings(ctx)
	if err != nil {
		return 0, connect.NewError(connect.CodeInternal, err)
	}
	if !settings.MembersCanCreateAgents {
		return 0, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("creating agents is disabled for members"),
		)
	}
	return identity.ActorID, nil
}

func (s *AgentServer) CreateAgent(
	ctx context.Context,
	req *agencyv1.CreateAgentRequest,
) (*agencyv1.CreateAgentResponse, error) {
	creatorActorID, err := s.requireCreateAgent(ctx)
	if err != nil {
		return nil, err
	}
	a, password, err := s.Agents.CreateOwned(
		ctx, req.Username, req.DisplayName, req.Description, creatorActorID,
	)
	if err != nil {
		return nil, mapError(err)
	}
	if s.Prompts != nil && req.Prompt != "" {
		if _, err := s.Prompts.Set(ctx, prompts.ScopeAgent, a.ID, "", req.Prompt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	token := ""
	if s.Accounts != nil {
		plaintext, _, err := s.Accounts.CreatePAT(ctx, a.ActorID, "agent", nil)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		token = plaintext
	}
	return &agencyv1.CreateAgentResponse{
		Agent:              toProto(a),
		CredentialUsername: a.Username,
		CredentialPassword: password,
		Token:              token,
	}, nil
}

func (s *AgentServer) UpdateAgent(
	ctx context.Context,
	req *agencyv1.UpdateAgentRequest,
) (*agencyv1.UpdateAgentResponse, error) {
	if err := s.requireAgentAccess(ctx, actorid.ParseAgent(req.Id)); err != nil {
		return nil, err
	}
	a, err := s.Agents.Update(ctx, actorid.ParseAgent(req.Id), req.DisplayName, req.Description)
	if err != nil {
		return nil, mapError(err)
	}
	out := toProto(a)
	s.enrichPicture(ctx, a.ID, out)
	return &agencyv1.UpdateAgentResponse{Agent: out}, nil
}

func (s *AgentServer) UpdateAgentRole(
	ctx context.Context,
	req *agencyv1.UpdateAgentRoleRequest,
) (*agencyv1.UpdateAgentRoleResponse, error) {
	if !auth.IsAdmin(ctx) {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("only a human administrator may change an agent's role"),
		)
	}
	a, err := s.Agents.SetRole(ctx, actorid.ParseAgent(req.Id), req.Role)
	if err != nil {
		return nil, mapError(err)
	}
	out := toProto(a)
	s.enrichPicture(ctx, a.ID, out)
	return &agencyv1.UpdateAgentRoleResponse{Agent: out}, nil
}

func (s *AgentServer) GetAgentDefaults(
	ctx context.Context,
	req *agencyv1.GetAgentDefaultsRequest,
) (*agencyv1.GetAgentDefaultsResponse, error) {
	defaults, err := s.Agents.Defaults(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.GetAgentDefaultsResponse{Defaults: toProtoDefaults(defaults)}, nil
}

func (s *AgentServer) UpdateAgentDefaults(
	ctx context.Context,
	req *agencyv1.UpdateAgentDefaultsRequest,
) (*agencyv1.UpdateAgentDefaultsResponse, error) {
	agentID := actorid.ParseAgent(req.AgentId)
	if err := s.requireAgentAccess(ctx, agentID); err != nil {
		return nil, err
	}
	defaults, err := s.Agents.SetDefaults(ctx, agentID, agents.Defaults{
		ModelProvider: req.GetDefaults().GetModelProvider(),
		ModelID:       req.GetDefaults().GetModelId(),
		ModelName:     req.GetDefaults().GetModelName(),
		ThinkingLevel: req.GetDefaults().GetThinkingLevel(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.UpdateAgentDefaultsResponse{Defaults: toProtoDefaults(defaults)}, nil
}

func toProtoDefaults(defaults agents.Defaults) *agencyv1.AgentDefaults {
	return &agencyv1.AgentDefaults{
		ModelProvider: defaults.ModelProvider,
		ModelId:       defaults.ModelID,
		ModelName:     defaults.ModelName,
		ThinkingLevel: defaults.ThinkingLevel,
	}
}

func (s *AgentServer) DeleteAgent(
	ctx context.Context,
	req *agencyv1.DeleteAgentRequest,
) (*agencyv1.DeleteAgentResponse, error) {
	if err := s.requireAgentAccess(ctx, actorid.ParseAgent(req.Id)); err != nil {
		return nil, err
	}
	if err := s.Agents.Delete(ctx, actorid.ParseAgent(req.Id)); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.DeleteAgentResponse{}, nil
}

func (s *AgentServer) GetAgentTaskLimits(
	ctx context.Context,
	req *agencyv1.GetAgentTaskLimitsRequest,
) (*agencyv1.GetAgentTaskLimitsResponse, error) {
	if _, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		return nil, mapError(err)
	}
	lim, err := s.Limits.Get(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.GetAgentTaskLimitsResponse{Limits: toProtoLimits(lim)}, nil
}

func (s *AgentServer) UpdateAgentTaskLimits(
	ctx context.Context,
	req *agencyv1.UpdateAgentTaskLimitsRequest,
) (*agencyv1.UpdateAgentTaskLimitsResponse, error) {
	if _, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		return nil, mapError(err)
	}
	lim, err := s.Limits.Update(ctx, actorid.ParseAgent(req.AgentId), fromProtoLimits(req.Limits))
	if err != nil {
		if errors.Is(err, limits.ErrInvalidLimit) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.Scheduler != nil {
		s.Scheduler.Wake()
	}
	if a, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId)); err == nil {
		s.Agents.Record(ctx, a.ActorID, audit.KindLimits, "", "", "",
			"Updated task concurrency limits", nil)
	}
	return &agencyv1.UpdateAgentTaskLimitsResponse{Limits: toProtoLimits(lim)}, nil
}

func (s *AgentServer) ListAgentCredentials(
	ctx context.Context,
	req *agencyv1.ListAgentCredentialsRequest,
) (*agencyv1.ListAgentCredentialsResponse, error) {
	creds, err := s.Agents.ListCredentials(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*agencyv1.AgentCredential, len(creds))
	for i, c := range creds {
		out[i] = &agencyv1.AgentCredential{
			Username:   c.Username,
			CreatedAt:  timestamppb.New(c.CreatedAt),
			LastUsedAt: timestampOrNil(c.LastUsedAt),
		}
	}
	return &agencyv1.ListAgentCredentialsResponse{Credentials: out}, nil
}

func (s *AgentServer) CreateAgentCredential(
	ctx context.Context,
	req *agencyv1.CreateAgentCredentialRequest,
) (*agencyv1.CreateAgentCredentialResponse, error) {
	c, password, err := s.Agents.CreateCredential(
		ctx,
		actorid.ParseAgent(req.AgentId),
		req.Username,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.CreateAgentCredentialResponse{
		Credential: &agencyv1.AgentCredential{
			Username:   c.Username,
			CreatedAt:  timestamppb.New(c.CreatedAt),
			LastUsedAt: timestampOrNil(c.LastUsedAt),
		},
		Password: password,
	}, nil
}

func (s *AgentServer) RevokeAgentCredential(
	ctx context.Context,
	req *agencyv1.RevokeAgentCredentialRequest,
) (*agencyv1.RevokeAgentCredentialResponse, error) {
	if err := s.Agents.RevokeCredential(
		ctx,
		actorid.ParseAgent(req.AgentId),
		req.Username,
	); err != nil {
		return nil, mapError(err)
	}
	return &agencyv1.RevokeAgentCredentialResponse{}, nil
}

func (s *AgentServer) ListAgentTokens(
	ctx context.Context,
	req *agencyv1.ListAgentTokensRequest,
) (*agencyv1.ListAgentTokensResponse, error) {
	if err := s.requireAgentAccess(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		return nil, err
	}
	if s.Accounts == nil {
		return &agencyv1.ListAgentTokensResponse{}, nil
	}
	a, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	pats, err := s.Accounts.ListPATs(ctx, a.ActorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*agencyv1.AgentToken, len(pats))
	for i, p := range pats {
		out[i] = toProtoAgentToken(p)
	}
	return &agencyv1.ListAgentTokensResponse{Tokens: out}, nil
}

func (s *AgentServer) CreateAgentToken(
	ctx context.Context,
	req *agencyv1.CreateAgentTokenRequest,
) (*agencyv1.CreateAgentTokenResponse, error) {
	if err := s.requireAgentAccess(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		return nil, err
	}
	if s.Accounts == nil {
		return nil, connect.NewError(
			connect.CodeUnavailable,
			errors.New("token management is unavailable"),
		)
	}
	a, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		expiresAt = &t
	}
	plaintext, pat, err := s.Accounts.CreatePAT(ctx, a.ActorID, req.Name, expiresAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.CreateAgentTokenResponse{
		Token:     toProtoAgentToken(pat),
		Plaintext: plaintext,
	}, nil
}

func (s *AgentServer) RevokeAgentToken(
	ctx context.Context,
	req *agencyv1.RevokeAgentTokenRequest,
) (*agencyv1.RevokeAgentTokenResponse, error) {
	if err := s.requireAgentAccess(ctx, actorid.ParseAgent(req.AgentId)); err != nil {
		return nil, err
	}
	if s.Accounts == nil {
		return nil, connect.NewError(
			connect.CodeUnavailable,
			errors.New("token management is unavailable"),
		)
	}
	a, err := s.Agents.Get(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, mapError(err)
	}
	if err := s.Accounts.RevokePAT(ctx, a.ActorID, req.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.RevokeAgentTokenResponse{}, nil
}

func toProtoAgentToken(p accounts.PAT) *agencyv1.AgentToken {
	out := &agencyv1.AgentToken{
		Id:        p.ID,
		Name:      p.Name,
		Prefix:    p.Prefix,
		CreatedAt: timestamppb.New(p.CreatedAt),
	}
	if p.LastUsedAt != nil {
		out.LastUsedAt = timestamppb.New(*p.LastUsedAt)
	}
	if p.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*p.ExpiresAt)
	}
	return out
}

func toProto(a agents.Agent) *agencyv1.Agent {
	return &agencyv1.Agent{
		Id:              actorid.Format(a.ID),
		ActorId:         actorid.Format(a.ActorID),
		Username:        a.Username,
		DisplayName:     a.DisplayName,
		Description:     a.Description,
		Role:            a.Role,
		Status:          agencyv1.AgentStatus(a.Status),
		Version:         a.Version,
		ProtocolVersion: a.ProtocolVersion,
		VersionStatus:   agencyv1.AgentVersionStatus(a.VersionStatus),
		RemoteAddr:      a.RemoteAddr,
		CreatedAt:       timestamppb.New(a.CreatedAt),
		UpdatedAt:       timestamppb.New(a.UpdatedAt),
		LastSeenAt:      timestampOrNil(a.LastSeenAt),
	}
}

func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoLimits(l limits.Limits) *agencyv1.AgentTaskLimits {
	out := &agencyv1.AgentTaskLimits{
		MaxConcurrent:           int32(l.MaxConcurrent),
		MaxConcurrentPerProject: int32(l.MaxConcurrentPerProject),
	}
	for tagName, cap := range l.Tags {
		out.TagLimits = append(out.TagLimits, &agencyv1.TagLimit{
			TagName:       tagName,
			MaxConcurrent: int32(cap),
		})
	}
	for projectID, cap := range l.Projects {
		out.ProjectLimits = append(out.ProjectLimits, &agencyv1.ProjectLimit{
			ProjectId:     projectID,
			MaxConcurrent: int32(cap),
		})
	}
	for projectID, byTag := range l.ProjectTags {
		for tagID, cap := range byTag {
			out.ProjectTagLimits = append(out.ProjectTagLimits, &agencyv1.ProjectTagLimit{
				ProjectId:     projectID,
				TagId:         tagID,
				MaxConcurrent: int32(cap),
			})
		}
	}
	return out
}

func fromProtoLimits(in *agencyv1.AgentTaskLimits) limits.Limits {
	out := limits.Limits{
		Tags:        map[string]int{},
		Projects:    map[string]int{},
		ProjectTags: map[string]map[string]int{},
	}
	if in == nil {
		return out
	}
	out.MaxConcurrent = int(in.MaxConcurrent)
	out.MaxConcurrentPerProject = int(in.MaxConcurrentPerProject)
	for _, l := range in.TagLimits {
		out.Tags[l.TagName] = int(l.MaxConcurrent)
	}
	for _, l := range in.ProjectLimits {
		out.Projects[l.ProjectId] = int(l.MaxConcurrent)
	}
	for _, l := range in.ProjectTagLimits {
		if out.ProjectTags[l.ProjectId] == nil {
			out.ProjectTags[l.ProjectId] = map[string]int{}
		}
		out.ProjectTags[l.ProjectId][l.TagId] = int(l.MaxConcurrent)
	}
	return out
}

func mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, agents.ErrAgentMissing), errors.Is(err, agents.ErrCredentialMissing):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, agents.ErrAgentExists), errors.Is(err, agents.ErrCredentialExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, agents.ErrInvalidUsername), errors.Is(err, agents.ErrInvalidDefaults),
		errors.Is(err, agents.ErrInvalidRole):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
