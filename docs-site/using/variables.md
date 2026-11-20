---
title: Variables & secrets
description: Give a task's shell an API key without pasting it into a prompt.
---

# Variables & secrets

A **variable** is an environment value a task's shell receives. Define it once,
at one scope, and every task that scope applies to gets it — without anyone
pasting a key into a prompt.

A variable is either **plain** or a **secret**:

|  | Plain | Secret |
| --- | --- | --- |
| Visible in the dashboard | Yes | **No** — masked, replace-only |
| Sealed at rest | No | Yes (`CHERRYPI_SECRET_KEY`) |
| Redacted from tool output | No | Yes, as `[secret:NAME]` |
| Reaches the task's shell | Yes | Yes |

Both kinds are delivered the same way; visibility is the only difference.

## Where a variable can be defined

| Scope | Applies to | Who manages it |
| --- | --- | --- |
| Organization | every task | administrators |
| Project | tasks in the project | project owners |
| Team | tasks run by an agent that belongs to the team | team owners |
| Agent group | tasks run by a member agent | group owners |
| Agent | tasks the agent runs | agent owners |

Team scope follows the **agent's** own memberships — directly or through one of
its agent groups — not the teams that own the task: a variable describes the
machine and its credentials, so it follows the machine. Parent projects do not
pass variables down to their children.

## Precedence

The most specific scope wins:

```
agent > agent group > team > project > organization
```

Two sibling scopes (two teams, say) resolve deterministically by ascending id,
and the losing definition is shown as **overridden** in the agent's _Effective
variables_ view, so an override is visible instead of surprising.

## Adding one

1. Open the scope's **Variables** tab — **Org → Settings → Variables**, a
   project's **Settings → Variables**, a team's **Variables**, an agent group's
   **Variables**, or an agent's **Settings → Variables**.
2. Choose **Add variable**, enter a name (for example `ACME_TOKEN`), the value
   and an optional note.
3. Turn on **Store as a secret** if the value must never be readable again.

The note is shown to the model next to the name; the value is not.

### Names

A name must be a portable environment variable name:
`[A-Za-z_][A-Za-z0-9_]{0,127}`. These are reserved and refused: `PATH`, `HOME`,
`USER`, `LOGNAME`, `SHELL`, `ENV`, `BASH_ENV`, `IFS`, `PWD`, `OLDPWD`,
`TMPDIR`, `NODE_OPTIONS`, `GIT_ASKPASS`, `SSH_AUTH_SOCK`, `SSH_ASKPASS`, and
anything under `CHERRYPI_`, `PI_`, `LD_` or `DYLD_`.

A value is 1 byte to 64 KiB and may contain newlines (a PEM key), but not a NUL.

## Using one in a task

A task's shell sees the merged set as ordinary environment variables:

```sh
curl -H "Authorization: Bearer $ACME_TOKEN" https://api.acme.example/v1/items
```

The value is added to each **bash child's** environment, never to the worker's
own process environment, so two tasks on one agent cannot see each other's
values. It never enters the task's workspace or its stored conversation, and a
secret's occurrences are redacted from tool output — `env` prints
`ACME_TOKEN=[secret:ACME_TOKEN]` instead of the value.

A rotation reaches the next command: the worker revalidates the set at every run
start and reads it on every shell call, so a new or replaced value is available
without restarting anything. The system prompt's _list_ of names is built once
per session, so a brand-new variable is named in the prompt from the next fresh
session; the value is already available to the shell before that.

## Promoting a plain variable

**Make secret** seals a plain variable's current value and hides it from then on.
The reverse is deliberately not offered: revealing a stored secret would break
the promise that no API returns one. Delete the variable and add a new plain one
instead.

## Requirements and limits

Secrets need `CHERRYPI_SECRET_KEY` (see
[Agency environment](/reference/environment#provider-secrets)). Without it,
plain variables keep working and the dashboard says why secrets cannot be
stored.

The feature keeps a value out of the transcript, the workspace, the model
provider request and the dashboard. It does not protect it from the agent's
owner (they control the machine), from a model the value was deliberately handed
to, or from an operator holding both the database and `CHERRYPI_SECRET_KEY`.
