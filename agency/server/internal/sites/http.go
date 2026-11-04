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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

const Prefix = "/api/rest/sites"

const ContentPrefix = "/sites/"

const contentCookieName = "cherrypi_site_session"

type Handler struct {
	Service  *Service
	Accounts *accounts.Service
	grants   grantSigner

	AllowInsecureHTTP bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, Prefix)
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(rest, "/")
	id := parts[0]
	switch {
	case len(parts) == 2 && parts[1] == "publish" && r.Method == http.MethodPost:
		h.publish(w, r, id)
	case len(parts) == 2 && parts[1] == "token" && r.Method == http.MethodPost:
		h.token(w, r, id)
	case len(parts) == 2 && parts[1] == "view" && r.Method == http.MethodPost:
		h.prepareView(w, r, id)
	case len(parts) == 1 && r.Method == http.MethodPatch:
		h.update(w, r, id)
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.get(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) ServeContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("siteID")
	if err := ValidateID(id); err != nil {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/bridge.js") {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.WriteString(w, bridgeScript)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/content") {
		target := r.URL.Path + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}

	subpath, grant, redirect := h.contentPath(id, r.URL.Path)
	if redirect != "" {
		if r.URL.RawQuery != "" {
			redirect += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, redirect, http.StatusMovedPermanently)
		return
	}
	if grant == "" && h.contentRequiresAuth(r) {
		if _, ok := h.contentIdentity(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	site, err := h.Service.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filePath, err := h.Service.ContentFile(r.Context(), site.ID, subpath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "could not serve site", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		http.Error(w, "could not serve site", http.StatusInternalServerError)
		return
	}

	contentType := contentTypeFor(filePath)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	exists := func(candidate string) bool {
		_, existsErr := h.Service.ContentFileIn(r.Context(), site, candidate)
		return existsErr == nil
	}
	if contentType == "text/html; charset=utf-8" {
		body, readErr := io.ReadAll(file)
		if readErr != nil {
			http.Error(w, "could not serve site", http.StatusInternalServerError)
			return
		}
		body = injectBridge(body, site.ID)
		body = rewriteSiteDocument(body, site.ID, grant, exists)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = w.Write(body)
		return
	}
	if contentType == "text/css; charset=utf-8" {
		body, readErr := io.ReadAll(file)
		if readErr != nil {
			http.Error(w, "could not serve site", http.StatusInternalServerError)
			return
		}
		body = rewriteSiteStylesheet(body, site.ID, grant, exists)
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(
			w, r, path.Base(filePath), info.ModTime(), bytes.NewReader(body),
		)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	http.ServeContent(w, r, path.Base(filePath), info.ModTime(), file)
}

func (h *Handler) contentPath(id, rawPath string) (subpath, grant, redirect string) {
	prefix := "/sites/" + id + "/content/"
	if !strings.HasPrefix(rawPath, prefix) {
		return strings.TrimPrefix(rawPath, prefix), "", ""
	}
	rest := strings.TrimPrefix(rawPath, prefix)
	if rest == "" {
		return "", "", ""
	}
	head, tail, hasSlash := strings.Cut(rest, "/")
	if !h.grants.Valid(id, head) {
		return rest, "", ""
	}
	if !hasSlash {
		return "", "", prefix + head + "/"
	}
	return tail, head, ""
}

func (h *Handler) contentRequiresAuth(r *http.Request) bool {
	if !h.AllowInsecureHTTP {
		return true
	}
	return isSecureRequest(r) || isLocalhostHost(r.Host)
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if proto := strings.TrimSpace(
		strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0],
	); strings.EqualFold(
		proto,
		"https",
	) {
		return true
	}
	for _, part := range strings.Split(r.Header.Get("Forwarded"), ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "proto") &&
			strings.EqualFold(strings.Trim(strings.TrimSpace(value), `"`), "https") {
			return true
		}
	}
	return false
}

func isLocalhostHost(hostport string) bool {
	host := strings.TrimSpace(hostport)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else {
		host = strings.Trim(host, "[]")
	}
	host = strings.ToLower(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func (h *Handler) contentIdentity(r *http.Request) (auth.Identity, bool) {
	for _, name := range []string{accounts.SessionCookieName, contentCookieName} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			if identity, _, err := h.Accounts.ResolveSession(
				r.Context(),
				cookie.Value,
			); err == nil {
				return toAuth(identity), true
			}
		}
	}
	token := ""
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	if token != "" {
		if identity, _, err := h.Accounts.ResolvePAT(r.Context(), token); err == nil {
			return toAuth(identity), true
		}
	}
	return auth.Identity{}, false
}

func toAuth(identity accounts.Identity) auth.Identity {
	return auth.Identity{
		ActorID:  identity.ActorID,
		Kind:     identity.Kind,
		Username: identity.Username,
		Name:     identity.Name,
		Role:     identity.Role,
		Status:   identity.Status,
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, id string) {
	site, err := h.Service.Get(r.Context(), id)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, siteJSON(site))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id string) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeSiteError(w, ErrForbidden)
		return
	}
	var body struct {
		Title              *string `json:"title"`
		Description        *string `json:"description"`
		OnlyCreatorMayEdit *bool   `json:"only_creator_may_edit"`
		AgencyWide         *bool   `json:"agency_wide"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	site, err := h.Service.UpdateSettings(r.Context(), identity, id, SettingsPatch{
		Title:              body.Title,
		Description:        body.Description,
		OnlyCreatorMayEdit: body.OnlyCreatorMayEdit,
		AgencyWide:         body.AgencyWide,
	})
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, siteJSON(site))
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request, id string) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeSiteError(w, ErrForbidden)
		return
	}
	if !h.Service.Enabled() {
		writeSiteError(w, ErrDisabled)
		return
	}
	if err := ValidateID(id); err != nil {
		writeSiteError(w, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.Service.MaxBytes()+2<<20)
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "expected a multipart publish request", http.StatusBadRequest)
		return
	}

	temp, err := os.CreateTemp("", "cherrypi-site-*.zip")
	if err != nil {
		http.Error(w, "could not store the archive", http.StatusInternalServerError)
		return
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	fields := map[string]string{}
	filename := ""
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, "invalid multipart body", http.StatusBadRequest)
			return
		}
		name := part.FormName()
		if name == "file" {
			filename = part.FileName()
			if _, err := io.Copy(temp, io.LimitReader(part, h.Service.MaxBytes()+1)); err != nil {
				_ = part.Close()
				http.Error(w, "could not read the archive", http.StatusBadRequest)
				return
			}
		} else {
			value, _ := io.ReadAll(io.LimitReader(part, 1<<20))
			fields[name] = string(value)
		}
		_ = part.Close()
	}
	if err := temp.Close(); err != nil {
		http.Error(w, "could not store the archive", http.StatusInternalServerError)
		return
	}
	if filename == "" {
		http.Error(w, "a zip archive is required in the file part", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(tempPath)
	if err != nil {
		http.Error(w, "could not read the archive", http.StatusInternalServerError)
		return
	}
	if info.Size() == 0 || (h.Service.MaxBytes() > 0 && info.Size() > h.Service.MaxBytes()) {
		writeSiteError(w, ErrTooLarge)
		return
	}

	in := PublishInput{
		ID:              id,
		ArchivePath:     tempPath,
		ArchiveFilename: filename,
		OriginTaskID:    fields["origin_task_id"],
	}
	if value, ok := fields["title"]; ok {
		in.Title = &value
	}
	if value, ok := fields["description"]; ok {
		in.Description = &value
	}
	if value, ok := fields["only_creator_may_edit"]; ok {
		parsed := value == "true" || value == "1"
		in.OnlyCreatorMayEdit = &parsed
	}
	if value, ok := fields["agency_wide"]; ok {
		parsed := value == "true" || value == "1"
		in.AgencyWide = &parsed
	}

	site, err := h.Service.Publish(r.Context(), identity, in)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, publishJSON(site, tempPath, h.Service.MaxBytes()))
}

// a Vite `base: "./"`) instead of shipping a site whose assets 200 with the

func publishJSON(site Site, archivePath string, maxBytes int64) map[string]any {
	body := siteJSON(site)
	if warnings := contentWarnings(archivePath, site.ID, maxBytes); len(warnings) > 0 {
		body["warnings"] = warnings
	}
	return body
}

func (h *Handler) prepareView(w http.ResponseWriter, r *http.Request, id string) {
	if err := ValidateID(id); err != nil {
		writeSiteError(w, err)
		return
	}
	if _, err := h.Service.Get(r.Context(), id); err != nil {
		writeSiteError(w, err)
		return
	}
	cookie, err := r.Cookie(accounts.SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeSiteError(w, ErrForbidden)
		return
	}
	if _, _, err := h.Accounts.ResolveSession(r.Context(), cookie.Value); err != nil {
		writeSiteError(w, ErrForbidden)
		return
	}
	if h.contentRequiresAuth(r) {
		http.SetCookie(w, &http.Cookie{
			Name:     contentCookieName,
			Value:    cookie.Value,
			Path:     ContentURLFor(id),
			MaxAge:   0,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		})
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"content_url": ContentURLFor(id) + h.grants.Mint(id) + "/",
	})
}

func (h *Handler) token(w http.ResponseWriter, r *http.Request, id string) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok || identity.Kind != accounts.KindUser {
		writeSiteError(w, ErrForbidden)
		return
	}
	site, err := h.Service.Get(r.Context(), id)
	if err != nil {
		writeSiteError(w, err)
		return
	}
	name := "Site: " + site.Title
	token, _, err := h.Accounts.CreateSitePAT(r.Context(), identity.ActorID, site.ID, name, nil)
	if err != nil {
		http.Error(w, "could not create a site token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"token":   token,
		"site_id": site.ID,
	})
}

func siteJSON(site Site) map[string]any {
	return map[string]any{
		"id":                    site.ID,
		"title":                 site.Title,
		"description":           site.Description,
		"only_creator_may_edit": site.OnlyCreatorMayEdit,
		"agency_wide":           site.AgencyWide,
		"creator_actor_id":      formatActorID(site.CreatorActorID),
		"origin_task_id":        site.OriginTaskID,
		"url":                   URLFor(site.ID),
		"content_url":           ContentURLFor(site.ID),
		"created_at":            site.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":            site.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func formatActorID(id ids.ActorID) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(int64(id), 10)
}

func contentTypeFor(filePath string) string {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	}
	if value := mime.TypeByExtension(path.Ext(filePath)); value != "" {
		return value
	}
	return "application/octet-stream"
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeSiteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		http.Error(w, "site not found", http.StatusNotFound)
	case errors.Is(err, ErrDisabled):
		http.Error(w, "object storage is not configured", http.StatusServiceUnavailable)
	case errors.Is(err, ErrForbidden):
		http.Error(w, "not allowed to edit this site", http.StatusForbidden)
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidArchive),
		errors.Is(err, ErrInvalidOriginTask):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrTooLarge):
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
	default:
		http.Error(w, "site request failed", http.StatusInternalServerError)
	}
}
