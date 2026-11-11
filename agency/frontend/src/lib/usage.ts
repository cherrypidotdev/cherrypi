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

  thinkingMs: number;

  modelWaitMs: number;

  toolMs: number;

  ttcMs: number;

  ttcSamples: number;
}

export interface UsageTiming {
  activeMs: number;
  toolMs: number;
  modelWaitMs: number;
}

export function thinkingMs(timing: Pick<UsageTiming, "activeMs" | "toolMs">) {
  return Math.max(0, timing.activeMs - timing.toolMs);
}

export function generationRate(
  outputTokens: number,
  thinking: number,
  modelWaitMs: number,
): number {
  const seconds = (thinking - modelWaitMs) / 1000;
  if (seconds <= 0 || outputTokens <= 0) return 0;
  return outputTokens / seconds;
}

export interface UsageInsights {
  tasks: number;

  ttcSamples: number;

  ttcMs: number;

  activeMs: number;

  rounds: number;

  tokens: number;

  cost: number;

  thinkingMs: number;

  modelWaitMs: number;

  tokensPerSecond: number;
}

export interface UsageInsightDay extends UsageInsights {
  date: Date;
}

export interface TaskInsightSample {
  queuedAtMs: number;

  settledAtMs: number;

  ttcMs?: number;
  inputTokens?: number;
  outputTokens?: number;
  cost?: number;
  activeMs?: number;
  runs?: number;
  toolMs?: number;
  modelWaitMs?: number;
}

export function taskTtcMs(
  queuedAtMs: number,
  settledAtMs: number,
): number | null {
  if (!queuedAtMs || !settledAtMs || settledAtMs < queuedAtMs) return null;
  return settledAtMs - queuedAtMs;
}

export function sampleTtcMs(sample: TaskInsightSample): number | null {
  if (sample.ttcMs && sample.ttcMs > 0) return sample.ttcMs;
  return taskTtcMs(sample.queuedAtMs, sample.settledAtMs);
}

export function summarizeTaskInsights(
  samples: TaskInsightSample[],
): UsageInsights {
  let ttcTotal = 0;
  let ttcSamples = 0;
  let tokens = 0;
  let cost = 0;
  let activeMs = 0;
  let runs = 0;
  let thinking = 0;
  let modelWait = 0;
  let generation = 0;
  let generationSamples = 0;

  for (const sample of samples) {
    const ttc = sampleTtcMs(sample);
    if (ttc !== null) {
      ttcTotal += ttc;
      ttcSamples++;
    }
    const timing = {
      activeMs: sample.activeMs ?? 0,
      toolMs: sample.toolMs ?? 0,
      modelWaitMs: sample.modelWaitMs ?? 0,
    };
    tokens += (sample.inputTokens ?? 0) + (sample.outputTokens ?? 0);
    cost += sample.cost ?? 0;
    activeMs += timing.activeMs;
    runs += sample.runs ?? 0;
    thinking += thinkingMs(timing);
    modelWait += timing.modelWaitMs;

    const rate = generationRate(
      sample.outputTokens ?? 0,
      thinkingMs(timing),
      timing.modelWaitMs,
    );
    if (rate > 0) {
      generation += rate;
      generationSamples++;
    }
  }

  const tasks = samples.length;
  const average = (total: number) => (tasks ? total / tasks : 0);

  return {
    tasks,
    ttcSamples,
    ttcMs: ttcSamples ? ttcTotal / ttcSamples : 0,
    activeMs: average(activeMs),
    rounds: average(runs),
    tokens: average(tokens),
    cost: average(cost),
    thinkingMs: average(thinking),
    modelWaitMs: average(modelWait),
    tokensPerSecond: generationSamples ? generation / generationSamples : 0,
  };
}

export interface LiveUsageTiming {
  activeMs: number;
  thinkingMs: number;
  modelWaitMs: number;
}

export function tickUsage(
  base: Pick<LiveUsageTiming, "activeMs" | "thinkingMs" | "modelWaitMs">,
  elapsedMs: number,
  live: { toolRunning: boolean; modelWaiting: boolean },
): LiveUsageTiming {
  const elapsed = Math.max(0, elapsedMs);
  return {
    activeMs: base.activeMs + elapsed,
    thinkingMs: base.thinkingMs + (live.toolRunning ? 0 : elapsed),
    modelWaitMs: base.modelWaitMs + (live.modelWaiting ? elapsed : 0),
  };
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

export function usageRangeKey(range: UsageRange): string {
  return range === "all" ? "usage.range.allTimeLower" : "usage.range.lastDays";
}

export function usageRangePillKey(range: UsageRange): string {
  return range === "all" ? "usage.range.allTime" : "usage.range.lastDaysPill";
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
