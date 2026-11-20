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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent.overflow-y-auto(class="max-h-[90vh] sm:max-w-md")
    DialogHeader
      DialogTitle {{ variable ? $t("variableDialog.editTitle") : $t("variableDialog.createTitle") }}
      DialogDescription
        | {{ variable ? $t("variableDialog.editHint") : $t("variableDialog.createHint", { scope: scopeLabel }) }}
    form.flex.flex-col.gap-4(@submit.prevent="save")
      Field
        FieldLabel(for="variable-name") {{ $t("variables.columnName") }}
        Input#variable-name(
          v-model="name"
          :placeholder="$t('variableDialog.namePlaceholder')"
          autocomplete="off"
          spellcheck="false"
          :disabled="Boolean(variable)"
          data-test="variable-name"
        )
        FieldDescription {{ $t("variableDialog.nameHint") }}

      Field
        FieldLabel(for="variable-value") {{ $t("variables.columnValue") }}
        Textarea#variable-value(
          v-model="value"
          :placeholder="valuePlaceholder"
          rows="3"
          autocomplete="off"
          spellcheck="false"
          data-test="variable-value"
        )
        FieldDescription
          | {{ variable ? $t("variableDialog.valueKeepHint") : $t("variableDialog.valueHint") }}

      Field
        FieldLabel(for="variable-note") {{ $t("variables.columnNote") }}
        Input#variable-note(
          v-model="note"
          :placeholder="$t('common.optional')"
          autocomplete="off"
        )
        FieldDescription {{ $t("variableDialog.noteHint") }}

      Field(v-if="!variable" orientation="horizontal")
        Switch#variable-secret(
          v-model="secret"
          :disabled="!secretsEnabled"
          data-test="variable-secret"
        )
        FieldContent
          FieldLabel(for="variable-secret") {{ $t("variableDialog.secret") }}
          FieldDescription
            | {{ secretsEnabled ? $t("variableDialog.secretHint") : $t("variables.noKeyHint") }}

      p.text-destructive.text-sm(v-if="error") {{ error }}

      DialogFooter
        Button(
          type="button"
          variant="outline"
          @click="emit('update:open', false)"
        )
          | {{ $t("common.cancel") }}
        Button(type="submit" :disabled="saving || !valid")
          Spinner(v-if="saving" data-icon="inline-start")
          span {{ $t("common.save") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { withProgress } from "@/composables/useProgress";
import { useVariables } from "@/composables/useVariables";
import type { Variable, VariableTargetInput } from "@cherrypi/sdk";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const props = defineProps<{
  open: boolean;
  variable: Variable | null;
  secretsEnabled: boolean;
  scopeLabel: string;
  target: VariableTargetInput;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  saved: [];
}>();

const { createVariable, updateVariable } = useVariables(() => props.target);

const name = ref("");
const value = ref("");
const note = ref("");
const secret = ref(false);
const saving = ref(false);
const error = ref("");

const namePattern = /^[A-Za-z_][A-Za-z0-9_]{0,127}$/;

const valid = computed(() => {
  if (props.variable) return true;
  return namePattern.test(name.value) && value.value.length > 0;
});

const valuePlaceholder = computed(() =>
  props.variable
    ? t("variableDialog.valueKeepPlaceholder")
    : t("variableDialog.valuePlaceholder"),
);

watch(
  () => [props.open, props.variable] as const,
  () => {
    if (!props.open) return;
    error.value = "";
    if (props.variable) {
      name.value = props.variable.name;
      value.value = "";
      note.value = props.variable.note;
      secret.value = props.variable.secret;
      return;
    }
    name.value = "";
    value.value = "";
    note.value = "";
    secret.value = false;
  },
  { immediate: true },
);

async function save() {
  if (!valid.value || saving.value) return;
  saving.value = true;
  error.value = "";
  try {
    if (props.variable) {
      await withProgress(() =>
        updateVariable(props.variable!.name, {
          note: note.value,

          value: value.value === "" ? undefined : value.value,
        }),
      );
    } else {
      await withProgress(() =>
        createVariable({
          name: name.value.trim(),
          value: value.value,
          note: note.value,
          secret: secret.value,
        }),
      );
    }
    toast.success(
      props.variable
        ? t("variableDialog.saved", { name: props.variable.name })
        : t("variableDialog.created", { name: name.value.trim() }),
    );
    emit("update:open", false);
    emit("saved");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  variableDialog:
    createTitle: Add variable
    editTitle: Edit variable
    createHint: Defined at {scope}.
    editHint: Changing the value replaces it; the old one cannot be recovered.
    namePlaceholder: ACME_TOKEN
    nameHint: Letters, digits and underscores; it becomes an environment variable name.
    valuePlaceholder: The value
    valueKeepPlaceholder: Leave empty to keep the current value
    valueHint: A secret's value is sealed and never shown again.
    valueKeepHint: Leave empty to keep the current value.
    noteHint: Shown to the model next to the name, never to a task's shell.
    secret: Store as a secret
    secretHint: The value is write-only. Only the task's shell receives it.
    created: Added {name}
    saved: Saved {name}
ru:
  variableDialog:
    createTitle: Добавить переменную
    editTitle: Изменить переменную
    createHint: Определяется на уровне {scope}.
    editHint: Новое значение заменяет старое без возможности восстановления.
    namePlaceholder: ACME_TOKEN
    nameHint: Буквы, цифры и подчёркивания; имя станет переменной окружения.
    valuePlaceholder: Значение
    valueKeepPlaceholder: Оставьте пустым, чтобы сохранить текущее значение
    valueHint: Значение секрета шифруется и больше не показывается.
    valueKeepHint: Оставьте пустым, чтобы сохранить текущее значение.
    noteHint: Показывается модели рядом с именем и никогда не попадает в оболочку задачи.
    secret: Хранить как секрет
    secretHint: Значение доступно только на запись. Его получает только оболочка задачи.
    created: Добавлена {name}
    saved: Сохранена {name}
</i18n>
