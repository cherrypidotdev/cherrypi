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
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TestCaseResult struct {
	File    string
	Passed  bool
	Message string
	Logs    []string
}

func RunTests(ctx context.Context, dir string) ([]TestCaseResult, error) {
	manifest, err := LintBundle(dir)
	if err != nil {
		return nil, err
	}
	bundle, err := LoadBundleDir(dir)
	if err != nil {
		return nil, err
	}
	_ = bundle
	p := Plugin{
		ID:       manifest.ID,
		Name:     manifest.Name,
		Version:  manifest.Version,
		Manifest: manifest,
		Entry:    manifest.Entry,
		ActorID:  1000,
	}
	runtime := NewRuntime()
	if err := runtime.Load(p, dir); err != nil {
		return nil, err
	}
	files, err := testFiles(dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no test files under %s/test", dir)
	}

	results := make([]TestCaseResult, 0, len(files))
	for _, file := range files {
		result := TestCaseResult{File: file, Passed: true}
		dataDir, err := os.MkdirTemp("", "cherrypi-plugin-test-")
		if err != nil {
			return nil, err
		}
		db, err := OpenPluginDB(ctx, "sqlite", p.ID, dataDir, false)
		if err != nil {
			_ = os.RemoveAll(dataDir)
			return nil, err
		}
		if err := db.Migrate(ctx, migrationsDir(dir, false)); err != nil {
			_ = db.Close()
			_ = os.RemoveAll(dataDir)
			return nil, err
		}
		fake := NewTestBridge(p.ID)
		host := NewHost(
			ctx,
			p,
			Caller{Kind: "test", Username: "test"},
			"test-"+strings.TrimSuffix(filepath.Base(file), ".lua"),
			map[string]any{},
			db,
			fake,
			true,
		)
		host.Fake = fake
		runErr := runtime.RunTestFile(ctx, p, host, file)
		_ = db.Close()
		_ = os.RemoveAll(dataDir)
		if runErr != nil {
			result.Passed = false
			result.Message = runErr.Error()
		}
		result.Logs = fake.Logs()
		results = append(results, result)
	}
	return results, nil
}

func testFiles(dir string) ([]string, error) {
	var out []string
	root := filepath.Join(dir, "test")
	ignores := newBundleIgnore(dir)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if vcsMetadata[entry.Name()] {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if entry.IsDir() {
			if ignores.ignored(name, true) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".lua") || ignores.ignored(name, false) {
			return nil
		}
		out = append(out, name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
