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

import { t } from "@/i18n/translate";
import type { ConcurrencyKpis } from "@/lib/concurrencyKpis";
import { formatCount, formatDuration, formatTokens } from "@/lib/format";
import {
  USAGE_RANGES,
  usageRangePillKey,
  type UsageInsightDay,
  type UsageInsights,
  type UsageRange,
  type UsageSummary,
} from "@/lib/usage";

export const KPI_GRID_COLUMNS = 4;
export const KPI_GRID_ROWS = 2;

export const KPI_MAX_CELLS = KPI_GRID_COLUMNS * KPI_GRID_ROWS;

export type KpiMetric =
  | "cost"
  | "tokens"
  | "rounds"
  | "completed"
  | "thinking"
  | "active"
  | "ttc"
  | "slotsHeld"
  | "waiting"
  | "capacity"
  | "capsFull";

export type KpiAggregation = "total" | "perTask" | "snapshot";

export type KpiGroup = "usage" | "concurrency";

export interface KpiWidgetDefinition {
  kind: string;
  group: KpiGroup;
  metric: KpiMetric;
  aggregation: KpiAggregation;

  labelKey: string;

  hintKey: string;

  projectLabel?: string;
  projectHint?: string;

  color: string;

  size: { width: number; height: number };

  ranged: boolean;

  scoped: boolean;
}

export const KPI_WIDGETS: KpiWidgetDefinition[] = [
  {
    kind: "usage.cost.total",
    group: "usage",
    metric: "cost",
    aggregation: "total",
    labelKey: "kpiWidget.totalCost",
    hintKey: "kpiHint.period",
    color: "var(--chart-1)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.tokens.total",
    group: "usage",
    metric: "tokens",
    aggregation: "total",
    labelKey: "kpiWidget.tokens",
    hintKey: "kpiHint.inOut",
    color: "var(--chart-2)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.rounds.total",
    group: "usage",
    metric: "rounds",
    aggregation: "total",
    labelKey: "kpiWidget.responseRounds",
    hintKey: "kpiHint.agentTurns",
    color: "var(--chart-3)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.thinking.total",
    group: "usage",
    metric: "thinking",
    aggregation: "total",
    labelKey: "kpiWidget.thinkingTime",
    hintKey: "kpiHint.modelWork",
    color: "var(--chart-2)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.active.total",
    group: "usage",
    metric: "active",
    aggregation: "total",
    labelKey: "kpiWidget.activeTime",
    hintKey: "kpiHint.agentWork",
    color: "var(--chart-4)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.ttc.total",
    group: "usage",
    metric: "ttc",
    aggregation: "total",
    labelKey: "kpiWidget.ttcTotal",
    hintKey: "kpiHint.outsideBacklog",
    color: "var(--chart-5)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.completed.total",
    group: "usage",
    metric: "completed",
    aggregation: "total",
    labelKey: "kpiWidget.completedTasks",
    hintKey: "kpiHint.inclArchived",
    color: "var(--chart-3)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.cost.per-task",
    group: "usage",
    metric: "cost",
    aggregation: "perTask",
    labelKey: "kpiWidget.costPerTask",
    hintKey: "kpiHint.average",
    color: "var(--chart-1)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.tokens.per-task",
    group: "usage",
    metric: "tokens",
    aggregation: "perTask",
    labelKey: "kpiWidget.tokensPerTask",
    hintKey: "kpiHint.average",
    color: "var(--chart-2)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.rounds.per-task",
    group: "usage",
    metric: "rounds",
    aggregation: "perTask",
    labelKey: "kpiWidget.roundsPerTask",
    hintKey: "kpiHint.average",
    color: "var(--chart-3)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.thinking.per-task",
    group: "usage",
    metric: "thinking",
    aggregation: "perTask",
    labelKey: "kpiWidget.thinkingPerTask",
    hintKey: "kpiHint.average",
    color: "var(--chart-2)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.active.per-task",
    group: "usage",
    metric: "active",
    aggregation: "perTask",
    labelKey: "kpiWidget.activePerTask",
    hintKey: "kpiHint.average",
    color: "var(--chart-4)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "usage.ttc.per-task",
    group: "usage",
    metric: "ttc",
    aggregation: "perTask",
    labelKey: "kpiWidget.ttcPerTask",
    hintKey: "kpiHint.average",
    color: "var(--chart-5)",
    size: { width: 1, height: 1 },
    ranged: true,
    scoped: true,
  },
  {
    kind: "concurrency.slots-held",
    group: "concurrency",
    metric: "slotsHeld",
    aggregation: "snapshot",
    labelKey: "kpiWidget.slotsHeld",
    hintKey: "kpiHint.thisAgent",
    projectLabel: "Held here",
    projectHint: "project slots",
    color: "var(--chart-1)",
    size: { width: 1, height: 1 },
    ranged: false,
    scoped: true,
  },
  {
    kind: "concurrency.waiting",
    group: "concurrency",
    metric: "waiting",
    aggregation: "snapshot",
    labelKey: "kpiWidget.waiting",
    hintKey: "kpiHint.pinned",
    projectLabel: "Waiting",
    projectHint: "queued",
    color: "var(--chart-3)",
    size: { width: 1, height: 1 },
    ranged: false,
    scoped: true,
  },
  {
    kind: "concurrency.capacity",
    group: "concurrency",
    metric: "capacity",
    aggregation: "snapshot",
    labelKey: "kpiWidget.freeSlots",
    hintKey: "kpiHint.available",
    projectLabel: "Member agents",
    projectHint: "serving",
    color: "var(--chart-4)",
    size: { width: 1, height: 1 },
    ranged: false,
    scoped: true,
  },
  {
    kind: "concurrency.caps-full",
    group: "concurrency",
    metric: "capsFull",
    aggregation: "snapshot",
    labelKey: "kpiWidget.capsFull",
    hintKey: "kpiHint.dimensions",
    projectLabel: "Caps full",
    projectHint: "agents",
    color: "var(--chart-5)",
    size: { width: 1, height: 1 },
    ranged: false,
    scoped: true,
  },
];

