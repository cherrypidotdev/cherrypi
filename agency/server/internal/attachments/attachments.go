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

// Package attachments is the agency's file-storage service. It stores object
// bytes in S3 (through internal/storage) and metadata in the `attachments`
// blob registry plus its link tables. The registry has no owner; ownership is
// expressed by `task_attachments` (a task's tray and agent artifacts),
// `message_attachments` (a message's files), and the `picture_attachment_id`
// columns on projects/agents. Blobs that lose every reference are reclaimed by
// GC.
package attachments

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/storage"
	"cherrypi.vladfaust.com/agency/internal/store"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	// ErrDisabled is returned when object storage is not configured.
	ErrDisabled = storage.ErrDisabled
	// ErrTooLarge is returned for a file over the configured byte cap.
	ErrTooLarge = errors.New("attachment exceeds the size limit")
	// ErrNotFound is returned for an unknown attachment id.
	ErrNotFound = errors.New("attachment not found")
)

// Config tunes the service.
type Config struct {
	// KeyPrefix is prepended to every object key (already ending in '/').
	KeyPrefix string
	// MaxBytes caps one uploaded file. Zero means no cap.
	MaxBytes int64
	// MaxPerMessage caps the attachments sent with one message.
	MaxPerMessage int
}

// Attachment is one stored blob, plus the message it was sent with when it is
// a message attachment.
type Attachment struct {
	ID          int64
	Filename    string
	ContentType string
	Size        int64
	StorageKey  string
	CreatedBy   string
	CreatedAt   time.Time
	// MessageID is set when the attachment was sent with a user message.
	MessageID string
}

// Message is one user prompt in a task's console, with its attachments.
type Message struct {
	ID          string
	Text        string
	ActorID     ids.ActorID
	CreatedAt   time.Time
	Attachments []Attachment
}

// URLFor is the agency-proxied content route for an attachment id.
func URLFor(id int64) string {
	return "/api/rest/attachments/" + strconv.FormatInt(id, 10)
}

// ToProto renders an attachment for the API. `url` is the agency-proxied
// content route, which the caller's credential authorizes.
func ToProto(a Attachment) *agencyv1.Attachment {
	return &agencyv1.Attachment{
		Id:          a.ID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		Size:        a.Size,
		Url:         URLFor(a.ID),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		CreatedBy:   a.CreatedBy,
	}
}

// File is one upload. Body must be rewindable if Size is negative; the REST
// handlers always provide a known Size.
type File struct {
	Filename    string
	ContentType string
	Body        io.Reader
	Size        int64
}

// Service stores attachments. It is safe for concurrent use.
type Service struct {
	store   store.Store
	objects storage.Store
	cfg     Config
}

// New builds the service.
func New(st store.Store, objects storage.Store, cfg Config) *Service {
	return &Service{store: st, objects: objects, cfg: cfg}
}

// Enabled reports whether object storage is configured.
func (s *Service) Enabled() bool { return s.objects.Enabled() }

// MaxBytes returns the per-file cap (0 = no cap).
func (s *Service) MaxBytes() int64 { return s.cfg.MaxBytes }

// MaxPerMessage returns the per-message attachment cap (0 = no cap).
func (s *Service) MaxPerMessage() int { return s.cfg.MaxPerMessage }

// Objects exposes the object store for the download handler.
func (s *Service) Objects() storage.Store { return s.objects }

// UploadTaskAttachment stores file and links it to the task's tray.
func (s *Service) UploadTaskAttachment(
	ctx context.Context,
	taskID string,
	file File,
	createdBy string,
) (Attachment, error) {
	row, err := s.storeFile(ctx, file, createdBy)
	if err != nil {
		return Attachment{}, err
	}

	now := time.Now().UnixMilli()
	if err := s.store.LinkTaskAttachment(ctx, store.LinkTaskAttachmentParams{
		TaskID:       taskID,
		AttachmentID: row.ID,
		CreatedAt:    now,
	}); err != nil {
		s.discard(ctx, row)
		return Attachment{}, err
	}

	return fromRow(row, ""), nil
}

