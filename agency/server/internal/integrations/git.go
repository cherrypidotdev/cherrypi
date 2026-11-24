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

package integrations

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrFeatureDisabled = errors.New("git sources are disabled")

var ErrGitUnavailable = errors.New("the agency has no git or ssh binary")

const (
	CredentialToken  = "token"
	CredentialSSHKey = "ssh_key"
)

const (
	UpdateStaged = "staged"

	UpdateAuto = "auto"
)

const (
	SyncStateNone    = ""
	SyncStateOK      = "ok"
	SyncStatePending = "pending"
	SyncStateError   = "error"
)

const (
	maxSkillFileBytes   = 256 << 10
	maxSkillFilesBytes  = 8 << 20
	maxSkillFiles       = 256
	maxPendingBytes     = 16 << 20
	maxGitOutputBytes   = 1 << 20
	minPollIntervalS    = 60
	maxPollIntervalS    = 24 * 60 * 60
	defaultPollInterval = 15 * 60
)

const gitTimeout = 2 * time.Minute

type Repo struct {
	URL    string
	Scheme string
	Host   string
	Port   int64
	User   string
	Path   string
}

var scpLikePattern = regexp.MustCompile(`^(?:([^@/:]+)@)?([^@/:]+):(.+)$`)
var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func parseRepoURL(raw string) (Repo, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Repo{}, invalidf("a repository URL is required")
	}
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return Repo{}, invalidf("the repository URL is not a URL")
		}
		if parsed.RawQuery != "" || parsed.Fragment != "" {
			return Repo{}, invalidf("the repository URL cannot carry a query or a fragment")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "https" && scheme != "http" && scheme != "ssh" {
			return Repo{}, invalidf("only http(s) and ssh repositories are supported")
		}
		host := strings.ToLower(parsed.Hostname())
		if host == "" {
			return Repo{}, invalidf("the repository URL has no host")
		}
		port := int64(0)
		if raw := parsed.Port(); raw != "" {
			value, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || value <= 0 || value > 65535 {
				return Repo{}, invalidf("the repository URL has an invalid port")
			}
			port = value
		}
		path := strings.TrimPrefix(parsed.Path, "/")
		switch scheme {
		case "https", "http":

			if parsed.User != nil {
				return Repo{}, invalidf(
					"put the credential in a credential, not in the repository URL",
				)
			}
			if path == "" {
				return Repo{}, invalidf("the repository URL has no path")
			}
			return Repo{
				URL:    trimmed,
				Scheme: scheme,
				Host:   host,
				Port:   port,
				Path:   path,
			}, nil
		case "ssh":
			if path == "" {
				return Repo{}, invalidf("the repository URL has no path")
			}
			if _, hasPassword := parsed.User.Password(); hasPassword {
				return Repo{}, invalidf(
					"put the credential in a credential, not in the repository URL",
				)
			}
			user := parsed.User.Username()
			if user == "" {
				user = "git"
			}
			return Repo{
				URL:    trimmed,
				Scheme: scheme,
				Host:   host,
				Port:   port,
				User:   user,
				Path:   path,
			}, nil
		default:
			return Repo{}, invalidf("only http(s) and ssh repositories are supported")
		}
	}

	match := scpLikePattern.FindStringSubmatch(trimmed)
	if match == nil {
		return Repo{}, invalidf(
			"the repository URL must be http(s)://… or user@host:path",
		)
	}
	user, host, path := match[1], strings.ToLower(match[2]), match[3]
	if strings.Contains(path, ":") {
		return Repo{}, invalidf("the repository URL has an invalid path")
	}
	return Repo{
		URL:    trimmed,
		Scheme: "ssh",
		Host:   host,
		User:   user,
		Path:   path,
	}, nil
}

// redacted renders the address for a log line or an error.
func (r Repo) redacted() string {
	return r.Scheme + "://" + r.Host + "/" + r.Path
}

type gitAuth struct {
	Env     []string
	cleanup func()

	redact []string
}

func (a *gitAuth) close() {
	if a != nil && a.cleanup != nil {
		a.cleanup()
	}
}

type credential struct {
	ID          int64
	Name        string
	Kind        string
	Host        string
	Port        int64
	Username    string
	Secret      string
	KnownHosts  string
	Fingerprint string
}

