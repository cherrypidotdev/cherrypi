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
  Tabs(v-model="tab")
    TabsList
      TabsTrigger(value="general") {{ $t("settings.tabs.general") }}
      TabsTrigger(value="projects") {{ $t("nav.projects") }}
      TabsTrigger(value="plugins") {{ $t("account.tabs.plugins") }}
      TabsTrigger(value="tasks") {{ $t("nav.tasks") }}
      TabsTrigger(value="security") {{ $t("account.tabs.security") }}

    TabsContent(value="general")
      Card
        CardHeader
          CardTitle {{ $t("settings.tabs.general") }}
          CardDescription
            | {{ $t("agentSettings.profileHint") }}
        CardContent
          form.flex.flex-col.gap-4(@submit.prevent="saveGeneral")
            FieldGroup
              Field
                FieldLabel {{ $t("projectSettings.general.picture") }}
                PictureUpload(
                  :picture="item?.picture"
                  :agent-id="id"
                  :fallback="pictureFallback"
                  @changed="onPictureChanged"
                )
              Field
                FieldLabel {{ $t("common.username") }}
                Input(disabled :model-value="item?.username ?? ''")
              Field
                FieldLabel(for="agent-display-name") {{ $t("common.name") }}
                Input#agent-display-name(v-model="name")
              Field
                FieldLabel(for="agent-description") {{ $t("common.description") }}
                Textarea#agent-description(v-model="description")
              Field
                FieldLabel {{ $t("common.status") }}
                .flex.items-center.gap-2
                  Badge(
                    variant="outline"
                    :class="agentStatusClass(item?.status)"
                  )
                    | {{ agentStatusLabel(item?.status) }}
                  span.text-muted-foreground.font-mono.text-xs {{ item?.remoteAddr || "—" }}
              Field
                FieldLabel {{ $t("admin.role") }}
                template(v-if="isAdmin")
                  Select(
                    :model-value="item?.role || 'member'"
                    @update:model-value="onRoleChange"
                  )
                    SelectTrigger.w-48
                      SelectValue
                    SelectContent
                      SelectItem(value="member") {{ $t("admin.roleMember") }}
                      SelectItem(value="admin") {{ $t("settings.tabs.admin") }}
                  FieldDescription
                    | {{ $t("agentSettings.adminHint") }}
                template(v-else)
                  Badge(variant="outline") {{ item?.role === "admin" ? "Admin" : "Member" }}
            p.text-destructive.text-sm(v-if="error") {{ error }}
            .flex.justify-end
              Button(type="submit" :disabled="saving || !name.trim()")
                Spinner(v-if="saving" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span {{ $t("common.save") }}

      Card
        CardHeader
          CardTitle {{ $t("agentSettings.health.title") }}
          CardDescription
            | {{ $t("agentSettings.health.description") }}
        CardContent.flex.flex-col.gap-4
          FieldGroup
            Field(data-testid="agent-health-interval-field")
              FieldLabel {{ $t("agentSettings.health.interval") }}
              Select(
                :model-value="healthChoice"
                @update:model-value="onHealthChoiceChange"
                :disabled="!canManage || savingHealth || !healthSettingsLoaded"
              )
                SelectTrigger.w-full(
                  class="sm:w-64"
                  data-testid="agent-health-interval-trigger"
                )
                  SelectValue
                SelectContent
                  SelectGroup
                    SelectItem(value="default") {{ $t("agentSettings.health.agentDefault") }}
                    SelectItem(
                      v-for="preset in HEALTH_INTERVAL_PRESETS"
                      :key="preset.seconds"
                      :value="String(preset.seconds)"
                    ) {{ preset.label }}
                    SelectItem(value="off") {{ $t("agentSettings.health.off") }}
                    SelectItem(value="custom") {{ $t("agentSettings.health.custom") }}
              FieldDescription(data-testid="agent-health-interval-summary")
                | {{ healthIntervalSummary }}
            Field(v-if="healthChoice === 'custom'")
              FieldLabel(for="agent-health-interval") {{ $t("agentSettings.health.customInterval") }}
              .flex.items-center.gap-2
                Input#agent-health-interval(
                  v-model="customHealthInterval"
                  placeholder="e.g. 90s, 5m, 1h"
                  autocomplete="off"
                  @keyup.enter="saveCustomHealthInterval"
                  :disabled="!canManage || savingHealth"
                )
                Button(
                  type="button"
                  variant="outline"
                  @click="saveCustomHealthInterval"
                  :disabled="!canManage || savingHealth"
                  data-testid="agent-health-interval-save"
                )
                  Spinner(v-if="savingHealth" data-icon="inline-start")
                  CheckIcon(v-else data-icon="inline-start")
                  span {{ $t("common.save") }}
              FieldDescription.text-destructive(v-if="customHealthError")
                | {{ customHealthError }}
          p.text-muted-foreground.text-xs(v-if="!canManage")
            | {{ $t("agentSettings.health.ownersOnly") }}

    TabsContent(value="projects")
      Card
        CardHeader
          CardTitle {{ $t("nav.projects") }}
          CardDescription
            | {{ $t("agentSettings.projects.description") }}
        CardContent
          p.text-muted-foreground.text-sm(v-if="!projectPrompts.length")
            | {{ $t("agentSettings.projects.empty") }}
          .flex.flex-col.gap-4
            Collapsible.rounded-lg.border(
              v-for="entry in projectPrompts"
              :key="entry.project?.id"
              class="group/collapsible"
            )
              .flex.items-center.gap-3.p-4
                CollapsibleTrigger.flex.flex-1.cursor-pointer.items-center.gap-2.text-left
                  ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
                    class="group-data-[state=open]/collapsible:rotate-90"
                  )
                  span.size-5.shrink-0(:innerHTML="projectLogo(entry)")
                  span.text-sm.font-medium {{ entry.project?.name }}
                RouterLink.text-muted-foreground(
                  class="hover:text-foreground"
                  :to="{ name: 'project', params: { slug: entry.project?.slug } }"
                  :aria-label="$t('team.projects.openAria')"
                )
                  ArrowUpRightIcon.size-4
                Button.text-destructive(
                  variant="outline"
                  size="sm"
                  @click="leaveProject(entry)"
                )
                  LogOutIcon(data-icon="inline-start")
                  span {{ $t("team.members.leave") }}
              CollapsibleContent
                .flex.flex-col.gap-3.px-4.pb-4
                  PromptEditor(
                    :placeholder="$t('agentSettings.projects.promptPlaceholder')"
                    :scope="PromptScope.AGENT_PROJECT"
                    :agent-id="id"
                    :project-id="entry.project?.id"
                  )
                  AgentProjectLimits(
                    v-if="entry.project?.id"
                    :agent-id="id"
                    :project-id="entry.project.id"
                  )

    TabsContent(value="plugins")
      AgentPluginsPanel(v-if="item" :agent-actor-id="item.actorId")

    TabsContent(value="tasks")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle {{ $t("agentSettings.prompt.title") }}
            CardDescription
              | {{ $t("agentSettings.prompt.description") }}
          CardContent.flex.flex-col.gap-3
            PromptEditor(
              :placeholder="$t('agentSettings.prompt.placeholder')"
              :scope="PromptScope.AGENT"
              :agent-id="id"
            )

        Card
          CardHeader
            CardTitle {{ $t("agentSettings.selfPrompt.title") }}
            CardDescription
              | {{ $t("agentSettings.selfPrompt.description") }}
          CardContent.flex.items-center.justify-between.gap-3
            p.text-muted-foreground.text-sm
              | {{ item?.promptAutoApprove ? "Enabled: self-proposals apply immediately." : "Disabled: an owner approves self-proposals." }}
            Switch(
              :model-value="item?.promptAutoApprove ?? false"
              @update:model-value="setAutoApprove"
              :disabled="!canManage || savingAutoApprove"
            )
        ConcurrencyPanel(:agent-id="id")

        Card
          CardHeader
            CardTitle {{ $t("agentSettings.models.title") }}
            CardDescription
              | {{ $t("agentSettings.models.description") }}
          CardContent.flex.flex-col.gap-4
            p.text-muted-foreground.text-sm(v-if="!canManage")
              | {{ $t("agentSettings.models.ownersOnly") }}
            FieldGroup
              Field
                .flex.items-center.justify-between.gap-2
                  FieldLabel {{ $t("domainLog.kind.model") }}

                  Button(
                    type="button"
                    variant="ghost"
                    size="sm"
                    :title="$t('agentSettings.models.resetTitle')"
                    @click="reloadModels"
                    :disabled="loadingDefaults"
                    data-testid="reset-models"
                  )
                    Spinner(v-if="loadingDefaults" data-icon="inline-start")
                    RefreshCwIcon(v-else data-icon="inline-start")
                    span {{ $t("agentSettings.models.reset") }}
                Select(
                  :model-value="modelChoice"
                  @update:model-value="onDefaultModelChange"
                  :disabled="!canManage || loadingDefaults || savingDefaults || !modelOptions.length"
                  data-testid="agent-default-model-select"
                )
                  SelectTrigger.w-full(data-testid="agent-default-model")
                    SelectValue(
                      :placeholder="$t('agentSettings.models.piDefaultPlaceholder')"
                    )
                  SelectContent
                    SelectGroup
                      SelectItem(:value="PI_DEFAULT") {{ $t("agentSettings.models.piDefaultPlaceholder") }}
                      SelectItem(
                        v-for="model in modelOptions"
                        :key="`${model.provider}/${model.id}`"
                        :value="`${model.provider}/${model.id}`"
                      ) {{ modelLabel(model) }}
              Field
                FieldLabel {{ $t("taskModel.thinking") }}
                Select(
                  :model-value="thinkingChoice"
                  @update:model-value="onDefaultThinkingChange"
                  :disabled="!canManage || loadingDefaults || savingDefaults || !thinkingOptions.length"
                  data-testid="agent-default-thinking-select"
                )
                  SelectTrigger.w-full(data-testid="agent-default-thinking")
                    SelectValue(
                      :placeholder="$t('agentSettings.models.defaultPlaceholder')"
                    )
                  SelectContent
                    SelectGroup
                      SelectItem(:value="PI_DEFAULT_LEVEL") {{ $t("agentSettings.models.defaultPlaceholder") }}
                      SelectItem(
                        v-for="level in thinkingOptions"
                        :key="level"
                        :value="level"
                      ) {{ level }}
            p.text-muted-foreground.text-xs(v-if="!modelOptions.length")
              | {{ $t("agentSettings.models.offline") }}

    TabsContent(value="security")
      .flex.flex-col.gap-6
        AgentOwnersPanel(:agent-id="id")
        AgentTeamsPanel(:agent-id="id")

        Card(v-if="canManage")
          CardHeader
            CardTitle {{ $t("account.tokens.title") }}
            CardDescription
              | {{ $t("agentSettings.tokens.description") }}
          CardContent.flex.flex-col.gap-4
            Alert(v-if="revealedToken")
              AlertTitle {{ $t("domainLog.kind.tokenCreated") }}
              AlertDescription
                | {{ $t("agentSettings.tokens.setToken") }}
              .mt-3.flex.flex-col.gap-1.font-mono.text-sm
                span {{ $t("agentSettings.tokenName", { name: revealedToken.name }) }}
                span.break-all {{ $t("agentSettings.tokenValue", { token: revealedToken.plaintext }) }}
              .mt-3.flex.gap-2
                Button(size="sm" variant="outline" @click="copyToken")
                  CopyIcon(data-icon="inline-start")
                  span {{ $t("common.copy") }}
                Button(
                  size="sm"
                  variant="ghost"
                  @click="revealedToken = undefined"
                ) {{ $t("common.dismiss") }}
            form.flex.items-end.gap-2(@submit.prevent="createToken")
              Field.flex-1
                FieldLabel(for="agent-token-name") {{ $t("common.name") }}
                Input#agent-token-name(
                  v-model="tokenName"
                  placeholder="agent"
                  autocomplete="off"
                )
                FieldDescription
                  | {{ $t("agentSettings.tokens.nameHint") }}
              Button(
                type="submit"
                variant="outline"
                :disabled="creatingToken || !tokenName.trim()"
              )
                Spinner(v-if="creatingToken" data-icon="inline-start")
                PlusIcon(v-else data-icon="inline-start")
                span {{ $t("agentSettings.tokens.create") }}
            Table
              TableHeader
                TableRow
                  TableHead {{ $t("common.name") }}
                  TableHead {{ $t("agentSettings.tokens.prefix") }}
                  TableHead {{ $t("admin.teams.columnCreated") }}
                  TableHead {{ $t("agentSettings.tokens.lastUsed") }}
                  TableHead {{ $t("users.invitations.expires") }}
                  TableHead
              TableBody
                TableRowsSkeleton(
                  v-if="loadingTokens && !tokens.length"
                  :widths="['w-28', 'w-24', 'w-28', 'w-28', 'w-24', 'w-16']"
                )
                TableRow(v-for="token in tokens" :key="token.id.toString()")
                  TableCell.text-sm {{ token.name }}
                  TableCell.text-muted-foreground.font-mono.text-xs {{ token.prefix }}
                  TableCell.text-muted-foreground {{ formatTimestamp(token.createdAt) }}
                  TableCell.text-muted-foreground
                    | {{ token.lastUsedAt ? formatTimestamp(token.lastUsedAt) : "Never" }}
                  TableCell.text-muted-foreground
                    | {{ token.expiresAt ? formatTimestamp(token.expiresAt) : "Never" }}
                  TableCell
                    .flex.justify-end
                      Button(
                        variant="outline"
                        size="sm"
                        @click="revokeToken(token)"
                        :disabled="revokingToken === token.id"
                      )
                        Trash2Icon(data-icon="inline-start")
                        span {{ $t("common.revoke") }}
                TableRow(v-if="!tokens.length && !loadingTokens")
                  TableCell.text-muted-foreground(:colspan="6") {{ $t("agentSettings.tokens.empty") }}

        AgentActivityPanel(:agent-id="id")

        Card
          Collapsible(v-slot="{ open }")
            CollapsibleTrigger.flex.w-full.cursor-pointer.items-center.justify-between.gap-3(
              type="button"
            )
              .flex.flex-col.text-left
                CardTitle {{ $t("agentPlugins.runtimeLogs") }}
                CardDescription
                  | {{ $t("agentSettings.logs.description") }}
              ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
                :class="open ? 'rotate-90' : ''"
              )
            CollapsibleContent.mt-4
              LogsView(embedded)
