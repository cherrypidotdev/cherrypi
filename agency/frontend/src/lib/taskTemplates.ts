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
  TaskPriority,
  TaskSize,
  TaskTemplateArgType,
  type TaskTemplateArgInput,
  type TaskTemplateContent,
  type TaskTemplateContentInput,
} from "@cherrypi/sdk";

export function emptyTemplateContent(): TaskTemplateContentInput {
  return {
    body: "",
    titleTemplate: "",
    args: [],
    defaultPriority: TaskPriority.NORMAL,
    defaultSize: TaskSize.MEDIUM,
    defaultTagIds: [],
    defaultTeamIds: [],
    defaultLeadActorIds: [],
  };
}

export function templateContentInput(
  content?: TaskTemplateContent,
): TaskTemplateContentInput {
  if (!content) return emptyTemplateContent();
  return {
    body: content.body,
    titleTemplate: content.titleTemplate,
    args: content.args.map((arg) => ({
      key: arg.key,
      label: arg.label,
      description: arg.description,
      type: arg.type,
      required: arg.required,
      defaultValue: arg.defaultValue,
      options: arg.options.map((option) => ({
        value: option.value,
        label: option.label,
      })),
    })),
    defaultPriority: content.defaultPriority,
    defaultSize: content.defaultSize,
    defaultTagIds: [...content.defaultTagIds],
    defaultTeamIds: [...content.defaultTeamIds],
    defaultLeadActorIds: [...content.defaultLeadActorIds],
  };
}

export function defaultTemplateArgs(
  content: TaskTemplateContentInput,
): Record<string, unknown> {
  const args: Record<string, unknown> = {};
  for (const arg of content.args ?? []) {
    if (!arg.key) continue;
    args[arg.key] = defaultArgValue(arg);
  }
  return args;
}

function defaultArgValue(arg: TaskTemplateArgInput): unknown {
  const raw = (arg.defaultValue ?? "").trim();
  switch (arg.type) {
    case TaskTemplateArgType.NUMBER:
      return raw === "" ? 0 : Number(raw);
    case TaskTemplateArgType.BOOLEAN:
      return ["true", "1", "yes", "on"].includes(raw.toLowerCase());
    case TaskTemplateArgType.SELECT:
      return raw || (arg.options?.[0]?.value ?? "");
    default:
      return raw;
  }
}

export const ARG_TYPE_LABELS: Record<TaskTemplateArgType, string> = {
  [TaskTemplateArgType.UNSPECIFIED]: "Text",
  [TaskTemplateArgType.TEXT]: "Text",
  [TaskTemplateArgType.MULTILINE]: "Multiline text",
  [TaskTemplateArgType.NUMBER]: "Number",
  [TaskTemplateArgType.BOOLEAN]: "Boolean",
  [TaskTemplateArgType.SELECT]: "Select",
};

export const ARG_TYPES: TaskTemplateArgType[] = [
  TaskTemplateArgType.TEXT,
  TaskTemplateArgType.MULTILINE,
  TaskTemplateArgType.NUMBER,
  TaskTemplateArgType.BOOLEAN,
  TaskTemplateArgType.SELECT,
];

export function parseOptions(raw: string): { value: string; label: string }[] {
  return raw
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean)
    .map((value) => ({ value, label: value }));
}

export function formatOptions(options?: { value: string }[]): string {
  return (options ?? []).map((option) => option.value).join(", ");
}

/** A compact one-line summary of argument values for a badge/tooltip. */
export function argsSummary(argsJSON: string): string {
  const parsed = parseArgsJSON(argsJSON);
  const entries = Object.entries(parsed);
  if (!entries.length) return "no arguments";
  return entries
    .map(([key, value]) => `${key}: ${formatArgValue(value)}`)
    .join(" · ");
}

export function parseArgsJSON(argsJSON: string): Record<string, unknown> {
  if (!argsJSON) return {};
  try {
    const parsed = JSON.parse(argsJSON);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

export function formatArgValue(value: unknown): string {
  if (typeof value === "string") return value || "—";
  if (value === null || value === undefined) return "—";
  return String(value);
}
