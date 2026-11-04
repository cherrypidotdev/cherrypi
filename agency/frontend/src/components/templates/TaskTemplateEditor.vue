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
Dialog(:open="open" @update:open="requestOpen")
  DialogContent.overflow-y-auto.overscroll-contain(class="max-h-[92vh] sm:max-w-4xl")
    DialogHeader
      DialogTitle {{ isEditing ? "Edit task template" : "New task template" }}
      DialogDescription
        | A Jinja prompt with declared arguments and task defaults. Saving a
        | changed template appends an immutable version; tasks pin the version
        | they were created from.
    Tabs(v-model="tab")
      TabsList
        TabsTrigger(value="content") Content
        TabsTrigger(value="defaults") Defaults
        TabsTrigger(value="preview") Preview
        TabsTrigger(v-if="isEditing" value="versions") Versions
      TabsContent.flex.flex-col.gap-4(value="content")
        FieldGroup
          Field
            FieldLabel(for="template-name") Name
            Input#template-name(v-model="name" placeholder="Bug report")
          Field
            FieldLabel(for="template-description") Description
            Input#template-description(
              v-model="description"
              placeholder="What this template is for"
            )
          Field
            FieldLabel(for="template-body") Prompt body (Jinja)
            Textarea#template-body.font-mono(
              v-model="content.body"
              placeholder="Fix the {{ component }} crash.\n\nSteps:\n{{ steps }}"
              rows="10"
            )
            FieldDescription {{ BODY_HINT }}
          Field
            FieldLabel(for="template-title") Title template (optional)
            Input#template-title.font-mono(
              v-model="content.titleTemplate"
              placeholder="Fix {{ component }}"
            )
            FieldDescription
              | Rendered into the task's title. Empty leaves the title to the
              | agent's auto-title.

        .flex.flex-col.gap-2
          .flex.items-center.justify-between.gap-2
            .flex.flex-col
              span.text-sm.font-medium Arguments
              span.text-muted-foreground.text-xs
                | Every variable the body or title references must be declared here.
            Button(type="button" variant="outline" size="sm" @click="addArg")
              PlusIcon(data-icon="inline-start")
              span Add argument
          p.text-muted-foreground.text-xs(v-if="!content.args?.length")
            | No arguments yet.
          .flex.flex-col.gap-2(
            v-for="(arg, index) in content.args"
            :key="index"
          )
            .grid.grid-cols-1.gap-2(
              class="sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_10rem_auto]"
            )
              Input.font-mono(
                v-model="arg.key"
                placeholder="key"
                aria-label="Argument key"
              )
              Input(
                v-model="arg.label"
                placeholder="Label"
                aria-label="Argument label"
              )
              Select(
                :model-value="arg.type ?? TaskTemplateArgType.TEXT"
                @update:model-value="setArgTypeFromSelect(arg, $event)"
              )
                SelectTrigger.w-full(aria-label="Argument type")
                  SelectValue
                SelectContent
                  SelectItem(
                    v-for="type in ARG_TYPES"
                    :key="type"
                    :value="type"
                  ) {{ ARG_TYPE_LABELS[type] }}
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                title="Remove argument"
                aria-label="Remove argument"
                @click="removeArg(index)"
              )
                XIcon.size-3
            .grid.grid-cols-1.gap-2(
              class="sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
            )
              Input(
                v-model="arg.defaultValue"
                :placeholder="defaultPlaceholder(arg)"
                aria-label="Argument default"
              )
              Input(
                v-if="arg.type === TaskTemplateArgType.SELECT"
                placeholder="Options, comma-separated"
                aria-label="Argument options"
                :model-value="formatOptions(arg.options)"
                @update:model-value="setArgOptions(arg, $event)"
              )
              Input(
                v-else
                v-model="arg.description"
                placeholder="Description (optional)"
                aria-label="Argument description"
              )
              label.flex.items-center.gap-2.text-xs.whitespace-nowrap
                Switch(
                  size="sm"
                  :model-value="arg.required ?? false"
                  @update:model-value="arg.required = $event"
                )
                span Required
            p.text-destructive.text-xs(
              v-if="arg.type === TaskTemplateArgType.SELECT && !arg.options?.length"
            )
              | A select needs at least one option.

      TabsContent.flex.flex-col.gap-4(value="defaults")
        p.text-muted-foreground.text-xs
          | Pre-filled in the composer. Explicit task fields override them, and
          | a template never pins the assignee or the model.
        .grid.grid-cols-2.gap-3
          Field
            FieldLabel Priority
            Select(
              :model-value="content.defaultPriority ?? TaskPriority.NORMAL"
              @update:model-value="setDefaultPriority"
            )
              SelectTrigger.w-full
                SelectValue
              SelectContent
                SelectItem(
                  v-for="option in PRIORITIES"
                  :key="option.value"
                  :value="option.value"
                ) {{ option.label }}
          Field
            FieldLabel Size
            Select(
              :model-value="content.defaultSize ?? TaskSize.MEDIUM"
              @update:model-value="setDefaultSize"
            )
              SelectTrigger.w-full
                SelectValue
              SelectContent
                SelectItem(
                  v-for="option in TASK_SIZES"
                  :key="option.value"
                  :value="option.value"
                ) {{ option.label }}
        .flex.flex-col.gap-2
          span.text-sm.font-medium Tags
          .flex.flex-wrap.items-center.gap-1
            span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
              v-for="tag in selectedTags"
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
            span.text-muted-foreground.text-xs(v-if="!selectedTags.length")
              | None.
            DropdownMenu
              DropdownMenuTrigger(as-child)
                Button(type="button" variant="outline" size="sm")
                  TagIcon(data-icon="inline-start")
                  span Add tag
              DropdownMenuContent.w-48(align="start")
                DropdownMenuLabel Tags
                DropdownMenuSeparator
                p.text-muted-foreground.px-2.py-1.text-xs(
                  v-if="!activeTags.length"
                )
                  | No tags in this project
                DropdownMenuCheckboxItem(
                  v-for="tag in activeTags"
                  :key="tag.id"
                  :model-value="(content.defaultTagIds ?? []).includes(tag.id)"
                  @update:model-value="toggleTag(tag.id)"
                ) {{ tag.name }}
        .flex.flex-col.gap-2
          span.text-sm.font-medium Owning teams
          .flex.flex-wrap.items-center.gap-1
            span.inline-flex.items-center.gap-1.rounded-full.border.pr-1.pl-2.text-xs(
              v-for="team in selectedTeams"
              :key="team.team_id"
              class="py-0.5"
            )
              UsersIcon.size-3.shrink-0
              span {{ team.team_name }}
              button.inline-flex.shrink-0.items-center.rounded-full.opacity-60.transition-opacity(
                type="button"
                class="p-0.5 hover:opacity-100"
                title="Remove team"
                @click="toggleTeam(team.team_id)"
              )
                XIcon.size-3
            span.text-muted-foreground.text-xs(v-if="!selectedTeams.length")
              | None; the project default applies.
            DropdownMenu
              DropdownMenuTrigger(as-child)
                Button(type="button" variant="outline" size="sm")
                  UsersIcon(data-icon="inline-start")
                  span Add team
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
                  :model-value="(content.defaultTeamIds ?? []).includes(team.team_id)"
                  @update:model-value="toggleTeam(team.team_id)"
                ) {{ team.team_name }}
        .flex.flex-col.gap-2
          span.text-sm.font-medium Leads
          .flex.flex-wrap.items-center.gap-1
            span.inline-flex.items-center.gap-1.rounded-full.border.pr-2.pl-1.text-xs(
              v-for="lead in selectedLeads"
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
            span.text-muted-foreground.text-xs(v-if="!selectedLeads.length")
              | None.
            DropdownMenu
              DropdownMenuTrigger(as-child)
                Button(type="button" variant="outline" size="sm")
                  UserCheckIcon(data-icon="inline-start")
                  span Add lead
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
                  :model-value="(content.defaultLeadActorIds ?? []).includes(candidate.actor_id)"
                  @update:model-value="toggleLead(candidate.actor_id)"
                )
                  ActorAvatar(
                    :src="candidate.picture_url"
                    :name="candidate.name"
                    :username="candidate.username"
                    :size="18"
                  )
                  span {{ candidate.name || candidate.username }}

      TabsContent.flex.flex-col.gap-3(value="preview")
        p.text-muted-foreground.text-xs
          | Rendered by the agency with every argument's default value, exactly
          | as a task created from this template would be.
        .flex.flex-col.gap-1(v-if="previewError")
          p.text-destructive.text-sm {{ previewError }}
        template(v-else-if="preview")
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Title
            span.text-sm.font-medium {{ preview.title || "— (auto-title)" }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs Prompt
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :text="preview.prompt"
            )
          .flex.flex-wrap.items-center.gap-2.text-xs
            Badge(variant="secondary") {{ taskPriorityLabel(preview.priority) }}
            Badge(variant="secondary") {{ taskSizeLabel(preview.size) }}
            Badge(v-if="preview.tagIds.length" variant="secondary")
              | {{ preview.tagIds.length }} tag(s)
        p.text-muted-foreground.text-xs(v-else)
          | Rendering…
        .flex.flex-col.gap-1(v-if="validationErrors.length")
          span.text-destructive.text-xs.font-medium Validation
          p.text-destructive.text-xs(
            v-for="error in validationErrors"
            :key="error"
          )
            | {{ error }}
        .flex.flex-col.gap-1(v-if="validationWarnings.length")
          span.text-xs.font-medium Warnings
          p.text-muted-foreground.text-xs(
            v-for="warning in validationWarnings"
            :key="warning"
          )
            | {{ warning }}

      TabsContent.flex.flex-col.gap-3(v-if="isEditing" value="versions")
        p.text-muted-foreground.text-xs
          | Versions are immutable. Tasks keep the version and arguments they
          | were created with, so editing only affects new tasks.
        p.text-muted-foreground.text-xs(v-if="!versions.length") No versions yet.
        .flex.items-center.justify-between.gap-2.rounded-md.border.px-3.py-2(
          v-for="version in versions"
          :key="String(version.version)"
        )
          .flex.flex-col
            span.text-sm.font-medium Version {{ version.version }}
            span.text-muted-foreground.text-xs
              | {{ formatTimestamp(version.createdAt) }}
              | · {{ version.content?.args?.length ?? 0 }} argument(s)
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="restoreVersion(version.version)"
            :disabled="busy || Number(version.version) === Number(template?.latestVersion ?? 0)"
          )
            ArchiveRestoreIcon(data-icon="inline-start")
            span Restore
    p.text-destructive.text-sm(v-if="error") {{ error }}
    DialogFooter
      Button(type="button" variant="outline" @click="requestClose")
        | Cancel
      Button(
        type="button"
        @click="save"
        :disabled="busy || !name.trim() || !content.body.trim()"
      )
        Spinner(v-if="busy" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ isEditing ? "Save" : "Create" }}