// and an SSH command line that refuses an unpinned host key.
func authEnv(repo Repo, cred *credential) (*gitAuth, error) {
	if cred == nil {
		if repo.Scheme == "ssh" {
			return nil, invalidf("an SSH repository needs a credential")
		}
		return &gitAuth{}, nil
	}
	switch cred.Kind {
	case CredentialToken:
		if repo.Scheme != "https" && repo.Scheme != "http" {
			return nil, invalidf("a token credential only reaches an http(s) repository")
		}
		username := cred.Username
		if username == "" {
			username = "git"
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + cred.Secret))
		header := "Authorization: Basic " + encoded
		return &gitAuth{
			Env: []string{
				"GIT_CONFIG_COUNT=1",
				"GIT_CONFIG_KEY_0=http.extraHeader",
				"GIT_CONFIG_VALUE_0=" + header,
			},
			redact: []string{header, encoded, cred.Secret},
		}, nil
	case CredentialSSHKey:
		if repo.Scheme != "ssh" {
			return nil, invalidf("an SSH key credential only reaches an ssh repository")
		}
		if strings.TrimSpace(cred.KnownHosts) == "" {
			return nil, invalidf(
				"pin the host key before fetching from an SSH repository",
			)
		}
		dir, err := os.MkdirTemp("", "cherrypi-git-")
		if err != nil {
			return nil, err
		}
		keyPath := filepath.Join(dir, "id")
		knownHostsPath := filepath.Join(dir, "known_hosts")
		key := cred.Secret
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}
		if err := os.WriteFile(keyPath, []byte(key), 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		known := cred.KnownHosts
		if !strings.HasSuffix(known, "\n") {
			known += "\n"
		}
		if err := os.WriteFile(knownHostsPath, []byte(known), 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		port := repo.Port
		if port == 0 {
			port = cred.Port
		}
		if port == 0 {
			port = 22
		}
		command := fmt.Sprintf(
			"ssh -i %q -o IdentitiesOnly=yes -o BatchMode=yes "+
				"-o StrictHostKeyChecking=yes -o UserKnownHostsFile=%q -p %d",
			keyPath, knownHostsPath, port,
		)
		return &gitAuth{
			Env:     []string{"GIT_SSH_COMMAND=" + command},
			cleanup: func() { _ = os.RemoveAll(dir) },
			redact:  []string{cred.Secret},
		}, nil
	default:
		return nil, invalidf("unknown credential kind %q", cred.Kind)
	}
}

func runGit(ctx context.Context, repo Repo, auth *gitAuth, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Env = append(gitEnvironment(auth), command.Env...)
	command.Env = append(command.Env, "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		message = redactSecrets(message, auth)
		message = truncate(message, maxGitOutputBytes)
		return "", fmt.Errorf(
			"git %s failed for %s: %s",
			args[0],
			repo.redacted(),
			message,
		)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// echo what it was given — a header, a URL, a host's error line — and that

func redactSecrets(message string, auth *gitAuth) string {
	if auth == nil {
		return message
	}
	for _, secret := range auth.redact {
		if secret == "" {
			continue
		}
		message = strings.ReplaceAll(message, secret, "***")
	}
	return message
}

func gitEnvironment(auth *gitAuth) []string {
	env := []string{

		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_LFS_SKIP_SMUDGE=1",
	}
	if auth != nil {
		env = append(env, auth.Env...)
	}
	return env
}

func remoteHead(
	ctx context.Context,
	repo Repo,
	auth *gitAuth,
	ref string,
) (commit, fetchRef string, err error) {
	ref = strings.TrimSpace(ref)
	if shaPattern.MatchString(ref) {
		return ref, ref, nil
	}
	if ref == "" {
		out, err := runGit(ctx, repo, auth, "ls-remote", "--symref", repo.URL, "HEAD")
		if err != nil {
			return "", "", err
		}
		commit, fetchRef = parseLsRemoteHead(out)
		if commit == "" {
			return "", "", fmt.Errorf("%s has no default branch", repo.redacted())
		}
		return commit, fetchRef, nil
	}
	out, err := runGit(
		ctx, repo, auth, "ls-remote", repo.URL,
		"refs/heads/"+ref, "refs/tags/"+ref, "refs/tags/"+ref+"^{}",
	)
	if err != nil {
		return "", "", err
	}
	commit, fetchRef = parseLsRemote(out, ref)
	if commit == "" {
		return "", "", invalidf("ref %q does not exist in %s", ref, repo.redacted())
	}
	return commit, fetchRef, nil
}

// `ref: refs/heads/main\tHEAD` line followed by the commit.
func parseLsRemoteHead(out string) (commit, fetchRef string) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "ref:" {
			fetchRef = fields[1]
			continue
		}
		if len(fields) == 2 && fields[1] == "HEAD" && shaPattern.MatchString(fields[0]) {
			commit = fields[0]
		}
	}
	return commit, fetchRef
}

func parseLsRemote(out, ref string) (commit, fetchRef string) {
	commits := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !shaPattern.MatchString(fields[0]) {
			continue
		}
		commits[fields[1]] = fields[0]
	}
	for _, candidate := range []string{
		"refs/heads/" + ref,
		"refs/tags/" + ref + "^{}",
		"refs/tags/" + ref,
	} {
		if sha := commits[candidate]; sha != "" {
			return sha, candidate
		}
	}
	return "", ""
}

