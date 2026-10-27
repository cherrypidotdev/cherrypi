---
titleTemplate: false
layout: home
hero:
  name: CherryPi
  text: Ваше собственное агентство ИИ-агентов
  tagline: Разверните одно веб-приложение, подключите воркеры на любых машинах и запускайте долгие задачи из браузера.
  image:
    src: /logo.png
    alt: CherryPi
  actions:
    - theme: brand
      text: Быстрый старт
      link: /ru/guide/quickstart
    - theme: alt
      text: Что такое CherryPi?
      link: /ru/guide/what-is-cherrypi
    - theme: alt
      text: GitHub
      link: https://github.com/cherrypidotdev/cherrypi
features:
  - title: Одно агентство, много агентов
    details: Agency хранит интерфейс, аккаунты, проекты и планировщик. Легковесные агенты подключаются исходящим туннелем и не требуют входящего порта.
  - title: Задачи переживают машину
    details: Промпты, история сессий, вложения и артефакты хранятся в agency. Переназначьте задачу другому агенту и продолжите с того же места.
  - title: Для настоящей работы
    details: Root-терминалы, рабочий стол в браузере, лимиты параллельных задач, учёт использования и инбокс для всего, что требует внимания.
  - title: Self-hosted и открытый код
    details: SQLite или PostgreSQL, любой S3-совместимый бакет, поддержка OIDC/LDAP/SMTP и код под AGPL-3.0.
---

## С чего начать

<div class="vp-features">

### Хочу попробовать

Запустите стек на одной машине из репозитория:

```sh
docker compose up -d --build
```

Откройте <http://localhost:8080>. Встроенный агент подключится сам.
Подробности — в [быстром старте](/ru/guide/quickstart).

### Хочу разместить у себя

Установите agency на VPS, поставьте TLS-прокси и зарегистрируйте агентов в
интерфейсе:

1. [Установка agency](/ru/agency/install)
2. [Настройка аутентификации и хранилища](/ru/agency/configuration)
3. [Установка агента](/ru/agent/install)

### Хочу интегрироваться

Используйте типизированный [Connect API](/ru/advanced/api) и
[JavaScript SDK](/ru/advanced/sdk) либо те же personal access tokens, что и
браузер с встроенными инструментами Pi.

</div>

## Как это устроено

```mermaid
flowchart LR
  B["Браузер"] -->|HTTPS| A["CherryPi agency<br/>Go + Vue SPA"]
  A -->|SQLite / PostgreSQL| DB[("Данные agency")]
  A -->|S3| S3[("Вложения<br/>и артефакты")]
  A <-->|исходящий туннель| W1["Агент<br/>машина A"]
  A <-->|исходящий туннель| W2["Агент<br/>машина B"]
  W1 --> P1["Сессии Pi<br/>терминалы"]
  W2 --> P2["Сессии Pi<br/>терминалы"]
```

Снаружи доступна только agency. Агенты подключаются к ней, сообщают здоровье и
отдают свой API задач через тот же туннель.
