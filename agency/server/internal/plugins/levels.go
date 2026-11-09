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

package plugins

import "strings"

const (
	LevelTrace = "trace"
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

var logLevels = []string{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError}

func ValidLogLevel(level string) bool {
	_, ok := logLevelRank(strings.ToLower(strings.TrimSpace(level)))
	return ok
}

func NormalizeLogLevel(level string) (string, bool) {
	level = strings.ToLower(strings.TrimSpace(level))
	_, ok := logLevelRank(level)
	return level, ok
}

func levelsAtLeast(min string) []string {
	min = strings.ToLower(strings.TrimSpace(min))
	if min == "" {
		return nil
	}
	rank, ok := logLevelRank(min)
	if !ok {
		return nil
	}
	return append([]string(nil), logLevels[rank:]...)
}

func logLevelRank(level string) (int, bool) {
	for i, candidate := range logLevels {
		if candidate == level {
			return i, true
		}
	}
	return 0, false
}
