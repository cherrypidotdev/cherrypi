#!/usr/bin/env node

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

import { createHash, randomBytes } from "node:crypto";
import { homedir } from "node:os";
import { join } from "node:path";
import { DatabaseSync } from "node:sqlite";

function args() {
  const out = { db: "", reset: false };
  const argv = process.argv.slice(2);
  for (let i = 0; i < argv.length; i += 1) {
    if (argv[i] === "--db") out.db = argv[++i] ?? "";
    else if (argv[i] === "--reset") out.reset = true;
  }
  out.db ||= process.env.CHERRYPI_AGENCY_DATABASE_DSN || "";
  out.db ||= join(homedir(), ".cherrypi-agency", "agency.sqlite3");
  return out;
}

const { db: dbPath, reset } = args();
const db = new DatabaseSync(dbPath);
db.exec("PRAGMA foreign_keys = ON");

const now = Date.now();
const minute = 60_000;
const day = 24 * 60 * minute;

function migrate() {
  const exists = db
    .prepare(
      "SELECT name FROM sqlite_master WHERE type='table' AND name='tasks'",
    )
    .get();
  if (!exists) {
    throw new Error(
      `No agency schema in ${dbPath}; start the agency once to migrate it.`,
    );
  }
}

function resetSeedData() {
  db.exec(`
    DELETE FROM task_session_entries;
    DELETE FROM task_logs;
    DELETE FROM task_tags;
    DELETE FROM task_links;
    DELETE FROM task_references;
    DELETE FROM task_messages;
    DELETE FROM task_artifacts;
    DELETE FROM task_attachments;
    DELETE FROM task_favorites;
    DELETE FROM task_leads;
    DELETE FROM task_subscriptions;
    DELETE FROM task_teams;
    DELETE FROM task_team_prompts;
    DELETE FROM notifications;
    DELETE FROM agent_usage;
    DELETE FROM agent_logs;
    DELETE FROM agent_domain_logs;
    DELETE FROM agent_task_limits;
    DELETE FROM agent_tag_limits;
    DELETE FROM agent_project_limits;
    DELETE FROM agent_project_tag_limits;
    DELETE FROM agent_owners;
    DELETE FROM webhooks WHERE agent_id IS NOT NULL;
    DELETE FROM project_team_memberships;
    DELETE FROM project_team_invitations;
    DELETE FROM project_actor_memberships;
    DELETE FROM project_ancestors;
    DELETE FROM project_parent_invitations;
    DELETE FROM project_logs;
    DELETE FROM prompts;
    DELETE FROM tasks;
    DELETE FROM projects;
    DELETE FROM tags;
    DELETE FROM actor_settings
      WHERE actor_id IN (SELECT id FROM actors WHERE kind = 'agent');
    DELETE FROM personal_access_tokens
      WHERE actor_id IN (SELECT id FROM actors WHERE kind = 'agent');
    DELETE FROM agents;
    DELETE FROM actors WHERE kind = 'agent';
  `);
}

function existingData() {
  return Boolean(
    db.prepare("SELECT 1 FROM agents LIMIT 1").get() ||
    db.prepare("SELECT 1 FROM projects LIMIT 1").get(),
  );
}

function insertActor(username, name, description) {
  const row = db
    .prepare(
      `INSERT INTO actors
         (kind, username, name, description, created_at, updated_at)
       VALUES ('agent', ?, ?, ?, ?, ?)`,
    )
    .run(username, name, description, now, now);
  return Number(row.lastInsertRowid);
}

function insertAgent(actorId) {
  const row = db
    .prepare(
      `INSERT INTO agents (actor_id, last_seen_at, role)
       VALUES (?, ?, 'member')`,
    )
    .run(actorId, null);
  return Number(row.lastInsertRowid);
}

