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
      CardTitle Signing you in
      CardDescription(v-if="!error") Verifying your magic link…
    CardContent.flex.flex-col.gap-4
      .flex.items-center.gap-2(v-if="!error")
        Spinner
        span.text-muted-foreground.text-sm Please wait
      template(v-else)
        p.text-destructive.text-sm {{ error }}
        Button.w-full(@click="goToLogin") Back to sign in
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
import { Spinner } from "@/components/ui/spinner";
import { accountApi, resetSession } from "@/lib/account";
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();
const error = ref("");

onMounted(async () => {
  const token = typeof route.query.token === "string" ? route.query.token : "";
  if (!token) {
    error.value = "This sign-in link is missing its token.";
    return;
  }
  try {
    const result = await accountApi.consumeMagicLink(token);
    resetSession();
    if (result.totpRequired && result.pending_token) {
      await router.replace(
        `/login?pending=${encodeURIComponent(result.pending_token)}&method=magic_link`,
      );
      return;
    }
    await router.replace("/tasks");
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : "This link is invalid or expired.";
  }
});

function goToLogin(): void {
  void router.replace({ name: "login" });
}
</script>
