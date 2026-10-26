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

import MarkdownEditor from "@/components/MarkdownEditor.vue";
import type { EditorView } from "@codemirror/view";
import type { VueWrapper } from "@vue/test-utils";
import { nextTick } from "vue";

export interface MarkdownEditorHandle {
  focus(): void;
  focusEnd(): void;
  getValue(): string;
  setValue(value: string): void;
  view: EditorView;
}

export function editorOf(wrapper: VueWrapper): MarkdownEditorHandle {
  const editor = wrapper.findComponent(MarkdownEditor);
  if (!editor.exists()) throw new Error("MarkdownEditor is not mounted");
  return editor.vm as unknown as MarkdownEditorHandle;
}

export function editorRoot(wrapper: VueWrapper): HTMLElement {
  return wrapper.findComponent(MarkdownEditor).element as HTMLElement;
}

export function editorContent(wrapper: VueWrapper): HTMLElement {
  const content = editorRoot(wrapper).querySelector<HTMLElement>(".cm-content");
  if (!content) throw new Error("CodeMirror content is not rendered");
  return content;
}

export async function setEditorText(wrapper: VueWrapper, text: string) {
  editorOf(wrapper).setValue(text);
  await nextTick();
}

export function editorText(wrapper: VueWrapper): string {
  return editorOf(wrapper).getValue();
}

export function pressKey(
  target: HTMLElement,
  init: KeyboardEventInit = {},
): KeyboardEvent {
  const event = new KeyboardEvent("keydown", {
    key: "Enter",
    code: "Enter",
    bubbles: true,
    cancelable: true,
    ...init,
  });
  target.dispatchEvent(event);
  return event;
}
