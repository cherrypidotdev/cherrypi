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
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
)

const (
	MaxBodyBytes = 64 * 1024
	MaxArgs      = 50
	MaxArgKeyLen = 64
)

var argKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeContent(content Content) (Content, Validation) {
	var v Validation
	out := Content{
		Body:                strings.TrimSpace(content.Body),
		TitleTemplate:       strings.TrimSpace(content.TitleTemplate),
		DefaultPriority:     content.DefaultPriority,
		DefaultSize:         strings.TrimSpace(content.DefaultSize),
		DefaultTagIDs:       normalizeStrings(content.DefaultTagIDs),
		DefaultTeamIDs:      normalizeInt64s(content.DefaultTeamIDs),
		DefaultLeadActorIDs: normalizeActorIDs(content.DefaultLeadActorIDs),
	}
	if out.Body == "" {
		v.Errors = append(v.Errors, "the template body must be non-empty")
	}
	if len(out.Body) > MaxBodyBytes {
		v.Errors = append(
			v.Errors,
			fmt.Sprintf("the template body must be at most %d bytes", MaxBodyBytes),
		)
	}
	if out.DefaultPriority < tasks.PriorityLow || out.DefaultPriority > tasks.PriorityCritical {
		v.Errors = append(v.Errors, "default priority is invalid")
	}
	if out.DefaultSize == "" {
		out.DefaultSize = tasks.SizeMedium
	}
	switch out.DefaultSize {
	case tasks.SizeSmall, tasks.SizeMedium, tasks.SizeBig, tasks.SizeEpic:
	default:
		v.Errors = append(v.Errors, "default size is invalid")
	}

	seen := map[string]bool{}
	for _, arg := range content.Args {
		arg.Key = strings.TrimSpace(arg.Key)
		arg.Label = strings.TrimSpace(arg.Label)
		arg.Description = strings.TrimSpace(arg.Description)
		arg.DefaultValue = strings.TrimSpace(arg.DefaultValue)
		if arg.Label == "" {
			arg.Label = arg.Key
		}
		if arg.Type == "" {
			arg.Type = ArgText
		}
		if len(arg.Key) > MaxArgKeyLen || !argKeyRe.MatchString(arg.Key) {
			v.Errors = append(
				v.Errors,
				fmt.Sprintf(
					"argument key %q must be a Jinja identifier (letters, digits, underscore; not starting with a digit)",
					arg.Key,
				),
			)
			continue
		}
		if seen[arg.Key] {
			v.Errors = append(
				v.Errors,
				fmt.Sprintf("argument %q is declared more than once", arg.Key),
			)
			continue
		}
		seen[arg.Key] = true
		switch arg.Type {
		case ArgText, ArgMultiline:
		case ArgNumber:
			if arg.DefaultValue != "" {
				if _, err := strconv.ParseFloat(arg.DefaultValue, 64); err != nil {
					v.Errors = append(
						v.Errors,
						fmt.Sprintf(
							"argument %q has a non-numeric default %q",
							arg.Key,
							arg.DefaultValue,
						),
					)
				}
			}
		case ArgBoolean:
			if arg.DefaultValue != "" {
				if _, err := parseBool(arg.DefaultValue); err != nil {
					v.Errors = append(
						v.Errors,
						fmt.Sprintf(
							"argument %q has a non-boolean default %q",
							arg.Key,
							arg.DefaultValue,
						),
					)
				}
			}
		case ArgSelect:
			options := normalizeOptions(arg.Options)
			arg.Options = options
			if len(options) == 0 {
				v.Errors = append(
					v.Errors,
					fmt.Sprintf("argument %q is a select but has no options", arg.Key),
				)
			}
			if arg.DefaultValue != "" && !hasOption(options, arg.DefaultValue) {
				v.Errors = append(
					v.Errors,
					fmt.Sprintf(
						"argument %q has a default %q that is not one of its options",
						arg.Key,
						arg.DefaultValue,
					),
				)
			}
		default:
			v.Errors = append(
				v.Errors,
				fmt.Sprintf("argument %q has an unknown type %q", arg.Key, arg.Type),
			)
		}
		out.Args = append(out.Args, arg)
	}
	if len(out.Args) > MaxArgs {
		v.Errors = append(
			v.Errors,
			fmt.Sprintf("a template may declare at most %d arguments", MaxArgs),
		)
	}

	bodyVars, unsupported, err := scanVariables(out.Body)
	if err != nil {
		v.Errors = append(v.Errors, "body: "+err.Error())
	}
	if out.TitleTemplate != "" {
		titleVars, titleUnsupported, titleErr := scanVariables(out.TitleTemplate)
		if titleErr != nil {
			v.Errors = append(v.Errors, "title: "+titleErr.Error())
		}
		unsupported = append(unsupported, titleUnsupported...)
		bodyVars = append(bodyVars, titleVars...)
	}
	for _, tag := range unsupported {
		v.Errors = append(v.Errors,
			fmt.Sprintf("tag {%% %s %%} is not supported in task templates", tag))
	}
	used := map[string]bool{}
	for _, name := range bodyVars {
		used[name] = true
		if !seen[name] {
			v.UndeclaredVariables = append(v.UndeclaredVariables, name)
		}
	}
	v.UndeclaredVariables = uniqueSorted(v.UndeclaredVariables)
	if len(v.UndeclaredVariables) > 0 {
		v.Errors = append(v.Errors,
			"declare the referenced variables: "+strings.Join(v.UndeclaredVariables, ", "))
	}
	for _, arg := range out.Args {
		if !used[arg.Key] {
			v.UnusedArguments = append(v.UnusedArguments, arg.Key)
		}
	}
	v.UnusedArguments = uniqueSorted(v.UnusedArguments)
	if len(v.UnusedArguments) > 0 {
		v.Warnings = append(v.Warnings,
			"declared but unused: "+strings.Join(v.UnusedArguments, ", "))
	}

	if out.Body != "" {
		if err := compile(out.Body); err != nil {
			v.Errors = append(v.Errors, "body: "+err.Error())
		}
	}
	if out.TitleTemplate != "" {
		if err := compile(out.TitleTemplate); err != nil {
			v.Errors = append(v.Errors, "title: "+err.Error())
		}
	}
	return out, v
}

