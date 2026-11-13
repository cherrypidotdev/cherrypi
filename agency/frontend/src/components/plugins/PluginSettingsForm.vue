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
form.flex.flex-col.gap-4(@submit.prevent="save")
  Collapsible.overflow-hidden.rounded-md.border(
    v-if="setup"
    v-model:open="setupOpen"
    class="bg-muted/30 group/setup"
  )
    CollapsibleTrigger.flex.w-full.cursor-pointer.items-center.gap-2.px-4.py-3.text-left
      ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
        class="group-data-[state=open]/setup:rotate-90"
      )
      span.text-sm.font-medium {{ setup.title }}
    CollapsibleContent
      .border-t.px-4.py-3
        MarkdownText.prose.prose-sm.max-w-none(
          class="dark:prose-invert prose-p:my-1 prose-pre:my-2"
          :text="setup.markdown"
        )
  template(v-if="hasProperties")
    .flex.flex-wrap.items-center.justify-between.gap-2(
      v-if="settingsGroups.length || Object.keys(templateVariables).length"
    )
      p.text-muted-foreground.text-xs(v-if="settingsGroups.length")
        | {{ $t("pluginSettings.grouped") }}
      .flex.items-center.gap-1
        template(v-if="settingsGroups.length")
          Button(
            type="button"
            variant="ghost"
            size="sm"
            @click="setAllGroups(true)"
          )
            | {{ $t("pluginSettings.expandAll") }}
          Button(
            type="button"
            variant="ghost"
            size="sm"
            @click="setAllGroups(false)"
          )
            | {{ $t("pluginSettings.collapseAll") }}
        Button(
          v-if="Object.keys(templateVariables).length"
          type="button"
          variant="outline"
          size="sm"
          @click="samplesOpen = true"
        )
          CodeIcon(data-icon="inline-start")
          span {{ $t("pluginSettings.sampleData") }}
    FieldGroup
      template(v-for="row in rows" :key="row.key")
        template(v-if="row.type === 'group'")
          button.flex.w-full.items-center.gap-2.border.px-3.py-3.text-left(
            type="button"
            class="bg-muted/30"
            :class="groupHeaderClass(row)"
            :aria-expanded="groupOpen[row.id]"
            @click="toggleGroup(row.id)"
          )
            ChevronRightIcon.text-muted-foreground.size-4.shrink-0.transition-transform(
              :class="groupOpen[row.id] ? 'rotate-90' : ''"
            )
            .min-w-0.flex-1
              .flex.flex-wrap.items-center.gap-2
                span.text-sm.font-medium {{ row.title }}
                span.text-muted-foreground.text-xs {{ $t("pluginSettings.settingsCount", { count: row.count }) }}
              p.text-muted-foreground.text-xs(v-if="row.description") {{ row.description }}
        Field(
          v-else-if="row.type === 'field' && isRowVisible(row)"
          :class="fieldRowClass(row)"
        )
          PluginTemplateField(
            v-if="fieldOf(row).kind === 'template'"
            :hint="fieldOf(row).templateHint"
            :embedded="row.groupId !== null"
            :field-key="fieldOf(row).key"
            :title="fieldOf(row).title"
            :description="fieldOf(row).description"
            :default-value="fieldOf(row).defaultValue || ''"
            :template="fieldOf(row).template ?? null"
            :model-value="String(values[fieldOf(row).key] ?? '')"
            :error="errorFor(fieldOf(row).key)"
            :plugin-id="pluginId"
            :scope="scope"
            :target-id="targetId"
            :sample-overrides="sampleOverrides"
            @update:model-value="values[fieldOf(row).key] = $event"
          )
          template(v-else)
            FieldLabel(:for="`plugin-setting-${fieldOf(row).key}`")
              | {{ fieldOf(row).title }}
              span.text-destructive(v-if="fieldOf(row).required") *
            template(v-if="fieldOf(row).kind === 'boolean'")
              Switch(
                :id="`plugin-setting-${fieldOf(row).key}`"
                :model-value="Boolean(values[fieldOf(row).key])"
                @update:model-value="values[fieldOf(row).key] = Boolean($event)"
              )
            Select(
              v-else-if="fieldOf(row).kind === 'enum'"
              :model-value="String(values[fieldOf(row).key] ?? '')"
              @update:model-value="values[fieldOf(row).key] = $event"
            )
              SelectTrigger.w-full(:id="`plugin-setting-${fieldOf(row).key}`")
                SelectValue(
                  :placeholder="fieldOf(row).placeholder || $t('pluginSettings.choose')"
                )
              SelectContent
                SelectItem(
                  v-for="(option, index) in fieldOf(row).options"
                  :key="String(option)"
                  :value="String(option)"
                ) {{ fieldOf(row).optionLabels[index] }}
            .flex.flex-wrap.gap-3(
              v-else-if="fieldOf(row).kind === 'enum-array'"
            )
              label.flex.items-center.gap-2.rounded-md.border.px-3.py-2.text-sm(
                v-for="(option, index) in fieldOf(row).itemOptions"
                :key="String(option)"
              )
                input.accent-primary(
                  type="checkbox"
                  :checked="arrayValue(fieldOf(row)).includes(option)"
                  :id="`plugin-setting-${fieldOf(row).key}-${String(option)}`"
                  @change="toggleEnumItem(fieldOf(row), option)"
                )
                span {{ fieldOf(row).itemOptionLabels[index] }}
            .flex.flex-col.gap-2(v-else-if="fieldOf(row).kind === 'array'")
              .flex.items-center.gap-2(
                v-for="(item, index) in arrayValue(fieldOf(row))"
                :key="`${fieldOf(row).key}-${index}`"
              )
                Switch(
                  v-if="fieldOf(row).itemKind === 'boolean'"
                  :model-value="Boolean(item)"
                  @update:model-value="setArrayItem(fieldOf(row), index, Boolean($event))"
                )
                Select(
                  v-else-if="fieldOf(row).itemKind === 'enum'"
                  :model-value="String(item ?? '')"
                  @update:model-value="setArrayItem(fieldOf(row), index, $event)"
                )
                  SelectTrigger.flex-1
                    SelectValue(:placeholder="$t('pluginSettings.choose')")
                  SelectContent
                    SelectItem(
                      v-for="(option, index) in fieldOf(row).itemOptions"
                      :key="String(option)"
                      :value="String(option)"
                    ) {{ fieldOf(row).itemOptionLabels[index] }}
                Textarea.font-mono.text-xs(
                  v-else-if="fieldOf(row).itemKind === 'json'"
                  :placeholder="$t('pluginSettings.jsonValue')"
                  rows="3"
                  :model-value="itemDraft(fieldOf(row), index)"
                  @update:model-value="setItemDraft(fieldOf(row), index, String($event))"
                )
                Input.flex-1(
                  v-else
                  :type="fieldOf(row).itemKind === 'number' ? 'number' : 'text'"
                  :placeholder="fieldOf(row).placeholder"
                  :model-value="String(item ?? '')"
                  @update:model-value="setArrayItem(fieldOf(row), index, $event)"
                )
                Button(
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="$t('pluginSettings.removeItem')"
                  @click="removeArrayItem(fieldOf(row), index)"
                )
                  MinusIcon
              Button.self-start(
                type="button"
                variant="outline"
                size="sm"
                @click="addArrayItem(fieldOf(row))"
              )
                PlusIcon(data-icon="inline-start")
                span {{ $t("common.add") }}
            Textarea.font-mono.text-xs(
              v-else-if="fieldOf(row).kind === 'json'"
              v-model="jsonDrafts[fieldOf(row).key]"
              :placeholder="fieldOf(row).placeholder"
              rows="4"
              :id="`plugin-setting-${fieldOf(row).key}`"
              @blur="commitJSON(fieldOf(row).key)"
            )
            Textarea(
              v-else-if="fieldOf(row).multiline"
              :placeholder="fieldOf(row).placeholder"
              rows="3"
              :id="`plugin-setting-${fieldOf(row).key}`"
              :model-value="String(values[fieldOf(row).key] ?? '')"
              @update:model-value="values[fieldOf(row).key] = $event"
            )
            .flex.items-center.gap-1(v-else)
              Input.flex-1(
                :type="fieldOf(row).secret && !revealed[fieldOf(row).key] ? 'password' : fieldOf(row).kind === 'number' ? 'number' : 'text'"
                :placeholder="fieldOf(row).placeholder"
                :id="`plugin-setting-${fieldOf(row).key}`"
                :model-value="String(values[fieldOf(row).key] ?? '')"
                :autocomplete="fieldOf(row).secret ? 'new-password' : 'off'"
                @update:model-value="values[fieldOf(row).key] = $event"
              )
              template(v-if="fieldOf(row).secret")
                Button(
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="revealed[fieldOf(row).key] ? $t('pluginSettings.hideValue') : $t('pluginSettings.revealValue')"
                  @click="revealed[fieldOf(row).key] = !revealed[fieldOf(row).key]"
                )
                  EyeOffIcon(v-if="revealed[fieldOf(row).key]")
                  EyeIcon(v-else)
                Button(
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="$t('pluginSettings.copyValue')"
                  @click="copyValue(fieldOf(row))"
                  :disabled="!values[fieldOf(row).key]"
                )
                  CheckIcon(v-if="copied === fieldOf(row).key")
                  CopyIcon(v-else)
            FieldDescription(v-if="fieldOf(row).description") {{ fieldOf(row).description }}
            FieldDescription.text-destructive(
              v-if="errorFor(fieldOf(row).key)"
            )
              | {{ errorFor(fieldOf(row).key) }}
            FieldDescription(
              v-else-if="fieldOf(row).secret && !values[fieldOf(row).key]"
            )
              | {{ $t("pluginSettings.emptyValue") }}
            FieldDescription(v-else-if="fieldOf(row).secret")
              | {{ $t("pluginSettings.storedValue") }}
  p.text-muted-foreground.text-sm(v-else-if="declaresNoValues")
    | {{ $t("pluginSettings.noForm") }}
  template(v-else)
    Field
      FieldLabel(for="plugin-settings-json") {{ $t("common.settings") }}
      Textarea#plugin-settings-json.font-mono.text-xs(
        v-model="rawDraft"
        placeholder="{  }"
        rows="6"
      )
      FieldDescription {{ $t("pluginSettings.jsonObjectHint") }}

  p.text-destructive.text-sm(v-if="localError || error" role="alert")
    | {{ localError || error }}
  .flex.flex-col.gap-1(v-if="unmatchedErrors.length")
    p.text-destructive.text-sm(
      v-for="item in unmatchedErrors"
      :key="item.path"
    )
      | {{ item.errorText }}
  p.text-muted-foreground.text-sm(v-else-if="saved") {{ $t("pluginSettings.saved") }}
  .flex.justify-end(v-if="!declaresNoValues")
    Button(type="submit" :disabled="saving")
      Spinner(v-if="saving" data-icon="inline-start")
      CheckIcon(v-else data-icon="inline-start")
      span {{ $t("common.save") }}
  PluginTemplateSamplesDialog(
    v-if="Object.keys(templateVariables).length"
    v-model:open="samplesOpen"
    :variables="templateVariables"
    :model-value="sampleOverrides"
    @update:model-value="setSampleOverrides"
  )
