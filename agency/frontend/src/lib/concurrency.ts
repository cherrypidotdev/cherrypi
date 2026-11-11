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

import { t } from "@/i18n/translate";
import { taskIncompleteBlockers, taskWaitingReason } from "@/lib/task";
import { create } from "@bufbuild/protobuf";
import {
  AgentStatus,
  AgentTaskLimitsSchema,
  TaskState,
  type Agent,
  type AgentTaskLimits,
  type Project,
  type Tag,
  type Task,
} from "@cherrypi/sdk";

export const CONCURRENCY_HELD_STATES = new Set<TaskState>([
  TaskState.RUNNING,
  TaskState.INPUT,
  TaskState.SETTLED,
  TaskState.FAILED,
]);

export type ConcurrencyState = "running" | "input" | "settled" | "failed";

export type CapacityKind = "overall" | "project" | "tag" | "project-tag";

export type BlockerKind =
  | "overall"
  | "project"
  | "tag"
  | "project-tag"
  | "offline"
  | "reference"
  | "reserved";

export interface TagReservation {
  tagName: string;
  reserved: number;
}

export interface ClassFloor {
  reserved: Record<string, number>;

  general: number;

  byClass: Record<string, number>;

  cap: number;
}

export interface ConcurrencyBlocker {
  kind: BlockerKind;
  label: string;
  detail: string;
}

export interface WaitingTask {
  task: Task;

  agent?: Agent;
  blockers: ConcurrencyBlocker[];

  reason?: string;
}

export interface CapacityRow {
  key: string;
  kind: CapacityKind;
  label: string;
  detail?: string;
  project?: Project;
  tag?: Tag;
  used: number;
  cap: number;

  reserved?: number;
  tasks: Task[];
  waiting: number;
  full: boolean;
  override: boolean;
}

export interface AgentConcurrencyBoard {
  kind: "agent";
  agent: Agent;
  projects: Project[];
  limits: AgentTaskLimits;
  held: Task[];
  queued: Task[];
  waiting: WaitingTask[];
  stateCounts: Record<ConcurrencyState, number>;
  overall: CapacityRow;
  projectRows: CapacityRow[];
  tagRows: CapacityRow[];
  uncappedTags: { tag?: Tag; name: string; used: number; waiting: number }[];
  projectTagRows: CapacityRow[];

  reservedByClass: Record<string, number>;
  free: number;
  fullCapCount: number;
  fullProjectCount: number;
  fullTagCount: number;
  fullProjectTagCount: number;
}

export interface ProjectTagCapacityRow {
  agent: Agent;
  tag: Tag;
  global?: CapacityRow;
  project?: CapacityRow;
  waiting: WaitingTask[];
}

export interface ProjectMember {
  agent: Agent;
  board: AgentConcurrencyBoard;
  projectRow: CapacityRow;
  waiting: WaitingTask[];
}

export interface ProjectConcurrencyBoard {
  kind: "project";
  project: Project;
  members: ProjectMember[];
  tagRows: ProjectTagCapacityRow[];
  waiting: WaitingTask[];

  reservations: TagReservation[];
  floor: ClassFloor;
  held: number;
  capSum: number;
  memberCount: number;
  onlineCount: number;
  fullMemberCount: number;
  fullCapCount: number;
}

interface CapacityCounts {
  total: number;
  byProject: Record<string, number>;
  byTag: Record<string, number>;
  byProjectTag: Record<string, Record<string, number>>;
}

function blankCounts(): CapacityCounts {
  return { total: 0, byProject: {}, byTag: {}, byProjectTag: {} };
}

function countTasks(tasks: Task[]): CapacityCounts {
  const counts = blankCounts();
  for (const task of tasks) {
    counts.total += 1;
    if (task.projectId) {
      counts.byProject[task.projectId] =
        (counts.byProject[task.projectId] ?? 0) + 1;
    }
    for (const tag of task.tags) {
      const name = tag.name.toLowerCase();
      counts.byTag[name] = (counts.byTag[name] ?? 0) + 1;
      if (task.projectId) {
        const byTag = (counts.byProjectTag[task.projectId] ??= {});
        byTag[tag.id] = (byTag[tag.id] ?? 0) + 1;
      }
    }
  }
  return counts;
}

