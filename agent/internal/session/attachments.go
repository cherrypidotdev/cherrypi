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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxAttachmentBytes = 256 * 1024 * 1024

type Attachment struct {
	ID          int64
	Filename    string
	ContentType string
	Size        int64
	URL         string
}

type imageAttachment struct {
	data     []byte
	mimeType string
}

func (m *Manager) materializeAttachments(
	ctx context.Context,
	workspace string,
	attachments []Attachment,
) ([]PromptAttachment, []imageAttachment, error) {
	if len(attachments) == 0 {
		return nil, nil, nil
	}
	if m.cfg.FetchAttachment == nil {
		return nil, nil, fmt.Errorf("attachments are not configured")
	}

	dir := filepath.Join(workspace, "attachments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}

	var files []PromptAttachment
	var images []imageAttachment
	for _, attachment := range attachments {
		body, err := m.cfg.FetchAttachment(ctx, attachment.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("download %s: %w", attachment.Filename, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(body, maxAttachmentBytes+1))
		_ = body.Close()
		if readErr != nil {
			return nil, nil, fmt.Errorf("read %s: %w", attachment.Filename, readErr)
		}
		if int64(len(data)) > maxAttachmentBytes {
			return nil, nil, fmt.Errorf("%s exceeds the size limit", attachment.Filename)
		}

		path := filepath.Join(dir, uniqueFilename(dir, attachment.Filename))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, nil, err
		}
		files = append(files, PromptAttachment{
			ID:       attachment.ID,
			Filename: filepath.Base(path),
			Path:     path,
		})

		if isImageContentType(attachment.ContentType) {
			images = append(images, imageAttachment{data: data, mimeType: attachment.ContentType})
		}
	}

	return files, images, nil
}

func uniqueFilename(dir, filename string) string {
	name := sanitizeAttachmentFilename(filename)
	candidate := name
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d%s", stem, i, ext)
	}
}

func sanitizeAttachmentFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == "/" {
		return "attachment"
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		if r == '(' || r == ')' || r == '[' || r == ']' {
			return -1
		}
		return r
	}, name)
	if name == "" {
		return "attachment"
	}
	return name
}

func isImageContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch mediaType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
