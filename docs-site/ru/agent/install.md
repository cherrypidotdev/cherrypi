---
title: Установка агента
description: Запустите CherryPi агента из контейнера или бинарника, зарегистрируйте и подключите к agency.
---

# Установка агента

Агент — headless-воркер. У него **нет входящего порта**: он подключается к
agency, аутентифицируется и отдаёт свой API задач через туннель. Установите его
на машину, которая должна запускать сессии Pi — ноутбук, build-сервер или
облачную VM.

## Перед началом

- Запущенная и доступная с хоста агента URL agency.
- **Personal access token** (`pat_…`) зарегистрированного агента.
- Достаточно места для `$CHERRYPI_DIR`: там рабочие пространства задач и
  сессии Pi.
- Креденшелы модели Pi, если используете бинарник. Образ содержит Pi, но не
  ваши ключи провайдера.

## Зарегистрируйте агента

1. Откройте **Agents → Register agent**.
2. Задайте имя и, опционально, описание.
3. Скопируйте одноразовый personal access token.
4. Сохраните его как `CHERRYPI_AGENT_TOKEN` на хосте агента или выпустите
   новый позже в **Agent → Settings → Security → Personal access tokens**.

Username/password tunnel-креденшела и стартового seed больше нет. Новый агент
не подключится, пока у него нет PAT.

## Вариант A — Docker-образ (рекомендуется)

Опубликованный образ агента содержит Pi, Node, X-утилиты и `x11vnc`.

```sh
docker run -d --name cherrypi-agent \
  --restart unless-stopped \
  -v /srv/cherrypi/agent:/root/.cherrypi \
  -v /srv/cherrypi/pi:/root/.pi/agent \
  -e CHERRYPI_AGENCY_URL='https://cherrypi.example.com' \
  -e CHERRYPI_AGENT_TOKEN='pat_…' \
  ghcr.io/cherrypidotdev/cherrypi-agent:latest
```

| Монтирование | Зачем |
| --- | --- |
| `/root/.cherrypi` | База агента, кэш токена и рабочие пространства задач. |
| `/root/.pi/agent` | Конфигурация Pi, креденшелы провайдеров, память. |

Агент кэширует токен в `/root/.cherrypi/agent.token` (режим `0600`).
Сохраняйте data-том — это кэш идентичности агента. Токен также можно передать
флагом `--token pat_…`; явный токен важнее кэшированного файла, поэтому
ротация токена — это перезапуск контейнера/процесса.

```sh
-e CHERRYPI_AGENT_TOKEN='pat_…'
```

Явный токен важнее кэшированного файла, поэтому ротация — это перезапуск
контейнера.

### Compose

```yaml
services:
  agent:
    image: ghcr.io/cherrypidotdev/cherrypi-agent:latest
    restart: unless-stopped
    environment:
      CHERRYPI_AGENCY_URL: https://cherrypi.example.com
      CHERRYPI_AGENT_TOKEN: pat_…
    volumes:
      - ./agent-data:/root/.cherrypi
      - ./pi-agent:/root/.pi/agent
```

## Вариант B — релизный бинарник

Скачайте `cherrypi-agent-<os>-<arch>-<sha>.zip` со
[страницы релизов](https://github.com/cherrypidotdev/cherrypi/releases) и
распакуйте:

```sh
unzip cherrypi-agent-linux-amd64- < sha > .zip
install -m 0755 cherrypi-agent /usr/local/bin/cherrypi-agent
```

Бинарнику нужны:

- Go **не** нужен во время работы.
- **Pi** в `PATH` (или `PI_BIN=/path/to/pi`). Установите совместимую с релизом
  CherryPi версию Pi.
- Креденшелы моделей в `$PI_CODING_AGENT_DIR` (по умолчанию `~/.pi/agent`).

Проверьте запуск:

```sh
CHERRYPI_AGENCY_URL=https://cherrypi.example.com \
  CHERRYPI_AGENT_TOKEN=pat_… \
  cherrypi-agent
```

### systemd

`/etc/systemd/system/cherrypi-agent.service`:

```ini
[Unit]
Description=CherryPi agent
After=network-online.target
Wants=network-online.target

[Service]
User=cherrypi
Group=cherrypi
Environment=CHERRYPI_AGENCY_URL=https://cherrypi.example.com
Environment=CHERRYPI_AGENT_TOKEN=pat_…
Environment=CHERRYPI_DIR=/var/lib/cherrypi
Environment=PI_CODING_AGENT_DIR=/var/lib/cherrypi/.pi/agent
ExecStart=/usr/local/bin/cherrypi-agent
Restart=on-failure
RestartSec=5
TimeoutStopSec=120

[Install]
WantedBy=multi-user.target
```

```sh
useradd --system --home /var/lib/cherrypi --create-home cherrypi
systemctl daemon-reload
systemctl enable --now cherrypi-agent
journalctl -u cherrypi-agent -f
```

Сервис должен запускать Pi и писать в `CHERRYPI_DIR`. Если используете
root-терминалы, пользователь сервиса — это пользователь shell; считайте это
привилегированным доступом.

## Проверка подключения

Страница агента в agency показывает **Online** после tunnel handshake. В логе
агента:

```text
CherryPi agent <version> starting
Tunneling to agency https://cherrypi.example.com as agent-username
```

Если агент остаётся offline, смотрите
[Решение проблем](/ru/advanced/troubleshooting#агент-остаётся-offline).

## Сетевые требования

| Направление | Порт | Зачем |
| --- | --- | --- |
| Агент → agency | `443` (или порт agency) | Туннель, API задач, история, вложения. |
| Агент → провайдер модели | по провайдеру | Запросы Pi к модели. |
| Браузер → agency | `443` | Дашборд. |

Агенту **не нужен входящий порт**. Его root-терминал доступен только через
аутентифицированный туннель agency.

## Рабочий стол (опционально)

Образ содержит `x11vnc` и X-утилиты. Чтобы показать уже запущенный X-дисплей,
передайте `DISPLAY` и смонтируйте сокет и authority-файл. См.
[Доступ к рабочему столу](/ru/agent/desktop).

## Обновление агента

Скачайте новый образ или замените бинарник и перезапустите сервис. Агент
переподключится, сообщит новую версию и восстановит прерванные задачи. Если
agency сильно новее, проверьте бейдж совместимости на странице агента.

## Далее

- [Настройка агента](/ru/agent/configuration)
- [Работа с агентами](/ru/using/agents)
- [Доступ к рабочему столу](/ru/agent/desktop)