export function projectCap(limits: AgentTaskLimits, projectId: string): number {
  const override = limits.projectLimits.find(
    (limit) => limit.projectId === projectId,
  );
  return override?.maxConcurrent ?? limits.maxConcurrentPerProject;
}

export function tagCap(limits: AgentTaskLimits, name: string): number | null {
  const key = name.toLowerCase();
  return (
    limits.tagLimits.find((limit) => limit.tagName.toLowerCase() === key)
      ?.maxConcurrent ?? null
  );
}

export function projectTagCap(
  limits: AgentTaskLimits,
  projectId: string,
  tagId: string,
): number | null {
  return (
    limits.projectTagLimits.find(
      (limit) => limit.projectId === projectId && limit.tagId === tagId,
    )?.maxConcurrent ?? null
  );
}

function taskTime(task: Task): number {
  const ts = task.queuedAt ?? task.createdAt;
  return ts ? Number(ts.seconds) * 1000 : 0;
}

function sortHeld(tasks: Task[]): Task[] {
  return [...tasks].sort((a, b) => {
    const at = a.startedAt ? Number(a.startedAt.seconds) : 0;
    const bt = b.startedAt ? Number(b.startedAt.seconds) : 0;
    return at - bt;
  });
}

function matchesTaskTag(task: Task, tag: Tag): boolean {
  return task.tags.some(
    (item) =>
      item.id === tag.id || item.name.toLowerCase() === tag.name.toLowerCase(),
  );
}

export function reservedRecord(
  list: TagReservation[] | undefined,
): Record<string, number> {
  const out: Record<string, number> = {};
  for (const item of list ?? []) {
    const name = item.tagName.toLowerCase();
    if (item.reserved <= 0) continue;
    out[name] = Math.max(out[name] ?? 0, item.reserved);
  }
  return out;
}

export function agentReserved(
  projects: Project[],
  byProject: Map<string, TagReservation[]> | undefined,
  reservableProjectIds?: Set<string>,
): Record<string, number> {
  const out: Record<string, number> = {};
  if (!byProject) return out;
  for (const project of projects) {
    if (reservableProjectIds && !reservableProjectIds.has(project.id)) continue;
    for (const [name, slots] of Object.entries(
      reservedRecord(byProject.get(project.id)),
    )) {
      out[name] = Math.max(out[name] ?? 0, slots);
    }
  }
  return out;
}

export function classFloor(
  tasks: Task[],
  reserved: Record<string, number>,
  cap: number,
): ClassFloor {
  const floor: ClassFloor = { reserved, general: 0, byClass: {}, cap };
  for (const task of tasks) {
    let classed = false;
    for (const tag of task.tags) {
      const name = tag.name.toLowerCase();
      if (!(name in reserved)) continue;
      floor.byClass[name] = (floor.byClass[name] ?? 0) + 1;
      classed = true;
    }
    if (!classed) floor.general += 1;
  }
  return floor;
}

export function reservedSlots(reserved: Record<string, number>): number {
  return Object.values(reserved).reduce((sum, slots) => sum + slots, 0);
}

export function fitsFloor(floor: ClassFloor, task: Task): boolean {
  const names = Object.keys(floor.reserved);
  if (!names.length) return true;
  const carried = new Set(task.tags.map((tag) => tag.name.toLowerCase()));
  let reservedTotal = 0;
  let spill = 0;
  let own = 0;
  for (const name of names) {
    const slots = floor.reserved[name]!;
    if (slots <= 0) continue;
    reservedTotal += slots;
    const held = floor.byClass[name] ?? 0;
    if (held > slots) spill += held - slots;
    if (!carried.has(name)) continue;

    if (slots - held > own) own = slots - held;
  }
  const generalFree = floor.cap - reservedTotal - floor.general - spill;
  return own + generalFree >= 1;
}