</template>

<script setup lang="ts">
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import PromptEditor from "@/components/PromptEditor.vue";
import AgentActivityPanel from "@/components/settings/AgentActivityPanel.vue";
import AgentOwnersPanel from "@/components/settings/AgentOwnersPanel.vue";
import AgentPluginsPanel from "@/components/settings/AgentPluginsPanel.vue";
import AgentProjectLimits from "@/components/settings/AgentProjectLimits.vue";
import AgentTeamsPanel from "@/components/settings/AgentTeamsPanel.vue";
import ConcurrencyPanel from "@/components/settings/ConcurrencyPanel.vue";
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
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
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useAgentAccess } from "@/composables/useAgentAccess";
import { withProgress } from "@/composables/useProgress";
import { currentAccount } from "@/lib/account";
import { agentStatusClass, agentStatusLabel } from "@/lib/agent";
import { formatTimestamp } from "@/lib/format";
import {
  formatHealthInterval,
  HEALTH_INTERVAL_MAX_SECONDS,
  HEALTH_INTERVAL_MIN_SECONDS,
  HEALTH_INTERVAL_PRESETS,
  parseHealthIntervalSeconds,
} from "@/lib/health";
import { agency, currentAgentId } from "@/lib/sdk";
import {
  PromptScope,
  type Agent,
  type AgentProjectPrompt,
  type AgentToken,
  type Attachment,
} from "@cherrypi/sdk";
import {
  ArrowUpRightIcon,
  CheckIcon,
  ChevronRightIcon,
  CopyIcon,
  LogOutIcon,
  PlusIcon,
  RefreshCwIcon,
  Trash2Icon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute } from "vue-router";
