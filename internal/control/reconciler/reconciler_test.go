package reconciler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flidai/outback/internal/control"
	"github.com/flidai/outback/internal/protocol"
)

func TestRunOnceConvergesTerminalOrphanAndMissingJobs(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	finished := now.Add(-time.Hour)
	store := &fakeStore{jobs: map[string]control.Job{
		"job-finished": {ID: "job-finished", ProjectID: "project-one", Status: protocol.StatusRunning},
		"job-live":     {ID: "job-live", ProjectID: "project-two", Status: protocol.StatusRunning},
		"job-missing":  {ID: "job-missing", ProjectID: "project-one", Status: protocol.StatusRunning},
	}}
	scheduler := &fakeScheduler{managed: []protocol.Job{
		{ID: "job-finished", Status: protocol.StatusSucceeded, FinishedAt: &finished, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "job-live", Status: protocol.StatusRunning, CreatedAt: now.Add(-time.Minute)},
		{ID: "orphan", Status: protocol.StatusRunning, CreatedAt: now.Add(-2 * time.Hour)},
	}}
	dispatcher := &fakeDispatcher{}

	reconciler := New(Config{Store: store, Scheduler: scheduler, Dispatcher: dispatcher, ServiceRetention: 30 * time.Minute, Now: func() time.Time { return now }})
	if err := reconciler.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if got := store.jobs["job-finished"].Status; got != protocol.StatusSucceeded {
		t.Fatalf("finished status = %s", got)
	}
	if got := store.jobs["job-live"].Status; got != protocol.StatusRunning {
		t.Fatalf("live status = %s", got)
	}
	if got := store.jobs["job-missing"].Status; got != protocol.StatusLost {
		t.Fatalf("missing status = %s", got)
	}
	if len(scheduler.removed) != 2 || scheduler.removed[0] != "job-finished" || scheduler.removed[1] != "orphan" {
		t.Fatalf("removed = %#v", scheduler.removed)
	}
	released := map[string]bool{}
	for _, operation := range dispatcher.released {
		released[operation.ID] = operation.Kind == control.OperationJob
	}
	if len(dispatcher.released) != 2 || !released["job-finished"] || !released["job-missing"] {
		t.Fatalf("released = %#v", dispatcher.released)
	}
}

func TestRunOnceCancelsExpiredBuildLeaseAndAdvancesQueue(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{builds: map[string]control.Build{
		"bld-stale": {ID: "bld-stale", Status: control.BuildRunning},
	}}
	dispatcher := &fakeDispatcher{}
	reconciler := New(Config{
		Store: store, Scheduler: &fakeScheduler{}, Dispatcher: dispatcher,
		BuildLeaseTimeout: time.Hour, Now: func() time.Time { return now },
	})
	if err := reconciler.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := store.builds["bld-stale"].Status; got != control.BuildCancelled {
		t.Fatalf("build status = %s, want cancelled", got)
	}
	if len(dispatcher.released) != 1 || dispatcher.released[0].Kind != control.OperationBuild || dispatcher.released[0].ID != "bld-stale" {
		t.Fatalf("released = %#v", dispatcher.released)
	}
}

func TestRunOnceDoesNotLoseJobWhileSchedulerIsCreatingService(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{jobs: map[string]control.Job{
		"job-admitting": {ID: "job-admitting", Status: protocol.StatusQueued},
	}, leasedAt: map[string]time.Time{"job-admitting": now.Add(-time.Second)}}
	reconciler := New(Config{
		Store: store, Scheduler: &fakeScheduler{}, Dispatcher: &fakeDispatcher{},
		AdmissionGrace: 15 * time.Second, Now: func() time.Time { return now },
	})
	if err := reconciler.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := store.jobs["job-admitting"].Status; got != protocol.StatusQueued {
		t.Fatalf("job status = %s, want queued during admission grace", got)
	}
}

type fakeDispatcher struct{ released []control.Operation }

func (f *fakeDispatcher) Release(_ context.Context, kind control.OperationKind, id string) error {
	f.released = append(f.released, control.Operation{Kind: kind, ID: id})
	return nil
}

type fakeStore struct {
	jobs     map[string]control.Job
	builds   map[string]control.Build
	leasedAt map[string]time.Time
}

func (f *fakeStore) StaleBuilds(context.Context, time.Time) ([]control.Build, error) {
	var builds []control.Build
	for _, build := range f.builds {
		if build.Status == control.BuildRunning {
			builds = append(builds, build)
		}
	}
	return builds, nil
}

func (f *fakeStore) FinishBuild(_ context.Context, id string, status control.BuildStatus, exitCode int) (control.Build, error) {
	build, ok := f.builds[id]
	if !ok {
		return control.Build{}, control.ErrNotFound
	}
	build.Status, build.ExitCode = status, &exitCode
	f.builds[id] = build
	return build, nil
}

func (f *fakeStore) Operation(_ context.Context, kind control.OperationKind, id string) (control.Operation, error) {
	if kind != control.OperationJob {
		return control.Operation{}, control.ErrNotFound
	}
	leased := f.leasedAt[id]
	if leased.IsZero() {
		leased = time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC)
	}
	return control.Operation{Kind: kind, ID: id, State: control.OperationActive, LeasedAt: &leased}, nil
}

func (f *fakeStore) ScheduledJobs(context.Context) ([]control.Job, error) {
	var jobs []control.Job
	for _, job := range f.jobs {
		if job.Status == protocol.StatusQueued || job.Status == protocol.StatusRunning {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (f *fakeStore) Job(_ context.Context, id string) (control.Job, error) {
	job, ok := f.jobs[id]
	if !ok {
		return control.Job{}, control.ErrNotFound
	}
	return job, nil
}

func (f *fakeStore) SyncJob(_ context.Context, id string, remote protocol.Job) (control.Job, error) {
	job, ok := f.jobs[id]
	if !ok {
		return control.Job{}, control.ErrNotFound
	}
	job.Status, job.FinishedAt, job.ExitCode, job.ErrorMessage = remote.Status, remote.FinishedAt, remote.ExitCode, remote.ErrorMessage
	f.jobs[id] = job
	return job, nil
}

type fakeScheduler struct {
	managed []protocol.Job
	removed []string
}

func (f *fakeScheduler) ManagedJobs(context.Context) ([]protocol.Job, error) {
	return append([]protocol.Job(nil), f.managed...), nil
}

func (f *fakeScheduler) Remove(_ context.Context, id string) error {
	f.removed = append(f.removed, id)
	return nil
}

func TestRunOnceReturnsDependencyErrors(t *testing.T) {
	want := errors.New("unavailable")
	reconciler := New(Config{Store: errorStore{err: want}, Scheduler: &fakeScheduler{}, Dispatcher: &fakeDispatcher{}})
	if err := reconciler.RunOnce(context.Background()); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

type errorStore struct{ err error }

func (e errorStore) ScheduledJobs(context.Context) ([]control.Job, error) { return nil, e.err }
func (e errorStore) Job(context.Context, string) (control.Job, error)     { return control.Job{}, e.err }
func (e errorStore) SyncJob(context.Context, string, protocol.Job) (control.Job, error) {
	return control.Job{}, e.err
}
func (e errorStore) Operation(context.Context, control.OperationKind, string) (control.Operation, error) {
	return control.Operation{}, e.err
}
func (e errorStore) StaleBuilds(context.Context, time.Time) ([]control.Build, error) {
	return nil, e.err
}
func (e errorStore) FinishBuild(context.Context, string, control.BuildStatus, int) (control.Build, error) {
	return control.Build{}, e.err
}
