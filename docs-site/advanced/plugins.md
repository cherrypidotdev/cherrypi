---
title: Plugins
description: Extend the agency with sandboxed Lua plugins — webhooks, task tools, and event handlers.
---

# Plugins

A plugin is a **sandboxed Lua bundle** installed at the agency level. It is not
a daemon: the agency invokes it as a request handler, the invocation runs to
completion, and the state it needs lives in its own database.

Plugins are how an integration reaches CherryPi without forking the agency: a
provider webhook creates a task, a task gets a provider-specific tool, or a task
event is delivered to a chat service.

::: tip
The agency runs the Lua code; you only ship text files. There is no compiler,
no WASM toolchain, and no sidecar process.
:::

## What a plugin can do

| Handler | Called when | Typical use |
| --- | --- | --- |
| `handle_http_request` | A request arrives at `/plugins/<id>/…` | Provider webhook, OAuth callback |
| `handle_tool_call` | An agent calls a tool the plugin attached to a task | Approve a merge request, reply in a thread |
| `handle_event` | The agency emits an event the plugin subscribed to | Notify a user that a task needs input |
| `handle_tick` | The plugin's manifest schedule fires while it is enabled | Poll a provider that cannot send webhooks |

A plugin can also call back into the agency: read a project or task, create a
task with a prompt and tools, and read its own settings.

## Install a plugin

1. Zip the bundle contents — `plugin.json`, the Lua files, and optional
   `migrations/` — not the directory itself:

   ```sh
   cd my-plugin
   cherrypi-plugin pack . -o ../my-plugin.zip
   ```

   `pack` skips version-control metadata (`.git`, `.hg`, `.svn`, and the
   other VCS directories) and anything matched by the tree's `.gitignore`
   files, so a repository root can be the bundle directory. `lint`, `test`,
   and `e2e` apply the same filter. A plain `zip -r` does not: it would ship
   the full history and ignored build output.

2. Open **Organization settings → Plugins** and upload the zip.
3. Installing enables the plugin's **organization** settings and creates its
   actor (`@plugin-<id>`).

Installing the same id again updates it and keeps its enabled state. Disabling
stops every handler; uninstalling removes the code and keeps the data unless
you ask to purge it.

## Settings surfaces

A plugin declares the surfaces it needs in its manifest. Each surface stores its
own JSON values, and each has its own enablement rule:

| Surface | Enabled by | Where it is edited |
| --- | --- | --- |
| `org` | Installing the plugin | Organization settings → Plugins |
| `project` | A project owner invites the plugin actor, then enables it | Project settings → Plugins |
| `agent` | The agent's owner enables it | Agent settings → Plugins |
| `user` | The user enables it | User settings → Plugins |

A project surface also carries two agency-enforced switches:

- **Allow task creation** — when off, `create_task` fails even if the plugin is
  enabled;
- **Maximum task priority** — defaults to medium, so a plugin cannot quietly
  create critical work.

A surface that stores no values of its own is declared with an empty
`properties` map. That is how a plugin asks project owners to install and
enable it without project-scoped settings:

```json
"settings": { "project": { "type": "object", "properties": {} } }
```

Such a surface still shows the agency-enforced switches and the runtime logs,
and says it declares no project settings form.

A surface with no `properties` key at all (`{"type":"object"}` or
`{"type":"object","additionalProperties":true}`) is the opposite: it keeps a
free-form JSON editor, for values the plugin reads without declaring a shape.
Declare fields in `properties` whenever the plugin knows them — that is what
gives the editor a typed form, titles, and secret masking.

The plugin actor must be a project member with member access before the project
surface can be enabled. Invite it from **Project settings → Access** or from the
plugin panel.

::: warning Secrets
A settings property with `"format": "password"` (or `"secret": true`) is
write-only. The agency redacts it in responses, and submitting an empty value
keeps the stored one.
:::

## Write a plugin

### The manifest

