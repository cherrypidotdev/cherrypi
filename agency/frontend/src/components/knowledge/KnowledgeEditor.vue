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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent.overflow-y-auto(class="max-h-[90vh] sm:max-w-3xl")
    DialogHeader
      DialogTitle {{ concept ? $t("knowledgeEditor.editTitle") : $t("knowledgeEditor.createTitle") }}
      DialogDescription
        | {{ $t("knowledgeEditor.hint") }}
    form.flex.flex-col.gap-4(@submit.prevent="save")
      .grid.gap-4(class="sm:grid-cols-2")
        Field
          FieldLabel(for="knowledge-title") {{ $t("knowledgeEditor.title") }}
          Input#knowledge-title(
            v-model="title"
            :placeholder="$t('knowledgeEditor.titlePlaceholder')"
            autocomplete="off"
          )
        Field
          FieldLabel(for="knowledge-path") {{ $t("knowledgeEditor.path") }}
          Input#knowledge-path(
            v-model="path"
            :placeholder="$t('knowledgeEditor.pathPlaceholder')"
            autocomplete="off"
            :disabled="Boolean(concept)"
          )
          FieldDescription
            | {{ $t("knowledgeEditor.pathHint") }}
      .grid.gap-4(class="sm:grid-cols-3")
        Field
          FieldLabel(for="knowledge-type") {{ $t("knowledgeEditor.type") }}
          Input#knowledge-type(
            v-model="type"
            :placeholder="$t('knowledgeEditor.typePlaceholder')"
          )
        Field
          FieldLabel(for="knowledge-status") {{ $t("knowledgeEditor.status") }}
          select#knowledge-status.border-input.w-full.rounded-lg.border.bg-transparent.px-2.py-1.text-sm(
            v-model="status"
          )
            option(value="stable") {{ $t("knowledgeEditor.stable") }}
            option(value="draft") {{ $t("knowledgeEditor.draft") }}
            option(value="deprecated") {{ $t("knowledgeEditor.deprecated") }}
        Field
          FieldLabel(for="knowledge-stale") {{ $t("knowledgeEditor.staleAfter") }}
          Input#knowledge-stale(v-model="staleAfter" type="date")
      Field
        FieldLabel(for="knowledge-description") {{ $t("knowledgeEditor.description") }}
        Input#knowledge-description(
          v-model="description"
          :placeholder="$t('knowledgeEditor.descriptionPlaceholder')"
        )
      Field
        FieldLabel(for="knowledge-tags") {{ $t("knowledgeEditor.tags") }}
        Input#knowledge-tags(
          v-model="tags"
          :placeholder="$t('knowledgeEditor.tagsPlaceholder')"
          autocomplete="off"
        )
      Field(v-if="decisions.length")
        FieldLabel {{ $t("knowledgeEditor.decisions") }}
        FieldDescription {{ $t("knowledgeEditor.decisionsHint") }}
        .flex.flex-col.gap-1
          button.flex.w-full.items-center.gap-2.rounded-md.border.px-2.py-1.text-left.text-xs(
            v-for="decision in decisions"
            :key="String(decision.id)"
            type="button"
            :class="picked.includes(String(decision.id)) ? 'bg-muted font-medium' : 'hover:bg-muted/60'"
            :title="decision.summary"
            @click="toggleDecision(decision.id)"
          )
            span.shrink-0.rounded-full.border.px-1 {{ kindLabel(decision.kind) }}
            span.min-w-0.flex-1.truncate {{ decision.summary || decision.actorRef }}
      Field
        FieldLabel(for="knowledge-body") {{ $t("knowledgeEditor.body") }}
        MarkdownEditor#knowledge-body(
          v-model="body"
          :placeholder="$t('knowledgeEditor.bodyPlaceholder')"
          :resizable="true"
          :height="'20rem'"
        )
      p.text-destructive.text-sm(v-if="error") {{ error }}
      DialogFooter
        DialogClose
          Button(type="button" variant="ghost") {{ $t("knowledgeEditor.cancel") }}
        Button(
          type="submit"
          :disabled="saving || !title.trim() || !path.trim()"
        )
          Spinner(v-if="saving" data-icon="inline-start")
          span {{ concept ? $t("knowledgeEditor.save") : $t("knowledgeEditor.create") }}
</template>

<script setup lang="ts">
import MarkdownEditor from "@/components/MarkdownEditor.vue";
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
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { agency } from "@/lib/sdk";
import type { Decision, KnowledgeConcept } from "@cherrypi/sdk";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  open: boolean;
  scope: string;
  scopeId?: string;
  concept?: KnowledgeConcept;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  saved: [concept: KnowledgeConcept];
}>();

const title = ref("");
const path = ref("");
const type = ref("Note");
const description = ref("");
const tags = ref("");
const status = ref("stable");
const staleAfter = ref("");
const body = ref("");
const saving = ref(false);
const error = ref("");
const decisions = ref<Decision[]>([]);
const picked = ref<string[]>([]);

function reset() {
  const concept = props.concept;
  title.value = concept?.title ?? "";
  path.value = concept?.path ?? "";
  type.value = concept?.type || "Note";
  description.value = concept?.description ?? "";
  tags.value = concept?.tags.join(", ") ?? "";
  status.value = concept?.status || "stable";
  staleAfter.value = concept?.staleAfter
    ? new Date(Number(concept.staleAfter.seconds) * 1000)
        .toISOString()
        .slice(0, 10)
    : "";
  body.value = concept?.body ?? "";
  picked.value = (concept?.decisions ?? []).map((item) => String(item.id));
  error.value = "";
}

