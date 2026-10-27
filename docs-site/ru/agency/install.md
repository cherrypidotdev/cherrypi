---
title: Установка agency
description: Разверните CherryPi agency через Docker или бинарник, за TLS reverse proxy.
---

# Установка agency

Agency — один Go-бинарник со встроенными дашбордом и этой документацией. Ему
нужны каталог данных, HTTP-порт и, для вложений, S3-совместимый бакет. Отдельный
веб-сервер или очередь сообщений не требуются.

## Требования

| Требование | Примечания |
| --- | --- |
| Linux-хост | Опубликованный образ — `linux/amd64`; бинарники есть для Linux и macOS, amd64 и arm64. |
| 1–2 vCPU, 1 ГБ RAM | Agency лёгкая; тяжёлую работу делают агенты. |
| Домен + TLS | Нужны для безопасных cookie и реалистичных OIDC callback. Используйте Caddy, Traefik, nginx или балансировщик. |
| Опционально PostgreSQL | SQLite по умолчанию и вполне достаточен для одного экземпляра. |
| Опционально S3-бакет | Нужен для вложений, аватаров и артефактов. Подходят MinIO, Garage, Cloudflare R2 и AWS S3. |

## Вариант A — Docker-образ

Опубликованные образы лежат на GHCR:

```sh
docker pull ghcr.io/cherrypidotdev/cherrypi-agency:latest
```

Минимальный production-контейнер:

```sh
docker run -d --name cherrypi-agency \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /srv/cherrypi/agency:/root/.cherrypi-agency \
  -e CHERRYPI_PUBLIC_URL='https://cherrypi.example.com' \
  -e CHERRYPI_ADMIN_USERNAME='admin' \
  -e CHERRYPI_ADMIN_PASSWORD='храните-в-секретах' \
  ghcr.io/cherrypidotdev/cherrypi-agency:latest
```

Порт привязан к localhost, поэтому до него дойдёт только reverse proxy.
Монтируйте каталог данных на постоянное хранилище: там SQLite-база и
process-логи.

Добавьте объектное хранилище:

```sh
-e CHERRYPI_S3_ENDPOINT='https://s3.example.com' \
  -e CHERRYPI_S3_BUCKET='cherrypi' \
  -e CHERRYPI_S3_REGION='us-east-1' \
  -e CHERRYPI_S3_ACCESS_KEY_ID='…' \
  -e CHERRYPI_S3_SECRET_ACCESS_KEY='…'
```

::: tip MinIO за reverse proxy
Если endpoint выглядит как `https://s3.example.com`, задайте
`CHERRYPI_S3_FORCE_PATH_STYLE=true`. Иначе agency запрашивает
`https://<bucket>.s3.example.com/...`, а для этого нужен wildcard-сертификат.
Agency проверяет endpoint и бакет при старте и падает с подсказкой, если они
недоступны.
:::

Полный список переменных — в [Настройка agency](/ru/agency/configuration).

## Вариант B — релизный бинарник

Каждый релиз публикует архивы
`cherrypi-agency-<os>-<arch>-<sha>.zip`. Скачайте нужный со
[страницы релизов](https://github.com/cherrypidotdev/cherrypi/releases), затем:

```sh
unzip cherrypi-agency-linux-amd64- < sha > .zip
install -m 0755 cherrypi-agency /usr/local/bin/cherrypi-agency
useradd --system --home /var/lib/cherrypi --create-home cherrypi
install -d -o cherrypi -g cherrypi /var/lib/cherrypi/agency
```

Создайте `/etc/cherrypi/agency.env`:

```sh
CHERRYPI_AGENCY_DIR=/var/lib/cherrypi/agency
CHERRYPI_AGENCY_PORT=8080
CHERRYPI_PUBLIC_URL=https://cherrypi.example.com
CHERRYPI_ADMIN_USERNAME=admin
CHERRYPI_ADMIN_PASSWORD=храните-в-секретах
CHERRYPI_S3_ENDPOINT=https://s3.example.com
CHERRYPI_S3_BUCKET=cherrypi
CHERRYPI_S3_REGION=us-east-1
CHERRYPI_S3_ACCESS_KEY_ID=…
CHERRYPI_S3_SECRET_ACCESS_KEY=…
```

И unit, `/etc/systemd/system/cherrypi-agency.service`:

```ini
[Unit]
Description=CherryPi agency
After=network-online.target
Wants=network-online.target

[Service]
User=cherrypi
Group=cherrypi
EnvironmentFile=/etc/cherrypi/agency.env
ExecStart=/usr/local/bin/cherrypi-agency
Restart=on-failure
RestartSec=3
TimeoutStopSec=90

[Install]
WantedBy=multi-user.target
```

```sh
systemctl daemon-reload
systemctl enable --now cherrypi-agency
journalctl -u cherrypi-agency -f
```

## Первый запуск

На пустой установке agency создаёт первого администратора из
`CHERRYPI_ADMIN_USERNAME` (по умолчанию `admin`) и
`CHERRYPI_ADMIN_PASSWORD`. Если пароль не задан, он генерируется и печатается
**один раз** в лог:

```sh
docker logs cherrypi-agency 2>&1 | grep -i password
```

Войдите и сразу:

1. Смените пароль администратора в **Account**.
2. Пригласите пользователей в **Settings → Users**.
3. Настройте SMTP и/или OIDC/LDAP в **Settings → Authentication**, если нужен
   вход без пароля или через каталог.
4. Настройте имя agency, логотип и основной цвет в **Settings → General**.
5. Проверьте, что **Settings → Storage** видит рабочий бакет.

Администратор из переменных окружения — только bootstrap: после появления
аккаунтов он их не перезаписывает.

## Reverse proxy

Agency говорит по HTTP/1.1 и, опционально, по cleartext HTTP/2. Терминалы,
рабочий стол, инбокс и fallback туннеля идут по обычным WebSocket. Обычного
reverse proxy достаточно.

::: warning Не форсируйте h2c перед CherryPi
HTTP/2 запрещает заголовок `Upgrade`, поэтому прокси, который сначала говорит
h2c, ломает WebSocket-fallback терминалов, рабочего стола и туннеля. Держите
upstream на HTTP/1.1. `CHERRYPI_H2C=1` нужен только для прямого plaintext-
подключения к агенту, не для браузерной стороны.
:::

Отключите **read timeout** прокси для долгих потоков. С Caddy:

```text
cherrypi.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Caddy по умолчанию поддерживает WebSocket и стриминг. С Traefik v3:

```yaml
entryPoints:
  websecure:
    address: ":443"
    transport:
      respondingTimeouts:
        readTimeout: 0
        idleTimeout: 0
```

Задайте `CHERRYPI_PUBLIC_URL` равным внешнему origin (без слэша на конце). Он
используется для magic link, сброса пароля и OIDC redirect.

## Проверка здоровья

```sh
curl -fsS http://127.0.0.1:8080/api/rest/health
# OK
```

Docker-образ уже содержит health check на этот endpoint.

## Далее

- [Настройка agency](/ru/agency/configuration)
- [Аутентификация](/ru/agency/authentication)
- [Установка агента](/ru/agent/install)
- [Эксплуатация и обновление](/ru/agency/operations)
