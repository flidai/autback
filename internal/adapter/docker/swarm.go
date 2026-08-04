package docker

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/flidai/autback/internal/control/swarmscheduler"
	"github.com/flidai/autback/internal/protocol"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/mount"
	engineswarm "github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

const (
	cancelledLabel    = "autback.cancelled"
	sharedMemorySize  = int64(1 << 30)
	minimumRuntimeAPI = "1.41"
)

type swarmEngine interface {
	ClientVersion() string
	Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error)
	ServiceCreate(context.Context, client.ServiceCreateOptions) (client.ServiceCreateResult, error)
	ServiceInspect(context.Context, string, client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	ServiceList(context.Context, client.ServiceListOptions) (client.ServiceListResult, error)
	TaskList(context.Context, client.TaskListOptions) (client.TaskListResult, error)
	ServiceUpdate(context.Context, string, client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
	ServiceRemove(context.Context, string, client.ServiceRemoveOptions) (client.ServiceRemoveResult, error)
	ServiceLogs(context.Context, string, client.ServiceLogsOptions) (client.ServiceLogsResult, error)
}

type imageEngine interface {
	ImagePull(context.Context, string, client.ImagePullOptions) (client.ImagePullResponse, error)
	ImageInspect(context.Context, string, ...client.ImageInspectOption) (client.ImageInspectResult, error)
}

func newRuntimeClient(runtime swarmEngine, images imageEngine) *Client {
	return &Client{runtime: runtime, images: images}
}

func (c *Client) Check(ctx context.Context) error {
	info, err := c.runtime.Info(ctx, client.InfoOptions{})
	if err != nil {
		return fmt.Errorf("inspect Docker Swarm: %w", err)
	}
	if !apiVersionAtLeast(c.runtime.ClientVersion(), minimumRuntimeAPI) {
		return fmt.Errorf("Docker Engine API %s is below Autback minimum %s: %w", c.runtime.ClientVersion(), minimumRuntimeAPI, errdefs.ErrOutOfRange)
	}
	state := info.Info.Swarm.LocalNodeState
	if state != engineswarm.LocalNodeStateActive {
		if state == "" {
			state = "unavailable"
		}
		return fmt.Errorf("Docker Swarm is %s, want active", state)
	}
	return nil
}

func apiVersionAtLeast(actual, minimum string) bool {
	actualMajor, actualMinor, actualOK := parseAPIVersion(actual)
	minimumMajor, minimumMinor, minimumOK := parseAPIVersion(minimum)
	return actualOK && minimumOK && (actualMajor > minimumMajor || actualMajor == minimumMajor && actualMinor >= minimumMinor)
}

func parseAPIVersion(value string) (int, int, bool) {
	majorText, minorText, found := strings.Cut(value, ".")
	if !found {
		return 0, 0, false
	}
	major, majorErr := strconv.Atoi(majorText)
	minor, minorErr := strconv.Atoi(minorText)
	return major, minor, majorErr == nil && minorErr == nil
}

func (c *Client) ValidateImage(ctx context.Context, image string) error {
	pull, err := c.images.ImagePull(ctx, image, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", image, err)
	}
	defer pull.Close()
	if err := pull.Wait(ctx); err != nil {
		return fmt.Errorf("pull image %s: %w", image, err)
	}
	inspection, err := c.images.ImageInspect(ctx, image)
	if err != nil {
		return fmt.Errorf("inspect image %s: %w", image, err)
	}
	if !strings.HasPrefix(inspection.ID, "sha256:") {
		return fmt.Errorf("inspect image %s: %w", image, errdefs.ErrDataLoss)
	}
	return nil
}

func (c *Client) Create(ctx context.Context, spec swarmscheduler.Spec) (string, error) {
	if spec.ID == "" {
		return "", fmt.Errorf("job ID is required: %w", errdefs.ErrInvalidArgument)
	}
	if spec.Image == "" {
		return "", fmt.Errorf("project image is required: %w", errdefs.ErrInvalidArgument)
	}
	result, err := c.runtime.ServiceCreate(ctx, client.ServiceCreateOptions{Spec: serviceSpec(spec), QueryRegistry: true})
	if err != nil {
		return "", fmt.Errorf("create Swarm job %s: %w", spec.ID, err)
	}
	return result.ID, nil
}

func serviceSpec(spec swarmscheduler.Spec) engineswarm.ServiceSpec {
	one := uint64(1)
	init := true
	stopGracePeriod := 15 * time.Second
	workspace := filepath.Join(spec.JobsRoot, spec.ID, "workspace")
	environment := []string{
		"AUTBACK_JOB_ID=" + spec.ID,
		"AUTBACK_WORKSPACE=" + workspace,
		"AUTBACK_HOST_UID=" + spec.HostUID,
		"AUTBACK_HOST_GID=" + spec.HostGID,
		"AUTBACK_CAS_ADDRESS=" + spec.CASAddress,
		"AUTBACK_CAS_INSTANCE=" + fallback(spec.CASInstance, "autback"),
		"AUTBACK_ROOT_DIGEST=" + spec.RootDigest,
		"AUTBACK_TIMEOUT_MILLIS=" + strconv.FormatInt(spec.Timeout.Milliseconds(), 10),
		"AUTBACK_WORKING_DIRECTORY=" + fallback(spec.WorkingDirectory, "."),
		"TESTCONTAINERS_HOST_OVERRIDE=localhost",
		"TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock",
		"RYUK_RECONNECTION_TIMEOUT=5s",
		"TMPDIR=/tmp",
		"TEST_DATA_DIR=" + filepath.Join(workspace, ".autback", "data"),
	}
	keys := make([]string, 0, len(spec.Environment))
	for key := range spec.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		environment = append(environment, key+"="+spec.Environment[key])
	}
	mounts := []mount.Mount{
		{Type: mount.TypeBind, Source: spec.JobsRoot, Target: spec.JobsRoot},
		{Type: mount.TypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
		{Type: mount.TypeTmpfs, Target: "/dev/shm", TmpfsOptions: &mount.TmpfsOptions{SizeBytes: sharedMemorySize}},
	}
	caches := append([]swarmscheduler.CacheMount(nil), spec.Caches...)
	sort.Slice(caches, func(i, j int) bool {
		if caches[i].Name == caches[j].Name {
			return caches[i].Target < caches[j].Target
		}
		return caches[i].Name < caches[j].Name
	})
	for _, cache := range caches {
		mounts = append(mounts, mount.Mount{
			Type: mount.TypeBind, Source: filepath.Join(spec.CacheRoot, spec.ProjectID, cache.Name), Target: cache.Target,
		})
	}
	if spec.EntrypointHostPath != "" {
		mounts = append(mounts, mount.Mount{
			Type: mount.TypeBind, Source: spec.EntrypointHostPath, Target: "/usr/local/bin/autback-job-entrypoint", ReadOnly: true,
		})
	}
	return engineswarm.ServiceSpec{
		Annotations: engineswarm.Annotations{Name: spec.ID, Labels: map[string]string{
			managedLabel: "true", "autback.project": spec.ProjectID, "autback.job": spec.ID,
			"autback.image": encodeLabel(spec.Image), "autback.timeout_seconds": strconv.Itoa(int(spec.Timeout.Seconds())),
			"autback.root_digest": spec.RootDigest,
		}},
		TaskTemplate: engineswarm.TaskSpec{
			ContainerSpec: &engineswarm.ContainerSpec{
				Image: spec.Image, Command: []string{"/usr/local/bin/autback-job-entrypoint"}, Args: append([]string(nil), spec.Command...),
				Env: environment, User: "0:0", Init: &init, Mounts: mounts, StopGracePeriod: &stopGracePeriod,
			},
			RestartPolicy: &engineswarm.RestartPolicy{Condition: engineswarm.RestartPolicyConditionNone},
			Networks:      []engineswarm.NetworkAttachmentConfig{{Target: "host"}},
			LogDriver: &engineswarm.Driver{Name: "local", Options: map[string]string{
				"max-size": "10m", "max-file": "2",
			}},
		},
		Mode: engineswarm.ServiceMode{ReplicatedJob: &engineswarm.ReplicatedJob{MaxConcurrent: &one, TotalCompletions: &one}},
	}
}

func (c *Client) Status(ctx context.Context, id string) (protocol.Job, error) {
	inspection, err := c.runtime.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return protocol.Job{}, fmt.Errorf("inspect Swarm job %s: %w", id, err)
	}
	return c.status(ctx, inspection.Service, id)
}

