// Command autback-console-preview serves the production console against a
// read-only fixture. It is a local design-system and browser-QA harness; it
// never opens the control database or exposes control-plane commands.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/flidai/autback/internal/console"
	"github.com/flidai/autback/internal/control"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:4173", "loopback preview address")
	flag.Parse()
	host, _, err := net.SplitHostPort(*listen)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		log.Fatal("preview must listen on an explicit loopback IP and port")
	}
	source := newFixtureSource()
	handler, err := console.New(console.Config{Source: source})
	if err != nil {
		log.Fatal(err)
	}
	preview := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		clone := request.Clone(request.Context())
		clone.Header = request.Header.Clone()
		clone.Header.Set("Authorization", "Bearer local-preview")
		handler.ServeHTTP(response, clone)
	})
	log.Printf("Autback console preview: http://%s/app", *listen)
	server := &http.Server{Addr: *listen, Handler: preview, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}

type fixtureSource struct {
	revision atomic.Int64
	updates  chan struct{}
	now      time.Time
	base     console.Snapshot
}

func newFixtureSource() *fixtureSource {
	now := time.Now().UTC()
	projects := []console.ProjectView{
		{ID: "prj_autback", Slug: "autback", Name: "Autback", ActiveImage: "ghcr.io/flidai/autback-runner@sha256:ddbf482db839f762ef0df044f1ea26ee90637a49e99fcd34aee06f28a9bd239b", Members: 3, Trusts: 2},
		{ID: "prj_leapview", Slug: "leapview", Name: "LeapView", ActiveImage: "ghcr.io/flidai/leapview-ci@sha256:8ec66114a7dc17a4869c6e7dc7b1df40d706143fb28d17fe4bc72d3c7d36b4ee", Members: 4, Trusts: 3},
		{ID: "prj_toolbelt", Slug: "toolbelt", Name: "Toolbelt", ActiveImage: "ghcr.io/yacobolo/toolbelt-ci@sha256:7cb74bf60fd7f13daec6a88e84a61834eb44189451bed85770bc72899ce24933", Members: 2, Trusts: 1},
	}
	operations := []console.OperationView{
		{Kind: "job", ID: "job_01K1QX7NWJ0M9C16G4G0S1VDFH", Project: "leapview", ProjectName: "LeapView", Status: "running", Command: "task ci", Image: projects[1].ActiveImage, CreatedAt: now.Add(-2 * time.Minute), StartedAt: pointer(now.Add(-95 * time.Second))},
		{Kind: "build", ID: "bld_01K1QX2K48PXB34W8S6EYVQ00R", Project: "autback", ProjectName: "Autback", Status: "queued", Command: "", CreatedAt: now.Add(-7 * time.Minute)},
		{Kind: "job", ID: "job_01K1QWJ38V21G68H58K1E9N3W4", Project: "toolbelt", ProjectName: "Toolbelt", Status: "succeeded", Command: "go test ./...", Image: projects[2].ActiveImage, CreatedAt: now.Add(-38 * time.Minute), StartedAt: pointer(now.Add(-37 * time.Minute)), FinishedAt: pointer(now.Add(-36*time.Minute - 48*time.Second)), ExitCode: intPointer(0)},
		{Kind: "build", ID: "bld_01K1QVBWQJVV03W4P2C7MEW1B6", Project: "leapview", ProjectName: "LeapView", Status: "succeeded", CreatedAt: now.Add(-2 * time.Hour), FinishedAt: pointer(now.Add(-2*time.Hour + 14*time.Second)), ExitCode: intPointer(0)},
		{Kind: "job", ID: "job_01K1QTYN9J4DGV7D0ZHRM8R84S", Project: "leapview", ProjectName: "LeapView", Status: "failed", Command: "task test", Image: projects[1].ActiveImage, CreatedAt: now.Add(-4 * time.Hour), StartedAt: pointer(now.Add(-4*time.Hour + time.Second)), FinishedAt: pointer(now.Add(-4*time.Hour + 18*time.Second)), ExitCode: intPointer(1)},
	}
	base := console.Snapshot{
		Session: console.SessionView{User: "Jacob Østergaard", Admin: true, Projects: projects},
		Service: console.ServiceView{Name: "Autback", Version: "0.1.0", Control: "CLI only", Admission: "Strict FIFO", StartedAt: now.Add(-31 * time.Hour)},
		Worker:  console.WorkerView{Status: "online", Capacity: "1 operation", ActiveID: operations[0].ID, UpdatedAt: now},
		Queue: []console.QueueView{
			{Position: 1, Kind: "job", ID: operations[0].ID, Project: "leapview", ProjectName: "LeapView", Status: "active", AcceptedAt: now.Add(-2 * time.Minute), LeasedAt: pointer(now.Add(-95 * time.Second))},
			{Position: 2, Kind: "build", ID: operations[1].ID, Project: "autback", ProjectName: "Autback", Status: "queued", AcceptedAt: now.Add(-7 * time.Minute)},
		},
		Operations: operations,
		Audit: []console.AuditView{
			{ID: 190, Actor: "Jacob Østergaard", Action: "job.start", Target: operations[0].ID, Project: "leapview", Metadata: map[string]string{"root_digest": "bd76a4d3…/12786"}, CreatedAt: now.Add(-95 * time.Second)},
			{ID: 189, Actor: "GitHub Actions", Action: "build.prepare", Target: operations[1].ID, Project: "autback", Metadata: map[string]string{"ref": "refs/pull/3/merge"}, CreatedAt: now.Add(-7 * time.Minute)},
			{ID: 188, Actor: "Jacob Østergaard", Action: "project.image.activate", Target: "prj_leapview", Project: "leapview", Metadata: map[string]string{"image": "sha256:8ec66114…"}, CreatedAt: now.Add(-21 * time.Minute)},
			{ID: 187, Actor: "GitHub Actions", Action: "github.exchange", Target: "ght_leapview_main", Project: "leapview", Metadata: map[string]string{"repository_id": "927418102"}, CreatedAt: now.Add(-39 * time.Minute)},
			{ID: 186, Actor: "Jacob Østergaard", Action: "device-token.create", Target: "dt_coworker", Metadata: map[string]string{}, CreatedAt: now.Add(-3 * time.Hour)},
		},
		Status: console.StatusView{Ready: true, Route: "overview", Message: "Live", UpdatedAt: now},
	}
	source := &fixtureSource{updates: make(chan struct{}), now: now, base: base}
	source.revision.Store(190)
	return source
}

