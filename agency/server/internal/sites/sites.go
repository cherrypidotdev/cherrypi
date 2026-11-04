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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/agents"
	"cherrypi.vladfaust.com/agency/internal/attachments"
	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/realtime"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tasks"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

var (
	ErrNotFound = errors.New("site not found")

	ErrDisabled = attachments.ErrDisabled

	ErrInvalidID = errors.New("site id must match [a-z0-9][a-z0-9-]{0,63}")

	ErrInvalidArchive = errors.New("invalid site archive")

	ErrTooLarge = errors.New("site archive exceeds the size limit")

	ErrForbidden = errors.New("not allowed to edit this site")

	ErrInvalidOriginTask = errors.New("origin task is not available to this agent")
)

const DefaultMaxBytes int64 = 50 << 20

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Config struct {
	CacheDir string

	MaxBytes int64
}

type Site struct {
	ID                 string
	Title              string
	Description        string
	OnlyCreatorMayEdit bool
	AgencyWide         bool
	CreatorActorID     ids.ActorID
	OriginTaskID       string
	AttachmentID       int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Settings struct {
	Title              string
	Description        string
	OnlyCreatorMayEdit bool
	AgencyWide         bool
}

type SettingsPatch struct {
	Title              *string
	Description        *string
	OnlyCreatorMayEdit *bool
	AgencyWide         *bool
}

type PublishInput struct {
	ID                 string
	ArchivePath        string
	ArchiveFilename    string
	OriginTaskID       string
	Title              *string
	Description        *string
	OnlyCreatorMayEdit *bool
	AgencyWide         *bool
}

type Service struct {
	store       store.Store
	objects     storageReader
	attachments *attachments.Service
	accounts    *accounts.Service
	agents      *agents.Registry
	tasks       *tasks.Service
	realtime    *realtime.Bus
	cfg         Config

	mu sync.Mutex
}

type storageReader interface {
	Open(ctx context.Context, id int64) (io.ReadCloser, attachments.Attachment, error)
	Delete(ctx context.Context, id int64) error
	Enabled() bool
}

func New(
	st store.Store,
	attachmentsSvc *attachments.Service,
	accountsSvc *accounts.Service,
	agentsReg *agents.Registry,
	tasksSvc *tasks.Service,
	bus *realtime.Bus,
	cfg Config,
) *Service {
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = DefaultMaxBytes
	}
	if cfg.CacheDir != "" {
		_ = os.MkdirAll(cfg.CacheDir, 0o755)
	}
	return &Service{
		store:       st,
		objects:     attachmentsSvc,
		attachments: attachmentsSvc,
		accounts:    accountsSvc,
		agents:      agentsReg,
		tasks:       tasksSvc,
		realtime:    bus,
		cfg:         cfg,
	}
}

func (s *Service) Enabled() bool { return s.attachments != nil && s.attachments.Enabled() }

func (s *Service) MaxBytes() int64 { return s.cfg.MaxBytes }

func URLFor(id string) string { return "/sites/" + id }

func ContentURLFor(id string) string { return "/sites/" + id + "/content/" }

func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return ErrInvalidID
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (Site, error) {
	row, err := s.store.GetSite(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, err
	}
	return fromRow(row), nil
}

func (s *Service) List(
	ctx context.Context,
	_ auth.Identity,
	originTaskID string,
	agencyWideOnly bool,
) ([]Site, error) {
	var (
		rows []store.Site
		err  error
	)
	if originTaskID != "" {
		rows, err = s.store.ListSitesByOriginTask(ctx, originTaskID)
	} else {
		rows, err = s.store.ListSites(ctx)
	}
	if err != nil {
		return nil, err
	}
	out := make([]Site, 0, len(rows))
	for _, row := range rows {
		site := fromRow(row)
		if agencyWideOnly && !site.AgencyWide {
			continue
		}
		out = append(out, site)
	}
	return out, nil
}

func (s *Service) CanEdit(ctx context.Context, site Site, identity auth.Identity) bool {
	if identity.ActorID == 0 {
		return false
	}
	if identity.IsAdmin() || identity.ActorID == site.CreatorActorID {
		return true
	}
	creator, err := s.accounts.GetActor(ctx, site.CreatorActorID)
	if err != nil || creator.Kind != accounts.KindAgent {
		return false
	}
	agent, err := s.agents.GetByActor(ctx, site.CreatorActorID)
	if err != nil {
		return false
	}
	ok, err := s.agents.IsOwner(ctx, agent.ID, identity.ActorID)
	return err == nil && ok
}

func canToolEdit(site Site, identity auth.Identity) bool {
	return identity.ActorID == site.CreatorActorID || !site.OnlyCreatorMayEdit
}

