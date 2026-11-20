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
aside#task-details.bg-card.absolute.top-0.right-0.bottom-0.flex.shrink-0.flex-col.border-l(
  v-if="open && task"
  :style="{ width: `${width}px` }"
)
  .group.absolute.inset-y-0.-left-1.z-20.w-2.cursor-col-resize.touch-none(
    role="separator"
    aria-orientation="vertical"
    tabindex="0"
    class="focus-visible:ring-ring/50 focus-visible:ring-2 focus-visible:outline-none"
    :aria-label="$t('taskDetails.resize')"
    :title="$t('taskDetails.resize')"
    :aria-valuenow="width"
    :aria-valuemin="minWidth"
    :aria-valuemax="maxWidth"
    @pointerdown="startResize"
    @keydown="onResizeKey"
    data-testid="task-details-resize"
  )
    .mx-auto.h-full.w-px.transition-colors(
      :class="resizing ? 'bg-primary' : 'bg-transparent group-hover:bg-border'"
    )
  .no-scrollbar.flex.h-11.shrink-0.items-center.gap-1.overflow-x-auto.border-b.px-4
    button.flex.items-center.rounded-md.px-2.py-1.text-xs.font-medium.whitespace-nowrap(
      v-for="option in detailsTabs"
      :key="option.value"
      type="button"
      :class="detailsTab === option.value ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/60'"
      @click="detailsTab = option.value"
    )
      span {{ option.label }}
      span.ml-1.inline-flex.items-center.justify-center.rounded-full.bg-amber-500.px-1.text-xs.font-semibold.text-white(
        v-if="option.value === 'comments' && task.unreadComments > 0"
        :title="$t('tasksTable.unreadComments', { count: task.unreadComments })"
        data-testid="comments-unread"
      ) {{ task.unreadComments > 99 ? "99+" : task.unreadComments }}
  .flex.flex-1.flex-col.gap-4.overflow-y-auto.p-4
    .flex.flex-col.gap-1(v-if="detailsTab === 'general' && task.badges.length")
      span.text-muted-foreground.text-xs {{ $t("taskColumn.badges") }}
      TaskBadges(:badges="task.badges" data-testid="task-details-badges")
    .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskColumn.assignee") }}

        Select(
          v-if="canEditAssignee"
          :model-value="assigneeChoice"
          @update:model-value="$emit('assignee-change', $event)"
          :disabled="saving.assignee"
        )
          SelectTrigger.w-full(
            size="sm"
            :title="$t('taskDetails.changeAssignee')"
          )
            SelectValue(:placeholder="$t('tasks.toasts.unassigned')")
          SelectContent
            SelectGroup
              SelectItem(:value="AUTO") {{ $t("editTask.autoAssign") }}
              SelectItem(
                v-for="item in assigneeOptions"
                :key="item.id"
                :value="item.id"
              ) {{ item.displayName || item.username }}
        template(v-else)
          .flex.items-center.gap-2(v-if="task.assigneeActorId")
            RouterLink.flex.items-center.gap-2(
              v-if="agentUsername(task.assigneeActorId)"
              class="hover:underline"
              :to="{ name: 'agent', params: { username: agentUsername(task.assigneeActorId) } }"
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
              :title="$t('tasksTable.pinned')"
              :aria-label="$t('tasksTable.pinned')"
            )
          span.text-muted-foreground.text-sm(v-else) {{ $t("tasks.toasts.unassigned") }}
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("common.status") }}
        .flex.items-center.gap-2
          span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
            class="gap-1.5 py-0.5"
            :class="taskStateClassFor(task)"
          )
            component.size-3.shrink-0(:is="taskStateIcon(task.state)")
            | {{ $t(taskStateKeyFor(task)) }}
          span.text-muted-foreground.text-xs(v-if="isActive") {{ $t("taskDetails.holdsSlot") }}

    .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskColumn.priority") }}
        Select(
          :model-value="priorityChoice"
          @update:model-value="$emit('priority-change', $event)"
          :disabled="saving.priority"
        )
          SelectTrigger.w-full(
            size="sm"
            :title="$t('taskDetails.changePriority')"
          )
            SelectValue
          SelectContent
            SelectGroup
              SelectItem(
                v-for="option in PRIORITIES"
                :key="option.value"
                :value="option.value"
              ) {{ $t(option.labelKey) }}
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskColumn.size") }}
        Select(
          :model-value="sizeChoice"
          @update:model-value="$emit('size-change', $event)"
          :disabled="saving.size"
        )
          SelectTrigger.w-full(size="sm" :title="$t('taskDetails.changeSize')")
            SelectValue
          SelectContent
            SelectGroup
              SelectItem(
                v-for="option in SIZES"
                :key="option.value"
                :value="option.value"
              ) {{ $t(option.labelKey) }}
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskColumn.project") }}

        Combobox(
          v-if="isEditableBacklog"
          :placeholder="$t('editTask.selectProject')"
          size="sm"
          empty-text="No matching project."
          :title="$t('taskDetails.changeProject')"
          :aria-label="$t('taskColumn.project')"
          :model-value="projectChoice"
          :options="projectComboboxOptions"
          @update:model-value="$emit('project-change', $event)"
          :disabled="saving.project"
        )
        template(v-else)
          RouterLink.text-sm(
            v-if="task.projectId && projectSlug(task.projectId)"
            class="hover:underline"
            :to="{ name: 'project', params: { slug: projectSlug(task.projectId) } }"
          ) {{ projectName(task.projectId) }}
          span.text-muted-foreground.text-sm(v-else) {{ $t("taskDetails.noProject") }}
      .flex.flex-col(v-if="task.automated" class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskDetails.mode") }}
        span.inline-flex.w-fit.items-center.gap-1.rounded-full.border.px-2.text-xs(
          class="py-0.5"
          :title="$t('taskDetails.automatedHint')"
          data-testid="task-automated-detail"
        )
          CogIcon.size-3
          | {{ $t("tasksTable.automated") }}

    .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskDetails.taskId") }}
        span.font-mono.text-xs.break-all(:title="task.id") {{ task.id }}
      .flex.flex-col(
        v-if="task.originTaskId"
        class="gap-0.5"
        data-testid="task-origin"
      )
        span.text-muted-foreground.text-xs {{ $t("taskDetails.origin") }}
        .flex.flex-wrap.items-center.gap-1
          template(v-if="task.originChain.length")
            template(
              v-for="(origin, index) in task.originChain"
              :key="origin.id"
            )
              RouterLink.font-mono.text-xs(
                class="hover:underline"
                :to="{ name: 'task', params: { id: origin.id } }"
                :title="origin.name || origin.prompt || origin.id"
              ) {{ origin.id }}
              ArrowRightIcon.text-muted-foreground.size-3.shrink-0(
                v-if="index < task.originChain.length - 1"
              )
          RouterLink.font-mono.text-xs(
            v-else
            class="hover:underline"
            :to="{ name: 'task', params: { id: task.originTaskId } }"
          ) {{ task.originTaskId }}
        span.text-muted-foreground.text-xs(
          :title="$t('taskDetails.originHint')"
        ) {{ $t("taskDetails.originDescription") }}
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("domainLog.kind.created") }}
        span.text-xs {{ formatTimestamp(task.createdAt) }}
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskDetails.creator") }}
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
        span.text-muted-foreground.text-xs {{ $t("domainLog.kind.teams") }}
        .flex.flex-wrap.items-center.gap-1
          span.rounded-full.border.px-2.text-xs(
            v-for="team in task.teams"
            :key="team.id"
            class="py-0.5"
          ) {{ team.name || $t("taskDetails.teamFallback", { id: team.id }) }}
          DropdownMenu
            DropdownMenuTrigger(as-child)
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                :title="$t('taskDetails.editTeams')"
                :disabled="saving.teams"
              )
                PlusIcon.size-3
            DropdownMenuContent.w-56(align="start")
              DropdownMenuLabel {{ $t("domainLog.kind.teams") }}
              DropdownMenuFilter(
                v-model="teamSearch"
                :placeholder="$t('express.searchTeams')"
                auto-focus
              )
              DropdownMenuSeparator
              p.text-muted-foreground.px-2.py-1.text-xs(
                v-if="!filteredProjectTeams.length"
              )
                | {{ $t("express.noTeams") }}
              DropdownMenuCheckboxItem(
                v-for="team in filteredProjectTeams"
                :key="team.team_id"
                :model-value="teamIds.includes(team.team_id)"
                @update:model-value="$emit('toggle-team', team.team_id)"
                @select.prevent
              ) {{ team.team_name }}
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("taskDetails.leader") }}
        .flex.flex-wrap.items-center.gap-1
          DropdownMenu
            DropdownMenuTrigger(as-child)
              button.inline-flex.items-center.gap-1.rounded-full.border.px-1.text-xs(
                type="button"
                class="hover:bg-muted/60 py-0.5 disabled:opacity-60"
                :title="$t('taskDetails.changeLeader')"
                :disabled="saving.leader"
              )
                template(v-if="task.leader")
                  ActorAvatar(
                    :src="task.leader.pictureUrl"
                    :name="task.leader.name"
                    :username="task.leader.username"
                    :size="16"
                  )
                  span {{ task.leader.name || task.leader.username || task.leader.actorId }}
                span.text-muted-foreground(v-else) {{ $t("taskDetails.noLeader") }}
                ChevronDownIcon.text-muted-foreground.size-3
            DropdownMenuContent.w-56(align="start")
              DropdownMenuLabel {{ $t("taskDetails.leader") }}
              DropdownMenuFilter(
                v-model="leadSearch"
                :placeholder="$t('express.searchMembers')"
                auto-focus
              )
              DropdownMenuSeparator
              p.text-muted-foreground.px-2.py-1.text-xs(
                v-if="!filteredLeadCandidates.length"
              )
                | {{ $t("express.noMembers") }}
              DropdownMenuRadioGroup(
                :model-value="leaderChoice"
                @update:model-value="pickLeader"
              )
                DropdownMenuRadioItem(v-if="canRemoveLead" :value="NO_LEADER") {{ $t("taskDetails.noLeader") }}
                DropdownMenuRadioItem(
                  v-for="candidate in filteredLeadCandidates"
                  :key="candidate.actor_id"
                  :value="candidate.actor_id"
                )
                  ActorAvatar(
                    :src="candidate.picture_url"
                    :name="candidate.name"
                    :username="candidate.username"
                    :size="18"
                  )
                  span {{ candidate.name || candidate.username }}
          button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
            v-if="canRemoveLead && task.leader"
            type="button"
            class="p-0.5 hover:opacity-100"
            :title="$t('express.removeLead')"
            @click="$emit('set-leader', '')"
            :disabled="saving.leader"
          )
            XIcon.size-3
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs {{ $t("account.tabs.notifications") }}
        Button(
          type="button"
          variant="outline"
          size="sm"
          @click="$emit('toggle-subscription')"
          :disabled="saving.subscription"
        ) {{ task.subscribed ? "Unsubscribe" : "Subscribe" }}

    Disclosure(
      v-if="detailsTab === 'general' && task.template"
      as="div"
      v-slot="{ open }"
      data-testid="task-template"
    )
      DisclosureButton.flex.w-full.items-center.rounded-md.border.px-2.text-left(
        class="hover:bg-muted/60 gap-1.5 py-1.5"
        :title="$t('taskDetails.createdFromTemplate', { version: Number(task.template.version) })"
      )
        FileCodeIcon.size-3.shrink-0
        span.min-w-0.truncate.text-xs
          strong.font-semibold
            | {{ task.template.templateName || $t("taskDetails.deletedTemplate") }}
          span.text-muted-foreground.ml-1
            | v{{ Number(task.template.version) }} ·
            | {{ $t("taskDetails.argumentsCount", { count: templateArgs.length }) }}
        ChevronRightIcon.ml-auto.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
      DisclosurePanel.mt-1.flex.flex-col.gap-1.rounded-md.border.px-2(class="py-1.5")
        p.text-muted-foreground.text-xs(v-if="!templateArgs.length")
          | {{ $t("taskDetails.noArguments") }}
        .grid.grid-cols-1.gap-1(v-else)
          .grid.items-baseline.gap-2(
            v-for="[key, value] in templateArgs"
            :key="key"
            class="grid-cols-[5.5rem_1fr]"
          )
            span.text-muted-foreground.text-xs.break-words {{ key }}
            span.text-xs.break-words.whitespace-pre-wrap {{ formatArgValue(value) }}

    .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
      span.text-muted-foreground.text-xs {{ $t("taskColumn.tags") }}
      .flex.flex-wrap.items-center.gap-1
        span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
          v-for="tag in task.tags"
          :key="tag.id"
          class="py-0.5"
          :style="tagStyle(tag.color)"
        )
          span.rounded-full.bg-current(class="size-1.5")
          | {{ tag.name }}
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="tag.isRequired"
            :title="$t('tasksTable.requiredTagHint')"
          )
          button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
            type="button"
            class="p-0.5 hover:opacity-100"
            :title="$t('taskDetails.removeTag')"
            @click="$emit('toggle-tag', tag)"
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
              :title="$t('tags.add')"
              :disabled="saving.tags"
            )
              PlusIcon.size-3
          DropdownMenuContent.w-48(align="start")
            DropdownMenuLabel {{ $t("tags.add") }}
            DropdownMenuSeparator
            DropdownMenuItem(
              v-for="tag in unattachedTags"
              :key="tag.id"
              @click="$emit('toggle-tag', tag)"
            )
              span.size-2.shrink-0.rounded-full(
                :style="{ backgroundColor: tagColor(tag.color) }"
              )
              span {{ tag.name }}
              LockIcon.size-3.shrink-0.text-amber-500(
                v-if="tag.isRequired"
                :title="$t('tasksTable.requiredTagHint')"
              )

    .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
      span.text-muted-foreground.text-xs {{ $t("taskDetails.attachments") }}
      p.text-muted-foreground.text-xs(v-if="canCompose")
        | {{ $t("taskDetails.attachmentsHint") }}
      AttachmentList(
        :attachments="task.attachments"
        @open="$emit('open-attachment', $event)"
      )

    .flex.flex-col.gap-1(
      v-if="detailsTab === 'general' && task.holds.length"
      data-testid="task-holds"
    )
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-xs {{ $t("taskDetails.holds") }}
        span.text-muted-foreground.text-xs
          | {{ $t("taskDetails.pendingHolds", { count: task.holds.length }) }}
      .flex.flex-col.gap-1
        .flex.flex-col.rounded-md.border.px-2.py-1.text-xs(
          v-for="hold in task.holds"
          :key="`${hold.plugin}:${hold.key}`"
          class="gap-0.5"
          :title="$t('taskDetails.heldSince', { at: formatTimestamp(hold.createdAt) })"
        )
          .flex.items-center.gap-2
            PauseIcon.text-muted-foreground.size-3.shrink-0
            span.truncate.font-medium {{ hold.pluginName || hold.plugin }}
            span.text-muted-foreground.ml-auto.shrink-0.font-mono {{ hold.key }}
            Button.shrink-0(
              v-if="canCloseOut"
              type="button"
              variant="ghost"
              size="icon-sm"
              :title="$t('taskDetails.releaseHold')"
              :aria-label="$t('taskDetails.releaseHoldAria', { plugin: hold.pluginName || hold.plugin, key: hold.key })"
              @click="emit('release-hold', hold)"
              data-testid="release-hold"
            )
              LockOpenIcon.size-3
          span.text-muted-foreground.whitespace-pre-wrap(v-if="hold.reason")
            | {{ hold.reason }}

    TaskReferences(
      v-if="detailsTab === 'general'"
      :task="task"
      @updated="$emit('reference-change', $event)"
    )

    .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
      .flex.items-center.justify-between
        span.text-muted-foreground.text-xs {{ $t("taskColumn.links") }}
        Button(
          type="button"
          variant="ghost"
          size="icon-sm"
          :title="$t('taskLinks.title')"
          :aria-label="$t('taskLinks.title')"
          @click="$emit('edit-links')"
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
      span.text-muted-foreground.text-xs {{ $t("pageTitle.usage") }}
      span.text-muted-foreground.text-sm(v-if="!isStarted || sessionDeleted") —
      .flex.flex-col.gap-1.text-xs(v-else)
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground {{ $t("taskDetails.usage.title") }}
          Skeleton.h-3.w-24(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else)
            | {{ $t("usage.kpis.inOutCounts", { in: formatTokens(taskUsage.inputTokens), out: formatTokens(taskUsage.outputTokens) }) }}
        .flex.items-baseline.justify-between.gap-3(v-if="cacheTokens > 0")
          span.text-muted-foreground(
            :title="$t('taskDetails.usage.cacheHint')"
          ) {{ $t("taskDetails.usage.cache") }}
          span.font-medium.tabular-nums
            | {{ $t("taskDetails.usage.cacheDetail", { read: formatTokens(Number(taskUsage?.cacheReadTokens ?? 0)), write: formatTokens(Number(taskUsage?.cacheWriteTokens ?? 0)) }) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground {{ $t("taskDetails.usage.totalCost") }}
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ formatCost(taskUsage.cost) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(:title="$t('taskDetails.usage.ttcHint')") {{ $t("taskDetails.usage.ttc") }}
          Skeleton.h-3.w-16(v-if="!task")
          span.font-medium.tabular-nums(v-else) {{ formatDuration(ttcMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            :title="$t('taskDetails.usage.activeHint')"
          ) {{ $t("taskDetails.usage.active") }}
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ formatDuration(activeMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(:title="$t('taskDetails.usage.toolHint')") {{ $t("taskDetails.usage.tool") }}
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ measured(toolTotalMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            :title="$t('taskDetails.usage.thinkingHint')"
          ) {{ $t("taskDetails.usage.thinking") }}
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ measured(thinkingTotalMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            :title="$t('taskDetails.usage.modelWaitHint')"
          ) {{ $t("taskDetails.usage.modelWait") }}
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ measured(modelWaitTotalMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            :title="$t('taskDetails.usage.speedHint')"
          ) {{ $t("taskDetails.usage.speed") }}
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ generationLabel }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground {{ $t("taskDetails.usage.workspace") }}
          Skeleton.h-3.w-16(v-if="taskDiskLoading")
          span.font-medium.tabular-nums(
            v-else-if="taskDiskBytes !== undefined"
          ) {{ formatBytes(taskDiskBytes) }}
          span.font-medium.tabular-nums(v-else) —

    .flex.flex-col.gap-1(v-if="detailsTab === 'decisions'")
      DecisionLedger(compact :task-id="task.id")

    .flex.flex-col.gap-1(v-if="detailsTab === 'comments'")
      TaskComments(
        :task-id="task.id"
        :project-id="task.projectId"
        :preview="commentsPreview"
        @read="$emit('comments-read')"
        @open-attachment="$emit('open-attachment', $event)"
      )

    .flex.flex-col.gap-1(v-if="detailsTab === 'plan'")
      span.text-muted-foreground.text-xs {{ $t("taskProgress.title") }}
      TaskProgressPanel(
        :progress="task.progress"
        :state="task.state"
        :size="task.size"
        :can-remind="canCompose"
        @remind="$emit('remind-plan')"
      )

    .flex.flex-col.gap-1(v-if="detailsTab === 'artifacts'")
      span.text-muted-foreground.text-xs {{ $t("taskDetails.artifacts") }}
      TaskArtifacts(
        :artifacts="task.artifacts"
        @open="$emit('open-media', $event)"
      )

    .flex.flex-col.gap-2(v-if="detailsTab === 'logs'")
      .flex.items-center.gap-1
        button.rounded-md.px-2.py-1.text-xs.font-medium(
          v-for="option in LOGS_TABS"
          :key="option.value"
          type="button"
          :class="logsTab === option.value ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/60'"
          @click="logsTab = option.value"
          :data-testid="`logs-tab-${option.value}`"
        ) {{ $t(option.labelKey) }}
      TaskLogs(v-if="logsTab === 'activity'" :logs="task.logs")
      TaskToolCalls(
        v-else
        v-model:names="toolFilter"
        :task-id="task.id"
        :preview="toolCallsPreview"
      )

    .flex.flex-col.gap-3(v-if="detailsTab === 'system'")
      TaskSystemPanel(
        :task-id="task.id"
        :preview="toolsPreview"
        :sections="systemPrompt"
        @filter="filterToolCalls"
      )

    .bg-border.h-px(v-if="detailsTab === 'general'")
    .flex.flex-col.gap-2(v-if="detailsTab === 'general'")
      DropdownMenu
        DropdownMenuTrigger(as-child)
          Button(
            type="button"
            variant="outline"
            :title="task.projectId ? $t('taskDetails.createRelated') : $t('taskDetails.createRelatedBlocked')"
            :disabled="actionBusy || !task.projectId"
            data-testid="task-new-related"
          )
            PlusIcon(data-icon="inline-start")
            span {{ $t("nav.newTask") }}
        DropdownMenuContent.w-72(align="start")
          DropdownMenuLabel {{ $t("taskDetails.relateTo", { id: task.id }) }}
          DropdownMenuSeparator
          DropdownMenuItem(
            v-for="option in NEW_TASK_RELATIONS"
            :key="option.value"
            :title="$t(option.descriptionKey)"
            @click="$emit('create-related', option.value)"
            :data-testid="`task-new-related-${option.value}`"
          )
            component.size-3.shrink-0(:is="newTaskRelationIcon(option.value)")
            .flex.min-w-0.flex-col
              span.font-medium {{ $t(option.labelKey) }}
              span.text-muted-foreground.text-xs {{ $t(option.descriptionKey) }}
      template(v-if="task.archived")
        Button(
          variant="outline"
          @click="$emit('unarchive')"
          :disabled="actionBusy"
        )
          ArchiveRestoreIcon(data-icon="inline-start")
          span {{ $t("tasksTable.unarchive") }}
        Button(
          v-if="task.sessionId && !task.sessionDeletedAt"
          variant="outline"
          @click="$emit('cleanup-session')"
          :disabled="actionBusy"
        )
          EraserIcon(data-icon="inline-start")
          span {{ $t("tasksTable.cleanUpSession") }}
        Button.text-destructive(
          variant="outline"
          @click="$emit('delete')"
          :disabled="actionBusy"
        )
          Trash2Icon(data-icon="inline-start")
          span {{ $t("common.delete") }}
      template(v-else)
        Button(
          v-if="isBacklog && !isPostponed"
          @click="$emit('queue')"
          :disabled="actionBusy"
        )
          PlayIcon(data-icon="inline-start")
          span {{ $t("taskDetails.queue") }}
        Button(
          v-if="isRunning && task.runtime?.runInProgress"
          variant="outline"
          @click="$emit('stop')"
          :disabled="stopping"
        )
          SquareIcon(data-icon="inline-start")
          span {{ $t("taskDetails.stop") }}
        Button(
          v-if="canRestartSession"
          variant="outline"
          :title="restartTitle"
          @click="$emit('restart-session')"
          :disabled="actionBusy || restartBlocked"
          data-test="restart-session"
        )
          RefreshCwIcon(data-icon="inline-start")
          span {{ $t("taskDetails.restartSession") }}
        Button(
          v-if="isQueued"
          variant="outline"
          @click="$emit('dequeue')"
          :disabled="actionBusy"
        )
          Undo2Icon(data-icon="inline-start")
          span {{ $t("tasks.action.moveToBacklog") }}
        Button(
          v-if="canMoveToBacklog"
          variant="outline"
          @click="$emit('reopen', false)"
          :disabled="actionBusy"
        )
          InboxIcon(data-icon="inline-start")
          span {{ $t("tasks.action.moveToBacklog") }}
        Button(
          v-if="canRework"
          @click="$emit('reopen', true)"
          :disabled="actionBusy"
        )
          PlayIcon(data-icon="inline-start")
          span {{ $t("taskConsolePage.requeue") }}
        Button(
          v-if="canCloseOut"
          variant="outline"
          :title="isRunning || isInput ? $t('closeOut.blockedHint') : $t('closeOut.title')"
          @click="$emit('close-out')"
          :disabled="actionBusy || isRunning || isInput"
        )
          CheckIcon(data-icon="inline-start")
          span {{ $t("closeOut.submit") }}
        Button(
          v-if="canChangeOutcome"
          variant="outline"
          :title="$t('taskDetails.changeOutcomeHint')"
          @click="$emit('close-out')"
          :disabled="actionBusy"
        )
          ArrowLeftRightIcon(data-icon="inline-start")
          span {{ $t("taskDetails.changeOutcome") }}
        Button(
          v-if="canArchive"
          variant="outline"
          @click="$emit('archive')"
          :disabled="actionBusy"
        )
          ArchiveIcon(data-icon="inline-start")
          span {{ $t("tags.archive") }}
        Button.text-destructive(
          variant="outline"
          @click="$emit('delete')"
          :disabled="actionBusy"
        )
          Trash2Icon(data-icon="inline-start")
          span {{ $t("common.delete") }}
        p.text-muted-foreground.text-xs(v-if="isBacklog && isPostponed")
          | {{ $t("taskDetails.alreadyRan") }}
        p.text-muted-foreground.text-xs(v-else-if="isBacklog")
          | {{ $t("taskDetails.editableInBacklog") }}
        p.text-muted-foreground.text-xs(v-else-if="!isBacklog")
          | {{ $t("taskDetails.followUpOrClose") }}

      TaskHistoryExport(:task="task")
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import AttachmentList from "@/components/attachments/AttachmentList.vue";
import DecisionLedger from "@/components/knowledge/DecisionLedger.vue";
import TaskArtifacts from "@/components/tasks/TaskArtifacts.vue";
import TaskBadges from "@/components/tasks/TaskBadges.vue";
import TaskComments from "@/components/tasks/TaskComments.vue";
import TaskHistoryExport from "@/components/tasks/TaskHistoryExport.vue";
import TaskLogs from "@/components/tasks/TaskLogs.vue";
import TaskProgressPanel from "@/components/tasks/TaskProgressPanel.vue";
import TaskReferences from "@/components/tasks/TaskReferences.vue";
import TaskSystemPanel from "@/components/tasks/TaskSystemPanel.vue";
import TaskToolCalls from "@/components/tasks/TaskToolCalls.vue";
import { Button } from "@/components/ui/button";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuFilter,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useTaskDetails } from "@/composables/useTaskDetails";
import {
  formatBytes,
  formatCost,
  formatDuration,
  formatRate,
  formatTimestamp,
  formatTokens,
} from "@/lib/format";
import type { LightboxItem } from "@/lib/lightbox";
import type { LeadCandidate, ProjectTeamLink } from "@/lib/projectAccess";
import type { DetailsPreview } from "@/lib/task";
import {
  NEW_TASK_RELATIONS,
  newTaskRelationIcon,
  tagColor,
  tagStyle,
  TASK_SIZES,
  taskStateClassFor,
  taskStateIcon,
  taskStateKeyFor,
  type NewTaskRelation,
} from "@/lib/task";
import { formatArgValue, parseArgsJSON } from "@/lib/taskTemplates";
import { tickUsage } from "@/lib/usage";
import {
  TaskPriority,
  TaskSize,
  TaskState,
  type Agent,
  type Attachment,
  type Project,
  type SystemPromptSection,
  type Tag,
  type Task,
  type TaskHold,
  type TaskUsage,
} from "@cherrypi/sdk";
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  ArrowLeftRightIcon,
  ArrowRightIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  CogIcon,
  EraserIcon,
  ExternalLinkIcon,
  FileCodeIcon,
  InboxIcon,
  LockIcon,
  LockOpenIcon,
  PauseIcon,
  PencilIcon,
  PinIcon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  SquareIcon,
  Trash2Icon,
  Undo2Icon,
  XIcon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const { t } = useI18n();

const { width, resizing, minWidth, maxWidth, startResize, resizeByKey } =
  useTaskDetails();

function onResizeKey(event: KeyboardEvent) {
  if (resizeByKey(event)) event.preventDefault();
}

type SavingState = {
  assignee: boolean;
  priority: boolean;
  size: boolean;
  project: boolean;
  tags: boolean;
  links: boolean;
  teams: boolean;
  leader: boolean;
  subscription: boolean;
  favorite: boolean;
  model: boolean;
};

const props = defineProps<{
  open: boolean;
  task?: Task;
  tab?: string;
  saving: SavingState;
  assigneeOptions: Agent[];
  projectOptions: Project[];
  projectTeams: ProjectTeamLink[];
  leadCandidates: LeadCandidate[];
  unattachedTags: Tag[];
  taskUsage?: TaskUsage;
  liveTiming?: { toolRunning: boolean; modelWaiting: boolean };
  taskDiskBytes?: bigint;
  taskDiskLoading: boolean;
  systemPrompt: SystemPromptSection[];
  actionBusy: boolean;
  stopping: boolean;
  canCompose: boolean;

  preview?: DetailsPreview;
}>();

const emit = defineEmits<{
  (e: "update:tab", value: string): void;
  (e: "assignee-change", value: unknown): void;
  (e: "priority-change", value: unknown): void;
  (e: "size-change", value: unknown): void;
  (e: "project-change", value: unknown): void;
  (e: "toggle-team", id: number): void;
  (e: "set-leader", id: string): void;
  (e: "toggle-tag", tag: Tag): void;
  (e: "toggle-subscription"): void;
  (e: "reference-change", task: Task): void;
  (e: "open-attachment", attachment: Attachment): void;
  (e: "remind-plan"): void;
  (e: "open-media", item: LightboxItem): void;
  (e: "comments-read"): void;
  (e: "edit-links"): void;
  (e: "queue"): void;
  (e: "dequeue"): void;
  (e: "restart-session"): void;
  (e: "stop"): void;
  (e: "reopen", toQueue: boolean): void;
  (e: "close-out"): void;
  (e: "release-hold", hold: TaskHold): void;
  (e: "archive"): void;
  (e: "unarchive"): void;
  (e: "cleanup-session"): void;
  (e: "delete"): void;
  (e: "create-related", relation: NewTaskRelation): void;
}>();

const commentsPreview = computed(() =>
  props.preview
    ? (props.preview.comments ?? { comments: [], unread: 0 })
    : undefined,
);
const toolsPreview = computed(() =>
  props.preview
    ? (props.preview.tools ?? { tools: [], recorded: false })
    : undefined,
);
const toolCallsPreview = computed(() =>
  props.preview
    ? (props.preview.toolCalls ?? { calls: [], total: 0, names: [] })
    : undefined,
);

const { agents, usernameFor, agentIdForActor } = useAgents();
const { active: allProjects } = useProjects();

const detailsTabs = computed(() => {
  const progress = props.task?.progress;
  const plan =
    progress && progress.total > 0
      ? `${t("taskDetails.tabs.plan")} · ${progress.done}/${progress.total}`
      : t("taskDetails.tabs.plan");
  return [
    { value: "general", label: t("taskDetails.tabs.general") },
    { value: "comments", label: t("taskDetails.tabs.comments") },
    { value: "decisions", label: t("taskDetails.tabs.decisions") },
    { value: "plan", label: plan },
    { value: "artifacts", label: t("taskDetails.artifacts") },
    { value: "system", label: t("taskDetails.tabs.system") },
    { value: "logs", label: t("taskDetails.tabs.logs") },
  ];
});

const LOGS_TABS = [
  { value: "activity", labelKey: "taskDetails.logsTab.activity" },
  { value: "tool-calls", labelKey: "taskDetails.logsTab.toolCalls" },
] as const;
const logsTab = ref<"activity" | "tool-calls">("activity");

const toolFilter = ref<string[]>([]);

function filterToolCalls(name: string) {
  toolFilter.value = [name];
  logsTab.value = "tool-calls";
  detailsTab.value = "logs";
}

const TAB_ALIASES: Record<string, string> = { prompts: "system" };

const detailsTab = computed({
  get: () => {
    const tab = props.tab ?? "general";
    return TAB_ALIASES[tab] ?? tab;
  },
  set: (value: string) => emit("update:tab", value),
});

const templateArgs = computed(() =>
  Object.entries(parseArgsJSON(props.task?.template?.argsJson ?? "")),
);

const PRIORITIES = [
  { value: TaskPriority.LOW, labelKey: "taskPriority.low" },
  { value: TaskPriority.NORMAL, labelKey: "taskPriority.normal" },
  { value: TaskPriority.HIGH, labelKey: "taskPriority.high" },
  { value: TaskPriority.CRITICAL, labelKey: "taskPriority.critical" },
];

const AUTO = "__auto__";

const SIZES = TASK_SIZES;

const isBacklog = computed(() => props.task?.state === TaskState.BACKLOG);

const hasStarted = computed(() => Boolean(props.task?.startedAt));
const isPostponed = computed(() => isBacklog.value && hasStarted.value);

const isEditableBacklog = computed(() => isBacklog.value && !hasStarted.value);
const isQueued = computed(() => props.task?.state === TaskState.QUEUED);
const isRunning = computed(() => props.task?.state === TaskState.RUNNING);
const isInput = computed(() => props.task?.state === TaskState.INPUT);
const isStarted = computed(() => Boolean(props.task?.sessionId));

const canEditAssignee = computed(() => !isRunning.value && !isInput.value);

const isActive = computed(() => {
  const state = props.task?.state;
  return (
    state === TaskState.RUNNING ||
    state === TaskState.INPUT ||
    state === TaskState.SETTLED ||
    state === TaskState.FAILED
  );
});
const sessionDeleted = computed(() => Boolean(props.task?.sessionDeletedAt));

const ttcCounts = computed(() => {
  const state = props.task?.state;
  return (
    state !== undefined &&
    state !== TaskState.BACKLOG &&
    state !== TaskState.COMPLETED &&
    state !== TaskState.CANCELLED
  );
});

const liveNow = ref(Date.now());
let liveTimer: ReturnType<typeof setInterval> | undefined;
function stopLiveClock() {
  if (liveTimer !== undefined) {
    clearInterval(liveTimer);
    liveTimer = undefined;
  }
}
const ttcFetchedAt = ref(Date.now());
watch(
  [() => props.task?.ttcMs, ttcCounts],
  () => {
    ttcFetchedAt.value = Date.now();
    liveNow.value = ttcFetchedAt.value;
    stopLiveClock();
    if (ttcCounts.value) {
      liveTimer = setInterval(() => {
        liveNow.value = Date.now();
      }, 1000);
    }
  },
  { immediate: true },
);
onBeforeUnmount(stopLiveClock);
const ttcMs = computed(() => {
  const task = props.task;
  if (!task) return 0;
  const base = Number(task.ttcMs);
  if (!ttcCounts.value) return base;
  return base + Math.max(0, liveNow.value - ttcFetchedAt.value);
});

const timingMeasured = computed(
  () => Number(props.taskUsage?.activeMs ?? 0) > 0,
);

const cacheTokens = computed(
  () =>
    Number(props.taskUsage?.cacheReadTokens ?? 0) +
    Number(props.taskUsage?.cacheWriteTokens ?? 0),
);
const toolMs = computed(() => Number(props.taskUsage?.toolMs ?? 0));
const thinkingMs = computed(() => Number(props.taskUsage?.thinkingMs ?? 0));

const usageFetchedAt = ref(Date.now());
const usageCounts = computed(
  () =>
    props.task?.runtime?.runInProgress === true &&
    props.task?.runtime?.waitingForInput !== true,
);
const usageExtraMs = computed(() =>
  usageCounts.value ? Math.max(0, liveNow.value - usageFetchedAt.value) : 0,
);

const liveUsage = computed(() =>
  tickUsage(
    {
      activeMs: Number(props.taskUsage?.activeMs ?? 0),
      toolMs: toolMs.value,
      thinkingMs: thinkingMs.value,
      modelWaitMs: modelWaitMs.value,
    },
    usageExtraMs.value,
    {
      toolRunning: props.liveTiming?.toolRunning === true,
      modelWaiting: props.liveTiming?.modelWaiting === true,
    },
  ),
);
const activeMs = computed(() => liveUsage.value.activeMs);
const toolTotalMs = computed(() => liveUsage.value.toolMs);
const thinkingTotalMs = computed(() => liveUsage.value.thinkingMs);
const modelWaitTotalMs = computed(() => liveUsage.value.modelWaitMs);
watch(
  [() => props.taskUsage, usageCounts],
  () => {
    usageFetchedAt.value = Date.now();
    liveNow.value = usageFetchedAt.value;
  },
  { immediate: true },
);

function measured(ms: number): string {
  return timingMeasured.value ? formatDuration(ms) : "—";
}
const modelWaitMs = computed(() => Number(props.taskUsage?.modelWaitMs ?? 0));
const generationLabel = computed(() => {
  const usage = props.taskUsage;
  if (!usage) return "—";
  const seconds = thinkingMs.value / 1000;
  if (seconds <= 0) return "—";
  const rate = Number(usage.outputTokens) / seconds;
  return rate > 0 ? formatRate(rate) : "—";
});

const canMoveToBacklog = computed(() => {
  const state = props.task?.state;
  return (
    state === TaskState.FAILED ||
    state === TaskState.SETTLED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
});

const canRestartSession = computed(
  () => Boolean(props.task?.sessionId) && !sessionDeleted.value,
);

const restartBlocked = computed(
  () => Boolean(props.task?.runtime?.runInProgress) || isInput.value,
);
const restartTitle = computed(() => {
  if (props.task?.runtime?.runInProgress) {
    return "Stop the run before restarting Pi";
  }
  if (isInput.value) {
    return "Answer or decline the questionnaire before restarting Pi";
  }
  return "Restart the runtime so it reloads the agent's model catalog";
});

const canRework = computed(
  () => props.task?.state === TaskState.FAILED && !sessionDeleted.value,
);

const canCloseOut = computed(() => {
  const state = props.task?.state;
  return (
    !!state &&
    state !== TaskState.RUNNING &&
    state !== TaskState.INPUT &&
    state !== TaskState.COMPLETED &&
    state !== TaskState.CANCELLED &&
    Boolean(props.task?.canCloseOut)
  );
});

const isClosedOut = computed(() => {
  const state = props.task?.state;
  return state === TaskState.COMPLETED || state === TaskState.CANCELLED;
});
const canChangeOutcome = computed(
  () => isClosedOut.value && Boolean(props.task?.canCloseOut),
);
const canArchive = computed(
  () => isClosedOut.value && Boolean(props.task?.canCloseOut),
);

const assigneeChoice = computed(() => {
  const actorId = props.task?.assigneeActorId;
  if (!actorId) return AUTO;

  return findAgent(actorId)?.id ?? agentIdForActor(actorId);
});
const priorityChoice = computed(
  () => props.task?.priority || TaskPriority.NORMAL,
);
const sizeChoice = computed(() => props.task?.size || TaskSize.MEDIUM);
const projectChoice = computed(() => props.task?.projectId ?? "");
const teamIds = computed(() =>
  (props.task?.teams ?? []).map((team) => team.id),
);

const NO_LEADER = "__none__";
const leaderChoice = computed(() => props.task?.leader?.actorId ?? NO_LEADER);

const canRemoveLead = computed(() => {
  const state = props.task?.state;
  return state === TaskState.BACKLOG || state === TaskState.CANCELLED;
});
function pickLeader(value: unknown): void {
  if (typeof value !== "string") return;
  emit("set-leader", value === NO_LEADER ? "" : value);
}
const teamSearch = ref("");
const leadSearch = ref("");
const filteredProjectTeams = computed(() => {
  const query = teamSearch.value.trim().toLowerCase();
  if (!query) return props.projectTeams;
  return props.projectTeams.filter((team) =>
    team.team_name.toLowerCase().includes(query),
  );
});
const filteredLeadCandidates = computed(() => {
  const query = leadSearch.value.trim().toLowerCase();
  if (!query) return props.leadCandidates;
  return props.leadCandidates.filter((candidate) =>
    `${candidate.name} ${candidate.username}`.toLowerCase().includes(query),
  );
});
const projectComboboxOptions = computed<ComboboxOption[]>(() =>
  props.projectOptions.map((project) => ({
    value: project.id,
    label: project.name,
    description: project.prefix ? `#${project.prefix}` : undefined,
  })),
);

function findProject(id: string): Project | undefined {
  return (
    props.projectOptions.find((project) => project.id === id) ??
    allProjects.value.find((project) => project.id === id)
  );
}

function findAgent(id: string): Agent | undefined {
  return (
    props.assigneeOptions.find(
      (item) => item.id === id || item.actorId === id,
    ) ?? agents.value.find((item) => item.id === id || item.actorId === id)
  );
}

function projectName(id: string): string {
  return findProject(id)?.name ?? id.slice(0, 8);
}

function projectSlug(id: string): string {
  return findProject(id)?.slug ?? "";
}

function agentName(id: string): string {
  const item = findAgent(id);
  return item?.displayName || item?.username || id.slice(0, 8);
}

function agentUsername(id: string): string {
  return findAgent(id)?.username ?? usernameFor(id);
}

function agentAvatar(id: string): string {
  return toSvg(id, 20);
}
</script>

<style lang="postcss" scoped></style>

<i18n lang="yaml">
en:
  taskDetails:
    resize: "Resize task details"
    holdsSlot: "holds the agent's slot"
    noProject: "No project"
    mode: "Mode"
    taskId: "Task id"
    origin: "Origin"
    originDescription: "The task this one was created from."
    creator: "Creator"
    leader: "Leader"
    noLeader: "No leader"
    changeLeader: "Change leader"
    noArguments: "No arguments."
    attachments: "Attachments"
    attachmentsHint: "Attach files to a message below."
    holds: "Holds"
    usage:
      title: "Token usage"
      cache: "Cache"
      cacheDetail: "{read} read · {write} written"
      cacheHint: "Input served from the provider's prompt cache and input written into it. Priced by the model's cache rates and already part of the total cost, so they are not counted again in the input/output line."
      totalCost: "Total cost"
      ttc: "Time till completion"
      active: "Active time"
      tool: "Tool call time"
      thinking: "Thinking time"
      modelWait: "Model wait"
      speed: "Generation speed"
      workspace: "Workspace size"
      ttcHint: "Time from the first queue to completion. Keeps counting while the task is open and stops in backlog."
      activeHint: "The agent's work time: idle gaps and user-input waits excluded, tool calls included."
      toolHint: "The time the agent spent running tools: file edits, shell commands, searches and the like."
      thinkingHint: "The model's own working time: active time minus tool calls and provider waits."
      modelWaitHint: "The time spent waiting for a provider response."
      speedHint: "Output tokens over the time the model spent generating them: thinking time."
    artifacts: "Artifacts"
    queue: "Queue task"
    stop: "Stop run"
    restartSession: "Restart Pi session"
    changeOutcome: "Change outcome"
    alreadyRan: "This task already ran. Send a follow-up to queue it again, or close it out."
    editableInBacklog: "You can edit the prompt, project, and model in Backlog."
    followUpOrClose: "Send a follow-up to run the task again, or close it out."
    changeAssignee: "Change assignee"
    changePriority: "Change priority"
    changeSize: "Change size"
    changeProject: "Change project"
    automatedHint: "Authored by the system or a plugin: no leader and no user-input tools"
    originHint: "Set once when the task was created and never changed"
    editTeams: "Edit owning teams"
    removeTag: "Remove tag"
    releaseHold: "Release this hold"
    changeOutcomeHint: "Switch between completed and cancelled"
    tabs:
      general: "General"
      comments: "Comments"
      decisions: "Decisions"
      plan: "Plan"
      system: "System"
      logs: "Logs"
    logsTab:
      activity: "Activity"
      toolCalls: "Tool calls"
    relateTo: "Relate it to {id}"
    createRelated: "Create a task related to this one"
    createRelatedBlocked: "A related task needs this task to belong to a project"
    teamFallback: "Team {id}"
    createdFromTemplate: "Created from template version {version}"
    argumentsCount: "{count} argument | {count} arguments"
    pendingHolds: "{count} pending"
    heldSince: "Held since {at}"
    releaseHoldAria: "Release the {plugin} hold {key}"
    deletedTemplate: "Deleted template"
  taskConsole:
    copyFailed: "Copy failed"
  taskView:
    files: "Files"
    task: "Task"
  newTaskRelation:
    informational: "Informational"
    informationalHint: "The new task links to this one."
    blockedBy: "Blocked by this task"
    blockedByHint: "The new task waits for this one to be completed."
    blocks: "Blocks this task"
    blocksHint: "This task waits for the new one to be completed."
    supersedes: "Supersedes this task"
    supersedesHint: "The new task replaces this one and cancels it."
ru:
  taskDetails:
    resize: "Изменить ширину подробностей"
    holdsSlot: "занимает слот агента"
    noProject: "Без проекта"
    mode: "Режим"
    taskId: "Идентификатор задачи"
    origin: "Источник"
    originDescription: "Задача, из которой создана эта."
    creator: "Создатель"
    leader: "Руководитель"
    noLeader: "Без руководителя"
    changeLeader: "Изменить руководителя"
    noArguments: "Аргументов нет."
    attachments: "Вложения"
    attachmentsHint: "Прикрепляйте файлы к сообщению ниже."
    holds: "Удержания"
    usage:
      title: "Использование токенов"
      cache: "Кэш"
      cacheDetail: "{read} прочитано · {write} записано"
      cacheHint: "Вход, прочитанный из кэша провайдера, и вход, записанный в него. Считается по ценам кэша модели и уже входит в общую стоимость, поэтому не дублируется в строке вход/выход."
      totalCost: "Общая стоимость"
      ttc: "Время до завершения"
      active: "Активное время"
      tool: "Время вызовов инструментов"
      thinking: "Время размышления"
      modelWait: "Ожидание модели"
      speed: "Скорость генерации"
      workspace: "Размер рабочего пространства"
      ttcHint: "Время от первой постановки в очередь до завершения. Считается, пока задача открыта, и останавливается в бэклоге."
      activeHint: "Рабочее время агента: без простоев и ожидания пользователя, включая вызовы инструментов."
      toolHint: "Время, которое агент провёл за вызовами инструментов: правки файлов, команды оболочки, поиск и т. п."
      thinkingHint: "Собственное рабочее время модели: активное время минус вызовы инструментов и ожидание провайдера."
      modelWaitHint: "Время, потраченное на ожидание ответа провайдера."
      speedHint: "Выходные токены за время генерации: время размышления."
    artifacts: "Артефакты"
    queue: "Поставить в очередь"
    stop: "Остановить запуск"
    restartSession: "Перезапустить сеанс Pi"
    changeOutcome: "Изменить результат"
    alreadyRan: "Задача уже выполнялась. Отправьте уточнение, чтобы поставить её снова в очередь, или закройте её."
    editableInBacklog: "В бэклоге можно изменить промпт, проект и модель."
    followUpOrClose: "Отправьте уточнение, чтобы запустить задачу снова, или закройте её."
    changeAssignee: "Изменить исполнителя"
    changePriority: "Изменить приоритет"
    changeSize: "Изменить размер"
    changeProject: "Изменить проект"
    automatedHint: "Создана системой или плагином: без руководителя и без инструментов ввода"
    originHint: "Задаётся при создании задачи и больше не меняется"
    editTeams: "Изменить команды-владельцы"
    removeTag: "Убрать тег"
    releaseHold: "Снять это удержание"
    changeOutcomeHint: "Переключить между «выполнено» и «отменено»"
    tabs:
      general: "Общее"
      comments: "Комментарии"
      decisions: "Решения"
      plan: "План"
      system: "Система"
      logs: "Журналы"
    logsTab:
      activity: "Активность"
      toolCalls: "Вызовы инструментов"
    relateTo: "Связать с {id}"
    createRelated: "Создать задачу, связанную с этой"
    createRelatedBlocked: "Связанной задаче нужно, чтобы эта задача принадлежала проекту"
    teamFallback: "Команда {id}"
    createdFromTemplate: "Создано из версии шаблона {version}"
    argumentsCount: "{count} аргумент | {count} аргумента | {count} аргументов"
    pendingHolds: "{count} в ожидании"
    heldSince: "Удерживается с {at}"
    releaseHoldAria: "Снять удержание {plugin} {key}"
    deletedTemplate: "Удалённый шаблон"
  taskConsole:
    copyFailed: "Не удалось скопировать"
  taskView:
    files: "Файлы"
    task: "Задача"
  newTaskRelation:
    informational: "Информационная"
    informationalHint: "Новая задача ссылается на эту."
    blockedBy: "Заблокирована этой задачей"
    blockedByHint: "Новая задача ждёт завершения этой."
    blocks: "Блокирует эту задачу"
    blocksHint: "Эта задача ждёт завершения новой."
    supersedes: "Заменяет эту задачу"
    supersedesHint: "Новая задача заменяет эту и отменяет её."
</i18n>
