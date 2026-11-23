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

import { timestampMs } from "@/lib/console";
import { formatBytes, isoInstant } from "@/lib/format";
import { toJson, type JsonValue } from "@bufbuild/protobuf";
import {
  PromptScope,
  TaskEventSchema,
  TaskPriority,
  TaskReferenceKind,
  TaskSize,
  TaskState,
  type Agent,
  type AgentGroup,
  type GetTaskHistoryOptions,
  type GetTaskHistoryResult,
  type Project,
  type SystemPromptSection,
  type Task,
  type TaskEvent,
  type TaskLink,
  type TaskLog,
  type TaskMessage,
  type TaskReference,
} from "@cherrypi/sdk";

export const TASK_HISTORY_FORMAT = "cherrypi.task-history";

export const TASK_HISTORY_VERSION = 1;

export const TASK_HISTORY_PAGE_LIMIT = 1000;

export const TASK_HISTORY_MAX_PAGES = 100;

export const TASK_HISTORY_MAX_BYTES = 64 * 1024 * 1024;

export class TaskHistoryTooLargeError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "TaskHistoryTooLargeError";
  }
}

export interface CollectTaskHistoryOptions {
  pageLimit?: number;

  maxPages?: number;

  maxBytes?: number;

  signal?: AbortSignal;
}

export type FetchTaskHistoryPage = (
  options: GetTaskHistoryOptions,
) => Promise<GetTaskHistoryResult>;

export interface TaskHistoryActor {
  actor_id: string;
  name: string;
  username: string;
  kind: string;

  picture_url?: string;
}

export interface TaskHistoryTag {
  id: string;
  name: string;
  color: string;
  required: boolean;
}

export interface TaskHistoryTeam {
  id: number;
  name: string;
}

export interface TaskHistoryHeader {
  id: string;
  seq: number;
  name: string;
  prompt: string;
  project_id: string;
  assignee_actor_id: string;
  creator_actor_id: string;
  state: string;
  priority: string;
  size: string;
  automated: boolean;
  archived: boolean;
  model_provider: string;
  model_id: string;
  thinking_level: string;
  session_id: string;
  created_at: string | null;
  queued_at: string | null;
  started_at: string | null;
  settled_at: string | null;
  updated_at: string | null;
  archived_at: string | null;
  error: string;

  pending_prompt: string;
  pending_prompt_actor_id: string;
  tags: TaskHistoryTag[];
  teams: TaskHistoryTeam[];
  creator: TaskHistoryActor | null;
  leader: TaskHistoryActor | null;
}

export interface TaskHistoryProject {
  id: string;
  name: string;
  slug: string;
  prefix: string;
  description: string;
  emoji: string;
  parent_project_id: string;
  archived_at: string | null;
}

export interface TaskHistoryGroup {
  id: number;
  name: string;
  slug: string;
  description: string;
}

export interface TaskHistoryAgent {
  id: string;
  actor_id: string;
  username: string;
  display_name: string;
  description: string;
  role: string;
  groups: TaskHistoryGroup[];
}

export interface TaskHistoryPrompt {
  scope: string;
  owner: string;
  version: number;
  pinned: boolean;
  preview: boolean;
  group_id: number;
  group_name: string;
  team_id: number;
  content: string;
}

export interface TaskHistoryMessage {
  id: string;
  text: string;
  queued_at: string | null;
  actor: TaskHistoryActor | null;
  attachments: { id: number; filename: string }[];
}

export interface TaskHistoryActivity {
  id: number;
  at: string | null;
  actor: string;
  actor_id: string;
  actor_name: string;
  actor_username: string;
  kind: string;
  field: string;
  old_value: string;
  new_value: string;
  message: string;

  data: JsonValue | null;
}

export interface TaskHistoryReference {
  id: string;
  name: string;
  state: string;
  kind: string;
  wake_on_resolve_when_many: boolean;
  priority: string;
  size: string;
  tags: TaskHistoryTag[];
}

export interface TaskHistoryLink {
  url: string;
  label: string;
}

export interface TaskHistoryDetails {
  project?: Project | null;
  assignee?: Agent | null;
  groups?: AgentGroup[];
  prompts?: SystemPromptSection[];
}

