package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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

func TestServiceHandlerKeepsTheConsoleOutsideTheConnectControlPlane(t *testing.T) {
	controlHandler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("X-Handler", "control")
	})
	consoleHandler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("X-Handler", "console")
	})
	handler := serviceHandler(controlHandler, consoleHandler)
	for _, test := range []struct{ path, want string }{
		{"/app", "console"}, {"/app/updates", "console"}, {"/rtest.v1.ControlService/GetServiceInfo", "control"}, {"/healthz", "control"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if got := response.Header().Get("X-Handler"); got != test.want {
			t.Errorf("%s handler=%q; want %q", test.path, got, test.want)
		}
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
