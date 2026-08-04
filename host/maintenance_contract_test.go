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
	if _, err := os.Stat(filepath.Join(root, "host", "maintain.sh")); !os.IsNotExist(err) {
		t.Fatalf("legacy shell janitor still exists: %v", err)
	}
	maintenance := readFile(t, filepath.Join(root, "host", "autback-maintenance.service"))
	for _, required := range []string{"User=autback", "SupplementaryGroups=docker", "AmbientCapabilities=CAP_DAC_OVERRIDE", "CapabilityBoundingSet=CAP_DAC_OVERRIDE", "ProtectSystem=strict", "ReadWritePaths=/var/lib/autback", "autback-server maintain --json"} {
		if !strings.Contains(maintenance, required) {
			t.Errorf("maintenance service missing %q", required)
		}
	}
	server := readFile(t, filepath.Join(root, "host", "autback-server.service"))
	for _, required := range []string{"AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_DAC_OVERRIDE", "CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_DAC_OVERRIDE", "ProtectSystem=strict", "ReadWritePaths=/var/lib/autback"} {
		if !strings.Contains(server, required) {
			t.Errorf("server service missing %q", required)
		}
	}

	buildkit := readFile(t, filepath.Join(root, "host", "autback-buildkit.service"))
	for _, required := range []string{"/etc/autback/buildkitd.toml", "--config /etc/buildkit/buildkitd.toml", "--log-driver local"} {
		if !strings.Contains(buildkit, required) {
			t.Errorf("BuildKit service missing %q", required)
		}
	}
	if strings.Contains(buildkit, "oci-worker-gc-keepstorage") {
		t.Fatal("BuildKit still uses the legacy single keep-storage switch")
	}

	cas := readFile(t, filepath.Join(root, "host", "autback-cas.service"))
	for _, required := range []string{"--max_size ${AUTBACK_CAS_MAX_SIZE}", "--max_size_hard_limit ${AUTBACK_CAS_HARD_LIMIT}", "--log-driver local"} {
		if !strings.Contains(cas, required) {
			t.Errorf("CAS service missing %q", required)
		}
	}

	install := readFile(t, filepath.Join(root, "host", "install-swarm.sh"))
	for _, required := range []string{"AUTBACK_WORKER_OWNERSHIP=exclusive", "disk_bytes=", "disk_bytes / 4", "disk_bytes / 10", "minFreeSpace = \"20%\"", "keepDuration = \"48h\""} {
		if !strings.Contains(install, required) {
			t.Errorf("installer capacity policy missing %q", required)
		}
	}
	stop := strings.Index(install, "systemctl stop autback-maintenance.timer autback-maintenance.service autback-server")
	pull := strings.Index(install, "docker pull \"$cas_image\"")
	if stop < 0 || pull < 0 || stop >= pull {
		t.Fatal("installer must stop every lifecycle writer before pulling owned Docker images")
	}
}

func TestPublicConsoleSecretsRemainOutsideTheCommittedServiceEnvironment(t *testing.T) {
	root := repositoryRoot(t)
	service := readFile(t, filepath.Join(root, "host", "autback-server.service"))
	if !strings.Contains(service, "EnvironmentFile=-/etc/autback/auth.env") {
		t.Fatal("server does not load the root-owned optional authentication environment")
	}
	install := readFile(t, filepath.Join(root, "host", "install-swarm.sh"))
	for _, required := range []string{"AUTBACK_PUBLIC_URL", "AUTBACK_ACME_DOMAIN", "AUTBACK_ACME_EMAIL"} {
		if !strings.Contains(install, required) {
			t.Errorf("installer does not persist %s", required)
		}
	}
	if strings.Contains(install, "AUTBACK_GITHUB_CLIENT_SECRET") {
		t.Fatal("installer accepts the GitHub client secret through process arguments")
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
