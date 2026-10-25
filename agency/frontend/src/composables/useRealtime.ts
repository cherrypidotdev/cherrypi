// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { agency } from "@/lib/sdk";
import type { RealtimeKind } from "@cherrypi/sdk";
import { onBeforeUnmount } from "vue";

interface Listener {
  kinds: Set<RealtimeKind>;
  run: () => void;
  timer: number | undefined;
}

const DEBOUNCE_MS = 120;
const BASE_RETRY_MS = 1000;
const MAX_RETRY_MS = 30_000;

const listeners = new Set<Listener>();
let active = false;
let controller: AbortController | undefined;
let retryTimer: number | undefined;
let retryDelay = BASE_RETRY_MS;

function notify(kind: RealtimeKind) {
  for (const listener of listeners) {
    if (kind !== "sync" && !listener.kinds.has(kind)) continue;
    schedule(listener);
  }
}

function schedule(listener: Listener) {
  if (listener.timer !== undefined) return;
  listener.timer = window.setTimeout(() => {
    listener.timer = undefined;
    listener.run();
  }, DEBOUNCE_MS);
}

function scheduleReconnect() {
  if (!active || retryTimer !== undefined) return;
  retryTimer = window.setTimeout(() => {
    retryTimer = undefined;
    connect();
  }, retryDelay);
  retryDelay = Math.min(retryDelay * 2, MAX_RETRY_MS);
}

function connect() {
  if (!active) return;
  controller?.abort();
  const stream = new AbortController();
  controller = stream;

  void agency
    .watchRealtime(
      (event) => {
        retryDelay = BASE_RETRY_MS;
        notify(event.kind);
      },
      { signal: stream.signal },
    )
    .catch((err) => {
      if (stream.signal.aborted) return;
      console.warn("[realtime] stream interrupted", err);
      scheduleReconnect();
    })
    .finally(() => {
      if (active && controller === stream && !stream.signal.aborted) {
        scheduleReconnect();
      }
    });
}

export function startRealtime() {
  if (active) return;
  active = true;
  retryDelay = BASE_RETRY_MS;
  connect();
}

export function stopRealtime() {
  active = false;
  if (retryTimer !== undefined) {
    window.clearTimeout(retryTimer);
    retryTimer = undefined;
  }
  controller?.abort();
  controller = undefined;
}

export function useRealtime(kinds: RealtimeKind[], run: () => void) {
  const listener: Listener = {
    kinds: new Set(kinds),
    run,
    timer: undefined,
  };
  listeners.add(listener);

  onBeforeUnmount(() => {
    listeners.delete(listener);
    if (listener.timer !== undefined) {
      window.clearTimeout(listener.timer);
    }
  });
}
