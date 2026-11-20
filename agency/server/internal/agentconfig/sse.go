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

package agentconfig

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func sseContentType(header http.Header) bool {
	return strings.Contains(strings.ToLower(header.Get("Content-Type")), "text/event-stream")
}

func rewritesContentBlocks(api string) bool {
	return api == "openai-completions"
}

// flushNormalized relays a streaming response line by line, normalizing each

func flushNormalized(w http.ResponseWriter, src io.Reader) error {
	controller := http.NewResponseController(w)
	reader := bufio.NewReaderSize(src, 32<<10)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := w.Write(normalizeSSELine(line)); writeErr != nil {
				return writeErr
			}
			// A blank line ends an SSE event, so this is the point a client can

			if len(bytes.TrimSpace(line)) == 0 {
				_ = controller.Flush()
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// normalizeSSELine rewrites one line of an SSE stream when its frame carries

func normalizeSSELine(line []byte) []byte {
	body := line
	trailer := []byte(nil)
	if trimmed := bytes.TrimRight(body, "\r\n"); len(trimmed) != len(body) {
		trailer = body[len(trimmed):]
		body = trimmed
	}
	if !bytes.HasPrefix(body, []byte("data:")) {
		return line
	}
	payload := bytes.TrimSpace(body[len("data:"):])
	if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
		return line
	}

	if !bytes.Contains(payload, []byte(`"content"`)) {
		return line
	}
	normalized, changed := normalizeContentBlocks(payload)
	if !changed {
		return line
	}
	out := make([]byte, 0, len(normalized)+len("data: ")+len(trailer))
	out = append(out, "data: "...)
	out = append(out, normalized...)
	return append(out, trailer...)
}

func normalizeContentBlocks(payload []byte) ([]byte, bool) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var frame map[string]any
	if err := decoder.Decode(&frame); err != nil {
		return payload, false
	}
	choices, _ := frame["choices"].([]any)
	changed := false
	for _, entry := range choices {
		choice, _ := entry.(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		if delta == nil {
			continue
		}
		blocks, ok := contentBlocks(delta["content"])
		if !ok {
			continue
		}
		reasoning, text := splitContentBlocks(blocks)
		delta["content"] = text
		if reasoning != "" {

			if existing, ok := delta["reasoning_content"].(string); ok {
				reasoning = existing + reasoning
			}
			delta["reasoning_content"] = reasoning
		}
		changed = true
	}
	if !changed {
		return payload, false
	}
	rewritten, err := json.Marshal(frame)
	if err != nil {
		return payload, false
	}
	return rewritten, true
}

func contentBlocks(value any) ([]any, bool) {
	switch content := value.(type) {
	case []any:
		return content, len(content) > 0
	case map[string]any:
		return []any{content}, true
	default:
		return nil, false
	}
}

func splitContentBlocks(blocks []any) (reasoning, text string) {
	var reasoningText, answer strings.Builder
	for _, entry := range blocks {
		block, ok := entry.(map[string]any)
		if !ok {
			answer.WriteString(blockText(entry))
			continue
		}
		kind, _ := block["type"].(string)
		switch kind {
		case "thinking", "reasoning":
			reasoningText.WriteString(blockText(block["thinking"]))
		case "text", "":
			answer.WriteString(blockText(block["text"]))
		default:

			if encoded, err := json.Marshal(block); err == nil {
				answer.Write(encoded)
			}
		}
	}
	return reasoningText.String(), answer.String()
}

func blockText(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		var out strings.Builder
		for _, entry := range v {
			out.WriteString(blockText(entry))
		}
		return out.String()
	case map[string]any:
		if inner, ok := v["text"]; ok {
			return blockText(inner)
		}
		if encoded, err := json.Marshal(v); err == nil {
			return string(encoded)
		}
		return ""
	default:
		if encoded, err := json.Marshal(v); err == nil {
			return string(encoded)
		}
		return ""
	}
}
