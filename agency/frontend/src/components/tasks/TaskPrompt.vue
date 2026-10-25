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
      template(v-if="!editing")
        .flex.flex-col.gap-2
          .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm.transition-colors(
            class="bg-primary/10"
            :class="previewClass"
            :role="editable ? 'button' : undefined"
            :tabindex="editable ? 0 : undefined"
            :title="editable ? 'Click to edit the prompt' : undefined"
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
                span Drop files to attach
            Textarea.min-h-40(
              v-if="tab === 'write'"
              ref="editor"
              v-model="draft"
              placeholder="What should the agent do? Markdown supported."
              @keydown.enter="onKeydown"
              @paste="onPaste"
              data-testid="prompt-textarea"
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
              label="Attach"
              title="Attach files to the task"
              :task-id="task.id"
              @uploaded="onUploaded"
            )
            Button(
              type="button"
              variant="outline"
              size="sm"
              title="Draw a sketch"
              @click="openNewSketch"
            )
              PencilIcon(data-icon="inline-start")
              span Sketch
            span.text-muted-foreground.text-xs(
              v-if="!task.attachments.length && !uploading"
            ) Paste, drop, or pick files to attach them to the prompt
            span.text-muted-foreground.text-xs(v-if="uploading")
              | Uploading…
          div(
            v-if="showAttachments && task.attachments.length"
            data-testid="prompt-attachments"
          )
            AttachmentList(
              removable
              :attachments="task.attachments"
              @remove="emit('remove', $event)"
              @open="emit('openAttachment', $event)"
            )
          .flex.items-center.gap-2
            span.text-muted-foreground.mr-auto.text-xs Ctrl+Enter to save
            Button(
              type="button"
              variant="ghost"
              size="sm"
              @click="cancel"
              :disabled="saving"
            ) Cancel
            Button(
              type="button"
              size="sm"
              @click="save"
              :disabled="saving || !draft.trim()"
              data-testid="prompt-save"
            )
              Spinner(v-if="saving" data-icon="inline-start")
              CheckIcon(v-else data-icon="inline-start")
              span Save
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
import MarkdownText from "@/components/MarkdownText.vue";
import WritePreviewTabs from "@/components/tasks/WritePreviewTabs.vue";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { useAttachmentUpload } from "@/composables/useAttachmentUpload";
import { useSketchEditor } from "@/composables/useSketchEditor";
import { agency } from "@/lib/sdk";
import {
  newSketchFilename,
  sketchSceneFilename,
  toStoredSnapshot,
} from "@/lib/sketches";
import {
  type Attachment,
  type Task,
  TaskState,
  type UpdateTaskInput,
} from "@cherrypi/sdk";
import { CheckIcon, PaperclipIcon, PencilIcon } from "@lucide/vue";
import { computed, defineAsyncComponent, nextTick, ref } from "vue";
import { toast } from "vue3-toastify";

const props = withDefaults(
  defineProps<{
    task: Task;
    editable?: boolean;
    hint?: string;

    text?: string;

    field?: "prompt" | "pendingPrompt";

    compact?: boolean;
  }>(),
  {
    editable: false,
    hint: "",
    text: undefined,
    field: "prompt",
    compact: false,
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
  () => props.hint || (props.editable ? "Unsent · click to edit" : ""),
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

const editing = ref(false);
const saving = ref(false);
const draft = ref("");
const error = ref("");
const editor = ref<{ $el?: HTMLTextAreaElement } | undefined>();
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
  onSave: onSketchSave,
} = useSketchEditor({
  siblings: () => props.task.attachments,
  persist: async ({ png, snapshot }) => {
    const image = await agency.uploadAttachment({
      file: png,
      filename: newSketchFilename(),
      taskId: props.task.id,
    });
    const scene = new Blob([toStoredSnapshot(snapshot)], {
      type: "application/json",
    });
    await agency.uploadAttachment({
      file: scene,
      filename: sketchSceneFilename(image.id),
      taskId: props.task.id,
    });
    emit("changed");
  },
});
const dragging = ref(false);
let dragDepth = 0;

function startEdit() {
  if (!props.editable) return;
  draft.value = value.value;
  error.value = "";
  tab.value = "write";
  editing.value = true;
  void nextTick(() => {
    const el = editor.value?.$el;
    el?.focus();
    el?.setSelectionRange(el.value.length, el.value.length);
  });
}

function cancel() {
  editing.value = false;
  error.value = "";
}

// prompt can be multi-line without a modifier.
function onKeydown(event: KeyboardEvent) {
  if (event.key !== "Enter" || event.isComposing) return;
  if (!event.ctrlKey && !event.metaKey) return;
  event.preventDefault();
  void save();
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
  return Array.from(event.dataTransfer?.types ?? []).includes("Files");
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
    } else if (current.state === TaskState.BACKLOG) {
      next = await agency.updateTask(current.id, { setPrompt: prompt });
    } else {
      const input: UpdateTaskInput = {
        prompt,
        name: current.name,
        autoTitle: current.autoTitle,
        workspace: current.workspace,
        priority: current.priority,
        size: current.size,
        projectId: current.projectId,
        assigneeActorId: current.assigneeActorId,
        tagIds: current.tags.map((tag) => tag.id),
        modelProvider: current.modelProvider,
        modelId: current.modelId,
        thinkingLevel: current.thinkingLevel,
      };
      next = await agency.updateTask(current.id, input);
    }
    editing.value = false;
    emit("saved", next);
    toast.success("Prompt updated");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>
