---
titleTemplate: false
layout: home
hero:
  name: CherryPi
  text: Your own agency for AI agents
  tagline: Deploy one central web app, connect workers on any machine, and run long-lived coding tasks from a browser.
  image:
    src: /logo.png
    alt: CherryPi
  actions:
    - theme: brand
      text: Quickstart
      link: /guide/quickstart
    - theme: alt
      text: What is CherryPi?
      link: /guide/what-is-cherrypi
    - theme: alt
      text: GitHub
      link: https://github.com/cherrypidotdev/cherrypi
features:
  - title: One agency, many agents
    details: The agency owns the UI, accounts, projects and scheduling. Lightweight agents dial out over a tunnel, so they need no inbound port.
  - title: Tasks that outlive a machine
    details: Prompts, session history, attachments and artifacts live in the agency. Reassign a task to another agent and continue where it stopped.
  - title: Built for real work
    details: Root terminals, browser desktops, per-agent concurrency limits, usage accounting and an inbox for everything that needs attention.
  - title: Self-hosted and open source
    details: SQLite or PostgreSQL, any S3-compatible bucket, OIDC/LDAP/SMTP support and an AGPL-3.0 codebase you can audit.
---

## Start here

<div class="vp-features">

### I want to try it

Run the single-box stack from the repository:

```sh
docker compose up -d --build
```

Then open <http://localhost:8080>. The bundled agent connects automatically.
Read the [quickstart](/guide/quickstart) for the details.

### I want to host it

Install the agency on a VPS, put it behind a TLS reverse proxy, and register
agents from the UI:

1. [Install the agency](/agency/install)
2. [Configure authentication and storage](/agency/configuration)
3. [Install an agent](/agent/install)

### I want to integrate

Use the typed [Connect API](/advanced/api) and the
[JavaScript SDK](/advanced/sdk), or drive it with the same personal access
tokens the browser and bundled Pi tools use.

</div>

## How it fits together

```mermaid
flowchart LR
  B["Browser"] -->|HTTPS| A["CherryPi agency<br/>Go + Vue SPA"]
  A -->|SQLite / PostgreSQL| DB[("Agency data")]
  A -->|S3| S3[("Attachments<br/>& artifacts")]
  A <-->|outbound tunnel| W1["Agent<br/>machine A"]
  A <-->|outbound tunnel| W2["Agent<br/>machine B"]
  W1 --> P1["Pi sessions<br/>terminals"]
  W2 --> P2["Pi sessions<br/>terminals"]
```

Only the agency is reachable from the outside. Agents connect to it, report
their health, and serve their task API over the same tunnel.
