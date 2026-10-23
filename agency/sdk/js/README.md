# SDK (JavaScript) — `@cherrypi/sdk`

The typed TypeScript client for the **CherryPi Agency**. `AgencyClient` wraps
the agency's Connect gRPC surface: agents, users, settings, notifications, and
webhooks, plus tasks, tags, terminals, and usage. The agency is the
only public surface — the browser SPA uses this SDK, so any SDK consumer has the
same access as the dashboard.

## Install

The package is not published yet; consume it from this repo by path (or via npm
workspaces) until then:

```json
{
  "dependencies": {
    "@cherrypi/sdk": "file:../../agency/sdk/js"
  }
}
```

```sh
cd agency/sdk/js
npm install
npm run build # emits dist/ (or `make build`)
```

The generated protobuf-ES client lives in `src/gen` (gitignored). It comes from
`buf generate` against the protos in `shared/proto` — run `make gen` from
`agency/sdk/js`, or `buf generate` from `shared`. Never hand-edit `src/gen`.

## Auth

All calls use basic HTTP auth with an agency **user credential**. The agency
seeds an initial set at startup (`CHERRYPI_AGENCY_CREDENTIALS`, in memory only);
new pairs are created through `createUser` and are persisted. Agent tunnel
credentials are separate and never used by the SDK.

```ts
import { AgencyClient } from "@cherrypi/sdk";

const agency = new AgencyClient({
  baseUrl: "https://cherrypi.vladfaust.com",
  username: "gitea",
  password: process.env.AGENCY_PASSWORD!,
});
```

A server with no user credentials still starts, but no client can
authenticate.

## Agents

```ts
const agents = await agency.listAgents();
for (const agent of agents) {
  console.log(agent.displayName, agent.status === AgentStatus.ONLINE);
}

// Register an agent; the password is returned exactly once.
const { agent, credentialUsername, credentialPassword } =
  await agency.createAgent({
    username: "builder-1",
    displayName: "Builder",
    prompt: "Prefer small, reviewable diffs.", // optional agent prompt
  });

await agency.updateAgent(agent.id, {
  displayName: "Builder 2",
  description: "",
});
await agency.deleteAgent(agent.id);

// Tunnel credentials (rotation / a second machine).
const creds = await agency.listAgentCredentials(agent.id);
const { credential, password } = await agency.createAgentCredential(
  agent.id,
  "",
);
await agency.revokeAgentCredential(agent.id, credential.username);

// Session cleanup (the agent settings' Cleanup tab). The list is an
// agency-side read; the (slow) per-session workspace sizes load separately.
const { sessions, online } = await agency.listAgentSessions(agent.id);
const { tasks: sizes, totalBytes } = await agency.getDiskUsage(agent.id);
const results = await agency.deleteAgentSessions(
  agent.id,
  sessions.filter((s) => !s.live).map((s) => s.taskId),
);
```

- `listAgents() → Agent[]` — the fleet, sorted by display name. `Agent.status`
  is derived from the tunnel (`ONLINE`/`OFFLINE`).
- `getAgent(id) → Agent`.
- `createAgent({ username, displayName?, description?, prompt? }) → { agent,
credentialUsername, credentialPassword }` — the password is shown once.
  `prompt` stores the agent scope's first prompt version.
- `updateAgent(id, { displayName, description }) → Agent`.
- `deleteAgent(id) → void` — removes the agent and its credentials.
- `listAgentCredentials(agentId)`, `createAgentCredential(agentId, username?)`,
  `revokeAgentCredential(agentId, username)`.
- `listAgentSessions(agentId) → { sessions, online }` — the agent's started Pi
  sessions (tasks that still own one), a fast agency-side read. `online`
  reflects the agent's tunnel, so deletion is possible whenever it is true.
- `getDiskUsage(agentId) → { tasks, totalBytes }` — the agent's per-session
  workspace sizes on disk, keyed by `taskId` (the agent-side session id, i.e.
  `AgentSession.sessionId`). Walking every workspace can be slow on a large
  agent, so it is a separate call.
- `deleteAgentSessions(agentId, taskIds) → AgentSessionDeleteResult[]` — removes
  each task's agent-side Pi session and workspace and marks the task
  session-deleted (its history is gone); the agency-side task is kept. Running
  and input tasks are rejected. Each task reports `deleted` or `error`.

