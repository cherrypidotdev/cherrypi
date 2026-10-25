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

export interface TimestampLike {
  seconds: bigint;
}

export function formatTimestamp(ts: TimestampLike | undefined): string {
  if (!ts || ts.seconds === 0n) return "—";
  return new Date(Number(ts.seconds) * 1000).toLocaleString();
}

export function formatRelativeTimestamp(ts: TimestampLike | undefined): string {
  if (!ts || ts.seconds === 0n) return "—";
  return formatRelativeTime(Number(ts.seconds) * 1000);
}

export function formatRelativeTime(ms: number, now = Date.now()): string {
  const seconds = Math.floor((now - ms) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  if (days < 30) return `${Math.floor(days / 7)}w ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.max(1, Math.floor(days / 365))}y ago`;
}

export function formatCost(cost: number): string {
  return `$${cost.toFixed(4)}`;
}

export function formatCostPrecise(cost: number): string {
  if (cost === 0) return "$0";
  if (Math.abs(cost) < 0.000001) return `$${cost.toExponential(1)}`;
  return `$${cost.toFixed(6)}`;
}

export function formatRate(tokensPerSecond: number): string {
  return `${tokensPerSecond.toFixed(1)} tok/s`;
}

export function formatDuration(ms: number | bigint): string {
  const total = Math.max(0, Math.floor(Number(ms) / 1000));
  const seconds = total % 60;
  const minutes = Math.floor(total / 60) % 60;
  const hours = Math.floor(total / 3600);
  if (hours > 0) return `${hours}h ${String(minutes).padStart(2, "0")}m`;
  if (minutes > 0) return `${minutes}m ${String(seconds).padStart(2, "0")}s`;
  return `${seconds}s`;
}

export function formatCount(n: number | bigint): string {
  return Number(n).toLocaleString();
}

export function formatTokens(n: number | bigint): string {
  const value = Number(n);
  if (value < 1000) return String(value);
  if (value < 1_000_000) return `${trimZero(value / 1000)}k`;
  return `${trimZero(value / 1_000_000)}M`;
}

function trimZero(value: number): string {
  return value.toFixed(1).replace(/\.0$/, "");
}

export function formatBytes(bytes: number | bigint): string {
  const n = Number(bytes);
  if (n < 1024) return `${n} B`;
  if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
  if (n < 1024 ** 4) return `${(n / 1024 ** 3).toFixed(1)} GB`;
  return `${(n / 1024 ** 4).toFixed(1)} TB`;
}
