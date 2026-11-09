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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cherrypi.vladfaust.com/agency/cmd/agency/buildinfo"
	"cherrypi.vladfaust.com/agency/internal/plugins"
	"cherrypi.vladfaust.com/agency/internal/plugins/e2e"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "version":
		fmt.Printf("cherrypi-plugin %s\n", buildinfo.Version)
		fmt.Printf("plugin API %s\n", plugins.APIVersion)
	case "lint":
		err = lint(os.Args[2:])
	case "test":
		err = test(os.Args[2:])
	case "pack":
		err = pack(os.Args[2:])
	case "e2e":
		err = runE2E(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func contextBackground() context.Context { return context.Background() }

func parseDir(fs *flag.FlagSet, args []string) (string, error) {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if strings.ContainsRune(name, '=') {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if err := fs.Parse(flags); err != nil {
		return "", err
	}
	if len(positional) > 0 {
		return positional[0], nil
	}
	return ".", nil
}

func usage() {
	fmt.Fprint(os.Stderr, `cherrypi-plugin — validate, test, pack, and run a CherryPi plugin

Usage:
  cherrypi-plugin lint [dir]                 Validate the bundle
  cherrypi-plugin test [dir]                 Run test/**/*.lua in the sandbox
  cherrypi-plugin pack [dir] [-o out.zip]    Build an installable bundle
  cherrypi-plugin e2e [dir] --scenario FILE  Drive the bundle against an agency
  cherrypi-plugin version                    Print the CLI and API versions

The e2e command boots a throwaway agency (--agency-bin, CHERRYPI_AGENCY_BIN, or
cherrypi-agency on PATH), seeds a project/agent, installs the bundle, applies
the scenario's settings, and runs its steps. Add --with-agent to run the tool
steps through a real agent and mock LLM (needs cherrypi-agent, cherrypi-mockllm,
and pi).
`)
}

func lint(args []string) error {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	manifest, err := plugins.LintBundle(dir)
	if err != nil {
		return err
	}
	fmt.Printf("ok %s %s (plugin API %s)\n", manifest.ID, manifest.Version, plugins.APIVersion)
	return nil
}

func test(args []string) error {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	results, err := plugins.RunTests(contextBackground(), dir)
	if err != nil {
		return err
	}
	failed := 0
	for _, result := range results {
		if result.Passed {
			fmt.Printf("PASS %s\n", result.File)
			continue
		}
		failed++
		fmt.Printf("FAIL %s\n     %s\n", result.File, result.Message)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d test files failed", failed, len(results))
	}
	fmt.Printf("%d test files passed\n", len(results))
	return nil
}

func pack(args []string) error {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	out := fs.String("o", "", "output zip (default <id>-<version>.zip)")
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	manifest, err := plugins.LintBundle(dir)
	if err != nil {
		return err
	}
	data, err := plugins.PackBundleDir(dir)
	if err != nil {
		return err
	}
	target := *out
	if target == "" {
		target = fmt.Sprintf("%s-%s.zip", manifest.ID, manifest.Version)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return err
	}
	abs, _ := filepath.Abs(target)
	fmt.Printf("packed %s (%d bytes)\n", abs, len(data))
	return nil
}

func runE2E(args []string) error {
	fs := flag.NewFlagSet("e2e", flag.ExitOnError)
	scenario := fs.String("scenario", "plugin-test.yaml", "scenario file")
	agencyBin := fs.String(
		"agency-bin",
		"",
		"agency binary (default: CHERRYPI_AGENCY_BIN or cherrypi-agency on PATH)",
	)
	agencyURL := fs.String("agency-url", "", "use an already-running agency instead of booting one")
	agencyUsername := fs.String(
		"agency-username",
		"",
		"admin username for --agency-url (default: CHERRYPI_ADMIN_USERNAME or admin)",
	)
	agencyPassword := fs.String(
		"agency-password",
		"",
		"admin password for --agency-url (default: CHERRYPI_ADMIN_PASSWORD)",
	)
	withAgent := fs.Bool("with-agent", false, "run tool steps through a real agent and mock LLM")
	agentBin := fs.String(
		"agent-bin",
		"",
		"agent binary (default: CHERRYPI_AGENT_BIN or cherrypi-agent on PATH)",
	)
	mockllmBin := fs.String(
		"mockllm-bin",
		"",
		"mock LLM binary (default: CHERRYPI_MOCKLLM_BIN or cherrypi-mockllm on PATH)",
	)
	keep := fs.Bool("keep", false, "keep the agency data directory for debugging")
	verbose := fs.Bool("v", false, "print agency and agent logs")
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	return e2e.Run(contextBackground(), e2e.Options{
		BundleDir:      dir,
		Scenario:       *scenario,
		AgencyBin:      *agencyBin,
		AgencyURL:      *agencyURL,
		AgencyUsername: *agencyUsername,
		AgencyPassword: *agencyPassword,
		WithAgent:      *withAgent,
		AgentBin:       *agentBin,
		MockLLMBin:     *mockllmBin,
		Keep:           *keep,
		Verbose:        *verbose,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
	})
}