export interface TaskHistoryExport {
  format: string;
  version: number;
  exported_at: string;
  event_count: number;
  activity_count: number;
  task: TaskHistoryHeader;
  project: TaskHistoryProject | null;
  assignee: TaskHistoryAgent | null;
  system_prompts: TaskHistoryPrompt[];
  messages: TaskHistoryMessage[];

  activity: TaskHistoryActivity[];
  references: {
    outgoing: TaskHistoryReference[];
    incoming: TaskHistoryReference[];
  };
  links: TaskHistoryLink[];

  events: JsonValue[];
}

function pageBytes(events: TaskEvent[]): number {
  const json = JSON.stringify(
    events,

    (_key, value: unknown) =>
      typeof value === "bigint" ? value.toString() : value,
    2,
  );
  return new TextEncoder().encode(json).length;
}

function enumName(enumType: Record<number, string>, value: number): string {
  return enumType[value] ?? String(value);
}

function instant(
  at: { seconds: bigint; nanos?: number } | undefined,
): string | null {
  return isoInstant(timestampMs(at)) || null;
}

function tagExport(tag: {
  id: string;
  name: string;
  color: string;
  isRequired?: boolean;
}): TaskHistoryTag {
  return {
    id: tag.id,
    name: tag.name,
    color: tag.color,
    required: tag.isRequired ?? false,
  };
}

export function taskHistoryHeader(task: Task): TaskHistoryHeader {
  return {
    id: task.id,
    seq: Number(task.seq),
    name: task.name,
    prompt: task.prompt,
    project_id: task.projectId,
    assignee_actor_id: task.assigneeActorId,
    creator_actor_id: task.creatorActorId,
    state: enumName(TaskState, task.state),
    priority: enumName(TaskPriority, task.priority),
    size: enumName(TaskSize, task.size),
    automated: task.automated,
    archived: task.archived,
    model_provider: task.modelProvider,
    model_id: task.modelId,
    thinking_level: task.thinkingLevel,
    session_id: task.sessionId,
    created_at: instant(task.createdAt),
    queued_at: instant(task.queuedAt),
    started_at: instant(task.startedAt),
    settled_at: instant(task.settledAt),
    updated_at: instant(task.updatedAt),
    archived_at: instant(task.archivedAt),
    error: task.error,
    pending_prompt: task.pendingPrompt,
    pending_prompt_actor_id: task.pendingPromptActorId,
    tags: task.tags.map(tagExport),
    teams: task.teams.map((team) => ({ id: team.id, name: team.name })),
    creator: task.creatorActorId
      ? {
          actor_id: task.creatorActorId,
          name: task.creatorName,
          username: task.creatorUsername,
          kind: task.creatorKind,
          picture_url: task.creatorPictureUrl,
        }
      : null,
    leader: task.leader
      ? {
          actor_id: task.leader.actorId,
          name: task.leader.name,
          username: task.leader.username,
          kind: task.leader.kind,
          picture_url: task.leader.pictureUrl,
        }
      : null,
  };
}

function projectExport(project: Project): TaskHistoryProject {
  return {
    id: project.id,
    name: project.name,
    slug: project.slug,
    prefix: project.prefix,
    description: project.description,
    emoji: project.emoji,
    parent_project_id: project.parentProjectId,
    archived_at: instant(project.archivedAt),
  };
}

function agentExport(agent: Agent, groups: AgentGroup[]): TaskHistoryAgent {
  return {
    id: agent.id,
    actor_id: agent.actorId,
    username: agent.username,
    display_name: agent.displayName,
    description: agent.description,
    role: agent.role,
    groups: groups.map((group) => ({
      id: Number(group.id),
      name: group.name,
      slug: group.slug,
      description: group.description,
    })),
  };
}

function promptExport(section: SystemPromptSection): TaskHistoryPrompt {
  return {
    scope: enumName(PromptScope, section.scope),
    owner: section.owner,
    version: Number(section.version),
    pinned: section.pinned,
    preview: section.preview,
    group_id: Number(section.groupId),
    group_name: section.groupName,
    team_id: Number(section.teamId),
    content: section.content,
  };
}

