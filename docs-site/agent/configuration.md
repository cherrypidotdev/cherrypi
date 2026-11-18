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
| `CHERRYPI_AGENT_TOKEN` | — | Personal access token (`pat_…`). `--token` overrides it. |
| `CHERRYPI_DIR` | `~/.cherrypi` | Data directory: SQLite database, token, per-task workspaces. |
| `CHERRYPI_DATABASE_DSN` | `$CHERRYPI_DIR/cherrypi.sqlite3` | Local agent database. |
| `CHERRYPI_AGENT_TUNNEL_TRANSPORT` | `auto` | `auto`, `http2` or `websocket`. Diagnostic only. |
| `CHERRYPI_LOG_FORMAT` | `text` | `text` or `json` per-line logs. |
| `CHERRYPI_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |
| `CHERRYPI_LOG_FILE` | `$CHERRYPI_DIR/logs/agent.log` | Durable copy of the process log; empty disables it. |
| `CHERRYPI_LOG_FILE_MAX_MB` | `8` | Size at which the log file rotates. |
| `CHERRYPI_LOG_FILES` | `3` | Rotated log files kept (`agent.log.1` …). |
| `DISPLAY` | — | X display for the desktop bridge. |
| `XAUTHORITY` | — | X authority file. |
| `SHELL_BIN` | `$SHELL`, then `/bin/bash` | Shell used for root terminals. |
| `CHERRYPI_HEALTH_INTERVAL` | `1m` | Host-health sampling interval. `0s` disables sampling. |
| `CHERRYPI_HEALTH_HOST_PROC` | `/host/proc` when present, else `/proc` | Host `/proc` mount used for CPU, memory, load and network. |
| `CHERRYPI_HEALTH_HOST_SYS` | `/host/sys` when present, else `/sys` | Host `/sys` mount. |
| `CHERRYPI_HEALTH_HOST_ROOT` | `/host/root` when present, else `/` | Host root; the default disk target. |
| `CHERRYPI_HEALTH_DISKS` | host root + `$CHERRYPI_DIR` | Extra filesystem targets, comma-separated `label=/path` or bare paths. |

## Tokens

A personal access token is the agent's only credential, and it comes from
`--token` or `CHERRYPI_AGENT_TOKEN` on every start. Nothing is written to disk,
so rotating a token means restarting the agent with the new value.

There is no username/password fallback. If neither source is set, the agent
exits with instructions.

## Data directory

```text
$CHERRYPI_DIR/
  cherrypi.sqlite3        # local task/usage cache
  sessions/<task id>/     # per-task workspaces and attachments
```

Per-task workspaces can grow large. Put `CHERRYPI_DIR` on a disk with room for
repositories, dependencies and build output. The agency stores the task's
session history, so a workspace can be recreated by re-running a task; but uncommitted
files that are not attached or published as artifacts will be lost if the
directory is deleted.

## Model configuration

The agent configures no models of its own. An administrator declares the
providers and models in the agency's **Settings → Models**; the agency seals each
provider key and proxies every request, so the worker holds no credential and the
declared models (narrowed by grants) are the only ones a task can choose. An
agent with no declared model offers nothing and its tasks fail with "no models
are available".

**OAuth logins are not supported.** The agency stores one API key per provider,
and no OAuth flow is implemented, so a Claude Pro/Max, Codex or Copilot
subscription cannot be used. Configure an API key on the provider page instead.

An edited catalog reaches a running agent:

- **Reset models** in **Agents → Tasks**, next to the model default, re-reads
  the catalog and shows it in that dropdown (the task composer picks it up the
  next time it opens).
- **Restart Pi session** in a task's details drawer drops the task's warm
  runtime, so its next run starts from a freshly read catalog. It is disabled
  while a run is in flight or a questionnaire waits for an answer, because
  restarting would drop that work.

## Health sampling

The agent reports host CPU, load, memory, swap, disk, network, uptime and NVIDIA
GPU readings every minute. To report the machine instead of the container, mount
the host paths read-only:

```yaml
volumes:
  - /proc:/host/proc:ro
  - /sys:/host/sys:ro
  - /:/host/root:ro
```

The agent auto-detects these mounts and marks the sample `host`; without them it
marks it `container`. The agent page's **Health** tab shows the data and keeps
30 days of history in the agency. See
[Using agents](/using/agents#health).

`CHERRYPI_HEALTH_INTERVAL` is the **local default**. An agent owner can override
it per agent in **Settings → General → Health sampling**; the agency stores the
value and pushes it over the tunnel when the agent connects and whenever it
changes, so no restart is needed. Accepted values are 10 seconds to 24 hours
(presets, a custom duration, or off); clearing the setting returns the agent to
this environment variable.

## Logging

Logs go to stdout, and a mirror of non-local records is sent to the agency for
the **Logs** tab on the agent page.

```sh
CHERRYPI_LOG_FORMAT=json
CHERRYPI_LOG_LEVEL=debug
```

The mirror needs a live tunnel, so every line is also appended to
`$CHERRYPI_DIR/logs/agent.log` (rotated by size). Read that file when the
agency's copy has a gap — after a tunnel outage or an agent restart, which is
exactly when the log matters. `docker exec <container> tail -f
/root/.cherrypi/logs/agent.log` is the usual entry point.

Task history sync is logged with its reason: a failed push names the agency's
answer and the backoff, and a session that gives up warns
`task history left unsynced` with the number of entries the agency never
received.

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
