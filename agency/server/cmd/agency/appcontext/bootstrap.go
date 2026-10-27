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

package appcontext

import (
	"context"
	"log"

	"cherrypi.vladfaust.com/agency/internal/accounts"
)

func bootstrapAccounts(
	ctx context.Context,
	svc *accounts.Service,
	cfg Config,
) error {
	count, err := svc.CountUsers(ctx)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	actor, generated, created, err := svc.EnsureFirstAdmin(
		ctx,
		cfg.AdminUsername,
		cfg.AdminPassword,
	)
	if err != nil {
		return err
	}
	if created {
		if generated != "" {
			log.Printf(
				"Created administrator %q with generated password: %s",
				actor.Username,
				generated,
			)
		} else {
			log.Printf("Created administrator %q", actor.Username)
		}
	}
	return nil
}