// Replace overwrites an existing attachment's bytes in place, keeping its id,
// filename, content type and every link to it. Editable attachments (sketches)
// use it so a re-save updates the same blob instead of orphaning the original.
func (s *Service) Replace(ctx context.Context, id int64, file File) (Attachment, error) {
	if !s.Enabled() {
		return Attachment{}, ErrDisabled
	}
	if s.cfg.MaxBytes > 0 && file.Size > s.cfg.MaxBytes {
		return Attachment{}, ErrTooLarge
	}

	row, err := s.store.GetAttachment(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, err
	}

	if err := s.objects.Put(
		ctx, row.StorageKey, file.Body, row.ContentType, file.Size,
	); err != nil {
		return Attachment{}, err
	}

	size := file.Size
	if size < 0 {
		size = 0
	}
	if err := s.store.SetAttachmentSize(ctx, id, size); err != nil {
		return Attachment{}, err
	}

	row.Size = size
	return fromRow(row, ""), nil
}

// UploadArtifactFile stores an artifact file part's bytes. It does not link
// the blob to the task tray: the artifact part that references it keeps it
// alive, and GC reclaims it if the artifact is never recorded.
func (s *Service) UploadArtifactFile(
	ctx context.Context,
	file File,
	createdBy string,
) (Attachment, error) {
	row, err := s.storeFile(ctx, file, createdBy)
	if err != nil {
		return Attachment{}, err
	}
	return fromRow(row, ""), nil
}

// SetProjectPicture stores file as the project's picture, replacing (and
// deleting) the previous one.
func (s *Service) SetProjectPicture(
	ctx context.Context,
	projectID string,
	file File,
	createdBy string,
) (Attachment, error) {
	previous, hasPrevious := s.projectPicture(ctx, projectID)

	row, err := s.storeFile(ctx, file, createdBy)
	if err != nil {
		return Attachment{}, err
	}

	if err := s.store.SetProjectPicture(ctx, store.SetProjectPictureParams{
		PictureAttachmentID: sql.NullInt64{Int64: row.ID, Valid: true},
		UpdatedAt:           time.Now().UnixMilli(),
		ID:                  projectID,
	}); err != nil {
		s.discard(ctx, row)
		return Attachment{}, err
	}

	if hasPrevious {
		_ = s.Delete(ctx, previous.ID)
	}

	return fromRow(row, ""), nil
}

// SetAgentPicture stores file as the agent's picture, replacing (and deleting)
// the previous one.
func (s *Service) SetAgentPicture(
	ctx context.Context,
	agentID ids.AgentID,
	file File,
	createdBy string,
) (Attachment, error) {
	previous, hasPrevious := s.agentPicture(ctx, agentID)

	row, err := s.storeFile(ctx, file, createdBy)
	if err != nil {
		return Attachment{}, err
	}

	if err := s.store.SetAgentPicture(ctx, store.SetAgentPictureParams{
		PictureAttachmentID: sql.NullInt64{Int64: row.ID, Valid: true},
		UpdatedAt:           time.Now().UnixMilli(),
		AgentID:             int64(agentID),
	}); err != nil {
		s.discard(ctx, row)
		return Attachment{}, err
	}

	if hasPrevious {
		_ = s.Delete(ctx, previous.ID)
	}

	return fromRow(row, ""), nil
}

// SetAgencyLogo stores file as the agency logo, replacing (and deleting) the
// previous one.
func (s *Service) SetAgencyLogo(
	ctx context.Context,
	file File,
	createdBy string,
) (Attachment, error) {
	return s.setAgencyPicture(ctx, settings.KeyAgencyLogoAttachmentID, file, createdBy)
}

// SetAgencyFavicon stores file as the agency favicon, replacing (and deleting)
// the previous one.
func (s *Service) SetAgencyFavicon(
	ctx context.Context,
	file File,
	createdBy string,
) (Attachment, error) {
	return s.setAgencyPicture(ctx, settings.KeyAgencyFaviconAttachmentID, file, createdBy)
}

// ClearAgencyLogo removes the agency logo and deletes its blob.
func (s *Service) ClearAgencyLogo(ctx context.Context) error {
	return s.clearAgencyPicture(ctx, settings.KeyAgencyLogoAttachmentID)
}

// ClearAgencyFavicon removes the agency favicon and deletes its blob.
func (s *Service) ClearAgencyFavicon(ctx context.Context) error {
	return s.clearAgencyPicture(ctx, settings.KeyAgencyFaviconAttachmentID)
}

// AgencyLogo returns the agency logo, if one is set.
func (s *Service) AgencyLogo(ctx context.Context) (Attachment, bool) {
	return s.agencyPicture(ctx, settings.KeyAgencyLogoAttachmentID)
}

