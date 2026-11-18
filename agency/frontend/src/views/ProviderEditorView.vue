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
.mx-auto.flex.w-full.max-w-5xl.flex-col.gap-6.p-6
  .flex.flex-col.gap-2
    Button.self-start(as-child variant="ghost" size="sm")
      RouterLink(:to="{ name: 'settings', query: { tab: 'models' } }")
        ArrowLeftIcon(data-icon="inline-start")
        span {{ $t("providerEditor.back") }}
    h1.text-lg.font-semibold {{ formTitle }}
    p.text-muted-foreground.text-sm {{ $t("providerEditor.description") }}

  Card
    CardHeader
      CardTitle {{ $t("providerEditor.connectionTitle") }}
      CardDescription {{ $t("providerEditor.connectionDescription") }}
    CardContent
      p.text-muted-foreground.text-sm(v-if="loading") {{ $t("common.loading") }}
      p.text-destructive.text-sm(v-if="notFound" role="alert")
        | {{ $t("providerEditor.notFound") }}
      p.text-destructive.text-sm(v-if="formError" role="alert")
        | {{ formError }}
      form.flex.flex-col.gap-4(v-else-if="draft" @submit.prevent="save")
        .grid.gap-3(class="sm:grid-cols-2")
          Field
            FieldLabel(for="provider-slug") {{ $t("providerEditor.slug") }}
            Input#provider-slug(
              v-model="draft.slug"
              :placeholder="$t('providerEditor.slugExample')"
              :disabled="editing || busy"
            )
            FieldDescription {{ $t("providerEditor.slugHint") }}
          Field
            FieldLabel(for="provider-name") {{ $t("providerEditor.name") }}
            Input#provider-name(
              v-model="draft.name"
              :placeholder="$t('providerEditor.nameExample')"
              :disabled="busy"
            )
          Field
            FieldLabel(for="provider-api") {{ $t("providerEditor.api") }}
            Select(v-model="draft.api")
              SelectTrigger#provider-api(:disabled="busy")
                SelectValue(:placeholder="$t('providerEditor.pickApi')")
              SelectContent
                SelectItem(
                  v-for="api in WIRE_APIS"
                  :key="api.value"
                  :value="api.value"
                ) {{ api.label }}
          Field
            FieldLabel(for="provider-base-url")
              | {{ $t("providerEditor.baseUrl") }}
            Input#provider-base-url(
              v-model="draft.baseUrl"
              :placeholder="$t('providerEditor.baseUrlExample')"
              :disabled="busy"
            )
          Field(class="sm:col-span-2")
            FieldLabel(for="provider-key") {{ $t("providerEditor.apiKey") }}
            Input#provider-key(
              v-model="draft.apiKey"
              type="password"
              :placeholder="keyPlaceholder"
              :disabled="busy"
            )
            FieldDescription {{ $t("providerEditor.keyHint") }}
            FieldDescription.text-amber-600(
              class="dark:text-amber-500"
              data-test="oauth-unsupported"
            )
              | {{ $t("providerEditor.oauthUnsupported") }}
          Field.flex.flex-row.items-center.justify-between.gap-3(
            v-if="editing"
            class="sm:col-span-2"
          )
            .flex.flex-col
              FieldLabel(for="provider-enabled")
                | {{ $t("providerEditor.enabled") }}
              FieldDescription {{ $t("providerEditor.enabledHint") }}
            Switch#provider-enabled(v-model="draft.enabled" :disabled="busy")

        Separator

        .flex.items-center.justify-between
          h3.text-sm.font-medium {{ $t("providerEditor.scopesTitle") }}
          Button(
            type="button"
            variant="outline"
            size="sm"
            @click="addScope"
            :disabled="busy"
          ) {{ $t("providerEditor.addScope") }}
        .flex.flex-col.gap-2
          .grid.items-center.gap-2(
            v-for="(grant, index) in draft.grantees"
            :key="index"
            class="sm:grid-cols-4"
          )
            Select(
              :model-value="grant.kind"
              @update:model-value="changeScopeKind(grant, String($event))"
            )
              SelectTrigger
                SelectValue(:placeholder="$t('providerEditor.scope')")
              SelectContent
                SelectItem(
                  v-for="kind in SCOPE_KINDS"
                  :key="kind.value"
                  :value="kind.value"
                ) {{ kind.label }}
            ActorCombobox(
              v-if="grant.kind === 'agent'"
              :placeholder="scopePlaceholder(grant.kind)"
              value-mode="actorId"
              kinds="agents"
              size="sm"
              :model-value="grant.id"
              @update:model-value="grant.id = String($event)"
            )
            Combobox(
              v-else-if="grant.kind !== 'org'"
              :placeholder="scopePlaceholder(grant.kind)"
              size="sm"
              :model-value="grant.id"
              :options="targetOptions(grant.kind)"
              @update:model-value="grant.id = String($event)"
            )
            .text-muted-foreground.text-xs(v-else)
              | {{ $t("providerEditor.everyAgent") }}
            Select(
              :model-value="grant.modelId || '__all__'"
              @update:model-value="grant.modelId = $event === '__all__' ? '' : String($event)"
            )
              SelectTrigger
                SelectValue(:placeholder="$t('providerEditor.models')")
              SelectContent
                SelectItem(value="__all__") {{ $t("providerEditor.allModels") }}
                SelectItem(
                  v-for="model in draft.models.filter((m) => m.id)"
                  :key="model.id"
                  :value="model.id"
                ) {{ model.id }}
            Button(
              type="button"
              variant="ghost"
              size="sm"
              @click="draft.grantees.splice(index, 1)"
              :disabled="busy || draft.grantees.length === 1"
            ) {{ $t("common.remove") }}

        Separator

        .flex.flex-wrap.items-center.justify-between.gap-2
          h3.text-sm.font-medium {{ $t("providerEditor.modelsTitle") }}
          .flex.items-center.gap-2
            Button(
              type="button"
              variant="outline"
              size="sm"
              :title="canFetch ? $t('providerEditor.fetchHint') : $t('providerEditor.fetchNeedsConnection')"
              @click="fetchModels"
              :disabled="busy || fetching || !canFetch"
              data-test="fetch-models"
            )
              Spinner(v-if="fetching" data-icon="inline-start")
              RefreshCwIcon(v-else data-icon="inline-start")
              span {{ $t("providerEditor.fetchModels") }}
            Button(
              type="button"
              variant="outline"
              size="sm"
              @click="addModel"
              :disabled="busy"
            ) {{ $t("providerEditor.addModel") }}

        p.text-muted-foreground.text-xs(v-if="!canFetch")
          | {{ $t("providerEditor.fetchNeedsConnection") }}
        p.text-destructive.text-sm(v-if="fetchError" role="alert")
          | {{ fetchError }}
        p.text-muted-foreground.text-xs(v-if="fetchedIds")
          | {{ $t("providerEditor.fetchedCount", { count: fetchedIds.size }) }}
        p.text-destructive.text-xs(v-if="missingCount")
          | {{ $t("providerEditor.missingCount", { count: missingCount }) }}

        p.text-muted-foreground.text-sm(v-if="!draft.models.length")
          | {{ $t("providerEditor.modelsEmpty") }}
        template(v-else)
          .flex.flex-wrap.items-center.gap-2
            Input(
              v-model="modelFilter"
              :placeholder="$t('providerEditor.filterModels')"
              class="sm:max-w-xs"
              data-test="model-filter"
            )
          p.text-muted-foreground.text-sm(v-if="!filteredModels.length")
            | {{ $t("providerEditor.filterEmpty") }}
          .flex.flex-col.gap-2(v-else)
            .text-muted-foreground.hidden.items-center.gap-2.px-3.text-xs(
              class="sm:grid"
              :class="MODEL_GRID"
            )
              span {{ $t("providerEditor.modelId") }}
              span {{ $t("providerEditor.modelName") }}
              span {{ $t("providerEditor.context") }}
              span {{ $t("providerEditor.maxTokens") }}
              span {{ $t("providerEditor.reasoning") }}
              span {{ $t("providerEditor.image") }}
              span {{ $t("providerEditor.enabled") }}
              span
            .grid.items-center.gap-2.rounded-md.border.p-3(
              v-for="entry in visibleModels"
              :key="entry.index"
              :class="rowClass(entry.model)"
            )
              .flex.items-center.gap-1
                Input(
                  v-model="entry.model.id"
                  :placeholder="$t('providerEditor.modelId')"
                  :aria-invalid="notReturned(entry.model)"
                )
                TriangleAlertIcon.text-destructive.size-4(
                  v-if="notReturned(entry.model)"
                  :title="$t('providerEditor.modelNotReturnedHint')"
                )
              Input(
                v-model="entry.model.name"
                :placeholder="$t('providerEditor.modelName')"
              )
              Input(
                type="number"
                :placeholder="$t('providerEditor.context')"
                :model-value="String(entry.model.contextWindow)"
                @update:model-value="entry.model.contextWindow = Number($event) || 0"
              )
              Input(
                type="number"
                :placeholder="$t('providerEditor.maxTokens')"
                :model-value="String(entry.model.maxTokens)"
                @update:model-value="entry.model.maxTokens = Number($event) || 0"
              )
              label.flex.items-center.gap-2.text-sm
                input(v-model="entry.model.reasoning" type="checkbox")
                | {{ $t("providerEditor.reasoning") }}
              label.flex.items-center.gap-2.text-sm
                input(
                  type="checkbox"
                  :checked="hasImage(entry.model)"
                  @change="setImage(entry.model, $event)"
                )
                | {{ $t("providerEditor.image") }}
              label.flex.items-center.gap-2.text-sm
                input(v-model="entry.model.enabled" type="checkbox")
                | {{ $t("providerEditor.enabled") }}
              Button(
                type="button"
                variant="ghost"
                size="sm"
                @click="draft.models.splice(entry.index, 1)"
                :disabled="busy"
              ) {{ $t("common.remove") }}
              .grid.gap-2.rounded-md.p-2(
                class="bg-muted/40 sm:col-span-full sm:grid-cols-4"
              )
                label.flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground
                    | {{ $t("providerEditor.costInput") }}
                  Input(
                    type="number"
                    min="0"
                    step="0.0001"
                    :model-value="String(entry.model.cost.input)"
                    @update:model-value="entry.model.cost.input = Number($event) || 0"
                  )
                label.flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground
                    | {{ $t("providerEditor.costOutput") }}
                  Input(
                    type="number"
                    min="0"
                    step="0.0001"
                    :model-value="String(entry.model.cost.output)"
                    @update:model-value="entry.model.cost.output = Number($event) || 0"
                  )
                label.flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground
                    | {{ $t("providerEditor.costCacheRead") }}
                  Input(
                    type="number"
                    min="0"
                    step="0.0001"
                    :model-value="String(entry.model.cost.cacheRead)"
                    @update:model-value="entry.model.cost.cacheRead = Number($event) || 0"
                  )
                label.flex.flex-col.gap-1
                  span.text-2xs.text-muted-foreground
                    | {{ $t("providerEditor.costCacheWrite") }}
                  Input(
                    type="number"
                    min="0"
                    step="0.0001"
                    :model-value="String(entry.model.cost.cacheWrite)"
                    @update:model-value="entry.model.cost.cacheWrite = Number($event) || 0"
                  )

                .flex.flex-wrap.items-center.gap-1(class="sm:col-span-full")
                  span.text-2xs.text-muted-foreground(
                    :title="$t('providerEditor.thinkingLevelsHint')"
                  )
                    | {{ $t("providerEditor.thinkingLevels") }}
                  span.text-2xs.inline-flex.items-center.gap-1.rounded-md.border(
                    v-for="level in entry.model.thinkingLevels"
                    :key="level"
                    class="px-1.5 py-0.5"
                    :data-test="'model-thinking-level-' + level"
                  )
                    span {{ level }}
                    button.inline-flex.shrink-0.items-center.rounded-sm.opacity-60.transition-opacity(
                      type="button"
                      class="hover:opacity-100"
                      :title="$t('providerEditor.removeThinkingLevel', { level })"
                      @click="removeThinkingLevel(entry.model, level)"
                      :disabled="busy"
                    )
                      XIcon.size-3
                  span.text-2xs.text-muted-foreground(
                    v-if="!entry.model.thinkingLevels.length"
                  )
                    | {{ $t("providerEditor.thinkingLevelsEmpty") }}
                  DropdownMenu
                    DropdownMenuTrigger(as-child)
                      Button(
                        type="button"
                        variant="ghost"
                        size="sm"
                        :title="$t('providerEditor.addThinkingLevelHint')"
                        :disabled="busy || !missingThinkingLevels(entry.model).length"
                        data-test="model-thinking-levels-add"
                      )
                        PlusIcon(data-icon="inline-start")
                        span {{ $t("providerEditor.addThinkingLevel") }}
                    DropdownMenuContent.w-40(align="start")
                      DropdownMenuLabel
                        | {{ $t("providerEditor.addThinkingLevel") }}
                      DropdownMenuCheckboxItem(
                        v-for="level in missingThinkingLevels(entry.model)"
                        :key="level"
                        :model-value="false"
                        @select.prevent="addThinkingLevel(entry.model, level)"
                      ) {{ level }}
            p.text-muted-foreground.text-xs
              | {{ $t("providerEditor.costHint") }}
            .flex.flex-wrap.items-center.gap-x-3.gap-y-2(
              data-test="model-pager"
            )
              label.text-muted-foreground.flex.items-center.gap-2.text-xs
                span {{ $t("providerEditor.resultPerPage") }}
                Select(
                  :model-value="String(modelPageSize)"
                  @update:model-value="modelPageSize = Number($event)"
                )
                  SelectTrigger.w-20(size="sm" data-test="model-page-size")
                    SelectValue
                  SelectContent
                    SelectItem(
                      v-for="size in MODEL_PAGE_SIZES"
                      :key="size"
                      :value="String(size)"
                    ) {{ size }}
              span.text-muted-foreground.text-xs(data-test="model-range") {{ rangeLabel }}
              .ml-auto.flex.items-center.gap-1
                Button(
                  type="button"
                  variant="outline"
                  size="icon-sm"
                  :title="$t('providerEditor.firstPage')"
                  :aria-label="$t('providerEditor.firstPage')"
                  @click="modelPage = 1"
                  :disabled="modelPage <= 1"
                  data-test="model-first"
                )
                  ChevronsLeftIcon.size-3
                Button(
                  type="button"
                  variant="outline"
                  size="icon-sm"
                  :title="$t('providerEditor.previousPage')"
                  :aria-label="$t('providerEditor.previousPage')"
                  @click="modelPage--"
                  :disabled="modelPage <= 1"
                  data-test="model-previous"
                )
                  ChevronLeftIcon.size-3
                template(
                  v-for="(entry, position) in pageWindow"
                  :key="position"
                )
                  span.text-muted-foreground.px-1.text-xs(
                    v-if="entry === 'gap'"
                  )
                    | …
                  Button(
                    v-else
                    type="button"
                    size="icon-sm"
                    :variant="entry === modelPage ? 'default' : 'outline'"
                    :aria-label="$t('providerEditor.goToPageNumber', { page: entry })"
                    :aria-current="entry === modelPage ? 'page' : undefined"
                    @click="modelPage = Number(entry)"
                    data-test="model-page"
                  ) {{ entry }}
                Button(
                  type="button"
                  variant="outline"
                  size="icon-sm"
                  :title="$t('providerEditor.nextPage')"
                  :aria-label="$t('providerEditor.nextPage')"
                  @click="modelPage++"
                  :disabled="modelPage >= modelPageCount"
                  data-test="model-next"
                )
                  ChevronRightIcon.size-3
                Button(
                  type="button"
                  variant="outline"
                  size="icon-sm"
                  :title="$t('providerEditor.lastPage')"
                  :aria-label="$t('providerEditor.lastPage')"
                  @click="modelPage = modelPageCount"
                  :disabled="modelPage >= modelPageCount"
                  data-test="model-last"
                )
                  ChevronsRightIcon.size-3
              .flex.items-center.gap-1
                label.text-muted-foreground.text-xs(for="model-page-input")
                  | {{ $t("providerEditor.goToPage") }}
                Input#model-page-input.h-7.w-16(
                  v-model="pageInput"
                  type="number"
                  min="1"
                  :max="modelPageCount"
                  @keyup.enter="goToModelPage"
                  data-test="model-page-input"
                )
                Button(
                  type="button"
                  variant="outline"
                  size="sm"
                  @click="goToModelPage"
                  data-test="model-go"
                )
                  | {{ $t("providerEditor.go") }}

        p.text-muted-foreground.text-xs(v-if="!editing")
          | {{ $t("providerEditor.createEnabledHint") }}
        .flex.items-center.gap-2
          Button(type="submit" :disabled="busy")
            Spinner(v-if="busy" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ editing ? $t("common.saveChanges") : $t("common.create") }}
          Button(type="button" as-child variant="outline")
            RouterLink(:to="{ name: 'settings', query: { tab: 'models' } }")
              | {{ $t("common.cancel") }}
