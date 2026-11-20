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
      CardTitle {{ $t("settings.tabs.general") }}
      CardDescription {{ $t("settings.general.description") }}
    CardContent.flex.flex-col.gap-6
      form.flex.flex-col.gap-4(@submit.prevent="save")
        FieldGroup
          Field
            FieldLabel(for="agency-name") {{ $t("settings.general.agencyName") }}
            Input#agency-name(
              v-model="name"
              placeholder="CherryPi"
              maxlength="100"
            )
          Field
            FieldLabel(for="agency-public-url") {{ $t("settings.general.publicUrl") }}
            Input#agency-public-url.font-mono.text-sm(
              readonly
              :model-value="settings?.publicUrl"
            )
            FieldDescription
              | {{ $t("settings.general.publicUrlHint") }}
          Field
            FieldLabel(for="primary-color") {{ $t("settings.general.primaryColor") }}
            .flex.items-center.gap-2
              input#primary-color.border-input.h-8.w-12.cursor-pointer.rounded-md.border.bg-transparent(
                type="color"
                :value="primary || DEFAULT_PRIMARY"
                @input="onPick"
              )
              Input.max-w-40.font-mono.text-sm(
                v-model="hexDraft"
                placeholder="#171717"
                @blur="commitHex"
                @keydown.enter.prevent="commitHex"
                @keydown.esc.prevent="cancelHex"
              )
              Button(
                type="button"
                variant="outline"
                size="sm"
                @click="resetColor"
              )
                RotateCcwIcon(data-icon="inline-start")
                span {{ $t("common.reset") }}
            FieldDescription {{ $t("settings.general.colorHint") }}
          Field
            FieldLabel {{ $t("settings.general.logo") }}
            PictureUpload(
              agency-kind="logo"
              fallback-image="/logo.png"
              shape="square"
              :picture="settings?.logo"
              @changed="onLogoChanged"
            )
            FieldDescription
              | {{ $t("settings.general.logoHint") }}
          Field
            FieldLabel {{ $t("settings.general.favicon") }}
            PictureUpload(
              agency-kind="favicon"
              shape="square"
              :fallback-image="settings?.logo?.url || '/logo.png'"
              :picture="settings?.favicon"
              @changed="onFaviconChanged"
            )
            FieldDescription {{ $t("settings.general.faviconHint") }}
        Separator
        .flex.flex-col.gap-4
          .flex.flex-col
            span.text-sm.font-medium {{ $t("settings.kpi.layouts.title") }}
            span.text-muted-foreground.text-xs
              | {{ $t("settings.kpi.layouts.description") }}
          KpiLayoutDefaultsPanel
        Separator
        .flex.flex-col.gap-3
          .flex.flex-col
            span.text-sm.font-medium {{ $t("settings.archive.title") }}
            span.text-muted-foreground.text-xs
              | {{ $t("settings.archive.description") }}
          .flex.items-center.justify-between.gap-3
            .flex.flex-col
              span.text-sm.font-medium {{ $t("settings.archive.enabled") }}
              span.text-muted-foreground.text-xs
                | {{ $t("settings.archive.enabledHint") }}
            Switch(
              :model-value="taskArchiveEnabled"
              @update:model-value="taskArchiveEnabled = Boolean($event)"
            )
          Field.max-w-xl
            FieldLabel(for="task-archive-days") {{ $t("settings.archive.days") }}
            Input#task-archive-days.max-w-32(
              v-model.number="taskArchiveDays"
              type="number"
              min="1"
              :max="MAX_TASK_ARCHIVE_DAYS"
              :disabled="!taskArchiveEnabled"
            )
            FieldDescription
              | {{ $t("settings.archive.daysHint") }}
        Separator
        .flex.flex-col.gap-3
          .flex.flex-col
            span.text-sm.font-medium {{ $t("settings.sounds.title") }}
            span.text-muted-foreground.text-xs
              | {{ $t("settings.sounds.description") }}
          .flex.flex-wrap.items-end.gap-3
            Field.flex-1
              FieldLabel(for="org-notification-sound-set") {{ $t("settings.sounds.set") }}
              Select(v-model="notificationSoundSet")
                SelectTrigger#org-notification-sound-set.w-full
                  SelectValue
                SelectContent
                  SelectItem(
                    v-for="set in SOUND_SETS"
                    :key="set.id"
                    :value="set.id"
                  ) {{ $t(set.nameKey) }}
              FieldDescription {{ $t(selectedSoundSet.descriptionKey) }}
            Button(
              type="button"
              variant="outline"
              size="sm"
              :aria-label="previewing ? $t('settings.sounds.stopAria') : $t('settings.sounds.previewAria')"
              @click="previewSound"
            )
              Spinner(v-if="previewing" data-icon="inline-start")
              PlayIcon(v-else data-icon="inline-start")
              span {{ previewing ? "Stop" : "Preview" }}
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end
          Button(type="submit" :disabled="saving || !taskArchiveDaysValid")
            Spinner(v-if="saving" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ $t("common.save") }}

      Separator

      .flex.flex-col.gap-2
        .flex.flex-col
          span.text-sm.font-medium {{ $t("settings.prompt.title") }}
          span.text-muted-foreground.text-xs
            | {{ $t("settings.prompt.description") }}
        PromptEditor(
          :placeholder="$t('settings.prompt.placeholder')"
          :scope="PromptScope.ORG"
        )

      Separator

      TaskColumnsPanel
