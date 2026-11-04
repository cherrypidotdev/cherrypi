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
	"net/url"
	"strings"
)

type urlKind string

const (
	urlKindSrc    urlKind = "src"
	urlKindHref   urlKind = "href"
	urlKindSrcset urlKind = "srcset"
	urlKindCSS    urlKind = "css"
)

type urlMapper func(kind urlKind, value string) string

// file in the site archive (a Vite build with the default `base: "/"` emits

func rewriteSiteDocument(
	body []byte,
	siteID, grant string,
	exists func(string) bool,
) []byte {
	body = rewriteGrantedLinks(body, siteID, grant)
	return mapHTMLURLs(body, siteURLMapper(siteID, grant, exists))
}

func rewriteSiteStylesheet(
	body []byte,
	siteID, grant string,
	exists func(string) bool,
) []byte {
	return mapCSSURLs(body, siteURLMapper(siteID, grant, exists))
}

func siteURLMapper(siteID, grant string, exists func(string) bool) urlMapper {
	contentPrefix := ContentURLFor(siteID)
	granted := contentPrefix
	if grant != "" {
		granted = contentPrefix + grant + "/"
	}

	staleGrant := grantPrefix + "." + siteID + "."
	return func(_ urlKind, value string) string {
		if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
			return ""
		}
		pathPart, suffix := splitURLSuffix(value)
		if pathPart == "" {
			return ""
		}
		if rest, ok := strings.CutPrefix(pathPart, contentPrefix); ok {
			if grant == "" {
				return ""
			}

			for {
				head, tail, hasSlash := strings.Cut(rest, "/")
				if !hasSlash || !strings.HasPrefix(head, staleGrant) {
					break
				}
				rest = tail
			}
			return granted + rest + suffix
		}
		decoded, err := url.PathUnescape(pathPart)
		if err != nil {
			return ""
		}
		if !exists(decoded) {
			return ""
		}
		return granted + strings.TrimPrefix(pathPart, "/") + suffix
	}
}

func splitURLSuffix(value string) (string, string) {
	if at := strings.IndexAny(value, "?#"); at >= 0 {
		return value[:at], value[at:]
	}
	return value, ""
}

func mapURLOrKeep(kind urlKind, value string, mapURL urlMapper) string {
	if replacement := mapURL(kind, value); replacement != "" {
		return replacement
	}
	return value
}

func mapHTMLURLs(html []byte, mapURL urlMapper) []byte {
	var out bytes.Buffer
	out.Grow(len(html) + len(html)/8)
	i := 0
	for i < len(html) {
		at := bytes.IndexByte(html[i:], '<')
		if at < 0 {
			out.Write(html[i:])
			break
		}
		at += i
		out.Write(html[i:at])
		if bytes.HasPrefix(html[at:], []byte("<!--")) {
			end := bytes.Index(html[at+4:], []byte("-->"))
			if end < 0 {
				out.Write(html[at:])
				break
			}
			end += at + 4 + 3
			out.Write(html[at:end])
			i = end
			continue
		}
		tagEnd, name, ok := scanTag(html, at)
		if !ok {
			out.WriteByte('<')
			i = at + 1
			continue
		}
		closing := html[at+1] == '/'
		out.Write(mapTagURLs(html[at:tagEnd], mapURL))
		i = tagEnd
		if closing || (name != "script" && name != "style") {
			continue
		}

		end := indexFold(html[i:], []byte("</"+name))
		if end < 0 {
			out.Write(html[i:])
			break
		}
		end += i
		if name == "style" {
			out.Write(mapCSSURLs(html[i:end], mapURL))
		} else {
			out.Write(html[i:end])
		}
		i = end
	}
	return out.Bytes()
}

func scanTag(html []byte, start int) (end int, name string, ok bool) {
	if start+1 >= len(html) || html[start] != '<' {
		return 0, "", false
	}
	i := start + 1
	if html[i] == '!' || html[i] == '?' {
		return scanTagEnd(html, i), "", true
	}
	if html[i] == '/' {
		i++
	}
	nameStart := i
	for i < len(html) && isTagNameChar(html[i]) {
		i++
	}
	if i == nameStart {
		return 0, "", false
	}
	return scanTagEnd(html, i), strings.ToLower(string(html[nameStart:i])), true
}

func scanTagEnd(html []byte, start int) int {
	var quote byte
	for i := start; i < len(html); i++ {
		switch c := html[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return i + 1
		}
	}
	return len(html)
}

