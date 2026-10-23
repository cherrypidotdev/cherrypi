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

// Config is the agent worker's configuration. The agent has no HTTP listener:
// it serves its API over the tunnel, so there is no port or local credential
// to configure.
type Config struct {
	// Dir is CherryPi's data directory: the SQLite database and per-session
	// workspaces live under it. Parsed from CHERRYPI_DIR. Default: ~/.cherrypi.
	Dir string

	// DatabaseDSN is the Data Source Name for the SQLite database.
	// Parsed from CHERRYPI_DATABASE_DSN. Default: $CHERRYPI_DIR/cherrypi.sqlite3.
	DatabaseDSN string

	// PiBin is the Pi executable. Default: "pi" (resolved from PATH).
	PiBin string

	// PiSessionDir optionally overrides Pi's session storage (`--session-dir`).
	PiSessionDir string

	// PiArgs are extra CLI args passed to `pi` after `--mode rpc --session-id`.
	// Parsed from space-separated PI_ARGS (used by tests to pin a model).
	PiArgs []string

	// WorkspaceRoot is the parent of per-session workspaces.
	// Default: $CHERRYPI_DIR/sessions.
	WorkspaceRoot string

	// Shell is the shell executable used for root terminals.
	// Parsed from SHELL_BIN. Default: $SHELL, then /bin/bash.
	Shell string

	// PiAgentDir is Pi's agent directory (memory, settings.json, …).
	// Parsed from PI_CODING_AGENT_DIR. Default: ~/.pi/agent.
	PiAgentDir string

	// Display is the X display the desktop bridge attaches to. Parsed from
	// DISPLAY; the --display flag overrides it. Empty disables the desktop
	// (CherryPi never boots a session).
	Display string

	// XAuthority is the X authority file for Display. Parsed from XAUTHORITY;
	// the --xauthority flag overrides it.
	XAuthority string

	// AgencyURL is the agency to tunnel to (CHERRYPI_AGENCY_URL or the
	// --agency-url flag).
	AgencyURL string

	// AgentUsername/AgentPassword authenticate the tunnel
	// (CHERRYPI_AGENT_AUTH or the --auth flag, `user:pass`). They are the
	// legacy fallback: the agent exchanges them for a personal access token on
	// first boot.
	AgentUsername string
	AgentPassword string

	// AgentToken is the agent's personal access token (CHERRYPI_AGENT_TOKEN).
	// When set it wins over the cached token file.
	AgentToken string
	// AgentTokenFixed is true when AgentToken came from the environment. A
	// rotated token file then must not override the explicit token.
	AgentTokenFixed bool

	// AgentTokenFile caches the agent's personal access token
	// (CHERRYPI_AGENT_TOKEN_FILE). Default: $CHERRYPI_DIR/agent.token. The
	// bundled Pi tools read it on every call, so rotating the file does not
	// require a Pi respawn.
	AgentTokenFile string

	// LogFormat is the stdout log format: "text" (default) or "json".
	// Parsed from CHERRYPI_LOG_FORMAT.
	LogFormat string

	// LogLevel is the minimum level to log: debug, info (default), warn,
	// error. Parsed from CHERRYPI_LOG_LEVEL.
	LogLevel string
}

// cherryPiDir resolves CherryPi's data directory: $CHERRYPI_DIR, or
// ~/.cherrypi when unset.
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

// ParseConfig reads the agent configuration from the environment.
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
