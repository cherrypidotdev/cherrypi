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

package session

import (
	"encoding/json"
)

const thinkingSpanCustomType = "cherrypi_thinking_span"

type thinkingSpanData struct {
	DurationMs int64 `json:"durationMs"`
	Chars      int32 `json:"chars"`
}

func parseThinkingSpan(raw json.RawMessage) (*ThinkingSpanEvent, bool) {
	var data thinkingSpanData
	if json.Unmarshal(raw, &data) != nil || data.DurationMs <= 0 {
		return nil, false
	}
	return &ThinkingSpanEvent{
		DurationMs: data.DurationMs,
		Chars:      data.Chars,
	}, true
}
