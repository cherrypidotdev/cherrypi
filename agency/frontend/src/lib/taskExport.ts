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
  TaskEventSchema,
  TaskPriority,
  TaskSize,
  TaskState,
  type GetTaskHistoryOptions,
  type GetTaskHistoryResult,
  type Task,
  type TaskEvent,
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
}

export interface TaskHistoryExport {
  format: string;
  version: number;
  exported_at: string;
  event_count: number;
  task: TaskHistoryHeader;

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
  };
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
  exportedAt = new Date(),
): TaskHistoryExport {
  return {
    format: TASK_HISTORY_FORMAT,
    version: TASK_HISTORY_VERSION,
    exported_at: exportedAt.toISOString(),
    event_count: events.length,
    task: taskHistoryHeader(task),
    events: events.map((event) => toJson(TaskEventSchema, event)),
  };
}

export function taskHistoryFilename(taskId: string): string {
  const safe = taskId.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "");
  return `${safe || "task"}-history.json`;
}

export function downloadTaskHistory(
  exported: TaskHistoryExport,
  taskId: string,
): void {
  const blob = new Blob([JSON.stringify(exported, null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = taskHistoryFilename(taskId);
  link.click();

  setTimeout(() => URL.revokeObjectURL(url), 0);
}
