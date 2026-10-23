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

// Browser attention affordances: the tab title badge, the favicon badge, a
// desktop notification, and a short chime. All are best-effort — a browser may
// block audio or notifications, and that must never affect the inbox itself.
import { soundNotesFor } from "@/lib/attentionEvents";
import type { AttentionNotification } from "@cherrypi/sdk";

// The bundled icons: the tab icon, the legacy favicon fallback, and the
// home-screen (apple-touch) icon. Each link falls back to its own artwork, and
// all of them follow the agency's configured icon when there is one.
export const DEFAULT_FAVICON = "/favicon.png";
const DEFAULT_SHORTCUT_ICON = "/favicon.ico";
const DEFAULT_APPLE_ICON = "/apple-touch-icon.png";

// The icon the shell last applied; also the desktop-notification icon.
let currentIcon = DEFAULT_FAVICON;
// The favicon's source URL and decoded image, cached between redraws.
let faviconImage: HTMLImageElement | null = null;
let faviconImageSrc = "";
let faviconImageReady = false;
// The latest unread count, so an in-flight image load redraws with it.
let currentCount = 0;

// setIconLink points every link with `rel` at href, creating one when the
// document has none (the SPA's index.html ships all three).
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

// badgeDataUrl draws the red unread bubble over the cached icon, or null until
// that image has loaded.
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

/**
 * Point the document's icons at the agency's configured icon (the uploaded
 * favicon, else the logo), or at the bundled artwork when `iconUrl` is empty.
 * The tab icon and its legacy `shortcut icon` alias get a red unread bubble;
 * the home-screen icon is never badged. Changing the icon reloads the cached
 * image; the first call for an icon loads it asynchronously.
 */
export function applyFavicon(count: number, iconUrl?: string) {
  const configured = iconUrl?.trim() ?? "";
  const src = configured || DEFAULT_FAVICON;
  currentIcon = src;
  currentCount = count;

  // The home-screen icon is never badged; it follows the agency icon when one
  // is set and the bundled apple-touch artwork otherwise.
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
      // A newer icon may have won the race while this one loaded.
      if (faviconImageSrc !== src) return;
      faviconImage = image;
      faviconImageReady = true;
      showBadge();
    };
    image.src = src;
    return;
  }

  if (faviconImageReady) {
    showBadge();
  }
}

let audioContext: AudioContext | null = null;

// One note lasts this long; notes in a motif start this far apart so they do
// not overlap.
const NOTE_SECONDS = 0.16;
const NOTE_SPACING = 0.14;

/**
 * Play the chime for one attention kind: a short motif of notes from the event
 * catalog (see [soundNotesFor]). Browsers without the Web Audio API, or that
 * block autoplay, are silently ignored.
 */
export function playAttentionSound(kind = "") {
  try {
    const Ctor = window.AudioContext;
    if (!Ctor) return;

    audioContext ??= new Ctor();
    const ctx = audioContext;
    if (ctx.state === "suspended") {
      void ctx.resume();
    }

    const start = ctx.currentTime;
    soundNotesFor(kind).forEach((frequency, index) => {
      const at = start + index * NOTE_SPACING;
      const oscillator = ctx.createOscillator();
      const gain = ctx.createGain();

      oscillator.type = "sine";
      oscillator.frequency.setValueAtTime(frequency, at);

      gain.gain.setValueAtTime(0.0001, at);
      gain.gain.exponentialRampToValueAtTime(0.18, at + 0.015);
      gain.gain.exponentialRampToValueAtTime(0.0001, at + NOTE_SECONDS + 0.1);

      oscillator.connect(gain);
      gain.connect(ctx.destination);

      oscillator.start(at);
      oscillator.stop(at + NOTE_SECONDS + 0.11);
    });
  } catch {
    // Audio is a nicety; never let it break the notification path.
  }
}

// Notifications still on screen. Holding a reference keeps them from being
// reclaimed before the browser has painted them.
const liveNotifications = new Set<Notification>();
// Alerts dropped while permission is still undecided. They are shown once the
// user grants permission, so a default-on event is not silently lost.
let pendingNotifications: AttentionNotification[] = [];
let permissionHandler: ((event: Event) => void) | null = null;

function display(n: AttentionNotification) {
  try {
    const notification = new Notification(n.title, {
      body: n.body,
      icon: currentIcon,
      // Stored notifications have a row id; a kind the inbox does not store is
      // still pushed with id 0, so fall back to the timestamp to keep the tag
      // unique.
      tag: `cherrypi-${n.id}-${n.createdAt}`,
      // The OS chime is generic and would fire for every browser-enabled event,
      // overriding the Play Sound toggle. Our own per-event chime is the only
      // sound the user controls.
      silent: true,
    });
    liveNotifications.add(notification);
    notification.onclose = () => liveNotifications.delete(notification);
  } catch {
    // Some browsers throw when notifications are disallowed; ignore it.
  }
}

// The permission prompt only works from a user gesture. Arm one-shot listeners
// so the next click or keypress asks, then flushes whatever was queued.
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

/** Drop any queued alert and armed permission listener. Used to reset tests. */
export function resetAttention() {
  if (permissionHandler) {
    window.removeEventListener("pointerdown", permissionHandler, true);
    window.removeEventListener("keydown", permissionHandler, true);
    permissionHandler = null;
  }
  pendingNotifications = [];
  liveNotifications.clear();
  audioContext = null;
  faviconImage = null;
  faviconImageReady = false;
  faviconImageSrc = "";
  currentIcon = DEFAULT_FAVICON;
  currentCount = 0;
}

/**
 * Arm the permission prompt when it is still undecided, so the first gesture
 * anywhere asks (rather than only after an alert arrives). Idempotent, and a
 * no-op once permission was granted or denied.
 */
export function armAttentionPermission() {
  if (typeof Notification === "undefined") return;
  if (Notification.permission !== "default") return;
  armPermissionRequest();
}

/**
 * Show a desktop notification for a new attention signal. When permission is
 * still undecided the signal is queued and permission is requested on the
 * user's next gesture (see [requestAttentionPermission]). Once denied or
 * unsupported, alerts are dropped silently.
 */
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

/** The current desktop-notification permission, or "unsupported". */
export function attentionPermission(): NotificationPermission | "unsupported" {
  if (typeof Notification === "undefined") return "unsupported";
  return Notification.permission;
}

/**
 * Ask the browser for desktop-notification permission. Must be called from a
 * user gesture; resolves to the resulting permission.
 */
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
