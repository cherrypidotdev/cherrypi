---
title: Тестирование плагинов
description: Проверяйте, юнит-тестируйте и запускайте плагин CherryPi end-to-end через cherrypi-plugin.
---

# Тестирование плагинов

`cherrypi-plugin` — инструмент автора плагина. Он запускает ту же валидацию и
тот же Lua-хост, что и agency, поэтому зелёный прогон локально означает, что
бандл ведёт себя и в продакшене.

CLI поставляется в образе `cherrypi/agency` и как release-артефакт рядом с
бинарниками agency и агента:

```sh
cherrypi-plugin version
# cherrypi-plugin dev (abc1234)
# plugin API 0.2.4
```

## Три уровня

| Уровень | Команда | Что доказывает | Нужна agency? |
| --- | --- | --- | --- |
| Валидация | `cherrypi-plugin lint` | Манифест, синтаксис Lua, контракт обработчиков, `requires_agency` | Нет |
| Юнит | `cherrypi-plugin test` | Логика обработчиков, SQL, вызовы провайдера | Нет — in-memory двойник |
| End-to-end | `cherrypi-plugin e2e` | Установка, настройки, `create_task`, инструменты, события | Да — одноразовая agency |

Запускайте их по порядку: ошибка lint — это секунда, ошибка юнита — несколько
секунд, а e2e поднимает настоящую agency.

## Валидация

```sh
cherrypi-plugin lint .
# ok acme-git-reviewer 0.1.0 (plugin API 0.2.4)
```

`lint` выполняет установочную валидацию (схема манифеста, entry-файл, пути и
лимиты бандла), компилирует каждый Lua-файл и проверяет два авторских правила:

- продакшен-код не должен делать `require("cherrypi.test")` — тестовый модуль
  существует только в раннере тестов;
- `requires_agency` должен принимать версию plugin API этого CLI.

## Юнит-тесты

Кладите тесты в `test/` бандла:

```text
my-plugin/
├── plugin.json
├── main.lua
├── migrations/sqlite/001_init.sql
└── test/
    └── webhook_test.lua
```

```sh
cherrypi-plugin test .
```

Каждый файл `test/**/*.lua` выполняется в **свежем Lua-состоянии**, против
свежей in-memory agency и собственной SQLite-базы с применёнными миграциями
бандла. `require("cherrypi.test")` возвращает харнесс:

```lua
local t = require("cherrypi.test")

-- Заполните двойника agency.
t.settings("org", {
  base_url = "https://messenger.example",
  api_key = "key",
  project = "INCIDENT",
  tags = { "incident" },
  webhook_secret = "s3cret",
})
t.project({
  id = "INCIDENT",
  name = "Incident Management",
  tags = { "incident" },
})

-- Ожидайте один исходящий вызов.
t.expect_http("POST", "/api/v4/posts", { status = 201, json = { id = "m1" } })

-- Вызовите обработчик.
local status, _, body = t.http_request("POST", "/webhook", t.json({
  secret = "s3cret",
  channel_id = "ops",
  text = "Database is down",
}))

t.equal(status, 200)
local payload = t.decode(body)
t.truthy(payload.task_id)

-- Проверьте, что сделал плагин.
local tasks = t.created_tasks()
t.equal(#tasks, 1)
t.equal(tasks[1].project, "INCIDENT")
t.contains(tasks[1].prompt, "Database is down")
t.equal(tasks[1].tags[1], "incident")
t.equal(tasks[1].tools[1], "reply_in_thread")

-- Инструменты работают так же.
t.tool_call(payload.task_id, "reply_in_thread", { text = "Looking into it." })
t.equal(#t.http_calls(), 1)
```

### Заполнение

