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
  .flex.flex-wrap.items-center.justify-between.gap-3
    .flex.flex-col
      h2.text-lg.font-semibold
        | {{ heading || (scope === "project" ? $t("knowledge.projectHeading") : $t("knowledge.orgHeading")) }}
      p.text-muted-foreground.max-w-3xl.text-sm
        | {{ description || (scope === "project" ? $t("knowledge.projectWikiDescription") : $t("knowledge.orgDescription")) }}
    Button(v-if="selected?.canEdit || !selected" @click="openCreate")
      PlusIcon(data-icon="inline-start")
      span {{ $t("knowledge.newPage") }}
  .grid.items-start.gap-4(class="lg:grid-cols-[18rem_minmax(0,1fr)]")
    Card
      CardHeader.pb-3
        CardTitle.text-sm {{ $t("knowledge.pages") }}
      CardContent.flex.flex-col.gap-3
        .relative
          SearchIcon.text-muted-foreground.absolute.top-2.left-2.size-4
          Input.pl-8(
            v-model="search"
            :placeholder="$t('knowledge.searchPlaceholder')"
            @keyup.enter="load()"
          )
        .flex.items-center.justify-between.gap-2
          span.text-muted-foreground.text-xs {{ pages.length }}
          Button(variant="ghost" size="sm" @click="load()" :disabled="loading")
            Spinner(v-if="loading" data-icon="inline-start")
            RefreshCwIcon(v-else data-icon="inline-start")
            span {{ $t("knowledge.refresh") }}
        p.text-destructive.text-xs(v-if="error") {{ error }}
        .flex.flex-col.gap-1
          button.w-full.rounded-lg.border.border-transparent.px-2.py-2.text-left.transition-colors(
            v-for="page in pages"
            :key="page.id"
            type="button"
            :class="selected?.id === page.id ? 'bg-muted' : 'hover:bg-muted/60'"
            @click="select(page)"
          )
            .flex.items-center.gap-2
              FileTextIcon.text-muted-foreground.size-3.shrink-0
              span.min-w-0.flex-1.truncate.text-sm.font-medium {{ page.title || page.path }}
            .flex.flex-wrap.items-center.gap-1.pl-5(class="mt-0.5")
              span.text-muted-foreground.truncate.font-mono(class="text-[10px]") {{ page.path }}
            .mt-1.flex.flex-wrap.items-center.gap-1.pl-5
              Badge(class="text-[10px]" :variant="trustVariant(page.trustTier)") {{ trustLabel(page.trustTier) }}
              Badge(class="text-[10px]" variant="outline") {{ page.status }}
          p.text-muted-foreground.text-sm(v-if="!pages.length && !loading")
            | {{ $t("knowledge.empty") }}

    div(v-if="selected")
      Card
        CardHeader.flex.flex-col.gap-2
          .flex.flex-wrap.items-start.justify-between.gap-3
            .flex.min-w-0.flex-col.gap-1
              .flex.flex-wrap.items-center.gap-2
                h3.min-w-0.truncate.text-xl.font-semibold {{ selected.title || selected.path }}
                Badge(:variant="trustVariant(selected.trustTier)") {{ trustLabel(selected.trustTier) }}
                Badge(variant="outline") {{ selected.status }}
              span.text-muted-foreground.truncate.font-mono.text-xs {{ selected.path }}
            .flex.flex-wrap.items-center.gap-1
              Button(
                v-if="selected.canEdit"
                variant="outline"
                size="sm"
                @click="openEdit"
              )
                PencilIcon(data-icon="inline-start")
                span {{ $t("knowledge.edit") }}
              Button(variant="outline" size="sm" @click="openHistory")
                HistoryIcon(data-icon="inline-start")
                span {{ $t("knowledge.history") }}
              Button(
                v-if="selected.canVerify"
                variant="outline"
                size="sm"
                @click="showVerify = true"
              )
                ShieldCheckIcon(data-icon="inline-start")
                span {{ $t("knowledge.verify") }}
              Button(
                v-if="selected.canEdit"
                variant="destructive"
                size="sm"
                @click="remove"
              )
                Trash2Icon(data-icon="inline-start")
                span {{ $t("knowledge.delete") }}
          p.text-muted-foreground.text-sm(v-if="selected.description") {{ selected.description }}
          .text-muted-foreground.flex.flex-wrap.items-center.gap-3.text-xs
            span {{ $t("knowledge.generatedBy") }}
              strong.text-foreground {{ " " + (selected.generatedBy || "unknown") }}
            span {{ formatTimestamp(selected.generatedAt) }}
            span {{ $t("knowledge.revision", { revision: String(selected.revision) }) }}
            span(v-if="selected.staleAfter") {{ $t("knowledge.staleAfter", { date: formatTimestamp(selected.staleAfter) }) }}
          .flex.flex-wrap.items-center.gap-1(v-if="selected.tags.length")
            Badge(v-for="tag in selected.tags" :key="tag" variant="secondary")
              | {{ "#" + tag }}
        CardContent.pt-0
          .rounded-lg.border.p-3.text-sm.text-amber-700(
            v-if="isStale"
            class="border-amber-500/40 bg-amber-500/10 dark:text-amber-300"
          )
            | {{ $t("knowledge.staleBanner") }}
          MarkdownText.prose.prose-sm.mt-4.max-w-none(
            class="prose-p:my-1 prose-pre:my-2"
            :text="bodyMarkdown"
          )
          .mt-6.flex.flex-col.gap-2(v-if="selected.sources.length")
            h4.text-sm.font-semibold {{ $t("knowledge.sources") }}
            template(v-for="source in selected.sources" :key="source.id")
              a.text-primary.text-sm.underline(
                v-if="isLink(source.resource)"
                rel="noopener noreferrer"
                :href="source.resource"
                target="_blank"
              ) {{ source.title || source.resource }}
              span.text-sm(v-else) {{ source.title || source.resource }}
          .mt-6.flex.flex-col.gap-2(v-if="selected.backlinks.length")
            h4.text-sm.font-semibold {{ $t("knowledge.referencedBy") }}
            RouterLink.text-primary.text-sm.underline(
              v-for="backlink in selected.backlinks"
              :key="backlink.id"
              :to="{ path: route.path, query: { ...route.query, path: backlink.path } }"
            ) {{ backlink.title || backlink.path }}
          .mt-6.flex.flex-col.gap-2(v-if="selected.decisions.length")
            .flex.flex-wrap.items-center.justify-between.gap-2
              h4.text-sm.font-semibold {{ $t("knowledge.decisions") }}
            .rounded-lg.border.p-2(
              v-for="decision in selected.decisions"
              :key="String(decision.id)"
            )
              .flex.flex-wrap.items-center.gap-2
                Badge(:variant="decisionKindVariant(decision.kind)")
                  | {{ decisionKindLabel(decision.kind) }}
                span.text-sm.font-medium {{ decisionActorName(decision.actorRef) }}
                span.text-muted-foreground.text-xs(v-if="decision.createdAt") {{ formatTimestamp(decision.createdAt) }}
              p.mt-1.text-sm.break-words(v-if="decision.summary") {{ decision.summary }}
          .mt-6.flex.flex-col.gap-2(v-if="selected.canEdit")
            Button(
              variant="outline"
              size="sm"
              @click="recordDecisionAboutPage"
            )
              PlusIcon(data-icon="inline-start")
              span {{ $t("knowledge.recordDecisionAboutPage") }}
          .mt-6.flex.flex-col.gap-2(v-if="selected.verifications.length")
            h4.text-sm.font-semibold {{ $t("knowledge.verifications") }}
            .flex.flex-wrap.items-center.gap-2.text-xs(
              v-for="verification in selected.verifications"
              :key="String(verification.id)"
            )
              Badge(
                :variant="verification.revokedAt ? 'outline' : 'secondary'"
              )
                | {{ verification.actorRef }}
              span.text-muted-foreground {{ formatTimestamp(verification.createdAt) }}
              span.text-muted-foreground(v-if="verification.note") {{ verification.note }}
    Card(v-else)
      CardContent.flex.flex-col.items-center.gap-3.p-10.text-center
        FileTextIcon.text-muted-foreground.size-8
        p.text-sm.font-medium {{ $t("knowledge.selectPage") }}
        p.text-muted-foreground.text-sm
          | {{ $t("knowledge.selectHint") }}

  KnowledgeEditor(
    v-model:open="showEditor"
    :scope="scope"
    :scope-id="scopeId"
    :concept="editing"
    @saved="onSaved"
  )

  Dialog(:open="showHistory" @update:open="showHistory = $event")
    DialogContent.overflow-y-auto(class="max-h-[85vh] sm:max-w-2xl")
      DialogHeader
        DialogTitle {{ $t("knowledge.historyTitle") }}
        DialogDescription {{ $t("knowledge.historyHint") }}
      .flex.flex-col.gap-2(v-if="revisions.length")
        .rounded-lg.border.p-3(
          v-for="revision in revisions"
          :key="String(revision.id)"
        )
          .flex.flex-wrap.items-center.justify-between.gap-2
            span.text-sm.font-medium {{ $t("knowledge.revisionLine", { revision: String(revision.revision), title: revision.title }) }}
            span.text-muted-foreground.text-xs {{ formatTimestamp(revision.createdAt) }}
          p.text-muted-foreground.text-xs
            | {{ revision.actorRef }} · {{ revision.summary || "No summary" }}
          details.mt-2
            summary.text-primary.cursor-pointer.text-xs {{ $t("knowledge.showBody") }}
            MarkdownText.mt-2.max-h-64.overflow-auto.text-xs(
              :text="revision.body"
            )
      p.text-muted-foreground.text-sm(v-else) {{ $t("knowledge.noRevisions") }}

  Dialog(:open="showVerify" @update:open="showVerify = $event")
    DialogContent(class="sm:max-w-md")
      DialogHeader
        DialogTitle {{ $t("knowledge.verifyTitle") }}
        DialogDescription {{ $t("knowledge.verifyHint") }}
      Field
        FieldLabel(for="knowledge-verify-note") {{ $t("knowledge.note") }}
        Input#knowledge-verify-note(
          v-model="verifyNote"
          :placeholder="$t('knowledge.verifyPlaceholder')"
        )
      p.text-destructive.text-sm(v-if="verifyError") {{ verifyError }}
      DialogFooter
        DialogClose
          Button(type="button" variant="ghost") {{ $t("knowledge.cancel") }}
        Button(type="button" @click="verify" :disabled="verifying")
          Spinner(v-if="verifying" data-icon="inline-start")
          span {{ $t("knowledge.verify") }}
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
import { useRealtime } from "@/composables/useRealtime";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type { KnowledgeConcept, KnowledgeRevision } from "@cherrypi/sdk";
import {
  FileTextIcon,
  HistoryIcon,
  PencilIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
  ShieldCheckIcon,
  Trash2Icon,
} from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute, useRouter } from "vue-router";

