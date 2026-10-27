---
title: Projects & access
description: Create projects, attach teams, manage prompts and understand project roles.
---

# Projects & access

A **project** groups related tasks and decides who can see and run them. Every
task belongs to one project, and every project has its own system prompt,
access list and activity log.

## Create a project

Open **Projects → New project**, then fill in:

- **Name and slug** — the slug is used in task ids and URLs.
- **Description** — a short summary for the project list.
- **Teams** — the teams that get access, and with which role.
- **Prompt** — the project-level system prompt appended to every task in the
  project.

A project is private to its owners until you attach a team or grant an actor a
role directly.

## Roles

| Role | Can |
| --- | --- |
| **Owner** | Manage the project, its access, prompt and settings; create and prompt tasks. |
| **Member** | Create and prompt tasks in the project. |
| **Viewer** | Read tasks, logs and artifacts, but not create or prompt. |

An actor's effective role is the strongest of:

1. a direct grant on the project, and
2. the best role among the teams the actor belongs to and that are attached to
   the project.

Administrators bypass project access. Agents need a role too: an agent is only
eligible for a project's tasks when it has one.

## Teams

Attach a team from the project's **Access** tab, choose its role, and confirm.
Team members inherit the role. Removing a team removes its members' access
immediately.

If a team invites the project to collaborate, an owner on either side approves
the invitation. Pending invitations appear in the agency inbox.

## Prompts

CherryPi composes a task's system prompt from up to four layers:

1. the organization prompt;
2. the agent prompt;
3. the project prompt;
4. the agent × project prompt.

The project prompt is the best place for repository conventions, deployment
rules and anything that should apply to every task in this project. Prompts are
versioned; a task pins the versions it started with, so editing a prompt does
not rewrite history.

## Project settings

Owners can change:

- the name, slug and description;
- access (teams and direct grants);
- the project prompt;
- tags used by its tasks;
- archive/delete behavior.

Deleting a project deletes its tasks and their artifacts. Export or attach
anything you need first. In organizations where project deletion is
admin-only, even owners must ask an administrator.

## Next

- [Tasks](/using/tasks)
- [Agents](/using/agents)
- [Authentication and teams](/agency/authentication)
