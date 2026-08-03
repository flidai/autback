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
	"sync"
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
	interval time.Duration
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
		{Kind: "job", ID: "job_01K1QX7NWJ0M9C16G4G0S1VDFH", Project: "leapview", ProjectName: "LeapView", Status: "running", Command: "task ci", Image: projects[1].ActiveImage, CreatedAt: now.Add(-2 * time.Minute), StartedAt: pointer(now.Add(-95 * time.Second)), QueueWaitMillis: int64Pointer(25000), Resources: resourceSummary(44, .38, .86, .51, .68, 5.4)},
		{Kind: "build", ID: "bld_01K1QX2K48PXB34W8S6EYVQ00R", Project: "autback", ProjectName: "Autback", Status: "queued", Command: "", CreatedAt: now.Add(-7 * time.Minute)},
		{Kind: "job", ID: "job_01K1QWJ38V21G68H58K1E9N3W4", Project: "toolbelt", ProjectName: "Toolbelt", Status: "succeeded", Command: "go test ./...", Image: projects[2].ActiveImage, CreatedAt: now.Add(-38 * time.Minute), StartedAt: pointer(now.Add(-37 * time.Minute)), FinishedAt: pointer(now.Add(-36*time.Minute - 48*time.Second)), ExitCode: intPointer(0), QueueWaitMillis: int64Pointer(60000), Resources: resourceSummary(6, .64, .91, .42, .55, 4.4)},
		{Kind: "build", ID: "bld_01K1QVBWQJVV03W4P2C7MEW1B6", Project: "leapview", ProjectName: "LeapView", Status: "succeeded", CreatedAt: now.Add(-2 * time.Hour), FinishedAt: pointer(now.Add(-2*time.Hour + 14*time.Second)), ExitCode: intPointer(0), QueueWaitMillis: int64Pointer(12000), Resources: resourceSummary(7, .55, .78, .36, .48, 3.8)},
		{Kind: "job", ID: "job_01K1QTYN9J4DGV7D0ZHRM8R84S", Project: "leapview", ProjectName: "LeapView", Status: "failed", Command: "task test", Image: projects[1].ActiveImage, CreatedAt: now.Add(-4 * time.Hour), StartedAt: pointer(now.Add(-4*time.Hour + time.Second)), FinishedAt: pointer(now.Add(-4*time.Hour + 18*time.Second)), ExitCode: intPointer(1), QueueWaitMillis: int64Pointer(1000), Resources: resourceSummary(8, .71, .96, .58, .72, 5.8)},
	}
	base := console.Snapshot{
		Session:   console.SessionView{User: "Jacob Østergaard", Admin: true, Projects: projects},
		Service:   console.ServiceView{Name: "Autback", Version: "0.1.0", Control: "CLI only", Admission: "One at a time", StartedAt: now.Add(-31 * time.Hour)},
		Worker:    console.WorkerView{Status: "online", Capacity: "1 operation", ActiveID: operations[0].ID, UpdatedAt: now},
		Resources: fixtureResources(now, 190),
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
	source := &fixtureSource{base: base, interval: 2 * time.Second}
	source.revision.Store(190)
	return source
}

func (s *fixtureSource) Authenticate(context.Context, string) (control.Principal, error) {
	return control.Principal{Kind: control.PrincipalDevice, UserID: "usr_owner", Admin: true}, nil
}

func (s *fixtureSource) Authorize(context.Context, control.Principal, console.Route) error {
	return nil
}

func (s *fixtureSource) SubscribeChanges() (<-chan struct{}, func()) {
	updates := make(chan struct{}, 1)
	done := make(chan struct{})
	var stop sync.Once
	go func() {
		interval := s.interval
		if interval <= 0 {
			interval = 2 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(updates)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				s.revision.Add(1)
				select {
				case updates <- struct{}{}:
				default:
				}
			}
		}
	}()
	return updates, func() { stop.Do(func() { close(done) }) }
}

func (s *fixtureSource) Snapshot(_ context.Context, _ control.Principal, route console.Route) (console.Snapshot, error) {
	now := time.Now().UTC()
	snapshot := s.base
	revision := s.revision.Load()
	snapshot.Revision = revision
	snapshot.Status.Route = string(route.Kind)
	snapshot.Status.UpdatedAt = now
	snapshot.Worker.UpdatedAt = now
	snapshot.Resources = fixtureResources(now, int(revision))
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
		if route.Project != "leapview" {
			snapshot.Resources.Samples = []console.ResourceSampleView{}
			snapshot.Resources.SampleCount = 0
			snapshot.Resources.ActiveSampleCount = 0
		}
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
				snapshot.Resources.Samples = operationSamples(now, int(revision))
				snapshot.Resources.SampleCount = len(snapshot.Resources.Samples)
				snapshot.Resources.ActiveSampleCount = len(snapshot.Resources.Samples)
				snapshot.Resources.BusyRatio = 1
				snapshot.Resources.CPUAverage = operation.Resources.CPUAverage
				snapshot.Resources.CPUPeak = operation.Resources.CPUPeak
				snapshot.Resources.MemoryAverage = operation.Resources.MemoryAverage
				snapshot.Resources.MemoryPeak = operation.Resources.MemoryPeak
				snapshot.Resources.MemoryBytesPeak = operation.Resources.MemoryBytesPeak
				break
			}
		}
	}
	return snapshot, nil
}

func pointer(value time.Time) *time.Time { return &value }
func intPointer(value int) *int          { return &value }
func int64Pointer(value int64) *int64    { return &value }

func resourceSummary(samples int, cpuAverage, cpuPeak, memoryAverage, memoryPeak, memoryGB float64) console.OperationResourceView {
	return console.OperationResourceView{SampleCount: samples, CPUAverage: cpuAverage, CPUPeak: cpuPeak, MemoryAverage: memoryAverage, MemoryPeak: memoryPeak, MemoryBytesPeak: uint64(memoryGB * float64(uint64(1)<<30))}
}

func fixtureResources(now time.Time, offset int) console.ResourceView {
	samples := make([]console.ResourceSampleView, 0, 120)
	active := 0
	for index := 0; index < 120; index++ {
		phase := (index + offset) % 40
		cpu, memory := .03, .21
		if phase >= 12 && phase <= 30 {
			cpu = .24 + float64((phase*17)%55)/100
			memory = .36 + float64((phase*7)%24)/100
			active++
		}
		samples = append(samples, console.ResourceSampleView{ObservedAt: now.Add(time.Duration(index-119) * 30 * time.Second), CPUUtilization: cpu, MemoryUtilization: memory})
	}
	return console.ResourceView{Samples: samples, SampleCount: len(samples), ActiveSampleCount: active, CPUCores: 4, MemoryTotalBytes: 8 << 30,
		DiskUsageBytes: 53 << 30, DiskTotalBytes: 160 << 30, BusyRatio: float64(active) / float64(len(samples)), CPUAverage: .48, CPUPeak: .86,
		MemoryAverage: .49, MemoryPeak: .68, MemoryBytesPeak: uint64(54) * (1 << 30) / 10, QueueWaitP95Millis: 60000}
}

func operationSamples(now time.Time, offset int) []console.ResourceSampleView {
	samples := make([]console.ResourceSampleView, 0, 48)
	for index := 0; index < 48; index++ {
		samples = append(samples, console.ResourceSampleView{ObservedAt: now.Add(time.Duration(index-47) * 2 * time.Second), CPUUtilization: .18 + float64(((index+offset)*13)%68)/100,
			MemoryUtilization: .38 + float64((index+offset)%15)/50})
	}
	return samples
}

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
