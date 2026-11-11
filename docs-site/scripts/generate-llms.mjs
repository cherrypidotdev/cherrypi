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

// VitePress copies public/ verbatim into the build (and serves it in dev), so

// before `vitepress build`; see package.json.

// Every page listed in .vitepress/sidebar.mjs needs a non-empty, single-line

// fails without it. VitePress directory indexes (`guide/index.md`) are reached

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { sidebarEn } from "../.vitepress/sidebar.mjs";
import { markdownLines } from "./markdown-lines.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const publicRoot = path.join(root, "public");

// Keep the base in sync with .vitepress/config.mts: /docs/ in the agency, /

const configuredBase = (process.env.DOCS_BASE ?? "/docs/").trim();
const base = configuredBase.endsWith("/")
  ? configuredBase
  : `${configuredBase}/`;
const baseWithoutSlash = base.slice(0, -1);

const linkPattern = /(!?\[[^\]]*\]\()([^)\s]+)((?:\s+"[^"]*")?\))/g;

const site = {
  title: "CherryPi",
  summary:
    "Self-hosted agency for fleets of AI agents: install, configure and operate CherryPi.",
  repository: "https://github.com/cherrypidotdev/cherrypi",
};

const pages = new Map();
for (const file of walk(root, { skipPublic: true })) {
  const rel = path.relative(root, file).split(path.sep).join("/");
  if (rel.startsWith("ru/")) continue;
  const clean = rel === "index.md" ? "" : rel.slice(0, -".md".length);
  const text = fs.readFileSync(file, "utf8");
  const { frontmatter, body } = splitFrontmatter(text);
  pages.set(clean, {
    clean,
    rel,
    text,
    frontmatter,
    body,
    ...parseFrontmatter(frontmatter),
  });
}

const sections = [];
const listed = new Set();
for (const group of sidebarEn) {
  const entries = [];
  collectSidebarItems(group.items, entries);
  if (entries.length === 0) continue;
  const section = { title: group.text, entries: [] };
  for (const item of entries) {
    const { clean, anchor } = splitLink(item.link);
    const pageKey = findPage(clean);
    if (pageKey === null) {
      fail(`sidebar links ${item.link}, but ${clean}.md does not exist`);
    }
    if (listed.has(pageKey) && !anchor) continue;
    const page = pages.get(pageKey);
    if (!page.description) {
      fail(`${page.rel} has no frontmatter description for llms.txt`);
    }
    listed.add(pageKey);
    section.entries.push({
      clean: pageKey,
      anchor,
      title: item.text ?? page.title,
      description: page.description,
    });
  }
  sections.push(section);
}

const unlisted = [...pages.values()]
  .filter((page) => page.clean !== "" && !listed.has(page.clean))
  .sort((a, b) => a.clean.localeCompare(b.clean));
if (unlisted.length > 0) {
  sections.push({
    title: "Other pages",
    entries: unlisted.map((page) => ({
      clean: page.clean,
      title: page.title ?? page.clean,
      description: page.description ?? "",
    })),
  });
}

const files = writeAgentFiles();
console.log(
  `Generated ${files.length} agent-facing docs files (${pages.size - 1} pages).`,
);

function walk(dir, { skipPublic = false } = {}) {
  const files = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (
      entry.name === "node_modules" ||
      entry.name === ".vitepress" ||
      (skipPublic && entry.name === "public")
    ) {
      continue;
    }
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(full, { skipPublic }));
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      files.push(full);
    }
  }
  return files;
}

function collectSidebarItems(items, out) {
  for (const item of items ?? []) {
    if (typeof item.link === "string") {
      out.push({ text: item.text, link: item.link });
    }
    if (Array.isArray(item.items)) {
      collectSidebarItems(item.items, out);
    }
  }
}

