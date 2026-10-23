# Legacy & tech debt register

CherryPi carries a few deliberate compatibility shims and known shortcuts. This
file is the single list of them: what the debt is, why it exists, and what
retires it. Add an entry when you introduce a shim and delete the entry when
the upgrade path lands — a stale entry here is itself debt.

## Policy: when an alias is allowed

Renames are normally lockstep and wire-breaking: the `Session` → `Task` rename
removed the old names outright with **no deprecated aliases** (see
[`tasks.md`](tasks.md)). The exception is a surface whose caller cannot upgrade
in lockstep with the agency. The bundled agent binary embeds its Pi extensions
and is deployed independently, so a field renamed on the agent-tool REST
surface keeps an agency-side alias until every agent carries the new extension
(the `agent-bump` rollout).

## Wire & protocol shims

| Debt | Where | Why | Retire when |
| --- | --- | --- | --- |
| `assignee_agent_id` alias | `agency/server/internal/agenttools/handler.go`; extension `agent/internal/session/extension_assets/agent_tools.ts` | A pre-rename agent binary still sends the old field. It carries an `agents.id`, so the agency resolves it to its actor id through the registry; `assignee_actor_id` wins when both are present. | Every agent has been rebuilt with the extension that sends `assignee_actor_id`. |
| `draft` / `blocking` aliases | `agency/server/internal/agenttools/handler.go`; extension `agent/internal/session/extension_assets/agent_tools.ts` | The CHPI-145 task rework made the create `state` required (`draft`/`backlog`) and replaced the reference `blocking` boolean with a `kind`. An un-updated agent binary may still send `draft: true` or `blocking: true`; the agency maps them to `draft` and the `blocking` kind, while `state`/`kind` win when present. A legacy `draft: false` (queue immediately) is still rejected, since agents may no longer queue. | Every agent has been rebuilt with the extension that sends `state` and `kind`. |
| `TASK_STATUS_CANCELLED` / `StatusCancelled` | `shared/proto/agent/v1/task.proto`, `agent/internal/session/event.go` | Cancelled runs were folded into `DONE`. Persisted rows may still hold value 4; clients receive `DONE` and the value is never emitted. | A migration rewrites value 4 and the next wire-breaking agent protocol release. |
| `ArchiveProjectResponse.offline_agents` | `shared/proto/agency/v1/projects.proto` | Archiving no longer calls agents, so the field is always empty. It is kept so older clients still decode the response. | The next wire-breaking agency protocol release. |
| Reserved proto tags | `shared/proto/agency/v1/task.proto`, `shared/proto/agent/v1/{task,terminal}.proto`, `shared/proto/agency/v1/agents.proto` | Removed features (task archive, terminal and disk-usage fields). The numbers are reserved so they are never reused with a different meaning. | Never — permanent wire hygiene, not debt. |
| Terminal frames capped at 32 KiB | `agent/internal/terminal/manager.go` (`maxOutputFrame`) | Coalesced output and the scrollback replay must fit `coder/websocket`'s default 32 KiB read limit on an agency relay that predates the raised limit; a larger frame would make an older agency drop the terminal. | Every deployed agency raises the relay read limit (it now does); the cap can then grow to cut per-frame overhead further. |
| Legacy task rows | `agency/server/internal/tasks/tasks.go` (`Enrich`), `agency/server/cmd/agency/grpc/task/task.go` (`enrichNames`, `taskRef`), `agency/frontend/src/lib/task.ts` | Tasks backfilled from the retired `session_context` have no project (orphans) and an empty name/prompt until enriched from the owning agent. The UI falls back to the task id. | A migration removes the orphans and enrichment has visited every remaining legacy row. |
| Legacy agent session with no `(project, seq)` reference | `agent/internal/session/session.go`, `agent/internal/session/manager.go`, `shared/proto/agent/v1/task.proto` | A session created before the agent persisted its stable reference only has the display id, which a project-prefix rename can stale. The agency re-sends the reference so the agent can backfill it; until then the display id is used. | All sessions carry a persisted reference (the backfill is opportunistic). |
| `tasks.participation` column + teamless tasks | `agency/server/sql/sqlite/migrations/042_task_participation.sql`, `043_task_ownership.sql`; `agency/server/internal/tasks/tasks.go` | Migration 043 superseded the participation level (`project`/`teams`/`assignee`) with owning teams, leads and subscriptions. The `tasks.participation` column is left in place because its inline CHECK blocks SQLite's `DROP COLUMN`; the code no longer reads it. A task created in a project with no teams is left teamless and falls back to the project prompt rule. | A table rebuild drops `tasks.participation`, and every project has at least one team (then `resolveTaskTeams` can require one). |
| Raw `{sendingActor, messageBody}` envelope in Pi's pending queue | `agent/cmd/agent/grpc/task/task.go` (`EventQueue`), `agency/frontend/src/lib/console.ts` (`parsePromptEnvelope` / `queuedMessageParts`) | `UserMessage` is parsed server-side into sender/body/attachments, but `PendingMessages.steering` is still `repeated string`, so Pi's echoed envelope reaches the console raw and the console decodes it. A legacy plain steering string falls through untouched. | The next agent-protocol release carries a structured pending prompt (or the agency parses it at the source), so no consumer sees a raw envelope in the queue. |

