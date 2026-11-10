---
title: Testing plugins
description: Validate, unit-test, and run a CherryPi plugin end to end with cherrypi-plugin.
---

# Testing plugins

`cherrypi-plugin` is the plugin author's toolbox. It runs the same validation
and the same Lua host the agency uses, so a green run locally means the bundle
behaves in production.

It ships in the `cherrypi/agency` image and as a release artifact next to the
agency and agent binaries:

```sh
cherrypi-plugin version
# cherrypi-plugin dev (abc1234)
# plugin API 0.2.3
```

## Three layers

| Layer | Command | What it proves | Needs an agency? |
| --- | --- | --- | --- |
| Validate | `cherrypi-plugin lint` | Manifest, Lua syntax, handler contract, `requires_agency` | No |
| Unit | `cherrypi-plugin test` | Handler logic, SQL, provider calls | No — in-memory double |
| End to end | `cherrypi-plugin e2e` | Installation, settings, `create_task`, tools, events | Yes — throwaway agency |

Run them in that order: a lint failure is a second, a unit failure is a few
seconds, and an e2e run boots a real agency.

## Validate

```sh
cherrypi-plugin lint .
# ok acme-git-reviewer 0.1.0 (plugin API 0.2.3)
```

`lint` runs the installation validation (manifest schema, entry file, bundle
paths and limits), compiles every Lua file, and checks two author-only rules:

- the production code must not `require("cherrypi.test")` — the test module
  exists only in the test runner;
- `requires_agency` must accept this CLI's plugin API version.

## Unit tests

Put tests under `test/` in the bundle:

```text
my-plugin/
├── plugin.json
├── main.lua
├── migrations/sqlite/001_init.sql
└── test/
    └── webhook_test.lua
```

```sh
cherrypi-plugin test .
```

Each `test/**/*.lua` file runs in a **fresh Lua state**, against a fresh
in-memory agency and its own SQLite database with the bundle's migrations
applied. `require("cherrypi.test")` returns the harness:

```lua
local t = require("cherrypi.test")

-- Seed the agency double.
t.settings("org", {
  base_url = "https://messenger.example",
  api_key = "key",
  project = "INCIDENT",
  tags = { "incident" },
  webhook_secret = "s3cret",
})
t.project({
  id = "INCIDENT",
  name = "Incident Management",
  tags = { "incident" },
})

-- Expect one outbound call.
t.expect_http("POST", "/api/v4/posts", { status = 201, json = { id = "m1" } })

-- Drive a handler.
local status, _, body = t.http_request("POST", "/webhook", t.json({
  secret = "s3cret",
  channel_id = "ops",
  text = "Database is down",
}))

t.equal(status, 200)
local payload = t.decode(body)
t.truthy(payload.task_id)

-- Assert on what the plugin did.
local tasks = t.created_tasks()
t.equal(#tasks, 1)
t.equal(tasks[1].project, "INCIDENT")
t.contains(tasks[1].prompt, "Database is down")
t.equal(tasks[1].tags[1], "incident")
t.equal(tasks[1].tools[1], "reply_in_thread")

-- Tools run the same way.
t.tool_call(payload.task_id, "reply_in_thread", { text = "Looking into it." })
t.equal(#t.http_calls(), 1)
```

### Seeding

| Function | Purpose |
| --- | --- |
| `t.settings(scope, values)` | One settings row (`scope` is `org`, `agent`, or `user`). |
| `t.settings(scope, target, values)` | A targeted row, e.g. `t.settings("agent", "7", {...})`. |
| `t.enable(scope, target)` | Mark a row enabled. |
| `t.project({ id, name, tags, plugin_role, enabled, allow_task_creation, max_priority, settings })` | A project, its tags, the plugin actor's role, and its project surface. |
| `t.project_role(project_id, actor_id, role)` | Grant another actor a role. |
| `t.agent({ id, actor_id, username, name, project, project_role })` | An agent (and optionally its project role). |
| `t.user({ id, username, name })` | A user. |
| `t.task({ id, project_id, state, prompt, ... })` | A task the plugin may read. `hold_task` on a `settled` one parks it in `waiting`, as the agency does. |
| `t.history(task_id, entries)` | Persisted Pi entries for `get_task_history`. |
| `t.team({ id, name })` | A team. |
| `t.expect_http(method, path, { status, json, body, headers })` | One outbound response; fixtures are consumed in order. |

### Driving handlers and asserting

