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

// This module is the single source of truth: `vite-plugin.mjs` compiles the

import { parse as parseSfc } from "@vue/compiler-sfc";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import lex from "pug-lexer";
import parse from "pug-parser";
import { parse as parseYaml } from "yaml";

const __dirname = dirname(fileURLToPath(import.meta.url));

export const projectRoot = resolve(__dirname, "..", "..");
export const srcRoot = join(projectRoot, "src");

export const LOCALES = ["en", "ru"];

const REFERENCE_EXTENSIONS = [".vue", ".ts"];
const BLOCK_LANG_DEFAULT = "yaml";

export function listFiles(dir, exts, out = []) {
  for (const name of readdirSync(dir)) {
    if (name === "node_modules" || name === "dist" || name === ".vitest") {
      continue;
    }
    const full = join(dir, name);
    const info = statSync(full);
    if (info.isDirectory()) {
      listFiles(full, exts, out);
    } else if (exts.includes(name.slice(name.lastIndexOf(".")))) {
      out.push(full);
    }
  }
  return out;
}

export function parseBlock(content, lang = BLOCK_LANG_DEFAULT) {
  if (lang === "json") return JSON.parse(content);
  if (lang === "yaml" || lang === "yml") return parseYaml(content) ?? {};
  throw new Error(`unsupported <i18n> lang: ${lang}`);
}

export function collectBlocks(root = srcRoot) {
  const blocks = [];
  const errors = [];
  for (const file of listFiles(root, [".vue"])) {
    const rel = relative(projectRoot, file);
    const source = readFileSync(file, "utf8");
    let descriptor;
    try {
      ({ descriptor } = parseSfc(source, { filename: file }));
    } catch (err) {
      errors.push(`${rel}: cannot parse SFC: ${err.message}`);
      continue;
    }
    for (const block of descriptor.customBlocks) {
      if (block.type !== "i18n") continue;
      const lang = block.lang || BLOCK_LANG_DEFAULT;
      let messages;
      try {
        messages = parseBlock(block.content, lang);
      } catch (err) {
        errors.push(
          `${rel}:${block.loc.start.line} invalid ${lang}: ${err.message}`,
        );
        continue;
      }
      const locales = Object.keys(messages);
      if (locales[0] !== "en") {
        errors.push(
          `${rel}:${block.loc.start.line} first locale in <i18n> must be "en", ` +
            `found "${locales[0] ?? "none"}"`,
        );
      }
      for (const locale of locales) {
        if (!LOCALES.includes(locale)) {
          errors.push(
            `${rel}:${block.loc.start.line} unsupported locale "${locale}"`,
          );
        }
      }
      blocks.push({
        file,
        rel,
        line: block.loc.start.line,
        lang,
        messages,
        loc: block.loc,
      });
    }
  }
  return { blocks, errors };
}