const DEFINITIONS = new Map(KPI_WIDGETS.map((item) => [item.kind, item]));

export function kpiWidgetDefinition(
  kind: string,
): KpiWidgetDefinition | undefined {
  return DEFINITIONS.get(kind);
}

export const KPI_WIDGET_MENU: {
  labelKey: string;
  widgets: KpiWidgetDefinition[];
}[] = [
  {
    labelKey: "kpiMenu.usageTotals",
    widgets: KPI_WIDGETS.filter(
      (item) => item.group === "usage" && item.aggregation === "total",
    ),
  },
  {
    labelKey: "kpiMenu.usagePerTask",
    widgets: KPI_WIDGETS.filter(
      (item) => item.group === "usage" && item.aggregation === "perTask",
    ),
  },
  {
    labelKey: "kpiMenu.concurrency",
    widgets: KPI_WIDGETS.filter((item) => item.group === "concurrency"),
  },
];

export interface KpiWidgetState {
  id: string;
  kind: string;
  width: number;
  height: number;

  range: string;

  scope: string;
}

export interface KpiWidgetWire {
  id: string;
  kind: string;
  width: number;
  height: number;
  range: string;
  scope: string;
}

export function toKpiWidgetState(widget: {
  id: string;
  kind: string;
  width: number;
  height: number;
  range?: string;
  scope?: string;
}): KpiWidgetState {
  return {
    id: widget.id,
    kind: widget.kind,
    width: clamp(widget.width, 1, KPI_GRID_COLUMNS),
    height: clamp(widget.height, 1, KPI_GRID_ROWS),
    range: widget.range ?? "",
    scope: widget.scope ?? "",
  };
}

export function toKpiWidgetWire(widget: KpiWidgetState): KpiWidgetWire {
  return {
    id: widget.id,
    kind: widget.kind,
    width: widget.width,
    height: widget.height,
    range: widget.range,
    scope: widget.scope,
  };
}

function clamp(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, Math.trunc(value)));
}

