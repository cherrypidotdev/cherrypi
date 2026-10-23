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

/** Per-call options shared by the client's methods. */
export interface CallOptions {
  signal?: AbortSignal;
}

/** Options for {@link AgencyClient.watchTask}. */
export interface WatchTaskOptions extends CallOptions {
  /**
   * Replay the task's persisted conversation history (user prompts,
   * assistant text/thinking, tool calls) before live events. The dashboard
   * console sets this so opening a finished task still shows the transcript.
   */
  replayHistory?: boolean;
}

/** Options for {@link AgencyClient.addToTask}. */
export interface AddToTaskOptions extends CallOptions {
  /**
   * Uploaded attachments to send with this prompt. They are linked to the
   * resulting message and pushed to the agent with the prompt.
   */
  attachmentIds?: bigint[];
  /** Optional client-generated message id; the agency mints one when empty. */
  messageId?: string;
}

/** Input for {@link AgencyClient.uploadAttachment}. */
export interface AttachmentUploadInput {
  /** The file bytes. */
  file: Blob;
  /** Filename sent to the agency; defaults to a File's name. */
  filename?: string;
  /** Attach to an existing task (the tray). */
  taskId?: string;
  /** Set as the agent's picture. */
  agentId?: string;
  /** Set as the project's picture. */
  projectId?: string;
  /** Set as the user's picture (the caller's own, or any user for admins). */
  userId?: string;
  /** Set as the agency logo. */
  agencyLogo?: boolean;
  /** Set as the agency favicon. */
  agencyFavicon?: boolean;
}

/** Input for {@link AgencyClient.replaceAttachment}. */
export interface AttachmentReplaceInput {
  /** The new file bytes. */
  file: Blob;
  /** Filename sent to the agency; defaults to a File's name. */
  filename?: string;
}

/** Options for {@link AgencyClient.getTaskHistory}. */
export interface GetTaskHistoryOptions extends CallOptions {
  /**
   * Opaque cursor from a previous page; events older than it are returned.
   * Omit it to start from the newest events.
   */
  before?: string;
  /** Maximum events in the page (0 = the server default). */
  limit?: number;
}

/** One page of a task's persisted conversation, oldest-first. */
export interface GetTaskHistoryResult {
  events: TaskEvent[];
  /** Cursor for the next older page; empty when the history is exhausted. */
  nextBefore: string;
}

/** The outcome of migrating a task's agent-side Pi history to the agency. */
export interface MigrateTaskHistoryResult {
  /** The task, refreshed with `historyMigrated` set when entries were stored. */
  task: Task;
  /** How many entries the agent had (0 when there was nothing to migrate). */
  entries: number;
}

/** The agent's shared desktop and whether it can be served. */
export interface GetDesktopResult {
  /** Whether the agent can serve the desktop (x11vnc installed, X display reachable). */
  available: boolean;
  /** Why the desktop is unavailable, when `available` is false. */
  unavailableReason: string;
  /** The running X session, when reachable. */
  desktop?: Desktop;
  /** Agency relay path for the RFB WebSocket; empty when unavailable. */
  wsPath: string;
}

/** Filters for {@link AgencyClient.listTasks}. All are optional. */
export interface ListTasksFilter {
  /** Restrict to one project (id or slug); omit for every project. */
  projectId?: string;
  /** Restrict to tasks assigned to / running on one agent. */
  agentId?: string;
  /** Restrict to one lifecycle state. */
  state?: TaskState;
  /** Restrict to tasks carrying one tag. */
  tagId?: string;
  /** Free-text match against a task's name, prompt, or `<prefix>-<seq>` id. */
  search?: string;
  /** Restrict to one manually set size. */
  size?: TaskSize;
  /**
   * List archived tasks (true) instead of the active list (the default
   * false). Archive is a boolean field, not a state.
   */
  archived?: boolean;
}

/** Options for {@link AgencyClient.reopenTask}. */
export interface ReopenTaskOptions {
  /** Queue the task straight away instead of parking it in `BACKLOG`. */
  queue?: boolean;
}

