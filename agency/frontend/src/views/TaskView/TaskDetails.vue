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
aside#task-details.bg-card.absolute.top-0.right-0.bottom-0.flex.w-80.shrink-0.flex-col.border-l(
  v-if="open && task"
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
        :title="`${task.unreadComments} unread comment${task.unreadComments === 1 ? '' : 's'}`"
        data-testid="comments-unread"
      ) {{ task.unreadComments > 99 ? "99+" : task.unreadComments }}
  .flex.flex-1.flex-col.gap-4.overflow-y-auto.p-4
    .flex.flex-col.gap-1(v-if="task.badges.length")
      span.text-muted-foreground.text-xs Badges
      TaskBadges(:badges="task.badges" data-testid="task-details-badges")
    .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs Assignee

        Select(
          v-if="canEditAssignee"
          :model-value="assigneeChoice"
          @update:model-value="$emit('assignee-change', $event)"
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
          @update:model-value="$emit('priority-change', $event)"
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
          @update:model-value="$emit('size-change', $event)"
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

        Combobox(
          v-if="isEditableBacklog"
          placeholder="Select project"
          size="sm"
          empty-text="No matching project."
          title="Change project"
          aria-label="Project"
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
          span.text-muted-foreground.text-sm(v-else) No project
      .flex.flex-col(v-if="!task.interactive" class="gap-0.5")
        span.text-muted-foreground.text-xs Mode
        span.inline-flex.w-fit.items-center.gap-1.rounded-full.border.px-2.text-xs(
          class="py-0.5"
          title="The agent cannot show questionnaires or extension dialogs"
          data-testid="task-non-interactive-detail"
        )
          MessageSquareOffIcon.size-3
          | Non-interactive

    .grid.grid-cols-2.gap-3(v-if="detailsTab === 'general'")
      .flex.flex-col(class="gap-0.5")
        span.text-muted-foreground.text-xs Task id
        span.font-mono.text-xs.break-all(:title="task.id") {{ task.id }}
      .flex.flex-col(v-if="task.parentTaskId" class="gap-0.5")
        span.text-muted-foreground.text-xs Parent task
        RouterLink.font-mono.text-xs(
          class="hover:underline"
          title="Sub-task of this task"
          :to="{ name: 'task', params: { id: task.parentTaskId } }"
          data-testid="task-parent-breadcrumb"
        ) {{ task.parentTaskId }}
      .flex.flex-col(
        v-if="task.originTaskId"
        class="gap-0.5"
        data-testid="task-origin"
      )
        span.text-muted-foreground.text-xs Origin
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
          title="Set once when the task was created and never changed"
        ) The task this one was created from.
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
              DropdownMenuFilter(
                v-model="teamSearch"
                placeholder="Search teams…"
                auto-focus
              )
              DropdownMenuSeparator
              p.text-muted-foreground.px-2.py-1.text-xs(
                v-if="!filteredProjectTeams.length"
              )
                | No teams in this project
              DropdownMenuCheckboxItem(
                v-for="team in filteredProjectTeams"
                :key="team.team_id"
                :model-value="teamIds.includes(team.team_id)"
                @update:model-value="$emit('toggle-team', team.team_id)"
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
            button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
              v-if="canRemoveLead"
              type="button"
              class="p-0.5 hover:opacity-100"
              title="Remove lead"
              @click="$emit('toggle-lead', lead.actorId)"
              :disabled="saving.leads"
            )
              XIcon.size-3
          span.text-muted-foreground.text-xs(v-if="!task.leads.length")
            | No lead yet — a task needs someone responsible before it runs
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
              DropdownMenuFilter(
                v-model="leadSearch"
                placeholder="Search members…"
                auto-focus
              )
              DropdownMenuSeparator
              p.text-muted-foreground.px-2.py-1.text-xs(
                v-if="!filteredLeadCandidates.length"
              )
                | No members to pick
              DropdownMenuCheckboxItem(
                v-for="candidate in filteredLeadCandidates"
                :key="candidate.actor_id"
                :model-value="leadIds.includes(candidate.actor_id)"
                @update:model-value="$emit('toggle-lead', candidate.actor_id)"
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
        :title="`Created from template version ${Number(task.template.version)}`"
      )
        FileCodeIcon.size-3.shrink-0
        span.min-w-0.truncate.text-xs
          strong.font-semibold
            | {{ task.template.templateName || "Deleted template" }}
          span.text-muted-foreground.ml-1
            | v{{ Number(task.template.version) }} ·
            | {{ templateArgs.length }} argument{{ templateArgs.length === 1 ? "" : "s" }}
        ChevronRightIcon.ml-auto.size-3.shrink-0.transition-transform(
          :class="open ? 'rotate-90' : ''"
        )
      DisclosurePanel.mt-1.flex.flex-col.gap-1.rounded-md.border.px-2(class="py-1.5")
        p.text-muted-foreground.text-xs(v-if="!templateArgs.length")
          | No arguments.
        .grid.grid-cols-1.gap-1(v-else)
          .grid.items-baseline.gap-2(
            v-for="[key, value] in templateArgs"
            :key="key"
            class="grid-cols-[5.5rem_1fr]"
          )
            span.text-muted-foreground.text-xs.break-words {{ key }}
            span.text-xs.break-words.whitespace-pre-wrap {{ formatArgValue(value) }}

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
          LockIcon.size-3.shrink-0.text-amber-500(
            v-if="tag.isRequired"
            title="Only runs on agents that declare this tag"
          )
          button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
            type="button"
            class="p-0.5 hover:opacity-100"
            title="Remove tag"
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
              @click="$emit('toggle-tag', tag)"
            )
              span.size-2.shrink-0.rounded-full(
                :style="{ backgroundColor: tagColor(tag.color) }"
              )
              span {{ tag.name }}
              LockIcon.size-3.shrink-0.text-amber-500(
                v-if="tag.isRequired"
                title="Only runs on agents that declare this tag"
              )

    .flex.flex-col.gap-1(v-if="detailsTab === 'general'")
      span.text-muted-foreground.text-xs Attachments
      p.text-muted-foreground.text-xs(v-if="canCompose")
        | Attach files to a message below.
      AttachmentList(
        :attachments="task.attachments"
        @open="$emit('open-attachment', $event)"
      )

    .flex.flex-col.gap-1(
      v-if="detailsTab === 'general' && task.childTasks.length"
      data-testid="task-subtasks"
    )
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-xs Sub-tasks
        span.text-muted-foreground.text-xs
          | {{ task.pendingChildren }} open
      .flex.flex-col.gap-1
        RouterLink.flex.items-center.gap-2.rounded-md.border.px-2.py-1.text-xs(
          v-for="child in task.childTasks"
          :key="child.id"
          class="hover:bg-muted/50"
          :to="{ name: 'task', params: { id: child.id } }"
          :title="child.name || child.prompt || child.id"
        )
          component.text-muted-foreground.size-3.shrink-0(
            :is="taskStateIcon(child.state)"
          )
          span.shrink-0.font-mono {{ child.id }}
          span.truncate {{ child.name || child.prompt || "Untitled" }}
          span.text-muted-foreground.ml-auto.shrink-0
            | {{ taskStateLabel(child.state) }}

    .flex.flex-col.gap-1(
      v-if="detailsTab === 'general' && task.holds.length"
      data-testid="task-holds"
    )
      .flex.items-center.justify-between.gap-2
        span.text-muted-foreground.text-xs Holds
        span.text-muted-foreground.text-xs
          | {{ task.holds.length }} pending
      .flex.flex-col.gap-1
        .flex.flex-col.rounded-md.border.px-2.py-1.text-xs(
          v-for="hold in task.holds"
          :key="`${hold.plugin}:${hold.key}`"
          class="gap-0.5"
          :title="`Held since ${formatTimestamp(hold.createdAt)}`"
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
              title="Release this hold"
              :aria-label="`Release the ${hold.pluginName || hold.plugin} hold ${hold.key}`"
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
        span.text-muted-foreground.text-xs Links
        Button(
          type="button"
          variant="ghost"
          size="icon-sm"
          title="Edit links"
          aria-label="Edit links"
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
      span.text-muted-foreground.text-xs Usage
      span.text-muted-foreground.text-sm(v-if="!isStarted || sessionDeleted") —
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
          span.text-muted-foreground(
            title="Time from the first queue to completion. Keeps counting while the task is open and stops in backlog."
          ) Time till completion
          Skeleton.h-3.w-16(v-if="!task")
          span.font-medium.tabular-nums(v-else) {{ formatDuration(ttcMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            title="The agent's work time: idle gaps and user-input waits excluded, tool calls included."
          ) Active time
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ formatDuration(activeMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            title="The model's own working time: active time minus tool calls, model waits included."
          ) Thinking time
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ measured(thinkingTotalMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            title="The part of the thinking time spent waiting for a provider response."
          ) Model wait
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ measured(modelWaitTotalMs) }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground(
            title="Output tokens over the time the model spent generating them: thinking time minus model waits."
          ) Generation speed
          Skeleton.h-3.w-16(v-if="!taskUsage")
          span.font-medium.tabular-nums(v-else) {{ generationLabel }}
        .flex.items-baseline.justify-between.gap-3
          span.text-muted-foreground Workspace size
          Skeleton.h-3.w-16(v-if="taskDiskLoading")
          span.font-medium.tabular-nums(
            v-else-if="taskDiskBytes !== undefined"
          ) {{ formatBytes(taskDiskBytes) }}
          span.font-medium.tabular-nums(v-else) —

    .flex.flex-col.gap-1(v-if="detailsTab === 'comments'")
      TaskComments(
        :task-id="task.id"
        :project-id="task.projectId"
        @read="$emit('comments-read')"
        @open-attachment="$emit('open-attachment', $event)"
      )

    .flex.flex-col.gap-1(v-if="detailsTab === 'plan'")
      span.text-muted-foreground.text-xs Plan
      TaskProgressPanel(
        :progress="task.progress"
        :state="task.state"
        :size="task.size"
        :can-remind="canCompose"
        @remind="$emit('remind-plan')"
      )

    .flex.flex-col.gap-1(v-if="detailsTab === 'artifacts'")
      span.text-muted-foreground.text-xs Artifacts
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
        ) {{ option.label }}
      TaskLogs(v-if="logsTab === 'activity'" :logs="task.logs")
      TaskToolCalls(v-else :task-id="task.id")

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
          p.text-muted-foreground.text-xs.italic(v-if="systemPromptPreview")
            | Showing the latest versions; the task pins them when it starts.

    .bg-border.h-px(v-if="detailsTab === 'general'")
    .flex.flex-col.gap-2(v-if="detailsTab === 'general'")
      DropdownMenu
        DropdownMenuTrigger(as-child)
          Button(
            type="button"
            variant="outline"
            :title="task.projectId ? 'Create a task related to this one' : 'A related task needs this task to belong to a project'"
            :disabled="actionBusy || !task.projectId"
            data-testid="task-new-related"
          )
            PlusIcon(data-icon="inline-start")
            span New task
        DropdownMenuContent.w-72(align="start")
          DropdownMenuLabel Relate it to {{ task.id }}
          DropdownMenuSeparator
          DropdownMenuItem(
            v-for="option in NEW_TASK_RELATIONS"
            :key="option.value"
            :title="option.description"
            @click="$emit('create-related', option.value)"
            :data-testid="`task-new-related-${option.value}`"
          )
            component.size-3.shrink-0(:is="newTaskRelationIcon(option.value)")
            .flex.min-w-0.flex-col
              span.font-medium {{ option.label }}
              span.text-muted-foreground.text-xs {{ option.description }}
      template(v-if="task.archived")
        Button(
          variant="outline"
          @click="$emit('unarchive')"
          :disabled="actionBusy"
        )
          ArchiveRestoreIcon(data-icon="inline-start")
          span Unarchive
        Button(
          v-if="task.sessionId && !task.sessionDeletedAt"
          variant="outline"
          @click="$emit('cleanup-session')"
          :disabled="actionBusy"
        )
          EraserIcon(data-icon="inline-start")
          span Clean up session
        Button.text-destructive(
          variant="outline"
          @click="$emit('delete')"
          :disabled="actionBusy"
        )
          Trash2Icon(data-icon="inline-start")
          span Delete
      template(v-else)
        Button(
          v-if="isBacklog && !isPostponed"
          @click="$emit('queue')"
          :disabled="actionBusy"
        )
          PlayIcon(data-icon="inline-start")
          span Queue task
        Button(
          v-if="isRunning && task.runtime?.runInProgress"
          variant="outline"
          @click="$emit('stop')"
          :disabled="stopping"
        )
          SquareIcon(data-icon="inline-start")
          span Stop run
        Button(
          v-if="canRestartSession"
          variant="outline"
          :title="restartTitle"
          @click="$emit('restart-session')"
          :disabled="actionBusy || restartBlocked"
          data-test="restart-session"
        )
          RefreshCwIcon(data-icon="inline-start")
          span Restart Pi session
        Button(
          v-if="isQueued"
          variant="outline"
          @click="$emit('dequeue')"
          :disabled="actionBusy"
        )
          Undo2Icon(data-icon="inline-start")
          span Move to Backlog
        Button(
          v-if="canMoveToBacklog"
          variant="outline"
          @click="$emit('reopen', false)"
          :disabled="actionBusy"
        )
          InboxIcon(data-icon="inline-start")
          span Move to Backlog
        Button(
          v-if="canRework"
          @click="$emit('reopen', true)"
          :disabled="actionBusy"
        )
          PlayIcon(data-icon="inline-start")
          span Re-queue
        Button(
          v-if="canCloseOut"
          variant="outline"
          :title="isRunning || isInput ? 'Stop the run or let it settle before closing out' : 'Close out task'"
          @click="$emit('close-out')"
          :disabled="actionBusy || isRunning || isInput"
        )
          CheckIcon(data-icon="inline-start")
          span Close out
        Button(
          v-if="canChangeOutcome"
          variant="outline"
          title="Switch between completed and cancelled"
          @click="$emit('close-out')"
          :disabled="actionBusy"
        )
          ArrowLeftRightIcon(data-icon="inline-start")
          span Change outcome
        Button(
          v-if="canArchive"
          variant="outline"
          @click="$emit('archive')"
          :disabled="actionBusy"
        )
          ArchiveIcon(data-icon="inline-start")
          span Archive
        Button.text-destructive(
          variant="outline"
          @click="$emit('delete')"
          :disabled="actionBusy"
        )
          Trash2Icon(data-icon="inline-start")
          span Delete
        p.text-muted-foreground.text-xs(v-if="isBacklog && isPostponed")
          | This task already ran. Send a follow-up to queue it again, or
          | close it out.
        p.text-muted-foreground.text-xs(v-else-if="isBacklog")
          | You can edit the prompt, project, and model in Backlog.
        p.text-muted-foreground.text-xs(v-else-if="!isBacklog")
          | Send a follow-up to run the task again, or close it out.
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import AttachmentList from "@/components/attachments/AttachmentList.vue";
import TaskArtifacts from "@/components/tasks/TaskArtifacts.vue";
import TaskBadges from "@/components/tasks/TaskBadges.vue";
import TaskComments from "@/components/tasks/TaskComments.vue";
import TaskLogs from "@/components/tasks/TaskLogs.vue";
import TaskProgressPanel from "@/components/tasks/TaskProgressPanel.vue";
import TaskReferences from "@/components/tasks/TaskReferences.vue";
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
import {
  NEW_TASK_RELATIONS,
  newTaskRelationIcon,
  tagColor,
  tagStyle,
  TASK_SIZES,
  taskStateClassFor,
  taskStateIcon,
  taskStateLabel,
  taskStateLabelFor,
  type NewTaskRelation,
} from "@/lib/task";
import { formatArgValue, parseArgsJSON } from "@/lib/taskTemplates";
import { tickUsage } from "@/lib/usage";
import {
  PromptScope,
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
  ChevronRightIcon,
  EraserIcon,
  ExternalLinkIcon,
  FileCodeIcon,
  InboxIcon,
  LockIcon,
  LockOpenIcon,
  MessageSquareOffIcon,
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
import { RouterLink } from "vue-router";

type SavingState = {
  assignee: boolean;
  priority: boolean;
  size: boolean;
  project: boolean;
  tags: boolean;
  links: boolean;
  teams: boolean;
  leads: boolean;
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
}>();

const emit = defineEmits<{
  (e: "update:tab", value: string): void;
  (e: "assignee-change", value: unknown): void;
  (e: "priority-change", value: unknown): void;
  (e: "size-change", value: unknown): void;
  (e: "project-change", value: unknown): void;
  (e: "toggle-team", id: number): void;
  (e: "toggle-lead", id: string): void;
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

const { agents, usernameFor, agentIdForActor } = useAgents();
const { active: allProjects } = useProjects();

const detailsTabs = computed(() => {
  const progress = props.task?.progress;
  const plan =
    progress && progress.total > 0
      ? `Plan · ${progress.done}/${progress.total}`
      : "Plan";
  return [
    { value: "general", label: "General" },
    { value: "comments", label: "Comments" },
    { value: "plan", label: plan },
    { value: "artifacts", label: "Artifacts" },
    { value: "prompts", label: "Prompts" },
    { value: "logs", label: "Logs" },
  ];
});

const LOGS_TABS = [
  { value: "activity", label: "Activity" },
  { value: "tool-calls", label: "Tool calls" },
] as const;
const logsTab = ref<"activity" | "tool-calls">("activity");

const detailsTab = computed({
  get: () => props.tab ?? "general",
  set: (value: string) => emit("update:tab", value),
});

const templateArgs = computed(() =>
  Object.entries(parseArgsJSON(props.task?.template?.argsJson ?? "")),
);

const PRIORITIES = [
  { value: TaskPriority.LOW, label: "Low" },
  { value: TaskPriority.NORMAL, label: "Normal" },
  { value: TaskPriority.HIGH, label: "High" },
  { value: TaskPriority.CRITICAL, label: "Critical" },
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
  const seconds = (thinkingMs.value - modelWaitMs.value) / 1000;
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
  return "Restart Pi so it reloads the agent's models.json";
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
  return actorId ? agentIdForActor(actorId) : AUTO;
});
const priorityChoice = computed(
  () => props.task?.priority || TaskPriority.NORMAL,
);
const sizeChoice = computed(() => props.task?.size || TaskSize.MEDIUM);
const projectChoice = computed(() => props.task?.projectId ?? "");
const teamIds = computed(() =>
  (props.task?.teams ?? []).map((team) => team.id),
);
const leadIds = computed(() =>
  (props.task?.leads ?? []).map((lead) => lead.actorId),
);

const canRemoveLead = computed(() => {
  const state = props.task?.state;
  const leadless = state === TaskState.BACKLOG || state === TaskState.CANCELLED;
  return leadless || leadIds.value.length > 1;
});
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

const systemPromptPreview = computed(
  () => props.systemPrompt.length > 0 && !props.systemPrompt[0]?.pinned,
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
</script>

<style lang="postcss" scoped></style>