func (s *fixtureSource) Authenticate(context.Context, string) (control.Principal, error) {
	return control.Principal{Kind: control.PrincipalDevice, UserID: "usr_owner", Admin: true}, nil
}

func (s *fixtureSource) Authorize(context.Context, control.Principal, console.Route) error {
	return nil
}

func (s *fixtureSource) SubscribeChanges() (<-chan struct{}, func()) { return s.updates, func() {} }

func (s *fixtureSource) Snapshot(_ context.Context, _ control.Principal, route console.Route) (console.Snapshot, error) {
	snapshot := s.base
	snapshot.Revision = s.revision.Load()
	snapshot.Status.Route = string(route.Kind)
	if route.Kind == console.RouteProject {
		filtered := make([]console.OperationView, 0)
		for _, operation := range snapshot.Operations {
			if operation.Project == route.Project {
				filtered = append(filtered, operation)
			}
		}
		snapshot.Operations = filtered
		queue := make([]console.QueueView, 0)
		for _, item := range snapshot.Queue {
			if item.Project == route.Project {
				queue = append(queue, item)
			}
		}
		snapshot.Queue = queue
	}
	if route.Kind == console.RouteOperation {
		for _, operation := range s.base.Operations {
			if operation.Kind == route.OperationKind && operation.ID == route.OperationID {
				snapshot.Operation = &console.OperationDetailView{
					OperationView: operation, WorkingDirectory: "/workspace", Environment: map[string]string{"CI": "true"},
					Caches:     []console.CacheView{{Name: "go-build", Target: "/root/.cache/go-build"}, {Name: "go-mod", Target: "/go/pkg/mod"}},
					RootDigest: "bd76a4d3bd9ec8f076ad844fb3b1e1b39b12acbe2c1d2df5f68761ee7e758c9d/12786",
				}
				snapshot.Log = console.LogView{Available: operation.Kind == "job", Content: previewLog, Truncated: true}
				break
			}
		}
	}
	return snapshot, nil
}

func pointer(value time.Time) *time.Time { return &value }
func intPointer(value int) *int          { return &value }

const previewLog = `:: preparing exact worktree
source  bd76a4d3bd9e/12786  0 B uploaded (cache hit)
runner  ghcr.io/flidai/leapview-ci@sha256:8ec66114…

task: [generate] go generate ./...
task: [test:go] go test ./...
ok  github.com/flidai/leapview/internal/access        (cached)
ok  github.com/flidai/leapview/internal/analytics     2.184s
ok  github.com/flidai/leapview/internal/deployment    (cached)
task: [test:web] bun test
  428 pass
  0 fail

Testcontainers: postgres:17-alpine healthy
Testcontainers: redis:7-alpine healthy
`
