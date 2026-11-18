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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	historySyncInterval = time.Second
	stateSyncInterval   = 30 * time.Second
	historyRetryMin     = time.Second
	historyRetryMax     = 30 * time.Second
)

type SessionEntry struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Type     string `json:"type"`
	Raw      string `json:"raw"`
}

var (
	ErrTaskHistoryNotFound = errors.New("task history not found")

	ErrTaskHistoryUnauthorized = errors.New("task history unauthorized")
)

type TaskHistoryStatusError struct {
	Status  int
	Message string
}

func (e *TaskHistoryStatusError) Error() string {
	return fmt.Sprintf("task history: status %d: %s", e.Status, e.Message)
}

func (e *TaskHistoryStatusError) Is(target error) bool {
	switch target {
	case ErrTaskHistoryNotFound:
		return e.Status == http.StatusNotFound &&
			strings.Contains(strings.ToLower(e.Message), "task not found")
	case ErrTaskHistoryUnauthorized:
		return e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden
	}
	return false
}
