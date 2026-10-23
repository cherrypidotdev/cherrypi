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
import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { defineConfig } from "vite";

// The dashboard footer shows the app version. Release builds pass it in as
// `VITE_APP_VERSION` (the Docker build context has no `.git`); a local build
// derives it from git, falling back to `package.json` without a repo.
function appVersion(): string {
  if (process.env.VITE_APP_VERSION) {
    return process.env.VITE_APP_VERSION;
  }
  try {
    return execSync("git describe --tags --always --dirty", {
      stdio: ["ignore", "pipe", "ignore"],
    })
      .toString()
      .trim();
  } catch {
    const pkg = JSON.parse(
      readFileSync(new URL("./package.json", import.meta.url), "utf8"),
    );
    return `v${pkg.version}`;
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
  // Some deps are only reached through a lazy view (the router statically
  // imports every view, so `@novnc/novnc` shows up once the suite touches the
  // router) or a dynamic import (`mermaid`). Pre-bundling them keeps Vite from
  // discovering them mid-run and reloading the page, which invalidates
  // in-flight modules and makes the browser suite flaky.
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
