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
  TaskReferenceKind,
  TaskSize,
  TaskState,
  type Task,
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
  LockIcon,
  MessageSquareIcon,
  PencilIcon,
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
  [TaskState.DRAFT]: PencilIcon,
  [TaskState.BACKLOG]: InboxIcon,
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
    case TaskState.DRAFT:
      return "Draft";
    case TaskState.BACKLOG:
      return "Backlog";
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
  task: Pick<Task, "state" | "blockers" | "references" | "waitingReason">,
): string {
  if (task.state !== TaskState.QUEUED) return "";
  if (task.waitingReason) return task.waitingReason;

  const blockers = taskIncompleteBlockers(task);
  if (blockers.length) {
    return `Waiting for tasks to become completed: ${blockers
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
    case TaskState.DRAFT:
      return "border-zinc-600/50 bg-zinc-500/10 text-zinc-600 dark:text-zinc-300";
    case TaskState.BACKLOG:
      return "border-indigo-500/35 bg-indigo-500/12 text-indigo-600 dark:text-indigo-400";
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
    case TaskState.DRAFT:
      return "text-zinc-600 dark:text-zinc-300";
    case TaskState.BACKLOG:
      return "text-indigo-600 dark:text-indigo-400";
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
}[] = [
  { value: TaskReferenceKind.INFORMATIONAL, label: "Informational" },
  { value: TaskReferenceKind.BLOCKING, label: "Blocking" },
  { value: TaskReferenceKind.SUPERSEDES, label: "Supersedes" },
];

export function taskReferenceKindLabel(
  kind: TaskReferenceKind | undefined,
): string {
  switch (kind) {
    case TaskReferenceKind.BLOCKING:
      return "Blocking";
    case TaskReferenceKind.SUPERSEDES:
      return "Supersedes";
    default:
      return "Informational";
  }
}

export function taskReferenceKindIcon(
  kind: TaskReferenceKind | undefined,
): LucideIcon {
  switch (kind) {
    case TaskReferenceKind.BLOCKING:
      return LockIcon;
    case TaskReferenceKind.SUPERSEDES:
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
      return "text-amber-600 dark:text-amber-400";
    case TaskReferenceKind.SUPERSEDES:
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
      return "border-amber-500/40 bg-amber-500/5";
    case TaskReferenceKind.SUPERSEDES:
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
  { key: "draft", label: "Draft", state: TaskState.DRAFT },
  { key: "backlog", label: "Backlog", state: TaskState.BACKLOG },
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
  [TaskState.QUEUED]: 4,
  [TaskState.BACKLOG]: 5,
  [TaskState.DRAFT]: 6,
  [TaskState.COMPLETED]: 7,
  [TaskState.CANCELLED]: 8,
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