/** Input for {@link AgencyClient.createTask}. */
export interface CreateTaskInput {
  /** The initial prompt; required and must be non-empty. */
  prompt: string;
  /** Optional display name. */
  name?: string;
  /**
   * Project the task belongs to (id or slug). The agency resolves the
   * project's prompts, composes them into the task's system prompt, and
   * records the association; a task cannot move between projects. Omit to
   * create an orphan task.
   */
  projectId?: string;
  /**
   * Pin the task to one agent. Empty leaves it for the scheduler when
   * queued. A pinned agent must be a member of {@link projectId}.
   */
  assigneeActorId?: string;
  /** Tags attached to the task; must all exist. */
  tagIds?: string[];
  /**
   * Tasks this task references. A `BLOCKING` reference gates queuing: a
   * queued task with an incomplete blocking reference is reported as blocked
   * and skipped by the scheduler. A `SUPERSEDES` reference cancels the
   * referenced task when it is in draft, backlog, queued, or failed.
   */
  references?: TaskReferenceInput[];
  /** Scheduler priority. Default `NORMAL`. */
  priority?: TaskPriority;
  /**
   * Manually estimated size. Default `MEDIUM`. It is informational only and
   * can be changed later in any state via
   * {@link AgencyClient.setTaskSize}.
   */
  size?: TaskSize;
  /**
   * The teams that own the task, as team ids. Each must be attached to the
   * task's project; a member of any owning team may prompt it. When omitted,
   * the project's teams are used.
   */
  teamIds?: number[];
  /**
   * The task's leads, as actor ids. Each must be a project member or a member
   * of one of {@link teamIds}.
   */
  leadActorIds?: string[];
  /**
   * Model provider for the task, e.g. `"anthropic"`. Set together with
   * {@link modelId}; omit both to use Pi's configured default.
   */
  modelProvider?: string;
  /** Model id for the task, e.g. `"claude-sonnet-4-5"`. */
  modelId?: string;
  /**
   * Reasoning level for the task, e.g. `"medium"`. Omit to use Pi's
   * configured default for the selected model.
   */
  thinkingLevel?: string;
  /**
   * Surface extension dialogs to the user. Default `false`: the task is
   * non-interactive and auto-declines dialogs (`cancelled: true`).
   */
  interactive?: boolean;
  /**
   * Directory to use as the task's workspace. Default: the agent assigns
   * `$CHERRYPI_DIR/sessions/<id>`. Created if missing.
   */
  workspace?: string;
  /**
   * Generate the display title from the initial prompt asynchronously.
   * Default `false`. Ignored when {@link name} is set.
   */
  autoTitle?: boolean;
  /**
   * The task's initial state: `DRAFT`, `BACKLOG`, or `QUEUED`. Default
   * `QUEUED`. A draft is never scheduled until promoted and queued; the agent
   * task tools only allow `DRAFT` or `BACKLOG`.
   */
  state?: TaskState;
}

/**
 * Input for {@link AgencyClient.updateTask}. There are two modes.
 *
 * A request that fills the draft fields (`prompt`, `projectId`, …) replaces a
 * draft task wholesale, so pass the full desired shape (omitted fields reset
 * to their empty/default value); only draft tasks are editable that way.
 *
 * A request that sets any of `setPriority`, `setSize`, or `setAssigneeActorId`
 * is an attribute-only update: the draft fields must be omitted, only the
 * present attributes change, and the task may be in any state except running
 * or waiting for input. An empty `setAssigneeActorId` clears the pin so the
 * scheduler picks an agent.
 */
/** One reference edge: the referenced task and the edge's kind. */
export interface TaskReferenceInput {
  /** The referenced task's id. */
  taskId: string;
  /**
   * The meaning of the edge. `BLOCKING` waits for the referenced task to
   * reach `COMPLETED`; `SUPERSEDES` cancels the referenced task in draft,
   * backlog, queued, or failed. Defaults to `INFORMATIONAL`.
   */
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
  interactive?: boolean;
  workspace?: string;
  autoTitle?: boolean;
  /** Attribute-only update: set the task's scheduler priority. */
  setPriority?: TaskPriority;
  /** Attribute-only update: set the task's estimated size. */
  setSize?: TaskSize;
  /**
   * Attribute-only update: pin the task to an agent. An empty string clears
   * the pin so the scheduler picks an agent.
   */
  setAssigneeActorId?: string;
  /**
   * Attribute-only update: replace a draft task's pending follow-up (the next
   * message a re-queue delivers). Only valid while the task is a draft.
   */
  setPendingPrompt?: string;
  /**
   * Attribute-only update: replace a task's prompt. Only valid before the
   * task is queued (draft or backlog); a queued or started task's prompt is
   * pinned into its Pi session.
   */
  setPrompt?: string;
}