func (c *Client) status(ctx context.Context, service engineswarm.Service, fallbackID string) (protocol.Job, error) {
	id := fallback(service.Spec.Name, fallbackID)
	container := service.Spec.TaskTemplate.ContainerSpec
	if container == nil {
		return protocol.Job{}, fmt.Errorf("inspect Swarm job %s: missing container spec: %w", id, errdefs.ErrDataLoss)
	}
	labels := service.Spec.Labels
	if service.ID == "" {
		return protocol.Job{}, poisonedService(id, "missing service ID")
	}
	for _, label := range []string{managedLabel, "autback.project", "autback.job", "autback.image", "autback.timeout_seconds", "autback.root_digest"} {
		if labels[label] == "" {
			return protocol.Job{}, poisonedService(id, "missing "+label+" label")
		}
	}
	if labels[managedLabel] != "true" {
		return protocol.Job{}, poisonedService(id, "invalid "+managedLabel+" label")
	}
	if labels["autback.job"] != id {
		return protocol.Job{}, poisonedService(id, "autback.job label does not match service name")
	}
	timeoutSeconds, err := strconv.Atoi(labels["autback.timeout_seconds"])
	if err != nil || timeoutSeconds < 0 {
		return protocol.Job{}, poisonedService(id, "invalid autback.timeout_seconds label")
	}
	job := protocol.Job{
		ID: id, ProjectID: labels["autback.project"], Image: decodeLabel(labels["autback.image"]),
		RootDigest: labels["autback.root_digest"], Command: append([]string(nil), container.Args...), CreatedAt: service.CreatedAt,
		Status: protocol.StatusQueued, CancelRequested: labels[cancelledLabel] == "true",
	}
	job.TimeoutSeconds = timeoutSeconds
	if job.CancelRequested {
		job.Status = protocol.StatusCancelled
		finished := service.UpdatedAt
		job.FinishedAt = &finished
	}
	tasks, err := c.runtime.TaskList(ctx, client.TaskListOptions{Filters: client.Filters{}.Add("service", service.ID)})
	if err != nil {
		return protocol.Job{}, fmt.Errorf("list Swarm job %s tasks: %w", id, err)
	}
	if len(tasks.Items) == 0 {
		return job, nil
	}
	sort.Slice(tasks.Items, func(i, j int) bool { return tasks.Items[i].CreatedAt.After(tasks.Items[j].CreatedAt) })
	task := tasks.Items[0]
	exitCode := 0
	if task.Status.ContainerStatus != nil {
		exitCode = task.Status.ContainerStatus.ExitCode
	} else if terminalTaskState(task.Status.State) {
		return protocol.Job{}, fmt.Errorf("inspect Swarm job %s task %s: missing container status: %w", id, task.ID, errdefs.ErrDataLoss)
	}
	job.Status = runtimeTaskStatus(task.Status.State, exitCode, job.CancelRequested)
	job.WorkerID = task.NodeID
	if !task.CreatedAt.IsZero() {
		started := task.CreatedAt
		job.StartedAt = &started
	}
	if job.Status.Terminal() {
		finished := task.Status.Timestamp
		if finished.IsZero() {
			finished = task.UpdatedAt
		}
		job.FinishedAt = &finished
		job.ExitCode = &exitCode
		job.ErrorMessage = task.Status.Err
	}
	return job, nil
}

