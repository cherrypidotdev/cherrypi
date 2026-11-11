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
Combobox(
  :placeholder="placeholderText"
  :id="id"
  :model-value="modelValue"
  :options="options"
  :empty-text="emptyTextValue"
  :size="size"
  :class="props.class"
  @update:model-value="emit('update:modelValue', $event)"
  :disabled="disabled"
)
  template(#item="{ option }")
    ActorAvatar(
      :src="actor(option)?.picture"
      :name="option.label"
      :username="option.value"
      :size="18"
    )
    span.flex.min-w-0.flex-col
      span.truncate {{ option.label }}
      span.text-muted-foreground.truncate.text-xs
        | @{{ option.value }}
        template(v-if="actor(option)?.kind === 'agent'") {{ $t("actorCombobox.agentSuffix") }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { useAgents } from "@/composables/useAgents";
import { useUsers } from "@/composables/useUsers";
import { computed, onMounted, type HTMLAttributes } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    modelValue?: string;

    kinds?: "all" | "users";
    placeholder?: string;
    emptyText?: string;
    disabled?: boolean;
    size?: "sm" | "default";
    id?: string;
    class?: HTMLAttributes["class"];
  }>(),
  {
    modelValue: "",
    kinds: "all",
    placeholder: undefined,
    emptyText: undefined,
    disabled: false,
    size: "default",
    id: undefined,
  },
);

const placeholderText = computed(
  () => props.placeholder ?? t("actorCombobox.search"),
);
const emptyTextValue = computed(
  () => props.emptyText ?? t("actorCombobox.empty"),
);

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

interface ActorChoice extends ComboboxOption {
  kind: "user" | "agent";
  picture?: string;
}

const { users, loadUsers } = useUsers();
const { agents, loadAgents } = useAgents();

onMounted(() => {
  void loadUsers();
  if (props.kinds !== "users") void loadAgents();
});

const actors = computed<ActorChoice[]>(() => {
  const group = (name: string) => (props.kinds === "users" ? undefined : name);
  const list: ActorChoice[] = users.value
    .map((user) => ({
      value: user.username,
      label: user.name || user.username,
      description: `@${user.username}`,
      group: group("Users"),
      kind: "user" as const,
      picture: user.picture?.url,
    }))
    .sort((a, b) => a.label.localeCompare(b.label));
  if (props.kinds !== "users") {
    list.push(
      ...agents.value
        .map((agent) => ({
          value: agent.username,
          label: agent.displayName || agent.username,
          description: `@${agent.username}`,
          group: group("Agents"),
          kind: "agent" as const,
          picture: agent.picture?.url,
        }))
        .sort((a, b) => a.label.localeCompare(b.label)),
    );
  }
  return list;
});

const options = computed<ComboboxOption[]>(() =>
  actors.value.map(({ value, label, description, group }) => ({
    value,
    label,
    description,
    group,
  })),
);

const byValue = computed(
  () => new Map(actors.value.map((actor) => [actor.value, actor])),
);

function actor(option: ComboboxOption): ActorChoice | undefined {
  return byValue.value.get(option.value);
}
</script>

<i18n lang="yaml">
en:
  actorCombobox:
    agentSuffix: "· agent"
    search: "Search actors…"
    empty: "No actors match."
ru:
  actorCombobox:
    agentSuffix: "· агент"
    search: "Поиск субъектов…"
    empty: "Субъекты не найдены."
</i18n>
