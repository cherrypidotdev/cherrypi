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

//go:embed extension_assets/set_task_urls.ts
var setTaskURLsExtension []byte

//go:embed extension_assets/upload_attachment.ts
var uploadAttachmentExtension []byte

//go:embed extension_assets/set_task_artifacts.ts
var setTaskArtifactsExtension []byte

//go:embed extension_assets/agent_tools.ts
var agentToolsExtension []byte

//go:embed extension_assets/ask_user_question.ts
var askUserQuestionExtension []byte

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

	sort.Strings(paths)
	return paths, nil
}
