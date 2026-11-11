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

// Vite plugin that compiles the inline `<i18n>` blocks into a single global

import { checkCatalog, projectRoot, srcRoot } from "./catalog.mjs";

export const MESSAGES_ID = "virtual:cherrypi-i18n-messages";
const RESOLVED_ID = "\0" + MESSAGES_ID;

export function i18nPlugin() {
  let command = "serve";

  return {
    name: "cherrypi-i18n",
    enforce: "pre",

    configResolved(config) {
      command = config.command;
    },

    resolveId(id) {
      if (id === MESSAGES_ID) return RESOLVED_ID;
      return null;
    },

    handleHotUpdate({ file, server }) {
      if (!file.endsWith(".vue")) return;
      const mod = server.moduleGraph.getModuleById(RESOLVED_ID);
      if (mod) server.moduleGraph.invalidateModule(mod);
    },

    load(id) {
      // letting Vite try to serve the raw YAML as JavaScript.
      if (id !== RESOLVED_ID) {
        if (id.includes("vue&type=i18n") || id.includes("blockType=i18n")) {
          return "export default {}";
        }
        return null;
      }
      const { messages, errors, refs } = checkCatalog(srcRoot);
      if (errors.length > 0) {
        const list = errors.map((e) => `  ✗ ${e}`).join("\n");
        const error = new Error(
          `i18n check failed with ${errors.length} problem(s):\n${list}`,
        );
        error.name = "I18nError";
        throw error;
      }
      if (command === "serve") {
        this.info?.(`i18n: ${refs.size} keys across en/ru — ok`);
      }
      return `export default ${JSON.stringify({ en: messages.en, ru: messages.ru })}`;
    },
  };
}

export function runCheck() {
  const { messages, errors, refs, blocks } = checkCatalog(srcRoot);
  if (errors.length > 0) {
    console.error(
      `i18n check failed with ${errors.length} problem(s) under ${projectRoot}:\n` +
        errors.map((e) => `  ✗ ${e}`).join("\n"),
    );
    return 1;
  }
  console.log(
    `i18n ok: ${blocks.length} block(s), ${refs.size} referenced keys, ` +
      `${Object.keys(messages.en).length} top-level en namespaces`,
  );
  return 0;
}