function insertProject(project) {
  db.prepare(
    `INSERT INTO projects
       (id, name, slug, description, prefix, task_counter, parent_project_id,
        created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?)`,
  ).run(
    project.id,
    project.name,
    project.slug,
    project.description,
    project.prefix,
    project.parentProjectId ?? null,
    now,
    now,
  );
  if (project.parentProjectId) {
    db.prepare(
      `INSERT INTO project_ancestors (project_id, ancestor_id, depth)
       VALUES (?, ?, 1)`,
    ).run(project.id, project.parentProjectId);
  }
}

function addMembership(projectId, actorId, role, grantedBy) {
  db.prepare(
    `INSERT OR IGNORE INTO project_actor_memberships
       (project_id, actor_id, role, granted_by_actor_id, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?)`,
  ).run(projectId, actorId, role, grantedBy, now, now);
}

function insertTag(tag) {
  db.prepare(
    `INSERT INTO tags
       (id, name, description, color, project_id, created_at, updated_at)
     VALUES (?, ?, '', ?, ?, ?, ?)`,
  ).run(tag.id, tag.name, tag.color, tag.projectId, now, now);
}

function insertTask(project, task, agent, tagIds) {
  project.seq += 1;
  const id = `${project.prefix}-${project.seq}`;
  const startedAt =
    task.sinceMinutes != null ? now - task.sinceMinutes * minute : null;
  const settledAt =
    task.state === "settled" || task.state === "failed"
      ? (startedAt ?? now) + 10 * minute
      : null;
  const queuedAt =
    task.state === "queued" ? now - task.sinceMinutes * minute : null;
  const row = db
    .prepare(
      `INSERT INTO tasks
         (id, seq, prompt, name, project_id, assignee_actor_id, assignee_pinned,
          priority, state, session_id, error, queued_at, started_at, settled_at,
          created_at, updated_at, interactive, size, state_updated_at,
          creator_actor_id, auto_title, participation)
       VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, '', '', ?, ?, ?, ?, ?, 1, 'medium', ?,
               1, 0, 'project')`,
    )
    .run(
      id,
      project.seq,
      task.title,
      task.title,
      project.id,
      agent?.actorId ?? null,
      task.pinned ? 1 : 0,
      task.state,
      queuedAt,
      startedAt,
      settledAt,
      now - day,
      now,
      now - day,
    );
  const rowid = Number(row.lastInsertRowid);
  for (const tagId of tagIds) {
    if (!tagId) {
      throw new Error(
        `Task ${task.title} has an unknown tag: ${tagIds.join(", ")}`,
      );
    }
    db.prepare("INSERT INTO task_tags (task_rowid, tag_id) VALUES (?, ?)").run(
      rowid,
      tagId,
    );
  }
  return id;
}

function insertLimits(agentId, plan, tagIdFor) {
  db.prepare(
    `INSERT INTO agent_task_limits
       (agent_id, max_concurrent, max_concurrent_per_project)
     VALUES (?, ?, ?)`,
  ).run(agentId, plan.maxConcurrent, plan.projectDefault);
  for (const [name, cap] of Object.entries(plan.tagLimits ?? {})) {
    db.prepare(
      `INSERT INTO agent_tag_limits (agent_id, tag_name, max_concurrent)
       VALUES (?, ?, ?)`,
    ).run(agentId, name.toLowerCase(), cap);
  }
  for (const [projectId, cap] of Object.entries(plan.projectLimits ?? {})) {
    db.prepare(
      `INSERT INTO agent_project_limits (agent_id, project_id, max_concurrent)
       VALUES (?, ?, ?)`,
    ).run(agentId, projectId, cap);
  }
  for (const limit of plan.projectTagLimits ?? []) {
    const tagId = tagIdFor(limit.projectId, limit.tag);
    if (!tagId) {
      throw new Error(`Unknown tag ${limit.projectId}:${limit.tag}`);
    }
    db.prepare(
      `INSERT INTO agent_project_tag_limits
         (agent_id, project_id, tag_id, max_concurrent)
       VALUES (?, ?, ?, ?)`,
    ).run(agentId, limit.projectId, tagId, limit.cap);
  }
}

