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
  .w-full.max-w-sm
    .mb-6.flex.flex-col.items-center.gap-2
      img.block.size-10.rounded-sm.object-contain(
        :src="logoUrl"
        alt="CherryPi"
      )
      h1.text-lg.font-semibold {{ agencyName }}
      p.text-muted-foreground.text-sm Sign in to continue
    Card
      CardHeader
        CardTitle {{ pendingToken ? "Two-factor verification" : "Sign in" }}
        CardDescription(v-if="pendingToken")
          | Enter the code from your authenticator app to finish signing in.
        CardDescription(v-else) Use your username or a confirmed e-mail.
      CardContent
        form.flex.flex-col.gap-4(
          v-if="pendingToken"
          @submit.prevent="submitSecondFactor"
        )
          FieldGroup
            Field
              FieldLabel(for="pending-totp") Two-factor code
              Input#pending-totp(
                v-model="totp"
                placeholder="123456 or a recovery code"
                inputmode="numeric"
                autocomplete="one-time-code"
                autofocus
              )
          p.text-destructive.text-sm(v-if="error") {{ error }}
          .flex.gap-2
            Button.flex-1(type="submit" :disabled="busy || !totp")
              Spinner(v-if="busy")
              span Verify
            Button.flex-1(
              type="button"
              variant="outline"
              @click="cancelPending"
            )
              span Cancel

        template(v-else)
          form.flex.flex-col.gap-4(@submit.prevent="submit")
            FieldGroup
              Field
                FieldLabel(for="identifier") Username or e-mail
                Input#identifier(
                  v-model="identifier"
                  autocomplete="username"
                  autofocus
                  required
                )
              Field
                FieldLabel(for="password") Password
                Input#password(
                  v-model="password"
                  type="password"
                  autocomplete="current-password"
                )
              Field(v-if="totpRequired")
                FieldLabel(for="totp") Two-factor code
                Input#totp(
                  v-model="totp"
                  placeholder="123456 or a recovery code"
                  inputmode="numeric"
                  autocomplete="one-time-code"
                )
            p.text-muted-foreground.text-xs(
              v-if="config && config.password_login === false"
            )
              | Password sign-in is disabled for members; administrators may still use it.
            p.text-destructive.text-sm(v-if="error") {{ error }}
            p.text-sm(v-if="notice") {{ notice }}
            Button.w-full(type="submit" :disabled="busy || !password")
              Spinner(v-if="busy")
              span {{ busy ? "Signing in…" : "Sign in" }}
            .flex.flex-col.gap-1(v-if="config?.magic_link_login !== false")
              Button.w-full(
                type="button"
                variant="outline"
                @click="sendMagicLink"
                :disabled="busy || !identifier"
              )
                MailIcon(data-icon="inline-start")
                span Send me a magic link instead
              p.text-muted-foreground.text-xs
                | You may use a magic link for password-less login or in case you forgot the password.
              a.text-primary.text-xs(v-if="magicLink" :href="magicLink") Open your sign-in link
              p.text-muted-foreground.text-xs(v-if="magicMessage") {{ magicMessage }}

          template(v-if="loginProviders.length || ldapProviders.length")
            .relative.my-2
              .absolute.inset-0.flex.items-center
                span.w-full.border-t
              .relative.flex.justify-center.text-xs.uppercase
                span.bg-background.text-muted-foreground.px-2 Or continue with
            .flex.flex-col.gap-2
              a(
                v-for="provider in loginProviders"
                :key="provider.slug"
                :href="providerStartUrl(provider.slug)"
              )
                Button.w-full(type="button" variant="outline") Sign in with {{ provider.name }}

              form.flex.flex-col.gap-2(
                v-if="ldapProviders.length"
                @submit.prevent="submitLDAP"
              )
                Select(
                  :model-value="ldapProvider"
                  @update:model-value="ldapProvider = String($event)"
                )
                  SelectTrigger.w-full
                    SelectValue(placeholder="LDAP directory")
                  SelectContent
                    SelectItem(
                      v-for="provider in ldapProviders"
                      :key="provider.slug"
                      :value="provider.slug"
                    ) {{ provider.name }}
                Input(
                  v-model="ldapUsername"
                  placeholder="Directory username"
                  autocomplete="username"
                )
                Input(
                  v-model="ldapPassword"
                  type="password"
                  placeholder="Directory password"
                  autocomplete="current-password"
                )
                Button.w-full(
                  type="submit"
                  variant="outline"
                  :disabled="busy || !ldapProvider || !ldapUsername || !ldapPassword"
                )
                  span Sign in with LDAP
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
import { useSettings } from "@/composables/useSettings";
import {
  accountApi,
  ensureSession,
  resetSession,
  type AuthConfig,
  type LoginProvider,
} from "@/lib/account";
import { MailIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();
const { settings, loadSettings } = useSettings();

const identifier = ref("");
const password = ref("");
const totp = ref("");
const totpRequired = ref(false);
const busy = ref(false);
const error = ref("");
const notice = ref("");

const magicLink = ref("");
const magicMessage = ref("");

const loginProviders = ref<LoginProvider[]>([]);
const ldapProviders = ref<LoginProvider[]>([]);
const ldapProvider = ref("");
const ldapUsername = ref("");
const ldapPassword = ref("");

const pendingToken = ref("");
const config = ref<AuthConfig | null>(null);

const agencyName = computed(
  () => settings.value?.agencyName?.trim() || "CherryPi",
);
const logoUrl = computed(() => settings.value?.logo?.url || "/logo.png");

const ERROR_MESSAGES: Record<string, string> = {
  state: "The sign-in session expired. Please try again.",
  provider: "The identity provider could not be reached.",
  unlinked: "No account is linked to that identity. Contact an administrator.",
  disabled: "This account is disabled.",
  denied: "Sign-in was cancelled.",
  auth: "Please sign in first.",
  server: "Something went wrong. Please try again.",
};

onMounted(() => {
  void loadSettings();
  void loadProviders();
  void loadConfig();
  applyRedirectQuery();

  void ensureSession().then((actor) => {
    if (actor) void router.replace(redirectTarget());
  });
});

function applyRedirectQuery(): void {
  const pending = route.query.pending;
  if (typeof pending === "string" && pending) {
    pendingToken.value = pending;
  }
  const code = route.query.error;
  if (typeof code === "string") {
    error.value =
      ERROR_MESSAGES[code] ?? "Something went wrong. Please try again.";
  }
}

async function loadConfig(): Promise<void> {
  try {
    config.value = await accountApi.getConfig();
  } catch {}
}

async function loadProviders(): Promise<void> {
  try {
    const payload = await accountApi.listLoginProviders();
    loginProviders.value = payload.providers ?? [];
    ldapProviders.value = payload.ldap_providers ?? [];
    if (ldapProviders.value.length === 1) {
      ldapProvider.value = ldapProviders.value[0]?.slug ?? "";
    }
  } catch {}
}

function redirectTarget(): string {
  const redirect = route.query.redirect;
  return typeof redirect === "string" && redirect.startsWith("/")
    ? redirect
    : "/tasks";
}

function providerStartUrl(slug: string): string {
  return accountApi.providerStartUrl(slug, "login", redirectTarget());
}

async function submit(): Promise<void> {
  error.value = "";
  notice.value = "";
  busy.value = true;
  try {
    const result = await accountApi.login(
      identifier.value.trim(),
      password.value,
      totp.value.trim(),
      true,
    );
    if (result.totpRequired) {
      totpRequired.value = true;
      notice.value = "Enter the code from your authenticator app.";
      return;
    }
    resetSession();
    await router.replace(redirectTarget());
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Sign in failed";
    if (totpRequired.value) totp.value = "";
  } finally {
    busy.value = false;
  }
}

async function submitLDAP(): Promise<void> {
  error.value = "";
  notice.value = "";
  busy.value = true;
  try {
    const result = await accountApi.ldapLogin(
      ldapProvider.value,
      ldapUsername.value.trim(),
      ldapPassword.value,
      totp.value.trim(),
    );
    if (result.totpRequired) {
      totpRequired.value = true;
      notice.value = "Enter the code from your authenticator app.";
      return;
    }
    resetSession();
    await router.replace(redirectTarget());
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Sign in failed";
  } finally {
    busy.value = false;
  }
}

async function submitSecondFactor(): Promise<void> {
  error.value = "";
  busy.value = true;
  try {
    await accountApi.completeSecondFactor(
      pendingToken.value,
      totp.value.trim(),
    );
    await router.replace(redirectTarget());
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Verification failed";
  } finally {
    busy.value = false;
  }
}

function cancelPending(): void {
  pendingToken.value = "";
  totp.value = "";
  error.value = "";
  void router.replace({ path: "/login" });
}

async function sendMagicLink(): Promise<void> {
  error.value = "";
  magicMessage.value = "";
  magicLink.value = "";
  if (!identifier.value.trim()) {
    error.value = "Enter a username or e-mail first.";
    return;
  }
  busy.value = true;
  try {
    const res = await accountApi.requestMagicLink(identifier.value.trim());
    magicLink.value = res.magic_link ?? "";
    magicMessage.value = magicLink.value
      ? "SMTP is not configured; use the link below."
      : "If the account exists, a sign-in link has been sent.";
  } catch {
    magicMessage.value = "If the account exists, a sign-in link has been sent.";
  } finally {
    busy.value = false;
  }
}
</script>
