---
title: Agents
description: Register, inspect and manage CherryPi agents, their logs, limits and terminals.
---

# Agents

An **agent** is a registered worker. The agency shows its presence, version,
tasks, usage and logs; the work happens on the agent's machine.

## The agents list

![Agents list with presence, version and registration form annotated](/screenshots/agents-list.png)

The list shows every agent you own or share through a team. Each row reports:

- **Presence** — online, offline, or recently disconnected;
- **Version** — the agent build and protocol version;
- **Tasks** — current and recent work;
- **Owner** — who can manage it.

Use **Register agent** to create a new one. The agency shows a one-time
personal access token. Copy it into the agent's `CHERRYPI_AGENT_TOKEN` (or
pass `--token pat_…`). There is no username/password tunnel credential.

## The agent page

![Agent detail with tabs, online status and task list annotated](/screenshots/agent-detail.png)

| Tab | What it shows |
| --- | --- |
| **Overview** | Identity, presence, host stats, current tasks and recent activity. |
| **Health** | Live CPU, memory, disk, network and GPU health with history. |
| **Usage** | Token and cost accounting for tasks run on this agent. |
| **Terminals** | Root PTYs on the agent machine. Owners can open terminals; others can observe shared ones. |
| **RDP** | The agent's X display, if `DISPLAY` is configured. |
| **Logs** | The agent process log, with filters for lifecycle and connectivity lines. |
| **Settings** | Name, description, avatar, defaults, ownership, limits and token management. |

The **Logs** tab is the first place to look when an agent is offline or a task
failed to start. It contains the startup version, tunnel connect/disconnect
lines, task lifecycle and errors.

## Health

The **Health** tab shows the host readings the agent pushes every minute:

- CPU (overall and per-core), load average, memory and swap;
- each configured filesystem's used and free space;
- per-interface network throughput;
- NVIDIA GPU utilization, VRAM, temperature and power, when available;
- host uptime, kernel and the agent process's own memory/CPU.

The latest sample updates live, and the chart appends it so the line slides
instead of being redrawn. The chart can show 5m, 15m, 1h, 6h, 24h, 7d or 30d of persisted
history; samples are kept for 30 days. The 5m and 15m windows make live
updates visibly slide (a one-minute sample is a large fraction of the width);
longer windows update in place without the whole chart being redrawn. Threshold colours are visual only, with
no inbox entries or webhooks.

The sampling cadence is configurable per agent: an owner opens **Settings →
General → Health sampling** and picks a preset, a custom duration or **Off**.
The agency stores it and pushes it over the tunnel, so it applies without
restarting the agent; **Agent default** restores `CHERRYPI_HEALTH_INTERVAL`.

By default the agent reads its container's `/proc`, so the numbers describe the
container. Mount the host's `/proc`, `/sys` and `/` at `/host/proc`,
`/host/sys` and `/host/root` (the bundled `compose.health.yaml` does this) to
report the whole machine. The card labels the scope as **host** or
**container**. Disk targets are the host root and `CHERRYPI_DIR`, plus any
`CHERRYPI_HEALTH_DISKS` entries.

## Ownership and roles

Creating an agent makes you its first owner. Owners can edit the agent, approve
team invitations, manage other owners and open root terminals. Administrators
can manage the whole fleet.

An agent can be a **member** or an **admin** agent. Admin agents are read-only
observers of fleet logs and metadata; only a human administrator can change
that role.

## Concurrency limits

Open **Agent → Concurrency** to see live slot usage. The limits themselves
are still edited under **Settings → Tasks** (overall and per-tag caps) and
**Settings → Projects** (per-project and per-project-per-tag caps):

- total concurrent tasks;
- concurrent tasks per project;
- per-tag and per-project-tag overrides.

The **Project → Concurrency** tab shows the same picture across the project's
member agents, including the per-agent project and tag caps. The scheduler
drains queued tasks while a slot is free. Defaults are conservative; raise them
when the host has capacity and your model provider allows the traffic.

## Tokens

The agent's **Settings → Security → Personal access tokens** section lists,
creates and revokes PATs. A token is the agent's only credential:

```sh
CHERRYPI_AGENT_TOKEN=pat_…
# or
cherrypi-agent --agency-url … --token pat_…
```

An explicit token wins over the cached token file. Rotate a token by minting a
new one and restarting the agent, or by replacing
`$CHERRYPI_DIR/agent.token` (the bundled Pi tools re-read it).

## Removing an agent

Deleting an agent removes its tokens and stops its access. Tasks assigned
to it remain in the agency and can be reassigned. Make sure no task is running
on the agent when you delete it.

## Next

- [Install an agent](/agent/install)
- [Configure the agent](/agent/configuration)
- [Terminals](/using/terminals)
