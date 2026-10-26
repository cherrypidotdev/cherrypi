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
.flex.flex-col(class="gap-0.5")
  p.text-muted-foreground.text-xs(v-if="!logs.length") No activity yet.
  .flex.items-start.gap-2.rounded-md.px-1(
    v-for="entry in logs"
    :key="entryKey(entry)"
    class="py-1.5"
  )
    span.text-muted-foreground.shrink-0(class="mt-0.5")
      ActivityIcon(class="size-3.5")
    .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
      .flex.flex-wrap.items-center(class="gap-1.5")
        span.inline-flex.items-center.rounded-full.border.px-2.text-xs.font-medium(
          class="py-0.5"
        ) {{ kindLabel(entry.kind) }}
        span.text-muted-foreground.inline-flex.items-center.rounded-full.border.px-2.text-xs(
          v-if="showScope"
          class="py-0.5"
        ) {{ entry.scope }} · {{ entry.scope_label || entry.scope_id }}
        span.text-sm.break-words {{ entry.message }}
      .flex.flex-wrap.items-center.gap-1.text-xs(v-if="entry.field")
        span.text-muted-foreground {{ entry.field }}
        span.text-muted-foreground.line-through(v-if="entry.old_value")
          | {{ entry.old_value }}
        ArrowRightIcon.text-muted-foreground.size-3.shrink-0(
          v-if="entry.old_value || entry.new_value"
        )
        span(v-if="entry.new_value") {{ entry.new_value }}
      .text-muted-foreground.flex.flex-wrap.items-center.gap-x-1.text-xs
        span(:title="formatAbsolute(entry.at)")
          | {{ formatRelative(entry.at) }}
        span ·
        span {{ actorLabel(entry) }}
</template>

<script setup lang="ts">
import type { AuditEntry } from "@/lib/audit";
import { formatRelativeTime } from "@/lib/format";
import { ActivityIcon, ArrowRightIcon } from "@lucide/vue";

withDefaults(
  defineProps<{
    logs: AuditEntry[];
    showScope?: boolean;
  }>(),
  { showScope: false },
);

const KIND_LABELS: Record<string, string> = {
  created: "Created",
  updated: "Updated",
  deleted: "Deleted",
  state: "State",
  name: "Title",
  prompt: "Prompt",
  priority: "Priority",
  size: "Size",
  assignee: "Assignee",
  tags: "Tags",
  references: "References",
  links: "Links",
  project: "Project",
  model: "Model",
  session: "Session",
  participation: "Participation",
  teams: "Owning teams",
  leads: "Leads",
  access: "Access",
  member: "Member",
  invitation: "Invitation",
  settings: "Settings",
  archived: "Archived",
  unarchived: "Unarchived",
  login: "Signed in",
  login_failed: "Sign-in failed",
  logout: "Signed out",
  registered: "Registered",
  password_change: "Password",
  password_reset: "Password reset",
  password_reset_requested: "Reset requested",
  magic_link: "Magic link",
  email_add: "E-mail added",
  email_remove: "E-mail removed",
  email_confirm: "E-mail confirmed",
  totp_enable: "2FA enabled",
  totp_disable: "2FA disabled",
  recovery_codes: "Recovery codes",
  session_revoke: "Session revoked",
  sessions_revoke: "Sessions revoked",
  pat_create: "Token created",
  pat_revoke: "Token revoked",
  user_create: "User created",
  user_update: "User updated",
  user_role: "Role changed",
  user_status: "Status changed",
  user_delete: "User deleted",
  invitation_create: "Invited",
  invitation_revoke: "Invite revoked",
  invitation_resend: "Invite resent",
  invitation_accept: "Invite accepted",
  identity_link: "Identity linked",
  identity_unlink: "Identity unlinked",
  provider_create: "Provider added",
  provider_update: "Provider updated",
  provider_delete: "Provider removed",
  auth_settings: "Auth settings",
  smtp_settings: "SMTP settings",
  credential_create: "Credential added",
  credential_revoke: "Credential revoked",
  token_create: "Token created",
  token_revoke: "Token revoked",
  owner_add: "Owner added",
  owner_remove: "Owner removed",
  defaults: "Model defaults changed",
  limits: "Limits changed",
  role: "Role changed",
  hierarchy: "Project hierarchy",
};

function kindLabel(kind: string): string {
  return KIND_LABELS[kind] ?? kind;
}

function actorLabel(entry: AuditEntry): string {
  return entry.actor_username || entry.actor_name || "system";
}

function formatRelative(at: string): string {
  return formatRelativeTime(Date.parse(at));
}

function formatAbsolute(at: string): string {
  return new Date(at).toLocaleString();
}

function entryKey(entry: AuditEntry): string {
  return `${entry.scope}:${entry.id}`;
}
</script>
