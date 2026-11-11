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

// Sidebar definitions shared by the VitePress config and the agent-facing

/** @type {import("vitepress").DefaultTheme.SidebarItem[]} */
export const sidebarEn = [
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
      { text: "Sites", link: "/using/sites" },
      { text: "Terminals", link: "/using/terminals" },
    ],
  },
  {
    text: "Advanced",
    items: [
      { text: "JavaScript SDK", link: "/advanced/sdk" },
      { text: "Connect API", link: "/advanced/api" },
      { text: "Plugins", link: "/advanced/plugins" },
      { text: "Testing plugins", link: "/advanced/plugin-testing" },
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

/** @type {import("vitepress").DefaultTheme.SidebarItem[]} */
export const sidebarRu = [
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
      { text: "Сайты", link: "/ru/using/sites" },
      { text: "Терминалы", link: "/ru/using/terminals" },
    ],
  },
  {
    text: "Продвинутое",
    items: [
      { text: "JavaScript SDK", link: "/ru/advanced/sdk" },
      { text: "Connect API", link: "/ru/advanced/api" },
      { text: "Плагины", link: "/ru/advanced/plugins" },
      { text: "Тестирование плагинов", link: "/ru/advanced/plugin-testing" },
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