function mintPAT(actorId, username) {
  const token = `pat_${randomBytes(32).toString("base64url")}`;
  const prefix = `pat_${token.slice(4, 12)}`;
  db.prepare(
    `INSERT INTO personal_access_tokens
       (actor_id, name, token_hash, prefix, created_at)
     VALUES (?, ?, ?, ?, ?)`,
  ).run(
    actorId,
    "concurrency demo",
    createHash("sha256").update(token).digest(),
    prefix,
    now,
  );
  return { username, token };
}

const PROJECTS = [
  {
    id: "p-agency",
    name: "Agency",
    slug: "agency",
    description: "Core scheduler, console and task lifecycle.",
    prefix: "AGENCY",
    tags: ["backend", "frontend", "infra", "docs", "review", "design"],
  },
  {
    id: "p-console",
    name: "Console",
    slug: "console",
    description: "The Vue frontend and its design system.",
    prefix: "CONSOLE",
    parentProjectId: "p-agency",
    tags: ["frontend", "design", "backend"],
  },
  {
    id: "p-website",
    name: "Website",
    slug: "website",
    description: "Marketing site and download page.",
    prefix: "WEB",
    tags: ["frontend", "design", "docs", "review"],
  },
  {
    id: "p-kb",
    name: "Knowledgebase",
    slug: "knowledgebase",
    description: "Searchable product documentation.",
    prefix: "KB",
    tags: ["backend", "frontend", "design", "docs"],
  },
];

const TAG_COLORS = {
  backend: "#3b82f6",
  frontend: "#8b5cf6",
  design: "#ec4899",
  docs: "#eab308",
  infra: "#06b6d4",
  review: "#f97316",
};

