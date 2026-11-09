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

import { isPromptTarget } from "@/lib/notificationLink";
import { plainNotificationText } from "@/lib/notificationText";
import type { AttentionNotification } from "@cherrypi/sdk";
import { h, type VNode } from "vue";
import type { RouteLocationRaw } from "vue-router";
import { toast, type Id, type ToastType } from "vue3-toastify";

export const MAX_ATTENTION_TOASTS = 5;

export const ATTENTION_TOAST_MS = 8000;

export const INPUT_REQUESTED_TOAST_MS = 20000;

const STICKY_KINDS = new Set(["failed"]);

export function isAttentionToastSticky(kind: string): boolean {
  return STICKY_KINDS.has(kind);
}

export function attentionToastAutoClose(kind: string): number | false {
  if (isAttentionToastSticky(kind)) return false;
  if (kind === "input_requested") return INPUT_REQUESTED_TOAST_MS;
  return ATTENTION_TOAST_MS;
}

const live: Id[] = [];

function forget(id: Id) {
  const index = live.indexOf(id);
  if (index >= 0) live.splice(index, 1);
}

function makeRoom() {
  while (live.length >= MAX_ATTENTION_TOASTS) {
    const oldest = live.shift();
    if (oldest !== undefined) toast.remove(oldest);
  }
}

let navigate: ((to: RouteLocationRaw) => void) | null = null;

export function bindAttentionNavigation(fn: (to: RouteLocationRaw) => void) {
  navigate = fn;
}

export function resetAttentionNavigation() {
  navigate = null;
}

export function navigateAttention(to: RouteLocationRaw) {
  navigate?.(to);
}

export function resetAttentionToasts() {
  for (const id of live.splice(0)) toast.remove(id);
}

function toastType(severity: string): ToastType {
  switch (severity) {
    case "error":
      return "error";
    case "warning":
      return "warning";
    default:
      return "info";
  }
}

function toastHint(n: AttentionNotification): string {
  if (n.taskId) return "Open task";
  if (n.kind === "team_invitation") return "Open teams";
  if (isPromptTarget(n)) return "Open prompt";
  return "";
}

function toastBody(n: AttentionNotification, open: () => void): VNode {
  const hint = toastHint(n);
  return h(
    "div",
    {
      class: "flex cursor-pointer flex-col gap-0.5",
      role: hint ? "button" : undefined,
      tabindex: hint ? 0 : undefined,
      "aria-label": hint ? `${hint}: ${n.title}` : undefined,
      onKeydown: (event: KeyboardEvent) => {
        if (!hint || (event.key !== "Enter" && event.key !== " ")) return;
        event.preventDefault();
        open();
      },
    },
    [
      h("span", { class: "font-medium" }, n.title),
      n.body
        ? h(
            "span",
            { class: "text-muted-foreground text-sm" },
            plainNotificationText(n.body),
          )
        : null,
      hint
        ? h(
            "span",
            { class: "text-muted-foreground mt-0.5 text-xs" },
            `${hint} →`,
          )
        : null,
    ],
  );
}

export function showAttentionToast(
  n: AttentionNotification,
  onOpen: () => void,
) {
  const id = `attention-${n.id}-${n.createdAt}`;
  const open = () => {
    onOpen();
    toast.remove(id);
  };
  makeRoom();
  live.push(id);

  toast(toastBody(n, open), {
    toastId: id,
    type: toastType(n.severity),
    position: "top-right",

    closeButton: true,

    clearOnUrlChange: false,
    closeOnClick: false,
    autoClose: attentionToastAutoClose(n.kind),
    onClick: open,
    onClose: () => forget(id),
  });
}
