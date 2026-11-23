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
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

func (s *Session) knowledgeTools() []agent.AgentTool {
	return []agent.AgentTool{
		s.addKnowledgeTool(),
		s.listKnowledgeTool(),
		s.readKnowledgeTool(),
		s.searchKnowledgeTool(),
		s.updateKnowledgeTool(),
		s.memoryWriteTool(),
		s.memoryReadTool(),
		s.memorySearchTool(),
		s.memoryForgetTool(),
		s.memoryRestoreTool(),
		s.scratchpadTool(),
		s.memoryStatusTool(),
	}
}

func (s *Session) knowledgeScope(a args) (*string, string, error) {
	switch a.str("scope") {
	case "", "project":
		if s.agencyProjectID == "" {
			return nil, "", nil
		}
		return nil, s.agencyProjectID, nil
	case "org":
		empty := ""
		return &empty, "", nil
	default:
		return nil, "", fmt.Errorf("scope must be project or org")
	}
}

func (s *Session) addKnowledgeTool() agent.AgentTool {
	return s.agencyTool(
		"add_knowledge",
		"Add Knowledge",
		"Record one crystallized piece of knowledge with hard links to the inputs it came from. An agent's bit starts pending and a human approves it; a human's bit is approved at once. Always attach the real source: a task prompt, a follow-up message, a questionnaire answer or a system prompt version.",
		ai.Object(
			ai.Prop("content", ai.String("The knowledge, as short markdown.")),
			ai.Opt(
				"title",
				ai.String("An optional title; derived from the first line when omitted."),
			),
			ai.Opt(
				"scope",
				ai.String("org or project; defaults to this task's project."),
			),
			ai.Opt(
				"tags",
				ai.ArrayOf(
					ai.String("A lower-case tag, e.g. deploy or ci-cd."),
					"Verbatim tags for the bit.",
				),
			),
			ai.Opt(
				"sources",
				ai.ArrayOf(
					ai.Object(
						ai.Prop(
							"kind",
							ai.String(
								"task_prompt, follow_up, questionnaire or system_prompt.",
							),
						),
						ai.Opt("task_id", ai.String("The task; defaults to this task.")),
						ai.Opt(
							"message_id",
							ai.String(
								"A follow-up message id (from the prompt envelope's messageId); omit for the latest follow-up.",
							),
						),
						ai.Opt("request_id", ai.String("A questionnaire dialog requestId.")),
						ai.Opt("question_id", ai.String("A questionnaire question id, e.g. q1.")),
						ai.Opt(
							"prompt_id",
							ai.Number("A prompt version id from get_prompt."),
						),
					),
					"The inputs this knowledge came from; at least one is expected.",
				),
			),
			ai.Opt(
				"related_ids",
				ai.ArrayOf(ai.String("A knowledge bit id."), "Related knowledge bits."),
			),
			ai.Opt(
				"supersedes_ids",
				ai.ArrayOf(
					ai.String("A knowledge bit id."),
					"Older bits this one replaces; they retire when this bit is approved.",
				),
			),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			projectOverride, _, err := s.knowledgeScope(a)
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.AddKnowledge(ctx, &agencyv1.AddKnowledgeRequest{
				ProjectId:     projectOverride,
				Title:         a.str("title"),
				Content:       a.raw("content"),
				Tags:          stringList(a.list("tags")),
				Sources:       knowledgeSourceInputs(a.list("sources")),
				RelatedIds:    stringList(a.list("related_ids")),
				SupersedesIds: stringList(a.list("supersedes_ids")),
				OriginTaskId:  s.agencyTaskID,
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) listKnowledgeTool() agent.AgentTool {
	return s.agencyTool(
		"list_knowledge",
		"List Knowledge",
		"List knowledge bits. The project scope defaults to this task's project; pass scope=org for the organization or scope=all for everything you can read. Lists show approved bits; pass status=pending, rejected or superseded for the others.",
		ai.Object(
			ai.Opt("scope", ai.String("org, project or all; defaults to this task's project.")),
			ai.Opt("query", ai.String("Filter by text in the title, content or tags.")),
			ai.Opt("tag", ai.String("Only bits carrying this tag.")),
			ai.Opt("status", ai.String("approved, pending, rejected or superseded.")),
			ai.Opt("task_id", ai.String("Only bits recorded from or citing this task.")),
			ai.Opt("limit", ai.Number("Maximum bits; defaults to 50.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			scope := a.str("scope")
			if scope == "" {
				if s.agencyProjectID != "" {
					scope = "project"
				} else {
					scope = "org"
				}
			}
			projectID := ""
			if scope == "project" {
				projectID = s.agencyProjectID
			}
			res, err := client.Knowledge.ListKnowledge(ctx, &agencyv1.ListKnowledgeRequest{
				Scope:     scope,
				ProjectId: projectID,
				Query:     a.str("query"),
				Tag:       a.str("tag"),
				Status:    a.str("status"),
				TaskId:    a.str("task_id"),
				Limit:     int32(intArg(a, "limit", 50)),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) readKnowledgeTool() agent.AgentTool {
	return s.agencyTool(
		"read_knowledge",
		"Read Knowledge",
		"Read one knowledge bit with its sources, related and superseding links, backlinks and revision history.",
		ai.Object(ai.Prop("id", ai.String("The knowledge bit id."))),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.GetKnowledge(ctx, &agencyv1.GetKnowledgeRequest{
				Id: a.str("id"),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) searchKnowledgeTool() agent.AgentTool {
	return s.agencyTool(
		"search_knowledge",
		"Search Knowledge",
		"Search recorded knowledge bits. Approved bits are searched by default; pass status=pending to search the review queue. Use memory_search for your own memory.",
		ai.Object(
			ai.Prop("query", ai.String("The text to search for.")),
			ai.Opt(
				"scope",
				ai.String(
					"org, project or all; defaults to every bit you can read.",
				),
			),
			ai.Opt("tag", ai.String("Only bits carrying this tag.")),
			ai.Opt("status", ai.String("approved, pending, rejected or superseded.")),
			ai.Opt("limit", ai.Number("Maximum results; defaults to 50.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			scope := a.str("scope")
			projectID := ""
			if scope == "project" {
				if s.agencyProjectID == "" {
					return toolOutput{}, fmt.Errorf("this task has no project")
				}
				projectID = s.agencyProjectID
			}
			res, err := client.Knowledge.SearchKnowledge(ctx, &agencyv1.SearchKnowledgeRequest{
				Query:     a.str("query"),
				Scope:     scope,
				ProjectId: projectID,
				Tag:       a.str("tag"),
				Status:    a.str("status"),
				Limit:     int32(intArg(a, "limit", 50)),
			})
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func (s *Session) updateKnowledgeTool() agent.AgentTool {
	return s.agencyTool(
		"update_knowledge",
		"Update Knowledge",
		"Edit a knowledge bit you wrote. Any content, tag or source edit returns an approved bit to pending for a human to re-approve.",
		ai.Object(
			ai.Prop("id", ai.String("The knowledge bit id.")),
			ai.Prop("revision", ai.Number("The revision you read; a mismatch fails.")),
			ai.Opt("title", ai.String("A new title.")),
			ai.Opt("content", ai.String("A new content body.")),
			ai.Opt(
				"tags",
				ai.ArrayOf(ai.String("A lower-case tag."), "The complete new tag list."),
			),
			ai.Opt(
				"sources",
				ai.ArrayOf(
					ai.Object(
						ai.Prop(
							"kind",
							ai.String("task_prompt, follow_up, questionnaire or system_prompt."),
						),
						ai.Opt("task_id", ai.String("The task.")),
						ai.Opt("message_id", ai.String("A follow-up message id.")),
						ai.Opt("request_id", ai.String("A questionnaire requestId.")),
						ai.Opt("question_id", ai.String("A questionnaire question id.")),
						ai.Opt("prompt_id", ai.Number("A prompt version id.")),
					),
					"The complete new source list.",
				),
			),
			ai.Opt(
				"related_ids",
				ai.ArrayOf(ai.String("A knowledge bit id."), "The complete related list."),
			),
			ai.Opt(
				"supersedes_ids",
				ai.ArrayOf(ai.String("A knowledge bit id."), "The complete supersedes list."),
			),
			ai.Opt("summary", ai.String("A short edit summary for the revision history.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			revision, _ := a.number("revision")
			req := &agencyv1.UpdateKnowledgeRequest{
				Id:           a.str("id"),
				BaseRevision: int64(revision),
				Summary:      a.str("summary"),
			}
			if a.present("title") {
				req.Title = ptr(a.str("title"))
			}
			if a.present("content") {
				req.Content = ptr(a.raw("content"))
			}
			if a.present("tags") {
				req.Tags = stringList(a.list("tags"))
				req.ReplaceTags = true
			}
			if a.present("sources") {
				req.Sources = knowledgeSourceInputs(a.list("sources"))
				req.ReplaceSources = true
			}
			if a.present("related_ids") {
				req.RelatedIds = stringList(a.list("related_ids"))
				req.ReplaceRelated = true
			}
			if a.present("supersedes_ids") {
				req.SupersedesIds = stringList(a.list("supersedes_ids"))
				req.ReplaceSupersedes = true
			}
			res, err := client.Knowledge.UpdateKnowledge(ctx, req)
			if err != nil {
				return toolOutput{}, err
			}
			return protoOutput(res), nil
		},
	)
}

func knowledgeSourceInputs(items []any) []*agencyv1.KnowledgeSourceInput {
	out := make([]*agencyv1.KnowledgeSourceInput, 0, len(items))
	for _, entry := range items {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		promptID, _ := item["prompt_id"].(float64)
		out = append(out, &agencyv1.KnowledgeSourceInput{
			Kind:       strings.TrimSpace(stringValue(item["kind"])),
			TaskId:     strings.TrimSpace(stringValue(item["task_id"])),
			MessageId:  strings.TrimSpace(stringValue(item["message_id"])),
			RequestId:  strings.TrimSpace(stringValue(item["request_id"])),
			QuestionId: strings.TrimSpace(stringValue(item["question_id"])),
			PromptId:   int64(promptID),
		})
	}
	return out
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func (s *Session) memoryWriteTool() agent.AgentTool {
	return s.agencyTool(
		"memory_write",
		"Write Memory",
		"Write your private memory: long-term facts (default), today's daily log, or the scratchpad checklist. Memory is yours alone; it is lower priority than the shared knowledge. Pass project=true to write the current project's memory layer.",
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
					return protoOutput(&agencyv1.GetAgentMemoryFileResponse{File: file}), nil
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
		"Search only your private memory. Use search_knowledge to search the shared knowledge.",
		ai.Object(
			ai.Prop("query", ai.String("The text to search for.")),
			ai.Opt("limit", ai.Number("Maximum results; defaults to 50.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			res, err := client.Knowledge.SearchMemory(ctx, &agencyv1.SearchMemoryRequest{
				Query: a.str("query"),
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
