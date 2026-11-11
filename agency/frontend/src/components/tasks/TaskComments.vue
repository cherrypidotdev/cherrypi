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
.flex.flex-col.gap-3
  .flex.items-center.justify-between
    span.text-muted-foreground.text-xs {{ $t("comments.title") }}
    span.text-muted-foreground.text-xs(v-if="unread")
      | {{ $t("comments.unread", { count: unread }) }}

  p.text-muted-foreground.text-xs(v-if="loading") {{ $t("common.loading") }}
  p.text-muted-foreground.text-xs(
    v-else-if="!comments.length"
    data-testid="comments-empty"
  )
    | {{ $t("comments.empty") }}

  .flex.flex-col.gap-2
    .group.rounded-md.border.p-2(
      v-for="comment in comments"
      :key="String(comment.id)"
      :data-testid="`comment-${comment.id}`"
    )
      template(v-if="comment.deleted")
        .text-muted-foreground.flex.items-center.gap-2
          Trash2Icon.size-3.shrink-0
          span.text-xs
            | {{ $t("comments.deletedBy", { actor: deleterName(comment) }) }}
          span.text-muted-foreground.ml-auto.text-xs(
            :title="formatTimestamp(comment.deletedAt)"
          )
            | {{ formatRelativeTimestamp(comment.deletedAt) }}
      template(v-else)
        .flex.items-start.gap-2
          .flex.min-w-0.flex-1.flex-col.gap-1
            .flex.flex-wrap.items-center.gap-1
              MessageSender(
                :sender="senderFor(comment)"
                :picture="pictureFor(senderFor(comment))"
              )
              span.text-muted-foreground.text-xs(
                :title="formatTimestamp(comment.createdAt)"
              )
                | {{ formatRelativeTimestamp(comment.createdAt) }}
              span.text-muted-foreground.text-xs(v-if="comment.edited")
                | {{ $t("comments.edited") }}
            MarkdownText.prose.prose-sm.max-w-none.break-words(
              class="prose-p:my-1 prose-pre:my-2"
              :text="comment.body"
            )
            AttachmentChips(
              v-if="comment.attachments.length"
              :attachments="comment.attachments"
              @open="emit('openAttachment', $event)"
            )
          .text-muted-foreground.flex.shrink-0.items-center(
            v-if="canModify(comment)"
            class="gap-0.5"
          )
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              :title="$t('comments.edit')"
              :aria-label="$t('comments.editAria', { id: comment.id })"
              @click="startEdit(comment)"
              :disabled="saving"
            )
              PencilIcon.size-3
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              :title="$t('comments.delete')"
              :aria-label="$t('comments.deleteAria', { id: comment.id })"
              @click="remove(comment)"
              :disabled="saving"
            )
              Trash2Icon.size-3

        form.mt-2.flex.flex-col.gap-2(
          v-if="editingId === comment.id"
          @submit.prevent="saveEdit(comment)"
          data-testid="comment-edit-form"
        )
          MarkdownEditor(
            v-model="editDraft"
            :placeholder="$t('comments.editPlaceholder')"
            references
            test-id="comment-edit-body"
            :aria-label="$t('comments.editAriaLabel')"
            :reference-project-id="projectId"
            :reference-task-id="taskId"
            height="7rem"
            @submit="saveEdit(comment)"
          )
          .flex.items-center.gap-2
            Button.ml-auto(
              type="button"
              variant="ghost"
              size="sm"
              @click="cancelEdit"
            ) {{ $t("common.cancel") }}
            Button(type="submit" size="sm" :disabled="!canSaveEdit")
              | {{ $t("common.save") }}

  form.flex.flex-col.gap-2.rounded-md.border.p-2(
    @submit.prevent="post"
    data-testid="comment-composer"
  )
    MarkdownEditor(
      v-model="draft"
      :placeholder="$t('comments.placeholder')"
      references
      test-id="comment-body"
      :aria-label="$t('comments.bodyAria')"
      :reference-project-id="projectId"
      :reference-task-id="taskId"
      height="8rem"
      @submit="post"
    )
    .flex.flex-col.gap-1(v-if="staged.length")
      .flex.items-center.gap-1.rounded-md.border.px-2.py-1.text-xs(
        v-for="file in staged"
        :key="String(file.id)"
      )
        FileIcon.text-muted-foreground.size-3.shrink-0
        span.min-w-0.flex-1.truncate(:title="file.filename") {{ file.filename }}
        Button(
          type="button"
          variant="ghost"
          size="icon-sm"
          :title="$t('comments.removeAttachment')"
          :aria-label="$t('comments.removeAttachmentAria', { filename: file.filename })"
          @click="unstage(file)"
        )
          XIcon.size-3
    .flex.items-center.gap-2
      AttachmentUploadButton(
        :label="$t('taskPrompt.attach')"
        comment
        :title="$t('comments.attachAria')"
        :task-id="taskId"
        @uploaded="stage"
      )
      Button.ml-auto(
        type="submit"
        size="sm"
        :disabled="!canPost"
        data-testid="comment-post"
      )
        SendIcon.size-3(data-icon="inline-start")
        span {{ $t("comments.post") }}
</template>

