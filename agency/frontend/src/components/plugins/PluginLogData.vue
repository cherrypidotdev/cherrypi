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
.flex.flex-wrap.gap-1(v-if="chips.length")
  span.inline-flex.max-w-full.items-center.gap-1.rounded-md.border.text-xs(
    v-for="chip in chips"
    :key="chip.key"
    class="bg-muted/40 px-1.5 py-0.5"
    :title="chip.title"
  )
    span.text-muted-foreground.shrink-0 {{ chip.label }}
    a.font-mono.break-all.underline-offset-2(
      v-if="chip.href"
      rel="noopener noreferrer"
      class="hover:underline"
      :href="chip.href"
      target="_blank"
    ) {{ chip.value }}
    span.font-mono.break-all(v-else) {{ chip.value }}
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{ dataJson?: string }>();

interface Chip {
  key: string;
  label: string;
  value: string;
  href?: string;
  title?: string;
}

const MAX_VALUE = 160;

function formatValue(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

const chips = computed<Chip[]>(() => {
  if (!props.dataJson) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(props.dataJson);
  } catch {
    return [];
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return [];
  }
  const out: Chip[] = [];
  for (const [key, raw] of Object.entries(parsed as Record<string, unknown>)) {
    if (raw === undefined || raw === "") continue;
    const full = formatValue(raw);
    const clipped = full.length > MAX_VALUE;
    out.push({
      key,
      label: key.replace(/_/g, " "),
      value: clipped ? `${full.slice(0, MAX_VALUE - 1)}…` : full,
      title: clipped ? full : undefined,
      href:
        typeof raw === "string" && /^https?:\/\//i.test(raw) ? raw : undefined,
    });
  }
  return out;
});
</script>
