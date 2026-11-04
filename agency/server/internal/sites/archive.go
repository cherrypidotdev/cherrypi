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

package sites

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const markerName = ".cherrypi-site"

func validateArchive(filePath string, maxBytes int64) error {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidArchive, err)
	}
	defer func() { _ = reader.Close() }()

	hasIndex := false
	var total uint64
	for _, entry := range reader.File {
		name, err := safeEntryName(entry.Name)
		if err != nil {
			return err
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlinks are not allowed (%s)", ErrInvalidArchive, entry.Name)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if !entry.Mode().IsRegular() {
			return fmt.Errorf("%w: non-regular file (%s)", ErrInvalidArchive, entry.Name)
		}
		if name == "index.html" {
			hasIndex = true
		}
		total += entry.UncompressedSize64
		if maxBytes > 0 && total > uint64(maxBytes) {
			return ErrTooLarge
		}
	}
	if !hasIndex {
		return fmt.Errorf("%w: index.html is required at the archive root", ErrInvalidArchive)
	}
	return nil
}

func (s *Service) extractToCache(_ context.Context, attachmentID int64, zipPath string) error {
	if s.cfg.CacheDir == "" {
		return nil
	}
	if attachmentID <= 0 {
		return fmt.Errorf("%w: invalid attachment", ErrInvalidArchive)
	}
	if err := validateArchive(zipPath, s.cfg.MaxBytes); err != nil {
		return err
	}

	if err := os.MkdirAll(s.cfg.CacheDir, 0o755); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(s.cfg.CacheDir, ".extract-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidArchive, err)
	}
	defer func() { _ = reader.Close() }()

	var written uint64
	for _, entry := range reader.File {
		name, err := safeEntryName(entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(
				filepath.Join(tempDir, filepath.FromSlash(name)),
				0o755,
			); err != nil {
				return err
			}
			continue
		}

		finalPath := filepath.Join(tempDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		target, err := os.OpenFile(finalPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if err != nil {
			_ = source.Close()
			return err
		}
		limit := int64(entry.UncompressedSize64) + 1
		if s.cfg.MaxBytes > 0 {
			remaining := s.cfg.MaxBytes - int64(written)
			if remaining < limit {
				limit = remaining + 1
			}
		}
		copied, copyErr := io.Copy(target, io.LimitReader(source, limit))
		closeErr := target.Close()
		_ = source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if s.cfg.MaxBytes > 0 && copied > s.cfg.MaxBytes-int64(written) {
			return ErrTooLarge
		}
		written += uint64(copied)
	}

	marker := filepath.Join(tempDir, markerName)
	content := strconv.FormatInt(attachmentID, 10) + "\n"
	if err := os.WriteFile(marker, []byte(content), 0o644); err != nil {
		return err
	}

	finalDir := s.cacheDir(attachmentID)
	_ = os.RemoveAll(finalDir)
	if err := os.Rename(tempDir, finalDir); err != nil {
		return err
	}
	return nil
}

func safeEntryName(raw string) (string, error) {
	if raw == "" || strings.ContainsRune(raw, 0) || strings.Contains(raw, "\\") {
		return "", fmt.Errorf("%w: unsafe entry %q", ErrInvalidArchive, raw)
	}
	cleaned := path.Clean(raw)
	if cleaned == "." || cleaned == "/" || path.IsAbs(cleaned) ||
		cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: unsafe entry %q", ErrInvalidArchive, raw)
	}
	return cleaned, nil
}

const maxContentWarnings = 8

func contentWarnings(zipPath, siteID string, maxBytes int64) []string {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil
	}
	defer func() { _ = reader.Close() }()

	files := make(map[string]struct{}, len(reader.File))
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if name, nameErr := safeEntryName(entry.Name); nameErr == nil {
			files[name] = struct{}{}
		}
	}

	seen := map[string]struct{}{}
	var warnings []string
	record := func(document, reference string) {
		key := document + "\x00" + reference
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if len(warnings) >= maxContentWarnings {
			return
		}
		warnings = append(warnings, fmt.Sprintf(
			"%s references %s, which is not in the archive. Root-absolute "+
				"paths are resolved against the agency origin instead of the "+
				"site; build with a relative base (Vite: base: \"./\").",
			document, reference,
		))
	}
	missing := func(reference string) bool {
		if !strings.HasPrefix(reference, "/") || strings.HasPrefix(reference, "//") {
			return false
		}
		pathPart, _ := splitURLSuffix(reference)
		if pathPart == "" {
			return false
		}
		if strings.HasPrefix(pathPart, ContentURLFor(siteID)) {
			return false
		}
		for _, agencyPrefix := range []string{"/api/", "/sites/", "/attachments/"} {
			if strings.HasPrefix(pathPart, agencyPrefix) {
				return false
			}
		}
		decoded, decodeErr := url.PathUnescape(pathPart)
		if decodeErr != nil {
			return false
		}
		return !archiveHasPath(files, decoded)
	}

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name, nameErr := safeEntryName(entry.Name)
		if nameErr != nil {
			continue
		}
		lower := strings.ToLower(name)
		isHTML := strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
		isCSS := strings.HasSuffix(lower, ".css")
		if !isHTML && !isCSS {
			continue
		}
		limit := maxBytes
		if limit <= 0 {
			limit = 8 << 20
		}
		if int64(entry.UncompressedSize64) > limit {
			continue
		}
		source, openErr := entry.Open()
		if openErr != nil {
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(source, limit+1))
		_ = source.Close()
		if readErr != nil {
			continue
		}
		mapper := func(kind urlKind, value string) string {

			if kind != urlKindHref && missing(value) {
				record(name, value)
			}
			return ""
		}
		if isHTML {
			_ = mapHTMLURLs(body, mapper)
		} else {
			_ = mapCSSURLs(body, mapper)
		}
	}
	return warnings
}

func archiveHasPath(files map[string]struct{}, urlPath string) bool {
	cleaned := path.Clean("/" + urlPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || strings.HasSuffix(urlPath, "/") {
		if cleaned == "" {
			return archiveHasFile(files, "index.html")
		}
		return archiveHasFile(files, cleaned+"/index.html")
	}
	return archiveHasFile(files, cleaned)
}

func archiveHasFile(files map[string]struct{}, name string) bool {
	_, ok := files[name]
	return ok
}
