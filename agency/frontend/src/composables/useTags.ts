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
import type { CreateTagInput, Tag, UpdateTagInput } from "@cherrypi/sdk";
import { computed, ref, toValue, watch, type MaybeRefOrGetter } from "vue";

export function useTags(projectId?: MaybeRefOrGetter<string | undefined>) {
  const all = ref<Tag[]>([]);
  const loading = ref(false);
  const error = ref("");

  const scoped = projectId !== undefined;
  const scope = () => toValue(projectId) ?? "";

  async function refresh() {
    if (scoped && scope() === "") {
      all.value = [];
      error.value = "";
      return;
    }
    loading.value = true;
    try {
      all.value = await agency.listTags({
        projectId: scope(),
        includeArchived: true,
      });
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  async function createTag(
    input: Omit<CreateTagInput, "projectId">,
  ): Promise<Tag> {
    const tag = await agency.createTag({ projectId: scope(), ...input });
    await refresh();
    return tag;
  }

  async function updateTag(id: string, input: UpdateTagInput): Promise<Tag> {
    const tag = await agency.updateTag(id, input);
    await refresh();
    return tag;
  }

  async function archiveTag(id: string): Promise<Tag> {
    const tag = await agency.archiveTag(id);
    await refresh();
    return tag;
  }

  async function restoreTag(id: string): Promise<Tag> {
    const tag = await agency.restoreTag(id);
    await refresh();
    return tag;
  }

  if (scoped) {
    watch(() => toValue(projectId), refresh, { immediate: true });
  }

  const active = computed(() => all.value.filter((tag) => !tag.archived));
  const archived = computed(() => all.value.filter((tag) => tag.archived));
  return {
    all,
    active,
    archived,
    loading,
    error,
    refresh,
    createTag,
    updateTag,
    archiveTag,
    restoreTag,
  };
}
