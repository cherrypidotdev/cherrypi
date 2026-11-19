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
AvatarTip(
  v-bind="$attrs"
  :src="pictureUrl"
  card-slot="actor-tip"
  card-width="w-60 max-w-60"
  :to="to"
  :emoji="emoji"
  :meta="meta"
  :name="label"
  :seed="actorId"
  :size="size"
)
  .flex.items-center.gap-2
    ActorAvatar(
      :src="pictureUrl"
      decorative
      :emoji="emoji"
      :name="label"
      :size="32"
    )
    .flex.min-w-0.flex-col
      span.font-medium {{ label }}
      span.text-muted-foreground(v-if="username") @{{ username }}
  .flex.flex-wrap.items-center.gap-1(v-if="kindLabel || actorId")
    span.bg-muted.text-muted-foreground.text-3xs.rounded.font-medium.tracking-wide.uppercase(
      v-if="kindLabel"
      class="px-1.5 py-0.5"
    ) {{ kindLabel }}

    span.text-muted-foreground.text-3xs(v-if="actorId && (name || username)")
      | {{ `#${actorId}` }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import AvatarTip from "@/components/AvatarTip.vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { RouteLocationRaw } from "vue-router";

// creator or leader, a comment's author), and the tip's second line doubles as

defineOptions({ inheritAttrs: false });

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    name?: string;

    username?: string;

    kind?: string;

    pictureUrl?: string;

    emoji?: string;

    actorId?: string;

    to?: RouteLocationRaw | null;

    size?: number;
  }>(),
  {
    name: "",
    username: "",
    kind: "",
    pictureUrl: "",
    emoji: "",
    actorId: "",
    to: null,
    size: 20,
  },
);

const label = computed(
  () =>
    props.name || props.username || (props.actorId ? `#${props.actorId}` : "?"),
);

const kindLabel = computed(() => {
  if (props.kind === "agent") return t("actorTooltip.agent");
  if (props.kind === "user") return t("actorTooltip.user");
  return "";
});

// `@handle · Agent`, the one-line summary the avatar's accessible name uses.

const meta = computed(() => {
  const parts = [
    props.username ? `@${props.username}` : "",
    kindLabel.value,
  ].filter(Boolean);
  if (parts.length) return parts.join(" · ");
  return props.actorId ? `#${props.actorId}` : "";
});
</script>

<i18n lang="yaml">
en:
  actorTooltip:
    agent: "Agent"
    user: "User"
ru:
  actorTooltip:
    agent: "Агент"
    user: "Пользователь"
</i18n>
