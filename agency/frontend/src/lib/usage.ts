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

export interface UsageSummary {
  cost: number;
  inputTokens: number;
  outputTokens: number;
  activeMs: number;
  runs: number;
  tasks: number;
  completed: number;
  archived: number;
  allTimeCompleted: number;
}

export interface UsageQuery {
  agentId?: string;
  projectId?: string;
  sinceMs?: number;
  untilMs?: number;
}

export const USAGE_RANGES = ["7", "30", "90", "all"] as const;

export type UsageRange = (typeof USAGE_RANGES)[number];

export const DAY_MS = 86_400_000;

export const USAGE_RANGE_OPTIONS: { value: UsageRange; label: string }[] = [
  { value: "7", label: "7d" },
  { value: "30", label: "30d" },
  { value: "90", label: "90d" },
  { value: "all", label: "All" },
];

export function usageRangeLabel(range: UsageRange): string {
  return range === "all" ? "all time" : `last ${range} days`;
}

export function usageRangePill(range: UsageRange): string {
  return range === "all" ? "All time" : `Last ${range} days`;
}

export function utcDayStart(ms: number): number {
  return Math.floor(ms / DAY_MS) * DAY_MS;
}

export function usageRangeBounds(
  range: UsageRange,
  now = Date.now(),
): { sinceMs: number; untilMs: number } {
  if (range === "all") return { sinceMs: 0, untilMs: 0 };
  return {
    sinceMs: utcDayStart(now) - (Number(range) - 1) * DAY_MS,
    untilMs: now,
  };
}

export const USAGE_COLORS = {
  cost: "#f97316",
  input: "#3b82f6",
  output: "#a855f7",
  runs: "#3b82f6",
  completed: "#14b8a6",
} as const;

export function usageAccent(id: string): string {
  const palette = ["#f97316", "#6366f1", "#0d9488", "#db2777", "#eab308"];
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = (hash * 31 + id.charCodeAt(i)) | 0;
  return palette[Math.abs(hash) % palette.length] ?? palette[0]!;
}
