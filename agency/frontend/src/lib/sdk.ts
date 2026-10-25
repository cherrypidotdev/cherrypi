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

import {
  AgencyClient,
  AgentLogLevel,
  TaskPriority,
  TaskState,
  type AddToTaskOptions,
  type AgencyUsageResponse,
  type AgencyWebhook,
  type AgencyWebhookEvent,
  type AgentLog,
  type AgentTaskLimits,
  type AgentTaskLimitsInput,
  type BulkUpdateTaskInput,
  type BulkUpdateTaskResult,
  type CallOptions,
  type CreateTagInput,
  type CreateTaskInput,
  type ExtensionUiResponse,
  type GetDesktopResult,
  type GetTaskHistoryOptions,
  type GetTaskHistoryResult,
  type ListTagsFilter,
  type ListTasksFilter,
  type MigrateTaskHistoryResult,
  type QuestionnaireAnswerInput,
  type ReopenTaskOptions,
  type SetTaskLeadsInput,
  type SetTaskModelInput,
  type SetTaskSubscriptionInput,
  type SetTaskTeamsInput,
  type Tag,
  type Task,
  type TaskConfig,
  type TaskEvent,
  type TaskSize,
  type TaskUsage,
  type Terminal,
  type TerminalKind,
  type TerminalLog,
  type TerminalStats,
  type UpdateTagInput,
  type UpdateTaskInput,
  type WatchTaskOptions,
} from "@cherrypi/sdk";
import { ref } from "vue";

export const agency = new AgencyClient({
  baseUrl: window.location.origin,
  basicAuth: false,
});

export const currentAgentId = ref("");

export const currentAgentUsername = ref("");

