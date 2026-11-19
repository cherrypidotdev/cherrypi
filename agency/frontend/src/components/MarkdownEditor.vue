<!--
CherryPi: A multi-agent-multi-human self-hosted agency
Copyright (C) 2026 CherryPi authors <cherrypi.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<template lang="pug">
div(:class="rootClass" :style="rootStyle" :data-testid="testId")
  MarkdownToolbar(:view="view" :disabled="disabled")
  .relative.min-h-0.flex-1(ref="host")

  .absolute.inset-x-0.bottom-0.z-10.flex.h-2.cursor-row-resize.items-center.justify-center(
    v-if="resizable && !fill"
    :title="$t('markdownEditor.resizeHint')"
    @pointerdown="startResize"
    data-testid="markdown-editor-resize"
  )
    .bg-border.h-1.w-10.rounded-full.transition-opacity(
      :class="dragging ? 'opacity-100' : 'opacity-40 hover:opacity-100'"
    )

  Teleport(to="body")
    .fixed.z-50.flex.flex-col(
      v-if="referencePickerState.open"
      ref="referencePickerRoot"
      :style="referencePickerStyle"
      data-dismissable-layer
    )
      ReferencePicker(
        ref="referencePicker"
        :search="referencePickerState.search"
        @pick="onReferencePicked"
      )
    .fixed.z-50.flex.flex-col(
      v-if="taskPickerState.open"
      ref="taskPickerRoot"
      :style="taskPickerStyle"
      data-dismissable-layer
    )
      TaskPicker(
        ref="taskPicker"
        :search="taskPickerState.search"
        :reference-project-id="referenceProjectId"
        :exclude-ids="taskPickerExcludeIds"
        @pick="onTaskPicked"
      )
</template>

<script setup lang="ts">
import MarkdownToolbar from "@/components/MarkdownToolbar.vue";
import { EDITOR_MIN_HEIGHT, clampEditorHeight } from "@/lib/editorHeight";
import { markdownFormattingKeymap } from "@/lib/markdownCommands";
import { markdownSpellcheck } from "@/lib/markdownSpellcheck";
import { IS_MAC } from "@/lib/platform";
import {
  referenceMarkdown,
  taskReferenceMarkdown,
  type ReferenceSuggestion,
} from "@/lib/references";
import { cn } from "@/lib/utils";
import type { Task } from "@cherrypi/sdk";
import { closeBrackets, closeBracketsKeymap } from "@codemirror/autocomplete";
import {
  addCursorAbove,
  addCursorBelow,
  defaultKeymap,
  deleteToLineEnd,
  history,
  historyKeymap,
  indentWithTab,
  insertNewlineAndIndent,
} from "@codemirror/commands";
import {
  markdown,
  markdownKeymap,
  markdownLanguage,
} from "@codemirror/lang-markdown";
import {
  HighlightStyle,
  syntaxHighlighting,
  syntaxTree,
} from "@codemirror/language";
import {
  highlightSelectionMatches,
  searchKeymap,
  selectNextOccurrence,
} from "@codemirror/search";
import { Compartment, EditorState } from "@codemirror/state";
import {
  EditorView,
  crosshairCursor,
  drawSelection,
  keymap,
  placeholder as placeholderExtension,
  rectangularSelection,
  type ViewUpdate,
} from "@codemirror/view";
import { tags } from "@lezer/highlight";
import { useEventListener } from "@vueuse/core";
import {
  computed,
  defineAsyncComponent,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  shallowRef,
  watch,
} from "vue";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    placeholder?: string;
    disabled?: boolean;

    height?: string;

    fill?: boolean;

    resizable?: boolean;

    resizeHeight?: number;

    submitOnEnter?: boolean;

    references?: boolean;

    referenceProjectId?: string;

    referenceTaskId?: string;
    testId?: string;
    ariaLabel?: string;
  }>(),
  {
    placeholder: "",
    disabled: false,
    height: "10rem",
    fill: false,
    resizable: false,
    resizeHeight: undefined,
    submitOnEnter: false,
    references: false,
    referenceProjectId: "",
    referenceTaskId: "",
    testId: undefined,
    ariaLabel: "Markdown editor",
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: string];

  submit: [];

  "task-picked": [task: Task];

  "update:resizeHeight": [height: number];
}>();

