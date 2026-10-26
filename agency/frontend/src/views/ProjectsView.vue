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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  Card
    CardHeader
      CardTitle Active projects
      CardDescription
        | Archiving a project hides it and its tasks; un-archiving shows them
        | again.
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Table
        TableHeader
          TableRow
            TableHead Project
            TableHead Slug
            TableHead Tasks
            TableHead Created
            TableHead
        TableBody
          TableRowsSkeleton(
            v-if="loading && !active.length"
            :widths="['w-40', 'w-24', 'w-16', 'w-28', 'w-24']"
          )
          TableRow(v-for="project in active" :key="project.id")
            TableCell
              RouterLink.flex.items-center.gap-2.font-medium(
                :to="{ name: 'project', params: { slug: project.slug } }"
              )
                span.size-6.shrink-0(:innerHTML="logo(project)")
                span {{ project.name }}
                span.text-muted-foreground.truncate.text-xs(
                  v-if="project.description"
                )
                  | {{ project.description }}
            TableCell.text-muted-foreground.font-mono.text-xs {{ project.slug }}
            TableCell.text-muted-foreground {{ taskCount(project.id) }}
            TableCell.text-muted-foreground {{ formatTimestamp(project.createdAt) }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="archive(project)")
                  ArchiveIcon(data-icon="inline-start")
                  span Archive
          TableRow(v-if="!active.length && !loading")
            TableCell.text-muted-foreground(:colspan="5") No active projects.

  Card
    CardHeader
      button.flex.w-full.cursor-pointer.items-center.gap-2(
        type="button"
        @click="showArchived = !showArchived"
      )
        ChevronDownIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
          :class="showArchived ? '' : '-rotate-90'"
        )
        CardTitle Archived projects
        Badge(variant="secondary") {{ archived.length }}
    CardContent(v-if="showArchived")
      Table
        TableHeader
          TableRow
            TableHead Project
            TableHead Slug
            TableHead Archived
            TableHead
        TableBody
          TableRow(v-for="project in archived" :key="project.id")
            TableCell
              RouterLink.flex.items-center.gap-2.font-medium(
                :to="{ name: 'project', params: { slug: project.slug } }"
              )
                span.size-6.shrink-0(:innerHTML="logo(project)")
                span {{ project.name }}
            TableCell.text-muted-foreground.font-mono.text-xs {{ project.slug }}
            TableCell.text-muted-foreground {{ formatTimestamp(project.archivedAt) }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="unarchive(project)")
                  ArchiveRestoreIcon(data-icon="inline-start")
                  span Unarchive
          TableRow(v-if="!archived.length")
            TableCell.text-muted-foreground(:colspan="4") No archived projects.

  Card
    CardHeader
      CardTitle New project
      CardDescription
        | Projects group tasks across every agent. The slug is set once and
        | used as the project's URL.
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="create")
        FieldGroup
          Field
            FieldLabel(for="project-name") Name
            Input#project-name(v-model="name" placeholder="Website")
          Field
            FieldLabel(for="project-slug") Slug
            Input#project-slug(
              v-model="slug"
              :placeholder="slugPlaceholder"
              autocomplete="off"
            )
            FieldDescription Lowercase kebab-case, unique, and immutable.
          Field
            FieldLabel(for="project-prefix") Task prefix
            Input#project-prefix(
              v-model="prefix"
              :placeholder="prefixPlaceholder"
              autocapitalize="characters"
              autocomplete="off"
            )
            FieldDescription
              | Uppercase letters, digits, and dashes prefixed to task ids;
              | unique. Derived from the slug when blank.
          Field
            FieldLabel(for="project-description") Description
            Textarea#project-description(v-model="description")
        p.text-destructive.text-sm(v-if="createError") {{ createError }}
        .flex.justify-end
          Button(type="submit" :disabled="creating || !name.trim()")
            Spinner(v-if="creating" data-icon="inline-start")
            PlusIcon(v-else data-icon="inline-start")
            span Create
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { formatTimestamp } from "@/lib/format";
import { normalizeTaskPrefix, slugify } from "@/lib/project";
import { agency } from "@/lib/sdk";
import { type Project } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  ChevronDownIcon,
  PlusIcon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { toast } from "vue3-toastify";

const {
  active,
  archived,
  loading,
  error,
  loadProjects,
  createProject,
  archiveProject,
  unarchiveProject,
} = useProjects();

const name = ref("");
const slug = ref("");
const prefix = ref("");
const description = ref("");
const creating = ref(false);
const createError = ref("");
const showArchived = ref(false);

const slugPlaceholder = computed(() => slugify(name.value) || "project");
const prefixPlaceholder = computed(
  () =>
    normalizeTaskPrefix(slug.value.trim() ? slug.value : name.value) ||
    "PROJECT",
);

const taskCounts = ref(new Map<string, number>());

async function loadCounts() {
  try {
    const tasks = await agency.listTasks();
    const counts = new Map<string, number>();
    for (const task of tasks) {
      if (!task.projectId) continue;
      counts.set(task.projectId, (counts.get(task.projectId) ?? 0) + 1);
    }
    taskCounts.value = counts;
  } catch {
    taskCounts.value = new Map();
  }
}

function taskCount(projectId: string): number {
  return taskCounts.value.get(projectId) ?? 0;
}

function logo(project: Project): string {
  return toSvg(project.id, 24);
}

async function create() {
  creating.value = true;
  createError.value = "";
  try {
    await withProgress(() =>
      createProject({
        name: name.value.trim(),
        slug: slug.value.trim(),
        prefix: prefix.value.trim(),
        description: description.value.trim(),
      }),
    );
    name.value = "";
    slug.value = "";
    prefix.value = "";
    description.value = "";
    toast.success("Project created");
    await loadCounts();
  } catch (err) {
    createError.value = err instanceof Error ? err.message : String(err);
  } finally {
    creating.value = false;
  }
}

async function archive(project: Project) {
  try {
    const offline = await withProgress(() => archiveProject(project.id));
    if (offline.length) {
      toast.warning(
        `Project archived, but ${offline.length} offline agent(s) were skipped`,
      );
    } else {
      toast.success("Project archived");
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function unarchive(project: Project) {
  try {
    await withProgress(() => unarchiveProject(project.id));
    toast.success("Project unarchived");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

onMounted(async () => {
  await Promise.all([loadProjects(true), loadCounts()]);
});
</script>
