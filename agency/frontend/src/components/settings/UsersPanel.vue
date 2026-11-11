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
.flex.flex-col.gap-6
  Card
    CardHeader
      CardTitle {{ $t("admin.invite.title") }}
      CardDescription
        | {{ $t("users.invite.hint") }}
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="submitInvite")
        FieldGroup
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel {{ $t("users.invite.kind") }}
              Select(
                :model-value="invite.kind"
                @update:model-value="onKindChange(String($event))"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="username") {{ $t("users.invite.kindPassword") }}
                  SelectItem(value="email") {{ $t("users.invite.kindEmail") }}
                  SelectItem(value="oidc") {{ $t("users.invite.oidcProvider") }}
                  SelectItem(value="ldap") {{ $t("account.identities.ldapDirectory") }}
            Field
              FieldLabel {{ $t("common.username") }}
              Input(
                v-model="invite.username"
                :placeholder="$t('admin.invite.usernamePlaceholder')"
                autocomplete="off"
              )
            Field(v-if="invite.kind === 'email'")
              FieldLabel {{ $t("common.email") }}
              Input(
                v-model="invite.email"
                type="email"
                :placeholder="$t('users.invite.emailPlaceholder')"
                autocomplete="off"
              )
            Field(v-if="invite.kind === 'oidc'")
              FieldLabel {{ $t("users.invite.provider") }}
              Select(
                :model-value="String(invite.provider_id)"
                @update:model-value="invite.provider_id = Number($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(
                    v-for="provider in providers"
                    :key="provider.id"
                    :value="String(provider.id)"
                  ) {{ provider.name }}
            Field(v-if="invite.kind === 'ldap'")
              FieldLabel {{ $t("users.invite.directory") }}
              Select(
                :model-value="String(invite.provider_id)"
                @update:model-value="invite.provider_id = Number($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(
                    v-for="provider in ldapProviders"
                    :key="provider.id"
                    :value="String(provider.id)"
                  ) {{ provider.name }}
            Field(v-if="invite.kind === 'oidc' || invite.kind === 'ldap'")
              FieldLabel {{ $t("users.invite.providerUsername") }}
              Input(
                v-model="invite.external_username"
                :placeholder="$t('users.invite.providerUsernamePlaceholder')"
                autocomplete="off"
              )
            Field
              FieldLabel {{ $t("admin.role") }}
              Select(
                :model-value="invite.role"
                @update:model-value="invite.role = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="member") {{ $t("admin.roleMember") }}
                  SelectItem(value="admin") {{ $t("admin.roleAdmin") }}
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end
          Button(type="submit" :disabled="saving || !invite.username.trim()")
            Spinner(v-if="saving" data-icon="inline-start")
            SendIcon(v-else data-icon="inline-start")
            span {{ $t("users.invite.send") }}

  Card(v-if="lastResult")
    CardHeader
      CardTitle {{ $t("admin.invite.created") }}
      CardDescription {{ $t("users.invite.shareOnce") }}
    CardContent.flex.flex-col.gap-3
      .flex.flex-col.gap-1(v-if="lastResult.password")
        p.text-sm.font-medium {{ $t("users.invite.passwordFor", { username: lastResult.invitation.username }) }}
        .flex.items-center.gap-2
          code.bg-muted.flex-1.rounded-md.px-2.py-1.font-mono.text-sm.break-all {{ lastResult.password }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="copy(lastResult.password)"
          )
            CopyIcon(data-icon="inline-start")
            span {{ $t("common.copy") }}
        p.text-muted-foreground.text-xs {{ $t("users.invite.mustChange") }}
      .flex.flex-col.gap-1(v-if="lastResult.link")
        p.text-sm.font-medium {{ $t("users.invite.acceptanceLink") }}
        p.text-muted-foreground.text-xs {{ $t("users.invite.acceptanceLinkHint") }}
        .flex.items-center.gap-2
          code.bg-muted.flex-1.rounded-md.px-2.py-1.font-mono.text-xs.break-all {{ lastResult.link }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="copy(lastResult.link)"
          )
            CopyIcon(data-icon="inline-start")
            span {{ $t("common.copy") }}
        p.text-destructive.text-xs(v-if="lastResult.delivery_error")
          | {{ $t("users.invite.deliveryFailed", { error: lastResult.delivery_error }) }}
      .flex.justify-end
        Button(
          type="button"
          variant="ghost"
          size="sm"
          @click="lastResult = null"
        ) {{ $t("common.dismiss") }}

  Card
    CardHeader
      CardTitle {{ $t("nav.users") }}
      CardDescription {{ $t("users.description") }}
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Table
        TableHeader
          TableRow
            TableHead {{ $t("common.username") }}
            TableHead.w-36 {{ $t("admin.role") }}
            TableHead.w-36 {{ $t("common.status") }}
            TableHead {{ $t("admin.users.lastLogin") }}
            TableHead.w-40
        TableBody
          TableRowsSkeleton(
            v-if="loading && !users.length"
            :widths="['w-24', 'w-24', 'w-24', 'w-28', 'w-20']"
          )
          TableRow(v-for="user in users" :key="user.id")
            TableCell.font-medium
              .flex.items-center.gap-2
                span {{ user.username }}
                Badge(v-if="user.invitation" variant="secondary") {{ $t("admin.users.statusInvited") }}
            TableCell
              Select(
                :model-value="user.role"
                @update:model-value="setRole(user, String($event))"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="member") {{ $t("admin.roleMember") }}
                  SelectItem(value="admin") {{ $t("settings.tabs.admin") }}
            TableCell
              Select(
                :model-value="user.status"
                @update:model-value="setStatus(user, String($event))"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="invited") {{ $t("admin.users.statusInvitedLabel") }}
                  SelectItem(value="active") {{ $t("admin.users.statusActive") }}
                  SelectItem(value="disabled") {{ $t("common.disabled") }}
            TableCell.text-muted-foreground
              | {{ formatDate(user.last_login_at) }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="openReset(user)")
                  KeyRoundIcon(data-icon="inline-start")
                  span {{ $t("common.password") }}
                Button(variant="destructive" size="sm" @click="remove(user)")
                  Trash2Icon(data-icon="inline-start")
                  span {{ $t("common.delete") }}
          TableRow(v-if="!users.length && !loading")
            TableCell.text-muted-foreground(:colspan="5") {{ $t("admin.users.empty") }}

  Card
    CardHeader
      CardTitle {{ $t("users.invitations.title") }}
      CardDescription {{ $t("users.invitations.description") }}
    CardContent
      Table
        TableHeader
          TableRow
            TableHead {{ $t("common.username") }}
            TableHead {{ $t("common.kind") }}
            TableHead.w-28 {{ $t("common.status") }}
            TableHead {{ $t("users.invitations.expires") }}
            TableHead.w-40
        TableBody
          TableRow(v-for="invitation in invitations" :key="invitation.id")
            TableCell.font-medium {{ invitation.username }}
            TableCell
              Badge(variant="secondary") {{ invitation.kind }}
            TableCell
              Badge(
                :variant="invitation.status === 'pending' ? 'default' : 'secondary'"
              ) {{ invitation.status }}
            TableCell.text-muted-foreground {{ formatDate(invitation.expires_at) }}
            TableCell
              .flex.justify-end.gap-2
                Button(
                  v-if="invitation.status === 'pending'"
                  variant="outline"
                  size="sm"
                  @click="resend(invitation)"
                )
                  RefreshCwIcon(data-icon="inline-start")
                  span {{ $t("users.invitations.resend") }}
                Button(
                  v-if="invitation.status === 'pending'"
                  variant="destructive"
                  size="sm"
                  @click="revoke(invitation)"
                )
                  Trash2Icon(data-icon="inline-start")
                  span {{ $t("common.revoke") }}
          TableRow(v-if="!invitations.length")
            TableCell.text-muted-foreground(:colspan="5") {{ $t("users.invitations.empty") }}

  Dialog(v-model:open="resetOpen")
    DialogContent
      DialogHeader
        DialogTitle {{ $t("users.password.reset") }}
        DialogDescription {{ $t("users.password.forUser", { username: resetTarget?.username }) }}
      .flex.flex-col.gap-3
        Field
          FieldLabel {{ $t("common.newPassword") }}
          Input(
            v-model="resetPassword"
            type="password"
            autocomplete="new-password"
          )
        p.text-destructive.text-sm(v-if="error") {{ error }}
      DialogFooter
        DialogClose
          Button(type="button" variant="outline") {{ $t("common.cancel") }}
        Button(
          type="button"
          @click="submitReset"
          :disabled="saving || !resetPassword"
        ) {{ $t("users.password.set") }}
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { withProgress } from "@/composables/useProgress";
import {
  accountApi,
  type AdminUser,
  type IdentityProvider,
  type Invitation,
  type InviteKind,
  type LDAPProvider,
} from "@/lib/account";
import {
  CopyIcon,
  KeyRoundIcon,
  RefreshCwIcon,
  SendIcon,
  Trash2Icon,
} from "@lucide/vue";
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const users = ref<AdminUser[]>([]);
const invitations = ref<Invitation[]>([]);
const providers = ref<IdentityProvider[]>([]);
const ldapProviders = ref<LDAPProvider[]>([]);
const loading = ref(false);
const saving = ref(false);
const error = ref("");

const invite = reactive({
  kind: "username" as InviteKind,
  username: "",
  email: "",
  provider_id: 0,
  external_username: "",
  role: "member",
});

const lastResult = ref<{
  invitation: Invitation;
  password?: string;
  link?: string;
  delivered?: boolean;
  delivery_error?: string;
} | null>(null);

const resetOpen = ref(false);
const resetTarget = ref<AdminUser | null>(null);
const resetPassword = ref("");

onMounted(load);

function formatDate(value?: string): string {
  if (!value) return "Never";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? "—" : parsed.toLocaleString();
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    const [userResult, invitationResult, providerResult, ldapResult] =
      await Promise.all([
        accountApi.listUsers(),
        accountApi.listInvitations(),
        accountApi.listProviders(),
        accountApi.listLDAPProviders(),
      ]);
    users.value = userResult.users;
    invitations.value = invitationResult.invitations;
    providers.value = providerResult.providers;
    ldapProviders.value = ldapResult.ldap_providers;
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("settings.auth.errors.load");
  } finally {
    loading.value = false;
  }
}

