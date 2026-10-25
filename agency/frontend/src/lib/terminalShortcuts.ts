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

export interface ShortcutDisplay {
  id: string;
  label: string;
  keys: string;
}

export interface TerminalShortcut extends ShortcutDisplay {
  matches: (event: KeyboardEvent) => boolean;
}

function isCtrlShift(event: KeyboardEvent): boolean {
  return event.ctrlKey && event.shiftKey && !event.altKey && !event.metaKey;
}

function isAltShift(event: KeyboardEvent): boolean {
  return event.altKey && event.shiftKey && !event.ctrlKey && !event.metaKey;
}

function chord(event: KeyboardEvent, code: string): boolean {
  return (isCtrlShift(event) || isAltShift(event)) && event.code === code;
}

export const TERMINAL_SHORTCUTS: TerminalShortcut[] = [
  {
    id: "new-tab",
    label: "New terminal tab",
    keys: "Ctrl+Shift+T / Alt+Shift+T",
    matches: (event) => chord(event, "KeyT"),
  },
  {
    id: "close-tab",
    label: "Close active tab",
    keys: "Ctrl+Shift+W / Alt+Shift+W",
    matches: (event) => chord(event, "KeyW"),
  },
  {
    id: "next-tab",
    label: "Next tab",
    keys: "Alt+PageDown",
    matches: (event) =>
      event.altKey &&
      !event.ctrlKey &&
      !event.metaKey &&
      !event.shiftKey &&
      event.code === "PageDown",
  },
  {
    id: "prev-tab",
    label: "Previous tab",
    keys: "Alt+PageUp",
    matches: (event) =>
      event.altKey &&
      !event.ctrlKey &&
      !event.metaKey &&
      !event.shiftKey &&
      event.code === "PageUp",
  },
  {
    id: "help",
    label: "Show these shortcuts",
    keys: "Ctrl+Shift+/ / Alt+Shift+/",
    matches: (event) => chord(event, "Slash"),
  },
];

export const TERMINAL_SHORTCUT_EXTRAS: ShortcutDisplay[] = [
  { id: "tab-n", label: "Switch to tab N", keys: "Alt+1 … Alt+9" },
];

export const ALL_TERMINAL_SHORTCUTS: ShortcutDisplay[] = [
  ...TERMINAL_SHORTCUTS,
  ...TERMINAL_SHORTCUT_EXTRAS,
];

export function matchShortcut(event: KeyboardEvent): string | null {
  for (const shortcut of TERMINAL_SHORTCUTS) {
    if (shortcut.matches(event)) {
      return shortcut.id;
    }
  }
  if (
    event.altKey &&
    !event.ctrlKey &&
    !event.metaKey &&
    !event.shiftKey &&
    /^Digit[1-9]$/.test(event.code)
  ) {
    return `tab-${event.code.slice(5)}`;
  }
  return null;
}
