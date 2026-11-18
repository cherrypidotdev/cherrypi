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
.flex.flex-col.gap-2(data-testid="task-prompt-list")
  p.text-muted-foreground.text-xs(v-if="!groups.length")
    | {{ $t("taskSystem.noPrompts") }}
  .flex.flex-col.gap-1(v-for="group in groups" :key="group.key")
    .flex.items-center.justify-between.gap-2
      span.text-muted-foreground.text-xs.font-medium {{ group.label }}
      span.text-muted-foreground.text-xs.tabular-nums {{ group.sections.length }}
    Disclosure.overflow-hidden.rounded-md.border(
      v-for="section in group.sections"
      :key="sectionKey(section)"
      as="div"
      v-slot="{ open: sectionOpen }"
    )
      DisclosureButton.flex.w-full.items-center.gap-1.px-2.text-left(
        class="hover:bg-muted/60 py-1.5"
      )
        ChevronRightIcon.size-3.shrink-0.transition-transform(
          :class="sectionOpen ? 'rotate-90' : ''"
        )
        span.min-w-0.truncate.text-xs {{ sectionLabel(section) }}
      DisclosurePanel.border-t.px-2.py-2
        pre.text-muted-foreground.font-sans.text-xs.break-words.whitespace-pre-wrap {{ section.content }}
</template>

<script setup lang="ts">
import { PromptScope, type SystemPromptSection } from "@cherrypi/sdk";
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import { ChevronRightIcon } from "@lucide/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{ sections: SystemPromptSection[] }>();

// i18n:dynamic — the scope heading is picked from this map by the section's

const PROMPT_SCOPE_KEYS: Record<number, string> = {
  [PromptScope.ORG]: "taskPromptScopes.org",
  [PromptScope.AGENT]: "taskPromptScopes.agent",
  [PromptScope.PROJECT]: "taskPromptScopes.project",
  [PromptScope.AGENT_PROJECT]: "taskPromptScopes.agentProject",
  [PromptScope.TEAM]: "taskPromptScopes.team",
  [PromptScope.TEAM_PROJECT]: "taskPromptScopes.teamProject",
  [PromptScope.GROUP]: "taskPromptScopes.group",
  [PromptScope.GROUP_PROJECT]: "taskPromptScopes.groupProject",
  [PromptScope.GROUP_TEAM]: "taskPromptScopes.groupTeam",
  [PromptScope.BUILTIN]: "taskPromptScopes.builtin",
};

function scopeLabel(section: SystemPromptSection): string {
  const key = PROMPT_SCOPE_KEYS[section.scope];
  return key ? t(key) : t("taskPromptScopes.other");
}

function sectionKey(section: SystemPromptSection): string {
  return `${section.scope}:${section.groupId ?? ""}:${section.teamId ?? ""}:${section.owner}`;
}

function sectionLabel(section: SystemPromptSection): string {
  const owner = section.owner || scopeLabel(section);
  const version =
    section.version > 0
      ? t("taskPromptScopes.version", { version: section.version })
      : t("taskPromptScopes.unset");
  const pin = section.preview ? ` · ${t("taskPromptScopes.latest")}` : "";
  return `${owner} · ${version}${pin}`;
}

interface PromptGroup {
  key: string;
  label: string;
  sections: SystemPromptSection[];
}

const groups = computed<PromptGroup[]>(() => {
  const out: PromptGroup[] = [];
  for (const section of props.sections) {
    const key = String(section.scope);
    let group = out.find((item) => item.key === key);
    if (!group) {
      group = { key, label: scopeLabel(section), sections: [] };
      out.push(group);
    }
    group.sections.push(section);
  }
  return out;
});
</script>

<i18n lang="yaml">
en:
  taskSystem:
    noPrompts: "No prompts apply to this task."
  taskPromptScopes:
    org: "Organization"
    agent: "Agent"
    project: "Project"
    agentProject: "Agent × Project"
    team: "Team"
    teamProject: "Team × Project"
    group: "Group"
    groupProject: "Group × Project"
    groupTeam: "Group × Team"
    builtin: "Built-in"
    other: "Prompt"
    version: "v{version}"
    unset: "unset"
    latest: "latest, pins on start"
ru:
  taskSystem:
    noPrompts: "К задаче не применяется ни одного промпта."
  taskPromptScopes:
    org: "Организация"
    agent: "Агент"
    project: "Проект"
    agentProject: "Агент × Проект"
    team: "Команда"
    teamProject: "Команда × Проект"
    group: "Группа"
    groupProject: "Группа × Проект"
    groupTeam: "Группа × Команда"
    builtin: "Встроенное"
    other: "Промпт"
    version: "v{version}"
    unset: "не задан"
    latest: "последняя, фиксируется при запуске"
</i18n>
