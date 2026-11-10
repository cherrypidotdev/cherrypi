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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/plugins"
)

type Options struct {
	BundleDir string
	Scenario  string
	AgencyBin string
	AgencyURL string

	AgencyUsername string
	AgencyPassword string
	WithAgent      bool
	AgentBin       string
	MockLLMBin     string
	Keep           bool
	Verbose        bool
	Stdout         io.Writer
	Stderr         io.Writer
}

func agencyOptions(opts Options, stderr io.Writer) AgencyOptions {
	return AgencyOptions{
		Bin:      opts.AgencyBin,
		URL:      opts.AgencyURL,
		Username: opts.AgencyUsername,
		Password: opts.AgencyPassword,
		Keep:     opts.Keep,
		Verbose:  opts.Verbose,
		Stderr:   stderr,
	}
}

func Run(ctx context.Context, opts Options) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	scenario, err := LoadScenario(opts.Scenario)
	if err != nil {
		return err
	}
	if _, err := plugins.LintBundle(opts.BundleDir); err != nil {
		return err
	}
	bundle, err := plugins.PackBundleDir(opts.BundleDir)
	if err != nil {
		return err
	}

	agency, err := StartAgency(ctx, agencyOptions(opts, stderr))
	if err != nil {
		return err
	}
	defer agency.Stop()
	_, _ = fmt.Fprintf(stdout, "agency %s\n", agency.URL)

	project, err := agency.createProject(scenario.Agency.Project)
	if err != nil {
		return fmt.Errorf("seed project: %w", err)
	}
	for _, tag := range scenario.Agency.Tags {
		if _, err := agency.createTag(project.ID, tag); err != nil {
			return fmt.Errorf("seed tag %s: %w", tag, err)
		}
	}
	agent, err := agency.createAgent(scenario.Agency.Agent)
	if err != nil {
		return fmt.Errorf("seed agent: %w", err)
	}
	if err := agency.addProjectAgent(project.ID, agent.ID); err != nil {
		return fmt.Errorf("add agent to project: %w", err)
	}
	if err := agency.grantProjectMember(project.ID, agent.ActorID); err != nil {
		return fmt.Errorf("grant the agent the member role: %w", err)
	}
	plugin, err := agency.installPlugin(bundle)
	if err != nil {
		return fmt.Errorf("install plugin: %w", err)
	}
	if err := agency.invitePlugin(plugin.ID, project.ID); err != nil {
		return fmt.Errorf("invite plugin: %w", err)
	}
	vars := map[string]string{
		"agency.url":     agency.URL,
		"project.id":     project.ID,
		"project.name":   project.Name,
		"project.slug":   project.Slug,
		"agent.id":       agent.ID,
		"agent.actor_id": agent.ActorID,
		"agent.token":    agent.Token,
		"plugin.id":      plugin.ID,
	}
	for key, value := range scenario.Vars {
		vars[key] = expand(value, vars)
	}
	orgSettings, _ := expandValue(scenario.Settings.Org, vars).(map[string]any)
	projectSettings, _ := expandValue(scenario.Settings.Project, vars).(map[string]any)
	agentSettings, _ := expandValue(scenario.Settings.Agent, vars).(map[string]any)
	userSettings, _ := expandValue(scenario.Settings.User, vars).(map[string]any)

	projectEnabled := true
	if scenario.Project.Enabled != nil {
		projectEnabled = *scenario.Project.Enabled
	}
	allowTaskCreation := true
	if scenario.Project.AllowTaskCreation != nil {
		allowTaskCreation = *scenario.Project.AllowTaskCreation
	}
	maxPriority := 0
	if scenario.Project.MaxPriority != nil {
		maxPriority = *scenario.Project.MaxPriority
	}
	if err := agency.enableProjectPlugin(
		plugin.ID,
		project.ID,
		projectEnabled,
		allowTaskCreation,
		maxPriority,
		projectSettings,
	); err != nil {
		return fmt.Errorf("enable project plugin: %w", err)
	}
	if len(orgSettings) > 0 {
		if err := agency.updateSettings(plugin.ID, "org", "", orgSettings); err != nil {
			return fmt.Errorf("org settings: %w", err)
		}
	}
	if len(agentSettings) > 0 {
		if err := agency.updateSettings(
			plugin.ID,
			"agent",
			agent.ActorID,
			agentSettings,
		); err != nil {
			return fmt.Errorf("agent settings: %w", err)
		}
		if err := agency.enableSettings(plugin.ID, "agent", agent.ActorID); err != nil {
			return fmt.Errorf("enable agent settings: %w", err)
		}
	}
	if len(userSettings) > 0 {
		if err := agency.updateSettings(
			plugin.ID,
			"user",
			agency.actorID,
			userSettings,
		); err != nil {
			return fmt.Errorf("user settings: %w", err)
		}
		if err := agency.enableSettings(plugin.ID, "user", agency.actorID); err != nil {
			return fmt.Errorf("enable user settings: %w", err)
		}
	}
	_, _ = fmt.Fprintf(
		stdout,
		"seeded project %s, agent %s, plugin %s\n",
		project.Name, agent.ID, plugin.ID,
	)

	var agentProcess *AgentProcess
	if opts.WithAgent {
		agentProcess, err = StartAgent(ctx, AgentOptions{
			Bin:        opts.AgentBin,
			MockLLMBin: opts.MockLLMBin,
			Tool:       firstToolName(bundle),
			Args:       firstToolArgs(scenario),
			Text:       "done",
			Keep:       opts.Keep,
			Verbose:    opts.Verbose,
			Stderr:     stderr,
		}, agency, agent)
		if err != nil {
			return err
		}
		defer agentProcess.Stop()
		_, _ = fmt.Fprintf(stdout, "agent online\n")
	}

	lastTask := ""
	for i, step := range scenario.Steps {
		name := step.Name
		if name == "" {
			name = fmt.Sprintf("step %d", i+1)
		}
		before, err := agency.listTasks(project.ID)
		if err != nil {
			return fmt.Errorf("%s: list tasks: %w", name, err)
		}
		created, nextTask, err := runStep(
			ctx, agency, scenario, step, plugin, agent, project, vars, lastTask, before, stdout,
		)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := assertTasks(step.Expect.Tasks, created, project, agent); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if nextTask != "" {
			lastTask = nextTask
		}
		_, _ = fmt.Fprintf(stdout, "ok %s\n", name)
	}
	_, _ = fmt.Fprintf(stdout, "passed %d steps\n", len(scenario.Steps))
	return nil
}

