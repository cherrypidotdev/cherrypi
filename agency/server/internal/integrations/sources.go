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

package integrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
)

type Credential struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Host        string `json:"host"`
	Port        int64  `json:"port"`
	Username    string `json:"username"`
	HasSecret   bool   `json:"hasSecret"`
	KnownHosts  string `json:"knownHosts"`
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`

	Sources   int64     `json:"sources"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CredentialInput struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Host string `json:"host"`
	Port int64  `json:"port"`

	Username string `json:"username"`

	Secret string `json:"secret"`

	KeepSecret bool   `json:"keepSecret"`
	KnownHosts string `json:"knownHosts"`
}

type SourceSkill struct {
	Path                   string      `json:"path"`
	RawName                string      `json:"rawName"`
	Name                   string      `json:"name"`
	Description            string      `json:"description"`
	Body                   string      `json:"body,omitempty"`
	Files                  []SkillFile `json:"files,omitempty"`
	DisableModelInvocation bool        `json:"disableModelInvocation"`

	Status string `json:"status"`

	EntryID int64  `json:"entryId,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Source struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       Scope  `json:"owner"`
	RepoURL     string `json:"repoUrl"`
	Ref         string `json:"ref"`
	PathGlob    string `json:"pathGlob"`

	CredentialID   int64     `json:"credentialId"`
	CredentialName string    `json:"credentialName"`
	UpdateMode     string    `json:"updateMode"`
	PollIntervalS  int64     `json:"pollIntervalSeconds"`
	Paused         bool      `json:"paused"`
	Installs       []Install `json:"installs"`

	SyncState     string         `json:"syncState"`
	SyncError     string         `json:"syncError"`
	SyncedCommit  string         `json:"syncedCommit"`
	SyncedAt      time.Time      `json:"syncedAt"`
	CheckedAt     time.Time      `json:"checkedAt"`
	PendingCommit string         `json:"pendingCommit"`
	Pending       []SourceSkill  `json:"pending"`
	Problems      []skillProblem `json:"problems"`

	SkillCount int64     `json:"skillCount"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type SourceInput struct {
	ID            int64     `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Owner         Scope     `json:"owner"`
	RepoURL       string    `json:"repoUrl"`
	Ref           string    `json:"ref"`
	PathGlob      string    `json:"pathGlob"`
	CredentialID  int64     `json:"credentialId"`
	UpdateMode    string    `json:"updateMode"`
	PollIntervalS int64     `json:"pollIntervalSeconds"`
	Paused        bool      `json:"paused"`
	Installs      []Install `json:"installs"`
}

type SourcePreview struct {
	Commit   string         `json:"commit"`
	Ref      string         `json:"ref"`
	Skills   []SourceSkill  `json:"skills"`
	Problems []skillProblem `json:"problems"`
}

type pendingSet struct {
	Commit string            `json:"commit"`
	Ref    string            `json:"ref"`
	Skills []discoveredSkill `json:"skills"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const maxSlugLength = 32

var hostPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

const (
	SkillAdded     = "added"
	SkillUpdated   = "updated"
	SkillUnchanged = "unchanged"
	SkillRemoved   = "removed"
	SkillConflict  = "conflict"
)

func (s *Service) GitStatus(ctx context.Context) (GitSourcesStatus, error) {
	status := GitSourcesStatus{}
	if s.features != nil {
		enabled, err := s.features.GitSourcesEnabled(ctx)
		if err != nil {
			return GitSourcesStatus{}, err
		}
		status.Enabled = enabled
	}
	status.Git, status.SSH = s.probeCLIs(ctx)
	status.Ready = status.Enabled && status.Git.OK && status.SSH.OK
	if status.Enabled && !status.Ready {
		missing := make([]string, 0, 2)
		if !status.Git.OK {
			missing = append(missing, "git")
		}
		if !status.SSH.OK {
			missing = append(missing, "ssh")
		}
		status.Message = "the agency has no " + strings.Join(missing, " and ") + " binary"
	}
	return status, nil
}

func (s *Service) SetGitSources(ctx context.Context, actorID int64, enabled bool) error {
	if !s.admin(ctx, actorID) {
		return ErrForbidden
	}
	if s.features == nil {
		return ErrFeatureDisabled
	}
	if enabled {
		gitCLI, sshCLI := s.probeCLIs(ctx)
		if !gitCLI.OK || !sshCLI.OK {
			return ErrGitUnavailable
		}
	}
	return s.features.SetGitSourcesEnabled(ctx, enabled)
}

func (s *Service) admin(ctx context.Context, actorID int64) bool {
	if s.auth == nil || s.auth.IsAdmin == nil || actorID == 0 {
		return false
	}
	return s.auth.IsAdmin(ctx, actorID)
}

func (s *Service) requireGitSources(ctx context.Context) error {
	if s.features == nil {
		return ErrFeatureDisabled
	}
	enabled, err := s.features.GitSourcesEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrFeatureDisabled
	}
	gitCLI, sshCLI := s.probeCLIs(ctx)
	if !gitCLI.OK || !sshCLI.OK {
		return ErrGitUnavailable
	}
	return nil
}

func (s *Service) Credentials(ctx context.Context, actorID int64) ([]Credential, error) {
	if !s.admin(ctx, actorID) {
		return nil, ErrForbidden
	}
	rows, err := s.store.ListGitCredentials(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Credential, 0, len(rows))
	for _, row := range rows {
		count, err := s.store.CountGitCredentialSources(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, credentialOf(row, count))
	}
	return out, nil
}

func (s *Service) SaveCredential(
	ctx context.Context,
	actorID int64,
	input CredentialInput,
) (Credential, error) {
	if !s.admin(ctx, actorID) {
		return Credential{}, ErrForbidden
	}
	existing, err := s.storedCredential(ctx, input.ID)
	if err != nil {
		return Credential{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || !namePattern.MatchString(name) {
		return Credential{}, invalidf(
			"a credential name is 1-64 letters, digits, dots, dashes or underscores",
		)
	}
	kind := strings.TrimSpace(input.Kind)
	if kind != CredentialToken && kind != CredentialSSHKey {
		return Credential{}, invalidf("a credential is either a token or an SSH key")
	}
	host := strings.ToLower(strings.TrimSpace(input.Host))
	if !validCredentialHost(host) {
		return Credential{}, invalidf(
			"a credential host is a bare host name, without a scheme or a path",
		)
	}
	port := input.Port
	if port < 0 || port > 65535 {
		return Credential{}, invalidf("a port is between 1 and 65535")
	}

	if input.ID != 0 {
		inUse, err := s.store.CountGitCredentialSources(ctx, input.ID)
		if err != nil {
			return Credential{}, err
		}
		if inUse > 0 && (existing.Kind != kind || existing.Host != host || existing.Port != port) {
			return Credential{}, invalidf(
				"%d source(s) use this credential; change their repository first",
				inUse,
			)
		}
	}
	username := strings.TrimSpace(input.Username)
	knownHosts := strings.TrimSpace(input.KnownHosts)
	if knownHosts != "" {
		knownHosts += "\n"
	}
	publicKey := ""
	fingerprint := ""

	sealedSecret := existing.SecretSealed
	secret := strings.TrimSpace(input.Secret)
	if secret != "" {
		if !s.Sealing() {
			return Credential{}, invalidf(
				"CHERRYPI_SECRET_KEY is not set, so the agency cannot hold a credential",
			)
		}
		if kind == CredentialSSHKey {
			if !strings.HasPrefix(secret, "-----BEGIN") {
				return Credential{}, invalidf("that does not look like an OpenSSH private key")
			}
			publicKey, err = publicKeyOf(secret + "\n")
			if err != nil {
				return Credential{}, err
			}
			fingerprint = fingerprintOfPublicKey(publicKey)
		}
		blob, err := s.seal.Seal(secret)
		if err != nil {
			return Credential{}, err
		}
		sealedSecret = blob
	}

	if input.ID != 0 && existing.Kind != kind && secret == "" {
		return Credential{}, invalidf(
			"changing the credential kind needs a new token or private key",
		)
	}
	if len(sealedSecret) == 0 {
		return Credential{}, invalidf("a credential needs a token or a private key")
	}
	if kind == CredentialSSHKey {
		if knownHosts == "" {
			return Credential{}, invalidf("an SSH credential needs a pinned host key")
		}
		if publicKey == "" {

			if opened, openErr := s.seal.Open(sealedSecret); openErr == nil {
				publicKey, _ = publicKeyOf(string(opened))
				fingerprint = fingerprintOfPublicKey(publicKey)
			}
		}
	} else {

		knownHosts = ""
		publicKey = ""
		fingerprint = ""
	}
	params := store.GitCredentialParams{
		ID:               input.ID,
		Name:             name,
		Kind:             kind,
		Host:             host,
		Port:             port,
		Username:         username,
		SecretSealed:     sealedSecret,
		KnownHosts:       knownHosts,
		PublicKey:        publicKey,
		Fingerprint:      fingerprint,
		CreatedByActorID: actorID,
		CreatedAt:        s.now().UnixMilli(),
		UpdatedAt:        s.now().UnixMilli(),
	}
	var saved store.GitCredential
	if input.ID == 0 {
		saved, err = s.store.CreateGitCredential(ctx, params)
	} else {
		saved, err = s.store.UpdateGitCredential(ctx, params)
	}
	if err != nil {
		return Credential{}, mapUniqueCredential(err)
	}
	count, err := s.store.CountGitCredentialSources(ctx, saved.ID)
	if err != nil {
		return Credential{}, err
	}
	return credentialOf(saved, count), nil
}

func (s *Service) DeleteCredential(ctx context.Context, actorID, id int64) error {
	if !s.admin(ctx, actorID) {
		return ErrForbidden
	}
	count, err := s.store.CountGitCredentialSources(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return invalidf("%d source(s) still use this credential", count)
	}
	deleted, err := s.store.DeleteGitCredential(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ScanHostKeys(
	ctx context.Context,
	actorID int64,
	host string,
	port int64,
) ([]HostKey, string, error) {
	if !s.admin(ctx, actorID) {
		return nil, "", ErrForbidden
	}
	trimmed := strings.ToLower(strings.TrimSpace(host))
	if !validCredentialHost(trimmed) {
		return nil, "", invalidf("a host is a bare host name, without a scheme or a path")
	}
	return scanHostKeys(ctx, trimmed, port)
}

func (s *Service) storedCredential(
	ctx context.Context,
	id int64,
) (store.GitCredential, error) {
	if id == 0 {
		return store.GitCredential{}, nil
	}
	row, err := s.store.GetGitCredential(ctx, id)
	if err != nil {
		return store.GitCredential{}, mapStoreError(err)
	}
	return row, nil
}

func credentialOf(row store.GitCredential, sources int64) Credential {
	return Credential{
		ID:          row.ID,
		Name:        row.Name,
		Kind:        row.Kind,
		Host:        row.Host,
		Port:        row.Port,
		Username:    row.Username,
		HasSecret:   len(row.SecretSealed) > 0,
		KnownHosts:  strings.TrimSpace(row.KnownHosts),
		PublicKey:   row.PublicKey,
		Fingerprint: row.Fingerprint,
		Sources:     sources,
		UpdatedAt:   time.UnixMilli(row.UpdatedAt).UTC(),
	}
}

func validCredentialHost(host string) bool {
	if host == "" || len(host) > 255 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	return hostPattern.MatchString(host)
}

func (s *Service) Sources(ctx context.Context, actorID int64) ([]Source, error) {
	rows, err := s.store.ListSkillSources(ctx)
	if err != nil {
		return nil, err
	}
	installs, err := s.store.ListSkillSourceInstalls(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := s.store.ListSkillEntries(ctx)
	if err != nil {
		return nil, err
	}
	credentials, err := s.store.ListGitCredentials(ctx)
	if err != nil {
		return nil, err
	}
	credentialNames := make(map[int64]string, len(credentials))
	for _, row := range credentials {
		credentialNames[row.ID] = row.Name
	}
	byName := make(map[string]store.Skill, len(skills))
	for _, row := range skills {
		byName[row.Name] = row
	}
	out := make([]Source, 0, len(rows))
	for _, row := range rows {
		visible, err := s.canSee(ctx, actorID, Scope{Kind: row.OwnerKind, ID: row.OwnerID})
		if err != nil {
			return nil, err
		}
		if !visible {
			continue
		}
		out = append(out, sourceOf(row, installs, skills, byName, credentialNames))
	}
	return out, nil
}

func (s *Service) GetSource(ctx context.Context, actorID, id int64) (Source, error) {
	row, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return Source{}, mapStoreError(err)
	}
	visible, err := s.canSee(ctx, actorID, Scope{Kind: row.OwnerKind, ID: row.OwnerID})
	if err != nil {
		return Source{}, err
	}
	if !visible {
		return Source{}, ErrNotFound
	}
	return s.sourceView(ctx, row)
}

func (s *Service) sourceView(
	ctx context.Context,
	row store.SkillSource,
) (Source, error) {
	installs, err := s.store.ListSkillSourceInstalls(ctx)
	if err != nil {
		return Source{}, err
	}
	skills, err := s.store.ListSkillEntries(ctx)
	if err != nil {
		return Source{}, err
	}
	credentials, err := s.store.ListGitCredentials(ctx)
	if err != nil {
		return Source{}, err
	}
	credentialNames := make(map[int64]string, len(credentials))
	for _, row := range credentials {
		credentialNames[row.ID] = row.Name
	}
	byName := make(map[string]store.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}
	return sourceOf(row, installs, skills, byName, credentialNames), nil
}

func sourceOf(
	row store.SkillSource,
	installs []store.SkillSourceInstall,
	skills []store.Skill,
	byName map[string]store.Skill,
	credentialNames map[int64]string,
) Source {
	out := Source{
		ID:            row.ID,
		Slug:          row.Slug,
		Name:          row.Name,
		Description:   row.Description,
		Owner:         Scope{Kind: row.OwnerKind, ID: row.OwnerID},
		RepoURL:       row.RepoUrl,
		Ref:           row.Ref,
		PathGlob:      row.PathGlob,
		UpdateMode:    row.UpdateMode,
		PollIntervalS: row.PollIntervalS,
		Paused:        row.Paused != 0,
		Installs:      []Install{},
		SyncState:     row.SyncState,
		SyncError:     row.SyncError,
		SyncedCommit:  row.SyncedCommit,
		SyncedAt:      time.UnixMilli(row.SyncedAt).UTC(),
		CheckedAt:     time.UnixMilli(row.CheckedAt).UTC(),
		PendingCommit: row.PendingCommit,
		Pending:       []SourceSkill{},
		Problems:      []skillProblem{},
		UpdatedAt:     time.UnixMilli(row.UpdatedAt).UTC(),
	}
	if row.CredentialID.Valid {
		out.CredentialID = row.CredentialID.Int64
		out.CredentialName = credentialNames[row.CredentialID.Int64]
	}
	for _, install := range installs {
		if install.SourceID != row.ID {
			continue
		}
		out.Installs = append(out.Installs, Install{
			Scope:   Scope{Kind: install.ScopeKind, ID: install.ScopeID},
			Enabled: install.Enabled != 0,
		})
	}
	sortInstalls(out.Installs)

	own := make(map[string]store.Skill, len(skills))
	for _, skill := range skills {
		if skill.SourceID.Valid && skill.SourceID.Int64 == row.ID {
			own[skill.SourcePath] = skill
			if skill.SourceMissing == 0 {
				out.SkillCount++
			}
		}
	}

	if strings.TrimSpace(row.ProblemsJson) != "" {
		_ = json.Unmarshal([]byte(row.ProblemsJson), &out.Problems)
	}

	var staged pendingSet
	if strings.TrimSpace(row.PendingJson) != "" {
		_ = json.Unmarshal([]byte(row.PendingJson), &staged)
	}
	if len(staged.Skills) == 0 {
		return out
	}
	stagedPaths := make(map[string]bool, len(staged.Skills))
	for _, skill := range staged.Skills {
		stagedPaths[skill.Path] = true
		item := SourceSkill{
			Path:                   skill.Path,
			RawName:                skill.RawName,
			Name:                   skill.Name,
			Description:            skill.Description,
			Body:                   skill.Body,
			Files:                  skill.Files,
			DisableModelInvocation: skill.DisableModelInvocation,
			Status:                 SkillAdded,
		}
		if existing, ok := own[skill.Path]; ok {
			item.EntryID = existing.ID
			item.Status = SkillUpdated
			if sameSkillContent(existing, skill) {
				item.Status = SkillUnchanged
			}
		}
		if holder, taken := byName[skill.Name]; taken {
			sameRow := holder.SourceID.Valid &&
				holder.SourceID.Int64 == row.ID &&
				holder.SourcePath == skill.Path
			if !sameRow {
				item.Status = SkillConflict
				item.Error = "another entry already has this name"
			}
		}
		out.Pending = append(out.Pending, item)
	}
	for skillPath, existing := range own {
		if stagedPaths[skillPath] {
			continue
		}

		out.Pending = append(out.Pending, SourceSkill{
			Path:    skillPath,
			Name:    existing.Name,
			RawName: existing.Name,
			EntryID: existing.ID,
			Status:  SkillRemoved,
		})
	}
	sort.Slice(out.Pending, func(i, j int) bool {
		return out.Pending[i].Name < out.Pending[j].Name
	})
	return out
}

func sameSkillContent(row store.Skill, skill discoveredSkill) bool {
	if row.Name != skill.Name ||
		row.Description != skill.Description ||
		row.Body != skill.Body ||
		row.DisableModelInvocation != boolInt(skill.DisableModelInvocation) {
		return false
	}
	var files []SkillFile
	if err := json.Unmarshal([]byte(row.FilesJson), &files); err != nil {
		return false
	}
	if len(files) != len(skill.Files) {
		return false
	}
	for i := range files {
		if files[i] != skill.Files[i] {
			return false
		}
	}
	return true
}

func (s *Service) SaveSource(
	ctx context.Context,
	actorID int64,
	input SourceInput,
) (Source, error) {
	if err := s.requireGitSources(ctx); err != nil {
		return Source{}, err
	}
	previous, err := s.storedSource(ctx, input.ID)
	if err != nil {
		return Source{}, err
	}
	if input.ID != 0 && len(input.Installs) > 0 {

		return Source{}, invalidf(
			"an edit does not change installs; use PUT/DELETE /api/rest/integrations/sources/<id>/install",
		)
	}
	if input.ID != 0 {
		if err := s.auth.CanAuthor(ctx, actorID, storedOwnerOfSource(previous)); err != nil {
			return Source{}, err
		}
	}
	for _, install := range input.Installs {
		scope := normalizeScope(install.Scope)
		if err := validScope(scope, true); err != nil {
			return Source{}, err
		}

		if err := s.auth.CanAdminister(ctx, actorID, scope); err != nil {
			return Source{}, err
		}
	}
	owner := normalizeScope(input.Owner)
	if err := validScope(owner, true); err != nil {
		return Source{}, err
	}
	if err := s.auth.CanAuthor(ctx, actorID, owner); err != nil {
		return Source{}, err
	}
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if slug == "" {
		return Source{}, invalidf("a source needs a slug")
	}
	if len(slug) > maxSlugLength || !slugPattern.MatchString(slug) {
		return Source{}, invalidf(
			"a source slug is lowercase a-z, 0-9 and single hyphens, up to %d characters",
			maxSlugLength,
		)
	}
	repo, err := parseRepoURL(input.RepoURL)
	if err != nil {
		return Source{}, err
	}
	glob, err := validateSkillGlob(input.PathGlob)
	if err != nil {
		return Source{}, err
	}
	ref := strings.TrimSpace(input.Ref)
	if strings.ContainsAny(ref, " \t\n") || strings.Contains(ref, "..") {
		return Source{}, invalidf("a ref cannot contain whitespace or ..")
	}
	credentialID := input.CredentialID
	if credentialID != 0 {
		credential, err := s.store.GetGitCredential(ctx, credentialID)
		if err != nil {
			return Source{}, mapStoreError(err)
		}
		if credential.Host != repo.Host {
			return Source{}, invalidf(
				"the credential belongs to %s, not %s",
				credential.Host, repo.Host,
			)
		}
		if credential.Kind == CredentialToken && repo.Scheme != "https" && repo.Scheme != "http" {
			return Source{}, invalidf("a token credential only reaches an http(s) repository")
		}
		if credential.Kind == CredentialSSHKey && repo.Scheme != "ssh" {
			return Source{}, invalidf("an SSH key credential only reaches an ssh repository")
		}
	} else if repo.Scheme == "ssh" {
		return Source{}, invalidf("an SSH repository needs a credential")
	}
	updateMode := strings.TrimSpace(input.UpdateMode)
	if updateMode == "" {
		updateMode = UpdateStaged
	}
	if updateMode != UpdateStaged && updateMode != UpdateAuto {
		return Source{}, invalidf("the update mode is either staged or auto")
	}
	poll := input.PollIntervalS
	if poll == 0 {
		poll = defaultPollInterval
	}
	if poll < minPollIntervalS {
		poll = minPollIntervalS
	}
	if poll > maxPollIntervalS {
		poll = maxPollIntervalS
	}
	params := store.SkillSourceParams{
		ID:               input.ID,
		Slug:             slug,
		Name:             strings.TrimSpace(input.Name),
		Description:      strings.TrimSpace(input.Description),
		OwnerKind:        owner.Kind,
		OwnerID:          owner.ID,
		RepoURL:          repo.URL,
		Ref:              ref,
		PathGlob:         glob,
		CredentialID:     credentialID,
		UpdateMode:       updateMode,
		PollIntervalS:    poll,
		Paused:           boolInt(input.Paused),
		CreatedByActorID: actorID,
		CreatedAt:        s.now().UnixMilli(),
		UpdatedAt:        s.now().UnixMilli(),
	}
	save := store.SaveSkillSourceParams{
		Entry:        params,
		KeepInstalls: input.ID != 0,

		Revision: s.nextRevision(),
	}
	if input.ID == 0 {
		save.Installs = installParams(input.Installs, actorID, s.now().UnixMilli())
	}
	saved, err := s.store.SaveSkillSource(ctx, save)
	if err != nil {
		return Source{}, mapUniqueSource(err)
	}
	return s.sourceView(ctx, saved)
}

func (s *Service) storedSource(
	ctx context.Context,
	id int64,
) (store.SkillSource, error) {
	if id == 0 {
		return store.SkillSource{}, nil
	}
	row, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return store.SkillSource{}, mapStoreError(err)
	}
	return row, nil
}

func storedOwnerOfSource(row store.SkillSource) Scope {
	return Scope{Kind: row.OwnerKind, ID: row.OwnerID}
}

func (s *Service) DeleteSource(ctx context.Context, actorID, id int64) error {
	row, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return mapStoreError(err)
	}
	visible, err := s.canSee(ctx, actorID, storedOwnerOfSource(row))
	if err != nil {
		return err
	}
	if !visible {
		return ErrNotFound
	}
	if err := s.auth.CanAuthor(ctx, actorID, storedOwnerOfSource(row)); err != nil {
		return err
	}
	deleted, err := s.store.DeleteSkillSource(ctx, id, s.nextRevision())
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) SetSourceInstall(
	ctx context.Context,
	actorID, id int64,
	scope Scope,
	enabled, remove bool,
) error {
	row, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return mapStoreError(err)
	}
	visible, err := s.canSee(ctx, actorID, storedOwnerOfSource(row))
	if err != nil {
		return err
	}
	if !visible {
		return ErrNotFound
	}
	scope = normalizeScope(scope)
	if err := validScope(scope, true); err != nil {
		return err
	}
	if err := s.auth.CanAdminister(ctx, actorID, scope); err != nil {
		return err
	}
	if remove {
		return s.store.RemoveSkillSourceInstall(ctx, store.RemoveInstallParams{
			EntryID:   id,
			ScopeKind: scope.Kind,
			ScopeID:   scope.ID,
			Revision:  s.nextRevision(),
		})
	}
	return s.store.SetSkillSourceInstall(ctx, store.SetInstallParams{
		EntryID:          id,
		ScopeKind:        scope.Kind,
		ScopeID:          scope.ID,
		Enabled:          boolInt(enabled),
		CreatedByActorID: actorID,
		CreatedAt:        s.now().UnixMilli(),
		Revision:         s.nextRevision(),
	})
}

func (s *Service) SyncSource(ctx context.Context, actorID, id int64) (Source, error) {
	if err := s.requireGitSources(ctx); err != nil {
		return Source{}, err
	}
	row, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return Source{}, mapStoreError(err)
	}
	if err := s.authorizeSource(ctx, actorID, row); err != nil {
		return Source{}, err
	}
	if err := s.checkSource(ctx, row, actorID); err != nil {
		return Source{}, err
	}

	updated, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return Source{}, err
	}
	return s.sourceView(ctx, updated)
}

