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
.flex.flex-col.gap-4(data-testid="integrations-panel")
  .flex.flex-wrap.items-center.gap-2
    .bg-muted.inline-flex.items-center.gap-1.rounded-lg.border(class="p-0.5")
      button.rounded-md.px-2.text-xs.font-medium.transition-colors(
        v-for="option in KINDS"
        :key="option.value"
        type="button"
        class="py-0.5"
        :class="kind === option.value ? 'bg-card shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'"
        @click="kind = option.value"
        :data-testid="`integrations-kind-${option.value}`"
      ) {{ $t(option.labelKey) }} · {{ countOf(option.value) }}
    Select(
      v-if="!scope"
      :model-value="ownerFilter"
      @update:model-value="ownerFilter = String($event)"
    )
      SelectTrigger.w-44(:aria-label="$t('library.ownerFilter')")
        SelectValue
      SelectContent
        SelectItem(value="all") {{ $t("library.allOwners") }}
        SelectItem(
          v-for="option in SCOPE_KINDS"
          :key="option.value"
          :value="option.value"
        ) {{ $t(option.labelKey) }}
    span.text-muted-foreground.ml-auto.text-xs.tabular-nums(
      v-if="kind !== 'git'"
    )
      | {{ $t("library.counts", { servers: servers.length, skills: skills.length }) }}
    Button(
      v-if="kind !== 'git'"
      type="button"
      size="sm"
      @click="startCreate"
      data-testid="integrations-new"
    )
      PlusIcon(data-icon="inline-start")
      span {{ $t("library.new") }}

  GitSourcesPanel(
    v-if="kind === 'git'"
    :git="git"
    :sources="sources"
    :credentials="credentials"
    :admin="admin"
    :scope="scope"
    @refresh="load"
  )

  p.text-muted-foreground.text-xs(v-if="loading") {{ $t("library.loading") }}
  p.text-destructive.text-xs(v-else-if="error") {{ error }}
  Empty(v-if="kind !== 'git' && !loading && !error && !rows.length")
    EmptyTitle {{ $t("library.empty.title") }}
    EmptyDescription {{ $t("library.empty.description") }}

  .flex.flex-col.divide-y(
    v-if="kind !== 'git' && !loading && !error && rows.length"
  )
    button.flex.w-full.items-start.gap-3.py-2.text-left(
      v-for="row in rows"
      :key="`${row.kind}:${row.id}`"
      type="button"
      @click="select(row)"
      :data-testid="`integrations-row-${row.kind}-${row.id}`"
    )
      component.size-4.shrink-0(
        class="mt-0.5"
        :is="row.kind === 'mcp' ? ServerIcon : FileTextIcon"
      )
      .flex.min-w-0.flex-1.flex-col
        .flex.flex-wrap.items-center.gap-2
          span.text-sm.font-medium {{ row.name }}
          Badge(variant="outline")
            | {{ $t(row.kind === "mcp" ? "library.kind.mcp" : "library.kind.skill") }}
          Badge(v-if="row.kind === 'mcp'" variant="secondary")
            | {{ $t("library.agencyTransport") }}
          Badge(v-if="!row.enabled" variant="destructive") {{ $t("library.policy") }}
          Badge(v-if="rowSource(row)" variant="secondary")
            GitBranchIcon(data-icon="inline-start")
            span {{ rowSource(row)?.slug }}
          Badge(v-if="rowSource(row)?.missing" variant="outline") {{ $t("library.importedMissing") }}
        span.text-muted-foreground.truncate.text-xs {{ row.description }}
      .flex.shrink-0.flex-col.items-end.gap-1
        span.text-muted-foreground.text-xs
          | {{ $t("library.ownedBy", { scope: scopeLabel(row.owner) }) }}
        .flex.flex-wrap.items-center.justify-end.gap-1
          Badge(
            v-for="install in row.installs"
            :key="`${row.kind}-${row.id}-${install.scope.kind}-${install.scope.id}`"
            :variant="install.enabled ? 'secondary' : 'outline'"
            :title="install.enabled ? $t('library.installedOn') : $t('library.installedOff')"
          )
            CheckIcon(v-if="install.enabled" data-icon="inline-start")
            XIcon(v-else data-icon="inline-start")
            span {{ scopeLabel(install.scope) }}

  Card(v-if="draft && kind !== 'git'" data-testid="integrations-editor")
    CardHeader
      CardTitle {{ draft.id ? $t("library.editor.edit") : $t("library.editor.create") }}
      CardDescription {{ $t("library.editor.description") }}
    CardContent.flex.flex-col.gap-4
      FieldGroup
        Field
          FieldLabel(for="integration-name") {{ $t("library.editor.name") }}
          Input#integration-name(v-model="draft.name" maxlength="64")
          FieldDescription {{ $t("library.editor.nameHint") }}
        Field
          FieldLabel(for="integration-description") {{ $t("library.editor.summary") }}
          Input#integration-description(
            v-model="draft.description"
            maxlength="200"
          )
        Field
          FieldLabel {{ $t("library.editor.owner") }}
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
          FieldDescription {{ $t("library.editor.ownerHint") }}

      template(v-if="kind === 'mcp'")
        Field
          FieldLabel(for="integration-url") {{ $t("library.editor.url") }}
          Input#integration-url.font-mono.text-sm(
            v-model="draft.url"
            :placeholder="$t('library.editor.urlPlaceholder')"
          )
          FieldDescription {{ $t("library.editor.urlHint") }}
        Field
          FieldLabel {{ $t("library.editor.headers") }}
          .flex.flex-col.gap-2
            .flex.items-center.gap-2(
              v-for="(pair, index) in valueRows"
              :key="`value-${index}`"
            )
              Input.w-40.font-mono.text-xs(
                v-model="pair.key"
                :placeholder="$t('library.editor.valueKey')"
              )
              Select(
                :model-value="pair.mode"
                @update:model-value="pair.mode = valueMode(String($event))"
              )
                SelectTrigger.w-32
                  SelectValue
                SelectContent
                  SelectItem(value="value") {{ $t("library.editor.valueLiteral") }}
                  SelectItem(value="env") {{ $t("library.editor.valueEnv") }}
                  SelectItem(value="secret") {{ $t("library.editor.valueSecret") }}
              Input.flex-1.font-mono.text-xs(
                v-model="pair.value"
                :placeholder="pair.hasSecret && pair.value === '' ? $t('library.editor.secretStored') : pair.mode === 'env' ? $t('library.editor.envPlaceholder') : $t('library.editor.valuePlaceholder')"
              )
              Button(
                type="button"
                variant="ghost"
                size="icon-sm"
                :aria-label="$t('library.editor.removeValue')"
                @click="removeValue(index)"
              )
                Trash2Icon
            Button(type="button" variant="outline" size="sm" @click="addValue")
              PlusIcon(data-icon="inline-start")
              span {{ $t("library.editor.addValue") }}
          FieldDescription {{ $t("library.editor.valuesHint") }}
        Field
          FieldLabel(for="integration-tools") {{ $t("library.editor.tools") }}
          Input#integration-tools.font-mono.text-sm(
            :placeholder="$t('library.editor.toolsPlaceholder')"
            :model-value="(draft.tools?.include || []).join(' ') + ((draft.tools?.exclude || []).length ? ' !' + (draft.tools?.exclude || []).join(' !') : '')"
            @update:model-value="applyToolFilter(String($event))"
          )
          FieldDescription {{ $t("library.editor.toolsHint") }}
      template(v-else)
        .flex.flex-wrap.items-start.gap-2.rounded-md.border.p-2(
          v-if="managedOrigin"
          data-testid="integrations-managed"
        )
          GitBranchIcon.size-4(class="mt-0.5")
          .flex.min-w-0.flex-1.flex-col.gap-1
            span.text-sm.font-medium {{ $t("library.editor.managed") }}
            span.text-muted-foreground.font-mono.text-xs
              | {{ managedOrigin.slug }} · {{ managedOrigin.path }}@{{ managedOrigin.commit.slice(0, 7) }}
            span.text-muted-foreground.text-xs {{ $t("library.editor.managedHint") }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="detach"
            :disabled="detaching"
            data-testid="integrations-detach"
          )
            Spinner(v-if="detaching" data-icon="inline-start")
            UnlinkIcon(v-else data-icon="inline-start")
            span {{ $t("library.editor.detach") }}
        Field
          FieldLabel(for="integration-body") {{ $t("library.editor.body") }}
          Textarea#integration-body.font-mono.text-sm(
            v-model="draft.body"
            :rows="8"
            :readonly="Boolean(managedOrigin)"
          )
          FieldDescription {{ $t("library.editor.bodyHint") }}
        .flex.items-center.gap-2
          Switch(
            :model-value="draft.disableModelInvocation"
            :aria-label="$t('library.editor.manual')"
            @update:model-value="draft.disableModelInvocation = Boolean($event)"
            :disabled="Boolean(managedOrigin)"
          )
          span.text-sm {{ $t("library.editor.manual") }}

      .flex.flex-col.gap-2
        .flex.items-center.justify-between.gap-2
          .flex.flex-col
            span.text-sm.font-medium {{ $t("library.editor.installs") }}
            span.text-muted-foreground.text-xs {{ $t("library.editor.installsHint") }}

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
          p.text-muted-foreground.text-xs(v-else)
            | {{ $t("library.editor.installsEmpty") }}
          .flex.flex-wrap.items-center.gap-2
            Select(
              :model-value="newInstall.scope.kind"
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
              v-if="newInstall.scope.kind !== 'org'"
              v-model="newInstall.scope.id"
              :placeholder="$t('library.editor.ownerId')"
              :disabled="installBusy"
            )
            Button(
              type="button"
              variant="outline"
              size="sm"
              @click="addInstallToEntry"
              :disabled="installBusy || (!newInstall.scope.id && newInstall.scope.kind !== 'org')"
              data-testid="integrations-install-add"
            )
              PlusIcon(data-icon="inline-start")
              span {{ $t("library.editor.addInstall") }}

        template(v-else)
          .flex.flex-wrap.items-center.gap-2(v-if="draft.installs.length")
            .flex.items-center.gap-2.rounded-md.border.px-2.py-1(
              v-for="(install, index) in draft.installs"
              :key="`staged-install-${index}`"
            )
              Select(
                :model-value="install.scope.kind"
                @update:model-value="onInstallKind(index, String($event))"
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
                v-if="install.scope.kind !== 'org'"
                v-model="install.scope.id"
                :placeholder="$t('library.editor.ownerId')"
              )
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
          p.text-muted-foreground.text-xs(v-else)
            | {{ $t("library.editor.installsEmpty") }}
          Button(type="button" variant="outline" size="sm" @click="addInstall")
            PlusIcon(data-icon="inline-start")
            span {{ $t("library.editor.addInstall") }}

      .flex.items-center.gap-2
        Switch(
          :model-value="draft.enabled"
          :aria-label="$t('library.editor.enabled')"
          @update:model-value="toggleEnabled(Boolean($event))"
        )
        .flex.flex-col
          span.text-sm {{ $t("library.editor.enabled") }}
          span.text-muted-foreground.text-xs {{ $t("library.editor.enabledHint") }}

      p.text-destructive.text-sm(v-if="saveError") {{ saveError }}
      .flex.items-center.gap-2
        Button(
          v-if="!managedOrigin"
          type="button"
          @click="save"
          :disabled="saving"
          data-testid="integrations-save"
        )
          Spinner(v-if="saving" data-icon="inline-start")
          CheckIcon(v-else data-icon="inline-start")
          span {{ $t("common.save") }}
        Button(type="button" variant="ghost" @click="draft = undefined")
          | {{ $t("common.cancel") }}
        span.flex-1
        Button.text-destructive(
          v-if="draft.id"
          type="button"
          variant="ghost"
          @click="remove"
          :disabled="saving"
          data-testid="integrations-delete"
        )
          Trash2Icon(data-icon="inline-start")
          span {{ $t("common.delete") }}
