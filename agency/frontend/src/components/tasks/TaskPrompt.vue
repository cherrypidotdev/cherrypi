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
.relative.flex.flex-col(:class="compact ? '' : 'min-h-0 flex-1'")
  .relative.flex.flex-col(
    :class="compact ? '' : 'min-h-0 flex-1 overflow-auto p-4'"
  )
    .flex.w-full.flex-col.gap-3
      .flex.flex-wrap.items-center.gap-2(v-if="showModelPicker")
        span.text-muted-foreground.text-xs(v-if="!agentId")
          | {{ $t("taskPrompt.assignAgentForModel") }}
        TaskModelPicker.ml-auto(
          :agent-id="agentId"
          :selection="modelSelection"
          @change="onModelChange"
          :disabled="savingModel || !agentId"
        )

      template(v-if="!editing")
        .flex.flex-col.gap-2
          .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm.transition-colors(
            class="bg-primary/10"
            :class="previewClass"
            :role="editable ? 'button' : undefined"
            :tabindex="editable ? 0 : undefined"
            :title="editable ? $t('taskPrompt.clickToEdit') : undefined"
            @click="startEdit()"
            @keydown.enter.prevent="startEdit()"
            @keydown.space.prevent="startEdit()"
            data-testid="prompt-preview"
          )
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :style="inheritProse"
              :text="value"
            )
          div(
            v-if="showAttachments && task.attachments.length"
            data-testid="prompt-attachments"
          )
            AttachmentList(
              compact
              :removable="editable"
              :attachments="task.attachments"
              @remove="emit('remove', $event)"
              @open="emit('openAttachment', $event)"
            )
        .text-muted-foreground.flex.items-center.gap-1.text-xs.select-none(
          v-if="hintText"
        )
          PencilIcon.size-3(v-if="editable")
          span {{ hintText }}

      template(v-else)
        .flex.flex-col.gap-2
          WritePreviewTabs(v-model="tab" test-id="prompt-tab")

          .relative(
            @dragenter.prevent="onDragEnter"
            @dragover.prevent
            @dragleave.prevent="onDragLeave"
            @drop.prevent="onDrop"
          )
            .border-primary.pointer-events-none.absolute.inset-0.z-20.flex.items-center.justify-center.rounded-lg.border-2.border-dashed(
              v-if="dragging"
              class="bg-background/80"
            )
              .flex.flex-col.items-center.gap-1.text-sm.font-medium
                PaperclipIcon.size-5
                span {{ $t("taskPrompt.dropFiles") }}
            MarkdownEditor(
              v-if="tab === 'write'"
              ref="editor"
              v-model="draft"
              v-model:resize-height="editorHeight"
              :placeholder="$t('taskPrompt.placeholder')"
              resizable
              references
              test-id="prompt-textarea"
              :aria-label="$t('taskPrompt.aria')"
              :title="$t('taskPrompt.shortcuts')"
              :reference-project-id="task.projectId"
              :reference-task-id="task.id"
              @submit="save"
              @paste="onPaste"
            )
            .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm(
              v-else
              data-testid="draft-preview"
            )
              MarkdownText.prose.prose-sm.max-w-none(
                class="prose-p:my-1 prose-pre:my-2"
                :style="inheritProse"
                :text="draft"
              )

          .flex.flex-wrap.items-center.gap-2(v-if="showAttachments")
            AttachmentUploadButton(
              :label="$t('taskPrompt.attach')"
              :title="$t('taskPrompt.attachAria')"
              :task-id="task.id"
              @uploaded="onUploaded"
            )

            Button(
              type="button"
              variant="outline"
              size="sm"
              :title="$t('taskPrompt.sketchAria')"
              @click="openNewSketch"
            )
              PencilIcon(data-icon="inline-start")
              span {{ $t("taskPrompt.sketch") }}

            span.text-muted-foreground.text-xs(
              v-if="!task.attachments.length && !uploading"
            ) {{ $t("taskPrompt.attachmentsHint") }}

            span.text-muted-foreground.text-xs(v-if="uploading")
              | {{ $t("taskPrompt.uploading") }}

          div(
            v-if="showAttachments && task.attachments.length"
            data-testid="prompt-attachments"
          )
            AttachmentList(
              removable
              sketchable
              compact
              :attachments="task.attachments"
              @remove="emit('remove', $event)"
              @sketch="openSketch"
              @open="emit('openAttachment', $event)"
            )

          .flex.items-center.gap-2
            span.text-muted-foreground.mr-auto.text-xs
              | {{ $t("taskPrompt.shortcutsShort") }}
            Button(
              type="button"
              variant="ghost"
              size="sm"
              @click="cancel"
              :disabled="saving"
            ) {{ $t("common.cancel") }}
            Button(
              type="button"
              size="sm"
              @click="save"
              :disabled="saving || !draft.trim()"
              data-testid="prompt-save"
            )
              Spinner(v-if="saving" data-icon="inline-start")
              CheckIcon(v-else data-icon="inline-start")
              span {{ $t("common.save") }}

          p.text-destructive.text-xs(v-if="error") {{ error }}

  SketchDialog(
    v-if="sketchMounted"
    v-model:open="sketchOpen"
    :title="sketchTitle"
    :snapshot="sketchSnapshot"
    :background-image="sketchBackground"
    :saving="sketchSaving"
    @save="onSketchSave"
  )
