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
	agencyv1 "cherrypi.vladfaust.com/shared/gen/buf/agency/v1"
)

const (
	ScopeHost      = "host"
	ScopeContainer = "container"
)

type Sample struct {
	SampledAt int64 `json:"sampled_at"`

	Scope         string    `json:"scope"`
	Hostname      string    `json:"hostname"`
	OS            string    `json:"os"`
	Platform      string    `json:"platform"`
	KernelVersion string    `json:"kernel_version"`
	KernelArch    string    `json:"kernel_arch"`
	CPUPercent    float64   `json:"cpu_percent"`
	CPUCores      []float64 `json:"cpu_cores"`
	Load1         float64   `json:"load1"`
	Load5         float64   `json:"load5"`
	Load15        float64   `json:"load15"`
	Memory        Memory    `json:"memory"`
	Swap          Memory    `json:"swap"`
	Disks         []Disk    `json:"disks"`
	Networks      []Network `json:"networks"`
	GPUs          []GPU     `json:"gpus"`

	UptimeSeconds int64 `json:"uptime_seconds"`

	BootTime        int64   `json:"boot_time"`
	ProcessCount    int64   `json:"process_count"`
	AgentRSSBytes   int64   `json:"agent_rss_bytes"`
	AgentCPUPercent float64 `json:"agent_cpu_percent"`
	GoHeapBytes     int64   `json:"go_heap_bytes"`
}

type Memory struct {
	TotalBytes     int64   `json:"total_bytes"`
	UsedBytes      int64   `json:"used_bytes"`
	AvailableBytes int64   `json:"available_bytes"`
	UsedPercent    float64 `json:"used_percent"`
}

type Disk struct {
	Label       string  `json:"label"`
	Path        string  `json:"path"`
	TotalBytes  int64   `json:"total_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	FreeBytes   int64   `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type Network struct {
	Interface string `json:"interface"`
	RxBytes   int64  `json:"rx_bytes"`
	TxBytes   int64  `json:"tx_bytes"`

	RxBytesPerSecond float64 `json:"rx_bytes_per_second"`
	TxBytesPerSecond float64 `json:"tx_bytes_per_second"`
}

type GPU struct {
	Index              int32   `json:"index"`
	Name               string  `json:"name"`
	UtilizationPercent float64 `json:"utilization_percent"`
	MemoryTotalBytes   int64   `json:"memory_total_bytes"`
	MemoryUsedBytes    int64   `json:"memory_used_bytes"`
	TemperatureCelsius float64 `json:"temperature_celsius"`
	PowerWatts         float64 `json:"power_watts"`
}

func (s Sample) Proto() *agencyv1.AgentHealthSample {
	out := &agencyv1.AgentHealthSample{
		SampledAt:       s.SampledAt,
		Scope:           s.Scope,
		Hostname:        s.Hostname,
		Os:              s.OS,
		Platform:        s.Platform,
		KernelVersion:   s.KernelVersion,
		KernelArch:      s.KernelArch,
		CpuPercent:      s.CPUPercent,
		CpuCores:        append([]float64(nil), s.CPUCores...),
		Load1:           s.Load1,
		Load5:           s.Load5,
		Load15:          s.Load15,
		Memory:          s.Memory.proto(),
		Swap:            s.Swap.proto(),
		UptimeSeconds:   s.UptimeSeconds,
		BootTime:        s.BootTime,
		ProcessCount:    s.ProcessCount,
		AgentRssBytes:   s.AgentRSSBytes,
		AgentCpuPercent: s.AgentCPUPercent,
		GoHeapBytes:     s.GoHeapBytes,
	}
	for _, disk := range s.Disks {
		out.Disks = append(out.Disks, &agencyv1.AgentDiskUsage{
			Label:       disk.Label,
			Path:        disk.Path,
			TotalBytes:  disk.TotalBytes,
			UsedBytes:   disk.UsedBytes,
			FreeBytes:   disk.FreeBytes,
			UsedPercent: disk.UsedPercent,
		})
	}
	for _, network := range s.Networks {
		out.Networks = append(out.Networks, &agencyv1.AgentNetworkUsage{
			Interface:        network.Interface,
			RxBytes:          network.RxBytes,
			TxBytes:          network.TxBytes,
			RxBytesPerSecond: network.RxBytesPerSecond,
			TxBytesPerSecond: network.TxBytesPerSecond,
		})
	}
	for _, gpu := range s.GPUs {
		out.Gpus = append(out.Gpus, &agencyv1.AgentGpuUsage{
			Index:              gpu.Index,
			Name:               gpu.Name,
			UtilizationPercent: gpu.UtilizationPercent,
			MemoryTotalBytes:   gpu.MemoryTotalBytes,
			MemoryUsedBytes:    gpu.MemoryUsedBytes,
			TemperatureCelsius: gpu.TemperatureCelsius,
			PowerWatts:         gpu.PowerWatts,
		})
	}
	return out
}

func (m Memory) proto() *agencyv1.AgentMemory {
	return &agencyv1.AgentMemory{
		TotalBytes:     m.TotalBytes,
		UsedBytes:      m.UsedBytes,
		AvailableBytes: m.AvailableBytes,
		UsedPercent:    m.UsedPercent,
	}
}