## Users

Agency user (basic-auth) credentials.

```ts
const users = await agency.listUsers();
const user = await agency.createUser("alice", "hunter2");
await agency.revokeUser("alice");

const me = await agency.whoAmI();
```

## Projects

Projects are agency-wide groupings of tasks. A task is created _within_ a
project (`createTask({ projectId })`) and never moves between projects. The
project's slug is set once at creation and is its route handle.

```ts
const projects = await agency.listProjects();
const project = await agency.createProject({
  name: "Website",
  // slug is optional; derived from the name when omitted, then immutable
  // the task prefix is optional too; defaults to the uppercased slug
  description: "The marketing site",
});
await agency.updateProject(project.id, {
  name: "Website v2",
  description: "",
  // changing the prefix renames the project's existing tasks
  prefix: "WEB",
});

// Archiving a project hides it and its tasks from the all-projects list.
const { project: archived, offlineAgents } = await agency.archiveProject(
  project.id,
);
// Unarchiving leaves the tasks archived.
await agency.unarchiveProject(project.id);

// Deletion fails while any task still belongs to the project.
await agency.deleteProject(project.id);
```

- `listProjects() → Project[]`, `getProject(idOrSlug) → Project`.
- `createProject({ name, slug?, prefix?, description? }) → Project`.
- `updateProject(id, { name, description, prefix? }) → Project` — the slug is
  immutable; a changed prefix re-renders the project tasks' ids without
  touching their references.
- `archiveProject(id) → { project, offlineAgents }` — archives the project and
  its tasks; `offlineAgents` are the agents left un-archived until retried.
- `unarchiveProject(id) → Project` — clears the project flag only.
- `deleteProject(id) → void` — fails with `FailedPrecondition` while tasks
  remain.

### Membership

A project's agents are its **members**, and only a member agent can create a
task in the project (the agency rejects a non-member with
`FailedPrecondition`). New agents start with no projects.

```ts
const members = await agency.listProjectAgents(project.id); // ProjectAgent[]
const projects = await agency.listAgentProjects(agentId); // Project[]
const memberships = await agency.listAgentMemberships(agentId); // AgentProjectMembership[]

await agency.addProjectAgent(project.id, agentId);
// Permissions gate the agent's bundled task tools; new memberships are
// list-only. Pass all three flags to replace them.
await agency.updateProjectAgent(project.id, agentId, {
  canListTasks: true,
  canCreateTasks: true,
  canUpdateTasks: false,
});
// Removal deletes the agent's tasks in the project first and fails while
// the agent is offline.
await agency.removeProjectAgent(project.id, agentId);
```

- `listProjectAgents(projectId) → ProjectAgent[]` — `{ agentId, taskCount,
canListTasks, canCreateTasks, canUpdateTasks }`.
- `listAgentProjects(agentId) → Project[]`.
- `listAgentMemberships(agentId) → AgentProjectMembership[]` — one entry per
  member project with the agent's permissions in it.
- `addProjectAgent(projectId, agentId) → ProjectAgent` — idempotent; new
  memberships start list-only.
- `updateProjectAgent(projectId, agentId, permissions) → ProjectAgent`.
- `removeProjectAgent(projectId, agentId) → void` — deletes the agent's
  tasks in the project (never archives them), then removes the membership.

## Prompts

Six versioned system-prompt scopes are injected into a task's system
prompt in order — organization, project, the owning teams' general and
per-project prompts, agent, and agent-project. Every write appends a version only
when the content changed; reads always return the latest, and the task pins the
versions it used (one pin per owning team), so edits never affect tasks created
earlier.

```ts
import { PromptScope } from "@cherrypi/sdk";

// Organization-wide (agency settings).
await agency.setPrompt(PromptScope.ORG, "Be concise and factual.");

// Per agent (agent settings).
await agency.setPrompt(PromptScope.AGENT, "Prefer TypeScript.", {
  agentId,
});

// Per project (project settings).
await agency.setPrompt(PromptScope.PROJECT, "Match the existing style.", {
  projectId,
});

// Per team (team settings); a team-project prompt adds { teamId, projectId }.
await agency.setPrompt(PromptScope.TEAM, "Prefer the team runbook.", {
  teamId,
});

// Per agent × project (agent settings).
await agency.setPrompt(
  PromptScope.AGENT_PROJECT,
  "Run the website test suite before finishing.",
  { agentId, projectId },
);

const org = await agency.getPrompt(PromptScope.ORG);
const perProject = await agency.listAgentProjectPrompts(agentId);
```

