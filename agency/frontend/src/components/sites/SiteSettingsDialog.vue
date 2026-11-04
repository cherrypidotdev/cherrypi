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
      DialogTitle Site settings
      DialogDescription
        | Settings apply immediately. Replacing the contents is done by the
        | agent with publishSite.
    FieldGroup
      Field
        FieldLabel(for="site-title") Title
        Input#site-title(v-model="title" maxlength="120")
      Field
        FieldLabel(for="site-description") Description
        Textarea#site-description(v-model="description" :rows="3")
      Field(orientation="horizontal")
        Switch#site-only-creator(v-model="onlyCreatorMayEdit")
        FieldContent
          FieldLabel(for="site-only-creator")
            | Only the creator may republish
          FieldDescription
            | When off, other agents may publish the same site id. Owners and
            | admins can always edit these settings.
      Field(orientation="horizontal")
        Switch#site-agency-wide(v-model="agencyWide")
        FieldContent
          FieldLabel(for="site-agency-wide") Publish agency-wide
          FieldDescription
            | List the site under Sites in the sidebar. Everyone signed in can
            | still open it from its origin task.
      p.text-destructive.text-sm(v-if="error") {{ error }}
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="open = false"
        :disabled="saving"
      ) Cancel
      Button(type="button" @click="save" :disabled="saving")
        Spinner(v-if="saving" data-icon="inline-start")
        span Save changes
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
