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
.mx-auto.flex.w-full.max-w-4xl.flex-col.gap-6.p-6
  .flex.flex-col.gap-1
    h1.text-lg.font-semibold {{ $t("nav.userSettings") }}
    p.text-muted-foreground.text-sm {{ $t("account.signedInAs", { username: account?.actor.username ?? "" }) }}
  Card.border-destructive(v-if="account?.actor.must_change_password")
    CardHeader
      CardTitle {{ $t("account.mustChange.title") }}
      CardDescription
        | {{ $t("account.mustChange.description") }}
    CardContent
      Button(type="button" @click="tab = 'security'") {{ $t("account.mustChange.action") }}
  Tabs(v-model="tab")
    TabsList
      TabsTrigger(value="profile") {{ $t("account.tabs.profile") }}
      TabsTrigger(value="security") {{ $t("account.tabs.security") }}
      TabsTrigger(value="auth") {{ $t("account.tabs.auth") }}
      TabsTrigger(value="tokens") {{ $t("account.tabs.tokens") }}
      TabsTrigger(value="plugins") {{ $t("account.tabs.plugins") }}
      TabsTrigger(value="notifications") {{ $t("account.tabs.notifications") }}

    TabsContent(value="profile")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle {{ $t("account.tabs.profile") }}
            CardDescription {{ $t("account.profile.description") }}
          CardContent.flex.flex-col.gap-6
            PictureUpload(
              v-if="account"
              :picture="account.actor.picture"
              :user-id="String(account.actor.id)"
              :fallback="profileAvatar"
              @changed="onPictureChanged"
            )
            form.flex.flex-col.gap-4(@submit.prevent="saveProfile")
              FieldGroup
                Field
                  FieldLabel(for="display-name") {{ $t("common.name") }}
                  Input#display-name(v-model="displayName" maxlength="100")
                Field
                  FieldLabel(for="description") {{ $t("common.description") }}
                  Textarea#description(v-model="description" rows="3")
                Field
                  FieldLabel {{ $t("common.username") }}
                  Input(disabled :model-value="account?.actor.username ?? ''")
                  FieldDescription {{ $t("account.profile.usernameHint") }}
              .flex.justify-end
                Button(type="submit" :disabled="savingProfile") {{ $t("account.profile.save") }}
        Card
          CardHeader
            CardTitle {{ $t("account.signOut.title") }}
            CardDescription {{ $t("account.signOut.description") }}
          CardContent
            Button(variant="outline" @click="signOut")
              LogOutIcon(data-icon="inline-start")
              span {{ $t("account.signOut.action") }}

    TabsContent(value="security")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle {{ $t("common.password") }}
            CardDescription {{ $t("account.security.password.description") }}
          CardContent
            form.flex.flex-col.gap-4(@submit.prevent="changePassword")
              FieldGroup
                Field
                  FieldLabel(for="current") {{ $t("account.security.password.current") }}
                  Input#current(
                    v-model="currentPassword"
                    type="password"
                    autocomplete="current-password"
                  )
                Field
                  FieldLabel(for="new") {{ $t("common.newPassword") }}
                  Input#new(
                    v-model="newPassword"
                    type="password"
                    autocomplete="new-password"
                  )
              Button(type="submit" :disabled="savingPassword") {{ $t("account.security.password.submit") }}
        Card
          CardHeader
            CardTitle {{ $t("account.security.totp.title") }}
            CardDescription(v-if="account?.actor.totp_enabled") {{ $t("account.security.totp.enabled", { count: recoveryLeft }) }}
            CardDescription(v-else) {{ $t("account.security.totp.disabled") }}
          CardContent.flex.flex-col.gap-4
            template(v-if="!account?.actor.totp_enabled")
              template(v-if="enrollment")
                p.text-sm {{ $t("account.security.totp.scan") }}
                code.bg-muted.block.overflow-x-auto.rounded-md.p-2.text-xs {{ enrollment.otpauth_url }}
                .flex.items-end.gap-2
                  Field.flex-1
                    FieldLabel(for="totp-code") {{ $t("account.security.totp.code") }}
                    Input#totp-code(v-model="totpCode" inputmode="numeric")
                  Button(@click="activateTOTP" :disabled="busyTOTP") {{ $t("common.verify") }}
              Button(v-else @click="startEnrollment" :disabled="busyTOTP") {{ $t("account.security.totp.enable") }}
            template(v-else)
              .flex.flex-col.gap-2
                Field
                  FieldLabel(for="disable-password") {{ $t("account.security.totp.confirmDisable") }}
                  Input#disable-password(
                    v-model="totpPassword"
                    type="password"
                  )
                .flex.gap-2
                  Button(
                    variant="outline"
                    @click="rotateCodes"
                    :disabled="busyTOTP"
                  ) {{ $t("account.security.totp.rotate") }}
                  Button(
                    variant="destructive"
                    @click="disableTOTP"
                    :disabled="busyTOTP"
                  ) {{ $t("common.disable") }}
            .rounded-md.border.p-3(v-if="recoveryCodes.length")
              p.text-sm.font-medium {{ $t("account.security.totp.recoveryCodes") }}
              .mt-2.grid.grid-cols-2.gap-1
                code.text-xs(v-for="code in recoveryCodes" :key="code") {{ code }}
        Card
          CardHeader
            CardTitle {{ $t("account.sessions.title") }}
            CardDescription {{ $t("account.sessions.description") }}
          CardContent.flex.flex-col.gap-3
            Button(variant="outline" size="sm" @click="revokeOthers") {{ $t("account.sessions.revokeOthers") }}
            .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
              v-for="session in account?.sessions ?? []"
              :key="session.id"
            )
              .min-w-0
                p.truncate.text-sm.font-medium {{ session.user_agent || $t("account.sessions.unknownDevice") }}
                p.text-muted-foreground.text-xs
                  | {{ session.ip }} · {{ $t("account.sessions.lastSeen", { date: formatDate(session.last_seen_at) }) }}
              Button(
                variant="destructive"
                size="sm"
                @click="revokeSession(session.id)"
              ) {{ $t("common.revoke") }}
        ActivityPanel

    TabsContent(value="auth")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle {{ $t("account.emails.title") }}
            CardDescription {{ $t("account.emails.description") }}
          CardContent.flex.flex-col.gap-4
            form.flex.items-end.gap-2(@submit.prevent="addEmail")
              Field.flex-1
                FieldLabel(for="email") {{ $t("account.emails.add") }}
                Input#email(
                  v-model="newEmail"
                  type="email"
                  :placeholder="$t('account.emails.placeholder')"
                )
              Button(type="submit" :disabled="busyEmails") {{ $t("common.add") }}
            .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
              v-for="email in account?.emails ?? []"
              :key="email.id"
            )
              .min-w-0
                p.truncate.text-sm.font-medium {{ email.email }}
                p.text-muted-foreground.text-xs {{ email.confirmed ? $t("account.emails.confirmed") : $t("account.emails.unconfirmed") }}
                a.text-primary.text-xs(v-if="confirmLink" :href="confirmLink") {{ $t("account.emails.openConfirm") }}
              .flex.gap-2
                Button(
                  v-if="!email.confirmed"
                  variant="outline"
                  size="sm"
                  @click="confirmEmail(email.id)"
                ) {{ $t("common.confirm") }}
                Button(
                  variant="destructive"
                  size="sm"
                  @click="deleteEmail(email.id)"
                ) {{ $t("common.remove") }}
        Card
          CardHeader
            CardTitle {{ $t("account.identities.title") }}
            CardDescription
              | {{ $t("account.identities.description") }}
          CardContent.flex.flex-col.gap-4
            .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
              v-for="identity in identities?.identities ?? []"
              :key="`ext-${identity.id}`"
            )
              .min-w-0
                p.truncate.text-sm.font-medium {{ identity.provider_name }}
                p.text-muted-foreground.text-xs
                  | {{ identity.email || identity.subject }} · {{ identity.kind }}
              Button(
                variant="destructive"
                size="sm"
                @click="unlinkIdentity(identity.id)"
              ) {{ $t("common.unlink") }}
            .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
              v-for="identity in identities?.ldap_identities ?? []"
              :key="`ldap-${identity.id}`"
            )
              .min-w-0
                p.truncate.text-sm.font-medium {{ identity.provider_name }}
                p.text-muted-foreground.text-xs LDAP · {{ identity.username }}
              Button(
                variant="destructive"
                size="sm"
                @click="unlinkLDAPIdentity(identity.id)"
              ) {{ $t("common.unlink") }}
            p.text-muted-foreground.text-sm(
              v-if="!((identities?.identities ?? []).length || (identities?.ldap_identities ?? []).length)"
            ) {{ $t("account.identities.empty") }}

            .flex.flex-wrap.gap-2(v-if="(identities?.providers ?? []).length")
              a(
                v-for="provider in identities?.providers ?? []"
                :key="provider.slug"
                :href="linkProviderStart(provider.slug)"
              )
                Button(type="button" variant="outline" size="sm")
                  | {{ $t("account.identities.linkProvider", { name: provider.name }) }}

            form.flex.flex-col.gap-2(
              v-if="(identities?.ldap_providers ?? []).length"
              @submit.prevent="linkLDAP"
            )
              Select(
                :model-value="linkLDAPProvider"
                @update:model-value="linkLDAPProvider = String($event)"
              )
                SelectTrigger.w-full
                  SelectValue(
                    :placeholder="$t('account.identities.ldapDirectory')"
                  )
                SelectContent
                  SelectItem(
                    v-for="provider in identities?.ldap_providers ?? []"
                    :key="provider.slug"
                    :value="provider.slug"
                  ) {{ provider.name }}
              Input(
                v-model="linkLDAPUsername"
                :placeholder="$t('account.identities.ldapUsername')"
                autocomplete="username"
              )
              Input(
                v-model="linkLDAPPassword"
                type="password"
                :placeholder="$t('account.identities.ldapPassword')"
                autocomplete="current-password"
              )
              Button(
                type="submit"
                variant="outline"
                :disabled="busyIdentities || !linkLDAPProvider || !linkLDAPUsername || !linkLDAPPassword"
              ) {{ $t("account.identities.linkLdap") }}

    TabsContent(value="tokens")
      Card
        CardHeader
          CardTitle {{ $t("account.tokens.title") }}
          CardDescription {{ $t("account.tokens.description") }}
        CardContent.flex.flex-col.gap-4
          form.flex.items-end.gap-2(@submit.prevent="createToken")
            Field.flex-1
              FieldLabel(for="token-name") {{ $t("common.name") }}
              Input#token-name(
                v-model="tokenName"
                :placeholder="$t('account.tokens.placeholder')"
              )
            Field.w-auto.shrink-0
              FieldLabel {{ $t("common.expiresIn") }}
              Select(
                :model-value="tokenExpiry"
                @update:model-value="tokenExpiry = String($event)"
              )
                SelectTrigger.w-40
                  SelectValue
                SelectContent
                  SelectGroup
                    SelectItem(
                      v-for="option in expiryOptions"
                      :key="option.value"
                      :value="option.value"
                    ) {{ option.label }}
            Button(type="submit" :disabled="busyTokens") {{ $t("common.create") }}
          FieldDescription {{ expiryPreview }}
          .rounded-md.border.p-3(v-if="createdToken")
            p.text-sm.font-medium {{ $t("account.tokens.newTitle") }}
            code.block.overflow-x-auto.text-xs {{ createdToken }}
          .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
            v-for="token in account?.tokens ?? []"
            :key="token.id"
          )
            .min-w-0
              p.truncate.text-sm.font-medium {{ token.name }}
              p.text-muted-foreground.text-xs
                | {{ token.prefix }}… · {{ $t("account.tokens.created", { date: formatDate(token.created_at) }) }}
                template(v-if="token.last_used_at") · {{ $t("account.tokens.used", { date: formatDate(token.last_used_at) }) }}
                template(v-if="token.expires_at") · {{ $t("account.tokens.expires", { date: formatDate(token.expires_at) }) }}
              p.text-muted-foreground.text-xs(v-if="token.site_id")
                | {{ $t("account.tokens.usedBySite") }}&#32;
                RouterLink.font-mono.underline(
                  :to="{ name: 'site', params: { id: token.site_id } }"
                ) {{ token.site_id }}
            Button(
              variant="destructive"
              size="sm"
              @click="revokeToken(token.id)"
            ) {{ $t("common.revoke") }}
          p.text-muted-foreground.text-sm(
            v-if="!(account?.tokens ?? []).length"
          ) {{ $t("account.tokens.empty") }}

    TabsContent(value="plugins")
      UserPluginsPanel(v-if="account" :actor-id="String(account.actor.id)")

    TabsContent(value="notifications")
      .flex.flex-col.gap-6
        NotificationsPanel
        Alert
          AlertTitle {{ $t("account.webhooks.title") }}
          AlertDescription
            | {{ $t("account.webhooks.description") }}
