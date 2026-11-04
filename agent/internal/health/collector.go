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
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	sharedhealth "cherrypi.vladfaust.com/shared/health"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

const nvidiaSMIQuery = "--query-gpu=index,name,utilization.gpu,memory.total," +
	"memory.used,temperature.gpu,power.draw --format=csv,noheader,nounits"

const gpuTimeout = 3 * time.Second

type DiskTarget struct {
	Label string
	Path  string
}

type Options struct {
	HostProc string
	HostSys  string
	HostRoot string
	Disks    []DiskTarget
}

type Collector struct {
	disks []DiskTarget

	mu        sync.Mutex
	first     bool
	lastNet   map[string]gopsnet.IOCountersStat
	lastNetAt time.Time
}

func New(options Options) *Collector {
	setEnv("HOST_PROC", options.HostProc)
	setEnv("HOST_SYS", options.HostSys)
	setEnv("HOST_ROOT", options.HostRoot)
	if options.HostRoot != "" && options.HostRoot != "/" {
		setEnv("HOST_ETC", options.HostRoot+"/etc")
	}

	return &Collector{
		disks: options.Disks,
		first: true,
	}
}

func setEnv(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	}
}

func (c *Collector) Scope() string {
	if os.Getenv("HOST_PROC") != "" && os.Getenv("HOST_PROC") != "/proc" {
		return sharedhealth.ScopeHost
	}
	return sharedhealth.ScopeContainer
}

func (c *Collector) Sample(ctx context.Context) sharedhealth.Sample {
	now := time.Now()
	sample := sharedhealth.Sample{
		SampledAt: now.UnixMilli(),
		Scope:     c.Scope(),
		Disks:     []sharedhealth.Disk{},
		Networks:  []sharedhealth.Network{},
		GPUs:      []sharedhealth.GPU{},
	}

	c.collectHost(&sample)
	c.collectCPU(&sample)
	c.collectMemory(&sample)
	c.collectDisks(&sample)
	c.collectNetworks(&sample, now)
	c.collectAgentProcess(&sample)
	c.collectGPU(ctx, &sample)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	sample.GoHeapBytes = int64(memStats.HeapAlloc)

	return sample
}

func (c *Collector) collectHost(sample *sharedhealth.Sample) {
	info, err := host.Info()
	if err != nil {
		sample.Hostname, _ = os.Hostname()
		return
	}
	sample.Hostname = info.Hostname
	sample.OS = info.OS
	sample.Platform = info.Platform
	sample.KernelVersion = info.KernelVersion
	sample.KernelArch = info.KernelArch
	sample.UptimeSeconds = int64(info.Uptime)
	sample.BootTime = int64(info.BootTime)
	sample.ProcessCount = int64(info.Procs)
}

func (c *Collector) collectCPU(sample *sharedhealth.Sample) {
	interval := 0 * time.Second
	if c.first {

		interval = 200 * time.Millisecond
		c.first = false
	}
	cores, err := cpu.Percent(interval, true)
	if err != nil && interval == 0 {
		return
	}
	sample.CPUCores = cores
	if len(cores) > 0 {
		var total float64
		for _, core := range cores {
			total += core
		}
		sample.CPUPercent = total / float64(len(cores))
	}
	if avg, err := load.Avg(); err == nil {
		sample.Load1 = avg.Load1
		sample.Load5 = avg.Load5
		sample.Load15 = avg.Load15
	}
}

func (c *Collector) collectMemory(sample *sharedhealth.Sample) {
	if v, err := mem.VirtualMemory(); err == nil {
		sample.Memory = sharedhealth.Memory{
			TotalBytes:     int64(v.Total),
			UsedBytes:      int64(v.Used),
			AvailableBytes: int64(v.Available),
			UsedPercent:    v.UsedPercent,
		}
	}
	if v, err := mem.SwapMemory(); err == nil {
		sample.Swap = sharedhealth.Memory{
			TotalBytes:     int64(v.Total),
			UsedBytes:      int64(v.Used),
			AvailableBytes: int64(v.Free),
			UsedPercent:    v.UsedPercent,
		}
	}
}

func (c *Collector) collectDisks(sample *sharedhealth.Sample) {
	for _, target := range c.disks {
		usage, err := disk.Usage(target.Path)
		if err != nil {
			continue
		}
		sample.Disks = append(sample.Disks, sharedhealth.Disk{
			Label:       target.Label,
			Path:        target.Path,
			TotalBytes:  int64(usage.Total),
			UsedBytes:   int64(usage.Used),
			FreeBytes:   int64(usage.Free),
			UsedPercent: usage.UsedPercent,
		})
	}
}

