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
      CardTitle Invite a user
      CardDescription
        | The username is reserved when the invitation is created. Username
        | invites return a one-time password; e-mail invites mail a link and
        | show a copyable fallback.
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="submitInvite")
        FieldGroup
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel Invitation kind
              Select(
                :model-value="invite.kind"
                @update:model-value="onKindChange(String($event))"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="username") Username + password
                  SelectItem(value="email") E-mail + link
                  SelectItem(value="oidc") OIDC provider
                  SelectItem(value="ldap") LDAP directory
            Field
              FieldLabel Username
              Input(
                v-model="invite.username"
                placeholder="jane"
                autocomplete="off"
              )
            Field(v-if="invite.kind === 'email'")
              FieldLabel E-mail
              Input(
                v-model="invite.email"
                type="email"
                placeholder="jane@example.com"
                autocomplete="off"
              )
            Field(v-if="invite.kind === 'oidc'")
              FieldLabel Provider
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
              FieldLabel Directory
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
              FieldLabel Provider username
              Input(
                v-model="invite.external_username"
                placeholder="directory username"
                autocomplete="off"
              )
            Field
              FieldLabel Role
              Select(
                :model-value="invite.role"
                @update:model-value="invite.role = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="member") Member
                  SelectItem(value="admin") Administrator
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end
          Button(type="submit" :disabled="saving || !invite.username.trim()")
            Spinner(v-if="saving" data-icon="inline-start")
            SendIcon(v-else data-icon="inline-start")
            span Send invitation

  Card(v-if="lastResult")
    CardHeader
      CardTitle Invitation created
      CardDescription Share it with the invitee; it is shown once.
    CardContent.flex.flex-col.gap-3
      .flex.flex-col.gap-1(v-if="lastResult.password")
        p.text-sm.font-medium One-time password for @{{ lastResult.invitation.username }}
        .flex.items-center.gap-2
          code.bg-muted.flex-1.rounded-md.px-2.py-1.font-mono.text-sm.break-all {{ lastResult.password }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="copy(lastResult.password)"
          )
            CopyIcon(data-icon="inline-start")
            span Copy
        p.text-muted-foreground.text-xs The user must change it on first sign-in.
      .flex.flex-col.gap-1(v-if="lastResult.link")
        p.text-sm.font-medium Acceptance link
        .flex.items-center.gap-2
          code.bg-muted.flex-1.rounded-md.px-2.py-1.font-mono.text-xs.break-all {{ lastResult.link }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="copy(lastResult.link)"
          )
            CopyIcon(data-icon="inline-start")
            span Copy
        p.text-destructive.text-xs(v-if="lastResult.delivery_error")
          | Mail delivery failed: {{ lastResult.delivery_error }}. Copy the link
          | above and share it directly.
      .flex.justify-end
        Button(
          type="button"
          variant="ghost"
          size="sm"
          @click="lastResult = null"
        ) Dismiss

  Card
    CardHeader
      CardTitle Users
      CardDescription Agency accounts, their roles and status.
    CardContent
      p.text-destructive.text-sm(v-if="error") {{ error }}
      Table
        TableHeader
          TableRow
            TableHead Username
            TableHead.w-36 Role
            TableHead.w-36 Status
            TableHead Last login
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
                Badge(v-if="user.invitation" variant="secondary") invited
            TableCell
              Select(
                :model-value="user.role"
                @update:model-value="setRole(user, String($event))"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="member") Member
                  SelectItem(value="admin") Admin
            TableCell
              Select(
                :model-value="user.status"
                @update:model-value="setStatus(user, String($event))"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="invited") Invited
                  SelectItem(value="active") Active
                  SelectItem(value="disabled") Disabled
            TableCell.text-muted-foreground
              | {{ formatDate(user.last_login_at) }}
            TableCell
              .flex.justify-end.gap-2
                Button(variant="outline" size="sm" @click="openReset(user)")
                  KeyRoundIcon(data-icon="inline-start")
                  span Password
                Button(variant="destructive" size="sm" @click="remove(user)")
                  Trash2Icon(data-icon="inline-start")
                  span Delete
          TableRow(v-if="!users.length && !loading")
            TableCell.text-muted-foreground(:colspan="5") No users.

  Card
    CardHeader
      CardTitle Invitations
      CardDescription Pending and past invitations.
    CardContent
      Table
        TableHeader
          TableRow
            TableHead Username
            TableHead Kind
            TableHead.w-28 Status
            TableHead Expires
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
                  span Resend
                Button(
                  v-if="invitation.status === 'pending'"
                  variant="destructive"
                  size="sm"
                  @click="revoke(invitation)"
                )
                  Trash2Icon(data-icon="inline-start")
                  span Revoke
          TableRow(v-if="!invitations.length")
            TableCell.text-muted-foreground(:colspan="5") No invitations.

  Dialog(v-model:open="resetOpen")
    DialogContent
      DialogHeader
        DialogTitle Reset password
        DialogDescription Set a new password for {{ resetTarget?.username }}.
      .flex.flex-col.gap-3
        Field
          FieldLabel New password
          Input(
            v-model="resetPassword"
            type="password"
            autocomplete="new-password"
          )
        p.text-destructive.text-sm(v-if="error") {{ error }}
      DialogFooter
        DialogClose
          Button(type="button" variant="outline") Cancel
        Button(
          type="button"
          @click="submitReset"
          :disabled="saving || !resetPassword"
        ) Set password
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
import { toast } from "vue3-toastify";

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
    error.value = err instanceof Error ? err.message : "Failed to load";
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
    toast.success("Invitation created");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Could not invite";
  } finally {
    saving.value = false;
  }
}

async function setRole(user: AdminUser, role: string): Promise<void> {
  try {
    const result = await accountApi.updateUser(user.id, { role });
    Object.assign(user, result.user);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Could not update role");
    await load();
  }
}

async function setStatus(user: AdminUser, status: string): Promise<void> {
  try {
    const result = await accountApi.updateUser(user.id, { status });
    Object.assign(user, result.user);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Could not update status");
    await load();
  }
}

async function remove(user: AdminUser): Promise<void> {
  if (!window.confirm(`Delete ${user.username}? This cannot be undone.`))
    return;
  try {
    await accountApi.deleteUser(user.id);
    await load();
    toast.success("User deleted");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Could not delete user");
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
    toast.success("Password updated");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Could not set password";
  } finally {
    saving.value = false;
  }
}

async function revoke(invitation: Invitation): Promise<void> {
  try {
    await accountApi.revokeInvitation(invitation.id);
    await load();
    toast.success("Invitation revoked");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Could not revoke");
  }
}

async function resend(invitation: Invitation): Promise<void> {
  try {
    const result = await accountApi.resendInvitation(invitation.id);
    lastResult.value = { invitation: result.invitation, link: result.link };
    toast.success("Invitation re-sent");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Could not resend");
  }
}

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Copied");
  } catch {
    toast.error("Could not copy");
  }
}
</script>
