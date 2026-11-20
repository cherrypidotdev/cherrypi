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

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
)

// Vite and VitePress fingerprint everything under assets/, and the app font

const (
	staticCacheImmutable  = "public, max-age=31536000, immutable"
	staticCacheRevalidate = "no-cache"
)

func staticCachePolicy(name string) (cacheControl string, immutable bool) {
	if staticTree(name, "assets") || staticTree(name, "fonts") {
		return staticCacheImmutable, true
	}
	return staticCacheRevalidate, false
}

func staticTree(name, tree string) bool {
	return name == tree || strings.HasPrefix(name, tree+"/")
}

type fileETags struct {
	mu   sync.Mutex
	tags map[string]string
}

func newFileETags() *fileETags {
	return &fileETags{tags: make(map[string]string)}
}

func (c *fileETags) lookup(name string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tag, ok := c.tags[name]
	return tag, ok
}

func (c *fileETags) store(name, tag string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tags[name] = tag
}

func contentETag(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func serveEmbeddedFile(
	w http.ResponseWriter,
	r *http.Request,
	fsys fs.FS,
	name string,
	contentType string,
	cacheControl string,
	etags *fileETags,
) bool {
	file, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	var (
		body io.ReadSeeker
		tag  string
	)
	if etags != nil {
		cached, ok := etags.lookup(name)
		if ok {
			tag = cached
		} else {
			data, readErr := io.ReadAll(file)
			if readErr != nil {
				return false
			}
			tag = contentETag(data)
			etags.store(name, tag)
			body = bytes.NewReader(data)
		}
	}
	if body == nil {
		if seeker, ok := file.(io.ReadSeeker); ok {
			body = seeker
		} else {
			data, readErr := io.ReadAll(file)
			if readErr != nil {
				return false
			}
			body = bytes.NewReader(data)
		}
	}

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", cacheControl)
	if etags != nil {
		w.Header().Set("ETag", tag)
	}
	http.ServeContent(w, r, name, info.ModTime(), body)
	return true
}

type statusOverrideWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusOverrideWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.wrote = true
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		code = w.status
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusOverrideWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
