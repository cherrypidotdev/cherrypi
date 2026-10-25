<!--
CherryPi: A multi-agent-multi-human self-hosted agency
Copyright (C) 2026 CherryPi authors <cherrypi.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<template lang="pug">
.flex.flex-col.gap-3.rounded-lg.border.p-4
  .flex.items-center.justify-between.gap-3
    .flex.flex-col
      span.text-sm.font-medium Project limits
      span.text-muted-foreground.text-xs
        | Caps that apply only inside this project. The overall caps live on the
        | Tasks tab.
    Spinner(v-if="loading" size="sm")
  .flex.flex-col.gap-2
    .flex.items-center.justify-between.gap-3
      .flex.flex-col
        span.text-sm.font-medium Max concurrent tasks in the project
        span.text-muted-foreground.text-xs Per-project cap.
      Input.w-24(v-model.number="maxConcurrent" type="number" min="1")
    .flex.flex-col.gap-2
      span.text-sm.font-medium Per-tag limits in the project
      p.text-muted-foreground.text-xs
        | Further narrows one tag inside this project.
      .flex.items-center.gap-2(
        v-for="(limit, index) in projectTagLimits"
        :key="limit.tagId"
      )
        span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
          class="py-0.5"
          :style="tagStyle(tagById(limit.tagId)?.color)"
        )
          span.rounded-full.bg-current(class="size-1.5")
          | {{ tagById(limit.tagId)?.name ?? limit.tagId.slice(0, 8) }}
        Input.ml-auto.w-20(
          v-model.number="limit.maxConcurrent"
          type="number"
          min="1"
        )
        Button(
          variant="ghost"
          size="icon-sm"
          title="Remove"
          @click="projectTagLimits.splice(index, 1)"
        )
          XIcon
      .flex.items-center.gap-2
        Select(:model-value="addTagId" @update:model-value="addTagLimit")
          SelectTrigger.w-48
            SelectValue(placeholder="Add a tag limit…")
          SelectContent
            SelectGroup
              SelectItem(
                v-for="tag in availableTags"
                :key="tag.id"
                :value="tag.id"
              ) {{ tag.name }}
        span.text-muted-foreground.text-xs(v-if="!availableTags.length")
          | No more tags

  p.text-destructive.text-xs(v-if="error") {{ error }}
  .flex.justify-end
    Button(size="sm" @click="save" :disabled="saving || loading")
      Spinner(v-if="saving" data-icon="inline-start")
      CheckIcon(v-else data-icon="inline-start")
      span Save
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { withProgress } from "@/composables/useProgress";
import { useTags } from "@/composables/useTags";
import { agent } from "@/lib/sdk";
import { tagStyle } from "@/lib/task";
import type { AgentTaskLimits, Tag } from "@cherrypi/sdk";
import { CheckIcon, XIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ agentId: string; projectId: string }>();

const { active: tags } = useTags(() => props.projectId);
const loading = ref(true);
const saving = ref(false);
const error = ref("");

const maxConcurrent = ref(1);
const projectTagLimits = ref<{ tagId: string; maxConcurrent: number }[]>([]);
const addTagId = ref("");

const availableTags = computed(() => {
  const used = new Set(projectTagLimits.value.map((limit) => limit.tagId));
  return tags.value.filter((tag) => !used.has(tag.id));
});

function tagById(id: string): Tag | undefined {
  return tags.value.find((tag) => tag.id === id);
}

function addTagLimit(value: unknown) {
  const id = String(value);
  addTagId.value = "";
  if (!id || projectTagLimits.value.some((limit) => limit.tagId === id)) return;
  projectTagLimits.value.push({ tagId: id, maxConcurrent: 1 });
}

async function load() {
  loading.value = true;
  try {
    const limits = await agent.getAgentTaskLimits(props.agentId);
    const project = limits.projectLimits.find(
      (limit) => limit.projectId === props.projectId,
    );
    maxConcurrent.value = project?.maxConcurrent ?? 1;
    projectTagLimits.value = limits.projectTagLimits
      .filter((limit) => limit.projectId === props.projectId)
      .map((limit) => ({
        tagId: limit.tagId,
        maxConcurrent: limit.maxConcurrent,
      }));
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  error.value = "";
  try {
    await withProgress(async () => {
      const current: AgentTaskLimits = await agent.getAgentTaskLimits(
        props.agentId,
      );
      await agent.updateAgentTaskLimits(props.agentId, {
        maxConcurrent: current.maxConcurrent,
        maxConcurrentPerProject: current.maxConcurrentPerProject,
        tagLimits: current.tagLimits,
        projectLimits: [
          ...current.projectLimits.filter(
            (limit) => limit.projectId !== props.projectId,
          ),
          {
            projectId: props.projectId,
            maxConcurrent: maxConcurrent.value || 1,
          },
        ],
        projectTagLimits: [
          ...current.projectTagLimits.filter(
            (limit) => limit.projectId !== props.projectId,
          ),
          ...projectTagLimits.value.map((limit) => ({
            projectId: props.projectId,
            tagId: limit.tagId,
            maxConcurrent: limit.maxConcurrent,
          })),
        ],
      });
    });
    toast.success("Project limits saved");
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
    toast.error(error.value);
  } finally {
    saving.value = false;
  }
}

watch(() => props.projectId, load, { immediate: true });
</script>
