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
	"strconv"
	"strings"
	"time"

	agenthealth "cherrypi.vladfaust.com/agent/internal/health"
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

	AgentToken string

	AgentTokenFixed bool

	AgentTokenFile string

	LogFormat string

	LogLevel string

	LogFile string

	LogFileMaxBytes int64

	LogFiles int

	HealthInterval time.Duration

	HealthHostProc string
	HealthHostSys  string
	HealthHostRoot string

	HealthDisks []agenthealth.DiskTarget
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
	cfg.LogFile = os.Getenv("CHERRYPI_LOG_FILE")
	if cfg.LogFile == "" {
		cfg.LogFile = filepath.Join(cfg.Dir, "logs", "agent.log")
	}
	cfg.LogFileMaxBytes = 8 << 20
	if raw := strings.TrimSpace(os.Getenv("CHERRYPI_LOG_FILE_MAX_MB")); raw != "" {
		if mb, err := strconv.Atoi(raw); err == nil && mb > 0 {
			cfg.LogFileMaxBytes = int64(mb) << 20
		} else {
			log.Printf("Ignoring CHERRYPI_LOG_FILE_MAX_MB=%q: expected a positive integer", raw)
		}
	}
	cfg.LogFiles = 3
	if raw := strings.TrimSpace(os.Getenv("CHERRYPI_LOG_FILES")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			cfg.LogFiles = n
		} else {
			log.Printf("Ignoring CHERRYPI_LOG_FILES=%q: expected a non-negative integer", raw)
		}
	}

	cfg.AgencyURL = os.Getenv("CHERRYPI_AGENCY_URL")
	cfg.TunnelTransport = strings.ToLower(
		strings.TrimSpace(os.Getenv("CHERRYPI_AGENT_TUNNEL_TRANSPORT")),
	)

	rawToken := strings.TrimSpace(os.Getenv("CHERRYPI_AGENT_TOKEN"))
	if strings.HasPrefix(rawToken, "pat_") {
		cfg.AgentToken = rawToken
	} else if rawToken != "" {
		log.Printf("Ignoring CHERRYPI_AGENT_TOKEN: expected a personal access token (pat_…)")
	}
	cfg.AgentTokenFixed = cfg.AgentToken != ""
	cfg.AgentTokenFile = os.Getenv("CHERRYPI_AGENT_TOKEN_FILE")
	if cfg.AgentTokenFile == "" {
		cfg.AgentTokenFile = filepath.Join(cfg.Dir, "agent.token")
	}

	cfg.HealthInterval = parseHealthInterval(os.Getenv("CHERRYPI_HEALTH_INTERVAL"))
	cfg.HealthHostProc = hostPath("CHERRYPI_HEALTH_HOST_PROC", "/host/proc", "/proc")
	cfg.HealthHostSys = hostPath("CHERRYPI_HEALTH_HOST_SYS", "/host/sys", "/sys")
	cfg.HealthHostRoot = hostPath("CHERRYPI_HEALTH_HOST_ROOT", "/host/root", "/")
	cfg.HealthDisks = parseHealthDisks(os.Getenv("CHERRYPI_HEALTH_DISKS"))
	cfg.HealthDisks = append(
		defaultHealthDisks(cfg.HealthHostRoot, cfg.Dir),
		cfg.HealthDisks...,
	)
	cfg.HealthDisks = dedupeHealthDisks(cfg.HealthDisks)

	return cfg
}

const DefaultHealthInterval = time.Minute

func parseHealthInterval(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultHealthInterval
	}
	interval, err := time.ParseDuration(raw)
	if err != nil || interval < 0 {
		log.Printf("Ignoring CHERRYPI_HEALTH_INTERVAL=%q: expected a duration", raw)
		return DefaultHealthInterval
	}
	return interval
}

func hostPath(env, host, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	if info, err := os.Stat(host); err == nil && info.IsDir() {
		return host
	}
	return fallback
}

func defaultHealthDisks(hostRoot, dir string) []agenthealth.DiskTarget {
	rootLabel := "Root"
	if hostRoot != "/" {
		rootLabel = "Host root"
	}
	disks := []agenthealth.DiskTarget{{Label: rootLabel, Path: hostRoot}}
	if dir != "" && dir != hostRoot {
		disks = append(disks, agenthealth.DiskTarget{Label: "CherryPi data", Path: dir})
	}
	return disks
}

func parseHealthDisks(raw string) []agenthealth.DiskTarget {
	var disks []agenthealth.DiskTarget
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		label, path := "", item
		if before, after, ok := strings.Cut(item, "="); ok {
			label, path = strings.TrimSpace(before), strings.TrimSpace(after)
		}
		if path == "" {
			continue
		}
		if label == "" {
			label = path
		}
		disks = append(disks, agenthealth.DiskTarget{Label: label, Path: path})
	}
	return disks
}

func dedupeHealthDisks(disks []agenthealth.DiskTarget) []agenthealth.DiskTarget {
	seen := map[string]struct{}{}
	out := disks[:0]
	for _, disk := range disks {
		if _, ok := seen[disk.Path]; ok {
			continue
		}
		seen[disk.Path] = struct{}{}
		out = append(out, disk)
	}
	return out
}
