package humanauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const (
	oauthStateCookie  = "autback_oauth_state"
	sessionCookie     = "autback_session"
	maxLimiterClients = 4096
)

//go:embed assets/*
var assets embed.FS

type GitHubIdentity struct {
	Subject string
	Login   string
	Name    string
}

type GitHubProvider interface {
	AuthorizationURL(state, codeChallenge string) string
	Exchange(context.Context, string, string) (GitHubIdentity, error)
}

type Config struct {
	Store      *controlsqlite.Store
	GitHub     GitHubProvider
	PublicURL  string
	SessionTTL time.Duration
	LoginTTL   time.Duration
	Now        func() time.Time
}

type server struct {
	store      *controlsqlite.Store
	github     GitHubProvider
	publicURL  *url.URL
	sessionTTL time.Duration
	loginTTL   time.Duration
	now        func() time.Time
	loginLimit *windowLimiter
	startLimit *windowLimiter
	pollLimit  *windowLimiter
}

type deviceStartRequest struct {
	DeviceName string `json:"device_name"`
}

type deviceStartResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresInSeconds        int64  `json:"expires_in_seconds"`
	IntervalSeconds         int64  `json:"interval_seconds"`
}

type deviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

type deviceTokenResponse struct {
	Status          string `json:"status"`
	Token           string `json:"token,omitempty"`
	TokenID         string `json:"token_id,omitempty"`
	IntervalSeconds int64  `json:"interval_seconds,omitempty"`
}

func New(config Config) (http.Handler, error) {
	if config.Store == nil || config.GitHub == nil {
		return nil, errors.New("human auth store and GitHub provider are required")
	}
	publicURL, err := url.Parse(strings.TrimRight(config.PublicURL, "/"))
	if err != nil || publicURL.Host == "" || publicURL.Scheme != "https" && !(publicURL.Scheme == "http" && isLoopbackHost(publicURL.Hostname())) {
		return nil, errors.New("human auth public URL must use HTTPS or loopback HTTP")
	}
	if config.SessionTTL == 0 {
		config.SessionTTL = 7 * 24 * time.Hour
	}
	if config.SessionTTL < time.Minute || config.SessionTTL > 30*24*time.Hour {
		return nil, errors.New("browser session TTL must be between one minute and 30 days")
	}
	if config.LoginTTL == 0 {
		config.LoginTTL = 10 * time.Minute
	}
	if config.LoginTTL < time.Minute || config.LoginTTL > 15*time.Minute {
		return nil, errors.New("login TTL must be between one and 15 minutes")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	assetFS, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, err
	}
	s := &server{
		store: config.Store, github: config.GitHub, publicURL: publicURL, sessionTTL: config.SessionTTL, loginTTL: config.LoginTTL, now: config.Now,
		loginLimit: newWindowLimiter(30, time.Minute, config.Now), startLimit: newWindowLimiter(20, time.Minute, config.Now), pollLimit: newWindowLimiter(300, time.Minute, config.Now),
	}
	mux := http.NewServeMux()
	mux.Handle("GET /auth/assets/", http.StripPrefix("/auth/assets/", http.FileServer(http.FS(assetFS))))
	mux.HandleFunc("GET /auth/login", s.login)
	mux.HandleFunc("GET /auth/github/callback", s.callback)
	mux.HandleFunc("GET /auth/logout", s.logoutPage)
	mux.HandleFunc("POST /auth/logout", s.logout)
	mux.HandleFunc("GET /auth/device", s.devicePage)
	mux.HandleFunc("POST /auth/device/approve", s.approveDevice)
	mux.HandleFunc("POST /auth/cli/start", s.startDevice)
	mux.HandleFunc("POST /auth/cli/token", s.exchangeDevice)
	return securityHeaders(mux), nil
}

func (s *server) logoutPage(response http.ResponseWriter, request *http.Request) {
	if _, ok := s.browserPrincipal(request); !ok {
		http.Redirect(response, request, "/auth/login", http.StatusSeeOther)
		return
	}
	s.render(response, http.StatusOK, "Sign out", h.Div(
		h.P(g.Text("End this browser session on Autback? Your device credentials remain active.")),
		h.Form(h.Method(http.MethodPost), h.Action("/auth/logout"), h.Button(h.Type("submit"), g.Text("Sign out"))),
	))
}

