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
  .absolute.top-4.right-4
    LanguageSwitcher
  .w-full.max-w-sm
    .mb-6.flex.flex-col.items-center.gap-2
      img.block.size-10.rounded-sm.object-contain(
        :src="logoUrl"
        alt="CherryPi"
      )
      h1.text-lg.font-semibold {{ agencyName }}
      p.text-muted-foreground.text-sm {{ $t("login.subtitle") }}
    Card
      CardHeader
        CardTitle {{ pendingToken ? $t("login.twoFactor.title") : $t("login.title") }}
        CardDescription(v-if="pendingToken")
          | {{ $t("login.twoFactor.description") }}
        CardDescription(v-else) {{ $t("login.description") }}
      CardContent
        form.flex.flex-col.gap-4(
          v-if="pendingToken"
          @submit.prevent="submitSecondFactor"
        )
          FieldGroup
            Field
              FieldLabel(for="pending-totp") {{ $t("login.twoFactor.code") }}
              Input#pending-totp(
                v-model="totp"
                :placeholder="$t('login.twoFactor.codePlaceholder')"
                inputmode="numeric"
                autocomplete="one-time-code"
                autofocus
              )
          p.text-destructive.text-sm(v-if="error") {{ error }}
          .flex.gap-2
            Button.flex-1(type="submit" :disabled="busy || !totp")
              Spinner(v-if="busy")
              span {{ $t("login.twoFactor.verify") }}
            Button.flex-1(
              type="button"
              variant="outline"
              @click="cancelPending"
            )
              span {{ $t("common.cancel") }}

        template(v-else)
          form.flex.flex-col.gap-4(@submit.prevent="submit")
            FieldGroup
              Field
                FieldLabel(for="identifier") {{ $t("login.identifier") }}
                Input#identifier(
                  v-model="identifier"
                  autocomplete="username"
                  autofocus
                  required
                )
              Field
                FieldLabel(for="password") {{ $t("login.password") }}
                Input#password(
                  v-model="password"
                  type="password"
                  autocomplete="current-password"
                )
              Field(v-if="totpRequired")
                FieldLabel(for="totp") {{ $t("login.twoFactor.code") }}
                Input#totp(
                  v-model="totp"
                  :placeholder="$t('login.twoFactor.codePlaceholder')"
                  inputmode="numeric"
                  autocomplete="one-time-code"
                )
            p.text-muted-foreground.text-xs(
              v-if="config && config.password_login === false"
            )
              | {{ $t("login.passwordDisabled") }}
            p.text-destructive.text-sm(v-if="error") {{ error }}
            p.text-sm(v-if="notice") {{ notice }}
            Button.w-full(type="submit" :disabled="busy || !password")
              Spinner(v-if="busy")
              span {{ busy ? $t("login.signingIn") : $t("login.title") }}
            .flex.flex-col.gap-1(v-if="config?.magic_link_login !== false")
              Button.w-full(
                type="button"
                variant="outline"
                @click="sendMagicLink"
                :disabled="busy || !identifier"
              )
                MailIcon(data-icon="inline-start")
                span {{ $t("login.magicLink.send") }}
              p.text-muted-foreground.text-xs
                | {{ $t("login.magicLink.hint") }}
              a.text-primary.text-xs(v-if="magicLink" :href="magicLink") {{ $t("login.magicLink.open") }}
              p.text-muted-foreground.text-xs(v-if="magicMessage") {{ magicMessage }}

          template(v-if="loginProviders.length || ldapProviders.length")
            .relative.my-2
              .absolute.inset-0.flex.items-center
                span.w-full.border-t
              .relative.flex.justify-center.text-xs.uppercase
                span.bg-background.text-muted-foreground.px-2 {{ $t("login.orContinue") }}
            .flex.flex-col.gap-2
              a(
                v-for="provider in loginProviders"
                :key="provider.slug"
                :href="providerStartUrl(provider.slug)"
              )
                Button.w-full(type="button" variant="outline") {{ $t("login.withProvider", { name: provider.name }) }}

              form.flex.flex-col.gap-2(
                v-if="ldapProviders.length"
                @submit.prevent="submitLDAP"
              )
                Select(
                  :model-value="ldapProvider"
                  @update:model-value="ldapProvider = String($event)"
                )
                  SelectTrigger.w-full
                    SelectValue(:placeholder="$t('login.ldap.directory')")
                  SelectContent
                    SelectItem(
                      v-for="provider in ldapProviders"
                      :key="provider.slug"
                      :value="provider.slug"
                    ) {{ provider.name }}
                Input(
                  v-model="ldapUsername"
                  :placeholder="$t('login.ldap.username')"
                  autocomplete="username"
                )
                Input(
                  v-model="ldapPassword"
                  type="password"
                  :placeholder="$t('login.ldap.password')"
                  autocomplete="current-password"
                )
                Button.w-full(
                  type="submit"
                  variant="outline"
                  :disabled="busy || !ldapProvider || !ldapUsername || !ldapPassword"
                )
                  span {{ $t("login.ldap.submit") }}
    p.mt-4.text-center.text-xs
      a.text-muted-foreground.inline-flex.items-center.gap-1.transition-colors(
        href="/docs/"
        rel="noopener"
        class="hover:text-foreground"
        target="_blank"
      )
        BookOpenIcon.size-3
        span {{ $t("nav.documentation") }}
