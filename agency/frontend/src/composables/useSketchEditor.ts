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

import type { SketchBackgroundImage } from "@/components/sketches/TldrawCanvas.vue";
import { loadImageSize, sketchSceneFor } from "@/lib/sketches";
import type { Attachment } from "@cherrypi/sdk";
import type { TLEditorSnapshot } from "tldraw";
import { computed, ref } from "vue";
import { toast } from "vue3-toastify";

/** What the sketch editor is open for: a blank canvas, or an image (with its
 * saved scene sidecar when it has one) being annotated. */
export interface SketchTarget {
  image?: Attachment;
  scene?: Attachment;
}

/** A captured sketch plus what it was drawn over. */
export interface SketchSavePayload {
  png: Blob;
  snapshot: { document: unknown };
  target: SketchTarget;
}

/**
 * Drives the tldraw sketch dialog: its open/loaded/saving state, the title, and
 * what the editor was opened for. The caller supplies `persist` to store the
 * captured pair (uploading for a task, or staging for a not-yet-created one).
 * tldraw is heavy, so the dialog stays unmounted until the first sketch.
 */
export function useSketchEditor(options: {
  /** Attachments that may hold the scene sidecar for an image being annotated. */
  siblings: () => Attachment[];
  /** Store the captured sketch; a rejection keeps the editor open. */
  persist: (payload: SketchSavePayload) => Promise<void>;
}) {
  const open = ref(false);
  const mounted = ref(false);
  const saving = ref(false);
  const snapshot = ref<unknown>();
  const background = ref<SketchBackgroundImage>();
  const target = ref<SketchTarget>({});

  const title = computed(() => {
    if (target.value.scene) return "Edit sketch";
    if (target.value.image) return "Annotate image";
    return "New sketch";
  });

  /** Open a blank canvas for a brand new sketch. */
  function openNew() {
    target.value = {};
    snapshot.value = undefined;
    background.value = undefined;
    mounted.value = true;
    open.value = true;
  }

  /** Open the editor for an image: its saved scene when it has one, else the
   * image as an annotatable background. */
  async function openImage(attachment: Attachment) {
    const scene = sketchSceneFor(options.siblings(), attachment);
    target.value = { image: attachment, scene };
    snapshot.value = undefined;
    background.value = undefined;
    mounted.value = true;
    try {
      if (scene) {
        const res = await fetch(scene.url);
        if (!res.ok) throw new Error(`could not load sketch (${res.status})`);
        snapshot.value = await res.json();
      } else {
        const size = await loadImageSize(attachment.url);
        background.value = {
          url: attachment.url,
          name: attachment.filename,
          width: size.width,
          height: size.height,
        };
      }
      open.value = true;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function onSave(payload: { png: Blob; snapshot: TLEditorSnapshot }) {
    saving.value = true;
    try {
      await options.persist({ ...payload, target: target.value });
      open.value = false;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      saving.value = false;
    }
  }

  return {
    open,
    mounted,
    saving,
    snapshot,
    background,
    target,
    title,
    openNew,
    openImage,
    onSave,
  };
}
