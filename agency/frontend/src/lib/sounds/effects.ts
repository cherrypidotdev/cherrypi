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
 * Delay and reverb effects for the patch player.
 *
 * Adapted from vladfaust/procedural-sounds
 * (https://github.com/vladfaust/procedural-sounds, MIT, Copyright (c) 2026
 * Mick Cesanek), which in turn adapts @web-kits/audio and cuelume. See
 * THIRD-PARTY-NOTICES.md in this directory.
 */

import type { DelayEffect, ReverbEffect } from "./patch";

export interface EffectNode {
  input: AudioNode;
  output: AudioNode;
}

function withMix(
  ctx: BaseAudioContext,
  mix: number,
  create: (wet: GainNode, wetOut: GainNode) => void,
): EffectNode {
  const input = ctx.createGain();
  const output = ctx.createGain();

  const dry = ctx.createGain();
  dry.gain.value = 1 - mix;
  input.connect(dry);
  dry.connect(output);

  const wet = ctx.createGain();
  wet.gain.value = mix;
  input.connect(wet);

  const wetOut = ctx.createGain();
  wetOut.connect(output);

  create(wet, wetOut);

  return { input, output };
}

export function createShimmer(
  ctx: BaseAudioContext,
  opts: DelayEffect,
): EffectNode {
  const input = ctx.createGain();
  const output = ctx.createGain();
  input.connect(output);

  const delay = ctx.createDelay(1);
  delay.delayTime.value = opts.delay;

  const feedbackFilter = ctx.createBiquadFilter();
  feedbackFilter.type = "lowpass";
  feedbackFilter.frequency.value = opts.lowpass ?? 4000;

  const feedbackGain = ctx.createGain();
  feedbackGain.gain.value = opts.feedback;

  const wetGain = ctx.createGain();
  wetGain.gain.value = opts.wet;

  input.connect(delay);
  delay.connect(feedbackFilter);
  feedbackFilter.connect(feedbackGain);
  feedbackGain.connect(delay);
  feedbackFilter.connect(wetGain);
  wetGain.connect(output);

  return { input, output };
}

const INAUDIBLE_GAIN = 0.001;

export function shimmerTail(opts: DelayEffect): number {
  if (opts.feedback <= 0) return 0;
  if (opts.feedback >= 1) return opts.delay;
  return (
    opts.delay *
    (1 + Math.ceil(Math.log(INAUDIBLE_GAIN) / Math.log(opts.feedback)))
  );
}

export function createReverb(
  ctx: BaseAudioContext,
  opts: ReverbEffect,
): EffectNode {
  const decay = opts.decay ?? 0.5;
  const mix = opts.mix ?? 0.3;
  const preDelay = opts.preDelay ?? 0;
  const damping = opts.damping ?? 0;
  const roomSize = opts.roomSize ?? 1;

  return withMix(ctx, mix, (wet, wetOut) => {
    const sampleRate = ctx.sampleRate;
    const effectiveDecay = decay * roomSize;
    const length = Math.ceil(sampleRate * effectiveDecay);
    const buffer = ctx.createBuffer(2, length, sampleRate);

    for (let ch = 0; ch < 2; ch++) {
      const data = buffer.getChannelData(ch);
      for (let i = 0; i < length; i++) {
        data[i] = (Math.random() * 2 - 1) * Math.exp(-i / (length * 0.28));
      }
    }

    if (damping > 0) {
      for (let ch = 0; ch < 2; ch++) {
        const data = buffer.getChannelData(ch);
        const coeff = Math.min(damping, 0.99);
        let prev = 0;
        for (let i = 0; i < length; i++) {
          prev = data[i]! * (1 - coeff) + prev * coeff;
          data[i] = prev;
        }
      }
    }

    const convolver = ctx.createConvolver();
    convolver.buffer = buffer;

    if (preDelay > 0) {
      const preDelayNode = ctx.createDelay(Math.max(preDelay + 0.01, 1));
      preDelayNode.delayTime.value = preDelay;
      wet.connect(preDelayNode);
      preDelayNode.connect(convolver);
    } else {
      wet.connect(convolver);
    }
    convolver.connect(wetOut);
  });
}
