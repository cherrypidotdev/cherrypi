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
 * CherryPi questionnaire.
 *
 * Registers the `ask_user_question` tool: a structured, tabbed questionnaire
 * the model puts to the user when its request is underspecified. The upstream
 * `@juicesharp/rpiv-ask-user-question` extension is not installed in CherryPi
 * images; this bundled extension is the single source of truth and, because it
 * is loaded with an explicit `--extension` flag, its `ask_user_question`
 * registration takes precedence over any question tool another extension may
 * register (Pi keeps the first registration per tool name).
 *
 * RPC hosts (the CherryPi web console) answer the whole questionnaire in one
 * extension-UI dialog: an `editor` request whose prefill is the questionnaire
 * JSON and whose value is the batch answer JSON. One dialog means one inbox
 * notification and one webhook — never one per question. Terminal Pi keeps a
 * native sequential walker.
 */

export const ASK_USER_QUESTION_TOOL_NAME = "ask_user_question";

/** Marker carried in the dialog prefill so the host recognizes a batch. */
export const QUESTIONNAIRE_DIALOG_KIND = "cherrypi.questionnaire";

export const MAX_QUESTIONS = 10;
export const MIN_OPTIONS = 2;
export const MAX_OPTIONS = 4;
export const MAX_HEADER_LENGTH = 16;
export const MAX_LABEL_LENGTH = 60;

const RESERVED_LABELS = ["Other", "Type something."];

export const OptionSchema = Type.Object({
  label: Type.String({
    maxLength: MAX_LABEL_LENGTH,
    description: `MAX ${MAX_LABEL_LENGTH} CHARACTERS — hard limit, requests over the limit are rejected. The display text for this option that the user will see and select. Should be concise (1-5 words) and clearly describe the choice.`,
  }),
  description: Type.String({
    description:
      "Explanation of what this option means or what will happen if chosen. Useful for providing context about trade-offs or implications.",
  }),
  recommended: Type.Optional(
    Type.Boolean({
      default: false,
      description:
        "Set to true to mark this as the recommended answer. The UI highlights it so the user can see the suggested choice at a glance.",
    }),
  ),
  preview: Type.Optional(
    Type.String({
      description:
        "Optional preview content rendered under this option. Use for mockups, code snippets, or visual comparisons that help the user compare choices.",
    }),
  ),
});

export const QuestionSchema = Type.Object({
  question: Type.String({
    description:
      "The complete question to ask the user. Clear, specific, and ending with a question mark. Raw HTML is allowed and rendered (Markdown too): use it for emphasis, links, lists, and small tables. If multiSelect is true, phrase it accordingly.",
  }),
  header: Type.String({
    maxLength: MAX_HEADER_LENGTH,
    description: `MAX ${MAX_HEADER_LENGTH} CHARACTERS — hard limit, requests over the limit are rejected. Very short chip/tag shown on the question tab. Examples: "Auth method", "Library", "Approach".`,
  }),
  options: Type.Array(OptionSchema, {
    minItems: MIN_OPTIONS,
    maxItems: MAX_OPTIONS,
    description: `The available choices for this question. Must have ${MIN_OPTIONS}-${MAX_OPTIONS} options. The "Type something." custom-answer row is appended automatically — do NOT author it.`,
  }),
  multiSelect: Type.Optional(
    Type.Boolean({
      default: false,
      description:
        "Set to true to let the user select several options. The user can still add a note to the answer.",
    }),
  ),
});

export const QuestionParamsSchema = Type.Object({
  questions: Type.Array(QuestionSchema, {
    minItems: 1,
    maxItems: MAX_QUESTIONS,
    description: `Questions to ask the user (1-${MAX_QUESTIONS} at once). All questions are shown as tabs and submitted together.`,
  }),
});

export interface OptionData {
  label: string;
  description: string;
  recommended?: boolean;
  preview?: string;
}

export interface QuestionData {
  question: string;
  header: string;
  options: OptionData[];
  multiSelect?: boolean;
}

export interface QuestionParams {
  questions: QuestionData[];
}

/** One answer as the host sends it back (per question, by index). */
interface WireAnswer {
  index?: number;
  /** 0-based selected option indices (choose-one or choose-many). */
  options?: number[];
  /** Present (possibly empty) when the custom row was chosen. */
  custom?: string;
  /** Present only when the user added a note. */
  notes?: string;
}

