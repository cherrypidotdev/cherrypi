# CherryPi

**CherryPi** manages a fleet of agents from one central web app. Deploy the
**agency** to a VPS (Go server + Vue SPA); run a **cherrypi-agent** on each
machine that should do work. Agents dial the agency, authenticate with a
generated credential, and serve their Pi sessions, terminals, and workspace
over the tunnel. The agency is the single point of control, monitoring, and
accounting.

> Every commit on `main` is a release: the tree is a snapshot of the CherryPi
> source with the test suite and CI harness removed, so the software here is the
> complete, buildable source. Prebuilt binaries, container images and release
> notes are on the [Releases](https://github.com/cherrypidotdev/cherrypi/releases)
> page; the per-release log is in [CHANGELOG.md](CHANGELOG.md).

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — design and host assumptions.
- [`VOCABULARY.md`](VOCABULARY.md) — key terms.
- [`docs/agency-rework.md`](docs/agency-rework.md) — the rework's design.
- [`docs/file-storage.md`](docs/file-storage.md) — S3-backed attachments and pictures.
- [`docs/task-artifacts.md`](docs/task-artifacts.md) — A2A-style task artifacts (agent outputs).
- [`agent/README.md`](agent/README.md) — the worker binary, flags, and configuration.
- [`docs/desktop.md`](docs/desktop.md) — browser access to an agent's running X desktop.
- [`docs/CI.md`](docs/CI.md) — how releases are built.
- [`agency/sdk/js/README.md`](agency/sdk/js/README.md) — the SDK and its methods.

## What's inside

- **Agency:** Go server (Connect, sqlc + goose over SQLite or PostgreSQL, agent registry,
  tunnel hub, webhooks, notifications inbox, usage aggregate) + a Vue 3 SPA
  (Vite, Pug, Tailwind, shadcn-vue, Monaco, xterm.js) embedded into the binary.
- **Agent:** Go worker — the Pi harness (`pi --mode rpc`), parallel session
  manager, PTYs, a desktop bridge, and a tunnel client.
- **Shared:** all protos (`agent.v1`, `agency.v1`, `tunnel.v1`) + generated
  stubs + the tunnel byte-pipe plumbing.
- **SDK:** `agency/sdk/js` (`@cherrypi/sdk`).

## Layout

```
agency/
  server/     Go module (agency API + tunnel hub + embedded SPA)
  frontend/   Vue 3 SPA
  sdk/js/     @cherrypi/sdk
agent/        Go module (worker)
shared/       Go module (protos, generated code, tunnel plumbing)
go.work       ties the three modules
```

## Run with Docker

```sh
docker compose up -d --build
```

Prebuilt images are published per release:

```sh
docker pull ghcr.io/cherrypidotdev/cherrypi-agency:latest
docker pull ghcr.io/cherrypidotdev/cherrypi-agent:latest
```

The agency image is `linux/amd64`; the agent image is the heavy runtime (Pi,
Node, a desktop bridge).

## Develop

```sh
# one-time: install workspaces + generate code
npm install
(cd agency/sdk/js && npm install && npm run build)
(cd agency/frontend && npm install)
(cd shared && buf generate)
(cd agent && sqlc generate)
(cd agency/server && make sqlc)

# agency + SPA (Vite dev server on :5173; set CHERRYPI_FRONTEND_PROXY_URL)
(cd agency/frontend && make dev)
(cd agency/server && CHERRYPI_FRONTEND_PROXY_URL=http://localhost:5173 go run -tags dev ./cmd/agency)

# an agent connected to it (register it first, or seed CHERRYPI_AGENCY_SEED_AGENT)
(cd agent && go run ./cmd/agent --agency-url http://localhost:8080 --auth <user>:<pass>)
```

Enable the repo hooks once per clone so commits land formatted (they also wire
up Git LFS):

```sh
git config core.hooksPath .githooks
```

## Build images and binaries

```sh
make build    # SPA + agent + agency binaries into their modules
make binaries # cross-compiled release binaries + SHA256SUMS in dist/
make docker   # agency and agent images (linux/amd64)
```

## Deploy

Releases publish `ghcr.io/cherrypidotdev/cherrypi-agency:<version>` (plus
`latest`) and the same tag for the agent. Only the agency needs to be reachable
from the browser; agents dial out and need no inbound port. Configuration is
environment-driven (`CHERRYPI_AGENCY_DIR`, `CHERRYPI_AGENCY_CREDENTIALS`,
`CHERRYPI_AGENCY_DATABASE_DSN`, S3 settings, …); see
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) and `compose.yaml`.

## Pi setup

Extensions installed in the agent image:

```sh
pi install npm:@juicesharp/rpiv-ask-user-question
pi install git:github.com/DietrichGebert/ponytail
pi install npm:@narumitw/pi-usage
pi install npm:@gotgenes/pi-permission-system
pi install npm:@tintinweb/pi-tasks
pi install npm:@tintinweb/pi-subagents
pi install npm:pi-agent-browser-native
pi install npm:pi-skills-sh
```

## License

[AGPL-3.0](LICENSE).
