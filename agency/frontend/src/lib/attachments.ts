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

import type { Attachment } from "@cherrypi/sdk";

// The agent appends this trailer to a prompt carrying files. Each entry is a
// markdown link carrying the agency attachment id and the local path, e.g.
// `[mockup.png (#7)](/workspace/attachments/mockup.png)`. The console strips
// the trailer and uses the ids to render the right chips under the message.
const TRAILER_MARKER = "\n\n--- attached: ";
const TRAILER_SUFFIX = " ---";
const TRAILER_ENTRY = /\[[^\]]*\(#(\d+)\)\]\(([^)]*)\)/g;

/** The attachment ids a user message carried, and the text without its trailer. */
export interface ParsedTrailer {
  text: string;
  attachmentIds: string[];
}

/** Split a user message into its visible text and the attachment ids its
 * `--- attached: … ---` trailer references. A message without a trailer (or
 * one in an unknown shape) yields no ids and its text unchanged. */
export function parseAttachmentTrailer(text: string): ParsedTrailer {
  const trimmed = text.trimEnd();
  const index = trimmed.lastIndexOf(TRAILER_MARKER);
  if (index === -1 || !trimmed.endsWith(TRAILER_SUFFIX)) {
    return { text, attachmentIds: [] };
  }

  const body = trimmed.slice(
    index + TRAILER_MARKER.length,
    trimmed.length - TRAILER_SUFFIX.length,
  );
  const attachmentIds: string[] = [];
  for (const match of body.matchAll(TRAILER_ENTRY)) {
    attachmentIds.push(match[1]!);
  }
  return { text: trimmed.slice(0, index), attachmentIds };
}

/** Drop the agent's `--- attached: … ---` trailer from a user message. */
export function stripAttachmentTrailer(text: string): string {
  return parseAttachmentTrailer(text).text;
}

/** A filename for an upload source. A nameless source (a clipboard image
 * pasted straight from the OS) gets one derived from its media type. */
export function attachmentFilename(file: File): string {
  if (file.name) return file.name;
  const extension = (file.type.split("/")[1] || "bin").split("+")[0];
  return `pasted.${extension}`;
}

const IMAGE_EXTENSIONS = new Set(["png", "jpg", "jpeg", "webp", "gif"]);

/** Whether a filename carries one of the inline-safe raster extensions. */
export function isImageFilename(filename: string): boolean {
  const extension = filename.split(".").pop()?.trim().toLowerCase() ?? "";
  return IMAGE_EXTENSIONS.has(extension);
}

/** Whether a media type is a safe raster image that can render inline. */
export function isImageMediaType(mediaType: string): boolean {
  const type = mediaType.split(";")[0]!.trim().toLowerCase();
  return (
    type === "image/png" ||
    type === "image/jpeg" ||
    type === "image/webp" ||
    type === "image/gif"
  );
}

/** Whether an attachment is a safe raster image that can render inline. An
 * agent-side upload can arrive as `application/octet-stream` (a Blob without
 * a type), so a generic/missing type falls back to the filename. */
export function isImageAttachment(attachment: Attachment): boolean {
  const type = attachment.contentType.split(";")[0]!.trim().toLowerCase();
  if (type === "" || type === "application/octet-stream") {
    return isImageFilename(attachment.filename);
  }
  return isImageMediaType(attachment.contentType);
}

/** Human-readable file size, e.g. `4.2 MB` / `812 B`. */
export function formatFileSize(bytes: number | bigint): string {
  const size = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (!Number.isFinite(size) || size <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = size;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const rounded = unit === 0 ? value : Math.round(value * 10) / 10;
  return `${rounded} ${units[unit]}`;
}
