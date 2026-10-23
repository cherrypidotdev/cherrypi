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

package accounts

import (
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
)

// Provider kinds. OIDC uses discovery (and verifies the ID token); OAuth2 is
// the generic shim over explicit endpoints and JSON paths.
const (
	ProviderOIDC   = "oidc"
	ProviderOAuth2 = "oauth2"
)

// LDAP encryption modes.
const (
	LDAPNone     = "none"
	LDAPStartTLS = "starttls"
	LDAPLDAPS    = "ldaps"
)

// Federation errors.
var (
	ErrInvalidProvider    = errors.New("invalid provider configuration")
	ErrProviderSlugInUse  = errors.New("provider slug is already taken")
	ErrProviderDisabled   = errors.New("provider is disabled")
	ErrIdentityLinked     = errors.New("identity is already linked to another account")
	ErrIdentityNotFound   = errors.New("identity is not linked")
	ErrIdentityExists     = errors.New("identity is already linked to this account")
	ErrLDAPUnavailable    = errors.New("LDAP server is unavailable")
	ErrLDAPUserNotFound   = errors.New("LDAP user not found")
	ErrExternalNotAllowed = errors.New("no account is linked to this identity")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// IdentityProvider is the public view of an OIDC/OAuth2 provider. The client
// secret is never exposed; HasClientSecret reports whether one is stored.
type IdentityProvider struct {
	ID                    int64
	Kind                  string
	Slug                  string
	Name                  string
	Enabled               bool
	Issuer                string
	AuthorizationEndpoint string
	TokenEndpoint         string
	UserinfoEndpoint      string
	JWKSURI               string
	ClientID              string
	HasClientSecret       bool
	Scopes                string
	SubjectPath           string
	EmailPath             string
	NamePath              string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IdentityProviderInput is the admin write model. An empty ClientSecret keeps
// the stored one on update.
type IdentityProviderInput struct {
	Kind                  string
	Slug                  string
	Name                  string
	Enabled               bool
	Issuer                string
	AuthorizationEndpoint string
	TokenEndpoint         string
	UserinfoEndpoint      string
	JWKSURI               string
	ClientID              string
	ClientSecret          string
	Scopes                string
	SubjectPath           string
	EmailPath             string
	NamePath              string
}

// ExternalIdentity is a provider identity linked to an actor.
type ExternalIdentity struct {
	ID           int64
	ProviderID   int64
	ProviderSlug string
	ProviderName string
	Kind         string
	Subject      string
	Email        string
	CreatedAt    time.Time
}

// LDAPProvider is the public view of an LDAP provider. The bind password is
// never exposed; HasBindPassword reports whether one is stored.
type LDAPProvider struct {
	ID              int64
	Slug            string
	Name            string
	Enabled         bool
	Host            string
	Port            int64
	Encryption      string
	Insecure        bool
	BaseDN          string
	BindDN          string
	HasBindPassword bool
	UserFilter      string
	UsernameAttr    string
	EmailAttr       string
	NameAttr        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// LDAPProviderInput is the admin write model. An empty BindPassword keeps the
// stored one on update.
type LDAPProviderInput struct {
	Slug         string
	Name         string
	Enabled      bool
	Host         string
	Port         int64
	Encryption   string
	Insecure     bool
	BaseDN       string
	BindDN       string
	BindPassword string
	UserFilter   string
	UsernameAttr string
	EmailAttr    string
	NameAttr     string
}

// LDAPIdentity is an LDAP username linked to an actor.
type LDAPIdentity struct {
	ID           int64
	ProviderID   int64
	ProviderSlug string
	ProviderName string
	Username     string
	CreatedAt    time.Time
}

// ExternalUser is the normalized result of a provider login.
type ExternalUser struct {
	Subject string
	Email   string
	Name    string
	// Username is the provider's username (OIDC preferred_username, or the
	// generic OAuth2 shim's name path). It matches provider-scoped invites.
	Username string
}

// --- Identity provider CRUD -------------------------------------------------

// ListIdentityProviders returns every configured provider, admin-only.
func (s *Service) ListIdentityProviders(ctx context.Context) ([]IdentityProvider, error) {
	rows, err := s.store.ListIdentityProviders(ctx)
	if err != nil {
		return nil, err
	}
	return identityProvidersFromRows(rows), nil
}

// ListEnabledIdentityProviders returns the providers the login form may offer.
func (s *Service) ListEnabledIdentityProviders(
	ctx context.Context,
) ([]IdentityProvider, error) {
	rows, err := s.store.ListEnabledIdentityProviders(ctx)
	if err != nil {
		return nil, err
	}
	return identityProvidersFromRows(rows), nil
}

// GetIdentityProvider returns one provider by id.
func (s *Service) GetIdentityProvider(ctx context.Context, id int64) (IdentityProvider, error) {
	row, err := s.store.GetIdentityProviderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return IdentityProvider{}, ErrNotFound
		}
		return IdentityProvider{}, err
	}
	return identityProviderFromRow(row), nil
}

// GetIdentityProviderBySlug returns one provider by slug.
func (s *Service) GetIdentityProviderBySlug(
	ctx context.Context,
	slug string,
) (IdentityProvider, error) {
	row, err := s.store.GetIdentityProviderBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return IdentityProvider{}, ErrNotFound
		}
		return IdentityProvider{}, err
	}
	return identityProviderFromRow(row), nil
}

