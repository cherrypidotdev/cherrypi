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
.flex.flex-col.gap-4
  Card
    CardHeader
      CardTitle Overall and per-tag limits
      CardDescription
        | A task consumes every cap that applies to it: the overall cap and
        | each of its tags. All must have room. Per-project and per-project×tag
        | limits live on the Projects tab.
    CardContent.flex.flex-col.gap-4
      .flex.items-center.justify-between.gap-3
        .flex.flex-col
          span.text-sm.font-medium Max concurrent tasks
          span.text-muted-foreground.text-xs Overall cap for this agent.
        Input.w-24(v-model.number="maxConcurrent" type="number" min="1")
      .bg-border.h-px
      .flex.flex-col.gap-2
        span.text-sm.font-medium Per-tag limits
        p.text-muted-foreground.text-xs
          | Keyed by tag name, these are global: one cap gates that name in
          | every project. Unlisted tags use the overall cap only.
        .flex.items-center.gap-2(
          v-for="(limit, index) in tagLimits"
          :key="limit.tagName"
        )
          span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
            class="py-0.5"
            :style="tagStyle(tagByName(limit.tagName)?.color)"
          )
            span.rounded-full.bg-current(class="size-1.5")
            | {{ limit.tagName }}
          Input.ml-auto.w-20(
            v-model.number="limit.maxConcurrent"
            type="number"
            min="1"
          )
          Button(
            variant="ghost"
            size="icon-sm"
            title="Remove"
            @click="tagLimits.splice(index, 1)"
          )
            XIcon
        .flex.items-center.gap-2
          Select(:model-value="addTagName" @update:model-value="addTagLimit")
            SelectTrigger.w-48
              SelectValue(placeholder="Add a tag limit…")
            SelectContent
              SelectGroup
                SelectItem(
                  v-for="name in availableTagNames"
                  :key="name"
                  :value="name"
                ) {{ name }}
          span.text-muted-foreground.text-xs(v-if="!availableTagNames.length")
            | No more tags

  p.text-destructive.text-sm(v-if="error") {{ error }}
  .flex.justify-end
    Button(@click="save" :disabled="saving")
      Spinner(v-if="saving" data-icon="inline-start")
      CheckIcon(v-else data-icon="inline-start")
      span Save limits

  Card(data-testid="agent-tag-declarations")
    CardHeader
      CardTitle Tags this agent handles
      CardDescription
        | A per-tag concurrency cap also declares the tag. Tasks whose tag is
        | marked required only run on agents that declare it, and a 0 cap still
        | declares it (the agent is interested but has no capacity right now).
    CardContent.flex.flex-col.gap-3
      p.text-muted-foreground.text-sm(
        v-if="!declaredGlobal.length && !declaredProjects.length"
      )
        | No per-tag caps yet, so this agent declares no tags.
      template(v-else)
        .flex.flex-col.gap-1(v-if="declaredGlobal.length")
          span.text-xs.font-medium Global
          .flex.flex-wrap.items-center(class="gap-1.5")
            span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
              v-for="name in declaredGlobal"
              :key="name"
              class="py-0.5"
              :style="tagStyle(tagByName(name)?.color)"
            )
              span.rounded-full.bg-current(class="size-1.5")
              span {{ name }}
        .flex.flex-col.gap-1(
          v-for="group in declaredProjects"
          :key="group.projectId"
        )
          span.text-xs.font-medium {{ group.projectName }}
          .flex.flex-wrap.items-center(class="gap-1.5")
            span.inline-flex.items-center.gap-1.rounded-full.border.px-2.text-xs(
              v-for="tag in group.tags"
              :key="tag.id"
              class="py-0.5"
              :style="tagStyle(tag.color)"
            )
              span.rounded-full.bg-current(class="size-1.5")
              span {{ tag.name }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { agency, agent } from "@/lib/sdk";
import { tagStyle } from "@/lib/task";
import type { AgentTaskLimits, Project, Tag } from "@cherrypi/sdk";
import { CheckIcon, XIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{ agentId: string }>();

const { all: allTags, active: tags, refresh: refreshTags } = useTags();
const saving = ref(false);
const error = ref("");

const maxConcurrent = ref(1);
const tagLimits = ref<{ tagName: string; maxConcurrent: number }[]>([]);
const projectTagLimits = ref<
  { projectId: string; tagId: string; maxConcurrent: number }[]
>([]);
const projects = ref<Project[]>([]);
const addTagName = ref("");

const availableTagNames = computed(() => {
  const used = new Set(
    tagLimits.value.map((limit) => limit.tagName.toLowerCase()),
  );
  const names: string[] = [];
  const seen = new Set<string>();
  for (const tag of tags.value) {
    const key = tag.name.toLowerCase();
    if (used.has(key) || seen.has(key)) continue;
    seen.add(key);
    names.push(tag.name);
  }
  return names.sort((a, b) => a.localeCompare(b));
});

function tagByName(name: string): Tag | undefined {
  const key = name.toLowerCase();
  return allTags.value.find((tag) => tag.name.toLowerCase() === key);
}

function tagById(id: string): Tag | undefined {
  return allTags.value.find((tag) => tag.id === id);
}

function projectName(id: string): string {
  return projects.value.find((project) => project.id === id)?.name ?? id;
}

const declaredGlobal = computed(() => {
  const names = new Set<string>();
  for (const limit of tagLimits.value) {
    const name = limit.tagName.trim().toLowerCase();
    if (name) names.add(name);
  }
  return [...names].sort((a, b) => a.localeCompare(b));
});

const declaredProjects = computed(() => {
  const groups = new Map<string, Tag[]>();
  for (const limit of projectTagLimits.value) {
    const tag = tagById(limit.tagId);
    if (!tag) continue;
    const list = groups.get(limit.projectId) ?? [];
    if (!list.some((item) => item.id === tag.id)) list.push(tag);
    groups.set(limit.projectId, list);
  }
  return [...groups.entries()]
    .map(([projectId, projectTags]) => ({
      projectId,
      projectName: projectName(projectId),
      tags: projectTags.sort((a, b) => a.name.localeCompare(b.name)),
    }))
    .sort((a, b) => a.projectName.localeCompare(b.projectName));
});

function addTagLimit(value: unknown) {
  const name = String(value);
  addTagName.value = "";
  if (
    !name ||
    tagLimits.value.some(
      (limit) => limit.tagName.toLowerCase() === name.toLowerCase(),
    )
  ) {
    return;
  }
  tagLimits.value.push({ tagName: name, maxConcurrent: 1 });
}

async function load() {
  try {
    const limits = await agent.getAgentTaskLimits(props.agentId);
    maxConcurrent.value = limits.maxConcurrent;
    tagLimits.value = limits.tagLimits.map((limit) => ({
      tagName: limit.tagName,
      maxConcurrent: limit.maxConcurrent,
    }));
    projectTagLimits.value = limits.projectTagLimits.map((limit) => ({
      projectId: limit.projectId,
      tagId: limit.tagId,
      maxConcurrent: limit.maxConcurrent,
    }));
    if (!projects.value.length) {
      try {
        projects.value = await agency.listAgentProjects(props.agentId);
      } catch {}
    }
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function save() {
  saving.value = true;
  error.value = "";
  try {
    const current: AgentTaskLimits = await agent.getAgentTaskLimits(
      props.agentId,
    );
    await withProgress(() =>
      agent.updateAgentTaskLimits(props.agentId, {
        maxConcurrent: maxConcurrent.value || 1,
        maxConcurrentPerProject: current.maxConcurrentPerProject,
        tagLimits: tagLimits.value,
        projectLimits: current.projectLimits,
        projectTagLimits: current.projectTagLimits,
      }),
    );
    toast.success("Concurrency limits saved");
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
    toast.error(error.value);
  } finally {
    saving.value = false;
  }
}

onMounted(async () => {
  await Promise.all([refreshTags(), load()]);
});
</script>
