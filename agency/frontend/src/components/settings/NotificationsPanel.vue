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
    CardTitle {{ $t("notifications.title") }}
    CardDescription
      | {{ $t("notifications.description") }}
  CardContent.flex.flex-col.gap-4
    .flex.items-center.justify-between.gap-3.rounded-md.border.p-3(class="bg-muted/30")
      .flex.flex-col(class="gap-0.5")
        span.text-sm.font-medium {{ $t("notifications.desktop.title") }}
        span.text-muted-foreground.text-xs {{ permissionHint }}
      Button(
        v-if="permission === 'default'"
        variant="outline"
        size="sm"
        @click="enablePermission"
        :disabled="requesting"
      )
        Spinner(v-if="requesting" data-icon="inline-start")
        BellIcon(v-else data-icon="inline-start")
        span {{ $t("common.enable") }}
    .flex.flex-col.gap-3.rounded-md.border.p-3(class="bg-muted/30")
      .flex.flex-wrap.items-end.gap-3
        Field.flex-1
          FieldLabel {{ $t("settings.sounds.set") }}
          .flex.items-center.gap-2
            span.text-sm.font-medium {{ $t(selectedSet.nameKey) }}
            Badge(variant="secondary") {{ $t("notifications.organization") }}
          FieldDescription
            | {{ $t(selectedSet.descriptionKey) }} {{ $t("notifications.organizationHint") }}
            RouterLink.text-underline(
              v-if="isAdmin"
              :to="{ name: 'settings', query: { tab: 'general' } }"
            ) {{ $t("notifications.changeInSettings") }}
        Button(
          variant="outline"
          size="sm"
          :aria-label="previewing ? $t('settings.sounds.stopAria') : $t('settings.sounds.previewAria')"
          @click="previewSet"
        )
          Spinner(v-if="previewing" data-icon="inline-start")
          PlayIcon(v-else data-icon="inline-start")
          span {{ previewing ? $t("settings.sounds.stop") : $t("settings.sounds.preview") }}
      .flex.items-center.gap-3
        span.text-sm.font-medium {{ $t("notifications.volume") }}
        input#notification-volume.accent-primary.h-2.flex-1.cursor-pointer(
          type="range"
          :value="volumePercent"
          min="0"
          max="100"
          step="1"
          @input="onVolumeInput"
          @change="onVolumeChange"
        )
        span.text-muted-foreground.w-10.text-right.text-xs
          | {{ volumePercent }}%
    p.text-destructive.text-sm(v-if="inboxError") {{ inboxError }}
    .overflow-hidden.rounded-md.border
      Table
        TableHeader
          TableRow
            TableHead {{ $t("notifications.column.event") }}
            TableHead.text-center {{ $t("nav.inbox") }}
            TableHead.text-center {{ $t("notifications.column.toast") }}
            TableHead.text-center {{ $t("notifications.column.sound") }}
            TableHead.text-center {{ $t("notifications.column.desktop") }}
        TableBody
          TableRow(v-for="event in ATTENTION_EVENTS" :key="event.kind")
            TableCell.text-sm {{ $t(event.labelKey) }}
            TableCell.text-center
              .flex.justify-center
                Switch(
                  :model-value="!disabledKinds.includes(event.kind)"
                  :aria-label="$t('notifications.aria.inbox', { event: $t(event.labelKey) })"
                  @update:model-value="(value) => toggleInbox(event.kind, value)"
                )
            TableCell.text-center
              .flex.items-center.justify-center.gap-1
                Switch(
                  :model-value="preferences[event.kind]?.toast ?? false"
                  :aria-label="$t('notifications.aria.toast', { event: $t(event.labelKey) })"
                  @update:model-value="(value) => setPreference(event.kind, 'toast', value)"
                )
            TableCell.text-center
              .flex.justify-center
                Switch(
                  :model-value="preferences[event.kind]?.sound ?? false"
                  :aria-label="$t('notifications.aria.sound', { event: $t(event.labelKey) })"
                  @update:model-value="(value) => setPreference(event.kind, 'sound', value)"
                )
                Button(
                  variant="ghost"
                  size="icon-xs"
                  :aria-label="soundAria(event)"
                  :title="playingKind === event.kind ? $t('notifications.stopPreview') : $t('notifications.previewHint')"
                  @click="playEventSound(event.kind)"
                )
                  SquareIcon(v-if="playingKind === event.kind")
                  PlayIcon(v-else)
            TableCell.text-center
              .flex.justify-center
                Switch(
                  :model-value="preferences[event.kind]?.browser ?? false"
                  :aria-label="$t('notifications.aria.desktop', { event: $t(event.labelKey) })"
                  @update:model-value="(value) => toggleDesktop(event.kind, value)"
                  :disabled="permission === 'unsupported' || permission === 'denied'"
                )
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
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAttentionPreferences } from "@/composables/useAttentionPreferences";
import { useSoundPreferences } from "@/composables/useSoundPreferences";
import { currentAccount } from "@/lib/account";
import {
  attentionPermission,
  requestAttentionPermission,
} from "@/lib/attention";
import { ATTENTION_EVENTS } from "@/lib/attentionEvents";
import { agency } from "@/lib/sdk";
import { playSound, previewSoundSet, stopSound } from "@/lib/sounds/play";
import { soundSetById } from "@/lib/sounds/sets";
import { BellIcon, PlayIcon, SquareIcon } from "@lucide/vue";
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const { preferences, setPreference } = useAttentionPreferences();
const {
  soundSet,
  soundVolume,
  load: loadSoundPreferences,
  setSoundVolumePreview,
  saveSoundVolume,
} = useSoundPreferences();
const isAdmin = computed(() => currentAccount.value?.is_admin === true);

