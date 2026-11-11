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
Dialog(v-model:open="open")
  DialogContent(class="sm:max-w-lg")
    DialogHeader
      DialogTitle {{ $t("siteSettings.title") }}
      DialogDescription
        | {{ $t("siteSettings.description") }}
    FieldGroup
      Field
        FieldLabel(for="site-title") {{ $t("common.title") }}
        Input#site-title(v-model="title" maxlength="120")
      Field
        FieldLabel(for="site-description") {{ $t("common.description") }}
        Textarea#site-description(v-model="description" :rows="3")
      Field(orientation="horizontal")
        Switch#site-only-creator(v-model="onlyCreatorMayEdit")
        FieldContent
          FieldLabel(for="site-only-creator")
            | {{ $t("siteSettings.creatorOnly") }}
          FieldDescription
            | {{ $t("siteSettings.creatorOnlyHint") }}
      Field(orientation="horizontal")
        Switch#site-agency-wide(v-model="agencyWide")
        FieldContent
          FieldLabel(for="site-agency-wide") {{ $t("siteSettings.agencyWide") }}
          FieldDescription
            | {{ $t("siteSettings.agencyWideHint") }}
      p.text-destructive.text-sm(v-if="error") {{ error }}
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="open = false"
        :disabled="saving"
      ) {{ $t("common.cancel") }}
      Button(type="button" @click="save" :disabled="saving")
        Spinner(v-if="saving" data-icon="inline-start")
        span {{ $t("common.saveChanges") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { agency } from "@/lib/sdk";
import type { Site } from "@cherrypi/sdk";
import { ref, watch } from "vue";

const props = defineProps<{ site: Site }>();
const emit = defineEmits<{ updated: [site: Site] }>();
const open = defineModel<boolean>("open", { default: false });

const title = ref("");
const description = ref("");
const onlyCreatorMayEdit = ref(true);
const agencyWide = ref(false);
const saving = ref(false);
const error = ref("");

watch(
  () => [props.site, open.value] as const,
  ([site, isOpen]) => {
    if (!isOpen) return;
    title.value = site.title;
    description.value = site.description;
    onlyCreatorMayEdit.value = site.onlyCreatorMayEdit;
    agencyWide.value = site.agencyWide;
    error.value = "";
  },
  { immediate: true },
);

async function save() {
  saving.value = true;
  error.value = "";
  try {
    const updated = await agency.updateSite(props.site.id, {
      title: title.value,
      description: description.value,
      onlyCreatorMayEdit: onlyCreatorMayEdit.value,
      agencyWide: agencyWide.value,
    });
    emit("updated", updated);
    open.value = false;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  siteSettings:
    title: "Site settings"
    description: "Settings apply immediately. Replacing the contents is done by the agent with publishSite."
    creatorOnly: "Only the creator may republish"
    creatorOnlyHint: "When off, other agents may publish the same site id. Owners and admins can always edit these settings."
    agencyWide: "Publish agency-wide"
    agencyWideHint: "List the site under Sites in the sidebar. Everyone signed in can still open it from its origin task."
ru:
  siteSettings:
    title: "Настройки сайта"
    description: "Настройки применяются сразу. Содержимое заменяет агент через publishSite."
    creatorOnly: "Переопубликовать может только автор"
    creatorOnlyHint: "Если выключено, другие агенты могут публиковать тот же идентификатор сайта. Владельцы и администраторы всегда могут менять эти настройки."
    agencyWide: "Публиковать для всего агентства"
    agencyWideHint: "Показывать сайт в разделе «Сайты» в боковой панели. Любой вошедший пользователь всё равно может открыть его из исходной задачи."
</i18n>