| Функция | Назначение |
| --- | --- |
| `t.settings(scope, values)` | Одна строка настроек (`scope` — `org`, `agent` или `user`). |
| `t.settings(scope, target, values)` | Адресная строка, например `t.settings("agent", "7", {...})`. |
| `t.enable(scope, target)` | Пометить строку включённой. |
| `t.project({ id, name, tags, plugin_role, enabled, allow_task_creation, max_priority, settings })` | Проект, его теги, роль актора плагина и поверхность проекта. |
| `t.project_role(project_id, actor_id, role)` | Выдать роль другому актору. |
| `t.agent({ id, actor_id, username, name, project, project_role })` | Агент (и, опционально, его роль в проекте). |
| `t.user({ id, username, name })` | Пользователь. |
| `t.task({ id, project_id, state, prompt, ... })` | Задача, которую плагин может читать. `hold_task` на `settled` паркует её в `waiting`, как в agency. |
| `t.history(task_id, entries)` | Сохранённые записи Pi для `get_task_history`. |
| `t.team({ id, name })` | Команда. |
| `t.expect_http(method, path, { status, json, body, headers })` | Один исходящий ответ; фикстуры расходуются по порядку. |

### Вызов обработчиков и проверки

| Функция | Назначение |
| --- | --- |
| `t.http_request(method, path, body, headers)` | Вызывает `handle_http_request`; возвращает три значения. |
| `t.tool_call(task_id, name, args)` | Вызывает `handle_tool_call`; возвращает значение. |
| `t.event(name, payload)` | Вызывает `handle_event` с `event` и `task_id`. |
| `t.tick(now_ms)` | Вызывает `handle_tick` с тем же временем, что передаёт настоящий тик; бандл без обработчика падает. |
| `t.ticks()` | Расписание из манифеста как `{ every_seconds, interval_ms }`, либо пустая таблица, если плагин не тикает. |
| `t.update_settings(scope, target, values)` | Прогоняет хук `update_settings` бандла так же, как agency: с сохранённой строкой поверхности в `previous` и с подстановкой сохранённого секрета в пустое поле; возвращает его список `{path, errorText}` (пустой, если хука нет). Результат проверяется как в `parseSettingErrors`, поэтому испорченный список поднимает ошибку. `t.update_settings(scope, values)` работает с безадресной строкой переданной поверхности. Хук, который сам поднимает ошибку, поднимает её и здесь — оберните в `pcall`. |
| `t.task_tools(task_id)` | Прогоняет хук `handle_task_tools` бандла и проверяет его результат так же, как планировщик. Возвращает имена инструментов проекта, доступных задаче (все объявленные, если обработчик `task_tools` не объявлен); испорченный результат или упавший хук поднимает ошибку и здесь. |
| `t.created_tasks()` | Вызовы `create_task` с `project`, `tags`, `tools`, `assignee`, `state`, `interactive`. |
| `t.holds(task_id)` | Удержания плагина на задаче (`plugin`, `plugin_name`, `key`, `reason`, `created_at`), от старых к новым. |
| `t.releases(task_id)` | Вызовы `release_task` (`key`, `mode`, `prompt`) в порядке вызовов. |
| `t.http_calls()` | Исходящие вызовы (`method`, `url`, `path`, `body`). |
| `t.logs()` | Строки лога плагина. |
| `t.equal`, `t.not_equal`, `t.contains`, `t.truthy`, `t.falsy`, `t.fail` | Проверки; ошибка останавливает файл с сообщением. |
| `t.json(value)`, `t.decode(text)` | JSON-хелперы. |

Исходящий хост по-прежнему соблюдает `network` из манифеста, поэтому используйте
хост, объявленный в бандле (или добавьте его).

Валидатор настроек проверяется так же: заполните строку, с которой он
сравнивает, прогоните хук и проверьте ошибки полей, которые отклоняют запись.

```lua
t.settings("org", { base_url = "https://gitea.example", api_key = "token-1" })

local errors = t.update_settings("org", { base_url = "", api_key = "" })
t.equal(#errors, 1)
t.equal(errors[1].path, "base_url")

-- Сохранённый секрет подставляется в пустое поле, как в форме с маской.
t.equal(#t.update_settings("org", { base_url = "https://gitea.example" }), 0)

-- Хук, который поднимает ошибку, отклоняет запись.
t.falsy(pcall(function()
  t.update_settings("org", { base_url = "not a url" })
end))
```

