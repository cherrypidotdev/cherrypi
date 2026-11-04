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

/*
 * The recipe format for a procedural sound. A patch is plain data: a list of
 * layers, each with a source, an envelope, a gain, an optional filter and
 * optional effects. The player in `synth.ts` knows how to build a Web Audio
 * graph from it.
 *
 * Adapted from the `Patch` type in vladfaust/procedural-sounds
 * (https://github.com/vladfaust/procedural-sounds, MIT, Copyright (c) 2026
 * Mick Cesanek). The synth it feeds is adapted from @web-kits/audio and
 * cuelume; see THIRD-PARTY-NOTICES.md in this directory.
 */

export type Waveform = "sine" | "triangle" | "square" | "sawtooth";

export type Frequency = number | { start: number; end: number; time?: number };

export interface FM {
  ratio: number;
  depth: number;
}

export interface OscillatorSource {
  type: Waveform;
  frequency: Frequency;
  fm?: FM;
  detune?: number;
}

export type NoiseColor = "white" | "pink" | "brown";

export interface NoiseSource {
  type: "noise";
  color?: NoiseColor;
}

export type Source = OscillatorSource | NoiseSource;

export interface Envelope {
  attack?: number;
  decay: number;
  sustain?: number;
  release?: number;
  curve?: "ramp";
}

export interface FilterEnvelope {
  attack?: number;
  peak: number;
  decay: number;
}

export interface Filter {
  type: BiquadFilterType;
  frequency: number;
  Q?: number;
  resonance?: number;
  envelope?: FilterEnvelope;
}

export interface ReverbEffect {
  type: "reverb";
  decay?: number;
  damping?: number;
  mix?: number;
  preDelay?: number;
  roomSize?: number;
}

export interface DelayEffect {
  type: "delay";
  delay: number;
  feedback: number;
  wet: number;
  lowpass?: number;
}

export type Effect = ReverbEffect | DelayEffect;

export interface Layer {
  source: Source;
  envelope?: Envelope;
  gain?: number;
  delay?: number;
  filter?: Filter | Filter[];
  effects?: Effect[];
}

export type Patch = Layer | { layers: Layer[] };

export function layersOf(patch: Patch): Layer[] {
  return "layers" in patch ? patch.layers : [patch];
}
