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

import type { Artifact, ArtifactPart, Attachment } from "@cherrypi/sdk";
import {
  isImageAttachment,
  isImageFilename,
  isImageMediaType,
} from "./attachments";
import { timestampMs } from "./console";
import { isSketchScene } from "./sketches";

export interface LightboxItem {
  key: string;
  url: string;
  filename: string;

  image: boolean;

  source: string;

  createdAt?: number;
}

export interface InlineImage {
  url: string;
  alt: string;
}

export interface TaskLightboxSources {
  attachments: Attachment[];

  stagedAttachments?: Attachment[];
  artifacts?: Artifact[];
}

export function attachmentItem(
  attachment: Attachment,
  source = "Task attachment",
): LightboxItem {
  return {
    key: `attachment:${attachment.id}`,
    url: attachment.url,
    filename: attachment.filename,
    image: isImageAttachment(attachment),
    source,
    createdAt: timestampMs(attachment.createdAt),
  };
}

export function isImageArtifactPart(part: ArtifactPart): boolean {
  if (part.content.case !== "file") return false;
  const type = (part.mediaType || "").split(";")[0]!.trim().toLowerCase();
  if (type === "" || type === "application/octet-stream") {
    return isImageFilename(part.filename);
  }
  return isImageMediaType(part.mediaType);
}

export function artifactPartItem(
  artifact: Artifact,
  part: ArtifactPart,
  index: number,
): LightboxItem | undefined {
  if (part.content.case !== "file") return undefined;
  const url = part.content.value.url;
  if (!url) return undefined;
  const name = artifact.name || artifact.artifactId;
  return {
    key: `artifact:${artifact.artifactId}:${index}`,
    url,
    filename: part.filename || name || "Artifact file",
    image: isImageArtifactPart(part),
    source: `Artifact · ${name}`,
    createdAt: timestampMs(artifact.createdAt),
  };
}

export function artifactItems(artifact: Artifact): LightboxItem[] {
  return artifact.parts
    .map((part, index) => artifactPartItem(artifact, part, index))
    .filter((item): item is LightboxItem => item !== undefined);
}

export function inlineImageItem(image: InlineImage): LightboxItem {
  const url = image.url.trim();
  return {
    key: `inline:${url}`,
    url,
    filename: image.alt.trim() || urlFilename(url) || "Image",
    image: true,
    source: inlineImageSource(url),
    createdAt: undefined,
  };
}

export function buildTaskLightboxItems({
  attachments,
  stagedAttachments = [],
  artifacts = [],
}: TaskLightboxSources): LightboxItem[] {
  const items: LightboxItem[] = [];
  const seen = new Set<string>();
  const add = (item: LightboxItem) => {
    if (!item.url || seen.has(item.url)) return;
    seen.add(item.url);
    items.push(item);
  };

  for (const attachment of attachments) {
    if (!isSketchScene(attachment)) add(attachmentItem(attachment));
  }
  for (const attachment of stagedAttachments) {
    if (!isSketchScene(attachment)) {
      add(attachmentItem(attachment, "Unsent attachment"));
    }
  }
  for (const artifact of artifacts) {
    for (const item of artifactItems(artifact)) add(item);
  }
  return sortLightboxItems(items);
}

export function sortLightboxItems(items: LightboxItem[]): LightboxItem[] {
  return [...items].sort((a, b) => {
    if (a.createdAt === b.createdAt) return 0;
    if (a.createdAt === undefined) return 1;
    if (b.createdAt === undefined) return -1;
    return a.createdAt - b.createdAt;
  });
}

export function findLightboxItemIndex(
  items: LightboxItem[],
  target: LightboxItem,
): number {
  const byKey = items.findIndex((item) => item.key === target.key);
  if (byKey !== -1) return byKey;
  return items.findIndex((item) => item.url === target.url);
}

export function withLightboxItem(
  items: LightboxItem[],
  target: LightboxItem,
): { items: LightboxItem[]; index: number } {
  const existing = findLightboxItemIndex(items, target);
  if (existing !== -1) return { items, index: existing };
  const next = sortLightboxItems([...items, target]);
  return { items: next, index: findLightboxItemIndex(next, target) };
}

function urlFilename(url: string): string {
  try {
    const parsed = new URL(url, "http://localhost");

    if (parsed.protocol === "data:") return "";
    return decodeURIComponent(parsed.pathname.split("/").pop() ?? "");
  } catch {
    return "";
  }
}

function inlineImageSource(url: string): string {
  try {
    const parsed = new URL(url, window.location.href);

    if (
      parsed.protocol === "data:" ||
      parsed.protocol === "blob:" ||
      parsed.origin === window.location.origin
    ) {
      return "Inline image";
    }
    return parsed.host ? `Inline image · ${parsed.host}` : "Inline image";
  } catch {
    return "Inline image";
  }
}