export interface KpiPlacement {
  x: number;
  y: number;
}

export function placeKpiWidgets(
  widgets: KpiWidgetState[],
): { widget: KpiWidgetState; placement: KpiPlacement }[] {
  const taken: boolean[][] = Array.from({ length: KPI_GRID_ROWS }, () =>
    Array.from({ length: KPI_GRID_COLUMNS }, () => false),
  );
  const out: { widget: KpiWidgetState; placement: KpiPlacement }[] = [];
  for (const widget of widgets) {
    const placement = firstFree(taken, widget.width, widget.height);
    if (!placement) continue;
    for (let dy = 0; dy < widget.height; dy++) {
      for (let dx = 0; dx < widget.width; dx++) {
        taken[placement.y + dy]![placement.x + dx] = true;
      }
    }
    out.push({ widget, placement });
  }
  return out;
}

export function kpiLayoutFits(widgets: KpiWidgetState[]): boolean {
  return placeKpiWidgets(widgets).length === widgets.length;
}

function firstFree(
  taken: boolean[][],
  width: number,
  height: number,
): KpiPlacement | undefined {
  for (let y = 0; y + height <= KPI_GRID_ROWS; y++) {
    for (let x = 0; x + width <= KPI_GRID_COLUMNS; x++) {
      let free = true;
      for (let dy = 0; dy < height && free; dy++) {
        for (let dx = 0; dx < width; dx++) {
          if (taken[y + dy]![x + dx]) {
            free = false;
            break;
          }
        }
      }
      if (free) return { x, y };
    }
  }
  return undefined;
}

export function kpiWidgetFits(
  widgets: KpiWidgetState[],
  id: string,
  width: number,
  height: number,
): boolean {
  return kpiLayoutFits(
    widgets.map((widget) =>
      widget.id === id ? { ...widget, width, height } : widget,
    ),
  );
}

let nextWidgetNumber = 0;

export function newKpiWidgetId(): string {
  const uuid = globalThis.crypto?.randomUUID?.();
  if (uuid) return `kpi-${uuid.slice(0, 8)}`;
  nextWidgetNumber += 1;
  return `kpi-${Date.now().toString(36)}-${nextWidgetNumber}`;
}

export function addKpiWidget(
  widgets: KpiWidgetState[],
  kind: string,
): KpiWidgetState[] {
  const definition = kpiWidgetDefinition(kind);
  if (!definition) return widgets;
  const widget: KpiWidgetState = {
    id: newKpiWidgetId(),
    kind,
    width: definition.size.width,
    height: definition.size.height,
    range: "",
    scope: "",
  };
  const next = [...widgets, widget];
  return kpiLayoutFits(next) ? next : widgets;
}

export function removeKpiWidget(
  widgets: KpiWidgetState[],
  id: string,
): KpiWidgetState[] {
  return widgets.filter((widget) => widget.id !== id);
}

export function moveKpiWidget(
  widgets: KpiWidgetState[],
  id: string,
  toIndex: number,
): KpiWidgetState[] {
  const from = widgets.findIndex((widget) => widget.id === id);
  if (from < 0) return widgets;
  const target = Math.min(Math.max(toIndex, 0), widgets.length - 1);
  if (target === from) return widgets;
  const next = [...widgets];
  const [moved] = next.splice(from, 1);
  next.splice(target, 0, moved!);
  return kpiLayoutFits(next) ? next : widgets;
}

export function updateKpiWidget(
  widgets: KpiWidgetState[],
  id: string,
  patch: Partial<Omit<KpiWidgetState, "id" | "kind">>,
): KpiWidgetState[] {
  const next = widgets.map((widget) =>
    widget.id === id ? { ...widget, ...patch } : widget,
  );
  if (patch.width !== undefined || patch.height !== undefined) {
    return kpiLayoutFits(next) ? next : widgets;
  }
  return next;
}

export interface KpiUsageData {
  summary: UsageSummary;
  insights: UsageInsights;
  insightDaily: UsageInsightDay[];
  loading: boolean;
  error: string;
}