## Database dialect debt

| Debt | Where | Why | Retire when |
| --- | --- | --- | --- |
| PostgreSQL schema starts at the consolidated baseline | `agency/server/sql/pg/migrations/044_baseline.sql` | PostgreSQL support arrived after 44 SQLite migrations, many of which are SQLite-only table rebuilds. There is no historical PostgreSQL deployment to replay, so the baseline is the schema those migrations produce. New schema changes are authored in both trees from 045 on. | Never — the baseline is the starting point for a fresh PostgreSQL install. |
| Two query trees | `agency/server/sql/sqlite/queries`, `agency/server/sql/pg/queries` | sqlc generates one Go package per engine, and the engines need different placeholder/function spellings and sometimes different SQL. The trees are allowed to diverge; only the Store interface is the contract. | Never — separate trees are the point. |
| No foreign keys in PostgreSQL | `agency/server/sql/pg/migrations/044_baseline.sql` | SQLite runs with `PRAGMA foreign_keys=OFF`, so the application has always deleted child rows itself. Enforcing the same DDL on PostgreSQL only would turn tolerated orphans into runtime failures. | SQLite enables foreign keys, orphan rows are cleaned, and the test suite passes with them on. |
| Many Store methods still mirror a single query | `agency/server/internal/store/{sqlite,pg}_methods.go` | `SqliteStore` and `PgStore` are separate implementations, but most of their methods are still one-query wrappers. The high-level cases (`RegisterAgent`, `TransitionTask`) show the target shape. | CHPI-184 redesigns the Store API around aggregates and removes the pass-through files. |

## Identity & auth debt

| Debt | Where | Why / state | Retire when |
| --- | --- | --- | --- |
| Legacy agent tunnel credential (`agent_credentials`) accepted as a Basic fallback | `agency/server/cmd/agency/auth.go` (`resolveAgentRequest`), `agency/server/cmd/agency/tunnel.go` | Deployed agents only hold a generated `user:password`; the phase-1.5 PAT rollout must not require a synchronised cutover, so the tunnel, agent tools, attachments and history sync still accept it. The agent exchanges it for a PAT on first boot. | Every agent carries a PAT and `CHERRYPI_AGENT_AUTH` / `agent_credentials` are removed (the user-facing Basic removal is tracked by CHPI-119). |
| `POST /api/rest/agent-tools/token` self-bootstrap/rotation | `agency/server/internal/agenttools/handler.go`, `agent/cmd/agent/appcontext/appcontext.go` | An agent that only holds the legacy credential needs a way to obtain — and later rotate — a PAT; the agency stores only hashes, so it cannot return an existing one, and a generated token cannot be copied through compose. The route revokes the actor's previous `agent-bootstrap` token. | Operators provision PATs out of band (`CHERRYPI_AGENT_TOKEN` or the UI) and the legacy credential is retired. |
| Flat `users(username, password_hash)` + ambient HTTP Basic | `agency/server/internal/auth`, `agency/server/cmd/agency/auth.go`, migration `001_init.sql` | The original single-namespace credential store: one dashboard password, no sessions, roles, TOTP or CSRF, and any valid credential can call every method. | The accounts phase replaces it with `actors` + credentials; see [`auth.md`](auth.md). |
| Legacy agent ids (TEXT UUIDs) | Migration `agency/server/sql/sqlite/migrations/025_actors.sql` | Agent identity moved to serial `actors.id` with an `agents.id` surrogate. The migration down path restores the decimal TEXT ids, so the round trip is only semantically faithful for rows that predate the change. | No further down migrations are expected; in practice treat 025 as one-way. |
| Coarse `task_logs.actor` (`"user" \| "agent" \| "system"`) | `agency/server/sql/sqlite/migrations/018_task_logs.sql` | Task logs predate actor ids. Migration 042 adds `task_logs.actor_id` (and `task_messages.actor_id`), so a row now names the acting actor; the coarse `actor` column is kept for legacy rows and as the kind label. | Done — see [`auth.md`](auth.md) phase 6; the column stays for readability. |
| Pre-phase-6 `--- attached: … ---` attachment trailer | `agency/frontend/src/lib/attachments.ts` (`parseAttachmentTrailer` / `stripAttachmentTrailer`), the fallback in `lib/console.ts` (`userMessageParts`) | Prompts written before phase 6 embed the attachment `id`s in a text trailer, and the console still parses it. New prompts carry structured `messageBody.attachments` instead, and the console reads `UserMessage.attachments`. The trailer survives only in stored Pi history, pending prompts, and `task_messages`. | A check proves no `task_session_entries.raw_json` or `task_messages.text` contains `--- attached: `; then the parser and its specs are deleted. Tracked by **CHPI-161** (legacy). |
| Per-agent project task flags (`project_agents.can_list_tasks` / `can_create_tasks` / `can_update_tasks`) | `agency/server/sql/sqlite/migrations/039_project_access.sql` | Phase 5 replaced the flags with project roles; the migration derives a role from the old flags and drops the table. The gRPC `UpdateProjectAgent` RPC is kept for wire compatibility and maps its flags back onto a role (create/update -> `member`, otherwise `viewer`), and `AddProjectAgent` still defaults to list-only. | A wire-breaking agency protocol release moves the RPC to an explicit role and removes the flags. |
| Separate `agent_domain_logs` instead of extending the operational `agent_logs` | `agency/server/sql/sqlite/migrations/044_domain_logs.sql`, `agency/server/internal/audit` | The auth plan wrote “`agent_logs` extended”, but `agent_logs` is the agent's operational stream (`level`/debug lines, migration 019). Recording mutations there would put operational noise behind the explorer's mutation-only filter, so the agent scope got its own shared-shape table. | Never — intentional split; merge only if the two streams are deliberately unified. |

