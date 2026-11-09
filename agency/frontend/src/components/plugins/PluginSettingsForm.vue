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
  template(v-if="hasProperties")
    FieldGroup
      Field(v-for="field in fields" :key="field.key")
        FieldLabel(:for="`plugin-setting-${field.key}`")
          | {{ field.title }}
          span.text-destructive(v-if="field.required") *
        template(v-if="field.kind === 'boolean'")
          Switch(
            :id="`plugin-setting-${field.key}`"
            :model-value="Boolean(values[field.key])"
            @update:model-value="values[field.key] = Boolean($event)"
          )
        Select(
          v-else-if="field.kind === 'enum'"
          :model-value="String(values[field.key] ?? '')"
          @update:model-value="values[field.key] = $event"
        )
          SelectTrigger.w-full(:id="`plugin-setting-${field.key}`")
            SelectValue(:placeholder="field.placeholder || 'Choose…'")
          SelectContent
            SelectItem(
              v-for="option in field.options"
              :key="String(option)"
              :value="String(option)"
            ) {{ option }}
        .flex.flex-wrap.gap-3(v-else-if="field.kind === 'enum-array'")
          label.flex.items-center.gap-2.rounded-md.border.px-3.py-2.text-sm(
            v-for="option in field.itemOptions"
            :key="String(option)"
          )
            input.accent-primary(
              type="checkbox"
              :checked="arrayValue(field).includes(option)"
              :id="`plugin-setting-${field.key}-${String(option)}`"
              @change="toggleEnumItem(field, option)"
            )
            span {{ option }}
        .flex.flex-col.gap-2(v-else-if="field.kind === 'array'")
          .flex.items-center.gap-2(
            v-for="(item, index) in arrayValue(field)"
            :key="`${field.key}-${index}`"
          )
            Switch(
              v-if="field.itemKind === 'boolean'"
              :model-value="Boolean(item)"
              @update:model-value="setArrayItem(field, index, Boolean($event))"
            )
            Select(
              v-else-if="field.itemKind === 'enum'"
              :model-value="String(item ?? '')"
              @update:model-value="setArrayItem(field, index, $event)"
            )
              SelectTrigger.flex-1
                SelectValue(placeholder="Choose…")
              SelectContent
                SelectItem(
                  v-for="option in field.itemOptions"
                  :key="String(option)"
                  :value="String(option)"
                ) {{ option }}
            Textarea.font-mono.text-xs(
              v-else-if="field.itemKind === 'json'"
              placeholder="JSON value"
              rows="3"
              :model-value="itemDraft(field, index)"
              @update:model-value="setItemDraft(field, index, String($event))"
            )
            Input.flex-1(
              v-else
              :type="field.itemKind === 'number' ? 'number' : 'text'"
              :placeholder="field.placeholder"
              :model-value="String(item ?? '')"
              @update:model-value="setArrayItem(field, index, $event)"
            )
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Remove item"
              @click="removeArrayItem(field, index)"
            )
              MinusIcon
          Button.self-start(
            type="button"
            variant="outline"
            size="sm"
            @click="addArrayItem(field)"
          )
            PlusIcon(data-icon="inline-start")
            span Add
        Textarea.font-mono.text-xs(
          v-else-if="field.kind === 'json'"
          v-model="jsonDrafts[field.key]"
          :placeholder="field.placeholder"
          rows="4"
          :id="`plugin-setting-${field.key}`"
          @blur="commitJSON(field.key)"
        )
        Textarea(
          v-else-if="field.multiline"
          :placeholder="field.placeholder"
          rows="3"
          :id="`plugin-setting-${field.key}`"
          :model-value="String(values[field.key] ?? '')"
          @update:model-value="values[field.key] = $event"
        )
        .flex.items-center.gap-1(v-else)
          Input.flex-1(
            :type="field.secret && !revealed[field.key] ? 'password' : field.kind === 'number' ? 'number' : 'text'"
            :placeholder="field.placeholder"
            :id="`plugin-setting-${field.key}`"
            :model-value="String(values[field.key] ?? '')"
            :autocomplete="field.secret ? 'new-password' : 'off'"
            @update:model-value="values[field.key] = $event"
          )
          template(v-if="field.secret")
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              :aria-label="revealed[field.key] ? 'Hide value' : 'Reveal value'"
              @click="revealed[field.key] = !revealed[field.key]"
            )
              EyeOffIcon(v-if="revealed[field.key]")
              EyeIcon(v-else)
            Button(
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Copy value"
              @click="copyValue(field)"
              :disabled="!values[field.key]"
            )
              CheckIcon(v-if="copied === field.key")
              CopyIcon(v-else)
        FieldDescription(v-if="field.description") {{ field.description }}
        FieldDescription.text-destructive(v-if="errorFor(field.key)")
          | {{ errorFor(field.key) }}
        FieldDescription(v-else-if="field.secret && !values[field.key]")
          | No value stored. Leave empty to keep it unset.
        FieldDescription(v-else-if="field.secret")
          | Stored. Use the eye to reveal or the copy button.
  p.text-muted-foreground.text-sm(v-else-if="declaresNoValues")
    | This plugin declares no settings form.
  template(v-else)
    Field
      FieldLabel(for="plugin-settings-json") Settings
      Textarea#plugin-settings-json.font-mono.text-xs(
        v-model="rawDraft"
        placeholder="{  }"
        rows="6"
      )
      FieldDescription A JSON object; this plugin did not declare a form schema.

  p.text-destructive.text-sm(v-if="localError || error" role="alert")
    | {{ localError || error }}
  .flex.flex-col.gap-1(v-if="unmatchedErrors.length")
    p.text-destructive.text-sm(
      v-for="item in unmatchedErrors"
      :key="item.path"
    )
      | {{ item.errorText }}
  p.text-muted-foreground.text-sm(v-else-if="saved") Saved.
  .flex.justify-end(v-if="!declaresNoValues")
    Button(type="submit" :disabled="saving")
      Spinner(v-if="saving" data-icon="inline-start")
      CheckIcon(v-else data-icon="inline-start")
      span Save
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
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
import { parseJSONSchema, pluginSchemaForm } from "@/lib/plugins";
import {
  CheckIcon,
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  MinusIcon,
  PlusIcon,
} from "@lucide/vue";
import { computed, reactive, ref, watch } from "vue";
import { toast } from "vue3-toastify";

