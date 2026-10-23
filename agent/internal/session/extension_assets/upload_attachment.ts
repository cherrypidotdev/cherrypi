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

/**
 * CherryPi attachment upload.
 *
 * Registers `uploadAttachment`, the tool a running task uses to share a file
 * from its workspace as a task attachment and get back a URL. Attachments are
 * transient, per-message context (a screenshot of current progress, a log tail,
 * a diff): they land in the task's Attachments tray for the current exchange.
 * Durable outputs belong in `setTaskArtifacts`, which owns the Artifacts tab.
 *
 * The agent injects `CHERRYPI_AGENCY_URL`, `CHERRYPI_TASK_ID`, and the path to
 * the agent's personal access token (`CHERRYPI_AGENT_TOKEN_FILE`); the token
 * only grants uploads the agent may already make.
 */
export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "uploadAttachment",
    label: "Upload Attachment",
    description:
      "Upload a file from this task's workspace as a task attachment and get back a URL. Use it for transient, current-status context (a screenshot of what you are doing, a log, a diff) that belongs with the conversation; it appears in the task Attachments tray. Durable task results belong in setTaskArtifacts instead, which owns the long-lived Artifacts tab. The path must be inside the task workspace.",
    parameters: Type.Object({
      path: Type.String({
        description:
          "Path to the file, relative to the task workspace (or absolute inside it).",
      }),
    }),
    execute: async (_toolCallId, params) => {
      const agencyUrl = (process.env.CHERRYPI_AGENCY_URL ?? "").replace(
        /\/+$/,
        "",
      );
      const token = await agentToken();
      const taskId = process.env.CHERRYPI_TASK_ID ?? "";
      if (!agencyUrl || !token || !taskId) {
        return text("Attachment upload is not configured for this task.");
      }

      const raw = String(params.path ?? "").trim();
      if (!raw) {
        return text("Provide the path of the file to upload.");
      }

      const workspace = process.cwd();
      const filePath = isAbsolute(raw) ? raw : resolve(workspace, raw);
      const rel = relative(workspace, filePath);
      if (rel.startsWith("..") || isAbsolute(rel)) {
        return text("Refusing to upload a file outside the task workspace.");
      }

      let data: Buffer;
      try {
        data = await readFile(filePath);
      } catch (err) {
        return text(`Could not read ${raw}: ${String(err)}`);
      }

      const form = new FormData();
      form.append("file", new Blob([data]), basename(filePath));
      const query = new URLSearchParams({ kind: "task", id: taskId });

      let res: Response;
      try {
        res = await fetch(`${agencyUrl}/api/rest/attachments?${query}`, {
          method: "POST",
          body: form,
          headers: { Authorization: `Bearer ${token}` },
        });
      } catch (err) {
        return text(`Upload failed: ${String(err)}`);
      }
      if (!res.ok) {
        return text(`Upload failed: ${res.status} ${res.statusText}`);
      }

      const attachment = (await res.json()) as {
        url?: string;
        filename?: string;
      };
      const url = attachment.url ?? "";
      const name = attachment.filename ?? basename(filePath);
      return text(
        `Uploaded ${name} as a task attachment. Reference it in your reply as a relative link, e.g. ![${name}](${url}). Use setTaskArtifacts for durable results.`,
      );
    },
  });
}

function text(value: string) {
  return {
    content: [{ type: "text" as const, text: value }],
    details: undefined,
  };
}

/**
 * Reads the agent's personal access token, preferring the token file so a
 * rotation applies without a Pi respawn, then falling back to the inherited
 * environment.
 */
async function agentToken(): Promise<string> {
  const file = process.env.CHERRYPI_AGENT_TOKEN_FILE ?? "";
  if (file) {
    try {
      const raw = (await readFile(file, "utf8")).trim();
      if (raw) return raw;
    } catch {
      // Fall through to the environment.
    }
  }
  return process.env.CHERRYPI_AGENT_TOKEN ?? "";
}
