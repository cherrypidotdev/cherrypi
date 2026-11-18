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
	"reflect"
	"unsafe"

	piagent "github.com/sky-valley/pi/agent"
)

func enableAllSteering(a *piagent.Agent) {
	if a == nil {
		return
	}
	queue := reflect.ValueOf(a).Elem().FieldByName("steeringQueue")
	if !queue.IsValid() {
		return
	}
	mode := queue.FieldByName("mode")
	if !mode.IsValid() || mode.Kind() != reflect.String {
		return
	}
	reflect.NewAt(mode.Type(), unsafe.Pointer(mode.UnsafeAddr())).
		Elem().
		SetString(string(piagent.QueueAll))
}
