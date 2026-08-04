package cli

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/flidai/autback/internal/authclient"
	"github.com/flidai/autback/internal/config"
	"github.com/flidai/autback/internal/control/controlclient"
)

const consoleSessionCookie = "autback_console_session"

func serviceConsole(ctx context.Context, settings config.Config, explicitToken string, args []string, streams IO) int {
	if len(args) != 0 {
		return failUsage(streams.Stderr, "console accepts no arguments")
	}
	token, _, err := authclient.Resolve(ctx, authclient.ResolveOptions{
		ExplicitToken: explicitToken, ServiceURL: settings.URL, Keyring: streams.Keyring,
	})
	if err != nil {
		return fail(streams.Stderr, err)
	}
	client, target, err := controlclient.NewHTTPClient(settings.URL, "", settings.Service.CACertFile)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	session, err := randomConsoleSession()
	if err != nil {
		return fail(streams.Stderr, err)
	}
	handler, err := newConsoleProxy(target, token, session, client.Transport)
	if err != nil {
		return fail(streams.Stderr, err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fail(streams.Stderr, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 2 * time.Minute}
	openURL := "http://" + listener.Addr().String() + "/session/" + session
	fmt.Fprintln(streams.Stdout, openURL)
	if err := streams.OpenURL(openURL); err != nil {
		fmt.Fprintln(streams.Stderr, "Open the console URL in a browser; automatic opening failed:", err)
	}
	errorsChannel := make(chan error, 1)
	go func() { errorsChannel <- server.Serve(listener) }()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return fail(streams.Stderr, err)
		}
		return 0
	case err := <-errorsChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return 0
		}
		return fail(streams.Stderr, err)
	}
}

func newConsoleProxy(target *url.URL, deviceToken, session string, transport http.RoundTripper) (http.Handler, error) {
	if target == nil || target.Host == "" || target.Scheme != "http" && target.Scheme != "https" {
		return nil, errors.New("console target must be an absolute HTTP(S) URL")
	}
	if deviceToken == "" || session == "" {
		return nil, errors.New("console device token and local session are required")
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	direct := proxy.Director
	proxy.Director = func(request *http.Request) {
		direct(request)
		request.Host = target.Host
		request.Header.Set("Authorization", "Bearer "+deviceToken)
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/session/") {
			candidate := strings.TrimPrefix(request.URL.Path, "/session/")
			if candidate == "" || strings.Contains(candidate, "/") || !constantEqual(candidate, session) {
				http.NotFound(response, request)
				return
			}
			http.SetCookie(response, &http.Cookie{
				Name: consoleSessionCookie, Value: session, Path: "/", HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			response.Header().Set("Cache-Control", "no-store")
			http.Redirect(response, request, "/app", http.StatusSeeOther)
			return
		}
		if request.Method != http.MethodGet && request.Method != http.MethodHead || request.URL.Path != "/app" && !strings.HasPrefix(request.URL.Path, "/app/") {
			http.NotFound(response, request)
			return
		}
		cookie, err := request.Cookie(consoleSessionCookie)
		if err != nil || !constantEqual(cookie.Value, session) {
			http.Error(response, "console session required", http.StatusUnauthorized)
			return
		}
		proxy.ServeHTTP(response, request)
	}), nil
}

func constantEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func randomConsoleSession() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func openBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}
