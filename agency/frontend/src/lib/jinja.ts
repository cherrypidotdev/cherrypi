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

export type JinjaTokenType =
  | "text"
  | "expression"
  | "tag"
  | "keyword"
  | "variable"
  | "string"
  | "number"
  | "operator"
  | "comment";

export interface JinjaToken {
  type: JinjaTokenType;
  text: string;
}

const KEYWORDS = new Set([
  "and",
  "autoescape",
  "block",
  "call",
  "cycle",
  "elif",
  "else",
  "endautoescape",
  "endblock",
  "endcall",
  "endfilter",
  "endfor",
  "endif",
  "endmacro",
  "endraw",
  "endset",
  "endtrans",
  "endwith",
  "extends",
  "false",
  "filter",
  "for",
  "from",
  "if",
  "import",
  "in",
  "include",
  "is",
  "loop",
  "macro",
  "none",
  "not",
  "or",
  "pluralize",
  "raw",
  "recursive",
  "set",
  "trans",
  "true",
  "with",
]);

const OPERATORS = new Set("+-*/%=<>!~|&.,():[]{}".split(""));
const WHITESPACE = /\s/;
const DIGIT = /[0-9]/;
const IDENTIFIER_START = /[A-Za-z_]/;
const IDENTIFIER_PART = /[A-Za-z0-9_.]/;

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function jinjaTokenize(source: string): JinjaToken[] {
  const tokens: JinjaToken[] = [];
  const push = (type: JinjaTokenType, text: string) => {
    if (text) tokens.push({ type, text });
  };
  let index = 0;
  while (index < source.length) {
    const expression = source.indexOf("{{", index);
    const statement = source.indexOf("{%", index);
    const comment = source.indexOf("{#", index);
    const starts = [expression, statement, comment].filter(
      (value) => value >= 0,
    );
    if (starts.length === 0) {
      push("text", source.slice(index));
      break;
    }
    const start = Math.min(...starts);
    push("text", source.slice(index, start));
    if (start === comment) {
      const close = source.indexOf("#}", start + 2);
      const end = close < 0 ? source.length : close + 2;
      push("comment", source.slice(start, end));
      index = end;
      continue;
    }
    const expressionStart = start === expression;
    const type: JinjaTokenType = expressionStart ? "expression" : "tag";
    const closing = expressionStart ? "}}" : "%}";
    push(type, source.slice(start, start + 2));
    index = start + 2;
    while (index < source.length && !source.startsWith(closing, index)) {
      const char = source[index]!;
      if (WHITESPACE.test(char)) {
        let end = index + 1;
        while (
          end < source.length &&
          WHITESPACE.test(source[end]!) &&
          !source.startsWith(closing, end)
        ) {
          end += 1;
        }
        push("text", source.slice(index, end));
        index = end;
        continue;
      }
      if (char === "'" || char === '"') {
        let end = index + 1;
        while (end < source.length) {
          if (source[end] === "\\") {
            end += 2;
            continue;
          }
          if (source[end] === char) {
            end += 1;
            break;
          }
          end += 1;
        }
        push("string", source.slice(index, Math.min(end, source.length)));
        index = Math.min(end, source.length);
        continue;
      }
      if (DIGIT.test(char)) {
        let end = index + 1;
        while (end < source.length && /[0-9._]/.test(source[end]!)) end += 1;
        push("number", source.slice(index, end));
        index = end;
        continue;
      }
      if (IDENTIFIER_START.test(char)) {
        let end = index + 1;
        while (end < source.length && IDENTIFIER_PART.test(source[end]!)) {
          end += 1;
        }
        const text = source.slice(index, end);
        push(KEYWORDS.has(text.toLowerCase()) ? "keyword" : "variable", text);
        index = end;
        continue;
      }
      if (OPERATORS.has(char)) {
        const pair = source.slice(index, index + 2);
        if (["==", "!=", ">=", "<=", "//", "**"].includes(pair)) {
          push("operator", pair);
          index += 2;
          continue;
        }
        push("operator", char);
        index += 1;
        continue;
      }
      push("text", char);
      index += 1;
    }
    if (source.startsWith(closing, index)) {
      push(type, closing);
      index += 2;
    }
  }
  return tokens;
}

export function renderJinjaHtml(source: string): string {
  return jinjaTokenize(source)
    .map((token) =>
      token.type === "text"
        ? escapeHtml(token.text)
        : `<span class="jinja-${token.type}">${escapeHtml(token.text)}</span>`,
    )
    .join("");
}
