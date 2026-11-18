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

package plugins

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxIconBytes = 64 << 10

	IconRoutePrefix = "/api/rest/plugins"

	iconCacheSeconds = 31536000
)

var iconContentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".svg":  "image/svg+xml",
}

func iconContentType(name string, body []byte) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	want, ok := iconContentTypes[ext]
	if !ok || len(body) == 0 {
		return "", false
	}
	if want == "image/svg+xml" {

		head := strings.ToLower(strings.TrimSpace(string(trimBOM(body))))
		if !strings.HasPrefix(head, "<svg") && !strings.HasPrefix(head, "<?xml") {
			return "", false
		}
		return want, true
	}

	if want == "image/avif" {
		return want, isAVIF(body)
	}
	if http.DetectContentType(body) != want {
		return "", false
	}
	return want, true
}

func isAVIF(body []byte) bool {
	if len(body) < 12 || string(body[4:8]) != "ftyp" {
		return false
	}
	brand := string(body[8:12])
	if brand == "avif" || brand == "avis" {
		return true
	}

	boxSize := int(uint32(body[0])<<24 | uint32(body[1])<<16 |
		uint32(body[2])<<8 | uint32(body[3]))
	if boxSize <= 16 || boxSize > len(body) {
		boxSize = len(body)
	}
	for i := 16; i+4 <= boxSize; i += 4 {
		compatible := string(body[i : i+4])
		if compatible == "avif" || compatible == "avis" {
			return true
		}
	}
	return false
}

func trimBOM(body []byte) []byte {
	return []byte(strings.TrimPrefix(string(body), "\ufeff"))
}

func validateBundleIcon(files map[string][]byte, m Manifest) error {
	if m.Icon == "" {
		return nil
	}
	body, ok := files[m.Icon]
	if !ok {
		return fmt.Errorf("bundle needs its icon %s", m.Icon)
	}
	if len(body) > MaxIconBytes {
		return fmt.Errorf("icon %s exceeds %d bytes", m.Icon, MaxIconBytes)
	}
	if _, ok := iconContentType(m.Icon, body); !ok {
		return fmt.Errorf(
			"icon %s must be a PNG, JPEG, GIF, WebP, AVIF, or SVG image", m.Icon,
		)
	}
	return nil
}

func (s *Service) Icon(ctx context.Context, id string) ([]byte, string, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if p.Manifest.Icon == "" {
		return nil, "", ErrNotFound
	}
	path := filepath.Join(
		s.Config.DataDir,
		"plugins",
		p.ID,
		"versions",
		p.Version,
		filepath.FromSlash(p.Manifest.Icon),
	)
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read plugin icon: %w", err)
	}
	contentType, ok := iconContentType(p.Manifest.Icon, body)
	if !ok {
		return nil, "", errors.New("the stored icon is not a served image")
	}
	return body, contentType, nil
}

func IconURL(p Plugin) string {
	if p.Manifest.Icon == "" {
		return ""
	}
	version := p.Checksum
	if len(version) > 12 {
		version = version[:12]
	}
	if version == "" {
		version = p.Version
	}
	return IconRoutePrefix + "/" + p.ID + "/icon?v=" + version
}

func IconDataURL(m Manifest, files map[string][]byte) string {
	if m.Icon == "" {
		return ""
	}
	body, ok := files[m.Icon]
	if !ok || len(body) > MaxIconBytes {
		return ""
	}
	contentType, ok := iconContentType(m.Icon, body)
	if !ok {
		return ""
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(body)
}

type IconHandler struct {
	Service *Service
}

func (h *IconHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, IconRoutePrefix)
	rest = strings.TrimPrefix(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "icon" || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	body, contentType, err := h.Service.Icon(r.Context(), parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; sandbox",
	)
	w.Header().Set(
		"Cache-Control",
		fmt.Sprintf("public, max-age=%d, immutable", iconCacheSeconds),
	)
	_, _ = w.Write(body)
}

func (s *Service) PluginIconURL(ctx context.Context, id string) string {
	if s == nil || s.Store == nil || id == "" {
		return ""
	}
	p, err := s.Get(ctx, id)
	if err != nil {
		return ""
	}
	return IconURL(p)
}
