package controlclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	autbackv1 "github.com/flidai/autback/internal/gen/rtest/v1"
	"github.com/flidai/autback/internal/version"
)

func TestClientSendsVersionAndCapabilities(t *testing.T) {
	var versionHeader, capabilitiesHeader string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		versionHeader = request.Header.Get(version.ClientVersionHeader)
		capabilitiesHeader = request.Header.Get(version.ClientCapabilitiesHeader)
		http.Error(response, "test response", http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := New(server.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.GetServiceInfo(context.Background(), connect.NewRequest(&autbackv1.GetServiceInfoRequest{}))
	if versionHeader != version.Current {
		t.Fatalf("client version header = %q, want %q", versionHeader, version.Current)
	}
	if capabilitiesHeader != version.CapabilityBuildLeaseHeartbeat {
		t.Fatalf("client capabilities header = %q, want %q", capabilitiesHeader, version.CapabilityBuildLeaseHeartbeat)
	}
}