</template>

<script setup lang="ts">
import AttachmentList from "@/components/attachments/AttachmentList.vue";
import AttachmentUploadButton from "@/components/attachments/AttachmentUploadButton.vue";
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import WritePreviewTabs from "@/components/tasks/WritePreviewTabs.vue";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAttachmentUpload } from "@/composables/useAttachmentUpload";
import { useSketchEditor } from "@/composables/useSketchEditor";
import { agency } from "@/lib/sdk";
import {
  bustAttachmentCache,
  newSketchFilename,
  sketchSceneFilename,
  toStoredSnapshot,
} from "@/lib/sketches";
import type { TaskModelSelection } from "@/lib/task";
import { type Attachment, type Task } from "@cherrypi/sdk";
import { CheckIcon, PaperclipIcon, PencilIcon } from "@lucide/vue";
import { computed, defineAsyncComponent, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = withDefaults(
  defineProps<{
    task: Task;
    editable?: boolean;
    hint?: string;

    text?: string;

    field?: "prompt" | "pendingPrompt";

    compact?: boolean;

    agentId?: string;
  }>(),
  {
    editable: false,
    hint: "",
    text: undefined,
    field: "prompt",
    compact: false,
    agentId: "",
  },
);
const emit = defineEmits<{
  saved: [task: Task];

  changed: [];

  remove: [attachment: Attachment];

  openAttachment: [attachment: Attachment];
}>();

const value = computed(() => props.text ?? props.task.prompt);

const showAttachments = computed(() => props.field === "prompt");

const hintText = computed(
  () => props.hint || (props.editable ? t("taskPrompt.unsentHint") : ""),
);

const previewClass = computed(() => [
  props.editable
    ? "cursor-text border-dashed hover:border-primary/60"
    : "border-solid",
  props.compact ? "max-h-[40vh] overflow-y-auto" : "",
]);

const inheritProse = {
  "--tw-prose-body": "currentColor",
  "--tw-prose-headings": "currentColor",
  "--tw-prose-lead": "currentColor",
  "--tw-prose-links": "currentColor",
  "--tw-prose-bold": "currentColor",
  "--tw-prose-counters": "currentColor",
  "--tw-prose-bullets": "currentColor",
  "--tw-prose-hr": "currentColor",
  "--tw-prose-quotes": "currentColor",
  "--tw-prose-quote-borders": "currentColor",
  "--tw-prose-captions": "currentColor",
  "--tw-prose-code": "currentColor",
  "--tw-prose-pre-code": "currentColor",
  "--tw-prose-pre-bg": "transparent",
  "--tw-prose-th-borders": "currentColor",
  "--tw-prose-td-borders": "currentColor",
};

const showModelPicker = computed(
  () => props.editable && !props.compact && props.field === "prompt",
);
const modelSelection = ref<TaskModelSelection>(selectionFromTask(props.task));
const savingModel = ref(false);

function selectionFromTask(task: Task): TaskModelSelection {
  return {
    modelProvider: task.modelProvider,
    modelId: task.modelId,
    thinkingLevel: task.thinkingLevel,
  };
}

watch(
  () =>
    [
      props.task.modelProvider,
      props.task.modelId,
      props.task.thinkingLevel,
    ] as const,
  () => {
    if (!savingModel.value) {
      modelSelection.value = selectionFromTask(props.task);
    }
  },
);

async function onModelChange(next: TaskModelSelection) {
  if (savingModel.value) return;
  const previous = modelSelection.value;
  const modelChanged =
    next.modelProvider !== previous.modelProvider ||
    next.modelId !== previous.modelId;
  const levelChanged = next.thinkingLevel !== previous.thinkingLevel;
  if (!modelChanged && !levelChanged) return;
  modelSelection.value = next;
  savingModel.value = true;
  try {
    const updated = await agency.updateTask(props.task.id, {
      setModel: {
        provider: next.modelProvider,
        modelId: next.modelId,
        thinkingLevel: next.thinkingLevel,
      },
    });
    modelSelection.value = selectionFromTask(updated);

    emit("changed");
  } catch (err) {
    modelSelection.value = previous;
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingModel.value = false;
  }
}

const editing = ref(false);
const saving = ref(false);
const draft = ref("");
const error = ref("");
const editor = ref<InstanceType<typeof MarkdownEditor> | null>(null);

const editorHeight = ref<number | undefined>();
const tab = ref<"write" | "preview">("write");

const { uploading, upload } = useAttachmentUpload(() => ({
  taskId: props.task.id,
}));

const SketchDialog = defineAsyncComponent(
  () => import("@/components/sketches/SketchDialog.vue"),
);
const {
  open: sketchOpen,
  mounted: sketchMounted,
  saving: sketchSaving,
  snapshot: sketchSnapshot,
  background: sketchBackground,
  title: sketchTitle,
  openNew: openNewSketch,
  openImage: openSketch,
  onSave: onSketchSave,
} = useSketchEditor({
  siblings: () => props.task.attachments,
  persist: async ({ png, snapshot, target }) => {
    const sceneBlob = new Blob([toStoredSnapshot(snapshot)], {
      type: "application/json",
    });
    const { image, scene } = target;
    if (image && scene) {
      const updatedImage = await agency.replaceAttachment(image.id, {
        file: png,
        filename: image.filename,
      });
      const updatedScene = await agency.replaceAttachment(scene.id, {
        file: sceneBlob,
        filename: scene.filename,
      });
      replaceTaskAttachment(bustAttachmentCache(updatedImage));
      replaceTaskAttachment(bustAttachmentCache(updatedScene));
    } else {
      const uploadedImage = await agency.uploadAttachment({
        file: png,
        filename: newSketchFilename(),
        taskId: props.task.id,
      });
      await agency.uploadAttachment({
        file: sceneBlob,
        filename: sketchSceneFilename(uploadedImage.id),
        taskId: props.task.id,
      });
    }
    emit("changed");
  },
});

function replaceTaskAttachment(updated: Attachment) {
  props.task.attachments = props.task.attachments.map((item) =>
    item.id === updated.id ? updated : item,
  );
}
const dragging = ref(false);
let dragDepth = 0;

function startEdit() {
  if (!props.editable) return;
  draft.value = value.value;
  error.value = "";
  tab.value = "write";
  editing.value = true;
  void nextTick(() => editor.value?.focusEnd());
}

function cancel() {
  editing.value = false;
  error.value = "";
}

async function onPaste(event: ClipboardEvent) {
  if (!showAttachments.value) return;
  const files: File[] = [];
  for (const item of event.clipboardData?.items ?? []) {
    if (item.kind !== "file") continue;
    const file = item.getAsFile();
    if (file) files.push(file);
  }
  if (!files.length) return;
  event.preventDefault();
  await stage(files);
}

function hasFiles(event: DragEvent): boolean {
  return Array.from(event.dataTransfer?.types ?? []).includes(
    t("taskView.files"),
  );
}

function onDragEnter(event: DragEvent) {
  if (!hasFiles(event)) return;
  dragDepth += 1;
  dragging.value = true;
}

function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1);
  if (dragDepth === 0) dragging.value = false;
}

