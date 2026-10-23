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

import { taskName } from "@/lib/task";
import type { Task } from "@cherrypi/sdk";

/**
 * Confirmation text for deleting a task. Deletion is irreversible, so the
 * caller should always confirm.
 */
export function deleteConfirmation(task: Task): string {
  const reasons = ["its workspace files will be removed"];
  if (task.runtime?.runInProgress) {
    reasons.push("its in-flight run will be stopped");
  }
  return `Delete "${taskName(task)}"? ${reasons.join(
    ", ",
  )}. Token usage history is kept. This cannot be undone.`;
}
