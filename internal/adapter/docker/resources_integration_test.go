package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
)

func TestResourceManagerCleansRealOperationResources(t *testing.T) {
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("Docker daemon is unavailable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	prefix := fmt.Sprintf("autback-it-%d", time.Now().UnixNano())
	preexisting := prefix + "-before"
	detached := prefix + "-detached"
	ryuk := prefix + "-ryuk"
	network := prefix + "-network"
	volume := prefix + "-volume"
	service := prefix + "-service"

	dockerRun(t, ctx, "pull", "alpine:3.20")
	dockerRun(t, ctx, "run", "--detach", "--name", preexisting, "alpine:3.20", "sleep", "300")
	t.Cleanup(func() { dockerCleanup(preexisting, detached, ryuk, network, volume, service) })
	initializedSwarm := false
	if !swarmActive(ctx) {
		if err := exec.CommandContext(ctx, "docker", "swarm", "init", "--advertise-addr", "127.0.0.1").Run(); err != nil {
			t.Skipf("Docker Swarm is unavailable: %v", err)
		}
		initializedSwarm = true
	}
	if initializedSwarm {
		defer exec.Command("docker", "swarm", "leave", "--force").Run() //nolint:errcheck // best-effort test cleanup
	}
	waitForSwarmInfrastructure(t, ctx)

	client := New(Config{})
	store := &integrationBaselineStore{}
	manager := operationcleanup.NewResourceManager(operationcleanup.ResourceManagerConfig{
		Store: store, Runtime: client, Wait: func(context.Context, time.Duration) error { return nil },
	})
	operation := control.Operation{Kind: control.OperationJob, ID: prefix}
	if err := manager.Prepare(ctx, operation); err != nil {
		t.Fatal(err)
	}

	dockerRun(t, ctx, "network", "create", network)
	dockerRun(t, ctx, "volume", "create", volume)
	dockerRun(t, ctx, "run", "--detach", "--name", detached, "--network", network, "--volume", volume+":/data", "alpine:3.20", "sleep", "300")
	dockerRun(t, ctx, "create", "--name", ryuk, "--label", "org.testcontainers.ryuk.container=true", "alpine:3.20", "true")

	dockerRun(t, ctx, "service", "create", "--detach=true", "--name", service, "alpine:3.20", "sleep", "300")

	if err := manager.Cleanup(ctx, operation); err != nil {
		t.Fatal(err)
	}
	for _, resource := range [][]string{
		{"container", "inspect", detached}, {"container", "inspect", ryuk},
		{"network", "inspect", network}, {"volume", "inspect", volume}, {"service", "inspect", service},
	} {
		if err := exec.CommandContext(ctx, "docker", resource...).Run(); err == nil {
			t.Fatalf("operation-owned Docker %s %s remains", resource[0], resource[2])
		}
	}
	if err := exec.CommandContext(ctx, "docker", "container", "inspect", preexisting).Run(); err != nil {
		t.Fatalf("pre-existing container was removed: %v", err)
	}
}

type integrationBaselineStore struct {
	baseline *operationcleanup.ResourceSet
}

func (s *integrationBaselineStore) ResourceBaseline(context.Context, control.OperationKind, string) (operationcleanup.ResourceSet, error) {
	if s.baseline == nil {
		return operationcleanup.ResourceSet{}, control.ErrNotFound
	}
	return *s.baseline, nil
}

func (s *integrationBaselineStore) SaveResourceBaseline(_ context.Context, _ control.OperationKind, _ string, resources operationcleanup.ResourceSet) error {
	if s.baseline != nil {
		return errors.New("baseline already exists")
	}
	s.baseline = &resources
	return nil
}

func dockerRun(t *testing.T, ctx context.Context, arguments ...string) {
	t.Helper()
	if output, err := exec.CommandContext(ctx, "docker", arguments...).CombinedOutput(); err != nil {
		t.Fatalf("docker %s: %v: %s", strings.Join(arguments, " "), err, strings.TrimSpace(string(output)))
	}
}

func swarmActive(ctx context.Context) bool {
	output, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Swarm.LocalNodeState}}").CombinedOutput()
	return err == nil && strings.TrimSpace(string(output)) == "active"
}

func waitForSwarmInfrastructure(t *testing.T, ctx context.Context) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		gatewayReady := exec.CommandContext(ctx, "docker", "network", "inspect", "docker_gwbridge").Run() == nil
		ingressReady := exec.CommandContext(ctx, "docker", "network", "inspect", "ingress").Run() == nil
		if gatewayReady && ingressReady {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Docker Swarm infrastructure did not converge: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func dockerCleanup(names ...string) {
	if len(names) != 6 {
		return
	}
	_ = exec.Command("docker", "service", "rm", names[5]).Run()
	_ = exec.Command("docker", "container", "rm", "--force", names[0], names[1], names[2]).Run()
	_ = exec.Command("docker", "network", "rm", names[3]).Run()
	_ = exec.Command("docker", "volume", "rm", "--force", names[4]).Run()
}
