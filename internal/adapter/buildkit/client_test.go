package buildkit

import (
	"context"
	"errors"
	"testing"
	"time"

	buildkitclient "github.com/moby/buildkit/client"
)

func TestClientChecksBuildKitInfo(t *testing.T) {
	engine := &fakeEngine{info: &buildkitclient.Info{BuildkitVersion: buildkitclient.BuildkitVersion{Version: "v0.30.0"}}}
	if err := newClient(engine).Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := errors.New("unavailable")
	engine.infoErr = want
	if err := newClient(engine).Check(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Check() error = %v, want %v", err, want)
	}
}

func TestClientPrunesAllCacheWhilePreservingMaximumUsage(t *testing.T) {
	engine := &fakeEngine{}
	if err := newClient(engine).Prune(context.Background(), 2_000_000_000); err != nil {
		t.Fatal(err)
	}
	if !engine.prune.All || engine.prune.MaxUsedSpace != 2_000_000_000 || engine.prune.ReservedSpace != 0 {
		t.Fatalf("prune = %#v", engine.prune)
	}
}

func TestClientBoundsBuildKitPrune(t *testing.T) {
	engine := &fakeEngine{blockPrune: true}
	runtime := &Client{engine: engine, operationTimeout: 10 * time.Millisecond}
	if err := runtime.Prune(context.Background(), 2_000_000_000); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Prune() error = %v, want deadline exceeded", err)
	}
}

func TestNormalizeAddressUsesBuildKitTCPScheme(t *testing.T) {
	for input, want := range map[string]string{
		"127.0.0.1:1234":       "tcp://127.0.0.1:1234",
		"tcp://builder:1234":   "tcp://builder:1234",
		"unix:///run/buildkit": "unix:///run/buildkit",
	} {
		if got := normalizeAddress(input); got != want {
			t.Fatalf("normalizeAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

type fakeEngine struct {
	info       *buildkitclient.Info
	infoErr    error
	prune      buildkitclient.PruneInfo
	blockPrune bool
}

func (f *fakeEngine) Info(context.Context) (*buildkitclient.Info, error) {
	return f.info, f.infoErr
}

func (f *fakeEngine) Prune(ctx context.Context, _ chan buildkitclient.UsageInfo, options ...buildkitclient.PruneOption) error {
	for _, option := range options {
		option.SetPruneOption(&f.prune)
	}
	if f.blockPrune {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (*fakeEngine) Close() error { return nil }