export interface KpiConcurrencyData extends ConcurrencyKpis {
  loading: boolean;
  error: string;
}

export interface KpiWidgetContent {
  label: string;
  hint: string;
  value: string;
  detail: string;
  pill: string;
  spark: number[];
}

export function kpiMoney(value: number): string {
  if (value >= 1) return `$${value.toFixed(2)}`;
  return `$${value.toFixed(4)}`;
}

/**
 * The content of a usage card: its figure, its detail line and its sparkline.
 * A per-task card reads the average over the completed tasks in the range; a
 * total card reads the accumulated figure.
 */
export function kpiUsageContent(
  widget: KpiWidgetState,
  data: KpiUsageData,
  range: UsageRange | "",
): KpiWidgetContent {
  const definition = kpiWidgetDefinition(widget.kind);
  if (!definition) {
    return {
      label: t("kpiWidget.unknown"),
      hint: widget.kind,
      value: "—",
      detail: t("kpiWidget.unknownHint"),
      pill: "",
      spark: [],
    };
  }
  const { summary, insights, insightDaily } = data;
  const perTask = definition.aggregation === "perTask";
  const completed = insights.tasks;
  const spark = (
    key:
      | "cost"
      | "tokens"
      | "rounds"
      | "thinking"
      | "active"
      | "ttc"
      | "completed",
  ) => insightDaily.map((day) => insightDayValue(day, key, perTask)).slice(-18);
  const tokens = summary.inputTokens + summary.outputTokens;
  const pill = range ? t(usageRangePillKey(range), { days: range }) : "";

  switch (definition.metric) {
    case "cost":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: kpiMoney(perTask ? insights.cost : summary.cost),
        detail: perTask
          ? `${formatCount(completed)} completed tasks`
          : `${summary.tasks} tasks in scope`,
        pill,
        spark: spark("cost"),
      };
    case "tokens":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: formatTokens(perTask ? insights.tokens : tokens),
        detail: perTask
          ? `${formatCount(completed)} completed tasks`
          : `${formatTokens(summary.inputTokens)} in · ${formatTokens(summary.outputTokens)} out`,
        pill,
        spark: spark("tokens"),
      };
    case "rounds":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: formatCount(perTask ? insights.rounds : summary.runs),
        detail: perTask
          ? `${formatCount(completed)} completed tasks`
          : `${summary.completed} completed · ${summary.tasks} tasks`,
        pill,
        spark: spark("rounds"),
      };
    case "completed":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: formatCount(summary.completed),
        detail: `${summary.archived} archived · ${summary.allTimeCompleted} all time`,
        pill,
        spark: spark("completed"),
      };
    case "thinking":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: formatDuration(
          perTask ? insights.thinkingMs : summary.thinkingMs,
        ),
        detail: perTask
          ? `${formatCount(completed)} completed tasks`
          : summary.runs
            ? `${formatDuration(summary.thinkingMs / summary.runs)} avg / round`
            : "no rounds",
        pill,
        spark: spark("thinking"),
      };
    case "active":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: formatDuration(perTask ? insights.activeMs : summary.activeMs),
        detail: perTask
          ? `${formatCount(completed)} completed tasks`
          : summary.runs
            ? `${formatDuration(summary.activeMs / summary.runs)} avg / round`
            : "no rounds",
        pill,
        spark: spark("active"),
      };
    case "ttc":
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: perTask
          ? insights.ttcSamples
            ? formatDuration(insights.ttcMs)
            : "—"
          : summary.ttcSamples
            ? formatDuration(summary.ttcMs)
            : "—",
        detail: perTask
          ? `${formatCount(insights.ttcSamples)} of ${formatCount(completed)} timed`
          : summary.ttcSamples
            ? `across ${formatCount(summary.ttcSamples)} completed tasks`
            : "no completed tasks in range",
        pill,
        spark: spark("ttc"),
      };
    default:
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: "—",
        detail: "",
        pill,
        spark: [],
      };
  }
}

