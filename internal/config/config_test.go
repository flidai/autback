package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flidai/outback/internal/config"
)

func TestLoadReadsSecureClientConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "token": "secret",
  "ssh": {
    "host": "203.0.113.10",
    "user": "root",
    "identity_file": "/tmp/operator",
    "remote_address": "127.0.0.1:8080"
  }
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	t.Setenv("OUTBACK_TOKEN", "")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "secret" || got.SSH == nil || got.SSH.User != "root" || got.SSH.RemoteAddress != "127.0.0.1:8080" {
		t.Fatalf("config = %#v", got)
	}
}

func TestEnvironmentOverridesConfigurationFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"url":"http://old.invalid","token":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "http://127.0.0.1:9999")
	t.Setenv("OUTBACK_TOKEN", "new")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "http://127.0.0.1:9999" || got.Token != "new" {
		t.Fatalf("config = %#v", got)
	}
}

func TestLoadReadsREAPIConfigurationWithoutLegacyToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "backend": "reapi",
  "ssh": {
    "host": "203.0.113.10",
    "identity_file": "/tmp/operator"
  },
  "reapi": {
    "instance": "outback"
  },
  "buildkit": {}
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	t.Setenv("OUTBACK_TOKEN", "")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Backend != config.BackendREAPI || got.REAPI == nil || got.REAPI.RemoteAddress != "127.0.0.1:50051" {
		t.Fatalf("config = %#v", got)
	}
	if got.BuildKit == nil || got.BuildKit.RemoteAddress != "127.0.0.1:1234" {
		t.Fatalf("buildkit config = %#v", got.BuildKit)
	}
}

func TestLoadReadsSwarmConfigurationWithoutLegacyToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "backend": "swarm",
  "ssh": {
    "host": "203.0.113.10",
    "identity_file": "/tmp/operator"
  },
  "cas": {
    "instance": "outback"
  },
  "swarm": {}
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	t.Setenv("OUTBACK_TOKEN", "")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Backend != config.BackendSwarm || got.CAS == nil || got.CAS.RemoteAddress != "127.0.0.1:50051" {
		t.Fatalf("config = %#v", got)
	}
	if got.Swarm == nil || got.Swarm.JobsRoot != "/var/lib/outback/jobs" || got.Swarm.Image != "outback-runner-standard:local" {
		t.Fatalf("swarm config = %#v", got.Swarm)
	}
}

func TestLoadRejectsSilentGlobalServiceProject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "backend": "service",
  "url": "https://outback.example",
  "service": {
    "project": "example-service",
    "image": "ghcr.io/example/ci@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "cpus": "2",
    "memory": "4g",
    "ca_cert_file": "/tmp/outback-ca.pem",
    "oidc_audience": "https://outback.example"
  }
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	t.Setenv("OUTBACK_TOKEN", "")

	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "outback.json") {
		t.Fatalf("global project error = %v", err)
	}
}

func TestLoadReadsSharedServiceConfigurationWithoutProjectOrPersistedToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "backend": "service",
  "url": "https://outback.example",
  "service": {
    "image": "ghcr.io/example/ci@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "cpus": "2",
    "memory": "4g"
  }
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	t.Setenv("OUTBACK_TOKEN", "")
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Backend != config.BackendService || got.Service == nil || got.Token != "" {
		t.Fatalf("config = %#v", got)
	}
}
