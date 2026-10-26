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
    form.flex.flex-col.gap-3(@submit.prevent="submit")
      .flex.flex-col.gap-2
        .flex.items-center.justify-between.gap-2
          WritePreviewTabs(v-model="tab" test-id="express-tab")
          .flex.shrink-0.items-center.gap-2
            .bg-muted.inline-flex.items-center.rounded-md.border(class="gap-0.5 p-0.5")
              button.rounded.px-2.py-1.text-xs.font-medium.transition-colors(
                v-for="option in TARGETS"
                :key="option.value"
                type="button"
                :class="target === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
                :aria-pressed="target === option.value"
                :title="option.hint"
                @click="target = option.value"
              ) {{ option.label }}
            Button(
              type="submit"
              :disabled="saving || !prompt.trim() || !projectReady"
            )
              Spinner(v-if="saving" data-icon="inline-start")
              PlusIcon(v-else data-icon="inline-start")
              span {{ submitLabel }}
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
          MarkdownEditor(
            v-if="tab === 'write'"
            ref="promptInput"
            v-model="prompt"
            placeholder="Describe a task…"
            resizable
            test-id="express-editor"
            aria-label="Task prompt"
            title="Tab indents · Ctrl+Enter creates the task · Ctrl+K deletes to the end of the line"
            @submit="submit"
            @paste="onPaste"
          )
          .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm(
            v-else
            data-testid="express-preview"
          )
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :style="INHERIT_PROSE"
              :text="prompt"
            )
        .flex.flex-wrap.items-center.gap-2
          Button(
            type="button"
            variant="outline"
            size="sm"
            title="Attach files to the task"
            @click="pickFiles"
          )
            PaperclipIcon(data-icon="inline-start")
            span Attach
          Button(
            type="button"
            variant="outline"
            size="sm"
            title="Draw a sketch"
            @click="openNewSketch"
          )
            PencilIcon(data-icon="inline-start")
            span Sketch
          span.text-muted-foreground.text-xs(v-if="!stagedCount")
            | Paste, drop, or pick files to attach them to the task
          span.text-muted-foreground.ml-auto.text-xs(
            data-testid="composer-submit-hint"
          )
            | Ctrl+Enter to create · Tab indents
          input.hidden(
            ref="fileInput"
            type="file"
            multiple
            @change="onFileChange"
          )
        StagedAttachmentList(:items="stagedItems" @remove="removeStaged")

      .rounded-lg.border.p-2(class="bg-muted/30")
        .flex.flex-wrap.items-center.gap-2
          Button(
            v-if="fixedProjectId"
            type="button"
            variant="outline"
            size="sm"
            disabled
            :title="`In ${fixedProjectLabel}`"
          )
            span {{ fixedProjectLabel }}
          Select(v-else v-model="projectChoice")
            SelectTrigger.w-40(size="sm" title="Project")
              SelectValue(placeholder="Select project")
            SelectContent
              SelectGroup
                SelectItem(
                  v-for="project in projectOptions"
                  :key="project.id"
                  :value="project.id"
                ) {{ project.name }}

          Button(
            v-if="fixedAgentId"
            type="button"
            variant="outline"
            size="sm"
            disabled
            :title="`Assigned to ${fixedAgentLabel}`"
          )
            span {{ fixedAgentLabel }}
          Select(v-else v-model="assigneeChoice")
            SelectTrigger.w-40(size="sm" title="Agent")
              SelectValue(placeholder="Auto-assign")
            SelectContent
              SelectGroup
                SelectItem(:value="AUTO") Auto-assign
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
              v-for="option in PRIORITIES"
              :key="option.value"
              type="button"
              class="py-0.5"
              :class="priority === option.value ? `bg-card shadow-sm ring-1 ring-border ${taskPriorityTextClass(option.value)}` : 'text-muted-foreground hover:text-foreground'"
              @click="priority = option.value"
            ) {{ option.label }}

          .bg-muted.inline-flex.items-center.gap-1.rounded-lg.border(class="p-0.5")
            button.rounded-md.px-2.text-xs.font-medium.transition-colors(
              v-for="option in SIZES"
              :key="option.value"
              type="button"
              class="py-0.5"
              :class="size === option.value ? `bg-card shadow-sm ring-1 ring-border ${taskSizeTextClass(option.value)}` : 'text-muted-foreground hover:text-foreground'"
              @click="size = option.value"
            ) {{ option.label }}

          DropdownMenu
            DropdownMenuTrigger(as-child)
              Button(
                type="button"
                variant="outline"
                size="sm"
                :title="selectedProjectId ? owningTeamsTitle : 'Select a project first'"
                :disabled="!selectedProjectId"
              )
                UsersIcon(data-icon="inline-start")
                span Teams
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
              title="Remove team"
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
                span Leads
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
              title="Remove lead"
              @click="toggleLead(lead.actor_id)"
            )
              XIcon.size-3

          DropdownMenu
            DropdownMenuTrigger(as-child)
              Button(
                type="button"
                variant="outline"
                size="sm"
                :title="selectedProjectId ? 'Tags' : 'Select a project first'"
                :disabled="!selectedProjectId"
              )
                TagIcon(data-icon="inline-start")
                span Tags
            DropdownMenuContent.w-48(align="start")
              DropdownMenuLabel Tags
              DropdownMenuSeparator
              p.text-muted-foreground.px-2.py-1.text-xs(
                v-if="!activeTags.length"
              )
                | No tags yet
              DropdownMenuCheckboxItem(
                v-for="tag in activeTags"
                :key="tag.id"
                :model-value="selectedTags.has(tag.id)"
                @update:model-value="toggleTag(tag.id)"
              ) {{ tag.name }}

          span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
            v-for="tag in selectedTagList"
            :key="tag.id"
            class="py-0.5"
            :style="tagStyle(tag.color)"
          )
            span.rounded-full.bg-current(class="size-1.5")
            span {{ tag.name }}
            button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
              type="button"
              class="p-0.5 hover:opacity-100"
              title="Remove tag"
              @click="toggleTag(tag.id)"
            )
              XIcon.size-3

      TaskReferencePicker(
        v-model="references"
        :project-id="selectedProjectId"
        :disabled="!projectReady"
      )

      p.text-muted-foreground.text-xs(v-if="noProjects")
        | A task must belong to a project. Create one or ask to be added.

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
import MarkdownText from "@/components/MarkdownText.vue";
import StagedAttachmentList from "@/components/attachments/StagedAttachmentList.vue";
import TaskModelPicker from "@/components/tasks/TaskModelPicker.vue";
import TaskReferencePicker from "@/components/tasks/TaskReferencePicker.vue";
import WritePreviewTabs from "@/components/tasks/WritePreviewTabs.vue";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
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
import { Spinner } from "@/components/ui/spinner";
import { useAgents } from "@/composables/useAgents";
import { useDraft } from "@/composables/useDraft";
import { useProjects } from "@/composables/useProjects";
import { useSketchEditor } from "@/composables/useSketchEditor";
import { useStagedAttachments } from "@/composables/useStagedAttachments";
import { useTags } from "@/composables/useTags";
import { currentAccount } from "@/lib/account";
import { composerDraftKey } from "@/lib/draft";
import { INHERIT_PROSE } from "@/lib/markdown";
import { sortProjectsByName } from "@/lib/project";
import {
  projectAccessApi,
  type LeadCandidate,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import { agency } from "@/lib/sdk";
import { sketchSceneFilename, toStoredSnapshot } from "@/lib/sketches";
import {
  EMPTY_TASK_MODEL_SELECTION,
  TASK_SIZES,
  tagStyle,
  taskPriorityLabel,
  taskPriorityTextClass,
  taskSizeLabel,
  taskSizeTextClass,
  type TaskModelSelection,
} from "@/lib/task";
import { teamApi } from "@/lib/team";
import {
  TaskPriority,
  TaskSize,
  TaskState,
  type CreateTaskInput,
  type Project,
  type TaskReferenceInput,
} from "@cherrypi/sdk";
import {
  PaperclipIcon,
  PencilIcon,
  PlusIcon,
  TagIcon,
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
import { toast } from "vue3-toastify";

const NO_PROJECT = "__none__";
const AUTO = "__auto__";

const PRIORITIES = [
  { value: TaskPriority.LOW, label: taskPriorityLabel(TaskPriority.LOW) },
  {
    value: TaskPriority.NORMAL,
    label: taskPriorityLabel(TaskPriority.NORMAL, true),
  },
  { value: TaskPriority.HIGH, label: taskPriorityLabel(TaskPriority.HIGH) },
  {
    value: TaskPriority.CRITICAL,
    label: taskPriorityLabel(TaskPriority.CRITICAL),
  },
];

const SIZES = TASK_SIZES.map((option) => ({
  ...option,
  label: taskSizeLabel(option.value, option.value === TaskSize.MEDIUM),
}));

const props = defineProps<{
  agentId?: string;

  projectId?: string;

  bare?: boolean;
}>();

const BARE_CARD =
  "gap-0 rounded-none bg-transparent py-0 text-inherit ring-0 ring-transparent";

const emit = defineEmits<{ created: [] }>();

const { agents, loadAgents, actorIdFor } = useAgents();
const { active: allProjects, loadProjects } = useProjects();

const prompt = useDraft(() =>
  composerDraftKey({ agent: props.agentId, project: props.projectId }),
);

const tab = ref<"write" | "preview">("write");
const priority = ref(TaskPriority.NORMAL);
const size = ref(TaskSize.MEDIUM);
const assigneeChoice = ref(props.agentId || AUTO);
const projectChoice = ref(props.projectId || NO_PROJECT);

const modelSelection = ref<TaskModelSelection>({
  ...EMPTY_TASK_MODEL_SELECTION,
});
const selectedTags = ref<Set<string>>(new Set());

const defaultTagsProject = ref("");
const references = ref<TaskReferenceInput[]>([]);

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
const selectedLeadList = computed(() =>
  leadCandidates.value.filter((lead) =>
    leadActorIds.value.includes(lead.actor_id),
  ),
);
const owningTeamsTitle = computed(() =>
  owningTeamIds.value.length
    ? "Members of these teams may prompt"
    : "No owning team; the project default applies",
);
const leadsTitle = "Leads are notified of the task's updates";
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

const target = ref<TaskState>(TaskState.DRAFT);
const TARGETS = [
  {
    value: TaskState.DRAFT,
    label: "Draft",
    hint: "Save a draft, never scheduled",
  },
  {
    value: TaskState.BACKLOG,
    label: "Backlog",
    hint: "Likely to run, not queued yet",
  },
  {
    value: TaskState.QUEUED,
    label: "Queue",
    hint: "Queue for the scheduler now",
  },
];
const submitLabel = computed(() => {
  switch (target.value) {
    case TaskState.DRAFT:
      return "Save draft";
    case TaskState.BACKLOG:
      return "Move to backlog";
    default:
      return "Queue task";
  }
});
const agentProjects = ref<Project[]>([]);

const projectMembers = ref<string[]>([]);
const saving = ref(false);
const promptInput = ref<InstanceType<typeof MarkdownEditor> | null>(null);

const {
  items: stagedItems,
  count: stagedCount,
  addFiles: addStagedFiles,
  addSketch: addStagedSketch,
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
  onSave: onSketchSave,
} = useSketchEditor({
  siblings: () => [],
  persist: async ({ png, snapshot }) => addStagedSketch(png, snapshot),
});

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
  return item?.displayName || item?.username || "Agent";
});

