package docker

import (
	"errors"
	"testing"

	"github.com/containerd/errdefs"
)

func TestClassifyEngineErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want ErrorClass
	}{
		{"none", nil, ErrorNone},
		{"not found", errdefs.ErrNotFound, ErrorNotFound},
		{"daemon unavailable", errdefs.ErrUnavailable, ErrorRetryable},
		{"malformed resource", errdefs.ErrDataLoss, ErrorPoisoned},
		{"unsupported API", errdefs.ErrNotImplemented, ErrorContract},
		{"permission", errors.New("permission denied"), ErrorPermanent},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Classify(test.err); got != test.want {
				t.Fatalf("Classify() = %s, want %s", got, test.want)
			}
		})
	}
}