| Function | Purpose |
| --- | --- |
| `t.http_request(method, path, body, headers)` | Calls `handle_http_request`; returns its three values. A query string in `path` is split off and percent-decoded into `route.query`, exactly as production does. |
| `t.tool_call(task_id, name, args)` | Calls `handle_tool_call`; returns its value. |
| `t.event(name, payload)` | Calls `handle_event` with `event` and `task_id` set. |
| `t.tick(now_ms)` | Calls `handle_tick` with the timestamp a real tick passes; a bundle without the handler raises. |
| `t.ticks()` | The manifest's `schedule` as `{ every_seconds, interval_ms }`, or an empty table when the bundle does not tick. |
| `t.update_settings(scope, target, values)` | Runs the bundle's `update_settings` hook the way the agency does: the stored row of that surface as `previous`, a stored secret filled into a blank field, and the return checked by `parseSettingErrors`. Returns the `{path, errorText}` list (empty when the bundle defines no hook); a malformed return raises, as production treats it as a hook bug. `t.update_settings(scope, values)` uses the untargeted row of whichever scope you pass. It only drives the hook — nothing is stored. A hook that raises raises here too, so wrap it in `pcall`. |
| `t.task_tools(task_id)` | Runs the bundle's `handle_task_tools` hook and checks its return the way the scheduler does. Returns the project tool names the task may use (every declared tool unless the manifest declares the `task_tools` handler); a malformed return or a raising hook raises here too. |
| `t.created_tasks()` | The `create_task` calls, with `project`, `tags`, `tools`, `assignee`, `state`, `interactive`. |
| `t.holds(task_id)` | The plugin's holds on a task (`plugin`, `plugin_name`, `key`, `reason`, `created_at`), oldest first. |
| `t.releases(task_id)` | The `release_task` calls (`key`, `mode`, `prompt`), in call order. |
| `t.http_calls()` | Outbound calls (`method`, `url`, `path`, `body`). |
| `t.logs()` | The plugin's log lines. |
| `t.equal`, `t.not_equal`, `t.contains`, `t.truthy`, `t.falsy`, `t.fail` | Assertions; a failure stops the file with its message. |
| `t.json(value)`, `t.decode(text)` | JSON helpers. |

The outbound host still enforces the manifest's `network` allowlist, so use a
host the bundle declares (or add it).

A settings validator is testable the same way: seed the row it compares against,
drive the hook, and assert on the field errors that reject the write.

```lua
t.settings("org", { base_url = "https://gitea.example", api_key = "token-1" })

local errors = t.update_settings("org", { base_url = "", api_key = "" })
t.equal(#errors, 1)
t.equal(errors[1].path, "base_url")

-- A stored secret fills a blank field, the way a masked form saves.
t.equal(#t.update_settings("org", { base_url = "https://gitea.example" }), 0)

-- A project surface reads its project row.
t.project({ id = "REPO", settings = { base_url = "https://stored.example" } })
t.equal(#t.update_settings("project", "REPO", { api_key = "" }), 0)

-- A hook that raises rejects the write.
t.falsy(pcall(function()
  t.update_settings("org", { base_url = "not a url" })
end))
```

A harness request target may carry a query string; it is decoded into the same
table-of-arrays shape a real request produces:

```lua
local status, _, body = t.http_request("GET", "/webhook?secret=a%20b&tag=x&tag=y")
-- route.path == "/webhook"
-- route.query.secret[1] == "a b"
-- route.query.tag[1] == "x", route.query.tag[2] == "y"
```

## End to end

`e2e` installs the bundle into a **throwaway agency**, seeds a project, agent,
and tags, applies the scenario's settings, and runs its steps. The seeded agent
is granted the project's **member** role, because a viewer may not be assigned
or run a task.

```yaml
# plugin-test.yaml
agency:
  project: { name: Incident Management, prefix: INC }
  agent:
    username: incident-bot
    name: Incident Bot
    tool_args: { text: Looking into it. } # --with-agent
  tags: [incident]

settings:
  org:
    base_url: ${env.MOCK_URL}
    api_key: mock-key
    project: ${project.id}
    tags: [incident]
    webhook_secret: incident-secret

steps:
  - name: the webhook creates an incident task
    http:
      method: POST
      path: /webhook
      json:
        secret: incident-secret
        channel_id: ops
        text: Database is down
    expect:
      status: 200
      json: { task_id: "@nonempty" }
      tasks:
        - project: Incident Management
          tags: [incident]
          prompt_contains: Database is down

  - name: the agent runs and settles
    wait:
      state: settled
      timeout: 180s

  - name: the messenger received the reply
    http:
      method: GET
      url: ${env.MOCK_URL}/__calls
    expect:
      text_contains: Looking into it.
```

```sh
MOCK_URL=http://127.0.0.1:8081 \
  cherrypi-plugin e2e . --scenario plugin-test.yaml --with-agent
```

### Variables

| Variable | Value |
| --- | --- |
| `${env.NAME}` | The environment variable. |
| `${var.name}` | A `vars:` entry from the scenario. |
| `${project.id}`, `${project.name}`, `${project.slug}` | The seeded project. |
| `${agent.id}`, `${agent.actor_id}`, `${agent.token}` | The seeded agent. |
| `${plugin.id}` | The installed plugin. |
| `${agency.url}` | The agency base URL. |

