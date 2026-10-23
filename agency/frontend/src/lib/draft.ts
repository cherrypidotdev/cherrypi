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

// Unsent text inputs survive a reload in localStorage. A follow-up chat draft
// is keyed by its task, and a task composer draft by the context it was written
// in (an agent page, a project page, or agency-wide), so each place keeps its
// own half-written prompt.
//
// Storage can be unavailable (private browsing, storage disabled) or full, so
// every access is guarded; a failure degrades to an in-memory-only draft.

const CHAT_PREFIX = "cherrypi:task-chat-draft:";
const COMPOSER_PREFIX = "cherrypi:task-composer-draft:";

// A context without an agent or project still needs a stable, unambiguous
// scope, so encode the missing side instead of leaving a bare separator.
const ANY = "any";

/** The localStorage key holding a task's unsent chat input. */
export function chatDraftKey(taskId: string): string {
  return `${CHAT_PREFIX}${taskId}`;
}

/** The localStorage key holding a composer draft for the given context. */
export function composerDraftKey(scope: {
  agent?: string;
  project?: string;
}): string {
  return `${COMPOSER_PREFIX}${scope.agent || ANY}:${scope.project || ANY}`;
}

/** The draft stored under `key`, or "" when none is saved. */
export function loadDraft(key: string): string {
  try {
    return localStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

/** Persist `value` under `key`; an empty value clears the entry. */
export function saveDraft(key: string, value: string): void {
  try {
    if (value) {
      localStorage.setItem(key, value);
    } else {
      localStorage.removeItem(key);
    }
  } catch {
    // Storage disabled: the draft just won't survive a reload.
  }
}
