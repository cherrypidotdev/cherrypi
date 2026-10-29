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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var ErrProviderUnavailable = errors.New("identity provider is unavailable")

type AuthRequest struct {
	URL      string
	State    string
	Nonce    string
	Verifier string
}

func (s *Service) BeginOIDC(
	ctx context.Context,
	p IdentityProvider,
	redirectURI string,
) (AuthRequest, error) {
	cfg, _, err := s.oidcOAuthConfig(ctx, p, redirectURI)
	if err != nil {
		return AuthRequest{}, err
	}
	state := randomToken(24)
	nonce := randomToken(24)
	verifier := oauth2.GenerateVerifier()
	url := cfg.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	)
	return AuthRequest{URL: url, State: state, Nonce: nonce, Verifier: verifier}, nil
}

func (s *Service) FinishOIDC(
	ctx context.Context,
	p IdentityProvider,
	code, nonce, verifier, redirectURI string,
) (ExternalUser, error) {
	cfg, oidcProvider, err := s.oidcOAuthConfig(ctx, p, redirectURI)
	if err != nil {
		return ExternalUser{}, err
	}
	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return ExternalUser{}, fmt.Errorf("%w: token exchange: %v", ErrProviderUnavailable, err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return ExternalUser{}, fmt.Errorf("%w: no id_token in response", ErrProviderUnavailable)
	}
	idToken, err := oidcProvider.Verifier(&oidc.Config{ClientID: p.ClientID}).
		Verify(ctx, rawIDToken)
	if err != nil {
		return ExternalUser{}, fmt.Errorf("%w: verify id_token: %v", ErrProviderUnavailable, err)
	}
	if nonce != "" && idToken.Nonce != nonce {
		return ExternalUser{}, fmt.Errorf("%w: nonce mismatch", ErrProviderUnavailable)
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return ExternalUser{}, fmt.Errorf("%w: decode claims: %v", ErrProviderUnavailable, err)
	}
	email := claims.Email
	if email == "" {

		email = s.userinfoField(ctx, token, p.UserinfoEndpoint, "email")
	}
	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}
	if name == "" && email != "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	return ExternalUser{
		Subject:       idToken.Subject,
		Email:         NormalizeEmail(email),
		EmailVerified: claims.EmailVerified,
		Name:          name,
		Username:      claims.PreferredUsername,
	}, nil
}

func (s *Service) BeginOAuth2(p IdentityProvider, redirectURI string) (AuthRequest, error) {
	state := randomToken(24)
	verifier := oauth2.GenerateVerifier()
	cfg := s.oauth2Config(p, redirectURI)
	url := cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return AuthRequest{URL: url, State: state, Verifier: verifier}, nil
}