function onKindChange(value: string): void {
  const kind = value as InviteKind;
  invite.kind = kind;
  invite.provider_id =
    kind === "oidc"
      ? (providers.value[0]?.id ?? 0)
      : kind === "ldap"
        ? (ldapProviders.value[0]?.id ?? 0)
        : 0;
}

async function submitInvite(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const result = await withProgress(() =>
      accountApi.createInvitation({
        kind: invite.kind,
        username: invite.username.trim(),
        email: invite.kind === "email" ? invite.email.trim() : undefined,
        provider_id:
          invite.kind === "oidc" || invite.kind === "ldap"
            ? invite.provider_id
            : undefined,
        external_username:
          invite.kind === "oidc" || invite.kind === "ldap"
            ? invite.external_username.trim()
            : undefined,
        role: invite.role,
      }),
    );
    lastResult.value = result;
    invite.username = "";
    invite.email = "";
    invite.external_username = "";
    await load();
    toast.success(t("admin.invite.created"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("users.toasts.inviteFailed");
  } finally {
    saving.value = false;
  }
}

async function setRole(user: AdminUser, role: string): Promise<void> {
  try {
    const result = await accountApi.updateUser(user.id, { role });
    Object.assign(user, result.user);
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("users.toasts.roleFailed"),
    );
    await load();
  }
}

