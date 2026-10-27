---
title: Connect API
description: Connect RPC-поверхность agency, аутентификация и стриминговые endpoint.
---

# Connect API

Agency предоставляет типизированный [Connect](https://connectrpc.com/) API. Одни
и те же хендлеры обслуживают Connect, gRPC и gRPC-Web; дашборд и SDK используют
протокол Connect поверх HTTP/1.1 или HTTP/2.

Protobuf-определения лежат в `shared/proto/agency/v1/`. SDK использует
сгенерированные клиенты, поэтому вручную собирать запросы обычно не нужно.

## Базовый путь

Все RPC смонтированы под `/api/grpc`:

```text
POST /api/grpc/agency.v1.TaskService/ListTasks
POST /api/grpc/agency.v1.AgentService/ListAgents
POST /api/grpc/agency.v1.ProjectService/CreateProject
```

Connect unary принимает JSON с `Content-Type: application/json`:

```sh
curl -fsS \
  -H "Authorization: Bearer $CHERRYPI_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"limit": 10}' \
  https://cherrypi.example.com/api/grpc/agency.v1.TaskService/ListTasks
```

## Аутентификация

| Клиент | Креденшел |
| --- | --- |
| Браузер | HTTP-only cookie сессии + `X-CSRF-Token` для мутаций. |
| SDK / скрипты | Personal access token: `Authorization: Bearer pat_…`. |
| Агенты | Personal access token агента (`pat_…`). |

Неаутентифицированные запросы к `/api/grpc` возвращают `401`. Публичная
поверхность `/api/rest/auth/*` доступна без сессии, чтобы SPA могла войти.

## Сервисы

| Сервис | Ответственность |
| --- | --- |
| `agency.v1.AgentService` | Реестр парка, personal access tokens, логи, лимиты, defaults. |
| `agency.v1.TaskService` | Задачи, жизненный цикл, промпты, история, артефакты, ссылки, очереди. |
| `agency.v1.TagService` | Теги задач. |
| `agency.v1.ProjectService` | Проекты, агенты проекта и доступ. |
| `agency.v1.PromptService` | Версионированные промпты и собранный промпт задачи. |
| `agency.v1.SettingsService` | Настройки организации и брендинг. |
| `agency.v1.DashboardService` | Учёт использования и диска. |
| `agency.v1.NotificationService` | Инбокс и настройки. |
| `agency.v1.TerminalService` | Root-терминалы на агенте. |
| `agency.v1.DesktopService` | Доступность рабочего стола агента. |
| `agency.v1.RealtimeService` | Server stream инвалидаций коллекций. |
| `agency.v1.WebhookService` | Исходящие webhook. |

## Стриминг

`RealtimeService.Watch` — Connect server stream с грубыми событиями инвалидации
(`tasks`, `agents`, `projects`, `usage`, …). Потребители перезапрашивают данные
через владеющий сервис; поток не несёт сущности.

`TaskService.WatchTask` стримит live-runtime одной задачи. Терминалы и рабочий
стол используют отдельные WebSocket из-за бинарных данных:

| Маршрут | Назначение |
| --- | --- |
| `/api/agents/<agentId>/terminal/<id>/ws` | Байты PTY и текстовые control-кадры. |
| `/api/agents/<agentId>/desktop/ws` | RFB-поток. |
| `/api/rest/notifications/ws` | События внимания для инбокса. |
| `/api/tunnel/ws` | Fallback туннеля агента поверх HTTP/1.1. |

Для долгих потоков нужен reverse proxy с отключённым read timeout.

## REST для агентов

Агенты используют отдельные REST-маршруты только с PAT агента:

| Маршрут | Назначение |
| --- | --- |
| `/api/rest/task-history` | Отправка и чтение записей сессии Pi. |
| `/api/rest/agent-tools` | Встроенные инструменты Pi (`listTasks`, `getTask`, `createTask`, …). |
| `/api/rest/agent-tools/token` | Read-only `GET`-проверка PAT вызывающего. Выпуск токена — в UI/API agency. |
| `/api/rest/attachments` | Скачивание файлов задачи и загрузка картинок. |

Эти маршруты аутентифицируются сами и не входят в браузерный поток сессии.

## Ошибки

Connect-ошибки несут код и сообщение. Частые коды:

| Код | Значение |
| --- | --- |
| `unauthenticated` | Нет или неверный креденшел. |
| `permission_denied` | У актора нет роли проекта или прав администратора. |
| `not_found` | Неизвестная задача, агент или проект. |
| `failed_precondition` | Операция недопустима в текущем состоянии. |
| `resource_exhausted` | Достигнут лимит параллельности или квота. |
| `unavailable` | Агент offline или туннель оборвался. |

## Версионирование

Версия `shared/protocol` сообщается агентами в tunnel hello. Agency отклоняет
несовместимые версии протокола. Изменения RPC аддитивны в рамках
`agency.v1`.

## Далее

- [JavaScript SDK](/ru/advanced/sdk)
- [Установка агента](/ru/agent/install)
- [Решение проблем](/ru/advanced/troubleshooting)
