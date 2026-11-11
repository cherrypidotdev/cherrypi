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
      CardTitle {{ $t("invite.title") }}
      CardDescription(v-if="preview")
        | {{ $t("invite.choosePassword", { username: preview.username }) }}
      CardDescription(v-else) {{ $t("invite.validating") }}
    CardContent
      template(v-if="preview")
        form.flex.flex-col.gap-4(@submit.prevent="submit")
          FieldGroup
            Field
              FieldLabel(for="username") {{ $t("common.username") }}
              Input#username(disabled :model-value="preview.username")
            Field
              FieldLabel(for="password") {{ $t("common.password") }}
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
          Button.w-full(type="submit" :disabled="busy || !password")
            Spinner(v-if="busy" data-icon="inline-start")
            span {{ busy ? "Joining…" : "Set password & sign in" }}
      template(v-else)
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-center
          Button(type="button" variant="outline" @click="goToLogin") {{ $t("auth.backToSignIn") }}
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
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

const { t } = useI18n();

const route = useRoute();
const router = useRouter();
const preview = ref<{ username: string; email: string } | null>(null);
const password = ref("");
const confirm = ref("");
const busy = ref(false);
const error = ref("");

const token = typeof route.query.token === "string" ? route.query.token : "";

onMounted(async () => {
  if (!token) {
    error.value = t("invite.missingToken");
    return;
  }
  try {
    preview.value = await accountApi.previewInvitation(token);
  } catch (err) {
    error.value = err instanceof Error ? err.message : t("invite.invalid");
  }
});

async function submit(): Promise<void> {
  error.value = "";
  if (password.value !== confirm.value) {
    error.value = t("auth.passwordsMismatch");
    return;
  }
  busy.value = true;
  try {
    await accountApi.acceptInvitation(token, password.value);
    await router.replace("/tasks");
  } catch (err) {
    error.value = err instanceof Error ? err.message : t("invite.failed");
  } finally {
    busy.value = false;
  }
}

function goToLogin(): void {
  void router.replace({ name: "login" });
}
</script>

<i18n lang="yaml">
en:
  invite:
    title: "Accept your invitation"
    validating: "Validating your invitation…"
    failed: "Could not accept"
    invalid: "This invitation is not valid."
    missingToken: "This invitation link is missing its token."
    choosePassword: "Choose a password for {'@'}{username} to finish joining."
  auth:
    backToSignIn: "Back to sign in"
    passwordsMismatch: "Passwords do not match."
ru:
  invite:
    title: "Примите приглашение"
    validating: "Проверяем приглашение…"
    failed: "Не удалось принять приглашение"
    invalid: "Приглашение недействительно."
    missingToken: "В ссылке-приглашении нет токена."
    choosePassword: "Выберите пароль для {'@'}{username}, чтобы завершить присоединение."
  auth:
    backToSignIn: "Назад ко входу"
    passwordsMismatch: "Пароли не совпадают."
</i18n>