func runStep(
	ctx context.Context,
	agency *Agency,
	scenario Scenario,
	step Step,
	plugin Plugin,
	agent Agent,
	project Project,
	vars map[string]string,
	lastTask string,
	before []Task,
	stdout io.Writer,
) ([]Task, string, error) {
	switch {
	case step.HTTP != nil:
		body, status, err := callPluginHTTP(agency, plugin, step.HTTP, vars)
		if err != nil {
			return nil, "", err
		}
		if step.Expect.Status != 0 && status != step.Expect.Status {
			return nil, "", fmt.Errorf(
				"status %d, want %d (body %s)",
				status,
				step.Expect.Status,
				body,
			)
		}
		if step.Expect.TextContains != "" &&
			!strings.Contains(string(body), expand(step.Expect.TextContains, vars)) {
			return nil, "", fmt.Errorf(
				"response %q does not contain %q",
				body,
				step.Expect.TextContains,
			)
		}
		var decoded map[string]any
		if len(step.Expect.JSON) > 0 {
			if err := json.Unmarshal(body, &decoded); err != nil {
				return nil, "", fmt.Errorf("response is not JSON: %w (%s)", err, body)
			}
			if err := matchJSON(expandValue(step.Expect.JSON, vars), decoded); err != nil {
				return nil, "", err
			}
		}
		after, err := agency.listTasks(project.ID)
		if err != nil {
			return nil, "", err
		}
		created := diffTasks(after, before)
		nextTask := ""
		if decoded != nil {
			nextTask = scalarString(decoded["task_id"])
		}
		if nextTask == "" && len(created) > 0 {
			nextTask = created[0].ID
		}
		return created, nextTask, nil

	case step.Tool != nil:
		taskID := expand(step.Tool.Task, vars)
		if taskID == "" {
			taskID = lastTask
		}
		if taskID == "" {
			return nil, "", fmt.Errorf("no task to call the tool on")
		}
		text, err := callPluginTool(
			agency,
			agent.Token,
			taskID,
			step.Tool.Name,
			expandValue(step.Tool.Args, vars),
		)
		if err != nil {
			return nil, "", err
		}
		if step.Expect.TextContains != "" &&
			!strings.Contains(text, expand(step.Expect.TextContains, vars)) {
			return nil, "", fmt.Errorf(
				"tool result %q does not contain %q",
				text,
				step.Expect.TextContains,
			)
		}
		_, _ = fmt.Fprintf(stdout, "  tool %s -> %s\n", step.Tool.Name, text)
		return nil, taskID, nil

	case step.Wait != nil:
		taskID := expand(step.Wait.Task, vars)
		if taskID == "" {
			taskID = lastTask
		}
		if taskID == "" {
			return nil, "", fmt.Errorf("no task to wait for")
		}
		timeout := 120 * time.Second
		if step.Wait.Timeout != "" {
			parsed, err := time.ParseDuration(expand(step.Wait.Timeout, vars))
			if err != nil {
				return nil, "", fmt.Errorf("timeout: %w", err)
			}
			timeout = parsed
		}
		want := normalizeState(expand(step.Wait.State, vars))
		deadline := time.Now().Add(timeout)
		for {
			task, err := agency.getTask(taskID)
			if err != nil {
				return nil, "", err
			}
			if want == "" || task.State == want {
				return []Task{task}, taskID, nil
			}
			if time.Now().After(deadline) {
				return nil, "", fmt.Errorf(
					"task %s is %s after %s, want %s",
					taskID,
					task.State,
					timeout,
					want,
				)
			}
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(300 * time.Millisecond):
			}
		}

	case step.Task != nil:
		projectID := expand(step.Task.Project, vars)
		switch projectID {
		case "", project.ID, project.Name, project.Slug:
			projectID = project.ID
		}
		assignee := expand(step.Task.Assignee, vars)
		switch assignee {
		case "@agent", "agent":
			if agent.ActorID == "" {
				return nil, "", fmt.Errorf(
					"task.assignee %q needs an agency.agent in the scenario",
					step.Task.Assignee,
				)
			}
			assignee = agent.ActorID
		}
		task, err := agency.seedTask(
			projectID,
			expand(step.Task.Name, vars),
			expand(step.Task.Prompt, vars),
			expand(step.Task.State, vars),
			assignee,
		)
		if err != nil {
			return nil, "", err
		}
		_, _ = fmt.Fprintf(stdout, "  task %s (%s)\n", task.ID, task.State)
		return []Task{task}, task.ID, nil

	case step.Event != nil:
		taskID := expand(step.Event.Task, vars)
		if taskID == "" || taskID == "last" {
			taskID = lastTask
		}
		if taskID == "" {
			return nil, "", fmt.Errorf("no task to dispatch the event for")
		}
		name := expand(step.Event.Name, vars)
		payload, _ := expandValue(step.Event.Payload, vars).(map[string]any)
		delivered, err := agency.dispatchEvent(name, taskID, payload)
		if err != nil {
			return nil, "", err
		}
		if delivered == 0 {
			return nil, "", fmt.Errorf(
				"no enabled plugin subscribes to %q (check the bundle's manifest events)",
				name,
			)
		}
		_, _ = fmt.Fprintf(stdout, "  event %s -> %d plugin(s)\n", name, delivered)
		return nil, taskID, nil

	case step.Tick != nil:
		if err := agency.invokeTick(plugin.ID, step.Tick.NowMs); err != nil {
			return nil, "", err
		}
		after, err := agency.listTasks(project.ID)
		if err != nil {
			return nil, "", err
		}
		created := diffTasks(after, before)
		nextTask := ""
		if len(created) > 0 {
			nextTask = created[0].ID
		}
		_, _ = fmt.Fprintf(stdout, "  tick -> %d task(s)\n", len(created))
		return created, nextTask, nil

	case step.Sleep != "":
		delay, err := time.ParseDuration(expand(step.Sleep, vars))
		if err != nil {
			return nil, "", fmt.Errorf("sleep: %w", err)
		}
		time.Sleep(delay)
		return nil, lastTask, nil

	default:
		return nil, "", fmt.Errorf("step has no action")
	}
}