const host = ref<HTMLElement>();
const view = shallowRef<EditorView>();

const dragHeight = ref(EDITOR_MIN_HEIGHT);
const boxHeight = computed(() => props.resizeHeight ?? dragHeight.value);
const dragging = ref(false);
let dragStartY = 0;
let dragStartHeight = EDITOR_MIN_HEIGHT;

const TaskPicker = defineAsyncComponent(
  () => import("@/components/tasks/TaskPicker.vue"),
);

const ReferencePicker = defineAsyncComponent(
  () => import("@/components/tasks/ReferencePicker.vue"),
);

interface InlinePickerHandle {
  moveActive: (delta: number) => void;
  pickActive: () => boolean;
}
const TASK_PICKER_WIDTH = 416;
const REFERENCE_PICKER_WIDTH = 432;

const PICKER_GAP = 4;
const PICKER_EDGE = 8;

const TASK_PICKER_FALLBACK_HEIGHT = 300;
const REFERENCE_PICKER_FALLBACK_HEIGHT = 320;
const taskPickerRoot = ref<HTMLElement>();
const taskPicker = ref<InlinePickerHandle>();
const taskPickerState = reactive({ open: false, from: 0, search: "" });
const taskPickerStyle = ref<Record<string, string>>({});
const taskPickerExcludeIds = computed(() =>
  props.referenceTaskId ? [props.referenceTaskId] : [],
);
const referencePickerRoot = ref<HTMLElement>();
const referencePicker = ref<InlinePickerHandle>();
const referencePickerState = reactive({ open: false, from: 0, search: "" });
const referencePickerStyle = ref<Record<string, string>>({});

function closePickers() {
  closeTaskPicker();
  closeReferencePicker();
}

useEventListener(
  window,
  "pointerdown",
  (event: PointerEvent) => {
    if (!taskPickerState.open && !referencePickerState.open) return;
    const target = event.target as Node | null;
    if (!target) return;
    if (taskPickerRoot.value?.contains(target)) return;
    if (referencePickerRoot.value?.contains(target)) return;
    if (host.value?.contains(target)) return;
    closePickers();
  },
  { capture: true },
);

useEventListener(
  window,
  "keydown",
  (event: KeyboardEvent) => {
    if (
      (!taskPickerState.open && !referencePickerState.open) ||
      event.key !== "Escape"
    ) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    closePickers();
    view.value?.focus();
  },
  { capture: true },
);

useEventListener(
  window,
  "keydown",
  (event: KeyboardEvent) => {
    if (
      (!taskPickerState.open && !referencePickerState.open) ||
      event.isComposing ||
      event.altKey ||
      event.ctrlKey ||
      event.metaKey ||
      event.shiftKey
    ) {
      return;
    }
    const target = event.target as Node | null;
    if (!target || !host.value?.contains(target)) return;
    const active = taskPickerState.open
      ? taskPicker.value
      : referencePicker.value;
    if (!active) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      event.stopPropagation();
      active.moveActive(event.key === "ArrowDown" ? 1 : -1);
      return;
    }
    if (event.key !== "Enter" && event.key !== "Tab") return;
    if (!active.pickActive()) return;
    event.preventDefault();
    event.stopPropagation();
  },
  { capture: true },
);

useEventListener(
  window,
  "scroll",
  () => {
    positionPickers();
  },
  { capture: true, passive: true },
);
useEventListener(window, "resize", () => {
  positionPickers();
});

function isEscaped(text: string, index: number): boolean {
  let backslashes = 0;
  for (let cursor = index - 1; cursor >= 0; cursor -= 1) {
    if (text[cursor] !== "\\") break;
    backslashes += 1;
  }
  return backslashes % 2 === 1;
}

type SyntaxNode = ReturnType<ReturnType<typeof syntaxTree>["resolveInner"]>;

function inCode(state: EditorState, position: number): boolean {
  for (
    let node: SyntaxNode | null = syntaxTree(state).resolveInner(position, 1);
    node;
    node = node.parent
  ) {
    if (
      node.name === "FencedCode" ||
      node.name === "CodeBlock" ||
      node.name === "InlineCode"
    ) {
      return true;
    }
  }
  return false;
}

