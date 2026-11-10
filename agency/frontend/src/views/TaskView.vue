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
  header.shrink-0.border-b(
    :class="detailsOpen ? 'mr-80' : ''"
    data-testid="task-header"
  )
    .flex.items-center.gap-3.px-4.py-2
      BackButton(:fallback="backFallback")
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
          title="The agent-side Pi session and workspace are gone (cleaned up, or the agent was restarted with fresh data); the conversation is stored in the agency and a follow-up starts a new Pi session"
        )
          CircleSlashIcon.size-3.shrink-0
          | Session gone
      .flex.shrink-0.items-center.gap-2(v-if="task")
        Button(
          variant="ghost"
          size="icon-sm"
          :class="task.favorite ? 'text-amber-500' : 'text-muted-foreground'"
          :title="task.favorite ? 'Remove from favorites' : 'Add to favorites'"
          :aria-label="task.favorite ? 'Remove from favorites' : 'Add to favorites'"
          :aria-pressed="task.favorite"
          @click="toggleFavorite"
          :disabled="saving.favorite"
        )
          StarIcon.size-4(:fill="task.favorite ? 'currentColor' : 'none'")
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

  Tabs(
    v-if="taskSites.length"
    :taskSites="taskSites"
    :activeTaskTab="activeTaskTab"
    :detailsOpen="detailsOpen"
    @selectSiteTab="selectSiteTab"
    @select-chat="activeTaskTab = 'chat'"
  )

  .flex.min-h-0.flex-1(:class="detailsOpen ? 'mr-80' : ''")
    Console(
      :active="activeTaskTab === 'chat'"
      :id="id"
      :task="task"
      :task-usage="taskUsage"
      :config="config"
      :model-selection="modelSelection"
      :saving="saving"
      :can-follow-up="canFollowUp"
      :can-compose="canCompose"
      :can-answer="canAnswerInput"
      :prompt="prompt"
      :tab="tab"
      :staged-attachments="stagedAttachments"
      :dragging-files="draggingFiles"
      :task-queue="taskQueue"
      :state-changes="stateChanges"
      :clearing-queue="clearingQueue"
      :sending="sending"
      :stopping="stopping"
      @update:prompt="prompt = $event"
      @update:tab="tab = $event"
      @send="send"
      @enter="onEnter"
      @stop="stop"
      @reopen="reopen"
      @remove-attachment="removeAttachment"
      @attachment-uploaded="onAttachmentUploaded"
      @draft-saved="onPromptSaved"
      @pending-prompt-saved="onPendingPromptSaved"
      @open-attachment="openAttachment"
      @open-image="openInlineImage"
      @open-diagram="openInlineDiagram"
      @open-new-sketch="openNewSketch"
      @open-sketch="openSketch"
      @model-change="onModelSelectionChange"
      @staged-uploaded="onStagedUploaded"
      @remove-staged="removeStaged"
      @drag-enter="onDragEnter"
      @drag-leave="onDragLeave"
      @drop="onDrop"
      @paste="onPaste"
      @queue="onQueue"
      @clear-queue="clearQueue"
      @open-plan="openPlan"
      @remind-plan="remindPlan"
      @live-timing="liveTiming = $event"
    )

    SiteVue(
      v-for="site in visitedSites"
      :key="site.id"
      v-show="activeSite?.id === site.id"
      :site="site"
      @open-full="openFullSite(site)"
      @open-new="openSiteNewTab(site)"
    )

    TaskDetails(
      v-model:tab="detailsTab"
      :open="detailsOpen"
      :task="task"
      :saving="saving"
      :assignee-options="assigneeOptions"
      :project-options="projectOptions"
      :project-teams="projectTeams"
      :lead-candidates="leadCandidates"
      :unattached-tags="unattachedTags"
      :task-usage="taskUsage"
      :live-timing="liveTiming"
      :task-disk-bytes="taskDiskBytes"
      :task-disk-loading="taskDiskLoading"
      :system-prompt="systemPrompt"
      :action-busy="actionBusy"
      :stopping="stopping"
      :can-compose="canCompose"
      @assignee-change="onAssigneeChange"
      @priority-change="onPriorityChange"
      @size-change="onSizeChange"
      @project-change="onProjectChange"
      @toggle-team="toggleTaskTeam"
      @toggle-lead="toggleTaskLead"
      @toggle-tag="toggleTag"
      @toggle-subscription="toggleSubscription"
      @reference-change="onReferenceChange"
      @create-related="createRelatedTask"
      @open-attachment="openAttachment"
      @remind-plan="remindPlan"
      @open-media="openLightboxItem"
      @comments-read="onCommentsRead"
      @edit-links="linksDialogOpen = true"
      @queue="queue"
      @dequeue="dequeue"
      @stop="stop"
      @reopen="reopen"
      @close-out="closeOut"
      @release-hold="releaseHoldFor"
      @archive="archive"
      @unarchive="unarchive"
      @cleanup-session="cleanupSession"
      @restart-session="restartSession"
      @delete="remove"
    )

  CloseOutTaskDialog(
    v-model:open="closeOutDialogOpen"
    :task="task ?? null"
    @closed="load"
  )

  ReleaseHoldDialog(
    v-model:open="releaseHoldOpen"
    :task="task ?? null"
    :hold="releaseHold"
    @released="onHoldReleased"
  )

  ArchiveTaskDialog(
    :open="archiveOpen"
    :tasks="task ? [task] : []"
    :busy="actionBusy"
    @update:open="archiveOpen = $event"
    @confirm="confirmArchive"
  )

  TaskComposerDialog(
    v-model:open="relatedTaskOpen"
    :related-task="task ?? null"
    :relation="relatedTaskRelation"
    @created="onRelatedTaskCreated"
  )

  TaskLinksDialog(
    :open="linksDialogOpen"
    :links="linkInputs"
    :busy="saving.links"
    @update:open="linksDialogOpen = $event"
    @save="onLinksSave"
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

  TaskLightbox(
    v-model:open="lightboxOpen"
    :items="lightboxEntries"
    :start-index="lightboxIndex"
  )
</template>

<script setup lang="ts">
import BackButton from "@/components/BackButton.vue";
import ArchiveTaskDialog from "@/components/tasks/ArchiveTaskDialog.vue";
import CloseOutTaskDialog from "@/components/tasks/CloseOutTaskDialog.vue";
import ReleaseHoldDialog from "@/components/tasks/ReleaseHoldDialog.vue";
import TaskComposerDialog from "@/components/tasks/TaskComposerDialog.vue";
import TaskLightbox from "@/components/tasks/TaskLightbox.vue";
import TaskLinksDialog from "@/components/tasks/TaskLinksDialog.vue";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import Console from "@/views/TaskView/Console.vue";
import SiteVue from "@/views/TaskView/Site.vue";
import { default as Tabs } from "@/views/TaskView/Tabs.vue";
import TaskDetails from "@/views/TaskView/TaskDetails.vue";

import { useActorPictures } from "@/composables/useActorPictures";
import { useAgents } from "@/composables/useAgents";
import { useAttachmentUpload } from "@/composables/useAttachmentUpload";
import type { BackFallback } from "@/composables/useBackNavigation";
import { useDraft } from "@/composables/useDraft";
import { setPageTitle } from "@/composables/usePageTitle";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { useRealtime } from "@/composables/useRealtime";
import { useSites } from "@/composables/useSites";
import {
  useSketchEditor,
  type SketchSavePayload,
} from "@/composables/useSketchEditor";
import { useTags } from "@/composables/useTags";
import { useTaskDetails } from "@/composables/useTaskDetails";
import { accountApi, type UserPreferences } from "@/lib/account";
import {
  queuedMessageParts,
  timestampMs,
  type StateChange,
  type UserMessageParts,
} from "@/lib/console";
import { chatDraftKey } from "@/lib/draft";
import {
  attachmentItem,
  buildTaskLightboxItems,
  inlineDiagramItem,
  inlineImageItem,
  withLightboxItem,
  type DiagramSource,
  type InlineImage,
  type LightboxItem,
} from "@/lib/lightbox";
import { sortProjectsByName } from "@/lib/project";
import {
  projectAccessApi,
  projectTeamsWithInherited,
  type LeadCandidate,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import { agency, agent } from "@/lib/sdk";
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
  type NewTaskRelation,
  type TaskModelSelection,
} from "@/lib/task";
import { deleteConfirmation } from "@/lib/taskActions";
import {
  TaskPriority,
  TaskSize,
  TaskState,
  type Attachment,
  type Project,
  type Site,
  type SystemPromptSection,
  type Tag,
  type Task,
  type TaskAttributeInput,
  type TaskConfig,
  type TaskHold,
  type TaskLinkInput,
  type TaskUsageWithRuns,
} from "@cherrypi/sdk";
import { CircleSlashIcon, MenuIcon, PencilIcon, StarIcon } from "@lucide/vue";
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{ id: string }>();

const SketchDialog = defineAsyncComponent(
  () => import("@/components/sketches/SketchDialog.vue"),
);

const router = useRouter();
const route = useRoute();
const { agents, loadAgents, actorIdFor, agentIdForActor } = useAgents();
const { loadActorPictures } = useActorPictures();
const { active: allProjects, loadProjects } = useProjects();
const { sites: taskSites, load: loadTaskSites } = useSites();

const activeTaskTab = ref("chat");

const SITE_TAB_PREFIX = "site:";
const activeSite = computed<Site | undefined>(() => {
  if (!activeTaskTab.value.startsWith(SITE_TAB_PREFIX)) return undefined;
  const id = activeTaskTab.value.slice(SITE_TAB_PREFIX.length);
  return taskSites.value.find((site) => site.id === id);
});

const visitedSiteIds = ref<string[]>([]);
const visitedSites = computed(() =>
  taskSites.value.filter((site) => visitedSiteIds.value.includes(site.id)),
);

function selectSiteTab(site: Site) {
  if (!visitedSiteIds.value.includes(site.id)) {
    visitedSiteIds.value = [...visitedSiteIds.value, site.id];
  }
  activeTaskTab.value = SITE_TAB_PREFIX + site.id;
}

async function loadSites() {
  await loadTaskSites({ originTaskId: props.id });
  visitedSiteIds.value = visitedSiteIds.value.filter((id) =>
    taskSites.value.some((site) => site.id === id),
  );
  const current = activeSite.value;
  if (current && !visitedSiteIds.value.includes(current.id)) {
    visitedSiteIds.value = [...visitedSiteIds.value, current.id];
  }
  if (activeTaskTab.value !== "chat" && !activeSite.value) {
    activeTaskTab.value = "chat";
  }
}

function openSiteNewTab(site: Site) {
  window.open(site.url, "_blank", "noopener");
}

function openFullSite(site: Site) {
  void router.push({ name: "site", params: { id: site.id } });
}

const task = ref<Task | undefined>();

const { active: activeTags } = useTags(() => task.value?.projectId ?? "");

const {
  open: detailsOpen,
  toggle: toggleDetails,
  setOpen: setDetailsOpen,
} = useTaskDetails();

const detailsTab = ref("general");

function openPlan() {
  detailsTab.value = "plan";
  if (!detailsOpen.value) toggleDetails();
}

const deepLinkedTab = ref("");
watch(
  () => route.query.tab,
  (tab) => {
    if (typeof tab !== "string" || !tab || tab === deepLinkedTab.value) return;
    deepLinkedTab.value = tab;
    detailsTab.value = tab;
    setDetailsOpen(true);
  },
  { immediate: true },
);

const lightboxOpen = ref(false);
const lightboxEntries = ref<LightboxItem[]>([]);
const lightboxIndex = ref(0);

const inlineLightboxItems = ref<LightboxItem[]>([]);
const taskUsage = ref<TaskUsageWithRuns | undefined>();

const stateChanges = computed<StateChange[]>(() => {
  const logs = task.value?.logs ?? [];
  const changes: StateChange[] = [];
  for (const log of logs) {
    if (log.kind !== "state") continue;
    let data: Record<string, unknown> | undefined;
    if (log.dataJson) {
      try {
        const parsed: unknown = JSON.parse(log.dataJson);
        if (parsed && typeof parsed === "object") {
          data = parsed as Record<string, unknown>;
        }
      } catch {
        data = undefined;
      }
    }
    changes.push({
      key: String(log.id),
      at: timestampMs(log.at),
      from: String(data?.["old"] ?? ""),
      to: String(data?.["new"] ?? ""),
      reason: String(data?.["reason"] ?? ""),
      actor: log.actor,
    });
  }
  return changes;
});

const liveTiming = ref({ toolRunning: false, modelWaiting: false });

const taskDiskBytes = ref<bigint | undefined>();
const taskDiskLoading = ref(false);

const config = ref<TaskConfig | undefined>();
const modelSelection = ref<TaskModelSelection>({
  ...EMPTY_TASK_MODEL_SELECTION,
});
const systemPrompt = ref<SystemPromptSection[]>([]);

const prompt = useDraft(() => chatDraftKey(props.id));

const tab = ref<"write" | "preview">("write");

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
  favorite: false,
  model: false,
});

