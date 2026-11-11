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

package plugins

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/templates"
)

const (
	MaxTemplateSourceBytes = 64 * 1024
	MaxTemplateOutputBytes = 128 * 1024
)

var (
	templateVariablePattern = regexp.MustCompile(
		`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`,
	)
	settingsGroupIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
)

type SettingsGroup struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Fields      []string `json:"fields"`
	Collapsed   bool     `json:"collapsed,omitempty"`
}

type TemplateVariable struct {
	Type        string                      `json:"type,omitempty"`
	Description string                      `json:"description,omitempty"`
	Sample      json.RawMessage             `json:"sample,omitempty"`
	Items       *TemplateVariable           `json:"items,omitempty"`
	Properties  map[string]TemplateVariable `json:"properties,omitempty"`
}

type TemplateSpec struct {
	Engine    string                      `json:"engine,omitempty"`
	Variables map[string]TemplateVariable `json:"variables,omitempty"`
}

type TemplateField struct {
	Key       string
	Title     string
	Default   string
	Variables map[string]TemplateVariable
}

type TemplatePreview struct {
	Rendered  string
	Errors    []string
	Variables []string
}

type settingsPropertyMeta struct {
	Type      string          `json:"type"`
	Format    string          `json:"format"`
	Title     string          `json:"title"`
	Default   json.RawMessage `json:"default"`
	XTemplate *TemplateSpec   `json:"x-template"`
}

type settingsSchemaMeta struct {
	Properties map[string]settingsPropertyMeta `json:"properties"`
	Groups     []SettingsGroup                 `json:"x-groups"`
}

