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

//go:build dev

package main

import "io/fs"

// devBuild reports whether this is a development build (`-tags dev`): it may
// proxy the frontend to the Vite dev server via CHERRYPI_FRONTEND_PROXY_URL and
// serves no embedded assets.
const devBuild = true

// frontendFS returns nothing in dev — the frontend is served by Vite, so no
// assets are embedded.
func frontendFS() (fs.FS, error) {
	return nil, fs.ErrNotExist
}
