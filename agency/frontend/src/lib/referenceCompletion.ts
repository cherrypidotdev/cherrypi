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
  loadReferenceDirectory,
  referenceMarkdown,
  searchReferences,
  type ReferenceKind,
} from "@/lib/references";
import type {
  Completion,
  CompletionContext,
  CompletionResult,
  CompletionSource,
} from "@codemirror/autocomplete";
import type { EditorView } from "@codemirror/view";

const AT_TOKEN = /@[\p{L}\p{N}_.-]*$/u;

function insertMarkdown(text: string) {
  return (
    view: EditorView,
    _completion: Completion,
    from: number,
    to: number,
  ) => {
    view.dispatch({
      changes: { from, to, insert: `${text} ` },
      selection: { anchor: from + text.length + 1 },
      scrollIntoView: true,
    });
  };
}

export function referenceCompletionSource(): CompletionSource {
  return async (
    context: CompletionContext,
  ): Promise<CompletionResult | null> => {
    const before = context.state.sliceDoc(0, context.pos);
    const at = AT_TOKEN.exec(before);
    if (!at) return null;
    return directoryCompletions(context, at[0].length);
  };
}

async function directoryCompletions(
  context: CompletionContext,
  length: number,
): Promise<CompletionResult | null> {
  const from = context.pos - length;
  const previous = from > 0 ? context.state.sliceDoc(from - 1, from) : "";

  if (/[\p{L}\p{N}_@]/u.test(previous)) return null;

  const query = context.state.sliceDoc(from + 1, context.pos);
  await loadReferenceDirectory();

  const suggestions = searchReferences(query);
  return {
    from,
    filter: false,
    validFor: AT_TOKEN,
    options: suggestions.map((suggestion) => ({
      label: suggestion.label,
      detail: suggestion.detail,
      section: completionSection(suggestion.reference.kind),
      type: suggestion.reference.kind,
      apply: insertMarkdown(referenceMarkdown(suggestion.reference)),
    })),
  };
}

function completionSection(kind: ReferenceKind): string {
  switch (kind) {
    case "actor":
      return "Actors";
    case "project":
      return "Projects";
    case "team":
      return "Teams";
    default:
      return "Tasks";
  }
}
