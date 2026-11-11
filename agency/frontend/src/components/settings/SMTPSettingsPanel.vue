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
    CardTitle {{ $t("settings.smtp.title") }}
    CardDescription
      | {{ $t("settings.smtp.description") }}
  CardContent
    form.flex.flex-col.gap-4(@submit.prevent="save")
      FieldGroup
        .grid.grid-cols-3.gap-3
          Field.col-span-2
            FieldLabel {{ $t("settings.auth.host") }}
            Input(
              v-model="form.host"
              :placeholder="$t('settings.smtp.hostPlaceholder')"
            )
          Field
            FieldLabel {{ $t("settings.auth.port") }}
            Input(
              v-model.number="form.port"
              type="number"
              placeholder="587"
              min="1"
            )
        .grid.grid-cols-2.gap-3
          Field
            FieldLabel {{ $t("common.username") }}
            Input(
              v-model="form.username"
              :placeholder="$t('settings.smtp.optional')"
              autocomplete="off"
            )
          Field
            FieldLabel {{ $t("common.password") }}
            Input(
              v-model="form.password"
              type="password"
              :placeholder="form.has_password ? $t('settings.smtp.unchanged') : $t('settings.smtp.optional')"
              autocomplete="new-password"
            )
        .grid.grid-cols-2.gap-3
          Field
            FieldLabel {{ $t("settings.smtp.from") }}
            Input(
              v-model="form.from"
              :placeholder="$t('settings.smtp.fromPlaceholder')"
            )
          Field
            FieldLabel {{ $t("settings.auth.encryption") }}
            Select(
              :model-value="form.encryption"
              @update:model-value="form.encryption = String($event)"
            )
              SelectTrigger
                SelectValue
              SelectContent
                SelectItem(value="none") {{ $t("common.none") }}
                SelectItem(value="starttls") {{ $t("settings.auth.starttls") }}
                SelectItem(value="tls") {{ $t("settings.auth.implicitTls") }}
      p.text-destructive.text-sm(v-if="error") {{ error }}
      .flex.items-center.justify-between.gap-2
        .flex.items-center.gap-2
          Input.w-56(
            v-model="testAddress"
            type="email"
            :placeholder="$t('account.emails.placeholder')"
          )
          Button(
            type="button"
            variant="outline"
            @click="sendTest"
            :disabled="testing || !testAddress"
          )
            Spinner(v-if="testing" data-icon="inline-start")
            SendIcon(v-else data-icon="inline-start")
            span {{ $t("settings.smtp.sendTest") }}
        Button(type="submit" :disabled="saving")
          Spinner(v-if="saving" data-icon="inline-start")
          CheckIcon(v-else data-icon="inline-start")
          span {{ $t("common.save") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { withProgress } from "@/composables/useProgress";
import { accountApi } from "@/lib/account";
import { CheckIcon, SendIcon } from "@lucide/vue";
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const form = reactive({
  host: "",
  port: 587,
  username: "",
  password: "",
  from: "",
  encryption: "starttls",
  has_password: false,
});
const testAddress = ref("");
const saving = ref(false);
const testing = ref(false);
const error = ref("");

onMounted(load);

async function load(): Promise<void> {
  try {
    const settings = await accountApi.getSMTPSettings();
    Object.assign(form, settings, { password: "" });
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("settings.auth.errors.load");
  }
}

async function save(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const settings = await withProgress(() =>
      accountApi.updateSMTPSettings({
        host: form.host,
        port: Number(form.port) || 0,
        username: form.username,
        from: form.from,
        encryption: form.encryption,
        ...(form.password ? { password: form.password } : {}),
      }),
    );
    Object.assign(form, settings, { password: "" });
    toast.success(t("settings.smtp.toasts.saved"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("account.toasts.saveFailed");
  } finally {
    saving.value = false;
  }
}

async function sendTest(): Promise<void> {
  testing.value = true;
  error.value = "";
  try {
    await accountApi.testSMTP(testAddress.value.trim());
    toast.success(t("settings.smtp.toasts.testSent"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("settings.smtp.toasts.testFailed");
  } finally {
    testing.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  settings:
    smtp:
      title: "Outbound e-mail (SMTP)"
      description: "Used for invitations, magic links, e-mail confirmation and password resets. E-mail invites are disabled until a host and sender are set."
      from: "From address"
      sendTest: "Send test"
      hostPlaceholder: "smtp.example.com"
      optional: "optional"
      fromPlaceholder: "CherryPi <no-reply{'@'}example.com>"
      toasts:
        saved: "SMTP settings saved"
        testSent: "Test message sent"
        testFailed: "Test failed"
      unchanged: "unchanged"
ru:
  settings:
    smtp:
      title: "Исходящая почта (SMTP)"
      description: "Используется для приглашений, ссылок для входа, подтверждения e-mail и сброса пароля. Приглашения по e-mail отключены, пока не заданы хост и отправитель."
      from: "Адрес отправителя"
      sendTest: "Отправить тест"
      hostPlaceholder: "smtp.example.com"
      optional: "необязательно"
      fromPlaceholder: "CherryPi <no-reply{'@'}example.com>"
      toasts:
        saved: "Настройки SMTP сохранены"
        testSent: "Тестовое письмо отправлено"
        testFailed: "Тест не прошёл"
      unchanged: "без изменений"
</i18n>
