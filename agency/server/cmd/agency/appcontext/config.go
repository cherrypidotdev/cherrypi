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
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/processlog"
	"cherrypi.vladfaust.com/agency/internal/storage"
)

type Credential struct {
	Username string
	Password string
}

type Config struct {
	Port uint16

	H2C bool

	FrontendProxyUrl *url.URL

	Dir string

	DatabaseDSN string

	Credentials []Credential

	AdminUsername string
	AdminPassword string

	InsecureFirstRegisteredUserBecomeAdmin bool

	PublicURL string

	ExposeMagicLinks bool

	SeedAgentUsername string
	SeedAgentPassword string

	S3 storage.Config

	AttachmentMaxBytes int64

	AttachmentMaxPerMessage int

	LogFiles int
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

func parseCredentials(s string) ([]Credential, error) {
	if s == "" {
		return nil, nil
	}

	var out []Credential
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		user, pass, ok := strings.Cut(pair, ":")
		if !ok || user == "" || pass == "" {
			return nil, fmt.Errorf("invalid credential pair %q (want user:hex)", pair)
		}
		out = append(out, Credential{Username: user, Password: pass})
	}
	return out, nil
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

	creds, err := parseCredentials(os.Getenv("CHERRYPI_AGENCY_CREDENTIALS"))
	if err != nil {
		log.Fatalf("Invalid CHERRYPI_AGENCY_CREDENTIALS: %v", err)
	}
	cfg.Credentials = creds

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

	if seed := os.Getenv("CHERRYPI_AGENCY_SEED_AGENT"); seed != "" {
		user, pass, ok := strings.Cut(seed, ":")
		if !ok || user == "" || pass == "" {
			log.Fatalf("Invalid CHERRYPI_AGENCY_SEED_AGENT: want user:password")
		}
		cfg.SeedAgentUsername = user
		cfg.SeedAgentPassword = pass
	}

	parseS3Config(&cfg)

	cfg.AttachmentMaxBytes = parseEnvInt64("CHERRYPI_ATTACHMENT_MAX_BYTES", 25*1024*1024)
	cfg.AttachmentMaxPerMessage = int(
		parseEnvInt64("CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE", 10),
	)

	cfg.LogFiles = processlog.ParseKeep(os.Getenv("CHERRYPI_AGENCY_LOG_FILES"))

	return cfg
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
