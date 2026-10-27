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
</template>

<script setup lang="ts">
import { cn } from "@/lib/utils";
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
} from "@codemirror/view";
import { tags } from "@lezer/highlight";
import {
  computed,
  onBeforeUnmount,
  onMounted,
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

const readOnly = new Compartment();
const layout = new Compartment();
const attributes = new Compartment();
const placeholders = new Compartment();
const sending = new Compartment();

const rootClass = computed(() =>
  cn(
    "border-input focus-within:border-ring focus-within:ring-ring/50 relative w-full rounded-lg border bg-transparent transition-colors focus-within:ring-3",
    "dark:bg-input/30",
    props.disabled && "opacity-50",
    props.resizable && "resize-y overflow-hidden",
  ),
);

const rootStyle = computed(() => {
  if (props.fill) return {};
  if (props.resizable) {
    return { height: "4rem", minHeight: "4rem", maxHeight: "24rem" };
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
  (disabled) => {
    view.value?.dispatch({
      effects: readOnly.reconfigure([
        EditorState.readOnly.of(disabled),
        EditorView.editable.of(!disabled),
      ]),
    });
  },
);

watch(
  () => [props.fill, props.resizable, props.height] as const,
  () => {
    view.value?.dispatch({
      effects: layout.reconfigure(layoutExtension()),
    });
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
