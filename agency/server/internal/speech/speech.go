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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/settings"
)

const Prefix = "/api/rest/speech"

const TranscribePath = Prefix + "/transcribe"

const MaxAudioBytes = 25 << 20

var ErrDisabled = errors.New("server transcription is not configured")

var ErrEmptyAudio = errors.New("the recording is empty")

type ProviderError struct {
	Status int
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("the speech provider rejected the request (HTTP %d)", e.Status)
}

type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

type Service struct {
	Settings *settings.Service

	Client *http.Client
}

func New(settingsSvc *settings.Service) *Service {
	return &Service{Settings: settingsSvc}
}

func (s *Service) Config(ctx context.Context) (Config, error) {
	if s == nil || s.Settings == nil {
		return Config{}, ErrDisabled
	}
	cur, err := s.Settings.Get(ctx)
	if err != nil {
		return Config{}, err
	}
	if !cur.ServerTranscription() {
		return Config{}, ErrDisabled
	}
	return Config{
		BaseURL: cur.STTBaseURL,
		Model:   cur.STTModel,
		APIKey:  cur.STTAPIKey,
	}, nil
}

func (s *Service) Transcribe(
	ctx context.Context,
	filename, contentType string,
	audio []byte,
) (string, error) {
	cfg, err := s.Config(ctx)
	if err != nil {
		return "", err
	}
	if len(audio) == 0 {
		return "", ErrEmptyAudio
	}
	if len(audio) > MaxAudioBytes {
		return "", fmt.Errorf("recording exceeds %d bytes", MaxAudioBytes)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set(
		"Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename=%q`, filename),
	)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	_ = writer.WriteField("model", cfg.Model)

	_ = writer.WriteField("response_format", "json")
	if err := writer.Close(); err != nil {
		return "", err
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, url, bytes.NewReader(body.Bytes()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logProviderError(cfg, resp.StatusCode, payload)
		return "", &ProviderError{Status: resp.StatusCode}
	}

	text := parseTranscript(payload)
	if strings.TrimSpace(text) == "" {
		return "", errors.New("the speech provider returned an empty transcript")
	}
	return strings.TrimSpace(text), nil
}

func parseTranscript(payload []byte) string {
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(payload, &parsed); err == nil {
		return parsed.Text
	}
	return string(payload)
}

// shortened, single-line and redacted: a gateway that echoes the request would

func logProviderError(cfg Config, status int, payload []byte) {
	message := strings.Join(strings.Fields(string(payload)), " ")
	if len(message) > 300 {
		message = message[:300] + "…"
	}
	log.Printf(
		"Speech provider %s rejected the request (HTTP %d): %s",
		cfg.BaseURL,
		status,
		redactSecrets(message, cfg.APIKey),
	)
}

func redactSecrets(message, apiKey string) string {
	if apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "[redacted]")
	}
	return bearerPattern.ReplaceAllString(message, "Bearer [redacted]")
}

var bearerPattern = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`)
