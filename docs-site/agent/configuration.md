---
title: Configure the agent
description: Flags, environment variables, token loading and storage for a CherryPi agent.
---

# Configure the agent

The agent reads flags, environment variables and an optional `.env` file in the
working directory (loaded automatically). Flags override the environment.

## Flags

| Flag | Overrides | Meaning |
| --- | --- | --- |
| `--agency-url` | `CHERRYPI_AGENCY_URL` | Agency URL to tunnel to. Required. |
| `--token` | `CHERRYPI_AGENT_TOKEN` | Personal access token (`pat_…`). |
| `--display` | `DISPLAY` | X display for the desktop bridge. |
| `--xauthority` | `XAUTHORITY` | X authority file for `--display`. |

## Environment

| Variable | Default | Meaning |
| --- | --- | --- |
| `CHERRYPI_AGENCY_URL` | — | Agency URL (required). |
| `CHERRYPI_AGENT_TOKEN` | — | Personal access token (`pat_…`). `--token` overrides it; it wins over the cached file. |
| `CHERRYPI_AGENT_TOKEN_FILE` | `$CHERRYPI_DIR/agent.token` | Cached token file read by the bundled Pi tools on every call. |
| `CHERRYPI_DIR` | `~/.cherrypi` | Data directory: SQLite database, token, per-task workspaces. |
| `CHERRYPI_DATABASE_DSN` | `$CHERRYPI_DIR/cherrypi.sqlite3` | Local agent database. |
| `CHERRYPI_AGENT_TUNNEL_TRANSPORT` | `auto` | `auto`, `http2` or `websocket`. Diagnostic only. |
| `CHERRYPI_LOG_FORMAT` | `text` | `text` or `json` per-line logs. |
| `CHERRYPI_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |
| `PI_BIN` | `pi` | Pi executable. |
| `PI_ARGS` | — | Extra space-separated arguments passed to Pi after `--mode rpc --session-id`. |
| `PI_SESSION_DIR` | Pi's default | Override Pi's session storage. |
| `PI_CODING_AGENT_DIR` | `~/.pi/agent` | Pi's agent directory: memory, `settings.json`, provider credentials. |
| `DISPLAY` | — | X display for the desktop bridge. |
| `XAUTHORITY` | — | X authority file. |
| `SHELL_BIN` | `$SHELL`, then `/bin/bash` | Shell used for root terminals. |

## Tokens

A personal access token is the agent's only credential. There are two paths,
in order of preference:

1. **`--token` / `CHERRYPI_AGENT_TOKEN`** — an explicit token. It is fixed: a
   rotated token file does not override it. Restart the agent to apply a new
   token.
2. **Cached token** — `CHERRYPI_AGENT_TOKEN_FILE`, default
   `$CHERRYPI_DIR/agent.token`. The agent validates it at startup and discards
   it if the agency rejects it. The bundled Pi tools re-read it on every call,
   so a rotated file applies without respawning Pi.

There is no username/password fallback. If both sources are missing, the agent
exits with instructions.

Treat the data directory as a secret: the cached token grants agent access.

## Data directory

```text
$CHERRYPI_DIR/
  cherrypi.sqlite3        # local task/usage cache
  agent.token             # cached personal access token (0600)
  sessions/<task id>/     # per-task workspaces and attachments
```

Per-task workspaces can grow large. Put `CHERRYPI_DIR` on a disk with room for
repositories, dependencies and build output. The agency stores the Pi session
history, so a workspace can be recreated by re-running a task; but uncommitted
files that are not attached or published as artifacts will be lost if the
directory is deleted.

## Pi configuration

The agent does not configure models. Pi does, through
`PI_CODING_AGENT_DIR` (default `~/.pi/agent`). In the container, mount that
directory from the host so model credentials and Pi memory survive restarts.

```sh
-v /srv/cherrypi/pi:/root/.pi/agent
```

The image already installs the Pi extensions CherryPi expects. If you run the
binary yourself, install a compatible Pi version and the extensions listed in
the release notes.

## Logging

Logs go to stdout, and a mirror of non-local records is sent to the agency for
the **Logs** tab on the agent page.

```sh
CHERRYPI_LOG_FORMAT=json
CHERRYPI_LOG_LEVEL=debug
```

Pi's own stderr stays local and reaches the agency as task console output.

## Concurrency limits

The agency, not the agent, decides how many tasks an agent may run. Choose the
agent in **Agents → Limits** to set:

- the total concurrent task cap;
- a per-project cap;
- per-tag and per-project-tag caps.

Defaults are deliberately conservative (`1` total, `1` per project). Raise them
only when the machine has CPU, RAM and provider rate limits to spare.

## Tunnel transport

Leave `CHERRYPI_AGENT_TUNNEL_TRANSPORT` unset in normal deployments. `auto`
tries HTTP/2 and falls back to a WebSocket over HTTP/1.1. The explicit values
are diagnostic: `http2` or `websocket`.

If a reverse proxy breaks the tunnel, do not force h2c in front of the agency;
fix the proxy to allow WebSockets and disable read timeouts.

## Desktop

Set `DISPLAY` (and usually `XAUTHORITY`) to expose an existing X session. Leave
them unset to disable the desktop. See [Desktop access](/agent/desktop).

## Next

- [Desktop access](/agent/desktop)
- [Using agents](/using/agents)
- [Environment reference](/reference/environment)
