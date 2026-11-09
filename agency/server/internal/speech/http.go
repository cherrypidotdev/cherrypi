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

package speech

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	Service *Service
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != TranscribePath {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxAudioBytes+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "the recording is too long")
			return
		}
		writeError(w, http.StatusBadRequest, "expected a multipart recording")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing the recording")
		return
	}
	defer func() { _ = file.Close() }()

	audio, err := io.ReadAll(io.LimitReader(file, MaxAudioBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the recording")
		return
	}
	if len(audio) > MaxAudioBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "the recording is too long")
		return
	}

	filename := header.Filename
	if strings.TrimSpace(filename) == "" {
		filename = "dictation.webm"
	}
	text, err := h.Service.Transcribe(
		r.Context(),
		filename,
		header.Header.Get("Content-Type"),
		audio,
	)
	if err != nil {
		writeTranscribeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
}

func writeTranscribeError(w http.ResponseWriter, err error) {
	var provider *ProviderError
	switch {
	case errors.Is(err, ErrDisabled):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, ErrEmptyAudio):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.As(err, &provider):
		status := http.StatusBadGateway
		if provider.Status == http.StatusUnauthorized ||
			provider.Status == http.StatusForbidden {

			status = http.StatusServiceUnavailable
		}
		writeError(w, status, provider.Error())
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "the speech provider timed out")
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
