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

export interface AttentionEventOption {
  kind: string;

  label: string;

  notes: number[];
}

export const ATTENTION_EVENTS: AttentionEventOption[] = [
  { kind: "task_started", label: "Task started", notes: [659.25, 987.77] },

  { kind: "input_requested", label: "Task needs input", notes: [880, 1174.66] },

  { kind: "task_finished", label: "Task settled", notes: [987.77, 783.99] },

  { kind: "failed", label: "Task failed", notes: [311.13, 233.08] },

  {
    kind: "task_completed",
    label: "Task completed",
    notes: [523.25, 659.25, 783.99],
  },

  { kind: "task_cancelled", label: "Task cancelled", notes: [392, 311.13] },

  { kind: "team_invitation", label: "Team invitation", notes: [587.33, 880] },
];

export const DEFAULT_SOUND_NOTES = [880, 1174.66];

export function soundNotesFor(kind: string): number[] {
  return (
    ATTENTION_EVENTS.find((event) => event.kind === kind)?.notes ??
    DEFAULT_SOUND_NOTES
  );
}

export type AttentionChannel = "sound" | "browser";

export interface AttentionPreference {
  sound: boolean;
  browser: boolean;
}

export type AttentionPreferences = Record<string, AttentionPreference>;

const ENABLED_BY_DEFAULT = new Set([
  "input_requested",
  "failed",
  "task_finished",
]);

export function defaultAttentionPreferences(): AttentionPreferences {
  const out: AttentionPreferences = {};
  for (const event of ATTENTION_EVENTS) {
    const enabled = ENABLED_BY_DEFAULT.has(event.kind);
    out[event.kind] = { sound: enabled, browser: enabled };
  }
  return out;
}
