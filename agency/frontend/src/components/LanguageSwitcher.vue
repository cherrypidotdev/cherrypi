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
DropdownMenu
  DropdownMenuTrigger(as-child)
    Button(
      type="button"
      variant="ghost"
      size="icon-sm"
      :title="$t('language.switch')"
      :aria-label="$t('language.switch')"
      data-test="language-switcher"
    )
      LanguagesIcon
  DropdownMenuContent.min-w-40(align="end")
    DropdownMenuLabel {{ $t("language.label") }}
    DropdownMenuRadioGroup(
      :model-value="locale"
      @update:model-value="change($event)"
    )
      DropdownMenuRadioItem(
        v-for="option in options"
        :key="option.value"
        :value="option.value"
        :data-test="`language-option-${option.value}`"
      ) {{ option.label }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLocale } from "@/composables/useLocale";
import { LOCALE_NAMES, LOCALES, type Locale } from "@/i18n";
import { LanguagesIcon } from "@lucide/vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();
const { locale, setLocale } = useLocale();

const options = computed(() =>
  LOCALES.map((value) => ({ value, label: LOCALE_NAMES[value] })),
);

function change(value: unknown): void {
  if (typeof value !== "string") return;

  setLocale(value as Locale).catch(() => {
    toast.error(t("language.saveFailed"));
  });
}
</script>

<i18n lang="yaml">
en:
  language:
    label: "Language"
    switch: "Change language"
    saveFailed: "Could not save the language; reverted."
ru:
  language:
    label: "Язык"
    switch: "Сменить язык"
    saveFailed: "Не удалось сохранить язык; изменение отменено."
</i18n>
