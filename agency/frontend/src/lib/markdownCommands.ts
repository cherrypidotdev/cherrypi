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

import type {
  ChangeSpec,
  EditorState,
  Line,
  SelectionRange,
} from "@codemirror/state";
import { EditorSelection } from "@codemirror/state";
import type { EditorView, KeyBinding } from "@codemirror/view";

type Marker = string | ((inner: string) => string);

function inlineCodeMarker(inner: string): string {
  const longest = Math.max(
    0,
    ...(inner.match(/`+/g) ?? []).map((run) => run.length),
  );
  return "`".repeat(longest + 1);
}

function resolveMarker(marker: Marker, inner: string): string {
  return typeof marker === "string" ? marker : marker(inner);
}

interface RangeEdit {
  changes: ChangeSpec[];
  selection: SelectionRange;
}

function applySelectionEdit(
  view: EditorView,
  edit: (state: EditorState, range: SelectionRange) => RangeEdit,
): boolean {
  if (view.composing) return false;
  const { state } = view;
  const edits = state.selection.ranges.map((range) => edit(state, range));
  if (!edits.length) return false;
  view.dispatch({
    changes: edits.flatMap((change) => change.changes),
    selection: EditorSelection.create(
      edits.map((change) => change.selection),
      state.selection.mainIndex,
    ),
    scrollIntoView: true,
    userEvent: "input.format.markdown",
  });
  return true;
}

function wrappedOutside(
  state: EditorState,
  range: SelectionRange,
  resolved: string,
): boolean {
  const char = resolved[0]!;
  if ([...resolved].some((each) => each !== char)) {
    return (
      state.sliceDoc(Math.max(0, range.from - resolved.length), range.from) ===
        resolved &&
      state.sliceDoc(
        range.to,
        Math.min(state.doc.length, range.to + resolved.length),
      ) === resolved
    );
  }
  let before = 0;
  while (
    range.from - before > 0 &&
    state.sliceDoc(range.from - before - 1, range.from - before) === char
  ) {
    before += 1;
  }
  let after = 0;
  while (
    range.to + after < state.doc.length &&
    state.sliceDoc(range.to + after, range.to + after + 1) === char
  ) {
    after += 1;
  }
  const length = resolved.length;
  return (
    before >= length &&
    after >= length &&
    before % 2 === length % 2 &&
    after % 2 === length % 2
  );
}

export function toggleWrap(view: EditorView, marker: Marker): boolean {
  return applySelectionEdit(view, (state, range) => {
    const resolved = resolveMarker(
      range.empty ? resolveMarker(marker, "") : marker,
      state.sliceDoc(range.from, range.to),
    );

    if (range.empty) {
      return {
        changes: [{ from: range.from, insert: resolved + resolved }],
        selection: EditorSelection.cursor(range.from + resolved.length),
      };
    }

    if (wrappedOutside(state, range, resolved)) {
      return {
        changes: [
          { from: range.from - resolved.length, to: range.from },
          { from: range.to, to: range.to + resolved.length },
        ],
        selection: EditorSelection.range(
          range.from - resolved.length,
          range.to - resolved.length,
        ),
      };
    }

    const inner = state.sliceDoc(range.from, range.to);
    if (
      inner.length >= resolved.length * 2 &&
      inner.startsWith(resolved) &&
      inner.endsWith(resolved)
    ) {
      return {
        changes: [
          { from: range.from, to: range.from + resolved.length },
          { from: range.to - resolved.length, to: range.to },
        ],
        selection: EditorSelection.range(
          range.from,
          range.to - resolved.length * 2,
        ),
      };
    }

    return {
      changes: [
        { from: range.from, insert: resolved },
        { from: range.to, insert: resolved },
      ],
      selection: EditorSelection.range(
        range.from + resolved.length,
        range.to + resolved.length,
      ),
    };
  });
}

export function toggleBold(view: EditorView): boolean {
  return toggleWrap(view, "**");
}

export function toggleItalic(view: EditorView): boolean {
  return toggleWrap(view, "*");
}

export function toggleStrikethrough(view: EditorView): boolean {
  return toggleWrap(view, "~~");
}

export function toggleInlineCode(view: EditorView): boolean {
  return toggleWrap(view, inlineCodeMarker);
}

export function wrapSelectionInBackticks(view: EditorView): boolean {
  if (view.state.selection.ranges.every((range) => range.empty)) return false;
  return toggleInlineCode(view);
}

export function insertLink(view: EditorView): boolean {
  return applySelectionEdit(view, (state, range) => {
    const label = state.sliceDoc(range.from, range.to);
    const insert = `[${label}](url)`;

    const urlFrom = range.from + label.length + 3;
    return {
      changes: [{ from: range.from, to: range.to, insert }],
      selection: EditorSelection.range(urlFrom, urlFrom + 3),
    };
  });
}

export type HeadingLevel = 1 | 2 | 3;

/**
 * Set, demote or clear the ATX heading prefix of every line the selection
 * touches. Clicking the current level removes the prefix instead of re-adding
 * it, so the heading buttons toggle.
 */
export function toggleHeading(
  level: HeadingLevel,
): (view: EditorView) => boolean {
  return (view) =>
    applyLineEdits(view, (text) => {
      const match = /^(#{1,6})(?:[ \t]+|$)/.exec(text);
      const existing = match ? match[0] : "";
      if (match && match[1]!.length === level) {
        return { from: 0, to: existing.length, insert: "" };
      }
      return { from: 0, to: existing.length, insert: `${"#".repeat(level)} ` };
    });
}

const BULLET_PATTERN = /^[-*+][ \t]+/;
const ORDERED_PATTERN = /^\d+[.)][ \t]+/;
const QUOTE_PATTERN = /^>[ \t]?/;

/**
 * Add `prefix` to the selected lines, or remove the matching marker from all of
 * them when every line already has one. Indentation is preserved, and blank
 * lines inside a multi-line selection count as paragraph breaks.
 */
function toggleLinePrefix(
  view: EditorView,
  prefix: string,
  pattern: RegExp,
): boolean {
  const lines = selectedLines(view.state);
  const entries = lines.map(({ line }) => {
    const indent = /^[ \t]*/.exec(line.text)![0].length;
    const body = line.text.slice(indent);
    return { indent, body, blank: line.text.trim() === "" };
  });
  const skipped = (index: number) => lines.length > 1 && entries[index]!.blank;
  const considered = entries.filter((_entry, index) => !skipped(index));
  const remove =
    considered.length > 0 &&
    considered.every((entry) => pattern.test(entry.body));

  return applyLineEdits(view, (_text, index) => {
    if (skipped(index)) return null;
    const { indent, body } = entries[index]!;
    if (remove) {
      const match = pattern.exec(body);
      if (!match) return null;
      return { from: indent, to: indent + match[0].length, insert: "" };
    }
    if (pattern.test(body)) return null;
    return { from: indent, to: indent, insert: prefix };
  });
}

export function toggleBulletList(view: EditorView): boolean {
  return toggleLinePrefix(view, "- ", BULLET_PATTERN);
}

export function toggleOrderedList(view: EditorView): boolean {
  return toggleLinePrefix(view, "1. ", ORDERED_PATTERN);
}

export function toggleBlockquote(view: EditorView): boolean {
  return toggleLinePrefix(view, "> ", QUOTE_PATTERN);
}

function selectedLines(state: EditorState): { line: Line; number: number }[] {
  const numbers = new Set<number>();
  for (const range of state.selection.ranges) {
    const first = state.doc.lineAt(range.from).number;
    const last = state.doc.lineAt(range.to).number;
    for (let number = first; number <= last; number += 1) {
      numbers.add(number);
    }
  }
  return [...numbers]
    .sort((left, right) => left - right)
    .map((number) => ({ number, line: state.doc.line(number) }));
}

interface LineEdit {
  from: number;
  to?: number;
  insert: string;
}

/**
 * Rewrite whole lines with per-line edits that keep the indentation intact.
 * Every edit stays inside its own line, so undo, cursor mapping and
 * multi-cursor edits behave as usual.
 */
function applyLineEdits(
  view: EditorView,
  edit: (text: string, index: number) => LineEdit | null,
): boolean {
  const changes: ChangeSpec[] = [];
  selectedLines(view.state).forEach(({ line }, index) => {
    const change = edit(line.text, index);
    if (!change) return;
    changes.push(
      change.to === undefined
        ? { from: line.from + change.from, insert: change.insert }
        : {
            from: line.from + change.from,
            to: line.from + change.to,
            insert: change.insert,
          },
    );
  });
  if (!changes.length) return true;
  view.dispatch({
    changes,
    scrollIntoView: true,
    userEvent: "input.format.markdown",
  });
  return true;
}

export const markdownFormattingKeymap: KeyBinding[] = [
  { key: "Mod-b", run: toggleBold },
  { key: "Mod-i", run: toggleItalic },
  { key: "Mod-Shift-x", run: toggleStrikethrough },
  { key: "Mod-e", run: toggleInlineCode },
  { key: "`", run: wrapSelectionInBackticks },
];