func (c *Collector) collectNetworks(sample *sharedhealth.Sample, now time.Time) {
	counters, err := gopsnet.IOCounters(true)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	elapsed := now.Sub(c.lastNetAt).Seconds()
	current := make(map[string]gopsnet.IOCountersStat, len(counters))
	for _, counter := range counters {
		if counter.Name == "lo" {
			continue
		}
		current[counter.Name] = counter
		network := sharedhealth.Network{
			Interface: counter.Name,
			RxBytes:   int64(counter.BytesRecv),
			TxBytes:   int64(counter.BytesSent),
		}
		if previous, ok := c.lastNet[counter.Name]; ok && elapsed > 0 {
			network.RxBytesPerSecond = float64(counter.BytesRecv-previous.BytesRecv) / elapsed
			network.TxBytesPerSecond = float64(counter.BytesSent-previous.BytesSent) / elapsed
		}
		sample.Networks = append(sample.Networks, network)
	}
	c.lastNet = current
	c.lastNetAt = now
}

func (c *Collector) collectAgentProcess(sample *sharedhealth.Sample) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return
	}
	if memory, err := proc.MemoryInfo(); err == nil {
		sample.AgentRSSBytes = int64(memory.RSS)
	}
	if percent, err := proc.CPUPercent(); err == nil {
		sample.AgentCPUPercent = percent
	}
}

func (c *Collector) collectGPU(ctx context.Context, sample *sharedhealth.Sample) {
	callCtx, cancel := context.WithTimeout(ctx, gpuTimeout)
	defer cancel()

	output, err := runNvidiaSMI(callCtx)
	if err != nil {
		return
	}
	sample.GPUs = parseGPUOutput(output)
}

func runNvidiaSMI(ctx context.Context) (string, error) {
	cmd := newNvidiaSMICommand(ctx)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func parseGPUOutput(output string) []sharedhealth.GPU {
	gpus := []sharedhealth.GPU{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 7 {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil {
			continue
		}
		gpu := sharedhealth.GPU{
			Index:              int32(index),
			Name:               strings.TrimSpace(fields[1]),
			UtilizationPercent: parseFloat(fields[2]),
			MemoryTotalBytes:   mebibytesToBytes(parseFloat(fields[3])),
			MemoryUsedBytes:    mebibytesToBytes(parseFloat(fields[4])),
			TemperatureCelsius: parseFloat(fields[5]),
			PowerWatts:         parseFloat(fields[6]),
		}
		gpus = append(gpus, gpu)
	}
	return gpus
}

func parseFloat(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "N/A") {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func mebibytesToBytes(value float64) int64 {
	return int64(value * 1024 * 1024)
}

type Sampler struct {
	Collector *Collector
	Publish   func(kind string, payload map[string]any)

	mu       sync.Mutex
	interval time.Duration
	changed  chan struct{}
}

func NewSampler(
	collector *Collector,
	interval time.Duration,
	publish func(kind string, payload map[string]any),
) *Sampler {
	return &Sampler{
		Collector: collector,
		Publish:   publish,
		interval:  interval,
		changed:   make(chan struct{}, 1),
	}
}

func (s *Sampler) SetInterval(interval time.Duration) {
	if interval < 0 {
		interval = 0
	}
	s.mu.Lock()
	unchanged := s.interval == interval
	s.interval = interval
	s.mu.Unlock()
	if unchanged {
		return
	}
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

func (s *Sampler) Interval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

func (s *Sampler) Run(ctx context.Context) {
	if s.Collector == nil || s.Publish == nil {
		return
	}

	select {
	case <-s.changed:
	default:
	}

	for {
		interval := s.Interval()
		if interval <= 0 {
			select {
			case <-ctx.Done():
				return
			case <-s.changed:
				continue
			}
		}

		s.sample(ctx)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.changed:
			timer.Stop()
		case <-timer.C:
		}
	}
}

func (s *Sampler) sample(ctx context.Context) {
	sample := s.Collector.Sample(ctx)
	s.Publish("health", samplePayload(sample))
}

func samplePayload(sample sharedhealth.Sample) map[string]any {
	return map[string]any{
		"sampled_at":        sample.SampledAt,
		"scope":             sample.Scope,
		"hostname":          sample.Hostname,
		"os":                sample.OS,
		"platform":          sample.Platform,
		"kernel_version":    sample.KernelVersion,
		"kernel_arch":       sample.KernelArch,
		"cpu_percent":       sample.CPUPercent,
		"cpu_cores":         sample.CPUCores,
		"load1":             sample.Load1,
		"load5":             sample.Load5,
		"load15":            sample.Load15,
		"memory":            sample.Memory,
		"swap":              sample.Swap,
		"disks":             sample.Disks,
		"networks":          sample.Networks,
		"gpus":              sample.GPUs,
		"uptime_seconds":    sample.UptimeSeconds,
		"boot_time":         sample.BootTime,
		"process_count":     sample.ProcessCount,
		"agent_rss_bytes":   sample.AgentRSSBytes,
		"agent_cpu_percent": sample.AgentCPUPercent,
		"go_heap_bytes":     sample.GoHeapBytes,
	}
}

func String(sample sharedhealth.Sample) string {
	return fmt.Sprintf(
		"cpu=%.1f%% mem=%.1f%% disks=%d gpus=%d",
		sample.CPUPercent,
		sample.Memory.UsedPercent,
		len(sample.Disks),
		len(sample.GPUs),
	)
}
