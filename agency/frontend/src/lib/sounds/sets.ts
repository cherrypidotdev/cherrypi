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

import { ATTENTION_EVENTS } from "../attentionEvents";
import type { Patch } from "./patch";

export interface SoundSet {
  id: string;

  nameKey: string;
  /** Catalog key for the set's one-line description. */
  descriptionKey: string;

  volume: number;

  patches: Record<string, Patch>;
}

export const DEFAULT_SOUND_SET_ID = "cherry";

export const DEFAULT_SOUND_VOLUME = 0.85;

export function clampSoundVolume(value: unknown): number {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return DEFAULT_SOUND_VOLUME;
  return Math.min(1, Math.max(0, n));
}

// prettier-ignore
export const SOUND_SETS: SoundSet[] = [
  {
    id: "classic",
    nameKey: "soundSet.classic",
    descriptionKey: "soundSet.classicHint",
    volume: 0.75,
    patches: {
      "task_started": {
        layers: [
          {"source":{"type":"sine","frequency":659.25},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":987.77},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "input_requested": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "task_finished": {
        layers: [
          {"source":{"type":"sine","frequency":987.77},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "failed": {
        layers: [
          {"source":{"type":"sine","frequency":311.13},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":233.08},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "task_completed": {
        layers: [
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":659.25},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.28},
        ],
      },
      "task_cancelled": {
        layers: [
          {"source":{"type":"sine","frequency":392},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":311.13},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "team_invitation": {
        layers: [
          {"source":{"type":"sine","frequency":587.33},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "task_comment": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "task_mention": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "prompt_proposal_created": {
        layers: [
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":698.46},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
      "default": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.015,"decay":0.245,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18,"delay":0.14},
        ],
      },
    },
  },
  {
    id: "cherry",
    nameKey: "soundSet.cherry",
    descriptionKey: "soundSet.cherryHint",
    volume: 1,
    patches: {
      "task_started": {
        layers: [
          {"source":{"type":"sine","frequency":261.63},"envelope":{"attack":0.006,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05},
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.004,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.22,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.09},
          {"source":{"type":"triangle","frequency":1046.5},"envelope":{"attack":0.003,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.035,"delay":0.09},
        ],
      },
      "input_requested": {
        layers: [
          {"source":{"type":"sine","frequency":440},"envelope":{"attack":0.008,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.045},
          {"source":{"type":"triangle","frequency":659.25},"envelope":{"attack":0.005,"decay":0.12,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.005,"decay":0.26,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.1},
          {"source":{"type":"sine","frequency":1318.51},"envelope":{"attack":0.003,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.025,"delay":0.1},
        ],
      },
      "task_finished": {
        layers: [
          {"source":{"type":"sine","frequency":392},"envelope":{"attack":0.006,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13},
          {"source":{"type":"sine","frequency":659.25},"envelope":{"attack":0.005,"decay":0.28,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14,"delay":0.1},
        ],
      },
      "failed": {
        layers: [
          {"source":{"type":"sine","frequency":110},"envelope":{"attack":0.003,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.09},
          {"source":{"type":"triangle","frequency":220},"envelope":{"attack":0.004,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"triangle","frequency":207.65},"envelope":{"attack":0.004,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.11},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.04,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"lowpass","frequency":500,"Q":0.8}},
        ],
      },
      "task_completed": {
        layers: [
          {"source":{"type":"sine","frequency":261.63},"envelope":{"attack":0.008,"decay":0.28,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.14},
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.004,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13},
          {"source":{"type":"sine","frequency":659.25},"envelope":{"attack":0.004,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13,"delay":0.07},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.22,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14,"delay":0.14,"effects":[{"type":"delay","delay":0.09,"feedback":0.18,"wet":0.06,"lowpass":4000}]},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.003,"decay":0.36,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.21,"effects":[{"type":"delay","delay":0.11,"feedback":0.2,"wet":0.08,"lowpass":4000}]},
        ],
      },
      "task_cancelled": {
        layers: [
          {"source":{"type":"sine","frequency":174.61},"envelope":{"attack":0.006,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.04},
          {"source":{"type":"sine","frequency":440},"envelope":{"attack":0.004,"decay":0.12,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13},
          {"source":{"type":"sine","frequency":349.23},"envelope":{"attack":0.005,"decay":0.22,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13,"delay":0.1},
        ],
      },
      "team_invitation": {
        layers: [
          {"source":{"type":"sine","frequency":293.66},"envelope":{"attack":0.006,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.04},
          {"source":{"type":"sine","frequency":587.33},"envelope":{"attack":0.004,"decay":0.13,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.004,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14,"delay":0.09},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.003,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.06,"delay":0.17},
        ],
      },
      "task_comment": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.005,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.005,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13,"delay":0.1},
        ],
      },
      "task_mention": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.005,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.005,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13,"delay":0.1},
        ],
      },
      "prompt_proposal_created": {
        layers: [
          {"source":{"type":"sine","frequency":261.63},"envelope":{"attack":0.006,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05},
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.004,"decay":0.13,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.26,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13,"delay":0.1},
        ],
      },
      "default": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.005,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.005,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13,"delay":0.1},
        ],
      },
    },
  },
  {
    id: "wood",
    nameKey: "soundSet.wood",
    descriptionKey: "soundSet.woodHint",
    volume: 1,
    patches: {
      "task_started": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1384,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":329.63,"end":324.9,"time":0.02}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":659.26},"envelope":{"attack":0.002,"decay":0.099,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1318.52},"envelope":{"attack":0.002,"decay":0.054,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.11,"filter":{"type":"bandpass","frequency":2074,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":493.88,"end":486.8,"time":0.02}},"envelope":{"attack":0.002,"decay":0.22,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.11},
          {"source":{"type":"sine","frequency":987.76},"envelope":{"attack":0.002,"decay":0.121,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.11},
          {"source":{"type":"sine","frequency":1975.52},"envelope":{"attack":0.002,"decay":0.066,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.11},
        ],
      },
      "input_requested": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1848,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":440,"end":433.69,"time":0.02}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.002,"decay":0.099,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.002,"decay":0.054,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.12,"filter":{"type":"bandpass","frequency":2467,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":587.33,"end":578.91,"time":0.02}},"envelope":{"attack":0.002,"decay":0.26,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.12},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.143,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.12},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.002,"decay":0.078,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.12},
        ],
      },
      "task_finished": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":2198,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":523.25,"end":515.75,"time":0.02}},"envelope":{"attack":0.002,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.002,"decay":0.11,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":2093},"envelope":{"attack":0.002,"decay":0.06,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.11,"filter":{"type":"bandpass","frequency":1646,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":392,"end":386.38,"time":0.02}},"envelope":{"attack":0.002,"decay":0.3,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.11},
          {"source":{"type":"sine","frequency":784},"envelope":{"attack":0.002,"decay":0.165,"sustain":0,"release":0,"curve":"ramp"},"gain":0.036,"delay":0.11},
          {"source":{"type":"sine","frequency":1568},"envelope":{"attack":0.002,"decay":0.09,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.11},
        ],
      },
      "failed": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.03,"sustain":0,"release":0,"curve":"ramp"},"gain":0.04,"filter":{"type":"bandpass","frequency":617,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":146.83,"end":144.72,"time":0.02}},"envelope":{"attack":0.002,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.18},
          {"source":{"type":"sine","frequency":293.66},"envelope":{"attack":0.002,"decay":0.11,"sustain":0,"release":0,"curve":"ramp"},"gain":0.043},
          {"source":{"type":"sine","frequency":587.32},"envelope":{"attack":0.002,"decay":0.06,"sustain":0,"release":0,"curve":"ramp"},"gain":0.013},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.03,"sustain":0,"release":0,"curve":"ramp"},"gain":0.04,"delay":0.13,"filter":{"type":"bandpass","frequency":582,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":138.59,"end":136.6,"time":0.02}},"envelope":{"attack":0.002,"decay":0.26,"sustain":0,"release":0,"curve":"ramp"},"gain":0.17,"delay":0.13},
          {"source":{"type":"sine","frequency":277.18},"envelope":{"attack":0.002,"decay":0.143,"sustain":0,"release":0,"curve":"ramp"},"gain":0.041,"delay":0.13},
          {"source":{"type":"sine","frequency":554.36},"envelope":{"attack":0.002,"decay":0.078,"sustain":0,"release":0,"curve":"ramp"},"gain":0.012,"delay":0.13},
        ],
      },
      "task_completed": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1646,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":392,"end":386.38,"time":0.02}},"envelope":{"attack":0.002,"decay":0.22,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":784},"envelope":{"attack":0.002,"decay":0.121,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1568},"envelope":{"attack":0.002,"decay":0.066,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.08,"filter":{"type":"bandpass","frequency":2198,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":523.25,"end":515.75,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.08},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.08},
          {"source":{"type":"sine","frequency":2093},"envelope":{"attack":0.002,"decay":0.072,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.08},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.16,"filter":{"type":"bandpass","frequency":2769,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":659.25,"end":649.8,"time":0.02}},"envelope":{"attack":0.002,"decay":0.26,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.16},
          {"source":{"type":"sine","frequency":1318.5},"envelope":{"attack":0.002,"decay":0.143,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.16},
          {"source":{"type":"sine","frequency":2637},"envelope":{"attack":0.002,"decay":0.078,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.16},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.24,"filter":{"type":"bandpass","frequency":3293,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":783.99,"end":772.75,"time":0.02}},"envelope":{"attack":0.002,"decay":0.4,"sustain":0,"release":0,"curve":"ramp"},"gain":0.17,"delay":0.24},
          {"source":{"type":"sine","frequency":1567.98},"envelope":{"attack":0.002,"decay":0.22,"sustain":0,"release":0,"curve":"ramp"},"gain":0.041,"delay":0.24},
          {"source":{"type":"sine","frequency":3135.96},"envelope":{"attack":0.002,"decay":0.12,"sustain":0,"release":0,"curve":"ramp"},"gain":0.012,"delay":0.24},
        ],
      },
      "task_cancelled": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1233,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":293.66,"end":289.45,"time":0.02}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":587.32},"envelope":{"attack":0.002,"decay":0.099,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1174.64},"envelope":{"attack":0.002,"decay":0.054,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.12,"filter":{"type":"bandpass","frequency":1037,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":246.94,"end":243.4,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.12},
          {"source":{"type":"sine","frequency":493.88},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.036,"delay":0.12},
          {"source":{"type":"sine","frequency":987.76},"envelope":{"attack":0.002,"decay":0.072,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.12},
        ],
      },
      "team_invitation": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1467,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":349.23,"end":344.22,"time":0.02}},"envelope":{"attack":0.002,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":698.46},"envelope":{"attack":0.002,"decay":0.11,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1396.92},"envelope":{"attack":0.002,"decay":0.06,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.1,"filter":{"type":"bandpass","frequency":2198,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":523.25,"end":515.75,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.1},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.1},
          {"source":{"type":"sine","frequency":2093},"envelope":{"attack":0.002,"decay":0.072,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.1},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.2,"filter":{"type":"bandpass","frequency":2934,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":698.46,"end":688.45,"time":0.02}},"envelope":{"attack":0.002,"decay":0.32,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.2},
          {"source":{"type":"sine","frequency":1396.92},"envelope":{"attack":0.002,"decay":0.176,"sustain":0,"release":0,"curve":"ramp"},"gain":0.036,"delay":0.2},
          {"source":{"type":"sine","frequency":2793.84},"envelope":{"attack":0.002,"decay":0.096,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.2},
        ],
      },
      "task_comment": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1848,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":440,"end":433.69,"time":0.02}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.002,"decay":0.099,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.002,"decay":0.054,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.12,"filter":{"type":"bandpass","frequency":2467,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":587.33,"end":578.91,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.12},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.12},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.002,"decay":0.072,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.12},
        ],
      },
      "task_mention": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1848,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":440,"end":433.69,"time":0.02}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.002,"decay":0.099,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.002,"decay":0.054,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.12,"filter":{"type":"bandpass","frequency":2467,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":587.33,"end":578.91,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.12},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.12},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.002,"decay":0.072,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.12},
        ],
      },
      "prompt_proposal_created": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1568,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":523.25,"end":515.75,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.1,"filter":{"type":"bandpass","frequency":2093,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":698.46,"end":688.45,"time":0.02}},"envelope":{"attack":0.002,"decay":0.3,"sustain":0,"release":0,"curve":"ramp"},"gain":0.15,"delay":0.1},
          {"source":{"type":"sine","frequency":1396.92},"envelope":{"attack":0.002,"decay":0.165,"sustain":0,"release":0,"curve":"ramp"},"gain":0.036,"delay":0.1},
        ],
      },
      "default": {
        layers: [
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"bandpass","frequency":1848,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":440,"end":433.69,"time":0.02}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16},
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.002,"decay":0.099,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.002,"decay":0.054,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.014,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"delay":0.12,"filter":{"type":"bandpass","frequency":2467,"Q":1.4}},
          {"source":{"type":"triangle","frequency":{"start":587.33,"end":578.91,"time":0.02}},"envelope":{"attack":0.002,"decay":0.24,"sustain":0,"release":0,"curve":"ramp"},"gain":0.16,"delay":0.12},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.132,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.12},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.002,"decay":0.072,"sustain":0,"release":0,"curve":"ramp"},"gain":0.011,"delay":0.12},
        ],
      },
    },
  },
  {
    id: "glass",
    nameKey: "soundSet.glass",
    descriptionKey: "soundSet.glassHint",
    volume: 1,
    patches: {
      "task_started": {
        layers: [
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.11,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1567.98},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.046},
          {"source":{"type":"sine","frequency":2163.8124},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.022},
          {"source":{"type":"sine","frequency":4233.546},"envelope":{"attack":0.002,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.008},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.004,"decay":0.42,"sustain":0,"release":0,"curve":"ramp"},"gain":0.07,"delay":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.003,"decay":0.294,"sustain":0,"release":0,"curve":"ramp"},"gain":0.029,"delay":0.1},
          {"source":{"type":"sine","frequency":3242.0616},"envelope":{"attack":0.003,"decay":0.21,"sustain":0,"release":0,"curve":"ramp"},"gain":0.014,"delay":0.1},
          {"source":{"type":"sine","frequency":6343.164000000001},"envelope":{"attack":0.002,"decay":0.118,"sustain":0,"release":0,"curve":"ramp"},"gain":0.005,"delay":0.1},
        ],
      },
      "input_requested": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.004,"decay":0.45,"sustain":0,"release":0,"curve":"ramp"},"gain":0.11,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.003,"decay":0.315,"sustain":0,"release":0,"curve":"ramp"},"gain":0.046},
          {"source":{"type":"sine","frequency":2428.7999999999997},"envelope":{"attack":0.003,"decay":0.225,"sustain":0,"release":0,"curve":"ramp"},"gain":0.022},
          {"source":{"type":"sine","frequency":4752},"envelope":{"attack":0.002,"decay":0.126,"sustain":0,"release":0,"curve":"ramp"},"gain":0.008},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.004,"decay":0.55,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.11,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.003,"decay":0.385,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042,"delay":0.11},
          {"source":{"type":"sine","frequency":3242.0616},"envelope":{"attack":0.003,"decay":0.275,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02,"delay":0.11},
          {"source":{"type":"sine","frequency":6343.164000000001},"envelope":{"attack":0.002,"decay":0.154,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007,"delay":0.11},
        ],
      },
      "task_finished": {
        layers: [
          {"source":{"type":"sine","frequency":987.77},"envelope":{"attack":0.004,"decay":0.42,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1975.54},"envelope":{"attack":0.003,"decay":0.294,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":2726.2452},"envelope":{"attack":0.003,"decay":0.21,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":5333.9580000000005},"envelope":{"attack":0.002,"decay":0.118,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.62,"sustain":0,"release":0,"curve":"ramp"},"gain":0.11,"delay":0.12,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1567.98},"envelope":{"attack":0.003,"decay":0.434,"sustain":0,"release":0,"curve":"ramp"},"gain":0.046,"delay":0.12},
          {"source":{"type":"sine","frequency":2163.8124},"envelope":{"attack":0.003,"decay":0.31,"sustain":0,"release":0,"curve":"ramp"},"gain":0.022,"delay":0.12},
          {"source":{"type":"sine","frequency":4233.546},"envelope":{"attack":0.002,"decay":0.174,"sustain":0,"release":0,"curve":"ramp"},"gain":0.008,"delay":0.12},
        ],
      },
      "failed": {
        layers: [
          {"source":{"type":"sine","frequency":196},"envelope":{"attack":0.004,"decay":0.55,"sustain":0,"release":0,"curve":"ramp"},"gain":0.14,"effects":[{"type":"reverb","decay":0.55,"damping":0.4,"mix":0.16,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":392},"envelope":{"attack":0.003,"decay":0.385,"sustain":0,"release":0,"curve":"ramp"},"gain":0.059},
          {"source":{"type":"sine","frequency":540.9599999999999},"envelope":{"attack":0.003,"decay":0.275,"sustain":0,"release":0,"curve":"ramp"},"gain":0.028},
          {"source":{"type":"sine","frequency":1058.4},"envelope":{"attack":0.002,"decay":0.154,"sustain":0,"release":0,"curve":"ramp"},"gain":0.01},
          {"source":{"type":"sine","frequency":207.65},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.11,"delay":0.06,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":415.3},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.046,"delay":0.06},
          {"source":{"type":"sine","frequency":573.1139999999999},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.022,"delay":0.06},
          {"source":{"type":"sine","frequency":1121.3100000000002},"envelope":{"attack":0.002,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.008,"delay":0.06},
        ],
      },
      "task_completed": {
        layers: [
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.004,"decay":0.4,"sustain":0,"release":0,"curve":"ramp"},"gain":0.09,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.003,"decay":0.28,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038},
          {"source":{"type":"sine","frequency":1444.1699999999998},"envelope":{"attack":0.003,"decay":0.2,"sustain":0,"release":0,"curve":"ramp"},"gain":0.018},
          {"source":{"type":"sine","frequency":2825.55},"envelope":{"attack":0.002,"decay":0.112,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006},
          {"source":{"type":"sine","frequency":659.25},"envelope":{"attack":0.004,"decay":0.45,"sustain":0,"release":0,"curve":"ramp"},"gain":0.09,"delay":0.07,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1318.5},"envelope":{"attack":0.003,"decay":0.315,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.07},
          {"source":{"type":"sine","frequency":1819.53},"envelope":{"attack":0.003,"decay":0.225,"sustain":0,"release":0,"curve":"ramp"},"gain":0.018,"delay":0.07},
          {"source":{"type":"sine","frequency":3559.9500000000003},"envelope":{"attack":0.002,"decay":0.126,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006,"delay":0.07},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.09,"delay":0.14,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1567.98},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.038,"delay":0.14},
          {"source":{"type":"sine","frequency":2163.8124},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.018,"delay":0.14},
          {"source":{"type":"sine","frequency":4233.546},"envelope":{"attack":0.002,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006,"delay":0.14},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.004,"decay":0.75,"sustain":0,"release":0,"curve":"ramp"},"gain":0.08,"delay":0.21,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":2093},"envelope":{"attack":0.003,"decay":0.525,"sustain":0,"release":0,"curve":"ramp"},"gain":0.034,"delay":0.21},
          {"source":{"type":"sine","frequency":2888.3399999999997},"envelope":{"attack":0.003,"decay":0.375,"sustain":0,"release":0,"curve":"ramp"},"gain":0.016,"delay":0.21},
          {"source":{"type":"sine","frequency":5651.1},"envelope":{"attack":0.002,"decay":0.21,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006,"delay":0.21},
        ],
      },
      "task_cancelled": {
        layers: [
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.004,"decay":0.36,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.003,"decay":0.252,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":1444.1699999999998},"envelope":{"attack":0.003,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":2825.55},"envelope":{"attack":0.002,"decay":0.101,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007},
          {"source":{"type":"sine","frequency":392},"envelope":{"attack":0.004,"decay":0.55,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.11,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":784},"envelope":{"attack":0.003,"decay":0.385,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042,"delay":0.11},
          {"source":{"type":"sine","frequency":1081.9199999999998},"envelope":{"attack":0.003,"decay":0.275,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02,"delay":0.11},
          {"source":{"type":"sine","frequency":2116.8},"envelope":{"attack":0.002,"decay":0.154,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007,"delay":0.11},
        ],
      },
      "team_invitation": {
        layers: [
          {"source":{"type":"sine","frequency":659.25},"envelope":{"attack":0.004,"decay":0.42,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1318.5},"envelope":{"attack":0.003,"decay":0.294,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":1819.53},"envelope":{"attack":0.003,"decay":0.21,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":3559.9500000000003},"envelope":{"attack":0.002,"decay":0.118,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007},
          {"source":{"type":"sine","frequency":987.77},"envelope":{"attack":0.004,"decay":0.6,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1975.54},"envelope":{"attack":0.003,"decay":0.42,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042,"delay":0.1},
          {"source":{"type":"sine","frequency":2726.2452},"envelope":{"attack":0.003,"decay":0.3,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02,"delay":0.1},
          {"source":{"type":"sine","frequency":5333.9580000000005},"envelope":{"attack":0.002,"decay":0.168,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007,"delay":0.1},
        ],
      },
      "task_comment": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.004,"decay":0.45,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.003,"decay":0.315,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":2428.7999999999997},"envelope":{"attack":0.003,"decay":0.225,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":4752},"envelope":{"attack":0.002,"decay":0.126,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.08,"delay":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.034,"delay":0.1},
          {"source":{"type":"sine","frequency":3242.0616},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.016,"delay":0.1},
          {"source":{"type":"sine","frequency":6343.164000000001},"envelope":{"attack":0.002,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006,"delay":0.1},
        ],
      },
      "task_mention": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.004,"decay":0.45,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.003,"decay":0.315,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":2428.7999999999997},"envelope":{"attack":0.003,"decay":0.225,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":4752},"envelope":{"attack":0.002,"decay":0.126,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.08,"delay":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.034,"delay":0.1},
          {"source":{"type":"sine","frequency":3242.0616},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.016,"delay":0.1},
          {"source":{"type":"sine","frequency":6343.164000000001},"envelope":{"attack":0.002,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006,"delay":0.1},
        ],
      },
      "prompt_proposal_created": {
        layers: [
          {"source":{"type":"sine","frequency":523.25},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1046.5},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":1444.1699999999998},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":783.99},"envelope":{"attack":0.004,"decay":0.6,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1567.98},"envelope":{"attack":0.003,"decay":0.42,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042,"delay":0.1},
          {"source":{"type":"sine","frequency":2163.8124},"envelope":{"attack":0.003,"decay":0.3,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02,"delay":0.1},
        ],
      },
      "default": {
        layers: [
          {"source":{"type":"sine","frequency":880},"envelope":{"attack":0.004,"decay":0.45,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":1760},"envelope":{"attack":0.003,"decay":0.315,"sustain":0,"release":0,"curve":"ramp"},"gain":0.042},
          {"source":{"type":"sine","frequency":2428.7999999999997},"envelope":{"attack":0.003,"decay":0.225,"sustain":0,"release":0,"curve":"ramp"},"gain":0.02},
          {"source":{"type":"sine","frequency":4752},"envelope":{"attack":0.002,"decay":0.126,"sustain":0,"release":0,"curve":"ramp"},"gain":0.007},
          {"source":{"type":"sine","frequency":1174.66},"envelope":{"attack":0.004,"decay":0.5,"sustain":0,"release":0,"curve":"ramp"},"gain":0.08,"delay":0.1,"effects":[{"type":"reverb","decay":0.5,"damping":0.25,"mix":0.18,"preDelay":0.01,"roomSize":1}]},
          {"source":{"type":"sine","frequency":2349.32},"envelope":{"attack":0.003,"decay":0.35,"sustain":0,"release":0,"curve":"ramp"},"gain":0.034,"delay":0.1},
          {"source":{"type":"sine","frequency":3242.0616},"envelope":{"attack":0.003,"decay":0.25,"sustain":0,"release":0,"curve":"ramp"},"gain":0.016,"delay":0.1},
          {"source":{"type":"sine","frequency":6343.164000000001},"envelope":{"attack":0.002,"decay":0.14,"sustain":0,"release":0,"curve":"ramp"},"gain":0.006,"delay":0.1},
        ],
      },
    },
  },
  {
    id: "arcade",
    nameKey: "soundSet.arcade",
    descriptionKey: "soundSet.arcadeHint",
    volume: 1.4,
    patches: {
      "task_started": {
        layers: [
          {"source":{"type":"triangle","frequency":261.63},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.06},
          {"source":{"type":"square","frequency":523.25},"envelope":{"attack":0.002,"decay":0.07,"sustain":0,"release":0,"curve":"ramp"},"gain":0.11},
          {"source":{"type":"square","frequency":659.25},"envelope":{"attack":0.002,"decay":0.12,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.08},
        ],
      },
      "input_requested": {
        layers: [
          {"source":{"type":"triangle","frequency":440},"envelope":{"attack":0.002,"decay":0.06,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05},
          {"source":{"type":"triangle","frequency":880},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"triangle","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.09},
        ],
      },
      "task_finished": {
        layers: [
          {"source":{"type":"square","frequency":987.77},"envelope":{"attack":0.002,"decay":0.09,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":783.99},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.09},
        ],
      },
      "failed": {
        layers: [
          {"source":{"type":"square","frequency":{"start":196,"end":146.83,"time":0.14}},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.13},
          {"source":{"type":"noise","color":"white"},"envelope":{"attack":0.001,"decay":0.05,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05,"filter":{"type":"lowpass","frequency":900,"Q":0.8}},
        ],
      },
      "task_completed": {
        layers: [
          {"source":{"type":"triangle","frequency":261.63},"envelope":{"attack":0.002,"decay":0.12,"sustain":0,"release":0,"curve":"ramp"},"gain":0.05},
          {"source":{"type":"square","frequency":523.25},"envelope":{"attack":0.002,"decay":0.07,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":659.25},"envelope":{"attack":0.002,"decay":0.07,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.06},
          {"source":{"type":"square","frequency":783.99},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.12},
          {"source":{"type":"square","frequency":1046.5},"envelope":{"attack":0.002,"decay":0.28,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.18},
        ],
      },
      "task_cancelled": {
        layers: [
          {"source":{"type":"square","frequency":440},"envelope":{"attack":0.002,"decay":0.07,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":349.23},"envelope":{"attack":0.002,"decay":0.15,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.09},
        ],
      },
      "team_invitation": {
        layers: [
          {"source":{"type":"triangle","frequency":587.33},"envelope":{"attack":0.002,"decay":0.07,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"triangle","frequency":880},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.07},
          {"source":{"type":"triangle","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.18,"sustain":0,"release":0,"curve":"ramp"},"gain":0.09,"delay":0.14},
        ],
      },
      "task_comment": {
        layers: [
          {"source":{"type":"square","frequency":880},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.09},
        ],
      },
      "task_mention": {
        layers: [
          {"source":{"type":"square","frequency":880},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.09},
        ],
      },
      "prompt_proposal_created": {
        layers: [
          {"source":{"type":"triangle","frequency":523.25},"envelope":{"attack":0.002,"decay":0.07,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":659.25},"envelope":{"attack":0.002,"decay":0.09,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.08},
        ],
      },
      "default": {
        layers: [
          {"source":{"type":"square","frequency":880},"envelope":{"attack":0.002,"decay":0.08,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1},
          {"source":{"type":"square","frequency":1174.66},"envelope":{"attack":0.002,"decay":0.16,"sustain":0,"release":0,"curve":"ramp"},"gain":0.1,"delay":0.09},
        ],
      },
    },
  },
];

const SETS_BY_ID = new Map(SOUND_SETS.map((set) => [set.id, set]));

export function isSoundSetId(value: unknown): value is string {
  return typeof value === "string" && SETS_BY_ID.has(value);
}

export function soundSetById(id: string): SoundSet {
  return SETS_BY_ID.get(id) ?? SETS_BY_ID.get(DEFAULT_SOUND_SET_ID)!;
}

export function soundKinds(): string[] {
  return [...ATTENTION_EVENTS.map((event) => event.kind), "default"];
}

export function patchFor(setId: string, kind: string): Patch {
  const set = soundSetById(setId);
  return (
    set.patches[kind] ??
    set.patches.default ??
    SETS_BY_ID.get(DEFAULT_SOUND_SET_ID)!.patches.default!
  );
}
