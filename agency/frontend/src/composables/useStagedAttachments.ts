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

import { attachmentFilename, isImageMediaType } from "@/lib/attachments";
import { newSketchFilename } from "@/lib/sketches";
import { computed, ref } from "vue";

/** A picked/dropped/pasted file held in the browser until its task exists. */
export interface StagedFile {
  kind: "file";
  id: string;
  file: File;
  filename: string;
  contentType: string;
  size: number;
  /** Object URL for an inline image preview, or "" for other files. */
  url: string;
}

/** A captured sketch held in the browser until its task exists. */
export interface StagedSketch {
  kind: "sketch";
  id: string;
  png: Blob;
  snapshot: { document: unknown };
  filename: string;
  url: string;
}

export type StagedAttachment = StagedFile | StagedSketch;

// jsdom (in tests) does not implement object URLs; an empty preview URL simply
// renders the file icon instead of a thumbnail.
function objectUrl(file: Blob): string {
  return typeof URL.createObjectURL === "function"
    ? URL.createObjectURL(file)
    : "";
}

function revoke(url: string) {
  if (url && typeof URL.revokeObjectURL === "function")
    URL.revokeObjectURL(url);
}

/**
 * Files and sketches awaiting a task. The create-task form does not own a task
 * until submit, so uploads are bound to a real owner only afterwards; until
 * then the picks live in memory with a local preview.
 */
export function useStagedAttachments() {
  const items = ref<StagedAttachment[]>([]);
  let seq = 0;

  function addFiles(files: Iterable<File>) {
    for (const file of files) {
      const image = isImageMediaType(file.type);
      items.value.push({
        kind: "file",
        id: `staged-${++seq}`,
        file,
        filename: attachmentFilename(file),
        contentType: file.type || "application/octet-stream",
        size: file.size,
        url: image ? objectUrl(file) : "",
      });
    }
  }

  function addSketch(png: Blob, snapshot: { document: unknown }) {
    items.value.push({
      kind: "sketch",
      id: `staged-${++seq}`,
      png,
      snapshot,
      filename: newSketchFilename(),
      url: objectUrl(png),
    });
  }

  function remove(id: string) {
    for (const item of items.value) if (item.id === id) revoke(item.url);
    items.value = items.value.filter((item) => item.id !== id);
  }

  function clear() {
    for (const item of items.value) revoke(item.url);
    items.value = [];
  }

  const count = computed(() => items.value.length);

  return { items, count, addFiles, addSketch, remove, clear };
}
