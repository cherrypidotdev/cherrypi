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

// The VitePress build catches template errors, but not broken internal links,

// script runs after `vitepress build` (and standalone via `npm run check`).

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const pagesRoot = root;
const publicRoot = path.join(root, "public");
const problems = [];

// Same base the VitePress config builds with: /docs/ in the agency, / for the

const configuredBase = (process.env.DOCS_BASE ?? "/docs/").trim();
const base = configuredBase.endsWith("/")
  ? configuredBase
  : `${configuredBase}/`;

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    if (entry.name === "node_modules" || entry.name === ".vitepress") continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) files.push(...walk(full));
    else if (entry.isFile() && entry.name.endsWith(".md")) files.push(full);
  }
  return files;
}

function isExternal(target) {
  return (
    target.startsWith("http://") ||
    target.startsWith("https://") ||
    target.startsWith("mailto:") ||
    target.startsWith("tel:")
  );
}

function pageFor(linkPath) {
  // A VitePress clean URL maps to <path>.md or <path>/index.md.
  const stripped = linkPath.startsWith("/") ? linkPath.slice(1) : linkPath;
  const clean = decodeURI(
    stripped.endsWith("/") ? stripped.slice(0, -1) : stripped,
  );
  const candidates = clean
    ? [
        path.join(pagesRoot, `${clean}.md`),
        path.join(pagesRoot, clean, "index.md"),
      ]
    : [path.join(pagesRoot, "index.md")];
  return candidates.find((candidate) => fs.existsSync(candidate));
}

const pages = walk(pagesRoot);
const linkPattern = /!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;

for (const page of pages) {
  const text = fs.readFileSync(page, "utf8");
  const relative = path.relative(root, page);
  let match;
  while ((match = linkPattern.exec(text)) !== null) {
    const raw = match[1];
    if (!raw || isExternal(raw) || raw.startsWith("#")) continue;
    const withoutAnchor = raw.split("#")[0];
    if (!withoutAnchor) continue;

    if (withoutAnchor.startsWith("/")) {
      if (
        withoutAnchor.startsWith("/screenshots/") ||
        /\.(png|jpe?g|webp|svg|ico)$/.test(withoutAnchor)
      ) {
        const assetPath = withoutAnchor.startsWith("/")
          ? withoutAnchor.slice(1)
          : withoutAnchor;
        const asset = path.join(publicRoot, assetPath);
        if (!fs.existsSync(asset)) {
          problems.push(`${relative}: missing asset ${raw}`);
        }
        continue;
      }
      if (!pageFor(withoutAnchor)) {
        problems.push(`${relative}: broken link ${raw}`);
      }
      continue;
    }

    // Relative links resolve against the page's directory.
    const target = path.resolve(path.dirname(page), withoutAnchor);
    if (!fs.existsSync(target)) {
      problems.push(`${relative}: broken relative link ${raw}`);
    }
  }
}

// Every English page has a Russian counterpart and vice versa (the landing page
// maps to ru/index.md).
function counterpart(page) {
  const relative = path.relative(root, page);
  if (relative.startsWith("ru" + path.sep)) {
    return path.join(root, relative.slice(3));
  }
  if (relative === "index.md") return path.join(root, "ru", "index.md");
  return path.join(root, "ru", relative);
}

for (const page of pages) {
  const other = counterpart(page);
  if (!fs.existsSync(other)) {
    problems.push(
      `${path.relative(root, page)}: missing ${path.relative(root, other)}`,
    );
  }
}

// Check that the configured VitePress base was applied to the built index,
// when a build exists.
const built = path.join(root, ".vitepress", "dist", "index.html");
if (fs.existsSync(built)) {
  const html = fs.readFileSync(built, "utf8");
  if (!html.includes(`${base}assets/`)) {
    problems.push(`built index.html does not reference ${base}assets/`);
  }
}

if (problems.length > 0) {
  console.error("Documentation check failed:");
  for (const problem of problems) console.error(`  - ${problem}`);
  process.exit(1);
}

console.log(`Documentation check passed (${pages.length} pages).`);
