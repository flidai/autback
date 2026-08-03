package host_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkerMaintenanceTargetsOwnedDockerStorage(t *testing.T) {
	root := repositoryRoot(t)
	maintain := readFile(t, filepath.Join(root, "host", "maintain.sh"))
	for _, required := range []string{
		"docker volume prune --force",
		"docker image prune --force --filter 'until=24h'",
		"docker exec autback-buildkit buildctl --addr tcp://127.0.0.1:1234 prune",
		"--keep-storage 4000",
		"--keep-storage 10000",
		"--all",
	} {
		if !strings.Contains(maintain, required) {
			t.Errorf("host maintenance does not own %q", required)
		}
	}
	if strings.Contains(maintain, "docker builder prune") {
		t.Fatal("host maintenance prunes Docker's default builder instead of the Autback BuildKit daemon")
	}

	service := readFile(t, filepath.Join(root, "host", "autback-buildkit.service"))
	if !strings.Contains(service, "--oci-worker-gc-keepstorage=10000") {
		t.Fatal("Autback BuildKit must keep at most 10 GB before its internal garbage collector runs")
	}
	if strings.Contains(service, "--oci-worker-gc-keepstorage=30000") {
		t.Fatal("Autback BuildKit retains 30 GB despite the documented 10 GB worker budget")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	return filepath.Dir(filepath.Dir(source))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
