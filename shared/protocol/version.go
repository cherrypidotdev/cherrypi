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

package protocol

import "golang.org/x/mod/semver"

const Version = "v0.1.1"

type Compatibility int64

const (
	Incompatible Compatibility = iota
	// Outdated means the same compatibility line but an older version: usable,

	Outdated
	// Current means the same compatibility line and equal or newer: nothing to

	Current
)

// It keys on the compatibility line returned by compatLine: the major once it
// is non-zero, otherwise the minor. A different line (or a version that cannot
// be parsed) is Incompatible, a version behind the agency on the same line is

func Compare(agent, agency string) Compatibility {
	if !semver.IsValid(agent) || !semver.IsValid(agency) {
		return Incompatible
	}
	if compatLine(agent) != compatLine(agency) {
		return Incompatible
	}
	if semver.Compare(agent, agency) < 0 {
		return Outdated
	}
	return Current
}

func compatLine(v string) string {
	if semver.Major(v) == "v0" {
		return semver.MajorMinor(v)
	}
	return semver.Major(v)
}
