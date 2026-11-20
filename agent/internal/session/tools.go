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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

func text(value string) agent.AgentToolResult {
	return agent.AgentToolResult{Content: ai.ContentList{ai.TextContent{Text: value}}}
}

func errorText(value string) agent.AgentToolResult {
	return agent.AgentToolResult{
		Content: ai.ContentList{ai.TextContent{Text: value}},
		IsError: true,
	}
}

func (s *Session) workspacePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("provide a path")
	}
	path := raw
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.Workspace, path)
	}
	rel, err := filepath.Rel(s.Workspace, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("refusing to use a file outside the task workspace")
	}
	return path, nil
}

func (s *Session) setTaskURLs() agent.AgentTool {
	return agent.AgentTool{
		Name:        "setTaskUrls",
		Label:       "Set Task URLs",
		Description: "Replace the links attached to this CherryPi task. Pass the complete list on every call, and pass an empty list to clear them. Use it to surface a pull request, issue, deployment, or any other URL; CherryPi shows the links on the task in the task list and detail view.",
		Parameters: ai.Object(
			ai.Prop("urls", ai.ArrayOf(ai.Object(
				ai.Prop("url", ai.String("Absolute http(s) URL.")),
				ai.Prop(
					"label",
					ai.String(`Short label shown next to the URL, e.g. "PR" or "Issue #42".`),
				),
			), "The complete list of links for this task.")),
		),
		Execute: func(_ context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			links := []map[string]any{}
			if urls, ok := params["urls"].([]any); ok {
				for _, raw := range urls {
					entry, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					url, _ := entry["url"].(string)
					url = strings.TrimSpace(url)
					if url == "" {
						continue
					}
					label, _ := entry["label"].(string)
					links = append(
						links,
						map[string]any{"url": url, "label": strings.TrimSpace(label)},
					)
				}
			}
			s.notify("task_link", map[string]any{"links": links})
			if len(links) == 0 {
				return text("Cleared this task's links."), nil
			}
			rendered := make([]string, 0, len(links))
			for _, link := range links {
				url, _ := link["url"].(string)
				label, _ := link["label"].(string)
				if label != "" {
					rendered = append(rendered, label+" ("+url+")")
					continue
				}
				rendered = append(rendered, url)
			}
			return text(
				"Set " + strconv.Itoa(len(links)) + " " +
					plural(len(links), "link", "links") + ": " +
					strings.Join(rendered, ", "),
			), nil
		},
	}
}

func (s *Session) setTaskArtifacts() agent.AgentTool {
	return agent.AgentTool{
		Name:        "setTaskArtifacts",
		Label:       "Set Task Artifacts",
		Description: "Publish this task's durable output artifacts (A2A artifacts): each has an id, a name, an optional description, and parts. Use `file` parts for workspace files (they are uploaded for you), `text` for inline text, and `data` for structured JSON. Artifacts represent the task's result and stay in its Artifacts tab; for transient, current-status context use uploadAttachment instead. By default the call replaces the task's whole artifact set; pass replace:false to update only the listed artifacts, and append:true with lastChunk:false to stream parts into one artifact across calls.",
		Parameters: ai.Object(
			ai.Opt(
				"replace",
				ai.Boolean(
					"Replace the task's complete artifact set (default true). Set false to update just the listed artifacts.",
				),
			),
			ai.Prop(
				"artifacts",
				ai.ArrayOf(
					ai.Object(
						ai.Opt(
							"id",
							ai.String(
								"Stable artifact id, unique within the task. Omit to let the agency generate one.",
							),
						),
						ai.Prop("name", ai.String("Human-readable artifact name.")),
						ai.Opt(
							"description",
							ai.String("What the artifact contains."),
						),
						ai.Opt(
							"append",
							ai.Boolean(
								"Append these parts to the existing artifact with this id instead of replacing them.",
							),
						),
						ai.Opt(
							"lastChunk",
							ai.Boolean(
								"Set false while more parts are still coming; defaults to true (complete).",
							),
						),
						ai.Prop(
							"parts",
							ai.ArrayOf(
								ai.Object(
									ai.Opt(
										"file",
										ai.String(
											"Workspace path of a file to upload as a file part.",
										),
									),
									ai.Opt("url", ai.String("A URL to reference as a file part.")),
									ai.Opt("text", ai.String("Inline text part.")),
									ai.Opt(
										"data",
										jsonValueSchema(
											"Structured JSON data part, encoded as a JSON string.",
										),
									),
									ai.Opt("filename", ai.String("Filename for a file part.")),
									ai.Opt("mediaType", ai.String("Media type for a file part.")),
								),
								"The artifact's parts, in order.",
							),
						),
					),
					"The artifacts to publish.",
				),
			),
		),
		Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			artifacts, _ := params["artifacts"].([]any)
			updates := make([]map[string]any, 0, len(artifacts))
			for _, raw := range artifacts {
				artifact, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				parts, err := s.artifactParts(ctx, artifact)
				if err != nil {
					return errorText(err.Error()), nil
				}
				updates = append(updates, map[string]any{
					"id":          stringArg(artifact, "id"),
					"name":        stringArg(artifact, "name"),
					"description": stringArg(artifact, "description"),
					"append":      boolArg(artifact, "append"),
					"last_chunk":  lastChunkArg(artifact),
					"parts":       parts,
				})
			}
			replace, ok := params["replace"].(bool)
			s.notify("task_artifact", map[string]any{
				"replace":   replace || !ok,
				"artifacts": updates,
			})
			if len(updates) == 0 {
				return text("Cleared this task's artifacts."), nil
			}
			names := make([]string, 0, len(updates))
			for _, update := range updates {
				name, _ := update["name"].(string)
				names = append(names, name)
			}
			return text(
				"Published " + strconv.Itoa(len(updates)) + " artifact(s): " +
					strings.Join(names, ", ") + ".",
			), nil
		},
	}
}

