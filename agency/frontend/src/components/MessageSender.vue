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
.flex.items-center.gap-1(v-if="sender" :data-sender-kind="sender.kind")
  template(v-if="isSystem")
    Badge.font-normal(variant="outline" class="h-4.5 px-1.5 text-[10px]") {{ kindLabel }}

  template(v-else)
    RouterLink.shrink-0(
      v-if="avatar && route"
      :to="route"
      :title="$t('messageSender.view', { name: displayName })"
    )
      ActorAvatar(
        :src="picture?.url"
        :name="sender.name"
        :username="sender.username"
        :size="20"
      )

    ActorAvatar(
      v-else-if="avatar"
      :src="picture?.url"
      :name="sender.name"
      :username="sender.username"
      :size="20"
    )

    component(class="hover:underline" :to="route" :is="route ? RouterLink : 'div'")
      span.text-xs.whitespace-nowrap(
        :class="{ 'text-muted-foreground font-mono': isUsername, 'text-foreground font-semibold': isName }"
      ) {{ isUsername ? "@" : "" }}{{ displayName }}

    Badge.font-normal(
      v-if="kindLabel"
      variant="outline"
      class="h-4.5 px-1.5 text-[10px]"
    ) {{ kindLabel }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import { Badge } from "@/components/ui/badge";
import { actorRoute } from "@/lib/actors";
import type { MessageSender } from "@/lib/console";
import type { Attachment } from "@cherrypi/sdk";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    sender: MessageSender;

    picture?: Attachment;

    avatar?: boolean;
  }>(),
  { avatar: true },
);

const isSystem = computed(() => props.sender.kind === "system");

const route = computed(() => actorRoute(props.sender));

const isName = computed(() => !!props.sender.name);

const isUsername = computed(
  () => !props.sender.name && !!props.sender.username,
);

const displayName = computed(
  () => props.sender.name || props.sender.username || "Unknown",
);

const kindLabel = computed(() => {
  switch (props.sender.kind) {
    case "agent":
      return t("admin.agents.columnAgent");
    case "plugin":
      return t("messageSender.plugin");
    case "system":
      return t("messageSender.system");
    default:
      return "";
  }
});
</script>

<i18n lang="yaml">
en:
  messageSender:
    view: "View {name}"
    plugin: "Plugin"
    system: "System"
ru:
  messageSender:
    view: "Открыть {name}"
    plugin: "Плагин"
    system: "Система"
</i18n>
