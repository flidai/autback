package cleanup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/flidai/autback/internal/control"
)

type Store interface {
	BeginOperationCleanup(context.Context, control.OperationKind, string) error
	ClaimOperationCleanup(context.Context) (*control.Operation, error)
	RecordOperationCleanupFailure(context.Context, control.OperationKind, string, string) error
	CompleteOperationCleanup(context.Context, control.OperationKind, string) error
}

type Cleaner interface {
	Cleanup(context.Context, control.Operation) error
}

type CleanerFunc func(context.Context, control.Operation) error

func (f CleanerFunc) Cleanup(ctx context.Context, operation control.Operation) error {
	return f(ctx, operation)
}

type Option func(*Coordinator)

func WithContext(ctx context.Context) Option {
	return func(coordinator *Coordinator) { coordinator.ctx = ctx }
}

func WithRetryDelay(delay time.Duration) Option {
	return func(coordinator *Coordinator) { coordinator.retryDelay = delay }
}

func WithErrorHandler(handler func(error)) Option {
	return func(coordinator *Coordinator) { coordinator.onError = handler }
}

func WithCompleted(handler func(control.Operation)) Option {
	return func(coordinator *Coordinator) { coordinator.onCompleted = handler }
}

type Coordinator struct {
	store   Store
	cleaner Cleaner

	ctx         context.Context
	retryDelay  time.Duration
	onError     func(error)
	onCompleted func(control.Operation)

	mu      sync.Mutex
	running bool
	pending bool
}

func New(store Store, cleaner Cleaner, options ...Option) *Coordinator {
	if cleaner == nil {
		cleaner = CleanerFunc(func(context.Context, control.Operation) error { return nil })
	}
	coordinator := &Coordinator{
		store: store, cleaner: cleaner, ctx: context.Background(), retryDelay: 5 * time.Second,
	}
	for _, option := range options {
		option(coordinator)
	}
	return coordinator
}

// Request commits the operation's terminalizing state using the caller's
// context, then schedules teardown on the coordinator context. The caller does
// not wait for Docker or other runtime cleanup.
func (c *Coordinator) Request(ctx context.Context, kind control.OperationKind, id string) error {
	if err := c.store.BeginOperationCleanup(ctx, kind, id); err != nil {
		return err
	}
	c.Advance()
	return nil
}

// Advance coalesces cleanup wakeups. Failures are durably recorded and retried
// until the coordinator context ends.
func (c *Coordinator) Advance() {
	c.mu.Lock()
	c.pending = true
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()
	go c.advance()
}

func (c *Coordinator) advance() {
	for {
		c.mu.Lock()
		c.pending = false
		c.mu.Unlock()

		claimed, err := c.RunOnce(c.ctx)
		if err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			if !wait(c.ctx, c.retryDelay) {
				c.finish()
				return
			}
			continue
		}
		if claimed {
			continue
		}

		c.mu.Lock()
		if c.pending {
			c.mu.Unlock()
			continue
		}
		c.running = false
		c.mu.Unlock()
		return
	}
}

// RunOnce claims and converges at most one durable cleanup record. A claimed
// cleanup may be a retry after process restart, so Cleaner must be idempotent.
func (c *Coordinator) RunOnce(ctx context.Context) (bool, error) {
	operation, err := c.store.ClaimOperationCleanup(ctx)
	if err != nil || operation == nil {
		return false, err
	}
	if err := c.cleaner.Cleanup(ctx, *operation); err != nil {
		recordErr := c.store.RecordOperationCleanupFailure(ctx, operation.Kind, operation.ID, err.Error())
		return true, errors.Join(fmt.Errorf("clean %s %s: %w", operation.Kind, operation.ID, err), recordErr)
	}
	if err := c.store.CompleteOperationCleanup(ctx, operation.Kind, operation.ID); err != nil {
		return true, fmt.Errorf("complete cleanup for %s %s: %w", operation.Kind, operation.ID, err)
	}
	operation.State = control.OperationReleased
	operation.CleanupError = ""
	if c.onCompleted != nil {
		c.onCompleted(*operation)
	}
	return true, nil
}

func (c *Coordinator) finish() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
}

func wait(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		delay = time.Millisecond
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
