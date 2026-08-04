package secrets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/flidai/autback/internal/control"
)

type staticJobStore struct{ job control.Job }

func (s staticJobStore) Job(context.Context, string) (control.Job, error) { return s.job, nil }

type rotatingResolver struct {
	values map[string][]byte
}

func (r *rotatingResolver) Resolve(_ context.Context, _, name string) ([]byte, error) {
	value, ok := r.values[name]
	if !ok {
		return nil, ErrRevoked
	}
	return append([]byte(nil), value...), nil
}

type accessRecord struct{ project, job, name string }
type accessRecorder struct{ records []accessRecord }

func (r *accessRecorder) RecordSecretAccess(_ context.Context, project, job, name string) error {
	r.records = append(r.records, accessRecord{project, job, name})
	return nil
}

func TestManagerMaterializesSnapshotWithoutValueInManifestAndCleansIt(t *testing.T) {
	root := t.TempDir()
	resolver := &rotatingResolver{values: map[string][]byte{"registry-token": []byte("sentinel-one"), "signing-key": []byte("sentinel-two")}}
	audit := &accessRecorder{}
	job := control.Job{ID: "job-1", ProjectID: "project-1", Secrets: []control.SecretBinding{
		{Name: "registry-token", Environment: "REGISTRY_TOKEN"},
		{Name: "signing-key", File: "/run/secrets/signing-key"},
	}}
	manager := NewManager(Config{JobsRoot: root, Store: staticJobStore{job}, Resolver: resolver, Access: audit})
	operation := control.Operation{Kind: control.OperationJob, ID: job.ID}
	if err := manager.Prepare(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, job.ID, "secrets")
	runtime, err := LoadRuntime(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runtime.Values, []string{"sentinel-one", "sentinel-two"}) || !reflect.DeepEqual(runtime.Environment, []string{"REGISTRY_TOKEN=sentinel-one"}) {
		t.Fatalf("runtime = %#v", runtime)
	}
	manifest, err := os.ReadFile(filepath.Join(directory, manifestName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifest), "sentinel-") {
		t.Fatalf("manifest contains secret value: %s", manifest)
	}
	if got := audit.records; !reflect.DeepEqual(got, []accessRecord{{"project-1", "job-1", "registry-token"}, {"project-1", "job-1", "signing-key"}}) {
		t.Fatalf("access records = %#v", got)
	}
	resolver.values["registry-token"] = []byte("rotated")
	runtime, err = LoadRuntime(directory)
	if err != nil || runtime.Values[0] != "sentinel-one" {
		t.Fatalf("running snapshot changed after rotation: %#v, %v", runtime, err)
	}
	if err := manager.Cleanup(context.Background(), operation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("secret directory remains after cleanup: %v", err)
	}
}

func TestManagerTreatsRevokedReferenceAsNamesOnlyPermanentFailure(t *testing.T) {
	job := control.Job{ID: "job-1", ProjectID: "project-1", Secrets: []control.SecretBinding{{Name: "revoked", Environment: "TOKEN"}}}
	manager := NewManager(Config{JobsRoot: t.TempDir(), Store: staticJobStore{job}, Resolver: &rotatingResolver{values: map[string][]byte{}}})
	err := manager.Prepare(context.Background(), control.Operation{Kind: control.OperationJob, ID: job.ID})
	var resolution *ResolutionError
	if !errors.As(err, &resolution) || !resolution.Permanent() || !errors.Is(err, ErrRevoked) {
		t.Fatalf("Prepare() error = %v", err)
	}
	if strings.Contains(err.Error(), "sentinel") {
		t.Fatalf("error leaked secret material: %v", err)
	}
}

func TestManagerSanitizesTransientProviderFailureAndAllowsRetry(t *testing.T) {
	job := control.Job{ID: "job-1", ProjectID: "project-1", Secrets: []control.SecretBinding{{Name: "token", Environment: "TOKEN"}}}
	manager := NewManager(Config{JobsRoot: t.TempDir(), Store: staticJobStore{job}, Resolver: resolverFunc(func(context.Context, string, string) ([]byte, error) {
		return nil, errors.New("provider failed near sentinel-secret-value")
	})})
	err := manager.Prepare(context.Background(), control.Operation{Kind: control.OperationJob, ID: job.ID})
	var resolution *ResolutionError
	if !errors.As(err, &resolution) || resolution.Permanent() {
		t.Fatalf("Prepare() error = %v", err)
	}
	if strings.Contains(err.Error(), "sentinel-secret-value") {
		t.Fatalf("provider error leaked through boundary: %v", err)
	}
}

type resolverFunc func(context.Context, string, string) ([]byte, error)

func (f resolverFunc) Resolve(ctx context.Context, project, name string) ([]byte, error) {
	return f(ctx, project, name)
}

func TestLoadRuntimeRejectsEscapingManifestSource(t *testing.T) {
	directory := t.TempDir()
	payload := `{"bindings":[{"name":"token","source":"../token","environment":"TOKEN"}]}`
	if err := os.WriteFile(filepath.Join(directory, manifestName), []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRuntime(directory); err == nil {
		t.Fatal("LoadRuntime accepted an escaping source")
	}
}
