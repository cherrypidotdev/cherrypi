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
  p.text-muted-foreground.text-xs(v-if="!logs.length") {{ $t("domainLog.empty") }}
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
import { useI18n } from "vue-i18n";

withDefaults(
  defineProps<{
    logs: AuditEntry[];
    showScope?: boolean;
  }>(),
  { showScope: false },
);

const KIND_LABEL_KEYS: Record<string, string> = {
  created: "domainLog.kind.created",
  updated: "domainLog.kind.updated",
  deleted: "domainLog.kind.deleted",
  state: "domainLog.kind.state",
  name: "domainLog.kind.name",
  prompt: "domainLog.kind.prompt",
  prompt_proposal_create: "domainLog.kind.proposalCreate",
  prompt_proposal_approve: "domainLog.kind.proposalApprove",
  prompt_proposal_reject: "domainLog.kind.proposalReject",
  prompt_proposal_withdraw: "domainLog.kind.proposalWithdraw",
  priority: "domainLog.kind.priority",
  size: "domainLog.kind.size",
  assignee: "domainLog.kind.assignee",
  tags: "domainLog.kind.tags",
  references: "domainLog.kind.references",
  links: "domainLog.kind.links",
  project: "domainLog.kind.project",
  model: "domainLog.kind.model",
  session: "domainLog.kind.session",
  session_lost: "domainLog.kind.sessionLost",
  participation: "domainLog.kind.participation",
  teams: "domainLog.kind.teams",
  leads: "domainLog.kind.leads",
  questionnaire: "domainLog.kind.questionnaire",
  template: "domainLog.kind.template",
  access: "domainLog.kind.access",
  member: "domainLog.kind.member",
  invitation: "domainLog.kind.invitation",
  settings: "domainLog.kind.settings",
  archived: "domainLog.kind.archived",
  unarchived: "domainLog.kind.unarchived",
  login: "domainLog.kind.login",
  login_failed: "domainLog.kind.loginFailed",
  logout: "domainLog.kind.logout",
  registered: "domainLog.kind.registered",
  password_change: "domainLog.kind.password",
  password_reset: "domainLog.kind.passwordReset",
  password_reset_requested: "domainLog.kind.resetRequested",
  magic_link: "domainLog.kind.magicLink",
  email_add: "domainLog.kind.emailAdd",
  email_remove: "domainLog.kind.emailRemove",
  email_confirm: "domainLog.kind.emailConfirm",
  totp_enable: "domainLog.kind.totpEnable",
  totp_disable: "domainLog.kind.totpDisable",
  recovery_codes: "domainLog.kind.recoveryCodes",
  session_revoke: "domainLog.kind.sessionRevoke",
  sessions_revoke: "domainLog.kind.sessionsRevoke",
  pat_create: "domainLog.kind.tokenCreated",
  pat_revoke: "domainLog.kind.tokenRevoked",
  user_create: "domainLog.kind.userCreate",
  user_update: "domainLog.kind.userUpdate",
  user_role: "domainLog.kind.roleChanged",
  user_status: "domainLog.kind.statusChanged",
  user_delete: "domainLog.kind.userDelete",
  invitation_create: "domainLog.kind.invitationCreate",
  invitation_revoke: "domainLog.kind.inviteRevoke",
  invitation_resend: "domainLog.kind.inviteResend",
  invitation_accept: "domainLog.kind.inviteAccept",
  identity_link: "domainLog.kind.identityLink",
  identity_unlink: "domainLog.kind.identityUnlink",
  provider_create: "domainLog.kind.providerAdd",
  provider_update: "domainLog.kind.providerUpdate",
  provider_delete: "domainLog.kind.providerRemove",
  auth_settings: "domainLog.kind.authSettings",
  smtp_settings: "domainLog.kind.smtpSettings",
  credential_create: "domainLog.kind.credentialAdd",
  credential_revoke: "domainLog.kind.credentialRevoke",
  token_create: "domainLog.kind.tokenCreated",
  token_revoke: "domainLog.kind.tokenRevoked",
  owner_add: "domainLog.kind.ownerAdd",
  owner_remove: "domainLog.kind.ownerRemove",
  defaults: "domainLog.kind.modelDefaults",
  limits: "domainLog.kind.limits",
  health_interval: "domainLog.kind.healthInterval",
  role: "domainLog.kind.roleChanged",
  hierarchy: "domainLog.kind.hierarchy",
};

function kindLabel(kind: string): string {
  const key = KIND_LABEL_KEYS[kind];
  return key ? t(key) : kind;
}

function actorLabel(entry: AuditEntry): string {
  return entry.actor_username || entry.actor_name || t("domainLog.system");
}

function formatRelative(at: string): string {
  return formatRelativeTime(Date.parse(at));
}

function formatAbsolute(at: string): string {
  return new Date(at).toLocaleString(locale.value);
}

function entryKey(entry: AuditEntry): string {
  return `${entry.scope}:${entry.id}`;
}

const { t, locale } = useI18n();
</script>

