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
.flex.h-full.w-full.flex-col.bg-black(
  :class="active ? 'ring-primary ring-1 ring-inset' : ''"
)
  .bg-muted.flex.shrink-0.items-center.justify-between.gap-2.px-2.py-1.text-xs(
    v-if="controlKnown && !controller && !exited"
  )
    span.text-muted-foreground.truncate {{ banner }}
    Button.shrink-0(variant="outline" size="sm" @click="takeControl") Take control
  .relative.min-h-0.flex-1.duration-150(
    class="transition-[filter]"
    :class="controlKnown && !controller ? 'brightness-[0.8] hover:brightness-100' : ''"
  )
    .h-full.w-full.p-1(ref="host")
    p.text-destructive-foreground.absolute.inset-x-0.bottom-0.px-2.py-1.text-xs(
      v-if="error"
      class="bg-destructive/80"
    ) {{ error }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { agent } from "@/lib/sdk";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal as Xterm } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";

const props = defineProps<{
  terminalId: string;
  active?: boolean;
}>();

const emit = defineEmits<{
  exit: [terminalId: string];

  lost: [terminalId: string];
}>();

const host = ref<HTMLDivElement | null>(null);
const error = ref("");

const controlKnown = ref(false);

const controller = ref(false);

const takenOver = ref(false);

const banner = computed(() =>
  takenOver.value
    ? "Another browser took control of this terminal."
    : "Read-only — this terminal is controlled in another browser.",
);

let term: Xterm | undefined;
let fit: FitAddon | undefined;
let socket: WebSocket | undefined;
let observer: ResizeObserver | undefined;

const WRITE_FLUSH_MS = 16;
const WRITE_FLUSH_BYTES = 512 * 1024;
let writeQueue: Uint8Array[] = [];
let writeBytes = 0;
let writeTimer: number | undefined;

let exited = false;

let leaving = false;

let disposed = false;

let retries = 0;
let reconnectTimer: number | undefined;

interface ControlMessage {
  type?: string;
  controller?: boolean;
  cols?: number;
  rows?: number;
}

function resetWrites() {
  if (writeTimer !== undefined) {
    window.clearTimeout(writeTimer);
    writeTimer = undefined;
  }
  writeQueue = [];
  writeBytes = 0;
}

function flushWrites() {
  if (writeTimer !== undefined) {
    window.clearTimeout(writeTimer);
    writeTimer = undefined;
  }
  if (disposed || !writeQueue.length) {
    resetWrites();
    return;
  }
  const merged = new Uint8Array(writeBytes);
  let offset = 0;
  for (const chunk of writeQueue) {
    merged.set(chunk, offset);
    offset += chunk.length;
  }
  writeQueue = [];
  writeBytes = 0;
  term?.write(merged);
}

function queueWrite(data: Uint8Array) {
  writeQueue.push(data);
  writeBytes += data.length;
  if (writeBytes >= WRITE_FLUSH_BYTES) {
    flushWrites();
    return;
  }
  if (writeTimer === undefined) {
    writeTimer = window.setTimeout(flushWrites, WRITE_FLUSH_MS);
  }
}

function send(message: unknown) {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(message));
  }
}

function takeControl() {
  send({ type: "take-control" });
}

function applyControl(msg: ControlMessage) {
  const nowController = Boolean(msg.controller);
  if (!nowController && controller.value) {
    takenOver.value = true;
  }
  if (nowController) {
    takenOver.value = false;
  }
  controller.value = nowController;
  controlKnown.value = true;
  if (term) {
    term.options.disableStdin = !nowController;
  }
  if (nowController) {
    void nextTick(() => {
      if (host.value && host.value.clientWidth > 0) {
        fit?.fit();
      }
      if (props.active) {
        term?.focus();
      }
    });
  } else if (msg.cols && msg.rows) {
    term?.resize(msg.cols, msg.rows);
  }
}

function onPageHide() {
  leaving = true;
}

function connect() {
  const ws = new WebSocket(agent.terminalWebSocketUrl(props.terminalId));
  ws.binaryType = "arraybuffer";
  socket = ws;

  ws.onopen = () => {
    retries = 0;
    error.value = "";

    resetWrites();
    term?.reset();
  };
  ws.onmessage = (event: MessageEvent) => {
    if (disposed) {
      return;
    }

    if (typeof event.data === "string") {
      let msg: ControlMessage | null = null;
      try {
        msg = JSON.parse(event.data) as ControlMessage;
      } catch {
        msg = null;
      }
      switch (msg?.type) {
        case "control":
          applyControl(msg);
          return;
        case "exit":
          exited = true;
          emit("exit", props.terminalId);
          return;
        default:
          queueWrite(new TextEncoder().encode(event.data));
          return;
      }
    }
    queueWrite(new Uint8Array(event.data as ArrayBuffer));
  };
  ws.onerror = () => {
    if (!leaving && !disposed && !exited) {
      error.value = "terminal connection failed";
    }
  };
  ws.onclose = () => {
    if (leaving || disposed || exited) {
      return;
    }
    error.value = "connection lost — reconnecting…";
    emit("lost", props.terminalId);
    retries += 1;
    const delay = Math.min(1000 * 2 ** (retries - 1), 5000);
    reconnectTimer = window.setTimeout(() => {
      if (!leaving && !disposed && !exited) {
        connect();
      }
    }, delay);
  };
}

onMounted(async () => {
  const el = host.value;
  if (!el) return;

  term = new Xterm({
    convertEol: true,
    cursorBlink: true,
    disableStdin: true,
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
    fontSize: 13,
    theme: { background: "#000000" },
  });
  fit = new FitAddon();
  term.loadAddon(fit);
  term.open(el);
  await nextTick();

  term.onData((data) => send({ type: "input", data }));
  term.onResize(({ cols, rows }) => {
    if (controller.value) {
      send({ type: "resize", cols, rows });
    }
  });

  connect();

  observer = new ResizeObserver(() => {
    if (!controller.value) {
      return;
    }
    if (el.clientWidth > 0 && el.clientHeight > 0) {
      fit?.fit();
    }
  });
  observer.observe(el);

  window.addEventListener("pagehide", onPageHide);
  window.addEventListener("beforeunload", onPageHide);
});

watch(
  () => props.active,
  (active) => {
    if (active && controller.value) {
      void nextTick(() => {
        if (host.value && host.value.clientWidth > 0) {
          fit?.fit();
        }
        term?.focus();
      });
    }
  },
);

onBeforeUnmount(() => {
  disposed = true;
  leaving = true;
  resetWrites();
  if (reconnectTimer !== undefined) {
    window.clearTimeout(reconnectTimer);
  }
  window.removeEventListener("pagehide", onPageHide);
  window.removeEventListener("beforeunload", onPageHide);
  observer?.disconnect();
  socket?.close();
  term?.dispose();
});
</script>
