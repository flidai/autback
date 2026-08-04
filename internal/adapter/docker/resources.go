package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
	"github.com/moby/moby/client"
)

const managedLabel = "autback.managed"

type engine interface {
	ServiceList(context.Context, client.ServiceListOptions) (client.ServiceListResult, error)
	ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error)
	NetworkList(context.Context, client.NetworkListOptions) (client.NetworkListResult, error)
	VolumeList(context.Context, client.VolumeListOptions) (client.VolumeListResult, error)
	ServiceRemove(context.Context, string, client.ServiceRemoveOptions) (client.ServiceRemoveResult, error)
	ContainerRemove(context.Context, string, client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	NetworkRemove(context.Context, string, client.NetworkRemoveOptions) (client.NetworkRemoveResult, error)
	VolumeRemove(context.Context, string, client.VolumeRemoveOptions) (client.VolumeRemoveResult, error)
}

type Config struct {
	Host string
}

type Client struct {
	engine engine
	close  func() error
}

func New(config Config) (*Client, error) {
	options := []client.Opt{client.WithAPIVersionNegotiation()}
	if config.Host != "" {
		options = append(options, client.WithHost(config.Host))
	}
	api, err := client.New(options...)
	if err != nil {
		return nil, fmt.Errorf("create Docker Engine client: %w", err)
	}
	return &Client{engine: api, close: api.Close}, nil
}

func newClient(api engine) *Client { return &Client{engine: api} }

func (c *Client) Close() error {
	if c.close != nil {
		return c.close()
	}
	return nil
}

func (c *Client) Inventory(ctx context.Context) (operationcleanup.ResourceSet, error) {
	services, err := c.engine.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		return operationcleanup.ResourceSet{}, fmt.Errorf("list Docker services: %w", err)
	}
	containers, err := c.engine.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return operationcleanup.ResourceSet{}, fmt.Errorf("list Docker containers: %w", err)
	}
	networks, err := c.engine.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return operationcleanup.ResourceSet{}, fmt.Errorf("list Docker networks: %w", err)
	}
	volumes, err := c.engine.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return operationcleanup.ResourceSet{}, fmt.Errorf("list Docker volumes: %w", err)
	}
	resources := operationcleanup.ResourceSet{}
	for _, service := range services.Items {
		if !managed(service.Spec.Labels) {
			resources.Services = append(resources.Services, service.ID)
		}
	}
	for _, container := range containers.Items {
		if !protectedContainer(container.Labels) {
			resources.Containers = append(resources.Containers, container.ID)
		}
	}
	for _, network := range networks.Items {
		if !managed(network.Labels) {
			resources.Networks = append(resources.Networks, network.ID)
		}
	}
	for _, volume := range volumes.Items {
		if managed(volume.Labels) {
			continue
		}
		id := volume.Name
		if volume.CreatedAt != "" {
			id += "\x00" + volume.CreatedAt
		}
		resources.Volumes = append(resources.Volumes, id)
	}
	sort.Strings(resources.Services)
	sort.Strings(resources.Containers)
	sort.Strings(resources.Networks)
	sort.Strings(resources.Volumes)
	return resources, nil
}

func (c *Client) RemoveService(ctx context.Context, id string) error {
	_, err := c.engine.ServiceRemove(ctx, id, client.ServiceRemoveOptions{})
	return classifyRemoval("service", id, err)
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	_, err := c.engine.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
	return classifyRemoval("container", id, err)
}

func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	_, err := c.engine.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})
	return classifyRemoval("network", id, err)
}

func (c *Client) RemoveVolume(ctx context.Context, id string) error {
	name, _, _ := strings.Cut(id, "\x00")
	_, err := c.engine.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: true})
	return classifyRemoval("volume", name, err)
}

func protectedContainer(labels map[string]string) bool {
	return managed(labels) || labels["com.docker.swarm.service.id"] != "" || labels["com.docker.swarm.task.id"] != ""
}

func managed(labels map[string]string) bool { return labels[managedLabel] == "true" }

func classifyRemoval(kind, id string, err error) error {
	if err == nil || Classify(err) == ErrorNotFound {
		return nil
	}
	return fmt.Errorf("remove Docker %s %s: %w", kind, id, err)
}

var _ operationcleanup.ResourceRuntime = (*Client)(nil)
