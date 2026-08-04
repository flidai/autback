package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/flidai/autback/internal/control/swarmscheduler"
	"github.com/flidai/autback/internal/protocol"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

func TestTypedSwarmListResultsIsolatesPoisonedService(t *testing.T) {
	createdAt := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
	finishedAt := createdAt.Add(time.Minute)
	api := &fakeSwarmEngine{
		services: []swarm.Service{
			{ID: "poisoned-id", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "poisoned"}}},
			{ID: "healthy-id", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "healthy"}}},
		},
		inspect: map[string]swarm.Service{
			"healthy-id": managedService("healthy-id", "healthy", createdAt),
		},
		inspectErrors: map[string]error{"poisoned-id": errdefs.ErrDataLoss},
		tasks: map[string][]swarm.Task{
			"healthy-id": {{
				Meta: swarm.Meta{CreatedAt: createdAt}, NodeID: "worker-1",
				Status: swarm.TaskStatus{State: swarm.TaskStateComplete, Timestamp: finishedAt, ContainerStatus: &swarm.ContainerStatus{}},
			}},
		},
	}

	results, err := newRuntimeClient(api, nil).ListResults(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	byID := map[string]swarmscheduler.JobResult{}
	for _, result := range results {
		byID[result.ID] = result
	}
	if got := Classify(byID["poisoned"].Err); got != ErrorPoisoned {
		t.Fatalf("poisoned error class = %s, want %s: %v", got, ErrorPoisoned, byID["poisoned"].Err)
	}
	if result := byID["healthy"]; result.Err != nil || result.Job.Status != protocol.StatusSucceeded || result.Job.WorkerID != "worker-1" {
		t.Fatalf("healthy result = %#v", result)
	}
}

func TestTypedSwarmListResultsSurfacesMalformedService(t *testing.T) {
	service := managedService("malformed-id", "malformed", time.Now())
	delete(service.Spec.Labels, "autback.timeout_seconds")
	api := &fakeSwarmEngine{
		services: []swarm.Service{{ID: service.ID, Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: service.Spec.Name}}}},
		inspect:  map[string]swarm.Service{service.ID: service},
	}
	results, err := newRuntimeClient(api, nil).ListResults(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || Classify(results[0].Err) != ErrorPoisoned || !strings.Contains(results[0].Err.Error(), "timeout") {
		t.Fatalf("results = %#v", results)
	}
}

func TestRuntimeTaskStatusClassifiesEverySwarmState(t *testing.T) {
	tests := []struct {
		state    swarm.TaskState
		exitCode int
		want     protocol.Status
	}{
		{state: swarm.TaskStateNew, want: protocol.StatusQueued},
		{state: swarm.TaskStateAllocated, want: protocol.StatusQueued},
		{state: swarm.TaskStatePending, want: protocol.StatusQueued},
		{state: swarm.TaskStateAssigned, want: protocol.StatusQueued},
		{state: swarm.TaskStateAccepted, want: protocol.StatusQueued},
		{state: swarm.TaskStatePreparing, want: protocol.StatusQueued},
		{state: swarm.TaskStateReady, want: protocol.StatusQueued},
		{state: swarm.TaskStateStarting, want: protocol.StatusQueued},
		{state: swarm.TaskStateRunning, want: protocol.StatusRunning},
		{state: swarm.TaskStateComplete, want: protocol.StatusSucceeded},
		{state: swarm.TaskStateComplete, exitCode: 2, want: protocol.StatusFailed},
		{state: swarm.TaskStateShutdown, want: protocol.StatusLost},
		{state: swarm.TaskStateFailed, exitCode: 1, want: protocol.StatusFailed},
		{state: swarm.TaskStateFailed, exitCode: 124, want: protocol.StatusTimedOut},
		{state: swarm.TaskStateRejected, exitCode: 1, want: protocol.StatusFailed},
		{state: swarm.TaskStateRemove, want: protocol.StatusLost},
		{state: swarm.TaskStateOrphaned, want: protocol.StatusLost},
		{state: swarm.TaskState("future-terminal-state"), want: protocol.StatusLost},
	}
	for _, test := range tests {
		t.Run(string(test.state)+fmt.Sprintf("-%d", test.exitCode), func(t *testing.T) {
			if got := runtimeTaskStatus(test.state, test.exitCode, false); got != test.want {
				t.Fatalf("runtimeTaskStatus(%q, %d, false) = %q, want %q", test.state, test.exitCode, got, test.want)
			}
		})
	}
	for _, state := range []swarm.TaskState{swarm.TaskStateNew, swarm.TaskStateRunning, swarm.TaskStateComplete, swarm.TaskStateShutdown, swarm.TaskState("future")} {
		if got := runtimeTaskStatus(state, 0, true); got != protocol.StatusCancelled {
			t.Fatalf("runtimeTaskStatus(%q, 0, true) = %q, want cancelled", state, got)
		}
	}
}

