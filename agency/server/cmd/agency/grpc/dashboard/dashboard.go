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
	"cherrypi.vladfaust.com/agency/internal/ids"
	"context"
	"database/sql"
	"errors"
	"time"

	"cherrypi.vladfaust.com/agency/internal/actorid"
	"cherrypi.vladfaust.com/agency/internal/store"
	"cherrypi.vladfaust.com/agency/internal/tunnel"
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
	agentv1 "cherrypi.vladfaust.com/shared/gen/buf/agent/v1"
	agentv1connect "cherrypi.vladfaust.com/shared/gen/buf/agent/v1/agentv1connect"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Hub interface {
	Connection(agentID ids.AgentID) (*tunnel.Connection, error)

	Agents() []ids.AgentID
}

type DashboardServer struct {
	Hub   Hub
	Store store.Store
}

func (s *DashboardServer) client(
	agentID ids.AgentID,
) (agentv1connect.DashboardServiceClient, error) {
	conn, err := s.Hub.Connection(agentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return conn.Dashboard, nil
}

func (s *DashboardServer) sessionClient(
	ctx context.Context,
	sessionID string,
) (agentv1connect.DashboardServiceClient, error) {
	if sessionID == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("task_id is required"),
		)
	}
	agentID, err := s.Store.SessionAgentID(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(
			connect.CodeNotFound,
			errors.New("no task owns this session"),
		)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return s.client(agentID)
}

const liveUsageTimeout = 3 * time.Second