const props = defineProps<{
  scope: "org" | "project";
  scopeId?: string;
  heading?: string;
  description?: string;
}>();

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const pages = ref<KnowledgeConcept[]>([]);
const selected = ref<KnowledgeConcept>();
const search = ref("");
const loading = ref(false);
const error = ref("");
const showEditor = ref(false);
const editing = ref<KnowledgeConcept>();
const showHistory = ref(false);
const revisions = ref<KnowledgeRevision[]>([]);
const showVerify = ref(false);
const verifyNote = ref("");
const verifyError = ref("");
const verifying = ref(false);

const isStale = computed(() => {
  const value = selected.value?.staleAfter;
  if (!value) return false;
  return Number(value.seconds) * 1000 < Date.now();
});

const bodyMarkdown = computed(() => {
  const body = selected.value?.body ?? "";
  const base = route.path;
  return body.replace(/\[\[([a-z0-9][a-z0-9._/-]*)\]\]/g, (_match, path) => {
    const target = `${base}?path=${encodeURIComponent(path)}`;
    return `[${path}](${target})`;
  });
});

function isLink(resource: string): boolean {
  return /^https?:\/\//i.test(resource);
}

function decisionKindLabel(kind: string): string {
  switch (kind) {
    case "task_created":
      return t("knowledge.decisionKind.taskCreated");
    case "follow_up":
      return t("knowledge.decisionKind.followUp");
    case "question_answered":
      return t("knowledge.decisionKind.questionAnswered");
    case "task_settled":
      return t("knowledge.decisionKind.taskSettled");
    case "recorded":
      return t("knowledge.decisionKind.recorded");
    default:
      return kind.replace(/_/g, " ");
  }
}

