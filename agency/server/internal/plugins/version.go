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

import (
	"fmt"
	"strconv"
	"strings"
)

// additive, so existing bundles that accept the 0.2.x line keep working; a

const APIVersion = "0.2.6"

type VersionRange struct {
	alternatives [][]comparator
	raw          string
}

type comparator struct {
	op      string
	major   int
	minor   int
	patch   int
	hasPart int
}

func ParseVersionRange(raw string) (VersionRange, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return VersionRange{raw: raw}, nil
	}
	out := VersionRange{raw: raw}
	for _, group := range strings.Split(raw, "||") {
		group = strings.TrimSpace(group)
		if group == "" || group == "*" {
			out.alternatives = append(out.alternatives, nil)
			continue
		}
		var comparators []comparator
		for _, field := range strings.Fields(group) {
			c, err := parseComparator(field)
			if err != nil {
				return VersionRange{}, fmt.Errorf("requires_agency %q: %w", raw, err)
			}
			comparators = append(comparators, c)
		}
		out.alternatives = append(out.alternatives, comparators)
	}
	if len(out.alternatives) == 0 {
		return VersionRange{}, fmt.Errorf("requires_agency %q is empty", raw)
	}
	return out, nil
}

func parseComparator(field string) (comparator, error) {
	op := ""
	for _, candidate := range []string{">=", "<=", ">", "<", "=", "^", "~"} {
		if strings.HasPrefix(field, candidate) {
			op = candidate
			field = strings.TrimSpace(strings.TrimPrefix(field, candidate))
			break
		}
	}
	if field == "" || field == "*" {
		if op != "" {
			return comparator{}, fmt.Errorf("operator %q needs a version", op)
		}
		return comparator{op: "*"}, nil
	}
	major, minor, patch, parts, err := parseVersion(field)
	if err != nil {
		return comparator{}, err
	}
	if op == "" {
		op = "="
	}
	return comparator{op: op, major: major, minor: minor, patch: patch, hasPart: parts}, nil
}

func parseVersion(value string) (major, minor, patch, parts int, err error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if value == "" {
		return 0, 0, 0, 0, fmt.Errorf("empty version")
	}
	// Drop a pre-release/build suffix: ranges compare the release line only.
	if i := strings.IndexAny(value, "-+"); i >= 0 {
		value = value[:i]
	}
	fields := strings.Split(value, ".")
	if len(fields) > 3 {
		return 0, 0, 0, 0, fmt.Errorf("%q is not a version", value)
	}
	numbers := []int{0, 0, 0}
	for i, field := range fields {
		if field == "" {
			return 0, 0, 0, 0, fmt.Errorf("%q is not a version", value)
		}
		n, convErr := strconv.Atoi(field)
		if convErr != nil || n < 0 {
			return 0, 0, 0, 0, fmt.Errorf("%q is not a version", value)
		}
		numbers[i] = n
	}
	return numbers[0], numbers[1], numbers[2], len(fields), nil
}

func (r VersionRange) Matches(version string) bool {
	if len(r.alternatives) == 0 {
		return true
	}
	major, minor, patch, _, err := parseVersion(version)
	if err != nil {
		return false
	}
	have := [3]int{major, minor, patch}
	for _, group := range r.alternatives {
		ok := true
		for _, c := range group {
			if !c.matches(have) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func (c comparator) matches(have [3]int) bool {
	switch c.op {
	case "*":
		return true
	case "=":
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, c.hasPart) == 0
	case ">=":
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, c.hasPart) >= 0
	case "<=":
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, c.hasPart) <= 0
	case ">":
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, c.hasPart) > 0
	case "<":
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, c.hasPart) < 0
	case "^":
		upper := caretUpper(c)
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, 3) >= 0 &&
			compareParts(have, upper, 3) < 0
	case "~":
		upper := tildeUpper(c)
		return compareParts(have, [3]int{c.major, c.minor, c.patch}, 3) >= 0 &&
			compareParts(have, upper, 3) < 0
	default:
		return false
	}
}

func caretUpper(c comparator) [3]int {
	switch {
	case c.major > 0:
		return [3]int{c.major + 1, 0, 0}
	case c.minor > 0:
		return [3]int{0, c.minor + 1, 0}
	default:
		return [3]int{0, 0, c.patch + 1}
	}
}

func tildeUpper(c comparator) [3]int {
	if c.hasPart == 1 {
		return [3]int{c.major + 1, 0, 0}
	}
	return [3]int{c.major, c.minor + 1, 0}
}

func compareParts(have, want [3]int, parts int) int {
	if parts <= 0 || parts > 3 {
		parts = 3
	}
	for i := 0; i < parts; i++ {
		switch {
		case have[i] < want[i]:
			return -1
		case have[i] > want[i]:
			return 1
		}
	}
	return 0
}
