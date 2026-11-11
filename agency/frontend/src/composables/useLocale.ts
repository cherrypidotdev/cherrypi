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

import {
  applyLocale,
  currentLocale,
  detectBrowserLocale,
  isLocale,
  type Locale,
} from "@/i18n";
import { accountApi, currentAccount } from "@/lib/account";
import { computed } from "vue";

const locale = computed<Locale>(() => currentLocale());

let loadPromise: Promise<void> | undefined;

let loadedForActor: number | null = null;

export function applyPreferenceLocale(preference: string): void {
  const next = preference === "" ? detectBrowserLocale() : preference;
  if (isLocale(next)) applyLocale(next);
}

export function loadLocalePreference(): Promise<void> {
  const actorId = currentAccount.value?.id ?? null;
  if (loadPromise && loadedForActor === actorId) return loadPromise;
  loadedForActor = actorId;
  loadPromise = accountApi
    .getPreferences()
    .then((prefs) => applyPreferenceLocale(prefs.locale))
    .catch(() => {});
  return loadPromise;
}

export function resetLocalePreference(): void {
  loadPromise = undefined;
  loadedForActor = null;
}

/**
 * Switch language immediately and persist it for the signed-in user. Anonymous
 * visitors (the login page and invite flow) only get the local switch.
 *
 * The switch is optimistic: the UI flips first, and a failed write rolls it
 * back and rethrows so the caller can tell the user instead of silently
 * dropping the choice.
 */
export async function setLocale(next: Locale): Promise<void> {
  const previous = currentLocale();
  applyLocale(next);
  if (!currentAccount.value) return;
  try {
    await accountApi.updatePreferences({ locale: next });
  } catch (err) {
    applyLocale(previous);
    throw err;
  }
}

export function useLocale() {
  return { locale, setLocale };
}
