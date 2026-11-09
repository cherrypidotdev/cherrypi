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
  TaskPriority,
  TaskProgressItemStatus,
  TaskReferenceKind,
  TaskSize,
  TaskState,
  type Task,
  type TaskProgress,
  type TaskReference,
} from "@cherrypi/sdk";
import {
  ArrowRightLeftIcon,
  BanIcon,
  CheckCheckIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  InboxIcon,
  InfoIcon,
  LinkIcon,
  ListTreeIcon,
  LockIcon,
  MessageSquareIcon,
  PauseIcon,
  PlayIcon,
  TimerIcon,
  type LucideIcon,
} from "@lucide/vue";

export interface TaskModelSelection {
  modelProvider: string;
  modelId: string;
  modelName?: string;
  thinkingLevel: string;
}

export const EMPTY_TASK_MODEL_SELECTION: TaskModelSelection = {
  modelProvider: "",
  modelId: "",
  thinkingLevel: "",
};

const TASK_STATE_ICONS: Record<string, LucideIcon> = {
  [TaskState.BACKLOG]: InboxIcon,
  [TaskState.WAITING]: PauseIcon,
  [TaskState.QUEUED]: TimerIcon,
  [TaskState.RUNNING]: PlayIcon,
  [TaskState.INPUT]: MessageSquareIcon,
  [TaskState.SETTLED]: CheckCheckIcon,
  [TaskState.FAILED]: CircleAlertIcon,
  [TaskState.COMPLETED]: CircleCheckIcon,
  [TaskState.CANCELLED]: BanIcon,
};

export function taskStateIcon(state: TaskState | undefined): LucideIcon {
  return (state ? TASK_STATE_ICONS[state] : undefined) ?? InfoIcon;
}

/**
 * A task's display name. Falls back to the first line of its prompt
 * (truncated) when the name is empty, e.g. for rows predating naming or while
 * an auto-title is still being generated.
 */
export function taskName(task: Pick<Task, "id" | "name" | "prompt">): string {
  const name = task.name?.trim();
  if (name) return name;

  const prompt = (task.prompt.trim().split(/\r?\n/, 1)[0] ?? "").trim();
  if (prompt) return prompt.length > 80 ? prompt.slice(0, 80) : prompt;

  return shortTaskId(task.id);
}

/**
 * Whether a task's displayed title is temporary: its name is not hard-set, so
 * {@link taskName} reuses the first line of the prompt. {@link taskNameClass}
 * renders such a title italic and dimmed.
 */
export function taskNameIsTemporary(
  task: Pick<Task, "name" | "prompt">,
): boolean {
  return !task.name?.trim() && !!task.prompt.trim();
}

export function taskNameClass(task: Pick<Task, "name" | "prompt">): string {
  if (taskNameIsTemporary(task)) return "italic opacity-80";

  return task.name?.trim() ? "font-semibold" : "";
}

export function taskTitlePending(
  task: Pick<Task, "name" | "prompt" | "autoTitle" | "state">,
): boolean {
  return (
    task.autoTitle &&
    task.state === TaskState.RUNNING &&
    !task.name?.trim() &&
    !!task.prompt.trim()
  );
}

export function shortTaskId(id: string): string {
  return id;
}

/**
 * A reference's display title: its name, else the first line of its prompt,
 * else its id. Pairs with the reference's bold id in a badge, e.g.
 * `AURORA-1: Set up the schema`.
 */
export function taskReferenceName(
  ref: Pick<TaskReference, "id" | "name" | "prompt">,
): string {
  return taskName(ref);
}

export function taskReferenceTitle(
  ref: Pick<TaskReference, "id" | "name" | "prompt">,
): string {
  const title = taskName(ref);
  return title !== ref.id ? `${ref.id} · ${title}` : ref.id;
}

