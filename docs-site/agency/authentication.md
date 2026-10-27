---
title: Authentication & teams
description: Accounts, login methods, personal access tokens, teams, roles and agent ownership in CherryPi.
---

# Authentication & teams

CherryPi uses one identity table for people and agents. A username is globally
unique, and every access rule refers to the underlying actor id — so renaming a
user or an agent does not break ownership or history.

## First boot

On an empty installation, the agency creates the first administrator from
`CHERRYPI_ADMIN_USERNAME` and `CHERRYPI_ADMIN_PASSWORD`. If no password is set,
one is generated and printed once to the process log.

After signing in, change the password and create real accounts. The
environment administrator is a bootstrap, not a permanent shared login.

If you prefer open registration on a private network, set
`CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN=true`. Do not enable it
on a public installation.

## Login methods

Configure these under **Settings → Authentication**.

| Method | How it works |
| --- | --- |
| **Password** | Username or confirmed e-mail + password. Can be disabled for members while administrators retain access. |
| **Magic link** | A single-use link sent by e-mail. |
| **OIDC / OAuth2** | Generic OIDC discovery, or a generic OAuth2 provider. Multiple providers are supported. |
| **LDAP** | Multiple directories. LDAP users are invite-only: an administrator invites the directory username first. |
| **TOTP** | Per-method two-factor policy with one-time recovery codes. |

Password login is the default. If you disable it, keep at least one working
provider or an administrator with a password.

### SMTP

Magic links, invitations and password resets need SMTP. Configure the server,
port, sender and TLS mode under **Settings → Authentication → SMTP**. When SMTP
is not configured, invitation and magic-link screens show a copyable link
instead.

::: warning Never enable exposed magic links in production
`CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS=true` returns generated links in API responses
for SMTP-less demos. It bypasses the e-mail ownership check.
:::

## Personal access tokens

Browsers authenticate with an HTTP-only session cookie. Everything else — the
SDK, scripts, agent tools — uses a personal access token starting with `pat_`.

Create and revoke tokens in **Account → Personal access tokens**. Tokens can
have a description and an expiry. Treat them as passwords: they grant the same
access as the account that created them.

Agents have their own tokens and no username/password fallback. Registering an
agent returns a PAT once; mint or revoke tokens from the agent's **Settings →
Security → Personal access tokens**. Pass the token as `--token` or
`CHERRYPI_AGENT_TOKEN`; the agent caches it in `$CHERRYPI_DIR/agent.token`
(mode `0600`). A token the agency rejects is discarded at startup so the
operator can supply a new one.

## Roles

| Actor | Roles | Meaning |
| --- | --- | --- |
| User | `admin`, `member` | Administrators manage users, settings and every task; members work in their projects and teams. |
| Agent | `member`, `admin` | Admin agents are read-only observers: they can read fleet logs and metadata, but get no human administrator powers. Only a human administrator can change an agent's role. |
| Project member | `owner`, `member`, `viewer` | Effective role is the strongest of a direct grant and the best role among the actor's teams. |

Project roles decide who may create, prompt, edit or only view tasks in a
project. An agent also needs a role in the project to be eligible for its
tasks.

## Teams

A team groups users and agents. Teams are the primary access mechanism:

- attach a team to a project with a role;
- share agents with a team (agents can be owned by any actor kind);
- invite people to a team;
- let a team own project work.

Team invitations and project invitations from a team require approval from an
owner. The agency's inbox collects these requests.

## Agent ownership

Every agent has one or more owners. Owners can:

- edit the agent's name, avatar, description and defaults;
- approve team invitations;
- add or remove other owners;
- open the agent's root terminal.

Creating an agent makes you its first owner. Administrators can always manage
the fleet, but ownership is the normal delegation mechanism.

## Sessions and expiry

Browser sessions, magic links, confirmations and personal access tokens have
configurable lifetimes under **Settings → Authentication**. The agency hashes
session and token secrets; the plaintext is shown only once.

## Next

- [Projects and access](/using/projects)
- [Install an agent](/agent/install)
- [Operate and upgrade](/agency/operations)
