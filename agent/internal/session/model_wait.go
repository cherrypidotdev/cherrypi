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

const modelWaitCustomType = "cherrypi_model_wait"

type modelWaitData struct {
	Kind       int64  `json:"kind"`
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error"`
	Attempt    uint32 `json:"attempt"`
	Reason     string `json:"reason"`
}

func parseModelWait(raw json.RawMessage) (*ModelWaitEvent, bool) {
	var data modelWaitData
	if json.Unmarshal(raw, &data) != nil || data.DurationMs <= 0 {
		return nil, false
	}
	kind := PendingKind(data.Kind)
	switch kind {
	case PendingModelRequest, PendingRetry, PendingCompaction:
	default:
		return nil, false
	}
	return &ModelWaitEvent{
		Kind:       kind,
		Model:      data.Model,
		Provider:   data.Provider,
		DurationMs: data.DurationMs,
		Error:      data.Error,
		Attempt:    data.Attempt,
		Reason:     data.Reason,
	}, true
}