export function taskStateLabel(state: TaskState): string {
  switch (state) {
    case TaskState.BACKLOG:
      return "Backlog";
    case TaskState.WAITING:
      return "Waiting";
    case TaskState.QUEUED:
      return "Queued";
    case TaskState.RUNNING:
      return "Running";
    case TaskState.INPUT:
      return "Input";
    case TaskState.SETTLED:
      return "Settled";
    case TaskState.FAILED:
      return "Failed";
    case TaskState.COMPLETED:
      return "Completed";
    case TaskState.CANCELLED:
      return "Cancelled";
    default:
      return "Unknown";
  }
}

export const TASK_BLOCKERS_REASON_PREFIX =
  "Waiting for tasks to become completed:";

export type WaitingReasonPart =
  | { kind: "text"; text: string }
  | { kind: "id"; id: string };

export function waitingReasonParts(reason: string): WaitingReasonPart[] {
  const parts: WaitingReasonPart[] = [];

  function text(value: string) {
    const last = parts.at(-1);
    if (last?.kind === "text") last.text += value;
    else parts.push({ kind: "text", text: value });
  }

  reason.split("; ").forEach((segment, index) => {
    if (index > 0) text("; ");
    const ids = segment
      .slice(TASK_BLOCKERS_REASON_PREFIX.length)
      .split(",")
      .map((id) => id.trim())
      .filter(Boolean);
    if (!segment.startsWith(TASK_BLOCKERS_REASON_PREFIX) || !ids.length) {
      text(segment);
      return;
    }
    text(`${TASK_BLOCKERS_REASON_PREFIX} `);
    ids.forEach((id, idIndex) => {
      if (idIndex > 0) text(", ");
      parts.push({ kind: "id", id });
    });
  });
  return parts;
}

export function taskIncompleteBlockers(
  task: Pick<Task, "state" | "references">,
): TaskReference[] {
  if (task.state !== TaskState.QUEUED) return [];
  return (task.references ?? [])
    .filter(
      (ref) =>
        ref.kind === TaskReferenceKind.BLOCKING &&
        ref.state !== TaskState.COMPLETED,
    )
    .sort((a, b) => compareTaskIDs(a.id, b.id));
}

function compareTaskIDs(a: string, b: string): number {
  const ai = a.lastIndexOf("-");
  const bi = b.lastIndexOf("-");
  const an = Number(a.slice(ai + 1));
  const bn = Number(b.slice(bi + 1));
  if (ai > 0 && bi > 0 && Number.isInteger(an) && Number.isInteger(bn)) {
    const ap = a.slice(0, ai);
    const bp = b.slice(0, bi);
    return ap === bp ? an - bn : ap.localeCompare(bp);
  }
  return a.localeCompare(b);
}

export function taskWaitingReason(
  task: Pick<
    Task,
    "state" | "blockers" | "references" | "waitingReason" | "holds"
  >,
): string {
  if (task.state !== TaskState.QUEUED && task.state !== TaskState.WAITING) {
    return "";
  }
  if (task.waitingReason) return task.waitingReason;

  const holds = (task.holds ?? []).map((hold) => {
    const name = hold.pluginName || hold.plugin;
    return hold.reason
      ? `Waiting for ${name}: ${hold.reason}`
      : `Waiting for ${name}`;
  });
  if (holds.length) return holds.join("; ");

  const blockers = taskIncompleteBlockers(task);
  if (blockers.length) {
    return `${TASK_BLOCKERS_REASON_PREFIX} ${blockers
      .map((ref) => ref.id)
      .join(", ")}`;
  }
  if ((task.blockers ?? 0) > 0) {
    return "Waiting for blocking tasks to become completed";
  }
  return "Waiting for free agent slot";
}

