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

// Keyboard Lock (https://w3c.github.io/keyboard-lock/) lets a page inside
// fullscreen receive the combinations the browser normally reserves for
// itself (Ctrl+W/T/N, Ctrl+P, …). A remote desktop needs those keys at the
// remote end, and noVNC cannot ask for them on its own.
//
// The API is secure-context-only and only honoured while a fullscreen element
// exists; Firefox and Safari do not implement it. Callers therefore treat the
// lock as a best-effort enhancement on top of plain fullscreen.

interface KeyboardLockAPI {
  lock(keys?: string | string[]): Promise<void>;
  unlock(): void;
}

function keyboard(): KeyboardLockAPI | undefined {
  if (typeof navigator === "undefined") {
    return undefined;
  }
  const api = (navigator as Navigator & { keyboard?: KeyboardLockAPI })
    .keyboard;
  if (
    !api ||
    typeof api.lock !== "function" ||
    typeof api.unlock !== "function"
  ) {
    return undefined;
  }
  return api;
}

/** Whether this browser exposes the Keyboard Lock API at all. */
export function supportsKeyboardLock(): boolean {
  return keyboard() !== undefined;
}

/**
 * Ask the browser to deliver every combination to the page. Resolves false when
 * the API is missing, the page is not fullscreen, or the browser refuses.
 */
export async function lockKeyboard(): Promise<boolean> {
  const api = keyboard();
  if (!api) {
    return false;
  }
  try {
    await api.lock();
    return true;
  } catch {
    return false;
  }
}

/** Release a previous lock; a no-op when the API is missing. */
export function unlockKeyboard(): void {
  try {
    keyboard()?.unlock();
  } catch {
    // Some browsers throw when nothing is locked; there is nothing to do.
  }
}