// AgencyFavicon returns the agency favicon, if one is set.
func (s *Service) AgencyFavicon(ctx context.Context) (Attachment, bool) {
	return s.agencyPicture(ctx, settings.KeyAgencyFaviconAttachmentID)
}

// ClearProjectPicture removes a project's picture and deletes its blob.
func (s *Service) ClearProjectPicture(ctx context.Context, projectID string) error {
	previous, hasPrevious := s.projectPicture(ctx, projectID)

	if err := s.store.ClearProjectPicture(ctx, store.ClearProjectPictureParams{
		UpdatedAt: time.Now().UnixMilli(),
		ID:        projectID,
	}); err != nil {
		return err
	}

	if hasPrevious {
		return s.Delete(ctx, previous.ID)
	}
	return nil
}

// ClearAgentPicture removes an agent's picture and deletes its blob.
func (s *Service) ClearAgentPicture(ctx context.Context, agentID ids.AgentID) error {
	previous, hasPrevious := s.agentPicture(ctx, agentID)

	if err := s.store.ClearAgentPicture(ctx, store.ClearAgentPictureParams{
		UpdatedAt: time.Now().UnixMilli(),
		AgentID:   int64(agentID),
	}); err != nil {
		return err
	}

	if hasPrevious {
		return s.Delete(ctx, previous.ID)
	}
	return nil
}

// SetUserPicture stores file as the user's picture, replacing (and deleting)
// the previous one.
func (s *Service) SetUserPicture(
	ctx context.Context,
	actorID ids.ActorID,
	file File,
	createdBy string,
) (Attachment, error) {
	previous, hasPrevious := s.userPicture(ctx, actorID)

	row, err := s.storeFile(ctx, file, createdBy)
	if err != nil {
		return Attachment{}, err
	}

	if err := s.store.SetUserPicture(ctx, store.SetUserPictureParams{
		PictureAttachmentID: sql.NullInt64{Int64: row.ID, Valid: true},
		UpdatedAt:           time.Now().UnixMilli(),
		ActorID:             int64(actorID),
	}); err != nil {
		s.discard(ctx, row)
		return Attachment{}, err
	}

	if hasPrevious {
		_ = s.Delete(ctx, previous.ID)
	}

	return fromRow(row, ""), nil
}

// ClearUserPicture removes a user's picture and deletes its blob.
func (s *Service) ClearUserPicture(ctx context.Context, actorID ids.ActorID) error {
	previous, hasPrevious := s.userPicture(ctx, actorID)

	if err := s.store.ClearUserPicture(ctx, store.ClearUserPictureParams{
		UpdatedAt: time.Now().UnixMilli(),
		ActorID:   int64(actorID),
	}); err != nil {
		return err
	}

	if hasPrevious {
		return s.Delete(ctx, previous.ID)
	}
	return nil
}

// setAgencyPicture links a freshly stored blob to the agency as its logo or
// favicon (the setting key names which). The previous blob is deleted.
func (s *Service) setAgencyPicture(
	ctx context.Context,
	key string,
	file File,
	createdBy string,
) (Attachment, error) {
	previous, hasPrevious := s.agencyPictureRow(ctx, key)

	row, err := s.storeFile(ctx, file, createdBy)
	if err != nil {
		return Attachment{}, err
	}

	if err := s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   key,
		Value: strconv.FormatInt(row.ID, 10),
	}); err != nil {
		s.discard(ctx, row)
		return Attachment{}, err
	}

	if hasPrevious {
		_ = s.Delete(ctx, previous.ID)
	}

	return fromRow(row, ""), nil
}

// clearAgencyPicture unlinks the agency's logo or favicon and deletes its blob.
func (s *Service) clearAgencyPicture(ctx context.Context, key string) error {
	previous, hasPrevious := s.agencyPictureRow(ctx, key)

	if err := s.store.UpsertSetting(ctx, store.UpsertSettingParams{
		Key:   key,
		Value: "",
	}); err != nil {
		return err
	}

	if hasPrevious {
		return s.Delete(ctx, previous.ID)
	}
	return nil
}

// agencyPictureRow reads the attachment an agency picture setting points at.
func (s *Service) agencyPictureRow(
	ctx context.Context,
	key string,
) (store.Attachment, bool) {
	value, err := s.store.GetSetting(ctx, key)
	if err != nil {
		return store.Attachment{}, false
	}
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return store.Attachment{}, false
	}
	row, err := s.store.GetAttachment(ctx, id)
	if err != nil {
		return store.Attachment{}, false
	}
	return row, true
}

