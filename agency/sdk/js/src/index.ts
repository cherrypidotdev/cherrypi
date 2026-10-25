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

export {
  basicAuthHeader,
  basicAuthInterceptor,
  bearerAuthInterceptor,
  csrfInterceptor,
  readCookie,
  CSRF_COOKIE,
  CSRF_HEADER,
  type AddToTaskOptions,
  type AgentTaskLimitsInput,
  type AgentDefaultsInput,
  type AttachmentReplaceInput,
  type AttachmentUploadInput,
  type AttentionEvent,
  type AttentionNotification,
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
  type ProjectLimitInput,
  type ProjectTagLimitInput,
  type QuestionnaireAnswerInput,
  type RealtimeEvent,
  type RealtimeKind,
  type WatchRealtimeOptions,
  type ReopenTaskOptions,
  type SetTaskModelInput,
  type SetTaskLeadsInput,
  type SetTaskSubscriptionInput,
  type SetTaskTeamsInput,
  type TagLimitInput,
  type TaskAttributeInput,
  type TaskLinkInput,
  type TaskReferenceInput,
  type UpdateTagInput,
  type UpdateTaskInput,
  type WatchTaskOptions,
} from "./common.js";

export {
  AgencyClient,
  AgentStatus,
  AgentVersionStatus,
  AgencyWebhookEvent,
  type AgencyClientOptions,
  type AgencyNotification,
  type AgencySettings,
  type AgencySettingsInput,
  type AgencyUsageResponse,
  type AgencyWebhook,
  type AgentTaskUsage,
  type Agent,
  type AgentCredential,
  type AgentLog,
  AgentLogLevel,
  type CreateAgentInput,
  type AgentProjectPrompt,
  type CreateProjectInput,
  type Prompt,
  type PromptTarget,
  type Project,
  type ProjectAgent,
  type AgentProjectMembership,
  type ProjectAgentPermissionsInput,
  type UpdateProjectInput,
  type SystemPromptSection,
  PromptScope,
} from "./agency.js";

export * from "./gen/buf/agency/v1/task_pb.js";

export * from "./gen/buf/agency/v1/attachments_pb.js";

export * from "./gen/buf/agency/v1/artifacts_pb.js";
export {
  AgentDefaultsSchema,
  AgentTaskLimitsSchema,
  ProjectLimitSchema,
  ProjectTagLimitSchema,
  TagLimitSchema,
} from "./gen/buf/agency/v1/agents_pb.js";
export type {
  AgentDefaults,
  AgentTaskLimits,
  ProjectLimit,
  ProjectTagLimit,
  TagLimit,
} from "./gen/buf/agency/v1/agents_pb.js";

export * from "./gen/buf/agent/v1/dashboard_pb.js";
export {
  AgentTextSchema,
  AgentThinkingSchema,
  ExtensionUiRequestSchema,
  LogLineSchema,
  RunSettledSchema,
  RunStartedSchema,
  TaskConfigSchema,
  TaskEventSchema,
  TaskModelSchema,
  TaskStatusChangedSchema,
  TaskStatusSchema,
  ToolExecutionSchema,
  ToolPhaseSchema,
  UserMessageSchema,
  type AgentText,
  type AgentThinking,
  type ExtensionUiRequest,
  type LogLine,
  type RunSettled,
  type RunStarted,
  type TaskConfig,
  type TaskEvent,
  type TaskModel,
  type TaskStatusChanged,
  TaskStatus,
  type ToolExecution,
  ToolPhase,
  type UserMessage,
} from "./gen/buf/agent/v1/task_pb.js";
export * from "./gen/buf/agent/v1/settings_pb.js";
export * from "./gen/buf/agent/v1/terminal_pb.js";
export * from "./gen/buf/agent/v1/desktop_pb.js";
