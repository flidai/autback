package humanauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGitHubAuthorizationURLUsesCodeFlowWithPKCE(t *testing.T) {
	client, err := NewGitHub(GitHubConfig{
		ClientID: "client-id", ClientSecret: "client-secret", CallbackURL: "https://console.autback.dev/auth/github/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse(client.AuthorizationURL("state", "challenge"))
	if err != nil {
		t.Fatal(err)
	}
	query := target.Query()
	for key, want := range map[string]string{
		"client_id": "client-id", "redirect_uri": "https://console.autback.dev/auth/github/callback",
		"state": "state", "code_challenge": "challenge", "code_challenge_method": "S256", "allow_signup": "false",
	} {
		if query.Get(key) != want {
			t.Fatalf("%s=%q, want %q in %s", key, query.Get(key), want, target)
		}
	}
	if query.Get("scope") != "" {
		t.Fatalf("identity-only login requested scope %q", query.Get("scope"))
	}
}

func TestGitHubExchangeValidatesTheAuthenticatedImmutableUserID(t *testing.T) {
	var exchanged url.Values
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/login/oauth/access_token":
			if err := request.ParseForm(); err != nil {
				t.Fatal(err)
			}
			exchanged = request.Form
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"access_token":"github-user-token","token_type":"bearer"}`))
		case "/user":
			if request.Header.Get("Authorization") != "Bearer github-user-token" {
				t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(response).Encode(map[string]any{"id": 12345678, "login": "yacobolo", "name": "Jacob"})
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	client, err := NewGitHub(GitHubConfig{
		ClientID: "client-id", ClientSecret: "client-secret", CallbackURL: "https://console.autback.dev/auth/github/callback",
		AuthorizeURL: server.URL + "/login/oauth/authorize", TokenURL: server.URL + "/login/oauth/access_token", APIURL: server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := client.Exchange(context.Background(), "temporary-code", "verifier")
	if err != nil {
		t.Fatal(err)
	}
	if identity != (GitHubIdentity{Subject: "12345678", Login: "yacobolo", Name: "Jacob"}) {
		t.Fatalf("identity = %#v", identity)
	}
	for key, want := range map[string]string{
		"client_id": "client-id", "client_secret": "client-secret", "code": "temporary-code",
		"redirect_uri": "https://console.autback.dev/auth/github/callback", "code_verifier": "verifier",
	} {
		if exchanged.Get(key) != want {
			t.Fatalf("exchange %s=%q, want %q", key, exchanged.Get(key), want)
		}
	}
}

func TestGitHubLookupResolvesLoginToImmutableUserID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/users/Yacobolo" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"id": 12345678, "login": "Yacobolo", "name": "Jacob"})
	}))
	defer server.Close()
	client, err := NewGitHub(GitHubConfig{
		ClientID: "client-id", ClientSecret: "client-secret", CallbackURL: "https://console.autback.dev/auth/github/callback",
		APIURL: server.URL, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := client.Lookup(context.Background(), "Yacobolo")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "12345678" || identity.Login != "Yacobolo" {
		t.Fatalf("identity = %#v", identity)
	}
}
