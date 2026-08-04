package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
)

const managedLabel = "autback.managed"

type commander interface {
	Output(context.Context, ...string) ([]byte, error)
	Run(context.Context, io.Writer, io.Writer, ...string) error
}

type Config struct {
	Binary string
	Host   string
}

type Client struct {
	commands commander
}

func New(config Config) *Client {
	binary := config.Binary
	if binary == "" {
		binary = "docker"
	}
	return newClient(&dockerCommander{binary: binary, host: config.Host})
}

func newClient(commands commander) *Client { return &Client{commands: commands} }

func (c *Client) Inventory(ctx context.Context) (operationcleanup.ResourceSet, error) {
	services, err := c.services(ctx)
	if err != nil {
		return operationcleanup.ResourceSet{}, err
	}
	containers, err := c.containers(ctx)
	if err != nil {
		return operationcleanup.ResourceSet{}, err
	}
	networks, err := c.networks(ctx)
	if err != nil {
		return operationcleanup.ResourceSet{}, err
	}
	volumes, err := c.volumes(ctx)
	if err != nil {
		return operationcleanup.ResourceSet{}, err
	}
	sort.Strings(services)
	sort.Strings(containers)
	sort.Strings(networks)
	sort.Strings(volumes)
	return operationcleanup.ResourceSet{Services: services, Containers: containers, Networks: networks, Volumes: volumes}, nil
}

func (c *Client) services(ctx context.Context) ([]string, error) {
	ids, err := c.ids(ctx, "service", "ls", "--quiet")
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	data, err := c.commands.Output(ctx, append([]string{"service", "inspect"}, ids...)...)
	if err != nil {
		return nil, fmt.Errorf("inspect Docker services: %w", err)
	}
	var inspected []struct {
		ID   string `json:"ID"`
		Spec struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Spec"`
	}
	if err := json.Unmarshal(data, &inspected); err != nil {
		return nil, fmt.Errorf("decode Docker services: %w", err)
	}
	result := make([]string, 0, len(inspected))
	for _, service := range inspected {
		if !managed(service.Spec.Labels) {
			result = append(result, service.ID)
		}
	}
	return result, nil
}

func (c *Client) containers(ctx context.Context) ([]string, error) {
	ids, err := c.ids(ctx, "container", "ls", "--all", "--quiet", "--no-trunc")
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	data, err := c.commands.Output(ctx, append([]string{"container", "inspect"}, ids...)...)
	if err != nil {
		return nil, fmt.Errorf("inspect Docker containers: %w", err)
	}
	var inspected []struct {
		ID     string `json:"Id"`
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
	}
	if err := json.Unmarshal(data, &inspected); err != nil {
		return nil, fmt.Errorf("decode Docker containers: %w", err)
	}
	result := make([]string, 0, len(inspected))
	for _, container := range inspected {
		if !protectedContainer(container.Config.Labels) {
			result = append(result, container.ID)
		}
	}
	return result, nil
}

func (c *Client) networks(ctx context.Context) ([]string, error) {
	ids, err := c.ids(ctx, "network", "ls", "--quiet", "--no-trunc")
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	data, err := c.commands.Output(ctx, append([]string{"network", "inspect"}, ids...)...)
	if err != nil {
		return nil, fmt.Errorf("inspect Docker networks: %w", err)
	}
	var inspected []struct {
		ID     string            `json:"Id"`
		Labels map[string]string `json:"Labels"`
	}
	if err := json.Unmarshal(data, &inspected); err != nil {
		return nil, fmt.Errorf("decode Docker networks: %w", err)
	}
	result := make([]string, 0, len(inspected))
	for _, network := range inspected {
		if !managed(network.Labels) {
			result = append(result, network.ID)
		}
	}
	return result, nil
}

func (c *Client) volumes(ctx context.Context) ([]string, error) {
	names, err := c.ids(ctx, "volume", "ls", "--quiet")
	if err != nil || len(names) == 0 {
		return nil, err
	}
	data, err := c.commands.Output(ctx, append([]string{"volume", "inspect"}, names...)...)
	if err != nil {
		return nil, fmt.Errorf("inspect Docker volumes: %w", err)
	}
	var inspected []struct {
		Name      string            `json:"Name"`
		CreatedAt string            `json:"CreatedAt"`
		Labels    map[string]string `json:"Labels"`
	}
	if err := json.Unmarshal(data, &inspected); err != nil {
		return nil, fmt.Errorf("decode Docker volumes: %w", err)
	}
	result := make([]string, 0, len(inspected))
	for _, volume := range inspected {
		if !managed(volume.Labels) {
			id := volume.Name
			if volume.CreatedAt != "" {
				id += "\x00" + volume.CreatedAt
			}
			result = append(result, id)
		}
	}
	return result, nil
}

func (c *Client) ids(ctx context.Context, args ...string) ([]string, error) {
	data, err := c.commands.Output(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("list Docker resources: %w", err)
	}
	return strings.Fields(string(data)), nil
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	return c.remove(ctx, "container", "rm", "--force", id)
}

func (c *Client) RemoveService(ctx context.Context, id string) error {
	return c.remove(ctx, "service", "rm", id)
}

func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	return c.remove(ctx, "network", "rm", id)
}

func (c *Client) RemoveVolume(ctx context.Context, id string) error {
	name, _, _ := strings.Cut(id, "\x00")
	return c.remove(ctx, "volume", "rm", "--force", name)
}

func (c *Client) remove(ctx context.Context, args ...string) error {
	var output bytes.Buffer
	err := c.commands.Run(ctx, &output, &output, args...)
	if err == nil {
		return nil
	}
	detail := fmt.Errorf("%w: %s", err, strings.TrimSpace(output.String()))
	if isNotFound(detail) {
		return nil
	}
	return fmt.Errorf("docker %s: %w", strings.Join(args, " "), detail)
}

func protectedContainer(labels map[string]string) bool {
	return managed(labels) || labels["com.docker.swarm.service.id"] != "" || labels["com.docker.swarm.task.id"] != ""
}

func managed(labels map[string]string) bool { return labels[managedLabel] == "true" }

func isNotFound(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such service") || strings.Contains(message, "no such container") || strings.Contains(message, "no such network") || strings.Contains(message, "no such volume")
}

type dockerCommander struct {
	binary string
	host   string
}

func (d *dockerCommander) Output(ctx context.Context, args ...string) ([]byte, error) {
	output, err := d.command(ctx, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (d *dockerCommander) Run(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	command := d.command(ctx, args...)
	command.Stdout, command.Stderr = stdout, stderr
	return command.Run()
}

func (d *dockerCommander) command(ctx context.Context, args ...string) *exec.Cmd {
	if d.host != "" {
		args = append([]string{"--host", d.host}, args...)
	}
	return exec.CommandContext(ctx, d.binary, args...)
}

var _ operationcleanup.ResourceRuntime = (*Client)(nil)
