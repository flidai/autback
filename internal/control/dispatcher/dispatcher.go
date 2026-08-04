package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flidai/autback/internal/control"
	operationcleanup "github.com/flidai/autback/internal/operation/cleanup"
)

var ErrDraining = errors.New("dispatcher is draining")

type Store interface {
	operationcleanup.Store
	AcquireNextOperation(context.Context) (*control.Operation, error)
	ActivateOperation(context.Context, control.OperationKind, string) error
	Job(context.Context, string) (control.Job, error)
	FailJob(context.Context, string, string) (control.Job, error)
}

type Scheduler interface {
	Create(context.Context, control.Job) error
	Cancel(context.Context, string) error
}

type Capacity interface {
	Admit(context.Context, func() error) error
}

type Option func(*Dispatcher)

func WithCapacity(capacity Capacity) Option {
	return func(dispatcher *Dispatcher) { dispatcher.capacity = capacity }
}

func WithAdvanceContext(ctx context.Context) Option {
	return func(dispatcher *Dispatcher) { dispatcher.advanceCtx = ctx }
}

func WithErrorHandler(handler func(error)) Option {
	return func(dispatcher *Dispatcher) { dispatcher.onError = handler }
}

func WithCleaner(cleaner operationcleanup.Cleaner) Option {
	return func(dispatcher *Dispatcher) { dispatcher.cleaner = cleaner }
}

func WithCleanupRetryDelay(delay time.Duration) Option {
	return func(dispatcher *Dispatcher) { dispatcher.cleanupRetryDelay = delay }
}

type Dispatcher struct {
	store     Store
	scheduler Scheduler
	capacity  Capacity
	cleaner   operationcleanup.Cleaner
	cleanups  *operationcleanup.Coordinator

	advanceCtx        context.Context
	onError           func(error)
	cleanupRetryDelay time.Duration
	advanceMu         sync.Mutex
	advancing         bool
	advancePending    bool
	advanceWake       chan struct{}
	advanceWG         sync.WaitGroup
	draining          atomic.Bool
}

func New(store Store, scheduler Scheduler, options ...Option) *Dispatcher {
	dispatcher := &Dispatcher{
		store: store, scheduler: scheduler, advanceCtx: context.Background(), cleanupRetryDelay: 5 * time.Second,
		advanceWake: make(chan struct{}, 1),
	}
	for _, option := range options {
		option(dispatcher)
	}
	dispatcher.cleanups = operationcleanup.New(store, dispatcher.cleaner,
		operationcleanup.WithContext(dispatcher.advanceCtx),
		operationcleanup.WithRetryDelay(dispatcher.cleanupRetryDelay),
		operationcleanup.WithErrorHandler(dispatcher.reportError),
		operationcleanup.WithCompleted(func(control.Operation) { dispatcher.Advance() }),
	)
	return dispatcher
}

// Advance schedules FIFO admission without coupling a terminal operation's
// acknowledgement to capacity reclaim or runtime creation. Concurrent wakeups
// are coalesced; transient failures retry until the server context ends.
func (d *Dispatcher) Advance() {
	if d.draining.Load() {
		return
	}
	d.cleanups.Advance()
	d.advanceMu.Lock()
	if d.draining.Load() {
		d.advanceMu.Unlock()
		return
	}
	d.advancePending = true
	if d.advancing {
		select {
		case d.advanceWake <- struct{}{}:
		default:
		}
		d.advanceMu.Unlock()
		return
	}
	d.advancing = true
	d.advanceWG.Add(1)
	d.advanceMu.Unlock()
	go func() {
		defer d.advanceWG.Done()
		d.advance()
	}()
}

func (d *Dispatcher) Drain() {
	d.draining.Store(true)
	d.cleanups.Drain()
}

// Wait joins admission and cleanup work that started before Drain.
func (d *Dispatcher) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		d.advanceWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return d.cleanups.Wait(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) reportError(err error) {
	if d.onError != nil {
		d.onError(err)
	}
}