</template>

<script setup lang="ts">
import ActivityPanel from "@/components/account/ActivityPanel.vue";
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import NotificationsPanel from "@/components/settings/NotificationsPanel.vue";
import UserPluginsPanel from "@/components/settings/UserPluginsPanel.vue";
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
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { resetLocalePreference, useLocale } from "@/composables/useLocale";
import {
  accountApi,
  currentAccount,
  type AccountPayload,
  type IdentitiesPayload,
} from "@/lib/account";
import type { Attachment } from "@cherrypi/sdk";
import { LogOutIcon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const { locale: i18nLocale } = useLocale();
const route = useRoute();
const router = useRouter();
const TABS = [
  "profile",
  "security",
  "auth",
  "tokens",
  "plugins",
  "notifications",
];
const tab = ref(
  route.query.must_change_password
    ? "security"
    : typeof route.query.tab === "string" && TABS.includes(route.query.tab)
      ? route.query.tab
      : "profile",
);
const account = ref<AccountPayload | null>(null);

const displayName = ref("");
const description = ref("");
const savingProfile = ref(false);

const profileAvatar = computed(() =>
  toSvg(account.value?.actor.username || "user", 48),
);

function onPictureChanged(picture: Attachment | undefined): void {
  if (!account.value) return;
  account.value.actor.picture = picture;
  currentAccount.value = account.value.actor;
}

const currentPassword = ref("");
const newPassword = ref("");
const savingPassword = ref(false);

const enrollment = ref<{ secret: string; otpauth_url: string } | null>(null);
const totpCode = ref("");
const totpPassword = ref("");
const busyTOTP = ref(false);
const recoveryCodes = ref<string[]>([]);

const tokenName = ref("");
const tokenExpiry = ref("7");
const busyTokens = ref(false);
const createdToken = ref("");

const expiryOptions = computed(() => [
  { value: "7", label: t("account.tokens.expiry.7days") },
  { value: "30", label: t("account.tokens.expiry.1month") },
  { value: "180", label: t("account.tokens.expiry.6months") },
  { value: "365", label: t("account.tokens.expiry.1year") },
  { value: "never", label: t("common.never") },
]);

const expiryPreview = computed(() => {
  if (tokenExpiry.value === "never") return t("common.neverExpires");
  const days = Number(tokenExpiry.value);
  const date = new Date(Date.now() + days * 24 * 60 * 60 * 1000);
  return t("account.tokens.expiresOn", {
    date: date.toLocaleDateString(i18nLocale.value, {
      year: "numeric",
      month: "short",
      day: "numeric",
    }),
  });
});

const newEmail = ref("");
const busyEmails = ref(false);
const confirmLink = ref("");

const identities = ref<IdentitiesPayload | null>(null);
const linkLDAPProvider = ref("");
const linkLDAPUsername = ref("");
const linkLDAPPassword = ref("");
const busyIdentities = ref(false);

const recoveryLeft = computed(() => account.value?.recovery_codes_left ?? 0);

onMounted(load);

async function load(): Promise<void> {
  try {
    const payload = await accountApi.getAccount();
    account.value = payload;
    currentAccount.value = payload.actor;
    displayName.value = payload.actor.name;
    description.value = payload.actor.description;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : t("account.errors.load"));
  }
  try {
    identities.value = await accountApi.getIdentities();
    const providers = identities.value.ldap_providers ?? [];
    if (providers.length === 1) {
      linkLDAPProvider.value = providers[0]?.slug ?? "";
    }
  } catch {}
}

