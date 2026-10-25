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

import {
  defaultTaskColumnPreferences,
  TASK_COLUMN_KEYS,
  TASK_COLUMNS,
  type TaskColumn,
  type TaskColumnKey,
  type TaskColumnPreference,
} from "@/lib/taskColumns";
import { computed, ref } from "vue";

const STORAGE_KEY = "cherrypi.tasks.columns";

export type TaskColumnSetting = TaskColumn & { visible: boolean };

function isColumnKey(value: unknown): value is TaskColumnKey {
  return (
    typeof value === "string" && (TASK_COLUMN_KEYS as string[]).includes(value)
  );
}

function load(): TaskColumnPreference[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultTaskColumnPreferences();
    const saved = JSON.parse(raw) as unknown;
    if (!Array.isArray(saved)) return defaultTaskColumnPreferences();

    const prefs: TaskColumnPreference[] = [];
    const seen = new Set<TaskColumnKey>();
    for (const item of saved) {
      if (!item || typeof item !== "object") continue;
      const { key, visible } = item as { key?: unknown; visible?: unknown };
      if (!isColumnKey(key) || seen.has(key)) continue;
      seen.add(key);
      prefs.push({ key, visible: visible !== false });
    }
    for (const column of TASK_COLUMNS) {
      if (!seen.has(column.key)) prefs.push({ key: column.key, visible: true });
    }
    return prefs;
  } catch {
    return defaultTaskColumnPreferences();
  }
}

const preferences = ref<TaskColumnPreference[]>(load());

function persist() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences.value));
  } catch {}
}

const columns = computed<TaskColumnSetting[]>(() =>
  preferences.value.flatMap((pref) => {
    const meta = TASK_COLUMNS.find((column) => column.key === pref.key);
    return meta ? [{ ...meta, visible: pref.visible }] : [];
  }),
);

function reload() {
  preferences.value = load();
}

function move(from: number, to: number) {
  const next = [...preferences.value];
  if (from < 0 || from >= next.length || to < 0 || to >= next.length) return;
  if (from === to) return;
  const [item] = next.splice(from, 1);
  if (!item) return;
  next.splice(to, 0, item);
  preferences.value = next;
  persist();
}

function moveBy(key: TaskColumnKey, delta: number) {
  const from = preferences.value.findIndex((pref) => pref.key === key);
  if (from === -1) return;
  move(from, from + delta);
}

function setVisible(key: TaskColumnKey, visible: boolean) {
  preferences.value = preferences.value.map((pref) =>
    pref.key === key ? { ...pref, visible } : pref,
  );
  persist();
}

function reset() {
  preferences.value = defaultTaskColumnPreferences();
  persist();
}

export function useTaskColumns() {
  return {
    preferences,
    columns,
    reload,
    move,
    moveBy,
    setVisible,
    reset,
  };
}
