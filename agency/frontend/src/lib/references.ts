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

import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useUsers } from "@/composables/useUsers";
import type { AccountActor } from "@/lib/account";
import { taskName } from "@/lib/task";
import { teamApi, type Team } from "@/lib/team";
import type { Agent, Project, Task } from "@cherrypi/sdk";
import type { MarkdownIt, StateCore } from "markdown-it";
import { ref } from "vue";

export type ReferenceKind = "actor" | "project" | "team" | "task";

export interface EntityReference {
  kind: ReferenceKind;
  key: string;
}

export interface ReferenceTarget extends EntityReference {
  label: string;

  href: string;

  avatarUrl: string;
}

const REFERENCE_HREF = /^ref:(actor|project|team|task)\/(.+)$/;

const KIND_LABEL: Record<ReferenceKind, string> = {
  actor: "Actor",
  project: "Project",
  team: "Team",
  task: "Task",
};

export function parseReference(href: string): EntityReference | null {
  const match = REFERENCE_HREF.exec(href);
  if (!match) return null;
  let key = match[2] ?? "";
  try {
    key = decodeURIComponent(key);
  } catch {}
  if (!key.trim()) return null;
  return { kind: match[1] as ReferenceKind, key };
}

export function referenceHref(reference: EntityReference): string {
  return `ref:${reference.kind}/${reference.key}`;
}

export function referenceKindLabel(kind: ReferenceKind): string {
  return KIND_LABEL[kind];
}

const { users, loadUsers } = useUsers();
const { agents, loadAgents } = useAgents();
const { projects, loadProjects } = useProjects();

const teams = ref<Team[]>([]);
let teamsLoaded = false;
let teamsPending: Promise<void> | null = null;
let directoryLoaded = false;
let directoryPending: Promise<void> | null = null;

async function loadTeams(force = false): Promise<void> {
  if (teamsPending) return teamsPending;
  if (teamsLoaded && !force) return;
  teamsPending = teamApi
    .list()
    .then(({ teams: list }) => {
      teams.value = list;
      teamsLoaded = true;
    })
    .catch(() => {})
    .finally(() => {
      teamsPending = null;
    });
  return teamsPending;
}

export async function loadReferenceDirectory(force = false): Promise<void> {
  if (directoryPending) return directoryPending;
  if (directoryLoaded && !force) return;
  directoryPending = Promise.all([
    loadUsers(force),
    loadAgents(force),
    loadProjects(force),
    loadTeams(force),
  ])
    .then(() => {
      directoryLoaded = true;
    })
    .catch(() => {})
    .finally(() => {
      directoryPending = null;
    });
  return directoryPending;
}

export interface ReferenceDirectory {
  users: AccountActor[];
  agents: Agent[];
  projects: Project[];
  teams: Team[];
}

export function referenceDirectory(): ReferenceDirectory {
  return {
    users: users.value,
    agents: agents.value,
    projects: projects.value,
    teams: teams.value,
  };
}

function initials(label: string): string {
  const parts = label.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (parts[0]![0]! + parts[parts.length - 1]![0]!).toUpperCase();
}

function actorTarget(
  user: AccountActor | undefined,
  agent: Agent | undefined,
  key: string,
): ReferenceTarget {
  if (agent) {
    return {
      kind: "actor",
      key,
      label: agent.displayName || agent.username || key,
      href: `/agents/${encodeURIComponent(agent.username || key)}`,
      avatarUrl: agent.picture?.url ?? "",
    };
  }
  if (user) {
    return {
      kind: "actor",
      key,
      label: user.name || user.username || key,
      href: `/users/${encodeURIComponent(user.username || key)}`,
      avatarUrl: user.picture?.url ?? "",
    };
  }
  return {
    kind: "actor",
    key,
    label: `@${key}`,
    href: `/users/${encodeURIComponent(key)}`,
    avatarUrl: "",
  };
}

