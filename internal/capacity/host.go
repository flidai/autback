package capacity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type CapacityStore interface {
	WorkerBusy(context.Context) (bool, error)
	TerminalJobIDsBefore(context.Context, time.Time) ([]string, error)
	CapacityImagePolicies(context.Context) ([]ImagePolicy, error)
}

type ImagePolicy struct {
	Reference  string
	LastUsedAt time.Time
	Protected  bool
}

type Commands interface {
	Run(context.Context, ...string) error
	Output(context.Context, ...string) ([]byte, error)
}

type HostConfig struct {
	CapacityPath string
	JobsRoot     string
	CacheRoot    string
	LockPath     string
	Store        CapacityStore
	Commands     Commands
	Emergency    func(context.Context) error
	Now          func() time.Time
	DryRun       bool
}

type Host struct {
	config HostConfig
}

func NewHost(config HostConfig) *Host {
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Commands == nil {
		config.Commands = DockerCommands{}
	}
	return &Host{config: config}
}

func (h *Host) Snapshot(context.Context) (Snapshot, error) {
	var stats unix.Statfs_t
	if err := unix.Statfs(h.config.CapacityPath, &stats); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		TotalBytes:  uint64(stats.Blocks) * uint64(stats.Bsize),
		FreeBytes:   uint64(stats.Bavail) * uint64(stats.Bsize),
		TotalInodes: stats.Files, FreeInodes: stats.Ffree,
		ObservedAt: h.config.Now().UTC(),
	}, nil
}

func (h *Host) Busy(ctx context.Context) (bool, error) {
	if h.config.Store == nil {
		return false, nil
	}
	return h.config.Store.WorkerBusy(ctx)
}

func (h *Host) Reclaim(ctx context.Context, request ReclaimRequest) (ReclaimReport, error) {
	before, err := h.Snapshot(ctx)
	if err != nil {
		return ReclaimReport{}, err
	}
	report := ReclaimReport{}
	var reclaimErrors []error

	removedJobs, err := h.removeTerminalJobs(ctx, request.JobRetention)
	report.RemovedJobs += removedJobs
	if err != nil {
		reclaimErrors = append(reclaimErrors, err)
	}
	if h.reachedTarget(ctx, request) {
		return h.finishReport(ctx, before, report, errors.Join(reclaimErrors...))
	}

	age := dockerAge(request.NormalObjectAge)
	containerCommand := []string{"container", "prune", "--force", "--filter", "label=org.testcontainers=true", "--filter", "until=" + age}
	if request.Pressure {
		age = "5m"
		containerCommand = []string{"container", "prune", "--force", "--filter", "until=" + age}
	}
	for _, command := range [][]string{containerCommand,
		{"network", "prune", "--force", "--filter", "until=" + age},
		{"volume", "prune", "--force"},
	} {
		report.Commands = append(report.Commands, "docker "+joinCommand(command))
		if !h.config.DryRun {
			if err := h.config.Commands.Run(ctx, command...); err != nil {
				reclaimErrors = append(reclaimErrors, fmt.Errorf("docker %s: %w", joinCommand(command), err))
			}
		}
		if h.reachedTarget(ctx, request) {
			return h.finishReport(ctx, before, report, errors.Join(reclaimErrors...))
		}
	}

	if request.Pressure {
		commands, err := h.pruneUnusedImages(ctx, request.TargetFreeBytes)
		report.Commands = append(report.Commands, commands...)
		if err != nil {
			reclaimErrors = append(reclaimErrors, err)
		}
	} else {
		imageCommand := []string{"image", "prune", "--force", "--filter", "until=" + age}
		report.Commands = append(report.Commands, "docker "+joinCommand(imageCommand))
		if !h.config.DryRun {
			if err := h.config.Commands.Run(ctx, imageCommand...); err != nil {
				reclaimErrors = append(reclaimErrors, fmt.Errorf("docker %s: %w", joinCommand(imageCommand), err))
			}
		}
	}
	if h.reachedTarget(ctx, request) {
		return h.finishReport(ctx, before, report, errors.Join(reclaimErrors...))
	}

	removedCaches, err := h.pruneCaches(request.CacheHighBytes, request.CacheLowBytes, request.Pressure)
	report.RemovedCaches += removedCaches
	if err != nil {
		reclaimErrors = append(reclaimErrors, err)
	}
	if h.reachedTarget(ctx, request) {
		return h.finishReport(ctx, before, report, errors.Join(reclaimErrors...))
	}

	if !request.Pressure {
		return h.finishReport(ctx, before, report, errors.Join(reclaimErrors...))
	}
	keepStorage := "2000"
	buildkitCommand := []string{"exec", "autback-buildkit", "buildctl", "--addr", "tcp://127.0.0.1:1234", "prune", "--all", "--keep-storage", keepStorage}
	report.Commands = append(report.Commands, "docker "+joinCommand(buildkitCommand))
	if !h.config.DryRun {
		if err := h.config.Commands.Run(ctx, buildkitCommand...); err != nil {
			reclaimErrors = append(reclaimErrors, fmt.Errorf("prune BuildKit: %w", err))
		}
	}
	return h.finishReport(ctx, before, report, errors.Join(reclaimErrors...))
}

