#!/usr/bin/env node

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

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const output = path.join(root, "public", "screenshots");
const configuredURL =
  process.env.CHERRYPI_SCREENSHOT_URL ?? "http://127.0.0.1:8080";
const baseURL = configuredURL.endsWith("/")
  ? configuredURL.slice(0, -1)
  : configuredURL;
const user = process.env.CHERRYPI_SCREENSHOT_USER ?? "admin";
const password = process.env.CHERRYPI_SCREENSHOT_PASSWORD ?? "demodemo";
const taskID = process.env.CHERRYPI_SCREENSHOT_TASK ?? "WEB-1";
const agent = process.env.CHERRYPI_SCREENSHOT_AGENT ?? "builder";
const [width, height] = (
  process.env.CHERRYPI_SCREENSHOT_VIEWPORT ?? "1600x1000"
)
  .split("x")
  .map(Number);

const accent = "#e11d48";

fs.mkdirSync(output, { recursive: true });

const browser = await chromium.launch();
const context = await browser.newContext({
  viewport: { width, height },
  deviceScaleFactor: 2,
  colorScheme: "light",
  reducedMotion: "reduce",
});
const page = await context.newPage();
page.on("console", (message) => {
  if (message.type() === "warning") {
    console.log(`  browser warning: ${message.text()}`);
  }
});

try {
  await login();
  await captureTasksList();
  await captureTaskDetail();
  await captureAgentsList();
  await captureAgentDetail();
} finally {
  await browser.close();
}

console.log(`Screenshots written to ${path.relative(process.cwd(), output)}`);

async function login() {
  await page.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
  await page.fill("#identifier", user);
  await page.fill("#password", password);
  await Promise.all([
    page.waitForURL("**/tasks", { timeout: 30_000 }),
    page.click('button[type="submit"]'),
  ]);
  await settle();
}

async function captureTasksList() {
  await page.goto(`${baseURL}/tasks`, { waitUntil: "domcontentloaded" });
  await settle();
  await hideTransientErrors();

  await annotate([
    {
      find: { selector: '[aria-placeholder="Describe a task…"]' },
      label: "Start with a prompt",
      side: "top",
    },
    {
      find: { textIncludes: "All", selector: "button" },
      label: "Filter by state",
      side: "bottom",
    },
    {
      find: { placeholder: "Search title or prompt…" },
      label: "Search and narrow the list",
      side: "top",
    },
    {
      find: { text: "Fix the onboarding flow" },
      label: "Open a task",
      side: "right",
    },
  ]);
  await shot("tasks-list.png");
}

async function captureTaskDetail() {
  await page.goto(`${baseURL}/tasks/${taskID}`, {
    waitUntil: "domcontentloaded",
  });
  await settle();
  await hideTransientErrors();

  await annotate([
    {
      find: { text: "Fix the onboarding flow" },
      label: "Editable title",
      side: "bottom",
    },
    {
      find: { text: "Settled" },
      label: "Lifecycle state",
      side: "left",
    },
    {
      find: { textIncludes: "I reproduced it locally" },
      label: "Pi session history",
      side: "right",
    },
    {
      find: { selector: '[aria-placeholder="Send a follow-up prompt…"]' },
      label: "Continue the session",
      side: "top",
    },
    {
      find: { text: "General" },
      label: "Details drawer",
      side: "left",
    },
  ]);
  await shot("task-detail.png");
}

async function captureAgentsList() {
  await page.goto(`${baseURL}/agents`, { waitUntil: "domcontentloaded" });
  await settle();
  await hideTransientErrors();

  await annotate([
    {
      find: { text: "Register agent" },
      label: "Register a new worker",
      side: "right",
    },
    {
      find: { text: "Online" },
      label: "Live tunnel presence",
      side: "right",
    },
  ]);
  await shot("agents-list.png");
}

