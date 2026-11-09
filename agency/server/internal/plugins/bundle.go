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

package plugins

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

const (
	maxBundleBytes = 32 << 20
	maxBundleFile  = 16 << 20
	maxBundleFiles = 512
)

type Bundle struct {
	Manifest    Manifest
	ManifestRaw []byte
	Files       map[string][]byte
	Checksum    string
}

func ReadBundle(data []byte) (Bundle, error) {
	if len(data) > maxBundleBytes {
		return Bundle{}, fmt.Errorf("bundle exceeds %d bytes", maxBundleBytes)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Bundle{}, fmt.Errorf("open bundle: %w", err)
	}
	var manifestRaw []byte
	files := map[string][]byte{}
	total := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !f.Mode().IsRegular() {
			return Bundle{}, fmt.Errorf("bundle entry %q is not a regular file", f.Name)
		}
		name := filepath.ToSlash(f.Name)
		if err := checkBundlePath(name); err != nil {
			return Bundle{}, err
		}
		if len(files) >= maxBundleFiles {
			return Bundle{}, errors.New("bundle has too many files")
		}
		rc, err := f.Open()
		if err != nil {
			return Bundle{}, err
		}
		body, err := io.ReadAll(io.LimitReader(rc, maxBundleFile+1))
		_ = rc.Close()
		if err != nil {
			return Bundle{}, err
		}
		if len(body) > maxBundleFile {
			return Bundle{}, fmt.Errorf("bundle entry %q is too large", name)
		}
		total += len(body)
		if total > maxBundleBytes {
			return Bundle{}, errors.New("bundle expands beyond the size limit")
		}
		if name == "plugin.json" {
			manifestRaw = body
			continue
		}
		files[name] = body
	}
	if len(manifestRaw) == 0 {
		return Bundle{}, errors.New("bundle needs plugin.json")
	}
	manifest, err := ParseManifest(manifestRaw)
	if err != nil {
		return Bundle{}, err
	}
	manifest.NormalizeHandlers()
	if files[manifest.Entry] == nil {
		return Bundle{}, fmt.Errorf("bundle needs %s", manifest.Entry)
	}
	sum := sha256.Sum256(data)
	return Bundle{
		Manifest:    manifest,
		ManifestRaw: manifestRaw,
		Files:       files,
		Checksum:    hex.EncodeToString(sum[:]),
	}, nil
}

var vcsMetadata = map[string]bool{
	".git":   true,
	".hg":    true,
	".svn":   true,
	".bzr":   true,
	".jj":    true,
	".pijul": true,
	"_darcs": true,
	"CVS":    true,
}

func LoadBundleDir(dir string) (Bundle, error) {
	manifestRaw, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return Bundle{}, fmt.Errorf("read plugin.json: %w", err)
	}
	manifest, err := ParseManifest(manifestRaw)
	if err != nil {
		return Bundle{}, err
	}
	manifest.NormalizeHandlers()

	files := map[string][]byte{}
	total := 0
	ignores := newBundleIgnore(dir)
	err = filepath.WalkDir(dir, func(entryPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(dir, entryPath)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if name == "." {
			return nil
		}
		if vcsMetadata[entry.Name()] {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if ignores.ignored(name, true) {
				return fs.SkipDir
			}
			return nil
		}
		if name == "plugin.json" || ignores.ignored(name, false) {
			return nil
		}
		if err := checkBundlePath(name); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("bundle entry %q is not a regular file", name)
		}
		if info.Size() > maxBundleFile {
			return fmt.Errorf("bundle entry %q is too large", name)
		}
		if len(files) >= maxBundleFiles {
			return errors.New("bundle has too many files")
		}
		body, err := os.ReadFile(entryPath)
		if err != nil {
			return err
		}
		total += len(body)
		if total > maxBundleBytes {
			return errors.New("bundle exceeds the size limit")
		}
		files[name] = body
		return nil
	})
	if err != nil {
		return Bundle{}, err
	}
	if files[manifest.Entry] == nil {
		return Bundle{}, fmt.Errorf("bundle needs %s", manifest.Entry)
	}
	return Bundle{Manifest: manifest, ManifestRaw: manifestRaw, Files: files}, nil
}