const AGENTS = [
  {
    username: "builder-1",
    name: "builder",
    description:
      "Implements features across the agency, console and website. Default worker for backend and frontend work.",
    projects: ["p-agency", "p-console", "p-website"],
    maxConcurrent: 30,
    projectDefault: 8,
    tagLimits: { frontend: 8, backend: 8, design: 5 },
    projectLimits: { "p-agency": 9, "p-console": 3, "p-website": 2 },
    projectTagLimits: [
      { projectId: "p-agency", tag: "frontend", cap: 4 },
      { projectId: "p-console", tag: "frontend", cap: 4 },
      { projectId: "p-website", tag: "design", cap: 2 },
    ],
    held: [
      ["AGENCY-1", "running", "Plan slot visualization", ["frontend"], 22],
      [
        "AGENCY-2",
        "running",
        "Stop sidebar flicker on collapse",
        ["frontend"],
        8,
      ],
      ["AGENCY-3", "input", "Webhook retry with backoff", ["frontend"], 14],
      ["AGENCY-4", "running", "Slot usage query", ["frontend"], 37],
      ["AGENCY-5", "running", "Scheduler readiness probe", ["backend"], 12],
      ["AGENCY-6", "input", "Task state transition audit", ["backend"], 19],
      ["AGENCY-7", "settled", "Event consumer cleanup", ["backend"], 41],
      ["AGENCY-8", "settled", "Compact task tables on mobile", ["design"], 55],
      ["AGENCY-9", "failed", "Broken dashboard tile", ["design"], 63],
      ["CONSOLE-1", "running", "Console keyboard shortcuts", ["frontend"], 16],
      ["CONSOLE-2", "settled", "Task table density", ["frontend"], 48],
      ["CONSOLE-3", "running", "Sidebar section animation", ["frontend"], 27],
      ["WEB-1", "running", "Download page art direction", ["design"], 35],
      ["WEB-2", "settled", "Pricing copy pass", ["frontend"], 72],
    ],
    queued: [
      ["AGENCY-10", "Slot bars in the sidebar", ["frontend"], 19],
      ["AGENCY-11", "Capacity board on Project pages", ["design"], 11],
      ["CONSOLE-4", "Console table filters", ["frontend"], 6],
      ["CONSOLE-5", "Keyboard shortcut help", ["backend"], 4],
      ["WEB-3", "Download page analytics", ["frontend"], 2],
      ["WEB-4", "OG image refresh", ["design"], 1],
    ],
  },
  {
    username: "design-1",
    name: "design",
    description:
      "Owns visual direction, illustrations and the design system. Limited by the shared #design tag.",
    projects: ["p-website", "p-kb", "p-console"],
    maxConcurrent: 8,
    projectDefault: 3,
    tagLimits: { design: 4 },
    projectLimits: { "p-website": 3, "p-kb": 2, "p-console": 2 },
    projectTagLimits: [
      { projectId: "p-website", tag: "design", cap: 3 },
      { projectId: "p-kb", tag: "design", cap: 2 },
    ],
    held: [
      ["WEB-5", "running", "Hero illustration", ["design"], 31],
      ["WEB-6", "settled", "Favicon set", ["design"], 68],
      ["KB-1", "running", "Onboarding illustrations", ["design"], 16],
      ["CONSOLE-6", "input", "Empty state artwork", ["design"], 9],
    ],
    queued: [
      ["WEB-7", "Download page illustrations", ["design"], 24],
      ["KB-2", "Card illustrations for guides", ["design"], 13],
      ["CONSOLE-7", "Settings icon refresh", ["design"], 5],
    ],
  },
  {
    username: "reviewer-1",
    name: "reviewer",
    description:
      "Reviews changes across projects and catches release blockers before they land.",
    projects: ["p-agency", "p-website"],
    maxConcurrent: 6,
    projectDefault: 2,
    tagLimits: { review: 3, frontend: 3 },
    projectLimits: { "p-agency": 2, "p-website": 2 },
    projectTagLimits: [{ projectId: "p-agency", tag: "backend", cap: 2 }],
    held: [
      [
        "AGENCY-12",
        "running",
        "Review auth refactor",
        ["backend", "review"],
        12,
      ],
      [
        "AGENCY-13",
        "input",
        "Review new task composer",
        ["frontend", "review"],
        6,
      ],
      ["WEB-8", "settled", "Review pricing copy", ["docs", "review"], 122],
    ],
    queued: [
      ["AGENCY-14", "Review scheduler cancellation", ["backend", "review"], 23],
      ["AGENCY-15", "Review slot tooltips", ["frontend", "review"], 7],
    ],
  },
  {
    username: "docs-1",
    name: "docs",
    description:
      "Writes and refreshes documentation. A single #docs slot keeps the voice consistent.",
    projects: ["p-agency", "p-website"],
    maxConcurrent: 4,
    projectDefault: 1,
    tagLimits: { docs: 1 },
    projectLimits: { "p-agency": 1, "p-website": 1 },
    projectTagLimits: [
      { projectId: "p-agency", tag: "docs", cap: 1 },
      { projectId: "p-website", tag: "docs", cap: 1 },
    ],
    held: [["AGENCY-16", "running", "Rewrite file storage docs", ["docs"], 31]],
    queued: [
      ["AGENCY-17", "Document the slot lifecycle", ["docs"], 33],
      ["AGENCY-18", "Refresh LEGACY notes", ["docs"], 9],
      ["WEB-9", "Publish download guide", ["docs"], 12],
    ],
  },
  {
    username: "ops-1",
    name: "ops",
    description:
      "Maintains the runners and infrastructure. Both infra slots are held by settled work.",
    projects: ["p-agency"],
    maxConcurrent: 6,
    projectDefault: 2,
    tagLimits: { infra: 2 },
    projectLimits: { "p-agency": 2 },
    projectTagLimits: [{ projectId: "p-agency", tag: "infra", cap: 2 }],
    held: [
      ["AGENCY-19", "settled", "Rotate runner build cache", ["infra"], 184],
      ["AGENCY-20", "failed", "Audit Docker address pools", ["infra"], 302],
    ],
    queued: [
      ["AGENCY-21", "Widen runner address pools", ["infra"], 71],
      ["AGENCY-22", "Prune stale job networks", ["infra"], 42],
    ],
  },
  {
    username: "scout-1",
    name: "scout",
    description:
      "A small spare agent for console chores. Plenty of overall room, but its project cap is full.",
    projects: ["p-console"],
    maxConcurrent: 4,
    projectDefault: 1,
    tagLimits: {},
    projectLimits: { "p-console": 1 },
    projectTagLimits: [],
    held: [
      ["CONSOLE-8", "running", "Console keyboard audit", ["frontend"], 64],
    ],
    queued: [["CONSOLE-9", "Console search polish", ["frontend"], 18]],
  },
];

