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
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/store"
)

const ProxyPrefix = "/api/rest/agent-mcp"

type ResolvedServer struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	ProxyPath        string     `json:"proxyPath"`
	ConnectTimeoutMs int64      `json:"connectTimeoutMs"`
	RequestTimeoutMs int64      `json:"requestTimeoutMs"`
	Tools            ToolFilter `json:"tools"`
	DirectTools      bool       `json:"directTools"`
	Owner            string     `json:"owner"`
}

type ResolvedSkill struct {
	Name                   string      `json:"name"`
	Description            string      `json:"description"`
	Body                   string      `json:"body"`
	Files                  []SkillFile `json:"files"`
	DisableModelInvocation bool        `json:"disableModelInvocation"`
	Owner                  string      `json:"owner"`
}

type Bundle struct {
	MCP    MCPBundle    `json:"mcp"`
	Skills SkillsBundle `json:"skills"`
}

type MCPBundle struct {
	Servers []ResolvedServer `json:"servers"`

	Deny []string `json:"deny"`
}

type SkillsBundle struct {
	Items []ResolvedSkill `json:"items"`
	Deny  []string        `json:"deny"`
}

type Resolution struct {
	Servers    []ResolvedServer
	Skills     []ResolvedSkill
	DenyMCP    []string
	DenySkills []string

	Applied []Applied
}

type Applied struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Owner  Scope  `json:"owner"`
	Source Scope  `json:"source"`
}

func Scopes(projectID string, teamIDs, groupIDs []string, agentActorID string) []Scope {
	out := []Scope{{Kind: KindOrg}}
	if projectID != "" {
		out = append(out, Scope{Kind: KindProject, ID: projectID})
	}
	for _, team := range sortedIDs(teamIDs) {
		out = append(out, Scope{Kind: KindTeam, ID: team})
	}
	for _, group := range sortedIDs(groupIDs) {
		out = append(out, Scope{Kind: KindGroup, ID: group})
	}
	if agentActorID != "" {
		out = append(out, Scope{Kind: KindAgent, ID: agentActorID})
	}
	return out
}

func sortedIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			out = append(out, strings.TrimSpace(id))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, leftErr := strconv.ParseInt(out[i], 10, 64)
		right, rightErr := strconv.ParseInt(out[j], 10, 64)
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return out[i] < out[j]
	})
	return out
}

func (s *Service) Resolve(ctx context.Context, caller Caller) (Resolution, error) {
	servers, err := s.store.ListMcpServers(ctx)
	if err != nil {
		return Resolution{}, err
	}
	skills, err := s.store.ListSkillEntries(ctx)
	if err != nil {
		return Resolution{}, err
	}
	mcpInstalls, err := s.store.ListMcpInstalls(ctx)
	if err != nil {
		return Resolution{}, err
	}
	skillInstalls, err := s.store.ListSkillInstalls(ctx)
	if err != nil {
		return Resolution{}, err
	}
	scopes := Scopes(caller.ProjectID, caller.TeamIDs, caller.GroupIDs, caller.AgentActorID)
	applicable := map[string]Scope{}
	for _, scope := range scopes {
		applicable[scope.String()] = scope
	}

	resolution := Resolution{}
	for _, row := range servers {
		source, denied := applicableInstall(row.ID, applicable, mcpInstalls)
		if source == nil && !denied {
			continue
		}
		if denied || row.Enabled == 0 {
			resolution.DenyMCP = append(resolution.DenyMCP, row.Name)
			continue
		}
		resolution.Servers = append(resolution.Servers, resolvedServer(row))
		resolution.Applied = append(resolution.Applied, Applied{
			Kind:   "mcp",
			Name:   row.Name,
			Owner:  Scope{Kind: row.OwnerKind, ID: row.OwnerID},
			Source: *source,
		})
	}
	sort.Strings(resolution.DenyMCP)

	for _, row := range skills {

		if row.SourceMissing != 0 {
			continue
		}
		source, denied := applicableSkillInstall(row.ID, applicable, skillInstalls)
		if source == nil && !denied {
			continue
		}
		if denied || row.Enabled == 0 {
			resolution.DenySkills = append(resolution.DenySkills, row.Name)
			continue
		}
		resolution.Skills = append(resolution.Skills, resolvedSkill(row, *source))
		resolution.Applied = append(resolution.Applied, Applied{
			Kind:   "skill",
			Name:   row.Name,
			Owner:  Scope{Kind: row.OwnerKind, ID: row.OwnerID},
			Source: *source,
		})
	}
	sort.Strings(resolution.DenySkills)
	return resolution, nil
}