interface WireResponse {
  answers?: WireAnswer[];
}

/** The LLM-facing answer. `kind` mirrors the upstream extension's vocabulary. */
export interface QuestionAnswer {
  questionIndex: number;
  question: string;
  kind: "option" | "custom" | "multi";
  answer: string | null;
  selected?: string[];
  notes?: string;
  preview?: string;
}

export interface QuestionnaireResult {
  answers: QuestionAnswer[];
  cancelled: boolean;
}

export const TOOL_DESCRIPTION = `Ask the user one or more structured questions during execution. Use when you need to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices as you work
4. Offer choices to the user about what direction to take

Usage notes:
- Up to ${MAX_QUESTIONS} questions per call (${MIN_OPTIONS}-${MAX_OPTIONS} options each). Group every clarifying question into one call instead of stacking several calls.
- Every question always offers a "Type something." custom-answer row; do NOT author "Other" or "Type something." labels yourself. An empty custom answer is a valid answer.
- The user can attach a free-form note to any answer, including a custom one.
- Question text is rendered as HTML/Markdown, so you may use simple formatting (bold, links, lists, small tables).
- If you recommend an option, set \`recommended: true\` on it. The user sees it highlighted; you may also keep it first in the list.
- Set \`multiSelect: true\` when several answers are valid.
- This tool is provided by CherryPi's bundled extension; its tools take precedence over any question-related tool another extension may register.`;

export const PROMPT_SNIPPET = `Ask the user up to ${MAX_QUESTIONS} structured questions (${MIN_OPTIONS}-${MAX_OPTIONS} options each) when requirements are ambiguous`;

export const PROMPT_GUIDELINES: string[] = [
  `Use ${ASK_USER_QUESTION_TOOL_NAME} whenever the user's request is underspecified and you cannot proceed without concrete decisions — you can ask up to ${MAX_QUESTIONS} questions per invocation.`,
  `Each question MUST have ${MIN_OPTIONS}-${MAX_OPTIONS} options. Every option requires a concise label (1-5 words) and a description explaining what the choice means or its trade-offs. Every question gets an automatically appended "Type something." custom-answer row, and an answer is always required — an empty custom answer counts. Do NOT author "Other" or "Type something." labels yourself.`,
  `Set multiSelect: true when multiple answers are valid. Mark the option you recommend with \`recommended: true\` so it is highlighted. Question text may use raw HTML/Markdown.`,
  `Do not stack multiple ${ASK_USER_QUESTION_TOOL_NAME} calls back-to-back — group all clarifying questions into one invocation.`,
];

const ERROR_NO_UI =
  "Error: UI not available (running in non-interactive mode). Ask the questions as plain chat text instead.";

/** A dialog host that can render the questionnaire's sequential primitives. */
type DialogUI = {
  select: (title: string, options: string[]) => Promise<string | undefined>;
  input: (title: string, placeholder?: string) => Promise<string | undefined>;
};

type Mode = string | undefined;

function hasDialogUI(ui: unknown): ui is DialogUI {
  const u = ui as Partial<Record<"select" | "input", unknown>> | null;
  return typeof u?.select === "function" && typeof u?.input === "function";
}

/** Normalize line terminators and trim, matching the upstream extension. */
function normalizeText(value: string): string {
  return value.replace(/\r\n?/g, "\n");
}

function normalizeParams(params: QuestionParams): QuestionData[] {
  const questions = Array.isArray(params.questions) ? params.questions : [];
  return questions.slice(0, MAX_QUESTIONS).map((q) => ({
    question: normalizeText(String(q.question ?? "")),
    header: String(q.header ?? ""),
    multiSelect: q.multiSelect === true,
    options: (Array.isArray(q.options) ? q.options : []).map((o) => ({
      label: String(o.label ?? ""),
      description: String(o.description ?? ""),
      recommended: o.recommended === true,
      preview:
        typeof o.preview === "string" && o.preview.length > 0
          ? o.preview
          : undefined,
    })),
  }));
}

