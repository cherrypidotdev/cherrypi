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

import { agency } from "@/lib/sdk";
import { PromptScope, type PromptTarget } from "@cherrypi/sdk";
import { ref, watch } from "vue";

export function usePrompt(
  scope: PromptScope,
  getTarget: () => PromptTarget = () => ({}),
) {
  const content = ref("");
  const version = ref<bigint>(0n);
  const loading = ref(true);
  const saving = ref(false);
  const error = ref("");
  const canEdit = ref(true);
  const canPropose = ref(false);
  const canApprove = ref(false);

  let readID = 0;

  async function load() {
    const id = ++readID;
    loading.value = true;
    try {
      const result = await agency.getPromptWithAccess(scope, getTarget());
      if (id !== readID) return;
      content.value = result.prompt.content;
      version.value = result.prompt.version;
      canEdit.value = result.canEdit;
      canPropose.value = result.canPropose;
      canApprove.value = result.canApprove;
      error.value = "";
    } catch (err) {
      if (id !== readID) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (id === readID) loading.value = false;
    }
  }

  watch(
    () => {
      const target = getTarget();
      return `${target.agentId ?? ""}\u0000${target.teamId ?? ""}\u0000${target.projectId ?? ""}`;
    },
    () => void load(),
    { immediate: true },
  );

  async function save(note = ""): Promise<boolean> {
    saving.value = true;
    try {
      const prompt = await agency.setPrompt(
        scope,
        content.value,
        getTarget(),
        note,
      );
      version.value = prompt.version;
      error.value = "";
      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
      return false;
    } finally {
      saving.value = false;
    }
  }

  return {
    content,
    version,
    loading,
    saving,
    error,
    canEdit,
    canPropose,
    canApprove,
    load,
    save,
  };
}