- `getPrompt(scope, { agentId?, teamId?, projectId? }) → Prompt` — `content` is empty
  and `version` 0 when the prompt was never set.
- `setPrompt(scope, content, { agentId?, teamId?, projectId? }) → Prompt`.
- `listAgentProjectPrompts(agentId) → AgentProjectPrompt[]` — one entry per
  project with the agent's latest agent-project prompt.

## Settings

```ts
const settings = await agency.getSettings();
await agency.updateSettings({
  agencyName: "CherryPi",
  primaryColor: "#e11d48",
});
```

`agencyName` is trimmed; empty falls back to "CherryPi". `primaryColor` is a
lowercase `#rrggbb`; empty uses the built-in default.

## Tasks

A task is CherryPi's unit of work: a prompt plus its project, assignee, tags,
priority, model, and lifecycle state. The agency owns the task and its
scheduler; every call below resolves the agent and Pi session from the task
id, so clients never track them. Tasks start **asynchronously** — a queued
task is picked up by the scheduler and bound to an agent, so the task returned
by `createTask` may still be `QUEUED`. Poll `getTask` (or watch it) until the
state is `RUNNING` before using the runtime calls.

A task's `id` is **rendered** from its project's prefix and its per-project
`seq` (`WEB-3`). The stable identity is `projectId` + `seq`: editing a
project's task prefix only changes the rendered `id`, and reference, tag,
link and inbox references stay attached to the task. Prefer `projectId` and
`seq` when you store a task reference yourself; the `id` string is a display
value.

```ts
import { TaskPriority, TaskState } from "@cherrypi/sdk";

// Create a queued task (the scheduler assigns an agent). Pass assigneeAgentId
// to pin one, or draft: true to keep it editable.
const task = await agency.createTask({
  prompt: "Fix the failing test",
  name: "Fix CI", // optional; may auto-title instead
  projectId: project.id, // required; every task belongs to a project
  assigneeAgentId: agent.id, // optional; must be a project member
  tagIds: [bugTag.id], // optional
  priority: TaskPriority.HIGH, // default NORMAL
  interactive: true, // surface extension dialogs
  workspace: "/srv/checkouts/repo-x", // optional; defaults to an assigned dir
  // Optional model/thinking selection; omit for Pi's configured defaults.
  modelProvider: "anthropic",
  modelId: "claude-sonnet-4-5",
  thinkingLevel: "high",
  autoTitle: false,
  draft: false,
});

// Draft-only edits; queue it when ready.
const draft = await agency.createTask({
  prompt: "draft",
  projectId: project.id, // required
  draft: true,
});
await agency.updateTask(draft.id, {
  prompt: "draft v2",
  projectId: project.id, // required on update too
  assigneeAgentId: agent.id,
});
await agency.queueTask(draft.id);
await agency.moveTaskToDraft(draft.id); // stops a running task, keeps its session

// Lifecycle. A run settles automatically; close out and archive when done.
await agency.archiveTask(task.id, TaskState.COMPLETED); // close out + archive
await agency.unarchiveTask(task.id);
await agency.reopenTask(task.id, { queue: true }); // back to draft, or queued
await agency.renameTask(task.id, "CI fix");
await agency.deleteTask(task.id); // any state; removes the Pi session

// Filters and lookup.
const tasks = await agency.listTasks({
  projectId: project.id,
  agentId: agent.id,
  state: TaskState.RUNNING,
  tagId: bugTag.id,
  search: "ci",
  archived: false, // true lists the collapsed archive
});
const one = await agency.getTask(task.id); // runtime is enriched while running

// Run-time actions (require a running task).
await agency.addToTask(task.id, "The CI is still red");
await agency.retryTask(task.id); // re-run the last prompt
await agency.stopTask(task.id);
```

- `listTasks({ projectId?, agentId?, state?, tagId?, search?, archived? }) →
Task[]` — the agency-side view, newest first. `archived` selects the
  active list (default) or the archive.
