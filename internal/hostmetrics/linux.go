package hostmetrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

type InodeUsageFunc func(string) (used, total uint64, err error)

type LinuxSamplerConfig struct {
	ProcStatPath    string
	ProcMeminfoPath string
	PressureRoot    string
	CgroupRoot      string
	DiskPath        string
	CPUCores        int
	Now             func() time.Time
	DiskUsage       DiskUsageFunc
	InodeUsage      InodeUsageFunc
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
	if config.PressureRoot == "" {
		config.PressureRoot = "/proc/pressure"
	}
	if config.CgroupRoot == "" {
		config.CgroupRoot = "/sys/fs/cgroup"
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
	if config.InodeUsage == nil {
		config.InodeUsage = inodeUsage
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
	inodesUsed, inodesTotal, err := s.config.InodeUsage(s.config.DiskPath)
	if err != nil {
		return control.ResourceSample{}, err
	}
	pressure, err := readPressure(s.config.PressureRoot)
	if err != nil {
		return control.ResourceSample{}, err
	}
	cgroup, err := readCgroupEvents(s.config.CgroupRoot)
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
		DiskInodesUsed: inodesUsed, DiskInodesTotal: inodesTotal,
		CPUPressure: pressure.cpuSome, MemoryPressure: pressure.memorySome, MemoryFullPressure: pressure.memoryFull,
		IOPressure: pressure.ioSome, IOFullPressure: pressure.ioFull,
		MemoryHighEvents: cgroup.memoryHigh, OOMEvents: cgroup.oom, OOMKills: cgroup.oomKills,
		PIDsCurrent: cgroup.pidsCurrent, PIDsLimit: cgroup.pidsLimit,
	}, nil
}

type pressureSample struct {
	cpuSome, memorySome, memoryFull, ioSome, ioFull float64
}

func readPressure(root string) (pressureSample, error) {
	var result pressureSample
	for _, item := range []struct {
		name string
		some *float64
		full *float64
	}{
		{name: "cpu", some: &result.cpuSome},
		{name: "memory", some: &result.memorySome, full: &result.memoryFull},
		{name: "io", some: &result.ioSome, full: &result.ioFull},
	} {
		data, err := os.ReadFile(filepath.Join(root, item.name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return pressureSample{}, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			var target *float64
			switch fields[0] {
			case "some":
				target = item.some
			case "full":
				target = item.full
			}
			if target == nil {
				continue
			}
			for _, field := range fields[1:] {
				value, found := strings.CutPrefix(field, "avg10=")
				if !found {
					continue
				}
				parsed, err := strconv.ParseFloat(value, 64)
				if err != nil || parsed < 0 || parsed > 100 {
					return pressureSample{}, fmt.Errorf("parse %s pressure avg10", item.name)
				}
				*target = parsed / 100
			}
		}
	}
	return result, nil
}

type cgroupEvents struct {
	memoryHigh, oom, oomKills, pidsCurrent, pidsLimit uint64
}

func readCgroupEvents(root string) (cgroupEvents, error) {
	var result cgroupEvents
	data, err := os.ReadFile(filepath.Join(root, "memory.events"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if err == nil {
		values, err := parseKeyValues(data)
		if err != nil {
			return result, err
		}
		result.memoryHigh, result.oom, result.oomKills = values["high"], values["oom"], values["oom_kill"]
	}
	if result.pidsCurrent, err = readUintFile(filepath.Join(root, "pids.current")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if err != nil {
		result.pidsCurrent = 0
	}
	limit, err := os.ReadFile(filepath.Join(root, "pids.max"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if err == nil && strings.TrimSpace(string(limit)) != "max" {
		result.pidsLimit, err = strconv.ParseUint(strings.TrimSpace(string(limit)), 10, 64)
		if err != nil {
			return result, fmt.Errorf("parse cgroup pids.max: %w", err)
		}
	}
	return result, nil
}

func parseKeyValues(data []byte) (map[string]uint64, error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, errors.New("invalid cgroup event record")
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse cgroup event %s: %w", fields[0], err)
		}
		values[fields[0]] = value
	}
	return values, nil
}

func readUintFile(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return value, nil
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

func inodeUsage(path string) (uint64, uint64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, 0, err
	}
	total, available := stats.Files, stats.Ffree
	if available > total {
		return 0, 0, errors.New("invalid inode capacity")
	}
	return total - available, total, nil
}