</template>

<script setup lang="ts">
import MarkdownText from "@/components/MarkdownText.vue";
import PluginTemplateField from "@/components/plugins/PluginTemplateField.vue";
import PluginTemplateSamplesDialog from "@/components/plugins/PluginTemplateSamplesDialog.vue";
import { Button } from "@/components/ui/button";
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
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  parseJSONSchema,
  pluginSchemaForm,
  pluginSettingsGroups,
  pluginTemplateDefault,
  pluginTemplateMeta,
  pluginTemplateVariables,
  type PluginTemplateMeta,
  type PluginTemplateVariable,
} from "@/lib/plugins";
import {
  CheckIcon,
  ChevronRightIcon,
  CodeIcon,
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  MinusIcon,
  PlusIcon,
} from "@lucide/vue";
import { computed, reactive, ref, watch } from "vue";
import { toast } from "vue3-toastify";

interface SettingsSetup {
  title?: string;
  markdown?: string;
}

interface SchemaProperty {
  type?: string;
  title?: string;
  description?: string;

  setup?: string | SettingsSetup;
  enum?: unknown[];

  enumNames?: unknown[];
  format?: string;
  secret?: boolean;
  required?: string[];
  default?: unknown;
  multiline?: boolean;
  "x-template"?: unknown;
  properties?: Record<string, SchemaProperty>;
  items?: SchemaProperty;
}