func TestTypedSwarmTerminalizesTasksWithoutContainerStatus(t *testing.T) {
	tests := []struct {
		state swarm.TaskState
		want  protocol.Status
	}{
		{state: swarm.TaskStateComplete, want: protocol.StatusFailed},
		{state: swarm.TaskStateFailed, want: protocol.StatusFailed},
		{state: swarm.TaskStateRejected, want: protocol.StatusFailed},
		{state: swarm.TaskStateShutdown, want: protocol.StatusLost},
		{state: swarm.TaskStateRemove, want: protocol.StatusLost},
		{state: swarm.TaskStateOrphaned, want: protocol.StatusLost},
		{state: swarm.TaskState("future-terminal-state"), want: protocol.StatusLost},
	}
	for _, test := range tests {
		t.Run(string(test.state), func(t *testing.T) {
			createdAt := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
			finishedAt := createdAt.Add(time.Minute)
			service := managedService("service-id", "job-1", createdAt)
			api := &fakeSwarmEngine{tasks: map[string][]swarm.Task{
				service.ID: {{ID: "task-1", Meta: swarm.Meta{CreatedAt: createdAt, UpdatedAt: finishedAt}, Status: swarm.TaskStatus{State: test.state, Timestamp: finishedAt}}},
			}}
			job, err := newRuntimeClient(api, nil).status(context.Background(), service, service.Spec.Name)
			if err != nil {
				t.Fatal(err)
			}
			if job.Status != test.want || job.FinishedAt == nil || job.ExitCode == nil || *job.ExitCode == 0 || job.ErrorMessage == "" {
				t.Fatalf("job = %#v, want terminal %q with failure diagnostics", job, test.want)
			}
		})
	}
}

func TestTypedSwarmCheckRecoversAfterDaemonOutage(t *testing.T) {
	api := &fakeSwarmEngine{apiVersion: client.MaxAPIVersion, infoErr: errdefs.ErrUnavailable}
	runtime := newRuntimeClient(api, nil)
	if err := runtime.Check(context.Background()); Classify(err) != ErrorRetryable {
		t.Fatalf("Check() error = %v, class = %s", err, Classify(err))
	}
	api.infoErr = nil
	api.info = system.Info{Swarm: swarm.Info{LocalNodeState: swarm.LocalNodeStateActive}}
	if err := runtime.Check(context.Background()); err != nil {
		t.Fatalf("Check() after recovery = %v", err)
	}
}

