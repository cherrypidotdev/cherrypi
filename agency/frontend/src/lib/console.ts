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

import { parseAttachmentTrailer } from "@/lib/attachments";
import {
  parseQuestionnaire,
  parseQuestionnaireReview,
  type QuestionnaireQuestion,
  type QuestionnaireReview,
} from "@/lib/questionnaire";
import { ToolPhase, type TaskEvent } from "@cherrypi/sdk";

export function timestampMs(
  at: { seconds: bigint; nanos?: number } | undefined,
): number | undefined {
  if (!at || at.seconds === 0n) return undefined;
  return Number(at.seconds) * 1000 + Math.round((at.nanos ?? 0) / 1e6);
}

export interface ActivityTiming {
  startedAt?: number;
  endedAt?: number;
}

export interface MessageSender {
  id?: string;
  username: string;
  name: string;

  kind: string;
}

/** One rendered line in the task console. */
export type Item =
  | {
      id: number;
      kind: "user";
      text: string;

      attachmentIds: string[];

      sender?: MessageSender;
    }
  | { id: number; kind: "text"; text: string }
  | ({ id: number; kind: "thinking"; text: string } & ActivityTiming)
  | ({
      id: number;
      kind: "tool";
      callId: string;
      name: string;
      phase: ToolPhase;
      args: string;
      result: string;

      details: string;
      isError: boolean;
    } & ActivityTiming)
  | { id: number; kind: "log"; text: string };

type DistributiveOmit<T, K extends PropertyKey> = T extends unknown
  ? Omit<T, K>
  : never;

export type NewItem = DistributiveOmit<Item, "id">;
export type ToolItem = Extract<Item, { kind: "tool" }>;
export type ToolInput = Omit<ToolItem, "id">;

export interface UserMessageParts {
  text: string;
  attachmentIds: string[];
  sender?: MessageSender;
}

function normalizeSender(
  sender:
    | {
        id?: unknown;
        actorId?: bigint;
        name?: string;
        username?: string;
        kind?: string;
      }
    | undefined,
): MessageSender | undefined {
  const name = sender?.name?.trim() ?? "";
  const username = sender?.username?.trim() ?? "";
  if (!name && !username) return undefined;
  const rawId = sender?.actorId ?? sender?.id;
  const id =
    typeof rawId === "bigint"
      ? rawId === 0n
        ? ""
        : String(rawId)
      : typeof rawId === "number"
        ? rawId === 0
          ? ""
          : String(rawId)
        : typeof rawId === "string"
          ? rawId
          : "";
  return {
    ...(id ? { id } : {}),
    name,
    username,
    kind: sender?.kind?.trim() ?? "",
  };
}

export function userMessageParts(value: {
  text: string;
  body?: string;
  sender?: {
    actorId?: bigint;
    id?: unknown;
    name?: string;
    username?: string;
    kind?: string;
  };
  attachments?: { id: bigint; filename: string }[];
}): UserMessageParts {
  if (!value.body && !value.sender && !value.attachments?.length) {
    const envelope = parsePromptEnvelope(value.text);
    if (envelope) return envelope;
  }
  const raw = value.body || value.text;
  const structured = value.attachments ?? [];
  const parsed =
    structured.length > 0
      ? { text: raw, attachmentIds: structured.map((a) => String(a.id)) }
      : parseAttachmentTrailer(raw);
  return {
    text: parsed.text,
    attachmentIds: parsed.attachmentIds,
    sender: normalizeSender(value.sender),
  };
}

function firstJsonObjectEnd(text: string): number {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (let i = 0; i < text.length; i++) {
    const char = text[i];
    if (inString) {
      if (escaped) escaped = false;
      else if (char === "\\") escaped = true;
      else if (char === '"') inString = false;
      continue;
    }
    if (char === '"') inString = true;
    else if (char === "{") depth++;
    else if (char === "}" && --depth === 0) return i;
  }
  return -1;
}

export function parsePromptEnvelope(text: string): UserMessageParts | null {
  const trimmed = text.trimStart();
  if (!trimmed.startsWith("{")) return null;
  const end = firstJsonObjectEnd(trimmed);
  if (end < 0) return null;
  let value: unknown;
  try {
    value = JSON.parse(trimmed.slice(0, end + 1));
  } catch {
    return null;
  }
  if (typeof value !== "object" || value === null) return null;
  const env = value as { sendingActor?: unknown; messageBody?: unknown };
  const actor =
    typeof env.sendingActor === "object" && env.sendingActor !== null
      ? (env.sendingActor as Record<string, unknown>)
      : {};
  const sender = normalizeSender({
    id: actor.id,
    name: typeof actor.name === "string" ? actor.name : undefined,
    username: typeof actor.username === "string" ? actor.username : undefined,
    kind: typeof actor.kind === "string" ? actor.kind : undefined,
  });
  if (!sender) return null;
  const body =
    typeof env.messageBody === "object" && env.messageBody !== null
      ? (env.messageBody as Record<string, unknown>)
      : {};
  const attachments = Array.isArray(body.attachments) ? body.attachments : [];
  const attachmentIds: string[] = [];
  for (const attachment of attachments) {
    if (typeof attachment !== "object" || attachment === null) continue;
    const id = (attachment as Record<string, unknown>).id;
    if (typeof id === "number" || typeof id === "string") {
      attachmentIds.push(String(id));
    }
  }
  return {
    text: typeof body.text === "string" ? body.text : "",
    attachmentIds,
    sender,
  };
}

export function queuedMessageParts(text: string): UserMessageParts {
  return parsePromptEnvelope(text) ?? userMessageParts({ text });
}

