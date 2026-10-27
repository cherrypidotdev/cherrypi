---
title: JavaScript SDK
description: Управляйте agency из TypeScript или JavaScript через @cherrypi/sdk.
---

# JavaScript SDK

`@cherrypi/sdk` — типизированный TypeScript-клиент Connect API agency. Дашборд
использует тот же SDK, поэтому у потребителей SDK тот же доступ, что и у веб-
интерфейса.

Пакет пока не опубликован в npm. Подключайте его из checkout:

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

## Аутентификация токеном

Создайте personal access token в **Account → Personal access tokens**, затем:

```ts
import { AgencyClient, TaskState } from "@cherrypi/sdk";

const agency = new AgencyClient({
  baseUrl: "https://cherrypi.example.com",
  token: process.env.CHERRYPI_TOKEN!,
});
```

`token` использует `Authorization: Bearer pat_…` и рекомендуется для скриптов.
Браузер использует cookie сессии с CSRF-интерцептором; SDK выбирает его
автоматически, если токен не передан.

## Агенты

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
// token — первый PAT агента, показывается один раз. Задайте CHERRYPI_AGENT_TOKEN.
```

SDK также покрывает `getAgent`, `updateAgent`, `updateAgentRole`,
`deleteAgent`, токены агента, логи, defaults и лимиты параллельности.

## Проекты и промпты

```ts
const project = await agency.createProject({
  name: "Website",
  description: "The marketing site",
  prefix: "WEB",
});

const members = await agency.listProjectAgents(project.id);
```

Промпты версионируются по scope: organization, project, team, agent и
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

## Задачи

```ts
import { TaskPriority, TaskState } from "@cherrypi/sdk";

const task = await agency.createTask({
  projectId: project.id,
  // Задачей владеет хотя бы одна команда проекта; передайте id существующей.
  teamIds: [teamId],
  assigneeActorId: agent.actorId, // не указывайте для auto-assign
  name: "Update the hero section",
  prompt: "Rewrite the hero copy and attach a screenshot.",
  state: TaskState.QUEUED,
  priority: TaskPriority.NORMAL,
});

const page = await agency.listTasks({ projectId: project.id, limit: 20 });
const fresh = await agency.getTask(task.id);

// Продолжить settled-задачу.
await agency.addToTask(task.id, "Also update the meta description.");
```

`watchTask` стримит live-события одной задачи; `getTaskHistory` читает
сохранённые записи сессии Pi с пагинацией.

## Ошибки и повторы

SDK бросает Connect-ошибки с `code` и человекочитаемым `message`. Повторяйте
идемпотентные чтения на `unavailable`, `deadline_exceeded` и
`resource_exhausted`; не повторяйте слепо мутации вроде `createTask` или
`addToTask`.

## Далее

- [Connect API](/ru/advanced/api) — HTTP-поверхность.
- [Основные понятия](/ru/guide/concepts) — состояния задач и роли проекта.
- [Работа с задачами](/ru/using/tasks)