function insightDayValue(
  day: UsageInsightDay,
  key:
    | "cost"
    | "tokens"
    | "rounds"
    | "thinking"
    | "active"
    | "ttc"
    | "completed",
  perTask: boolean,
): number {
  const scale = perTask || day.tasks === 0 ? 1 : day.tasks;
  switch (key) {
    case "cost":
      return day.cost * scale;
    case "tokens":
      return day.tokens * scale;
    case "rounds":
      return day.rounds * scale;
    case "thinking":
      return day.thinkingMs * scale;
    case "active":
      return day.activeMs * scale;
    case "ttc":
      return day.ttcSamples ? day.ttcMs : 0;
    case "completed":
      return day.tasks;
  }
}

export function kpiConcurrencyContent(
  widget: KpiWidgetState,
  data: KpiConcurrencyData,
): KpiWidgetContent {
  const definition = kpiWidgetDefinition(widget.kind);
  if (!definition) {
    return {
      label: t("kpiWidget.unknown"),
      hint: widget.kind,
      value: "—",
      detail: t("kpiWidget.unknownHint"),
      pill: "",
      spark: [],
    };
  }
  switch (definition.metric) {
    case "slotsHeld":
      return {
        label: data.heldLabel,
        hint: data.heldHint,
        value: `${data.held} / ${data.cap}`,
        detail: data.heldDetail,
        pill: "",
        spark: [],
      };
    case "waiting":
      return {
        label: data.waitingLabel,
        hint: data.waitingHint,
        value: formatCount(data.waiting),
        detail: data.waitingDetail,
        pill: "",
        spark: [],
      };
    case "capacity":
      return {
        label: data.thirdLabel,
        hint: data.thirdHint,
        value: formatCount(data.thirdValue),
        detail: data.thirdDetail,
        pill: "",
        spark: [],
      };
    case "capsFull":
      return {
        label: t("kpiWidget.capsFull"),
        hint: data.capsHint,
        value: formatCount(data.caps),
        detail: data.capsDetail,
        pill: "",
        spark: [],
      };
    default:
      return {
        label: t(definition.labelKey),
        hint: t(definition.hintKey),
        value: "—",
        detail: "",
        pill: "",
        spark: [],
      };
  }
}

export interface KpiPageContext {
  kind: "agency" | "project" | "agent";

  agentId?: string;

  projectId?: string;

  filterAgentId?: string;

  filterProjectId?: string;
}

export function kpiWidgetScope(
  widget: KpiWidgetState,
  page: KpiPageContext,
): { agentId: string; projectId: string } {
  let agentId = page.agentId ?? page.filterAgentId ?? "";
  let projectId = page.projectId ?? page.filterProjectId ?? "";
  const pinned = parseKpiScope(widget.scope);
  if (pinned?.kind === "agent") agentId = pinned.id;
  if (pinned?.kind === "project") projectId = pinned.id;
  return { agentId, projectId };
}

export function kpiWidgetRange(
  widget: KpiWidgetState,
  pageRange: UsageRange,
): UsageRange {
  return (USAGE_RANGES as readonly string[]).includes(widget.range)
    ? (widget.range as UsageRange)
    : pageRange;
}

export function kpiUsageScopeKey(
  scope: {
    agentId: string;
    projectId: string;
  },
  range: UsageRange,
): string {
  return `usage|${scope.agentId}|${scope.projectId}|${range}`;
}

export function kpiBoardScope(
  widget: KpiWidgetState,
  page: KpiPageContext,
): { kind: "agent" | "project"; id: string } | undefined {
  const scope = kpiWidgetScope(widget, page);
  if (scope.agentId) return { kind: "agent", id: scope.agentId };
  if (scope.projectId) return { kind: "project", id: scope.projectId };
  return undefined;
}

export function parseKpiScope(
  scope: string,
): { kind: "agent" | "project"; id: string } | undefined {
  if (!scope) return undefined;
  const [kind, id] = scope.split(":");
  if ((kind !== "agent" && kind !== "project") || !id) return undefined;
  return { kind, id };
}

export function formatKpiScope(kind: "agent" | "project", id: string): string {
  return id ? `${kind}:${id}` : "";
}
