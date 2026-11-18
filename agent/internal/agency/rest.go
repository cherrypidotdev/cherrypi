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

package agency

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func (c *Client) do(
	ctx context.Context,
	method, path string,
	body any,
	file *upload,
) ([]byte, error) {
	raw, notModified, err := c.doConditionalWithBody(ctx, method, path, body, file)
	if notModified {
		return nil, nil
	}
	return raw, err
}

func (c *Client) doConditional(
	ctx context.Context,
	method, path string,
) ([]byte, bool, error) {
	return c.doConditionalWithBody(ctx, method, path, nil, nil)
}

func (c *Client) doConditionalWithBody(
	ctx context.Context,
	method, path string,
	body any,
	file *upload,
) ([]byte, bool, error) {
	var reader io.Reader
	contentType := ""
	switch {
	case file != nil:
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)

		for _, field := range file.fields {
			if err := writer.WriteField(field.name, field.value); err != nil {
				return nil, false, err
			}
		}
		part, err := writer.CreateFormFile("file", file.filename)
		if err != nil {
			return nil, false, err
		}
		if _, err := part.Write(file.data); err != nil {
			return nil, false, err
		}
		if err := writer.Close(); err != nil {
			return nil, false, err
		}
		reader = &buffer
		contentType = writer.FormDataContentType()
	case body != nil:
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, false, err
		}
		reader = bytes.NewReader(raw)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, false, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, false, err
	}
	if res.StatusCode == http.StatusNotModified {
		return nil, true, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, false, fmt.Errorf(
			"%s %s: %s: %s",
			method,
			path,
			res.Status,
			strings.TrimSpace(string(raw)),
		)
	}
	return raw, false, nil
}
