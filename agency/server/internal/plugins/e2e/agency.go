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
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Agency struct {
	URL      string
	dir      string
	keep     bool
	cmd      *exec.Cmd
	client   *http.Client
	csrf     string
	actorID  string
	username string
	password string
	log      *bytes.Buffer
	verbose  bool
	stderr   io.Writer
}

type AgencyOptions struct {
	Bin string

	URL      string
	Username string
	Password string
	Keep     bool
	Verbose  bool
	Stderr   io.Writer
}

const defaultAdminUsername = "admin"

func StartAgency(ctx context.Context, opts AgencyOptions) (*Agency, error) {
	if opts.URL != "" {
		a := &Agency{
			URL:      strings.TrimRight(opts.URL, "/"),
			username: adminUsername(opts.Username),
			password: firstNonEmpty(opts.Password, os.Getenv("CHERRYPI_ADMIN_PASSWORD")),
			stderr:   opts.Stderr,
		}
		if a.password == "" {
			return nil, fmt.Errorf(
				"--agency-url %s needs the agency's admin credentials: pass "+
					"--agency-password (and --agency-username when it is not %q), "+
					"or set CHERRYPI_ADMIN_PASSWORD",
				a.URL, defaultAdminUsername,
			)
		}
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		a.client = &http.Client{Jar: jar, Timeout: 30 * time.Second}
		if err := a.login(); err != nil {
			return nil, fmt.Errorf(
				"%w\ncheck --agency-username/--agency-password (or "+
					"CHERRYPI_ADMIN_USERNAME/CHERRYPI_ADMIN_PASSWORD) for %s",
				err, a.URL,
			)
		}
		return a, nil
	}
	bin := opts.Bin
	if bin == "" {
		bin = os.Getenv("CHERRYPI_AGENCY_BIN")
	}
	if bin == "" {
		found, err := exec.LookPath("cherrypi-agency")
		if err != nil {
			return nil, fmt.Errorf(
				"agency binary not found: pass --agency-bin, set CHERRYPI_AGENCY_BIN, " +
					"or run inside the agency image",
			)
		}
		bin = found
	}
	dir, err := os.MkdirTemp("", "cherrypi-plugin-e2e-")
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	password, err := randomHex()
	if err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	logBuf := &bytes.Buffer{}
	cmd := exec.Command(bin)
	cmd.Env = append(cleanEnv(),
		"CHERRYPI_AGENCY_DIR="+dir,
		"CHERRYPI_AGENCY_PORT="+strconv.Itoa(port),
		"CHERRYPI_ADMIN_USERNAME="+defaultAdminUsername,
		"CHERRYPI_ADMIN_PASSWORD="+password,
		"CHERRYPI_PUBLIC_URL="+baseURL,
		"CHERRYPI_PLUGIN_ALLOW_PRIVATE_NETWORK=1",

		"CHERRYPI_PLUGIN_DISABLE_TICKS=1",
	)
	if opts.Verbose {
		cmd.Stdout = opts.Stderr
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stdout = logBuf
		cmd.Stderr = logBuf
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	a := &Agency{
		URL:      baseURL,
		dir:      dir,
		keep:     opts.Keep,
		cmd:      cmd,
		username: defaultAdminUsername,
		password: password,
		log:      logBuf,
		verbose:  opts.Verbose,
		stderr:   opts.Stderr,
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	a.client = &http.Client{Jar: jar, Timeout: 30 * time.Second}
	if err := a.waitHealthy(30 * time.Second); err != nil {
		a.Stop()
		return nil, fmt.Errorf("%w\n%s", err, logBuf.String())
	}
	if err := a.login(); err != nil {
		a.Stop()
		return nil, err
	}
	if opts.Keep {
		_, _ = fmt.Fprintf(opts.Stderr, "agency data: %s\n", dir)
	}
	return a, nil
}

func adminUsername(explicit string) string {
	return strings.TrimSpace(firstNonEmpty(
		explicit,
		os.Getenv("CHERRYPI_ADMIN_USERNAME"),
		defaultAdminUsername,
	))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func cleanEnv() []string {
	drop := map[string]bool{
		"CHERRYPI_AGENCY_DATABASE_DSN": true,
		"CHERRYPI_AGENCY_DIR":          true,
		"CHERRYPI_AGENCY_PORT":         true,
		"CHERRYPI_FRONTEND_PROXY_URL":  true,
		"CHERRYPI_DOCS_PROXY_URL":      true,
		"CHERRYPI_S3_ENDPOINT":         true,
		"CHERRYPI_S3_BUCKET":           true,
		"CHERRYPI_ADMIN_USERNAME":      true,
		"CHERRYPI_ADMIN_PASSWORD":      true,
		"CHERRYPI_PUBLIC_URL":          true,
	}
	var out []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !drop[key] {
			out = append(out, entry)
		}
	}
	return out
}

func (a *Agency) Stop() {
	if a == nil || a.cmd == nil {
		return
	}
	_ = a.cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_ = a.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = a.cmd.Process.Kill()
		<-done
	}
	if !a.keep {
		_ = os.RemoveAll(a.dir)
	}
}

func (a *Agency) waitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(a.URL + "/api/rest/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("agency at %s did not become healthy", a.URL)
}

func (a *Agency) login() error {
	body, err := a.postJSON("/api/rest/auth/login", map[string]any{
		"identifier": a.username,
		"password":   a.password,
	})
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	if token, ok := body["csrf_token"].(string); ok {
		a.csrf = token
	}
	if actor, ok := body["actor"].(map[string]any); ok {
		a.actorID = scalarString(actor["id"])
	}
	if a.csrf == "" || a.actorID == "" {
		return fmt.Errorf("login returned no csrf token or actor")
	}
	return nil
}

func (a *Agency) postJSON(path string, payload any) (map[string]any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, a.URL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.csrf != "" {
		req.Header.Set("X-CSRF-Token", a.csrf)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST %s: %d: %s", path, resp.StatusCode, body)
	}
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("POST %s: %d: not JSON: %s", path, resp.StatusCode, snippet(body))
	}
	return out, nil
}

