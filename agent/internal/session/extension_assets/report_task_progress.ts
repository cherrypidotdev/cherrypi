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

import { Type } from "typebox";
import type {
  ExtensionAPI,
  ExtensionContext,
} from "@earendil-works/pi-coding-agent";

/**
 * CherryPi task progress.
 *
 * Registers `reportTaskProgress`, the tool a running task uses to publish its
 * plan: an optional one-line summary plus an ordered checklist. Each update
 * appends a custom Pi session entry and the CherryPi agent forwards it to the
 * agency as a `task_progress` event. The agency derives numeric progress from
 * the checklist (done/total), so the number always matches what is visible.
 *
 * The checklist doubles as the task's TODO list. Full replacement is
 * intentional: a call always sends the complete plan, so the model never has
 * to track item ids. Omitting `todos` reads the current plan back.
 *
 * Planning is mandatory, and a prompt is only a nudge: the extension enforces
 * it in code. A run that starts without a plan gets a hidden reminder, every
 * tool-using turn that does not touch the plan gets another (capped per run),
 * and `agent_before_settle` forces up to two continuations while the plan is
 * missing, unfinished, or stale. The reminders are `display: false` custom
 * messages: they reach the model without appearing as user turns.
 *
 * Every nudge says the same thing about what comes next: publish or update the
 * plan and carry on with the work in the same turn. A plan update is a
 * bookkeeping step, not a turn of its own, so a model that stops right after
 * `reportTaskProgress` still gets nudged at settle — but the reminder should
 * make that unnecessary.
 */
