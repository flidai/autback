package dispatcher_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/control/dispatcher"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
)

func TestDispatcherAdmitsExactlyOneOperationInSharedFIFO(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	first := queueJob(t, store, projectID, "first")
	build, _, err := store.CreateBuild(ctx, projectID, control.Idempotency{Key: "build", RequestHash: "build"})
	if err != nil {
		t.Fatal(err)
	}
	second := queueJob(t, store, projectID, "second")
	scheduler := &fakeScheduler{}
	d := dispatcher.New(store, scheduler)

	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.created) != 1 || scheduler.created[0].ID != first.ID {
		t.Fatalf("scheduled = %#v, want first job", scheduler.created)
	}
	if stored, _ := store.Build(ctx, build.ID); stored.Status != control.BuildQueued {
		t.Fatalf("build status = %s, want queued", stored.Status)
	}
	if err := d.RunOnce(ctx); err != nil || len(scheduler.created) != 1 {
		t.Fatalf("second dispatch err=%v scheduled=%d", err, len(scheduler.created))
	}

	if err := d.Release(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	if stored, _ := store.Build(ctx, build.ID); stored.Status != control.BuildRunning {
		t.Fatalf("build status = %s, want running", stored.Status)
	}
	if err := d.Release(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.created) != 2 || scheduler.created[1].ID != second.ID {
		t.Fatalf("scheduled = %#v, want second job after build", scheduler.created)
	}
}

func TestDispatcherFailsUnschedulableJobAndAdvances(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	first := queueJob(t, store, projectID, "first")
	second := queueJob(t, store, projectID, "second")
	scheduler := &fakeScheduler{failID: first.ID}
	d := dispatcher.New(store, scheduler)

	if err := d.RunOnce(ctx); err == nil {
		t.Fatal("dispatch error = nil")
	}
	failed, _ := store.Job(ctx, first.ID)
	if failed.Status != "failed" {
		t.Fatalf("first status = %s", failed.Status)
	}
	if len(scheduler.created) != 1 || scheduler.created[0].ID != second.ID {
		t.Fatalf("scheduled = %#v, want second job", scheduler.created)
	}
}

func TestDispatcherActivatesJobOnlyAfterSchedulerCreatesService(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "job")
	scheduler := &observingScheduler{store: store}
	d := dispatcher.New(store, scheduler)

	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if scheduler.stateDuringCreate != control.OperationAdmitting {
		t.Fatalf("state during create = %s, want admitting", scheduler.stateDuringCreate)
	}
	state, err := store.OperationState(ctx, control.OperationJob, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state != control.OperationActive {
		t.Fatalf("state after create = %s, want active", state)
	}
}

func TestDispatcherForwardsCancellationRequestedDuringAdmission(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "job-cancel")
	scheduler := &fakeScheduler{onCreate: func(job control.Job) error {
		_, err := store.RequestJobCancellation(ctx, job.ID)
		return err
	}}
	d := dispatcher.New(store, scheduler)

	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.cancelled) != 1 || scheduler.cancelled[0] != job.ID {
		t.Fatalf("cancelled = %#v, want %s", scheduler.cancelled, job.ID)
	}
}

func TestDispatcherPreservesAdmissionWhenActivationWriteFails(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "job-activation-failure")
	want := errors.New("database temporarily unavailable")
	d := dispatcher.New(activationFailStore{Store: store, err: want}, &fakeScheduler{})

	if err := d.RunOnce(ctx); !errors.Is(err, want) {
		t.Fatalf("dispatch error = %v, want %v", err, want)
	}
	stored, err := store.Job(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.Status) != "queued" {
		t.Fatalf("job status = %s, want queued for reconciliation", stored.Status)
	}
	state, err := store.OperationState(ctx, control.OperationJob, job.ID)
	if err != nil || state != control.OperationAdmitting {
		t.Fatalf("operation state = %s, %v; want admitting", state, err)
	}
}

func TestDispatcherLeavesFIFOQueuedWhenCapacityIsUnavailable(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "capacity-blocked")
	scheduler := &fakeScheduler{}
	want := errors.New("worker capacity exhausted")
	d := dispatcher.New(store, scheduler, dispatcher.WithCapacity(fakeCapacity{err: want}))

	if err := d.RunOnce(ctx); !errors.Is(err, want) {
		t.Fatalf("dispatch error = %v, want %v", err, want)
	}
	if len(scheduler.created) != 0 {
		t.Fatalf("scheduled = %#v, want none", scheduler.created)
	}
	state, err := store.OperationState(ctx, control.OperationJob, job.ID)
	if err != nil || state != control.OperationQueued {
		t.Fatalf("operation state = %s, %v; want queued", state, err)
	}
}

type activationFailStore struct {
	*controlsqlite.Store
	err error
}

type fakeCapacity struct{ err error }

func (f fakeCapacity) Ensure(context.Context) error { return f.err }

func (s activationFailStore) ActivateOperation(context.Context, control.OperationKind, string) error {
	return s.err
}

type fakeScheduler struct {
	created   []control.Job
	failID    string
	cancelled []string
	onCreate  func(control.Job) error
}

type observingScheduler struct {
	store             *controlsqlite.Store
	stateDuringCreate control.OperationState
}

func (s *observingScheduler) Create(ctx context.Context, job control.Job) error {
	state, err := s.store.OperationState(ctx, control.OperationJob, job.ID)
	if err != nil {
		return err
	}
	s.stateDuringCreate = state
	return nil
}

func (s *observingScheduler) Cancel(context.Context, string) error { return nil }

func (s *fakeScheduler) Create(_ context.Context, job control.Job) error {
	if job.ID == s.failID {
		return errors.New("worker rejected job")
	}
	s.created = append(s.created, job)
	if s.onCreate != nil {
		return s.onCreate(job)
	}
	return nil
}

func (s *fakeScheduler) Cancel(_ context.Context, id string) error {
	s.cancelled = append(s.cancelled, id)
	return nil
}

func queueFixture(t *testing.T) (*controlsqlite.Store, string) {
	t.Helper()
	store, err := controlsqlite.Open(t.TempDir(), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bootstrap, err := store.Bootstrap(context.Background(), control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	return store, bootstrap.Project.ID
}

func queueJob(t *testing.T, store *controlsqlite.Store, projectID, key string) control.Job {
	t.Helper()
	job, _, err := store.CreatePreparedJob(context.Background(), control.PrepareJob{
		ProjectID: projectID, Image: "runner@test", Command: []string{"true"}, Timeout: time.Minute,
	}, control.Idempotency{Key: key, RequestHash: key})
	if err != nil {
		t.Fatal(err)
	}
	job, err = store.QueueJob(context.Background(), job.ID, "digest/1")
	if err != nil {
		t.Fatal(err)
	}
	return job
}