export function resolveReference(
  reference: EntityReference,
  fallbackLabel = "",
): ReferenceTarget {
  const { kind, key } = reference;

  if (kind === "task") {
    return {
      kind,
      key,
      label: fallbackLabel.trim() || `#${key}`,
      href: `/tasks/${encodeURIComponent(key)}`,
      avatarUrl: "",
    };
  }

  if (kind === "project") {
    const project = projects.value.find(
      (item) => item.slug === key || item.id === key,
    );
    if (project) {
      return {
        kind,
        key,
        label: project.name || project.slug || key,
        href: `/projects/${encodeURIComponent(project.slug)}`,
        avatarUrl: project.picture?.url ?? "",
      };
    }
    return {
      kind,
      key,
      label: fallbackLabel.trim() || `@${key}`,
      href: `/projects/${encodeURIComponent(key)}`,
      avatarUrl: "",
    };
  }

  if (kind === "team") {
    const team = teams.value.find((item) => String(item.id) === key);
    if (team) {
      return {
        kind,
        key,
        label: team.name || fallbackLabel.trim() || key,
        href: `/teams/${team.id}`,
        avatarUrl: "",
      };
    }
    return {
      kind,
      key,
      label: fallbackLabel.trim() || `@${key}`,
      href: `/teams/${encodeURIComponent(key)}`,
      avatarUrl: "",
    };
  }

  const user = users.value.find(
    (item) => item.username === key || String(item.id) === key,
  );
  const agent = agents.value.find(
    (item) => item.username === key || item.actorId === key || item.id === key,
  );
  return actorTarget(user, agent, key);
}

const HTML_ESCAPES: Record<string, string> = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
  "'": "&#39;",
};

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (char) => HTML_ESCAPES[char] ?? char);
}

export function referenceChipHtml(target: ReferenceTarget): string {
  const label = target.label || target.key;
  const avatar = target.avatarUrl
    ? `<img class="ref-chip-avatar" src="${escapeHtml(target.avatarUrl)}" alt="" loading="lazy">`
    : kindAvatar(target.kind, label);
  const title = `${referenceKindLabel(target.kind)}: ${label}`;
  return (
    `<a class="ref-chip ref-chip--${target.kind}"` +
    ` href="${escapeHtml(target.href)}"` +
    ` data-ref="${escapeHtml(referenceHref(target))}"` +
    ` title="${escapeHtml(title)}">` +
    avatar +
    `<span class="ref-chip-label">${escapeHtml(label)}</span>` +
    `</a>`
  );
}

function kindAvatar(kind: ReferenceKind, label: string): string {
  if (kind === "task") return "";
  return `<span class="ref-chip-avatar ref-chip-avatar--initials" aria-hidden="true">${escapeHtml(initials(label))}</span>`;
}

export function referencePlugin(
  md: MarkdownIt,
  resolve: (
    reference: EntityReference,
    label: string,
  ) => ReferenceTarget = resolveReference,
): void {
  md.core.ruler.after("inline", "cherrypi_references", (state: StateCore) => {
    for (const block of state.tokens) {
      const children = block.children;
      if (block.type !== "inline" || !children) continue;

      for (let index = 0; index < children.length; index += 1) {
        const open = children[index]!;
        if (open.type !== "link_open") continue;

        const reference = parseReference(String(open.attrGet("href") ?? ""));
        if (!reference) continue;

        let close = -1;
        for (let next = index + 1; next < children.length; next += 1) {
          const token = children[next]!;
          if (token.type === "link_close") {
            close = next;
            break;
          }

          if (token.type === "link_open") break;
        }
        if (close < 0) continue;

        const label = children
          .slice(index + 1, close)
          .map((token) => token.content)
          .join("")
          .trim();
        const chip = new state.Token("html_inline", "", 0);
        chip.content = referenceChipHtml(resolve(reference, label));
        children.splice(index, close - index + 1, chip);
      }
    }
  });
}

