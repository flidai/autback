package reconciler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/protocol"
)

type Store interface {
	ScheduledJobs(context.Context) ([]control.Job, error)
	Job(context.Context, string) (control.Job, error)
	SyncJob(context.Context, string, protocol.Job) (control.Job, error)
	Operation(context.Context, control.OperationKind, string) (control.Operation, error)
	StaleBuilds(context.Context, time.Time) ([]control.Build, error)
	FinishBuild(context.Context, string, control.BuildStatus, int) (control.Build, error)
}

type Scheduler interface {
	ManagedJobs(context.Context) ([]protocol.Job, error)
	Remove(context.Context, string) error
}

type Dispatcher interface {
	Release(context.Context, control.OperationKind, string) error
}

type Config struct {
	Store             Store
	Scheduler         Scheduler
	Dispatcher        Dispatcher
	ServiceRetention  time.Duration
	AdmissionGrace    time.Duration
	BuildLeaseTimeout time.Duration
	Now               func() time.Time
}

type Reconciler struct {
	config Config
}

func New(config Config) *Reconciler {
	if config.ServiceRetention <= 0 {
		config.ServiceRetention = time.Hour
	}
	if config.AdmissionGrace <= 0 {
		config.AdmissionGrace = 15 * time.Second
	}
	if config.BuildLeaseTimeout <= 0 {
		config.BuildLeaseTimeout = 2 * time.Hour
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Reconciler{config: config}
}

func (r *Reconciler) RunOnce(ctx context.Context) error {
	now := r.config.Now().UTC()
	staleBuilds, err := r.config.Store.StaleBuilds(ctx, now.Add(-r.config.BuildLeaseTimeout))
	if err != nil {
		return fmt.Errorf("list stale builds: %w", err)
	}
	for _, build := range staleBuilds {
		if _, err := r.config.Store.FinishBuild(ctx, build.ID, control.BuildCancelled, 130); err != nil {
			return fmt.Errorf("cancel stale build %s: %w", build.ID, err)
		}
		if r.config.Dispatcher != nil {
			if err := r.config.Dispatcher.Release(ctx, control.OperationBuild, build.ID); err != nil {
				return fmt.Errorf("release stale build %s: %w", build.ID, err)
			}
		}
	}
	scheduled, err := r.config.Store.ScheduledJobs(ctx)
	if err != nil {
		return fmt.Errorf("list scheduled jobs: %w", err)
	}
	managed, err := r.config.Scheduler.ManagedJobs(ctx)
	if err != nil {
		return fmt.Errorf("list managed jobs: %w", err)
	}
	remoteByID := make(map[string]protocol.Job, len(managed))
	for _, job := range managed {
		remoteByID[job.ID] = job
	}
	for _, job := range scheduled {
		remote, ok := remoteByID[job.ID]
		if !ok {
			operation, operationErr := r.config.Store.Operation(ctx, control.OperationJob, job.ID)
			if operationErr != nil && !errors.Is(operationErr, control.ErrNotFound) {
				return fmt.Errorf("read admission lease for job %s: %w", job.ID, operationErr)
			}
			if operationErr == nil && operation.LeasedAt != nil && operation.LeasedAt.After(now.Add(-r.config.AdmissionGrace)) {
				continue
			}
			finished, exitCode := now, 1
			remote = protocol.Job{ID: job.ID, Status: protocol.StatusLost, FinishedAt: &finished, ExitCode: &exitCode, ErrorMessage: "managed Swarm service is missing"}
		}
		stored, err := r.config.Store.SyncJob(ctx, job.ID, remote)
		if err != nil {
			return fmt.Errorf("synchronize job %s: %w", job.ID, err)
		}
		if stored.Status.Terminal() && r.config.Dispatcher != nil {
			if err := r.config.Dispatcher.Release(ctx, control.OperationJob, job.ID); err != nil {
				return fmt.Errorf("release job %s: %w", job.ID, err)
			}
		}
	}

	cutoff := now.Add(-r.config.ServiceRetention)
	for _, remote := range managed {
		stored, err := r.config.Store.Job(ctx, remote.ID)
		if err != nil && !errors.Is(err, control.ErrNotFound) {
			return fmt.Errorf("read managed job %s: %w", remote.ID, err)
		}
		removeAfter := remote.CreatedAt
		if err == nil {
			if !stored.Status.Terminal() {
				continue
			}
			if stored.FinishedAt != nil {
				removeAfter = *stored.FinishedAt
			} else if remote.FinishedAt != nil {
				removeAfter = *remote.FinishedAt
			}
		} else if remote.FinishedAt != nil {
			removeAfter = *remote.FinishedAt
		}
		if removeAfter.IsZero() || removeAfter.After(cutoff) {
			continue
		}
		if err := r.config.Scheduler.Remove(ctx, remote.ID); err != nil {
			return fmt.Errorf("remove managed job %s: %w", remote.ID, err)
		}
	}
	return nil
}
