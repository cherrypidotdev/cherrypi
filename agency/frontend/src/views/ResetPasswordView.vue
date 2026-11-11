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
.bg-background.text-foreground.flex.min-h-screen.items-center.justify-center.p-6
  Card.w-full.max-w-sm
    CardHeader
      CardTitle {{ $t("resetPassword.title") }}
      CardDescription {{ $t("resetPassword.description") }}
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="submit")
        FieldGroup
          Field
            FieldLabel(for="password") {{ $t("common.newPassword") }}
            Input#password(
              v-model="password"
              type="password"
              autocomplete="new-password"
              required
            )
          Field
            FieldLabel(for="confirm") {{ $t("common.confirmPassword") }}
            Input#confirm(
              v-model="confirm"
              type="password"
              autocomplete="new-password"
              required
            )
        p.text-destructive.text-sm(v-if="error") {{ error }}
        Button.w-full(type="submit" :disabled="busy")
          Spinner(v-if="busy")
          span {{ busy ? "Saving…" : "Set password" }}
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
import { Spinner } from "@/components/ui/spinner";
import { accountApi } from "@/lib/account";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

const { t } = useI18n();

const route = useRoute();
const router = useRouter();
const password = ref("");
const confirm = ref("");
const busy = ref(false);
const error = ref("");

async function submit(): Promise<void> {
  error.value = "";
  if (password.value !== confirm.value) {
    error.value = t("auth.passwordsMismatch");
    return;
  }
  const token = typeof route.query.token === "string" ? route.query.token : "";
  if (!token) {
    error.value = t("resetPassword.missingToken");
    return;
  }
  busy.value = true;
  try {
    await accountApi.resetPassword(token, password.value);
    await router.replace({ name: "login" });
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("resetPassword.failed");
  } finally {
    busy.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  resetPassword:
    title: "Choose a new password"
    description: "Enter a new password for your account."
    failed: "Reset failed"
    missingToken: "This reset link is missing its token."
ru:
  resetPassword:
    title: "Выберите новый пароль"
    description: "Введите новый пароль для своего аккаунта."
    failed: "Не удалось сбросить пароль"
    missingToken: "В ссылке для сброса нет токена."
</i18n>
