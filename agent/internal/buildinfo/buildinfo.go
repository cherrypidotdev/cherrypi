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

// Package buildinfo carries the agent's build version, injected at link time.
package buildinfo

// Version is the agent build version, set by the build with
//
//	-ldflags "-X cherrypi.vladfaust.com/agent/internal/buildinfo.Version=…"
//
// It is the git-describe stamp baked into the image (see the Makefile and
// agent/Dockerfile), for operators to identify the exact binary. It is not the
// protocol version: compatibility is judged on shared/protocol.Version.
var Version = "dev"
