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

export function attachmentFilename(file: File): string {
  if (file.name) return file.name;
  const extension = (file.type.split("/")[1] || "bin").split("+")[0];
  return `pasted.${extension}`;
}

const IMAGE_EXTENSIONS = new Set(["png", "jpg", "jpeg", "webp", "gif"]);

export function isImageFilename(filename: string): boolean {
  const extension = filename.split(".").pop()?.trim().toLowerCase() ?? "";
  return IMAGE_EXTENSIONS.has(extension);
}

export function isImageMediaType(mediaType: string): boolean {
  const type = mediaType.split(";")[0]!.trim().toLowerCase();
  return (
    type === "image/png" ||
    type === "image/jpeg" ||
    type === "image/webp" ||
    type === "image/gif"
  );
}

export function isImageAttachment(attachment: Attachment): boolean {
  const type = attachment.contentType.split(";")[0]!.trim().toLowerCase();
  if (type === "" || type === "application/octet-stream") {
    return isImageFilename(attachment.filename);
  }
  return isImageMediaType(attachment.contentType);
}

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