</template>

<script setup lang="ts">
import ActorCombobox from "@/components/ActorCombobox.vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { useProjects } from "@/composables/useProjects";
import {
  agentConfigApi,
  emptyModel,
  emptyProvider,
  SCOPE_KINDS,
  WIRE_APIS,
  type AgentConfig,
  type AgentModel,
  type ModelScope,
  type ProviderInput,
} from "@/lib/agentConfig";
import { sortThinkingLevels, THINKING_LEVELS } from "@/lib/model";
import { teamApi, type Team } from "@/lib/team";
import {
  ArrowLeftIcon,
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
  PlusIcon,
  RefreshCwIcon,
  TriangleAlertIcon,
  XIcon,
} from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const config = ref<AgentConfig | null>(null);
const loading = ref(true);
const busy = ref(false);
const fetching = ref(false);
const notFound = ref(false);
const formError = ref("");
const fetchError = ref("");
const draft = ref<DraftProvider | null>(null);

const fetchedIds = ref<Set<string> | null>(null);
const teams = ref<Team[]>([]);

const MODEL_PAGE_SIZE = 20;
const MODEL_PAGE_SIZES = [10, 20, 50, 100];

const MODEL_GRID =
  "sm:grid-cols-[minmax(0,2fr)_minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)_auto_auto_auto_auto]";
