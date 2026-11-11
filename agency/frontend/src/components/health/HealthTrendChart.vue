<!--
CherryPi: A multi-agent-multi-human self-hosted agency
Copyright (C) 2026 CherryPi authors <cherrypi.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<template lang="pug">
Card
  CardHeader
    CardTitle {{ $t("health.trend.title") }}
    CardDescription {{ subtitle }}
    CardAction
      .bg-muted.flex.rounded-lg.p-1
        button.text-11.rounded-md.px-2.py-1.font-medium(
          v-for="option in METRICS"
          :key="option.value"
          :class="metric === option.value ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="metric = option.value"
        ) {{ $t(option.labelKey) }}
  CardContent
    p.text-muted-foreground.py-10.text-center.text-sm(v-if="!points.length")
      | {{ $t("health.trend.empty") }}
    .health-chart(v-else ref="chartRoot")
      ChartContainer.chart.w-full(class="h-[238px]" :config="config")
        VisXYContainer(
          ref="containerRef"
          :data="points"
          :x-domain="xDomain"
          :y-domain="yDomain"
          :duration="0"
        )
          template(v-if="metric === 'network'")
            VisArea(
              :color="HEALTH_COLORS.network"
              :x="x"
              :y="(d: HealthPoint) => d.rx"
              :opacity="0.25"
            )
            VisLine(
              :color="HEALTH_COLORS.network"
              :x="x"
              :y="(d: HealthPoint) => d.rx"
            )
            VisLine(
              :color="HEALTH_COLORS.cpu"
              :x="x"
              :y="(d: HealthPoint) => d.tx"
            )
          template(v-else)
            VisArea(:color="activeColor" :x="x" :y="activeY" :opacity="0.25")
            VisLine(:color="activeColor" :x="x" :y="activeY")
          VisAxis(
            type="x"
            :tick-line="false"
            :domain-line="false"
            :grid-line="false"
            :tick-format="timeTick"
          )
          VisAxis(type="y" :tick-line="false" :domain-line="false")
          ChartTooltip
          ChartCrosshair(
            :template="componentToString(config, ChartTooltipContent, { labelFormatter: (d: number | Date) => new Date(Number(d)).toLocaleString() })"
          )
  CardFooter.justify-between.gap-3
    .text-2xs.text-muted-foreground.flex.w-full.flex-wrap.items-center.justify-between.gap-3
      .flex.flex-wrap.items-center.gap-3
        span.flex.items-center.gap-1(v-for="item in legend" :key="item.label")
          i.inline-block.size-2.rounded-sm(:style="{ background: item.color }")
          | {{ item.label }}
      span.tabular-nums {{ $t("health.trend.samples", { count: points.length, range: rangeLabel }) }}
</template>

<script setup lang="ts">
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartCrosshair,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  formatBytesPerSecond,
  fullestDisk,
  HEALTH_COLORS,
  healthRangeLabel,
  healthRangeMs,
  niceCeil,
  type HealthRange,
} from "@/lib/health";
import type { AgentHealthSample } from "@cherrypi/sdk";
import { VisArea, VisAxis, VisLine, VisXYContainer } from "@unovis/vue";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

type Metric = "cpu" | "memory" | "disk" | "load" | "network" | "gpu";

interface HealthPoint {
  time: number;
  cpu: number;
  memory: number;
  disk: number;
  load: number;
  rx: number;
  tx: number;
  gpu: number;
}

const props = defineProps<{
  samples: AgentHealthSample[];
  range: HealthRange;
}>();

const METRICS: { value: Metric; labelKey: string }[] = [
  { value: "cpu", labelKey: "health.kpis.cpu" },
  { value: "memory", labelKey: "health.trend.memory" },
  { value: "disk", labelKey: "health.trend.disk" },
  { value: "load", labelKey: "health.trend.load" },
  { value: "network", labelKey: "health.trend.network" },
  { value: "gpu", labelKey: "health.gpus.title" },
];

function metricLabel(): string {
  const option = METRICS.find((item) => item.value === metric.value);
  return option ? t(option.labelKey) : t("health.kpis.cpu");
}
const metric = ref<Metric>("cpu");

const points = computed<HealthPoint[]>(() =>
  props.samples.map((sample) => ({
    time: Number(sample.sampledAt),
    cpu: sample.cpuPercent,
    memory: sample.memory?.usedPercent ?? 0,
    disk: fullestDisk(sample.disks)?.usedPercent ?? 0,
    load: sample.load1,
    rx: aggregateRate(sample, "rx"),
    tx: aggregateRate(sample, "tx"),
    gpu: maxGpu(sample),
  })),
);