**Removed in auth phase 3.** The legacy gRPC `AuthService` credential surface
(`CreateCredential` / `ListCredentials` / `RevokeCredential` / `WhoAmI`) and
`shared/proto/agency/v1/auth.proto` were deleted when admin user management
moved to `/api/rest/account/users`; the SDK's `whoAmI` now reads
`/api/rest/auth/session`. An external caller of the gRPC credential RPCs must
move to the REST admin surface.

**Removed in the task rework (CHPI-145).** The `review` task state, the
`Blocked` state (blocked is now a computed flag on `queued`), and the per-agent
Cleanup tab are gone. `ReviewTask`/`MoveTaskToDraft` were replaced by
`PromoteTask`/`DemoteTask`/`DequeueTask`, and `ListAgentSessions` /
`DeleteAgentSessions` were replaced by per-task `ArchiveTask` /
`CleanupTaskSession`. Migration `041_task_states_archive.sql` maps existing
`review` rows to `settled`, blocked drafts to `queued`, and the old
`task_references.blocking` boolean to a `kind`. The proto enum values
`TASK_STATE_BLOCKED` and `TASK_STATE_REVIEW` and the removed RPCs are reserved
where the wire allows; SQLite rows are not.

**Dropped in auth phase 6.** Task comments were designed (a separate Comments
tab, human discussion only, never sent to Pi) but removed from phase 6 before
implementation, so no `task_comments` table, no tab, and no comment events
exist. The task console remains the only prompt surface. A draft agency task
tracks reintroducing comments.

## Consistency debt

| Debt | Where | Why / state | Retire when |
| --- | --- | --- | --- |
| `kind=artifact` uploads are authorized by the actor's PAT, not a task-scoped token | `agency/server/internal/attachments/http.go` | The per-spawn artifact token was one of the removed 1-hour authorities. Artifact blobs are stored without a tray link (kept alive by `artifact_parts`), so the task linkage comes from the `task_artifact` event rather than the upload credential. | A task-scoped upload credential is reintroduced, or attachment uploads gain a per-task authorization check. |
| `listTasks` filters by *agent* id while the assignee is an *actor* id | `agency/server/internal/agenttools/handler.go`, `agency/server/sql/queries/tasks.sql` | The `agent_id` query param is resolved to an actor id with a `agents` subquery. `listProjects` now exposes both `agent_id` and `actor_id`, so the model can still address an assignee by actor id. | The filter accepts an `actor_id` directly and `agent_id` becomes a deprecated alias. |
| A task's prompt is edited through two paths | `agency/server/internal/tasks/tasks.go` (`Update` and `applyAttributes`), `shared/proto/agency/v1/task.proto` | `Update` replaces a draft task's whole shape (and renumbers it on a project move); the attribute-only `set_prompt` edits the prompt of a draft *or* backlog task without touching the rest. The split keeps the draft-form semantics intact while unlocking backlog editing. | `Update` is folded into the attribute API, or the project move keeps its task id. |

