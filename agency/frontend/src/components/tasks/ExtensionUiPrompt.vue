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

    CardDescription.whitespace-pre-wrap(
      v-if="blockedReason"
      data-test="dialog-blocked"
    ) {{ blockedReason }}
  CardContent
    template(v-if="request.method === 'select'")
      .flex.flex-col.gap-2
        Button.h-auto.justify-start.py-2.text-left.whitespace-normal(
          v-for="option in options"
          :key="option.raw"
          variant="outline"
          @click="respond({ value: option.raw })"
          :disabled="disabled"
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
        Button(@click="respond({ confirmed: true })" :disabled="disabled")
          CheckIcon(data-icon="inline-start")
          span {{ $t("common.confirm") }}
        Button(
          variant="outline"
          @click="respond({ confirmed: false })"
          :disabled="disabled"
        )
          XIcon(data-icon="inline-start")
          span {{ $t("extensionUi.decline") }}
    template(v-else-if="request.method === 'input'")
      form.flex.flex-col.gap-2(@submit.prevent="respond({ value: text })")
        Input(
          v-model="text"
          :placeholder="request.placeholder || ''"
          :disabled="disabled"
        )
        .flex.gap-2
          Button(type="submit" :disabled="disabled")
            SendIcon(data-icon="inline-start")
            span {{ $t("extensionUi.submit") }}
          Button(
            type="button"
            variant="outline"
            @click="cancel"
            :disabled="disabled"
          )
            XIcon(data-icon="inline-start")
            span {{ $t("common.cancel") }}
    template(v-else-if="request.method === 'editor'")
      form.flex.flex-col.gap-2(@submit.prevent="respond({ value: text })")
        Textarea(v-model="text" rows="8" :disabled="disabled")
        .flex.gap-2
          Button(type="submit" :disabled="disabled")
            CheckIcon(data-icon="inline-start")
            span {{ $t("common.save") }}
          Button(
            type="button"
            variant="outline"
            @click="cancel"
            :disabled="disabled"
          )
            XIcon(data-icon="inline-start")
            span {{ $t("common.cancel") }}
  CardFooter.justify-end.border-t(v-if="!blockedReason")
    Button(variant="ghost" size="sm" @click="cancel" :disabled="disabled")
      XIcon(data-icon="inline-start")
      span {{ $t("extensionUi.decline") }}
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
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    request: ExtensionUiRequest;
    busy?: boolean;

    blockedReason?: string;
  }>(),
  { busy: false, blockedReason: undefined },
);
const emit = defineEmits<{ respond: [response: ExtensionUiResponse] }>();

const text = ref(props.request.prefill ?? "");

const disabled = computed(
  () => props.busy || props.blockedReason !== undefined,
);

const options = computed(() => props.request.options.map(parseOption));

// i18n:dynamic — the dialog title is picked from this map by request method,

const labels: Record<string, string> = {
  select: "extensionUi.selectOption",
  confirm: "common.confirm",
  input: "extensionUi.inputRequired",
  editor: "common.edit",
};
const dialogLabel = computed(() => {
  const key = labels[props.request.method];
  return key ? t(key) : props.request.method;
});

function respond(response: ExtensionUiResponse) {
  emit("respond", response);
}

function cancel() {
  respond({ cancelled: true });
}
</script>

<i18n lang="yaml">
en:
  extensionUi:
    decline: "Decline"
    submit: "Submit"
    selectOption: "Select an option"
    inputRequired: "Input required"
ru:
  extensionUi:
    decline: "Отклонить"
    submit: "Отправить"
    selectOption: "Выберите вариант"
    inputRequired: "Требуется ввод"
</i18n>
