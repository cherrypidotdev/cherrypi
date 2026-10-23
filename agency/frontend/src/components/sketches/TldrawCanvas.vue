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
.tldraw-host.relative.h-full.w-full(ref="host" data-testid="tldraw-canvas")
</template>

<script setup lang="ts">
import { createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import {
  AssetRecordType,
  Tldraw,
  createShapeId,
  type Editor,
  type TLEditorSnapshot,
} from "tldraw";
import "tldraw/tldraw.css";
import { onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from "vue";

// Keep tldraw off its production license gate (see the module).
import "./tldrawLicense";

/** An image to open in the editor as an annotatable background. */
export interface SketchBackgroundImage {
  url: string;
  name: string;
  width: number;
  height: number;
  mimeType?: string;
}

const props = defineProps<{
  /** A tldraw snapshot to load into the canvas on mount. */
  snapshot?: unknown;
  /** An image shape to add on mount (used to annotate an existing image). */
  backgroundImage?: SketchBackgroundImage;
}>();

const emit = defineEmits<{ ready: [] }>();

const host = useTemplateRef<HTMLDivElement>("host");
const editor = shallowRef<Editor>();
let root: Root | undefined;

/** Add the background image as a locked-look image shape covering the origin. */
function addBackgroundImage(instance: Editor, image: SketchBackgroundImage) {
  const assetId = AssetRecordType.createId();
  instance.createAssets([
    {
      id: assetId,
      typeName: "asset",
      type: "image",
      props: {
        name: image.name,
        src: image.url,
        w: image.width,
        h: image.height,
        mimeType: image.mimeType ?? "image/png",
        isAnimated: false,
      },
      meta: {},
    },
  ]);
  instance.createShape({
    id: createShapeId(),
    type: "image",
    x: 0,
    y: 0,
    props: { assetId, w: image.width, h: image.height },
  });
}

// macOS two-finger swipe navigates back unless the document opts out of
// overscroll; toggle it for as long as the editor is mounted so drawing never
// triggers a history swipe.
const OVERSCROLL_CLASS = "sketch-editor-active";

onMounted(() => {
  document.documentElement.classList.add(OVERSCROLL_CLASS);
  if (!host.value) return;
  root = createRoot(host.value);
  root.render(
    createElement(Tldraw, {
      onMount: (instance: Editor) => {
        editor.value = instance;
        if (props.snapshot) {
          instance.loadSnapshot(props.snapshot as Partial<TLEditorSnapshot>);
        } else if (props.backgroundImage) {
          addBackgroundImage(instance, props.backgroundImage);
        }
        instance.selectNone();
        instance.zoomToFit();
        emit("ready");
      },
    }),
  );
});

onBeforeUnmount(() => {
  document.documentElement.classList.remove(OVERSCROLL_CLASS);
  root?.unmount();
  root = undefined;
  editor.value = undefined;
});

/** The number of shapes on the current page. */
function shapeCount(): number {
  return editor.value?.getCurrentPageShapeIds().size ?? 0;
}

/** Export the current page as a PNG plus its editable snapshot. */
async function capture(): Promise<
  { png: Blob; snapshot: TLEditorSnapshot } | undefined
> {
  const instance = editor.value;
  if (!instance) return undefined;
  const ids = [...instance.getCurrentPageShapeIds()];
  if (!ids.length) return undefined;
  const { blob } = await instance.toImage(ids, {
    format: "png",
    background: true,
    scale: 2,
  });
  return { png: blob, snapshot: instance.getSnapshot() };
}

defineExpose({ shapeCount, capture, editor });
</script>
