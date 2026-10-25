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

package audit

import (
	"encoding/csv"
	"encoding/json"
	"io"

	"cherrypi.vladfaust.com/agency/internal/actorid"
)

var CSVHeader = []string{
	"at", "scope", "scope_id", "scope_label", "actor_id", "actor_kind",
	"actor_username", "actor_name", "kind", "field", "old_value", "new_value",
	"message", "data_json",
}

func WriteCSV(w io.Writer, entries []Entry) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(CSVHeader); err != nil {
		return err
	}
	for _, entry := range entries {
		row := []string{
			entry.At.UTC().Format("2006-01-02T15:04:05.000Z"),
			string(entry.Scope),
			entry.ScopeID,
			entry.ScopeLabel,
			actorid.Format(entry.ActorID),
			entry.ActorKind,
			entry.ActorUsername,
			entry.ActorName,
			entry.Kind,
			entry.Field,
			entry.OldValue,
			entry.NewValue,
			entry.Message,
			entry.DataJSON,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func WriteJSON(w io.Writer, entries []Entry) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(EntriesJSON(entries))
}

func EntriesJSON(entries []Entry) []map[string]any {
	out := make([]map[string]any, len(entries))
	for i, entry := range entries {
		out[i] = EntryJSON(entry)
	}
	return out
}

func EntryJSON(entry Entry) map[string]any {
	return map[string]any{
		"id":             entry.ID,
		"scope":          string(entry.Scope),
		"scope_id":       entry.ScopeID,
		"scope_label":    entry.ScopeLabel,
		"at":             entry.At.UTC().Format("2006-01-02T15:04:05.000Z"),
		"actor_id":       actorid.Format(entry.ActorID),
		"actor_kind":     entry.ActorKind,
		"actor_username": entry.ActorUsername,
		"actor_name":     entry.ActorName,
		"kind":           entry.Kind,
		"field":          entry.Field,
		"old_value":      entry.OldValue,
		"new_value":      entry.NewValue,
		"message":        entry.Message,
		"data_json":      entry.DataJSON,
	}
}
