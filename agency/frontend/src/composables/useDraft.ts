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

import { loadDraft, saveDraft } from "@/lib/draft";
import { ref, watch, type Ref } from "vue";

/**
 * A text ref mirrored into localStorage under the key returned by `getKey`.
 * The key follows the surrounding context (navigating between tasks or
 * agent/project pages), so the ref switches to that context's saved draft.
 */
export function useDraft(getKey: () => string): Ref<string> {
  const draft = ref(loadDraft(getKey()));

  // Synchronous swaps/writes keep a keystroke from being persisted under a key
  // the context has already moved on from.
  watch(
    getKey,
    (key) => {
      draft.value = loadDraft(key);
    },
    { flush: "sync" },
  );
  watch(
    draft,
    (value) => {
      saveDraft(getKey(), value);
    },
    { flush: "sync" },
  );

  return draft;
}
