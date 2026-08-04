package dispatcher_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/control/dispatcher"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
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
	if scheduled := scheduler.createdJobs(); len(scheduled) != 1 || scheduled[0].ID != first.ID {
		t.Fatalf("scheduled = %#v, want first job", scheduled)
	}
	if stored, _ := store.Build(ctx, build.ID); stored.Status != control.BuildQueued {
		t.Fatalf("build status = %s, want queued", stored.Status)
	}
	if err := d.RunOnce(ctx); err != nil || len(scheduler.createdJobs()) != 1 {
		t.Fatalf("second dispatch err=%v scheduled=%d", err, len(scheduler.createdJobs()))
	}

	if err := d.Release(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		stored, _ := store.Build(ctx, build.ID)
		return stored.Status == control.BuildRunning
	})
	if err := d.Release(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { return len(scheduler.createdJobs()) == 2 })
	if scheduled := scheduler.createdJobs(); scheduled[1].ID != second.ID {
		t.Fatalf("scheduled = %#v, want second job after build", scheduled)
	}
}

func TestDispatcherReservesPreparedJobWithoutCreatingRuntime(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: projectID, Image: "runner@test", Command: []string{"true"}, Timeout: time.Minute,
	}, control.Idempotency{Key: "prepared-job", RequestHash: "prepared-job"})
	if err != nil {
		t.Fatal(err)
	}
	scheduler := &fakeScheduler{}
	preparations := 0
	d := dispatcher.New(store, scheduler, dispatcher.WithAdmissionPreparer(admissionPreparerFunc(func(context.Context, control.Operation) error {
		preparations++
		return nil
	})))

	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if scheduled := scheduler.createdJobs(); len(scheduled) != 0 {
		t.Fatalf("scheduled prepared job before source upload: %#v", scheduled)
	}
	if preparations != 0 {
		t.Fatalf("runtime resources prepared %d times before source upload", preparations)
	}
	if state, err := store.OperationState(ctx, control.OperationJob, job.ID); err != nil || state != control.OperationAdmitting {
		t.Fatalf("prepared operation state = %s, %v; want admitting", state, err)
	}

	if _, err := store.QueueJob(ctx, job.ID, "digest/1"); err != nil {
		t.Fatal(err)
	}
	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if scheduled := scheduler.createdJobs(); len(scheduled) != 1 || scheduled[0].ID != job.ID {
		t.Fatalf("scheduled after source upload = %#v, want %s", scheduled, job.ID)
	}
	if preparations != 1 {
		t.Fatalf("runtime resources prepared %d times, want once after source upload", preparations)
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
	eventually(t, func() bool { return len(scheduler.createdJobs()) == 1 })
	if scheduled := scheduler.createdJobs(); scheduled[0].ID != second.ID {
		t.Fatalf("scheduled = %#v, want second job", scheduled)
	}
}

func TestBackgroundDispatcherWakesImmediatelyAfterFailedAdmissionCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, projectID := queueFixture(t)
	first := queueJob(t, store, projectID, "background-failure-first")
	second := queueJob(t, store, projectID, "background-failure-second")
	scheduler := &fakeScheduler{failID: first.ID}
	d := dispatcher.New(store, scheduler, dispatcher.WithAdvanceContext(ctx))

	d.Advance()
	eventually(t, func() bool {
		scheduled := scheduler.createdJobs()
		return len(scheduled) == 1 && scheduled[0].ID == second.ID
	})
}

func TestDispatcherReleaseWaitsForAsynchronousCleanupBeforeAdvancing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, projectID := queueFixture(t)
	first := queueJob(t, store, projectID, "cleanup-first")
	second := queueJob(t, store, projectID, "cleanup-second")
	scheduler := &fakeScheduler{}
	started := make(chan struct{})
	release := make(chan struct{})
	d := dispatcher.New(store, scheduler,
		dispatcher.WithAdvanceContext(ctx),
		dispatcher.WithCleaner(operationcleanup.CleanerFunc(func(_ context.Context, operation control.Operation) error {
			if operation.ID != first.ID {
				t.Fatalf("cleanup operation = %s, want %s", operation.ID, first.ID)
			}
			close(started)
			<-release
			return nil
		})),
	)

	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err := d.Release(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not start")
	}
	if scheduled := scheduler.createdJobs(); len(scheduled) != 1 {
		t.Fatalf("scheduled during cleanup = %#v", scheduled)
	}
	close(release)
	eventually(t, func() bool { return len(scheduler.createdJobs()) == 2 })
	if scheduled := scheduler.createdJobs(); scheduled[1].ID != second.ID {
		t.Fatalf("scheduled after cleanup = %#v", scheduled)
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

func TestDispatcherCapturesResourceBaselineBeforeCreatingRuntime(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "resource-baseline-before-runtime")
	prepared := false
	preparer := admissionPreparerFunc(func(_ context.Context, operation control.Operation) error {
		if operation.ID != job.ID || operation.State != control.OperationAdmitting {
			t.Fatalf("prepared operation = %#v", operation)
		}
		prepared = true
		return nil
	})
	scheduler := &fakeScheduler{onCreate: func(control.Job) error {
		if !prepared {
			t.Fatal("runtime created before resource baseline")
		}
		return nil
	}}
	d := dispatcher.New(store, scheduler, dispatcher.WithAdmissionPreparer(preparer))

	if err := d.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestDispatcherRequeuesOperationWhenResourceBaselineFails(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "resource-baseline-failure")
	want := errors.New("Docker daemon unavailable")
	d := dispatcher.New(store, &fakeScheduler{}, dispatcher.WithAdmissionPreparer(admissionPreparerFunc(func(context.Context, control.Operation) error {
		return want
	})))

	if err := d.RunOnce(ctx); !errors.Is(err, want) {
		t.Fatalf("RunOnce error = %v, want %v", err, want)
	}
	state, err := store.OperationState(ctx, control.OperationJob, job.ID)
	if err != nil || state != control.OperationQueued {
		t.Fatalf("operation state = %s, %v; want queued", state, err)
	}
}