## End-to-end

`e2e` устанавливает бандл в **одноразовую agency**, заполняет проект, агента и
теги, применяет настройки сценария и выполняет его шаги. Заполненный агент
получает роль **member** в проекте, потому что viewer не может быть назначен на
задачу и не может её выполнять.

```yaml
# plugin-test.yaml
agency:
  project: { name: Incident Management, prefix: INC }
  agent:
    username: incident-bot
    name: Incident Bot
    tool_args: { text: Looking into it. } # --with-agent
  tags: [incident]

settings:
  org:
    base_url: ${env.MOCK_URL}
    api_key: mock-key
    project: ${project.id}
    tags: [incident]
    webhook_secret: incident-secret

steps:
  - name: the webhook creates an incident task
    http:
      method: POST
      path: /webhook
      json:
        secret: incident-secret
        channel_id: ops
        text: Database is down
    expect:
      status: 200
      json: { task_id: "@nonempty" }
      tasks:
        - project: Incident Management
          tags: [incident]
          prompt_contains: Database is down

  - name: the agent runs and settles
    wait:
      state: settled
      timeout: 180s

  - name: the messenger received the reply
    http:
      method: GET
      url: ${env.MOCK_URL}/__calls
    expect:
      text_contains: Looking into it.
```

```sh
MOCK_URL=http://127.0.0.1:8081 \
  cherrypi-plugin e2e . --scenario plugin-test.yaml --with-agent
```

### Переменные

| Переменная | Значение |
| --- | --- |
| `${env.NAME}` | Переменная окружения. |
| `${var.name}` | Запись `vars:` из сценария. |
| `${project.id}`, `${project.name}`, `${project.slug}` | Заполненный проект. |
| `${agent.id}`, `${agent.actor_id}`, `${agent.token}` | Заполненный агент. |
| `${plugin.id}` | Установленный плагин. |
| `${agency.url}` | Базовый URL agency. |

### Шаги

| Действие | Поля | Примечания |
| --- | --- | --- |
| `http` | `method`, `path` (маршрут плагина) или `url` (абсолютный) | `json` или `body`; `headers`; `expect.status`, `expect.json`, `expect.text_contains`, `expect.tasks`. |
| `tool` | `name`, `args`, `task` | Вызывает agent-tool route токеном заполненного агента. `task` по умолчанию — последняя созданная сценарием задача. |
| `wait` | `state`, `timeout`, `task` | Опрашивает задачу до нужного состояния. |
| `task` | `project`, `name`, `prompt`, `state` | Заполняет задачу для шага `event`. `project` по умолчанию — заполненный проект, `state` — `backlog`. |
| `event` | `name`, `task`, `payload` | Отправляет событие agency через сервис плагинов. `task` — id задачи или `last` (по умолчанию); `payload` добавляет поля в событие. |
| `sleep` | длительность | Пауза, например `2s`. |

`expect.tasks` сопоставляет задачи, **созданные этим шагом**. `assignee:
"@agent"` соответствует заполненному агенту. `expect.json` сравнивает
подмножество; `"@nonempty"` проверяет, что строка непустая.

### Плагины только с событиями

У бандла, который реализует только `handle_event`, нет ни маршрута, ни
инструмента — до него ничего не дойдёт. Его запускают шаги `task` и `event`:
`task` заполняет задачу, на которую указывает событие, а `event` отправляет
событие через сервис плагинов — тем же путём, что и настоящий переход задачи.

```yaml
agency:
  project: { name: Acme User Bridge, prefix: AUB }

settings:
  org:
    base_url: ${env.MOCK_URL}
    api_key: mock-key
  user:
    acme_user: alice
    events: [task_completed, task_requested_input]

steps:
  - name: a completed task exists
    task:
      project: ${project.id} # необязательно, по умолчанию заполненный проект
      name: Fix the deploy # необязательно
      prompt: The deploy is broken.
      state: completed # необязательно, по умолчанию backlog
    expect:
      tasks:
        - project: Acme User Bridge
          state: completed

  - name: the completion is delivered as a direct message
    event: task_completed # краткая форма: последняя задача, без payload

  - name: the messenger received it
    http:
      method: GET
      url: ${env.MOCK_URL}/__calls
    expect:
      text_contains: Fix the deploy

  - name: the input request carries its title
    event:
      name: task_requested_input
      task: last # id задачи или last (по умолчанию)
      payload: { title: "Which branch should I deploy?" }
```