// CreateIdentityProvider validates and stores a provider.
func (s *Service) CreateIdentityProvider(
	ctx context.Context,
	in IdentityProviderInput,
) (IdentityProvider, error) {
	in, err := s.normalizeIdentityProvider(ctx, 0, in)
	if err != nil {
		return IdentityProvider{}, err
	}
	if _, err := s.store.GetIdentityProviderBySlug(ctx, in.Slug); err == nil {
		return IdentityProvider{}, ErrProviderSlugInUse
	} else if !errors.Is(err, sql.ErrNoRows) {
		return IdentityProvider{}, err
	}

	now := s.now()
	row, err := s.store.CreateIdentityProvider(ctx, store.CreateIdentityProviderParams{
		Kind:                  in.Kind,
		Slug:                  in.Slug,
		Name:                  in.Name,
		Enabled:               boolToInt(in.Enabled),
		Issuer:                in.Issuer,
		AuthorizationEndpoint: in.AuthorizationEndpoint,
		TokenEndpoint:         in.TokenEndpoint,
		UserinfoEndpoint:      in.UserinfoEndpoint,
		JwksUri:               in.JWKSURI,
		ClientID:              in.ClientID,
		ClientSecret:          in.ClientSecret,
		Scopes:                in.Scopes,
		SubjectPath:           in.SubjectPath,
		EmailPath:             in.EmailPath,
		NamePath:              in.NamePath,
		CreatedAt:             now.UnixMilli(),
		UpdatedAt:             now.UnixMilli(),
	})
	if err != nil {
		if isUniqueError(err) {
			return IdentityProvider{}, ErrProviderSlugInUse
		}
		return IdentityProvider{}, err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindProviderCreate, "slug",
		"", in.Slug, "Created identity provider "+in.Name, nil)
	return identityProviderFromRow(row), nil
}