const modelFilter = ref("");
const modelPage = ref(1);
const modelPageSize = ref(MODEL_PAGE_SIZE);
const pageInput = ref("");

interface ModelEntry {
  model: DraftModel;
  index: number;
}

type DraftModel = AgentModel;

interface DraftProvider extends Omit<ProviderInput, "models"> {
  models: DraftModel[];
}

function draftModel(model: AgentModel): DraftModel {
  return {
    ...model,
    thinkingLevels: sortThinkingLevels(model.thinkingLevels ?? []),
  };
}

function missingThinkingLevels(model: DraftModel): string[] {
  return THINKING_LEVELS.filter(
    (level) => !model.thinkingLevels.includes(level),
  );
}

function addThinkingLevel(model: DraftModel, level: string) {
  model.thinkingLevels = sortThinkingLevels([...model.thinkingLevels, level]);
}

function removeThinkingLevel(model: DraftModel, level: string) {
  model.thinkingLevels = model.thinkingLevels.filter(
    (entry) => entry !== level,
  );
}

const filteredModels = computed<ModelEntry[]>(() => {
  const query = modelFilter.value.trim().toLowerCase();
  return (draft.value?.models ?? [])
    .map((model, index) => ({ model, index }))
    .filter(({ model }) => {
      if (!query) return true;
      return (
        model.id.toLowerCase().includes(query) ||
        model.name.toLowerCase().includes(query)
      );
    });
});

