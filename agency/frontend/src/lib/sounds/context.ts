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
 * One shared AudioContext for attention sounds.
 *
 * Adapted from vladfaust/procedural-sounds
 * (https://github.com/vladfaust/procedural-sounds, MIT, Copyright (c) 2026
 * Mick Cesanek), which in turn adapts @web-kits/audio. See
 * THIRD-PARTY-NOTICES.md in this directory.
 */

let ctx: AudioContext | null = null;

export function getContext(): AudioContext {
  if (!ctx || ctx.state === "closed") {
    ctx = new AudioContext();
  }
  return ctx;
}

export async function ensureReady(): Promise<AudioContext> {
  const audio = getContext();
  if (audio.state === "suspended") {
    await audio.resume();
  }
  return audio;
}

export function closeContext(): void {
  if (!ctx) return;
  try {
    void ctx.close();
  } catch {}
  ctx = null;
}