async function setStatus(user: AdminUser, status: string): Promise<void> {
  try {
    const result = await accountApi.updateUser(user.id, { status });
    Object.assign(user, result.user);
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("users.toasts.statusFailed"),
    );
    await load();
  }
}

async function remove(user: AdminUser): Promise<void> {
  if (!window.confirm(`Delete ${user.username}? This cannot be undone.`))
    return;
  try {
    await accountApi.deleteUser(user.id);
    await load();
    toast.success(t("admin.toasts.userDeleted"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("users.toasts.deleteFailed"),
    );
  }
}

function openReset(user: AdminUser): void {
  resetTarget.value = user;
  resetPassword.value = "";
  error.value = "";
  resetOpen.value = true;
}

async function submitReset(): Promise<void> {
  if (!resetTarget.value) return;
  saving.value = true;
  try {
    await accountApi.setUserPassword(resetTarget.value.id, resetPassword.value);
    resetOpen.value = false;
    toast.success(t("users.toasts.passwordUpdated"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("users.toasts.passwordFailed");
  } finally {
    saving.value = false;
  }
}

async function revoke(invitation: Invitation): Promise<void> {
  try {
    await accountApi.revokeInvitation(invitation.id);
    await load();
    toast.success(t("users.toasts.revoked"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("users.toasts.revokeFailed"),
    );
  }
}

async function resend(invitation: Invitation): Promise<void> {
  try {
    const result = await accountApi.resendInvitation(invitation.id);
    lastResult.value = { invitation: result.invitation, link: result.link };
    toast.success(t("users.toasts.resent"));
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : t("users.toasts.resendFailed"),
    );
  }
}

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(t("common.copied"));
  } catch {
    toast.error(t("users.toasts.copyFailed"));
  }
}
</script>

<i18n lang="yaml">
en:
  users:
    invite:
      hint: "The username is reserved when the invitation is created. Username invites return a one-time password; e-mail invites mail a link and show a copyable fallback."
      kind: "Invitation kind"
      kindPassword: "Username + password"
      kindEmail: "E-mail + link"
      oidcProvider: "OIDC provider"
      provider: "Provider"
      directory: "Directory"
      providerUsername: "Provider username"
      send: "Send invitation"
      shareOnce: "Share it with the invitee; it is shown once."
      mustChange: "The user must change it on first sign-in."
      acceptanceLink: "Acceptance link"
      emailPlaceholder: "jane{'@'}example.com"
      providerUsernamePlaceholder: "directory username"
      acceptanceLinkHint: "Copy the link above and share it directly."
      deliveryFailed: "Mail delivery failed: {error}. Copy the link above and share it directly."
      passwordFor: "One-time password for {'@'}{username}"
    description: "Agency accounts, their roles and status."
    invitations:
      title: "Invitations"
      description: "Pending and past invitations."
      expires: "Expires"
      resend: "Resend"
      empty: "No invitations."
    password:
      reset: "Reset password"
      set: "Set password"
      forUser: "Set a new password for {username}."
    toasts:
      copyFailed: "Could not copy"
      resendFailed: "Could not resend"
      resent: "Invitation re-sent"
      revokeFailed: "Could not revoke"
      revoked: "Invitation revoked"
      passwordFailed: "Could not set password"
      passwordUpdated: "Password updated"
      deleteFailed: "Could not delete user"
      statusFailed: "Could not update status"
      roleFailed: "Could not update role"
      inviteFailed: "Could not invite"
ru:
  users:
    invite:
      hint: "Имя пользователя резервируется при создании приглашения. Приглашения по имени пользователя возвращают одноразовый пароль; приглашения по e-mail отправляют ссылку и показывают копируемый запасной вариант."
      kind: "Тип приглашения"
      kindPassword: "Имя пользователя + пароль"
      kindEmail: "E-mail + ссылка"
      oidcProvider: "Провайдер OIDC"
      provider: "Провайдер"
      directory: "Каталог"
      providerUsername: "Имя пользователя у провайдера"
      send: "Отправить приглашение"
      shareOnce: "Передайте его приглашённому; оно показывается один раз."
      mustChange: "Пользователь должен сменить его при первом входе."
      acceptanceLink: "Ссылка для принятия"
      emailPlaceholder: "jane{'@'}example.com"
      providerUsernamePlaceholder: "имя пользователя в каталоге"
      acceptanceLinkHint: "Скопируйте ссылку выше и передайте её напрямую."
      deliveryFailed: "Не удалось отправить письмо: {error}. Скопируйте ссылку выше и передайте её напрямую."
      passwordFor: "Одноразовый пароль для {'@'}{username}"
    description: "Аккаунты агентства, их роли и статус."
    invitations:
      title: "Приглашения"
      description: "Ожидающие и прошлые приглашения."
      expires: "Истекает"
      resend: "Отправить снова"
      empty: "Приглашений нет."
    password:
      reset: "Сбросить пароль"
      set: "Задать пароль"
      forUser: "Задайте новый пароль для {username}."
    toasts:
      copyFailed: "Не удалось скопировать"
      resendFailed: "Не удалось отправить снова"
      resent: "Приглашение отправлено снова"
      revokeFailed: "Не удалось отозвать"
      revoked: "Приглашение отозвано"
      passwordFailed: "Не удалось задать пароль"
      passwordUpdated: "Пароль обновлён"
      deleteFailed: "Не удалось удалить пользователя"
      statusFailed: "Не удалось изменить статус"
      roleFailed: "Не удалось изменить роль"
      inviteFailed: "Не удалось пригласить"
</i18n>
