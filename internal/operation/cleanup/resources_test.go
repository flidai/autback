package cleanup_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/operation/cleanup"
)

func TestResourceManagerCapturesBaselineOnceAndRemovesOnlyOperationResources(t *testing.T) {
	store := &resourceStore{}
	runtime := &resourceRuntime{inventory: cleanup.ResourceSet{
		Containers: []string{"infra-container"}, Networks: []string{"infra-network"}, Volumes: []string{"infra-volume"},
	}}
	manager := cleanup.NewResourceManager(cleanup.ResourceManagerConfig{
		Store: store, Runtime: runtime, Wait: noWait,
	})
	operation := control.Operation{Kind: control.OperationJob, ID: "job-1"}

	if err := manager.Prepare(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	runtime.inventory = cleanup.ResourceSet{
		Services:   []string{"nested-service"},
		Containers: []string{"infra-container", "ryuk", "detached"},
		Networks:   []string{"infra-network", "test-network"},
		Volumes:    []string{"infra-volume", "test-volume"},
	}
	if err := manager.Prepare(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	if runtime.inventoryCalls != 1 {
		t.Fatalf("inventory calls after repeated prepare = %d, want 1", runtime.inventoryCalls)
	}
	if err := manager.Cleanup(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"service:nested-service",
		"container:detached", "container:ryuk",
		"network:test-network",
		"volume:test-volume",
	}
	if !reflect.DeepEqual(runtime.removed, want) {
		t.Fatalf("removed = %#v, want %#v", runtime.removed, want)
	}
}

func TestResourceManagerRetriesPartialCleanupFromPersistedBaseline(t *testing.T) {
	store := &resourceStore{baseline: &cleanup.ResourceSet{Containers: []string{"infra"}}}
	runtime := &resourceRuntime{
		inventory: cleanup.ResourceSet{Containers: []string{"infra", "owned-a", "owned-b"}},
		failOnce:  map[string]error{"container:owned-a": errors.New("Docker daemon restarting")},
	}
	manager := cleanup.NewResourceManager(cleanup.ResourceManagerConfig{Store: store, Runtime: runtime, Wait: noWait})
	operation := control.Operation{Kind: control.OperationJob, ID: "job-retry"}

	if err := manager.Cleanup(context.Background(), operation); err == nil {
		t.Fatal("first cleanup error = nil")
	}
	if err := manager.Cleanup(context.Background(), operation); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
	if runtime.contains("owned-a") || runtime.contains("owned-b") {
		t.Fatalf("owned containers remain: %#v", runtime.inventory.Containers)
	}
}

func TestResourceManagerDoesNotReleaseWhileOwnedResourcesRemain(t *testing.T) {
	store := &resourceStore{baseline: &cleanup.ResourceSet{}}
	runtime := &resourceRuntime{
		inventory:    cleanup.ResourceSet{Containers: []string{"still-running"}},
		keepOnRemove: map[string]bool{"container:still-running": true},
	}
	manager := cleanup.NewResourceManager(cleanup.ResourceManagerConfig{Store: store, Runtime: runtime, Wait: noWait})

	err := manager.Cleanup(context.Background(), control.Operation{Kind: control.OperationJob, ID: "job-running"})
	if err == nil {
		t.Fatal("cleanup error = nil while owned container remains")
	}
}

func TestResourceManagerRemovesVolumeRecreatedWithSameName(t *testing.T) {
	store := &resourceStore{baseline: &cleanup.ResourceSet{Volumes: []string{"database\x00created-before"}}}
	runtime := &resourceRuntime{inventory: cleanup.ResourceSet{Volumes: []string{"database\x00created-during"}}}
	manager := cleanup.NewResourceManager(cleanup.ResourceManagerConfig{Store: store, Runtime: runtime, Wait: noWait})

	if err := manager.Cleanup(context.Background(), control.Operation{Kind: control.OperationJob, ID: "job-volume"}); err != nil {
		t.Fatal(err)
	}
	if len(runtime.removed) != 1 || runtime.removed[0] != "volume:database\x00created-during" {
		t.Fatalf("removed = %#v", runtime.removed)
	}
}

func TestResourceManagerTreatsMissingBaselineAsNoCreatedRuntime(t *testing.T) {
	manager := cleanup.NewResourceManager(cleanup.ResourceManagerConfig{
		Store: &resourceStore{}, Runtime: &resourceRuntime{}, Wait: noWait,
	})
	if err := manager.Cleanup(context.Background(), control.Operation{Kind: control.OperationJob, ID: "never-started"}); err != nil {
		t.Fatal(err)
	}
}

func TestResourceManagerBoundsEachCleanupAttempt(t *testing.T) {
	manager := cleanup.NewResourceManager(cleanup.ResourceManagerConfig{
		Store: &resourceStore{baseline: &cleanup.ResourceSet{}}, Runtime: &resourceRuntime{},
		Timeout: 10 * time.Millisecond,
		Wait: func(ctx context.Context, _ time.Duration) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	err := manager.Cleanup(context.Background(), control.Operation{Kind: control.OperationJob, ID: "bounded"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cleanup error = %v, want deadline exceeded", err)
	}
}

type resourceStore struct {
	baseline *cleanup.ResourceSet
}

func (s *resourceStore) ResourceBaseline(context.Context, control.OperationKind, string) (cleanup.ResourceSet, error) {
	if s.baseline == nil {
		return cleanup.ResourceSet{}, control.ErrNotFound
	}
	return *s.baseline, nil
}

func (s *resourceStore) SaveResourceBaseline(_ context.Context, _ control.OperationKind, _ string, resources cleanup.ResourceSet) error {
	if s.baseline == nil {
		copy := resources
		s.baseline = &copy
	}
	return nil
}

type resourceRuntime struct {
	inventory      cleanup.ResourceSet
	inventoryCalls int
	removed        []string
	failOnce       map[string]error
	keepOnRemove   map[string]bool
}

func (r *resourceRuntime) Inventory(context.Context) (cleanup.ResourceSet, error) {
	r.inventoryCalls++
	return r.inventory, nil
}

func (r *resourceRuntime) RemoveContainer(_ context.Context, id string) error {
	return r.remove("container", id, &r.inventory.Containers)
}

func (r *resourceRuntime) RemoveService(_ context.Context, id string) error {
	return r.remove("service", id, &r.inventory.Services)
}

func (r *resourceRuntime) RemoveNetwork(_ context.Context, id string) error {
	return r.remove("network", id, &r.inventory.Networks)
}

func (r *resourceRuntime) RemoveVolume(_ context.Context, id string) error {
	return r.remove("volume", id, &r.inventory.Volumes)
}

func (r *resourceRuntime) remove(kind, id string, resources *[]string) error {
	key := kind + ":" + id
	r.removed = append(r.removed, key)
	if err := r.failOnce[key]; err != nil {
		delete(r.failOnce, key)
		return err
	}
	if r.keepOnRemove[key] {
		return nil
	}
	for index, resource := range *resources {
		if resource == id {
			*resources = append((*resources)[:index], (*resources)[index+1:]...)
			break
		}
	}
	return nil
}

func (r *resourceRuntime) contains(id string) bool {
	for _, candidate := range r.inventory.Containers {
		if candidate == id {
			return true
		}
	}
	return false
}

func noWait(context.Context, time.Duration) error { return nil }
