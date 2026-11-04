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
component(
  ref="root"
  v-html="html"
  :is="tag"
  :class="{ 'md-zoomable': zoomable }"
  @click="onClick"
  @keydown="onKeydown"
)
</template>

<script setup lang="ts">
import type { InlineImage } from "@/lib/lightbox";
import { loadReferenceDirectory, referencePlugin } from "@/lib/references";
import DOMPurify from "dompurify";
import MarkdownIt from "markdown-it";
import {
  computed,
  getCurrentInstance,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import type { Router } from "vue-router";

// as line breaks. Streaming works because the parent re-renders on every delta

function createMarkdown(html: boolean) {
  const md = new MarkdownIt({ html, linkify: true, breaks: true });

  const fence = md.renderer.rules.fence!;
  md.renderer.rules.fence = (tokens, idx, options, env, self) => {
    const token = tokens[idx]!;
    if (token.info.trim().toLowerCase().startsWith("mermaid")) {
      return `<pre class="mermaid">${md.utils.escapeHtml(token.content)}</pre>\n`;
    }
    return fence(tokens, idx, options, env, self);
  };

  const image = md.renderer.rules.image!;
  md.renderer.rules.image = (tokens, idx, options, env, self) => {
    const rendered = image(tokens, idx, options, env, self);
    if (!env?.zoomable || insideLink(tokens, idx)) return rendered;
    const token = tokens[idx]!;
    const url = md.utils.escapeHtml(String(token.attrGet("src") ?? ""));
    const alt = md.utils.escapeHtml(token.content);
    return (
      `<span class="md-image" role="button" tabindex="0"` +
      ` data-lightbox-url="${url}" data-lightbox-alt="${alt}">` +
      rendered +
      `<span class="md-image-hint" aria-hidden="true"></span></span>`
    );
  };

  md.use(referencePlugin);
  return md;
}

function insideLink(tokens: { type: string }[], idx: number): boolean {
  let depth = 0;
  for (let i = idx - 1; i >= 0; i -= 1) {
    const token = tokens[i]!;
    if (token.type === "link_close") depth += 1;
    else if (token.type === "link_open") {
      if (depth === 0) return true;
      depth -= 1;
    }
  }
  return false;
}

const md = createMarkdown(true);

const mdMarkdownOnly = createMarkdown(false);

const props = withDefaults(
  defineProps<{
    text: string;
    inline?: boolean;

    allowHtml?: boolean;

    zoomable?: boolean;
  }>(),
  { allowHtml: true, zoomable: false },
);
const emit = defineEmits<{ openImage: [image: InlineImage] }>();

const tag = computed(() => (props.inline ? "span" : "div"));
const root = ref<HTMLElement>();
const markdown = computed(() =>
  props.allowHtml === false ? mdMarkdownOnly : md,
);
const html = computed(() =>
  DOMPurify.sanitize(
    props.inline
      ? markdown.value.renderInline(props.text, { zoomable: props.zoomable })
      : markdown.value.render(props.text, { zoomable: props.zoomable }),
  ),
);

const hasReferences = computed(() =>
  /\]\(ref:(?:actor|project|team)\//.test(props.text),
);
watch(
  hasReferences,
  (present) => {
    if (present) void loadReferenceDirectory();
  },
  { immediate: true },
);

const router = getCurrentInstance()?.appContext.config.globalProperties
  .$router as Router | undefined;

function onReferenceClick(event: MouseEvent) {
  if (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return;
  }
  const target = event.target as Element | null;
  const anchor = target?.closest<HTMLAnchorElement>("a.ref-chip");
  if (!anchor) return;
  const href = anchor.getAttribute("href");
  if (!href || !href.startsWith("/")) return;
  if (!router) return;
  event.preventDefault();
  void router.push(href);
}

function onClick(event: MouseEvent) {
  onReferenceClick(event);
  if (event.defaultPrevented || !props.zoomable) return;
  if (
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return;
  }
  const target = event.target as Element | null;

  const wrapper = target?.closest<HTMLElement>(".md-image");
  const img = target?.closest<HTMLImageElement>("img");
  const source = wrapper ?? img;
  if (!source) return;

  if (source.closest("a")) return;
  const url =
    wrapper?.getAttribute("data-lightbox-url") ?? img?.getAttribute("src");
  if (!url) return;
  event.preventDefault();
  emit("openImage", {
    url,
    alt:
      wrapper?.getAttribute("data-lightbox-alt") ??
      img?.getAttribute("alt") ??
      "",
  });
}

function onKeydown(event: KeyboardEvent) {
  if (!props.zoomable) return;
  if (event.key !== "Enter" && event.key !== " ") return;
  const target = event.target as Element | null;
  const wrapper = target?.closest<HTMLElement>(".md-image");
  if (!wrapper) return;
  const url = wrapper.getAttribute("data-lightbox-url");
  if (!url) return;
  event.preventDefault();
  emit("openImage", {
    url,
    alt: wrapper.getAttribute("data-lightbox-alt") ?? "",
  });
}

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

<style scoped>
:deep(.md-image) {
  position: relative;
  display: inline-block;
  max-width: 100%;
  cursor: zoom-in;
  border-radius: var(--radius-md);
}

:deep(.md-image > img) {
  display: block;
  max-width: 100%;
  height: auto;
  border-radius: inherit;
}

:deep(.md-image-hint) {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: var(--radius-sm);
  background-color: rgb(0 0 0 / 0.6);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23fff' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='15 3 21 3 21 9'/%3E%3Cpolyline points='9 21 3 21 3 15'/%3E%3Cline x1='21' y1='3' x2='14' y2='10'/%3E%3Cline x1='3' y1='21' x2='10' y2='14'/%3E%3C/svg%3E");
  background-position: center;
  background-repeat: no-repeat;
  background-size: 0.875rem;
  opacity: 0;
  pointer-events: none;
  transition: opacity 150ms;
}

:deep(.md-image:hover .md-image-hint),
:deep(.md-image:focus-visible .md-image-hint) {
  opacity: 1;
}

:deep(.md-image:focus-visible) {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}

:deep(.md-zoomable img) {
  cursor: zoom-in;
}
</style>