</template>

<script setup lang="ts">
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import PromptEditor from "@/components/PromptEditor.vue";
import KpiLayoutDefaultsPanel from "@/components/settings/KpiLayoutDefaultsPanel.vue";
import TaskColumnsPanel from "@/components/settings/TaskColumnsPanel.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { withProgress } from "@/composables/useProgress";
import { useSettings } from "@/composables/useSettings";
import { useSoundPreferences } from "@/composables/useSoundPreferences";
import { ATTENTION_EVENTS } from "@/lib/attentionEvents";
import { previewSoundSet, stopSound } from "@/lib/sounds/play";
import { SOUND_SETS, soundSetById } from "@/lib/sounds/sets";
import { PromptScope, type Attachment } from "@cherrypi/sdk";
import { CheckIcon, PlayIcon, RotateCcwIcon } from "@lucide/vue";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const DEFAULT_PRIMARY = "#171717";

const { settings, error, loadSettings, saveSettings } = useSettings();

const name = ref("");
const primary = ref("");
const hexDraft = ref("");
const notificationSoundSet = ref("cherry");
const taskArchiveEnabled = ref(true);
const taskArchiveDays = ref(30);
const saving = ref(false);

const MAX_TASK_ARCHIVE_DAYS = 3650;
const taskArchiveDaysValid = computed(() => {
  const days = Number(taskArchiveDays.value);
  return Number.isInteger(days) && days >= 1 && days <= MAX_TASK_ARCHIVE_DAYS;
});

const { soundVolume } = useSoundPreferences();
const previewing = ref(false);
let previewToken = 0;
const selectedSoundSet = computed(() =>
  soundSetById(notificationSoundSet.value),
);

async function previewSound() {
  if (previewing.value) {
    stopSound();
    previewing.value = false;
    return;
  }
  previewing.value = true;
  const token = ++previewToken;
  await previewSoundSet(
    notificationSoundSet.value,
    soundVolume.value,
    ATTENTION_EVENTS.map((event) => event.kind),
  );
  if (token === previewToken) previewing.value = false;
}

watch(primary, (value) => {
  hexDraft.value = value;
});

onMounted(async () => {
  await loadSettings(true);
  name.value = settings.value?.agencyName ?? "";
  primary.value = settings.value?.primaryColor ?? "";
  notificationSoundSet.value = settings.value?.notificationSoundSet || "cherry";
  taskArchiveEnabled.value = settings.value?.taskAutoArchive?.enabled ?? true;
  taskArchiveDays.value = settings.value?.taskAutoArchive?.days || 30;
});

onUnmounted(() => {
  previewToken += 1;
  stopSound();
});

function onPick(event: Event) {
  primary.value = (event.target as HTMLInputElement).value;
}

function onLogoChanged(picture: Attachment | undefined) {
  if (settings.value) settings.value.logo = picture;
}

function onFaviconChanged(picture: Attachment | undefined) {
  if (settings.value) settings.value.favicon = picture;
}

function commitHex() {
  const raw = hexDraft.value.trim();
  if (raw === "") {
    primary.value = "";
    return;
  }

  const hex = (raw.startsWith("#") ? raw : `#${raw}`).toLowerCase();
  if (/^#[0-9a-f]{6}$/.test(hex)) {
    primary.value = hex;
  } else {
    hexDraft.value = primary.value;
  }
}

function cancelHex() {
  hexDraft.value = primary.value;
}

function resetColor() {
  primary.value = "";
  hexDraft.value = "";
}