const agentProjects = ref<Project[]>([]);
const projectMembers = ref<string[]>([]);
const closeOutDialogOpen = ref(false);
const linksDialogOpen = ref(false);
const releaseHoldOpen = ref(false);
const releaseHold = ref<TaskHold | null>(null);

const relatedTaskOpen = ref(false);
const relatedTaskRelation = ref<NewTaskRelation | undefined>();
const titleEditing = ref(false);
const titleDraft = ref("");
const titleInput = ref<{ $el?: HTMLInputElement } | undefined>();

const AUTO = "__auto__";

let generation = 0;

let configGeneration = 0;

const isBacklog = computed(() => task.value?.state === TaskState.BACKLOG);

const hasStarted = computed(() => Boolean(task.value?.startedAt));
const isPostponed = computed(() => isBacklog.value && hasStarted.value);
const isInput = computed(() => task.value?.state === TaskState.INPUT);

const projectTeams = ref<ProjectTeamLink[]>([]);
const leadCandidates = ref<LeadCandidate[]>([]);
const teamIds = computed(() =>
  (task.value?.teams ?? []).map((team) => team.id),
);
const leadIds = computed(() =>
  (task.value?.leads ?? []).map((lead) => lead.actorId),
);
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

const sessionDeleted = computed(() => Boolean(task.value?.sessionDeletedAt));

