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

/**
 * Version label rendering. The label identifies a build exactly:
 *
 *   [<tag>-]<flavor> (<short sha>[+dirty])
 *
 * e.g. `v0.1.0-main (48ea594)`, `dev (48ea594)`, `v0.1.0-public (0f2ab91)`.
 *
 * The flavor is the distribution line, not the branch name:
 *   - `dev`    anything built from an unmerged or local tree (the default)
 *   - `main`   a main-branch or tagged build of the origin repository
 *   - `public` the truncated snapshot built by the public GitHub repository
 *
 * The build injects the finished label as `VITE_APP_VERSION` (Docker and CI);
 * a local Vite dev server derives it from git with the same rules.
 */
export interface VersionParts {
  tag?: string;

  flavor: string;

  sha: string;

  dirty?: boolean;
}

export function formatVersion(parts: VersionParts): string {
  const base = parts.tag ? `${parts.tag}-${parts.flavor}` : parts.flavor;
  const sha = parts.dirty ? `${parts.sha}+dirty` : parts.sha;
  return `${base} (${sha})`;
}
