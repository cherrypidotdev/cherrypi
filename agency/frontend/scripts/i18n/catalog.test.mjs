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

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import {
  checkCatalog,
  collectBlocks,
  mergeCatalog,
  scanReferences,
  validateLocales,
} from "./catalog.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));

function fixture(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "i18n-catalog-"));
  for (const [name, body] of Object.entries(files)) {
    const file = path.join(dir, name);
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, body);
  }
  return dir;
}

function sfc(body, block) {
  return `<template lang="pug">\n${body}\n</template>\n\n<i18n lang="yaml">\n${block}\n</i18n>\n`;
}

const tests = [];
function test(name, fn) {
  tests.push([name, fn]);
}

test("a clean catalog passes", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.hello') }}",
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет",
    ),
  });
  const { errors, refs } = checkCatalog(root);
  assert.deepEqual(errors, []);
  assert.ok(refs.has("a.hello"));
});

test("a missing reference fails", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.missing') }}",
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(errors.join("\n"), /missing translation for t\("a\.missing"\)/);
});

test("an untranslated key fails", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.hello') }}",
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: ''",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(errors.join("\n"), /ru\."a\.hello" is empty/);
});

test("a misplaced placeholder fails", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.count', { count: 1 }) }}",
      "en:\n  a:\n    count: '{count} items'\nru:\n  a:\n    count: '{total} штук'",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(errors.join("\n"), /ru\."a\.count" is missing \{count\}/);
});

test("an unescaped @ fails", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.mail') }}",
      "en:\n  a:\n    mail: me@example.com\nru:\n  a:\n    mail: me@example.com",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(errors.join("\n"), /unescaped @/);
});

test("a leaf/branch collision is reported, not thrown", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('x.y') }}",
      "en:\n  x:\n    y: Leaf\n    z: Branch\nru:\n  x:\n    y: Лист\n    z: Ветка",
    ),
    "B.vue": sfc(
      "p {{ $t('x.y.z') }}",
      "en:\n  x:\n    y:\n      z: Nested\nru:\n  x:\n    y:\n      z: Вложенный",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(
    errors.join("\n"),
    /key "x\.y\.z" nests under "x\.y", which is already a leaf/,
  );
});

test("a duplicate key fails", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.hello') }}",
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет",
    ),
    "B.vue": sfc(
      "p x",
      "en:\n  a:\n    hello: Again\nru:\n  a:\n    hello: Снова",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(errors.join("\n"), /key "a\.hello" is already defined by/);
});

test("a dead key fails", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.hello') }}",
      "en:\n  a:\n    hello: Hi\n    dead: Unused\nru:\n  a:\n    hello: Привет\n    dead: Неиспользовано",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(errors.join("\n"), /dead translation "a\.dead"/);
});

test("a key property counts as a reference", () => {
  const root = fixture({
    "A.vue": `<template lang="pug">\np {{ $t(option.labelKey) }}\n</template>\n\n<script setup lang="ts">\nconst option = { labelKey: "a.hello" };\n</script>\n\n<i18n lang="yaml">\nen:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет\n</i18n>\n`,
  });
  const { errors } = checkCatalog(root);
  assert.deepEqual(errors, []);
});

test("the i18n:dynamic marker exempts a file's keys", () => {
  const root = fixture({
    "A.vue": `<template lang="pug">\np x\n</template>\n\n<script setup lang="ts">\n// i18n:dynamic — keys are picked from a map.\nconst labels = { a: "a.hello" };\nvoid labels;\n</script>\n\n<i18n lang="yaml">\nen:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет\n</i18n>\n`,
  });
  const { errors } = checkCatalog(root);
  assert.deepEqual(errors, []);
});

test("a computed family is exempt", () => {
  const root = fixture({
    "A.vue": sfc(
      "p x",
      "en:\n  taskState:\n    backlog: Backlog\nru:\n  taskState:\n    backlog: Бэклог",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.deepEqual(errors, []);
});

test("a key helper rendered without $t() fails", () => {
  const root = fixture({
    "lib/key.ts":
      "export function taskStateKey(state: string): string {\n  return `taskState.${state}`;\n}\n",
    "A.vue": sfc(
      'p {{ taskStateKey("backlog") }}',
      "en:\n  taskState:\n    backlog: Backlog\nru:\n  taskState:\n    backlog: Бэклог",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(
    errors.join("\n"),
    /taskStateKey\(\) is rendered without \$t\(\)/,
  );
});

test("undeclared template copy fails", () => {
  const root = fixture({
    "A.vue": sfc(
      `p Some untranslated sentence\n    button(title="Also English") x`,
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.match(
    errors.join("\n"),
    /untranslated text "Some untranslated sentence"/,
  );
  assert.match(errors.join("\n"), /untranslated title "Also English"/);
});

test("a protocol literal and a proper noun are not copy", () => {
  const root = fixture({
    "A.vue": sfc(
      `p(:title="kind === 'project' ? 'CherryPi' : undefined") {{ $t('a.hello') }}`,
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет",
    ),
  });
  const { errors } = checkCatalog(root);
  assert.deepEqual(errors, []);
});

test("the i18n:untranslated marker exempts a surface", () => {
  const root = fixture({
    "A.vue": `<template lang="pug">\np Development studio only · {{ $t('a.hello') }}\n</template>\n\n<script setup lang="ts">\n// i18n:untranslated — dev-only chrome.\n</script>\n\n<i18n lang="yaml">\nen:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет\n</i18n>\n`,
  });
  const { errors } = checkCatalog(root);
  assert.deepEqual(errors, []);
});

test("plural forms may differ in count across locales", () => {
  const messages = {
    en: { a: { count: "{n} item | {n} items" } },
    ru: { a: { count: "{n} штука | {n} штуки | {n} штук" } },
  };
  assert.deepEqual(validateLocales(messages), []);
});

test("blocks and references are collected from the tree", () => {
  const root = fixture({
    "A.vue": sfc(
      "p {{ $t('a.hello') }}",
      "en:\n  a:\n    hello: Hi\nru:\n  a:\n    hello: Привет",
    ),
  });
  const { blocks, errors } = collectBlocks(root);
  assert.deepEqual(errors, []);
  assert.equal(blocks.length, 1);
  assert.ok(scanReferences(root).has("a.hello"));
  const merged = mergeCatalog(blocks);
  assert.deepEqual(merged.errors, []);
  assert.equal(merged.messages.en.a.hello, "Hi");
});

let failed = 0;
for (const [name, fn] of tests) {
  try {
    fn();
    console.log(`ok - ${name}`);
  } catch (err) {
    failed += 1;
    console.error(`not ok - ${name}\n  ${err.message}`);
  }
}
if (failed > 0) {
  console.error(`${failed} of ${tests.length} i18n catalog tests failed`);
  process.exit(1);
}
console.log(`${tests.length} i18n catalog tests passed`);
void here;
