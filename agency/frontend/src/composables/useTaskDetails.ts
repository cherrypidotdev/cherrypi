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

// Whether the task details drawer is shown. It is one per-session view
// preference shared by every task, so it lives in sessionStorage and resets
// when the browser session ends.
const STORAGE_KEY = "cherrypi.task.details";

function load(): boolean {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (raw === "0") return false;
    if (raw === "1") return true;
  } catch {
    // Unavailable storage: fall through to the default.
  }
  return true;
}

// Module-scoped so every task page shares one value.
const open = ref(load());

function persist() {
  try {
    sessionStorage.setItem(STORAGE_KEY, open.value ? "1" : "0");
  } catch {
    // Best effort; blocked storage must never break the view.
  }
}

/** Show or hide the task details drawer and remember the choice. */
export function setTaskDetailsOpen(value: boolean) {
  open.value = value;
  persist();
}

/** Toggle the task details drawer. */
export function toggleTaskDetails() {
  setTaskDetailsOpen(!open.value);
}

/** Re-read the stored preference (used by tests after seeding storage). */
function reload() {
  open.value = load();
}

export function useTaskDetails() {
  return {
    open,
    toggle: toggleTaskDetails,
    setOpen: setTaskDetailsOpen,
    reload,
  };
}
