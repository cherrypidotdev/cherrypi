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
RouterLink.d-chat-image(
  v-if="sender && route"
  :to="route"
  :title="$t('chatAvatar.view', { label })"
)
  ActorAvatar(
    :src="picture?.url"
    :name="sender.name"
    :username="sender.username"
    :size="size"
  )
.d-chat-image(v-else-if="sender")
  ActorAvatar(
    :src="picture?.url"
    :name="sender.name"
    :username="sender.username"
    :size="size"
  )
.d-chat-image(v-else)
  .d-avatar.d-avatar-placeholder
    .bg-muted.text-muted-foreground.rounded-full.border(
      :style="{ width: `${size}px`, height: `${size}px` }"
    )
      BotIcon(:style="{ width: `${Math.round(size / 2)}px` }")
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import { actorRoute } from "@/lib/actors";
import type { MessageSender } from "@/lib/console";
import type { Attachment } from "@cherrypi/sdk";
import { BotIcon } from "@lucide/vue";
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    sender?: MessageSender;

    picture?: Attachment;

    size?: number;
  }>(),
  { sender: undefined, picture: undefined, size: 32 },
);

const route = computed(() => actorRoute(props.sender));
const label = computed(
  () =>
    props.sender?.name ||
    props.sender?.username ||
    props.sender?.kind ||
    "actor",
);
</script>

<i18n lang="yaml">
en:
  chatAvatar:
    view: "View {label}"
ru:
  chatAvatar:
    view: "Открыть {label}"
</i18n>
