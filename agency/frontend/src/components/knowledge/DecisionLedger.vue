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
  .flex.flex-wrap.items-start.justify-between.gap-3(v-if="!compact")
    .flex.flex-col
      h2.text-lg.font-semibold {{ heading || $t("decisionLedger.title") }}
      p.text-muted-foreground.max-w-3xl.text-sm {{ description || $t("decisionLedger.hint") }}
    .flex.items-center.gap-2
      Button(variant="ghost" size="sm" @click="load()" :disabled="loading")
        Spinner(v-if="loading" data-icon="inline-start")
        RefreshCwIcon(v-else data-icon="inline-start")
        span {{ $t("decisionLedger.refresh") }}
      Button(v-if="recordScope" size="sm" @click="openRecord")
        PlusIcon(data-icon="inline-start")
        span {{ $t("decisionLedger.record") }}
  p.text-destructive.text-sm(v-if="error") {{ error }}

  .flex.flex-col.gap-3(v-if="groups.length")
    .flex.flex-col.gap-2(v-for="group in groups" :key="group.day")
      span.text-muted-foreground.text-xs.font-medium {{ group.day }}
      .rounded-lg.border.p-3(v-for="row in group.rows" :key="String(row.id)")
        .flex.flex-wrap.items-start.justify-between.gap-2
          .flex.min-w-0.items-center.gap-2
            Badge(:variant="kindVariant(row.kind)") {{ kindLabel(row.kind) }}
            RouterLink.text-sm.font-medium(
              v-if="actorLink(row)"
              class="hover:underline"
              :to="actorLink(row) || { name: 'knowledge' }"
            ) {{ actorName(row) }}
            span.text-sm.font-medium(v-else) {{ actorName(row) }}
            RouterLink.text-muted-foreground.font-mono.text-xs(
              v-if="row.taskId"
              class="hover:text-foreground"
              :to="{ name: 'task', params: { id: row.taskId } }"
            ) {{ row.taskId }}
          span.text-muted-foreground.text-xs(
            :title="formatInstant(rowMs(row))"
          )
            | {{ formatRelativeTime(rowMs(row)) }}
        p.mt-1.text-sm.break-words(v-if="row.summary") {{ row.summary }}
        .mt-1.flex.flex-wrap.items-center.gap-1(v-if="row.conceptPaths.length")
          RouterLink.text-primary.text-xs.underline(
            v-for="path in row.conceptPaths"
            :key="path"
            :to="conceptLink(row, path)"
          ) {{ path }}
        .mt-2.flex.flex-wrap.items-center.gap-1
          Button(
            v-if="canPromote(row)"
            variant="outline"
            size="xs"
            @click="promote(row)"
            :disabled="promoting === String(row.id)"
          )
            Spinner(
              v-if="promoting === String(row.id)"
              data-icon="inline-start"
            )
            FileTextIcon(v-else data-icon="inline-start")
            span {{ $t("decisionLedger.promote") }}
  p.text-muted-foreground.text-sm(v-else-if="!loading") {{ $t("decisionLedger.empty") }}

  Dialog(:open="showRecord" @update:open="showRecord = $event")
    DialogContent(class="sm:max-w-lg")
      DialogHeader
        DialogTitle {{ $t("decisionLedger.recordTitle") }}
        DialogDescription {{ $t("decisionLedger.recordHint") }}
      Field
        FieldLabel(for="decision-summary") {{ $t("decisionLedger.summary") }}
        Input#decision-summary(
          v-model="summary"
          :placeholder="$t('decisionLedger.summaryPlaceholder')"
        )
      Field
        FieldLabel(for="decision-rationale") {{ $t("decisionLedger.rationale") }}
        Textarea#decision-rationale(
          v-model="rationale"
          :placeholder="$t('decisionLedger.rationalePlaceholder')"
        )
      Field(v-if="pageOptions.length")
        FieldLabel {{ $t("decisionLedger.linkPages") }}
        .flex.flex-wrap.items-center.gap-1
          button.rounded-full.border.px-2.text-xs(
            v-for="page in pageOptions"
            :key="page.path"
            type="button"
            class="py-0.5"
            :class="linkedPaths.includes(page.path) ? 'bg-muted font-medium' : 'hover:bg-muted/60'"
            @click="togglePath(page.path)"
          ) {{ page.path }}
      p.text-destructive.text-sm(v-if="recordError") {{ recordError }}
      DialogFooter
        DialogClose
          Button(type="button" variant="ghost") {{ $t("common.cancel") }}
        Button(
          type="button"
          @click="record"
          :disabled="recording || !summary.trim()"
        )
          Spinner(v-if="recording" data-icon="inline-start")
          span {{ $t("decisionLedger.record") }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { formatInstant, formatRelativeTime } from "@/lib/format";
import { agency } from "@/lib/sdk";
import type { Decision, KnowledgeConcept } from "@cherrypi/sdk";
import { FileTextIcon, PlusIcon, RefreshCwIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { RouteLocationRaw } from "vue-router";
import { useRoute, useRouter } from "vue-router";

const props = defineProps<{
  scope?: "org" | "project";
  scopeId?: string;
  taskId?: string;
  heading?: string;
  description?: string;

  compact?: boolean;
}>();

const router = useRouter();
const route = useRoute();
const { t } = useI18n();

const rows = ref<Decision[]>([]);
const loading = ref(false);
const error = ref("");
const showRecord = ref(false);
const summary = ref("");
const rationale = ref("");
const linkedPaths = ref<string[]>([]);
const pageOptions = ref<KnowledgeConcept[]>([]);
const recordError = ref("");
const recording = ref(false);
const promoting = ref("");

const recordScope = computed(() => {
  if (!props.scope) return false;
  if (props.scope === "project") return Boolean(props.scopeId);
  return true;
});

const groups = computed(() => {
  const out: { day: string; rows: Decision[] }[] = [];
  for (const row of rows.value) {
    const day = new Date(rowMs(row)).toLocaleDateString();
    const last = out.at(-1);
    if (last && last.day === day) last.rows.push(row);
    else out.push({ day, rows: [row] });
  }
  return out;
});

function rowMs(row: Decision): number {
  return row.createdAt ? Number(row.createdAt.seconds) * 1000 : 0;
}

function conceptLink(row: Decision, path: string): RouteLocationRaw {
  if (row.projectSlug && row.projectSlug !== props.scopeId) {
    return {
      name: "project",
      params: { slug: row.projectSlug },
      query: { path, tab: "wiki" },
    };
  }
  return { name: "knowledge", query: { path } };
}

function kindLabel(kind: string): string {
  switch (kind) {
    case "task_created":
      return t("decisionLedger.kind.taskCreated");
    case "follow_up":
      return t("decisionLedger.kind.followUp");
    case "question_answered":
      return t("decisionLedger.kind.questionAnswered");
    case "task_settled":
      return t("decisionLedger.kind.taskSettled");
    case "recorded":
      return t("decisionLedger.kind.recorded");
    default:
      return kind.replace(/_/g, " ");
  }
}

function kindVariant(
  kind: string,
): "default" | "secondary" | "outline" | "destructive" {
  switch (kind) {
    case "recorded":
      return "default";
    case "task_created":
      return "secondary";
    default:
      return "outline";
  }
}

function actorLink(row: Decision): RouteLocationRaw | undefined {
  if (row.actorRef.startsWith("human:")) {
    const username = row.actorRef.slice("human:".length);
    return username ? { name: "user", params: { username } } : undefined;
  }
  if (row.actorRef.startsWith("cherrypi-agent/")) {
    const username = row.actorRef.slice("cherrypi-agent/".length);
    return username ? { name: "agent", params: { username } } : undefined;
  }
  return undefined;
}

function actorName(row: Decision): string {
  if (row.actorRef.startsWith("human:"))
    return row.actorRef.slice("human:".length);
  if (row.actorRef.startsWith("cherrypi-agent/")) {
    return row.actorRef.slice("cherrypi-agent/".length);
  }
  return row.actorRef || "unknown";
}

function promoteTarget(
  row: Decision,
): { scope: string; scopeId: string } | undefined {
  if (row.projectId) return { scope: "project", scopeId: row.projectId };
  if (props.scope === "project" && props.scopeId) {
    return { scope: "project", scopeId: props.scopeId };
  }
  if (props.scope === "org") return { scope: "org", scopeId: "" };
  return undefined;
}

function canPromote(row: Decision): boolean {
  return Boolean(promoteTarget(row)) && row.kind !== "recorded";
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    rows.value = await agency.listDecisions({
      scope: props.scope === "project" ? "project" : "",
      scopeId: props.scopeId,
      taskId: props.taskId,
      limit: 200,
    });
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function openRecord(preselect?: string) {
  showRecord.value = true;
  recordError.value = "";
  try {
    pageOptions.value = await agency.listConcepts({
      scope: props.scope ?? "org",
      scopeId: props.scopeId,
      limit: 100,
    });
  } catch {
    pageOptions.value = [];
  }
  if (preselect) {
    linkedPaths.value = pageOptions.value.some(
      (page) => page.path === preselect,
    )
      ? [preselect]
      : [];
  }
}

function consumeRecordQuery() {
  if (typeof route.query.record !== "string") return;
  const page =
    typeof route.query.page === "string" ? route.query.page : undefined;
  const next = Object.fromEntries(
    Object.entries(route.query).filter(
      ([key]) => key !== "record" && key !== "page",
    ),
  );
  void router.replace({ query: next });
  void openRecord(page);
}

watch(
  () => route.query.record,
  () => consumeRecordQuery(),
);

function togglePath(path: string) {
  linkedPaths.value = linkedPaths.value.includes(path)
    ? linkedPaths.value.filter((item) => item !== path)
    : [...linkedPaths.value, path];
}

async function record() {
  recording.value = true;
  recordError.value = "";
  try {
    await agency.recordDecision({
      scope: props.scope,
      scopeId: props.scopeId,
      taskId: props.taskId,
      summary: summary.value.trim(),
      rationale: rationale.value.trim(),
      conceptPaths: linkedPaths.value,
    });
    summary.value = "";
    rationale.value = "";
    linkedPaths.value = [];
    showRecord.value = false;
    await load();
  } catch (err) {
    recordError.value = err instanceof Error ? err.message : String(err);
  } finally {
    recording.value = false;
  }
}

async function promote(row: Decision) {
  const target = promoteTarget(row);
  if (!target) return;
  promoting.value = String(row.id);
  try {
    const date = new Date(rowMs(row)).toISOString().slice(0, 10);
    const title =
      row.summary.split("\n")[0]!.slice(0, 80) || `Decision ${date}`;
    const path = `decisions/${row.kind}-${row.taskId || row.id}`
      .toLowerCase()
      .replace(/[^a-z0-9._/-]+/g, "-")
      .replace(/-+/g, "-");
    const task = row.taskId
      ? `\n\nTask: [${row.taskId}](/tasks/${row.taskId})`
      : "";
    await agency.createConcept({
      scope: target.scope,
      scopeId: target.scopeId,
      path,
      type: "Decision",
      title,
      description: t("decisionLedger.promotedDescription", {
        actor: row.actorRef,
      }),
      status: "draft",
      body:
        `# ${title}\n\n${row.summary}${task}\n\n` +
        t("decisionLedger.promotedAttribution", {
          actor: row.actorRef,
          date,
        }),
      summary: t("decisionLedger.promotedSummary", {
        kind: kindLabel(row.kind),
      }),
    });
    if (target.scope === "project") {
      const project = await agency.getProject(target.scopeId);
      await router.push({
        name: "project",
        params: { slug: project.slug },
        query: { tab: "wiki", path },
      });
    } else {
      await router.push({ name: "knowledge", query: { path } });
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    promoting.value = "";
  }
}

onMounted(() => {
  void load();
  consumeRecordQuery();
});

useRealtime(["knowledge"], () => {
  void load();
});
</script>

<i18n lang="yaml">
en:
  decisionLedger:
    title: "Decisions"
    hint: "Who is responsible for what: task creations, follow-up prompts, answered questions and settlements the agency captured, plus decisions recorded on purpose."
    refresh: "Refresh"
    record: "Record decision"
    empty: "No decisions recorded yet."
    recordTitle: "Record a decision"
    recordHint: "It is attributed to you and stays in the ledger. Link the pages it produced so a reader can see who decided and why."
    summary: "Decision"
    summaryPlaceholder: "What was decided, in one line"
    rationale: "Rationale"
    rationalePlaceholder: "Why, in your own words"
    linkPages: "Pages this decision produced"
    promote: "Promote to page"
    promotedDescription: "Recorded by {actor}"
    promotedSummary: "Promoted from the decision ledger ({kind})"
    promotedAttribution: "Recorded by {actor} on {date}."
    kind:
      taskCreated: "task created"
      followUp: "follow-up"
      questionAnswered: "question answered"
      taskSettled: "task settled"
      recorded: "recorded"
ru:
  decisionLedger:
    title: "Решения"
    hint: "Кто за что отвечает: создание задач, дополнительные промпты, ответы на вопросы и завершения, записанные агентством, а также решения, записанные явно."
    refresh: "Обновить"
    record: "Записать решение"
    empty: "Решений пока нет."
    recordTitle: "Записать решение"
    recordHint: "Оно будет атрибутировано вам и останется в журнале. Свяжите страницы, которые оно породило, чтобы читатель видел, кто решил и почему."
    summary: "Решение"
    summaryPlaceholder: "Что решено, одной строкой"
    rationale: "Обоснование"
    rationalePlaceholder: "Почему, своими словами"
    linkPages: "Страницы, которые породило решение"
    promote: "В страницу"
    promotedDescription: "Записал(а) {actor}"
    promotedSummary: "Вынесено из журнала решений ({kind})"
    promotedAttribution: "Записал(а) {actor}, {date}."
    kind:
      taskCreated: "задача создана"
      followUp: "доп. промпт"
      questionAnswered: "ответ на вопрос"
      taskSettled: "прогон завершён"
      recorded: "записано"
</i18n>
