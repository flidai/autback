package console

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Yacobolo/toolbelt/pagestream"
	"github.com/flidai/autback/internal/control"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

//go:embed assets/*
var assets embed.FS

type Source interface {
	Authenticate(context.Context, string) (control.Principal, error)
	Authorize(context.Context, control.Principal, Route) error
	Snapshot(context.Context, control.Principal, Route) (Snapshot, error)
	SubscribeChanges() (<-chan struct{}, func())
	SubscribeLog(context.Context, control.Principal, Route) (<-chan LogView, error)
}

type Config struct {
	Source            Source
	ClockInterval     time.Duration
	Now               func() time.Time
	LoginURL          string
	SessionCookieName string
}

func New(config Config) (http.Handler, error) {
	if config.Source == nil {
		return nil, errors.New("console source is required")
	}
	if config.ClockInterval <= 0 {
		config.ClockInterval = time.Second
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	assetFS, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, err
	}
	if config.SessionCookieName == "" {
		config.SessionCookieName = "autback_session"
	}
	server := &server{source: config.Source, clockInterval: config.ClockInterval, now: config.Now, loginURL: config.LoginURL, sessionCookieName: config.SessionCookieName}
	mux := http.NewServeMux()
	mux.Handle("GET /app/assets/", http.StripPrefix("/app/assets/", http.FileServer(http.FS(assetFS))))
	mux.HandleFunc("GET /app", server.page(Route{Kind: RouteOverview}))
	mux.HandleFunc("GET /app/projects/{project}", func(response http.ResponseWriter, request *http.Request) {
		server.renderPage(response, request, Route{Kind: RouteProject, Project: request.PathValue("project")})
	})
	mux.HandleFunc("GET /app/runs/{kind}/{id}", func(response http.ResponseWriter, request *http.Request) {
		server.renderPage(response, request, Route{Kind: RouteOperation, OperationKind: request.PathValue("kind"), OperationID: request.PathValue("id")})
	})
	mux.HandleFunc("GET /app/audit", server.page(Route{Kind: RouteAudit}))
	mux.HandleFunc("GET /app/updates", server.updates)
	return securityHeaders(mux), nil
}

type server struct {
	source            Source
	clockInterval     time.Duration
	now               func() time.Time
	loginURL          string
	sessionCookieName string
}

func (s *server) page(route Route) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) { s.renderPage(response, request, route) }
}

func (s *server) renderPage(response http.ResponseWriter, request *http.Request, route Route) {
	principal, ok := s.authenticate(response, request)
	if !ok {
		return
	}
	if err := validateRoute(route); err != nil {
		http.NotFound(response, request)
		return
	}
	// Resolve once before rendering so canonical documents never mount a route
	// the principal cannot read. The actual state still hydrates over /updates.
	if err := s.source.Authorize(request.Context(), principal, route); err != nil {
		writeSourceError(response, err)
		return
	}
	query := url.Values{"route": []string{string(route.Kind)}}
	if route.Project != "" {
		query.Set("project", route.Project)
	}
	if route.OperationKind != "" {
		query.Set("kind", route.OperationKind)
		query.Set("id", route.OperationID)
	}
	page := pagestream.RenderPage(pagestream.PageSpec{
		Title:             pageTitle(route),
		DatastarScriptURL: "/app/assets/datastar.js",
		UpdatesURL:        "/app/updates?" + query.Encode(),
		Head: []g.Node{
			h.Meta(h.Name("theme-color"), h.Content("#0d0d0f")),
			h.Link(h.Rel("icon"), h.Href("/app/assets/favicon.svg")),
			h.Link(h.Rel("stylesheet"), h.Href("/app/assets/document.css")),
			h.Script(h.Type("module"), h.Src("/app/assets/console.js")),
		},
		Body: []g.Node{g.El("autback-console",
			g.Attr("route-kind", string(route.Kind)),
			g.Attr("project", route.Project),
			g.Attr("operation-kind", route.OperationKind),
			g.Attr("operation-id", route.OperationID),
			g.Attr("human-auth", strconv.FormatBool(s.loginURL != "")),
		)},
	})
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(response); err != nil {
		http.Error(response, "render console", http.StatusInternalServerError)
	}
}