func fetchRefspec(commit, fetchRef string) string {
	if fetchRef == "" {
		return commit
	}
	return fetchRef
}

func exportRepo(
	ctx context.Context,
	repo Repo,
	auth *gitAuth,
	commit, fetchRef, dir, sparsePrefix string,
) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := runGit(ctx, repo, auth, "init", "-q", dir); err != nil {
		return err
	}
	if _, err := runGit(
		ctx,
		repo,
		auth,
		"-C",
		dir,
		"remote",
		"add",
		"origin",
		repo.URL,
	); err != nil {
		return err
	}
	if sparsePrefix != "" {
		if _, err := runGit(
			ctx, repo, auth, "-C", dir, "sparse-checkout", "set", "--no-cone", sparsePrefix,
		); err != nil {
			return err
		}
	}
	if _, err := runGit(
		ctx, repo, auth, "-C", dir, "fetch", "--depth=1", "--no-tags",
		"origin", fetchRefspec(commit, fetchRef),
	); err != nil {
		return err
	}
	if _, err := runGit(
		ctx,
		repo,
		auth,
		"-C",
		dir,
		"checkout",
		"-q",
		"--detach",
		"FETCH_HEAD",
	); err != nil {
		return err
	}

	_, err := runGit(ctx, repo, auth, "-C", dir, "rev-parse", "HEAD")
	return err
}

type CLIStatus struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	OK      bool   `json:"ok"`
}

type GitSourcesStatus struct {
	Enabled bool      `json:"enabled"`
	Ready   bool      `json:"ready"`
	Git     CLIStatus `json:"git"`
	SSH     CLIStatus `json:"ssh"`

	Message string `json:"message,omitempty"`
}

func (s *Service) probeCLIs(ctx context.Context) (gitCLI, sshCLI CLIStatus) {
	s.cliMu.Lock()
	defer s.cliMu.Unlock()
	if s.cliCached && time.Since(s.cliAt) < 30*time.Second {
		return s.cliGit, s.cliSSH
	}
	s.cliGit = probeCLI(ctx, "git", "--version")
	s.cliSSH = probeCLI(ctx, "ssh", "-V")
	s.cliAt = time.Now()
	s.cliCached = true
	return s.cliGit, s.cliSSH
}

func probeCLI(ctx context.Context, name string, args ...string) CLIStatus {
	path, err := exec.LookPath(name)
	if err != nil {
		return CLIStatus{}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	if err := command.Run(); err != nil {

		return CLIStatus{Path: path, Version: truncate(strings.TrimSpace(out.String()), 200)}
	}
	return CLIStatus{
		Path:    path,
		Version: truncate(strings.TrimSpace(out.String()), 200),
		OK:      true,
	}
}

func scanHostKeys(
	ctx context.Context,
	host string,
	port int64,
) (keys []HostKey, knownHosts string, err error) {
	if _, lookErr := exec.LookPath("ssh-keyscan"); lookErr != nil {
		return nil, "", ErrGitUnavailable
	}
	if port == 0 {
		port = 22
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		"ssh-keyscan",
		"-T",
		"10",
		"-p",
		strconv.FormatInt(port, 10),
		host,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if runErr := command.Run(); runErr != nil && stdout.Len() == 0 {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = runErr.Error()
		}
		return nil, "", fmt.Errorf("ssh-keyscan %s failed: %s", host, truncate(message, 400))
	}
	lines := make([]string, 0, 4)
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		lines = append(lines, line)
		keys = append(keys, HostKey{
			Type:        fields[1],
			Fingerprint: fingerprintOf(ctx, line),
		})
	}
	if len(lines) == 0 {
		return nil, "", fmt.Errorf("ssh-keyscan %s returned no host key", host)
	}
	return keys, strings.Join(lines, "\n") + "\n", nil
}

type HostKey struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
}

// fingerprintOf renders the SHA256 fingerprint of one known-hosts line through

func fingerprintOf(ctx context.Context, line string) string {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return ""
	}
	dir, err := os.MkdirTemp("", "cherrypi-scan-")
	if err != nil {
		return ""
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "key.pub")
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "ssh-keygen", "-lf", path, "-E", "sha256")
	out, err := command.Output()
	if err != nil {
		return ""
	}
	fields = strings.Fields(string(out))
	if len(fields) < 2 {
		return ""
	}
	return fields[1]
}

func fingerprintOfPublicKey(key string) string {
	return fingerprintOf(context.Background(), key)
}

func publicKeyOf(privateKey string) (string, error) {
	dir, err := os.MkdirTemp("", "cherrypi-key-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "id")
	if err := os.WriteFile(path, []byte(privateKey), 0o600); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "ssh-keygen", "-y", "-f", path)
	out, err := command.Output()
	if err != nil {
		return "", invalidf("the SSH private key could not be read")
	}
	return strings.TrimSpace(string(out)), nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}
