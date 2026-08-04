package buildkit

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestRunWithRunnerRetriesBoundedBuilderRemovalAfterCancelledBuild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &recordingRunner{onRun: func(call int, command []string, stderr io.Writer) error {
		switch call {
		case 1:
			return nil
		case 2:
			cancel()
			return context.Canceled
		case 3, 4:
			if command[1] != "rm" {
				t.Fatalf("cleanup command = %#v", command)
			}
			return errors.New("daemon temporarily unavailable")
		default:
			return nil
		}
	}}

	code, err := runWithRunner(ctx, runner, "docker", "tcp://builder:1234", "builder", ".", []string{"."}, TLS{}, io.Discard, io.Discard, removePolicy{
		Timeout: time.Second, Attempts: 3, Wait: func(context.Context, time.Duration) error { return nil },
	})
	if code != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("Run = (%d, %v), want cancelled build", code, err)
	}
	if len(runner.calls) != 5 {
		t.Fatalf("calls = %#v, want create, build, and three cleanup attempts", runner.calls)
	}
	for _, call := range runner.calls[2:] {
		if call.contextErr != nil || !call.hasDeadline {
			t.Fatalf("cleanup context = err %v deadline %v", call.contextErr, call.hasDeadline)
		}
	}
}

func TestRunWithRunnerTreatsMissingBuilderAsCleaned(t *testing.T) {
	runner := &recordingRunner{onRun: func(call int, _ []string, stderr io.Writer) error {
		if call == 3 {
			_, _ = io.WriteString(stderr, "ERROR: no builder named missing found")
			return errors.New("exit status 1")
		}
		return nil
	}}
	code, err := runWithRunner(context.Background(), runner, "docker", "tcp://builder:1234", "missing", ".", []string{"."}, TLS{}, io.Discard, io.Discard, removePolicy{
		Timeout: time.Second, Attempts: 3, Wait: func(context.Context, time.Duration) error { return nil },
	})
	if code != 0 || err != nil {
		t.Fatalf("Run = (%d, %v), want success", code, err)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("cleanup retried not-found: %#v", runner.calls)
	}
}

func TestRunWithRunnerReturnsBuilderCleanupFailure(t *testing.T) {
	want := errors.New("Docker unavailable")
	runner := &recordingRunner{onRun: func(call int, _ []string, _ io.Writer) error {
		if call >= 3 {
			return want
		}
		return nil
	}}
	code, err := runWithRunner(context.Background(), runner, "docker", "tcp://builder:1234", "leaked", ".", []string{"."}, TLS{}, io.Discard, io.Discard, removePolicy{
		Timeout: time.Second, Attempts: 2, Wait: func(context.Context, time.Duration) error { return nil },
	})
	if code != 0 || !errors.Is(err, want) {
		t.Fatalf("Run = (%d, %v), want cleanup error", code, err)
	}
	if len(runner.calls) != 4 {
		t.Fatalf("calls = %#v, want two cleanup attempts", runner.calls)
	}
}

type recordedCall struct {
	command     []string
	contextErr  error
	hasDeadline bool
}

type recordingRunner struct {
	calls []recordedCall
	onRun func(int, []string, io.Writer) error
}

func (r *recordingRunner) Run(ctx context.Context, _ string, arguments []string, _ string, _, stderr io.Writer) error {
	_, hasDeadline := ctx.Deadline()
	command := append([]string(nil), arguments...)
	r.calls = append(r.calls, recordedCall{command: command, contextErr: ctx.Err(), hasDeadline: hasDeadline})
	return r.onRun(len(r.calls), command, stderr)
}
