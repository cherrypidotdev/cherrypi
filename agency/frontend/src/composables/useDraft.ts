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

export function useDraft(getKey: () => string): Ref<string> {
  const draft = ref(loadDraft(getKey()));

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