func (s *Service) Publish(
	ctx context.Context,
	identity auth.Identity,
	in PublishInput,
) (Site, error) {
	if err := ValidateID(in.ID); err != nil {
		return Site{}, err
	}
	if !s.Enabled() {
		return Site{}, ErrDisabled
	}

	existing, err := s.Get(ctx, in.ID)
	exists := err == nil
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Site{}, err
	}
	if exists {
		allowed := false
		if identity.Kind == accounts.KindAgent {
			allowed = canToolEdit(existing, identity)
		} else {
			allowed = s.CanEdit(ctx, existing, identity)
		}
		if !allowed {
			return Site{}, ErrForbidden
		}
	}

	originTaskID := existing.OriginTaskID
	if !exists {
		originTaskID = strings.TrimSpace(in.OriginTaskID)
		if originTaskID != "" {
			if err := s.validateOriginTask(ctx, identity, originTaskID); err != nil {
				return Site{}, err
			}
		}
	}

	if err := validateArchive(in.ArchivePath, s.cfg.MaxBytes); err != nil {
		return Site{}, err
	}
	info, err := os.Stat(in.ArchivePath)
	if err != nil {
		return Site{}, err
	}

	archive, err := os.Open(in.ArchivePath)
	if err != nil {
		return Site{}, err
	}
	defer func() { _ = archive.Close() }()

	filename := strings.TrimSpace(in.ArchiveFilename)
	if filename == "" {
		filename = in.ID + ".zip"
	}
	stored, err := s.attachments.UploadSiteArchive(ctx, attachments.File{
		Filename:    filename,
		ContentType: "application/zip",
		Body:        archive,
		Size:        info.Size(),
	}, actorid.Format(identity.ActorID))
	if err != nil {
		return Site{}, err
	}

	if err := s.extractToCache(ctx, stored.ID, in.ArchivePath); err != nil {
		_ = s.attachments.Delete(ctx, stored.ID)
		s.removeCache(stored.ID)
		return Site{}, err
	}

	now := time.Now()
	title := in.ID
	description := ""
	onlyCreator := true
	agencyWide := false
	if exists {
		title = existing.Title
		description = existing.Description
		onlyCreator = existing.OnlyCreatorMayEdit
		agencyWide = existing.AgencyWide
	}
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
	}
	if title == "" {
		title = in.ID
	}
	if in.Description != nil {
		description = strings.TrimSpace(*in.Description)
	}
	if in.OnlyCreatorMayEdit != nil {
		onlyCreator = *in.OnlyCreatorMayEdit
	}
	if in.AgencyWide != nil {
		agencyWide = *in.AgencyWide
	}

	var row store.Site
	if exists {
		row, err = s.store.UpdateSite(ctx, store.UpdateSiteParams{
			Title:              title,
			Description:        description,
			OnlyCreatorMayEdit: boolInt(onlyCreator),
			AgencyWide:         boolInt(agencyWide),
			AttachmentID:       stored.ID,
			UpdatedAt:          now.UnixMilli(),
			ID:                 in.ID,
		})
	} else {
		row, err = s.store.CreateSite(ctx, store.CreateSiteParams{
			ID:                 in.ID,
			Title:              title,
			Description:        description,
			OnlyCreatorMayEdit: boolInt(onlyCreator),
			AgencyWide:         boolInt(agencyWide),
			CreatorActorID:     int64(identity.ActorID),
			OriginTaskID:       originTaskID,
			AttachmentID:       stored.ID,
			CreatedAt:          now.UnixMilli(),
			UpdatedAt:          now.UnixMilli(),
		})
	}
	if err != nil {
		_ = s.attachments.Delete(ctx, stored.ID)
		s.removeCache(stored.ID)
		return Site{}, err
	}

	if exists && existing.AttachmentID != stored.ID {
		_ = s.attachments.Delete(ctx, existing.AttachmentID)
		s.removeCache(existing.AttachmentID)
	}
	s.signal()
	return fromRow(row), nil
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	identity auth.Identity,
	id string,
	patch SettingsPatch,
) (Site, error) {
	if err := ValidateID(id); err != nil {
		return Site{}, err
	}
	site, err := s.Get(ctx, id)
	if err != nil {
		return Site{}, err
	}
	allowed := false
	if identity.Kind == accounts.KindAgent {
		allowed = canToolEdit(site, identity)
	} else {
		allowed = s.CanEdit(ctx, site, identity)
	}
	if !allowed {
		return Site{}, ErrForbidden
	}

	title, description := site.Title, site.Description
	onlyCreator, agencyWide := site.OnlyCreatorMayEdit, site.AgencyWide
	if patch.Title != nil {
		title = strings.TrimSpace(*patch.Title)
	}
	if title == "" {
		title = id
	}
	if patch.Description != nil {
		description = strings.TrimSpace(*patch.Description)
	}
	if patch.OnlyCreatorMayEdit != nil {
		onlyCreator = *patch.OnlyCreatorMayEdit
	}
	if patch.AgencyWide != nil {
		agencyWide = *patch.AgencyWide
	}

	row, err := s.store.UpdateSiteSettings(ctx, store.UpdateSiteSettingsParams{
		Title:              title,
		Description:        description,
		OnlyCreatorMayEdit: boolInt(onlyCreator),
		AgencyWide:         boolInt(agencyWide),
		UpdatedAt:          time.Now().UnixMilli(),
		ID:                 id,
	})
	if err != nil {
		return Site{}, err
	}
	s.signal()
	return fromRow(row), nil
}

