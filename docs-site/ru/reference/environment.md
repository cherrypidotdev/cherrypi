---
title: Переменные agency
description: Полный справочник переменных окружения CherryPi agency.
---

# Переменные agency

Все переменные необязательны, если не отмечено **обязательно**. Значения
читаются при старте; после изменения перезапустите agency.

## Основное

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_AGENCY_DIR` | `~/.cherrypi-agency` | Каталог данных для SQLite и process-логов. |
| `CHERRYPI_AGENCY_PORT` | `8080` | HTTP-порт. |
| `CHERRYPI_PUBLIC_URL` | `http://localhost:<port>` | Внешний origin без слэша на конце. Для ссылок, cookie и OIDC callback. |
| `CHERRYPI_H2C` | выкл. | Cleartext HTTP/2 на слушателе. Только диагностика/прямой агент; не форсируйте через прокси. |
| `CHERRYPI_AGENCY_LOG_FILES` | `7` | Сколько process-логов хранить. |

## База данных

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_AGENCY_DATABASE_DSN` | `$CHERRYPI_AGENCY_DIR/agency.sqlite3` | Путь SQLite или URL `postgres://` / `postgresql://`. |

Миграции выполняются при старте. Используйте отдельную базу и роль; agency
нужен владелец схемы.

## Первый запуск и bootstrap

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_ADMIN_USERNAME` | `admin` | Имя первого администратора. |
| `CHERRYPI_ADMIN_PASSWORD` | генерируется и печатается один раз | Пароль первого администратора. |
| `CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN` | выкл. | Открытая регистрация, где первый пользователь становится админом. |
| `CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS` | выкл. | Возвращать сгенерированные ссылки в API. Не включайте публично. |

## Объектное хранилище

Хранилище включается, когда заданы endpoint и бакет.

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_S3_ENDPOINT` | — | S3-совместимый endpoint. Голый хост получает `https://`. |
| `CHERRYPI_S3_BUCKET` | — | Имя бакета. Должен существовать. |
| `CHERRYPI_S3_REGION` | `us-east-1` | Регион бакета; проверяется при старте. |
| `CHERRYPI_S3_ACCESS_KEY_ID` | — | Access key ID. |
| `CHERRYPI_S3_SECRET_ACCESS_KEY` | — | Secret access key. |
| `CHERRYPI_S3_FORCE_PATH_STYLE` | выкл. | Path-style запросы: `https://host/bucket/...`. Для MinIO за прокси. |
| `CHERRYPI_S3_KEY_PREFIX` | — | Префикс всех ключей объектов; слэш добавляется. |

Agency проверяет бакет при старте через `HeadBucket`, сверяет регион и валит
boot с понятной ошибкой.

