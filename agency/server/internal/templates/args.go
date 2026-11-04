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

package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("not a boolean: %q", raw)
	}
}

func defaultArgValue(arg Arg) (any, error) {
	switch arg.Type {
	case ArgNumber:
		if arg.DefaultValue == "" {
			return float64(0), nil
		}
		value, err := strconv.ParseFloat(arg.DefaultValue, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"argument %q has a non-numeric default %q",
				arg.Key,
				arg.DefaultValue,
			)
		}
		return value, nil
	case ArgBoolean:
		if arg.DefaultValue == "" {
			return false, nil
		}
		return parseBool(arg.DefaultValue)
	default:
		return arg.DefaultValue, nil
	}
}

func normalizeArgs(specs []Arg, argsJSON string) (map[string]any, string, error) {
	input := map[string]any{}
	if strings.TrimSpace(argsJSON) != "" {
		decoder := json.NewDecoder(bytes.NewReader([]byte(argsJSON)))
		decoder.UseNumber()
		if err := decoder.Decode(&input); err != nil {
			return nil, "", fmt.Errorf("arguments must be a JSON object: %w", err)
		}
		if input == nil {
			input = map[string]any{}
		}
	}
	declared := make(map[string]bool, len(specs))
	values := make(map[string]any, len(specs))
	normalized := make(map[string]any, len(specs))
	for _, arg := range specs {
		declared[arg.Key] = true
		raw, present := input[arg.Key]
		missing := !present || raw == nil ||
			(isString(raw) && strings.TrimSpace(raw.(string)) == "")
		if missing {
			if arg.Required && arg.DefaultValue == "" {
				return nil, "", fmt.Errorf("argument %q is required", arg.Key)
			}
			value, err := defaultArgValue(arg)
			if err != nil {
				return nil, "", err
			}
			values[arg.Key] = value
			normalized[arg.Key] = value
			continue
		}
		value, err := coerceArg(arg, raw)
		if err != nil {
			return nil, "", err
		}
		values[arg.Key] = value
		normalized[arg.Key] = value
	}
	for key := range input {
		if !declared[key] {
			return nil, "", fmt.Errorf("unknown argument %q", key)
		}
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return nil, "", err
	}
	return values, string(raw), nil
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func coerceArg(arg Arg, raw any) (any, error) {
	switch arg.Type {
	case ArgNumber:
		switch value := raw.(type) {
		case json.Number:
			number, err := value.Float64()
			if err != nil {
				return nil, fmt.Errorf("argument %q must be a number", arg.Key)
			}
			return number, nil
		case float64:
			return value, nil
		case int:
			return float64(value), nil
		case string:
			number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				return nil, fmt.Errorf("argument %q must be a number", arg.Key)
			}
			return number, nil
		default:
			return nil, fmt.Errorf("argument %q must be a number", arg.Key)
		}
	case ArgBoolean:
		switch value := raw.(type) {
		case bool:
			return value, nil
		case string:
			parsed, err := parseBool(value)
			if err != nil {
				return nil, fmt.Errorf("argument %q must be a boolean", arg.Key)
			}
			return parsed, nil
		case json.Number:
			if value.String() == "0" {
				return false, nil
			}
			if value.String() == "1" {
				return true, nil
			}
			return nil, fmt.Errorf("argument %q must be a boolean", arg.Key)
		default:
			return nil, fmt.Errorf("argument %q must be a boolean", arg.Key)
		}
	case ArgSelect:
		value := fmt.Sprint(raw)
		if !hasOption(arg.Options, value) {
			return nil, fmt.Errorf("argument %q must be one of its options", arg.Key)
		}
		return value, nil
	default:
		return fmt.Sprint(raw), nil
	}
}