type ignoreRule struct {
	line    string
	negate  bool
	matcher *ignore.GitIgnore
}

func parseIgnoreRules(data []byte) []ignoreRule {
	var rules []ignoreRule
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.Trim(line, " ")
		if line == "" {
			continue
		}
		negate := false
		if strings.HasPrefix(line, "!") {
			negate = true
			line = line[1:]
			if line == "" {
				continue
			}
		}
		rules = append(rules, ignoreRule{
			line:    line,
			negate:  negate,
			matcher: ignore.CompileIgnoreLines(line),
		})
	}
	return rules
}

type bundleIgnore struct {
	root  string
	cache map[string][]ignoreRule
}

func newBundleIgnore(root string) *bundleIgnore {
	return &bundleIgnore{root: root, cache: map[string][]ignoreRule{}}
}

func (b *bundleIgnore) rules(dir string) []ignoreRule {
	if rules, ok := b.cache[dir]; ok {
		return rules
	}
	data, err := os.ReadFile(filepath.Join(b.root, filepath.FromSlash(dir), ".gitignore"))
	var rules []ignoreRule
	if err == nil {
		rules = parseIgnoreRules(data)
	}
	b.cache[dir] = rules
	return rules
}

func (b *bundleIgnore) ignored(rel string, isDir bool) bool {
	if rel == "" || rel == "." {
		return false
	}
	chain := []string{""}
	if dir := path.Dir(rel); dir != "." {
		parts := strings.Split(dir, "/")
		for i := range parts {
			chain = append(chain, strings.Join(parts[:i+1], "/"))
		}
	}
	ignored := false
	for _, base := range chain {
		name := rel
		if base != "" {
			name = strings.TrimPrefix(rel, base+"/")
		}
		for _, rule := range b.rules(base) {
			candidate := name
			if isDir {
				switch {
				case strings.HasSuffix(rule.line, "/"):
					candidate += "/"
				case strings.HasSuffix(rule.line, "/**"):

					continue
				}
			}
			if rule.matcher.MatchesPath(candidate) {
				ignored = !rule.negate
			}
		}
	}
	return ignored
}

func checkBundlePath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") ||
		strings.ContainsRune(name, '\\') {
		return fmt.Errorf("unsafe bundle path %q", name)
	}
	return nil
}

func PackBundleDir(dir string) ([]byte, error) {
	bundle, err := LoadBundleDir(dir)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name string, body []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		return err
	}
	if err := write("plugin.json", bundle.ManifestRaw); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := write(name, bundle.Files[name]); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var testModuleReference = regexp.MustCompile(`require\s*\(?\s*["']cherrypi\.test["']`)

func LintBundle(dir string) (Manifest, error) {
	bundle, err := LoadBundleDir(dir)
	if err != nil {
		return Manifest{}, err
	}
	if ok, err := bundle.Manifest.RequiresAPIVersion(APIVersion); err != nil {
		return Manifest{}, err
	} else if !ok {
		return Manifest{}, fmt.Errorf(
			"bundle requires agency %s, this agency implements %s",
			bundle.Manifest.RequiresAgency,
			APIVersion,
		)
	}
	for name, body := range bundle.Files {
		if !strings.HasSuffix(name, ".lua") {
			continue
		}
		if !strings.HasPrefix(name, "test/") && testModuleReference.Match(body) {
			return Manifest{}, fmt.Errorf(
				"%s requires cherrypi.test, which exists only in the test runner",
				name,
			)
		}
		if _, err := compileLua(body, name); err != nil {
			return Manifest{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	return bundle.Manifest, nil
}
