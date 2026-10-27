---
title: Configure the agency
description: Environment variables, database, object storage and dashboard settings for the CherryPi agency.
---

# Configure the agency

The agency is configured through environment variables and, once running,
through **Settings** in the dashboard. Prefer the UI for organization settings:
they are stored in the database and apply immediately.

## Environment at a glance

| Variable | Default | Purpose |
| --- | --- | --- |
| `CHERRYPI_AGENCY_DIR` | `~/.cherrypi-agency` | Data directory: SQLite database and process logs. |
| `CHERRYPI_AGENCY_PORT` | `8080` | HTTP listen port. |
| `CHERRYPI_AGENCY_DATABASE_DSN` | `$CHERRYPI_AGENCY_DIR/agency.sqlite3` | SQLite path or PostgreSQL URL. |
| `CHERRYPI_PUBLIC_URL` | `http://localhost:<port>` | External origin used in links and OIDC callbacks. |
| `CHERRYPI_ADMIN_USERNAME` | `admin` | First-boot administrator username. |
| `CHERRYPI_ADMIN_PASSWORD` | generated + printed once | First-boot administrator password. |
| `CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN` | off | Open registration where the first user becomes admin. |
| `CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS` | off | Return magic links in API responses for SMTP-less installs. Never enable in production. |
| `CHERRYPI_S3_ENDPOINT` | — | S3-compatible endpoint. Enables storage with the bucket. |
| `CHERRYPI_S3_BUCKET` | — | Bucket name. |
| `CHERRYPI_S3_REGION` | `us-east-1` | Bucket region. |
| `CHERRYPI_S3_ACCESS_KEY_ID` | — | Access key. |
| `CHERRYPI_S3_SECRET_ACCESS_KEY` | — | Secret key. |
| `CHERRYPI_S3_FORCE_PATH_STYLE` | off | Use path-style requests (`https://host/bucket/...`). |
| `CHERRYPI_S3_KEY_PREFIX` | — | Optional prefix for every object. |
| `CHERRYPI_ATTACHMENT_MAX_BYTES` | `26214400` (25 MiB) | Per-file upload limit. |
| `CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE` | `10` | Attachments per message. |
| `CHERRYPI_AGENCY_LOG_FILES` | `7` | Process-log files to retain. |
| `CHERRYPI_H2C` | off | Serve cleartext HTTP/2. Diagnostic/direct-agent use only. |
| `CHERRYPI_FRONTEND_PROXY_URL` | — | Development only: proxy the SPA to Vite. |
| `CHERRYPI_DOCS_PROXY_URL` | — | Development only: proxy `/docs` to VitePress. |

The complete list, including S3 region validation and storage failure hints, is
in the [environment reference](/reference/environment).

## Database

SQLite is the default and runs with no extra service. The database file lives
in `CHERRYPI_AGENCY_DIR`.

To use PostgreSQL, provide a URL:

```sh
CHERRYPI_AGENCY_DATABASE_DSN='postgres://cherrypi:secret@db:5432/cherrypi?sslmode=disable'
```

The agency applies its migrations automatically on boot. Use a dedicated
database and role; the agency needs ownership of its schema. Back up the
database together with the S3 bucket — attachments live in the bucket, not in
the database.

::: tip When to choose PostgreSQL
A single agency on one host is fine with SQLite. Choose PostgreSQL when you
need managed backups, point-in-time recovery, or a database shared with
operational tooling.
:::

## Object storage

Attachments, avatars and task artifacts are stored in one bucket. The agency
proxies every upload and download, so the bucket does not need to be public.

```sh
CHERRYPI_S3_ENDPOINT=https://s3.example.com
CHERRYPI_S3_BUCKET=cherrypi
CHERRYPI_S3_REGION=eu-central-1
CHERRYPI_S3_ACCESS_KEY_ID=…
CHERRYPI_S3_SECRET_ACCESS_KEY=…
```

At startup the agency calls `HeadBucket` and verifies the reported region. A
wrong endpoint, bucket or region fails the boot with an actionable error rather
than producing broken attachments later.

For MinIO or any endpoint served through a proxy without wildcard TLS, use
path-style addressing:

```sh
CHERRYPI_S3_FORCE_PATH_STYLE=true
```

The bucket must exist before the agency starts. With the bundled compose file,
the `minio-init` service creates it.

## Dashboard settings

**Settings** is available to administrators and is stored in the database.

| Section | What you configure |
| --- | --- |
| General | Agency name, logo, favicon and primary color. |
| Users | Invite people, change roles, disable accounts, resend invitations. |
| Authentication | Password/magic-link toggles, TOTP policy, SMTP, OIDC and LDAP providers. |
| Agents | Who may create agents, default concurrency behavior. |
| Storage | S3 status, upload limits and object retention. |
| Logs | The agency process log with follow, download and tail. |

See [Authentication and teams](/agency/authentication) for the identity model.

## Data directory

Everything the agency writes locally lives under `CHERRYPI_AGENCY_DIR`:

```text
$CHERRYPI_AGENCY_DIR/
  agency.sqlite3      # when using SQLite
  logs/               # per-run, timestamped process logs
```

The process log is also shown at **Settings → Logs**. It is safe to delete old
files; the agency keeps `CHERRYPI_AGENCY_LOG_FILES` of them.

## Development proxies

Release builds always serve the embedded dashboard and docs. A `-tags dev` build
can proxy them to their Vite servers instead:

```sh
CHERRYPI_FRONTEND_PROXY_URL=http://localhost:5173 \
  CHERRYPI_DOCS_PROXY_URL=http://localhost:5174 \
  go run -tags dev ./cmd/agency
```

These variables are ignored by release builds.

## Reverse proxy requirements

- Forward `Host` and `X-Forwarded-Proto` so links and cookies are correct.
- Do not terminate HTTP/2 into the agency over cleartext (`h2c`).
- Disable read timeouts for long-lived WebSocket and streaming routes.
- Keep `CHERRYPI_PUBLIC_URL` in sync with the public origin.

See [Install the agency](/agency/install#reverse-proxy) for a working Caddy
configuration.
