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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/processlog"
	"cherrypi.vladfaust.com/agency/internal/storage"
)

type Config struct {
	Port uint16

	H2C bool

	FrontendProxyUrl *url.URL

	// URL of the VitePress dev server to proxy /docs to. Only honored in

	DocsProxyUrl *url.URL

	Dir string

	DatabaseDSN string

	AdminUsername string
	AdminPassword string

	InsecureFirstRegisteredUserBecomeAdmin bool

	PublicURL string

	ExposeMagicLinks bool

	S3 storage.Config

	AttachmentMaxBytes int64

	AttachmentMaxPerMessage int

	SitesInsecureHTTP bool

	LogFiles int

	PluginDataDir string

	PluginAllowPrivateNetwork bool

	PluginDisableTicks bool

	PluginPGRequireDatabase bool

	OTelEnabled bool
}

func parseEnvUint16(envVar string) uint16 {
	value := os.Getenv(envVar)
	if value == "" {
		return 0
	}
	v, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		log.Fatalf("Invalid %s: %v", envVar, err)
	}
	return uint16(v)
}

func parseEnvBool(envVar string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envVar))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func parseEnvInt64(envVar string, def int64) int64 {
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return def
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		log.Fatalf("Invalid %s: want a positive integer, got %q", envVar, value)
	}
	return n
}

func agencyDir() string {
	if dir := os.Getenv("CHERRYPI_AGENCY_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Cannot resolve CHERRYPI_AGENCY_DIR: %v", err)
	}
	return filepath.Join(home, ".cherrypi-agency")
}

func ParseConfig() Config {
	cfg := Config{}

	cfg.Dir = agencyDir()

	dsn := os.Getenv("CHERRYPI_AGENCY_DATABASE_DSN")
	if dsn == "" {
		dsn = filepath.Join(cfg.Dir, "agency.sqlite3")
	}
	cfg.DatabaseDSN = dsn

	port := parseEnvUint16("CHERRYPI_AGENCY_PORT")
	if port == 0 {
		port = 8080
	}
	cfg.Port = port
	cfg.H2C = parseEnvBool("CHERRYPI_H2C")

	if value := os.Getenv("CHERRYPI_FRONTEND_PROXY_URL"); value != "" {
		u, err := url.Parse(value)
		if err != nil {
			log.Fatalf("Invalid CHERRYPI_FRONTEND_PROXY_URL: %v", err)
		}
		cfg.FrontendProxyUrl = u
	}

	if value := os.Getenv("CHERRYPI_DOCS_PROXY_URL"); value != "" {
		u, err := url.Parse(value)
		if err != nil {
			log.Fatalf("Invalid CHERRYPI_DOCS_PROXY_URL: %v", err)
		}
		cfg.DocsProxyUrl = u
	}

	cfg.AdminUsername = strings.TrimSpace(os.Getenv("CHERRYPI_ADMIN_USERNAME"))
	if cfg.AdminUsername == "" {
		cfg.AdminUsername = "admin"
	}
	cfg.AdminPassword = os.Getenv("CHERRYPI_ADMIN_PASSWORD")
	cfg.InsecureFirstRegisteredUserBecomeAdmin = parseEnvBool(
		"CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN",
	)
	cfg.ExposeMagicLinks = parseEnvBool("CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS")

	cfg.PublicURL = strings.TrimRight(os.Getenv("CHERRYPI_PUBLIC_URL"), "/")
	if cfg.PublicURL == "" {
		cfg.PublicURL = "http://localhost:" + strconv.Itoa(int(cfg.Port))
	}

	parseS3Config(&cfg)

	cfg.SitesInsecureHTTP = parseEnvBool("CHERRYPI_SITES_INSECURE_HTTP")

	cfg.AttachmentMaxBytes = parseEnvInt64("CHERRYPI_ATTACHMENT_MAX_BYTES", 25*1024*1024)
	cfg.AttachmentMaxPerMessage = int(
		parseEnvInt64("CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE", 10),
	)

	cfg.LogFiles = processlog.ParseKeep(os.Getenv("CHERRYPI_AGENCY_LOG_FILES"))

	cfg.PluginDataDir = strings.TrimSpace(os.Getenv("CHERRYPI_PLUGIN_DATA_DIR"))
	if cfg.PluginDataDir == "" {
		cfg.PluginDataDir = cfg.Dir
	}
	cfg.PluginAllowPrivateNetwork = parseEnvBool("CHERRYPI_PLUGIN_ALLOW_PRIVATE_NETWORK")
	cfg.PluginDisableTicks = parseEnvBool("CHERRYPI_PLUGIN_DISABLE_TICKS")
	cfg.PluginPGRequireDatabase = parseEnvBool("CHERRYPI_PLUGIN_PG_REQUIRE_DATABASE")

	cfg.OTelEnabled = parseOTelEnabled()

	return cfg
}

func parseOTelEnabled() bool {
	if value := strings.TrimSpace(os.Getenv("CHERRYPI_OTEL_ENABLED")); value != "" {
		return parseEnvBool("CHERRYPI_OTEL_ENABLED")
	}

	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}

	return false
}

func parseS3Config(cfg *Config) {
	region := strings.TrimSpace(os.Getenv("CHERRYPI_S3_REGION"))
	if region == "" {
		region = "us-east-1"
	}

	prefix := strings.Trim(os.Getenv("CHERRYPI_S3_KEY_PREFIX"), "/")
	if prefix != "" {
		prefix += "/"
	}

	cfg.S3 = storage.Config{
		Endpoint:        storage.NormalizeEndpoint(os.Getenv("CHERRYPI_S3_ENDPOINT")),
		Bucket:          strings.TrimSpace(os.Getenv("CHERRYPI_S3_BUCKET")),
		Region:          region,
		AccessKeyID:     os.Getenv("CHERRYPI_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CHERRYPI_S3_SECRET_ACCESS_KEY"),
		ForcePathStyle:  parseEnvBool("CHERRYPI_S3_FORCE_PATH_STYLE"),
		KeyPrefix:       prefix,
	}
}