func (d *Dispatcher) advance() {
	for {
		d.advanceMu.Lock()
		d.advancePending = false
		d.advanceMu.Unlock()

		if err := d.RunOnce(d.advanceCtx); err != nil {
			if errors.Is(err, ErrDraining) || isContextCancellation(err, d.advanceCtx) {
				d.finishAdvance()
				return
			}
			d.reportError(err)
			timer := time.NewTimer(5 * time.Second)
			select {
			case <-d.advanceCtx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				d.finishAdvance()
				return
			case <-d.advanceWake:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				continue
			case <-timer.C:
				continue
			}
		}

		d.advanceMu.Lock()
		if d.advancePending {
			d.advanceMu.Unlock()
			continue
		}
		d.advancing = false
		d.advanceMu.Unlock()
		return
	}
}

func (d *Dispatcher) finishAdvance() {
	d.advanceMu.Lock()
	d.advancing = false
	d.advanceMu.Unlock()
}

// RunOnce reserves the oldest operation only when the worker is idle. The
// operation becomes active after its runtime is ready. Failed job admission is
// terminal and does not block FIFO.
func (d *Dispatcher) RunOnce(ctx context.Context) error {
	if d.draining.Load() {
		return ErrDraining
	}
	if d.capacity != nil {
		return d.capacity.Admit(ctx, func() error { return d.runOnce(ctx) })
	}
	return d.runOnce(ctx)
}

func (d *Dispatcher) runOnce(ctx context.Context) error {
	if d.draining.Load() {
		return ErrDraining
	}
	var admissionErrors []error
	for {
		operation, err := d.store.AcquireNextOperation(ctx)
		if err != nil {
			return errors.Join(append(admissionErrors, err)...)
		}
		if operation == nil {
			return errors.Join(admissionErrors...)
		}
		if operation.Kind == control.OperationBuild {
			if err := d.store.ActivateOperation(ctx, operation.Kind, operation.ID); err != nil {
				return errors.Join(append(admissionErrors, fmt.Errorf("activate build %s: %w", operation.ID, err))...)
			}
			return errors.Join(admissionErrors...)
		}
		job, err := d.store.Job(ctx, operation.ID)
		if err != nil {
			return errors.Join(append(admissionErrors, fmt.Errorf("read admitted job %s: %w", operation.ID, err))...)
		}
		if err := d.scheduler.Create(ctx, job); err != nil {
			if isContextCancellation(err, ctx) {
				return errors.Join(append(admissionErrors, err)...)
			}
			admissionErrors = append(admissionErrors, fmt.Errorf("admit job %s: %w", operation.ID, err))
			if _, failErr := d.store.FailJob(ctx, operation.ID, err.Error()); failErr != nil {
				return errors.Join(append(admissionErrors, failErr)...)
			}
			if releaseErr := d.cleanups.Request(ctx, control.OperationJob, operation.ID); releaseErr != nil {
				return errors.Join(append(admissionErrors, releaseErr)...)
			}
			return errors.Join(admissionErrors...)
		}
		if err := d.store.ActivateOperation(ctx, operation.Kind, operation.ID); err != nil {
			return errors.Join(append(admissionErrors, fmt.Errorf("activate job %s: %w", operation.ID, err))...)
		}
		job, err = d.store.Job(ctx, operation.ID)
		if err != nil {
			return errors.Join(append(admissionErrors, fmt.Errorf("read active job %s: %w", operation.ID, err))...)
		}
		if job.CancelRequested {
			if err := d.scheduler.Cancel(ctx, operation.ID); err != nil {
				return errors.Join(append(admissionErrors, fmt.Errorf("cancel admitted job %s: %w", operation.ID, err))...)
			}
		}
		return errors.Join(admissionErrors...)
	}
}

func (d *Dispatcher) Release(ctx context.Context, kind control.OperationKind, id string) error {
	return d.cleanups.Request(ctx, kind, id)
}

func isContextCancellation(err error, ctx context.Context) bool {
	return ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}
