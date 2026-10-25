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
.bg-background.flex.h-full.flex-col.overflow-hidden(
  ref="rootEl"
  data-testid="desktop-view"
)
  AgentHeader(v-if="!embedded")
  header.flex.h-12.shrink-0.items-center.gap-3.border-b.px-6
    .flex.min-w-0.items-center.gap-2
      MonitorIcon.text-muted-foreground.size-4
      span.text-sm.font-semibold Desktop
      Badge(v-if="desktop" variant="outline")
        | {{ desktop.display }} · {{ desktop.width }}×{{ desktop.height }}
      Badge(v-else-if="!loading && !available" variant="destructive") Unavailable
    .ml-auto.flex.items-center.gap-2
      Badge(
        v-if="!canControl"
        variant="outline"
        title="Only agent owners can control this desktop"
      ) View only
      Badge(v-if="connected" variant="secondary") Connected
      Badge(v-else-if="available" variant="secondary") Connecting…
      Button(
        variant="outline"
        size="sm"
        :title="fullscreenHint"
        :aria-pressed="isFullscreen"
        @click="toggleFullscreen"
        :disabled="!available"
      )
        MinimizeIcon(v-if="isFullscreen")
        MaximizeIcon(v-else)
        span {{ isFullscreen ? "Exit fullscreen" : "Fullscreen" }}
      Button(
        variant="outline"
        size="sm"
        @click="reconnect"
        :disabled="!available"
      )
        RefreshCwIcon
        span Reconnect
  .relative.flex-1.overflow-hidden.bg-neutral-950(data-testid="desktop-stage")
    .h-full.w-full(ref="screenEl" v-show="available")
    .flex.h-full.flex-col.items-center.justify-center.gap-3.p-6.text-center(
      v-if="!available"
    )
      MonitorIcon.text-muted-foreground.size-8
      p.text-sm.font-medium Desktop unavailable
      p.text-muted-foreground.max-w-md.text-xs(
        v-if="unavailableReason || error"
      ) {{ unavailableReason || error }}
      p.text-muted-foreground.max-w-md.text-xs
        | CherryPi reuses the X session already running on this agent. Inside
        | Docker, mount the host's X socket and set DISPLAY and XAUTHORITY.
    .pointer-events-none.absolute.inset-x-0.top-0.flex.justify-center.p-2(
      v-else-if="!connected"
    )
      Badge(variant="secondary") Connecting…
    .pointer-events-none.absolute.inset-x-0.bottom-0.flex.justify-center.p-2(
      v-if="keyboardLocked"
    )
      Badge(variant="secondary") Keyboard captured · Esc to exit
</template>

<script setup lang="ts">
import AgentHeader from "@/components/AgentHeader.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAgentAccess } from "@/composables/useAgentAccess";
import { useDesktop } from "@/composables/useDesktop";
import {
  lockKeyboard,
  supportsKeyboardLock,
  unlockKeyboard,
} from "@/lib/keyboardLock";
import { agent, currentAgentId } from "@/lib/sdk";
import {
  MaximizeIcon,
  MinimizeIcon,
  MonitorIcon,
  RefreshCwIcon,
} from "@lucide/vue";
import RFB from "@novnc/novnc";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";

withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false });

const {
  desktop,
  wsPath,
  available,
  unavailableReason,
  loading,
  error,
  refresh,
} = useDesktop();

const { canManageAgent } = useAgentAccess();

const canControl = computed(() => canManageAgent(currentAgentId.value));

const screenEl = ref<HTMLDivElement>();
const rootEl = ref<HTMLDivElement>();
const connected = ref(false);
const isFullscreen = ref(false);
const keyboardLocked = ref(false);
let rfb: RFB | null = null;

const fullscreenHint = supportsKeyboardLock()
  ? "Enter fullscreen and send every shortcut (Ctrl+W, Ctrl+T, …) to the desktop. Esc exits."
  : "Enter fullscreen. This browser cannot capture shortcuts it reserves for itself.";

function disconnect() {
  if (rfb) {
    rfb.disconnect();
    rfb = null;
  }
  connected.value = false;
}

function connect() {
  disconnect();
  if (!available.value || !wsPath.value || !screenEl.value) {
    return;
  }
  rfb = new RFB(screenEl.value, agent.desktopWebSocketUrl(), {
    credentials: { password: "" },
  });
  rfb.viewOnly = !canControl.value;
  rfb.scaleViewport = true;
  rfb.resizeSession = false;
  rfb.background = "#0a0a0a";
  rfb.addEventListener("connect", () => {
    connected.value = true;

    rfb?.focus({ preventScroll: true });
  });
  rfb.addEventListener("disconnect", () => {
    connected.value = false;
  });
  rfb.addEventListener("credentialsrequired", () => {
    rfb?.sendCredentials({ password: "" });
  });
}

async function reconnect() {
  disconnect();
  await refresh();
  await nextTick();
  connect();
}

function applyKeyboardLock(): void {
  if (!isFullscreen.value || !canControl.value) {
    releaseKeyboardLock();
    return;
  }
  if (keyboardLocked.value) {
    return;
  }
  void lockKeyboard().then((locked) => {
    keyboardLocked.value = locked;
  });
}

function releaseKeyboardLock(): void {
  if (!keyboardLocked.value) {
    return;
  }
  unlockKeyboard();
  keyboardLocked.value = false;
}

function onFullscreenChange(): void {
  isFullscreen.value =
    !!rootEl.value && document.fullscreenElement === rootEl.value;
  if (isFullscreen.value) {
    applyKeyboardLock();

    rfb?.focus({ preventScroll: true });
  } else {
    releaseKeyboardLock();
  }
}

async function toggleFullscreen(): Promise<void> {
  if (!available.value || !rootEl.value) {
    return;
  }
  try {
    if (document.fullscreenElement === rootEl.value) {
      await document.exitFullscreen();
    } else {
      await rootEl.value.requestFullscreen();
    }
  } catch {}
}

onMounted(async () => {
  document.addEventListener("fullscreenchange", onFullscreenChange);
  await refresh();
});

onBeforeUnmount(() => {
  document.removeEventListener("fullscreenchange", onFullscreenChange);
  releaseKeyboardLock();
  if (rootEl.value && document.fullscreenElement === rootEl.value) {
    void document.exitFullscreen().catch(() => {});
  }
  disconnect();
});

watch([available, wsPath], async () => {
  await nextTick();
  connect();
});

watch(canControl, (control) => {
  if (rfb) rfb.viewOnly = !control;
  applyKeyboardLock();
});
</script>