func (s *server) logout(response http.ResponseWriter, request *http.Request) {
	if !sameOrigin(request, s.publicURL) {
		http.Error(response, "invalid origin", http.StatusForbidden)
		return
	}
	cookie, err := request.Cookie(sessionCookie)
	if err == nil && cookie.Value != "" {
		principal, authenticated := s.browserPrincipal(request)
		if authenticated {
			if err := s.store.RevokeBrowserSession(request.Context(), cookie.Value); err != nil {
				http.Error(response, "sign out unavailable", http.StatusInternalServerError)
				return
			}
			if err := s.store.Audit(request.Context(), principal, "", "auth.browser.logout", principal.UserID, nil); err != nil {
				http.Error(response, "sign out unavailable", http.StatusInternalServerError)
				return
			}
		}
	}
	http.SetCookie(response, &http.Cookie{
		Name: sessionCookie, Path: "/", MaxAge: -1,
		Secure: s.publicURL.Scheme == "https", HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(response, request, "/auth/login", http.StatusSeeOther)
}

func (s *server) login(response http.ResponseWriter, request *http.Request) {
	if !s.loginLimit.Allow(clientAddress(request)) {
		http.Error(response, "too many login attempts", http.StatusTooManyRequests)
		return
	}
	returnTo := request.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/app"
	}
	if !validReturnTo(returnTo) {
		http.Error(response, "invalid return path", http.StatusBadRequest)
		return
	}
	verifier, err := randomValue(32)
	if err != nil {
		http.Error(response, "login unavailable", http.StatusInternalServerError)
		return
	}
	state, err := s.store.CreateOAuthLoginState(request.Context(), returnTo, verifier, s.now().UTC().Add(s.loginTTL))
	if err != nil {
		http.Error(response, "login unavailable", http.StatusInternalServerError)
		return
	}
	http.SetCookie(response, &http.Cookie{
		Name: oauthStateCookie, Value: state.State, Path: "/auth/github/callback", MaxAge: int(s.loginTTL.Seconds()),
		Secure: s.publicURL.Scheme == "https", HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(response, request, s.github.AuthorizationURL(state.State, pkceChallenge(verifier)), http.StatusSeeOther)
}

func (s *server) callback(response http.ResponseWriter, request *http.Request) {
	state := request.URL.Query().Get("state")
	cookie, err := request.Cookie(oauthStateCookie)
	if err != nil || !constantEqual(cookie.Value, state) || request.URL.Query().Get("code") == "" {
		http.Error(response, "invalid login state", http.StatusUnauthorized)
		return
	}
	transaction, err := s.store.ConsumeOAuthLoginState(request.Context(), state, s.now().UTC())
	if err != nil {
		http.Error(response, "invalid or expired login", http.StatusUnauthorized)
		return
	}
	identity, err := s.github.Exchange(request.Context(), request.URL.Query().Get("code"), transaction.CodeVerifier)
	if err != nil || identity.Subject == "" || identity.Login == "" {
		http.Error(response, "GitHub login failed", http.StatusBadGateway)
		return
	}
	user, err := s.store.UserByExternalIdentity(request.Context(), "github", identity.Subject, identity.Login)
	if errors.Is(err, control.ErrForbidden) || errors.Is(err, control.ErrNotFound) {
		s.render(response, http.StatusForbidden, "Access not provisioned", h.P(g.Text("This GitHub account is not provisioned in Autback. Ask an administrator to grant access.")))
		return
	}
	if err != nil {
		http.Error(response, "login unavailable", http.StatusInternalServerError)
		return
	}
	session, err := s.store.CreateBrowserSession(request.Context(), user.ID, s.now().UTC().Add(s.sessionTTL))
	if err != nil {
		http.Error(response, "login unavailable", http.StatusInternalServerError)
		return
	}
	sessionPrincipal := control.Principal{Kind: control.PrincipalBrowser, UserID: user.ID, Admin: user.Admin}
	if err := s.store.Audit(request.Context(), sessionPrincipal, "", "auth.github.login", user.ID, map[string]string{"github_id": identity.Subject, "github_login": identity.Login}); err != nil {
		_ = s.store.RevokeBrowserSession(request.Context(), session.Token)
		http.Error(response, "login unavailable", http.StatusInternalServerError)
		return
	}
	http.SetCookie(response, &http.Cookie{
		Name: sessionCookie, Value: session.Token, Path: "/", Expires: session.ExpiresAt,
		Secure: s.publicURL.Scheme == "https", HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(response, &http.Cookie{Name: oauthStateCookie, Path: "/auth/github/callback", MaxAge: -1, Secure: s.publicURL.Scheme == "https", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(response, request, transaction.ReturnTo, http.StatusSeeOther)
}

func (s *server) devicePage(response http.ResponseWriter, request *http.Request) {
	principal, ok := s.browserPrincipal(request)
	if !ok {
		returnTo := "/auth/device?" + url.Values{"code": {request.URL.Query().Get("code")}}.Encode()
		http.Redirect(response, request, "/auth/login?"+url.Values{"return_to": {returnTo}}.Encode(), http.StatusSeeOther)
		return
	}
	login, err := s.store.DeviceLoginByUserCode(request.Context(), request.URL.Query().Get("code"), s.now().UTC())
	if err != nil {
		s.render(response, http.StatusNotFound, "Login request unavailable", h.P(g.Text("This device login request is invalid, expired, or already used.")))
		return
	}
	user, err := s.store.User(request.Context(), principal.UserID)
	if err != nil {
		http.Error(response, "login unavailable", http.StatusInternalServerError)
		return
	}
	s.render(response, http.StatusOK, "Approve device", h.Div(
		h.P(g.Textf("Signed in as %s.", user.Name)),
		h.P(g.Text("Approve this device to receive an independent, revocable Autback credential.")),
		h.Dl(h.Dt(g.Text("Device")), h.Dd(g.Text(login.DeviceName)), h.Dt(g.Text("Code")), h.Dd(h.Code(g.Text(login.UserCode)))),
		h.Form(h.Method(http.MethodPost), h.Action("/auth/device/approve"),
			h.Input(h.Type("hidden"), h.Name("code"), h.Value(login.UserCode)),
			h.Button(h.Type("submit"), g.Text("Approve device")),
		),
	))
}

func (s *server) approveDevice(response http.ResponseWriter, request *http.Request) {
	principal, ok := s.browserPrincipal(request)
	if !ok {
		http.Error(response, "authentication required", http.StatusUnauthorized)
		return
	}
	if !sameOrigin(request, s.publicURL) {
		http.Error(response, "invalid origin", http.StatusForbidden)
		return
	}
	if err := request.ParseForm(); err != nil {
		http.Error(response, "invalid approval", http.StatusBadRequest)
		return
	}
	login, err := s.store.DeviceLoginByUserCode(request.Context(), request.Form.Get("code"), s.now().UTC())
	if err != nil {
		s.render(response, http.StatusNotFound, "Login request unavailable", h.P(g.Text("This device login request is invalid, expired, or already approved.")))
		return
	}
	if err := s.store.ApproveDeviceLogin(request.Context(), login.UserCode, principal, s.now().UTC()); err != nil {
		s.render(response, http.StatusNotFound, "Login request unavailable", h.P(g.Text("This device login request is invalid, expired, or already approved.")))
		return
	}
	if err := s.store.Audit(request.Context(), principal, "", "auth.device.approve", principal.UserID, map[string]string{"device_login_id": login.ID, "device_name": login.DeviceName}); err != nil {
		http.Error(response, "approve device unavailable", http.StatusInternalServerError)
		return
	}
	s.render(response, http.StatusOK, "Device approved", h.P(g.Text("The device is approved. You can close this window and return to the terminal.")))
}

func (s *server) startDevice(response http.ResponseWriter, request *http.Request) {
	if !s.startLimit.Allow(clientAddress(request)) {
		http.Error(response, "too many login attempts", http.StatusTooManyRequests)
		return
	}
	var input deviceStartRequest
	if err := decodeJSON(response, request, &input); err != nil || strings.TrimSpace(input.DeviceName) == "" {
		http.Error(response, "valid device_name is required", http.StatusBadRequest)
		return
	}
	issued, err := s.store.CreateDeviceLogin(request.Context(), input.DeviceName, s.now().UTC().Add(s.loginTTL))
	if err != nil {
		http.Error(response, "create login request", http.StatusBadRequest)
		return
	}
	verification := s.publicURL.ResolveReference(&url.URL{Path: "/auth/device"})
	complete := *verification
	complete.RawQuery = url.Values{"code": {issued.Login.UserCode}}.Encode()
	writeJSON(response, http.StatusCreated, deviceStartResponse{
		DeviceCode: issued.DeviceCode, UserCode: issued.Login.UserCode,
		VerificationURI: verification.String(), VerificationURIComplete: complete.String(),
		ExpiresInSeconds: int64(s.loginTTL.Seconds()), IntervalSeconds: 2,
	})
}

func (s *server) exchangeDevice(response http.ResponseWriter, request *http.Request) {
	if !s.pollLimit.Allow(clientAddress(request)) {
		http.Error(response, "too many login attempts", http.StatusTooManyRequests)
		return
	}
	var input deviceTokenRequest
	if err := decodeJSON(response, request, &input); err != nil || input.DeviceCode == "" {
		http.Error(response, "valid device_code is required", http.StatusBadRequest)
		return
	}
	issued, err := s.store.ExchangeDeviceLogin(request.Context(), input.DeviceCode, s.now().UTC())
	if errors.Is(err, control.ErrLoginPending) {
		writeJSON(response, http.StatusAccepted, deviceTokenResponse{Status: "authorization_pending", IntervalSeconds: 2})
		return
	}
	if err != nil {
		writeJSON(response, http.StatusUnauthorized, deviceTokenResponse{Status: "invalid_or_expired"})
		return
	}
	principal := control.Principal{Kind: control.PrincipalDevice, TokenID: issued.Metadata.ID, UserID: issued.Metadata.UserID}
	if err := s.store.Audit(request.Context(), principal, "", "auth.device.issue", issued.Metadata.ID, map[string]string{"device_name": issued.Metadata.Name}); err != nil {
		_ = s.store.RevokeDeviceToken(request.Context(), principal, issued.Metadata.ID)
		http.Error(response, "issue device credential", http.StatusInternalServerError)
		return
	}
	writeJSON(response, http.StatusOK, deviceTokenResponse{Status: "authorized", Token: issued.Secret, TokenID: issued.Metadata.ID})
}

func (s *server) browserPrincipal(request *http.Request) (control.Principal, bool) {
	cookie, err := request.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return control.Principal{}, false
	}
	principal, err := s.store.Authenticate(request.Context(), cookie.Value)
	return principal, err == nil && principal.Kind == control.PrincipalBrowser
}

func (s *server) render(response http.ResponseWriter, status int, title string, content g.Node) {
	document := h.Doctype(g.Group{h.HTML(
		h.Lang("en"),
		h.Head(h.Meta(h.Charset("utf-8")), h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")), h.TitleEl(g.Text(title+" · Autback")), h.Link(h.Rel("stylesheet"), h.Href("/auth/assets/auth.css"))),
		h.Body(h.Main(h.A(h.Class("brand"), h.Href("https://autback.dev"), g.Text("AUTBACK")), h.Section(h.H1(g.Text(title)), content))),
	)})
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(status)
	if err := document.Render(response); err != nil {
		return
	}
}

func validReturnTo(value string) bool {
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	return parsed.Path == "/app" || strings.HasPrefix(parsed.Path, "/app/") || parsed.Path == "/auth/device"
}

func randomValue(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func pkceChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func constantEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func sameOrigin(request *http.Request, publicURL *url.URL) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Scheme == publicURL.Scheme && parsed.Host == publicURL.Host
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func decodeJSON(response http.ResponseWriter, request *http.Request, output any) error {
	request.Body = http.MaxBytesReader(response, request.Body, 4096)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(output)
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src 'self' data:; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(response, request)
	})
}

type windowLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	now      func() time.Time
	requests map[string][]time.Time
}

func newWindowLimiter(limit int, window time.Duration, now func() time.Time) *windowLimiter {
	return &windowLimiter{limit: limit, window: window, now: now, requests: make(map[string][]time.Time)}
}

func (l *windowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now().UTC()
	threshold := now.Add(-l.window)
	if _, exists := l.requests[key]; !exists && len(l.requests) >= maxLimiterClients {
		for tracked, timestamps := range l.requests {
			if len(timestamps) == 0 || !timestamps[len(timestamps)-1].After(threshold) {
				delete(l.requests, tracked)
			}
		}
		if len(l.requests) >= maxLimiterClients {
			return false
		}
	}
	values := l.requests[key]
	first := 0
	for first < len(values) && !values[first].After(threshold) {
		first++
	}
	values = append(values[first:], now)
	l.requests[key] = values
	return len(values) <= l.limit
}

func clientAddress(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return request.RemoteAddr
}

func (s deviceStartResponse) String() string {
	return fmt.Sprintf("%s (%s)", s.VerificationURI, s.UserCode)
}
