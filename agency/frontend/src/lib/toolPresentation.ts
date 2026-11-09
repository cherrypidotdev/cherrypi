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

import type { ActivityItem, ToolItem } from "./console";

/**
 * How the console presents a tool call: a one-line summary (an icon, a verb and
 * what the call acts on) plus the bodies its expanded row shows. Pi's built-in
 * `read`, `write`, `edit` and `bash` calls carry JSON arguments and plain-text
 * results, so each one is read into that shape here; every other tool falls
 * back to its raw arguments and result.
 */

export type ToolIcon = "bash" | "read" | "edit" | "write" | "generic";

export interface ChangedFile {
  path: string;
  added: number;
  removed: number;
}

export interface ToolSection {
  label: string;
  text: string;

  kind: "command" | "text" | "diff";

  source: "args" | "result";
}

export interface ToolView {
  icon: ToolIcon;

  verb: string;

  target: string;

  changes: ChangedFile[];

  sections: ToolSection[];
}

export function parseObject(raw: string): Record<string, unknown> | null {
  if (!raw.trim()) return null;
  try {
    const value: unknown = JSON.parse(raw);
    return typeof value === "object" && value !== null && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : null;
  } catch {
    return null;
  }
}

function text(value: unknown): string {
  return typeof value === "string" ? value : "";
}

export function lineCount(content: string): number {
  if (!content) return 0;
  const lines = content.split("\n");
  return content.endsWith("\n") ? lines.length - 1 : lines.length;
}

export function visualLines(text: string, width = 80): number {
  let rows = 0;
  for (const line of text.split("\n")) {
    rows += Math.max(1, Math.ceil(line.length / width));
  }
  return rows;
}

export function diffStat(diff: string): { added: number; removed: number } {
  let added = 0;
  let removed = 0;
  for (const line of diff.split("\n")) {
    if (line.startsWith("+++") || line.startsWith("---")) continue;
    if (line.startsWith("+")) added += 1;
    else if (line.startsWith("-")) removed += 1;
  }
  return { added, removed };
}

function editCount(args: Record<string, unknown> | null): number {
  const edits = args?.["edits"];
  return Array.isArray(edits) ? edits.length : 0;
}

function editArgsStat(args: Record<string, unknown> | null): {
  added: number;
  removed: number;
} {
  const edits = args?.["edits"];
  if (!Array.isArray(edits)) return { added: 0, removed: 0 };
  let added = 0;
  let removed = 0;
  for (const edit of edits) {
    if (typeof edit !== "object" || edit === null) continue;
    const fields = edit as Record<string, unknown>;
    added += lineCount(text(fields["newText"]));
    removed += lineCount(text(fields["oldText"]));
  }
  return { added, removed };
}

/** The files a single call changed, with their line counts. */
export function toolChanges(
  item: Pick<ToolItem, "name" | "args" | "details">,
): ChangedFile[] {
  const args = parseObject(item.args);
  const path = text(args?.["path"]);
  if (!path) return [];
  if (item.name === "write") {
    return [{ path, added: lineCount(text(args?.["content"])), removed: 0 }];
  }
  if (item.name === "edit") {
    const diff = text(parseObject(item.details)?.["diff"]);
    const stat = diff ? diffStat(diff) : editArgsStat(args);
    return [{ path, ...stat }];
  }
  return [];
}

export function displayPath(path: string): string {
  const repo = path.lastIndexOf("/repo/");
  if (repo !== -1) return path.slice(repo + "/repo/".length);
  const parts = path.split("/").filter(Boolean);
  if (!path.startsWith("/") || parts.length <= 3) return path;
  return `…/${parts.slice(-3).join("/")}`;
}

function editVerb(count: number): string {
  if (count <= 0) return "edit";
  return count === 1 ? "1 edit" : `${count} edits`;
}

function section(
  label: string,
  text: string,
  kind: ToolSection["kind"],
  source: ToolSection["source"],
): ToolSection | null {
  return text.trim() === "" ? null : { label, text, kind, source };
}

function sections(...entries: (ToolSection | null)[]): ToolSection[] {
  return entries.filter((entry): entry is ToolSection => entry !== null);
}

export function toolView(
  item: Pick<ToolItem, "name" | "args" | "result" | "details">,
): ToolView {
  const args = parseObject(item.args);
  const details = parseObject(item.details);
  const changes = toolChanges(item);

  switch (item.name) {
    case "bash": {
      const command = text(args?.["command"]);
      return {
        icon: "bash",
        verb: "",
        target: command || "bash",
        changes,
        sections: sections(
          section("Command", command, "command", "args"),
          section("Output", item.result, "text", "result"),
        ),
      };
    }
    case "read": {
      const path = text(args?.["path"]);
      return {
        icon: "read",
        verb: readRange(args),
        target: path || "read",
        changes,
        sections: sections(section("Content", item.result, "text", "result")),
      };
    }
    case "write": {
      const path = text(args?.["path"]);
      return {
        icon: "write",
        verb: "Write",
        target: path || "write",
        changes,
        sections: sections(
          section("Content", text(args?.["content"]), "text", "args"),
        ),
      };
    }
    case "edit": {
      const path = text(args?.["path"]);
      return {
        icon: "edit",
        verb: editVerb(editCount(args)),
        target: path || "edit",
        changes,
        sections: sections(
          section("Diff", text(details?.["diff"]), "diff", "result"),
        ),
      };
    }
    default:
      return {
        icon: "generic",
        verb: "",
        target: item.name,
        changes,
        sections: sections(
          section(
            "Arguments",
            item.args ? prettyJson(item.args) : "",
            "text",
            "args",
          ),
          section(
            "Result",
            item.result.trim() ? prettyJson(item.result) : "",
            "text",
            "result",
          ),
        ),
      };
  }
}

export function prettyJson(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed || (!trimmed.startsWith("{") && !trimmed.startsWith("["))) {
    return raw;
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return raw;
  }
}

/**
 * The files an activity group changed, in the order their calls first touched
 * them, with the line counts of every call on the same file added up. An edit
 * that only replaced lines still counts the lines it rewrote, so a file's
 * badge reads as "how much of this file this block touched".
 */
export function groupChangedFiles(items: ActivityItem[]): ChangedFile[] {
  const byPath = new Map<string, ChangedFile>();
  for (const item of items) {
    if (item.kind !== "tool") continue;
    for (const change of toolChanges(item)) {
      const existing = byPath.get(change.path);
      if (existing) {
        existing.added += change.added;
        existing.removed += change.removed;
      } else {
        byPath.set(change.path, { ...change });
      }
    }
  }
  return [...byPath.values()];
}

/** The line range a `read` call asked for, as a row label; empty for a whole
 * file. */
function readRange(args: Record<string, unknown> | null): string {
  const offset = args?.["offset"];
  const limit = args?.["limit"];
  if (typeof offset !== "number" && typeof limit !== "number") return "";
  const start = typeof offset === "number" ? offset : 1;
  if (typeof limit !== "number") return `from line ${start}`;
  return `lines ${start}–${start + limit - 1}`;
}
