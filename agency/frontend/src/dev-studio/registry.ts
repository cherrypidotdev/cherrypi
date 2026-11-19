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

import ActorCombobox from "@/components/ActorCombobox.vue";
import ActorTooltip from "@/components/ActorTooltip.vue";
import AvatarTip from "@/components/AvatarTip.vue";
import ActivityWait from "@/components/tasks/ActivityWait.vue";
import ConsoleActivity from "@/components/tasks/ConsoleActivity.vue";
import ConsoleAgentMessage from "@/components/tasks/ConsoleAgentMessage.vue";
import ConsoleUserMessage from "@/components/tasks/ConsoleUserMessage.vue";
import ExpressTaskForm from "@/components/tasks/ExpressTaskForm.vue";
import QuestionnaireWizard from "@/components/tasks/QuestionnaireWizard.vue";
import ReferencePicker from "@/components/tasks/ReferencePicker.vue";
import TaskConsole from "@/components/tasks/TaskConsole.vue";
import TaskPicker from "@/components/tasks/TaskPicker.vue";
import ThinkingBlock from "@/components/tasks/ThinkingBlock.vue";
import TaskTooltip from "@/components/TaskTooltip.vue";
import {
  blockWaiting,
  WAIT_RENDER_MS,
  type ActivityItem,
  type PendingStep,
  type RenderActivity,
  type RunState,
} from "@/lib/console";
import {
  emptyAnswer,
  type QuestionnaireAnswer,
  type QuestionnaireQuestion,
} from "@/lib/questionnaire";
import type { ReferenceDirectory } from "@/lib/references";
import type { TaskModelSelection } from "@/lib/task";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import {
  ArtifactFileSchema,
  ArtifactPartSchema,
  ArtifactSchema,
  AttachmentSchema,
  ModelWait_Kind,
  PendingActivity_Kind,
  TagSchema,
  TaskCommentSchema,
  TaskConfigSchema,
  TaskEventSchema,
  TaskLogSchema,
  TaskModelSchema,
  TaskPriority,
  TaskProgressItemSchema,
  TaskProgressItemStatus,
  TaskProgressSchema,
  TaskReferenceKind,
  TaskSchema,
  TaskSize,
  TaskState,
  TaskToolCallSchema,
  TaskToolOrigin,
  TaskToolSchema,
  TaskUsageSchema,
  ToolPhase,
  type Agent,
  type Project,
  type SystemPromptSection,
  type Task,
  type TaskEvent,
} from "@cherrypi/sdk";
import TaskDetailsStage from "./TaskDetailsStage.vue";
import TaskFollowUpComposerStage from "./TaskFollowUpComposerStage.vue";
import type { StudioComponent } from "./types";

const questions: QuestionnaireQuestion[] = [
  {
    question: "Which database should this service use?",
    header: "Database",
    multiSelect: false,
    options: [
      {
        label: "PostgreSQL",
        description:
          "Already runs in the cluster, with migrations and backups in place.",
        recommended: true,
      },
      {
        label: "SQLite",
        description: "One file, zero operations. Fine until writes contend.",
      },
    ],
  },
  {
    question: "Which test layers should I cover?",
    header: "Testing",
    multiSelect: true,
    options: [
      { label: "Unit", description: "Fast, no I/O." },
      { label: "Integration", description: "Against a real database." },
      {
        label: "End-to-end",
        description: "Drives the UI through the tunnel.",
      },
    ],
  },
];

const topics = [
  "Hosting",
  "Auth",
  "Observability",
  "Rollout",
  "Rollback",
  "Docs",
];
const longQuestions: QuestionnaireQuestion[] = topics.map((topic, index) => ({
  question: `What should the ${topic.toLowerCase()} step do?`,
  header: topic,
  multiSelect: index % 2 === 1,
  options: [
    {
      label: "Ship the default",
      description: `Use the ${topic.toLowerCase()} setup this repo already documents.`,
      recommended: true,
    },
    {
      label: "Ask me first",
      description: "Stop before touching anything and wait for a decision.",
    },
  ],
}));

const answered: QuestionnaireAnswer[] = [
  {
    selected: [0],
    customSelected: false,
    custom: "",
    notes: "**Postgres** it is — the cluster already backs it up.",
  },
  { selected: [0, 1], customSelected: false, custom: "", notes: "" },
];

const custom: QuestionnaireAnswer[] = [
  {
    selected: [],
    customSelected: true,
    custom: "**DuckDB** for the analytics sidecar.",
    notes: "",
  },
  {
    selected: [2],
    customSelected: false,
    custom: "",
    notes: "E2E only for the tunnel path.",
  },
];

const noAnswers = () => questions.map(() => emptyAnswer());

const modelConfig = create(TaskConfigSchema, {
  modelProvider: "anthropic",
  modelId: "claude-sonnet-4-5",
  modelName: "Claude Sonnet 4.5",
  models: [
    create(TaskModelSchema, {
      provider: "anthropic",
      id: "claude-sonnet-4-5",
      name: "Claude Sonnet 4.5",
      reasoning: true,
      thinkingLevels: ["off", "low", "medium", "high"],
    }),
    create(TaskModelSchema, {
      provider: "anthropic",
      id: "claude-haiku-4-5",
      name: "Claude Haiku 4.5",
      reasoning: true,
      thinkingLevels: ["off", "low"],
    }),
    create(TaskModelSchema, {
      provider: "openai",
      id: "gpt-5",
      name: "GPT-5",
      reasoning: true,
      thinkingLevels: ["off", "low", "medium", "high"],
    }),
  ],
  thinkingLevel: "medium",
  thinkingLevels: ["off", "low", "medium", "high"],
});

const modelSelection: TaskModelSelection = {
  modelProvider: "anthropic",
  modelId: "claude-sonnet-4-5",
  modelName: "Claude Sonnet 4.5",
  thinkingLevel: "medium",
};

const agentId = "agent-vlad-desk";

const reasoning =
  "The wake guard records the attempt but never backs off, so a dead tunnel is " +
  "retried in a tight cycle. I need to add the delay where the attempt is " +
  "recorded, then check the tests that count wake calls.";

const startedAgo = (ms: number) => Date.now() - ms;

const promptSender = {
  id: "4",
  username: "vladfaust",
  name: "Vlad Faust",
  kind: "user",
};

const promptText =
  "The console still shows a dead session after the agent restarts. " +
  "Reproduce it with `restart` and fix the transcript replay.";

const promptItem = {
  id: 12,
  kind: "user" as const,
  text: promptText,
  attachmentIds: [],
  sender: promptSender,
  at: Date.UTC(2026, 10, 9, 14, 5, 32),
};

const promptAttachments = [
  create(AttachmentSchema, {
    id: 8n,
    filename: "dead-session.png",
    contentType: "image/png",
    size: 182_400n,
    url:
      "data:image/svg+xml," +
      encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32">' +
          '<rect width="32" height="32" fill="#64748b"/></svg>',
      ),
  }),
  create(AttachmentSchema, {
    id: 9n,
    filename: "agent-restart.log",
    contentType: "text/plain",
    size: 12_288n,
    url: "data:text/plain,restart",
  }),
];

