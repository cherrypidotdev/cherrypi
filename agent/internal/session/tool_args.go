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
	"slices"
	"strconv"
	"strings"

	"github.com/sky-valley/pi/ai"
)

var argumentAliases = map[string]string{
	"tags":        "tagIds",
	"tag":         "tagIds",
	"refs":        "references",
	"reference":   "references",
	"project":     "projectId",
	"projectslug": "projectId",
	"template":    "templateId",
	"version":     "templateVersion",
	"args":        "templateArgs",
	"arguments":   "templateArgs",
	"assignee":    "assigneeId",
	"actor":       "assigneeId",
	"actorid":     "assigneeId",
	"state":       "outcome",
	"queued":      "queue",
	"queuenow":    "queue",
	"exclude":     "excludeMessages",
	"before":      "beforeId",
	"task":        "taskId",
}

const (
	argumentExact = iota
	argumentNormalized
	argumentAlias
)

type aliasCandidate struct {
	key    string
	target string
	rank   int
}

func normalizeToolArguments(schema *ai.Schema, raw map[string]any) map[string]any {
	if schema == nil || raw == nil {
		return raw
	}
	out := make(map[string]any, len(raw)+len(schema.Properties))

	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	candidates := make([]aliasCandidate, 0, len(raw))
	for _, key := range keys {
		value := raw[key]
		if property, ok := schema.Properties[key]; ok {
			out[key] = normalizeArgumentValue(property, value)
			continue
		}
		if target, rank, ok := declaredArgument(schema, key); ok {
			candidates = append(candidates, aliasCandidate{key: key, target: target, rank: rank})
			continue
		}
		if !schemaClosed(schema) {
			out[key] = value
		}
	}
	slices.SortFunc(candidates, func(a, b aliasCandidate) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return strings.Compare(a.key, b.key)
	})
	for _, candidate := range candidates {
		if _, taken := out[candidate.target]; taken {
			continue
		}
		out[candidate.target] = normalizeArgumentValue(
			schema.Properties[candidate.target],
			raw[candidate.key],
		)
	}
	for name, property := range schema.Properties {
		if _, ok := out[name]; ok || property == nil || !property.Nullable {
			continue
		}
		out[name] = nil
	}
	return out
}

func declaredArgument(schema *ai.Schema, key string) (string, int, bool) {
	if _, ok := schema.Properties[key]; ok {
		return key, argumentExact, true
	}
	normalized := normalizeArgumentName(key)
	for name := range schema.Properties {
		if normalizeArgumentName(name) == normalized {
			return name, argumentNormalized, true
		}
	}
	if target, ok := argumentAliases[normalized]; ok {
		if _, declared := schema.Properties[target]; declared {
			return target, argumentAlias, true
		}
	}
	return "", 0, false
}

func normalizeArgumentName(name string) string {
	var out strings.Builder
	for _, char := range strings.ToLower(name) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			out.WriteRune(char)
		}
	}
	return out.String()
}

func normalizeArgumentValue(schema *ai.Schema, value any) any {
	if schema == nil || value == nil {
		return value
	}
	if len(schema.Enum) > 0 {
		if text, ok := value.(string); ok {
			if match, ok := matchEnum(schema.Enum, text); ok {
				return match
			}
		}
	}
	if len(schema.Properties) > 0 || schema.Type == "object" {
		switch typed := value.(type) {
		case map[string]any:
			return normalizeToolArguments(schema, typed)
		case string:
			var decoded map[string]any
			if json.Unmarshal([]byte(typed), &decoded) == nil {
				return normalizeToolArguments(schema, decoded)
			}
		}
		return value
	}
	if schema.Type == "array" || schema.Items != nil {
		switch typed := value.(type) {
		case string:

			if strings.TrimSpace(typed) == "" {
				return value
			}
			var decoded []any
			if json.Unmarshal([]byte(typed), &decoded) == nil {
				return normalizeArgumentValue(schema, decoded)
			}
			return []any{normalizeArgumentValue(schema.Items, value)}
		case []any:
			out := make([]any, len(typed))
			for i, item := range typed {
				out[i] = normalizeArgumentValue(schema.Items, item)
			}
			return out
		case []string:
			out := make([]any, len(typed))
			for i, item := range typed {
				out[i] = normalizeArgumentValue(schema.Items, item)
			}
			return out
		}
		return []any{normalizeArgumentValue(schema.Items, value)}
	}
	switch schema.Type {
	case "number", "integer":
		if text, ok := value.(string); ok {
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
				return parsed
			}
		}
	case "boolean":
		if text, ok := value.(string); ok {
			switch strings.ToLower(strings.TrimSpace(text)) {
			case "true":
				return true
			case "false":
				return false
			}
		}
	case "string":
		switch value.(type) {
		case map[string]any, []any, float64, int, int64, bool:
			if encoded, err := json.Marshal(value); err == nil {
				return string(encoded)
			}
		}
	}
	return value
}

func matchEnum(values []any, value string) (any, bool) {
	name := enumName(value)
	for _, candidate := range values {
		text, ok := candidate.(string)
		if !ok {
			continue
		}
		if text == value || enumName(text) == name {
			return text, true
		}
	}
	return nil, false
}

func schemaClosed(schema *ai.Schema) bool {
	return schema.AdditionalAllowed != nil && !*schema.AdditionalAllowed
}
