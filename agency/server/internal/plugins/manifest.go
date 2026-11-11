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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`

	Runtime string `json:"runtime"`

	RequiresAgency string `json:"requires_agency"`

	Entry string `json:"entry"`

	Handlers []string `json:"handlers"`

	Schedule *Schedule `json:"schedule"`

	Settings map[string]json.RawMessage `json:"settings"`

	Tools []Tool `json:"tools"`

	ProjectTools []string `json:"project_tools"`

	Events []string `json:"events"`

	Network []string `json:"network"`

	Routes []Route `json:"routes"`
	Limits Limits  `json:"limits"`
}

type Schedule struct {
	EverySeconds int `json:"every_seconds"`
}

const (
	MinTickIntervalSeconds = 5
	MaxTickIntervalSeconds = 3600
)

type Limits struct {
	TimeoutSeconds int   `json:"timeout_seconds"`
	MaxResponse    int64 `json:"max_response_bytes"`
	MaxBody        int64 `json:"max_request_bytes"`
	MaxConcurrent  int   `json:"max_concurrent"`
}

type Route struct {
	Path        string   `json:"path"`
	Methods     []string `json:"methods"`
	Description string   `json:"description"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

const (
	ScopeOrg     = "org"
	ScopeProject = "project"
	ScopeAgent   = "agent"
	ScopeUser    = "user"
)

const (
	HandlerHTTP  = "http"
	HandlerTool  = "tool"
	HandlerEvent = "event"
	HandlerTick  = "tick"

	HandlerSettings = "settings"

	HandlerTaskTools = "task_tools"
)

var validEvents = map[string]bool{
	"task_created":         true,
	"task_started":         true,
	"task_settled":         true,
	"task_held":            true,
	"task_hold_released":   true,
	"task_requested_input": true,
	"task_completed":       true,
	"task_failed":          true,
	"task_cancelled":       true,
}

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,47}$`)
var entryPattern = regexp.MustCompile(`^[A-Za-z0-9_./-]+\.lua$`)
var toolNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (m *Manifest) Validate() error {
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Version = strings.TrimSpace(m.Version)
	m.Runtime = strings.TrimSpace(m.Runtime)
	m.Entry = strings.TrimSpace(m.Entry)
	if !slugPattern.MatchString(m.ID) {
		return errors.New("manifest id must be a lowercase slug (2-48 chars)")
	}
	if m.Name == "" {
		return errors.New("manifest name is required")
	}
	if m.Version == "" {
		return errors.New("manifest version is required")
	}
	if m.Runtime == "" {
		m.Runtime = "lua"
	}
	if m.Runtime != "lua" {
		return fmt.Errorf(
			"runtime %q is not available yet; this agency runs lua plugins",
			m.Runtime,
		)
	}
	if m.RequiresAgency != "" {
		if _, err := ParseVersionRange(m.RequiresAgency); err != nil {
			return err
		}
	}
	if m.Entry == "" {
		m.Entry = "main.lua"
	}
	if !entryPattern.MatchString(m.Entry) || strings.Contains(m.Entry, "..") {
		return fmt.Errorf("entry %q must be a .lua path inside the bundle", m.Entry)
	}

	anyHost := false
	for i, host := range m.Network {
		normalized, err := normalizeNetworkEntry(host)
		if err != nil {
			return err
		}
		if normalized == NetworkAnyHost {
			anyHost = true
		}
		m.Network[i] = normalized
	}
	if anyHost && len(m.Network) > 1 {
		return fmt.Errorf(
			"network %q must be the only entry when it allows any host",
			NetworkAnyHost,
		)
	}

	if len(m.Settings) > 0 {
		normalized := make(map[string]json.RawMessage, len(m.Settings))
		for scope, raw := range m.Settings {
			scope = strings.TrimSpace(scope)
			switch scope {
			case ScopeOrg, ScopeProject, ScopeAgent, ScopeUser:
			default:
				return fmt.Errorf("unknown settings surface %q", scope)
			}
			if err := validateSchema("settings."+scope, raw); err != nil {
				return err
			}
			if err := validateSettingsSchema("settings."+scope, raw); err != nil {
				return err
			}
			normalized[scope] = raw
		}
		m.Settings = normalized
	}

	seenTools := map[string]bool{}
	for i := range m.Tools {
		t := &m.Tools[i]
		t.Name = strings.TrimSpace(t.Name)
		if !toolNamePattern.MatchString(t.Name) {
			return fmt.Errorf("tool name %q must be a Lua identifier", t.Name)
		}
		if seenTools[t.Name] {
			return fmt.Errorf("tool name %q is duplicated", t.Name)
		}
		seenTools[t.Name] = true
		if strings.TrimSpace(t.Description) == "" {
			return fmt.Errorf("tool %q needs a description", t.Name)
		}
		if len(t.InputSchema) == 0 {
			t.InputSchema = json.RawMessage(`{"type":"object"}`)
		}
		if err := validateSchema("tool "+t.Name, t.InputSchema); err != nil {
			return err
		}
	}

	seenProjectTools := map[string]bool{}
	for i, name := range m.ProjectTools {
		name = strings.TrimSpace(name)
		if !seenTools[name] {
			return fmt.Errorf(
				"project tool %q must be declared in tools", name,
			)
		}
		if seenProjectTools[name] {
			return fmt.Errorf("project tool %q is duplicated", name)
		}
		seenProjectTools[name] = true
		m.ProjectTools[i] = name
	}

	seenEvents := map[string]bool{}
	for i, event := range m.Events {
		event = strings.TrimSpace(event)
		if !validEvents[event] {
			return fmt.Errorf("unknown event %q", event)
		}
		if seenEvents[event] {
			return fmt.Errorf("event %q is duplicated", event)
		}
		seenEvents[event] = true
		m.Events[i] = event
	}

	seenRoutes := map[string]bool{}
	for i := range m.Routes {
		r := &m.Routes[i]
		r.Path = strings.TrimSpace(r.Path)
		if r.Path == "" || !strings.HasPrefix(r.Path, "/") ||
			strings.Contains(r.Path, "..") || strings.ContainsAny(r.Path, " \t") {
			return fmt.Errorf("route %q must be an absolute plugin path", r.Path)
		}
		if len(r.Methods) == 0 {
			return fmt.Errorf("route %q has no methods", r.Path)
		}
		for j, method := range r.Methods {
			r.Methods[j] = strings.ToUpper(strings.TrimSpace(method))
		}
		key := strings.Join(r.Methods, ",") + " " + r.Path
		if seenRoutes[key] {
			return fmt.Errorf("duplicate route %s", key)
		}
		seenRoutes[key] = true
	}

	if m.Limits.TimeoutSeconds < 0 || m.Limits.MaxResponse < 0 ||
		m.Limits.MaxBody < 0 || m.Limits.MaxConcurrent < 0 {
		return errors.New("limits must not be negative")
	}
	if m.Schedule != nil {
		if m.Schedule.EverySeconds < MinTickIntervalSeconds ||
			m.Schedule.EverySeconds > MaxTickIntervalSeconds {
			return fmt.Errorf(
				"schedule.every_seconds must be between %d and %d",
				MinTickIntervalSeconds,
				MaxTickIntervalSeconds,
			)
		}
	}
	return nil
}

func validateSchema(name string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) {
		return fmt.Errorf("%s must be valid JSON", name)
	}
	var shape struct {
		Type  string          `json:"type"`
		Setup json.RawMessage `json:"setup"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if shape.Type != "" && shape.Type != "object" {
		return fmt.Errorf("%s must be an object schema", name)
	}

	if strings.HasPrefix(name, "settings.") && len(shape.Setup) > 0 {
		if err := validateSetupBlock(name, shape.Setup); err != nil {
			return err
		}
	}
	return nil
}

