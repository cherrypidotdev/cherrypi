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
        | {{ $t("templateEditor.description") }}
    Tabs(v-model="tab")
      TabsList
        TabsTrigger(value="content") {{ $t("templateEditor.tab.content") }}
        TabsTrigger(value="defaults") {{ $t("templateEditor.tab.defaults") }}
        TabsTrigger(value="preview") {{ $t("settings.sounds.preview") }}
        TabsTrigger(v-if="isEditing" value="versions") {{ $t("templateEditor.tab.versions") }}
      TabsContent.flex.flex-col.gap-4(value="content")
        FieldGroup
          Field
            FieldLabel(for="template-name") {{ $t("common.name") }}
            Input#template-name(
              v-model="name"
              :placeholder="$t('templateEditor.namePlaceholder')"
            )
          Field
            FieldLabel(for="template-description") {{ $t("common.description") }}
            Input#template-description(
              v-model="description"
              :placeholder="$t('templateEditor.summaryPlaceholder')"
            )
          Field
            FieldLabel(for="template-body") {{ $t("templateEditor.body") }}
            Textarea#template-body.font-mono(
              v-model="content.body"
              :placeholder="SAMPLE_BODY"
              rows="10"
            )
            FieldDescription {{ BODY_HINT }}
          Field
            FieldLabel(for="template-title") {{ $t("templateEditor.titleTemplate") }}
            Input#template-title.font-mono(
              v-model="content.titleTemplate"
              :placeholder="SAMPLE_TITLE"
            )
            FieldDescription
              | {{ $t("templateEditor.titleHint") }}

        .flex.flex-col.gap-2
          .flex.items-center.justify-between.gap-2
            .flex.flex-col
              span.text-sm.font-medium {{ $t("templateEditor.arguments") }}
              span.text-muted-foreground.text-xs
                | {{ $t("templateEditor.argumentsHint") }}
            Button(type="button" variant="outline" size="sm" @click="addArg")
              PlusIcon(data-icon="inline-start")
              span {{ $t("templateEditor.addArgument") }}
          p.text-muted-foreground.text-xs(v-if="!content.args?.length")
            | {{ $t("templateEditor.noArguments") }}
          .flex.flex-col.gap-2(
            v-for="(arg, index) in content.args"
            :key="index"
          )
            .grid.grid-cols-1.gap-2(
              class="sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_10rem_auto]"
            )
              Input.font-mono(
                v-model="arg.key"
                :placeholder="$t('templateEditor.keyPlaceholder')"
                :aria-label="$t('templateEditor.keyAria')"
              )
              Input(
                v-model="arg.label"
                :placeholder="$t('templateEditor.labelPlaceholder')"
                :aria-label="$t('templateEditor.labelAria')"
              )
              Select(
                :model-value="arg.type ?? TaskTemplateArgType.TEXT"
                @update:model-value="setArgTypeFromSelect(arg, $event)"
              )
                SelectTrigger.w-full(
                  :aria-label="$t('templateEditor.typeAria')"
                )
                  SelectValue
                SelectContent
                  SelectItem(
                    v-for="type in ARG_TYPES"
                    :key="type"
                    :value="type"
                  ) {{ $t(ARG_TYPE_KEYS[type]) }}
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                :title="$t('templateEditor.removeArgument')"
                :aria-label="$t('templateEditor.removeArgument')"
                @click="removeArg(index)"
              )
                XIcon.size-3
            .grid.grid-cols-1.gap-2(
              class="sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
            )
              Input(
                v-model="arg.defaultValue"
                :placeholder="defaultPlaceholder(arg)"
                :aria-label="$t('templateEditor.defaultAria')"
              )
              Input(
                v-if="arg.type === TaskTemplateArgType.SELECT"
                :placeholder="$t('templateEditor.optionsPlaceholder')"
                :aria-label="$t('templateEditor.optionsAria')"
                :model-value="formatOptions(arg.options)"
                @update:model-value="setArgOptions(arg, $event)"
              )
              Input(
                v-else
                v-model="arg.description"
                :placeholder="$t('templateEditor.descriptionPlaceholder')"
                :aria-label="$t('templateEditor.descriptionAria')"
              )
              label.flex.items-center.gap-2.text-xs.whitespace-nowrap
                Switch(
                  size="sm"
                  :model-value="arg.required ?? false"
                  @update:model-value="arg.required = $event"
                )
                span {{ $t("tags.required") }}
            p.text-destructive.text-xs(
              v-if="arg.type === TaskTemplateArgType.SELECT && !arg.options?.length"
            )
              | {{ $t("templateEditor.selectNeedsOption") }}

      TabsContent.flex.flex-col.gap-4(value="defaults")
        p.text-muted-foreground.text-xs
          | {{ $t("templateEditor.defaultsHint") }}
        .grid.grid-cols-2.gap-3
          Field
            FieldLabel {{ $t("domainLog.kind.priority") }}
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
            FieldLabel {{ $t("domainLog.kind.size") }}
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
                ) {{ $t(option.labelKey) }}
        .flex.flex-col.gap-2
          span.text-sm.font-medium {{ $t("domainLog.kind.tags") }}
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
                :title="$t('taskDetails.removeTag')"
                @click="toggleTag(tag.id)"
              )
                XIcon.size-3
            span.text-muted-foreground.text-xs(v-if="!selectedTags.length")
              | {{ $t("projectSettings.hierarchy.none") }}
            DropdownMenu
              DropdownMenuTrigger(as-child)
                Button(type="button" variant="outline" size="sm")
                  TagIcon(data-icon="inline-start")
                  span {{ $t("tags.add") }}
              DropdownMenuContent.w-48(align="start")
                DropdownMenuLabel {{ $t("domainLog.kind.tags") }}
                DropdownMenuSeparator
                p.text-muted-foreground.px-2.py-1.text-xs(
                  v-if="!activeTags.length"
                )
                  | {{ $t("tasks.noTagsInProject") }}
                DropdownMenuCheckboxItem(
                  v-for="tag in activeTags"
                  :key="tag.id"
                  :model-value="(content.defaultTagIds ?? []).includes(tag.id)"
                  @update:model-value="toggleTag(tag.id)"
                ) {{ tag.name }}
        .flex.flex-col.gap-2
          span.text-sm.font-medium {{ $t("domainLog.kind.teams") }}
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
                :title="$t('express.removeTeam')"
                @click="toggleTeam(team.team_id)"
              )
                XIcon.size-3
            span.text-muted-foreground.text-xs(v-if="!selectedTeams.length")
              | {{ $t("templateEditor.defaultTeams.none") }}
            DropdownMenu
              DropdownMenuTrigger(as-child)
                Button(type="button" variant="outline" size="sm")
                  UsersIcon(data-icon="inline-start")
                  span {{ $t("templateEditor.addTeam") }}
              DropdownMenuContent.w-56(align="start")
                DropdownMenuLabel {{ $t("domainLog.kind.teams") }}
                DropdownMenuSeparator
                p.text-muted-foreground.px-2.py-1.text-xs(
                  v-if="!projectTeams.length"
                )
                  | {{ $t("express.noTeams") }}
                DropdownMenuCheckboxItem(
                  v-for="team in projectTeams"
                  :key="team.team_id"
                  :model-value="(content.defaultTeamIds ?? []).includes(team.team_id)"
                  @update:model-value="toggleTeam(team.team_id)"
                ) {{ team.team_name }}
        .flex.flex-col.gap-2
          span.text-sm.font-medium {{ $t("domainLog.kind.leads") }}
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
                :title="$t('express.removeLead')"
                @click="toggleLead(lead.actor_id)"
              )
                XIcon.size-3
            span.text-muted-foreground.text-xs(v-if="!selectedLeads.length")
              | {{ $t("projectSettings.hierarchy.none") }}
            DropdownMenu
              DropdownMenuTrigger(as-child)
                Button(type="button" variant="outline" size="sm")
                  UserCheckIcon(data-icon="inline-start")
                  span {{ $t("templateEditor.addLead") }}
              DropdownMenuContent.w-56(align="start")
                DropdownMenuLabel {{ $t("domainLog.kind.leads") }}
                DropdownMenuSeparator
                p.text-muted-foreground.px-2.py-1.text-xs(
                  v-if="!leadCandidates.length"
                )
                  | {{ $t("express.noMembers") }}
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
          | {{ $t("templateEditor.previewHint") }}
        .flex.flex-col.gap-1(v-if="previewError")
          p.text-destructive.text-sm {{ previewError }}
        template(v-else-if="preview")
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs {{ $t("common.title") }}
            span.text-sm.font-medium {{ preview.title || "— (auto-title)" }}
          .flex.flex-col(class="gap-0.5")
            span.text-muted-foreground.text-xs {{ $t("domainLog.kind.prompt") }}
            MarkdownText.prose.prose-sm.max-w-none(
              class="prose-p:my-1 prose-pre:my-2"
              :text="preview.prompt"
            )
          .flex.flex-wrap.items-center.gap-2.text-xs
            Badge(variant="secondary") {{ $t(taskPriorityKey(preview.priority)) }}
            Badge(variant="secondary") {{ $t(taskSizeKey(preview.size)) }}
            Badge(v-if="preview.tagIds.length" variant="secondary")
              | {{ $t("templateEditor.tags", { count: preview.tagIds.length }) }}
        p.text-muted-foreground.text-xs(v-else)
          | {{ $t("templateApply.rendering") }}
        .flex.flex-col.gap-1(v-if="validationErrors.length")
          span.text-destructive.text-xs.font-medium {{ $t("templateEditor.validation") }}
          p.text-destructive.text-xs(
            v-for="error in validationErrors"
            :key="error"
          )
            | {{ error }}
        .flex.flex-col.gap-1(v-if="validationWarnings.length")
          span.text-xs.font-medium {{ $t("templateEditor.warnings") }}
          p.text-muted-foreground.text-xs(
            v-for="warning in validationWarnings"
            :key="warning"
          )
            | {{ warning }}

      TabsContent.flex.flex-col.gap-3(v-if="isEditing" value="versions")
        p.text-muted-foreground.text-xs
          | {{ $t("templateEditor.versionsHint") }}
        p.text-muted-foreground.text-xs(v-if="!versions.length") {{ $t("templateEditor.noVersions") }}
        .flex.items-center.justify-between.gap-2.rounded-md.border.px-3.py-2(
          v-for="version in versions"
          :key="String(version.version)"
        )
          .flex.flex-col
            span.text-sm.font-medium {{ $t("templateEditor.version", { version: version.version }) }}
            span.text-muted-foreground.text-xs
              | {{ formatTimestamp(version.createdAt) }}
              | {{ $t("templateEditor.argumentsCount", { count: version.content?.args?.length ?? 0 }) }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="restoreVersion(version.version)"
            :disabled="busy || Number(version.version) === Number(template?.latestVersion ?? 0)"
          )
            ArchiveRestoreIcon(data-icon="inline-start")
            span {{ $t("tags.restore") }}
    p.text-destructive.text-sm(v-if="error") {{ error }}
    DialogFooter
      Button(type="button" variant="outline" @click="requestClose")
        | {{ $t("common.cancel") }}
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
      DialogTitle {{ $t("templateEditor.discardQuestion") }}
      DialogDescription
        | {{ $t("templateEditor.discardHint") }}
    DialogFooter
      Button(type="button" variant="outline" @click="discardOpen = false")
        | {{ $t("templateEditor.keepEditing") }}
      Button(type="button" variant="destructive" @click="discardChanges")
        | {{ $t("templateEditor.discard") }}
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
import { TASK_SIZES, tagStyle, taskPriorityKey, taskSizeKey } from "@/lib/task";
import {
  ARG_TYPES,
  ARG_TYPE_KEYS,
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
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

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
  { value: TaskPriority.LOW, label: t(taskPriorityKey(TaskPriority.LOW)) },
  {
    value: TaskPriority.NORMAL,
    label: t(taskPriorityKey(TaskPriority.NORMAL)),
  },
  { value: TaskPriority.HIGH, label: t(taskPriorityKey(TaskPriority.HIGH)) },
  {
    value: TaskPriority.CRITICAL,
    label: t(taskPriorityKey(TaskPriority.CRITICAL)),
  },
];

