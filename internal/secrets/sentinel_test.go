package secrets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	secretstore "github.com/flidai/autback/internal/adapter/secretstore"
	"github.com/flidai/autback/internal/control"
	controlsqlite "github.com/flidai/autback/internal/control/sqlite"
	"github.com/flidai/autback/internal/operation/redact"
	"github.com/flidai/autback/internal/secrets"
)

func TestSentinelNeverEntersControlStateAuditLogsOrBackup(t *testing.T) {
	const sentinel = "autback-sentinel-secret-value-279"
	ctx := context.Background()
	dataRoot, jobsRoot, externalRoot := t.TempDir(), t.TempDir(), t.TempDir()
	store, err := controlsqlite.Open(dataRoot, []byte("test-pepper-that-is-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	projectSecrets := filepath.Join(externalRoot, bootstrap.Project.ID)
	if err := os.MkdirAll(projectSecrets, 0o700); err != nil {
		t.Fatal(err)
	}
	externalValue := filepath.Join(projectSecrets, "registry-token")
	if err := os.WriteFile(externalValue, []byte(sentinel), 0o600); err != nil {
		t.Fatal(err)
	}
	job, _, err := store.CreatePreparedJob(ctx, control.PrepareJob{
		ProjectID: bootstrap.Project.ID, Image: "runner@test", Command: []string{"task", "ci"}, Timeout: time.Minute,
		Secrets: []control.SecretBinding{{Name: "registry-token", Environment: "REGISTRY_TOKEN"}},
	}, control.Idempotency{Key: "secret-sentinel", RequestHash: "secret-sentinel"})
	if err != nil {
		t.Fatal(err)
	}
	manager := secrets.NewManager(secrets.Config{
		JobsRoot: jobsRoot, Store: store, Resolver: secretstore.Directory{Root: externalRoot}, Access: store,
	})
	operation := control.Operation{Kind: control.OperationJob, ID: job.ID}
	if err := manager.Prepare(ctx, operation); err != nil {
		t.Fatal(err)
	}
	runtimeDirectory := filepath.Join(jobsRoot, job.ID, "secrets")
	runtimeValues, err := secrets.LoadRuntime(runtimeDirectory)
	if err != nil {
		t.Fatal(err)
	}
	var durable bytes.Buffer
	writer, err := redact.NewWriter(&durable, runtimeValues.Values)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.Write([]byte("output " + sentinel + "\n"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(durable.String(), sentinel) || !strings.Contains(durable.String(), "[REDACTED]") {
		t.Fatalf("durable output = %q", durable.String())
	}

	principal := control.Principal{Kind: control.PrincipalDevice, UserID: bootstrap.User.ID, Admin: true}
	audit, err := store.ListAuditEvents(ctx, principal, bootstrap.Project.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	encodedAudit, err := json.Marshal(audit)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encodedAudit), sentinel) {
		t.Fatalf("audit contains sentinel: %s", encodedAudit)
	}
	foundAccess := false
	for _, event := range audit {
		if event.Action == "job.secret.access" && event.TargetID == job.ID && event.Metadata["name"] == "registry-token" && event.ActorKind == control.PrincipalSystem {
			foundAccess = true
		}
	}
	if !foundAccess {
		t.Fatalf("secret access audit missing: %#v", audit)
	}

	backup := filepath.Join(t.TempDir(), "control.db")
	if err := store.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(runtimeDirectory, "manifest.json")
	for _, path := range []string{filepath.Join(dataRoot, "control.db"), filepath.Join(dataRoot, "control.db-wal"), backup, manifest} {
		contents, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(contents, []byte(sentinel)) {
			t.Fatalf("sentinel persisted in %s", path)
		}
	}
	if err := manager.Cleanup(ctx, operation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(runtimeDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("operation-scoped material remains: %v", err)
	}
	if contents, err := os.ReadFile(externalValue); err != nil || string(contents) != sentinel {
		t.Fatalf("external source changed during cleanup: %q, %v", contents, err)
	}
}
