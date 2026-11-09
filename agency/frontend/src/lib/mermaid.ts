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

export const MERMAID_MIN_SCALE = 1;
export const MERMAID_MAX_SCALE = 4;

export const MERMAID_ZOOM_STEP = 1.25;

export const MERMAID_PAN_STEP = 0.12;

export interface MermaidBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface MermaidView {
  scale: number;
  x: number;
  y: number;
}

export const MERMAID_HOME: MermaidView = { scale: 1, x: 0, y: 0 };

export function clampScale(scale: number): number {
  const rounded = Math.round(scale * 100) / 100;
  return Math.min(Math.max(rounded, MERMAID_MIN_SCALE), MERMAID_MAX_SCALE);
}

export function clampOffset(offset: number): number {
  const limit = 0.5 - 0.5 / MERMAID_MAX_SCALE;
  return Math.min(Math.max(offset, -limit), limit);
}

export function zoomedView(view: MermaidView, direction: 1 | -1): MermaidView {
  const factor = direction === 1 ? MERMAID_ZOOM_STEP : 1 / MERMAID_ZOOM_STEP;
  const scale = clampScale(view.scale * factor);
  if (scale === view.scale) return view;
  if (scale === MERMAID_MIN_SCALE) return MERMAID_HOME;

  return {
    scale,
    x: clampOffset(view.x),
    y: clampOffset(view.y),
  };
}

export function pannedView(
  view: MermaidView,
  dx: number,
  dy: number,
): MermaidView {
  if (view.scale <= MERMAID_MIN_SCALE) return MERMAID_HOME;
  return {
    scale: view.scale,
    x: clampOffset(view.x + dx),
    y: clampOffset(view.y + dy),
  };
}

export function isHomeView(view: MermaidView): boolean {
  return view.scale === MERMAID_MIN_SCALE && view.x === 0 && view.y === 0;
}

export function fitViewBox(base: MermaidBox, view: MermaidView): MermaidBox {
  const width = base.width / view.scale;
  const height = base.height / view.scale;
  return {
    x: base.x + (base.width - width) / 2 + view.x * width,
    y: base.y + (base.height - height) / 2 + view.y * height,
    width,
    height,
  };
}

export function viewBoxAttribute(box: MermaidBox): string {
  const round = (value: number) => Math.round(value * 1000) / 1000;
  return `${round(box.x)} ${round(box.y)} ${round(box.width)} ${round(box.height)}`;
}

export function parseViewBox(value: string | null): MermaidBox | null {
  if (!value) return null;
  const parts = value
    .trim()
    .split(/[\s,]+/)
    .map(Number);
  if (parts.length !== 4) return null;
  const [x, y, width, height] = parts as [number, number, number, number];
  if (![x, y, width, height].every(Number.isFinite)) return null;
  if (width <= 0 || height <= 0) return null;
  return { x, y, width, height };
}

export function mermaidTheme(dark: boolean): "dark" | "default" {
  return dark ? "dark" : "default";
}
