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
	"slices"

	"github.com/sky-valley/pi/agent"
	"github.com/sky-valley/pi/ai"
)

func strictTools(tools []agent.AgentTool) []agent.AgentTool {
	out := make([]agent.AgentTool, len(tools))
	for i, tool := range tools {
		out[i] = tool
		if tool.Parameters == nil || strictDeclaredOpen(tool.Parameters) {
			return nonStrictTools(tools)
		}
		strict := strictSchema(tool.Parameters.Clone(), false)
		if strict.Type != "object" || strict.Nullable || !strictSafe(strict) {
			return nonStrictTools(tools)
		}
		out[i].Parameters = strict
	}
	prefer := &ai.ConstrainedSamplingConfig{
		Type:   ai.ConstrainedSamplingJSONSchema,
		Strict: ai.ConstrainedSamplingPrefer,
	}
	for i := range out {
		out[i].ConstrainedSampling = prefer
	}
	return out
}

func strictDeclaredOpen(schema *ai.Schema) bool {
	if schema == nil {
		return false
	}
	if schema.AdditionalSchema != nil ||
		(schema.AdditionalAllowed != nil && *schema.AdditionalAllowed) {
		return true
	}

	if schema.Type == "object" && len(schema.Properties) == 0 &&
		schema.AdditionalAllowed == nil {
		return true
	}
	for _, property := range schema.Properties {
		if strictDeclaredOpen(property) {
			return true
		}
	}
	if strictDeclaredOpen(schema.Items) {
		return true
	}
	for _, variant := range schema.AnyOf {
		if strictDeclaredOpen(variant) {
			return true
		}
	}
	return false
}

func nonStrictTools(tools []agent.AgentTool) []agent.AgentTool {
	out := make([]agent.AgentTool, len(tools))
	for i, tool := range tools {
		out[i] = tool
		out[i].ConstrainedSampling = nil
	}
	return out
}

func strictSchema(schema *ai.Schema, optional bool) *ai.Schema {
	if schema == nil {
		return nil
	}
	if optional {
		schema.Nullable = true
	}
	if len(schema.Properties) > 0 {
		required := make([]string, 0, len(schema.Properties))
		for name, property := range schema.Properties {
			wasOptional := !containsString(schema.Required, name)
			schema.Properties[name] = strictSchema(property, wasOptional)
			required = append(required, name)
		}
		ordered := make([]string, 0, len(required))
		for _, name := range schema.PropertyOrder {
			if _, ok := schema.Properties[name]; ok {
				ordered = append(ordered, name)
			}
		}
		for _, name := range sortedStrings(required) {
			if !containsString(ordered, name) {
				ordered = append(ordered, name)
			}
		}
		schema.PropertyOrder = ordered
		schema.Required = required
	}
	if schema.Type == "object" || len(schema.Properties) > 0 {
		schema.Type = "object"
		allowed := false
		schema.AdditionalAllowed = &allowed
		schema.AdditionalSchema = nil
	}
	schema.Items = strictSchema(schema.Items, false)
	schema.AdditionalSchema = strictSchema(schema.AdditionalSchema, false)
	for i := range schema.AnyOf {
		schema.AnyOf[i] = strictSchema(schema.AnyOf[i], false)
	}
	for i := range schema.OneOf {
		schema.OneOf[i] = strictSchema(schema.OneOf[i], false)
	}
	for i := range schema.AllOf {
		schema.AllOf[i] = strictSchema(schema.AllOf[i], false)
	}
	stripStrictConstraints(schema)
	return schema
}

func stripStrictConstraints(schema *ai.Schema) {
	schema.Minimum = nil
	schema.Maximum = nil
	schema.ExclusiveMinimum = nil
	schema.ExclusiveMaximum = nil
	schema.MultipleOf = nil
	schema.MinItems = nil
	schema.MaxItems = nil
	schema.MinLength = nil
	schema.MaxLength = nil
	schema.Pattern = ""
	schema.Format = ""
	schema.Default = nil
	for _, key := range []string{
		"uniqueItems", "minContains", "maxContains", "minProperties",
		"maxProperties", "allOf", "oneOf", "format", "default",
	} {
		delete(schema.Extra, key)
	}
}

func strictSafe(schema *ai.Schema) bool {
	if schema == nil {
		return false
	}
	for _, key := range []string{
		"$ref", "$defs", "definitions", "patternProperties", "dependentSchemas",
		"dependencies", "unevaluatedProperties", "propertyNames", "contains",
		"prefixItems", "not", "if", "then", "else", "items",
	} {
		if _, ok := schema.Extra[key]; ok {
			return false
		}
	}
	if schema.AnyOf != nil || schema.OneOf != nil || schema.AllOf != nil {
		return false
	}
	if schema.AdditionalSchema != nil {
		return false
	}
	if schema.Type == "object" {

		if schema.Nullable && len(schema.Properties) > 0 {
			return false
		}
		if schema.AdditionalAllowed == nil || *schema.AdditionalAllowed {
			return false
		}
	}
	if len(schema.Properties) > 0 {
		if schema.Type != "object" || len(schema.Required) != len(schema.Properties) {
			return false
		}
		for _, name := range schema.Required {
			if _, ok := schema.Properties[name]; !ok {
				return false
			}
		}
	}
	for _, property := range schema.Properties {
		if !strictSafe(property) {
			return false
		}
	}
	if schema.Items != nil && !strictSafe(schema.Items) {
		return false
	}
	return true
}

func sortedStrings(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return out
}
