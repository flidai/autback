package retry

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestDoGrowsJittersCapsAndHonorsElapsedBudget(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	var waits []time.Duration
	attempts := 0
	err := Do(context.Background(), Policy{
		InitialDelay: time.Second, MaxDelay: 4 * time.Second, MaxElapsed: 9 * time.Second,
		Now:    func() time.Time { return now },
		Jitter: func(delay time.Duration) time.Duration { return delay + delay/2 },
		Wait: func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			now = now.Add(delay)
			return nil
		},
	}, func(context.Context) error { attempts++; return errors.New("unavailable") }, func(error) bool { return true })
	if !errors.Is(err, ErrExhausted) {
		t.Fatalf("error = %v, want ErrExhausted", err)
	}
	if attempts != 4 || !reflect.DeepEqual(waits, []time.Duration{1500 * time.Millisecond, 3 * time.Second, 4500 * time.Millisecond}) {
		t.Fatalf("attempts=%d waits=%v", attempts, waits)
	}
}

func TestDoBoundsAttemptsAndPerAttemptDeadline(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), Policy{
		InitialDelay: time.Millisecond, MaxAttempts: 2, AttemptTimeout: time.Millisecond,
		Jitter: func(delay time.Duration) time.Duration { return delay },
		Wait:   func(context.Context, time.Duration) error { return nil },
	}, func(ctx context.Context) error {
		attempts++
		<-ctx.Done()
		return ctx.Err()
	}, func(error) bool { return true })
	if !errors.Is(err, ErrExhausted) || attempts != 2 {
		t.Fatalf("error=%v attempts=%d", err, attempts)
	}
}

func TestDoStopsImmediatelyOnCancellationOrPermanentFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	if err := Do(ctx, Policy{}, func(context.Context) error { attempts++; return nil }, func(error) bool { return true }); !errors.Is(err, context.Canceled) || attempts != 0 {
		t.Fatalf("cancelled error=%v attempts=%d", err, attempts)
	}
	permanent := errors.New("permission denied")
	if err := Do(context.Background(), Policy{}, func(context.Context) error { attempts++; return permanent }, func(error) bool { return false }); !errors.Is(err, permanent) {
		t.Fatalf("permanent error=%v", err)
	}
}
