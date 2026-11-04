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

import { useSettings } from "@/composables/useSettings";
import {
  clampSoundVolume,
  DEFAULT_SOUND_SET_ID,
  DEFAULT_SOUND_VOLUME,
  isSoundSetId,
} from "@/lib/sounds/sets";
import { computed, ref } from "vue";

const STORAGE_KEY = "cherrypi.notification.volume";

function loadVolume(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === null) return DEFAULT_SOUND_VOLUME;
    return clampSoundVolume(Number(raw));
  } catch {
    return DEFAULT_SOUND_VOLUME;
  }
}

const soundVolume = ref(loadVolume());

const { settings, loadSettings } = useSettings();

const soundSet = computed(() => {
  const id = settings.value?.notificationSoundSet ?? "";
  return isSoundSetId(id) ? id : DEFAULT_SOUND_SET_ID;
});

async function loadSoundPreferences(force = false): Promise<void> {
  soundVolume.value = loadVolume();
  await loadSettings(force);
}

function setSoundVolumePreview(value: number): void {
  soundVolume.value = clampSoundVolume(value);
}

function saveSoundVolume(): void {
  try {
    localStorage.setItem(STORAGE_KEY, String(soundVolume.value));
  } catch {}
}

function resetSoundPreferences(): void {
  soundVolume.value = DEFAULT_SOUND_VOLUME;
}

export function useSoundPreferences() {
  return {
    soundSet,
    soundVolume,
    load: loadSoundPreferences,
    setSoundVolumePreview,
    saveSoundVolume,
    reset: resetSoundPreferences,
  };
}

export { resetSoundPreferences, soundSet, soundVolume };