function messageExport(message: TaskMessage): TaskHistoryMessage {
  const actorId = message.actorId;
  return {
    id: message.id,
    text: message.text,
    queued_at: instant(message.createdAt),
    actor: actorId
      ? {
          actor_id: actorId,
          name: message.actorName,
          username: message.actorUsername,
          kind: message.actorKind,
        }
      : null,
    attachments: message.attachments.map((attachment) => ({
      id: Number(attachment.id),
      filename: attachment.filename,
    })),
  };
}

function parseData(dataJSON: string): JsonValue | null {
  if (!dataJSON) return null;
  try {
    return JSON.parse(dataJSON) as JsonValue;
  } catch {
    return null;
  }
}

function activityExport(log: TaskLog): TaskHistoryActivity {
  return {
    id: Number(log.id),
    at: instant(log.at),
    actor: log.actor,
    actor_id: log.actorId,
    actor_name: log.actorName,
    actor_username: log.actorUsername,
    kind: log.kind,
    field: log.field,
    old_value: log.oldValue,
    new_value: log.newValue,
    message: log.message,
    data: parseData(log.dataJson),
  };
}

function referenceExport(ref: TaskReference): TaskHistoryReference {
  return {
    id: ref.id,
    name: ref.name,
    state: enumName(TaskState, ref.state),
    kind: enumName(TaskReferenceKind, ref.kind),
    wake_on_resolve_when_many: ref.wakeOnResolveWhenMany,
    priority: enumName(TaskPriority, ref.priority),
    size: enumName(TaskSize, ref.size),
    tags: ref.tags.map(tagExport),
  };
}

function linkExport(link: TaskLink): TaskHistoryLink {
  return { url: link.url, label: link.label };
}

function activityChronological(logs: TaskLog[]): TaskLog[] {
  return [...logs].sort((a, b) => {
    const at = (timestampMs(a.at) ?? 0) - (timestampMs(b.at) ?? 0);
    return at || Number(a.id - b.id);
  });
}

export async function collectTaskHistory(
  fetchPage: FetchTaskHistoryPage,
  options: CollectTaskHistoryOptions = {},
): Promise<TaskEvent[]> {
  const pageLimit = options.pageLimit ?? TASK_HISTORY_PAGE_LIMIT;
  const maxPages = options.maxPages ?? TASK_HISTORY_MAX_PAGES;
  const maxBytes = options.maxBytes ?? TASK_HISTORY_MAX_BYTES;
  const { signal } = options;

  const pages: TaskEvent[][] = [];
  const seen = new Set<string>();
  let before = "";
  let bytes = 0;
  for (;;) {
    signal?.throwIfAborted();
    const page = await fetchPage({ before, limit: pageLimit, signal });
    signal?.throwIfAborted();
    pages.push(page.events);
    bytes += pageBytes(page.events);
    if (bytes >= maxBytes) {
      throw new TaskHistoryTooLargeError(
        `This task's history is too large to export as one file: it passed ${formatBytes(maxBytes)}.`,
      );
    }
    const next = page.nextBefore;

    if (!next || next === before || seen.has(next)) break;

    if (pages.length >= maxPages) {
      throw new TaskHistoryTooLargeError(
        `This task's history is too large to export as one file: it passed ${maxPages} pages.`,
      );
    }
    seen.add(next);
    before = next;
  }
  return pages.reverse().flat();
}

export function buildTaskHistoryExport(
  task: Task,
  events: TaskEvent[],
  details: TaskHistoryDetails = {},
  exportedAt = new Date(),
): TaskHistoryExport {
  return {
    format: TASK_HISTORY_FORMAT,
    version: TASK_HISTORY_VERSION,
    exported_at: exportedAt.toISOString(),
    event_count: events.length,
    activity_count: task.logs.length,
    task: taskHistoryHeader(task),
    project: details.project ? projectExport(details.project) : null,
    assignee: details.assignee
      ? agentExport(details.assignee, details.groups ?? [])
      : null,
    system_prompts: (details.prompts ?? []).map(promptExport),
    messages: task.messages.map(messageExport),
    activity: activityChronological(task.logs).map(activityExport),
    references: {
      outgoing: task.references.map(referenceExport),
      incoming: task.referencedBy.map(referenceExport),
    },
    links: task.links.map(linkExport),
    events: exportEvents(events),
  };
}

