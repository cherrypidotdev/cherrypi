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
      CardTitle Accept your invitation
      CardDescription(v-if="preview")
        | Choose a password for @{{ preview.username }} to finish joining.
      CardDescription(v-else) Validating your invitation…
    CardContent
      template(v-if="preview")
        form.flex.flex-col.gap-4(@submit.prevent="submit")
          FieldGroup
            Field
              FieldLabel(for="username") Username
              Input#username(disabled :model-value="preview.username")
            Field
              FieldLabel(for="password") Password
              Input#password(
                v-model="password"
                type="password"
                autocomplete="new-password"
                required
              )
            Field
              FieldLabel(for="confirm") Confirm password
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
          Button(type="button" variant="outline" @click="goToLogin") Back to sign in
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
import { useRoute, useRouter } from "vue-router";

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
    error.value = "This invitation link is missing its token.";
    return;
  }
  try {
    preview.value = await accountApi.previewInvitation(token);
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : "This invitation is not valid.";
  }
});

async function submit(): Promise<void> {
  error.value = "";
  if (password.value !== confirm.value) {
    error.value = "Passwords do not match.";
    return;
  }
  busy.value = true;
  try {
    await accountApi.acceptInvitation(token, password.value);
    await router.replace("/tasks");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Could not accept";
  } finally {
    busy.value = false;
  }
}

function goToLogin(): void {
  void router.replace({ name: "login" });
}
</script>
