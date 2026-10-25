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

import { agency } from "@/lib/sdk";
import { type AgencySettings, type AgencySettingsInput } from "@cherrypi/sdk";
import { ref } from "vue";

const settings = ref<AgencySettings | undefined>();
const loading = ref(false);
const error = ref("");

async function loadSettings(force = false) {
  if (settings.value && !force) return;
  loading.value = true;
  try {
    settings.value = await agency.getSettings();
    applyPrimaryColor(settings.value.primaryColor);
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

async function saveSettings(
  next: AgencySettingsInput,
): Promise<AgencySettings> {
  settings.value = await agency.updateSettings(next);
  applyPrimaryColor(settings.value.primaryColor);
  return settings.value;
}

function applyPrimaryColor(hex: string) {
  const root = document.documentElement;
  const color = (hex ?? "").trim();
  if (!/^#[0-9a-fA-F]{6}$/.test(color)) {
    root.style.removeProperty("--primary");
    root.style.removeProperty("--primary-foreground");
    return;
  }
  root.style.setProperty("--primary", color);
  root.style.setProperty("--primary-foreground", contrastForeground(color));
}

function contrastForeground(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  return luminance > 0.6 ? "#0a0a0a" : "#fafafa";
}

export function useSettings() {
  return { settings, loading, error, loadSettings, saveSettings };
}