const modelPageCount = computed(() =>
  Math.max(1, Math.ceil(filteredModels.value.length / modelPageSize.value)),
);

const visibleModels = computed<ModelEntry[]>(() => {
  const start = (modelPage.value - 1) * modelPageSize.value;
  return filteredModels.value.slice(start, start + modelPageSize.value);
});

const rangeLabel = computed(() => {
  const total = filteredModels.value.length;
  const from = (modelPage.value - 1) * modelPageSize.value + 1;
  const to = Math.min(modelPage.value * modelPageSize.value, total);
  return t("providerEditor.rangeOf", { from, to, total });
});

const pageWindow = computed<(number | "gap")[]>(() => {
  const pages = modelPageCount.value;
  const current = modelPage.value;
  if (pages <= 7) {
    return Array.from({ length: pages }, (_, index) => index + 1);
  }
  const window = new Set<number>([1, pages, current]);
  for (let offset = 1; offset <= 2; offset++) {
    if (current - offset >= 1) window.add(current - offset);
    if (current + offset <= pages) window.add(current + offset);
  }
  const sorted = [...window].sort((a, b) => a - b);
  const out: (number | "gap")[] = [];
  for (const page of sorted) {
    const previous = out[out.length - 1];
    if (typeof previous === "number" && page - previous > 1) out.push("gap");
    out.push(page);
  }
  return out;
});

