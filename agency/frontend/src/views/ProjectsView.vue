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
.mx-auto.flex.w-full.max-w-5xl.flex-col.gap-3.p-3
  section.d-card.bg-base-100.border
    .d-card-body
      .d-card-title.flex.items-center.gap-2
        FolderIcon
        h2 Active projects
        span.d-badge.d-badge-sm.d-badge-soft {{ active.length }}

      .d-alert.d-alert-error.d-alert-soft.text-sm(v-if="error" role="alert")
        | {{ error }}

      .overflow-x-auto
        table.d-table
          thead
            tr
              th Project
              th Slug
              th Created
              th.text-right Actions

          tbody
            template(v-if="loading && !active.length")
              tr(v-for="row in 3" :key="row")
                td(v-for="(width, index) in skeletonWidths" :key="index")
                  .d-skeleton.h-4(:class="width")

            tr(v-for="project in active" :key="project.id")
              td
                RouterLink.d-link-hover.flex.items-center.gap-2(
                  :to="{ name: 'project', params: { slug: project.slug } }"
                )
                  span.size-6.shrink-0(:innerHTML="logo(project)")
                  span.font-semibold {{ project.name }}

              td.text-muted-foreground.font-mono.text-xs @{{ project.slug }}
              td.text-muted-foreground {{ formatTimestamp(project.createdAt) }}

              td
                .flex.justify-end
                  button.d-btn.d-btn-outline.d-btn-sm(
                    type="button"
                    @click="archive(project)"
                  )
                    ArchiveIcon(:size="16")
                    span Archive

            tr(v-if="!active.length && !loading")
              td.text-muted-foreground(:colspan="4") No active projects.

  section.d-card.bg-base-100.border
    .d-card-body(class="@container")
      .d-card-title.flex.items-center.gap-2
        FolderPlusIcon
        h2 New project

      form.grid.gap-3(class="@3xl:grid-cols-2" @submit.prevent="create")
        fieldset.d-fieldset
          label.d-fieldset-legend(for="project-name") Name
          input#project-name.d-input.w-full(
            v-model="name"
            placeholder="Website"
          )

        fieldset.d-fieldset
          label.d-fieldset-legend(for="project-slug") Slug
          input#project-slug.d-input.w-full(
            v-model="slug"
            :placeholder="slugPlaceholder"
            autocomplete="off"
          )
          p.d-fieldset-label
            | Lowercase kebab-case, unique, and immutable.

        fieldset.d-fieldset
          label.d-fieldset-legend(for="project-prefix") Task prefix
          input#project-prefix.d-input.w-full(
            v-model="prefix"
            :placeholder="prefixPlaceholder"
            autocapitalize="characters"
            autocomplete="off"
          )
          p.d-fieldset-label
            | Uppercase letters, digits, and dashes prefixed to task ids;
            | unique. Derived from the slug when blank.

        fieldset.d-fieldset
          label.d-fieldset-legend(for="project-description")
            | Description
          textarea#project-description.d-textarea.w-full(v-model="description")

        .d-alert.d-alert-error.d-alert-soft.col-span-full(
          v-if="createError"
          role="alert"
        )
          | {{ createError }}

        .col-span-full.flex.justify-end
          button.d-btn.d-btn-primary(
            type="submit"
            :disabled="creating || !name.trim()"
          )
            span.d-loading.d-loading-spinner.d-loading-xs(v-if="creating")
            SparklesIcon(v-else :size="16")
            span Create

  section.d-collapse.d-collapse-arrow.bg-base-100.border
    input(type="checkbox" aria-label="Show archived projects")

    .d-collapse-title.flex.items-center.gap-2
      ArchiveIcon
      h2 Archived projects
      span.d-badge.d-badge-sm.d-badge-soft {{ archived.length }}

    .d-collapse-content.p-0
      .flex.flex-col.gap-2.border-t.p-3
        .d-alert(role="alert")
          InfoIcon(:size="16")
          p
            | Archiving a project hides it and its tasks; un-archiving shows them
            | again.

        .overflow-x-auto
          table.d-table
            thead
              tr
                th Project
                th Slug
                th Archived
                th.text-right Actions

            tbody
              tr(v-for="project in archived" :key="project.id")
                td
                  RouterLink.d-link-hover.flex.items-center.gap-2(
                    :to="{ name: 'project', params: { slug: project.slug } }"
                  )
                    span.size-6.shrink-0(:innerHTML="logo(project)")
                    span.font-semibold {{ project.name }}

                td.text-muted-foreground.font-mono.text-xs @{{ project.slug }}
                td.text-muted-foreground
                  | {{ formatTimestamp(project.archivedAt) }}

                td
                  .flex.justify-end
                    button.d-btn.d-btn-outline.d-btn-sm(
                      type="button"
                      @click="unarchive(project)"
                    )
                      ArchiveRestoreIcon(:size="16")
                      span Unarchive

              tr(v-if="!archived.length")
                td.text-muted-foreground(:colspan="4") No archived projects.
</template>

<script setup lang="ts">
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { formatTimestamp } from "@/lib/format";
import { normalizeTaskPrefix, slugify } from "@/lib/project";
import { type Project } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  FolderIcon,
  FolderPlusIcon,
  InfoIcon,
  SparklesIcon,
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

const skeletonWidths = ["w-40", "w-24", "w-28", "w-16"];

const slugPlaceholder = computed(() => slugify(name.value) || "project");
const prefixPlaceholder = computed(
  () =>
    normalizeTaskPrefix(slug.value.trim() ? slug.value : name.value) ||
    "PROJECT",
);

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

onMounted(() => loadProjects(true));
</script>
