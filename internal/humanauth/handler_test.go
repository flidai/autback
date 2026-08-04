package humanauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
)

func TestLoginRedirectUsesPKCEAndReturnsOnlyToAutback(t *testing.T) {
	store, _ := authStore(t)
	github := &fakeGitHub{}
	handler := newAuthHandler(t, store, github)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/login?return_to=%2Fapp%2Fprojects%2Fexample", nil))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Host != "github.example" || location.Query().Get("state") == "" || location.Query().Get("code_challenge") == "" {
		t.Fatalf("OAuth redirect = %s", location)
	}
	if location.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("challenge method = %q", location.Query().Get("code_challenge_method"))
	}
	cookie := response.Result().Cookies()[0]
	if cookie.Name != oauthStateCookie || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("state cookie = %#v", cookie)
	}

	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/auth/login?return_to=https%3A%2F%2Fevil.example", nil))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("external return status=%d", bad.Code)
	}
}

func TestCallbackCreatesAnAutbackBrowserSessionForALinkedGitHubIdentity(t *testing.T) {
	store, bootstrap := authStore(t)
	owner, _ := store.Authenticate(context.Background(), bootstrap.Token)
	if _, err := store.BindExternalIdentity(context.Background(), owner, bootstrap.User.ID, control.ExternalIdentity{Provider: "github", Subject: "1234", Login: "owner"}); err != nil {
		t.Fatal(err)
	}
	github := &fakeGitHub{identity: GitHubIdentity{Subject: "1234", Login: "owner-renamed"}}
	handler := newAuthHandler(t, store, github)
	state, stateCookie := beginLogin(t, handler, "/app")

	request := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=temporary-code&state="+url.QueryEscape(state), nil)
	request.AddCookie(stateCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/app" {
		t.Fatalf("status=%d location=%q body=%q", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	if github.exchangeCode != "temporary-code" || github.exchangeVerifier == "" {
		t.Fatalf("exchange = code %q verifier %q", github.exchangeCode, github.exchangeVerifier)
	}
	sessionCookie := cookieNamed(t, response.Result().Cookies(), sessionCookie)
	if !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %#v", sessionCookie)
	}
	principal, err := store.Authenticate(context.Background(), sessionCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Kind != control.PrincipalBrowser || principal.UserID != bootstrap.User.ID {
		t.Fatalf("principal = %#v", principal)
	}
	requireAuditAction(t, store, owner, "auth.github.login", bootstrap.User.ID)
}

func TestCallbackRejectsAnUnprovisionedGitHubIdentity(t *testing.T) {
	store, _ := authStore(t)
	handler := newAuthHandler(t, store, &fakeGitHub{identity: GitHubIdentity{Subject: "unbound", Login: "stranger"}})
	state, stateCookie := beginLogin(t, handler, "/app")
	request := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=code&state="+url.QueryEscape(state), nil)
	request.AddCookie(stateCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "not provisioned") {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookie && cookie.Value != "" {
			t.Fatal("unprovisioned identity received a session")
		}
	}
}

func TestDeviceLoginRequiresBrowserApprovalAndReturnsOneDeviceToken(t *testing.T) {
	store, bootstrap := authStore(t)
	handler := newAuthHandler(t, store, &fakeGitHub{})
	startBody := `{"device_name":"work-laptop"}`
	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/auth/cli/start", strings.NewReader(startBody)))
	if start.Code != http.StatusCreated {
		t.Fatalf("start status=%d body=%q", start.Code, start.Body.String())
	}
	var issued deviceStartResponse
	if err := json.NewDecoder(start.Body).Decode(&issued); err != nil {
		t.Fatal(err)
	}
	if issued.DeviceCode == "" || issued.UserCode == "" || !strings.Contains(issued.VerificationURIComplete, issued.UserCode) {
		t.Fatalf("issued login = %#v", issued)
	}

	pending := exchangeDevice(t, handler, issued.DeviceCode)
	if pending.Code != http.StatusAccepted {
		t.Fatalf("pending status=%d body=%q", pending.Code, pending.Body.String())
	}
	session, err := store.CreateBrowserSession(context.Background(), bootstrap.User.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	pageRequest := httptest.NewRequest(http.MethodGet, "/auth/device?code="+url.QueryEscape(issued.UserCode), nil)
	pageRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: session.Token})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, pageRequest)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "work-laptop") || !strings.Contains(page.Body.String(), issued.UserCode) {
		t.Fatalf("approval page status=%d body=%q", page.Code, page.Body.String())
	}
	approveRequest := httptest.NewRequest(http.MethodPost, "/auth/device/approve", strings.NewReader(url.Values{"code": {issued.UserCode}}.Encode()))
	approveRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: session.Token})
	approved := httptest.NewRecorder()
	handler.ServeHTTP(approved, approveRequest)
	if approved.Code != http.StatusOK || !strings.Contains(approved.Body.String(), "approved") {
		t.Fatalf("approve status=%d body=%q", approved.Code, approved.Body.String())
	}

	exchanged := exchangeDevice(t, handler, issued.DeviceCode)
	if exchanged.Code != http.StatusOK {
		t.Fatalf("exchange status=%d body=%q", exchanged.Code, exchanged.Body.String())
	}
	var token deviceTokenResponse
	if err := json.NewDecoder(exchanged.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	principal, err := store.Authenticate(context.Background(), token.Token)
	if err != nil || principal.Kind != control.PrincipalDevice || principal.UserID != bootstrap.User.ID {
		t.Fatalf("device principal=%#v err=%v", principal, err)
	}
	owner, err := store.Authenticate(context.Background(), bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	requireAuditAction(t, store, owner, "auth.device.approve", bootstrap.User.ID)
	requireAuditAction(t, store, owner, "auth.device.issue", token.TokenID)
	second := exchangeDevice(t, handler, issued.DeviceCode)
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("second exchange status=%d body=%q", second.Code, second.Body.String())
	}
}

