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
import type { RealtimeEvent, RealtimeKind } from "@cherrypi/sdk";
import { onBeforeUnmount } from "vue";

interface Listener {
  kinds: Set<RealtimeKind>;
  run: (event: RealtimeEvent) => void;

  rows: boolean;
  timer: number | undefined;

  pending: Map<string, RealtimeEvent>;
}

const DEBOUNCE_MS = 120;

const MAX_PUSHED_ROWS = 8;
const BASE_RETRY_MS = 1000;
const MAX_RETRY_MS = 30_000;

const listeners = new Set<Listener>();
let active = false;
let controller: AbortController | undefined;
let retryTimer: number | undefined;
let retryDelay = BASE_RETRY_MS;

function notify(event: RealtimeEvent) {
  for (const listener of listeners) {
    if (event.kind !== "sync" && !listener.kinds.has(event.kind)) continue;
    schedule(listener, event);
  }
}

function schedule(listener: Listener, event: RealtimeEvent) {
  listener.pending.set(signalKey(event), event);
  if (listener.timer !== undefined) return;
  listener.timer = window.setTimeout(() => flush(listener), DEBOUNCE_MS);
}

function flush(listener: Listener) {
  listener.timer = undefined;
  const burst = [...listener.pending.values()];
  listener.pending.clear();

  const byKind = new Map<RealtimeKind, RealtimeEvent[]>();
  for (const event of burst) {
    const events = byKind.get(event.kind);
    if (events) events.push(event);
    else byKind.set(event.kind, [event]);
  }

  for (const [kind, events] of byKind) {
    const pushable =
      listener.rows &&
      events.length <= MAX_PUSHED_ROWS &&
      events.every((event) => event.task);
    if (!pushable) {
      listener.run({ kind });
      continue;
    }
    for (const event of events) listener.run(event);
  }
}

function signalKey(event: RealtimeEvent): string {
  return `${event.kind}\u0000${event.task?.id ?? ""}`;
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
        notify(event);
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

export interface RealtimeOptions {
  rows?: boolean;
}

export function useRealtime(
  kinds: RealtimeKind[],
  run: (event: RealtimeEvent) => void,
  options: RealtimeOptions = {},
) {
  const listener: Listener = {
    kinds: new Set(kinds),
    run,
    rows: options.rows ?? false,
    timer: undefined,
    pending: new Map(),
  };
  listeners.add(listener);

  onBeforeUnmount(() => {
    listeners.delete(listener);
    listener.pending.clear();
    if (listener.timer !== undefined) {
      window.clearTimeout(listener.timer);
    }
  });
}
