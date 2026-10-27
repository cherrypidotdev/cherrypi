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

import { timestampMs } from "@/lib/console";
import type { AgentLog, TerminalLog } from "@cherrypi/sdk";

export const DEFAULT_HIDDEN_LOG_PATTERNS = [
  "pi started",
  "pi exited",
  "pi stopped",
];

export type MergedLogRow =
  | { kind: "agent"; at: number; log: AgentLog }
  | { kind: "terminal"; at: number; log: TerminalLog };

export interface AgentLogFilter {
  search?: string;

  hiddenPatterns?: string[];
}

function messageMatches(message: string, needle: string): boolean {
  return message.toLowerCase().includes(needle.toLowerCase());
}

/** Whether an agent log line is hidden by one of the configured patterns. */
export function isHiddenAgentLog(
  log: AgentLog,
  hiddenPatterns: string[],
): boolean {
  return hiddenPatterns.some(
    (pattern) => pattern.trim() !== "" && messageMatches(log.message, pattern),
  );
}

function terminalText(log: TerminalLog): string {
  return [log.event, log.reason, log.detail].filter(Boolean).join(" ");
}

export function mergeLogRows(
  agentLogs: AgentLog[],
  terminalLogs: TerminalLog[],
  filter: AgentLogFilter = {},
): MergedLogRow[] {
  const search = filter.search?.trim() ?? "";
  const hidden = filter.hiddenPatterns ?? [];
  const rows: MergedLogRow[] = [];

  for (const log of agentLogs) {
    if (hidden.length && isHiddenAgentLog(log, hidden)) continue;
    if (
      search &&
      !messageMatches(`${log.message} ${log.taskId} ${log.sessionId}`, search)
    ) {
      continue;
    }
    rows.push({ kind: "agent", at: timestampMs(log.at) ?? 0, log });
  }
  for (const log of terminalLogs) {
    if (search && !messageMatches(terminalText(log), search)) continue;
    rows.push({ kind: "terminal", at: timestampMs(log.at) ?? 0, log });
  }

  return rows.sort((a, b) => b.at - a.at || Number(b.log.id - a.log.id));
}
