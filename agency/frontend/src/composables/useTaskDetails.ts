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

const STORAGE_KEY = "cherrypi.task.details";

function load(): boolean {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (raw === "0") return false;
    if (raw === "1") return true;
  } catch {}
  return true;
}

const open = ref(load());

function persist() {
  try {
    sessionStorage.setItem(STORAGE_KEY, open.value ? "1" : "0");
  } catch {}
}

export function setTaskDetailsOpen(value: boolean) {
  open.value = value;
  persist();
}

export function toggleTaskDetails() {
  setTaskDetailsOpen(!open.value);
}

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
