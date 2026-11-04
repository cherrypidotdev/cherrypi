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

import { formatBytes } from "@/lib/format";
import type { AgentDiskUsage, AgentHealthSample } from "@cherrypi/sdk";

export const HEALTH_RANGES = [
  "5m",
  "15m",
  "1h",
  "6h",
  "24h",
  "7d",
  "30d",
] as const;
export type HealthRange = (typeof HEALTH_RANGES)[number];

export const HEALTH_RANGE_OPTIONS: { value: HealthRange; label: string }[] = [
  { value: "5m", label: "5m" },
  { value: "15m", label: "15m" },
  { value: "1h", label: "1h" },
  { value: "6h", label: "6h" },
  { value: "24h", label: "24h" },
  { value: "7d", label: "7d" },
  { value: "30d", label: "30d" },
];

const RANGE_MS: Record<HealthRange, number> = {
  "5m": 300_000,
  "15m": 900_000,
  "1h": 3_600_000,
  "6h": 21_600_000,
  "24h": 86_400_000,
  "7d": 604_800_000,
  "30d": 2_592_000_000,
};

export function healthRangeMs(range: HealthRange): number {
  return RANGE_MS[range];
}

const RANGE_LABELS: Partial<Record<HealthRange, string>> = {
  "5m": "last 5 minutes",
  "15m": "last 15 minutes",
  "24h": "last 24 hours",
};

export function healthRangeLabel(range: HealthRange): string {
  return RANGE_LABELS[range] ?? `last ${range}`;
}

export function healthRangeBounds(
  range: HealthRange,
  now = Date.now(),
): { sinceMs: number; untilMs: number } {
  return { sinceMs: now - RANGE_MS[range], untilMs: now };
}

export const HEALTH_MAX_LIVE_POINTS = 1440;

export function mergeHealthSample(
  existing: AgentHealthSample[],
  sample: AgentHealthSample,
  range: HealthRange,
  now = Date.now(),
): AgentHealthSample[] {
  const cutoff = now - RANGE_MS[range];
  const sampleAt = Number(sample.sampledAt);
  if (sampleAt < cutoff) return existing;
  const byTime = new Map<number, AgentHealthSample>();
  for (const item of existing) {
    const at = Number(item.sampledAt);
    if (at >= cutoff) byTime.set(at, item);
  }
  byTime.set(sampleAt, sample);
  const merged = [...byTime.values()].sort(
    (a, b) => Number(a.sampledAt) - Number(b.sampledAt),
  );
  return capHealthSamples(merged);
}

export function capHealthSamples(
  samples: AgentHealthSample[],
  maxPoints = HEALTH_MAX_LIVE_POINTS,
): AgentHealthSample[] {
  if (samples.length <= maxPoints || maxPoints <= 2) return samples;
  const step = (samples.length - 1) / (maxPoints - 1);
  const out: AgentHealthSample[] = [];
  let last = -1;
  for (let i = 0; i < maxPoints; i += 1) {
    const index = Math.round(i * step);
    if (index === last) continue;
    const sample = samples[index];
    if (!sample) continue;
    out.push(sample);
    last = index;
  }
  return out;
}

export function niceCeil(value: number): number {
  if (!(value > 0) || !Number.isFinite(value)) return 1;
  const exponent = Math.floor(Math.log10(value));
  const base = 10 ** exponent;
  const fraction = value / base;
  const step = fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 5 ? 5 : 10;
  return step * base;
}

export const HEALTH_INTERVAL_MIN_SECONDS = 10;
export const HEALTH_INTERVAL_MAX_SECONDS = 86_400;

export const HEALTH_INTERVAL_PRESETS: { seconds: number; label: string }[] = [
  { seconds: 30, label: "30 seconds" },
  { seconds: 60, label: "1 minute" },
  { seconds: 300, label: "5 minutes" },
  { seconds: 900, label: "15 minutes" },
  { seconds: 1800, label: "30 minutes" },
  { seconds: 3600, label: "1 hour" },
];

export function parseHealthIntervalSeconds(raw: string): number | undefined {
  const text = raw.trim().toLowerCase();
  if (!text) return undefined;
  if (/^\d+$/.test(text)) return Number(text);

  let total = 0;
  let consumed = "";
  const parts = /(\d+)\s*([hms])/g;
  for (const match of text.matchAll(parts)) {
    const value = Number(match[1]);
    switch (match[2]) {
      case "h":
        total += value * 3600;
        break;
      case "m":
        total += value * 60;
        break;
      default:
        total += value;
    }
    consumed += match[0];
  }
  if (consumed !== text) return undefined;
  return total;
}

export function formatHealthInterval(seconds: number): string {
  if (seconds <= 0) return "off";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const rest = seconds % 60;
  return `${hours ? `${hours}h` : ""}${minutes ? `${minutes}m` : ""}${
    rest || (!hours && !minutes) ? `${rest}s` : ""
  }`;
}

export const HEALTH_COLORS = {
  cpu: "#f97316",
  memory: "#3b82f6",
  disk: "#14b8a6",
  load: "#a855f7",
  network: "#eab308",
  gpu: "#ec4899",
} as const;

export type HealthTone = "ok" | "warn" | "critical";

export function healthTone(
  percent: number,
  warn = 75,
  critical = 90,
): HealthTone {
  if (percent >= critical) return "critical";
  if (percent >= warn) return "warn";
  return "ok";
}

export function healthToneText(tone: HealthTone): string {
  switch (tone) {
    case "critical":
      return "text-red-600 dark:text-red-400";
    case "warn":
      return "text-amber-600 dark:text-amber-400";
    default:
      return "text-emerald-600 dark:text-emerald-400";
  }
}

export function healthToneBar(tone: HealthTone): string {
  switch (tone) {
    case "critical":
      return "bg-red-500";
    case "warn":
      return "bg-amber-500";
    default:
      return "bg-emerald-500";
  }
}

export function fullestDisk(
  disks: AgentDiskUsage[] | undefined,
): AgentDiskUsage | undefined {
  let worst: AgentDiskUsage | undefined;
  for (const disk of disks ?? []) {
    if (!worst || disk.usedPercent > worst.usedPercent) worst = disk;
  }
  return worst;
}

export function busiestGpu(sample: AgentHealthSample | undefined) {
  let worst = sample?.gpus[0];
  for (const gpu of sample?.gpus ?? []) {
    if (!worst || gpu.utilizationPercent > worst.utilizationPercent)
      worst = gpu;
  }
  return worst;
}

export function formatBytesPerSecond(bytes: number): string {
  return `${formatBytes(Math.max(0, bytes))}/s`;
}

export function formatUptime(seconds: number): string {
  if (seconds <= 0) return "—";
  const days = Math.floor(seconds / 86_400);
  const hours = Math.floor((seconds % 86_400) / 3_600);
  const minutes = Math.floor((seconds % 3_600) / 60);
  if (days > 0) return `${days}d ${String(hours).padStart(2, "0")}h`;
  if (hours > 0) return `${hours}h ${String(minutes).padStart(2, "0")}m`;
  return `${minutes}m`;
}

export function sampleAgeLabel(sampledAtMs: number | bigint): string {
  const ms = Number(sampledAtMs);
  if (!ms) return "never";
  return new Date(ms).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}
