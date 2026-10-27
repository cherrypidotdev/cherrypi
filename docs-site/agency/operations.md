---
title: Operate & upgrade
description: Backups, upgrades, logs, retention and a production checklist for the CherryPi agency.
---

# Operate & upgrade

## Backups

A complete backup has two parts:

1. **The database** — SQLite file or PostgreSQL dump.
2. **The S3 bucket** — attachments, avatars and artifacts.

The data directory also contains process logs, which are useful for debugging
but not required for restore.

SQLite, with the agency stopped:

```sh
systemctl stop cherrypi-agency
tar -czf cherrypi-agency-$(date +%F).tar.gz \
  -C /var/lib/cherrypi agency
systemctl start cherrypi-agency
```

PostgreSQL:

```sh
pg_dump --format=custom --file=cherrypi.dump 'postgres://…'
```

MinIO / S3: use the provider's versioning and lifecycle tools, or mirror the
bucket with `mc mirror` / `aws s3 sync`.

::: tip Test restores
A backup you have never restored is a guess. Restore the database and bucket
into a staging agency and confirm that attachments and task history open.
:::

## Upgrades

Docker:

```sh
docker pull ghcr.io/cherrypidotdev/cherrypi-agency:latest
docker compose up -d agency
```

Binary:

```sh
systemctl stop cherrypi-agency
install -m 0755 cherrypi-agency /usr/local/bin/cherrypi-agency
systemctl start cherrypi-agency
```

Migrations run automatically on startup. They are forward-only: to roll back,
restore the backup taken before the upgrade.

Upgrade agents separately. The agency reports each agent's build and protocol
version; an outdated agent may still run tasks but can miss newer features. If
an agent is several protocol versions behind, the agency marks it incompatible
and stops scheduling to it.

## Logs

- **Agency process log** — **Settings → Logs**, also stored as timestamped
  files under `$CHERRYPI_AGENCY_DIR/logs`. Follow it live or download it.
- **Agent process log** — the **Logs** tab on the agent's page. It includes
  startup, connectivity, task lifecycle and failures.
- **Task console** — the live Pi transcript, including tool calls and errors.
- **Container logs** — `docker logs cherrypi-agency` for boot failures that
  happen before the process log is available.

Set `CHERRYPI_AGENCY_LOG_FILES` to control how many process-log files are kept.

## Retention

Administrators can configure retention for logs and history under
**Settings**. Pruning removes old operational records, not tasks or user data.
Artifacts and attachments are removed with their owning task or by the storage
lifecycle rules you configure on the bucket.

## Resource sizing

| Deployment | Suggested |
| --- | --- |
| Personal / small team | 1 vCPU, 1 GB RAM, SQLite, 10 GB disk + bucket. |
| Team with many concurrent tasks | 2 vCPU, 2–4 GB RAM, PostgreSQL, managed bucket. |

The agency streams large terminal and desktop payloads but does not run models
or execute task code. Scale the machines running agents for CPU, RAM and disk.

## Production checklist

- [ ] TLS is terminated by a reverse proxy; the agency port is not public.
- [ ] `CHERRYPI_PUBLIC_URL` is the real HTTPS origin.
- [ ] The bootstrap administrator password was changed.
- [ ] SMTP and/or an identity provider is configured; at least one admin can
      always sign in.
- [ ] The S3 bucket is private, versioned, and backed up.
- [ ] The database and bucket are backed up on a schedule, and a restore was
      tested.
- [ ] Reverse-proxy read timeouts are disabled for long-lived streams.
- [ ] The agency and all agents run supported versions.
- [ ] Agent data directories are on persistent storage.
- [ ] Network egress from agents to the agency is allowed; no inbound agent
      port is exposed.

## Troubleshooting

See [Troubleshooting](/advanced/troubleshooting) for the common failure modes:
agency boot and storage errors, agents staying offline, WebSocket failures
through a proxy, and missing attachments.