func (s *Session) artifactParts(
	ctx context.Context,
	artifact map[string]any,
) ([]map[string]any, error) {
	parts, _ := artifact["parts"].([]any)
	out := make([]map[string]any, 0, len(parts))
	for _, raw := range parts {
		part, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		file := strings.TrimSpace(stringArg(part, "file"))
		mediaType := stringArg(part, "mediaType")
		switch {
		case file != "":
			path, err := s.workspacePath(file)
			if err != nil {
				return nil, errors.New("refusing to upload a file outside the task workspace")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, errors.New("could not read " + file + ": " + err.Error())
			}
			raw, err := s.cfg.Agency.Upload(
				ctx,
				"/api/rest/attachments?kind=artifact&id="+url.QueryEscape(s.agencyTaskID),
				filepath.Base(path),
				data,
			)
			if err != nil {
				return nil, errors.New("upload failed for " + file + ": " + err.Error())
			}
			var uploaded struct {
				ID          int64  `json:"id"`
				URL         string `json:"url"`
				Filename    string `json:"filename"`
				ContentType string `json:"contentType"`
			}
			if err := json.Unmarshal(raw, &uploaded); err != nil {
				return nil, errors.New("upload failed for " + file + ": " + err.Error())
			}
			filename := stringArg(part, "filename")
			if filename == "" {
				filename = uploaded.Filename
			}
			if filename == "" {
				filename = filepath.Base(path)
			}
			if mediaType == "" {
				mediaType = uploaded.ContentType
			}
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			out = append(out, map[string]any{
				"kind":          "file",
				"attachment_id": uploaded.ID,
				"url":           uploaded.URL,
				"filename":      filename,
				"media_type":    mediaType,
			})
		case strings.TrimSpace(stringArg(part, "url")) != "":
			out = append(out, map[string]any{
				"kind":       "file",
				"url":        strings.TrimSpace(stringArg(part, "url")),
				"filename":   stringArg(part, "filename"),
				"media_type": mediaType,
			})
		case part["data"] != nil:
			encoded, err := jsonText(part["data"])
			if err != nil {
				return nil, err
			}
			out = append(out, map[string]any{"kind": "data", "data": encoded})
		case part["text"] != nil:
			text, _ := part["text"].(string)
			out = append(out, map[string]any{"kind": "text", "text": text})
		}
	}
	return out, nil
}

func jsonValueSchema(description string) *ai.Schema {
	return ai.String(description)
}

func objectArg(values map[string]any, name string) map[string]any {
	switch value := values[name].(type) {
	case map[string]any:
		return value
	case string:
		var out map[string]any
		if json.Unmarshal([]byte(value), &out) == nil {
			return out
		}
	}
	return nil
}

