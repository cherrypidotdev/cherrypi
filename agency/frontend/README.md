# CherryPi dashboard

The per-agent Vue 3 SPA: sessions, their consoles, root terminals, usage,
settings, and the dashboard data sources. Built by Vite and embedded into the
Go binary.

## Stack

- **Vue 3** + **Pug** (`<template lang="pug">`) + **Tailwind v4** + **Vite**.
- **shadcn-vue** for UI components (`src/components/ui`, generated — leave
  as-is) and **lucide** icons (`@lucide/vue`).
- **vue-router** for navigation and **xterm.js** for terminals.
- The **`@cherrypi/sdk`** client (`./sdk/js`, a file dependency) for both the
  gRPC surface and the terminal WebSockets, in **ambient-auth** mode (the
  browser supplies the cached basic-auth credential; the SDK adds no header).

## Scripts

```sh
npm run dev       # Vite dev server (proxy the backend with CHERRYPI_FRONTEND_PROXY_URL)
npm run build     # vue-tsc + vite build -> dist/
npm run test:run  # Vitest browser mode (Playwright chromium)
```

## Adding components

```sh
npx shadcn-vue@latest add <component>
```

See [`../../AGENTS.md`](../../AGENTS.md) for the frontend conventions.
