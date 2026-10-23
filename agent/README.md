# CherryPi Agent

The **agent** is the CherryPi worker: a long-running, headless binary on a
machine that should do work. It spawns and drives Pi sessions, hosts browser
terminals and a desktop bridge, and **dials the agency over a single tunnel** —
there is no HTTP listener, no port, and no local login. The agency is its only
client.

Repo-wide context: [`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md),
[`../VOCABULARY.md`](../VOCABULARY.md), and the backend rules in
[`AGENTS.md`](AGENTS.md).

## Build

The generated trees (`internal/gen/sqlc` and `shared/gen`) are gitignored, so
regenerate before building via the Makefile:

```sh
make build # sqlc generate + go build -o bin/cherrypi-agent ./cmd/agent
make test  # clean + sqlc generate + go test ./...
```

Requires Go 1.26 (`/usr/local/go/bin`), `sqlc`, and `buf` (in `shared/`). The
`pi` coding agent must be on `PATH` at runtime (or set `PI_BIN`).

## Run

```sh
./bin/cherrypi-agent \
  --agency-url https://cherrypi.example.com \
  --auth builder:s3cret
```

A `.env` file in the working directory is loaded automatically (via
`godotenv`). Flags override the environment.

### Flags

| Flag | Overrides | Meaning |
| --- | --- | --- |
| `--agency-url` | `CHERRYPI_AGENCY_URL` | Agency URL to tunnel to. Required. |
| `--auth` | `CHERRYPI_AGENT_AUTH` | Legacy tunnel credentials, `user:password` (fallback). |
| `--display` | `DISPLAY` | X display the desktop bridge attaches to. |
| `--xauthority` | `XAUTHORITY` | X authority file for `--display`. |

Register an agent and mint its personal access token (`pat_…`) in the agency
UI, then set `CHERRYPI_AGENT_TOKEN`. Alternatively, `--auth` takes the legacy
`user:password` tunnel credential; the agent exchanges it for a token on first
boot and caches it at `$CHERRYPI_DIR/agent.token` (0600).

### Environment

| Variable | Default | Meaning |
| --- | --- | --- |
| `CHERRYPI_AGENCY_URL` | — | Agency URL (required; `--agency-url`). |
| `CHERRYPI_AGENT_AUTH` | — | Legacy tunnel credentials `user:password` (`--auth`); used to bootstrap a token and as a fallback. |
| `CHERRYPI_AGENT_TOKEN` | — | Personal access token (`pat_…`); wins over the cached file. |
| `CHERRYPI_AGENT_TOKEN_FILE` | `$CHERRYPI_DIR/agent.token` | Cached token file the bundled Pi tools read on every call. |
| `CHERRYPI_DIR` | `~/.cherrypi` | Data dir: SQLite DB, session workspaces, installed Pi extensions. |
| `CHERRYPI_DATABASE_DSN` | `$CHERRYPI_DIR/cherrypi.sqlite3` | SQLite DSN. |
| `CHERRYPI_LOG_FORMAT` | `text` | Stdout log format: `text` or `json`. |
| `CHERRYPI_LOG_LEVEL` | `info` | Minimum level: `debug`, `info`, `warn`, `error`. |
| `DISPLAY` | — | X display for the desktop bridge (`--display`). |
| `XAUTHORITY` | — | X authority file (`--xauthority`). |
| `PI_BIN` | `pi` | Pi executable. |
| `PI_ARGS` | — | Extra space-separated args passed to `pi` after `--mode rpc --session-id`. |
| `PI_SESSION_DIR` | Pi's default | Overrides Pi's session storage (`--session-dir`). |
| `PI_CODING_AGENT_DIR` | `~/.pi/agent` | Pi's agent dir (memory, `settings.json`, …). |
| `SHELL_BIN` | `$SHELL`, then `/bin/bash` | Shell for root terminals. |

Per-task workspaces live at `$CHERRYPI_DIR/sessions/<task id>`. Store internals
(goose migrations + sqlc queries) are under [`sql/`](sql).

## Logging

The agent logs structured `slog` records (level plus `task_id`/`session_id`
attributes) to stdout and mirrors them into its event stream, which the agency
persists in `agent_logs` and shows on the per-agent Logs page. Set
`CHERRYPI_LOG_FORMAT=json` for one JSON object per line. Tunnel connectivity is
logged to stdout only — the agency records the authoritative connect/disconnect
lines. Pi's stderr stays per-task in the task console and is not mirrored.

## Desktop

The **RDP** button on the agent page streams the machine's X session to the
browser (RFB via noVNC). The agent does **not** boot a desktop session: it
attaches `x11vnc` to an already-running display on the first browser attach.

```sh
# Reuse a host X session.
./bin/cherrypi-agent --agency-url … --auth … \
  --display :0 \
  --xauthority "$HOME/.Xauthority"
```

Leave `--display`/`DISPLAY` unset to disable the
feature; the agent page then reports why. In Docker, mount the host's X socket
and authority file — see [`../docs/desktop.md`](../docs/desktop.md) and
[`../compose.desktop.yaml`](../compose.desktop.yaml). GPU acceleration and
session booting are out of scope.

## How it runs

- **Tunnel** — the agent dials `--agency-url`, authenticates with the tunnel
  credential, and serves its HTTP API over a yamux session on a Connect bidi
  stream (`internal/tunnel`). The agency forwards the authenticated user as
  `X-CherryPi-User`.
- **Tasks** — `internal/session` spawns one `pi --mode rpc` child per task,
  streams its events to the agency, and materializes attachments into the
  workspace.
- **Terminals** — `internal/terminal` brokers root PTYs over WebSockets.
- **Desktop** — `internal/desktop` bridges the running X display over a
  WebSocket.
- **Extensions** — three bundled Pi extensions (`setTaskUrls`,
  `setTaskArtifacts`, `uploadAttachment`) live in
  `internal/session/extension_assets`, are embedded with `//go:embed`, and are
  registered with each Pi child (see [`AGENTS.md`](AGENTS.md)).

## Layout

```
cmd/
  agent/          binary (flags, wiring) + appcontext (config, singletons)
  agent/grpc/     Connect service handlers (task, terminal, desktop, …)
internal/
  session/        Pi child lifecycle, attachments, usage, bundled extensions
  terminal/       root PTYs and their WebSocket data plane
  desktop/        x11vnc bridge to the running X session
  dashboard/      disk/host stats for the agency
  events/         in-process event hub
  tunnel/         yamux tunnel client
  store/          SQLite store
  auth/           forwarded `X-CherryPi-User` context
sql/              goose migrations + sqlc queries
```

## Test

```sh
go test ./...                   # all packages
go test ./internal/desktop/...  # the desktop bridge
cd ../agency/sdk/js && npm test # SDK integration (spawns this agent)
```

No Docker daemon is needed for the Go tests; the SDK suite builds and runs the
binary against the mock LLM.
