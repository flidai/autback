package console

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	"github.com/flidai/autback/internal/operation/redact"
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
	observedAt := time.Now().UTC().Add(-time.Second)
	if err := store.AppendResourceSample(ctx, control.ResourceSample{
		ObservedAt: observedAt, ResourceScope: control.ResourceScope{ProjectID: bootstrap.Project.ID, OperationKind: control.OperationJob, OperationID: job.ID},
		CPUUtilization: .75, CPUCores: 4, MemoryUtilization: .5, MemoryUsageBytes: 4 << 30, MemoryTotalBytes: 8 << 30,
		DiskUsageBytes: 40 << 30, DiskTotalBytes: 80 << 30,
	}); err != nil {
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
	if snapshot.Service.Control != "CLI only" || snapshot.Service.Admission != "One at a time" || snapshot.Worker.Status != "online" {
		t.Fatalf("service=%#v worker=%#v", snapshot.Service, snapshot.Worker)
	}
	if len(snapshot.Queue) != 1 || snapshot.Queue[0].ID != job.ID || snapshot.Queue[0].Position != 1 {
		t.Fatalf("queue = %#v", snapshot.Queue)
	}
	if len(snapshot.Operations) == 0 || snapshot.Operations[0].Command != "task ci" {
		t.Fatalf("operations = %#v", snapshot.Operations)
	}
	if snapshot.Resources.CPUCores != 4 || snapshot.Resources.CPUPeak != .75 || len(snapshot.Resources.Samples) != 1 || snapshot.Operations[0].Resources.SampleCount != 1 {
		t.Fatalf("resources=%#v operation=%#v", snapshot.Resources, snapshot.Operations[0])
	}
	if len(snapshot.Audit) == 0 || snapshot.Audit[0].Action != "job.start" {
		t.Fatalf("audit = %#v", snapshot.Audit)
	}
}

