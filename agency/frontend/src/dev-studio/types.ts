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

import type { Component } from "vue";

export interface StudioControl {
  prop: string;
  label: string;
  type: "boolean" | "text";

  initial: boolean | string;
}

export interface StudioScenario {
  id: string;
  name: string;

  description: string;

  props: Record<string, unknown>;

  events?: string[];

  controls?: StudioControl[];

  width?: number;
}

export interface StudioComponent {
  id: string;
  name: string;

  source: string;
  /** One line about what the component is for. */
  blurb: string;
  component: Component;
  scenarios: StudioScenario[];
}