func parseSettingsSchema(raw json.RawMessage) (settingsSchemaMeta, error) {
	var meta settingsSchemaMeta
	if len(raw) == 0 {
		return meta, nil
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func (m Manifest) SettingsGroups(scope string) []SettingsGroup {
	meta, err := parseSettingsSchema(m.Settings[scope])
	if err != nil {
		return nil
	}
	return meta.Groups
}

func (m Manifest) TemplateFields(scope string) map[string]TemplateField {
	meta, err := parseSettingsSchema(m.Settings[scope])
	if err != nil {
		return map[string]TemplateField{}
	}
	out := map[string]TemplateField{}
	for key, property := range meta.Properties {
		if property.Format != "template" || property.XTemplate == nil {
			continue
		}
		var def string
		if len(property.Default) > 0 {
			_ = json.Unmarshal(property.Default, &def)
		}
		out[key] = TemplateField{
			Key:       key,
			Title:     property.Title,
			Default:   def,
			Variables: property.XTemplate.Variables,
		}
	}
	return out
}

func (m Manifest) TemplateField(scope, key string) (TemplateField, bool) {
	field, ok := m.TemplateFields(scope)[key]
	return field, ok
}

func (m Manifest) SettingsDefaults(scope string) map[string]any {
	out := map[string]any{}
	for key, field := range m.TemplateFields(scope) {
		if field.Default != "" {
			out[key] = field.Default
		}
	}
	return out
}

func (m Manifest) ApplySettingsDefaults(scope string, values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	for key, value := range m.SettingsDefaults(scope) {
		if _, ok := out[key]; !ok {
			out[key] = value
		}
	}
	return out
}

func validateSettingsSchema(name string, raw json.RawMessage) error {
	meta, err := parseSettingsSchema(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	propertyKeys := map[string]bool{}
	for key, property := range meta.Properties {
		propertyKeys[key] = true
		if property.Format != "template" {
			if property.XTemplate != nil {
				return fmt.Errorf(
					"%s.%s: x-template is only allowed on format: template",
					name, key,
				)
			}
			continue
		}
		if property.Type != "" && property.Type != "string" {
			return fmt.Errorf("%s.%s: a template must have type string", name, key)
		}
		if property.XTemplate == nil {
			return fmt.Errorf(
				"%s.%s: format: template needs an x-template object", name, key,
			)
		}
		if property.XTemplate.Engine != "" && property.XTemplate.Engine != "jinja" {
			return fmt.Errorf(
				"%s.%s: x-template.engine must be jinja", name, key,
			)
		}
		if len(property.Default) == 0 {
			return fmt.Errorf(
				"%s.%s: format: template needs a string default", name, key,
			)
		}
		var def string
		if err := json.Unmarshal(property.Default, &def); err != nil {
			return fmt.Errorf(
				"%s.%s: format: template needs a string default", name, key,
			)
		}
		if def != "" {
			if err := validateTemplateSource(def, property.XTemplate.Variables); err != nil {
				return fmt.Errorf("%s.%s default: %w", name, key, err)
			}
		}
	}

	seenGroups := map[string]bool{}
	seenFields := map[string]bool{}
	for _, group := range meta.Groups {
		if !settingsGroupIDPattern.MatchString(group.ID) {
			return fmt.Errorf(
				"%s.x-groups: id %q must be a lowercase slug", name, group.ID,
			)
		}
		if seenGroups[group.ID] {
			return fmt.Errorf("%s.x-groups: duplicate id %q", name, group.ID)
		}
		seenGroups[group.ID] = true
		if strings.TrimSpace(group.Title) == "" {
			return fmt.Errorf("%s.x-groups.%s: title is required", name, group.ID)
		}
		for _, field := range group.Fields {
			if !propertyKeys[field] {
				return fmt.Errorf(
					"%s.x-groups.%s: unknown field %q", name, group.ID, field,
				)
			}
			if seenFields[field] {
				return fmt.Errorf(
					"%s.x-groups: field %q is in more than one group",
					name, field,
				)
			}
			seenFields[field] = true
		}
	}
	return nil
}

func declaresTemplateVariable(
	variables map[string]TemplateVariable, name string,
) bool {
	if _, ok := variables[name]; ok {
		return true
	}
	prefix := name + "."
	for path := range variables {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func validateTemplateSource(
	source string, variables map[string]TemplateVariable,
) error {
	if len(source) > MaxTemplateSourceBytes {
		return fmt.Errorf(
			"the template is larger than %d bytes", MaxTemplateSourceBytes,
		)
	}
	used, unsupported, err := templates.ScanVariables(source)
	if err != nil {
		return err
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("tags %s are not supported", strings.Join(unsupported, ", "))
	}
	for _, name := range used {
		if !declaresTemplateVariable(variables, name) {
			return fmt.Errorf("undeclared variable %q", name)
		}
	}
	if err := templates.Compile(source); err != nil {
		return err
	}
	sample, err := templateSampleContext(variables)
	if err != nil {
		return err
	}
	if _, err := templates.Render(source, sample); err != nil {
		return err
	}
	return nil
}

func (f TemplateField) Preview(
	source string, overrides map[string]any,
) TemplatePreview {
	if strings.TrimSpace(source) == "" {
		source = f.Default
	}
	out := TemplatePreview{}
	if source == "" {
		out.Errors = append(out.Errors, "the template is empty")
		return out
	}
	if len(source) > MaxTemplateSourceBytes {
		out.Errors = append(out.Errors, fmt.Sprintf(
			"the template is larger than %d bytes", MaxTemplateSourceBytes,
		))
		return out
	}
	used, unsupported, err := templates.ScanVariables(source)
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
		return out
	}
	if len(unsupported) > 0 {
		out.Errors = append(out.Errors, fmt.Sprintf(
			"tags %s are not supported", strings.Join(unsupported, ", "),
		))
	}
	for _, name := range used {
		if !declaresTemplateVariable(f.Variables, name) {
			out.Errors = append(out.Errors, fmt.Sprintf(
				"undeclared variable %q", name,
			))
		}
	}
	if len(out.Errors) > 0 {
		return out
	}
	sample, err := templateSampleContext(f.Variables)
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
		return out
	}
	for path, value := range overrides {
		setNestedValue(sample, path, value)
	}
	rendered, err := templates.Render(source, sample)
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
		return out
	}
	if len(rendered) > MaxTemplateOutputBytes {
		out.Errors = append(out.Errors, fmt.Sprintf(
			"the rendered text is larger than %d bytes", MaxTemplateOutputBytes,
		))
		return out
	}
	out.Rendered = rendered
	out.Variables = used
	return out
}

func (m Manifest) ValidateSettingsTemplates(
	scope string, values map[string]any,
) []SettingError {
	fields := m.TemplateFields(scope)
	out := make([]SettingError, 0)
	for key, field := range fields {
		raw, ok := values[key]
		if !ok {
			continue
		}
		source, ok := raw.(string)
		if !ok {
			out = append(out, SettingError{
				Path:      key,
				ErrorText: "the template must be a string",
			})
			continue
		}
		if strings.TrimSpace(source) == "" {
			continue
		}
		preview := field.Preview(source, nil)
		if len(preview.Errors) > 0 {
			out = append(out, SettingError{
				Path:      key,
				ErrorText: strings.Join(preview.Errors, "; "),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func templateSampleContext(
	variables map[string]TemplateVariable,
) (map[string]any, error) {
	out := map[string]any{}
	for path, variable := range variables {
		if !templateVariablePattern.MatchString(path) {
			return nil, fmt.Errorf("variable %q is not a valid path", path)
		}
		value, err := templateSampleValue(variable)
		if err != nil {
			return nil, fmt.Errorf("variable %q: %w", path, err)
		}
		setNestedValue(out, path, value)
	}
	return out, nil
}

func templateSampleValue(variable TemplateVariable) (any, error) {
	if len(variable.Sample) > 0 {
		var value any
		if err := json.Unmarshal(variable.Sample, &value); err != nil {
			return nil, fmt.Errorf("sample is not valid JSON")
		}
		return value, nil
	}
	switch variable.Type {
	case "array":
		return []any{}, nil
	case "object":
		return map[string]any{}, nil
	case "number", "integer":
		return float64(0), nil
	case "boolean":
		return false, nil
	default:
		return "", nil
	}
}

func setNestedValue(root map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}
