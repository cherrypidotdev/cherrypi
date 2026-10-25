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

import type { Interceptor } from "@connectrpc/connect";
import type { Desktop } from "./gen/buf/agent/v1/desktop_pb.js";
import type { TaskEvent } from "./gen/buf/agent/v1/task_pb.js";
import type {
  Task,
  TaskPriority,
  TaskReferenceKind,
  TaskSize,
  TaskState,
} from "./gen/buf/agency/v1/task_pb.js";

export interface CallOptions {
  signal?: AbortSignal;
}

export interface WatchTaskOptions extends CallOptions {
  replayHistory?: boolean;
}

export type RealtimeKind =
  "sync" | "tasks" | "usage" | "terminals" | "agent-logs" | "agents";

export interface RealtimeEvent {
  kind: RealtimeKind;
}

export interface WatchRealtimeOptions extends CallOptions {}

export interface AddToTaskOptions extends CallOptions {
  attachmentIds?: bigint[];

  messageId?: string;
}

export interface AttachmentUploadInput {
  file: Blob;

  filename?: string;

  taskId?: string;

  agentId?: string;

  projectId?: string;

  userId?: string;

  agencyLogo?: boolean;

  agencyFavicon?: boolean;
}

export interface AttachmentReplaceInput {
  file: Blob;

  filename?: string;
}

export interface GetTaskHistoryOptions extends CallOptions {
  before?: string;

  limit?: number;
}

export interface GetTaskHistoryResult {
  events: TaskEvent[];

  nextBefore: string;
}

export interface MigrateTaskHistoryResult {
  task: Task;

  entries: number;
}

export interface GetDesktopResult {
  available: boolean;

  unavailableReason: string;

  desktop?: Desktop;

  wsPath: string;
}

export interface ListTasksFilter {
  projectId?: string;

  agentId?: string;

  state?: TaskState;

  tagId?: string;

  search?: string;

  size?: TaskSize;

  archived?: boolean;
}

export interface ReopenTaskOptions {
  queue?: boolean;
}

export interface CreateTaskInput {
  prompt: string;

  name?: string;

  projectId?: string;

  assigneeActorId?: string;

  tagIds?: string[];

  references?: TaskReferenceInput[];

  priority?: TaskPriority;

  size?: TaskSize;

  teamIds?: number[];

  leadActorIds?: string[];

  modelProvider?: string;

  modelId?: string;

  thinkingLevel?: string;

  workspace?: string;

  autoTitle?: boolean;

  state?: TaskState;
}

export interface TaskReferenceInput {
  taskId: string;

  kind?: TaskReferenceKind;
}

export interface UpdateTaskInput {
  prompt?: string;
  name?: string;
  projectId?: string;
  assigneeActorId?: string;
  tagIds?: string[];
  priority?: TaskPriority;
  size?: TaskSize;
  modelProvider?: string;
  modelId?: string;
  thinkingLevel?: string;
  workspace?: string;
  autoTitle?: boolean;

  setPriority?: TaskPriority;

  setSize?: TaskSize;

  setAssigneeActorId?: string;

  setPendingPrompt?: string;

  setPrompt?: string;
}

export interface TaskLinkInput {
  url: string;

  label?: string;
}

export interface TaskAttributeInput {
  priority?: TaskPriority;
  size?: TaskSize;

  assigneeActorId?: string;

  addTagIds?: string[];

  removeTagIds?: string[];
}

export interface BulkUpdateTaskInput {
  ids: string[];

  attributes: TaskAttributeInput;
}

export interface SetTaskTeamsInput {
  teamIds: number[];
}

export interface SetTaskLeadsInput {
  leadActorIds: string[];
}

export interface SetTaskSubscriptionInput {
  subscribed: boolean;
}

export interface BulkUpdateTaskResult {
  id: string;
  task?: Task;
  error?: string;
}

export interface SetTaskModelInput {
  provider: string;

  modelId: string;
}

export interface ListTagsFilter {
  projectId?: string;

  includeArchived?: boolean;
}

export interface CreateTagInput {
  projectId: string;

  name: string;
  description?: string;

  color?: string;

  isDefault?: boolean;
}

export interface UpdateTagInput {
  name: string;
  description?: string;
  color?: string;

  isDefault?: boolean;
}

export interface TagLimitInput {
  tagName: string;
  maxConcurrent: number;
}

export interface ProjectLimitInput {
  projectId: string;
  maxConcurrent: number;
}

export interface ProjectTagLimitInput {
  projectId: string;
  tagId: string;
  maxConcurrent: number;
}

export interface AgentTaskLimitsInput {
  maxConcurrent?: number;

  maxConcurrentPerProject?: number;

  tagLimits?: TagLimitInput[];

  projectLimits?: ProjectLimitInput[];

  projectTagLimits?: ProjectTagLimitInput[];
}

export interface AgentDefaultsInput {
  modelProvider?: string;
  modelId?: string;
  modelName?: string;
  thinkingLevel?: string;
}

export type ExtensionUiResponse =
  { value: string } | { confirmed: boolean } | { cancelled: true };

export interface QuestionnaireAnswerInput {
  question: string;
  answer: string;
}

export interface AttentionNotification {
  id: number;

  agentId: string;

  taskId: string;

  kind: string;

  severity: string;
  title: string;
  body: string;

  createdAt: string;

  readAt?: string;
}

export type AttentionEvent =
  | { type: "hello"; unread: number }
  | { type: "notification"; notification: AttentionNotification };

export function basicAuthHeader(username: string, password: string): string {
  const bytes = new TextEncoder().encode(`${username}:${password}`);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return `Basic ${btoa(binary)}`;
}

export function basicAuthInterceptor(
  username: string,
  password: string,
): Interceptor {
  const header = basicAuthHeader(username, password);
  return (next) => async (req) => {
    req.header.set("Authorization", header);
    return await next(req);
  };
}

export function bearerAuthInterceptor(token: string): Interceptor {
  const header = `Bearer ${token}`;
  return (next) => async (req) => {
    req.header.set("Authorization", header);
    return await next(req);
  };
}

export const CSRF_COOKIE = "cherrypi_csrf";
export const CSRF_HEADER = "X-CSRF-Token";

export function readCookie(name: string): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/[.$?*|{}()[\]\\]/g, "\\$&")}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : "";
}

export function csrfInterceptor(): Interceptor {
  return (next) => async (req) => {
    const token = readCookie(CSRF_COOKIE);
    if (token) req.header.set(CSRF_HEADER, token);
    return await next(req);
  };
}
