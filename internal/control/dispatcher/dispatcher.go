package dispatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/flidai/outback/internal/control"
)

type Store interface {
	AcquireNextOperation(context.Context) (*control.Operation, error)
	ReleaseOperation(context.Context, control.OperationKind, string) error
	Job(context.Context, string) (control.Job, error)
	FailJob(context.Context, string, string) (control.Job, error)
}

type Scheduler interface {
	Create(context.Context, control.Job) error
}

type Dispatcher struct {
	store     Store
	scheduler Scheduler
}

func New(store Store, scheduler Scheduler) *Dispatcher {
	return &Dispatcher{store: store, scheduler: scheduler}
}

// RunOnce admits the oldest operation only when the worker is idle. A build is
// admitted by changing its durable state to running; a job additionally creates
// its worker service. Failed job admission is terminal and does not block FIFO.
func (d *Dispatcher) RunOnce(ctx context.Context) error {
	var admissionErrors []error
	for {
		operation, err := d.store.AcquireNextOperation(ctx)
		if err != nil {
			return errors.Join(append(admissionErrors, err)...)
		}
		if operation == nil || operation.Kind == control.OperationBuild {
			return errors.Join(admissionErrors...)
		}
		job, err := d.store.Job(ctx, operation.ID)
		if err == nil {
			err = d.scheduler.Create(ctx, job)
		}
		if err == nil {
			return errors.Join(admissionErrors...)
		}
		admissionErrors = append(admissionErrors, fmt.Errorf("admit job %s: %w", operation.ID, err))
		if _, failErr := d.store.FailJob(ctx, operation.ID, err.Error()); failErr != nil {
			return errors.Join(append(admissionErrors, failErr)...)
		}
		if releaseErr := d.store.ReleaseOperation(ctx, control.OperationJob, operation.ID); releaseErr != nil {
			return errors.Join(append(admissionErrors, releaseErr)...)
		}
	}
}

func (d *Dispatcher) Release(ctx context.Context, kind control.OperationKind, id string) error {
	if err := d.store.ReleaseOperation(ctx, kind, id); err != nil {
		return err
	}
	return d.RunOnce(ctx)
}
