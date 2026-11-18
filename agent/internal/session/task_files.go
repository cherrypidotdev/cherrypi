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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
	"google.golang.org/protobuf/encoding/protojson"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

func (s *Session) fetchCacheRoot() string {
	if dir := strings.TrimSpace(os.Getenv("CHERRYPI_CACHE_DIR")); dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), "cherrypi-cache")
}

func (s *Session) fetchToCache(
	ctx context.Context,
	attachmentID int64,
	filename, url string,
) (string, error) {
	if attachmentID == 0 && strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("the part has no downloadable content")
	}
	if s.cfg.FetchAttachment == nil {
		return "", fmt.Errorf("attachments are not configured on this agent")
	}
	name := sanitizeFetchName(filename)
	if name == "" {
		name = "download"
	}
	dir := filepath.Join(s.fetchCacheRoot(), fmt.Sprintf("%d", attachmentID))
	if attachmentID == 0 {
		dir = filepath.Join(s.fetchCacheRoot(), "url", sanitizeFetchName(url))
	}
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	body, err := s.cfg.FetchAttachment(ctx, url)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	defer func() { _ = body.Close() }()
	temp, err := os.CreateTemp(dir, name+".*")
	if err != nil {
		return "", err
	}
	discard := func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}
	if _, err := io.Copy(temp, body); err != nil {
		discard()
		return "", err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(temp.Name())
		return "", err
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		_ = os.Remove(temp.Name())
		return "", err
	}
	return path, nil
}