- `getTask(id) → Task` — one task, with `runtime` (`runInProgress`,
  `waitingForInput`, `processRunning`) when it is running on a connected
  agent.
- `createTask(input) → Task` — a full task input (`prompt`, `name`,
  `projectId` (required), `assigneeAgentId`, `tagIds`, `references`
  (`{ taskId, blocking? }[]`), `priority`, `size`, `modelProvider`,
  `modelId`, `thinkingLevel`, `interactive`, `workspace`, `autoTitle`,
  `draft`). `draft: true` creates an editable draft; otherwise the task is
  queued (and parks as `BLOCKED` when any of its blocking `references` is not
  yet `COMPLETED`).
- `updateTask(id, input) → Task` — edits a draft as a wholesale replacement
  (`projectId` is required and the full shape you want must be passed), or
  updates a task's attributes in any state when `input` carries `setPriority`,
  `setSize`, or `setAssigneeAgentId` (draft fields must be omitted; an empty
  `setAssigneeAgentId` clears the pin). A running or input task cannot be
  reassigned.
- `bulkUpdateTask({ ids, attributes }) → { id, task?, error? }[]` — applies
  the same `priority`/`size`/`assigneeAgentId` and/or tag changes
  (`addTagIds`, `removeTagIds`) to many tasks in one call. Every task is
  checked individually, so one that cannot change (a live run, an unknown id,
  an assignee or tag outside its project) reports an `error` while the rest
  still update. `setTaskPriority(id, priority)` and
  `setTaskSize(id, size)` remain for a single attribute.
- `queueTask` / `moveTaskToDraft` / `archiveTask(id, outcome?)` /
  `unarchiveTask` / `reopenTask(id, { queue? })` / `deleteTask` drive the
  lifecycle; `renameTask(id, name) → Task`. Only a `COMPLETED` or `CANCELLED`
  task can be archived: pass `outcome` (`TaskState.COMPLETED` or
  `TaskState.CANCELLED`) to close a non-terminal task out first. A running or
  input task must be stopped or settled first.
- `listTaskReferences(id) → Task[]` / `addTaskReference(id, referencedTaskId,
blocking?) → Task` / `removeTaskReference(id, referencedTaskId) → Task` manage a
  task's references; see [Task references](#task-references).
- `addToTask(id, prompt)` / `stopTask(id)` / `retryTask(id)` act on the
  running task. A prompt sent while a run is in flight is queued by Pi as a
  **steering** message and injected between the run's blocks (after the
  current assistant turn's tool calls, before the next model call); call
  `stopTask` first to interrupt the run and send a fresh prompt instead.
  `clearTaskQueue(id)` discards the messages that are queued but not injected
  yet.
- `watchTask(id, onEvent, { replayHistory?, signal? }) → void` — streams
  normalized `TaskEvent`s. The `pendingMessages` event carries Pi's current
  steering queue, re-emitted as messages are queued and injected, so a console
  can show what is waiting.
- `getTaskHistory(id, { before?, limit? }) → { events, nextBefore }`.
- `getTaskConfig(id) → TaskConfig`; `setTaskModel(id, { provider, modelId })`
  / `setTaskThinkingLevel(id, level) → TaskConfig`.
- `getTaskDefaults(agentId) → TaskConfig` — a new task's starting model/level
  and the pickable models (each with its `reasoning` flag and
  `thinkingLevels`).
- `respondToExtensionUi(id, requestId, response)` — `{ value }` for
  select/input/editor, `{ confirmed }` for confirm, `{ cancelled: true }` to
  decline.

Task states are `DRAFT`, `BLOCKED`, `QUEUED`, `RUNNING`, `INPUT`, `SETTLED`,
`FAILED`, `COMPLETED`, and `CANCELLED` (`TaskState`); priorities are `LOW`,
`NORMAL`, `HIGH`, and `CRITICAL` (`TaskPriority`). A run that finishes settles
automatically (`SETTLED`); a run waiting on an extension dialog shows `INPUT`.
A task with an incomplete blocking reference queues as `BLOCKED` and is
promoted to `QUEUED` when its last blocking reference completes. `COMPLETED`
and `CANCELLED` are
the user's close-out states, and only they can be archived. `running`, `input`,
and `settled` tasks hold the agent's concurrency slot; archiving releases it.

## Task links

