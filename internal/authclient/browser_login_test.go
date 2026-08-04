package authclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBrowserLoginClientStartsAndPollsAnAutbackDeviceLogin(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/auth/cli/start":
			var input struct {
				DeviceName string `json:"device_name"`
			}
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.DeviceName != "work-laptop" {
				t.Fatalf("start input=%#v err=%v", input, err)
			}
			response.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(response).Encode(map[string]any{
				"device_code": "secret-code", "user_code": "ABCD-EFGH",
				"verification_uri":          serverURL(request) + "/auth/device",
				"verification_uri_complete": serverURL(request) + "/auth/device?code=ABCD-EFGH",
				"expires_in_seconds":        600, "interval_seconds": 1,
			})
		case "/auth/cli/token":
			_ = json.NewEncoder(response).Encode(map[string]any{"status": "authorized", "token": "autback_dt_token_secret", "token_id": "token"})
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	client, err := NewBrowserLoginClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	login, err := client.Start(context.Background(), "work-laptop")
	if err != nil {
		t.Fatal(err)
	}
	if login.DeviceCode != "secret-code" || login.UserCode != "ABCD-EFGH" || login.Interval != time.Second {
		t.Fatalf("login = %#v", login)
	}
	token, pending, err := client.Poll(context.Background(), login.DeviceCode)
	if err != nil || pending || token.Token != "autback_dt_token_secret" {
		t.Fatalf("token=%#v pending=%v err=%v", token, pending, err)
	}
}

func TestBrowserLoginClientReportsPendingWithoutTreatingItAsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(response).Encode(map[string]any{"status": "authorization_pending", "interval_seconds": 2})
	}))
	defer server.Close()
	client, err := NewBrowserLoginClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, pending, err := client.Poll(context.Background(), "secret-code")
	if err != nil || !pending {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
}

func TestBrowserLoginClientRejectsCleartextRemoteServices(t *testing.T) {
	if _, err := NewBrowserLoginClient("http://autback.example.com", http.DefaultClient); err == nil {
		t.Fatal("remote cleartext browser login was accepted")
	}
}

func serverURL(request *http.Request) string { return "https://" + request.Host }
