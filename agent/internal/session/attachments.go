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

// maxAttachmentBytes caps one downloaded attachment. The agency enforces its
// own cap; this is a defensive limit so a misbehaving server cannot fill the
// disk.
const maxAttachmentBytes = 256 * 1024 * 1024

// Attachment is one file the agency attached to a prompt for the agent to
// pull. The agent joins URL with its configured agency URL and authenticates
// with its tunnel credential.
type Attachment struct {
	ID          int64
	Filename    string
	ContentType string
	Size        int64
	URL         string
}

// imageAttachment is an image handed to Pi alongside the prompt text so the
// model can see it, in addition to the file written to the workspace.
type imageAttachment struct {
	data     []byte
	mimeType string
}

// materializeAttachments downloads every attachment into the workspace's
// `attachments/` directory and returns the prompt's files (id, final filename,
// on-disk path) plus the images to hand to Pi. The paths go into the prompt
// envelope so the model can open each file; the console reads the ids and
// filenames off the parsed message.
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

// uniqueFilename returns filename, suffixed with `-<n>` before its extension
// until it does not collide with an existing file in dir.
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

// sanitizeAttachmentFilename strips any directory component and control
// characters, plus the brackets and parens the trailer's markdown link uses,
// from a downloaded attachment's name.
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

// isImageContentType reports whether Pi should receive the file as image
// content. Only raster formats Pi's vision supports.
func isImageContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch mediaType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
