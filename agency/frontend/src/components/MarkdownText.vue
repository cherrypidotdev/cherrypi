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
component(ref="root" v-html="html" :is="tag")
</template>

<script setup lang="ts">
import DOMPurify from "dompurify";
import MarkdownIt from "markdown-it";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

// as line breaks. Streaming works because the parent re-renders on every delta

const md = new MarkdownIt({ html: true, linkify: true, breaks: true });

const fence = md.renderer.rules.fence!;
md.renderer.rules.fence = (tokens, idx, options, env, self) => {
  const token = tokens[idx]!;
  if (token.info.trim().toLowerCase().startsWith("mermaid")) {
    return `<pre class="mermaid">${md.utils.escapeHtml(token.content)}</pre>\n`;
  }
  return fence(tokens, idx, options, env, self);
};

const props = defineProps<{ text: string; inline?: boolean }>();

const tag = computed(() => (props.inline ? "span" : "div"));
const root = ref<HTMLElement>();
const html = computed(() =>
  DOMPurify.sanitize(
    props.inline ? md.renderInline(props.text) : md.render(props.text),
  ),
);

let timer: ReturnType<typeof setTimeout> | undefined;

async function renderMermaid() {
  const nodes = root.value?.querySelectorAll<HTMLElement>(
    "pre.mermaid:not([data-processed])",
  );
  if (!nodes || nodes.length === 0) {
    return;
  }

  const snapshot = html.value;
  try {
    const { default: mermaid } = await import("mermaid");

    if (snapshot !== html.value) {
      return;
    }
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme: document.documentElement.classList.contains("dark")
        ? "dark"
        : "default",
    });
    await mermaid.run({ nodes: Array.from(nodes), suppressErrors: true });
  } catch {}
}

function scheduleMermaid() {
  if (timer !== undefined) {
    clearTimeout(timer);
  }
  timer = setTimeout(() => void renderMermaid(), 200);
}

watch(html, scheduleMermaid, { flush: "post" });
onMounted(scheduleMermaid);
onBeforeUnmount(() => {
  if (timer !== undefined) {
    clearTimeout(timer);
  }
});
</script>
