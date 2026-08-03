package capacity

import "testing"

func TestPolicyScalesWorkerThresholds(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	policy := DefaultPolicy()
	thresholds := policy.Thresholds(150 * gib)

	if thresholds.SoftFreeBytes != 30*gib {
		t.Fatalf("soft free bytes = %d, want %d", thresholds.SoftFreeBytes, 30*gib)
	}
	if thresholds.TargetFreeBytes != 37*gib+512*1024*1024 {
		t.Fatalf("target free bytes = %d, want 37.5 GiB", thresholds.TargetFreeBytes)
	}
	if thresholds.HardFreeBytes != 8*gib {
		t.Fatalf("hard free bytes = %d, want %d", thresholds.HardFreeBytes, 8*gib)
	}
}

func TestPolicyUsesAbsoluteFloorsForSmallWorkers(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	thresholds := DefaultPolicy().Thresholds(50 * gib)

	if thresholds.SoftFreeBytes != 20*gib {
		t.Fatalf("soft free bytes = %d, want %d", thresholds.SoftFreeBytes, 20*gib)
	}
	if thresholds.TargetFreeBytes != 25*gib {
		t.Fatalf("target free bytes = %d, want %d", thresholds.TargetFreeBytes, 25*gib)
	}
	if thresholds.HardFreeBytes != 8*gib {
		t.Fatalf("hard free bytes = %d, want %d", thresholds.HardFreeBytes, 8*gib)
	}
}
