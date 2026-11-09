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

// The diagrams are coloured from the VitePress design tokens (`--vp-c-*`) read

// value VitePress ships, which keeps a diagram readable even if a custom.css

import type { MermaidConfig } from "mermaid";

type Mermaid = (typeof import("mermaid"))["default"];

function token(name: string, fallback: string): string {
  if (typeof document === "undefined") return fallback;
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  return value || fallback;
}

function themeVariables(darkMode: boolean): MermaidConfig["themeVariables"] {
  const brand = token("--vp-c-brand-1", darkMode ? "#a8b1ff" : "#3451b2");
  const accent = token("--vp-c-brand-soft", "rgba(100, 108, 255, 0.2)");
  const text = token("--vp-c-text-1", darkMode ? "#dfdfd6" : "#3c3c43");
  const muted = token("--vp-c-text-3", darkMode ? "#98989f" : "#67676c");
  const background = token("--vp-c-bg", darkMode ? "#1b1b1f" : "#ffffff");
  const surface = token("--vp-c-bg-soft", darkMode ? "#202127" : "#f6f6f7");
  const raised = token("--vp-c-bg-alt", darkMode ? "#161618" : "#f6f6f7");
  const divider = token("--vp-c-divider", darkMode ? "#2e2e32" : "#e2e2e3");
  const note = darkMode ? "#3a3520" : "#fff8dd";
  const noteBorder = darkMode ? "#8a7b3a" : "#d9b44a";

  return {
    fontFamily: token(
      "--vp-font-family-base",
      "Inter, ui-sans-serif, system-ui, sans-serif",
    ),
    fontSize: "15px",
    background: "transparent",
    primaryColor: accent,
    primaryBorderColor: brand,
    primaryTextColor: text,
    secondaryColor: surface,
    secondaryBorderColor: divider,
    secondaryTextColor: text,
    tertiaryColor: raised,
    tertiaryBorderColor: divider,
    tertiaryTextColor: text,
    mainBkg: accent,
    nodeBorder: brand,
    nodeTextColor: text,
    lineColor: muted,
    textColor: text,
    titleColor: text,
    edgeLabelBackground: background,

    clusterBkg: "transparent",
    clusterBorder: divider,
    actorBkg: accent,
    actorBorder: brand,
    actorTextColor: text,
    actorLineColor: muted,
    signalColor: muted,
    signalTextColor: text,
    labelBoxBkgColor: surface,
    labelBoxBorderColor: brand,
    labelTextColor: text,
    loopTextColor: text,
    noteBkgColor: note,
    noteBorderColor: noteBorder,
    noteTextColor: text,
    activationBkgColor: surface,
    activationBorderColor: brand,
    sequenceNumberColor: background,
  };
}

function config(darkMode: boolean): MermaidConfig {
  return {
    startOnLoad: false,

    securityLevel: "strict",

    theme: "base",
    themeVariables: themeVariables(darkMode),

    flowchart: {
      curve: "basis",
      htmlLabels: true,
      useMaxWidth: true,
      padding: 12,
      nodeSpacing: 26,
      rankSpacing: 34,
      subGraphTitleMargin: { top: 0, bottom: 10 },
    },
    sequence: {
      useMaxWidth: true,
      mirrorActors: false,
      actorMargin: 44,
      messageMargin: 30,
      wrap: true,
    },
  };
}

let loading: Promise<Mermaid> | null = null;
let initializedFor: boolean | null = null;
let counter = 0;

async function load(darkMode: boolean): Promise<Mermaid> {
  loading ??= import("mermaid").then((module) => module.default);
  const mermaid = await loading;

  if (typeof document !== "undefined" && document.fonts) {
    await document.fonts.ready;
  }
  if (initializedFor !== darkMode) {
    mermaid.initialize(config(darkMode));
    initializedFor = darkMode;
  }
  return mermaid;
}

export async function renderDiagram(
  source: string,
  darkMode: boolean,
): Promise<string> {
  const mermaid = await load(darkMode);
  const id = `cherrypi-mermaid-${++counter}`;
  try {
    const { svg } = await mermaid.render(id, source);
    return svg;
  } catch (error) {
    document.getElementById(`d${id}`)?.remove();
    document.getElementById(id)?.remove();
    throw error;
  }
}

export function naturalSize(svg: string): { width: number; height: number } {
  const holder = document.createElement("div");
  holder.innerHTML = svg;
  const root = holder.querySelector("svg");
  if (!root) return { width: 960, height: 540 };

  const viewBox = (root.getAttribute("viewBox") ?? "")
    .split(/[\s,]+/)
    .map((part) => Number.parseFloat(part));
  if (
    viewBox.length === 4 &&
    viewBox.every((value) => Number.isFinite(value))
  ) {
    return { width: viewBox[2]!, height: viewBox[3]! };
  }

  const width = Number.parseFloat(root.getAttribute("width") ?? "");
  const height = Number.parseFloat(root.getAttribute("height") ?? "");
  return {
    width: Number.isFinite(width) && width > 0 ? width : 960,
    height: Number.isFinite(height) && height > 0 ? height : 540,
  };
}
