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

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"github.com/sky-valley/pi/coding"

	"cherrypi.vladfaust.com/agent/internal/mcp"
)

type AgencyMCP struct {
	Servers []mcp.Server `json:"servers"`
	Deny    []string     `json:"deny"`
}

type AgencySkills struct {
	Items []AgencySkill `json:"items"`
	Deny  []string      `json:"deny"`
}

type AgencySkill struct {
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	Body                   string          `json:"body"`
	Files                  []SkillFileBody `json:"files"`
	DisableModelInvocation bool            `json:"disableModelInvocation"`
	Owner                  string          `json:"owner"`
}

type SkillFileBody struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type AgencyIntegrations struct {
	Revision string
	MCP      AgencyMCP
	Skills   AgencySkills
}

type IntegrationReport struct {
	Skills  []ReportedSkill  `json:"skills"`
	Servers []ReportedServer `json:"servers"`

	Tools []ReportedTool `json:"tools"`
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

func (s *Session) integrations(ctx context.Context) AgencyIntegrations {
	if s.integrationsFn == nil {
		return AgencyIntegrations{}
	}
	return s.integrationsFn(ctx)
}

// Nothing here can fail the run: a server that cannot connect is a log line and

func (s *Session) startIntegrations(
	ctx context.Context,
	declared AgencyIntegrations,
) ([]agent.AgentTool, string, IntegrationReport) {
	report := IntegrationReport{
		Skills:  []ReportedSkill{},
		Servers: []ReportedServer{},
		Tools:   []ReportedTool{},
	}

	hostSkills, _ := hostSkills(s.Workspace)
	denied := map[string]bool{}
	for _, name := range declared.Skills.Deny {
		denied[name] = true
	}
	for _, skill := range hostSkills {
		if denied[skill.Name] {
			continue
		}
		report.Skills = append(report.Skills, ReportedSkill{
			Name:                   skill.Name,
			Description:            skill.Description,
			Origin:                 "host",
			DisableModelInvocation: skill.DisableModelInvocation,
		})
	}

	materialized := make([]coding.Skill, 0, len(declared.Skills.Items))
	for _, skill := range declared.Skills.Items {
		path, err := s.materializeSkill(skill)
		if err != nil {
			slog.Warn("agency skill", "session_id", s.ID, "skill", skill.Name, "error", err)
			continue
		}
		materialized = append(materialized, coding.Skill{
			Name:                   skill.Name,
			Description:            skill.Description,
			FilePath:               path,
			BaseDir:                filepath.Dir(path),
			DisableModelInvocation: skill.DisableModelInvocation,
		})
		report.Skills = append(report.Skills, ReportedSkill{
			Name:                   skill.Name,
			Description:            skill.Description,
			Origin:                 "agency",
			Owner:                  skill.Owner,
			DisableModelInvocation: skill.DisableModelInvocation,
		})
	}
	sort.SliceStable(report.Skills, func(i, j int) bool {
		return report.Skills[i].Name < report.Skills[j].Name
	})
	promptBlock := renderAgencySkills(materialized)

	declaredServers := make([]mcp.Server, 0, len(declared.MCP.Servers))
	for _, server := range declared.MCP.Servers {

		server.Transport = mcp.TransportAgency
		server.AgencyURL = s.cfg.AgencyURL
		server.AgencyToken = s.cfg.AgentToken
		server.TaskID = s.agencyTaskID
		declaredServers = append(declaredServers, server)
	}
	servers := mcp.Merge(mcp.HostServers(), declaredServers, declared.MCP.Deny)
	tools := make([]agent.AgentTool, 0, len(servers)*4)
	for _, server := range servers {
		entry := ReportedServer{
			Name:      server.Name,
			Origin:    originOf(server),
			Owner:     server.Owner,
			Transport: server.Transport,
		}
		resolved, ok := server.Resolve()
		if !ok {
			entry.Error = "an environment reference in this server's values has no value on this host"
			slog.Warn("mcp server skipped",
				"session_id", s.ID,
				"server", server.Name,
				"error", entry.Error,
			)
			report.Servers = append(report.Servers, entry)
			continue
		}
		server = resolved
		conn, err := mcp.Connect(ctx, server)
		if err != nil {
			entry.Error = err.Error()
			slog.Warn("mcp server",
				"session_id", s.ID,
				"server", server.Name,
				"error", err,
			)
			report.Servers = append(report.Servers, entry)
			continue
		}
		s.mu.Lock()
		s.mcpConns = append(s.mcpConns, conn)
		s.mu.Unlock()
		entry.Connected = true
		entry.ToolCount = len(conn.Tools())
		report.Servers = append(report.Servers, entry)
		for _, tool := range conn.Tools() {
			tools = append(tools, mcpTool(conn, tool))
		}
	}
	sort.SliceStable(report.Servers, func(i, j int) bool {
		return report.Servers[i].Name < report.Servers[j].Name
	})
	return tools, promptBlock, report
}

func originOf(server mcp.Server) string {
	if server.Origin == "" {
		return "agency"
	}
	return server.Origin
}

func mcpTool(conn *mcp.Conn, tool mcp.Tool) agent.AgentTool {
	name := tool.Name
	server := conn.Server().Name
	description := tool.Description
	if description == "" {
		description = fmt.Sprintf("%s tool from the %s MCP server.", name, server)
	}
	return agent.AgentTool{
		Name:        mcpToolName(server, name),
		Label:       name,
		Description: description,
		Parameters:  mcpSchema(tool.Schema),
		Execute: func(
			ctx context.Context,
			_ string,
			params map[string]any,
			_ agent.ToolUpdateFunc,
		) (agent.AgentToolResult, error) {
			result, isError, err := conn.Call(ctx, name, params)
			if err != nil {
				return errorText(fmt.Sprintf("mcp %s/%s: %v", server, name, err)), nil
			}
			if isError {
				return errorText(result), nil
			}
			return text(result), nil
		},
	}
}

// mcpToolName renders the model-facing tool name: `mcp__<server>__<tool>`,

func mcpToolName(server, tool string) string {
	raw := "mcp__" + server + "__" + tool
	var out strings.Builder
	for _, char := range raw {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '_', char == '-':
			out.WriteRune(char)
		default:
			out.WriteRune('_')
		}
	}
	name := out.String()
	if len(name) <= 64 {
		return name
	}
	sum := 0
	for _, char := range name {
		sum = sum*31 + int(char)
	}
	return fmt.Sprintf("%s_%08x", name[:55], sum)
}