const consoleAgent = {
  id: "7",
  username: "reviewer",
  name: "Reviewer",
  kind: "agent",
};

const agentMessageItem = {
  id: 21,
  kind: "text" as const,
  text:
    "The replay reads the session file before the transcript is paged, so the " +
    "first page can land twice. I moved the read behind the first page and " +
    "kept the dedupe.",
  at: Date.UTC(2026, 10, 9, 14, 6, 18),
};

const runUsage = {
  provider: "refuseless",
  model: "deepseek-v4.1-flash",
  inputTokens: 18_400,
  outputTokens: 1_240,
  cost: 0.0123,
  activeMs: 42_000,
  thinkingMs: 31_000,
  modelWaitMs: 2_400,
  at: Date.UTC(2026, 10, 9, 14, 6, 18),
};

const blockStart = Date.UTC(2026, 10, 9, 14, 5, 0);

function activityBlock(
  items: ActivityItem[],
  options: { running?: boolean; pending?: PendingStep; newestId?: number } = {},
): RenderActivity {
  return {
    id: 40,
    kind: "activity",
    items: items.filter(
      (item) => item.kind !== "wait" || item.durationMs >= WAIT_RENDER_MS,
    ),
    span: items,
    waiting: blockWaiting(items),
    running: options.running ?? false,
    pending: options.pending,
  };
}

const settledThinking: ActivityItem = {
  id: 41,
  kind: "thinking",
  text:
    "The wake guard records the attempt but never backs off, so a dead tunnel " +
    "is retried in a tight cycle. The delay belongs where the attempt is " +
    "recorded; the wake tests count retries.",
  startedAt: blockStart,
  endedAt: blockStart + 9_000,
};

const bashCall: ActivityItem = {
  id: 42,
  kind: "tool",
  callId: "call-bash",
  name: "bash",
  phase: ToolPhase.FINISHED,
  args: JSON.stringify({ command: "go test ./internal/tunnel/... -run Wake" }),
  result: "ok  \tcherrypi/tunnel\t1.284s",
  details: "",
  isError: false,
  startedAt: blockStart + 9_000,
  endedAt: blockStart + 10_500,
};

const readCall: ActivityItem = {
  id: 43,
  kind: "tool",
  callId: "call-read",
  name: "read",
  phase: ToolPhase.FINISHED,
  args: JSON.stringify({ path: "/repo/agent/internal/tunnel/wake.go" }),
  result: "func (c *client) wake() { … }",
  details: "",
  isError: false,
  startedAt: blockStart + 10_500,
  endedAt: blockStart + 10_900,
};

const editCall: ActivityItem = {
  id: 44,
  kind: "tool",
  callId: "call-edit",
  name: "edit",
  phase: ToolPhase.FINISHED,
  args: JSON.stringify({ path: "/repo/agent/internal/tunnel/wake.go" }),
  result: "Edited",
  details: JSON.stringify({
    diff: "--- a/wake.go\n+++ b/wake.go\n@@ -1,2 +1,4 @@\n+wakeBackoff()\n+",
  }),
  isError: false,
  startedAt: blockStart + 10_900,
  endedAt: blockStart + 11_400,
};

const requestWait: ActivityItem = {
  id: 45,
  kind: "wait",
  waitKind: ModelWait_Kind.MODEL_REQUEST,
  model: "anthropic/claude-sonnet-4.5",
  provider: "openrouter",
  durationMs: 4_200,
  error: "",
  attempt: 0,
  reason: "",
  startedAt: blockStart + 11_400,
  endedAt: blockStart + 15_600,
};

const finishedTurn = [
  settledThinking,
  bashCall,
  readCall,
  editCall,
  requestWait,
];

const streamingThinking: ActivityItem = {
  id: 46,
  kind: "thinking",
  text:
    "Both failures come from the same place: the transcript read runs before " +
    "the first page settles, so the replay can append the same",
  startedAt: startedAgo(3_000),
};

const interruptedCall: ActivityItem = {
  id: 47,
  kind: "tool",
  callId: "call-started",
  name: "bash",
  phase: ToolPhase.STARTED,
  args: JSON.stringify({ command: "go test ./... -count=1" }),
  result: "",
  details: "",
  isError: false,
  startedAt: blockStart + 9_000,
};

const runningNow = (newestId: number): RunState => ({
  live: true,
  waitingForInput: false,
  pending: false,
  newestId,
});

const stillRun: RunState = {
  live: false,
  waitingForInput: false,
  pending: false,
};

function stamp(ms: number) {
  return {
    seconds: BigInt(Math.floor(ms / 1000)),
    nanos: Math.round((ms % 1000) * 1e6),
  };
}

const consoleStart = Date.UTC(2026, 10, 9, 14, 5, 0);

const consoleAgentMessage = {
  $typeName: "agency.v1.Agent",
  id: "agent-1",
  actorId: "100",
  username: "reviewer",
  displayName: "Reviewer",
} as unknown as Agent;

function consoleTurn(
  offsetMs: number,
  prompt: string,
  reply: string,
): TaskEvent[] {
  const at = (delta: number) => stamp(consoleStart + offsetMs + delta);
  return [
    create(TaskEventSchema, {
      at: at(0),
      event: {
        case: "userMessage",
        value: { text: prompt, sender: promptSender },
      },
    }),
    create(TaskEventSchema, {
      at: at(1_000),
      event: {
        case: "thinking",
        value: {
          delta:
            "The transcript read runs before the first page settles, so the " +
            "replay can append the same events twice.",
        },
      },
    }),
    create(TaskEventSchema, {
      at: at(9_000),
      event: {
        case: "tool",
        value: {
          toolCallId: "call-1",
          toolName: "bash",
          phase: ToolPhase.STARTED,
          argsJson: JSON.stringify({
            command: "go test ./components/tasks/...",
          }),
          resultJson: "",
        },
      },
    }),
    create(TaskEventSchema, {
      at: at(11_000),
      event: {
        case: "tool",
        value: {
          toolCallId: "call-1",
          toolName: "bash",
          phase: ToolPhase.FINISHED,
          argsJson: JSON.stringify({
            command: "go test ./components/tasks/...",
          }),
          resultJson: "ok  cherrypi/dashboard  11.284s",
        },
      },
    }),
    create(TaskEventSchema, {
      at: at(12_000),
      event: { case: "text", value: { delta: reply } },
    }),
  ];
}

const consoleTurnEvents = consoleTurn(
  0,
  "The console still shows a dead session after the agent restarts. " +
    "Reproduce it with `restart` and fix the transcript replay.",
  "The replay read the session file before the first page settled. " +
    "I moved the read behind it, so the first page can no longer land twice.",
);

const consoleLongEvents = [
  ...consoleTurn(
    0,
    "Start by reading the console's history paging.",
    "Read it: two pages, newest first.",
  ),
  ...consoleTurn(
    60_000,
    "Now check what a restart leaves behind.",
    "An unfinished call, pinned at its last recorded stamp.",
  ),
  ...consoleTurn(
    120_000,
    "And keep the pinned prompt honest.",
    "It only pins prompts an actor wrote.",
  ),
];

