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
    CardTitle {{ $t("voice.title") }}
    CardDescription
      | {{ $t("voice.description") }}
  CardContent.flex.flex-col.gap-4
    .flex.items-center.justify-between.gap-4.rounded-md.border.p-3(class="bg-muted/30")
      .flex.flex-col(class="gap-0.5")
        span.text-sm.font-medium {{ $t("voice.enable") }}
        span.text-muted-foreground.text-xs
          | {{ $t("voice.enableHint") }}
      Switch(
        v-model="enabled"
        :aria-label="$t('voice.enable')"
        data-testid="voice-input-enabled"
      )

    template(v-if="enabled")
      .grid.gap-4(class="sm:grid-cols-2")
        Field
          FieldLabel(for="voice-input-base-url") {{ $t("voice.baseUrl") }}
          Input#voice-input-base-url(
            v-model="baseUrl"
            :placeholder="$t('voice.baseUrlPlaceholder')"
            autocomplete="off"
            data-testid="voice-input-base-url"
          )
          FieldDescription
            | {{ $t("voice.baseUrlHint2") }}
        Field
          FieldLabel(for="voice-input-model") {{ $t("domainLog.kind.model") }}
          Input#voice-input-model(
            v-model="model"
            :placeholder="$t('voice.modelPlaceholder')"
            autocomplete="off"
            data-testid="voice-input-model"
          )
          FieldDescription {{ $t("voice.modelHint") }}
        Field(class="sm:col-span-2")
          FieldLabel(for="voice-input-api-key") {{ $t("voice.apiKey") }}
          .flex.items-center.gap-2
            Input#voice-input-api-key.flex-1(
              v-model="apiKey"
              type="password"
              :placeholder="hasStoredKey ? $t('voice.apiKeyKeep') : $t('voice.apiKeyOptional')"
              autocomplete="new-password"
              data-testid="voice-input-api-key"
            )
            Badge(v-if="hasStoredKey" variant="secondary") {{ $t("voice.keyStored") }}
            Button(
              v-if="hasStoredKey"
              type="button"
              variant="outline"
              size="sm"
              :title="$t('voice.forgetKey')"
              @click="clearKey"
              data-testid="voice-input-clear-key"
            )
              Trash2Icon(data-icon="inline-start")
              span {{ $t("common.remove") }}
          FieldDescription
            | {{ $t("voice.keyHint") }}
        .flex.flex-col(class="gap-0.5 sm:col-span-2")
          span.text-sm.font-medium {{ $t("language.label") }}
          span.text-muted-foreground.text-xs
            | {{ $t("voice.languageHint") }}

      Alert
        AlertTitle {{ $t("voice.browserFallback") }}
        AlertDescription
          | {{ $t("voice.browserFallbackHint") }}

    p.text-destructive.text-sm(v-if="error") {{ error }}
    .flex.justify-end
      Button(
        type="button"
        @click="save"
        :disabled="saving || !dirty"
        data-testid="voice-input-save"
      )
        Spinner(v-if="saving" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ $t("common.save") }}
</template>