func (s *Service) ApplySource(ctx context.Context, actorID, id int64) (Source, error) {
	if err := s.requireGitSources(ctx); err != nil {
		return Source{}, err
	}
	row, err := s.store.GetSkillSource(ctx, id)
	if err != nil {
		return Source{}, mapStoreError(err)
	}
	if err := s.authorizeSource(ctx, actorID, row); err != nil {
		return Source{}, err
	}

	if !s.beginSync(row.ID) {
		return Source{}, invalidf("a sync of this source is already running")
	}
	defer s.endSync(row.ID)
	set, err := s.stagedSet(row)
	if err != nil {
		return Source{}, err
	}
	if set == nil {
		return Source{}, invalidf("there is no staged update to apply")
	}
	problems, err := problemsOf(row)
	if err != nil {
		return Source{}, err
	}
	if err := s.applySet(ctx, row, *set, problems, actorID); err != nil {
		return Source{}, err
	}
	row, err = s.store.GetSkillSource(ctx, id)
	if err != nil {
		return Source{}, err
	}
	return s.sourceView(ctx, row)
}

func (s *Service) DetachSkill(ctx context.Context, actorID, id int64) (Skill, error) {
	row, err := s.store.GetSkillEntry(ctx, id)
	if err != nil {
		return Skill{}, mapStoreError(err)
	}
	owner := storedOwnerOfSkill(row)
	visible, err := s.canSee(ctx, actorID, owner)
	if err != nil {
		return Skill{}, err
	}
	if !visible {
		return Skill{}, ErrNotFound
	}
	if err := s.auth.CanAuthor(ctx, actorID, owner); err != nil {
		return Skill{}, err
	}
	if !row.SourceID.Valid {
		return Skill{}, invalidf("this skill is not imported from a Git source")
	}
	if err := s.store.DetachSkill(ctx, id, s.now().UnixMilli(), s.nextRevision()); err != nil {
		return Skill{}, err
	}
	installs, err := s.store.ListSkillInstalls(ctx)
	if err != nil {
		return Skill{}, err
	}
	detached, err := s.store.GetSkillEntry(ctx, id)
	if err != nil {
		return Skill{}, err
	}
	return skillOf(detached, installs, nil), nil
}