const studioProject = {
  $typeName: "agency.v1.Project",
  id: "p-1",
  name: "CherryPi",
  slug: "cherrypi",
  description: "The agency, the agents and this console.",
} as unknown as Project;

const studioAgentOption = {
  $typeName: "agency.v1.Agent",
  id: "agent-1",
  actorId: "100",
  username: "reviewer",
  displayName: "Reviewer",
} as unknown as Agent;

const studioTeams = [
  {
    team_id: 1,
    team_name: "Platform",
    team_description: "Owns the tunnel and the scheduler.",
    role: 2,
    member_count: 4,
    created_at: "2026-11-01T10:00:00Z",
    updated_at: "2026-11-01T10:00:00Z",
  },
];

const studioLeadCandidates = [
  {
    actor_id: "4",
    kind: "user",
    username: "vladfaust",
    name: "Vlad Faust",
  },
  {
    actor_id: "100",
    kind: "agent",
    username: "reviewer",
    name: "Reviewer",
  },
];

const studioTags = [
  create(TagSchema, { id: "t-1", name: "console", color: "#6366f1" }),
  create(TagSchema, { id: "t-2", name: "frontend", color: "#22c55e" }),
];

const studioSystemPrompt: SystemPromptSection[] = [
  {
    $typeName: "agency.v1.SystemPromptSection",
    scope: 0,
    owner: "CherryPi",
    version: 3n,
    content: "Keep the transcript honest about what the run did.",
    pinned: true,
  } as unknown as SystemPromptSection,
];

const studioTaskUsage = create(TaskUsageSchema, {
  taskId: "CHPI-541",
  inputTokens: 184_000n,
  outputTokens: 12_400n,
  cost: 0.4213,
  activeMs: 1_842_000n,
  thinkingMs: 1_204_000n,
  modelWaitMs: 96_000n,
  toolMs: 214_000n,
  contextTokens: 62_000n,
  contextWindow: 200_000n,
});

const studioProgress = create(TaskProgressSchema, {
  summary: "Fix the console's transcript replay",
  done: 2,
  total: 5,
  items: [
    create(TaskProgressItemSchema, {
      text: "Reproduce the duplicated first page",
      status: TaskProgressItemStatus.DONE,
    }),
    create(TaskProgressItemSchema, {
      text: "Read the session file after the first page",
      status: TaskProgressItemStatus.DONE,
    }),
    create(TaskProgressItemSchema, {
      text: "Cover the restart path with a test",
      status: TaskProgressItemStatus.IN_PROGRESS,
    }),
    create(TaskProgressItemSchema, {
      text: "Re-run the console suite",
      status: TaskProgressItemStatus.PENDING,
    }),
    create(TaskProgressItemSchema, {
      text: "Update the architecture note",
      status: TaskProgressItemStatus.PENDING,
    }),
  ],
});

const savingNothing = {
  assignee: false,
  priority: false,
  size: false,
  project: false,
  tags: false,
  links: false,
  teams: false,
  leader: false,
  subscription: false,
  favorite: false,
  model: false,
};

const runningTask = create(TaskSchema, {
  id: "CHPI-541",
  name: "The console shows a dead session after a restart",
  state: TaskState.RUNNING,
  priority: TaskPriority.HIGH,
  size: TaskSize.MEDIUM,
  projectId: "p-1",
  assigneeActorId: "100",
  assigneePinned: true,
  sessionId: "s-541",
  subscribed: true,
  creatorActorId: "4",
  creatorName: "Vlad Faust",
  createdAt: stamp(consoleStart),
  startedAt: stamp(consoleStart + 60_000),
  progress: studioProgress,
  tags: [studioTags[0]!],
  leader: {
    $typeName: "agency.v1.TaskLeader",
    actorId: "4",
    username: "vladfaust",
    name: "Vlad Faust",
    kind: "user",
    pictureUrl: "",
  },
  teams: [
    { $typeName: "agency.v1.TaskTeam", id: 1, name: "Platform" },
  ] as Task["teams"],
});

const backlogTask = create(TaskSchema, {
  id: "CHPI-602",
  name: "Pin the prompt the console has scrolled past",
  state: TaskState.BACKLOG,
  priority: TaskPriority.NORMAL,
  size: TaskSize.SMALL,
  projectId: "p-1",
  creatorActorId: "4",
  creatorName: "Vlad Faust",
  createdAt: stamp(consoleStart),
});

const detailsProps = {
  open: true,

  preview: {},
  saving: savingNothing,
  assigneeOptions: [studioAgentOption],
  projectOptions: [studioProject],
  projectTeams: studioTeams,
  leadCandidates: studioLeadCandidates,
  unattachedTags: studioTags,
  taskUsage: studioTaskUsage,
  liveTiming: { toolRunning: true, modelWaiting: false },
  taskDiskBytes: 18_874_368n,
  taskDiskLoading: false,
  systemPrompt: studioSystemPrompt,
  actionBusy: false,
  stopping: false,
  canCompose: true,
};

const settledAttachments = [
  create(AttachmentSchema, {
    id: 21n,
    filename: "dead-session.png",
    contentType: "image/png",
    size: 182_400n,
    url:
      "data:image/svg+xml," +
      encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64">' +
          '<rect width="64" height="64" fill="#475569"/></svg>',
      ),
    createdAt: create(TimestampSchema, stamp(consoleStart)),
    createdBy: "4",
  }),
  create(AttachmentSchema, {
    id: 22n,
    filename: "agent-restart.log",
    contentType: "text/plain",
    size: 12_288n,
    url: "data:text/plain,restart",
    createdAt: create(TimestampSchema, stamp(consoleStart + 1_000)),
    createdBy: "4",
  }),
];

const settledArtifacts = [
  create(ArtifactSchema, {
    artifactId: "a-1",
    name: "Transcript replay diff",
    description: "What the fix changed in the replay path.",
    parts: [
      create(ArtifactPartSchema, {
        filename: "replay.diff",
        mediaType: "text/x-diff",
        content: {
          case: "text",
          value:
            "--- a/components/tasks/TaskConsole.vue\n+++ b/components/tasks/TaskConsole.vue\n@@ -1,3 +1,3 @@\n-  await loadHistory();\n+  await firstPage();\n",
        },
      }),
    ],
    createdAt: create(TimestampSchema, stamp(consoleStart + 1_800_000)),
  }),
  create(ArtifactSchema, {
    artifactId: "a-2",
    name: "Console screenshot",
    parts: [
      create(ArtifactPartSchema, {
        filename: "console.png",
        mediaType: "image/png",
        content: {
          case: "file",
          value: create(ArtifactFileSchema, {
            attachmentId: 21n,
            url:
              "data:image/svg+xml," +
              encodeURIComponent(
                '<svg xmlns="http://www.w3.org/2000/svg" width="64" height="36">' +
                  '<rect width="64" height="36" fill="#0f172a"/>' +
                  '<rect x="4" y="4" width="56" height="8" fill="#334155"/>' +
                  "</svg>",
              ),
            raw: true,
          }),
        },
      }),
    ],
    createdAt: create(TimestampSchema, stamp(consoleStart + 1_900_000)),
  }),
];

