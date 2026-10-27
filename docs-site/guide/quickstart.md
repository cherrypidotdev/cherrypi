---
title: Quickstart
description: Run CherryPi locally from the repository, or connect your first agent to an existing agency.
---

# Quickstart

There are two ways to get started. Use the **single-box stack** to try CherryPi
on one machine, or **install the agency and an agent separately** for a real
deployment.

## Option 1 — the single-box stack

The repository ships a `compose.yaml` that builds and runs an agency, an agent,
and a local MinIO bucket for attachments.

**Requirements:** Docker with Compose v2, plus a checkout of the repository.

```sh
git clone https://github.com/cherrypidotdev/cherrypi.git
cd cherrypi
docker compose up -d --build agency minio minio-init
```

Open <http://localhost:8080> and sign in as `admin` with the demo password
`deadbeef`. Change it before exposing the stack, or choose your own password
on the first boot:

```sh
CHERRYPI_ADMIN_PASSWORD='choose-a-long-password' \
  docker compose up -d --build agency minio minio-init
```

Register the bundled agent next: open **Agents → Register agent**, create
`builder`, and copy its one-time personal access token. Put it in a local
`.env` file and start the agent:

```sh
echo 'CHERRYPI_AGENT_TOKEN=pat_…' >> .env
docker compose up -d agent
```

The stack no longer seeds a username/password agent credential. Do not use the
demo admin password in production.

::: tip What is running?

- `agency` — the web app, API and tunnel hub on port `8080`
- `agent` — a worker connected to the agency
- `minio` — S3-compatible storage on ports `9000`/`9001`
  :::

To stop everything and keep the data:

```sh
docker compose down
```

To start over, delete the data directories first:

```sh
docker compose down -v
rm -rf agency_home cherrypi_home minio_data
```

## Option 2 — install each piece

### 1. Install the agency

```sh
docker run -d --name cherrypi-agency \
  --restart unless-stopped \
  -p 8080:8080 \
  -v "$PWD/agency_data:/root/.cherrypi-agency" \
  -e CHERRYPI_ADMIN_PASSWORD='choose-a-long-password' \
  -e CHERRYPI_PUBLIC_URL='https://cherrypi.example.com' \
  ghcr.io/cherrypidotdev/cherrypi-agency:latest
```

Put a TLS reverse proxy in front of it. Read
[Install the agency](/agency/install) before exposing it to the internet.

### 2. Create an account and sign in

Open the agency URL and sign in as `admin` with the password you set. Invite
other people from **Settings → Users**, or let them use OIDC/LDAP if you
configure a provider.

### 3. Register an agent

In the agency, open **Agents → Register agent** and give it a name. Copy the
one-time personal access token shown after creation.

### 4. Install the agent

On the worker machine:

```sh
docker run -d --name cherrypi-agent \
  --restart unless-stopped \
  -v "$PWD/cherrypi_data:/root/.cherrypi" \
  -e CHERRYPI_AGENCY_URL='https://cherrypi.example.com' \
  -e CHERRYPI_AGENT_TOKEN='pat_…' \
  ghcr.io/cherrypidotdev/cherrypi-agent:latest
```

The agent dials out with that token and caches it locally. It needs no open
port. See [Install an agent](/agent/install) for the binary, systemd and
desktop options.

### 5. Create a project and a task

1. Open **Projects** and create a project.
2. Add a team to the project or keep it private to you.
3. Open the project and create a task, choosing the agent.
4. Write a prompt. The scheduler starts it as soon as the agent has capacity.

The task page shows the live console, attachments, artifacts, session history
and the agent's terminal.

## Where to go next

- [Concepts](/guide/concepts) — agency, agent, project, task, prompt, artifact.
- [Configure the agency](/agency/configuration) — database, storage, auth.
- [Configure the agent](/agent/configuration) — tokens, data dir, desktop.
- [Troubleshooting](/advanced/troubleshooting) — when the agent stays offline.