func snippet(body []byte) string {
	const limit = 120
	text := strings.TrimSpace(string(body))
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "…"
}

func (a *Agency) connect(service, method string, payload any) (map[string]any, error) {
	path := "/api/grpc/agency.v1." + service + "/" + method
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, a.URL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("X-CSRF-Token", a.csrf)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s/%s: %d: %s", service, method, resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%s/%s: %w", service, method, err)
	}
	return out, nil
}

type Project struct {
	ID   string
	Name string
	Slug string
}

func (a *Agency) createProject(seed ProjectSeed) (Project, error) {
	out, err := a.connect("ProjectService", "CreateProject", map[string]any{
		"name":   seed.Name,
		"slug":   seed.Slug,
		"prefix": seed.Prefix,
	})
	if err != nil {
		return Project{}, err
	}
	project, _ := out["project"].(map[string]any)
	return Project{
		ID:   scalarString(project["id"]),
		Name: scalarString(project["name"]),
		Slug: scalarString(project["slug"]),
	}, nil
}

func (a *Agency) createTag(projectID, name string) (string, error) {
	out, err := a.connect("TagService", "CreateTag", map[string]any{
		"projectId": projectID,
		"name":      name,
	})
	if err != nil {
		return "", err
	}
	tag, _ := out["tag"].(map[string]any)
	return scalarString(tag["id"]), nil
}

type Agent struct {
	ID      string
	ActorID string
	Token   string
}

func (a *Agency) createAgent(seed AgentSeed) (Agent, error) {
	username := seed.Username
	if username == "" {
		username = "plugin-e2e-agent"
	}
	out, err := a.connect("AgentService", "CreateAgent", map[string]any{
		"username":    username,
		"displayName": seed.Name,
	})
	if err != nil {
		return Agent{}, err
	}
	agent, _ := out["agent"].(map[string]any)
	return Agent{
		ID:      scalarString(agent["id"]),
		ActorID: scalarString(agent["actorId"]),
		Token:   scalarString(out["token"]),
	}, nil
}

func (a *Agency) addProjectAgent(projectID, agentID string) error {
	_, err := a.connect("ProjectService", "AddProjectAgent", map[string]any{
		"projectId": projectID,
		"agentId":   agentID,
	})
	return err
}

func (a *Agency) grantProjectMember(projectID, actorID string) error {
	path := fmt.Sprintf("/api/rest/projects/%s/members", projectID)
	_, err := a.postJSON(path, map[string]any{
		"actor_id": actorID,
		"role":     "member",
	})
	return err
}

type Plugin struct {
	ID      string
	ActorID string
}

func (a *Agency) installPlugin(bundle []byte) (Plugin, error) {
	out, err := a.connect("PluginService", "InstallPlugin", map[string]any{
		"bundle": base64.StdEncoding.EncodeToString(bundle),
	})
	if err != nil {
		return Plugin{}, err
	}
	plugin, _ := out["plugin"].(map[string]any)
	return Plugin{
		ID:      scalarString(plugin["id"]),
		ActorID: scalarString(plugin["actorId"]),
	}, nil
}

func (a *Agency) invitePlugin(pluginID, projectID string) error {
	_, err := a.connect("PluginService", "InvitePluginToProject", map[string]any{
		"pluginId":  pluginID,
		"projectId": projectID,
		"role":      "member",
	})
	return err
}