// UpdateIdentityProvider validates and replaces a provider. An empty
// ClientSecret keeps the stored one.
func (s *Service) UpdateIdentityProvider(
	ctx context.Context,
	id int64,
	in IdentityProviderInput,
) (IdentityProvider, error) {
	current, err := s.store.GetIdentityProviderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return IdentityProvider{}, ErrNotFound
		}
		return IdentityProvider{}, err
	}
	if in.ClientSecret == "" {
		in.ClientSecret = current.ClientSecret
	}
	in, err = s.normalizeIdentityProvider(ctx, id, in)
	if err != nil {
		return IdentityProvider{}, err
	}
	if err := s.store.UpdateIdentityProvider(ctx, store.UpdateIdentityProviderParams{
		Kind:                  in.Kind,
		Slug:                  in.Slug,
		Name:                  in.Name,
		Enabled:               boolToInt(in.Enabled),
		Issuer:                in.Issuer,
		AuthorizationEndpoint: in.AuthorizationEndpoint,
		TokenEndpoint:         in.TokenEndpoint,
		UserinfoEndpoint:      in.UserinfoEndpoint,
		JwksUri:               in.JWKSURI,
		ClientID:              in.ClientID,
		ClientSecret:          in.ClientSecret,
		Scopes:                in.Scopes,
		SubjectPath:           in.SubjectPath,
		EmailPath:             in.EmailPath,
		NamePath:              in.NamePath,
		UpdatedAt:             s.now().UnixMilli(),
		ID:                    id,
	}); err != nil {
		if isUniqueError(err) {
			return IdentityProvider{}, ErrProviderSlugInUse
		}
		return IdentityProvider{}, err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindProviderUpdate, "slug",
		"", in.Slug, "Updated identity provider "+in.Name, nil)
	return s.GetIdentityProvider(ctx, id)
}

// DeleteIdentityProvider removes a provider and its external identities.
func (s *Service) DeleteIdentityProvider(ctx context.Context, id int64) error {
	if err := s.store.DeleteIdentityProvider(ctx, id); err != nil {
		return err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindProviderDelete, "", "", "",
		"Deleted an identity provider", nil)
	return nil
}

func (s *Service) normalizeIdentityProvider(
	ctx context.Context,
	id int64,
	in IdentityProviderInput,
) (IdentityProviderInput, error) {
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Name = strings.TrimSpace(in.Name)
	in.Issuer = strings.TrimRight(strings.TrimSpace(in.Issuer), "/")
	in.AuthorizationEndpoint = strings.TrimSpace(in.AuthorizationEndpoint)
	in.TokenEndpoint = strings.TrimSpace(in.TokenEndpoint)
	in.UserinfoEndpoint = strings.TrimSpace(in.UserinfoEndpoint)
	in.JWKSURI = strings.TrimSpace(in.JWKSURI)
	in.ClientID = strings.TrimSpace(in.ClientID)
	in.Scopes = strings.TrimSpace(in.Scopes)
	in.SubjectPath = strings.TrimSpace(in.SubjectPath)
	in.EmailPath = strings.TrimSpace(in.EmailPath)
	in.NamePath = strings.TrimSpace(in.NamePath)

	if in.Kind != ProviderOIDC && in.Kind != ProviderOAuth2 {
		return in, ErrInvalidProvider
	}
	if !slugPattern.MatchString(in.Slug) || in.Name == "" {
		return in, ErrInvalidProvider
	}
	if in.ClientID == "" {
		return in, ErrInvalidProvider
	}
	if in.Scopes == "" {
		if in.Kind == ProviderOIDC {
			in.Scopes = "openid email profile"
		} else {
			in.Scopes = "read"
		}
	}
	switch in.Kind {
	case ProviderOIDC:
		if in.Issuer == "" {
			return in, ErrInvalidProvider
		}
	case ProviderOAuth2:
		if in.AuthorizationEndpoint == "" || in.TokenEndpoint == "" ||
			in.UserinfoEndpoint == "" || in.SubjectPath == "" {
			return in, ErrInvalidProvider
		}
	}
	// A rename onto another provider's slug is rejected up front for a clear
	// error; the unique index is the real guard.
	if other, err := s.store.GetIdentityProviderBySlug(ctx, in.Slug); err == nil &&
		other.ID != id {
		return in, ErrProviderSlugInUse
	}
	return in, nil
}

func identityProvidersFromRows(rows []store.IdentityProvider) []IdentityProvider {
	out := make([]IdentityProvider, len(rows))
	for i, row := range rows {
		out[i] = identityProviderFromRow(row)
	}
	return out
}

