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
  DialogScrollContent(class="sm:max-w-2xl")
    DialogHeader
      DialogTitle {{ source ? $t("git.source.edit") : $t("git.source.create") }}
      DialogDescription {{ $t("git.source.description") }}
    .flex.flex-col.gap-4
      FieldGroup
        Field
          FieldLabel(for="source-slug") {{ $t("git.source.slug") }}
          .flex.items-center.gap-2
            Input#source-slug.font-mono.text-sm(
              v-model="draft.slug"
              :placeholder="$t('git.source.slugPlaceholder')"
              maxlength="32"
            )
            Badge(variant="secondary") {{ previewName }}
          FieldDescription {{ $t("git.source.slugHint", { prefix: prefix }) }}
        Field
          FieldLabel(for="source-name") {{ $t("git.source.name") }}
          Input#source-name(v-model="draft.name" maxlength="100")
          FieldDescription {{ $t("git.source.nameHint") }}
        Field
          FieldLabel(for="source-description") {{ $t("git.source.summary") }}
          Input#source-description(v-model="draft.description" maxlength="200")
        Field
          FieldLabel(for="source-url") {{ $t("git.source.url") }}
          Input#source-url.font-mono.text-sm(
            v-model="draft.repoUrl"
            :placeholder="$t('git.source.urlPlaceholder')"
          )
          FieldDescription {{ $t("git.source.urlHint") }}
        .grid.grid-cols-2.gap-3
          Field
            FieldLabel(for="source-ref") {{ $t("git.source.ref") }}
            Input#source-ref.font-mono.text-sm(
              v-model="draft.ref"
              :placeholder="$t('git.source.refPlaceholder')"
            )
          Field
            FieldLabel(for="source-credential") {{ $t("git.source.credential") }}
            Select(
              :model-value="String(draft.credentialId)"
              @update:model-value="draft.credentialId = Number($event)"
            )
              SelectTrigger.w-full
                SelectValue
              SelectContent
                SelectItem(value="0") {{ $t("git.source.noCredential") }}
                SelectItem(
                  v-for="credential in credentials"
                  :key="credential.id"
                  :value="String(credential.id)"
                ) {{ credentialLabel(credential) }}
          Field.col-span-2
            FieldLabel(for="source-glob") {{ $t("git.source.glob") }}
            Input#source-glob.font-mono.text-sm(
              v-model="draft.pathGlob"
              :placeholder="DEFAULT_GLOB"
            )
            FieldDescription {{ $t("git.source.globHint") }}
        Field
          FieldLabel {{ $t("git.source.owner") }}
          .flex.flex-wrap.items-center.gap-2
            Select(
              :model-value="draft.owner.kind"
              @update:model-value="onOwnerKind(String($event))"
            )
              SelectTrigger.w-40
                SelectValue
              SelectContent
                SelectItem(
                  v-for="option in SCOPE_KINDS"
                  :key="option.value"
                  :value="option.value"
                ) {{ $t(option.labelKey) }}
            Input.w-48(
              v-if="draft.owner.kind !== 'org'"
              v-model="draft.owner.id"
              :placeholder="$t('library.editor.ownerId')"
            )
          FieldDescription {{ $t("git.source.ownerHint") }}
        Field
          FieldLabel {{ $t("git.source.update") }}
          .flex.flex-col.gap-2
            label.flex.items-start.gap-2.text-sm
              input.mt-1(
                v-model="draft.updateMode"
                type="radio"
                value="staged"
              )
              .flex.flex-col
                span {{ $t("git.source.modeStaged") }}
                span.text-muted-foreground.text-xs {{ $t("git.source.modeStagedHint") }}
            label.flex.items-start.gap-2.text-sm
              input.mt-1(v-model="draft.updateMode" type="radio" value="auto")
              .flex.flex-col
                span {{ $t("git.source.modeAuto") }}
                span.text-muted-foreground.text-xs {{ $t("git.source.modeAutoHint") }}
          .mt-2.flex.items-center.gap-2
            FieldLabel.text-xs(for="source-interval") {{ $t("git.source.interval") }}
            Input#source-interval.w-24(
              v-model.number="intervalMinutes"
              type="number"
              min="1"
              max="1440"
            )
            span.text-muted-foreground.text-xs {{ $t("git.source.minutes") }}
          .mt-2.flex.items-center.gap-2
            Switch(
              :model-value="draft.paused"
              :aria-label="$t('git.source.paused')"
              @update:model-value="draft.paused = Boolean($event)"
            )
            .flex.flex-col
              span.text-sm {{ $t("git.source.paused") }}
              span.text-muted-foreground.text-xs {{ $t("git.source.pausedHint") }}

      .flex.flex-col.gap-2
        .flex.items-center.justify-between.gap-2
          .flex.flex-col
            span.text-sm.font-medium {{ $t("git.source.installs") }}
            span.text-muted-foreground.text-xs {{ $t("git.source.installsHint") }}
        template(v-if="draft.id")
          .flex.flex-col.gap-2(v-if="draft.installs.length")
            .flex.flex-wrap.items-center.gap-2.rounded-md.border.px-2.py-1(
              v-for="install in draft.installs"
              :key="`install-${install.scope.kind}-${install.scope.id}`"
            )
              span.text-xs.font-medium {{ scopeLabel(install.scope) }}
              Switch(
                :model-value="install.enabled"
                :aria-label="$t('library.editor.installEnabled')"
                @update:model-value="toggleInstall(install.scope, Boolean($event))"
                :disabled="installBusy"
              )
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                :aria-label="$t('library.editor.removeInstall')"
                @click="dropInstall(install.scope)"
                :disabled="installBusy"
              )
                Trash2Icon
          .flex.flex-wrap.items-center.gap-2
            Select(
              :model-value="newInstall.kind"
              @update:model-value="setNewInstallKind(String($event))"
              :disabled="installBusy"
            )
              SelectTrigger.w-32
                SelectValue
              SelectContent
                SelectItem(
                  v-for="option in SCOPE_KINDS"
                  :key="option.value"
                  :value="option.value"
                ) {{ $t(option.labelKey) }}
            Input.w-36(
              v-if="newInstall.kind !== 'org'"
              v-model="newInstall.id"
              :placeholder="$t('library.editor.ownerId')"
              :disabled="installBusy"
            )
            Button(
              type="button"
              variant="outline"
              size="sm"
              @click="addInstall"
              :disabled="installBusy || (!newInstall.id && newInstall.kind !== 'org')"
            )
              PlusIcon(data-icon="inline-start")
              span {{ $t("library.editor.addInstall") }}
        template(v-else)
          .flex.flex-col.gap-2(v-if="draft.installs.length")
            .flex.items-center.gap-2.rounded-md.border.px-2.py-1(
              v-for="(install, index) in draft.installs"
              :key="`staged-install-${index}`"
            )
              span.text-xs.font-medium {{ scopeLabel(install.scope) }}
              Switch(
                :model-value="install.enabled"
                :aria-label="$t('library.editor.installEnabled')"
                @update:model-value="install.enabled = Boolean($event)"
              )
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                :aria-label="$t('library.editor.removeInstall')"
                @click="draft.installs.splice(index, 1)"
              )
                Trash2Icon
          .flex.flex-wrap.items-center.gap-2
            Select(
              :model-value="newInstall.kind"
              @update:model-value="setNewInstallKind(String($event))"
            )
              SelectTrigger.w-32
                SelectValue
              SelectContent
                SelectItem(
                  v-for="option in SCOPE_KINDS"
                  :key="option.value"
                  :value="option.value"
                ) {{ $t(option.labelKey) }}
            Input.w-36(
              v-if="newInstall.kind !== 'org'"
              v-model="newInstall.id"
              :placeholder="$t('library.editor.ownerId')"
            )
            Button(
              type="button"
              variant="outline"
              size="sm"
              @click="stageInstall"
              :disabled="!newInstall.id && newInstall.kind !== 'org'"
            )
              PlusIcon(data-icon="inline-start")
              span {{ $t("library.editor.addInstall") }}

      .flex.flex-wrap.items-center.gap-2
        Button(
          type="button"
          variant="outline"
          size="sm"
          @click="test"
          :disabled="testing || !draft.repoUrl.trim() || !draft.slug.trim()"
          data-testid="git-source-test"
        )
          Spinner(v-if="testing" data-icon="inline-start")
          PlayIcon(v-else data-icon="inline-start")
          span {{ $t("git.source.test") }}

      template(v-if="preview")
        .flex.flex-col.gap-2.rounded-md.border.p-2
          .flex.flex-wrap.items-center.gap-2
            span.text-xs.font-medium {{ $t("git.source.previewTitle") }}
            Badge(variant="secondary") {{ preview.commit.slice(0, 7) }}
            span.text-muted-foreground.text-xs {{ preview.ref }}
          ul.flex.flex-col.divide-y
            li.flex.items-start.gap-2.py-1(
              v-for="skill in preview.skills"
              :key="skill.path"
            )
              Badge(:variant="statusVariant(skill.status)") {{ $t(statusLabel(skill.status)) }}
              .flex.min-w-0.flex-col
                span.text-sm {{ skill.name }}
                span.text-muted-foreground.truncate.text-xs {{ skill.description }}
                span.text-destructive.text-xs(v-if="skill.error") {{ skill.error }}
          p.text-destructive.text-xs(
            v-for="(problem, index) in preview.problems"
            :key="`problem-${index}`"
          ) {{ problemLabel(problem) }}

      p.text-destructive.text-sm(v-if="error") {{ error }}
    DialogFooter
      Button(type="button" variant="ghost" @click="close") {{ $t("common.cancel") }}
      Button(
        type="button"
        @click="save"
        :disabled="saving"
        data-testid="git-source-save"
      )
        Spinner(v-if="saving" data-icon="inline-start")
        CheckIcon(v-else data-icon="inline-start")
        span {{ $t("common.save") }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
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
import { Switch } from "@/components/ui/switch";
import {
  SCOPE_KINDS,
  integrationsApi,
  scopeLabel,
  type GitCredential,
  type GitSource,
  type GitSourceInput,
  type GitSourcePreview,
  type GitSourceSkill,
  type IntegrationScope,
} from "@/lib/integrations";
import { CheckIcon, PlayIcon, PlusIcon, Trash2Icon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

const DEFAULT_GLOB = ".well-known/skills/*/SKILL.md";

const STATUS_OPTIONS = [
  { value: "added", labelKey: "git.status.added" },
  { value: "updated", labelKey: "git.status.updated" },
  { value: "unchanged", labelKey: "git.status.unchanged" },
  { value: "removed", labelKey: "git.status.removed" },
  { value: "conflict", labelKey: "git.status.conflict" },
] as const;

const props = defineProps<{
  open: boolean;
  source?: GitSource;
  credentials: GitCredential[];
  defaultOwner?: IntegrationScope;
}>();

const emit = defineEmits<{
  (event: "update:open", open: boolean): void;
  (event: "saved", source: GitSource): void;
}>();

const draft = ref<GitSourceInput>(blank());
const preview = ref<GitSourcePreview | undefined>();
const testing = ref(false);
const saving = ref(false);
const installBusy = ref(false);
const error = ref("");
const newInstall = ref<IntegrationScope>({ kind: "org", id: "" });

const intervalMinutes = computed({
  get: () => Math.max(1, Math.round(draft.value.pollIntervalSeconds / 60)),
  set: (minutes: number) => {
    const bounded = Math.min(1440, Math.max(1, Number(minutes) || 15));
    draft.value.pollIntervalSeconds = bounded * 60;
  },
});

const prefix = computed(() => {
  const slug = draft.value.slug.trim();
  return slug ? `${slug}-` : "";
});

const previewName = computed(() =>
  draft.value.slug.trim() ? `${draft.value.slug.trim()}-code-review` : "",
);

function blank(): GitSourceInput {
  return {
    slug: "",
    name: "",
    description: "",
    owner: props.defaultOwner ?? { kind: "org", id: "" },
    repoUrl: "",
    ref: "",
    pathGlob: DEFAULT_GLOB,
    credentialId: 0,
    updateMode: "staged",
    pollIntervalSeconds: 900,
    paused: false,
    installs: [],
  };
}

function fill(): void {
  error.value = "";
  preview.value = undefined;
  newInstall.value = { kind: "org", id: "" };
  const source = props.source;
  if (!source) {
    draft.value = blank();
    return;
  }
  draft.value = {
    id: source.id,
    slug: source.slug,
    name: source.name,
    description: source.description,
    owner: { ...source.owner },
    repoUrl: source.repoUrl,
    ref: source.ref,
    pathGlob: source.pathGlob,
    credentialId: source.credentialId,
    updateMode: source.updateMode,
    pollIntervalSeconds: source.pollIntervalSeconds,
    paused: source.paused,
    installs: (source.installs ?? []).map((install) => ({
      scope: { ...install.scope },
      enabled: install.enabled,
    })),
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

function onOwnerKind(next: string): void {
  draft.value.owner = {
    kind: next as IntegrationScope["kind"],
    id: next === "org" ? "" : draft.value.owner.id,
  };
}

function setNewInstallKind(next: string): void {
  newInstall.value = {
    kind: next as IntegrationScope["kind"],
    id: next === "org" ? "" : newInstall.value.id,
  };
}

function stageInstall(): void {
  draft.value.installs.push({
    scope: { ...newInstall.value },
    enabled: true,
  });
  newInstall.value = { kind: "org", id: "" };
}

function credentialLabel(credential: GitCredential): string {
  return `${credential.name} · ${credential.host}`;
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

function input(): GitSourceInput {
  return {
    ...draft.value,
    slug: draft.value.slug.trim(),
    repoUrl: draft.value.repoUrl.trim(),
    ref: draft.value.ref.trim(),
    pathGlob: draft.value.pathGlob.trim(),
    installs: draft.value.id ? [] : draft.value.installs,
  };
}

async function test(): Promise<void> {
  testing.value = true;
  error.value = "";
  try {
    preview.value = await integrationsApi.previewSource(input());
  } catch (err) {
    preview.value = undefined;
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    testing.value = false;
  }
}

async function save(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const saved = await integrationsApi.saveSource(input());
    emit("saved", saved);
    emit("update:open", false);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function reloadInstall(): Promise<void> {
  if (!draft.value.id) return;
  const fresh = await integrationsApi.getSource(draft.value.id);
  draft.value.installs = (fresh.installs ?? []).map((install) => ({
    scope: { ...install.scope },
    enabled: install.enabled,
  }));
  emit("saved", fresh);
}

async function addInstall(): Promise<void> {
  const source = draft.value.id;
  if (!source) {
    stageInstall();
    return;
  }
  installBusy.value = true;
  error.value = "";
  try {
    await integrationsApi.setSourceInstall(source, newInstall.value, true);
    newInstall.value = { kind: "org", id: "" };
    await reloadInstall();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    installBusy.value = false;
  }
}

async function toggleInstall(
  scope: IntegrationScope,
  enabled: boolean,
): Promise<void> {
  const source = draft.value.id;
  if (!source) return;
  installBusy.value = true;
  error.value = "";
  try {
    await integrationsApi.setSourceInstall(source, scope, enabled);
    await reloadInstall();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    installBusy.value = false;
  }
}

async function dropInstall(scope: IntegrationScope): Promise<void> {
  const source = draft.value.id;
  if (!source) return;
  installBusy.value = true;
  error.value = "";
  try {
    await integrationsApi.removeSourceInstall(source, scope);
    await reloadInstall();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    installBusy.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  git:
    status:
      added: "new"
      updated: "changed"
      unchanged: "same"
      removed: "removed"
      conflict: "name taken"
    source:
      create: "New Git source"
      edit: "Edit Git source"
      description: "A repository the agency fetches itself. Every SKILL.md it holds becomes a library skill, named with the source prefix."
      slug: "Source slug"
      slugHint: "Lowercase letters, digits and single hyphens. Produced names are prefixed with it: {prefix}code-review."
      slugPlaceholder: "acme"
      name: "Display name"
      nameHint: "Shown in the sources list; the slug is what the produced skill names carry."
      summary: "Description"
      url: "Repository URL"
      urlHint: "https://host/owner/repo.git for a token or a public repository, or git{'@'}host:owner/repo.git for SSH."
      urlPlaceholder: "git{'@'}git.example.com:acme/skills.git"
      ref: "Ref"
      refPlaceholder: "default branch"
      credential: "Credential"
      noCredential: "Public repository"
      glob: "Files to import"
      globHint: "Relative to the repository root and ending with SKILL.md; one level per *. Each skill imports its whole directory."
      owner: "Owner"
      ownerHint: "Ownership decides who may sync and edit the source; its skills belong to the same scope."
      update: "Updates"
      modeStaged: "Review each update"
      modeStagedHint: "A fetch stages the new commit; an owner applies it after seeing the diff."
      modeAuto: "Apply updates automatically"
      modeAutoHint: "Every fetched commit lands immediately. Whoever can push to the repository can change the prompts of every task in scope."
      interval: "Check every"
      minutes: "minutes"
      paused: "Paused"
      pausedHint: "Pausing stops polling; the imported skills stay."
      installs: "Installs"
      installsHint: "Installing the source switches on every skill it produces, including ones added later."
      test: "Test and preview"
      previewTitle: "Would import"
ru:
  git:
    status:
      added: "новый"
      updated: "изменён"
      unchanged: "без изменений"
      removed: "удалён"
      conflict: "имя занято"
    source:
      create: "Новый Git-источник"
      edit: "Изменить Git-источник"
      description: "Репозиторий, который агентство забирает само. Каждый SKILL.md становится навыком библиотеки с префиксом источника."
      slug: "Слаг источника"
      slugHint: "Строчные буквы, цифры и одиночные дефисы. Имена навыков получают префикс: {prefix}code-review."
      slugPlaceholder: "acme"
      name: "Отображаемое имя"
      nameHint: "Показывается в списке источников; префикс имён навыков берётся из слага."
      summary: "Описание"
      url: "URL репозитория"
      urlHint: "https://host/owner/repo.git для токена или публичного репозитория, либо git{'@'}host:owner/repo.git для SSH."
      urlPlaceholder: "git{'@'}git.example.com:acme/skills.git"
      ref: "Ссылка"
      refPlaceholder: "ветка по умолчанию"
      credential: "Учётные данные"
      noCredential: "Публичный репозиторий"
      glob: "Какие файлы импортировать"
      globHint: "От корня репозитория, заканчивается на SKILL.md; по одному * на уровень. Каждый навык импортируется вместе с каталогом."
      owner: "Владелец"
      ownerHint: "Владение решает, кто может синхронизировать и править источник; его навыки принадлежат той же области."
      update: "Обновления"
      modeStaged: "Проверять каждое обновление"
      modeStagedHint: "Загрузка сохраняет новый коммит; владелец применяет его, посмотрев diff."
      modeAuto: "Применять обновления автоматически"
      modeAutoHint: "Каждый загруженный коммит применяется сразу. Тот, кто может пушить в репозиторий, меняет промпты всех задач области."
      interval: "Проверять каждые"
      minutes: "минут"
      paused: "Пауза"
      pausedHint: "Пауза останавливает опрос; импортированные навыки остаются."
      installs: "Установки"
      installsHint: "Установка источника включает все его навыки, включая добавленные позже."
      test: "Проверить и посмотреть"
      previewTitle: "Будет импортировано"
</i18n>