/** Validate a normalized questionnaire, returning an LLM-facing error. */
function validate(questions: QuestionData[]): string | null {
  if (questions.length === 0) {
    return "Error: at least one question is required.";
  }
  if (questions.length > MAX_QUESTIONS) {
    return `Error: at most ${MAX_QUESTIONS} questions per call.`;
  }
  for (const [i, q] of questions.entries()) {
    if (q.question.trim().length === 0) {
      return `Error: question ${i + 1} is empty.`;
    }
    if (q.header.length > MAX_HEADER_LENGTH) {
      return `Error: question ${i + 1} header exceeds ${MAX_HEADER_LENGTH} characters.`;
    }
    if (q.options.length < MIN_OPTIONS || q.options.length > MAX_OPTIONS) {
      return `Error: question ${i + 1} must have ${MIN_OPTIONS}-${MAX_OPTIONS} options.`;
    }
    const seen = new Set<string>();
    for (const o of q.options) {
      if (o.label.trim().length === 0) {
        return `Error: question ${i + 1} has an empty option label.`;
      }
      if (o.label.length > MAX_LABEL_LENGTH) {
        return `Error: question ${i + 1} option "${o.label}" exceeds ${MAX_LABEL_LENGTH} characters.`;
      }
      if (RESERVED_LABELS.includes(o.label.trim())) {
        return `Error: question ${i + 1} uses reserved option label "${o.label.trim()}".`;
      }
      if (seen.has(o.label)) {
        return `Error: question ${i + 1} repeats option label "${o.label}".`;
      }
      seen.add(o.label);
    }
  }
  return null;
}

/** Format one option row for the sequential dialog walker. */
function formatOption(option: OptionData, index: number): string {
  const recommended = option.recommended ? " (Recommended)" : "";
  return `${index + 1}. ${option.label}${recommended} — ${option.description}`;
}

function answerFromWire(
  q: QuestionData,
  qi: number,
  wire: WireAnswer,
): QuestionAnswer {
  const notes =
    typeof wire.notes === "string" && wire.notes.trim().length > 0
      ? wire.notes
      : undefined;
  const base = { questionIndex: qi, question: q.question } as const;

  if (typeof wire.custom === "string") {
    // The custom row was chosen; an empty value is a deliberate answer.
    return {
      ...base,
      kind: "custom",
      answer: wire.custom,
      ...(notes ? { notes } : {}),
    };
  }

  const indices = (Array.isArray(wire.options) ? wire.options : []).filter(
    (n): n is number => Number.isInteger(n) && n >= 0 && n < q.options.length,
  );
  if (q.multiSelect) {
    return {
      ...base,
      kind: "multi",
      answer: null,
      selected: indices.map((n) => q.options[n]!.label),
      ...(notes ? { notes } : {}),
    };
  }
  const chosen = indices[0] !== undefined ? q.options[indices[0]] : undefined;
  return {
    ...base,
    kind: "option",
    answer: chosen?.label ?? "",
    ...(chosen?.preview && chosen.preview.length > 0
      ? { preview: chosen.preview }
      : {}),
    ...(notes ? { notes } : {}),
  };
}

function parseWire(value: string, questions: QuestionData[]): QuestionAnswer[] {
  let parsed: WireResponse;
  try {
    parsed = JSON.parse(value) as WireResponse;
  } catch {
    parsed = {};
  }
  const byIndex = new Map<number, WireAnswer>();
  for (const answer of Array.isArray(parsed.answers) ? parsed.answers : []) {
    if (typeof answer?.index === "number") {
      byIndex.set(answer.index, answer);
    }
  }
  return questions.map((q, qi) =>
    answerFromWire(q, qi, byIndex.get(qi) ?? { options: [] }),
  );
}

/** The `"Q"="A"` model-facing envelope, matching the upstream wording. */
export function buildEnvelope(
  answers: QuestionAnswer[],
  cancelled: boolean,
): string {
  if (cancelled) {
    return "User declined to answer questions";
  }
  const segments: string[] = [];
  for (const a of answers) {
    const value =
      a.kind === "multi"
        ? (a.selected ?? []).join(", ")
        : a.answer === null
          ? ""
          : a.answer;
    let segment = `"${a.question}"="${value}"`;
    if (a.preview && a.preview.length > 0) {
      segment += `. selected preview: ${a.preview}`;
    }
    if (a.notes && a.notes.length > 0) {
      segment += `. user notes: ${a.notes}`;
    }
    segments.push(`${segment}.`);
  }
  if (segments.length === 0) {
    return "User declined to answer questions";
  }
  return `User has answered your questions: ${segments.join(" ")} You can now continue with the user's answers in mind.`;
}

