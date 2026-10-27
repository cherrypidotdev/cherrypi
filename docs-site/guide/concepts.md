---
title: Concepts
description: The vocabulary and data model used by CherryPi.
---

# Concepts

CherryPi has a deliberately small vocabulary. The same terms are used in the
dashboard, the API and these docs.

## Agency

The central web application. It serves the dashboard, the Connect API, the
tunnel hub, the scheduler and the object-storage proxy. It is the only piece
you expose to the internet.

An agency owns:

- **people** — users, invitations, sessions, personal access tokens;
- **agents** — the registered workers and their ownership;
- **teams and projects** — access control;
- **tasks** — work items, their prompts, history and artifacts;
- **configuration** — authentication providers, SMTP, storage, branding.

## Agent

A long-running worker process on a machine that should do work. It has no HTTP
listener and no local login. It connects **outbound** to the agency over the
tunnel and serves its API only through that connection.

An agent:

- spawns one `pi --mode rpc` child per running task;
- keeps a per-task workspace on disk;
- hosts root PTY terminals;
- can expose an already-running X display to the browser;
- reports its version, protocol version, health and usage.

An agent is an actor too: you can own it, share it with a team, and set its
concurrency limits.

## Tunnel

The persistent connection between an agent and the agency. It prefers a
Connect bidirectional stream over HTTP/2 and falls back to a WebSocket at
`/api/tunnel/ws` over HTTP/1.1. The fallback is what makes a normal TLS reverse
proxy work without cleartext HTTP/2.

If the tunnel drops, the agent reconnects with backoff. Tasks that were running
are recovered when the agent comes back.

## Project

A grouping of tasks with its own description, system prompt and access rules.
Every task belongs to exactly one project.

Access is granted to **teams** (with a role of owner, member or viewer) or to
individual actors as an exception. A user or agent can only run tasks in a
project where it has a role.

## Task

One unit of work. A task has:

- a name and a prompt;
- a project and an assignee agent;
- a lifecycle state — `draft`, `queued`, `running`, `input`, `settled`,
  `failed`, `completed` or `cancelled`;
- a transcript made of user messages and Pi session entries;
- attachments, links and artifacts;
- optional tags, priority and references to other tasks.

A task runs in a **workspace** on the agent. The agency stores the non-header
Pi session entries, so a task that is moved to another agent can be resumed
there from its history.

## Prompt

The text you send to a task, plus the system prompt composed from the
organization, agent, project and agent × project prompt versions. The agency
pins the prompt versions a task started with, so the task remains reproducible
even after a prompt is edited.

## Attachment and artifact

**Attachments** are files you add to a message or task — screenshots, logs,
documents. They are stored in the S3-compatible bucket and proxied by the
agency.

**Artifacts** are files an agent produces and publishes as the outcome of a
task. They appear in the task's **Artifacts** tab, with a name, description and
optional preview.

## Team

A group of actors (users and agents). Teams are the primary unit of access:
attach a team to a project to grant its members a role. Teams can also own
agents, and they make directory information available to their members.

## Personal access token

A `pat_…` credential used by the SDK, the CLI and agent tools. The browser uses
an HTTP-only session cookie instead. Tokens can be scoped, rotated and revoked
from **Account → Personal access tokens**.

## Pi

The coding-agent harness that does the actual model work. CherryPi does not
bundle a model provider: the agent runs `pi`, and Pi uses its own configuration
in `$PI_CODING_AGENT_DIR` (default `~/.pi/agent`). CherryPi adds a task prompt,
the workspace, attachments and a set of bundled tools on top.

## Usage

Token and cost accounting reported by Pi and aggregated by the agency. Usage
is attributed to the task, project and agent that produced it, and survives
agent downtime.
