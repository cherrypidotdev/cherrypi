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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"
)

// LDAPUser is the identity resolved from a successful LDAP bind.
type LDAPUser struct {
	Username string
	Email    string
	Name     string
}

// BindLDAP authenticates a username/password against an LDAP provider and
// returns the user's directory attributes.
//
// When BindDN contains the literal `{username}`, the provider binds directly as
// the substituted DN (glauth-style). Otherwise it binds with the service
// account, searches BaseDN for UserFilter and re-binds as the user's DN, which
// verifies the password. A search that matches nothing returns
// ErrLDAPUserNotFound; a failed bind returns ErrInvalidCredentials.
func (s *Service) BindLDAP(
	ctx context.Context,
	p LDAPProvider,
	username, password string,
) (LDAPUser, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return LDAPUser{}, ErrInvalidCredentials
	}

	conn, err := dialLDAP(p)
	if err != nil {
		return LDAPUser{}, err
	}
	defer func() { _ = conn.Close() }()

	if strings.Contains(p.BindDN, "{username}") {
		// Direct bind: the DN template carries the username.
		userDN := strings.ReplaceAll(p.BindDN, "{username}", ldap.EscapeDN(username))
		if err := conn.Bind(userDN, password); err != nil {
			return LDAPUser{}, ldapError(err, ErrInvalidCredentials)
		}
	} else {
		// Service bind, search for the user's DN, then re-bind as the user.
		bindPassword, err := s.ldapBindPassword(ctx, p.ID)
		if err != nil {
			return LDAPUser{}, err
		}
		if p.BindDN != "" {
			if err := conn.Bind(p.BindDN, bindPassword); err != nil {
				return LDAPUser{}, ldapError(err, ErrLDAPUnavailable)
			}
		}
		entry, err := searchLDAPEntry(conn, p, username)
		if err != nil {
			return LDAPUser{}, err
		}
		if err := conn.Bind(entry.DN, password); err != nil {
			return LDAPUser{}, ldapError(err, ErrInvalidCredentials)
		}
	}

	entry, err := searchLDAPEntry(conn, p, username)
	if err != nil {
		// A successful bind is enough when the directory will not let the
		// bound user read its own entry; synthesize from the input.
		return LDAPUser{Username: username, Name: username}, nil
	}
	return ldapUserFromEntry(entry, p, username), nil
}

func dialLDAP(p LDAPProvider) (*ldap.Conn, error) {
	address := fmt.Sprintf("%s:%d", p.Host, p.Port)
	scheme := "ldap"
	if p.Encryption == LDAPLDAPS {
		scheme = "ldaps"
	}
	opts := []ldap.DialOpt{}
	if p.Encryption == LDAPLDAPS || p.Encryption == LDAPStartTLS {
		opts = append(opts, ldap.DialWithTLSConfig(&tls.Config{
			InsecureSkipVerify: p.Insecure, //nolint:gosec // explicit test/dev opt-in
			ServerName:         p.Host,
		}))
	}
	conn, err := ldap.DialURL(scheme+"://"+address, opts...)
	if err != nil {
		return nil, ldapError(err, ErrLDAPUnavailable)
	}
	if p.Encryption == LDAPStartTLS {
		if err := conn.StartTLS(&tls.Config{
			InsecureSkipVerify: p.Insecure, //nolint:gosec // explicit test/dev opt-in
			ServerName:         p.Host,
		}); err != nil {
			_ = conn.Close()
			return nil, ldapError(err, ErrLDAPUnavailable)
		}
	}
	return conn, nil
}

func searchLDAPEntry(conn *ldap.Conn, p LDAPProvider, username string) (*ldap.Entry, error) {
	filter := strings.ReplaceAll(
		p.UserFilter,
		"{username}",
		ldap.EscapeFilter(username),
	)
	attrs := uniqueStrings(p.UsernameAttr, p.EmailAttr, p.NameAttr)
	req := ldap.NewSearchRequest(
		p.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		2, 0, false,
		filter,
		attrs,
		nil,
	)
	result, err := conn.Search(req)
	if err != nil {
		return nil, ldapError(err, ErrLDAPUnavailable)
	}
	if len(result.Entries) == 0 {
		return nil, ErrLDAPUserNotFound
	}
	return result.Entries[0], nil
}

func (s *Service) ldapBindPassword(ctx context.Context, id int64) (string, error) {
	row, err := s.store.GetLDAPProviderByID(ctx, id)
	if err != nil {
		return "", err
	}
	return row.BindPassword, nil
}

func ldapUserFromEntry(entry *ldap.Entry, p LDAPProvider, fallback string) LDAPUser {
	username := entry.GetAttributeValue(p.UsernameAttr)
	if username == "" {
		username = fallback
	}
	name := entry.GetAttributeValue(p.NameAttr)
	if name == "" {
		name = username
	}
	return LDAPUser{
		Username: username,
		Email:    NormalizeEmail(entry.GetAttributeValue(p.EmailAttr)),
		Name:     name,
	}
}

// ldapError classifies a go-ldap error, defaulting to fallback.
func ldapError(err error, fallback error) error {
	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) {
		switch ldapErr.ResultCode {
		case ldap.LDAPResultInvalidCredentials:
			return ErrInvalidCredentials
		case ldap.LDAPResultNoSuchObject, ldap.LDAPResultEntryAlreadyExists:
			return ErrLDAPUserNotFound
		}
	}
	return fallback
}

func uniqueStrings(values ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