function aggregateRate(
  sample: AgentHealthSample,
  direction: "rx" | "tx",
): number {
  return sample.networks.reduce(
    (sum, network) =>
      sum +
      (direction === "rx"
        ? network.rxBytesPerSecond
        : network.txBytesPerSecond),
    0,
  );
}

function maxGpu(sample: AgentHealthSample): number {
  return sample.gpus.reduce(
    (max, gpu) => Math.max(max, gpu.utilizationPercent),
    0,
  );
}

const x = (point: HealthPoint) => point.time;

const xDomain = computed<[number, number]>(() => {
  const end = points.value[points.value.length - 1]?.time ?? Date.now();
  return [end - healthRangeMs(props.range), end];
});

const yDomain = computed<[number, number]>(() => {
  if (metric.value === "load") {
    return [0, niceCeil(Math.max(1, ...points.value.map((p) => p.load)))];
  }
  if (metric.value === "network") {
    const max = Math.max(
      1,
      ...points.value.flatMap((point) => [point.rx, point.tx]),
    );
    return [0, niceCeil(max)];
  }
  return [0, 100];
});

const SLIDE_MS = 320;

const chartRoot = ref<HTMLElement>();
const containerRef = ref<{
  component?: { value?: XYContainerInstance } | XYContainerInstance;
}>();
let slideTimer: number | undefined;

interface XYContainerInstance {
  components?: { xScale?: XScale; yScale?: YScale }[];
}

interface XScale {
  (value: number): number;
  range(): number[];
}

interface YScale {
  range(): number[];
}

function containerInstance(): XYContainerInstance | undefined {
  const exposed = containerRef.value?.component;
  if (!exposed) return undefined;
  return "value" in exposed
    ? (exposed as { value?: XYContainerInstance }).value
    : (exposed as XYContainerInstance);
}

function plotScales() {
  const components = containerInstance()?.components ?? [];
  const xScale = components.find((item) => item.xScale)?.xScale;
  const yScale = components.find((item) => item.yScale)?.yScale;
  if (!xScale || !yScale) return undefined;
  return { xScale, yScale };
}

/**
 * Clip the line and area to the plot rect, so the newest sample entering from
 * the right during the slide is cut at the plot edge instead of drawing over
 * the axis margin.
 */
function applyPlotClip() {
  const root = chartRoot.value;
  const scales = plotScales();
  const svg = root?.querySelector("svg");
  if (!root || !svg || !scales) return;

  const [x0 = 0, x1 = 0] = scales.xScale.range();
  const [y0 = 0, y1 = 0] = scales.yScale.range();
  const width = Math.abs(x1 - x0);
  const height = Math.abs(y1 - y0);
  if (!(width > 0) || !(height > 0)) return;

  const ns = "http://www.w3.org/2000/svg";
  let defs = svg.querySelector("defs[data-health-clip]");
  if (!defs) {
    defs = document.createElementNS(ns, "defs");
    defs.setAttribute("data-health-clip", "");
    const clip = document.createElementNS(ns, "clipPath");
    clip.setAttribute("id", "health-plot-clip");
    const rect = document.createElementNS(ns, "rect");
    rect.setAttribute("x", "0");
    rect.setAttribute("y", "0");
    clip.appendChild(rect);
    defs.appendChild(clip);
    svg.prepend(defs);
  }
  const rect = defs.querySelector("rect");
  rect?.setAttribute("width", String(width));
  rect?.setAttribute("height", String(height));

  for (const node of plotPaths(root)) {
    node.style.clipPath = "url(#health-plot-clip)";
  }
}

function plotPaths(root: HTMLElement): SVGPathElement[] {
  return [
    ...root.querySelectorAll<SVGPathElement>(
      'path[class*="linePath"], path[class*="area"]',
    ),
  ];
}

function slideBy(distance: number) {
  const root = chartRoot.value;
  if (!root || !(distance > 0.1)) return;
  if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return;

  const nodes = plotPaths(root);
  if (!nodes.length) return;

  for (const node of nodes) {
    node.style.transition = "none";
    node.style.transform = `translateX(${distance}px)`;
  }

  void root.getBoundingClientRect();

  requestAnimationFrame(() => {
    for (const node of nodes) {
      node.style.transition = `transform ${SLIDE_MS}ms cubic-bezier(0.22, 0.61, 0.36, 1)`;
      node.style.transform = "translateX(0)";
    }
    window.clearTimeout(slideTimer);
    slideTimer = window.setTimeout(() => {
      for (const node of nodes) {
        node.style.transition = "";
        node.style.transform = "";
      }
    }, SLIDE_MS + 80);
  });
}

