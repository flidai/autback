package cleanup

import (
	"context"
	"errors"

	"github.com/flidai/autback/internal/control"
)

type Preparer interface {
	Prepare(context.Context, control.Operation) error
}

type Lifecycle struct {
	Preparers []Preparer
	Cleaners  []Cleaner
}

func (l Lifecycle) Prepare(ctx context.Context, operation control.Operation) error {
	for _, preparer := range l.Preparers {
		if preparer == nil {
			continue
		}
		if err := preparer.Prepare(ctx, operation); err != nil {
			return err
		}
	}
	return nil
}

func (l Lifecycle) Cleanup(ctx context.Context, operation control.Operation) error {
	var cleanupErrors []error
	for _, cleaner := range l.Cleaners {
		if cleaner != nil {
			cleanupErrors = append(cleanupErrors, cleaner.Cleanup(ctx, operation))
		}
	}
	return errors.Join(cleanupErrors...)
}