async function save() {
  saving.value = true;
  try {
    await withProgress(() =>
      saveSettings({
        agencyName: name.value.trim(),
        primaryColor: primary.value,
        notificationSoundSet: notificationSoundSet.value,
        taskAutoArchive: {
          enabled: taskArchiveEnabled.value,
          days: Number(taskArchiveDays.value),
        },
      }),
    );
    toast.success("Settings saved");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  settings:
    sounds:
      title: "Notification sounds"
      description: "The sound set every browser in the agency uses for task notifications. Each user controls their own volume in User settings."
      set: "Sound set"
      preview: "Preview"
      stop: "Stop"
      previewAria: "Preview this sound set"
      stopAria: "Stop the sound preview"
    general:
      description: "The agency's identity, task columns, and organization prompt."
      agencyName: "Agency name"
      publicUrl: "Public URL"
      publicUrlHint: "The external origin from CHERRYPI_PUBLIC_URL. It is used for links, cookies, OIDC callbacks, and plugin templates; change it in the environment and restart the agency."
      primaryColor: "Primary color"
      colorHint: "Empty uses the built-in default; text contrast adapts automatically."
      logo: "Logo"
      logoHint: "Shown in the sidebar and used as the default favicon. Empty uses the bundled cherry artwork."
      favicon: "Favicon"
      faviconHint: "Defaults to the logo when empty."
    prompt:
      title: "Organization prompt"
      description: "Injected into every new task's system prompt, before the agent, project, and agent-project prompts. Edits affect future tasks only."
      placeholder: "Organization-wide instructions…"
    archive:
      title: "Automatic archive"
      description: "Completed and cancelled tasks leave the active lists after a while. Their agent-side sessions and workspaces are cleaned up when the archive lands."
      enabled: "Archive closed tasks automatically"
      enabledHint: "A project may override this policy for its own tasks."
      days: "Days after close-out"
      daysHint: "Between 1 and 3650 days. A project owner may set a shorter or longer delay, or keep its tasks forever."
    kpi:
      layouts:
        title: "Overview widget layouts"
        description: "The agency Tasks page and every project and agent Overview render a widget layout. Administrators arrange the Tasks page; a project or agent owner arranges that page. Project and agent pages without their own layout follow the organization default, while the Tasks page starts empty. The Usage and Concurrency tabs keep their fixed strips."
ru:
  settings:
    sounds:
      title: "Звуки уведомлений"
      description: "Набор звуков для уведомлений о задачах во всех браузерах агентства. Каждый пользователь настраивает громкость в настройках пользователя."
      set: "Набор звуков"
      preview: "Прослушать"
      stop: "Остановить"
      previewAria: "Прослушать этот набор звуков"
      stopAria: "Остановить прослушивание"
    general:
      description: "Идентичность агентства, колонки задач и общий промпт."
      agencyName: "Название агентства"
      publicUrl: "Публичный URL"
      publicUrlHint: "Внешний origin из CHERRYPI_PUBLIC_URL. Он используется в ссылках, cookie, OIDC callback и шаблонах плагинов; измените его в окружении и перезапустите агентство."
      primaryColor: "Основной цвет"
      colorHint: "Пустое значение использует встроенный цвет по умолчанию; контраст текста подстраивается автоматически."
      logo: "Логотип"
      logoHint: "Показывается в боковой панели и используется как favicon по умолчанию. Пустое значение использует встроенный рисунок вишни."
      favicon: "Favicon"
      faviconHint: "Если пусто, используется логотип."
    prompt:
      title: "Общий промпт"
      description: "Добавляется в системный промпт каждой новой задачи, перед промптами агента, проекта и агента в проекте. Изменения влияют только на будущие задачи."
      placeholder: "Инструкции для всего агентства…"
    archive:
      title: "Автоматический архив"
      description: "Завершённые и отменённые задачи со временем покидают активные списки. Их сессии и рабочие пространства на агентах очищаются при архивации."
      enabled: "Автоматически архивировать закрытые задачи"
      enabledHint: "Проект может переопределить это правило для своих задач."
      days: "Дней после закрытия"
      daysHint: "От 1 до 3650 дней. Владелец проекта может задать другой срок или хранить задачи бессрочно."
    kpi:
      layouts:
        title: "Макеты виджетов обзора"
        description: "На странице задач агентства и в каждом обзоре проекта и агента показывается макет виджетов. Страницу задач настраивают администраторы; страницу проекта или агента — её владелец. Страницы проекта и агента без собственного макета следуют значению организации по умолчанию, а страница задач начинается пустой. Вкладки «Использование» и «Параллельность» сохраняют свои фиксированные сводки."
</i18n>