A running task publishes labeled links (a PR, an issue, a deployment) with its
agent-side `setTaskUrls` tool. Each call replaces the task's whole list, and the
agency returns it as `task.links` (`{ url, label }[]`) from `getTask` and
`listTasks`. Links are read-only on the client; there is no SDK method to edit
them.

## Task references

A task points at other tasks with **references**. A reference is either
**blocking** (the referencing task is queued only once the referenced task has
reached `COMPLETED`) or purely informational. Only blocking references gate
queuing: a draft with an incomplete one parks as `BLOCKED`, and completing the
last one promotes it to `QUEUED` in the same operation. References are editable
in any state, and the agency rejects a self-reference and a blocking cycle.

A reference is directed one-way. `task.references` lists a task's outgoing
edges (each with `blocking`), while `task.referencedBy` lists the incoming ones.
Adding a blocking reference to a `QUEUED`, `RUNNING`, or `INPUT` task parks it
as `BLOCKED` (a live run is stopped first).

```ts
// `task` now references two prerequisite tasks.
await agency.addTaskReference(task.id, designDoc.id); // informational
await agency.addTaskReference(task.id, schemaMigration.id, true); // blocking

await agency.queueTask(task.id); // -> BLOCKED while the blocking one is open
// ... the prerequisite is run and closed out with TaskState.COMPLETED ...
// task is now QUEUED automatically.

const refs = await agency.listTaskReferences(task.id);
await agency.removeTaskReference(task.id, designDoc.id); // drop one
```

- `listTaskReferences(id) → Task[]` — the tasks `id` references, newest first.
- Every `Task` also carries a lightweight `references` list
  (`{ id, name, state, blocking }[]`) and a `referencedBy` list (the tasks that
  reference it) on `getTask`/`listTasks`, so a table can render each row's
  references without another call.
- `addTaskReference(id, referencedTaskId, blocking = false) → Task` — add or
  update a reference. The call rejects a self-reference and a blocking cycle.
- `removeTaskReference(id, referencedTaskId) → Task` — drop a reference;
  removing a blocked task's last incomplete blocking one promotes it to
  `QUEUED`.

## Task tags

Tags are project-scoped labels used for organization and scheduling limits. A
tag belongs to one project and its name is unique within it; the same name may
exist in other projects. Tags are soft-deleted (archived) so settled tasks
keep rendering them. A tag may be marked `isDefault`: the task composer
preselects it for its project, and the creator can still remove it.

```ts
const tag = await agency.createTag({
  projectId: project.id,
  name: "Bug",
  description: "Something is broken",
  color: "#ef4444",
  isDefault: true, // preselected in the task composer
});
const tags = await agency.listTags({ projectId: project.id }); // active only
const all = await agency.listTags({
  projectId: project.id,
  includeArchived: true,
});
await agency.updateTag(tag.id, {
  name: "Bugfix",
  description: "",
  color: "",
  isDefault: false,
});
await agency.archiveTag(tag.id);
await agency.restoreTag(tag.id);
```

- `listTags({ projectId?, includeArchived? }) → Tag[]` — pass `projectId` to
  scope to one project; omit it to list every project's tags.
- `createTag({ projectId, name, description?, color?, isDefault? }) → Tag` — the
  name is unique within the project, case-insensitively.
- `updateTag(id, { name, description?, color?, isDefault? }) → Tag`.
- `archiveTag(id) → Tag` / `restoreTag(id) → Tag`.

## Agent model defaults

Each agent has an agency-side default model and thinking level. The scheduler
applies them to a fresh task session that does not pin its own selection.
Owners and administrators may change them; every authenticated actor may read
them, which lets the agent page show the selection read-only.

```ts
const defaults = await agency.getAgentDefaults(agent.id);
// { modelProvider: "anthropic", modelId: "claude-sonnet-4-5", … }

await agency.updateAgentDefaults(agent.id, {
  modelProvider: "anthropic",
  modelId: "claude-sonnet-4-5",
  modelName: "Claude Sonnet 4.5",
  thinkingLevel: "medium",
});
```

- `getAgentDefaults(agentId) → AgentDefaults`.
- `updateAgentDefaults(agentId, defaults) → AgentDefaults` — replaces the
  pair. An empty `modelProvider`/`modelId` falls back to Pi's configured
  default; `thinkingLevel` is optional.

