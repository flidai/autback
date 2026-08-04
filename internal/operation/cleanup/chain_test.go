package cleanup

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/flidai/autback/internal/control"
)

type preparerFunc func(context.Context, control.Operation) error

func (f preparerFunc) Prepare(ctx context.Context, operation control.Operation) error {
	return f(ctx, operation)
}

func TestLifecyclePreparesInOrderAndStopsBeforeRuntimeOnFailure(t *testing.T) {
	want := errors.New("secret provider unavailable")
	var calls []string
	lifecycle := Lifecycle{Preparers: []Preparer{
		preparerFunc(func(context.Context, control.Operation) error { calls = append(calls, "resources"); return nil }),
		preparerFunc(func(context.Context, control.Operation) error { calls = append(calls, "secrets"); return want }),
		preparerFunc(func(context.Context, control.Operation) error { calls = append(calls, "runtime"); return nil }),
	}}
	if err := lifecycle.Prepare(context.Background(), control.Operation{Kind: control.OperationJob, ID: "job-1"}); !errors.Is(err, want) {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"resources", "secrets"}) {
		t.Fatalf("prepare calls = %#v", calls)
	}
}

func TestLifecycleRunsEveryCleanerInDependencyOrderAndJoinsFailures(t *testing.T) {
	secretErr, resourceErr := errors.New("secret cleanup failed"), errors.New("resource cleanup failed")
	var calls []string
	lifecycle := Lifecycle{Cleaners: []Cleaner{
		CleanerFunc(func(context.Context, control.Operation) error { calls = append(calls, "secrets"); return secretErr }),
		CleanerFunc(func(context.Context, control.Operation) error { calls = append(calls, "resources"); return resourceErr }),
	}}
	err := lifecycle.Cleanup(context.Background(), control.Operation{Kind: control.OperationJob, ID: "job-1"})
	if !errors.Is(err, secretErr) || !errors.Is(err, resourceErr) {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"secrets", "resources"}) {
		t.Fatalf("cleanup calls = %#v", calls)
	}
}