func (a *Agency) enableProjectPlugin(
	pluginID, projectID string,
	enabled, allowTaskCreation bool,
	maxPriority int,
	settings map[string]any,
) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = a.connect("PluginService", "UpdateProjectPlugin", map[string]any{
		"pluginId":          pluginID,
		"projectId":         projectID,
		"enabled":           enabled,
		"allowTaskCreation": allowTaskCreation,
		"maxPriority":       maxPriority,
		"settingsJson":      string(raw),
	})
	return err
}

func (a *Agency) updateSettings(pluginID, scope, target string, settings map[string]any) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = a.connect("PluginService", "UpdatePluginSettings", map[string]any{
		"pluginId":     pluginID,
		"scope":        scope,
		"targetId":     target,
		"settingsJson": string(raw),
	})
	return err
}

func (a *Agency) enableSettings(pluginID, scope, target string) error {
	_, err := a.connect("PluginService", "SetPluginSettingsEnabled", map[string]any{
		"pluginId": pluginID,
		"scope":    scope,
		"targetId": target,
		"enabled":  true,
	})
	return err
}

type Task struct {
	ID        string
	ProjectID string
	Name      string
	Prompt    string
	State     string
	Assignee  string
	Tags      []string
}

func (a *Agency) listTasks(projectID string) ([]Task, error) {
	out, err := a.connect("TaskService", "ListTasks", map[string]any{
		"projectId": projectID,
		"limit":     500,
	})
	if err != nil {
		return nil, err
	}
	rows, _ := out["tasks"].([]any)
	tasksOut := make([]Task, 0, len(rows))
	for _, row := range rows {
		if tbl, ok := row.(map[string]any); ok {
			tasksOut = append(tasksOut, toTask(tbl))
		}
	}
	return tasksOut, nil
}

func (a *Agency) getTask(id string) (Task, error) {
	out, err := a.connect("TaskService", "GetTask", map[string]any{"id": id})
	if err != nil {
		return Task{}, err
	}
	tbl, _ := out["task"].(map[string]any)
	return toTask(tbl), nil
}

func (a *Agency) seedTask(
	projectID, name, prompt, state, assignee string,
) (Task, error) {
	payload := map[string]any{
		"project_id": projectID,
		"name":       name,
		"prompt":     prompt,
		"state":      state,
	}
	if assignee != "" {
		payload["assignee_actor_id"] = assignee
	}
	out, err := a.postJSON("/api/dev/plugins/tasks", payload)
	if err != nil {
		return Task{}, fmt.Errorf("%w (a task: step needs an agency built with -tags dev)", err)
	}
	id := scalarString(out["task_id"])
	if id == "" {
		return Task{}, fmt.Errorf("the seed endpoint returned no task id")
	}
	return a.getTask(id)
}

func (a *Agency) dispatchEvent(name, taskID string, payload map[string]any) (int, error) {
	out, err := a.postJSON("/api/dev/plugins/events", map[string]any{
		"name":    name,
		"task_id": taskID,
		"payload": payload,
	})
	if err != nil {
		return 0, fmt.Errorf("%w (an event: step needs an agency built with -tags dev)", err)
	}
	return scalarInt(out["delivered"]), nil
}

func (a *Agency) invokeTick(pluginID string, nowMs int64) error {
	payload := map[string]any{"plugin_id": pluginID}
	if nowMs != 0 {
		payload["now_ms"] = nowMs
	}
	_, err := a.postJSON("/api/dev/plugins/tick", payload)
	if err != nil {
		return fmt.Errorf("%w (a tick: step needs an agency built with -tags dev)", err)
	}
	return nil
}

func toTask(tbl map[string]any) Task {
	task := Task{
		ID:        scalarString(tbl["id"]),
		ProjectID: scalarString(tbl["projectId"]),
		Name:      scalarString(tbl["name"]),
		Prompt:    scalarString(tbl["prompt"]),
		State:     normalizeState(scalarString(tbl["state"])),
		Assignee:  scalarString(tbl["assigneeActorId"]),
	}
	if tags, ok := tbl["tags"].([]any); ok {
		for _, tag := range tags {
			if m, ok := tag.(map[string]any); ok {
				task.Tags = append(task.Tags, scalarString(m["name"]))
			}
		}
	}
	return task
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func randomHex() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func scalarString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func scalarInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func normalizeState(value string) string {
	return strings.ToLower(strings.TrimPrefix(value, "TASK_STATE_"))
}

func normalizeAgentStatus(value string) string {
	return strings.ToLower(strings.TrimPrefix(value, "AGENT_STATUS_"))
}