export function blockingClass(floor: ClassFloor, task: Task): string {
  const carried = new Set(task.tags.map((tag) => tag.name.toLowerCase()));
  const names = Object.keys(floor.reserved).filter(
    (name) => !carried.has(name),
  );
  const pool = names.length ? names : Object.keys(floor.reserved);
  pool.sort(
    (a, b) =>
      (floor.reserved[b] ?? 0) - (floor.reserved[a] ?? 0) || a.localeCompare(b),
  );
  return pool[0] ?? "";
}

export function reservedBlocker(
  floor: ClassFloor,
  task: Task,
): ConcurrencyBlocker | null {
  if (fitsFloor(floor, task)) return null;
  const name = blockingClass(floor, task);
  if (!name) return null;
  const slots = floor.reserved[name] ?? 0;
  const used = floor.byClass[name] ?? 0;
  return {
    kind: "reserved",
    label: t("wait.blocker.reservedLabel", { tag: name }),
    detail: t("wait.blocker.reservedDetail", {
      slots,
      tag: name,
      used,
    }),
  };
}

export function blockersFor(
  agent: Agent,
  limits: AgentTaskLimits,
  counts: CapacityCounts,
  task: Task,
  tagCatalog: Tag[],
  floor?: ClassFloor,
): ConcurrencyBlocker[] {
  const blockers: ConcurrencyBlocker[] = [];
  if (agent.status !== AgentStatus.ONLINE) {
    blockers.push({
      kind: "offline",
      label: t("wait.blocker.offlineLabel"),
      detail: t("wait.blocker.offlineDetail"),
    });
  }

  if (counts.total >= limits.maxConcurrent) {
    blockers.push({
      kind: "overall",
      label: t("wait.blocker.overallLabel", {
        used: counts.total,
        cap: limits.maxConcurrent,
      }),
      detail: t("wait.blocker.overallDetail"),
    });
  }

  if (task.projectId) {
    const cap = projectCap(limits, task.projectId);
    const used = counts.byProject[task.projectId] ?? 0;
    if (used >= cap) {
      blockers.push({
        kind: "project",
        label: t("wait.blocker.projectLabel", { used, cap }),
        detail: t("wait.blocker.projectDetail"),
      });
    }
  }

  for (const tag of task.tags) {
    const cap = tagCap(limits, tag.name);
    const used = counts.byTag[tag.name.toLowerCase()] ?? 0;
    if (cap !== null && used >= cap) {
      blockers.push({
        kind: "tag",
        label: t("wait.blocker.tagLabel", { tag: tag.name, used, cap }),
        detail: t("wait.blocker.tagDetail", { tag: tag.name }),
      });
    }
  }

  if (task.projectId) {
    for (const tag of task.tags) {
      const cap = projectTagCap(limits, task.projectId, tag.id);
      const used = counts.byProjectTag[task.projectId]?.[tag.id] ?? 0;
      if (cap !== null && used >= cap) {
        blockers.push({
          kind: "project-tag",
          label: t("wait.blocker.projectTagLabel", {
            tag: tag.name,
            used,
            cap,
          }),
          detail: t("wait.blocker.projectTagDetail", { tag: tag.name }),
        });
      }
    }
  }

  if (floor) {
    const reserved = reservedBlocker(floor, task);
    if (reserved) blockers.push(reserved);
  }

  const references = taskIncompleteBlockers(task);
  if (references.length) {
    blockers.push({
      kind: "reference",
      label:
        references.length === 1
          ? t("wait.blocker.referenceLabelOne", {
              task: references[0]?.id ?? t("wait.blocker.aTask"),
            })
          : t("wait.blocker.referenceLabelMany", {
              count: references.length,
            }),
      detail: taskWaitingReason(task),
    });
  }

  void tagCatalog;
  return blockers;
}

