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

package store

import (
	"database/sql"
	"errors"

	"cherrypi.vladfaust.com/agency/internal/ids"
)

type RegisterAgentParams struct {
	Username     string
	DisplayName  string
	Description  string
	OwnerActorID ids.ActorID

	CreatedAt int64
	UpdatedAt int64
}

type TaskEvent string

const (
	TaskQueue  TaskEvent = "queue"
	TaskWait   TaskEvent = "block"
	TaskUnwait TaskEvent = "unblock"

	TaskWakeWaiting TaskEvent = "wake_waiting"
	TaskDequeue     TaskEvent = "dequeue"
	TaskArchive     TaskEvent = "archive"
	TaskUnarchive   TaskEvent = "unarchive"
	TaskSettle      TaskEvent = "settle"

	TaskReleaseSettle     TaskEvent = "release_settle"
	TaskReleaseQueue      TaskEvent = "release_queue"
	TaskInput             TaskEvent = "input"
	TaskClearInput        TaskEvent = "clear_input"
	TaskResume            TaskEvent = "resume"
	TaskComplete          TaskEvent = "complete"
	TaskCancel            TaskEvent = "cancel"
	TaskReopen            TaskEvent = "reopen"
	TaskClaim             TaskEvent = "claim"
	TaskFail              TaskEvent = "fail"
	TaskFailResting       TaskEvent = "fail_resting"
	TaskFailBySession     TaskEvent = "fail_by_session"
	TaskRequeue           TaskEvent = "requeue"
	TaskRequeueWithPrompt TaskEvent = "requeue_with_prompt"
)

type TransitionTaskParams struct {
	Event TaskEvent
	ID    string

	UpdatedAt  int64
	SettledAt  sql.NullInt64
	QueuedAt   sql.NullInt64
	StartedAt  sql.NullInt64
	ArchivedAt sql.NullInt64

	Error string
	State string

	AssigneeActorID sql.NullInt64
	AssigneePinned  int64

	OrgPromptID          sql.NullInt64
	AgentPromptID        sql.NullInt64
	ProjectPromptID      sql.NullInt64
	AgentProjectPromptID sql.NullInt64

	PendingPrompt        string
	PendingPromptActorID sql.NullInt64

	AgentID   ids.AgentID
	SessionID string
}

var (
	ErrPromptProposalNotFound   = errors.New("prompt proposal not found")
	ErrPromptProposalNotPending = errors.New("prompt proposal is not pending")
	ErrPromptProposalStale      = errors.New("prompt proposal is stale")
)

type ApprovePromptProposalParams struct {
	ID               int64
	DecidedByActorID ids.ActorID
	DecisionNote     string
	AutoApproved     bool
	Now              int64
}
