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

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

var (
	ErrCommentNotFound = errors.New("comment not found")

	ErrCommentBodyRequired = errors.New("comment body is required")

	ErrCommentForbidden = errors.New("only the author or an administrator may change this comment")

	ErrCommentAttachmentNotFound = errors.New("comment attachment not found")
)

type Comment struct {
	ID            int64
	TaskID        string
	AuthorActorID ids.ActorID

	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time

	EditedAt      time.Time
	DeletedAt     time.Time
	DeletedByID   ids.ActorID
	AttachmentIDs []int64

	MentionedActorIDs []ids.ActorID
}

func (c Comment) Edited() bool { return !c.EditedAt.IsZero() }

func (c Comment) Deleted() bool { return !c.DeletedAt.IsZero() }

func (s *Service) ListComments(
	ctx context.Context,
	taskID string,
	actorID ids.ActorID,
) ([]Comment, int, error) {
	rows, err := s.store.ListTaskComments(ctx, taskID)
	if err != nil {
		return nil, 0, err
	}
	attachments, err := s.commentAttachmentIDs(ctx, taskID)
	if err != nil {
		return nil, 0, err
	}

	lastRead := int64(0)
	if actorID != 0 {
		lastRead, err = s.store.GetTaskCommentRead(ctx, store.GetTaskCommentReadParams{
			TaskID:  taskID,
			ActorID: int64(actorID),
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, err
		}
		if errors.Is(err, sql.ErrNoRows) {
			lastRead = 0
		}
	}

	out := make([]Comment, 0, len(rows))
	maxID := int64(0)
	unread := 0
	for _, row := range rows {
		if row.ID > maxID {
			maxID = row.ID
		}
		comment := toComment(row, taskID, attachments[row.ID])
		if actorID != 0 &&
			!comment.Deleted() &&
			comment.AuthorActorID != actorID &&
			row.ID > lastRead {
			unread++
		}
		out = append(out, comment)
	}

	if actorID != 0 && maxID > lastRead {
		if err := s.store.UpsertTaskCommentRead(ctx, store.UpsertTaskCommentReadParams{
			TaskID:            taskID,
			ActorID:           int64(actorID),
			LastReadCommentID: maxID,
			ReadAt:            time.Now().UnixMilli(),
		}); err != nil {
			return nil, 0, err
		}
	}

	return out, unread, nil
}

func (s *Service) AddComment(
	ctx context.Context,
	taskID string,
	actorID ids.ActorID,
	body string,
	attachmentIDs []int64,
) (Comment, error) {
	if actorID == 0 {
		return Comment{}, ErrActorRequired
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return Comment{}, ErrCommentBodyRequired
	}
	attachments, err := s.validateCommentAttachments(ctx, attachmentIDs)
	if err != nil {
		return Comment{}, err
	}

	task, err := s.Get(ctx, taskID)
	if err != nil {
		return Comment{}, err
	}
	mentioned := s.MentionedActors(ctx, task, body, actorID)

	now := time.Now().UnixMilli()
	row, err := s.store.CreateTaskComment(ctx, store.CreateTaskCommentParams{
		TaskID:        taskID,
		AuthorActorID: nullID(actorID),
		Body:          body,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return Comment{}, err
	}
	for position, attachmentID := range attachments {
		if err := s.store.LinkTaskCommentAttachment(
			ctx,
			store.LinkTaskCommentAttachmentParams{
				CommentID:    row.ID,
				AttachmentID: attachmentID,
				Position:     int64(position),
			},
		); err != nil {
			return Comment{}, err
		}
	}

	s.recordCommentLog(ctx, taskID, "added", "Comment added", row.ID)
	s.changed(ctx, taskID)
	comment := toComment(row, taskID, attachments)
	comment.MentionedActorIDs = mentioned
	if s.OnComment != nil {
		s.OnComment(ctx, taskID, comment)
	}
	if len(mentioned) > 0 && s.OnMention != nil {
		s.OnMention(ctx, taskID, comment)
	}
	return comment, nil
}

func (s *Service) UpdateComment(
	ctx context.Context,
	commentID int64,
	actorID ids.ActorID,
	admin bool,
	body string,
) (Comment, error) {
	if actorID == 0 {
		return Comment{}, ErrActorRequired
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return Comment{}, ErrCommentBodyRequired
	}
	row, taskID, err := s.loadComment(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	if !admin && ids.ActorID(row.AuthorActorID.Int64) != actorID {
		return Comment{}, ErrCommentForbidden
	}

	task, err := s.Get(ctx, taskID)
	if err != nil {
		return Comment{}, err
	}

	before := s.MentionedActors(ctx, task, row.Body, actorID)

	now := time.Now().UnixMilli()
	if err := s.store.UpdateTaskCommentBody(ctx, store.UpdateTaskCommentBodyParams{
		ID:        commentID,
		Body:      body,
		UpdatedAt: now,
		EditedAt:  sql.NullInt64{Int64: now, Valid: true},
	}); err != nil {
		return Comment{}, err
	}
	updated, err := s.store.GetTaskComment(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	attachments, err := s.commentAttachmentIDs(ctx, taskID)
	if err != nil {
		return Comment{}, err
	}

	s.recordCommentLog(ctx, taskID, "edited", "Comment edited", commentID)
	s.changed(ctx, taskID)
	comment := toComment(updated, taskID, attachments[commentID])
	comment.MentionedActorIDs = addedActors(
		s.MentionedActors(ctx, task, body, actorID),
		before,
	)
	if len(comment.MentionedActorIDs) > 0 && s.OnMention != nil {
		s.OnMention(ctx, taskID, comment)
	}
	return comment, nil
}

func (s *Service) DeleteComment(
	ctx context.Context,
	commentID int64,
	actorID ids.ActorID,
	admin bool,
) (Comment, error) {
	if actorID == 0 {
		return Comment{}, ErrActorRequired
	}
	row, taskID, err := s.loadComment(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	if !admin && ids.ActorID(row.AuthorActorID.Int64) != actorID {
		return Comment{}, ErrCommentForbidden
	}

	if err := s.store.SoftDeleteTaskComment(ctx, store.SoftDeleteTaskCommentParams{
		ID:               commentID,
		DeletedAt:        sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		DeletedByActorID: nullID(actorID),
	}); err != nil {
		return Comment{}, err
	}
	if err := s.store.DeleteTaskCommentAttachments(ctx, commentID); err != nil {
		return Comment{}, err
	}
	deleted, err := s.store.GetTaskComment(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}

	s.recordCommentLog(ctx, taskID, "deleted", "Comment deleted", commentID)
	s.changed(ctx, taskID)
	return toComment(deleted, taskID, nil), nil
}

func (s *Service) CommentTaskID(ctx context.Context, commentID int64) (string, error) {
	row, err := s.store.GetTaskComment(ctx, commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrCommentNotFound
	}
	if err != nil {
		return "", err
	}
	if row.DeletedAt.Valid {
		return "", ErrCommentNotFound
	}
	taskID, err := s.store.GetTaskCommentTaskID(ctx, commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrCommentNotFound
	}
	if err != nil {
		return "", err
	}
	return taskID, nil
}

func (s *Service) UnreadComments(
	ctx context.Context,
	taskID string,
	actorID ids.ActorID,
) (int, error) {
	if actorID == 0 {
		return 0, nil
	}
	count, err := s.store.CountUnreadTaskComments(ctx, store.CountUnreadTaskCommentsParams{
		TaskID:  taskID,
		ActorID: int64(actorID),
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Service) loadComment(
	ctx context.Context,
	commentID int64,
) (store.TaskComment, string, error) {
	row, err := s.store.GetTaskComment(ctx, commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TaskComment{}, "", ErrCommentNotFound
	}
	if err != nil {
		return store.TaskComment{}, "", err
	}
	if row.DeletedAt.Valid {
		return store.TaskComment{}, "", ErrCommentNotFound
	}
	taskID, err := s.store.GetTaskCommentTaskID(ctx, commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TaskComment{}, "", ErrCommentNotFound
	}
	if err != nil {
		return store.TaskComment{}, "", err
	}
	return row, taskID, nil
}

func (s *Service) validateCommentAttachments(
	ctx context.Context,
	attachmentIDs []int64,
) ([]int64, error) {
	out := make([]int64, 0, len(attachmentIDs))
	seen := make(map[int64]bool, len(attachmentIDs))
	for _, id := range attachmentIDs {
		if id <= 0 || seen[id] {
			continue
		}
		if _, err := s.store.GetAttachment(ctx, id); errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %d", ErrCommentAttachmentNotFound, id)
		} else if err != nil {
			return nil, err
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) commentAttachmentIDs(
	ctx context.Context,
	taskID string,
) (map[int64][]int64, error) {
	rows, err := s.store.ListTaskCommentAttachments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]int64, len(rows))
	for _, row := range rows {
		out[row.CommentID] = append(out[row.CommentID], row.AttachmentID)
	}
	for id := range out {
		sort.Slice(out[id], func(i, j int) bool { return out[id][i] < out[id][j] })
	}
	return out, nil
}

func (s *Service) recordCommentLog(
	ctx context.Context,
	taskID, field, message string,
	commentID int64,
) {
	s.recordLog(
		ctx, actorFrom(ctx), time.Now().UnixMilli(),
		taskID, LogComment, field, "", "", message,
		map[string]any{"comment_id": commentID},
	)
}

func addedActors(after, before []ids.ActorID) []ids.ActorID {
	if len(after) == 0 {
		return nil
	}
	seen := make(map[ids.ActorID]bool, len(before))
	for _, id := range before {
		seen[id] = true
	}
	out := make([]ids.ActorID, 0, len(after))
	for _, id := range after {
		if !seen[id] {
			out = append(out, id)
		}
	}
	return out
}

func toComment(row store.TaskComment, taskID string, attachmentIDs []int64) Comment {
	comment := Comment{
		ID:            row.ID,
		TaskID:        taskID,
		AuthorActorID: ids.ActorID(row.AuthorActorID.Int64),
		Body:          row.Body,
		CreatedAt:     time.UnixMilli(row.CreatedAt),
		UpdatedAt:     time.UnixMilli(row.UpdatedAt),
		DeletedByID:   ids.ActorID(row.DeletedByActorID.Int64),
		AttachmentIDs: attachmentIDs,
	}
	if row.EditedAt.Valid {
		comment.EditedAt = time.UnixMilli(row.EditedAt.Int64)
	}
	if row.DeletedAt.Valid {
		comment.DeletedAt = time.UnixMilli(row.DeletedAt.Int64)
		comment.Body = ""
		comment.AttachmentIDs = nil
	}
	return comment
}