func renderSmokeCheck(content Content) (errorsOut, warnings []string) {
	args := map[string]any{}
	for _, arg := range content.Args {
		value, err := defaultArgValue(arg)
		if err != nil {
			errorsOut = append(errorsOut, err.Error())
			continue
		}
		args[arg.Key] = value
	}
	if len(errorsOut) > 0 {
		return errorsOut, nil
	}
	if _, err := renderSource(content.Body, args); err != nil {
		errorsOut = append(errorsOut, "body: "+err.Error())
	}
	if content.TitleTemplate != "" {
		if _, err := renderSource(content.TitleTemplate, args); err != nil {
			errorsOut = append(errorsOut, "title: "+err.Error())
		}
	}
	return errorsOut, nil
}

func normalizeOptions(in []ArgOption) []ArgOption {
	out := make([]ArgOption, 0, len(in))
	seen := map[string]bool{}
	for _, option := range in {
		option.Value = strings.TrimSpace(option.Value)
		option.Label = strings.TrimSpace(option.Label)
		if option.Value == "" || seen[option.Value] {
			continue
		}
		seen[option.Value] = true
		if option.Label == "" {
			option.Label = option.Value
		}
		out = append(out, option)
	}
	return out
}

func hasOption(options []ArgOption, value string) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func normalizeStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeInt64s(in []int64) []int64 {
	out := make([]int64, 0, len(in))
	seen := map[int64]bool{}
	for _, value := range in {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeActorIDs(in []ids.ActorID) []ids.ActorID {
	out := make([]ids.ActorID, 0, len(in))
	seen := map[ids.ActorID]bool{}
	for _, value := range in {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func compile(source string) error {
	_, err := compileTemplate(source)
	return err
}

func compileTemplate(source string) (*exec.Template, error) {
	loader, err := loaders.NewMemoryLoader(map[string]string{"/template": source})
	if err != nil {
		return nil, err
	}
	cfg := config.New()
	cfg.StrictUndefined = true
	return exec.NewTemplate("/template", cfg, loader, gonja.DefaultEnvironment)
}

func renderSource(source string, data map[string]any) (string, error) {
	t, err := compileTemplate(source)
	if err != nil {
		return "", err
	}
	out, err := t.ExecuteToString(exec.NewContext(data))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func renderVersion(t Template, v Version, argsJSON string) (Rendered, error) {
	values, normalizedJSON, err := normalizeArgs(v.Args, argsJSON)
	if err != nil {
		return Rendered{}, fmt.Errorf("%w: %s", ErrInvalidContent, err)
	}
	prompt, err := renderSource(v.Body, values)
	if err != nil {
		return Rendered{}, fmt.Errorf("%w: render body: %s", ErrInvalidContent, err)
	}
	if prompt == "" {
		return Rendered{}, fmt.Errorf(
			"%w: the template rendered an empty prompt", ErrInvalidContent,
		)
	}
	title := ""
	if v.TitleTemplate != "" {
		title, err = renderSource(v.TitleTemplate, values)
		if err != nil {
			return Rendered{}, fmt.Errorf(
				"%w: render title: %s", ErrInvalidContent, err,
			)
		}
	}
	return Rendered{
		TemplateID:   t.ID,
		TemplateName: t.Name,
		Version:      v.Version,
		Prompt:       prompt,
		Title:        title,
		Priority:     v.DefaultPriority,
		Size:         v.DefaultSize,
		TagIDs:       v.DefaultTagIDs,
		TeamIDs:      v.DefaultTeamIDs,
		LeadActorIDs: v.DefaultLeadActorIDs,
		ArgsJSON:     normalizedJSON,
	}, nil
}