const settledLogs = [
  create(TaskLogSchema, {
    id: 1n,
    actor: "user",
    actorId: "4",
    kind: "created",
    message: "The console shows a dead session after a restart",
    at: stamp(consoleStart),
  }),
  create(TaskLogSchema, {
    id: 2n,
    actor: "agent",
    actorId: "100",
    kind: "state",
    message: "State changed from Queued to Running",
    at: stamp(consoleStart + 60_000),
    dataJson: JSON.stringify({ old: "queued", new: "running" }),
  }),
  create(TaskLogSchema, {
    id: 3n,
    actor: "agent",
    actorId: "100",
    kind: "state",
    message: "State changed from Running to Settled",
    at: stamp(consoleStart + 2_400_000),
    dataJson: JSON.stringify({ old: "running", new: "settled" }),
  }),
];

const settledTask = create(TaskSchema, {
  id: "CHPI-541",
  name: "The console shows a dead session after a restart",
  prompt: "Reproduce it with `restart` and fix the transcript replay.",
  state: TaskState.SETTLED,
  priority: TaskPriority.HIGH,
  size: TaskSize.MEDIUM,
  projectId: "p-1",
  assigneeActorId: "100",
  assigneePinned: true,
  sessionId: "s-541",
  subscribed: true,
  creatorActorId: "4",
  creatorName: "Vlad Faust",
  createdAt: stamp(consoleStart),
  startedAt: stamp(consoleStart + 60_000),
  settledAt: stamp(consoleStart + 2_400_000),
  progress: studioProgress,
  attachments: settledAttachments,
  artifacts: settledArtifacts,
  logs: settledLogs,
  tags: [studioTags[0]!],
  leader: runningTask.leader,
  teams: runningTask.teams,
});

const settledComments = [
  create(TaskCommentSchema, {
    id: 1n,
    taskId: "CHPI-541",
    authorActorId: "4",
    authorName: "Vlad Faust",
    authorUsername: "vladfaust",
    authorKind: "user",
    body: "The screenshot is from before the fix — the **first page** lands twice.",
    attachments: [settledAttachments[0]!],
    createdAt: create(TimestampSchema, stamp(consoleStart + 300_000)),
    updatedAt: create(TimestampSchema, stamp(consoleStart + 300_000)),
  }),
  create(TaskCommentSchema, {
    id: 2n,
    taskId: "CHPI-541",
    authorActorId: "100",
    authorName: "Reviewer",
    authorUsername: "reviewer",
    authorKind: "agent",
    body: "Fixed in the replay path; the test covers a restart mid-turn.",
    createdAt: create(TimestampSchema, stamp(consoleStart + 2_000_000)),
    updatedAt: create(TimestampSchema, stamp(consoleStart + 2_000_000)),
  }),
];

const settledTools = {
  recorded: true,
  tools: [
    create(TaskToolSchema, {
      name: "bash",
      description: "Run a shell command in the task's workspace.",
      parametersJson:
        '{"type":"object","properties":{"command":{"type":"string","description":"Shell command"}}}',
      origin: TaskToolOrigin.BUILTIN,
      callCount: 2,
    }),
    create(TaskToolSchema, {
      name: "openGiteaPr",
      description: "Open a pull request on the project's Gitea.",
      parametersJson:
        '{"type":"object","properties":{"branch":{"type":"string"}}}',
      origin: TaskToolOrigin.PLUGIN,
      pluginId: "gitea",
      pluginName: "Gitea",
      callCount: 0,
    }),
  ],
};

const settledToolCalls = {
  calls: [
    create(TaskToolCallSchema, {
      callId: "c-1",
      name: "bash",
      argsJson: JSON.stringify({ command: "go test ./components/tasks/..." }),
      resultBytes: 2_048n,
      startedAt: stamp(consoleStart + 200_000),
      durationMs: 11_000n,
    }),
    create(TaskToolCallSchema, {
      callId: "c-2",
      name: "bash",
      argsJson: JSON.stringify({ command: "go test ./... -count=1" }),
      resultBytes: 512n,
      isError: true,
      startedAt: stamp(consoleStart + 400_000),
      durationMs: 3_400n,
    }),
    create(TaskToolCallSchema, {
      callId: "c-3",
      name: "read",
      argsJson: JSON.stringify({ path: "/repo/agent/internal/tunnel/wake.go" }),
      resultBytes: 8_192n,
      startedAt: stamp(consoleStart + 500_000),
      durationMs: 120n,
    }),
  ],
  total: 3,
  names: ["bash", "read"],
};

const settledPreview = {
  comments: { comments: settledComments, unread: 2 },
  tools: settledTools,
  toolCalls: settledToolCalls,
};

const detailsEvents = [
  "assignee-change",
  "priority-change",
  "size-change",
  "project-change",
  "toggle-team",
  "toggle-lead",
  "toggle-tag",
  "toggle-subscription",
  "reference-change",
  "open-attachment",
  "remind-plan",
  "open-media",
  "comments-read",
  "edit-links",
  "queue",
  "dequeue",
  "restart-session",
  "stop",
  "reopen",
  "close-out",
  "release-hold",
  "archive",
  "unarchive",
  "cleanup-session",
  "delete",
  "create-related",
];

const composerPreview = {
  projects: [studioProject],
  agents: [studioAgentOption],
  tags: studioTags,
};

function pickedTask(init: MessageInitShape<typeof TaskSchema>): Task {
  return create(TaskSchema, init);
}

const pickerTasks = [
  pickedTask({
    id: "CHPI-541",
    name: "The console shows a dead session after a restart",
    prompt: "Reproduce it with restart and fix the transcript replay.",
    state: TaskState.RUNNING,
    priority: TaskPriority.HIGH,
    size: TaskSize.MEDIUM,
    projectId: "p-1",
    assigneeActorId: "100",
    tags: [studioTags[0]!],
  }),
  pickedTask({
    id: "CHPI-602",
    name: "Pin the prompt the console has scrolled past",
    prompt: "Keep the pinned header honest about the newest prompt.",
    state: TaskState.BACKLOG,
    priority: TaskPriority.NORMAL,
    size: TaskSize.SMALL,
    projectId: "p-1",
    tags: [studioTags[1]!],
  }),
  pickedTask({
    id: "CHPI-588",
    name: "Count a queued steering message at once",
    prompt: "The composer waits for the next event; count it locally.",
    state: TaskState.INPUT,
    priority: TaskPriority.CRITICAL,
    size: TaskSize.BIG,
    projectId: "p-1",
    assigneeActorId: "100",
  }),
  pickedTask({
    id: "CHPI-503",
    name: "Archive a project's tasks after its agent goes offline",
    prompt: "Queue the archive and warn about the agents left behind.",
    state: TaskState.SETTLED,
    priority: TaskPriority.LOW,
    size: TaskSize.EPIC,
    projectId: "p-1",
    tags: [studioTags[0]!, studioTags[1]!],
  }),
  pickedTask({
    id: "CHPI-477",
    name: "Replay the transcript from the session file",
    prompt: "Read the session once the first page has settled.",
    state: TaskState.FAILED,
    priority: TaskPriority.NORMAL,
    size: TaskSize.MEDIUM,
    projectId: "p-1",
    error: "The agent's tunnel dropped mid-run.",
  }),
  pickedTask({
    id: "CHPI-410",
    name: "Write the architecture note for the tunnel",
    prompt: "Yamux over a Connect stream, in one diagram.",
    state: TaskState.QUEUED,
    priority: TaskPriority.NORMAL,
    size: TaskSize.BIG,
    projectId: "p-1",
    archived: true,
  }),
];

