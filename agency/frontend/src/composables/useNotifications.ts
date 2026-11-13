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
import { navigateAttention, showAttentionToast } from "@/lib/attentionToast";
import {
  groupRoute,
  loadPromptRouteTargets,
  promptRoute,
} from "@/lib/notificationLink";
import { agency } from "@/lib/sdk";
import type {
  AgencyNotification,
  AttentionEvent,
  AttentionNotification,
} from "@cherrypi/sdk";
import { computed, ref } from "vue";

const notifications = ref<AgencyNotification[]>([]);
const unread = ref(0);

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

async function markTaskRead(taskId: string) {
  if (!taskId) return;
  try {
    await agency.markTaskNotificationsRead(taskId);
  } catch {
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

const BASE_RETRY_MS = 2000;
const MAX_RETRY_MS = 30000;

let socket: WebSocket | undefined;
let retryTimer: number | undefined;
let retryDelay = BASE_RETRY_MS;
let active = false;

function desktopUnavailable(): boolean {
  const permission = attentionPermission();
  return permission === "denied" || permission === "unsupported";
}

function isTaskOnScreen(taskId: string): boolean {
  if (!taskId) return false;
  const path = window.location.pathname.replace(/\/$/, "");
  return path === `/tasks/${taskId}`;
}

function openFromToast(n: AttentionNotification) {
  if (n.id > 0) void markRead(n.id);
  if (n.taskId) {
    navigateAttention({
      name: "task",
      params: { id: n.taskId },

      query: n.kind === "task_mention" ? { tab: "comments" } : undefined,
    });
  } else if (n.kind === "team_invitation") {
    navigateAttention({ name: "teams" });
  } else if (n.kind.startsWith("agent_group_member")) {
    const route = groupRoute(n, "members");
    if (route) navigateAttention(route);
  } else {
    const route = promptRoute(n);
    if (route) navigateAttention(route);
  }
}

function handleFrame(event: MessageEvent) {
  let frame: AttentionEvent;
  try {
    frame = JSON.parse(String(event.data)) as AttentionEvent;
  } catch {
    return;
  }

  if (frame.type !== "notification") {
    void refresh();
    return;
  }

  const notification = frame.notification;
  const pref = attentionPreference(notification.kind);

  const blocked = desktopUnavailable();
  const wantsToast = pref.toast || (pref.browser && blocked);
  if (wantsToast && !isTaskOnScreen(notification.taskId)) {
    showAttentionToast(notification, () => openFromToast(notification));
  }
  if (pref.browser && !blocked) showAttentionNotification(notification);
  if (pref.sound) playAttentionSound(notification.kind);

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

  ws.onerror = () => ws.close();
}

function start() {
  if (active) return;
  active = true;
  retryDelay = BASE_RETRY_MS;
  void refresh();

  void loadPromptRouteTargets();
  connect();
}

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
