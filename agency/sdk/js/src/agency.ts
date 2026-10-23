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

import { fromJson } from "@bufbuild/protobuf";
import {
  createClient,
  type Client,
  type Interceptor,
  type Transport,
} from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  AgentService,
  type Agent,
  type AgentCredential,
  type AgentDefaults,
  type AgentLog,
  AgentLogLevel,
  type AgentTaskLimits,
  type AgentToken,
} from "./gen/buf/agency/v1/agents_pb.js";
import {
  TagService,
  TaskService as AgencyTaskService,
  TaskPriority,
  TaskReferenceKind,
  TaskSize,
  TaskState,
  type RespondToExtensionUiRequest,
  type Tag,
  type Task,
} from "./gen/buf/agency/v1/task_pb.js";
import {
  AttachmentSchema,
  type Attachment,
} from "./gen/buf/agency/v1/attachments_pb.js";
import {
  ProjectService,
  type AgentProjectMembership,
  type Project,
  type ProjectAgent,
} from "./gen/buf/agency/v1/projects_pb.js";
import {
  PromptScope,
  PromptService,
  type AgentProjectPrompt,
  type Prompt,
  type SystemPromptSection,
} from "./gen/buf/agency/v1/prompts_pb.js";
import { DashboardService as AgencyDashboardService } from "./gen/buf/agency/v1/dashboard_pb.js";
import {
  SettingsService,
  type Settings,
} from "./gen/buf/agency/v1/settings_pb.js";
import { TerminalService as AgencyTerminalService } from "./gen/buf/agency/v1/terminal_pb.js";
import { DesktopService as AgencyDesktopService } from "./gen/buf/agency/v1/desktop_pb.js";
import {
  WebhookEvent,
  WebhookService as AgencyWebhookService,
  type TestWebhookResponse,
  type Webhook,
} from "./gen/buf/agency/v1/webhook_pb.js";
import {
  NotificationService as AgencyNotificationService,
  type Notification,
} from "./gen/buf/agency/v1/notifications_pb.js";
import type { GetUsageResponse } from "./gen/buf/agency/v1/dashboard_pb.js";
import {
  TerminalKind,
  type Terminal,
  type TerminalLog,
  type TerminalStats,
} from "./gen/buf/agent/v1/terminal_pb.js";
import type { TaskConfig, TaskEvent } from "./gen/buf/agent/v1/task_pb.js";
import type {
  TaskDiskUsage,
  TaskUsage,
} from "./gen/buf/agent/v1/dashboard_pb.js";
import {
  basicAuthHeader,
  basicAuthInterceptor,
  bearerAuthInterceptor,
  csrfInterceptor,
  readCookie,
  CSRF_COOKIE,
  CSRF_HEADER,
  type AddToTaskOptions,
  type AgentDefaultsInput,
  type AgentTaskLimitsInput,
  type AttachmentReplaceInput,
  type AttachmentUploadInput,
  type BulkUpdateTaskInput,
  type BulkUpdateTaskResult,
  type CallOptions,
  type CreateTaskInput,
  type CreateTagInput,
  type ExtensionUiResponse,
  type GetTaskHistoryOptions,
  type GetTaskHistoryResult,
  type GetDesktopResult,
  type ListTagsFilter,
  type ListTasksFilter,
  type MigrateTaskHistoryResult,
  type QuestionnaireAnswerInput,
  type ReopenTaskOptions,
  type SetTaskModelInput,
  type TaskLinkInput,
  type SetTaskLeadsInput,
  type SetTaskSubscriptionInput,
  type SetTaskTeamsInput,
  type UpdateTagInput,
  type UpdateTaskInput,
  type WatchTaskOptions,
} from "./common.js";

export type { Agent, AgentCredential, AgentDefaults, AgentLog };
export {
  AgentLogLevel,
  AgentStatus,
  AgentVersionStatus,
} from "./gen/buf/agency/v1/agents_pb.js";
export type { Project, ProjectAgent, AgentProjectMembership };
export type { AgentProjectPrompt, Prompt, SystemPromptSection };
export { PromptScope };
export type { Settings as AgencySettings };
export type { Webhook as AgencyWebhook };
export type { Notification as AgencyNotification };
export { WebhookEvent as AgencyWebhookEvent };
export type {
  GetUsageResponse as AgencyUsageResponse,
  AgentTaskUsage,
  UsageDay as AgencyUsageDay,
} from "./gen/buf/agency/v1/dashboard_pb.js";

/** Options for {@link AgencyClient}. */
export interface AgencyClientOptions {
  /** Agency base URL, e.g. `https://cherrypi.vladfaust.com`. */
  baseUrl: string;
  /**
   * Send basic-auth credentials on every request. Default `true`. Set to
   * `false` for **ambient auth**: the SDK adds no `Authorization` header and
   * the browser supplies its cached credentials.
   */
  basicAuth?: boolean;
  /** Basic-auth username. Required unless `basicAuth` is `false`. */
  username?: string;
  /** Basic-auth password. Required unless `basicAuth` is `false`. */
  password?: string;
  /**
   * Personal access token (`pat_…`). When set, the SDK authenticates with
   * `Authorization: Bearer <token>` and ignores username/password. This is the
   * SDK/CLI credential.
   */
  token?: string;
  /** Override the RPC transport (you then own its authentication). */
  transport?: Transport;
}

/** Writable agency configuration (name and primary accent color). */ export interface AgencySettingsInput {
  agencyName: string;
  /** Primary color as `#rrggbb`; empty uses the built-in default. */
  primaryColor: string;
}

/** Input for {@link AgencyClient.createAgent}. */
export interface CreateAgentInput {
  username: string;
  displayName?: string;
  description?: string;
  /** Optional initial agent prompt (the agent scope's first version). */
  prompt?: string;
}

/** Input for {@link AgencyClient.createProject}. */
export interface CreateProjectInput {
  name: string;
  /** Lower-kebab slug; derived from the name when omitted. Immutable. */
  slug?: string;
  /** Task id prefix; derived from the slug (uppercased) when omitted. */
  prefix?: string;
  description?: string;
}

/** Input for {@link AgencyClient.updateProject}. The slug is immutable. */
export interface UpdateProjectInput {
  name: string;
  description: string;
  /**
   * New task id prefix. Omit to leave it unchanged; changing it renames the
   * project's existing tasks.
   */
  prefix?: string;
}

/** The agent, team, and/or project a prompt belongs to, per scope. */
export interface PromptTarget {
  agentId?: string;
  teamId?: string | number;
  projectId?: string;
}

