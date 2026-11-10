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

package plugins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Preview struct {
	Plugin           Plugin
	Installed        bool
	InstalledVersion string
	Compatible       bool
	Warnings         []string
}

func (s *Service) Inspect(ctx context.Context, data []byte) (Preview, error) {
	bundle, err := ReadBundle(data)
	if err != nil {
		return Preview{}, err
	}
	if err := compileBundle(bundle); err != nil {
		return Preview{}, err
	}
	manifest := bundle.Manifest
	compatible, rangeErr := manifest.RequiresAPIVersion(APIVersion)
	preview := Preview{
		Plugin: Plugin{
			ID:          manifest.ID,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Manifest:    manifest,
			ManifestRaw: bundle.ManifestRaw,
			Runtime:     manifest.Runtime,
			Entry:       manifest.Entry,
			Checksum:    bundle.Checksum,
		},
		Compatible: compatible,
	}
	if rangeErr != nil {
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf("The bundle declares an invalid agency range: %v", rangeErr),
		)
	} else if !compatible {
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf(
				"This agency implements plugin API %s, outside the bundle's %s range; installation is refused.",
				APIVersion,
				manifest.RequiresAgency,
			),
		)
	}
	if existing, err := s.Store.Get(ctx, manifest.ID); err == nil {
		preview.Installed = true
		preview.InstalledVersion = existing.Version
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf(
				"Version %s is already installed; installing replaces it and keeps its enabled state.",
				existing.Version,
			),
		)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Preview{}, err
	}
	if manifest.NetworkAllowsAnyHost() {
		preview.Warnings = append(
			preview.Warnings,
			"This bundle may call any public HTTPS host; its network allowlist is \"*\".",
		)
	}
	if manifest.HasHandler(HandlerHTTP) {
		preview.Warnings = append(
			preview.Warnings,
			"Its HTTP routes are reachable anonymously under /plugins/<id>/; the plugin validates its own secret.",
		)
	}
	if len(manifest.Events) > 0 {
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf("It subscribes to %d agency events.", len(manifest.Events)),
		)
	}
	if manifest.Schedule != nil {
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf(
				"It calls handle_tick every %d seconds while enabled.",
				manifest.Schedule.EverySeconds,
			),
		)
	}
	if len(manifest.Tools) > 0 {
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf("It attaches %d tools to the tasks it creates.", len(manifest.Tools)),
		)
	}
	if len(manifest.Settings) > 0 {
		preview.Warnings = append(
			preview.Warnings,
			fmt.Sprintf("It declares settings for: %s.", surfaceList(manifest)),
		)
	}
	return preview, nil
}

func compileBundle(bundle Bundle) error {
	names := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		if strings.HasSuffix(name, ".lua") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := compileLua(bundle.Files[name], name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func surfaceList(m Manifest) string {
	out := ""
	for i, scope := range m.Surfaces() {
		if i > 0 {
			out += ", "
		}
		out += scope
	}
	return out
}
