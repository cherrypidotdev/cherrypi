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

export interface PaneWidthRange {
  min: number;
  max: number;
}

export function clampPaneWidth(width: number, range: PaneWidthRange): number {
  if (!Number.isFinite(width)) return range.min;
  return Math.round(Math.min(range.max, Math.max(range.min, width)));
}

export function loadPaneWidth(
  key: string,
  range: PaneWidthRange,
): number | null {
  try {
    const raw = sessionStorage.getItem(key);
    if (!raw) return null;
    const width = Number(raw);
    return Number.isFinite(width) ? clampPaneWidth(width, range) : null;
  } catch {
    return null;
  }
}

export function savePaneWidth(key: string, width: number): void {
  try {
    sessionStorage.setItem(key, String(Math.round(width)));
  } catch {}
}
