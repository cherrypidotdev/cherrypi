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

import {
  parseQuestionnaire,
  parseQuestionnaireReview,
  type QuestionnaireQuestion,
  type QuestionnaireReview,
} from "@/lib/questionnaire";
import { ModelWait_Kind, ToolPhase, type TaskEvent } from "@cherrypi/sdk";

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

export const SYSTEM_SENDER: MessageSender = {
  username: "system",
  name: "System",
  kind: "system",
};

/** One rendered line in the task console. */
export type Item =
  | {
      id: number;
      kind: "user";
      text: string;

      attachmentIds: string[];

      sender: MessageSender;

      system?: boolean;

      at?: number;
    }
  | { id: number; kind: "text"; text: string; at?: number }
  | ({ id: number; kind: "thinking"; text: string } & ActivityTiming)
  | ({
      id: number;
      kind: "wait";

      waitKind: ModelWait_Kind;

      model: string;

      provider: string;

      durationMs: number;

      error: string;
      attempt: number;

      reason: string;
    } & ActivityTiming)
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

      interrupted?: boolean;
    } & ActivityTiming)
  | { id: number; kind: "log"; text: string; at?: number };

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

  system?: boolean;
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
  const sender = normalizeSender(value.sender);
  return {
    text: value.body || value.text,
    attachmentIds: (value.attachments ?? []).map((a) => String(a.id)),
    sender,
    system: sender?.kind === "system",
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
    system: sender.kind === "system",
  };
}

export function queuedMessageParts(text: string): UserMessageParts {
  return parsePromptEnvelope(text) ?? userMessageParts({ text });
}

export function isBlankDelta(delta: string): boolean {
  return delta.trim() === "";
}

export function newItem(event: TaskEvent): NewItem | null {
  const payload = event.event;
  const at = timestampMs(event.at);
  switch (payload.case) {
    case "text":
      if (isBlankDelta(payload.value.delta)) return null;
      return { kind: "text", text: payload.value.delta, at };
    case "thinking":
      if (isBlankDelta(payload.value.delta)) return null;
      return {
        kind: "thinking",
        text: payload.value.delta,
        startedAt: at,
        endedAt: at,
      };
    case "userMessage": {
      const parts = userMessageParts(payload.value);
      return {
        kind: "user",
        ...parts,
        sender: parts.sender ?? SYSTEM_SENDER,
        system: parts.system === true,
        at,
      };
    }
    case "log":
      return { kind: "log", text: payload.value.line, at };
    case "modelWait": {
      const wait = payload.value;
      const durationMs = Number(wait.durationMs);
      if (durationMs <= 0) return null;
      return {
        kind: "wait",
        waitKind: wait.kind,
        model: wait.model,
        provider: wait.provider,
        durationMs,
        error: wait.error,
        attempt: wait.attempt,
        reason: wait.reason,
        startedAt: at !== undefined ? at - durationMs : undefined,
        endedAt: at,
      };
    }
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

export type ActivityItem = Extract<
  Item,
  { kind: "thinking" | "tool" | "wait" }
>;
export type WaitItem = Extract<Item, { kind: "wait" }>;
export type PlainItem = Extract<Item, { kind: "user" | "text" | "log" }>;

function isActivity(item: Item): item is ActivityItem {
  return (
    item.kind === "thinking" || item.kind === "tool" || item.kind === "wait"
  );
}

export function questionnaireReview(item: Item): QuestionnaireReview | null {
  if (
    item.kind !== "tool" ||
    item.name !== "ask_user_question" ||
    item.phase !== ToolPhase.FINISHED
  ) {
    return null;
  }
  const review = parseQuestionnaireReview(item.args, item.details);
  if (!review) {
    return null;
  }

  return { ...review, answeredAt: item.endedAt };
}

export interface StateChange {
  key: string;

  at?: number;

  from: string;
  to: string;

  reason: string;

  actor: string;
}

export type Group =
  | { id: number; kind: "single"; item: PlainItem }
  | { id: number; kind: "activity"; items: ActivityItem[] }
  | { id: number; kind: "state"; change: StateChange }
  | ({ id: number; kind: "questionnaire" } & QuestionnaireReview);

export function groupEnd(
  group: { id: number; kind: string },
  items: Item[],
): { at?: number; open: boolean } {
  if (group.kind === "activity") {
    const activity = group as Extract<Group, { kind: "activity" }>;
    let at: number | undefined;
    let seen = false;
    for (const item of activity.items) {
      if (item.startedAt !== undefined || item.endedAt !== undefined) {
        seen = true;
      }

      if (
        item.kind === "tool" &&
        item.phase !== ToolPhase.FINISHED &&
        item.interrupted !== true
      ) {
        return { at, open: true };
      }
      const end = item.endedAt ?? item.startedAt;
      if (end !== undefined && (at === undefined || end > at)) at = end;
    }
    return { at: seen ? at : undefined, open: false };
  }
  if (group.kind === "state") {
    return {
      at: (group as Extract<Group, { kind: "state" }>).change.at,
      open: false,
    };
  }
  if (group.kind === "questionnaire") {
    const review = group as Extract<Group, { kind: "questionnaire" }>;
    return { at: review.answeredAt, open: review.answeredAt === undefined };
  }
  if (group.kind === "questionnaire-live") {
    return { at: undefined, open: true };
  }
  const item = items.find((candidate) => candidate.id === group.id);
  if (!item) return { at: undefined, open: false };
  if (
    item.kind === "thinking" ||
    item.kind === "tool" ||
    item.kind === "wait"
  ) {
    return {
      at: item.endedAt ?? item.startedAt,
      open: item.endedAt === undefined,
    };
  }
  return { at: item.at, open: false };
}

const STATE_GROUP_ID_BASE = -2;

export function mergeStateChanges<T extends { id: number; kind: string }>(
  groups: T[],
  changes: StateChange[],
  items: Item[],
): (T | Extract<Group, { kind: "state" }>)[] {
  if (!changes.length) return groups;
  const out: (T | Extract<Group, { kind: "state" }>)[] = [...groups];
  const rows: Extract<Group, { kind: "state" }>[] = changes.map(
    (change, index) => ({
      id: STATE_GROUP_ID_BASE - index,
      kind: "state",
      change,
    }),
  );
  for (const row of rows) {
    if (row.kind !== "state") continue;
    const at = row.change.at;
    let index = out.length;
    if (at !== undefined) {
      index = out.findIndex((group) => {
        const end = groupEnd(group, items);
        if (end.open) return true;
        return end.at !== undefined && end.at > at;
      });
      if (index < 0) index = out.length;
    }
    out.splice(index, 0, row);
  }
  return out;
}

export function groupItems(items: Item[]): Group[] {
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
      });
    }
  }
  return groups;
}

