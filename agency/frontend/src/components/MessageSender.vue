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
.flex.items-center.gap-2(v-if="sender" :data-sender-kind="sender.kind")
  //- System is not a person: it gets its badge and nothing else.
  template(v-if="isSystem")
    Badge.font-normal(variant="outline" class="h-4.5 px-1.5 text-[10px]") {{ kindLabel }}
  template(v-else)
    ActorAvatar(
      :src="picture?.url"
      :name="sender.name"
      :username="sender.username"
      :size="20"
    )
    span.text-foreground.text-xs.font-medium.whitespace-nowrap {{ displayName }}
    span.text-muted-foreground.text-xs.whitespace-nowrap(
      v-if="sender.username && sender.username !== displayName"
    ) @{{ sender.username }}
    Badge.font-normal(
      v-if="kindLabel"
      variant="outline"
      class="h-4.5 px-1.5 text-[10px]"
    ) {{ kindLabel }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import { Badge } from "@/components/ui/badge";
import type { MessageSender } from "@/lib/console";
import type { Attachment } from "@cherrypi/sdk";
import { computed } from "vue";

const props = defineProps<{
  sender: MessageSender;
  /** The actor's uploaded picture; initials stand in when it is unset. */
  picture?: Attachment;
}>();

const isSystem = computed(() => props.sender.kind === "system");

/** The name to show, preferring the display name over the username. */
const displayName = computed(
  () => props.sender.name || props.sender.username || "Unknown",
);

// A user is the default author, so only a non-user kind gets a badge.
const kindLabel = computed(() => {
  switch (props.sender.kind) {
    case "agent":
      return "Agent";
    case "system":
      return "System";
    default:
      return "";
  }
});
</script>
