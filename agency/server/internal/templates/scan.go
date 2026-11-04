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
	"fmt"

	"github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/tokens"
)

var unsupportedTags = map[string]bool{
	"include":    true,
	"extends":    true,
	"import":     true,
	"from":       true,
	"macro":      true,
	"call":       true,
	"block":      true,
	"autoescape": true,
	"trans":      true,
	"endtrans":   true,
	"pluralize":  true,
	"do":         true,
}

var jinjaKeywords = map[string]bool{
	"if": true, "elif": true, "else": true, "endif": true,
	"for": true, "endfor": true, "in": true, "recursive": true,
	"set": true, "endset": true, "with": true, "endwith": true,
	"filter": true, "endfilter": true, "raw": true, "endraw": true,
	"comment": true, "endcomment": true,
	"macro": true, "endmacro": true, "call": true, "endcall": true,
	"block": true, "endblock": true, "include": true, "extends": true,
	"import": true, "from": true, "as": true, "do": true,
	"autoescape": true, "endautoescape": true,
	"trans": true, "endtrans": true, "pluralize": true,
	"and": true, "or": true, "not": true, "is": true,
	"true": true, "false": true, "none": true,
	"True": true, "False": true, "None": true,
}

var jinjaGlobals = map[string]bool{
	"range": true, "dict": true, "lipsum": true, "cycler": true,
	"joiner": true, "namespace": true, "gettext": true, "ngettext": true,
	"_": true, "loop": true, "self": true, "super": true,
	"varargs": true, "kwargs": true, "caller": true,
}

func scanVariables(source string) ([]string, []string, error) {
	stream := tokens.LexAll(source, config.New())
	var (
		vars        []string
		unsupported []string
		locals      = map[string]bool{}
		atBlockTag  bool
		blockTag    string
		forTargets  bool
		setTarget   bool
		skipName    bool
	)
	for !stream.EOF() {
		tok := stream.Current()
		if tok == nil {
			break
		}
		switch tok.Type {
		case tokens.Error:
			return nil, nil, fmt.Errorf("template syntax error: %s", tok.Val)
		case tokens.BlockBegin:
			atBlockTag = true
			blockTag = ""
			forTargets = false
			setTarget = false
			skipName = false
		case tokens.VariableBegin:
			atBlockTag = false
			skipName = false
		case tokens.BlockEnd:
			atBlockTag = false
			blockTag = ""
			forTargets = false
			setTarget = false
			skipName = false
		case tokens.VariableEnd:
			skipName = false
		case tokens.Pipe, tokens.Dot, tokens.Is:
			skipName = true
		case tokens.In:
			if blockTag == "for" {
				forTargets = false
			}
		case tokens.Assign:
			if blockTag == "set" || blockTag == "with" {
				setTarget = false
			}
		case tokens.Name:
			name := tok.Val
			if atBlockTag {
				atBlockTag = false
				blockTag = name
				if unsupportedTags[name] {
					unsupported = append(unsupported, name)
				}
				switch name {
				case "for":
					forTargets = true
				case "set", "with":
					setTarget = true
				case "filter":
					skipName = true
				}
				break
			}
			if skipName {
				skipName = false
				break
			}
			if forTargets {
				locals[name] = true
				break
			}
			if setTarget {
				locals[name] = true
				break
			}
			if jinjaKeywords[name] || jinjaGlobals[name] || locals[name] {
				break
			}

			if next := stream.Peek(); next != nil && next.Type == tokens.Assign {
				break
			}
			vars = append(vars, name)
		}
		stream.Next()
	}
	return uniqueSorted(vars), uniqueSorted(unsupported), nil
}
