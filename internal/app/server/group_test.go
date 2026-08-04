package server_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appserver "github.com/flidai/autback/internal/app/server"
)

func TestGroupDrainsCancelsStopsAndJoinsEveryComponent(t *testing.T) {
	want := errors.New("listener failed")
	var draining atomic.Bool
	var siblingStopped atomic.Bool
	var stopOrderMu sync.Mutex
	var stopOrder []string
	group := appserver.New(appserver.Config{
		ShutdownTimeout: time.Second,
		OnDrain:         func() { draining.Store(true) },
	})
	group.Add(appserver.Component{
		Name: "background",
		Run: func(ctx context.Context) error {
			<-ctx.Done()
			if !draining.Load() {
				t.Error("component cancellation was observed before draining")
			}
			siblingStopped.Store(true)
			return nil
		},
		Stop: func(context.Context) error {
			stopOrderMu.Lock()
			defer stopOrderMu.Unlock()
			stopOrder = append(stopOrder, "background")
			return nil
		},
	})
	group.Add(appserver.Component{
		Name: "listener",
		Run:  func(context.Context) error { return want },
		Stop: func(context.Context) error {
			stopOrderMu.Lock()
			defer stopOrderMu.Unlock()
			stopOrder = append(stopOrder, "listener")
			return nil
		},
	})

	err := group.Run(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Run error = %v, want %v", err, want)
	}
	if !draining.Load() || !siblingStopped.Load() {
		t.Fatalf("draining=%v sibling stopped=%v", draining.Load(), siblingStopped.Load())
	}
	stopOrderMu.Lock()
	defer stopOrderMu.Unlock()
	if len(stopOrder) != 2 || stopOrder[0] != "listener" || stopOrder[1] != "background" {
		t.Fatalf("stop order = %#v", stopOrder)
	}
}

func TestGroupTreatsParentCancellationAsCleanShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	group := appserver.New(appserver.Config{ShutdownTimeout: time.Second})
	started := make(chan struct{})
	joined := make(chan struct{})
	group.Add(appserver.Component{Name: "worker", Run: func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(joined)
		return nil
	}})
	result := make(chan error, 1)
	go func() { result <- group.Run(ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("component did not start")
	}
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return")
	}
	select {
	case <-joined:
	default:
		t.Fatal("component was not joined")
	}
}

func TestGroupRejectsInvalidComponentsBeforeStarting(t *testing.T) {
	group := appserver.New(appserver.Config{})
	group.Add(appserver.Component{Name: "missing-run"})
	if err := group.Run(context.Background()); err == nil {
		t.Fatal("Run error = nil")
	}
}

func TestGroupReturnsStopErrors(t *testing.T) {
	want := errors.New("shutdown failed")
	group := appserver.New(appserver.Config{ShutdownTimeout: time.Second})
	group.Add(appserver.Component{
		Name: "listener",
		Run:  func(context.Context) error { return errors.New("serve failed") },
		Stop: func(context.Context) error { return want },
	})

	if err := group.Run(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Run error = %v, want %v", err, want)
	}
}

func TestGroupTimesOutJoiningComponents(t *testing.T) {
	release := make(chan struct{})
	group := appserver.New(appserver.Config{ShutdownTimeout: 10 * time.Millisecond})
	group.Add(appserver.Component{
		Name: "stubborn-worker",
		Run: func(context.Context) error {
			<-release
			return nil
		},
	})
	group.Add(appserver.Component{
		Name: "failed-listener",
		Run:  func(context.Context) error { return errors.New("serve failed") },
	})

	err := group.Run(context.Background())
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run error = %v, want deadline exceeded", err)
	}
	if !errors.Is(err, appserver.ErrJoinTimeout) {
		t.Fatalf("Run error = %v, want join timeout", err)
	}
}

func TestGroupConvergesAfterFailureOfEveryServerComponent(t *testing.T) {
	componentNames := []string{
		"resource metrics",
		"reconciler",
		"capacity controller",
		"dispatcher",
		"CAS mTLS proxy",
		"BuildKit mTLS proxy",
		"control HTTP server",
	}
	for failedIndex, failedName := range componentNames {
		t.Run(failedName, func(t *testing.T) {
			want := errors.New("injected component failure")
			var joined atomic.Int32
			var stopped atomic.Int32
			group := appserver.New(appserver.Config{ShutdownTimeout: time.Second})
			for componentIndex, componentName := range componentNames {
				componentIndex := componentIndex
				group.Add(appserver.Component{
					Name: componentName,
					Run: func(ctx context.Context) error {
						if componentIndex == failedIndex {
							return want
						}
						<-ctx.Done()
						joined.Add(1)
						return nil
					},
					Stop: func(context.Context) error {
						stopped.Add(1)
						return nil
					},
				})
			}

			if err := group.Run(context.Background()); !errors.Is(err, want) {
				t.Fatalf("Run error = %v, want injected failure", err)
			}
			if got, wantCount := joined.Load(), int32(len(componentNames)-1); got != wantCount {
				t.Fatalf("joined components = %d, want %d", got, wantCount)
			}
			if got, wantCount := stopped.Load(), int32(len(componentNames)); got != wantCount {
				t.Fatalf("stopped components = %d, want %d", got, wantCount)
			}
		})
	}
}
