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

type PutTaskSessionStateParams struct {
	TaskID           string
	ExpectedRevision int64
	ModelProvider    sql.NullString
	ModelID          sql.NullString
	ThinkingLevel    sql.NullString
	EntriesJSON      string
	UpdatedAt        int64
}

type TaskTransition string

const (
	TaskQueue  TaskTransition = "queue"
	TaskWait   TaskTransition = "block"
	TaskUnwait TaskTransition = "unblock"

	TaskWakeWaiting TaskTransition = "wake_waiting"
	TaskDequeue     TaskTransition = "dequeue"
	TaskArchive     TaskTransition = "archive"
	TaskUnarchive   TaskTransition = "unarchive"
	TaskSettle      TaskTransition = "settle"

	TaskReleaseSettle     TaskTransition = "release_settle"
	TaskReleaseQueue      TaskTransition = "release_queue"
	TaskInput             TaskTransition = "input"
	TaskClearInput        TaskTransition = "clear_input"
	TaskResume            TaskTransition = "resume"
	TaskComplete          TaskTransition = "complete"
	TaskCancel            TaskTransition = "cancel"
	TaskReopen            TaskTransition = "reopen"
	TaskClaim             TaskTransition = "claim"
	TaskFail              TaskTransition = "fail"
	TaskFailResting       TaskTransition = "fail_resting"
	TaskFailBySession     TaskTransition = "fail_by_session"
	TaskRequeue           TaskTransition = "requeue"
	TaskRequeueWithPrompt TaskTransition = "requeue_with_prompt"
)

type TransitionTaskParams struct {
	Event TaskTransition
	ID    string

	UpdatedAt  int64
	SettledAt  sql.NullInt64
	QueuedAt   sql.NullInt64
	StartedAt  sql.NullInt64
	ArchivedAt sql.NullInt64

	NotBefore sql.NullInt64

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
