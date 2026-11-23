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

import type { TaskPriority, TaskSize } from "@cherrypi/sdk";

export const THINKING_LEVELS = [
  "off",
  "minimal",
  "low",
  "medium",
  "high",
  "xhigh",
  "max",
] as const;

export interface TaskRoutingContext {
  projectId: string;
  priority: TaskPriority;
  size: TaskSize;

  tags?: string[];
}

export interface ModelLike {
  provider?: string;

  name?: string;

  modelName?: string;

  id?: string;

  modelId?: string;
}

export function modelName(model: ModelLike): string {
  return model.name || model.modelName || model.id || model.modelId || "";
}

export function modelLabel(model: ModelLike): string {
  const name = modelName(model);
  return model.provider ? `${model.provider}/${name}` : name;
}

export function sortThinkingLevels(levels: readonly string[]): string[] {
  const declared = new Set(levels);
  return THINKING_LEVELS.filter((level) => declared.has(level));
}

const THINKING_LEVEL_LABELS: Record<string, string> = {
  off: "Off",
  minimal: "Minimal",
  low: "Low",
  medium: "Medium",
  high: "High",
  xhigh: "XHigh",
  max: "Max",
};

export function thinkingLevelLabel(level: string): string {
  return THINKING_LEVEL_LABELS[level] ?? level;
}