### Steps

| Action | Fields | Notes |
| --- | --- | --- |
| `http` | `method`, `path` (plugin route), or `url` (absolute) | `json` or `body`; `headers`; `expect.status`, `expect.json`, `expect.text_contains`, `expect.tasks`. |
| `tool` | `name`, `args`, `task` | Calls the agent-tool route with the seeded agent's token. `task` defaults to the last task the scenario created. |
| `wait` | `state`, `timeout`, `task` | Polls the task until it reaches the state. |
| `task` | `project`, `name`, `prompt`, `state` | Seeds a task for an `event` step. `project` defaults to the seeded project, `state` to `backlog`. |
| `event` | `name`, `task`, `payload` | Dispatches an agency event through the plugin service. `task` is a task id or `last` (the default); `payload` adds fields to it. |
| `sleep` | duration | Pauses, e.g. `2s`. |

`expect.tasks` matches tasks **created by that step**. `assignee: "@agent"`
matches the seeded agent. `expect.json` compares a subset; `"@nonempty"` checks
that a string is present.

### Event-only bundles

A bundle that only implements `handle_event` has no route and no tool to call,
so nothing would ever reach it. The `task` and `event` steps drive it: `task`
seeds the task an event points at, and `event` dispatches the event through the
plugin service — the same path a real task transition takes.

```yaml
agency:
  project: { name: Acme User Bridge, prefix: AUB }

settings:
  org:
    base_url: ${env.MOCK_URL}
    api_key: mock-key
  user:
    acme_user: alice
    events: [task_completed, task_requested_input]

steps:
  - name: a completed task exists
    task:
      project: ${project.id} # optional, defaults to the seeded project
      name: Fix the deploy # optional
      prompt: The deploy is broken.
      state: completed # optional, defaults to backlog
      assignee: "@agent" # optional: pins the task to the seeded agent
    expect:
      tasks:
        - project: Acme User Bridge
          state: completed

  - name: the completion is delivered as a direct message
    event: task_completed # shorthand: the last task, no payload extras

  - name: the messenger received it
    http:
      method: GET
      url: ${env.MOCK_URL}/__calls
    expect:
      text_contains: Fix the deploy

  - name: the input request carries its title
    event:
      name: task_requested_input
      task: last # a task id, or last (the default)
      payload: { title: "Which branch should I deploy?" }
```

`task.state` is the state the task is **stored** in — `backlog`, `queued`,
`running`, `settled`, `failed`, `input`, `completed`, or `cancelled` — so the
event payload's task snapshot reports it. `task.assignee` pins the task to the
seeded agent (`"@agent"`) or to an agent actor id; a `tool` step needs it,
because the agent-tool route only accepts a call for a task assigned to the
calling agent. Seeding writes the row directly and
never dispatches an event of its own, so the scenario decides exactly which
events the plugin sees. `event` waits for every subscriber's `handle_event` to
finish, so the next step cannot race the delivery.

Both steps call development-only agency routes, so they need an agency built
with `-tags dev`; the `cherrypi/plugin-ci` image is, the `cherrypi/agency` image
is a release build.

### Scheduled ticks

A bundle with a manifest `schedule` also needs its `handle_tick` driven. The
`tick` step runs one tick through the same single-flight path the ticker uses
and returns once the handler finished, so the next step can assert on what it
created. The e2e agency starts with its background ticker disabled
(`CHERRYPI_PLUGIN_DISABLE_TICKS=1`), so an interval cannot race the scenario.

```yaml
- name: a tick polls the mentions
  tick: {} # or `tick: true`
  expect:
    tasks:
      - project: Incident Management
        prompt_contains: Please look at the queue
```

`tick: { now_ms: 1700000000000 }` (or the shorthand `tick: 1700000000000`)
fixes the timestamp `handle_tick` receives.

### The two e2e modes

- **Agency only** (default): fast, and enough for webhooks, settings,
  `create_task`, and the `task`/`event` steps. A `tool` step calls the
  agent-tool route directly, so the task must be assigned to the seeded agent.
- **With an agent** (`--with-agent`): boots the real agent and a mock LLM whose
  first turn calls the bundle's first tool. This exercises the whole path —
  scheduler, Pi, the bundled tool extension, the agency route, and
  `handle_tool_call`. The scenario's `agent.tool_args` become the tool
  arguments.

`--with-agent` needs `cherrypi-agent`, `cherrypi-mockllm`, and `pi` (all in the
`cherrypi/plugin-ci` image). Pass `--keep` to keep the data directory and `-v`
to stream the agency and agent logs.

