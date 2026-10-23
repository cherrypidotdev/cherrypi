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

import * as tldraw from "tldraw";

// CherryPi is a self-hosted deployment that ships without a tldraw license
// key. In a production bundle, tldraw's LicenseProvider logs a license error
// and, after a 5 second grace period, swaps the whole editor for a hidden
// license gate — the editor then looks like it crashed. We are not a
// commercial production deployment, so keep tldraw on its development path:
// no console errors, no gate, and every feature enabled. The unlicensed
// watermark stays, as the tldraw license requires.
type LicenseManagerLike = {
  prototype: { getIsDevelopment?: () => boolean };
};

const LicenseManager = (
  tldraw as unknown as { LicenseManager?: LicenseManagerLike }
).LicenseManager;

/** Force tldraw's development license path. Idempotent. */
export function forceTldrawDevelopmentMode(): void {
  if (LicenseManager?.prototype) {
    LicenseManager.prototype.getIsDevelopment = () => true;
  }
}

/** Whether the development override is installed (used by tests). */
export function tldrawIsForcedDevelopment(): boolean {
  return LicenseManager?.prototype?.getIsDevelopment?.() === true;
}

forceTldrawDevelopmentMode();
