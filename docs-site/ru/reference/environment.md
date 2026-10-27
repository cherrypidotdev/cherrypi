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
