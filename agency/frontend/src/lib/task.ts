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

/** A per-task model/thinking override; empty fields keep the agent default. */
export interface TaskModelSelection {
  modelProvider: string;
  modelId: string;
  modelName?: string;
  thinkingLevel: string;
}

/** The unset selection: the scheduler applies the agent's defaults. */
export const EMPTY_TASK_MODEL_SELECTION: TaskModelSelection = {
  modelProvider: "",
  modelId: "",
  thinkingLevel: "",
};

/**
 * The icon cue for each state. Icons are a second, colorblind-safe channel:
the tabs lead with them and the table badges use them in place of the bullet.
 */
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

/** The icon component for a task state; `InfoIcon` for an unknown one. */
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

  // A legacy row the agency has not enriched yet: fall back to its id rather
  // than rendering nothing.
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

/**
 * Classes for a task's title span. A temporary (prompt-derived) title is
 * italic and dimmed to read as a placeholder; a hard-set title is bold, like
 * the task header.
 */
export function taskNameClass(task: Pick<Task, "name" | "prompt">): string {
  if (taskNameIsTemporary(task)) return "italic opacity-80";
  // A legacy row with neither a name nor a prompt shows its id; leave it
  // unstyled rather than passing it off as a hard-set title.
  return task.name?.trim() ? "font-semibold" : "";
}

/**
 * Whether a task's title is still being auto-generated: it opted into
 * auto-titling, its run is active, and its name has not arrived yet. A draft
 * or queued task has not started titling, so it falls back to its prompt.
 */
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

/** Task id used in lists and links (`MYPROJECT-1`), rendered as-is. */
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

/** Full `id · title` tooltip for a reference badge. */
export function taskReferenceTitle(
  ref: Pick<TaskReference, "id" | "name" | "prompt">,
): string {
  const title = taskName(ref);
  return title !== ref.id ? `${ref.id} · ${title}` : ref.id;
}

/** Human-readable label for a task state. */
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

/**
 * Whether a queued task is blocked by an incomplete blocking reference. The
 * blocked flag is computed, not a state: the task stays queued and the
 * scheduler skips it until its blockers complete.
 */
export function isTaskBlocked(task: Pick<Task, "state" | "blockers">): boolean {
  return task.state === TaskState.QUEUED && (task.blockers ?? 0) > 0;
}

/** The tooltip for a blocked task's counter. */
export function taskBlockedTitle(
  task: Pick<Task, "state" | "blockers">,
): string {
  const n = task.blockers ?? 0;
  return n === 1
    ? "Blocked by 1 incomplete reference"
    : `Blocked by ${n} incomplete references`;
}

/**
 * Pill classes for a task state. Each state keeps its own hue (zinc draft,
 * gray blocked, amber queued, blue running, cyan input, emerald settled, red
 * failed, light-blue review, teal completed, gray cancelled) rather than
 * following the customizable primary color.
 */
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

/**
 * Text color for a task state's label, matching {@link taskStateClass}. Use it
 * where only the hue is wanted (e.g. the state tabs and their counts) rather
 * than a full pill.
 */
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

/** A task's display state label. */
export function taskStateLabelFor(task: Pick<Task, "state">): string {
  return taskStateLabel(task.state);
}

/** The three reference-edge kinds, in display order. */
export const TASK_REFERENCE_KINDS: {
  value: TaskReferenceKind;
  label: string;
}[] = [
  { value: TaskReferenceKind.INFORMATIONAL, label: "Informational" },
  { value: TaskReferenceKind.BLOCKING, label: "Blocking" },
  { value: TaskReferenceKind.SUPERSEDES, label: "Supersedes" },
];

/** Human-readable label for a reference kind. */
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

/** The icon for a reference kind: lock, swap, or plain link. */
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

/** Text color for a reference kind. */
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

/** Pill/chip classes for a reference kind. */
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

/** Pill classes matching {@link taskStateLabelFor}. */
export function taskStateClassFor(task: Pick<Task, "state">): string {
  return taskStateClass(task.state);
}

/** The bare name of a task priority ("Low", "Normal", …). */
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

/**
 * Human-readable label for a task priority. Pass `withDimension` where the
 * value stands alone (such as the task header badge), so "Normal" reads as
 * "Normal priority" instead of an unlabelled "Normal".
 */
export function taskPriorityLabel(
  priority: TaskPriority,
  withDimension = false,
): string {
  const name = taskPriorityName(priority);
  return withDimension ? `${name} priority` : name;
}

/**
 * Text color for a priority label, without the pill background or border.
 */
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

/**
 * Pill classes for a task priority, warming from green (low) through yellow
 * (normal) and orange (high) to red (critical).
 */
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

/** The bare name of a task size ("Small", "Medium", …). */
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

/**
 * Human-readable label for a task size. Pass `withDimension` where the value
 * stands alone (such as the task header badge), so "Medium" reads as
 * "Medium size" instead of an unlabelled "Medium".
 */
export function taskSizeLabel(size: TaskSize, withDimension = false): string {
  const name = taskSizeName(size);
  return withDimension ? `${name} size` : name;
}

/** Text color for a size label, without the pill background or border. */
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

/**
 * Pill classes for a task size, warming from green (small) through yellow
 * (medium) and orange (big) to red (epic).
 */
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

/** The sizes shown in a picker, ascending from small. */
export const TASK_SIZES: { value: TaskSize; label: string }[] = [
  { value: TaskSize.SMALL, label: "Small" },
  { value: TaskSize.MEDIUM, label: "Medium" },
  { value: TaskSize.BIG, label: "Big" },
  { value: TaskSize.EPIC, label: "Epic" },
];

/** Ascending sort rank for sizes. */
export const TASK_SIZE_RANK: Record<number, number> = {
  [TaskSize.SMALL]: 0,
  [TaskSize.MEDIUM]: 1,
  [TaskSize.BIG]: 2,
  [TaskSize.EPIC]: 3,
};

/**
 * The default close-out preselected by the close-out modal: a failed task is
 * usually abandoned (cancelled), everything else is completed.
 */
export function defaultCloseOutOutcome(state: TaskState): TaskState {
  return state === TaskState.FAILED ? TaskState.CANCELLED : TaskState.COMPLETED;
}

/** The states shown as tabs in the task list. */ export const TASK_STATE_TABS: {
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

/**
 * Default ordering for the All tab: the states a user should look at first
 * (failures, dialogs, settled runs) sort above the rest.
 */
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

/** A tag's CSS color, normalized to a `#rrggbb` string (grey when unset). */
export function tagColor(color?: string): string {
  const value = (color ?? "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value.toLowerCase() : "#a1a1aa";
}

/**
 * Inline style for a colored tag chip: the tag hue as text/outline with a
 * translucent fill. Appending an alpha suffix to a `#rrggbb` yields an
 * 8-digit CSS hex color.
 */
export function tagStyle(color?: string): Record<string, string> {
  const value = tagColor(color);
  return {
    color: value,
    borderColor: `${value}66`,
    backgroundColor: `${value}1a`,
  };
}