func jsonText(value any) (string, error) {
	if text, ok := value.(string); ok {
		if !json.Valid([]byte(text)) {
			return "", errors.New("the value must be valid JSON")
		}
		return text, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func dropNilValues(value any) any {
	switch value := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if item == nil {
				continue
			}
			out[key] = dropNilValues(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = dropNilValues(item)
		}
		return out
	default:
		return value
	}
}

func stringArg(values map[string]any, name string) string {
	value, _ := values[name].(string)
	return strings.TrimSpace(value)
}

func boolArg(values map[string]any, name string) bool {
	value, _ := values[name].(bool)
	return value
}

func lastChunkArg(values map[string]any) bool {
	value, ok := values["lastChunk"].(bool)
	return !ok || value
}

func (s *Session) uploadAttachment() agent.AgentTool {
	return agent.AgentTool{
		Name:        "uploadAttachment",
		Label:       "Upload Attachment",
		Description: "Upload a file from this task's workspace as a task attachment and get back a URL. Use it for transient, current-status context (a screenshot of what you are doing, a log, a diff) that belongs with the conversation; it appears in the task Attachments tray. Durable task results belong in setTaskArtifacts instead, which owns the long-lived Artifacts tab. The path must be inside the task workspace.",
		Parameters: ai.Object(
			ai.Prop(
				"path",
				ai.String(
					"Path to the file, relative to the task workspace (or absolute inside it).",
				),
			),
		),
		Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			raw := strings.TrimSpace(stringArg(params, "path"))
			if raw == "" {
				return text("Provide the path of the file to upload."), nil
			}
			path, err := s.workspacePath(raw)
			if err != nil {
				return text("Refusing to upload a file outside the task workspace."), nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return text("Could not read " + raw + ": " + err.Error()), nil
			}
			query := "/api/rest/attachments?kind=task&id=" + url.QueryEscape(s.agencyTaskID)
			uploaded, err := s.cfg.Agency.Upload(ctx, query, filepath.Base(path), data)
			if err != nil {
				return text("Upload failed: " + err.Error()), nil
			}
			var attachment struct {
				URL      string `json:"url"`
				Filename string `json:"filename"`
			}
			_ = json.Unmarshal(uploaded, &attachment)
			name := attachment.Filename
			if name == "" {
				name = filepath.Base(path)
			}
			return text(
				"Uploaded " + name + " as a task attachment. Reference it in your " +
					"reply as a relative link, e.g. ![" + name + "](" + attachment.URL +
					"). Use setTaskArtifacts for durable results.",
			), nil
		},
	}
}

