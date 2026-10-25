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

import { readFile } from "node:fs/promises";
import { basename, isAbsolute, relative, resolve } from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "setTaskArtifacts",
    label: "Set Task Artifacts",
    description:
      "Publish this task's durable output artifacts (A2A artifacts): each has an id, a name, an optional description, and parts. Use `file` parts for workspace files (they are uploaded for you), `text` for inline text, and `data` for structured JSON. Artifacts represent the task's result and stay in its Artifacts tab; for transient, current-status context use uploadAttachment instead. By default the call replaces the task's whole artifact set; pass replace:false to update only the listed artifacts, and append:true with lastChunk:false to stream parts into one artifact across calls.",
    parameters: Type.Object({
      replace: Type.Optional(
        Type.Boolean({
          description:
            "Replace the task's complete artifact set (default true). Set false to update just the listed artifacts.",
        }),
      ),
      artifacts: Type.Array(
        Type.Object({
          id: Type.Optional(
            Type.String({
              description:
                "Stable artifact id, unique within the task. Omit to let the agency generate one.",
            }),
          ),
          name: Type.String({ description: "Human-readable artifact name." }),
          description: Type.Optional(
            Type.String({ description: "What the artifact contains." }),
          ),
          append: Type.Optional(
            Type.Boolean({
              description:
                "Append these parts to the existing artifact with this id instead of replacing them.",
            }),
          ),
          lastChunk: Type.Optional(
            Type.Boolean({
              description:
                "Set false while more parts are still coming; defaults to true (complete).",
            }),
          ),
          parts: Type.Array(
            Type.Object({
              file: Type.Optional(
                Type.String({
                  description:
                    "Workspace path of a file to upload as a file part.",
                }),
              ),
              url: Type.Optional(
                Type.String({
                  description: "A URL to reference as a file part.",
                }),
              ),
              text: Type.Optional(
                Type.String({ description: "Inline text part." }),
              ),
              data: Type.Optional(
                Type.Any({
                  description: "Structured JSON data part.",
                }),
              ),
              filename: Type.Optional(
                Type.String({ description: "Filename for a file part." }),
              ),
              mediaType: Type.Optional(
                Type.String({ description: "Media type for a file part." }),
              ),
            }),
            { description: "The artifact's parts, in order." },
          ),
        }),
        { description: "The artifacts to publish." },
      ),
    }),
    execute: async (_toolCallId, params) => {
      const agencyUrl = (process.env.CHERRYPI_AGENCY_URL ?? "").replace(
        /\/+$/,
        "",
      );
      const token = await agentToken();
      const taskId = process.env.CHERRYPI_TASK_ID ?? "";
      if (!agencyUrl || !token || !taskId) {
        return text("Artifact publishing is not configured for this task.");
      }

      const workspace = process.cwd();
      const artifacts = [];

      for (const artifact of params.artifacts ?? []) {
        const parts = [];
        for (const part of artifact.parts ?? []) {
          if (typeof part.file === "string" && part.file.trim() !== "") {
            const uploaded = await uploadFile(
              agencyUrl,
              token,
              taskId,
              workspace,
              part.file.trim(),
            );
            if (typeof uploaded === "string") {
              return text(uploaded);
            }
            parts.push({
              kind: "file",
              attachment_id: Number(uploaded.id),
              url: uploaded.url,
              filename: uploaded.filename ?? basename(part.file),
              media_type:
                part.mediaType ??
                uploaded.contentType ??
                "application/octet-stream",
            });
          } else if (typeof part.url === "string" && part.url.trim() !== "") {
            parts.push({
              kind: "file",
              url: part.url.trim(),
              filename: part.filename ?? "",
              media_type: part.mediaType ?? "",
            });
          } else if (part.data !== undefined && part.data !== null) {
            parts.push({ kind: "data", data: JSON.stringify(part.data) });
          } else if (typeof part.text === "string") {
            parts.push({ kind: "text", text: part.text });
          }
        }

        artifacts.push({
          id: artifact.id ?? "",
          name: artifact.name,
          description: artifact.description ?? "",
          append: artifact.append ?? false,
          last_chunk: artifact.lastChunk ?? true,
          parts,
        });
      }

      pi.appendEntry("cherrypi_task_artifacts", {
        replace: params.replace ?? true,
        artifacts,
      });

      const names = artifacts.map((a) => a.name).join(", ");
      return text(
        artifacts.length === 0
          ? "Cleared this task's artifacts."
          : `Published ${artifacts.length} artifact(s): ${names}.`,
      );
    },
  });
}

async function uploadFile(
  agencyUrl: string,
  token: string,
  taskId: string,
  workspace: string,
  raw: string,
): Promise<
  { id: number; url: string; filename?: string; contentType?: string } | string
> {
  const filePath = isAbsolute(raw) ? raw : resolve(workspace, raw);
  const rel = relative(workspace, filePath);
  if (rel.startsWith("..") || isAbsolute(rel)) {
    return "Refusing to upload a file outside the task workspace.";
  }

  let data: Buffer;
  try {
    data = await readFile(filePath);
  } catch (err) {
    return `Could not read ${raw}: ${String(err)}`;
  }

  const form = new FormData();
  form.append("file", new Blob([data]), basename(filePath));
  const query = new URLSearchParams({ kind: "artifact", id: taskId });

  let res: Response;
  try {
    res = await fetch(`${agencyUrl}/api/rest/attachments?${query}`, {
      method: "POST",
      body: form,
      headers: { Authorization: `Bearer ${token}` },
    });
  } catch (err) {
    return `Upload failed for ${raw}: ${String(err)}`;
  }
  if (!res.ok) {
    return `Upload failed for ${raw}: ${res.status} ${res.statusText}`;
  }

  const body = (await res.json()) as {
    id?: string | number;
    url?: string;
    filename?: string;
    contentType?: string;
  };
  return {
    id: Number(body.id ?? 0),
    url: body.url ?? "",
    filename: body.filename,
    contentType: body.contentType,
  };
}

function text(value: string) {
  return {
    content: [{ type: "text" as const, text: value }],
    details: undefined,
  };
}

async function agentToken(): Promise<string> {
  const file = process.env.CHERRYPI_AGENT_TOKEN_FILE ?? "";
  if (file) {
    try {
      const raw = (await readFile(file, "utf8")).trim();
      if (raw) return raw;
    } catch {}
  }
  return process.env.CHERRYPI_AGENT_TOKEN ?? "";
}
