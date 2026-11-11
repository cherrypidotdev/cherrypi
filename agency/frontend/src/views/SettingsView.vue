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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  .flex.flex-col.gap-1
    h1.text-lg.font-semibold {{ $t("nav.orgSettings") }}
    p.text-muted-foreground.text-sm {{ $t("settings.subtitle") }}

  Card(v-if="!isAdmin")
    CardHeader
      CardTitle {{ $t("settings.adminOnly.title") }}
      CardDescription
        | {{ $t("settings.adminOnly.description") }}
  template(v-else)
    Tabs(v-model="tab")
      TabsList
        TabsTrigger(value="general") {{ $t("settings.tabs.general") }}
        TabsTrigger(value="plugins") {{ $t("account.tabs.plugins") }}
        TabsTrigger(value="auth") {{ $t("account.tabs.auth") }}
        TabsTrigger(value="integrations") {{ $t("settings.tabs.integrations") }}
        TabsTrigger(value="admin") {{ $t("settings.tabs.admin") }}
        TabsTrigger(value="logs") {{ $t("settings.tabs.logs") }}

      TabsContent(value="general")
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
                  Button(type="submit" :disabled="saving")
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

      TabsContent(value="plugins")
        PluginsPanel

      TabsContent(value="auth")
        .flex.flex-col.gap-6
          AuthProvidersPanel
          SMTPSettingsPanel

      TabsContent(value="integrations")
        .flex.flex-col.gap-6
          VoiceInputPanel
          Alert
            AlertTitle {{ $t("settings.webhooks.title") }}
            AlertDescription
              | {{ $t("settings.webhooks.description") }}
          WebhooksPanel

      TabsContent(value="admin")
        AdminPanel

      TabsContent(value="logs")
        ProcessLogsPanel
</template>

<script setup lang="ts">
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import PromptEditor from "@/components/PromptEditor.vue";
import AdminPanel from "@/components/settings/AdminPanel.vue";
import AuthProvidersPanel from "@/components/settings/AuthProvidersPanel.vue";
import KpiLayoutDefaultsPanel from "@/components/settings/KpiLayoutDefaultsPanel.vue";
import PluginsPanel from "@/components/settings/PluginsPanel.vue";
import ProcessLogsPanel from "@/components/settings/ProcessLogsPanel.vue";
import SMTPSettingsPanel from "@/components/settings/SMTPSettingsPanel.vue";
import TaskColumnsPanel from "@/components/settings/TaskColumnsPanel.vue";
import VoiceInputPanel from "@/components/settings/VoiceInputPanel.vue";
import WebhooksPanel from "@/components/settings/WebhooksPanel.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { withProgress } from "@/composables/useProgress";
import { useSettings } from "@/composables/useSettings";
import { useSoundPreferences } from "@/composables/useSoundPreferences";
import { currentAccount } from "@/lib/account";
import { ATTENTION_EVENTS } from "@/lib/attentionEvents";
import { previewSoundSet, stopSound } from "@/lib/sounds/play";
import { SOUND_SETS, soundSetById } from "@/lib/sounds/sets";
import { PromptScope, type Attachment } from "@cherrypi/sdk";
import { CheckIcon, PlayIcon, RotateCcwIcon } from "@lucide/vue";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { toast } from "vue3-toastify";

const DEFAULT_PRIMARY = "#171717";

const { settings, error, loadSettings, saveSettings } = useSettings();
const isAdmin = computed(() => currentAccount.value?.is_admin ?? false);
const route = useRoute();

const tab = ref(
  typeof route.query.tab === "string" ? route.query.tab : "general",
);

watch(
  () => route.query.tab,
  (value) => {
    if (typeof value === "string") tab.value = value;
  },
);
const name = ref("");
const primary = ref("");
const hexDraft = ref("");
const notificationSoundSet = ref("cherry");
const saving = ref(false);

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
  if (!isAdmin.value) return;
  await loadSettings(true);
  name.value = settings.value?.agencyName ?? "";
  primary.value = settings.value?.primaryColor ?? "";
  notificationSoundSet.value = settings.value?.notificationSoundSet || "cherry";
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
    tabs:
      general: "General"
      integrations: "Integrations"
      admin: "Admin"
      logs: "Logs"
    sounds:
      title: "Notification sounds"
      description: "The sound set every browser in the agency uses for task notifications. Each user controls their own volume in User settings."
      set: "Sound set"
      preview: "Preview"
      stop: "Stop"
      previewAria: "Preview this sound set"
      stopAria: "Stop the sound preview"
    webhooks:
      title: "Webhooks"
      description: "Org-wide webhooks are being simplified: they will fire on every agency event and no longer offer an event picker. That change is tracked in CHPI-176."
    subtitle: "Agency-wide configuration. Only administrators can open this page."
    adminOnly:
      title: "Administrators only"
      description: "Ask an administrator if you need an agency-wide setting changed."
    general:
      description: "The agency's identity, task columns, and organization prompt."
      agencyName: "Agency name"
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
    kpi:
      layouts:
        title: "Overview KPI layouts"
        description: "The agency Tasks page and every project and agent Overview render a KPI widget layout. Administrators arrange the Tasks page; a project or agent owner arranges that page. A page nobody has arranged shows no KPIs, and the Usage and Concurrency tabs keep their fixed strips."
ru:
  settings:
    tabs:
      general: "Общие"
      integrations: "Интеграции"
      admin: "Администрирование"
      logs: "Журналы"
    sounds:
      title: "Звуки уведомлений"
      description: "Набор звуков для уведомлений о задачах во всех браузерах агентства. Каждый пользователь настраивает громкость в настройках пользователя."
      set: "Набор звуков"
      preview: "Прослушать"
      stop: "Остановить"
      previewAria: "Прослушать этот набор звуков"
      stopAria: "Остановить прослушивание"
    webhooks:
      title: "Вебхуки"
      description: "Вебхуки агентства упрощаются: они будут срабатывать на каждое событие агентства, а выбор событий исчезнет. Это изменение отслеживается в CHPI-176."
    subtitle: "Настройки агентства. Эту страницу могут открывать только администраторы."
    adminOnly:
      title: "Только для администраторов"
      description: "Попросите администратора изменить настройку агентства."
    general:
      description: "Идентичность агентства, колонки задач и общий промпт."
      agencyName: "Название агентства"
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
    kpi:
      layouts:
        title: "Макеты KPI в обзоре"
        description: "На странице задач агентства и в каждом обзоре проекта и агента показывается макет виджетов KPI. Страницу задач настраивают администраторы; страницу проекта или агента — её владелец. Страница без настроенного макета не показывает KPI, а вкладки «Использование» и «Параллельность» сохраняют свои фиксированные сводки."
</i18n>
