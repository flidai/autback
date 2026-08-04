package hostmetrics

import (
	"context"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
)

func TestCollectorAttributesSamplesToTheActiveOperation(t *testing.T) {
	now := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)
	store := &collectorStore{scope: control.ResourceScope{
		ProjectID: "prj_example", OperationKind: control.OperationJob, OperationID: "job_example",
	}}
	collector, err := NewCollector(CollectorConfig{
		Store: store,
		Sampler: sampleFunc(func(context.Context) (control.ResourceSample, error) {
			return control.ResourceSample{ObservedAt: now, CPUUtilization: .75, MemoryUtilization: .5, MemoryUsageBytes: 4 << 30, MemoryTotalBytes: 8 << 30, CPUCores: 4}, nil
		}),
		Now: func() time.Time { return now }, RawRetention: 14 * 24 * time.Hour, RollupRetention: 180 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.appended.OperationID != "job_example" || store.appended.OperationKind != control.OperationJob || store.appended.ProjectID != "prj_example" {
		t.Fatalf("sample scope=%#v", store.appended.ResourceScope)
	}
}

func TestCollectorAttributesOnlyNewCgroupEvents(t *testing.T) {
	now := time.Date(2026, time.August, 4, 8, 0, 0, 0, time.UTC)
	store := &collectorStore{scope: control.ResourceScope{ProjectID: "project", OperationKind: control.OperationJob, OperationID: "job"}}
	samples := []control.ResourceSample{
		{ObservedAt: now, CPUCores: 1, MemoryTotalBytes: 1, OOMEvents: 4, OOMKills: 2, MemoryHighEvents: 7},
		{ObservedAt: now.Add(time.Second), CPUCores: 1, MemoryTotalBytes: 1, OOMEvents: 5, OOMKills: 3, MemoryHighEvents: 9},
	}
	collector, err := NewCollector(CollectorConfig{Store: store, Sampler: sampleFunc(func(context.Context) (control.ResourceSample, error) {
		sample := samples[0]
		samples = samples[1:]
		return sample, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.appended.OOMKills != 0 || store.appended.OOMEvents != 0 || store.appended.MemoryHighEvents != 0 {
		t.Fatalf("first sample attributed historical events: %#v", store.appended)
	}
	if err := collector.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.appended.OOMKills != 1 || store.appended.OOMEvents != 1 || store.appended.MemoryHighEvents != 2 || store.appended.OperationID != "job" {
		t.Fatalf("event deltas = %#v", store.appended)
	}
}

type sampleFunc func(context.Context) (control.ResourceSample, error)

func (f sampleFunc) Sample(ctx context.Context) (control.ResourceSample, error) { return f(ctx) }

type collectorStore struct {
	scope    control.ResourceScope
	appended control.ResourceSample
}

func (s *collectorStore) ActiveResourceScope(context.Context) (control.ResourceScope, bool, error) {
	return s.scope, s.scope.OperationID != "", nil
}

func (s *collectorStore) AppendResourceSample(_ context.Context, sample control.ResourceSample) error {
	s.appended = sample
	return nil
}

func (s *collectorStore) CompactResourceSamples(context.Context, time.Time, time.Time) error {
	return nil
}
