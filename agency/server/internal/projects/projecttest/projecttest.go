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

package projecttest

import (
	"context"
	"testing"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/projects"
)

func AddAgent(
	t *testing.T,
	svc *projects.Service,
	projectID string,
	agentID ids.AgentID,
	role string,
) projects.ProjectAgent {
	t.Helper()
	member, err := svc.UpdateAgentPermissions(
		context.Background(),
		projectID,
		agentID,
		projects.PermissionsForRole(role),
	)
	if err != nil {
		t.Fatalf("grant %s on project %s to agent %d: %v", role, projectID, agentID, err)
	}
	return member
}

func AddMemberAgent(
	t *testing.T,
	svc *projects.Service,
	projectID string,
	agentID ids.AgentID,
) projects.ProjectAgent {
	t.Helper()
	return AddAgent(t, svc, projectID, agentID, projects.RoleMember)
}