func (s *server) updates(response http.ResponseWriter, request *http.Request) {
	principal, ok := s.authenticate(response, request)
	if !ok {
		return
	}
	route := Route{
		Kind: RouteKind(request.URL.Query().Get("route")), Project: request.URL.Query().Get("project"),
		OperationKind: request.URL.Query().Get("kind"), OperationID: request.URL.Query().Get("id"),
	}
	if err := validateRoute(route); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	updates, unsubscribe := s.source.SubscribeChanges()
	defer unsubscribe()
	stream := pagestream.NewSignalStream(response, request)
	snapshot, err := s.source.Snapshot(request.Context(), principal, route)
	if err != nil {
		writeSourceError(response, err)
		return
	}
	snapshot.Clock = ClockView{Now: s.now().UTC()}
	if err := stream.Patch(snapshot.patch()); err != nil {
		return
	}
	logUpdates, err := s.source.SubscribeLog(request.Context(), principal, route)
	if err != nil {
		return
	}
	revision := snapshot.Revision
	clock := time.NewTicker(s.clockInterval)
	defer clock.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-clock.C:
			if err := stream.Patch(map[string]any{"clock": ClockView{Now: s.now().UTC()}}); err != nil {
				return
			}
		case log, open := <-logUpdates:
			if !open {
				logUpdates = nil
				continue
			}
			if err := stream.Patch(map[string]any{"log": log}); err != nil {
				return
			}
		case _, open := <-updates:
			if !open {
				return
			}
			next, err := s.source.Snapshot(request.Context(), principal, route)
			if err != nil {
				return
			}
			if next.Revision <= revision {
				continue
			}
			next.Clock = ClockView{Now: s.now().UTC()}
			if err := stream.Patch(next.refreshPatch()); err != nil {
				return
			}
			revision = next.Revision
		}
	}
}

func (s *server) authenticate(response http.ResponseWriter, request *http.Request) (control.Principal, bool) {
	value := request.Header.Get("Authorization")
	token, ok := strings.CutPrefix(value, "Bearer ")
	if !ok || token == "" || strings.ContainsAny(token, " \t\r\n") {
		if cookie, err := request.Cookie(s.sessionCookieName); err == nil {
			token = cookie.Value
		}
	}
	principal, err := s.source.Authenticate(request.Context(), token)
	if err != nil || principal.Kind != control.PrincipalDevice && principal.Kind != control.PrincipalBrowser {
		if s.loginURL != "" && request.Method == http.MethodGet {
			target := s.loginURL + "?" + url.Values{"return_to": {request.URL.RequestURI()}}.Encode()
			http.Redirect(response, request, target, http.StatusSeeOther)
			return control.Principal{}, false
		}
		response.Header().Set("WWW-Authenticate", `Bearer realm="autback-console"`)
		http.Error(response, "authentication required; open the console through the autback CLI", http.StatusUnauthorized)
		return control.Principal{}, false
	}
	return principal, true
}

func validateRoute(route Route) error {
	switch route.Kind {
	case RouteOverview, RouteAudit:
		if route.Project == "" && route.OperationKind == "" && route.OperationID == "" {
			return nil
		}
	case RouteProject:
		if route.Project != "" && route.OperationKind == "" && route.OperationID == "" {
			return nil
		}
	case RouteOperation:
		if (route.OperationKind == "job" || route.OperationKind == "build") && route.OperationID != "" && route.Project == "" {
			return nil
		}
	}
	return fmt.Errorf("invalid console route")
}

func pageTitle(route Route) string {
	switch route.Kind {
	case RouteProject:
		return route.Project + " · Autback"
	case RouteOperation:
		return route.OperationID + " · Autback"
	case RouteAudit:
		return "Audit · Autback"
	default:
		return "Autback Console"
	}
}

func writeSourceError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, control.ErrUnauthenticated):
		http.Error(response, "unauthenticated", http.StatusUnauthorized)
	case errors.Is(err, control.ErrForbidden):
		http.Error(response, "forbidden", http.StatusForbidden)
	case errors.Is(err, control.ErrNotFound):
		http.Error(response, "not found", http.StatusNotFound)
	default:
		http.Error(response, "console unavailable", http.StatusInternalServerError)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		// Datastar compiles declarative data-* expressions with Function, so its
		// self-hosted runtime requires unsafe-eval. Scripts remain same-origin and
		// the console exposes no write endpoints or form actions.
		response.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self'; connect-src 'self'; img-src 'self' data:; font-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(response, request)
	})
}
