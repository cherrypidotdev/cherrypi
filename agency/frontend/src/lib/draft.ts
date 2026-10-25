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

const CHAT_PREFIX = "cherrypi:task-chat-draft:";
const COMPOSER_PREFIX = "cherrypi:task-composer-draft:";

const ANY = "any";

export function chatDraftKey(taskId: string): string {
  return `${CHAT_PREFIX}${taskId}`;
}

export function composerDraftKey(scope: {
  agent?: string;
  project?: string;
}): string {
  return `${COMPOSER_PREFIX}${scope.agent || ANY}:${scope.project || ANY}`;
}

export function loadDraft(key: string): string {
  try {
    return localStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

export function saveDraft(key: string, value: string): void {
  try {
    if (value) {
      localStorage.setItem(key, value);
    } else {
      localStorage.removeItem(key);
    }
  } catch {}
}