func identityProviderFromRow(row store.IdentityProvider) IdentityProvider {
	return IdentityProvider{
		ID:                    row.ID,
		Kind:                  row.Kind,
		Slug:                  row.Slug,
		Name:                  row.Name,
		Enabled:               row.Enabled != 0,
		Issuer:                row.Issuer,
		AuthorizationEndpoint: row.AuthorizationEndpoint,
		TokenEndpoint:         row.TokenEndpoint,
		UserinfoEndpoint:      row.UserinfoEndpoint,
		JWKSURI:               row.JwksUri,
		ClientID:              row.ClientID,
		HasClientSecret:       row.ClientSecret != "",
		Scopes:                row.Scopes,
		SubjectPath:           row.SubjectPath,
		EmailPath:             row.EmailPath,
		NamePath:              row.NamePath,
		CreatedAt:             time.UnixMilli(row.CreatedAt),
		UpdatedAt:             time.UnixMilli(row.UpdatedAt),
	}
}

// --- External identities ----------------------------------------------------

// ListIdentities returns the actor's linked OIDC/OAuth2 identities.
func (s *Service) ListIdentities(
	ctx context.Context,
	actorID ids.ActorID,
) ([]ExternalIdentity, error) {
	rows, err := s.store.ListExternalCredentialsByActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]ExternalIdentity, 0, len(rows))
	for _, row := range rows {
		provider, err := s.store.GetIdentityProviderByID(ctx, row.ProviderID)
		if err != nil {
			continue
		}
		out = append(out, externalIdentityFromRow(row, provider))
	}
	return out, nil
}

// ResolveExternalActor returns the account linked to a provider identity.
// Unlinked identities are rejected (invite-only): ErrExternalNotAllowed.
func (s *Service) ResolveExternalActor(
	ctx context.Context,
	providerID int64,
	subject string,
) (Actor, error) {
	row, err := s.store.GetExternalCredential(ctx, providerID, subject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Actor{}, ErrExternalNotAllowed
		}
		return Actor{}, err
	}
	return s.GetActor(ctx, ids.ActorID(row.ActorID))
}

// LinkIdentity links a provider identity to the actor. A subject already
// linked to another actor is rejected; relinking the same pair is a no-op.
func (s *Service) LinkIdentity(
	ctx context.Context,
	actorID ids.ActorID,
	providerID int64,
	subject, email string,
) (ExternalIdentity, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ExternalIdentity{}, ErrInvalidProvider
	}
	existing, err := s.store.GetExternalCredential(ctx, providerID, subject)
	if err == nil {
		if ids.ActorID(existing.ActorID) != actorID {
			return ExternalIdentity{}, ErrIdentityLinked
		}
		return s.externalIdentity(ctx, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ExternalIdentity{}, err
	}

	now := s.now()
	row, err := s.store.CreateExternalCredential(ctx, store.CreateExternalCredentialParams{
		ActorID:    int64(actorID),
		ProviderID: providerID,
		Subject:    subject,
		Email:      NormalizeEmail(email),
		CreatedAt:  now.UnixMilli(),
		UpdatedAt:  now.UnixMilli(),
	})
	if err != nil {
		if isUniqueError(err) {
			return ExternalIdentity{}, ErrIdentityLinked
		}
		return ExternalIdentity{}, err
	}
	s.auditUserAs(ctx, actorID, actorID, audit.KindIdentityLink, "subject",
		"", subject, "Linked an external identity", nil)
	return s.externalIdentity(ctx, row)
}

// UnlinkIdentity removes one of the actor's linked identities.
func (s *Service) UnlinkIdentity(ctx context.Context, actorID ids.ActorID, id int64) error {
	if err := s.store.DeleteExternalCredential(ctx, id, actorID); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindIdentityUnlink, "", "", "",
		"Unlinked an external identity", nil)
	return nil
}

func (s *Service) externalIdentity(
	ctx context.Context,
	row store.ExternalCredential,
) (ExternalIdentity, error) {
	provider, err := s.store.GetIdentityProviderByID(ctx, row.ProviderID)
	if err != nil {
		return ExternalIdentity{}, err
	}
	return externalIdentityFromRow(row, provider), nil
}

