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
//- `bare` drops the card chrome so the form can sit inside a dialog surface
//- without nesting a second bordered panel.
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
          Textarea.field-sizing-fixed.max-h-96.min-h-16.w-full.resize-y(
            v-if="tab === 'write'"
            ref="promptInput"
            v-model="prompt"
            placeholder="Describe a task…"
            @keydown.enter="onKeydown"
            @paste="onPaste"
          )
          .text-foreground.border-border.w-full.rounded-lg.border.px-4.py-3.text-sm(
            v-else
            data-testid="express-preview"
          )
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :style="inheritProse"
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
            | Ctrl+Enter to create
          input.hidden(
            ref="fileInput"
            type="file"
            multiple
            @change="onFileChange"
          )
        StagedAttachmentList(:items="stagedItems" @remove="removeStaged")

      .rounded-lg.border.p-2(class="bg-muted/30")
        .flex.flex-wrap.items-center.gap-2
          // A task must land in a project; a pinned project is read-only.
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

          // On an agent page the assignee is fixed; elsewhere the scheduler may
          // pick one. A pinned agent narrows the project list to its memberships.
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
import { Textarea } from "@/components/ui/textarea";
import { useAgents } from "@/composables/useAgents";
import { useDraft } from "@/composables/useDraft";
import { useProjects } from "@/composables/useProjects";
import { useSketchEditor } from "@/composables/useSketchEditor";
import { useStagedAttachments } from "@/composables/useStagedAttachments";
import { useTags } from "@/composables/useTags";
import { currentAccount } from "@/lib/account";
import { composerDraftKey } from "@/lib/draft";
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

// Sentinels for the optionless choices (reka-ui select items cannot carry an
// empty-string value).
const NO_PROJECT = "__none__";
const AUTO = "__auto__";

// The quick creator only spells out the default dimension: the pre-selected
// "Normal"/"Medium" reads ambiguously, while the other choices are picked
// deliberately and are clear in context.
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
  // When set, the task is pinned to this agent and the picker is disabled.
  agentId?: string;
  // When set, the task is pinned to this project and the picker is disabled.
  // The assignee options narrow to the project's members.
  projectId?: string;
  // Drop the card surface; the surrounding dialog already frames the form.
  bare?: boolean;
}>();

// Passed as the card's class so `cn` merges it after the defaults and
// tailwind-merge drops the ring, fill, and padding that would otherwise double
// up inside a dialog.
const BARE_CARD =
  "gap-0 rounded-none bg-transparent py-0 text-inherit ring-0 ring-transparent";

const emit = defineEmits<{ created: [] }>();

const { agents, loadAgents, actorIdFor } = useAgents();
const { active: allProjects, loadProjects } = useProjects();

const prompt = useDraft(() =>
  composerDraftKey({ agent: props.agentId, project: props.projectId }),
);
// The editor's Markdown tab; preview renders the draft exactly as the prompt
// box does.
const tab = ref<"write" | "preview">("write");
// Render Markdown in the surrounding text color instead of the typography
// plugin's fixed palette, so the preview stays legible.
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
const priority = ref(TaskPriority.NORMAL);
const size = ref(TaskSize.MEDIUM);
const assigneeChoice = ref(props.agentId || AUTO);
const projectChoice = ref(props.projectId || NO_PROJECT);
// A per-task model/thinking override; cleared whenever the assignee changes so
// it can never be sent to an agent that does not offer it. With the default
// selection the scheduler applies the agent's owner-managed defaults.
const modelSelection = ref<TaskModelSelection>({
  ...EMPTY_TASK_MODEL_SELECTION,
});
const selectedTags = ref<Set<string>>(new Set());
// The project whose default tags were last applied, so a tag-list refresh
// never re-adds a default the creator removed for this task.
const defaultTagsProject = ref("");
const references = ref<TaskReferenceInput[]>([]);
// Ownership: the task's owning teams (project teams whose members may prompt)
// and its leads (project members shown on the task and notified of its
// updates). Both lists are loaded for the selected project.
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
// The initial lifecycle state: draft (never scheduled), backlog (likely to
// run, not queued), or queued immediately. A task with attachments is created
// as a draft and promoted after the uploads land.
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
// Member agent ids of the pinned project; empty while unset.
const projectMembers = ref<string[]>([]);
const saving = ref(false);
const promptInput = ref<InstanceType<typeof Textarea> | null>(null);

// Files and sketches are staged in the browser; the task they belong to does
// not exist until submit, so uploads happen against the freshly created task.
const {
  items: stagedItems,
  count: stagedCount,
  addFiles: addStagedFiles,
  addSketch: addStagedSketch,
  remove: removeStaged,
  clear: clearStaged,
} = useStagedAttachments();
const fileInput = ref<HTMLInputElement>();

// The sketch editor lazily loads tldraw; a captured sketch is staged until the
// task exists.
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

// A depth counter tracks enter/leave across child elements so the drop overlay
// only shows while a file drag is over the editor.
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

// A file paste (an image from the clipboard, or files from the file manager)
// attaches; a text paste is left to the textarea.
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

/** Upload every staged item to the new task; returns the failure count. A
 * sketch uploads its PNG first so the sidecar can key on the image id. */
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

// A different agent has a different model catalog, so a selection made for the
// previous one is not carried over.
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