<script setup lang="ts">
import AttachmentChips from "@/components/attachments/AttachmentChips.vue";
import AttachmentUploadButton from "@/components/attachments/AttachmentUploadButton.vue";
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import { Button } from "@/components/ui/button";
import { useActorPictures } from "@/composables/useActorPictures";
import { useRealtime } from "@/composables/useRealtime";
import { currentAccount, ensureSession } from "@/lib/account";
import type { MessageSender as Sender } from "@/lib/console";
import { formatRelativeTimestamp, formatTimestamp } from "@/lib/format";
import { agent as agencyFacade } from "@/lib/sdk";
import type { Attachment, TaskComment } from "@cherrypi/sdk";
import { FileIcon, PencilIcon, SendIcon, Trash2Icon, XIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { toast } from "vue3-toastify";

const props = withDefaults(
  defineProps<{ taskId: string; projectId?: string }>(),
  { projectId: "" },
);
const emit = defineEmits<{
  read: [];
  openAttachment: [attachment: Attachment];
}>();

const comments = ref<TaskComment[]>([]);
const unread = ref(0);
const loading = ref(true);
const draft = ref("");
const staged = ref<Attachment[]>([]);
const editingId = ref<bigint | null>(null);
const editDraft = ref("");
const saving = ref(false);

const { pictureFor } = useActorPictures();

const canPost = computed(() => draft.value.trim().length > 0 && !saving.value);
const canSaveEdit = computed(
  () => editDraft.value.trim().length > 0 && !saving.value,
);

function senderFor(comment: TaskComment): Sender {
  return {
    id: comment.authorActorId,
    username: comment.authorUsername,
    name: comment.authorName,
    kind: comment.authorKind || "user",
  };
}

function deleterName(comment: TaskComment): string {
  return (
    comment.deletedByName || comment.deletedByUsername || "an administrator"
  );
}

function canModify(comment: TaskComment): boolean {
  if (comment.deleted) return false;
  const me = currentAccount.value;
  if (!me) return false;
  if (me.is_admin) return true;
  return String(me.id) === comment.authorActorId;
}

async function load() {
  try {
    const result = await agencyFacade.listTaskComments(props.taskId);
    comments.value = result.comments;
    unread.value = result.unread;

    emit("read");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    loading.value = false;
  }
}

async function post() {
  const body = draft.value.trim();
  if (!body || saving.value) return;
  saving.value = true;
  try {
    await agencyFacade.addTaskComment({
      taskId: props.taskId,
      body,
      attachmentIds: staged.value.map((file) => file.id),
    });
    draft.value = "";
    staged.value = [];
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}

function stage(file: Attachment) {
  staged.value = [...staged.value, file];
}

function unstage(file: Attachment) {
  staged.value = staged.value.filter((item) => item.id !== file.id);
}

function startEdit(comment: TaskComment) {
  editingId.value = comment.id;
  editDraft.value = comment.body;
}

function cancelEdit() {
  editingId.value = null;
  editDraft.value = "";
}

async function saveEdit(comment: TaskComment) {
  const body = editDraft.value.trim();
  if (!body || saving.value) return;
  saving.value = true;
  try {
    await agencyFacade.updateTaskComment(comment.id, body);
    cancelEdit();
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}

async function remove(comment: TaskComment) {
  saving.value = true;
  try {
    await agencyFacade.deleteTaskComment(comment.id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void ensureSession();
  void load();
});

useRealtime(
  ["tasks"],
  (event) => {
    if (!event.task || event.task.id === props.taskId) void load();
  },
  { rows: true },
);
</script>

<i18n lang="yaml">
en:
  comments:
    title: "Comments"
    empty: "No comments yet. Discuss the task here — comments are never sent to Pi."
    edited: "· edited"
    post: "Post"
    edit: "Edit comment"
    delete: "Delete comment"
    editAriaLabel: "Edit comment body"
    bodyAria: "Comment body"
    removeAttachment: "Remove attachment"
    attachAria: "Attach a file to this comment"
    editAria: "Edit comment {id}"
    deleteAria: "Delete comment {id}"
    editPlaceholder: "Edit the comment… Type {'@'} to mention someone or # to link a task."
    placeholder: "Write a comment… Type {'@'} to mention someone or # to link a task."
    removeAttachmentAria: "Remove {filename}"
    unread: "{count} new"
    deletedBy: "Comment deleted by {actor}"
ru:
  comments:
    title: "Комментарии"
    empty: "Комментариев пока нет. Обсуждайте задачу здесь — комментарии никогда не отправляются в Pi."
    edited: "· изменён"
    post: "Отправить"
    edit: "Редактировать комментарий"
    delete: "Удалить комментарий"
    editAriaLabel: "Текст комментария для редактирования"
    bodyAria: "Текст комментария"
    removeAttachment: "Удалить вложение"
    attachAria: "Прикрепить файл к этому комментарию"
    editAria: "Редактировать комментарий {id}"
    deleteAria: "Удалить комментарий {id}"
    editPlaceholder: "Измените комментарий… Введите {'@'}, чтобы упомянуть кого-то, или #, чтобы сослаться на задачу."
    placeholder: "Напишите комментарий… Введите {'@'}, чтобы упомянуть кого-то, или #, чтобы сослаться на задачу."
    removeAttachmentAria: "Удалить {filename}"
    unread: "новых: {count}"
    deletedBy: "Комментарий удалён: {actor}"
</i18n>
