package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
)

func TestOperationsShareOneDurableFIFO(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "job-1", RequestHash: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.QueueJob(ctx, first.ID, "digest/1"); err != nil {
		t.Fatal(err)
	}
	build, _, err := store.CreateBuild(ctx, bootstrap.Project.ID, control.Idempotency{Key: "build-1", RequestHash: "build-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "job-2", RequestHash: "job-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.QueueJob(ctx, second.ID, "digest/2"); err != nil {
		t.Fatal(err)
	}

	assertNextOperation(t, store, control.OperationJob, first.ID)
	if operation, err := store.AcquireNextOperation(ctx); err != nil || operation != nil {
		t.Fatalf("second acquire = %#v, %v; want no operation while lease is active", operation, err)
	}
	if err := store.ActivateOperation(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.ReleaseOperation(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	assertNextOperation(t, store, control.OperationBuild, build.ID)
	if err := store.ActivateOperation(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	storedBuild, err := store.Build(ctx, build.ID)
	if err != nil || storedBuild.Status != control.BuildRunning {
		t.Fatalf("build = %#v, %v; want running", storedBuild, err)
	}
	if err := store.ReleaseOperation(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	assertNextOperation(t, store, control.OperationJob, second.ID)
}

func TestCancelQueuedOperationPreservesFIFO(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "job-1", RequestHash: "job-1"})
	second, _, _ := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "job-2", RequestHash: "job-2"})
	_, _ = store.QueueJob(ctx, first.ID, "digest/1")
	_, _ = store.QueueJob(ctx, second.ID, "digest/2")
	if cancelled, err := store.CancelQueuedOperation(ctx, control.OperationJob, first.ID); err != nil || !cancelled {
		t.Fatalf("cancelled=%v err=%v", cancelled, err)
	}
	assertNextOperation(t, store, control.OperationJob, second.ID)
}

func TestActiveWorkerLeaseSurvivesStoreRestart(t *testing.T) {
	root := t.TempDir()
	pepper := []byte("test-pepper-that-is-at-least-32-bytes")
	store, err := controlsqlite.Open(root, pepper)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "job-1", RequestHash: "job-1"})
	second, _, _ := store.CreatePreparedJob(ctx, testPreparedJob(bootstrap.Project.ID), control.Idempotency{Key: "job-2", RequestHash: "job-2"})
	_, _ = store.QueueJob(ctx, first.ID, "digest/1")
	_, _ = store.QueueJob(ctx, second.ID, "digest/2")
	assertNextOperation(t, store, control.OperationJob, first.ID)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = controlsqlite.Open(root, pepper)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if operation, err := store.AcquireNextOperation(ctx); err != nil || operation != nil {
		t.Fatalf("acquire after restart = %#v, %v; active lease was not preserved", operation, err)
	}
	if err := store.ReleaseOperation(ctx, control.OperationJob, first.ID); err != nil {
		t.Fatal(err)
	}
	assertNextOperation(t, store, control.OperationJob, second.ID)
}

