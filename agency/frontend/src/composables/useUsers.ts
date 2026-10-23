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

import { accountApi, type AccountActor } from "@/lib/account";
import { ref } from "vue";

// Module-scoped so the sidebar and the user profile page share one list.
const users = ref<AccountActor[]>([]);
const loading = ref(false);
const error = ref("");

async function refresh() {
  loading.value = true;
  try {
    users.value = (await accountApi.listDirectoryUsers()).users;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

let loaded = false;

async function loadUsers(force = false) {
  if (loaded && !force) return;
  loaded = true;
  await refresh();
}

export function useUsers() {
  return {
    users,
    loading,
    error,
    loadUsers,
    refresh,
  };
}