async function captureAgentDetail() {
  await page.goto(`${baseURL}/agents/${agent}`, {
    waitUntil: "domcontentloaded",
  });
  await settle();
  await hideTransientErrors();

  await annotate([
    {
      find: { text: "Online" },
      label: "Tunnel presence",
      side: "right",
    },
    {
      find: { textIncludes: "Overview" },
      label: "Logs, terminals, desktop",
      side: "bottom",
    },
    {
      find: { text: "Fix the onboarding flow" },
      label: "Recent tasks",
      side: "right",
    },
  ]);
  await shot("agent-detail.png");
}

async function settle() {
  await page.evaluate(() => document.fonts?.ready);

  await page.waitForTimeout(1200);
}

async function hideTransientErrors() {
  await page.evaluate(() => {
    for (const element of document.querySelectorAll(
      '[class*="bg-destructive"], [class*="text-destructive"]',
    )) {
      const text = element.textContent ?? "";
      if (text.includes("Connection lost") || text.includes("[not_found]")) {
        const banner = element.closest('[class*="border-b"]') ?? element;
        banner.style.display = "none";
      }
    }
  });
}

async function shot(name) {
  await page.screenshot({
    path: path.join(output, name),
    animations: "disabled",
  });
  console.log(`  ${name}`);
}