func scopeRank(kind string) int {
	for i, candidate := range Kinds {
		if candidate == kind {
			return i
		}
	}
	return -1
}

func applicableInstall(
	entryID int64,
	applicable map[string]Scope,
	rows []store.McpInstall,
) (*Scope, bool) {
	var best *store.McpInstall
	var bestScope Scope
	for _, row := range rows {
		if row.EntryID != entryID {
			continue
		}
		scope := Scope{Kind: row.ScopeKind, ID: row.ScopeID}
		if _, ok := applicable[scope.String()]; !ok {
			continue
		}
		if best != nil && scopeRank(row.ScopeKind) <= scopeRank(best.ScopeKind) {
			continue
		}
		candidate := row
		best = &candidate
		bestScope = scope
	}
	if best == nil {
		return nil, false
	}
	if best.Enabled == 0 {
		return nil, true
	}
	return &bestScope, false
}

func applicableSkillInstall(
	entryID int64,
	applicable map[string]Scope,
	rows []store.SkillInstall,
) (*Scope, bool) {
	var best *store.SkillInstall
	var bestScope Scope
	for _, row := range rows {
		if row.EntryID != entryID {
			continue
		}
		scope := Scope{Kind: row.ScopeKind, ID: row.ScopeID}
		if _, ok := applicable[scope.String()]; !ok {
			continue
		}
		if best != nil && scopeRank(row.ScopeKind) <= scopeRank(best.ScopeKind) {
			continue
		}
		candidate := row
		best = &candidate
		bestScope = scope
	}
	if best == nil {
		return nil, false
	}
	if best.Enabled == 0 {
		return nil, true
	}
	return &bestScope, false
}

func (s *Service) Bundle(ctx context.Context, caller Caller) (Bundle, error) {
	resolution, err := s.Resolve(ctx, caller)
	if err != nil {
		return Bundle{}, err
	}
	out := Bundle{
		MCP: MCPBundle{
			Servers: resolution.Servers,
			Deny:    resolution.DenyMCP,
		},
		Skills: SkillsBundle{
			Items: resolution.Skills,
			Deny:  resolution.DenySkills,
		},
	}

	if out.MCP.Servers == nil {
		out.MCP.Servers = []ResolvedServer{}
	}
	if out.MCP.Deny == nil {
		out.MCP.Deny = []string{}
	}
	if out.Skills.Items == nil {
		out.Skills.Items = []ResolvedSkill{}
	}
	if out.Skills.Deny == nil {
		out.Skills.Deny = []string{}
	}
	return out, nil
}

func resolvedServer(row store.McpServer) ResolvedServer {
	out := ResolvedServer{
		Name:             row.Name,
		Description:      row.Description,
		ProxyPath:        ProxyPrefix + "/" + row.Name,
		ConnectTimeoutMs: row.ConnectTimeoutMs,
		RequestTimeoutMs: row.RequestTimeoutMs,
		DirectTools:      row.DirectTools != 0,
		Owner:            (Scope{Kind: row.OwnerKind, ID: row.OwnerID}).String(),
	}
	if err := json.Unmarshal([]byte(row.ToolsJson), &out.Tools); err != nil {
		out.Tools = ToolFilter{}
	}
	if out.Tools.Include == nil {
		out.Tools.Include = []string{}
	}
	if out.Tools.Exclude == nil {
		out.Tools.Exclude = []string{}
	}
	return out
}

func resolvedSkill(row store.Skill, source Scope) ResolvedSkill {
	out := ResolvedSkill{
		Name:                   row.Name,
		Description:            row.Description,
		Body:                   row.Body,
		DisableModelInvocation: row.DisableModelInvocation != 0,
		Owner:                  (Scope{Kind: row.OwnerKind, ID: row.OwnerID}).String(),
	}
	if err := json.Unmarshal([]byte(row.FilesJson), &out.Files); err != nil {
		out.Files = nil
	}
	_ = source
	return out
}

type MissingEnvError struct {
	Header string
	Env    string
}

func (e MissingEnvError) Error() string {
	return fmt.Sprintf(
		"the %q header references the unset agency environment variable %q",
		e.Header, e.Env,
	)
}