</template>

<script setup lang="ts">
import GitSourcesPanel from "@/components/integrations/GitSourcesPanel.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
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
import { Textarea } from "@/components/ui/textarea";
import {
  SCOPE_KINDS,
  integrationsApi,
  scopeLabel,
  type GitCredential,
  type GitSource,
  type GitSourcesStatus,
  type IntegrationInput,
  type IntegrationInstall,
  type IntegrationLibrary,
  type IntegrationScope,
  type IntegrationValue,
  type McpServerEntry,
  type SkillEntry,
  type SkillOrigin,
} from "@/lib/integrations";
import {
  CheckIcon,
  FileTextIcon,
  GitBranchIcon,
  PlusIcon,
  ServerIcon,
  Trash2Icon,
  UnlinkIcon,
  XIcon,
} from "@lucide/vue";
import { computed, onMounted, ref } from "vue";

const props = defineProps<{
  scope?: IntegrationScope;

  preview?: IntegrationLibrary;
}>();

const KINDS = [
  { value: "mcp", labelKey: "library.kind.mcp" },
  { value: "skill", labelKey: "library.kind.skill" },
  { value: "git", labelKey: "library.kind.git" },
] as const;

type Kind = (typeof KINDS)[number]["value"];
type ValueMode = "value" | "env" | "secret";

