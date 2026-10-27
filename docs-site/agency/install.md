---
title: Install the agency
description: Deploy the CherryPi agency with Docker or a release binary, behind a TLS reverse proxy.
---

# Install the agency

The agency is a single Go binary with the dashboard and these docs embedded.
It needs a data directory, an HTTP port and — for attachments — an
S3-compatible bucket. It does not need a separate web server or a message
queue.

## Requirements

| Requirement | Notes |
| --- | --- |
| Linux host | `linux/amd64` for the published image; release binaries cover Linux and macOS, amd64 and arm64. |
| 1–2 vCPU, 1 GB RAM | The agency is lightweight; the agents do the heavy work. |
| A domain + TLS | Required for secure cookies and realistic OIDC callbacks. Use Caddy, Traefik, nginx or a load balancer. |
| Optional PostgreSQL | SQLite is the default and is fine for a single agency instance. |
| Optional S3 bucket | Required for attachments, avatars and artifacts. MinIO, Garage, Cloudflare R2 and AWS S3 all work. |

## Option A — Docker image

Published images are available on GHCR:

```sh
docker pull ghcr.io/cherrypidotdev/cherrypi-agency:latest
```

A minimal production container:

```sh
docker run -d --name cherrypi-agency \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /srv/cherrypi/agency:/root/.cherrypi-agency \
  -e CHERRYPI_PUBLIC_URL='https://cherrypi.example.com' \
  -e CHERRYPI_ADMIN_USERNAME='admin' \
  -e CHERRYPI_ADMIN_PASSWORD='use-a-secret-manager' \
  ghcr.io/cherrypidotdev/cherrypi-agency:latest
```

The port is bound to localhost so only the reverse proxy can reach it. Mount the
data directory on persistent storage: it contains the SQLite database and the
process logs.

Add object storage for attachments:

```sh
-e CHERRYPI_S3_ENDPOINT='https://s3.example.com' \
  -e CHERRYPI_S3_BUCKET='cherrypi' \
  -e CHERRYPI_S3_REGION='us-east-1' \
  -e CHERRYPI_S3_ACCESS_KEY_ID='…' \
  -e CHERRYPI_S3_SECRET_ACCESS_KEY='…'
```

::: tip MinIO behind a reverse proxy
If your endpoint is a virtual-host style like `https://s3.example.com`, set
`CHERRYPI_S3_FORCE_PATH_STYLE=true`. The agency requests
`https://<bucket>.s3.example.com/...` otherwise, which needs a wildcard
certificate. The agency validates the endpoint and bucket at startup and fails
with a hint if they are unreachable.
:::

See [Configure the agency](/agency/configuration) for the full environment.

## Option B — release binary

Each release publishes zips named
`cherrypi-agency-<os>-<arch>-<sha>.zip`. Download the one for your host from the
[releases page](https://github.com/cherrypidotdev/cherrypi/releases), then:

```sh
unzip cherrypi-agency-linux-amd64- < sha > .zip
install -m 0755 cherrypi-agency /usr/local/bin/cherrypi-agency
useradd --system --home /var/lib/cherrypi --create-home cherrypi
install -d -o cherrypi -g cherrypi /var/lib/cherrypi/agency
```

Create `/etc/cherrypi/agency.env`:

```sh
CHERRYPI_AGENCY_DIR=/var/lib/cherrypi/agency
CHERRYPI_AGENCY_PORT=8080
CHERRYPI_PUBLIC_URL=https://cherrypi.example.com
CHERRYPI_ADMIN_USERNAME=admin
CHERRYPI_ADMIN_PASSWORD=use-a-secret-manager
CHERRYPI_S3_ENDPOINT=https://s3.example.com
CHERRYPI_S3_BUCKET=cherrypi
CHERRYPI_S3_REGION=us-east-1
CHERRYPI_S3_ACCESS_KEY_ID=…
CHERRYPI_S3_SECRET_ACCESS_KEY=…
```

And a systemd unit, `/etc/systemd/system/cherrypi-agency.service`:

```ini
[Unit]
Description=CherryPi agency
After=network-online.target
Wants=network-online.target

[Service]
User=cherrypi
Group=cherrypi
EnvironmentFile=/etc/cherrypi/agency.env
ExecStart=/usr/local/bin/cherrypi-agency
Restart=on-failure
RestartSec=3
# Long-lived streams need generous timeouts.
TimeoutStopSec=90

[Install]
WantedBy=multi-user.target
```

```sh
systemctl daemon-reload
systemctl enable --now cherrypi-agency
journalctl -u cherrypi-agency -f
```

## First boot

On an empty installation the agency creates the first administrator from
`CHERRYPI_ADMIN_USERNAME` (default `admin`) and `CHERRYPI_ADMIN_PASSWORD`. When
the password is not set, a random one is generated and printed **once** to the
agency log:

```sh
docker logs cherrypi-agency 2>&1 | grep -i password
```

Sign in, then immediately:

1. Change the administrator password in **Account**.
2. Invite the real users from **Settings → Users**.
3. Configure SMTP and/or OIDC/LDAP under **Settings → Authentication** if you
   want password-less or directory logins.
4. Configure the agency name, logo and primary color under **Settings → General**.
5. Verify that **Settings → Storage** reports a working bucket.

The environment administrator is only a bootstrap: once accounts exist it does
not overwrite them.

## Reverse proxy

The agency speaks HTTP/1.1 and, optionally, cleartext HTTP/2. Standard
WebSockets carry terminals, desktop streams, the inbox and the tunnel fallback.
A normal reverse proxy is enough.

::: warning Do not force h2c in front of CherryPi
HTTP/2 forbids the `Upgrade` header, so a proxy configured to speak h2c first
breaks the WebSocket fallback used by terminals, desktop and the tunnel. Keep
the upstream at HTTP/1.1. `CHERRYPI_H2C=1` is only for a direct plaintext
connection to an agent, not for the browser side.
:::

Disable the proxy's **read timeout** for long-lived streams. With Caddy:

```text
cherrypi.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Caddy handles WebSockets and streaming by default. With Traefik v3:

```yaml
entryPoints:
  websecure:
    address: ":443"
    transport:
      respondingTimeouts:
        readTimeout: 0
        idleTimeout: 0
```

Set `CHERRYPI_PUBLIC_URL` to the external origin (no trailing slash). It is used
to build magic links, password-reset links and OIDC redirects.

## Health check

```sh
curl -fsS http://127.0.0.1:8080/api/rest/health
# OK
```

The Docker image already declares a health check against this endpoint.

## Next steps

- [Configure the agency](/agency/configuration)
- [Set up authentication](/agency/authentication)
- [Install an agent](/agent/install)
- [Operate and upgrade](/agency/operations)
