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

package actorprofile

import agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"

func ToProtoLinks(links []Link) []*agencyv1.ActorLink {
	out := make([]*agencyv1.ActorLink, len(links))
	for i, link := range links {
		out[i] = &agencyv1.ActorLink{Url: link.URL, Label: link.Label}
	}
	return out
}

func FromProtoLinks(links []*agencyv1.ActorLink) []Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]Link, 0, len(links))
	for _, link := range links {
		if link == nil {
			continue
		}
		out = append(out, Link{URL: link.Url, Label: link.Label})
	}
	return out
}
