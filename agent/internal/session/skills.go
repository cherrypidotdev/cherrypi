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

package session

import (
	"github.com/sky-valley/pi/coding"
)

func hostSkills(workspace string) ([]coding.Skill, []coding.SkillDiagnostic) {
	return coding.LoadSkillsWithTrust(workspace, false)
}

func promptSkillNames(skills []coding.Skill) []string {
	return collectSkillNames(skills, false)
}

// line can answer "why is the skill I installed not offered?".
func manualSkillNames(skills []coding.Skill) []string {
	return collectSkillNames(skills, true)
}

func collectSkillNames(skills []coding.Skill, manual bool) []string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		if skill.DisableModelInvocation == manual {
			names = append(names, skill.Name)
		}
	}
	return names
}
