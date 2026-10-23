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
.flex.flex-col(class="gap-0.5")
  p.text-muted-foreground.px-3.py-1.text-xs.font-medium(
    v-if="!items.length && !collapsed"
  )
    | No projects
  RouterLink.flex.min-w-0.items-center.gap-2.rounded-md.text-sm.transition-colors(
    v-for="item in items"
    :key="item.project.id"
    class="py-1.5"
    :to="{ name: 'project', params: { slug: item.project.slug } }"
    :title="collapsed ? item.project.name : undefined"
    :style="collapsed ? undefined : { paddingLeft: `${12 + item.depth * 14}px` }"
    :class="[isActive(item.project.slug) ? 'text-primary' : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground', collapsed ? 'justify-center px-0' : 'px-3']"
  )
    span.relative.shrink-0
      img.size-4.rounded-full.object-cover(
        v-if="item.project.picture"
        :src="item.project.picture.url"
        :alt="item.project.name"
      )
      span.size-4(v-else :innerHTML="logo(item.project)")
      span.border-sidebar.absolute.-right-0.-bottom-0.size-2.rounded-full.border.bg-current(
        v-if="collapsed && item.state"
        :class="taskStateTextClass(item.state)"
        :title="item.label"
      )
    span.min-w-0.truncate(:class="collapsed ? 'sr-only' : ''") {{ item.project.name }}
    span.ml-auto.flex.shrink-0.items-center(class="gap-1.5")
      span.bg-primary.text-primary-foreground.rounded-full.text-xs.font-medium(
        v-if="unreadCount(item.project.id) && !collapsed"
        class="px-1.5"
        :title="unreadCount(item.project.id) + ' unread'"
      ) {{ unreadCount(item.project.id) }}
      component.size-3(
        v-if="item.state && !collapsed"
        :is="taskStateIcon(item.state)"
        :class="[taskStateTextClass(item.state), item.iconClass]"
        :title="item.label"
      )
</template>

<script setup lang="ts">
import { useNotifications } from "@/composables/useNotifications";
import { useProjectStatuses } from "@/composables/useProjectStatuses";
import { useProjects } from "@/composables/useProjects";
import { PROJECT_STATUS_LABEL, PROJECT_STATUS_STATE } from "@/lib/project";
import { taskStateIcon, taskStateTextClass } from "@/lib/task";
import { type Project } from "@cherrypi/sdk";
import { toSvg } from "jdenticon";
import { computed, onMounted } from "vue";
import { RouterLink, useRoute } from "vue-router";

const { active, loadProjects } = useProjects();
const { statuses } = useProjectStatuses();
const { unreadByProject } = useNotifications();
const route = useRoute();

// Unread inbox entries for one project, shown as the sidebar badge.
function unreadCount(projectId: string): number {
  return unreadByProject.value.get(projectId) ?? 0;
}

// The shell collapses its own sidebar; projects mirror the rail's icon-only
// mode so labels and the status icon give way to a status dot.
withDefaults(defineProps<{ collapsed?: boolean }>(), { collapsed: false });

interface NavItem {
  project: Project;
  state:
    | (typeof PROJECT_STATUS_STATE)[keyof typeof PROJECT_STATUS_STATE]
    | undefined;
  label: string;
  iconClass: string;
  depth: number;
}

const items = computed(() => {
  const all: NavItem[] = active.value.map((project) => {
    const status = statuses.value.get(project.id);
    return {
      project,
      state: status ? PROJECT_STATUS_STATE[status] : undefined,
      label: status ? PROJECT_STATUS_LABEL[status] : "",
      // The in-flight icon breathes; reduced motion turns it off.
      iconClass:
        status === "running" ? "animate-pulse motion-reduce:animate-none" : "",
      depth: 0,
    };
  });
  const byID = new Map(all.map((item) => [item.project.id, item]));
  const children = new Map<string, NavItem[]>();
  const roots: NavItem[] = [];
  for (const item of all) {
    const parentID = item.project.parentProjectId;
    // A parent that is archived or not visible turns its child into a root, so
    // the sidebar never loses a project.
    if (parentID && byID.has(parentID)) {
      const siblings = children.get(parentID) ?? [];
      siblings.push(item);
      children.set(parentID, siblings);
    } else {
      roots.push(item);
    }
  }
  const out: NavItem[] = [];
  const seen = new Set<string>();
  const visit = (item: NavItem, depth: number) => {
    if (seen.has(item.project.id)) return;
    seen.add(item.project.id);
    out.push({ ...item, depth });
    for (const child of children.get(item.project.id) ?? []) {
      visit(child, depth + 1);
    }
  };
  for (const root of roots) visit(root, 0);
  return out;
});

onMounted(() => void loadProjects());

function isActive(slug: string): boolean {
  return route.name === "project" && route.params.slug === slug;
}

function logo(project: Project): string {
  return toSvg(project.id, 16);
}
</script>