Dialog(v-model:open="discardOpen")
  DialogContent(class="sm:max-w-md")
    DialogHeader
      DialogTitle Discard unsaved changes?
      DialogDescription
        | This template has edits that were never saved. Discarding them cannot
        | be undone.
    DialogFooter
      Button(type="button" variant="outline" @click="discardOpen = false")
        | Keep editing
      Button(type="button" variant="destructive" @click="discardChanges")
        | Discard
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import MarkdownText from "@/components/MarkdownText.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useTags } from "@/composables/useTags";
import { formatTimestamp } from "@/lib/format";
import {
  projectAccessApi,
  projectTeamsWithInherited,
  type LeadCandidate,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import { agency } from "@/lib/sdk";
import {
  TASK_SIZES,
  tagStyle,
  taskPriorityLabel,
  taskSizeLabel,
} from "@/lib/task";
import {
  ARG_TYPES,
  ARG_TYPE_LABELS,
  defaultTemplateArgs,
  emptyTemplateContent,
  formatOptions,
  parseOptions,
  templateContentInput,
} from "@/lib/taskTemplates";
import {
  TaskPriority,
  TaskSize,
  TaskTemplateArgType,
  type TaskTemplate,
  type TaskTemplateArgInput,
  type TaskTemplateContentInput,
  type TaskTemplateVersion,
} from "@cherrypi/sdk";
import {
  ArchiveRestoreIcon,
  CheckIcon,
  PlusIcon,
  TagIcon,
  UserCheckIcon,
  UsersIcon,
  XIcon,
} from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  open: boolean;
  projectId: string;
  template?: TaskTemplate | null;
}>();
const emit = defineEmits<{
  "update:open": [boolean];
  saved: [TaskTemplate];
}>();