func TestBuildLeaseHeartbeatCoversQueuedAndRunningBuilds(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	build, _, err := store.CreateBuild(ctx, bootstrap.Project.ID, control.Idempotency{Key: "build-heartbeat", RequestHash: "build-heartbeat"})
	if err != nil {
		t.Fatal(err)
	}
	beforeHeartbeat := time.Now().UTC()
	if err := store.RenewOperationLease(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	if stale, err := store.StaleBuilds(ctx, beforeHeartbeat); err != nil || len(stale) != 0 {
		t.Fatalf("queued stale builds = %#v, %v; want none after heartbeat", stale, err)
	}
	assertNextOperation(t, store, control.OperationBuild, build.ID)
	if err := store.ActivateOperation(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	beforeHeartbeat = time.Now().UTC()
	if err := store.RenewOperationLease(ctx, control.OperationBuild, build.ID); err != nil {
		t.Fatal(err)
	}
	if stale, err := store.StaleBuilds(ctx, beforeHeartbeat); err != nil || len(stale) != 0 {
		t.Fatalf("running stale builds = %#v, %v; want none after heartbeat", stale, err)
	}
	if stale, err := store.StaleBuilds(ctx, time.Now().Add(time.Second)); err != nil || len(stale) != 1 || stale[0].ID != build.ID {
		t.Fatalf("expired stale builds = %#v, %v; want %s", stale, err, build.ID)
	}
}

func assertNextOperation(t *testing.T, store *controlsqlite.Store, kind control.OperationKind, id string) {
	t.Helper()
	operation, err := store.AcquireNextOperation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if operation == nil || operation.Kind != kind || operation.ID != id || operation.State != control.OperationAdmitting {
		t.Fatalf("operation = %#v, want %s %s admitting", operation, kind, id)
	}
}

func testPreparedJob(projectID string) control.PrepareJob {
	return control.PrepareJob{ProjectID: projectID, Image: "runner@test", Command: []string{"true"}, Timeout: time.Minute}
}

func TestOpenMigratesAdmissionIdempotencyColumns(t *testing.T) {
	root := t.TempDir()
	database, err := sql.Open("sqlite", filepath.Join(root, "control.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`
CREATE TABLE projects (
  id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, name TEXT NOT NULL, created_at INTEGER NOT NULL
);
CREATE TABLE control_jobs (
  id TEXT PRIMARY KEY, project_id TEXT NOT NULL, image TEXT NOT NULL, command_json TEXT NOT NULL,
  working_directory TEXT NOT NULL, environment_json TEXT NOT NULL, root_digest TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL, timeout_millis INTEGER NOT NULL, cpus TEXT NOT NULL, memory TEXT NOT NULL,
  created_at INTEGER NOT NULL, started_at INTEGER, finished_at INTEGER, exit_code INTEGER,
  error_message TEXT NOT NULL DEFAULT '', cancel_requested INTEGER NOT NULL DEFAULT 0, worker_id TEXT NOT NULL DEFAULT ''
);
CREATE TABLE control_builds (
  id TEXT PRIMARY KEY, project_id TEXT NOT NULL, status TEXT NOT NULL, created_at INTEGER NOT NULL,
  finished_at INTEGER, exit_code INTEGER
);`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := controlsqlite.Open(root, []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	database, err = sql.Open("sqlite", filepath.Join(root, "control.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for _, table := range []string{"control_jobs", "control_builds"} {
		rows, err := database.Query(`PRAGMA table_info(` + table + `)`)
		if err != nil {
			t.Fatal(err)
		}
		columns := map[string]bool{}
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, kind string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
				t.Fatal(err)
			}
			columns[name] = true
		}
		_ = rows.Close()
		if !columns["idempotency_key"] || !columns["request_hash"] || table == "control_jobs" && !columns["caches_json"] {
			t.Fatalf("%s columns = %#v", table, columns)
		}
	}
	rows, err := database.Query(`PRAGMA table_info(projects)`)
	if err != nil {
		t.Fatal(err)
	}
	projectColumns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		projectColumns[name] = true
	}
	_ = rows.Close()
	for _, name := range []string{"active_image", "previous_image", "allow_image_overrides"} {
		if !projectColumns[name] {
			t.Fatalf("projects columns = %#v", projectColumns)
		}
	}
}

func TestBootstrapTokenAuthenticatesWithoutPersistingSecret(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{
		UserName: "Owner", ProjectSlug: "example-service", ProjectName: "Example Service", TokenName: "owner-laptop",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bootstrap.Token == "" || bootstrap.User.ID == "" || bootstrap.Project.ID == "" {
		t.Fatalf("bootstrap = %#v", bootstrap)
	}

	principal, err := store.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.UserID != bootstrap.User.ID || !principal.Admin || principal.Kind != control.PrincipalDevice {
		t.Fatalf("principal = %#v", principal)
	}
	project, err := store.AuthorizeProject(ctx, principal, "example-service")
	if err != nil || project.ID != bootstrap.Project.ID {
		t.Fatalf("project=%#v err=%v", project, err)
	}

	database, err := sql.Open("sqlite", filepath.Join(store.Root(), "control.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var persisted string
	if err := database.QueryRowContext(ctx, `SELECT digest FROM access_tokens LIMIT 1`).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted == bootstrap.Token || persisted == "" {
		t.Fatalf("persisted token material = %q", persisted)
	}
}

func TestBackupProducesAConsistentStandaloneDatabase(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	if _, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"}); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "control.db")
	if err := store.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	copy, err := controlsqlite.Open(filepath.Dir(backup), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	defer copy.Close()
	initialized, err := copy.Initialized(ctx)
	if err != nil || !initialized {
		t.Fatalf("initialized=%v err=%v", initialized, err)
	}
}

func TestDeviceTokensAreIndependentlyRevocable(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{
		UserName: "Owner", ProjectSlug: "example-service", ProjectName: "Example Service", TokenName: "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := store.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateDeviceToken(ctx, principal, control.CreateDeviceToken{
		UserID: bootstrap.User.ID, Name: "second", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, second.Secret); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeDeviceToken(ctx, principal, second.Metadata.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, second.Secret); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("revoked authentication error = %v", err)
	}
	if _, err := store.Authenticate(ctx, bootstrap.Token); err != nil {
		t.Fatalf("first device token was affected: %v", err)
	}
}

func TestEnrollmentCodeIsSingleUseAndCreatesIndependentDeviceToken(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "one", TokenName: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := store.Authenticate(ctx, bootstrap.Token)
	user, err := store.CreateUser(ctx, owner, "Coworker", false)
	if err != nil {
		t.Fatal(err)
	}
	enrollment, err := store.CreateEnrollmentCode(ctx, owner, user.ID, "coworker-laptop", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", filepath.Join(store.Root(), "control.db"))
	if err != nil {
		t.Fatal(err)
	}
	var persistedDigest string
	if err := database.QueryRowContext(ctx, `SELECT digest FROM enrollment_codes WHERE id=?`, enrollment.Metadata.ID).Scan(&persistedDigest); err != nil {
		t.Fatal(err)
	}
	_ = database.Close()
	if persistedDigest == "" || persistedDigest == enrollment.Secret {
		t.Fatalf("persisted enrollment material = %q", persistedDigest)
	}
	issued, _, err := store.ExchangeEnrollmentCode(ctx, enrollment.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Metadata.UserID != user.ID || issued.Metadata.Name != "coworker-laptop" || issued.Secret == "" {
		t.Fatalf("issued token = %#v", issued)
	}
	if _, _, err := store.ExchangeEnrollmentCode(ctx, enrollment.Secret); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("reuse error = %v", err)
	}
	principal, err := store.Authenticate(ctx, issued.Secret)
	if err != nil || principal.UserID != user.ID {
		t.Fatalf("principal=%#v err=%v", principal, err)
	}
	if _, err := store.Authenticate(ctx, bootstrap.Token); err != nil {
		t.Fatalf("owner token was affected: %v", err)
	}
}

func TestEnrollmentCodeExpiryAndRetryLimitFailClosed(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "one", TokenName: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := store.Authenticate(ctx, bootstrap.Token)
	expired, err := store.CreateEnrollmentCode(ctx, owner, bootstrap.User.ID, "expired", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", filepath.Join(store.Root(), "control.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE enrollment_codes SET expires_at=? WHERE id=?`, time.Now().Add(-time.Second).UnixNano(), expired.Metadata.ID); err != nil {
		t.Fatal(err)
	}
	_ = database.Close()
	if _, _, err := store.ExchangeEnrollmentCode(ctx, expired.Secret); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("expired error = %v", err)
	}

	limited, err := store.CreateEnrollmentCode(ctx, owner, bootstrap.User.ID, "limited", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	replacement := byte('x')
	if limited.Secret[len(limited.Secret)-1] == replacement {
		replacement = 'y'
	}
	wrong := limited.Secret[:len(limited.Secret)-1] + string(replacement)
	for attempt := 0; attempt < limited.Metadata.MaxAttempts; attempt++ {
		if _, _, err := store.ExchangeEnrollmentCode(ctx, wrong); !errors.Is(err, control.ErrUnauthenticated) {
			t.Fatalf("attempt %d error = %v", attempt+1, err)
		}
	}
	if _, _, err := store.ExchangeEnrollmentCode(ctx, limited.Secret); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("correct code after retry limit error = %v", err)
	}
}

func TestProjectMembershipAndGitHubTrustAreScoped(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{
		UserName: "Owner", ProjectSlug: "one", ProjectName: "One", TokenName: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := store.Authenticate(ctx, bootstrap.Token)
	second, err := store.CreateProject(ctx, owner, "two", "Two")
	if err != nil {
		t.Fatal(err)
	}
	member, err := store.CreateUser(ctx, owner, "Member", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddProjectMember(ctx, owner, bootstrap.Project.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	memberToken, err := store.CreateDeviceToken(ctx, owner, control.CreateDeviceToken{UserID: member.ID, Name: "member"})
	if err != nil {
		t.Fatal(err)
	}
	memberPrincipal, _ := store.Authenticate(ctx, memberToken.Secret)
	if _, err := store.AuthorizeProject(ctx, memberPrincipal, bootstrap.Project.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AuthorizeProject(ctx, memberPrincipal, second.ID); !errors.Is(err, control.ErrForbidden) {
		t.Fatalf("other project authorization error = %v", err)
	}

	trust, err := store.CreateGitHubTrust(ctx, owner, control.GitHubTrust{
		ProjectID: bootstrap.Project.ID, RepositoryOwnerID: "100", RepositoryID: "200",
		WorkflowRef: "flidai/leapview/.github/workflows/ci.yml@refs/heads/*", Ref: "refs/heads/*",
		Environment: "autback", Events: []string{"push", "workflow_dispatch"},
	})
	if err != nil {
		t.Fatal(err)
	}
	validClaims := control.GitHubClaims{
		RepositoryOwnerID: "100", RepositoryID: "200",
		Repository:  "renamed-owner/renamed-repository",
		WorkflowRef: "flidai/leapview/.github/workflows/ci.yml@refs/heads/main", Ref: "refs/heads/main",
		Environment: "autback", EventName: "push",
	}
	matched, err := store.MatchGitHubTrust(ctx, bootstrap.Project.ID, validClaims)
	if err != nil || matched.ID != trust.ID {
		t.Fatalf("matched=%#v err=%v", matched, err)
	}
	nestedBranchClaims := validClaims
	nestedBranchClaims.WorkflowRef = "flidai/leapview/.github/workflows/ci.yml@refs/heads/codex/benchmark"
	nestedBranchClaims.Ref = "refs/heads/codex/benchmark"
	matched, err = store.MatchGitHubTrust(ctx, bootstrap.Project.ID, nestedBranchClaims)
	if err != nil || matched.ID != trust.ID {
		t.Fatalf("nested branch matched=%#v err=%v", matched, err)
	}
	tests := []struct {
		name   string
		mutate func(*control.GitHubClaims)
	}{
		{name: "owner ID", mutate: func(claims *control.GitHubClaims) { claims.RepositoryOwnerID = "999" }},
		{name: "repository ID", mutate: func(claims *control.GitHubClaims) { claims.RepositoryID = "999" }},
		{name: "workflow", mutate: func(claims *control.GitHubClaims) {
			claims.WorkflowRef = "flidai/leapview/.github/workflows/other.yml@refs/heads/main"
		}},
		{name: "ref", mutate: func(claims *control.GitHubClaims) { claims.Ref = "refs/tags/release" }},
		{name: "environment", mutate: func(claims *control.GitHubClaims) { claims.Environment = "production" }},
		{name: "event", mutate: func(claims *control.GitHubClaims) { claims.EventName = "pull_request" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := validClaims
			test.mutate(&claims)
			if _, err := store.MatchGitHubTrust(ctx, bootstrap.Project.ID, claims); !errors.Is(err, control.ErrForbidden) {
				t.Fatalf("mismatched %s error = %v", test.name, err)
			}
		})
	}

	_, err = store.CreateGitHubTrust(ctx, owner, control.GitHubTrust{
		ProjectID: bootstrap.Project.ID, RepositoryOwnerID: "100", RepositoryID: "200",
		WorkflowRef: "flidai/leapview/.github/workflows/ci.yml@refs/pull/*/merge", Ref: "refs/pull/*/merge",
		Events: []string{"pull_request"},
	})
	if err == nil {
		t.Fatal("pull_request trust without an environment gate was accepted")
	}
	if _, err := store.CreateGitHubTrust(ctx, owner, control.GitHubTrust{
		ProjectID: bootstrap.Project.ID, RepositoryOwnerID: "100", RepositoryID: "200",
		WorkflowRef: "flidai/leapview/.github/workflows/ci.yml@refs/pull/*/merge", Ref: "refs/pull/*/merge",
		Environment: "autback-pr", Events: []string{"pull_request"},
	}); err != nil {
		t.Fatalf("protected pull_request trust: %v", err)
	}
}

func openStore(t *testing.T) *controlsqlite.Store {
	t.Helper()
	store, err := controlsqlite.Open(t.TempDir(), []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}