interface SettingFieldError {
  path: string;
  errorText: string;
}

type ItemKind = "string" | "number" | "boolean" | "enum" | "json";

const props = defineProps<{
  schema?: string;
  settings: Record<string, unknown>;
  secretKeys?: string[];
  saving?: boolean;
  error?: string;
  saved?: boolean;
  errors?: SettingFieldError[];
  pluginId?: string;
  scope?: string;
  targetId?: string;
}>();

const emit = defineEmits<{
  save: [values: Record<string, unknown>];
}>();

const values = reactive<Record<string, unknown>>({});
const jsonDrafts = reactive<Record<string, string>>({});
const itemDrafts = reactive<Record<string, string>>({});
const revealed = reactive<Record<string, boolean>>({});
const rawDraft = ref("{}");
const localError = ref("");
const copied = ref("");
const setupOpen = ref(true);

const parsedSchema = computed(
  () => parseJSONSchema(props.schema) as SchemaProperty | null,
);

const form = computed(() => pluginSchemaForm(props.schema));
const hasProperties = computed(() => form.value === "form");

const setup = computed<{ title: string; markdown: string } | null>(() => {
  const raw = parsedSchema.value?.setup;
  if (typeof raw === "string") {
    return raw.trim() ? { title: "Setup", markdown: raw } : null;
  }
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const markdown = typeof raw.markdown === "string" ? raw.markdown : "";
  if (!markdown.trim()) return null;
  const title = typeof raw.title === "string" ? raw.title.trim() : "";
  return { title: title || "Setup", markdown };
});