const PRIORITIES = [
  { value: TaskPriority.LOW, label: taskPriorityLabel(TaskPriority.LOW) },
  { value: TaskPriority.NORMAL, label: taskPriorityLabel(TaskPriority.NORMAL) },
  { value: TaskPriority.HIGH, label: taskPriorityLabel(TaskPriority.HIGH) },
  {
    value: TaskPriority.CRITICAL,
    label: taskPriorityLabel(TaskPriority.CRITICAL),
  },
];

const BODY_HINT =
  "Use {{ variable }} for arguments and {% if %} / {% for %} blocks. Includes and macros are not supported.";

const isEditing = computed(() => !!props.template);
const tab = ref("content");
const name = ref("");
const description = ref("");
const content = ref<TaskTemplateContentInput>(emptyTemplateContent());
const versions = ref<TaskTemplateVersion[]>([]);
const busy = ref(false);
const error = ref("");

const { active: activeTags } = useTags(() => props.projectId);
const projectTeams = ref<ProjectTeamLink[]>([]);
const leadCandidates = ref<LeadCandidate[]>([]);

const selectedTags = computed(() =>
  activeTags.value.filter((tag) =>
    (content.value.defaultTagIds ?? []).includes(tag.id),
  ),
);
const selectedTeams = computed(() =>
  projectTeams.value.filter((team) =>
    (content.value.defaultTeamIds ?? []).includes(team.team_id),
  ),
);
const selectedLeads = computed(() =>
  leadCandidates.value.filter((lead) =>
    (content.value.defaultLeadActorIds ?? []).includes(lead.actor_id),
  ),
);

