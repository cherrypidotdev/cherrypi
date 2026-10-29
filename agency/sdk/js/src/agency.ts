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
  RealtimeKind as RealtimeKindPb,
  RealtimeService,
  type RealtimeEvent as RealtimeEventPb,
} from "./gen/buf/agency/v1/realtime_pb.js";
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
  type QuestionnaireAnswerInput,
  type RealtimeEvent,
  type RealtimeKind,
  type WatchRealtimeOptions,
  type ReopenTaskOptions,
  type SetTaskModelInput,
  type TaskLinkInput,
  type SetTaskLeadsInput,
  type SetTaskFavoriteInput,
  type SetTaskSubscriptionInput,
  type SetTaskTeamsInput,
  type UpdateTagInput,
  type UpdateTaskInput,
  type WatchTaskOptions,
} from "./common.js";

export type { Agent, AgentDefaults, AgentLog, AgentToken };
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

export interface GetUsageOptions extends CallOptions {
  sinceMs?: number;

  untilMs?: number;
}

export interface AgencyClientOptions {
  baseUrl: string;

  token?: string;

  transport?: Transport;
}

export interface AgencySettingsInput {
  agencyName: string;

  primaryColor: string;

  showUsageKpi?: boolean;

  showConcurrencyKpi?: boolean;
}

export interface CreateAgentInput {
  username: string;
  displayName?: string;
  description?: string;

  prompt?: string;
}

export interface CreateProjectInput {
  name: string;

  slug?: string;

  prefix?: string;
  description?: string;
}

export interface UpdateProjectInput {
  name: string;
  description: string;

  prefix?: string;
}

export interface PromptTarget {
  agentId?: string;
  teamId?: string | number;
  projectId?: string;
}

export interface ProjectAgentPermissionsInput {
  canListTasks: boolean;
  canCreateTasks: boolean;
  canUpdateTasks: boolean;
}

export class AgencyClient {
  readonly baseUrl: string;

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
  private readonly realtime: Client<typeof RealtimeService>;