import { toast } from "vue3-toastify";
import LogsView from "./LogsView.vue";

const { t } = useI18n();

const id = computed(() => currentAgentId.value);
const { canManageAgent } = useAgentAccess();
const canManage = computed(() => canManageAgent(id.value));

const isAdmin = computed(() => currentAccount.value?.is_admin ?? false);

const PI_DEFAULT = "__pi_default__";
const PI_DEFAULT_LEVEL = "__model_default__";

const SETTINGS_TABS = ["general", "projects", "plugins", "tasks", "security"];
const settingsRoute = useRoute();
const tab = ref(
  typeof settingsRoute.query.section === "string" &&
    SETTINGS_TABS.includes(settingsRoute.query.section)
    ? settingsRoute.query.section
    : "general",
);
const item = ref<Agent>();
const name = ref("");
const description = ref("");
const saving = ref(false);
const savingAutoApprove = ref(false);
const error = ref("");
const loading = ref(true);

const healthIntervalSeconds = ref<number>();
const healthSettingsLoaded = ref(false);
const savingHealth = ref(false);
const customHealthInterval = ref("");
const customHealthOpen = ref(false);
const customHealthError = ref("");

const PRESET_SECONDS = new Set(
  HEALTH_INTERVAL_PRESETS.map((item) => item.seconds),
);