## Migration history kept intact

The task-size feature was removed and then restored, and both directions stay in
the migration chain rather than editing an applied file:
`011_task_size.sql` adds the column, `022_drop_task_size.sql` removes it, and
`034_task_size.sql` restores it. A fresh database and an existing one therefore
converge on the same schema and goose never sees a missing version. Do not
delete the superseded migrations; a future toggle should add another one. (The
number skips 032 and 033, which the in-flight federated-auth migrations claim.)

## Deliberate shortcuts (`ponytail:`)

Each of these is an intentional simplification with a known ceiling, kept
because the workload does not justify more. The source comment names the
upgrade path.

| Shortcut | Where | Ceiling / upgrade path |
| --- | --- | --- |
| Bootstrap rotation revokes tokens by the `agent-bootstrap` name | `agency/server/internal/agenttools/handler.go` | A repeated bootstrap replaces the agent's own previous token instead of accumulating. An operator-created token that happened to be named `agent-bootstrap` would be revoked with it; revisit if token names become user-managed. |
| Sparse limit overrides cleared and rewritten without a transaction | `agency/server/internal/limits/limits.go` | Safe while one operator edits limits rarely; wrap in a store transaction if that changes. |
| Task tag replace is delete+insert without a transaction | `agency/server/internal/tasks/tasks.go` (`SetTags`) | Safe while the agency is a single writer and tag sets are tiny; wrap in a transaction if concurrent edits land. |
| Task link replace is delete+insert without a transaction | `agency/server/internal/tasks/tasks.go` (`SetLinks`) | Now reachable from the UI (`SetTaskLinks`) as well as the agent's `setTaskUrls`, so a user edit can race an agent update; the last writer wins whole-list. Wrap in a transaction (or add a revision check) if that race matters. |
| Model defaults probe a fresh `pi` child per call | `agent/internal/session/manager.go` (`Defaults`) | Cache on the manager with a short TTL if the New Session form becomes hot. |
| Session watcher drops events for a slow subscriber | `agent/internal/session/session.go` | A slow watcher must not stall the run; revisit if a consumer needs lossless delivery. |
| Task console coalesces streaming deltas for 50 ms | `agency/frontend/src/components/tasks/TaskConsole.vue` (`queueDelta`) | A burst of Pi deltas re-renders and re-scrolls once per tick instead of once per token, which keeps a long thinking run from pinning the main thread. The transcript is up to 50 ms behind the live stream; shorten the interval (or render only the streaming tail as plain text) if that becomes noticeable. |
| `log.Printf` bridged into `slog` | `agent/internal/logging/logging.go` | The adapter keeps unstructured `log` lines reaching stdout and the sink while call sites migrate to `slog`; remove it once no `log.Printf` remains. |

## Stale docs

- [`tasks.md`](tasks.md) still describes the pre-actor schema and proto
  (`assignee_agent_id` referencing `agents(id)`). It predates migration 025 and
  should be refreshed with the actor model.

## Interface rework (CHPI-173)

- **Old settings URLs redirect.** `/projects/:slug/settings` now redirects to
  `/projects/:slug?tab=settings`, and the old per-agent
  `/agents/:username/{root-terminals,desktop,logs,settings}` URLs redirect to
  the matching agent-page tab (`?tab=terminals|rdp|settings`). Keep the
  redirects until external links have moved; they cost one route entry each.
- **Project task permissions are retired from the UI.** The agent Settings →
  Projects card no longer shows the old `can_list_tasks` / `can_create_tasks` /
  `can_update_tasks` switches: project roles and the effective-role model govern
  those capabilities now. `UpdateProjectAgent` still accepts the flags for
  older clients.
- **Agent runtime logs moved under Settings → Security.** `LogsView.vue` now
  accepts `embedded` and is mounted inside the Security tab's Collapsible
  (it also keeps its own Agent/Terminals sub-tabs). The old `/logs` URL
  redirects there; the `setPageTitle("Logs")` side effect is skipped while
  embedded.
- **Org settings are admin-only.** Non-admins no longer see the footer entry or
  can open `/settings`; their preferences live in User settings. The same page
  now owns compact Admin tables (users, teams, projects, agents), so the old
  standalone Users panel is no longer embedded in org settings.
- **Two webhook TODOs:** org webhooks are still event-filtered (they are meant
  to fire on every event — CHPI-176), and user-level webhooks are not built
  (CHPI-177). Advanced usage tabs on project/agent pages are placeholders
  (CHPI-175).