type dockerImage struct {
	ID          string    `json:"Id"`
	RepoTags    []string  `json:"RepoTags"`
	RepoDigests []string  `json:"RepoDigests"`
	Created     time.Time `json:"Created"`
	lastUsed    time.Time
	protected   bool
}

func (h *Host) pruneUnusedImages(ctx context.Context, targetFreeBytes uint64) ([]string, error) {
	idsOutput, err := h.config.Commands.Output(ctx, "image", "ls", "--quiet", "--no-trunc")
	if err != nil {
		return nil, fmt.Errorf("list Docker images: %w", err)
	}
	ids := strings.Fields(string(idsOutput))
	if len(ids) == 0 {
		return nil, nil
	}
	inspectOutput, err := h.config.Commands.Output(ctx, append([]string{"image", "inspect"}, ids...)...)
	if err != nil {
		return nil, fmt.Errorf("inspect Docker images: %w", err)
	}
	var images []dockerImage
	if err := json.Unmarshal(inspectOutput, &images); err != nil {
		return nil, fmt.Errorf("decode Docker images: %w", err)
	}
	policies, err := h.config.Store.CapacityImagePolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("load protected Docker images: %w", err)
	}
	byReference := make(map[string]ImagePolicy, len(policies))
	for _, policy := range policies {
		byReference[policy.Reference] = policy
	}
	for index := range images {
		images[index].lastUsed = images[index].Created
		for _, reference := range append(append([]string(nil), images[index].RepoTags...), images[index].RepoDigests...) {
			policy, ok := byReference[reference]
			if !ok {
				continue
			}
			images[index].protected = images[index].protected || policy.Protected
			if policy.LastUsedAt.After(images[index].lastUsed) {
				images[index].lastUsed = policy.LastUsedAt
			}
		}
	}
	sort.Slice(images, func(i, j int) bool {
		if images[i].lastUsed.Equal(images[j].lastUsed) {
			return images[i].ID < images[j].ID
		}
		return images[i].lastUsed.Before(images[j].lastUsed)
	})
	creationGrace := h.config.Now().Add(-5 * time.Minute)
	var commands []string
	for _, image := range images {
		if image.protected || image.lastUsed.After(creationGrace) {
			continue
		}
		command := []string{"image", "rm", image.ID}
		commands = append(commands, "docker "+joinCommand(command))
		if !h.config.DryRun {
			// Docker refuses removal while any container references the image. That
			// runtime ownership protection is intentional, so a conflict is skipped.
			_ = h.config.Commands.Run(ctx, command...)
		}
		if targetFreeBytes > 0 {
			snapshot, snapshotErr := h.Snapshot(ctx)
			if snapshotErr != nil {
				return commands, snapshotErr
			}
			if snapshot.FreeBytes >= targetFreeBytes {
				break
			}
		}
	}
	return commands, nil
}