interface Draft extends IntegrationInput {
  installs: IntegrationInstall[];
}

const kind = ref<Kind>("mcp");
const ownerFilter = ref("all");
const servers = ref<McpServerEntry[]>([]);
const skills = ref<SkillEntry[]>([]);
const seal = ref(false);
const git = ref<GitSourcesStatus | undefined>();
const sources = ref<GitSource[]>([]);
const credentials = ref<GitCredential[]>([]);
const admin = ref(false);
const loading = ref(true);
const error = ref("");
const draft = ref<Draft | undefined>();
const saving = ref(false);
const saveError = ref("");
const installBusy = ref(false);
const detaching = ref(false);

const managedOrigin = ref<SkillOrigin | undefined>();
const newInstall = ref<{ scope: IntegrationScope }>({
  scope: { kind: "org", id: "" },
});
const valueRows = ref<
  { key: string; mode: ValueMode; value: string; hasSecret: boolean }[]
>([]);

interface Row {
  kind: Kind;
  id: number;
  name: string;
  description: string;
  owner: IntegrationScope;
  enabled: boolean;
  installs: IntegrationInstall[];
  entry: McpServerEntry | SkillEntry;
}

const allRows = computed<Row[]>(() => [
  ...servers.value.map((entry) => ({
    kind: "mcp" as const,
    id: entry.id,
    name: entry.name,
    description: entry.description,
    owner: entry.owner,
    enabled: entry.enabled,
    installs: entry.installs ?? [],
    entry,
  })),
  ...skills.value.map((entry) => ({
    kind: "skill" as const,
    id: entry.id,
    name: entry.name,
    description: entry.description,
    owner: entry.owner,
    enabled: entry.enabled,
    installs: entry.installs ?? [],
    entry,
  })),
]);

