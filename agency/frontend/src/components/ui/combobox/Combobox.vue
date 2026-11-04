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

<script setup lang="ts">
import { cn } from "@/lib/utils";
import { CheckIcon, ChevronsUpDownIcon } from "@lucide/vue";
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxLabel,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxTrigger,
  ComboboxViewport,
} from "reka-ui";
import { computed, useAttrs, type HTMLAttributes } from "vue";
import type { ComboboxOption } from "./types";

defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    modelValue?: string;
    options: ComboboxOption[];

    placeholder?: string;

    emptyText?: string;
    disabled?: boolean;

    size?: "sm" | "default";
    class?: HTMLAttributes["class"];
  }>(),
  {
    modelValue: "",
    placeholder: "",
    emptyText: "No matches.",
    disabled: false,
    size: "default",
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const attrs = useAttrs();

const optionsKey = computed(() =>
  props.options
    .map((option) => `${option.value}\u0000${option.label}`)
    .join("\u0001"),
);

const groups = computed(() => {
  const order: string[] = [];
  const byGroup = new Map<string, ComboboxOption[]>();
  for (const option of props.options) {
    const key = option.group ?? "";
    if (!byGroup.has(key)) {
      byGroup.set(key, []);
      order.push(key);
    }
    byGroup.get(key)!.push(option);
  }
  return order.map((key) => ({ label: key, options: byGroup.get(key)! }));
});

function displayValue(value: string): string {
  return props.options.find((option) => option.value === value)?.label ?? "";
}

function onUpdate(value: unknown): void {
  emit("update:modelValue", typeof value === "string" ? value : "");
}

function searchText(option: ComboboxOption): string {
  return [option.label, option.description ?? "", option.value]
    .filter(Boolean)
    .join(" ");
}
</script>

<template>
  <ComboboxRoot
    :key="optionsKey"
    :model-value="modelValue"
    :disabled="disabled"
    :open-on-click="true"
    reset-search-term-on-select
    class="relative"
    @update:model-value="onUpdate"
  >
    <ComboboxAnchor class="relative block">
      <ComboboxInput
        v-bind="attrs"
        :display-value="displayValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :class="
          cn(
            'dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 disabled:bg-input/50 dark:disabled:bg-input/80 h-8 w-full min-w-0 rounded-lg border bg-transparent py-1 pr-8 pl-2.5 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:ring-3 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
            size === 'sm' && 'h-7 rounded-[min(var(--radius-md),10px)]',
            props.class,
          )
        "
      />
      <ComboboxTrigger
        type="button"
        :disabled="disabled"
        class="text-muted-foreground absolute top-1/2 right-2 flex -translate-y-1/2 items-center justify-center outline-none disabled:pointer-events-none disabled:opacity-50"
        aria-label="Toggle options"
      >
        <ChevronsUpDownIcon class="size-4 shrink-0 opacity-60" />
      </ComboboxTrigger>
    </ComboboxAnchor>

    <ComboboxPortal>
      <ComboboxContent
        position="popper"
        :side-offset="4"
        align="start"
        class="bg-popover text-popover-foreground ring-foreground/10 z-50 max-h-72 w-(--reka-combobox-trigger-width) min-w-48 overflow-hidden rounded-lg shadow-md ring-1"
      >
        <ComboboxViewport class="max-h-72 overflow-y-auto p-1">
          <ComboboxEmpty class="text-muted-foreground py-6 text-center text-sm">
            {{ emptyText }}
          </ComboboxEmpty>
          <ComboboxGroup
            v-for="group in groups"
            :key="group.label"
            class="overflow-hidden"
          >
            <ComboboxLabel
              v-if="group.label"
              class="text-muted-foreground px-2 py-1.5 text-xs font-medium"
            >
              {{ group.label }}
            </ComboboxLabel>
            <ComboboxItem
              v-for="option in group.options"
              :key="option.value"
              :value="option.value"
              :text-value="searchText(option)"
              class="focus:bg-accent focus:text-accent-foreground relative flex w-full cursor-default items-center gap-1.5 rounded-md py-1 pr-8 pl-1.5 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
            >
              <slot name="item" :option="option">
                <span class="flex min-w-0 flex-col">
                  <span class="truncate">{{ option.label }}</span>
                  <span
                    v-if="option.description"
                    class="text-muted-foreground truncate text-xs"
                  >
                    {{ option.description }}
                  </span>
                </span>
              </slot>
              <span
                class="pointer-events-none absolute right-2 flex size-4 items-center justify-center"
              >
                <ComboboxItemIndicator>
                  <CheckIcon class="size-4" />
                </ComboboxItemIndicator>
              </span>
            </ComboboxItem>
          </ComboboxGroup>
        </ComboboxViewport>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>