`task.state` — состояние, в котором задача **хранится**: `backlog`, `queued`,
`running`, `settled`, `failed`, `input`, `completed` или `cancelled`; его и
показывает снимок задачи в событии. Заполнение пишет строку напрямую и никогда
не отправляет событие само, поэтому сценарий сам решает, какие события увидит
плагин. `event` ждёт завершения `handle_event` у всех подписчиков, поэтому
следующий шаг не соревнуется с доставкой.

Оба шага вызывают маршруты, существующие только в dev-сборке, поэтому им нужна
agency, собранная с `-tags dev`; образ `cherrypi/plugin-ci` такой, а
`cherrypi/agency` — release-сборка.

### Запланированные тики

Бандлу с `schedule` в манифесте нужно прогонять и `handle_tick`. Шаг `tick`
выполняет один тик через тот же single-flight путь, что и тикер, и возвращается
после завершения обработчика, поэтому следующий шаг может проверить созданное.
E2E-agency стартует с выключенным фоновым тикером
(`CHERRYPI_PLUGIN_DISABLE_TICKS=1`), поэтому интервал не гоняется со сценарием.

```yaml
- name: тик опрашивает упоминания
  tick: {} # или `tick: true`
  expect:
    tasks:
      - project: Incident Management
        prompt_contains: Please look at the queue
```

`tick: { now_ms: 1700000000000 }` (или краткая форма `tick: 1700000000000`)
фиксирует время, которое получает `handle_tick`.

### Два режима e2e

- **Только agency** (по умолчанию): быстро и достаточно для вебхуков, настроек,
  `create_task` и шагов `task`/`event`. Шаг `tool` вызывает agent-tool route
  напрямую, поэтому задача должна быть назначена заполненному агенту.
- **С агентом** (`--with-agent`): поднимает настоящего агента и mock LLM, чей
  первый ход вызывает первый инструмент бандла. Это проверяет весь путь —
  планировщик, Pi, встроенное расширение инструментов, маршрут agency и
  `handle_tool_call`. `agent.tool_args` сценария становятся аргументами
  инструмента.

`--with-agent` требует `cherrypi-agent`, `cherrypi-mockllm` и `pi` (всё есть в
образе `cherrypi/plugin-ci`). Передайте `--keep`, чтобы сохранить каталог данных,
и `-v`, чтобы видеть логи agency и агента.

### Работа с agency, которую вы не запускали

`--agency-url` прогоняет сценарий против уже запущенной agency — dev-agency в
терминале или стенда — вместо одноразовой. Харнесс входит в неё как
администратор, поэтому ему нужны учётные данные администратора этой agency:

| Флаг | Переменная окружения | По умолчанию |
| --- | --- | --- |
| `--agency-username` | `CHERRYPI_ADMIN_USERNAME` | `admin` |
| `--agency-password` | `CHERRYPI_ADMIN_PASSWORD` | нет; запуск останавливается и называет флаг |

```sh
CHERRYPI_ADMIN_PASSWORD=adminpass \
  cherrypi-plugin e2e . --scenario plugin-test.yaml --agency-url http://127.0.0.1:8096
```

Всё остальное как в режиме запуска, но харнесс заполняет ту agency, на которую
указали, и оставляет в ней проект, агента, установленный плагин и задачи,
поэтому используйте одноразовый экземпляр. Этой agency также нужен
`CHERRYPI_PLUGIN_ALLOW_PRIVATE_NETWORK=1`, если сценарий обращается к моку
провайдера на `127.0.0.1`, и сборка с `-tags dev` для шагов `task` и `event`.

