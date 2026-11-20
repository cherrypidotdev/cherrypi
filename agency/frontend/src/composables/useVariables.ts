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
import type {
  CreateVariableInput,
  EffectiveVariable,
  Variable,
  VariableTargetInput,
} from "@cherrypi/sdk";
import { ref, toValue, watch, type MaybeRefOrGetter } from "vue";

export function useVariables(
  target?: MaybeRefOrGetter<VariableTargetInput | undefined>,
) {
  const items = ref<Variable[]>([]);
  const loading = ref(false);
  const error = ref("");
  const secretsEnabled = ref(true);

  const current = () => (target ? toValue(target) : undefined);

  async function refresh() {
    const scope = current();
    if (!scope) {
      items.value = [];
      return;
    }
    loading.value = true;
    try {
      items.value = await agency.listVariables(scope);
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  async function loadCapabilities() {
    try {
      const capabilities = await agency.getVariableCapabilities();
      secretsEnabled.value = capabilities.secretsEnabled;
    } catch {}
  }

  async function createVariable(
    input: Omit<CreateVariableInput, "target">,
  ): Promise<Variable> {
    const scope = current();
    if (!scope) throw new Error("no variable scope");
    const created = await agency.createVariable({ ...input, target: scope });
    await refresh();
    return created;
  }

  async function updateVariable(
    name: string,
    patch: { note?: string; value?: string },
  ): Promise<Variable> {
    const scope = current();
    if (!scope) throw new Error("no variable scope");
    const updated = await agency.updateVariable(scope, name, patch);
    await refresh();
    return updated;
  }

  async function promoteVariable(name: string): Promise<Variable> {
    const scope = current();
    if (!scope) throw new Error("no variable scope");
    const promoted = await agency.promoteVariable(scope, name);
    await refresh();
    return promoted;
  }

  async function deleteVariable(name: string): Promise<void> {
    const scope = current();
    if (!scope) throw new Error("no variable scope");
    await agency.deleteVariable(scope, name);
    await refresh();
  }

  if (target) {
    watch(
      () => {
        const scope = current();
        return scope ? JSON.stringify(scope) : "";
      },
      () => void refresh(),
      { immediate: true },
    );
  }

  return {
    items,
    loading,
    error,
    secretsEnabled,
    refresh,
    loadCapabilities,
    createVariable,
    updateVariable,
    promoteVariable,
    deleteVariable,
  };
}

export function useEffectiveVariables(
  agentId: MaybeRefOrGetter<string | undefined>,
  projectId?: MaybeRefOrGetter<string | undefined>,
) {
  const items = ref<EffectiveVariable[]>([]);
  const loading = ref(false);
  const error = ref("");

  async function refresh() {
    const id = toValue(agentId);
    if (!id) {
      items.value = [];
      return;
    }
    loading.value = true;
    try {
      items.value = await agency.listEffectiveVariables(
        id,
        projectId ? toValue(projectId) : undefined,
      );
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  watch(
    () => [toValue(agentId), projectId ? toValue(projectId) : ""].join("|"),
    () => void refresh(),
    { immediate: true },
  );

  return { items, loading, error, refresh };
}
