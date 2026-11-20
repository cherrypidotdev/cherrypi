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

package sealed

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

var ErrNoKey = errors.New(
	"CHERRYPI_SECRET_KEY is not set, so agency-held secrets cannot be stored",
)

type Sealer struct {
	aead cipher.AEAD
}

func New(encoded string) (*Sealer, error) {
	if strings.TrimSpace(encoded) == "" {
		return &Sealer{}, nil
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("CHERRYPI_SECRET_KEY must be base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("CHERRYPI_SECRET_KEY must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead}, nil
}

func (s *Sealer) Enabled() bool { return s != nil && s.aead != nil }

func (s *Sealer) Seal(value string) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrNoKey
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, []byte(value), nil), nil
}

func (s *Sealer) Open(sealed []byte) (string, error) {
	if !s.Enabled() {
		return "", ErrNoKey
	}
	size := s.aead.NonceSize()
	if len(sealed) < size {
		return "", errors.New("sealed value is truncated")
	}
	plain, err := s.aead.Open(nil, sealed[:size], sealed[size:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