func validateSetupBlock(name string, raw json.RawMessage) error {
	var markdown string
	if err := json.Unmarshal(raw, &markdown); err == nil {
		if strings.TrimSpace(markdown) == "" {
			return fmt.Errorf("%s.setup must not be empty", name)
		}
		return nil
	}
	var block struct {
		Title    string `json:"title"`
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(raw, &block); err != nil {
		return fmt.Errorf(
			"%s.setup must be a Markdown string or an object with a markdown string",
			name,
		)
	}
	if strings.TrimSpace(block.Markdown) == "" {
		return fmt.Errorf("%s.setup.markdown is required", name)
	}
	return nil
}

func (m *Manifest) NormalizeHandlers() {
	seen := map[string]bool{}
	out := make([]string, 0, 3)
	for _, h := range m.Handlers {
		h = strings.TrimSpace(h)
		switch h {
		case HandlerHTTP, HandlerTool, HandlerEvent, HandlerTick,
			HandlerTaskTools:
			if !seen[h] {
				seen[h] = true
				out = append(out, h)
			}
		}
	}
	if len(m.Tools) > 0 && !seen[HandlerTool] {
		out = append(out, HandlerTool)
	}
	if len(m.Events) > 0 && !seen[HandlerEvent] {
		out = append(out, HandlerEvent)
	}
	if m.Schedule != nil && !seen[HandlerTick] {
		out = append(out, HandlerTick)
	}
	if len(out) == 0 {
		out = append(out, HandlerHTTP)
	}
	sort.Strings(out)
	m.Handlers = out
}

func (m Manifest) RequiresAPIVersion(version string) (bool, error) {
	r, err := ParseVersionRange(m.RequiresAgency)
	if err != nil {
		return false, err
	}
	return r.Matches(version), nil
}

func (m Manifest) TickInterval() time.Duration {
	if m.Schedule == nil || m.Schedule.EverySeconds <= 0 {
		return 0
	}
	return time.Duration(m.Schedule.EverySeconds) * time.Second
}

func (m Manifest) HasHandler(kind string) bool {
	for _, h := range m.Handlers {
		if h == kind {
			return true
		}
	}
	return false
}

func (m Manifest) HasSurface(scope string) bool {
	_, ok := m.Settings[scope]
	return ok
}

func (m Manifest) declaresProjectTool(name string) bool {
	for _, declared := range m.ProjectTools {
		if declared == name {
			return true
		}
	}
	return false
}

func (m Manifest) Surfaces() []string {
	out := make([]string, 0, len(m.Settings))
	for scope := range m.Settings {
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}

func (m Manifest) SettingsSchema(scope string) json.RawMessage {
	raw, ok := m.Settings[scope]
	if !ok || len(raw) == 0 || !json.Valid(raw) {
		return json.RawMessage(`{"type":"object","additionalProperties":true}`)
	}
	return raw
}

func (m Manifest) RenderSettingsSchema(
	scope, target, agencyURL string,
) json.RawMessage {
	agencyURL = strings.TrimRight(agencyURL, "/")
	values := [][2]string{
		{"agencyUrl", agencyURL},
		{"pluginId", m.ID},
		{"pluginName", m.Name},
		{"pluginUrl", agencyURL + "/plugins/" + m.ID},
		{"scope", scope},
		{"targetId", target},
	}
	return renderSchemaPlaceholders(m.SettingsSchema(scope), values)
}

func renderSchemaPlaceholders(
	raw json.RawMessage, values [][2]string,
) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	out := string(raw)
	for _, pair := range values {
		encoded, err := json.Marshal(pair[1])
		if err != nil || len(encoded) < 2 {
			continue
		}
		value := string(encoded[1 : len(encoded)-1])
		out = strings.ReplaceAll(out, "{{ "+pair[0]+" }}", value)
		out = strings.ReplaceAll(out, "{{"+pair[0]+"}}", value)
	}
	rendered := json.RawMessage(out)
	if !json.Valid(rendered) {

		return raw
	}
	return rendered
}

func (m Manifest) SecretKeys(scope string) map[string]bool {
	raw, ok := m.Settings[scope]
	if !ok || len(raw) == 0 {
		return map[string]bool{}
	}
	var schema struct {
		Properties map[string]struct {
			Secret bool   `json:"secret"`
			Format string `json:"format"`
		} `json:"properties"`
	}
	out := map[string]bool{}
	if json.Unmarshal(raw, &schema) != nil {
		return out
	}
	for key, field := range schema.Properties {
		if field.Secret || field.Format == "password" {
			out[key] = true
		}
	}
	return out
}

func (m Manifest) Tool(name string) (Tool, bool) {
	for _, t := range m.Tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

func (m Manifest) SubscribesTo(event string) bool {
	for _, e := range m.Events {
		if e == event {
			return true
		}
	}
	return false
}

const NetworkAnyHost = "*"

func (m Manifest) NetworkAllowsAnyHost() bool {
	for _, host := range m.Network {
		if host == NetworkAnyHost {
			return true
		}
	}
	return false
}

func normalizeNetworkEntry(raw string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "" || strings.ContainsAny(host, "/ \t") {
		return "", fmt.Errorf("network host %q is not a hostname", raw)
	}
	switch {
	case host == NetworkAnyHost:
		return NetworkAnyHost, nil
	case strings.HasPrefix(host, "*."):
		suffix := strings.TrimPrefix(host, "*.")
		if suffix == "" || strings.Contains(suffix, "*") ||
			strings.ContainsAny(suffix, "/ \t") {
			return "", fmt.Errorf(
				"network host %q is not a valid subdomain wildcard",
				raw,
			)
		}
		return "*." + suffix, nil
	case strings.Contains(host, "*"):
		return "", fmt.Errorf(
			"network host %q: use %q for any host or \"*.example.com\" for subdomains",
			raw,
			NetworkAnyHost,
		)
	default:
		return host, nil
	}
}

func (m Manifest) NetworkAllowed(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return false
	}
	for _, allowed := range m.Network {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		switch {
		case allowed == NetworkAnyHost:
			return true
		case strings.HasPrefix(allowed, "*."):
			suffix := allowed[1:]
			if len(host) > len(suffix) && strings.HasSuffix(host, suffix) {
				return true
			}
		case host == allowed:
			return true
		}
	}
	return false
}
