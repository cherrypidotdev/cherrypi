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

/**
 * Titles for the console's collapsed "activity" blocks (a run of consecutive
 * thinking + tool items). One is picked per block, weighted so the plain
 * "Thinking" label stays the common case and the jokes are the exception.
 */
export interface ActivityPhrase {
  title: string;
  weight: number;
}

export const ACTIVITY_PHRASES: ActivityPhrase[] = [
  { title: "Thinking", weight: 30 },
  { title: "Crunching numbers", weight: 12 },
  { title: "Reticulating splines", weight: 10 },
  { title: "Bribing the hamster", weight: 9 },
  { title: "Consulting the rubber duck", weight: 8 },
  { title: "Herding electrons", weight: 7 },
  { title: "Sharpening pencils", weight: 6 },
  { title: "Brewing ideas", weight: 6 },
  { title: "Asking the oracle", weight: 5 },
  { title: "Warming up the abacus", weight: 5 },
  { title: "Chasing rabbit holes", weight: 4 },
  { title: "Untangling spaghetti", weight: 4 },
  { title: "Summoning the muse", weight: 3 },
  { title: "Negotiating with the compiler", weight: 3 },
  { title: "Counting to infinity", weight: 2 },
  { title: "Doing important science", weight: 1 },
];

const TOTAL_WEIGHT = ACTIVITY_PHRASES.reduce(
  (sum, phrase) => sum + phrase.weight,
  0,
);

/** Weighted-random pick; `random` is injectable for tests. */
export function pickActivityTitle(random: () => number = Math.random): string {
  let roll = random() * TOTAL_WEIGHT;
  for (const phrase of ACTIVITY_PHRASES) {
    roll -= phrase.weight;
    if (roll < 0) return phrase.title;
  }
  return ACTIVITY_PHRASES[0]!.title;
}
