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
      DialogTitle {{ bit ? $t("knowledgeEditor.editTitle") : $t("knowledgeEditor.createTitle") }}
      DialogDescription {{ $t("knowledgeEditor.hint") }}
    form.flex.flex-col.gap-4(@submit.prevent="save")
      Field
        FieldLabel(for="knowledge-title") {{ $t("knowledgeEditor.title") }}
        Input#knowledge-title(
          v-model="title"
          :placeholder="$t('knowledgeEditor.titlePlaceholder')"
          autocomplete="off"
        )
        FieldDescription {{ $t("knowledgeEditor.titleHint") }}
      Field
        FieldLabel(for="knowledge-content") {{ $t("knowledgeEditor.content") }}
        Textarea#knowledge-content(
          v-model="content"
          :placeholder="$t('knowledgeEditor.contentPlaceholder')"
          rows="8"
        )
      Field
        FieldLabel(for="knowledge-tags") {{ $t("knowledgeEditor.tags") }}
        Input#knowledge-tags(
          v-model="tags"
          :placeholder="$t('knowledgeEditor.tagsPlaceholder')"
          autocomplete="off"
        )
        FieldDescription {{ $t("knowledgeEditor.tagsHint") }}
      Field
        FieldLabel {{ $t("knowledgeEditor.sources") }}
        FieldDescription {{ $t("knowledgeEditor.sourcesHint") }}
        .flex.flex-col.gap-2
          .grid.gap-2(
            v-for="(source, index) in sources"
            :key="index"
            class="sm:grid-cols-5"
          )
            select.border-input.w-full.rounded-lg.border.bg-transparent.px-2.py-1.text-sm(
              v-model="source.kind"
            )
              option(value="task_prompt") {{ $t("knowledgeEditor.kind.taskPrompt") }}
              option(value="follow_up") {{ $t("knowledgeEditor.kind.followUp") }}
              option(value="questionnaire") {{ $t("knowledgeEditor.kind.questionnaire") }}
              option(value="system_prompt") {{ $t("knowledgeEditor.kind.systemPrompt") }}
            Input(
              v-if="source.kind === 'task_prompt' || source.kind === 'follow_up' || source.kind === 'questionnaire'"
              v-model="source.taskId"
              :placeholder="$t('knowledgeEditor.taskId')"
            )
            Input(
              v-if="source.kind === 'follow_up'"
              v-model="source.messageId"
              :placeholder="$t('knowledgeEditor.messageId')"
            )
            Input(
              v-if="source.kind === 'questionnaire'"
              v-model="source.requestId"
              :placeholder="$t('knowledgeEditor.requestId')"
            )
            Input(
              v-if="source.kind === 'questionnaire'"
              v-model="source.questionId"
              :placeholder="$t('knowledgeEditor.questionId')"
            )
            Input(
              v-if="source.kind === 'system_prompt'"
              v-model="source.promptId"
              :placeholder="$t('knowledgeEditor.promptId')"
            )
            Button(
              type="button"
              variant="ghost"
              size="sm"
              @click="sources.splice(index, 1)"
            )
              Trash2Icon(data-icon="inline-start")
        Button(type="button" variant="outline" size="sm" @click="addSource")
          PlusIcon(data-icon="inline-start")
          span {{ $t("knowledgeEditor.addSource") }}
      .grid.gap-4(class="sm:grid-cols-2")
        Field
          FieldLabel(for="knowledge-related") {{ $t("knowledgeEditor.related") }}
          Input#knowledge-related(
            v-model="related"
            :placeholder="$t('knowledgeEditor.idsPlaceholder')"
            autocomplete="off"
          )
        Field
          FieldLabel(for="knowledge-supersedes") {{ $t("knowledgeEditor.supersedes") }}
          Input#knowledge-supersedes(
            v-model="supersedes"
            :placeholder="$t('knowledgeEditor.idsPlaceholder')"
            autocomplete="off"
          )
      p.text-destructive.text-sm(v-if="error") {{ error }}
      DialogFooter
        DialogClose
          Button(type="button" variant="ghost") {{ $t("knowledgeEditor.cancel") }}
        Button(type="submit" :disabled="saving || !content.trim()")
          Spinner(v-if="saving" data-icon="inline-start")
          span {{ bit ? $t("knowledgeEditor.save") : $t("knowledgeEditor.create") }}
</template>

<script setup lang="ts">
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
import { Textarea } from "@/components/ui/textarea";
import { agency } from "@/lib/sdk";
import type { KnowledgeBit, KnowledgeSourceRef } from "@cherrypi/sdk";
import { PlusIcon, Trash2Icon } from "@lucide/vue";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  open: boolean;

  projectId?: string;
  bit?: KnowledgeBit;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  saved: [bit: KnowledgeBit];
}>();

const { t } = useI18n();
const title = ref("");
const content = ref("");
const tags = ref("");
const related = ref("");
const supersedes = ref("");
const sources = ref<SourceRow[]>([]);
const saving = ref(false);
const error = ref("");

interface SourceRow {
  kind: string;
  taskId: string;
  messageId: string;
  requestId: string;
  questionId: string;
  promptId: string;
}

function addSource() {
  sources.value.push({
    kind: "task_prompt",
    taskId: "",
    messageId: "",
    requestId: "",
    questionId: "",
    promptId: "",
  });
}

