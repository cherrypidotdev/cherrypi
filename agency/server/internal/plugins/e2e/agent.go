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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type AgentOptions struct {
	Bin        string
	MockLLMBin string
	PiBin      string
	Tool       string
	Args       string
	Text       string
	Keep       bool
	Verbose    bool
	Stderr     io.Writer
}

type AgentProcess struct {
	cmd  *exec.Cmd
	mock *exec.Cmd
	dir  string
	keep bool
	log  *bytes.Buffer
}

type mockLLMReady struct {
	AgentDir string `json:"agentDir"`
	BaseURL  string `json:"baseUrl"`
}

func StartAgent(
	ctx context.Context,
	opts AgentOptions,
	agency *Agency,
	agent Agent,
) (*AgentProcess, error) {
	agentBin := resolveBin(opts.Bin, "CHERRYPI_AGENT_BIN", "cherrypi-agent")
	if agentBin == "" {
		return nil, fmt.Errorf(
			"agent binary not found: pass --agent-bin, set CHERRYPI_AGENT_BIN, " +
				"or run inside the plugin-ci image",
		)
	}
	mockBin := resolveBin(opts.MockLLMBin, "CHERRYPI_MOCKLLM_BIN", "cherrypi-mockllm")
	if mockBin == "" {
		return nil, fmt.Errorf(
			"mock LLM binary not found: pass --mockllm-bin, set CHERRYPI_MOCKLLM_BIN, " +
				"or run inside the plugin-ci image",
		)
	}
	piBin := resolveBin(opts.PiBin, "PI_BIN", "pi")
	if piBin == "" {
		return nil, fmt.Errorf("pi not found: set PI_BIN or put pi on PATH")
	}

	dir, err := os.MkdirTemp("", "cherrypi-plugin-agent-")
	if err != nil {
		return nil, err
	}
	agentDir := dir + "/agent"
	logBuf := &bytes.Buffer{}
	stdio := io.Writer(logBuf)
	if opts.Verbose {
		stdio = opts.Stderr
	}

	mockArgs := []string{"--agent-dir", agentDir, "--text", opts.Text}
	if opts.Tool != "" {
		mockArgs = append(mockArgs, "--tool", opts.Tool, "--args", opts.Args)
	}
	mock := exec.Command(mockBin, mockArgs...)
	mock.Stderr = stdio
	mockOut, err := mock.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := mock.Start(); err != nil {
		return nil, fmt.Errorf("start mockllm: %w", err)
	}
	ready, err := readReady(mockOut)
	if err != nil {
		_ = mock.Process.Kill()
		return nil, err
	}

	cmd := exec.Command(agentBin, "--agency-url", agency.URL)
	cmd.Env = append(cleanEnv(),
		"CHERRYPI_AGENCY_URL="+agency.URL,
		"CHERRYPI_AGENT_TOKEN="+agent.Token,
		"CHERRYPI_DIR="+dir+"/cherrypi",
		"PI_BIN="+piBin,
		"PI_SESSION_DIR="+dir+"/tasks",
		"PI_ARGS=--model test/deterministic",
		"PI_CODING_AGENT_DIR="+ready.AgentDir,
		"PI_OFFLINE=1",
		"PI_TELEMETRY=0",
		"PI_SKIP_VERSION_CHECK=1",
	)
	cmd.Stdout = stdio
	cmd.Stderr = stdio
	if err := cmd.Start(); err != nil {
		_ = mock.Process.Kill()
		return nil, fmt.Errorf("start agent: %w", err)
	}
	process := &AgentProcess{cmd: cmd, mock: mock, dir: dir, keep: opts.Keep, log: logBuf}
	if err := waitAgentOnline(ctx, agency, agent.ID, 60*time.Second); err != nil {
		process.Stop()
		return nil, fmt.Errorf("%w\n%s", err, logBuf.String())
	}
	return process, nil
}

func (a *AgentProcess) Stop() {
	if a == nil {
		return
	}
	for _, cmd := range []*exec.Cmd{a.cmd, a.mock} {
		if cmd == nil || cmd.Process == nil {
			continue
		}
		_ = cmd.Process.Signal(os.Interrupt)
	}
	time.Sleep(300 * time.Millisecond)
	for _, cmd := range []*exec.Cmd{a.cmd, a.mock} {
		if cmd == nil || cmd.Process == nil {
			continue
		}
		_ = cmd.Process.Kill()
	}
	if a.cmd != nil {
		_ = a.cmd.Wait()
	}
	if a.mock != nil {
		_ = a.mock.Wait()
	}
	if !a.keep {
		_ = os.RemoveAll(a.dir)
	}
}

func readReady(reader io.Reader) (mockLLMReady, error) {
	type result struct {
		ready mockLLMReady
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		if !scanner.Scan() {
			ch <- result{err: fmt.Errorf("mockllm exited before reporting readiness")}
			return
		}
		var ready mockLLMReady
		if err := json.Unmarshal(scanner.Bytes(), &ready); err != nil {
			ch <- result{err: fmt.Errorf("mockllm readiness: %w", err)}
			return
		}
		ch <- result{ready: ready}
	}()
	select {
	case res := <-ch:
		return res.ready, res.err
	case <-time.After(15 * time.Second):
		return mockLLMReady{}, fmt.Errorf("mockllm did not report readiness")
	}
}

func waitAgentOnline(
	ctx context.Context,
	agency *Agency,
	agentID string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := agency.connect("AgentService", "ListAgents", map[string]any{})
		if err == nil {
			rows, _ := out["agents"].([]any)
			for _, row := range rows {
				if tbl, ok := row.(map[string]any); ok {
					if scalarString(tbl["id"]) == agentID &&
						normalizeAgentStatus(scalarString(tbl["status"])) == "online" {
						return nil
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("agent %s did not come online", agentID)
}

func resolveBin(explicit, envVar, name string) string {
	if explicit != "" {
		return explicit
	}
	if value := strings.TrimSpace(os.Getenv(envVar)); value != "" {
		return value
	}
	if found, err := exec.LookPath(name); err == nil {
		return found
	}
	return ""
}
