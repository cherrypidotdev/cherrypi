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

export const EDITOR_MIN_HEIGHT = 96;

export const COMPACT_EDITOR_HEIGHT = 112;

export const EDITOR_MAX_HEIGHT = 384;

const VIEWPORT_MARGIN = 160;

export const NEW_TASK_PROMPT_HEIGHT_KEY = "cherrypi:new-task-prompt-height";

export const QUESTIONNAIRE_NOTE_HEIGHT_KEY =
  "cherrypi:questionnaire-note-height";

export const QUESTIONNAIRE_FINAL_NOTE_HEIGHT_KEY =
  "cherrypi:questionnaire-final-note-height";

export function clampEditorHeightInRange(
  height: number,
  min: number,
  max: number,
): number {
  const viewportMax =
    typeof window === "undefined" || !Number.isFinite(window.innerHeight)
      ? max
      : Math.max(min, window.innerHeight - VIEWPORT_MARGIN);
  const upper = Math.max(min, Math.min(max, viewportMax));
  if (!Number.isFinite(height)) return min;
  return Math.round(Math.min(upper, Math.max(min, height)));
}

export function clampEditorHeight(height: number): number {
  return clampEditorHeightInRange(height, EDITOR_MIN_HEIGHT, EDITOR_MAX_HEIGHT);
}

export function loadEditorHeight(key: string): number | null {
  try {
    const raw = sessionStorage.getItem(key);
    if (!raw) return null;
    const height = Number(raw);
    return Number.isFinite(height) ? clampEditorHeight(height) : null;
  } catch {
    return null;
  }
}

export function saveEditorHeight(key: string, height: number): void {
  try {
    sessionStorage.setItem(key, String(Math.round(height)));
  } catch {}
}
