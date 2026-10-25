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

package appcontext

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Dir string

	DatabaseDSN string

	PiBin string

	PiSessionDir string

	PiArgs []string

	WorkspaceRoot string

	Shell string

	PiAgentDir string

	Display string

	XAuthority string

	AgencyURL string

	TunnelTransport string

	AgentUsername string
	AgentPassword string

	AgentToken string

	AgentTokenFixed bool

	AgentTokenFile string

	LogFormat string

	LogLevel string
}

func cherryPiDir() string {
	if dir := os.Getenv("CHERRYPI_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Cannot resolve CHERRYPI_DIR: %v", err)
	}

	return filepath.Join(home, ".cherrypi")
}

func ParseConfig() Config {
	cfg := Config{}

	cfg.Dir = cherryPiDir()
	cfg.WorkspaceRoot = filepath.Join(cfg.Dir, "sessions")

	databaseDSN := os.Getenv("CHERRYPI_DATABASE_DSN")
	if databaseDSN == "" {
		databaseDSN = filepath.Join(cfg.Dir, "cherrypi.sqlite3")
	}
	cfg.DatabaseDSN = databaseDSN

	cfg.PiBin = os.Getenv("PI_BIN")
	cfg.PiSessionDir = os.Getenv("PI_SESSION_DIR")
	cfg.PiArgs = strings.Fields(os.Getenv("PI_ARGS"))
	cfg.Shell = os.Getenv("SHELL_BIN")

	cfg.PiAgentDir = os.Getenv("PI_CODING_AGENT_DIR")
	if cfg.PiAgentDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.PiAgentDir = filepath.Join(home, ".pi", "agent")
		}
	}

	cfg.Display = os.Getenv("DISPLAY")
	cfg.XAuthority = os.Getenv("XAUTHORITY")

	cfg.LogFormat = os.Getenv("CHERRYPI_LOG_FORMAT")
	cfg.LogLevel = os.Getenv("CHERRYPI_LOG_LEVEL")

	cfg.AgencyURL = os.Getenv("CHERRYPI_AGENCY_URL")
	cfg.TunnelTransport = strings.ToLower(
		strings.TrimSpace(os.Getenv("CHERRYPI_AGENT_TUNNEL_TRANSPORT")),
	)
	if auth := os.Getenv("CHERRYPI_AGENT_AUTH"); auth != "" {
		user, pass, ok := strings.Cut(auth, ":")
		if !ok || user == "" || pass == "" {
			log.Fatalf("Invalid CHERRYPI_AGENT_AUTH: want user:pass")
		}
		cfg.AgentUsername = user
		cfg.AgentPassword = pass
	}

	cfg.AgentToken = strings.TrimSpace(os.Getenv("CHERRYPI_AGENT_TOKEN"))
	cfg.AgentTokenFixed = cfg.AgentToken != ""
	cfg.AgentTokenFile = os.Getenv("CHERRYPI_AGENT_TOKEN_FILE")
	if cfg.AgentTokenFile == "" {
		cfg.AgentTokenFile = filepath.Join(cfg.Dir, "agent.token")
	}

	return cfg
}
