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

// Package protocol defines the agent↔agency protocol version: the semver an
// agent reports in its tunnel hello frame and the agency compares against to
// decide whether a binary is current, outdated, or incompatible.
//
// The version is deliberately separate from the agent's build version (the
// git-describe stamp baked into the image). The build version tells an operator
// exactly which image an agent runs; the protocol version is what compatibility
// is judged on, so a cosmetic release does not look like a required upgrade.
//
// The protocol is in beta, so it is versioned below 1.0: the compatibility unit
// is the **minor** (a minor bump is breaking, a patch bump is additive). Once
// it graduates to 1.0 the unit becomes the major, as usual. See
// docs/agent-compat.md for the bump policy.
package protocol

import "golang.org/x/mod/semver"

// Version is the current agent↔agency protocol version. Both the agent and the
// agency import this package, so a freshly built agent always reports the
// agency's current protocol.
//
// Pre-1.0 (beta): bump the *minor* for a breaking change (the tunnel hello
// handshake or the agent-tool REST contract) and the *patch* for an additive
// change; patch-only fixes that change no interface still take a patch bump.
// Post-1.0 this flips back to the usual major/minor/patch.
const Version = "v0.1.0"

// Compatibility is the agency's verdict on an agent's reported protocol
// version. It is never persisted: it is derived by comparing the reported
// version with the running agency's Version.
type Compatibility int64

const (
	// Incompatible means the reported version is missing, unparseable, or on a
	// different compatibility line. The tunnel refuses the connection.
	Incompatible Compatibility = iota
	// Outdated means the same compatibility line but an older version: usable,
	// but the binary should be updated.
	Outdated
	// Current means the same compatibility line and equal or newer: nothing to
	// do.
	Current
)

// Compare reports how an agent's protocol version relates to the agency's
// current version.
//
// It keys on the compatibility line returned by compatLine: the major once it
// is non-zero, otherwise the minor. A different line (or a version that cannot
// be parsed) is Incompatible, a version behind the agency on the same line is
// Outdated, and anything else is Current.
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

// compatLine is the version segment the compatibility verdict is keyed on: the
// major once it is non-zero, else the minor (pre-1.0 beta, where a minor bump
// is breaking).
func compatLine(v string) string {
	if semver.Major(v) == "v0" {
		return semver.MajorMinor(v)
	}
	return semver.Major(v)
}
