package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control/pki"
)

func TestControlTLSUsesACMEForThePublicNameAndPrivatePKIForLegacyClients(t *testing.T) {
	pkiDir := filepath.Join(t.TempDir(), "pki")
	if _, err := pki.Ensure(pkiDir, []string{"console.autback.dev", "62.238.54.70"}); err != nil {
		t.Fatal(err)
	}
	config, certificate, key, err := controlTLS(t.TempDir(), pkiDir, []string{"console.autback.dev", "62.238.54.70"}, "console.autback.dev", "ops@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if certificate != "" || key != "" || config.GetCertificate == nil || config.MinVersion != tls.VersionTLS13 || len(config.Certificates) != 1 {
		t.Fatalf("config=%#v certificate=%q key=%q", config, certificate, key)
	}
	legacy, err := config.GetCertificate(&tls.ClientHelloInfo{ServerName: "62.238.54.70"})
	if err != nil || legacy == nil || len(legacy.Certificate) == 0 {
		t.Fatalf("legacy certificate=%#v err=%v", legacy, err)
	}
	foundALPN := false
	for _, protocol := range config.NextProtos {
		foundALPN = foundALPN || protocol == "acme-tls/1"
	}
	if !foundALPN {
		t.Fatalf("ACME ALPN missing from %#v", config.NextProtos)
	}
	if _, _, _, err := controlTLS(t.TempDir(), pkiDir, []string{"62.238.54.70"}, "console.autback.dev", ""); err == nil {
		t.Fatal("ACME domain outside AUTBACK_SERVER_NAMES was accepted")
	}
}

func TestEndpointUsesThePublicServerNameAndListenerPort(t *testing.T) {
	for _, test := range []struct {
		name, listen, want string
	}{
		{"autback.example.com", ":50052", "autback.example.com:50052"},
		{"127.0.0.1", "127.0.0.1:1235", "127.0.0.1:1235"},
		{"2001:db8::1", "[::]:50052", "[2001:db8::1]:50052"},
	} {
		if got := endpoint(test.name, test.listen); got != test.want {
			t.Errorf("endpoint(%q, %q) = %q, want %q", test.name, test.listen, got, test.want)
		}
	}
}

func TestSplitNamesRejectsEmptyConfiguration(t *testing.T) {
	if _, err := splitNames(" , \t,"); err == nil {
		t.Fatal("splitNames error = nil")
	}
	names, err := splitNames(" autback.example.com, 127.0.0.1 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "autback.example.com" || names[1] != "127.0.0.1" {
		t.Fatalf("names = %#v", names)
	}
}

func TestServiceHandlerKeepsTheConsoleOutsideTheConnectControlPlane(t *testing.T) {
	controlHandler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("X-Handler", "control")
	})
	consoleHandler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("X-Handler", "console")
	})
	authHandler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("X-Handler", "auth")
	})
	handler := serviceHandler(controlHandler, consoleHandler, authHandler)
	for _, test := range []struct{ path, want string }{
		{"/app", "console"}, {"/app/updates", "console"}, {"/auth/login", "auth"}, {"/auth/cli/start", "auth"},
		{"/rtest.v1.ControlService/GetServiceInfo", "control"}, {"/healthz", "control"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if got := response.Header().Get("X-Handler"); got != test.want {
			t.Errorf("%s handler=%q; want %q", test.path, got, test.want)
		}
	}
}

func TestServiceHandlerDoesNotExposeAuthWhenGitHubLoginIsDisabled(t *testing.T) {
	handler := serviceHandler(http.NotFoundHandler(), http.NotFoundHandler(), nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestReconcilerRunsImmediatelyWithoutBlockingStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &recordingReconciler{called: make(chan struct{}, 1)}
	go runReconciler(ctx, runner, time.Hour)
	select {
	case <-runner.called:
	case <-time.After(time.Second):
		t.Fatal("reconciler did not run immediately")
	}
}

type recordingReconciler struct{ called chan struct{} }

func (r *recordingReconciler) RunOnce(context.Context) error {
	r.called <- struct{}{}
	return nil
}