func (s *Service) SetSkillEnabled(ctx context.Context, actorID, id int64, enabled bool) error {
	row, err := s.store.GetSkillEntry(ctx, id)
	if err != nil {
		return mapStoreError(err)
	}
	owner := storedOwnerOfSkill(row)
	visible, err := s.canSee(ctx, actorID, owner)
	if err != nil {
		return err
	}
	if !visible {
		return ErrNotFound
	}
	if err := s.auth.CanAuthor(ctx, actorID, owner); err != nil {
		return err
	}
	_, err = s.store.SaveSkillEntry(ctx, store.SaveSkillEntryParams{
		Entry: store.SkillParams{
			ID:                     row.ID,
			Name:                   row.Name,
			Description:            row.Description,
			Body:                   row.Body,
			FilesJson:              row.FilesJson,
			DisableModelInvocation: row.DisableModelInvocation,
			OwnerKind:              row.OwnerKind,
			OwnerID:                row.OwnerID,
			Enabled:                boolInt(enabled),
			UpdatedAt:              s.now().UnixMilli(),
		},
		KeepInstalls: true,
		Revision:     s.nextRevision(),
	})
	return err
}

func (s *Service) PreviewSource(
	ctx context.Context,
	actorID int64,
	input SourceInput,
) (SourcePreview, error) {
	if err := s.requireGitSources(ctx); err != nil {
		return SourcePreview{}, err
	}
	previous, err := s.storedSource(ctx, input.ID)
	if err != nil {
		return SourcePreview{}, err
	}
	if input.ID != 0 {
		if err := s.authorizeSource(ctx, actorID, previous); err != nil {
			return SourcePreview{}, err
		}
	}
	owner := normalizeScope(input.Owner)
	if err := validScope(owner, true); err != nil {
		return SourcePreview{}, err
	}
	if err := s.auth.CanAuthor(ctx, actorID, owner); err != nil {
		return SourcePreview{}, err
	}
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if slug == "" {
		return SourcePreview{}, invalidf("a source needs a slug")
	}
	if len(slug) > maxSlugLength || !slugPattern.MatchString(slug) {
		return SourcePreview{}, invalidf(
			"a source slug is lowercase a-z, 0-9 and single hyphens, up to %d characters",
			maxSlugLength,
		)
	}
	repo, err := parseRepoURL(input.RepoURL)
	if err != nil {
		return SourcePreview{}, err
	}
	glob, err := validateSkillGlob(input.PathGlob)
	if err != nil {
		return SourcePreview{}, err
	}
	draft := previous
	draft.ID = input.ID
	draft.Slug = slug
	draft.RepoUrl = repo.URL
	draft.Ref = strings.TrimSpace(input.Ref)
	draft.PathGlob = glob
	credentialID := input.CredentialID
	draft.CredentialID = sqlNullID(credentialID)
	if credentialID != 0 {
		credential, err := s.store.GetGitCredential(ctx, credentialID)
		if err != nil {
			return SourcePreview{}, mapStoreError(err)
		}
		if credential.Host != repo.Host {
			return SourcePreview{}, invalidf(
				"the credential belongs to %s, not %s", credential.Host, repo.Host,
			)
		}
	} else if repo.Scheme == "ssh" {
		return SourcePreview{}, invalidf("an SSH repository needs a credential")
	}
	previewRepo, auth, commit, ref, err := s.resolveSource(ctx, draft)
	if err != nil {
		return SourcePreview{}, err
	}
	defer auth.close()
	skills, problems, err := s.readSource(ctx, draft, previewRepo, auth, commit, ref)
	if err != nil {
		return SourcePreview{}, err
	}

	problems = boundProblems(problems)
	skills2, err := s.store.ListSkillEntries(ctx)
	if err != nil {
		return SourcePreview{}, err
	}
	byName := make(map[string]store.Skill, len(skills2))
	for _, row := range skills2 {
		byName[row.Name] = row
	}
	ownByPath := map[string]store.Skill{}
	if input.ID != 0 {
		own, err := s.store.ListSkillsBySource(ctx, input.ID)
		if err != nil {
			return SourcePreview{}, err
		}
		for _, row := range own {
			ownByPath[row.SourcePath] = row
		}
	}
	out := SourcePreview{
		Commit:   commit,
		Ref:      ref,
		Problems: problems,
		Skills:   make([]SourceSkill, 0, len(skills)),
	}
	for _, skill := range skills {
		item := SourceSkill{
			Path:                   skill.Path,
			RawName:                skill.RawName,
			Name:                   skill.Name,
			Description:            skill.Description,
			Body:                   skill.Body,
			Files:                  skill.Files,
			DisableModelInvocation: skill.DisableModelInvocation,
			Status:                 SkillAdded,
		}
		if existing, ok := ownByPath[skill.Path]; ok {
			item.EntryID = existing.ID
			item.Status = SkillUpdated
			if sameSkillContent(existing, skill) {
				item.Status = SkillUnchanged
			}
		}
		if holder, taken := byName[skill.Name]; taken {
			sameRow := holder.SourceID.Valid && input.ID != 0 &&
				holder.SourceID.Int64 == input.ID && holder.SourcePath == skill.Path
			if !sameRow {
				item.Status = SkillConflict
				item.Error = "another entry already has this name"
			}
		}
		out.Skills = append(out.Skills, item)
	}
	return out, nil
}