function exportEvents(events: TaskEvent[]): JsonValue[] {
  const out: Record<string, any>[] = [];

  // block ends, ahead of the transcript line that stores its reasoning, so a

  const spans: PendingSpan[] = [];
  for (const event of events) {
    const json = toJson(TaskEventSchema, event) as Record<string, any>;
    const kind = event.event.case;
    if (kind !== "text" && kind !== "thinking") {
      out.push(json);
      continue;
    }
    const body = json[kind] as Record<string, any>;
    body.text = body.delta ?? "";
    delete body.delta;

    if (kind === "thinking" && body.text === "") {
      const chars = Number(body.chars ?? 0);
      if (Number(body.durationMs ?? 0) > 0 && chars > 0) {
        spans.push({ durationMs: body.durationMs, chars });
      }
      continue;
    }

    if (kind === "thinking") {
      const span = takeSpan(spans, body.text);
      if (span) applySpan(body, span);
    }

    const last = out[out.length - 1];
    if (last && last.at !== undefined && last.at === json.at && last[kind]) {
      const target = last[kind] as Record<string, any>;
      target.text += body.text;
      if (kind === "thinking") {
        mergeThinkingSpan(target, body);
      }
      continue;
    }
    out.push(json);
  }

  for (const span of spans) attachSpan(out, span);
  return out;
}

interface PendingSpan {
  durationMs: unknown;
  chars: number;
}

function codePoints(text: string): number {
  return Array.from(text).length;
}

function takeSpan(spans: PendingSpan[], text: string): PendingSpan | undefined {
  const chars = codePoints(text);
  const index = spans.findIndex((span) => span.chars === chars);
  if (index < 0) return undefined;
  return spans.splice(index, 1)[0];
}

function applySpan(target: Record<string, any>, span: PendingSpan): void {
  target.durationMs = span.durationMs;
  target.chars = span.chars;
  delete target.estimated;
}

function mergeThinkingSpan(
  target: Record<string, any>,
  body: Record<string, any>,
): void {
  const targetMeasured =
    target.durationMs !== undefined && target.estimated !== true;
  const bodyMeasured = body.durationMs !== undefined && body.estimated !== true;
  if (targetMeasured && bodyMeasured) {
    target.durationMs = String(
      Number(target.durationMs) + Number(body.durationMs),
    );
    delete target.estimated;
  } else if (body.durationMs !== undefined) {
    const current =
      target.durationMs === undefined ? -1 : Number(target.durationMs);
    if (current < 0 || Number(body.durationMs) > current) {
      target.durationMs = body.durationMs;
      if (body.estimated) target.estimated = true;
      else delete target.estimated;
    }
  }

  target.chars = codePoints(String(target.text ?? ""));
}

function attachSpan(out: Record<string, any>[], span: PendingSpan): void {
  for (let i = out.length - 1; i >= 0; i--) {
    const entry = out[i]?.thinking as Record<string, any> | undefined;
    if (!entry) continue;
    if (entry.durationMs !== undefined && entry.estimated !== true) continue;
    if (entry.text === undefined || codePoints(entry.text) !== span.chars) {
      continue;
    }
    applySpan(entry, span);
    return;
  }
}

export function taskHistoryFilename(taskId: string): string {
  const safe = taskId.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "");
  return `${safe || "task"}-history.json`;
}

export function downloadTaskHistory(
  exported: TaskHistoryExport,
  taskId: string,
  maxBytes = TASK_HISTORY_MAX_BYTES,
): void {
  const body = JSON.stringify(exported, null, 2);
  const bytes = new TextEncoder().encode(body).length;
  if (bytes >= maxBytes) {
    throw new TaskHistoryTooLargeError(
      `This task's history is too large to export as one file: it passed ${formatBytes(maxBytes)}.`,
    );
  }
  const blob = new Blob([body], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = taskHistoryFilename(taskId);
  link.click();

  setTimeout(() => URL.revokeObjectURL(url), 0);
}
