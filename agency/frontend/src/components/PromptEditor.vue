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
.flex.flex-col.gap-2
  .flex.flex-wrap.items-center.justify-between.gap-2
    span.text-muted-foreground.text-xs {{ $t("promptEditor.version", { version }) }}
    .flex.items-center.gap-2
      Button(
        type="button"
        variant="outline"
        size="sm"
        @click="historyOpen = true"
        :disabled="loading"
      )
        HistoryIcon(data-icon="inline-start")
        span {{ $t("promptEditor.history") }}
      Button(
        v-if="canEdit"
        type="button"
        @click="saveClick"
        :disabled="saving || loading"
      )
        Spinner(v-if="saving" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ $t("common.save") }}
  Textarea.field-sizing-fixed.resize-y(
    v-model="content"
    :placeholder="placeholder"
    :disabled="loading || !canEdit"
  )
  Input(
    v-if="canEdit"
    v-model="note"
    :placeholder="$t('promptEditor.notePlaceholder')"
  )
  p.text-muted-foreground.text-xs(v-if="!canEdit && !loading")
    | {{ $t("promptEditor.ownersOnly") }}
  p.text-destructive.text-sm(v-if="error") {{ error }}
  PromptProposals(
    v-if="!loading && supportsProposals"
    :scope="scope"
    :target="target"
    :current-content="content"
    :current-version="version"
    :can-edit="canEdit"
    :can-propose="canPropose"
    :can-approve="canApprove"
    @changed="load"
  )
  PromptHistoryDialog(
    v-model:open="historyOpen"
    :scope="scope"
    :target="target"
    :current-version="version"
  )
</template>

<script setup lang="ts">
import PromptHistoryDialog from "@/components/PromptHistoryDialog.vue";
import PromptProposals from "@/components/PromptProposals.vue";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { usePrompt } from "@/composables/usePrompt";
import { PromptScope, type PromptTarget } from "@cherrypi/sdk";
import { CheckIcon, HistoryIcon } from "@lucide/vue";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = defineProps<{
  scope: PromptScope;
  agentId?: string;
  teamId?: string;
  projectId?: string;
  placeholder?: string;
}>();

const target = computed<PromptTarget>(() => ({
  agentId: props.agentId ?? "",
  teamId: props.teamId ?? "",
  projectId: props.projectId ?? "",
}));

const {
  content,
  version,
  loading,
  saving,
  error,
  canEdit,
  canPropose,
  canApprove,
  load,
  save,
} = usePrompt(props.scope, () => target.value);

const note = ref("");
const historyOpen = ref(false);
const supportsProposals = computed(() =>
  [PromptScope.AGENT, PromptScope.AGENT_PROJECT, PromptScope.PROJECT].includes(
    props.scope,
  ),
);

async function saveClick() {
  if (await save(note.value)) {
    note.value = "";
    toast.success(t("promptEditor.toasts.saved"));
  }
}

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  promptEditor:
    history: "History"
    ownersOnly: "Only this scope's owners may edit the prompt directly."
    notePlaceholder: "Change note (optional)"
    toasts:
      saved: "Prompt saved"
    version: "Version {version}"
ru:
  promptEditor:
    history: "История"
    ownersOnly: "Напрямую редактировать промпт могут только владельцы этой области."
    notePlaceholder: "Заметка об изменении (необязательно)"
    toasts:
      saved: "Промпт сохранён"
    version: "Версия {version}"
</i18n>
