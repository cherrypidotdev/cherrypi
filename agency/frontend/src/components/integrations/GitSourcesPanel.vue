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
.flex.flex-col.gap-4(data-testid="git-sources-panel")
  Card(v-if="!git || !git.enabled" data-testid="git-enable-card")
    CardHeader
      CardTitle.flex.items-center.gap-2
        GitBranchIcon.size-4
        span {{ $t("git.enable.title") }}
      CardDescription {{ $t("git.enable.description") }}
    CardContent.flex.flex-col.gap-4
      .flex.flex-col.gap-1
        .flex.items-center.gap-2.text-sm(v-for="cli in clis" :key="cli.name")
          CheckIcon.size-4.text-emerald-500(v-if="cli.status?.ok")
          XIcon.text-destructive.size-4(v-else)
          span.font-mono.text-xs {{ cli.name }}
          span.text-muted-foreground.truncate.text-xs(v-if="cli.status?.ok") {{ cli.status.path }} · {{ cli.status.version }}
          span.text-destructive.text-xs(v-else) {{ $t("git.enable.missing") }}
      p.text-muted-foreground.text-xs {{ $t("git.enable.hint") }}
      .flex.flex-wrap.items-center.gap-2
        Button(
          v-if="admin"
          type="button"
          @click="enable"
          :disabled="enabling || !ready"
          data-testid="git-enable"
        )
          Spinner(v-if="enabling" data-icon="inline-start")
          CheckIcon(v-else data-icon="inline-start")
          span {{ $t("git.enable.action") }}
        p.text-muted-foreground.text-xs(v-else)
          | {{ $t("git.enable.adminOnly") }}
      p.text-destructive.text-sm(v-if="error") {{ error }}

  Card(v-else-if="!git.ready" data-testid="git-unavailable-card")
    CardHeader
      CardTitle {{ $t("git.enable.unavailableTitle") }}
      CardDescription {{ git.message }}
    CardContent
      .flex.items-center.gap-2.text-sm(v-for="cli in clis" :key="cli.name")
        CheckIcon.size-4.text-emerald-500(v-if="cli.status?.ok")
        XIcon.text-destructive.size-4(v-else)
        span.font-mono.text-xs {{ cli.name }}
        span.text-muted-foreground.truncate.text-xs {{ cli.status?.path || $t("git.enable.missing") }}

  template(v-else)
    Card
      CardHeader
        .flex.flex-wrap.items-center.justify-between.gap-2
          .flex.flex-col
            CardTitle {{ $t("git.sources.title") }}
            CardDescription {{ $t("git.sources.description") }}
          Button(
            type="button"
            size="sm"
            @click="startCreate"
            data-testid="git-source-new"
          )
            PlusIcon(data-icon="inline-start")
            span {{ $t("git.sources.new") }}
      CardContent.flex.flex-col.gap-3
        p.text-muted-foreground.text-xs(v-if="!visibleSources.length")
          | {{ $t("git.sources.empty") }}
        .flex.flex-col.gap-3(v-else)
          .flex.flex-col.gap-2.rounded-md.border.p-3(
            v-for="source in visibleSources"
            :key="source.id"
            :data-testid="`git-source-${source.id}`"
          )
            .flex.flex-wrap.items-start.gap-2
              .flex.min-w-0.flex-1.flex-col.gap-1
                .flex.flex-wrap.items-center.gap-2
                  Badge.font-mono(variant="secondary") {{ source.slug }}-
                  span.text-sm.font-medium {{ source.name || source.slug }}
                  Badge(variant="outline")
                    | {{ source.updateMode === "auto" ? $t("git.sources.modeAuto") : $t("git.sources.modeStaged") }}
                  Badge(
                    :variant="stateVariant(source)"
                    :data-testid="`git-source-state-${source.id}`"
                  ) {{ stateLabel(source) }}
                  Badge(v-if="source.paused" variant="outline") {{ $t("git.sources.paused") }}
                a.text-muted-foreground.truncate.font-mono.text-xs(
                  rel="noreferrer"
                  :href="source.repoUrl"
                  target="_blank"
                ) {{ source.repoUrl }}
                .flex.flex-wrap.items-center.gap-2.text-xs
                  span.text-muted-foreground {{ $t("git.sources.skills", { count: source.skillCount }) }}
                  span.text-muted-foreground(v-if="source.syncedCommit")
                    | · {{ source.syncedCommit.slice(0, 7) }}
                  span.text-muted-foreground(v-if="source.syncedAt")
                    | · {{ $t("git.sources.synced", { when: relative(source.syncedAt) }) }}
                  span.text-muted-foreground
                    | · {{ $t("library.ownedBy", { scope: scopeLabel(source.owner) }) }}
                .flex.flex-wrap.items-center.gap-1
                  Badge(
                    v-for="install in source.installs"
                    :key="`${source.id}-${install.scope.kind}-${install.scope.id}`"
                    :variant="install.enabled ? 'secondary' : 'outline'"
                  )
                    CheckIcon(v-if="install.enabled" data-icon="inline-start")
                    XIcon(v-else data-icon="inline-start")
                    span {{ scopeLabel(install.scope) }}
              .flex.flex-wrap.items-center.gap-2
                Button(
                  type="button"
                  variant="outline"
                  size="sm"
                  @click="sync(source)"
                  :disabled="busy === source.id"
                  :data-testid="`git-source-sync-${source.id}`"
                )
                  Spinner(v-if="busy === source.id" data-icon="inline-start")
                  RefreshCwIcon(v-else data-icon="inline-start")
                  span {{ $t("git.sources.sync") }}
                Button(
                  v-if="source.pending.length"
                  type="button"
                  size="sm"
                  @click="apply(source)"
                  :data-testid="`git-source-apply-${source.id}`"
                )
                  CheckIcon(data-icon="inline-start")
                  span {{ $t("git.sources.apply", { count: pendingCount(source) }) }}
                Button(
                  type="button"
                  variant="ghost"
                  size="sm"
                  @click="startEdit(source)"
                )
                  PencilIcon(data-icon="inline-start")
                  span {{ $t("common.edit") }}
                Button(
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="$t('common.delete')"
                  @click="remove(source)"
                )
                  Trash2Icon
            p.text-destructive.text-xs(
              v-if="source.syncError"
              :data-testid="`git-source-error-${source.id}`"
            ) {{ source.syncError }}
            p.text-destructive.text-xs(
              v-for="(problem, index) in source.problems"
              :key="`problem-${index}`"
            ) {{ problemLabel(problem) }}
            template(v-if="source.pending.length")
              .flex.flex-col.divide-y.rounded-md.border
                .flex.items-start.gap-2.px-2.py-1(
                  v-for="skill in source.pending"
                  :key="skill.path"
                )
                  Badge(:variant="statusVariant(skill.status)") {{ $t(statusLabel(skill.status)) }}
                  .flex.min-w-0.flex-col
                    span.font-mono.text-xs {{ skill.name }}
                    span.text-muted-foreground.truncate.text-xs {{ skill.description }}
                    span.text-destructive.text-xs(v-if="skill.error") {{ skill.error }}

    Card(v-if="admin" data-testid="git-credentials-card")
      CardHeader
        .flex.flex-wrap.items-center.justify-between.gap-2
          .flex.flex-col
            CardTitle {{ $t("git.credentials.title") }}
            CardDescription {{ $t("git.credentials.description") }}
          Button(type="button" size="sm" @click="startCredential()")
            PlusIcon(data-icon="inline-start")
            span {{ $t("git.credentials.new") }}
      CardContent.flex.flex-col.gap-2
        p.text-muted-foreground.text-xs(v-if="!credentials.length")
          | {{ $t("git.credentials.empty") }}
        .flex.flex-wrap.items-center.gap-2.rounded-md.border.px-2.py-1(
          v-for="credential in credentials"
          :key="credential.id"
        )
          KeyRoundIcon.size-4
          .flex.min-w-0.flex-1.flex-col
            .flex.flex-wrap.items-center.gap-2
              span.text-sm.font-medium {{ credential.name }}
              Badge(variant="outline")
                | {{ credential.kind === "token" ? $t("git.credential.kindToken") : $t("git.credential.kindSSH") }}
              span.text-muted-foreground.font-mono.text-xs {{ credential.host }}
            span.text-muted-foreground.truncate.font-mono.text-xs(
              v-if="credential.fingerprint"
            ) {{ credential.fingerprint }}
            span.text-muted-foreground.text-xs
              | {{ $t("git.credentials.usedBy", { count: credential.sources }) }}
          Button(
            type="button"
            variant="ghost"
            size="sm"
            @click="startCredential(credential)"
          )
            PencilIcon(data-icon="inline-start")
            span {{ $t("common.edit") }}
          Button(
            type="button"
            variant="ghost"
            size="icon-sm"
            :aria-label="$t('common.delete')"
            @click="removeCredential(credential)"
            :disabled="credential.sources > 0"
          )
            Trash2Icon

  GitSourceDialog(
    v-model:open="sourceDialog"
    :source="editing"
    :credentials="credentials"
    :default-owner="scope"
    @saved="onSaved"
  )
  GitCredentialDialog(
    v-model:open="credentialDialog"
    :credential="editingCredential"
    @saved="onCredentialSaved"
  )
