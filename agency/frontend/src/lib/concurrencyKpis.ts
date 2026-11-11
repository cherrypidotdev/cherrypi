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

import { t } from "@/i18n/translate";
import type {
  AgentConcurrencyBoard,
  ProjectConcurrencyBoard,
} from "@/lib/concurrency";

export interface ConcurrencyKpis {
  kind: "agent" | "project";
  heldLabel: string;
  heldHint: string;
  held: number;
  cap: number;
  heldDetail: string;
  waitingLabel: string;
  waiting: number;
  waitingHint: string;
  waitingDetail: string;

  thirdLabel: string;
  thirdHint: string;
  thirdValue: number;
  thirdDetail: string;
  caps: number;
  capsHint: string;
  capsDetail: string;
}

export function emptyConcurrencyKpis(): ConcurrencyKpis {
  return {
    kind: "agent",
    heldLabel: t("concurrencyKpis.slotsHeld"),
    heldHint: t("concurrencyKpis.thisAgent"),
    held: 0,
    cap: 0,
    heldDetail: "",
    waitingLabel: t("concurrencyKpis.waiting"),
    waiting: 0,
    waitingHint: "",
    waitingDetail: "",
    thirdLabel: t("concurrencyKpis.free"),
    thirdHint: "",
    thirdValue: 0,
    thirdDetail: "",
    caps: 0,
    capsHint: "",
    capsDetail: "",
  };
}

export function agentConcurrencyKpis(
  board: AgentConcurrencyBoard,
): ConcurrencyKpis {
  const fullCapDetail = [
    board.fullProjectCount
      ? t("concurrencyKpis.fullProjects", { count: board.fullProjectCount })
      : "",
    board.fullTagCount
      ? t("concurrencyKpis.fullTags", { count: board.fullTagCount })
      : "",
    board.fullProjectTagCount
      ? t("concurrencyKpis.fullProjectTags", {
          count: board.fullProjectTagCount,
        })
      : "",
  ]
    .filter(Boolean)
    .join(" · ");
  return {
    kind: "agent",
    heldLabel: t("concurrencyKpis.slotsHeld"),
    heldHint: t("concurrencyKpis.thisAgent"),
    held: board.overall.used,
    cap: board.overall.cap,
    heldDetail: t("concurrencyKpis.ofAgentCap", {
      percent:
        board.overall.cap > 0
          ? Math.round((board.overall.used / board.overall.cap) * 100)
          : 0,
    }),
    waitingLabel: t("concurrencyKpis.waiting"),
    waiting: board.waiting.length,
    waitingHint: t("concurrencyKpis.pinned"),
    waitingDetail: board.waiting.length
      ? t("concurrencyKpis.pinnedToAgent")
      : t("concurrencyKpis.nothingPinned"),
    thirdLabel: t("concurrencyKpis.free"),
    thirdHint: t("concurrencyKpis.slots"),
    thirdValue: board.free,
    thirdDetail: board.free
      ? t("concurrencyKpis.availableNow")
      : t("concurrencyKpis.noFreeSlots"),
    caps: board.fullCapCount,
    capsHint: t("concurrencyKpis.dimensions"),
    capsDetail: fullCapDetail || t("concurrencyKpis.noCapFull"),
  };
}

export function projectConcurrencyKpis(
  board: ProjectConcurrencyBoard,
): ConcurrencyKpis {
  const pinned = board.waiting.filter((item) => item.agent).length;
  const auto = board.waiting.length - pinned;
  return {
    kind: "project",
    heldLabel: t("projectConcurrency.heldHere"),
    heldHint: t("concurrencyKpis.projectSlots"),
    held: board.held,
    cap: board.capSum,
    heldDetail: t("concurrencyKpis.sumOfCaps", { count: board.memberCount }),
    waitingLabel: t("concurrencyKpis.waiting"),
    waiting: board.waiting.length,
    waitingHint: t("concurrencyKpis.queued"),
    waitingDetail: auto
      ? t("concurrencyKpis.pinnedAndAuto", { pinned, auto })
      : pinned
        ? t("concurrencyKpis.pinnedCount", { count: pinned })
        : t("concurrencyKpis.nothingWaiting"),
    thirdLabel: t("concurrencyKpis.memberAgents"),
    thirdHint: t("concurrencyKpis.serving"),
    thirdValue: board.memberCount,
    thirdDetail: t("concurrencyKpis.onlineCount", {
      count: board.onlineCount,
    }),
    caps: board.fullMemberCount,
    capsHint: t("concurrencyKpis.agentsHint"),
    capsDetail: t("concurrencyKpis.agentsFull", {
      full: board.fullMemberCount,
      total: board.memberCount,
    }),
  };
}
