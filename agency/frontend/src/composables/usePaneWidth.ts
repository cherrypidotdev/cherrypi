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

import {
  clampPaneWidth,
  loadPaneWidth,
  savePaneWidth,
  type PaneWidthRange,
} from "@/lib/paneWidth";
import { ref, watch, type Ref } from "vue";

export type PaneEdge = "left" | "right";

export interface PaneWidthOptions extends PaneWidthRange {
  fallback: number;

  edge: PaneEdge;
}

export interface ResizablePane {
  width: Ref<number>;

  dragging: Ref<boolean>;

  min: number;
  max: number;

  startResize: (event: PointerEvent) => void;

  resizeByKey: (event: KeyboardEvent) => boolean;
}

export function createResizablePane(
  key: string,
  options: PaneWidthOptions,
): ResizablePane {
  const width = ref(
    loadPaneWidth(key, options) ?? clampPaneWidth(options.fallback, options),
  );
  const dragging = ref(false);

  watch(width, (value) => savePaneWidth(key, value));

  let startX = 0;
  let startWidth = width.value;

  function resizeBy(delta: number) {
    const next =
      options.edge === "left" ? width.value - delta : width.value + delta;
    width.value = clampPaneWidth(next, options);
  }

  function onResizeMove(event: PointerEvent) {
    const delta = event.clientX - startX;
    const next =
      options.edge === "left" ? startWidth - delta : startWidth + delta;
    width.value = clampPaneWidth(next, options);
  }

  function stopResize() {
    dragging.value = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    window.removeEventListener("pointermove", onResizeMove);
    window.removeEventListener("pointerup", stopResize);

    window.removeEventListener("pointercancel", stopResize);
  }

  function startResize(event: PointerEvent) {
    if (event.button !== 0) return;
    event.preventDefault();
    dragging.value = true;
    startX = event.clientX;
    startWidth = width.value;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", onResizeMove);
    window.addEventListener("pointerup", stopResize);
    window.addEventListener("pointercancel", stopResize);
  }

  function resizeByKey(event: KeyboardEvent): boolean {
    const step = event.shiftKey ? 64 : 16;
    switch (event.key) {
      case "ArrowLeft":
        resizeBy(-step);
        return true;
      case "ArrowRight":
        resizeBy(step);
        return true;
      case "Home":
        width.value = options.min;
        return true;
      case "End":
        width.value = options.max;
        return true;
      default:
        return false;
    }
  }

  return {
    width,
    dragging,
    min: options.min,
    max: options.max,
    startResize,
    resizeByKey,
  };
}
