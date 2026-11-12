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
  type AgentHealthSettings,
  type AgentLog,
  AgentLogLevel,
  type AgentTaskLimits,
  type AgentToken,
} from "./gen/buf/agency/v1/agents_pb.js";
import {
  TagService,
  TaskService as AgencyTaskService,
  TaskHoldReleaseMode,
  TaskPriority,
  TaskReferenceKind,
  TaskSize,
  TaskState,
  type RespondToExtensionUiRequest,
  type Tag,
  type Task,
  type TaskComment,
  type TaskTool,
  type TaskToolCallDetail,
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
  type ProjectTagReservation,
} from "./gen/buf/agency/v1/projects_pb.js";
import {
  PromptProposalStatus,
  PromptScope,
  PromptService,
  type AgentProjectPrompt,
  type Prompt,
  type PromptActor,
  type PromptProposal,
  type SystemPromptSection,
} from "./gen/buf/agency/v1/prompts_pb.js";
import {
  TaskTemplateArgType,
  TaskTemplateService,
  type TaskTemplate,
  type TaskTemplateContent,
  type TaskTemplateVersion,
} from "./gen/buf/agency/v1/task_templates_pb.js";
import { DashboardService as AgencyDashboardService } from "./gen/buf/agency/v1/dashboard_pb.js";
import {
  KpiLayoutService,
  KpiPageKind as KpiPageKindPb,
  type KpiLayout,
  type KpiWidget,
} from "./gen/buf/agency/v1/kpi_pb.js";
import {
  SettingsService,
  type Settings,
  type TaskAutoArchive,
  type VoiceInput,
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
import { SiteService, type Site } from "./gen/buf/agency/v1/sites_pb.js";
import type {
  AgentHealthSample,
  AgentUsageRun,
  GetUsageResponse,
} from "./gen/buf/agency/v1/dashboard_pb.js";
import {
  PluginService,
  type Plugin,
  type PluginLog,
  type PluginPreview,
  type PluginSettings,
  type PreviewPluginTemplateResponse,
  type ProjectPlugin,
} from "./gen/buf/agency/v1/plugins_pb.js";
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
  type AddTaskCommentInput,
  type AddToTaskOptions,
  type AgentDefaultsInput,
  type AgentHealthSettingsInput,
  type AgentTaskLimitsInput,
  type AttachmentReplaceInput,
  type AttachmentUploadInput,
  type BulkUpdateTaskInput,
  type BulkUpdateTaskResult,
  type CallOptions,
  type CreateTaskInput,
  type CreateTaskTemplateInput,
  type CreateTagInput,
  type ExtensionUiResponse,
  type GetTaskHistoryOptions,
  type GetTaskHistoryResult,
  type GetTaskToolCallOptions,
  type GetDesktopResult,
  type ListSitesFilter,
  type ListTagsFilter,
  type ListTaskToolCallsOptions,
  type ListTaskToolCallsResult,
  type ListTaskCommentsResult,
  type ListTasksFilter,
  type QuestionnaireAnswerInput,
  type RealtimeEvent,
  type RealtimeKind,
  type WatchRealtimeOptions,
  type ReopenTaskOptions,
  type ReleaseTaskHoldInput,
  type SetTaskModelInput,
  type TaskLinkInput,
  type SetTaskLeadsInput,
  type SetTaskFavoriteInput,
  type SetTaskSubscriptionInput,
  type SetTaskTeamsInput,
  type UpdateTagInput,
  type UpdateTaskInput,
  type UpdateTaskTemplateInput,
  type RenderTaskTemplateInput,
  type TaskTemplateArgInput,
  type TaskTemplateContentInput,
  type TaskTemplateValidationResult,
  type WatchTaskOptions,
} from "./common.js";

export type { Agent, AgentDefaults, AgentLog, AgentToken };
export {
  AgentLogLevel,
  AgentStatus,
  AgentVersionStatus,
} from "./gen/buf/agency/v1/agents_pb.js";
export type {
  Project,
  ProjectAgent,
  ProjectTagReservation,
  AgentProjectMembership,
};
export type { TaskTemplate, TaskTemplateVersion };
export { TaskTemplateArgType };
export type { Site };
export type { AgentProjectPrompt, Prompt, SystemPromptSection };
export { PromptScope };
export type { Settings as AgencySettings, TaskAutoArchive, VoiceInput };
export type { Webhook as AgencyWebhook };
export type { Notification as AgencyNotification };
export { WebhookEvent as AgencyWebhookEvent };
export type {
  GetUsageResponse as AgencyUsageResponse,
  AgentTaskUsage,
  AgentUsageRun,
  UsageDay as AgencyUsageDay,
  UsageModel as AgencyUsageModel,
} from "./gen/buf/agency/v1/dashboard_pb.js";

export type TaskUsageWithRuns = TaskUsage & { runs: AgentUsageRun[] };

export const UNKNOWN_USAGE_MODEL = "__unknown__";

