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
.flex.flex-col.gap-6
  Card
    CardHeader
      CardTitle {{ $t("projectSettings.archive.title") }}
      CardDescription
        | {{ $t("projectSettings.archive.description") }}
    CardContent
      form.flex.flex-col.gap-3(@submit.prevent="saveArchivePolicy")
        .flex.flex-wrap.items-end.gap-3
          Field.w-72
            FieldLabel(for="project-archive-mode") {{ $t("projectSettings.archive.mode") }}
            Select(v-model="archiveMode" :disabled="!page.canManageProject")
              SelectTrigger#project-archive-mode.w-full
                SelectValue
              SelectContent
                SelectItem(value="inherit") {{ inheritLabel }}
                SelectItem(value="custom") {{ $t("projectSettings.archive.custom") }}
                SelectItem(value="never") {{ $t("projectSettings.archive.never") }}
          Field.w-36(v-if="archiveMode === 'custom'")
            FieldLabel(for="project-archive-days") {{ $t("projectSettings.archive.days") }}
            Input#project-archive-days(
              v-model.number="archiveCustomDays"
              type="number"
              min="1"
              :max="MAX_ARCHIVE_DAYS"
              :disabled="!page.canManageProject"
            )
          Button(
            type="submit"
            :disabled="!page.canManageProject || savingArchive || !archivePolicyValid"
          )
            Spinner(v-if="savingArchive" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ $t("common.save") }}
        p.text-muted-foreground.text-xs {{ archiveModeHint }}
        p.text-muted-foreground.text-xs(v-if="!page.canManageProject")
          | {{ $t("projectSettings.archive.ownersOnly") }}

  Card
    CardHeader
      CardTitle {{ $t("projectSettings.prompt.title") }}
      CardDescription
        | {{ $t("projectSettings.prompt.description") }}
    CardContent
      PromptEditor(
        :placeholder="$t('projectSettings.prompt.placeholder')"
        :scope="PromptScope.PROJECT"
        :project-id="page.project?.id"
      )

  Card
    CardHeader
      CardTitle {{ $t("domainLog.kind.tags") }}
      CardDescription {{ $t("projectSettings.tags.description") }}
    CardContent
      TagsPanel(:project-id="page.project?.id ?? ''")
</template>

<script setup lang="ts">
import PromptEditor from "@/components/PromptEditor.vue";
import TagsPanel from "@/components/settings/TagsPanel.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { useSettings } from "@/composables/useSettings";
import type { ProjectPageState, ProjectSettingsState } from "@/lib/projectPage";
import { agency } from "@/lib/sdk";
import { PromptScope, type Project } from "@cherrypi/sdk";
import { CheckIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = defineProps<{
  page: ProjectPageState;
  settings: ProjectSettingsState;
}>();

const { t } = useI18n();
const { refresh: refreshProjects } = useProjects();
const { settings: orgSettings, loadSettings } = useSettings();

const savingArchive = ref(false);

const MAX_ARCHIVE_DAYS = 3650;
const archiveMode = ref<"inherit" | "custom" | "never">("inherit");
const archiveCustomDays = ref(30);

const orgArchiveDays = computed(
  () => orgSettings.value?.taskAutoArchive?.days || 30,
);
const orgArchiveEnabled = computed(
  () => orgSettings.value?.taskAutoArchive?.enabled ?? true,
);
const inheritLabel = computed(() =>
  orgArchiveEnabled.value
    ? t("projectSettings.archive.inheritDays", { days: orgArchiveDays.value })
    : t("projectSettings.archive.inheritOff"),
);
const archiveCustomDaysValid = computed(() => {
  const days = Number(archiveCustomDays.value);
  return Number.isInteger(days) && days >= 1 && days <= MAX_ARCHIVE_DAYS;
});
const archivePolicyValid = computed(
  () => archiveMode.value !== "custom" || archiveCustomDaysValid.value,
);
const archiveModeHint = computed(() => {
  switch (archiveMode.value) {
    case "custom":
      return t("projectSettings.archive.customHint");
    case "never":
      return t("projectSettings.archive.neverHint");
    default:
      return t("projectSettings.archive.inheritHint");
  }
});

watch(archiveMode, (mode) => {
  if (mode === "custom" && !archiveCustomDaysValid.value) {
    archiveCustomDays.value = orgArchiveDays.value;
  }
});

function applyArchivePolicy(found: Project) {
  const days = found.archiveAfterDays;
  if (days === undefined) {
    archiveMode.value = "inherit";
  } else if (days === 0) {
    archiveMode.value = "never";
  } else {
    archiveMode.value = "custom";
    archiveCustomDays.value = days;
  }
}

const project = computed(() => props.page.project);
watch(
  project,
  (found) => {
    if (!found) return;
    applyArchivePolicy(found);
  },
  { immediate: true },
);

async function saveArchivePolicy() {
  if (!project.value) return;
  const days =
    archiveMode.value === "inherit"
      ? null
      : archiveMode.value === "never"
        ? 0
        : Number(archiveCustomDays.value);
  if (days !== null && days !== 0 && !archiveCustomDaysValid.value) return;
  savingArchive.value = true;
  try {
    const updated = await withProgress(() =>
      agency.setProjectArchiveAfterDays(project.value!.id, days),
    );
    applyArchivePolicy(updated);
    props.page.setProject(updated);
    await refreshProjects();
    toast.success(t("projectSettings.archive.saved"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingArchive.value = false;
  }
}

onMounted(() => void loadSettings());
</script>

<i18n lang="yaml">
en:
  projectSettings:
    archive:
      title: "Automatic archive"
      description: "Completed and cancelled tasks leave the active lists after a while, and their agent-side sessions are cleaned up. The organization sets the default policy; a project owner may override it here."
      mode: "When to archive"
      inheritDays: "Organization default ({days} days)"
      inheritOff: "Organization default (off)"
      inheritHint: "Follows the organization's policy, including when it is switched off."
      custom: "After a custom delay"
      customHint: "This project's own delay, regardless of the organization's policy."
      never: "Never"
      neverHint: "Closed tasks stay in the active lists even when the organization archives its own."
      days: "Days after close-out"
      ownersOnly: "Only project owners can change this policy."
      saved: "Archive policy saved"
    prompt:
      title: "Project prompt"
      description: "Injected into each new task's system prompt, after the agency and agent prompts. Edits affect future tasks only."
      placeholder: "Instructions for tasks in this project…"
    tags:
      description: "The tags task creators can pick inside this project."
ru:
  projectSettings:
    archive:
      title: "Автоматический архив"
      description: "Завершённые и отменённые задачи со временем покидают активные списки, а их сессии на агентах очищаются. По умолчанию действует правило агентства; владелец проекта может переопределить его здесь."
      mode: "Когда архивировать"
      inheritDays: "Как в агентстве ({days} дн.)"
      inheritOff: "Как в агентстве (выключено)"
      inheritHint: "Следует правилу агентства, в том числе когда оно выключено."
      custom: "Через свой срок"
      customHint: "Собственный срок проекта, независимо от правила агентства."
      never: "Никогда"
      neverHint: "Закрытые задачи остаются в активных списках, даже если агентство архивирует свои."
      days: "Дней после закрытия"
      ownersOnly: "Это правило могут менять только владельцы проекта."
      saved: "Правило архивации сохранено"
    prompt:
      title: "Промпт проекта"
      description: "Добавляется в системный промпт каждой новой задачи после промптов агентства и агента. Изменения влияют только на будущие задачи."
      placeholder: "Инструкции для задач этого проекта…"
    tags:
      description: "Теги, которые создатели задач могут выбрать в этом проекте."
</i18n>
