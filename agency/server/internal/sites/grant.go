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

package sites

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
	"time"
)

const grantPrefix = "g1"

const grantTTL = time.Hour

type grantSigner struct {
	once sync.Once
	key  []byte
	now  func() time.Time
}

func (g *grantSigner) secret() []byte {
	g.once.Do(func() {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			panic("sites: cannot generate grant key: " + err.Error())
		}
		g.key = key
	})
	return g.key
}

func (g *grantSigner) clock() time.Time {
	if g.now != nil {
		return g.now()
	}
	return time.Now()
}

func (g *grantSigner) Mint(siteID string) string {
	expiry := g.clock().Add(grantTTL).Unix()
	payload := grantPrefix + "." + siteID + "." + strconv.FormatInt(expiry, 36)
	mac := hmac.New(sha256.New, g.secret())
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (g *grantSigner) Valid(siteID, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != grantPrefix || parts[1] != siteID {
		return false
	}
	expiry, err := strconv.ParseInt(parts[2], 36, 64)
	if err != nil || g.clock().Unix() >= expiry {
		return false
	}
	payload := strings.Join(parts[:3], ".")
	mac := hmac.New(sha256.New, g.secret())
	_, _ = mac.Write([]byte(payload))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	return hmac.Equal(got, want)
}

func rewriteGrantedLinks(body []byte, siteID, grant string) []byte {
	if grant == "" {
		return body
	}
	prefix := []byte(ContentURLFor(siteID))
	suffix := []byte(grant + "/")
	target := append(append([]byte{}, prefix...), suffix...)
	var out []byte
	for {
		at := bytes.Index(body, prefix)
		if at < 0 {
			return append(out, body...)
		}
		out = append(out, body[:at]...)
		rest := body[at+len(prefix):]
		if bytes.HasPrefix(rest, suffix) {
			out = append(out, prefix...)
		} else {
			out = append(out, target...)
		}
		body = rest
	}
}
