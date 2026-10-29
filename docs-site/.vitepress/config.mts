// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { defineConfig, type DefaultTheme } from "vitepress";

const configuredBase = (process.env.DOCS_BASE ?? "/docs/").trim();
const base = configuredBase.endsWith("/")
  ? configuredBase
  : `${configuredBase}/`;
const repository = "https://github.com/cherrypidotdev/cherrypi";

// Nav items use VitePress path matching, which includes the configured base.

const basePattern = base.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const navMatch = (suffix: string) => `^${basePattern}${suffix}`;

const sidebarEn: DefaultTheme.SidebarItem[] = [
  {
    text: "Getting started",
    items: [
      { text: "What is CherryPi?", link: "/guide/what-is-cherrypi" },
      { text: "Quickstart", link: "/guide/quickstart" },
      { text: "Concepts", link: "/guide/concepts" },
    ],
  },
  {
    text: "Agency",
    items: [
      { text: "Install the agency", link: "/agency/install" },
      { text: "Configure the agency", link: "/agency/configuration" },
      { text: "Authentication & teams", link: "/agency/authentication" },
      { text: "Operate & upgrade", link: "/agency/operations" },
    ],
  },
  {
    text: "Agent",
    items: [
      { text: "Install an agent", link: "/agent/install" },
      { text: "Configure the agent", link: "/agent/configuration" },
      { text: "Desktop access", link: "/agent/desktop" },
    ],
  },
  {
    text: "Using CherryPi",
    items: [
      { text: "Projects & access", link: "/using/projects" },
      { text: "Tasks", link: "/using/tasks" },
      { text: "Agents", link: "/using/agents" },
      { text: "Attachments & artifacts", link: "/using/artifacts" },
      { text: "Terminals", link: "/using/terminals" },
    ],
  },
  {
    text: "Advanced",
    items: [
      { text: "JavaScript SDK", link: "/advanced/sdk" },
      { text: "Connect API", link: "/advanced/api" },
      { text: "Troubleshooting", link: "/advanced/troubleshooting" },
    ],
  },
  {
    text: "Reference",
    items: [
      { text: "Agency environment", link: "/reference/environment" },
      { text: "Agent environment", link: "/agent/configuration#environment" },
    ],
  },
];

const sidebarRu: DefaultTheme.SidebarItem[] = [
  {
    text: "Начало работы",
    items: [
      { text: "Что такое CherryPi?", link: "/ru/guide/what-is-cherrypi" },
      { text: "Быстрый старт", link: "/ru/guide/quickstart" },
      { text: "Основные понятия", link: "/ru/guide/concepts" },
    ],
  },
  {
    text: "Agency",
    items: [
      { text: "Установка agency", link: "/ru/agency/install" },
      { text: "Настройка agency", link: "/ru/agency/configuration" },
      { text: "Аутентификация и команды", link: "/ru/agency/authentication" },
      { text: "Эксплуатация и обновление", link: "/ru/agency/operations" },
    ],
  },
  {
    text: "Агент",
    items: [
      { text: "Установка агента", link: "/ru/agent/install" },
      { text: "Настройка агента", link: "/ru/agent/configuration" },
      { text: "Доступ к рабочему столу", link: "/ru/agent/desktop" },
    ],
  },
  {
    text: "Работа с CherryPi",
    items: [
      { text: "Проекты и доступ", link: "/ru/using/projects" },
      { text: "Задачи", link: "/ru/using/tasks" },
      { text: "Агенты", link: "/ru/using/agents" },
      { text: "Вложения и артефакты", link: "/ru/using/artifacts" },
      { text: "Терминалы", link: "/ru/using/terminals" },
    ],
  },
  {
    text: "Продвинутое",
    items: [
      { text: "JavaScript SDK", link: "/ru/advanced/sdk" },
      { text: "Connect API", link: "/ru/advanced/api" },
      { text: "Решение проблем", link: "/ru/advanced/troubleshooting" },
    ],
  },
  {
    text: "Справочник",
    items: [
      { text: "Переменные agency", link: "/ru/reference/environment" },
      {
        text: "Переменные агента",
        link: "/ru/agent/configuration#переменные-окружения",
      },
    ],
  },
];

const navEn: DefaultTheme.NavItem[] = [
  {
    text: "Guide",
    activeMatch: navMatch("(guide|agency|agent)/"),
    items: [
      { text: "What is CherryPi?", link: "/guide/what-is-cherrypi" },
      { text: "Quickstart", link: "/guide/quickstart" },
      { text: "Install the agency", link: "/agency/install" },
      { text: "Install an agent", link: "/agent/install" },
    ],
  },
  {
    text: "Using CherryPi",
    activeMatch: navMatch("using/"),
    items: [
      { text: "Projects", link: "/using/projects" },
      { text: "Tasks", link: "/using/tasks" },
      { text: "Agents", link: "/using/agents" },
      { text: "Artifacts", link: "/using/artifacts" },
    ],
  },
  {
    text: "Developers",
    activeMatch: navMatch("(advanced|reference)/"),
    items: [
      { text: "JavaScript SDK", link: "/advanced/sdk" },
      { text: "Connect API", link: "/advanced/api" },
      { text: "Environment reference", link: "/reference/environment" },
      { text: "Troubleshooting", link: "/advanced/troubleshooting" },
    ],
  },
  { text: "GitHub", link: repository },
];

