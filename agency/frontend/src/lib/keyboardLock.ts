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

export function supportsKeyboardLock(): boolean {
  return keyboard() !== undefined;
}

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

export function unlockKeyboard(): void {
  try {
    keyboard()?.unlock();
  } catch {}
}