const rows = computed(() =>
  allRows.value
    .filter((row) => row.kind === kind.value)
    .filter(
      (row) =>
        ownerFilter.value === "all" || row.owner.kind === ownerFilter.value,
    )
    .filter((row) => !props.scope || appliesHere(row)),
);

function appliesHere(row: Row): boolean {
  const scope = props.scope;
  if (!scope) return true;
  if (row.owner.kind === scope.kind && row.owner.id === scope.id) return true;
  return row.installs.some(
    (install) =>
      (install.scope.kind === scope.kind && install.scope.id === scope.id) ||
      install.scope.kind === "org",
  );
}

function countOf(target: Kind): number {
  if (target === "git") return sources.value.length;
  return allRows.value.filter(
    (row) => row.kind === target && (!props.scope || appliesHere(row)),
  ).length;
}

function valueMode(next: string): ValueMode {
  if (next === "env") return "env";
  if (next === "secret") return "secret";
  return "value";
}

function onOwnerKind(next: string): void {
  if (!draft.value) return;
  draft.value.owner = {
    kind: next as IntegrationScope["kind"],
    id: next === "org" ? "" : draft.value.owner.id,
  };
}

function onInstallKind(index: number, next: string): void {
  const install = draft.value?.installs[index];
  if (!install) return;
  install.scope = {
    kind: next as IntegrationScope["kind"],
    id: next === "org" ? "" : install.scope.id,
  };
}

function setNewInstallKind(next: string): void {
  newInstall.value.scope = {
    kind: next as IntegrationScope["kind"],
    id: next === "org" ? "" : newInstall.value.scope.id,
  };
}

async function addInstallToEntry(): Promise<void> {
  if (!draft.value?.id) return;
  await applyInstall(newInstall.value.scope, true);
  newInstall.value = { scope: { kind: "org", id: "" } };
}

async function toggleInstall(
  scope: IntegrationScope,
  enabled: boolean,
): Promise<void> {
  await applyInstall(scope, enabled);
}

async function dropInstall(scope: IntegrationScope): Promise<void> {
  const entry = draft.value;
  if (!entry?.id) return;
  installBusy.value = true;
  saveError.value = "";
  try {
    await integrationsApi.removeInstall(entryKind(), entry.id, scope);
    await reloadEntry();
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    installBusy.value = false;
  }
}

