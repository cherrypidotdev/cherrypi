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
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultSkillGlob = ".well-known/skills/*/SKILL.md"

const maxSkillNameLength = 64

const maxSkillDescriptionLength = 1024

const maxSkillProblems = 50

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type discoveredSkill struct {
	Path                   string      `json:"path"`
	RawName                string      `json:"rawName"`
	Name                   string      `json:"name"`
	Description            string      `json:"description"`
	Body                   string      `json:"body"`
	Files                  []SkillFile `json:"files"`
	DisableModelInvocation bool        `json:"disableModelInvocation"`
}

type skillProblem struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func boundProblems(problems []skillProblem) []skillProblem {
	if len(problems) <= maxSkillProblems {
		return problems
	}
	out := make([]skillProblem, 0, maxSkillProblems+1)
	out = append(out, problems[:maxSkillProblems]...)
	out = append(out, skillProblem{
		Message: fmt.Sprintf(
			"and %d more refused document(s); narrow the glob or fix them in the repository",
			len(problems)-maxSkillProblems,
		),
	})
	return out
}

func validateSkillGlob(glob string) (string, error) {
	trimmed := strings.TrimSpace(glob)
	if trimmed == "" {
		return defaultSkillGlob, nil
	}
	if strings.Contains(trimmed, "\\") {
		return "", invalidf("the glob must use / as its separator")
	}
	if strings.Contains(trimmed, "**") {
		return "", invalidf("the glob does not support **; name the levels explicitly")
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", invalidf("the glob must be relative to the repository root")
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == ".." || segment == "." || segment == "" {
			return "", invalidf("the glob cannot contain an empty or relative segment")
		}
	}
	if path.Base(trimmed) != "SKILL.md" {
		return "", invalidf("the glob must end with SKILL.md")
	}
	return trimmed, nil
}

func sparsePrefixOf(glob string) string {
	segments := strings.Split(glob, "/")
	stable := make([]string, 0, len(segments))
	for _, segment := range segments[:len(segments)-1] {
		if strings.ContainsAny(segment, "*?[") {
			break
		}
		stable = append(stable, segment)
	}
	return strings.Join(stable, "/")
}

func discoverSkills(
	root, glob, slug string,
) ([]discoveredSkill, []skillProblem, error) {
	matches, err := matchSkillGlob(root, glob)
	if err != nil {
		return nil, nil, invalidf("the glob is not valid")
	}
	sort.Strings(matches)
	skills := make([]discoveredSkill, 0, len(matches))
	problems := []skillProblem{}
	for _, match := range matches {
		skill, problem := readSkillDocument(match, root, slug)
		if problem != nil {
			problems = append(problems, *problem)
			continue
		}
		skills = append(skills, skill)
	}
	if len(matches) == 0 {
		return nil, problems, invalidf(
			"the glob %q matched no SKILL.md in the repository", glob,
		)
	}
	if len(skills) == 0 {
		return nil, problems, invalidf(
			"none of the %d matched SKILL.md files is a valid skill", len(matches),
		)
	}
	return skills, problems, nil
}

func matchSkillGlob(root, glob string) ([]string, error) {
	matches := []string{}
	err := filepath.WalkDir(root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {

			if p != root && entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		matched, matchErr := path.Match(glob, filepath.ToSlash(rel))
		if matchErr != nil {
			return matchErr
		}
		if matched {
			matches = append(matches, p)
		}
		return nil
	})
	return matches, err
}