- **Team page tabs:** invitations, logs, and delete moved under Settings; the
  Members/Projects/Tasks tabs are read/maintain views. The team Projects tab
  uses the new `GET /api/rest/teams/{id}/projects`.

### Interface follow-up (CHPI-173 review)

- Agent page tabs now live in the identity header (Overview / Usage /
  Terminals / RDP / Settings), matching the project page's header tabs.
- A team's tasks render on its main **Profile** tab; the separate Tasks tab was
  removed. Use the team filter on the shared `TasksList` rather than a second
  list implementation.
- Sidebar user rows must keep the display-name `<span>`; an earlier revision
  rendered only the avatar. The clipped name is the whole row's accessible
  label when the sidebar is expanded, and `title` covers the collapsed rail.
- Admin activity moved out of **Org settings → Integrations** into
  **Org settings → Admin**:
  - `Activity` shows the full explorer (filters, CSV/JSON, retention).
  - `Users`, `Teams`, `Projects`, and `Agents` each embed a scoped
    `AdminActivityCard` for their own domain log.
    The old global Events panel is not mounted under Integrations any more.

## Project hierarchy

- **Closure maintenance** (`045_project_hierarchy`,
  `agency/server/internal/projects/hierarchy.go`): reparenting/detaching
  rebuilds the whole `project_ancestors` table inside one transaction. This is
  O(projects × depth) and fine at the agency's scale; a per-subtree update or a
  trigger would be the upgrade path.
- **Project moves are intentionally narrow.** Only a project with no parent can
  be attached, and only the parent's owner may detach. Moving a subtree between
  parents therefore means detach then attach; the old parent's owner and the
  new parent's side act separately, and existing cross-project references are
  re-validated on each step.
- **`projects.parent_project_id` has no database-level cycle guard** (SQLite
  cannot express it); the service validates every change and rebuilds the
  closure, but out-of-band SQL edits can still corrupt the tree.

## Team prompts, default tags, and attachment vs artifact

- **Team prompt pins are written after the claim transition.** `tasks.Claim`
  flips the task to `running` with the four scalar prompt pins in one query, but
  the per-team pins live in `task_team_prompts` and are written immediately
  afterwards (there is no cross-method transaction on the `Store` interface).
  A database error between the two leaves a running task without team pins; its
  system-prompt preview then omits the team scopes until a fresh session
  resolves them again. The upgrade path is a dialect-specific claim method that
  writes both in one transaction.
- **The `Prompt` converters are hand-written.** PostgreSQL appends the
  migration-added `prompts.team_id` column at the end of its physical row order,
  while the rebuilt SQLite table keeps it in the middle; `internal/store/gen`
  pairs structs by field order, so it cannot generate the converter. They live
  in `internal/store/prompt_convert.go`; if the generator starts emitting them,
  delete the hand-written copies.
- **`uploadAttachment` keeps its old alias in older sessions.** The bundled Pi
  tool was named `uploadArtifact`; running sessions from before the upgrade may
  still call that name. New agents install only `upload_attachment.ts`, so a
  restarted session gets the new name. Attachments are transient/current-status
  context; durable outputs go through `setTaskArtifacts`.

## Release pipeline debt

| Debt | Where | Why | Retire when |
| --- | --- | --- | --- |
| The public repository is a text transform, not a build product | `.github/scripts/publish-public.sh` | The public snapshot strips tests and test tooling from the origin tree. The strip list is explicit (test globs, helper packages, `vitest.config.ts`, `public-overlay/`, `deploy/github-runner/`, `release-notes/`) and the `Makefile` marks its dropped sections with `# BEGIN public:strip` / `# END public:strip`; the frontend Vitest config lives in its own file so `vite.config.ts` needs no patching. | A new test location or a new public-only path must be added here. A generated manifest (for example a `.publicignore`) that both the script and a CI guard share would retire the hand-maintained list. |
| The release job does not build the public snapshot | `.github/workflows/release.yml` | Preflighting a full build in origin would duplicate the public workflow. A broken snapshot therefore fails *after* the public commit and tag exist, and is recovered by re-running the public `Release` workflow from the tag. | The public build is cheap enough to run as a `dry_run` gate, or the release pushes to a validation branch first. |
| The public release body lives in the git tag annotation | `.github/workflows/release.yml` (origin) and the public `release.yml` | The notes are generated in origin but the GitHub Release is created by the public repository, so the tag annotation is the transport. Editing a release body in the GitHub UI does not survive a re-tag. | GitHub Releases cross-repo creation from origin (needs a token with `contents: write` on the public repo, which the job already has) replaces the annotation. |