func (h *Host) Lock(ctx context.Context) (func(), error) {
	if h.config.LockPath == "" {
		return func() {}, nil
	}
	file, err := os.OpenFile(h.config.LockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
			return func() {
				_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
				_ = file.Close()
			}, nil
		} else if !errors.Is(err, unix.EWOULDBLOCK) {
			_ = file.Close()
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			_ = file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (h *Host) Emergency(ctx context.Context) error {
	if h.config.DryRun {
		return nil
	}
	if h.config.Emergency == nil {
		return errors.New("no emergency operation stopper is configured")
	}
	return h.config.Emergency(ctx)
}

func (h *Host) removeTerminalJobs(ctx context.Context, retention time.Duration) (int, error) {
	if h.config.Store == nil || retention <= 0 {
		return 0, nil
	}
	ids, err := h.config.Store.TerminalJobIDsBefore(ctx, h.config.Now().UTC().Add(-retention))
	if err != nil {
		return 0, fmt.Errorf("list retained terminal jobs: %w", err)
	}
	removed := 0
	for _, id := range ids {
		if !safeChild(id) {
			return removed, fmt.Errorf("refuse unsafe job directory %q", id)
		}
		path := filepath.Join(h.config.JobsRoot, id)
		if !h.config.DryRun {
			if err := os.RemoveAll(path); err != nil {
				return removed, fmt.Errorf("remove terminal job %s: %w", id, err)
			}
		}
		removed++
	}
	return removed, nil
}

type cacheDirectory struct {
	path    string
	modTime time.Time
	size    uint64
}

func (h *Host) pruneCaches(high, low uint64, pressure bool) (int, error) {
	caches, total, err := cacheDirectories(h.config.CacheRoot)
	if err != nil {
		return 0, err
	}
	if !pressure && (high == 0 || total <= high) {
		return 0, nil
	}
	if low > high && high != 0 {
		low = high
	}
	sort.Slice(caches, func(i, j int) bool {
		if caches[i].modTime.Equal(caches[j].modTime) {
			return caches[i].path < caches[j].path
		}
		return caches[i].modTime.Before(caches[j].modTime)
	})
	removed := 0
	for _, cache := range caches {
		if total <= low {
			break
		}
		if !h.config.DryRun {
			if err := os.RemoveAll(cache.path); err != nil {
				return removed, fmt.Errorf("remove project cache %s: %w", cache.path, err)
			}
		}
		removed++
		if cache.size >= total {
			total = 0
		} else {
			total -= cache.size
		}
	}
	return removed, nil
}

func cacheDirectories(root string) ([]cacheDirectory, uint64, error) {
	projects, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var result []cacheDirectory
	var total uint64
	for _, project := range projects {
		if !project.IsDir() || !safeChild(project.Name()) {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, project.Name()))
		if err != nil {
			return nil, 0, err
		}
		for _, entry := range entries {
			if !entry.IsDir() || !safeChild(entry.Name()) {
				continue
			}
			path := filepath.Join(root, project.Name(), entry.Name())
			info, err := entry.Info()
			if err != nil {
				return nil, 0, err
			}
			size, err := directorySize(path)
			if err != nil {
				return nil, 0, err
			}
			result = append(result, cacheDirectory{path: path, modTime: info.ModTime(), size: size})
			total += size
		}
	}
	return result, total, nil
}

func directorySize(root string) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += uint64(info.Size())
		}
		return nil
	})
	return total, err
}

func (h *Host) reachedTarget(ctx context.Context, request ReclaimRequest) bool {
	if !request.Pressure || request.TargetFreeBytes == 0 {
		return false
	}
	snapshot, err := h.Snapshot(ctx)
	return err == nil && snapshot.FreeBytes >= request.TargetFreeBytes
}

func (h *Host) finishReport(ctx context.Context, before Snapshot, report ReclaimReport, err error) (ReclaimReport, error) {
	after, snapshotErr := h.Snapshot(ctx)
	if after.FreeBytes > before.FreeBytes {
		report.ReclaimedBytes = after.FreeBytes - before.FreeBytes
	}
	return report, errors.Join(err, snapshotErr)
}

func safeChild(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value
}

func dockerAge(duration time.Duration) string {
	if duration <= 0 {
		return "24h"
	}
	if duration%time.Hour == 0 {
		return strconv.FormatInt(int64(duration/time.Hour), 10) + "h"
	}
	return duration.String()
}

func joinCommand(arguments []string) string {
	result := ""
	for index, argument := range arguments {
		if index > 0 {
			result += " "
		}
		result += argument
	}
	return result
}

type DockerCommands struct {
	Binary string
	Host   string
}

func (d DockerCommands) Run(ctx context.Context, arguments ...string) error {
	_, err := d.Output(ctx, arguments...)
	return err
}

func (d DockerCommands) Output(ctx context.Context, arguments ...string) ([]byte, error) {
	binary := d.Binary
	if binary == "" {
		binary = "docker"
	}
	if d.Host != "" {
		arguments = append([]string{"--host", d.Host}, arguments...)
	}
	output, err := exec.CommandContext(ctx, binary, arguments...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(output))
	}
	return output, nil
}