func (s *Service) authorizeSource(
	ctx context.Context,
	actorID int64,
	row store.SkillSource,
) error {
	visible, err := s.canSee(ctx, actorID, storedOwnerOfSource(row))
	if err != nil {
		return err
	}
	if !visible {
		return ErrNotFound
	}
	return s.auth.CanAuthor(ctx, actorID, storedOwnerOfSource(row))
}

func (s *Service) beginSync(id int64) bool {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.inFlight == nil {
		s.inFlight = map[int64]bool{}
	}
	if s.inFlight[id] {
		return false
	}
	s.inFlight[id] = true
	return true
}

func (s *Service) endSync(id int64) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	delete(s.inFlight, id)
}

func (s *Service) checkSource(
	ctx context.Context,
	row store.SkillSource,
	actorID int64,
) error {
	if !s.beginSync(row.ID) {
		return invalidf("a sync of this source is already running")
	}
	defer s.endSync(row.ID)
	return s.runCheck(ctx, row, actorID)
}

func (s *Service) runCheck(
	ctx context.Context,
	row store.SkillSource,
	actorID int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	now := s.now().UnixMilli()
	repo, auth, commit, fetchRef, err := s.resolveSource(ctx, row)
	if err != nil {
		return s.recordCheckError(context.WithoutCancel(ctx), row, now, err)
	}
	defer auth.close()

	if commit != "" && (commit == row.SyncedCommit || commit == row.PendingCommit) {
		state := SyncStateOK
		pendingCommit, pendingJSON := "", ""
		if row.PendingCommit == commit {

			state = SyncStatePending
			pendingCommit, pendingJSON = row.PendingCommit, row.PendingJson
		}
		return s.store.SetSkillSourceCheck(context.WithoutCancel(ctx), store.SkillSourceCheckParams{
			SourceID:      row.ID,
			CheckedAt:     now,
			SyncState:     state,
			PendingCommit: pendingCommit,
			PendingJSON:   pendingJSON,
			ProblemsJSON:  row.ProblemsJson,
		})
	}
	skills, problems, err := s.readSource(ctx, row, repo, auth, commit, fetchRef)
	if err != nil {
		return s.recordCheckError(context.WithoutCancel(ctx), row, now, err)
	}
	problems = boundProblems(problems)
	encodedProblems, err := json.Marshal(problems)
	if err != nil {
		return err
	}
	set := pendingSet{Commit: commit, Ref: fetchRef, Skills: skills}
	if row.UpdateMode == UpdateAuto {
		return s.applySet(ctx, row, set, problems, actorID)
	}
	encoded, err := json.Marshal(set)
	if err != nil {
		return err
	}

	if len(encoded)+len(encodedProblems) > maxPendingBytes {
		return s.recordCheckError(
			context.WithoutCancel(ctx),
			row,
			now,
			invalidf(
				"the fetched skills and refused documents exceed %d MiB; narrow the glob",
				maxPendingBytes>>20,
			),
		)
	}
	return s.store.SetSkillSourceCheck(context.WithoutCancel(ctx), store.SkillSourceCheckParams{
		SourceID:      row.ID,
		CheckedAt:     now,
		SyncState:     SyncStatePending,
		PendingCommit: commit,
		PendingJSON:   string(encoded),
		ProblemsJSON:  string(encodedProblems),
	})
}

