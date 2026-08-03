package swarm

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flidai/autback/internal/protocol"
)

func TestCreateArgsUseReplicatedJobAndSamePathWorkspace(t *testing.T) {
	spec := Spec{
		ID:    "autback-job-1",
		Image: "runner:test", CASAddress: "127.0.0.1:50051", CASInstance: "autback",
		RootDigest: "abc/123", JobsRoot: "/var/lib/autback/jobs", Command: []string{"go", "test", "./..."},
		Timeout:          15 * time.Minute,
		WorkingDirectory: "cmd/service", Environment: map[string]string{"AUTBACK_PROOF": "generic-oci"},
		EntrypointHostPath: "/usr/local/lib/autback/autback-job-entrypoint",
		HostUID:            "123", HostGID: "456",
		CacheRoot: "/var/lib/autback/cache", ProjectID: "prj-example",
		Caches: []CacheMount{{Name: "go-build", Target: "/root/.cache/go-build"}, {Name: "modules", Target: "/go/pkg/mod"}},
	}
	args := CreateArgs(spec)
	for _, sequence := range [][]string{
		{"--mode", "replicated-job"},
		{"--restart-condition", "none"},
		{"--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=2"},
		{"--network", "host"},
		{"--mount", "type=tmpfs,dst=/dev/shm,tmpfs-size=1073741824"},
		{"--mount", "type=bind,src=/var/lib/autback/jobs,dst=/var/lib/autback/jobs"},
		{"--mount", "type=bind,src=/var/lib/autback/cache/prj-example/go-build,dst=/root/.cache/go-build"},
		{"--mount", "type=bind,src=/var/lib/autback/cache/prj-example/modules,dst=/go/pkg/mod"},
		{"--mount", "type=bind,src=/usr/local/lib/autback/autback-job-entrypoint,dst=/usr/local/bin/autback-job-entrypoint,readonly"},
		{"--label", "autback.project=prj-example"},
		{"--label", "autback.job=autback-job-1"},
		{"--label", "autback.image=cnVubmVyOnRlc3Q"},
		{"--env", "AUTBACK_WORKSPACE=/var/lib/autback/jobs/autback-job-1/workspace"},
		{"--env", "AUTBACK_HOST_UID=123"},
		{"--env", "AUTBACK_HOST_GID=456"},
		{"--env", "AUTBACK_ROOT_DIGEST=abc/123"},
		{"--env", "AUTBACK_WORKING_DIRECTORY=cmd/service"},
		{"--env", "TMPDIR=/tmp"},
		{"--env", "AUTBACK_PROOF=generic-oci"},
		{"--entrypoint", "/usr/local/bin/autback-job-entrypoint"},
		{"--user", "0:0"},
		{"runner:test", "go", "test", "./..."},
	} {
		if !containsSequence(args, sequence) {
			t.Fatalf("args %#v missing sequence %#v", args, sequence)
		}
	}
	for _, forbidden := range []string{"autback-go-build-cache", "autback-go-mod-cache", "--limit-cpu", "--reserve-cpu", "--limit-memory", "--reserve-memory", "AUTBACK_WORKER_LOCK"} {
		if strings.Contains(strings.Join(args, "\n"), forbidden) {
			t.Fatalf("args %#v contain global cache %q", args, forbidden)
		}
	}
}

func TestTaskStatusMapsDockerLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		state     string
		exitCode  int
		cancelled bool
		want      protocol.Status
	}{
		{name: "pending", state: "pending", want: protocol.StatusQueued},
		{name: "running", state: "running", want: protocol.StatusRunning},
		{name: "success", state: "complete", want: protocol.StatusSucceeded},
		{name: "failure", state: "failed", exitCode: 2, want: protocol.StatusFailed},
		{name: "timeout", state: "failed", exitCode: 124, want: protocol.StatusTimedOut},
		{name: "cancel", state: "shutdown", cancelled: true, want: protocol.StatusCancelled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := taskStatus(test.state, test.exitCode, test.cancelled); got != test.want {
				t.Fatalf("taskStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

func containsSequence(values, sequence []string) bool {
	for i := 0; i+len(sequence) <= len(values); i++ {
		if slices.Equal(values[i:i+len(sequence)], sequence) {
			return true
		}
	}
	return false
}