export function newItem(event: TaskEvent): NewItem | null {
  const payload = event.event;
  const at = timestampMs(event.at);
  switch (payload.case) {
    case "text":
      return { kind: "text", text: payload.value.delta };
    case "thinking":
      return {
        kind: "thinking",
        text: payload.value.delta,
        startedAt: at,
        endedAt: at,
      };
    case "userMessage": {
      return { kind: "user", ...userMessageParts(payload.value) };
    }
    case "log":
      return { kind: "log", text: payload.value.line };
    case "tool": {
      const tool = payload.value;
      return {
        kind: "tool",
        callId: tool.toolCallId,
        name: tool.toolName,
        phase: tool.phase,
        args: tool.argsJson,
        result: tool.resultJson,
        details: tool.detailsJson,
        isError: tool.isError,
        startedAt: at,
        endedAt: tool.phase === ToolPhase.FINISHED ? at : undefined,
      };
    }
    default:
      return null;
  }
}

export type ActivityItem = Extract<Item, { kind: "thinking" | "tool" }>;
export type PlainItem = Extract<Item, { kind: "user" | "text" | "log" }>;

function isActivity(item: Item): item is ActivityItem {
  return item.kind === "thinking" || item.kind === "tool";
}

export function questionnaireReview(item: Item): QuestionnaireReview | null {
  if (
    item.kind !== "tool" ||
    item.name !== "ask_user_question" ||
    item.phase !== ToolPhase.FINISHED
  ) {
    return null;
  }
  return parseQuestionnaireReview(item.args, item.details);
}

export type Group =
  | { id: number; kind: "single"; item: PlainItem }
  | { id: number; kind: "activity"; items: ActivityItem[]; title: string }
  | ({ id: number; kind: "questionnaire" } & QuestionnaireReview);

export function groupItems(
  items: Item[],
  title: (id: number) => string,
): Group[] {
  const groups: Group[] = [];
  for (const item of items) {
    const review = questionnaireReview(item);
    if (review) {
      groups.push({ id: item.id, kind: "questionnaire", ...review });
      continue;
    }
    if (!isActivity(item)) {
      groups.push({ id: item.id, kind: "single", item });
      continue;
    }
    const last = groups[groups.length - 1];
    if (last && last.kind === "activity") {
      last.items.push(item);
    } else {
      groups.push({
        id: item.id,
        kind: "activity",
        items: [item],
        title: title(item.id),
      });
    }
  }
  return groups;
}

export function pinnedUserIndex(
  offsets: number[],
  scrollTop: number,
  tolerance = 16,
): number {
  let index = 0;
  for (let i = 0; i < offsets.length; i++) {
    if ((offsets[i] ?? 0) - scrollTop <= tolerance) {
      index = i;
    } else {
      break;
    }
  }
  return index;
}

export function findTool(items: Item[], callId: string): ToolItem | undefined {
  return items.find(
    (item): item is ToolItem => item.kind === "tool" && item.callId === callId,
  );
}

export function activeQuestionnaire(
  items: Item[],
): { callId: string; questions: QuestionnaireQuestion[] } | null {
  for (let i = items.length - 1; i >= 0; i--) {
    const item = items[i];
    if (item?.kind !== "tool" || item.name !== "ask_user_question") {
      continue;
    }
    if (item.phase === ToolPhase.FINISHED) {
      return null;
    }
    const parsed = parseQuestionnaire(item.args);
    return parsed ? { callId: item.callId, questions: parsed.questions } : null;
  }
  return null;
}

export function mergeTool(existing: ToolItem, incoming: ToolInput) {
  if (incoming.phase > existing.phase) {
    existing.phase = incoming.phase;
  }
  if (incoming.args) existing.args = incoming.args;
  if (incoming.result) existing.result = incoming.result;
  if (incoming.details) existing.details = incoming.details;
  existing.isError = existing.isError || incoming.isError;
  if (incoming.startedAt !== undefined) {
    existing.startedAt = Math.min(
      existing.startedAt ?? incoming.startedAt,
      incoming.startedAt,
    );
  }
  if (incoming.endedAt !== undefined) {
    existing.endedAt = Math.max(
      existing.endedAt ?? incoming.endedAt,
      incoming.endedAt,
    );
  }
}

export function itemElapsed(item: ActivityItem): number | undefined {
  if (item.startedAt === undefined || item.endedAt === undefined) {
    return undefined;
  }
  return item.endedAt - item.startedAt;
}

export function groupElapsed(items: ActivityItem[]): number | undefined {
  let start: number | undefined;
  let end: number | undefined;
  for (const item of items) {
    if (
      item.startedAt !== undefined &&
      (start === undefined || item.startedAt < start)
    ) {
      start = item.startedAt;
    }
    if (
      item.endedAt !== undefined &&
      (end === undefined || item.endedAt > end)
    ) {
      end = item.endedAt;
    }
  }
  if (start === undefined || end === undefined) return undefined;
  return end - start;
}

export function prependEvents(
  items: Item[],
  events: TaskEvent[],
  allocateId: () => number,
): Item[] {
  const fresh: Item[] = [];
  for (const event of events) {
    const item = newItem(event);
    if (!item) continue;

    if (item.kind === "tool") {
      const existing =
        findTool(items, item.callId) ?? findTool(fresh, item.callId);
      if (existing) {
        mergeTool(existing, item);
        continue;
      }
    }
    fresh.push({ ...item, id: allocateId() } as Item);
  }
  return fresh.length ? [...fresh, ...items] : items;
}
