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

package sites

import (
	"bytes"
	_ "embed"
)

//go:embed bridge.js
var bridgeScript string

func injectBridge(html []byte, siteID string) []byte {
	tag := []byte(`<script src="/sites/` + siteID + `/bridge.js"></script>`)
	lower := bytes.ToLower(html)
	if head := bytes.Index(lower, []byte("<head")); head >= 0 {
		if end := bytes.IndexByte(lower[head:], '>'); end >= 0 {
			at := head + end + 1
			out := make([]byte, 0, len(html)+len(tag))
			out = append(out, html[:at]...)
			out = append(out, tag...)
			out = append(out, html[at:]...)
			return out
		}
	}
	return append(tag, html...)
}