const SAMPLE_BODY = "Fix the {{ component }} crash.\n\nSteps:\n{{ steps }}";
const SAMPLE_TITLE = "Fix {{ component }}";

const BODY_HINT = computed(() => t("templateEditor.bodyHint"));

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
    ? t("templateEditor.defaultOption")
    : t("templateEditor.defaultValue");
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
      error.value = t("templateEditor.errors.validation");
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
      toast.success(t("templateEditor.toasts.versionCreated"));
    } else {
      toast.success(
        props.template
          ? t("templateEditor.toasts.saved")
          : t("templateEditor.toasts.created"),
      );
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

<i18n lang="yaml">
en:
  templateEditor:
    noVersions: "No versions yet."
    description: "A Jinja prompt with declared arguments and task defaults. Saving a changed template appends an immutable version; tasks pin the version they were created from."
    tab:
      content: "Content"
      defaults: "Defaults"
      versions: "Versions"
    body: "Prompt body (Jinja)"
    bodyHint: "Use {'{'}variable{'}'} for arguments and {'{'}% if %{'}'} / {'{'}% for %{'}'} blocks. Includes and macros are not supported."
    titleTemplate: "Title template (optional)"
    titleHint: "Rendered into the task's title. Empty leaves the title to the agent's auto-title."
    arguments: "Arguments"
    argumentsHint: "Every variable the body or title references must be declared here."
    addArgument: "Add argument"
    noArguments: "No arguments yet."
    selectNeedsOption: "A select needs at least one option."
    defaultsHint: "Pre-filled in the composer. Explicit task fields override them, and a template never pins the assignee or the model."
    defaultTeams:
      none: "None; the project default applies."
    addTeam: "Add team"
    addLead: "Add lead"
    previewHint: "Rendered by the agency with every argument's default value, exactly as a task created from this template would be."
    validation: "Validation"
    warnings: "Warnings"
    versionsHint: "Versions are immutable. Tasks keep the version and arguments they were created with, so editing only affects new tasks."
    discardQuestion: "Discard unsaved changes?"
    discardHint: "This template has edits that were never saved. Discarding them cannot be undone."
    keepEditing: "Keep editing"
    discard: "Discard"
    namePlaceholder: "Bug report"
    summaryPlaceholder: "What this template is for"
    keyPlaceholder: "key"
    keyAria: "Argument key"
    labelPlaceholder: "Label"
    labelAria: "Argument label"
    typeAria: "Argument type"
    removeArgument: "Remove argument"
    defaultAria: "Argument default"
    optionsPlaceholder: "Options, comma-separated"
    optionsAria: "Argument options"
    descriptionPlaceholder: "Description (optional)"
    descriptionAria: "Argument description"
    toasts:
      created: "Template created"
      saved: "Template saved"
      versionCreated: "Saved as a new version"
    errors:
      validation: "Fix the validation errors before saving."
    defaultValue: "Default value"
    defaultOption: "Default option"
    tags: "{count} tag | {count} tags"
    version: "Version {version}"
    argumentsCount: "· {count} argument | · {count} arguments"
ru:
  templateEditor:
    noVersions: "Версий пока нет."
    description: "Jinja-промпт с объявленными аргументами и значениями по умолчанию. При сохранении изменённого шаблона добавляется неизменяемая версия; задачи фиксируют версию, из которой созданы."
    tab:
      content: "Содержимое"
      defaults: "Значения по умолчанию"
      versions: "Версии"
    body: "Тело промпта (Jinja)"
    bodyHint: "Используйте {'{'}variable{'}'} для аргументов и блоки {'{'}% if %{'}'} / {'{'}% for %{'}'}. Include и макросы не поддерживаются."
    titleTemplate: "Шаблон названия (необязательно)"
    titleHint: "Подставляется в название задачи. Если пусто, название создаёт агент автоматически."
    arguments: "Аргументы"
    argumentsHint: "Каждая переменная, используемая в теле или названии, должна быть объявлена здесь."
    addArgument: "Добавить аргумент"
    noArguments: "Аргументов пока нет."
    selectNeedsOption: "Для списка нужен хотя бы один вариант."
    defaultsHint: "Подставляются в композер. Явно заданные поля задачи их переопределяют, а шаблон никогда не фиксирует исполнителя и модель."
    defaultTeams:
      none: "Нет; действует значение проекта по умолчанию."
    addTeam: "Добавить команду"
    addLead: "Добавить руководителя"
    previewHint: "Агентство формирует его со значениями всех аргументов по умолчанию — так же, как для созданной из шаблона задачи."
    validation: "Проверка"
    warnings: "Предупреждения"
    versionsHint: "Версии неизменяемы. Задачи сохраняют версию и аргументы, с которыми созданы, поэтому правки влияют только на новые задачи."
    discardQuestion: "Отменить несохранённые изменения?"
    discardHint: "В шаблоне есть несохранённые правки. Отмена необратима."
    keepEditing: "Продолжить редактирование"
    discard: "Отменить"
    namePlaceholder: "Отчёт об ошибке"
    summaryPlaceholder: "Для чего этот шаблон"
    keyPlaceholder: "ключ"
    keyAria: "Ключ аргумента"
    labelPlaceholder: "Подпись"
    labelAria: "Подпись аргумента"
    typeAria: "Тип аргумента"
    removeArgument: "Удалить аргумент"
    defaultAria: "Значение аргумента по умолчанию"
    optionsPlaceholder: "Варианты через запятую"
    optionsAria: "Варианты аргумента"
    descriptionPlaceholder: "Описание (необязательно)"
    descriptionAria: "Описание аргумента"
    toasts:
      created: "Шаблон создан"
      saved: "Шаблон сохранён"
      versionCreated: "Сохранено как новая версия"
    errors:
      validation: "Исправьте ошибки проверки перед сохранением."
    defaultValue: "Значение по умолчанию"
    defaultOption: "Вариант по умолчанию"
    tags: "{count} тег | {count} тега | {count} тегов"
    version: "Версия {version}"
    argumentsCount: "· {count} аргумент | · {count} аргумента | · {count} аргументов"
</i18n>