func TestDispatcherFailsPermanentPreparationAndAdvancesAfterCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, projectID := queueFixture(t)
	first := queueJob(t, store, projectID, "revoked-secret-first")
	second := queueJob(t, store, projectID, "revoked-secret-second")
	d := dispatcher.New(store, &fakeScheduler{},
		dispatcher.WithAdvanceContext(ctx),
		dispatcher.WithAdmissionPreparer(admissionPreparerFunc(func(_ context.Context, operation control.Operation) error {
			if operation.ID == first.ID {
				return permanentPreparationError{errors.New("secret reference revoked")}
			}
			return nil
		})),
	)
	d.Advance()
	eventually(t, func() bool {
		job, _ := store.Job(ctx, first.ID)
		state, _ := store.OperationState(ctx, control.OperationJob, first.ID)
		return job.Status == "failed" && state == control.OperationReleased
	})
	eventually(t, func() bool {
		state, _ := store.OperationState(ctx, control.OperationJob, second.ID)
		return state == control.OperationActive
	})
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
	if cancelled := scheduler.cancelledJobs(); len(cancelled) != 1 || cancelled[0] != job.ID {
		t.Fatalf("cancelled = %#v, want %s", cancelled, job.ID)
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
	if scheduled := scheduler.createdJobs(); len(scheduled) != 0 {
		t.Fatalf("scheduled = %#v, want none", scheduled)
	}
	state, err := store.OperationState(ctx, control.OperationJob, job.ID)
	if err != nil || state != control.OperationQueued {
		t.Fatalf("operation state = %s, %v; want queued", state, err)
	}
}

func TestDispatcherDrainStopsNewAdmission(t *testing.T) {
	ctx := context.Background()
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "draining")
	scheduler := &fakeScheduler{}
	d := dispatcher.New(store, scheduler)

	d.Drain()
	if err := d.RunOnce(ctx); !errors.Is(err, dispatcher.ErrDraining) {
		t.Fatalf("RunOnce error = %v, want %v", err, dispatcher.ErrDraining)
	}
	d.Advance()
	time.Sleep(10 * time.Millisecond)
	if scheduled := scheduler.createdJobs(); len(scheduled) != 0 {
		t.Fatalf("scheduled while draining = %#v", scheduled)
	}
	if state, err := store.OperationState(ctx, control.OperationJob, job.ID); err != nil || state != control.OperationQueued {
		t.Fatalf("queued operation = %s, %v", state, err)
	}
}

func TestDispatcherDrainWaitsForInFlightAdmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store, projectID := queueFixture(t)
	job := queueJob(t, store, projectID, "in-flight-drain")
	scheduler := &blockingScheduler{started: make(chan struct{})}
	var reported atomic.Int32
	d := dispatcher.New(store, scheduler,
		dispatcher.WithAdvanceContext(ctx),
		dispatcher.WithErrorHandler(func(error) { reported.Add(1) }),
	)

	d.Advance()
	select {
	case <-scheduler.started:
	case <-time.After(time.Second):
		t.Fatal("admission did not start")
	}
	d.Drain()
	waitCtx, stopWaiting := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stopWaiting()
	if err := d.Wait(waitCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error before cancellation = %v, want deadline exceeded", err)
	}

	cancel()
	waitCtx, stopWaiting = context.WithTimeout(context.Background(), time.Second)
	defer stopWaiting()
	if err := d.Wait(waitCtx); err != nil {
		t.Fatalf("Wait error after cancellation = %v", err)
	}
	if reported.Load() != 0 {
		t.Fatalf("reported errors during normal drain = %d, want 0", reported.Load())
	}
	if state, err := store.OperationState(context.Background(), control.OperationJob, job.ID); err != nil || state != control.OperationAdmitting {
		t.Fatalf("operation after interrupted admission = %s, %v; want admitting for restart reconciliation", state, err)
	}
}

type activationFailStore struct {
	*controlsqlite.Store
	err error
}

type fakeCapacity struct{ err error }

func (f fakeCapacity) Admit(_ context.Context, reserve func() error) error {
	if f.err != nil {
		return f.err
	}
	return reserve()
}

func (s activationFailStore) ActivateOperation(context.Context, control.OperationKind, string) error {
	return s.err
}

type fakeScheduler struct {
	mu        sync.Mutex
	created   []control.Job
	failID    string
	cancelled []string
	onCreate  func(control.Job) error
}

type blockingScheduler struct {
	started chan struct{}
}

type admissionPreparerFunc func(context.Context, control.Operation) error

type permanentPreparationError struct{ error }

func (permanentPreparationError) Permanent() bool { return true }

func (f admissionPreparerFunc) Prepare(ctx context.Context, operation control.Operation) error {
	return f(ctx, operation)
}

func (s *blockingScheduler) Create(ctx context.Context, _ control.Job) error {
	close(s.started)
	<-ctx.Done()
	return ctx.Err()
}

func (*blockingScheduler) Cancel(context.Context, string) error { return nil }

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
	s.mu.Lock()
	s.created = append(s.created, job)
	s.mu.Unlock()
	if s.onCreate != nil {
		return s.onCreate(job)
	}
	return nil
}

func (s *fakeScheduler) Cancel(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelled = append(s.cancelled, id)
	return nil
}

func (s *fakeScheduler) createdJobs() []control.Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]control.Job(nil), s.created...)
}

func (s *fakeScheduler) cancelledJobs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.cancelled...)
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

func eventually(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