func (s *Service) recordCheckError(
	ctx context.Context,
	row store.SkillSource,
	now int64,
	cause error,
) error {
	return s.store.SetSkillSourceCheck(ctx, store.SkillSourceCheckParams{
		SourceID:      row.ID,
		CheckedAt:     now,
		SyncState:     SyncStateError,
		SyncError:     cause.Error(),
		PendingCommit: row.PendingCommit,
		PendingJSON:   row.PendingJson,
		ProblemsJSON:  row.ProblemsJson,
	})
}

func (s *Service) resolveSource(
	ctx context.Context,
	row store.SkillSource,
) (Repo, *gitAuth, string, string, error) {
	repo, err := parseRepoURL(row.RepoUrl)
	if err != nil {
		return Repo{}, nil, "", "", err
	}
	credential, err := s.credentialFor(ctx, row, repo)
	if err != nil {
		return Repo{}, nil, "", "", err
	}
	auth, err := authEnv(repo, credential)
	if err != nil {
		return Repo{}, nil, "", "", err
	}
	commit, fetchRef, err := s.git.head(ctx, repo, auth, row.Ref)
	if err != nil {
		auth.close()
		return Repo{}, nil, "", "", err
	}
	return repo, auth, commit, fetchRef, nil
}

func (s *Service) readSource(
	ctx context.Context,
	row store.SkillSource,
	repo Repo,
	auth *gitAuth,
	commit, fetchRef string,
) ([]discoveredSkill, []skillProblem, error) {
	dir, err := os.MkdirTemp("", "cherrypi-source-")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := s.git.export(
		ctx, repo, auth, commit, fetchRef, dir, sparsePrefixOf(row.PathGlob),
	); err != nil {
		return nil, nil, err
	}
	skills, problems, err := discoverSkills(dir, row.PathGlob, row.Slug)
	if err != nil {
		return nil, problems, err
	}
	if err := sourceSkillBudget(skills); err != nil {
		return nil, problems, err
	}
	return skills, problems, nil
}

