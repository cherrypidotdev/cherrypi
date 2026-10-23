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
import { Type } from "typebox";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/**
 * CherryPi metadata and tools.
 *
 * Registers the task/project tools a running task can call back into the
 * agency: `listTasks`, `getTask`, `getTaskHistory`, `createTask`, `updateTask`,
 * and `listProjects`. The agent injects `CHERRYPI_AGENCY_URL` and the path to
 * its personal access token (`CHERRYPI_AGENT_TOKEN_FILE`); the token grants
 * only what the agent's per-project permissions and the "own task" exception
 * allow. The token is read from the file on every call, so a rotation reaches
 * a running session without a Pi respawn.
 */
export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "listTasks",
    label: "List Tasks",
    description:
      "List CherryPi tasks. With a projectId, lists that project's tasks when this agent has the can-list permission (otherwise only the agent's own tasks in it); without one, lists only this agent's own tasks. Optionally filter by assigned agent id, state, or a free-text search.",
    parameters: Type.Object({
      projectId: Type.Optional(
        Type.String({ description: "Project id or slug to list tasks in." }),
      ),
      agentId: Type.Optional(
        Type.String({ description: "Only tasks assigned to this agent id." }),
      ),
      state: Type.Optional(
        Type.String({
          description:
            "Only tasks in this state (draft, backlog, queued, running, settled, input, failed, completed, cancelled).",
        }),
      ),
      search: Type.Optional(
        Type.String({
          description: "Match task names, prompts, task ids, or numbers.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      if (params.projectId) query.set("project_id", String(params.projectId));
      if (params.agentId) query.set("agent_id", String(params.agentId));
      if (params.state) query.set("state", String(params.state));
      if (params.search) query.set("search", String(params.search));
      return request("GET", `/tasks?${query.toString()}`);
    },
  });

  pi.registerTool({
    name: "getTask",
    label: "Get Task",
    description:
      "Get one CherryPi task by id: its prompt, state, priority, size, tags, references, links, and assignee. The conversation history is not included; read it with getTaskHistory. An agent may always read its own tasks, and another task when it has the can-list permission in the task's project.",
    parameters: Type.Object({
      taskId: Type.String({ description: "The task id to read." }),
    }),
    execute: async (_toolCallId, params) => {
      return request(
        "GET",
        `/tasks/${encodeURIComponent(String(params.taskId ?? ""))}`,
      );
    },
  });

  pi.registerTool({
    name: "getTaskHistory",
    label: "Get Task History",
    description:
      "Read one page of a CherryPi task's persisted conversation, newest page first: user prompts, assistant text and thinking, and tool calls. Events within a page stay chronological; pass the returned next_cursor as cursor to page back to older entries. An agent may always read its own tasks, and another task when it has the can-list permission in the task's project.",
    parameters: Type.Object({
      taskId: Type.String({ description: "The task id to read." }),
      limit: Type.Optional(
        Type.Number({
          description:
            "Maximum entries per page; defaults to 200 and caps at 1000.",
        }),
      ),
      cursor: Type.Optional(
        Type.String({
          description:
            "Cursor from a previous page's next_cursor; omit for the newest page.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      if (params.limit !== undefined) {
        query.set("limit", String(params.limit));
      }
      if (params.cursor) query.set("cursor", String(params.cursor));
      const suffix = query.toString();
      return request(
        "GET",
        `/tasks/${encodeURIComponent(String(params.taskId ?? ""))}/history` +
          (suffix ? `?${suffix}` : ""),
      );
    },
  });

  pi.registerTool({
    name: "createTask",
    label: "Create Task",
    description:
      "Create a CherryPi task in a project this agent can create in. `state` is required and must be `draft` or `backlog`: an agent cannot queue work directly. Optionally assign it to a member agent, add tags or references, name it, and set its priority or size.",
    parameters: Type.Object({
      projectId: Type.String({ description: "Project id or slug." }),
      prompt: Type.String({ description: "The task prompt." }),
      state: Type.Union([Type.Literal("draft"), Type.Literal("backlog")], {
        description:
          "Required initial state. `draft` is never scheduled; `backlog` is likely to run but not queued yet.",
      }),
      assigneeId: Type.Optional(
        Type.String({
          description:
            "Actor id to assign the task to. Use the member's `actor_id` from listProjects. Omit to let the scheduler pick a member agent.",
        }),
      ),
      name: Type.Optional(
        Type.String({ description: "Optional display name." }),
      ),
      priority: Type.Optional(
        Type.Union([
          Type.Literal("low"),
          Type.Literal("normal"),
          Type.Literal("high"),
          Type.Literal("critical"),
        ]),
      ),
      size: Type.Optional(
        Type.Union([
          Type.Literal("small"),
          Type.Literal("medium"),
          Type.Literal("big"),
          Type.Literal("epic"),
        ]),
      ),
      tagIds: Type.Optional(
        Type.Array(Type.String(), { description: "Tag ids from the project." }),
      ),
      references: Type.Optional(
        Type.Array(
          Type.Object({
            task_id: Type.String({ description: "Referenced task id." }),
            kind: Type.Optional(
              Type.Union(
                [
                  Type.Literal("blocking"),
                  Type.Literal("informational"),
                  Type.Literal("supersedes"),
                ],
                {
                  description:
                    "Edge kind. `blocking` gates this task's queuing; `supersedes` cancels the referenced task when it is in draft, backlog, queued, or failed. Defaults to informational.",
                },
              ),
            ),
          }),
          {
            description:
              "Tasks this task references; blocking references gate queuing.",
          },
        ),
      ),
    }),
    execute: async (_toolCallId, params) => {
      return request("POST", "/tasks", {
        project_id: String(params.projectId ?? ""),
        prompt: String(params.prompt ?? ""),
        assignee_actor_id: params.assigneeId ? String(params.assigneeId) : "",
        name: params.name ? String(params.name) : "",
        priority: priorityValue(params.priority),
        size: params.size ? String(params.size) : "",
        tag_ids: params.tagIds ?? [],
        references: (params.references ?? []).map((ref) => ({
          task_id: String(ref.task_id ?? ""),
          kind: ref.kind ? String(ref.kind) : "informational",
        })),
        state: String(params.state ?? ""),
      });
    },
  });

  pi.registerTool({
    name: "updateTask",
    label: "Update Task",
    description:
      "Update a CherryPi task's name, links, tags, references, priority, size, or assignee. The agent may always update the task it is running; updating any other task needs the can-update permission in its project. Links, tag ids, and references replace the existing list.",
    parameters: Type.Object({
      taskId: Type.String({ description: "The task id to update." }),
      name: Type.Optional(Type.String({ description: "New display name." })),
      links: Type.Optional(
        Type.Array(
          Type.Object({
            url: Type.String({ description: "Absolute http(s) URL." }),
            label: Type.String({ description: 'Short label, e.g. "PR".' }),
          }),
          { description: "The complete link list; empty clears it." },
        ),
      ),
      tagIds: Type.Optional(
        Type.Array(Type.String(), {
          description: "The complete tag id list; empty clears it.",
        }),
      ),
      references: Type.Optional(
        Type.Array(
          Type.Object({
            task_id: Type.String({ description: "Referenced task id." }),
            kind: Type.Optional(
              Type.Union(
                [
                  Type.Literal("blocking"),
                  Type.Literal("informational"),
                  Type.Literal("supersedes"),
                ],
                {
                  description:
                    "Edge kind. `blocking` gates this task's queuing; `supersedes` cancels the referenced task when it is in draft, backlog, queued, or failed.",
                },
              ),
            ),
          }),
          {
            description:
              "The complete reference list; an empty list clears it.",
          },
        ),
      ),
      priority: Type.Optional(
        Type.Union([
          Type.Literal("low"),
          Type.Literal("normal"),
          Type.Literal("high"),
          Type.Literal("critical"),
        ]),
      ),
      size: Type.Optional(
        Type.Union([
          Type.Literal("small"),
          Type.Literal("medium"),
          Type.Literal("big"),
          Type.Literal("epic"),
        ]),
      ),
      assigneeId: Type.Optional(
        Type.String({
          description:
            "Pin the task to this actor id; empty clears the pin. Use the member's `actor_id` from listProjects.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const body: Record<string, unknown> = {};
      if (params.name !== undefined) body.name = String(params.name);
      if (params.links !== undefined) body.links = params.links;
      if (params.tagIds !== undefined) body.tag_ids = params.tagIds;
      if (params.references !== undefined)
        body.references = params.references.map((ref) => ({
          task_id: String(ref.task_id ?? ""),
          kind: ref.kind ? String(ref.kind) : "informational",
        }));
      if (params.priority !== undefined)
        body.priority = priorityValue(params.priority);
      if (params.size !== undefined) body.size = String(params.size);
      if (params.assigneeId !== undefined)
        body.assignee_actor_id = String(params.assigneeId);
      return request(
        "PATCH",
        `/tasks/${encodeURIComponent(String(params.taskId ?? ""))}`,
        body,
      );
    },
  });

  pi.registerTool({
    name: "listProjects",
    label: "List Projects",
    description:
      "List the CherryPi projects this agent is a member of, including each project's description, this agent's permissions in it, and every member (name, username, description, permissions).",
    parameters: Type.Object({}),
    execute: async () => request("GET", "/projects"),
  });
}

function agencyUrl(): string {
  return (process.env.CHERRYPI_AGENCY_URL ?? "").replace(/\/+$/, "");
}

/**
 * Reads the agent's personal access token, preferring the token file so a
 * rotation applies without a Pi respawn, then falling back to the inherited
 * environment.
 */
async function token(): Promise<string> {
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

function priorityValue(value: unknown): number | undefined {
  switch (value) {
    case "low":
      return -1;
    case "high":
      return 1;
    case "critical":
      return 2;
    case "normal":
      return 0;
    default:
      return undefined;
  }
}

async function request(
  method: string,
  path: string,
  body?: Record<string, unknown>,
) {
  const base = agencyUrl();
  const bearer = await token();
  if (!base || !bearer) {
    return text("CherryPi task tools are not configured for this task.");
  }

  let res: Response;
  try {
    res = await fetch(`${base}/api/rest/agent-tools${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${bearer}`,
        ...(body ? { "Content-Type": "application/json" } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (err) {
    return text(`Request failed: ${String(err)}`);
  }

  const raw = await res.text();
  if (!res.ok) {
    let message = raw;
    try {
      message = (JSON.parse(raw) as { error?: string }).error ?? raw;
    } catch {
      // Keep the raw body.
    }
    return text(`CherryPi request failed (${res.status}): ${message}`);
  }
  return text(raw || "OK");
}

function text(value: string) {
  return {
    content: [{ type: "text" as const, text: value }],
    details: undefined,
  };
}