func mcpSchema(raw json.RawMessage) *ai.Schema {
	open := true
	schema := &ai.Schema{Type: "object", AdditionalAllowed: &open}
	if len(raw) == 0 {
		return schema
	}
	var decoded ai.Schema
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return schema
	}
	if decoded.Type == "" {
		decoded.Type = "object"
	}
	return &decoded
}

func (s *Session) materializeSkill(skill AgencySkill) (string, error) {
	if strings.TrimSpace(skill.Name) == "" || strings.TrimSpace(skill.Body) == "" {
		return "", fmt.Errorf("a skill needs a name and a body")
	}
	root := filepath.Join(s.Workspace, ".cherrypi", "skills", skill.Name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(root, "SKILL.md")
	if err := os.WriteFile(path, []byte(skillBody(skill)), 0o644); err != nil {
		return "", err
	}
	for _, file := range skill.Files {
		target, err := skillFilePath(root, file.Path)
		if err != nil {
			slog.Warn("agency skill file",
				"session_id", s.ID, "skill", skill.Name, "path", file.Path, "error", err,
			)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, []byte(file.Text), 0o644); err != nil {
			return "", err
		}
	}
	if err := s.excludeFromGit(".cherrypi"); err != nil {
		slog.Debug("git exclude", "session_id", s.ID, "error", err)
	}
	return path, nil
}

func skillBody(skill AgencySkill) string {
	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString("name: " + skill.Name + "\n")
	out.WriteString("description: " + strings.ReplaceAll(skill.Description, "\n", " ") + "\n")
	if skill.DisableModelInvocation {
		out.WriteString("disable-model-invocation: true\n")
	}
	out.WriteString("---\n\n")
	out.WriteString(skill.Body)
	if !strings.HasSuffix(skill.Body, "\n") {
		out.WriteString("\n")
	}
	return out.String()
}

func skillFilePath(root, raw string) (string, error) {
	clean := filepath.Clean("/" + raw)
	target := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("refusing a path outside the skill directory")
	}
	return target, nil
}

func (s *Session) excludeFromGit(entry string) error {
	gitPath := filepath.Join(s.Workspace, ".git")
	info, err := os.Stat(gitPath)
	if err != nil || !info.IsDir() {
		return nil
	}
	excludePath := filepath.Join(gitPath, "info", "exclude")
	existing, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == "/"+entry+"/" {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
		return err
	}
	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "/" + entry + "/\n"
	return os.WriteFile(excludePath, []byte(content), 0o644)
}

func renderAgencySkills(skills []coding.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	return strings.TrimSpace(
		coding.FormatSkillsForPromptWithTool(skills, coding.SkillFileReadToolRead),
	)
}

func (s *Session) closeIntegrations() {
	s.mu.Lock()
	conns := s.mcpConns
	s.mcpConns = nil
	s.mu.Unlock()
	for _, conn := range conns {
		conn.Close()
	}
}

func (s *Session) reportIntegrations(
	ctx context.Context,
	report IntegrationReport,
	revision string,
) {
	if s.cfg.ReportIntegrations == nil || s.agencyTaskID == "" {
		return
	}
	if err := s.cfg.ReportIntegrations(ctx, s.agencyTaskID, revision, report); err != nil {
		slog.Warn("integration report",
			"session_id", s.ID,
			"task_id", s.agencyTaskID,
			"error", err,
		)
	}
}

func reportedTools(tools []agent.AgentTool) []ReportedTool {
	out := make([]ReportedTool, 0, len(tools))
	for _, tool := range tools {
		entry := ReportedTool{Name: tool.Name, Description: tool.Description}
		if tool.Parameters != nil {
			if raw, err := json.Marshal(tool.Parameters); err == nil {
				entry.Parameters = raw
			} else {
				slog.Warn("report tool schema",
					"tool", tool.Name,
					"error", err,
				)
			}
		}
		out = append(out, entry)
	}
	return out
}
