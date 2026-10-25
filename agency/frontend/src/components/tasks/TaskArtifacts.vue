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
.flex.flex-col.gap-3(v-if="artifacts.length")
  .flex.flex-col.gap-2.rounded-md.border.p-3(
    v-for="artifact in artifacts"
    :key="artifact.artifactId"
  )
    .flex.items-start.justify-between.gap-2
      .flex.min-w-0.flex-col(class="gap-0.5")
        span.truncate.text-sm.font-medium {{ artifact.name || artifact.artifactId }}
        span.text-muted-foreground.text-xs.break-words(
          v-if="artifact.description"
        ) {{ artifact.description }}
      span.text-muted-foreground.inline-flex.shrink-0.items-center.gap-1.text-xs(
        v-if="artifact.streaming"
        title="The agent is still adding parts"
      )
        Spinner.size-3
        | streaming
    .flex.flex-col.gap-2(v-if="artifact.parts.length")
      template(v-for="(part, index) in artifact.parts" :key="index")
        p.text-xs.break-words.whitespace-pre-wrap(
          v-if="part.content.case === 'text'"
        ) {{ part.content.value }}
        pre.text-muted-foreground.font-mono.text-xs.break-words.whitespace-pre-wrap(
          v-else-if="part.content.case === 'dataJson'"
        ) {{ formatData(part.content.value) }}
        template(v-else-if="part.content.case === 'file'")
          img.max-h-48.w-full.rounded-md.border.object-contain(
            v-if="isImage(part)"
            :src="part.content.value.url"
            :alt="part.filename || artifact.name"
          )
          a.flex.items-center.gap-2.text-xs(
            v-else
            class="hover:underline"
            rel="noopener noreferrer"
            :href="part.content.value.url"
            :title="part.content.value.url"
            target="_blank"
          )
            FileIcon.size-3.shrink-0
            span.truncate {{ part.filename || part.content.value.url }}
    span.text-muted-foreground.text-xs(v-else) No parts.
span.text-muted-foreground.text-sm(v-else)
  | No artifacts yet. The agent publishes its outputs here.
</template>

<script setup lang="ts">
import { Spinner } from "@/components/ui/spinner";
import { isImageMediaType } from "@/lib/attachments";
import type { Artifact, ArtifactPart } from "@cherrypi/sdk";
import { FileIcon } from "@lucide/vue";

defineProps<{ artifacts: Artifact[] }>();

function isImage(part: ArtifactPart): boolean {
  return (
    part.content.case === "file" &&
    isImageMediaType(part.mediaType || guessMediaType(part.filename))
  );
}

function guessMediaType(filename: string): string {
  const ext = filename.split(".").pop()?.toLowerCase() ?? "";
  switch (ext) {
    case "png":
      return "image/png";
    case "jpg":
    case "jpeg":
      return "image/jpeg";
    case "webp":
      return "image/webp";
    case "gif":
      return "image/gif";
    default:
      return "";
  }
}

function formatData(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
</script>