const declaresNoValues = computed(() => form.value === "none");

interface FieldSpec {
  key: string;
  title: string;
  description?: string;
  placeholder?: string;
  kind:
    | "string"
    | "number"
    | "boolean"
    | "enum"
    | "json"
    | "array"
    | "enum-array"
    | "template";
  itemKind?: ItemKind;
  options: unknown[];
  itemOptions: unknown[];
  optionLabels: string[];
  itemOptionLabels: string[];
  secret: boolean;
  multiline: boolean;
  required: boolean;
  defaultValue?: string;
  template?: PluginTemplateMeta | null;
  templateHint?: string;
}

function enumLabels(options: unknown, names: unknown): string[] {
  if (!Array.isArray(options)) return [];
  const labels = Array.isArray(names) ? names : [];
  return options.map((option, index) => {
    const label = labels[index];
    return typeof label === "string" && label.trim()
      ? label.trim()
      : String(option);
  });
}

function itemKindOf(property: SchemaProperty | undefined): ItemKind {
  if (Array.isArray(property?.enum) && property.enum.length) return "enum";
  switch (property?.type) {
    case "boolean":
      return "boolean";
    case "number":
    case "integer":
      return "number";
    case "object":
    case "array":
      return "json";
    default:
      return "string";
  }
}

const fields = computed<FieldSpec[]>(() => {
  const properties = parsedSchema.value?.properties ?? {};
  const required = new Set(parsedSchema.value?.required ?? []);
  const secrets = new Set(props.secretKeys ?? []);
  return Object.entries(properties).map(([key, property]) => {
    const type = property.type ?? "string";

    const options = Array.isArray(property.enum) ? property.enum : [];
    const itemEnum = property.items?.enum;
    const itemOptions = Array.isArray(itemEnum) ? itemEnum : [];
    const template = pluginTemplateMeta(property as Record<string, unknown>);
    const kind: FieldSpec["kind"] = template
      ? "template"
      : type === "boolean"
        ? "boolean"
        : type === "number" || type === "integer"
          ? "number"
          : type === "array"
            ? Array.isArray(property.items?.enum) && property.items.enum.length
              ? "enum-array"
              : "array"
            : type === "object"
              ? "json"
              : Array.isArray(property.enum) && property.enum.length > 0
                ? "enum"
                : "string";
    return {
      key,
      title: property.title || key,
      description: property.description,
      placeholder: property.description,
      kind,
      itemKind: type === "array" ? itemKindOf(property.items) : undefined,
      options,
      itemOptions,
      optionLabels: enumLabels(options, property.enumNames),
      itemOptionLabels: enumLabels(itemOptions, property.items?.enumNames),
      secret:
        secrets.has(key) ||
        property.secret === true ||
        property.format === "password",
      multiline: property.format === "textarea" || property.multiline === true,
      required: required.has(key),
      defaultValue: template
        ? pluginTemplateDefault(property as Record<string, unknown>)
        : undefined,
      template,
      templateHint: template
        ? "Rendered by the agency with each event's variables."
        : undefined,
    };
  });
});

