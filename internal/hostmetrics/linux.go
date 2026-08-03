package hostmetrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/flidai/autback/internal/control"
)

var ErrNotReady = errors.New("resource sampler is warming up")

type DiskUsageFunc func(string) (used, total uint64, err error)

type LinuxSamplerConfig struct {
	ProcStatPath    string
	ProcMeminfoPath string
	DiskPath        string
	CPUCores        int
	Now             func() time.Time
	DiskUsage       DiskUsageFunc
}

type LinuxSampler struct {
	config    LinuxSamplerConfig
	mu        sync.Mutex
	previous  cpuCounters
	hasSample bool
}

type cpuCounters struct{ total, idle uint64 }

func NewLinuxSampler(config LinuxSamplerConfig) (*LinuxSampler, error) {
	if config.ProcStatPath == "" {
		config.ProcStatPath = "/proc/stat"
	}
	if config.ProcMeminfoPath == "" {
		config.ProcMeminfoPath = "/proc/meminfo"
	}
	if config.DiskPath == "" {
		return nil, errors.New("resource sampler disk path is required")
	}
	if config.CPUCores == 0 {
		config.CPUCores = runtime.NumCPU()
	}
	if config.CPUCores < 1 {
		return nil, errors.New("resource sampler CPU capacity is invalid")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.DiskUsage == nil {
		config.DiskUsage = diskUsage
	}
	return &LinuxSampler{config: config}, nil
}

func (s *LinuxSampler) Sample(ctx context.Context) (control.ResourceSample, error) {
	if err := ctx.Err(); err != nil {
		return control.ResourceSample{}, err
	}
	cpu, err := readCPU(s.config.ProcStatPath)
	if err != nil {
		return control.ResourceSample{}, err
	}
	memoryUsed, memoryTotal, err := readMemory(s.config.ProcMeminfoPath)
	if err != nil {
		return control.ResourceSample{}, err
	}
	diskUsed, diskTotal, err := s.config.DiskUsage(s.config.DiskPath)
	if err != nil {
		return control.ResourceSample{}, err
	}
	s.mu.Lock()
	previous, ready := s.previous, s.hasSample
	s.previous, s.hasSample = cpu, true
	s.mu.Unlock()
	if !ready {
		return control.ResourceSample{}, ErrNotReady
	}
	if cpu.total <= previous.total || cpu.idle < previous.idle {
		return control.ResourceSample{}, errors.New("CPU counters regressed")
	}
	totalDelta, idleDelta := cpu.total-previous.total, cpu.idle-previous.idle
	if idleDelta > totalDelta {
		return control.ResourceSample{}, errors.New("CPU idle counter exceeds total")
	}
	return control.ResourceSample{
		ObservedAt: s.config.Now().UTC(), CPUUtilization: float64(totalDelta-idleDelta) / float64(totalDelta), CPUCores: s.config.CPUCores,
		MemoryUtilization: float64(memoryUsed) / float64(memoryTotal), MemoryUsageBytes: memoryUsed, MemoryTotalBytes: memoryTotal,
		DiskUsageBytes: diskUsed, DiskTotalBytes: diskTotal,
	}, nil
}

func readCPU(path string) (cpuCounters, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cpuCounters{}, err
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuCounters{}, errors.New("invalid aggregate CPU counters")
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuCounters{}, fmt.Errorf("parse CPU counter: %w", err)
		}
		values = append(values, value)
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuCounters{total: total, idle: idle}, nil
}

func readMemory(path string) (uint64, uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	values := map[string]uint64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ":")
		if name != "MemTotal" && name != "MemAvailable" {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		values[name] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	total, available := values["MemTotal"], values["MemAvailable"]
	if total == 0 || available > total {
		return 0, 0, errors.New("invalid memory capacity")
	}
	return total - available, total, nil
}

func diskUsage(path string) (uint64, uint64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, 0, err
	}
	total := uint64(stats.Blocks) * uint64(stats.Bsize)
	available := uint64(stats.Bavail) * uint64(stats.Bsize)
	if available > total {
		return 0, 0, errors.New("invalid disk capacity")
	}
	return total - available, total, nil
}