async function applyInstall(
  scope: IntegrationScope,
  enabled: boolean,
): Promise<void> {
  const entry = draft.value;
  if (!entry?.id) return;
  installBusy.value = true;
  saveError.value = "";
  try {
    await integrationsApi.setInstall(entryKind(), entry.id, scope, enabled);
    await reloadEntry();
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    installBusy.value = false;
  }
}

function rowSource(row: Row): SkillOrigin | undefined {
  if (row.kind !== "skill") return undefined;
  return (row.entry as SkillEntry).source;
}

function entryKind(): "mcp" | "skill" {
  return kind.value === "mcp" ? "mcp" : "skill";
}

async function reloadEntry(): Promise<void> {
  const entry = draft.value;
  if (!entry?.id) return;
  const library = await integrationsApi.list();
  servers.value = library.servers ?? [];
  skills.value = library.skills ?? [];
  seal.value = library.seal;
  if (kind.value === "mcp") {
    const fresh = (library.servers ?? []).find((row) => row.id === entry.id);
    if (fresh)
      entry.installs = (fresh.installs ?? []).map((install) => ({
        ...install,
      }));
    return;
  }
  const fresh = (library.skills ?? []).find((row) => row.id === entry.id);
  if (fresh)
    entry.installs = (fresh.installs ?? []).map((install) => ({ ...install }));
}

function addInstall(): void {
  if (!draft.value) return;
  draft.value.installs.push({
    scope: { kind: "org", id: "" },
    enabled: true,
  });
}

function addValue(): void {
  valueRows.value.push({ key: "", mode: "value", value: "", hasSecret: false });
}

function removeValue(index: number): void {
  valueRows.value.splice(index, 1);
}

function applyToolFilter(raw: string): void {
  if (!draft.value) return;
  const include: string[] = [];
  const exclude: string[] = [];
  for (const token of raw.split(/\s+/).filter(Boolean)) {
    if (token.startsWith("!")) exclude.push(token.slice(1));
    else include.push(token);
  }
  draft.value.tools = { include, exclude };
}

function startCreate(): void {
  saveError.value = "";
  managedOrigin.value = undefined;
  valueRows.value = [];
  draft.value = {
    name: "",
    description: "",
    owner: props.scope ?? { kind: "org", id: "" },
    enabled: true,
    installs: [],
    url: "",
    headers: {},
    connectTimeoutMs: 15000,
    requestTimeoutMs: 900000,
    tools: { include: [], exclude: [] },
    directTools: true,
    body: "",
    files: [],
    disableModelInvocation: false,
  };
}

function select(row: Row): void {
  saveError.value = "";
  managedOrigin.value = undefined;
  newInstall.value = { scope: { ...row.owner } };
  if (row.kind === "mcp") {
    const entry = row.entry as McpServerEntry;
    valueRows.value = Object.entries(entry.headers ?? {}).map(
      ([key, value]) => ({
        key,
        mode: value.env ? "env" : value.hasSecret ? "secret" : "value",
        value: value.value ?? value.env ?? "",

        hasSecret: value.hasSecret === true,
      }),
    );
    draft.value = {
      id: entry.id,
      name: entry.name,
      description: entry.description,
      owner: { ...entry.owner },
      enabled: entry.enabled,
      installs: (entry.installs ?? []).map((install) => ({
        scope: { ...install.scope },
        enabled: install.enabled,
      })),
      url: entry.url,
      connectTimeoutMs: entry.connectTimeoutMs,
      requestTimeoutMs: entry.requestTimeoutMs,

      tools: {
        include: [...(entry.tools?.include ?? [])],
        exclude: [...(entry.tools?.exclude ?? [])],
      },
      directTools: entry.directTools,
    };
    return;
  }
  const entry = row.entry as SkillEntry;
  managedOrigin.value = entry.source;
  valueRows.value = [];
  draft.value = {
    id: entry.id,
    name: entry.name,
    description: entry.description,
    owner: { ...entry.owner },
    enabled: entry.enabled,
    installs: (entry.installs ?? []).map((install) => ({
      scope: { ...install.scope },
      enabled: install.enabled,
    })),
    body: entry.body,
    files: [...(entry.files ?? [])],
    disableModelInvocation: entry.disableModelInvocation,
  };
}

