package docker

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
)

func TestClientInventoriesUnprotectedDockerResources(t *testing.T) {
	commands := &fakeCommander{outputs: map[string]string{
		"service ls --quiet": "managed-service\nnested-service\n",
		"service inspect managed-service nested-service": `[
{"ID":"managed-service","Spec":{"Labels":{"autback.managed":"true"}}},
{"ID":"nested-service","Spec":{"Labels":{}}}
]`,
		"container ls --all --quiet --no-trunc": "container-before\nswarm-task\nowned-container\n",
		"container inspect container-before swarm-task owned-container": `[
{"Id":"container-before","Config":{"Labels":{}}},
{"Id":"swarm-task","Config":{"Labels":{"com.docker.swarm.service.id":"service-id"}}},
{"Id":"owned-container","Config":{"Labels":{"org.testcontainers":"true"}}}
]`,
		"network ls --quiet --no-trunc": "network-before\nmanaged-network\nowned-network\n",
		"network inspect network-before managed-network owned-network": `[
{"Id":"network-before","Labels":{}},
{"Id":"managed-network","Labels":{"autback.managed":"true"}},
{"Id":"owned-network","Labels":{"org.testcontainers":"true"}}
]`,
		"volume ls --quiet": "volume-before\nmanaged-volume\nowned-volume\n",
		"volume inspect volume-before managed-volume owned-volume": `[
{"Name":"volume-before","CreatedAt":"2026-08-04T07:00:00Z","Labels":{}},
{"Name":"managed-volume","Labels":{"autback.managed":"true"}},
{"Name":"owned-volume","CreatedAt":"2026-08-04T08:00:00Z","Labels":{"org.testcontainers":"true"}}
]`,
	}}
	client := newClient(commands)

	got, err := client.Inventory(context.Background())
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

func TestClientRemovesResourcesWithIdempotentNotFoundHandling(t *testing.T) {
	commands := &fakeCommander{runErrors: map[string]error{
		"network rm gone": errors.New("Error: No such network: gone"),
	}}
	client := newClient(commands)
	if err := client.RemoveService(context.Background(), "nested-service"); err != nil {
		t.Fatal(err)
	}
	if err := client.RemoveContainer(context.Background(), "container-1"); err != nil {
		t.Fatal(err)
	}
	if err := client.RemoveNetwork(context.Background(), "gone"); err != nil {
		t.Fatalf("remove missing network: %v", err)
	}
	if err := client.RemoveVolume(context.Background(), "volume-1\x002026-08-04T08:00:00Z"); err != nil {
		t.Fatal(err)
	}
	want := []string{"service rm nested-service", "container rm --force container-1", "network rm gone", "volume rm --force volume-1"}
	if !reflect.DeepEqual(commands.runs, want) {
		t.Fatalf("commands = %#v, want %#v", commands.runs, want)
	}
}

type fakeCommander struct {
	outputs   map[string]string
	runErrors map[string]error
	runs      []string
}

func (f *fakeCommander) Output(_ context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	value, ok := f.outputs[key]
	if !ok {
		return nil, errors.New("unexpected command: " + key)
	}
	return []byte(value), nil
}

func (f *fakeCommander) Run(_ context.Context, _ io.Writer, _ io.Writer, args ...string) error {
	key := strings.Join(args, " ")
	f.runs = append(f.runs, key)
	return f.runErrors[key]
}
