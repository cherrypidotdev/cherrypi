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
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

// docsPrefix is where the embedded VitePress site is mounted. The docs are

const docsPrefix = "/docs"

// serveEmbeddedDocs mounts the VitePress build from docsdist. Release builds

func serveEmbeddedDocs(mux *http.ServeMux) {
	docsFs, err := docsFS()
	if err != nil {
		if !devBuild {
			log.Fatalf("Embedded docs unavailable: %v", err)
		}
		mux.HandleFunc(docsPrefix, http.NotFound)
		mux.HandleFunc(docsPrefix+"/", http.NotFound)
		return
	}

	handler := newDocsHandler(docsFs)
	mux.Handle(docsPrefix, handler)
	mux.Handle(docsPrefix+"/", handler)
}

// serveDocsProxy mounts /docs on a VitePress dev server for `-tags dev` builds.

func serveDocsProxy(mux *http.ServeMux, target *url.URL) {
	proxy := httputil.NewSingleHostReverseProxy(target)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == docsPrefix {
			http.Redirect(w, r, docsPrefix+"/", http.StatusMovedPermanently)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	mux.Handle(docsPrefix, handler)
	mux.Handle(docsPrefix+"/", handler)

	log.Printf("Proxying %s requests to %s", docsPrefix, target)
}

// newDocsHandler serves the built VitePress site. cleanUrls makes VitePress emit

func newDocsHandler(docsFs fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == docsPrefix {
			http.Redirect(w, r, docsPrefix+"/", http.StatusMovedPermanently)
			return
		}

		rel := strings.TrimPrefix(r.URL.Path, docsPrefix+"/")
		rel = strings.TrimPrefix(path.Clean("/"+rel), "/")

		file, ok := resolveDocsFile(docsFs, rel)
		if !ok {
			serveDocsNotFound(w, r, docsFs)
			return
		}

		// VitePress fingerprints the assets/ tree; HTML entry points must stay

		if strings.HasPrefix(file, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		serveDocsFile(w, r, docsFs, file)
	})
}

func serveDocsFile(w http.ResponseWriter, r *http.Request, docsFs fs.FS, file string) {
	opened, err := docsFs.Open(file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = opened.Close() }()

	info, err := opened.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if seeker, ok := opened.(io.ReadSeeker); ok {
		http.ServeContent(w, r, file, info.ModTime(), seeker)
		return
	}

	data, err := io.ReadAll(opened)
	if err != nil {
		http.Error(w, "read docs file", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, file, info.ModTime(), bytes.NewReader(data))
}

func resolveDocsFile(docsFs fs.FS, rel string) (string, bool) {
	if rel == "" || strings.HasSuffix(rel, "/") {
		rel = path.Join(rel, "index.html")
	}

	candidates := []string{rel}
	if !strings.HasSuffix(rel, "/") && path.Ext(rel) == "" {
		candidates = append(candidates, rel+".html", path.Join(rel, "index.html"))
	}

	for _, candidate := range candidates {
		info, err := fs.Stat(docsFs, candidate)
		if err == nil && !info.IsDir() {
			return candidate, true
		}
	}

	return "", false
}

func serveDocsNotFound(w http.ResponseWriter, r *http.Request, docsFs fs.FS) {
	page, err := docsFs.Open("404.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = page.Close() }()

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.Copy(w, page)
}
