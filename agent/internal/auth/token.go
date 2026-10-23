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

package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// TokenPrefix is the displayable prefix of an agent personal access token.
const TokenPrefix = "pat_"

// TokenFile returns the conventional agent token path under dir.
func TokenFile(dir string) string {
	return filepath.Join(dir, "agent.token")
}

// LoadToken returns the agent's personal access token: the CHERRYPI_AGENT_TOKEN
// value wins, then the token file. It returns "" when neither holds a token.
// Callers must never log the result.
func LoadToken(env, file string) string {
	if token := strings.TrimSpace(env); strings.HasPrefix(token, TokenPrefix) {
		return token
	}
	if file == "" {
		return ""
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	token := strings.TrimSpace(string(data))
	if !strings.HasPrefix(token, TokenPrefix) {
		return ""
	}
	return token
}

// SaveToken writes the token to path with owner-only permissions, replacing any
// previous value atomically so a running Pi child always reads a whole token.
func SaveToken(path, token string) error {
	if !strings.HasPrefix(token, TokenPrefix) {
		return errors.New("not an agent token")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
