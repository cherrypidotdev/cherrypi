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
    h1.text-lg.font-semibold User settings
    p.text-muted-foreground.text-sm Signed in as {{ account?.actor.username }}
  Card.border-destructive(v-if="account?.actor.must_change_password")
    CardHeader
      CardTitle Set a new password
      CardDescription
        | Your temporary password must be replaced before you can use the
        | agency.
    CardContent
      Button(type="button" @click="tab = 'security'") Go to Security
  Tabs(v-model="tab")
    TabsList
      TabsTrigger(value="profile") Profile
      TabsTrigger(value="security") Security
      TabsTrigger(value="auth") Auth
      TabsTrigger(value="tokens") Tokens
      TabsTrigger(value="notifications") Notifications

    TabsContent(value="profile")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle Profile
            CardDescription Your name, picture, and description.
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
                  FieldLabel(for="display-name") Name
                  Input#display-name(v-model="displayName" maxlength="100")
                Field
                  FieldLabel(for="description") Description
                  Textarea#description(v-model="description" rows="3")
                Field
                  FieldLabel Username
                  Input(disabled :model-value="account?.actor.username ?? ''")
                  FieldDescription Usernames are unique across users and agents.
              .flex.justify-end
                Button(type="submit" :disabled="savingProfile") Save profile
        Card
          CardHeader
            CardTitle Sign out
            CardDescription End this browser session.
          CardContent
            Button(variant="outline" @click="signOut")
              LogOutIcon(data-icon="inline-start")
              span Log out

    TabsContent(value="security")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle Password
            CardDescription Change the password used to sign in.
          CardContent
            form.flex.flex-col.gap-4(@submit.prevent="changePassword")
              FieldGroup
                Field
                  FieldLabel(for="current") Current password
                  Input#current(
                    v-model="currentPassword"
                    type="password"
                    autocomplete="current-password"
                  )
                Field
                  FieldLabel(for="new") New password
                  Input#new(
                    v-model="newPassword"
                    type="password"
                    autocomplete="new-password"
                  )
              Button(type="submit" :disabled="savingPassword") Change password
        Card
          CardHeader
            CardTitle Two-factor authentication
            CardDescription(v-if="account?.actor.totp_enabled") Enabled. {{ recoveryLeft }} recovery code(s) left.
            CardDescription(v-else) Add a TOTP authenticator app.
          CardContent.flex.flex-col.gap-4
            template(v-if="!account?.actor.totp_enabled")
              template(v-if="enrollment")
                p.text-sm Scan the URI in your authenticator, then enter a code.
                code.bg-muted.block.overflow-x-auto.rounded-md.p-2.text-xs {{ enrollment.otpauth_url }}
                .flex.items-end.gap-2
                  Field.flex-1
                    FieldLabel(for="totp-code") Code
                    Input#totp-code(v-model="totpCode" inputmode="numeric")
                  Button(@click="activateTOTP" :disabled="busyTOTP") Verify
              Button(v-else @click="startEnrollment" :disabled="busyTOTP") Enable two-factor
            template(v-else)
              .flex.flex-col.gap-2
                Field
                  FieldLabel(for="disable-password") Confirm password to disable
                  Input#disable-password(
                    v-model="totpPassword"
                    type="password"
                  )
                .flex.gap-2
                  Button(
                    variant="outline"
                    @click="rotateCodes"
                    :disabled="busyTOTP"
                  ) New recovery codes
                  Button(
                    variant="destructive"
                    @click="disableTOTP"
                    :disabled="busyTOTP"
                  ) Disable
            .rounded-md.border.p-3(v-if="recoveryCodes.length")
              p.text-sm.font-medium Recovery codes (save them now)
              .mt-2.grid.grid-cols-2.gap-1
                code.text-xs(v-for="code in recoveryCodes" :key="code") {{ code }}
        Card
          CardHeader
            CardTitle Sessions
            CardDescription Browsers signed in to your account.
          CardContent.flex.flex-col.gap-3
            Button(variant="outline" size="sm" @click="revokeOthers") Sign out everywhere else
            .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
              v-for="session in account?.sessions ?? []"
              :key="session.id"
            )
              .min-w-0
                p.truncate.text-sm.font-medium {{ session.user_agent || "Unknown device" }}
                p.text-muted-foreground.text-xs
                  | {{ session.ip }} · last seen {{ formatDate(session.last_seen_at) }}
              Button(
                variant="destructive"
                size="sm"
                @click="revokeSession(session.id)"
              ) Revoke
        ActivityPanel

    TabsContent(value="auth")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle E-mail addresses
            CardDescription A confirmed address can receive magic links.
          CardContent.flex.flex-col.gap-4
            form.flex.items-end.gap-2(@submit.prevent="addEmail")
              Field.flex-1
                FieldLabel(for="email") Add e-mail
                Input#email(
                  v-model="newEmail"
                  type="email"
                  placeholder="you@example.com"
                )
              Button(type="submit" :disabled="busyEmails") Add
            .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
              v-for="email in account?.emails ?? []"
              :key="email.id"
            )
              .min-w-0
                p.truncate.text-sm.font-medium {{ email.email }}
                p.text-muted-foreground.text-xs {{ email.confirmed ? "Confirmed" : "Unconfirmed" }}
                a.text-primary.text-xs(v-if="confirmLink" :href="confirmLink") Open confirmation link
              .flex.gap-2
                Button(
                  v-if="!email.confirmed"
                  variant="outline"
                  size="sm"
                  @click="confirmEmail(email.id)"
                ) Confirm
                Button(
                  variant="destructive"
                  size="sm"
                  @click="deleteEmail(email.id)"
                ) Remove
        Card
          CardHeader
            CardTitle Linked accounts
            CardDescription
              | OIDC, OAuth2 and LDAP identities linked to this account. Sign in
              | through them without a password.
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
              ) Unlink
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
              ) Unlink
            p.text-muted-foreground.text-sm(
              v-if="!((identities?.identities ?? []).length || (identities?.ldap_identities ?? []).length)"
            ) No linked identities yet.

            .flex.flex-wrap.gap-2(v-if="(identities?.providers ?? []).length")
              a(
                v-for="provider in identities?.providers ?? []"
                :key="provider.slug"
                :href="linkProviderStart(provider.slug)"
              )
                Button(type="button" variant="outline" size="sm")
                  | Link {{ provider.name }}

            form.flex.flex-col.gap-2(
              v-if="(identities?.ldap_providers ?? []).length"
              @submit.prevent="linkLDAP"
            )
              Select(
                :model-value="linkLDAPProvider"
                @update:model-value="linkLDAPProvider = String($event)"
              )
                SelectTrigger.w-full
                  SelectValue(placeholder="LDAP directory")
                SelectContent
                  SelectItem(
                    v-for="provider in identities?.ldap_providers ?? []"
                    :key="provider.slug"
                    :value="provider.slug"
                  ) {{ provider.name }}
              Input(
                v-model="linkLDAPUsername"
                placeholder="Directory username"
                autocomplete="username"
              )
              Input(
                v-model="linkLDAPPassword"
                type="password"
                placeholder="Directory password"
                autocomplete="current-password"
              )
              Button(
                type="submit"
                variant="outline"
                :disabled="busyIdentities || !linkLDAPProvider || !linkLDAPUsername || !linkLDAPPassword"
              ) Link LDAP account

    TabsContent(value="tokens")
      Card
        CardHeader
          CardTitle Personal access tokens
          CardDescription API tokens for the SDK and CLI. Shown once.
        CardContent.flex.flex-col.gap-4
          form.flex.items-end.gap-2(@submit.prevent="createToken")
            Field.flex-1
              FieldLabel(for="token-name") Name
              Input#token-name(v-model="tokenName" placeholder="laptop")
            Field.w-auto.shrink-0
              FieldLabel Expires in
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
            Button(type="submit" :disabled="busyTokens") Create
          FieldDescription {{ expiryPreview }}
          .rounded-md.border.p-3(v-if="createdToken")
            p.text-sm.font-medium Your new token
            code.block.overflow-x-auto.text-xs {{ createdToken }}
          .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
            v-for="token in account?.tokens ?? []"
            :key="token.id"
          )
            .min-w-0
              p.truncate.text-sm.font-medium {{ token.name }}
              p.text-muted-foreground.text-xs
                | {{ token.prefix }}… · created {{ formatDate(token.created_at) }}
                template(v-if="token.last_used_at") · used {{ formatDate(token.last_used_at) }}
                template(v-if="token.expires_at") · expires {{ formatDate(token.expires_at) }}
            Button(
              variant="destructive"
              size="sm"
              @click="revokeToken(token.id)"
            ) Revoke
          p.text-muted-foreground.text-sm(
            v-if="!(account?.tokens ?? []).length"
          ) No tokens yet.

    TabsContent(value="notifications")
      .flex.flex-col.gap-6
        NotificationsPanel
        Alert
          AlertTitle User webhooks
          AlertDescription
            | Per-user, per-event webhook delivery is not built yet. It is tracked
            | in CHPI-177; the browser and inbox preferences above work today.
