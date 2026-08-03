package capacity

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Trigger string

const (
	TriggerAdmission Trigger = "admission"
	TriggerPeriodic  Trigger = "periodic"
	TriggerPressure  Trigger = "pressure"
	TriggerManual    Trigger = "manual"
)

type Snapshot struct {
	TotalBytes  uint64    `json:"total_bytes"`
	FreeBytes   uint64    `json:"free_bytes"`
	TotalInodes uint64    `json:"total_inodes"`
	FreeInodes  uint64    `json:"free_inodes"`
	ObservedAt  time.Time `json:"observed_at"`
}

type ReclaimRequest struct {
	Trigger         Trigger
	Pressure        bool
	TargetFreeBytes uint64
	JobRetention    time.Duration
	CacheHighBytes  uint64
	CacheLowBytes   uint64
	NormalObjectAge time.Duration
}

type ReclaimReport struct {
	ReclaimedBytes uint64   `json:"reclaimed_bytes"`
	RemovedJobs    int      `json:"removed_jobs"`
	RemovedCaches  int      `json:"removed_caches"`
	Commands       []string `json:"commands,omitempty"`
}

type Backend interface {
	Snapshot(context.Context) (Snapshot, error)
	Reclaim(context.Context, ReclaimRequest) (ReclaimReport, error)
	Emergency(context.Context) error
}

type ResourceExhaustedError struct {
	FreeBytes      uint64
	RequiredBytes  uint64
	FreeInodes     uint64
	RequiredInodes uint64
}

func (e *ResourceExhaustedError) Error() string {
	return fmt.Sprintf("worker capacity exhausted: %d bytes and %d inodes free; require %d bytes and %d inodes", e.FreeBytes, e.FreeInodes, e.RequiredBytes, e.RequiredInodes)
}

type Status struct {
	State       string        `json:"state"`
	Trigger     Trigger       `json:"trigger"`
	Before      Snapshot      `json:"before"`
	After       Snapshot      `json:"after"`
	Thresholds  Thresholds    `json:"thresholds"`
	Reclaim     ReclaimReport `json:"reclaim"`
	Error       string        `json:"error,omitempty"`
	CompletedAt time.Time     `json:"completed_at"`
}

type Controller struct {
	policy       Policy
	backend      Backend
	mu           sync.Mutex
	status       Status
	lastPressure time.Time
}

func New(policy Policy, backend Backend) *Controller {
	return &Controller{policy: policy, backend: backend}
}

func (c *Controller) Ensure(ctx context.Context) error {
	_, err := c.run(ctx, TriggerAdmission)
	return err
}

func (c *Controller) Maintain(ctx context.Context, trigger Trigger) (Status, error) {
	return c.run(ctx, trigger)
}

func (c *Controller) Check(ctx context.Context) error {
	snapshot, err := c.backend.Snapshot(ctx)
	if err != nil {
		return err
	}
	thresholds := c.policy.Thresholds(snapshot.TotalBytes)
	if snapshot.FreeBytes < thresholds.HardFreeBytes || inodePressure(snapshot, c.policy.MinimumFreeInodes) {
		return capacityError(snapshot, thresholds, c.policy.MinimumFreeInodes)
	}
	return nil
}

func (c *Controller) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *Controller) run(ctx context.Context, trigger Trigger) (Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	before, err := c.backend.Snapshot(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("measure worker capacity: %w", err)
	}
	thresholds := c.policy.Thresholds(before.TotalBytes)
	pressure := before.FreeBytes < thresholds.SoftFreeBytes || inodePressure(before, c.policy.MinimumFreeInodes)
	status := Status{State: "healthy", Trigger: trigger, Before: before, After: before, Thresholds: thresholds}
	if trigger == TriggerPressure {
		if !pressure {
			status.CompletedAt = time.Now().UTC()
			c.status = status
			return status, nil
		}
		now := time.Now().UTC()
		if c.policy.PressureThrottle > 0 && now.Sub(c.lastPressure) < c.policy.PressureThrottle {
			status.State = "reclaiming"
			status.CompletedAt = now
			c.status = status
			return status, nil
		}
		c.lastPressure = now
	}
	if pressure {
		status.State = "reclaiming"
	}
	if pressure || trigger != TriggerAdmission {
		status.Reclaim, err = c.backend.Reclaim(ctx, ReclaimRequest{
			Trigger: trigger, Pressure: pressure, TargetFreeBytes: thresholds.TargetFreeBytes,
			JobRetention: c.policy.JobRetention, CacheHighBytes: before.TotalBytes * c.policy.CacheHighPercent / 100,
			CacheLowBytes: before.TotalBytes * c.policy.CacheLowPercent / 100, NormalObjectAge: c.policy.NormalObjectAge,
		})
		if err != nil {
			status.State, status.Error = "blocked", err.Error()
			status.CompletedAt = time.Now().UTC()
			c.status = status
			return status, fmt.Errorf("reclaim worker capacity: %w", err)
		}
		status.After, err = c.backend.Snapshot(ctx)
		if err != nil {
			return Status{}, fmt.Errorf("remeasure worker capacity: %w", err)
		}
	}

	if status.After.FreeBytes < thresholds.HardFreeBytes || inodePressure(status.After, c.policy.MinimumFreeInodes) {
		status.State = "emergency"
		if err := c.backend.Emergency(ctx); err != nil {
			status.Error = err.Error()
		} else {
			report, reclaimErr := c.backend.Reclaim(ctx, ReclaimRequest{
				Trigger: trigger, Pressure: true, TargetFreeBytes: thresholds.TargetFreeBytes,
				JobRetention: c.policy.JobRetention, CacheHighBytes: before.TotalBytes * c.policy.CacheHighPercent / 100,
				CacheLowBytes: before.TotalBytes * c.policy.CacheLowPercent / 100, NormalObjectAge: c.policy.NormalObjectAge,
			})
			status.Reclaim.ReclaimedBytes += report.ReclaimedBytes
			status.Reclaim.RemovedJobs += report.RemovedJobs
			status.Reclaim.RemovedCaches += report.RemovedCaches
			status.Reclaim.Commands = append(status.Reclaim.Commands, report.Commands...)
			if reclaimErr != nil {
				status.Error = reclaimErr.Error()
			}
			status.After, _ = c.backend.Snapshot(ctx)
		}
	}

	if status.After.FreeBytes < thresholds.SoftFreeBytes || inodePressure(status.After, c.policy.MinimumFreeInodes) {
		status.State = "blocked"
		err = capacityError(status.After, thresholds, c.policy.MinimumFreeInodes)
		status.Error = err.Error()
	} else {
		status.State = "healthy"
	}
	status.CompletedAt = time.Now().UTC()
	c.status = status
	return status, err
}

func inodePressure(snapshot Snapshot, minimumPercent uint64) bool {
	return snapshot.TotalInodes > 0 && snapshot.FreeInodes*100 < snapshot.TotalInodes*minimumPercent
}

func capacityError(snapshot Snapshot, thresholds Thresholds, minimumInodes uint64) error {
	requiredInodes := snapshot.TotalInodes * minimumInodes / 100
	return &ResourceExhaustedError{FreeBytes: snapshot.FreeBytes, RequiredBytes: thresholds.SoftFreeBytes, FreeInodes: snapshot.FreeInodes, RequiredInodes: requiredInodes}
}
