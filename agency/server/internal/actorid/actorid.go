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

// Package actorid converts between an actor's serial id and its opaque wire
// form. The gRPC surface still carries the id as a string; the domain layer
// uses the integer.
package actorid

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Parse returns the integer id for a wire value, or 0 when empty or invalid.
func Parse(s string) int64 {
	if s == "" {
		return 0
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// Format renders an actor id for the wire. A zero id becomes the empty string.
func Format[T ~int64](id T) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(int64(id), 10)
}

// ParsePtr parses an optional wire id, returning nil for a nil pointer.
func ParsePtr(s *string) *int64 {
	if s == nil {
		return nil
	}
	id := Parse(*s)
	return &id
}

// ParseAgent parses a wire value as an agent id.
func ParseAgent(s string) ids.AgentID { return ids.AgentID(Parse(s)) }

// ParseActor parses a wire value as an actor id.
func ParseActor(s string) ids.ActorID { return ids.ActorID(Parse(s)) }

// ParseActorPtr parses an optional wire value as an actor id.
func ParseActorPtr(s *string) *ids.ActorID {
	if s == nil {
		return nil
	}
	id := ids.ActorID(Parse(*s))
	return &id
}

// WireID is an actor id in its REST JSON form. It accepts either a JSON number
// or a numeric string, because the proto/SDK surface carries ids as strings
// while a hand-written client may send a number. A `null` or `""` value means
// unset (0).
//
// WireID also remembers whether the JSON key was present, so an update can
// tell "leave the assignee unchanged" from "clear the pin".
type WireID struct {
	id  int64
	set bool
}

// UnmarshalJSON implements [json.Unmarshaler]. A malformed JSON number is an
// error; a non-numeric string parses to 0 like [Parse].
func (w *WireID) UnmarshalJSON(data []byte) error {
	w.set = true
	text := strings.TrimSpace(string(data))
	if text == "null" {
		w.id = 0
		return nil
	}
	if strings.HasPrefix(text, `"`) {
		var quoted string
		if err := json.Unmarshal(data, &quoted); err != nil {
			return err
		}
		w.id = Parse(quoted)
		return nil
	}
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid actor id %q", text)
	}
	w.id = id
	return nil
}

// MarshalJSON implements [json.Marshaler], rendering the id as the string the
// proto/SDK surface uses (empty when unset).
func (w WireID) MarshalJSON() ([]byte, error) {
	return json.Marshal(Format(w.id))
}

// Set reports whether the JSON key was present. An explicit empty value sets
// it, which is how an update clears the assignee pin.
func (w WireID) Set() bool { return w.set }

// ID returns the parsed id as an actor id.
func (w WireID) ID() ids.ActorID { return ids.ActorID(w.id) }

// AgentID returns the parsed id as an agent id.
func (w WireID) AgentID() ids.AgentID { return ids.AgentID(w.id) }
