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
.flex.flex-col.gap-2
  .flex.flex-col.gap-2
    .flex.items-center.gap-2(v-for="(link, index) in rows" :key="index")
      Input.flex-1(
        :placeholder="$t('actor.links.urlPlaceholder')"
        :model-value="link.url"
        :aria-label="$t('actor.links.url')"
        @update:model-value="update(index, 'url', String($event))"
      )
      Input.w-40(
        :placeholder="$t('actor.links.labelPlaceholder')"
        :model-value="link.label ?? ''"
        :aria-label="$t('actor.links.label')"
        @update:model-value="update(index, 'label', String($event))"
      )
      Button(
        type="button"
        variant="ghost"
        size="icon"
        :aria-label="$t('common.remove')"
        @click="remove(index)"
      )
        Trash2Icon
  p.text-destructive.text-sm(v-if="error") {{ error }}
  .flex.items-center.justify-between.gap-2
    p.text-muted-foreground.text-xs {{ $t("actor.links.hint") }}
    Button(
      type="button"
      variant="outline"
      size="sm"
      @click="add"
      :disabled="rows.length >= maxLinks"
    )
      PlusIcon(data-icon="inline-start")
      span {{ $t("actor.links.add") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ActorLinkInput } from "@cherrypi/sdk";
import { PlusIcon, Trash2Icon } from "@lucide/vue";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const MAX_LINKS = 20;
const { t } = useI18n();

const props = withDefaults(
  defineProps<{ modelValue?: ActorLinkInput[]; disabled?: boolean }>(),
  { modelValue: () => [], disabled: false },
);
const emit = defineEmits<{ "update:modelValue": [value: ActorLinkInput[]] }>();

const rows = ref<ActorLinkInput[]>([]);
const error = ref("");

watch(
  () => props.modelValue,
  (value) => {
    const next = (value ?? []).map((link) => ({ ...link }));

    const current = rows.value.filter((link) => link.url.trim() !== "");
    if (sameLinks(next, current)) return;
    rows.value = next;
  },
  { immediate: true, deep: true },
);

const maxLinks = MAX_LINKS;

function update(index: number, field: "url" | "label", value: string) {
  const row = rows.value[index];
  if (!row) return;
  row[field] = value;
  emitChange();
}

function add() {
  if (rows.value.length >= MAX_LINKS) return;

  rows.value.push({ url: "", label: "" });
}

function remove(index: number) {
  rows.value.splice(index, 1);
  emitChange();
}

function emitChange() {
  const links = rows.value
    .map((link) => ({
      url: link.url.trim(),
      label: (link.label ?? "").trim(),
    }))
    .filter((link) => link.url !== "");
  const invalid = links.find((link) => !/^https?:\/\/\S+$/i.test(link.url));
  error.value = invalid ? t("actor.links.invalid", { url: invalid.url }) : "";
  emit("update:modelValue", links);
}

function sameLinks(a: ActorLinkInput[], b: ActorLinkInput[]): boolean {
  if (a.length !== b.length) return false;
  return a.every(
    (link, index) =>
      link.url === b[index]?.url &&
      (link.label ?? "") === (b[index]?.label ?? ""),
  );
}
</script>

<i18n lang="yaml">
en:
  actor:
    links:
      title: "Links"
      add: "Add link"
      url: "URL"
      label: "Label"
      urlPlaceholder: "https://example.com"
      labelPlaceholder: "Label (optional)"
      hint: "Absolute http(s) URLs, up to 20. The label falls back to the host."
      invalid: "{url} is not an absolute http(s) URL."
ru:
  actor:
    links:
      title: "Ссылки"
      add: "Добавить ссылку"
      url: "URL"
      label: "Подпись"
      urlPlaceholder: "https://example.com"
      labelPlaceholder: "Подпись (необязательно)"
      hint: "Абсолютные http(s)-адреса, до 20. Подпись по умолчанию — домен."
      invalid: "{url} не является абсолютным http(s)-адресом."
</i18n>