func mapTagURLs(tag []byte, mapURL urlMapper) []byte {
	if len(tag) < 2 || tag[0] != '<' || tag[1] == '/' {
		return tag
	}
	var out bytes.Buffer
	out.Grow(len(tag))
	i := 1
	for i < len(tag) && isTagNameChar(tag[i]) {
		i++
	}
	out.Write(tag[:i])
	for i < len(tag) {
		space := i
		for i < len(tag) && isHTMLSpace(tag[i]) {
			i++
		}
		out.Write(tag[space:i])
		if i >= len(tag) {
			break
		}
		if tag[i] == '>' {
			out.WriteByte('>')
			i++
			continue
		}
		if tag[i] == '/' {
			out.WriteByte('/')
			i++
			continue
		}
		nameStart := i
		for i < len(tag) && !isHTMLSpace(tag[i]) && tag[i] != '=' &&
			tag[i] != '>' && tag[i] != '/' {
			i++
		}
		if nameStart == i {
			out.WriteByte(tag[i])
			i++
			continue
		}
		name := strings.ToLower(string(tag[nameStart:i]))
		out.Write(tag[nameStart:i])

		space = i
		for i < len(tag) && isHTMLSpace(tag[i]) {
			i++
		}
		out.Write(tag[space:i])
		if i >= len(tag) || tag[i] != '=' {
			continue
		}
		out.WriteByte('=')
		i++
		space = i
		for i < len(tag) && isHTMLSpace(tag[i]) {
			i++
		}
		out.Write(tag[space:i])
		if i >= len(tag) {
			break
		}

		var quote byte
		if tag[i] == '"' || tag[i] == '\'' {
			quote = tag[i]
			i++
		}
		valueStart := i
		if quote != 0 {
			for i < len(tag) && tag[i] != quote {
				i++
			}
		} else {
			for i < len(tag) && !isHTMLSpace(tag[i]) && tag[i] != '>' {
				i++
			}
		}
		value := string(tag[valueStart:i])
		if quote != 0 && i < len(tag) {
			i++
		}

		if quote != 0 {
			out.WriteByte(quote)
		}
		switch name {
		case "src", "poster", "action", "formaction", "manifest",
			"data-src", "data-href":
			out.WriteString(mapURLOrKeep(urlKindSrc, value, mapURL))
		case "href":
			out.WriteString(mapURLOrKeep(urlKindHref, value, mapURL))
		case "srcset", "imagesrcset":
			out.WriteString(mapSrcset(value, mapURL))
		case "style":
			out.Write(mapCSSURLs([]byte(value), mapURL))
		default:
			out.WriteString(value)
		}
		if quote != 0 {
			out.WriteByte(quote)
		}
	}
	return out.Bytes()
}

func mapSrcset(value string, mapURL urlMapper) string {
	parts := strings.Split(value, ",")
	changed := false
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		reference, descriptor := trimmed, ""
		if at := strings.IndexAny(trimmed, " \t\r\n"); at >= 0 {
			reference, descriptor = trimmed[:at], trimmed[at:]
		}
		if replacement := mapURL(urlKindSrcset, reference); replacement != "" {
			parts[i] = replacement + descriptor
			changed = true
		} else {
			parts[i] = trimmed
		}
	}
	if !changed {
		return ""
	}
	return strings.Join(parts, ", ")
}

func mapCSSURLs(css []byte, mapURL urlMapper) []byte {
	var out bytes.Buffer
	out.Grow(len(css))
	i := 0
	for i < len(css) {
		at, isImport := nextCSSRef(css, i)
		if at < 0 {
			out.Write(css[i:])
			break
		}
		out.Write(css[i:at])
		if isImport {
			i = mapCSSImport(css, at, &out, mapURL)
			continue
		}
		i = mapCSSURL(css, at, &out, mapURL)
	}
	return out.Bytes()
}

func nextCSSRef(css []byte, from int) (at int, isImport bool) {
	urlAt := indexFold(css[from:], []byte("url("))
	importAt := indexFold(css[from:], []byte("@import"))
	switch {
	case urlAt < 0 && importAt < 0:
		return -1, false
	case importAt < 0 || (urlAt >= 0 && urlAt < importAt):
		return from + urlAt, false
	default:
		return from + importAt, true
	}
}

func mapCSSURL(css []byte, at int, out *bytes.Buffer, mapURL urlMapper) int {
	i := at + len("url(")
	for i < len(css) && isHTMLSpace(css[i]) {
		i++
	}
	var quote byte
	if i < len(css) && (css[i] == '"' || css[i] == '\'') {
		quote = css[i]
		i++
	}
	valueStart := i
	if quote != 0 {
		for i < len(css) && css[i] != quote {
			i++
		}
	} else {
		for i < len(css) && css[i] != ')' {
			i++
		}
	}
	value := string(css[valueStart:i])
	out.Write(css[at:valueStart])
	out.WriteString(mapURLOrKeep(urlKindCSS, value, mapURL))
	if quote != 0 && i < len(css) {
		out.WriteByte(quote)
		i++
	}
	return i
}

func mapCSSImport(css []byte, at int, out *bytes.Buffer, mapURL urlMapper) int {
	i := at + len("@import")
	out.Write(css[at:i])
	for i < len(css) && isHTMLSpace(css[i]) {
		out.WriteByte(css[i])
		i++
	}
	if i >= len(css) || (css[i] != '"' && css[i] != '\'') {

		return i
	}
	quote := css[i]
	out.WriteByte(quote)
	i++
	valueStart := i
	for i < len(css) && css[i] != quote {
		i++
	}
	out.WriteString(mapURLOrKeep(urlKindCSS, string(css[valueStart:i]), mapURL))
	if i < len(css) {
		out.WriteByte(quote)
		i++
	}
	return i
}

func indexFold(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if lowerASCII(haystack[i+j]) != lowerASCII(needle[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func isHTMLSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '\f':
		return true
	}
	return false
}

func isTagNameChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == ':', c == '_':
		return true
	}
	return false
}