/**
 * Whether an unmatched opening backtick on the hash's line puts it inside
 * inline code that is still being typed. The Markdown parser only marks a
 * closed pair as `InlineCode`, but a `#` after a lone opening backtick is code
 * in progress, and a task picker there would be in the way.
 */
function inUnclosedInlineCode(state: EditorState, position: number): boolean {
  const line = state.doc.lineAt(position);
  const text = state.sliceDoc(line.from, position);
  let open = 0;
  for (let index = 0; index < text.length; ) {
    if (text[index] !== "`") {
      index += 1;
      continue;
    }
    let run = 0;
    while (index + run < text.length && text[index + run] === "`") run += 1;
    if (open === 0 && isEscaped(text, index)) {
      index += run;
      continue;
    }

    if (open === 0) open = run;
    else if (run === open) open = 0;
    index += run;
  }
  return open > 0;
}

function taskQuerySearch(query: string): string | null {
  let search = "";
  let quoted = false;
  for (let index = 0; index < query.length; index += 1) {
    const char = query[index]!;
    if (index === 0 && char === '"') {
      quoted = true;
      continue;
    }
    if (char === '"') {
      quoted = !quoted;
      continue;
    }

    if (char === "\n" || char === "\r") return null;
    if (!quoted && char === "#") return null;
    if (!quoted && (char === " " || char === "\t")) return null;
    search += char;
  }
  return search;
}

function taskQueryAt(
  state: EditorState,
): { from: number; search: string } | null {
  const selection = state.selection.main;
  if (!selection.empty) return null;
  const before = state.sliceDoc(0, selection.head);
  const hash = before.lastIndexOf("#");
  if (hash < 0) return null;
  if (hash > 0 && before[hash - 1] === "#") return null;

  if (isEscaped(before, hash)) return null;
  if (inCode(state, hash) || inUnclosedInlineCode(state, hash)) return null;
  const search = taskQuerySearch(before.slice(hash + 1));
  if (search === null) return null;
  return { from: hash, search };
}

function atQueryAt(
  state: EditorState,
): { from: number; search: string } | null {
  const selection = state.selection.main;
  if (!selection.empty) return null;
  const before = state.sliceDoc(0, selection.head);
  const at = before.lastIndexOf("@");
  if (at < 0) return null;
  // A username, project slug or team name: the query ends at the first space

  const token = before.slice(at);
  if (!/^@[\p{L}\p{N}_.-]*$/u.test(token)) return null;

  if (at > 0 && /[\p{L}\p{N}_@]/u.test(before[at - 1]!)) return null;
  if (isEscaped(before, at)) return null;
  if (inCode(state, at) || inUnclosedInlineCode(state, at)) return null;
  return { from: at, search: token.slice(1) };
}

function charWasTyped(update: ViewUpdate, char: string): boolean {
  const head = update.state.selection.main.head;
  if (!update.state.selection.main.empty) return false;
  for (const transaction of update.transactions) {
    let insertedChar = false;
    transaction.changes.iterChanges((_fromA, _toA, fromB, _toB, inserted) => {
      if (
        fromB === head - 1 &&
        inserted.length === 1 &&
        inserted.toString() === char
      ) {
        insertedChar = true;
      }
    });
    if (insertedChar) return true;
  }
  return false;
}

function hashWasTyped(update: ViewUpdate): boolean {
  return charWasTyped(update, "#");
}

function atWasTyped(update: ViewUpdate): boolean {
  return charWasTyped(update, "@");
}

function querySpaceWasDeleted(update: ViewUpdate): boolean {
  if (!update.state.selection.main.empty) return false;
  const head = update.state.selection.main.head;
  for (const transaction of update.transactions) {
    let deleted = false;
    transaction.changes.iterChanges((fromA, toA, fromB, toB, inserted) => {
      if (inserted.length > 0 || fromB !== toB || toB !== head) return;
      if (toA - fromA !== 1) return;
      const removed = transaction.startState.sliceDoc(fromA, toA);
      if (removed !== " " && removed !== "\t") return;
      deleted = true;
    });
    if (deleted) return true;
  }
  return false;
}

function sameStyle(a: Record<string, string>, b: Record<string, string>) {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
  return [...keys].every((key) => a[key] === b[key]);
}