interface SchemaProperty {
  type?: string;
  title?: string;
  description?: string;
  enum?: unknown[];
  format?: string;
  secret?: boolean;
  required?: string[];
  default?: unknown;
  multiline?: boolean;
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

const parsedSchema = computed(
  () => parseJSONSchema(props.schema) as SchemaProperty | null,
);

const form = computed(() => pluginSchemaForm(props.schema));
const hasProperties = computed(() => form.value === "form");

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
    | "enum-array";
  itemKind?: ItemKind;
  options: unknown[];
  itemOptions: unknown[];
  secret: boolean;
  multiline: boolean;
  required: boolean;
}

function itemKindOf(property: SchemaProperty | undefined): ItemKind {
  if (property?.enum?.length) return "enum";
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
    const kind: FieldSpec["kind"] =
      type === "boolean"
        ? "boolean"
        : type === "number" || type === "integer"
          ? "number"
          : type === "array"
            ? property.items?.enum?.length
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
      options: property.enum ?? [],
      itemOptions: property.items?.enum ?? [],
      secret:
        secrets.has(key) ||
        property.secret === true ||
        property.format === "password",
      multiline: property.format === "textarea" || property.multiline === true,
      required: required.has(key),
    };
  });
});

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
  rawDraft.value = JSON.stringify(props.settings ?? {}, null, 2);
  localError.value = "";
}

watch(
  () => [props.settings, props.schema],
  () => reset(),
  { immediate: true, deep: true },
);

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
  emit("save", { ...values });
}
</script>