const fixedProjectLabel = computed(() => {
  const item = allProjects.value.find(
    (project) => project.id === fixedProjectId.value,
  );
  return item?.name || "Project";
});

const projectReady = computed(
  () => !!fixedProjectId.value || projectChoice.value !== NO_PROJECT,
);

const projectOptions = computed(() =>
  sortProjectsByName(
    assigneeChoice.value === AUTO ? allProjects.value : agentProjects.value,
  ),
);

const noProjects = computed(
  () => !fixedProjectId.value && projectOptions.value.length === 0,
);

const selectedProjectId = computed(
  () =>
    fixedProjectId.value ||
    (projectChoice.value === NO_PROJECT ? "" : projectChoice.value),
);

const { active: activeTags } = useTags(() => selectedProjectId.value);

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

const assigneeOptions = computed(() => {
  if (!fixedProjectId.value) return agents.value;
  const memberIds = new Set(projectMembers.value);
  return agents.value.filter((agent) => memberIds.has(agent.id));
});

watch(
  fixedProjectId,
  async (id) => {
    if (!id) {
      projectMembers.value = [];
      return;
    }
    projectChoice.value = id;
    try {
      const members = await agency.listProjectAgents(id);
      projectMembers.value = members.map((member) => member.agentId);
    } catch {
      projectMembers.value = [];
    }
  },
  { immediate: true },
);

