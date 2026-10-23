# Agents

Guidelines for agents (and humans) working in this repository. Architecture
lives in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), terms in
[`VOCABULARY.md`](VOCABULARY.md), observability in
[`docs/OBSERVABILITY.md`](docs/OBSERVABILITY.md), releases in
[`docs/CI.md`](docs/CI.md) and [`CHANGELOG.md`](CHANGELOG.md).

## This repository is generated

Every commit on `main` is a release snapshot, generated from the private
development repository. Do not open pull requests here: the tree is replaced on
every release. User-facing problems and questions belong in
[Issues](https://github.com/cherrypidotdev/cherrypi/issues).

The test suite and the CI harness live in the private repository, so `make
test` and `make lint` do not exist here. Build and runtime documentation below
still applies.

## Commits

Commits here are generated (`chore(release): vX.Y.Z`) and are never authored by
hand.

## Scope

[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) is the map. This repository ships
three Go modules plus an SDK:

- `agency/server` — the agency: agent registry, users, settings, tags, task
  lifecycle + scheduler, webhooks, notifications, usage aggregation, the tunnel
  hub, and the embedded SPA.
- `agency/frontend` — the Vue SPA (agency shell + per-agent views).
- `agency/sdk/js` — `@cherrypi/sdk`.
- `agent` — the worker: tasks (Pi sessions), terminals, workspace, tunnel client.
- `shared` — all protos + generated stubs + the tunnel byte-pipe plumbing.

The agency is the **only orchestrator**; agents are separate binaries it never
calls directly — they dial in over the tunnel. Host concerns (GPU drivers, user
isolation, networking, reverse proxy) are documented, not built.

## Backend conventions

- **Proto first.** All protos live in `shared/proto/**/*.v1`, regenerated with
  `buf generate` (from `shared/`); **SQL first** — per-module schema in
  `sql/migrations` (goose), queries in `sql/queries` (sqlc), regenerated with
  `sqlc generate`. The agency has one tree per database dialect
  (`agency/server/sql/{sqlite,pg}/{migrations,queries}`) because sqlc generates
  one package per engine. Add every schema migration to both trees; queries
  may differ where the engines need different SQL. Run `make sqlc` in
  `agency/server` to regenerate the neutral store types and converters.
  **Store methods are use-case operations, not query wrappers**: a method may
  run a transaction and use several queries, and the two dialects may
  implement it differently (see `RegisterAgent` / `TransitionTask`). Add the
  method to the `Store` interface and implement it in both stores; don't add a
  Store method just because sqlc generated a query. Never hand-edit `gen/`
  trees, and keep ad-hoc SQL out of handlers.
- **Tasks, no lock table.** Work is a [`task`](VOCABULARY.md#task) the agency
  scheduler drains onto an agent; concurrency is capped per agent (overall,
  per-project, per-tag, per-project-per-tag) by the scheduler. There is no queue
  table and no lock table.
- **Tasks run Pi child processes.** Drive Pi via `pi --mode rpc` JSONL; wait
  for `agent_settled`, not the `prompt` response. stdout = protocol, stderr =
  logs; pass the task's own UUID as `--session-id`. The agent's
  `internal/session` package keeps its name because it wraps a Pi session.
- **Pi tools come from bundled extensions.** RPC mode cannot register a tool,
  so the agent embeds extensions (`//go:embed`), writes them out, and passes
  `--extension`; see [`agent/AGENTS.md`](agent/AGENTS.md#bundled-pi-extensions).
- **The tunnel is a byte pipe.** Agents run yamux over a Connect bidi stream
  and serve their HTTP API; the agency talks to them with generated Connect
  clients. Do not add per-method dispatchers.
- Task workspaces default to `$CHERRYPI_DIR/sessions/<id>` on the agent.
- Use **Mermaid** for diagrams in docs, never ASCII art.

## Agency conventions

- User credentials (`users`) are separate from agent tunnel credentials
  (`agent_credentials`).
- The tunnel authenticates agents itself; the user basic-auth middleware must
  skip that route.
- Forward the authenticated user to agents as `X-CherryPi-User`.

## SDK conventions

- `agency/sdk/js` (`@cherrypi/sdk`) is the SDK. The frontend uses it — **never
  hand-roll API calls**; methods are documented in
  `agency/sdk/js/README.md`.
- Generated protobuf-ES stubs (`agency/sdk/js/src/gen`, gitignored) come from
  `shared/buf.gen.yaml`; regenerate with `buf generate`, never by hand.
- `AgencyClient` is the client; task calls are agency-level (a task id),
  per-agent calls (terminals, usage) take an `agentId`.