</template>

<script setup lang="ts">
import GitCredentialDialog from "@/components/integrations/GitCredentialDialog.vue";
import GitSourceDialog from "@/components/integrations/GitSourceDialog.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import {
  integrationsApi,
  scopeLabel,
  type GitCLIStatus,
  type GitCredential,
  type GitSource,
  type GitSourceSkill,
  type GitSourcesStatus,
  type IntegrationScope,
} from "@/lib/integrations";
import {
  CheckIcon,
  GitBranchIcon,
  KeyRoundIcon,
  PencilIcon,
  PlusIcon,
  RefreshCwIcon,
  Trash2Icon,
  XIcon,
} from "@lucide/vue";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  git?: GitSourcesStatus;
  sources: GitSource[];
  credentials: GitCredential[];
  admin: boolean;
  scope?: IntegrationScope;
}>();

const emit = defineEmits<{
  (event: "refresh"): void;
}>();

const { t } = useI18n();

const STATUS_OPTIONS = [
  { value: "added", labelKey: "git.status.added" },
  { value: "updated", labelKey: "git.status.updated" },
  { value: "unchanged", labelKey: "git.status.unchanged" },
  { value: "removed", labelKey: "git.status.removed" },
  { value: "conflict", labelKey: "git.status.conflict" },
] as const;