function goToModelPage() {
  const page = Number(pageInput.value);
  if (!Number.isFinite(page) || page <= 0) return;
  modelPage.value = Math.min(
    Math.max(Math.trunc(page), 1),
    modelPageCount.value,
  );
  pageInput.value = "";
}

watch(
  modelFilter,
  () => {
    modelPage.value = 1;
  },
  { flush: "sync" },
);
watch(modelPageCount, (count) => {
  if (modelPage.value > count) modelPage.value = count;
});

watch(modelPageSize, () => {
  modelPage.value = 1;
});

const { projects, loadProjects } = useProjects();
const { groups, loadAgentGroups } = useAgentGroups();

const slug = computed(() =>
  typeof route.params.slug === "string" ? route.params.slug : "",
);
const editing = computed(() => slug.value !== "");

const formTitle = computed(() =>
  editing.value
    ? t("providerEditor.editTitle", { slug: slug.value })
    : t("providerEditor.createTitle"),
);

const providerList = computed(() => config.value?.providers ?? []);

const hasStoredKey = computed(
  () =>
    providerList.value.find((provider) => provider.slug === slug.value)
      ?.hasKey ?? false,
);

const keyPlaceholder = computed(() =>
  editing.value && hasStoredKey.value
    ? t("providerEditor.keyUnchanged")
    : "sk-…",
);

const canFetch = computed(
  () =>
    Boolean(draft.value?.baseUrl.trim()) &&
    Boolean(draft.value?.api) &&
    Boolean(
      draft.value?.apiKey?.trim() || (editing.value && hasStoredKey.value),
    ),
);

const missingCount = computed(() => {
  if (!fetchedIds.value || !draft.value) return 0;
  return draft.value.models.filter((model) => notReturned(model)).length;
});

const projectOptions = computed<ComboboxOption[]>(() =>
  projects.value
    .filter((project) => !project.archivedAt)
    .map((project) => ({
      value: project.id,
      label: project.name,
      description: project.slug,
    })),
);

const teamOptions = computed<ComboboxOption[]>(() =>
  teams.value.map((team) => ({
    value: String(team.id),
    label: team.name,
    description: team.description,
  })),
);