const selectedTagList = computed(() =>
  activeTags.value.filter((tag) => selectedTags.value.has(tag.id)),
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
  }
});

watch(
  selectedProjectId,
  async (id) => {
    references.value = [];
    selectedTags.value = new Set();
    defaultTagsProject.value = "";
    owningTeamIds.value = [];
    leadActorIds.value = [];
    projectTeams.value = [];
    leadCandidates.value = [];
    if (!id) return;
    try {
      const access = await projectAccessApi.access(id);
      if (selectedProjectId.value !== id) return;
      projectTeams.value = access.teams;
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
    if (leadCandidates.value.some((lead) => lead.actor_id === creator)) {
      leadActorIds.value = [creator];
    }
  },
  { immediate: true },
);

watch(
  () => props.agentId,
  (id) => {
    if (id) assigneeChoice.value = id;
  },
);

function toggleTag(id: string) {
  const next = new Set(selectedTags.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selectedTags.value = next;
}

async function submit() {
  if (!prompt.value.trim() || saving.value || !projectReady.value) return;
  saving.value = true;
  try {
    const desired = target.value;
    const hasAttachments = stagedCount.value > 0;
    const input: CreateTaskInput = {
      prompt: prompt.value.trim(),

      autoTitle: true,
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

      state: hasAttachments ? TaskState.DRAFT : desired,
    };
    const task = await agency.createTask(input);
    let failed = false;
    if (hasAttachments) {
      failed = (await uploadStaged(task.id)) > 0;

      if (!failed && desired !== TaskState.DRAFT) {
        await agency.promoteTask(task.id);
        if (desired === TaskState.QUEUED) await agency.queueTask(task.id);
      }
    }
    if (failed) {
      toast.error("Task saved as draft — some attachments failed to upload");
    } else if (desired === TaskState.DRAFT) {
      toast.success("Task saved as draft");
    } else if (desired === TaskState.BACKLOG) {
      toast.success("Task moved to backlog");
    } else {
      toast.success("Task queued");
    }

    prompt.value = "";
    clearStaged();
    references.value = [];

    tab.value = "write";
    await nextTick();
    promptInput.value?.focus();
    emit("created");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void Promise.all([loadAgents(), loadProjects()]);
});

onBeforeUnmount(clearStaged);
</script>
