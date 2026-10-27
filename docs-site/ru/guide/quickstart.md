---
title: Быстрый старт
description: Запустите CherryPi локально из репозитория или подключите первого агента к существующей agency.
---

# Быстрый старт

Есть два пути. Используйте **стек на одной машине**, чтобы попробовать
CherryPi, или **установите agency и агента отдельно** для реального
развёртывания.

## Вариант 1 — стек на одной машине

В репозитории есть `compose.yaml`, который собирает и запускает agency, агента
и локальный MinIO для вложений.

**Требования:** Docker с Compose v2 и копия репозитория.

```sh
git clone https://github.com/cherrypidotdev/cherrypi.git
cd cherrypi
docker compose up -d --build agency minio minio-init
```

Откройте <http://localhost:8080> и войдите как `admin` с демо-паролем
`deadbeef`. Смените его до публикации стека или задайте свой пароль при первом
запуске:

```sh
CHERRYPI_ADMIN_PASSWORD='длинный-пароль' \
  docker compose up -d --build agency minio minio-init
```

Затем зарегистрируйте агента: откройте **Agents → Register agent**, создайте
`builder` и скопируйте его одноразовый personal access token. Положите токен в
локальный `.env` и запустите агента:

```sh
echo 'CHERRYPI_AGENT_TOKEN=pat_…' >> .env
docker compose up -d agent
```

Стек больше не создаёт username/password креденшел агента. Не используйте
демо-пароль администратора в production.

::: tip Что запущено

- `agency` — веб-приложение, API и tunnel hub на порту `8080`
- `agent` — воркер, подключённый к agency
- `minio` — S3-совместимое хранилище на портах `9000`/`9001`
  :::

Остановить, сохранив данные:

```sh
docker compose down
```

Начать заново:

```sh
docker compose down -v
rm -rf agency_home cherrypi_home minio_data
```

## Вариант 2 — установить по частям

### 1. Установите agency

```sh
docker run -d --name cherrypi-agency \
  --restart unless-stopped \
  -p 8080:8080 \
  -v "$PWD/agency_data:/root/.cherrypi-agency" \
  -e CHERRYPI_ADMIN_PASSWORD='длинный-пароль' \
  -e CHERRYPI_PUBLIC_URL='https://cherrypi.example.com' \
  ghcr.io/cherrypidotdev/cherrypi-agency:latest
```

Поставьте перед ним TLS reverse proxy. Перед публикацией в интернет прочитайте
[Установка agency](/ru/agency/install).

### 2. Создайте аккаунт и войдите

Откройте URL agency и войдите как `admin` с заданным паролем. Пригласите людей
в **Settings → Users** или настройте OIDC/LDAP.

### 3. Зарегистрируйте агента

В agency откройте **Agents → Register agent** и задайте имя. Скопируйте
одноразовый personal access token, показанный после создания.

### 4. Установите агента

На машине-воркере:

```sh
docker run -d --name cherrypi-agent \
  --restart unless-stopped \
  -v "$PWD/cherrypi_data:/root/.cherrypi" \
  -e CHERRYPI_AGENCY_URL='https://cherrypi.example.com' \
  -e CHERRYPI_AGENT_TOKEN='pat_…' \
  ghcr.io/cherrypidotdev/cherrypi-agent:latest
```

Агент подключится с этим токеном и закэширует его локально. Входящий порт не
нужен. Варианты с бинарником, systemd и рабочим столом — в
[Установка агента](/ru/agent/install).

### 5. Создайте проект и задачу

1. Откройте **Projects** и создайте проект.
2. Добавьте команду или оставьте проект приватным.
3. Откройте проект и создайте задачу, выбрав агента.
4. Напишите промпт. Планировщик запустит задачу, как только у агента появится
   слот.

Страница задачи показывает живой консоль, вложения, артефакты, историю сессии
и терминал агента.

## Куда дальше

- [Основные понятия](/ru/guide/concepts) — agency, агент, проект, задача,
  промпт, артефакт.
- [Настройка agency](/ru/agency/configuration) — база, хранилище, авторизация.
- [Настройка агента](/ru/agent/configuration) — токены, data dir, рабочий стол.
- [Решение проблем](/ru/advanced/troubleshooting) — если агент остаётся offline.