const groupOptions = computed<ComboboxOption[]>(() =>
  groups.value
    .filter((group) => !group.archivedAt)
    .map((group) => ({
      value: String(group.id),
      label: group.name,
      description: group.slug,
    })),
);

function targetOptions(kind: string): ComboboxOption[] {
  switch (kind) {
    case "project":
      return projectOptions.value;
    case "team":
      return teamOptions.value;
    case "group":
      return groupOptions.value;
    default:
      return [];
  }
}

function changeScopeKind(grant: ModelScope, kind: string): void {
  grant.kind = kind as ModelScope["kind"];
  grant.id = "";
}

function scopePlaceholder(kind: string): string {
  switch (kind) {
    case "agent":
      return t("providerEditor.agentIdHint");
    case "team":
      return t("providerEditor.teamIdHint");
    case "project":
      return t("providerEditor.projectIdHint");
    case "group":
      return t("providerEditor.groupIdHint");
    default:
      return "";
  }
}

function addScope() {
  draft.value?.grantees.push({ kind: "agent", id: "" });
}

function addModel() {
  const models = draft.value?.models;
  if (!models) return;
  models.push(draftModel(emptyModel()));

  modelFilter.value = "";
  modelPage.value = Math.max(1, Math.ceil(models.length / modelPageSize.value));
}

function hasImage(model: AgentModel): boolean {
  return model.input.includes("image");
}

function setImage(model: AgentModel, event: Event): void {
  const checked = (event.target as HTMLInputElement).checked;
  const input = model.input.filter((entry) => entry !== "image");
  model.input = checked ? [...input, "image"] : input.length ? input : ["text"];
}

function rowClass(model: AgentModel): string {
  return notReturned(model)
    ? `${MODEL_GRID} border-destructive/60`
    : MODEL_GRID;
}

function notReturned(model: AgentModel): boolean {
  const id = model.id.trim();
  return Boolean(fetchedIds.value && id && !fetchedIds.value.has(id));
}

