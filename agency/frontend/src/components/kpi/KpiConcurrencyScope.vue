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
slot(:data="data")
</template>

<script setup lang="ts">
import {
  useAgentConcurrency,
  useProjectConcurrency,
} from "@/composables/useConcurrency";
import { useProjects } from "@/composables/useProjects";
import {
  agentConcurrencyKpis,
  emptyConcurrencyKpis,
  projectConcurrencyKpis,
} from "@/lib/concurrencyKpis";
import type { KpiConcurrencyData } from "@/lib/kpiWidgets";
import { computed, onMounted } from "vue";

const props = defineProps<{
  boardKind: "agent" | "project";
  boardId: string;
}>();

const { projects, loadProjects } = useProjects();

const project = computed(() =>
  props.boardKind === "project"
    ? projects.value.find((item) => item.id === props.boardId)
    : undefined,
);

const agentScope = useAgentConcurrency(() =>
  props.boardKind === "agent" ? props.boardId : "",
);
const projectScope = useProjectConcurrency(() =>
  props.boardKind === "project" ? project.value : undefined,
);

const board = computed(() =>
  props.boardKind === "agent"
    ? agentScope.board.value
    : projectScope.board.value,
);
const loading = computed(() =>
  props.boardKind === "agent"
    ? agentScope.loading.value
    : projectScope.loading.value,
);
const error = computed(() =>
  props.boardKind === "agent"
    ? agentScope.error.value
    : projectScope.error.value,
);

const data = computed<KpiConcurrencyData>(() => {
  const value = board.value;
  const kpis = !value
    ? emptyConcurrencyKpis()
    : value.kind === "agent"
      ? agentConcurrencyKpis(value)
      : projectConcurrencyKpis(value);
  return { ...kpis, loading: loading.value, error: error.value };
});

onMounted(() => {
  if (props.boardKind === "project") void loadProjects();
});
</script>