export interface GetUsageOptions extends CallOptions {
  sinceMs?: number;

  untilMs?: number;

  model?: string;
}

export interface RemoveTaskReferenceOptions extends CallOptions {
  reverse?: boolean;
}

export interface AgencyClientOptions {
  baseUrl: string;

  token?: string;

  transport?: Transport;
}

export interface VoiceInputSettingsInput {
  enabled: boolean;

  baseUrl: string;

  model: string;
}

export interface TaskAutoArchiveInput {
  enabled: boolean;

  days: number;
}

export interface AgencySettingsInput {
  agencyName: string;

  primaryColor: string;

  notificationSoundSet?: string;

  taskAutoArchive?: TaskAutoArchiveInput;

  voiceInput?: VoiceInputSettingsInput;

  voiceInputApiKey?: string;

  clearApiKey?: boolean;
}

export interface TranscribeAudioInput {
  file: Blob;

  filename?: string;
}

export interface CreateAgentInput {
  username: string;
  displayName?: string;
  description?: string;

  prompt?: string;
}

export interface UpdateProjectPluginInput {
  enabled: boolean;
  allowTaskCreation: boolean;
  maxPriority: number;
  settingsJson?: string;
}

export interface UpdateSiteInput {
  title?: string;
  description?: string;
  onlyCreatorMayEdit?: boolean;
  agencyWide?: boolean;
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

export interface PromptAccess {
  canEdit: boolean;
  canPropose: boolean;
  canApprove: boolean;
}

export interface PromptWithAccess extends PromptAccess {
  prompt: Prompt;
}

export interface ListPromptVersionsOptions {
  limit?: number;

  beforeVersion?: bigint | number;
}

export interface CreatePromptProposalInput {
  scope: PromptScope;
  target?: PromptTarget;

  content: string;

  note?: string;
}

export interface ListPromptProposalsFilter {
  target?: PromptTarget;
  scope?: PromptScope;
  status?: PromptProposalStatus;

  mine?: boolean;
  limit?: number;
  beforeId?: bigint | number;
}

export interface PromptProposalDecision {
  proposal: PromptProposal;

  prompt?: Prompt;
}

export type { PromptActor, PromptProposal };
export { PromptProposalStatus };

export interface ProjectAgentPermissionsInput {
  canListTasks: boolean;
  canCreateTasks: boolean;
  canUpdateTasks: boolean;
}

export type KpiPage =
  | { kind: "agency"; id?: undefined }
  | { kind: "project" | "agent"; id: string };

export interface KpiWidgetInput {
  id: string;

  kind: string;

  width: number;
  height: number;

  range?: string;

  scope?: string;
}

export interface KpiPageLayout {
  widgets: KpiWidgetInput[];

  inherited: boolean;

  canEdit: boolean;

  fallback: KpiWidgetInput[];
}

export interface KpiLayoutDefaults {
  project: KpiWidgetInput[];
  agent: KpiWidgetInput[];

  canEdit: boolean;
}

export interface KpiLayoutDefaultsInput {
  project: KpiWidgetInput[];
  agent: KpiWidgetInput[];
}

export class AgencyClient {
  readonly baseUrl: string;

  private readonly authHeader?: string;