const preview = ref<{
  prompt: string;
  title: string;
  priority: TaskPriority;
  size: TaskSize;
  tagIds: string[];
} | null>(null);
const previewError = ref("");
const validationErrors = ref<string[]>([]);
const validationWarnings = ref<string[]>([]);
let previewTimer: ReturnType<typeof setTimeout> | undefined;

const discardOpen = ref(false);
const savedSnapshot = ref("");

function snapshot(): string {
  return JSON.stringify({
    name: name.value,
    description: description.value,
    content: content.value,
  });
}

const dirty = computed(() => snapshot() !== savedSnapshot.value);

function requestOpen(next: boolean) {
  if (next) {
    emit("update:open", true);
    return;
  }
  requestClose();
}

function requestClose() {
  if (!dirty.value) {
    emit("update:open", false);
    return;
  }
  discardOpen.value = true;
}

function discardChanges() {
  discardOpen.value = false;
  savedSnapshot.value = snapshot();
  emit("update:open", false);
}

watch(
  () => JSON.stringify(content.value),
  () => schedulePreview(),
);

watch(
  () => [props.open, props.template],
  () => {
    if (!props.open) return;
    tab.value = "content";
    error.value = "";
    preview.value = null;
    previewError.value = "";
    validationErrors.value = [];
    validationWarnings.value = [];
    if (props.template) {
      name.value = props.template.name;
      description.value = props.template.description;
      content.value = templateContentInput(props.template.latest?.content);
      void loadVersions();
    } else {
      name.value = "";
      description.value = "";
      content.value = emptyTemplateContent();
      versions.value = [];
    }
    savedSnapshot.value = snapshot();
    discardOpen.value = false;
    void loadProjectContext();
    schedulePreview();
  },
  { immediate: true },
);

async function loadVersions() {
  if (!props.template) return;
  try {
    versions.value = await agency.listTaskTemplateVersions(props.template.id);
  } catch {
    versions.value = [];
  }
}

async function loadProjectContext() {
  try {
    const access = await projectAccessApi.access(props.projectId);
    projectTeams.value = projectTeamsWithInherited(access);
  } catch {
    projectTeams.value = [];
  }
  try {
    leadCandidates.value = await projectAccessApi.leadCandidates(
      props.projectId,
    );
  } catch {
    leadCandidates.value = [];
  }
}

function addArg() {
  content.value.args = [
    ...(content.value.args ?? []),
    { key: "", label: "", type: TaskTemplateArgType.TEXT, required: false },
  ];
}

