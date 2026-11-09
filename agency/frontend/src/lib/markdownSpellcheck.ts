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

/**
 * Browser spell checking for the Markdown editor (MarkdownEditor.vue).
 *
 * CodeMirror disables spell checking on its content element by default, so the
 * editor turns it on through `contentAttributes` and this extension switches it
 * back off inside code, where red squiggles over `foo_bar` or `/v1/tasks` would
 * only be noise. The attribute is inherited, so each code span carries its own
 * `spellcheck="false"` instead of the whole editor losing the feature.
 */
import { syntaxTree } from "@codemirror/language";
import type { EditorState } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  type DecorationSet,
  type ViewUpdate,
} from "@codemirror/view";

const CODE_NODES = new Set([
  "CodeBlock",
  "CodeText",
  "CodeInfo",
  "CodeMark",
  "FencedCode",
  "InlineCode",
]);

const noSpellcheck = Decoration.mark({ attributes: { spellcheck: "false" } });

/**
 * Mark every code range in `ranges` as unspellcheckable. Marks are added per
 * line because a Markdown code block spans several of them and inline
 * decorations stay within one line.
 */
export function codeSpellcheckDecorations(
  state: EditorState,
  ranges: readonly { from: number; to: number }[],
): DecorationSet {
  const marks: ReturnType<typeof noSpellcheck.range>[] = [];
  for (const { from, to } of ranges) {
    syntaxTree(state).iterate({
      from,
      to,
      enter: (node) => {
        if (!CODE_NODES.has(node.name)) return;
        const last = state.doc.lineAt(node.to).number;
        for (
          let number = state.doc.lineAt(node.from).number;
          number <= last;
          number += 1
        ) {
          const line = state.doc.line(number);
          const start = Math.max(line.from, node.from);
          const end = Math.min(line.to, node.to);
          if (end > start) marks.push(noSpellcheck.range(start, end));
        }

        return false;
      },
    });
  }
  return Decoration.set(marks, true);
}

export const markdownSpellcheck = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = codeSpellcheckDecorations(
        view.state,
        view.visibleRanges,
      );
    }

    update(update: ViewUpdate) {
      if (!update.docChanged && !update.viewportChanged) return;
      this.decorations = codeSpellcheckDecorations(
        update.state,
        update.view.visibleRanges,
      );
    }
  },
  { decorations: (plugin) => plugin.decorations },
);
