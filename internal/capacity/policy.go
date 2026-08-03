package capacity

import "time"

const gib = uint64(1024 * 1024 * 1024)

type Policy struct {
	SoftFreePercent      uint64
	SoftFreeMinimumBytes uint64
	HardFreePercent      uint64
	HardFreeMinimumBytes uint64
	SweepPercent         uint64
	SweepMinimumBytes    uint64
	MinimumFreeInodes    uint64
	JobRetention         time.Duration
	CacheHighPercent     uint64
	CacheLowPercent      uint64
	NormalObjectAge      time.Duration
	PressureThrottle     time.Duration
}

type Thresholds struct {
	SoftFreeBytes   uint64
	TargetFreeBytes uint64
	HardFreeBytes   uint64
}

func DefaultPolicy() Policy {
	return Policy{
		SoftFreePercent: 20, SoftFreeMinimumBytes: 20 * gib,
		HardFreePercent: 5, HardFreeMinimumBytes: 8 * gib,
		SweepPercent: 5, SweepMinimumBytes: 5 * gib,
		MinimumFreeInodes: 10,
		JobRetention:      7 * 24 * time.Hour,
		CacheHighPercent:  10, CacheLowPercent: 8,
		NormalObjectAge:  24 * time.Hour,
		PressureThrottle: 30 * time.Second,
	}
}

func (p Policy) Thresholds(total uint64) Thresholds {
	soft := max(total*p.SoftFreePercent/100, p.SoftFreeMinimumBytes)
	hard := max(total*p.HardFreePercent/100, p.HardFreeMinimumBytes)
	sweep := max(total*p.SweepPercent/100, p.SweepMinimumBytes)
	return Thresholds{SoftFreeBytes: soft, TargetFreeBytes: soft + sweep, HardFreeBytes: hard}
}
