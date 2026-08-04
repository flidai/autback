package docker

import (
	"errors"
	"net/url"
	"syscall"
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
		{"daemon connection refused", &url.Error{Op: "GET", URL: "http://docker/info", Err: syscall.ECONNREFUSED}, ErrorRetryable},
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
