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

import { readdirSync, readFileSync } from "node:fs";

const POINTER = "version https://git-lfs.github.com/spec/v1";
const publicDir = new URL("../public/", import.meta.url);

function assets(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = new URL(entry.name + (entry.isDirectory() ? "/" : ""), dir);
    if (entry.isDirectory()) {
      return assets(path);
    }
    return /\.(png|ico|ttf|otf|woff2?)$/i.test(entry.name) ? [path] : [];
  });
}

const pointers = assets(publicDir)
  .filter((path) => readFileSync(path, "utf8").startsWith(POINTER))
  .map((path) => path.pathname.slice(publicDir.pathname.length));

if (pointers.length > 0) {
  console.error(
    `public/ contains Git LFS pointers (${pointers.join(", ")}). ` +
      "Materialize them first: `git lfs pull` (or check out with " +
      "actions/checkout `lfs: true`).",
  );
  process.exit(1);
}
