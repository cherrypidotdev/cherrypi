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
import {
  agentTokens,
  credentialNote,
  credentialsLine,
  fetchWithAgentToken,
} from "./agent_token.ts";

export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "publishSite",
    label: "Publish Site",
    description:
      "Publish a static website from a zip archive in this task's workspace. The archive must have index.html at its root and contain only regular files (no symlinks or ../ paths), up to 50 MiB. `id` is an agency-unique slug ([a-z0-9][a-z0-9-]{0,63}); publishing the same id replaces the contents and settings. Returns the published site URL and makes a Site tab available in this task's console. You are encouraged to use this built-in feature to demonstrate your findings and present visual proposals, such as landing pages, dashboards, report pages, and interactive prototypes, directly beside the task chat. Sites are served from a sandboxed iframe with an opaque origin: build with a relative base (Vite: base: \"./\") because root-absolute asset paths resolve against the agency and return its HTML shell; use window.cherrypi for auth; localStorage/sessionStorage work through a shell-backed shim, while cookies, IndexedDB and service workers do not.",
    parameters: Type.Object({
      id: Type.String({
        description:
          "Agency-unique site id, lower-kebab, e.g. `landing-page`. Publishing the same id updates that site.",
      }),
      archiveLocalPath: Type.String({
        description:
          "Path to a .zip archive inside the task workspace (absolute or relative to the workspace).",
      }),
      settings: Type.Optional(
        Type.Object({
          title: Type.Optional(
            Type.String({ description: "Human-readable site title." }),
          ),
          description: Type.Optional(
            Type.String({ description: "What the site contains." }),
          ),
          onlyCreatorMayEdit: Type.Optional(
            Type.Boolean({
              description:
                "When true (default), only this task's agent may republish or edit through the tools.",
            }),
          ),
          agencyWide: Type.Optional(
            Type.Boolean({
              description:
                "When true, the site is listed under the sidebar's Sites section for signed-in users. Default false.",
            }),
          ),
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const agencyUrl = (process.env.CHERRYPI_AGENCY_URL ?? "").replace(
        /\/+$/,
        "",
      );
      const tokens = await agentTokens();
      const taskId = process.env.CHERRYPI_TASK_ID ?? "";
      if (!agencyUrl || tokens.length === 0 || !taskId) {
        return text("Site publishing is not configured for this task.");
      }

      const id = String(params.id ?? "").trim();
      if (!/^[a-z0-9][a-z0-9-]{0,63}$/.test(id)) {
        return text(
          "The site id must be lower-kebab: [a-z0-9][a-z0-9-]{0,63}.",
        );
      }

      const raw = String(params.archiveLocalPath ?? "").trim();
      if (!raw) return text("Provide the path of the zip archive to publish.");
      const workspace = process.cwd();
      const filePath = isAbsolute(raw) ? raw : resolve(workspace, raw);
      const rel = relative(workspace, filePath);
      if (rel.startsWith("..") || isAbsolute(rel)) {
        return text("Refusing to publish a file outside the task workspace.");
      }

      let data: Buffer;
      try {
        data = await readFile(filePath);
      } catch (err) {
        return text(`Could not read ${raw}: ${String(err)}`);
      }

      const form = new FormData();
      form.append("file", new Blob([data]), basename(filePath));
      form.append("origin_task_id", taskId);
      const settings = params.settings ?? {};
      if (settings.title !== undefined)
        form.append("title", String(settings.title));
      if (settings.description !== undefined)
        form.append("description", String(settings.description));
      if (settings.onlyCreatorMayEdit !== undefined)
        form.append(
          "only_creator_may_edit",
          settings.onlyCreatorMayEdit ? "true" : "false",
        );
      if (settings.agencyWide !== undefined)
        form.append("agency_wide", settings.agencyWide ? "true" : "false");

      const outcome = await fetchWithAgentToken(
        `${agencyUrl}/api/rest/sites/${encodeURIComponent(id)}/publish`,
        { method: "POST", body: form },
        tokens,
      );
      const res = outcome.res;
      if (!res) {
        return text(
          `Publish failed: ${outcome.error ?? "unknown error"}${credentialsLine(outcome)}`,
        );
      }
      if (!res.ok) {
        return text(
          `Publish failed: ${res.status} ${await res.text()}${credentialsLine(outcome)}`,
        );
      }
      const site = (await res.json()) as {
        url?: string;
        title?: string;
        warnings?: string[];
      };
      const url = site.url ?? `/sites/${id}`;
      const warnings = Array.isArray(site.warnings) ? site.warnings : [];
      const warningText = warnings.length
        ? `\n\nWarnings:\n${warnings.map((warning) => `- ${warning}`).join("\n")}`
        : "";
      return text(
        `Published ${site.title || id} at ${agencyUrl}${url}. This task now has a Site tab.${warningText}` +
          credentialNote(outcome),
      );
    },
  });

  pi.registerTool({
    name: "editSite",
    label: "Edit Site Settings",
    description:
      "Change a published site's settings without replacing its contents: title, description, onlyCreatorMayEdit, and/or agencyWide. Only the site's creator, or any agent when the creator allowed collaborative editing, may call it.",
    parameters: Type.Object({
      id: Type.String({ description: "The published site id." }),
      updatedSettings: Type.Object({
        title: Type.Optional(
          Type.String({ description: "Human-readable site title." }),
        ),
        description: Type.Optional(
          Type.String({ description: "What the site contains." }),
        ),
        onlyCreatorMayEdit: Type.Optional(
          Type.Boolean({
            description:
              "When true, only the creating agent may republish or edit through the tools.",
          }),
        ),
        agencyWide: Type.Optional(
          Type.Boolean({
            description:
              "When true, list the site under the sidebar's Sites section.",
          }),
        ),
      }),
    }),
    execute: async (_toolCallId, params) => {
      const agencyUrl = (process.env.CHERRYPI_AGENCY_URL ?? "").replace(
        /\/+$/,
        "",
      );
      const tokens = await agentTokens();
      if (!agencyUrl || tokens.length === 0) {
        return text("Site editing is not configured for this task.");
      }
      const id = String(params.id ?? "").trim();
      if (!id) return text("Provide the site id to edit.");

      const body: Record<string, unknown> = {};
      const updated = params.updatedSettings ?? {};
      if (updated.title !== undefined) body.title = String(updated.title);
      if (updated.description !== undefined)
        body.description = String(updated.description);
      if (updated.onlyCreatorMayEdit !== undefined)
        body.only_creator_may_edit = Boolean(updated.onlyCreatorMayEdit);
      if (updated.agencyWide !== undefined)
        body.agency_wide = Boolean(updated.agencyWide);

      const outcome = await fetchWithAgentToken(
        `${agencyUrl}/api/rest/sites/${encodeURIComponent(id)}`,
        {
          method: "PATCH",
          body: JSON.stringify(body),
          headers: { "Content-Type": "application/json" },
        },
        tokens,
      );
      const res = outcome.res;
      if (!res) {
        return text(
          `Edit failed: ${outcome.error ?? "unknown error"}${credentialsLine(outcome)}`,
        );
      }
      if (!res.ok) {
        return text(
          `Edit failed: ${res.status} ${await res.text()}${credentialsLine(outcome)}`,
        );
      }
      const site = (await res.json()) as { url?: string; title?: string };
      return text(
        `Updated ${site.title || id} at ${agencyUrl}${site.url ?? `/sites/${id}`}.` +
          credentialNote(outcome),
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
