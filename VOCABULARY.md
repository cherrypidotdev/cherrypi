# Vocabulary

Canonical terms for this system. Link to a term with its anchor, e.g.
`[agent](#agent)`.

## Agency

The central web app (Go server + Vue SPA, `agency/`) deployed to a VPS. It
manages the fleet, users, settings, webhooks, tags, and the notifications inbox,
schedules [tasks](#task) onto [agents](#agent), and routes every task/terminal
call over the [tunnel](#tunnel). It is the only public surface: one SPA, one
login, one API.

## Agent

One long-running worker on its own machine, registered in the agency before it
connects. It runs `cherrypi-agent --agency-url <url> --auth <user>:<pass>`,
shares its identity as a username + display name, and hosts its own
[agent database](#workspace) and tasks. An agent has no web UI of its own. Its
dashboard pages are keyed by its **username** (a stable slug), not its id, and
it belongs to zero or more [projects](#project). Each agent carries its own
[concurrency limits](#concurrency-limit).

## Tunnel

The single long-lived Connect bidi stream an [agent](#agent) opens to the
[agency](#agency). It is a full-duplex byte pipe: the agent runs yamux over it
and serves its HTTP API there, so the agency reuses generated Connect clients
and the agent keeps its HTTP handlers. An agent reports **online** while its
tunnel is connected.

## Agent credential

A username–password pair an [agent](#agent) authenticated its
[tunnel](#tunnel) with before personal access tokens. At least one is generated
when the agent is registered and shown **once**; more can be added for rotation
or a second machine. **Legacy fallback:** the tunnel, the bundled
[agent tools](#agent-tool) and the history sync accept it, but the agent
prefers its [personal access token](#personal-access-token) and uses the
credential only to bootstrap one. Kept separate from
[user credentials](#user-credential).

## User credential

A username—password pair an [actor](#actor) uses to sign in to the
[agency](#agency). In the accounts model the password lives on the actor; the
account may also carry e-mail credentials, TOTP and recovery codes. Seeded from
the environment or created through the admin surface.

## Actor

The shared identity of every human and agent (`actors`): a globally unique
username, display name, description, picture and password. A `users` row adds
the account state (role, status); an `agents` row adds the tunnel surrogate.
Every reference in the agency is an actor id. See
[`docs/auth.md`](docs/auth.md).

## Account session

A browser login. The raw token lives in an httpOnly `cherrypi_session` cookie;
only its SHA-256 is stored. A session slides on use (7 days, or 30 with
"remember me") and carries the login method and whether the second factor was
verified. Mutating requests also require the double-submit `cherrypi_csrf`
cookie echoed in `X-CSRF-Token`.

## Invitation

The only way a [user](#user-credential) joins an installation. The inviter
always reserves the globally unique username; the invited actor is created
up-front with status `invited`. Four kinds (`invitations.kind`): `username`
(one-time generated password, forced reset on first sign-in), `email` (a
single-use link mailed through SMTP, with a copyable fallback), and `oidc` /
`ldap` (provider-scoped; the first matching federated login links the identity).
Pending invitations expire after 7 days and are pruned by the scheduler. See
[`docs/auth.md`](docs/auth.md).

## Team

A named group of [actors](#actor) that is the primary way project access is
granted. The creator becomes the team's first **owner**; other members are
plain **members**. Any member may invite an actor; a user accepts their own
invitation, while an agent's invitation is approved by one of the agent's
[owners](#agent-owner). Only an owner (or an administrator) may edit the team,
invite as an owner or delete it, and a team with members cannot be deleted.
Every mutation is recorded in the team-scoped domain log. An agent joins only
with one of its owners' approval, managed from the agent's Settings **Teams**
tab; agent invitations never appear on the global teams list. See
[`docs/auth.md`](docs/auth.md).

## Agent owner

An actor that may manage an [agent](#agent) (`agent_owners`): edit its identity
and settings, approve its [team](#team) invitations and add or remove owners.
Any actor kind may own an agent, and creating an agent makes its creator the
first owner; an agent with no owner is administrator-only. See
[`docs/auth.md`](docs/auth.md).

## Personal access token

A `pat_…` credential for an [actor](#actor): the [SDK](#sdk), the CLI, and an
[agent](#agent)'s tunnel and [tools](#agent-tool). It carries the actor's
permissions (minus admin capabilities), expires optionally, and is shown only
once; only its SHA-256 hash is stored. An agent's token is cached in
`$CHERRYPI_DIR/agent.token` (0600), which the bundled tools read on every call
so a rotation needs no Pi respawn.

## TOTP

Time-based one-time passwords: an optional second factor on an actor. Each
login method can require it (`auth.totp.{password,magic_link,oidc,oauth2,ldap}`
in org settings); when a federated method requires it, the login pauses on a
short-lived pending token and the SPA asks for the code. Enrolling mints ten
one-time recovery codes.

## Org auth settings

Admin-only toggles in the `settings` table: which sign-in methods are enabled
(`auth.password.enabled`, `auth.magic_link.enabled`), whether a confirmed e-mail
is required (`auth.email_confirmation.required`), the sliding session lifetimes
(`auth.session.{ttl,remember_ttl}_hours`), the per-method TOTP flags, the
member-governance flags (`members.can_invite_users`,
`members.can_create_agents`, `members.can_create_teams`) and the destructive
guards (`admin_only.{project,team}_deletion`). The SMTP configuration
(`smtp.*`, password write-only) lives here too.

## Identity provider

An org-wide OIDC or generic OAuth2 configuration (`identity_providers`,
admin-only): a client id/secret and either a discovery issuer (OIDC) or
explicit authorization/token/userinfo endpoints plus the userinfo JSON paths
(the OAuth2 shim). The sign-in page lists the enabled ones. A generic OAuth2
provider is identified from its userinfo JSON by the configured subject path.

## External credential

A provider identity linked to an [actor](#actor) (`external_credentials`),
keyed by `(provider, subject)`. Federated sign-in is **invite-only**: it
resolves an existing link and never auto-provisions, so an unlinked identity is
rejected. A user links one from the account page (an OIDC/OAuth2 flow in
`mode=link`, or an LDAP bind).

## LDAP credential

An LDAP username linked to an [actor](#actor) (`ldap_credentials`), keyed by
`(provider, username)`. Like an external credential, it is created by linking
from the account page and is what makes an invite-only directory login resolve
to an account.

## Task

CherryPi's unit of work, owned by the [agency](#agency). A task has a prompt, a
[project](#project), an optional **assignee** [agent](#agent), zero or more
[tags](#task-tag), zero or more [links](#task-link), zero or more
[references](#task-reference), a [priority](#priority), and a
[state](#task-state). A task must be created within a project (legacy rows
without one are **orphans**). It
is bound to an agent and a [Pi session](#pi-session) only once it starts. The
lifecycle is `draft → backlog → queued → running → settled`, with `running →
input` while the agent waits on a dialog, `running → failed`, and a queued task
returning to `backlog` (then `draft`). A task **settles automatically** when its
Pi run finishes. `running`, `input`, `settled`, and `failed` tasks hold an
agent's [concurrency slot](#concurrency-limit) (so a retry can reuse the warm
Pi session). A follow-up to any resting task (`settled`, `failed`, `completed`,
or `cancelled`) re-queues it; the scheduler delivers the prompt to the warm
session once the agent is reachable. Closing a task out as `completed` or
`cancelled` keeps it in the
active lists; it leaves them only when **archived**, which also cleans up its
agent-side Pi session and workspace. Surfaced as a [console](#console) plus an
attachable workspace-shell [terminal](#terminal).

A task is owned by one or more of its project's [teams](#team) (its **owning
teams**); its **leads** are the accountable project members, drawn from the
owning teams or the rest of the project. A member of an owning team may prompt
the task, and a lead is shown on it, may prompt it and receives its
notifications. Any project member may **subscribe** to a task's
updates; only leads and subscribers receive its inbox notifications. A new
task defaults its owning teams to the creator's teams that are attached to
the project. See
[`auth.md`](docs/auth.md#task-collaboration).

## Pi session

The actual Pi conversation a [task](#task) runs: a `pi --mode rpc` child process
with its own JSONL session file and `--session-id`. "Session" in CherryPi always
means this Pi-level concept; the CherryPi work item is a [task](#task). The
agency keeps a canonical copy of its JSONL entries so the conversation survives
its [agent](#agent); cleaning up a session deletes only the agent's workspace
and JSONL, and a later follow-up replays the stored conversation into a fresh
Pi session. See [`docs/session-history.md`](docs/session-history.md).

## Task state

One of `draft` (editable, never scheduled), `backlog` (very likely to run, but
not queued yet; indigo), `queued` (waiting for an eligible agent with a free
[slot](#concurrency-limit); a queued task with incomplete blocking references is
reported as **blocked**, shown with an octagon-pause counter, and skipped by the
scheduler), `running` (assigned to an agent with a
[Pi session](#pi-session); a run is in flight), `input` (the run is
blocked on an extension dialog and needs the user's answer; cyan), `settled`
(the Pi run finished; the task is idle but still holds its slot), `failed` (a
run errored; the warm session still holds its slot for a retry or follow-up),
`completed` (closed out as done), or `cancelled` (closed out as abandoned).
`running`, `input`, `settled`, and `failed` hold an agent
[concurrency slot](#concurrency-limit); the rest do not. The ladder is strict:
`draft → backlog → queued`, `queued → backlog`, and `backlog → draft`; only a
backlog task (or a creation request) is queued. A started task returned to
backlog or draft keeps its console history and shows its pending follow-up (the
next message a re-queue delivers) as the last, editable message.

## Task reference

A directed edge from one [task](#task) to another: the **referencing** task
points at a **referenced** task. An edge has a **kind**: **blocking**,
**informational**, or **supersedes**. Queuing is gated only by blocking
references: a queued task waits until **every** blocking reference has reached
`completed`. A queued task with an incomplete blocking reference stays queued
but is reported as **blocked** (and skipped by the scheduler); completing the
last one simply clears the flag, with no state transition.

References are editable in **any** state. They are directed one-way, so the
referencing task's **References** list shows its outgoing edges and the
referenced task's **Referenced by** list shows the incoming ones; the incoming
side is read-only and is managed from the referencing task. A cross-project
reference must follow the project tree: a task may reference tasks in its own
project, any [descendant](#project), or a sibling (a project with the same
parent, where all roots share the null parent). A parent, an unrelated cousin
or any other shape is refused, and a hierarchy change that would invalidate an
existing edge is rejected. Only blocking edges gate queuing, and the blocking
graph is kept acyclic. A **supersedes**
edge cancels the referenced task when it is in `draft`, `backlog`, `queued`, or
`failed`; any other state is left alone. Deleting a task removes every edge that
touches it.

## Task archive

A per-[task](#task) boolean (not a [state](#task-state)) that hides it from the
active lists. Archiving stops a live run and removes the agent-side
[Pi session](#pi-session) and workspace; the agency keeps the conversation,
attachments, and artifacts, so the task page stays readable. If the agent is
offline the archive still succeeds and the cleanup is left for a manual retry,
which the archived row highlights.

## Task merge

Merging several [tasks](#task) creates a new draft task whose prompt
concatenates theirs, separated by a `---` rule, whose tags are unioned, whose
size is the largest, and which folds in their attachments, links, and
references. Every mergee must
share a project and an assignee, and be in `draft`, `backlog`, `queued`, or
`failed`. The merged task adds **supersedes** [references](#task-reference) to
each mergee, cancelling it.

## Priority

A [task](#task)'s queue ranking: low, normal (default), or high. The
[scheduler](#scheduler) drains higher priority first, breaking ties by creation
time.

## Task size

A [task](#task)'s manually set scope estimate: small, medium (the default on a
new task), big, or epic. It is **informational only** — the
[scheduler](#scheduler) ignores it — and lives purely to help a human triage the
board, so it changes in any state and shows as a colored badge
(green/yellow/orange/red) in the task page header, task table and logs, as a
picker in the details drawer, and on the create form.

## Task tag

A project-scoped label (`name`, `description`, `color`, `is_default`) attached to
zero or more [tasks](#task). A default tag is pre-selected in the task composer
for its project; the creator may still remove it from one task. A tag's name is unique within its [project](#project); the same
name may exist in other projects. Tags are soft-deleted (archived) so settled
tasks keep rendering them, and they double as a scheduling dimension: an
agent's [concurrency limits](#concurrency-limit) can set a cap per tag **name**,
which is global across projects, so one `unity-editor` cap gates that name
everywhere. A task's tags can only change while it is a draft, and only tags of
its own project apply.

## Task link

A labeled URL (`url`, `label`) a running [task](#task) publishes on itself — a
PR, an issue, a deployment. The agent attaches them with its bundled
`setTaskUrls` tool, whose every call **replaces** the task's whole list; the
agency stores them and renders them in the task list and detail. Unlike
[tags](#task-tag), links are read-only for users and may change in any state.

## Attachment

A stored file or image. The agency keeps one S3-backed blob registry
(`attachments`: rowid, storage key, filename, content type, size, audit),
while ownership lives in link tables: `task_attachments` (a task's tray),
`message_attachments` (a user message's files), and a
`picture_attachment_id` on a [project](#project) or [agent](#agent). Objects
are keyed by rowid with the original extension preserved
(`attachments/<rowid><ext>`). Clients and agents read the bytes only through
the agency's `/api/rest/attachments` routes; the credentials never leave the
server.

## Artifact

An **output** a running [task](#task) produces, modeled on the
[A2A protocol](https://a2a-protocol.org/latest/specification/): a named
`artifact_id` unique within the task, a name/description, and one or more
**parts** that are text, structured data (JSON), or a file. A file part is
backed by an [attachment](#attachment) (raw bytes) or a bare URL. Artifacts
stream: `streaming` is true while the agent is still appending parts. The agent
publishes them with its bundled `setTaskArtifacts` Pi tool (which uploads file
parts and emits a `task_artifact` event, like `setTaskUrls` does for links),
and the console renders them in the task's **Artifacts** tab. Artifacts are
distinct from messages: outputs are not delivered in the reply text. The
single-file `uploadAttachment` tool is different: it publishes transient,
per-message context into the task's Attachments tray (for example a current
screenshot or log) and returns a URL for the reply. See
[`docs/task-artifacts.md`](docs/task-artifacts.md).

## Picture

An uploaded image that replaces a [project's](#project) or [agent's](#agent)
jdenticon avatar. It is an [attachment](#attachment) referenced by the
owner's `picture_attachment_id`; uploading replaces the previous one.

## Scheduler

The agency's in-process component that drains `queued` [tasks](#task) onto
eligible [agents](#agent): it picks the pinned assignee or a member agent that
has every applicable [concurrency cap](#concurrency-limit) free, atomically
claims and starts the task, and reconsiders on task changes, an agent
connecting, and a periodic tick. The same tick reconciles in-flight runs,
materializing `input` while the agent waits on a dialog and `settled` once the
run finishes. There is no queue table.

## Concurrency limit

A per-[agent](#agent) cap on simultaneously active [tasks](#task): those in
`running` or `settled` state that are not archived. Four kinds compose and all
must have room: the overall total, per [project](#project), per
[task tag](#task-tag) name (global across projects), and per project × tag.
Limits are stored agency-side so the [scheduler](#scheduler) can gate even
offline agents.

## Project

An agency-wide grouping of [tasks](#task) that carries a [prompt](#prompt). A
project has a name, an **immutable** lower-kebab slug (its route handle), a
unique **task prefix** (its slug uppercased by default, editable in project
settings), a description, and a jdenticon logo, plus a set of member
[agents](#agent). A task's public id reads `<prefix>-<seq>`, numbered per
project (`seq` is the stable per-project sequence). The id is rendered from
the project's current prefix: changing the prefix renames the display id of its
tasks, while references, tags, links and inbox references stay attached to
the task's internal rowid. Every task is created
_within_ a project and never moves between projects; the agency refuses to
create or edit a task without one, so only legacy rows are **orphans**. Access
is role-based (see [project role](#project-role)): an actor's effective role is
the strongest of a [direct project grant](#direct-project-grant) and the roles
of the [teams](#team) they belong to that are attached to the project. Removing
(or leaving) a direct grant deletes that agent's tasks there when it is an
agent — deletion, never archive. Projects are archivable: archiving a project
hides its tasks in lists without changing their states, and un-archiving
reveals them again. A project can only be deleted once every one of its tasks
has been deleted and it has no child projects.

Projects form a strict **tree**: a project has at most one parent and any
number of children, and cycles are rejected. A parent link is consented to by
the child side (see [parent project invitation](#parent-project-invitation)).
Viewing is inherited: anyone with viewer-or-better access to a parent gets the
**viewer** role on every descendant, never member or owner. An explicit grant
on the child wins when it is stronger.

## Project role

An actor's level of access to a [project](#project), ordered **owner** >
**member** > **viewer**. A **viewer** may view and comment (list tasks only); a
**member** may prompt (create tasks) and edit; an **owner** may also manage
access. The role is derived from the strongest of a
[direct project grant](#direct-project-grant), the attached [teams](#team) the
actor belongs to, and the inherited **viewer** role from any
[ancestor project](#project) (capped at viewer), except that an administrator
holds owner everywhere. The retired per-agent `can_list_tasks` / `can_create_tasks` /
`can_update_tasks` flags are derived from the role for the
[agent tools](#agent-tool); an agent may always list and update the tasks
assigned to itself, regardless of role.

## Direct project grant

An exception to team-based project access: a row in
`project_actor_memberships` that gives one [actor](#actor) a
[project role](#project-role) on one [project](#project) without the actor
belonging to an attached [team](#team). A project owner adds, re-roles and
removes direct grants. Attaching a team is instead consented to through a
**project-team invitation** approved by one of the invitee team's owners.

## Parent project invitation

A consented link that makes one [project](#project) a child of another. The
parent's owner creates the invitation; an owner of the child (a direct grant or
a member of an owner-role team) accepts or declines it. When both projects share
an owner-role [team](#team), a member of that team may link them directly
without an invitation; administrators skip both steps. A pending invitation
reserves the child, since a project can have only one parent. Only the parent's
owner (or an administrator) may detach the link afterwards. Every link, detach
and invitation writes to both projects' domain logs.

## Agent tool

A model-callable operation the agency exposes to a running
[Pi session](#pi-session) through a bundled Pi extension: `listTasks`,
`getTask`, `getTaskHistory`, `createTask`, `updateTask`, and `listProjects`.
`listTasks` and `getTask` return task metadata only; the conversation is kept
out of both and read in newest-page-first pages with
`getTaskHistory(limit, cursor)`. The extension calls the agency's agent-tool
REST surface with the agent's
[personal access token](#personal-access-token), read from the token file on
every call; the agency resolves the agent actor from the token and enforces the
[task permissions](#task-permission) and the own-task exception. `listProjects`
reports the projects the caller has joined, each with its members (name,
username, description, permissions, and both their **agent id** and **actor
id**). An assignee is addressed by **actor id** (`assignee_actor_id`); the
pre-migration `assignee_agent_id`, which carries an agent id, is still accepted
as a deprecated alias, with `assignee_actor_id` winning when both are present.

## Prompt

A versioned block of instructions injected into a [task](#task)'s system prompt.
Six scopes exist, injected in order: **organization** (agency settings),
**project** (project settings), the own [teams'](#team) **team** prompts (team
settings), their **team-project** prompts (team settings), **agent** (agent
settings and registration), and **agent-project** (per agent × project, in
agent settings). Writes are append-only — a change adds a version, the UI always shows
the latest — and a task pins the exact versions it started with (including one
pin per owning team), so later edits never affect it.

## Steering message

A prompt a user composes while a [task](#task)'s run is in flight. Pi queues it
and injects it between the run's **blocks** (after the current assistant turn's
tool calls, before the next model call), so it can redirect the run without
interrupting it. The console surfaces the pending queue and can clear it. A
steering message sent when no run is in flight is a normal follow-up.

## Console

The rendered view of a [task](#task)'s [Pi session](#pi-session) — assistant
text, thinking, tool calls/results, stderr, and extension dialogs —
reconstructed from the `WatchTask` event stream (RPC mode has no TTY). Distinct
from a [terminal](#terminal), which is a real PTY.

## Terminal

A PTY exposed in the browser, rendered by xterm.js: a **root terminal**, a
shell on the agent machine. Each is **owned** by the
[user credential](#user-credential) that created it. The agent owns the
terminal list — every browser renders the same tabs (one per terminal) and
polls for changes — and a terminal may have several consumers at once: the
**controller** owns the keyboard and PTY size, while **observers** are
read-only, stream the same output, and can take control. The agency relays the
PTY WebSocket to the agent over the [tunnel](#tunnel).

## Desktop

The agent machine's **already-running** X session, exposed to the browser as an
RFB (noVNC) stream and reached from the agent header's **RDP** button. CherryPi
does not boot a session: the operator provides one (the host user's Xfce/X11
session, or the host's X socket mounted into Docker) and the agent attaches
`x11vnc` to `$DISPLAY` on the first attach. One desktop is
shared by every [user credential](#user-credential) of an agent — unlike
[terminals](#terminal), which are per-user. The agency relays the RFB WebSocket
over the [tunnel](#tunnel).

## Workspace

The [agent](#agent)'s filesystem area where [tasks](#task) do their work. Each
task defaults to its own directory (server-assigned, overridable by the client
via `CreateTask`), clones what it needs, and shares results as artifacts.

## SDK

`agency/sdk/js` (`@cherrypi/sdk`): the typed client (`AgencyClient`) over the
[agency](#agency)'s gRPC surface — agents (including task limits), users,
settings, notifications, webhooks, projects, prompts, tags, and the task
lifecycle plus its runtime actions. The dashboard uses it.

## Dashboard

The agency's Vue SPA. An agency sidebar lists every [agent](#agent) (jdenticon
avatars and presence) and links to the fleet-wide Tasks, Usage, Inbox, and
Settings pages. Selecting an agent reveals a second sidebar with its tasks,
[terminals](#terminal), usage, and per-agent settings; webhooks and user
credentials live under the agency Settings page, while
[task tags](#task-tag) live under each project's settings.

## Usage

Cost accounting: token counts and estimated spend per [task](#task). Agents push
per-run deltas to the [agency](#agency), which aggregates them across the fleet
(`agent_usage`); a task's live overlay comes from its agent.

## Webhook

An outbound JSON notification the [agency](#agency) POSTs to a configured URL
when events arrive from agents — `user_input_requested` or `agent_state_changed`.
Webhooks may be filtered to one agent; payloads include `agent_id`. Delivery is
fire-and-forget.

## Notification

A persistent inbox entry in the [agency](#agency), created from an agent's
attention signals (input requested, failures). Unlike [webhooks](#webhook), the
inbox survives long absences with nothing configured.

## Host

The machine (VPS) that runs the [agency](#agency), and each machine that runs an
[agent](#agent). The project ships the binaries and Docker images; the host
provides GPU drivers, user isolation, and networking — see
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