export interface ReferenceSuggestion {
  reference: EntityReference;

  label: string;

  detail: string;

  score: number;
}

function normalize(value: string): string {
  return value.trim().toLocaleLowerCase();
}

function scoreMatch(query: string, ...fields: string[]): number {
  if (!query) return 0;
  const needle = normalize(query);
  let best = -1;
  for (const field of fields) {
    const value = normalize(field);
    if (!value) continue;
    const at = value.indexOf(needle);
    if (at < 0) continue;
    const boundary = at === 0 || /[\s\-_/]/.test(value[at - 1] ?? "");
    const score = at === 0 ? 0 : boundary ? 1 : 2;
    if (best < 0 || score < best) best = score;
  }
  return best;
}

function compareSuggestions(
  a: ReferenceSuggestion,
  b: ReferenceSuggestion,
): number {
  return (
    a.score - b.score ||
    a.label.localeCompare(b.label, undefined, { sensitivity: "base" })
  );
}

export function searchReferences(
  query: string,
  directory: ReferenceDirectory = referenceDirectory(),
  limitPerKind = 5,
): ReferenceSuggestion[] {
  const actors: ReferenceSuggestion[] = [];
  const seenActors = new Set<string>();

  for (const user of directory.users) {
    const username = user.username || "";
    if (seenActors.has(username)) continue;
    const score = scoreMatch(query, username, user.name);
    if (!query || score >= 0) {
      seenActors.add(username);
      actors.push({
        reference: { kind: "actor", key: username || String(user.id) },
        label: user.name || username || String(user.id),
        detail: username ? `@${username} · User` : "User",
        score,
      });
    }
  }
  for (const agent of directory.agents) {
    const username = agent.username || "";
    if (seenActors.has(username)) continue;
    const score = scoreMatch(query, username, agent.displayName);
    if (!query || score >= 0) {
      seenActors.add(username);
      actors.push({
        reference: { kind: "actor", key: username || agent.id },
        label: agent.displayName || username || agent.id,
        detail: username ? `@${username} · Agent` : "Agent",
        score,
      });
    }
  }

  const projects: ReferenceSuggestion[] = [];
  for (const project of directory.projects) {
    if (project.archivedAt) continue;
    const score = scoreMatch(query, project.slug, project.name);
    if (!query || score >= 0) {
      projects.push({
        reference: { kind: "project", key: project.slug || project.id },
        label: project.name || project.slug,
        detail: `@${project.slug} · Project`,
        score,
      });
    }
  }

  const teamSuggestions: ReferenceSuggestion[] = [];
  for (const team of directory.teams) {
    const score = scoreMatch(query, team.name);
    if (!query || score >= 0) {
      teamSuggestions.push({
        reference: { kind: "team", key: String(team.id) },
        label: team.name,
        detail:
          team.member_count === 1 ? "1 member" : `${team.member_count} members`,
        score,
      });
    }
  }

  return [
    ...actors.sort(compareSuggestions).slice(0, limitPerKind),
    ...projects.sort(compareSuggestions).slice(0, limitPerKind),
    ...teamSuggestions.sort(compareSuggestions).slice(0, limitPerKind),
  ];
}

export function markdownLabel(value: string): string {
  return value.replace(/[\\[\]]/g, (char) => `\\${char}`);
}

export function referenceMarkdown(reference: EntityReference): string {
  const target = resolveReference(reference);
  const label = markdownLabel(target.label);
  return `[@${label}](ref:${reference.kind}/${reference.key})`;
}

export function taskReferenceMarkdown(task: Pick<Task, "id">): string {
  return `[#${markdownLabel(task.id)}](ref:task/${task.id})`;
}

export function taskSuggestion(task: Task): ReferenceSuggestion {
  return {
    reference: { kind: "task", key: task.id },
    label: task.id,
    detail: taskName(task),
    score: 0,
  };
}