async function load() {
  loading.value = true;
  notFound.value = false;
  try {
    config.value = await agentConfigApi.get();
    if (!editing.value) {
      draft.value = { ...emptyProvider(), models: [] };
      return;
    }
    const provider = providerList.value.find(
      (entry) => entry.slug === slug.value,
    );
    if (!provider) {
      notFound.value = true;
      draft.value = null;
      return;
    }
    draft.value = {
      slug: provider.slug,
      name: provider.name,
      api: provider.api,
      baseUrl: provider.baseUrl,
      apiKey: "",
      headers: provider.headers ?? {},
      enabled: provider.enabled,
      models: provider.models.map((model) =>
        draftModel({
          ...model,
          input: [...model.input],
          cost: { ...model.cost },
        }),
      ),
      grantees: provider.grantees.map((grant) => ({ ...grant })),
    };
  } catch (err) {
    formError.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function fetchModels() {
  const current = draft.value;
  if (!current || fetching.value || !canFetch.value) return;
  fetching.value = true;
  fetchError.value = "";
  try {
    const discovered = await agentConfigApi.discoverModels({
      slug: editing.value ? current.slug : undefined,
      api: current.api,
      baseUrl: current.baseUrl,
      apiKey: current.apiKey,
      headers: current.headers,
    });
    const remaining = new Map(
      discovered.map((model) => [model.id.trim(), model]),
    );
    for (const model of current.models) {
      const found = remaining.get(model.id.trim());
      if (!found) continue;
      model.name = found.name;
      model.reasoning = found.reasoning;
      model.input = [...found.input];
      model.thinkingLevels = sortThinkingLevels(found.thinkingLevels ?? []);
      model.contextWindow = found.contextWindow;
      model.maxTokens = found.maxTokens;
      model.cost = { ...found.cost };
      remaining.delete(model.id.trim());
    }
    for (const model of remaining.values()) {
      current.models.push(
        draftModel({
          ...model,
          input: [...model.input],
          cost: { ...model.cost },

          enabled: false,
        }),
      );
    }
    fetchedIds.value = new Set(
      discovered.map((model) => model.id.trim()).filter(Boolean),
    );
    toast.success(
      t("providerEditor.fetchedToast", { count: discovered.length }),
    );
  } catch (err) {
    fetchError.value = err instanceof Error ? err.message : String(err);
  } finally {
    fetching.value = false;
  }
}

function toProviderInput(current: DraftProvider): ProviderInput {
  return {
    ...current,
    models: current.models.map((model) => ({
      id: model.id,
      name: model.name,
      reasoning: model.reasoning,
      input: [...model.input],
      thinkingLevels: [...model.thinkingLevels],
      contextWindow: model.contextWindow,
      maxTokens: model.maxTokens,
      cost: { ...model.cost },
      enabled: model.enabled,
    })),
  };
}

async function save() {
  const current = draft.value;
  if (!current || busy.value) return;
  if (current.models.some((model) => !model.id.trim())) {
    formError.value = t("providerEditor.blankRow");
    return;
  }

  if (
    !editing.value &&
    providerList.value.some((provider) => provider.slug === current.slug.trim())
  ) {
    formError.value = t("providerEditor.slugTaken");
    return;
  }
  busy.value = true;
  formError.value = "";
  try {
    config.value = await agentConfigApi.saveProvider(toProviderInput(current));
    toast.success(
      editing.value
        ? t("providerEditor.savedToast")
        : t("providerEditor.createdToast"),
    );
    await router.push({ name: "settings", query: { tab: "models" } });
  } catch (err) {
    formError.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

onMounted(async () => {
  await load();
  await Promise.all([
    loadProjects(),
    loadAgentGroups(),
    teamApi
      .list()
      .then((payload) => {
        teams.value = payload.teams;
      })
      .catch(() => {
        teams.value = [];
      }),
  ]);
});
</script>

<i18n lang="yaml">
en:
  providerEditor:
    back: "Back to models"
    createTitle: "New provider"
    editTitle: "Edit {slug}"
    description: "A provider is a base URL, a wire API, and the models it serves. Its key stays on the agency and is attached by the proxy."
    connectionTitle: "Connection"
    connectionDescription: "The connection details and the key the agency holds. The key is never returned to this page."
    notFound: "This provider no longer exists. It may have been deleted in another tab."
    slug: "Slug"
    slugExample: "acme"
    slugHint: "Used as the provider id in model references. It cannot be changed after creation."
    name: "Name"
    nameExample: "Acme"
    api: "Wire API"
    pickApi: "Pick an API"
    baseUrl: "Base URL"
    baseUrlExample: "https://api.acme.example/v1"
    apiKey: "API key"
    keyUnchanged: "unchanged"
    keyHint: "Sealed on the agency and attached by the proxy; never sent to an agent."
    oauthUnsupported: "OAuth logins are not supported: a provider signed in through Claude Pro/Max, Codex or Copilot contributes no models, so use an API key."
    enabled: "Enabled"
    enabledHint: "A disabled provider is hidden from every picker and the proxy."
    scopesTitle: "Who may use it"
    addScope: "Add a scope"
    scope: "Scope"
    everyAgent: "every agent"
    agentIdHint: "Search agents…"
    teamIdHint: "Search teams…"
    projectIdHint: "Search projects…"
    groupIdHint: "Search agent groups…"
    models: "Models"
    allModels: "All models"
    modelsTitle: "Models"
    addModel: "Add a model"
    fetchModels: "Fetch models"
    fetchHint: "Read the model list from the provider. New models are added disabled; matching rows keep their enabled flag and every other field is overwritten."
    fetchNeedsConnection: "Fill in the API, base URL and API key to fetch models."
    fetchedCount: "Fetched {count} model(s) from the provider."
    fetchedToast: "Fetched {count} model(s)"
    missingCount: "{count} existing row(s) were not returned by the provider and are kept highlighted."
    modelsEmpty: "No models yet. Add one by hand, or fetch the list from the provider."
    filterModels: "Filter by name or id…"
    filterEmpty: "No models match this filter."
    resultPerPage: "Result per page"
    rangeOf: "{from}–{to} of {total}"
    firstPage: "First page"
    lastPage: "Last page"
    goToPage: "Go to page"
    goToPageNumber: "Page {page}"
    go: "Go"
    previousPage: "Previous"
    nextPage: "Next"
    modelId: "model id"
    modelName: "display name"
    context: "context"
    maxTokens: "max tokens"
    reasoning: "reasoning"
    image: "image"
    costInput: "input $/1M"
    costOutput: "output $/1M"
    costCacheRead: "cached input $/1M"
    costCacheWrite: "cache write $/1M"
    costHint: "Prices are per 1M tokens. Fetched models take the provider's own pricing (or the built-in catalog's when the provider omits it); usage is priced with these values, so correct a missing one here."
    thinkingLevels: "Thinking levels"
    thinkingLevelsEmpty: "None declared; the worker falls back to the built-in catalog."
    thinkingLevelsHint: "The levels this model accepts. A level that is not listed is one the provider rejects, so a task cannot be pinned to it. Fetched from the provider's reasoning efforts."
    addThinkingLevel: "Add level"
    addThinkingLevelHint: "Declare another thinking level for this model"
    removeThinkingLevel: "Remove {level}"
    modelNotReturnedHint: "The provider did not return this model id. It is kept as you entered it."
    blankRow: "Every model row needs an id. Remove the empty row, or fill it in."
    createEnabledHint: "The provider is created enabled. You can disable it on its edit page."
    slugTaken: "A provider with this slug already exists. Edit it instead of creating it again."
    savedToast: "Provider saved"
    createdToast: "Provider created"
ru:
  providerEditor:
    back: "Назад к моделям"
    createTitle: "Новый провайдер"
    editTitle: "Изменить {slug}"
    description: "Провайдер — это базовый URL, wire API и модели, которые он обслуживает. Его ключ остаётся на agency и подставляется прокси."
    connectionTitle: "Подключение"
    connectionDescription: "Параметры подключения и ключ, который хранит agency. Ключ никогда не возвращается на эту страницу."
    notFound: "Этот провайдер больше не существует. Возможно, его удалили в другой вкладке."
    slug: "Slug"
    slugExample: "acme"
    slugHint: "Используется как id провайдера в ссылках на модели. После создания его нельзя изменить."
    name: "Название"
    nameExample: "Acme"
    api: "Wire API"
    pickApi: "Выберите API"
    baseUrl: "Базовый URL"
    baseUrlExample: "https://api.acme.example/v1"
    apiKey: "API-ключ"
    keyUnchanged: "без изменений"
    keyHint: "Хранится зашифрованным на agency и подставляется прокси; агенту не передаётся."
    oauthUnsupported: "OAuth-логины не поддерживаются: провайдер, подключённый через Claude Pro/Max, Codex или Copilot, не даёт ни одной модели — используйте API-ключ."
    enabled: "Включён"
    enabledHint: "Отключённый провайдер скрыт из всех выборов и из прокси."
    scopesTitle: "Кому можно использовать"
    addScope: "Добавить область"
    scope: "Область"
    everyAgent: "все агенты"
    agentIdHint: "Поиск агентов…"
    teamIdHint: "Поиск команд…"
    projectIdHint: "Поиск проектов…"
    groupIdHint: "Поиск групп агентов…"
    models: "Модели"
    allModels: "Все модели"
    modelsTitle: "Модели"
    addModel: "Добавить модель"
    fetchModels: "Получить модели"
    fetchHint: "Прочитать список моделей у провайдера. Новые модели добавляются отключёнными; у совпавших строк сохраняется флаг включения, остальные поля перезаписываются."
    fetchNeedsConnection: "Заполните API, базовый URL и API-ключ, чтобы получить модели."
    fetchedCount: "Получено моделей от провайдера: {count}."
    fetchedToast: "Получено моделей: {count}"
    missingCount: "Провайдер не вернул строк: {count}. Они сохранены и подсвечены."
    modelsEmpty: "Моделей пока нет. Добавьте вручную или получите список у провайдера."
    filterModels: "Фильтр по названию или id…"
    filterEmpty: "Ни одна модель не подходит под фильтр."
    resultPerPage: "Строк на странице"
    rangeOf: "{from}–{to} из {total}"
    firstPage: "Первая страница"
    lastPage: "Последняя страница"
    goToPage: "К странице"
    goToPageNumber: "Страница {page}"
    go: "Перейти"
    previousPage: "Назад"
    nextPage: "Вперёд"
    modelId: "id модели"
    modelName: "отображаемое имя"
    context: "контекст"
    maxTokens: "макс. токенов"
    reasoning: "reasoning"
    image: "изображения"
    costInput: "вход $/1M"
    costOutput: "выход $/1M"
    costCacheRead: "кэш-вход $/1M"
    costCacheWrite: "запись кэша $/1M"
    costHint: "Цены указаны за 1M токенов. Полученные модели берут цены провайдера (или встроенного каталога, если провайдер их не прислал); по ним считается стоимость использования, поэтому исправьте пропущенное значение здесь."
    thinkingLevels: "Уровни размышления"
    thinkingLevelsEmpty: "Ничего не объявлено; агент возьмёт данные из встроенного каталога."
    thinkingLevelsHint: "Уровни, которые принимает эта модель. Уровень, которого нет в списке, провайдер отвергает, поэтому задачу нельзя закрепить на нём. Заполняется из reasoning efforts провайдера."
    addThinkingLevel: "Добавить уровень"
    addThinkingLevelHint: "Объявить для этой модели ещё один уровень размышления"
    removeThinkingLevel: "Убрать {level}"
    modelNotReturnedHint: "Провайдер не вернул этот id модели. Строка сохранена как вы её ввели."
    blankRow: "У каждой строки модели должен быть id. Удалите пустую строку или заполните её."
    createEnabledHint: "Провайдер создаётся включённым. Отключить его можно на странице редактирования."
    slugTaken: "Провайдер с таким slug уже существует. Отредактируйте его вместо повторного создания."
    savedToast: "Провайдер сохранён"
    createdToast: "Провайдер создан"
</i18n>
