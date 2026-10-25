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

export interface SketchTarget {
  image?: Attachment;
  scene?: Attachment;
}

export interface SketchSavePayload {
  png: Blob;
  snapshot: { document: unknown };
  target: SketchTarget;
}

export function useSketchEditor(options: {
  siblings: () => Attachment[];

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

  function openNew() {
    target.value = {};
    snapshot.value = undefined;
    background.value = undefined;
    mounted.value = true;
    open.value = true;
  }

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
