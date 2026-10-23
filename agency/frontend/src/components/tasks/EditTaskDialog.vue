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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent.overflow-y-auto(class="max-h-[90vh] sm:max-w-lg")
    DialogHeader
      DialogTitle Edit task
    form.flex.flex-col.gap-4(v-if="task" @submit.prevent="save")
      Field
        FieldLabel(for="edit-prompt") Prompt
        Textarea#edit-prompt.field-sizing-content.resize-y(
          v-model="prompt"
          placeholder="What should the agent do?"
        )
      Field
        FieldLabel(for="edit-title") Title
        Input#edit-title(v-model="title" placeholder="Optional title")

      .grid.gap-4(class="sm:grid-cols-2")
        Field
          FieldLabel(for="edit-assignee") Assignee
          Select(
            v-model="assigneeChoice"
            @update:model-value="onAssigneeChange"
          )
            SelectTrigger#edit-assignee.w-full
              SelectValue(placeholder="Auto-assign")
            SelectContent
              SelectGroup
                SelectItem(:value="AUTO") Auto-assign
                SelectItem(
                  v-for="item in agents"
                  :key="item.id"
                  :value="item.id"
                )
                  | {{ item.displayName || item.username }}
        Field
          FieldLabel(for="edit-project") Project
          Select(v-model="projectChoice")
            SelectTrigger#edit-project.w-full
              SelectValue(placeholder="Select project")
            SelectContent
              SelectGroup
                SelectItem(
                  v-for="project in projectOptions"
                  :key="project.id"
                  :value="project.id"
                ) {{ project.name }}
          FieldDescription Every task belongs to a project.

      Field
        FieldLabel Tags
        .flex.min-h-9.flex-wrap.items-center.rounded-md.border.px-2(
          class="gap-1.5 py-1.5"
        )
          span.text-muted-foreground.text-xs(v-if="!activeTags.length")
            | No tags yet
          button.rounded-full.border.px-2.text-xs.transition-opacity(
            v-for="tag in activeTags"
            :key="tag.id"
            type="button"
            class="py-0.5"
            :style="selectedTags.has(tag.id) ? tagStyle(tag.color) : undefined"
            :class="selectedTags.has(tag.id) ? '' : 'border-border text-muted-foreground'"
            @click="toggleTag(tag.id)"
          ) {{ tag.name }}

      .grid.gap-4(class="sm:grid-cols-2")
        Field
          FieldLabel Priority
          .bg-muted.inline-flex.items-center.gap-1.rounded-lg.border(class="p-0.5")
            button.rounded-md.px-3.py-1.text-xs.font-medium.transition-colors(
              v-for="option in PRIORITIES"
              :key="option.value"
              type="button"
              :class="priority === option.value ? 'bg-card text-foreground shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'"
              @click="priority = option.value"
            ) {{ option.label }}
        Field
          FieldLabel Size
          .bg-muted.inline-flex.flex-wrap.items-center.gap-1.rounded-lg.border(
            class="p-0.5"
          )
            button.rounded-md.px-3.py-1.text-xs.font-medium.transition-colors(
              v-for="option in SIZES"
              :key="option.value"
              type="button"
              :class="size === option.value ? 'bg-card text-foreground shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'"
              @click="size = option.value"
            ) {{ option.label }}

      p.text-destructive.text-sm(v-if="error") {{ error }}
      DialogFooter
        Button.text-destructive(
          type="button"
          variant="outline"
          @click="remove"
          :disabled="deleting"
        )
          Trash2Icon(data-icon="inline-start")
          span Delete
        Button(
          type="button"
          variant="outline"
          @click="emit('update:open', false)"
        ) Cancel
        Button(
          type="submit"
          :disabled="saving || !prompt.trim() || !projectReady"
        )
          Spinner(v-if="saving" data-icon="inline-start")
          CheckIcon(v-else data-icon="inline-start")
          span Save
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldLabel } from "@/components/ui/field";
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
import { Textarea } from "@/components/ui/textarea";
import { useAgents } from "@/composables/useAgents";
import { useProjects } from "@/composables/useProjects";
import { useTags } from "@/composables/useTags";
import { sortProjectsByName } from "@/lib/project";
import { agency } from "@/lib/sdk";
import { TASK_SIZES, tagStyle, taskName } from "@/lib/task";
import {
  TaskPriority,
  TaskSize,
  type Project,
  type Task,
  type UpdateTaskInput,
} from "@cherrypi/sdk";
import { CheckIcon, Trash2Icon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const NO_PROJECT = "__none__";
const AUTO = "__auto__";

const PRIORITIES = [
  { value: TaskPriority.LOW, label: "Low" },
  { value: TaskPriority.NORMAL, label: "Normal" },
  { value: TaskPriority.HIGH, label: "High" },
  { value: TaskPriority.CRITICAL, label: "Critical" },
];

const SIZES = TASK_SIZES;

const props = defineProps<{
  open: boolean;
  task?: Task | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  saved: [];
  deleted: [];
}>();

const { agents, loadAgents, actorIdFor, agentIdForActor } = useAgents();
const { active: allProjects, loadProjects } = useProjects();

const prompt = ref("");
const title = ref("");
const priority = ref<TaskPriority>(TaskPriority.NORMAL);
const size = ref<TaskSize>(TaskSize.MEDIUM);
const assigneeChoice = ref(AUTO);
const projectChoice = ref(NO_PROJECT);
const selectedTags = ref<Set<string>>(new Set());
const agentProjects = ref<Project[]>([]);
const saving = ref(false);
const deleting = ref(false);
const error = ref("");

const projectOptions = computed(() =>
  sortProjectsByName(
    assigneeChoice.value === AUTO ? allProjects.value : agentProjects.value,
  ),
);

// Update requires a project, so an orphan draft must pick one before saving.
const projectReady = computed(() => projectChoice.value !== NO_PROJECT);

const { active: activeTags } = useTags(() =>
  projectChoice.value === NO_PROJECT ? "" : projectChoice.value,
);

// A tag that does not belong to the picked project is dropped, so the update
// never sends a cross-project tag.
watch(activeTags, (list) => {
  const valid = new Set(list.map((tag) => tag.id));
  const next = new Set([...selectedTags.value].filter((id) => valid.has(id)));
  if (next.size !== selectedTags.value.size) selectedTags.value = next;
});

function toggleTag(id: string) {
  const next = new Set(selectedTags.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selectedTags.value = next;
}

async function loadAssignee(agentId: string) {
  agentProjects.value = [];
  if (!agentId) return;
  try {
    agentProjects.value = await agency.listAgentProjects(agentId);
  } catch {
    agentProjects.value = [];
  }
}

// Only a user picking a different agent reloads that agent's projects.
function onAssigneeChange(value: unknown) {
  void loadAssignee(value === AUTO ? "" : String(value));
}

async function hydrate() {
  const current = props.task;
  if (!current) return;
  prompt.value = current.prompt;
  title.value = current.name;
  priority.value = current.priority || TaskPriority.NORMAL;
  size.value = current.size || TaskSize.MEDIUM;
  selectedTags.value = new Set(current.tags.map((tag) => tag.id));
  const assignee = current.assigneeActorId || "";
  assigneeChoice.value = assignee ? agentIdForActor(assignee) : AUTO;
  await loadAssignee(assigneeChoice.value);
  projectChoice.value = current.projectId || NO_PROJECT;
  error.value = "";
}

watch(
  () => props.open,
  (open) => {
    if (open) void hydrate();
  },
);

async function save() {
  const current = props.task;
  if (!current || !prompt.value.trim() || !projectReady.value) return;
  saving.value = true;
  error.value = "";
  try {
    const input: UpdateTaskInput = {
      prompt: prompt.value.trim(),
      name: title.value.trim(),
      // These options are no longer editable; preserve the task's values.
      autoTitle: current.autoTitle,
      interactive: current.interactive,
      workspace: current.workspace,
      priority: priority.value,
      size: size.value,
      projectId: projectChoice.value === NO_PROJECT ? "" : projectChoice.value,
      assigneeActorId:
        assigneeChoice.value === AUTO ? "" : actorIdFor(assigneeChoice.value),
      tagIds: [...selectedTags.value],
    };

    await agency.updateTask(current.id, input);
    toast.success("Task updated");
    emit("saved");
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function remove() {
  const current = props.task;
  if (!current) return;
  const label = taskName(current);
  if (!window.confirm(`Delete "${label}"?`)) return;
  deleting.value = true;
  error.value = "";
  try {
    await agency.deleteTask(current.id);
    toast.success("Task deleted");
    emit("deleted");
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    deleting.value = false;
  }
}

onMounted(() => {
  void Promise.all([loadAgents(), loadProjects()]);
});
</script>
