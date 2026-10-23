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

import { Type } from "typebox";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/**
 * CherryPi task links.
 *
 * Registers `setTaskUrls`, the tool a running task uses to publish links (a
 * PR, an issue, a deployment) on itself. Each call records the task's complete
 * link list as a custom Pi session entry; the CherryPi agent forwards that
 * entry to the agency, which renders the links on the task everywhere.
 */
export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "setTaskUrls",
    label: "Set Task URLs",
    description:
      "Replace the links attached to this CherryPi task. Pass the complete list on every call, and pass an empty list to clear them. Use it to surface a pull request, issue, deployment, or any other URL; CherryPi shows the links on the task in the task list and detail view.",
    parameters: Type.Object({
      urls: Type.Array(
        Type.Object({
          url: Type.String({ description: "Absolute http(s) URL." }),
          label: Type.String({
            description:
              'Short label shown next to the URL, e.g. "PR" or "Issue #42".',
          }),
        }),
        { description: "The complete list of links for this task." },
      ),
    }),
    execute: async (_toolCallId, params) => {
      const urls = (params.urls ?? [])
        .map((u) => ({
          url: String(u.url ?? "").trim(),
          label: String(u.label ?? "").trim(),
        }))
        .filter((u) => u.url.length > 0);

      pi.appendEntry("cherrypi_task_urls", { urls });

      const text =
        urls.length === 0
          ? "Cleared this task's links."
          : `Set ${urls.length} link${urls.length === 1 ? "" : "s"}: ` +
            urls
              .map((u) => (u.label ? `${u.label} (${u.url})` : u.url))
              .join(", ");

      return { content: [{ type: "text", text }], details: undefined };
    },
  });
}
