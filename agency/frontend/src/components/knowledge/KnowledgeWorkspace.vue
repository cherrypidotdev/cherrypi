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
  .flex.flex-wrap.items-start.justify-between.gap-3
    .flex.flex-col
      h2.text-lg.font-semibold {{ heading || defaultHeading }}
      p.text-muted-foreground.max-w-3xl.text-sm {{ description || defaultDescription }}
    .flex.items-center.gap-2
      Button(variant="outline" size="sm" @click="load()" :disabled="loading")
        RefreshCwIcon(data-icon="inline-start")
        span {{ $t("knowledge.refresh") }}
      Button(v-if="canCreate" size="sm" @click="openCreate")
        PlusIcon(data-icon="inline-start")
        span {{ $t("knowledge.newBit") }}

  .grid.items-start.gap-4(class="lg:grid-cols-[20rem_minmax(0,1fr)]")
    Card
      CardHeader.pb-3
        CardTitle.text-sm {{ $t("knowledge.bits") }}
      CardContent.flex.flex-col.gap-3
        .relative
          SearchIcon.text-muted-foreground.absolute.top-2.left-2.size-4
          Input.pl-8(
            v-model="search"
            :placeholder="$t('knowledge.searchPlaceholder')"
            @keyup.enter="load()"
          )
        select.border-input.w-full.rounded-lg.border.bg-transparent.px-2.py-1.text-sm(
          v-if="scope === 'all'"
          v-model="projectFilter"
          @change="load()"
        )
          option(value) {{ $t("knowledge.allProjects") }}
          option(
            v-for="project in projects"
            :key="project.id"
            :value="project.id"
          )
            | {{ project.name }}
        .flex.flex-wrap.items-center.gap-1
          Button(
            v-for="option in statuses"
            :key="option.value"
            size="sm"
            :variant="status === option.value ? 'default' : 'outline'"
            @click="setStatus(option.value)"
          ) {{ option.label }}
        .flex.flex-wrap.items-center.gap-1(v-if="tags.length")
          button.rounded-full.px-2.text-xs(
            v-for="item in tags"
            :key="item.tag"
            type="button"
            class="py-0.5"
            :class="tag === item.tag ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:bg-muted/70'"
            @click="setTag(item.tag)"
          ) {{ tagLabel(item.tag) }} {{ item.count }}
        p.text-destructive.text-xs(v-if="error") {{ error }}
        .flex.items-center.justify-between.gap-2
          span.text-muted-foreground.text-xs {{ bits.length }}
          Spinner.size-3(v-if="loading")
        .flex.flex-col.gap-1
          button.w-full.rounded-lg.border.border-transparent.px-2.py-2.text-left.transition-colors(
            v-for="bit in bits"
            :key="bit.id"
            type="button"
            :class="selected?.id === bit.id ? 'bg-muted' : 'hover:bg-muted/60'"
            @click="select(bit)"
          )
            .flex.items-center.gap-2
              span.min-w-0.flex-1.truncate.text-sm.font-medium {{ bit.title || bit.content }}
              Badge(class="text-[10px]" :variant="statusVariant(bit.status)") {{ statusLabel(bit.status) }}
            .flex.flex-wrap.items-center.gap-1.pl-0(class="mt-0.5")
              span.text-muted-foreground.font-mono(class="text-[10px]") {{ shortId(bit.id) }}
              span.text-muted-foreground(v-if="bit.projectSlug" class="text-[10px]") · {{ bit.projectSlug }}
              span.text-muted-foreground(v-if="bit.tags.length" class="text-[10px]") · {{ bit.tags.map((tag) => "#" + tag).join(" ") }}
          p.text-muted-foreground.text-sm(v-if="!bits.length && !loading") {{ $t("knowledge.empty") }}

    div(v-if="selected")
      Card
        CardHeader.flex.flex-col.gap-2
          .flex.flex-wrap.items-start.justify-between.gap-3
            .flex.min-w-0.flex-col.gap-1
              .flex.flex-wrap.items-center.gap-2
                h3.min-w-0.text-xl.font-semibold.break-words {{ selected.title || selected.content }}
                Badge(:variant="statusVariant(selected.status)") {{ statusLabel(selected.status) }}
                Badge(v-if="selected.projectSlug" variant="outline") {{ selected.projectSlug }}
              p.text-muted-foreground.text-xs
                | {{ $t("knowledge.by") }} {{ selected.createdBy }}
                template(v-if="selected.reviewedBy")
                  | · {{ $t("knowledge.reviewedBy") }} {{ selected.reviewedBy }}
                | · r{{ selected.revision }}
            .flex.flex-wrap.items-center.gap-1
              Button(
                v-if="selected.canEdit"
                variant="outline"
                size="sm"
                @click="openEdit"
              )
                PencilIcon(data-icon="inline-start")
                span {{ $t("knowledge.edit") }}
              Button(
                v-if="selected.canReview"
                variant="outline"
                size="sm"
                @click="openReview('approve')"
              )
                ShieldCheckIcon(data-icon="inline-start")
                span {{ $t("knowledge.approve") }}
              Button(
                v-if="selected.canReview"
                variant="outline"
                size="sm"
                @click="openReview('reject')"
              )
                XIcon(data-icon="inline-start")
                span {{ $t("knowledge.reject") }}
              Button(
                v-if="selected.canDelete"
                variant="ghost"
                size="sm"
                @click="remove"
              )
                Trash2Icon(data-icon="inline-start")
          .flex.flex-wrap.items-center.gap-1(v-if="selected.tags.length")
            Badge(v-for="tag in selected.tags" :key="tag" variant="secondary") {{ tagLabel(tag) }}
        CardContent.flex.flex-col.gap-4
          p.text-muted-foreground.text-sm(v-if="selected.reviewNote")
            span.font-medium {{ $t("knowledge.reviewNote") }}:
            | {{ selected.reviewNote }}
          RouterLink.text-muted-foreground.text-xs.underline.underline-offset-2(
            v-if="selected.originTaskId"
            class-hover="text-foreground"
            :to="{ name: 'task', params: { id: selected.originTaskId } }"
          ) {{ $t("knowledge.originTask") }} {{ selected.originTaskId }}
          MarkdownText.prose.prose-sm.max-w-none(:text="selected.content")

          section.flex.flex-col.gap-2(v-if="selected.sources.length")
            h4.text-sm.font-semibold {{ $t("knowledge.sources") }}
            .flex.flex-col.gap-1
              template(v-for="source in selected.sources" :key="source.id")
                RouterLink.text-primary.text-sm.underline.underline-offset-2(
                  v-if="source.href.startsWith('/')"
                  :to="source.href"
                ) {{ source.label || source.kind }}
                a.text-primary.text-sm.underline.underline-offset-2(
                  v-else-if="source.href"
                  rel="noopener"
                  :href="source.href"
                  target="_blank"
                ) {{ source.label || source.kind }}
                span.text-sm(v-else) {{ source.label || source.kind }}

          section.flex.flex-col.gap-2(
            v-if="selected.links.length || selected.backlinks.length"
          )
            h4.text-sm.font-semibold {{ $t("knowledge.related") }}
            .flex.flex-wrap.items-center.gap-1
              button.bg-muted.rounded-full.px-2.text-xs(
                v-for="link in selected.links"
                :key="'out-' + link.knowledgeId + link.kind"
                type="button"
                class="hover:bg-muted/70 py-0.5"
                @click="selectById(link.knowledgeId)"
              )
                | {{ link.kind === "supersedes" ? $t("knowledge.supersedes") : $t("knowledge.related") }}: {{ link.title || shortId(link.knowledgeId) }}
              button.bg-muted.rounded-full.px-2.text-xs(
                v-for="link in selected.backlinks"
                :key="'in-' + link.knowledgeId + link.kind"
                type="button"
                class="hover:bg-muted/70 py-0.5"
                @click="selectById(link.knowledgeId)"
              )
                | {{ link.kind === "supersedes" ? $t("knowledge.supersededBy") : $t("knowledge.linkedFrom") }}: {{ link.title || shortId(link.knowledgeId) }}

          details(v-if="selected.revisions.length")
            summary.cursor-pointer.text-sm.font-semibold {{ $t("knowledge.history") }}
            .mt-2.flex.flex-col.gap-2
              .flex.flex-col.gap-1.rounded-lg.border.p-2(
                v-for="revision in selected.revisions"
                :key="String(revision.revision)"
              )
                .flex.items-center.justify-between.gap-2.text-xs
                  span.font-medium r{{ revision.revision }} · {{ revision.status }}
                  span.text-muted-foreground {{ revision.actorRef }}
                p.text-muted-foreground.text-xs {{ revision.summary }}
                MarkdownText.prose.prose-sm.max-w-none(
                  :text="revision.content"
                )
    Card(v-else)
      CardContent.py-10
        p.text-muted-foreground.text-center.text-sm {{ $t("knowledge.selectHint") }}

  KnowledgeEditor(
    v-model:open="showEditor"
    :project-id="editorProjectId"
    :bit="editing"
    @saved="onSaved"
  )

  Dialog(:open="Boolean(reviewing)" @update:open="closeReview")
    DialogContent(class="sm:max-w-md")
      DialogHeader
        DialogTitle {{ reviewing === "reject" ? $t("knowledge.rejectTitle") : $t("knowledge.approveTitle") }}
        DialogDescription {{ reviewing === "reject" ? $t("knowledge.rejectHint") : $t("knowledge.approveHint") }}
      Field
        FieldLabel(for="knowledge-review-note") {{ $t("knowledge.reviewNote") }}
        Textarea#knowledge-review-note(v-model="reviewNote" rows="3")
      p.text-destructive.text-sm(v-if="reviewError") {{ reviewError }}
      DialogFooter
        DialogClose
          Button(type="button" variant="ghost") {{ $t("knowledge.cancel") }}
        Button(
          type="button"
          :variant="reviewing === 'reject' ? 'destructive' : 'default'"
          @click="submitReview"
          :disabled="reviewSaving"
        )
          Spinner(v-if="reviewSaving" data-icon="inline-start")
          span {{ reviewing === "reject" ? $t("knowledge.reject") : $t("knowledge.approve") }}