</template>

<script setup lang="ts">
import ActivityPanel from "@/components/account/ActivityPanel.vue";
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import NotificationsPanel from "@/components/settings/NotificationsPanel.vue";
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
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const route = useRoute();
const router = useRouter();
const TABS = ["profile", "security", "auth", "tokens", "notifications"];
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

const expiryOptions = [
  { value: "7", label: "7 days" },
  { value: "30", label: "1 month" },
  { value: "180", label: "6 months" },
  { value: "365", label: "1 year" },
  { value: "never", label: "Never" },
];

const expiryPreview = computed(() => {
  if (tokenExpiry.value === "never") return "Never expires";
  const days = Number(tokenExpiry.value);
  const date = new Date(Date.now() + days * 24 * 60 * 60 * 1000);
  return `Expires on ${date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })}`;
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
    toast.error(err instanceof Error ? err.message : "Failed to load account");
  }
  try {
    identities.value = await accountApi.getIdentities();
    const providers = identities.value.ldap_providers ?? [];
    if (providers.length === 1) {
      linkLDAPProvider.value = providers[0]?.slug ?? "";
    }
  } catch {
    // The identities surface is optional for accounts that have no providers.
  }
}

async function saveProfile(): Promise<void> {
  savingProfile.value = true;
  try {
    await accountApi.updateProfile(displayName.value, description.value);
    await load();
    toast.success("Profile saved");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Save failed");
  } finally {
    savingProfile.value = false;
  }
}

