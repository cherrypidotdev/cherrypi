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
  bearerAuthInterceptor,
  csrfInterceptor,
  readCookie,
  CSRF_COOKIE,
  CSRF_HEADER,
  type AddTaskCommentInput,
  type AddToTaskOptions,
  type AgentTaskLimitsInput,
  type AgentDefaultsInput,
  type AgentHealthSettingsInput,
  type AttachmentReplaceInput,
  type AttachmentUploadInput,
  type AttentionEvent,
  type AttentionNotification,
  type BulkUpdateTaskInput,
  type BulkUpdateTaskResult,
  type CallOptions,
  type CreateTaskInput,
  type CreateTaskTemplateInput,
  type CreateTagInput,
  type ExtensionUiResponse,
  type GetTaskHistoryOptions,
  type GetTaskHistoryResult,
  type GetDesktopResult,
  type ListSitesFilter,
  type ListTagsFilter,
  type ListTaskToolCallsOptions,
  type ListTaskToolCallsResult,
  type ListTaskCommentsResult,
  type ListTasksFilter,
  type ProjectLimitInput,
  type ProjectTagLimitInput,
  type QuestionnaireAnswerInput,
  type RealtimeEvent,
  type RealtimeKind,
  type WatchRealtimeOptions,
  type ReopenTaskOptions,
  type ReleaseTaskHoldInput,
  type SetTaskModelInput,
  type SetTaskLeadsInput,
  type SetTaskFavoriteInput,
  type SetTaskSubscriptionInput,
  type SetTaskTeamsInput,
  type TagLimitInput,
  type TaskAttributeInput,
  type TaskLinkInput,
  type TaskModelInput,
  type TaskTemplateArgInput,
  type TaskTemplateArgOptionInput,
  type TaskTemplateContentInput,
  type TaskTemplateValidationResult,
  type TaskReferenceInput,
  type RenderTaskTemplateInput,
  type UpdateTagInput,
  type UpdateTaskInput,
  type UpdateTaskTemplateInput,
  type WatchTaskOptions,
} from "./common.js";

export {
  AgencyClient,
  AgentStatus,
  AgentVersionStatus,
  AgencyWebhookEvent,
  UNKNOWN_USAGE_MODEL,
  type AgencyClientOptions,
  type AgencyNotification,
  type AgencySettings,
  type AgencySettingsInput,
  type VoiceInput,
  type VoiceInputSettingsInput,
  type TranscribeAudioInput,
  type AgencyUsageResponse,
  type AgencyUsageModel,
  type AgentUsageRun,
  type TaskUsageWithRuns,
  type GetUsageOptions,
  type RemoveTaskReferenceOptions,
  type AgencyWebhook,
  type AgentTaskUsage,
  type Agent,
  type AgentLog,
  AgentLogLevel,
  type CreateAgentInput,
  type AgentProjectPrompt,
  type CreateProjectInput,
  type CreatePromptProposalInput,
  type ListPromptProposalsFilter,
  type ListPromptVersionsOptions,
  type Prompt,
  type PromptAccess,
  type PromptActor,
  type PromptProposal,
  type PromptProposalDecision,
  PromptProposalStatus,
  type PromptTarget,
  type PromptWithAccess,
  type Project,
  type ProjectAgent,
  type ProjectTagReservation,
  type AgentProjectMembership,
  type ProjectAgentPermissionsInput,
  type Site,
  type UpdateProjectInput,
  type UpdateProjectPluginInput,
  type UpdateSiteInput,
  type SystemPromptSection,
  PromptScope,
} from "./agency.js";

export * from "./gen/buf/agency/v1/plugins_pb.js";

export * from "./gen/buf/agency/v1/task_pb.js";

export * from "./gen/buf/agency/v1/task_templates_pb.js";

export * from "./gen/buf/agency/v1/attachments_pb.js";

export * from "./gen/buf/agency/v1/artifacts_pb.js";
export {
  AgentDefaultsSchema,
  AgentHealthSettingsSchema,
  AgentTaskLimitsSchema,
  ProjectLimitSchema,
  ProjectTagLimitSchema,
  TagLimitSchema,
} from "./gen/buf/agency/v1/agents_pb.js";
export type {
  AgentDefaults,
  AgentHealthSettings,
  AgentTaskLimits,
  AgentToken,
  ProjectLimit,
  ProjectTagLimit,
  TagLimit,
} from "./gen/buf/agency/v1/agents_pb.js";

export type {
  AgentDiskUsage,
  AgentGpuUsage,
  AgentHealthSample,
  AgentMemory,
  AgentNetworkUsage,
  GetAgentHealthHistoryRequest,
  GetAgentHealthHistoryResponse,
  GetAgentHealthRequest,
  GetAgentHealthResponse,
} from "./gen/buf/agency/v1/dashboard_pb.js";

export * from "./gen/buf/agent/v1/dashboard_pb.js";
export {
  AgentTextSchema,
  AgentThinkingSchema,
  ExtensionUiRequestSchema,
  LogLineSchema,
  ModelWait_Kind,
  ModelWaitSchema,
  PendingActivity_Kind,
  PendingActivitySchema,
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
  type ModelWait,
  type PendingActivity,
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
