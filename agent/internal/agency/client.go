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
	"context"
	"net/http"
	"strings"

	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"cherrypi.vladfaust.com/shared/gen/buf/agency/v1/agencyv1connect"
)

type Client struct {
	BaseURL   string
	Token     func() string
	HTTP      *http.Client
	Tasks     agencyv1connect.TaskServiceClient
	Projects  agencyv1connect.ProjectServiceClient
	Agents    agencyv1connect.AgentServiceClient
	Prompts   agencyv1connect.PromptServiceClient
	Agency    agencyv1connect.AgencyServiceClient
	Plugins   agencyv1connect.PluginServiceClient
	Sites     agencyv1connect.SiteServiceClient
	Knowledge agencyv1connect.KnowledgeServiceClient
	Tags      agencyv1connect.TagServiceClient
}

func New(baseURL string, token func() string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpClient = withBearer(httpClient, token)
	baseURL = strings.TrimRight(baseURL, "/")

	grpcURL := baseURL + "/api/grpc"
	return &Client{
		BaseURL:   baseURL,
		Token:     token,
		HTTP:      httpClient,
		Tasks:     agencyv1connect.NewTaskServiceClient(httpClient, grpcURL),
		Projects:  agencyv1connect.NewProjectServiceClient(httpClient, grpcURL),
		Agents:    agencyv1connect.NewAgentServiceClient(httpClient, grpcURL),
		Prompts:   agencyv1connect.NewPromptServiceClient(httpClient, grpcURL),
		Agency:    agencyv1connect.NewAgencyServiceClient(httpClient, grpcURL),
		Plugins:   agencyv1connect.NewPluginServiceClient(httpClient, grpcURL),
		Sites:     agencyv1connect.NewSiteServiceClient(httpClient, grpcURL),
		Knowledge: agencyv1connect.NewKnowledgeServiceClient(httpClient, grpcURL),
		Tags:      agencyv1connect.NewTagServiceClient(httpClient, grpcURL),
	}
}

type bearerTransport struct {
	next  http.RoundTripper
	token func() string
}

func withBearer(client *http.Client, token func() string) *http.Client {
	if token == nil {
		return client
	}
	clone := *client
	next := client.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	clone.Transport = &bearerTransport{next: next, token: token}
	return &clone
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if token := t.token(); token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return t.next.RoundTrip(req)
}

func (c *Client) Task(ctx context.Context, id string) (*agencyv1.Task, error) {
	res, err := c.Tasks.GetTask(ctx, &agencyv1.GetTaskRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return res.Task, nil
}

func (c *Client) REST(ctx context.Context, method, path string, body any) ([]byte, error) {
	return c.do(ctx, method, path, body, nil)
}

func (c *Client) RESTConditional(
	ctx context.Context,
	path string,
) (raw []byte, notModified bool, err error) {
	return c.doConditional(ctx, http.MethodGet, path)
}

func (c *Client) Upload(ctx context.Context, path, filename string, data []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, nil, &upload{filename: filename, data: data})
}

func (c *Client) UploadWithFields(
	ctx context.Context,
	path, filename string,
	data []byte,
	fields [][2]string,
) ([]byte, error) {
	file := &upload{filename: filename, data: data}
	for _, field := range fields {
		file.fields = append(file.fields, uploadField{name: field[0], value: field[1]})
	}
	return c.do(ctx, http.MethodPost, path, nil, file)
}

type upload struct {
	filename string
	data     []byte
	fields   []uploadField
}

type uploadField struct {
	name  string
	value string
}