watch(
  points,
  (next, previous) => {
    applyPlotClip();
    const newest = next[next.length - 1]?.time ?? 0;
    const previousNewest = previous?.[previous.length - 1]?.time ?? 0;
    const step = newest - previousNewest;

    if (!previousNewest || step <= 0 || step > healthRangeMs(props.range) / 4) {
      return;
    }
    const scale = plotScales()?.xScale;
    if (!scale) return;
    void nextTick(() => slideBy(scale(newest) - scale(previousNewest)));
  },
  { flush: "post" },
);

watch([metric, () => props.range], () => void nextTick(applyPlotClip), {
  flush: "post",
});

onMounted(() => {
  void nextTick(applyPlotClip);
});

onBeforeUnmount(() => {
  window.clearTimeout(slideTimer);
});

type ScalarMetric = Exclude<Metric, "network">;

function activeY(point: HealthPoint): number {
  return point[metric.value as ScalarMetric];
}

const activeColor = computed(() => {
  switch (metric.value) {
    case "memory":
      return HEALTH_COLORS.memory;
    case "disk":
      return HEALTH_COLORS.disk;
    case "load":
      return HEALTH_COLORS.load;
    case "gpu":
      return HEALTH_COLORS.gpu;
    default:
      return HEALTH_COLORS.cpu;
  }
});

const config = computed<ChartConfig>(() => {
  if (metric.value === "network") {
    return {
      rx: {
        label: t("health.trend.received"),
        color: HEALTH_COLORS.network,
        format: (value: number) => formatBytesPerSecond(value),
      },
      tx: {
        label: t("health.trend.sent"),
        color: HEALTH_COLORS.cpu,
        format: (value: number) => formatBytesPerSecond(value),
      },
    };
  }
  return {
    [metric.value]: {
      label: metric.value === "load" ? t("health.trend.load1m") : metricLabel(),
      color: activeColor.value,
      format:
        metric.value === "load"
          ? (value: number) => value.toFixed(2)
          : (value: number) => `${value.toFixed(0)}%`,
    },
  };
});

const subtitle = computed(() => {
  switch (metric.value) {
    case "network":
      return t("health.trend.subtitleNetwork", {
        range: healthRangeLabel(props.range),
      });
    case "gpu":
      return t("health.trend.subtitleGpu", {
        range: healthRangeLabel(props.range),
      });
    case "load":
      return t("health.trend.subtitleLoad", {
        range: healthRangeLabel(props.range),
      });
    default:
      return t("health.trend.subtitleMetric", {
        metric: metricLabel(),
        range: healthRangeLabel(props.range),
      });
  }
});

const rangeLabel = computed(() => healthRangeLabel(props.range));

const legend = computed(() => {
  if (metric.value === "network") {
    return [
      { label: t("health.trend.received"), color: HEALTH_COLORS.network },
      { label: t("health.trend.sent"), color: HEALTH_COLORS.cpu },
    ];
  }
  return [{ label: metricLabel(), color: activeColor.value }];
});

function timeTick(value: number): string {
  const date = new Date(value);
  const sameDay = date.toDateString() === new Date().toDateString();
  return sameDay
    ? date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
</script>

<i18n lang="yaml">
en:
  health:
    trend:
      title: "Health over time"
      empty: "No health samples in this range."
      sent: "Sent"
      received: "Received"
      load1m: "Load 1m"
      memory: "Memory"
      disk: "Disk"
      subtitleNetwork: "Interface throughput · {range}."
      subtitleGpu: "Busiest GPU utilization · {range}."
      subtitleLoad: "Load average (1m) · {range}."
      subtitleMetric: "{metric} over {range}."
      load: "Load"
      network: "Network"
      samples: "{count} samples · {range}"
ru:
  health:
    trend:
      title: "Здоровье во времени"
      empty: "В этом диапазоне нет замеров здоровья."
      sent: "Отправлено"
      received: "Получено"
      load1m: "Нагрузка 1м"
      memory: "Память"
      disk: "Диск"
      subtitleNetwork: "Пропускная способность интерфейса · {range}."
      subtitleGpu: "Загрузка самой нагруженной GPU · {range}."
      subtitleLoad: "Средняя нагрузка (1м) · {range}."
      subtitleMetric: "{metric} за {range}."
      load: "Нагрузка"
      network: "Сеть"
      samples: "замеров: {count} · {range}"
</i18n>
