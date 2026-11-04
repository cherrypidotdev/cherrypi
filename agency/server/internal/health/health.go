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

package health

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	sharedhealth "cherrypi.vladfaust.com/shared/health"
)

const DefaultRetention = 30 * 24 * time.Hour

const DefaultMaxPoints = 720

type Service struct {
	store store.Store

	Retention time.Duration

	MaxPoints int
}

func New(st store.Store) *Service {
	return &Service{
		store:     st,
		Retention: DefaultRetention,
		MaxPoints: DefaultMaxPoints,
	}
}

func (s *Service) Insert(
	ctx context.Context,
	agentID ids.AgentID,
	sample sharedhealth.Sample,
) error {
	if sample.SampledAt == 0 {
		sample.SampledAt = time.Now().UnixMilli()
	}

	cores := marshalJSON(sample.CPUCores)
	disks := marshalJSON(sample.Disks)
	networks := marshalJSON(sample.Networks)
	gpus := marshalJSON(sample.GPUs)

	before := time.Now().Add(-s.Retention).UnixMilli()
	return s.store.InsertAgentHealthSample(
		ctx,
		store.InsertAgentHealthSampleParams{
			AgentID:         int64(agentID),
			SampledAt:       sample.SampledAt,
			Scope:           sample.Scope,
			Hostname:        sample.Hostname,
			Os:              sample.OS,
			Platform:        sample.Platform,
			KernelVersion:   sample.KernelVersion,
			KernelArch:      sample.KernelArch,
			CpuPercent:      sample.CPUPercent,
			CpuCoresJson:    cores,
			Load1:           sample.Load1,
			Load5:           sample.Load5,
			Load15:          sample.Load15,
			MemTotal:        sample.Memory.TotalBytes,
			MemUsed:         sample.Memory.UsedBytes,
			MemAvailable:    sample.Memory.AvailableBytes,
			MemUsedPercent:  sample.Memory.UsedPercent,
			SwapTotal:       sample.Swap.TotalBytes,
			SwapUsed:        sample.Swap.UsedBytes,
			SwapAvailable:   sample.Swap.AvailableBytes,
			SwapUsedPercent: sample.Swap.UsedPercent,
			DisksJson:       disks,
			NetworksJson:    networks,
			GpusJson:        gpus,
			UptimeSeconds:   sample.UptimeSeconds,
			BootTime:        sample.BootTime,
			ProcessCount:    sample.ProcessCount,
			AgentRssBytes:   sample.AgentRSSBytes,
			AgentCpuPercent: sample.AgentCPUPercent,
			GoHeapBytes:     sample.GoHeapBytes,
		},
		before,
	)
}

func (s *Service) Latest(
	ctx context.Context,
	agentID ids.AgentID,
) (sharedhealth.Sample, error) {
	row, err := s.store.GetLatestAgentHealthSample(ctx, agentID)
	if err != nil {
		return sharedhealth.Sample{}, err
	}
	return sampleFromRow(row), nil
}

func (s *Service) History(
	ctx context.Context,
	agentID ids.AgentID,
	since, until int64,
	maxPoints int,
) ([]sharedhealth.Sample, error) {
	if since == 0 {
		since = 0
	}
	if until == 0 {
		until = math.MaxInt64
	}
	if since > until {
		since, until = until, since
	}
	if maxPoints <= 0 {
		maxPoints = s.MaxPoints
	}
	if maxPoints <= 0 {
		maxPoints = DefaultMaxPoints
	}

	rows, err := s.store.ListAgentHealthSamples(
		ctx,
		store.ListAgentHealthSamplesParams{
			AgentID: int64(agentID),
			Since:   since,
			Until:   until,
		},
	)
	if err != nil {
		return nil, err
	}

	samples := make([]sharedhealth.Sample, len(rows))
	for i, row := range rows {
		samples[i] = sampleFromRow(row)
	}
	return downsample(samples, maxPoints), nil
}

func (s *Service) Delete(ctx context.Context, agentID ids.AgentID) error {
	return s.store.DeleteAgentHealthSamples(ctx, agentID)
}

func sampleFromRow(row store.AgentHealthSample) sharedhealth.Sample {
	sample := sharedhealth.Sample{
		SampledAt:     row.SampledAt,
		Scope:         row.Scope,
		Hostname:      row.Hostname,
		OS:            row.Os,
		Platform:      row.Platform,
		KernelVersion: row.KernelVersion,
		KernelArch:    row.KernelArch,
		CPUPercent:    row.CpuPercent,
		Load1:         row.Load1,
		Load5:         row.Load5,
		Load15:        row.Load15,
		Memory: sharedhealth.Memory{
			TotalBytes:     row.MemTotal,
			UsedBytes:      row.MemUsed,
			AvailableBytes: row.MemAvailable,
			UsedPercent:    row.MemUsedPercent,
		},
		Swap: sharedhealth.Memory{
			TotalBytes:     row.SwapTotal,
			UsedBytes:      row.SwapUsed,
			AvailableBytes: row.SwapAvailable,
			UsedPercent:    row.SwapUsedPercent,
		},
		UptimeSeconds:   row.UptimeSeconds,
		BootTime:        row.BootTime,
		ProcessCount:    row.ProcessCount,
		AgentRSSBytes:   row.AgentRssBytes,
		AgentCPUPercent: row.AgentCpuPercent,
		GoHeapBytes:     row.GoHeapBytes,
	}
	unmarshalJSON(row.CpuCoresJson, &sample.CPUCores)
	unmarshalJSON(row.DisksJson, &sample.Disks)
	unmarshalJSON(row.NetworksJson, &sample.Networks)
	unmarshalJSON(row.GpusJson, &sample.GPUs)
	if sample.CPUCores == nil {
		sample.CPUCores = []float64{}
	}
	if sample.Disks == nil {
		sample.Disks = []sharedhealth.Disk{}
	}
	if sample.Networks == nil {
		sample.Networks = []sharedhealth.Network{}
	}
	if sample.GPUs == nil {
		sample.GPUs = []sharedhealth.GPU{}
	}
	return sample
}

func downsample(samples []sharedhealth.Sample, maxPoints int) []sharedhealth.Sample {
	if len(samples) <= maxPoints || maxPoints <= 2 {
		return samples
	}
	out := make([]sharedhealth.Sample, 0, maxPoints)
	step := float64(len(samples)-1) / float64(maxPoints-1)
	last := -1
	for i := 0; i < maxPoints; i++ {
		index := int(math.Round(float64(i) * step))
		if index == last {
			continue
		}
		out = append(out, samples[index])
		last = index
	}
	return out
}

func marshalJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func unmarshalJSON(raw string, target any) {
	if raw == "" {
		return
	}
	_ = json.Unmarshal([]byte(raw), target)
}