const error = ref("");
const busy = ref(0);
const enabling = ref(false);
const sourceDialog = ref(false);
const credentialDialog = ref(false);
const editing = ref<GitSource | undefined>();
const editingCredential = ref<GitCredential | undefined>();

const clis = computed<{ name: string; status?: GitCLIStatus }[]>(() => [
  { name: "git", status: props.git?.git },
  { name: "ssh", status: props.git?.ssh },
]);

const ready = computed(
  () => Boolean(props.git?.git.ok) && Boolean(props.git?.ssh.ok),
);

const visibleSources = computed(() => {
  const scope = props.scope;
  if (!scope) return props.sources;
  return props.sources.filter(
    (source) =>
      (source.owner.kind === scope.kind && source.owner.id === scope.id) ||
      source.installs.some(
        (install) =>
          (install.scope.kind === scope.kind &&
            install.scope.id === scope.id) ||
          install.scope.kind === "org",
      ),
  );
});

function pendingCount(source: GitSource): number {
  return source.pending.filter(
    (skill) => skill.status !== "unchanged" && skill.status !== "removed",
  ).length;
}

function stateLabel(source: GitSource): string {
  switch (source.syncState) {
    case "ok":
      return t("git.sources.stateOk");
    case "pending":
      return t("git.sources.statePending");
    case "error":
      return t("git.sources.stateError");
    default:
      return t("git.sources.stateNew");
  }
}

function stateVariant(
  source: GitSource,
): "default" | "secondary" | "outline" | "destructive" {
  if (source.syncState === "error") return "destructive";
  if (source.syncState === "pending") return "default";
  if (source.syncState === "ok") return "secondary";
  return "outline";
}

function statusLabel(status: GitSourceSkill["status"]): string {
  return (
    STATUS_OPTIONS.find((option) => option.value === status)?.labelKey ??
    "git.status.unchanged"
  );
}

function problemLabel(problem: { path: string; message: string }): string {
  return problem.path ? `${problem.path}: ${problem.message}` : problem.message;
}

function statusVariant(
  status: GitSourceSkill["status"],
): "default" | "secondary" | "outline" | "destructive" {
  if (status === "conflict") return "destructive";
  if (status === "removed") return "outline";
  if (status === "added") return "default";
  return "secondary";
}

function relative(raw: string): string {
  if (!raw) return "";
  const when = new Date(raw);
  if (Number.isNaN(when.getTime())) return "";
  const minutes = Math.round((Date.now() - when.getTime()) / 60000);
  if (minutes < 1) return t("git.sources.justNow");
  if (minutes < 60) return t("git.sources.minutesAgo", { count: minutes });
  const hours = Math.round(minutes / 60);
  if (hours < 48) return t("git.sources.hoursAgo", { count: hours });
  return when.toLocaleDateString();
}

async function enable(): Promise<void> {
  enabling.value = true;
  error.value = "";
  try {
    await integrationsApi.setGitSources(true);
    emit("refresh");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    enabling.value = false;
  }
}

function startCreate(): void {
  editing.value = undefined;
  sourceDialog.value = true;
}