func TestTypedSwarmConvergesAfterRestartDuringConcurrentRefreshAndCancellation(t *testing.T) {
	createdAt := time.Now().UTC()
	service := managedService("service-id", "job-1", createdAt)
	api := &fakeSwarmEngine{
		apiVersion: client.MaxAPIVersion,
		info:       system.Info{Swarm: swarm.Info{LocalNodeState: swarm.LocalNodeStateActive}},
		services:   []swarm.Service{{ID: service.ID, Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: service.Spec.Name}}}},
		inspect:    map[string]swarm.Service{service.ID: service, service.Spec.Name: service},
		tasks:      map[string][]swarm.Task{service.ID: {{Meta: swarm.Meta{CreatedAt: createdAt}, Status: swarm.TaskStatus{State: swarm.TaskStateRunning}}}},
	}
	runtime := newRuntimeClient(api, nil)
	api.setDaemonError(errdefs.ErrUnavailable)
	if err := runtime.Check(context.Background()); Classify(err) != ErrorRetryable {
		t.Fatalf("Check() during outage = %v", err)
	}
	if _, err := runtime.ListResults(context.Background()); Classify(err) != ErrorRetryable {
		t.Fatalf("ListResults() during outage = %v", err)
	}
	api.setDaemonError(nil)

	errorsFound := make(chan error, 24)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(3)
		go func() {
			defer wait.Done()
			errorsFound <- runtime.Check(context.Background())
		}()
		go func() {
			defer wait.Done()
			_, err := runtime.ListResults(context.Background())
			errorsFound <- err
		}()
		go func() {
			defer wait.Done()
			errorsFound <- runtime.Cancel(context.Background(), "job-1")
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTypedSwarmRejectsEngineWithoutReplicatedJobAPI(t *testing.T) {
	api := &fakeSwarmEngine{
		apiVersion: client.MinAPIVersion,
		info:       system.Info{Swarm: swarm.Info{LocalNodeState: swarm.LocalNodeStateActive}},
	}
	err := newRuntimeClient(api, nil).Check(context.Background())
	if Classify(err) != ErrorContract || !strings.Contains(err.Error(), minimumRuntimeAPI) {
		t.Fatalf("Check() error = %v, class = %s", err, Classify(err))
	}
}

func TestTypedSwarmCancelMarksAndStopsJobAtomically(t *testing.T) {
	service := managedService("service-id", "job-1", time.Now())
	service.Version.Index = 7
	api := &fakeSwarmEngine{inspect: map[string]swarm.Service{"job-1": service}}
	if err := newRuntimeClient(api, nil).Cancel(context.Background(), "job-1"); err != nil {
		t.Fatal(err)
	}
	if len(api.updates) != 1 {
		t.Fatalf("updates = %#v", api.updates)
	}
	update := api.updates[0]
	if update.id != "service-id" || update.options.Version.Index != 7 {
		t.Fatalf("update identity = %#v", update)
	}
	if update.options.Spec.Labels["autback.cancelled"] != "true" {
		t.Fatalf("labels = %#v", update.options.Spec.Labels)
	}
	mode := update.options.Spec.Mode.ReplicatedJob
	if mode == nil || mode.TotalCompletions == nil || *mode.TotalCompletions != 0 {
		t.Fatalf("mode = %#v", mode)
	}
}

func TestTypedSwarmCancelRetriesVersionConflict(t *testing.T) {
	service := managedService("service-id", "job-1", time.Now())
	api := &fakeSwarmEngine{
		inspect:      map[string]swarm.Service{"job-1": service},
		updateErrors: []error{errdefs.ErrConflict, nil},
	}
	if err := newRuntimeClient(api, nil).Cancel(context.Background(), "job-1"); err != nil {
		t.Fatal(err)
	}
	if len(api.updates) != 2 {
		t.Fatalf("updates = %d, want 2", len(api.updates))
	}
}

func TestTypedSwarmCreatePreservesJobRuntimeContract(t *testing.T) {
	api := &fakeSwarmEngine{createID: "service-id"}
	spec := swarmscheduler.Spec{
		ID: "job-1", Image: "example/image@sha256:abc", ProjectID: "project-1",
		JobsRoot: "/jobs", CacheRoot: "/cache", CASAddress: "cas:50051", RootDigest: "abc/1",
		Command: []string{"go", "test", "./..."}, Environment: map[string]string{"Z": "last", "A": "first"},
		Timeout: time.Minute, HostUID: "1000", HostGID: "1000",
		Caches: []swarmscheduler.CacheMount{{Name: "gomod", Target: "/go/pkg/mod"}},
	}
	id, err := newRuntimeClient(api, nil).Create(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if id != "service-id" || len(api.creates) != 1 {
		t.Fatalf("id = %q, creates = %#v", id, api.creates)
	}
	created := api.creates[0]
	container := created.Spec.TaskTemplate.ContainerSpec
	if created.Spec.Name != "job-1" || created.Spec.Labels["autback.managed"] != "true" || created.Spec.Mode.ReplicatedJob == nil {
		t.Fatalf("service spec = %#v", created.Spec)
	}
	if container == nil || !reflect.DeepEqual(container.Command, []string{"/usr/local/bin/autback-job-entrypoint"}) || !reflect.DeepEqual(container.Args, spec.Command) {
		t.Fatalf("container spec = %#v", container)
	}
	if !contains(container.Env, "AUTBACK_JOB_ID=job-1") || !contains(container.Env, "A=first") || !contains(container.Env, "Z=last") {
		t.Fatalf("environment = %#v", container.Env)
	}
	if len(container.Mounts) != 4 || container.Mounts[3].Source != "/cache/project-1/gomod" {
		t.Fatalf("mounts = %#v", container.Mounts)
	}
}

func TestTypedSwarmKeepsSecretValuesOutOfServiceSpec(t *testing.T) {
	api := &fakeSwarmEngine{createID: "service-id"}
	spec := swarmscheduler.Spec{
		ID: "job-1", Image: "example/image@sha256:abc", ProjectID: "project-1", JobsRoot: "/jobs",
		Timeout: time.Minute, HasSecrets: true,
		Secrets: []swarmscheduler.SecretMount{{Source: "/jobs/job-1/secrets/001-signing-key", Target: "/run/secrets/signing-key"}},
	}
	if _, err := newRuntimeClient(api, nil).Create(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	container := api.creates[0].Spec.TaskTemplate.ContainerSpec
	if container == nil || len(container.Mounts) != 5 {
		t.Fatalf("container mounts = %#v", container)
	}
	if runtime := container.Mounts[3]; runtime.Source != "/jobs/job-1/secrets" || runtime.Target != "/run/autback/secrets" || !runtime.ReadOnly {
		t.Fatalf("runtime secret mount = %#v", runtime)
	}
	if file := container.Mounts[4]; file.Source != spec.Secrets[0].Source || file.Target != spec.Secrets[0].Target || !file.ReadOnly {
		t.Fatalf("file secret mount = %#v", file)
	}
	encoded := fmt.Sprintf("%#v", api.creates[0].Spec)
	if strings.Contains(encoded, "sentinel-secret-value") {
		t.Fatalf("service spec contains secret value: %s", encoded)
	}
}

func TestTypedSwarmLogsDemultiplexesEngineStream(t *testing.T) {
	var stream bytes.Buffer
	writeLogFrame(&stream, stdcopy.Stdout, "first\n")
	writeLogFrame(&stream, stdcopy.Stderr, "second\n")
	api := &fakeSwarmEngine{logContent: stream.Bytes()}
	var output bytes.Buffer
	if err := newRuntimeClient(api, nil).Logs(context.Background(), "job-1", true, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "first\nsecond\n" || !api.logOptions.Follow || !api.logOptions.ShowStdout || !api.logOptions.ShowStderr {
		t.Fatalf("output = %q, options = %#v", output.String(), api.logOptions)
	}
}

func writeLogFrame(stream *bytes.Buffer, streamType stdcopy.StdType, payload string) {
	header := make([]byte, 8)
	header[0] = byte(streamType)
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	_, _ = stream.Write(header)
	_, _ = stream.WriteString(payload)
}

func TestTypedSwarmRemoveTreatsNotFoundAsSuccess(t *testing.T) {
	api := &fakeSwarmEngine{removeErr: errdefs.ErrNotFound}
	if err := newRuntimeClient(api, nil).Remove(context.Background(), "gone"); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyConflictAsRetryable(t *testing.T) {
	if got := Classify(errdefs.ErrConflict); got != ErrorRetryable {
		t.Fatalf("Classify(conflict) = %s, want %s", got, ErrorRetryable)
	}
}

func managedService(id, name string, createdAt time.Time) swarm.Service {
	one := uint64(1)
	return swarm.Service{
		ID:   id,
		Meta: swarm.Meta{CreatedAt: createdAt, UpdatedAt: createdAt},
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Name: name, Labels: map[string]string{
				"autback.managed": "true", "autback.project": "project-1", "autback.job": name,
				"autback.image": "ZXhhbXBsZS9pbWFnZQ", "autback.timeout_seconds": "60", "autback.root_digest": "abc/1",
			}},
			TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Args: []string{"true"}}},
			Mode:         swarm.ServiceMode{ReplicatedJob: &swarm.ReplicatedJob{MaxConcurrent: &one, TotalCompletions: &one}},
		},
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type serviceUpdate struct {
	id      string
	options client.ServiceUpdateOptions
}

type fakeSwarmEngine struct {
	mu            sync.Mutex
	apiVersion    string
	info          system.Info
	infoErr       error
	listErr       error
	services      []swarm.Service
	inspect       map[string]swarm.Service
	inspectErrors map[string]error
	tasks         map[string][]swarm.Task
	createID      string
	creates       []client.ServiceCreateOptions
	updates       []serviceUpdate
	updateErrors  []error
	removeErr     error
	logContent    []byte
	logOptions    client.ServiceLogsOptions
}

func (f *fakeSwarmEngine) ClientVersion() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.apiVersion == "" {
		return client.MaxAPIVersion
	}
	return f.apiVersion
}

func (f *fakeSwarmEngine) Info(context.Context, client.InfoOptions) (client.SystemInfoResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return client.SystemInfoResult{Info: f.info}, f.infoErr
}

func (f *fakeSwarmEngine) ServiceCreate(_ context.Context, options client.ServiceCreateOptions) (client.ServiceCreateResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates = append(f.creates, options)
	return client.ServiceCreateResult{ID: f.createID}, nil
}

func (f *fakeSwarmEngine) ServiceInspect(_ context.Context, id string, _ client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.inspectErrors[id]; err != nil {
		return client.ServiceInspectResult{}, err
	}
	service, ok := f.inspect[id]
	if !ok {
		return client.ServiceInspectResult{}, errors.New("unexpected service inspect: " + id)
	}
	return client.ServiceInspectResult{Service: cloneService(service)}, nil
}

func (f *fakeSwarmEngine) ServiceList(context.Context, client.ServiceListOptions) (client.ServiceListResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	services := make([]swarm.Service, len(f.services))
	for index, service := range f.services {
		services[index] = cloneService(service)
	}
	return client.ServiceListResult{Items: services}, f.listErr
}

func (f *fakeSwarmEngine) TaskList(_ context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id := range options.Filters["service"] {
		return client.TaskListResult{Items: append([]swarm.Task(nil), f.tasks[id]...)}, nil
	}
	return client.TaskListResult{}, errors.New("service filter is required")
}

func (f *fakeSwarmEngine) ServiceUpdate(_ context.Context, id string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, serviceUpdate{id: id, options: options})
	if len(f.updateErrors) == 0 {
		return client.ServiceUpdateResult{}, nil
	}
	err := f.updateErrors[0]
	f.updateErrors = f.updateErrors[1:]
	return client.ServiceUpdateResult{}, err
}

func (f *fakeSwarmEngine) ServiceRemove(context.Context, string, client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return client.ServiceRemoveResult{}, f.removeErr
}

func (f *fakeSwarmEngine) ServiceLogs(_ context.Context, _ string, options client.ServiceLogsOptions) (client.ServiceLogsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logOptions = options
	return io.NopCloser(bytes.NewReader(f.logContent)), nil
}

func (f *fakeSwarmEngine) setDaemonError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.infoErr = err
	f.listErr = err
}

func cloneService(service swarm.Service) swarm.Service {
	service.Spec.Labels = cloneMap(service.Spec.Labels)
	if container := service.Spec.TaskTemplate.ContainerSpec; container != nil {
		containerCopy := *container
		containerCopy.Command = append([]string(nil), container.Command...)
		containerCopy.Args = append([]string(nil), container.Args...)
		containerCopy.Env = append([]string(nil), container.Env...)
		containerCopy.Mounts = append([]mount.Mount(nil), container.Mounts...)
		service.Spec.TaskTemplate.ContainerSpec = &containerCopy
	}
	if mode := service.Spec.Mode.ReplicatedJob; mode != nil {
		modeCopy := *mode
		if mode.MaxConcurrent != nil {
			value := *mode.MaxConcurrent
			modeCopy.MaxConcurrent = &value
		}
		if mode.TotalCompletions != nil {
			value := *mode.TotalCompletions
			modeCopy.TotalCompletions = &value
		}
		service.Spec.Mode.ReplicatedJob = &modeCopy
	}
	return service
}
