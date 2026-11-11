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
Card.overflow-visible(:class="bare ? BARE_CARD : ''")
  CardContent(:class="bare ? 'p-0' : 'p-3'")
    form.flex.flex-col.gap-3(@submit.prevent="submit()")
      .flex.flex-col.gap-2
        .flex.items-center.gap-2
          Input.flex-1(
            v-model="title"
            :placeholder="$t('express.autoTitle')"
            autocomplete="off"
            :aria-label="$t('express.titlePlaceholder')"
            @keydown.enter.prevent
            data-testid="composer-title"
          )
          .flex.shrink-0.items-center.gap-2
            Button(
              type="button"
              variant="outline"
              :title="$t('express.toBacklogHint')"
              @click="submit(TaskState.BACKLOG)"
              :disabled="submitDisabled"
            )
              Spinner(
                v-if="saving && submitting === TaskState.BACKLOG"
                data-icon="inline-start"
              )
              InboxIcon(v-else data-icon="inline-start")
              span {{ $t("express.toBacklog") }}
            Button(
              type="button"
              :title="$t('express.toQueueHint')"
              @click="submit(TaskState.QUEUED)"
              :disabled="submitDisabled"
            )
              Spinner(
                v-if="saving && submitting === TaskState.QUEUED"
                data-icon="inline-start"
              )
              PlayIcon(v-else data-icon="inline-start")
              span {{ $t("tasks.action.queue") }}
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
            ref="promptInput"
            v-model="prompt"
            v-model:resize-height="promptHeight"
            :placeholder="$t('express.promptPlaceholder')"
            resizable
            references
            test-id="express-editor"
            :title="$t('express.shortcuts')"
            :aria-label="$t('taskPrompt.aria')"
            :reference-project-id="selectedProjectId"
            @submit="submit"
            @task-picked="addPromptReference"
            @paste="onPaste"
          )
        .flex.flex-wrap.items-center.gap-2
          Button(
            type="button"
            variant="outline"
            size="sm"
            :title="$t('taskPrompt.attachAria')"
            @click="pickFiles"
          )
            PaperclipIcon(data-icon="inline-start")
            span {{ $t("taskPrompt.attach") }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            :title="$t('taskPrompt.sketchAria')"
            @click="openNewSketch"
          )
            PencilIcon(data-icon="inline-start")
            span {{ $t("taskPrompt.sketch") }}
          span.text-muted-foreground.text-xs(v-if="!stagedCount")
            | {{ $t("express.attachmentsHint") }}
          span.text-muted-foreground.ml-auto.text-xs(
            data-testid="composer-submit-hint"
          )
            | {{ $t("express.shortcutsShort") }}
          input.hidden(
            ref="fileInput"
            type="file"
            multiple
            @change="onFileChange"
          )
        StagedAttachmentList(
          :items="stagedItems"
          @remove="removeStaged"
          @edit="editStagedSketch"
        )

      .rounded-lg.border.p-2(class="bg-muted/30")
        .flex.flex-wrap.items-center.gap-2
          Button(
            v-if="fixedProjectId"
            type="button"
            variant="outline"
            size="sm"
            disabled
            :title="$t('express.inProject', { project: fixedProjectLabel })"
          )
            span {{ fixedProjectLabel }}
          Combobox.w-40(
            v-else
            :placeholder="$t('editTask.selectProject')"
            size="sm"
            empty-text="No matching project."
            :title="$t('admin.projects.columnProject')"
            :aria-label="$t('admin.projects.columnProject')"
            :model-value="projectChoice"
            :options="projectComboboxOptions"
            @update:model-value="pickProject"
          )

          Button(
            type="button"
            variant="outline"
            size="sm"
            :title="selectedProjectId ? $t('express.templateHint') : $t('express.selectProjectFirst')"
            @click="templateOpen = true"
            :disabled="!selectedProjectId"
            data-testid="composer-template"
          )
            FileCodeIcon(data-icon="inline-start")
            span {{ $t("domainLog.kind.template") }}

          span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
            v-if="appliedTemplate"
            class="py-0.5"
            :title="$t('express.templateVersion', { version: appliedTemplate.version })"
          )
            FileCodeIcon.size-3.shrink-0
            span {{ appliedTemplate.templateName }} · v{{ appliedTemplate.version }}
            button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
              type="button"
              class="p-0.5 hover:opacity-100"
              :title="$t('express.detachTemplate')"
              @click="detachTemplate"
            )
              XIcon.size-3

          TaskParentPicker(
            v-model="parentChoice"
            :project-id="selectedProjectId"
            :disabled="!projectReady"
          )

          label.inline-flex.items-center.text-xs(
            v-if="parentChoice"
            class="gap-1.5"
            :title="$t('express.interactiveHint')"
            data-testid="composer-interactive"
          )
            Switch(
              v-model="interactive"
              :aria-label="$t('express.interactive')"
            )
            span {{ $t("express.interactive") }}

          Button(
            v-if="fixedAgentId"
            type="button"
            variant="outline"
            size="sm"
            disabled
            :title="$t('express.assignedTo', { agent: fixedAgentLabel })"
          )
            span {{ fixedAgentLabel }}
          Select(v-else v-model="assigneeChoice")
            SelectTrigger.w-40(
              size="sm"
              :title="$t('admin.agents.columnAgent')"
            )
              SelectValue(:placeholder="$t('editTask.autoAssign')")
            SelectContent
              SelectGroup
                SelectItem(:value="AUTO") {{ $t("editTask.autoAssign") }}
                SelectItem(
                  v-for="item in assigneeOptions"
                  :key="item.id"
                  :value="item.id"
                ) {{ item.displayName || item.username }}

          TaskModelPicker(
            :agent-id="selectedAgentId"
            :selection="modelSelection"
            @change="modelSelection = $event"
            :disabled="saving || !selectedAgentId"
          )

          .bg-muted.inline-flex.items-center.gap-1.rounded-lg.border(class="p-0.5")
            button.rounded-md.px-2.text-xs.font-medium.transition-colors(
              v-for="option in SIZES"
              :key="option.value"
              type="button"
              class="py-0.5"
              :class="size === option.value ? `bg-card shadow-sm ring-1 ring-border ${taskSizeTextClass(option.value)}` : 'text-muted-foreground hover:text-foreground'"
              @click="size = option.value"
            ) {{ option.label }}

          .bg-muted.inline-flex.items-center.gap-1.rounded-lg.border(class="p-0.5")
            button.rounded-md.px-2.text-xs.font-medium.transition-colors(
              v-for="option in PRIORITIES"
              :key="option.value"
              type="button"
              class="py-0.5"
              :class="priority === option.value ? `bg-card shadow-sm ring-1 ring-border ${taskPriorityTextClass(option.value)}` : 'text-muted-foreground hover:text-foreground'"
              @click="priority = option.value"
            ) {{ option.label }}

          DropdownMenu
            DropdownMenuTrigger(as-child)
              Button(
                type="button"
                variant="outline"
                size="sm"
                :title="selectedProjectId ? owningTeamsTitle : $t('express.selectProjectFirst')"
                :disabled="!selectedProjectId"
              )
                UsersIcon(data-icon="inline-start")
                span {{ $t("nav.teams") }}
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
                :model-value="owningTeamIds.includes(team.team_id)"
                @update:model-value="toggleOwningTeam(team.team_id)"
                @select.prevent
              ) {{ team.team_name }}

          span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
            v-for="team in selectedTeamList"
            :key="team.team_id"
            class="py-0.5"
            :title="team.team_description"
          )
            UsersIcon.size-3.shrink-0
            span {{ team.team_name }}
            button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
              type="button"
              class="p-0.5 hover:opacity-100"
              :title="$t('express.removeTeam')"
              @click="toggleOwningTeam(team.team_id)"
            )
              XIcon.size-3

          DropdownMenu
            DropdownMenuTrigger(as-child)
              Button(
                type="button"
                variant="outline"
                size="sm"
                :title="leadsTitle"
              )
                UserCheckIcon(data-icon="inline-start")
                span {{ $t("domainLog.kind.leads") }}
            DropdownMenuContent.w-56(align="start")
              DropdownMenuLabel {{ $t("domainLog.kind.leads") }}
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
              DropdownMenuCheckboxItem(
                v-for="candidate in filteredLeadCandidates"
                :key="candidate.actor_id"
                :model-value="leadActorIds.includes(candidate.actor_id)"
                @update:model-value="toggleLead(candidate.actor_id)"
                @select.prevent
              )
                ActorAvatar(
                  :src="candidate.picture_url"
                  :name="candidate.name"
                  :username="candidate.username"
                  :size="18"
                )
                span {{ candidate.name || candidate.username }}

          span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-1.text-xs(
            v-for="lead in selectedLeadList"
            :key="lead.actor_id"
            class="py-0.5"
          )
            ActorAvatar(
              :src="lead.picture_url"
              :name="lead.name"
              :username="lead.username"
              :size="16"
            )
            span {{ lead.name || lead.username }}
            button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
              type="button"
              class="p-0.5 hover:opacity-100"
              :title="$t('express.removeLead')"
              @click="toggleLead(lead.actor_id)"
            )
              XIcon.size-3

          TagPicker(
            :project-id="selectedProjectId"
            :tags="activeTags"
            :model-value="[...selectedTags]"
            @update:model-value="onTagsChange"
            @created="onTagCreated"
            :disabled="!selectedProjectId"
          )

          TaskReferencePicker(
            v-model="references"
            :project-id="selectedProjectId"
            :known-tasks="referenceTasks"
            :disabled="!projectReady"
          )

      p.text-muted-foreground.text-xs(v-if="noProjects")
        | {{ $t("express.noProject") }}

  TaskTemplateApplyDialog(
    v-model:open="templateOpen"
    :project-id="selectedProjectId"
    @apply="applyTemplate"
    @create="createFromTemplate"
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
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import MarkdownEditor from "@/components/MarkdownEditor.vue";
import StagedAttachmentList from "@/components/attachments/StagedAttachmentList.vue";
import TagPicker from "@/components/tags/TagPicker.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import TaskParentPicker from "@/components/tasks/TaskParentPicker.vue";
import TaskReferencePicker from "@/components/tasks/TaskReferencePicker.vue";
import TaskTemplateApplyDialog, {
  type AppliedTemplate,
} from "@/components/templates/TaskTemplateApplyDialog.vue";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuFilter,
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
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { useAgents } from "@/composables/useAgents";
import { useDraft } from "@/composables/useDraft";
import { useSessionEditorHeight } from "@/composables/useEditorHeight";
import { useProjects } from "@/composables/useProjects";
import { useSketchEditor } from "@/composables/useSketchEditor";
import {
  useStagedAttachments,
  type StagedAttachment,
} from "@/composables/useStagedAttachments";
import { useTags } from "@/composables/useTags";
import { currentAccount } from "@/lib/account";
import {
  composerDraftKey,
  composerProjectDraftKey,
  composerTitleDraftKey,
} from "@/lib/draft";
import { NEW_TASK_PROMPT_HEIGHT_KEY } from "@/lib/editorHeight";
import { sortProjectsByName } from "@/lib/project";
import {
  projectAccessApi,
  projectTeamsWithInherited,
  type LeadCandidate,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import { agency } from "@/lib/sdk";
import { sketchSceneFilename, toStoredSnapshot } from "@/lib/sketches";
import {
  EMPTY_TASK_MODEL_SELECTION,
  TASK_SIZES,
  newTaskReferenceKind,
  taskPriorityKey,
  taskPriorityTextClass,
  taskSizeKey,
  taskSizeTextClass,
  type NewTaskRelation,
  type TaskModelSelection,
} from "@/lib/task";
import { teamApi } from "@/lib/team";
import {
  TaskPriority,
  TaskReferenceKind,
  TaskSize,
  TaskState,
  type CreateTaskInput,
  type Project,
  type ProjectAgent,
  type Task,
  type TaskReferenceInput,
} from "@cherrypi/sdk";
import {
  FileCodeIcon,
  InboxIcon,
  PaperclipIcon,
  PencilIcon,
  PlayIcon,
  UserCheckIcon,
  UsersIcon,
  XIcon,
} from "@lucide/vue";
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const NO_PROJECT = "__none__";
const AUTO = "__auto__";

const PRIORITIES = [
  { value: TaskPriority.LOW, label: t(taskPriorityKey(TaskPriority.LOW)) },
  {
    value: TaskPriority.NORMAL,
    label: t(taskPriorityKey(TaskPriority.NORMAL, true)),
  },
  { value: TaskPriority.HIGH, label: t(taskPriorityKey(TaskPriority.HIGH)) },
  {
    value: TaskPriority.CRITICAL,
    label: t(taskPriorityKey(TaskPriority.CRITICAL)),
  },
];

const SIZES = TASK_SIZES.map((option) => ({
  ...option,
  label: t(taskSizeKey(option.value, option.value === TaskSize.MEDIUM)),
}));

const props = defineProps<{
  agentId?: string;

  projectId?: string;

  defaultProjectId?: string;

  bare?: boolean;

  relatedTask?: Task | null;

  relation?: NewTaskRelation;
}>();

const BARE_CARD =
  "gap-0 rounded-none bg-transparent py-0 text-inherit ring-0 ring-transparent";

const emit = defineEmits<{ created: [Task] }>();

const { agents, loadAgents, actorIdFor } = useAgents();
const { active: allProjects, loadProjects } = useProjects();

const prompt = useDraft(() =>
  composerDraftKey({ agent: props.agentId, project: props.projectId }),
);

const title = useDraft(() =>
  composerTitleDraftKey({ agent: props.agentId, project: props.projectId }),
);
const priority = ref(TaskPriority.NORMAL);
const size = ref(TaskSize.MEDIUM);
const assigneeChoice = ref(props.agentId || AUTO);

const draftProject = useDraft(() =>
  composerProjectDraftKey({ agent: props.agentId, project: props.projectId }),
);
const projectChoice = ref(props.projectId || draftProject.value || NO_PROJECT);

function commitProject(id: string) {
  draftProject.value = id === NO_PROJECT ? "" : id;
}

function pickProject(id: string) {
  projectChoice.value = id;
  commitProject(id);
}

watch(
  [prompt, title],
  ([promptText, titleText]) => {
    if (!promptText.trim() && !titleText.trim()) return;
    if (draftProject.value || projectChoice.value === NO_PROJECT) return;
    commitProject(projectChoice.value);
  },
  { flush: "sync" },
);

const modelSelection = ref<TaskModelSelection>({
  ...EMPTY_TASK_MODEL_SELECTION,
});
const selectedTags = ref<Set<string>>(new Set());

const defaultTagsProject = ref("");
const references = ref<TaskReferenceInput[]>([]);

const appliedTemplate = ref<AppliedTemplate | null>(null);
const templateOpen = ref(false);

const parentChoice = ref<Task | null>(null);

const interactive = ref(true);

let seededRelation = "";

watch(parentChoice, (parent) => {
  if (!parent) interactive.value = true;
});

const referenceTasks = ref<Task[]>([]);

const projectTeams = ref<ProjectTeamLink[]>([]);
const leadCandidates = ref<LeadCandidate[]>([]);
const owningTeamIds = ref<number[]>([]);
const leadActorIds = ref<string[]>([]);
const creatorActorId = computed(() =>
  currentAccount.value ? String(currentAccount.value.id) : "",
);

const selectedTeamList = computed(() =>
  projectTeams.value.filter((team) =>
    owningTeamIds.value.includes(team.team_id),
  ),
);
const teamSearch = ref("");
const filteredProjectTeams = computed(() => {
  const query = teamSearch.value.trim().toLowerCase();
  if (!query) return projectTeams.value;
  return projectTeams.value.filter((team) =>
    team.team_name.toLowerCase().includes(query),
  );
});
const leadSearch = ref("");
const filteredLeadCandidates = computed(() => {
  const query = leadSearch.value.trim().toLowerCase();
  if (!query) return leadCandidates.value;
  return leadCandidates.value.filter((candidate) =>
    `${candidate.name} ${candidate.username}`.toLowerCase().includes(query),
  );
});
const projectComboboxOptions = computed<ComboboxOption[]>(() =>
  projectOptions.value.map((project) => ({
    value: project.id,
    label: project.name,
    description: project.prefix ? `#${project.prefix}` : undefined,
  })),
);
const selectedLeadList = computed(() =>
  leadCandidates.value.filter((lead) =>
    leadActorIds.value.includes(lead.actor_id),
  ),
);
const owningTeamsTitle = computed(() =>
  owningTeamIds.value.length
    ? t("express.owningTeams")
    : t("express.noOwningTeam"),
);
const leadsTitle = t("express.leadsHint");
function toggleOwningTeam(id: number) {
  const next = new Set(owningTeamIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  owningTeamIds.value = [...next];
}
function toggleLead(id: string) {
  const next = new Set(leadActorIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  leadActorIds.value = [...next];
}

const submitting = ref<TaskState | null>(null);
const submitDisabled = computed(
  () => saving.value || !prompt.value.trim() || !projectReady.value,
);
const agentProjects = ref<Project[]>([]);

const projectMembers = ref<ProjectAgent[]>([]);

const projectMembersLoaded = ref(false);
const saving = ref(false);
const promptInput = ref<InstanceType<typeof MarkdownEditor> | null>(null);

const promptHeight = useSessionEditorHeight(NEW_TASK_PROMPT_HEIGHT_KEY);

const {
  items: stagedItems,
  count: stagedCount,
  addFiles: addStagedFiles,
  addSketch: addStagedSketch,
  updateSketch: updateStagedSketch,
  remove: removeStaged,
  clear: clearStaged,
} = useStagedAttachments();
const fileInput = ref<HTMLInputElement>();

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
  openStaged: openStagedSketch,
  onSave: onSketchSave,
} = useSketchEditor({
  siblings: () => [],
  persist: async ({ png, snapshot, target }) => {
    if (target.stagedId) {
      updateStagedSketch(target.stagedId, png, snapshot);
    } else {
      addStagedSketch(png, snapshot);
    }
  },
});

function editStagedSketch(item: StagedAttachment) {
  if (item.kind === "sketch") openStagedSketch(item);
}

const dragging = ref(false);
let dragDepth = 0;

function pickFiles() {
  fileInput.value?.click();
}

function onFileChange(event: Event) {
  const element = event.target as HTMLInputElement;
  const files = Array.from(element.files ?? []);
  element.value = "";
  if (files.length) addStagedFiles(files);
}

function onPaste(event: ClipboardEvent) {
  const files: File[] = [];
  for (const item of event.clipboardData?.items ?? []) {
    if (item.kind !== "file") continue;
    const file = item.getAsFile();
    if (file) files.push(file);
  }
  if (!files.length) return;
  event.preventDefault();
  addStagedFiles(files);
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

function onDrop(event: DragEvent) {
  dragDepth = 0;
  dragging.value = false;
  const files = event.dataTransfer?.files;
  if (files?.length) addStagedFiles(files);
}

async function uploadStaged(taskId: string): Promise<number> {
  let failures = 0;
  for (const item of stagedItems.value) {
    try {
      if (item.kind === "sketch") {
        const image = await agency.uploadAttachment({
          file: item.png,
          filename: item.filename,
          taskId,
        });
        const scene = new Blob([toStoredSnapshot(item.snapshot)], {
          type: "application/json",
        });
        await agency.uploadAttachment({
          file: scene,
          filename: sketchSceneFilename(image.id),
          taskId,
        });
      } else {
        await agency.uploadAttachment({
          file: item.file,
          filename: item.filename,
          taskId,
        });
      }
    } catch (err) {
      failures += 1;
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }
  return failures;
}

const fixedAgentId = computed(() => props.agentId || "");
const fixedProjectId = computed(() => props.projectId || "");
const selectedAgentId = computed(
  () =>
    fixedAgentId.value ||
    (assigneeChoice.value === AUTO ? "" : assigneeChoice.value),
);

watch(selectedAgentId, () => {
  modelSelection.value = { ...EMPTY_TASK_MODEL_SELECTION };
});

const fixedAgentLabel = computed(() => {
  const item = agents.value.find((agent) => agent.id === props.agentId);
  return item?.displayName || item?.username || t("admin.agents.columnAgent");
});

const fixedProjectLabel = computed(() => {
  const item = allProjects.value.find(
    (project) => project.id === fixedProjectId.value,
  );
  return item?.name || t("admin.projects.columnProject");
});

const projectReady = computed(
  () => !!fixedProjectId.value || projectChoice.value !== NO_PROJECT,
);

const projectOptions = computed(() => {
  if (fixedAgentId.value) return sortProjectsByName(agentProjects.value);
  if (projectChoice.value === NO_PROJECT && assigneeChoice.value !== AUTO) {
    return sortProjectsByName(agentProjects.value);
  }
  return sortProjectsByName(allProjects.value);
});

const noProjects = computed(
  () => !fixedProjectId.value && projectOptions.value.length === 0,
);

const selectedProjectId = computed(
  () =>
    fixedProjectId.value ||
    (projectChoice.value === NO_PROJECT ? "" : projectChoice.value),
);

const { active: activeTags, refresh: refreshTags } = useTags(
  () => selectedProjectId.value,
);

watch(activeTags, (list) => {
  const valid = new Set(list.map((tag) => tag.id));
  const next = new Set([...selectedTags.value].filter((id) => valid.has(id)));
  if (next.size !== selectedTags.value.size) selectedTags.value = next;

  const projectID = selectedProjectId.value;
  if (!projectID || defaultTagsProject.value === projectID) return;

  if (list.some((tag) => tag.projectId !== projectID)) return;
  selectedTags.value = new Set(
    list.filter((tag) => tag.isDefault).map((tag) => tag.id),
  );
  defaultTagsProject.value = projectID;
});

const eligibleProjectAgentIds = computed(
  () =>
    new Set(
      projectMembers.value
        .filter((member) => member.canCreateTasks)
        .map((member) => member.agentId),
    ),
);

const assigneeOptions = computed(() => {
  if (!selectedProjectId.value || !projectMembersLoaded.value) {
    return agents.value;
  }
  const memberIds = eligibleProjectAgentIds.value;
  return agents.value.filter((agent) => memberIds.has(agent.id));
});

watch(
  fixedProjectId,
  (id) => {
    if (id) projectChoice.value = id;
  },
  { immediate: true },
);

function adoptPageProject() {
  const page = props.defaultProjectId || "";
  if (!page || fixedProjectId.value || draftProject.value) return;

  if (
    projectOptions.value.length &&
    !projectOptions.value.some((project) => project.id === page)
  ) {
    return;
  }
  if (projectChoice.value === page) return;
  projectChoice.value = page;
}

watch(() => props.defaultProjectId, adoptPageProject, { immediate: true });

function dropIneligibleAssignee() {
  if (
    fixedAgentId.value ||
    !selectedProjectId.value ||
    !projectMembersLoaded.value ||
    assigneeChoice.value === AUTO
  ) {
    return;
  }
  if (eligibleProjectAgentIds.value.has(assigneeChoice.value)) return;
  const agent = agents.value.find((item) => item.id === assigneeChoice.value);
  const project = allProjects.value.find(
    (item) => item.id === selectedProjectId.value,
  );
  assigneeChoice.value = AUTO;
  toast.info(
    t("express.agentNotMember", {
      agent: agent?.displayName || agent?.username || t("express.thatAgent"),
      project: project?.name || t("express.theProject"),
    }),
  );
}

watch(
  [
    assigneeChoice,
    selectedProjectId,
    projectMembersLoaded,
    eligibleProjectAgentIds,
    agents,
  ],
  dropIneligibleAssignee,
);

watch(
  assigneeChoice,
  async (value) => {
    if (fixedProjectId.value || value === AUTO) {
      agentProjects.value = [];
      return;
    }
    try {
      agentProjects.value = await agency.listAgentProjects(value);
    } catch {
      agentProjects.value = [];
    }
  },
  { immediate: true },
);

watch(projectOptions, (options) => {
  if (fixedProjectId.value) return;
  if (
    projectChoice.value !== NO_PROJECT &&
    !options.some((project) => project.id === projectChoice.value)
  ) {
    projectChoice.value = NO_PROJECT;

    commitProject(NO_PROJECT);
    adoptPageProject();
  }
});

watch(
  selectedProjectId,
  async (id) => {
    references.value = [];
    referenceTasks.value = [];
    appliedTemplate.value = null;
    parentChoice.value = null;
    interactive.value = true;
    selectedTags.value = new Set();
    defaultTagsProject.value = "";
    owningTeamIds.value = [];
    leadActorIds.value = [];
    projectTeams.value = [];
    leadCandidates.value = [];
    projectMembers.value = [];
    projectMembersLoaded.value = false;
    if (!id) return;
    try {
      const members = await agency.listProjectAgents(id);
      if (selectedProjectId.value !== id) return;
      projectMembers.value = members;
    } catch {
    } finally {
      if (selectedProjectId.value === id) projectMembersLoaded.value = true;
    }
    dropIneligibleAssignee();
    try {
      const access = await projectAccessApi.access(id);
      if (selectedProjectId.value !== id) return;
      projectTeams.value = projectTeamsWithInherited(access);
    } catch {}
    try {
      const candidates = await projectAccessApi.leadCandidates(id);
      if (selectedProjectId.value !== id) return;
      leadCandidates.value = candidates;
    } catch {}

    const creator = creatorActorId.value;
    if (!creator) return;
    try {
      const { teams } = await teamApi.list();
      if (selectedProjectId.value !== id) return;
      const mine = new Set(
        teams.filter((team) => team.role).map((team) => team.id),
      );
      owningTeamIds.value = projectTeams.value
        .map((team) => team.team_id)
        .filter((teamID) => mine.has(teamID));
    } catch {}

    if (
      !leadActorIds.value.length &&
      leadCandidates.value.some((lead) => lead.actor_id === creator)
    ) {
      leadActorIds.value = [creator];
    }
  },
  { immediate: true },
);

watch(
  [() => props.relatedTask, () => props.relation, projectReady],
  () => {
    const related = props.relatedTask;
    const relation = props.relation;
    if (!related || !relation || !projectReady.value) return;
    const key = `${related.id}:${relation}`;
    if (seededRelation === key) return;
    seededRelation = key;
    const kind = newTaskReferenceKind(relation);
    if (!kind) {
      parentChoice.value = related;
      return;
    }
    references.value = [{ taskId: related.id, kind }];
    referenceTasks.value = [related];
  },
  { immediate: true },
);

watch(
  () => props.agentId,
  (id) => {
    if (id) assigneeChoice.value = id;
  },
);

function onTagsChange(ids: string[]) {
  selectedTags.value = new Set(ids);
}

function onTagCreated() {
  void refreshTags();
}

function addPromptReference(task: Task) {
  if (!references.value.some((ref) => ref.taskId === task.id)) {
    references.value = [
      ...references.value,
      { taskId: task.id, kind: TaskReferenceKind.INFORMATIONAL },
    ];
  }
  if (!referenceTasks.value.some((item) => item.id === task.id)) {
    referenceTasks.value = [...referenceTasks.value, task];
  }
}

async function submit(desired: TaskState = TaskState.BACKLOG) {
  if (!prompt.value.trim() || saving.value || !projectReady.value) return;
  saving.value = true;
  submitting.value = desired;
  try {
    const hasAttachments = stagedCount.value > 0;
    const input: CreateTaskInput = {
      prompt: prompt.value.trim(),

      name: title.value.trim() || undefined,
      autoTitle: !title.value.trim(),
      priority: priority.value,
      size: size.value,
      projectId:
        fixedProjectId.value ||
        (projectChoice.value === NO_PROJECT ? undefined : projectChoice.value),
      assigneeActorId:
        assigneeChoice.value === AUTO
          ? undefined
          : actorIdFor(assigneeChoice.value),
      modelProvider: modelSelection.value.modelProvider || undefined,
      modelId: modelSelection.value.modelId || undefined,
      thinkingLevel: modelSelection.value.thinkingLevel || undefined,
      tagIds: [...selectedTags.value],
      references: references.value,
      teamIds: owningTeamIds.value,
      leadActorIds: leadActorIds.value,
      templateId: appliedTemplate.value?.templateId,
      templateVersion: appliedTemplate.value?.version,
      templateArgs: appliedTemplate.value?.args,

      templatePromptOverride: !!appliedTemplate.value,
      parentTaskId: parentChoice.value?.id,

      interactive: parentChoice.value ? interactive.value : undefined,

      originTaskId: props.relatedTask?.id,

      state: hasAttachments ? TaskState.BACKLOG : desired,
    };
    const task = await agency.createTask(input);
    let failed = false;
    if (hasAttachments) {
      failed = (await uploadStaged(task.id)) > 0;

      if (!failed && desired === TaskState.QUEUED) {
        await agency.queueTask(task.id);
      }
    }
    if (failed) {
      toast.error(t("express.toasts.uploadFailed"));
    } else if (desired === TaskState.BACKLOG) {
      toast.success(t("express.toasts.movedToBacklog"));
    } else {
      toast.success(t("express.toasts.queued"));
    }

    prompt.value = "";
    title.value = "";
    draftProject.value = "";
    clearStaged();
    references.value = [];
    referenceTasks.value = [];
    appliedTemplate.value = null;
    adoptPageProject();
    await nextTick();
    promptInput.value?.focus();
    emit("created", task);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
    submitting.value = null;
  }
}

function fillFromTemplate(payload: AppliedTemplate) {
  prompt.value = payload.prompt;
  title.value = payload.title;
  priority.value = payload.priority;
  size.value = payload.size;
  selectedTags.value = new Set(payload.tagIds);

  if (payload.teamIds.length) owningTeamIds.value = [...payload.teamIds];
  if (payload.leadActorIds.length)
    leadActorIds.value = [...payload.leadActorIds];
  appliedTemplate.value = payload;
}

function applyTemplate(payload: AppliedTemplate) {
  fillFromTemplate(payload);
  toast.success(`Loaded template “${payload.templateName}”`);
  void nextTick(() => promptInput.value?.focus());
}

function createFromTemplate(payload: AppliedTemplate, state: TaskState) {
  fillFromTemplate(payload);
  void submit(state);
}

function detachTemplate() {
  appliedTemplate.value = null;
}

onMounted(() => {
  void Promise.all([loadAgents(), loadProjects()]);
});

onBeforeUnmount(clearStaged);
</script>

<i18n lang="yaml">
en:
  express:
    selectProjectFirst: "Select a project first"
    toBacklog: "Put to Backlog"
    attachmentsHint: "Paste, drop, or pick files to attach them to the task"
    shortcutsShort: "Ctrl+Enter to create · Tab indents"
    interactive: "Interactive"
    noTeams: "No teams in this project"
    noMembers: "No members to pick"
    noProject: "A task must belong to a project. Create one or ask to be added."
    autoTitle: "Auto-generated title"
    titlePlaceholder: "Task title"
    toBacklogHint: "Create the task in Backlog, not queued yet"
    toQueueHint: "Create the task and queue it for the scheduler"
    promptPlaceholder: "Describe a task…"
    detachTemplate: "Detach the template (keeps the rendered prompt)"
    interactiveHint: "Surface questionnaires and extension dialogs for this sub-task; turn off for background work that must not wait for the user"
    searchTeams: "Search teams…"
    removeTeam: "Remove team"
    searchMembers: "Search members…"
    removeLead: "Remove lead"
    toasts:
      queued: "Task queued"
      movedToBacklog: "Task moved to backlog"
      uploadFailed: "Task saved to backlog — some attachments failed to upload"
    thatAgent: "That agent"
    noOwningTeam: "No owning team; the project default applies"
    owningTeams: "Members of these teams may prompt"
    shortcuts: "Tab indents · Ctrl+Enter creates the task · Ctrl+K deletes to the end of the line · {'@'} mentions · # tasks"
    inProject: "In {project}"
    templateHint: "Start from a task template"
    templateVersion: "Rendered from template version {version}"
    assignedTo: "Assigned to {agent}"
    leadsHint: "Leads are notified of the task's updates"
    agentNotMember: "{agent} is not a member of {project}; auto-assign is selected."
    theProject: "the project"
ru:
  express:
    selectProjectFirst: "Сначала выберите проект"
    toBacklog: "В бэклог"
    attachmentsHint: "Вставьте, перетащите или выберите файлы, чтобы прикрепить их к задаче"
    shortcutsShort: "Ctrl+Enter — создать · Tab — отступ"
    interactive: "Интерактивная"
    noTeams: "В этом проекте нет команд"
    noMembers: "Нет участников для выбора"
    noProject: "Задача должна принадлежать проекту. Создайте проект или попросите добавить вас."
    autoTitle: "Автоматическое название"
    titlePlaceholder: "Название задачи"
    toBacklogHint: "Создать задачу в бэклоге, без постановки в очередь"
    toQueueHint: "Создать задачу и поставить её в очередь планировщика"
    promptPlaceholder: "Опишите задачу…"
    detachTemplate: "Отвязать шаблон (промпт останется)"
    interactiveHint: "Показывать опросники и диалоги расширений для этой подзадачи; отключите для фоновой работы, которая не должна ждать пользователя"
    searchTeams: "Поиск команд…"
    removeTeam: "Убрать команду"
    searchMembers: "Поиск участников…"
    removeLead: "Убрать руководителя"
    toasts:
      queued: "Задача поставлена в очередь"
      movedToBacklog: "Задача перемещена в бэклог"
      uploadFailed: "Задача сохранена в бэклог — часть вложений не загрузилась"
    thatAgent: "Этот агент"
    noOwningTeam: "Нет команды-владельца; действует значение проекта по умолчанию"
    owningTeams: "Участники этих команд могут отправлять промпты"
    shortcuts: "Tab — отступ · Ctrl+Enter — создать задачу · Ctrl+K — удалить до конца строки · {'@'} — упоминания · # — задачи"
    inProject: "В проекте {project}"
    templateHint: "Начать с шаблона задачи"
    templateVersion: "Сформировано из версии шаблона {version}"
    assignedTo: "Назначено: {agent}"
    leadsHint: "Руководители получают уведомления об изменениях задачи"
    agentNotMember: "{agent} не входит в проект {project}; выбрано автоназначение."
    theProject: "проект"
</i18n>