// A task must belong to a project, so the form cannot submit until one is
// pinned or picked.
const projectReady = computed(
  () => !!fixedProjectId.value || projectChoice.value !== NO_PROJECT,
);

// Only a member agent can run a task in a project, so pinning an agent narrows
// the project list to its memberships.
const projectOptions = computed(() =>
  sortProjectsByName(
    assigneeChoice.value === AUTO ? allProjects.value : agentProjects.value,
  ),
);

// No selectable projects means there is nowhere to put a new task.
const noProjects = computed(
  () => !fixedProjectId.value && projectOptions.value.length === 0,
);

// The project a new task will land in, used to scope reference candidates and
// the tag list (tags belong to a project).
const selectedProjectId = computed(
  () =>
    fixedProjectId.value ||
    (projectChoice.value === NO_PROJECT ? "" : projectChoice.value),
);

const { active: activeTags } = useTags(() => selectedProjectId.value);

// Dropping a tag that the newly picked project does not define keeps the form
// from submitting a cross-project tag. When a project's tag list first
// arrives, its default tags are selected; they stay removable afterwards.
watch(activeTags, (list) => {
  const valid = new Set(list.map((tag) => tag.id));
  const next = new Set([...selectedTags.value].filter((id) => valid.has(id)));
  if (next.size !== selectedTags.value.size) selectedTags.value = next;

  const projectID = selectedProjectId.value;
  if (!projectID || defaultTagsProject.value === projectID) return;
  // Ignore the previous project's list while the new one loads.
  if (list.some((tag) => tag.projectId !== projectID)) return;
  selectedTags.value = new Set(
    list.filter((tag) => tag.isDefault).map((tag) => tag.id),
  );
  defaultTagsProject.value = projectID;
});

// Pinning a project narrows the assignee list to its members; otherwise any
// agent may be pinned (or the scheduler may pick one).
const assigneeOptions = computed(() => {
  if (!fixedProjectId.value) return agents.value;
  const memberIds = new Set(projectMembers.value);
  return agents.value.filter((agent) => memberIds.has(agent.id));
});

// Load the pinned project's members and pin the project choice.
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

// Load the pinned agent's memberships; an auto-assigned task may use any. A
// pinned project already fixes the choice, so skip the membership lookup.
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

// A project the newly chosen agent is not a member of disappears; drop it.
// A pinned project is fixed and never dropped.
watch(projectOptions, (options) => {
  if (fixedProjectId.value) return;
  if (
    projectChoice.value !== NO_PROJECT &&
    !options.some((project) => project.id === projectChoice.value)
  ) {
    projectChoice.value = NO_PROJECT;
  }
});

// A reference must belong to the task's project, so changing the target
// invalidates any picks made for the previous project, and the owning teams
// and lead candidates are reloaded for the new project.
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
    } catch {
      // A failed load only hides the team picker; the task can still be saved.
    }
    try {
      const candidates = await projectAccessApi.leadCandidates(id);
      if (selectedProjectId.value !== id) return;
      leadCandidates.value = candidates;
    } catch {
      // Likewise, the lead picker degrades to empty.
    }
    // The creator's teams that can access this project are the task's owning
    // teams, and the creator is its first lead so they are notified without an
    // extra pick. A lead may be any project member, so the default does not
    // depend on an owning-team membership.
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
    } catch {
      // Without the membership lookup the server derives the teams itself.
    }
    if (leadCandidates.value.some((lead) => lead.actor_id === creator)) {
      leadActorIds.value = [creator];
    }
  },
  { immediate: true },
);

// The agent page may be reused when navigating between agents (same route
// record), so keep the fixed assignee in sync with the prop.
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

// Ctrl/Cmd+Enter creates the task; a bare Enter or Shift+Enter keeps the
// textarea's default behavior and inserts a newline, so a prompt can be
// multi-line. Ignore the Enter that confirms an IME candidate so composition
// isn't swallowed.
function onKeydown(event: KeyboardEvent) {
  if (event.key !== "Enter" || event.isComposing) return;
  if (!event.ctrlKey && !event.metaKey) return;
  event.preventDefault();
  void submit();
}

async function submit() {
  if (!prompt.value.trim() || saving.value || !projectReady.value) return;
  saving.value = true;
  try {
    const desired = target.value;
    const hasAttachments = stagedCount.value > 0;
    const input: CreateTaskInput = {
      prompt: prompt.value.trim(),
      // Auto-titling and extension dialogs stay on; the backend still supports
      // per-task values for these via the API.
      autoTitle: true,
      interactive: true,
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
      // A task with files is created as a draft so uploads land before the
      // scheduler starts it; it is moved to the selected state once they are
      // in.
      state: hasAttachments ? TaskState.DRAFT : desired,
    };
    const task = await agency.createTask(input);
    let failed = false;
    if (hasAttachments) {
      failed = (await uploadStaged(task.id)) > 0;
      // A failed upload leaves the task a draft so the files can be retried.
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
    // Keep every setting so another task can be drafted swiftly; clear the
    // prompt, the staged files and the one-shot reference picks, then put the
    // caret back.
    prompt.value = "";
    clearStaged();
    references.value = [];
    // Return to Write so the next task starts in the textarea.
    tab.value = "write";
    await nextTick();
    promptInput.value?.$el?.focus();
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

// Drop any object URLs a staged preview still holds when the form goes away.
onBeforeUnmount(clearStaged);
</script>
