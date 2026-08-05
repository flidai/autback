// Package retry provides the one bounded, cancellation-aware retry policy used
// by Autback's transient recovery loops.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

var ErrExhausted = errors.New("retry budget exhausted")

type Policy struct {
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	MaxAttempts    int
	MaxElapsed     time.Duration
	AttemptTimeout time.Duration
	Now            func() time.Time
	Wait           func(context.Context, time.Duration) error
	Jitter         func(time.Duration) time.Duration
}

type Operation func(context.Context) error
type Retryable func(error) bool

func Do(ctx context.Context, policy Policy, operation Operation, retryable Retryable) error {
	if operation == nil || retryable == nil {
		return errors.New("retry operation and classifier are required")
	}
	policy = normalize(policy)
	started := policy.Now()
	delay := policy.InitialDelay
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		attemptCtx, cancel := attemptContext(ctx, policy.AttemptTimeout)
		err := operation(attemptCtx)
		cancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !retryable(err) {
			return err
		}
		if attempt >= policy.MaxAttempts {
			return errors.Join(fmt.Errorf("%w after %d attempts", ErrExhausted, attempt), err)
		}
		remaining := policy.MaxElapsed - policy.Now().Sub(started)
		if remaining <= 0 {
			return errors.Join(fmt.Errorf("%w after %s", ErrExhausted, policy.MaxElapsed), err)
		}
		wait := policy.Jitter(delay)
		if wait <= 0 {
			wait = delay
		}
		if wait > remaining {
			wait = remaining
		}
		if waitErr := policy.Wait(ctx, wait); waitErr != nil {
			return errors.Join(err, waitErr)
		}
		if delay < policy.MaxDelay {
			delay *= 2
			if delay > policy.MaxDelay {
				delay = policy.MaxDelay
			}
		}
	}
}

func normalize(policy Policy) Policy {
	if policy.InitialDelay <= 0 {
		policy.InitialDelay = 250 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 5 * time.Second
	}
	if policy.MaxDelay < policy.InitialDelay {
		policy.MaxDelay = policy.InitialDelay
	}
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 5
	}
	if policy.MaxElapsed <= 0 {
		policy.MaxElapsed = 30 * time.Second
	}
	if policy.Now == nil {
		policy.Now = time.Now
	}
	if policy.Wait == nil {
		policy.Wait = wait
	}
	if policy.Jitter == nil {
		policy.Jitter = func(delay time.Duration) time.Duration {
			return time.Duration(float64(delay) * (.75 + rand.Float64()*.5))
		}
	}
	return policy
}

func attemptContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