### Drive an agency you did not boot

`--agency-url` runs the scenario against an agency that is already up — a dev
agency in a terminal, or a staging deployment — instead of booting a throwaway
one. The harness logs in as an administrator, so it needs that agency's admin
credentials:

| Flag | Environment variable | Default |
| --- | --- | --- |
| `--agency-username` | `CHERRYPI_ADMIN_USERNAME` | `admin` |
| `--agency-password` | `CHERRYPI_ADMIN_PASSWORD` | none; the run stops and names the flag |

```sh
CHERRYPI_ADMIN_PASSWORD=adminpass \
  cherrypi-plugin e2e . --scenario plugin-test.yaml --agency-url http://127.0.0.1:8096
```

Everything else is the same as the boot path, but the harness seeds into the
agency you point it at and leaves the project, agent, plugin installation, and
tasks behind, so use a scratch instance. That agency also needs
`CHERRYPI_PLUGIN_ALLOW_PRIVATE_NETWORK=1` when a scenario calls a provider mock
on `127.0.0.1`, and it must be built with `-tags dev` for the `task` and `event`
steps.

### Provider mocks

The plugin repo owns its provider mocks. Start a small HTTP server that speaks
the provider API your plugin calls (and an assertion endpoint such as
`/__calls`), then point the plugin's `base_url` at it through `${env.MOCK_URL}`.
The agency is started with `CHERRYPI_PLUGIN_ALLOW_PRIVATE_NETWORK=1`, so
`127.0.0.1` works; the manifest's `network` allowlist still applies.

## CI

The harness runs in a container, so a plugin job is a checkout and a few
commands. No agency secrets are needed: the agency is throwaway and seeded by
the CLI.

```yaml
name: plugin
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: gitea.vladfaust.com/cherrypi/plugin-ci:dev
      credentials:
        username: ${{ secrets.REGISTRY_USERNAME }}
        password: ${{ secrets.REGISTRY_TOKEN }}
    steps:
      - uses: actions/checkout@v4

      - name: Lint
        run: cherrypi-plugin lint .

      - name: Unit tests
        run: cherrypi-plugin test .

      - name: End to end
        env:
          MOCK_URL: http://127.0.0.1:8081
        run: |
          node mock/server.mjs &      # the plugin's own provider mock
          for _ in $(seq 1 30); do curl -fsS "$MOCK_URL/__calls" && break; sleep 0.2; done
          cherrypi-plugin e2e . --scenario plugin-test.yaml --with-agent

      - name: Package
        run: cherrypi-plugin pack . -o plugin.zip
```

Drop `--with-agent` for a bundle that has no tools: an event-only scenario runs
without an agent.

- Use `cherrypi/agency:<tag>` when you do not need the agent (lint, unit tests,
  webhook e2e); it is much smaller. It is a release build, so it has no
  `task`/`event` routes: an event-only scenario needs `cherrypi/plugin-ci`.
- Use `cherrypi/plugin-ci:<tag>` for `--with-agent` and for the `task`/`event`
  steps.
- `pack` runs `lint` first and writes a deterministic zip you can attach to a
  release and install through the UI.

### Test more than one agency version

`requires_agency` is the contract your bundle declares. Test the range's edges
with a matrix:

```yaml
strategy:
  matrix:
    image:
      - gitea.vladfaust.com/cherrypi/plugin-ci:dev
      - gitea.vladfaust.com/cherrypi/plugin-ci:v0.1.0
```

`cherrypi-plugin lint` fails when the image's plugin API is outside
`requires_agency`, so the matrix tells you exactly which version broke.

## Troubleshooting

| Symptom | Cause |
| --- | --- |
| `module 'cherrypi.test' is not in the bundle` | The test module is only injected by `cherrypi-plugin test`; run through the CLI. |
| `host "…" is not in the manifest allowlist` | Add the provider host to `network`; unit tests use the same guard. |
| `the plugin is not enabled in this project` | The scenario's `agency.project` and `settings.org.project` disagree, or the project surface was not enabled. |
| `priority … exceeds the project's plugin cap` | Raise the project's maximum priority, or lower the requested one. |
| `task is not assigned to this agent` | A `tool` step ran on an unassigned task; use `--with-agent`, or assign the task to the agent. |
| `… needs an agency built with -tags dev` | A `task` or `event` step ran against a release agency (the `cherrypi/agency` image); use `cherrypi/plugin-ci`. |
| `no enabled plugin subscribes to "…"` | The event name is not in the bundle's manifest `events` list, or the plugin is disabled. |
| `bundle requires agency …` | The image's plugin API is outside `requires_agency`. |

## Next steps

- [Plugins](/advanced/plugins) — handlers, methods, and settings surfaces.
- [JavaScript SDK](/advanced/sdk) — the client the harness uses under the hood.
