package cli

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestConsoleProxyExchangesAnEphemeralLocalSessionAndInjectsTheDeviceToken(t *testing.T) {
	requests := make(chan *http.Request, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(request.Context())
		response.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := newConsoleProxy(target, "device-secret", "local-session", upstream.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/app", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}

	exchange := httptest.NewRecorder()
	handler.ServeHTTP(exchange, httptest.NewRequest(http.MethodGet, "/session/local-session", nil))
	if exchange.Code != http.StatusSeeOther || exchange.Header().Get("Location") != "/app" {
		t.Fatalf("exchange status=%d location=%q", exchange.Code, exchange.Header().Get("Location"))
	}
	cookies := exchange.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Value == "device-secret" || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookies=%#v", cookies)
	}

	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("proxy status=%d body=%q", response.Code, response.Body.String())
	}
	forwarded := <-requests
	if got := forwarded.Header.Get("Authorization"); got != "Bearer device-secret" {
		t.Fatalf("Authorization=%q", got)
	}
	if forwarded.URL.Path != "/app" {
		t.Fatalf("path=%q", forwarded.URL.Path)
	}
}

func TestConsoleProxyCannotReachTheControlAPI(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	handler, err := newConsoleProxy(target, "device-secret", "local-session", upstream.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/rtest.v1.ControlService/CreateProject", nil)
	request.AddCookie(&http.Cookie{Name: consoleSessionCookie, Value: "local-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d; control API must not be reachable through the browser proxy", response.Code)
	}
}
