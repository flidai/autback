package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/flidai/autback/internal/control"
)

type Store interface {
	AcquireNextOperation(context.Context) (*control.Operation, error)
	ActivateOperation(context.Context, control.OperationKind, string) error
	ReleaseOperation(context.Context, control.OperationKind, string) error
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

type Dispatcher struct {
	store     Store
	scheduler Scheduler
	capacity  Capacity

	advanceCtx     context.Context
	onError        func(error)
	advanceMu      sync.Mutex
	advancing      bool
	advancePending bool
}

func New(store Store, scheduler Scheduler, options ...Option) *Dispatcher {
	dispatcher := &Dispatcher{store: store, scheduler: scheduler, advanceCtx: context.Background()}
	for _, option := range options {
		option(dispatcher)
	}
	return dispatcher
}

// Advance schedules FIFO admission without coupling a terminal operation's
// acknowledgement to capacity reclaim or runtime creation. Concurrent wakeups
// are coalesced; transient failures retry until the server context ends.
func (d *Dispatcher) Advance() {
	d.advanceMu.Lock()
	d.advancePending = true
	if d.advancing {
		d.advanceMu.Unlock()
		return
	}
	d.advancing = true
	d.advanceMu.Unlock()
	go d.advance()
}

func (d *Dispatcher) advance() {
	for {
		d.advanceMu.Lock()
		d.advancePending = false
		d.advanceMu.Unlock()

		if err := d.RunOnce(d.advanceCtx); err != nil {
			if d.onError != nil {
				d.onError(err)
			}
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
	if d.capacity != nil {
		return d.capacity.Admit(ctx, func() error { return d.runOnce(ctx) })
	}
	return d.runOnce(ctx)
}

func (d *Dispatcher) runOnce(ctx context.Context) error {
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
			admissionErrors = append(admissionErrors, fmt.Errorf("admit job %s: %w", operation.ID, err))
			if _, failErr := d.store.FailJob(ctx, operation.ID, err.Error()); failErr != nil {
				return errors.Join(append(admissionErrors, failErr)...)
			}
			if releaseErr := d.store.ReleaseOperation(ctx, control.OperationJob, operation.ID); releaseErr != nil {
				return errors.Join(append(admissionErrors, releaseErr)...)
			}
			continue
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
	if err := d.store.ReleaseOperation(ctx, kind, id); err != nil {
		return err
	}
	return d.RunOnce(ctx)
}