async function loadDecisions() {
  try {
    decisions.value = await agency.listDecisions({
      scope: props.scope,
      scopeId: props.scopeId,
      limit: 50,
    });
  } catch {
    decisions.value = [];
  }
}

function toggleDecision(id: bigint) {
  const key = String(id);
  picked.value = picked.value.includes(key)
    ? picked.value.filter((item) => item !== key)
    : [...picked.value, key];
}

function kindLabel(kind: string): string {
  switch (kind) {
    case "task_created":
      return t("knowledgeEditor.kind.taskCreated");
    case "follow_up":
      return t("knowledgeEditor.kind.followUp");
    case "question_answered":
      return t("knowledgeEditor.kind.questionAnswered");
    case "task_settled":
      return t("knowledgeEditor.kind.taskSettled");
    case "recorded":
      return t("knowledgeEditor.kind.recorded");
    default:
      return kind.replace(/_/g, " ");
  }
}

watch(
  () => [props.open, props.concept],
  () => {
    if (!props.open) return;
    reset();
    void loadDecisions();
  },
  { immediate: true },
);

async function save() {
  saving.value = true;
  error.value = "";
  const tagsList = tags.value
    .split(",")
    .map((tag) => tag.trim().replace(/^#/, ""))
    .filter(Boolean);
  const stale = staleAfter.value
    ? new Date(`${staleAfter.value}T00:00:00Z`)
    : undefined;
  try {
    if (props.concept) {
      const updated = await agency.updateConcept({
        scope: props.scope,
        scopeId: props.scopeId,
        path: props.concept.path,
        baseRevision: props.concept.revision,
        title: title.value.trim(),
        description: description.value.trim(),
        type: type.value.trim() || "Note",
        status: status.value,
        tags: tagsList,
        replaceTags: true,
        staleAfter: stale,
        clearStaleAfter: !stale,
        body: body.value,
        decisionIds: picked.value.map((id) => BigInt(id)),
        replaceDecisions: true,
        summary: "Edited in the dashboard",
      });
      emit("saved", updated);
    } else {
      const created = await agency.createConcept({
        scope: props.scope,
        scopeId: props.scopeId,
        path: path.value.trim(),
        title: title.value.trim(),
        description: description.value.trim(),
        type: type.value.trim() || "Note",
        status: status.value,
        tags: tagsList,
        staleAfter: stale,
        body: body.value,
        decisionIds: picked.value.map((id) => BigInt(id)),
        summary: "Created in the dashboard",
      });
      emit("saved", created);
    }
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  knowledgeEditor:
    createTitle: "New page"
    editTitle: "Edit page"
    hint: "Writes are attributed to you and stay unverified until a human verifies them."
    title: "Title"
    titlePlaceholder: "A short, human title"
    path: "Path"
    pathPlaceholder: "architecture/overview"
    pathHint: "Lower-kebab path; immutable after creation."
    type: "Type"
    typePlaceholder: "Note"
    status: "Status"
    stable: "stable"
    draft: "draft"
    deprecated: "deprecated"
    staleAfter: "Stale after"
    description: "Description"
    descriptionPlaceholder: "One line that lets a reader decide whether to open the page"
    tags: "Tags"
    tagsPlaceholder: "decision, api, onboarding"
    body: "Body"
    bodyPlaceholder: "Write the page in Markdown. Link other pages with [[path]]."
    decisions: "Decisions this page records"
    decisionsHint: "A linked decision appears on the page and in the ledger, so a reader sees who decided and why."
    kind:
      taskCreated: "task created"
      followUp: "follow-up"
      questionAnswered: "question answered"
      taskSettled: "task settled"
      recorded: "recorded"
    cancel: "Cancel"
    create: "Create page"
    save: "Save"
ru:
  knowledgeEditor:
    createTitle: "Новая страница"
    editTitle: "Редактировать страницу"
    hint: "Записи атрибутируются вам и остаются неподтверждёнными, пока человек их не проверит."
    title: "Заголовок"
    titlePlaceholder: "Короткий понятный заголовок"
    path: "Путь"
    pathPlaceholder: "architecture/overview"
    pathHint: "Путь в нижнем регистре через дефис; после создания не меняется."
    type: "Тип"
    typePlaceholder: "Note"
    status: "Статус"
    stable: "stable"
    draft: "draft"
    deprecated: "deprecated"
    staleAfter: "Устаревает после"
    description: "Описание"
    descriptionPlaceholder: "Одна строка, по которой читатель решает, открывать ли страницу"
    tags: "Теги"
    tagsPlaceholder: "decision, api, onboarding"
    body: "Текст"
    bodyPlaceholder: "Пишите страницу в Markdown. Ссылайтесь на другие страницы через [[path]]."
    decisions: "Решения, которые фиксирует страница"
    decisionsHint: "Связанное решение видно на странице и в журнале, чтобы читатель понимал, кто решил и почему."
    kind:
      taskCreated: "задача создана"
      followUp: "доп. промпт"
      questionAnswered: "ответ на вопрос"
      taskSettled: "прогон завершён"
      recorded: "записано"
    cancel: "Отмена"
    create: "Создать страницу"
    save: "Сохранить"
</i18n>