const healthChoice = computed(() => {
  if (!healthSettingsLoaded.value) return "default";
  if (customHealthOpen.value) return "custom";
  const seconds = healthIntervalSeconds.value;
  if (seconds === undefined) return "default";
  if (seconds === 0) return "off";
  if (PRESET_SECONDS.has(seconds)) return String(seconds);
  return "custom";
});

const healthIntervalSummary = computed(() => {
  if (!healthSettingsLoaded.value) return t("agentSettings.health.loading");
  const seconds = healthIntervalSeconds.value;
  let summary: string;
  if (seconds === undefined) {
    summary = t("agentSettings.health.inherits");
  } else if (seconds === 0) {
    summary = t("agentSettings.health.disabled");
  } else {
    summary = `The agent probes the host every ${formatHealthInterval(seconds)}.`;
  }
  return canManage.value
    ? summary
    : `${summary} Only agent owners can change it.`;
});

const tokens = ref<AgentToken[]>([]);
const loadingTokens = ref(false);
const tokenName = ref("agent");
const creatingToken = ref(false);
const revokingToken = ref<bigint | null>(null);
const revealedToken = ref<{ name: string; plaintext: string } | undefined>();
const projectPrompts = ref<AgentProjectPrompt[]>([]);
const defaultProvider = ref("");
const defaultModelId = ref("");
const defaultModelName = ref("");
const defaultThinking = ref("");
interface ModelOption {
  provider: string;
  id: string;
  name: string;
  thinkingLevels: string[];
}

