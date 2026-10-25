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
.flex.h-full.min-h-0.flex-col(ref="workspaceRoot")
  .flex.h-11.shrink-0.items-center.gap-2.border-b.px-2
    .flex.min-w-0.flex-1.items-center.gap-1.overflow-x-auto
      .flex.shrink-0.items-center.rounded-md.transition-colors(
        v-for="(terminal, index) in terminals"
        :key="terminal.id"
        :class="activeId === terminal.id ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent/60'"
      )
        button.flex.max-w-40.items-center.gap-1.px-2.py-1.text-xs.font-medium(
          type="button"
          @click="selectTab(terminal.id)"
        )
          TerminalIcon.shrink-0(class="size-3.5")
          span.truncate Terminal {{ index + 1 }}
        button.flex.shrink-0.items-center.px-1.py-1(
          type="button"
          title="Close tab"
          @click.stop="closeTab(terminal.id)"
        )
          XIcon.size-3.shrink-0
      Button.shrink-0(
        variant="ghost"
        size="sm"
        title="New terminal tab"
        @click="addTab"
      )
        PlusIcon(data-icon="inline-start")
        span New terminal tab
    .flex.shrink-0.items-center.gap-1
      Button(
        variant="ghost"
        size="icon-sm"
        title="Keyboard shortcuts"
        @click="showShortcuts = true"
      )
        KeyboardIcon
      span.text-muted-foreground.text-xs.whitespace-nowrap
        | {{ terminalLabel }}
      Button(
        variant="outline"
        size="sm"
        :title="closeAllTitle"
        @click="closeAll"
        :disabled="terminalCount === 0"
      )
        Trash2Icon(data-icon="inline-start")
        span Close all
  .relative.min-h-0.flex-1
    template(v-if="ready")
      .absolute.inset-0(
        v-for="terminal in terminals"
        :key="terminal.id"
        v-show="activeId === terminal.id"
      )
        TerminalPane(
          :terminal-id="terminal.id"
          :active="activeId === terminal.id"
          @exit="onExit"
          @lost="onLost"
        )
      .text-muted-foreground.flex.h-full.flex-col.items-center.justify-center.gap-2.text-center.text-sm(
        v-if="!terminals.length"
      )
        span No terminal is running.
        span.text-xs
          | Press “New terminal tab” to start one.
    .flex.h-full.items-center.justify-center(v-else)
      Spinner.text-muted-foreground.size-5
  TerminalShortcutDialog(v-model:open="showShortcuts")
</template>

<script setup lang="ts">
import TerminalPane from "@/components/terminals/TerminalPane.vue";
import TerminalShortcutDialog from "@/components/terminals/TerminalShortcutDialog.vue";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useRealtime } from "@/composables/useRealtime";
import { agent } from "@/lib/sdk";
import { matchShortcut } from "@/lib/terminalShortcuts";
import { TerminalKind, type Terminal } from "@cherrypi/sdk";
import {
  KeyboardIcon,
  PlusIcon,
  TerminalIcon,
  Trash2Icon,
  XIcon,
} from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const surface = "root";
const closeAllTitle = "Close all of your root terminals";

const terminals = ref<Terminal[]>([]);

const activeId = ref<string | null>(null);
const ready = ref(false);
const workspaceRoot = ref<HTMLElement | null>(null);
const showShortcuts = ref(false);

const terminalCount = computed(() => terminals.value.length);
const terminalLabel = computed(
  () =>
    `${terminalCount.value} root terminal${
      terminalCount.value === 1 ? "" : "s"
    }`,
);

function activeKey(): string {
  return `cherrypi:terminals:active:${surface}`;
}

function loadActive(): string | null {
  try {
    return sessionStorage.getItem(activeKey());
  } catch {
    return null;
  }
}

function saveActive(id: string | null) {
  try {
    if (id) {
      sessionStorage.setItem(activeKey(), id);
    } else {
      sessionStorage.removeItem(activeKey());
    }
  } catch {}
}

watch(activeId, (id) => saveActive(id));

let refreshing = false;
async function refresh() {
  if (refreshing) {
    return;
  }
  refreshing = true;
  let list: Terminal[];
  try {
    list = await agent.listTerminals();
  } catch {
    return;
  } finally {
    refreshing = false;
  }
  terminals.value = list;

  if (!list.some((terminal) => terminal.id === activeId.value)) {
    activeId.value = list[0]?.id ?? null;
  }
}

async function addTab() {
  try {
    const { terminal } = await agent.createTerminal(TerminalKind.ROOT, {
      cols: 80,
      rows: 24,
    });
    await refresh();
    activeId.value = terminal.id;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function closeTab(id: string) {
  try {
    await agent.closeTerminal(id);
  } catch {}
  terminals.value = terminals.value.filter((terminal) => terminal.id !== id);
  if (activeId.value === id) {
    activeId.value = terminals.value[0]?.id ?? null;
  }
  void refresh();
}

async function closeAll() {
  if (terminalCount.value === 0) {
    return;
  }
  if (
    !window.confirm("Close all root terminals? Running shells will be killed.")
  ) {
    return;
  }
  try {
    await agent.closeAllTerminals({ kind: TerminalKind.ROOT });
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
    return;
  }
  terminals.value = [];
  activeId.value = null;
  void refresh();
}

function selectTab(id: string) {
  activeId.value = id;
}

function onExit(id: string) {
  terminals.value = terminals.value.filter((terminal) => terminal.id !== id);
  if (activeId.value === id) {
    activeId.value = terminals.value[0]?.id ?? null;
  }
  void refresh();
}

function onLost() {
  void refresh();
}

function selectRelativeTab(delta: number): boolean {
  if (terminals.value.length === 0) {
    return false;
  }
  const current = terminals.value.findIndex(
    (terminal) => terminal.id === activeId.value,
  );
  const base = current === -1 ? (delta > 0 ? -1 : 0) : current;
  const next =
    terminals.value[
      (base + delta + terminals.value.length) % terminals.value.length
    ];
  if (!next) {
    return false;
  }
  activeId.value = next.id;
  return true;
}

function runShortcut(id: string): boolean {
  if (id.startsWith("tab-")) {
    const terminal = terminals.value[Number(id.slice(4)) - 1];
    if (!terminal) {
      return false;
    }
    activeId.value = terminal.id;
    return true;
  }
  switch (id) {
    case "new-tab":
      void addTab();
      return true;
    case "close-tab":
      if (!activeId.value) {
        return false;
      }
      void closeTab(activeId.value);
      return true;
    case "next-tab":
      return selectRelativeTab(1);
    case "prev-tab":
      return selectRelativeTab(-1);
    case "help":
      showShortcuts.value = true;
      return true;
    default:
      return false;
  }
}

function onKeydownCapture(event: KeyboardEvent): void {
  const active = document.activeElement;
  if (
    active &&
    active !== document.body &&
    workspaceRoot.value &&
    !workspaceRoot.value.contains(active)
  ) {
    return;
  }
  const id = matchShortcut(event);
  if (!id || !runShortcut(id)) {
    return;
  }
  event.preventDefault();
  event.stopPropagation();
}

onMounted(() => {
  window.addEventListener("keydown", onKeydownCapture, true);
  activeId.value = loadActive();
  void (async () => {
    await refresh();
    ready.value = true;
  })();
});

useRealtime(["terminals"], () => void refresh());

onBeforeUnmount(() => {
  window.removeEventListener("keydown", onKeydownCapture, true);
});
</script>
