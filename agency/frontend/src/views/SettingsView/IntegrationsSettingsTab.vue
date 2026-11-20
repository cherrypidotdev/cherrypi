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
.flex.flex-col.gap-6
  VoiceInputPanel
  Alert
    AlertTitle {{ $t("settings.webhooks.title") }}
    AlertDescription
      | {{ $t("settings.webhooks.description") }}
  WebhooksPanel
</template>

<script setup lang="ts">
import VoiceInputPanel from "@/components/settings/VoiceInputPanel.vue";
import WebhooksPanel from "@/components/settings/WebhooksPanel.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
</script>

<i18n lang="yaml">
en:
  settings:
    webhooks:
      title: "Webhooks"
      description: "Org-wide webhooks are being simplified: they will fire on every agency event and no longer offer an event picker. That change is tracked in CHPI-176."
ru:
  settings:
    webhooks:
      title: "Вебхуки"
      description: "Вебхуки агентства упрощаются: они будут срабатывать на каждое событие агентства, а выбор событий исчезнет. Это изменение отслеживается в CHPI-176."
</i18n>
