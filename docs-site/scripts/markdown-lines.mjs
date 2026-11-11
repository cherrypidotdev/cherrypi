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

export function* markdownLines(text) {
  const lines = text.split("\n");
  let fence = null;
  let indentedCode = false;
  let previousBlank = true;

  const listStack = [];

  for (const line of lines) {
    const fenceMatch = /^\s*(`{3,}|~{3,})/.exec(line);
    if (fence) {
      if (
        fenceMatch &&
        fenceMatch[1][0] === fence[0] &&
        fenceMatch[1].length >= fence.length
      ) {
        fence = null;
      }
      yield { line, code: true };
      continue;
    }
    if (fenceMatch) {
      fence = fenceMatch[1];
      yield { line, code: true };
      continue;
    }

    if (line.trim() === "") {
      previousBlank = true;
      yield { line, code: indentedCode };
      continue;
    }

    const indent = indentWidth(line);
    updateListStack(listStack, line, indent);
    const baseIndent =
      listStack.length > 0 ? listStack[listStack.length - 1] : 0;
    const relativeIndent = indent - baseIndent;

    if (indentedCode && relativeIndent >= 4) {
      previousBlank = false;
      yield { line, code: true };
      continue;
    }
    indentedCode = false;

    if (relativeIndent >= 4 && previousBlank) {
      indentedCode = true;
      previousBlank = false;
      yield { line, code: true };
      continue;
    }

    previousBlank = false;
    yield { line, code: false };
  }
}

function indentWidth(line) {
  let width = 0;
  for (const char of line) {
    if (char === " ") width++;
    else if (char === "\t") width += 4 - (width % 4);
    else break;
  }
  return width;
}

function updateListStack(stack, line, indent) {
  const marker = /^([ \t]*)([-*+]|\d{1,9}[.)])([ \t]+)/.exec(line);
  if (marker) {
    const markerIndent = indentWidth(marker[1]);
    while (stack.length > 0 && markerIndent < stack[stack.length - 1]) {
      stack.pop();
    }

    stack.push(markerIndent + marker[2].length + marker[3].length);
    return;
  }
  while (stack.length > 0 && indent < stack[stack.length - 1]) {
    stack.pop();
  }
}