let positionFrame = 0;

function schedulePickerPosition() {
  if (positionFrame) return;
  positionFrame = requestAnimationFrame(() => {
    positionFrame = 0;
    positionPickers();
  });
}

function panelStyle(
  editor: EditorView,
  root: HTMLElement | undefined,
  widthLimit: number,
  fallbackHeight: number,
): Record<string, string> | null {
  const coords = editor.coordsAtPos(editor.state.selection.main.head);
  if (!coords) return null;
  const width = Math.min(widthLimit, window.innerWidth - PICKER_EDGE * 2);
  const left = Math.max(
    PICKER_EDGE,
    Math.min(coords.left, window.innerWidth - width - PICKER_EDGE),
  );
  const measured = root?.getBoundingClientRect().height ?? 0;
  const height = measured || fallbackHeight;
  const spaceBelow =
    window.innerHeight - coords.bottom - PICKER_GAP - PICKER_EDGE;
  const spaceAbove = coords.top - PICKER_GAP - PICKER_EDGE;
  const dropUp = height > spaceBelow && spaceAbove > spaceBelow;
  const style: Record<string, string> = {
    left: `${left}px`,
    width: `${width}px`,
    maxHeight: `${Math.max(0, dropUp ? spaceAbove : spaceBelow)}px`,

    pointerEvents: "auto",
  };
  if (dropUp) {
    style.bottom = `${window.innerHeight - coords.top + PICKER_GAP}px`;
  } else {
    style.top = `${coords.bottom + PICKER_GAP}px`;
  }
  return style;
}

function applyPanelStyle(
  target: { value: Record<string, string> },
  style: Record<string, string> | null,
) {
  if (style && !sameStyle(style, target.value)) target.value = style;
}

function positionTaskPicker() {
  const editor = view.value;
  if (!editor) return;
  applyPanelStyle(
    taskPickerStyle,
    panelStyle(
      editor,
      taskPickerRoot.value,
      TASK_PICKER_WIDTH,
      TASK_PICKER_FALLBACK_HEIGHT,
    ),
  );
}

function positionReferencePicker() {
  const editor = view.value;
  if (!editor) return;
  applyPanelStyle(
    referencePickerStyle,
    panelStyle(
      editor,
      referencePickerRoot.value,
      REFERENCE_PICKER_WIDTH,
      REFERENCE_PICKER_FALLBACK_HEIGHT,
    ),
  );
}

function positionPickers() {
  if (taskPickerState.open) positionTaskPicker();
  if (referencePickerState.open) positionReferencePicker();
}

let taskPickerResizeObserver: ResizeObserver | null = null;
watch(
  taskPickerRoot,
  (element) => {
    taskPickerResizeObserver?.disconnect();
    taskPickerResizeObserver = null;
    if (!element || typeof ResizeObserver === "undefined") return;
    taskPickerResizeObserver = new ResizeObserver(schedulePickerPosition);
    taskPickerResizeObserver.observe(element);
  },
  { flush: "post" },
);

let referencePickerResizeObserver: ResizeObserver | null = null;
watch(
  referencePickerRoot,
  (element) => {
    referencePickerResizeObserver?.disconnect();
    referencePickerResizeObserver = null;
    if (!element || typeof ResizeObserver === "undefined") return;
    referencePickerResizeObserver = new ResizeObserver(schedulePickerPosition);
    referencePickerResizeObserver.observe(element);
  },
  { flush: "post" },
);

watch(
  () => taskPickerState.open,
  (open) => {
    if (open) schedulePickerPosition();
  },
  { flush: "post" },
);
watch(
  () => referencePickerState.open,
  (open) => {
    if (open) schedulePickerPosition();
  },
  { flush: "post" },
);

function syncTaskPicker(update: ViewUpdate) {
  if (!props.references) {
    closeTaskPicker();
    return;
  }
  const match = taskQueryAt(update.state);
  if (!match) {
    closeTaskPicker();
    return;
  }
  if (!taskPickerState.open) {
    if (!props.referenceProjectId) return;

    if (!hashWasTyped(update) && !querySpaceWasDeleted(update)) return;
    taskPickerState.from = match.from;
    taskPickerState.open = true;
    closeReferencePicker();
  }
  taskPickerState.search = match.search;
  positionTaskPicker();
}