func externalIdentityFromRow(
	row store.ExternalCredential,
	provider store.IdentityProvider,
) ExternalIdentity {
	return ExternalIdentity{
		ID:           row.ID,
		ProviderID:   row.ProviderID,
		ProviderSlug: provider.Slug,
		ProviderName: provider.Name,
		Kind:         provider.Kind,
		Subject:      row.Subject,
		Email:        row.Email,
		CreatedAt:    time.UnixMilli(row.CreatedAt),
	}
}

// --- LDAP providers ---------------------------------------------------------

// ListLDAPProviders returns every configured LDAP provider, admin-only.
func (s *Service) ListLDAPProviders(ctx context.Context) ([]LDAPProvider, error) {
	rows, err := s.store.ListLDAPProviders(ctx)
	if err != nil {
		return nil, err
	}
	return ldapProvidersFromRows(rows), nil
}

// ListEnabledLDAPProviders returns the providers the login form may offer.
func (s *Service) ListEnabledLDAPProviders(ctx context.Context) ([]LDAPProvider, error) {
	rows, err := s.store.ListEnabledLDAPProviders(ctx)
	if err != nil {
		return nil, err
	}
	return ldapProvidersFromRows(rows), nil
}

// GetLDAPProvider returns one LDAP provider by id.
func (s *Service) GetLDAPProvider(ctx context.Context, id int64) (LDAPProvider, error) {
	row, err := s.store.GetLDAPProviderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LDAPProvider{}, ErrNotFound
		}
		return LDAPProvider{}, err
	}
	return ldapProviderFromRow(row), nil
}

// GetLDAPProviderBySlug returns one LDAP provider by slug.
func (s *Service) GetLDAPProviderBySlug(ctx context.Context, slug string) (LDAPProvider, error) {
	row, err := s.store.GetLDAPProviderBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LDAPProvider{}, ErrNotFound
		}
		return LDAPProvider{}, err
	}
	return ldapProviderFromRow(row), nil
}

// CreateLDAPProvider validates and stores an LDAP provider.
func (s *Service) CreateLDAPProvider(
	ctx context.Context,
	in LDAPProviderInput,
) (LDAPProvider, error) {
	in, err := s.normalizeLDAPProvider(ctx, 0, in)
	if err != nil {
		return LDAPProvider{}, err
	}
	if _, err := s.store.GetLDAPProviderBySlug(ctx, in.Slug); err == nil {
		return LDAPProvider{}, ErrProviderSlugInUse
	} else if !errors.Is(err, sql.ErrNoRows) {
		return LDAPProvider{}, err
	}

	now := s.now()
	row, err := s.store.CreateLDAPProvider(ctx, store.CreateLDAPProviderParams{
		Slug:         in.Slug,
		Name:         in.Name,
		Enabled:      boolToInt(in.Enabled),
		Host:         in.Host,
		Port:         in.Port,
		Encryption:   in.Encryption,
		Insecure:     boolToInt(in.Insecure),
		BaseDn:       in.BaseDN,
		BindDn:       in.BindDN,
		BindPassword: in.BindPassword,
		UserFilter:   in.UserFilter,
		UsernameAttr: in.UsernameAttr,
		EmailAttr:    in.EmailAttr,
		NameAttr:     in.NameAttr,
		CreatedAt:    now.UnixMilli(),
		UpdatedAt:    now.UnixMilli(),
	})
	if err != nil {
		if isUniqueError(err) {
			return LDAPProvider{}, ErrProviderSlugInUse
		}
		return LDAPProvider{}, err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindProviderCreate, "ldap_slug",
		"", in.Slug, "Created LDAP provider "+in.Name, nil)
	return ldapProviderFromRow(row), nil
}

