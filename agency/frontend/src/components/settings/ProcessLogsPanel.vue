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
      .flex.flex-wrap.items-start.justify-between.gap-3
        .flex.flex-col.gap-1
          CardTitle Agency process log
          CardDescription
            | Everything this agency process wrote to stdout and stderr, captured
            | on disk with a UTC timestamp before every line. This is the current
            | run only; older runs stay in the log directory until rotated out.
        .flex.items-center.gap-2
          Button(
            variant="outline"
            size="sm"
            @click="toggleFollow"
            :disabled="loading"
          )
            PauseIcon(v-if="following" data-icon="inline-start")
            PlayIcon(v-else data-icon="inline-start")
            span {{ following ? "Pause" : "Follow" }}
          Button(
            variant="outline"
            size="sm"
            @click="reload"
            :disabled="loading"
          )
            RefreshCwIcon(data-icon="inline-start")
            span Refresh
          Button(as-child variant="outline" size="sm")
            a(download :href="processLogDownloadUrl()")
              DownloadIcon(data-icon="inline-start")
              span Download

    CardContent.flex.flex-col.gap-3
      p.text-destructive.text-sm(v-if="error") {{ error }}
      p.text-muted-foreground.text-sm(v-else-if="loading") Loading the process log…

      template(v-else)
        .text-muted-foreground.flex.flex-wrap.items-center.gap-x-4.gap-y-1.text-xs
          span.font-mono {{ meta?.version || "unknown version" }}
          span(v-if="meta") started {{ formatStarted(meta.started_at) }}
          span(v-if="meta") pid {{ meta.pid }}
          span.font-mono(v-if="meta" :title="meta.dir") {{ meta.file }}
          Badge(v-if="following" variant="secondary") Live
          span(v-else) Paused

        .relative
          pre.bg-muted.text-foreground.min-h-40.overflow-auto.rounded-md.border.p-3.font-mono.text-xs.leading-relaxed.break-words.whitespace-pre-wrap(
            ref="output"
            class="max-h-[65vh]"
          )
          p.text-muted-foreground.absolute.top-3.left-3.text-xs(
            v-if="!hasOutput"
          ) No output captured yet.
        p.text-muted-foreground.text-xs
          | Lines are raw process output and may contain secrets; the page and the
          | API are administrator-only.
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  getProcessLog,
  processLogDownloadUrl,
  processLogStreamUrl,
  type ProcessLogMeta,
} from "@/lib/processLogs";
import { DownloadIcon, PauseIcon, PlayIcon, RefreshCwIcon } from "@lucide/vue";
import { nextTick, onMounted, onUnmounted, ref } from "vue";

const MAX_LINES = 5000;

const meta = ref<ProcessLogMeta>();
const offset = ref(0);
const following = ref(true);
const loading = ref(true);
const error = ref("");
const hasOutput = ref(false);
const output = ref<HTMLElement>();

let source: EventSource | undefined;

let rendered: string[] = [];

onMounted(reload);
onUnmounted(stopStream);

function append(line: string) {
  rendered.push(line);
  if (rendered.length > MAX_LINES) {
    rendered = rendered.slice(-MAX_LINES);
    paint(rendered);
  } else {
    const element = output.value;
    if (element) {
      const prefix = element.textContent ? "\n" : "";
      element.textContent = `${element.textContent}${prefix}${line}`;
    }
    hasOutput.value = true;
  }
  if (following.value) scrollToBottom();
}

function paint(lines: string[]) {
  if (output.value) output.value.textContent = lines.join("\n");
  hasOutput.value = lines.length > 0;
}

function scrollToBottom() {
  const element = output.value;
  if (!element) return;
  element.scrollTop = element.scrollHeight;
}

async function reload() {
  loading.value = true;
  error.value = "";
  stopStream();
  try {
    const page = await getProcessLog(1000);
    meta.value = page.meta;
    rendered = page.lines;
    offset.value = page.offset;

    loading.value = false;
    await nextTick();
    paint(rendered);
    if (following.value) startStream();
    scrollToBottom();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
    loading.value = false;
  }
}

function startStream() {
  stopStream();
  const stream = new EventSource(processLogStreamUrl(offset.value));
  stream.onopen = () => {
    error.value = "";
  };
  stream.onmessage = (event: MessageEvent<string>) => {
    try {
      const data = JSON.parse(event.data) as { offset: number; text: string };

      if (data.offset <= offset.value) return;
      offset.value = data.offset;
      append(data.text);
    } catch {}
  };

  source = stream;
}

function stopStream() {
  source?.close();
  source = undefined;
}

function toggleFollow() {
  following.value = !following.value;
  if (following.value) {
    void reload();
  } else {
    stopStream();
  }
}

function formatStarted(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
</script>
