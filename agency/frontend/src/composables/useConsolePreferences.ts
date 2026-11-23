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

import { ref } from "vue";

const STORAGE_KEY = "cherrypi.console.markdownIntervalMs";

export const DEFAULT_MARKDOWN_INTERVAL_MS = 400;

export const MIN_MARKDOWN_INTERVAL_MS = 100;

export const MAX_MARKDOWN_INTERVAL_MS = 5000;

export function clampMarkdownIntervalMs(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_MARKDOWN_INTERVAL_MS;
  return Math.min(
    Math.max(Math.round(value), MIN_MARKDOWN_INTERVAL_MS),
    MAX_MARKDOWN_INTERVAL_MS,
  );
}

function load(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === null) return DEFAULT_MARKDOWN_INTERVAL_MS;
    return clampMarkdownIntervalMs(Number(raw));
  } catch {
    return DEFAULT_MARKDOWN_INTERVAL_MS;
  }
}

const markdownIntervalMs = ref(load());

function setMarkdownIntervalMs(value: number): void {
  markdownIntervalMs.value = clampMarkdownIntervalMs(value);
  try {
    localStorage.setItem(STORAGE_KEY, String(markdownIntervalMs.value));
  } catch {}
}

function reloadConsolePreferences(): void {
  markdownIntervalMs.value = load();
}

function resetConsolePreferences(): void {
  markdownIntervalMs.value = DEFAULT_MARKDOWN_INTERVAL_MS;
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {}
}

export function useConsolePreferences() {
  return { markdownIntervalMs, setMarkdownIntervalMs };
}

export {
  markdownIntervalMs,
  reloadConsolePreferences,
  resetConsolePreferences,
  setMarkdownIntervalMs,
};
