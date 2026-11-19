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
.bg-popover.text-popover-foreground.flex.min-h-0.flex-col.overflow-hidden.rounded-md.border.shadow-md(
  data-testid="reference-picker"
)
  .max-h-96.min-h-0.flex-1.overflow-y-auto.p-1(
    ref="list"
    role="listbox"
    data-testid="reference-picker-list"
  )
    p.text-muted-foreground.px-2.text-xs(v-if="loading && !rows.length") {{ $t("referencePicker.searching") }}
    p.text-muted-foreground.px-2.text-xs(v-else-if="!rows.length") {{ $t("referencePicker.empty") }}
    template(v-for="(suggestion, index) in rows" :key="rowKey(suggestion)")
      p.text-muted-foreground.px-2.pt-2.pb-1.text-xs.font-semibold.tracking-wide.uppercase(
        v-if="showHeader(index)"
      ) {{ sectionLabel(suggestion.reference.kind) }}
      button.inline-flex.w-full.min-w-0.cursor-pointer.items-center.gap-2.overflow-hidden.rounded-md.px-2.py-1.text-left.text-xs(
        type="button"
        role="option"
        :class="index === activeIndex ? 'bg-accent text-accent-foreground' : ''"
        :aria-selected="index === activeIndex"
        @click="pick(suggestion)"
        @mouseenter="activeIndex = index"
        data-reference-candidate
      )
        ActorAvatar(
          :src="avatarUrl(suggestion)"
          decorative
          :name="suggestion.label"
          :size="22"
        )
        span.flex.min-w-0.flex-1.items-center.gap-1
          span.shrink-0.truncate.font-medium {{ suggestion.label }}
          span.text-muted-foreground.truncate.text-xs {{ suggestion.detail }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import {
  loadReferenceDirectory,
  referenceDirectory,
  searchReferences,
  type ReferenceDirectory,
  type ReferenceSuggestion,
} from "@/lib/references";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    search?: string;

    preview?: ReferenceDirectory;
  }>(),
  { search: "", preview: undefined },
);

const emit = defineEmits<{ pick: [suggestion: ReferenceSuggestion] }>();

const list = ref<HTMLElement>();
const loading = ref(!props.preview);
const activeIndex = ref(0);

const rows = computed(() =>
  searchReferences(props.search, props.preview ?? referenceDirectory()),
);

onMounted(async () => {
  if (props.preview) return;
  await loadReferenceDirectory();
  loading.value = false;
});

watch(
  () => props.search,
  () => {
    activeIndex.value = 0;

    if (list.value) list.value.scrollTop = 0;
  },
);

watch(
  () => rows.value.length,
  (length) => {
    if (activeIndex.value >= length) {
      activeIndex.value = Math.max(0, length - 1);
    }
  },
);

function rowKey(suggestion: ReferenceSuggestion): string {
  return `${suggestion.reference.kind}:${suggestion.reference.key}`;
}

function showHeader(index: number): boolean {
  const kind = rows.value[index]?.reference.kind;
  if (!kind) return false;
  return index === 0 || rows.value[index - 1]?.reference.kind !== kind;
}

function sectionLabel(kind: ReferenceSuggestion["reference"]["kind"]): string {
  switch (kind) {
    case "actor":
      return t("referencePicker.actors");
    case "project":
      return t("referencePicker.projects");
    case "team":
      return t("referencePicker.teams");
    default:
      return "";
  }
}

function avatarUrl(suggestion: ReferenceSuggestion): string {
  if (suggestion.reference.kind !== "actor") return "";
  const directory = props.preview ?? referenceDirectory();
  const key = suggestion.reference.key;
  const user = directory.users.find(
    (item) => item.username === key || String(item.id) === key,
  );
  if (user) return user.picture?.url ?? "";
  const agent = directory.agents.find(
    (item) => item.username === key || item.actorId === key || item.id === key,
  );
  return agent?.picture?.url ?? "";
}

function scrollActiveIntoView() {
  const options = list.value?.querySelectorAll<HTMLElement>(
    "[data-reference-candidate]",
  );
  options?.[activeIndex.value]?.scrollIntoView({ block: "nearest" });
}

function moveActive(delta: number) {
  const count = rows.value.length;
  if (!count) return;
  activeIndex.value = (activeIndex.value + delta + count) % count;
  scrollActiveIntoView();
}

function pick(suggestion: ReferenceSuggestion) {
  emit("pick", suggestion);
}

function pickActive(): boolean {
  const candidate = rows.value[activeIndex.value];
  if (!candidate) return false;
  pick(candidate);
  return true;
}

defineExpose({ moveActive, pickActive });
</script>

<i18n lang="yaml">
en:
  referencePicker:
    actors: "Actors"
    projects: "Projects"
    teams: "Teams"
    searching: "Searching…"
    empty: "No matching actors, projects or teams."
ru:
  referencePicker:
    actors: "Субъекты"
    projects: "Проекты"
    teams: "Команды"
    searching: "Поиск…"
    empty: "Нет подходящих субъектов, проектов или команд."
</i18n>