<script setup lang="ts">
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { useSettings } from "@/composables/useSettings";
import { CheckIcon, Trash2Icon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const { settings, loadSettings, saveSettings } = useSettings();

const enabled = ref(false);
const baseUrl = ref("");
const model = ref("whisper-1");
const apiKey = ref("");
const saving = ref(false);
const error = ref("");

const hasStoredKey = computed(
  () => settings.value?.voiceInput?.hasApiKey ?? false,
);

function fillFromSettings() {
  const voice = settings.value?.voiceInput;
  if (!voice) return;
  enabled.value = voice.enabled;
  baseUrl.value = voice.baseUrl;
  model.value = voice.model;
}

watch(
  () => settings.value?.voiceInput,
  () => fillFromSettings(),
);

onMounted(async () => {
  await loadSettings();
  fillFromSettings();
});

const dirty = computed(() => {
  const voice = settings.value?.voiceInput;
  if (!voice) return false;
  return (
    enabled.value !== voice.enabled ||
    baseUrl.value.trim() !== voice.baseUrl ||
    model.value.trim() !== voice.model ||
    apiKey.value !== ""
  );
});

async function clearKey() {
  const current = settings.value;
  if (!current || saving.value) return;

  const draft = {
    enabled: enabled.value,
    baseUrl: baseUrl.value,
    model: model.value,
    apiKey: apiKey.value,
  };
  saving.value = true;
  error.value = "";
  try {
    await saveSettings({
      agencyName: current.agencyName,
      primaryColor: current.primaryColor,
      notificationSoundSet: current.notificationSoundSet,
      clearApiKey: true,
    });
    toast.success(t("voice.toasts.keyRemoved"));
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    enabled.value = draft.enabled;
    baseUrl.value = draft.baseUrl;
    model.value = draft.model;
    apiKey.value = draft.apiKey;
    saving.value = false;
  }
}

async function save() {
  const current = settings.value;
  if (!current || saving.value) return;
  saving.value = true;
  error.value = "";
  try {
    await saveSettings({
      agencyName: current.agencyName,
      primaryColor: current.primaryColor,
      notificationSoundSet: current.notificationSoundSet,
      voiceInput: {
        enabled: enabled.value,
        baseUrl: baseUrl.value.trim(),
        model: model.value.trim(),
      },
      voiceInputApiKey: apiKey.value,
    });
    apiKey.value = "";
    fillFromSettings();
    toast.success(t("voice.toasts.saved"));
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  voice:
    title: "Voice input"
    description: "Dictate task prompts and follow-ups from the microphone in every Markdown composer. With a speech-to-text endpoint configured, the agency transcribes the recording itself; without one, supported browsers use their own Web Speech recognition."
    enable: "Enable voice input"
    enableHint: "Show the microphone in the task composer, prompt editor, follow-ups and comments."
    baseUrl: "API base URL"
    modelHint: "Any model the endpoint accepts."
    apiKey: "API key"
    keyStored: "Key stored"
    keyHint: "Stored by the agency and never returned to the browser. Leave blank to keep the current key; it is not cleared when the base URL changes."
    languageHint: "Detected automatically: no language is sent, so the provider decides. The browser fallback uses the browser's own language."
    browserFallback: "Browser fallback"
    browserFallbackHint: "Without a configured endpoint, Chrome, Edge and Safari dictate with their Web Speech API and send audio to the browser vendor. Firefox has no such API, so the microphone stays hidden there."
    baseUrlPlaceholder: "https://api.example.com/v1"
    modelPlaceholder: "whisper-1"
    forgetKey: "Forget the stored API key"
    toasts:
      saved: "Voice input settings saved"
      keyRemoved: "Stored API key removed"
    apiKeyKeep: "Leave blank to keep the stored key"
    apiKeyOptional: "Optional for a local endpoint"
    baseUrlHint2: "The agency appends /audio/transcriptions. Leave empty to use the browser recognizer only."
ru:
  voice:
    title: "Голосовой ввод"
    description: "Диктуйте промпты и уточнения с микрофона в любом Markdown-редакторе. Если настроен сервис распознавания речи, агентство расшифровывает запись само; иначе поддерживаемые браузеры используют собственное распознавание Web Speech."
    enable: "Включить голосовой ввод"
    enableHint: "Показывать микрофон в композере задач, редакторе промптов, уточнениях и комментариях."
    baseUrl: "Базовый URL API"
    modelHint: "Любая модель, которую принимает сервис."
    apiKey: "API-ключ"
    keyStored: "Ключ сохранён"
    keyHint: "Хранится в агентстве и никогда не возвращается в браузер. Оставьте пустым, чтобы сохранить текущий ключ; он не сбрасывается при изменении базового URL."
    languageHint: "Определяется автоматически: язык не отправляется, решение принимает провайдер. Резервный вариант в браузере использует язык браузера."
    browserFallback: "Резервный вариант в браузере"
    browserFallbackHint: "Без настроенного сервиса Chrome, Edge и Safari диктуют через Web Speech API и отправляют аудио вендору браузера. В Firefox такого API нет, поэтому микрофон там не показывается."
    baseUrlPlaceholder: "https://api.example.com/v1"
    modelPlaceholder: "whisper-1"
    forgetKey: "Забыть сохранённый API-ключ"
    toasts:
      saved: "Настройки голосового ввода сохранены"
      keyRemoved: "Сохранённый API-ключ удалён"
    apiKeyKeep: "Оставьте пустым, чтобы сохранить ключ"
    apiKeyOptional: "Необязательно для локального сервиса"
    baseUrlHint2: "Агентство добавляет /audio/transcriptions. Оставьте пустым, чтобы использовать только распознавание браузера."
</i18n>
