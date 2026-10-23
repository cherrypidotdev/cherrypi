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
.flex.items-center.gap-3
  span.relative.size-12.shrink-0.overflow-hidden.border(
    :class="shape === 'square' ? 'rounded-lg' : 'rounded-full'"
  )
    img.absolute.inset-0.size-full.object-cover(
      v-if="picture"
      :src="picture.url"
      :alt="alt"
    )
    img.absolute.inset-0.size-full.object-contain(
      v-else-if="fallbackImage"
      :src="fallbackImage"
      :alt="''"
    )
    span.absolute.inset-0(v-else :innerHTML="fallback")
  .flex.flex-col.gap-1
    .flex.items-center.gap-2
      Button(
        type="button"
        variant="outline"
        size="sm"
        @click="pick"
        :disabled="busy"
      )
        Spinner(v-if="busy" data-icon="inline-start")
        UploadIcon(v-else data-icon="inline-start")
        span {{ picture ? "Replace" : "Upload" }}
      Button(
        v-if="picture"
        type="button"
        variant="ghost"
        size="sm"
        @click="remove"
        :disabled="busy"
      )
        Trash2Icon(data-icon="inline-start")
        span Remove
    input.hidden(
      ref="input"
      type="file"
      accept="image/png,image/jpeg,image/webp,image/gif"
      @change="onChange"
    )
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { agency } from "@/lib/sdk";
import type { Attachment } from "@cherrypi/sdk";
import { Trash2Icon, UploadIcon } from "@lucide/vue";
import { ref } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  picture?: Attachment;
  agentId?: string;
  projectId?: string;
  /** Set as the user's picture (the caller's own, or any user for admins). */
  userId?: string;
  /** Set as the agency logo or favicon instead of an agent/project picture. */
  agencyKind?: "logo" | "favicon";
  /** jdenticon-style SVG fallback, rendered as innerHTML. */
  fallback?: string;
  /** Image fallback (e.g. the bundled artwork), rendered as an img. */
  fallbackImage?: string;
  alt?: string;
  /** Preview crop: agents/projects are circular, the agency logo is square. */
  shape?: "circle" | "square";
}>();
const emit = defineEmits<{ changed: [picture: Attachment | undefined] }>();

const input = ref<HTMLInputElement>();
const busy = ref(false);

function pick() {
  input.value?.click();
}

async function onChange(event: Event) {
  const element = event.target as HTMLInputElement;
  const file = element.files?.[0];
  element.value = "";
  if (!file) return;

  busy.value = true;
  try {
    const attachment = await agency.uploadAttachment({
      file,
      filename: file.name,
      agentId: props.agentId,
      projectId: props.projectId,
      userId: props.userId,
      agencyLogo: props.agencyKind === "logo" ? true : undefined,
      agencyFavicon: props.agencyKind === "favicon" ? true : undefined,
    });
    emit("changed", attachment);
    toast.success("Picture updated");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = false;
  }
}

async function remove() {
  if (!props.picture) return;
  busy.value = true;
  try {
    await agency.deleteAttachment(props.picture.id);
    emit("changed", undefined);
    toast.success("Picture removed");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = false;
  }
}
</script>