function syncReferencePicker(update: ViewUpdate) {
  if (!props.references) {
    closeReferencePicker();
    return;
  }
  const match = atQueryAt(update.state);
  if (!match) {
    closeReferencePicker();
    return;
  }
  if (!referencePickerState.open) {
    if (!atWasTyped(update) && !querySpaceWasDeleted(update)) return;
    referencePickerState.from = match.from;
    referencePickerState.open = true;
    closeTaskPicker();
  }
  referencePickerState.search = match.search;
  positionReferencePicker();
}

function closeTaskPicker() {
  if (!taskPickerState.open) return;
  taskPickerState.open = false;
  taskPickerState.search = "";
  taskPickerState.from = 0;
}

function closeReferencePicker() {
  if (!referencePickerState.open) return;
  referencePickerState.open = false;
  referencePickerState.search = "";
  referencePickerState.from = 0;
}

function onTaskPicked(task: Task) {
  const editor = view.value;
  if (!editor || !taskPickerState.open) return;
  const head = editor.state.selection.main.head;
  const from = Math.min(taskPickerState.from, head);
  const markdown = `${taskReferenceMarkdown(task)} `;
  editor.dispatch({
    changes: { from, to: head, insert: markdown },
    selection: { anchor: from + markdown.length },
    scrollIntoView: true,
    userEvent: "input.complete",
  });
  closeTaskPicker();
  editor.focus();
  emit("task-picked", task);
}

function onReferencePicked(suggestion: ReferenceSuggestion) {
  const editor = view.value;
  if (!editor || !referencePickerState.open) return;
  const head = editor.state.selection.main.head;
  const from = Math.min(referencePickerState.from, head);
  const markdown = `${referenceMarkdown(suggestion.reference)} `;
  editor.dispatch({
    changes: { from, to: head, insert: markdown },
    selection: { anchor: from + markdown.length },
    scrollIntoView: true,
    userEvent: "input.complete",
  });
  closeReferencePicker();
  editor.focus();
}

const readOnly = new Compartment();
const layout = new Compartment();
const attributes = new Compartment();
const placeholders = new Compartment();
const sending = new Compartment();

const rootClass = computed(() =>
  cn(
    "border-input focus-within:border-ring focus-within:ring-ring/50 relative flex w-full flex-col rounded-lg border bg-transparent transition-colors focus-within:ring-3",
    "dark:bg-input/30",
    props.disabled && "opacity-50",
    props.resizable && "overflow-hidden",
  ),
);

const rootStyle = computed(() => {
  if (props.fill) return {};
  if (props.resizable) {
    return { height: `${clampEditorHeight(boxHeight.value)}px` };
  }
  return { height: props.height };
});

const markdownHighlighting = HighlightStyle.define([
  { tag: tags.heading1, class: "cm-md-heading cm-md-heading-1" },
  { tag: tags.heading2, class: "cm-md-heading cm-md-heading-2" },
  { tag: tags.heading3, class: "cm-md-heading cm-md-heading-3" },
  {
    tag: [tags.heading4, tags.heading5, tags.heading6],
    class: "cm-md-heading",
  },
  { tag: tags.strong, class: "cm-md-strong" },
  { tag: tags.emphasis, class: "cm-md-emphasis" },
  { tag: tags.strikethrough, class: "cm-md-strikethrough" },
  { tag: [tags.link, tags.url], class: "cm-md-link" },
  { tag: tags.monospace, class: "cm-md-code" },
  { tag: tags.quote, class: "cm-md-quote" },
  {
    tag: [tags.processingInstruction, tags.meta, tags.contentSeparator],
    class: "cm-md-mark",
  },
  { tag: tags.comment, class: "cm-md-comment" },
]);

