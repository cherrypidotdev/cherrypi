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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"google.golang.org/protobuf/types/known/timestamppb"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

func (s *Session) knowledgeTools() []agent.AgentTool {
	return []agent.AgentTool{
		s.wikiListTool(),
		s.wikiReadTool(),
		s.wikiWriteTool(),
		s.wikiSearchTool(),
		s.memoryWriteTool(),
		s.memoryReadTool(),
		s.memorySearchTool(),
		s.memoryForgetTool(),
		s.memoryRestoreTool(),
		s.scratchpadTool(),
		s.memoryStatusTool(),
		s.knowledgeSearchTool(),
		s.listDecisionsTool(),
		s.recordDecisionTool(),
	}
}

func (s *Session) wikiRef(params args) (string, string, error) {
	scope := params.str("scope")
	switch scope {
	case "", "project":
		if s.agencyProjectID == "" {
			return "org", "", nil
		}
		return "project", s.agencyProjectID, nil
	case "org":
		return "org", "", nil
	default:
		return "", "", fmt.Errorf("scope must be org or project")
	}
}

func (s *Session) wikiListTool() agent.AgentTool {
	return s.agencyTool(
		"wiki_list",
		"List Wiki Pages",
		"List the organization or project wiki pages. The project scope defaults to this task's project; pass scope=org for the organization wiki. Returns titles, paths, status, trust tier and update time; use wiki_read for the body.",
		ai.Object(
			ai.Opt(
				"scope",
				ai.String(
					"org or project; defaults to this task's project, or org when the task has none.",
				),
			),
			ai.Opt("query", ai.String("Filter by text in the title, description, tags or body.")),
			ai.Opt("path_prefix", ai.String("Only pages whose path starts with this prefix.")),
			ai.Opt("status", ai.String("draft, stable or deprecated.")),
			ai.Opt("tag", ai.String("Only pages carrying this tag.")),
			ai.Opt("limit", ai.Number("Maximum pages; defaults to 100.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			scope, scopeID, err := s.wikiRef(a)
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.ListConcepts(ctx, &agencyv1.ListConceptsRequest{
				Scope:      scope,
				ScopeId:    scopeID,
				Query:      a.str("query"),
				PathPrefix: a.str("path_prefix"),
				Status:     a.str("status"),
				Tag:        a.str("tag"),
				Limit:      int32(intArg(a, "limit", 100)),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) wikiReadTool() agent.AgentTool {
	return s.agencyTool(
		"wiki_read",
		"Read Wiki Page",
		"Read one wiki page with its OKF frontmatter, sources, links, backlinks and verification history. The project scope defaults to this task's project; pass scope=org for the organization wiki.",
		ai.Object(
			ai.Prop("path", ai.String("The page path, e.g. architecture/overview.")),
			ai.Opt(
				"scope",
				ai.String(
					"org or project; defaults to this task's project, or org when the task has none.",
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			scope, scopeID, err := s.wikiRef(a)
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.GetConcept(ctx, &agencyv1.GetConceptRequest{
				Scope: scope, ScopeId: scopeID, Path: a.str("path"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) wikiWriteTool() agent.AgentTool {
	return s.agencyTool(
		"wiki_write",
		"Write Wiki Page",
		"Create or update a wiki page. Your write is attributed to you and stays unverified until a human verifies it. Pass base_revision when you have read the page to detect a concurrent edit; omit it to write on top of the current revision. Pass decision_ids (from list_decisions) to record which decisions the page captures; they are added to any the page already cites.",
		ai.Object(
			ai.Prop("path", ai.String("The page path, e.g. architecture/overview.")),
			ai.Opt(
				"scope",
				ai.String(
					"org or project; defaults to this task's project, or org when the task has none.",
				),
			),
			ai.Opt("title", ai.String("The page title.")),
			ai.Opt("body", ai.String("The markdown body.")),
			ai.Opt("type", ai.String("The OKF type, e.g. Note, Decision, Reference.")),
			ai.Opt("description", ai.String("A one-line description.")),
			ai.Opt(
				"tags",
				ai.ArrayOf(ai.String("A tag without the leading #."), "Tags for the page."),
			),
			ai.Opt("status", ai.String("draft, stable or deprecated.")),
			ai.Opt(
				"stale_after",
				ai.String("An RFC3339 time or YYYY-MM-DD date after which the page is stale."),
			),
			ai.Opt(
				"base_revision",
				ai.Number("The revision you read; a mismatch fails instead of overwriting."),
			),
			ai.Opt("summary", ai.String("A short edit summary for the revision history.")),
			ai.Opt(
				"decision_ids",
				ai.ArrayOf(
					ai.Number("A decision id from list_decisions."),
					"Decisions this page records or relies on; they appear on the page and in the ledger.",
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			scope, scopeID, err := s.wikiRef(a)
			if err != nil {
				return toolOutput{}, err
			}
			path := a.str("path")
			existing, getErr := client.Knowledge.GetConcept(ctx, &agencyv1.GetConceptRequest{
				Scope: scope, ScopeId: scopeID, Path: path,
			})
			if getErr != nil {
				_, err := client.Knowledge.CreateConcept(ctx, &agencyv1.CreateConceptRequest{
					Scope:       scope,
					ScopeId:     scopeID,
					Path:        path,
					Type:        a.str("type"),
					Title:       firstNonEmpty(a.str("title"), path),
					Description: a.str("description"),
					Body:        a.raw("body"),
					Status:      a.str("status"),
					Tags:        stringList(a.list("tags")),
					StaleAfter:  timestampArg(a.str("stale_after")),
					Summary:     firstNonEmpty(a.str("summary"), "wiki_write"),
					DecisionIds: decisionIDsArg(a),
				})
				if err != nil {
					return toolOutput{}, err
				}
				res, err := client.Knowledge.GetConcept(ctx, &agencyv1.GetConceptRequest{
					Scope: scope, ScopeId: scopeID, Path: path,
				})
				if err != nil {
					return toolOutput{}, err
				}
				return protoOutput(res), nil
			}
			base := existing.GetConcept().GetRevision()
			if v, ok := a.number("base_revision"); ok && int64(v) > 0 {
				base = int64(v)
			}
			req := &agencyv1.UpdateConceptRequest{
				Scope:        scope,
				ScopeId:      scopeID,
				Path:         path,
				BaseRevision: base,
				Summary:      firstNonEmpty(a.str("summary"), "wiki_write"),
			}
			if a.present("title") {
				req.Title = ptr(a.str("title"))
			}
			if a.present("body") {
				req.Body = ptr(a.raw("body"))
			}
			if a.present("type") {
				req.Type = ptr(a.str("type"))
			}
			if a.present("description") {
				req.Description = ptr(a.str("description"))
			}
			if a.present("status") {
				req.Status = ptr(a.str("status"))
			}
			if a.present("tags") {
				req.Tags = stringList(a.list("tags"))
				req.ReplaceTags = true
			}
			if a.present("stale_after") {
				req.StaleAfter = timestampArg(a.str("stale_after"))
			}
			if a.present("decision_ids") {

				req.DecisionIds = mergeIDs(
					decisionIDsArg(a),
					existing.GetConcept().GetDecisions(),
				)
				req.ReplaceDecisions = true
			}
			res, err := client.Knowledge.UpdateConcept(ctx, req)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) wikiSearchTool() agent.AgentTool {
	return s.agencyTool(
		"wiki_search",
		"Search Wiki",
		"Search wiki pages visible to you: the org wiki and every project wiki you can read. Use memory_search for your own memory.",
		ai.Object(
			ai.Prop("query", ai.String("The text to search for.")),
			ai.Opt("scope", ai.String("Restrict to org or project.")),
			ai.Opt("limit", ai.Number("Maximum results; defaults to 50.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			req := &agencyv1.SearchKnowledgeRequest{
				Query: a.str("query"),
				Limit: int32(intArg(a, "limit", 50)),
			}
			if scope := a.str("scope"); scope != "" {
				req.Scope = scope
				if scope == "project" {
					req.ScopeId = s.agencyProjectID
				}
			}
			res, err := client.Knowledge.SearchKnowledge(ctx, req)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) knowledgeSearchTool() agent.AgentTool {
	return s.agencyTool(
		"knowledge_search",
		"Search Knowledge",
		"Search the wiki and your own memory in one call. Wiki pages come first, then memory; use wiki_read or memory_read to read a result in full.",
		ai.Object(
			ai.Prop("query", ai.String("The text to search for.")),
			ai.Opt("limit", ai.Number("Maximum results; defaults to 50.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}

			limit := int32(intArg(a, "limit", 50))
			wiki, err := client.Knowledge.SearchKnowledge(
				ctx,
				&agencyv1.SearchKnowledgeRequest{Query: a.str("query"), Limit: limit},
			)
			if err != nil {
				return toolOutput{}, err
			}
			memory, err := client.Knowledge.SearchKnowledge(
				ctx,
				&agencyv1.SearchKnowledgeRequest{
					Query: a.str("query"),
					Scope: "agent",
					Limit: limit,
				},
			)
			if err != nil {
				return toolOutput{}, err
			}
			wiki.Concepts = append(wiki.GetConcepts(), memory.GetConcepts()...)
			return protoOutput(wiki), nil
		},
	)
}

func (s *Session) memoryWriteTool() agent.AgentTool {
	return s.agencyTool(
		"memory_write",
		"Write Memory",
		"Write your private memory: long-term facts (default), today's daily log, or the scratchpad checklist. Memory is yours alone; it is lower priority than the shared wiki. Pass project=true to write the current project's memory layer.",
		ai.Object(
			ai.Opt("target", ai.String("long_term, daily or scratchpad; defaults to long_term.")),
			ai.Prop(
				"content",
				ai.String(
					"The text to remember. Appended with a timestamp header unless replace=true.",
				),
			),
			ai.Opt(
				"project",
				ai.Boolean("Write the current project's memory layer instead of the global one."),
			),
			ai.Opt("path", ai.String("An explicit memory path under memory/.")),
			ai.Opt("replace", ai.Boolean("Replace the file body instead of appending.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			projectID := ""
			if value, _ := a.boolean("project"); value {
				if s.agencyProjectID == "" {
					return toolOutput{}, fmt.Errorf("this task has no project memory layer")
				}
				projectID = s.agencyProjectID
			}
			res, err := client.Knowledge.WriteMemory(ctx, &agencyv1.WriteMemoryRequest{
				ProjectId: projectID,
				Target:    firstNonEmpty(a.str("target"), "long_term"),
				Path:      a.str("path"),
				Content:   a.raw("content"),
				Replace:   booleanArg(a, "replace"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) memoryReadTool() agent.AgentTool {
	return s.agencyTool(
		"memory_read",
		"Read Memory",
		"Read a memory file, or list every memory file with its size. Memory lives at memory/MEMORY.md, memory/SCRATCHPAD.md and memory/daily/YYYY-MM-DD.md (and the same paths under memory/projects/<project id>/).",
		ai.Object(
			ai.Opt("path", ai.String("A memory path; omit to list the files.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			path := a.str("path")
			if path == "" {
				res, err := client.Knowledge.GetMemoryStatus(
					ctx,
					&agencyv1.GetMemoryStatusRequest{},
				)
				if err != nil {
					return toolOutput{}, err
				}
				return protoOutput(res), nil
			}
			res, err := client.Knowledge.GetMemoryStatus(ctx, &agencyv1.GetMemoryStatusRequest{})
			if err != nil {
				return toolOutput{}, err
			}
			for _, file := range res.GetFiles() {
				if file.GetPath() == path {
					return protoOutput(&agencyv1.GetAgentMemoryFileResponse{Concept: file}), nil
				}
			}
			return toolOutput{}, fmt.Errorf("no memory file at %s", path)
		},
	)
}

func (s *Session) memorySearchTool() agent.AgentTool {
	return s.agencyTool(
		"memory_search",
		"Search Memory",
		"Search only your private memory. Use knowledge_search to search the wiki and memory together.",
		ai.Object(
			ai.Prop("query", ai.String("The text to search for.")),
			ai.Opt("limit", ai.Number("Maximum results; defaults to 50.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.SearchKnowledge(ctx, &agencyv1.SearchKnowledgeRequest{
				Query: a.str("query"),
				Scope: "agent",
				Limit: int32(intArg(a, "limit", 50)),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) memoryForgetTool() agent.AgentTool {
	return s.agencyTool(
		"memory_forget",
		"Forget Memory",
		"Remove every memory line matching the query and return a recovery id. Use memory_restore to undo.",
		ai.Object(
			ai.Prop("query", ai.String("The text whose matching lines should be removed.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.ForgetMemory(ctx, &agencyv1.ForgetMemoryRequest{
				Query: a.str("query"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) memoryRestoreTool() agent.AgentTool {
	return s.agencyTool(
		"memory_restore",
		"Restore Memory",
		"Restore the memory files removed by a memory_forget, using its recovery id. It is idempotent.",
		ai.Object(
			ai.Prop("recovery_id", ai.String("The id returned by memory_forget.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.RestoreMemory(ctx, &agencyv1.RestoreMemoryRequest{
				RecoveryId: a.str("recovery_id"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) scratchpadTool() agent.AgentTool {
	return s.agencyTool(
		"scratchpad",
		"Scratchpad",
		"Manage your scratchpad checklist: add, done, undo, clear_done or list. Open items are injected into your context on the next run.",
		ai.Object(
			ai.Prop("action", ai.String("add, done, undo, clear_done or list.")),
			ai.Opt("text", ai.String("The item text for add, done or undo.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.Scratchpad(ctx, &agencyv1.ScratchpadRequest{
				Action: a.str("action"),
				Text:   a.str("text"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) memoryStatusTool() agent.AgentTool {
	return s.agencyTool(
		"memory_status",
		"Memory Status",
		"Report your memory files, sizes, today's log path and the current injected snapshot (hash, layers and whether a layer was truncated).",
		ai.Object(
			ai.Opt(
				"project",
				ai.Boolean("Include the current project's memory layer in the snapshot report."),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			req := &agencyv1.GetMemoryStatusRequest{}
			if value, _ := a.boolean("project"); value {
				if s.agencyProjectID == "" {
					return toolOutput{}, fmt.Errorf("this task has no project memory layer")
				}
				req.ProjectId = s.agencyProjectID
			}
			res, err := client.Knowledge.GetMemoryStatus(ctx, req)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) memoryContext(ctx context.Context) (string, error) {
	client, err := s.agencyClient()
	if err != nil {
		return "", err
	}
	res, err := client.Knowledge.GetMemoryContext(ctx, &agencyv1.GetMemoryContextRequest{
		ProjectId: s.agencyProjectID,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.GetText()), nil
}

func intArg(a args, name string, fallback int) int {
	if value, ok := a.number(name); ok && value > 0 {
		return int(value)
	}
	return fallback
}

func booleanArg(a args, name string) bool {
	value, _ := a.boolean(name)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ptr(value string) *string { return &value }

func timestampArg(raw string) *timestamppb.Timestamp {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return timestamppb.New(parsed)
		}
	}
	return nil
}

func (s *Session) listDecisionsTool() agent.AgentTool {
	return s.agencyTool(
		"list_decisions",
		"List Decisions",
		"Read the decision ledger: who created this task, sent each follow-up, answered each questionnaire and settled it, plus decisions recorded on purpose. Defaults to this task's rows; pass project=true for the project's ledger or scope=org for the organization's.",
		ai.Object(
			ai.Opt(
				"project",
				ai.Boolean("List the current project's ledger instead of this task's."),
			),
			ai.Opt("scope", ai.String("org or project; defaults to this task's rows.")),
			ai.Opt(
				"kind",
				ai.String("task_created, follow_up, question_answered, task_settled or recorded."),
			),
			ai.Opt("limit", ai.Number("Maximum rows; defaults to 100.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			req := &agencyv1.ListDecisionsRequest{
				Kind:  a.str("kind"),
				Limit: int32(intArg(a, "limit", 100)),
			}
			switch a.str("scope") {
			case "org":
				req.Scope = "org"
			case "project":
				if s.agencyProjectID == "" {
					return toolOutput{}, fmt.Errorf("this task has no project")
				}
				req.Scope = "project"
				req.ScopeId = s.agencyProjectID
			default:
				if value, _ := a.boolean("project"); value {
					if s.agencyProjectID == "" {
						return toolOutput{}, fmt.Errorf("this task has no project")
					}
					req.Scope = "project"
					req.ScopeId = s.agencyProjectID
				} else if s.agencyTaskID != "" {
					req.TaskId = s.agencyTaskID
				} else if s.agencyProjectID != "" {
					req.Scope = "project"
					req.ScopeId = s.agencyProjectID
				}
			}
			res, err := client.Knowledge.ListDecisions(ctx, req)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) recordDecisionTool() agent.AgentTool {
	return s.agencyTool(
		"record_decision",
		"Record Decision",
		"Record a decision you made, attributed to you, and optionally link the wiki pages it produced. The project scope defaults to this task's project; pass scope=org for an organization-wide decision.",
		ai.Object(
			ai.Prop("summary", ai.String("What was decided, in one line.")),
			ai.Opt("rationale", ai.String("Why, in your own words.")),
			ai.Opt(
				"scope",
				ai.String(
					"org or project; defaults to this task's project, or org when the task has none.",
				),
			),
			ai.Opt(
				"concept_paths",
				ai.ArrayOf(
					ai.String("An existing page path in the same scope."),
					"Pages this decision produced or relies on.",
				),
			),
			ai.Opt("project", ai.Boolean("Record into this task's project (the default).")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			scope, scopeID, err := s.wikiRef(a)
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.RecordDecision(ctx, &agencyv1.RecordDecisionRequest{
				Scope:        scope,
				ScopeId:      scopeID,
				TaskId:       s.agencyTaskID,
				Summary:      a.str("summary"),
				Rationale:    a.raw("rationale"),
				ConceptPaths: stringList(a.list("concept_paths")),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func mergeIDs(wanted []int64, existing []*agencyv1.Decision) []int64 {
	seen := make(map[int64]struct{}, len(wanted)+len(existing))
	out := make([]int64, 0, len(wanted)+len(existing))
	for _, id := range wanted {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, decision := range existing {
		id := decision.GetId()
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func decisionIDsArg(a args) []int64 {
	out := make([]int64, 0, 4)
	for _, raw := range a.list("decision_ids") {
		switch value := raw.(type) {
		case float64:
			if value > 0 {
				out = append(out, int64(value))
			}
		case string:
			if id, err := strconv.ParseInt(value, 10, 64); err == nil && id > 0 {
				out = append(out, id)
			}
		}
	}
	return out
}