async function signOut(): Promise<void> {
  try {
    await accountApi.logout();
  } finally {
    await router.replace({ name: "login" });
  }
}

async function changePassword(): Promise<void> {
  savingPassword.value = true;
  try {
    await accountApi.changePassword(currentPassword.value, newPassword.value);
    toast.success("Password changed; sign in again");
    currentPassword.value = "";
    newPassword.value = "";
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Change failed");
  } finally {
    savingPassword.value = false;
  }
}

async function startEnrollment(): Promise<void> {
  busyTOTP.value = true;
  try {
    enrollment.value = await accountApi.enrollTOTP();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Enrollment failed");
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
    toast.success("Two-factor authentication enabled");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Invalid code");
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
    toast.success("Two-factor authentication disabled");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Disable failed");
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
    toast.error(err instanceof Error ? err.message : "Rotate failed");
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
    toast.error(err instanceof Error ? err.message : "Create failed");
  } finally {
    busyTokens.value = false;
  }
}

async function revokeToken(id: number): Promise<void> {
  try {
    await accountApi.revokeToken(id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Revoke failed");
  }
}

async function revokeSession(id: number): Promise<void> {
  try {
    await accountApi.revokeSession(id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Revoke failed");
  }
}

async function revokeOthers(): Promise<void> {
  try {
    await accountApi.revokeOtherSessions();
    await load();
    toast.success("Other sessions revoked");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Revoke failed");
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
    toast.error(err instanceof Error ? err.message : "Could not add e-mail");
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
    toast.error(err instanceof Error ? err.message : "Confirm failed");
  }
}

async function deleteEmail(id: number): Promise<void> {
  try {
    await accountApi.deleteEmail(id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Remove failed");
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
    toast.error(err instanceof Error ? err.message : "Unlink failed");
  }
}

async function unlinkLDAPIdentity(id: number): Promise<void> {
  try {
    await accountApi.unlinkLDAPIdentity(id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Unlink failed");
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
    toast.success("LDAP account linked");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Link failed");
  } finally {
    busyIdentities.value = false;
  }
}
</script>
