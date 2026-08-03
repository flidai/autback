package dispatcher

import (
	"context"
	"errors"
	"fmt"

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

type Dispatcher struct {
	store     Store
	scheduler Scheduler
}

func New(store Store, scheduler Scheduler) *Dispatcher {
	return &Dispatcher{store: store, scheduler: scheduler}
}

// RunOnce reserves the oldest operation only when the worker is idle. The
// operation becomes active after its runtime is ready. Failed job admission is
// terminal and does not block FIFO.
func (d *Dispatcher) RunOnce(ctx context.Context) error {
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