function isPlainObject(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function flatten(tree, prefix = "", out = new Map(), errors = []) {
  if (tree === null || tree === undefined) return out;
  for (const [key, value] of Object.entries(tree)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (isPlainObject(value)) {
      flatten(value, path, out, errors);
    } else if (typeof value === "string") {
      out.set(path, value);
    } else {
      errors.push(`"${path}" must be a string, got ${typeof value}`);
    }
  }
  return out;
}

const KEY_RE = /^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$/;

export function mergeCatalog(blocks) {
  const messages = Object.fromEntries(LOCALES.map((l) => [l, {}]));
  const owners = new Map();
  const errors = [];

  for (const block of blocks) {
    block.id = `${block.rel}:${block.line}`;
    for (const locale of LOCALES) {
      const tree = block.messages[locale];
      if (tree === undefined) continue;
      const leaves = flatten(tree, "", new Map());
      for (const [path, value] of leaves) {
        if (!KEY_RE.test(path)) {
          errors.push(`${block.rel}: invalid key "${path}"`);
        }
        const owner = owners.get(path);
        if (owner && owner.id !== block.id) {
          errors.push(
            `${block.rel}: key "${path}" is already defined by ${owner.label}`,
          );
          continue;
        }
        owners.set(path, { ...block, label: block.id });
        setLeaf(messages[locale], path, value, block, errors);
      }
    }
  }
  return { messages, owners, errors };
}

function setLeaf(tree, path, value, block, errors) {
  const parts = path.split(".");
  let node = tree;
  for (let i = 0; i < parts.length - 1; i += 1) {
    const key = parts[i];
    const branch = parts.slice(0, i + 1).join(".");
    if (typeof node[key] === "string") {
      errors.push(
        `${block.rel}: key "${path}" nests under "${branch}", which is already a leaf`,
      );
      return;
    }
    node[key] ??= {};
    node = node[key];
  }
  const leaf = parts.at(-1);
  if (typeof node[leaf] === "object" && node[leaf] !== null) {
    errors.push(
      `${block.rel}: key "${path}" is already a branch (it has nested keys)`,
    );
    return;
  }
  node[leaf] = value;
}

export function placeholders(message) {
  return new Set(
    [...message.matchAll(/\{([a-zA-Z0-9_]+)\}/g)].map((m) => m[1]),
  );
}

function validateSyntax(path, locale, message) {
  const errors = [];
  const unescaped = message.replace(/\{'@'\}/g, "");
  if (unescaped.includes("@")) {
    errors.push(
      `${locale}."${path}" has an unescaped @ (write {'@'} for a literal at-sign)`,
    );
  }
  return errors;
}

function validateShape(tree, locale, errors, prefix = "") {
  for (const [key, value] of Object.entries(tree)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (isPlainObject(value)) {
      if (flatten(value).size === 0) {
        errors.push(`${locale}."${path}" has no messages`);
      }
      validateShape(value, locale, errors, path);
    }
  }
}

function pluralPlaceholders(message) {
  const out = new Set();
  for (const form of message.split("|")) {
    for (const name of placeholders(form.trim())) out.add(name);
  }
  return out;
}

export function validateLocales(messages) {
  const errors = [];
  for (const locale of LOCALES) validateShape(messages[locale], locale, errors);
  const en = flatten(messages.en, "", new Map());
  const ru = flatten(messages.ru, "", new Map());
  for (const [path, value] of en) {
    if (value.trim() === "") errors.push(`en."${path}" is empty`);
    errors.push(...validateSyntax(path, "en", value));
    if (!ru.has(path)) {
      errors.push(`ru is missing "${path}" (en: "${value}")`);
      continue;
    }
    const ruValue = ru.get(path);
    if (ruValue.trim() === "") errors.push(`ru."${path}" is empty`);
    errors.push(...validateSyntax(path, "ru", ruValue));

    const a = pluralPlaceholders(value);
    const b = pluralPlaceholders(ruValue);
    for (const name of a) {
      if (!b.has(name)) errors.push(`ru."${path}" is missing {${name}}`);
    }
    for (const name of b) {
      if (!a.has(name)) errors.push(`en."${path}" is missing {${name}}`);
    }
  }
  for (const [path] of ru) {
    if (!en.has(path))
      errors.push(`en is missing "${path}" (ru: "${ru.get(path)}")`);
  }
  return errors;
}

const REFERENCE_RE =
  /(?:\$t|\bi18n\.t|(?<![\w$.])t)\(\s*(['"`])([^'"`\\]+)\1|v-t="(['"])([^'"\\]+)\3/g;

const KEY_PROPERTY_RE =
  /\b(?:[a-zA-Z]*[Kk]ey)\s*:\s*(['"])([a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)+)\1/g;

export function scanReferences(root = srcRoot) {
  const refs = new Map();
  for (const file of listFiles(root, REFERENCE_EXTENSIONS)) {
    const rel = relative(projectRoot, file);
    const source = readFileSync(file, "utf8");
    for (const match of source.matchAll(REFERENCE_RE)) {
      const key = match[2] ?? match[4];
      if (!key) continue;
      const list = refs.get(key) ?? [];
      if (!list.includes(rel)) list.push(rel);
      refs.set(key, list);
    }
    for (const match of source.matchAll(KEY_PROPERTY_RE)) {
      const key = match[2];
      if (!key) continue;
      const list = refs.get(key) ?? [];
      if (!list.includes(rel)) list.push(rel);
      refs.set(key, list);
    }
  }
  return refs;
}

export const DYNAMIC_KEY_FAMILIES = [
  "taskState.",
  "taskPriority.",
  "taskSize.",
  "taskColumn.",
  "taskRefs.kind.",
  "projectRole.",
  "projectStatus.",
  "newTaskRelation.",
  "domainLog.kind.",
  "taskLog.kind.",

  "attention.event.",
  "pluginLog.level.",
  "soundSet.",
  "usage.range.",
  "taskDetails.prompts.",
  "taskDetails.logsTab.",
  "pluginScope.",
  "pluginHandler.",
  "argType.",
  "pageTitle.",
  "login.errors.",
];

/**
 * A file opts the keys it owns out of the dead-key report with this comment.
 *
 * Both markers are load-bearing outside this check: the public snapshot's
 * `.gitea/scripts/strip-comments.mjs` keeps `i18n:` comments in its
 * `directivePattern`, because every release build runs this check over the
 * comment-stripped tree. Renaming a marker means updating that pattern too.
 */
const DYNAMIC_MARKER = "i18n:dynamic";

const UNTRANSLATED_MARKER = "i18n:untranslated";

function isDynamicKey(key) {
  return DYNAMIC_KEY_FAMILIES.some((family) => key.startsWith(family));
}

/**
 * Files that carry the `// i18n:dynamic` marker, mapped to their project
 * relative path: a component that assembles keys from a map it owns cannot be
 * verified statically either.
 */
function markedFiles(root) {
  const marked = new Set();
  for (const file of listFiles(root, REFERENCE_EXTENSIONS)) {
    const source = readFileSync(file, "utf8");
    if (source.includes(DYNAMIC_MARKER))
      marked.add(relative(projectRoot, file));
  }
  return marked;
}

const TRANSLATED_RE = /(?:\$t|\bt)\(\s*[a-zA-Z_$]/;

function catalogKeyHelpers() {
  const helpers = new Set();
  for (const file of listFiles(join(srcRoot, "lib"), [".ts"])) {
    const source = readFileSync(file, "utf8");
    for (const match of source.matchAll(
      /export function ([a-zA-Z][\w]*Key(?:For)?)\(/g,
    )) {
      helpers.add(match[1]);
    }
  }
  return helpers;
}

function findUnwrappedKeys(root) {
  const errors = [];
  const helpers = catalogKeyHelpers();
  for (const file of listFiles(root, [".vue"])) {
    const rel = relative(projectRoot, file);
    const source = readFileSync(file, "utf8");
    const bindings = [
      ...source.matchAll(/\{\{([^}]*)\}\}/g),
      ...source.matchAll(/:[\w-]+="([^"]*)"/g),
    ];
    for (const match of bindings) {
      const expr = match[1] ?? "";
      const call = [...expr.matchAll(/\b([a-zA-Z][\w]*)\(/g)].find((match) =>
        helpers.has(match[1]),
      );
      if (!call) continue;

      if (TRANSLATED_RE.test(expr)) continue;
      if (/labelKey|descriptionKey|nameKey|hintKey/.test(expr)) continue;
      errors.push(
        `${rel}: ${call[1]}() is rendered without $t() — the key would be ` +
          `printed instead of the translation`,
      );
    }
  }
  return errors;
}

const UNTRANSLATED_ALLOW = new Set([
  "CherryPi",
  "Jinja",
  "agent",
  "RDP",
  "LDAP",
  "OIDC",
  "OAuth2",
  "SMTP",
  "TOTP",
  "TTFT",
  "TTC",
  "CPU",
  "GPU",
  "API",
  "CLI",
  "SDK",
  "CSV",
  "JSON",
  "URL",
  "HTTP",
  "HTTPS",
  "Lua",
  "Pi",
]);

const COPY_ATTRS = new Set([
  "title",
  "placeholder",
  "aria-label",
  "alt",

  "label",
]);

const looksTranslatable = (value) => {
  const plain = value
    .replace(/\{\{[\s\S]*?\}\}/g, " ")
    .replace(/\$\{[\s\S]*?\}/g, " ")
    .trim();
  if (/\$t\(/.test(value)) return false;

  if (plain.startsWith("/")) return false;

  const words = plain
    .replace(/[^A-Za-z]+/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  if (words.length > 0 && words.every((word) => UNTRANSLATED_ALLOW.has(word))) {
    return false;
  }
  return /[A-Za-z]{2,}/.test(plain);
};

function expressionLiterals(expression) {
  const out = [];
  const re = /(['"`])((?:\\.|(?!\1)[\s\S])*?)\1/g;
  for (const match of expression.matchAll(re)) {
    const before = expression.slice(0, match.index).trimEnd();

    if (/(?:\$t|(?:^|[^\w$.])t)\(?\s*$/.test(before)) continue;
    if (/(?:===|!==|==|!=)$/.test(before)) continue;
    out.push(match[2]);
  }
  return out;
}

function findUntranslatedTemplates(root) {
  const errors = [];
  for (const file of listFiles(root, [".vue"])) {
    const rel = relative(projectRoot, file);
    const raw = readFileSync(file, "utf8");
    if (raw.includes(UNTRANSLATED_MARKER)) continue;
    const { descriptor } = parseSfc(raw, { filename: file });
    if (descriptor.template?.lang !== "pug") continue;
    const base = descriptor.template.loc.start.line;
    const ast = parse(lex(descriptor.template.content, { filename: file }));
    const report = (line, kind, value) => {
      errors.push(
        `${rel}:${base + line - 1} untranslated ${kind} "${value.slice(0, 60)}" — wrap it in $t()`,
      );
    };
    const walk = (node) => {
      if (!node || typeof node !== "object") return;
      if (node.type === "Text" && looksTranslatable(node.val)) {
        report(node.line, "text", node.val.trim());
      }
      if (node.type === "Tag") {
        for (const attr of node.attrs ?? []) {
          const bound =
            attr.name.startsWith(":") || attr.name.startsWith("v-bind:");
          const name = attr.name.replace(/^(:|v-bind:)/, "");
          if (!COPY_ATTRS.has(name) || typeof attr.val !== "string") continue;

          const literal = attr.val.trim();
          const quoted =
            (literal.startsWith('"') && literal.endsWith('"')) ||
            (literal.startsWith("'") && literal.endsWith("'"));
          const expression = quoted ? literal.slice(1, -1) : literal;
          if (bound) {
            for (const literalValue of expressionLiterals(expression)) {
              if (looksTranslatable(literalValue)) {
                report(attr.line, name, literalValue);
              }
            }
            continue;
          }
          if (quoted && looksTranslatable(expression)) {
            report(attr.line, name, expression);
          }
        }
      }
      for (const child of node.nodes ?? []) walk(child);
      for (const block of node.block?.nodes ?? []) walk(block);
      for (const attr of node.attrs ?? [])
        for (const block of attr.block?.nodes ?? []) walk(block);
    };
    walk(ast);
  }
  return errors;
}

function findDeadKeys(refs, owners, root) {
  const marked = markedFiles(root);
  const errors = [];
  for (const key of owners.keys()) {
    if (refs.has(key) || isDynamicKey(key)) continue;
    const owner = owners.get(key);
    if (marked.has(owner.rel)) continue;
    errors.push(
      `dead translation "${key}" defined by ${owner.rel} is never referenced ` +
        `(remove it, reference it, or mark the file with ${DYNAMIC_MARKER})`,
    );
  }
  return errors;
}

export function checkCatalog(root = srcRoot) {
  const { blocks, errors: blockErrors } = collectBlocks(root);
  const { messages, owners, errors: mergeErrors } = mergeCatalog(blocks);
  const errors = [...blockErrors, ...mergeErrors, ...validateLocales(messages)];

  const refs = scanReferences(root);
  for (const [key, files] of refs) {
    if (!hasKey(messages.en, key)) {
      errors.push(
        `missing translation for t("${key}") referenced by ${files.join(", ")}`,
      );
    }
  }
  errors.push(...findDeadKeys(refs, owners, root));
  errors.push(...findUnwrappedKeys(root));
  errors.push(...findUntranslatedTemplates(root));
  return { blocks, messages, refs, owners, errors };
}

export function hasKey(tree, path) {
  let node = tree;
  for (const part of path.split(".")) {
    if (!isPlainObject(node) || !(part in node)) return false;
    node = node[part];
  }
  return typeof node === "string";
}

export function getKey(tree, path) {
  let node = tree;
  for (const part of path.split(".")) {
    if (!isPlainObject(node) || !(part in node)) return undefined;
    node = node[part];
  }
  return typeof node === "string" ? node : undefined;
}
