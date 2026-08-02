package reapi_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/flidai/outback/internal/reapi"
)

func TestActionUsesSelectedFilesAndPinnedRunner(t *testing.T) {
	got, options := reapi.Action(reapi.Request{
		Root:       "/repo",
		Files:      []string{"go.mod", "internal/proof_test.go"},
		Command:    []string{"go", "test", "./..."},
		Timeout:    15 * time.Minute,
		Repository: "example/service",
		Runner:     "standard",
	})

	if got.ExecRoot != "/repo" || !reflect.DeepEqual(got.InputSpec.Inputs, []string{"go.mod", "internal/proof_test.go"}) {
		t.Fatalf("command inputs = %#v", got)
	}
	if !reflect.DeepEqual(got.Args, []string{"go", "test", "./..."}) || got.Timeout != 15*time.Minute {
		t.Fatalf("command = %#v", got)
	}
	if got.Platform["outback.runner"] != "standard" || got.Platform["outback.repository"] != "example/service" {
		t.Fatalf("platform = %#v", got.Platform)
	}
	if options.AcceptCached || !options.DoNotCache || !options.DownloadOutErr || !options.StreamOutErr {
		t.Fatalf("execution options = %#v", options)
	}
}