async function toggleEnabled(enabled: boolean): Promise<void> {
  const entry = draft.value;
  if (!entry) return;
  entry.enabled = enabled;
  if (!entry.id || !managedOrigin.value) return;
  saveError.value = "";
  try {
    await integrationsApi.setSkillEnabled(entry.id, enabled);
    await load();
  } catch (err) {
    entry.enabled = !enabled;
    saveError.value = err instanceof Error ? err.message : String(err);
  }
}

async function detach(): Promise<void> {
  const entry = draft.value;
  if (!entry?.id) return;
  detaching.value = true;
  saveError.value = "";
  try {
    const detached = await integrationsApi.detachSkill(entry.id);
    managedOrigin.value = undefined;
    draft.value = {
      ...entry,
      body: detached.body,
      files: [...(detached.files ?? [])],
      disableModelInvocation: detached.disableModelInvocation,
    };
    await load();
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    detaching.value = false;
  }
}

function collectValues(): Record<string, IntegrationValue> {
  const out: Record<string, IntegrationValue> = {};
  for (const row of valueRows.value) {
    const key = row.key.trim();
    if (!key) continue;
    if (row.mode === "env") out[key] = { env: row.value.trim() };
    else if (row.mode === "secret")
      out[key] =
        row.hasSecret && row.value === ""
          ? { secret: "", hasSecret: true }
          : { secret: row.value };
    else out[key] = { value: row.value };
  }
  return out;
}

