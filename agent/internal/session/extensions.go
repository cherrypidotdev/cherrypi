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
	_ "embed"
	"os"
	"path/filepath"
	"sort"
)

// setTaskURLsExtension is the bundled Pi extension that registers the
// `setTaskUrls` tool. The agent loads it into every Pi child with
// `--extension`; the tool appends a custom entry that Session.handleRecord
// forwards to the agency.
//
//go:embed extension_assets/set_task_urls.ts
var setTaskURLsExtension []byte

// uploadAttachmentExtension registers the `uploadAttachment` tool. It uploads a
// task-workspace file to the agency with the agent's personal access token
// (read from the injected token file) and returns the REST URL, which the model
// can embed in its reply.
//
//go:embed extension_assets/upload_attachment.ts
var uploadAttachmentExtension []byte

// setTaskArtifactsExtension registers the `setTaskArtifacts` tool. It uploads
// file parts and records the task's artifact set as a custom entry, which the
// session forwards to the agency as a `task_artifact` event (A2A artifacts).
//
//go:embed extension_assets/set_task_artifacts.ts
var setTaskArtifactsExtension []byte

// agentToolsExtension registers the `listTasks`, `getTask`,
// `getTaskHistory`, `createTask`, `updateTask`, and `listProjects` tools. They
// call the agency's agent-tool REST surface with the agent's personal access
// token, read from the injected token file on every call.
//
//go:embed extension_assets/agent_tools.ts
var agentToolsExtension []byte

// askUserQuestionExtension registers the `ask_user_question` tool: the
// CherryPi questionnaire. In RPC mode it sends one batch dialog carrying every
// question (up to 10, with recommended flags and notes) and receives every
// answer together, so the inbox/webhook fire once rather than once per
// question. Because it is loaded with `--extension`, it wins over any
// question tool another extension may register.
//
//go:embed extension_assets/ask_user_question.ts
var askUserQuestionExtension []byte

// InstallExtensions writes the bundled Pi extensions into dir and returns
// their paths, for use as `pi --extension <path>` args. It is idempotent: a
// restart overwrites the previous copy, so an upgraded agent ships the current
// extension.
func InstallExtensions(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	paths := make([]string, 0, 5)
	for name, content := range map[string][]byte{
		"set_task_urls.ts":      setTaskURLsExtension,
		"upload_attachment.ts":  uploadAttachmentExtension,
		"set_task_artifacts.ts": setTaskArtifactsExtension,
		"agent_tools.ts":        agentToolsExtension,
		"ask_user_question.ts":  askUserQuestionExtension,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	// A stable order keeps `pi --extension` args deterministic across runs.
	sort.Strings(paths)
	return paths, nil
}
