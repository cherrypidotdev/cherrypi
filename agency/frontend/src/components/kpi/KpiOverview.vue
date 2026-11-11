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
.flex.flex-col.gap-3(v-if="!hidden")
  p.text-destructive.text-xs(v-if="error") {{ error }}

  template(v-if="!editing")
    .flex.flex-wrap.items-center.justify-between.gap-2(
      v-if="canEdit || (inherited && widgets.length)"
    )
      span.text-muted-foreground.text-3xs(v-if="inherited && widgets.length")
        | {{ $t("kpiOverview.followingDefault") }}
      Button(
        v-if="canEdit"
        type="button"
        variant="ghost"
        size="sm"
        @click="startEdit"
        data-kpi-enable-edit
      )
        PencilIcon(data-icon="inline-start")
        span {{ $t("kpiOverview.enableEdit") }}

    KpiGrid(
      v-if="widgets.length"
      :widgets="widgets"
      :page="pageContext"
      :range="range"
      :can-edit="canEdit"
      :agent-options="agentOptions"
      :project-options="projectOptions"
    )
    .rounded-xl.border.border-dashed.p-4(v-else-if="canEdit && !loading")
      .flex.flex-wrap.items-center.justify-between.gap-3
        .flex.min-w-0.flex-col
          span.text-sm.font-medium {{ $t("kpiOverview.empty") }}
          span.text-muted-foreground.text-xs
            | {{ $t("kpiOverview.emptyHint") }}{{ inherited ? $t("kpiOverview.emptyInherited") : "" }}
        Button(
          type="button"
          variant="outline"
          size="sm"
          @click="startEdit"
          data-kpi-enable-edit
        )
          PlusIcon(data-icon="inline-start")
          span {{ $t("kpiOverview.addWidgets") }}

  template(v-else)
    .flex.flex-wrap.items-center.justify-between.gap-2
      .flex.items-center.gap-2
        span.text-primary.text-3xs.rounded-full.px-2.font-semibold(
          class="bg-primary/10 py-0.5"
        )
          | {{ $t("kpiOverview.editMode") }}
        span.text-muted-foreground.text-3xs {{ $t("kpiOverview.unsavedBanner") }}
      Button(type="button" variant="ghost" size="sm" @click="cancelEdit")
        | {{ $t("common.done") }}
    KpiGrid(
      editing
      :widgets="draft"
      :page="pageContext"
      :range="range"
      :saving="saving"
      :can-edit="canEdit"
      :inherited="inherited"
      :agent-options="agentOptions"
      :project-options="projectOptions"
      @update:widgets="draft = $event"
      @save="saveDraft"
      @cancel="cancelEdit"
      @reset="resetLayout"
    )

  Dialog(
    :open="leavePrompt"
    @update:open="(value) => !value && answerLeave(false)"
  )
    DialogContent(class="sm:max-w-md")
      DialogHeader
        DialogTitle {{ $t("kpiOverview.discardTitle") }}
        DialogDescription
          | {{ $t("kpiOverview.unsavedHint") }}
      DialogFooter
        Button(type="button" variant="outline" @click="answerLeave(false)")
          | {{ $t("kpiOverview.keepEditing") }}
        Button(type="button" variant="destructive" @click="answerLeave(true)")
          | {{ $t("kpiOverview.discard") }}
</template>

<script setup lang="ts">
import KpiGrid from "@/components/kpi/KpiGrid.vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useAgents } from "@/composables/useAgents";
import { useKpiLayout } from "@/composables/useKpiLayout";
import { useProjects } from "@/composables/useProjects";
import type { KpiPageContext, KpiWidgetState } from "@/lib/kpiWidgets";
import { sortProjectsByName } from "@/lib/project";
import type { UsageRange } from "@/lib/usage";
import type { KpiPage } from "@cherrypi/sdk";
import { PencilIcon, PlusIcon } from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{
  page: KpiPage | undefined;
  pageContext: KpiPageContext;

  range?: UsageRange;

  hidden?: boolean;
}>();

const router = useRouter();
const { agents, loadAgents } = useAgents();
const { active, loadProjects } = useProjects();