async function saveProfile(): Promise<void> {
  savingProfile.value = true;
  try {
    await accountApi.updateProfile(displayName.value, description.value);
    await load();
    toast.success(t("account.toasts.profileSaved"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.saveFailed"),
    );
  } finally {
    savingProfile.value = false;
  }
}

async function signOut(): Promise<void> {
  try {
    await accountApi.logout();
  } finally {
    resetLocalePreference();
    await router.replace({ name: "login" });
  }
}

async function changePassword(): Promise<void> {
  savingPassword.value = true;
  try {
    await accountApi.changePassword(currentPassword.value, newPassword.value);
    toast.success(t("account.toasts.passwordChanged"));
    currentPassword.value = "";
    newPassword.value = "";
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.changeFailed"),
    );
  } finally {
    savingPassword.value = false;
  }
}

async function startEnrollment(): Promise<void> {
  busyTOTP.value = true;
  try {
    enrollment.value = await accountApi.enrollTOTP();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.enrollmentFailed"),
    );
  } finally {
    busyTOTP.value = false;
  }
}

async function activateTOTP(): Promise<void> {
  busyTOTP.value = true;
  try {
    const res = await accountApi.activateTOTP(totpCode.value.trim());
    recoveryCodes.value = res.recovery_codes;
    enrollment.value = null;
    totpCode.value = "";
    await load();
    toast.success(t("account.toasts.totpEnabled"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.invalidCode"),
    );
  } finally {
    busyTOTP.value = false;
  }
}

async function disableTOTP(): Promise<void> {
  busyTOTP.value = true;
  try {
    await accountApi.disableTOTP(totpPassword.value);
    totpPassword.value = "";
    recoveryCodes.value = [];
    await load();
    toast.success(t("account.toasts.totpDisabled"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.disableFailed"),
    );
  } finally {
    busyTOTP.value = false;
  }
}

async function rotateCodes(): Promise<void> {
  busyTOTP.value = true;
  try {
    const res = await accountApi.rotateRecoveryCodes(totpPassword.value);
    recoveryCodes.value = res.recovery_codes;
    totpPassword.value = "";
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.rotateFailed"),
    );
  } finally {
    busyTOTP.value = false;
  }
}

async function createToken(): Promise<void> {
  busyTokens.value = true;
  try {
    const days = tokenExpiry.value === "never" ? 0 : Number(tokenExpiry.value);
    const res = await accountApi.createToken(tokenName.value, days);
    createdToken.value = res.token;
    tokenName.value = "";
    tokenExpiry.value = "7";
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.createFailed"),
    );
  } finally {
    busyTokens.value = false;
  }
}