function modelLabel(model: ModelOption): string {
  const name = model.name || model.id;
  return model.provider ? `${model.provider}/${name}` : name;
}

const modelCatalog = ref<ModelOption[]>([]);

const defaultThinkingLevels = ref<string[]>([]);
const loadingDefaults = ref(false);
const savingDefaults = ref(false);
async function load() {
  loading.value = true;
  try {
    const [agentItem, tokenList] = await Promise.all([
      agency.getAgent(id.value),
      agency.listAgentTokens(id.value),
    ]);
    item.value = agentItem;
    name.value = agentItem.displayName || agentItem.username;
    description.value = agentItem.description;
    tokens.value = tokenList;
    projectPrompts.value = await agency.listAgentProjectPrompts(id.value);
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

const modelOptions = computed<ModelOption[]>(() => {
  const list = [...modelCatalog.value];
  if (
    defaultProvider.value &&
    !list.some(
      (model) =>
        model.provider === defaultProvider.value &&
        model.id === defaultModelId.value,
    )
  ) {
    list.unshift({
      provider: defaultProvider.value,
      id: defaultModelId.value,
      name: defaultModelName.value || defaultModelId.value,
      thinkingLevels: [],
    });
  }
  return list;
});

const selectedModel = computed(() =>
  modelOptions.value.find(
    (model) =>
      model.provider === defaultProvider.value &&
      model.id === defaultModelId.value,
  ),
);
const modelChoice = computed(() =>
  defaultProvider.value
    ? `${defaultProvider.value}/${defaultModelId.value}`
    : PI_DEFAULT,
);
const thinkingOptions = computed(() => {
  const levels = [
    ...(selectedModel.value
      ? selectedModel.value.thinkingLevels
      : defaultThinkingLevels.value),
  ];
  if (defaultThinking.value && !levels.includes(defaultThinking.value)) {
    levels.push(defaultThinking.value);
  }
  return levels;
});
const thinkingChoice = computed(
  () => defaultThinking.value || PI_DEFAULT_LEVEL,
);

async function loadDefaults() {
  loadingDefaults.value = true;
  try {
    const [saved, catalog] = await Promise.all([
      agency.getAgentDefaults(id.value),
      agency.getTaskDefaults(id.value).catch(() => undefined),
    ]);
    defaultProvider.value = saved.modelProvider;
    defaultModelId.value = saved.modelId;
    defaultModelName.value = saved.modelName;
    defaultThinking.value = saved.thinkingLevel;
    defaultThinkingLevels.value = [...(catalog?.thinkingLevels ?? [])];
    modelCatalog.value = (catalog?.models ?? []).map((model) => ({
      provider: model.provider,
      id: model.id,
      name: model.name,
      thinkingLevels: [...model.thinkingLevels],
    }));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    loadingDefaults.value = false;
  }
}

async function loadHealthSettings() {
  try {
    const settings = await agency.getAgentHealthSettings(id.value);
    const seconds =
      settings.intervalSeconds === undefined
        ? undefined
        : Number(settings.intervalSeconds);
    healthIntervalSeconds.value = seconds;
    customHealthOpen.value =
      seconds !== undefined && seconds > 0 && !PRESET_SECONDS.has(seconds);
    if (seconds !== undefined && seconds > 0) {
      customHealthInterval.value = formatHealthInterval(seconds);
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    healthSettingsLoaded.value = true;
  }
}

async function saveHealthInterval(seconds: number | undefined) {
  if (!canManage.value) return;
  savingHealth.value = true;
  customHealthError.value = "";
  try {
    const saved = await withProgress(() =>
      agency.updateAgentHealthSettings(id.value, { intervalSeconds: seconds }),
    );
    healthIntervalSeconds.value =
      saved.intervalSeconds === undefined
        ? undefined
        : Number(saved.intervalSeconds);
    if (healthIntervalSeconds.value) {
      customHealthInterval.value = formatHealthInterval(
        healthIntervalSeconds.value,
      );
    }
    toast.success(t("agentSettings.toasts.intervalUpdated"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
    await loadHealthSettings();
  } finally {
    savingHealth.value = false;
  }
}

function onHealthChoiceChange(value: unknown) {
  const choice = String(value);
  if (choice === "custom") {
    customHealthOpen.value = true;
    return;
  }
  customHealthOpen.value = false;
  customHealthError.value = "";
  if (choice === "default") {
    void saveHealthInterval(undefined);
    return;
  }
  if (choice === "off") {
    void saveHealthInterval(0);
    return;
  }
  const seconds = Number(choice);
  if (Number.isFinite(seconds)) void saveHealthInterval(seconds);
}

function saveCustomHealthInterval() {
  const seconds = parseHealthIntervalSeconds(customHealthInterval.value);
  if (seconds === undefined) {
    customHealthError.value = t("agentSettings.errors.duration");
    return;
  }
  if (
    seconds !== 0 &&
    (seconds < HEALTH_INTERVAL_MIN_SECONDS ||
      seconds > HEALTH_INTERVAL_MAX_SECONDS)
  ) {
    customHealthError.value = t("agentSettings.errors.intervalRange");
    return;
  }
  customHealthError.value = "";
  void saveHealthInterval(seconds);
}

async function reloadModels() {
  await loadDefaults();
  if (modelCatalog.value.length) {
    toast.success(`${modelCatalog.value.length} models loaded from the agent`);
    return;
  }
  toast.error(t("agentSettings.models.offline"));
}

async function saveDefaults(
  provider: string,
  modelId: string,
  modelName: string,
  thinkingLevel: string,
) {
  if (!canManage.value) return;

  const previous = {
    provider: defaultProvider.value,
    id: defaultModelId.value,
    name: defaultModelName.value,
    thinking: defaultThinking.value,
  };
  defaultProvider.value = provider;
  defaultModelId.value = modelId;
  defaultModelName.value = modelName;
  defaultThinking.value = thinkingLevel;
  savingDefaults.value = true;
  try {
    const saved = await withProgress(() =>
      agency.updateAgentDefaults(id.value, {
        modelProvider: provider,
        modelId,
        modelName,
        thinkingLevel,
      }),
    );
    defaultProvider.value = saved.modelProvider;
    defaultModelId.value = saved.modelId;
    defaultModelName.value = saved.modelName;
    defaultThinking.value = saved.thinkingLevel;
    toast.success(t("agentSettings.toasts.modelsUpdated"));
  } catch (err) {
    defaultProvider.value = previous.provider;
    defaultModelId.value = previous.id;
    defaultModelName.value = previous.name;
    defaultThinking.value = previous.thinking;
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingDefaults.value = false;
  }
}

function onDefaultModelChange(value: unknown) {
  const choice = String(value);
  if (choice === PI_DEFAULT) {
    const levels = defaultThinkingLevels.value;
    const level =
      levels.length === 0 || levels.includes(defaultThinking.value)
        ? defaultThinking.value
        : "";
    void saveDefaults("", "", "", level);
    return;
  }
  const model = modelOptions.value.find(
    (item) => `${item.provider}/${item.id}` === choice,
  );
  if (!model) return;
  const levels = model.thinkingLevels ?? [];
  const level = levels.includes(defaultThinking.value)
    ? defaultThinking.value
    : "";
  void saveDefaults(model.provider, model.id, model.name, level);
}

function onDefaultThinkingChange(value: unknown) {
  const choice = String(value);
  const level = choice === PI_DEFAULT_LEVEL ? "" : choice;
  void saveDefaults(
    defaultProvider.value,
    defaultModelId.value,
    defaultModelName.value,
    level,
  );
}

const pictureFallback = computed(() => toSvg(id.value || "agent", 48));

function onPictureChanged(picture: Attachment | undefined) {
  if (item.value) item.value.picture = picture;
}

async function saveGeneral() {
  saving.value = true;
  try {
    item.value = await withProgress(() =>
      agency.updateAgent(id.value, {
        displayName: name.value.trim(),
        description: description.value.trim(),
      }),
    );
    toast.success(t("agentSettings.toasts.updated"));
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function onRoleChange(value: unknown) {
  const role = value === "admin" ? "admin" : "member";
  if (!item.value || item.value.role === role) return;
  try {
    item.value = await withProgress(() =>
      agency.updateAgentRole(id.value, role),
    );
    toast.success(
      role === "admin"
        ? t("agentSettings.toasts.promoted")
        : t("agentSettings.toasts.demoted"),
    );
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function setAutoApprove(value: unknown) {
  const enabled = Boolean(value);
  savingAutoApprove.value = true;
  try {
    item.value = await withProgress(() =>
      agency.updateAgent(id.value, {
        displayName: item.value?.displayName || item.value?.username || "",
        description: item.value?.description ?? "",
        promptAutoApprove: enabled,
      }),
    );
    toast.success(
      enabled
        ? t("agentSettings.toasts.selfPromptOn")
        : t("agentSettings.toasts.selfPromptOff"),
    );
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingAutoApprove.value = false;
  }
}

async function loadTokens() {
  loadingTokens.value = true;
  try {
    tokens.value = await agency.listAgentTokens(id.value);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    loadingTokens.value = false;
  }
}

async function createToken() {
  if (!canManage.value) return;
  creatingToken.value = true;
  try {
    const result = await withProgress(() =>
      agency.createAgentToken(id.value, tokenName.value.trim() || "agent"),
    );
    revealedToken.value = {
      name: result.token.name,
      plaintext: result.plaintext,
    };
    tokenName.value = "agent";
    await loadTokens();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    creatingToken.value = false;
  }
}

async function revokeToken(token: AgentToken) {
  if (
    !window.confirm(
      `Revoke token "${token.name}" (${token.prefix})? Any agent using it will disconnect.`,
    )
  ) {
    return;
  }
  revokingToken.value = token.id;
  try {
    await withProgress(() => agency.revokeAgentToken(id.value, token.id));
    toast.success(t("domainLog.kind.tokenRevoked"));
    await loadTokens();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    revokingToken.value = null;
  }
}

async function copyToken() {
  if (!revealedToken.value) return;
  try {
    await navigator.clipboard.writeText(revealedToken.value.plaintext);
    toast.success(t("common.copied"));
  } catch {
    toast.error(t("taskConsole.copyFailed"));
  }
}

function projectLogo(entry: AgentProjectPrompt): string {
  return toSvg(entry.project?.id ?? "project", 20);
}

async function leaveProject(entry: AgentProjectPrompt) {
  const project = entry.project;
  if (!project) return;
  const warning =
    `Leave "${project.name}"? This permanently deletes this agent's tasks ` +
    "in the project.";
  if (!window.confirm(warning)) return;
  try {
    await withProgress(() => agency.removeProjectAgent(project.id, id.value));
    toast.success(t("agentSettings.toasts.leftProject"));
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

watch(
  canManage,
  (can) => {
    if (can) void loadTokens();
  },
  { immediate: true },
);

onMounted(() => {
  void load();
  void loadDefaults();
  void loadHealthSettings();
});
</script>

<i18n lang="yaml">
en:
  agentSettings:
    tokens:
      description: "Long-lived credentials for this agent's tunnel, bundled Pi tools and artifact uploads. The plaintext is shown only once, at creation."
      setToken: "Set CHERRYPI_AGENT_TOKEN to this value. It is shown only once."
      nameHint: "A label for this token; it does not grant permissions."
      create: "Create token"
      prefix: "Prefix"
      lastUsed: "Last used"
      empty: "No tokens."
    profileHint: "The agent's picture, name, description, and username. The username is the agent's stable identifier."
    adminHint: "Admin agents may read the agency's process log and every agent's logs and metadata, read-only. Only a human administrator can change the role."
    health:
      title: "Health sampling"
      description: "How often the agent probes its host for the Health tab. The agency stores the interval and pushes it over the agent's tunnel, so a change applies without restarting the agent."
      interval: "Probe interval"
      agentDefault: "Agent default"
      off: "Off"
      custom: "Custom…"
      customInterval: "Custom interval"
      ownersOnly: "Only agent owners can change the sampling interval."
      disabled: "Sampling is disabled; the Health tab keeps its existing history."
      inherits: "Inherits the agent's local default (CHERRYPI_HEALTH_INTERVAL, one minute unless set)."
      loading: "Loading the current cadence…"
    projects:
      description: "Projects this agent has joined, with a project-specific prompt and concurrency caps. Task access is governed by the project's roles, so there are no per-project task permissions here."
      empty: "This agent has not joined any project."
      promptPlaceholder: "Project-specific instructions for this agent…"
    prompt:
      title: "Agent prompt"
      description: "Injected into new tasks on this agent, after the organization prompt and before the project prompts. Edits affect future tasks only."
      placeholder: "Agent-wide instructions…"
    selfPrompt:
      title: "Self-prompt auto-approve"
      description: "When enabled, this agent's proposals to its own agent and agent-project prompts apply immediately instead of waiting for an owner. Project prompt proposals still need a project owner."
    models:
      title: "Model defaults"
      description: "New tasks start with this model and thinking level. Only agent owners can change them; everyone can see the selection."
      ownersOnly: "Only agent owners can change the default model and thinking level."
      reset: "Reset models"
      piDefaultPlaceholder: "Pi default"
      defaultPlaceholder: "Model default"
      offline: "The agent is offline, so its model list is unavailable."
      resetTitle: "Re-read the agent's models.json"
    logs:
      description: "Operational records this agent pushed to the agency, plus the terminal lifecycle events the agency observed."
    toasts:
      leftProject: "Left project"
      selfPromptOff: "Self-prompt auto-approve disabled"
      selfPromptOn: "Self-prompt auto-approve enabled"
      demoted: "Agent demoted to member"
      promoted: "Agent promoted to admin"
      updated: "Agent updated"
      modelsUpdated: "Model defaults updated"
      intervalUpdated: "Health sampling interval updated"
    errors:
      intervalRange: "Choose an interval between 10s and 24h, or Off."
      duration: "Enter a duration like 90s, 5m or 1h."
    tokenName: "Name: {name}"
    tokenValue: "Token: {token}"
ru:
  agentSettings:
    tokens:
      description: "Долгоживущие учётные данные для туннеля агента, встроенных инструментов Pi и загрузки артефактов. Открытый текст показывается только один раз, при создании."
      setToken: "Задайте это значение в CHERRYPI_AGENT_TOKEN. Оно показывается только один раз."
      nameHint: "Подпись для токена; разрешений она не даёт."
      create: "Создать токен"
      prefix: "Префикс"
      lastUsed: "Последнее использование"
      empty: "Токенов нет."
    profileHint: "Изображение, название, описание и имя пользователя агента. Имя пользователя — стабильный идентификатор агента."
    adminHint: "Агенты-администраторы могут читать журнал процесса агентства и журналы и метаданные всех агентов, только для чтения. Изменять роль может только человек-администратор."
    health:
      title: "Частота проверок здоровья"
      description: "Как часто агент опрашивает свой хост для вкладки «Здоровье». Агентство хранит интервал и передаёт его через туннель агента, поэтому изменение применяется без перезапуска агента."
      interval: "Интервал опроса"
      agentDefault: "По умолчанию у агента"
      off: "Выключено"
      custom: "Другое…"
      customInterval: "Свой интервал"
      ownersOnly: "Изменять интервал опроса могут только владельцы агента."
      disabled: "Опрос отключён; вкладка «Здоровье» сохраняет прежнюю историю."
      inherits: "Наследует локальное значение агента (CHERRYPI_HEALTH_INTERVAL, иначе одна минута)."
      loading: "Загружаем текущую частоту…"
    projects:
      description: "Проекты, в которые вступил агент, с промптом и лимитами параллельности для каждого проекта. Доступ к задачам определяется ролями проекта, поэтому отдельных разрешений на задачи здесь нет."
      empty: "Агент не вступил ни в один проект."
      promptPlaceholder: "Инструкции для этого агента в проекте…"
    prompt:
      title: "Промпт агента"
      description: "Добавляется в новые задачи этого агента после промпта организации и перед промптами проекта. Изменения влияют только на будущие задачи."
      placeholder: "Инструкции для всего агента…"
    selfPrompt:
      title: "Автоодобрение собственных промптов"
      description: "Если включено, предложения агента к его собственным промптам и промптам в проектах применяются сразу, без ожидания владельца. Предложения промптов проекта по-прежнему требуют владельца проекта."
    models:
      title: "Модели по умолчанию"
      description: "Новые задачи начинаются с этой модели и уровня размышления. Изменять их могут только владельцы агента; выбор видят все."
      ownersOnly: "Изменять модель и уровень размышления по умолчанию могут только владельцы агента."
      reset: "Сбросить модели"
      piDefaultPlaceholder: "Значение Pi"
      defaultPlaceholder: "Модель по умолчанию"
      offline: "Агент не в сети, поэтому список моделей недоступен."
      resetTitle: "Перечитать models.json агента"
    logs:
      description: "Операционные записи, которые агент отправил в агентство, а также события жизненного цикла терминалов, замеченные агентством."
    toasts:
      leftProject: "Вы вышли из проекта"
      selfPromptOff: "Автоодобрение собственных промптов отключено"
      selfPromptOn: "Автоодобрение собственных промптов включено"
      demoted: "Агент понижен до участника"
      promoted: "Агент повышен до администратора"
      updated: "Агент обновлён"
      modelsUpdated: "Модели по умолчанию обновлены"
      intervalUpdated: "Интервал проверок здоровья обновлён"
    errors:
      intervalRange: "Выберите интервал от 10s до 24h или «Выключено»."
      duration: "Введите длительность, например 90s, 5m или 1h."
    tokenName: "Имя: {name}"
    tokenValue: "Токен: {token}"
</i18n>
