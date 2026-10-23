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
 * A single option of an extension `select` dialog. The questionnaire
 * extension renders options as `N. Label — description` lines, so split them
 * back into parts the console can lay out over multiple rows.
 */
export interface OptionLine {
  /** The exact string to send back to Pi when this option is chosen. */
  raw: string;
  /** 1-based position when the option is prefixed with `N. `, else null. */
  index: number | null;
  /** The option's short label. */
  label: string;
  /** The optional description shown below the label. */
  description: string;
}

export function parseOption(option: string): OptionLine {
  const match = /^(\d+)\.\s*([\s\S]*)$/.exec(option);
  const index = match ? Number.parseInt(match[1] ?? "", 10) : null;
  const rest = match ? (match[2] ?? "") : option;

  const separator = rest.indexOf(" — ");
  const label = separator >= 0 ? rest.slice(0, separator) : rest;
  const description = separator >= 0 ? rest.slice(separator + 3) : "";

  return {
    raw: option,
    index: Number.isNaN(index) ? null : index,
    label: label.trim(),
    description: description.trim(),
  };
}

/** One authored choice of a questionnaire question. */
export interface QuestionnaireOption {
  label: string;
  description: string;
  /** Marked by the model as the suggested choice; the UI highlights it. */
  recommended?: boolean;
  /** Optional markdown artifact rendered beside the option in the TUI. */
  preview?: string;
}

/** One question of an `ask_user_question` call. */
export interface QuestionnaireQuestion {
  question: string;
  header: string;
  multiSelect: boolean;
  options: QuestionnaireOption[];
}

/** The parsed arguments of an `ask_user_question` tool call. */
export interface Questionnaire {
  questions: QuestionnaireQuestion[];
}

/**
 * Parse the `ask_user_question` tool arguments. The upstream extension's RPC
 * fallback only sends one question at a time, so the console reconstructs the
 * whole questionnaire from the tool call — available live and in history — to
 * render it as one wizard. Returns null when the payload is not a
 * questionnaire (a different tool, a future schema, malformed JSON), in which
 * case the console falls back to the sequential dialogs.
 */
export function parseQuestionnaire(argsJson: string): Questionnaire | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(argsJson);
  } catch {
    return null;
  }
  const raw = parsed as { questions?: unknown } | null;
  if (!raw || !Array.isArray(raw.questions) || raw.questions.length === 0) {
    return null;
  }

  const questions: QuestionnaireQuestion[] = [];
  for (const entry of raw.questions) {
    const q = entry as Record<string, unknown> | null;
    if (!q || typeof q.question !== "string" || !Array.isArray(q.options)) {
      return null;
    }
    const options: QuestionnaireOption[] = [];
    for (const entryOption of q.options) {
      const o = entryOption as Record<string, unknown> | null;
      if (!o || typeof o.label !== "string") {
        return null;
      }
      options.push({
        label: o.label,
        description: typeof o.description === "string" ? o.description : "",
        recommended: o.recommended === true,
        preview: typeof o.preview === "string" ? o.preview : undefined,
      });
    }
    if (options.length === 0) {
      return null;
    }
    questions.push({
      question: q.question,
      header: typeof q.header === "string" ? q.header : "",
      multiSelect: q.multiSelect === true,
      options,
    });
  }

  return { questions };
}

/** The marker CherryPi's extension puts in the batch dialog prefill. */
export const QUESTIONNAIRE_DIALOG_KIND = "cherrypi.questionnaire";

/**
 * One answered question as the bundled extension reports it in the tool
 * result's `details`. Mirrors `QuestionAnswer` in `ask_user_question.ts`.
 */
export interface QuestionnaireAnswerReview {
  questionIndex: number;
  question: string;
  kind: "option" | "custom" | "multi";
  /** The chosen option label (single-select); null for multi answers. */
  answer: string | null;
  /** Chosen option labels for a multi-select answer. */
  selected?: string[];
  notes?: string;
  preview?: string;
}

/** A finished questionnaire rebuilt from the tool call and its result. */
export interface QuestionnaireReview {
  questions: QuestionnaireQuestion[];
  answers: QuestionnaireAnswerReview[];
  cancelled: boolean;
}

/**
 * Rebuild an answered questionnaire for read-only review from the
 * `ask_user_question` tool call arguments (the questions) and the tool result's
 * `details` (the answers). Returns null when either side is missing or is not a
 * questionnaire, so the console falls back to the raw tool output.
 */
export function parseQuestionnaireReview(
  argsJson: string,
  detailsJson: string,
): QuestionnaireReview | null {
  const parsed = parseQuestionnaire(argsJson);
  if (!parsed || !detailsJson) {
    return null;
  }
  let details: unknown;
  try {
    details = JSON.parse(detailsJson);
  } catch {
    return null;
  }
  const raw = details as { answers?: unknown; cancelled?: unknown } | null;
  if (!raw || !Array.isArray(raw.answers)) {
    return null;
  }

  const answers: QuestionnaireAnswerReview[] = [];
  for (const entry of raw.answers) {
    const a = entry as Record<string, unknown> | null;
    if (!a || typeof a.question !== "string" || typeof a.kind !== "string") {
      return null;
    }
    answers.push({
      questionIndex:
        typeof a.questionIndex === "number" ? a.questionIndex : answers.length,
      question: a.question,
      kind: a.kind === "custom" || a.kind === "multi" ? a.kind : "option",
      answer: typeof a.answer === "string" ? a.answer : null,
      selected: Array.isArray(a.selected)
        ? a.selected.filter((s): s is string => typeof s === "string")
        : undefined,
      notes: typeof a.notes === "string" ? a.notes : undefined,
      preview: typeof a.preview === "string" ? a.preview : undefined,
    });
  }
  return {
    questions: parsed.questions,
    answers,
    cancelled: raw.cancelled === true,
  };
}

