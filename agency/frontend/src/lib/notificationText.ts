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

const REFERENCE_LINK = /\[([^\]\n]*)\]\(ref:[^)]+\)/g;
const IMAGE = /!\[([^\]\n]*)\]\([^)]+\)/g;

/** Flatten a notification body to one readable line of plain text. */
export function plainNotificationText(body: string): string {
  return body
    .replace(REFERENCE_LINK, "$1")
    .replace(IMAGE, "$1")
    .replace(/\s+/g, " ")
    .trim();
}