const permission = ref(attentionPermission());
const requesting = ref(false);

const disabledKinds = ref<string[]>([]);
const inboxError = ref("");

const previewing = ref(false);
let previewToken = 0;

const playingKind = ref("");
let playToken = 0;

const selectedSet = computed(() => soundSetById(soundSet.value));
const volumePercent = computed(() => Math.round(soundVolume.value * 100));

onMounted(() => {
  void loadSoundPreferences(true);
  void loadInboxPreferences();
});

onUnmounted(() => {
  playToken += 1;
  previewToken += 1;
  stopSound();
});

function onVolumeInput(event: Event) {
  const value = Number((event.target as HTMLInputElement).value);
  setSoundVolumePreview(value / 100);
}

function onVolumeChange() {
  saveSoundVolume();
}

async function previewSet() {
  if (previewing.value) {
    stopSound();
    previewing.value = false;
    return;
  }
  previewing.value = true;
  const token = ++previewToken;
  await previewSoundSet(
    soundSet.value,
    soundVolume.value,
    ATTENTION_EVENTS.map((event) => event.kind),
  );
  if (token === previewToken) previewing.value = false;
}

async function playEventSound(kind: string) {
  if (playingKind.value === kind) {
    stopSound();
    playingKind.value = "";
    return;
  }

  previewToken += 1;
  previewing.value = false;
  const token = ++playToken;
  playingKind.value = kind;
  await playSound(soundSet.value, kind, soundVolume.value);
  if (token === playToken) playingKind.value = "";
}

async function loadInboxPreferences() {
  try {
    disabledKinds.value = await agency.getInboxPreferences();
    inboxError.value = "";
  } catch (err) {
    inboxError.value =
      err instanceof Error ? err.message : t("notifications.errors.inbox");
  }
}

function soundAria(event: { kind: string; labelKey: string }): string {
  const name = t(event.labelKey);
  return playingKind.value === event.kind
    ? t("notifications.aria.stopSound", { event: name })
    : t("notifications.aria.playSound", { event: name });
}

const permissionHint = computed(() => {
  switch (permission.value) {
    case "granted":
      return t("notifications.permission.granted");
    case "denied":
      return t("notifications.permission.denied");
    case "unsupported":
      return t("notifications.permission.unsupported");
    default:
      return t("notifications.permission.default");
  }
});

async function enablePermission(): Promise<
  NotificationPermission | "unsupported"
> {
  if (requesting.value) return permission.value;
  requesting.value = true;
  try {
    permission.value = await requestAttentionPermission();
    return permission.value;
  } finally {
    requesting.value = false;
  }
}

async function toggleDesktop(kind: string, value: boolean) {
  if (value && permission.value !== "granted") {
    const result = await enablePermission();
    if (result !== "granted") return;
  }
  setPreference(kind, "browser", value);
}

async function toggleInbox(kind: string, enabled: boolean) {
  const next = enabled
    ? disabledKinds.value.filter((item) => item !== kind)
    : [...disabledKinds.value, kind];
  const previous = disabledKinds.value;
  disabledKinds.value = next;
  try {
    disabledKinds.value = await agency.setInboxPreferences(next);
    inboxError.value = "";
  } catch (err) {
    disabledKinds.value = previous;
    inboxError.value = err instanceof Error ? err.message : String(err);
  }
}
</script>

