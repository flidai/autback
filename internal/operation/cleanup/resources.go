package cleanup

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/flidai/autback/internal/control"
)

type ResourceSet struct {
	Containers []string `json:"containers"`
	Networks   []string `json:"networks"`
	Volumes    []string `json:"volumes"`
}

type ResourceBaselineStore interface {
	ResourceBaseline(context.Context, control.OperationKind, string) (ResourceSet, error)
	SaveResourceBaseline(context.Context, control.OperationKind, string, ResourceSet) error
}

type ResourceRuntime interface {
	Inventory(context.Context) (ResourceSet, error)
	RemoveContainer(context.Context, string) error
	RemoveNetwork(context.Context, string) error
	RemoveVolume(context.Context, string) error
}

type ResourceManagerConfig struct {
	Store       ResourceBaselineStore
	Runtime     ResourceRuntime
	GracePeriod time.Duration
	Timeout     time.Duration
	Wait        func(context.Context, time.Duration) error
}

type ResourceManager struct {
	config ResourceManagerConfig
}

func NewResourceManager(config ResourceManagerConfig) *ResourceManager {
	if config.GracePeriod <= 0 {
		config.GracePeriod = 10 * time.Second
	}
	if config.Timeout <= 0 {
		config.Timeout = 2 * time.Minute
	}
	if config.Wait == nil {
		config.Wait = waitDuration
	}
	return &ResourceManager{config: config}
}

// Prepare persists the restart-safe inventory boundary before an operation is
// allowed to create Docker resources. An existing baseline is immutable.
func (m *ResourceManager) Prepare(ctx context.Context, operation control.Operation) error {
	if err := m.validate(); err != nil {
		return err
	}
	if _, err := m.config.Store.ResourceBaseline(ctx, operation.Kind, operation.ID); err == nil {
		return nil
	} else if !errors.Is(err, control.ErrNotFound) {
		return fmt.Errorf("read resource baseline: %w", err)
	}
	resources, err := m.config.Runtime.Inventory(ctx)
	if err != nil {
		return fmt.Errorf("inventory Docker resources: %w", err)
	}
	if err := m.config.Store.SaveResourceBaseline(ctx, operation.Kind, operation.ID, normalize(resources)); err != nil {
		return fmt.Errorf("save resource baseline: %w", err)
	}
	return nil
}

// Cleanup removes every unprotected resource created after the persisted
// baseline. Removal order reverses resource dependencies: containers first,
// then networks, then volumes. A final inventory prevents lease release while
// any operation-owned resource remains.
func (m *ResourceManager) Cleanup(ctx context.Context, operation control.Operation) error {
	if err := m.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()
	baseline, err := m.config.Store.ResourceBaseline(ctx, operation.Kind, operation.ID)
	if errors.Is(err, control.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read resource baseline: %w", err)
	}
	if operation.Kind == control.OperationJob {
		if err := m.config.Wait(ctx, m.config.GracePeriod); err != nil {
			return err
		}
	}
	current, err := m.config.Runtime.Inventory(ctx)
	if err != nil {
		return fmt.Errorf("inventory Docker resources: %w", err)
	}
	owned := difference(current, baseline)
	var cleanupErrors []error
	for _, id := range owned.Containers {
		if err := m.config.Runtime.RemoveContainer(ctx, id); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove container %s: %w", id, err))
		}
	}
	for _, id := range owned.Networks {
		if err := m.config.Runtime.RemoveNetwork(ctx, id); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove network %s: %w", id, err))
		}
	}
	for _, id := range owned.Volumes {
		if err := m.config.Runtime.RemoveVolume(ctx, id); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove volume %s: %w", id, err))
		}
	}
	remaining, err := m.config.Runtime.Inventory(ctx)
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("verify Docker resource cleanup: %w", err))
	} else if remaining = difference(remaining, baseline); !remaining.empty() {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("operation-owned Docker resources remain: containers=%v networks=%v volumes=%v",
			remaining.Containers, remaining.Networks, remaining.Volumes))
	}
	return errors.Join(cleanupErrors...)
}

func (m *ResourceManager) validate() error {
	if m == nil || m.config.Store == nil || m.config.Runtime == nil {
		return errors.New("resource cleanup store and runtime are required")
	}
	return nil
}

func difference(current, baseline ResourceSet) ResourceSet {
	return normalize(ResourceSet{
		Containers: subtract(current.Containers, baseline.Containers),
		Networks:   subtract(current.Networks, baseline.Networks),
		Volumes:    subtract(current.Volumes, baseline.Volumes),
	})
}

func subtract(current, baseline []string) []string {
	protected := make(map[string]struct{}, len(baseline))
	for _, id := range baseline {
		protected[id] = struct{}{}
	}
	result := make([]string, 0, len(current))
	for _, id := range current {
		if _, ok := protected[id]; !ok && id != "" {
			result = append(result, id)
		}
	}
	return result
}

func normalize(resources ResourceSet) ResourceSet {
	resources.Containers = uniqueSorted(resources.Containers)
	resources.Networks = uniqueSorted(resources.Networks)
	resources.Volumes = uniqueSorted(resources.Volumes)
	return resources
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (r ResourceSet) empty() bool {
	return len(r.Containers) == 0 && len(r.Networks) == 0 && len(r.Volumes) == 0
}

func waitDuration(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

var _ Cleaner = (*ResourceManager)(nil)