```json
{
  "id": "acme-git-reviewer",
  "name": "AcmeGit Reviewer",
  "version": "0.1.0",
  "requires_agency": ">=0.2.0 <0.3.0",
  "description": "Turns review events into review tasks.",
  "runtime": "lua",
  "entry": "main.lua",
  "handlers": ["http", "tool"],
  "settings": {
    "org": {
      "type": "object",
      "properties": {
        "base_url": { "type": "string", "title": "Git base URL" },
        "api_key": { "type": "string", "format": "password" }
      }
    },
    "project": {
      "type": "object",
      "properties": {
        "git_project_id": { "type": "string" },
        "webhook_secret": { "type": "string", "format": "password" }
      }
    }
  },
  "tools": [
    {
      "name": "approve_mr",
      "description": "Approve the merge request under review.",
      "input_schema": {
        "type": "object",
        "required": ["approve"],
        "properties": { "approve": { "type": "boolean" } }
      }
    }
  ],
  "project_tools": ["open_merge_request"],
  "events": ["task_settled"],
  "network": ["git.example.com"],
  "routes": [{ "path": "/webhook", "methods": ["POST"] }],
  "limits": { "timeout_seconds": 10, "max_concurrent": 2 }
}
```

| Field | Meaning |
| --- | --- |
| `id` | Lowercase slug, 2–48 characters. Becomes `/plugins/<id>/` and `@plugin-<id>`. |
| `requires_agency` | Semver range of the plugin API the bundle supports. Empty accepts any. |
| `handlers` | `http`, `tool`, `event`, `tick`, and the optional `task_tools` hook. The first four are inferred from `tools`, `events`, and `schedule` when omitted; `task_tools` is only ever declared. |
| `settings` | Per-surface JSON Schemas. Only declared surfaces are rendered and readable. |
| `tools` | Named tools the plugin can attach to a task. |
| `project_tools` | Names from `tools` that every task started in an enabled project can call. See [Project tools](#project-tools). |
| `schedule` | Optional periodic trigger for `handle_tick`: `{ "every_seconds": 5…3600 }`. See [Scheduled ticks](#scheduled-ticks). |
| `events` | Agency events `handle_event` receives. |
| `network` | Outbound host allowlist: an exact host, a `*.example.com` subdomain wildcard, or `*` for any host. See [Network allowlist](#network-allowlist). |
| `routes` | Documentation for the paths under `/plugins/<id>/`; dispatch does not require it. |
| `limits` | Lower the timeout, response size, request size, or concurrency. |

`requires_agency` is the plugin API range the bundle supports. The current API
is `0.2.3`, which added the optional `task_tools` handler on top of `0.2.2`'s
`schedule` field and `handle_tick` and `0.2.1`'s `task_hold_released` event. A
bundle that ticks declares `>=0.2.2 <0.3.0`, because an older agency rejects the
unknown manifest field. A bundle that narrows its project tools declares
`>=0.2.3 <0.3.0`, because an older agency drops the unknown handler kind and
attaches every project tool. Nothing existing changes, so a bundle that does
neither keeps `>=0.2.0 <0.3.0` and installs on a `0.2.2` agency unchanged — the
same rule `0.2.1` followed for its optional event, while `0.2.0`'s task holds
moved the range because a bundle built against them cannot run on a `0.1.0`
agency.

### Network allowlist

`network` is the outbound host allowlist. Each entry is one of:

| Entry | Matches |
| --- | --- |
| `git.example.com` | Exactly that host. Subdomains are **not** included. |
| `*.example.com` | Any subdomain (`api.example.com`, `a.b.example.com`), but not the apex `example.com`. List the apex separately when the plugin needs it. |
| `*` | Any host. |

`*` is a broad, explicit permission: the plugin may call any public HTTPS host.
That is what a plugin needs when the provider URL is an operator setting, for
example a self-hosted MatterMost `base_url`. It must be the only entry in
`network`, it renders as a warning in the plugin detail UI, and it deserves a
manifest review before the bundle is installed. It does not relax the HTTPS
rule or the agency's private-address guard.

### Handlers

```lua
function handle_http_request(route, headers, body)
  -- route.method, route.path, route.query (a table of arrays)
  -- headers is a lower-cased table of every request header; body is the raw
  -- request body string.
  if route.method ~= "POST" or route.path ~= "/webhook" then
    return 404, { ["content-type"] = "text/plain" }, "not found"
  end
  local settings = cherrypi.settings.get("org", "")
  local signature = headers["x-gitea-signature"] or ""
  local expected = cherrypi.crypto.hmac_sha256(settings.webhook_secret, body)
  if not cherrypi.crypto.secure_equals(signature, expected) then
    return 401, { ["content-type"] = "application/json" }, '{"error":"bad signature"}'
  end
  local payload = cherrypi.json.decode(body)
  local task_id = cherrypi.create_task({
    project = settings.project,
    title = "Review: " .. payload.title,
    prompt = "Review this change:\n" .. payload.text,
    tools = { { name = "approve_mr" } },
  })
  return 200, { ["content-type"] = "application/json" },
    cherrypi.json.encode({ task_id = task_id })
end

function handle_tool_call(task_id, name, args)
  if name ~= "approve_mr" then
    error("unknown tool " .. name)
  end
  local rows = cherrypi.db.query(
    "SELECT mr_iid FROM reviews WHERE task_id = ?", { task_id }
  )
  -- call the provider with cherrypi.http.request ...
  return "Merge request approved."
end

function handle_event(payload)
  cherrypi.log.info("task " .. payload.task_id .. " is " .. payload.state)
end
```

`handle_http_request` returns `(status, headers, body)`. A body string is sent
verbatim; a table is JSON-encoded. `handle_tool_call` returns text (a table is
JSON-encoded). `handle_event` returns nothing.

### Scheduled ticks

A provider that offers no inbound push — or whose operator refuses to configure
a webhook — is reachable only by polling. The optional `schedule` manifest field
invokes `handle_tick` periodically while the plugin is enabled:

```json
"schedule": { "every_seconds": 15 }
```

```lua
function handle_tick(now_ms)
  -- now_ms is the tick's scheduled time in Unix milliseconds.
  local cursor = cherrypi.settings.get("org", "").cursor or "0"
  local response = cherrypi.http.request(
    "GET", "https://chat.example.com/api/mentions?since=" .. cursor)
  -- Create tasks for new mentions, then store the new cursor in the plugin DB.
end
```

- `every_seconds` is an integer from 5 to 3600; an absent `schedule` means the
  plugin is never ticked. Validation rejects an out-of-range value and a
  schedule on a bundle whose entry file does not define `handle_tick`.
- The ticker starts after install or enable and on agency boot for every
  enabled plugin; disable and uninstall stop it. The first tick is offset by a
  random jitter smaller than one interval, so a fleet does not tick in lockstep.
- Ticks are **single-flight per plugin**: while one is running the next interval
  is skipped and a `warn` line is logged. Ticks never queue.
- A tick that raises or times out is logged at `error` and the ticker keeps its
  schedule. There is no catch-up after downtime: skipped intervals are skipped,
  and the plugin keeps its own cursor for what it has processed.
- A tick is a normal invocation: a fresh `LState`, a `max_concurrent` slot, the
  manifest `timeout_seconds` (default 10 s), the network allowlist, and
  `cherrypi.log.*` with whatever surfaces the plugin passes.

### Settings validation

A plugin that wants to check its settings defines `update_settings`. The agency
calls it before storing a write for one surface, with the new values and the
stored ones:

```lua
function update_settings(scope, target_id, values, previous)
  if scope ~= "project" then return nil end
  local ok, tags = pcall(cherrypi.list_project_tags, target_id)
  if ok and not has_tag(tags, values.tag) then
    return { { path = "tag", errorText = "The project has no such tag." } }
  end
  return nil
end
```

- Return `nil` or an empty list to accept the write.
- Return a list of `{ path, errorText }` to **reject** it: nothing is stored, the
  form highlights each path with its message, and a toast reports the failure.
- Raising a Lua error also rejects the write, with one form-level message.
- The hook runs only while the plugin is enabled, so a disabled plugin's
  settings stay editable.

Validate offline first (shape, allowed values) and only then call the provider:
an `update_settings` that fetches on every keystroke save is slower than the
form it backs.

The webhook endpoint is **public**: the agency does not require a session, so
the plugin validates its own secret. A caller with a session or personal access
token is resolved and available as `cherrypi.actor`.

Every request header reaches `headers`, including `authorization`, `cookie`,
`x-csrf-token`, and `proxy-authorization`. That is what lets a provider webhook
use its static **Authorization Header** or an `X-Hub-Signature-256` style
signature, and it means a plugin that also serves signed-in callers can read
those callers' agency session headers. Treat the handler's `headers` table as
secret material: never log it, and never forward it to an outbound host. See
`SECURITY.md` in the repository.

### Task tool filter

A plugin that declares `task_tools` in `handlers` defines `handle_task_tools`.
The scheduler calls it when a task starts — after the task is claimed, so the
assignee is known — and asks which of the plugin's declared project tools that
task may use. `ResolveTool` applies the same filter, so a tool the hook hides
cannot be called directly either.

```lua
function handle_task_tools(task_id)
  local task = cherrypi.get_task(task_id)
  local settings = cherrypi.settings.get("agent", tostring(task.assignee))
  if (settings.api_key or "") == "" then
    return {}
  end
  return { "open_merge_request", "reply_in_discussion" }
end
```

- Return a list of declared project tool names to narrow the task to them; an
  empty list hides all of them, and `nil` attaches every declared one.
- The handler is never inferred: a plugin without it keeps attaching every
  project tool, exactly as before.
- A hook that raises, or returns something other than a list of the plugin's
  declared project tools, logs a warning and falls back to every declared tool
  (fail open); `handle_tool_call` still validates its own preconditions.
- Task tools attached with `create_task` are not filtered: the hook narrows
  project tools only.
- The filter is evaluated when the scheduler creates a task's Pi session, the
  only start payload that carries tools. A task resumed on a warm session — a
  settle followed by a follow-up, or a retry of a live session — keeps the tool
  list it started with, so a credential stored mid-task takes effect on the next
  fresh start.

This is how a provider tool that needs an agent credential stays off a task
whose assignee has none, instead of being offered and failing at call time.

### Agency methods

| Method | What it does |
| --- | --- |
| `cherrypi.get_org()` | Agency identity. |
| `cherrypi.get_project(id_or_slug)` | Project metadata. Requires membership or an enabled project surface. |
| `cherrypi.get_team(id)` | Team metadata. Requires plugin team membership. |
| `cherrypi.get_agent(id_or_actor)` | Agent metadata. Requires the agent or a shared project to have enabled the plugin. |
| `cherrypi.get_user(id_or_username)` | User metadata. Requires that user to have enabled the plugin. |
| `cherrypi.get_task(id)` | Task snapshot. Requires a plugin-created task or an enabled project. |
| `cherrypi.get_project_tasks(project_id, opts)` | Project tasks (`limit`, `state`, `search`, `archived`). |
| `cherrypi.get_task_history(task_id, opts)` | Persisted Pi session entries. |
| `cherrypi.create_task(input)` | Create a queued task. See below. |
| `cherrypi.add_task_message(task_id, text)` | Send a follow-up prompt to a task: queues a resting task, appends to a queued, waiting, or input one's pending prompt, steers a running one. A text stashed on an input task is delivered once its dialog is answered, or with the next start if the run ends without an answer; consecutive follow-ups are capped at 20 per task. |
| `cherrypi.close_task(task_id, outcome)` | Close a task out as `completed` or `cancelled`, attributed to the plugin's actor. A running task is refused, and so is a completed close-out of a task that has not run a follow-up stashed for it yet; a cancel may discard that follow-up. |
| `cherrypi.set_task_badges(task_id, badges)` | Replace the badges the plugin shows for a task. |
| `cherrypi.hold_task(task_id, hold)` | Hold a task until the plugin verifies its outcome (`key`, `reason`). Idempotent per key, at most 8 holds per task. See [Task holds](#task-holds). |
| `cherrypi.release_task(task_id, key, options)` | Remove one hold. `options.mode` is `settle`, `queue`, or `complete`; `queue` also needs a `prompt`. |
| `cherrypi.list_task_holds(task_id)` | The task's holds, oldest first. |
| `cherrypi.list_agents()` | Agents that enabled the plugin, with their agent settings. |
| `cherrypi.list_settings(scope)` | The plugin's own settings rows for a surface. |

`create_task` takes:

```lua
{
  project = "PROJ",              -- id or slug (required)
  title = "Incident: DB down",   -- optional; auto-titled when empty
  prompt = "Investigate …",      -- required
  tags = { "incident" },         -- names or ids defined in the project
  assignee = "agent-actor-id",   -- an agent that is a project member
  size = "medium",               -- small | medium | big | epic
  priority = "normal",           -- low | normal | high | critical
  interactive = true,            -- false runs without extension dialogs
  task_references = {            -- informational unless kind = "blocking"
    "PROJ-1",
    { task_id = "PROJ-2", kind = "blocking" },
  },
  tools = { { name = "approve_mr" } },
}
```

The project's **maximum task priority** caps the request; a higher priority is
an error, not a silent downgrade.

`interactive` defaults to `true`. A non-interactive task never surfaces a
questionnaire or another extension dialog: the agent is told to ask in plain
text instead. Unlike the task API and the agent tools, a plugin may create a
non-interactive top-level task. The mode is fixed at creation.

### Host APIs

| API | Purpose |
| --- | --- |
| `cherrypi.log.info/warn/error(message, data)` | Structured log lines in the plugin panel. |
| `cherrypi.json.encode/decode` | JSON helpers. |
| `cherrypi.settings.get(scope, target)` | The plugin's own settings for a surface. `cherrypi.settings.<key>` reads the org values. |
| `cherrypi.http.request(method, url, opts)` | Outbound HTTPS, restricted to `network` (exact hosts, `*.` subdomains, or `*`). `opts` accepts `headers`, `body`, `timeout_ms`. |
| `cherrypi.db.query(sql, args)` / `cherrypi.db.exec(sql, args)` | The plugin's isolated database. |
| `cherrypi.crypto.hmac_sha256(secret, message)` | Lowercase hex HMAC-SHA256, e.g. to verify `X-Gitea-Signature` or `X-Hub-Signature-256`. |
| `cherrypi.crypto.secure_equals(a, b)` | Constant-time string compare; use it for signatures and shared secrets. |
| `cherrypi.time.now_ms()` | Unix milliseconds. |
| `cherrypi.actor` | The caller: `kind`, `actor_id`, `username`, `admin`. |

### Task tools

A tool listed in `tools` becomes callable when the plugin attaches it with
`create_task({ tools = { { name = "approve_mr" } } })`. The agency hands the
definitions to the agent, which registers one Pi tool per definition; a call
goes back to the agency, which verifies the task belongs to the calling agent
and that the tool is attached before invoking `handle_tool_call`.

Because the plugin keeps the provider mapping in its own database, the tool
arguments do not repeat context the plugin already has — the task id is enough.

### Project tools

A tool named in `project_tools` is attached to **every** task started in a
project where the plugin is enabled, not only to the tasks the plugin creates.
That is how a provider integration offers a tool to work the project already
does: for example a `open_merge_request` tool that opens the pull request and
records which task it belongs to, instead of the agent hand-rolling API calls.

```json
"tools": [
  { "name": "open_merge_request", "description": "…", "input_schema": { … } }
],
"project_tools": ["open_merge_request"]
```

- The names must be declared in `tools`; the agency validates them at install.
- A project tool applies while the plugin is enabled in the project, so
  disabling the plugin removes it from new task runs.
- A plugin that declares the `task_tools` handler narrows the list per task
  with `handle_task_tools(task_id)`; see
  [Task tool filter](#task-tool-filter).
- `handle_tool_call(task_id, name, args)` receives the task the tool was called
  from, whether or not the plugin created it.
- A task tool of the same name wins, because it carries task-specific context.

### Task badges

A plugin can show provider state for a task — the pull request behind it and its
merge state, a CI result:

```lua
cherrypi.set_task_badges(task_id, {
  { key = "pr", label = "Pull request", value = "#42 open", tone = "info",
    url = "https://git.example/org/repo/pulls/42" },
  { key = "ci", label = "CI", value = "success", tone = "success" },
})
```

A badge has the two-part shape of a shields.io badge: a neutral `label` on the
left and an optional coloured `value` on the right. With no `value` it renders
as a single neutral part.

- `key` is stable within the plugin and `tone` colours the value part:
  `neutral`, `info`, `success`, `warning`, or `danger`. `url` is an optional
  link.
- Badges appear in the task details panel, in the tasks table's **Badges**
  column, and in the task's activity log.
- An update replaces the plugin's previous set (at most 8 badges); an empty list
  removes them.
- Every change is also appended to the task's activity log, so the history stays
  auditable.
- Plugins are request-invoked, so a plugin refreshes its badges when a provider
  webhook or its own tool call gives it new information.

### Task holds

A task's work can be finished while its outcome is still unverified: a pull
request whose CI is not green, a review nobody approved yet. A plugin holds the
task until it can verify the outcome:

```lua
cherrypi.hold_task(task_id, {
  key = "ci",
  reason = "CI is not green on pull request #12",
})

-- Later, when every check passed:
cherrypi.release_task(task_id, "ci", {
  mode = "queue",
  prompt = "CI is green; finish the review.",
})
```

- A held task is parked in `waiting`: it frees its agent slot and the
  scheduler never claims it. The task details panel lists the holds beside its
  references and sub-tasks, and its waiting reason names the plugin and the
  reason.
- `key` is the condition's stable name within the plugin (`ci`, `approval`),
  `[a-z0-9_-]{1,32}`; holding the same key twice only refreshes the reason. A
  plugin may hold one task for at most 8 conditions.
- A run that ends while a hold exists parks the task in `waiting` and fires the
  `task_held` event instead of settling it. `task_settled` still means the task
  really settled.
- An operator can release one hold from the task drawer (`ReleaseTaskHold`): a
  lead or an administrator picks the mode, and a hold that is no longer there is
  refused instead of applying a stale mode.
- Every hold that leaves a task without the plugin releasing it fires
  `task_hold_released` with `plugin`, `key`, `reason`, `mode`, `applied`, and
  `cause` (`operator`, `closed_out`, `dequeued`, `archived`, or
  `plugin_disabled`). `applied` is false when another hold or obstacle still
  keeps the task waiting. A `plugin_disabled` release reaches the plugin being
  disabled before its runtime is unloaded, though it can no longer act on tasks
  by then.
- The release mode applies only when this was the task's last hold, the task has
  no unfinished blocker or sub-task, and no other plugin holds it. `settle` (the
  default) settles it, `queue` queues it again with the prompt, and `complete`
  closes it out. If an obstacle remains, the task stays `waiting` and the mode is
  forgotten. A mode the task refuses keeps the hold and returns the error, so the
  task stays parked instead of stranding with nothing to release it.
- A manual close-out, `Dequeue`, or `Archive` clears every hold, and an explicit
  queue of a held task is refused. Disabling or uninstalling the plugin releases
  its holds with mode `settle`, so a hold whose owner is gone cannot strand a
  task.

## Data, secrets, and logs

- Each plugin gets an isolated SQLite file (or a dedicated PostgreSQL
  database/schema) and applies its `migrations/sqlite` or `migrations/pg`
  files in order.
- Settings are stored per surface; secret values never leave the agency.
- Every invocation and host call is logged; **Organization settings → Plugins
  → Logs** shows the recent lines.
- Bundles are validated before they touch the disk: path traversal, oversized
  entries, too many files, and a missing manifest or entry file are rejected.
- Packing and loading skip version-control metadata and paths matched by the
  tree's `.gitignore` files, so a repository root is a safe bundle directory.

## Runtime and security

- Outbound calls are HTTPS-only and host-allowlisted: an entry matches an exact
  host, a `*.` subdomain wildcard, or `*` for any host. The dev/CI private-network
  switch lifts the private-address guard so mock providers on localhost can be
  reached; the hostname allowlist still applies.
- `cherrypi.crypto.hmac_sha256` and `cherrypi.crypto.secure_equals` cover
  provider webhook signatures without exposing key material beyond the
  invocation.
- `handle_http_request` receives every request header, including the agency
  session's `cookie` and `authorization`. A plugin is trusted code once
  installed; review it before enabling. See `SECURITY.md` in the repository.

## Limits

- Only Lua is available today; JavaScript and WASM are planned behind the same
  handler contract.
- There is no plugin-internal project type: a plugin works in ordinary
  projects where its actor is a member.
- Plugin pages and widgets are not available yet; `handle_http_request` is the
  only HTTP surface.
- Team-scoped settings are reserved for a later round.

## Next steps

- [Testing plugins](/advanced/plugin-testing) — the `cherrypi-plugin` harness
  and CI.
- [Projects & access](/using/projects) — how membership and roles work.
- [Tasks](/using/tasks) — the lifecycle a plugin-created task enters.
