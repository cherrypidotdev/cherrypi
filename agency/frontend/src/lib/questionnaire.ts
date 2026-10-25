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

export interface OptionLine {
  raw: string;

  index: number | null;

  label: string;

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

export interface QuestionnaireOption {
  label: string;
  description: string;

  recommended?: boolean;

  preview?: string;
}

export interface QuestionnaireQuestion {
  question: string;
  header: string;
  multiSelect: boolean;
  options: QuestionnaireOption[];
}

export interface Questionnaire {
  questions: QuestionnaireQuestion[];
}

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

export const QUESTIONNAIRE_DIALOG_KIND = "cherrypi.questionnaire";

export interface QuestionnaireAnswerReview {
  questionIndex: number;
  question: string;
  kind: "option" | "custom" | "multi";

  answer: string | null;

  selected?: string[];
  notes?: string;
  preview?: string;
}

export interface QuestionnaireReview {
  questions: QuestionnaireQuestion[];
  answers: QuestionnaireAnswerReview[];
  cancelled: boolean;

  unavailable: boolean;
}

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
  const raw = details as {
    answers?: unknown;
    cancelled?: unknown;
    unavailable?: unknown;
  } | null;
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
    unavailable: raw.unavailable === true,
  };
}

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

export interface QuestionnaireAnswer {
  selected: number[];

  customSelected: boolean;

  custom: string;

  notes: string;
}

export function emptyAnswer(): QuestionnaireAnswer {
  return { selected: [], customSelected: false, custom: "", notes: "" };
}

export function isAnswered(answer: QuestionnaireAnswer): boolean {
  return answer.customSelected || answer.selected.length > 0;
}

export interface QuestionnaireAnswerWire {
  index: number;

  options?: number[];

  custom?: string;

  notes?: string;
}

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
