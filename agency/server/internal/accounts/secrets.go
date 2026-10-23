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
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

// Cookie and header names shared by the server and the account transports.
const (
	SessionCookieName = "cherrypi_session"
	CSRFCookieName    = "cherrypi_csrf"
	CSRFHeaderName    = "X-CSRF-Token"
)

// PATPrefix is the displayable prefix of a personal access token.
const PATPrefix = "pat_"

// randomToken returns n cryptographically random bytes as unpadded base64url.
func randomToken(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("accounts: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// HashToken is the one-way hash stored for sessions, tokens and PATs.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// EqualHash compares a stored hash with the hash of candidate in constant time.
func EqualHash(stored []byte, candidate string) bool {
	return subtle.ConstantTimeCompare(stored, HashToken(candidate)) == 1
}