export function buildAgentBoard(options: {
  agent: Agent;
  projects: Project[];
  limits: AgentTaskLimits;
  tasks: Task[];
  allTags: Tag[];

  reservations?: Map<string, TagReservation[]>;

  reservableProjectIds?: Set<string>;
}): AgentConcurrencyBoard {
  const {
    agent,
    projects,
    limits,
    tasks,
    allTags,
    reservations,
    reservableProjectIds,
  } = options;
  const held = sortHeld(
    tasks.filter((task) => CONCURRENCY_HELD_STATES.has(task.state)),
  );
  const queued = tasks
    .filter((task) => task.state === TaskState.QUEUED)
    .sort((a, b) => taskTime(a) - taskTime(b));
  const counts = countTasks(held);
  const reserved = agentReserved(projects, reservations, reservableProjectIds);
  const floor = classFloor(held, reserved, limits.maxConcurrent);

  const waiting: WaitingTask[] = queued.map((task) => ({
    task,
    agent,
    blockers: blockersFor(agent, limits, counts, task, allTags, floor),
  }));
  const waitingByProject = (projectId: string) =>
    waiting.filter((item) => item.task.projectId === projectId);
  const waitingByTag = (tag: Tag) =>
    waiting.filter((item) => matchesTaskTag(item.task, tag));

  const stateCounts: Record<ConcurrencyState, number> = {
    running: 0,
    input: 0,
    settled: 0,
    failed: 0,
  };
  for (const task of held) {
    if (task.state === TaskState.RUNNING) stateCounts.running += 1;
    else if (task.state === TaskState.INPUT) stateCounts.input += 1;
    else if (task.state === TaskState.SETTLED) stateCounts.settled += 1;
    else if (task.state === TaskState.FAILED) stateCounts.failed += 1;
  }

  const overall: CapacityRow = {
    key: "overall",
    kind: "overall",
    label: t("wait.row.allProjects"),
    detail: t("wait.row.agentCap", { cap: limits.maxConcurrent }),
    used: counts.total,
    cap: limits.maxConcurrent,
    reserved: reservedSlots(reserved),
    tasks: held,
    waiting: waiting.length,
    full: counts.total >= limits.maxConcurrent,
    override: false,
  };

  const projectRows: CapacityRow[] = projects.map((project) => {
    const cap = projectCap(limits, project.id);
    const used = counts.byProject[project.id] ?? 0;
    const rowWaiting = waitingByProject(project.id);
    return {
      key: `project:${project.id}`,
      kind: "project",
      label: project.name,
      detail: limits.projectLimits.some(
        (limit) => limit.projectId === project.id,
      )
        ? "project override"
        : `default project cap ${limits.maxConcurrentPerProject}`,
      project,
      used,
      cap,

      tasks: held.filter((task) => task.projectId === project.id),
      waiting: rowWaiting.length,
      full: used >= cap,
      override: limits.projectLimits.some(
        (limit) => limit.projectId === project.id,
      ),
    };
  });

  projectRows.sort((a, b) => a.label.localeCompare(b.label));

  const tagByName = new Map(
    allTags.map((tag) => [tag.name.toLowerCase(), tag] as const),
  );
  const tagRows: CapacityRow[] = limits.tagLimits.map((limit) => {
    const name = limit.tagName;
    const tag = tagByName.get(name.toLowerCase());
    const used = counts.byTag[name.toLowerCase()] ?? 0;
    const rowWaiting = tag ? waitingByTag(tag) : [];
    return {
      key: `tag:${name.toLowerCase()}`,
      kind: "tag",
      label: tag?.name ?? name,
      detail: t("wait.row.globalNameCap", { cap: limit.maxConcurrent }),
      tag,
      used,
      cap: limit.maxConcurrent,
      tasks: tag
        ? held.filter((task) => matchesTaskTag(task, tag))
        : held.filter((task) =>
            task.tags.some(
              (item) => item.name.toLowerCase() === name.toLowerCase(),
            ),
          ),
      waiting: rowWaiting.length,
      full: used >= limit.maxConcurrent,
      override: false,
    };
  });

  const cappedTagNames = new Set(
    limits.tagLimits.map((limit) => limit.tagName.toLowerCase()),
  );
  const uncappedNames = new Map<
    string,
    { tag?: Tag; name: string; used: number; waiting: number }
  >();
  for (const task of [...held, ...queued]) {
    for (const tag of task.tags) {
      if (cappedTagNames.has(tag.name.toLowerCase())) continue;
      const entry = uncappedNames.get(tag.name.toLowerCase()) ?? {
        tag,
        name: tag.name,
        used: 0,
        waiting: 0,
      };
      entry.tag ??= tag;
      if (CONCURRENCY_HELD_STATES.has(task.state)) entry.used += 1;
      else if (task.state === TaskState.QUEUED) entry.waiting += 1;
      uncappedNames.set(tag.name.toLowerCase(), entry);
    }
  }

  const projectTagRows: CapacityRow[] = [];
  for (const limit of limits.projectTagLimits) {
    const project = projects.find((item) => item.id === limit.projectId);
    const tag =
      allTags.find((item) => item.id === limit.tagId) ??
      held.flatMap((task) => task.tags).find((item) => item.id === limit.tagId);
    if (!project) continue;
    const used = counts.byProjectTag[limit.projectId]?.[limit.tagId] ?? 0;
    const rowWaiting = tag
      ? waiting.filter(
          (item) =>
            item.task.projectId === limit.projectId &&
            matchesTaskTag(item.task, tag),
        ).length
      : 0;
    projectTagRows.push({
      key: `project-tag:${limit.projectId}:${limit.tagId}`,
      kind: "project-tag",
      label: t("wait.row.projectTag", {
        project: project.name,
        tag: tag?.name ?? limit.tagId.slice(0, 8),
      }),
      detail: t("wait.row.projectTagCap", { cap: limit.maxConcurrent }),
      project,
      tag,
      used,
      cap: limit.maxConcurrent,
      tasks: tag
        ? held.filter(
            (task) =>
              task.projectId === limit.projectId && matchesTaskTag(task, tag),
          )
        : [],
      waiting: rowWaiting,
      full: used >= limit.maxConcurrent,
      override: true,
    });
  }

  projectTagRows.sort((a, b) => a.label.localeCompare(b.label));

  const free = Math.max(0, limits.maxConcurrent - counts.total);
  const fullProjectCount = projectRows.filter((row) => row.full).length;
  const fullTagCount = tagRows.filter((row) => row.full).length;
  const fullProjectTagCount = projectTagRows.filter((row) => row.full).length;

  return {
    kind: "agent",
    agent,
    projects,
    limits,
    held,
    queued,
    waiting,
    stateCounts,
    overall,
    projectRows,
    tagRows,
    uncappedTags: [...uncappedNames.values()].sort((a, b) =>
      (a.tag?.name ?? "").localeCompare(b.tag?.name ?? ""),
    ),
    projectTagRows,
    reservedByClass: reserved,
    free,
    fullCapCount:
      (overall.full ? 1 : 0) +
      fullProjectCount +
      fullTagCount +
      fullProjectTagCount,
    fullProjectCount,
    fullTagCount,
    fullProjectTagCount,
  };
}