func (c *Client) Logs(ctx context.Context, id string, follow bool, output io.Writer) error {
	logs, err := c.runtime.ServiceLogs(ctx, id, client.ServiceLogsOptions{ShowStdout: true, ShowStderr: true, Follow: follow})
	if err != nil {
		return fmt.Errorf("read Swarm job %s logs: %w", id, err)
	}
	defer logs.Close()
	_, err = stdcopy.StdCopy(output, output, logs)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("read Swarm job %s logs: %w", id, err)
	}
	return ctx.Err()
}

func (c *Client) Cancel(ctx context.Context, id string) error {
	for attempt := 0; attempt < 3; attempt++ {
		inspection, err := c.runtime.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
		if err != nil {
			if Classify(err) == ErrorNotFound {
				return nil
			}
			return fmt.Errorf("inspect Swarm job %s for cancellation: %w", id, err)
		}
		service := inspection.Service
		if service.Spec.Mode.ReplicatedJob == nil {
			return fmt.Errorf("cancel Swarm job %s: service is not a replicated job: %w", id, errdefs.ErrDataLoss)
		}
		service.Spec.Labels = cloneMap(service.Spec.Labels)
		service.Spec.Labels[cancelledLabel] = "true"
		zero := uint64(0)
		service.Spec.Mode.ReplicatedJob.TotalCompletions = &zero
		_, err = c.runtime.ServiceUpdate(ctx, service.ID, client.ServiceUpdateOptions{Version: service.Version, Spec: service.Spec})
		if err == nil || Classify(err) == ErrorNotFound {
			return nil
		}
		if Classify(err) != ErrorRetryable || attempt == 2 {
			return fmt.Errorf("cancel Swarm job %s: %w", id, err)
		}
	}
	return nil
}

