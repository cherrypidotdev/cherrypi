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

import { soundSet, soundVolume } from "@/composables/useSoundPreferences";
import { plainNotificationText } from "@/lib/notificationText";
import { closeContext, ensureReady } from "@/lib/sounds/context";
import { playSound, stopSound } from "@/lib/sounds/play";
import type { AttentionNotification } from "@cherrypi/sdk";

export const DEFAULT_FAVICON = "/favicon.png";
const DEFAULT_SHORTCUT_ICON = "/favicon.ico";
const DEFAULT_APPLE_ICON = "/apple-touch-icon.png";

let currentIcon = DEFAULT_FAVICON;

let faviconImage: HTMLImageElement | null = null;
let faviconImageSrc = "";
let faviconImageReady = false;

let currentCount = 0;

function setIconLink(rel: string, href: string) {
  let links = Array.from(
    document.querySelectorAll<HTMLLinkElement>(`link[rel="${rel}"]`),
  );
  if (links.length === 0) {
    const link = document.createElement("link");
    link.rel = rel;
    document.head.appendChild(link);
    links = [link];
  }
  for (const link of links) link.href = href;
}

function badgeDataUrl(): string | null {
  if (!faviconImage) return null;

  const canvas = document.createElement("canvas");
  canvas.width = 64;
  canvas.height = 64;

  const ctx = canvas.getContext("2d");
  if (!ctx) return null;

  ctx.drawImage(faviconImage, 0, 0, 64, 64);

  const centerX = 46;
  const centerY = 18;
  const radius = 19;
  ctx.beginPath();
  ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
  ctx.fillStyle = "#ef4444";
  ctx.fill();

  ctx.fillStyle = "#ffffff";
  ctx.font = "bold 26px sans-serif";
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(
    currentCount > 99 ? "99+" : String(currentCount),
    centerX,
    centerY + 1,
  );

  return canvas.toDataURL("image/png");
}

export function applyFavicon(count: number, iconUrl?: string) {
  const configured = iconUrl?.trim() ?? "";
  const src = configured || DEFAULT_FAVICON;
  currentIcon = src;
  currentCount = count;

  setIconLink("apple-touch-icon", configured || DEFAULT_APPLE_ICON);

  if (src !== faviconImageSrc) {
    faviconImage = null;
    faviconImageReady = false;
    faviconImageSrc = src;
  }

  const showPlain = () => {
    setIconLink("icon", src);
    setIconLink("shortcut icon", configured || DEFAULT_SHORTCUT_ICON);
  };

  if (count <= 0) {
    showPlain();
    return;
  }

  const showBadge = () => {
    const dataUrl = badgeDataUrl();
    if (!dataUrl) return;
    setIconLink("icon", dataUrl);
    setIconLink("shortcut icon", dataUrl);
  };

  if (!faviconImage) {
    const image = new Image();
    image.onload = () => {
      if (faviconImageSrc !== src) return;
      faviconImage = image;
      faviconImageReady = true;

      if (currentCount <= 0) return;
      showBadge();
    };
    image.src = src;
    return;
  }

  if (faviconImageReady) {
    showBadge();
  }
}

let audioUnlockHandler: (() => void) | null = null;

export function playAttentionSound(kind = "") {
  void playSound(soundSet.value, kind, soundVolume.value);
}

export function armAttentionAudio() {
  if (audioUnlockHandler) return;
  if (typeof window === "undefined" || !window.AudioContext) return;

  const handler = () => {
    window.removeEventListener("pointerdown", handler, true);
    window.removeEventListener("keydown", handler, true);
    audioUnlockHandler = null;
    void ensureReady().catch(() => {});
  };

  audioUnlockHandler = handler;
  window.addEventListener("pointerdown", handler, true);
  window.addEventListener("keydown", handler, true);
}

const liveNotifications = new Set<Notification>();

let pendingNotifications: AttentionNotification[] = [];
let permissionHandler: ((event: Event) => void) | null = null;

function display(n: AttentionNotification) {
  try {
    const notification = new Notification(n.title, {
      body: plainNotificationText(n.body),
      icon: currentIcon,

      tag: `cherrypi-${n.id}-${n.createdAt}`,

      silent: true,
    });
    liveNotifications.add(notification);
    notification.onclose = () => liveNotifications.delete(notification);
  } catch {}
}

function armPermissionRequest() {
  if (permissionHandler) return;

  const handler = async () => {
    window.removeEventListener("pointerdown", handler, true);
    window.removeEventListener("keydown", handler, true);
    permissionHandler = null;

    const permission = await requestAttentionPermission();
    if (permission !== "granted") {
      pendingNotifications = [];
      return;
    }
    const queued = pendingNotifications;
    pendingNotifications = [];
    for (const item of queued) display(item);
  };

  permissionHandler = handler;
  window.addEventListener("pointerdown", handler, true);
  window.addEventListener("keydown", handler, true);
}

export function resetAttention() {
  if (permissionHandler) {
    window.removeEventListener("pointerdown", permissionHandler, true);
    window.removeEventListener("keydown", permissionHandler, true);
    permissionHandler = null;
  }
  pendingNotifications = [];
  liveNotifications.clear();
  if (audioUnlockHandler) {
    window.removeEventListener("pointerdown", audioUnlockHandler, true);
    window.removeEventListener("keydown", audioUnlockHandler, true);
    audioUnlockHandler = null;
  }
  stopSound();
  closeContext();
  faviconImage = null;
  faviconImageReady = false;
  faviconImageSrc = "";
  currentIcon = DEFAULT_FAVICON;
  currentCount = 0;
}

export function armAttentionPermission() {
  if (typeof Notification === "undefined") return;
  if (Notification.permission !== "default") return;
  armPermissionRequest();
}

export function showAttentionNotification(n: AttentionNotification) {
  if (typeof Notification === "undefined") return;

  if (Notification.permission === "granted") {
    display(n);
    return;
  }
  if (Notification.permission === "default") {
    pendingNotifications.push(n);
    armPermissionRequest();
  }
}

export function attentionPermission(): NotificationPermission | "unsupported" {
  if (typeof Notification === "undefined") return "unsupported";
  return Notification.permission;
}

export async function requestAttentionPermission(): Promise<
  NotificationPermission | "unsupported"
> {
  if (typeof Notification === "undefined") return "unsupported";
  try {
    return await Notification.requestPermission();
  } catch {
    return Notification.permission;
  }
}
