package capacity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestHostLockSerializesIndependentCapacityControllers(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "capacity.lock")
	first := NewHost(HostConfig{LockPath: lockPath})
	second := NewHost(HostConfig{LockPath: lockPath})
	unlock, err := first.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := second.Lock(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second lock error = %v, want deadline exceeded", err)
	}
	unlock()
	secondUnlock, err := second.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	secondUnlock()
}

func TestHostReportsDurableWorkerActivity(t *testing.T) {
	host := NewHost(HostConfig{Store: &fakeCapacityStore{busy: true}})
	busy, err := host.Busy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !busy {
		t.Fatal("busy = false, want true")
	}
}

func TestHostRemovesOnlyTerminalRetainedJobDirectories(t *testing.T) {
	root := t.TempDir()
	jobs := filepath.Join(root, "jobs")
	cache := filepath.Join(root, "cache")
	for _, id := range []string{"job-terminal", "job-active"} {
		if err := os.MkdirAll(filepath.Join(jobs, id, "workspace"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	host := NewHost(HostConfig{
		CapacityPath: root, JobsRoot: jobs, CacheRoot: cache,
		Store: &fakeCapacityStore{terminal: []string{"job-terminal"}}, Commands: &recordingCommands{},
	})

	report, err := host.Reclaim(context.Background(), ReclaimRequest{JobRetention: 7 * 24 * time.Hour, NormalObjectAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if report.RemovedJobs != 1 {
		t.Fatalf("removed jobs = %d, want 1", report.RemovedJobs)
	}
	if _, err := os.Stat(filepath.Join(jobs, "job-terminal")); !os.IsNotExist(err) {
		t.Fatalf("terminal job still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(jobs, "job-active")); err != nil {
		t.Fatalf("active job was removed: %v", err)
	}
}

func TestHostPrunesProjectCachesOldestFirstToLowWatermark(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "cache")
	oldCache := filepath.Join(cacheRoot, "project", "old")
	newCache := filepath.Join(cacheRoot, "project", "new")
	writeSizedFile(t, oldCache, 8)
	writeSizedFile(t, newCache, 8)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldCache, old, old); err != nil {
		t.Fatal(err)
	}
	host := NewHost(HostConfig{
		CapacityPath: root, JobsRoot: filepath.Join(root, "jobs"), CacheRoot: cacheRoot,
		Store: &fakeCapacityStore{}, Commands: &recordingCommands{},
	})

	report, err := host.Reclaim(context.Background(), ReclaimRequest{CacheHighBytes: 15, CacheLowBytes: 8, NormalObjectAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if report.RemovedCaches != 1 {
		t.Fatalf("removed caches = %d, want 1", report.RemovedCaches)
	}
	if _, err := os.Stat(oldCache); !os.IsNotExist(err) {
		t.Fatalf("old cache still exists: %v", err)
	}
	if _, err := os.Stat(newCache); err != nil {
		t.Fatalf("new cache was removed: %v", err)
	}
}

func TestHostUsesNativeDockerFiltersWithoutParsingTimestamps(t *testing.T) {
	root := t.TempDir()
	commands := &recordingCommands{}
	host := NewHost(HostConfig{
		CapacityPath: root, JobsRoot: filepath.Join(root, "jobs"), CacheRoot: filepath.Join(root, "cache"),
		Store: &fakeCapacityStore{}, Commands: commands,
	})

	_, err := host.Reclaim(context.Background(), ReclaimRequest{
		Pressure: true, TargetFreeBytes: ^uint64(0), NormalObjectAge: 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"container", "prune", "--force", "--filter", "until=5m"},
		{"volume", "prune", "--force"},
		{"exec", "autback-buildkit", "buildctl", "--addr", "tcp://127.0.0.1:1234", "prune", "--all", "--keep-storage", "2000"},
	}
	for _, command := range want {
		if !slices.ContainsFunc(commands.calls, func(call []string) bool { return slices.Equal(call, command) }) {
			t.Errorf("commands %#v missing %#v", commands.calls, command)
		}
	}
}

func TestPressureImageCleanupProtectsProjectImagesAndUsesRecordedLastUse(t *testing.T) {
	root := t.TempDir()
	commands := &recordingCommands{outputs: map[string]string{
		"image ls --quiet --no-trunc": "sha256:protected\nsha256:old\nsha256:recent\n",
		"image inspect sha256:protected sha256:old sha256:recent": `[
  {"Id":"sha256:protected","RepoDigests":["ghcr.io/example/runner@sha256:111"],"Created":"2025-01-01T00:00:00Z"},
  {"Id":"sha256:old","RepoDigests":["ghcr.io/example/old@sha256:222"],"Created":"2025-01-01T00:00:00Z"},
  {"Id":"sha256:recent","RepoDigests":["ghcr.io/example/recent@sha256:333"],"Created":"2025-01-01T00:00:00Z"}
]`,
	}}
	store := &fakeCapacityStore{images: []ImagePolicy{
		{Reference: "ghcr.io/example/runner@sha256:111", Protected: true},
		{Reference: "ghcr.io/example/old@sha256:222", LastUsedAt: time.Now().Add(-time.Hour)},
		{Reference: "ghcr.io/example/recent@sha256:333", LastUsedAt: time.Now()},
	}}
	host := NewHost(HostConfig{
		CapacityPath: root, JobsRoot: filepath.Join(root, "jobs"), CacheRoot: filepath.Join(root, "cache"),
		Store: store, Commands: commands,
	})

	_, err := host.Reclaim(context.Background(), ReclaimRequest{Pressure: true, TargetFreeBytes: ^uint64(0), NormalObjectAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(commands.calls, func(call []string) bool { return slices.Equal(call, []string{"image", "rm", "sha256:old"}) }) {
		t.Fatalf("commands %#v do not remove old unused image", commands.calls)
	}
	for _, forbidden := range []string{"sha256:protected", "sha256:recent"} {
		if slices.ContainsFunc(commands.calls, func(call []string) bool { return slices.Equal(call, []string{"image", "rm", forbidden}) }) {
			t.Fatalf("commands %#v remove protected/recent image %s", commands.calls, forbidden)
		}
	}
}

func writeSizedFile(t *testing.T, directory string, size int) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "data"), make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
}

type fakeCapacityStore struct {
	terminal []string
	images   []ImagePolicy
	busy     bool
}

func (f *fakeCapacityStore) WorkerBusy(context.Context) (bool, error) { return f.busy, nil }

func (f *fakeCapacityStore) TerminalJobIDsBefore(context.Context, time.Time) ([]string, error) {
	return append([]string(nil), f.terminal...), nil
}

func (f *fakeCapacityStore) CapacityImagePolicies(context.Context) ([]ImagePolicy, error) {
	return append([]ImagePolicy(nil), f.images...), nil
}

type recordingCommands struct {
	calls   [][]string
	outputs map[string]string
}

func (r *recordingCommands) Run(_ context.Context, arguments ...string) error {
	r.calls = append(r.calls, append([]string(nil), arguments...))
	return nil
}

func (r *recordingCommands) Output(_ context.Context, arguments ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), arguments...))
	return []byte(r.outputs[joinCommand(arguments)]), nil
}