<i18n lang="yaml">
en:
  notifications:
    title: "Browser notifications"
    description: "Choose which events are stored in the inbox, pop an in-app toast, play a sound, and show a desktop notification. The inbox is stored by the agency, the sound set is organization-wide, and the volume, per-event switches, and desktop alerts are per-browser."
    desktop:
      title: "Desktop notifications"
    organization: "Organization"
    organizationHint: "An administrator picks the set for the whole agency."
    changeInSettings: "Change in organization settings"
    volume: "Volume"
    column:
      event: "Event"
      toast: "Toast"
      sound: "Play Sound"
      desktop: "Desktop Notification"
    aria:
      inbox: "Store in inbox: {event}"
      toast: "Show a toast for: {event}"
      sound: "Play a sound for: {event}"
      playSound: "Play the {event} sound"
      stopSound: "Stop the {event} sound"
      desktop: "Desktop notification for: {event}"
    stopPreview: "Stop the sound"
    previewHint: "Preview the sound for this event"
    permission:
      granted: "Allowed for this browser."
      denied: "Blocked in your browser settings; re-allow notifications for this site."
      unsupported: "This browser does not support desktop notifications."
      default: "Allow your browser to show task alerts on screen."
    errors:
      inbox: "Could not save the inbox preference"
  attention:
    event:
      taskStarted: "Task started"
      needsInput: "Task needs input"
      settled: "Task settled"
      failed: "Task failed"
      completed: "Task completed"
      cancelled: "Task cancelled"
      teamInvitation: "Team invitation"
      comment: "New task comment"
      promptProposal: "Prompt proposal"
      mention: "You were mentioned"
      groupMemberRequested: "Group membership request"
      groupMemberApproved: "Group membership approved"
      groupMemberDeclined: "Group membership declined"
      groupMemberRemoved: "Removed from a group"
      groupMemberLeft: "Agent left a group"
      groupAccessInvited: "Group invited to a project or team"
      groupAccessRemoved: "Group lost project or team access"
      groupAccessMemberAdded: "A group member gained access"
ru:
  notifications:
    title: "Уведомления в браузере"
    description: "Выберите, какие события попадают в уведомления, показываются всплывающим сообщением, сопровождаются звуком и уведомлением рабочего стола. Список уведомлений хранится в агентстве, набор звуков общий для организации, а громкость, переключатели событий и уведомления рабочего стола задаются для каждого браузера."
    desktop:
      title: "Уведомления рабочего стола"
    organization: "Организация"
    organizationHint: "Администратор выбирает набор для всего агентства."
    changeInSettings: "Изменить в настройках организации"
    volume: "Громкость"
    column:
      event: "Событие"
      toast: "Всплывающее"
      sound: "Звук"
      desktop: "Рабочий стол"
    aria:
      inbox: "Хранить в уведомлениях: {event}"
      toast: "Показывать всплывающее для: {event}"
      sound: "Проигрывать звук для: {event}"
      playSound: "Проиграть звук {event}"
      stopSound: "Остановить звук {event}"
      desktop: "Уведомление рабочего стола для: {event}"
    stopPreview: "Остановить звук"
    previewHint: "Прослушать звук этого события"
    permission:
      granted: "Разрешено для этого браузера."
      denied: "Заблокировано в настройках браузера; снова разрешите уведомления для этого сайта."
      unsupported: "Этот браузер не поддерживает уведомления рабочего стола."
      default: "Разрешите браузеру показывать оповещения о задачах на экране."
    errors:
      inbox: "Не удалось сохранить настройку уведомлений"
  attention:
    event:
      taskStarted: "Задача запущена"
      needsInput: "Задаче нужен ввод"
      settled: "Задача завершена"
      failed: "Задача завершилась ошибкой"
      completed: "Задача выполнена"
      cancelled: "Задача отменена"
      teamInvitation: "Приглашение в команду"
      comment: "Новый комментарий к задаче"
      promptProposal: "Предложение промпта"
      mention: "Вас упомянули"
      groupMemberRequested: "Запрос на участие в группе"
      groupMemberApproved: "Участие в группе одобрено"
      groupMemberDeclined: "Участие в группе отклонено"
      groupMemberRemoved: "Удаление из группы"
      groupMemberLeft: "Агент покинул группу"
      groupAccessInvited: "Группа приглашена в проект или команду"
      groupAccessRemoved: "Группа потеряла доступ к проекту или команде"
      groupAccessMemberAdded: "Участник группы получил доступ"
</i18n>
