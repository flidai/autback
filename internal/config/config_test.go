package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flidai/autback/internal/config"
)

func TestLoadReadsServiceConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "url": "https://autback.example",
  "service": {
    "image": "ghcr.io/example/ci@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "ca_cert_file": "/tmp/autback-ca.pem",
    "oidc_audience": "https://autback.example"
  }
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTBACK_CONFIG", path)
	t.Setenv("AUTBACK_URL", "")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://autback.example" || got.Service == nil || got.Service.CACertFile != "/tmp/autback-ca.pem" {
		t.Fatalf("config = %#v", got)
	}
}

func TestEnvironmentOverridesServiceURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"url":"https://old.invalid","service":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTBACK_CONFIG", path)
	t.Setenv("AUTBACK_URL", "https://autback.example")

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://autback.example" {
		t.Fatalf("url = %q", got.URL)
	}
}

func TestLoadRejectsRemovedClientBackends(t *testing.T) {
	for _, backend := range []string{"legacy", "reapi", "swarm", "service"} {
		t.Run(backend, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			data := []byte(`{"backend":"` + backend + `","url":"https://autback.example","service":{}}`)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AUTBACK_CONFIG", path)
			t.Setenv("AUTBACK_URL", "")
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
			data := []byte(`{"url":"https://autback.example","service":{},"` + field + `":{}}`)
			if field == "token" {
				data = []byte(`{"url":"https://autback.example","service":{},"token":"secret"}`)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AUTBACK_CONFIG", path)
			t.Setenv("AUTBACK_URL", "")
			if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}

func TestLoadRejectsGlobalProjectDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"url":"https://autback.example","service":{"project":"example"}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTBACK_CONFIG", path)
	t.Setenv("AUTBACK_URL", "")
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadAppliesServiceDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"url":"https://autback.example","service":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTBACK_CONFIG", path)
	t.Setenv("AUTBACK_URL", "")
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Service.OIDCAudience != got.URL {
		t.Fatalf("service defaults = %#v", got.Service)
	}
}