export interface RunUsage {
  provider: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  activeMs: number;

  thinkingMs: number;

  modelWaitMs: number;

  at?: number;
}

export function runUsageByMessage(
  items: Item[],
  runs: RunUsage[],
): Map<number, RunUsage> {
  const texts = items.filter(
    (item): item is Extract<Item, { kind: "text" }> =>
      item.kind === "text" && item.at !== undefined,
  );
  const settled = runs
    .filter((run): run is RunUsage & { at: number } => run.at !== undefined)
    .sort((a, b) => a.at - b.at);

  const byMessage = new Map<number, RunUsage>();
  let cursor = 0;
  for (const run of settled) {
    let last: Extract<Item, { kind: "text" }> | undefined;
    while (cursor < texts.length && (texts[cursor]?.at ?? 0) <= run.at) {
      last = texts[cursor];
      cursor++;
    }
    if (last) byMessage.set(last.id, run);
  }
  return byMessage;
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

  if (incoming.phase === ToolPhase.FINISHED) existing.interrupted = false;
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

export function itemElapsed(
  item: ActivityItem,
  now?: number,
): number | undefined {
  if (item.startedAt === undefined) return undefined;
  const end = item.endedAt ?? now;
  if (end === undefined) return undefined;
  return end - item.startedAt;
}

export function groupWaitingMs(items: ActivityItem[]): number {
  let total = 0;
  for (const item of items) {
    if (item.kind === "wait" && item.durationMs >= WAIT_RENDER_MS) {
      total += item.durationMs;
    }
  }
  return total;
}

export const WAIT_RENDER_MS = 100;

export function absorbsWait(item: Item, next: Item | undefined): boolean {
  return (
    item.kind === "wait" &&
    item.waitKind === ModelWait_Kind.MODEL_REQUEST &&
    next?.kind === "thinking"
  );
}

export function groupElapsed(
  items: ActivityItem[],
  now?: number,
): number | undefined {
  let start: number | undefined;
  let end: number | undefined;
  for (const item of items) {
    if (
      item.startedAt !== undefined &&
      (start === undefined || item.startedAt < start)
    ) {
      start = item.startedAt;
    }
    const itemEnd =
      item.endedAt ??
      (item.kind === "tool" &&
      item.phase !== ToolPhase.FINISHED &&
      item.interrupted !== true
        ? now
        : undefined);
    if (itemEnd !== undefined && (end === undefined || itemEnd > end)) {
      end = itemEnd;
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