const baseTheme = EditorView.theme({
  "&": {
    color: "var(--foreground)",
    backgroundColor: "transparent",
    fontSize: "0.875rem",
    lineHeight: "1.5",
  },
  "&.cm-focused": { outline: "none" },
  ".cm-scroller": {
    fontFamily: "inherit",
    lineHeight: "1.5",
    overflowY: "auto",
  },
  ".cm-content": {
    padding: "0.5rem 0.625rem",
    caretColor: "var(--foreground)",
  },
  ".cm-line": { padding: "0" },
  ".cm-placeholder": {
    color: "var(--muted-foreground)",
    fontStyle: "normal",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection":
    {
      backgroundColor: "color-mix(in oklch, var(--primary) 18%, transparent)",
    },
  ".cm-cursor, .cm-dropCursor": {
    borderLeftColor: "var(--foreground)",
    borderLeftWidth: "1.5px",
  },
  ".cm-panels": {
    backgroundColor: "var(--card)",
    color: "var(--card-foreground)",
    borderColor: "var(--border-color)",
  },
  ".cm-panel.cm-search": {
    padding: "0.375rem 0.5rem",
    fontFamily: "inherit",
    fontSize: "0.75rem",
  },
  ".cm-panel.cm-search input, .cm-panel.cm-search button": {
    fontFamily: "inherit",
    fontSize: "0.75rem",
  },
  ".cm-textfield": {
    backgroundColor: "var(--background)",
    borderColor: "var(--border-color)",
    borderRadius: "0.375rem",
    color: "var(--foreground)",
  },
  ".cm-button": {
    backgroundImage: "none",
    backgroundColor: "var(--secondary)",
    borderColor: "var(--border-color)",
    borderRadius: "0.375rem",
    color: "var(--secondary-foreground)",
  },
  ".cm-searchMatch": {
    backgroundColor: "color-mix(in oklch, var(--primary) 16%, transparent)",
    outline: "1px solid color-mix(in oklch, var(--primary) 30%, transparent)",
  },
  ".cm-searchMatch.cm-searchMatch-selected": {
    backgroundColor: "color-mix(in oklch, var(--primary) 30%, transparent)",
  },
  ".cm-selectionMatch": {
    backgroundColor: "color-mix(in oklch, var(--primary) 10%, transparent)",
  },
  ".cm-matchingBracket, .cm-nonmatchingBracket": {
    backgroundColor: "transparent",
    outline: "1px solid var(--ring)",
  },
});

const fillLayout = EditorView.theme({
  "&": { height: "100%" },
  ".cm-scroller": { height: "100%", maxHeight: "none" },
  ".cm-content": { minHeight: "100%" },
});

function contentAttributes() {
  return EditorView.contentAttributes.of({
    "aria-label": props.ariaLabel,
    "aria-multiline": "true",
    role: "textbox",
    // CodeMirror ships `spellcheck="false"`; the editor is prose, so the

    spellcheck: "true",
  });
}

function readOnlyExtensions() {
  return [
    EditorState.readOnly.of(props.disabled),
    EditorView.editable.of(!props.disabled),
  ];
}

function layoutExtension() {
  return [fillLayout];
}

function submitCommand(editor: EditorView): boolean {
  if (editor.composing || props.disabled) return false;
  emit("submit");
  return true;
}

function sendingExtensions() {
  return keymap.of([
    { key: "Mod-Enter", run: submitCommand },
    ...(props.submitOnEnter ? [{ key: "Enter", run: submitCommand }] : []),
    { key: "Shift-Enter", run: insertNewlineAndIndent },
    ...markdownFormattingKeymap,
    { key: "Mod-k", run: deleteToLineEnd },
    { key: "Ctrl-k", run: deleteToLineEnd },
    { key: "Ctrl-Alt-ArrowUp", run: addCursorAbove },
    { key: "Ctrl-Alt-ArrowDown", run: addCursorBelow },
    { key: "Mod-Alt-ArrowUp", run: addCursorAbove },
    { key: "Mod-Alt-ArrowDown", run: addCursorBelow },
    { key: "Mod-d", run: selectNextOccurrence },
    indentWithTab,
  ]);
}

function extensions() {
  return [
    sending.of(sendingExtensions()),
    history(),
    drawSelection(),
    rectangularSelection(),
    crosshairCursor(),
    EditorView.lineWrapping,

    EditorView.clickAddsSelectionRange.of(
      (event) => event.altKey || (IS_MAC ? event.metaKey : event.ctrlKey),
    ),
    EditorState.allowMultipleSelections.of(true),
    EditorState.tabSize.of(2),
    markdown({ base: markdownLanguage, addKeymap: false }),
    syntaxHighlighting(markdownHighlighting),
    markdownSpellcheck,
    closeBrackets(),
    highlightSelectionMatches(),
    baseTheme,
    layout.of(layoutExtension()),
    readOnly.of(readOnlyExtensions()),
    attributes.of(contentAttributes()),
    placeholders.of(placeholderExtension(props.placeholder)),
    keymap.of([
      ...markdownKeymap,
      ...closeBracketsKeymap,
      ...searchKeymap,
      ...historyKeymap,
      ...defaultKeymap,
    ]),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        emit("update:modelValue", update.state.doc.toString());
      }
      if (update.docChanged || update.selectionSet) {
        syncTaskPicker(update);
        syncReferencePicker(update);
      }
    }),
  ];
}