func callPluginHTTP(
	agency *Agency,
	plugin Plugin,
	step *HTTPStep,
	vars map[string]string,
) ([]byte, int, error) {
	method := strings.ToUpper(step.Method)
	if method == "" {
		method = http.MethodPost
	}
	target := expand(step.URL, vars)
	if target == "" {
		target = agency.URL + "/plugins/" + plugin.ID + expand(step.Path, vars)
	}
	var body io.Reader
	if len(step.JSON) > 0 {
		raw, err := json.Marshal(expandValue(step.JSON, vars))
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(raw)
	} else if step.Body != "" {
		body = strings.NewReader(expand(step.Body, vars))
	}
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range step.Headers {
		req.Header.Set(key, expand(value, vars))
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return payload, resp.StatusCode, nil
}

func callPluginTool(
	agency *Agency,
	token, taskID, tool string,
	args any,
) (string, error) {
	raw, err := json.Marshal(map[string]any{
		"task_id": taskID,
		"tool":    tool,
		"args":    args,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		agency.URL+"/api/rest/agent-tools/plugin-tools",
		bytes.NewReader(raw),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("plugin tool: %d: %s", resp.StatusCode, body)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.Text, nil
}

func diffTasks(after, before []Task) []Task {
	seen := make(map[string]bool, len(before))
	for _, task := range before {
		seen[task.ID] = true
	}
	var out []Task
	for _, task := range after {
		if !seen[task.ID] {
			out = append(out, task)
		}
	}
	return out
}

func assertTasks(expectations []TaskExpect, tasks []Task, project Project, agent Agent) error {
	for _, expected := range expectations {
		matched := false
		for _, task := range tasks {
			if taskMatches(expected, task, project, agent) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf(
				"no task matches %s; observed %s",
				describeTaskExpect(expected),
				describeTasks(tasks),
			)
		}
	}
	return nil
}

func taskMatches(expected TaskExpect, task Task, project Project, agent Agent) bool {
	if expected.Project != "" &&
		expected.Project != project.Name &&
		expected.Project != project.ID &&
		expected.Project != project.Slug {
		return false
	}
	if expected.State != "" && normalizeState(expected.State) != task.State {
		return false
	}
	switch expected.Assignee {
	case "":
	case "@agent", "agent":
		if task.Assignee != agent.ActorID {
			return false
		}
	default:
		if task.Assignee != expected.Assignee {
			return false
		}
	}
	for _, tag := range expected.Tags {
		found := false
		for _, have := range task.Tags {
			if have == tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if expected.PromptContains != "" && !strings.Contains(task.Prompt, expected.PromptContains) {
		return false
	}
	if expected.NameContains != "" && !strings.Contains(task.Name, expected.NameContains) {
		return false
	}
	return true
}

func describeTaskExpect(expected TaskExpect) string {
	raw, _ := json.Marshal(expected)
	return string(raw)
}

func describeTasks(tasks []Task) string {
	raw, _ := json.Marshal(tasks)
	return string(raw)
}

func matchJSON(expected, actual any) error {
	expected = normalizeJSON(expected)
	actual = normalizeJSON(actual)
	switch want := expected.(type) {
	case map[string]any:
		have, ok := actual.(map[string]any)
		if !ok {
			return fmt.Errorf("response is not an object: %v", actual)
		}
		for key, value := range want {
			if err := matchJSON(value, have[key]); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
		return nil
	case []any:
		have, ok := actual.([]any)
		if !ok || len(have) != len(want) {
			return fmt.Errorf("want %v, got %v", want, actual)
		}
		for i := range want {
			if err := matchJSON(want[i], have[i]); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		return nil
	case string:
		if want == "@nonempty" {
			text, ok := actual.(string)
			if !ok || text == "" {
				return fmt.Errorf("want a non-empty string, got %v", actual)
			}
			return nil
		}
		if want != actual {
			return fmt.Errorf("want %v, got %v", want, actual)
		}
		return nil
	default:
		if !reflect.DeepEqual(want, actual) {
			return fmt.Errorf("want %v, got %v", want, actual)
		}
		return nil
	}
}

func normalizeJSON(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out any
	if json.Unmarshal(raw, &out) != nil {
		return value
	}
	return out
}

func firstToolName(bundle []byte) string {
	loaded, err := plugins.ReadBundle(bundle)
	if err != nil || len(loaded.Manifest.Tools) == 0 {
		return ""
	}
	return loaded.Manifest.Tools[0].Name
}

func firstToolArgs(scenario Scenario) string {
	if scenario.Agency.Agent.ToolArgs != nil {
		if raw, err := json.Marshal(scenario.Agency.Agent.ToolArgs); err == nil {
			return string(raw)
		}
	}
	for _, step := range scenario.Steps {
		if step.Tool != nil && step.Tool.Args != nil {
			if raw, err := json.Marshal(step.Tool.Args); err == nil {
				return string(raw)
			}
		}
	}
	return "{}"
}
