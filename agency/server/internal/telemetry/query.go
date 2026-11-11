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

package telemetry

import (
	"regexp"
	"strings"
)

//	-- name: ListTasks :many

var sqlcNamePattern = regexp.MustCompile(`^\s*--\s*name:\s*([A-Za-z0-9_]+)\s*:`)

var sqlTablePattern = regexp.MustCompile(
	`(?is)\b(?:FROM|INTO|UPDATE|JOIN|TABLE)\s+["` + "`" + `\[]?([A-Za-z_][A-Za-z0-9_$]*(?:\.[A-Za-z_][A-Za-z0-9_$]*)?)`,
)

func querySummary(query string) string {
	if name := sqlcQueryName(query); name != "" {
		return "store." + name
	}

	return rawQuerySummary(query)
}

// sqlcQueryName reads the name from an sqlc "-- name:" comment.
func sqlcQueryName(query string) string {
	match := sqlcNamePattern.FindStringSubmatch(query)
	if len(match) != 2 {
		return ""
	}

	return match[1]
}

func rawQuerySummary(query string) string {
	statement := stripLeadingSQLComments(query)
	if statement == "" {
		return ""
	}

	operation := strings.ToUpper(firstSQLWord(statement))
	if operation == "" {
		return ""
	}

	search := statement
	if operation == "WITH" {

		if main, index := topLevelStatementVerb(statement); main != "" {
			operation = main
			if main != "SELECT" {
				search = statement[index:]
			}
		}
	}

	table := firstSQLTable(maskSQLStrings(search))
	if table != "" {
		return operation + " " + table
	}

	return operation
}

func stripLeadingSQLComments(query string) string {
	remaining := strings.TrimSpace(query)
	for {
		switch {
		case strings.HasPrefix(remaining, "--"):
			if index := strings.IndexByte(remaining, '\n'); index >= 0 {
				remaining = strings.TrimSpace(remaining[index+1:])
				continue
			}

			return ""
		case strings.HasPrefix(remaining, "/*"):
			if index := strings.Index(remaining[2:], "*/"); index >= 0 {
				remaining = strings.TrimSpace(remaining[index+4:])
				continue
			}

			return ""
		default:
			return remaining
		}
	}
}

func firstSQLWord(statement string) string {
	for index, r := range statement {
		if !isSQLWordRune(r) {
			return statement[:index]
		}
	}

	return statement
}

func isSQLWordRune(r rune) bool {
	return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

func topLevelStatementVerb(statement string) (string, int) {
	depth := 0
	for index := 0; index < len(statement); {
		switch {
		case statement[index] == '-':
			if next := skipLineComment(statement, index); next > index {
				index = next
				continue
			}
		case statement[index] == '/':
			if next := skipBlockComment(statement, index); next > index {
				index = next
				continue
			}
		case statement[index] == '\'' || statement[index] == '"' ||
			statement[index] == '`':
			index = skipQuoted(statement, index)
			continue
		case statement[index] == '[':
			index = skipBracketed(statement, index)
			continue
		case statement[index] == '(':
			depth++
		case statement[index] == ')':
			if depth > 0 {
				depth--
			}
		case isSQLWordRune(rune(statement[index])):
			start := index
			for index < len(statement) && isSQLWordRune(rune(statement[index])) {
				index++
			}
			if depth == 0 {
				word := strings.ToUpper(statement[start:index])
				switch word {
				case "SELECT", "INSERT", "UPDATE", "DELETE", "MERGE":
					return word, start
				}
			}
			continue
		}
		index++
	}

	return "", 0
}

func maskSQLStrings(statement string) string {
	masked := []byte(statement)
	for index := 0; index < len(masked); {
		switch masked[index] {
		case '-':
			if next := skipLineComment(string(masked), index); next > index {
				for i := index; i < next; i++ {
					masked[i] = ' '
				}
				index = next
				continue
			}
		case '/':
			if next := skipBlockComment(string(masked), index); next > index {
				for i := index; i < next; i++ {
					masked[i] = ' '
				}
				index = next
				continue
			}
		case '\'':
			next := skipQuoted(string(masked), index)
			for i := index; i < next; i++ {
				masked[i] = ' '
			}
			index = next
			continue
		}
		index++
	}

	return string(masked)
}

func skipLineComment(statement string, index int) int {
	if index+1 >= len(statement) || statement[index] != '-' || statement[index+1] != '-' {
		return index
	}
	if newline := strings.IndexByte(statement[index:], '\n'); newline >= 0 {
		return index + newline + 1
	}

	return len(statement)
}

func skipBlockComment(statement string, index int) int {
	if index+1 >= len(statement) || statement[index] != '/' || statement[index+1] != '*' {
		return index
	}
	if close := strings.Index(statement[index+2:], "*/"); close >= 0 {
		return index + close + 4
	}

	return len(statement)
}

func skipQuoted(statement string, index int) int {
	quote := statement[index]
	index++
	for index < len(statement) {
		if statement[index] != quote {
			index++
			continue
		}

		if index+1 < len(statement) && statement[index+1] == quote {
			index += 2
			continue
		}

		return index + 1
	}

	return len(statement)
}

func skipBracketed(statement string, index int) int {
	if close := strings.IndexByte(statement[index:], ']'); close >= 0 {
		return index + close + 1
	}

	return len(statement)
}

func firstSQLTable(statement string) string {
	match := sqlTablePattern.FindStringSubmatch(statement)
	if len(match) != 2 {
		return ""
	}

	table := strings.Trim(match[1], "\"`[]")
	if index := strings.LastIndexByte(table, '.'); index >= 0 {
		table = table[index+1:]
	}

	return table
}
