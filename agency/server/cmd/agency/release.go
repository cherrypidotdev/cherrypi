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

//go:build !dev

package main

import (
	"embed"
	"io/fs"
)

// devBuild is false for release builds: the frontend is always embedded and
// CHERRYPI_FRONTEND_PROXY_URL is ignored.
const devBuild = false

// The blank declaration below is a compile-time canary: the release build
// fails if the embedded SPA is missing, because frontenddist/index.html must
// exist. (The `*` glob below would happily match the .gitkeep placeholder on
// its own.)
//
//go:embed frontenddist/index.html
var _ string

//go:embed frontenddist/*
var frontendDistFs embed.FS

// frontendFS returns the embedded SPA filesystem. Release builds embed by
// construction; only dev builds return an error.
func frontendFS() (fs.FS, error) {
	return fs.Sub(frontendDistFs, "frontenddist")
}
