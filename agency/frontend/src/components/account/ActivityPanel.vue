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
    CardTitle Activity
    CardDescription
      | Sign-ins, password and two-factor changes, tokens, sessions and e-mail
      | changes on your account. Newest first.
  CardContent
    p.text-destructive.text-sm(v-if="error") {{ error }}
    DomainLogList(:logs="logs")
    .flex.justify-center.pt-3(v-if="canLoadMore")
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
import { Spinner } from "@/components/ui/spinner";
import { listAccountLogs, type AuditEntry } from "@/lib/audit";
import { computed, onMounted, ref } from "vue";

const PAGE_SIZE = 50;

const logs = ref<AuditEntry[]>([]);
const loading = ref(false);
const error = ref("");
const reachedEnd = ref(false);

const canLoadMore = computed(() => !reachedEnd.value && logs.value.length > 0);

function oldestID(): number {
  const last = logs.value.at(-1);
  return last ? last.id : 0;
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const page = await listAccountLogs(oldestID(), PAGE_SIZE);
    logs.value = [...logs.value, ...page.logs];
    if (page.logs.length < PAGE_SIZE) reachedEnd.value = true;
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : "Could not load activity.";
  } finally {
    loading.value = false;
  }
}

function loadMore() {
  void load();
}

onMounted(() => {
  void load();
});
</script>