try {
  migrate();
  if (existingData() && !reset) {
    throw new Error(
      `${dbPath} already has agents or projects; re-run with --reset to replace the demo data.`,
    );
  }

  if (reset) {
    db.exec("PRAGMA foreign_keys = OFF");
    resetSeedData();
    db.exec("PRAGMA foreign_keys = ON");
  }

  db.exec("BEGIN");
  try {
    const admin = db
      .prepare("SELECT id FROM actors WHERE kind = 'user' ORDER BY id LIMIT 1")
      .get();
    if (!admin)
      throw new Error("No human account exists; start the agency first.");
    const adminId = Number(admin.id);

    const projectState = new Map();
    for (const project of PROJECTS) {
      insertProject(project);
      projectState.set(project.id, {
        ...project,
        seq: 0,
        tags: new Map(),
        tagIds: new Map(),
      });
      addMembership(project.id, adminId, "owner", adminId);
    }

    for (const project of PROJECTS) {
      const state = projectState.get(project.id);
      for (const name of project.tags) {
        const tag = {
          id: `tag-${project.id}-${name}`,
          name,
          color: TAG_COLORS[name] ?? "#94a3b8",
          projectId: project.id,
        };
        insertTag(tag);
        state.tags.set(name, tag);
        state.tagIds.set(name, tag.id);
      }
    }

    const agentState = new Map();
    for (const plan of AGENTS) {
      const actorId = insertActor(plan.username, plan.name, plan.description);
      const agentId = insertAgent(actorId);
      db.prepare(
        "INSERT INTO agent_owners (agent_actor_id, owner_actor_id, created_at) VALUES (?, ?, ?)",
      ).run(actorId, adminId, now);
      for (const projectId of plan.projects) {
        addMembership(projectId, actorId, "member", adminId);
      }
      agentState.set(plan.username, { ...plan, actorId, agentId });
      insertLimits(agentId, plan, (projectId, name) =>
        projectState.get(projectId)?.tagIds.get(name),
      );
    }

    for (const plan of AGENTS) {
      const agent = agentState.get(plan.username);
      for (const [spec, state, title, tags, minutes] of plan.held) {
        const project = PROJECTS.find((item) =>
          spec.startsWith(`${item.prefix}-`),
        );
        const stateValue = projectState.get(project.id);
        insertTask(
          stateValue,
          { state, title, sinceMinutes: minutes, pinned: true },
          agent,
          tags.map((name) => stateValue.tagIds.get(name)),
        );
      }
      for (const [spec, title, tags, minutes] of plan.queued) {
        const project = PROJECTS.find((item) =>
          spec.startsWith(`${item.prefix}-`),
        );
        const stateValue = projectState.get(project.id);
        insertTask(
          stateValue,
          { state: "queued", title, sinceMinutes: minutes, pinned: true },
          agent,
          tags.map((name) => stateValue.tagIds.get(name)),
        );
      }
    }

    for (const project of projectState.values()) {
      db.prepare("UPDATE projects SET task_counter = ? WHERE id = ?").run(
        project.seq,
        project.id,
      );
    }

    const tokens = AGENTS.map((plan) => {
      const agent = agentState.get(plan.username);
      return mintPAT(agent.actorId, plan.username);
    });

    db.exec("COMMIT");

    console.log(
      `Seeded ${AGENTS.length} agents, ${PROJECTS.length} projects, 58 slots.`,
    );
    console.log(`Database: ${dbPath}`);
    console.log("Agent tokens (dev only):");
    for (const item of tokens) console.log(`  ${item.username} ${item.token}`);
  } catch (err) {
    db.exec("ROLLBACK");
    throw err;
  }
} finally {
  db.close();
}
