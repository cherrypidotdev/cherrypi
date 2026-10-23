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
Dialog(:open="open" @update:open="emit('update:open', $event)")
  DialogContent
    DialogHeader
      DialogTitle Terminal shortcuts
      DialogDescription
        | Available while a terminal is focused. Some Ctrl+Shift combos are
        | reserved by the browser; the Alt+Shift equivalents always work.
    dl.flex.flex-col.gap-2
      .flex.items-center.justify-between.gap-6(
        v-for="shortcut in shortcuts"
        :key="shortcut.id"
      )
        dt.text-sm {{ shortcut.label }}
        dd.shrink-0
          kbd.bg-muted.text-muted-foreground.rounded.border.font-mono.text-xs.whitespace-nowrap(
            class="px-1.5 py-0.5"
          ) {{ shortcut.keys }}
</template>

<script setup lang="ts">
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ALL_TERMINAL_SHORTCUTS } from "@/lib/terminalShortcuts";

defineProps<{ open: boolean }>();

const emit = defineEmits<{ "update:open": [value: boolean] }>();

const shortcuts = ALL_TERMINAL_SHORTCUTS;
</script>