## Вложения

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_ATTACHMENT_MAX_BYTES` | `26214400` (25 МиБ) | Максимальный размер одного файла. |
| `CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE` | `10` | Максимум вложений на сообщение. |

## Сайты

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_SITES_INSECURE_HTTP` | выкл. | Отдавать сырое содержимое сайтов вообще без учётных данных по обычному HTTP, если хост не localhost. Нужен только для прямого неаутентифицированного доступа (обычный viewer использует короткоживущие URL-grant'ы); статические байты становятся публичными по URL. Никогда не включайте на публичном развёртывании. |

## Наблюдаемость (OpenTelemetry)

Agency экспортирует трейсы, метрики и логи по OTLP. Используются стандартные
переменные OpenTelemetry, поэтому подойдёт любой collector или бэкенд с OTLP —
специальных настроек CherryPi не нужно.

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `CHERRYPI_OTEL_ENABLED` | авто | Включить (`1`) или выключить (`0`) телеметрию. Если не задана, OTLP-endpoint в переменных ниже включает её автоматически. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Базовый URL collector'а, например `http://otel-collector:4317` (gRPC) или `http://otel-collector:4318` (HTTP). |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `grpc` | `grpc` или `http/protobuf`. Переопределения для сигнала (`OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`, `..._METRICS_PROTOCOL`, `..._LOGS_PROTOCOL`) важнее. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | — | Endpoint для конкретного сигнала. Аналогично для метрик и логов. |
| `OTEL_EXPORTER_OTLP_HEADERS` | — | Дополнительные заголовки OTLP-запросов: пары `key=value` через запятую, значения URL-encoded, например `Authorization=Bearer%20<token>` или `x-otlp-api-key=<token>`. |
| `OTEL_EXPORTER_OTLP_<SIGNAL>_HEADERS` | — | Заголовки для конкретного сигнала (`..._TRACES_HEADERS`, `..._METRICS_HEADERS`, `..._LOGS_HEADERS`); важнее общего значения. |
| `OTEL_SERVICE_NAME` | `cherrypi-agency` | Имя сервиса во всех сигналах. |
| `OTEL_RESOURCE_ATTRIBUTES` | — | Дополнительные resource-атрибуты, например `deployment.environment=prod`. |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | Стандартный sampler (`always_on`, `always_off`, `traceidratio`, `parentbased_traceidratio`, …). |
| `OTEL_TRACES_SAMPLER_ARG` | — | Доля для ratio-sampler'ов, например `0.1`. |
| `OTEL_TRACES_EXPORTER` | `otlp` | `none` отключает трейсы, оставляя метрики и логи. |
| `OTEL_METRICS_EXPORTER` | `otlp` | `none` отключает метрики. |
| `OTEL_LOGS_EXPORTER` | `otlp` | `none` отключает мост логов. |
| `OTEL_METRIC_EXPORT_INTERVAL` | `60000` | Интервал экспорта метрик в миллисекундах. |
| `OTEL_BSP_*` / `OTEL_BLRP_*` | значения SDK | Настройка батчей трейсов/логов (очередь, задержка, таймаут, размер). |

```sh
CHERRYPI_OTEL_ENABLED=1
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_SERVICE_NAME=cherrypi-agency
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=prod,team=platform
# Collector с включённой авторизацией ([OtlpDashboard](https://github.com/Adiakys/OtlpDashboard) принимает оба варианта):
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer%20<token>
# OTEL_EXPORTER_OTLP_HEADERS=x-otlp-api-key=<token>
```

Инструментируется:

- каждый унарный Connect/gRPC вызов (`agency.v1.TaskService/GetTask`, …) —
  RPC-спан и гистограмма длительности RPC. Потоковые соединения (туннель агента,
  realtime- и task-watcher) не трейсятся: их спан жил бы столько же, сколько
  соединение;
- каждый REST-запрос под `/api/` и `/plugins/` — HTTP-спан и
  гистограмма длительности HTTP;
- каждый SQL-запрос agency через `database/sql` (SQLite и PostgreSQL), включая
  `db.query.text`, плюс gauge'и пула `sql.DB`. Спаны SQL и метрика
  `db.client.operation.duration` несут `db.query.summary` с именем sqlc-запроса
  (например, `store.ListTasks`), поэтому представление самых нагруженных
  операций показывает store-операцию, а не только `sql.conn.query`;
- строки стандартного логгера процесса как OTel-логи, дополнительно к
  process-логу на диске.

В трейс-вьюере, который понимает W3C trace context (`traceparent`), видно
дерево запроса: HTTP- или RPC-спан, его SQL-потомки и ошибки. Пример:

```text
POST /api/rest/auth/login           (HTTP server, ~2 ms)
└── SELECT … FROM users …           (SQL, ~1 ms)
```

Встроенный фронтенд, документация и статические байты сайтов не трейсятся: это
файлы, а не работа agency.

## Только для разработки

Игнорируются релизными сборками.

| Переменная | Описание |
| --- | --- |
| `CHERRYPI_FRONTEND_PROXY_URL` | Проксировать SPA в Vite dev server. |
| `CHERRYPI_DOCS_PROXY_URL` | Проксировать `/docs` в VitePress dev server. |

## Настройки организации

Провайдеры аутентификации (OIDC, LDAP), SMTP, брендинг, retention и политики
хранятся в базе и настраиваются в дашборде, а не через окружение. См.
[Настройка agency](/ru/agency/configuration) и
[Аутентификация](/ru/agency/authentication).

## Окружение агента

У агента свои переменные и флаги. См.
[Настройка агента](/ru/agent/configuration#переменные-окружения).
