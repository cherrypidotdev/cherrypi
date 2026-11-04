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

import { tagColor } from "@/lib/task";

export interface TagPaletteColor {
  name: string;
  value: string;
}

export const TAG_COLORS: readonly TagPaletteColor[] = [
  { name: "Slate", value: "#64748b" },
  { name: "Red", value: "#ef4444" },
  { name: "Orange", value: "#f97316" },
  { name: "Amber", value: "#f59e0b" },
  { name: "Lime", value: "#84cc16" },
  { name: "Green", value: "#22c55e" },
  { name: "Teal", value: "#14b8a6" },
  { name: "Sky", value: "#0ea5e9" },
  { name: "Indigo", value: "#6366f1" },
  { name: "Violet", value: "#8b5cf6" },
  { name: "Pink", value: "#ec4899" },
];

export interface TagPreset {
  name: string;
  description: string;
  color: string;
}

export const TAG_PRESETS: readonly TagPreset[] = [
  {
    name: "feature",
    description: "New functionality",
    color: "#6366f1",
  },
  {
    name: "bug",
    description: "Defect or regression",
    color: "#ef4444",
  },
  {
    name: "chore",
    description: "Maintenance and housekeeping",
    color: "#64748b",
  },
  {
    name: "docs",
    description: "Documentation",
    color: "#0ea5e9",
  },
  {
    name: "refactor",
    description: "Restructuring without behaviour change",
    color: "#8b5cf6",
  },
  {
    name: "test",
    description: "Test coverage",
    color: "#84cc16",
  },
  {
    name: "perf",
    description: "Performance work",
    color: "#f59e0b",
  },
  {
    name: "design",
    description: "Design and UX",
    color: "#ec4899",
  },
  {
    name: "infra",
    description: "Build, CI and infrastructure",
    color: "#f97316",
  },
  {
    name: "question",
    description: "Needs an answer or decision",
    color: "#14b8a6",
  },
];

export function suggestTagColor(tags: readonly { color: string }[]): string {
  const used = new Set(tags.map((tag) => tagColor(tag.color)));
  const free = TAG_COLORS.find((color) => !used.has(color.value));
  if (free) return free.value;
  return TAG_COLORS[tags.length % TAG_COLORS.length]?.value ?? "#64748b";
}
