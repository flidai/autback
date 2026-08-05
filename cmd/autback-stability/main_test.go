package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFaultMatrixCoversEveryStabilityClaim(t *testing.T) {
	want := []string{
		"cas-restart-transfer", "buildkit-restart", "docker-daemon-loss", "swarm-node-drain",
		"server-sigkill-phases", "disk-and-inode-pressure", "credential-rotation",
		"partial-cleanup-restart", "term-resistant-process-tree", "memory-and-pid-exhaustion",
	}
	var got []string
	for _, scenario := range scenarios("fast") {
		got = append(got, scenario.ID)
		if scenario.Timeout <= 0 || len(scenario.Invocations) == 0 || scenario.Phase == "" || scenario.Fault == "" {
			t.Fatalf("incomplete scenario: %#v", scenario)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scenario IDs = %#v, want %#v", got, want)
	}
	if len(scenarios("full")) <= len(got) {
		t.Fatal("full matrix does not add privileged/resource proofs")
	}
}

func TestRunMatrixRetainsBoundedDiagnosticArtifactsAndContinues(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{errors: []error{errors.New("injected failure"), nil}}
	matrix := []scenario{{
		ID: "fault", Phase: "cleanup", Fault: "daemon restart", Timeout: time.Second,
		Invocations: []invocation{{Package: "./one", Pattern: "TestOne"}, {Package: "./two", Pattern: "TestTwo"}},
	}}
	err := runMatrix(context.Background(), runner, matrix, runConfig{Artifacts: root, Seed: 42, Now: func() time.Time {
		return time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	}})
	if err == nil || runner.calls != 2 {
		t.Fatalf("error=%v calls=%d", err, runner.calls)
	}
	data, readErr := os.ReadFile(filepath.Join(root, "manifest.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	var manifest manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Seed != 42 || len(manifest.Scenarios) != 1 || manifest.Scenarios[0].Status != "failed" || manifest.Scenarios[0].Phase != "cleanup" {
		t.Fatalf("manifest = %#v", manifest)
	}
	logData, err := os.ReadFile(filepath.Join(root, "fault.log"))
	if err != nil || len(logData) == 0 {
		t.Fatalf("log error=%v content=%q", err, logData)
	}
}

func TestRunMatrixRecordsExplicitSkip(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{errors: []error{nil}, outputs: []string{"--- SKIP: TestDocker (0.00s)\n"}}
	matrix := []scenario{{ID: "docker", Phase: "cleanup", Fault: "daemon unavailable", Timeout: time.Second, Invocations: []invocation{{Package: "./docker", Pattern: "TestDocker"}}}}
	if err := runMatrix(context.Background(), runner, matrix, runConfig{Artifacts: root, Seed: 1}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report manifest
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Scenarios) != 1 || report.Scenarios[0].Status != "skipped" {
		t.Fatalf("manifest = %#v", report)
	}
}

type fakeRunner struct {
	calls   int
	errors  []error
	outputs []string
}

func (f *fakeRunner) Run(_ context.Context, _ []string, _ []string, output *os.File) error {
	f.calls++
	text := "FAULT_PHASE observed\n"
	if len(f.outputs) > 0 {
		text, f.outputs = f.outputs[0], f.outputs[1:]
	}
	_, _ = output.WriteString(text)
	err := f.errors[0]
	f.errors = f.errors[1:]
	return err
}
