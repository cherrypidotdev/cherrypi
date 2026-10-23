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
      TabsTrigger(value="general") General
      TabsTrigger(value="projects") Projects
      TabsTrigger(value="tasks") Tasks
      TabsTrigger(value="security") Security

    TabsContent(value="general")
      Card
        CardHeader
          CardTitle General
          CardDescription
            | The agent's picture, name, description, and username. The username
            | is the agent's stable identifier.
        CardContent
          form.flex.flex-col.gap-4(@submit.prevent="saveGeneral")
            FieldGroup
              Field
                FieldLabel Picture
                PictureUpload(
                  :picture="item?.picture"
                  :agent-id="id"
                  :fallback="pictureFallback"
                  @changed="onPictureChanged"
                )
              Field
                FieldLabel Username
                Input(disabled :model-value="item?.username ?? ''")
              Field
                FieldLabel(for="agent-display-name") Name
                Input#agent-display-name(v-model="name")
              Field
                FieldLabel(for="agent-description") Description
                Textarea#agent-description(v-model="description")
              Field
                FieldLabel Status
                .flex.items-center.gap-2
                  Badge(
                    variant="outline"
                    :class="agentStatusClass(item?.status)"
                  )
                    | {{ agentStatusLabel(item?.status) }}
                  span.text-muted-foreground.font-mono.text-xs {{ item?.remoteAddr || "—" }}
            p.text-destructive.text-sm(v-if="error") {{ error }}
            .flex.justify-end
              Button(type="submit" :disabled="saving || !name.trim()")
                Spinner(v-if="saving" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span Save

    TabsContent(value="projects")
      Card
        CardHeader
          CardTitle Projects
          CardDescription
            | Projects this agent has joined, with a project-specific prompt and
            | concurrency caps. Task access is governed by the project's roles, so
            | there are no per-project task permissions here.
        CardContent
          p.text-muted-foreground.text-sm(v-if="!projectPrompts.length")
            | This agent has not joined any project.
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
                  aria-label="Open project"
                  :to="{ name: 'project', params: { slug: entry.project?.slug } }"
                )
                  ArrowUpRightIcon.size-4
                Button.text-destructive(
                  variant="outline"
                  size="sm"
                  @click="leaveProject(entry)"
                )
                  LogOutIcon(data-icon="inline-start")
                  span Leave
              CollapsibleContent
                .flex.flex-col.gap-3.px-4.pb-4
                  PromptEditor(
                    placeholder="Project-specific instructions for this agent…"
                    :scope="PromptScope.AGENT_PROJECT"
                    :agent-id="id"
                    :project-id="entry.project?.id"
                  )
                  AgentProjectLimits(
                    v-if="entry.project?.id"
                    :agent-id="id"
                    :project-id="entry.project.id"
                  )

    TabsContent(value="tasks")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle Agent prompt
            CardDescription
              | Injected into new tasks on this agent, after the organization prompt
              | and before the project prompts. Edits affect future tasks only.
          CardContent.flex.flex-col.gap-3
            .flex.items-center.justify-between.gap-2
              span.text-sm.font-medium Overall instructions
              span.text-muted-foreground.text-xs Version {{ agentPromptVersion }}
            Textarea#agent-prompt(
              v-model="agentPrompt"
              placeholder="Agent-wide instructions…"
            )
            .flex.justify-end
              Button(
                type="button"
                @click="savePrompt"
                :disabled="savingPrompt"
              )
                Spinner(v-if="savingPrompt" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span Save prompt
        ConcurrencyPanel(:agent-id="id")

        Card
          CardHeader
            CardTitle Model defaults
            CardDescription
              | New tasks start with this model and thinking level. Only agent
              | owners can change them; everyone can see the selection.
          CardContent.flex.flex-col.gap-4
            p.text-muted-foreground.text-sm(v-if="!canManage")
              | Only agent owners can change the default model and thinking
              | level.
            FieldGroup
              Field
                FieldLabel Model
                Select(
                  :model-value="modelChoice"
                  @update:model-value="onDefaultModelChange"
                  :disabled="!canManage || loadingDefaults || savingDefaults || !modelOptions.length"
                  data-testid="agent-default-model-select"
                )
                  SelectTrigger.w-full(data-testid="agent-default-model")
                    SelectValue(placeholder="Pi default")
                  SelectContent
                    SelectGroup
                      SelectItem(:value="PI_DEFAULT") Pi default
                      SelectItem(
                        v-for="model in modelOptions"
                        :key="`${model.provider}/${model.id}`"
                        :value="`${model.provider}/${model.id}`"
                      ) {{ model.name || model.id }}
              Field
                FieldLabel Thinking level
                Select(
                  :model-value="thinkingChoice"
                  @update:model-value="onDefaultThinkingChange"
                  :disabled="!canManage || loadingDefaults || savingDefaults || !thinkingOptions.length"
                  data-testid="agent-default-thinking-select"
                )
                  SelectTrigger.w-full(data-testid="agent-default-thinking")
                    SelectValue(placeholder="Model default")
                  SelectContent
                    SelectGroup
                      SelectItem(:value="PI_DEFAULT_LEVEL") Model default
                      SelectItem(
                        v-for="level in thinkingOptions"
                        :key="level"
                        :value="level"
                      ) {{ level }}
            p.text-muted-foreground.text-xs(v-if="!modelOptions.length")
              | The agent is offline, so its model list is unavailable.

    TabsContent(value="security")
      .flex.flex-col.gap-6
        AgentOwnersPanel(:agent-id="id")
        AgentTeamsPanel(:agent-id="id")

        Card
          CardHeader
            CardTitle Credentials
            CardDescription
              | Each credential authenticates this agent's tunnel. Passwords are
              | shown once, at creation.
          CardContent
            .flex.flex-col.gap-3
              Alert(v-if="revealed")
                AlertTitle Credential generated
                AlertDescription Copy the password now — it is shown only once.
                .mt-3.flex.flex-col.gap-1.font-mono.text-sm
                  span Username: {{ revealed.username }}
                  span Password: {{ revealed.password }}
                .mt-3.flex.gap-2
                  Button(size="sm" variant="outline" @click="copyCredential")
                    CopyIcon(data-icon="inline-start")
                    span Copy
                  Button(
                    size="sm"
                    variant="ghost"
                    @click="revealed = undefined"
                  ) Dismiss
              Table
                TableHeader
                  TableRow
                    TableHead Username
                    TableHead Created
                    TableHead Last used
                    TableHead
                TableBody
                  TableRowsSkeleton(
                    v-if="loading && !credentials.length"
                    :widths="['w-28', 'w-28', 'w-28', 'w-16']"
                  )
                  TableRow(
                    v-for="credential in credentials"
                    :key="credential.username"
                  )
                    TableCell.font-mono.text-sm {{ credential.username }}
                    TableCell.text-muted-foreground {{ formatTimestamp(credential.createdAt) }}
                    TableCell.text-muted-foreground
                      | {{ credential.lastUsedAt ? formatTimestamp(credential.lastUsedAt) : "Never" }}
                    TableCell
                      .flex.justify-end
                        Button(
                          variant="outline"
                          size="sm"
                          @click="revoke(credential.username)"
                          :disabled="busy === credential.username"
                        )
                          Trash2Icon(data-icon="inline-start")
                          span Revoke
                  TableRow(v-if="!credentials.length && !loading")
                    TableCell.text-muted-foreground(:colspan="4") No credentials.
              .flex.justify-end
                Button(
                  variant="outline"
                  size="sm"
                  @click="addCredential"
                  :disabled="revealing"
                )
                  Spinner(v-if="revealing" data-icon="inline-start")
                  PlusIcon(v-else data-icon="inline-start")
                  span Add credential

        AgentActivityPanel(:agent-id="id")

        Card
          Collapsible(v-slot="{ open }")
            CollapsibleTrigger.flex.w-full.cursor-pointer.items-center.justify-between.gap-3(
              type="button"
            )
              .flex.flex-col.text-left
                CardTitle Runtime logs
                CardDescription
                  | Operational records this agent pushed to the agency, plus the
                  | terminal lifecycle events the agency observed.
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
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
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
import { agentStatusClass, agentStatusLabel } from "@/lib/agent";
import { formatTimestamp } from "@/lib/format";
import { agency, currentAgentId } from "@/lib/sdk";
import {
  PromptScope,
  type Agent,
  type AgentCredential,
  type AgentProjectPrompt,
  type Attachment,
} from "@cherrypi/sdk";
import {
  ArrowUpRightIcon,
  CheckIcon,
  ChevronRightIcon,
  CopyIcon,
  LogOutIcon,
  PlusIcon,
  Trash2Icon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { toast } from "vue3-toastify";
import LogsView from "./LogsView.vue";

// Set by the router guard from the route's username param.
const id = computed(() => currentAgentId.value);
const { canManageAgent } = useAgentAccess();
const canManage = computed(() => canManageAgent(id.value));

// Reka selects cannot carry an empty value, so "Pi default" and "model
// default" use sentinels translated back to "" on save.
const PI_DEFAULT = "__pi_default__";
const PI_DEFAULT_LEVEL = "__model_default__";

const tab = ref("general");
const item = ref<Agent>();
const name = ref("");
const description = ref("");
const agentPrompt = ref("");
const agentPromptVersion = ref<bigint>(0n);
const saving = ref(false);
const savingPrompt = ref(false);
const error = ref("");
const loading = ref(true);

const credentials = ref<AgentCredential[]>([]);
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

const modelCatalog = ref<ModelOption[]>([]);
// Thinking levels of Pi's configured default model, used while no explicit
// model is selected.
const defaultThinkingLevels = ref<string[]>([]);
const loadingDefaults = ref(false);
const savingDefaults = ref(false);
const revealing = ref(false);
const busy = ref("");
const revealed = ref<{ username: string; password: string } | undefined>();

const revealText = computed(() =>
  revealed.value ? `${revealed.value.username}:${revealed.value.password}` : "",
);

async function load() {
  loading.value = true;
  try {
    const [agentItem, creds, prompt] = await Promise.all([
      agency.getAgent(id.value),
      agency.listAgentCredentials(id.value),
      agency.getPrompt(PromptScope.AGENT, { agentId: id.value }),
    ]);
    item.value = agentItem;
    name.value = agentItem.displayName || agentItem.username;
    description.value = agentItem.description;
    agentPrompt.value = prompt.content;
    agentPromptVersion.value = prompt.version;
    credentials.value = creds;
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
    // Keep a saved model selectable when the agent is offline or Pi no longer
    // lists it, so the owner can see (and clear) the stored choice.
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

async function saveDefaults(
  provider: string,
  modelId: string,
  modelName: string,
  thinkingLevel: string,
) {
  if (!canManage.value) return;
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
    toast.success("Model defaults updated");
  } catch (err) {
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

// The upload/remove already happened agency-side; reflect it locally.
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
    toast.success("Agent updated");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

async function savePrompt() {
  savingPrompt.value = true;
  try {
    const prompt = await withProgress(() =>
      agency.setPrompt(PromptScope.AGENT, agentPrompt.value, {
        agentId: id.value,
      }),
    );
    agentPromptVersion.value = prompt.version;
    toast.success("Agent prompt updated");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingPrompt.value = false;
  }
}

async function addCredential() {
  revealing.value = true;
  try {
    const result = await withProgress(() =>
      agency.createAgentCredential(id.value, ""),
    );
    revealed.value = {
      username: result.credential.username,
      password: result.password,
    };
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    revealing.value = false;
  }
}

async function revoke(username: string) {
  if (
    !window.confirm(
      `Revoke credential "${username}"? Any tunnel using it will disconnect.`,
    )
  ) {
    return;
  }
  busy.value = username;
  try {
    await withProgress(() => agency.revokeAgentCredential(id.value, username));
    toast.success("Credential revoked");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    busy.value = "";
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
    toast.success("Left project");
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function copyCredential() {
  try {
    await navigator.clipboard.writeText(revealText.value);
    toast.success("Copied");
  } catch {
    toast.error("Copy failed");
  }
}

onMounted(() => {
  void load();
  void loadDefaults();
});
</script>