  private readonly agents: Client<typeof AgentService>;
  private readonly settings: Client<typeof SettingsService>;
  private readonly kpiLayouts: Client<typeof KpiLayoutService>;
  private readonly tasks: Client<typeof AgencyTaskService>;
  private readonly tags: Client<typeof TagService>;
  private readonly projects: Client<typeof ProjectService>;
  private readonly prompts: Client<typeof PromptService>;
  private readonly taskTemplates: Client<typeof TaskTemplateService>;
  private readonly dashboard: Client<typeof AgencyDashboardService>;
  private readonly terminals: Client<typeof AgencyTerminalService>;
  private readonly desktops: Client<typeof AgencyDesktopService>;
  private readonly notifications: Client<typeof AgencyNotificationService>;
  private readonly sites: Client<typeof SiteService>;
  private readonly plugins: Client<typeof PluginService>;
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
    this.kpiLayouts = createClient(KpiLayoutService, transport);
    this.tasks = createClient(AgencyTaskService, transport);
    this.tags = createClient(TagService, transport);
    this.projects = createClient(ProjectService, transport);
    this.prompts = createClient(PromptService, transport);
    this.taskTemplates = createClient(TaskTemplateService, transport);
    this.dashboard = createClient(AgencyDashboardService, transport);
    this.terminals = createClient(AgencyTerminalService, transport);
    this.desktops = createClient(AgencyDesktopService, transport);
    this.notifications = createClient(AgencyNotificationService, transport);
    this.sites = createClient(SiteService, transport);
    this.plugins = createClient(PluginService, transport);
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
    input: {
      displayName: string;
      description: string;

      promptAutoApprove?: boolean;
    },
    options?: CallOptions,
  ): Promise<Agent> {
    const res = await this.agents.updateAgent(
      {
        id,
        displayName: input.displayName,
        description: input.description,
        promptAutoApprove: input.promptAutoApprove,
      },
      options,
    );
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

  async setProjectArchiveAfterDays(
    id: string,
    days: number | null,
    options?: CallOptions,
  ): Promise<Project> {
    const res = await this.projects.setProjectArchiveAfterDays(
      { id, archiveAfterDays: days ?? undefined },
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

  async listProjectTagReservations(
    projectId: string,
    options?: CallOptions,
  ): Promise<ProjectTagReservation[]> {
    const res = await this.projects.listProjectTagReservations(
      { projectId, projectIds: [] },
      options,
    );
    return res.reservations;
  }

  async listProjectTagReservationsFor(
    projectIds: string[],
    options?: CallOptions,
  ): Promise<ProjectTagReservation[]> {
    if (!projectIds.length) return [];
    const res = await this.projects.listProjectTagReservations(
      { projectId: "", projectIds },
      options,
    );
    return res.reservations;
  }

  async updateProjectTagReservations(
    projectId: string,
    reservations: { tagName: string; reserved: number }[],
    options?: CallOptions,
  ): Promise<ProjectTagReservation[]> {
    const res = await this.projects.updateProjectTagReservations(
      {
        projectId,
        reservations: reservations.map((reservation) => ({
          tagName: reservation.tagName,
          reserved: reservation.reserved,
        })),
      },
      options,
    );
    return res.reservations;
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

  async listTaskTemplates(
    projectId: string,
    includeArchived = false,
    options?: CallOptions,
  ): Promise<TaskTemplate[]> {
    const res = await this.taskTemplates.listTaskTemplates(
      { projectId, includeArchived },
      options,
    );
    return res.templates;
  }

  async getTaskTemplate(
    id: string,
    options?: CallOptions,
  ): Promise<TaskTemplate> {
    const res = await this.taskTemplates.getTaskTemplate({ id }, options);
    return res.template!;
  }

  async createTaskTemplate(
    input: CreateTaskTemplateInput,
    options?: CallOptions,
  ): Promise<{ template: TaskTemplate; warnings: string[] }> {
    const res = await this.taskTemplates.createTaskTemplate(
      {
        projectId: input.projectId,
        name: input.name,
        description: input.description ?? "",
        content: templateContent(input.content),
      },
      options,
    );
    return { template: res.template!, warnings: res.warnings };
  }

  async updateTaskTemplate(
    id: string,
    input: UpdateTaskTemplateInput,
    options?: CallOptions,
  ): Promise<{
    template: TaskTemplate;
    versionCreated: boolean;
    warnings: string[];
  }> {
    const res = await this.taskTemplates.updateTaskTemplate(
      {
        id,
        name: input.name,
        description: input.description ?? "",
        content: input.content ? templateContent(input.content) : undefined,
      },
      options,
    );
    return {
      template: res.template!,
      versionCreated: res.versionCreated,
      warnings: res.warnings,
    };
  }

  async archiveTaskTemplate(
    id: string,
    options?: CallOptions,
  ): Promise<TaskTemplate> {
    const res = await this.taskTemplates.archiveTaskTemplate({ id }, options);
    return res.template!;
  }

  async unarchiveTaskTemplate(
    id: string,
    options?: CallOptions,
  ): Promise<TaskTemplate> {
    const res = await this.taskTemplates.unarchiveTaskTemplate({ id }, options);
    return res.template!;
  }

  async listTaskTemplateVersions(
    templateId: string,
    options?: CallOptions,
  ): Promise<TaskTemplateVersion[]> {
    const res = await this.taskTemplates.listTaskTemplateVersions(
      { templateId },
      options,
    );
    return res.versions;
  }

  async restoreTaskTemplateVersion(
    templateId: string,
    version: number,
    options?: CallOptions,
  ): Promise<{ template: TaskTemplate; createdVersion: number }> {
    const res = await this.taskTemplates.restoreTaskTemplateVersion(
      { templateId, version: BigInt(version) },
      options,
    );
    return {
      template: res.template!,
      createdVersion: Number(res.createdVersion),
    };
  }

  async validateTaskTemplate(
    projectId: string,
    content: TaskTemplateContentInput,
    options?: CallOptions,
  ): Promise<TaskTemplateValidationResult> {
    const res = await this.taskTemplates.validateTaskTemplate(
      { projectId, content: templateContent(content) },
      options,
    );
    return {
      valid: res.valid,
      errors: res.errors,
      warnings: res.warnings,
      undeclaredVariables: res.undeclaredVariables,
      unusedArguments: res.unusedArguments,
    };
  }

  async renderTaskTemplate(
    input: RenderTaskTemplateInput,
    options?: CallOptions,
  ): Promise<{
    templateId: string;
    templateName: string;
    version: number;
    prompt: string;
    title: string;
    priority: TaskPriority;
    size: TaskSize;
    tagIds: string[];
    teamIds: number[];
    leadActorIds: string[];
    argsJson: string;
  }> {
    const res = await this.taskTemplates.renderTaskTemplate(
      {
        templateId: input.templateId ?? "",
        version: BigInt(input.version ?? 0),
        argsJson: input.args ? JSON.stringify(input.args) : "",
        projectId: input.projectId ?? "",
        content: input.content ? templateContent(input.content) : undefined,
      },
      options,
    );
    return {
      templateId: res.templateId,
      templateName: res.templateName,
      version: Number(res.version),
      prompt: res.prompt,
      title: res.title,
      priority: res.priority,
      size: res.size,
      tagIds: res.tagIds,
      teamIds: res.teamIds,
      leadActorIds: res.leadActorIds,
      argsJson: res.argsJson,
    };
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

  async getPromptWithAccess(
    scope: PromptScope,
    target?: PromptTarget,
    options?: CallOptions,
  ): Promise<PromptWithAccess> {
    const res = await this.prompts.getPrompt(
      {
        scope,
        agentId: target?.agentId ?? "",
        teamId: BigInt(target?.teamId ?? 0),
        projectId: target?.projectId ?? "",
      },
      options,
    );
    return {
      prompt: res.prompt!,
      canEdit: res.canEdit,
      canPropose: res.canPropose,
      canApprove: res.canApprove,
    };
  }

  async setPrompt(
    scope: PromptScope,
    content: string,
    target?: PromptTarget,
    note?: string,
    options?: CallOptions,
  ): Promise<Prompt> {
    const res = await this.prompts.setPrompt(
      {
        scope,
        agentId: target?.agentId ?? "",
        teamId: BigInt(target?.teamId ?? 0),
        projectId: target?.projectId ?? "",
        content,
        note: note ?? "",
      },
      options,
    );
    return res.prompt!;
  }

  async listPromptVersions(
    scope: PromptScope,
    target?: PromptTarget,
    options?: ListPromptVersionsOptions,
    callOptions?: CallOptions,
  ): Promise<Prompt[]> {
    const res = await this.prompts.listPromptVersions(
      {
        scope,
        agentId: target?.agentId ?? "",
        teamId: BigInt(target?.teamId ?? 0),
        projectId: target?.projectId ?? "",
        limit: options?.limit ?? 50,
        beforeVersion: BigInt(options?.beforeVersion ?? 0),
      },
      callOptions,
    );
    return res.versions;
  }

  async createPromptProposal(
    input: CreatePromptProposalInput,
    options?: CallOptions,
  ): Promise<PromptProposal> {
    const res = await this.prompts.createPromptProposal(
      {
        scope: input.scope,
        agentId: input.target?.agentId ?? "",
        teamId: BigInt(input.target?.teamId ?? 0),
        projectId: input.target?.projectId ?? "",
        content: input.content,
        note: input.note ?? "",
      },
      options,
    );
    return res.proposal!;
  }

  async listPromptProposals(
    filter?: ListPromptProposalsFilter,
    options?: CallOptions,
  ): Promise<PromptProposal[]> {
    const target = filter?.target;
    const scope = filter?.scope;
    const mine = filter?.mine ?? !target;
    const res = await this.prompts.listPromptProposals(
      {
        scope: scope ?? PromptScope.UNSPECIFIED,
        agentId: target?.agentId ?? "",
        teamId: BigInt(target?.teamId ?? 0),
        projectId: target?.projectId ?? "",
        status: filter?.status ?? PromptProposalStatus.UNSPECIFIED,
        mine,
        limit: filter?.limit ?? 50,
        beforeId: BigInt(filter?.beforeId ?? 0),
      },
      options,
    );
    return res.proposals;
  }

  async decidePromptProposal(
    id: bigint | number,
    approve: boolean,
    note = "",
    options?: CallOptions,
  ): Promise<PromptProposalDecision> {
    const res = await this.prompts.decidePromptProposal(
      { id: BigInt(id), approve, note },
      options,
    );
    return { proposal: res.proposal!, prompt: res.prompt };
  }

  async approvePromptProposal(
    id: bigint | number,
    note = "",
    options?: CallOptions,
  ): Promise<PromptProposalDecision> {
    return this.decidePromptProposal(id, true, note, options);
  }

  async rejectPromptProposal(
    id: bigint | number,
    note = "",
    options?: CallOptions,
  ): Promise<PromptProposalDecision> {
    return this.decidePromptProposal(id, false, note, options);
  }

  async withdrawPromptProposal(
    id: bigint | number,
    note = "",
    options?: CallOptions,
  ): Promise<PromptProposal> {
    const res = await this.prompts.withdrawPromptProposal(
      { id: BigInt(id), note },
      options,
    );
    return res.proposal!;
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

  async getKpiLayout(
    page: KpiPage,
    options?: CallOptions,
  ): Promise<KpiPageLayout> {
    const res = await this.kpiLayouts.getKpiLayout(
      { kind: kpiPageKind(page), id: page.id ?? "" },
      options,
    );
    return kpiPageLayout(res);
  }

  async updateKpiLayout(
    page: KpiPage,
    widgets: KpiWidgetInput[],
    options?: CallOptions,
  ): Promise<KpiPageLayout> {
    const res = await this.kpiLayouts.updateKpiLayout(
      {
        kind: kpiPageKind(page),
        id: page.id ?? "",
        layout: { widgets: widgets.map(kpiWidget) },
      },
      options,
    );
    return kpiPageLayout(res);
  }

  async resetKpiLayout(
    page: KpiPage,
    options?: CallOptions,
  ): Promise<KpiPageLayout> {
    const res = await this.kpiLayouts.resetKpiLayout(
      { kind: kpiPageKind(page), id: page.id ?? "" },
      options,
    );
    return kpiPageLayout(res);
  }

  async getKpiLayoutDefaults(
    options?: CallOptions,
  ): Promise<KpiLayoutDefaults> {
    const res = await this.kpiLayouts.getKpiLayoutDefaults({}, options);
    return {
      project: kpiWidgets(res.project),
      agent: kpiWidgets(res.agent),
      canEdit: res.canEdit,
    };
  }

  async updateKpiLayoutDefaults(
    defaults: KpiLayoutDefaultsInput,
    options?: CallOptions,
  ): Promise<KpiLayoutDefaults> {
    const res = await this.kpiLayouts.updateKpiLayoutDefaults(
      {
        project: { widgets: defaults.project.map(kpiWidget) },
        agent: { widgets: defaults.agent.map(kpiWidget) },
      },
      options,
    );
    return {
      project: kpiWidgets(res.project),
      agent: kpiWidgets(res.agent),
      canEdit: res.canEdit,
    };
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
          notificationSoundSet: input.notificationSoundSet ?? "",
          taskAutoArchive: input.taskAutoArchive
            ? {
                enabled: input.taskAutoArchive.enabled,
                days: input.taskAutoArchive.days,
              }
            : undefined,
          voiceInput: input.voiceInput
            ? {
                enabled: input.voiceInput.enabled,
                baseUrl: input.voiceInput.baseUrl,
                model: input.voiceInput.model,
              }
            : undefined,
        },
        voiceInputApiKey: input.voiceInputApiKey ?? "",
        clearApiKey: input.clearApiKey ?? false,
      },
      options,
    );
    return res.settings!;
  }

  async transcribeAudio(
    input: TranscribeAudioInput,
    options?: CallOptions,
  ): Promise<string> {
    const form = new FormData();
    form.append("file", input.file, input.filename ?? "dictation.webm");

    const res = await fetch(`${this.baseUrl}/api/rest/speech/transcribe`, {
      method: "POST",
      body: form,
      headers: this.restHeaders(),
      signal: options?.signal,
    });
    if (!res.ok) {
      throw new Error(await restError(res));
    }
    const payload = (await res.json()) as { text?: string };
    return payload.text ?? "";
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
        includeArchived: filter?.includeArchived ?? false,
        parentTaskId: filter?.parentTaskId ?? "",
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
        templateId: input.templateId ?? "",
        templateVersion: BigInt(input.templateVersion ?? 0),
        templateArgsJson: input.templateArgs
          ? JSON.stringify(input.templateArgs)
          : "",
        templatePromptOverride: input.templatePromptOverride ?? false,
        parentTaskId: input.parentTaskId ?? "",
        interactive: input.interactive,
        originTaskId: input.originTaskId ?? "",
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
        setModel: input.setModel
          ? {
              provider: input.setModel.provider,
              modelId: input.setModel.modelId,
              thinkingLevel: input.setModel.thinkingLevel ?? "",
            }
          : undefined,
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

  async releaseTaskHold(
    id: string,
    input: ReleaseTaskHoldInput,
    options?: CallOptions,
  ): Promise<Task> {
    const res = await this.tasks.releaseTaskHold(
      {
        id,
        plugin: input.plugin,
        key: input.key,
        mode: input.mode ?? TaskHoldReleaseMode.SETTLE,
        prompt: input.prompt ?? "",
      },
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

  async listTaskComments(
    taskId: string,
    options?: CallOptions,
  ): Promise<ListTaskCommentsResult> {
    const res = await this.tasks.listTaskComments({ taskId }, options);
    return { comments: res.comments, unread: res.unread };
  }

  async addTaskComment(
    input: AddTaskCommentInput,
    options?: CallOptions,
  ): Promise<TaskComment> {
    const res = await this.tasks.addTaskComment(
      {
        taskId: input.taskId,
        body: input.body,
        attachmentIds: input.attachmentIds ?? [],
      },
      options,
    );
    return res.comment!;
  }

  async updateTaskComment(
    id: bigint,
    body: string,
    options?: CallOptions,
  ): Promise<TaskComment> {
    const res = await this.tasks.updateTaskComment({ id, body }, options);
    return res.comment!;
  }

  async deleteTaskComment(id: bigint, options?: CallOptions): Promise<void> {
    await this.tasks.deleteTaskComment({ id }, options);
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
    options?: RemoveTaskReferenceOptions,
  ): Promise<Task> {
    const res = await this.tasks.removeTaskReference(
      { id, referencedTaskId, reverse: options?.reverse ?? false },
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
      input.commentAttachment
        ? { kind: "comment" }
        : input.taskId !== undefined
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
        "uploadAttachment: one of taskId, commentAttachment, agentId, projectId, userId, agencyLogo, or agencyFavicon is required",
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

  async listPlugins(options?: CallOptions): Promise<Plugin[]> {
    const res = await this.plugins.listPlugins({}, options);
    return res.plugins;
  }

  async getPlugin(id: string, options?: CallOptions): Promise<Plugin> {
    const res = await this.plugins.getPlugin({ id }, options);
    return res.plugin!;
  }

  async inspectPlugin(
    bundle: Uint8Array,
    options?: CallOptions,
  ): Promise<PluginPreview> {
    const res = await this.plugins.inspectPlugin({ bundle }, options);
    return res.preview!;
  }

  async installPlugin(
    bundle: Uint8Array,
    options?: CallOptions,
  ): Promise<Plugin> {
    const res = await this.plugins.installPlugin({ bundle }, options);
    return res.plugin!;
  }

  async setPluginEnabled(
    id: string,
    enabled: boolean,
    options?: CallOptions,
  ): Promise<Plugin> {
    const res = await this.plugins.setPluginEnabled({ id, enabled }, options);
    return res.plugin!;
  }

  async uninstallPlugin(
    id: string,
    purge = false,
    options?: CallOptions,
  ): Promise<void> {
    await this.plugins.uninstallPlugin({ id, purge }, options);
  }

  async listPluginLogs(
    id: string,
    query: { limit?: number; beforeId?: number } = {},
    options?: CallOptions,
  ): Promise<PluginLog[]> {
    const res = await this.plugins.listPluginLogs(
      {
        id,
        limit: query.limit ?? 100,
        beforeId: BigInt(query.beforeId ?? 0),
      },
      options,
    );
    return res.logs;
  }

  async listPluginRuntimeLogs(
    pluginId: string,
    query: {
      scope?: string;
      targetId?: string;
      minLevel?: string;
      search?: string;
      limit?: number;
      beforeId?: number;
    } = {},
    options?: CallOptions,
  ): Promise<PluginLog[]> {
    const res = await this.plugins.listPluginRuntimeLogs(
      {
        pluginId,
        scope: query.scope ?? "",
        targetId: query.targetId ?? "",
        minLevel: query.minLevel ?? "",
        search: query.search ?? "",
        limit: query.limit ?? 100,
        beforeId: BigInt(query.beforeId ?? 0),
      },
      options,
    );
    return res.logs;
  }

  async getPluginSettings(
    pluginId: string,
    scope: string,
    targetId = "",
    options?: CallOptions,
  ): Promise<PluginSettings> {
    const res = await this.plugins.getPluginSettings(
      { pluginId, scope, targetId },
      options,
    );
    return res.settings!;
  }

  async updatePluginSettings(
    pluginId: string,
    scope: string,
    targetId: string,
    settings: Record<string, unknown>,
    options?: CallOptions,
  ): Promise<PluginSettings> {
    const res = await this.plugins.updatePluginSettings(
      {
        pluginId,
        scope,
        targetId,
        settingsJson: JSON.stringify(settings),
      },
      options,
    );
    return res.settings!;
  }

  async previewPluginTemplate(
    pluginId: string,
    scope: string,
    targetId: string,
    field: string,
    source: string,
    samples: Record<string, unknown> = {},
    options?: CallOptions,
  ): Promise<PreviewPluginTemplateResponse> {
    return this.plugins.previewPluginTemplate(
      {
        pluginId,
        scope,
        targetId,
        field,
        source,
        samplesJson: JSON.stringify(samples),
      },
      options,
    );
  }

  async setPluginSettingsEnabled(
    pluginId: string,
    scope: string,
    targetId: string,
    enabled: boolean,
    options?: CallOptions,
  ): Promise<PluginSettings> {
    const res = await this.plugins.setPluginSettingsEnabled(
      { pluginId, scope, targetId, enabled },
      options,
    );
    return res.settings!;
  }

  async listSurfacePlugins(
    scope: string,
    targetId = "",
    options?: CallOptions,
  ): Promise<{ settings: PluginSettings[]; projectPlugins: ProjectPlugin[] }> {
    const res = await this.plugins.listSurfacePlugins(
      { scope, targetId },
      options,
    );
    return {
      settings: res.settings,
      projectPlugins: res.projectPlugins,
    };
  }

  async getProjectPlugin(
    pluginId: string,
    projectId: string,
    options?: CallOptions,
  ): Promise<ProjectPlugin> {
    const res = await this.plugins.getProjectPlugin(
      { pluginId, projectId },
      options,
    );
    return res.projectPlugin!;
  }

  async updateProjectPlugin(
    pluginId: string,
    projectId: string,
    input: UpdateProjectPluginInput,
    options?: CallOptions,
  ): Promise<ProjectPlugin> {
    const res = await this.plugins.updateProjectPlugin(
      {
        pluginId,
        projectId,
        enabled: input.enabled,
        allowTaskCreation: input.allowTaskCreation,
        maxPriority: input.maxPriority,
        settingsJson: input.settingsJson,
      },
      options,
    );
    return res.projectPlugin!;
  }

  /** Invite a plugin actor to a project so its project surface can be used. */
  async invitePluginToProject(
    pluginId: string,
    projectId: string,
    role = "member",
    options?: CallOptions,
  ): Promise<ProjectPlugin> {
    const res = await this.plugins.invitePluginToProject(
      { pluginId, projectId, role },
      options,
    );
    return res.projectPlugin!;
  }

  async listSites(
    filter?: ListSitesFilter,
    options?: CallOptions,
  ): Promise<Site[]> {
    const res = await this.sites.listSites(
      {
        originTaskId: filter?.originTaskId ?? "",
        agencyWideOnly: filter?.agencyWideOnly ?? false,
      },
      options,
    );
    return res.sites;
  }

  async getSite(id: string, options?: CallOptions): Promise<Site> {
    const res = await this.sites.getSite({ id }, options);
    return res.site!;
  }

  async updateSite(
    id: string,
    input: UpdateSiteInput,
    options?: CallOptions,
  ): Promise<Site> {
    const res = await this.sites.updateSite(
      {
        id,
        title: input.title,
        description: input.description,
        onlyCreatorMayEdit: input.onlyCreatorMayEdit,
        agencyWide: input.agencyWide,
      },
      options,
    );
    return res.site!;
  }

  async deleteSite(id: string, options?: CallOptions): Promise<void> {
    await this.sites.deleteSite({ id }, options);
  }

  async prepareSiteView(id: string, options?: CallOptions): Promise<string> {
    const res = await fetch(
      `${this.baseUrl}/api/rest/sites/${encodeURIComponent(id)}/view`,
      {
        method: "POST",
        credentials: "include",
        headers: this.restHeaders(),
        signal: options?.signal,
      },
    );
    if (!res.ok) {
      throw new Error(`prepareSiteView: ${await restError(res)}`);
    }
    const body = (await res.json()) as { content_url?: string };
    if (!body.content_url) {
      throw new Error("prepareSiteView: the agency returned no content URL");
    }
    return body.content_url;
  }

  async createSiteToken(id: string, options?: CallOptions): Promise<string> {
    const res = await fetch(
      `${this.baseUrl}/api/rest/sites/${encodeURIComponent(id)}/token`,
      {
        method: "POST",
        credentials: "include",
        headers: this.restHeaders(),
        signal: options?.signal,
      },
    );
    if (!res.ok) {
      throw new Error(`createSiteToken: ${await restError(res)}`);
    }
    const body = (await res.json()) as { token?: string };
    return body.token ?? "";
  }

  async watchRealtime(
    onEvent: (event: RealtimeEvent) => void,
    options?: WatchRealtimeOptions,
  ): Promise<void> {
    const stream = await this.realtime.watch({}, { signal: options?.signal });
    for await (const message of stream) {
      if (!message.event) continue;
      const kind = realtimeKindName(message.event);
      if (kind) onEvent({ kind, task: message.event.task });
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

  async listTaskToolCalls(
    taskId: string,
    options?: ListTaskToolCallsOptions,
  ): Promise<ListTaskToolCallsResult> {
    const res = await this.tasks.listTaskToolCalls(
      {
        taskId,
        toolNames: options?.toolNames ?? [],
        errorsOnly: options?.errorsOnly ?? false,
        interruptedOnly: options?.interruptedOnly ?? false,
        search: options?.search ?? "",
        limit: options?.limit ?? 0,
        offset: options?.offset ?? 0,
      },
      { signal: options?.signal },
    );
    return { calls: res.calls, total: res.total, names: res.names };
  }

  async listTaskTools(
    taskId: string,
    options?: CallOptions,
  ): Promise<{ tools: TaskTool[]; recorded: boolean }> {
    const res = await this.tasks.listTaskTools({ taskId }, options);
    return { tools: res.tools, recorded: res.recorded };
  }

  async getTaskToolCall(
    taskId: string,
    callId: string,
    options?: GetTaskToolCallOptions,
  ): Promise<TaskToolCallDetail> {
    const res = await this.tasks.getTaskToolCall(
      { taskId, callId, full: options?.full ?? false },
      { signal: options?.signal },
    );
    return res.call!;
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

  async removeTaskQueuedMessage(
    id: string,
    queueId: string,
    options?: CallOptions,
  ): Promise<void> {
    await this.tasks.removeTaskQueuedMessage({ id, queueId }, options);
  }

  async restartTaskSession(id: string, options?: CallOptions): Promise<void> {
    await this.tasks.restartTaskSession({ id }, options);
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
        isRequired: input.isRequired ?? false,
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
        isRequired: input.isRequired ?? false,
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

  async getAgentHealthSettings(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentHealthSettings> {
    const res = await this.agents.getAgentHealthSettings({ agentId }, options);
    return res.settings!;
  }

  async updateAgentHealthSettings(
    agentId: string,
    input: AgentHealthSettingsInput,
    options?: CallOptions,
  ): Promise<AgentHealthSettings> {
    const res = await this.agents.updateAgentHealthSettings(
      {
        agentId,
        settings: {
          intervalSeconds:
            input.intervalSeconds === undefined
              ? undefined
              : BigInt(input.intervalSeconds),
        },
      },
      options,
    );
    return res.settings!;
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
  ): Promise<TaskUsageWithRuns> {
    const res = await this.dashboard.getTaskUsage({ taskId }, options);
    return Object.assign(res.usage!, { runs: res.runs });
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

  async getAgentHealth(
    agentId: string,
    options?: CallOptions,
  ): Promise<AgentHealthSample | undefined> {
    const res = await this.dashboard.getAgentHealth({ agentId }, options);
    return res.sample;
  }

  async getAgentHealthHistory(
    agentId: string,
    sinceMs = 0,
    untilMs = 0,
    maxPoints = 720,
    options?: CallOptions,
  ): Promise<AgentHealthSample[]> {
    const res = await this.dashboard.getAgentHealthHistory(
      {
        agentId,
        sinceMs: BigInt(sinceMs),
        untilMs: BigInt(untilMs),
        maxPoints,
      },
      options,
    );
    return res.samples;
  }

  async getUsage(
    agentId = "",
    projectId = "",
    options?: GetUsageOptions,
  ): Promise<GetUsageResponse> {
    const { sinceMs, untilMs, model, ...callOptions } = options ?? {};
    return await this.dashboard.getUsage(
      {
        agentId,
        projectId,
        sinceMs: BigInt(sinceMs ?? 0),
        untilMs: BigInt(untilMs ?? 0),
        model: model ?? "",
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
    case RealtimeKindPb.SITES:
      return "sites";
    case RealtimeKindPb.HEALTH:
      return "health";
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

function kpiPageKind(page: KpiPage): KpiPageKindPb {
  switch (page.kind) {
    case "agency":
      return KpiPageKindPb.AGENCY;
    case "project":
      return KpiPageKindPb.PROJECT;
    case "agent":
      return KpiPageKindPb.AGENT;
  }
}

function kpiPageLayout(res: {
  layout?: KpiLayout;
  inherited: boolean;
  canEdit: boolean;
  fallback?: KpiLayout;
}): KpiPageLayout {
  return {
    widgets: kpiWidgets(res.layout),
    inherited: res.inherited,
    canEdit: res.canEdit,
    fallback: kpiWidgets(res.fallback),
  };
}

function kpiWidgets(layout: KpiLayout | undefined): KpiWidgetInput[] {
  return (layout?.widgets ?? []).map((widget) => ({
    id: widget.id,
    kind: widget.kind,
    width: widget.width,
    height: widget.height,
    range: widget.range,
    scope: widget.scope,
  }));
}

function kpiWidget(widget: KpiWidgetInput): KpiWidget {
  return {
    id: widget.id,
    kind: widget.kind,
    width: widget.width,
    height: widget.height,
    range: widget.range ?? "",
    scope: widget.scope ?? "",
  } as KpiWidget;
}

function templateContent(input: TaskTemplateContentInput): TaskTemplateContent {
  return {
    body: input.body,
    titleTemplate: input.titleTemplate ?? "",
    args: (input.args ?? []).map((arg: TaskTemplateArgInput) => ({
      key: arg.key,
      label: arg.label ?? "",
      description: arg.description ?? "",
      type: arg.type ?? TaskTemplateArgType.TEXT,
      required: arg.required ?? false,
      defaultValue: arg.defaultValue ?? "",
      options: (arg.options ?? []).map((option) => ({
        value: option.value,
        label: option.label ?? "",
      })),
    })),
    defaultPriority: input.defaultPriority,
    defaultSize: input.defaultSize,
    defaultTagIds: input.defaultTagIds ?? [],
    defaultTeamIds: input.defaultTeamIds ?? [],
    defaultLeadActorIds: input.defaultLeadActorIds ?? [],
  } as TaskTemplateContent;
}
