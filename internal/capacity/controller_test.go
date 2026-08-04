package capacity

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
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

func TestMaintainDefersSoftReclaimWhileOperationIsActive(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{
		snapshot: Snapshot{TotalBytes: 150 * gib, FreeBytes: 25 * gib, TotalInodes: 1000, FreeInodes: 500},
		busy:     true,
	}
	controller := New(DefaultPolicy(), backend)

	status, err := controller.Maintain(context.Background(), TriggerPressure)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "deferred" {
		t.Fatalf("state = %q, want deferred", status.State)
	}
	if backend.reclaims != 0 {
		t.Fatalf("reclaims = %d, want 0", backend.reclaims)
	}
}

func TestMaintainStopsActiveOperationBeforeHardReclaim(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{
		snapshot:           Snapshot{TotalBytes: 150 * gib, FreeBytes: 6 * gib, TotalInodes: 1000, FreeInodes: 500},
		busy:               true,
		freeAfterEmergency: 40 * gib,
	}
	controller := New(DefaultPolicy(), backend)

	if _, err := controller.Maintain(context.Background(), TriggerPressure); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(backend.events, []string{"emergency", "reclaim"}) {
		t.Fatalf("events = %#v, want emergency before reclaim", backend.events)
	}
}

func TestAdmissionCannotRaceMaintenance(t *testing.T) {
	const gib = uint64(1024 * 1024 * 1024)
	backend := &fakeBackend{snapshot: Snapshot{TotalBytes: 150 * gib, FreeBytes: 40 * gib, TotalInodes: 1000, FreeInodes: 500}}
	controller := New(DefaultPolicy(), backend)
	admitted := make(chan struct{})
	release := make(chan struct{})
	admissionDone := make(chan error, 1)
	go func() {
		admissionDone <- controller.Admit(context.Background(), func() error {
			close(admitted)
			<-release
			return nil
		})
	}()
	<-admitted
	maintenanceDone := make(chan error, 1)
	go func() {
		_, err := controller.Maintain(context.Background(), TriggerPeriodic)
		maintenanceDone <- err
	}()

	select {
	case err := <-maintenanceDone:
		t.Fatalf("maintenance completed during admission: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-admissionDone; err != nil {
		t.Fatal(err)
	}
	if err := <-maintenanceDone; err != nil {
		t.Fatal(err)
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
	busy               bool
	events             []string
	lock               sync.Mutex
}

func (f *fakeBackend) Lock(context.Context) (func(), error) {
	f.lock.Lock()
	return f.lock.Unlock, nil
}

func (f *fakeBackend) Snapshot(context.Context) (Snapshot, error) { return f.snapshot, nil }

func (f *fakeBackend) Busy(context.Context) (bool, error) { return f.busy, nil }

func (f *fakeBackend) Reclaim(_ context.Context, request ReclaimRequest) (ReclaimReport, error) {
	f.events = append(f.events, "reclaim")
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
	f.events = append(f.events, "emergency")
	f.emergencies++
	f.busy = false
	if f.freeAfterEmergency != 0 {
		f.freeAfterGC = f.freeAfterEmergency
	}
	return nil
}