  constructor(options: AgencyClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");

    const interceptors: Interceptor[] = [csrfInterceptor()];

    if (options.token) {
      this.authHeader = `Bearer ${options.token}`;
      interceptors.unshift(bearerAuthInterceptor(options.token));
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
    this.realtime = createClient(RealtimeService, transport);
  }

  private restHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};
    if (this.authHeader) headers.Authorization = this.authHeader;
    const csrf = readCookie(CSRF_COOKIE);
    if (csrf) headers[CSRF_HEADER] = csrf;
    return headers;
  }

  async listAgents(options?: CallOptions): Promise<Agent[]> {
    const res = await this.agents.listAgents({}, options);
    return res.agents;
  }

  async getAgent(id: string, options?: CallOptions): Promise<Agent> {
    const res = await this.agents.getAgent({ id }, options);
    return res.agent!;
  }

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

  async createAgent(
    input: CreateAgentInput,
    options?: CallOptions,
  ): Promise<{ agent: Agent; token: string }> {
    const res = await this.agents.createAgent(
      {
        username: input.username,
        displayName: input.displayName ?? "",
        description: input.description ?? "",
        prompt: input.prompt ?? "",
      },
      options,
    );
    return { agent: res.agent!, token: res.token };
  }

  async updateAgent(
    id: string,
    input: { displayName: string; description: string },
    options?: CallOptions,
  ): Promise<Agent> {
    const res = await this.agents.updateAgent({ id, ...input }, options);
    return res.agent!;
  }

  async updateAgentRole(
    id: string,
    role: "member" | "admin",
    options?: CallOptions,
  ): Promise<Agent> {
    const res = await this.agents.updateAgentRole({ id, role }, options);
    return res.agent!;
  }

  async deleteAgent(id: string, options?: CallOptions): Promise<void> {
    await this.agents.deleteAgent({ id }, options);
  }

  async listAgentTokens(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentToken[]> {
    const res = await this.agents.listAgentTokens({ agentId }, options);
    return res.tokens;
  }

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

  async revokeAgentToken(
    agentId: string,
    id: bigint | number,
    options?: CallOptions,
  ): Promise<void> {
    await this.agents.revokeAgentToken({ agentId, id: BigInt(id) }, options);
  }

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

  async listProjects(options?: CallOptions): Promise<Project[]> {
    const res = await this.projects.listProjects({}, options);
    return res.projects;
  }

  async getProject(idOrSlug: string, options?: CallOptions): Promise<Project> {
    const res = await this.projects.getProject({ id: idOrSlug }, options);
    return res.project!;
  }

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

  async archiveProject(
    id: string,
    options?: CallOptions,
  ): Promise<{ project: Project; offlineAgents: string[] }> {
    const res = await this.projects.archiveProject({ id }, options);
    return { project: res.project!, offlineAgents: res.offlineAgents };
  }

  async unarchiveProject(id: string, options?: CallOptions): Promise<Project> {
    const res = await this.projects.unarchiveProject({ id }, options);
    return res.project!;
  }

  async deleteProject(id: string, options?: CallOptions): Promise<void> {
    await this.projects.deleteProject({ id }, options);
  }

  async listProjectAgents(
    projectId: string,
    options?: CallOptions,
  ): Promise<ProjectAgent[]> {
    const res = await this.projects.listProjectAgents({ projectId }, options);
    return res.agents;
  }

  async listAgentProjects(
    agentId: string,
    options?: CallOptions,
  ): Promise<Project[]> {
    const res = await this.projects.listAgentProjects({ agentId }, options);
    return res.projects;
  }

  async listAgentMemberships(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentProjectMembership[]> {
    const res = await this.projects.listAgentMemberships({ agentId }, options);
    return res.memberships;
  }

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

  async removeProjectAgent(
    projectId: string,
    agentId: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.projects.removeProjectAgent({ projectId, agentId }, options);
  }

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

  async getTaskSystemPrompt(
    taskId: string,
    options?: CallOptions,
  ): Promise<SystemPromptSection[]> {
    const res = await this.prompts.getTaskSystemPrompt({ taskId }, options);
    return res.sections;
  }

  async getSettings(options?: CallOptions): Promise<Settings> {
    const res = await this.settings.getSettings({}, options);
    return res.settings!;
  }

  async updateSettings(
    input: AgencySettingsInput,
    options?: CallOptions,
  ): Promise<Settings> {
    const res = await this.settings.updateSettings(
      {
        settings: {
          agencyName: input.agencyName,
          primaryColor: input.primaryColor,
          showUsageKpi: input.showUsageKpi ?? true,
          showConcurrencyKpi: input.showConcurrencyKpi ?? true,
        },
      },
      options,
    );
    return res.settings!;
  }

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
        priority: filter?.priority ?? TaskPriority.UNSPECIFIED,
        archived: filter?.archived ?? false,
        referenceProjectId: filter?.referenceProjectId ?? "",
        limit: filter?.limit ?? 0,
        offset: filter?.offset ?? 0,
      },
      options,
    );
    return res.tasks;
  }

  async getTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.getTask({ id }, options);
    return res.task!;
  }

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

  async queueTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.queueTask({ id }, options);
    return res.task!;
  }

  async promoteTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.promoteTask({ id }, options);
    return res.task!;
  }

  async demoteTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.demoteTask({ id }, options);
    return res.task!;
  }

  async dequeueTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.dequeueTask({ id }, options);
    return res.task!;
  }

  async mergeTasks(ids: string[], options?: CallOptions): Promise<Task> {
    const res = await this.tasks.mergeTasks({ ids }, options);
    return res.task!;
  }

  async closeOutTask(
    id: string,
    outcome: TaskState,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.closeOutTask({ id, outcome }, options);
    return res.task!;
  }

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

  async archiveTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.archiveTask({ id }, options);
    return res.task!;
  }

  async unarchiveTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.unarchiveTask({ id }, options);
    return res.task!;
  }

  async cleanupTaskSession(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.cleanupTaskSession({ id }, options);
    return res.task!;
  }

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

  async deleteTask(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.deleteTask({ id }, options);
  }

  async renameTask(
    id: string,
    name: string,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.renameTask({ id, name }, options);
    return res.task!;
  }

  async setTaskPriority(
    id: string,
    priority: TaskPriority,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskPriority({ id, priority }, options);
    return res.task!;
  }

  async setTaskSize(
    id: string,
    size: TaskSize,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskSize({ id, size }, options);
    return res.task!;
  }

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

  async setTaskFavorite(
    id: string,
    input: SetTaskFavoriteInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.setTaskFavorite(
      { id, favorite: input.favorite },
      options,
    );
    return res.task!;
  }

  async listTaskReferences(id: string, options?: CallOptions): Promise<Task[]> {
    const res = await this.tasks.listTaskReferences({ id }, options);
    return res.tasks;
  }

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

  async stopTask(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.stopTask({ id }, options);
  }

  async retryTask(id: string, options?: CallOptions): Promise<Task> {
    const res = await this.tasks.retryTask({ id }, options);
    return res.task!;
  }

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

  async watchRealtime(
    onEvent: (event: RealtimeEvent) => void,
    options?: WatchRealtimeOptions,
  ): Promise<void> {
    const stream = await this.realtime.watch({}, { signal: options?.signal });
    for await (const message of stream) {
      if (!message.event) continue;
      const kind = realtimeKindName(message.event);
      if (kind) onEvent({ kind });
    }
  }

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

  async getTaskConfig(id: string, options?: CallOptions): Promise<TaskConfig> {
    const res = await this.tasks.getTaskConfig({ id }, options);
    return res.config!;
  }

  async getTaskDefaults(
    agentId: string,
    options?: CallOptions,
  ): Promise<TaskConfig> {
    const res = await this.tasks.getTaskDefaults({ agentId }, options);
    return res.defaults!;
  }

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

  async setTaskThinkingLevel(
    id: string,
    level: string,
    options?: CallOptions,
  ): Promise<TaskConfig> {
    const res = await this.tasks.setTaskThinkingLevel({ id, level }, options);
    return res.config!;
  }

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

  async clearTaskQueue(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.clearTaskQueue({ id }, options);
  }

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

  async archiveTag(id: string, options?: CallOptions): Promise<Tag> {
    const res = await this.tags.archiveTag({ id }, options);
    return res.tag!;
  }

  async restoreTag(id: string, options?: CallOptions): Promise<Tag> {
    const res = await this.tags.restoreTag({ id }, options);
    return res.tag!;
  }

  async getAgentDefaults(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentDefaults> {
    const res = await this.agents.getAgentDefaults({ agentId }, options);
    return res.defaults!;
  }

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

  async getAgentTaskLimits(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentTaskLimits> {
    const res = await this.agents.getAgentTaskLimits({ agentId }, options);
    return res.limits!;
  }

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

  async getTaskUsage(
    taskId: string,
    options?: CallOptions,
  ): Promise<TaskUsage> {
    const res = await this.dashboard.getTaskUsage({ taskId }, options);
    return res.usage!;
  }

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

  async getTaskDiskUsage(
    taskId: string,
    options?: CallOptions,
  ): Promise<bigint> {
    const res = await this.dashboard.getTaskDiskUsage({ taskId }, options);
    return res.usage?.bytes ?? 0n;
  }

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

  async listTerminals(
    agentId: string,
    options?: CallOptions,
  ): Promise<Terminal[]> {
    const res = await this.terminals.listTerminals({ agentId }, options);
    return res.terminals;
  }

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

  async closeTerminal(
    agentId: string,
    id: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.terminals.closeTerminal({ agentId, request: { id } }, options);
  }

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

  async getTerminalStats(
    agentId: string,
    options?: CallOptions,
  ): Promise<TerminalStats> {
    const res = await this.terminals.getTerminalStats({ agentId }, options);
    return res.stats!;
  }

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

  terminalWsPath(agentId: string, terminalId: string): string {
    return `/api/agents/${agentId}/terminal/${terminalId}/ws`;
  }

  private webSocketUrl(wsPath: string): string {
    const url = new URL(this.baseUrl);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = wsPath;
    url.search = "";
    return url.toString();
  }

  terminalWebSocketUrl(wsPath: string): string {
    return this.webSocketUrl(wsPath);
  }

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

  desktopWsPath(agentId: string): string {
    return `/api/agents/${agentId}/desktop/ws`;
  }

  desktopWebSocketUrl(wsPath: string): string {
    return this.webSocketUrl(wsPath);
  }

  notificationWsPath(): string {
    return "/api/rest/notifications/ws";
  }

  notificationWebSocketUrl(): string {
    return this.webSocketUrl(this.notificationWsPath());
  }

  async getUsage(
    agentId = "",
    projectId = "",
    options?: GetUsageOptions,
  ): Promise<GetUsageResponse> {
    const { sinceMs, untilMs, ...callOptions } = options ?? {};
    return await this.dashboard.getUsage(
      {
        agentId,
        projectId,
        sinceMs: BigInt(sinceMs ?? 0),
        untilMs: BigInt(untilMs ?? 0),
      },
      callOptions,
    );
  }

  async listNotifications(
    limit = 100,
    options?: CallOptions,
  ): Promise<Notification[]> {
    const res = await this.notifications.listNotifications({ limit }, options);
    return res.notifications;
  }

  async getUnreadCount(options?: CallOptions): Promise<number> {
    const res = await this.notifications.getUnreadCount({}, options);
    return Number(res.count);
  }

  async markNotificationRead(
    id: number | bigint,
    options?: CallOptions,
  ): Promise<void> {
    await this.notifications.markNotificationRead({ id: BigInt(id) }, options);
  }

  async markTaskNotificationsRead(
    taskId: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.notifications.markTaskNotificationsRead({ taskId }, options);
  }

  async markAllNotificationsRead(options?: CallOptions): Promise<void> {
    await this.notifications.markAllNotificationsRead({}, options);
  }

  async deleteNotification(
    id: number | bigint,
    options?: CallOptions,
  ): Promise<void> {
    await this.notifications.deleteNotification({ id: BigInt(id) }, options);
  }

  async deleteAllNotifications(options?: CallOptions): Promise<void> {
    await this.notifications.deleteAllNotifications({}, options);
  }

  async getInboxPreferences(options?: CallOptions): Promise<string[]> {
    const res = await this.notifications.getInboxPreferences({}, options);
    return res.disabledKinds;
  }

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

  async listWebhooks(options?: CallOptions): Promise<Webhook[]> {
    const res = await this.webhooks.listWebhooks({}, options);
    return res.webhooks;
  }

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

  async deleteWebhook(
    id: number | bigint,
    options?: CallOptions,
  ): Promise<void> {
    await this.webhooks.deleteWebhook({ id: BigInt(id) }, options);
  }

  async testWebhook(
    id: number | bigint,
    event: WebhookEvent,
    options?: CallOptions,
  ): Promise<TestWebhookResponse> {
    return await this.webhooks.testWebhook({ id: BigInt(id), event }, options);
  }
}

function realtimeKindName(event: RealtimeEventPb): RealtimeKind | undefined {
  switch (event.kind) {
    case RealtimeKindPb.SYNC:
      return "sync";
    case RealtimeKindPb.TASKS:
      return "tasks";
    case RealtimeKindPb.USAGE:
      return "usage";
    case RealtimeKindPb.TERMINALS:
      return "terminals";
    case RealtimeKindPb.AGENT_LOGS:
      return "agent-logs";
    case RealtimeKindPb.AGENTS:
      return "agents";
    default:
      return undefined;
  }
}

async function restError(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    if (body.error) {
      return body.error;
    }
  } catch {}
  return `${res.status} ${res.statusText}`;
}
