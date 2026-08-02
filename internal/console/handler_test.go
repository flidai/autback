package console

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	for _, root := range []string{`"session"`, `"service"`, `"worker"`, `"queue"`, `"operations"`, `"operation"`, `"log"`, `"audit"`, `"status"`} {
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
		{"/app/operations/job/job_example", "operation", "", "job", "job_example"},
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

type fakeSource struct {
	snapshot  Snapshot
	snapshots []Snapshot
	updates   <-chan struct{}
	lastRoute Route
}

func (f *fakeSource) Authenticate(_ context.Context, token string) (control.Principal, error) {
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

func exampleSnapshot() Snapshot {
	now := time.Date(2026, time.August, 2, 18, 0, 0, 0, time.UTC)
	return Snapshot{
		Revision:   42,
		Session:    SessionView{User: "Jacob", Admin: true, Projects: []ProjectView{{ID: "prj_example", Slug: "example", Name: "Example Service"}}},
		Service:    ServiceView{Name: "Autback", Version: "0.1.0", Control: "CLI only", Admission: "Strict FIFO"},
		Worker:     WorkerView{Status: "online", Capacity: "1 operation", UpdatedAt: now},
		Queue:      []QueueView{{Position: 1, Kind: "job", ID: "job_example", Project: "example", Status: "active", AcceptedAt: now}},
		Operations: []OperationView{{Kind: "job", ID: "job_example", Project: "example", ProjectName: "Example Service", Status: "running", Command: "task ci", CreatedAt: now}},
		Audit:      []AuditView{{ID: 1, Actor: "Jacob", Action: "job.start", Target: "job_example", CreatedAt: now}},
		Status:     StatusView{Ready: true, Route: "overview", Message: "Live from SQLite", UpdatedAt: now},
	}
}
