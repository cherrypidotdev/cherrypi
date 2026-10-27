---
title: Настройка agency
description: Переменные окружения, база данных, объектное хранилище и настройки дашборда CherryPi agency.
---

# Настройка agency

Agency настраивается переменными окружения и, после запуска, через **Settings**
в дашборде. Организационные настройки лучше менять в UI: они хранятся в базе и
применяются сразу.

## Переменные окружения

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `CHERRYPI_AGENCY_DIR` | `~/.cherrypi-agency` | Каталог данных: SQLite и process-логи. |
| `CHERRYPI_AGENCY_PORT` | `8080` | HTTP-порт. |
| `CHERRYPI_AGENCY_DATABASE_DSN` | `$CHERRYPI_AGENCY_DIR/agency.sqlite3` | Путь SQLite или URL PostgreSQL. |
| `CHERRYPI_PUBLIC_URL` | `http://localhost:<port>` | Внешний origin для ссылок и OIDC callback. |
| `CHERRYPI_ADMIN_USERNAME` | `admin` | Имя первого администратора. |
| `CHERRYPI_ADMIN_PASSWORD` | генерируется и печатается один раз | Пароль первого администратора. |
| `CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN` | выкл. | Открытая регистрация, где первый пользователь становится админом. |
| `CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS` | выкл. | Возвращать magic link в ответах API без SMTP. Не включайте в production. |
| `CHERRYPI_S3_ENDPOINT` | — | S3-совместимый endpoint. Включает хранилище вместе с бакетом. |
| `CHERRYPI_S3_BUCKET` | — | Имя бакета. |
| `CHERRYPI_S3_REGION` | `us-east-1` | Регион; проверяется при старте. |
| `CHERRYPI_S3_ACCESS_KEY_ID` | — | Access key. |
| `CHERRYPI_S3_SECRET_ACCESS_KEY` | — | Secret key. |
| `CHERRYPI_S3_FORCE_PATH_STYLE` | выкл. | Path-style запросы (`https://host/bucket/...`). |
| `CHERRYPI_S3_KEY_PREFIX` | — | Префикс всех объектов. |
| `CHERRYPI_ATTACHMENT_MAX_BYTES` | `26214400` (25 МиБ) | Лимит одного файла. |
| `CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE` | `10` | Вложений на сообщение. |
| `CHERRYPI_AGENCY_LOG_FILES` | `7` | Сколько process-логов хранить. |
| `CHERRYPI_H2C` | выкл. | Cleartext HTTP/2 для диагностики/прямого агента. |
| `CHERRYPI_FRONTEND_PROXY_URL` | — | Только dev: проксировать SPA в Vite. |
| `CHERRYPI_DOCS_PROXY_URL` | — | Только dev: проксировать `/docs` в VitePress. |

Полный список — в [справочнике окружения](/ru/reference/environment).

## База данных

SQLite используется по умолчанию и не требует отдельного сервиса. Файл базы
лежит в `CHERRYPI_AGENCY_DIR`.

Для PostgreSQL укажите URL:

```sh
CHERRYPI_AGENCY_DATABASE_DSN='postgres://cherrypi:secret@db:5432/cherrypi?sslmode=disable'
```

Agency применяет миграции при старте. Используйте отдельную базу и роль; agency
нужен владелец схемы. Бэкапьте базу вместе с S3-бакетом — вложения лежат в
бакете, не в базе.

::: tip Когда выбирать PostgreSQL
Одна agency на одном хосте отлично живёт на SQLite. PostgreSQL нужен для
управляемых бэкапов, point-in-time recovery или общей базы с операционными
инструментами.
:::

## Объектное хранилище

Вложения, аватары и артефакты лежат в одном бакете. Agency проксирует каждую
загрузку и скачивание, поэтому бакет не должен быть публичным.

```sh
CHERRYPI_S3_ENDPOINT=https://s3.example.com
CHERRYPI_S3_BUCKET=cherrypi
CHERRYPI_S3_REGION=eu-central-1
CHERRYPI_S3_ACCESS_KEY_ID=…
CHERRYPI_S3_SECRET_ACCESS_KEY=…
```

При старте agency вызывает `HeadBucket` и проверяет регион. Неверный endpoint,
бакет или регион валят запуск с понятной ошибкой, а не ломают вложения позже.

Для MinIO или endpoint за прокси без wildcard TLS используйте path-style:

```sh
CHERRYPI_S3_FORCE_PATH_STYLE=true
```

Бакет должен существовать до старта. В compose-файле сервис `minio-init`
создаёт его.

## Настройки дашборда

**Settings** доступны администраторам и хранятся в базе.

| Раздел | Что настраивается |
| --- | --- |
| General | Имя agency, логотип, favicon, основной цвет. |
| Users | Приглашения, роли, отключение аккаунтов. |
| Authentication | Пароль/magic link, TOTP, SMTP, OIDC и LDAP. |
| Agents | Кто может создавать агентов, поведение лимитов. |
| Storage | Статус S3, лимиты загрузок, retention. |
| Logs | Process-лог agency с follow, скачиванием и хвостом. |

О модели идентичности — в [Аутентификация и команды](/ru/agency/authentication).

## Каталог данных

Всё, что agency пишет локально, лежит в `CHERRYPI_AGENCY_DIR`:

```text
$CHERRYPI_AGENCY_DIR/
  agency.sqlite3      # при использовании SQLite
  logs/               # process-логи по запускам
```

Process-лог также виден в **Settings → Logs**. Старые файлы можно удалять;
agency хранит `CHERRYPI_AGENCY_LOG_FILES` штук.

## Dev-прокси

Релизные сборки всегда отдают встроенные дашборд и документацию. Сборка с
`-tags dev` может проксировать их в Vite:

```sh
CHERRYPI_FRONTEND_PROXY_URL=http://localhost:5173 \
  CHERRYPI_DOCS_PROXY_URL=http://localhost:5174 \
  go run -tags dev ./cmd/agency
```

В релизных сборках эти переменные игнорируются.

## Требования к reverse proxy

- Передавайте `Host` и `X-Forwarded-Proto`, чтобы ссылки и cookie были верными.
- Не терминируйте HTTP/2 в agency по cleartext (`h2c`).
- Отключите read timeout для долгих WebSocket и стриминговых маршрутов.
- Держите `CHERRYPI_PUBLIC_URL` синхронным с публичным origin.

Рабочая конфигурация Caddy — в [Установка agency](/ru/agency/install#reverse-proxy).