function reset() {
  const bit = props.bit;
  title.value = bit?.title ?? "";
  content.value = bit?.content ?? "";
  tags.value = (bit?.tags ?? []).join(", ");
  related.value = (bit?.links ?? [])
    .filter((link) => link.kind === "related")
    .map((link) => link.knowledgeId)
    .join(", ");
  supersedes.value = (bit?.links ?? [])
    .filter((link) => link.kind === "supersedes")
    .map((link) => link.knowledgeId)
    .join(", ");
  sources.value = (bit?.sources ?? []).map((source) => ({
    kind: source.kind,
    taskId: source.taskId,
    messageId: source.messageId,
    requestId: source.requestId,
    questionId: source.questionId,
    promptId: source.promptId ? String(source.promptId) : "",
  }));
  if (!sources.value.length) addSource();
  error.value = "";
}

watch(
  () => [props.open, props.bit],
  () => {
    if (props.open) reset();
  },
  { immediate: true },
);

function splitList(value: string): string[] {
  return value
    .split(/[\s,]+/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function sourceRefs(): KnowledgeSourceRef[] {
  return sources.value
    .filter(
      (source) =>
        source.kind &&
        (source.taskId.trim() ||
          source.messageId.trim() ||
          source.requestId.trim() ||
          source.questionId.trim() ||
          source.promptId.trim()),
    )
    .map((source) => ({
      kind: source.kind,
      taskId: source.taskId.trim() || undefined,
      messageId: source.messageId.trim() || undefined,
      requestId: source.requestId.trim() || undefined,
      questionId: source.questionId.trim() || undefined,
      promptId: source.promptId.trim() ? Number(source.promptId) : undefined,
    }));
}

async function save() {
  saving.value = true;
  error.value = "";
  try {
    let saved: KnowledgeBit;
    if (props.bit) {
      saved = await agency.updateKnowledge({
        id: props.bit.id,
        baseRevision: props.bit.revision,
        title: title.value.trim(),
        content: content.value,
        tags: splitList(tags.value),
        sources: sourceRefs(),
        relatedIds: splitList(related.value),
        supersedesIds: splitList(supersedes.value),
        summary: "knowledge edit",
      });
    } else {
      saved = await agency.addKnowledge({
        projectId: props.projectId,
        title: title.value.trim(),
        content: content.value,
        tags: splitList(tags.value),
        sources: sourceRefs(),
        relatedIds: splitList(related.value),
        supersedesIds: splitList(supersedes.value),
      });
    }
    emit("saved", saved);
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

void t;
</script>

<i18n lang="yaml">
en:
  knowledgeEditor:
    createTitle: "New knowledge"
    editTitle: "Edit knowledge"
    hint: "A bit is a short, attributed fact with hard links to the inputs it came from. A human's bit is approved at once; an edit after approval returns it to review."
    title: "Title"
    titlePlaceholder: "A short, human title"
    titleHint: "Optional; the first line of the content is used when empty."
    content: "Knowledge"
    contentPlaceholder: "What is true, in one or two short paragraphs of markdown."
    tags: "Tags"
    tagsPlaceholder: "deploy, ci-cd, scheduler"
    tagsHint: "Lower-case words with dashes or underscores, separated by commas."
    sources: "Sources"
    sourcesHint: "Hard links to the real inputs. A questionnaire answer needs the task, the dialog request id and the question id."
    addSource: "Add source"
    kind:
      taskPrompt: "task prompt"
      followUp: "follow-up"
      questionnaire: "questionnaire"
      systemPrompt: "system prompt"
    taskId: "task id"
    messageId: "message id"
    requestId: "request id"
    questionId: "question id"
    promptId: "prompt version id"
    related: "Related knowledge ids"
    supersedes: "Supersedes knowledge ids"
    idsPlaceholder: "Comma-separated ids"
    cancel: "Cancel"
    create: "Record"
    save: "Save"
ru:
  knowledgeEditor:
    createTitle: "Новый факт"
    editTitle: "Редактировать факт"
    hint: "Факт — это короткое атрибутированное знание с жёсткими ссылками на исходные данные. Факт человека подтверждается сразу; правка после подтверждения возвращает его на проверку."
    title: "Заголовок"
    titlePlaceholder: "Короткий понятный заголовок"
    titleHint: "Необязательно; если пусто, берётся первая строка текста."
    content: "Знание"
    contentPlaceholder: "Что верно, в одном-двух коротких абзацах markdown."
    tags: "Теги"
    tagsPlaceholder: "deploy, ci-cd, scheduler"
    tagsHint: "Слова в нижнем регистре с дефисами или подчёркиваниями, через запятую."
    sources: "Источники"
    sourcesHint: "Жёсткие ссылки на реальные входные данные. Для ответа на опрос нужны задача, id запроса диалога и id вопроса."
    addSource: "Добавить источник"
    kind:
      taskPrompt: "промпт задачи"
      followUp: "доп. промпт"
      questionnaire: "опрос"
      systemPrompt: "системный промпт"
    taskId: "id задачи"
    messageId: "id сообщения"
    requestId: "id запроса"
    questionId: "id вопроса"
    promptId: "id версии промпта"
    related: "Связанные факты"
    supersedes: "Заменяет факты"
    idsPlaceholder: "id через запятую"
    cancel: "Отмена"
    create: "Записать"
    save: "Сохранить"
</i18n>