onMounted(() => {
  if (!host.value) return;
  view.value = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: extensions(),
    }),
  });
});

onBeforeUnmount(() => {
  stopResize();
  if (positionFrame) cancelAnimationFrame(positionFrame);
  taskPickerResizeObserver?.disconnect();
  taskPickerResizeObserver = null;
  referencePickerResizeObserver?.disconnect();
  referencePickerResizeObserver = null;
  view.value?.destroy();
  view.value = undefined;
});

watch(
  () => props.modelValue,
  (value) => {
    const editor = view.value;
    if (!editor || value === editor.state.doc.toString()) return;
    editor.dispatch({
      changes: { from: 0, to: editor.state.doc.length, insert: value },
    });
  },
);

watch(
  () => props.disabled,
  () => {
    view.value?.dispatch({
      effects: readOnly.reconfigure(readOnlyExtensions()),
    });
  },
);

function startResize(event: PointerEvent) {
  if (event.button !== 0 || !props.resizable || props.fill) return;
  event.preventDefault();
  dragging.value = true;
  dragStartY = event.clientY;
  dragStartHeight = boxHeight.value;
  document.body.style.cursor = "row-resize";
  document.body.style.userSelect = "none";
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", stopResize, { once: true });
}

function onResizeMove(event: PointerEvent) {
  const next = clampEditorHeight(dragStartHeight + event.clientY - dragStartY);
  dragHeight.value = next;
  emit("update:resizeHeight", next);
}

function stopResize() {
  dragging.value = false;
  document.body.style.cursor = "";
  document.body.style.userSelect = "";
  window.removeEventListener("pointermove", onResizeMove);
}

watch(
  () => [props.fill, props.resizable, props.height] as const,
  () => {
    view.value?.dispatch({
      effects: layout.reconfigure(layoutExtension()),
    });
  },
);

watch(
  () => props.references,
  (enabled) => {
    if (!enabled) closePickers();
  },
);

watch(
  () => props.referenceProjectId,
  (projectId) => {
    if (!projectId) closeTaskPicker();
  },
);

watch(
  () => props.ariaLabel,
  () => {
    view.value?.dispatch({
      effects: attributes.reconfigure(contentAttributes()),
    });
  },
);

watch(
  () => props.placeholder,
  (value) => {
    view.value?.dispatch({
      effects: placeholders.reconfigure(placeholderExtension(value)),
    });
  },
);

watch(
  () => props.submitOnEnter,
  () => {
    view.value?.dispatch({
      effects: sending.reconfigure(sendingExtensions()),
    });
  },
);

function focus() {
  view.value?.focus();
}

function focusEnd() {
  const editor = view.value;
  if (!editor) return;
  editor.focus();
  editor.dispatch({ selection: { anchor: editor.state.doc.length } });
}

function getValue(): string {
  return view.value?.state.doc.toString() ?? "";
}

function setValue(value: string) {
  const editor = view.value;
  if (!editor || value === editor.state.doc.toString()) return;
  editor.dispatch({
    changes: { from: 0, to: editor.state.doc.length, insert: value },
    selection: { anchor: value.length },
  });
}

defineExpose({ focus, focusEnd, getValue, setValue, view });
</script>

<i18n lang="yaml">
en:
  markdownEditor:
    resizeHint: "Drag to resize"
ru:
  markdownEditor:
    resizeHint: "Потяните, чтобы изменить размер"
</i18n>