func (s *Service) credentialFor(
	ctx context.Context,
	row store.SkillSource,
	repo Repo,
) (*credential, error) {
	if !row.CredentialID.Valid {
		if repo.Scheme == "ssh" {
			return nil, invalidf("an SSH repository needs a credential")
		}
		return nil, nil
	}
	stored, err := s.store.GetGitCredential(ctx, row.CredentialID.Int64)
	if err != nil {
		return nil, mapStoreError(err)
	}
	if stored.Host != repo.Host {
		return nil, invalidf(
			"the credential belongs to %s, not %s", stored.Host, repo.Host,
		)
	}
	secret, err := s.seal.Open(stored.SecretSealed)
	if err != nil {
		return nil, fmt.Errorf("the credential %q cannot be opened: %w", stored.Name, err)
	}
	return &credential{
		ID:          stored.ID,
		Name:        stored.Name,
		Kind:        stored.Kind,
		Host:        stored.Host,
		Port:        stored.Port,
		Username:    stored.Username,
		Secret:      secret,
		KnownHosts:  stored.KnownHosts,
		Fingerprint: stored.Fingerprint,
	}, nil
}

func problemsOf(row store.SkillSource) ([]skillProblem, error) {
	problems := []skillProblem{}
	if strings.TrimSpace(row.ProblemsJson) == "" {
		return problems, nil
	}
	if err := json.Unmarshal([]byte(row.ProblemsJson), &problems); err != nil {
		return nil, invalidf("the refused documents are unreadable; sync again")
	}
	return problems, nil
}