func (s *Service) FinishOAuth2(
	ctx context.Context,
	p IdentityProvider,
	code, verifier, redirectURI string,
) (ExternalUser, error) {
	cfg := s.oauth2Config(p, redirectURI)
	secret, err := s.providerSecret(ctx, p.ID)
	if err != nil {
		return ExternalUser{}, err
	}
	cfg.ClientSecret = secret
	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return ExternalUser{}, fmt.Errorf("%w: token exchange: %v", ErrProviderUnavailable, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserinfoEndpoint, nil)
	if err != nil {
		return ExternalUser{}, fmt.Errorf("%w: userinfo request: %v", ErrProviderUnavailable, err)
	}
	token.SetAuthHeader(req)
	resp, err := cfg.Client(ctx, token).Do(req)
	if err != nil {
		return ExternalUser{}, fmt.Errorf("%w: userinfo: %v", ErrProviderUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ExternalUser{}, fmt.Errorf("%w: read userinfo: %v", ErrProviderUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		return ExternalUser{}, fmt.Errorf(
			"%w: userinfo returned %d", ErrProviderUnavailable, resp.StatusCode,
		)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ExternalUser{}, fmt.Errorf("%w: decode userinfo: %v", ErrProviderUnavailable, err)
	}
	subject, _ := lookupJSONPath(payload, p.SubjectPath)
	email, _ := lookupJSONPath(payload, p.EmailPath)
	name, _ := lookupJSONPath(payload, p.NamePath)
	if subject == "" {
		return ExternalUser{}, fmt.Errorf(
			"%w: userinfo has no value at subject path %q",
			ErrProviderUnavailable, p.SubjectPath,
		)
	}
	if name == "" && email != "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	return ExternalUser{
		Subject:  subject,
		Email:    NormalizeEmail(email),
		Name:     name,
		Username: name,
	}, nil
}

func (s *Service) oidcOAuthConfig(
	ctx context.Context,
	p IdentityProvider,
	redirectURI string,
) (*oauth2.Config, *oidc.Provider, error) {
	discovered, err := oidc.NewProvider(ctx, p.Issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: discovery: %v", ErrProviderUnavailable, err)
	}
	endpoint := discovered.Endpoint()
	if p.AuthorizationEndpoint != "" {
		endpoint.AuthURL = p.AuthorizationEndpoint
	}
	if p.TokenEndpoint != "" {
		endpoint.TokenURL = p.TokenEndpoint
	}
	cfg := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: "",
		Endpoint:     endpoint,
		RedirectURL:  redirectURI,
		Scopes:       scopesWithOpenID(p.Scopes),
	}
	secret, err := s.providerSecret(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	cfg.ClientSecret = secret
	return cfg, discovered, nil
}

func (s *Service) oauth2Config(p IdentityProvider, redirectURI string) *oauth2.Config {
	scopes := strings.Fields(p.Scopes)
	if len(scopes) == 0 {
		scopes = []string{"read"}
	}
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: "",
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthorizationEndpoint,
			TokenURL: p.TokenEndpoint,
		},
		RedirectURL: redirectURI,
		Scopes:      scopes,
	}
}

func (s *Service) providerSecret(ctx context.Context, id int64) (string, error) {
	row, err := s.store.GetIdentityProviderByID(ctx, id)
	if err != nil {
		return "", err
	}
	return row.ClientSecret, nil
}

func (s *Service) userinfoField(
	ctx context.Context,
	token *oauth2.Token,
	userinfoEndpoint, field string,
) string {
	if userinfoEndpoint == "" {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoEndpoint, nil)
	if err != nil {
		return ""
	}
	token.SetAuthHeader(req)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return ""
	}
	value, _ := lookupJSONPath(payload, field)
	return value
}

func scopesWithOpenID(scopes string) []string {
	fields := strings.Fields(scopes)
	for _, scope := range fields {
		if scope == oidc.ScopeOpenID {
			return fields
		}
	}
	return append([]string{oidc.ScopeOpenID}, fields...)
}

func lookupJSONPath(payload map[string]any, path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	var current any = payload
	for _, rawPart := range strings.Split(path, ".") {
		part, indices := splitPathPart(rawPart)
		obj, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = obj[part]
		if !ok {
			return "", false
		}
		for _, index := range indices {
			arr, ok := current.([]any)
			if !ok || index < 0 || index >= len(arr) {
				return "", false
			}
			current = arr[index]
		}
	}
	switch value := current.(type) {
	case string:
		return value, true
	case nil:
		return "", false
	case float64:
		if value == float64(int64(value)) {
			return fmt.Sprintf("%d", int64(value)), true
		}
		return fmt.Sprintf("%v", value), true
	default:
		return fmt.Sprintf("%v", value), true
	}
}

func splitPathPart(part string) (string, []int) {
	open := strings.IndexByte(part, '[')
	if open < 0 {
		return part, nil
	}
	name := part[:open]
	var indices []int
	rest := part[open:]
	for len(rest) > 0 {
		start := strings.IndexByte(rest, '[')
		end := strings.IndexByte(rest, ']')
		if start != 0 || end < 0 {
			break
		}
		var index int
		if _, err := fmt.Sscanf(rest[1:end], "%d", &index); err != nil {
			break
		}
		indices = append(indices, index)
		rest = rest[end+1:]
	}
	return name, indices
}