async function onDrop(event: DragEvent) {
  dragDepth = 0;
  dragging.value = false;
  if (!showAttachments.value) return;
  await stage(event.dataTransfer?.files ?? []);
}

async function stage(files: Iterable<File>) {
  const uploaded = await upload(files);
  if (uploaded.length) emit("changed");
}

function onUploaded() {
  emit("changed");
}

async function save() {
  const prompt = draft.value.trim();
  if (!prompt || saving.value) return;
  const current = props.task;
  saving.value = true;
  error.value = "";
  try {
    let next: Task;
    if (props.field === "pendingPrompt") {
      next = await agency.updateTask(current.id, { setPendingPrompt: prompt });
    } else {
      next = await agency.updateTask(current.id, { setPrompt: prompt });
    }
    editing.value = false;
    emit("saved", next);
    toast.success(t("taskPrompt.toasts.updated"));
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  taskPrompt:
    dropFiles: "Drop files to attach"
    attach: "Attach"
    sketch: "Sketch"
    aria: "Task prompt"
    attachAria: "Attach files to the task"
    sketchAria: "Draw a sketch"
    assignAgentForModel: "Assign an agent to choose a model"
    attachmentsHint: "Paste, drop, or pick files to attach them to the prompt"
    uploading: "Uploading…"
    shortcutsShort: "Ctrl+Enter to save · Tab indents"
    placeholder: "What should the agent do? Markdown supported."
    shortcuts: "Tab indents · Ctrl+Enter saves · Ctrl+K deletes to the end of the line · {'@'} mentions · # tasks"
    clickToEdit: "Click to edit the prompt"
    toasts:
      updated: "Prompt updated"
    unsentHint: "Unsent · click to edit"
ru:
  taskPrompt:
    dropFiles: "Перетащите файлы, чтобы прикрепить"
    attach: "Прикрепить"
    sketch: "Набросок"
    aria: "Промпт задачи"
    attachAria: "Прикрепить файлы к задаче"
    sketchAria: "Нарисовать набросок"
    assignAgentForModel: "Назначьте агента, чтобы выбрать модель"
    attachmentsHint: "Вставьте, перетащите или выберите файлы, чтобы прикрепить их к промпту"
    uploading: "Загрузка…"
    shortcutsShort: "Ctrl+Enter — сохранить · Tab — отступ"
    placeholder: "Что должен сделать агент? Поддерживается Markdown."
    shortcuts: "Tab — отступ · Ctrl+Enter — сохранить · Ctrl+K — удалить до конца строки · {'@'} — упоминания · # — задачи"
    clickToEdit: "Нажмите, чтобы изменить промпт"
    toasts:
      updated: "Промпт обновлён"
    unsentHint: "Не отправлено · нажмите, чтобы изменить"
</i18n>