## Agent task limits

Each agent has agency-side concurrency caps the scheduler enforces: an overall
cap, a per-project cap, plus optional per-tag and per-project-per-tag caps. A
task consumes every applicable cap, and all of them must have room. The limits
live on the agent registry, so they can be edited while the agent is offline.

Per-tag caps are keyed by **name** and are global across projects: one
`unity-editor` cap gates every project's `#unity-editor` tag, which is what
lets a single shared resource limit concurrency even though each project owns
its own tag row.

```ts
const limits = await agency.getAgentTaskLimits(agent.id);
// { maxConcurrent: 1, maxConcurrentPerProject: 1, tagLimits: [], … }

await agency.updateAgentTaskLimits(agent.id, {
  maxConcurrent: 3,
  maxConcurrentPerProject: 2,
  tagLimits: [{ tagName: "bug", maxConcurrent: 1 }],
  projectLimits: [{ projectId: project.id, maxConcurrent: 1 }],
  projectTagLimits: [
    { projectId: project.id, tagId: tag.id, maxConcurrent: 1 },
  ],
});
```

- `getAgentTaskLimits(agentId) → AgentTaskLimits`.
- `updateAgentTaskLimits(agentId, limits) → AgentTaskLimits` — replaces the
  caps wholesale; omitted overall caps default to 1.

## Usage

```ts
const usage = await agency.getUsage(); // cross-agent aggregate
for (const bucket of usage.tasks) {
  console.log(bucket.agentId, bucket.taskId, bucket.cost);
}

const taskUsage = await agency.getTaskUsage(agentId, task.id); // live overlay
```

- `getUsage(agentId?) → GetUsageResponse` — token/cost totals aggregated from
  the agency's usage table, with per-task buckets and per-day totals.
- `getTaskUsage(agentId, taskId) → TaskUsage` — one task's totals, with the
  agent's live overlay. While the Pi process is running, `contextTokens` and
  `contextWindow` report the session's context-window usage (both zero when
  unknown), so a client can render a context meter.
- `getTaskDiskUsage(agentId, taskId) → bigint` — one task's workspace size on
  disk, in bytes. Unlike {@link getDiskUsage}, it measures only the named
  workspace, so it is cheap enough to refresh while a run is in flight.

## Terminals

Terminals are real PTYs on the agent. Control flows over gRPC; the PTY data
plane is a WebSocket the agency relays.

```ts
import { TerminalKind } from "@cherrypi/sdk";

const { terminal, wsPath } = await agency.createTerminal(
  agentId,
  TerminalKind.ROOT,
  { cols: 80, rows: 24 },
);

const ws = new WebSocket(agency.terminalWebSocketUrl(wsPath));
ws.binaryType = "arraybuffer";
ws.onopen = () => {
  ws.send(JSON.stringify({ type: "resize", cols: 120, rows: 40 }));
  ws.send(JSON.stringify({ type: "input", data: "ls\n" }));
};

const mine = await agency.listTerminals(agentId);
const stats = await agency.getTerminalStats(agentId); // { root, total }
await agency.resizeTerminal(agentId, terminal.id, 120, 40);
await agency.closeAllTerminals(agentId, { kind: TerminalKind.ROOT });
await agency.closeTerminal(agentId, terminal.id);
```

- `createTerminal(agentId, kind, { cols?, rows? }) → { terminal, wsPath }`.
- `listTerminals(agentId) → Terminal[]` — only the caller's terminals.
- `resizeTerminal` / `closeTerminal`; `closeAllTerminals(agentId, { kind? }) → number`.
- `getTerminalStats(agentId) → TerminalStats`.
- `terminalWebSocketUrl(wsPath) → string` — absolute `ws(s)://` URL (the agency
  relays it to the agent over the tunnel).
- `terminalWsPath(agentId, terminalId) → string` — the agency-side path for a
  terminal you already know by id (e.g. one restored from a persisted layout),
  so you can reconnect without re-creating it.

## Desktop

The agent's desktop is its already-running X session, streamed as RFB (noVNC
in the browser). CherryPi never boots a session; the agent attaches `x11vnc` to
the configured display on the first attach. One desktop is shared by every
user of the agent.