/** One labeled URL attached to a task (e.g. a PR or an issue). */
export interface TaskLinkInput {
  /** Absolute http(s) URL. Others are dropped by the agency. */
  url: string;
  /** Short human label; clients fall back to the URL when empty. */
  label?: string;
}

/**
 * A partial attribute update shared by the single- and bulk-task paths. A
 * field left `undefined` is unchanged; an empty `assigneeActorId` clears the
 * pin. Priority and size are safe in any state; the assignee cannot change on
 * a running or input task.
 *
 * Tag changes are bulk-only: `addTagIds` attaches tags and `removeTagIds`
 * detaches them. Each added tag must belong to the task's project, so a task
 * in another project is reported with an error. Adding a tag a task already
 * carries, or removing one it does not, is a no-op.
 */
export interface TaskAttributeInput {
  priority?: TaskPriority;
  size?: TaskSize;
  /** Pin to an agent; an empty string clears the pin. */
  assigneeActorId?: string;
  /** Tags to attach to every task; must belong to each task's project. */
  addTagIds?: string[];
  /** Tags to detach from every task; a tag a task lacks is skipped. */
  removeTagIds?: string[];
}

/** Input for {@link AgencyClient.bulkUpdateTask}. */
export interface BulkUpdateTaskInput {
  /** The task ids to update, in order. */
  ids: string[];
  /** The attributes to apply to every task; at least one is required. */
  attributes: TaskAttributeInput;
}

/** Input for {@link AgencyClient.setTaskTeams}. */
export interface SetTaskTeamsInput {
  /**
   * The complete replacement list of owning team ids. At least one team is
   * required; a task never loses its last owning team.
   */
  teamIds: number[];
}

/** Input for {@link AgencyClient.setTaskLeads}. */
export interface SetTaskLeadsInput {
  /**
   * The complete replacement list of lead actor ids. Every actor must be a
   * project member or a member of one of the task's owning teams.
   */
  leadActorIds: string[];
}

/** Input for {@link AgencyClient.setTaskSubscription}. */
export interface SetTaskSubscriptionInput {
  /** True subscribes the caller, false unsubscribes them. */
  subscribed: boolean;
}

/**
 * One task's outcome in a {@link AgencyClient.bulkUpdateTask} call: the
 * refreshed task on success, or an error describing why it was skipped.
 */
export interface BulkUpdateTaskResult {
  id: string;
  task?: Task;
  error?: string;
}

/** Input for {@link AgencyClient.setTaskModel}. */
export interface SetTaskModelInput {
  /** Provider, e.g. `"anthropic"`. */
  provider: string;
  /** Model id, e.g. `"claude-sonnet-4-5"`. */
  modelId: string;
}

/** Filters for {@link AgencyClient.listTags}. */
export interface ListTagsFilter {
  /** Restrict to one project's tags. Omit to list every project's tags. */
  projectId?: string;
  /** Include archived tags (default `false`). */
  includeArchived?: boolean;
}

/** Input for {@link AgencyClient.createTag}. */
export interface CreateTagInput {
  /** The project that owns the tag. */
  projectId: string;
  /** Display name; unique within the project, case-insensitively. */
  name: string;
  description?: string;
  /** `#rrggbb`; empty picks a default. */
  color?: string;
  /** Pre-select the tag in the task composer for its project. */
  isDefault?: boolean;
}

/** Input for {@link AgencyClient.updateTag}. */
export interface UpdateTagInput {
  name: string;
  description?: string;
  color?: string;
  /** Pre-select the tag in the task composer for its project. */
  isDefault?: boolean;
}

/**
 * One per-tag concurrency cap in {@link AgentTaskLimitsInput}. The cap is
 * keyed by tag name and applies globally, across every project's tag of that
 * name (so one `unity-editor` cap gates every project's #unity-editor).
 */
export interface TagLimitInput {
  tagName: string;
  maxConcurrent: number;
}