const pickerPreview = {
  tasks: pickerTasks,
  projects: [studioProject],
  agents: [studioAgentOption],
  tags: studioTags,
};

const directoryUsers = [
  {
    id: 4,
    kind: "user",
    username: "vladfaust",
    name: "Vlad Faust",
    description: "Owns the agency's tunnel and scheduler.",
    role: "admin",
    status: "active",
    password_set: true,
    totp_enabled: true,
    must_change_password: false,
    is_admin: true,
    created_at: "2026-01-04T10:00:00Z",
    last_login_at: "2026-11-09T13:00:00Z",
  },
  {
    id: 5,
    kind: "user",
    username: "marina",
    name: "Marina Petrova",
    description: "Reads the board and reviews prompts.",
    role: "user",
    status: "active",
    password_set: true,
    totp_enabled: false,
    must_change_password: false,
    is_admin: false,
    created_at: "2026-02-11T09:30:00Z",
  },
  {
    id: 6,
    kind: "user",
    username: "dmitry",
    name: "Dmitry Sokolov",
    description: "Runs the agents on the GPU host.",
    role: "user",
    status: "active",
    password_set: true,
    totp_enabled: false,
    must_change_password: false,
    is_admin: false,
    created_at: "2026-03-02T12:00:00Z",
  },
];

const pickerAgents = [
  {
    $typeName: "agency.v1.Agent",
    id: "agent-1",
    actorId: "100",
    username: "reviewer",
    displayName: "Reviewer",
  },
  {
    $typeName: "agency.v1.Agent",
    id: "agent-2",
    actorId: "101",
    username: "builder",
    displayName: "Builder",
  },
] as unknown as Agent[];

const actorPreview = { users: directoryUsers, agents: pickerAgents };

const referenceDirectoryPreview = {
  users: directoryUsers,
  agents: pickerAgents,
  projects: [studioProject],
  teams: [
    {
      id: 1,
      name: "Platform",
      description: "Owns the tunnel and the scheduler.",
      member_count: 4,
    },
  ],
} as unknown as ReferenceDirectory;

const followUpProps = {
  task: runningTask,
  config: modelConfig,
  modelSelection,
  taskUsage: studioTaskUsage,
  canCompose: true,
  canFollowUp: true,
  sending: false,
  stopping: false,
};

const cleanedUpTask = create(TaskSchema, {
  id: runningTask.id,
  name: runningTask.name,
  state: runningTask.state,
  priority: runningTask.priority,
  size: runningTask.size,
  projectId: runningTask.projectId,
  assigneeActorId: runningTask.assigneeActorId,
  sessionId: runningTask.sessionId,
  creatorActorId: "4",
  creatorName: "Vlad Faust",
  createdAt: runningTask.createdAt,
  startedAt: runningTask.startedAt,
  sessionDeletedAt: create(TimestampSchema, stamp(consoleStart + 3_600_000)),
});