/**
 * Convert a reviewed questionnaire into the wizard's answer shape so the same
 * component can render it read-only: option answers map back to their option
 * indices, custom answers keep their text, and notes are carried over.
 */
export function questionnaireAnswersFromReview(
  questions: QuestionnaireQuestion[],
  answers: QuestionnaireAnswerReview[],
): QuestionnaireAnswer[] {
  return questions.map((question, index) => {
    const answer = answers[index];
    if (!answer) {
      return emptyAnswer();
    }
    const notes = answer.notes ?? "";
    if (answer.kind === "custom") {
      return {
        selected: [],
        customSelected: true,
        custom: answer.answer ?? "",
        notes,
      };
    }
    const labels =
      answer.kind === "multi"
        ? (answer.selected ?? [])
        : answer.answer
          ? [answer.answer]
          : [];
    const selected = labels
      .map((label) => question.options.findIndex((o) => o.label === label))
      .filter((i) => i >= 0);
    return { selected, customSelected: false, custom: "", notes };
  });
}

/**
 * Detect CherryPi's batch questionnaire dialog and parse its questions. The
 * extension sends one `editor` request whose prefill is the questionnaire JSON
 * (marked with `kind`), so the console can render the whole wizard from the
 * dialog alone; other dialogs yield null.
 */
export function parseQuestionnaireDialog(request: {
  method: string;
  prefill: string;
}): Questionnaire | null {
  if (request.method !== "editor" || !request.prefill) {
    return null;
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(request.prefill);
  } catch {
    return null;
  }
  const raw = parsed as { kind?: unknown; questions?: unknown } | null;
  if (!raw || raw.kind !== QUESTIONNAIRE_DIALOG_KIND) {
    return null;
  }
  return parseQuestionnaire(JSON.stringify({ questions: raw.questions }));
}

/** The wizard's answer to one question. */
export interface QuestionnaireAnswer {
  /** Selected option indices; single-select uses at most one. */
  selected: number[];
  /** Whether the "Type something." row is the chosen answer. */
  customSelected: boolean;
  /**
   * Custom answer text. The wizard fills it from the same free-text field the
   * user uses for notes, so it is never edited directly.
   */
  custom: string;
  /** Optional free-form note attached to an option answer. */
  notes: string;
}

/** A fresh, unanswered answer slot. */
export function emptyAnswer(): QuestionnaireAnswer {
  return { selected: [], customSelected: false, custom: "", notes: "" };
}

/**
 * Whether a question is answered enough to submit. Choosing the custom row is
 * always a valid answer — even with empty text — because the user explicitly
 * picked it; an unanswered question has neither a selection nor the custom row.
 */
export function isAnswered(answer: QuestionnaireAnswer): boolean {
  return answer.customSelected || answer.selected.length > 0;
}

/** One answer in the JSON payload CherryPi's extension expects back. */
export interface QuestionnaireAnswerWire {
  index: number;
  /** 0-based selected option indices; omitted for a custom answer. */
  options?: number[];
  /** Present (possibly empty) when the custom row was chosen. */
  custom?: string;
  /** Present only when the user wrote a note. */
  notes?: string;
}

/**
 * Build the JSON value the batch dialog expects: one entry per question, with
 * the note omitted when it is empty. Choosing the custom row sends a `custom`
 * key (even when the text is empty), which the extension treats as a valid
 * answer.
 */
export function questionnaireResponseValue(
  questions: QuestionnaireQuestion[],
  answers: QuestionnaireAnswer[],
): string {
  const payload = questions.map((_, index) => {
    const answer = answers[index] ?? emptyAnswer();
    const wire: QuestionnaireAnswerWire = { index };
    if (answer.customSelected) {
      wire.custom = answer.custom.trim();
    } else {
      wire.options = [...answer.selected].sort((a, b) => a - b);
    }
    const notes = answer.notes.trim();
    if (notes.length > 0) {
      wire.notes = notes;
    }
    return wire;
  });
  return JSON.stringify({ answers: payload });
}

/**
 * The value to send for a multi-select question's `input` dialog: the 1-based
 * option numbers comma-separated, or the typed answer verbatim. Mirrors
 * `rpc-fallback.ts` — any non-index token is taken as a custom answer.
 */
export function multiSelectValue(answer: QuestionnaireAnswer): string {
  const custom = answer.custom.trim();
  if (answer.customSelected && custom.length > 0) {
    return custom;
  }
  return answer.selected
    .map((index) => index + 1)
    .sort((a, b) => a - b)
    .join(",");
}
