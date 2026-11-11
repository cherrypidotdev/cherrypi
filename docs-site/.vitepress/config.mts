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

import {
  defineConfig,
  type DefaultTheme,
  type MarkdownOptions,
} from "vitepress";
import { sidebarEn, sidebarRu } from "./sidebar.mjs";

// Markdown-It's type is re-exported by VitePress rather than pulled in as a

type MarkdownIt = Parameters<NonNullable<MarkdownOptions["config"]>>[0];
type FenceRule = NonNullable<MarkdownIt["renderer"]["rules"]["fence"]>;

const configuredBase = (process.env.DOCS_BASE ?? "/docs/").trim();
const base = configuredBase.endsWith("/")
  ? configuredBase
  : `${configuredBase}/`;
const repository = "https://github.com/cherrypidotdev/cherrypi";

// Nav items use VitePress path matching, which includes the configured base.

const basePattern = base.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const navMatch = (suffix: string) => `^${basePattern}${suffix}`;

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
      { text: "Sites", link: "/using/sites" },
    ],
  },
  {
    text: "Developers",
    activeMatch: navMatch("(advanced|reference)/"),
    items: [
      { text: "JavaScript SDK", link: "/advanced/sdk" },
      { text: "Connect API", link: "/advanced/api" },
      { text: "Plugins", link: "/advanced/plugins" },
      { text: "Testing plugins", link: "/advanced/plugin-testing" },
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
      { text: "Сайты", link: "/ru/using/sites" },
    ],
  },
  {
    text: "Разработчикам",
    activeMatch: navMatch("ru/(advanced|reference)/"),
    items: [
      { text: "JavaScript SDK", link: "/ru/advanced/sdk" },
      { text: "Connect API", link: "/ru/advanced/api" },
      { text: "Плагины", link: "/ru/advanced/plugins" },
      { text: "Тестирование плагинов", link: "/ru/advanced/plugin-testing" },
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

  srcExclude: ["public/**/*.md"],

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

  markdown: {
    config: (md) => renderMermaid(md),
  },
  vite: {
    server: {
      port: 5174,
      strictPort: true,
    },
  },
});

function renderMermaid(md: MarkdownIt): void {
  const fence: FenceRule =
    md.renderer.rules.fence ??
    ((tokens, index, options, _env, self) =>
      self.renderToken(tokens, index, options));

  md.renderer.rules.fence = (tokens, index, options, env, self) => {
    const token = tokens[index]!;
    const language = token.info.trim().split(/\s+/)[0];
    if (language !== "mermaid") {
      return fence(tokens, index, options, env, self);
    }

    return `<MermaidDiagram code="${encodeURIComponent(
      token.content.trim(),
    )}"></MermaidDiagram>\n`;
  };
}
