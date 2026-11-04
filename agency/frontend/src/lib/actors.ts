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

import type { MessageSender } from "@/lib/console";
import type { RouteLocationRaw } from "vue-router";

export interface ActorRef {
  kind?: string;
  username?: string;
}

export function actorRoute(
  actor: ActorRef | undefined,
): RouteLocationRaw | undefined {
  const username = actor?.username?.trim() ?? "";
  if (!username || actor?.kind === "system") return undefined;
  return {
    name: actor?.kind === "agent" ? "agent" : "user",
    params: { username },
  };
}

export function namedSender(
  sender: MessageSender | undefined,
  names: Map<string, string>,
): MessageSender | undefined {
  if (!sender || sender.kind === "system" || !names.size) return sender;
  if (sender.name && sender.name !== sender.username) return sender;
  const known =
    (sender.id ? names.get(sender.id) : undefined) ??
    names.get(sender.username);
  return known ? { ...sender, name: known } : sender;
}
