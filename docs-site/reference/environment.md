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

## Sites

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_SITES_INSECURE_HTTP` | off | Serve raw site content without any credential on plain HTTP when the host is not localhost. Only needed for direct unauthenticated access (the normal viewer uses short-lived URL grants); makes the static bytes public by URL. Never enable on an Internet-facing deployment. |

## Observability (OpenTelemetry)

The agency exports traces, metrics and logs over OTLP. It follows the standard
OpenTelemetry environment variables, so any collector or backend that speaks
OTLP works without CherryPi-specific settings.

| Variable | Default | Description |
| --- | --- | --- |
| `CHERRYPI_OTEL_ENABLED` | auto | Turn the pipeline on (`1`) or off (`0`). When unset, an OTLP endpoint in the variables below enables it automatically. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Collector base URL, e.g. `http://otel-collector:4317` (gRPC) or `http://otel-collector:4318` (HTTP). |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `grpc` | `grpc` or `http/protobuf`. Per-signal overrides (`OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`, `..._METRICS_PROTOCOL`, `..._LOGS_PROTOCOL`) win over it. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | — | Signal-specific endpoint. The same pattern exists for metrics and logs. |
| `OTEL_SERVICE_NAME` | `cherrypi-agency` | Service name on every signal. |
| `OTEL_RESOURCE_ATTRIBUTES` | — | Extra resource attributes, e.g. `deployment.environment=prod`. |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | Standard sampler (`always_on`, `always_off`, `traceidratio`, `parentbased_traceidratio`, …). |
| `OTEL_TRACES_SAMPLER_ARG` | — | Ratio for the ratio samplers, e.g. `0.1`. |
| `OTEL_TRACES_EXPORTER` | `otlp` | Set to `none` to disable traces while keeping metrics and logs. |
| `OTEL_METRICS_EXPORTER` | `otlp` | Set to `none` to disable metrics. |
| `OTEL_LOGS_EXPORTER` | `otlp` | Set to `none` to disable the log bridge. |
| `OTEL_METRIC_EXPORT_INTERVAL` | `60000` | Metric export interval in milliseconds. |
| `OTEL_BSP_*` / `OTEL_BLRP_*` | SDK defaults | Batch span/log processor tuning (queue size, delay, timeout, batch size). |

```sh
CHERRYPI_OTEL_ENABLED=1
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_SERVICE_NAME=cherrypi-agency
```

The pipeline instruments:

- every unary Connect/gRPC call (`agency.v1.TaskService/GetTask`, …) with an RPC
  span and the RPC duration histogram. Streaming connections (the agent tunnel,
  the realtime and task watchers) are not traced: their span would last as long
  as the connection;
- every REST request under `/api/` and `/plugins/` with an HTTP
  server span and the HTTP duration histogram;
- every agency SQL statement through `database/sql` (SQLite and PostgreSQL),
  including the `db.query.text`, plus `sql.DB` pool gauges;
- the process's standard log lines as OTel log records, in addition to the
  on-disk process log.

Use a trace viewer that follows the W3C trace context (`traceparent`) to see a
request's full tree: HTTP or RPC span, its SQL children, and any errors. A
sample request tree:

```text
POST /api/rest/auth/login           (HTTP server, ~2 ms)
└── SELECT … FROM users …           (SQL, ~1 ms)
```

The embedded frontend, docs and static site bytes are not traced; they are
files, not agency work.

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