function removeArg(index: number) {
  content.value.args = (content.value.args ?? []).filter(
    (_arg, position) => position !== index,
  );
}

function defaultPlaceholder(arg: TaskTemplateArgInput): string {
  return arg.type === TaskTemplateArgType.SELECT
    ? "Default option"
    : "Default value";
}

function setArgType(arg: TaskTemplateArgInput, type: TaskTemplateArgType) {
  arg.type = type;
  if (type !== TaskTemplateArgType.SELECT) arg.options = [];
}

function setArgTypeFromSelect(arg: TaskTemplateArgInput, value: unknown) {
  const type = ARG_TYPES.find((candidate) => candidate === value);
  if (type !== undefined) setArgType(arg, type);
}

function setArgOptions(arg: TaskTemplateArgInput, value: unknown) {
  arg.options = parseOptions(String(value ?? ""));
}

function setDefaultPriority(value: unknown) {
  const priority = PRIORITIES.find((option) => option.value === value);
  if (priority) content.value.defaultPriority = priority.value;
}

function setDefaultSize(value: unknown) {
  const size = TASK_SIZES.find((option) => option.value === value);
  if (size) content.value.defaultSize = size.value;
}

function toggleTag(id: string) {
  const current = content.value.defaultTagIds ?? [];
  content.value.defaultTagIds = current.includes(id)
    ? current.filter((value) => value !== id)
    : [...current, id];
}

function toggleTeam(id: number) {
  const current = content.value.defaultTeamIds ?? [];
  content.value.defaultTeamIds = current.includes(id)
    ? current.filter((value) => value !== id)
    : [...current, id];
}

function toggleLead(id: string) {
  const current = content.value.defaultLeadActorIds ?? [];
  content.value.defaultLeadActorIds = current.includes(id)
    ? current.filter((value) => value !== id)
    : [...current, id];
}

function schedulePreview() {
  if (!props.open) return;
  clearTimeout(previewTimer);
  previewTimer = setTimeout(() => void renderPreview(), 350);
}

async function renderPreview() {
  if (!content.value.body.trim()) {
    preview.value = null;
    previewError.value = "";
    return;
  }
  try {
    const rendered = await agency.renderTaskTemplate({
      projectId: props.projectId,
      content: content.value,
      args: defaultTemplateArgs(content.value),
    });
    preview.value = {
      prompt: rendered.prompt,
      title: rendered.title,
      priority: rendered.priority,
      size: rendered.size,
      tagIds: rendered.tagIds,
    };
    previewError.value = "";
    validationErrors.value = [];
    validationWarnings.value = [];
  } catch (err) {
    preview.value = null;
    previewError.value = err instanceof Error ? err.message : String(err);
  }
}

async function save() {
  if (!name.value.trim() || !content.value.body.trim()) return;
  busy.value = true;
  error.value = "";
  try {
    const validation = await agency.validateTaskTemplate(
      props.projectId,
      content.value,
    );
    validationErrors.value = validation.errors;
    validationWarnings.value = validation.warnings;
    if (!validation.valid) {
      tab.value = "preview";
      error.value = "Fix the validation errors before saving.";
      return;
    }
    const result = props.template
      ? await agency.updateTaskTemplate(props.template.id, {
          name: name.value.trim(),
          description: description.value.trim(),
          content: content.value,
        })
      : await agency.createTaskTemplate({
          projectId: props.projectId,
          name: name.value.trim(),
          description: description.value.trim(),
          content: content.value,
        });

    savedSnapshot.value = snapshot();
    if ("versionCreated" in result && result.versionCreated) {
      toast.success("Saved as a new version");
    } else {
      toast.success(props.template ? "Template saved" : "Template created");
    }
    emit("saved", result.template);
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

async function restoreVersion(version: number | bigint) {
  if (!props.template) return;
  busy.value = true;
  error.value = "";
  try {
    const result = await agency.restoreTaskTemplateVersion(
      props.template.id,
      Number(version),
    );
    toast.success(`Restored version ${version}`);
    emit("saved", result.template);
    name.value = result.template.name;
    description.value = result.template.description;
    content.value = templateContentInput(result.template.latest?.content);
    await loadVersions();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}
</script>