func (c *Client) Remove(ctx context.Context, id string) error {
	_, err := c.runtime.ServiceRemove(ctx, id, client.ServiceRemoveOptions{})
	if err == nil || Classify(err) == ErrorNotFound {
		return nil
	}
	return fmt.Errorf("remove Swarm job %s: %w", id, err)
}

func (c *Client) ListResults(ctx context.Context) ([]swarmscheduler.JobResult, error) {
	services, err := c.runtime.ServiceList(ctx, client.ServiceListOptions{
		Filters: client.Filters{}.Add("label", managedLabel+"=true"),
	})
	if err != nil {
		return nil, fmt.Errorf("list Swarm jobs: %w", err)
	}
	results := make([]swarmscheduler.JobResult, 0, len(services.Items))
	for _, summary := range services.Items {
		id := fallback(summary.Spec.Name, summary.ID)
		inspection, inspectErr := c.runtime.ServiceInspect(ctx, fallback(summary.ID, id), client.ServiceInspectOptions{})
		if inspectErr != nil {
			results = append(results, swarmscheduler.JobResult{ID: id, Err: fmt.Errorf("inspect Swarm job %s: %w", id, inspectErr)})
			continue
		}
		job, statusErr := c.status(ctx, inspection.Service, id)
		results = append(results, swarmscheduler.JobResult{ID: id, Job: job, Err: statusErr})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Job.CreatedAt.Equal(results[j].Job.CreatedAt) {
			return results[i].ID < results[j].ID
		}
		return results[i].Job.CreatedAt.After(results[j].Job.CreatedAt)
	})
	return results, nil
}

func poisonedService(id, detail string) error {
	return fmt.Errorf("inspect Swarm job %s: %s: %w", id, detail, errdefs.ErrDataLoss)
}

func runtimeTaskStatus(state engineswarm.TaskState, exitCode int, cancelled bool) protocol.Status {
	if cancelled {
		return protocol.StatusCancelled
	}
	switch state {
	case engineswarm.TaskStateComplete:
		if exitCode == 0 {
			return protocol.StatusSucceeded
		}
		return protocol.StatusFailed
	case engineswarm.TaskStateFailed, engineswarm.TaskStateRejected, engineswarm.TaskStateOrphaned:
		if exitCode == 124 {
			return protocol.StatusTimedOut
		}
		return protocol.StatusFailed
	case engineswarm.TaskStateRunning:
		return protocol.StatusRunning
	default:
		return protocol.StatusQueued
	}
}

func terminalTaskState(state engineswarm.TaskState) bool {
	return state == engineswarm.TaskStateComplete || state == engineswarm.TaskStateFailed || state == engineswarm.TaskStateRejected || state == engineswarm.TaskStateOrphaned
}

func encodeLabel(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeLabel(value string) string {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	return string(decoded)
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func cloneMap(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values)+1)
	for key, value := range values {
		clone[key] = value
	}
	return clone
}
