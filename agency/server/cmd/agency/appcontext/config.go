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

	"cherrypi.vladfaust.com/agency/internal/storage"
)

// Credential is a username/password pair for basic HTTP auth.
type Credential struct {
	Username string
	Password string
}

// Config is the agency's configuration.
type Config struct {
	// Port to listen on. Defaults to 8080.
	Port uint16

	// URL of the frontend dev server to proxy requests to. Only honored in
	// development builds (see the `dev` build tag); nil means serve the
	// embedded SPA.
	FrontendProxyUrl *url.URL

	// Dir is the agency's data directory: the default SQLite database lives
	// under it. Parsed from CHERRYPI_AGENCY_DIR. Default: ~/.cherrypi-agency.
	Dir string

	// DatabaseDSN is the database DSN, parsed from
	// CHERRYPI_AGENCY_DATABASE_DSN. A postgres:// or postgresql:// URL selects
	// PostgreSQL; anything else is a SQLite DSN. Default:
	// $CHERRYPI_AGENCY_DIR/agency.sqlite3.
	DatabaseDSN string

	// Credentials are the initial user basic-auth pairs
	// (CHERRYPI_AGENCY_CREDENTIALS). They seed user accounts on boot; existing
	// accounts keep their role but take the configured password.
	Credentials []Credential

	// AdminUsername/AdminPassword are the first-boot administrator
	// (CHERRYPI_ADMIN_USERNAME, default "admin"; CHERRYPI_ADMIN_PASSWORD,
	// generated and printed once when unset). They only apply to an empty
	// installation.
	AdminUsername string
	AdminPassword string

	// InsecureFirstRegisteredUserBecomeAdmin enables open registration where
	// the first registered user becomes admin
	// (CHERRYPI_INSECURE_FIRST_REGISTERED_USER_BECOMES_ADMIN).
	InsecureFirstRegisteredUserBecomeAdmin bool

	// PublicURL is the external origin used to build magic-link and reset
	// links (CHERRYPI_PUBLIC_URL). Defaults to http://localhost:<port>.
	PublicURL string

	// ExposeMagicLinks returns generated links in API responses. It exists for
	// installations without SMTP and must not be enabled in production
	// (CHERRYPI_AUTH_EXPOSE_MAGIC_LINKS).
	ExposeMagicLinks bool

	// SeedAgentUsername/SeedAgentPassword, when set
	// (CHERRYPI_AGENCY_SEED_AGENT=user:password), ensure an agent and tunnel
	// credential exist at startup. A dev bootstrap for single-box compose.
	SeedAgentUsername string
	SeedAgentPassword string

	// S3 is the object-storage configuration (CHERRYPI_S3_*). Storage is
	// disabled when its endpoint and bucket are unset.
	S3 storage.Config

	// AttachmentMaxBytes caps one uploaded file (CHERRYPI_ATTACHMENT_MAX_BYTES).
	AttachmentMaxBytes int64
	// AttachmentMaxPerMessage caps the attachments sent with one message
	// (CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE).
	AttachmentMaxPerMessage int
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

// parseCredentials parses CHERRYPI_AGENCY_CREDENTIALS: comma-separated
// `user:hex` pairs.
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

// parseEnvBool reads a boolean env var; anything other than 1/true/yes (case
// insensitive) is false.
func parseEnvBool(envVar string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envVar))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// parseEnvInt64 reads a positive integer env var, falling back to def when
// unset.
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

// ParseConfig reads the agency configuration from the environment.
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

	return cfg
}

// parseS3Config fills cfg.S3 from the CHERRYPI_S3_* environment. The endpoint
// and bucket are the enable switch; region defaults, and the key prefix is
// normalized to end in '/'.
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
		Endpoint:        strings.TrimRight(os.Getenv("CHERRYPI_S3_ENDPOINT"), "/"),
		Bucket:          strings.TrimSpace(os.Getenv("CHERRYPI_S3_BUCKET")),
		Region:          region,
		AccessKeyID:     os.Getenv("CHERRYPI_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CHERRYPI_S3_SECRET_ACCESS_KEY"),
		ForcePathStyle:  parseEnvBool("CHERRYPI_S3_FORCE_PATH_STYLE"),
		KeyPrefix:       prefix,
	}
}
