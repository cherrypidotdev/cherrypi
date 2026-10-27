---
title: Install an agent
description: Run a CherryPi agent from the container image or a release binary, register it, and connect it to your agency.
---

# Install an agent

An agent is a headless worker. It has **no inbound port**: it dials the agency,
authenticates, and serves its task API over the tunnel. Install it on any
machine that should run Pi sessions — a laptop, a build box, or a cloud VM.

## Before you start

- A running agency URL, reachable from the agent host.
- A **personal access token** (`pat_…`) for a registered agent.
- Enough disk for `$CHERRYPI_DIR` — per-task workspaces and Pi sessions live
  there.
- Pi model credentials, if you use the binary rather than the image. The image
  bundles Pi but not your provider keys.

## Register the agent

1. Open **Agents → Register agent**.
2. Give it a name and, optionally, a description.
3. Copy the one-time personal access token shown after creation.
4. Store it as `CHERRYPI_AGENT_TOKEN` on the agent host, or mint a new token
   later from **Agent → Settings → Security → Personal access tokens**.

There is no username/password tunnel credential and no startup seed. A new
agent cannot connect until it has a PAT.

## Option A — Docker image (recommended)

The published agent image includes Pi, Node, X utilities and `x11vnc`.

```sh
docker run -d --name cherrypi-agent \
  --restart unless-stopped \
  -v /srv/cherrypi/agent:/root/.cherrypi \
  -v /srv/cherrypi/pi:/root/.pi/agent \
  -e CHERRYPI_AGENCY_URL='https://cherrypi.example.com' \
  -e CHERRYPI_AGENT_TOKEN='pat_…' \
  ghcr.io/cherrypidotdev/cherrypi-agent:latest
```

| Mount | Why |
| --- | --- |
| `/root/.cherrypi` | Agent database, cached token and task workspaces. |
| `/root/.pi/agent` | Pi configuration, model/provider credentials, memory. |

The agent caches the token at `/root/.cherrypi/agent.token` (mode `0600`).
Keep the data volume: it is the agent's identity cache. You can also pass the
token as `--token pat_…`; an explicit token wins over the cached file, so
token rotation is a container/process restart.

### Compose

```yaml
services:
  agent:
    image: ghcr.io/cherrypidotdev/cherrypi-agent:latest
    restart: unless-stopped
    environment:
      CHERRYPI_AGENCY_URL: https://cherrypi.example.com
      CHERRYPI_AGENT_TOKEN: pat_…
    volumes:
      - ./agent-data:/root/.cherrypi
      - ./pi-agent:/root/.pi/agent
```

## Option B — release binary

Download `cherrypi-agent-<os>-<arch>-<sha>.zip` from the
[releases page](https://github.com/cherrypidotdev/cherrypi/releases) and unpack
it:

```sh
unzip cherrypi-agent-linux-amd64- < sha > .zip
install -m 0755 cherrypi-agent /usr/local/bin/cherrypi-agent
```

The binary needs:

- Go is **not** required at runtime.
- **Pi** on `PATH` (or `PI_BIN=/path/to/pi`). Install the Pi coding agent
  version compatible with your CherryPi release.
- Your model credentials in `$PI_CODING_AGENT_DIR` (default `~/.pi/agent`).

Run it once to verify:

```sh
CHERRYPI_AGENCY_URL=https://cherrypi.example.com \
  CHERRYPI_AGENT_TOKEN=pat_… \
  cherrypi-agent
```

### systemd

`/etc/systemd/system/cherrypi-agent.service`:

```ini
[Unit]
Description=CherryPi agent
After=network-online.target
Wants=network-online.target

[Service]
User=cherrypi
Group=cherrypi
Environment=CHERRYPI_AGENCY_URL=https://cherrypi.example.com
Environment=CHERRYPI_AGENT_TOKEN=pat_…
Environment=CHERRYPI_DIR=/var/lib/cherrypi
Environment=PI_CODING_AGENT_DIR=/var/lib/cherrypi/.pi/agent
ExecStart=/usr/local/bin/cherrypi-agent
Restart=on-failure
RestartSec=5
TimeoutStopSec=120

[Install]
WantedBy=multi-user.target
```

```sh
useradd --system --home /var/lib/cherrypi --create-home cherrypi
systemctl daemon-reload
systemctl enable --now cherrypi-agent
journalctl -u cherrypi-agent -f
```

The service must be able to run Pi and write to `CHERRYPI_DIR`. If you use root
terminals, the service user is the shell user — treat that as privileged access.

## Verify the connection

The agent page in the agency shows **Online** after the tunnel handshake. The
agent's own log shows:

```text
CherryPi agent <version> starting
Tunneling to agency https://cherrypi.example.com
```

If it stays offline, see [Troubleshooting](/advanced/troubleshooting#the-agent-stays-offline).

## Network requirements

| Direction | Port | Why |
| --- | --- | --- |
| Agent → agency | `443` (or the agency port) | Tunnel, task API, history, attachments. |
| Agent → model provider | provider-defined | Pi's model calls. |
| Browser → agency | `443` | The dashboard. |

The agent needs **no inbound port**. Its root terminal is reachable only through
the agency's authenticated tunnel.

## Desktop access (optional)

The image contains `x11vnc` and X utilities. To expose an already-running X
display, pass `DISPLAY` and mount the socket and authority file. See
[Desktop access](/agent/desktop).

## Update an agent

Pull the new image or replace the binary, then restart the service. The agent
reconnects, reports its new version, and recovers interrupted tasks. Check the
compatibility badge on the agent page if the agency is much newer.

## Next

- [Configure the agent](/agent/configuration)
- [Using agents](/using/agents)
- [Desktop access](/agent/desktop)