/** One per-project concurrency cap in {@link AgentTaskLimitsInput}. */
export interface ProjectLimitInput {
  projectId: string;
  maxConcurrent: number;
}

/** One per-project-per-tag concurrency cap in {@link AgentTaskLimitsInput}. */
export interface ProjectTagLimitInput {
  projectId: string;
  tagId: string;
  maxConcurrent: number;
}

/**
 * Input for {@link AgencyClient.updateAgentTaskLimits}. Replaces the agent's
 * caps wholesale; omitted overall caps default to 1.
 */
export interface AgentTaskLimitsInput {
  /** Overall cap for the agent. Default 1. */
  maxConcurrent?: number;
  /** Cap on tasks per project. Default 1. */
  maxConcurrentPerProject?: number;
  /**
   * Per-tag caps, keyed by tag name and global across projects; an unlisted
   * tag imposes no extra cap beyond the above.
   */
  tagLimits?: TagLimitInput[];
  /** Per-project caps. */
  projectLimits?: ProjectLimitInput[];
  /** Per-project-per-tag caps. */
  projectTagLimits?: ProjectTagLimitInput[];
}

/**
 * Input for {@link AgencyClient.updateAgentDefaults}. Replaces the agent's
 * default model/thinking pair; empty model fields fall back to Pi's configured
 * default.
 */
export interface AgentDefaultsInput {
  modelProvider?: string;
  modelId?: string;
  modelName?: string;
  thinkingLevel?: string;
}

/** A reply to a pending extension dialog. */
export type ExtensionUiResponse =
  { value: string } | { confirmed: boolean } | { cancelled: true };

/**
 * One question/answer pair of a submitted `ask_user_question` wizard. The
 * console sends the whole set with the first answer so the agent can carry
 * every answer into the resume prompt when the dialog survived a restart.
 */
export interface QuestionnaireAnswerInput {
  question: string;
  answer: string;
}

/**
 * One notification as pushed over the realtime attention WebSocket, mirroring
 * the agency's `notifications` inbox entry.
 */
export interface AttentionNotification {
  /** Inbox row id. */
  id: number;
  /** Agent that raised the signal, empty when agency-level. */
  agentId: string;
  /** Agency task the signal belongs to, empty when not task-scoped. */
  taskId: string;
  /** Signal kind, e.g. `input_requested` or `failed`. */
  kind: string;
  /** `info`, `warning`, or `error`. */
  severity: string;
  title: string;
  body: string;
  /** RFC 3339 creation timestamp. */
  createdAt: string;
  /** RFC 3339 read timestamp, absent while unread. */
  readAt?: string;
}

/** A frame from the realtime attention WebSocket. */
export type AttentionEvent =
  | { type: "hello"; unread: number }
  | { type: "notification"; notification: AttentionNotification };

/** `Authorization: Basic …`, UTF-8 safe. */
export function basicAuthHeader(username: string, password: string): string {
  const bytes = new TextEncoder().encode(`${username}:${password}`);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return `Basic ${btoa(binary)}`;
}

/** A Connect interceptor that adds the basic-auth `Authorization` header. */
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

/**
 * A Connect interceptor that adds a personal-access-token `Authorization`
 * header (`Bearer pat_…`).
 */
export function bearerAuthInterceptor(token: string): Interceptor {
  const header = `Bearer ${token}`;
  return (next) => async (req) => {
    req.header.set("Authorization", header);
    return await next(req);
  };
}

/** The CSRF cookie/header pair the agency uses for cookie sessions. */
export const CSRF_COOKIE = "cherrypi_csrf";
export const CSRF_HEADER = "X-CSRF-Token";

/** Reads a cookie by name in the browser; "" elsewhere. */
export function readCookie(name: string): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/[.$?*|{}()[\]\\]/g, "\\$&")}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : "";
}

/**
 * A Connect interceptor that mirrors the CSRF cookie into the `X-CSRF-Token`
 * header, so cookie-authenticated mutations pass the double-submit check. It
 * is a no-op outside the browser (PAT/basic clients do not need it).
 */
export function csrfInterceptor(): Interceptor {
  return (next) => async (req) => {
    const token = readCookie(CSRF_COOKIE);
    if (token) req.header.set(CSRF_HEADER, token);
    return await next(req);
  };
}
