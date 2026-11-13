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
.h-full.min-h-0(ref="host")
</template>

<script setup lang="ts">
import { jinjaHighlighting } from "@/lib/jinjaHighlight";
import {
  defaultKeymap,
  history,
  historyKeymap,
  indentWithTab,
} from "@codemirror/commands";
import { EditorState } from "@codemirror/state";
import { EditorView, drawSelection, keymap } from "@codemirror/view";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    label: string;
    invalid?: boolean;
  }>(),
  { invalid: false },
);

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const host = ref<HTMLElement | null>(null);
let view: EditorView | null = null;
let syncing = false;

const editorTheme = EditorView.theme({
  "&": {
    height: "100%",
    color: "var(--foreground)",
    backgroundColor: "transparent",
    fontSize: "0.75rem",
    lineHeight: "1.6",
  },
  "&.cm-focused": { outline: "none" },
  ".cm-scroller": {
    fontFamily:
      "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    height: "100%",
    overflow: "auto",
  },
  ".cm-content": {
    padding: "0.5rem",
    caretColor: "var(--foreground)",
  },
  ".cm-line": { padding: "0" },
  ".cm-cursor, .cm-dropCursor": {
    borderLeftColor: "var(--foreground)",
    borderLeftWidth: "1.5px",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection":
    {
      backgroundColor: "color-mix(in oklch, var(--primary) 18%, transparent)",
    },
  ".cm-placeholder": { color: "var(--muted-foreground)" },
});

function contentAttributes() {
  return EditorView.contentAttributes.of({
    "aria-label": props.label,
    "aria-invalid": props.invalid ? "true" : "false",
    "aria-multiline": "true",
    role: "textbox",
    spellcheck: "false",
  });
}

function createView(): EditorView {
  return new EditorView({
    doc: props.modelValue,
    parent: host.value!,
    extensions: [
      history(),
      drawSelection(),
      EditorState.tabSize.of(2),
      keymap.of([indentWithTab, ...defaultKeymap, ...historyKeymap]),
      EditorView.lineWrapping,
      jinjaHighlighting(),
      editorTheme,
      contentAttributes(),
      EditorView.updateListener.of((update) => {
        if (!update.docChanged || syncing) return;
        emit("update:modelValue", update.state.doc.toString());
      }),
    ],
  });
}

function insert(snippet: string) {
  if (!view) return;
  const selection = view.state.selection.main;
  view.dispatch({
    changes: { from: selection.from, to: selection.to, insert: snippet },
    selection: { anchor: selection.from + snippet.length },
  });
  view.focus();
}

function setValue(value: string) {
  if (!view) return;
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: value },
  });
}

function getValue(): string {
  return view?.state.doc.toString() ?? "";
}

defineExpose({ insert, setValue, getValue });

onMounted(() => {
  if (!host.value) return;
  view = createView();
});

watch(
  () => props.invalid,
  () => {
    view?.contentDOM.setAttribute(
      "aria-invalid",
      props.invalid ? "true" : "false",
    );
  },
);

watch(
  () => props.modelValue,
  (value) => {
    if (!view || view.state.doc.toString() === value) return;
    syncing = true;
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: value },
    });
    syncing = false;
  },
);

onBeforeUnmount(() => {
  view?.destroy();
  view = null;
});
</script>