func (s *Service) stagedSet(row store.SkillSource) (*pendingSet, error) {
	if strings.TrimSpace(row.PendingJson) == "" {
		return nil, nil
	}
	var set pendingSet
	if err := json.Unmarshal([]byte(row.PendingJson), &set); err != nil {
		return nil, invalidf("the staged update is unreadable; sync again")
	}
	return &set, nil
}

func (s *Service) applySet(
	ctx context.Context,
	row store.SkillSource,
	set pendingSet,
	problems []skillProblem,
	actorID int64,
) error {
	skills := make([]store.SkillSourceSkillParams, 0, len(set.Skills))
	for _, skill := range set.Skills {
		filesJSON, err := json.Marshal(skill.Files)
		if err != nil {
			return err
		}
		skills = append(skills, store.SkillSourceSkillParams{
			Path:                   skill.Path,
			Name:                   skill.Name,
			Description:            skill.Description,
			Body:                   skill.Body,
			FilesJSON:              string(filesJSON),
			DisableModelInvocation: boolInt(skill.DisableModelInvocation),
		})
	}
	installs, err := s.store.ListSkillSourceInstalls(ctx)
	if err != nil {
		return err
	}
	stamps := make([]store.InstallParams, 0, len(installs))
	for _, install := range installs {
		if install.SourceID != row.ID {
			continue
		}
		stamps = append(stamps, store.InstallParams{
			ScopeKind:        install.ScopeKind,
			ScopeID:          install.ScopeID,
			Enabled:          install.Enabled,
			CreatedByActorID: install.CreatedByActorID.Int64,
			CreatedAt:        install.CreatedAt,
		})
	}
	now := s.now().UnixMilli()
	result, err := s.store.ApplySkillSource(ctx, store.ApplySkillSourceParams{
		SourceID:         row.ID,
		Commit:           set.Commit,
		OwnerKind:        row.OwnerKind,
		OwnerID:          row.OwnerID,
		Skills:           skills,
		Installs:         stamps,
		CreatedByActorID: row.CreatedByActorID.Int64,
		Now:              now,
		Revision:         s.nextRevision(),
	})
	if err != nil {
		return err
	}
	conflicts := make([]string, 0, len(result.Conflicts))
	for _, conflict := range result.Conflicts {
		conflicts = append(conflicts, conflict.Name)
	}
	message := ""
	if len(conflicts) > 0 {
		message = fmt.Sprintf(
			"%d skill(s) kept their old entry because the name is taken: %s",
			len(conflicts), strings.Join(conflicts, ", "),
		)
	}
	encodedProblems, err := json.Marshal(problems)
	if err != nil {
		return err
	}
	return s.store.SetSkillSourceApplied(ctx, store.SkillSourceAppliedParams{
		SourceID:     row.ID,
		Commit:       set.Commit,
		AppliedAt:    now,
		CheckedAt:    now,
		SyncError:    message,
		ProblemsJSON: string(encodedProblems),
	})
}