export function taskStateClass(state: TaskState): string {
  switch (state) {
    case TaskState.BACKLOG:
      return "border-indigo-500/35 bg-indigo-500/12 text-indigo-600 dark:text-indigo-400";
    case TaskState.WAITING:
      return "border-gray-500/40 bg-gray-500/10 text-gray-500 dark:text-gray-400";
    case TaskState.QUEUED:
      return "border-amber-500/35 bg-amber-500/12 text-amber-600 dark:text-amber-400";
    case TaskState.RUNNING:
      return "border-blue-500/35 bg-blue-500/12 text-blue-600 dark:text-blue-400";
    case TaskState.INPUT:
      return "border-cyan-400/40 bg-cyan-400/12 text-cyan-500 dark:text-cyan-300";
    case TaskState.SETTLED:
      return "border-emerald-500/32 bg-emerald-500/12 text-emerald-600 dark:text-emerald-400";
    case TaskState.FAILED:
      return "border-red-500/32 bg-red-500/12 text-red-600 dark:text-red-400";
    case TaskState.COMPLETED:
      return "border-teal-500/32 bg-teal-500/12 text-teal-600 dark:text-teal-400";
    case TaskState.CANCELLED:
      return "border-gray-500/40 bg-gray-500/10 text-gray-500 dark:text-gray-400";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}

export function taskStateTextClass(state: TaskState): string {
  switch (state) {
    case TaskState.BACKLOG:
      return "text-indigo-600 dark:text-indigo-400";
    case TaskState.WAITING:
      return "text-gray-500 dark:text-gray-400";
    case TaskState.QUEUED:
      return "text-amber-600 dark:text-amber-400";
    case TaskState.RUNNING:
      return "text-blue-600 dark:text-blue-400";
    case TaskState.INPUT:
      return "text-cyan-500 dark:text-cyan-300";
    case TaskState.SETTLED:
      return "text-emerald-600 dark:text-emerald-400";
    case TaskState.FAILED:
      return "text-red-600 dark:text-red-400";
    case TaskState.COMPLETED:
      return "text-teal-600 dark:text-teal-400";
    case TaskState.CANCELLED:
      return "text-gray-500 dark:text-gray-400";
    default:
      return "text-muted-foreground";
  }
}

export function taskStateLabelFor(task: Pick<Task, "state">): string {
  return taskStateLabel(task.state);
}

export const TASK_REFERENCE_KINDS: {
  value: TaskReferenceKind;
  label: string;
  description: string;
}[] = [
  {
    value: TaskReferenceKind.INFORMATIONAL,
    label: "Informational",
    description: "A plain relation that does not gate queuing.",
  },
  {
    value: TaskReferenceKind.BLOCKED_BY,
    label: "Blocked by",
    description: "This task waits for the other task to complete.",
  },
  {
    value: TaskReferenceKind.BLOCKS,
    label: "Blocks",
    description: "The other task waits for this task to complete.",
  },
  {
    value: TaskReferenceKind.SUPERSEDES,
    label: "Supersedes",
    description: "This task replaces the other task and cancels it.",
  },
  {
    value: TaskReferenceKind.SUPERSEDED_BY,
    label: "Superseded by",
    description: "The other task replaces this task and cancels it.",
  },
];

export type NewTaskRelation =
  | "informational"
  | "blocked_by"
  | "blocks"
  | "supersedes"
  | "parent";

export const NEW_TASK_RELATIONS: {
  value: NewTaskRelation;
  label: string;
  description: string;
}[] = [
  {
    value: "informational",
    label: "Informational",
    description: "The new task links to this one.",
  },
  {
    value: "blocked_by",
    label: "Blocked by this task",
    description: "The new task waits for this one to be completed.",
  },
  {
    value: "blocks",
    label: "Blocks this task",
    description: "This task waits for the new one to be completed.",
  },
  {
    value: "supersedes",
    label: "Supersedes this task",
    description: "The new task replaces this one and cancels it.",
  },
  {
    value: "parent",
    label: "Sub-task",
    description: "The new task nests under this one.",
  },
];

export function newTaskReferenceKind(
  relation: NewTaskRelation | null | undefined,
): TaskReferenceKind | null {
  switch (relation) {
    case "informational":
      return TaskReferenceKind.INFORMATIONAL;
    case "blocked_by":
      return TaskReferenceKind.BLOCKED_BY;
    case "blocks":
      return TaskReferenceKind.BLOCKS;
    case "supersedes":
      return TaskReferenceKind.SUPERSEDES;
    default:
      return null;
  }
}

export function newTaskRelationIcon(
  relation: NewTaskRelation | undefined,
): LucideIcon {
  if (relation === "parent") return ListTreeIcon;
  return taskReferenceKindIcon(newTaskReferenceKind(relation) ?? undefined);
}

export function newTaskRelationNote(
  relation: NewTaskRelation | null | undefined,
  taskID: string,
): string {
  switch (relation) {
    case "informational":
      return `The new task links to ${taskID}.`;
    case "blocked_by":
      return `The new task cannot be queued until ${taskID} is completed.`;
    case "blocks":
      return `${taskID} cannot be queued until the new task is completed.`;
    case "supersedes":
      return `The new task replaces ${taskID}, which is cancelled when it is in backlog, queued, or failed.`;
    case "parent":
      return `${taskID} becomes the parent of the new task.`;
    default:
      return "";
  }
}

export function taskReferencePerspectiveKind(
  kind: TaskReferenceKind | undefined,
  incoming: boolean,
): TaskReferenceKind {
  if (kind === TaskReferenceKind.BLOCKING) {
    return incoming ? TaskReferenceKind.BLOCKS : TaskReferenceKind.BLOCKED_BY;
  }
  if (kind === TaskReferenceKind.SUPERSEDES) {
    return incoming
      ? TaskReferenceKind.SUPERSEDED_BY
      : TaskReferenceKind.SUPERSEDES;
  }
  return kind ?? TaskReferenceKind.INFORMATIONAL;
}

const STORED_REFERENCE_KIND_LABELS: Partial<Record<TaskReferenceKind, string>> =
  {
    [TaskReferenceKind.BLOCKING]: "Blocked by",
    [TaskReferenceKind.SUPERSEDES]: "Supersedes",
  };

export function taskReferenceKindLabel(
  kind: TaskReferenceKind | undefined,
): string {
  return (
    TASK_REFERENCE_KINDS.find((option) => option.value === kind)?.label ??
    (kind ? STORED_REFERENCE_KIND_LABELS[kind] : undefined) ??
    "Informational"
  );
}

export function taskReferenceKindFromLog(
  kind: string | undefined,
): TaskReferenceKind {
  switch (kind) {
    case "blocking":
    case "blocked_by":
      return TaskReferenceKind.BLOCKED_BY;
    case "blocks":
      return TaskReferenceKind.BLOCKS;
    case "supersedes":
      return TaskReferenceKind.SUPERSEDES;
    case "superseded_by":
      return TaskReferenceKind.SUPERSEDED_BY;
    default:
      return TaskReferenceKind.INFORMATIONAL;
  }
}

export function taskReferenceKindIcon(
  kind: TaskReferenceKind | undefined,
): LucideIcon {
  switch (kind) {
    case TaskReferenceKind.BLOCKING:
    case TaskReferenceKind.BLOCKED_BY:
    case TaskReferenceKind.BLOCKS:
      return LockIcon;
    case TaskReferenceKind.SUPERSEDES:
    case TaskReferenceKind.SUPERSEDED_BY:
      return ArrowRightLeftIcon;
    default:
      return LinkIcon;
  }
}

export function taskReferenceKindTextClass(
  kind: TaskReferenceKind | undefined,
): string {
  switch (kind) {
    case TaskReferenceKind.BLOCKING:
    case TaskReferenceKind.BLOCKED_BY:
    case TaskReferenceKind.BLOCKS:
      return "text-amber-600 dark:text-amber-400";
    case TaskReferenceKind.SUPERSEDES:
    case TaskReferenceKind.SUPERSEDED_BY:
      return "text-rose-600 dark:text-rose-400";
    default:
      return "text-muted-foreground";
  }
}

export function taskReferenceKindClass(
  kind: TaskReferenceKind | undefined,
): string {
  switch (kind) {
    case TaskReferenceKind.BLOCKING:
    case TaskReferenceKind.BLOCKED_BY:
    case TaskReferenceKind.BLOCKS:
      return "border-amber-500/40 bg-amber-500/5";
    case TaskReferenceKind.SUPERSEDES:
    case TaskReferenceKind.SUPERSEDED_BY:
      return "border-rose-500/40 bg-rose-500/5";
    default:
      return "";
  }
}

export function taskStateClassFor(task: Pick<Task, "state">): string {
  return taskStateClass(task.state);
}

function taskPriorityName(priority: TaskPriority): string {
  switch (priority) {
    case TaskPriority.LOW:
      return "Low";
    case TaskPriority.HIGH:
      return "High";
    case TaskPriority.CRITICAL:
      return "Critical";
    default:
      return "Normal";
  }
}

export function taskPriorityLabel(
  priority: TaskPriority,
  withDimension = false,
): string {
  const name = taskPriorityName(priority);
  return withDimension ? `${name} priority` : name;
}

export function taskPriorityTextClass(priority: TaskPriority): string {
  switch (priority) {
    case TaskPriority.LOW:
      return "text-green-600 dark:text-green-400";
    case TaskPriority.HIGH:
      return "text-orange-600 dark:text-orange-400";
    case TaskPriority.CRITICAL:
      return "text-red-600 dark:text-red-400";
    default:
      return "text-yellow-600 dark:text-yellow-400";
  }
}

export function taskPriorityClass(priority: TaskPriority): string {
  switch (priority) {
    case TaskPriority.LOW:
      return "border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400";
    case TaskPriority.HIGH:
      return "border-orange-500/30 bg-orange-500/10 text-orange-600 dark:text-orange-400";
    case TaskPriority.CRITICAL:
      return "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400";
    default:
      return "border-yellow-500/30 bg-yellow-500/10 text-yellow-600 dark:text-yellow-400";
  }
}

export function taskBadgeToneClass(tone: string): string {
  switch (tone) {
    case "success":
      return "border-emerald-500/32 bg-emerald-500/12 text-emerald-600 dark:text-emerald-400";
    case "info":
      return "border-blue-500/35 bg-blue-500/12 text-blue-600 dark:text-blue-400";
    case "warning":
      return "border-amber-500/35 bg-amber-500/12 text-amber-600 dark:text-amber-400";
    case "danger":
      return "border-red-500/32 bg-red-500/12 text-red-600 dark:text-red-400";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}

function taskSizeName(size: TaskSize): string {
  switch (size) {
    case TaskSize.SMALL:
      return "Small";
    case TaskSize.BIG:
      return "Big";
    case TaskSize.EPIC:
      return "Epic";
    default:
      return "Medium";
  }
}

export function taskSizeLabel(size: TaskSize, withDimension = false): string {
  const name = taskSizeName(size);
  return withDimension ? `${name} size` : name;
}

export function taskSizeTextClass(size: TaskSize): string {
  switch (size) {
    case TaskSize.SMALL:
      return "text-green-600 dark:text-green-400";
    case TaskSize.BIG:
      return "text-orange-600 dark:text-orange-400";
    case TaskSize.EPIC:
      return "text-red-600 dark:text-red-400";
    default:
      return "text-yellow-600 dark:text-yellow-400";
  }
}

export function taskSizeClass(size: TaskSize): string {
  switch (size) {
    case TaskSize.SMALL:
      return "border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400";
    case TaskSize.BIG:
      return "border-orange-500/30 bg-orange-500/10 text-orange-600 dark:text-orange-400";
    case TaskSize.EPIC:
      return "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400";
    default:
      return "border-yellow-500/30 bg-yellow-500/10 text-yellow-600 dark:text-yellow-400";
  }
}

export const TASK_SIZES: { value: TaskSize; label: string }[] = [
  { value: TaskSize.SMALL, label: "Small" },
  { value: TaskSize.MEDIUM, label: "Medium" },
  { value: TaskSize.BIG, label: "Big" },
  { value: TaskSize.EPIC, label: "Epic" },
];

export const TASK_SIZE_RANK: Record<number, number> = {
  [TaskSize.SMALL]: 0,
  [TaskSize.MEDIUM]: 1,
  [TaskSize.BIG]: 2,
  [TaskSize.EPIC]: 3,
};

export function defaultCloseOutOutcome(state: TaskState): TaskState {
  return state === TaskState.FAILED ? TaskState.CANCELLED : TaskState.COMPLETED;
}

export const TASK_STATE_TABS: {
  key: string;
  label: string;
  state?: TaskState;
}[] = [
  { key: "all", label: "All" },
  { key: "backlog", label: "Backlog", state: TaskState.BACKLOG },
  { key: "waiting", label: "Waiting", state: TaskState.WAITING },
  { key: "queued", label: "Queued", state: TaskState.QUEUED },
  { key: "failed", label: "Failed", state: TaskState.FAILED },
  { key: "running", label: "Running", state: TaskState.RUNNING },
  { key: "input", label: "Input", state: TaskState.INPUT },
  { key: "settled", label: "Settled", state: TaskState.SETTLED },
  { key: "completed", label: "Completed", state: TaskState.COMPLETED },
  { key: "cancelled", label: "Cancelled", state: TaskState.CANCELLED },
];

export const TASK_STATE_RANK: Record<number, number> = {
  [TaskState.FAILED]: 0,
  [TaskState.INPUT]: 1,
  [TaskState.SETTLED]: 2,
  [TaskState.RUNNING]: 3,
  [TaskState.WAITING]: 4,
  [TaskState.QUEUED]: 5,
  [TaskState.BACKLOG]: 6,
  [TaskState.COMPLETED]: 8,
  [TaskState.CANCELLED]: 9,
};

export function tagColor(color?: string): string {
  const value = (color ?? "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value.toLowerCase() : "#a1a1aa";
}

export function tagStyle(color?: string): Record<string, string> {
  const value = tagColor(color);
  return {
    color: value,
    borderColor: `${value}66`,
    backgroundColor: `${value}1a`,
  };
}

export function taskProgressHasPlan(
  progress: TaskProgress | undefined,
): boolean {
  return Boolean(progress && progress.total > 0);
}

export function taskPlanExpected(size: TaskSize | undefined): boolean {
  return (size ?? TaskSize.UNSPECIFIED) !== TaskSize.SMALL;
}

export function taskPlanApplies(state: TaskState | undefined): boolean {
  return (
    state === TaskState.RUNNING ||
    state === TaskState.INPUT ||
    state === TaskState.SETTLED ||
    state === TaskState.FAILED
  );
}

export type TaskPlanFlag = "reported" | "missing" | "hint" | "idle";

export function taskPlanFlag(
  progress: TaskProgress | undefined,
  size: TaskSize | undefined,
  state: TaskState | undefined,
): TaskPlanFlag {
  if (taskProgressHasPlan(progress)) return "reported";
  if (!taskPlanApplies(state)) return "idle";
  return taskPlanExpected(size) ? "missing" : "hint";
}

export function taskProgressPercent(
  progress: TaskProgress | undefined,
): number {
  if (!progress || progress.total <= 0) return 0;
  return Math.round((progress.done / progress.total) * 100);
}

export function taskProgressCount(progress: TaskProgress | undefined): string {
  if (!progress || progress.total <= 0) return "—";
  return `${progress.done}/${progress.total}`;
}

export function taskProgressInProgress(
  progress: TaskProgress | undefined,
): boolean {
  return Boolean(
    progress?.items.some(
      (item) => item.status === TaskProgressItemStatus.IN_PROGRESS,
    ),
  );
}

export function taskProgressCurrent(
  progress: TaskProgress | undefined,
): string {
  if (!progress) return "";
  const active = progress.items.find(
    (item) => item.status === TaskProgressItemStatus.IN_PROGRESS,
  );
  if (active) return active.text;
  if (progress.summary) return progress.summary;
  if (progress.total > 0 && progress.done === progress.total) {
    return "All steps complete";
  }
  return (
    progress.items.find(
      (item) => item.status === TaskProgressItemStatus.PENDING,
    )?.text ?? ""
  );
}
