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

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxLinks = 20

const MaxEmojiBytes = 32

const MaxURLLength = 2048

const MaxLabelLength = 80

var (
	ErrInvalidEmoji = errors.New("emoji must be a single short symbol")
	ErrTooManyLinks = fmt.Errorf("at most %d links are allowed", MaxLinks)
	ErrInvalidLink  = errors.New("links must be absolute http(s) URLs")
	ErrLinkLabel    = fmt.Errorf("link labels must be at most %d characters", MaxLabelLength)
)

type Link struct {
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

func NormalizeEmoji(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > MaxEmojiBytes || !utf8.ValidString(value) {
		return "", ErrInvalidEmoji
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", ErrInvalidEmoji
		}
	}
	return value, nil
}

func NormalizeLinks(links []Link) ([]Link, error) {
	if len(links) > MaxLinks {
		return nil, ErrTooManyLinks
	}
	out := make([]Link, 0, len(links))
	for _, link := range links {
		raw := strings.TrimSpace(link.URL)
		if raw == "" {
			continue
		}
		if len(raw) > MaxURLLength {
			return nil, ErrInvalidLink
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, ErrInvalidLink
		}
		label := strings.TrimSpace(link.Label)
		if utf8.RuneCountInString(label) > MaxLabelLength {
			return nil, ErrLinkLabel
		}
		out = append(out, Link{URL: raw, Label: label})
	}
	return out, nil
}

func EncodeLinks(links []Link) (string, error) {
	if links == nil {
		links = []Link{}
	}
	raw, err := json.Marshal(links)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func DecodeLinks(raw string) []Link {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var links []Link
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return nil
	}
	return links
}
