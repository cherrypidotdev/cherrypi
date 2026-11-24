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

package taskhistory

import (
	"encoding/json"

	"cherrypi.vladfaust.com/shared/agenttools"
	"github.com/sky-valley/pi/coding"
)

func BuiltinBaseline(interactive bool) []Tool {
	names := agenttools.Names(interactive)
	out := make([]Tool, 0, len(names)+4)
	for _, tool := range coding.CreateCodingTools("") {
		entry := Tool{Name: tool.Name, Description: tool.Description}
		if tool.Parameters != nil {
			if raw, err := json.Marshal(tool.Parameters); err == nil {
				entry.ParametersJSON = string(raw)
			}
		}
		out = append(out, entry)
	}
	for _, name := range names {
		out = append(out, Tool{
			Name: name,

			Description: "Built-in CherryPi tool; described when the run starts.",
		})
	}
	return out
}
