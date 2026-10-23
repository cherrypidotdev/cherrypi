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
  Card
    CardHeader
      CardTitle Filters
      CardDescription
        | Every mutation and auth event across the five domain logs. Reads are
        | never logged.
    CardContent.flex.flex-col.gap-4
      .grid.gap-3(class="sm:grid-cols-2 lg:grid-cols-3")
        Field
          FieldLabel Scope
          Select(v-model="scope")
            SelectTrigger
              SelectValue
            SelectContent
              SelectItem(value="all") All scopes
              SelectItem(v-for="s in AUDIT_SCOPES" :key="s" :value="s")
                | {{ scopeLabel(s) }}
        Field
          FieldLabel Actor
          Input(v-model="actor" placeholder="username")
        Field
          FieldLabel Kind
          Input(v-model="kind" placeholder="login, created…")
        Field
          FieldLabel Since
          Input(v-model="since" type="date")
        Field
          FieldLabel Until
          Input(v-model="until" type="date")
        Field
          FieldLabel Search
          Input(v-model="q" placeholder="message text")
      .flex.flex-wrap.items-center.gap-2
        Button(size="sm" @click="apply" :disabled="loading")
          SearchIcon(data-icon="inline-start")
          span Apply
        Button(variant="outline" size="sm" @click="reset" :disabled="loading")
          | Reset
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

  Card
    CardHeader
      CardTitle
        | Activity
        span.text-muted-foreground.font-normal(v-if="logs.length")
          | &nbsp;({{ logs.length }} loaded)
      CardDescription Newest first.
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      DomainLogList(show-scope :logs="logs")
      .flex.justify-center.pt-3(v-if="nextCursor")
        Button(
          variant="outline"
          size="sm"
          @click="loadMore"
          :disabled="loading"
        )
          Spinner(v-if="loading" data-icon="inline-start")
          span Load more

  Card
    CardHeader
      CardTitle Retention
      CardDescription
        | Domain-log entries older than this are pruned by the scheduler.
    CardContent
      form.flex.items-end.gap-3(@submit.prevent="saveRetention")
        Field.w-40
          FieldLabel(for="retention-days") Days
          Input#retention-days(
            v-model.number="retentionDays"
            type="number"
            min="1"
            max="3650"
          )
        Button(type="submit" variant="outline" :disabled="savingRetention")
          Spinner(v-if="savingRetention" data-icon="inline-start")
          span Save
      p.text-destructive.mt-2.text-sm(v-if="settingsError")
        | {{ settingsError }}
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import {
  AUDIT_SCOPES,
  eventsExportUrl,
  getEventsSettings,
  listEvents,
  setEventsSettings,
  type AuditEntry,
  type AuditScope,
} from "@/lib/audit";
import { DownloadIcon, SearchIcon } from "@lucide/vue";
import { onMounted, ref } from "vue";

const logs = ref<AuditEntry[]>([]);
const loading = ref(false);
const error = ref("");
const nextCursor = ref("");

const scope = ref<AuditScope | "all">("all");
const actor = ref("");
const kind = ref("");
const since = ref("");
const until = ref("");
const q = ref("");

const retentionDays = ref(90);
const savingRetention = ref(false);
const settingsError = ref("");

function scopeLabel(value: AuditScope): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function currentFilter() {
  return {
    scope: scope.value === "all" ? ("" as const) : scope.value,
    actor: actor.value.trim(),
    kind: kind.value.trim(),
    since: since.value,
    until: until.value,
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

function reset() {
  scope.value = "all";
  actor.value = "";
  kind.value = "";
  since.value = "";
  until.value = "";
  q.value = "";
  apply();
}

function loadMore() {
  void load(false);
}

async function loadSettings() {
  try {
    const settings = await getEventsSettings();
    retentionDays.value = settings.retention_days;
  } catch {
    // Non-admins never reach this panel; a failure leaves the default.
  }
}

async function saveRetention() {
  savingRetention.value = true;
  settingsError.value = "";
  try {
    await setEventsSettings(retentionDays.value);
  } catch (err) {
    settingsError.value =
      err instanceof Error ? err.message : "Could not save settings.";
  } finally {
    savingRetention.value = false;
  }
}

onMounted(() => {
  void load(true);
  void loadSettings();
});
</script>
