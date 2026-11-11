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
import {
  fitViewBox,
  isHomeView,
  MERMAID_HOME,
  MERMAID_MAX_SCALE,
  MERMAID_PAN_STEP,
  pannedView,
  parseViewBox,
  viewBoxAttribute,
  zoomedView,
  type MermaidView,
} from "./mermaid";

export interface MermaidToolsOptions {
  code: string;

  onExpand?: () => void;

  clipboard?: Pick<Clipboard, "writeText">;
}

export interface MermaidTools {
  dispose(): void;
}

const ICONS = {
  left: '<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>',
  up: '<path d="m5 12 7-7 7 7"/><path d="M12 19V5"/>',
  down: '<path d="M12 5v14"/><path d="m19 12-7 7-7-7"/>',
  right: '<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>',
  zoomIn:
    '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>' +
    '<path d="M11 8v6"/><path d="M8 11h6"/>',
  zoomOut:
    '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>' +
    '<path d="M8 11h6"/>',
  reset:
    '<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>' +
    '<path d="M3 3v5h5"/>',
  copy:
    '<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>' +
    '<path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>',
  copied: '<path d="M20 6 9 17l-5-5"/>',
  expand:
    '<path d="M15 3h6v6"/><path d="M9 21H3v-6"/>' +
    '<path d="m21 3-7 7"/><path d="m3 21 7-7"/>',
} as const;

type IconName = keyof typeof ICONS;

const COPIED_MS = 1200;

function iconMarkup(icon: IconName): string {
  return (
    `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"` +
    ` stroke-width="2" stroke-linecap="round" stroke-linejoin="round"` +
    ` aria-hidden="true">${ICONS[icon]}</svg>`
  );
}

function iconButton(label: string, icon: IconName): HTMLButtonElement {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "mermaid-tool";
  button.title = label;
  button.setAttribute("aria-label", label);
  button.innerHTML = iconMarkup(icon);
  return button;
}

export function attachMermaidTools(
  box: Element,
  options: MermaidToolsOptions,
): MermaidTools {
  const drawing = box.querySelector("svg");
  if (!drawing) return { dispose() {} };
  const svg = drawing;
  const base = parseViewBox(svg.getAttribute("viewBox"));
  const original = svg.getAttribute("viewBox");

  const toolbar = document.createElement("div");
  toolbar.className = "mermaid-toolbar";
  toolbar.setAttribute("role", "toolbar");
  toolbar.setAttribute("aria-label", t("mermaid.controls"));

  box.classList.add("mermaid-diagram");

  let view: MermaidView = MERMAID_HOME;

  const controls: {
    button: HTMLButtonElement;
    enabled(view: MermaidView): boolean;
  }[] = [];

  function add(
    button: HTMLButtonElement,
    enabled: (view: MermaidView) => boolean,
  ) {
    controls.push({ button, enabled });
    toolbar.append(button);
  }

  function apply(next: MermaidView) {
    if (next === view) return;
    view = next;
    for (const { button, enabled } of controls) {
      button.disabled = !enabled(view);
    }
    if (base) {
      svg.setAttribute("viewBox", viewBoxAttribute(fitViewBox(base, view)));
    }
  }

  const zoomed = (v: MermaidView) => v.scale > MERMAID_HOME.scale;
  const zoomable = (v: MermaidView) => v.scale < MERMAID_MAX_SCALE;

  const panLeft = iconButton(t("mermaid.panLeft"), "left");
  panLeft.addEventListener("click", () =>
    apply(pannedView(view, -MERMAID_PAN_STEP, 0)),
  );
  add(panLeft, zoomed);

  const panUp = iconButton(t("mermaid.panUp"), "up");
  panUp.addEventListener("click", () =>
    apply(pannedView(view, 0, -MERMAID_PAN_STEP)),
  );
  add(panUp, zoomed);

  const panDown = iconButton(t("mermaid.panDown"), "down");
  panDown.addEventListener("click", () =>
    apply(pannedView(view, 0, MERMAID_PAN_STEP)),
  );
  add(panDown, zoomed);

  const panRight = iconButton(t("mermaid.panRight"), "right");
  panRight.addEventListener("click", () =>
    apply(pannedView(view, MERMAID_PAN_STEP, 0)),
  );
  add(panRight, zoomed);

  const zoomOut = iconButton(t("mermaid.zoomOut"), "zoomOut");
  zoomOut.addEventListener("click", () => apply(zoomedView(view, -1)));
  add(zoomOut, (v) => base !== null && zoomed(v));

  const zoomIn = iconButton(t("mermaid.zoomIn"), "zoomIn");
  zoomIn.addEventListener("click", () => apply(zoomedView(view, 1)));
  add(zoomIn, (v) => base !== null && zoomable(v));

  const reset = iconButton(t("mermaid.resetZoom"), "reset");
  reset.addEventListener("click", () => apply(MERMAID_HOME));
  add(reset, (v) => !isHomeView(v));

  let copiedTimer: ReturnType<typeof setTimeout> | undefined;
  const copy = iconButton(t("mermaid.copySource"), "copy");
  copy.addEventListener("click", () => {
    const clipboard = options.clipboard ?? navigator.clipboard;
    void clipboard
      ?.writeText(options.code)
      .then(() => {
        copy.innerHTML = iconMarkup("copied");
        copy.title = t("common.copied");
        copy.setAttribute("aria-label", t("common.copied"));
        clearTimeout(copiedTimer);
        copiedTimer = setTimeout(() => {
          copy.innerHTML = iconMarkup("copy");
          copy.title = t("mermaid.copySource");
          copy.setAttribute("aria-label", t("mermaid.copySource"));
        }, COPIED_MS);
      })
      .catch(() => {});
  });
  add(copy, () => options.code.trim() !== "");

  if (options.onExpand) {
    const expand = iconButton(t("mermaid.openEnlarged"), "expand");
    expand.addEventListener("click", () => options.onExpand?.());
    add(expand, () => true);
  }

  box.append(toolbar);
  apply({ ...MERMAID_HOME });

  return {
    dispose() {
      clearTimeout(copiedTimer);
      toolbar.remove();
      box.classList.remove("mermaid-diagram");
      if (original) svg.setAttribute("viewBox", original);
      else svg.removeAttribute("viewBox");
    },
  };
}
