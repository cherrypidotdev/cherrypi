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

package webhookgrpc

import (
	"cherrypi.vladfaust.com/agency/internal/actorid"
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/webhook"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WebhookServer implements agency.v1.WebhookService.
type WebhookServer struct {
	Store    store.Store
	Notifier *webhook.Notifier
}

func (s *WebhookServer) ListWebhooks(
	ctx context.Context,
	_ *agencyv1.ListWebhooksRequest,
) (*agencyv1.ListWebhooksResponse, error) {
	rows, err := s.Store.ListWebhooks(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := make([]*agencyv1.Webhook, len(rows))
	for i, row := range rows {
		out[i] = toProto(row)
	}
	return &agencyv1.ListWebhooksResponse{Webhooks: out}, nil
}

func (s *WebhookServer) CreateWebhook(
	ctx context.Context,
	req *agencyv1.CreateWebhookRequest,
) (*agencyv1.CreateWebhookResponse, error) {
	if req.Url == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("url is required"))
	}
	events, err := eventsToString(req.Events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	now := time.Now().UnixMilli()
	row, err := s.Store.CreateWebhook(ctx, store.CreateWebhookParams{
		AgentID: sql.NullInt64{
			Int64: int64(actorid.ParseAgent(req.AgentId)),
			Valid: req.AgentId != "",
		},
		Url:       req.Url,
		Events:    events,
		Secret:    req.Secret,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.CreateWebhookResponse{Webhook: toProto(row)}, nil
}

func (s *WebhookServer) UpdateWebhook(
	ctx context.Context,
	req *agencyv1.UpdateWebhookRequest,
) (*agencyv1.UpdateWebhookResponse, error) {
	existing, err := s.Store.GetWebhook(ctx, req.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("webhook not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if req.Url == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("url is required"))
	}
	events, err := eventsToString(req.Events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	secret := existing.Secret
	if req.Secret != nil {
		secret = *req.Secret
	}

	row, err := s.Store.UpdateWebhook(ctx, store.UpdateWebhookParams{
		AgentID: sql.NullInt64{
			Int64: int64(actorid.ParseAgent(req.AgentId)),
			Valid: req.AgentId != "",
		},
		Url:       req.Url,
		Events:    events,
		Secret:    secret,
		UpdatedAt: time.Now().UnixMilli(),
		ID:        req.Id,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.UpdateWebhookResponse{Webhook: toProto(row)}, nil
}

func (s *WebhookServer) DeleteWebhook(
	ctx context.Context,
	req *agencyv1.DeleteWebhookRequest,
) (*agencyv1.DeleteWebhookResponse, error) {
	if err := s.Store.DeleteWebhook(ctx, req.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agencyv1.DeleteWebhookResponse{}, nil
}

func (s *WebhookServer) TestWebhook(
	ctx context.Context,
	req *agencyv1.TestWebhookRequest,
) (*agencyv1.TestWebhookResponse, error) {
	row, err := s.Store.GetWebhook(ctx, req.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("webhook not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	name := eventName(req.Event)
	payload, ok := webhook.SamplePayload(name)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unknown event"))
	}

	status, err := s.Notifier.Send(ctx, row.Url, row.Secret, name, payload)
	resp := &agencyv1.TestWebhookResponse{
		StatusCode: int32(status),
		Ok:         status >= 200 && status < 300,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return resp, nil
}

func toProto(row store.Webhook) *agencyv1.Webhook {
	out := &agencyv1.Webhook{
		Id:        row.ID,
		Url:       row.Url,
		Events:    eventsToProto(row.Events),
		HasSecret: row.Secret != "",
		CreatedAt: timestamppb.New(time.UnixMilli(row.CreatedAt)),
		UpdatedAt: timestamppb.New(time.UnixMilli(row.UpdatedAt)),
	}
	if row.AgentID.Valid {
		out.AgentId = actorid.Format(row.AgentID.Int64)
	}
	return out
}

func eventsToString(events []agencyv1.WebhookEvent) (string, error) {
	if len(events) == 0 {
		return "", errors.New("at least one event is required")
	}
	names := make([]string, 0, len(events))
	for _, e := range events {
		name := eventName(e)
		if name == "" {
			return "", errors.New("unknown event")
		}
		names = append(names, name)
	}
	return strings.Join(names, ","), nil
}

func eventsToProto(csv string) []agencyv1.WebhookEvent {
	var out []agencyv1.WebhookEvent
	for _, name := range strings.Split(csv, ",") {
		switch strings.TrimSpace(name) {
		case webhook.EventUserInputRequested:
			out = append(out, agencyv1.WebhookEvent_WEBHOOK_EVENT_USER_INPUT_REQUESTED)
		case webhook.EventAgentStateChanged:
			out = append(out, agencyv1.WebhookEvent_WEBHOOK_EVENT_AGENT_STATE_CHANGED)
		}
	}
	return out
}

func eventName(e agencyv1.WebhookEvent) string {
	switch e {
	case agencyv1.WebhookEvent_WEBHOOK_EVENT_USER_INPUT_REQUESTED:
		return webhook.EventUserInputRequested
	case agencyv1.WebhookEvent_WEBHOOK_EVENT_AGENT_STATE_CHANGED:
		return webhook.EventAgentStateChanged
	default:
		return ""
	}
}
