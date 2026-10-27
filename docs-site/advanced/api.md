---
title: Connect API
description: The agency's Connect RPC surface, authentication and streaming endpoints.
---

# Connect API

The agency exposes a typed [Connect](https://connectrpc.com/) API. The same
handlers serve Connect, gRPC and gRPC-Web clients; the dashboard and the SDK use
the Connect protocol over HTTP/1.1 or HTTP/2.

Protobuf definitions live in `shared/proto/agency/v1/`. Generated clients are
used by the SDK, so you rarely need to construct requests by hand.

## Base path

All RPCs are mounted under `/api/grpc`:

```text
POST /api/grpc/agency.v1.TaskService/ListTasks
POST /api/grpc/agency.v1.AgentService/ListAgents
POST /api/grpc/agency.v1.ProjectService/CreateProject
```

The Connect unary protocol accepts JSON with
`Content-Type: application/json`:

```sh
curl -fsS \
  -H "Authorization: Bearer $CHERRYPI_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"limit": 10}' \
  https://cherrypi.example.com/api/grpc/agency.v1.TaskService/ListTasks
```

## Authentication

| Client | Credential |
| --- | --- |
| Browser | HTTP-only session cookie + `X-CSRF-Token` for mutations. |
| SDK / scripts | Personal access token: `Authorization: Bearer pat_…`. |
| Agents | Agent personal access token (`pat_…`). |

Unauthenticated requests to `/api/grpc` return `401`. The public auth surface at
`/api/rest/auth/*` is reachable without a session so the SPA can sign in.

## Services

| Service | Responsibility |
| --- | --- |
| `agency.v1.AgentService` | Fleet registry, personal access tokens, logs, limits, defaults. |
| `agency.v1.TaskService` | Tasks, lifecycle, prompts, history, artifacts, references, queues. |
| `agency.v1.TagService` | Task tags. |
| `agency.v1.ProjectService` | Projects, project agents and access. |
| `agency.v1.PromptService` | Versioned prompts and the composed task prompt. |
| `agency.v1.SettingsService` | Organization settings and branding. |
| `agency.v1.DashboardService` | Usage and disk accounting. |
| `agency.v1.NotificationService` | Inbox entries and preferences. |
| `agency.v1.TerminalService` | Root terminals on an agent. |
| `agency.v1.DesktopService` | Desktop availability for an agent. |
| `agency.v1.RealtimeService` | Server stream of collection invalidations. |
| `agency.v1.WebhookService` | Outbound webhooks. |

## Streaming

`RealtimeService.Watch` is a Connect server stream that emits coarse
invalidation events (`tasks`, `agents`, `projects`, `usage`, …). Consumers
refetch through the owning service; the stream never carries entities.

`TaskService.WatchTask` streams one task's live runtime. Terminals and the
desktop use dedicated WebSockets because they carry binary data:

| Route | Purpose |
| --- | --- |
| `/api/agents/<agentId>/terminal/<id>/ws` | PTY bytes plus text control frames. |
| `/api/agents/<agentId>/desktop/ws` | RFB stream. |
| `/api/rest/notifications/ws` | Inbox attention events. |
| `/api/tunnel/ws` | Agent tunnel fallback over HTTP/1.1. |

Long-lived streams require a reverse proxy with read timeouts disabled.

## Agent-facing REST

Agents do not call the public RPCs for everything. They use dedicated REST
routes authenticated with an agent PAT only:

| Route | Purpose |
| --- | --- |
| `/api/rest/task-history` | Push and read Pi session entries. |
| `/api/rest/agent-tools` | The bundled Pi tools (`listTasks`, `getTask`, `createTask`, …). |
| `/api/rest/agent-tools/token` | Read-only `GET` probe that validates the caller's PAT. Minting happens in the agency UI/API. |
| `/api/rest/attachments` | Pull task files and upload pictures. |

These routes run their own authentication and are not part of the browser
session flow.

## Errors

Connect errors carry a code and a message. Common codes:

| Code | Meaning |
| --- | --- |
| `unauthenticated` | Missing or invalid credential. |
| `permission_denied` | The actor lacks the project role or admin right. |
| `not_found` | Unknown task, agent or project. |
| `failed_precondition` | The operation is invalid in the current state. |
| `resource_exhausted` | Concurrency limit or quota reached. |
| `unavailable` | The agent is offline or the tunnel dropped. |

## Versioning

The `shared/protocol` version is reported by agents in the tunnel hello. The
agency rejects agents whose protocol version is incompatible. RPC changes are
additive within `agency.v1`.

## Next

- [JavaScript SDK](/advanced/sdk)
- [Install an agent](/agent/install)
- [Troubleshooting](/advanced/troubleshooting)
