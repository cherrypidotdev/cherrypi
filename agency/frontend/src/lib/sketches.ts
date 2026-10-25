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

import { clone } from "@bufbuild/protobuf";
import { AttachmentSchema, type Attachment } from "@cherrypi/sdk";

const SCENE_PATTERN = /^sketch-(\d+)\.tldraw\.json$/;

export function sketchSceneFilename(imageId: bigint | string): string {
  return `sketch-${imageId.toString()}.tldraw.json`;
}

export function sketchSceneImageId(filename: string): string | null {
  return SCENE_PATTERN.exec(filename)?.[1] ?? null;
}

export function isSketchScene(attachment: Attachment): boolean {
  return sketchSceneImageId(attachment.filename) !== null;
}

export function sketchSceneFor(
  attachments: Attachment[],
  image: Attachment,
): Attachment | undefined {
  const name = sketchSceneFilename(image.id);
  return attachments.find(
    (attachment) => attachment.filename === name && isSketchScene(attachment),
  );
}

export function newSketchFilename(now = new Date()): string {
  const stamp = now.toISOString().replace(/[:.]/g, "-");
  const rand = Math.random().toString(36).slice(2, 8);
  return `sketch-${stamp}-${rand}.png`;
}

export function toStoredSnapshot(
  snapshot: { document: unknown } & Record<string, unknown>,
): string {
  return JSON.stringify({ document: snapshot.document });
}

export function bustAttachmentCache(attachment: Attachment): Attachment {
  const copy = clone(AttachmentSchema, attachment);
  const separator = copy.url.includes("?") ? "&" : "?";
  copy.url = `${copy.url}${separator}rev=${Date.now()}`;
  return copy;
}

export function loadImageSize(
  url: string,
): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () =>
      resolve({ width: image.naturalWidth, height: image.naturalHeight });
    image.onerror = () => reject(new Error("could not load image"));
    image.src = url;
  });
}