/** The task permissions a member agent holds in one project. */
export interface ProjectAgentPermissionsInput {
  canListTasks: boolean;
  canCreateTasks: boolean;
  canUpdateTasks: boolean;
}

/**
 * AgencyClient is the typed client for the CherryPi **agency**: the central app
 * that manages agents and users, and owns the task lifecycle the scheduler
 * runs on them over their tunnels.
 */
export class AgencyClient {
  readonly baseUrl: string;

  /** Basic-auth header for the fetch-based REST routes; unset in ambient auth. */
  private readonly authHeader?: string;

  private readonly agents: Client<typeof AgentService>;
  private readonly settings: Client<typeof SettingsService>;
  private readonly tasks: Client<typeof AgencyTaskService>;
  private readonly tags: Client<typeof TagService>;
  private readonly projects: Client<typeof ProjectService>;
  private readonly prompts: Client<typeof PromptService>;
  private readonly dashboard: Client<typeof AgencyDashboardService>;
  private readonly terminals: Client<typeof AgencyTerminalService>;
  private readonly desktops: Client<typeof AgencyDesktopService>;
  private readonly notifications: Client<typeof AgencyNotificationService>;
  private readonly webhooks: Client<typeof AgencyWebhookService>;

  constructor(options: AgencyClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");

    const useBasicAuth = options.basicAuth !== false;
    const interceptors: Interceptor[] = [csrfInterceptor()];

    if (options.token) {
      this.authHeader = `Bearer ${options.token}`;
      interceptors.unshift(bearerAuthInterceptor(options.token));
    } else if (useBasicAuth) {
      if (options.username === undefined || options.password === undefined) {
        throw new Error(
          "AgencyClient: username and password are required unless basicAuth is false",
        );
      }
      this.authHeader = basicAuthHeader(options.username!, options.password!);
      interceptors.unshift(
        basicAuthInterceptor(options.username!, options.password!),
      );
    }

    const transport =
      options.transport ??
      createConnectTransport({
        baseUrl: `${this.baseUrl}/api/grpc`,
        interceptors,
      });

    this.agents = createClient(AgentService, transport);
    this.settings = createClient(SettingsService, transport);
    this.tasks = createClient(AgencyTaskService, transport);
    this.tags = createClient(TagService, transport);
    this.projects = createClient(ProjectService, transport);
    this.prompts = createClient(PromptService, transport);
    this.dashboard = createClient(AgencyDashboardService, transport);
    this.terminals = createClient(AgencyTerminalService, transport);
    this.desktops = createClient(AgencyDesktopService, transport);
    this.notifications = createClient(AgencyNotificationService, transport);
    this.webhooks = createClient(AgencyWebhookService, transport);
  }

