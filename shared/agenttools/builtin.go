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

package agenttools

type Audience int

const (
	EveryRun Audience = iota

	InteractiveOnly

	AutomatedOnly
)

type Tool struct {
	Name     string
	Audience Audience
}

var Tools = []Tool{

	{Name: "setTaskUrls", Audience: EveryRun},
	{Name: "setTaskArtifacts", Audience: EveryRun},
	{Name: "uploadAttachment", Audience: EveryRun},
	{Name: "reportTaskProgress", Audience: EveryRun},
	{Name: "publishSite", Audience: EveryRun},
	{Name: "editSite", Audience: EveryRun},
	{Name: "listTaskArtifacts", Audience: EveryRun},
	{Name: "getTaskArtifact", Audience: EveryRun},
	{Name: "fetchTaskArtifact", Audience: EveryRun},
	{Name: "fetchAttachment", Audience: EveryRun},
	{Name: "downloadPublishedSiteZip", Audience: EveryRun},
	{Name: "ask_user_question", Audience: InteractiveOnly},

	{Name: "listTasks", Audience: EveryRun},
	{Name: "getTask", Audience: EveryRun},
	{Name: "getTaskHistory", Audience: EveryRun},
	{Name: "create_backlog_task", Audience: EveryRun},
	{Name: "queue_child_task", Audience: EveryRun},
	{Name: "updateTask", Audience: EveryRun},
	{Name: "queueTask", Audience: EveryRun},
	{Name: "closeOutTask", Audience: EveryRun},
	{Name: "listAgents", Audience: EveryRun},
	{Name: "readAgentLogs", Audience: EveryRun},
	{Name: "getAgencyMetadata", Audience: EveryRun},
	{Name: "readAgencyLogs", Audience: EveryRun},
	{Name: "getPrompt", Audience: EveryRun},
	{Name: "listPromptVersions", Audience: EveryRun},
	{Name: "proposePromptChange", Audience: EveryRun},
	{Name: "listPromptProposals", Audience: EveryRun},
	{Name: "listProjects", Audience: EveryRun},
	{Name: "getTaskInput", Audience: EveryRun},

	{Name: "add_knowledge", Audience: EveryRun},
	{Name: "list_knowledge", Audience: EveryRun},
	{Name: "read_knowledge", Audience: EveryRun},
	{Name: "search_knowledge", Audience: EveryRun},
	{Name: "update_knowledge", Audience: EveryRun},
	{Name: "memory_write", Audience: EveryRun},
	{Name: "memory_read", Audience: EveryRun},
	{Name: "memory_search", Audience: EveryRun},
	{Name: "memory_forget", Audience: EveryRun},
	{Name: "memory_restore", Audience: EveryRun},
	{Name: "scratchpad", Audience: EveryRun},
	{Name: "memory_status", Audience: EveryRun},

	{Name: "respondToTaskInput", Audience: InteractiveOnly},
	{Name: "answer_origin_task_questions", Audience: AutomatedOnly},
	{Name: "complete_origin_task", Audience: AutomatedOnly},
	{Name: "cancel_origin_task", Audience: AutomatedOnly},
}

func Names(interactive bool) []string {
	out := make([]string, 0, len(Tools))
	for _, tool := range Tools {
		switch tool.Audience {
		case InteractiveOnly:
			if !interactive {
				continue
			}
		case AutomatedOnly:
			if interactive {
				continue
			}
		}
		out = append(out, tool.Name)
	}
	return out
}