```ts
const { available, unavailableReason, desktop, wsPath } =
  await agency.getDesktop(agentId);

if (available) {
  // Hand `wsPath` to noVNC, e.g.
  // new RFB(container, agency.desktopWebSocketUrl(wsPath)).
}
```

noVNC creates its own `<canvas>` inside the target element, so pass a plain
container, not a canvas.

- `getDesktop(agentId) → { available, unavailableReason, desktop?, wsPath }` —
  probes the display and reports its geometry; it starts nothing.
- `desktopWebSocketUrl(wsPath) → string` — absolute `ws(s)://` URL (the agency
  relays it to the agent over the tunnel).
- `desktopWsPath(agentId) → string` — the agency-side relay path
  (`/api/agents/<agentId>/desktop/ws`).

## Webhooks

Agency-level outbound webhooks, optionally filtered to one agent.

```ts
import { AgencyWebhookEvent } from "@cherrypi/sdk";

const hook = await agency.createWebhook(
  "",
  "https://example.com/cherrypi",
  [
    AgencyWebhookEvent.USER_INPUT_REQUESTED,
    AgencyWebhookEvent.AGENT_STATE_CHANGED,
  ],
  process.env.WEBHOOK_SECRET, // optional; sent as X-CherryPi-Secret
);

await agency.listWebhooks();
await agency.updateWebhook(hook.id, {
  agentId: "",
  url: hook.url,
  events: [AgencyWebhookEvent.AGENT_STATE_CHANGED],
});
await agency.testWebhook(hook.id, AgencyWebhookEvent.AGENT_STATE_CHANGED);
await agency.deleteWebhook(hook.id);
```

- `listWebhooks() → Webhook[]` — secrets are never returned (`hasSecret`).
- `createWebhook(agentId, url, events, secret?) → Webhook` — `agentId` empty
  fires for all agents; `events` must be non-empty.
- `updateWebhook(id, { agentId, url, events, secret? }) → Webhook` — omit
  `secret` to keep it, pass `""` to clear it.
- `testWebhook(id, event) → TestWebhookResponse` — posts a sample payload and
  reports the target's HTTP status.

## Notifications

The agency's persistent inbox, fed by agents' attention signals.

```ts
const notifications = await agency.listNotifications();
const unread = await agency.getUnreadCount();
await agency.markNotificationRead(notifications[0].id);
await agency.markTaskNotificationsRead("PROJECT-1");
await agency.markAllNotificationsRead();
await agency.deleteNotification(notifications[0].id);
await agency.deleteAllNotifications();
// Kinds not stored in the inbox (still pushed for live alerts).
const disabled = await agency.getInboxPreferences();
await agency.setInboxPreferences([...disabled, "task_started"]);
```

### Realtime stream

The agency pushes new entries over a WebSocket so the UI can alert without
polling. The first frame is `{ type: "hello", unread }`; later frames are
`{ type: "notification", notification }`.

```ts
const ws = new WebSocket(agency.notificationWebSocketUrl());
ws.onmessage = (event) => {
  const frame: AttentionEvent = JSON.parse(event.data);
  if (frame.type === "notification") {
    console.log(frame.notification.title, frame.notification.body);
  }
};
```

- `notificationWebSocketUrl() → string` — absolute `ws(s)://` URL of the
  stream.
- `notificationWsPath() → string` — the agency-side path
  (`/api/rest/notifications/ws`).

## Options & types

```ts
import {
  AgencyClient,
  AgentStatus,
  AgencyWebhookEvent,
  TerminalKind,
  type AttentionEvent,
} from "@cherrypi/sdk";
```

`AgencyClientOptions` takes `baseUrl`, `username`, `password`, and an optional
`transport`. By default the SDK builds a Connect transport pointed at
`<baseUrl>/api/grpc` with the basic-auth interceptor; supply your own
`transport` to take over transport and auth configuration.

Set `basicAuth: false` for **ambient auth**: the SDK adds no `Authorization`
header and the browser (or an interactive client) supplies the credentials — how
the dashboard reuses its cached basic-auth entry.

Every method also accepts `{ signal }` to cancel the call. The package
re-exports the generated protobuf-ES messages and enums (`Task`, `Tag`,
`TaskState`, `TaskPriority`, `TaskEvent`, `TaskConfig`, `Terminal`,
`TerminalKind`, `Desktop`, `Agent`, …) so consumers never import from `src/gen` directly.

