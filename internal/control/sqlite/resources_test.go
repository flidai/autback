package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
)

func TestResourceSamplesAreDurableAttributedAndSummarized(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", ProjectName: "Example", TokenName: "device"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)
	updates, unsubscribe := store.SubscribeChanges()
	defer unsubscribe()
	for index, cpu := range []float64{.25, .75, .5} {
		sample := control.ResourceSample{
			ObservedAt:     now.Add(time.Duration(index) * 2 * time.Second),
			ResourceScope:  control.ResourceScope{ProjectID: bootstrap.Project.ID, OperationKind: control.OperationJob, OperationID: "job_example"},
			CPUUtilization: cpu, CPUCores: 4, MemoryUtilization: .5 + float64(index)*.1,
			MemoryUsageBytes: uint64(4+index) << 30, MemoryTotalBytes: 8 << 30,
			DiskUsageBytes: 40 << 30, DiskTotalBytes: 80 << 30,
			DiskInodesUsed: 100, DiskInodesTotal: 1000,
			CPUPressure: .1, MemoryPressure: .2, MemoryFullPressure: .05, IOPressure: .3, IOFullPressure: .1,
			MemoryHighEvents: uint64(index), OOMEvents: uint64(index), OOMKills: uint64(index), PIDsCurrent: 100, PIDsLimit: 4096,
		}
		if err := store.AppendResourceSample(ctx, sample); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("resource commit did not wake console subscribers")
	}
	samples, err := store.ListResourceSamples(ctx, control.ResourceFilter{OperationKind: control.OperationJob, OperationID: "job_example", From: now.Add(-time.Second)}, 100)
	if err != nil || len(samples) != 3 {
		t.Fatalf("samples=%#v err=%v", samples, err)
	}
	if sample := samples[2]; sample.DiskInodesUsed != 100 || sample.DiskInodesTotal != 1000 || sample.CPUPressure != .1 || sample.MemoryPressure != .2 || sample.MemoryFullPressure != .05 || sample.IOPressure != .3 || sample.IOFullPressure != .1 || sample.OOMKills != 2 || sample.PIDsCurrent != 100 || sample.PIDsLimit != 4096 {
		t.Fatalf("resource evidence was not durable: %#v", sample)
	}
	summary, err := store.ResourceSummary(ctx, control.ResourceFilter{OperationKind: control.OperationJob, OperationID: "job_example"})
	if err != nil {
		t.Fatal(err)
	}
	if summary.SampleCount != 3 || summary.CPUAverage != .5 || summary.CPUPeak != .75 || summary.MemoryPeak != .7 || summary.MemoryBytesPeak != 6<<30 {
		t.Fatalf("summary=%#v", summary)
	}
	summaries, err := store.ListResourceSummaries(ctx, bootstrap.Project.ID, 10)
	if err != nil || len(summaries) != 1 || summaries[0].OperationID != "job_example" || summaries[0].CPUAverage != .5 {
		t.Fatalf("summaries=%#v err=%v", summaries, err)
	}
}

func TestResourceCompactionCreatesMinuteRollupsAndPreservesRunSummary(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, time.August, 3, 8, 0, 0, 0, time.UTC)
	old := now.Add(-15 * 24 * time.Hour)
	sample := control.ResourceSample{
		ObservedAt:     old,
		ResourceScope:  control.ResourceScope{ProjectID: "prj_example", OperationKind: control.OperationBuild, OperationID: "bld_example"},
		CPUUtilization: .8, CPUCores: 4, MemoryUtilization: .6, MemoryUsageBytes: 5 << 30, MemoryTotalBytes: 8 << 30,
		DiskUsageBytes: 40 << 30, DiskTotalBytes: 80 << 30,
	}
	if err := store.AppendResourceSample(ctx, sample); err != nil {
		t.Fatal(err)
	}
	if err := store.CompactResourceSamples(ctx, now.Add(-14*24*time.Hour), now.Add(-180*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	samples, err := store.ListResourceSamples(ctx, control.ResourceFilter{From: old.Add(-time.Minute)}, 100)
	if err != nil || len(samples) != 0 {
		t.Fatalf("raw samples=%#v err=%v", samples, err)
	}
	rollups, err := store.ListResourceRollups(ctx, control.ResourceFilter{OperationKind: control.OperationBuild, OperationID: "bld_example", From: old.Add(-time.Minute)}, 100)
	if err != nil || len(rollups) != 1 || rollups[0].SampleCount != 1 {
		t.Fatalf("rollups=%#v err=%v", rollups, err)
	}
	summary, err := store.ResourceSummary(ctx, control.ResourceFilter{OperationKind: control.OperationBuild, OperationID: "bld_example"})
	if err != nil || summary.SampleCount != 1 {
		t.Fatalf("summary=%#v err=%v", summary, err)
	}
}
