---
title: JavaScript SDK
description: Use @cherrypi/sdk to drive the agency from TypeScript or JavaScript.
---

# JavaScript SDK

`@cherrypi/sdk` is the typed TypeScript client for the agency's Connect API. The
dashboard uses the same SDK, so SDK consumers have the same access as the web
UI.

The package is not published to npm yet. Consume it from a checkout:

```json
{
  "dependencies": {
    "@cherrypi/sdk": "file:../../agency/sdk/js"
  }
}
```

```sh
cd agency/sdk/js
npm ci
npm run build
```

## Authenticate with a token

Create a personal access token in **Account → Personal access tokens**, then:

```ts
import { AgencyClient, TaskState } from "@cherrypi/sdk";

const agency = new AgencyClient({
  baseUrl: "https://cherrypi.example.com",
  token: process.env.CHERRYPI_TOKEN!,
});
```

`token` uses `Authorization: Bearer pat_…` and is the recommended option for
scripts. The browser uses the session cookie with a CSRF interceptor instead;
the SDK selects it automatically when no token is supplied.

## Agents

```ts
import { AgentStatus } from "@cherrypi/sdk";

const agents = await agency.listAgents();
for (const agent of agents) {
  console.log(agent.displayName, agent.status === AgentStatus.ONLINE);
}

const { agent, token } = await agency.createAgent({
  username: "builder-2",
  displayName: "Builder 2",
  prompt: "Prefer small, reviewable diffs.",
});
// token is the agent's first PAT, shown once. Set CHERRYPI_AGENT_TOKEN.
```

The SDK also covers `getAgent`, `updateAgent`, `updateAgentRole`,
`deleteAgent`, agent tokens, logs, defaults and concurrency limits.

## Projects and prompts

```ts
const project = await agency.createProject({
  name: "Website",
  description: "The marketing site",
  prefix: "WEB",
});

const members = await agency.listProjectAgents(project.id);
```

Prompts are versioned per scope: organization, project, team, agent and
agent × project.

```ts
import { PromptScope } from "@cherrypi/sdk";

await agency.setPrompt(PromptScope.ORG, "Be concise and factual.");
await agency.setPrompt(PromptScope.PROJECT, "Match the existing style.", {
  projectId: project.id,
});
await agency.setPrompt(PromptScope.AGENT_PROJECT, "Run the tests first.", {
  agentId: agent.id,
  projectId: project.id,
});
```

## Tasks

```ts
import { TaskPriority, TaskState } from "@cherrypi/sdk";

const task = await agency.createTask({
  projectId: project.id,
  // A task is owned by at least one project team; pass an existing team id.
  teamIds: [teamId],
  assigneeActorId: agent.actorId, // omit to auto-assign
  name: "Update the hero section",
  prompt: "Rewrite the hero copy and attach a screenshot.",
  state: TaskState.QUEUED,
  priority: TaskPriority.NORMAL,
});

const page = await agency.listTasks({ projectId: project.id, limit: 20 });
const fresh = await agency.getTask(task.id);

// Follow up on a settled task.
await agency.addToTask(task.id, "Also update the meta description.");
```

`watchTask` streams live runtime events for one task; `getTaskHistory` reads the
stored Pi session entries with pagination.

## Errors and retries

The SDK throws Connect errors with a `code` and a human-readable `message`.
Retry idempotent reads on `unavailable`, `deadline_exceeded` and
`resource_exhausted`; do not blindly retry mutations such as `createTask` or
`addToTask`.

## Next

- [Connect API](/advanced/api) — the underlying HTTP surface.
- [Concepts](/guide/concepts) — task states and project roles.
- [Using tasks](/using/tasks)
