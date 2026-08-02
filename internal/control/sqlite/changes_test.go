package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
)

func TestCommittedControlChangesAreDurableAndNotifySubscribers(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{
		UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device",
	})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := store.CurrentRevision(ctx)
	if err != nil || revision == 0 {
		t.Fatalf("revision=%d err=%v; bootstrap must create durable changes", revision, err)
	}

	updates, unsubscribe := store.SubscribeChanges()
	defer unsubscribe()
	job, _, err := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "change-job", RequestHash: "change-job"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("committed change did not notify subscribers")
	}

	changes, err := store.ChangesAfter(ctx, revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) == 0 {
		t.Fatal("committed job was not journaled")
	}
	last := changes[len(changes)-1]
	if last.ProjectID != bootstrap.Project.ID || last.EntityKind != "job" || last.EntityID != job.ID {
		t.Fatalf("last change = %#v", last)
	}
}

func TestIdempotentReplayDoesNotCreateAnotherControlChange(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	input := testPreparedJob(bootstrap.Project.ID)
	idempotency := control.Idempotency{Key: "same-request", RequestHash: "same-request"}
	if _, _, err := store.CreatePreparedJob(ctx, input, idempotency); err != nil {
		t.Fatal(err)
	}
	revision, err := store.CurrentRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, replayed, err := store.CreatePreparedJob(ctx, input, idempotency); err != nil || !replayed {
		t.Fatalf("replayed=%v err=%v", replayed, err)
	}
	after, err := store.CurrentRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after != revision {
		t.Fatalf("revision advanced from %d to %d on a read-only replay", revision, after)
	}
}

func TestChangeJournalSurvivesStoreRestart(t *testing.T) {
	root := t.TempDir()
	pepper := []byte("test-pepper-that-is-at-least-32-bytes")
	store, err := controlsqlite.Open(root, pepper)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Bootstrap(context.Background(), control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"}); err != nil {
		t.Fatal(err)
	}
	want, err := store.CurrentRevision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = controlsqlite.Open(root, pepper)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.CurrentRevision(context.Background())
	if err != nil || got != want {
		t.Fatalf("revision after restart=%d err=%v; want %d", got, err, want)
	}
}
