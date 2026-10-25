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

package dashboardgrpc

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"cherrypi.vladfaust.com/agent/internal/dashboard"
	"cherrypi.vladfaust.com/agent/internal/session"
	"cherrypi.vladfaust.com/agent/internal/store"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DashboardServer struct {
	Store store.Store
	Tasks *session.Manager
}

func (d *DashboardServer) GetUsage(
	ctx context.Context,
	req *agentv1.GetUsageRequest,
) (*agentv1.GetUsageResponse, error) {
	if req.TaskId != "" {
		return d.taskUsage(ctx, req.TaskId)
	}

	rows, err := d.Store.ListUsageBySession(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	buckets := make(map[string]*agentv1.TaskUsage, len(rows))
	order := make([]string, 0, len(rows))
	for _, row := range rows {
		buckets[row.SessionID] = &agentv1.TaskUsage{
			TaskId:       row.SessionID,
			InputTokens:  row.InputTokens,
			OutputTokens: row.OutputTokens,
			Cost:         row.Cost,
			ActiveMs:     row.ActiveMs,
		}
		order = append(order, row.SessionID)
	}

	liveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for id, usage := range d.Tasks.LiveUsage(liveCtx) {
		if _, ok := buckets[id]; !ok {
			order = append(order, id)
		}
		buckets[id] = liveBucket(id, usage)
	}

	resp := &agentv1.GetUsageResponse{Tasks: make([]*agentv1.TaskUsage, 0, len(order))}
	for _, id := range order {
		bucket := buckets[id]
		resp.Tasks = append(resp.Tasks, bucket)
		resp.InputTokens += bucket.InputTokens
		resp.OutputTokens += bucket.OutputTokens
		resp.Cost += bucket.Cost
	}

	days, err := d.Store.ListUsageByDay(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	const dayMs = int64(24 * time.Hour / time.Millisecond)
	resp.Days = make([]*agentv1.UsageDay, len(days))
	for i, day := range days {
		resp.Days[i] = &agentv1.UsageDay{
			Day:          timestamppb.New(time.UnixMilli(day.Day * dayMs)),
			InputTokens:  day.InputTokens,
			OutputTokens: day.OutputTokens,
			Cost:         day.Cost,
		}
	}

	return resp, nil
}

func (d *DashboardServer) taskUsage(
	ctx context.Context,
	id string,
) (*agentv1.GetUsageResponse, error) {
	sum, err := d.Store.SumUsageBySession(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	bucket := &agentv1.TaskUsage{
		TaskId:       id,
		InputTokens:  sum.InputTokens,
		OutputTokens: sum.OutputTokens,
		Cost:         sum.Cost,
		ActiveMs:     sum.ActiveMs,
	}

	liveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if usage, ok := d.Tasks.LiveUsageFor(liveCtx, id); ok {
		bucket = liveBucket(id, usage)
	}

	return &agentv1.GetUsageResponse{
		InputTokens:  bucket.InputTokens,
		OutputTokens: bucket.OutputTokens,
		Cost:         bucket.Cost,
		Tasks:        []*agentv1.TaskUsage{bucket},
	}, nil
}

func liveBucket(id string, usage session.Usage) *agentv1.TaskUsage {
	return &agentv1.TaskUsage{
		TaskId:          id,
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
		Cost:            usage.Cost,
		Live:            true,
		TokensPerSecond: usage.TokensPerSecond,
		CostPerHour:     usage.CostPerHour,
		ActiveMs:        usage.ActiveMs,
		ContextTokens:   usage.ContextTokens,
		ContextWindow:   usage.ContextWindow,
	}
}

func (d *DashboardServer) GetDiskUsage(
	ctx context.Context,
	_ *agentv1.GetDiskUsageRequest,
) (*agentv1.GetDiskUsageResponse, error) {
	rows, err := d.Store.ListSessions(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &agentv1.GetDiskUsageResponse{Tasks: make([]*agentv1.TaskDiskUsage, 0, len(rows))}
	for _, row := range rows {
		size := dashboard.DirSize(row.Workspace)
		resp.Tasks = append(resp.Tasks, &agentv1.TaskDiskUsage{
			TaskId: row.ID,
			Bytes:  size,
		})
		resp.TotalBytes += size
	}
	return resp, nil
}

func (d *DashboardServer) GetTaskDiskUsage(
	ctx context.Context,
	req *agentv1.GetTaskDiskUsageRequest,
) (*agentv1.GetTaskDiskUsageResponse, error) {
	row, err := d.Store.GetSession(ctx, req.TaskId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &agentv1.GetTaskDiskUsageResponse{Bytes: dashboard.DirSize(row.Workspace)}, nil
}
