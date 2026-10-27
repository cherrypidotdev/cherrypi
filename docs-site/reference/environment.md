---
title: Agency environment
description: Complete environment reference for the CherryPi agency.
---

# Agency environment

All variables are optional unless marked **required**. Values are read at
startup; restart the agency after changing them.

## Core

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_AGENCY_DIR` | `~/.cherrypi-agency` | Data directory for the SQLite database and process logs. |
| `CHERRYPI_AGENCY_PORT` | `8080` | HTTP listen port. |
| `CHERRYPI_PUBLIC_URL` | `http://localhost:<port>` | External origin without a trailing slash. Used for links, cookies and OIDC callbacks. |
| `CHERRYPI_H2C` | off | Enable cleartext HTTP/2 on the listener. Diagnostic/direct-agent use only; do not force it through a proxy. |
| `CHERRYPI_AGENCY_LOG_FILES` | `7` | Number of timestamped process-log files to keep. |

## Database

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_AGENCY_DATABASE_DSN` | `$CHERRYPI_AGENCY_DIR/agency.sqlite3` | SQLite path, or a `postgres://` / `postgresql://` URL. |

Migrations run on boot. Use a dedicated database and role; the agency needs
schema ownership.

## First boot and bootstrap

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_ADMIN_USERNAME` | `admin` | First administrator username. |
| `CHERRYPI_ADMIN_PASSWORD` | generated, printed once | First administrator password. |
| `CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN` | off | Open registration where the first user becomes admin. |
| `CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS` | off | Return generated links in API responses. Never enable publicly. |

## Object storage

Storage is enabled when both the endpoint and the bucket are set.

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_S3_ENDPOINT` | — | S3-compatible endpoint. A bare host gets `https://`. |
| `CHERRYPI_S3_BUCKET` | — | Bucket name. Must exist. |
| `CHERRYPI_S3_REGION` | `us-east-1` | Bucket region; validated at startup. |
| `CHERRYPI_S3_ACCESS_KEY_ID` | — | Access key ID. |
| `CHERRYPI_S3_SECRET_ACCESS_KEY` | — | Secret access key. |
| `CHERRYPI_S3_FORCE_PATH_STYLE` | off | Path-style requests: `https://host/bucket/...`. Use for MinIO behind a proxy. |
| `CHERRYPI_S3_KEY_PREFIX` | — | Prefix for every object key; a trailing slash is added. |

The agency verifies the bucket at startup with `HeadBucket`, checks the reported
region and fails the boot with an actionable error.

## Attachments

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_ATTACHMENT_MAX_BYTES` | `26214400` (25 MiB) | Maximum size of one uploaded file. |
| `CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE` | `10` | Maximum attachments per message. |

## Development only

Ignored by release builds.

| Variable | Description |
| --- | --- |
| `CHERRYPI_FRONTEND_PROXY_URL` | Proxy the SPA to the Vite dev server. |
| `CHERRYPI_DOCS_PROXY_URL` | Proxy `/docs` to the VitePress dev server. |

## Organization settings

Authentication providers (OIDC, LDAP), SMTP, branding, retention and policy
toggles are stored in the database and configured in the dashboard, not through
environment variables. See [Configure the agency](/agency/configuration) and
[Authentication](/agency/authentication).

## Agent environment

The agent has its own variables and flags. See
[Configure the agent](/agent/configuration#environment).
