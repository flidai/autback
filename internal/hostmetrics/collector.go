package hostmetrics

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/flidai/autback/internal/control"
)

type Sampler interface {
	Sample(context.Context) (control.ResourceSample, error)
}

type Store interface {
	ActiveResourceScope(context.Context) (control.ResourceScope, bool, error)
	AppendResourceSample(context.Context, control.ResourceSample) error
	CompactResourceSamples(context.Context, time.Time, time.Time) error
}

type CollectorConfig struct {
	Store           Store
	Sampler         Sampler
	Interval        time.Duration
	RawRetention    time.Duration
	RollupRetention time.Duration
	Now             func() time.Time
	OnError         func(error)
}

type Collector struct {
	config         CollectorConfig
	mu             sync.Mutex
	lastCompaction time.Time
	eventsReady    bool
	memoryHigh     uint64
	oom            uint64
	oomKills       uint64
}

func NewCollector(config CollectorConfig) (*Collector, error) {
	if config.Store == nil || config.Sampler == nil {
		return nil, errors.New("resource collector store and sampler are required")
	}
	if config.Interval <= 0 {
		config.Interval = 2 * time.Second
	}
	if config.RawRetention <= 0 {
		config.RawRetention = 14 * 24 * time.Hour
	}
	if config.RollupRetention <= config.RawRetention {
		config.RollupRetention = 180 * 24 * time.Hour
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Collector{config: config}, nil
}

func (c *Collector) CollectOnce(ctx context.Context) error {
	sample, err := c.config.Sampler.Sample(ctx)
	if errors.Is(err, ErrNotReady) {
		return nil
	}
	if err != nil {
		return err
	}
	scope, active, err := c.config.Store.ActiveResourceScope(ctx)
	if err != nil {
		return err
	}
	if active {
		sample.ResourceScope = scope
	}
	c.mu.Lock()
	previousHigh, previousOOM, previousKills, eventsReady := c.memoryHigh, c.oom, c.oomKills, c.eventsReady
	c.memoryHigh, c.oom, c.oomKills, c.eventsReady = sample.MemoryHighEvents, sample.OOMEvents, sample.OOMKills, true
	c.mu.Unlock()
	if !eventsReady {
		sample.MemoryHighEvents, sample.OOMEvents, sample.OOMKills = 0, 0, 0
	} else {
		sample.MemoryHighEvents = counterDelta(sample.MemoryHighEvents, previousHigh)
		sample.OOMEvents = counterDelta(sample.OOMEvents, previousOOM)
		sample.OOMKills = counterDelta(sample.OOMKills, previousKills)
	}
	if err := c.config.Store.AppendResourceSample(ctx, sample); err != nil {
		return err
	}
	now := c.config.Now().UTC()
	c.mu.Lock()
	compact := c.lastCompaction.IsZero() || now.Sub(c.lastCompaction) >= time.Hour
	if compact {
		c.lastCompaction = now
	}
	c.mu.Unlock()
	if compact {
		return c.config.Store.CompactResourceSamples(ctx, now.Add(-c.config.RawRetention), now.Add(-c.config.RollupRetention))
	}
	return nil
}

func counterDelta(current, previous uint64) uint64 {
	if current < previous {
		return current
	}
	return current - previous
}

func (c *Collector) Run(ctx context.Context) {
	c.collect(ctx)
	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *Collector) collect(ctx context.Context) {
	if err := c.CollectOnce(ctx); err != nil && ctx.Err() == nil && c.config.OnError != nil {
		c.config.OnError(err)
	}
}
