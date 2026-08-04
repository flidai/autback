package docker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flidai/autback/internal/capacity"
	"github.com/moby/moby/client"
)

type capacityEngine interface {
	ContainerPrune(context.Context, client.ContainerPruneOptions) (client.ContainerPruneResult, error)
	NetworkPrune(context.Context, client.NetworkPruneOptions) (client.NetworkPruneResult, error)
	VolumePrune(context.Context, client.VolumePruneOptions) (client.VolumePruneResult, error)
	ImagePrune(context.Context, client.ImagePruneOptions) (client.ImagePruneResult, error)
	ImageList(context.Context, client.ImageListOptions) (client.ImageListResult, error)
	ImageRemove(context.Context, string, client.ImageRemoveOptions) (client.ImageRemoveResult, error)
}

func newCapacityClient(engine capacityEngine) *Client {
	return &Client{capacity: engine}
}

func (c *Client) PruneContainers(ctx context.Context, age time.Duration, all bool) error {
	filters := client.Filters{}.Add("until", dockerDuration(age))
	if !all {
		filters = filters.Add("label", "org.testcontainers=true")
	}
	_, err := c.capacity.ContainerPrune(ctx, client.ContainerPruneOptions{Filters: filters})
	if err != nil {
		return fmt.Errorf("prune Docker containers: %w", err)
	}
	return nil
}

func (c *Client) PruneNetworks(ctx context.Context, age time.Duration) error {
	_, err := c.capacity.NetworkPrune(ctx, client.NetworkPruneOptions{Filters: client.Filters{}.Add("until", dockerDuration(age))})
	if err != nil {
		return fmt.Errorf("prune Docker networks: %w", err)
	}
	return nil
}

func (c *Client) PruneVolumes(ctx context.Context) error {
	_, err := c.capacity.VolumePrune(ctx, client.VolumePruneOptions{})
	if err != nil {
		return fmt.Errorf("prune Docker volumes: %w", err)
	}
	return nil
}

func (c *Client) PruneImages(ctx context.Context, age time.Duration) error {
	filters := client.Filters{}.Add("dangling", "true").Add("until", dockerDuration(age))
	_, err := c.capacity.ImagePrune(ctx, client.ImagePruneOptions{Filters: filters})
	if err != nil {
		return fmt.Errorf("prune Docker images: %w", err)
	}
	return nil
}

func (c *Client) ListImages(ctx context.Context) ([]capacity.RuntimeImage, error) {
	result, err := c.capacity.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list Docker images: %w", err)
	}
	images := make([]capacity.RuntimeImage, 0, len(result.Items))
	for _, image := range result.Items {
		images = append(images, capacity.RuntimeImage{
			ID: image.ID, RepoTags: append([]string(nil), image.RepoTags...), RepoDigests: append([]string(nil), image.RepoDigests...),
			CreatedAt: time.Unix(image.Created, 0).UTC(),
		})
	}
	return images, nil
}

func (c *Client) RemoveImage(ctx context.Context, id string) error {
	_, err := c.capacity.ImageRemove(ctx, id, client.ImageRemoveOptions{PruneChildren: true})
	if err == nil || Classify(err) == ErrorNotFound {
		return nil
	}
	return fmt.Errorf("remove Docker image %s: %w", id, err)
}

func dockerDuration(duration time.Duration) string {
	if duration <= 0 {
		duration = 24 * time.Hour
	}
	if duration%time.Hour == 0 {
		return strconv.FormatInt(int64(duration/time.Hour), 10) + "h"
	}
	if duration%time.Minute == 0 {
		return strconv.FormatInt(int64(duration/time.Minute), 10) + "m"
	}
	return duration.String()
}

var _ capacity.Runtime = (*Client)(nil)
