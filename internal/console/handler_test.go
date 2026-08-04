package console

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
)

func TestConsoleRequiresAnAuthenticatedDevice(t *testing.T) {
	handler, err := New(Config{Source: &fakeSource{}})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/app", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestPublicConsoleRedirectsToLoginAndAcceptsABrowserSession(t *testing.T) {
	source := &fakeSource{snapshot: exampleSnapshot()}
	handler, err := New(Config{Source: source, LoginURL: "/auth/login", SessionCookieName: "autback_session"})
	if err != nil {
		t.Fatal(err)
	}
	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/app/projects/example", nil))
	if unauthenticated.Code != http.StatusSeeOther || unauthenticated.Header().Get("Location") != "/auth/login?return_to=%2Fapp%2Fprojects%2Fexample" {
		t.Fatalf("status=%d location=%q", unauthenticated.Code, unauthenticated.Header().Get("Location"))
	}

	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	request.AddCookie(&http.Cookie{Name: "autback_session", Value: "browser-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `human-auth="true"`) {
		t.Fatalf("public console did not expose its sign-out mode: %s", response.Body.String())
	}
}

func TestConsoleDocumentIsAStreamFirstReadOnlyShell(t *testing.T) {
	source := &fakeSource{snapshot: exampleSnapshot()}
	handler, err := New(Config{Source: source})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`data-init="@get(&#39;/app/updates?route=overview&#39;, {openWhenHidden: true})"`,
		`<autback-console`,
		`route-kind="overview"`,
		`href="/app/assets/document.css"`,
		`/app/assets/datastar.js`,
		`/app/assets/console.js`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("document missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Example Service") || strings.Contains(body, "job_example") {
		t.Fatalf("document serialized read-model state instead of hydrating over /updates:\n%s", body)
	}
	if strings.Contains(body, "<form") || strings.Contains(body, "<button") {
		t.Fatalf("read-only shell contains a mutation affordance:\n%s", body)
	}
	if csp := response.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "script-src 'self' 'unsafe-eval'") {
		t.Fatalf("CSP=%q", csp)
	}
}

func TestConsoleDocumentStylesRemoveBrowserPageMargins(t *testing.T) {
	handler, err := New(Config{Source: &fakeSource{}})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/app/assets/document.css", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "margin: 0") {
		t.Fatalf("document stylesheet does not reset the browser margin:\n%s", response.Body.String())
	}
}

func TestUpdatesHydrateDedicatedSignalRoots(t *testing.T) {
	updates := make(chan struct{})
	close(updates)
	source := &fakeSource{snapshot: exampleSnapshot(), updates: updates}
	handler, err := New(Config{Source: source})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/app/updates?route=overview", nil)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, root := range []string{`"session"`, `"service"`, `"worker"`, `"resources"`, `"queue"`, `"operations"`, `"operation"`, `"log"`, `"audit"`, `"status"`, `"clock"`} {
		if !strings.Contains(body, root) {
			t.Fatalf("signal stream missing %s:\n%s", root, body)
		}
	}
	for _, value := range []string{"Example Service", "job_example", "CLI only"} {
		if !strings.Contains(body, value) {
			t.Fatalf("signal stream missing %q:\n%s", value, body)
		}
	}
}

func TestUpdatesStreamBackendClockWithoutControlChanges(t *testing.T) {
	updates := make(chan struct{})
	source := &fakeSource{snapshot: exampleSnapshot(), updates: updates}
	var tick atomic.Int64
	base := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)
	handler, err := New(Config{
		Source: source, ClockInterval: time.Millisecond,
		Now: func() time.Time { return base.Add(time.Duration(tick.Add(1)) * time.Second) },
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/app/updates?route=overview", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if count := strings.Count(response.Body.String(), `"clock"`); count < 2 {
		t.Fatalf("clock patches=%d; body=%q", count, response.Body.String())
	}
}

func TestUpdatesStreamLiveJobLogTailWithoutControlChanges(t *testing.T) {
	changes := make(chan struct{})
	logs := make(chan LogView, 1)
	logs <- LogView{Available: true, Content: "initial-output\nlive-output\n"}
	close(logs)
	snapshot := exampleSnapshot()
	snapshot.Log = LogView{Available: true, Content: "initial-output\n"}
	source := &fakeSource{snapshot: snapshot, updates: changes, logs: logs}
	handler, err := New(Config{Source: source})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Millisecond)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/app/updates?route=operation&kind=job&id=job_example", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if body := response.Body.String(); !strings.Contains(body, "live-output") {
		t.Fatalf("stream did not publish the live log tail:\n%s", body)
	}
}

