package cleanup_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	"github.com/flidai/autback/internal/operation/cleanup"
)

func TestCoordinatorKeepsLeaseUntilAsynchronousCleanupCompletes(t *testing.T) {
	store, first, second := cleanupFixture(t)
	started := make(chan struct{})
	release := make(chan struct{})
	completed := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	coordinator := cleanup.New(store, cleanup.CleanerFunc(func(context.Context, control.Operation) error {
		close(started)
		<-release
		return nil
	}), cleanup.WithContext(ctx), cleanup.WithCompleted(func(control.Operation) { completed <- struct{}{} }))

	requestCtx, stopRequest := context.WithCancel(context.Background())
	if err := coordinator.Request(requestCtx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	stopRequest()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not start")
	}
	if next, err := store.AcquireNextOperation(context.Background()); err != nil || next != nil {
		t.Fatalf("acquire while cleanup is blocked = %#v, %v", next, err)
	}
	close(release)
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not complete")
	}
	operation, err := store.Operation(context.Background(), control.OperationJob, first.ID)
	if err != nil || operation.State != control.OperationReleased {
		t.Fatalf("released operation = %#v, %v", operation, err)
	}
	next, err := store.AcquireNextOperation(context.Background())
	if err != nil || next == nil || next.ID != second.ID {
		t.Fatalf("next operation = %#v, %v; want %s", next, err, second.ID)
	}
}

func TestCoordinatorRetriesFailedCleanupAndRecordsAttempts(t *testing.T) {
	store, first, _ := cleanupFixture(t)
	var attempts atomic.Int32
	errorsSeen := make(chan error, 1)
	completed := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	coordinator := cleanup.New(store, cleanup.CleanerFunc(func(context.Context, control.Operation) error {
		if attempts.Add(1) == 1 {
			return errors.New("docker temporarily unavailable")
		}
		return nil
	}),
		cleanup.WithContext(ctx),
		cleanup.WithRetryDelay(time.Millisecond),
		cleanup.WithErrorHandler(func(err error) { errorsSeen <- err }),
		cleanup.WithCompleted(func(control.Operation) { completed <- struct{}{} }),
	)

	if err := coordinator.Request(context.Background(), control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errorsSeen:
		if err == nil || err.Error() == "" {
			t.Fatalf("cleanup error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first cleanup failure was not reported")
	}
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("cleanup retry did not complete")
	}
	operation, err := store.Operation(context.Background(), control.OperationJob, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if operation.State != control.OperationReleased || operation.CleanupAttempts != 2 || attempts.Load() != 2 {
		t.Fatalf("operation after retry = %#v, cleaner attempts = %d", operation, attempts.Load())
	}
}

func cleanupFixture(t *testing.T) (*controlsqlite.Store, control.Job, control.Job) {
	t.Helper()
	store, err := controlsqlite.Open(t.TempDir(), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	create := func(key string) control.Job {
		job, _, createErr := store.CreatePreparedJob(ctx, control.PrepareJob{
			ProjectID: bootstrap.Project.ID, Image: "runner@test", Command: []string{"true"}, Timeout: time.Minute,
		}, control.Idempotency{Key: key, RequestHash: key})
		if createErr != nil {
			t.Fatal(createErr)
		}
		job, createErr = store.QueueJob(ctx, job.ID, "digest/1")
		if createErr != nil {
			t.Fatal(createErr)
		}
		return job
	}
	first, second := create("cleanup-first"), create("cleanup-second")
	operation, err := store.AcquireNextOperation(ctx)
	if err != nil || operation == nil || operation.ID != first.ID {
		t.Fatalf("first operation = %#v, %v", operation, err)
	}
	if err := store.ActivateOperation(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	return store, first, second
}