async function save(): Promise<void> {
  if (!draft.value) return;
  if (managedOrigin.value) return;
  saving.value = true;
  saveError.value = "";
  try {
    const input: IntegrationInput = {
      ...draft.value,

      installs: draft.value.id
        ? []
        : draft.value.installs.map((install) => ({
            scope: install.scope,
            enabled: install.enabled,
          })),
    };
    if (kind.value === "mcp") {
      input.headers = collectValues();
      await integrationsApi.saveServer(input);
    } else {
      await integrationsApi.saveSkill(input);
    }
    draft.value = undefined;
    await load();
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function remove(): Promise<void> {
  if (!draft.value?.id) return;
  saving.value = true;
  saveError.value = "";
  try {
    if (kind.value === "mcp")
      await integrationsApi.deleteServer(draft.value.id);
    else await integrationsApi.deleteSkill(draft.value.id);
    draft.value = undefined;
    await load();
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = "";
  try {
    const library = props.preview ?? (await integrationsApi.list());
    servers.value = library.servers ?? [];
    skills.value = library.skills ?? [];
    seal.value = library.seal;
    git.value = library.git;
    sources.value = library.sources ?? [];
    credentials.value = library.credentials ?? [];
    admin.value = library.admin === true;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<i18n lang="yaml">
en:
  library:
    ownerFilter: "Filter by owner"
    allOwners: "All owners"
    counts: "{servers} MCP servers · {skills} skills"
    ownedBy: "Owned by {scope}"
    installedOn: "Installed here: switched on"
    installedOff: "Installed here: switched off (subtracts the entry)"
    loading: "Loading the library…"
    new: "New entry"
    policy: "Policy"
    importedMissing: "removed from repo"
    agencyTransport: "HTTP via agency"
    scope:
      org: "Org-wide"
      project: "Project"
      team: "Team"
      group: "Group"
      agent: "Agent"
    kind:
      mcp: "MCP server"
      skill: "Skill"
      git: "Git sources"
    empty:
      title: "Nothing here yet"
      description: "Add a server or a skill, then install it into the scopes that should receive it."
    editor:
      create: "New entry"
      edit: "Edit entry"
      description: "An entry belongs to one scope; installs switch it on for the scopes that should receive it."
      name: "Name"
      nameHint: "Letters, digits, dots, dashes and underscores. Names are unique across the agency."
      summary: "Description"
      owner: "Owner"
      ownerId: "Scope id"
      ownerHint: "Ownership decides who may edit the entry. An install decides where it applies."
      urlPlaceholder: "https://mcp.example.com/mcp"
      urlHint: "A streamable HTTP endpoint. The agency calls it; a worker never connects to it directly, so the address and every credential stay here."
      envPlaceholder: "DATABASE_URL"
      url: "URL"
      headers: "Headers"
      valueKey: "Name"
      valueLiteral: "Value"
      valueEnv: "Agency env"
      valueSecret: "Secret"
      valuePlaceholder: "value"
      secretStored: "stored secret — leave blank to keep"
      valuesHint: "A secret is sealed by the agency and never returned; an agency env value is read from the agency's environment."
      addValue: "Add value"
      removeValue: "Remove value"
      tools: "Exposed tools"
      toolsHint: "Globs, space separated; prefix a glob with ! to exclude it. Empty exposes every tool."
      toolsPlaceholder: "query !admin_*"
      body: "SKILL.md body"
      bodyHint: "The instructions the model reads when it loads the skill."
      manual: "Manual invocation only"
      installs: "Installs"
      installsHint: "A switched-off install subtracts the entry for that scope's tasks."
      addInstall: "Add install"
      installEnabled: "Install switched on"
      installsEmpty: "Not installed anywhere yet."
      removeInstall: "Remove install"
      enabled: "Entry enabled"
      enabledHint: "Switched off is a policy entry: it never connects and removes the name everywhere it applies."
      managed: "Imported from a Git source"
      managedHint: "The source owns the body and files, so the next sync would overwrite an edit. Detach it to keep a hand-written copy."
      detach: "Detach to edit"
ru:
  library:
    ownerFilter: "Фильтр по владельцу"
    allOwners: "Все владельцы"
    counts: "{servers} MCP-серверов · {skills} навыков"
    ownedBy: "Владелец: {scope}"
    installedOn: "Установлено здесь: включено"
    installedOff: "Установлено здесь: выключено (вычитает запись)"
    loading: "Загружаем библиотеку…"
    new: "Новая запись"
    policy: "Политика"
    importedMissing: "удалён из репозитория"
    agencyTransport: "HTTP через агентство"
    scope:
      org: "Вся организация"
      project: "Проект"
      team: "Команда"
      group: "Группа"
      agent: "Агент"
    kind:
      mcp: "MCP-сервер"
      skill: "Навык"
      git: "Git-источники"
    empty:
      title: "Пока пусто"
      description: "Добавьте сервер или навык и установите его в нужные области."
    editor:
      create: "Новая запись"
      edit: "Изменить запись"
      description: "Запись принадлежит одной области; установки включают её для областей, которые должны её получать."
      name: "Название"
      nameHint: "Буквы, цифры, точки, дефисы и подчёркивания. Названия уникальны в пределах агентства."
      summary: "Описание"
      owner: "Владелец"
      ownerId: "Идентификатор области"
      ownerHint: "Владение решает, кто может править запись. Установка решает, где она применяется."
      urlPlaceholder: "https://mcp.example.com/mcp"
      urlHint: "Конечная точка streamable HTTP. Запрос делает агентство; воркер к ней не подключается, поэтому адрес и все учётные данные остаются здесь."
      envPlaceholder: "DATABASE_URL"
      url: "URL"
      headers: "Заголовки"
      valueKey: "Имя"
      valueLiteral: "Значение"
      valueEnv: "Окружение агентства"
      valueSecret: "Секрет"
      valuePlaceholder: "значение"
      secretStored: "секрет сохранён — оставьте пустым, чтобы не менять"
      valuesHint: "Секрет запечатывает агентство и никогда не возвращает; значение «окружение агентства» читается из окружения агентства."
      addValue: "Добавить значение"
      removeValue: "Удалить значение"
      tools: "Доступные инструменты"
      toolsHint: "Шаблоны через пробел; префикс ! исключает шаблон. Пусто — все инструменты."
      toolsPlaceholder: "query !admin_*"
      body: "Текст SKILL.md"
      bodyHint: "Инструкции, которые модель читает, загружая навык."
      manual: "Только ручной вызов"
      installs: "Установки"
      installsHint: "Выключенная установка убирает запись для задач этой области."
      addInstall: "Добавить установку"
      installEnabled: "Установка включена"
      installsEmpty: "Пока никуда не установлено."
      removeInstall: "Удалить установку"
      enabled: "Запись включена"
      enabledHint: "Выключенная запись — это политика: она не подключается и убирает имя везде, где применяется."
      managed: "Импортировано из Git-источника"
      managedHint: "Текстом и файлами владеет источник, поэтому следующая синхронизация перезапишет правку. Отсоедините навык, чтобы оставить ручную копию."
      detach: "Отсоединить для правки"
</i18n>