func readSkillDocument(
	filePath, root, slug string,
) (discoveredSkill, *skillProblem) {
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		return discoveredSkill{}, &skillProblem{Path: filePath, Message: "unreadable"}
	}
	relPath := filepath.ToSlash(rel)
	problem := func(message string) (discoveredSkill, *skillProblem) {
		return discoveredSkill{}, &skillProblem{Path: relPath, Message: message}
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return problem("the file could not be read")
	}
	if len(data) > maxSkillFileBytes {
		return problem(fmt.Sprintf("SKILL.md is larger than %d KiB", maxSkillFileBytes>>10))
	}
	front, body, present, err := splitFrontmatter(string(data))
	if err != nil {
		return problem("the frontmatter is not valid YAML: " + err.Error())
	}
	if !present {
		return problem("SKILL.md has no frontmatter")
	}
	rawName := strings.TrimSpace(front.Name)
	if rawName == "" {
		rawName = filepath.Base(filepath.Dir(filePath))
	}
	if err := validateRawSkillName(rawName); err != nil {
		return problem(err.Error())
	}
	description := strings.TrimSpace(front.Description)
	if description == "" {
		return problem("the frontmatter has no description")
	}
	if len([]rune(description)) > maxSkillDescriptionLength {
		return problem(fmt.Sprintf(
			"the description exceeds %d characters", maxSkillDescriptionLength,
		))
	}
	name := slug + "-" + rawName
	if len(name) > maxSkillNameLength {
		return problem(fmt.Sprintf(
			"the prefixed name %q exceeds %d characters; shorten the source slug or the skill name",
			name, maxSkillNameLength,
		))
	}
	files, err := collectSkillFiles(filepath.Dir(filePath))
	if err != nil {
		return problem(err.Error())
	}
	return discoveredSkill{
		Path:                   relPath,
		RawName:                rawName,
		Name:                   name,
		Description:            description,
		Body:                   body,
		Files:                  files,
		DisableModelInvocation: front.DisableModelInvocation,
	}, nil
}

type skillFrontmatter struct {
	Name                   string `yaml:"name"`
	Description            string `yaml:"description"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation"`
}

func splitFrontmatter(document string) (skillFrontmatter, string, bool, error) {
	normalized := strings.ReplaceAll(document, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return skillFrontmatter{}, "", false, nil
	}
	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	trailing := "\n---\n"
	if end < 0 && strings.HasSuffix(rest, "\n---") {
		end = len(rest) - len("\n---")
		trailing = "\n---"
	}
	if end < 0 {
		return skillFrontmatter{}, "", false, nil
	}
	header := rest[:end]
	body := strings.TrimLeft(rest[end+len(trailing):], "\n")
	var front skillFrontmatter
	if err := yaml.Unmarshal([]byte(header), &front); err != nil {
		return skillFrontmatter{}, "", true, err
	}
	return front, strings.TrimRight(body, "\n") + "\n", true, nil
}

func validateRawSkillName(name string) error {
	if len(name) > maxSkillNameLength {
		return fmt.Errorf("the name exceeds %d characters", maxSkillNameLength)
	}
	if !skillNamePattern.MatchString(name) {
		return fmt.Errorf(
			"the name %q must be lowercase a-z, 0-9 and single hyphens", name,
		)
	}
	return nil
}

func collectSkillFiles(dir string) ([]SkillFile, error) {
	files := []SkillFile{}
	total := 0
	err := filepath.WalkDir(dir, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if p != dir && entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if p == dir {
			return nil
		}
		if entry.Name() == "SKILL.md" && filepath.Dir(p) == dir {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.Size() > maxSkillFileBytes {
			return fmt.Errorf("the file %s is larger than %d KiB",
				filepath.ToSlash(rel), maxSkillFileBytes>>10)
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		total += len(data)
		if total > maxSkillFilesBytes {
			return fmt.Errorf("the skill's files exceed %d MiB", maxSkillFilesBytes>>20)
		}
		if len(files) >= maxSkillFiles {
			return fmt.Errorf("the skill has more than %d files", maxSkillFiles)
		}
		files = append(files, SkillFile{Path: filepath.ToSlash(rel), Text: string(data)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func sourceSkillBudget(skills []discoveredSkill) error {
	total := 0
	for _, skill := range skills {
		total += len(skill.Body) + len(skill.Path) + len(skill.Description)
		for _, file := range skill.Files {
			total += len(file.Path) + len(file.Text)
		}
		if total > maxPendingBytes {
			return invalidf(
				"the fetched skills exceed %d MiB; narrow the glob", maxPendingBytes>>20,
			)
		}
	}
	return nil
}
