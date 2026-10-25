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
.relative.flex.h-full.flex-col
  header.shrink-0.border-b(:class="detailsOpen ? 'mr-80' : ''")
    .flex.items-center.gap-3.px-4.py-2
      Button(variant="ghost" size="sm" @click="back")
        ArrowLeftIcon(data-icon="inline-start")
        span Tasks
      .flex.min-w-0.flex-1.items-center.gap-2
        Input#task-title-input.h-8.max-w-md(
          v-if="task && titleEditing"
          ref="titleInput"
          v-model="titleDraft"
          placeholder="Task title"
          @keydown.enter.prevent="saveTitle"
          @keydown.esc="cancelTitleEdit"
          @blur="saveTitle"
        )
        template(v-else)
          Skeleton.h-4.w-48(v-if="!task")
          button.group.-mx-1.flex.min-w-0.items-center.gap-1.rounded-md.px-1.text-left.transition-colors.outline-none(
            v-else
            type="button"
            title="Click to edit the title"
            class="hover:bg-muted/60 focus-visible:ring-ring/50 focus-visible:ring-2"
            @click="startTitleEdit"
          )
            Skeleton.h-4.w-48(v-if="taskTitlePending(task)")
            span.truncate.text-sm(v-else :class="taskNameClass(task)") {{ taskName(task) }}
            PencilIcon.size-3.shrink-0.opacity-0.transition-opacity(
              v-if="!taskTitlePending(task)"
              class="group-hover:opacity-60"
            )
        RouterLink.text-muted-foreground.text-xs(
          v-if="task?.projectId && projectSlug(task.projectId)"
          class="hover:text-foreground"
          :to="{ name: 'project', params: { slug: projectSlug(task.projectId) } }"
        ) {{ projectName(task.projectId) }}
        span.text-muted-foreground.inline-flex.shrink-0.items-center.gap-1.rounded-full.border.px-2.text-xs(
          v-if="sessionDeleted"
          class="py-0.5"
          title="The agent's Pi session and workspace were cleaned up; the conversation is stored in the agency and a follow-up starts a new Pi session"
        )
          CircleSlashIcon.size-3.shrink-0
          | Session cleaned up
      .flex.shrink-0.items-center.gap-2(v-if="task")
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          class="gap-1.5 py-0.5"
          :class="taskStateClassFor(task)"
        )
          component.size-3.shrink-0(:is="taskStateIcon(task.state)")
          | {{ taskStateLabelFor(task) }}
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          class="py-0.5"
          :class="taskPriorityClass(task.priority)"
          :title="`Task priority: ${taskPriorityLabel(task.priority)}`"
        ) {{ taskPriorityLabel(task.priority, true) }}
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          class="py-0.5"
          :class="taskSizeClass(task.size)"
          :title="`Task size: ${taskSizeLabel(task.size)}`"
        ) {{ taskSizeLabel(task.size, true) }}
        Button(
          variant="ghost"
          size="icon-sm"
          aria-controls="task-details"
          :title="detailsOpen ? 'Hide task details' : 'Show task details'"
          :aria-expanded="detailsOpen"
          @click="toggleDetails"
        )
          MenuIcon.size-4

  .flex.min-h-0.flex-1(:class="detailsOpen ? 'mr-80' : ''")
    .flex.min-w-0.flex-1.flex-col
      .shrink-0.border-b.px-4.py-2(
        v-if="task && task.state === TaskState.FAILED"
        class="bg-destructive/10"
      )
        .mx-auto.flex.max-w-3xl.items-start.gap-3
          TriangleAlertIcon.text-destructive.size-4.shrink-0(class="mt-0.5")
          .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
            p.text-destructive.text-sm.font-medium The task failed.
            p.text-muted-foreground.text-xs.break-words {{ task.error || "No error details." }}
          Button.shrink-0(
            v-if="!sessionDeleted"
            variant="outline"
            size="sm"
            @click="reopen(true)"
          )
            PlayIcon(data-icon="inline-start")
            span Re-queue

      TaskPrompt.flex-1.overflow-hidden(
        v-if="task && showPrompt"
        :hint="promptHint"
        :task="task"
        :editable="canEditPrompt"
        @saved="onDraftSaved"
        @changed="onAttachmentUploaded"
        @remove="removeAttachment"
        @open-attachment="openAttachment"
      )
      .flex.flex-1.items-center.justify-center.p-6(
        v-else-if="task && !isStarted"
      )
        p.text-muted-foreground.max-w-md.text-center.text-sm
          | This task has no console yet. Its Pi session starts when the scheduler
          | finds an eligible agent with a free slot.
      .relative.flex.min-h-0.flex-1.flex-col.overflow-hidden(
        v-else-if="task"
        @dragenter.prevent="onDragEnter"
        @dragover.prevent
        @dragleave.prevent="onDragLeave"
        @drop.prevent="onDrop"
        @paste="onPaste"
        data-testid="console-dropzone"
      )
        .border-primary.pointer-events-none.absolute.inset-2.z-20.flex.items-center.justify-center.rounded-lg.border-2.border-dashed(
          v-if="draggingFiles && canFollowUp"
          class="bg-background/80"
        )
          .flex.flex-col.items-center.gap-1.text-sm.font-medium
            PaperclipIcon.size-5
            span Drop files to attach
        TaskConsole.min-h-0.flex-1(
          :key="task.sessionId"
          :id="id"
          :busy="busy"
          :waiting-for-input="task?.runtime?.waitingForInput"
          :attachments="task?.attachments ?? []"
          :actor-pictures="actorPictures"
          @queue="onQueue"
          @open-attachment="openAttachment"
        )
        .shrink-0.px-4.pt-3.pb-3(v-if="showPendingPrompt")
          .mx-auto.w-full.max-w-3xl
            TaskPrompt(
              field="pendingPrompt"
              compact
              :hint="pendingPromptHint"
              :task="task"
              :editable="canEditPrompt"
              :text="task.pendingPrompt"
              @saved="onPendingPromptSaved"
            )
        .shrink-0.border-t.px-4.pt-3(v-if="taskQueue.length")
          .mx-auto.flex.max-w-3xl.flex-col.gap-1
            .flex.items-center.gap-2
              p.text-muted-foreground.text-xs
                | Queued steering — injected between blocks
              Button.ml-auto(
                variant="outline"
                size="sm"
                @click="clearQueue"
                :disabled="clearingQueue"
              )
                Spinner(v-if="clearingQueue" data-icon="inline-start")
                Trash2Icon(v-else data-icon="inline-start")
                span Clear queue
            .flex.flex-col.items-end.gap-1(
              v-for="item in taskQueue"
              :key="item.id"
              data-testid="steering-message"
            )
              MessageSender(
                v-if="item.sender"
                :sender="item.sender"
                :picture="pictureFor(item.sender)"
              )
              .rounded-lg.border.border-dashed.px-3.py-2(
                class="bg-muted/60 max-w-[80%]"
              )
                .text-sm.break-words.whitespace-pre-wrap(v-if="!item.text")
                MarkdownText.prose.prose-sm.max-w-none(
                  v-else
                  class="dark:prose-invert prose-p:my-1 prose-pre:my-2"
                  :text="item.text"
                )
              AttachmentChips(
                v-if="item.attachmentIds.length"
                :attachments="queueAttachments(item.attachmentIds)"
                @open="openAttachment"
              )
        form.relative.shrink-0.border-t.p-4(
          ref="composer"
          :class="canFollowUp ? '' : 'opacity-50'"
          @submit.prevent="send"
        )
          .absolute.inset-x-0.-top-1.z-10.flex.h-2.cursor-row-resize.items-center.justify-center(
            v-if="canFollowUp && tab === 'write'"
            title="Drag to resize the composer"
            @pointerdown="startResize"
            data-testid="composer-resize-handle"
          )
            .bg-border.h-1.w-10.rounded-full.transition-opacity(
              :class="resizing ? 'opacity-100' : 'opacity-40 hover:opacity-100'"
            )
          .mx-auto.flex.max-w-3xl.flex-col.gap-2
            p.text-muted-foreground.inline-flex.items-start.gap-1.text-xs(
              v-if="sessionDeleted"
              data-testid="session-cleaned-up-hint"
            )
              CircleSlashIcon.size-3.shrink-0(class="mt-0.5")
              span
                | The agent's Pi session and workspace were cleaned up. Sending a
                | message replays the stored conversation in a new Pi session.
            .flex.flex-col.gap-2
              .flex.flex-wrap.items-center.gap-2
                WritePreviewTabs(v-model="tab" test-id="composer-tab")
                .ml-auto.flex.flex-wrap.items-center.gap-2
                  TaskModelPicker(
                    v-if="task.assigneeActorId && !sessionDeleted"
                    :agent-id="agentIdForActor(task.assigneeActorId)"
                    :config="config"
                    :selection="modelSelection"
                    :use-defaults="false"
                    @change="onModelSelectionChange"
                    :disabled="saving.model || !config"
                  )
                  Button.shrink-0(
                    v-if="steer"
                    type="button"
                    variant="outline"
                    title="Stop the agent"
                    @click="stop"
                    :disabled="stopping"
                  )
                    Spinner(v-if="stopping" data-icon="inline-start")
                    SquareIcon(v-else data-icon="inline-start")
                    span Stop
                  Button.shrink-0(
                    type="submit"
                    :title="steer ? 'Steer: injected between the run\u2019s blocks' : 'Send prompt'"
                    :disabled="!canFollowUp || !prompt.trim() || sending"
                  )
                    SendIcon(data-icon="inline-start")
                    span {{ steer ? "Steer" : "Send" }}
              Textarea.field-sizing-content.min-h-9.w-full.resize-none(
                v-if="tab === 'write'"
                ref="promptInput"
                v-model="prompt"
                placeholder="Send a follow-up prompt…"
                :class="textareaHeight === null ? 'max-h-40' : ''"
                :style="textareaStyle"
                @keydown.enter.exact="onEnter"
                :disabled="!canFollowUp"
              )
              .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm(
                v-else
                data-testid="composer-preview"
              )
                MarkdownText.prose.prose-sm.max-w-none(
                  class="prose-p:my-1 prose-pre:my-2"
                  :style="inheritProse"
                  :text="prompt"
                )
              .flex.flex-col.gap-1(v-if="canFollowUp")
                .flex.items-center.gap-2
                  AttachmentUploadButton(
                    label="Attach"
                    :task-id="task.id"
                    @uploaded="onStagedUploaded"
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
                    v-if="!stagedAttachments.length"
                  ) Paste, drop, or pick files to send with this prompt
                AttachmentList(
                  v-if="stagedAttachments.length"
                  removable
                  sketchable
                  :attachments="stagedAttachments"
                  @remove="removeStaged"
                  @sketch="openSketch"
                  @open="openAttachment"
                )
              .flex.min-h-7.items-center.gap-2(v-if="contextWindow")
                ContextMeter.ml-auto(
                  :tokens="taskUsage?.contextTokens ?? 0"
                  :window="contextWindow"
                )

    aside#task-details.bg-card.absolute.top-0.right-0.bottom-0.flex.w-80.shrink-0.flex-col.border-l(
      v-if="detailsOpen"
    )
      .flex.h-11.shrink-0.items-center.gap-1.border-b.px-4
        button.rounded-md.px-2.py-1.text-xs.font-medium(
          v-for="option in DETAILS_TABS"
          :key="option.value"
          type="button"
          :class="detailsTab === option.value ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/60'"
          @click="detailsTab = option.value"
        ) {{ option.label }}
      .flex.flex-1.flex-col.gap-4.overflow-y-auto.p-4(v-if="task")
        .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Assignee

            Select(
              v-if="canEditAssignee"
              :model-value="assigneeChoice"
              @update:model-value="onAssigneeChange"
              :disabled="saving.assignee"
            )
              SelectTrigger.w-full(size="sm" title="Change assignee")
                SelectValue(placeholder="Unassigned")
              SelectContent
                SelectGroup
                  SelectItem(:value="AUTO") Auto-assign
                  SelectItem(
                    v-for="item in assigneeOptions"
                    :key="item.id"
                    :value="item.id"
                  ) {{ item.displayName || item.username }}
            template(v-else)
              .flex.items-center.gap-2(v-if="task.assigneeActorId")
                RouterLink.flex.items-center.gap-2(
                  v-if="usernameFor(task.assigneeActorId)"
                  class="hover:underline"
                  :to="{ name: 'agent', params: { username: usernameFor(task.assigneeActorId) } }"
                )
                  span.size-5.shrink-0(
                    :innerHTML="agentAvatar(task.assigneeActorId)"
                  )
                  span.text-sm {{ agentName(task.assigneeActorId) }}
                template(v-else)
                  span.size-5.shrink-0(
                    :innerHTML="agentAvatar(task.assigneeActorId)"
                  )
                  span.text-sm {{ agentName(task.assigneeActorId) }}
                PinIcon.text-muted-foreground.size-3.shrink-0(
                  v-if="task.assigneePinned"
                  title="Pinned to this agent"
                  aria-label="Pinned to this agent"
                )
              span.text-muted-foreground.text-sm(v-else) Unassigned
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Status
            .flex.items-center.gap-2
              span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
                class="gap-1.5 py-0.5"
                :class="taskStateClassFor(task)"
              )
                component.size-3.shrink-0(:is="taskStateIcon(task.state)")
                | {{ taskStateLabelFor(task) }}
              span.text-muted-foreground.text-xs(v-if="isActive") holds the agent's slot

        .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Priority
            Select(
              :model-value="priorityChoice"
              @update:model-value="onPriorityChange"
              :disabled="saving.priority"
            )
              SelectTrigger.w-full(size="sm" title="Change priority")
                SelectValue
              SelectContent
                SelectGroup
                  SelectItem(
                    v-for="option in PRIORITIES"
                    :key="option.value"
                    :value="option.value"
                  ) {{ option.label }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Size
            Select(
              :model-value="sizeChoice"
              @update:model-value="onSizeChange"
              :disabled="saving.size"
            )
              SelectTrigger.w-full(size="sm" title="Change size")
                SelectValue
              SelectContent
                SelectGroup
                  SelectItem(
                    v-for="option in SIZES"
                    :key="option.value"
                    :value="option.value"
                  ) {{ option.label }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Project

            Select(
              v-if="isDraft"
              :model-value="projectChoice"
              @update:model-value="onProjectChange"
              :disabled="saving.project"
            )
              SelectTrigger.w-full(size="sm" title="Change project")
                SelectValue(placeholder="Select project")
              SelectContent
                SelectGroup
                  SelectItem(
                    v-for="project in projectOptions"
                    :key="project.id"
                    :value="project.id"
                  ) {{ project.name }}
            template(v-else)
              RouterLink.text-sm(
                v-if="task.projectId && projectSlug(task.projectId)"
                class="hover:underline"
                :to="{ name: 'project', params: { slug: projectSlug(task.projectId) } }"
              ) {{ projectName(task.projectId) }}
              span.text-muted-foreground.text-sm(v-else) No project

        .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Task id
            span.font-mono.text-xs.break-all(:title="task.id") {{ task.id }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Created
            span.text-xs {{ formatTimestamp(task.createdAt) }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Creator
            .flex.items-center.gap-2(
              v-if="task.creatorName || task.creatorActorId"
            )
              ActorAvatar(
                :src="task.creatorPictureUrl"
                :name="task.creatorName"
                :username="task.creatorActorId"
                :size="20"
              )
              span.text-sm(v-if="task.creatorName") {{ task.creatorName }}
              span.font-mono.text-xs(v-else) {{ task.creatorActorId }}
            span.text-muted-foreground.text-sm(v-else) —
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Owning teams
            .flex.flex-wrap.items-center.gap-1
              span.rounded-full.border.px-2.text-xs(
                v-for="team in task.teams"
                :key="team.id"
                class="py-0.5"
              ) {{ team.name || `Team ${team.id}` }}
              DropdownMenu
                DropdownMenuTrigger(as-child)
                  Button(
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    title="Edit owning teams"
                    :disabled="saving.teams"
                  )
                    PlusIcon.size-3
                DropdownMenuContent.w-56(align="start")
                  DropdownMenuLabel Owning teams
                  DropdownMenuSeparator
                  p.text-muted-foreground.px-2.py-1.text-xs(
                    v-if="!projectTeams.length"
                  )
                    | No teams in this project
                  DropdownMenuCheckboxItem(
                    v-for="team in projectTeams"
                    :key="team.team_id"
                    :model-value="teamIds.includes(team.team_id)"
                    @update:model-value="toggleTaskTeam(team.team_id)"
                    @select.prevent
                  ) {{ team.team_name }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Leads
            .flex.flex-wrap.items-center.gap-1
              span.inline-flex.items-center.gap-1.rounded-full.border.pr-2.pl-1.text-xs(
                v-for="lead in task.leads"
                :key="lead.actorId"
                class="py-0.5"
              )
                ActorAvatar(
                  :src="lead.pictureUrl"
                  :name="lead.name"
                  :username="lead.username"
                  :size="16"
                )
                span {{ lead.name || lead.username || lead.actorId }}
              DropdownMenu
                DropdownMenuTrigger(as-child)
                  Button(
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    title="Edit leads"
                    :disabled="saving.leads"
                  )
                    PlusIcon.size-3
                DropdownMenuContent.w-56(align="start")
                  DropdownMenuLabel Leads
                  DropdownMenuSeparator
                  p.text-muted-foreground.px-2.py-1.text-xs(
                    v-if="!leadCandidates.length"
                  )
                    | No members to pick
                  DropdownMenuCheckboxItem(
                    v-for="candidate in leadCandidates"
                    :key="candidate.actor_id"
                    :model-value="leadIds.includes(candidate.actor_id)"
                    @update:model-value="toggleTaskLead(candidate.actor_id)"
                    @select.prevent
                  )
                    ActorAvatar(
                      :src="candidate.picture_url"
                      :name="candidate.name"
                      :username="candidate.username"
                      :size="18"
                    )
                    span {{ candidate.name || candidate.username }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Notifications
            Button(
              type="button"
              variant="outline"
              size="sm"
              @click="toggleSubscription"
              :disabled="saving.subscription"
            ) {{ task.subscribed ? "Unsubscribe" : "Subscribe" }}

        .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
          span.text-muted-foreground.text-xs Tags
          .flex.flex-wrap.items-center.gap-1
            span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
              v-for="tag in task.tags"
              :key="tag.id"
              class="py-0.5"
              :style="tagStyle(tag.color)"
            )
              span.rounded-full.bg-current(class="size-1.5")
              | {{ tag.name }}
              button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
                type="button"
                class="p-0.5 hover:opacity-100"
                title="Remove tag"
                @click="toggleTag(tag)"
                :disabled="saving.tags"
              )
                XIcon.size-3
            span.text-muted-foreground.text-sm(v-if="!task.tags.length") —
            DropdownMenu(v-if="unattachedTags.length")
              DropdownMenuTrigger(as-child)
                Button(
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  title="Add tag"
                  :disabled="saving.tags"
                )
                  PlusIcon.size-3
              DropdownMenuContent.w-48(align="start")
                DropdownMenuLabel Add tag
                DropdownMenuSeparator
                DropdownMenuItem(
                  v-for="tag in unattachedTags"
                  :key="tag.id"
                  @click="toggleTag(tag)"
                )
                  span.size-2.shrink-0.rounded-full(
                    :style="{ backgroundColor: tagColor(tag.color) }"
                  )
                  span {{ tag.name }}

        .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
          span.text-muted-foreground.text-xs Attachments
          p.text-muted-foreground.text-xs(v-if="canFollowUp")
            | Attach files to a message below.
          AttachmentList(
            :attachments="task.attachments"
            @open="openAttachment"
          )

        TaskReferences(
          v-if="detailsTab === 'general'"
          :task="task"
          @updated="onReferenceChange"
        )

        .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
          .flex.items-center.justify-between
            span.text-muted-foreground.text-xs Links
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              title="Edit links"
              aria-label="Edit links"
              @click="linksDialogOpen = true"
              data-test="edit-links"
            )
              PencilIcon.size-3
          .flex.flex-col.gap-1(v-if="task.links.length")
            a.flex.items-center.gap-2.rounded-md.border.px-2.py-1.text-xs(
              v-for="link in task.links"
              :key="link.url"
              rel="noopener noreferrer"
              class="hover:bg-muted/50"
              :href="link.url"
              :title="link.url"
              target="_blank"
            )
              ExternalLinkIcon.size-3.shrink-0
              .flex.min-w-0.flex-col
                span.font-medium {{ link.label || link.url }}
                span.text-muted-foreground.truncate(v-if="link.label") {{ link.url }}
          span.text-muted-foreground.text-sm(v-else) —

        .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
          span.text-muted-foreground.text-xs Usage
          span.text-muted-foreground.text-sm(
            v-if="!isStarted || sessionDeleted"
          ) —
          .flex.flex-col.gap-1.text-xs(v-else)
            .flex.items-baseline.justify-between.gap-3
              span.text-muted-foreground Token usage
              Skeleton.h-3.w-24(v-if="!taskUsage")
              span.font-medium.tabular-nums(v-else)
                | {{ formatTokens(taskUsage.inputTokens) }} in ·
                | {{ formatTokens(taskUsage.outputTokens) }} out
            .flex.items-baseline.justify-between.gap-3
              span.text-muted-foreground Total cost
              Skeleton.h-3.w-16(v-if="!taskUsage")
              span.font-medium.tabular-nums(v-else) {{ formatCost(taskUsage.cost) }}
            .flex.items-baseline.justify-between.gap-3
              span.text-muted-foreground Total run time
              Skeleton.h-3.w-16(v-if="!taskUsage")
              span.font-medium.tabular-nums(v-else) {{ formatDuration(taskUsage.activeMs) }}
            .flex.items-baseline.justify-between.gap-3
              span.text-muted-foreground Workspace size
              Skeleton.h-3.w-16(v-if="taskDiskLoading")
              span.font-medium.tabular-nums(
                v-else-if="taskDiskBytes !== undefined"
              ) {{ formatBytes(taskDiskBytes) }}
              span.font-medium.tabular-nums(v-else) —

        .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
          span.text-muted-foreground.text-xs Session history
          span.inline-flex.items-center.gap-1.text-sm(v-if="sessionDeleted")
            CheckIcon.size-3.shrink-0.text-emerald-600
            | Stored in the agency (agent session cleaned up)
          span.text-muted-foreground.text-sm(v-else-if="!isStarted") —
          span.inline-flex.items-center.gap-1.text-sm(
            v-else-if="task.historyMigrated"
          )
            CheckIcon.size-3.shrink-0.text-emerald-600
            | Stored in the agency
          .flex.flex-col.gap-1(v-else)
            span.text-muted-foreground.text-xs
              | Only on this agent. Migrate it so the conversation survives
              | cleanup or an agent replacement.
            Button.self-start(
              variant="outline"
              size="sm"
              @click="migrate"
              :disabled="migrating"
            )
              Spinner(v-if="migrating" data-icon="inline-start")
              DatabaseIcon(v-else data-icon="inline-start")
              span Migrate storage to Agency

        .flex.flex-col.gap-1(v-if="detailsTab === 'artifacts'")
          span.text-muted-foreground.text-xs Artifacts
          TaskArtifacts(:artifacts="task.artifacts")

        .flex.flex-col.gap-1(v-if="detailsTab === 'logs'")
          span.text-muted-foreground.text-xs Activity
          TaskLogs(:logs="task.logs")

        .flex.flex-col.gap-1(v-if="detailsTab === 'prompts'")
          Disclosure(v-slot="{ open }")
            DisclosureButton.group.flex.w-full.items-center.gap-1.text-left
              ChevronRightIcon.size-3.shrink-0.transition-transform(
                :class="open ? 'rotate-90' : ''"
              )
              span.text-muted-foreground.text-xs System Prompt
            DisclosurePanel.mt-2.flex.flex-col.gap-1
              p.text-muted-foreground.text-xs(v-if="!systemPrompt.length") —
              template(v-for="section in systemPrompt" :key="section.scope")
                Disclosure.overflow-hidden.rounded-md.border(
                  as="div"
                  v-slot="{ open: sectionOpen }"
                )
                  DisclosureButton.flex.w-full.items-center.gap-1.px-2.text-left(
                    class="hover:bg-muted/60 py-1.5"
                  )
                    ChevronRightIcon.size-3.shrink-0.transition-transform(
                      :class="sectionOpen ? 'rotate-90' : ''"
                    )
                    span.text-xs {{ sectionLabel(section) }}
                  DisclosurePanel.border-t.px-2.py-2
                    pre.text-muted-foreground.font-sans.text-xs.break-words.whitespace-pre-wrap(
                      v-if="section.content"
                    ) {{ section.content }}
                    span.text-muted-foreground.text-xs.italic(v-else) No prompt set.
              p.text-muted-foreground.text-xs.italic(
                v-if="systemPromptPreview"
              )
                | Showing the latest versions; the task pins them when it starts.

        .bg-border.h-px(v-if="detailsTab === 'general'")
        .flex.flex-col.gap-2(v-if="detailsTab === 'general'")
          template(v-if="task.archived")
            Button(variant="outline" @click="unarchive" :disabled="actionBusy")
              ArchiveRestoreIcon(data-icon="inline-start")
              span Unarchive
            Button(
              v-if="task.sessionId && !task.sessionDeletedAt"
              variant="outline"
              @click="cleanupSession"
              :disabled="actionBusy"
            )
              EraserIcon(data-icon="inline-start")
              span Clean up session
            Button.text-destructive(
              variant="outline"
              @click="remove"
              :disabled="actionBusy"
            )
              Trash2Icon(data-icon="inline-start")
              span Delete
          template(v-else)
            Button(v-if="isDraft" @click="promote" :disabled="actionBusy")
              InboxIcon(data-icon="inline-start")
              span Move to Backlog
            Button(v-if="isBacklog" @click="queue" :disabled="actionBusy")
              PlayIcon(data-icon="inline-start")
              span Queue task
            Button(
              v-if="isBacklog"
              variant="outline"
              @click="demote"
              :disabled="actionBusy"
            )
              Undo2Icon(data-icon="inline-start")
              span Back to Draft
            Button(
              v-if="isRunning && task.runtime?.runInProgress"
              variant="outline"
              @click="stop"
              :disabled="stopping"
            )
              SquareIcon(data-icon="inline-start")
              span Stop run
            Button(
              v-if="isQueued"
              variant="outline"
              @click="dequeue"
              :disabled="actionBusy"
            )
              Undo2Icon(data-icon="inline-start")
              span Back to Backlog
            Button(
              v-if="canMoveToBacklog"
              variant="outline"
              @click="reopen(false)"
              :disabled="actionBusy"
            )
              InboxIcon(data-icon="inline-start")
              span Move to Backlog
            Button(
              v-if="canRework"
              @click="reopen(true)"
              :disabled="actionBusy"
            )
              PlayIcon(data-icon="inline-start")
              span Re-queue
            Button(
              v-if="canCloseOut"
              variant="outline"
              :title="isRunning || isInput ? 'Stop the run or let it settle before closing out' : 'Close out task'"
              @click="closeOut"
              :disabled="actionBusy || isRunning || isInput"
            )
              CheckIcon(data-icon="inline-start")
              span Close out
            Button(
              v-if="canChangeOutcome"
              variant="outline"
              title="Switch between completed and cancelled"
              @click="closeOut"
              :disabled="actionBusy"
            )
              ArrowLeftRightIcon(data-icon="inline-start")
              span Change outcome
            Button(
              v-if="canArchive"
              variant="outline"
              @click="archive"
              :disabled="actionBusy"
            )
              ArchiveIcon(data-icon="inline-start")
              span Archive
            Button.text-destructive(
              variant="outline"
              @click="remove"
              :disabled="actionBusy"
            )
              Trash2Icon(data-icon="inline-start")
              span Delete
            p.text-muted-foreground.text-xs(v-if="isBacklog")
              | You can edit the prompt in Backlog. Move the task back to Draft
              | to change its project.
            p.text-muted-foreground.text-xs(v-if="!isDraft && !isBacklog")
              | Re-queue the task, then move it back to draft to edit its prompt
              | or project.

  CloseOutTaskDialog(
    v-model:open="closeOutDialogOpen"
    :task="task ?? null"
    @closed="load"
  )

  ArchiveTaskDialog(
    :open="archiveOpen"
    :tasks="task ? [task] : []"
    :busy="actionBusy"
    @update:open="archiveOpen = $event"
    @confirm="confirmArchive"
  )

  TaskLinksDialog(
    :open="linksDialogOpen"
    :links="linkInputs"
    :busy="saving.links"
    @update:open="linksDialogOpen = $event"
    @save="onLinksSave"
  )

  MigrateDeleteDialog(
    v-model:open="deleteDialogOpen"
    :task="task ?? null"
    @deleted="onDeleted"
  )

  SketchDialog(
    v-if="sketchMounted"
    v-model:open="sketchOpen"
    :title="sketchTitle"
    :snapshot="sketchSnapshot"
    :background-image="sketchBackground"
    :saving="sketchSaving"
    @save="onSketchSave"
  )

  AttachmentLightbox(
    v-model:open="lightboxOpen"
    :attachments="lightboxAttachments"
    :start-index="lightboxIndex"
  )
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import AttachmentChips from "@/components/attachments/AttachmentChips.vue";
import AttachmentLightbox from "@/components/attachments/AttachmentLightbox.vue";
import AttachmentList from "@/components/attachments/AttachmentList.vue";
import AttachmentUploadButton from "@/components/attachments/AttachmentUploadButton.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import MessageSender from "@/components/MessageSender.vue";
import ArchiveTaskDialog from "@/components/tasks/ArchiveTaskDialog.vue";
import CloseOutTaskDialog from "@/components/tasks/CloseOutTaskDialog.vue";
import ContextMeter from "@/components/tasks/ContextMeter.vue";
import MigrateDeleteDialog from "@/components/tasks/MigrateDeleteDialog.vue";
import TaskArtifacts from "@/components/tasks/TaskArtifacts.vue";
import TaskConsole from "@/components/tasks/TaskConsole.vue";
import TaskLinksDialog from "@/components/tasks/TaskLinksDialog.vue";
import TaskLogs from "@/components/tasks/TaskLogs.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import TaskPrompt from "@/components/tasks/TaskPrompt.vue";
import TaskReferences from "@/components/tasks/TaskReferences.vue";
import WritePreviewTabs from "@/components/tasks/WritePreviewTabs.vue";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { useActorPictures } from "@/composables/useActorPictures";
import { useAgents } from "@/composables/useAgents";
import { useAttachmentUpload } from "@/composables/useAttachmentUpload";
import { useDraft } from "@/composables/useDraft";
import { setPageTitle } from "@/composables/usePageTitle";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { useRealtime } from "@/composables/useRealtime";
import {
  useSketchEditor,
  type SketchSavePayload,
} from "@/composables/useSketchEditor";
import { useTags } from "@/composables/useTags";
import { useTaskDetails } from "@/composables/useTaskDetails";
import { accountApi, type UserPreferences } from "@/lib/account";
import { queuedMessageParts, type UserMessageParts } from "@/lib/console";
import { chatDraftKey } from "@/lib/draft";
import {
  formatBytes,
  formatCost,
  formatDuration,
  formatTimestamp,
  formatTokens,
} from "@/lib/format";
import { sortProjectsByName } from "@/lib/project";
import {
  projectAccessApi,
  type LeadCandidate,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import { agency, agent, currentAgentId, currentAgentUsername } from "@/lib/sdk";
import {
  bustAttachmentCache,
  isSketchScene,
  newSketchFilename,
  sketchSceneFilename,
  sketchSceneFor,
  toStoredSnapshot,
} from "@/lib/sketches";
import {
  EMPTY_TASK_MODEL_SELECTION,
  TASK_SIZES,
  tagColor,
  tagStyle,
  taskName,
  taskNameClass,
  taskPriorityClass,
  taskPriorityLabel,
  taskSizeClass,
  taskSizeLabel,
  taskStateClassFor,
  taskStateIcon,
  taskStateLabelFor,
  taskTitlePending,
  type TaskModelSelection,
} from "@/lib/task";
import { deleteConfirmation } from "@/lib/taskActions";
import {
  PromptScope,
  TaskPriority,
  TaskSize,
  TaskState,
  type Attachment,
  type Project,
  type SystemPromptSection,
  type Tag,
  type Task,
  type TaskAttributeInput,
  type TaskConfig,
  type TaskLinkInput,
  type TaskUsage,
} from "@cherrypi/sdk";
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  ArrowLeftIcon,
  ArrowLeftRightIcon,
  CheckIcon,
  ChevronRightIcon,
  CircleSlashIcon,
  DatabaseIcon,
  EraserIcon,
  ExternalLinkIcon,
  InboxIcon,
  MenuIcon,
  PaperclipIcon,
  PencilIcon,
  PinIcon,
  PlayIcon,
  PlusIcon,
  SendIcon,
  SquareIcon,
  Trash2Icon,
  TriangleAlertIcon,
  Undo2Icon,
  XIcon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{ id: string }>();

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

const SketchDialog = defineAsyncComponent(
  () => import("@/components/sketches/SketchDialog.vue"),
);

const router = useRouter();
const route = useRoute();
const { agents, loadAgents, usernameFor, actorIdFor, agentIdForActor } =
  useAgents();
const { actorPictures, loadActorPictures, pictureFor } = useActorPictures();
const { active: allProjects, loadProjects } = useProjects();

const task = ref<Task | undefined>();

const { active: activeTags } = useTags(() => task.value?.projectId ?? "");

const { open: detailsOpen, toggle: toggleDetails } = useTaskDetails();

const lightboxOpen = ref(false);
const lightboxAttachments = ref<Attachment[]>([]);
const lightboxIndex = ref(0);
const taskUsage = ref<TaskUsage | undefined>();

const taskDiskBytes = ref<bigint | undefined>();
const taskDiskLoading = ref(false);

const config = ref<TaskConfig | undefined>();
const modelSelection = ref<TaskModelSelection>({
  ...EMPTY_TASK_MODEL_SELECTION,
});
const systemPrompt = ref<SystemPromptSection[]>([]);

const prompt = useDraft(() => chatDraftKey(props.id));

const tab = ref<"write" | "preview">("write");

const composer = ref<HTMLElement | null>(null);
const promptInput = ref<InstanceType<typeof Textarea> | null>(null);
const textareaHeight = ref<number | null>(null);
const textareaStyle = computed(() =>
  textareaHeight.value === null
    ? undefined
    : { height: `${textareaHeight.value}px` },
);

const stagedAttachments = ref<Attachment[]>([]);

const { upload: uploadAttachments } = useAttachmentUpload(() => ({
  taskId: props.id,
}));
const draggingFiles = ref(false);
let dragDepth = 0;

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
  siblings: knownAttachments,
  persist: persistSketch,
});
const sending = ref(false);
const stopping = ref(false);
const clearingQueue = ref(false);

const taskQueue = ref<(UserMessageParts & { id: number })[]>([]);
let taskQueueId = 0;
const actionBusy = ref(false);

const saving = reactive({
  assignee: false,
  priority: false,
  size: false,
  project: false,
  tags: false,
  links: false,
  teams: false,
  leads: false,
  subscription: false,
  model: false,
});

const agentProjects = ref<Project[]>([]);
const projectMembers = ref<string[]>([]);
const closeOutDialogOpen = ref(false);
const deleteDialogOpen = ref(false);
const linksDialogOpen = ref(false);
const migrating = ref(false);
const titleEditing = ref(false);
const titleDraft = ref("");
const titleInput = ref<{ $el?: HTMLInputElement } | undefined>();

const DETAILS_TABS = [
  { value: "general", label: "General" },
  { value: "artifacts", label: "Artifacts" },
  { value: "prompts", label: "Prompts" },
  { value: "logs", label: "Logs" },
];
const detailsTab = ref("general");

const PRIORITIES = [
  { value: TaskPriority.LOW, label: "Low" },
  { value: TaskPriority.NORMAL, label: "Normal" },
  { value: TaskPriority.HIGH, label: "High" },
  { value: TaskPriority.CRITICAL, label: "Critical" },
];

const AUTO = "__auto__";

const SIZES = TASK_SIZES;

let generation = 0;

let configGeneration = 0;

const isDraft = computed(() => task.value?.state === TaskState.DRAFT);
const isBacklog = computed(() => task.value?.state === TaskState.BACKLOG);

const canEditPrompt = computed(() => isDraft.value || isBacklog.value);
const isQueued = computed(() => task.value?.state === TaskState.QUEUED);

const isResting = computed(() => {
  const state = task.value?.state;
  return (
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});
const isRunning = computed(() => task.value?.state === TaskState.RUNNING);
const isInput = computed(() => task.value?.state === TaskState.INPUT);

const isBlockedTask = computed(
  () => isQueued.value && (task.value?.blockers ?? 0) > 0,
);

const assigneeChoice = computed(() => {
  const actorId = task.value?.assigneeActorId;
  return actorId ? agentIdForActor(actorId) : AUTO;
});
const priorityChoice = computed(
  () => task.value?.priority || TaskPriority.NORMAL,
);
const sizeChoice = computed(() => task.value?.size || TaskSize.MEDIUM);
const projectChoice = computed(() => task.value?.projectId ?? "");

const projectTeams = ref<ProjectTeamLink[]>([]);
const leadCandidates = ref<LeadCandidate[]>([]);
const teamIds = computed(() =>
  (task.value?.teams ?? []).map((team) => team.id),
);
const leadIds = computed(() =>
  (task.value?.leads ?? []).map((lead) => lead.actorId),
);
const canEditAssignee = computed(() => !isRunning.value && !isInput.value);
const assigneeOptions = computed(() => {
  const members = new Set(projectMembers.value);
  if (!members.size) return agents.value;
  return agents.value.filter((item) => members.has(item.id));
});
const projectOptions = computed(() =>
  sortProjectsByName(
    task.value?.assigneeActorId ? agentProjects.value : allProjects.value,
  ),
);
const unattachedTags = computed(() => {
  const attached = new Set((task.value?.tags ?? []).map((tag) => tag.id));
  return activeTags.value.filter((tag) => !attached.has(tag.id));
});

const isActive = computed(() => {
  const state = task.value?.state;
  return (
    state === TaskState.RUNNING ||
    state === TaskState.INPUT ||
    state === TaskState.SETTLED ||
    state === TaskState.FAILED
  );
});

const sessionDeleted = computed(() => Boolean(task.value?.sessionDeletedAt));

const canFollowUp = computed(
  () => !!task.value && acceptsPrompt(task.value.state),
);

function acceptsPrompt(state: TaskState): boolean {
  return (
    state === TaskState.RUNNING ||
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
}

const canMoveToBacklog = computed(() => {
  const state = task.value?.state;
  return (
    state === TaskState.SETTLED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});

const canRework = computed(() => {
  const state = task.value?.state;
  if (sessionDeleted.value) return false;
  return (
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});

const canCloseOut = computed(() => {
  const state = task.value?.state;
  return (
    !!state &&
    state !== TaskState.RUNNING &&
    state !== TaskState.INPUT &&
    state !== TaskState.COMPLETED &&
    state !== TaskState.CANCELLED
  );
});

const isClosedOut = computed(() => {
  const state = task.value?.state;
  return state === TaskState.COMPLETED || state === TaskState.CANCELLED;
});
const canChangeOutcome = isClosedOut;
const canArchive = isClosedOut;
const isStarted = computed(() => Boolean(task.value?.sessionId));

const canMigrate = computed(
  () =>
    !sessionDeleted.value &&
    isStarted.value &&
    !!task.value &&
    !task.value.historyMigrated,
);

const showPrompt = computed(() => !isStarted.value);
const promptHint = computed(() => {
  if (isBacklog.value) return "Backlog · click to edit";
  if (isBlockedTask.value) return "Blocked · waiting on blocking references";
  if (isQueued.value) return "Queued · waiting for a free slot";
  return "";
});

const showPendingPrompt = computed(
  () => !!task.value?.pendingPrompt && isStarted.value,
);
const pendingPromptHint = computed(() => {
  if (isDraft.value) return "Unsent follow-up · click to edit";
  if (isBacklog.value) return "Backlog · not queued yet";
  if (isBlockedTask.value) return "Blocked · waiting on blocking references";
  if (isQueued.value) return "Queued · waiting for a free slot";
  if (isResting.value)
    return "Undelivered follow-up · the next run delivers it";
  return "";
});
const busy = computed(() => !task.value || task.value.runtime?.runInProgress);

const steer = computed(() => task.value?.runtime?.runInProgress === true);

const contextWindow = computed(() =>
  Number(taskUsage.value?.contextWindow ?? 0),
);

const systemPromptPreview = computed(
  () => systemPrompt.value.length > 0 && !systemPrompt.value[0]?.pinned,
);

const PROMPT_SCOPE_LABELS: Record<number, string> = {
  [PromptScope.ORG]: "Organization Prompt",
  [PromptScope.AGENT]: "Agent Prompt",
  [PromptScope.PROJECT]: "Project Prompt",
  [PromptScope.AGENT_PROJECT]: "Agent-Project prompt",
  [PromptScope.TEAM]: "Team Prompt",
  [PromptScope.TEAM_PROJECT]: "Team-Project prompt",
  [PromptScope.BUILTIN]: "Built-in guidance",
};

function sectionLabel(section: SystemPromptSection): string {
  const base = PROMPT_SCOPE_LABELS[section.scope] ?? "Prompt";
  const owner = section.owner ? ` (${section.owner})` : "";
  const version = section.version > 0 ? `Version ${section.version}` : "Unset";
  return `${base}${owner} -- ${version}`;
}

async function loadSystemPrompt() {
  try {
    systemPrompt.value = await agency.getTaskSystemPrompt(props.id);
  } catch {
    systemPrompt.value = [];
  }
}

function startTitleEdit() {
  const current = task.value;
  if (!current) return;
  titleDraft.value = current.name;
  titleEditing.value = true;
  void nextTick(() => {
    const el = titleInput.value?.$el;
    el?.focus();
    el?.select();
  });
}

function cancelTitleEdit() {
  titleEditing.value = false;
}

async function saveTitle() {
  const current = task.value;
  if (!current || !titleEditing.value) return;
  const name = titleDraft.value.trim();
  if (name === "" || name === current.name) {
    titleEditing.value = false;
    return;
  }
  try {
    task.value = await agency.renameTask(current.id, name);
    toast.success("Title updated");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    titleEditing.value = false;
  }
}

async function onPriorityChange(value: unknown) {
  const current = task.value;
  if (!current) return;
  const priority = Number(value) as TaskPriority;
  if (priority === current.priority) return;
  saving.priority = true;
  try {
    task.value = await agency.setTaskPriority(current.id, priority);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.priority = false;
  }
}

async function onAssigneeChange(value: unknown) {
  const current = task.value;
  if (!current) return;
  const agentId = String(value) === AUTO ? "" : String(value);
  if (agentId === agentIdForActor(current.assigneeActorId)) return;
  saving.assignee = true;
  try {
    task.value = await agency.updateTask(current.id, {
      setAssigneeActorId: actorIdFor(agentId),
    });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.assignee = false;
  }
}

async function toggleTag(tag: Tag) {
  const current = task.value;
  if (!current) return;
  const attached = current.tags.some((item) => item.id === tag.id);
  const attributes: TaskAttributeInput = attached
    ? { removeTagIds: [tag.id] }
    : { addTagIds: [tag.id] };
  saving.tags = true;
  try {
    const [result] = await agency.bulkUpdateTask({
      ids: [current.id],
      attributes,
    });
    if (result?.task) task.value = result.task;
    if (result?.error) toast.error(result.error);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.tags = false;
  }
}

async function onProjectChange(value: unknown) {
  const current = task.value;
  if (!current) return;
  const projectId = String(value);
  if (!projectId || projectId === current.projectId) return;
  saving.project = true;
  try {
    const available = await agency.listTags({
      projectId,
      includeArchived: true,
    });
    const valid = new Set(
      available.filter((tag) => !tag.archived).map((tag) => tag.id),
    );
    const next = await agency.updateTask(current.id, {
      prompt: current.prompt,
      name: current.name,
      autoTitle: current.autoTitle,
      workspace: current.workspace,
      priority: current.priority,
      projectId,
      assigneeActorId: current.assigneeActorId,
      tagIds: current.tags.map((tag) => tag.id).filter((id) => valid.has(id)),
      modelProvider: current.modelProvider,
      modelId: current.modelId,
      thinkingLevel: current.thinkingLevel,
    });
    task.value = next;
    if (next.id !== current.id) {
      void router.replace({ name: "task", params: { id: next.id } });
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.project = false;
  }
}

async function loadAgentProjects(agentId: string) {
  agentProjects.value = [];
  if (!agentId) return;
  try {
    agentProjects.value = await agency.listAgentProjects(agentId);
  } catch {
    agentProjects.value = [];
  }
}

async function loadProjectMembers(projectId: string) {
  projectMembers.value = [];
  if (!projectId) return;
  try {
    const members = await agency.listProjectAgents(projectId);
    projectMembers.value = members.map((member) => member.agentId);
  } catch {
    projectMembers.value = [];
  }
}

watch(
  () => task.value?.assigneeActorId ?? "",
  (id) => void loadAgentProjects(id),
  { immediate: true },
);

watch(
  () => task.value?.projectId ?? "",
  (id) => void loadProjectMembers(id),
  { immediate: true },
);

async function onSizeChange(value: unknown) {
  const current = task.value;
  if (!current) return;
  const size = Number(value) as TaskSize;
  if (size === current.size) return;
  saving.size = true;
  try {
    task.value = await agency.setTaskSize(current.id, size);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.size = false;
  }
}

async function setTaskTeams(ids: number[]) {
  const current = task.value;
  if (!current) return;
  saving.teams = true;
  try {
    task.value = await agency.setTaskTeams(current.id, { teamIds: ids });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.teams = false;
  }
}

async function toggleTaskTeam(id: number) {
  const next = new Set(teamIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  await setTaskTeams([...next]);
}

async function setTaskLeads(ids: string[]) {
  const current = task.value;
  if (!current) return;
  saving.leads = true;
  try {
    task.value = await agency.setTaskLeads(current.id, { leadActorIds: ids });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.leads = false;
  }
}

async function toggleTaskLead(id: string) {
  const next = new Set(leadIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  await setTaskLeads([...next]);
}

async function toggleSubscription() {
  const current = task.value;
  if (!current) return;
  saving.subscription = true;
  try {
    task.value = await agency.setTaskSubscription(current.id, {
      subscribed: !current.subscribed,
    });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.subscription = false;
  }
}

function projectName(id: string): string {
  return (
    allProjects.value.find((project) => project.id === id)?.name ??
    id.slice(0, 8)
  );
}

function projectSlug(id: string): string {
  return allProjects.value.find((project) => project.id === id)?.slug ?? "";
}

function agentName(id: string): string {
  const item = agents.value.find(
    (item) => item.id === id || item.actorId === id,
  );
  return item?.displayName || item?.username || id.slice(0, 8);
}

function agentAvatar(id: string): string {
  return toSvg(id, 20);
}

async function loadUsage(agentId: string, sessionId: string) {
  if (!sessionId) {
    taskUsage.value = undefined;
    return;
  }
  try {
    taskUsage.value = await agent.getTaskUsage(agentId, sessionId);
  } catch {
    taskUsage.value = undefined;
  }
}

const DISK_REFRESH_MS = 10_000;
let diskKey = "";
let diskAt = 0;

async function loadDiskUsage(agentId: string, sessionId: string) {
  if (!sessionId) {
    taskDiskBytes.value = undefined;
    taskDiskLoading.value = false;
    return;
  }
  const key = `${agentId}\u0000${sessionId}`;
  const now = Date.now();
  if (key === diskKey && now - diskAt < DISK_REFRESH_MS) return;
  diskKey = key;
  diskAt = now;
  taskDiskLoading.value = true;
  try {
    taskDiskBytes.value = await agent.getTaskDiskUsage(agentId, sessionId);
  } catch {
    taskDiskBytes.value = undefined;
  } finally {
    taskDiskLoading.value = false;
  }
}

async function loadConfig() {
  const gen = ++configGeneration;
  try {
    const next = await agent.getTaskConfig(props.id);
    if (gen !== configGeneration) return;
    config.value = next;
    modelSelection.value = {
      modelProvider: next.modelProvider,
      modelId: next.modelId,
      modelName: next.modelName,
      thinkingLevel: next.thinkingLevel,
    };
  } catch {}
}

async function onModelSelectionChange(next: TaskModelSelection) {
  const current = task.value;
  if (!current) return;
  const previous = modelSelection.value;
  const modelChanged =
    next.modelProvider !== previous.modelProvider ||
    next.modelId !== previous.modelId;
  const levelChanged = next.thinkingLevel !== previous.thinkingLevel;
  if (!modelChanged && !levelChanged) return;
  saving.model = true;
  try {
    let changed = config.value;
    if (modelChanged) {
      if (!next.modelProvider || !next.modelId) return;
      changed = await withProgress(() =>
        agent.setTaskModel(current.id, {
          provider: next.modelProvider,
          modelId: next.modelId,
        }),
      );
    }
    if (
      levelChanged &&
      next.thinkingLevel &&
      changed &&
      next.thinkingLevel !== changed.thinkingLevel
    ) {
      changed = await withProgress(() =>
        agent.setTaskThinkingLevel(current.id, next.thinkingLevel),
      );
    }
    if (changed) {
      config.value = changed;
      modelSelection.value = {
        modelProvider: changed.modelProvider,
        modelId: changed.modelId,
        modelName: changed.modelName,
        thinkingLevel: changed.thinkingLevel,
      };
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.model = false;
  }
}

async function load() {
  const gen = ++generation;
  try {
    const next = await agency.getTask(props.id);
    if (gen !== generation) return;
    task.value = next;

    if (!config.value) {
      modelSelection.value = {
        modelProvider: next.modelProvider,
        modelId: next.modelId,
        thinkingLevel: next.thinkingLevel,
      };
    }
    const owner = next.assigneeActorId;
    if (owner) {
      currentAgentId.value = owner;
      currentAgentUsername.value = usernameFor(owner);

      if (!next.sessionDeletedAt) {
        void loadUsage(owner, next.sessionId);
        void loadDiskUsage(owner, next.sessionId);
        if (acceptsPrompt(next.state) && !saving.model) {
          void loadConfig();
        }
      }
    }
  } catch (err) {
    if (gen !== generation) return;
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function onDraftSaved(next: Task) {
  task.value = next;
}

function onPendingPromptSaved() {
  void load();
}

function onReferenceChange(next: Task) {
  task.value = next;
}

const linkInputs = computed<TaskLinkInput[]>(() =>
  (task.value?.links ?? []).map((link) => ({
    url: link.url,
    label: link.label,
  })),
);

async function onLinksSave(links: TaskLinkInput[]) {
  const current = task.value;
  if (!current) return;
  saving.links = true;
  try {
    task.value = await agency.setTaskLinks(current.id, links);
    linksDialogOpen.value = false;
    toast.success("Links updated");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.links = false;
  }
}

function back() {
  const from = route.query.from;
  if (from === "inbox") {
    void router.push({ name: "inbox" });
    return;
  }
  if (typeof from === "string" && from) {
    void router.push({ name: "agent", params: { username: from } });
    return;
  }
  void router.push({ name: "agency-tasks" });
}

async function queue() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.queueTask(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

async function promote() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.promoteTask(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

async function demote() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.demoteTask(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

async function dequeue() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.dequeueTask(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

const archiveOpen = ref(false);
const archivePrefs = ref<UserPreferences>({ notify_first_archive: true });

async function runArchive() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.archiveTask(props.id));
    toast.success("Task archived");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

function archive() {
  if (archivePrefs.value.notify_first_archive) {
    archiveOpen.value = true;
    return;
  }
  void runArchive();
}

async function confirmArchive(dontShowAgain: boolean) {
  archiveOpen.value = false;
  if (dontShowAgain && archivePrefs.value.notify_first_archive) {
    archivePrefs.value = { ...archivePrefs.value, notify_first_archive: false };
    try {
      await accountApi.updatePreferences({ notify_first_archive: false });
    } catch {}
  }
  await runArchive();
}

async function unarchive() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.unarchiveTask(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

async function cleanupSession() {
  actionBusy.value = true;
  try {
    task.value = await withProgress(() => agent.cleanupTaskSession(props.id));
    toast.success("Session cleaned up");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

function closeOut() {
  closeOutDialogOpen.value = true;
}

async function reopen(toQueue: boolean) {
  actionBusy.value = true;
  try {
    if (
      toQueue &&
      !window.confirm(`Re-queue "${task.value ? taskName(task.value) : ""}"?`)
    ) {
      return;
    }
    task.value = await withProgress(() =>
      agent.reopenTask(props.id, { queue: toQueue }),
    );
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

async function stop() {
  stopping.value = true;
  try {
    await withProgress(() => agent.stopTask(props.id));
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    stopping.value = false;
  }
}

async function send() {
  if (!prompt.value.trim()) return;
  sending.value = true;
  try {
    const attachmentIds = stagedAttachments.value
      .filter((attachment) => !isSketchScene(attachment))
      .map((attachment) => attachment.id);
    await withProgress(() =>
      agent.addToTask(props.id, prompt.value, { attachmentIds }),
    );
    prompt.value = "";
    stagedAttachments.value = [];

    tab.value = "write";
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    sending.value = false;
  }
}

const MIN_TEXTAREA_HEIGHT = 48;
const resizing = ref(false);
let resizeStartY = 0;
let resizeStartHeight = 0;

function startResize(event: PointerEvent) {
  if (event.button !== 0 || !canFollowUp.value) return;
  const el = promptInput.value?.$el;
  if (!el) return;
  event.preventDefault();
  resizing.value = true;
  resizeStartY = event.clientY;
  resizeStartHeight = el.offsetHeight;
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", stopResize, { once: true });
}

function onResizeMove(event: PointerEvent) {
  const column = composer.value?.parentElement;
  const max = column
    ? Math.max(MIN_TEXTAREA_HEIGHT, column.clientHeight - 120)
    : 480;
  const next = resizeStartHeight + (resizeStartY - event.clientY);
  textareaHeight.value = Math.min(Math.max(next, MIN_TEXTAREA_HEIGHT), max);
}

function stopResize() {
  resizing.value = false;
  window.removeEventListener("pointermove", onResizeMove);
}

async function onAttachmentUploaded() {
  await load();
}

function onStagedUploaded(attachment: Attachment) {
  stagedAttachments.value = [...stagedAttachments.value, attachment];
}

function hasFiles(event: DragEvent): boolean {
  return Array.from(event.dataTransfer?.types ?? []).includes("Files");
}

function onDragEnter(event: DragEvent) {
  if (canFollowUp.value && hasFiles(event)) {
    dragDepth += 1;
    draggingFiles.value = true;
  }
}

function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1);
  if (dragDepth === 0) draggingFiles.value = false;
}

async function onDrop(event: DragEvent) {
  dragDepth = 0;
  draggingFiles.value = false;
  if (!canFollowUp.value) return;
  await stageUploads(event.dataTransfer?.files ?? []);
}

async function onPaste(event: ClipboardEvent) {
  if (!canFollowUp.value) return;
  const files: File[] = [];
  for (const item of event.clipboardData?.items ?? []) {
    if (item.kind !== "file") continue;
    const file = item.getAsFile();
    if (file) files.push(file);
  }
  if (!files.length) return;
  event.preventDefault();
  await stageUploads(files);
}

async function stageUploads(files: Iterable<File>) {
  const uploaded = await uploadAttachments(files);
  if (uploaded.length) {
    stagedAttachments.value = [...stagedAttachments.value, ...uploaded];
  }
}

function removeStaged(attachment: Attachment) {
  const scene = sketchSceneFor(stagedAttachments.value, attachment);
  const dropped = new Set([attachment.id, scene?.id]);
  stagedAttachments.value = stagedAttachments.value.filter(
    (item) => !dropped.has(item.id),
  );
}

async function removeAttachment(attachment: Attachment) {
  try {
    const scene = sketchSceneFor(task.value?.attachments ?? [], attachment);
    await agency.deleteAttachment(attachment.id);
    if (scene) await agency.deleteAttachment(scene.id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function knownAttachments(): Attachment[] {
  return [...stagedAttachments.value, ...(task.value?.attachments ?? [])];
}

function lightboxItems(): Attachment[] {
  const seen = new Set<string>();
  return knownAttachments().filter((attachment) => {
    const id = attachment.id.toString();
    if (isSketchScene(attachment) || seen.has(id)) return false;
    seen.add(id);
    return true;
  });
}

function openAttachment(attachment: Attachment) {
  const items = lightboxItems();
  let index = items.findIndex((item) => item.id === attachment.id);
  if (index === -1) {
    items.push(attachment);
    index = items.length - 1;
  }
  lightboxAttachments.value = items;
  lightboxIndex.value = index;
  lightboxOpen.value = true;
}

function replaceKnownAttachment(updated: Attachment) {
  stagedAttachments.value = stagedAttachments.value.map((item) =>
    item.id === updated.id ? updated : item,
  );
  if (task.value) {
    task.value.attachments = task.value.attachments.map((item) =>
      item.id === updated.id ? updated : item,
    );
  }
}

async function persistSketch({ png, snapshot, target }: SketchSavePayload) {
  if (!task.value) return;
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
    replaceKnownAttachment(bustAttachmentCache(updatedImage));
    replaceKnownAttachment(bustAttachmentCache(updatedScene));
  } else {
    const uploadedImage = await agency.uploadAttachment({
      file: png,
      filename: newSketchFilename(),
      taskId: task.value.id,
    });
    const uploadedScene = await agency.uploadAttachment({
      file: sceneBlob,
      filename: sketchSceneFilename(uploadedImage.id),
      taskId: task.value.id,
    });
    if (canFollowUp.value) {
      stagedAttachments.value = [
        ...stagedAttachments.value,
        uploadedImage,
        uploadedScene,
      ];
    } else {
      await load();
    }
  }
}

function onQueue(steering: string[]) {
  taskQueue.value = steering.map((text) => ({
    id: taskQueueId++,
    ...queuedMessageParts(text),
  }));
}

function queueAttachments(ids: string[]): Attachment[] {
  const byId = new Map(
    [...stagedAttachments.value, ...(task.value?.attachments ?? [])].map(
      (attachment) => [String(attachment.id), attachment],
    ),
  );
  return ids
    .map((id) => byId.get(id))
    .filter((attachment): attachment is Attachment => attachment !== undefined);
}

async function clearQueue() {
  clearingQueue.value = true;
  try {
    await withProgress(() => agency.clearTaskQueue(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    clearingQueue.value = false;
  }
}

function onEnter(event: KeyboardEvent) {
  if (event.isComposing) return;
  event.preventDefault();
  void send();
}

async function remove() {
  const current = task.value;
  if (!current) return;

  if (canMigrate.value) {
    deleteDialogOpen.value = true;
    return;
  }
  if (!window.confirm(deleteConfirmation(current))) {
    return;
  }
  await deleteTask();
}

async function migrate() {
  if (!task.value) return;
  migrating.value = true;
  try {
    const result = await agent.migrateTaskHistory(props.id);
    task.value = result.task;
    if (result.task.historyMigrated) {
      toast.success(
        `Stored ${result.entries} entr${result.entries === 1 ? "y" : "ies"} in the agency`,
      );
    } else {
      toast.error("The agent had no history to migrate");
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    migrating.value = false;
  }
}

async function deleteTask() {
  actionBusy.value = true;
  try {
    await withProgress(() => agent.deleteTask(props.id));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
    return;
  } finally {
    actionBusy.value = false;
  }
  void router.push({ name: "agency-tasks" });
}

function onDeleted() {
  void router.push({ name: "agency-tasks" });
}

watch(
  () => task.value?.projectId ?? "",
  async (id) => {
    projectTeams.value = [];
    leadCandidates.value = [];
    if (!id) return;
    try {
      const access = await projectAccessApi.access(id);
      projectTeams.value = access.teams;
    } catch {}
    try {
      leadCandidates.value = await projectAccessApi.leadCandidates(id);
    } catch {}
  },
  { immediate: true },
);

onMounted(() => {
  void loadAgents();
  void loadActorPictures();
  void loadProjects();
  void load();
  void accountApi
    .getPreferences()
    .then((prefs) => {
      archivePrefs.value = prefs;
    })
    .catch(() => {});
});

useRealtime(["tasks", "usage"], () => void load());

watch(
  () => (task.value ? taskName(task.value) : "Task"),
  (title) => setPageTitle(title),
  { immediate: true },
);

watch(
  () => props.id,
  () => {
    task.value = undefined;
    config.value = undefined;
    modelSelection.value = { ...EMPTY_TASK_MODEL_SELECTION };
    configGeneration += 1;
    taskUsage.value = undefined;
    taskDiskBytes.value = undefined;
    taskDiskLoading.value = false;
    diskKey = "";
    void load();
    void loadSystemPrompt();
  },
);

watch(
  () => `${task.value?.state ?? ""}\u0000${task.value?.sessionId ?? ""}`,
  () => void loadSystemPrompt(),
);

onBeforeUnmount(() => {
  window.removeEventListener("pointermove", onResizeMove);
});
</script>
