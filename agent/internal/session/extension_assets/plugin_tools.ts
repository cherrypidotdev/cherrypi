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

import { Type } from "typebox";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import {
  agentTokens,
  credentialNote,
  credentialsLine,
  fetchWithAgentToken,
} from "./agent_token.ts";

interface PluginTool {
  name: string;
  description?: string;
  input_schema?: string;
}

export default function (pi: ExtensionAPI) {
  const tools = pluginTools();
  for (const tool of tools) {
    pi.registerTool({
      name: tool.name,
      label: tool.name,
      description: tool.description || `Plugin tool ${tool.name}.`,
      parameters: parametersFor(tool),
      execute: async (_toolCallId, params) => {
        return callPluginTool(tool.name, params as Record<string, unknown>);
      },
    });
  }
}

function pluginTools(): PluginTool[] {
  const raw = process.env.CHERRYPI_PLUGIN_TOOLS ?? "";
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as PluginTool[];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (tool) => typeof tool?.name === "string" && tool.name.length > 0,
    );
  } catch {
    return [];
  }
}

function parametersFor(tool: PluginTool) {
  try {
    const schema = JSON.parse(tool.input_schema || '{"type":"object"}');
    return Type.Unsafe(schema);
  } catch {
    return Type.Unsafe({ type: "object" });
  }
}

async function callPluginTool(tool: string, args: Record<string, unknown>) {
  const base = (process.env.CHERRYPI_AGENCY_URL ?? "").replace(/\/+$/, "");
  const taskId = process.env.CHERRYPI_TASK_ID ?? "";
  const tokens = await agentTokens();
  if (!base || tokens.length === 0 || !taskId) {
    return text("The plugin tool bridge is not configured for this task.");
  }

  const outcome = await fetchWithAgentToken(
    `${base}/api/rest/agent-tools/plugin-tools`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ task_id: taskId, tool, args }),
    },
    tokens,
  );
  const res = outcome.res;
  if (!res) {
    return text(
      `Plugin tool request failed: ${outcome.error ?? "unknown error"}${credentialsLine(outcome)}`,
    );
  }

  const raw = await res.text();
  if (!res.ok) {
    let message = raw;
    try {
      message = (JSON.parse(raw) as { error?: string }).error ?? raw;
    } catch {}
    return text(
      `Plugin tool failed (${res.status}): ${message}${credentialsLine(outcome)}`,
    );
  }
  const note = credentialNote(outcome);
  try {
    const out = JSON.parse(raw) as { text?: string };
    return text((out.text ?? raw) + note);
  } catch {
    return text(raw + note);
  }
}

function text(value: string) {
  return {
    content: [{ type: "text" as const, text: value }],
    details: undefined,
  };
}