// agencyPicture resolves an agency picture setting to its stored attachment.
func (s *Service) agencyPicture(ctx context.Context, key string) (Attachment, bool) {
	row, ok := s.agencyPictureRow(ctx, key)
	if !ok {
		return Attachment{}, false
	}
	return fromRow(row, ""), true
}

// Open returns the blob body and metadata for a download.
func (s *Service) Open(ctx context.Context, id int64) (io.ReadCloser, Attachment, error) {
	row, err := s.store.GetAttachment(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, Attachment{}, ErrNotFound
	}
	if err != nil {
		return nil, Attachment{}, err
	}

	body, _, err := s.objects.Get(ctx, row.StorageKey)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, Attachment{}, ErrNotFound
	}
	if err != nil {
		return nil, Attachment{}, err
	}

	return body, fromRow(row, ""), nil
}

// ProjectPictures returns every project's picture, keyed by project id, for
// enriching a project list without a query per row.
func (s *Service) ProjectPictures(ctx context.Context) (map[string]Attachment, error) {
	rows, err := s.store.ListProjectPictures(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Attachment, len(rows))
	for _, row := range rows {
		out[row.OwnerID] = Attachment{
			ID:          row.ID,
			Filename:    row.Filename,
			ContentType: row.ContentType,
			Size:        row.Size,
			StorageKey:  row.StorageKey,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

// AgentPictures returns every agent's picture, keyed by agent id.
func (s *Service) AgentPictures(ctx context.Context) (map[ids.AgentID]Attachment, error) {
	rows, err := s.store.ListAgentPictures(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[ids.AgentID]Attachment, len(rows))
	for _, row := range rows {
		out[ids.AgentID(row.OwnerID)] = Attachment{
			ID:          row.ID,
			Filename:    row.Filename,
			ContentType: row.ContentType,
			Size:        row.Size,
			StorageKey:  row.StorageKey,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

// UserPictures returns every user's picture, keyed by actor id.
func (s *Service) UserPictures(ctx context.Context) (map[ids.ActorID]Attachment, error) {
	rows, err := s.store.ListUserPictures(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[ids.ActorID]Attachment, len(rows))
	for _, row := range rows {
		out[ids.ActorID(row.OwnerID)] = Attachment{
			ID:          row.ID,
			Filename:    row.Filename,
			ContentType: row.ContentType,
			Size:        row.Size,
			StorageKey:  row.StorageKey,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   time.UnixMilli(row.CreatedAt),
		}
	}
	return out, nil
}

// UserPicture returns a user's picture, if it has one.
func (s *Service) UserPicture(ctx context.Context, actorID ids.ActorID) (Attachment, bool) {
	row, ok := s.userPicture(ctx, actorID)
	if !ok {
		return Attachment{}, false
	}
	return fromRow(row, ""), true
}

// ProjectPicture returns a project's picture, if it has one.
func (s *Service) ProjectPicture(ctx context.Context, projectID string) (Attachment, bool) {
	row, ok := s.projectPicture(ctx, projectID)
	if !ok {
		return Attachment{}, false
	}
	return fromRow(row, ""), true
}

// AgentPicture returns an agent's picture, if it has one.
func (s *Service) AgentPicture(ctx context.Context, agentID ids.AgentID) (Attachment, bool) {
	row, ok := s.agentPicture(ctx, agentID)
	if !ok {
		return Attachment{}, false
	}
	return fromRow(row, ""), true
}

// Get returns an attachment's metadata.
func (s *Service) Get(ctx context.Context, id int64) (Attachment, error) {
	row, err := s.store.GetAttachment(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, err
	}
	return fromRow(row, ""), nil
}

// TaskAttachments returns a task's tray, oldest first.
func (s *Service) TaskAttachments(ctx context.Context, taskID string) ([]Attachment, error) {
	rows, err := s.store.ListTaskAttachments(ctx, taskID)
	if err != nil {
		return nil, err
	}

	out := make([]Attachment, 0, len(rows))
	for _, row := range rows {
		out = append(out, Attachment{
			ID:          row.ID,
			Filename:    row.Filename,
			ContentType: row.ContentType,
			Size:        row.Size,
			StorageKey:  row.StorageKey,
			CreatedBy:   row.CreatedBy,
			CreatedAt:   time.UnixMilli(row.CreatedAt),
			MessageID:   row.MessageID.String,
		})
	}
	return out, nil
}

// TaskMessages returns a task's user-prompt log, each message carrying its
// attachments.
func (s *Service) TaskMessages(ctx context.Context, taskID string) ([]Message, error) {
	rows, err := s.store.ListTaskMessages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	atts, err := s.TaskAttachments(ctx, taskID)
	if err != nil {
		return nil, err
	}

	byMessage := map[string][]Attachment{}
	for _, a := range atts {
		if a.MessageID != "" {
			byMessage[a.MessageID] = append(byMessage[a.MessageID], a)
		}
	}

	out := make([]Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, Message{
			ID:          row.ID,
			Text:        row.Text,
			ActorID:     ids.ActorID(row.ActorID.Int64),
			CreatedAt:   time.UnixMilli(row.CreatedAt),
			Attachments: byMessage[row.ID],
		})
	}
	return out, nil
}

// RecordMessage logs a user prompt and links its attachments. It is called by
// the task service when a prompt is accepted.
func (s *Service) RecordMessage(
	ctx context.Context,
	taskID, messageID, text string,
	actorID ids.ActorID,
	attachmentIDs []int64,
	now int64,
) error {
	if err := s.store.CreateTaskMessage(ctx, store.CreateTaskMessageParams{
		ID:        messageID,
		TaskID:    taskID,
		Text:      text,
		ActorID:   nullActorID(actorID),
		CreatedAt: now,
	}); err != nil {
		return err
	}

	for _, attachmentID := range attachmentIDs {
		if err := s.store.LinkMessageAttachment(ctx, store.LinkMessageAttachmentParams{
			MessageID:    messageID,
			AttachmentID: attachmentID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes an attachment: its blob and its row (links cascade).
func (s *Service) Delete(ctx context.Context, id int64) error {
	row, err := s.store.GetAttachment(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if err := s.objects.Delete(ctx, row.StorageKey); err != nil &&
		!errors.Is(err, storage.ErrObjectNotFound) {
		return err
	}
	if err := s.store.DeleteAttachment(ctx, id); err != nil {
		return err
	}

	// A blob deleted by id may have been the agency logo/favicon; blank a
	// setting that still points at it so it does not dangle.
	value := strconv.FormatInt(id, 10)
	for _, key := range []string{
		settings.KeyAgencyLogoAttachmentID,
		settings.KeyAgencyFaviconAttachmentID,
	} {
		if err := s.store.ClearSettingValue(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

// RunGC sweeps orphaned blobs until ctx is cancelled. It is started once by
// the agency; deletes also trigger a sweep, so this only catches stragglers.
func (s *Service) RunGC(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.GC(ctx)
		}
	}
}

// GC deletes every blob that lost its last reference. It is best-effort: one
// failed object delete does not stop the sweep.
func (s *Service) GC(ctx context.Context) error {
	orphans, err := s.store.ListOrphanAttachments(ctx)
	if err != nil {
		return err
	}

	var firstErr error
	for _, row := range orphans {
		if err := s.objects.Delete(ctx, row.StorageKey); err != nil &&
			!errors.Is(err, storage.ErrObjectNotFound) && firstErr == nil {
			firstErr = err
		}
		if err := s.store.DeleteAttachment(ctx, row.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// storeFile inserts the registry row, streams the bytes to S3, and records the
// size. The object key is derived from the rowid, so the row is inserted first
// with a provisional unique key and repointed once the id is known.
func (s *Service) storeFile(
	ctx context.Context,
	file File,
	createdBy string,
) (store.Attachment, error) {
	if !s.Enabled() {
		return store.Attachment{}, ErrDisabled
	}
	if s.cfg.MaxBytes > 0 && file.Size > s.cfg.MaxBytes {
		return store.Attachment{}, ErrTooLarge
	}

	filename := sanitizeFilename(file.Filename)
	contentType := inferContentType(filename, file.ContentType)

	row, err := s.store.CreateAttachment(ctx, store.CreateAttachmentParams{
		StorageKey:  s.cfg.KeyPrefix + "attachments/pending-" + randomToken(),
		Filename:    filename,
		ContentType: contentType,
		Size:        0,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().UnixMilli(),
	})
	if err != nil {
		return store.Attachment{}, err
	}

	key := s.objectKey(row.ID, filename, contentType)
	if err := s.store.SetAttachmentStorageKey(ctx, row.ID, key); err != nil {
		s.discard(ctx, row)
		return store.Attachment{}, err
	}

	if err := s.objects.Put(ctx, key, file.Body, contentType, file.Size); err != nil {
		s.discard(ctx, row)
		return store.Attachment{}, err
	}

	size := file.Size
	if size < 0 {
		size = 0
	}
	if err := s.store.SetAttachmentSize(ctx, row.ID, size); err != nil {
		s.discard(ctx, row)
		return store.Attachment{}, err
	}

	row.StorageKey = key
	row.Size = size
	return row, nil
}

// discard removes a row whose upload failed, and its object if one was
// written.
func (s *Service) discard(ctx context.Context, row store.Attachment) {
	if row.StorageKey != "" && !strings.Contains(row.StorageKey, "/pending-") {
		_ = s.objects.Delete(ctx, row.StorageKey)
	}
	_ = s.store.DeleteAttachment(ctx, row.ID)
}

func (s *Service) projectPicture(
	ctx context.Context,
	projectID string,
) (store.Attachment, bool) {
	row, err := s.store.GetProjectPicture(ctx, projectID)
	if err != nil {
		return store.Attachment{}, false
	}
	return row, true
}

func (s *Service) agentPicture(
	ctx context.Context,
	agentID ids.AgentID,
) (store.Attachment, bool) {
	row, err := s.store.GetAgentPicture(ctx, agentID)
	if err != nil {
		return store.Attachment{}, false
	}
	return row, true
}

func (s *Service) userPicture(
	ctx context.Context,
	actorID ids.ActorID,
) (store.Attachment, bool) {
	row, err := s.store.GetUserPicture(ctx, actorID)
	if err != nil {
		return store.Attachment{}, false
	}
	return row, true
}

// objectKey is `[<prefix>]attachments/<rowid><ext>`: the registry rowid names
// the object and the original extension is preserved. Nothing else about the
// filename is used.
func (s *Service) objectKey(id int64, filename, contentType string) string {
	return s.cfg.KeyPrefix + "attachments/" +
		strconv.FormatInt(id, 10) + extensionFor(filename, contentType)
}

func fromRow(row store.Attachment, messageID string) Attachment {
	return Attachment{
		ID:          row.ID,
		Filename:    row.Filename,
		ContentType: row.ContentType,
		Size:        row.Size,
		StorageKey:  row.StorageKey,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   time.UnixMilli(row.CreatedAt),
		MessageID:   messageID,
	}
}

// sanitizeFilename strips any directory component and control characters.
func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == "/" {
		return "attachment"
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if name == "" {
		return "attachment"
	}
	return name
}

// inferContentType keeps a concrete upload type but fills in an empty or
// generic one from the filename. Agent tools upload a Blob without a type, so
// without this an image would be stored as application/octet-stream and its
// task-detail thumbnail would fall back to a file icon (and the download would
// be an attachment instead of an inline image).
func inferContentType(filename, contentType string) string {
	contentType = strings.TrimSpace(contentType)
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if contentType != "" && mediaType != "application/octet-stream" {
		return contentType
	}
	if byExtension := mime.TypeByExtension(filepath.Ext(filename)); byExtension != "" {
		return byExtension
	}
	return "application/octet-stream"
}

// extensionFor preserves the filename's extension when it is a safe token,
// falling back to the content type's canonical extension.
func extensionFor(filename, contentType string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if validExt(ext) {
		return "." + ext
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if mapped, ok := extensionByContentType[mediaType]; ok {
		return mapped
	}
	return ""
}

func validExt(ext string) bool {
	if len(ext) == 0 || len(ext) > 10 {
		return false
	}
	for _, r := range ext {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

var extensionByContentType = map[string]string{
	"image/png":        ".png",
	"image/jpeg":       ".jpg",
	"image/webp":       ".webp",
	"image/gif":        ".gif",
	"image/svg+xml":    ".svg",
	"application/pdf":  ".pdf",
	"application/zip":  ".zip",
	"application/gzip": ".gz",
	"application/json": ".json",
	"text/plain":       ".txt",
	"text/markdown":    ".md",
	"text/csv":         ".csv",
	"text/html":        ".html",
}

// randomToken returns a short random hex token for a provisional object key.
func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// nullActorID maps a zero actor id onto SQL NULL.
func nullActorID(id ids.ActorID) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0}
}
