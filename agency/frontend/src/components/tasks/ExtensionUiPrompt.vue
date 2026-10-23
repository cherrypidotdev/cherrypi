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
Card.border-primary
  CardHeader
    CardTitle.text-sm.whitespace-pre-wrap {{ request.title || dialogLabel }}
    CardDescription.whitespace-pre-wrap(v-if="request.message") {{ request.message }}
  CardContent
    template(v-if="request.method === 'select'")
      .flex.flex-col.gap-2
        Button.h-auto.justify-start.py-2.text-left.whitespace-normal(
          v-for="option in options"
          :key="option.raw"
          variant="outline"
          @click="respond({ value: option.raw })"
          :disabled="busy"
        )
          .flex.flex-col(class="gap-0.5")
            .flex.items-baseline.gap-2
              span.text-muted-foreground.shrink-0.text-xs(
                v-if="option.index !== null"
              ) {{ option.index }}.
              span.font-medium {{ option.label }}
            p.text-muted-foreground.text-xs(v-if="option.description") {{ option.description }}
    template(v-else-if="request.method === 'confirm'")
      .flex.gap-2
        Button(@click="respond({ confirmed: true })" :disabled="busy")
          CheckIcon(data-icon="inline-start")
          span Confirm
        Button(
          variant="outline"
          @click="respond({ confirmed: false })"
          :disabled="busy"
        )
          XIcon(data-icon="inline-start")
          span Decline
    template(v-else-if="request.method === 'input'")
      form.flex.flex-col.gap-2(@submit.prevent="respond({ value: text })")
        Input(v-model="text" :placeholder="request.placeholder || ''")
        .flex.gap-2
          Button(type="submit" :disabled="busy")
            SendIcon(data-icon="inline-start")
            span Submit
          Button(
            type="button"
            variant="outline"
            @click="cancel"
            :disabled="busy"
          )
            XIcon(data-icon="inline-start")
            span Cancel
    template(v-else-if="request.method === 'editor'")
      form.flex.flex-col.gap-2(@submit.prevent="respond({ value: text })")
        Textarea(v-model="text" rows="8")
        .flex.gap-2
          Button(type="submit" :disabled="busy")
            CheckIcon(data-icon="inline-start")
            span Save
          Button(
            type="button"
            variant="outline"
            @click="cancel"
            :disabled="busy"
          )
            XIcon(data-icon="inline-start")
            span Cancel
  CardFooter.justify-end.border-t
    Button(variant="ghost" size="sm" @click="cancel" :disabled="busy")
      XIcon(data-icon="inline-start")
      span Decline
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { parseOption } from "@/lib/questionnaire";
import {
  type ExtensionUiRequest,
  type ExtensionUiResponse,
} from "@cherrypi/sdk";
import { CheckIcon, SendIcon, XIcon } from "@lucide/vue";
import { computed, ref } from "vue";

const props = defineProps<{ request: ExtensionUiRequest; busy?: boolean }>();
const emit = defineEmits<{ respond: [response: ExtensionUiResponse] }>();

const text = ref(props.request.prefill ?? "");

// Break the questionnaire's `N. Label — description` lines into rows so long
// options wrap instead of being clipped onto a single line.
const options = computed(() => props.request.options.map(parseOption));

const labels: Record<string, string> = {
  select: "Select an option",
  confirm: "Confirm",
  input: "Input required",
  editor: "Edit",
};
const dialogLabel = computed(
  () => labels[props.request.method] ?? props.request.method,
);

function respond(response: ExtensionUiResponse) {
  emit("respond", response);
}

function cancel() {
  respond({ cancelled: true });
}
</script>
