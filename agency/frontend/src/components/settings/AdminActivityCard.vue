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
Card
  CardHeader
    CardTitle {{ title }}
    CardDescription {{ description }}
  CardContent.flex.flex-col.gap-4
    form.flex.flex-wrap.items-end.gap-2(@submit.prevent="apply")
      Field.w-44
        FieldLabel(for="activity-actor") Actor
        Input#activity-actor(v-model="actor" placeholder="username")
      Field.flex-1
        FieldLabel(for="activity-search") Search
        Input#activity-search(v-model="q" placeholder="message text")
      Button(type="submit" size="sm" :disabled="loading")
        SearchIcon(data-icon="inline-start")
        span Apply
      .flex-1
      a.inline-flex.h-8.items-center.gap-1.rounded-md.border.px-3.text-sm(
        download
        class="hover:bg-muted"
        :href="exportUrl('csv')"
      )
        DownloadIcon(class="size-3.5")
        span CSV
      a.inline-flex.h-8.items-center.gap-1.rounded-md.border.px-3.text-sm(
        download
        class="hover:bg-muted"
        :href="exportUrl('json')"
      )
        DownloadIcon(class="size-3.5")
        span JSON
    p.text-destructive.text-sm(v-if="error") {{ error }}
    DomainLogList(:show-scope="!scope" :logs="logs")
    .flex.justify-center.pt-2(v-if="nextCursor")
      Button(variant="outline" size="sm" @click="loadMore" :disabled="loading")
        Spinner(v-if="loading" data-icon="inline-start")
        span Load more
</template>

<script setup lang="ts">
import DomainLogList from "@/components/audit/DomainLogList.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  eventsExportUrl,
  listEvents,
  type AuditEntry,
  type AuditFilter,
  type AuditScope,
} from "@/lib/audit";
import { DownloadIcon, SearchIcon } from "@lucide/vue";
import { onMounted, ref } from "vue";

const props = withDefaults(
  defineProps<{
    scope?: AuditScope;
    title: string;
    description: string;
  }>(),
  { scope: undefined },
);

const logs = ref<AuditEntry[]>([]);
const loading = ref(false);
const error = ref("");
const nextCursor = ref("");
const actor = ref("");
const q = ref("");

function currentFilter(): AuditFilter {
  return {
    scope: props.scope ?? "",
    actor: actor.value.trim(),
    q: q.value.trim(),
  };
}

function exportUrl(format: "csv" | "json"): string {
  return eventsExportUrl(format, currentFilter());
}

async function load(reset: boolean) {
  loading.value = true;
  error.value = "";
  try {
    const page = await listEvents({
      ...currentFilter(),
      limit: 50,
      cursor: reset ? "" : nextCursor.value,
    });
    logs.value = reset ? page.logs : [...logs.value, ...page.logs];
    nextCursor.value = page.next_cursor;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Could not load events.";
  } finally {
    loading.value = false;
  }
}

function apply() {
  nextCursor.value = "";
  void load(true);
}

function loadMore() {
  void load(false);
}

onMounted(() => void load(true));
</script>