async function annotate(items) {
  await page.evaluate(
    ({ items, accent }) => {
      document.getElementById("docs-annotation-layer")?.remove();

      const layer = document.createElement("div");
      layer.id = "docs-annotation-layer";
      layer.style.cssText =
        "position:fixed;inset:0;z-index:2147483647;pointer-events:none";

      const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("width", String(window.innerWidth));
      svg.setAttribute("height", String(window.innerHeight));
      svg.style.cssText = "position:absolute;inset:0;overflow:visible";
      const defs = document.createElementNS(
        "http://www.w3.org/2000/svg",
        "defs",
      );
      defs.innerHTML = `
        <marker id="docs-arrow" viewBox="0 0 10 10" refX="8" refY="5"
          markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M 0 0 L 10 5 L 0 10 z" fill="${accent}"/>
        </marker>`;
      svg.append(defs);

      const targets = items.map((item, index) => {
        const element = resolve(item.find);
        if (!element) {
          console.warn("annotation target not found", item.label);
          return null;
        }
        const rect = element.getBoundingClientRect();
        if (rect.width < 4 || rect.height < 4) return null;

        const highlight = document.createElementNS(
          "http://www.w3.org/2000/svg",
          "rect",
        );
        highlight.setAttribute("x", String(rect.left - 4));
        highlight.setAttribute("y", String(rect.top - 4));
        highlight.setAttribute("width", String(rect.width + 8));
        highlight.setAttribute("height", String(rect.height + 8));
        highlight.setAttribute("rx", "10");
        highlight.setAttribute("fill", "none");
        highlight.setAttribute("stroke", accent);
        highlight.setAttribute("stroke-width", "2");
        highlight.setAttribute("stroke-dasharray", "6 5");
        highlight.setAttribute("opacity", "0.9");
        svg.append(highlight);

        return { item, index, rect, element };
      });

      layer.append(svg);
      document.body.append(layer);

      for (const target of targets) {
        if (!target) continue;
        const { item, index, rect } = target;
        const pill = document.createElement("div");
        pill.style.cssText = [
          "position:absolute",
          "display:flex",
          "align-items:center",
          "gap:8px",
          "padding:7px 12px 7px 8px",
          "border-radius:999px",
          "background:white",
          "color:#111827",
          "border:1px solid rgba(17,24,39,.12)",
          "box-shadow:0 8px 24px rgba(15,23,42,.18)",
          "font:600 13px/1.2 ui-sans-serif,system-ui,sans-serif",
          "white-space:nowrap",
        ].join(";");
        const number = document.createElement("span");
        number.textContent = String(index + 1);
        number.style.cssText = [
          "display:inline-flex",
          "align-items:center",
          "justify-content:center",
          "width:22px",
          "height:22px",
          "border-radius:999px",
          `background:${accent}`,
          "color:white",
          "font:700 12px/1 ui-sans-serif,system-ui,sans-serif",
        ].join(";");
        pill.append(number, document.createTextNode(item.label));
        layer.append(pill);

        const pillRect = pill.getBoundingClientRect();
        let x = rect.left;
        let y = rect.top;
        switch (item.side) {
          case "left":
            x = rect.left - pillRect.width - 54;
            y = rect.top + rect.height / 2 - pillRect.height / 2;
            break;
          case "right":
            x = rect.right + 54;
            y = rect.top + rect.height / 2 - pillRect.height / 2;
            break;
          case "top":
            x = rect.left + rect.width / 2 - pillRect.width / 2;
            y = rect.top - pillRect.height - 46;
            break;
          case "bottom":
          default:
            x = rect.left + rect.width / 2 - pillRect.width / 2;
            y = rect.bottom + 46;
            break;
        }
        x = Math.min(Math.max(10, x), window.innerWidth - pillRect.width - 10);
        y = Math.min(
          Math.max(10, y),
          window.innerHeight - pillRect.height - 10,
        );
        pill.style.left = `${x}px`;
        pill.style.top = `${y}px`;

        const pillCenter = {
          x: x + pillRect.width / 2,
          y: y + pillRect.height / 2,
        };
        const targetCenter = {
          x: rect.left + rect.width / 2,
          y: rect.top + rect.height / 2,
        };
        const dx = targetCenter.x - pillCenter.x;
        const dy = targetCenter.y - pillCenter.y;
        const length = Math.hypot(dx, dy) || 1;
        const edge = {
          x: pillCenter.x + (dx / length) * (pillRect.width / 2 + 2),
          y: pillCenter.y + (dy / length) * (pillRect.height / 2 + 2),
        };
        const control = {
          x: (edge.x + targetCenter.x) / 2 + (dy / length) * 18,
          y: (edge.y + targetCenter.y) / 2 - (dx / length) * 18,
        };
        const path = document.createElementNS(
          "http://www.w3.org/2000/svg",
          "path",
        );
        path.setAttribute(
          "d",
          `M ${edge.x} ${edge.y} Q ${control.x} ${control.y} ${targetCenter.x} ${targetCenter.y}`,
        );
        path.setAttribute("fill", "none");
        path.setAttribute("stroke", accent);
        path.setAttribute("stroke-width", "2.5");
        path.setAttribute("stroke-linecap", "round");
        path.setAttribute("marker-end", "url(#docs-arrow)");
        svg.append(path);
      }

      function resolve(find) {
        if (find.placeholder) {
          return (
            document.querySelector(`[placeholder="${find.placeholder}"]`) ??
            null
          );
        }
        if (find.text) {
          return smallest(
            [...scope(find.within)].filter(
              (element) =>
                element.offsetParent !== null &&
                (element.innerText ?? element.textContent ?? "").trim() ===
                  find.text,
            ),
          );
        }
        if (find.textIncludes) {
          const pool = find.selector
            ? [...document.querySelectorAll(find.selector)]
            : [...scope(find.within)];
          return smallest(
            pool.filter(
              (element) =>
                element.offsetParent !== null &&
                (element.innerText ?? element.textContent ?? "").includes(
                  find.textIncludes,
                ),
            ),
          );
        }
        if (find.within && find.selector) {
          return document.querySelector(`${find.within} ${find.selector}`);
        }
        if (find.selector) return document.querySelector(find.selector);
        return null;
      }

      function scope(within) {
        const root = within ? document.querySelector(within) : document.body;
        if (!root) return [];
        return [root, ...root.querySelectorAll("*")];
      }

      function smallest(elements) {
        return elements
          .filter((element) => {
            const rect = element.getBoundingClientRect();
            return rect.width > 4 && rect.height > 4;
          })
          .sort((a, b) => {
            const areaA =
              a.getBoundingClientRect().width *
              a.getBoundingClientRect().height;
            const areaB =
              b.getBoundingClientRect().width *
              b.getBoundingClientRect().height;
            return areaA - areaB;
          })[0];
      }
    },
    { items, accent },
  );
  const pills = await page.locator("#docs-annotation-layer > div").count();
  console.log(`  annotations: ${pills}/${items.length}`);
  await page.waitForTimeout(150);
}