func sanitizeFetchName(raw string) string {
	base := filepath.Base(strings.TrimSpace(raw))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func (s *Session) listTaskArtifactsTool() agent.AgentTool {
	return s.agencyTool(
		"listTaskArtifacts",
		"List Task Artifacts",
		"List the durable artifacts a task published: id, name, description, "+
			"part count, media types, and whether more parts are streaming. It "+
			"reads a short list, not the parts; use getTaskArtifact for one "+
			"artifact's full content, or fetchTaskArtifact to save its files "+
			"locally.",
		ai.Object(
			ai.Opt("taskId", ai.String("The task id; defaults to this task.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			task, err := s.readTask(ctx, a.str("taskId"))
			if err != nil {
				return toolOutput{}, err
			}
			list := make([]map[string]any, 0, len(task.GetArtifacts()))
			for _, artifact := range task.GetArtifacts() {
				mediaTypes := make([]string, 0)
				files := 0
				for _, part := range artifact.GetParts() {
					if part.GetFile() != nil {
						files++
					}
					if mt := part.GetMediaType(); mt != "" {
						mediaTypes = append(mediaTypes, mt)
					}
				}
				list = append(list, map[string]any{
					"artifact_id": artifact.GetArtifactId(),
					"name":        artifact.GetName(),
					"description": artifact.GetDescription(),
					"parts":       len(artifact.GetParts()),
					"files":       files,
					"media_types": mediaTypes,
					"streaming":   artifact.GetStreaming(),
					"updated_at": artifact.GetUpdatedAt().
						AsTime().
						Format("2006-01-02T15:04:05Z07:00"),
				})
			}
			return jsonOutput(map[string]any{
				"task_id":   task.GetId(),
				"artifacts": list,
			})
		},
	)
}

func (s *Session) getTaskArtifactTool() agent.AgentTool {
	return s.agencyTool(
		"getTaskArtifact",
		"Get Task Artifact",
		"Read one artifact in full: every text part, file part (with its "+
			"attachment id and content URL), and structured-data part, plus its "+
			"metadata. Use fetchTaskArtifact to save the file parts locally "+
			"instead.",
		ai.Object(
			ai.Prop("artifactId", ai.String("The artifact id within the task.")),
			ai.Opt("taskId", ai.String("The task id; defaults to this task.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			task, err := s.readTask(ctx, a.str("taskId"))
			if err != nil {
				return toolOutput{}, err
			}
			id := a.str("artifactId")
			for _, artifact := range task.GetArtifacts() {
				if artifact.GetArtifactId() != id {
					continue
				}

				raw, err := protojson.Marshal(artifact)
				if err != nil {
					return toolOutput{}, err
				}
				return toolOutput{raw: raw}, nil
			}
			return toolOutput{}, fmt.Errorf(
				"task %s has no artifact %q; list them with listTaskArtifacts",
				task.GetId(), id,
			)
		},
	)
}

func (s *Session) fetchTaskArtifactTool() agent.AgentTool {
	return s.agencyTool(
		"fetchTaskArtifact",
		"Fetch Task Artifact",
		"Save an artifact's file parts to this machine and return their local "+
			"paths. The bytes are cached by attachment, so fetching the same "+
			"artifact again is free. Read a returned path with the usual file "+
			"tools; text and data parts are already inline, so read them with "+
			"getTaskArtifact.",
		ai.Object(
			ai.Prop("artifactId", ai.String("The artifact id within the task.")),
			ai.Opt("taskId", ai.String("The task id; defaults to this task.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			task, err := s.readTask(ctx, a.str("taskId"))
			if err != nil {
				return toolOutput{}, err
			}
			id := a.str("artifactId")
			for _, artifact := range task.GetArtifacts() {
				if artifact.GetArtifactId() != id {
					continue
				}
				paths := make([]map[string]any, 0)
				for _, part := range artifact.GetParts() {
					file := part.GetFile()
					if file == nil {
						continue
					}
					path, err := s.fetchToCache(
						ctx,
						file.GetAttachmentId(),
						part.GetFilename(),
						file.GetUrl(),
					)
					if err != nil {
						return toolOutput{}, err
					}
					paths = append(paths, map[string]any{
						"filename":   part.GetFilename(),
						"media_type": part.GetMediaType(),
						"path":       path,
					})
				}
				if len(paths) == 0 {
					return toolOutput{}, fmt.Errorf(
						"artifact %q has no file parts; read it with getTaskArtifact",
						id,
					)
				}
				return jsonOutput(map[string]any{
					"artifact_id": id,
					"files":       paths,
				})
			}
			return toolOutput{}, fmt.Errorf(
				"task %s has no artifact %q; list them with listTaskArtifacts",
				task.GetId(), id,
			)
		},
	)
}

func (s *Session) fetchAttachmentTool() agent.AgentTool {
	return s.agencyTool(
		"fetchAttachment",
		"Fetch Attachment",
		"Save one of the task's attachments (its file tray) to this machine and "+
			"return the local path. The bytes are cached by attachment, so a "+
			"second fetch is free. Use it after a machine switch, when the "+
			"workspace no longer holds the files the conversation mentions.",
		ai.Object(
			ai.Prop("attachmentId", ai.String("The attachment id to fetch.")),
			ai.Opt("taskId", ai.String("The task id; defaults to this task.")),
		),
		func(ctx context.Context, a args) (toolOutput, error) {
			task, err := s.readTask(ctx, a.str("taskId"))
			if err != nil {
				return toolOutput{}, err
			}
			want := a.str("attachmentId")
			if want == "" {
				return toolOutput{}, fmt.Errorf("provide an attachmentId")
			}
			for _, attachment := range task.GetAttachments() {
				if fmt.Sprint(attachment.GetId()) != want {
					continue
				}
				path, err := s.fetchToCache(
					ctx,
					attachment.GetId(),
					attachment.GetFilename(),
					attachment.GetUrl(),
				)
				if err != nil {
					return toolOutput{}, err
				}
				return jsonOutput(map[string]any{
					"attachment_id": attachment.GetId(),
					"filename":      attachment.GetFilename(),
					"content_type":  attachment.GetContentType(),
					"path":          path,
				})
			}
			return toolOutput{}, fmt.Errorf(
				"task %s has no attachment %q; read the task to list its tray",
				task.GetId(), want,
			)
		},
	)
}

func (s *Session) downloadPublishedSiteZipTool() agent.AgentTool {
	return s.agencyTool(
		"downloadPublishedSiteZip",
		"Download Published Site Zip",
		"Save the source zip of a published site to this machine and return the "+
			"local path. Publishing needs the zip in the workspace, so a run "+
			"resumed on another machine downloads it first, edits the files, and "+
			"republishes.",
		ai.Object(ai.Prop("siteId", ai.String("The published site's slug."))),
		func(ctx context.Context, a args) (toolOutput, error) {
			client, err := s.agencyClient()
			if err != nil {
				return toolOutput{}, err
			}
			id := a.str("siteId")
			if id == "" {
				return toolOutput{}, fmt.Errorf("provide a siteId")
			}
			res, err := client.Sites.GetSite(ctx, &agencyv1.GetSiteRequest{Id: id})
			if err != nil {
				return toolOutput{}, err
			}
			site := res.GetSite()
			attachmentID := site.GetSourceZipAttachmentId()
			if attachmentID == 0 {
				return toolOutput{}, fmt.Errorf(
					"site %q has no source zip", id,
				)
			}
			path, err := s.fetchToCache(
				ctx,
				attachmentID,
				id+".zip",
				attachmentURL(attachmentID),
			)
			if err != nil {
				return toolOutput{}, err
			}
			return jsonOutput(map[string]any{
				"site_id": id,
				"path":    path,
			})
		},
	)
}

func (s *Session) readTask(
	ctx context.Context,
	explicit string,
) (*agencyv1.Task, error) {
	id := strings.TrimSpace(explicit)
	if id == "" {
		id = s.agencyTaskID
	}
	if id == "" {
		return nil, fmt.Errorf("this session has no task; pass taskId")
	}
	client, err := s.agencyClient()
	if err != nil {
		return nil, err
	}
	res, err := client.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return res.GetTask(), nil
}

func attachmentURL(id int64) string {
	return fmt.Sprintf("/api/rest/attachments/%d/content", id)
}
