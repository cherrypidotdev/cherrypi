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

import { clampSoundVolume, patchFor, soundSetById } from "./sets";
import { playPatch, type PlayHandle } from "./synth";

let active: PlayHandle | null = null;
let generation = 0;

let interruptWait: (() => void) | null = null;

function stopActive() {
  active?.stop();
  active = null;
  interruptWait?.();
  interruptWait = null;
}

function effectiveVolume(setId: string, volume: number): number {
  return clampSoundVolume(volume) * soundSetById(setId).volume;
}

function wait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    let done = false;
    const finish = () => {
      if (done) return;
      done = true;
      window.clearTimeout(timer);
      if (interruptWait === finish) interruptWait = null;
      resolve();
    };
    const timer = window.setTimeout(finish, ms);
    interruptWait = finish;
  });
}

export async function playSound(
  setId: string,
  kind: string,
  volume: number,
): Promise<void> {
  const token = ++generation;
  stopActive();
  let handle: PlayHandle;
  try {
    handle = await playPatch(patchFor(setId, kind), {
      volume: effectiveVolume(setId, volume),
    });
  } catch {
    return;
  }
  if (token !== generation) {
    handle.stop();
    return;
  }
  active = handle;
  await wait(handle.duration * 1000);
  if (token === generation) active = null;
}

export function stopSound(): void {
  generation += 1;
  stopActive();
}

export async function previewSoundSet(
  setId: string,
  volume: number,
  kinds: string[],
  gapMs = 250,
): Promise<void> {
  const token = ++generation;
  stopActive();
  for (const kind of kinds) {
    if (token !== generation) return;
    let handle: PlayHandle;
    try {
      handle = await playPatch(patchFor(setId, kind), {
        volume: effectiveVolume(setId, volume),
      });
    } catch {
      return;
    }
    if (token !== generation) {
      handle.stop();
      return;
    }
    active = handle;
    await wait(Math.min(handle.duration, 0.9) * 1000 + gapMs);
  }
  if (token === generation) active = null;
}
