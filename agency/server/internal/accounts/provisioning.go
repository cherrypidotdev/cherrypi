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
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"cherrypi.vladfaust.com/agency/internal/audit"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
)

var ErrProvisionUnavailable = errors.New("could not provision an account")

const maxProvisionedUsernameLen = 64

const provisionUsernameSuffixes = 100

var usernameUnsafe = regexp.MustCompile(`[^\p{L}\p{M}\p{N}._-]+`)

func (s *Service) ProvisionExternalActor(
	ctx context.Context,
	provider IdentityProvider,
	user ExternalUser,
) (Actor, error) {
	return s.provisionFederatedActor(
		ctx,
		"identity provider "+provider.Name,
		[]string{user.Username, emailLocalPart(user.Email), user.Name},
		user.Name, user.Email, user.EmailVerified,
		func(actorID ids.ActorID) error {
			_, err := s.LinkIdentity(
				ctx, actorID, provider.ID, user.Subject, user.Email,
			)
			return err
		},
		func() (Actor, error) {
			return s.ResolveExternalActor(ctx, provider.ID, user.Subject)
		},
	)
}

func (s *Service) ProvisionLDAPActor(
	ctx context.Context,
	provider LDAPProvider,
	user LDAPUser,
) (Actor, error) {
	return s.provisionFederatedActor(
		ctx,
		"LDAP directory "+provider.Name,
		[]string{user.Username, emailLocalPart(user.Email), user.Name},
		user.Name, user.Email, true,
		func(actorID ids.ActorID) error {
			_, err := s.LinkLDAPIdentity(
				ctx, actorID, provider.ID, user.Username,
			)
			return err
		},
		func() (Actor, error) {
			return s.ResolveLDAPActor(ctx, provider.ID, user.Username)
		},
	)
}

func (s *Service) provisionFederatedActor(
	ctx context.Context,
	source string,
	candidates []string,
	name, email string,
	emailVerified bool,
	link func(actorID ids.ActorID) error,
	resolve func() (Actor, error),
) (Actor, error) {
	bases := usernameBases(candidates)
	for _, base := range bases {
		for attempt := 1; attempt <= provisionUsernameSuffixes; attempt++ {
			username := base
			if attempt > 1 {
				username = fmt.Sprintf("%s-%d", base, attempt)
			}
			actor, err := s.CreateUser(
				ctx, username, "", RoleMember, StatusActive,
			)
			if errors.Is(err, ErrUsernameInUse) {
				continue
			}
			if err != nil {
				return Actor{}, err
			}
			if err := link(actor.ID); err != nil {
				s.discardProvisionedActor(ctx, actor.ID)

				if errors.Is(err, ErrIdentityLinked) {
					if existing, resolveErr := resolve(); resolveErr == nil {
						return existing, nil
					}
				}
				return Actor{}, err
			}
			return s.completeProvisionedActor(
				ctx, actor, source, name, email, emailVerified,
			), nil
		}
	}
	return Actor{}, ErrProvisionUnavailable
}

func (s *Service) discardProvisionedActor(ctx context.Context, actorID ids.ActorID) {
	_ = s.DeleteUser(ctx, actorID)
}

func (s *Service) completeProvisionedActor(
	ctx context.Context,
	actor Actor,
	source, name, email string,
	emailVerified bool,
) Actor {
	if name = strings.TrimSpace(name); name != "" && name != actor.Username {
		if err := s.UpdateProfile(ctx, actor.ID, name, ""); err == nil {
			actor.Name = name
		}
	}
	if email != "" {
		if credential, err := s.AddEmail(ctx, actor.ID, email); err == nil &&
			emailVerified {
			_ = s.store.ConfirmEmailCredential(ctx, store.ConfirmEmailCredentialParams{
				ConfirmedAt: sql.NullInt64{Int64: s.now().UnixMilli(), Valid: true},
				ID:          credential.ID,
			})
		}
	}
	s.auditUserAs(ctx, actor.ID, actor.ID, audit.KindUserCreate, "source", "",
		source, "Created account from "+source, nil)
	return actor
}

func usernameBases(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		base := sanitizeUsername(candidate)
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		out = append(out, base)
	}
	if len(out) == 0 {
		out = append(out, "user")
	}
	return out
}

func sanitizeUsername(raw string) string {
	base := strings.ToLower(strings.TrimSpace(raw))
	base = usernameUnsafe.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-._")
	if runes := []rune(base); len(runes) > maxProvisionedUsernameLen {
		base = strings.Trim(string(runes[:maxProvisionedUsernameLen]), "-._")
	}
	return base
}

func emailLocalPart(email string) string {
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return ""
}
