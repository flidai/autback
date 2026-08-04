package swarmscheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
)

func TestProjectCachesUseIndependentPrivateDirectories(t *testing.T) {
	root := t.TempDir()
	caches := []control.CacheMount{{Name: "modules", Target: "/go/pkg/mod"}}
	for _, project := range []string{"project-one", "project-two"} {
		if err := prepareCacheDirectories(root, project, caches); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Join(root, project, "modules"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("cache mode = %o", info.Mode().Perm())
		}
	}
	one := specForJob(Config{CacheRoot: root}, control.Job{ID: "job-one", ProjectID: "project-one", Caches: caches})
	two := specForJob(Config{CacheRoot: root}, control.Job{ID: "job-two", ProjectID: "project-two", Caches: caches})
	if filepath.Join(one.CacheRoot, one.ProjectID, one.Caches[0].Name) == filepath.Join(two.CacheRoot, two.ProjectID, two.Caches[0].Name) {
		t.Fatal("two projects resolved to the same writable cache")
	}
}

func TestCacheDirectoriesRejectUnsafeComponents(t *testing.T) {
	if err := prepareCacheDirectories(t.TempDir(), "project", []control.CacheMount{{Name: "../shared", Target: "/cache"}}); err == nil {
		t.Fatal("unsafe cache name was accepted")
	}
}

func TestPreparingCacheRecordsDurableLastUse(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "project", "modules")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(directory, old, old); err != nil {
		t.Fatal(err)
	}
	if err := prepareCacheDirectories(root, "project", []control.CacheMount{{Name: "modules", Target: "/go/pkg/mod"}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().After(old) {
		t.Fatalf("cache last use = %s, want after %s", info.ModTime(), old)
	}
}

func TestSecretSpecUsesOperationSnapshotAndDedicatedTargets(t *testing.T) {
	root := t.TempDir()
	job := control.Job{ID: "job-1", ProjectID: "project-1", Secrets: []control.SecretBinding{
		{Name: "registry-token", Environment: "REGISTRY_TOKEN"},
		{Name: "signing-key", File: "/run/secrets/signing-key"},
	}}
	spec := specForJob(Config{JobsRoot: root}, job)
	if !spec.HasSecrets || len(spec.Secrets) != 1 {
		t.Fatalf("secret spec = %#v", spec)
	}
	wantSource := filepath.Join(root, job.ID, "secrets", "001-signing-key")
	if spec.Secrets[0].Source != wantSource || spec.Secrets[0].Target != "/run/secrets/signing-key" {
		t.Fatalf("secret mounts = %#v", spec.Secrets)
	}
}

func TestJobSpecCarriesConfiguredResourceEnvelope(t *testing.T) {
	envelope := ResourceEnvelope{
		CPULimitNano: 3_000_000_000, CPUReservationNano: 1_000_000_000,
		MemoryLimitBytes: 5 << 30, MemoryReservationBytes: 1 << 30, PIDsLimit: 4096,
	}
	spec := specForJob(Config{Resources: envelope}, control.Job{ID: "job-1"})
	if spec.Resources != envelope {
		t.Fatalf("resources = %#v, want %#v", spec.Resources, envelope)
	}
}

func TestResourceEnvelopeRejectsUnboundedOrImpossiblePolicies(t *testing.T) {
	valid := ResourceEnvelope{CPULimitNano: 2, CPUReservationNano: 1, MemoryLimitBytes: 2, MemoryReservationBytes: 1, PIDsLimit: 1}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := []ResourceEnvelope{{}, {CPULimitNano: 1, CPUReservationNano: 2, MemoryLimitBytes: 2, MemoryReservationBytes: 1, PIDsLimit: 1}, {CPULimitNano: 2, CPUReservationNano: 1, MemoryLimitBytes: 1, MemoryReservationBytes: 2, PIDsLimit: 1}}
	for _, envelope := range invalid {
		if err := envelope.Validate(); err == nil {
			t.Fatalf("accepted invalid envelope %#v", envelope)
		}
	}
}