export const studioComponents: StudioComponent[] = [
  {
    id: "questionnaire",
    name: "Questionnaire",
    source: "@/components/tasks/QuestionnaireWizard.vue",
    component: QuestionnaireWizard,
    scenarios: [
      {
        id: "live",
        name: "Live",
        description:
          "The waiting wizard with its model controls. Answer both questions, then submit — the emission is recorded below.",
        props: {
          questions,
          agentId,
          modelConfig,
          modelSelection,
        },
        events: ["submit", "decline", "modelChange"],
        controls: [
          {
            prop: "submitting",
            label: "Submitting",
            type: "boolean",
            initial: false,
          },
          {
            prop: "modelDisabled",
            label: "Model controls disabled",
            type: "boolean",
            initial: false,
          },
          { prop: "errors", label: "Error message", type: "text", initial: "" },
        ],
        width: 640,
      },
      {
        id: "single",
        name: "Single question",
        description: "One question needs no tab strip at all.",
        props: { questions: [questions[0]!] },
        events: ["submit", "decline"],
        width: 480,
      },
      {
        id: "long",
        name: "Many questions",
        description:
          "Six questions: the tab strip scrolls sideways and the check marks track progress.",
        props: { questions: longQuestions },
        events: ["submit", "decline"],
        width: 560,
      },
      {
        id: "submitting",
        name: "Submitting",
        description:
          "While the answers are on their way nothing is editable, model controls included.",
        props: {
          questions,
          agentId,
          modelConfig,
          modelSelection,
          submitting: true,
        },
        width: 640,
      },
      {
        id: "invalid",
        name: "Validation error",
        description: "An error from the host sits under the questions.",
        props: {
          questions,
          errors: "Pick an option for every question before submitting.",
        },
        width: 480,
      },
      {
        id: "answered",
        name: "Answered",
        description:
          "Read-only review of an answered questionnaire, attributed to its author.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          reviewAnswers: answered,
          reviewFinalNote: "Keep the docs in step with the schema.",
          answeredBy: {
            id: "4",
            username: "vladfaust",
            name: "Vlad Faust",
          },
          answeredAt: Date.UTC(2026, 10, 9, 14, 5, 32),
        },
        width: 520,
      },
      {
        id: "answered-by-agent",
        name: "Answered by an agent",
        description:
          "Read-only review of a questionnaire a lead agent answered for its sub-task.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          reviewAnswers: answered,
          answeredBy: {
            id: "9",
            kind: "agent",
            username: "reviewer",
            name: "Reviewer",
          },
          answeredAt: Date.UTC(2026, 10, 9, 14, 5, 32),
        },
        width: 520,
      },
      {
        id: "blocked",
        name: "Cannot answer",
        description:
          "A viewer who may read the task but does not lead it: the questions render read-only, with the rule instead of a form.",
        props: {
          questions,
          submitting: false,
          blockedReason:
            "Answering takes a lead of this task or an administrator. " +
            "You can read the questions here.",
        },
        width: 520,
      },
      {
        id: "custom",
        name: "Custom answers",
        description:
          "The free-text escape chosen on one question and notes kept on the other.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          reviewAnswers: custom,
          answeredBy: { id: "4", username: "vladfaust", name: "Vlad Faust" },
        },
        width: 520,
      },
      {
        id: "declined",
        name: "Declined",
        description: "The user declined, so the answers stay empty.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          cancelled: true,
          reviewAnswers: noAnswers(),
        },
        width: 520,
      },
      {
        id: "unavailable",
        name: "Not shown",
        description:
          "The host had no interactive client, so the questionnaire never appeared.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          unavailable: true,
          reviewAnswers: noAnswers(),
        },
        width: 520,
      },
    ],
  },
  {
    id: "model-wait",
    name: "Model wait",
    source: "@/components/tasks/ActivityWait.vue",
    component: ActivityWait,
    scenarios: [
      {
        id: "live",
        name: "Waiting for model",
        description:
          "A request that has not produced its first token yet: the row counts up on its own.",
        props: {
          kind: PendingActivity_Kind.MODEL_REQUEST,
          live: true,
          since: startedAgo(2_400),
          provider: "openrouter",
          model: "anthropic/claude-sonnet-4.5",
        },
        width: 480,
      },
      {
        id: "recorded",
        name: "Recorded request",
        description:
          "The same request after it answered, replayed from the transcript with the span it took.",
        props: {
          kind: ModelWait_Kind.MODEL_REQUEST,
          durationMs: 4_200,
          provider: "openrouter",
          model: "anthropic/claude-sonnet-4.5",
        },
        width: 480,
      },
      {
        id: "retry",
        name: "Retry backoff",
        description:
          "Pi is backing off before it retries; the provider error stays in the tooltip and the attempt comes with it.",
        props: {
          kind: ModelWait_Kind.RETRY,
          durationMs: 2_000,
          model: "opus-4",
          error: "529 overloaded",
          attempt: 1,
        },
        width: 480,
      },
      {
        id: "compaction",
        name: "Compaction",
        description: "The context was compacted before the run could continue.",
        props: {
          kind: ModelWait_Kind.COMPACTION,
          durationMs: 3_400,
          reason: "threshold",
        },
        width: 480,
      },
    ],
  },
  {
    id: "thinking",
    name: "Thinking block",
    source: "@/components/tasks/ThinkingBlock.vue",
    component: ThinkingBlock,
    scenarios: [
      {
        id: "settled",
        name: "Settled",
        description:
          "A finished block: the span, the characters and the rate it wrote them at.",
        props: {
          text: reasoning,
          startedAt: Date.UTC(2026, 10, 9, 14, 5, 0),
          endedAt: Date.UTC(2026, 10, 9, 14, 5, 9),
        },
        width: 480,
      },
      {
        id: "streaming",
        name: "Streaming",
        description:
          "The same block while it streams: it stays open on the newest words, the spinner shows, and the numbers keep moving.",
        props: {
          text: reasoning,
          startedAt: startedAgo(3_000),
          running: true,
        },
        width: 480,
      },
      {
        id: "replayed",
        name: "Replayed from a session",
        description:
          "Pi stores one stamp per message, so a block read back from the session carries the span the timings extension measured.",
        props: {
          text: reasoning,
          startedAt: 1_000,
          endedAt: 1_000,
          durationMs: 9_200,
        },
        width: 480,
      },
      {
        id: "estimated",
        name: "Estimated from the message",
        description:
          "A transcript written before the timing entries names no span of its own: the history parser estimates one from the assistant message, and the tilde mark and tooltip say so.",
        props: {
          text: reasoning,
          startedAt: 1_000,
          endedAt: 1_000,
          durationMs: 9_200,
          estimated: true,
        },
        width: 480,
      },
      {
        id: "unmeasured",
        name: "A block with no measured span",
        description:
          "A block neither measured nor estimable (no message timing either) reports the characters and invents nothing.",
        props: { text: reasoning, startedAt: 1_000, endedAt: 1_000 },
        width: 480,
      },
      {
        id: "short",
        name: "A quick thought",
        description:
          "Under a second, the span keeps its decimals rather than reading “0s”.",
        props: { text: "Hm.", startedAt: 1_000, endedAt: 1_420 },
        width: 480,
      },
    ],
  },
  {
    id: "console-user-message",
    name: "User message",
    source: "@/components/tasks/ConsoleUserMessage.vue",
    component: ConsoleUserMessage,
    scenarios: [
      {
        id: "prompt",
        name: "Prompt",
        description:
          "A prompt as the agent saw it, stamped with the time the envelope carried.",
        props: { item: promptItem },
        events: ["copy", "openAttachment", "openImage", "openDiagram"],
        controls: [
          {
            prop: "copied",
            label: "Copy confirmed",
            type: "boolean",
            initial: false,
          },
        ],
        width: 640,
      },
      {
        id: "attachments",
        name: "With attachments",
        description:
          "A prompt that carried files: each chip opens the attachment preview.",
        props: {
          item: { ...promptItem, attachmentIds: ["8", "9"] },
          attachments: promptAttachments,
        },
        events: ["copy", "openAttachment"],
        width: 640,
      },
      {
        id: "long",
        name: "Long, with Markdown",
        description:
          "A prompt with a list, inline code and a fenced block, so the bubble's wrapping and code styling can be reviewed.",
        props: {
          item: {
            ...promptItem,
            text:
              "Two things before you start:\n\n" +
              "1. Keep `runOver` as it is.\n" +
              "2. Replay the transcript from the session file.\n\n" +
              "```sh\npi --mode rpc --session-id <task-id>\n```",
            at: undefined,
          },
        },
        events: ["copy"],
        width: 640,
      },
    ],
  },
  {
    id: "console-agent-message",
    name: "Agent message",
    source: "@/components/tasks/ConsoleAgentMessage.vue",
    component: ConsoleAgentMessage,
    scenarios: [
      {
        id: "saying",
        name: "Saying",
        description:
          "A finished saying with no run behind it yet, as it streams into a live transcript.",
        props: { item: agentMessageItem, sender: consoleAgent },
        events: ["copy", "openImage", "openDiagram"],
        controls: [
          {
            prop: "copied",
            label: "Copy confirmed",
            type: "boolean",
            initial: false,
          },
        ],
        width: 640,
      },
      {
        id: "usage",
        name: "Turn settled",
        description:
          "The message that finished a run: the footer carries the model, the tokens, the cost and how long the run worked.",
        props: {
          item: agentMessageItem,
          sender: consoleAgent,
          usage: runUsage,
        },
        events: ["copy"],
        width: 640,
      },
      {
        id: "markdown",
        name: "Markdown reply",
        description:
          "A reply with a list, inline code and a fenced block, so the bubble's wrapping and code styling can be reviewed.",
        props: {
          item: {
            ...agentMessageItem,
            text:
              "Two fixes went in:\n\n" +
              "1. The history page is read after the first page settles.\n" +
              "2. `sessionFilePath()` resolves the workspace first.\n\n" +
              "```go\npath, err := filepath.EvalSymlinks(workspace)\n```",
            at: undefined,
          },
          sender: consoleAgent,
        },
        events: ["copy"],
        width: 640,
      },
      {
        id: "unattributed",
        name: "No sender",
        description:
          "A saying from a task whose assignee is unknown: the bot mark and no name line, which is how an old transcript reads.",
        props: { item: agentMessageItem },
        events: ["copy"],
        width: 640,
      },
    ],
  },
  {
    id: "console-activity",
    name: "Activity block",
    source: "@/components/tasks/ConsoleActivity.vue",
    component: ConsoleActivity,
    scenarios: [
      {
        id: "tools",
        name: "Thinking and tool calls",
        description:
          "A settled block: a thinking block, then the bash, read and edit calls it led to, with the changed file and the request that answered it.",
        props: {
          group: activityBlock(finishedTurn),
          run: stillRun,
          sender: consoleAgent,
        },
        events: ["toggle"],
        width: 640,
      },
      {
        id: "streaming",
        name: "Thinking",
        description:
          "The newest block while the model is still writing thinking: the spinner runs, the block counts up, and the thinking header streams its own rate.",
        props: {
          group: activityBlock([streamingThinking], { running: true }),
          run: runningNow(streamingThinking.id),
          now: Date.now(),
          sender: consoleAgent,
        },
        events: ["toggle"],
        width: 640,
      },
      {
        id: "waiting",
        name: "Waiting for the model",
        description:
          "A block that holds nothing but the live request: the wait row is what explains the silence under it.",
        props: {
          group: activityBlock([], {
            running: true,
            pending: {
              kind: PendingActivity_Kind.MODEL_REQUEST,
              since: startedAgo(2_400),
              model: "deepseek-v4.1-flash",
              provider: "refuseless",
              error: "",
              attempt: 0,
              maxAttempts: 0,
              reason: "",
            },
          }),
          run: { ...runningNow(0) },
          sender: consoleAgent,
        },
        events: ["toggle"],
        width: 640,
      },
      {
        id: "interrupted",
        name: "Interrupted call",
        description:
          "A run was stopped between a call and its result: the call freezes, reads as interrupted, and the block stops counting.",
        props: {
          group: activityBlock([bashCall, readCall, interruptedCall]),
          run: stillRun,
          sender: consoleAgent,
        },
        events: ["toggle"],
        width: 640,
      },
    ],
  },
  {
    id: "task-console",
    name: "Task console",
    source: "@/components/tasks/TaskConsole.vue",
    component: TaskConsole,
    scenarios: [
      {
        id: "turn",
        name: "One turn",
        description:
          "A prompt, its thinking and tool call, and the agent's answer — drawn from a fixed transcript, so no agent is involved.",
        props: {
          id: "CHPI-541",
          agent: consoleAgentMessage,
          history: consoleTurnEvents,
          class: "h-[32rem] rounded-lg border",
        },
        events: ["openAttachment", "openImage", "openDiagram", "modelChange"],
        width: 720,
      },
      {
        id: "long",
        name: "Scrolled transcript",
        description:
          "Three turns in a short viewport: the pinned prompt header follows the newest prompt scrolled past, and the scroll controls appear.",
        props: {
          id: "CHPI-541",
          agent: consoleAgentMessage,
          history: consoleLongEvents,
          class: "h-[24rem] rounded-lg border",
        },
        width: 720,
      },
    ],
  },
  {
    id: "task-details",
    name: "Task details",
    source: "@/views/TaskView/TaskDetails.vue",
    component: TaskDetailsStage,
    scenarios: [
      {
        id: "running",
        name: "Running task",
        description:
          "A running task with its tabs live: the assignee is pinned by the run, the usage tiles count, and Plan, Artifacts, Comments, System and Logs each open in place. The data tabs stay empty here — a preview never asks for a task that does not exist.",
        props: { ...detailsProps, task: runningTask },
        events: detailsEvents,
        width: 360,
      },
      {
        id: "settled",
        name: "Settled task",
        description:
          "A finished task with its record filled in: the attachments it carried, the artifacts its run published, the comment thread, the tools its runs declared and the calls and log entries they left behind.",
        props: { ...detailsProps, task: settledTask, preview: settledPreview },
        events: detailsEvents,
        width: 360,
      },
      {
        id: "backlog",
        name: "Backlog task",
        description:
          "An unstarted backlog task keeps its project and every field editable, so the pickers are live instead of reading a pin.",
        props: { ...detailsProps, task: backlogTask, taskUsage: undefined },
        events: detailsEvents,
        width: 360,
      },
    ],
  },
  {
    id: "task-composer",
    name: "Task composer",
    source: "@/components/tasks/ExpressTaskForm.vue",
    component: ExpressTaskForm,
    scenarios: [
      {
        id: "page",
        name: "On a page",
        description:
          "The composer as the tasks page renders it, filled in from fixture projects, agents and tags: nothing is created and nothing is fetched.",
        props: { preview: composerPreview },
        events: ["created"],
        controls: [
          {
            prop: "defaultProjectId",
            label: "Default project id",
            type: "text",
            initial: "p-1",
          },
        ],
        width: 640,
      },
      {
        id: "dialog",
        name: "Inside a dialog",
        description:
          "The same composer with `bare` set: the card surface is dropped, because the dialog that frames it already supplies one.",
        props: { preview: composerPreview, bare: true },
        events: ["created"],
        width: 640,
      },
    ],
  },
  {
    id: "task-picker",
    name: "Inline task picker",
    source: "@/components/tasks/TaskPicker.vue",
    component: TaskPicker,
    scenarios: [
      {
        id: "browse",
        name: "Browsing",
        description:
          "The picker with nothing typed: the recent candidates, their state pills and the filter row above them.",
        props: { preview: pickerPreview },
        events: ["pick"],
        controls: [
          { prop: "search", label: "Query", type: "text", initial: "" },
        ],
        width: 480,
      },
      {
        id: "embedded",
        name: "Embedded in a menu",
        description:
          "The same list inside the parent selector's menu: the popover chrome is dropped so it does not nest a second bordered surface.",
        props: {
          preview: pickerPreview,
          embedded: true,
          includeArchived: false,
        },
        events: ["pick"],
        controls: [
          { prop: "search", label: "Query", type: "text", initial: "" },
        ],
        width: 420,
      },
    ],
  },
  {
    id: "reference-picker",
    name: "Inline reference picker",
    source: "@/components/tasks/ReferencePicker.vue",
    component: ReferencePicker,
    scenarios: [
      {
        id: "browse",
        name: "Browsing",
        description:
          "The picker with nothing typed: every actor, then the project and team groups. Typing narrows all three lists locally.",
        props: { preview: referenceDirectoryPreview },
        events: ["pick"],
        controls: [
          { prop: "search", label: "Query", type: "text", initial: "" },
        ],
        width: 440,
      },
    ],
  },
  {
    id: "actor-combobox",
    name: "Inline user picker",
    source: "@/components/ActorCombobox.vue",
    component: ActorCombobox,
    scenarios: [
      {
        id: "all",
        name: "People and agents",
        description:
          "Every actor: people first, then the agents, each naming what it is. Typing filters the list, and the artifact shows the value that is picked.",
        props: { preview: actorPreview, placeholder: "Search actors…" },
        events: ["update:modelValue"],
        width: 420,
      },
      {
        id: "users",
        name: "People only",
        description:
          "An owner or a lead must be a person, so the picker offers users alone.",
        props: {
          preview: actorPreview,
          kinds: "users",
          placeholder: "Search people…",
        },
        events: ["update:modelValue"],
        width: 420,
      },
    ],
  },
  {
    id: "follow-up-composer",
    name: "Follow-up prompt",
    source: "@/components/tasks/TaskFollowUpComposer.vue",
    component: TaskFollowUpComposerStage,
    scenarios: [
      {
        id: "steering",
        name: "Steering a run",
        description:
          "A run is in flight, so this prompt is injected between its blocks: the footer offers Stop, the button says Steer, and the context gauge tracks the session.",
        props: {
          ...followUpProps,
          steer: true,
          awaitingAnswers: false,
        },
        events: ["send", "enter", "stop", "modelChange"],
        controls: [
          { prop: "prompt", label: "Draft", type: "text", initial: "" },
        ],
        width: 720,
      },
      {
        id: "queued",
        name: "Next run",
        description:
          "Nothing is running, so Send starts the next run with files already staged beside the prompt.",
        props: {
          ...followUpProps,
          steer: false,
          awaitingAnswers: false,
          prompt: "Also cover the restart path with a test.\n",
          stagedAttachments: promptAttachments,
        },
        events: ["send", "enter", "removeStaged", "openAttachment"],
        width: 720,
      },
      {
        id: "awaiting",
        name: "Waiting on answers",
        description:
          "The agent is blocked on a questionnaire: the draft is kept and the hint says sending unlocks once the answers are in.",
        props: {
          ...followUpProps,
          steer: false,
          awaitingAnswers: true,
          canFollowUp: false,
          prompt: "While you wait: what did the restart leave behind?\n",
        },
        events: ["send", "enter"],
        width: 720,
      },
      {
        id: "cleaned-up",
        name: "Session cleaned up",
        description:
          "The agent-side Pi session is gone, so sending replays the stored conversation in a new one — and the model controls step aside.",
        props: {
          ...followUpProps,
          steer: false,
          awaitingAnswers: false,
          task: cleanedUpTask,
        },
        events: ["send", "enter"],
        width: 720,
      },
    ],
  },
  {
    id: "actor-tooltip",
    name: "Actor tooltip",
    source: "@/components/ActorTooltip.vue",
    component: ActorTooltip,
    scenarios: [
      {
        id: "user",
        name: "User",
        description:
          "A person: the picture (or initials), the display name, the handle and the kind. Hovering the avatar opens the card; the avatar links to the actor's page.",
        props: {
          actorId: "4",
          name: "Vlad Faust",
          username: "vladfaust",
          kind: "user",
          to: { name: "user", params: { username: "vladfaust" } },
          size: 24,
        },
        controls: [
          { prop: "name", label: "Name", type: "text", initial: "Vlad Faust" },
          {
            prop: "username",
            label: "Handle",
            type: "text",
            initial: "vladfaust",
          },
        ],
      },
      {
        id: "agent",
        name: "Agent",
        description:
          "An agent actor: no uploaded picture, so the generated mark stands in. The kind chip is what tells the two apart.",
        props: {
          actorId: "100",
          name: "Reviewer",
          username: "reviewer",
          kind: "agent",
          to: { name: "agent", params: { username: "reviewer" } },
          size: 24,
        },
        controls: [
          { prop: "name", label: "Name", type: "text", initial: "Reviewer" },
          {
            prop: "username",
            label: "Handle",
            type: "text",
            initial: "reviewer",
          },
        ],
      },
      {
        id: "unknown",
        name: "Unknown actor",
        description:
          "An actor id the directories could not resolve: the tip falls back to the bare id and the avatar is not a link, because there is no page to open.",
        props: { actorId: "999", size: 24 },
        width: 360,
      },
    ],
  },
  {
    id: "task-tooltip",
    name: "Task tooltip",
    source: "@/components/TaskTooltip.vue",
    component: TaskTooltip,
    scenarios: [
      {
        id: "blocking",
        name: "Blocking reference",
        description:
          "A blocking dependency with everything the list read loads: id and state on the chip, the trimmed title, priority, size and the target's tags. The lock marks the edge as blocking.",
        props: {
          id: "CHPI-542",
          name: "Ship the rollout notes",
          prompt: "Ship the rollout notes before the release cut.",
          state: TaskState.QUEUED,
          priority: TaskPriority.HIGH,
          size: TaskSize.MEDIUM,
          tags: studioTags,
          kind: TaskReferenceKind.BLOCKING,
        },
        controls: [
          {
            prop: "name",
            label: "Title",
            type: "text",
            initial: "Ship the rollout notes",
          },
        ],
      },
      {
        id: "informational",
        name: "Informational edge",
        description:
          "A non-blocking edge: no lock, and the tip names the kind so a reader knows the task does not gate anything.",
        props: {
          id: "CHPI-603",
          name: "Publish the rollout notes",
          state: TaskState.SETTLED,
          priority: TaskPriority.NORMAL,
          size: TaskSize.SMALL,
          kind: TaskReferenceKind.INFORMATIONAL,
        },
        width: 360,
      },
      {
        id: "long-title",
        name: "Long title",
        description:
          "A long name wraps instead of clamping: the card is what makes it readable, so the tip never falls back to the browser's native title.",
        props: {
          id: "CHPI-604",
          name: "Rework the console's transcript replay so a restart stops duplicating the first page of every settled run",
          state: TaskState.RUNNING,
          priority: TaskPriority.CRITICAL,
          size: TaskSize.EPIC,
          tags: studioTags,
        },
        width: 360,
      },
      {
        id: "bare",
        name: "Bare reference",
        description:
          "Only the id and state, which is all an older edge read carries: the optional rows drop out instead of rendering empty.",
        props: { id: "CHPI-410", state: TaskState.BACKLOG },
        width: 360,
      },
    ],
  },
  {
    id: "avatar-tip",
    name: "Avatar tip",
    source: "@/components/AvatarTip.vue",
    component: AvatarTip,
    scenarios: [
      {
        id: "project",
        name: "Project",
        description:
          "A project's generated mark with its name and path in the tip: the generic entity avatar, for what is neither an actor nor a task.",
        props: {
          name: "Console",
          meta: "/console",
          seed: "p-1",
          size: 24,
          to: { name: "project", params: { slug: "console" } },
        },
        controls: [
          { prop: "name", label: "Name", type: "text", initial: "Console" },
          {
            prop: "meta",
            label: "Tip line",
            type: "text",
            initial: "/console",
          },
        ],
      },
      {
        id: "team",
        name: "Team",
        description:
          "A team: the initials stand in for a picture, and the tip says what the entity is.",
        props: {
          name: "Platform",
          meta: "Team",
          size: 24,
          to: { name: "team", params: { id: "1" } },
        },
        width: 360,
      },
    ],
  },
];
