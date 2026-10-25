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

import {
  CircleCheckIcon,
  CircleXIcon,
  InfoIcon,
  LoaderCircleIcon,
  TriangleAlertIcon,
} from "@lucide/vue";
import { h, type Component, type VNode } from "vue";
import type { IconProps, ToastContainerOptions } from "vue3-toastify";

const ICONS: Partial<Record<IconProps["type"], Component>> = {
  success: CircleCheckIcon,
  warning: TriangleAlertIcon,
  error: CircleXIcon,
  info: InfoIcon,
  loading: LoaderCircleIcon,
};

export function toastIcon({ type }: IconProps): VNode | undefined {
  const icon = ICONS[type];
  if (!icon) return undefined;
  return h(icon, {
    class: type === "loading" ? "size-4 animate-spin" : "size-4",
  });
}

export const toastOptions: Partial<ToastContainerOptions> = {
  autoClose: 3000,
  transition: "slide",
  toastClassName: "cherrypi-toast",
  icon: toastIcon,
};
