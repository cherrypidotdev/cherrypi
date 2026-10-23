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

import { attentionPreference } from "@/composables/useAttentionPreferences";
import {
  attentionPermission,
  playAttentionSound,
  showAttentionNotification,
} from "@/lib/attention";
import { agency } from "@/lib/sdk";
import type {
  AgencyNotification,
  AttentionEvent,
  AttentionNotification,
} from "@cherrypi/sdk";
import { computed, ref } from "vue";
import { toast } from "vue3-toastify";

// Module-scoped so the sidebar's unread badge and the inbox share one value.
const notifications = ref<AgencyNotification[]>([]);
const unread = ref(0);
// Unread entries grouped by the project of their task, so the sidebar can
// badge each project with the inbox items that concern it.
const unreadByProject = computed(() => {
  const counts = new Map<string, number>();
  for (const item of notifications.value) {
    if (item.readAt || !item.projectId) continue;
    counts.set(item.projectId, (counts.get(item.projectId) ?? 0) + 1);
  }
  return counts;
});
const loading = ref(false);
const error = ref("");

async function refresh() {
  loading.value = true;
  try {
    const [list, count] = await Promise.all([
      agency.listNotifications(),
      agency.getUnreadCount(),
    ]);
    notifications.value = list;
    unread.value = count;
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function markRead(id: number) {
  await agency.markNotificationRead(id);
  await refresh();
}

async function markAllRead() {
  await agency.markAllNotificationsRead();
  await refresh();
}

// Visiting a task means its signals have been seen, so clear their unread
// flags without touching the rest of the inbox.
async function markTaskRead(taskId: string) {
  if (!taskId) return;
  try {
    await agency.markTaskNotificationsRead(taskId);
  } catch {
    // Best effort: a stale unread flag is not worth breaking navigation.
    return;
  }
  await refresh();
}

async function remove(id: number) {
  await agency.deleteNotification(id);
  await refresh();
}

async function deleteAll() {
  await agency.deleteAllNotifications();
  await refresh();
}

// -- Realtime stream --------------------------------------------------------

const BASE_RETRY_MS = 2000;
const MAX_RETRY_MS = 30000;

let socket: WebSocket | undefined;
let retryTimer: number | undefined;
let retryDelay = BASE_RETRY_MS;
let active = false;

// Show a desktop notification when the browser can, and fall back to an
// in-app floating message when it cannot (permission denied or unsupported),
// so a desktop-enabled event is never invisible.
function alertInBrowser(n: AttentionNotification) {
  const permission = attentionPermission();
  if (permission === "denied" || permission === "unsupported") {
    toast.info(n.body ? `${n.title}: ${n.body}` : n.title, { autoClose: 8000 });
    return;
  }
  showAttentionNotification(n);
}

function handleFrame(event: MessageEvent) {
  let frame: AttentionEvent;
  try {
    frame = JSON.parse(String(event.data)) as AttentionEvent;
  } catch {
    return;
  }

  if (frame.type !== "notification") {
    // "hello" just confirms the stream is live.
    void refresh();
    return;
  }

  // Each kind has its own sound/browser channels. Unknown kinds stay silent.
  const pref = attentionPreference(frame.notification.kind);
  if (pref.browser) alertInBrowser(frame.notification);
  if (pref.sound) playAttentionSound(frame.notification.kind);
  // The inbox stays the source of truth, so refetch rather than trusting the
  // pushed copy — it also picks up anything created while reconnecting.
  void refresh();
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

  let ws: WebSocket;
  try {
    ws = new WebSocket(agency.notificationWebSocketUrl());
  } catch {
    scheduleReconnect();
    return;
  }
  socket = ws;

  ws.onopen = () => {
    retryDelay = BASE_RETRY_MS;
  };
  ws.onmessage = handleFrame;
  ws.onclose = () => {
    if (socket === ws) socket = undefined;
    scheduleReconnect();
  };
  // onclose always follows onerror, so reconnection is handled there.
  ws.onerror = () => ws.close();
}

/**
 * Open the realtime attention stream and load the inbox. Idempotent: calling it
 * again while the stream is open is a no-op.
 */
function start() {
  if (active) return;
  active = true;
  retryDelay = BASE_RETRY_MS;
  void refresh();
  connect();
}

/** Close the stream and stop reconnecting. */
function stop() {
  active = false;
  if (retryTimer !== undefined) {
    window.clearTimeout(retryTimer);
    retryTimer = undefined;
  }
  if (socket) {
    socket.onopen = null;
    socket.onmessage = null;
    socket.onerror = null;
    socket.onclose = null;
    socket.close();
    socket = undefined;
  }
}

export function useNotifications() {
  return {
    notifications,
    unread,
    unreadByProject,
    loading,
    error,
    refresh,
    markRead,
    markAllRead,
    markTaskRead,
    remove,
    deleteAll,
    start,
    stop,
  };
}