<i18n lang="yaml">
en:
  domainLog:
    empty: "No activity yet."
    system: "system"
    kind:
      created: "Created"
      updated: "Updated"
      deleted: "Deleted"
      state: "State"
      name: "Title"
      prompt: "Prompt"
      proposalCreate: "Proposal created"
      proposalApprove: "Proposal approved"
      proposalReject: "Proposal rejected"
      proposalWithdraw: "Proposal withdrawn"
      priority: "Priority"
      size: "Size"
      assignee: "Assignee"
      tags: "Tags"
      references: "References"
      links: "Links"
      project: "Project"
      model: "Model"
      session: "Session"
      sessionLost: "Session lost"
      participation: "Participation"
      teams: "Owning teams"
      leads: "Leads"
      questionnaire: "Questionnaire answered"
      template: "Template"
      access: "Access"
      member: "Member"
      invitation: "Invitation"
      settings: "Settings"
      archived: "Archived"
      unarchived: "Unarchived"
      login: "Signed in"
      loginFailed: "Sign-in failed"
      logout: "Signed out"
      registered: "Registered"
      password: "Password"
      passwordReset: "Password reset"
      resetRequested: "Reset requested"
      magicLink: "Magic link"
      emailAdd: "E-mail added"
      emailRemove: "E-mail removed"
      emailConfirm: "E-mail confirmed"
      totpEnable: "2FA enabled"
      totpDisable: "2FA disabled"
      recoveryCodes: "Recovery codes"
      sessionRevoke: "Session revoked"
      sessionsRevoke: "Sessions revoked"
      tokenCreated: "Token created"
      tokenRevoked: "Token revoked"
      userCreate: "User created"
      userUpdate: "User updated"
      roleChanged: "Role changed"
      statusChanged: "Status changed"
      userDelete: "User deleted"
      invitationCreate: "Invited"
      inviteRevoke: "Invite revoked"
      inviteResend: "Invite resent"
      inviteAccept: "Invite accepted"
      identityLink: "Identity linked"
      identityUnlink: "Identity unlinked"
      providerAdd: "Provider added"
      providerUpdate: "Provider updated"
      providerRemove: "Provider removed"
      authSettings: "Auth settings"
      smtpSettings: "SMTP settings"
      credentialAdd: "Credential added"
      credentialRevoke: "Credential revoked"
      ownerAdd: "Owner added"
      ownerRemove: "Owner removed"
      modelDefaults: "Model defaults changed"
      limits: "Limits changed"
      healthInterval: "Health interval changed"
      hierarchy: "Project hierarchy"
ru:
  domainLog:
    empty: "Активности пока нет."
    system: "система"
    kind:
      created: "Создано"
      updated: "Обновлено"
      deleted: "Удалено"
      state: "Состояние"
      name: "Название"
      prompt: "Промпт"
      proposalCreate: "Предложение создано"
      proposalApprove: "Предложение одобрено"
      proposalReject: "Предложение отклонено"
      proposalWithdraw: "Предложение отозвано"
      priority: "Приоритет"
      size: "Размер"
      assignee: "Исполнитель"
      tags: "Теги"
      references: "Связи"
      links: "Ссылки"
      project: "Проект"
      model: "Модель"
      session: "Сеанс"
      sessionLost: "Сеанс потерян"
      participation: "Участие"
      teams: "Команды-владельцы"
      leads: "Руководители"
      questionnaire: "Опросник заполнен"
      template: "Шаблон"
      access: "Доступ"
      member: "Участник"
      invitation: "Приглашение"
      settings: "Настройки"
      archived: "Архивировано"
      unarchived: "Разархивировано"
      login: "Вход выполнен"
      loginFailed: "Вход не удался"
      logout: "Выход выполнен"
      registered: "Зарегистрировано"
      password: "Пароль"
      passwordReset: "Пароль сброшен"
      resetRequested: "Запрошен сброс"
      magicLink: "Ссылка для входа"
      emailAdd: "E-mail добавлен"
      emailRemove: "E-mail удалён"
      emailConfirm: "E-mail подтверждён"
      totpEnable: "2FA включена"
      totpDisable: "2FA отключена"
      recoveryCodes: "Коды восстановления"
      sessionRevoke: "Сеанс отозван"
      sessionsRevoke: "Сеансы отозваны"
      tokenCreated: "Токен создан"
      tokenRevoked: "Токен отозван"
      userCreate: "Пользователь создан"
      userUpdate: "Пользователь обновлён"
      roleChanged: "Роль изменена"
      statusChanged: "Статус изменён"
      userDelete: "Пользователь удалён"
      invitationCreate: "Приглашён"
      inviteRevoke: "Приглашение отозвано"
      inviteResend: "Приглашение отправлено снова"
      inviteAccept: "Приглашение принято"
      identityLink: "Учётная запись привязана"
      identityUnlink: "Учётная запись отвязана"
      providerAdd: "Провайдер добавлен"
      providerUpdate: "Провайдер обновлён"
      providerRemove: "Провайдер удалён"
      authSettings: "Настройки аутентификации"
      smtpSettings: "Настройки SMTP"
      credentialAdd: "Учётные данные добавлены"
      credentialRevoke: "Учётные данные отозваны"
      ownerAdd: "Владелец добавлен"
      ownerRemove: "Владелец удалён"
      modelDefaults: "Модели по умолчанию изменены"
      limits: "Лимиты изменены"
      healthInterval: "Интервал проверки здоровья изменён"
      hierarchy: "Иерархия проектов"
</i18n>