export default function (pi: ExtensionAPI) {
  const MAX_ITEMS = 50;
  const MAX_CHARS = 240;
  const CUSTOM_TYPE = "cherrypi_task_progress";
  const REMINDER_TYPE = "cherrypi_task_progress_reminder";

  const MAX_TURN_REMINDERS = 2;
  const MAX_SETTLE_NUDGES = 2;

  const CARRY_ON =
    "Then carry on with the work immediately in the same turn: a plan update " +
    "is bookkeeping, never a stopping point, so do not end the run right " +
    "after updating the plan.";

  type Status = "pending" | "in_progress" | "done";
  interface Todo {
    text: string;
    status: Status;
  }
  interface Progress {
    summary: string;
    todos: Todo[];
  }

  let current: Progress = { summary: "", todos: [] };

  let touchedThisTurn = false;
  let touchedThisRun = false;
  let turnReminders = 0;
  let settleNudges = 0;

  const clamp = (value: unknown): string => {
    const text = String(value ?? "").trim();
    const chars = Array.from(text);
    return chars.length > MAX_CHARS
      ? chars.slice(0, MAX_CHARS).join("").trim()
      : text;
  };

  const normalizeStatus = (value: unknown): Status => {
    const status = String(value ?? "")
      .trim()
      .toLowerCase();
    if (status === "in_progress" || status === "in-progress") {
      return "in_progress";
    }
    if (status === "done" || status === "completed") return "done";
    return "pending";
  };

  const normalize = (summary: unknown, todos: unknown): Progress => {
    const items: Todo[] = [];
    if (Array.isArray(todos)) {
      for (const raw of todos) {
        const item = raw as { text?: unknown; status?: unknown };
        const text = clamp(item?.text);
        if (!text) continue;
        items.push({ text, status: normalizeStatus(item?.status) });
        if (items.length === MAX_ITEMS) break;
      }
    }
    return { summary: clamp(summary), todos: items };
  };

  const hasPlan = (): boolean => current.todos.length > 0;
  const hasUnfinished = (): boolean =>
    current.todos.some((item) => item.status !== "done");

  const resetRun = (): void => {
    touchedThisTurn = false;
    touchedThisRun = false;
    turnReminders = 0;
    settleNudges = 0;
  };

  const reminder = (content: string) => ({
    type: "custom_message" as const,
    customType: REMINDER_TYPE,
    content,
    display: false,
  });

  const render = (heading: string): string => {
    if (current.todos.length === 0) {
      return current.summary
        ? `${heading}: no plan. Summary: ${current.summary}`
        : `${heading}: no plan yet.`;
    }
    const done = current.todos.filter((item) => item.status === "done").length;
    const active = current.todos.filter(
      (item) => item.status === "in_progress",
    ).length;
    const lines = current.todos.map((item) => {
      const mark =
        item.status === "done"
          ? "[x]"
          : item.status === "in_progress"
            ? "[~]"
            : "[ ]";
      return `${mark} ${item.text}`;
    });
    const summary = current.summary ? ` — ${current.summary}` : "";
    const warning =
      active > 1
        ? `\nNote: ${active} items are in_progress; prefer a single current step.`
        : "";
    return `${heading}: ${done}/${current.todos.length} done${summary}\n${lines.join("\n")}${warning}`;
  };

  const reconstruct = (ctx: ExtensionContext): void => {
    current = { summary: "", todos: [] };
    for (const entry of ctx.sessionManager.getBranch()) {
      if (entry.type !== "custom" || entry.customType !== CUSTOM_TYPE) continue;
      const data = entry.data as
        { summary?: unknown; todos?: unknown } | undefined;
      current = normalize(data?.summary, data?.todos);
    }
    resetRun();
  };

  pi.on("session_start", async (_event, ctx) => reconstruct(ctx));
  pi.on("session_tree", async (_event, ctx) => reconstruct(ctx));

  pi.on("before_agent_start", async () => {
    resetRun();
    if (hasPlan()) return;
    return {
      message: {
        customType: REMINDER_TYPE,
        content:
          "Plan reminder: this task has no plan yet. Publish one with " +
          "`reportTaskProgress` before doing the work: 3–8 ordered items, each " +
          "`pending` (a genuinely one-step task gets one item). " +
          CARRY_ON +
          " Keep it current afterwards: mark an item `in_progress` when you " +
          "begin it and `done` when it lands.",
        display: false,
      },
    };
  });

  pi.on("turn_start", async () => {
    touchedThisTurn = false;
  });

  pi.on("turn_end", async (event) => {
    if (event.outcome !== "completed") return;
    if (event.toolResults.length === 0) return;
    if (touchedThisTurn) return;
    if (turnReminders >= MAX_TURN_REMINDERS) return;
    turnReminders += 1;
    return {
      entries: [
        ...event.entries,
        reminder(
          "Plan reminder: you used tools this turn without updating the plan. " +
            "Call `reportTaskProgress` with the complete list: mark finished " +
            "items `done`, the step you are on `in_progress`, and keep the " +
            "summary to one line so the operator can see what you are doing. " +
            CARRY_ON,
        ),
      ],
    };
  });

  pi.on("agent_before_settle", async (event) => {
    if (settleNudges >= MAX_SETTLE_NUDGES) return;
    const missing = !hasPlan();
    const unfinished = hasUnfinished();
    const stale = !touchedThisRun;
    if (!missing && !unfinished && !stale) return;
    settleNudges += 1;
    const reason = missing
      ? "you have not published a plan for this task"
      : unfinished
        ? "your plan still has unfinished items"
        : "you did not update the plan during this run";
    return {
      entries: [
        ...event.entries,
        reminder(
          `Plan reminder: ${reason}. Before finishing, call ` +
            "`reportTaskProgress` with the complete list: mark every completed " +
            "step `done`, leave nothing you finished as `pending`, and replace " +
            "the list if the task changed. " +
            CARRY_ON,
        ),
      ],
      continue: true,
    };
  });

  pi.registerTool({
    name: "reportTaskProgress",
    label: "Report Task Progress",
    description:
      "Publish this task's plan and progress. Pass the complete todo list every time (an empty list clears it) plus an optional one-line summary of what you are doing now. Every task starts with a plan: set a short one (3–8 items) before doing the work, then update it at the end of every turn in which you used tools — mark an item `in_progress` when you begin it and `done` when it lands. Updating the plan is bookkeeping, never a stopping point: call the tool and carry on with the work in the same turn, and only end the run when the task is actually done. Plans are expected to change when new input or an external event (a CI result, a review) invalidates a step; replace the list instead of forcing the old one. A run does not settle while the plan is missing, unfinished, or stale. Omit `todos` to read the current plan back. The dashboard derives numeric progress (done/total) from the list; users cannot edit it.",
    parameters: Type.Object({
      summary: Type.Optional(
        Type.String({
          description:
            "One line describing what you are doing now. Omitted or empty clears it.",
        }),
      ),
      todos: Type.Optional(
        Type.Array(
          Type.Object({
            text: Type.String({
              description: "The step's text (plain text, max 240 chars).",
            }),
            status: Type.Optional(
              Type.String({
                description:
                  'One of "pending" (default), "in_progress", or "done".',
              }),
            ),
          }),
          {
            description:
              "The complete plan in order. Replaces the previous list; [] clears it. Max 50 items.",
          },
        ),
      ),
    }),
    execute: async (_toolCallId, params) => {
      if (params.todos === undefined) {
        return {
          content: [{ type: "text", text: render("Current plan") }],
          details: undefined,
        };
      }

      const rawCount = Array.isArray(params.todos) ? params.todos.length : 0;
      current = normalize(params.summary, params.todos);
      touchedThisTurn = true;
      touchedThisRun = true;
      pi.appendEntry(CUSTOM_TYPE, {
        summary: current.summary,
        todos: current.todos,
      });

      const dropped = rawCount - current.todos.length;
      const notes: string[] = [];
      if (dropped > 0) notes.push(`dropped ${dropped} empty item(s)`);
      if (rawCount > MAX_ITEMS) {
        notes.push(`kept the first ${MAX_ITEMS} items`);
      }
      const suffix = notes.length ? `\n(${notes.join("; ")})` : "";
      return {
        content: [{ type: "text", text: render("Updated plan") + suffix }],
        details: undefined,
      };
    },
  });
}
