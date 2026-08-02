package dispatcher_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flidai/outback/internal/control"
	"github.com/flidai/outback/internal/control/dispatcher"
	controlsqlite "github.com/flidai/outback/internal/control/sqlite"
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

type fakeScheduler struct {
	created []control.Job
	failID  string
}

func (s *fakeScheduler) Create(_ context.Context, job control.Job) error {
	if job.ID == s.failID {
		return errors.New("worker rejected job")
	}
	s.created = append(s.created, job)
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
