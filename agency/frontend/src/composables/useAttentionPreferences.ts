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
  ATTENTION_EVENTS,
  type AttentionChannel,
  type AttentionPreference,
  type AttentionPreferences,
  defaultAttentionPreferences,
} from "@/lib/attentionEvents";
import { ref } from "vue";

const STORAGE_KEY = "cherrypi.attention.preferences";

function load(): AttentionPreferences {
  const base = defaultAttentionPreferences();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return base;
    const saved = JSON.parse(raw) as Partial<AttentionPreferences>;
    for (const event of ATTENTION_EVENTS) {
      const pref = saved[event.kind];
      if (
        pref &&
        typeof pref.sound === "boolean" &&
        typeof pref.browser === "boolean"
      ) {
        base[event.kind] = { sound: pref.sound, browser: pref.browser };
      }
    }
  } catch {}
  return base;
}

const preferences = ref<AttentionPreferences>(load());

function persist() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences.value));
  } catch {}
}

export function attentionPreference(kind: string): AttentionPreference {
  return preferences.value[kind] ?? { sound: false, browser: false };
}

export function setAttentionPreference(
  kind: string,
  channel: AttentionChannel,
  enabled: boolean,
) {
  const current = attentionPreference(kind);
  preferences.value = {
    ...preferences.value,
    [kind]: { ...current, [channel]: enabled },
  };
  persist();
}

export function resetAttentionPreferences() {
  preferences.value = defaultAttentionPreferences();
  persist();
}

export function useAttentionPreferences() {
  return {
    preferences,
    setPreference: setAttentionPreference,
    reset: resetAttentionPreferences,
  };
}
