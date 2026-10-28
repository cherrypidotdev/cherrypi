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
  .h-full(ref="host")

  .absolute.inset-x-0.bottom-0.z-10.flex.h-2.cursor-row-resize.items-center.justify-center(
    v-if="resizable && !fill"
    title="Drag to resize"
    @pointerdown="startResize"
    data-testid="markdown-editor-resize"
  )
    .bg-border.h-1.w-10.rounded-full.transition-opacity(
      :class="dragging ? 'opacity-100' : 'opacity-40 hover:opacity-100'"
    )

  Teleport(to="body")
    .fixed.z-50(
      v-if="taskPickerState.open"
      ref="taskPickerRoot"
      :style="taskPickerStyle"
    )
      TaskPicker(
        :search="taskPickerState.search"
        :reference-project-id="referenceProjectId"
        :exclude-ids="taskPickerExcludeIds"
        @pick="onTaskPicked"
      )
</template>

<script setup lang="ts">
import { referenceCompletionSource } from "@/lib/referenceCompletion";
import { taskReferenceMarkdown } from "@/lib/references";
import { cn } from "@/lib/utils";
import type { Task } from "@cherrypi/sdk";
import {
  autocompletion,
  closeBrackets,
  closeBracketsKeymap,
  closeCompletion,
} from "@codemirror/autocomplete";
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
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
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
  tooltips,
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
}>();

const MAC = /Mac|iPhone|iPad/.test(navigator.platform);

const host = ref<HTMLElement>();
const view = shallowRef<EditorView>();

const MIN_HEIGHT = 64;
const MAX_HEIGHT = 384;
const dragHeight = ref(MIN_HEIGHT);
const dragging = ref(false);
let dragStartY = 0;
let dragStartHeight = MIN_HEIGHT;

const TaskPicker = defineAsyncComponent(
  () => import("@/components/tasks/TaskPicker.vue"),
);
const TASK_PICKER_WIDTH = 416;
const TASK_PICKER_HEIGHT = 300;
const taskPickerRoot = ref<HTMLElement>();
const taskPickerState = reactive({ open: false, from: 0, search: "" });
const taskPickerStyle = ref<Record<string, string>>({});
const taskPickerExcludeIds = computed(() =>
  props.referenceTaskId ? [props.referenceTaskId] : [],
);

useEventListener(
  window,
  "pointerdown",
  (event: PointerEvent) => {
    if (!taskPickerState.open) return;
    const target = event.target as Node | null;
    if (!target) return;
    if (taskPickerRoot.value?.contains(target)) return;
    if (host.value?.contains(target)) return;
    closeTaskPicker();
  },
  { capture: true },
);

useEventListener(
  window,
  "keydown",
  (event: KeyboardEvent) => {
    if (!taskPickerState.open || event.key !== "Escape") return;
    event.preventDefault();
    event.stopPropagation();
    closeTaskPicker();
    view.value?.focus();
  },
  { capture: true },
);

useEventListener(
  window,
  "scroll",
  () => {
    if (taskPickerState.open) positionTaskPicker();
  },
  { capture: true, passive: true },
);
useEventListener(window, "resize", () => {
  if (taskPickerState.open) positionTaskPicker();
});

function taskQueryAt(
  state: EditorState,
): { from: number; search: string } | null {
  const selection = state.selection.main;
  if (!selection.empty) return null;
  const before = state.sliceDoc(0, selection.head);
  const hash = before.lastIndexOf("#");
  if (hash < 0) return null;
  if (hash > 0 && before[hash - 1] === "#") return null;
  const search = before.slice(hash + 1);

  if (search.includes("\n") || search.includes("#")) return null;

  if (search.startsWith(" ") || search.startsWith("\t")) return null;
  return { from: hash, search };
}

function hashWasTyped(update: ViewUpdate): boolean {
  const head = update.state.selection.main.head;
  if (!update.state.selection.main.empty) return false;
  for (const transaction of update.transactions) {
    let insertedHash = false;
    transaction.changes.iterChanges((_fromA, _toA, fromB, _toB, inserted) => {
      if (
        fromB === head - 1 &&
        inserted.length === 1 &&
        inserted.toString() === "#"
      ) {
        insertedHash = true;
      }
    });
    if (insertedHash) return true;
  }
  return false;
}

