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
.flex.flex-col.overflow-hidden.rounded-md.border
  .bg-muted.flex.items-center.justify-between.border-b.px-3.text-xs(
    v-if="label"
    class="py-1.5"
  )
    span.font-medium {{ label }}
    span.text-muted-foreground {{ summary }}
  pre.max-h-80.overflow-auto.font-mono.text-xs.leading-5
    template(v-if="!rows.length")
      span.text-muted-foreground.block.px-3.py-2 {{ $t("promptDiff.noChanges") }}
    template(v-for="(row, index) in rows" :key="index")
      .flex
        span.text-muted-foreground.w-10.shrink-0.border-r.px-2.text-right.select-none(
          :class="row.type === 'add' ? 'bg-emerald-500/10' : row.type === 'remove' ? 'bg-red-500/10' : ''"
        ) {{ row.oldLine ?? "" }}
        span.text-muted-foreground.w-10.shrink-0.border-r.px-2.text-right.select-none(
          :class="row.type === 'add' ? 'bg-emerald-500/10' : row.type === 'remove' ? 'bg-red-500/10' : ''"
        ) {{ row.newLine ?? "" }}
        span.w-5.shrink-0.text-center.select-none(
          :class="row.type === 'add' ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300' : row.type === 'remove' ? 'bg-red-500/10 text-red-700 dark:text-red-300' : 'text-muted-foreground'"
        ) {{ row.type === "add" ? "+" : row.type === "remove" ? "−" : " " }}
        span.px-2.break-words.whitespace-pre-wrap(
          :class="row.type === 'add' ? 'bg-emerald-500/10' : row.type === 'remove' ? 'bg-red-500/10' : ''"
        ) {{ row.text || " " }}
</template>

<script setup lang="ts">
import { diffLines } from "diff";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

type RowType = "same" | "add" | "remove";

interface Row {
  type: RowType;
  text: string;
  oldLine?: number;
  newLine?: number;
}

const props = withDefaults(
  defineProps<{
    before: string;
    after: string;
    label?: string;
  }>(),
  { label: "" },
);

const rows = computed<Row[]>(() => {
  const parts = diffLines(props.before, props.after);
  const out: Row[] = [];
  let oldLine = 1;
  let newLine = 1;
  for (const part of parts) {
    const lines = part.value.split("\n");

    if (lines.length > 1 && lines[lines.length - 1] === "") lines.pop();
    for (const line of lines) {
      if (part.added) {
        out.push({ type: "add", text: line, newLine });
        newLine++;
      } else if (part.removed) {
        out.push({ type: "remove", text: line, oldLine });
        oldLine++;
      } else {
        out.push({ type: "same", text: line, oldLine, newLine });
        oldLine++;
        newLine++;
      }
    }
  }
  return out;
});

const summary = computed(() => {
  let added = 0;
  let removed = 0;
  for (const row of rows.value) {
    if (row.type === "add") added++;
    if (row.type === "remove") removed++;
  }
  if (!added && !removed) return t("promptDiff.noChanges");
  return `+${added} −${removed}`;
});
</script>

<i18n lang="yaml">
en:
  promptDiff:
    noChanges: "No changes."
ru:
  promptDiff:
    noChanges: "Изменений нет."
</i18n>
