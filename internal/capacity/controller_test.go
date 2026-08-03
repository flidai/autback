package capacity

import (
	"context"
	"errors"
	"testing"
)

func TestEnsureDoesNotCollectHealthyWorker(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{snapshot: Snapshot{TotalBytes: 150 * gib, FreeBytes: 40 * gib, TotalInodes: 1000, FreeInodes: 500}}
	controller := New(DefaultPolicy(), backend)

	if err := controller.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if backend.reclaims != 0 {
		t.Fatalf("reclaims = %d, want 0", backend.reclaims)
	}
}

func TestEnsureCollectsPastSoftFloor(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{
		snapshot:      Snapshot{TotalBytes: 150 * gib, FreeBytes: 25 * gib, TotalInodes: 1000, FreeInodes: 500},
		freeAfterGC:   38 * gib,
		inodesAfterGC: 500,
	}
	controller := New(DefaultPolicy(), backend)

	if err := controller.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if backend.reclaims != 1 {
		t.Fatalf("reclaims = %d, want 1", backend.reclaims)
	}
	if backend.lastTarget != 37*gib+512*1024*1024 {
		t.Fatalf("target = %d, want 37.5 GiB", backend.lastTarget)
	}
}

func TestEnsureReturnsResourceExhaustedWhenReclaimCannotReachFloor(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{
		snapshot:      Snapshot{TotalBytes: 150 * gib, FreeBytes: 25 * gib, TotalInodes: 1000, FreeInodes: 500},
		freeAfterGC:   27 * gib,
		inodesAfterGC: 500,
	}
	controller := New(DefaultPolicy(), backend)

	err := controller.Ensure(context.Background())
	var exhausted *ResourceExhaustedError
	if !errors.As(err, &exhausted) {
		t.Fatalf("error = %v, want ResourceExhaustedError", err)
	}
	if exhausted.FreeBytes != 27*gib || exhausted.RequiredBytes != 30*gib {
		t.Fatalf("capacity error = %#v", exhausted)
	}
}

func TestEnsureUsesInodePressure(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{
		snapshot:      Snapshot{TotalBytes: 150 * gib, FreeBytes: 80 * gib, TotalInodes: 1000, FreeInodes: 40},
		freeAfterGC:   80 * gib,
		inodesAfterGC: 200,
	}
	controller := New(DefaultPolicy(), backend)

	if err := controller.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if backend.reclaims != 1 {
		t.Fatalf("reclaims = %d, want 1", backend.reclaims)
	}
}

func TestEnsureInvokesEmergencyReclaimBelowHardFloor(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{
		snapshot:           Snapshot{TotalBytes: 150 * gib, FreeBytes: 6 * gib, TotalInodes: 1000, FreeInodes: 500},
		freeAfterGC:        6 * gib,
		inodesAfterGC:      500,
		freeAfterEmergency: 40 * gib,
	}
	controller := New(DefaultPolicy(), backend)

	if err := controller.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if backend.emergencies != 1 {
		t.Fatalf("emergencies = %d, want 1", backend.emergencies)
	}
	if backend.reclaims != 2 {
		t.Fatalf("reclaims = %d, want 2", backend.reclaims)
	}
}

type fakeBackend struct {
	snapshot           Snapshot
	freeAfterGC        uint64
	inodesAfterGC      uint64
	freeAfterEmergency uint64
	reclaims           int
	emergencies        int
	lastTarget         uint64
}

func (f *fakeBackend) Snapshot(context.Context) (Snapshot, error) { return f.snapshot, nil }

func (f *fakeBackend) Reclaim(_ context.Context, request ReclaimRequest) (ReclaimReport, error) {
	f.reclaims++
	f.lastTarget = request.TargetFreeBytes
	before := f.snapshot.FreeBytes
	if f.freeAfterGC != 0 {
		f.snapshot.FreeBytes = f.freeAfterGC
	}
	if f.inodesAfterGC != 0 {
		f.snapshot.FreeInodes = f.inodesAfterGC
	}
	return ReclaimReport{ReclaimedBytes: f.snapshot.FreeBytes - before}, nil
}

func (f *fakeBackend) Emergency(context.Context) error {
	f.emergencies++
	if f.freeAfterEmergency != 0 {
		f.freeAfterGC = f.freeAfterEmergency
	}
	return nil
}
