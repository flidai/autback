package console

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	"github.com/flidai/autback/internal/protocol"
)

func TestSQLiteSourceBuildsAnAuthorizedConsoleProjection(t *testing.T) {
	store, bootstrap, principal := consoleStore(t)
	ctx := context.Background()
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: bootstrap.Project.ID,
		Image:     "registry.example/autback-runner@sha256:" + strings.Repeat("a", 64),
		Command:   []string{"task", "ci"}, WorkingDirectory: "/workspace",
		Environment: map[string]string{"CI": "true"},
		Caches:      []control.CacheMount{{Name: "go-build", Target: "/root/.cache/go-build"}},
		Timeout:     15 * time.Minute,
	}, control.Idempotency{Key: "console-job", RequestHash: "console-job"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.QueueJob(ctx, job.ID, "root/42"); err != nil {
		t.Fatal(err)
	}
	if err := store.Audit(ctx, principal, bootstrap.Project.ID, "job.start", job.ID, map[string]string{"command": "task ci"}); err != nil {
		t.Fatal(err)
	}

	source, err := NewSQLiteSource(SQLiteSourceConfig{
		Store: store, Scheduler: &consoleScheduler{logs: "compile\ntest\nPASS\n"}, Version: "0.1.0", StartedAt: time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := source.Snapshot(ctx, principal, Route{Kind: RouteOverview})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision == 0 || snapshot.Session.User != "Owner" || len(snapshot.Session.Projects) != 1 {
		t.Fatalf("session snapshot = %#v", snapshot)
	}
	if snapshot.Service.Control != "CLI only" || snapshot.Service.Admission != "Strict FIFO" || snapshot.Worker.Status != "online" {
		t.Fatalf("service=%#v worker=%#v", snapshot.Service, snapshot.Worker)
	}
	if len(snapshot.Queue) != 1 || snapshot.Queue[0].ID != job.ID || snapshot.Queue[0].Position != 1 {
		t.Fatalf("queue = %#v", snapshot.Queue)
	}
	if len(snapshot.Operations) == 0 || snapshot.Operations[0].Command != "task ci" {
		t.Fatalf("operations = %#v", snapshot.Operations)
	}
	if len(snapshot.Audit) == 0 || snapshot.Audit[0].Action != "job.start" {
		t.Fatalf("audit = %#v", snapshot.Audit)
	}
}

func TestSQLiteSourceBuildsOperationDetailAndBoundedLogTail(t *testing.T) {
	store, bootstrap, principal := consoleStore(t)
	ctx := context.Background()
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: bootstrap.Project.ID, Image: "runner@test", Command: []string{"go", "test", "./..."},
		WorkingDirectory: "/workspace", Timeout: time.Minute,
	}, control.Idempotency{Key: "operation-detail", RequestHash: "operation-detail"})
	if err != nil {
		t.Fatal(err)
	}
	logs := strings.Repeat("old output\n", 8000) + "last useful line\n"
	source, err := NewSQLiteSource(SQLiteSourceConfig{Store: store, Scheduler: &consoleScheduler{logs: logs}, Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := source.Snapshot(ctx, principal, Route{Kind: RouteOperation, OperationKind: "job", OperationID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Operation == nil || snapshot.Operation.ID != job.ID || snapshot.Operation.WorkingDirectory != "/workspace" {
		t.Fatalf("operation = %#v", snapshot.Operation)
	}
	if !snapshot.Log.Available || !snapshot.Log.Truncated || !strings.HasSuffix(snapshot.Log.Content, "last useful line\n") || len(snapshot.Log.Content) > maxLogTailBytes {
		t.Fatalf("log = available:%v truncated:%v bytes:%d suffix:%q", snapshot.Log.Available, snapshot.Log.Truncated, len(snapshot.Log.Content), tail(snapshot.Log.Content, 32))
	}
}

func TestSQLiteSourceRejectsAnOperationOutsideThePrincipalProjects(t *testing.T) {
	store, bootstrap, _ := consoleStore(t)
	ctx := context.Background()
	admin := control.Principal{Kind: control.PrincipalDevice, UserID: bootstrap.User.ID, Admin: true}
	other, err := store.CreateProject(ctx, admin, "other", "Other")
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{ProjectID: other.ID, Image: "runner@test", Command: []string{"true"}, Timeout: time.Minute}, control.Idempotency{Key: "other-operation", RequestHash: "other-operation"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewSQLiteSource(SQLiteSourceConfig{Store: store, Scheduler: &consoleScheduler{}, Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	github := control.Principal{Kind: control.PrincipalGitHub, ProjectID: bootstrap.Project.ID, Subject: "repo:trusted"}
	if _, err := source.Snapshot(ctx, github, Route{Kind: RouteOperation, OperationKind: "job", OperationID: job.ID}); err != control.ErrForbidden {
		t.Fatalf("err=%v; want forbidden", err)
	}
}

func consoleStore(t *testing.T) (*controlsqlite.Store, control.BootstrapResult, control.Principal) {
	t.Helper()
	store, err := controlsqlite.Open(filepath.Join(t.TempDir(), "control"), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bootstrap, err := store.Bootstrap(context.Background(), control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example Service", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := store.Authenticate(context.Background(), bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	return store, bootstrap, principal
}

type consoleScheduler struct {
	logs string
	err  error
}

func (s *consoleScheduler) Check(context.Context) error                 { return s.err }
func (s *consoleScheduler) ValidateImage(context.Context, string) error { return nil }
func (s *consoleScheduler) Create(context.Context, control.Job) error   { return nil }
func (s *consoleScheduler) Status(context.Context, string) (protocol.Job, error) {
	return protocol.Job{}, nil
}
func (s *consoleScheduler) Cancel(context.Context, string) error { return nil }
func (s *consoleScheduler) Logs(_ context.Context, _ string, _ bool, output io.Writer) error {
	_, err := io.WriteString(output, s.logs)
	return err
}

func tail(value string, size int) string {
	if len(value) <= size {
		return value
	}
	return value[len(value)-size:]
}