const navRu: DefaultTheme.NavItem[] = [
  {
    text: "Руководство",
    activeMatch: navMatch("ru/(guide|agency|agent)/"),
    items: [
      { text: "Что такое CherryPi?", link: "/ru/guide/what-is-cherrypi" },
      { text: "Быстрый старт", link: "/ru/guide/quickstart" },
      { text: "Установка agency", link: "/ru/agency/install" },
      { text: "Установка агента", link: "/ru/agent/install" },
    ],
  },
  {
    text: "Работа с CherryPi",
    activeMatch: navMatch("ru/using/"),
    items: [
      { text: "Проекты", link: "/ru/using/projects" },
      { text: "Задачи", link: "/ru/using/tasks" },
      { text: "Агенты", link: "/ru/using/agents" },
      { text: "Артефакты", link: "/ru/using/artifacts" },
    ],
  },
  {
    text: "Разработчикам",
    activeMatch: navMatch("ru/(advanced|reference)/"),
    items: [
      { text: "JavaScript SDK", link: "/ru/advanced/sdk" },
      { text: "Connect API", link: "/ru/advanced/api" },
      { text: "Переменные окружения", link: "/ru/reference/environment" },
      { text: "Решение проблем", link: "/ru/advanced/troubleshooting" },
    ],
  },
  { text: "GitHub", link: repository },
];

export default defineConfig({
  base,
  lang: "en-US",
  title: "CherryPi",
  titleTemplate: ":title · CherryPi",
  description:
    "Self-hosted agency for fleets of AI agents: install, configure and operate CherryPi.",
  cleanUrls: true,

  ignoreDeadLinks: [/^http:\/\/localhost(?::\d+)?\/?$/],
  lastUpdated: true,
  head: [
    ["link", { rel: "icon", type: "image/png", href: `${base}favicon.png` }],
    ["link", { rel: "apple-touch-icon", href: `${base}favicon.png` }],
    ["meta", { name: "theme-color", content: "#18181b" }],
    ["meta", { property: "og:type", content: "website" }],
    ["meta", { property: "og:site_name", content: "CherryPi" }],
    ["meta", { property: "og:image", content: `${base}logo.png` }],
  ],
  themeConfig: {
    logo: "/logo.png",
    siteTitle: "CherryPi",
    search: {
      provider: "local",
      options: {
        translations: {
          button: {
            buttonText: "Search the docs",
            buttonAriaLabel: "Search the docs",
          },
          modal: {
            noResultsText: "No results for",
            resetButtonTitle: "Clear the search",
            footer: {
              selectText: "to select",
              navigateText: "to navigate",
              closeText: "to close",
            },
          },
        },
      },
    },
    socialLinks: [{ icon: "github", link: repository }],
    editLink: {
      pattern: `${repository}/edit/main/docs-site/:path`,
      text: "Edit this page on GitHub",
    },
    footer: {
      message:
        'Released under the <a href="https://github.com/cherrypidotdev/cherrypi/blob/main/LICENSE">AGPL-3.0 license</a>.',
      copyright: "Copyright © CherryPi authors",
    },
    nav: navEn,
    sidebar: sidebarEn,
    outline: { level: [2, 3], label: "On this page" },
    docFooter: { prev: "Previous page", next: "Next page" },
    lastUpdated: {
      text: "Last updated",
      formatOptions: { dateStyle: "medium", timeStyle: "short" },
    },
    returnToTopLabel: "Back to top",
    darkModeSwitchLabel: "Appearance",
    langMenuLabel: "Change language",
    sidebarMenuLabel: "Menu",
    skipToContentLabel: "Skip to content",
    externalLinkIcon: true,
  },
  // VitePress reads the language menu (and the per-locale theme config) from

  locales: {
    root: {
      label: "English",
      lang: "en-US",
    },
    ru: {
      label: "Русский",
      lang: "ru-RU",
      link: "/ru/",
      themeConfig: {
        nav: navRu,
        sidebar: sidebarRu,
        outline: { level: [2, 3], label: "На этой странице" },
        docFooter: {
          prev: "Предыдущая страница",
          next: "Следующая страница",
        },
        lastUpdated: {
          text: "Обновлено",
          formatOptions: { dateStyle: "medium", timeStyle: "short" },
        },
        editLink: {
          pattern: `${repository}/edit/main/docs-site/:path`,
          text: "Редактировать на GitHub",
        },
        returnToTopLabel: "Наверх",
        darkModeSwitchLabel: "Оформление",
        langMenuLabel: "Сменить язык",
        sidebarMenuLabel: "Меню",
        skipToContentLabel: "Перейти к содержимому",
        footer: {
          message:
            'Распространяется по лицензии <a href="https://github.com/cherrypidotdev/cherrypi/blob/main/LICENSE">AGPL-3.0</a>.',
          copyright: "© Авторы CherryPi",
        },
        search: {
          provider: "local",
          options: {
            translations: {
              button: {
                buttonText: "Поиск по документации",
                buttonAriaLabel: "Поиск по документации",
              },
              modal: {
                noResultsText: "Ничего не найдено:",
                resetButtonTitle: "Очистить запрос",
                footer: {
                  selectText: "выбрать",
                  navigateText: "перейти",
                  closeText: "закрыть",
                },
              },
            },
          },
        },
      },
    },
  },
  vite: {
    server: {
      port: 5174,
      strictPort: true,
    },
  },
});