function decisionKindVariant(
  kind: string,
): "default" | "secondary" | "outline" {
  return kind === "recorded" ? "default" : "outline";
}

function decisionActorName(ref: string): string {
  if (ref.startsWith("human:")) return ref.slice("human:".length);
  if (ref.startsWith("cherrypi-agent/")) {
    return ref.slice("cherrypi-agent/".length);
  }
  return ref;
}

function recordDecisionAboutPage() {
  if (!selected.value) return;

  void router.push({
    name: route.name,
    query: {
      ...route.query,
      ...(route.name === "project" ? { tab: "decisions" } : {}),
      record: "decision",
      page: selected.value.path,
    },
  });
}

function trustLabel(tier: string): string {
  switch (tier) {
    case "human-reviewed":
      return "human reviewed";
    case "machine-confirmed":
      return "machine confirmed";
    default:
      return "unverified";
  }
}

function trustVariant(
  tier: string,
): "default" | "secondary" | "outline" | "destructive" {
  switch (tier) {
    case "human-reviewed":
      return "default";
    case "machine-confirmed":
      return "secondary";
    default:
      return "outline";
  }
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    pages.value = await agency.listConcepts({
      scope: props.scope,
      scopeId: props.scopeId,
      query: search.value.trim(),
      limit: 200,
    });
    const wanted =
      typeof route.query.path === "string" ? route.query.path : undefined;
    const current = wanted || selected.value?.path || pages.value[0]?.path;
    if (current) {
      await selectPath(current, current !== route.query.path);
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function selectPath(path: string, updateRoute = true) {
  try {
    selected.value = await agency.getConcept(
      props.scope,
      props.scopeId ?? "",
      path,
    );
    if (updateRoute) {
      const next = { ...route.query, path };
      void router.replace({ query: next });
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function select(page: KnowledgeConcept) {
  await selectPath(page.path);
}

function openCreate() {
  editing.value = undefined;
  showEditor.value = true;
}

function openEdit() {
  editing.value = selected.value;
  showEditor.value = true;
}

async function onSaved(concept: KnowledgeConcept) {
  await load();
  await selectPath(concept.path);
}

async function openHistory() {
  if (!selected.value) return;
  showHistory.value = true;
  try {
    revisions.value = await agency.listConceptRevisions(
      props.scope,
      props.scopeId ?? "",
      selected.value.path,
    );
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function verify() {
  if (!selected.value) return;
  verifying.value = true;
  verifyError.value = "";
  try {
    selected.value = await agency.verifyConcept(
      props.scope,
      props.scopeId ?? "",
      selected.value.path,
      verifyNote.value.trim(),
    );
    verifyNote.value = "";
    showVerify.value = false;
    await load();
  } catch (err) {
    verifyError.value = err instanceof Error ? err.message : String(err);
  } finally {
    verifying.value = false;
  }
}

async function remove() {
  if (!selected.value) return;
  if (
    !window.confirm(
      `Delete “${selected.value.title}”? It disappears from the wiki; the agency keeps its history.`,
    )
  ) {
    return;
  }
  try {
    await agency.deleteConcept(
      props.scope,
      props.scopeId ?? "",
      selected.value.path,
      selected.value.revision,
    );
    selected.value = undefined;
    const next = Object.fromEntries(
      Object.entries(route.query).filter(([key]) => key !== "path"),
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
  () => [props.scope, props.scopeId],
  () => {
    selected.value = undefined;
    void load();
  },
);

watch(
  () => route.query.path,
  (value) => {
    if (typeof value === "string" && value && value !== selected.value?.path) {
      void selectPath(value, false);
    }
  },
);

onMounted(() => {
  void load();
});

useRealtime(["knowledge"], () => {
  void load();
});
</script>

<i18n lang="yaml">
en:
  knowledge:
    newPage: "New page"
    pages: "Pages"
    searchPlaceholder: "Search the wiki…"
    refresh: "Refresh"
    empty: "No pages yet. Create the first one."
    edit: "Edit"
    history: "History"
    verify: "Verify"
    delete: "Delete"
    generatedBy: "Generated by"
    revision: "Revision {revision}"
    staleAfter: "Stale after {date}"
    staleBanner: "This page is past its stale-after date. Verify it or refresh it before relying on it."
    sources: "Sources"
    referencedBy: "Referenced by"
    verifications: "Verifications"
    decisions: "Decisions"
    recordDecisionAboutPage: "Record a decision about this page"
    decisionKind:
      taskCreated: "task created"
      followUp: "follow-up"
      questionAnswered: "question answered"
      taskSettled: "task settled"
      recorded: "recorded"
    selectPage: "Select a page"
    selectHint: "Pick a page from the list, or create a new one. The wiki is shared by humans and agents; every write is attributed."
    historyTitle: "Revision history"
    historyHint: "Every write snapshots the page; older revisions stay readable."
    revisionLine: "Revision {revision} · {title}"
    showBody: "Show body"
    noRevisions: "No revisions recorded."
    verifyTitle: "Verify page"
    verifyHint: "A verification records that a human checked this page. It does not change the body."
    note: "Note"
    verifyPlaceholder: "What did you check?"
    cancel: "Cancel"
    wikiTab: "Wiki"
    decisionsTab: "Decisions"
    projectWikiDescription: "OKF pages shared by this project's humans and agents. Writes are attributed and unverified until a human verifies them."
    projectHeading: "Project wiki"
    orgHeading: "Organization wiki"
    orgDescription: "The agency-wide OKF knowledge base. Humans and agents write here; every page carries its author, revision history and verification state. Project wikis live on each project's Wiki tab."
    memoryTab: "Memory"
ru:
  knowledge:
    newPage: "Новая страница"
    pages: "Страницы"
    searchPlaceholder: "Искать по вики…"
    refresh: "Обновить"
    empty: "Страниц пока нет. Создайте первую."
    edit: "Редактировать"
    history: "История"
    verify: "Подтвердить"
    delete: "Удалить"
    generatedBy: "Автор"
    revision: "Ревизия {revision}"
    staleAfter: "Устаревает после {date}"
    staleBanner: "Срок актуальности страницы истёк. Подтвердите или обновите её, прежде чем полагаться на неё."
    sources: "Источники"
    referencedBy: "Ссылаются на страницу"
    verifications: "Подтверждения"
    decisions: "Решения"
    recordDecisionAboutPage: "Записать решение по этой странице"
    decisionKind:
      taskCreated: "задача создана"
      followUp: "доп. промпт"
      questionAnswered: "ответ на вопрос"
      taskSettled: "прогон завершён"
      recorded: "записано"
    selectPage: "Выберите страницу"
    selectHint: "Выберите страницу из списка или создайте новую. Вики общая для людей и агентов; каждая запись атрибутируется."
    historyTitle: "История ревизий"
    historyHint: "Каждая запись фиксирует снимок страницы; старые ревизии остаются доступными."
    revisionLine: "Ревизия {revision} · {title}"
    showBody: "Показать текст"
    noRevisions: "Ревизий пока нет."
    verifyTitle: "Подтвердить страницу"
    verifyHint: "Подтверждение фиксирует, что человек проверил страницу. Текст при этом не меняется."
    note: "Заметка"
    verifyPlaceholder: "Что вы проверили?"
    cancel: "Отмена"
    wikiTab: "Вики"
    decisionsTab: "Решения"
    projectWikiDescription: "Страницы OKF, общие для людей и агентов этого проекта. Записи атрибутируются и остаются неподтверждёнными, пока человек их не проверит."
    projectHeading: "Вики проекта"
    orgHeading: "Вики организации"
    orgDescription: "Общая база знаний агентства в формате OKF. Здесь пишут люди и агенты; у каждой страницы есть автор, история ревизий и состояние проверки. Вики проектов живут на вкладке Wiki соответствующего проекта."
    memoryTab: "Память"
</i18n>
