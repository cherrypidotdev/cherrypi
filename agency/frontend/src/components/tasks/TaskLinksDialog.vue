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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent(class="sm:max-w-lg")
    DialogHeader
      DialogTitle Edit links
      DialogDescription
        | Labeled URLs shown on the task, such as a PR, an issue, or a
        | deployment. Only absolute http(s) URLs are kept.
    .flex.flex-col.gap-2
      .flex.flex-col.gap-1(v-if="rows.length")
        .flex.items-start.gap-2(v-for="(row, index) in rows" :key="index")
          Input.flex-1(
            v-model="row.label"
            placeholder="Label (e.g. PR)"
            data-test="link-label"
          )
          Input(
            v-model="row.url"
            placeholder="https://…"
            class="flex-[2]"
            data-test="link-url"
          )
          Button(
            type="button"
            variant="ghost"
            size="icon-sm"
            title="Remove link"
            aria-label="Remove link"
            @click="removeRow(index)"
            data-test="link-remove"
          )
            XIcon.size-3
      p.text-muted-foreground.text-xs(v-else)
        | No links yet. Add one below.
      Button.self-start(
        type="button"
        variant="outline"
        size="sm"
        @click="addRow"
        data-test="link-add"
      )
        PlusIcon(data-icon="inline-start")
        span Add link
      p.text-destructive.text-xs(v-if="hasInvalidURL")
        | Only absolute http(s) URLs are allowed.
    DialogFooter
      Button(
        type="button"
        variant="outline"
        @click="emit('update:open', false)"
      )
        | Cancel
      Button(
        type="button"
        @click="save"
        :disabled="busy || !canSave"
        data-test="link-save"
      )
        Spinner(v-if="busy" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span Save
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
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { CheckIcon, PlusIcon, XIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

interface LinkRow {
  label: string;
  url: string;
}

const props = defineProps<{
  open: boolean;

  links: { url: string; label?: string }[];
  busy?: boolean;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  save: [links: { url: string; label: string }[]];
}>();

const rows = ref<LinkRow[]>([]);

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    rows.value = props.links.map((link) => ({
      label: link.label ?? "",
      url: link.url,
    }));
  },
  { immediate: true },
);

function addRow() {
  rows.value.push({ label: "", url: "" });
}

function removeRow(index: number) {
  rows.value.splice(index, 1);
}

function isHTTPURL(raw: string): boolean {
  try {
    const url = new URL(raw);
    return (
      (url.protocol === "http:" || url.protocol === "https:") && !!url.host
    );
  } catch {
    return false;
  }
}

const cleaned = computed(() =>
  rows.value
    .map((row) => ({ label: row.label.trim(), url: row.url.trim() }))
    .filter((row) => row.url !== ""),
);

const hasInvalidURL = computed(() =>
  cleaned.value.some((row) => !isHTTPURL(row.url)),
);

const canSave = computed(() => !hasInvalidURL.value);

function save() {
  if (!canSave.value || props.busy) return;
  emit("save", cleaned.value);
}
</script>
