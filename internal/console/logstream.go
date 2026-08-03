package console

import (
	"context"
	"time"

	"github.com/flidai/autback/internal/control"
)

const logFollowRetryInterval = 250 * time.Millisecond

// SubscribeLog returns a bounded live tail for an authorized job detail route.
// Other routes have no log stream and receive an already-closed channel.
func (s *SQLiteSource) SubscribeLog(ctx context.Context, principal control.Principal, route Route) (<-chan LogView, error) {
	output := make(chan LogView, 1)
	if route.Kind != RouteOperation || route.OperationKind != "job" {
		close(output)
		return output, nil
	}
	job, err := s.store.Job(ctx, route.OperationID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.AuthorizeProject(ctx, principal, job.ProjectID); err != nil {
		return nil, err
	}
	changes, unsubscribe := s.store.SubscribeChanges()
	go func() {
		defer close(output)
		defer unsubscribe()
		s.streamJobLog(ctx, job.ID, changes, output)
	}()
	return output, nil
}

func (s *SQLiteSource) streamJobLog(ctx context.Context, jobID string, changes <-chan struct{}, output chan LogView) {
	for {
		job, err := s.store.Job(ctx, jobID)
		if err != nil || ctx.Err() != nil {
			return
		}
		if job.Status.Terminal() {
			s.readFinalLog(ctx, jobID, output)
			return
		}
		state, err := s.store.OperationState(ctx, control.OperationJob, jobID)
		if err == nil && state == control.OperationActive {
			if s.followActiveJobLog(ctx, jobID, changes, output) {
				return
			}
			if !waitForLogRetry(ctx, changes) {
				return
			}
			continue
		}
		select {
		case <-ctx.Done():
			return
		case _, open := <-changes:
			if !open {
				return
			}
		}
	}
}

// followActiveJobLog returns true when the subscription should stop. A log
// follower can close before the durable job state becomes terminal, so callers
// retry from the scheduler's current output.
func (s *SQLiteSource) followActiveJobLog(ctx context.Context, jobID string, changes <-chan struct{}, output chan LogView) bool {
	followContext, stop := context.WithCancel(ctx)
	writer := newLiveLogWriter(output)
	writer.publish()
	done := make(chan error, 1)
	go func() { done <- s.scheduler.Logs(followContext, jobID, true, writer) }()
	for {
		select {
		case <-ctx.Done():
			stop()
			<-done
			return true
		case <-done:
			stop()
			return false
		case _, open := <-changes:
			if !open {
				stop()
				<-done
				return true
			}
			job, err := s.store.Job(ctx, jobID)
			if err != nil {
				stop()
				<-done
				return true
			}
			if job.Status.Terminal() {
				stop()
				<-done
				s.readFinalLog(ctx, jobID, output)
				return true
			}
		}
	}
}

func (s *SQLiteSource) readFinalLog(ctx context.Context, jobID string, output chan LogView) {
	writer := newLiveLogWriter(output)
	if err := s.scheduler.Logs(ctx, jobID, false, writer); err == nil && !writer.published {
		writer.publish()
	}
}

func waitForLogRetry(ctx context.Context, changes <-chan struct{}) bool {
	timer := time.NewTimer(logFollowRetryInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case _, open := <-changes:
		return open
	case <-timer.C:
		return true
	}
}

type liveLogWriter struct {
	tailWriter
	output    chan LogView
	published bool
}

func newLiveLogWriter(output chan LogView) *liveLogWriter {
	return &liveLogWriter{tailWriter: tailWriter{limit: maxLogTailBytes}, output: output}
}

func (w *liveLogWriter) Write(input []byte) (int, error) {
	written, err := w.tailWriter.Write(input)
	w.publish()
	return written, err
}

func (w *liveLogWriter) publish() {
	w.published = true
	view := LogView{Available: true, Truncated: w.truncated, Content: string(w.bytes)}
	select {
	case w.output <- view:
		return
	default:
	}
	select {
	case <-w.output:
	default:
	}
	select {
	case w.output <- view:
	default:
	}
}
