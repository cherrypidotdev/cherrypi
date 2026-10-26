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

import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { defineConfig } from "vite";

import { formatVersion } from "./src/lib/version";

// in as `VITE_APP_VERSION` (the Docker context has no `.git`); a local dev

function git(args: string[]): string {
  return execFileSync("git", args, { stdio: ["ignore", "pipe", "ignore"] })
    .toString()
    .trim();
}

function appVersion(): string {
  if (process.env.VITE_APP_VERSION) {
    return process.env.VITE_APP_VERSION;
  }
  try {
    const sha = git(["rev-parse", "--short", "HEAD"]);
    let tag = "";
    try {
      tag = git(["describe", "--tags", "--exact-match"]);
    } catch {}
    let branch = "";
    try {
      branch = git(["symbolic-ref", "--short", "HEAD"]);
    } catch {
      // A detached HEAD (tag checkout) is a main-line build.
    }
    let dirty = false;
    try {
      git(["diff", "--quiet", "--ignore-submodules"]);
    } catch {
      dirty = true;
    }
    const flavor =
      process.env.CHERRYPI_FLAVOR || (branch === "main" ? "main" : "dev");
    return formatVersion({ tag, flavor, sha, dirty });
  } catch {
    const pkg = JSON.parse(
      readFileSync(new URL("./package.json", import.meta.url), "utf8"),
    );
    return formatVersion({ flavor: "dev", sha: `v${pkg.version}` });
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  define: {
    __APP_VERSION__: JSON.stringify(appVersion()),
  },
  resolve: {
    alias: {
      "@": "/src",
    },
  },

  // router) or a dynamic import (`mermaid`). Pre-bundling them keeps Vite from

  optimizeDeps: {
    include: [
      "@bufbuild/protobuf",
      "@novnc/novnc",
      "@vueuse/integrations/useNProgress",
      "@xterm/addon-fit",
      "@xterm/xterm",
      "mermaid",
      "vue3-toastify",
    ],
  },
});