function toolResult(
  text: string,
  answers: QuestionAnswer[],
  cancelled: boolean,
) {
  return {
    content: [{ type: "text" as const, text }],
    details: { answers, cancelled },
  };
}

/**
 * RPC path: one `editor` dialog carries the whole questionnaire and receives
 * the whole answer set. The dialog is a pure transport — the console renders
 * the wizard from the tool call / prefill, so the editor is never shown raw.
 */
async function runBatch(ctx: ExtensionContext, questions: QuestionData[]) {
  const title = `CherryPi questionnaire (${questions.length} ${
    questions.length === 1 ? "question" : "questions"
  })`;
  const prefill = JSON.stringify({
    kind: QUESTIONNAIRE_DIALOG_KIND,
    questions,
  });

  const raw = await ctx.ui.editor(title, prefill);
  if (raw == null) {
    return toolResult(buildEnvelope([], true), [], true);
  }
  const answers = parseWire(raw, questions);
  return toolResult(buildEnvelope(answers, false), answers, false);
}

/**
 * Terminal fallback: walk the questions one dialog at a time with the plain
 * select/input primitives (no tabs, no HTML, but the same answer model). Notes
 * are collected after each answer.
 */
async function runWalker(ctx: ExtensionContext, questions: QuestionData[]) {
  const ui = ctx.ui as unknown as DialogUI;
  const answers: QuestionAnswer[] = [];

  for (const [qi, q] of questions.entries()) {
    const header = q.header ? `[${q.header}] ` : "";
    let wire: WireAnswer;

    if (q.multiSelect) {
      const list = q.options.map(formatOption).join("\n");
      const value = await ui.input(
        `${header}${q.question}\n\n${list}\n\nEnter the numbers of all that apply, comma-separated (e.g. "1,3"), or type a custom answer.`,
        "1,3",
      );
      if (value == null) {
        return toolResult(buildEnvelope(answers, true), answers, true);
      }
      const trimmed = value.trim();
      const tokens = trimmed.split(/[,\s]+/).filter((t) => t.length > 0);
      const indices = tokens.map((t) =>
        /^\d+\.?$/.test(t) ? Number.parseInt(t, 10) - 1 : -1,
      );
      if (indices.every((i) => i >= 0 && i < q.options.length)) {
        wire = { options: indices };
      } else {
        wire = { custom: trimmed };
      }
    } else {
      const options = q.options.map(formatOption);
      options.push(`${q.options.length + 1}. Type something.`);
      const chosen = await ui.select(`${header}${q.question}`, options);
      if (chosen == null) {
        return toolResult(buildEnvelope(answers, true), answers, true);
      }
      const chosenIndex = Number.parseInt(chosen, 10) - 1;
      if (chosenIndex >= 0 && chosenIndex < q.options.length) {
        wire = { options: [chosenIndex] };
      } else {
        const typed = await ui.input(
          `${header}${q.question}\n\nType your answer:`,
          "",
        );
        if (typed == null) {
          return toolResult(buildEnvelope(answers, true), answers, true);
        }
        wire = { custom: typed };
      }
    }

    const notes = await ui.input(
      `${header}${q.question}\n\nNote (optional):`,
      "",
    );
    if (notes != null && notes.trim().length > 0) {
      wire.notes = notes;
    }
    answers.push(answerFromWire(q, qi, wire));
  }

  return toolResult(buildEnvelope(answers, false), answers, false);
}

export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: ASK_USER_QUESTION_TOOL_NAME,
    label: "Ask User Question",
    description: TOOL_DESCRIPTION,
    promptSnippet: PROMPT_SNIPPET,
    promptGuidelines: PROMPT_GUIDELINES,
    parameters: QuestionParamsSchema,

    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const questions = normalizeParams(params as unknown as QuestionParams);
      if (!ctx.hasUI) {
        return toolResult(ERROR_NO_UI, [], true);
      }
      const invalid = validate(questions);
      if (invalid) {
        return toolResult(invalid, [], true);
      }

      const mode: Mode = (ctx as { mode?: string }).mode;
      if (mode === "rpc") {
        return runBatch(ctx, questions);
      }
      if (hasDialogUI(ctx.ui)) {
        return runWalker(ctx, questions);
      }
      return toolResult(ERROR_NO_UI, [], true);
    },
  });
}