const settingsGroups = computed(() => pluginSettingsGroups(props.schema));
const templateVariables = computed<Record<string, PluginTemplateVariable>>(() =>
  pluginTemplateVariables(props.schema),
);
const samplesOpen = ref(false);
const sampleOverrides = reactive<Record<string, unknown>>({});
const groupOpen = reactive<Record<string, boolean>>({});
let groupSignature = "";

watch(
  settingsGroups,
  (groups) => {
    const signature = groups
      .map((group) => `${group.id}:${group.title}:${group.fields.join(",")}`)
      .join("|");
    if (signature !== groupSignature) {
      groupSignature = signature;
      for (const id of Object.keys(groupOpen)) delete groupOpen[id];
    }
    for (const group of groups) {
      if (groupOpen[group.id] === undefined) groupOpen[group.id] = false;
    }
  },
  { immediate: true, deep: true },
);

function toggleGroup(id: string) {
  groupOpen[id] = !groupOpen[id];
}

function setAllGroups(open: boolean) {
  for (const group of settingsGroups.value) groupOpen[group.id] = open;
}

interface SettingsRow {
  type: "group" | "field";
  key: string;
  id: string;
  title: string;
  description?: string;
  count: number;
  field: FieldSpec | null;
  groupId: string | null;
  lastInGroup: boolean;
}

function fieldOf(row: SettingsRow): FieldSpec {
  return (
    row.field ?? {
      key: "",
      title: "",
      kind: "string",
      options: [],
      itemOptions: [],
      optionLabels: [],
      itemOptionLabels: [],
      secret: false,
      multiline: false,
      required: false,
    }
  );
}

const rows = computed<SettingsRow[]>(() => {
  const byKey = new Map(fields.value.map((field) => [field.key, field]));
  const groups = settingsGroups.value;
  const fieldRow = (field: FieldSpec, groupId: string | null): SettingsRow => ({
    type: "field",
    key: `field:${field.key}`,
    id: "",
    title: "",
    count: 0,
    field,
    groupId,
    lastInGroup: false,
  });
  if (!groups.length) {
    return fields.value.map((field) => fieldRow(field, null));
  }
  const grouped = new Set(groups.flatMap((group) => group.fields));
  const out: SettingsRow[] = [];
  const pushGroup = (
    id: string,
    title: string,
    description: string | undefined,
    keys: string[],
  ) => {
    const groupFields = keys
      .map((key) => byKey.get(key))
      .filter((field): field is FieldSpec => !!field);
    if (!groupFields.length) return;
    out.push({
      type: "group",
      key: `group:${id}`,
      id,
      title,
      description,
      count: groupFields.length,
      field: null,
      groupId: id,
      lastInGroup: false,
    });
    for (const field of groupFields) out.push(fieldRow(field, id));
    out[out.length - 1]!.lastInGroup = true;
  };
  for (const group of groups) {
    pushGroup(group.id, group.title, group.description, group.fields);
  }
  const other = fields.value
    .filter((field) => !grouped.has(field.key))
    .map((field) => field.key);
  pushGroup("other", "Other", undefined, other);
  return out;
});

function isRowVisible(row: SettingsRow): boolean {
  if (row.type !== "field") return false;
  return row.groupId === null || groupOpen[row.groupId] !== false;
}

function groupHeaderClass(row: SettingsRow): string {
  return groupOpen[row.id] ? "rounded-t-md border-b" : "rounded-md";
}

function fieldRowClass(row: SettingsRow): string {
  if (row.groupId === null) {
    return "rounded-md border bg-card px-3 py-3";
  }
  const base = "border-x border-b bg-card px-4 py-3 pl-7";
  return row.lastInGroup ? `${base} rounded-b-md` : base;
}

function setSampleOverrides(value: Record<string, unknown>) {
  for (const key of Object.keys(sampleOverrides)) delete sampleOverrides[key];
  Object.assign(sampleOverrides, value);
}