// UpdateLDAPProvider validates and replaces an LDAP provider. An empty
// BindPassword keeps the stored one.
func (s *Service) UpdateLDAPProvider(
	ctx context.Context,
	id int64,
	in LDAPProviderInput,
) (LDAPProvider, error) {
	current, err := s.store.GetLDAPProviderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LDAPProvider{}, ErrNotFound
		}
		return LDAPProvider{}, err
	}
	if in.BindPassword == "" {
		in.BindPassword = current.BindPassword
	}
	in, err = s.normalizeLDAPProvider(ctx, id, in)
	if err != nil {
		return LDAPProvider{}, err
	}
	if err := s.store.UpdateLDAPProvider(ctx, store.UpdateLDAPProviderParams{
		Slug:         in.Slug,
		Name:         in.Name,
		Enabled:      boolToInt(in.Enabled),
		Host:         in.Host,
		Port:         in.Port,
		Encryption:   in.Encryption,
		Insecure:     boolToInt(in.Insecure),
		BaseDn:       in.BaseDN,
		BindDn:       in.BindDN,
		BindPassword: in.BindPassword,
		UserFilter:   in.UserFilter,
		UsernameAttr: in.UsernameAttr,
		EmailAttr:    in.EmailAttr,
		NameAttr:     in.NameAttr,
		UpdatedAt:    s.now().UnixMilli(),
		ID:           id,
	}); err != nil {
		if isUniqueError(err) {
			return LDAPProvider{}, ErrProviderSlugInUse
		}
		return LDAPProvider{}, err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindProviderUpdate, "ldap_slug",
		"", in.Slug, "Updated LDAP provider "+in.Name, nil)
	return s.GetLDAPProvider(ctx, id)
}

// DeleteLDAPProvider removes a provider and its linked credentials.
func (s *Service) DeleteLDAPProvider(ctx context.Context, id int64) error {
	if err := s.store.DeleteLDAPProvider(ctx, id); err != nil {
		return err
	}
	s.auditUser(ctx, audit.ActorID(ctx), audit.KindProviderDelete, "ldap",
		"", "", "Deleted an LDAP provider", nil)
	return nil
}

func (s *Service) normalizeLDAPProvider(
	ctx context.Context,
	id int64,
	in LDAPProviderInput,
) (LDAPProviderInput, error) {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Name = strings.TrimSpace(in.Name)
	in.Host = strings.TrimSpace(in.Host)
	in.Encryption = strings.ToLower(strings.TrimSpace(in.Encryption))
	in.BaseDN = strings.TrimSpace(in.BaseDN)
	in.BindDN = strings.TrimSpace(in.BindDN)
	in.UserFilter = strings.TrimSpace(in.UserFilter)
	in.UsernameAttr = strings.TrimSpace(in.UsernameAttr)
	in.EmailAttr = strings.TrimSpace(in.EmailAttr)
	in.NameAttr = strings.TrimSpace(in.NameAttr)

	if in.Encryption == "" {
		in.Encryption = LDAPNone
	}
	switch in.Encryption {
	case LDAPNone, LDAPStartTLS, LDAPLDAPS:
	default:
		return in, ErrInvalidProvider
	}
	if in.Port == 0 {
		if in.Encryption == LDAPLDAPS {
			in.Port = 636
		} else {
			in.Port = 389
		}
	}
	if in.UserFilter == "" {
		in.UserFilter = "(uid={username})"
	}
	if in.UsernameAttr == "" {
		in.UsernameAttr = "uid"
	}
	if !slugPattern.MatchString(in.Slug) || in.Name == "" || in.Host == "" ||
		in.BaseDN == "" {
		return in, ErrInvalidProvider
	}
	if other, err := s.store.GetLDAPProviderBySlug(ctx, in.Slug); err == nil &&
		other.ID != id {
		return in, ErrProviderSlugInUse
	}
	return in, nil
}

func ldapProvidersFromRows(rows []store.LdapProvider) []LDAPProvider {
	out := make([]LDAPProvider, len(rows))
	for i, row := range rows {
		out[i] = ldapProviderFromRow(row)
	}
	return out
}

func ldapProviderFromRow(row store.LdapProvider) LDAPProvider {
	return LDAPProvider{
		ID:              row.ID,
		Slug:            row.Slug,
		Name:            row.Name,
		Enabled:         row.Enabled != 0,
		Host:            row.Host,
		Port:            row.Port,
		Encryption:      row.Encryption,
		Insecure:        row.Insecure != 0,
		BaseDN:          row.BaseDn,
		BindDN:          row.BindDn,
		HasBindPassword: row.BindPassword != "",
		UserFilter:      row.UserFilter,
		UsernameAttr:    row.UsernameAttr,
		EmailAttr:       row.EmailAttr,
		NameAttr:        row.NameAttr,
		CreatedAt:       time.UnixMilli(row.CreatedAt),
		UpdatedAt:       time.UnixMilli(row.UpdatedAt),
	}
}

