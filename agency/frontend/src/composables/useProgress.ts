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

import { useNProgress } from "@vueuse/integrations/useNProgress";

const { start, done } = useNProgress();

// NProgress has no request nesting of its own; keep a counter so overlapping
// operations don't hide the bar early.
let active = 0;

/**
 * Runs `fn` with the top progress bar active. Wraps the dashboard's
 * presumably-long-waited operations (creating a task, sending a prompt,
 * re-running a failure, credential changes, webhook tests, …).
 */
export async function withProgress<T>(fn: () => Promise<T>): Promise<T> {
  active += 1;
  start();
  try {
    return await fn();
  } finally {
    active -= 1;
    if (active === 0) done();
  }
}
