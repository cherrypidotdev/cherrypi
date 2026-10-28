---
title: Troubleshooting
description: Diagnose agency boot failures, offline agents and streaming problems.
---

# Troubleshooting

Start with the process log: **Settings → Logs** on the agency, the **Logs** tab
on the agent, and the task console for task-level failures.

## The agency does not start

| Symptom | Likely cause |
| --- | --- |
| `Embedded frontend unavailable` | A source build without the SPA; run `make build` or use a release binary. |
| `Embedded docs unavailable` | A source build without the docs; run `make docs copy_docs`. |
| `Invalid CHERRYPI_*` | A malformed environment value; the log names the variable. |
| Storage error at boot | Endpoint, bucket, region or credentials are wrong; read the hint in the error. |

### Storage boot check

The agency calls `HeadBucket` at startup. A failure aborts the boot with the
bucket, endpoint and addressing style. Common fixes:

- create the bucket first;
- use `CHERRYPI_S3_FORCE_PATH_STYLE=true` for MinIO behind a proxy without
  wildcard TLS;
- set the real region (`CHERRYPI_S3_REGION`), including non-AWS regions;
- grant `s3:ListBucket`/`s3:HeadBucket` to the access key.

## The agent stays offline

1. Is the agent process running? `docker logs cherrypi-agent` or
   `journalctl -u cherrypi-agent`.
2. Does the agent log show `Tunneling to agency …`? If not, the URL is wrong or
   the process exits on startup.
3. Can the host reach the agency?
   `curl -fsS https://cherrypi.example.com/api/rest/health`.
4. Is the token right? Mint a fresh personal access token from the agent page
   and pass it as `--token` or `CHERRYPI_AGENT_TOKEN`. A rejected cached token
   is discarded at startup with an explanatory log line.
5. Is the tunnel blocked by a proxy? Allow WebSockets at `/api/tunnel/ws` and
   disable read timeouts. Do not force h2c upstream.
6. Is the agent version compatible? Check the badge and the agency log.

After a token rotation, restart the agent so the new token is picked up.

## A task does not start

The queued task's page shows its exact waiting reason, for example
`Waiting for tasks to become completed: PRFX-1, PRFX-2` or
`Waiting for free agent slot`.

- **No eligible agent** — the assignee must be a member of the task's project
  and have a free slot. Check project access and the agent's concurrency limits.
- **Agent offline** — the scheduler will not send work to a disconnected agent.
- **Ahead in the queue** — tasks drain in order; check the project's queue.
- **Failed immediately** — read the task console and the agent log; missing Pi
  configuration or model credentials is a common cause.

## WebSockets and streaming fail

Symptoms: terminals close immediately, the inbox does not update, the agent
tunnel drops, desktop disconnects.

- Keep the upstream protocol at HTTP/1.1; do not enable h2c-first.
- Disable the proxy read timeout / buffering for long-lived routes.
- Allow `Upgrade` and `Connection` headers to pass through.
- Put the agency behind a proxy that supports WebSockets. Caddy and nginx do.

## Attachments fail

| Error | Fix |
| --- | --- |
| `storage is not configured` | Set `CHERRYPI_S3_ENDPOINT` and `CHERRYPI_S3_BUCKET`. |
| `403` from the bucket | Check keys, bucket policy and region. |
| Upload too large | Raise `CHERRYPI_ATTACHMENT_MAX_BYTES` or lower the file size. |
| Images missing in Pi | Ensure the agent can download attachments; check its token. |

## Magic links or invitations do not arrive

Configure SMTP under **Settings → Authentication**. Without SMTP, the UI shows
a copyable link. Do not enable `CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS` in production.

## Login loops or secure cookies

- `CHERRYPI_PUBLIC_URL` must match the external origin, including the scheme.
- Terminate TLS at the proxy and forward `X-Forwarded-Proto`.
- Clear stale cookies after changing the hostname.

## Docs return 404

Release builds embed `/docs`. A development build without
`CHERRYPI_DOCS_PROXY_URL` has no docs assets; run the docs dev server and set
the proxy, or run `make docs copy_docs` before `make build_agency`.

## Getting help

Open an issue with:

- the agency version (footer or startup log);
- the agent version and protocol status;
- the relevant log excerpt;
- your reverse proxy and database/storage backend.

Use [GitHub Issues](https://github.com/cherrypidotdev/cherrypi/issues).