function startEdit(source: GitSource): void {
  editing.value = source;
  sourceDialog.value = true;
}

function startCredential(credential?: GitCredential): void {
  editingCredential.value = credential;
  credentialDialog.value = true;
}

async function sync(source: GitSource): Promise<void> {
  busy.value = source.id;
  error.value = "";
  try {
    await integrationsApi.syncSource(source.id);
    emit("refresh");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = 0;
  }
}

async function apply(source: GitSource): Promise<void> {
  busy.value = source.id;
  error.value = "";
  try {
    await integrationsApi.applySource(source.id);
    emit("refresh");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = 0;
  }
}

async function remove(source: GitSource): Promise<void> {
  if (
    !window.confirm(
      t("git.sources.deleteConfirm", {
        slug: source.slug,
        count: source.skillCount,
      }),
    )
  ) {
    return;
  }
  error.value = "";
  try {
    await integrationsApi.deleteSource(source.id);
    emit("refresh");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function removeCredential(credential: GitCredential): Promise<void> {
  if (
    !window.confirm(
      t("git.credentials.deleteConfirm", { name: credential.name }),
    )
  ) {
    return;
  }
  error.value = "";
  try {
    await integrationsApi.deleteCredential(credential.id);
    emit("refresh");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

function onSaved(): void {
  emit("refresh");
}

function onCredentialSaved(): void {
  emit("refresh");
}
</script>

<i18n lang="yaml">
en:
  git:
    enable:
      title: "Git sources"
      description: "Import skills from a repository: the agency fetches it itself, so no worker ever holds a Git credential."
      action: "Enable Git sources"
      hint: "The feature runs the agency's own git and ssh. It stays off until an administrator enables it."
      missing: "not found"
      adminOnly: "Ask an agency administrator to enable this feature."
      unavailableTitle: "The feature is enabled but cannot fetch"
    sources:
      title: "Git sources"
      description: "A repository fetched into ordinary skills. Every SKILL.md becomes an entry named with the source prefix."
      new: "New source"
      empty: "No Git sources yet."
      modeAuto: "auto-update"
      modeStaged: "review updates"
      paused: "paused"
      skills: "{count} skills"
      synced: "synced {when}"
      sync: "Sync now"
      apply: "Apply {count}"
      stateOk: "up to date"
      statePending: "update available"
      stateError: "failed"
      stateNew: "not synced"
      justNow: "just now"
      minutesAgo: "{count}m ago"
      hoursAgo: "{count}h ago"
      deleteConfirm: "Delete the source {slug} and its {count} skills?"
    credentials:
      title: "Git credentials"
      description: "Tokens and SSH keys the agency fetches with. Sealed, host-bound, and never returned by the API."
      new: "New credential"
      empty: "No credentials. A public repository needs none."
      usedBy: "used by {count} sources"
      deleteConfirm: "Delete the credential {name}?"
ru:
  git:
    enable:
      title: "Git-источники"
      description: "Импорт навыков из репозитория: агентство забирает его само, поэтому ни один воркер не держит Git-учётные данные."
      action: "Включить Git-источники"
      hint: "Функция запускает собственные git и ssh агентства. Она выключена, пока администратор её не включит."
      missing: "не найден"
      adminOnly: "Попросите администратора агентства включить эту функцию."
      unavailableTitle: "Функция включена, но загрузить не может"
    sources:
      title: "Git-источники"
      description: "Репозиторий, превращённый в обычные навыки. Каждый SKILL.md становится записью с префиксом источника."
      new: "Новый источник"
      empty: "Git-источников пока нет."
      modeAuto: "автообновление"
      modeStaged: "проверка обновлений"
      paused: "пауза"
      skills: "навыков: {count}"
      synced: "синхронизировано {when}"
      sync: "Синхронизировать"
      apply: "Применить {count}"
      stateOk: "актуально"
      statePending: "есть обновление"
      stateError: "ошибка"
      stateNew: "не синхронизировано"
      justNow: "только что"
      minutesAgo: "{count} мин назад"
      hoursAgo: "{count} ч назад"
      deleteConfirm: "Удалить источник {slug} и его навыки ({count})?"
    credentials:
      title: "Git-учётные данные"
      description: "Токены и SSH-ключи, которыми загружает агентство. Запечатаны, привязаны к хосту и не возвращаются через API."
      new: "Новые данные"
      empty: "Учётных данных нет. Публичному репозиторию они не нужны."
      usedBy: "источников: {count}"
      deleteConfirm: "Удалить учётные данные {name}?"
</i18n>