async function revokeToken(id: number): Promise<void> {
  try {
    await accountApi.revokeToken(id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.revokeFailed"),
    );
  }
}

async function revokeSession(id: number): Promise<void> {
  try {
    await accountApi.revokeSession(id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.revokeFailed"),
    );
  }
}

async function revokeOthers(): Promise<void> {
  try {
    await accountApi.revokeOtherSessions();
    await load();
    toast.success(t("account.toasts.revokedOthers"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.revokeFailed"),
    );
  }
}

async function addEmail(): Promise<void> {
  busyEmails.value = true;
  confirmLink.value = "";
  try {
    const res = await accountApi.addEmail(newEmail.value.trim());
    confirmLink.value = res.confirm_link ?? "";
    newEmail.value = "";
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.emailFailed"),
    );
  } finally {
    busyEmails.value = false;
  }
}

async function confirmEmail(id: number): Promise<void> {
  try {
    const res = await accountApi.confirmEmailById(id);
    confirmLink.value = res.confirm_link ?? "";
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.confirmFailed"),
    );
  }
}

async function deleteEmail(id: number): Promise<void> {
  try {
    await accountApi.deleteEmail(id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.removeFailed"),
    );
  }
}

function formatDate(value: string): string {
  try {
    return new Date(value).toLocaleString();
  } catch {
    return value;
  }
}

