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

import { createContext } from "reka-ui";
import type { Component, Ref } from "vue";

export { default as ChartContainer } from "./ChartContainer.vue";
export { default as ChartLegendContent } from "./ChartLegendContent.vue";
export { default as ChartTooltipContent } from "./ChartTooltipContent.vue";
export { componentToString } from "./utils";

export const THEMES = { light: "", dark: ".dark" } as const;

export type ChartConfig = {
  [k in string]: {
    label?: string | Component;
    icon?: string | Component;

    format?: (value: number) => string;
  } & (
    | { color?: string; theme?: never }
    | { color?: never; theme: Record<keyof typeof THEMES, string> }
  );
};

interface ChartContextProps {
  id: string;
  config: Ref<ChartConfig>;
}

export const [useChart, provideChartContext] =
  createContext<ChartContextProps>("Chart");

export {
  VisCrosshair as ChartCrosshair,
  VisTooltip as ChartTooltip,
} from "@unovis/vue";