const unmatchedErrors = computed(() =>
  (props.errors ?? []).filter(
    (item) =>
      !fields.value.some(
        (field) =>
          item.path === field.key || item.path.startsWith(`${field.key}.`),
      ),
  ),
);

watch(
  () => props.saved,
  (saved) => {
    if (saved) toast.success("Settings saved");
  },
);

const errorSignature = computed(() =>
  JSON.stringify(
    (props.errors ?? []).map((item) => [item.path, item.errorText]),
  ),
);

watch(errorSignature, (signature) => {
  const count = JSON.parse(signature).length as number;
  if (count === 0) return;
  toast.error(
    count === 1
      ? "Settings were not saved: 1 field needs attention."
      : `Settings were not saved: ${count} fields need attention.`,
  );
});

watch(
  () => props.error,
  (message) => {
    if (message) toast.error(message);
  },
);

function reset() {
  for (const key of Object.keys(values)) delete values[key];
  for (const key of Object.keys(jsonDrafts)) delete jsonDrafts[key];
  for (const key of Object.keys(itemDrafts)) delete itemDrafts[key];
  for (const key of Object.keys(revealed)) delete revealed[key];
  for (const [key, value] of Object.entries(props.settings ?? {})) {
    values[key] = value;
  }
  for (const field of fields.value) {
    if (field.kind === "json") {
      jsonDrafts[field.key] = JSON.stringify(
        values[field.key] ?? null,
        null,
        2,
      );
    }
    if (field.kind === "boolean" && values[field.key] === undefined) {
      values[field.key] = false;
    }
    if (field.kind === "array" || field.kind === "enum-array") {
      if (!Array.isArray(values[field.key])) values[field.key] = [];
      if (field.kind === "array" && field.itemKind === "json") {
        arrayValue(field).forEach((_, index) => {
          itemDrafts[draftKey(field, index)] = JSON.stringify(
            arrayValue(field)[index] ?? null,
            null,
            2,
          );
        });
      }
    }
  }
  for (const key of Object.keys(sampleOverrides)) delete sampleOverrides[key];
  rawDraft.value = JSON.stringify(props.settings ?? {}, null, 2);
  localError.value = "";
}

const sourceSignature = computed(() =>
  JSON.stringify([props.schema ?? "", props.settings ?? {}]),
);

watch(sourceSignature, () => reset(), { immediate: true });

function arrayValue(field: FieldSpec): unknown[] {
  const value = values[field.key];
  return Array.isArray(value) ? value : [];
}

function draftKey(field: FieldSpec, index: number): string {
  return `${field.key}:${index}`;
}

function itemDraft(field: FieldSpec, index: number): string {
  const key = draftKey(field, index);
  if (itemDrafts[key] === undefined) {
    itemDrafts[key] = JSON.stringify(arrayValue(field)[index] ?? null, null, 2);
  }
  return itemDrafts[key];
}

function setItemDraft(field: FieldSpec, index: number, value: string) {
  itemDrafts[draftKey(field, index)] = value;
}

function setArrayItem(field: FieldSpec, index: number, value: unknown) {
  const list = arrayValue(field).slice();
  list[index] =
    field.itemKind === "number" && value !== "" ? Number(value) : value;
  values[field.key] = list;
}

function addArrayItem(field: FieldSpec) {
  const list = arrayValue(field).slice();
  const empty =
    field.itemKind === "boolean"
      ? false
      : field.itemKind === "number"
        ? 0
        : field.itemKind === "json"
          ? null
          : "";
  list.push(empty);
  values[field.key] = list;
  if (field.itemKind === "json") {
    itemDrafts[draftKey(field, list.length - 1)] = JSON.stringify(
      empty,
      null,
      2,
    );
  }
}

function removeArrayItem(field: FieldSpec, index: number) {
  const list = arrayValue(field).slice();
  list.splice(index, 1);
  values[field.key] = list;
  for (const key of Object.keys(itemDrafts)) {
    if (key.startsWith(`${field.key}:`)) delete itemDrafts[key];
  }
  if (field.itemKind === "json") {
    list.forEach((item, i) => {
      itemDrafts[draftKey(field, i)] = JSON.stringify(item ?? null, null, 2);
    });
  }
}