func (s *Session) reportTaskProgress() agent.AgentTool {
	return agent.AgentTool{
		Name:  "reportTaskProgress",
		Label: "Report Task Progress",

		ExecutionMode: agent.ToolSequential,
		Description:   "Publish this task's plan and progress. Pass the complete todo list every time (an empty list clears it) plus an optional one-line summary of what you are doing now. Every task starts with a plan: set a short one (3–8 items) at the start of every activity block — your first block in a run, and every block after an output message. The worker asks for it through the provider's tool choice at the start of every block. One plan covers the whole block, however many messages it spans, so keep working through it and do not publish again until the next block. Mark an item `in_progress` when you begin it and `done` when it lands. Updating the plan is bookkeeping, never a stopping point: call the tool and carry on with the work in the same turn, and only end the run when the task is actually done. Plans are expected to change when new input or an external event (a CI result, a review) invalidates a step; replace the list instead of forcing the old one. A run does not settle while the plan is missing, unfinished, or stale. Omit `todos` to read the current plan back. The dashboard derives numeric progress (done/total) from the list; users cannot edit it.",
		Parameters: ai.Object(
			ai.Opt(
				"summary",
				ai.String(
					"One line describing what you are doing now. Omitted or empty clears it.",
				),
			),
			ai.Opt("todos", ai.ArrayOf(ai.Object(
				ai.Prop("text", ai.String("The step's text (plain text, max 240 chars).")),
				ai.Opt(
					"status",
					ai.String(`One of "pending" (default), "in_progress", or "done".`),
				),
			), "The complete plan in order. Replaces the previous list; [] clears it. Max 50 items.")),
		),
		Execute: func(_ context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {

			if todos, present := params["todos"]; !present || todos == nil {
				return text(renderProgress("Current plan", s.progressState())), nil
			}
			rawTodos, _ := params["todos"].([]any)
			todos := make([]ProgressTodo, 0, len(rawTodos))
			for _, raw := range rawTodos {
				entry, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				step, _ := entry["text"].(string)
				status, _ := entry["status"].(string)
				todos = append(todos, ProgressTodo{Text: step, Status: status})
			}
			summary, _ := params["summary"].(string)
			progress := normalizeProgress(summary, todos)
			s.setProgress(progress)

			notes := make([]string, 0, 2)
			if dropped := len(rawTodos) - len(progress.Todos); dropped > 0 {
				notes = append(notes, fmt.Sprintf("dropped %d empty item(s)", dropped))
			}
			if len(rawTodos) > maxProgressItems {
				notes = append(
					notes,
					fmt.Sprintf("kept the first %d items", maxProgressItems),
				)
			}
			suffix := ""
			if len(notes) > 0 {
				suffix = "\n(" + strings.Join(notes, "; ") + ")"
			}
			return text(renderProgress("Updated plan", progress) + suffix), nil
		},
	}
}

func (s *Session) publishSite() agent.AgentTool {
	return agent.AgentTool{
		Name:        "publishSite",
		Label:       "Publish Site",
		Description: "Publish a static website from a zip archive in this task's workspace. The archive must have index.html at its root and contain only regular files (no symlinks or ../ paths), up to 50 MiB. `id` is an agency-unique slug ([a-z0-9][a-z0-9-]{0,63}); publishing the same id replaces the contents and settings. Returns the published site URL and makes a Site tab available in this task's console. You are encouraged to use this built-in feature to demonstrate your findings and present visual proposals, such as landing pages, dashboards, report pages, and interactive prototypes, directly beside the task chat. Sites are served from a sandboxed iframe with an opaque origin: build with a relative base (Vite: base: \"./\") because root-absolute asset paths resolve against the agency and return its HTML shell; use window.cherrypi for auth; localStorage/sessionStorage work through a shell-backed shim, while cookies, IndexedDB and service workers do not.",
		Parameters: ai.Object(
			ai.Prop(
				"id",
				ai.String(
					"Agency-unique site id, lower-kebab, e.g. `landing-page`. Publishing the same id updates that site.",
				),
			),
			ai.Prop(
				"archiveLocalPath",
				ai.String(
					"Path to a .zip archive inside the task workspace (absolute or relative to the workspace).",
				),
			),
			ai.Opt(
				"settings",
				jsonValueSchema(
					"Settings encoded as a JSON object string: title, description, "+
						"onlyCreatorMayEdit, and/or agencyWide.",
				),
			),
		),
		Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			id := stringArg(params, "id")
			if !siteIDPattern.MatchString(id) {
				return text("The site id must be lower-kebab: [a-z0-9][a-z0-9-]{0,63}."), nil
			}
			archive := stringArg(params, "archiveLocalPath")
			if archive == "" {
				return text("Provide the path of the zip archive to publish."), nil
			}
			path, err := s.workspacePath(archive)
			if err != nil {
				return text("Refusing to publish a file outside the task workspace."), nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return text("Could not read " + archive + ": " + err.Error()), nil
			}

			fields := [][2]string{{"origin_task_id", s.agencyTaskID}}
			settings := objectArg(params, "settings")
			for _, field := range []struct{ form, arg string }{
				{"title", "title"},
				{"description", "description"},
			} {
				if _, ok := settings[field.arg]; ok {
					fields = append(fields, [2]string{field.form, stringArg(settings, field.arg)})
				}
			}
			for _, field := range []struct{ form, arg string }{
				{"only_creator_may_edit", "onlyCreatorMayEdit"},
				{"agency_wide", "agencyWide"},
			} {
				if value, ok := settings[field.arg].(bool); ok {
					fields = append(fields, [2]string{field.form, strconv.FormatBool(value)})
				}
			}
			raw, err := s.cfg.Agency.UploadWithFields(
				ctx,
				"/api/rest/sites/"+url.PathEscape(id)+"/publish",
				filepath.Base(path),
				data,
				fields,
			)
			if err != nil {
				return text("Publish failed: " + err.Error()), nil
			}
			var site struct {
				URL      string   `json:"url"`
				Title    string   `json:"title"`
				Warnings []string `json:"warnings"`
			}
			_ = json.Unmarshal(raw, &site)
			siteURL := site.URL
			if siteURL == "" {
				siteURL = "/sites/" + id
			}
			title := site.Title
			if title == "" {
				title = id
			}
			out := "Published " + title + " at " + s.cfg.AgencyURL + siteURL +
				". This task now has a Site tab."
			if len(site.Warnings) > 0 {
				out += "\n\nWarnings:"
				for _, warning := range site.Warnings {
					out += "\n- " + warning
				}
			}
			return text(out), nil
		},
	}
}

var siteIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func (s *Session) editSite() agent.AgentTool {
	return agent.AgentTool{
		Name:        "editSite",
		Label:       "Edit Site Settings",
		Description: "Change a published site's settings without replacing its contents: title, description, onlyCreatorMayEdit, and/or agencyWide. Only the site's creator, or any agent when the creator allowed collaborative editing, may call it.",
		Parameters: ai.Object(
			ai.Prop("id", ai.String("The published site id.")),
			ai.Prop("updatedSettings", ai.Object(
				ai.Opt("title", ai.String("Human-readable site title.")),
				ai.Opt("description", ai.String("What the site contains.")),
				ai.Opt(
					"onlyCreatorMayEdit",
					ai.Boolean(
						"When true, only the creating agent may republish or edit through the tools.",
					),
				),
				ai.Opt(
					"agencyWide",
					ai.Boolean("When true, list the site under the sidebar's Sites section."),
				),
			)),
		),
		Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
			id := stringArg(params, "id")
			if id == "" {
				return text("Provide the site id to edit."), nil
			}
			body := map[string]any{}
			settings, _ := params["updatedSettings"].(map[string]any)
			for _, field := range []struct{ form, arg string }{
				{"title", "title"},
				{"description", "description"},
			} {
				if _, ok := settings[field.arg]; ok {
					body[field.form] = stringArg(settings, field.arg)
				}
			}
			for _, field := range []struct{ form, arg string }{
				{"only_creator_may_edit", "onlyCreatorMayEdit"},
				{"agency_wide", "agencyWide"},
			} {
				if value, ok := settings[field.arg].(bool); ok {
					body[field.form] = value
				}
			}
			raw, err := s.cfg.Agency.REST(
				ctx,
				http.MethodPatch,
				"/api/rest/sites/"+url.PathEscape(id),
				body,
			)
			if err != nil {
				return text("Edit failed: " + err.Error()), nil
			}
			var site struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			}
			_ = json.Unmarshal(raw, &site)
			siteURL := site.URL
			if siteURL == "" {
				siteURL = "/sites/" + id
			}
			title := site.Title
			if title == "" {
				title = id
			}
			return text("Updated " + title + " at " + s.cfg.AgencyURL + siteURL + "."), nil
		},
	}
}

func (s *Session) pluginAgentTools() []agent.AgentTool {
	s.mu.Lock()
	defs := s.pluginToolDefs
	s.mu.Unlock()
	out := make([]agent.AgentTool, 0, len(defs))
	for _, def := range defs {
		def := def
		var parameters *ai.Schema
		if def.InputSchema != "" {
			var schema ai.Schema
			if err := json.Unmarshal([]byte(def.InputSchema), &schema); err == nil {
				parameters = &schema
			}
		}
		if parameters == nil {
			parameters = ai.Object()
		}
		description := def.Description
		if strings.TrimSpace(description) == "" {
			description = "Plugin tool " + def.Name + "."
		}
		out = append(out, agent.AgentTool{
			Name:        def.Name,
			Label:       def.Name,
			Description: description,
			Parameters:  parameters,
			Execute: func(ctx context.Context, _ string, params map[string]any, _ agent.ToolUpdateFunc) (agent.AgentToolResult, error) {
				if s.cfg.Agency == nil {
					return text("Plugin tool failed: the agency client is not configured"), nil
				}

				argsJSON, err := json.Marshal(dropNilValues(params))
				if err != nil {
					return text("Plugin tool failed: " + err.Error()), nil
				}
				res, err := s.cfg.Agency.Plugins.InvokePluginTool(
					ctx,
					&agencyv1.InvokePluginToolRequest{
						TaskId:   s.agencyTaskID,
						Tool:     def.Name,
						ArgsJson: string(argsJSON),
					},
				)
				if err != nil {
					return text("Plugin tool failed: " + err.Error()), nil
				}
				if res.GetText() != "" {
					return text(res.GetText()), nil
				}
				return text("OK"), nil
			},
		})
	}
	return out
}

func (s *Session) tools() []agent.AgentTool {
	if s.fixedTools != nil {
		return s.fixedTools
	}
	out := []agent.AgentTool{
		s.setTaskURLs(),
		s.setTaskArtifacts(),
		s.uploadAttachment(),
		s.reportTaskProgress(),
		s.publishSite(),
		s.editSite(),

		s.listTaskArtifactsTool(),
		s.getTaskArtifactTool(),
		s.fetchTaskArtifactTool(),
		s.fetchAttachmentTool(),
		s.downloadPublishedSiteZipTool(),
	}

	if s.interactive {
		out = append(out, s.askUserQuestion())
	}
	out = append(out, s.agencyTools()...)
	out = append(out, s.knowledgeTools()...)
	out = append(out, s.inputTools()...)
	out = append(out, s.pluginAgentTools()...)
	return out
}