export const agent = {
  listTasks: (filter?: ListTasksFilter, o?: CallOptions): Promise<Task[]> =>
    agency.listTasks(filter, o),
  getTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.getTask(id, o),
  createTask: (input: CreateTaskInput, o?: CallOptions): Promise<Task> =>
    agency.createTask(input, o),
  updateTask: (
    id: string,
    input: UpdateTaskInput,
    o?: CallOptions,
  ): Promise<Task> => agency.updateTask(id, input, o),
  queueTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.queueTask(id, o),
  promoteTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.promoteTask(id, o),
  demoteTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.demoteTask(id, o),
  dequeueTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.dequeueTask(id, o),
  mergeTasks: (ids: string[], o?: CallOptions): Promise<Task> =>
    agency.mergeTasks(ids, o),
  archiveTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.archiveTask(id, o),
  unarchiveTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.unarchiveTask(id, o),
  cleanupTaskSession: (id: string, o?: CallOptions): Promise<Task> =>
    agency.cleanupTaskSession(id, o),
  closeOutTask: (
    id: string,
    outcome: TaskState,
    o?: CallOptions,
  ): Promise<Task> => agency.closeOutTask(id, outcome, o),
  reopenTask: (
    id: string,
    reopen?: ReopenTaskOptions,
    o?: CallOptions,
  ): Promise<Task> => agency.reopenTask(id, reopen, o),
  retryTask: (id: string, o?: CallOptions): Promise<Task> =>
    agency.retryTask(id, o),
  deleteTask: (id: string, o?: CallOptions): Promise<void> =>
    agency.deleteTask(id, o),
  renameTask: (id: string, name: string, o?: CallOptions): Promise<Task> =>
    agency.renameTask(id, name, o),
  setTaskPriority: (
    id: string,
    priority: TaskPriority,
    o?: CallOptions,
  ): Promise<Task> => agency.setTaskPriority(id, priority, o),
  setTaskSize: (id: string, size: TaskSize, o?: CallOptions): Promise<Task> =>
    agency.setTaskSize(id, size, o),
  setTaskTeams: (
    id: string,
    input: SetTaskTeamsInput,
    o?: CallOptions,
  ): Promise<Task> => agency.setTaskTeams(id, input, o),
  setTaskLeads: (
    id: string,
    input: SetTaskLeadsInput,
    o?: CallOptions,
  ): Promise<Task> => agency.setTaskLeads(id, input, o),
  setTaskSubscription: (
    id: string,
    input: SetTaskSubscriptionInput,
    o?: CallOptions,
  ): Promise<Task> => agency.setTaskSubscription(id, input, o),
  bulkUpdateTask: (
    input: BulkUpdateTaskInput,
    o?: CallOptions,
  ): Promise<BulkUpdateTaskResult[]> => agency.bulkUpdateTask(input, o),
  addToTask: (
    id: string,
    prompt: string,
    o?: AddToTaskOptions,
  ): Promise<string> => agency.addToTask(id, prompt, o),
  stopTask: (id: string, o?: CallOptions): Promise<void> =>
    agency.stopTask(id, o),
  watchTask: (
    id: string,
    onEvent: (event: TaskEvent) => void,
    o?: WatchTaskOptions,
  ): Promise<void> => agency.watchTask(id, onEvent, o),
  getTaskHistory: (
    id: string,
    o?: GetTaskHistoryOptions,
  ): Promise<GetTaskHistoryResult> => agency.getTaskHistory(id, o),
  migrateTaskHistory: (
    id: string,
    o?: CallOptions,
  ): Promise<MigrateTaskHistoryResult> => agency.migrateTaskHistory(id, o),
  getTaskConfig: (id: string, o?: CallOptions): Promise<TaskConfig> =>
    agency.getTaskConfig(id, o),
  getTaskDefaults: (agentId: string, o?: CallOptions): Promise<TaskConfig> =>
    agency.getTaskDefaults(agentId, o),
  setTaskModel: (
    id: string,
    input: SetTaskModelInput,
    o?: CallOptions,
  ): Promise<TaskConfig> => agency.setTaskModel(id, input, o),
  setTaskThinkingLevel: (
    id: string,
    level: string,
    o?: CallOptions,
  ): Promise<TaskConfig> => agency.setTaskThinkingLevel(id, level, o),
  respondToExtensionUi: (
    id: string,
    requestId: string,
    response: ExtensionUiResponse,
    questionnaire?: QuestionnaireAnswerInput[],
    o?: CallOptions,
  ): Promise<void> =>
    agency.respondToExtensionUi(id, requestId, response, questionnaire, o),

  getUsage: (projectId = "", o?: CallOptions): Promise<AgencyUsageResponse> =>
    agency.getUsage(currentAgentId.value, projectId, o),
  getTaskUsage: (
    agentId: string,
    taskId: string,
    o?: CallOptions,
  ): Promise<TaskUsage> => agency.getTaskUsage(agentId, taskId, o),
  getTaskDiskUsage: (
    agentId: string,
    taskId: string,
    o?: CallOptions,
  ): Promise<bigint> => agency.getTaskDiskUsage(agentId, taskId, o),

  listTags: (filter?: ListTagsFilter, o?: CallOptions): Promise<Tag[]> =>
    agency.listTags(filter, o),
  createTag: (input: CreateTagInput, o?: CallOptions): Promise<Tag> =>
    agency.createTag(input, o),
  updateTag: (
    id: string,
    input: UpdateTagInput,
    o?: CallOptions,
  ): Promise<Tag> => agency.updateTag(id, input, o),
  archiveTag: (id: string, o?: CallOptions): Promise<Tag> =>
    agency.archiveTag(id, o),
  restoreTag: (id: string, o?: CallOptions): Promise<Tag> =>
    agency.restoreTag(id, o),

  listAgentLogs: (
    options?: {
      minLevel?: AgentLogLevel;
      limit?: number;
      beforeId?: bigint;
    },
    o?: CallOptions,
  ): Promise<AgentLog[]> =>
    agency.listAgentLogs(currentAgentId.value, options, o),

  getAgentTaskLimits: (
    agentId: string,
    o?: CallOptions,
  ): Promise<AgentTaskLimits> => agency.getAgentTaskLimits(agentId, o),
  updateAgentTaskLimits: (
    agentId: string,
    input: AgentTaskLimitsInput,
    o?: CallOptions,
  ): Promise<AgentTaskLimits> =>
    agency.updateAgentTaskLimits(agentId, input, o),

  listWebhooks: (): Promise<AgencyWebhook[]> => agency.listWebhooks(),
  createWebhook: (
    url: string,
    events: AgencyWebhookEvent[],
    secret?: string,
  ): Promise<AgencyWebhook> =>
    agency.createWebhook(currentAgentId.value, url, events, secret ?? ""),
  updateWebhook: (
    id: bigint,
    url: string,
    events: AgencyWebhookEvent[],
    secret?: string,
  ): Promise<AgencyWebhook> =>
    agency.updateWebhook(Number(id), {
      agentId: currentAgentId.value,
      url,
      events,
      secret,
    }),
  deleteWebhook: (id: bigint): Promise<void> =>
    agency.deleteWebhook(Number(id)),
  testWebhook: (id: bigint, event: AgencyWebhookEvent) =>
    agency.testWebhook(Number(id), event),

  createTerminal: (
    kind: TerminalKind,
    init?: { cols?: number; rows?: number },
  ): Promise<{ terminal: Terminal; wsPath: string }> =>
    agency.createTerminal(currentAgentId.value, kind, init),
  listTerminals: (): Promise<Terminal[]> =>
    agency.listTerminals(currentAgentId.value),
  resizeTerminal: (id: string, cols: number, rows: number): Promise<void> =>
    agency.resizeTerminal(currentAgentId.value, id, cols, rows),
  closeTerminal: (id: string): Promise<void> =>
    agency.closeTerminal(currentAgentId.value, id),
  closeAllTerminals: (filter?: { kind?: TerminalKind }): Promise<number> =>
    agency.closeAllTerminals(currentAgentId.value, filter),
  getTerminalStats: (): Promise<TerminalStats> =>
    agency.getTerminalStats(currentAgentId.value),
  listTerminalLogs: (limit?: number): Promise<TerminalLog[]> =>
    agency.listTerminalLogs(currentAgentId.value, limit),

  terminalWebSocketUrl: (terminalId: string): string =>
    agency.terminalWebSocketUrl(
      agency.terminalWsPath(currentAgentId.value, terminalId),
    ),

  getDesktop: (): Promise<GetDesktopResult> =>
    agency.getDesktop(currentAgentId.value),
  desktopWebSocketUrl: (): string =>
    agency.desktopWebSocketUrl(agency.desktopWsPath(currentAgentId.value)),
};
