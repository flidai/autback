package docker

import (
	"errors"
	"io"
	"net/url"

	"github.com/containerd/errdefs"
)

type ErrorClass string

const (
	ErrorNone      ErrorClass = "none"
	ErrorNotFound  ErrorClass = "not_found"
	ErrorRetryable ErrorClass = "retryable"
	ErrorPoisoned  ErrorClass = "poisoned_resource"
	ErrorContract  ErrorClass = "contract_violation"
	ErrorPermanent ErrorClass = "permanent"
)

func Classify(err error) ErrorClass {
	var transportError *url.Error
	switch {
	case err == nil:
		return ErrorNone
	case errdefs.IsNotFound(err):
		return ErrorNotFound
	case errdefs.IsUnavailable(err), errdefs.IsDeadlineExceeded(err), errdefs.IsAborted(err), errdefs.IsResourceExhausted(err), errdefs.IsConflict(err), errdefs.IsInternal(err),
		errors.As(err, &transportError), errors.Is(err, io.ErrUnexpectedEOF):
		return ErrorRetryable
	case errdefs.IsDataLoss(err), errdefs.IsInvalidArgument(err):
		return ErrorPoisoned
	case errdefs.IsNotImplemented(err), errdefs.IsOutOfRange(err):
		return ErrorContract
	default:
		return ErrorPermanent
	}
}