func TestSQLiteSourceBuildsOperationDetailAndStreamsBoundedLogTail(t *testing.T) {
	store, bootstrap, principal := consoleStore(t)
	ctx := context.Background()
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: bootstrap.Project.ID, Image: "runner@test", Command: []string{"go", "test", "./..."},
		WorkingDirectory: "/workspace", Timeout: time.Minute,
	}, control.Idempotency{Key: "operation-detail", RequestHash: "operation-detail"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.FailJob(ctx, job.ID, "test failure"); err != nil {
		t.Fatal(err)
	}
	logContent := strings.Repeat("old output\n", 8000) + "last useful line\n"
	source, err := NewSQLiteSource(SQLiteSourceConfig{Store: store, Scheduler: &consoleScheduler{logs: logContent}, Version: "0.1.0"})
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
	logs, err := source.SubscribeLog(ctx, principal, Route{Kind: RouteOperation, OperationKind: "job", OperationID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	var view LogView
	var open bool
	select {
	case view, open = <-logs:
	case <-time.After(time.Second):
		t.Fatal("durable log tail was not published")
	}
	if !open || !view.Available || !view.Truncated || !strings.HasSuffix(view.Content, "last useful line\n") || len(view.Content) > maxLogTailBytes {
		t.Fatalf("log = open:%v available:%v truncated:%v bytes:%d suffix:%q", open, view.Available, view.Truncated, len(view.Content), tail(view.Content, 32))
	}
}

func TestConsoleProjectionContainsSecretReferenceButNeverResolvedValue(t *testing.T) {
	const sentinel = "autback-console-sentinel-secret-value"
	store, bootstrap, principal := consoleStore(t)
	ctx := context.Background()
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: bootstrap.Project.ID, Image: "runner@test", Command: []string{"task", "ci"}, Timeout: time.Minute,
		Secrets: []control.SecretBinding{{Name: "registry-token", Environment: "REGISTRY_TOKEN"}},
	}, control.Idempotency{Key: "console-secret", RequestHash: "console-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RecordSecretAccess(ctx, bootstrap.Project.ID, job.ID, "registry-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FailJob(ctx, job.ID, "provider reference revoked"); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	redactor, err := redact.NewWriter(&logs, []string{sentinel})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = redactor.Write([]byte("output " + sentinel + "\n"))
	if err := redactor.Close(); err != nil {
		t.Fatal(err)
	}
	source, err := NewSQLiteSource(SQLiteSourceConfig{Store: store, Scheduler: &consoleScheduler{logs: logs.String()}, Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	route := Route{Kind: RouteOperation, OperationKind: "job", OperationID: job.ID}
	snapshot, err := source.Snapshot(ctx, principal, route)
	if err != nil {
		t.Fatal(err)
	}
	updates, err := source.SubscribeLog(ctx, principal, route)
	if err != nil {
		t.Fatal(err)
	}
	logView := <-updates
	payload, err := json.Marshal(struct {
		Snapshot Snapshot
		Log      LogView
	}{snapshot, logView})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), sentinel) {
		t.Fatalf("console projection contains sentinel: %s", payload)
	}
	foundName := false
	for _, event := range snapshot.Audit {
		if event.Action == "job.secret.access" && event.Metadata["name"] == "registry-token" {
			foundName = true
		}
	}
	if !foundName {
		t.Fatalf("console audit omits useful secret reference name: %#v", snapshot.Audit)
	}
}

func TestSQLiteSourceStreamsABoundedLiveJobLogTail(t *testing.T) {
	store, bootstrap, principal := consoleStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: bootstrap.Project.ID, Image: "runner@test", Command: []string{"task", "ci"}, Timeout: time.Minute,
	}, control.Idempotency{Key: "live-log", RequestHash: "live-log"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.QueueJob(ctx, job.ID, "root/42"); err != nil {
		t.Fatal(err)
	}
	if operation, err := store.AcquireNextOperation(ctx); err != nil || operation == nil || operation.ID != job.ID {
		t.Fatalf("acquire operation=%#v err=%v", operation, err)
	}
	if err := store.ActivateOperation(ctx, control.OperationJob, job.ID); err != nil {
		t.Fatal(err)
	}
	chunks := make(chan string, 2)
	source, err := NewSQLiteSource(SQLiteSourceConfig{Store: store, Scheduler: &consoleScheduler{follow: chunks}, Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	logs, err := source.SubscribeLog(ctx, principal, Route{Kind: RouteOperation, OperationKind: "job", OperationID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	chunks <- strings.Repeat("old output\n", 8000)
	chunks <- "live useful line\n"
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		select {
		case view, open := <-logs:
			if !open {
				t.Fatal("live log stream closed before publishing output")
			}
			if !strings.HasSuffix(view.Content, "live useful line\n") {
				continue
			}
			if !view.Available || !view.Truncated || len(view.Content) > maxLogTailBytes {
				t.Fatalf("log = available:%v truncated:%v bytes:%d", view.Available, view.Truncated, len(view.Content))
			}
			return
		case <-deadline.C:
			t.Fatal("live log tail was not published")
		}
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
	route := Route{Kind: RouteOperation, OperationKind: "job", OperationID: job.ID}
	if _, err := source.Snapshot(ctx, github, route); err != control.ErrForbidden {
		t.Fatalf("err=%v; want forbidden", err)
	}
	if _, err := source.SubscribeLog(ctx, github, route); err != control.ErrForbidden {
		t.Fatalf("subscribe log err=%v; want forbidden", err)
	}
}

func TestResourceRollupViewKeepsHistoricalRunCharts(t *testing.T) {
	now := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)
	view := resourceRollupView([]control.ResourceRollup{
		{BucketAt: now.Add(time.Minute), SampleCount: 2, CPUAverage: .8, CPUPeak: .9, MemoryAverage: .6, MemoryPeak: .7, MemoryBytesPeak: 6 << 30, CPUCores: 4, DiskUsageBytes: 40 << 30, DiskTotalBytes: 80 << 30},
		{BucketAt: now, SampleCount: 1, CPUAverage: .2, CPUPeak: .3, MemoryAverage: .3, MemoryPeak: .4, MemoryBytesPeak: 4 << 30, CPUCores: 4, DiskUsageBytes: 39 << 30, DiskTotalBytes: 80 << 30},
	}, nil)
	if view.SampleCount != 3 || len(view.Samples) != 2 || !view.Samples[0].ObservedAt.Equal(now) || view.CPUAverage != .6 || view.CPUPeak != .9 || view.MemoryBytesPeak != 6<<30 {
		t.Fatalf("view=%#v", view)
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
	logs   string
	err    error
	follow <-chan string
}

func (s *consoleScheduler) Check(context.Context) error                 { return s.err }
func (s *consoleScheduler) ValidateImage(context.Context, string) error { return nil }
func (s *consoleScheduler) Create(context.Context, control.Job) error   { return nil }
func (s *consoleScheduler) Status(context.Context, string) (protocol.Job, error) {
	return protocol.Job{}, nil
}
func (s *consoleScheduler) Cancel(context.Context, string) error { return nil }
func (s *consoleScheduler) Logs(ctx context.Context, _ string, follow bool, output io.Writer) error {
	if !follow || s.follow == nil {
		_, err := io.WriteString(output, s.logs)
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, open := <-s.follow:
			if !open {
				return nil
			}
			if _, err := io.WriteString(output, chunk); err != nil {
				return err
			}
		}
	}
}

func tail(value string, size int) string {
	if len(value) <= size {
		return value
	}
	return value[len(value)-size:]
}