func (s *Service) Delete(ctx context.Context, identity auth.Identity, id string) error {
	if identity.Kind != accounts.KindUser {
		return ErrForbidden
	}
	site, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !s.CanEdit(ctx, site, identity) {
		return ErrForbidden
	}
	if err := s.store.DeleteSite(ctx, id); err != nil {
		return err
	}
	if site.AttachmentID > 0 {
		_ = s.attachments.Delete(ctx, site.AttachmentID)
	}
	s.removeCache(site.AttachmentID)
	s.signal()
	return nil
}

func (s *Service) ContentFile(ctx context.Context, id, requestPath string) (string, error) {
	site, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return s.ContentFileIn(ctx, site, requestPath)
}

func (s *Service) ContentFileIn(
	ctx context.Context,
	site Site,
	requestPath string,
) (string, error) {
	if err := s.ensureCache(ctx, site); err != nil {
		return "", err
	}

	name, err := cleanContentPath(requestPath)
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(s.cacheDir(site.AttachmentID), filepath.FromSlash(name))
	info, err := os.Stat(filePath)
	if err == nil && info.IsDir() {
		filePath = filepath.Join(filePath, "index.html")
		info, err = os.Stat(filePath)
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", ErrNotFound
	}
	return filePath, nil
}

func (s *Service) cacheDir(attachmentID int64) string {
	return filepath.Join(s.cfg.CacheDir, strconv.FormatInt(attachmentID, 10))
}

func (s *Service) removeCache(attachmentID int64) {
	if s.cfg.CacheDir == "" || attachmentID <= 0 {
		return
	}
	_ = os.RemoveAll(s.cacheDir(attachmentID))
}

func (s *Service) signal() {
	if s.realtime != nil {
		s.realtime.Publish(agencyv1.RealtimeKind_REALTIME_KIND_SITES)
	}
}

func (s *Service) validateOriginTask(
	ctx context.Context,
	identity auth.Identity,
	taskID string,
) error {
	if s.tasks == nil {
		return nil
	}
	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return ErrInvalidOriginTask
	}
	if identity.Kind == accounts.KindAgent &&
		task.AssigneeActorID != identity.ActorID {
		return ErrInvalidOriginTask
	}
	return nil
}

func cleanContentPath(raw string) (string, error) {
	raw = strings.ReplaceAll(raw, "\\", "/")
	if strings.Contains(raw, "..") {
		return "", ErrNotFound
	}
	cleaned := path.Clean("/" + raw)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || strings.HasSuffix(raw, "/") {
		if cleaned == "" {
			return "index.html", nil
		}
		return cleaned + "/index.html", nil
	}
	return cleaned, nil
}

func fromRow(row store.Site) Site {
	return Site{
		ID:                 row.ID,
		Title:              row.Title,
		Description:        row.Description,
		OnlyCreatorMayEdit: row.OnlyCreatorMayEdit != 0,
		AgencyWide:         row.AgencyWide != 0,
		CreatorActorID:     ids.ActorID(row.CreatorActorID),
		OriginTaskID:       row.OriginTaskID,
		AttachmentID:       row.AttachmentID,
		CreatedAt:          time.UnixMilli(row.CreatedAt),
		UpdatedAt:          time.UnixMilli(row.UpdatedAt),
	}
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func (s *Service) ensureCache(ctx context.Context, site Site) error {
	if s.cfg.CacheDir == "" {
		return errors.New("site cache is not configured")
	}
	dir := s.cacheDir(site.AttachmentID)
	if _, err := os.Stat(filepath.Join(dir, markerName)); err == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(filepath.Join(dir, markerName)); err == nil {
		return nil
	}

	body, _, err := s.attachments.Open(ctx, site.AttachmentID)
	if err != nil {
		return fmt.Errorf("open site archive: %w", err)
	}
	defer func() { _ = body.Close() }()

	zipFile, err := os.CreateTemp(s.cfg.CacheDir, ".archive-*.zip")
	if err != nil {
		return err
	}
	zipPath := zipFile.Name()
	defer func() { _ = os.Remove(zipPath) }()
	if _, err := io.Copy(zipFile, body); err != nil {
		_ = zipFile.Close()
		return err
	}
	if err := zipFile.Close(); err != nil {
		return err
	}
	if err := s.extractToCache(ctx, site.AttachmentID, zipPath); err != nil {
		return err
	}
	return nil
}
