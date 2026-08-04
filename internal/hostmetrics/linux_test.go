package hostmetrics

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxSamplerReportsNormalizedHostUtilization(t *testing.T) {
	root := t.TempDir()
	statPath := filepath.Join(root, "stat")
	meminfoPath := filepath.Join(root, "meminfo")
	writeFixture(t, statPath, "cpu  100 0 100 800 0 0 0 0\n")
	writeFixture(t, meminfoPath, "MemTotal:       8388608 kB\nMemAvailable:   3145728 kB\n")
	now := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)
	sampler, err := NewLinuxSampler(LinuxSamplerConfig{
		ProcStatPath: statPath, ProcMeminfoPath: meminfoPath, DiskPath: root, CPUCores: 4,
		Now:       func() time.Time { return now },
		DiskUsage: func(string) (uint64, uint64, error) { return 20, 100, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sampler.Sample(context.Background()); !errors.Is(err, ErrNotReady) {
		t.Fatalf("first sample error=%v, want ErrNotReady", err)
	}

	writeFixture(t, statPath, "cpu  200 0 200 900 0 0 0 0\n")
	now = now.Add(2 * time.Second)
	sample, err := sampler.Sample(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if difference := math.Abs(sample.CPUUtilization - 2.0/3.0); difference > 0.0001 {
		t.Fatalf("cpu utilization=%f", sample.CPUUtilization)
	}
	if sample.CPUCores != 4 || sample.MemoryTotalBytes != 8<<30 || sample.MemoryUsageBytes != 5<<30 || sample.MemoryUtilization != 0.625 {
		t.Fatalf("sample=%#v", sample)
	}
	if sample.DiskUsageBytes != 20 || sample.DiskTotalBytes != 100 || !sample.ObservedAt.Equal(now) {
		t.Fatalf("sample=%#v", sample)
	}
}

func TestLinuxSamplerRejectsRegressingCPUCounters(t *testing.T) {
	root := t.TempDir()
	statPath := filepath.Join(root, "stat")
	meminfoPath := filepath.Join(root, "meminfo")
	writeFixture(t, statPath, "cpu  100 0 100 800 0 0 0 0\n")
	writeFixture(t, meminfoPath, "MemTotal: 1000 kB\nMemAvailable: 500 kB\n")
	sampler, err := NewLinuxSampler(LinuxSamplerConfig{
		ProcStatPath: statPath, ProcMeminfoPath: meminfoPath, DiskPath: root, CPUCores: 1,
		DiskUsage: func(string) (uint64, uint64, error) { return 1, 2, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = sampler.Sample(context.Background())
	writeFixture(t, statPath, "cpu  1 0 1 8 0 0 0 0\n")
	if _, err := sampler.Sample(context.Background()); err == nil {
		t.Fatal("regressing counters were accepted")
	}
}

func TestLinuxSamplerReportsPressureOOMAndInodeEvidence(t *testing.T) {
	root := t.TempDir()
	statPath := filepath.Join(root, "stat")
	meminfoPath := filepath.Join(root, "meminfo")
	pressureRoot := filepath.Join(root, "pressure")
	cgroupRoot := filepath.Join(root, "cgroup")
	if err := os.MkdirAll(pressureRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cgroupRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, statPath, "cpu  100 0 100 800 0 0 0 0\n")
	writeFixture(t, meminfoPath, "MemTotal: 8388608 kB\nMemAvailable: 3145728 kB\n")
	writeFixture(t, filepath.Join(pressureRoot, "cpu"), "some avg10=12.50 avg60=1.00 avg300=0.50 total=100\n")
	writeFixture(t, filepath.Join(pressureRoot, "memory"), "some avg10=5.25 avg60=1.00 avg300=0.50 total=100\nfull avg10=2.50 avg60=0.50 avg300=0.25 total=50\n")
	writeFixture(t, filepath.Join(pressureRoot, "io"), "some avg10=4.00 avg60=1.00 avg300=0.50 total=100\nfull avg10=1.50 avg60=0.50 avg300=0.25 total=50\n")
	writeFixture(t, filepath.Join(cgroupRoot, "memory.events"), "low 0\nhigh 7\nmax 2\noom 3\noom_kill 2\n")
	writeFixture(t, filepath.Join(cgroupRoot, "pids.current"), "321\n")
	writeFixture(t, filepath.Join(cgroupRoot, "pids.max"), "8192\n")
	sampler, err := NewLinuxSampler(LinuxSamplerConfig{
		ProcStatPath: statPath, ProcMeminfoPath: meminfoPath, PressureRoot: pressureRoot, CgroupRoot: cgroupRoot,
		DiskPath: root, CPUCores: 4, DiskUsage: func(string) (uint64, uint64, error) { return 20, 100, nil },
		InodeUsage: func(string) (uint64, uint64, error) { return 25, 1000, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sampler.Sample(context.Background()); !errors.Is(err, ErrNotReady) {
		t.Fatalf("warmup = %v", err)
	}
	writeFixture(t, statPath, "cpu  200 0 200 900 0 0 0 0\n")
	sample, err := sampler.Sample(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sample.CPUPressure != .125 || sample.MemoryPressure != .0525 || sample.MemoryFullPressure != .025 || sample.IOPressure != .04 || sample.IOFullPressure != .015 {
		t.Fatalf("pressure sample = %#v", sample)
	}
	if sample.OOMEvents != 3 || sample.OOMKills != 2 || sample.MemoryHighEvents != 7 || sample.PIDsCurrent != 321 || sample.PIDsLimit != 8192 {
		t.Fatalf("cgroup sample = %#v", sample)
	}
	if sample.DiskInodesUsed != 25 || sample.DiskInodesTotal != 1000 {
		t.Fatalf("inode sample = %#v", sample)
	}
}

func writeFixture(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