export function buildProjectBoard(options: {
  project: Project;
  members: Agent[];
  tasks: Task[];
  limitsByAgent: Map<string, AgentTaskLimits>;
  allTags: Tag[];

  reservations?: TagReservation[];

  capacityAgentIds?: Set<string>;
}): ProjectConcurrencyBoard {
  const {
    project,
    members,
    tasks,
    limitsByAgent,
    allTags,
    reservations,
    capacityAgentIds,
  } = options;
  const projectTasks = tasks.filter((task) => task.projectId === project.id);
  const reservationList = reservations ?? [];
  const memberBoards: ProjectMember[] = [];

  for (const agent of members) {
    const limits =
      limitsByAgent.get(agent.id) ??
      create(AgentTaskLimitsSchema, {
        maxConcurrent: 1,
        maxConcurrentPerProject: 1,
      });
    const agentTasks = tasks.filter(
      (task) => task.assigneeActorId === agent.actorId,
    );

    const runsHere = !capacityAgentIds || capacityAgentIds.has(agent.id);
    const board = buildAgentBoard({
      agent,
      projects: [project],
      limits,
      tasks: agentTasks,
      allTags,
      reservations: runsHere
        ? new Map([[project.id, reservationList]])
        : undefined,
    });
    const projectRow =
      board.projectRows.find((row) => row.project?.id === project.id) ??
      board.overall;
    memberBoards.push({
      agent,
      board,
      projectRow,
      waiting: board.waiting.filter(
        (item) => item.task.projectId === project.id,
      ),
    });
  }

  const capSum = memberBoards.reduce(
    (sum, member) =>
      capacityAgentIds && !capacityAgentIds.has(member.agent.id)
        ? sum
        : sum + member.projectRow.cap,
    0,
  );
  const projectHeld = projectTasks.filter((task) =>
    CONCURRENCY_HELD_STATES.has(task.state),
  );
  const floor = classFloor(
    projectHeld,
    reservedRecord(reservationList),
    capSum,
  );

  const memberByActor = new Map(
    memberBoards.map((member) => [member.agent.actorId, member] as const),
  );
  const waiting: WaitingTask[] = projectTasks
    .filter((task) => task.state === TaskState.QUEUED)
    .sort((a, b) => taskTime(a) - taskTime(b))
    .map((task) => {
      const member = memberByActor.get(task.assigneeActorId);

      const viewerTask =
        !!member &&
        !!capacityAgentIds &&
        !capacityAgentIds.has(member.agent.id);

      const reserved = viewerTask ? null : reservedBlocker(floor, task);
      const merged = [
        ...(member?.board.waiting.find((item) => item.task.id === task.id)
          ?.blockers ?? []),
        ...(reserved ? [reserved] : []),
      ];
      const blockers = merged.filter(
        (blocker, index) =>
          merged.findIndex(
            (other) =>
              other.kind === blocker.kind && other.label === blocker.label,
          ) === index,
      );
      if (member) {
        return { task, agent: member.agent, blockers };
      }
      return {
        task,
        blockers,
        reason: taskWaitingReason(task),
      };
    });

  const tagRows: ProjectTagCapacityRow[] = [];
  for (const member of memberBoards) {
    for (const tag of allTags.filter((item) => item.projectId === project.id)) {
      const global = member.board.tagRows.find(
        (row) =>
          row.tag?.id === tag.id || row.key === `tag:${tag.name.toLowerCase()}`,
      );
      const ptag = member.board.projectTagRows.find(
        (row) => row.project?.id === project.id && row.tag?.id === tag.id,
      );
      const usedHere = member.projectRow.tasks.filter((task) =>
        matchesTaskTag(task, tag),
      ).length;
      const waitingHere = member.waiting.filter((item) =>
        matchesTaskTag(item.task, tag),
      ).length;
      if (!global && !ptag && usedHere === 0 && waitingHere === 0) continue;
      tagRows.push({
        agent: member.agent,
        tag,
        global,
        project: ptag,
        waiting: member.waiting.filter((item) =>
          matchesTaskTag(item.task, tag),
        ),
      });
    }
  }
  tagRows.sort((a, b) => {
    const byTag = a.tag.name.localeCompare(b.tag.name);
    return byTag || a.agent.username.localeCompare(b.agent.username);
  });

  const fullMemberCount = memberBoards.filter((member) => {
    const caps = [
      member.board.overall,
      ...member.board.projectRows,
      ...member.board.tagRows,
      ...member.board.projectTagRows,
    ];
    return caps.some((row) => row.full);
  }).length;
  const fullCapCount = memberBoards.reduce(
    (sum, member) =>
      sum +
      (member.board.overall.full ? 1 : 0) +
      member.board.projectRows.filter((row) => row.full).length +
      member.board.tagRows.filter((row) => row.full).length +
      member.board.projectTagRows.filter((row) => row.full).length,
    0,
  );

  return {
    kind: "project",
    project,
    members: memberBoards,
    tagRows,
    waiting,
    reservations: reservationList,
    floor,
    held: memberBoards.reduce((sum, member) => sum + member.projectRow.used, 0),
    capSum,
    memberCount: memberBoards.length,
    onlineCount: memberBoards.filter(
      (member) => member.agent.status === AgentStatus.ONLINE,
    ).length,
    fullMemberCount,
    fullCapCount,
  };
}

export function capacityPercent(used: number, cap: number): number {
  if (cap <= 0) return 100;
  return Math.max(0, Math.min(100, (used / cap) * 100));
}

export function capacityLabel(used: number, cap: number): string {
  return `${used}/${cap}`;
}
