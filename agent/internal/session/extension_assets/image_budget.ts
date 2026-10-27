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

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { resizeImage } from "@earendil-works/pi-coding-agent";

const MAX_WIDTH = 1568;
const MAX_HEIGHT = 1568;
const MAX_IMAGE_BYTES = 512 * 1024;
const MAX_RESULT_BYTES = 4 * 1024 * 1024;

function omitted(reason: string) {
  return {
    type: "text" as const,
    text: `[image omitted: ${reason}]`,
  };
}

export default function (pi: ExtensionAPI) {
  pi.on("tool_result", async (event) => {
    if (!event.content?.some((part) => part.type === "image")) return;

    let budget = MAX_RESULT_BYTES;
    let changed = false;
    const content = [];

    for (const part of event.content) {
      if (part.type !== "image") {
        content.push(part);
        continue;
      }
      if (budget <= 0) {
        changed = true;
        content.push(omitted("this result exceeds the request image budget"));
        continue;
      }

      const input = Buffer.from(part.data, "base64");
      const resized = await resizeImage(input, part.mimeType, {
        maxWidth: MAX_WIDTH,
        maxHeight: MAX_HEIGHT,
        maxBytes: Math.min(MAX_IMAGE_BYTES, budget),
        jpegQuality: 75,
      });
      if (!resized) {
        changed = true;
        content.push(omitted("it could not be encoded within the budget"));
        continue;
      }

      const bytes = Buffer.byteLength(resized.data, "base64");
      budget -= bytes;
      changed = true;
      content.push({
        type: "image" as const,
        data: resized.data,
        mimeType: resized.mimeType,
      });
    }

    if (!changed) return;
    return { content };
  });
}
