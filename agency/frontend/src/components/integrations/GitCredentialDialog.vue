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
Dialog(:open="open" @update:open="onOpenChange")
  DialogScrollContent(class="sm:max-w-lg")
    DialogHeader
      DialogTitle {{ credential ? $t("git.credential.edit") : $t("git.credential.create") }}
      DialogDescription {{ $t("git.credential.description") }}
    .flex.flex-col.gap-4
      FieldGroup
        Field
          FieldLabel(for="credential-name") {{ $t("git.credential.name") }}
          Input#credential-name(v-model="draft.name" maxlength="64")
        Field
          FieldLabel {{ $t("git.credential.kind") }}
          Select(
            :model-value="draft.kind"
            @update:model-value="setKind(String($event))"
          )
            SelectTrigger.w-full
              SelectValue
            SelectContent
              SelectItem(value="token") {{ $t("git.credential.kindToken") }}
              SelectItem(value="ssh_key") {{ $t("git.credential.kindSSH") }}
        Field
          FieldLabel(for="credential-host") {{ $t("git.credential.host") }}
          .flex.items-center.gap-2
            Input#credential-host.flex-1.font-mono.text-sm(
              v-model="draft.host"
              :placeholder="$t('git.credential.hostPlaceholder')"
            )
            Input.w-24.font-mono.text-sm(
              type="number"
              :placeholder="$t('git.credential.portHint')"
              min="1"
              max="65535"
              :aria-label="$t('git.credential.port')"
              :model-value="draft.port || ''"
              @update:model-value="setPort(String($event))"
            )
          FieldDescription {{ $t("git.credential.hostHint") }}
        Field(v-if="draft.kind === 'token'")
          FieldLabel(for="credential-username") {{ $t("git.credential.username") }}
          Input#credential-username.font-mono.text-sm(
            v-model="draft.username"
            :placeholder="$t('git.credential.usernamePlaceholder')"
          )
        Field
          FieldLabel(for="credential-secret")
            | {{ draft.kind === "token" ? $t("git.credential.token") : $t("git.credential.privateKey") }}
          Textarea#credential-secret.font-mono.text-xs(
            v-if="draft.kind === 'ssh_key'"
            v-model="draft.secret"
            :placeholder="hasStoredSecret ? $t('git.credential.secretStored') : $t('git.credential.keyPlaceholder')"
            :rows="5"
          )
          Input#credential-secret(
            v-else
            v-model="draft.secret"
            type="password"
            :placeholder="hasStoredSecret ? $t('git.credential.secretStored') : $t('git.credential.tokenPlaceholder')"
            autocomplete="new-password"
          )
          FieldDescription {{ $t("git.credential.secretHint") }}
      template(v-if="draft.kind === 'ssh_key'")
        Field
          FieldLabel {{ $t("git.credential.hostKey") }}
          .flex.flex-col.gap-2
            Textarea.font-mono.text-xs(
              v-model="draft.knownHosts"
              :placeholder="$t('git.credential.hostKeyPlaceholder')"
              :rows="2"
            )
            .flex.flex-wrap.items-center.gap-2
              Button(
                type="button"
                variant="outline"
                size="sm"
                @click="scan"
                :disabled="scanning || !draft.host.trim()"
              )
                Spinner(v-if="scanning" data-icon="inline-start")
                ScanLineIcon(v-else data-icon="inline-start")
                span {{ $t("git.credential.scan") }}
              span.text-muted-foreground.text-xs {{ $t("git.credential.scanHint") }}
            ul.flex.flex-col.gap-1(v-if="scanned.length")
              li.font-mono.text-xs(
                v-for="key in scanned"
                :key="key.fingerprint"
              )
                | {{ key.type }} · {{ key.fingerprint }}
          FieldDescription {{ $t("git.credential.hostKeyHint") }}
        p.text-muted-foreground.text-xs(v-if="publicKey")
          | {{ $t("git.credential.publicKey") }}
        pre.text-muted-foreground.overflow-x-auto.text-xs(v-if="publicKey")
          | {{ publicKey }}
      p.text-destructive.text-sm(v-if="error") {{ error }}
    DialogFooter
      Button(type="button" variant="ghost" @click="close") {{ $t("common.cancel") }}
      Button(
        type="button"
        @click="save"
        :disabled="saving"
        data-testid="git-credential-save"
      )
        Spinner(v-if="saving" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ $t("common.save") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogScrollContent,
  DialogTitle,
} from "@/components/ui/dialog";
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
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import {
  integrationsApi,
  type GitCredential,
  type GitCredentialInput,
  type GitHostKey,
} from "@/lib/integrations";
import { CheckIcon, ScanLineIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

const props = defineProps<{
  open: boolean;
  credential?: GitCredential;
}>();

const emit = defineEmits<{
  (event: "update:open", open: boolean): void;
  (event: "saved", credential: GitCredential): void;
}>();

const draft = ref<GitCredentialInput>(blank());
const scanned = ref<GitHostKey[]>([]);
const scanning = ref(false);
const saving = ref(false);
const error = ref("");

const hasStoredSecret = computed(
  () => Boolean(props.credential?.hasSecret) && draft.value.secret === "",
);

const publicKey = computed(() => props.credential?.publicKey ?? "");

function blank(): GitCredentialInput {
  return {
    name: "",
    kind: "token",
    host: "",
    port: 0,
    username: "git",
    secret: "",
    keepSecret: true,
    knownHosts: "",
  };
}

function setPort(raw: string): void {
  const value = Number.parseInt(raw, 10);
  draft.value.port = Number.isFinite(value) && value > 0 ? value : 0;
}

function setKind(next: string): void {
  draft.value.kind = next === "ssh_key" ? "ssh_key" : "token";
}

function fill(): void {
  const credential = props.credential;
  error.value = "";
  scanned.value = [];
  if (!credential) {
    draft.value = blank();
    return;
  }
  draft.value = {
    id: credential.id,
    name: credential.name,
    kind: credential.kind,
    host: credential.host,
    port: credential.port,
    username: credential.username,
    secret: "",
    keepSecret: true,
    knownHosts: credential.knownHosts,
  };
}

watch(
  () => props.open,
  (open) => {
    if (open) fill();
  },
  { immediate: true },
);

function onOpenChange(open: boolean): void {
  emit("update:open", open);
}

function close(): void {
  emit("update:open", false);
}

async function scan(): Promise<void> {
  scanning.value = true;
  error.value = "";
  try {
    const result = await integrationsApi.scanHostKeys(
      draft.value.host.trim(),
      draft.value.port,
    );
    draft.value.knownHosts = result.knownHosts.trim();
    scanned.value = result.keys;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    scanning.value = false;
  }
}

async function save(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const saved = await integrationsApi.saveCredential({
      ...draft.value,
      name: draft.value.name.trim(),
      host: draft.value.host.trim(),
      username: draft.value.username.trim(),
      keepSecret: true,
    });
    emit("saved", saved);
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  git:
    credential:
      create: "New Git credential"
      edit: "Edit Git credential"
      description: "A token or an SSH key the agency uses to fetch a repository. It is sealed, never returned, and only ever sent to the host it names."
      name: "Credential name"
      kind: "Kind"
      kindToken: "HTTPS token"
      kindSSH: "SSH private key"
      host: "Host"
      hostPlaceholder: "gitea.example.com"
      port: "Port"
      portHint: "22"
      hostHint: "A bare host name: no scheme, no path. A source may only use this credential against this host."
      username: "HTTPS user"
      usernamePlaceholder: "git"
      token: "Token"
      privateKey: "Private key"
      keyPlaceholder: "-----BEGIN OPENSSH PRIVATE KEY-----"
      secretStored: "stored — leave blank to keep"
      tokenPlaceholder: "a personal access token"
      secretHint: "Sealed with the agency's key and never returned by the API."
      hostKey: "Pinned host key"
      hostKeyPlaceholder: "[gitea.example.com]:2222 ssh-ed25519 AAAA…"
      hostKeyHint: "The fetch runs with StrictHostKeyChecking=yes, so no connection is made before this line is pinned."
      scan: "Scan host keys"
      scanHint: "Asks the host, then shows the fingerprints to compare."
      publicKey: "Authorize this key on the Git host:"
ru:
  git:
    credential:
      create: "Новые Git-учётные данные"
      edit: "Изменить Git-учётные данные"
      description: "Токен или SSH-ключ, которым агентство забирает репозиторий. Он запечатан, никогда не возвращается и отправляется только на указанный хост."
      name: "Название"
      kind: "Тип"
      kindToken: "HTTPS-токен"
      kindSSH: "SSH-ключ"
      host: "Хост"
      hostPlaceholder: "gitea.example.com"
      port: "Порт"
      portHint: "22"
      hostHint: "Только имя хоста: без схемы и пути. Источник может использовать эти данные лишь для этого хоста."
      username: "HTTPS-пользователь"
      usernamePlaceholder: "git"
      token: "Токен"
      privateKey: "Приватный ключ"
      keyPlaceholder: "-----BEGIN OPENSSH PRIVATE KEY-----"
      secretStored: "сохранено — оставьте пустым, чтобы не менять"
      tokenPlaceholder: "personal access token"
      secretHint: "Запечатывается ключом агентства и никогда не возвращается через API."
      hostKey: "Закреплённый ключ хоста"
      hostKeyPlaceholder: "[gitea.example.com]:2222 ssh-ed25519 AAAA…"
      hostKeyHint: "Загрузка идёт со StrictHostKeyChecking=yes, поэтому без этой строки соединение не выполняется."
      scan: "Сканировать ключи хоста"
      scanHint: "Спрашивает хост и показывает отпечатки для сверки."
      publicKey: "Авторизуйте этот ключ на Git-хосте:"
</i18n>