func TestControlRefreshDoesNotOverwriteLiveJobLogTail(t *testing.T) {
	changes := make(chan struct{}, 1)
	logs := make(chan LogView, 1)
	first := exampleSnapshot()
	first.Revision = 42
	second := exampleSnapshot()
	second.Revision = 43
	second.Status.Message = "State refreshed"
	source := &fakeSource{snapshots: []Snapshot{first, second}, updates: changes, logs: logs}
	handler, err := New(Config{Source: source, ClockInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		logs <- LogView{Available: true, Content: "live-output\n"}
		close(logs)
		time.Sleep(2 * time.Millisecond)
		changes <- struct{}{}
		close(changes)
	}()
	request := httptest.NewRequest(http.MethodGet, "/app/updates?route=operation&kind=job&id=job_example", nil)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	live := strings.Index(body, "live-output")
	if live < 0 || !strings.Contains(body[live:], "State refreshed") {
		t.Fatalf("stream did not publish log before refreshed state:\n%s", body)
	}
	if strings.Contains(body[live+len("live-output"):], `"log"`) {
		t.Fatalf("state refresh overwrote the live log signal:\n%s", body)
	}
}

func TestUpdatesRequeryAfterACommittedChangeNotification(t *testing.T) {
	updates := make(chan struct{}, 1)
	updates <- struct{}{}
	close(updates)
	first := exampleSnapshot()
	first.Revision = 42
	first.Status.Message = "Initial state"
	second := exampleSnapshot()
	second.Revision = 43
	second.Status.Message = "Queue changed"
	second.Queue = append(second.Queue, QueueView{Position: 2, Kind: "build", ID: "bld_next", Project: "example", Status: "queued", AcceptedAt: time.Now().UTC()})
	source := &fakeSource{snapshot: first, snapshots: []Snapshot{first, second}, updates: updates}
	handler, err := New(Config{Source: source})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/app/updates?route=overview", nil)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	for _, want := range []string{"Initial state", "Queue changed", "bld_next"} {
		if !strings.Contains(body, want) {
			t.Fatalf("stream did not publish %q after notification:\n%s", want, body)
		}
	}
}

func TestCanonicalRoutesSelectTheirOwnReadModel(t *testing.T) {
	source := &fakeSource{snapshot: exampleSnapshot()}
	handler, err := New(Config{Source: source})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path, kind, project, operationKind, operationID string
	}{
		{"/app", "overview", "", "", ""},
		{"/app/projects/example", "project", "example", "", ""},
		{"/app/runs/job/job_example", "operation", "", "job", "job_example"},
		{"/app/audit", "audit", "", "", ""},
	} {
		t.Run(test.kind, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Authorization", "Bearer device-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
			if source.lastRoute != (Route{Kind: RouteKind(test.kind), Project: test.project, OperationKind: test.operationKind, OperationID: test.operationID}) {
				t.Fatalf("route=%#v", source.lastRoute)
			}
		})
	}
}

func TestLegacyOperationRouteIsNotExposed(t *testing.T) {
	handler, err := New(Config{Source: &fakeSource{snapshot: exampleSnapshot()}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/app/operations/job/job_example", nil)
	request.Header.Set("Authorization", "Bearer device-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

type fakeSource struct {
	snapshot  Snapshot
	snapshots []Snapshot
	updates   <-chan struct{}
	logs      <-chan LogView
	lastRoute Route
}

func (f *fakeSource) Authenticate(_ context.Context, token string) (control.Principal, error) {
	if token == "browser-token" {
		return control.Principal{Kind: control.PrincipalBrowser, UserID: "usr_owner", Admin: true}, nil
	}
	if token != "device-token" {
		return control.Principal{}, control.ErrUnauthenticated
	}
	return control.Principal{Kind: control.PrincipalDevice, UserID: "usr_owner", Admin: true}, nil
}

func (f *fakeSource) Authorize(_ context.Context, _ control.Principal, route Route) error {
	f.lastRoute = route
	return nil
}

func (f *fakeSource) Snapshot(_ context.Context, _ control.Principal, route Route) (Snapshot, error) {
	f.lastRoute = route
	if len(f.snapshots) > 0 {
		snapshot := f.snapshots[0]
		f.snapshots = f.snapshots[1:]
		return snapshot, nil
	}
	return f.snapshot, nil
}

func (f *fakeSource) SubscribeChanges() (<-chan struct{}, func()) {
	if f.updates != nil {
		return f.updates, func() {}
	}
	updates := make(chan struct{})
	close(updates)
	return updates, func() {}
}

func (f *fakeSource) SubscribeLog(context.Context, control.Principal, Route) (<-chan LogView, error) {
	if f.logs != nil {
		return f.logs, nil
	}
	logs := make(chan LogView)
	close(logs)
	return logs, nil
}

func exampleSnapshot() Snapshot {
	now := time.Date(2026, time.August, 2, 18, 0, 0, 0, time.UTC)
	return Snapshot{
		Revision:   42,
		Session:    SessionView{User: "Jacob", Admin: true, Projects: []ProjectView{{ID: "prj_example", Slug: "example", Name: "Example Service"}}},
		Service:    ServiceView{Name: "Autback", Version: "0.1.0", Control: "CLI only", Admission: "One at a time"},
		Worker:     WorkerView{Status: "online", Capacity: "1 operation", UpdatedAt: now},
		Queue:      []QueueView{{Position: 1, Kind: "job", ID: "job_example", Project: "example", Status: "active", AcceptedAt: now}},
		Operations: []OperationView{{Kind: "job", ID: "job_example", Project: "example", ProjectName: "Example Service", Status: "running", Command: "task ci", CreatedAt: now}},
		Audit:      []AuditView{{ID: 1, Actor: "Jacob", Action: "job.start", Target: "job_example", CreatedAt: now}},
		Status:     StatusView{Ready: true, Route: "overview", Message: "Live", UpdatedAt: now},
	}
}