function splitLink(link) {
  const hashAt = link.indexOf("#");
  return {
    clean: (hashAt === -1 ? link : link.slice(0, hashAt)).replace(
      /^\/+|\/+$/g,
      "",
    ),
    anchor: hashAt === -1 ? "" : link.slice(hashAt),
  };
}

function findPage(clean) {
  const candidate = clean.replace(/^\/+|\/+$/g, "");
  const candidates =
    candidate === "" ? [""] : [candidate, `${candidate}/index`];
  return candidates.find((key) => pages.has(key)) ?? null;
}

function splitFrontmatter(text) {
  const match = /^(---\r?\n[\s\S]*?\r?\n---[ \t]*\r?\n?)/.exec(text);
  if (!match) return { frontmatter: "", body: text };
  return { frontmatter: match[1], body: text.slice(match[1].length) };
}

function parseFrontmatter(frontmatter) {
  const data = {};
  const inner = frontmatter
    .replace(/^---\r?\n/, "")
    .replace(/\r?\n---[ \t]*\r?\n?$/, "");
  for (const line of inner.split(/\r?\n/)) {
    const match = /^([A-Za-z0-9_-]+):\s*(.*)$/.exec(line);
    if (!match) continue;
    let value = match[2].trim();
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }
    if (value !== "") data[match[1]] = value;
  }
  return data;
}

function pageUrl(clean, anchor = "") {
  return clean === "" ? `${base}${anchor}` : `${base}${clean}.md${anchor}`;
}

function fileUrl(rel) {
  return `${base}${rel}`;
}

function writeAgentFiles() {
  fs.mkdirSync(publicRoot, { recursive: true });

  removeGeneratedMarkdown(publicRoot);

  const written = [];

  const index = [
    `# ${site.title}`,
    "",
    `> ${site.summary}`,
    "",
    "This file indexes the CherryPi documentation for language models. Every link",
    "points at the Markdown source of a page; fetch a page by appending `.md` to its",
    `URL. The complete text is available in [llms-full.txt](${fileUrl("llms-full.txt")}).`,
  ];
  for (const section of sections) {
    index.push("", `## ${section.title}`, "");
    for (const entry of section.entries) {
      index.push(
        `- [${entry.title}](${pageUrl(entry.clean, entry.anchor)})${entry.description ? `: ${entry.description}` : ""}`,
      );
    }
  }
  index.push(
    "",
    "## Optional",
    "",
    `- [Full documentation](${fileUrl("llms-full.txt")}): Every English page in a single Markdown file.`,
    `- [CherryPi repository](${site.repository}): Source code, releases and issue tracker.`,
    "",
  );
  writeFile("llms.txt", index.join("\n"));
  written.push("llms.txt");

  const full = [
    `# ${site.title} — full documentation`,
    "",
    `> ${site.summary}`,
    "",
    "Every English page of the CherryPi documentation, in reading order. Each",
    "section is the Markdown source served at the URL in its `source` comment.",
  ];
  const fullSeen = new Set();
  for (const section of sections) {
    for (const entry of section.entries) {
      if (fullSeen.has(entry.clean)) continue;
      fullSeen.add(entry.clean);
      const page = pages.get(entry.clean);
      full.push(
        "",
        "---",
        "",
        `<!-- source: ${pageUrl(page.clean)} -->`,
        "",
        rewriteMarkdown(page.body.trim(), page.rel),
        "",
      );
    }
  }
  writeFile("llms-full.txt", full.join("\n"));
  written.push("llms-full.txt");

  for (const page of pages.values()) {
    if (page.clean === "") continue;
    writeFile(
      page.rel,
      `${page.frontmatter}${rewriteMarkdown(page.body, page.rel)}`,
    );
    written.push(page.rel);
  }

  return written;
}

function writeFile(rel, contents) {
  const target = path.join(publicRoot, rel);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, contents);
}

function removeGeneratedMarkdown(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      removeGeneratedMarkdown(full);
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      fs.rmSync(full);
    }
  }
}