const canFollowUp = computed(
  () => !!task.value && (acceptsPrompt(task.value.state) || isPostponed.value),
);

const canCompose = computed(() => canFollowUp.value || isInput.value);

const canAnswerInput = computed(() => Boolean(task.value?.canAnswerInput));

function acceptsPrompt(state: TaskState): boolean {
  return (
    state === TaskState.RUNNING ||
    state === TaskState.SETTLED ||
    state === TaskState.FAILED ||
    state === TaskState.COMPLETED ||
    state === TaskState.CANCELLED
  );
}

function hasLiveConfig(state: TaskState): boolean {
  return acceptsPrompt(state) || state === TaskState.INPUT;
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

async function loadAgentProjects(assigneeActorId: string) {
  agentProjects.value = [];
  if (!assigneeActorId) return;
  await loadAgents();
  const owner = agents.value.find((item) => item.actorId === assigneeActorId);
  if (!owner) return;
  try {
    agentProjects.value = await agency.listAgentProjects(owner.id);
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
  (actorId) => void loadAgentProjects(actorId),
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

const leadRequired = computed(() => {
  const state = task.value?.state;
  return (
    !!state && state !== TaskState.BACKLOG && state !== TaskState.CANCELLED
  );
});

async function toggleTaskLead(id: string) {
  const next = new Set(leadIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  if (!next.size && leadRequired.value) {
    toast.error("A task needs at least one lead");
    return;
  }
  await setTaskLeads([...next]);
}

async function toggleFavorite() {
  const current = task.value;
  if (!current) return;
  saving.favorite = true;
  try {
    task.value = await agency.setTaskFavorite(current.id, {
      favorite: !current.favorite,
    });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.favorite = false;
  }
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

const backFallback = computed<BackFallback>(() => {
  const projectId = task.value?.projectId;
  if (projectId) {
    const slug = projectSlug(projectId);
    if (slug) {
      return {
        to: { name: "project", params: { slug } },
        label: projectName(projectId),
      };
    }
  }
  return { to: { name: "agency-tasks" }, label: "Tasks" };
});

async function loadUsage(sessionId: string) {
  if (!sessionId) {
    taskUsage.value = undefined;
    return;
  }
  try {
    taskUsage.value = await agent.getTaskUsage(sessionId);
  } catch {
    taskUsage.value = undefined;
  }
}

const DISK_REFRESH_MS = 10_000;
let diskKey = "";
let diskAt = 0;

async function loadDiskUsage(sessionId: string) {
  if (!sessionId) {
    taskDiskBytes.value = undefined;
    taskDiskLoading.value = false;
    return;
  }
  const now = Date.now();
  if (sessionId === diskKey && now - diskAt < DISK_REFRESH_MS) return;
  diskKey = sessionId;
  diskAt = now;
  taskDiskLoading.value = true;
  try {
    taskDiskBytes.value = await agent.getTaskDiskUsage(sessionId);
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

  modelSelection.value = next;
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
    modelSelection.value = previous;
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
    applyTask(next, { refetchSession: true });
  } catch (err) {
    if (gen !== generation) return;
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function applyTask(next: Task, options: { refetchSession?: boolean } = {}) {
  const previous = task.value;
  task.value = next;

  if (!config.value) {
    modelSelection.value = {
      modelProvider: next.modelProvider,
      modelId: next.modelId,
      thinkingLevel: next.thinkingLevel,
    };
  }

  if (!next.assigneeActorId || next.sessionDeletedAt) return;
  const sessionChanged = next.sessionId !== previous?.sessionId;
  const stateChanged = next.state !== previous?.state;
  if (options.refetchSession || sessionChanged) {
    void loadUsage(next.sessionId);
    void loadDiskUsage(next.sessionId);
  }
  if (
    (options.refetchSession || sessionChanged || stateChanged) &&
    hasLiveConfig(next.state) &&
    !saving.model
  ) {
    void loadConfig();
  }
}

function onPromptSaved(next: Task) {
  task.value = next;
}

function onPendingPromptSaved() {
  void load();
}

function onReferenceChange(next: Task) {
  task.value = next;
}

function createRelatedTask(relation: NewTaskRelation) {
  relatedTaskRelation.value = relation;
  relatedTaskOpen.value = true;
}

function onRelatedTaskCreated() {
  relatedTaskOpen.value = false;
  relatedTaskRelation.value = undefined;
  void load();
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
const archivePrefs = ref<UserPreferences>({
  notify_first_archive: true,
});

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

async function restartSession() {
  actionBusy.value = true;
  try {
    await withProgress(() => agent.restartTaskSession(props.id));
    config.value = undefined;
    await loadConfig();
    toast.success("Pi session restarted");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    actionBusy.value = false;
  }
}

function closeOut() {
  closeOutDialogOpen.value = true;
}

function releaseHoldFor(hold: TaskHold) {
  releaseHold.value = hold;
  releaseHoldOpen.value = true;
}

function onHoldReleased(updated: Task) {
  task.value = updated;
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

const PLAN_REMINDER_PROMPT =
  "You have not published a plan for this task. Publish it now with " +
  "`reportTaskProgress` (3–8 ordered items, each `pending`), then carry on " +
  "with the work immediately in the same turn: a plan update is bookkeeping, " +
  "never a stopping point. Keep the plan current afterwards — mark an item " +
  "`in_progress` when you begin it and `done` when it lands.";

async function remindPlan() {
  try {
    await withProgress(() => agent.addToTask(props.id, PLAN_REMINDER_PROMPT));
    toast.success("Reminded the agent to publish a plan");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function send() {
  if (!canFollowUp.value || !prompt.value.trim()) return;
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
  if (canCompose.value && hasFiles(event)) {
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
  if (!canCompose.value) return;
  await stageUploads(event.dataTransfer?.files ?? []);
}

async function onPaste(event: ClipboardEvent) {
  if (!canCompose.value) return;
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

function taskLightboxItems(): LightboxItem[] {
  const items = buildTaskLightboxItems({
    attachments: task.value?.attachments ?? [],
    stagedAttachments: stagedAttachments.value,
    artifacts: task.value?.artifacts ?? [],
  });
  for (const item of inlineLightboxItems.value) {
    if (!items.some((known) => known.url === item.url)) items.push(item);
  }
  return items;
}

function openLightboxItem(item: LightboxItem) {
  const gallery = withLightboxItem(taskLightboxItems(), item);
  lightboxEntries.value = gallery.items;
  lightboxIndex.value = gallery.index;
  lightboxOpen.value = true;
}

function onCommentsRead() {
  if (task.value) task.value.unreadComments = 0;
}

function openAttachment(attachment: Attachment) {
  openLightboxItem(attachmentItem(attachment));
}

function openInlineImage(image: InlineImage) {
  const item = inlineImageItem(image);
  if (!inlineLightboxItems.value.some((known) => known.key === item.key)) {
    inlineLightboxItems.value = [...inlineLightboxItems.value, item];
  }
  openLightboxItem(item);
}

function openInlineDiagram(diagram: DiagramSource) {
  const item = inlineDiagramItem(diagram);
  if (!inlineLightboxItems.value.some((known) => known.key === item.key)) {
    inlineLightboxItems.value = [...inlineLightboxItems.value, item];
  }
  openLightboxItem(item);
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
    if (canCompose.value) {
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

function onEnter() {
  if (!canFollowUp.value) return;
  void send();
}

async function remove() {
  const current = task.value;
  if (!current) return;
  if (!window.confirm(deleteConfirmation(current))) {
    return;
  }
  await deleteTask();
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

watch(
  () => task.value?.projectId ?? "",
  async (id) => {
    projectTeams.value = [];
    leadCandidates.value = [];
    if (!id) return;
    try {
      const access = await projectAccessApi.access(id);
      projectTeams.value = projectTeamsWithInherited(access);
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
  void loadSites();
  void accountApi
    .getPreferences()
    .then((prefs) => {
      archivePrefs.value = prefs;
    })
    .catch(() => {});
});

useRealtime(
  ["tasks", "usage", "sites"],
  (event) => {
    if (event.kind === "sites") {
      void loadSites();
      return;
    }
    if (event.kind === "usage") {
      const sessionId = task.value?.sessionId;
      if (sessionId) void loadUsage(sessionId);
      return;
    }
    if (event.task) {
      if (event.task.id === props.id) applyTask(event.task);
      return;
    }
    void load();
    void loadSites();
  },
  { rows: true },
);

watch(
  () => (task.value ? taskName(task.value) : "Task"),
  (title) => setPageTitle(title),
  { immediate: true },
);

watch(
  () => props.id,
  () => {
    task.value = undefined;
    activeTaskTab.value = "chat";
    visitedSiteIds.value = [];
    config.value = undefined;
    modelSelection.value = { ...EMPTY_TASK_MODEL_SELECTION };
    configGeneration += 1;
    taskUsage.value = undefined;
    liveTiming.value = { toolRunning: false, modelWaiting: false };
    taskDiskBytes.value = undefined;
    taskDiskLoading.value = false;
    diskKey = "";
    void load();
    void loadSites();
    void loadSystemPrompt();
  },
);

watch(
  () => `${task.value?.state ?? ""}\u0000${task.value?.sessionId ?? ""}`,
  () => void loadSystemPrompt(),
);
</script>