// --- LDAP credentials -------------------------------------------------------

// ListLDAPIdentities returns the actor's linked LDAP usernames.
func (s *Service) ListLDAPIdentities(
	ctx context.Context,
	actorID ids.ActorID,
) ([]LDAPIdentity, error) {
	rows, err := s.store.ListLDAPCredentialsByActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]LDAPIdentity, 0, len(rows))
	for _, row := range rows {
		provider, err := s.store.GetLDAPProviderByID(ctx, row.ProviderID)
		if err != nil {
			continue
		}
		out = append(out, ldapIdentityFromRow(row, provider))
	}
	return out, nil
}

// ResolveLDAPActor returns the account linked to an LDAP username.
func (s *Service) ResolveLDAPActor(
	ctx context.Context,
	providerID int64,
	username string,
) (Actor, error) {
	row, err := s.store.GetLDAPCredential(ctx, providerID, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Actor{}, ErrExternalNotAllowed
		}
		return Actor{}, err
	}
	return s.GetActor(ctx, ids.ActorID(row.ActorID))
}

// LinkLDAPIdentity links an LDAP username to the actor after a successful bind.
func (s *Service) LinkLDAPIdentity(
	ctx context.Context,
	actorID ids.ActorID,
	providerID int64,
	username string,
) (LDAPIdentity, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return LDAPIdentity{}, ErrInvalidProvider
	}
	existing, err := s.store.GetLDAPCredential(ctx, providerID, username)
	if err == nil {
		if ids.ActorID(existing.ActorID) != actorID {
			return LDAPIdentity{}, ErrIdentityLinked
		}
		return s.ldapIdentity(ctx, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return LDAPIdentity{}, err
	}

	now := s.now()
	row, err := s.store.CreateLDAPCredential(ctx, store.CreateLDAPCredentialParams{
		ActorID:    int64(actorID),
		ProviderID: providerID,
		Username:   username,
		CreatedAt:  now.UnixMilli(),
		UpdatedAt:  now.UnixMilli(),
	})
	if err != nil {
		if isUniqueError(err) {
			return LDAPIdentity{}, ErrIdentityLinked
		}
		return LDAPIdentity{}, err
	}
	s.auditUserAs(ctx, actorID, actorID, audit.KindIdentityLink, "ldap",
		"", username, "Linked an LDAP identity", nil)
	return s.ldapIdentity(ctx, row)
}

// UnlinkLDAPIdentity removes one of the actor's linked LDAP usernames.
func (s *Service) UnlinkLDAPIdentity(ctx context.Context, actorID ids.ActorID, id int64) error {
	if err := s.store.DeleteLDAPCredential(ctx, id, actorID); err != nil {
		return err
	}
	s.auditUser(ctx, actorID, audit.KindIdentityUnlink, "ldap", "", "",
		"Unlinked an LDAP identity", nil)
	return nil
}

func (s *Service) ldapIdentity(
	ctx context.Context,
	row store.LdapCredential,
) (LDAPIdentity, error) {
	provider, err := s.store.GetLDAPProviderByID(ctx, row.ProviderID)
	if err != nil {
		return LDAPIdentity{}, err
	}
	return ldapIdentityFromRow(row, provider), nil
}

func ldapIdentityFromRow(
	row store.LdapCredential,
	provider store.LdapProvider,
) LDAPIdentity {
	return LDAPIdentity{
		ID:           row.ID,
		ProviderID:   row.ProviderID,
		ProviderSlug: provider.Slug,
		ProviderName: provider.Name,
		Username:     row.Username,
		CreatedAt:    time.UnixMilli(row.CreatedAt),
	}
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
