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
.border-border.flex.shrink-0.flex-wrap.items-center.gap-px.border-b.px-1(
  class="py-0.5"
  @mousedown.prevent
  data-testid="markdown-editor-toolbar"
)
  template(v-for="(action, index) in actions" :key="action.id")
    .bg-border.w-px(
      v-if="index > 0 && action.group !== actions[index - 1]?.group"
      class="mx-0.5 h-3.5"
      aria-hidden="true"
      data-testid="markdown-editor-toolbar-separator"
    )
    Button.size-5(
      type="button"
      variant="ghost"
      size="icon-xs"
      :title="actionTitle(action)"
      :aria-label="actionTitle(action)"
      @click="run(action)"
      :disabled="disabled || !view"
      :data-testid="`markdown-editor-${action.id}`"
    )
      component(:is="action.icon")
  VoiceInputButton.ml-auto(:view="view" :disabled="disabled")
</template>

<script setup lang="ts">
import VoiceInputButton from "@/components/VoiceInputButton.vue";
import { Button } from "@/components/ui/button";
import {
  insertLink,
  toggleBlockquote,
  toggleBold,
  toggleBulletList,
  toggleHeading,
  toggleInlineCode,
  toggleItalic,
  toggleOrderedList,
  toggleStrikethrough,
} from "@/lib/markdownCommands";
import { IS_MAC } from "@/lib/platform";
import type { EditorView } from "@codemirror/view";
import {
  BoldIcon,
  CodeIcon,
  Heading1Icon,
  Heading2Icon,
  Heading3Icon,
  ItalicIcon,
  LinkIcon,
  ListIcon,
  ListOrderedIcon,
  StrikethroughIcon,
  TextQuoteIcon,
} from "@lucide/vue";
import type { Component } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{ view?: EditorView; disabled?: boolean }>(),
  { view: undefined, disabled: false },
);

interface ToolbarAction {
  id: string;

  titleKey: string;
  icon: Component;

  group: string;
  command: (view: EditorView) => boolean;
}

const mod = IS_MAC ? "⌘" : "Ctrl+";

const actions: ToolbarAction[] = [
  {
    id: "bold",
    titleKey: "toolbar.bold",
    icon: BoldIcon,
    group: "inline",
    command: toggleBold,
  },
  {
    id: "italic",
    titleKey: "toolbar.italic",
    icon: ItalicIcon,
    group: "inline",
    command: toggleItalic,
  },
  {
    id: "strikethrough",
    titleKey: "toolbar.strikethrough",
    icon: StrikethroughIcon,
    group: "inline",
    command: toggleStrikethrough,
  },
  {
    id: "code",
    titleKey: "toolbar.code",
    icon: CodeIcon,
    group: "inline",
    command: toggleInlineCode,
  },
  {
    id: "link",
    titleKey: "toolbar.link",
    icon: LinkIcon,
    group: "link",
    command: insertLink,
  },
  {
    id: "heading-1",
    titleKey: "toolbar.heading1",
    icon: Heading1Icon,
    group: "heading",
    command: toggleHeading(1),
  },
  {
    id: "heading-2",
    titleKey: "toolbar.heading2",
    icon: Heading2Icon,
    group: "heading",
    command: toggleHeading(2),
  },
  {
    id: "heading-3",
    titleKey: "toolbar.heading3",
    icon: Heading3Icon,
    group: "heading",
    command: toggleHeading(3),
  },
  {
    id: "bullet-list",
    titleKey: "toolbar.bulletList",
    icon: ListIcon,
    group: "block",
    command: toggleBulletList,
  },
  {
    id: "ordered-list",
    titleKey: "toolbar.numberedList",
    icon: ListOrderedIcon,
    group: "block",
    command: toggleOrderedList,
  },
  {
    id: "blockquote",
    titleKey: "toolbar.quote",
    icon: TextQuoteIcon,
    group: "block",
    command: toggleBlockquote,
  },
];

function actionTitle(action: ToolbarAction): string {
  return t(action.titleKey, { mod, shift: IS_MAC ? "⇧" : "Shift+" });
}

function run(action: ToolbarAction) {
  const editor = props.view;
  if (!editor || props.disabled) return;
  action.command(editor);

  editor.focus();
}
</script>

<i18n lang="yaml">
en:
  toolbar:
    bold: "Bold ({mod}B)"
    italic: "Italic ({mod}I)"
    strikethrough: "Strikethrough ({mod}{shift}X)"
    code: "Inline code ({mod}E)"
    link: "Link"
    heading1: "Heading 1"
    heading2: "Heading 2"
    heading3: "Heading 3"
    bulletList: "Bullet list"
    numberedList: "Numbered list"
    quote: "Quote"
ru:
  toolbar:
    bold: "Жирный ({mod}B)"
    italic: "Курсив ({mod}I)"
    strikethrough: "Зачёркнутый ({mod}{shift}X)"
    code: "Код в строке ({mod}E)"
    link: "Ссылка"
    heading1: "Заголовок 1"
    heading2: "Заголовок 2"
    heading3: "Заголовок 3"
    bulletList: "Маркированный список"
    numberedList: "Нумерованный список"
    quote: "Цитата"
</i18n>
