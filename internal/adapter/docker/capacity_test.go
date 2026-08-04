package docker

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

func TestTypedCapacityPruneUsesEngineFilters(t *testing.T) {
	api := &fakeCapacityEngine{}
	runtime := newCapacityClient(api)
	if err := runtime.PruneContainers(context.Background(), 24*time.Hour, false); err != nil {
		t.Fatal(err)
	}
	if err := runtime.PruneNetworks(context.Background(), 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := runtime.PruneVolumes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.PruneImages(context.Background(), 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if !api.containerFilters["label"]["org.testcontainers=true"] || !api.containerFilters["until"]["24h"] {
		t.Fatalf("container filters = %#v", api.containerFilters)
	}
	if !api.networkFilters["until"]["5m"] {
		t.Fatalf("network filters = %#v", api.networkFilters)
	}
	if api.volumeAll {
		t.Fatal("volume prune unexpectedly includes named volumes")
	}
	if !api.imageFilters["dangling"]["true"] || !api.imageFilters["until"]["24h"] {
		t.Fatalf("image filters = %#v", api.imageFilters)
	}
}

func TestTypedCapacityPressurePruneIncludesAllStoppedContainers(t *testing.T) {
	api := &fakeCapacityEngine{}
	if err := newCapacityClient(api).PruneContainers(context.Background(), 5*time.Minute, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := api.containerFilters["label"]; ok {
		t.Fatalf("pressure filters = %#v", api.containerFilters)
	}
}

func TestTypedCapacityListsImagesWithoutInspectRoundTrips(t *testing.T) {
	createdAt := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
	api := &fakeCapacityEngine{images: []image.Summary{{
		ID: "sha256:abc", RepoTags: []string{"example:latest"}, RepoDigests: []string{"example@sha256:def"}, Created: createdAt.Unix(),
	}}}
	images, err := newCapacityClient(api).ListImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || images[0].ID != "sha256:abc" || !images[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("images = %#v", images)
	}
	if !reflect.DeepEqual(images[0].RepoDigests, []string{"example@sha256:def"}) {
		t.Fatalf("digests = %#v", images[0].RepoDigests)
	}
}

type fakeCapacityEngine struct {
	containerFilters client.Filters
	networkFilters   client.Filters
	imageFilters     client.Filters
	volumeAll        bool
	images           []image.Summary
}

func (f *fakeCapacityEngine) ContainerPrune(_ context.Context, options client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	f.containerFilters = options.Filters
	return client.ContainerPruneResult{}, nil
}

func (f *fakeCapacityEngine) NetworkPrune(_ context.Context, options client.NetworkPruneOptions) (client.NetworkPruneResult, error) {
	f.networkFilters = options.Filters
	return client.NetworkPruneResult{}, nil
}

func (f *fakeCapacityEngine) VolumePrune(_ context.Context, options client.VolumePruneOptions) (client.VolumePruneResult, error) {
	f.volumeAll = options.All
	return client.VolumePruneResult{}, nil
}

func (f *fakeCapacityEngine) ImagePrune(_ context.Context, options client.ImagePruneOptions) (client.ImagePruneResult, error) {
	f.imageFilters = options.Filters
	return client.ImagePruneResult{}, nil
}

func (f *fakeCapacityEngine) ImageList(context.Context, client.ImageListOptions) (client.ImageListResult, error) {
	return client.ImageListResult{Items: f.images}, nil
}

func (f *fakeCapacityEngine) ImageRemove(context.Context, string, client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	return client.ImageRemoveResult{}, nil
}