function positionTaskPicker() {
  const editor = view.value;
  if (!editor) return;
  const coords = editor.coordsAtPos(editor.state.selection.main.head);
  if (!coords) return;
  const width = Math.min(TASK_PICKER_WIDTH, window.innerWidth - 16);
  const left = Math.max(
    8,
    Math.min(coords.left, window.innerWidth - width - 8),
  );
  const style: Record<string, string> = {
    left: `${left}px`,
    width: `${width}px`,
  };
  if (
    coords.bottom + TASK_PICKER_HEIGHT > window.innerHeight &&
    coords.top > TASK_PICKER_HEIGHT
  ) {
    style.bottom = `${window.innerHeight - coords.top + 4}px`;
  } else {
    style.top = `${coords.bottom + 4}px`;
  }
  taskPickerStyle.value = style;
}

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
    if (!props.referenceProjectId || !hashWasTyped(update)) return;
    taskPickerState.from = match.from;
    taskPickerState.open = true;
    if (view.value) closeCompletion(view.value);
  }
  taskPickerState.search = match.search;
  positionTaskPicker();
}

function closeTaskPicker() {
  if (!taskPickerState.open) return;
  taskPickerState.open = false;
  taskPickerState.search = "";
  taskPickerState.from = 0;
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
}

const readOnly = new Compartment();
const layout = new Compartment();
const attributes = new Compartment();
const placeholders = new Compartment();
const sending = new Compartment();
const references = new Compartment();

const rootClass = computed(() =>
  cn(
    "border-input focus-within:border-ring focus-within:ring-ring/50 relative w-full rounded-lg border bg-transparent transition-colors focus-within:ring-3",
    "dark:bg-input/30",
    props.disabled && "opacity-50",
    props.resizable && "overflow-hidden",
  ),
);

const rootStyle = computed(() => {
  if (props.fill) return {};
  if (props.resizable) {
    return { height: `${dragHeight.value}px` };
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
    tag: [
      tags.processingInstruction,
      tags.meta,
      tags.contentSeparator,
      tags.list,
    ],
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
    borderColor: "var(--border)",
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
    borderColor: "var(--border)",
    borderRadius: "0.375rem",
    color: "var(--foreground)",
  },
  ".cm-button": {
    backgroundImage: "none",
    backgroundColor: "var(--secondary)",
    borderColor: "var(--border)",
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

function referenceExtensions() {
  if (!props.references) return [];
  return [
    autocompletion({
      override: [referenceCompletionSource()],
      icons: false,
      tooltipClass: () => "cm-reference-tooltip",
    }),
  ];
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

    tooltips({ parent: document.body }),
    EditorView.clickAddsSelectionRange.of(
      (event) => event.altKey || (MAC ? event.metaKey : event.ctrlKey),
    ),
    EditorState.allowMultipleSelections.of(true),
    EditorState.tabSize.of(2),
    markdown({ base: markdownLanguage, addKeymap: false }),
    syntaxHighlighting(markdownHighlighting),
    closeBrackets(),
    highlightSelectionMatches(),
    baseTheme,
    layout.of(layoutExtension()),
    references.of(referenceExtensions()),
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
      if (update.docChanged || update.selectionSet) syncTaskPicker(update);
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
  dragStartHeight = dragHeight.value;
  document.body.style.cursor = "row-resize";
  document.body.style.userSelect = "none";
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", stopResize, { once: true });
}

function onResizeMove(event: PointerEvent) {
  const next = dragStartHeight + event.clientY - dragStartY;
  dragHeight.value = Math.min(MAX_HEIGHT, Math.max(MIN_HEIGHT, next));
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
    const editor = view.value;
    if (!editor) return;
    if (!enabled) {
      closeCompletion(editor);
      closeTaskPicker();
    }
    editor.dispatch({
      effects: references.reconfigure(referenceExtensions()),
    });
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