const range = computed<UsageRange>(() => props.range ?? "30");
const { widgets, inherited, canEdit, loading, saving, error, save, reset } =
  useKpiLayout(() => props.page);

const editing = ref(false);
const draft = ref<KpiWidgetState[]>([]);
const leavePrompt = ref(false);
let answerLeaveGuard: ((leave: boolean) => void) | undefined;

const agentOptions = computed(() =>
  agents.value.map((agent) => ({
    id: agent.id,
    name: agent.displayName || agent.username,
  })),
);
const projectOptions = computed(() =>
  sortProjectsByName(active.value).map((project) => ({
    id: project.id,
    name: project.name,
  })),
);

const dirty = computed(
  () =>
    editing.value &&
    JSON.stringify(draft.value) !== JSON.stringify(widgets.value),
);

watch(widgets, (value) => {
  if (!editing.value) draft.value = value.map((widget) => ({ ...widget }));
});

const pageKey = computed(() =>
  props.page
    ? props.page.kind === "agency"
      ? "agency"
      : `${props.page.kind}:${props.page.id}`
    : "",
);
watch(pageKey, () => {
  editing.value = false;
  draft.value = [];
});

function startEdit() {
  draft.value = widgets.value.map((widget) => ({ ...widget }));
  editing.value = true;
}

function cancelEdit() {
  editing.value = false;
  draft.value = [];
}

async function saveDraft() {
  try {
    await save(draft.value);
    editing.value = false;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function resetLayout() {
  try {
    await reset();
    editing.value = false;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function answerLeave(leave: boolean) {
  leavePrompt.value = false;
  if (leave) cancelEdit();
  const resolve = answerLeaveGuard;
  answerLeaveGuard = undefined;
  resolve?.(leave);
}

let unregisterGuard: (() => void) | undefined;

function guardLeave(): boolean | Promise<boolean> {
  if (!dirty.value) return true;
  leavePrompt.value = true;
  return new Promise<boolean>((resolve) => {
    answerLeaveGuard = resolve;
  });
}

function onBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return;
  event.preventDefault();
  event.returnValue = "";
}

onMounted(() => {
  void loadAgents();
  void loadProjects();
  window.addEventListener("beforeunload", onBeforeUnload);
  unregisterGuard = router.beforeEach(guardLeave);
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", onBeforeUnload);
  unregisterGuard?.();
  unregisterGuard = undefined;

  answerLeaveGuard?.(true);
  answerLeaveGuard = undefined;
});
</script>

<i18n lang="yaml">
en:
  kpiOverview:
    followingDefault: "Following the organization default."
    editMode: "Edit mode"
    discardTitle: "Discard unsaved layout changes?"
    unsavedHint: "This page's widgets were changed but not saved. Leaving now keeps the layout the agency has stored."
    keepEditing: "Keep editing"
    discard: "Discard changes"
    enableEdit: "Enable edit mode"
    empty: "No KPIs on this page yet"
    emptyHint: "Pick usage and concurrency widgets to watch here."
    emptyInherited: " This page follows the organization default, which is empty."
    addWidgets: "Add widgets"
    unsavedBanner: "Unsaved changes stay on this page until you save."
  common:
    done: "Done"
ru:
  kpiOverview:
    followingDefault: "Используется значение организации по умолчанию."
    editMode: "Режим редактирования"
    discardTitle: "Отменить несохранённые изменения макета?"
    unsavedHint: "Виджеты страницы изменены, но не сохранены. При уходе останется макет, сохранённый в агентстве."
    keepEditing: "Продолжить редактирование"
    discard: "Отменить изменения"
    enableEdit: "Включить режим редактирования"
    empty: "На этой странице пока нет KPI"
    emptyHint: "Выберите виджеты использования и параллельности, чтобы следить за ними здесь."
    emptyInherited: " Эта страница следует значению организации по умолчанию, а оно пустое."
    addWidgets: "Добавить виджеты"
    unsavedBanner: "Несохранённые изменения остаются на этой странице до сохранения."
  common:
    done: "Готово"
</i18n>