func (s *DashboardServer) GetUsage(
	ctx context.Context,
	req *agencyv1.GetUsageRequest,
) (*agencyv1.GetUsageResponse, error) {
	sinceMs := sql.NullInt64{Int64: req.SinceMs, Valid: req.SinceMs != 0}
	untilMs := sql.NullInt64{Int64: req.UntilMs, Valid: req.UntilMs != 0}
	rows, err := s.Store.ListAgentUsage(ctx,
		sql.NullInt64{Int64: int64(actorid.ParseAgent(req.AgentId)), Valid: req.AgentId != ""},
		sql.NullString{String: req.ProjectId, Valid: req.ProjectId != ""},
		sinceMs,
		untilMs,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &agencyv1.GetUsageResponse{}
	index := make(map[string]int, len(rows))
	for _, row := range rows {
		index[usageKey(ids.AgentID(row.AgentID.Int64), row.SessionID)] = len(resp.Tasks)
		resp.Tasks = append(resp.Tasks, &agencyv1.AgentTaskUsage{
			AgentId:      actorid.Format(row.AgentID.Int64),
			TaskId:       row.SessionID,
			InputTokens:  row.InputTokens,
			OutputTokens: row.OutputTokens,
			Cost:         row.Cost,
			ActiveMs:     row.ActiveMs,
			Runs:         row.Runs,
		})
		resp.InputTokens += row.InputTokens
		resp.OutputTokens += row.OutputTokens
		resp.Cost += row.Cost
	}

	days, err := s.Store.ListAgentUsageByDay(ctx,
		sql.NullInt64{Int64: int64(actorid.ParseAgent(req.AgentId)), Valid: req.AgentId != ""},
		sql.NullString{String: req.ProjectId, Valid: req.ProjectId != ""},
		sinceMs,
		untilMs,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, day := range days {
		resp.Days = append(resp.Days, &agencyv1.UsageDay{
			Day:          timestamppb.New(time.UnixMilli(day.Day * 86400000)),
			InputTokens:  day.InputTokens,
			OutputTokens: day.OutputTokens,
			Cost:         day.Cost,
			Runs:         day.Runs,
		})
	}

	s.overlayLiveUsage(ctx, req, resp, index)

	return resp, nil
}

func usageKey(agentID ids.AgentID, sessionID string) string {
	return actorid.Format(agentID) + "\x00" + sessionID
}

func (s *DashboardServer) overlayLiveUsage(
	ctx context.Context,
	req *agencyv1.GetUsageRequest,
	resp *agencyv1.GetUsageResponse,
	index map[string]int,
) {
	if s.Hub == nil {
		return
	}

	tasks, err := s.Store.ListRunningTasks(ctx)
	if err != nil {
		return
	}
	allowed, actorIDs := liveOverlayTargets(tasks, req, s.filterActorID(ctx, req))
	if len(actorIDs) == 0 {
		return
	}
	agents := make(map[ids.AgentID]struct{}, len(actorIDs))
	for actorID := range actorIDs {
		if row, rowErr := s.Store.GetAgentByActor(ctx, actorID); rowErr == nil {
			agents[row.ID] = struct{}{}
		}
	}

	for _, agentID := range s.Hub.Agents() {
		if _, ok := agents[agentID]; !ok {
			continue
		}
		conn, err := s.Hub.Connection(agentID)
		if err != nil {
			continue
		}

		callCtx, cancel := context.WithTimeout(ctx, liveUsageTimeout)
		res, err := conn.Dashboard.GetUsage(callCtx, &agentv1.GetUsageRequest{})
		cancel()
		if err != nil {
			continue
		}

		for _, bucket := range res.Tasks {

			if !bucket.Live {
				continue
			}
			mergeLiveUsage(resp, index, agentID, bucket, allowed)
		}
	}
}

func mergeLiveUsage(
	resp *agencyv1.GetUsageResponse,
	index map[string]int,
	agentID ids.AgentID,
	bucket *agentv1.TaskUsage,
	allowed map[string]struct{},
) {
	if _, ok := allowed[bucket.TaskId]; !ok {
		return
	}
	live := &agencyv1.AgentTaskUsage{
		AgentId:      actorid.Format(agentID),
		TaskId:       bucket.TaskId,
		InputTokens:  bucket.InputTokens,
		OutputTokens: bucket.OutputTokens,
		Cost:         bucket.Cost,
		ActiveMs:     bucket.ActiveMs,
		Runs:         1,
	}
	key := usageKey(agentID, bucket.TaskId)
	if i, ok := index[key]; ok {

		live.Runs = resp.Tasks[i].Runs + 1
		resp.Tasks[i] = live
		return
	}
	index[key] = len(resp.Tasks)
	resp.Tasks = append(resp.Tasks, live)
}

func liveOverlayTargets(
	tasks []store.Task,
	req *agencyv1.GetUsageRequest,
	filterActor ids.ActorID,
) (allowed map[string]struct{}, agents map[ids.ActorID]struct{}) {
	allowed = make(map[string]struct{}, len(tasks))
	agents = make(map[ids.ActorID]struct{})
	for _, task := range tasks {
		if task.SessionID == "" {
			continue
		}
		if req.ProjectId != "" &&
			(!task.ProjectID.Valid || task.ProjectID.String != req.ProjectId) {
			continue
		}
		if req.AgentId != "" &&
			(!task.AssigneeActorID.Valid || ids.ActorID(task.AssigneeActorID.Int64) != filterActor) {
			continue
		}
		allowed[task.SessionID] = struct{}{}
		if task.AssigneeActorID.Valid {
			agents[ids.ActorID(task.AssigneeActorID.Int64)] = struct{}{}
		}
	}
	return allowed, agents
}

func (s *DashboardServer) filterActorID(
	ctx context.Context,
	req *agencyv1.GetUsageRequest,
) ids.ActorID {
	if req.AgentId == "" {
		return 0
	}
	row, err := s.Store.GetAgent(ctx, actorid.ParseAgent(req.AgentId))
	if err != nil {
		return 0
	}
	return row.ActorID
}

func (s *DashboardServer) GetTaskUsage(
	ctx context.Context,
	req *agencyv1.GetTaskUsageRequest,
) (*agencyv1.GetTaskUsageResponse, error) {
	c, err := s.sessionClient(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	res, err := c.GetUsage(ctx, &agentv1.GetUsageRequest{TaskId: req.TaskId})
	if err != nil {
		return nil, err
	}

	usage := &agentv1.TaskUsage{TaskId: req.TaskId}
	if len(res.Tasks) > 0 {
		usage = res.Tasks[0]
	}
	return &agencyv1.GetTaskUsageResponse{Usage: usage}, nil
}

func (s *DashboardServer) GetDiskUsage(
	ctx context.Context,
	req *agencyv1.GetDiskUsageRequest,
) (*agencyv1.GetDiskUsageResponse, error) {
	c, err := s.client(actorid.ParseAgent(req.AgentId))
	if err != nil {
		return nil, err
	}

	res, err := c.GetDiskUsage(ctx, &agentv1.GetDiskUsageRequest{})
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetDiskUsageResponse{Usage: res}, nil
}

func (s *DashboardServer) GetTaskDiskUsage(
	ctx context.Context,
	req *agencyv1.GetTaskDiskUsageRequest,
) (*agencyv1.GetTaskDiskUsageResponse, error) {
	c, err := s.sessionClient(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	res, err := c.GetTaskDiskUsage(ctx, &agentv1.GetTaskDiskUsageRequest{TaskId: req.TaskId})
	if err != nil {
		return nil, err
	}
	return &agencyv1.GetTaskDiskUsageResponse{Usage: res}, nil
}