</template>

<script setup lang="ts">
import LanguageSwitcher from "@/components/LanguageSwitcher.vue";
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
import { BookOpenIcon, MailIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

const { t } = useI18n();

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

const ERROR_KEYS: Record<string, string> = {
  state: "login.errors.state",
  provider: "login.errors.provider",
  unlinked: "login.errors.unlinked",
  provision: "login.errors.provision",
  disabled: "login.errors.disabled",
  denied: "login.errors.denied",
  auth: "login.errors.auth",
  server: "login.errors.server",
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
    error.value = t(ERROR_KEYS[code] ?? "login.errors.server");
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
      notice.value = t("login.twoFactor.notice");
      return;
    }
    resetSession();
    await router.replace(redirectTarget());
  } catch (err) {
    error.value = err instanceof Error ? err.message : t("login.errors.failed");
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
      notice.value = t("login.twoFactor.notice");
      return;
    }
    resetSession();
    await router.replace(redirectTarget());
  } catch (err) {
    error.value = err instanceof Error ? err.message : t("login.errors.failed");
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
    error.value =
      err instanceof Error ? err.message : t("login.errors.verificationFailed");
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
    error.value = t("login.errors.identifierRequired");
    return;
  }
  busy.value = true;
  try {
    const res = await accountApi.requestMagicLink(identifier.value.trim());
    magicLink.value = res.magic_link ?? "";
    magicMessage.value = magicLink.value
      ? t("login.magicLink.noSmtp")
      : t("login.magicLink.sent");
  } catch {
    magicMessage.value = t("login.magicLink.sent");
  } finally {
    busy.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  login:
    title: "Sign in"
    subtitle: "Sign in to continue"
    description: "Use your username or a confirmed e-mail."
    identifier: "Username or e-mail"
    password: "Password"
    passwordDisabled: "Password sign-in is disabled for members; administrators may still use it."
    signingIn: "Signing in…"
    orContinue: "Or continue with"
    withProvider: "Sign in with {name}"
    twoFactor:
      title: "Two-factor verification"
      description: "Enter the code from your authenticator app to finish signing in."
      code: "Two-factor code"
      codePlaceholder: "123456 or a recovery code"
      verify: "Verify"
      notice: "Enter the code from your authenticator app."
    magicLink:
      send: "Send me a magic link instead"
      hint: "You may use a magic link for password-less login or in case you forgot the password."
      open: "Open your sign-in link"
      noSmtp: "SMTP is not configured; use the link below."
      sent: "If the account exists, a sign-in link has been sent."
    ldap:
      directory: "LDAP directory"
      username: "Directory username"
      password: "Directory password"
      submit: "Sign in with LDAP"
    errors:
      state: "The sign-in session expired. Please try again."
      provider: "The identity provider could not be reached."
      unlinked: "No account is linked to that identity. Ask an administrator to invite you, or to enable automatic account creation for this provider."
      provision: "Your identity was verified, but CherryPi could not create an account for it. Ask an administrator to invite you."
      disabled: "This account is disabled."
      denied: "Sign-in was cancelled."
      auth: "Please sign in first."
      server: "Something went wrong. Please try again."
      failed: "Sign in failed"
      verificationFailed: "Verification failed"
      identifierRequired: "Enter a username or e-mail first."
ru:
  login:
    title: "Вход"
    subtitle: "Войдите, чтобы продолжить"
    description: "Используйте имя пользователя или подтверждённый e-mail."
    identifier: "Имя пользователя или e-mail"
    password: "Пароль"
    passwordDisabled: "Вход по паролю отключён для участников; администраторы могут им пользоваться."
    signingIn: "Вход…"
    orContinue: "Или продолжите с"
    withProvider: "Войти через {name}"
    twoFactor:
      title: "Двухфакторная проверка"
      description: "Введите код из приложения-аутентификатора, чтобы завершить вход."
      code: "Код двухфакторной аутентификации"
      codePlaceholder: "123456 или код восстановления"
      verify: "Подтвердить"
      notice: "Введите код из приложения-аутентификатора."
    magicLink:
      send: "Отправить ссылку для входа"
      hint: "Ссылка для входа подойдёт для входа без пароля или если вы забыли пароль."
      open: "Открыть ссылку для входа"
      noSmtp: "SMTP не настроен; воспользуйтесь ссылкой ниже."
      sent: "Если аккаунт существует, ссылка для входа уже отправлена."
    ldap:
      directory: "Каталог LDAP"
      username: "Имя пользователя в каталоге"
      password: "Пароль в каталоге"
      submit: "Войти через LDAP"
    errors:
      state: "Сессия входа истекла. Попробуйте ещё раз."
      provider: "Не удалось связаться с провайдером идентификации."
      unlinked: "С этой учётной записью не связан ни один аккаунт. Попросите администратора пригласить вас или включить автоматическое создание аккаунтов для этого провайдера."
      provision: "Учётные данные подтверждены, но CherryPi не смог создать для них аккаунт. Попросите администратора пригласить вас."
      disabled: "Этот аккаунт отключён."
      denied: "Вход отменён."
      auth: "Сначала войдите в систему."
      server: "Что-то пошло не так. Попробуйте ещё раз."
      failed: "Не удалось войти"
      verificationFailed: "Не удалось подтвердить код"
      identifierRequired: "Сначала введите имя пользователя или e-mail."
</i18n>