</template>

<script setup lang="ts">
import KnowledgeEditor from "@/components/knowledge/KnowledgeEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { useRealtime } from "@/composables/useRealtime";
import { agency } from "@/lib/sdk";
import type { KnowledgeBit, KnowledgeTagCount, Project } from "@cherrypi/sdk";
import {
  PencilIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
  ShieldCheckIcon,
  Trash2Icon,
  XIcon,
} from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute, useRouter } from "vue-router";

const props = defineProps<{
  scope: "org" | "project" | "all";
  projectId?: string;

  taskId?: string;
  heading?: string;
  description?: string;
  initialStatus?: string;
}>();

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const bits = ref<KnowledgeBit[]>([]);
const selected = ref<KnowledgeBit>();
const tags = ref<KnowledgeTagCount[]>([]);
const projects = ref<Project[]>([]);
const projectFilter = ref("");
const search = ref("");
const tag = ref("");
const status = ref(props.initialStatus ?? "approved");
const loading = ref(false);
const error = ref("");
const showEditor = ref(false);
const editing = ref<KnowledgeBit>();
const reviewing = ref<"" | "approve" | "reject">("");
const reviewNote = ref("");
const reviewError = ref("");
const reviewSaving = ref(false);

const statuses = computed(() => [
  { value: "approved", label: t("knowledge.approved") },
  { value: "pending", label: t("knowledge.pending") },
  { value: "rejected", label: t("knowledge.rejected") },
  { value: "superseded", label: t("knowledge.superseded") },
]);

