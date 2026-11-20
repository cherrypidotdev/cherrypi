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
	etags := newFileETags()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == docsPrefix {
			http.Redirect(w, r, docsPrefix+"/", http.StatusMovedPermanently)
			return
		}

		rel := strings.TrimPrefix(r.URL.Path, docsPrefix+"/")
		rel = strings.TrimPrefix(path.Clean("/"+rel), "/")

		file, ok := resolveDocsFile(docsFs, rel)
		if !ok {
			serveDocsNotFound(w, r, docsFs, etags)
			return
		}

		cacheControl, immutable := staticCachePolicy(file)
		var etagStore *fileETags
		if !immutable {
			etagStore = etags
		}

		contentType := ""
		switch path.Ext(file) {
		case ".md":
			contentType = "text/markdown; charset=utf-8"
		case ".txt":
			contentType = "text/plain; charset=utf-8"
		}

		if !serveEmbeddedFile(w, r, docsFs, file, contentType, cacheControl, etagStore) {
			http.NotFound(w, r)
		}
	})
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

func serveDocsNotFound(w http.ResponseWriter, r *http.Request, docsFs fs.FS, etags *fileETags) {
	withStatus := &statusOverrideWriter{ResponseWriter: w, status: http.StatusNotFound}
	if !serveEmbeddedFile(
		withStatus,
		r,
		docsFs,
		"404.html",
		"text/html; charset=utf-8",
		staticCacheRevalidate,
		etags,
	) {
		http.NotFound(w, r)
	}
}