function rewriteMarkdown(markdown, fromRel) {
  const rewritten = [];
  for (const { line, code } of markdownLines(markdown)) {
    if (code) {
      rewritten.push(line);
      continue;
    }
    rewritten.push(
      rewriteReferenceDefinition(
        replaceOutsideCode(line, (chunk) => rewriteLinks(chunk, fromRel)),
        fromRel,
      ),
    );
  }
  return rewritten.join("\n");
}

// Run `render` on the parts of a line that are not inline code spans.
function replaceOutsideCode(line, render) {
  const code = [];
  let masked = "";
  let i = 0;
  while (i < line.length) {
    if (line[i] !== "`") {
      masked += line[i++];
      continue;
    }
    let ticks = 1;
    while (line[i + ticks] === "`") ticks++;
    const marker = "`".repeat(ticks);
    const close = line.indexOf(marker, i + ticks);
    if (close === -1) {
      masked += line[i++];
      continue;
    }
    masked += `\u0000${code.length}\u0000`;
    code.push(line.slice(i, close + ticks));
    i = close + ticks;
  }

  const rendered = render(masked);
  return rendered.replace(
    /\u0000(\d+)\u0000/g,
    (_, index) => code[Number(index)],
  );
}

function rewriteLinks(chunk, fromRel) {
  return chunk.replace(linkPattern, (match, open, target, close) => {
    let wrapped = "";
    let bare = target;
    if (target.startsWith("<") && target.endsWith(">")) {
      wrapped = "<";
      bare = target.slice(1, -1);
    }
    const rewritten = rewriteTarget(bare, fromRel);
    return rewritten === bare
      ? match
      : `${open}${wrapped}${rewritten}${wrapped ? ">" : ""}${close}`;
  });
}

function rewriteReferenceDefinition(line, fromRel) {
  const match = /^(\s{0,3}\[[^\]]+\]:\s*)(<[^>\s]+>|\S+)(.*)$/.exec(line);
  if (!match) return line;
  const [, prefix, target, rest] = match;
  const wrapped = target.startsWith("<") && target.endsWith(">");
  const bare = wrapped ? target.slice(1, -1) : target;
  const rewritten = rewriteTarget(bare, fromRel);
  return `${prefix}${wrapped ? `<${rewritten}>` : rewritten}${rest}`;
}

function rewriteTarget(target, fromRel) {
  if (!target || /^(?:[a-z][a-z0-9+.-]*:|\/\/|#)/i.test(target)) return target;

  let pathPart = target;
  let suffix = "";
  const hashAt = pathPart.indexOf("#");
  if (hashAt !== -1) {
    suffix = pathPart.slice(hashAt);
    pathPart = pathPart.slice(0, hashAt);
  }
  const queryAt = pathPart.indexOf("?");
  if (queryAt !== -1) {
    suffix = pathPart.slice(queryAt) + suffix;
    pathPart = pathPart.slice(0, queryAt);
  }
  if (!pathPart) return target;

  let clean;
  if (pathPart.startsWith("/")) {
    clean = pathPart.slice(1);
    if (
      baseWithoutSlash &&
      (clean === baseWithoutSlash.slice(1) ||
        clean.startsWith(`${baseWithoutSlash.slice(1)}/`))
    ) {
      clean = clean.slice(baseWithoutSlash.length);
    }
  } else {
    clean = path.posix.normalize(
      path.posix.join(path.posix.dirname(fromRel), pathPart),
    );
  }

  const page = resolvePage(clean);
  if (page !== null) return `${pageUrl(page)}${suffix}`;
  return `${base}${clean}${suffix}`;
}

function resolvePage(clean) {
  let candidate = clean;
  if (candidate.endsWith(".md")) candidate = candidate.slice(0, -3);
  return findPage(candidate);
}

function fail(message) {
  console.error(`llms.txt generation failed: ${message}`);
  process.exit(1);
}