### Моки провайдера

Моки провайдера принадлежат репозиторию плагина. Запустите небольшой HTTP-сервер,
говорящий на API провайдера, который вызывает плагин (и эндпоинт проверки,
например `/__calls`), затем направьте `base_url` плагина на него через
`${env.MOCK_URL}`. Agency запускается с
`CHERRYPI_PLUGIN_ALLOW_PRIVATE_NETWORK=1`, поэтому `127.0.0.1` работает;
allowlist `network` из манифеста по-прежнему действует.

## CI

Харнесс работает в контейнере, поэтому job плагина — это checkout и несколько
команд. Секреты agency не нужны: agency одноразовая и заполняется CLI.

```yaml
name: plugin
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: gitea.vladfaust.com/cherrypi/plugin-ci:dev
      credentials:
        username: ${{ secrets.REGISTRY_USERNAME }}
        password: ${{ secrets.REGISTRY_TOKEN }}
    steps:
      - uses: actions/checkout@v4

      - name: Lint
        run: cherrypi-plugin lint .

      - name: Unit tests
        run: cherrypi-plugin test .

      - name: End to end
        env:
          MOCK_URL: http://127.0.0.1:8081
        run: |
          node mock/server.mjs &      # мок провайдера самого плагина
          for _ in $(seq 1 30); do curl -fsS "$MOCK_URL/__calls" && break; sleep 0.2; done
          cherrypi-plugin e2e . --scenario plugin-test.yaml --with-agent

      - name: Package
        run: cherrypi-plugin pack . -o plugin.zip
```

Уберите `--with-agent` для бандла без инструментов: сценарий только с событиями
работает без агента.

- Используйте `cherrypi/agency:<tag>`, если агент не нужен (lint, юнит-тесты,
  e2e вебхуков); он заметно меньше. Это release-сборка, поэтому маршрутов
  `task`/`event` в ней нет: сценарию только с событиями нужен
  `cherrypi/plugin-ci`.
- Используйте `cherrypi/plugin-ci:<tag>` для `--with-agent` и для шагов
  `task`/`event`.
- `pack` сначала запускает `lint` и пишет детерминированный zip, который можно
  приложить к релизу и установить через UI.

### Проверка нескольких версий agency

`requires_agency` — контракт, который объявляет бандл. Проверяйте края диапазона
матрицей:

```yaml
strategy:
  matrix:
    image:
      - gitea.vladfaust.com/cherrypi/plugin-ci:dev
      - gitea.vladfaust.com/cherrypi/plugin-ci:v0.1.0
```

`cherrypi-plugin lint` падает, когда plugin API образа вне `requires_agency`,
поэтому матрица точно показывает, какая версия сломала совместимость.

## Решение проблем

| Симптом | Причина |
| --- | --- |
| `module 'cherrypi.test' is not in the bundle` | Тестовый модуль внедряет только `cherrypi-plugin test`; запускайте через CLI. |
| `host "…" is not in the manifest allowlist` | Добавьте хост провайдера в `network`; юнит-тесты используют ту же защиту. |
| `the plugin is not enabled in this project` | `agency.project` сценария и `settings.org.project` не совпадают, либо поверхность проекта не включена. |
| `priority … exceeds the project's plugin cap` | Поднимите максимальный приоритет проекта или снизьте запрошенный. |
| `task is not assigned to this agent` | Шаг `tool` вызван на неназначенной задаче; используйте `--with-agent` или назначьте задачу агенту. |
| `… needs an agency built with -tags dev` | Шаг `task` или `event` выполнен на release-agency (образ `cherrypi/agency`); используйте `cherrypi/plugin-ci`. |
| `no enabled plugin subscribes to "…"` | События нет в списке `events` манифеста бандла, либо плагин выключен. |
| `bundle requires agency …` | Plugin API образа вне `requires_agency`. |

## Дальше

- [Плагины](/ru/advanced/plugins) — обработчики, методы и поверхности настроек.
- [JavaScript SDK](/ru/advanced/sdk) — клиент, который харнесс использует внутри.