func (s *Service) Poller(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollDue(ctx)
		}
	}
}

func (s *Service) pollDue(ctx context.Context) {
	if s.features == nil {
		return
	}
	enabled, err := s.features.GitSourcesEnabled(ctx)
	if err != nil || !enabled {
		return
	}
	gitCLI, sshCLI := s.probeCLIs(ctx)
	if !gitCLI.OK || !sshCLI.OK {
		return
	}
	rows, err := s.store.ListSkillSources(ctx)
	if err != nil {
		return
	}
	now := s.now().UnixMilli()
	slots := make(chan struct{}, 4)
	for _, row := range rows {
		if row.Paused != 0 || row.PollIntervalS <= 0 {
			continue
		}
		if row.CheckedAt+row.PollIntervalS*1000 > now {
			continue
		}
		if !s.beginSync(row.ID) {
			continue
		}
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			s.endSync(row.ID)
			return
		}
		s.pollWG.Add(1)
		go func(row store.SkillSource) {
			defer s.pollWG.Done()
			defer func() { <-slots }()
			defer s.endSync(row.ID)
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			_ = s.runCheck(checkCtx, row, 0)
		}(row)
	}
}

func (s *Service) Wait() { s.pollWG.Wait() }

func mapUniqueSource(err error) error {
	return uniqueErr(err, "a source with this slug already exists")
}

func mapUniqueCredential(err error) error {
	return uniqueErr(err, "a credential with this name already exists")
}

func uniqueErr(err error, message string) error {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return invalidf("%s", message)
	}
	return err
}

func sqlNullID(id int64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}
