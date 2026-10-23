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

// The catalog of events the inbox can alert on. The `kind` values are the
// notification kinds the agency records; the browser decides which ones play a
// chime and pop a desktop notification. Keep in sync with the agency's
// notification kinds (`task_started`, `input_requested`, `task_finished`,
// `failed`, `task_completed`, `task_cancelled`, `team_invitation`).
export interface AttentionEventOption {
  /** Notification kind, as stored by the agency. */
  kind: string;
  /** Human label for the preferences table. */
  label: string;
  /**
   * Frequencies (Hz) of the chime's notes, played in order. Each event gets a
   * distinct motif so it is recognizable without looking at the screen.
   */
  notes: number[];
}

export const ATTENTION_EVENTS: AttentionEventOption[] = [
  // A rising fifth: the run picked up speed.
  { kind: "task_started", label: "Task started", notes: [659.25, 987.77] },
  // The existing two-tone question.
  { kind: "input_requested", label: "Task needs input", notes: [880, 1174.66] },
  // A soft descending pair: the run is done and idle.
  { kind: "task_finished", label: "Task settled", notes: [987.77, 783.99] },
  // Low, dissonant and insistent: something broke.
  { kind: "failed", label: "Task failed", notes: [311.13, 233.08] },
  // A rising major triad fanfare.
  {
    kind: "task_completed",
    label: "Task completed",
    notes: [523.25, 659.25, 783.99],
  },
  // A flat descending pair: dropped.
  { kind: "task_cancelled", label: "Task cancelled", notes: [392, 311.13] },
  // A bright rising fifth: a new team opened a door.
  { kind: "team_invitation", label: "Team invitation", notes: [587.33, 880] },
];

// Used for kinds the catalog does not know about.
export const DEFAULT_SOUND_NOTES = [880, 1174.66];

/** The chime motifs by notification kind. */
export function soundNotesFor(kind: string): number[] {
  return (
    ATTENTION_EVENTS.find((event) => event.kind === kind)?.notes ??
    DEFAULT_SOUND_NOTES
  );
}

/** The two alert channels a task event can trigger. */
export type AttentionChannel = "sound" | "browser";

/** Whether each channel is enabled for one event kind. */
export interface AttentionPreference {
  sound: boolean;
  browser: boolean;
}

/** Per-kind alert channels, keyed by notification kind. */
export type AttentionPreferences = Record<string, AttentionPreference>;

// The events worth interrupting the user for out of the box: a wait that needs
// attention, a failure, and a finished run. Everything else is opt-in.
const ENABLED_BY_DEFAULT = new Set([
  "input_requested",
  "failed",
  "task_finished",
]);

/** Fresh preferences: built-in defaults for every catalog event. */
export function defaultAttentionPreferences(): AttentionPreferences {
  const out: AttentionPreferences = {};
  for (const event of ATTENTION_EVENTS) {
    const enabled = ENABLED_BY_DEFAULT.has(event.kind);
    out[event.kind] = { sound: enabled, browser: enabled };
  }
  return out;
}