func TestUnauthenticatedDeviceApprovalStartsGitHubLogin(t *testing.T) {
	store, _ := authStore(t)
	handler := newAuthHandler(t, store, &fakeGitHub{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/device?code=ABCD-EFGH", nil))
	if response.Code != http.StatusSeeOther || !strings.HasPrefix(response.Header().Get("Location"), "/auth/login?return_to=") {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
}

func TestLogoutRevokesBrowserSessionAndClearsCookie(t *testing.T) {
	store, bootstrap := authStore(t)
	handler := newAuthHandler(t, store, &fakeGitHub{})
	session, err := store.CreateBrowserSession(context.Background(), bootstrap.User.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	pageRequest := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	pageRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: session.Token})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, pageRequest)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Sign out") {
		t.Fatalf("logout page status=%d body=%q", page.Code, page.Body.String())
	}

	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.Header.Set("Origin", "https://console.autback.dev")
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: session.Token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/auth/login" {
		t.Fatalf("logout status=%d location=%q body=%q", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	cleared := cookieNamed(t, response.Result().Cookies(), sessionCookie)
	if cleared.MaxAge != -1 {
		t.Fatalf("session cookie was not cleared: %#v", cleared)
	}
	if _, err := store.Authenticate(context.Background(), session.Token); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("authenticate logged-out session: %v", err)
	}
	owner, err := store.Authenticate(context.Background(), bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	requireAuditAction(t, store, owner, "auth.browser.logout", bootstrap.User.ID)
}

func TestDeviceLoginCreationIsRateLimitedPerClient(t *testing.T) {
	store, _ := authStore(t)
	handler := newAuthHandler(t, store, &fakeGitHub{})
	for attempt := 0; attempt < 20; attempt++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/cli/start", strings.NewReader(`{"device_name":"work-laptop"}`)))
		if response.Code != http.StatusCreated {
			t.Fatalf("attempt %d status=%d", attempt, response.Code)
		}
	}
	limited := httptest.NewRecorder()
	handler.ServeHTTP(limited, httptest.NewRequest(http.MethodPost, "/auth/cli/start", strings.NewReader(`{"device_name":"work-laptop"}`)))
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%q", limited.Code, limited.Body.String())
	}
}

func TestWindowLimiterBoundsClientTracking(t *testing.T) {
	now := time.Now().UTC()
	limiter := newWindowLimiter(1, time.Minute, func() time.Time { return now })
	for index := 0; index < maxLimiterClients+100; index++ {
		limiter.Allow(fmt.Sprintf("client-%d", index))
	}
	if got := len(limiter.requests); got > maxLimiterClients {
		t.Fatalf("tracked clients = %d, want <= %d", got, maxLimiterClients)
	}
}

type fakeGitHub struct {
	identity         GitHubIdentity
	exchangeCode     string
	exchangeVerifier string
}

func (f *fakeGitHub) AuthorizationURL(state, challenge string) string {
	return "https://github.example/login/oauth/authorize?" + url.Values{
		"state": {state}, "code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}.Encode()
}

func (f *fakeGitHub) Exchange(_ context.Context, code, verifier string) (GitHubIdentity, error) {
	f.exchangeCode, f.exchangeVerifier = code, verifier
	return f.identity, nil
}

func newAuthHandler(t *testing.T, store *controlsqlite.Store, github GitHubProvider) http.Handler {
	t.Helper()
	handler, err := New(Config{
		Store: store, GitHub: github, PublicURL: "https://console.autback.dev",
		SessionTTL: 24 * time.Hour, LoginTTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func authStore(t *testing.T) (*controlsqlite.Store, control.BootstrapResult) {
	t.Helper()
	store, err := controlsqlite.Open(filepath.Join(t.TempDir(), "control"), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bootstrap, err := store.Bootstrap(context.Background(), control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	return store, bootstrap
}

func beginLogin(t *testing.T, handler http.Handler, returnTo string) (string, *http.Cookie) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/login?return_to="+url.QueryEscape(returnTo), nil))
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return location.Query().Get("state"), cookieNamed(t, response.Result().Cookies(), oauthStateCookie)
}

func cookieNamed(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}

func exchangeDevice(t *testing.T, handler http.Handler, deviceCode string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(deviceTokenRequest{DeviceCode: deviceCode})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/cli/token", strings.NewReader(string(body))))
	return response
}

func requireAuditAction(t *testing.T, store *controlsqlite.Store, principal control.Principal, action, targetID string) {
	t.Helper()
	events, err := store.ListAuditEvents(context.Background(), principal, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Action == action && event.TargetID == targetID {
			return
		}
	}
	t.Fatalf("audit action %q for %q not found in %#v", action, targetID, events)
}
