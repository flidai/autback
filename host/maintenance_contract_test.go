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
	for _, required := range []string{
		"docker volume create --label autback.managed=true autback-buildkit-state",
		"--label autback.managed=true",
		"/etc/autback/buildkitd.toml",
		"--config /etc/buildkit/buildkitd.toml",
		"--log-driver local",
	} {
		if !strings.Contains(buildkit, required) {
			t.Errorf("BuildKit service missing %q", required)
		}
	}
	if strings.Contains(buildkit, "oci-worker-gc-keepstorage") {
		t.Fatal("BuildKit still uses the legacy single keep-storage switch")
	}

	cas := readFile(t, filepath.Join(root, "host", "autback-cas.service"))
	for _, required := range []string{
		"--label autback.managed=true",
		"--max_size ${AUTBACK_CAS_MAX_SIZE}",
		"--max_size_hard_limit ${AUTBACK_CAS_HARD_LIMIT}",
		"--log-driver local",
	} {
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

func TestWorkerResourcePolicyReservesControlPlaneHeadroom(t *testing.T) {
	root := repositoryRoot(t)
	server := readFile(t, filepath.Join(root, "host", "autback-server.service"))
	for _, required := range []string{"MemoryHigh=512M", "MemoryMax=768M", "TasksMax=1024"} {
		if !strings.Contains(server, required) {
			t.Errorf("server service missing %q", required)
		}
	}
	buildkit := readFile(t, filepath.Join(root, "host", "autback-buildkit.service"))
	for _, required := range []string{"--memory ${AUTBACK_BUILDKIT_MEMORY_LIMIT}", "--memory-reservation ${AUTBACK_BUILDKIT_MEMORY_RESERVATION}", "--cpus ${AUTBACK_BUILDKIT_CPU_LIMIT}", "--cgroup-parent autback-infrastructure.slice"} {
		if !strings.Contains(buildkit, required) {
			t.Errorf("BuildKit service missing %q", required)
		}
	}
	cas := readFile(t, filepath.Join(root, "host", "autback-cas.service"))
	if !strings.Contains(cas, "--cgroup-parent autback-infrastructure.slice") {
		t.Fatal("CAS is not isolated from the workload slice")
	}
	install := readFile(t, filepath.Join(root, "host", "install-swarm.sh"))
	for _, required := range []string{"autback-workloads.slice", `config["cgroup-parent"] = "autback-workloads.slice"`, "AUTBACK_JOB_MEMORY_LIMIT_BYTES", "AUTBACK_JOB_CPU_LIMIT_NANO", "AUTBACK_JOB_PIDS_LIMIT"} {
		if !strings.Contains(install, required) {
			t.Errorf("installer resource policy missing %q", required)
		}
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
	for _, required := range []string{"existing_service_env=/etc/autback/service.env", "read_existing_setting", "AUTBACK_GITHUB_OIDC_AUDIENCES", "oidc_audiences"} {
		if !strings.Contains(install, required) {
			t.Errorf("installer does not preserve migration configuration through %q", required)
		}
	}
	if strings.Contains(install, "AUTBACK_GITHUB_OIDC_AUDIENCE=https://%s") {
		t.Fatal("installer still replaces the OIDC audience with one server name")
	}
	deploy := readFile(t, filepath.Join(root, "scripts", "deploy-swarm.zsh"))
	for _, required := range []string{"/etc/autback/service.env", "remote_service_setting", "AUTBACK_GITHUB_OIDC_AUDIENCES"} {
		if !strings.Contains(deploy, required) {
			t.Errorf("deployment does not preserve installed service configuration through %q", required)
		}
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