const defaultHeading = computed(() =>
  props.scope === "project"
    ? t("knowledge.projectHeading")
    : t("knowledge.orgHeading"),
);
const defaultDescription = computed(() =>
  props.scope === "project"
    ? t("knowledge.projectDescription")
    : t("knowledge.orgDescription"),
);
const canCreate = computed(
  () => props.scope !== "project" || Boolean(props.projectId),
);
const editorProjectId = computed(() => {
  if (props.scope === "project") return props.projectId;
  return projectFilter.value || "";
});

function statusLabel(value: string): string {
  switch (value) {
    case "pending":
      return t("knowledge.pending");
    case "rejected":
      return t("knowledge.rejected");
    case "superseded":
      return t("knowledge.superseded");
    default:
      return t("knowledge.approved");
  }
}

function statusVariant(
  value: string,
): "default" | "secondary" | "outline" | "destructive" {
  switch (value) {
    case "pending":
      return "secondary";
    case "rejected":
      return "destructive";
    case "superseded":
      return "outline";
    default:
      return "default";
  }
}

function shortId(id: string): string {
  return id.slice(0, 8);
}

function tagLabel(tag: string): string {
  return `#${tag}`;
}

function setStatus(value: string) {
  status.value = value;
  void load();
}

function setTag(value: string) {
  tag.value = tag.value === value ? "" : value;
  void load();
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const projectId =
      props.scope === "project" ? props.projectId : projectFilter.value;
    bits.value = await agency.listKnowledge({
      scope: props.scope,
      projectId: projectId || undefined,
      tag: tag.value,
      status: status.value,
      query: search.value.trim(),
      taskId: props.taskId,
      limit: 100,
    });
    tags.value = await agency.listKnowledgeTags({
      scope: props.scope,
      projectId: projectId || undefined,
    });
    const wanted =
      typeof route.query.bit === "string" ? route.query.bit : undefined;
    const current = wanted || selected.value?.id || bits.value[0]?.id;
    if (current) {
      await selectById(current, current !== route.query.bit);
    } else {
      selected.value = undefined;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function selectById(id: string, updateRoute = true) {
  try {
    selected.value = await agency.getKnowledge(id);
    if (updateRoute) {
      void router.replace({ query: { ...route.query, bit: id } });
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function select(bit: KnowledgeBit) {
  await selectById(bit.id);
}

function openCreate() {
  editing.value = undefined;
  showEditor.value = true;
}

function openEdit() {
  editing.value = selected.value;
  showEditor.value = true;
}

async function onSaved(bit: KnowledgeBit) {
  selected.value = bit;
  void router.replace({ query: { ...route.query, bit: bit.id } });
  await load();
}

function openReview(kind: "approve" | "reject") {
  reviewing.value = kind;
  reviewNote.value = "";
  reviewError.value = "";
}

function closeReview() {
  reviewing.value = "";
}

async function submitReview() {
  if (!selected.value || !reviewing.value) return;
  reviewSaving.value = true;
  reviewError.value = "";
  try {
    selected.value =
      reviewing.value === "approve"
        ? await agency.approveKnowledge(selected.value.id, reviewNote.value)
        : await agency.rejectKnowledge(selected.value.id, reviewNote.value);
    closeReview();
    await load();
  } catch (err) {
    reviewError.value = err instanceof Error ? err.message : String(err);
  } finally {
    reviewSaving.value = false;
  }
}

async function remove() {
  if (!selected.value) return;
  if (!window.confirm(t("knowledge.deleteConfirm"))) return;
  try {
    await agency.deleteKnowledge(selected.value.id);
    selected.value = undefined;
    const next = Object.fromEntries(
      Object.entries(route.query).filter(([key]) => key !== "bit"),
    );
    void router.replace({ query: next });
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

let timer: ReturnType<typeof setTimeout> | undefined;
watch(search, () => {
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => void load(), 250);
});

watch(
  () => [props.scope, props.projectId],
  () => {
    selected.value = undefined;
    void load();
  },
);

watch(
  () => route.query.bit,
  (value) => {
    if (typeof value === "string" && value && value !== selected.value?.id) {
      void selectById(value, false);
    }
  },
);

onMounted(async () => {
  if (props.scope === "all") {
    try {
      projects.value = await agency.listProjects();
    } catch {
      projects.value = [];
    }
  }
  await load();
});

useRealtime(["knowledge"], () => {
  void load();
});
</script>

<i18n lang="yaml">
en:
  knowledge:
    newBit: "New knowledge"
    bits: "Knowledge"
    searchPlaceholder: "Search knowledge…"
    refresh: "Refresh"
    empty: "No knowledge here yet."
    allProjects: "All projects"
    approved: "Approved"
    pending: "Pending"
    rejected: "Rejected"
    superseded: "Superseded"
    by: "by"
    reviewedBy: "reviewed by"
    reviewNote: "Review note"
    originTask: "Recorded in task"
    sources: "Sources"
    related: "Related knowledge"
    supersedes: "supersedes"
    supersededBy: "superseded by"
    linkedFrom: "linked from"
    history: "Revision history"
    edit: "Edit"
    approve: "Approve"
    reject: "Reject"
    approveTitle: "Approve knowledge"
    approveHint: "Approving makes this bit the shared truth and retires the bits it supersedes."
    rejectTitle: "Reject knowledge"
    rejectHint: "A rejected bit stays readable to reviewers; its author can edit it back into review."
    cancel: "Cancel"
    deleteConfirm: "Delete this knowledge bit? Its history stays in the agency."
    selectHint: "Pick a bit from the list, or record a new one. Approved bits are the shared truth; pending ones wait for a human."
    projectHeading: "Project knowledge"
    orgHeading: "Knowledge"
    projectDescription: "Recorded facts shared by this project's humans and agents. Bits are attributed to their author and their real sources."
    orgDescription: "Recorded facts with hard links to the inputs they came from. Filter by project, tag or status; agents record them during a task and a human approves them."
ru:
  knowledge:
    newBit: "Новый факт"
    bits: "Знания"
    searchPlaceholder: "Искать знания…"
    refresh: "Обновить"
    empty: "Здесь пока нет знаний."
    allProjects: "Все проекты"
    approved: "Подтверждено"
    pending: "На проверке"
    rejected: "Отклонено"
    superseded: "Заменено"
    by: "автор"
    reviewedBy: "проверил"
    reviewNote: "Заметка проверки"
    originTask: "Записано в задаче"
    sources: "Источники"
    related: "Связанные знания"
    supersedes: "заменяет"
    supersededBy: "заменено на"
    linkedFrom: "ссылается"
    history: "История ревизий"
    edit: "Редактировать"
    approve: "Подтвердить"
    reject: "Отклонить"
    approveTitle: "Подтвердить знание"
    approveHint: "Подтверждение делает факт общей истиной и снимает заменённые им факты."
    rejectTitle: "Отклонить знание"
    rejectHint: "Отклонённый факт остаётся доступным проверяющим; автор может отредактировать его и вернуть на проверку."
    cancel: "Отмена"
    deleteConfirm: "Удалить этот факт? История останется в агентстве."
    selectHint: "Выберите факт из списка или запишите новый. Подтверждённые факты — общая истина; ожидающие ждут человека."
    projectHeading: "Знания проекта"
    orgHeading: "Знания"
    projectDescription: "Записанные факты, общие для людей и агентов проекта. У факта есть автор и жёсткие ссылки на реальные источники."
    orgDescription: "Записанные факты с жёсткими ссылками на исходные данные. Фильтруйте по проекту, тегу или статусу; агенты записывают их во время задачи, а человек подтверждает."
</i18n>
