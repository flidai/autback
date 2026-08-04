package server

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrJoinTimeout = errors.New("server component join timed out")

type Component struct {
	Name string
	Run  func(context.Context) error
	Stop func(context.Context) error
}

type Config struct {
	ShutdownTimeout time.Duration
	OnDrain         func()
}

type Group struct {
	config     Config
	components []Component
}

func New(config Config) *Group {
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 15 * time.Second
	}
	return &Group{config: config}
}

func (g *Group) Add(component Component) {
	g.components = append(g.components, component)
}

type result struct {
	name string
	err  error
}

// Run owns the complete process lifecycle. The first unexpected component
// exit begins draining, cancels every sibling, invokes stops in reverse order,
// and joins all component goroutines before returning.
func (g *Group) Run(ctx context.Context) error {
	if len(g.components) == 0 {
		return errors.New("server group requires at least one component")
	}
	for _, component := range g.components {
		if component.Name == "" || component.Run == nil {
			return errors.New("server component name and run function are required")
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan result, len(g.components))
	for _, component := range g.components {
		component := component
		go func() { results <- result{name: component.Name, err: component.Run(runCtx)} }()
	}

	var primary error
	consumed := 0
	if ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case first := <-results:
			consumed++
			if ctx.Err() != nil {
				break
			}
			if first.err == nil {
				primary = fmt.Errorf("server component %s stopped unexpectedly", first.name)
			} else if !isExpectedCancellation(first.err, runCtx) {
				primary = fmt.Errorf("server component %s: %w", first.name, first.err)
			}
		}
	}
	if g.config.OnDrain != nil {
		g.config.OnDrain()
	}
	cancel()

	shutdownCtx, stopShutdown := context.WithTimeout(context.Background(), g.config.ShutdownTimeout)
	defer stopShutdown()
	var shutdownErrors []error
	for index := len(g.components) - 1; index >= 0; index-- {
		component := g.components[index]
		if component.Stop == nil {
			continue
		}
		if err := component.Stop(shutdownCtx); err != nil && !isExpectedCancellation(err, shutdownCtx) {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("stop server component %s: %w", component.Name, err))
		}
	}

	for consumed < len(g.components) {
		select {
		case componentResult := <-results:
			consumed++
			if componentResult.err != nil && !isExpectedCancellation(componentResult.err, runCtx) {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("server component %s: %w", componentResult.name, componentResult.err))
			}
		case <-shutdownCtx.Done():
			shutdownErrors = append(shutdownErrors, fmt.Errorf("%w: %w", ErrJoinTimeout, shutdownCtx.Err()))
			return errors.Join(append([]error{primary}, shutdownErrors...)...)
		}
	}
	return errors.Join(append([]error{primary}, shutdownErrors...)...)
}

func isExpectedCancellation(err error, ctx context.Context) bool {
	return ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}
