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

package attachments

import (
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/auth"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const UploadPath = "/api/rest/attachments"

type Handler struct {
	Service *Service
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == UploadPath:
		switch r.Method {
		case http.MethodPost:
			h.upload(w, r)
		case http.MethodDelete:
			h.clear(w, r)
		default:
			methodNotAllowed(w)
		}
		return
	case strings.HasPrefix(r.URL.Path, UploadPath+"/"):
		id, ok := attachmentID(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.download(w, r, id)
		case http.MethodPut:
			h.replace(w, r, id)
		case http.MethodDelete:
			h.delete(w, r, id)
		default:
			methodNotAllowed(w)
		}
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	if !h.Service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "object storage is not configured")
		return
	}

	actor := actorFrom(r.Context())
	if actor == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	kind := r.URL.Query().Get("kind")
	ownerID := strings.TrimSpace(r.URL.Query().Get("id"))
	switch kind {
	case "task", "agent_picture", "project_picture", "user_picture":
		if ownerID == "" {
			writeError(w, http.StatusBadRequest, "missing id")
			return
		}
	case "artifact":

	case "agency_logo", "agency_favicon":

	default:
		writeError(w, http.StatusBadRequest, "unknown kind")
		return
	}

	maxBytes := h.Service.MaxBytes()
	if maxBytes <= 0 {
		maxBytes = 25 * 1024 * 1024
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1024*1024)

	file, cleanup, err := readFilePart(r, maxBytes)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error())
		return
	}
	defer cleanup()

	ctx := r.Context()
	var attachment Attachment
	switch kind {
	case "task":
		attachment, err = h.Service.UploadTaskAttachment(ctx, ownerID, file, actor)
	case "artifact":
		attachment, err = h.Service.UploadArtifactFile(ctx, file, actor)
	case "agent_picture":
		actorID, convErr := strconv.ParseInt(ownerID, 10, 64)
		if convErr != nil {
			writeError(w, http.StatusBadRequest, "invalid agent id")
			return
		}
		attachment, err = h.Service.SetAgentPicture(ctx, ids.AgentID(actorID), file, actor)
	case "project_picture":
		attachment, err = h.Service.SetProjectPicture(ctx, ownerID, file, actor)
	case "user_picture":
		actorID, convErr := strconv.ParseInt(ownerID, 10, 64)
		if convErr != nil {
			writeError(w, http.StatusBadRequest, "invalid user id")
			return
		}
		identity, ok := auth.IdentityFrom(ctx)
		if !ok || identity.Kind != "user" {
			writeError(w, http.StatusForbidden, "a user identity is required")
			return
		}
		if !identity.IsAdmin() && int64(identity.ActorID) != actorID {
			writeError(w, http.StatusForbidden, "cannot change another user's picture")
			return
		}
		attachment, err = h.Service.SetUserPicture(ctx, ids.ActorID(actorID), file, actor)
	case "agency_logo":
		attachment, err = h.Service.SetAgencyLogo(ctx, file, actor)
	case "agency_favicon":
		attachment, err = h.Service.SetAgencyFavicon(ctx, file, actor)
	}
	if err != nil {
		writeAttachmentError(w, err)
		return
	}

	writeProto(w, http.StatusCreated, ToProto(attachment))
}

func (h *Handler) clear(w http.ResponseWriter, r *http.Request) {
	if !h.Service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "object storage is not configured")
		return
	}
	if actor := actorFrom(r.Context()); actor == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var err error
	switch r.URL.Query().Get("kind") {
	case "agency_logo":
		err = h.Service.ClearAgencyLogo(r.Context())
	case "agency_favicon":
		err = h.Service.ClearAgencyFavicon(r.Context())
	default:
		writeError(w, http.StatusBadRequest, "unknown kind")
		return
	}
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) replace(w http.ResponseWriter, r *http.Request, id int64) {
	if !h.Service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "object storage is not configured")
		return
	}

	if actor := actorFrom(r.Context()); actor == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	maxBytes := h.Service.MaxBytes()
	if maxBytes <= 0 {
		maxBytes = 25 * 1024 * 1024
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1024*1024)

	file, cleanup, err := readFilePart(r, maxBytes)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error())
		return
	}
	defer cleanup()

	attachment, err := h.Service.Replace(r.Context(), id, file)
	if err != nil {
		writeAttachmentError(w, err)
		return
	}

	writeProto(w, http.StatusOK, ToProto(attachment))
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request, id int64) {
	body, attachment, err := h.Service.Open(r.Context(), id)
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	defer func() { _ = body.Close() }()

	contentType := inferContentType(attachment.Filename, attachment.ContentType)
	mediaType := strings.ToLower(strings.TrimSpace(
		strings.Split(contentType, ";")[0],
	))
	disposition := "attachment"
	if inlineImageTypes[mediaType] {
		disposition = "inline"
	}

	etag := `"` + strconv.FormatInt(attachment.ID, 10) + "-" +
		strconv.FormatInt(attachment.Size, 10) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType(
		disposition, map[string]string{"filename": attachment.Filename},
	))
	if attachment.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(attachment.Size, 10))
	}

	_, _ = io.Copy(w, body)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.Service.Delete(r.Context(), id); err != nil {
		writeAttachmentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var inlineImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"image/gif":  true,
}

func readFilePart(r *http.Request, maxBytes int64) (File, func(), error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return File{}, func() {}, errors.New("expected a multipart upload")
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return File{}, func() {}, errors.New("missing file part")
		}
		if err != nil {
			return File{}, func() {}, err
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}

		filename := part.FileName()
		contentType := part.Header.Get("Content-Type")

		tmp, err := os.CreateTemp("", "cherrypi-upload-*")
		if err != nil {
			_ = part.Close()
			return File{}, func() {}, err
		}
		cleanup := func() {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}

		size, err := io.Copy(tmp, io.LimitReader(part, maxBytes+1))
		_ = part.Close()
		if err != nil {
			cleanup()
			return File{}, func() {}, err
		}
		if size > maxBytes {
			cleanup()
			return File{}, func() {}, ErrTooLarge
		}
		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			cleanup()
			return File{}, func() {}, err
		}

		return File{
			Filename:    filename,
			ContentType: contentType,
			Body:        tmp,
			Size:        size,
		}, cleanup, nil
	}
}

func attachmentID(path string) (int64, bool) {
	raw := strings.TrimPrefix(path, UploadPath+"/")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func actorFrom(ctx context.Context) string {

	if identity, ok := auth.IdentityFrom(ctx); ok && identity.Kind == "agent" {
		return actorid.Format(identity.ActorID)
	}
	if username, ok := auth.Username(ctx); ok {
		return username
	}
	if identity, ok := auth.IdentityFrom(ctx); ok {
		return actorid.Format(identity.ActorID)
	}
	if actorID, ok := auth.AgentID(ctx); ok {
		return actorid.Format(actorID)
	}
	return ""
}

func writeAttachmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrDisabled):
		writeError(w, http.StatusServiceUnavailable, "object storage is not configured")
	case errors.Is(err, ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, `{"error":`+strconv.Quote(message)+`}`)
}

func writeProto(w http.ResponseWriter, status int, message *agencyv1.Attachment) {
	body, err := protojson.Marshal(message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
