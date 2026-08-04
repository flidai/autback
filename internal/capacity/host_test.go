package capacity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
		Store: &fakeCapacityStore{terminal: []string{"job-terminal"}}, Runtime: &recordingRuntime{},
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
		Store: &fakeCapacityStore{}, Runtime: &recordingRuntime{},
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

func TestHostUsesTypedDockerRuntimeForCapacityReclaim(t *testing.T) {
	root := t.TempDir()
	runtime := &recordingRuntime{}
	buildCache := &recordingBuildCache{}
	host := NewHost(HostConfig{
		CapacityPath: root, JobsRoot: filepath.Join(root, "jobs"), CacheRoot: filepath.Join(root, "cache"),
		Store: &fakeCapacityStore{}, Runtime: runtime, BuildCache: buildCache,
	})

	_, err := host.Reclaim(context.Background(), ReclaimRequest{
		Pressure: true, TargetFreeBytes: ^uint64(0), NormalObjectAge: 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRuntime := []string{
		"containers:5m0s:true",
		"networks:5m0s",
		"volumes",
	}
	for _, call := range wantRuntime {
		if !slices.Contains(runtime.calls, call) {
			t.Errorf("runtime calls %#v missing %q", runtime.calls, call)
		}
	}
	if !slices.Equal(buildCache.maxUsed, []int64{2_000_000_000}) {
		t.Errorf("BuildKit max-used calls = %#v", buildCache.maxUsed)
	}
}

func TestPressureImageCleanupProtectsProjectImagesAndUsesRecordedLastUse(t *testing.T) {
	root := t.TempDir()
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	runtime := &recordingRuntime{images: []RuntimeImage{
		{ID: "sha256:protected", RepoDigests: []string{"ghcr.io/example/runner@sha256:111"}, CreatedAt: created},
		{ID: "sha256:old", RepoDigests: []string{"ghcr.io/example/old@sha256:222"}, CreatedAt: created},
		{ID: "sha256:recent", RepoDigests: []string{"ghcr.io/example/recent@sha256:333"}, CreatedAt: created},
	}}
	store := &fakeCapacityStore{images: []ImagePolicy{
		{Reference: "ghcr.io/example/runner@sha256:111", Protected: true},
		{Reference: "ghcr.io/example/old@sha256:222", LastUsedAt: time.Now().Add(-time.Hour)},
		{Reference: "ghcr.io/example/recent@sha256:333", LastUsedAt: time.Now()},
	}}
	host := NewHost(HostConfig{
		CapacityPath: root, JobsRoot: filepath.Join(root, "jobs"), CacheRoot: filepath.Join(root, "cache"),
		Store: store, Runtime: runtime, BuildCache: &recordingBuildCache{},
	})

	_, err := host.Reclaim(context.Background(), ReclaimRequest{Pressure: true, TargetFreeBytes: ^uint64(0), NormalObjectAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(runtime.removed, "sha256:old") {
		t.Fatalf("removed %#v does not include old unused image", runtime.removed)
	}
	for _, forbidden := range []string{"sha256:protected", "sha256:recent"} {
		if slices.Contains(runtime.removed, forbidden) {
			t.Fatalf("removed %#v includes protected/recent image %s", runtime.removed, forbidden)
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

type recordingRuntime struct {
	calls   []string
	images  []RuntimeImage
	removed []string
}

type recordingBuildCache struct {
	maxUsed []int64
}

func (r *recordingBuildCache) Prune(_ context.Context, maxUsedBytes int64) error {
	r.maxUsed = append(r.maxUsed, maxUsedBytes)
	return nil
}

func (r *recordingRuntime) PruneContainers(_ context.Context, age time.Duration, all bool) error {
	r.calls = append(r.calls, "containers:"+age.String()+":"+strconv.FormatBool(all))
	return nil
}

func (r *recordingRuntime) PruneNetworks(_ context.Context, age time.Duration) error {
	r.calls = append(r.calls, "networks:"+age.String())
	return nil
}

func (r *recordingRuntime) PruneVolumes(context.Context) error {
	r.calls = append(r.calls, "volumes")
	return nil
}

func (r *recordingRuntime) PruneImages(_ context.Context, age time.Duration) error {
	r.calls = append(r.calls, "images:"+age.String())
	return nil
}

func (r *recordingRuntime) ListImages(context.Context) ([]RuntimeImage, error) {
	return append([]RuntimeImage(nil), r.images...), nil
}

func (r *recordingRuntime) RemoveImage(_ context.Context, id string) error {
	r.removed = append(r.removed, id)
	return nil
}