function toggleEnumItem(field: FieldSpec, option: unknown) {
  const list = arrayValue(field).slice();
  const index = list.indexOf(option);
  if (index >= 0) list.splice(index, 1);
  else list.push(option);
  values[field.key] = list;
}

function commitJSON(key: string) {
  try {
    values[key] = JSON.parse(jsonDrafts[key] || "null");
    localError.value = "";
  } catch {
    localError.value = `${key} must be valid JSON`;
  }
}

function errorFor(key: string): string {
  return (
    props.errors?.find(
      (item) => item.path === key || item.path.startsWith(`${key}.`),
    )?.errorText ?? ""
  );
}

async function copyValue(field: FieldSpec) {
  const value = String(values[field.key] ?? "");
  if (!value) return;
  try {
    await navigator.clipboard.writeText(value);
    copied.value = field.key;
    window.setTimeout(() => {
      if (copied.value === field.key) copied.value = "";
    }, 1500);
  } catch {
    localError.value = "Could not copy to the clipboard";
  }
}

function save() {
  if (!hasProperties.value) {
    try {
      const parsed = JSON.parse(rawDraft.value || "{}");
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        localError.value = "Settings must be a JSON object";
        return;
      }
      localError.value = "";
      emit("save", parsed as Record<string, unknown>);
      return;
    } catch {
      localError.value = "Settings must be valid JSON";
      return;
    }
  }
  for (const field of fields.value) {
    if (field.kind === "json") {
      commitJSON(field.key);
    }
    if (field.kind === "array") {
      const list = arrayValue(field).slice();
      if (field.itemKind === "json") {
        for (let index = 0; index < list.length; index += 1) {
          try {
            list[index] = JSON.parse(itemDraft(field, index) || "null");
          } catch {
            localError.value = `${field.title} item ${index + 1} must be valid JSON`;
            return;
          }
        }
        values[field.key] = list;
      }
      if (field.itemKind === "number") {
        values[field.key] = list.map((item) =>
          item === "" ? 0 : Number(item),
        );
      }
    }
    if (field.kind === "number") {
      const raw = values[field.key];
      if (raw === "" || raw === undefined || raw === null) {
        delete values[field.key];
      } else {
        const number = Number(raw);
        if (Number.isNaN(number)) {
          localError.value = `${field.title} must be a number`;
          return;
        }
        values[field.key] = number;
      }
    }
  }
  if (localError.value) return;
  const payload = { ...values };

  for (const field of fields.value) {
    if (field.kind !== "template") continue;
    if (payload[field.key] === field.defaultValue) {
      delete payload[field.key];
    }
  }
  emit("save", payload);
}
</script>

<i18n lang="yaml">
en:
  pluginSettings:
    emptyValue: "No value stored. Leave empty to keep it unset."
    storedValue: "Stored. Use the eye to reveal or the copy button."
    noForm: "This plugin declares no settings form."
    jsonObjectHint: "A JSON object; this plugin did not declare a form schema."
    saved: "Saved."
    choose: "Choose…"
    jsonValue: "JSON value"
    removeItem: "Remove item"
    copyValue: "Copy value"
    grouped: "Grouped settings"
    expandAll: "Expand all"
    collapseAll: "Collapse all"
    sampleData: "Sample data"
    hideValue: "Hide value"
    revealValue: "Reveal value"
    settingsCount: "{count} setting | {count} settings"
ru:
  pluginSettings:
    emptyValue: "Значение не сохранено. Оставьте пустым, чтобы не задавать его."
    storedValue: "Сохранено. Используйте глаз, чтобы показать, или кнопку копирования."
    noForm: "Этот плагин не объявляет форму настроек."
    jsonObjectHint: "Объект JSON; этот плагин не объявил схему формы."
    saved: "Сохранено."
    choose: "Выберите…"
    jsonValue: "Значение JSON"
    removeItem: "Удалить элемент"
    copyValue: "Скопировать значение"
    grouped: "Сгруппированные настройки"
    expandAll: "Развернуть все"
    collapseAll: "Свернуть все"
    sampleData: "Пример данных"
    hideValue: "Скрыть значение"
    revealValue: "Показать значение"
    settingsCount: "{count} настройка | {count} настройки | {count} настроек"
</i18n>
