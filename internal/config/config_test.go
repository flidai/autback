package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flidai/outback/internal/config"
)

func TestLoadReadsServiceConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "url": "https://outback.example",
  "service": {
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

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://outback.example" || got.Service == nil || got.Service.CACertFile != "/tmp/outback-ca.pem" {
		t.Fatalf("config = %#v", got)
	}
}

func TestEnvironmentOverridesServiceURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"url":"https://old.invalid","service":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "https://outback.example")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://outback.example" {
		t.Fatalf("url = %q", got.URL)
	}
}

func TestLoadRejectsRemovedClientBackends(t *testing.T) {
	for _, backend := range []string{"legacy", "reapi", "swarm", "service"} {
		t.Run(backend, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			data := []byte(`{"backend":"` + backend + `","url":"https://outback.example","service":{}}`)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("OUTBACK_CONFIG", path)
			t.Setenv("OUTBACK_URL", "")
			if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}

func TestLoadRejectsPersistedCredentialsAndTransportTunnels(t *testing.T) {
	for _, field := range []string{"token", "ssh", "reapi", "cas", "swarm", "buildkit"} {
		t.Run(field, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			data := []byte(`{"url":"https://outback.example","service":{},"` + field + `":{}}`)
			if field == "token" {
				data = []byte(`{"url":"https://outback.example","service":{},"token":"secret"}`)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("OUTBACK_CONFIG", path)
			t.Setenv("OUTBACK_URL", "")
			if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}

func TestLoadRejectsGlobalProjectDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"url":"https://outback.example","service":{"project":"example"}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadAppliesServiceDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"url":"https://outback.example","service":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUTBACK_CONFIG", path)
	t.Setenv("OUTBACK_URL", "")
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Service.CPUs != "2" || got.Service.Memory != "4g" || got.Service.OIDCAudience != got.URL {
		t.Fatalf("service defaults = %#v", got.Service)
	}
}
