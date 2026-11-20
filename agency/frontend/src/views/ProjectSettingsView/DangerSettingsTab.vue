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
Card.border-destructive
  CardHeader
    CardTitle {{ $t("projectSettings.dangerZone") }}
    CardDescription
      | {{ $t("projectSettings.danger.description") }}
  CardContent
    .flex.items-center.justify-between.gap-4
      .flex.flex-col
        span.text-sm.font-medium {{ page.project?.archivedAt ? "Archived" : "Archive this project" }}
        span.text-muted-foreground.text-xs
          | {{ page.project?.archivedAt ? "Un-archive to show it in the sidebar and task lists again." : "The project keeps its data and can be restored at any time." }}
      Button(
        v-if="page.project && !page.project.archivedAt"
        variant="destructive"
        @click="archive"
      )
        ArchiveIcon(data-icon="inline-start")
        span {{ $t("tags.archive") }}
      Button(v-else-if="page.project" variant="outline" @click="unarchive")
        ArchiveRestoreIcon(data-icon="inline-start")
        span {{ $t("tasksTable.unarchive") }}
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
import { withProgress } from "@/composables/useProgress";
import type { ProjectPageState, ProjectSettingsState } from "@/lib/projectPage";
import { agency } from "@/lib/sdk";
import { ArchiveIcon, ArchiveRestoreIcon } from "@lucide/vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = defineProps<{
  page: ProjectPageState;
  settings: ProjectSettingsState;
}>();

const { t } = useI18n();

async function archive() {
  const project = props.page.project;
  if (!project) return;
  try {
    const { offlineAgents } = await withProgress(() =>
      agency.archiveProject(project.id),
    );
    props.page.setProject(await agency.getProject(project.id));
    if (offlineAgents.length) {
      toast.warning(
        `Project archived, but ${offlineAgents.length} offline agent(s) were skipped`,
      );
    } else {
      toast.success(t("admin.toasts.projectArchived"));
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function unarchive() {
  const project = props.page.project;
  if (!project) return;
  try {
    props.page.setProject(
      await withProgress(() => agency.unarchiveProject(project.id)),
    );
    toast.success(t("admin.toasts.projectUnarchived"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}
</script>

<i18n lang="yaml">
en:
  projectSettings:
    dangerZone: "Danger zone"
    danger:
      description: "Archiving hides the project and its tasks from the active lists. Un-archiving brings them back. Nothing is deleted."
ru:
  projectSettings:
    dangerZone: "Опасная зона"
    danger:
      description: "Архивация скрывает проект и его задачи из активных списков. Разархивация возвращает их. Ничего не удаляется."
</i18n>
