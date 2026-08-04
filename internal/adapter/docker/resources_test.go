package docker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

func TestClientInventoriesUnprotectedDockerResources(t *testing.T) {
	api := &fakeEngine{
		services: []swarm.Service{
			{ID: "managed-service", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Labels: map[string]string{operationcleanup.ProtectedResourceLabel: "true"}}}},
			{ID: "nested-service", Spec: swarm.ServiceSpec{}},
		},
		containers: []container.Summary{
			{ID: "container-before", Labels: map[string]string{}},
			{ID: "managed-container", Labels: map[string]string{operationcleanup.ProtectedResourceLabel: "true"}},
			{ID: "swarm-task", Labels: map[string]string{"com.docker.swarm.service.id": "service-id"}},
			{ID: "owned-container", Labels: map[string]string{"org.testcontainers": "true"}},
		},
		networks: []network.Summary{
			{Network: network.Network{ID: "network-before", Labels: map[string]string{}}},
			{Network: network.Network{ID: "managed-network", Labels: map[string]string{operationcleanup.ProtectedResourceLabel: "true"}}},
			{Network: network.Network{ID: "owned-network", Labels: map[string]string{"org.testcontainers": "true"}}},
		},
		volumes: []volume.Volume{
			{Name: "volume-before", CreatedAt: "2026-08-04T07:00:00Z", Labels: map[string]string{}},
			{Name: "managed-volume", Labels: map[string]string{operationcleanup.ProtectedResourceLabel: "true"}},
			{Name: "owned-volume", CreatedAt: "2026-08-04T08:00:00Z", Labels: map[string]string{"org.testcontainers": "true"}},
		},
	}
	got, err := newClient(api).Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := operationcleanup.ResourceSet{
		Services:   []string{"nested-service"},
		Containers: []string{"container-before", "owned-container"},
		Networks:   []string{"network-before", "owned-network"},
		Volumes:    []string{"owned-volume\x002026-08-04T08:00:00Z", "volume-before\x002026-08-04T07:00:00Z"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventory = %#v, want %#v", got, want)
	}
}

func TestClientRemovesResourcesAndTreatsNotFoundAsIdempotent(t *testing.T) {
	api := &fakeEngine{removeErrors: map[string]error{"network:gone": errdefs.ErrNotFound}}
	client := newClient(api)
	if err := client.RemoveService(context.Background(), "nested-service"); err != nil {
		t.Fatal(err)
	}
	if err := client.RemoveContainer(context.Background(), "container-1"); err != nil {
		t.Fatal(err)
	}
	if err := client.RemoveNetwork(context.Background(), "gone"); err != nil {
		t.Fatal(err)
	}
	if err := client.RemoveVolume(context.Background(), "volume-1\x00created-at"); err != nil {
		t.Fatal(err)
	}
	want := []string{"service:nested-service", "container:container-1", "network:gone", "volume:volume-1"}
	if !reflect.DeepEqual(api.removed, want) {
		t.Fatalf("removed = %#v, want %#v", api.removed, want)
	}
}

func TestClientReturnsTypedDaemonFailure(t *testing.T) {
	want := errors.New("daemon unavailable")
	client := newClient(&fakeEngine{listErr: want})
	if _, err := client.Inventory(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Inventory error = %v, want %v", err, want)
	}
}

func TestClientNegotiatesOldestAndNewestSupportedEngineAPIs(t *testing.T) {
	for _, apiVersion := range []string{minimumRuntimeAPI, client.MaxAPIVersion} {
		t.Run(apiVersion, func(t *testing.T) {
			var mu sync.Mutex
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				mu.Lock()
				paths = append(paths, request.URL.Path)
				mu.Unlock()
				if request.URL.Path == "/_ping" {
					response.Header().Set("API-Version", apiVersion)
					_, _ = response.Write([]byte("OK"))
					return
				}
				switch {
				case strings.HasSuffix(request.URL.Path, "/services"):
					_, _ = response.Write([]byte("[]"))
				case strings.HasSuffix(request.URL.Path, "/containers/json"):
					_, _ = response.Write([]byte("[]"))
				case strings.HasSuffix(request.URL.Path, "/containers/prune"):
					_, _ = response.Write([]byte(`{"ContainersDeleted":[],"SpaceReclaimed":0}`))
				case strings.HasSuffix(request.URL.Path, "/networks"):
					_, _ = response.Write([]byte("[]"))
				case strings.HasSuffix(request.URL.Path, "/volumes"):
					_, _ = response.Write([]byte(`{"Volumes":[],"Warnings":[]}`))
				case strings.HasSuffix(request.URL.Path, "/info"):
					_, _ = response.Write([]byte(`{"Swarm":{"LocalNodeState":"active"}}`))
				default:
					http.NotFound(response, request)
				}
			}))
			defer server.Close()
			api, err := New(Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer api.Close()
			if _, err := api.Inventory(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := api.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := api.ListResults(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := api.PruneContainers(context.Background(), time.Hour, true); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, suffix := range []string{"/services", "/containers/json", "/containers/prune", "/networks", "/volumes", "/info"} {
				want := "/v" + apiVersion + suffix
				found := false
				for _, path := range paths {
					found = found || path == want
				}
				if !found {
					t.Fatalf("paths = %#v, missing %s", paths, want)
				}
			}
		})
	}
}

type fakeEngine struct {
	services     []swarm.Service
	containers   []container.Summary
	networks     []network.Summary
	volumes      []volume.Volume
	listErr      error
	removeErrors map[string]error
	removed      []string
}

func (f *fakeEngine) ServiceList(context.Context, client.ServiceListOptions) (client.ServiceListResult, error) {
	return client.ServiceListResult{Items: f.services}, f.listErr
}
func (f *fakeEngine) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: f.containers}, f.listErr
}
func (f *fakeEngine) NetworkList(context.Context, client.NetworkListOptions) (client.NetworkListResult, error) {
	return client.NetworkListResult{Items: f.networks}, f.listErr
}
func (f *fakeEngine) VolumeList(context.Context, client.VolumeListOptions) (client.VolumeListResult, error) {
	return client.VolumeListResult{Items: f.volumes}, f.listErr
}
func (f *fakeEngine) ServiceRemove(_ context.Context, id string, _ client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
	return client.ServiceRemoveResult{}, f.removedResource("service", id)
}
func (f *fakeEngine) ContainerRemove(_ context.Context, id string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, f.removedResource("container", id)
}
func (f *fakeEngine) NetworkRemove(_ context.Context, id string, _ client.NetworkRemoveOptions) (client.NetworkRemoveResult, error) {
	return client.NetworkRemoveResult{}, f.removedResource("network", id)
}
func (f *fakeEngine) VolumeRemove(_ context.Context, id string, _ client.VolumeRemoveOptions) (client.VolumeRemoveResult, error) {
	return client.VolumeRemoveResult{}, f.removedResource("volume", id)
}
func (f *fakeEngine) removedResource(kind, id string) error {
	key := kind + ":" + id
	f.removed = append(f.removed, key)
	return f.removeErrors[key]
}
