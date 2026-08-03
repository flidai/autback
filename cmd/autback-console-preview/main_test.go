package main

import (
	"context"
	"testing"
	"time"

	"github.com/flidai/autback/internal/console"
	"github.com/flidai/autback/internal/control"
)

func TestFixtureSourcePublishesEvolvingRunnerUtilization(t *testing.T) {
	source := newFixtureSource()
	source.interval = time.Millisecond
	principal := control.Principal{Kind: control.PrincipalDevice, UserID: "usr_owner", Admin: true}
	before, err := source.Snapshot(context.Background(), principal, console.Route{Kind: console.RouteOverview})
	if err != nil {
		t.Fatal(err)
	}
	updates, unsubscribe := source.SubscribeChanges()
	defer unsubscribe()
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("fixture did not publish a resource update")
	}
	after, err := source.Snapshot(context.Background(), principal, console.Route{Kind: console.RouteOverview})
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision <= before.Revision || after.Resources.Samples[0].CPUUtilization == before.Resources.Samples[0].CPUUtilization {
		t.Fatalf("before revision=%d CPU=%v; after revision=%d CPU=%v", before.Revision, before.Resources.Samples[0].CPUUtilization, after.Revision, after.Resources.Samples[0].CPUUtilization)
	}
}