function linkProviderStart(slug: string): string {
  return accountApi.providerStartUrl(slug, "link", "/account");
}

async function unlinkIdentity(id: number): Promise<void> {
  try {
    await accountApi.unlinkIdentity(id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.unlinkFailed"),
    );
  }
}

async function unlinkLDAPIdentity(id: number): Promise<void> {
  try {
    await accountApi.unlinkLDAPIdentity(id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.unlinkFailed"),
    );
  }
}

async function linkLDAP(): Promise<void> {
  busyIdentities.value = true;
  try {
    await accountApi.linkLDAPIdentity(
      linkLDAPProvider.value,
      linkLDAPUsername.value.trim(),
      linkLDAPPassword.value,
    );
    linkLDAPUsername.value = "";
    linkLDAPPassword.value = "";
    await load();
    toast.success(t("account.toasts.ldapLinked"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("account.toasts.linkFailed"),
    );
  } finally {
    busyIdentities.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  account:
    toasts:
      saveFailed: "Save failed"
      linkFailed: "Link failed"
      ldapLinked: "LDAP account linked"
      unlinkFailed: "Unlink failed"
      removeFailed: "Remove failed"
      confirmFailed: "Confirm failed"
      emailFailed: "Could not add e-mail"
      revokeFailed: "Revoke failed"
      revokedOthers: "Other sessions revoked"
      createFailed: "Create failed"
      rotateFailed: "Rotate failed"
      disableFailed: "Disable failed"
      totpDisabled: "Two-factor authentication disabled"
      invalidCode: "Invalid code"
      totpEnabled: "Two-factor authentication enabled"
      enrollmentFailed: "Enrollment failed"
      changeFailed: "Change failed"
      passwordChanged: "Password changed; sign in again"
      profileSaved: "Profile saved"
    tabs:
      plugins: "Plugins"
      tokens: "Tokens"
      profile: "Profile"
      security: "Security"
      auth: "Auth"
      notifications: "Notifications"
    emails:
      placeholder: "you{'@'}example.com"
      title: "E-mail addresses"
      description: "A confirmed address can receive magic links."
      add: "Add e-mail"
      openConfirm: "Open confirmation link"
      confirmed: "Confirmed"
      unconfirmed: "Unconfirmed"
    identities:
      ldapDirectory: "LDAP directory"
      title: "Linked accounts"
      description: "OIDC, OAuth2 and LDAP identities linked to this account. Sign in through them without a password."
      empty: "No linked identities yet."
      linkLdap: "Link LDAP account"
      ldapUsername: "Directory username"
      ldapPassword: "Directory password"
      linkProvider: "Link {name}"
    mustChange:
      title: "Set a new password"
      description: "Your temporary password must be replaced before you can use the agency."
      action: "Go to Security"
    profile:
      description: "Your name, picture, and description."
      usernameHint: "Usernames are unique across users and agents."
      save: "Save profile"
    signOut:
      description: "End this browser session."
      action: "Log out"
      title: "Sign out"
    security:
      password:
        description: "Change the password used to sign in."
        current: "Current password"
        submit: "Change password"
      totp:
        title: "Two-factor authentication"
        disabled: "Add a TOTP authenticator app."
        scan: "Scan the URI in your authenticator, then enter a code."
        code: "Code"
        enable: "Enable two-factor"
        confirmDisable: "Confirm password to disable"
        rotate: "New recovery codes"
        recoveryCodes: "Recovery codes (save them now)"
        enabled: "Enabled. {count} recovery code(s) left."
    sessions:
      title: "Sessions"
      description: "Browsers signed in to your account."
      revokeOthers: "Sign out everywhere else"
      unknownDevice: "Unknown device"
      lastSeen: "last seen {date}"
    tokens:
      title: "Personal access tokens"
      description: "API tokens for the SDK and CLI. Shown once."
      newTitle: "Your new token"
      empty: "No tokens yet."
      placeholder: "laptop"
      expiry:
        7days: "7 days"
        1month: "1 month"
        6months: "6 months"
        1year: "1 year"
      expiresOn: "Expires on {date}"
      created: "created {date}"
      used: "used {date}"
      expires: "expires {date}"
      usedBySite: "Used by site"
    webhooks:
      title: "User webhooks"
      description: "Per-user, per-event webhook delivery is not built yet. It is tracked in CHPI-177; the browser and inbox preferences above work today."
    errors:
      load: "Failed to load account"
    signedInAs: "Signed in as {username}"
ru:
  account:
    toasts:
      saveFailed: "Не удалось сохранить"
      linkFailed: "Не удалось связать"
      ldapLinked: "Аккаунт LDAP связан"
      unlinkFailed: "Не удалось отвязать"
      removeFailed: "Не удалось удалить"
      confirmFailed: "Не удалось подтвердить"
      emailFailed: "Не удалось добавить e-mail"
      revokeFailed: "Не удалось отозвать"
      revokedOthers: "Остальные сеансы отозваны"
      createFailed: "Не удалось создать"
      rotateFailed: "Не удалось обновить коды"
      disableFailed: "Не удалось отключить"
      totpDisabled: "Двухфакторная аутентификация отключена"
      invalidCode: "Неверный код"
      totpEnabled: "Двухфакторная аутентификация включена"
      enrollmentFailed: "Не удалось начать подключение"
      changeFailed: "Не удалось изменить"
      passwordChanged: "Пароль изменён; войдите заново"
      profileSaved: "Профиль сохранён"
    tabs:
      plugins: "Плагины"
      tokens: "Токены"
      profile: "Профиль"
      security: "Безопасность"
      auth: "Входы"
      notifications: "Уведомления"
    emails:
      placeholder: "you{'@'}example.com"
      title: "Адреса e-mail"
      description: "На подтверждённый адрес можно получать ссылки для входа."
      add: "Добавить e-mail"
      openConfirm: "Открыть ссылку подтверждения"
      confirmed: "Подтверждён"
      unconfirmed: "Не подтверждён"
    identities:
      ldapDirectory: "Каталог LDAP"
      title: "Связанные аккаунты"
      description: "Учётные записи OIDC, OAuth2 и LDAP, связанные с этим аккаунтом. Через них можно входить без пароля."
      empty: "Связанных учётных записей пока нет."
      linkLdap: "Связать аккаунт LDAP"
      ldapUsername: "Имя пользователя в каталоге"
      ldapPassword: "Пароль в каталоге"
      linkProvider: "Связать {name}"
    mustChange:
      title: "Задайте новый пароль"
      description: "Временный пароль нужно заменить, прежде чем пользоваться агентством."
      action: "Перейти к безопасности"
    profile:
      description: "Имя, изображение и описание."
      usernameHint: "Имена пользователей уникальны среди пользователей и агентов."
      save: "Сохранить профиль"
    signOut:
      description: "Завершить сеанс в этом браузере."
      action: "Выйти"
      title: "Выход"
    security:
      password:
        description: "Изменить пароль для входа."
        current: "Текущий пароль"
        submit: "Изменить пароль"
      totp:
        title: "Двухфакторная аутентификация"
        disabled: "Подключите приложение-аутентификатор TOTP."
        scan: "Отсканируйте URI в аутентификаторе, затем введите код."
        code: "Код"
        enable: "Включить двухфакторную защиту"
        confirmDisable: "Подтвердите пароль, чтобы отключить"
        rotate: "Новые коды восстановления"
        recoveryCodes: "Коды восстановления (сохраните их сейчас)"
        enabled: "Включено. Осталось кодов восстановления: {count}."
    sessions:
      title: "Сеансы"
      description: "Браузеры, в которых выполнен вход в аккаунт."
      revokeOthers: "Выйти на всех остальных устройствах"
      unknownDevice: "Неизвестное устройство"
      lastSeen: "последний вход {date}"
    tokens:
      title: "Персональные токены доступа"
      description: "API-токены для SDK и CLI. Показываются один раз."
      newTitle: "Ваш новый токен"
      empty: "Токенов пока нет."
      placeholder: "ноутбук"
      expiry:
        7days: "7 дней"
        1month: "1 месяц"
        6months: "6 месяцев"
        1year: "1 год"
      expiresOn: "Истекает {date}"
      created: "создан {date}"
      used: "использован {date}"
      expires: "истекает {date}"
      usedBySite: "Используется сайтом"
    webhooks:
      title: "Пользовательские вебхуки"
      description: "Доставка вебхуков для отдельных пользователей и событий пока не реализована. Это отслеживается в CHPI-177; настройки браузера и уведомлений выше уже работают."
    errors:
      load: "Не удалось загрузить аккаунт"
    signedInAs: "Вы вошли как {username}"
</i18n>