  /**
   * Headers for the fetch-based REST routes: the configured credential plus the
   * CSRF header when a cookie session is in play (the browser's same-origin
   * cookie is sent automatically).
   */
  private restHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};
    if (this.authHeader) headers.Authorization = this.authHeader;
    const csrf = readCookie(CSRF_COOKIE);
    if (csrf) headers[CSRF_HEADER] = csrf;
    return headers;
  }

  // -- Agents --------------------------------------------------------------

  /** Every registered agent, sorted by display name. */
  async listAgents(options?: CallOptions): Promise<Agent[]> {
    const res = await this.agents.listAgents({}, options);
    return res.agents;
  }

  /** One agent by id. */
  async getAgent(id: string, options?: CallOptions): Promise<Agent> {
    const res = await this.agents.getAgent({ id }, options);
    return res.agent!;
  }

  /**
   * One page of an agent's persisted operational logs, newest first. Pass
   * `beforeId` (the last row's `id`) to page further back.
   */
  async listAgentLogs(
    agentId: string,
    options?: {
      minLevel?: AgentLogLevel;
      limit?: number;
      beforeId?: bigint;
    },
    callOptions?: CallOptions,
  ): Promise<AgentLog[]> {
    const res = await this.agents.listAgentLogs(
      {
        agentId,
        minLevel: options?.minLevel ?? AgentLogLevel.UNSPECIFIED,
        limit: options?.limit ?? 0,
        beforeId: options?.beforeId ?? 0n,
      },
      callOptions,
    );
    return res.logs;
  }

  /**
   * Register an agent and generate its first tunnel credential and personal
   * access token. Both secrets are returned **once**; show them to the operator
   * and keep only the hashes.
   */
  async createAgent(
    input: CreateAgentInput,
    options?: CallOptions,
  ): Promise<{
    agent: Agent;
    credentialUsername: string;
    credentialPassword: string;
    token: string;
  }> {
    const res = await this.agents.createAgent(
      {
        username: input.username,
        displayName: input.displayName ?? "",
        description: input.description ?? "",
        prompt: input.prompt ?? "",
      },
      options,
    );
    return {
      agent: res.agent!,
      credentialUsername: res.credentialUsername,
      credentialPassword: res.credentialPassword,
      token: res.token,
    };
  }

  /** Update an agent's display name and description. */
  async updateAgent(
    id: string,
    input: { displayName: string; description: string },
    options?: CallOptions,
  ): Promise<Agent> {
    const res = await this.agents.updateAgent({ id, ...input }, options);
    return res.agent!;
  }

  /** Remove an agent and its credentials. */
  async deleteAgent(id: string, options?: CallOptions): Promise<void> {
    await this.agents.deleteAgent({ id }, options);
  }

  /** An agent's tunnel credentials. */
  async listAgentCredentials(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentCredential[]> {
    const res = await this.agents.listAgentCredentials({ agentId }, options);
    return res.credentials;
  }

  /** Add a tunnel credential; the password is returned **once**. */
  async createAgentCredential(
    agentId: string,
    username: string,
    options?: CallOptions,
  ): Promise<{ credential: AgentCredential; password: string }> {
    const res = await this.agents.createAgentCredential(
      { agentId, username },
      options,
    );
    return { credential: res.credential!, password: res.password };
  }

  /** Revoke an agent tunnel credential. */
  async revokeAgentCredential(
    agentId: string,
    username: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.agents.revokeAgentCredential({ agentId, username }, options);
  }

  /** An agent's personal access tokens; secrets are never returned. */
  async listAgentTokens(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentToken[]> {
    const res = await this.agents.listAgentTokens({ agentId }, options);
    return res.tokens;
  }

  /**
   * Mint an agent personal access token; the plaintext is returned **once**.
   * The management surface is administrator-only until agent ownership lands.
   */
  async createAgentToken(
    agentId: string,
    name: string,
    expiresAt?: Date,
    options?: CallOptions,
  ): Promise<{ token: AgentToken; plaintext: string }> {
    const res = await this.agents.createAgentToken(
      {
        agentId,
        name,
        expiresAt: expiresAt
          ? { seconds: BigInt(Math.floor(expiresAt.getTime() / 1000)) }
          : undefined,
      },
      options,
    );
    return { token: res.token!, plaintext: res.plaintext };
  }

  /** Revoke an agent personal access token. */
  async revokeAgentToken(
    agentId: string,
    id: bigint | number,
    options?: CallOptions,
  ): Promise<void> {
    await this.agents.revokeAgentToken({ agentId, id: BigInt(id) }, options);
  }

  // -- Users ---------------------------------------------------------------

  /**
   * The authenticated user's username. The legacy gRPC credential service is
   * gone; identity comes from the session/PAT surface.
   */
  async whoAmI(options?: CallOptions): Promise<string> {
    const res = await fetch(`${this.baseUrl}/api/rest/auth/session`, {
      method: "GET",
      headers: this.restHeaders(),
      signal: options?.signal,
    });
    if (!res.ok) {
      throw new Error(`whoAmI failed: ${res.status} ${res.statusText}`);
    }
    const payload = (await res.json()) as { actor?: { username?: string } };
    return payload.actor?.username ?? "";
  }

  // -- Projects ------------------------------------------------------------

  /** Every project, active and archived. */
  async listProjects(options?: CallOptions): Promise<Project[]> {
    const res = await this.projects.listProjects({}, options);
    return res.projects;
  }

  /** One project by id or slug. */
  async getProject(idOrSlug: string, options?: CallOptions): Promise<Project> {
    const res = await this.projects.getProject({ id: idOrSlug }, options);
    return res.project!;
  }

  /** Create a project. The slug is derived from the name when omitted. */
  async createProject(
    input: CreateProjectInput,
    options?: CallOptions,
  ): Promise<Project> {
    const res = await this.projects.createProject(
      {
        name: input.name,
        slug: input.slug ?? "",
        prefix: input.prefix ?? "",
        description: input.description ?? "",
      },
      options,
    );
    return res.project!;
  }

  /** Update a project's name, description and task prefix (the slug is immutable). */
  async updateProject(
    id: string,
    input: UpdateProjectInput,
    options?: CallOptions,
  ): Promise<Project> {
    const res = await this.projects.updateProject(
      {
        id,
        name: input.name,
        description: input.description,
        prefix: input.prefix ?? "",
      },
      options,
    );
    return res.project!;
  }

  /**
   * Archive a project. An archived project and its tasks are hidden from the
   * all-projects task list.
   */
  async archiveProject(
    id: string,
    options?: CallOptions,
  ): Promise<{ project: Project; offlineAgents: string[] }> {
    const res = await this.projects.archiveProject({ id }, options);
    return { project: res.project!, offlineAgents: res.offlineAgents };
  }

  /** Unarchive a project so it and its tasks show again. */
  async unarchiveProject(id: string, options?: CallOptions): Promise<Project> {
    const res = await this.projects.unarchiveProject({ id }, options);
    return res.project!;
  }

  /** Delete a project; fails while any of its tasks still exists. */
  async deleteProject(id: string, options?: CallOptions): Promise<void> {
    await this.projects.deleteProject({ id }, options);
  }

  /** A project's member agents, with each agent's task count. */
  async listProjectAgents(
    projectId: string,
    options?: CallOptions,
  ): Promise<ProjectAgent[]> {
    const res = await this.projects.listProjectAgents({ projectId }, options);
    return res.agents;
  }

  /** The projects an agent is a member of. */
  async listAgentProjects(
    agentId: string,
    options?: CallOptions,
  ): Promise<Project[]> {
    const res = await this.projects.listAgentProjects({ agentId }, options);
    return res.projects;
  }

  /**
   * The projects an agent has joined, each with the agent's task permissions
   * in it (the agent settings Projects tab).
   */
  async listAgentMemberships(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentProjectMembership[]> {
    const res = await this.projects.listAgentMemberships({ agentId }, options);
    return res.memberships;
  }

  /** Replace a member's task permissions in one project. */
  async updateProjectAgent(
    projectId: string,
    agentId: string,
    input: ProjectAgentPermissionsInput,
    options?: CallOptions,
  ): Promise<ProjectAgent> {
    const res = await this.projects.updateProjectAgent(
      {
        projectId,
        agentId,
        canListTasks: input.canListTasks,
        canCreateTasks: input.canCreateTasks,
        canUpdateTasks: input.canUpdateTasks,
      },
      options,
    );
    return res.agent!;
  }

  /** Add an agent to a project (idempotent). */
  async addProjectAgent(
    projectId: string,
    agentId: string,
    options?: CallOptions,
  ): Promise<ProjectAgent> {
    const res = await this.projects.addProjectAgent(
      { projectId, agentId },
      options,
    );
    return res.agent!;
  }

  /**
   * Remove an agent from a project. The agency deletes the agent's tasks in
   * the project first (deletion, never archive) and fails while the agent is
   * offline.
   */
  async removeProjectAgent(
    projectId: string,
    agentId: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.projects.removeProjectAgent({ projectId, agentId }, options);
  }

  // -- Prompts -------------------------------------------------------------

  /** The latest version of a scoped prompt (empty content when never set). */
  async getPrompt(
    scope: PromptScope,
    target?: PromptTarget,
    options?: CallOptions,
  ): Promise<Prompt> {
    const res = await this.prompts.getPrompt(
      {
        scope,
        agentId: target?.agentId ?? "",
        teamId: BigInt(target?.teamId ?? 0),
        projectId: target?.projectId ?? "",
      },
      options,
    );
    return res.prompt!;
  }

  /** Append a new prompt version when the content changed. */
  async setPrompt(
    scope: PromptScope,
    content: string,
    target?: PromptTarget,
    options?: CallOptions,
  ): Promise<Prompt> {
    const res = await this.prompts.setPrompt(
      {
        scope,
        agentId: target?.agentId ?? "",
        teamId: BigInt(target?.teamId ?? 0),
        projectId: target?.projectId ?? "",
        content,
      },
      options,
    );
    return res.prompt!;
  }

  /** Every project's agent-project prompt for one agent, newest prompt first. */
  async listAgentProjectPrompts(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentProjectPrompt[]> {
    const res = await this.prompts.listAgentProjectPrompts(
      { agentId },
      options,
    );
    return res.prompts;
  }

  /**
   * The scoped system prompts a task applies, in injection order. A started
   * task reports the exact versions it pinned; an unstarted one previews the
   * latest versions it would start with.
   */
  async getTaskSystemPrompt(
    taskId: string,
    options?: CallOptions,
  ): Promise<SystemPromptSection[]> {
    const res = await this.prompts.getTaskSystemPrompt({ taskId }, options);
    return res.sections;
  }

  // -- Settings ------------------------------------------------------------

  /** Agency configuration (name, primary color). */
  async getSettings(options?: CallOptions): Promise<Settings> {
    const res = await this.settings.getSettings({}, options);
    return res.settings!;
  }

  /** Persist and return the agency configuration. */
  async updateSettings(
    input: AgencySettingsInput,
    options?: CallOptions,
  ): Promise<Settings> {
    const res = await this.settings.updateSettings(
      { settings: input },
      options,
    );
    return res.settings!;
  }

  // -- Tasks ---------------------------------------------------------------

  /** Tasks matching the optional filters, newest first. */
  async listTasks(
    filter?: ListTasksFilter,
    options?: CallOptions,
  ): Promise<Task[]> {
    const res = await this.tasks.listTasks(
      {
        projectId: filter?.projectId ?? "",
        agentId: filter?.agentId ?? "",
        state: filter?.state ?? TaskState.UNSPECIFIED,
        tagId: filter?.tagId ?? "",
        search: filter?.search ?? "",
        size: filter?.size ?? TaskSize.UNSPECIFIED,
        archived: filter?.archived ?? false,
      },
      options,
    );
    return res.tasks;
  }

  /** One task, enriched with live runtime state when it is running. */
  async getTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.getTask({ id }, options);
    return res.task!;
  }

  /**
   * Create a draft, backlog, or queued task. A queued task is picked up by the
   * scheduler; an unassigned one is auto-assigned to a member agent.
   */
  async createTask(
    input: CreateTaskInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.createTask(
      {
        prompt: input.prompt,
        name: input.name ?? "",
        projectId: input.projectId ?? "",
        assigneeActorId: input.assigneeActorId ?? "",
        tagIds: input.tagIds ?? [],
        references: (input.references ?? []).map((ref) => ({
          referencedTaskId: ref.taskId,
          kind: ref.kind ?? TaskReferenceKind.INFORMATIONAL,
        })),
        priority: input.priority ?? TaskPriority.UNSPECIFIED,
        size: input.size ?? TaskSize.UNSPECIFIED,
        modelProvider: input.modelProvider ?? "",
        modelId: input.modelId ?? "",
        thinkingLevel: input.thinkingLevel ?? "",
        interactive: input.interactive ?? false,
        workspace: input.workspace ?? "",
        autoTitle: input.autoTitle ?? false,
        state: input.state ?? TaskState.UNSPECIFIED,
        teamIds: input.teamIds ?? [],
        leadActorIds: input.leadActorIds ?? [],
      },
      options,
    );
    return res.task!;
  }

  /**
   * Edit a draft task, or update a task's attributes (priority, size,
   * assignee, pending follow-up) in any state when the input's `set*` fields
   * are used.
   */
  async updateTask(
    id: string,
    input: UpdateTaskInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.updateTask(
      {
        id,
        prompt: input.prompt ?? "",
        name: input.name ?? "",
        projectId: input.projectId ?? "",
        assigneeActorId: input.assigneeActorId ?? "",
        tagIds: input.tagIds ?? [],
        priority: input.priority ?? TaskPriority.UNSPECIFIED,
        size: input.size ?? TaskSize.UNSPECIFIED,
        modelProvider: input.modelProvider ?? "",
        modelId: input.modelId ?? "",
        thinkingLevel: input.thinkingLevel ?? "",
        interactive: input.interactive ?? false,
        workspace: input.workspace ?? "",
        autoTitle: input.autoTitle ?? false,
        setPriority: input.setPriority,
        setSize: input.setSize,
        setAssigneeActorId: input.setAssigneeActorId,
        setPendingPrompt: input.setPendingPrompt,
        setPrompt: input.setPrompt,
      },
      options,
    );
    return res.task!;
  }

  /**
   * Apply the same attributes (priority, size, assignee, and/or tag changes)
   * to many tasks in one call. Every task is checked individually, so one that
   * cannot change (a live run, an unknown id, an assignee or tag outside its
   * project) is reported with an error while the rest still update.
   */
  async bulkUpdateTask(
    input: BulkUpdateTaskInput,
    options?: CallOptions,
  ): Promise<BulkUpdateTaskResult[]> {
    const res = await this.tasks.bulkUpdateTask(
      {
        ids: input.ids,
        setPriority: input.attributes.priority,
        setSize: input.attributes.size,
        setAssigneeActorId: input.attributes.assigneeActorId,
        addTagIds: input.attributes.addTagIds ?? [],
        removeTagIds: input.attributes.removeTagIds ?? [],
      },
      options,
    );
    return res.results.map((result) => ({
      id: result.id,
      task: result.task,
      error: result.error || undefined,
    }));
  }

  /** Queue a backlog task so the scheduler can start it. */
  async queueTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.queueTask({ id }, options);
    return res.task!;
  }

  /** Promote a draft task to backlog: very likely to run, not queued yet. */
  async promoteTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.promoteTask({ id }, options);
    return res.task!;
  }

  /** Demote a backlog task back to draft. */
  async demoteTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.demoteTask({ id }, options);
    return res.task!;
  }

  /** Move a queued task back to backlog without starting it. */
  async dequeueTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.dequeueTask({ id }, options);
    return res.task!;
  }

  /**
   * Merge several tasks into a new draft task. All mergees must share one
   * project and the same assignee, and be in draft, backlog, queued, or
   * failed. The merged task supersedes and cancels each mergee.
   */
  async mergeTasks(ids: string[], options?: CallOptions): Promise<Task> {
    const res = await this.tasks.mergeTasks({ ids }, options);
    return res.task!;
  }

  /**
   * Close a non-running task out as `TaskState.COMPLETED` or
   * `TaskState.CANCELLED`. A running or input task is rejected.
   */
  async closeOutTask(
    id: string,
    outcome: TaskState,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.closeOutTask({ id, outcome }, options);
    return res.task!;
  }

  /**
   * Reopen a settled, failed, completed, or cancelled task; it lands in backlog
   * or straight in the queue when `queue` is set.
   */
  async reopenTask(
    id: string,
    reopen?: ReopenTaskOptions,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.reopenTask(
      { id, queue: reopen?.queue ?? false },
      options,
    );
    return res.task!;
  }

  /**
   * Archive a task: hide it from the active lists and remove its agent-side Pi
   * session and workspace. A live run is stopped first; when the agent is
   * offline the archive still succeeds and the cleanup can be retried with
   * {@link AgencyClient.cleanupTaskSession}.
   */
  async archiveTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.archiveTask({ id }, options);
    return res.task!;
  }

  /** Bring an archived task back into the active lists. */
  async unarchiveTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.unarchiveTask({ id }, options);
    return res.task!;
  }

  /**
   * Delete an archived task's agent-side Pi session and workspace when the
   * archive had to skip it (the agent was offline).
   */
  async cleanupTaskSession(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.cleanupTaskSession({ id }, options);
    return res.task!;
  }

  /**
   * Replace a task's labeled links (e.g. a PR or an issue). Only absolute
   * http(s) URLs are kept and duplicates collapse; an empty list clears them.
   */
  async setTaskLinks(
    id: string,
    links: TaskLinkInput[],
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskLinks(
      {
        id,
        links: links.map((link) => ({
          url: link.url,
          label: link.label ?? "",
        })),
      },
      options,
    );
    return res.task!;
  }

  /**
   * Delete a task from any state. A running task is stopped and its
   * agent-side Pi session and workspace are removed; usage history stays.
   */
  async deleteTask(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.deleteTask({ id }, options);
  }

  /** Set a task's display name (forwarded to a running or settled session). */
  async renameTask(
    id: string,
    name: string,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.renameTask({ id, name }, options);
    return res.task!;
  }

  /**
   * Set a task's scheduler priority in any state. A queued task is drained in
   * priority order, so this reorders it; a running task picks the value up on
   * a re-queue.
   */
  async setTaskPriority(
    id: string,
    priority: TaskPriority,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskPriority({ id, priority }, options);
    return res.task!;
  }

  /**
   * Set a task's manually estimated size in any state. Size is informational;
   * it does not affect scheduling.
   */
  async setTaskSize(
    id: string,
    size: TaskSize,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskSize({ id, size }, options);
    return res.task!;
  }

  /**
   * Replace a task's owning teams: one or more teams attached to the task's
   * project. A member of any owning team may prompt the task.
   */
  async setTaskTeams(
    id: string,
    input: SetTaskTeamsInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskTeams(
      { id, teamIds: input.teamIds },
      options,
    );
    return res.task!;
  }

  /**
   * Replace a task's leads: the actors accountable for it, shown on the task
   * and notified of its updates. Each must be a member of an owning team.
   */
  async setTaskLeads(
    id: string,
    input: SetTaskLeadsInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskLeads(
      { id, leadActorIds: input.leadActorIds },
      options,
    );
    return res.task!;
  }

  /** Subscribe or unsubscribe the caller to a task's updates. */
  async setTaskSubscription(
    id: string,
    input: SetTaskSubscriptionInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskSubscription(
      { id, subscribed: input.subscribed },
      options,
    );
    return res.task!;
  }

  /** The tasks a task references (outgoing edges). */
  async listTaskReferences(id: string, options?: CallOptions): Promise<Task[]> {
    const res = await this.tasks.listTaskReferences({ id }, options);
    return res.tasks;
  }

  /**
   * Add or update a reference from one task to another. The agency rejects a
   * self-reference and a blocking cycle. A `BLOCKING` reference gates queuing;
   * a `SUPERSEDES` reference cancels the referenced task when it is in draft,
   * backlog, queued, or failed. References are editable in any state.
   */
  async addTaskReference(
    id: string,
    referencedTaskId: string,
    kind: TaskReferenceKind = TaskReferenceKind.INFORMATIONAL,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.addTaskReference(
      { id, referencedTaskId, kind },
      options,
    );
    return res.task!;
  }

  /** Drop one reference. */
  async removeTaskReference(
    id: string,
    referencedTaskId: string,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.removeTaskReference(
      { id, referencedTaskId },
      options,
    );
    return res.task!;
  }

  // -- Task runtime --------------------------------------------------------

  /** Send a follow-up prompt to a running or settled task. */
  async addToTask(
    id: string,
    prompt: string,
    options?: AddToTaskOptions,
  ): Promise<string> {
    const res = await this.tasks.addToTask(
      {
        id,
        prompt,
        attachmentIds: options?.attachmentIds ?? [],
        messageId: options?.messageId ?? "",
      },
      options,
    );
    return res.messageId;
  }

  // -- Attachments ---------------------------------------------------------

  /**
   * Upload one file to the agency and return its metadata. Exactly one of
   * `taskId`, `agentId`, or `projectId` selects where it lands. The byte
   * upload is the one place the SDK speaks plain REST instead of Connect
   * (Connect-Web cannot client-stream a request body).
   */
  async uploadAttachment(
    input: AttachmentUploadInput,
    options?: CallOptions,
  ): Promise<Attachment> {
    const selector: { kind: string; id?: string } | undefined =
      input.taskId !== undefined
        ? { kind: "task", id: input.taskId }
        : input.agentId !== undefined
          ? { kind: "agent_picture", id: input.agentId }
          : input.projectId !== undefined
            ? { kind: "project_picture", id: input.projectId }
            : input.userId !== undefined
              ? { kind: "user_picture", id: input.userId }
              : input.agencyLogo
                ? { kind: "agency_logo" }
                : input.agencyFavicon
                  ? { kind: "agency_favicon" }
                  : undefined;
    if (selector === undefined) {
      throw new Error(
        "uploadAttachment: one of taskId, agentId, projectId, userId, agencyLogo, or agencyFavicon is required",
      );
    }

    const params = new URLSearchParams({ kind: selector.kind });
    if (selector.id !== undefined) params.set("id", selector.id);
    const form = new FormData();
    const filename =
      input.filename ?? (input.file instanceof File ? input.file.name : "file");
    form.append("file", input.file, filename);

    const res = await fetch(`${this.baseUrl}/api/rest/attachments?${params}`, {
      method: "POST",
      body: form,
      headers: this.restHeaders(),
      signal: options?.signal,
    });
    if (!res.ok) {
      throw new Error(await restError(res));
    }
    return fromJson(AttachmentSchema, await res.json());
  }

  /**
   * Replace an attachment's bytes in place, keeping its id, filename and every
   * link to it. Editable attachments (sketches) use it to re-save without
   * orphaning the original.
   */
  async replaceAttachment(
    id: bigint,
    input: AttachmentReplaceInput,
    options?: CallOptions,
  ): Promise<Attachment> {
    const form = new FormData();
    const filename =
      input.filename ?? (input.file instanceof File ? input.file.name : "file");
    form.append("file", input.file, filename);

    const res = await fetch(
      `${this.baseUrl}/api/rest/attachments/${id.toString()}`,
      {
        method: "PUT",
        body: form,
        headers: this.restHeaders(),
        signal: options?.signal,
      },
    );
    if (!res.ok) {
      throw new Error(await restError(res));
    }
    return fromJson(AttachmentSchema, await res.json());
  }

  /** Delete an attachment (its blob and every link to it). */
  async deleteAttachment(id: bigint, options?: CallOptions): Promise<void> {
    const res = await fetch(
      `${this.baseUrl}/api/rest/attachments/${id.toString()}`,
      {
        method: "DELETE",
        headers: this.restHeaders(),
        signal: options?.signal,
      },
    );
    if (!res.ok) {
      throw new Error(await restError(res));
    }
  }

  /** Abort a running task's in-flight run. */
  async stopTask(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.stopTask({ id }, options);
  }

  /** Re-run a failed task's last prompt. */
  async retryTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.retryTask({ id }, options);
    return res.task!;
  }

  /**
   * Stream a task's events. Pass `replayHistory` to emit the persisted
   * conversation first.
   */
  async watchTask(
    id: string,
    onEvent: (event: TaskEvent) => void,
    options?: WatchTaskOptions,
  ): Promise<void> {
    const stream = await this.tasks.watchTask(
      { id, replayHistory: options?.replayHistory ?? false },
      { signal: options?.signal },
    );
    for await (const message of stream) {
      if (message.event) {
        onEvent(message.event);
      }
    }
  }

  /** One page of a task's persisted conversation, oldest-first. */
  async getTaskHistory(
    id: string,
    page?: GetTaskHistoryOptions,
  ): Promise<GetTaskHistoryResult> {
    const res = await this.tasks.getTaskHistory(
      { id, before: page?.before ?? "", limit: page?.limit ?? 0 },
      { signal: page?.signal },
    );
    return { events: res.events, nextBefore: res.nextBefore };
  }

  /**
   * Push a task's agent-side Pi history into the agency now, for a task whose
   * session predates automatic sync. Returns the refreshed task and the number
   * of entries the agent had (0 when there was nothing to migrate).
   */
  async migrateTaskHistory(
    id: string,
    options?: CallOptions,
  ): Promise<MigrateTaskHistoryResult> {
    const res = await this.tasks.migrateTaskHistory({ id }, options);
    return { task: res.task!, entries: res.entries };
  }

  /** The task's live model/reasoning config. */
  async getTaskConfig(id: string, options?: CallOptions): Promise<TaskConfig> {
    const res = await this.tasks.getTaskConfig({ id }, options);
    return res.config!;
  }

  /**
   * The model/thinking options a new task can start with on an agent,
   * including each model's supported thinking levels.
   */
  async getTaskDefaults(
    agentId: string,
    options?: CallOptions,
  ): Promise<TaskConfig> {
    const res = await this.tasks.getTaskDefaults({ agentId }, options);
    return res.defaults!;
  }

  /** Switch the task's model. */
  async setTaskModel(
    id: string,
    input: SetTaskModelInput,
    options?: CallOptions,
  ): Promise<TaskConfig> {
    const res = await this.tasks.setTaskModel(
      { id, provider: input.provider, modelId: input.modelId },
      options,
    );
    return res.config!;
  }

  /** Set the task's reasoning level. */
  async setTaskThinkingLevel(
    id: string,
    level: string,
    options?: CallOptions,
  ): Promise<TaskConfig> {
    const res = await this.tasks.setTaskThinkingLevel({ id, level }, options);
    return res.config!;
  }

  /** Answer a pending extension dialog. */
  async respondToExtensionUi(
    id: string,
    requestId: string,
    response: ExtensionUiResponse,
    questionnaire?: QuestionnaireAnswerInput[],
    options?: CallOptions,
  ): Promise<void> {
    let inner: RespondToExtensionUiRequest["response"];
    if ("value" in response) {
      inner = { case: "value", value: response.value };
    } else if ("confirmed" in response) {
      inner = { case: "confirmed", value: response.confirmed };
    } else {
      inner = { case: "cancelled", value: true };
    }

    await this.tasks.respondToExtensionUi(
      { id, requestId, response: inner, questionnaire },
      options,
    );
  }

  /**
   * Drop every queued (not yet injected) message on a task. A prompt sent
   * while a run is in flight is queued by Pi and injected between blocks.
   */
  async clearTaskQueue(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.clearTaskQueue({ id }, options);
  }

  // -- Tags ----------------------------------------------------------------

  /**
   * Task tags. Pass a project to scope to it; omit to list every project's
   * tags. Archived tags are hidden by default.
   */
  async listTags(
    filter?: ListTagsFilter,
    options?: CallOptions,
  ): Promise<Tag[]> {
    const res = await this.tags.listTags(
      {
        projectId: filter?.projectId ?? "",
        includeArchived: filter?.includeArchived ?? false,
      },
      options,
    );
    return res.tags;
  }

  /** Create a tag in a project. Names are unique per project, case-insensitively. */
  async createTag(input: CreateTagInput, options?: CallOptions): Promise<Tag> {
    const res = await this.tags.createTag(
      {
        projectId: input.projectId,
        name: input.name,
        description: input.description ?? "",
        color: input.color ?? "",
        isDefault: input.isDefault ?? false,
      },
      options,
    );
    return res.tag!;
  }

  /** Update a tag's name, description, and color. */
  async updateTag(
    id: string,
    input: UpdateTagInput,
    options?: CallOptions,
  ): Promise<Tag> {
    const res = await this.tags.updateTag(
      {
        id,
        name: input.name,
        description: input.description ?? "",
        color: input.color ?? "",
        isDefault: input.isDefault ?? false,
      },
      options,
    );
    return res.tag!;
  }

  /** Archive a tag (soft delete); it still renders on completed tasks. */
  async archiveTag(id: string, options?: CallOptions): Promise<Tag> {
    const res = await this.tags.archiveTag({ id }, options);
    return res.tag!;
  }

  /** Restore an archived tag. */
  async restoreTag(id: string, options?: CallOptions): Promise<Tag> {
    const res = await this.tags.restoreTag({ id }, options);
    return res.tag!;
  }

  // -- Agent defaults ------------------------------------------------------

  /** The agent's default model and thinking level for new tasks. */
  async getAgentDefaults(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentDefaults> {
    const res = await this.agents.getAgentDefaults({ agentId }, options);
    return res.defaults!;
  }

  /** Replace the agent's default model and thinking level wholesale. */
  async updateAgentDefaults(
    agentId: string,
    input: AgentDefaultsInput,
    options?: CallOptions,
  ): Promise<AgentDefaults> {
    const res = await this.agents.updateAgentDefaults(
      {
        agentId,
        defaults: {
          modelProvider: input.modelProvider ?? "",
          modelId: input.modelId ?? "",
          modelName: input.modelName ?? "",
          thinkingLevel: input.thinkingLevel ?? "",
        },
      },
      options,
    );
    return res.defaults!;
  }

  // -- Agent task limits ---------------------------------------------------

  /** An agent's task concurrency caps, with agency defaults applied. */
  async getAgentTaskLimits(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentTaskLimits> {
    const res = await this.agents.getAgentTaskLimits({ agentId }, options);
    return res.limits!;
  }

  /** Replace an agent's task concurrency caps wholesale. */
  async updateAgentTaskLimits(
    agentId: string,
    input: AgentTaskLimitsInput,
    options?: CallOptions,
  ): Promise<AgentTaskLimits> {
    const res = await this.agents.updateAgentTaskLimits(
      {
        agentId,
        limits: {
          maxConcurrent: input.maxConcurrent ?? 1,
          maxConcurrentPerProject: input.maxConcurrentPerProject ?? 1,
          tagLimits: (input.tagLimits ?? []).map((limit) => ({
            tagName: limit.tagName,
            maxConcurrent: limit.maxConcurrent,
          })),
          projectLimits: (input.projectLimits ?? []).map((limit) => ({
            projectId: limit.projectId,
            maxConcurrent: limit.maxConcurrent,
          })),
          projectTagLimits: (input.projectTagLimits ?? []).map((limit) => ({
            projectId: limit.projectId,
            tagId: limit.tagId,
            maxConcurrent: limit.maxConcurrent,
          })),
        },
      },
      options,
    );
    return res.limits!;
  }

  // -- Usage (routed to the agent by agentId) ------------------------------

  /** One task's usage, with the agent's live overlay. */
  async getTaskUsage(
    agentId: string,
    taskId: string,
    options?: CallOptions,
  ): Promise<TaskUsage> {
    const res = await this.dashboard.getTaskUsage({ agentId, taskId }, options);
    return res.usage!;
  }

  /**
   * One agent's per-session workspace sizes on disk, keyed by the agent-side
   * session id (the same id as `AgentSession.sessionId`). Walking every
   * workspace can take a while on an agent with many large sessions, so the
   * Cleanup tab loads this separately from {@link listAgentSessions}.
   */
  async getDiskUsage(
    agentId: string,
    options?: CallOptions,
  ): Promise<{ tasks: TaskDiskUsage[]; totalBytes: bigint }> {
    const res = await this.dashboard.getDiskUsage({ agentId }, options);
    return {
      tasks: res.usage?.tasks ?? [],
      totalBytes: res.usage?.totalBytes ?? 0n,
    };
  }

  /**
   * One task's workspace size on disk, in bytes. Unlike
   * {@link getDiskUsage}, this measures only the named task's workspace, so it
   * is cheap enough for the task view to refresh while a run is in flight.
   */
  async getTaskDiskUsage(
    agentId: string,
    taskId: string,
    options?: CallOptions,
  ): Promise<bigint> {
    const res = await this.dashboard.getTaskDiskUsage(
      { agentId, taskId },
      options,
    );
    return res.usage?.bytes ?? 0n;
  }

  // -- Terminals (routed to the agent by agentId) --------------------------

  /** Create a terminal; the returned `wsPath` is the agency relay path. */
  async createTerminal(
    agentId: string,
    kind: TerminalKind,
    init?: { cols?: number; rows?: number },
    options?: CallOptions,
  ): Promise<{ terminal: Terminal; wsPath: string }> {
    const res = await this.terminals.createTerminal(
      {
        agentId,
        request: {
          kind,
          cols: init?.cols ?? 0,
          rows: init?.rows ?? 0,
        },
      },
      options,
    );
    return { terminal: res.terminal!, wsPath: res.wsPath };
  }

  /** The caller's terminals on one agent. */
  async listTerminals(
    agentId: string,
    options?: CallOptions,
  ): Promise<Terminal[]> {
    const res = await this.terminals.listTerminals({ agentId }, options);
    return res.terminals;
  }

  /** Resize a terminal. */
  async resizeTerminal(
    agentId: string,
    id: string,
    cols: number,
    rows: number,
    options?: CallOptions,
  ): Promise<void> {
    await this.terminals.resizeTerminal(
      { agentId, request: { id, cols, rows } },
      options,
    );
  }

  /** Close a terminal. */
  async closeTerminal(
    agentId: string,
    id: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.terminals.closeTerminal({ agentId, request: { id } }, options);
  }

  /** Bulk-close the caller's terminals, optionally narrowed. */
  async closeAllTerminals(
    agentId: string,
    filter?: { kind?: TerminalKind },
    options?: CallOptions,
  ): Promise<number> {
    const res = await this.terminals.closeAllTerminals(
      {
        agentId,
        request: {
          kind: filter?.kind ?? TerminalKind.UNSPECIFIED,
        },
      },
      options,
    );
    return res.closed;
  }

  /** The caller's live terminal counts on one agent. */
  async getTerminalStats(
    agentId: string,
    options?: CallOptions,
  ): Promise<TerminalStats> {
    const res = await this.terminals.getTerminalStats({ agentId }, options);
    return res.stats!;
  }

  /**
   * The agent's persisted terminal lifecycle log (created/exited/closed/
   * interrupted), newest first, including terminals lost to a restart.
   */
  async listTerminalLogs(
    agentId: string,
    limit?: number,
    options?: CallOptions,
  ): Promise<TerminalLog[]> {
    const res = await this.terminals.listTerminalLogs(
      { agentId, request: { limit: limit ?? 0 } },
      options,
    );
    return res.logs;
  }

  /**
   * Agency-side WebSocket path for one of an agent's terminals, e.g.
   * `/api/agents/<agentId>/terminal/<terminalId>/ws`. `createTerminal`
   * returns this already; use it to reconnect to a persisted terminal id.
   */
  terminalWsPath(agentId: string, terminalId: string): string {
    return `/api/agents/${agentId}/terminal/${terminalId}/ws`;
  }

  /** Absolute `ws(s)://` URL for an agency WebSocket path. */
  private webSocketUrl(wsPath: string): string {
    const url = new URL(this.baseUrl);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = wsPath;
    url.search = "";
    return url.toString();
  }

  /** Absolute `ws(s)://` URL for an agency terminal path. */
  terminalWebSocketUrl(wsPath: string): string {
    return this.webSocketUrl(wsPath);
  }

  // -- Desktop (routed to the agent by agentId) ---------------------------

  /**
   * The agent's shared desktop: whether it can be served and, when reachable,
   * the X display's geometry. This does not start anything — CherryPi reuses
   * the session already running on the agent.
   */
  async getDesktop(
    agentId: string,
    options?: CallOptions,
  ): Promise<GetDesktopResult> {
    const res = await this.desktops.getDesktop({ agentId }, options);
    return {
      available: res.available,
      unavailableReason: res.unavailableReason,
      desktop: res.desktop,
      wsPath: res.wsPath,
    };
  }

  /**
   * Agency-side WebSocket path of an agent's desktop, e.g.
   * `/api/agents/<agentId>/desktop/ws`.
   */
  desktopWsPath(agentId: string): string {
    return `/api/agents/${agentId}/desktop/ws`;
  }

  /** Absolute `ws(s)://` URL for an agency desktop path. */
  desktopWebSocketUrl(wsPath: string): string {
    return this.webSocketUrl(wsPath);
  }

  /**
   * Agency-side WebSocket path of the realtime attention stream: new inbox
   * entries are pushed here instead of being polled.
   */
  notificationWsPath(): string {
    return "/api/rest/notifications/ws";
  }

  /** Absolute `ws(s)://` URL for the realtime attention stream. */
  notificationWebSocketUrl(): string {
    return this.webSocketUrl(this.notificationWsPath());
  }

  // -- Usage aggregate -----------------------------------------------------

  /** Cross-agent usage totals (optionally filtered to one agent/project). */
  async getUsage(
    agentId = "",
    projectId = "",
    options?: CallOptions,
  ): Promise<GetUsageResponse> {
    return await this.dashboard.getUsage({ agentId, projectId }, options);
  }

  // -- Notifications -------------------------------------------------------

  /** The inbox, newest first. */
  async listNotifications(
    limit = 100,
    options?: CallOptions,
  ): Promise<Notification[]> {
    const res = await this.notifications.listNotifications({ limit }, options);
    return res.notifications;
  }

  /** Number of unread notifications. */
  async getUnreadCount(options?: CallOptions): Promise<number> {
    const res = await this.notifications.getUnreadCount({}, options);
    return Number(res.count);
  }

  /** Mark one notification read. */
  async markNotificationRead(
    id: number | bigint,
    options?: CallOptions,
  ): Promise<void> {
    await this.notifications.markNotificationRead({ id: BigInt(id) }, options);
  }

  /** Mark every notification scoped to one task read. */
  async markTaskNotificationsRead(
    taskId: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.notifications.markTaskNotificationsRead({ taskId }, options);
  }

  /** Mark every notification read. */
  async markAllNotificationsRead(options?: CallOptions): Promise<void> {
    await this.notifications.markAllNotificationsRead({}, options);
  }

  /** Delete a notification. */
  async deleteNotification(
    id: number | bigint,
    options?: CallOptions,
  ): Promise<void> {
    await this.notifications.deleteNotification({ id: BigInt(id) }, options);
  }

  /** Empty the inbox. */
  async deleteAllNotifications(options?: CallOptions): Promise<void> {
    await this.notifications.deleteAllNotifications({}, options);
  }

  /**
   * Notification kinds the inbox does not store. A disabled kind is still
   * pushed for live sound/desktop alerts.
   */
  async getInboxPreferences(options?: CallOptions): Promise<string[]> {
    const res = await this.notifications.getInboxPreferences({}, options);
    return res.disabledKinds;
  }

  /** Persist the kinds the inbox does not store; returns the stored list. */
  async setInboxPreferences(
    disabledKinds: string[],
    options?: CallOptions,
  ): Promise<string[]> {
    const res = await this.notifications.setInboxPreferences(
      { disabledKinds },
      options,
    );
    return res.disabledKinds;
  }

  // -- Webhooks ------------------------------------------------------------

  /** Agency webhooks. */
  async listWebhooks(options?: CallOptions): Promise<Webhook[]> {
    const res = await this.webhooks.listWebhooks({}, options);
    return res.webhooks;
  }

  /** Create a webhook; `agentId` empty fires for all agents. */
  async createWebhook(
    agentId: string,
    url: string,
    events: WebhookEvent[],
    secret = "",
    options?: CallOptions,
  ): Promise<Webhook> {
    const res = await this.webhooks.createWebhook(
      { agentId, url, events, secret },
      options,
    );
    return res.webhook!;
  }

  /** Update a webhook (omit `secret` to keep it, pass "" to clear). */
  async updateWebhook(
    id: number | bigint,
    input: {
      agentId: string;
      url: string;
      events: WebhookEvent[];
      secret?: string;
    },
    options?: CallOptions,
  ): Promise<Webhook> {
    const res = await this.webhooks.updateWebhook(
      { id: BigInt(id), ...input },
      options,
    );
    return res.webhook!;
  }

  /** Delete a webhook. */
  async deleteWebhook(
    id: number | bigint,
    options?: CallOptions,
  ): Promise<void> {
    await this.webhooks.deleteWebhook({ id: BigInt(id) }, options);
  }

  /** Post a sample payload of `event` to a webhook. */
  async testWebhook(
    id: number | bigint,
    event: WebhookEvent,
    options?: CallOptions,
  ): Promise<TestWebhookResponse> {
    return await this.webhooks.testWebhook({ id: BigInt(id), event }, options);
  }
}

/** Turn a failed REST response into an Error, preferring the JSON `error`. */
async function restError(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    if (body.error) {
      return body.error;
    }
  } catch {
    // fall through to the status text
  }
  return `${res.status} ${res.statusText}`;
}
