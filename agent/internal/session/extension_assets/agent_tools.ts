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
            "Only tasks in this state (backlog, queued, running, settled, input, failed, completed, cancelled).",
        }),
      ),
      search: Type.Optional(
        Type.String({
          description: "Match task names, prompts, task ids, or numbers.",
        }),
      ),
      parentTaskId: Type.Optional(
        Type.String({
          description: "Only this task's sub-tasks.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      if (params.projectId) query.set("project_id", String(params.projectId));
      if (params.agentId) query.set("agent_id", String(params.agentId));
      if (params.state) query.set("state", String(params.state));
      if (params.search) query.set("search", String(params.search));
      if (params.parentTaskId)
        query.set("parent_task_id", String(params.parentTaskId));
      return request("GET", `/tasks?${query.toString()}`);
    },
  });

  pi.registerTool({
    name: "getTask",
    label: "Get Task",
    description:
      "Get one CherryPi task by id: its prompt, state, priority, size, tags, references, links, assignee, parent, and sub-tasks. The conversation history is not included; read it with getTaskHistory. An agent may always read its own tasks, and another task when it has the can-list permission in the task's project.",
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
      "Read one page of a CherryPi task's persisted conversation, newest page first: user prompts, assistant text and thinking, and tool calls. Events within a page stay chronological; pass the returned next_cursor as cursor to page back to older entries. The page defaults to the latest 20 entries and caps at 1000, but may hold fewer when large messages consume its byte budget. An oversized message keeps its start and end around a `start of message <truncated text> end of message` marker. An agent may always read its own tasks, and another task when it has the can-list permission in the task's project.",
    parameters: Type.Object({
      taskId: Type.String({ description: "The task id to read." }),
      limit: Type.Optional(
        Type.Number({
          description:
            "Maximum entries per page; defaults to 20 and caps at 1000.",
        }),
      ),
      cursor: Type.Optional(
        Type.String({
          description:
            "Cursor from a previous page's next_cursor; omit for the newest page.",
        }),
      ),
    }),
    outputSchema: Type.Object({
      task_id: Type.String(),
      events: Type.Array(
        Type.Object({
          type: Type.String(),
          at: Type.Optional(Type.String()),
          text: Type.Optional(Type.String()),
          tool_call_id: Type.Optional(Type.String()),
          tool_name: Type.Optional(Type.String()),
          phase: Type.Optional(Type.String()),
          args: Type.Optional(Type.String()),
          result: Type.Optional(Type.String()),
          is_error: Type.Optional(Type.Boolean()),
        }),
      ),
      next_cursor: Type.String(),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      query.set("limit", String(params.limit ?? DEFAULT_HISTORY_LIMIT));
      if (params.cursor) query.set("cursor", String(params.cursor));
      return request(
        "GET",
        `/tasks/${encodeURIComponent(String(params.taskId ?? ""))}/history?${query.toString()}`,
        undefined,
        true,
      );
    },
  });

  pi.registerTool({
    name: "createTask",
    label: "Create Task",
    description:
      "Create a CherryPi task in a project this agent can create in. `state` is required and must be `backlog` or `queued`; queuing asks the scheduler to start it immediately. Optionally create it from a project task template with `templateId` and `templateArgs`; an explicit `prompt` is then appended as extra instructions, and explicit fields override the template defaults. Otherwise assign it to a member agent, add tags or references, name it, and set its priority or size.",
    parameters: Type.Object({
      projectId: Type.String({ description: "Project id or slug." }),
      prompt: Type.Optional(
        Type.String({
          description:
            "The task prompt, or extra instructions appended to a template's rendered prompt.",
        }),
      ),
      templateId: Type.Optional(
        Type.String({
          description:
            "Task template id from the project. The agency renders it with templateArgs.",
        }),
      ),
      templateVersion: Type.Optional(
        Type.Number({
          description: "Template version to render; omit or 0 for the latest.",
        }),
      ),
      templateArgs: Type.Optional(
        Type.Record(Type.String(), Type.Any(), {
          description:
            "Template argument values, keyed by argument key. Strings, numbers and booleans are accepted.",
        }),
      ),
      state: Type.Union([Type.Literal("backlog"), Type.Literal("queued")], {
        description:
          "Required initial state. `backlog` is the editable pre-queue state; `queued` starts it as soon as an agent has a slot.",
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
                  Type.Literal("informational"),
                  Type.Literal("blocked_by"),
                  Type.Literal("blocks"),
                  Type.Literal("supersedes"),
                  Type.Literal("superseded_by"),
                  Type.Literal("blocking"),
                ],
                {
                  description:
                    "Edge kind from this task's perspective: `blocked_by` gates this task's queuing, `blocks` makes the referenced task wait for this one, `supersedes` cancels the referenced task, and `superseded_by` cancels this one. Defaults to informational.",
                },
              ),
            ),
          }),
          {
            description:
              "Tasks this task references. `blocked_by` and `blocks` gate queuing; the inverse kinds store the edge on the referenced task.",
          },
        ),
      ),
      parentTaskId: Type.Optional(
        Type.String({
          description:
            "Make this a sub-task of the given task. Defaults to this agent's own task (CHERRYPI_TASK_ID), so a sub-agent call nests automatically. Pass an empty string to create a top-level task.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const ownTask = process.env.CHERRYPI_TASK_ID ?? "";
      const parentTaskId =
        params.parentTaskId !== undefined
          ? String(params.parentTaskId)
          : ownTask;
      const body: Record<string, unknown> = {
        project_id: String(params.projectId ?? ""),
        prompt: String(params.prompt ?? ""),
        template_id: params.templateId ? String(params.templateId) : "",
        template_version: params.templateVersion ?? 0,
        template_args: params.templateArgs ?? {},
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
        parent_task_id: parentTaskId,
      };
      return request("POST", "/tasks", body);
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
                  Type.Literal("informational"),
                  Type.Literal("blocked_by"),
                  Type.Literal("blocks"),
                  Type.Literal("supersedes"),
                  Type.Literal("superseded_by"),
                  Type.Literal("blocking"),
                ],
                {
                  description:
                    "Edge kind from this task's perspective: `blocked_by` gates this task's queuing, `blocks` makes the referenced task wait for this one, `supersedes` cancels the referenced task, and `superseded_by` cancels this one.",
                },
              ),
            ),
          }),
          {
            description:
              "The complete outgoing reference list; an empty list clears it. Incoming edges belong to the task that created them, so an inverse kind here adds an edge this list does not replace.",
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
      state: Type.Optional(
        Type.Union([Type.Literal("completed"), Type.Literal("cancelled")], {
          description:
            "Close the task out: `completed` or `cancelled`. Only a lead or an administrator may close out. Queueing is a separate `queueTask` call.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const body: Record<string, unknown> = {};
      if (params.state !== undefined) body.state = String(params.state);
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
    name: "queueTask",
    label: "Queue Task",
    description:
      "Move a CherryPi task to queued so the scheduler starts it as soon as an agent has a free slot. The agent may queue the task it is running; queueing any other task needs the can-update permission in its project. Queuing a task that holds an incomplete blocking reference or an unfinished sub-task parks it in `waiting` instead (slot-free), and it is queued again automatically when the last blocker finishes — this is how a task waits for the sub-tasks it just created. Without a blocker, a task that already started is queued by sending it a follow-up instead.",
    parameters: Type.Object({
      taskId: Type.String({ description: "The task id to queue." }),
    }),
    execute: async (_toolCallId, params) => {
      return request(
        "POST",
        `/tasks/${encodeURIComponent(String(params.taskId ?? ""))}/queue`,
      );
    },
  });

  pi.registerTool({
    name: "closeOutTask",
    label: "Close Out Task",
    description:
      "Complete or cancel a CherryPi task. Only a lead or an administrator may close out; an agent-created task adds its creator agent as a lead, so an agent can close out the work it spawned. A running task must settle or be stopped first.",
    parameters: Type.Object({
      taskId: Type.String({ description: "The task id to close out." }),
      outcome: Type.Union(
        [Type.Literal("completed"), Type.Literal("cancelled")],
        {
          description:
            "`completed` when the work is done, `cancelled` when abandoned.",
        },
      ),
    }),
    execute: async (_toolCallId, params) => {
      return request(
        "POST",
        `/tasks/${encodeURIComponent(String(params.taskId ?? ""))}/close-out`,
        { outcome: String(params.outcome ?? "") },
      );
    },
  });

  pi.registerTool({
    name: "listAgents",
    label: "List Agents",
    description:
      "Admin agents only. List every agent with its metadata: role, presence, build version, protocol version, the address its tunnel connected from, and last seen time. Read-only.",
    parameters: Type.Object({}),
    execute: async () => request("GET", "/agents"),
  });

  pi.registerTool({
    name: "readAgentLogs",
    label: "Read Agent Logs",
    description:
      "Admin agents only. Read one agent's persisted operational logs (process lines, task/session lifecycle, connectivity), newest first. Unfiltered by default; pass excludeMessages to omit routine lines. Get ids from listAgents. Read-only.",
    parameters: Type.Object({
      agentId: Type.String({
        description: "The numeric agent id from listAgents.",
      }),
      limit: Type.Optional(
        Type.Number({
          description: "Maximum lines; defaults to 200 and caps at 1000.",
        }),
      ),
      minLevel: Type.Optional(
        Type.Union(
          [
            Type.Literal("debug"),
            Type.Literal("info"),
            Type.Literal("warn"),
            Type.Literal("error"),
          ],
          { description: "Only lines at or above this severity." },
        ),
      ),
      beforeId: Type.Optional(
        Type.Number({
          description:
            "Page older than this log id (the last row's id from a previous call).",
        }),
      ),
      excludeMessages: Type.Optional(
        Type.Array(
          Type.String({
            description:
              'A message substring to omit, e.g. "pi started". Case-insensitive.',
          }),
          {
            description:
              'Message substrings to omit. Unfiltered by default; the dashboard uses ["pi started", "pi exited", "pi stopped"].',
          },
        ),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      if (params.limit !== undefined) query.set("limit", String(params.limit));
      if (params.minLevel) query.set("min_level", String(params.minLevel));
      if (params.beforeId !== undefined) {
        query.set("before_id", String(params.beforeId));
      }
      for (const pattern of params.excludeMessages ?? []) {
        query.append("exclude", String(pattern));
      }
      const suffix = query.toString();
      return request(
        "GET",
        `/agents/${encodeURIComponent(String(params.agentId ?? ""))}/logs` +
          (suffix ? `?${suffix}` : ""),
      );
    },
  });

  pi.registerTool({
    name: "getAgencyMetadata",
    label: "Get Agency Metadata",
    description:
      "Admin agents only. Read the agency's own metadata: build version, process id, start time and uptime, plus this agent's role. Read-only.",
    parameters: Type.Object({}),
    execute: async () => request("GET", "/agency"),
  });

  pi.registerTool({
    name: "readAgencyLogs",
    label: "Read Agency Logs",
    description:
      "Admin agents only. Read the tail of the agency's current process log (everything the process wrote to stdout/stderr, timestamp-prefixed). Read-only.",
    parameters: Type.Object({
      lines: Type.Optional(
        Type.Number({
          description:
            "How many newest lines; defaults to 200 and caps at 2000.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      if (params.lines !== undefined) query.set("lines", String(params.lines));
      const suffix = query.toString();
      return request("GET", `/agency/logs${suffix ? `?${suffix}` : ""}`);
    },
  });

  pi.registerTool({
    name: "getPrompt",
    label: "Get Prompt",
    description:
      "Read a versioned CherryPi prompt and whether this agent may propose a change. `scope` is `agent` (your own global prompt), `agent_project` (your prompt for one project), or `project` (a project's shared prompt). Pass `projectId` for the latter two. The response includes the content, version, author, and the caller's can_edit/can_propose/can_approve flags.",
    parameters: Type.Object({
      scope: Type.Union(
        [
          Type.Literal("agent"),
          Type.Literal("agent_project"),
          Type.Literal("project"),
        ],
        { description: "Which prompt to read." },
      ),
      projectId: Type.Optional(
        Type.String({
          description:
            "Project id or slug; required for agent_project and project.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = promptQuery(params.scope, params.projectId);
      return request("GET", `/prompts?${query.toString()}`);
    },
  });

  pi.registerTool({
    name: "listPromptVersions",
    label: "List Prompt Versions",
    description:
      "Read a CherryPi prompt's version history, newest first, with each version's author, approver, note, and content. Use it to understand how the prompt evolved before proposing a change. Pass the same scope/projectId as getPrompt.",
    parameters: Type.Object({
      scope: Type.Union(
        [
          Type.Literal("agent"),
          Type.Literal("agent_project"),
          Type.Literal("project"),
        ],
        { description: "Which prompt's history to read." },
      ),
      projectId: Type.Optional(
        Type.String({
          description:
            "Project id or slug; required for agent_project and project.",
        }),
      ),
      limit: Type.Optional(
        Type.Number({
          description: "Maximum versions; defaults to 50 and caps at 200.",
        }),
      ),
      beforeVersion: Type.Optional(
        Type.Number({
          description: "Page older than this version (0 = from latest).",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = promptQuery(params.scope, params.projectId);
      query.set("limit", String(params.limit ?? 50));
      if (params.beforeVersion !== undefined) {
        query.set("before_version", String(params.beforeVersion));
      }
      return request("GET", `/prompts/versions?${query.toString()}`);
    },
  });

  pi.registerTool({
    name: "proposePromptChange",
    label: "Propose Prompt Change",
    description:
      "Propose a complete replacement for a CherryPi prompt. Use `scope=agent` for your own global prompt, `agent_project` for your prompt in one project, or `project` for a project's shared prompt (project members only). The change becomes a pending proposal that an owner approves, unless your agent has self-prompt auto-approve enabled (agent scopes only). Include a short `note` explaining why.",
    parameters: Type.Object({
      scope: Type.Union(
        [
          Type.Literal("agent"),
          Type.Literal("agent_project"),
          Type.Literal("project"),
        ],
        { description: "Which prompt to propose a change to." },
      ),
      content: Type.String({
        description: "The complete replacement prompt content.",
      }),
      note: Type.Optional(
        Type.String({ description: "Why this change is useful." }),
      ),
      projectId: Type.Optional(
        Type.String({
          description:
            "Project id or slug; required for agent_project and project.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const body: Record<string, unknown> = {
        scope: String(params.scope),
        content: String(params.content ?? ""),
        note: params.note ? String(params.note) : "",
      };
      if (params.projectId) body.project_id = String(params.projectId);
      return request("POST", "/prompts/proposals", body);
    },
  });

  pi.registerTool({
    name: "listPromptProposals",
    label: "List Prompt Proposals",
    description:
      "List prompt change proposals. By default returns your own proposals across scopes, newest first. Pass `mine=false` with a scope/projectId to list a target's proposals when you are allowed to approve them.",
    parameters: Type.Object({
      scope: Type.Optional(
        Type.Union(
          [
            Type.Literal("agent"),
            Type.Literal("agent_project"),
            Type.Literal("project"),
          ],
          { description: "Optional target scope filter." },
        ),
      ),
      projectId: Type.Optional(
        Type.String({
          description:
            "Project id or slug; used with agent_project and project scopes.",
        }),
      ),
      status: Type.Optional(
        Type.Union(
          [
            Type.Literal("pending"),
            Type.Literal("approved"),
            Type.Literal("rejected"),
            Type.Literal("withdrawn"),
          ],
          { description: "Only proposals in this status." },
        ),
      ),
      mine: Type.Optional(
        Type.Boolean({
          description:
            "Only your own proposals. Defaults to true; set false with a scope to list a target's proposals for approval.",
        }),
      ),
      limit: Type.Optional(
        Type.Number({
          description: "Maximum proposals; defaults to 50 and caps at 200.",
        }),
      ),
      beforeId: Type.Optional(
        Type.Number({
          description: "Page older than this proposal id.",
        }),
      ),
    }),
    execute: async (_toolCallId, params) => {
      const query = new URLSearchParams();
      const mine = params.mine ?? true;
      query.set("mine", String(mine));
      if (params.scope) query.set("scope", String(params.scope));
      if (params.projectId) query.set("project_id", String(params.projectId));
      if (params.status) query.set("status", String(params.status));
      query.set("limit", String(params.limit ?? 50));
      if (params.beforeId !== undefined) {
        query.set("before_id", String(params.beforeId));
      }
      return request("GET", `/prompts/proposals?${query.toString()}`);
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

function promptQuery(scope: unknown, projectId: unknown): URLSearchParams {
  const query = new URLSearchParams();
  query.set("scope", String(scope ?? ""));
  if (projectId) query.set("project_id", String(projectId));
  return query;
}

function agencyUrl(): string {
  return (process.env.CHERRYPI_AGENCY_URL ?? "").replace(/\/+$/, "");
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

const DEFAULT_HISTORY_LIMIT = 20;

async function request(
  method: string,
  path: string,
  body?: Record<string, unknown>,
  structured = false,
) {
  const base = agencyUrl();
  const tokens = await agentTokens();
  if (!base || tokens.length === 0) {
    return text("CherryPi task tools are not configured for this task.");
  }

  const outcome = await fetchWithAgentToken(
    `${base}/api/rest/agent-tools${path}`,
    {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    },
    tokens,
  );
  const res = outcome.res;
  if (!res) {
    return text(
      `Request failed: ${outcome.error ?? "unknown error"}${credentialsLine(outcome)}`,
    );
  }

  const raw = await res.text();
  if (!res.ok) {
    let message = raw;
    try {
      message = (JSON.parse(raw) as { error?: string }).error ?? raw;
    } catch {}
    return text(
      `CherryPi request failed (${res.status}): ${message}${credentialsLine(outcome)}`,
    );
  }
  const note = credentialNote(outcome);
  if (structured) {
    try {
      const parsed = JSON.parse(raw) as Record<string, unknown>;
      return {
        content: [{ type: "text" as const, text: (raw || "OK") + note }],
        details: parsed,
        structuredContent: parsed,
      };
    } catch {}
  }
  return text((raw || "OK") + note);
}

function text(value: string) {
  return {
    content: [{ type: "text" as const, text: value }],
    details: undefined,
  };
}