func (s *Service) openValues(raw string) (map[string]string, error) {
	out := map[string]string{}
	stored := map[string]Value{}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return out, nil
	}
	for key, value := range stored {
		switch {
		case value.Secret != "":
			sealed, err := base64Decode(value.Secret)
			if err != nil {
				return nil, err
			}
			plain, err := s.seal.Open(sealed)
			if err != nil {
				return nil, err
			}
			out[key] = plain
		case value.Env != "":
			resolved, ok := os.LookupEnv(value.Env)
			if !ok {
				return nil, MissingEnvError{Header: key, Env: value.Env}
			}
			out[key] = resolved
		default:
			out[key] = value.Value
		}
	}
	return out, nil
}

type Report struct {
	Skills  []ReportedSkill  `json:"skills"`
	Servers []ReportedServer `json:"servers"`

	Tools []ReportedTool `json:"tools,omitempty"`
}

type ReportedTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type ReportedSkill struct {
	Name                   string `json:"name"`
	Description            string `json:"description"`
	Origin                 string `json:"origin"`
	Owner                  string `json:"owner"`
	DisableModelInvocation bool   `json:"disableModelInvocation"`
}

type ReportedServer struct {
	Name      string `json:"name"`
	Origin    string `json:"origin"`
	Owner     string `json:"owner"`
	Transport string `json:"transport"`
	Connected bool   `json:"connected"`
	ToolCount int    `json:"toolCount"`
	Error     string `json:"error"`
}

type TaskReport struct {
	TaskID         string `json:"taskId"`
	Report         Report `json:"report"`
	ConfigRevision string `json:"configRevision"`
	ReportedAt     int64  `json:"reportedAt"`
	Recorded       bool   `json:"recorded"`
}

func (s *Service) PutReport(
	ctx context.Context,
	taskID, revision string,
	report Report,
) error {
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("%w: a report needs a task", ErrInvalid)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	return s.store.PutTaskIntegrationReport(ctx, store.PutTaskIntegrationReportParams{
		TaskID:         taskID,
		PayloadJson:    string(raw),
		ConfigRevision: revision,
		ReportedAt:     s.now().UnixMilli(),
	})
}

func (s *Service) TaskReport(ctx context.Context, taskID string) (TaskReport, error) {
	row, err := s.store.GetTaskIntegrationReport(ctx, taskID)
	if err != nil {
		if IsNoRows(err) {
			return TaskReport{TaskID: taskID}, nil
		}
		return TaskReport{}, err
	}
	out := TaskReport{
		TaskID:         row.TaskID,
		ConfigRevision: row.ConfigRevision,
		ReportedAt:     row.ReportedAt,
		Recorded:       true,
	}
	if err := json.Unmarshal([]byte(row.PayloadJson), &out.Report); err != nil {
		return TaskReport{}, err
	}
	return out, nil
}

func (s *Service) ResolveBundle(
	ctx context.Context,
	agentActorID, projectID string,
	teamIDs, groupIDs []string,
) (any, any, error) {
	caller := Caller{
		AgentActorID: agentActorID,
		ProjectID:    projectID,
		TeamIDs:      teamIDs,
		GroupIDs:     groupIDs,
	}
	actorID, err := strconv.ParseInt(agentActorID, 10, 64)
	if err == nil {
		if lister, ok := s.store.(groupLister); ok {
			groups, err := lister.ListAgentGroupIDsForAgentActor(ctx, actorID)
			if err != nil {
				return nil, nil, err
			}
			for _, group := range groups {
				caller.GroupIDs = append(caller.GroupIDs, strconv.FormatInt(group, 10))
			}
		}
	}
	bundle, err := s.Bundle(ctx, caller)
	if err != nil {
		return nil, nil, err
	}
	return bundle.MCP, bundle.Skills, nil
}

type groupLister interface {
	ListAgentGroupIDsForAgentActor(ctx context.Context, actorID int64) ([]int64, error)
}

func (s *Service) ApplicableServer(
	ctx context.Context,
	name string,
	caller Caller,
) (store.McpServer, bool, error) {
	row, err := s.store.GetMcpServerByName(ctx, name)
	if err != nil {
		return store.McpServer{}, false, mapStoreError(err)
	}
	resolution, err := s.Resolve(ctx, caller)
	if err != nil {
		return store.McpServer{}, false, err
	}
	for _, server := range resolution.Servers {
		if server.Name == name {
			return row, true, nil
		}
	}
	return row, false, nil
}
